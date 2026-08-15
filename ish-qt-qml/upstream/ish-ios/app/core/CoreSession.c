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

#include "kernel/task.h"
#include "kernel/calls.h"
#include "kernel/errno.h"
#include "xX_main_Xx.h"

struct IshCoreSession {
    char *root_path;
    char **boot_argv;
    size_t boot_argc;
    char **launch_argv;
    size_t launch_argc;

    int input_pipe[2];
    int output_pipe[2];
    pthread_t worker;
    pthread_t reader;
    bool worker_started;
    bool reader_started;
    bool stopping;

    pthread_mutex_t lock;
    bool state_sent;
    int exit_code;
    int saved_stdio[3];
    ish_core_output_callback output;
    ish_core_state_callback state;
    void *cookie;
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
    char buffer[8192];
    for (;;) {
        ssize_t count = read(session->output_pipe[0], buffer, sizeof(buffer));
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
        if (count == 0 || (count < 0 && errno != EINTR))
            break;
    }
    return NULL;
}

static void core_thread_cleanup(void *opaque) {
    struct IshCoreSession *session = opaque;
    active_session = NULL;
    for (int i = 0; i < 3; ++i) {
        if (session->saved_stdio[i] >= 0) {
            (void)dup2(session->saved_stdio[i], i);
            close(session->saved_stdio[i]);
            session->saved_stdio[i] = -1;
        }
    }
    if (session->output_pipe[1] >= 0) {
        close(session->output_pipe[1]);
        session->output_pipe[1] = -1;
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

static void *core_worker(void *opaque) {
    struct IshCoreSession *session = opaque;
    int saved[3] = {-1, -1, -1};
    char **argv = NULL;
    size_t argc = 0;
    int status = 1;

    pthread_cleanup_push(core_thread_cleanup, session);
    active_session = session;
    pthread_setcancelstate(PTHREAD_CANCEL_ENABLE, NULL);
    pthread_setcanceltype(PTHREAD_CANCEL_DEFERRED, NULL);

    for (int i = 0; i < 3; ++i) {
        saved[i] = dup(i);
        session->saved_stdio[i] = saved[i];
    }
    if (saved[0] < 0 || saved[1] < 0 || saved[2] < 0)
        goto finish;

    (void)dup2(session->input_pipe[0], STDIN_FILENO);
    (void)dup2(session->output_pipe[1], STDOUT_FILENO);
    (void)dup2(session->output_pipe[1], STDERR_FILENO);
    close(session->input_pipe[0]);
    session->input_pipe[0] = -1;
    close(session->output_pipe[1]);
    session->output_pipe[1] = -1;
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

    /* xX_main_Xx installs the iSH process-exit handler; replace only the hook. */
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
    session->input_pipe[0] = session->input_pipe[1] = -1;
    session->output_pipe[0] = session->output_pipe[1] = -1;
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
    if (pipe(session->input_pipe) < 0 || pipe(session->output_pipe) < 0)
        return false;
    if (pthread_create(&session->reader, NULL, core_reader, session) != 0)
        return false;
    session->reader_started = true;
    if (pthread_create(&session->worker, NULL, core_worker, session) != 0) {
        close(session->input_pipe[0]);
        close(session->input_pipe[1]);
        close(session->output_pipe[0]);
        close(session->output_pipe[1]);
        session->input_pipe[0] = session->input_pipe[1] = -1;
        session->output_pipe[0] = session->output_pipe[1] = -1;
        pthread_join(session->reader, NULL);
        session->reader_started = false;
        return false;
    }
    session->worker_started = true;
    return true;
}

size_t ish_core_session_write(IshCoreSession *session, const char *bytes, size_t length) {
    if (session == NULL || bytes == NULL || length == 0 || session->input_pipe[1] < 0)
        return 0;
    size_t sent = 0;
    while (sent < length) {
        ssize_t result = write(session->input_pipe[1], bytes + sent, length - sent);
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

void ish_core_session_resize(IshCoreSession *session, int columns, int rows) {
    (void)session;
    (void)columns;
    (void)rows;
    /* create_piped_stdio intentionally has no host PTY; term.js performs rendering. */
}

void ish_core_session_stop(IshCoreSession *session) {
    if (session == NULL)
        return;
    if (session->input_pipe[1] >= 0) {
        close(session->input_pipe[1]);
        session->input_pipe[1] = -1;
    }
    if (session->worker_started) {
        pthread_mutex_lock(&session->lock);
        session->stopping = true;
        pthread_mutex_unlock(&session->lock);
        pthread_cancel(session->worker);
        pthread_join(session->worker, NULL);
        session->worker_started = false;
        if (!session->state_sent)
            emit_state(session, 143);
    }
    if (session->reader_started) {
        pthread_join(session->reader, NULL);
        session->reader_started = false;
    }
    if (session->input_pipe[0] >= 0) {
        close(session->input_pipe[0]);
        session->input_pipe[0] = -1;
    }
    if (session->output_pipe[0] >= 0) {
        close(session->output_pipe[0]);
        session->output_pipe[0] = -1;
    }
    if (session->output_pipe[1] >= 0) {
        close(session->output_pipe[1]);
        session->output_pipe[1] = -1;
    }
}

void ish_core_session_destroy(IshCoreSession *session) {
    if (session == NULL)
        return;
    ish_core_session_stop(session);
    free(session->root_path);
    free_argv(session->boot_argv, session->boot_argc);
    free_argv(session->launch_argv, session->launch_argc);
    pthread_mutex_destroy(&session->lock);
    free(session);
}
