#ifndef ISH_CORE_SESSION_H
#define ISH_CORE_SESSION_H

#include <stdbool.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct IshCoreSession IshCoreSession;

typedef void (*ish_core_output_callback)(void *cookie, const char *bytes, size_t length);
typedef void (*ish_core_state_callback)(void *cookie, int exit_code);

IshCoreSession *ish_core_session_create(const char *root_path,
                                        const char *const *boot_argv,
                                        size_t boot_argc,
                                        const char *const *launch_argv,
                                        size_t launch_argc,
                                        ish_core_output_callback output,
                                        ish_core_state_callback state,
                                        void *cookie);

bool ish_core_session_start(IshCoreSession *session);

/* Input is an arbitrary UTF-8 byte sequence. It is not interpreted or re-encoded. */
size_t ish_core_session_write(IshCoreSession *session, const char *bytes, size_t length);

/* The iSH emulated tty currently reports its size through the host tty path. */
void ish_core_session_resize(IshCoreSession *session, int columns, int rows);

void ish_core_session_stop(IshCoreSession *session);
void ish_core_session_destroy(IshCoreSession *session);

#ifdef __cplusplus
}
#endif

#endif
