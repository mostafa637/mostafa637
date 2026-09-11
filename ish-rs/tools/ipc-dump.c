// Reference-output generator for the ipc differential test.
//
// This compiles the unmodified kernel/ipc.c. The implementation has no host or
// guest-memory dependencies: it only traces its inputs (when tracing is on)
// and returns _ENOSYS. The varied raw i386 argument patterns below make that
// narrow contract explicit in the checked fixture.

#include <stdint.h>
#include <stdio.h>

#include "kernel/calls.h"

static void call_ipc(uint_t call, int_t first, int_t second, int_t third,
                     addr_t ptr, int_t fifth) {
    printf("O IPC %08x %08x %08x %08x %08x %08x\n", call, (dword_t) first,
           (dword_t) second, (dword_t) third, ptr, (dword_t) fifth);
    printf("R %08x\n", (dword_t) sys_ipc(call, first, second, third, ptr, fifth));
}

int main(void) {
    puts("# ipc reference output, generated from unmodified iSH kernel/ipc.c");
    puts("# every record is raw i386 call/argument bits followed by its return");

    call_ipc(0, 0, 0, 0, 0, 0);
    call_ipc(1, 1, 2, 3, 4, 5);
    call_ipc(0xffffffffu, (int_t) 0x80000000u, 0x7fffffff, -1, 0xffffffffu,
             (int_t) 0x87654321u);
    call_ipc(0x0001000du, -100, 0x10203040, (int_t) 0xfedcba98u, 0x00123000u,
             99);
    call_ipc(0x12, -42, -43, -44, 0, -45);
    return 0;
}
