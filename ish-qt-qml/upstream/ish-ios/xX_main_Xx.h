#include <string.h>
#include <unistd.h>
#include <signal.h>
#include <errno.h>
#include <syslog.h>
#include "kernel/init.h"
#include "kernel/fs.h"
#include "fs/devices.h"
#include "fs/real.h"
#ifdef __APPLE__
#include <sys/resource.h>
#define IOPOL_TYPE_VFS_HFS_CASE_SENSITIVITY 1
#define IOPOL_VFS_HFS_CASE_SENSITIVITY_FORCE_CASE_SENSITIVE 1
#endif

void real_tty_reset_term(void);

static void exit_handler(struct task *task, int code) {
    if (task->parent != NULL)
        return;
    real_tty_reset_term();
    if (code & 0xff)
        raise(code & 0xff);
    exit(code >> 8);
}

// this function parses command line arguments and initializes global
// data structures. thanks programming discussions discord server for the name.
// https://discord.gg/9zT7NHP
static inline int xX_main_Xx(int argc, char *const argv[], const char *envp) {
#ifdef __APPLE__
    // Enable case-sensitive filesystem mode on macOS, if possible.
    // In order for this to succeed, either we need to be running as root, or
    // be given the com.apple.private.iopol.case_sensitivity entitlement. The
    // second option isn't possible so you'll need to give iSH the setuid root
    // bit. In that case it's important to drop root permissions ASAP.
    // https://worthdoingbadly.com/casesensitive-iossim/
    int iopol_err = setiopolicy_np(IOPOL_TYPE_VFS_HFS_CASE_SENSITIVITY,
            IOPOL_SCOPE_PROCESS,
            IOPOL_VFS_HFS_CASE_SENSITIVITY_FORCE_CASE_SENSITIVE);
    if (iopol_err != 0 && errno != EPERM)
        perror("could not enable case sensitivity");
    setgid(getgid());
    setuid(getuid());
#endif

    // parse cli options
    int opt;
    const char *root = NULL;
    const char *workdir = NULL;
    const struct fs_ops *fs = &realfs;
    const char *console = "/dev/tty1";
    while ((opt = getopt(argc, argv, "+r:f:d:c:")) != -1) {
        switch (opt) {
            case 'r':
            case 'f':
                root = optarg;
                if (opt == 'f')
                    fs = &fakefs;
                break;
            case 'd':
                workdir = optarg;
                break;
            case 'c':
                console = optarg;
                break;

        }
    }

    openlog(argv[0], 0, LOG_USER);

    char root_realpath[MAX_PATH + 1] = "/";
    if (root != NULL) {
#if defined(__ANDROID__) && !defined(ISH_CORE_HOST)
        /*
         * Android's x86_64 seccomp policy rejects the legacy lstat syscall
         * used by bionic realpath().  The Go layer supplies an absolute,
         * app-private path, so canonicalisation is unnecessary here.
         */
        size_t root_len = strnlen(root, MAX_PATH);
        if (root_len == 0 || root_len >= sizeof(root_realpath)) {
            errno = ENAMETOOLONG;
            perror(root);
            exit(1);
        }
        memcpy(root_realpath, root, root_len + 1);
#else
        if (realpath(root, root_realpath) == NULL) {
            perror(root);
            exit(1);
        }
#endif
    }
    if (fs == &fakefs) {
        size_t root_len = strlen(root_realpath);
        if (root_len + 5 >= sizeof(root_realpath)) {
            errno = ENAMETOOLONG;
            perror(root_realpath);
            exit(1);
        }
        strcat(root_realpath, "/data");
    }
    int err = mount_root(fs, root_realpath);
    if (err < 0)
        return err;

    become_first_process();
    current->thread = pthread_self();
    char cwd[MAX_PATH + 1];
    if (root == NULL && workdir == NULL) {
        getcwd(cwd, sizeof(cwd));
        workdir = cwd;
    }
    if (workdir != NULL) {
        struct fd *pwd = generic_open(workdir, O_RDONLY_, 0);
        if (IS_ERR(pwd)) {
            fprintf(stderr, "error opening working dir: %ld\n", PTR_ERR(pwd));
            return 1;
        }
        fs_chdir(current->fs, pwd);
    }

#if defined(ISH_CORE_SESSION)
    /*
     * CoreSession follows the native iSH application bootstrap.  Leave the
     * current task and root filesystem ready for the caller to install the
     * console/devices and execute the guest only after stdio is connected.
     */
    return 0;
#endif

    char argv_copy[4096];
    int i = optind;
    size_t p = 0;
    while (i < argc) {
        strcpy(&argv_copy[p], argv[i]);
        p += strlen(argv[i]) + 1;
        i++;
    }
    argv_copy[p] = '\0';
    if (argv[optind] == NULL)
	    return _ENOENT;
    err = do_execve(argv[optind], argc - optind, argv_copy, envp == NULL ? "\0" : envp);
    if (err < 0)
        return err;
    tty_drivers[TTY_CONSOLE_MAJOR] = &real_tty_driver;
    if (isatty(STDIN_FILENO) && isatty(STDOUT_FILENO)) {
        err = create_stdio(console, TTY_CONSOLE_MAJOR, 1);
        if (err < 0)
            return err;
    } else {
        err = create_piped_stdio();
        if (err < 0)
            return err;
    }
    exit_hook = exit_handler;
    return 0;
}
