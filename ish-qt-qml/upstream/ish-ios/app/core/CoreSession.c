#define _GNU_SOURCE
#include "CoreSession.h"

#include <errno.h>
#include <fcntl.h>
#include <pthread.h>
#include <signal.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/wait.h>
#include <termios.h>
#include <unistd.h>

#include "fs/fd.h"
#include "kernel/task.h"
#include "kernel/calls.h"
#include "kernel/errno.h"
#include "kernel/init.h"
#include "xX_main_Xx.h"

struct IshCoreSession {
    char *root_path;
    char **boot_argv;
    size_t boot_argc;
    char **launch_argv;
    size_t launch_argc;

    int pty_master;
    int pty_slave;
    pthread_t worker;
    pthread_t reader;
    bool worker_started;
    bool reader_started;
    bool stopping;
    struct sigaction saved_sighup;
    bool sighup_saved;

    pthread_mutex_t lock;
    bool state_sent;
    int exit_code;
    int saved_stdio[3];
    ish_core_output_callback output;
    ish_core_state_callback state;
    void *cookie;
    char *resolver_config;
    size_t resolver_config_length;
};

static _Thread_local struct IshCoreSession *active_session;

static char *copy_string(const char *value) {
    if (value == NULL)
        value = "";
    return strdup(value);
}

static char **copy_argv(const char *const *argv, size_t argc) {
    char **copy = calloc(argc + 1, sizeof(*copy));
    if (copy == NULL)
        return NULL;
    for (size_t i = 0; i < argc; ++i) {
        copy[i] = copy_string(argv[i]);
        if (copy[i] == NULL) {
            for (size_t j = 0; j < i; ++j)
                free(copy[j]);
            free(copy);
            return NULL;
        }
    }
    copy[argc] = NULL;
    return copy;
}

static void free_argv(char **argv, size_t argc) {
    if (argv == NULL)
        return;
    for (size_t i = 0; i < argc; ++i)
        free(argv[i]);
    free(argv);
}

static void emit_state(struct IshCoreSession *session, int exit_code) {
    ish_core_state_callback callback = NULL;
    void *cookie = NULL;
    pthread_mutex_lock(&session->lock);
    if (!session->state_sent) {
        session->state_sent = true;
        session->exit_code = exit_code;
        callback = session->state;
        cookie = session->cookie;
    }
    pthread_mutex_unlock(&session->lock);
    if (callback != NULL)
        callback(cookie, exit_code);
}

static void core_exit_hook(struct task *task, int code) {
    struct IshCoreSession *session = active_session;
    /* Child tasks are allowed to exit without ending the terminal session. */
    if (task != NULL && task->parent != NULL)
        return;
    if (session != NULL)
        emit_state(session, (code & 0xff) ? 128 + (code & 0xff) : (code >> 8));
}

static void restore_stdio(int saved[3]) {
    for (int i = 0; i < 3; ++i) {
        if (saved[i] >= 0) {
            (void)dup2(saved[i], i);
            close(saved[i]);
        }
    }
}

static void *core_reader(void *opaque) {
    struct IshCoreSession *session = opaque;
    const int master = session->pty_master;
    char buffer[8192];
    for (;;) {
        ssize_t count = read(master, buffer, sizeof(buffer));
        if (count > 0) {
            ish_core_output_callback callback = NULL;
            void *cookie = NULL;
            pthread_mutex_lock(&session->lock);
            callback = session->output;
            cookie = session->cookie;
            pthread_mutex_unlock(&session->lock);
            if (callback != NULL)
                callback(cookie, buffer, (size_t)count);
            continue;
        }
        if (count == 0)
            break;
        if (count < 0 && (errno == EINTR))
            continue;
        if (count < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
            pthread_mutex_lock(&session->lock);
            const bool stopping = session->stopping;
            pthread_mutex_unlock(&session->lock);
            if (stopping)
                break;
            struct timespec pause = {.tv_sec = 0, .tv_nsec = 5000000L};
            nanosleep(&pause, NULL);
            continue;
        }
        break;
    }
    return NULL;
}

static void core_thread_cleanup(void *opaque) {
    struct IshCoreSession *session = opaque;
    active_session = NULL;
#if defined(__ANDROID__) && !defined(ISH_CORE_HOST)
    if (session->sighup_saved) {
        (void)sigaction(SIGHUP, &session->saved_sighup, NULL);
        session->sighup_saved = false;
    }
#endif
    for (int i = 0; i < 3; ++i) {
        if (session->saved_stdio[i] >= 0) {
            (void)dup2(session->saved_stdio[i], i);
            close(session->saved_stdio[i]);
            session->saved_stdio[i] = -1;
        }
    }
    if (session->pty_slave >= 0) {
        close(session->pty_slave);
        session->pty_slave = -1;
    }
}

static int make_core_argv(struct IshCoreSession *session, char ***argv_out, size_t *argc_out) {
    const char *const *selected = (const char *const *)session->launch_argv;
    size_t selected_count = session->launch_argc;
    if (selected_count == 0) {
        selected = (const char *const *)session->boot_argv;
        selected_count = session->boot_argc;
    }
    if (selected_count == 0) {
        static const char *const fallback[] = {"/bin/sh", NULL};
        selected = fallback;
        selected_count = 1;
    }

    size_t argc = selected_count + 8;
    char **argv = calloc(argc + 1, sizeof(*argv));
    if (argv == NULL)
        return ENOMEM;
    size_t n = 0;
    argv[n++] = copy_string("ish");
    argv[n++] = copy_string("-f");
    argv[n++] = copy_string(session->root_path);
    argv[n++] = copy_string("-d");
    argv[n++] = copy_string("/");
    argv[n++] = copy_string("-c");
    argv[n++] = copy_string("/dev/tty1");
    for (size_t i = 0; i < selected_count; ++i)
        argv[n++] = copy_string(selected[i]);
    argv[n] = NULL;
    for (size_t i = 0; i < n; ++i) {
        if (argv[i] == NULL) {
            free_argv(argv, n);
            return ENOMEM;
        }
    }
    *argv_out = argv;
    *argc_out = n;
    return 0;
}

static int create_host_pty(struct IshCoreSession *session) {
    int master = open("/dev/ptmx", O_RDWR | O_NOCTTY | O_CLOEXEC);
    if (master < 0)
        return -1;

    if (grantpt(master) < 0 || unlockpt(master) < 0) {
        close(master);
        return -1;
    }

    char slave_name[64];
    if (ptsname_r(master, slave_name, sizeof(slave_name)) != 0) {
        close(master);
        return -1;
    }

    int slave = open(slave_name, O_RDWR | O_NOCTTY | O_CLOEXEC);
    if (slave < 0) {
        close(master);
        return -1;
    }

    // Match Termux: enable UTF-8 input and disable software flow control.
    struct termios tios;
    if (tcgetattr(master, &tios) == 0) {
#ifdef IUTF8
        tios.c_iflag |= IUTF8;
#endif
        tios.c_iflag &= ~(IXON | IXOFF);
        (void)tcsetattr(master, TCSANOW, &tios);
    }

    struct winsize size = {
        .ws_row = 24,
        .ws_col = 80,
        .ws_xpixel = 0,
        .ws_ypixel = 0
    };
    (void)ioctl(master, TIOCSWINSZ, &size);

    // Keep both PTY descriptors outside the low-numbered range used by Qt's
    // event loop, sockets, pipes, and eventfds.  The iSH core shares the
    // process with Qt, so another subsystem may legitimately close/reuse a
    // low descriptor during startup or shutdown.  Moving the descriptors
    // before publishing them prevents the PTY master from being invalidated
    // behind the reader thread's back.
#ifdef F_DUPFD_CLOEXEC
    int high_master = fcntl(master, F_DUPFD_CLOEXEC, 100);
    int high_slave = fcntl(slave, F_DUPFD_CLOEXEC, 101);
#else
    int high_master = fcntl(master, F_DUPFD, 100);
    int high_slave = fcntl(slave, F_DUPFD, 101);
    if (high_master >= 0)
        (void)fcntl(high_master, F_SETFD, FD_CLOEXEC);
    if (high_slave >= 0)
        (void)fcntl(high_slave, F_SETFD, FD_CLOEXEC);
#endif
    if (high_master < 0 || high_slave < 0) {
        if (high_master >= 0)
            close(high_master);
        if (high_slave >= 0)
            close(high_slave);
        close(slave);
        close(master);
        return -1;
    }
    close(master);
    close(slave);
    int flags = fcntl(high_master, F_GETFL, 0);
    if (flags >= 0)
        (void)fcntl(high_master, F_SETFL, flags | O_NONBLOCK);
    session->pty_master = high_master;
    session->pty_slave = high_slave;
    return 0;
}

static void close_host_pty(struct IshCoreSession *session) {
    if (session->pty_master >= 0) {
        close(session->pty_master);
        session->pty_master = -1;
    }
    if (session->pty_slave >= 0) {
        close(session->pty_slave);
        session->pty_slave = -1;
    }
}

static bool configure_dns(const struct IshCoreSession *session) {
    if (session == NULL || session->resolver_config == NULL ||
        session->resolver_config_length == 0)
        return false;

    struct fd *file = generic_open("/etc/resolv.conf",
                                   O_WRONLY_ | O_CREAT_ | O_TRUNC_, 0666);
    if (IS_ERR(file) || file->ops == NULL || file->ops->write == NULL) {
        if (!IS_ERR(file) && file != NULL)
            fd_close(file);
        return false;
    }
    ssize_t written = file->ops->write(file, session->resolver_config,
                                       session->resolver_config_length);
    fd_close(file);
    return written == (ssize_t)session->resolver_config_length;
}

static void *core_worker(void *opaque) {
    struct IshCoreSession *session = opaque;
    int saved[3] = {-1, -1, -1};
    char **argv = NULL;
    size_t argc = 0;
    int status = 1;

    pthread_cleanup_push(core_thread_cleanup, session);
    active_session = session;
#if !defined(__ANDROID__) || defined(ISH_CORE_HOST)
    pthread_setcancelstate(PTHREAD_CANCEL_ENABLE, NULL);
    pthread_setcanceltype(PTHREAD_CANCEL_DEFERRED, NULL);
#endif

    for (int i = 0; i < 3; ++i) {
        saved[i] = dup(i);
        session->saved_stdio[i] = saved[i];
    }
    if (saved[0] < 0 || saved[1] < 0 || saved[2] < 0)
        goto finish;

    // On Android the app has no terminal of its own, so establish the PTY
    // slave as the controlling terminal as Termux does. The Linux host
    // smoke process must keep its Qt/launcher session unchanged; the PTY
    // still provides tty line discipline there, but setsid is omitted.
#if defined(__ANDROID__) && !defined(ISH_CORE_HOST)
    struct sigaction ignore_sighup;
    memset(&ignore_sighup, 0, sizeof(ignore_sighup));
    sigemptyset(&ignore_sighup.sa_mask);
    ignore_sighup.sa_handler = SIG_IGN;
    if (sigaction(SIGHUP, &ignore_sighup, &session->saved_sighup) == 0)
        session->sighup_saved = true;
    (void)setsid();
    (void)ioctl(session->pty_slave, TIOCSCTTY, 0);
#endif
    (void)dup2(session->pty_slave, STDIN_FILENO);
    (void)dup2(session->pty_slave, STDOUT_FILENO);
    (void)dup2(session->pty_slave, STDERR_FILENO);
    close(session->pty_slave);
    session->pty_slave = -1;
    signal(SIGPIPE, SIG_IGN);

    if (make_core_argv(session, &argv, &argc) != 0)
        goto finish;

    const char envp[] = "TERM=xterm-256color\0LANG=C.UTF-8\0LC_ALL=C.UTF-8\0\0";
    optind = 1;
    opterr = 0;
    int result = xX_main_Xx((int)argc, argv, envp);
    free_argv(argv, argc);
    argv = NULL;
    if (result < 0)
        goto finish;

    /*
     * Keep the same bootstrap sequence as upstream iSH's main.c.  xX_main_Xx
     * creates and mounts the real rootfs, but Alpine still needs the guest
     * device nodes plus procfs/devpts before /bin/ash can start correctly.
     */
    create_some_device_nodes();
    (void)do_mount(&procfs, "proc", "/proc", "", 0);
    (void)do_mount(&devptsfs, "devpts", "/dev/pts", "", 0);

    /* xX_main_Xx installs the iSH process-exit handler; replace only the hook. */
    /* Keep the resolver configuration inside fakefs, just like iSH iOS. */
    (void)configure_dns(session);
    exit_hook = core_exit_hook;
    status = 0;
    task_run_current();

finish:
    free_argv(argv, argc);
    restore_stdio(saved);
    for (int i = 0; i < 3; ++i)
        session->saved_stdio[i] = -1;
    if (!session->state_sent)
        emit_state(session, status);
    pthread_cleanup_pop(1);
    return NULL;
}

IshCoreSession *ish_core_session_create(const char *root_path,
                                        const char *const *boot_argv,
                                        size_t boot_argc,
                                        const char *const *launch_argv,
                                        size_t launch_argc,
                                        ish_core_output_callback output,
                                        ish_core_state_callback state,
                                        void *cookie) {
    struct IshCoreSession *session = calloc(1, sizeof(*session));
    if (session == NULL)
        return NULL;
    session->pty_master = -1;
    session->pty_slave = -1;
    session->root_path = copy_string(root_path);
    session->boot_argv = copy_argv(boot_argv, boot_argc);
    session->launch_argv = copy_argv(launch_argv, launch_argc);
    session->boot_argc = boot_argc;
    session->launch_argc = launch_argc;
    session->output = output;
    session->state = state;
    session->cookie = cookie;
    session->saved_stdio[0] = session->saved_stdio[1] = session->saved_stdio[2] = -1;
    pthread_mutex_init(&session->lock, NULL);
    if (session->root_path == NULL || (boot_argc && session->boot_argv == NULL) ||
        (launch_argc && session->launch_argv == NULL)) {
        ish_core_session_destroy(session);
        return NULL;
    }
    return session;
}

bool ish_core_session_start(IshCoreSession *session) {
    if (session == NULL || session->worker_started)
        return false;
    if (create_host_pty(session) != 0)
        return false;
    if (pthread_create(&session->reader, NULL, core_reader, session) != 0) {
        close_host_pty(session);
        return false;
    }
    session->reader_started = true;
    if (pthread_create(&session->worker, NULL, core_worker, session) != 0) {
        close_host_pty(session);
        pthread_join(session->reader, NULL);
        session->reader_started = false;
        return false;
    }
    session->worker_started = true;
    return true;
}

size_t ish_core_session_write(IshCoreSession *session, const char *bytes, size_t length) {
    if (session == NULL || bytes == NULL || length == 0 || session->pty_master < 0)
        return 0;
    size_t sent = 0;
    while (sent < length) {
        ssize_t result = write(session->pty_master, bytes + sent, length - sent);
        if (result > 0) {
            sent += (size_t)result;
            continue;
        }
        if (result < 0 && errno == EINTR)
            continue;
        break;
    }
    return sent;
}

size_t ish_core_session_set_resolver_config(IshCoreSession *session,
                                            const char *config,
                                            size_t length) {
    if (session == NULL || session->worker_started || config == NULL || length == 0)
        return 0;
    char *copy = malloc(length);
    if (copy == NULL)
        return 0;
    memcpy(copy, config, length);
    free(session->resolver_config);
    session->resolver_config = copy;
    session->resolver_config_length = length;
    return length;
}

void ish_core_session_resize(IshCoreSession *session, int columns, int rows) {
    if (session == NULL || session->pty_master < 0 || columns <= 0 || rows <= 0)
        return;
    struct winsize size = {
        .ws_row = (unsigned short)rows,
        .ws_col = (unsigned short)columns,
        .ws_xpixel = 0,
        .ws_ypixel = 0
    };
    (void)ioctl(session->pty_master, TIOCSWINSZ, &size);
}

void ish_core_session_stop(IshCoreSession *session) {
    if (session == NULL)
        return;
    // Ask an interactive shell to exit while the PTY is still connected.
    // Closing the master first can leave iSH's terminal wait loop alive, so
    // give the shell a normal command path before using EOT/hangup as the
    // fallback wake-up.
    pthread_mutex_lock(&session->lock);
    session->stopping = true;
    pthread_mutex_unlock(&session->lock);
    if (session->pty_master >= 0) {
        static const char exit_command[] = "exit\n";
        (void)write(session->pty_master, exit_command, sizeof(exit_command) - 1);
    }
    if (session->worker_started) {
#if !defined(__ANDROID__) || defined(ISH_CORE_HOST)
        struct timespec grace = {.tv_sec = 0, .tv_nsec = 50000000L};
        nanosleep(&grace, NULL);
        pthread_cancel(session->worker);
        pthread_join(session->worker, NULL);
#else
        // Android bionic does not expose POSIX pthread cancellation. The
        // queued exit command is consumed by the interactive shell before
        // the PTY is closed below.
        pthread_join(session->worker, NULL);
#endif
        session->worker_started = false;
    }
    if (session->pty_master >= 0) {
        const char eot = '\004';
        (void)write(session->pty_master, &eot, 1);
        close(session->pty_master);
        session->pty_master = -1;
        if (!session->state_sent)
            emit_state(session, 143);
    }
    if (session->reader_started) {
        pthread_join(session->reader, NULL);
        session->reader_started = false;
    }
    close_host_pty(session);
}

void ish_core_session_destroy(IshCoreSession *session) {
    if (session == NULL)
        return;
    ish_core_session_stop(session);
    free(session->root_path);
    free_argv(session->boot_argv, session->boot_argc);
    free_argv(session->launch_argv, session->launch_argc);
    free(session->resolver_config);
    pthread_mutex_destroy(&session->lock);
    free(session);
}
