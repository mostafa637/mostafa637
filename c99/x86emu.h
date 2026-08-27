#ifndef X86EMU_H
#define X86EMU_H

#include "x86asm.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef enum x86emu_error {
    X86EMU_OK = 0,
    X86EMU_ERR_BAD_ARGUMENT,
    X86EMU_ERR_MEMORY,
    X86EMU_ERR_ARITHMETIC,
    X86EMU_ERR_PRIVILEGED,
    X86EMU_ERR_DECODE,
    X86EMU_ERR_UNSUPPORTED,
    X86EMU_ERR_BREAKPOINT,
    X86EMU_ERR_INTERRUPT,
    X86EMU_ERR_STEP_LIMIT
} x86emu_error;

enum {
    X86EMU_RAX = 0, X86EMU_RCX, X86EMU_RDX, X86EMU_RBX,
    X86EMU_RSP, X86EMU_RBP, X86EMU_RSI, X86EMU_RDI,
    X86EMU_R8, X86EMU_R9, X86EMU_R10, X86EMU_R11,
    X86EMU_R12, X86EMU_R13, X86EMU_R14, X86EMU_R15
};

enum {
    X86EMU_FLAG_CF = UINT64_C(1) << 0,
    X86EMU_FLAG_PF = UINT64_C(1) << 2,
    X86EMU_FLAG_AF = UINT64_C(1) << 4,
    X86EMU_FLAG_ZF = UINT64_C(1) << 6,
    X86EMU_FLAG_SF = UINT64_C(1) << 7,
    X86EMU_FLAG_TF = UINT64_C(1) << 8,
    X86EMU_FLAG_IF = UINT64_C(1) << 9,
    X86EMU_FLAG_DF = UINT64_C(1) << 10,
    X86EMU_FLAG_OF = UINT64_C(1) << 11
};

typedef struct x86emu_memory {
    uint8_t *data;       /* owned by the caller */
    size_t size;
    uint64_t base_address;
} x86emu_memory;

struct x86emu_cpu;
typedef bool (*x86emu_interrupt_handler)(struct x86emu_cpu *cpu,
                                         uint8_t vector, void *user_data);
typedef bool (*x86emu_system_handler)(struct x86emu_cpu *cpu,
                                      x86asm_opcode opcode, void *user_data);

typedef struct x86emu_cpu {
    uint64_t registers[16];
    uint8_t vector_registers[32][64]; /* XMM/YMM/ZMM state; owned by CPU value storage */
    uint64_t mmx_registers[8];
    uint8_t x87_registers[8][10]; /* 80-bit extended values, stored without host padding */
    uint16_t x87_control;
    uint16_t x87_status;
    uint16_t x87_tag;
    uint32_t mxcsr;
    uint64_t rip;
    uint64_t rflags;
    x86emu_memory memory; /* borrowed; never freed by x86emu */
    x86emu_interrupt_handler interrupt_handler;
    x86emu_system_handler system_handler;
    void *user_data;
    uint64_t steps;
    bool halted;
    bool break_on_int3;
    uint64_t breakpoints[32];
    size_t breakpoint_count;
    x86asm_instruction last_instruction;
    x86emu_error last_error;
} x86emu_cpu;

/* The caller owns memory and must keep it alive while CPU is in use. */
void x86emu_init(x86emu_cpu *cpu, x86emu_memory memory, uint64_t entry);
x86emu_error x86emu_step(x86emu_cpu *cpu);
x86emu_error x86emu_run(x86emu_cpu *cpu, uint64_t max_steps);
const char *x86emu_error_string(x86emu_error error);

bool x86emu_add_breakpoint(x86emu_cpu *cpu, uint64_t address);
bool x86emu_remove_breakpoint(x86emu_cpu *cpu, uint64_t address);
uint64_t x86emu_get_register(const x86emu_cpu *cpu, unsigned index);
void x86emu_set_register(x86emu_cpu *cpu, unsigned index, uint64_t value);
void x86emu_set_interrupt_handler(x86emu_cpu *cpu, x86emu_interrupt_handler handler);
void x86emu_set_system_handler(x86emu_cpu *cpu, x86emu_system_handler handler);

#ifdef __cplusplus
}
#endif

#endif /* X86EMU_H */
