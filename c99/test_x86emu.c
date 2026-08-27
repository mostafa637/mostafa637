#include "x86emu.h"

#include <assert.h>
#include <stdio.h>
#include <string.h>

static void test_add_and_flags(void)
{
    uint8_t code[] = { 0x48, 0xB8, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
                       0x48, 0x83, 0xC0, 0x01,
                       0xCC };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    x86emu_init(&cpu, memory, 0);

    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_MAX);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 0);
    assert((cpu.rflags & X86EMU_FLAG_ZF) != 0);
    assert((cpu.rflags & X86EMU_FLAG_CF) != 0);
}

static void test_call_return(void)
{
    uint8_t memory_bytes[128] = {
        0xE8, 0x02, 0x00, 0x00, 0x00, /* call +2: target is offset 7 */
        0x90,                            /* return address */
        0xCC,                            /* stop after returning */
        0x48, 0xB8, 0x2A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0xC3                             /* ret */
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { memory_bytes, sizeof(memory_bytes), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RSP] = 120;

    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.rip == 7);
    assert(cpu.registers[X86EMU_RSP] == 112);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 42);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.rip == 5);
    assert(cpu.registers[X86EMU_RSP] == 120);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.rip == 6);
}

static void test_vector_execution(void)
{
    uint8_t memory_bytes[128] = {
        0xC5, 0xE4, 0x58, 0xCA, /* vaddps ymm1, ymm3, ymm2 */
        0xC5, 0xFC, 0x10, 0x08  /* vmovups ymm1, [rax] */
    };
    static const uint8_t blend_code[] = { 0xC4, 0xE3, 0x65, 0x0C, 0xCA, 0x01 }; /* vblendps ymm1, ymm3, ymm2, 1 */
    static const uint8_t compare_code[] = { 0xC5, 0xE5, 0x74, 0xCA }; /* vpcmpeqb ymm1, ymm3, ymm2 */
    static const uint8_t compare_word_code[] = { 0xC5, 0xE5, 0x75, 0xCA };
    static const uint8_t compare_dword_code[] = { 0xC5, 0xE5, 0x76, 0xCA };
    static const uint8_t subtract_code[] = { 0xC5, 0xE5, 0xFA, 0xCA }; /* vpsubd ymm1, ymm3, ymm2 */
    static const uint8_t packed_code[] = {
        0xC5, 0xE5, 0xFC, 0xCA, 0xC5, 0xE5, 0xFD, 0xCA,
        0xC5, 0xE5, 0xF8, 0xCA, 0xC5, 0xE5, 0xF9, 0xCA
    };
    float left[8] = { 1.0f, 2.0f, 3.0f, 4.0f, 5.0f, 6.0f, 7.0f, 8.0f };
    float right[8] = { 3.0f, 4.0f, 5.0f, 6.0f, 7.0f, 8.0f, 9.0f, 10.0f };
    float loaded[8] = { 11.0f, 12.0f, 13.0f, 14.0f, 15.0f, 16.0f, 17.0f, 18.0f };
    uint32_t subtract_left[8] = { 100u, 200u, 300u, 400u, 500u, 600u, 700u, 800u };
    uint32_t subtract_right[8] = { 1u, 2u, 3u, 4u, 5u, 6u, 7u, 8u };
    x86emu_cpu cpu;
    x86emu_memory memory = { memory_bytes, sizeof(memory_bytes), 0 };
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[3], left, sizeof(left));
    memcpy(cpu.vector_registers[2], right, sizeof(right));
    memcpy(memory_bytes + 64, loaded, sizeof(loaded));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    float sum[8];
    memcpy(sum, cpu.vector_registers[1], sizeof(sum));
    for (unsigned i = 0; i < 8; ++i) assert(sum[i] == left[i] + right[i]);
    cpu.registers[X86EMU_RAX] = 64;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(sum, cpu.vector_registers[1], sizeof(sum));
    for (unsigned i = 0; i < 8; ++i) assert(sum[i] == loaded[i]);
    memcpy(memory_bytes + 8, blend_code, sizeof(blend_code));
    memcpy(cpu.vector_registers[3], left, sizeof(left));
    memcpy(cpu.vector_registers[2], right, sizeof(right));
    cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(sum, cpu.vector_registers[1], sizeof(sum));
    for (unsigned i = 0; i < 8; ++i) {
        float expected = (i == 0) ? right[i] : left[i];
        assert(sum[i] == expected);
    }
    memcpy(memory_bytes + 14, compare_code, sizeof(compare_code));
    memcpy(cpu.vector_registers[3], left, sizeof(left));
    memcpy(cpu.vector_registers[2], left, sizeof(left));
    cpu.rip = 14;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    for (unsigned i = 0; i < 32; ++i) assert(cpu.vector_registers[1][i] == UINT8_MAX);
    memcpy(memory_bytes + 18, subtract_code, sizeof(subtract_code));
    memcpy(cpu.vector_registers[3], subtract_left, sizeof(subtract_left));
    memcpy(cpu.vector_registers[2], subtract_right, sizeof(subtract_right));
    cpu.rip = 18;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    uint32_t subtract_result[8];
    memcpy(subtract_result, cpu.vector_registers[1], sizeof(subtract_result));
    for (unsigned i = 0; i < 8; ++i) assert(subtract_result[i] == subtract_left[i] - subtract_right[i]);

    uint8_t byte_left[32];
    uint8_t byte_right[32];
    uint8_t byte_result[32];
    uint16_t word_left[16];
    uint16_t word_right[16];
    uint16_t word_result[16];
    for (unsigned i = 0; i < 32; ++i) {
        byte_left[i] = (uint8_t)i;
        byte_right[i] = 2;
    }
    for (unsigned i = 0; i < 16; ++i) {
        word_left[i] = (uint16_t)(100u + i);
        word_right[i] = 3;
    }
    memcpy(memory_bytes + 22, packed_code, sizeof(packed_code));
    memcpy(cpu.vector_registers[3], byte_left, sizeof(byte_left));
    memcpy(cpu.vector_registers[2], byte_right, sizeof(byte_right));
    cpu.rip = 22;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(byte_result, cpu.vector_registers[1], sizeof(byte_result));
    for (unsigned i = 0; i < 32; ++i) assert(byte_result[i] == (uint8_t)(i + 2u));

    memcpy(cpu.vector_registers[3], word_left, sizeof(word_left));
    memcpy(cpu.vector_registers[2], word_right, sizeof(word_right));
    cpu.rip = 26;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(word_result, cpu.vector_registers[1], sizeof(word_result));
    for (unsigned i = 0; i < 16; ++i) assert(word_result[i] == (uint16_t)(word_left[i] + 3u));

    memcpy(cpu.vector_registers[3], byte_left, sizeof(byte_left));
    memcpy(cpu.vector_registers[2], byte_right, sizeof(byte_right));
    cpu.rip = 30;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(byte_result, cpu.vector_registers[1], sizeof(byte_result));
    for (unsigned i = 0; i < 32; ++i) assert(byte_result[i] == (uint8_t)(i - 2u));

    memcpy(cpu.vector_registers[3], word_left, sizeof(word_left));
    memcpy(cpu.vector_registers[2], word_right, sizeof(word_right));
    cpu.rip = 34;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(word_result, cpu.vector_registers[1], sizeof(word_result));
    for (unsigned i = 0; i < 16; ++i) assert(word_result[i] == (uint16_t)(word_left[i] - 3u));

    uint16_t compare_word_left[16];
    uint16_t compare_word_right[16];
    for (unsigned i = 0; i < 16; ++i) {
        compare_word_left[i] = (uint16_t)(i + 1u);
        compare_word_right[i] = compare_word_left[i];
    }
    compare_word_right[5] = 99;
    memcpy(memory_bytes + 38, compare_word_code, sizeof(compare_word_code));
    memcpy(cpu.vector_registers[3], compare_word_left, sizeof(compare_word_left));
    memcpy(cpu.vector_registers[2], compare_word_right, sizeof(compare_word_right));
    cpu.rip = 38;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(compare_word_left, cpu.vector_registers[1], sizeof(compare_word_left));
    for (unsigned i = 0; i < 16; ++i) {
        uint16_t expected = i == 5 ? 0 : UINT16_MAX;
        assert(compare_word_left[i] == expected);
    }

    uint32_t compare_dword_left[8];
    uint32_t compare_dword_right[8];
    for (unsigned i = 0; i < 8; ++i) {
        compare_dword_left[i] = i + 1u;
        compare_dword_right[i] = compare_dword_left[i];
    }
    compare_dword_right[2] = 77;
    memcpy(memory_bytes + 42, compare_dword_code, sizeof(compare_dword_code));
    memcpy(cpu.vector_registers[3], compare_dword_left, sizeof(compare_dword_left));
    memcpy(cpu.vector_registers[2], compare_dword_right, sizeof(compare_dword_right));
    cpu.rip = 42;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(compare_dword_left, cpu.vector_registers[1], sizeof(compare_dword_left));
    for (unsigned i = 0; i < 8; ++i) {
        uint32_t expected = i == 2 ? 0u : UINT32_MAX;
        assert(compare_dword_left[i] == expected);
    }
}

static void test_arithmetic_and_compare(void)
{
    static const uint8_t code[] = {
        0x48, 0xF7, 0xE1, /* mul rcx */
        0x48, 0x31, 0xD2, /* xor edx, edx */
        0x48, 0xF7, 0xF1, /* div rcx */
        0x48, 0xF7, 0xF9, /* idiv rcx */
        0x48, 0x0F, 0xB1, 0xCB /* cmpxchg rbx, rcx */
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { (uint8_t *)code, sizeof(code), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 2;
    cpu.registers[X86EMU_RCX] = 3;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 6);
    assert(cpu.registers[X86EMU_RDX] == 0);

    cpu.registers[X86EMU_RAX] = 10;
    cpu.registers[X86EMU_RDX] = 0;
    cpu.rip = 6;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 3);
    assert(cpu.registers[X86EMU_RDX] == 1);

    cpu.registers[X86EMU_RAX] = UINT64_MAX - 9u;
    cpu.registers[X86EMU_RDX] = UINT64_MAX;
    cpu.rip = 9;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_MAX - 2u);
    assert(cpu.registers[X86EMU_RDX] == UINT64_MAX);

    cpu.registers[X86EMU_RAX] = 5;
    cpu.registers[X86EMU_RBX] = 5;
    cpu.registers[X86EMU_RCX] = 9;
    cpu.rip = 12;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RBX] == 9);
    assert((cpu.rflags & X86EMU_FLAG_ZF) != 0);

    cpu.registers[X86EMU_RAX] = 4;
    cpu.registers[X86EMU_RBX] = 7;
    cpu.rip = 12;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 7);
    assert((cpu.rflags & X86EMU_FLAG_ZF) == 0);
}

static void test_rotates(void)
{
    uint8_t code[16] = {
        0x48, 0xD1, 0xC0,
        0x48, 0xD3, 0xC8,
        0xD0, 0xD0,
        0xD0, 0xD8
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = UINT64_C(0x8000000000000000);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 1 && (cpu.rflags & X86EMU_FLAG_CF) != 0);

    cpu.registers[X86EMU_RAX] = 1;
    cpu.registers[X86EMU_RCX] = 1;
    cpu.rip = 3;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_C(0x8000000000000000) && (cpu.rflags & X86EMU_FLAG_CF) != 0);

    cpu.registers[X86EMU_RAX] = 0xFF;
    cpu.rflags &= ~UINT64_C(1);
    cpu.rip = 6;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert((cpu.registers[X86EMU_RAX] & 0xFFu) == 0xFE && (cpu.rflags & X86EMU_FLAG_CF) != 0);

    cpu.registers[X86EMU_RAX] = 0;
    cpu.rflags |= X86EMU_FLAG_CF;
    cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert((cpu.registers[X86EMU_RAX] & 0xFFu) == 0x80 && (cpu.rflags & X86EMU_FLAG_CF) == 0);
}

static void test_double_shifts(void)
{
    uint8_t code[32] = {
        0x48, 0x0F, 0xA4, 0xC8, 0x04,
        0x48, 0x0F, 0xAC, 0xC8, 0x04,
        0x48, 0x0F, 0xA5, 0xC8,
        0x48, 0x0F, 0xAD, 0xC8
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 0x1000;
    cpu.registers[X86EMU_RCX] = 3;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 0x10000);
    assert((cpu.rflags & X86EMU_FLAG_CF) == 0);

    cpu.registers[X86EMU_RAX] = 0x1000;
    cpu.registers[X86EMU_RCX] = 3;
    cpu.rip = 5;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_C(0x3000000000000100));

    cpu.registers[X86EMU_RAX] = 0x1000;
    cpu.registers[X86EMU_RCX] = 4;
    cpu.rip = 10;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 0x10000);

    cpu.registers[X86EMU_RAX] = UINT64_C(0x8000000000000000);
    cpu.registers[X86EMU_RCX] = 4;
    cpu.rip = 14;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_C(0x4800000000000000));
}

static void test_movd_movq(void)
{
    uint8_t code[32] = {
        0x0F, 0x6E, 0xC0,
        0x0F, 0x7E, 0xC0,
        0x48, 0x0F, 0x6E, 0xC0,
        0x48, 0x0F, 0x7E, 0xC0
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = UINT64_C(0xFFFFFFFFAABBCCDD);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.vector_registers[0][0] == 0xDD && cpu.vector_registers[0][1] == 0xCC &&
           cpu.vector_registers[0][2] == 0xBB && cpu.vector_registers[0][3] == 0xAA);
    for (unsigned i = 4; i < 16; ++i) assert(cpu.vector_registers[0][i] == 0);
    cpu.rip = 3;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_C(0x00000000AABBCCDD));

    cpu.registers[X86EMU_RAX] = UINT64_C(0x1122334455667788);
    cpu.rip = 6;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[0], "\x88\x77\x66\x55\x44\x33\x22\x11", 8) == 0);
    for (unsigned i = 8; i < 16; ++i) assert(cpu.vector_registers[0][i] == 0);
    cpu.rip = 10;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_C(0x1122334455667788));
}

static void test_movq_xmm_forms(void)
{
    uint8_t code[128] = { 0xF3,0x0F,0x7E,0xC1,
                          0x66,0x0F,0xD6,0xC1 };
    uint8_t source[16];
    for (unsigned i = 0; i < sizeof(source); ++i) source[i] = (uint8_t)(0x90u + i);
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };

    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    memset(cpu.vector_registers[0], 0xA5, sizeof(cpu.vector_registers[0]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[0], source, 8u) == 0);
    for (unsigned i = 8u; i < 16u; ++i) assert(cpu.vector_registers[0][i] == 0u);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[0][i] == 0xA5u);

    cpu.rip = 4u;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, 8u) == 0);
    for (unsigned i = 8u; i < 16u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC5,0x83,0x7E,0xC1 }, 4u);
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    memset(cpu.vector_registers[0], 0xA5, sizeof(cpu.vector_registers[0]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[0], source, 8u) == 0);
    for (unsigned i = 8u; i < 64u; ++i) assert(cpu.vector_registers[0][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC5,0x81,0xD6,0xC8 }, 4u);
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    memset(cpu.vector_registers[0], 0xA5, sizeof(cpu.vector_registers[0]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[0], source, 8u) == 0);
    for (unsigned i = 8u; i < 64u; ++i) assert(cpu.vector_registers[0][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xF3,0x0F,0x7E,0x00 }, 4u);
    memcpy(code + 64u, source, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memset(cpu.vector_registers[0], 0xA5, sizeof(cpu.vector_registers[0]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[0], source, 8u) == 0);
    for (unsigned i = 8u; i < 16u; ++i) assert(cpu.vector_registers[0][i] == 0u);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[0][i] == 0xA5u);

    memcpy(code, (const uint8_t[]){ 0x66,0x0F,0xD6,0x08 }, 4u);
    memset(code + 64u, 0xA5, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 64u, source, 8u) == 0);

    memcpy(code, (const uint8_t[]){ 0xC5,0x83,0x7E,0x00 }, 4u);
    memcpy(code + 64u, source, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memset(cpu.vector_registers[0], 0xA5, sizeof(cpu.vector_registers[0]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[0], source, 8u) == 0);
    for (unsigned i = 8u; i < 64u; ++i) assert(cpu.vector_registers[0][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC5,0x81,0xD6,0x08 }, 4u);
    memset(code + 64u, 0xA5, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 64u, source, 8u) == 0);

    x86emu_memory short_memory = { code, 71u, 0 };
    x86emu_init(&cpu, short_memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    assert(x86emu_step(&cpu) == X86EMU_ERR_MEMORY);
    assert(cpu.rip == 0u);
}

static void test_partial_xmm_moves(void)
{
    uint8_t code[96];
    uint8_t source[16];
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    for (unsigned i = 0; i < sizeof(source); ++i) source[i] = (uint8_t)(0xA0u + i);

    memcpy(code, (const uint8_t[]){ 0x0F,0x12,0x08 }, 3u);
    memcpy(code + 40u, source, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 40u;
    memset(cpu.vector_registers[1], 0xC3, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, 8u) == 0);
    for (unsigned i = 8u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xC3u);

    memcpy(code, (const uint8_t[]){ 0x0F,0x16,0x08 }, 3u);
    memcpy(code + 40u, source, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 40u;
    memset(cpu.vector_registers[1], 0xC3, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1] + 8u, source, 8u) == 0);
    for (unsigned i = 0u; i < 8u; ++i) assert(cpu.vector_registers[1][i] == 0xC3u);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xC3u);

    memcpy(code, (const uint8_t[]){ 0x66,0x0F,0x12,0x08 }, 4u);
    memset(code + 40u, 0xC3, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 40u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 40u, (const uint8_t[8]){ 0xC3,0xC3,0xC3,0xC3,0xC3,0xC3,0xC3,0xC3 }, 8u) == 0);
    assert(memcmp(cpu.vector_registers[1], (const uint8_t[8]){ 0xC3,0xC3,0xC3,0xC3,0xC3,0xC3,0xC3,0xC3 }, 8u) == 0);
    assert(memcmp(cpu.vector_registers[1] + 8u, source + 8u, 8u) == 0);

    memcpy(code, (const uint8_t[]){ 0x66,0x0F,0x16,0x08 }, 4u);
    memcpy(code + 40u, source, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 40u;
    memset(cpu.vector_registers[1], 0xC3, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1] + 8u, source, 8u) == 0);
    for (unsigned i = 0u; i < 8u; ++i) assert(cpu.vector_registers[1][i] == 0xC3u);

    memcpy(code, (const uint8_t[]){ 0x0F,0x13,0x08 }, 3u);
    memset(code + 40u, 0xC3, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 40u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 40u, source, 8u) == 0);

    memcpy(code, (const uint8_t[]){ 0x0F,0x17,0x08 }, 3u);
    memset(code + 40u, 0xC3, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 40u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 40u, source + 8u, 8u) == 0);

    memcpy(code, (const uint8_t[]){ 0x66,0x0F,0x13,0x08 }, 4u);
    memset(code + 40u, 0xC3, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 40u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 40u, source, 8u) == 0);

    memcpy(code, (const uint8_t[]){ 0x66,0x0F,0x17,0x08 }, 4u);
    memset(code + 40u, 0xC3, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 40u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 40u, source + 8u, 8u) == 0);

    x86emu_memory short_memory = { code, 47u, 0 };
    x86emu_init(&cpu, short_memory, 0);
    cpu.registers[X86EMU_RAX] = 40u;
    assert(x86emu_step(&cpu) == X86EMU_ERR_MEMORY);
    assert(cpu.rip == 0u);
}

static void test_vmov_partial_moves(void)
{
    uint8_t code[128] = { 0 };
    uint8_t memory_value[8];
    uint8_t source[16];
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    for (unsigned i = 0; i < sizeof(memory_value); ++i) memory_value[i] = (uint8_t)(0xA0u + i);
    for (unsigned i = 0; i < sizeof(source); ++i) source[i] = (uint8_t)(0x10u + i);

    memcpy(code + 64u, memory_value, sizeof(memory_value));
    memcpy(code, (const uint8_t[]){ 0xC5,0xE8,0x12,0x08 }, 4u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[2], source, sizeof(source));
    memset(cpu.vector_registers[1], 0xCC, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], memory_value, 8u) == 0);
    assert(memcmp(cpu.vector_registers[1] + 8u, source + 8u, 8u) == 0);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC5,0xE8,0x16,0x08 }, 4u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[2], source, sizeof(source));
    memset(cpu.vector_registers[1], 0xCC, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, 8u) == 0);
    assert(memcmp(cpu.vector_registers[1] + 8u, memory_value, 8u) == 0);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC5,0xE9,0x12,0x08 }, 4u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[2], source, sizeof(source));
    memset(cpu.vector_registers[1], 0xCC, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], memory_value, 8u) == 0);
    assert(memcmp(cpu.vector_registers[1] + 8u, source + 8u, 8u) == 0);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC5,0xE9,0x16,0x08 }, 4u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[2], source, sizeof(source));
    memset(cpu.vector_registers[1], 0xCC, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, 8u) == 0);
    assert(memcmp(cpu.vector_registers[1] + 8u, memory_value, 8u) == 0);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC5,0x80,0x13,0x08 }, 4u);
    memset(code + 64u, 0xCC, sizeof(memory_value));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 64u, source, 8u) == 0);

    memcpy(code, (const uint8_t[]){ 0xC5,0x80,0x17,0x08 }, 4u);
    memset(code + 64u, 0xCC, sizeof(memory_value));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 64u, source + 8u, 8u) == 0);

    memcpy(code, (const uint8_t[]){ 0xC5,0x81,0x13,0x08 }, 4u);
    memset(code + 64u, 0xCC, sizeof(memory_value));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 64u, source, 8u) == 0);

    memcpy(code, (const uint8_t[]){ 0xC5,0x81,0x17,0x08 }, 4u);
    memset(code + 64u, 0xCC, sizeof(memory_value));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 64u, source + 8u, 8u) == 0);

    x86emu_memory short_memory = { code, 71u, 0 };
    x86emu_init(&cpu, short_memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[2], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_ERR_MEMORY);
    assert(cpu.rip == 0u);
}

static void test_vmovd_vmovq(void)
{
    uint8_t code[128] = { 0xC5,0x81,0x6E,0xC0,
                          0xC5,0x81,0x7E,0xC0,
                          0xC4,0xE1,0x81,0x6E,0xC0,
                          0xC4,0xE1,0x81,0x7E,0xC0 };
    uint8_t source[16];
    for (unsigned i = 0; i < sizeof(source); ++i) source[i] = (uint8_t)(0x70u + i);
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };

    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = UINT64_C(0xFFFFFFFFAABBCCDD);
    memset(cpu.vector_registers[0], 0xA5, sizeof(cpu.vector_registers[0]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[0], "\xDD\xCC\xBB\xAA", 4u) == 0);
    for (unsigned i = 4u; i < 64u; ++i) assert(cpu.vector_registers[0][i] == 0u);

    cpu.rip = 4u;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_C(0x00000000AABBCCDD));

    cpu.registers[X86EMU_RAX] = UINT64_C(0x1122334455667788);
    cpu.rip = 8u;
    memset(cpu.vector_registers[0], 0xA5, sizeof(cpu.vector_registers[0]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[0], "\x88\x77\x66\x55\x44\x33\x22\x11", 8u) == 0);
    for (unsigned i = 8u; i < 64u; ++i) assert(cpu.vector_registers[0][i] == 0u);

    cpu.rip = 13u;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_C(0x1122334455667788));

    memcpy(code, (const uint8_t[]){ 0xC4,0x41,0x01,0x6E,0xC8 }, 5u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_R8] = UINT64_C(0xFFFFFFFF12345678);
    memset(cpu.vector_registers[9], 0xA5, sizeof(cpu.vector_registers[9]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[9], "\x78\x56\x34\x12", 4u) == 0);
    for (unsigned i = 4u; i < 64u; ++i) assert(cpu.vector_registers[9][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC4,0x41,0x81,0x7E,0xC8 }, 5u);
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[9], "\x88\x77\x66\x55\x44\x33\x22\x11", 8u);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_R8] == UINT64_C(0x1122334455667788));

    memcpy(code, (const uint8_t[]){ 0xC5,0x81,0x6E,0x08 }, 4u);
    memcpy(code + 64u, source, 4u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memset(cpu.vector_registers[1], 0xA5, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, 4u) == 0);
    for (unsigned i = 4u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC5,0x81,0x7E,0x08 }, 4u);
    memset(code + 64u, 0xA5, 4u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 64u, source, 4u) == 0);

    memcpy(code, (const uint8_t[]){ 0xC4,0xE1,0x81,0x6E,0x08 }, 5u);
    memcpy(code + 64u, source, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memset(cpu.vector_registers[1], 0xA5, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, 8u) == 0);
    for (unsigned i = 8u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC4,0xE1,0x81,0x7E,0x08 }, 5u);
    memset(code + 64u, 0xA5, 8u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 64u, source, 8u) == 0);

    x86emu_memory short_memory = { code, 71u, 0 };
    x86emu_init(&cpu, short_memory, 0);
    cpu.registers[X86EMU_RAX] = 64u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_ERR_MEMORY);
    assert(cpu.rip == 0u);
}

static void test_sse2_integer(void)
{
    uint8_t code[64] = {
        0x66, 0x0F, 0x6F, 0xC1,
        0x66, 0x0F, 0xFC, 0xC2,
        0x66, 0x0F, 0xFA, 0xC2,
        0x66, 0x0F, 0x75, 0xC2,
        0x66, 0x0F, 0xEF, 0xC2,
        0x66, 0x0F, 0xDB, 0xC2,
        0x66, 0x0F, 0xEB, 0xC2
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint8_t left[16];
    uint8_t right[16];
    uint8_t result[16];
    x86emu_init(&cpu, memory, 0);
    for (unsigned i = 0; i < 16; ++i) {
        left[i] = (uint8_t)(i + 1u);
        right[i] = (uint8_t)(2u * i);
    }
    memcpy(cpu.vector_registers[1], left, sizeof(left));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[0], left, sizeof(left)) == 0);

    memcpy(cpu.vector_registers[0], left, sizeof(left));
    memcpy(cpu.vector_registers[2], right, sizeof(right));
    cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result, cpu.vector_registers[0], sizeof(result));
    for (unsigned i = 0; i < 16; ++i) assert(result[i] == (uint8_t)(left[i] + right[i]));

    uint32_t sub_left[4] = { 100u, 200u, 300u, 400u };
    uint32_t sub_right[4] = { 1u, 2u, 3u, 4u };
    memcpy(cpu.vector_registers[0], sub_left, sizeof(sub_left));
    memcpy(cpu.vector_registers[2], sub_right, sizeof(sub_right));
    cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    uint32_t sub_result[4];
    memcpy(sub_result, cpu.vector_registers[0], sizeof(sub_result));
    for (unsigned i = 0; i < 4; ++i) assert(sub_result[i] == sub_left[i] - sub_right[i]);

    uint16_t equal_left[8] = { 1u, 2u, 3u, 4u, 5u, 6u, 7u, 8u };
    uint16_t equal_right[8] = { 1u, 9u, 3u, 4u, 5u, 6u, 0u, 8u };
    memcpy(cpu.vector_registers[0], equal_left, sizeof(equal_left));
    memcpy(cpu.vector_registers[2], equal_right, sizeof(equal_right));
    cpu.rip = 12;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    uint16_t equal_result[8];
    memcpy(equal_result, cpu.vector_registers[0], sizeof(equal_result));
    for (unsigned i = 0; i < 8; ++i) assert(equal_result[i] == (i == 1 || i == 6 ? 0u : UINT16_MAX));

    memset(cpu.vector_registers[0], 0xAA, sizeof(result));
    memset(cpu.vector_registers[2], 0x0F, sizeof(result));
    cpu.rip = 16;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    for (unsigned i = 0; i < 16; ++i) assert(cpu.vector_registers[0][i] == 0xA5);

    memset(cpu.vector_registers[0], 0xF0, sizeof(result));
    memset(cpu.vector_registers[2], 0x0F, sizeof(result));
    cpu.rip = 20;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    for (unsigned i = 0; i < 16; ++i) assert(cpu.vector_registers[0][i] == 0x00);

    memset(cpu.vector_registers[0], 0xF0, sizeof(result));
    memset(cpu.vector_registers[2], 0x0F, sizeof(result));
    cpu.rip = 24;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    for (unsigned i = 0; i < 16; ++i) assert(cpu.vector_registers[0][i] == 0xFF);
}

static void test_packed_shifts(void)
{
    uint8_t code[32] = {
        0x66, 0x0F, 0x71, 0xF0, 0x01,
        0x66, 0x0F, 0x72, 0xD0, 0x02,
        0x66, 0x0F, 0x72, 0xE0, 0x01,
        0x66, 0x0F, 0x73, 0xF8, 0x04,
        0x66, 0x0F, 0x73, 0xD8, 0x04
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint16_t words[8] = { 1u, 0x8000u, 0xFFFFu, 0x1234u, 0u, 0u, 0u, 0u };
    uint32_t dwords[4] = { UINT32_C(0x80000000), 4u, 0u, 0u };
    uint32_t arithmetic[4] = { UINT32_C(0xFFFFFFFE), UINT32_C(0x7FFFFFFF), 0u, 0u };
    uint8_t bytes[16];
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[0], words, sizeof(words));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(words, cpu.vector_registers[0], sizeof(words));
    assert(words[0] == 2u && words[1] == 0u && words[2] == 0xFFFEu && words[3] == 0x2468u);

    memcpy(cpu.vector_registers[0], dwords, sizeof(dwords));
    cpu.rip = 5;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(dwords, cpu.vector_registers[0], sizeof(dwords));
    assert(dwords[0] == UINT32_C(0x20000000) && dwords[1] == 1u);

    memcpy(cpu.vector_registers[0], arithmetic, sizeof(arithmetic));
    cpu.rip = 10;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(arithmetic, cpu.vector_registers[0], sizeof(arithmetic));
    assert(arithmetic[0] == UINT32_C(0xFFFFFFFF) && arithmetic[1] == UINT32_C(0x3FFFFFFF));

    for (unsigned i = 0; i < 16; ++i) bytes[i] = (uint8_t)i;
    memcpy(cpu.vector_registers[0], bytes, sizeof(bytes));
    cpu.rip = 15;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    for (unsigned i = 0; i < 4; ++i) assert(cpu.vector_registers[0][i] == 0);
    for (unsigned i = 4; i < 16; ++i) assert(cpu.vector_registers[0][i] == (uint8_t)(i - 4u));

    memcpy(cpu.vector_registers[0], bytes, sizeof(bytes));
    cpu.rip = 20;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    for (unsigned i = 0; i < 12; ++i) assert(cpu.vector_registers[0][i] == (uint8_t)(i + 4u));
    for (unsigned i = 12; i < 16; ++i) assert(cpu.vector_registers[0][i] == 0);
}

static void test_packed_multiply(void)
{
    uint8_t code[24] = {
        0x66, 0x0F, 0xD5, 0xC1,
        0x66, 0x0F, 0xE5, 0xC1,
        0x66, 0x0F, 0xE4, 0xC1,
        0x66, 0x0F, 0xF4, 0xC1
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint16_t left_words[8] = { 2u, 0x4000u, 0x8000u, 0xFFFFu, 3u, 5u, 7u, 9u };
    uint16_t right_words[8] = { 3u, 4u, 2u, 0xFFFFu, 11u, 13u, 17u, 19u };
    uint16_t result_words[8];
    uint32_t left_dwords[4] = { UINT32_MAX, 0u, 3u, 0u };
    uint32_t right_dwords[4] = { 2u, 0u, 4u, 0u };
    uint64_t result_qwords[2];
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[1], right_words, sizeof(right_words));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[0] == 6u && result_words[1] == 0x0000u && result_words[2] == 0u && result_words[3] == 1u);

    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[1], right_words, sizeof(right_words));
    cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[1] == 1u && result_words[2] == 0xFFFFu);

    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[1], right_words, sizeof(right_words));
    cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[3] == 0xFFFEu);

    memcpy(cpu.vector_registers[0], left_dwords, sizeof(left_dwords));
    memcpy(cpu.vector_registers[1], right_dwords, sizeof(right_dwords));
    cpu.rip = 12;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_qwords, cpu.vector_registers[0], sizeof(result_qwords));
    assert(result_qwords[0] == UINT64_C(0x00000001FFFFFFFE) && result_qwords[1] == 12u);
}

static void test_pmulld(void)
{
    static const uint8_t legacy_code[] = { 0x66,0x0F,0x38,0x40,0xC1 };
    static const uint8_t vex128_code[] = { 0xC4,0xE2,0x61,0x40,0xCA };
    static const uint8_t vex256_code[] = { 0xC4,0xE2,0x65,0x40,0xCA };
    uint32_t left[8] = { UINT32_C(0x80000000), UINT32_C(0xFFFFFFFD), UINT32_C(0x7FFFFFFF), UINT32_C(0xFFFFFFFF), 5u, UINT32_C(0xFFFFFFFE), 7u, 9u };
    uint32_t right[8] = { 2u, UINT32_C(0xFFFFFFFC), 2u, UINT32_C(0xFFFFFFFF), 3u, 4u, 6u, 8u };
    uint32_t expected[8] = { 0u, 12u, UINT32_C(0xFFFFFFFE), 1u, 15u, UINT32_C(0xFFFFFFF8), 42u, 72u };
    x86emu_cpu cpu;
    x86emu_memory memory = { (uint8_t *)legacy_code, sizeof(legacy_code), 0 };
    x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[0], left, 16); memcpy(cpu.vector_registers[1], right, 16); memset(cpu.vector_registers[0] + 16u, 0xA5, 48u); assert(x86emu_step(&cpu) == X86EMU_OK); uint32_t result[8]; memcpy(result, cpu.vector_registers[0], 16); for (unsigned i = 0; i < 4u; ++i) assert(result[i] == expected[i]); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[0][i] == 0xA5u);
    memory = (x86emu_memory){ (uint8_t *)vex128_code, sizeof(vex128_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[3], left, 16); memcpy(cpu.vector_registers[2], right, 16); memset(cpu.vector_registers[1] + 16u, 0xA5, 48u); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], 16); for (unsigned i = 0; i < 4u; ++i) assert(result[i] == expected[i]); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    memory = (x86emu_memory){ (uint8_t *)vex256_code, sizeof(vex256_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[3], left, sizeof(left)); memcpy(cpu.vector_registers[2], right, sizeof(right)); memset(cpu.vector_registers[1] + 32u, 0xA5, 32u); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); for (unsigned i = 0; i < 8u; ++i) assert(result[i] == expected[i]); for (unsigned i = 32u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    uint8_t memory_code[64] = { 0xC4,0xE2,0x65,0x40,0x08 }; memcpy(memory_code + 32u, right, sizeof(right)); memory = (x86emu_memory){ memory_code, sizeof(memory_code), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[3], left, sizeof(left)); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); for (unsigned i = 0; i < 8u; ++i) assert(result[i] == expected[i]);
}

static void test_vpacked_multiply(void)
{
    uint8_t code[20] = {
        0xC5, 0xE5, 0xD5, 0xCA,
        0xC5, 0xE5, 0xE5, 0xCA,
        0xC5, 0xE5, 0xE4, 0xCA,
        0xC5, 0xE5, 0xF4, 0xCA
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint16_t left_words[16] = { 2u, 0x4000u, 0x8000u, 0xFFFFu, 3u, 5u, 7u, 9u,
                                11u, 13u, 15u, 17u, 19u, 21u, 23u, 25u };
    uint16_t right_words[16] = { 3u, 4u, 2u, 0xFFFFu, 11u, 13u, 17u, 19u,
                                 2u, 3u, 4u, 5u, 6u, 7u, 8u, 9u };
    uint16_t result_words[16];
    uint32_t left_dwords[8] = { UINT32_MAX, 0u, 3u, 0u, 5u, 0u, 7u, 0u };
    uint32_t right_dwords[8] = { 2u, 0u, 4u, 0u, 6u, 0u, 8u, 0u };
    uint64_t result_qwords[4];
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[2], right_words, sizeof(right_words));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[1], sizeof(result_words));
    assert(result_words[0] == 6u && result_words[1] == 0u && result_words[2] == 0u && result_words[3] == 1u);

    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[2], right_words, sizeof(right_words));
    cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[1], sizeof(result_words));
    assert(result_words[1] == 1u && result_words[2] == 0xFFFFu);

    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[2], right_words, sizeof(right_words));
    cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[1], sizeof(result_words));
    assert(result_words[3] == 0xFFFEu);

    memcpy(cpu.vector_registers[3], left_dwords, sizeof(left_dwords));
    memcpy(cpu.vector_registers[2], right_dwords, sizeof(right_dwords));
    cpu.rip = 12;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_qwords, cpu.vector_registers[1], sizeof(result_qwords));
    assert(result_qwords[0] == UINT64_C(0x00000001FFFFFFFE) && result_qwords[1] == 12u &&
           result_qwords[2] == 30u && result_qwords[3] == 56u);
}

static void test_vpacked_shifts(void)
{
    uint8_t code[40] = {
        0xC5, 0xF5, 0x71, 0xF2, 0x01,
        0xC5, 0xF5, 0x72, 0xD2, 0x02,
        0xC5, 0xF5, 0x72, 0xE2, 0x01,
        0xC5, 0xF5, 0x73, 0xF2, 0x03,
        0xC5, 0xF5, 0x73, 0xD2, 0x03,
        0xC5, 0xF5, 0x73, 0xFA, 0x04,
        0xC5, 0xF5, 0x73, 0xDA, 0x04
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint16_t words[16] = { 1u, 0x8000u, 0xFFFFu, 0x1234u, 0u, 0u, 0u, 0u,
                           0u, 0u, 0u, 0u, 0u, 0u, 0u, 0u };
    uint32_t dwords[8] = { UINT32_C(0x80000000), 4u, 0u, 0u, 0u, 0u, 0u, 0u };
    uint8_t bytes[32];
    uint64_t observed;
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[2], words, sizeof(words));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(words, cpu.vector_registers[1], sizeof(words));
    assert(words[0] == 2u && words[1] == 0u && words[2] == 0xFFFEu && words[3] == 0x2468u);

    memcpy(cpu.vector_registers[2], dwords, sizeof(dwords));
    cpu.rip = 5;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(dwords, cpu.vector_registers[1], sizeof(dwords));
    assert(dwords[0] == UINT32_C(0x20000000) && dwords[1] == 1u);

    dwords[0] = UINT32_C(0xFFFFFFFE);
    dwords[1] = UINT32_C(0x7FFFFFFF);
    memcpy(cpu.vector_registers[2], dwords, sizeof(dwords));
    cpu.rip = 10;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(dwords, cpu.vector_registers[1], sizeof(dwords));
    assert(dwords[0] == UINT32_C(0xFFFFFFFF) && dwords[1] == UINT32_C(0x3FFFFFFF));

    memset(bytes, 0, sizeof(bytes));
    for (unsigned i = 0; i < sizeof(bytes); ++i) bytes[i] = (uint8_t)i;
    memcpy(cpu.vector_registers[2], bytes, sizeof(bytes));
    cpu.rip = 15;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(bytes, cpu.vector_registers[1], sizeof(bytes));
    memcpy(&observed, bytes, sizeof(observed));
    assert(observed == UINT64_C(0x3830282018100800));

    for (unsigned i = 0; i < sizeof(bytes); ++i) bytes[i] = (uint8_t)i;
    memcpy(cpu.vector_registers[2], bytes, sizeof(bytes));
    cpu.rip = 20;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(bytes, cpu.vector_registers[1], sizeof(bytes));
    memcpy(&observed, bytes, sizeof(observed));
    assert(observed == UINT64_C(0x00E0C0A080604020));

    for (unsigned i = 0; i < sizeof(bytes); ++i) bytes[i] = (uint8_t)i;
    memcpy(cpu.vector_registers[2], bytes, sizeof(bytes));
    cpu.rip = 25;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.vector_registers[1][0] == 0 && cpu.vector_registers[1][4] == 0 && cpu.vector_registers[1][20] == 16);

    memcpy(cpu.vector_registers[2], bytes, sizeof(bytes));
    cpu.rip = 30;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.vector_registers[1][0] == 4 && cpu.vector_registers[1][12] == 0 && cpu.vector_registers[1][16] == 20);
}

static void test_legacy_sse41_minmax(void)
{
    uint8_t code[48] = {
        0x66,0x0F,0x38,0x38,0xC1, 0x66,0x0F,0x38,0x3C,0xC1,
        0x66,0x0F,0x38,0x3A,0xC1, 0x66,0x0F,0x38,0x3E,0xC1,
        0x66,0x0F,0x38,0x39,0xC1, 0x66,0x0F,0x38,0x3D,0xC1,
        0x66,0x0F,0x38,0x3B,0xC1, 0x66,0x0F,0x38,0x3F,0xC1
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    int8_t signed_bytes_left[16] = { INT8_MIN, -1, 0, INT8_MAX, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120 };
    int8_t signed_bytes_right[16] = { INT8_MAX, -2, -3, 126, 9, 21, 29, 41, 49, 61, 69, 81, 89, 101, 109, 121 };
    uint16_t words_left[8] = { 0u, UINT16_MAX, UINT16_C(0x8000), UINT16_C(0x7FFF), 4u, 5u, 6u, 7u };
    uint16_t words_right[8] = { 1u, 0u, UINT16_C(0x7FFF), UINT16_C(0x8000), 3u, 6u, 5u, 8u };
    int32_t dwords_left[4] = { INT32_MIN, -1, 0, INT32_MAX };
    int32_t dwords_right[4] = { INT32_MAX, INT32_MIN, -1, 0 };
    uint32_t unsigned_left[4] = { 0u, UINT32_MAX, UINT32_C(0x80000000), 1u };
    uint32_t unsigned_right[4] = { 1u, 0u, UINT32_C(0x7FFFFFFF), UINT32_MAX };
    uint8_t expected[16];
    x86emu_init(&cpu, memory, 0);
    memset(cpu.vector_registers[0] + 16u, 0xA5, 48u);

    memcpy(cpu.vector_registers[0], signed_bytes_left, sizeof(signed_bytes_left)); memcpy(cpu.vector_registers[1], signed_bytes_right, sizeof(signed_bytes_right)); assert(x86emu_step(&cpu) == X86EMU_OK); for (unsigned i = 0; i < 16u; ++i) expected[i] = (uint8_t)((signed_bytes_left[i] < signed_bytes_right[i]) ? signed_bytes_left[i] : signed_bytes_right[i]); assert(memcmp(cpu.vector_registers[0], expected, 16) == 0);
    memcpy(cpu.vector_registers[0], signed_bytes_left, sizeof(signed_bytes_left)); memcpy(cpu.vector_registers[1], signed_bytes_right, sizeof(signed_bytes_right)); cpu.rip = 5; assert(x86emu_step(&cpu) == X86EMU_OK); for (unsigned i = 0; i < 16u; ++i) expected[i] = (uint8_t)((signed_bytes_left[i] > signed_bytes_right[i]) ? signed_bytes_left[i] : signed_bytes_right[i]); assert(memcmp(cpu.vector_registers[0], expected, 16) == 0);
    memcpy(cpu.vector_registers[0], words_left, sizeof(words_left)); memcpy(cpu.vector_registers[1], words_right, sizeof(words_right)); cpu.rip = 10; assert(x86emu_step(&cpu) == X86EMU_OK); { uint16_t out[8]; memcpy(out, cpu.vector_registers[0], sizeof(out)); for (unsigned i = 0; i < 8u; ++i) assert(out[i] == (words_left[i] < words_right[i] ? words_left[i] : words_right[i])); }
    memcpy(cpu.vector_registers[0], words_left, sizeof(words_left)); memcpy(cpu.vector_registers[1], words_right, sizeof(words_right)); cpu.rip = 15; assert(x86emu_step(&cpu) == X86EMU_OK); { uint16_t out[8]; memcpy(out, cpu.vector_registers[0], sizeof(out)); for (unsigned i = 0; i < 8u; ++i) assert(out[i] == (words_left[i] > words_right[i] ? words_left[i] : words_right[i])); }
    memcpy(cpu.vector_registers[0], dwords_left, sizeof(dwords_left)); memcpy(cpu.vector_registers[1], dwords_right, sizeof(dwords_right)); cpu.rip = 20; assert(x86emu_step(&cpu) == X86EMU_OK); { int32_t out[4]; memcpy(out, cpu.vector_registers[0], sizeof(out)); for (unsigned i = 0; i < 4u; ++i) assert(out[i] == (dwords_left[i] < dwords_right[i] ? dwords_left[i] : dwords_right[i])); }
    memcpy(cpu.vector_registers[0], dwords_left, sizeof(dwords_left)); memcpy(cpu.vector_registers[1], dwords_right, sizeof(dwords_right)); cpu.rip = 25; assert(x86emu_step(&cpu) == X86EMU_OK); { int32_t out[4]; memcpy(out, cpu.vector_registers[0], sizeof(out)); for (unsigned i = 0; i < 4u; ++i) assert(out[i] == (dwords_left[i] > dwords_right[i] ? dwords_left[i] : dwords_right[i])); }
    memcpy(cpu.vector_registers[0], unsigned_left, sizeof(unsigned_left)); memcpy(cpu.vector_registers[1], unsigned_right, sizeof(unsigned_right)); cpu.rip = 30; assert(x86emu_step(&cpu) == X86EMU_OK); { uint32_t out[4]; memcpy(out, cpu.vector_registers[0], sizeof(out)); for (unsigned i = 0; i < 4u; ++i) assert(out[i] == (unsigned_left[i] < unsigned_right[i] ? unsigned_left[i] : unsigned_right[i])); }
    memcpy(cpu.vector_registers[0], unsigned_left, sizeof(unsigned_left)); memcpy(cpu.vector_registers[1], unsigned_right, sizeof(unsigned_right)); cpu.rip = 35; assert(x86emu_step(&cpu) == X86EMU_OK); { uint32_t out[4]; memcpy(out, cpu.vector_registers[0], sizeof(out)); for (unsigned i = 0; i < 4u; ++i) assert(out[i] == (unsigned_left[i] > unsigned_right[i] ? unsigned_left[i] : unsigned_right[i])); }
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[0][i] == 0xA5u);

    uint8_t memory_code[48] = { 0x66,0x0F,0x38,0x3B,0x00 }; memcpy(memory_code + 32u, unsigned_right, sizeof(unsigned_right)); memory = (x86emu_memory){ memory_code, sizeof(memory_code), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[0], unsigned_left, sizeof(unsigned_left)); assert(x86emu_step(&cpu) == X86EMU_OK); { uint32_t out[4]; memcpy(out, cpu.vector_registers[0], sizeof(out)); for (unsigned i = 0; i < 4u; ++i) assert(out[i] == (unsigned_left[i] < unsigned_right[i] ? unsigned_left[i] : unsigned_right[i])); }
}

static void test_variable_shifts(void)
{
    uint8_t code[32] = {
        0xC4, 0xE2, 0x65, 0x47, 0xCA,
        0xC4, 0xE2, 0x65, 0x45, 0xCA,
        0xC4, 0xE2, 0x65, 0x46, 0xCA,
        0xC4, 0xE2, 0xE5, 0x47, 0xCA,
        0xC4, 0xE2, 0xE5, 0x45, 0xCA
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint32_t source_d[8] = { 1u, UINT32_C(0x80000000), 4u, UINT32_MAX, 0u, 0u, 0u, 0u };
    uint32_t counts_d[8] = { 1u, 1u, 3u, 32u, 0u, 0u, 0u, 0u };
    uint64_t source_q[4] = { 1u, UINT64_C(0x8000000000000000), 0u, 0u };
    uint64_t counts_q[4] = { 3u, 1u, 0u, 0u };
    uint32_t result_d[8];
    uint64_t result_q[4];
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[3], source_d, sizeof(source_d));
    memcpy(cpu.vector_registers[2], counts_d, sizeof(counts_d));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_d, cpu.vector_registers[1], sizeof(result_d));
    assert(result_d[0] == 2u && result_d[1] == 0u && result_d[2] == 32u && result_d[3] == 0u);

    source_d[0] = 4u;
    source_d[1] = UINT32_C(0x80000000);
    memcpy(cpu.vector_registers[3], source_d, sizeof(source_d));
    memcpy(cpu.vector_registers[2], counts_d, sizeof(counts_d));
    cpu.rip = 5;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_d, cpu.vector_registers[1], sizeof(result_d));
    assert(result_d[0] == 2u && result_d[1] == UINT32_C(0x40000000));

    source_d[0] = UINT32_C(0xFFFFFFFC);
    source_d[1] = UINT32_C(0x80000000);
    memcpy(cpu.vector_registers[3], source_d, sizeof(source_d));
    memcpy(cpu.vector_registers[2], counts_d, sizeof(counts_d));
    cpu.rip = 10;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_d, cpu.vector_registers[1], sizeof(result_d));
    assert(result_d[0] == UINT32_C(0xFFFFFFFE) && result_d[1] == UINT32_C(0xC0000000));

    memcpy(cpu.vector_registers[3], source_q, sizeof(source_q));
    memcpy(cpu.vector_registers[2], counts_q, sizeof(counts_q));
    cpu.rip = 15;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_q, cpu.vector_registers[1], sizeof(result_q));
    assert(result_q[0] == 8u && result_q[1] == 0u);

    source_q[0] = 8u;
    source_q[1] = UINT64_C(0x8000000000000000);
    memcpy(cpu.vector_registers[3], source_q, sizeof(source_q));
    memcpy(cpu.vector_registers[2], counts_q, sizeof(counts_q));
    cpu.rip = 20;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_q, cpu.vector_registers[1], sizeof(result_q));
    assert(result_q[0] == 1u && result_q[1] == UINT64_C(0x4000000000000000));
}

static void test_unpack_pack(void)
{
    uint8_t code[36] = {
        0x66, 0x0F, 0x60, 0xC1, 0x66, 0x0F, 0x61, 0xC1,
        0x66, 0x0F, 0x62, 0xC1, 0x66, 0x0F, 0x68, 0xC1,
        0x66, 0x0F, 0x69, 0xC1, 0x66, 0x0F, 0x6A, 0xC1,
        0x66, 0x0F, 0x63, 0xC1, 0x66, 0x0F, 0x67, 0xC1,
        0x66, 0x0F, 0x6B, 0xC1
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint8_t left_bytes[16];
    uint8_t right_bytes[16];
    uint16_t left_words[8] = { 1u, 2u, 3u, 4u, 5u, 6u, 7u, 8u };
    uint16_t right_words[8] = { 101u, 102u, 103u, 104u, 105u, 106u, 107u, 108u };
    uint32_t left_dwords[4] = { 1u, 2u, 3u, 4u };
    uint32_t right_dwords[4] = { 101u, 102u, 103u, 104u };
    uint8_t result[16];
    uint8_t expected_low[16] = { 1u, 101u, 2u, 102u, 3u, 103u, 4u, 104u, 5u, 105u, 6u, 106u, 7u, 107u, 8u, 108u };
    int16_t pack_words[8] = { -200, -1, 0, 127, 128, 300, 32767, -32768 };
    int16_t pack_words_right[8] = { 1, 2, 3, 4, 5, 6, 7, 8 };
    int32_t pack_dwords[4] = { INT32_C(-40000), INT32_C(-32768), INT32_C(32767), INT32_C(40000) };
    int32_t pack_dwords_right[4] = { -1, 0, 1, 2 };
    x86emu_init(&cpu, memory, 0);
    for (unsigned i = 0; i < 16u; ++i) { left_bytes[i] = (uint8_t)(i + 1u); right_bytes[i] = (uint8_t)(i + 101u); }
    memcpy(cpu.vector_registers[0], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[1], right_bytes, sizeof(right_bytes));
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[0], sizeof(result)); assert(memcmp(result, expected_low, sizeof(result)) == 0);
    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[1], right_words, sizeof(right_words)); cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[0], sizeof(result)); { uint16_t out[8]; memcpy(out, result, sizeof(out)); assert(out[0] == 1u && out[1] == 101u && out[2] == 2u && out[3] == 102u && out[4] == 3u && out[5] == 103u && out[6] == 4u && out[7] == 104u); }
    memcpy(cpu.vector_registers[0], left_dwords, sizeof(left_dwords)); memcpy(cpu.vector_registers[1], right_dwords, sizeof(right_dwords)); cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[0], sizeof(result)); { uint32_t out[4]; memcpy(out, result, sizeof(out)); assert(out[0] == 1u && out[1] == 101u && out[2] == 2u && out[3] == 102u); }
    memcpy(cpu.vector_registers[0], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[1], right_bytes, sizeof(right_bytes)); cpu.rip = 12;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[0], sizeof(result)); assert(result[0] == 9u && result[1] == 109u && result[14] == 16u && result[15] == 116u);
    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[1], right_words, sizeof(right_words)); cpu.rip = 16;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[0], sizeof(result)); { uint16_t out[8]; memcpy(out, result, sizeof(out)); assert(out[0] == 5u && out[1] == 105u && out[6] == 8u && out[7] == 108u); }
    memcpy(cpu.vector_registers[0], left_dwords, sizeof(left_dwords)); memcpy(cpu.vector_registers[1], right_dwords, sizeof(right_dwords)); cpu.rip = 20;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[0], sizeof(result)); { uint32_t out[4]; memcpy(out, result, sizeof(out)); assert(out[0] == 3u && out[1] == 103u && out[2] == 4u && out[3] == 104u); }
    memcpy(cpu.vector_registers[0], pack_words, sizeof(pack_words)); memcpy(cpu.vector_registers[1], pack_words_right, sizeof(pack_words_right)); cpu.rip = 24;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[0], sizeof(result)); assert((int8_t)result[0] == -128 && (int8_t)result[1] == -1 && (int8_t)result[4] == 127 && (int8_t)result[7] == -128 && result[8] == 1u);
    memcpy(cpu.vector_registers[0], pack_words, sizeof(pack_words)); memcpy(cpu.vector_registers[1], pack_words_right, sizeof(pack_words_right)); cpu.rip = 28;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[0], sizeof(result)); assert(result[0] == 0u && result[1] == 0u && result[4] == 128u && result[5] == 255u && result[8] == 1u);
    memcpy(cpu.vector_registers[0], pack_dwords, sizeof(pack_dwords)); memcpy(cpu.vector_registers[1], pack_dwords_right, sizeof(pack_dwords_right)); cpu.rip = 32;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[0], sizeof(result)); { int16_t out[8]; memcpy(out, result, sizeof(out)); assert(out[0] == -32768 && out[1] == -32768 && out[2] == 32767 && out[3] == 32767 && out[4] == -1 && out[5] == 0); }
}

static void test_saturating_and_minmax(void)
{
    uint8_t code[64] = {
        0x66, 0x0F, 0xDC, 0xC1, 0x66, 0x0F, 0xDD, 0xC1,
        0x66, 0x0F, 0xEC, 0xC1, 0x66, 0x0F, 0xED, 0xC1,
        0x66, 0x0F, 0xD8, 0xC1, 0x66, 0x0F, 0xD9, 0xC1,
        0x66, 0x0F, 0xE8, 0xC1, 0x66, 0x0F, 0xE9, 0xC1,
        0x66, 0x0F, 0xDA, 0xC1, 0x66, 0x0F, 0xDE, 0xC1,
        0x66, 0x0F, 0xEA, 0xC1, 0x66, 0x0F, 0xEE, 0xC1,
        0x66, 0x0F, 0xE0, 0xC1, 0x66, 0x0F, 0xE3, 0xC1,
        0x66, 0x0F, 0xF6, 0xC1
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint8_t left_bytes[16] = { 250u, 0u, 120u, 0u, 2u, 0u, 0x88u, 0u, 1u, 0u, 0xFEu, 0u, 2u, 0u, 0u, 0u };
    uint8_t right_bytes[16] = { 10u, 0u, 20u, 0u, 10u, 0u, 20u, 0u, 2u, 0u, 1u, 0u, 3u, 0u, 0u, 0u };
    uint16_t left_words[8] = { 0u, 65530u, 0u, 32760u, 0u, 0u, 0u, 0u };
    uint16_t right_words[8] = { 0u, 10u, 0u, 10u, 0u, 0u, 0u, 0u };
    uint8_t result_bytes[16];
    uint16_t result_words[8];
    x86emu_init(&cpu, memory, 0);

    memcpy(cpu.vector_registers[0], left_bytes, sizeof(left_bytes));
    memcpy(cpu.vector_registers[1], right_bytes, sizeof(right_bytes));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_bytes, cpu.vector_registers[0], sizeof(result_bytes));
    assert(result_bytes[0] == 255u);

    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[1], right_words, sizeof(right_words));
    cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[1] == 65535u);

    left_bytes[0] = 120u; right_bytes[0] = 20u;
    memcpy(cpu.vector_registers[0], left_bytes, sizeof(left_bytes));
    memcpy(cpu.vector_registers[1], right_bytes, sizeof(right_bytes));
    cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_bytes, cpu.vector_registers[0], sizeof(result_bytes));
    assert(result_bytes[0] == 127u);

    left_words[0] = 0u; left_words[1] = 32760u;
    right_words[0] = 0u; right_words[1] = 10u;
    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[1], right_words, sizeof(right_words));
    cpu.rip = 12;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[1] == 32767u);

    left_bytes[0] = 2u; right_bytes[0] = 10u;
    memcpy(cpu.vector_registers[0], left_bytes, sizeof(left_bytes));
    memcpy(cpu.vector_registers[1], right_bytes, sizeof(right_bytes));
    cpu.rip = 16;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_bytes, cpu.vector_registers[0], sizeof(result_bytes));
    assert(result_bytes[0] == 0u);

    left_words[0] = 2u; left_words[1] = 0u; right_words[0] = 10u; right_words[1] = 0u;
    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[1], right_words, sizeof(right_words));
    cpu.rip = 20;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[0] == 0u);

    left_bytes[0] = 0x88u; right_bytes[0] = 20u;
    memcpy(cpu.vector_registers[0], left_bytes, sizeof(left_bytes));
    memcpy(cpu.vector_registers[1], right_bytes, sizeof(right_bytes));
    cpu.rip = 24;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_bytes, cpu.vector_registers[0], sizeof(result_bytes));
    assert((int8_t)result_bytes[0] == -128);

    left_words[0] = 0u; left_words[1] = 0x8008u; right_words[0] = 0u; right_words[1] = 20u;
    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[1], right_words, sizeof(right_words));
    cpu.rip = 28;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[1] == 0x8000u);

    left_bytes[0] = 1u; right_bytes[0] = 2u;
    memcpy(cpu.vector_registers[0], left_bytes, sizeof(left_bytes));
    memcpy(cpu.vector_registers[1], right_bytes, sizeof(right_bytes));
    cpu.rip = 32;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_bytes, cpu.vector_registers[0], sizeof(result_bytes));
    assert(result_bytes[0] == 1u);
    cpu.rip = 36;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_bytes, cpu.vector_registers[0], sizeof(result_bytes));
    assert(result_bytes[0] == 2u);

    left_words[0] = 0xFFFEu; right_words[0] = 1u;
    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[1], right_words, sizeof(right_words));
    cpu.rip = 40;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[0] == 0xFFFEu);
    cpu.rip = 44;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[0] == 1u);

    left_bytes[0] = 2u; right_bytes[0] = 3u;
    memcpy(cpu.vector_registers[0], left_bytes, sizeof(left_bytes));
    memcpy(cpu.vector_registers[1], right_bytes, sizeof(right_bytes));
    cpu.rip = 48;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_bytes, cpu.vector_registers[0], sizeof(result_bytes));
    assert(result_bytes[0] == 3u);
    left_words[0] = 2u; right_words[0] = 3u;
    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[1], right_words, sizeof(right_words));
    cpu.rip = 52;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[0] == 3u);

    for (unsigned i = 0; i < 8u; ++i) { left_bytes[i] = (uint8_t)i; right_bytes[i] = (uint8_t)(i + 1u); }
    memset(left_bytes + 8, 0, 8); memset(right_bytes + 8, 0, 8);
    memcpy(cpu.vector_registers[0], left_bytes, sizeof(left_bytes));
    memcpy(cpu.vector_registers[1], right_bytes, sizeof(right_bytes));
    cpu.rip = 56;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_words, cpu.vector_registers[0], sizeof(result_words));
    assert(result_words[0] == 8u && result_words[1] == 0u && result_words[2] == 0u && result_words[3] == 0u);
}

static void test_v_saturating_and_minmax(void)
{
    uint8_t code[64] = {
        0xC5, 0xE5, 0xDC, 0xCA, 0xC5, 0xE5, 0xDD, 0xCA,
        0xC5, 0xE5, 0xEC, 0xCA, 0xC5, 0xE5, 0xED, 0xCA,
        0xC5, 0xE5, 0xD8, 0xCA, 0xC5, 0xE5, 0xD9, 0xCA,
        0xC5, 0xE5, 0xE8, 0xCA, 0xC5, 0xE5, 0xE9, 0xCA,
        0xC5, 0xE5, 0xDA, 0xCA, 0xC5, 0xE5, 0xDE, 0xCA,
        0xC5, 0xE5, 0xEA, 0xCA, 0xC5, 0xE5, 0xEE, 0xCA,
        0xC5, 0xE5, 0xE0, 0xCA, 0xC5, 0xE5, 0xE3, 0xCA,
        0xC5, 0xE5, 0xF6, 0xCA
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint8_t left_bytes[32] = { 250u, 120u, 2u, 0x88u, 1u, 0xFEu };
    uint8_t right_bytes[32] = { 10u, 20u, 10u, 20u, 2u, 1u };
    uint16_t left_words[16] = { 65530u, 0x8008u, 0xFFFEu, 2u };
    uint16_t right_words[16] = { 10u, 20u, 1u, 3u };
    uint8_t result_bytes[32];
    uint16_t result_words[16];
    uint64_t observed;
    x86emu_init(&cpu, memory, 0);

    memcpy(cpu.vector_registers[3], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes));
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_bytes, cpu.vector_registers[1], sizeof(result_bytes)); assert(result_bytes[0] == 255u);
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_words, cpu.vector_registers[1], sizeof(result_words)); assert(result_words[0] == 65535u);
    memcpy(cpu.vector_registers[3], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes)); cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_bytes, cpu.vector_registers[1], sizeof(result_bytes)); assert(result_bytes[1] == 127u);
    left_words[0] = 32760u; right_words[0] = 10u;
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 12;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_words, cpu.vector_registers[1], sizeof(result_words)); assert(result_words[0] == 32767u);
    memcpy(cpu.vector_registers[3], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes)); cpu.rip = 16;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_bytes, cpu.vector_registers[1], sizeof(result_bytes)); assert(result_bytes[2] == 0u);
    left_words[2] = 2u; right_words[2] = 10u;
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 20;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_words, cpu.vector_registers[1], sizeof(result_words)); assert(result_words[2] == 0u);
    memcpy(cpu.vector_registers[3], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes)); cpu.rip = 24;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_bytes, cpu.vector_registers[1], sizeof(result_bytes)); assert((int8_t)result_bytes[3] == -128);
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 28;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_words, cpu.vector_registers[1], sizeof(result_words)); assert(result_words[1] == 0x8000u);
    memcpy(cpu.vector_registers[3], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes)); cpu.rip = 32;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_bytes, cpu.vector_registers[1], sizeof(result_bytes)); assert(result_bytes[4] == 1u);
    cpu.rip = 36; assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_bytes, cpu.vector_registers[1], sizeof(result_bytes)); assert(result_bytes[4] == 2u);
    left_words[2] = 0xFFFEu; right_words[2] = 1u;
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 40;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_words, cpu.vector_registers[1], sizeof(result_words)); assert(result_words[2] == 0xFFFEu);
    left_words[3] = 2u; right_words[3] = 3u;
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 44;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_words, cpu.vector_registers[1], sizeof(result_words)); assert(result_words[3] == 3u);
    memcpy(cpu.vector_registers[3], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes)); cpu.rip = 48;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_bytes, cpu.vector_registers[1], sizeof(result_bytes)); assert(result_bytes[0] == 130u);
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 52;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_words, cpu.vector_registers[1], sizeof(result_words)); assert(result_words[3] == 3u);
    for (unsigned i = 0; i < 8u; ++i) { left_bytes[i] = (uint8_t)i; right_bytes[i] = (uint8_t)(i + 1u); }
    memset(left_bytes + 8, 0, 24); memset(right_bytes + 8, 0, 24);
    memcpy(cpu.vector_registers[3], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes)); cpu.rip = 56;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_bytes, cpu.vector_registers[1], sizeof(result_bytes)); memcpy(&observed, result_bytes, sizeof(observed)); assert(observed == 8u);
}

static void test_ptest(void)
{
    uint8_t code[10] = {
        0x66, 0x0F, 0x38, 0x17, 0xCA,
        0xC4, 0xE2, 0x05, 0x17, 0xCA
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    x86emu_init(&cpu, memory, 0);
    memset(cpu.vector_registers[1], 0xF0, 32);
    memset(cpu.vector_registers[2], 0x0F, 32);
    cpu.rflags |= X86EMU_FLAG_CF | X86EMU_FLAG_ZF | X86EMU_FLAG_OF |
                  X86EMU_FLAG_AF | X86EMU_FLAG_PF | X86EMU_FLAG_SF;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert((cpu.rflags & X86EMU_FLAG_ZF) != 0);
    assert((cpu.rflags & X86EMU_FLAG_CF) == 0);
    assert((cpu.rflags & (X86EMU_FLAG_OF | X86EMU_FLAG_AF | X86EMU_FLAG_PF | X86EMU_FLAG_SF)) == 0);
    for (unsigned i = 0; i < 32u; ++i) assert(cpu.vector_registers[1][i] == 0xF0u);

    memset(cpu.vector_registers[1], 0xFF, 32);
    memset(cpu.vector_registers[2], 0x0F, 32);
    cpu.rip = 5;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert((cpu.rflags & X86EMU_FLAG_ZF) == 0);
    assert((cpu.rflags & X86EMU_FLAG_CF) != 0);
    for (unsigned i = 0; i < 32u; ++i) assert(cpu.vector_registers[1][i] == 0xFFu);
}

static void test_mask_extraction(void)
{
    uint8_t code[23] = {
        0x0F, 0x50, 0xC2,
        0x66, 0x0F, 0x50, 0xC2,
        0x66, 0x0F, 0xD7, 0xC2,
        0xC5, 0x84, 0x50, 0xC2,
        0xC5, 0x85, 0x50, 0xC2,
        0xC5, 0x85, 0xD7, 0xC2
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint32_t single[8] = { UINT32_C(0x80000000), 0u, UINT32_C(0x80000000), 0u,
                           0u, UINT32_C(0x80000000), 0u, UINT32_C(0x80000000) };
    uint64_t legacy_doubles[2] = { 0u, UINT64_C(0x8000000000000000) };
    uint64_t doubles[4] = { UINT64_C(0x8000000000000000), 0u,
                            0u, UINT64_C(0x8000000000000000) };
    uint8_t bytes[32] = { 0x80u, 0u, 0u, 0x80u };
    x86emu_init(&cpu, memory, 0);

    memcpy(cpu.vector_registers[2], single, sizeof(single));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 5u);

    memcpy(cpu.vector_registers[2], legacy_doubles, sizeof(legacy_doubles));
    cpu.rip = 3;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 2u);

    memset(cpu.vector_registers[2], 0, sizeof(cpu.vector_registers[2]));
    for (unsigned i = 0; i < 32u; ++i) cpu.vector_registers[2][i] = (uint8_t)(i == 0u || i == 3u || i == 15u ? 0x80u : 0u);
    cpu.rip = 7;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_C(0x8009));

    memcpy(cpu.vector_registers[2], single, sizeof(single));
    cpu.rip = 11;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 0xA5u);

    memcpy(cpu.vector_registers[2], doubles, sizeof(doubles));
    cpu.rip = 15;
    assert(x86emu_step(&cpu) == X86EMU_OK);
        assert(cpu.registers[X86EMU_RAX] == 9u);
    
    memset(cpu.vector_registers[2], 0, sizeof(cpu.vector_registers[2]));

    for (unsigned i = 0; i < 32u; ++i) bytes[i] = (uint8_t)(i == 0u || i == 8u || i == 16u || i == 31u ? 0x80u : 0u);
    memcpy(cpu.vector_registers[2], bytes, sizeof(bytes));
    cpu.rip = 19;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_C(0x80010101));
}

static void test_legacy_compare_greater(void)
{
    uint8_t code[12] = {
        0x66, 0x0F, 0x64, 0xC2,
        0x66, 0x0F, 0x65, 0xC2,
        0x66, 0x0F, 0x66, 0xC2
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint8_t left_bytes[16] = { 0x80u, 0x7Fu };
    uint8_t right_bytes[16] = { 0x7Fu, 0x80u };
    uint16_t left_words[8] = { 0x8000u, 0x7FFFu };
    uint16_t right_words[8] = { 0x7FFFu, 0x8000u };
    uint32_t left_dwords[4] = { UINT32_C(0x80000000), UINT32_C(0x7FFFFFFF) };
    uint32_t right_dwords[4] = { UINT32_C(0x7FFFFFFF), UINT32_C(0x80000000) };
    x86emu_init(&cpu, memory, 0);

    memcpy(cpu.vector_registers[0], left_bytes, sizeof(left_bytes));
    memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.vector_registers[0][0] == 0u && cpu.vector_registers[0][1] == UINT8_MAX);

    memcpy(cpu.vector_registers[0], left_words, sizeof(left_words));
    memcpy(cpu.vector_registers[2], right_words, sizeof(right_words));
    cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    uint16_t word_result[8];
    memcpy(word_result, cpu.vector_registers[0], sizeof(word_result));
    assert(word_result[0] == 0u && word_result[1] == UINT16_MAX);

    memcpy(cpu.vector_registers[0], left_dwords, sizeof(left_dwords));
    memcpy(cpu.vector_registers[2], right_dwords, sizeof(right_dwords));
    cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    uint32_t dword_result[4];
    memcpy(dword_result, cpu.vector_registers[0], sizeof(dword_result));
    assert(dword_result[0] == 0u && dword_result[1] == UINT32_MAX);
}

static void test_scalar_float_arithmetic(void)
{
    uint8_t code[32] = { 0xF3,0x0F,0x58,0xCA, 0xF3,0x0F,0x5C,0xCA, 0xF2,0x0F,0x58,0xCA, 0xF2,0x0F,0x5C,0xCA,
                         0xC5,0xE2,0x58,0xCA, 0xC5,0xE2,0x5C,0xCA, 0xC5,0xE3,0x58,0xCA, 0xC5,0xE3,0x5C,0xCA };
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; uint32_t f[4] = { 2u, 0xAAAAAAAAu, 0xBBBBBBBBu, 0xCCCCCCCCu }; uint32_t g[4] = { 3u, 0, 0, 0 }; uint64_t d[2] = { 6u, UINT64_C(0xAAAAAAAAAAAAAAAA) }; uint64_t e[2] = { 2u, 0 }; x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[1], f, sizeof(f)); memcpy(cpu.vector_registers[2], g, sizeof(g)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(f,cpu.vector_registers[1],sizeof(f)); assert(f[0]==5u && f[1]==0xAAAAAAAAu);
    cpu.rip=4; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(f,cpu.vector_registers[1],sizeof(f)); assert(f[0]==2u && f[1]==0xAAAAAAAAu);
    memcpy(cpu.vector_registers[1],d,sizeof(d)); memcpy(cpu.vector_registers[2],e,sizeof(e)); cpu.rip=8; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(d,cpu.vector_registers[1],sizeof(d)); assert(d[0]==8u && d[1]==UINT64_C(0xAAAAAAAAAAAAAAAA));
    cpu.rip=13; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(d,cpu.vector_registers[1],sizeof(d)); assert(d[0]==6u);
    memcpy(cpu.vector_registers[3], f, sizeof(f)); memcpy(cpu.vector_registers[2], g, sizeof(g)); memset(cpu.vector_registers[1] + 16u, 0x5Au, 48u); cpu.rip=16; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(f,cpu.vector_registers[1],sizeof(f)); assert(f[0]==5u && f[1]==0xAAAAAAAAu); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
}

static void test_scalar_float_mul_div(void)
{
    uint8_t code[32] = { 0xF3,0x0F,0x59,0xCA, 0xF3,0x0F,0x5E,0xCA, 0xF2,0x0F,0x59,0xCA, 0xF2,0x0F,0x5E,0xCA, 0xC5,0xE2,0x59,0xCA, 0xC5,0xE2,0x5E,0xCA, 0xC5,0xE3,0x59,0xCA, 0xC5,0xE3,0x5E,0xCA };
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 };
    uint32_t f[4] = { UINT32_C(0x40000000), 0xAAAAAAAAu, 0xBBBBBBBBu, 0xCCCCCCCCu };
    uint32_t g[4] = { UINT32_C(0x40800000), 0, 0, 0 };
    uint64_t d[2] = { UINT64_C(0x3FF8000000000000), UINT64_C(0xAAAAAAAAAAAAAAAA) };
    uint64_t e[2] = { UINT64_C(0x4000000000000000), 0 }; x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[1], f, sizeof(f)); memcpy(cpu.vector_registers[2], g, sizeof(g)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(f,cpu.vector_registers[1],sizeof(f)); assert(f[0]==UINT32_C(0x41000000) && f[1]==0xAAAAAAAAu);
    cpu.rip=4; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(f,cpu.vector_registers[1],sizeof(f)); assert(f[0]==UINT32_C(0x40000000) && f[1]==0xAAAAAAAAu);
    memcpy(cpu.vector_registers[1],d,sizeof(d)); memcpy(cpu.vector_registers[2],e,sizeof(e)); cpu.rip=8; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(d,cpu.vector_registers[1],sizeof(d)); assert(d[0]==UINT64_C(0x4008000000000000) && d[1]==UINT64_C(0xAAAAAAAAAAAAAAAA));
    cpu.rip=12; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(d,cpu.vector_registers[1],sizeof(d)); assert(d[0]==UINT64_C(0x3FF8000000000000));
    memcpy(cpu.vector_registers[3], f, sizeof(f)); memcpy(cpu.vector_registers[2], g, sizeof(g)); memset(cpu.vector_registers[1] + 16u, 0xA5, 48u); cpu.rip=16; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(f,cpu.vector_registers[1],sizeof(f)); assert(f[0]==UINT32_C(0x41000000) && f[1]==0xAAAAAAAAu); for (unsigned i=16u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
    cpu.rip=20; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(f,cpu.vector_registers[1],sizeof(f)); assert(f[0]==UINT32_C(0x3F000000));
    memcpy(cpu.vector_registers[3], d, sizeof(d)); memcpy(cpu.vector_registers[2], e, sizeof(e)); cpu.rip=24; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(d,cpu.vector_registers[1],sizeof(d)); assert(d[0]==UINT64_C(0x4008000000000000));
    cpu.rip=28; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(d,cpu.vector_registers[1],sizeof(d)); assert(d[0]==UINT64_C(0x3FE8000000000000));
}

static void test_scalar_float_memory(void)
{
    uint8_t code[64] = { 0xF3,0x0F,0x59,0x08, 0xF2,0x0F,0x5E,0x08, 0xC5,0xE2,0x59,0x08, 0xC5,0xE3,0x5E,0x08 };
    uint32_t single = UINT32_C(0x40800000); uint64_t dbl = UINT64_C(0x4000000000000000);
    memcpy(code + 32, &single, sizeof(single)); memcpy(code + 40, &dbl, sizeof(dbl));
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[0] = 32;
    uint32_t fs = UINT32_C(0x40000000); memcpy(cpu.vector_registers[1], &fs, sizeof(fs)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(&fs,cpu.vector_registers[1],sizeof(fs)); assert(fs==UINT32_C(0x41000000));
    cpu.registers[0] = 40; uint64_t ds = UINT64_C(0x4008000000000000); memcpy(cpu.vector_registers[1], &ds, sizeof(ds)); cpu.rip=4; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(&ds,cpu.vector_registers[1],sizeof(ds)); assert(ds==UINT64_C(0x3FF8000000000000));
}

static void test_scalar_moves(void)
{
    uint8_t code[32] = { 0xF3,0x0F,0x10,0xCA, 0xF3,0x0F,0x10,0x08, 0xF3,0x0F,0x11,0x08, 0xC5,0xE2,0x10,0xCA, 0xF2,0x0F,0x10,0xCA };
    uint32_t source[4] = { UINT32_C(0x3F800000), 0x11111111u, 0x22222222u, 0x33333333u };
    uint32_t first[4] = { UINT32_C(0x40000000), 0xAAAAAAAAu, 0xBBBBBBBBu, 0xCCCCCCCCu };
    uint32_t third[4] = { UINT32_C(0x40400000), 0xDDDDDDDDu, 0xEEEEEEEEu, 0xFFFFFFFFu };
    memcpy(code + 24, source, sizeof(uint32_t));
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], first, sizeof(first)); memcpy(cpu.vector_registers[2], source, sizeof(source));
    assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(first,cpu.vector_registers[1],sizeof(first)); assert(first[0]==UINT32_C(0x3F800000) && first[1]==0xAAAAAAAAu);
    cpu.registers[0]=24; cpu.rip=4; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(first,cpu.vector_registers[1],sizeof(first)); assert(first[0]==UINT32_C(0x3F800000) && first[1]==0u && first[2]==0u && first[3]==0u);
    cpu.rip=8; assert(x86emu_step(&cpu)==X86EMU_OK); uint32_t stored=0; memcpy(&stored, code+24, sizeof(stored)); assert(stored==UINT32_C(0x3F800000));
    memcpy(cpu.vector_registers[1], first, sizeof(first)); memcpy(cpu.vector_registers[2], source, sizeof(source)); memcpy(cpu.vector_registers[3], third, sizeof(third)); memset(cpu.vector_registers[1]+16u,0xA5,48u); cpu.rip=12; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(first,cpu.vector_registers[1],sizeof(first)); assert(first[0]==UINT32_C(0x3F800000) && first[1]==0xDDDDDDDDu); for (unsigned i=16u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
    cpu.rip=16; assert(x86emu_step(&cpu)==X86EMU_OK);
}

static void test_scalar_float_minmax(void)
{
    uint8_t code[20] = { 0xF3,0x0F,0x5D,0xCA, 0xF3,0x0F,0x5F,0xCA, 0xF2,0x0F,0x5D,0xCA, 0xF2,0x0F,0x5F,0xCA, 0xC5,0xE2,0x5D,0xCA };
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0);
    uint32_t a[4] = { 0u, 0xAAAAAAAAu, 0u, 0u }; uint32_t b[4] = { UINT32_C(0x80000000), 0u, 0u, 0u }; memcpy(cpu.vector_registers[1], a, sizeof(a)); memcpy(cpu.vector_registers[2], b, sizeof(b)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(a,cpu.vector_registers[1],sizeof(a)); assert(a[0]==UINT32_C(0x80000000) && a[1]==0xAAAAAAAAu);
    a[0]=UINT32_C(0x3F800000); b[0]=UINT32_C(0x7FC00001); memcpy(cpu.vector_registers[1],a,sizeof(a)); memcpy(cpu.vector_registers[2],b,sizeof(b)); cpu.rip=4; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(a,cpu.vector_registers[1],sizeof(a)); assert(a[0]==UINT32_C(0x7FC00001));
    uint64_t da[2] = { UINT64_C(0x7FF8000000000001), 0 }; uint64_t db[2] = { UINT64_C(0x4000000000000000), 0 }; memcpy(cpu.vector_registers[1],da,sizeof(da)); memcpy(cpu.vector_registers[2],db,sizeof(db)); cpu.rip=8; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(da,cpu.vector_registers[1],sizeof(da)); assert(da[0]==UINT64_C(0x4000000000000000));
    da[0]=UINT64_C(0xC000000000000000); db[0]=UINT64_C(0x7FF8000000000001); memcpy(cpu.vector_registers[1],da,sizeof(da)); memcpy(cpu.vector_registers[2],db,sizeof(db)); cpu.rip=12; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(da,cpu.vector_registers[1],sizeof(da)); assert(da[0]==UINT64_C(0x7FF8000000000001));
    uint32_t first[4] = { UINT32_C(0x40400000), 0xDDDDDDDDu, 0xEEEEEEEEu, 0xFFFFFFFFu }; b[0] = UINT32_C(0x40000000); memcpy(cpu.vector_registers[3],first,sizeof(first)); memcpy(cpu.vector_registers[2],b,sizeof(b)); cpu.rip=16; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(a,cpu.vector_registers[1],sizeof(a)); assert(a[0]==UINT32_C(0x40000000)); assert(cpu.vector_registers[1][4]==0xDDu);
}

static void test_packed_float_compare(void)
{
    uint8_t code[19] = {
        0x0F, 0xC2, 0xCA, 0x00,
        0x66, 0x0F, 0xC2, 0xCA, 0x03,
        0xC5, 0xE4, 0xC2, 0xCA, 0x0E,
        0xC5, 0xE5, 0xC2, 0xCA, 0x03
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint32_t single_left[8] = { UINT32_C(0x3F800000), UINT32_C(0x7FC00001), 3u, 4u, 0u, 0u, 0u, 0u };
    uint32_t single_right[8] = { UINT32_C(0x3F800000), 0u, 3u, 5u, 0u, 0u, 0u, 0u };
    uint32_t single_gt_left[8] = { 3u, UINT32_C(0x7FC00001), 1u, 5u, 0u, 0u, 0u, 0u };
    uint32_t single_gt_right[8] = { 2u, 0u, 1u, 4u, 0u, 0u, 0u, 0u };
    uint64_t double_left[4] = { UINT64_C(0x7FF8000000000001), 2u, 0u, 0u };
    uint64_t double_right[4] = { 1u, UINT64_C(0x7FF8000000000001), 0u, 0u };
    uint32_t result_single[8];
    uint64_t result_double[4];
    x86emu_init(&cpu, memory, 0);

    memcpy(cpu.vector_registers[1], single_left, sizeof(single_left)); memcpy(cpu.vector_registers[2], single_right, sizeof(single_right));
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_single, cpu.vector_registers[1], sizeof(result_single));
    assert(result_single[0] == UINT32_MAX && result_single[1] == 0u && result_single[2] == UINT32_MAX && result_single[3] == 0u);

    memcpy(cpu.vector_registers[1], double_left, sizeof(double_left)); memcpy(cpu.vector_registers[2], double_right, sizeof(double_right)); cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_double, cpu.vector_registers[1], sizeof(result_double));
    assert(result_double[0] == UINT64_MAX && result_double[1] == UINT64_MAX);

    memcpy(cpu.vector_registers[3], single_gt_left, sizeof(single_gt_left)); memcpy(cpu.vector_registers[2], single_gt_right, sizeof(single_gt_right)); cpu.rip = 9;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_single, cpu.vector_registers[1], sizeof(result_single));
    assert(result_single[0] == UINT32_MAX && result_single[1] == 0u && result_single[2] == 0u && result_single[3] == UINT32_MAX);

    memcpy(cpu.vector_registers[3], double_left, sizeof(double_left)); memcpy(cpu.vector_registers[2], double_right, sizeof(double_right)); cpu.rip = 14;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result_double, cpu.vector_registers[1], sizeof(result_double));
    assert(result_double[0] == UINT64_MAX && result_double[1] == UINT64_MAX);
}

static void test_packed_float_minmax(void)
{
    uint8_t code[30] = {
        0x0F, 0x5D, 0xCA, 0x0F, 0x5F, 0xCA,
        0x66, 0x0F, 0x5D, 0xCA, 0x66, 0x0F, 0x5F, 0xCA,
        0xC5, 0xE4, 0x5D, 0xCA, 0xC5, 0xE4, 0x5F, 0xCA,
        0xC5, 0xE5, 0x5D, 0xCA, 0xC5, 0xE5, 0x5F, 0xCA
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint32_t single_left[8] = { UINT32_C(0x7FC00001), UINT32_C(0x80000000), 3u, 0u,
                                 0u, 0u, 0u, 0u };
    uint32_t single_right[8] = { UINT32_C(0x3F800000), 0u, 2u, 0u,
                                  0u, 0u, 0u, 0u };
    uint64_t double_left[4] = { UINT64_C(0x7FF8000000000001), UINT64_C(0x8000000000000000), 3u, 0u };
    uint64_t double_right[4] = { UINT64_C(0x3FF0000000000000), 0u, 2u, 0u };
    uint32_t result_single[8];
    uint64_t result_double[4];
    x86emu_init(&cpu, memory, 0);

    memcpy(cpu.vector_registers[1], single_left, sizeof(single_left));
    memcpy(cpu.vector_registers[2], single_right, sizeof(single_right));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_single, cpu.vector_registers[1], sizeof(result_single));
    assert(result_single[0] == single_right[0] && result_single[1] == single_right[1]);

    memcpy(cpu.vector_registers[1], single_left, sizeof(single_left));
    memcpy(cpu.vector_registers[2], single_right, sizeof(single_right));
    cpu.rip = 3;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_single, cpu.vector_registers[1], sizeof(result_single));
    assert(result_single[0] == single_right[0] && result_single[1] == single_right[1]);

    memcpy(cpu.vector_registers[1], double_left, sizeof(double_left));
    memcpy(cpu.vector_registers[2], double_right, sizeof(double_right));
    cpu.rip = 6;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_double, cpu.vector_registers[1], sizeof(result_double));
    assert(result_double[0] == double_right[0] && result_double[1] == double_right[1]);

    memcpy(cpu.vector_registers[1], double_left, sizeof(double_left));
    memcpy(cpu.vector_registers[2], double_right, sizeof(double_right));
    cpu.rip = 10;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_double, cpu.vector_registers[1], sizeof(result_double));
    assert(result_double[0] == double_right[0] && result_double[1] == double_right[1]);

    memcpy(cpu.vector_registers[3], single_left, sizeof(single_left));
    memcpy(cpu.vector_registers[2], single_right, sizeof(single_right));
    cpu.rip = 14;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_single, cpu.vector_registers[1], sizeof(result_single));
    assert(result_single[0] == single_right[0] && result_single[1] == single_right[1]);

    memcpy(cpu.vector_registers[3], single_left, sizeof(single_left));
    memcpy(cpu.vector_registers[2], single_right, sizeof(single_right));
    cpu.rip = 18;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_single, cpu.vector_registers[1], sizeof(result_single));
    assert(result_single[0] == single_right[0] && result_single[1] == single_right[1]);

    memcpy(cpu.vector_registers[3], double_left, sizeof(double_left));
    memcpy(cpu.vector_registers[2], double_right, sizeof(double_right));
    cpu.rip = 22;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_double, cpu.vector_registers[1], sizeof(result_double));
    assert(result_double[0] == double_right[0] && result_double[1] == double_right[1]);

    memcpy(cpu.vector_registers[3], double_left, sizeof(double_left));
    memcpy(cpu.vector_registers[2], double_right, sizeof(double_right));
    cpu.rip = 26;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(result_double, cpu.vector_registers[1], sizeof(result_double));
    assert(result_double[0] == double_right[0] && result_double[1] == double_right[1]);
}

static void test_packed_float_arithmetic(void)
{
    uint8_t code[53] = {
        0x0F, 0x5C, 0xCA, 0x0F, 0x59, 0xCA, 0x0F, 0x5E, 0xCA,
        0x66, 0x0F, 0x58, 0xCA, 0x66, 0x0F, 0x5C, 0xCA,
        0x66, 0x0F, 0x59, 0xCA, 0x66, 0x0F, 0x5E, 0xCA,
        0xC5, 0xE4, 0x5C, 0xCA, 0xC5, 0xE4, 0x59, 0xCA,
        0xC5, 0xE4, 0x5E, 0xCA, 0xC5, 0xE5, 0x58, 0xCA,
        0xC5, 0xE5, 0x5C, 0xCA, 0xC5, 0xE5, 0x59, 0xCA,
        0xC5, 0xE5, 0x5E, 0xCA
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    float left_single[8] = { 8.0f, 6.0f, 4.0f, 2.0f, 16.0f, 12.0f, 10.0f, 6.0f };
    float right_single[8] = { 2.0f, 3.0f, 2.0f, 1.0f, 4.0f, 3.0f, 2.0f, 3.0f };
    double left_double[4] = { 8.0, 6.0, 16.0, 12.0 };
    double right_double[4] = { 2.0, 3.0, 2.0, 3.0 };
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[1], left_single, 16); memcpy(cpu.vector_registers[2], right_single, 16);
    assert(x86emu_step(&cpu) == X86EMU_OK); { float out[4]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 6.0f && out[1] == 3.0f && out[2] == 2.0f && out[3] == 1.0f); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { float out[4]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 12.0f && out[1] == 9.0f && out[2] == 4.0f && out[3] == 1.0f); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { float out[4]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 6.0f && out[1] == 3.0f && out[2] == 2.0f && out[3] == 1.0f); }
    memcpy(cpu.vector_registers[1], left_double, 16); memcpy(cpu.vector_registers[2], right_double, 16); cpu.rip = 9;
    assert(x86emu_step(&cpu) == X86EMU_OK); { double out[2]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 10.0 && out[1] == 9.0); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { double out[2]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 8.0 && out[1] == 6.0); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { double out[2]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 16.0 && out[1] == 18.0); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { double out[2]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 8.0 && out[1] == 6.0); }
    memcpy(cpu.vector_registers[3], left_single, sizeof(left_single)); memcpy(cpu.vector_registers[2], right_single, sizeof(right_single)); cpu.rip = 25;
    assert(x86emu_step(&cpu) == X86EMU_OK); { float out[8]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 6.0f && out[7] == 3.0f); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { float out[8]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 16.0f && out[7] == 18.0f); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { float out[8]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 4.0f && out[7] == 2.0f); }
    memcpy(cpu.vector_registers[3], left_double, sizeof(left_double)); memcpy(cpu.vector_registers[2], right_double, sizeof(right_double)); cpu.rip = 37;
    assert(x86emu_step(&cpu) == X86EMU_OK); { double out[4]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 10.0 && out[1] == 9.0 && out[2] == 18.0 && out[3] == 15.0); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { double out[4]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 6.0 && out[1] == 3.0 && out[2] == 14.0 && out[3] == 9.0); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { double out[4]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 16.0 && out[1] == 18.0 && out[2] == 32.0 && out[3] == 36.0); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { double out[4]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 4.0 && out[1] == 2.0 && out[2] == 8.0 && out[3] == 4.0); }
}

static void test_packed_shuffles(void)
{
    uint8_t code[30] = {
        0x66, 0x0F, 0x70, 0xCA, 0x1B,
        0xF2, 0x0F, 0x70, 0xCA, 0x1B,
        0xF3, 0x0F, 0x70, 0xCA, 0x1B,
        0xC5, 0xFD, 0x70, 0xCA, 0x1B,
        0xC5, 0xF6, 0x70, 0xCA, 0x1B,
        0xC5, 0xF7, 0x70, 0xCA, 0x1B
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint32_t dwords[4] = { 1u, 2u, 3u, 4u };
    uint16_t words[8] = { 1u, 2u, 3u, 4u, 5u, 6u, 7u, 8u };
    uint32_t ymm_dwords[8] = { 1u, 2u, 3u, 4u, 11u, 12u, 13u, 14u };
    uint16_t ymm_words[16] = { 1u, 2u, 3u, 4u, 5u, 6u, 7u, 8u,
                               11u, 12u, 13u, 14u, 15u, 16u, 17u, 18u };
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[2], dwords, sizeof(dwords));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(dwords, cpu.vector_registers[1], sizeof(dwords));
    assert(dwords[0] == 4u && dwords[1] == 3u && dwords[2] == 2u && dwords[3] == 1u);
    memcpy(cpu.vector_registers[2], words, sizeof(words));
    cpu.rip = 5;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(words, cpu.vector_registers[1], sizeof(words));
    assert(words[0] == 4u && words[1] == 3u && words[2] == 2u && words[3] == 1u &&
           words[4] == 5u && words[5] == 6u && words[6] == 7u && words[7] == 8u);
    memcpy(cpu.vector_registers[2], words, sizeof(words));
    cpu.rip = 10;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(words, cpu.vector_registers[1], sizeof(words));
    assert(words[0] == 4u && words[1] == 3u && words[2] == 2u && words[3] == 1u &&
           words[4] == 8u && words[5] == 7u && words[6] == 6u && words[7] == 5u);

    memcpy(cpu.vector_registers[2], ymm_dwords, sizeof(ymm_dwords));
    cpu.rip = 15;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(ymm_dwords, cpu.vector_registers[1], sizeof(ymm_dwords));
    assert(ymm_dwords[0] == 4u && ymm_dwords[1] == 3u && ymm_dwords[2] == 2u && ymm_dwords[3] == 1u &&
           ymm_dwords[4] == 14u && ymm_dwords[5] == 13u && ymm_dwords[6] == 12u && ymm_dwords[7] == 11u);
    memcpy(cpu.vector_registers[2], ymm_words, sizeof(ymm_words));
    cpu.rip = 20;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(ymm_words, cpu.vector_registers[1], sizeof(ymm_words));
    assert(ymm_words[0] == 4u && ymm_words[1] == 3u && ymm_words[2] == 2u && ymm_words[3] == 1u &&
           ymm_words[4] == 5u && ymm_words[5] == 6u && ymm_words[6] == 7u && ymm_words[7] == 8u &&
           ymm_words[8] == 14u && ymm_words[9] == 13u && ymm_words[10] == 12u && ymm_words[11] == 11u &&
           ymm_words[12] == 15u && ymm_words[13] == 16u && ymm_words[14] == 17u && ymm_words[15] == 18u);
    memcpy(cpu.vector_registers[2], ymm_words, sizeof(ymm_words));
    cpu.rip = 25;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(ymm_words, cpu.vector_registers[1], sizeof(ymm_words));
    assert(ymm_words[0] == 4u && ymm_words[1] == 3u && ymm_words[2] == 2u && ymm_words[3] == 1u &&
           ymm_words[4] == 8u && ymm_words[5] == 7u && ymm_words[6] == 6u && ymm_words[7] == 5u &&
           ymm_words[8] == 14u && ymm_words[9] == 13u && ymm_words[10] == 12u && ymm_words[11] == 11u &&
           ymm_words[12] == 18u && ymm_words[13] == 17u && ymm_words[14] == 16u && ymm_words[15] == 15u);
}

static void test_pshufb(void)
{
    uint8_t code[10] = {
        0x66, 0x0F, 0x38, 0x00, 0xCA,
        0xC4, 0xE2, 0x05, 0x00, 0xD2
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint8_t source[32];
    uint8_t control[32];
    x86emu_init(&cpu, memory, 0);
    for (unsigned i = 0; i < sizeof(source); ++i) {
        source[i] = (uint8_t)i;
        control[i] = (uint8_t)(15u - (i & 15u));
    }
    control[3] = 0x80u;
    memcpy(cpu.vector_registers[1], source, 16);
    memcpy(cpu.vector_registers[2], control, 16);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    for (unsigned i = 0; i < 16u; ++i) assert(cpu.vector_registers[1][i] == (i == 3u ? 0u : (uint8_t)(15u - i)));

    for (unsigned i = 0; i < sizeof(source); ++i) {
        source[i] = (uint8_t)(0x80u + i);
        control[i] = (uint8_t)(i & 15u);
    }
    control[16] = 0x80u;
    memcpy(cpu.vector_registers[15], source, sizeof(source));
    memcpy(cpu.vector_registers[2], control, sizeof(control));
    cpu.rip = 5;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    for (unsigned i = 0; i < 16u; ++i) assert(cpu.vector_registers[2][i] == (uint8_t)(0x80u + i));
    assert(cpu.vector_registers[2][16] == 0u);
    for (unsigned i = 17; i < 32u; ++i) assert(cpu.vector_registers[2][i] == (uint8_t)(0x80u + i));
}

static void test_v_extended_minmax(void)
{
    uint8_t code[40] = {
        0xC4, 0xE2, 0x65, 0x38, 0xCA, 0xC4, 0xE2, 0x65, 0x3C, 0xCA,
        0xC4, 0xE2, 0x65, 0x3A, 0xCA, 0xC4, 0xE2, 0x65, 0x3E, 0xCA,
        0xC4, 0xE2, 0x65, 0x39, 0xCA, 0xC4, 0xE2, 0x65, 0x3D, 0xCA,
        0xC4, 0xE2, 0x65, 0x3B, 0xCA, 0xC4, 0xE2, 0x65, 0x3F, 0xCA
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint8_t left_bytes[32] = { 0x80u, 0x7Fu };
    uint8_t right_bytes[32] = { 0x7Fu, 0x80u };
    uint16_t left_words[16] = { 0u, UINT16_MAX };
    uint16_t right_words[16] = { 1u, 1u };
    uint32_t left_dwords[8] = { UINT32_C(0x80000000), UINT32_C(0x7FFFFFFF) };
    uint32_t right_dwords[8] = { UINT32_C(0x7FFFFFFF), UINT32_C(0x80000000) };
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[3], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes));
    assert(x86emu_step(&cpu) == X86EMU_OK); assert(cpu.vector_registers[1][0] == 0x80u);
    assert(x86emu_step(&cpu) == X86EMU_OK); assert(cpu.vector_registers[1][0] == 0x7Fu);
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 10;
    assert(x86emu_step(&cpu) == X86EMU_OK); { uint16_t out[16]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 0u && out[1] == 1u); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { uint16_t out[16]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 1u && out[1] == UINT16_MAX); }
    memcpy(cpu.vector_registers[3], left_dwords, sizeof(left_dwords)); memcpy(cpu.vector_registers[2], right_dwords, sizeof(right_dwords)); cpu.rip = 20;
    assert(x86emu_step(&cpu) == X86EMU_OK); { uint32_t out[8]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == UINT32_C(0x80000000) && out[1] == UINT32_C(0x80000000)); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { uint32_t out[8]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == UINT32_C(0x7FFFFFFF) && out[1] == UINT32_C(0x7FFFFFFF)); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { uint32_t out[8]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == UINT32_C(0x7FFFFFFF) && out[1] == UINT32_C(0x7FFFFFFF)); }
    assert(x86emu_step(&cpu) == X86EMU_OK); { uint32_t out[8]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == UINT32_C(0x80000000) && out[1] == UINT32_C(0x80000000)); }
}

static void test_v_compare_greater(void)
{
    uint8_t code[17] = {
        0xC5, 0xE5, 0x64, 0xCA, 0xC5, 0xE5, 0x65, 0xCA,
        0xC5, 0xE5, 0x66, 0xCA, 0xC4, 0xE2, 0xE5, 0x37, 0xCA
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint8_t left[32] = { 0x80u, 0x7Fu };
    uint8_t right[32] = { 0x7Fu, 0x80u };
    uint16_t left_words[16] = { 0x8000u, 0x7FFFu };
    uint16_t right_words[16] = { 0x7FFFu, 0x8000u };
    uint32_t left_dwords[8] = { UINT32_C(0x80000000), UINT32_C(0x7FFFFFFF) };
    uint32_t right_dwords[8] = { UINT32_C(0x7FFFFFFF), UINT32_C(0x80000000) };
    uint64_t left_qwords[4] = { UINT64_C(0x8000000000000000), UINT64_C(0x7FFFFFFFFFFFFFFF) };
    uint64_t right_qwords[4] = { UINT64_C(0x7FFFFFFFFFFFFFFF), UINT64_C(0x8000000000000000) };
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[3], left, sizeof(left)); memcpy(cpu.vector_registers[2], right, sizeof(right));
    assert(x86emu_step(&cpu) == X86EMU_OK); assert(cpu.vector_registers[1][0] == 0u && cpu.vector_registers[1][1] == 0xFFu);
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK); { uint16_t out[16]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 0u && out[1] == UINT16_MAX); }
    memcpy(cpu.vector_registers[3], left_dwords, sizeof(left_dwords)); memcpy(cpu.vector_registers[2], right_dwords, sizeof(right_dwords)); cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK); { uint32_t out[8]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 0u && out[1] == UINT32_MAX); }
    memcpy(cpu.vector_registers[3], left_qwords, sizeof(left_qwords)); memcpy(cpu.vector_registers[2], right_qwords, sizeof(right_qwords)); cpu.rip = 12;
    assert(x86emu_step(&cpu) == X86EMU_OK); { uint64_t out[4]; memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 0u && out[1] == UINT64_MAX); }
}

static void test_v_unpack_pack(void)
{
    uint8_t code[36] = {
        0xC5, 0xE5, 0x60, 0xCA, 0xC5, 0xE5, 0x61, 0xCA,
        0xC5, 0xE5, 0x62, 0xCA, 0xC5, 0xE5, 0x63, 0xCA,
        0xC5, 0xE5, 0x67, 0xCA, 0xC5, 0xE5, 0x68, 0xCA,
        0xC5, 0xE5, 0x69, 0xCA, 0xC5, 0xE5, 0x6A, 0xCA,
        0xC5, 0xE5, 0x6B, 0xCA
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    uint8_t left_bytes[32];
    uint8_t right_bytes[32];
    uint8_t result[32];
    uint16_t left_words[16] = { 1u, 2u, 3u, 4u, 5u, 6u, 7u, 8u, 11u, 12u, 13u, 14u, 15u, 16u, 17u, 18u };
    uint16_t right_words[16] = { 101u, 102u, 103u, 104u, 105u, 106u, 107u, 108u, 111u, 112u, 113u, 114u, 115u, 116u, 117u, 118u };
    uint32_t left_dwords[8] = { 1u, 2u, 3u, 4u, 11u, 12u, 13u, 14u };
    uint32_t right_dwords[8] = { 101u, 102u, 103u, 104u, 111u, 112u, 113u, 114u };
    int16_t pack_words[16] = { -200, -1, 0, 127, 128, 300, 32767, -32768, -200, -1, 0, 127, 128, 300, 32767, -32768 };
    int16_t pack_words_right[16] = { 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8 };
    int32_t pack_dwords[8] = { INT32_C(-40000), INT32_C(-32768), INT32_C(32767), INT32_C(40000), INT32_C(-40000), INT32_C(-32768), INT32_C(32767), INT32_C(40000) };
    int32_t pack_dwords_right[8] = { -1, 0, 1, 2, -1, 0, 1, 2 };
    x86emu_init(&cpu, memory, 0);
    for (unsigned i = 0; i < 32u; ++i) { left_bytes[i] = (uint8_t)(i + 1u); right_bytes[i] = (uint8_t)(i + 101u); }
    memcpy(cpu.vector_registers[3], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes));
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); assert(result[0] == 1u && result[1] == 101u && result[14] == 8u && result[15] == 108u && result[16] == 17u && result[17] == 117u && result[30] == 24u && result[31] == 124u);
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); { uint16_t out[16]; memcpy(out, result, sizeof(out)); assert(out[0] == 1u && out[1] == 101u && out[6] == 4u && out[7] == 104u && out[8] == 11u && out[9] == 111u); }
    memcpy(cpu.vector_registers[3], left_dwords, sizeof(left_dwords)); memcpy(cpu.vector_registers[2], right_dwords, sizeof(right_dwords)); cpu.rip = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); { uint32_t out[8]; memcpy(out, result, sizeof(out)); assert(out[0] == 1u && out[1] == 101u && out[2] == 2u && out[3] == 102u && out[4] == 11u && out[5] == 111u); }
    memcpy(cpu.vector_registers[3], pack_words, sizeof(pack_words)); memcpy(cpu.vector_registers[2], pack_words_right, sizeof(pack_words_right)); cpu.rip = 12;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); assert((int8_t)result[0] == -128 && (int8_t)result[4] == 127 && (int8_t)result[7] == -128 && (int8_t)result[16] == -128);
    memcpy(cpu.vector_registers[3], pack_words, sizeof(pack_words)); memcpy(cpu.vector_registers[2], pack_words_right, sizeof(pack_words_right)); cpu.rip = 16;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); assert(result[0] == 0u && result[4] == 128u && result[5] == 255u && result[16] == 0u);
    memcpy(cpu.vector_registers[3], left_bytes, sizeof(left_bytes)); memcpy(cpu.vector_registers[2], right_bytes, sizeof(right_bytes)); cpu.rip = 20;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); assert(result[0] == 9u && result[1] == 109u && result[14] == 16u && result[15] == 116u && result[16] == 25u);
    memcpy(cpu.vector_registers[3], left_words, sizeof(left_words)); memcpy(cpu.vector_registers[2], right_words, sizeof(right_words)); cpu.rip = 24;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); { uint16_t out[16]; memcpy(out, result, sizeof(out)); assert(out[0] == 5u && out[1] == 105u && out[6] == 8u && out[7] == 108u && out[8] == 15u); }
    memcpy(cpu.vector_registers[3], left_dwords, sizeof(left_dwords)); memcpy(cpu.vector_registers[2], right_dwords, sizeof(right_dwords)); cpu.rip = 28;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); { uint32_t out[8]; memcpy(out, result, sizeof(out)); assert(out[0] == 3u && out[1] == 103u && out[2] == 4u && out[3] == 104u && out[4] == 13u); }
    memcpy(cpu.vector_registers[3], pack_dwords, sizeof(pack_dwords)); memcpy(cpu.vector_registers[2], pack_dwords_right, sizeof(pack_dwords_right)); cpu.rip = 32;
    assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(result, cpu.vector_registers[1], sizeof(result)); { int16_t out[16]; memcpy(out, result, sizeof(out)); assert(out[0] == -32768 && out[1] == -32768 && out[2] == 32767 && out[3] == 32767 && out[8] == -32768); }
}

static void test_movbe(void)
{
    uint8_t memory_bytes[128] = {
        0x49, 0x0F, 0x38, 0xF0, 0x00,
        0x49, 0x0F, 0x38, 0xF1, 0x00
    };
    x86emu_cpu cpu;
    x86emu_memory memory = { memory_bytes, sizeof(memory_bytes), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_R8] = 64;
    for (unsigned i = 0; i < 8; ++i) memory_bytes[64 + i] = (uint8_t)(0x11u * (i + 1u));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == UINT64_C(0x1122334455667788));
    cpu.rip = 5;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memory_bytes[64] == 0x11 && memory_bytes[65] == 0x22 &&
           memory_bytes[66] == 0x33 && memory_bytes[67] == 0x44 &&
           memory_bytes[68] == 0x55 && memory_bytes[69] == 0x66 &&
           memory_bytes[70] == 0x77 && memory_bytes[71] == 0x88);
}

static void test_bit_instructions(void)
{
    static const uint8_t bsf_code[] = { 0x48, 0x0F, 0xBC, 0xC1 };
    static const uint8_t bsr_code[] = { 0x48, 0x0F, 0xBD, 0xC1 };
    static const uint8_t bt_code[] = { 0x48, 0x0F, 0xA3, 0xC8 };
    static const uint8_t bts_code[] = { 0x48, 0x0F, 0xBA, 0xE8, 0x03 };
    static const uint8_t btr_code[] = { 0x48, 0x0F, 0xBA, 0xF0, 0x03 };
    static const uint8_t btc_code[] = { 0x48, 0x0F, 0xBA, 0xF8, 0x03 };
    x86emu_cpu cpu;
    x86emu_memory memory;
    uint8_t buffer[32] = { 0 };

    memory = (x86emu_memory){ (uint8_t *)bsf_code, sizeof(bsf_code), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RCX] = 0x1008;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 3 && (cpu.rflags & X86EMU_FLAG_ZF) == 0);

    memory = (x86emu_memory){ (uint8_t *)bsr_code, sizeof(bsr_code), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RCX] = 0x1008;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 12 && (cpu.rflags & X86EMU_FLAG_ZF) == 0);

    memory = (x86emu_memory){ (uint8_t *)bt_code, sizeof(bt_code), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 8;
    cpu.registers[X86EMU_RCX] = 3;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert((cpu.rflags & X86EMU_FLAG_CF) != 0 && cpu.registers[X86EMU_RAX] == 8);

    memcpy(buffer, bts_code, sizeof(bts_code));
    memory = (x86emu_memory){ buffer, sizeof(buffer), 0 };
    x86emu_init(&cpu, memory, 0);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 8 && (cpu.rflags & X86EMU_FLAG_CF) == 0);

    memcpy(buffer, btr_code, sizeof(btr_code));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 8;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 0 && (cpu.rflags & X86EMU_FLAG_CF) != 0);

    memcpy(buffer, btc_code, sizeof(btc_code));
    x86emu_init(&cpu, memory, 0);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 8 && (cpu.rflags & X86EMU_FLAG_CF) == 0);
}

static void test_cmpxchg16b(void)
{
    uint8_t memory_bytes[128] = { 0x49, 0x0F, 0xC7, 0x08 };
    x86emu_cpu cpu;
    x86emu_memory memory = { memory_bytes, sizeof(memory_bytes), 0 };
    uint64_t low = UINT64_C(0x1122334455667788);
    uint64_t high = UINT64_C(0x99AABBCCDDEEFF00);
    memcpy(memory_bytes + 64, &low, sizeof(low));
    memcpy(memory_bytes + 72, &high, sizeof(high));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_R8] = 64;
    cpu.registers[X86EMU_RAX] = low;
    cpu.registers[X86EMU_RDX] = high;
    cpu.registers[X86EMU_RBX] = UINT64_C(0x0123456789ABCDEF);
    cpu.registers[X86EMU_RCX] = UINT64_C(0xFEDCBA9876543210);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    memcpy(&low, memory_bytes + 64, sizeof(low));
    memcpy(&high, memory_bytes + 72, sizeof(high));
    assert(low == UINT64_C(0x0123456789ABCDEF));
    assert(high == UINT64_C(0xFEDCBA9876543210));
    assert((cpu.rflags & X86EMU_FLAG_ZF) != 0);

    cpu.rip = 0;
    cpu.registers[X86EMU_RAX] = 0;
    cpu.registers[X86EMU_RDX] = 0;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == low && cpu.registers[X86EMU_RDX] == high);
    assert((cpu.rflags & X86EMU_FLAG_ZF) == 0);
}

static void test_cmpxchg8b(void)
{
    uint8_t memory_bytes[128] = { 0x41, 0x0F, 0xC7, 0x08 };
    x86emu_cpu cpu;
    x86emu_memory memory = { memory_bytes, sizeof(memory_bytes), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 0x55667788u;
    cpu.registers[X86EMU_RDX] = 0x11223344u;
    cpu.registers[X86EMU_RBX] = 0xDDEEFF00u;
    cpu.registers[X86EMU_RCX] = 0x99AABBCCu;
    cpu.registers[X86EMU_R8] = 64;
    memory_bytes[64] = 0x88;
    memory_bytes[65] = 0x77;
    memory_bytes[66] = 0x66;
    memory_bytes[67] = 0x55;
    memory_bytes[68] = 0x44;
    memory_bytes[69] = 0x33;
    memory_bytes[70] = 0x22;
    memory_bytes[71] = 0x11;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 0x55667788u);
    assert(cpu.registers[X86EMU_RDX] == 0x11223344u);
    assert((cpu.rflags & X86EMU_FLAG_ZF) != 0);
    assert(memory_bytes[64] == 0x00 && memory_bytes[65] == 0xFF &&
           memory_bytes[66] == 0xEE && memory_bytes[67] == 0xDD);
    assert(memory_bytes[68] == 0xCC && memory_bytes[69] == 0xBB &&
           memory_bytes[70] == 0xAA && memory_bytes[71] == 0x99);
}

static void test_string_instructions(void)
{
    uint8_t memory_bytes[256] = { 0xF3, 0xA4, 0xF3, 0xAA, 0xF3, 0xA6 };
    x86emu_cpu cpu;
    x86emu_memory memory = { memory_bytes, sizeof(memory_bytes), 0 };
    x86emu_init(&cpu, memory, 0);
    memory_bytes[20] = 0x11;
    memory_bytes[21] = 0x22;
    memory_bytes[22] = 0x33;
    memory_bytes[23] = 0x44;
    cpu.registers[X86EMU_RSI] = 20;
    cpu.registers[X86EMU_RDI] = 40;
    cpu.registers[X86EMU_RCX] = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RSI] == 24);
    assert(cpu.registers[X86EMU_RDI] == 44);
    assert(cpu.registers[X86EMU_RCX] == 0);
    assert(memcmp(memory_bytes + 40, memory_bytes + 20, 4) == 0);

    cpu.registers[X86EMU_RAX] = 0xAA;
    cpu.registers[X86EMU_RDI] = 50;
    cpu.registers[X86EMU_RCX] = 3;
    cpu.rip = 2;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RDI] == 53);
    assert(cpu.registers[X86EMU_RCX] == 0);
    assert(memory_bytes[50] == 0xAA && memory_bytes[51] == 0xAA && memory_bytes[52] == 0xAA);

    memory_bytes[60] = 1;
    memory_bytes[61] = 2;
    memory_bytes[62] = 9;
    memory_bytes[70] = 1;
    memory_bytes[71] = 2;
    memory_bytes[72] = 3;
    cpu.registers[X86EMU_RSI] = 60;
    cpu.registers[X86EMU_RDI] = 70;
    cpu.registers[X86EMU_RCX] = 4;
    cpu.rip = 4;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RSI] == 63);
    assert(cpu.registers[X86EMU_RDI] == 73);
    assert(cpu.registers[X86EMU_RCX] == 1);
    assert((cpu.rflags & X86EMU_FLAG_ZF) == 0);
}

static bool handle_system_call(x86emu_cpu *cpu, x86asm_opcode opcode, void *user_data)
{
    (void)user_data;
    if (opcode != X86ASM_OP_SYSCALL) return false;
    cpu->registers[X86EMU_RAX] = 123;
    return true;
}

static void test_flags_and_system_hook(void)
{
    uint8_t code[64] = { 0x9C, 0x9D, 0x0F, 0x05, 0x0F, 0x07, 0xF4 };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RSP] = 32;
    cpu.rflags |= X86EMU_FLAG_CF;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    cpu.rflags &= ~UINT64_C(1);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert((cpu.rflags & X86EMU_FLAG_CF) != 0);
    x86emu_set_system_handler(&cpu, handle_system_call);
    cpu.rip = 2;
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(cpu.registers[X86EMU_RAX] == 123 && cpu.rip == 4);
    assert(x86emu_step(&cpu) == X86EMU_ERR_PRIVILEGED);
    cpu.rip = 6;
    assert(x86emu_step(&cpu) == X86EMU_ERR_PRIVILEGED);
}

static void test_pabs(void)
{
    uint8_t code[35] = {
        0x66,0x0F,0x38,0x1C,0xCA, 0x66,0x0F,0x38,0x1D,0xCA, 0x66,0x0F,0x38,0x1E,0xCA,
        0xC4,0xE2,0x05,0x1C,0xCA, 0xC4,0xE2,0x05,0x1D,0xCA, 0xC4,0xE2,0x05,0x1E,0xCA
    };
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0);
    uint8_t bytes[16] = { 0x80u,0x7Fu,0xFFu,0x01u,0x80u,0x00u,0xFEu,0x7Fu,0x12u,0x34u,0x80u,0xFFu,0x55u,0xAAu,0x00u,0x01u };
    memcpy(cpu.vector_registers[2], bytes, sizeof(bytes)); cpu.rip=0; assert(x86emu_step(&cpu)==X86EMU_OK); assert(cpu.vector_registers[1][0]==0x80u && cpu.vector_registers[1][1]==0x7Fu && cpu.vector_registers[1][2]==0x01u && cpu.vector_registers[1][4]==0x80u);
    memcpy(cpu.vector_registers[2], bytes, sizeof(bytes)); cpu.rip=5; assert(x86emu_step(&cpu)==X86EMU_OK); uint16_t words[8]; memcpy(words,cpu.vector_registers[1],sizeof(words)); assert(words[0]==UINT16_C(0x7F80) && words[1]==UINT16_C(0x01FF) && words[2]==UINT16_C(0x0080));
    memcpy(cpu.vector_registers[2], bytes, sizeof(bytes)); cpu.rip=10; assert(x86emu_step(&cpu)==X86EMU_OK); uint32_t dwords[4]; memcpy(dwords,cpu.vector_registers[1],sizeof(dwords)); assert(dwords[0]==UINT32_C(0x01FF7F80));
    uint8_t ymm_input[32]; for (unsigned i=0;i<32u;++i) ymm_input[i] = (uint8_t)(i * 13u); ymm_input[0]=0x80u; ymm_input[16]=0xFFu; memcpy(cpu.vector_registers[2],ymm_input,sizeof(ymm_input)); memset(cpu.vector_registers[1]+32u,0xA5,32u); cpu.rip=15; assert(x86emu_step(&cpu)==X86EMU_OK); assert(cpu.vector_registers[1][0]==0x80u && cpu.vector_registers[1][16]==0x01u); for (unsigned i=32u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
    cpu.rip=20; assert(x86emu_step(&cpu)==X86EMU_OK); cpu.rip=25; assert(x86emu_step(&cpu)==X86EMU_OK);
}

static void test_psign(void)
{
    uint8_t code[35] = {
        0x66,0x0F,0x38,0x08,0xCA, 0x66,0x0F,0x38,0x09,0xCA, 0x66,0x0F,0x38,0x0A,0xCA,
        0xC4,0xE2,0x65,0x08,0xCA, 0xC4,0xE2,0x65,0x09,0xCA, 0xC4,0xE2,0x65,0x0A,0xCA
    };
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0);
    uint8_t input[16] = { 1u,2u,0x80u,0x7Fu,0xFFu,4u,5u,6u,7u,8u,9u,10u,11u,12u,13u,14u };
    uint8_t control[16] = { 1u,0u,0x80u,2u,0xFFu,0u,1u,0x80u,1u,0u,0xFFu,2u,0u,1u,0x80u,0u };
    memcpy(cpu.vector_registers[1], input, sizeof(input)); memcpy(cpu.vector_registers[2], control, sizeof(control)); cpu.rip=0; assert(x86emu_step(&cpu)==X86EMU_OK); assert(cpu.vector_registers[1][0]==1u && cpu.vector_registers[1][1]==0u && cpu.vector_registers[1][2]==0x80u && cpu.vector_registers[1][3]==0x7Fu && cpu.vector_registers[1][4]==1u);
    uint16_t words[8] = { UINT16_C(0x0002), UINT16_C(0xFFFE), UINT16_C(0x7FFF), UINT16_C(0x8000), UINT16_C(0x1234), UINT16_C(0xFFFF), UINT16_C(0x0001), UINT16_C(0x8000) }; uint16_t signs[8] = { UINT16_C(0x0001),0u,UINT16_C(0x8000),UINT16_C(0x0002),UINT16_C(0xFFFF),UINT16_C(0x0001),0u,UINT16_C(0x8000) }; memcpy(cpu.vector_registers[1],words,sizeof(words)); memcpy(cpu.vector_registers[2],signs,sizeof(signs)); cpu.rip=5; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(words,cpu.vector_registers[1],sizeof(words)); assert(words[0]==UINT16_C(0x0002) && words[1]==0u && words[2]==UINT16_C(0x8001) && words[3]==UINT16_C(0x8000) && words[4]==UINT16_C(0xEDCC));
    uint32_t dwords[4] = { 2u, UINT32_C(0xFFFFFFFE), UINT32_C(0x80000000), 4u }; uint32_t dsigns[4] = { 0u, UINT32_C(0xFFFFFFFF), 0u, UINT32_C(0x80000000) }; memcpy(cpu.vector_registers[1],dwords,sizeof(dwords)); memcpy(cpu.vector_registers[2],dsigns,sizeof(dsigns)); cpu.rip=10; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(dwords,cpu.vector_registers[1],sizeof(dwords)); assert(dwords[0]==0u && dwords[1]==2u && dwords[2]==0u && dwords[3]==UINT32_C(0xFFFFFFFC));
    uint8_t ymm_input[32]; uint8_t ymm_control[32]; for (unsigned i=0;i<32u;++i) { ymm_input[i]=(uint8_t)(i+1u); ymm_control[i]=(uint8_t)(i%3u==0u?0x80u:(i%3u==1u?0u:1u)); } memcpy(cpu.vector_registers[3],ymm_input,sizeof(ymm_input)); memcpy(cpu.vector_registers[2],ymm_control,sizeof(ymm_control)); memset(cpu.vector_registers[1]+32u,0xA5,32u); cpu.rip=15; assert(x86emu_step(&cpu)==X86EMU_OK); assert(cpu.vector_registers[1][0]==0xFFu && cpu.vector_registers[1][1]==0u && cpu.vector_registers[1][2]==3u); for (unsigned i=32u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
    cpu.rip=20; assert(x86emu_step(&cpu)==X86EMU_OK); cpu.rip=25; assert(x86emu_step(&cpu)==X86EMU_OK);
}

static void test_phadd(void)
{
    uint8_t code[30] = {
        0x66,0x0F,0x38,0x01,0xCA, 0x66,0x0F,0x38,0x02,0xCA, 0x66,0x0F,0x38,0x03,0xCA,
        0xC4,0xE2,0x65,0x01,0xCA, 0xC4,0xE2,0x65,0x02,0xCA, 0xC4,0xE2,0x65,0x03,0xCA
    };
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0);
    int16_t words_left[8] = { 1,2,3,4,5,6,7,8 }; int16_t words_right[8] = { 10,20,30,40,50,60,70,80 }; int16_t words_out[8];
    memcpy(cpu.vector_registers[1], words_left, sizeof(words_left)); memcpy(cpu.vector_registers[2], words_right, sizeof(words_right)); cpu.rip=0; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(words_out,cpu.vector_registers[1],sizeof(words_out)); assert(words_out[0]==3 && words_out[1]==7 && words_out[2]==11 && words_out[3]==15 && words_out[4]==30 && words_out[5]==70 && words_out[6]==110 && words_out[7]==150);
    int32_t dwords_left[4] = { 1,2,3,4 }; int32_t dwords_right[4] = { 10,20,30,40 }; int32_t dwords_out[4]; memcpy(cpu.vector_registers[1],dwords_left,sizeof(dwords_left)); memcpy(cpu.vector_registers[2],dwords_right,sizeof(dwords_right)); cpu.rip=5; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(dwords_out,cpu.vector_registers[1],sizeof(dwords_out)); assert(dwords_out[0]==3 && dwords_out[1]==7 && dwords_out[2]==30 && dwords_out[3]==70);
    int16_t sat_left[8] = { INT16_MAX,1,INT16_MIN,-1,100,200,-200,-100 }; int16_t sat_right[8] = { 30000,10000,-30000,-10000,INT16_MAX,1,INT16_MIN,-1 }; memcpy(cpu.vector_registers[1],sat_left,sizeof(sat_left)); memcpy(cpu.vector_registers[2],sat_right,sizeof(sat_right)); cpu.rip=10; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(words_out,cpu.vector_registers[1],sizeof(words_out)); assert(words_out[0]==INT16_MAX && words_out[1]==INT16_MIN && words_out[2]==300 && words_out[3]==-300 && words_out[4]==INT16_MAX && words_out[5]==INT16_MIN && words_out[6]==INT16_MAX && words_out[7]==INT16_MIN);
    int16_t ymm_left_words[16]; int16_t ymm_right_words[16]; for (unsigned i=0;i<16u;++i) { ymm_left_words[i]=(int16_t)(i+1); ymm_right_words[i]=(int16_t)(100+i); } memcpy(cpu.vector_registers[3],ymm_left_words,sizeof(ymm_left_words)); memcpy(cpu.vector_registers[2],ymm_right_words,sizeof(ymm_right_words)); memset(cpu.vector_registers[1]+32u,0xA5,32u); cpu.rip=15; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(words_out,cpu.vector_registers[1],sizeof(words_out)); assert(words_out[0]==3 && words_out[1]==7 && words_out[2]==11 && words_out[3]==15 && words_out[4]==201 && words_out[5]==205 && words_out[6]==209 && words_out[7]==213); for (unsigned i=32u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
    cpu.rip=20; assert(x86emu_step(&cpu)==X86EMU_OK); cpu.rip=25; assert(x86emu_step(&cpu)==X86EMU_OK);
    uint8_t memory_code[48] = { 0x66,0x0F,0x38,0x01,0x08 };
    int16_t memory_source[8] = { 10,20,30,40,50,60,70,80 }; memcpy(memory_code + 32u, memory_source, sizeof(memory_source));
    x86emu_cpu memory_cpu; x86emu_memory borrowed = { memory_code, sizeof(memory_code), 0 }; x86emu_init(&memory_cpu, borrowed, 0); int16_t memory_dest[8] = { 1,2,3,4,5,6,7,8 }; memcpy(memory_cpu.vector_registers[1], memory_dest, sizeof(memory_dest)); memory_cpu.registers[X86EMU_RAX] = 32u; memory_cpu.rip = 0; assert(x86emu_step(&memory_cpu) == X86EMU_OK); memcpy(words_out, memory_cpu.vector_registers[1], sizeof(words_out)); assert(words_out[0] == 3 && words_out[1] == 7 && words_out[2] == 11 && words_out[3] == 15 && words_out[4] == 30 && words_out[5] == 70 && words_out[6] == 110 && words_out[7] == 150);
}

static void test_phsub(void)
{
    uint8_t code[30] = {
        0x66,0x0F,0x38,0x05,0xCA, 0x66,0x0F,0x38,0x06,0xCA, 0x66,0x0F,0x38,0x07,0xCA,
        0xC4,0xE2,0x65,0x05,0xCA, 0xC4,0xE2,0x65,0x06,0xCA, 0xC4,0xE2,0x65,0x07,0xCA
    };
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0);
    int16_t left[8] = { 1,2,3,4,5,6,7,8 }; int16_t right[8] = { 10,20,30,40,50,60,70,80 }; int16_t out[8];
    memcpy(cpu.vector_registers[1],left,sizeof(left)); memcpy(cpu.vector_registers[2],right,sizeof(right)); cpu.rip=0; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(out,cpu.vector_registers[1],sizeof(out)); assert(out[0]==-1 && out[1]==-1 && out[2]==-1 && out[3]==-1 && out[4]==-10 && out[5]==-10 && out[6]==-10 && out[7]==-10);
    int32_t dleft[4] = { 1,2,3,4 }; int32_t dright[4] = { 10,20,30,40 }; int32_t dout[4]; memcpy(cpu.vector_registers[1],dleft,sizeof(dleft)); memcpy(cpu.vector_registers[2],dright,sizeof(dright)); cpu.rip=5; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(dout,cpu.vector_registers[1],sizeof(dout)); assert(dout[0]==-1 && dout[1]==-1 && dout[2]==-10 && dout[3]==-10);
    int16_t sat_left[8] = { INT16_MIN,1,INT16_MAX,-1,100,-200,-300,400 }; int16_t sat_right[8] = { INT16_MAX,-1,-32768,1,1000,-2000,2000,-1000 }; memcpy(cpu.vector_registers[1],sat_left,sizeof(sat_left)); memcpy(cpu.vector_registers[2],sat_right,sizeof(sat_right)); cpu.rip=10; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(out,cpu.vector_registers[1],sizeof(out)); assert(out[0]==INT16_MIN && out[1]==INT16_MAX && out[2]==300 && out[3]==-700 && out[4]==INT16_MAX && out[5]==INT16_MIN && out[6]==3000 && out[7]==3000);
    int16_t ymm_left[16]; int16_t ymm_right[16]; int16_t ymm_out[16]; for (unsigned i=0;i<16u;++i) { ymm_left[i]=(int16_t)(i*10u+1u); ymm_right[i]=(int16_t)(i*3u+2u); } memcpy(cpu.vector_registers[3],ymm_left,sizeof(ymm_left)); memcpy(cpu.vector_registers[2],ymm_right,sizeof(ymm_right)); memset(cpu.vector_registers[1]+32u,0xA5,32u); cpu.rip=15; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(ymm_out,cpu.vector_registers[1],sizeof(ymm_out)); for (unsigned i=0;i<4u;++i) assert(ymm_out[i]==-10); for (unsigned i=4u;i<8u;++i) assert(ymm_out[i]==-3); for (unsigned i=8u;i<12u;++i) assert(ymm_out[i]==-10); for (unsigned i=12u;i<16u;++i) assert(ymm_out[i]==-3); for (unsigned i=32u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
    cpu.rip=20; assert(x86emu_step(&cpu)==X86EMU_OK); cpu.rip=25; assert(x86emu_step(&cpu)==X86EMU_OK);
}

static void test_pmaddubsw(void)
{
    uint8_t code[10] = { 0x66,0x0F,0x38,0x04,0xCA, 0xC4,0xE2,0x65,0x04,0xCA };
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0);
    uint8_t left[16] = { 2u,3u, 10u,20u, 255u,2u, 128u,127u, 1u,1u, 200u,100u, 255u,255u, 128u,128u };
    int8_t right[16] = { 4,5, -1,2, 127,127, -128,-128, -1,1, -2,3, -128,127, -128,127 };
    int16_t expected[8] = { 23,30, 32639,-32640, 0,-100, -255,-128 };
    memcpy(cpu.vector_registers[1],left,sizeof(left)); memcpy(cpu.vector_registers[2],right,sizeof(right)); cpu.rip=0; assert(x86emu_step(&cpu)==X86EMU_OK); int16_t output[8]; memcpy(output,cpu.vector_registers[1],sizeof(output)); for (unsigned i=0;i<8u;++i) assert(output[i]==expected[i]);
    uint8_t ymm_left[32]; int8_t ymm_right[32]; int16_t ymm_expected[16]; for (unsigned i=0;i<32u;++i) { ymm_left[i]=(uint8_t)(i+1u); ymm_right[i]=(int8_t)((i%4u)==0u ? -2 : ((i%4u)==1u ? 3 : 1)); } for (unsigned i=0;i<16u;++i) ymm_expected[i]=(int16_t)((int32_t)ymm_left[2u*i]*(int32_t)ymm_right[2u*i]+(int32_t)ymm_left[2u*i+1u]*(int32_t)ymm_right[2u*i+1u]); memcpy(cpu.vector_registers[3],ymm_left,sizeof(ymm_left)); memcpy(cpu.vector_registers[2],ymm_right,sizeof(ymm_right)); memset(cpu.vector_registers[1]+32u,0xA5,32u); cpu.rip=5; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(output,cpu.vector_registers[1],sizeof(output)); for (unsigned i=0;i<8u;++i) assert(output[i]==ymm_expected[i]); for (unsigned i=32u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
}

static void test_pmaddwd(void)
{
    uint8_t code[9] = { 0x66,0x0F,0xF5,0xCA, 0xC4,0xE1,0x65,0xF5,0xCA };
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0);
    int16_t left[8] = { 1,2, -3,4, 300,-2, INT16_MIN,INT16_MIN }; int16_t right[8] = { 10,20, 5,-6, -2,3, INT16_MIN,INT16_MIN }; int32_t expected[4] = { 50, -39, -606, INT32_MIN }; int32_t output[4];
    memcpy(cpu.vector_registers[1],left,sizeof(left)); memcpy(cpu.vector_registers[2],right,sizeof(right)); cpu.rip=0; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(output,cpu.vector_registers[1],sizeof(output)); for (unsigned i=0;i<4u;++i) assert(output[i]==expected[i]);
    int16_t ymm_left[16]; int16_t ymm_right[16]; int32_t ymm_expected[8]; for (unsigned i=0;i<16u;++i) { ymm_left[i]=(int16_t)(i+1u); ymm_right[i]=(int16_t)(i%2u==0u ? 2 : -1); } for (unsigned i=0;i<8u;++i) ymm_expected[i]=(int32_t)ymm_left[2u*i]*ymm_right[2u*i]+(int32_t)ymm_left[2u*i+1u]*ymm_right[2u*i+1u]; memcpy(cpu.vector_registers[3],ymm_left,sizeof(ymm_left)); memcpy(cpu.vector_registers[2],ymm_right,sizeof(ymm_right)); memset(cpu.vector_registers[1]+32u,0xA5,32u); cpu.rip=4; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(output,cpu.vector_registers[1],sizeof(output)); for (unsigned i=0;i<4u;++i) assert(output[i]==ymm_expected[i]); for (unsigned i=32u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
}

static void test_pmuldq(void)
{
    uint8_t code[10] = { 0x66,0x0F,0x38,0x28,0xCA, 0xC4,0xE2,0x65,0x28,0xCA };
    x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0);
    int32_t left[4] = { 2,99,-3,7 }; int32_t right[4] = { 10,4,-5,6 }; int64_t expected[2] = { 20,15 }; int64_t output[2];
    memcpy(cpu.vector_registers[1],left,sizeof(left)); memcpy(cpu.vector_registers[2],right,sizeof(right)); cpu.rip=0; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(output,cpu.vector_registers[1],sizeof(output)); assert(output[0]==expected[0] && output[1]==expected[1]);
    int32_t ymm_left[8] = { 2,99,-3,7,4,88,-5,9 }; int32_t ymm_right[8] = { 10,4,-5,6,-2,1,7,8 }; int64_t ymm_expected[4] = { 20,15,-8,-35 }; int64_t ymm_output[4]; memcpy(cpu.vector_registers[3],ymm_left,sizeof(ymm_left)); memcpy(cpu.vector_registers[2],ymm_right,sizeof(ymm_right)); cpu.rip=5; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(ymm_output,cpu.vector_registers[1],sizeof(ymm_output)); for (unsigned i=0;i<4u;++i) assert(ymm_output[i]==ymm_expected[i]);
    uint8_t memory_code[32] = { 0x66,0x0F,0x38,0x28,0x08 }; int32_t memory_source[4] = { 10,111,-5,222 }; memcpy(memory_code+16u,memory_source,sizeof(memory_source)); x86emu_cpu memory_cpu; x86emu_memory borrowed = { memory_code,sizeof(memory_code),0 }; x86emu_init(&memory_cpu,borrowed,0); memcpy(memory_cpu.vector_registers[1],left,sizeof(left)); memory_cpu.registers[X86EMU_RAX]=16u; assert(x86emu_step(&memory_cpu)==X86EMU_OK); memcpy(output,memory_cpu.vector_registers[1],sizeof(output)); assert(output[0]==20 && output[1]==15);
}

static void test_pmov(void)
{
    uint8_t code[5] = { 0x66,0x0F,0x38,0x20,0xCA }; x86emu_cpu cpu; x86emu_memory memory = { code, sizeof(code), 0 }; x86emu_init(&cpu, memory, 0);
    int8_t bytes[16] = { 0, -1, 127, -128, 1, -2, 64, -64, 8, -8, 9, -9, 10, -10, 11, -11 }; int16_t words[8]; memcpy(cpu.vector_registers[2],bytes,sizeof(bytes)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(words,cpu.vector_registers[1],sizeof(words)); assert(words[0]==0 && words[1]==-1 && words[2]==127 && words[3]==-128 && words[4]==1 && words[5]==-2 && words[6]==64 && words[7]==-64);
    code[3]=0x30; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],bytes,sizeof(bytes)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(words,cpu.vector_registers[1],sizeof(words)); assert(words[0]==0 && words[1]==255 && words[2]==127 && words[3]==128 && words[4]==1 && words[5]==254 && words[6]==64 && words[7]==192);
    code[3]=0x21; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],bytes,sizeof(bytes)); int32_t dwords[4]; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(dwords,cpu.vector_registers[1],sizeof(dwords)); assert(dwords[0]==0 && dwords[1]==-1 && dwords[2]==127 && dwords[3]==-128);
    code[3]=0x31; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],bytes,sizeof(bytes)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(dwords,cpu.vector_registers[1],sizeof(dwords)); assert(dwords[0]==0 && dwords[1]==255 && dwords[2]==127 && dwords[3]==128);
    code[3]=0x22; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],bytes,sizeof(bytes)); int64_t qwords[2]; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(qwords,cpu.vector_registers[1],sizeof(qwords)); assert(qwords[0]==0 && qwords[1]==-1);
    code[3]=0x32; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],bytes,sizeof(bytes)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(qwords,cpu.vector_registers[1],sizeof(qwords)); assert(qwords[0]==0 && qwords[1]==255);
    int16_t source_words[8] = { -1, 32767, INT16_MIN, 2, -3, 4, 5, -6 }; code[3]=0x23; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],source_words,sizeof(source_words)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(dwords,cpu.vector_registers[1],sizeof(dwords)); assert(dwords[0]==-1 && dwords[1]==32767 && dwords[2]==INT16_MIN && dwords[3]==2);
    code[3]=0x33; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],source_words,sizeof(source_words)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(dwords,cpu.vector_registers[1],sizeof(dwords)); assert(dwords[0]==65535 && dwords[1]==32767 && dwords[2]==32768 && dwords[3]==2);
    code[3]=0x24; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],source_words,sizeof(source_words)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(qwords,cpu.vector_registers[1],sizeof(qwords)); assert(qwords[0]==-1 && qwords[1]==32767);
    code[3]=0x34; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],source_words,sizeof(source_words)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(qwords,cpu.vector_registers[1],sizeof(qwords)); assert(qwords[0]==65535 && qwords[1]==32767);
    int32_t source_dwords[4] = { -1, 2, INT32_MIN, 3 }; code[3]=0x25; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],source_dwords,sizeof(source_dwords)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(qwords,cpu.vector_registers[1],sizeof(qwords)); assert(qwords[0]==-1 && qwords[1]==2);
    code[3]=0x35; x86emu_init(&cpu,memory,0); memcpy(cpu.vector_registers[2],source_dwords,sizeof(source_dwords)); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(qwords,cpu.vector_registers[1],sizeof(qwords)); assert(qwords[0]==UINT64_C(4294967295) && qwords[1]==UINT64_C(2));
    uint8_t memory_code[40] = { 0x66,0x0F,0x38,0x21,0x08 }; memcpy(memory_code+32u,bytes,4u); x86emu_memory borrowed = { memory_code,sizeof(memory_code),0 }; x86emu_init(&cpu,borrowed,0); cpu.registers[X86EMU_RAX]=32u; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(dwords,cpu.vector_registers[1],sizeof(dwords)); assert(dwords[0]==0 && dwords[1]==-1 && dwords[2]==127 && dwords[3]==-128);
    uint8_t vex_code[5] = { 0xC4,0xE2,0x05,0x20,0xCA }; x86emu_memory vex_memory = { vex_code,sizeof(vex_code),0 }; x86emu_init(&cpu,vex_memory,0); int8_t vex_bytes[16]; for (unsigned i=0;i<16u;++i) vex_bytes[i]=(int8_t)(i<8u ? -(int)i : (int)i); memcpy(cpu.vector_registers[2],vex_bytes,sizeof(vex_bytes)); memset(cpu.vector_registers[1]+32u,0xA5,32u); assert(x86emu_step(&cpu)==X86EMU_OK); int16_t vex_words[16]; memcpy(vex_words,cpu.vector_registers[1],sizeof(vex_words)); for (unsigned i=0;i<16u;++i) assert(vex_words[i]==(int16_t)vex_bytes[i]); for (unsigned i=32u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
    vex_code[3]=0x35; x86emu_init(&cpu,vex_memory,0); memcpy(cpu.vector_registers[2],source_dwords,sizeof(source_dwords)); memset(cpu.vector_registers[1]+32u,0xA5,32u); assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(qwords,cpu.vector_registers[1],sizeof(qwords)); assert(qwords[0]==UINT64_C(4294967295) && qwords[1]==UINT64_C(2)); for (unsigned i=32u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
    uint8_t vex_memory_code[24] = { 0xC4,0xE2,0x05,0x21,0x08 }; int8_t vex_memory_source[8] = { -1,2,-3,4,127,-128,5,-6 }; memcpy(vex_memory_code+16u,vex_memory_source,sizeof(vex_memory_source)); x86emu_memory vex_borrowed = { vex_memory_code,sizeof(vex_memory_code),0 }; x86emu_init(&cpu,vex_borrowed,0); cpu.registers[X86EMU_RAX]=16u; int32_t vex_memory_output[8]; assert(x86emu_step(&cpu)==X86EMU_OK); memcpy(vex_memory_output,cpu.vector_registers[1],sizeof(vex_memory_output)); for (unsigned i=0;i<8u;++i) assert(vex_memory_output[i]==(int32_t)vex_memory_source[i]);
}

static void test_phminposuw(void)
{
    uint8_t code[5] = { 0x66,0x0F,0x38,0x41,0xCA }; x86emu_cpu cpu; x86emu_memory memory = { code,sizeof(code),0 }; x86emu_init(&cpu,memory,0);
    uint16_t source[8] = { 500, 7, 7, 2, 65535, 2, 9, 3 }; uint8_t expected[16] = { 0x02,0x00,0x03,0x00,0,0,0,0,0,0,0,0,0,0,0,0 }; memcpy(cpu.vector_registers[2],source,sizeof(source)); memset(cpu.vector_registers[1],0xA5,64u); assert(x86emu_step(&cpu)==X86EMU_OK); assert(memcmp(cpu.vector_registers[1],expected,sizeof(expected))==0); for (unsigned i=16u;i<64u;++i) assert(cpu.vector_registers[1][i]==0xA5u);
    uint8_t memory_code[32] = { 0x66,0x0F,0x38,0x41,0x08 }; uint16_t memory_source[8] = { 10, 9, 8, 7, 6, 5, 4, 3 }; memcpy(memory_code+16u,memory_source,sizeof(memory_source)); x86emu_memory borrowed = { memory_code,sizeof(memory_code),0 }; x86emu_init(&cpu,borrowed,0); cpu.registers[X86EMU_RAX]=16u; memset(cpu.vector_registers[1],0xA5,64u); assert(x86emu_step(&cpu)==X86EMU_OK); assert(cpu.vector_registers[1][0]==3u && cpu.vector_registers[1][1]==0u && cpu.vector_registers[1][2]==7u && cpu.vector_registers[1][3]==0u);
    uint8_t vex_code[5] = { 0xC4,0xE2,0x01,0x41,0xCA }; x86emu_memory vex_memory = { vex_code,sizeof(vex_code),0 }; x86emu_init(&cpu,vex_memory,0); memcpy(cpu.vector_registers[2],source,sizeof(source)); memset(cpu.vector_registers[1]+16u,0xA5,48u); assert(x86emu_step(&cpu)==X86EMU_OK); assert(memcmp(cpu.vector_registers[1],expected,sizeof(expected))==0); for (unsigned i=16u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
}

static void test_movntdq(void)
{
    uint8_t source[32]; for (unsigned i = 0; i < sizeof(source); ++i) source[i] = (uint8_t)(0x20u + i);
    uint8_t legacy_memory[64] = { 0x66,0x0F,0xE7,0x08 }; memset(legacy_memory + 32u, 0xA5, 16u); x86emu_cpu cpu; x86emu_memory memory = { legacy_memory, sizeof(legacy_memory), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[1], source, 16u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(legacy_memory + 32u, source, 16u) == 0);
    uint8_t vex128_memory[64] = { 0xC4,0xE1,0x01,0xE7,0x08 }; memset(vex128_memory + 32u, 0xA5, 16u); memory = (x86emu_memory){ vex128_memory, sizeof(vex128_memory), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[1], source, 16u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(vex128_memory + 32u, source, 16u) == 0);
    uint8_t vex256_memory[80] = { 0xC4,0xE1,0x05,0xE7,0x08 }; memset(vex256_memory + 32u, 0xA5, sizeof(source)); memory = (x86emu_memory){ vex256_memory, sizeof(vex256_memory), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[1], source, sizeof(source)); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(vex256_memory + 32u, source, sizeof(source)) == 0);
    x86emu_memory short_memory = { vex256_memory, 63u, 0 }; x86emu_init(&cpu, short_memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[1], source, sizeof(source)); assert(x86emu_step(&cpu) == X86EMU_ERR_MEMORY); assert(cpu.rip == 0u);
}

static void test_legacy_movdqa_movdqu(void)
{
    uint8_t source[16];
    for (unsigned i = 0; i < sizeof(source); ++i) source[i] = (uint8_t)(0x30u + i);
    uint8_t code[96] = { 0x66,0x0F,0x6F,0x08 };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };

    memcpy(code + 32u, source, sizeof(source));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 32u;
    memset(cpu.vector_registers[1], 0xA5, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, sizeof(source)) == 0);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xA5u);

    memcpy(code, (const uint8_t[]){ 0x66,0x0F,0x7F,0x08 }, 4u);
    memset(code + 32u, 0xA5, sizeof(source));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 32u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 32u, source, sizeof(source)) == 0);

    memcpy(code, (const uint8_t[]){ 0xF3,0x0F,0x6F,0x08 }, 4u);
    memcpy(code + 33u, source, sizeof(source));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 33u;
    memset(cpu.vector_registers[1], 0xA5, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, sizeof(source)) == 0);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xA5u);

    memcpy(code, (const uint8_t[]){ 0xF3,0x0F,0x7F,0x08 }, 4u);
    memset(code + 33u, 0xA5, sizeof(source));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 33u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 33u, source, sizeof(source)) == 0);

    x86emu_memory short_memory = { code, 48u, 0 };
    x86emu_init(&cpu, short_memory, 0);
    cpu.registers[X86EMU_RAX] = 33u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_ERR_MEMORY);
    assert(cpu.rip == 0u);
}

static void test_vmovdqa_vmovdqu(void)
{
    uint8_t source[32];
    for (unsigned i = 0; i < sizeof(source); ++i) source[i] = (uint8_t)(0x60u + i);
    uint8_t code[96] = { 0xC4,0xE1,0x01,0x6F,0xCA };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };

    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[2], source, 16u);
    memset(cpu.vector_registers[1], 0xA5, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, 16u) == 0);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memset(code, 0, sizeof(code));
    memcpy(code, (const uint8_t[]){ 0xC4,0xE1,0x06,0x6F,0xCA }, 5u);
    x86emu_init(&cpu, memory, 0);
    memcpy(cpu.vector_registers[2], source, sizeof(source));
    memset(cpu.vector_registers[1], 0xA5, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, sizeof(source)) == 0);
    for (unsigned i = 32u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC4,0xE1,0x01,0x6F,0x08 }, 5u);
    memcpy(code + 32u, source, 16u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 32u;
    memset(cpu.vector_registers[1], 0xA5, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, 16u) == 0);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC4,0xE1,0x02,0x6F,0x08 }, 5u);
    memcpy(code + 32u, source, 16u);
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 32u;
    memset(cpu.vector_registers[1], 0xA5, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, 16u) == 0);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC4,0xE1,0x06,0x6F,0x08 }, 5u);
    memcpy(code + 32u, source, sizeof(source));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 32u;
    memset(cpu.vector_registers[1], 0xA5, sizeof(cpu.vector_registers[1]));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], source, sizeof(source)) == 0);
    for (unsigned i = 32u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);

    memcpy(code, (const uint8_t[]){ 0xC4,0xE1,0x06,0x7F,0x08 }, 5u);
    memset(code + 32u, 0xA5, sizeof(source));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 32u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 32u, source, sizeof(source)) == 0);

    memcpy(code, (const uint8_t[]){ 0xC4,0xE1,0x02,0x7F,0x08 }, 5u);
    memset(code + 32u, 0xA5, sizeof(source));
    x86emu_init(&cpu, memory, 0);
    cpu.registers[X86EMU_RAX] = 32u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(code + 32u, source, 16u) == 0);
    for (unsigned i = 16u; i < 32u; ++i) assert(code[32u + i] == 0xA5u);

    x86emu_memory short_memory = { code, 47u, 0 };
    x86emu_init(&cpu, short_memory, 0);
    cpu.registers[X86EMU_RAX] = 32u;
    memcpy(cpu.vector_registers[1], source, sizeof(source));
    assert(x86emu_step(&cpu) == X86EMU_ERR_MEMORY);
    assert(cpu.rip == 0u);
}

static void test_lddqu(void)
{
    uint8_t source[32]; for (unsigned i = 0; i < sizeof(source); ++i) source[i] = (uint8_t)(0x80u + i);
    uint8_t legacy_memory[64] = { 0xF2,0x0F,0xF0,0x08 }; memcpy(legacy_memory + 17u, source, 16u); x86emu_cpu cpu; x86emu_memory memory = { legacy_memory, sizeof(legacy_memory), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 17u; memset(cpu.vector_registers[1], 0xA5, 64u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(cpu.vector_registers[1], source, 16u) == 0); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xA5u);
    uint8_t vex128_memory[64] = { 0xC4,0xE1,0x02,0xF0,0x08 }; memcpy(vex128_memory + 17u, source, 16u); memory = (x86emu_memory){ vex128_memory, sizeof(vex128_memory), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 17u; memset(cpu.vector_registers[1], 0xA5, 64u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(cpu.vector_registers[1], source, 16u) == 0); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    uint8_t vex256_memory[80] = { 0xC4,0xE1,0x06,0xF0,0x08 }; memcpy(vex256_memory + 17u, source, sizeof(source)); memory = (x86emu_memory){ vex256_memory, sizeof(vex256_memory), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 17u; memset(cpu.vector_registers[1], 0xA5, 64u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(cpu.vector_registers[1], source, sizeof(source)) == 0); for (unsigned i = 32u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    x86emu_memory short_memory = { vex256_memory, 48u, 0 }; x86emu_init(&cpu, short_memory, 0); cpu.registers[X86EMU_RAX] = 17u; assert(x86emu_step(&cpu) == X86EMU_ERR_MEMORY); assert(cpu.rip == 0u);
}

static void test_movntdqa(void)
{
    uint8_t source[32]; for (unsigned i = 0; i < sizeof(source); ++i) source[i] = (uint8_t)(0x40u + i);
    uint8_t memory_code[64] = { 0x66,0x0F,0x38,0x2A,0x08 }; memcpy(memory_code + 32u, source, 16u); x86emu_cpu cpu; x86emu_memory memory = { memory_code, sizeof(memory_code), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memset(cpu.vector_registers[1], 0xA5, 64u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(cpu.vector_registers[1], source, 16u) == 0); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xA5u);
    uint8_t vex128_memory[64] = { 0xC4,0xE2,0x01,0x2A,0x08 }; memcpy(vex128_memory + 32u, source, 16u); memory = (x86emu_memory){ vex128_memory, sizeof(vex128_memory), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memset(cpu.vector_registers[1], 0xA5, 64u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(cpu.vector_registers[1], source, 16u) == 0); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    uint8_t vex256_memory[80] = { 0xC4,0xE2,0x05,0x2A,0x08 }; memcpy(vex256_memory + 32u, source, sizeof(source)); memory = (x86emu_memory){ vex256_memory, sizeof(vex256_memory), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memset(cpu.vector_registers[1], 0xA5, 64u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(cpu.vector_registers[1], source, sizeof(source)) == 0); for (unsigned i = 32u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    x86emu_memory short_memory = { vex256_memory, 63u, 0 }; x86emu_init(&cpu, short_memory, 0); cpu.registers[X86EMU_RAX] = 32u; assert(x86emu_step(&cpu) == X86EMU_ERR_MEMORY); assert(cpu.rip == 0u);
}

static void test_vpblendd(void)
{
    static const uint8_t xmm_code[] = { 0xC4,0xE3,0x61,0x02,0xCA,0xA5 };
    static const uint8_t ymm_code[] = { 0xC4,0xE3,0x65,0x02,0xCA,0x5A };
    uint32_t left[8] = { 1u,2u,3u,4u,5u,6u,7u,8u }, right[8] = { 101u,102u,103u,104u,105u,106u,107u,108u }, out[8];
    x86emu_cpu cpu; x86emu_memory memory = { (uint8_t *)xmm_code, sizeof(xmm_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[3], left, 16); memcpy(cpu.vector_registers[2], right, 16); memset(cpu.vector_registers[1] + 16u, 0xA5, 48u); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out, cpu.vector_registers[1], 16); assert(out[0] == 101u && out[1] == 2u && out[2] == 103u && out[3] == 4u); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    memory = (x86emu_memory){ (uint8_t *)ymm_code, sizeof(ymm_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[3], left, sizeof(left)); memcpy(cpu.vector_registers[2], right, sizeof(right)); memset(cpu.vector_registers[1] + 32u, 0xA5, 32u); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 1u && out[1] == 102u && out[2] == 3u && out[3] == 104u && out[4] == 105u && out[5] == 6u && out[6] == 107u && out[7] == 8u); for (unsigned i = 32u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    uint8_t memory_code[96] = { 0xC4,0xE3,0x65,0x02,0x08,0x5A }; memcpy(memory_code + 64u, right, sizeof(right)); memory = (x86emu_memory){ memory_code, sizeof(memory_code), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 64u; memcpy(cpu.vector_registers[3], left, sizeof(left)); memset(cpu.vector_registers[1] + 32u, 0xA5, 32u); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out, cpu.vector_registers[1], sizeof(out)); assert(out[0] == 1u && out[1] == 102u && out[2] == 3u && out[3] == 104u && out[4] == 105u && out[5] == 6u && out[6] == 107u && out[7] == 8u); for (unsigned i = 32u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
}

static void test_blend_immediate_legacy(void)
{
    static const uint8_t blendps_code[] = { 0x66,0x0F,0x3A,0x0C,0xCA,0x0A };
    static const uint8_t blendpd_code[] = { 0x66,0x0F,0x3A,0x0D,0xCA,0x02 };
    uint32_t left_ps[4] = { 1u,2u,3u,4u }, right_ps[4] = { 101u,102u,103u,104u }, out_ps[4];
    uint64_t left_pd[2] = { 11u,12u }, right_pd[2] = { 111u,112u }, out_pd[2];
    x86emu_cpu cpu; x86emu_memory memory = { (uint8_t *)blendps_code, sizeof(blendps_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], left_ps, sizeof(left_ps)); memcpy(cpu.vector_registers[2], right_ps, sizeof(right_ps)); memset(cpu.vector_registers[1] + 16u, 0xA5, 48u); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out_ps, cpu.vector_registers[1], sizeof(out_ps)); assert(out_ps[0] == 1u && out_ps[1] == 102u && out_ps[2] == 3u && out_ps[3] == 104u); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xA5u);
    memory = (x86emu_memory){ (uint8_t *)blendpd_code, sizeof(blendpd_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], left_pd, sizeof(left_pd)); memcpy(cpu.vector_registers[2], right_pd, sizeof(right_pd)); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out_pd, cpu.vector_registers[1], sizeof(out_pd)); assert(out_pd[0] == 11u && out_pd[1] == 112u);
    uint8_t memory_code[48] = { 0x66,0x0F,0x3A,0x0C,0x00,0x0A }; memcpy(memory_code + 32u, right_ps, sizeof(right_ps)); memory = (x86emu_memory){ memory_code, sizeof(memory_code), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[0], left_ps, sizeof(left_ps)); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out_ps, cpu.vector_registers[0], sizeof(out_ps)); assert(out_ps[0] == 1u && out_ps[1] == 102u && out_ps[2] == 3u && out_ps[3] == 104u);
}

static void test_blendv_fp(void)
{
    static const uint8_t legacy_ps[] = { 0x66,0x0F,0x38,0x14,0xCA };
    static const uint8_t legacy_pd[] = { 0x66,0x0F,0x38,0x15,0xCA };
    static const uint8_t vex_ps[] = { 0xC4,0xE3,0x61,0x4A,0xCA,0x40 };
    static const uint8_t vex_pd[] = { 0xC4,0xE3,0x61,0x4B,0xCA,0x40 };
    uint32_t left_ps[8] = { 1u,2u,3u,4u,5u,6u,7u,8u }, right_ps[8] = { 101u,102u,103u,104u,105u,106u,107u,108u }, mask_ps[8] = { 0u,UINT32_C(0x80000000),0u,UINT32_C(0x80000000),0u,UINT32_C(0x80000000),0u,UINT32_C(0x80000000) }, out_ps[8];
    uint64_t left_pd[4] = { 11u,12u,13u,14u }, right_pd[4] = { 111u,112u,113u,114u }, mask_pd[4] = { 0u,UINT64_C(0x8000000000000000),0u,UINT64_C(0x8000000000000000) }, out_pd[4];
    x86emu_cpu cpu; x86emu_memory memory = { (uint8_t *)legacy_ps, sizeof(legacy_ps), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], left_ps, 16); memcpy(cpu.vector_registers[2], right_ps, 16); memcpy(cpu.vector_registers[0], mask_ps, 16); memset(cpu.vector_registers[1] + 16u, 0xA5, 48u); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out_ps, cpu.vector_registers[1], 16); assert(out_ps[0] == 1u && out_ps[1] == 102u && out_ps[2] == 3u && out_ps[3] == 104u); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xA5u);
    memory = (x86emu_memory){ (uint8_t *)legacy_pd, sizeof(legacy_pd), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], left_pd, 16); memcpy(cpu.vector_registers[2], right_pd, 16); memcpy(cpu.vector_registers[0], mask_pd, 16); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out_pd, cpu.vector_registers[1], 16); assert(out_pd[0] == 11u && out_pd[1] == 112u);
    memory = (x86emu_memory){ (uint8_t *)vex_ps, sizeof(vex_ps), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[3], left_ps, 16); memcpy(cpu.vector_registers[2], right_ps, 16); memcpy(cpu.vector_registers[4], mask_ps, 16); memset(cpu.vector_registers[1] + 16u, 0xA5, 48u); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out_ps, cpu.vector_registers[1], 16); assert(out_ps[0] == 1u && out_ps[1] == 102u && out_ps[2] == 3u && out_ps[3] == 104u); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    memory = (x86emu_memory){ (uint8_t *)vex_pd, sizeof(vex_pd), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[3], left_pd, 16); memcpy(cpu.vector_registers[2], right_pd, 16); memcpy(cpu.vector_registers[4], mask_pd, 16); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out_pd, cpu.vector_registers[1], 16); assert(out_pd[0] == 11u && out_pd[1] == 112u);
    uint8_t memory_code[64] = { 0xC4,0xE3,0x61,0x4A,0x08,0x40 }; memcpy(memory_code + 32u, right_ps, 16); memory = (x86emu_memory){ memory_code, sizeof(memory_code), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[3], left_ps, 16); memcpy(cpu.vector_registers[4], mask_ps, 16); assert(x86emu_step(&cpu) == X86EMU_OK); memcpy(out_ps, cpu.vector_registers[1], 16); assert(out_ps[0] == 1u && out_ps[1] == 102u && out_ps[2] == 3u && out_ps[3] == 104u);
}

static void test_pblendvb(void)
{
    uint8_t legacy_code[5] = { 0x66,0x0F,0x38,0x10,0xCA }; x86emu_cpu cpu; x86emu_memory legacy_memory = { legacy_code,sizeof(legacy_code),0 }; x86emu_init(&cpu,legacy_memory,0);
    uint8_t left[16]; uint8_t right[16]; uint8_t mask[16]; uint8_t expected[16];
    for (unsigned i=0;i<16u;++i) { left[i]=(uint8_t)i; right[i]=(uint8_t)(0x80u+i); mask[i]=(uint8_t)(i%2u==0u ? 0x80u : 0u); expected[i]=(mask[i]&0x80u)!=0u ? right[i] : left[i]; }
    memcpy(cpu.vector_registers[1],left,sizeof(left)); memcpy(cpu.vector_registers[2],right,sizeof(right)); memcpy(cpu.vector_registers[0],mask,sizeof(mask)); assert(x86emu_step(&cpu)==X86EMU_OK); assert(memcmp(cpu.vector_registers[1],expected,sizeof(expected))==0);
    uint8_t vex_code[6] = { 0xC4,0xE3,0x65,0x4C,0xCA,0x40 }; x86emu_memory vex_memory = { vex_code,sizeof(vex_code),0 }; x86emu_init(&cpu,vex_memory,0); uint8_t ymm_left[32]; uint8_t ymm_right[32]; uint8_t ymm_mask[32]; uint8_t ymm_expected[32]; for (unsigned i=0;i<32u;++i) { ymm_left[i]=(uint8_t)(i+1u); ymm_right[i]=(uint8_t)(0xC0u+i); ymm_mask[i]=(uint8_t)(i%3u==0u ? 0x80u : 0u); ymm_expected[i]=(ymm_mask[i]&0x80u)!=0u ? ymm_right[i] : ymm_left[i]; } memcpy(cpu.vector_registers[3],ymm_left,sizeof(ymm_left)); memcpy(cpu.vector_registers[2],ymm_right,sizeof(ymm_right)); memcpy(cpu.vector_registers[4],ymm_mask,sizeof(ymm_mask)); memset(cpu.vector_registers[1]+32u,0xA5,32u); assert(x86emu_step(&cpu)==X86EMU_OK); assert(memcmp(cpu.vector_registers[1],ymm_expected,sizeof(ymm_expected))==0); for (unsigned i=32u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
    uint8_t memory_code[64] = { 0xC4,0xE3,0x65,0x4C,0x08,0x40 }; memcpy(memory_code+32u,ymm_right,sizeof(ymm_right)); x86emu_memory borrowed = { memory_code,sizeof(memory_code),0 }; x86emu_init(&cpu,borrowed,0); cpu.registers[X86EMU_RAX]=32u; memcpy(cpu.vector_registers[3],ymm_left,sizeof(ymm_left)); memcpy(cpu.vector_registers[4],ymm_mask,sizeof(ymm_mask)); assert(x86emu_step(&cpu)==X86EMU_OK); assert(memcmp(cpu.vector_registers[1],ymm_expected,sizeof(ymm_expected))==0);
}

static void test_pblendw(void)
{
    uint8_t legacy_code[6] = { 0x66,0x0F,0x3A,0x0E,0xCA,0xA5 }; x86emu_cpu cpu; x86emu_memory legacy_memory = { legacy_code,sizeof(legacy_code),0 }; x86emu_init(&cpu,legacy_memory,0);
    uint16_t left[8] = { 0x1000u,0x1001u,0x1002u,0x1003u,0x1004u,0x1005u,0x1006u,0x1007u }; uint16_t right[8] = { 0x2000u,0x2001u,0x2002u,0x2003u,0x2004u,0x2005u,0x2006u,0x2007u }; uint16_t expected[8];
    for (unsigned i = 0; i < 8u; ++i) {
        expected[i] = (UINT8_C(0xA5) & (uint8_t)(UINT8_C(1) << i)) != 0u ? right[i] : left[i];
    }
    memcpy(cpu.vector_registers[1], left, sizeof(left));
    memcpy(cpu.vector_registers[2], right, sizeof(right));
    memset(cpu.vector_registers[1] + 16u, 0xA5, 48u);
    assert(x86emu_step(&cpu) == X86EMU_OK);
    assert(memcmp(cpu.vector_registers[1], expected, sizeof(expected)) == 0);
    for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xA5u);
    uint8_t vex_xmm_code[6] = { 0xC4,0xE3,0x61,0x0E,0xCA,0xA5 }; x86emu_memory vex_xmm_memory = { vex_xmm_code,sizeof(vex_xmm_code),0 }; x86emu_init(&cpu,vex_xmm_memory,0); memcpy(cpu.vector_registers[3],left,sizeof(left)); memcpy(cpu.vector_registers[2],right,sizeof(right)); memset(cpu.vector_registers[1]+16u,0xA5,48u); assert(x86emu_step(&cpu)==X86EMU_OK); assert(memcmp(cpu.vector_registers[1],expected,sizeof(expected))==0); for (unsigned i=16u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
    uint8_t ymm_left[16] = { 0x10u,0x00u,0x11u,0x00u,0x12u,0x00u,0x13u,0x00u,0x14u,0x00u,0x15u,0x00u,0x16u,0x00u,0x17u,0x00u }; uint8_t ymm_right[32]; uint8_t ymm_expected[32]; for (unsigned i=0;i<32u;++i) { ymm_right[i]=(uint8_t)(0x80u+i); ymm_expected[i]=(i<16u ? ymm_left[i] : ymm_left[i-16u]); } for (unsigned lane=0;lane<32u;lane+=16u) for (unsigned word=0;word<8u;++word) if ((UINT8_C(0x5A)&(uint8_t)(UINT8_C(1)<<word))!=0u) { ymm_expected[lane+2u*word]=ymm_right[lane+2u*word]; ymm_expected[lane+2u*word+1u]=ymm_right[lane+2u*word+1u]; }
    uint8_t memory_code[64] = { 0xC4,0xE3,0x65,0x0E,0x08,0x5A }; memcpy(memory_code+32u,ymm_right,sizeof(ymm_right)); x86emu_memory borrowed = { memory_code,sizeof(memory_code),0 }; x86emu_init(&cpu,borrowed,0); cpu.registers[X86EMU_RAX]=32u; memcpy(cpu.vector_registers[3],ymm_expected,sizeof(ymm_expected)); memset(cpu.vector_registers[1]+32u,0xA5,32u); assert(x86emu_step(&cpu)==X86EMU_OK); assert(memcmp(cpu.vector_registers[1],ymm_expected,sizeof(ymm_expected))==0); for (unsigned i=32u;i<64u;++i) assert(cpu.vector_registers[1][i]==0u);
}

static void test_palignr(void)
{
    uint8_t legacy_code[6] = { 0x66,0x0F,0x3A,0x0F,0xCA,0x04 }; x86emu_cpu cpu; x86emu_memory legacy_memory = { legacy_code,sizeof(legacy_code),0 }; x86emu_init(&cpu,legacy_memory,0);
    uint8_t left[16]; uint8_t right[16]; uint8_t expected[16];
    for (unsigned i = 0; i < 16u; ++i) { left[i] = (uint8_t)(0xA0u + i); right[i] = (uint8_t)(0xB0u + i); }
    for (unsigned i = 0; i < 16u; ++i) expected[i] = i < 12u ? right[i + 4u] : left[i - 12u];
    memcpy(cpu.vector_registers[1], left, sizeof(left)); memcpy(cpu.vector_registers[2], right, sizeof(right)); memset(cpu.vector_registers[1] + 16u, 0xA5, 48u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(cpu.vector_registers[1], expected, sizeof(expected)) == 0); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xA5u);
    uint8_t vex_xmm_code[6] = { 0xC4,0xE3,0x61,0x0F,0xCA,0x04 }; x86emu_memory vex_xmm_memory = { vex_xmm_code,sizeof(vex_xmm_code),0 }; x86emu_init(&cpu,vex_xmm_memory,0); memcpy(cpu.vector_registers[3], left, sizeof(left)); memcpy(cpu.vector_registers[2], right, sizeof(right)); memset(cpu.vector_registers[1] + 16u, 0xA5, 48u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(cpu.vector_registers[1], expected, sizeof(expected)) == 0); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    uint8_t left_ymm[32]; uint8_t right_ymm[32]; uint8_t expected_ymm[32];
    for (unsigned i = 0; i < 32u; ++i) { left_ymm[i] = (uint8_t)(0x10u + i); right_ymm[i] = (uint8_t)(0x80u + i); }
    for (unsigned lane = 0; lane < 32u; lane += 16u) for (unsigned i = 0; i < 16u; ++i) expected_ymm[lane + i] = i < 8u ? right_ymm[lane + i + 8u] : left_ymm[lane + i - 8u];
    uint8_t memory_code[64] = { 0xC4,0xE3,0x65,0x0F,0x08,0x08 }; memcpy(memory_code + 32u, right_ymm, sizeof(right_ymm)); x86emu_memory borrowed = { memory_code,sizeof(memory_code),0 }; x86emu_init(&cpu,borrowed,0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[3], left_ymm, sizeof(left_ymm)); memset(cpu.vector_registers[1] + 32u, 0xA5, 32u); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(cpu.vector_registers[1], expected_ymm, sizeof(expected_ymm)) == 0); for (unsigned i = 32u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    uint8_t zero_code[6] = { 0xC4,0xE3,0x61,0x0F,0xCA,0x20 }; x86emu_memory zero_memory = { zero_code,sizeof(zero_code),0 }; x86emu_init(&cpu,zero_memory,0); memcpy(cpu.vector_registers[3], left, sizeof(left)); memcpy(cpu.vector_registers[2], right, sizeof(right)); assert(x86emu_step(&cpu) == X86EMU_OK); for (unsigned i = 0; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
}

static void test_pinsextr(void)
{
    static const uint8_t pinsrb_code[] = { 0x66,0x0F,0x3A,0x20,0xCA,0x1F };
    static const uint8_t pinsrw_code[] = { 0x66,0x0F,0xC4,0xCA,0x0F };
    static const uint8_t pinsrd_code[] = { 0x66,0x0F,0x3A,0x22,0xCA,0x07 };
    static const uint8_t pinsrq_code[] = { 0x66,0x48,0x0F,0x3A,0x22,0xCA,0x03 };
    static const uint8_t pextrb_code[] = { 0x66,0x0F,0x3A,0x14,0xCA,0x1F };
    static const uint8_t pextrw_code[] = { 0x66,0x0F,0xC5,0xCA,0x0F };
    static const uint8_t pextrd_code[] = { 0x66,0x0F,0x3A,0x16,0xCA,0x07 };
    static const uint8_t pextrq_code[] = { 0x66,0x48,0x0F,0x3A,0x16,0xCA,0x03 };
    x86emu_cpu cpu;
    uint8_t vector[16];
    for (unsigned i = 0; i < 16u; ++i) vector[i] = (uint8_t)(0xA0u + i);

    x86emu_memory memory = { (uint8_t *)pinsrb_code, sizeof(pinsrb_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], vector, 16); memset(cpu.vector_registers[1] + 16u, 0xA5, 48u); cpu.registers[X86EMU_RDX] = UINT64_C(0x123456789ABCDE7E); assert(x86emu_step(&cpu) == X86EMU_OK); assert(cpu.vector_registers[1][15] == 0x7Eu); for (unsigned i = 0; i < 15u; ++i) assert(cpu.vector_registers[1][i] == vector[i]); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0xA5u);
    memory = (x86emu_memory){ (uint8_t *)pinsrw_code, sizeof(pinsrw_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], vector, 16); cpu.registers[X86EMU_RDX] = UINT64_C(0x123456789ABC7E7E); assert(x86emu_step(&cpu) == X86EMU_OK); uint16_t pinsrw_result; memcpy(&pinsrw_result, cpu.vector_registers[1] + 14u, sizeof(pinsrw_result)); assert(pinsrw_result == UINT16_C(0x7E7E));
    memory = (x86emu_memory){ (uint8_t *)pinsrd_code, sizeof(pinsrd_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], vector, 16); cpu.registers[X86EMU_RDX] = UINT64_C(0x12345678CAFEBABE); assert(x86emu_step(&cpu) == X86EMU_OK); uint32_t pinsrd_result; memcpy(&pinsrd_result, cpu.vector_registers[1] + 12u, sizeof(pinsrd_result)); assert(pinsrd_result == UINT32_C(0xCAFEBABE));
    memory = (x86emu_memory){ (uint8_t *)pinsrq_code, sizeof(pinsrq_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], vector, 16); cpu.registers[X86EMU_RDX] = UINT64_C(0x1122334455667788); assert(x86emu_step(&cpu) == X86EMU_OK); uint64_t pinsrq_result; memcpy(&pinsrq_result, cpu.vector_registers[1] + 8u, sizeof(pinsrq_result)); assert(pinsrq_result == UINT64_C(0x1122334455667788));

    memory = (x86emu_memory){ (uint8_t *)pextrb_code, sizeof(pextrb_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], vector, 16); cpu.registers[X86EMU_RDX] = UINT64_MAX; assert(x86emu_step(&cpu) == X86EMU_OK); assert(cpu.registers[X86EMU_RDX] == 0xAFu);
    memory = (x86emu_memory){ (uint8_t *)pextrw_code, sizeof(pextrw_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], vector, 16); cpu.registers[X86EMU_RDX] = UINT64_MAX; assert(x86emu_step(&cpu) == X86EMU_OK); assert(cpu.registers[X86EMU_RDX] == UINT64_C(0xAFAE));
    memory = (x86emu_memory){ (uint8_t *)pextrd_code, sizeof(pextrd_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], vector, 16); cpu.registers[X86EMU_RDX] = UINT64_MAX; assert(x86emu_step(&cpu) == X86EMU_OK); assert(cpu.registers[X86EMU_RDX] == UINT64_C(0xAFAEADAC));
    memory = (x86emu_memory){ (uint8_t *)pextrq_code, sizeof(pextrq_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[1], vector, 16); cpu.registers[X86EMU_RDX] = 0; assert(x86emu_step(&cpu) == X86EMU_OK); assert(cpu.registers[X86EMU_RDX] == UINT64_C(0xAFAEADACABAAA9A8));

    uint8_t vpinsrd_code[64] = { 0xC4,0xE3,0x61,0x22,0x08,0x07 }; memcpy(vpinsrd_code + 32u, "\xBE\xBA\xFE\xCA", 4); memory = (x86emu_memory){ vpinsrd_code, sizeof(vpinsrd_code), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[3], vector, 16); memset(cpu.vector_registers[1] + 16u, 0xA5, 48u); assert(x86emu_step(&cpu) == X86EMU_OK); uint32_t vpinsrd_result; memcpy(&vpinsrd_result, cpu.vector_registers[1] + 12u, sizeof(vpinsrd_result)); assert(vpinsrd_result == UINT32_C(0xCAFEBABE)); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
    uint8_t vpextrd_code[64] = { 0xC4,0xE3,0x01,0x16,0x08,0x07 }; memory = (x86emu_memory){ vpextrd_code, sizeof(vpextrd_code), 0 }; x86emu_init(&cpu, memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[1], vector, 16); assert(x86emu_step(&cpu) == X86EMU_OK); assert(memcmp(vpextrd_code + 32u, "\xAC\xAD\xAE\xAF", 4) == 0);
    x86emu_memory short_memory = { vpextrd_code, 35u, 0 }; x86emu_init(&cpu, short_memory, 0); cpu.registers[X86EMU_RAX] = 32u; memcpy(cpu.vector_registers[1], vector, 16); assert(x86emu_step(&cpu) == X86EMU_ERR_MEMORY); assert(cpu.rip == 0u);
    uint8_t vpinsrb_code[6] = { 0xC4,0xE3,0x61,0x20,0xCA,0x0F }; memory = (x86emu_memory){ vpinsrb_code, sizeof(vpinsrb_code), 0 }; x86emu_init(&cpu, memory, 0); memcpy(cpu.vector_registers[3], vector, 16); cpu.registers[X86EMU_RDX] = 0x5Au; assert(x86emu_step(&cpu) == X86EMU_OK); assert(cpu.vector_registers[1][15] == 0x5Au); for (unsigned i = 16u; i < 64u; ++i) assert(cpu.vector_registers[1][i] == 0u);
}

static void test_xop_unsupported(void)
{
    uint8_t code[5] = { 0x8F, 0xE8, 0x78, 0xA2, 0xC0 };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    x86emu_init(&cpu, memory, 0);
    assert(x86emu_step(&cpu) == X86EMU_ERR_UNSUPPORTED);
    assert(cpu.rip == 0u);
    assert(cpu.steps == 0u);
}

static void test_breakpoint(void)
{
    uint8_t code[] = { 0x90 };
    x86emu_cpu cpu;
    x86emu_memory memory = { code, sizeof(code), 0 };
    x86emu_init(&cpu, memory, 0);
    assert(x86emu_add_breakpoint(&cpu, 0));
    assert(x86emu_step(&cpu) == X86EMU_ERR_BREAKPOINT);
    assert(x86emu_remove_breakpoint(&cpu, 0));
    assert(x86emu_step(&cpu) == X86EMU_OK);
}

int main(void)
{
    test_add_and_flags();
    test_call_return();
    test_vector_execution();
    test_rotates();
    test_double_shifts();
    test_arithmetic_and_compare();
    test_movd_movq();
    test_movq_xmm_forms();
    test_partial_xmm_moves();
    test_vmov_partial_moves();
    test_vmovd_vmovq();
    test_sse2_integer();
    test_packed_shifts();
    test_packed_multiply();
    test_pmulld();
    test_vpacked_multiply();
    test_vpacked_shifts();
    test_legacy_sse41_minmax();
    test_variable_shifts();
    test_saturating_and_minmax();
    test_unpack_pack();
    test_v_unpack_pack();
    test_v_compare_greater();
    test_ptest();
    test_mask_extraction();
    test_legacy_compare_greater();
    test_scalar_float_arithmetic();
    test_scalar_float_mul_div();
    test_scalar_float_memory();
    test_scalar_moves();
    test_scalar_float_minmax();
    test_packed_float_compare();
    test_packed_float_minmax();
    test_packed_float_arithmetic();
    test_packed_shuffles();
    test_pshufb();
    test_v_extended_minmax();
    test_v_saturating_and_minmax();
    test_movbe();
    test_bit_instructions();
    test_cmpxchg16b();
    test_cmpxchg8b();
    test_string_instructions();
    test_flags_and_system_hook();
    test_pabs();
    test_psign();
    test_phadd();
    test_phsub();
    test_pmaddubsw();
    test_pmaddwd();
    test_pmuldq();
    test_pmov();
    test_movntdq();
    test_lddqu();
    test_movntdqa();
    test_legacy_movdqa_movdqu();
    test_vmovdqa_vmovdqu();
    test_vpblendd();
    test_blend_immediate_legacy();
    test_blendv_fp();
    test_pblendvb();
    test_pblendw();
    test_palignr();
    test_pinsextr();
    test_phminposuw();
    test_xop_unsupported();
    test_breakpoint();
    puts("x86emu tests passed");
    return 0;
}
