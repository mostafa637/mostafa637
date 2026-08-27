#include "x86emu.h"

#include <math.h>
#include <string.h>

#define X86EMU_MAX_BREAKPOINTS 32
#define X86EMU_STACK_WIDTH 8

static uint64_t width_mask(unsigned bits)
{
    if (bits >= 64) return UINT64_MAX;
    return (UINT64_C(1) << bits) - 1;
}

static bool register_location(x86asm_register reg, unsigned *index,
                              unsigned *width, unsigned *shift)
{
    if (reg >= X86ASM_REG_AL && reg <= X86ASM_REG_R15B) {
        *width = 8;
        if (reg >= X86ASM_REG_AH && reg <= X86ASM_REG_BH) {
            *index = (unsigned)(reg - X86ASM_REG_AH);
            *shift = 8;
        } else if (reg >= X86ASM_REG_SPL) {
            *index = (unsigned)(reg - X86ASM_REG_SPL) + 4u;
            *shift = 0;
        } else {
            *index = (unsigned)(reg - X86ASM_REG_AL);
            *shift = 0;
        }
        return *index < 16;
    }
    if (reg >= X86ASM_REG_AX && reg <= X86ASM_REG_R15W) {
        *index = (unsigned)(reg - X86ASM_REG_AX);
        *width = 16;
        *shift = 0;
        return *index < 16;
    }
    if (reg >= X86ASM_REG_EAX && reg <= X86ASM_REG_R15D) {
        *index = (unsigned)(reg - X86ASM_REG_EAX);
        *width = 32;
        *shift = 0;
        return *index < 16;
    }
    if (reg >= X86ASM_REG_RAX && reg <= X86ASM_REG_R15) {
        *index = (unsigned)(reg - X86ASM_REG_RAX);
        *width = 64;
        *shift = 0;
        return *index < 16;
    }
    return false;
}

static bool memory_address(const x86emu_cpu *cpu,
                          const x86asm_instruction *instruction,
                          const x86asm_memory *memory, uint64_t *address)
{
    uint64_t result = (uint64_t)memory->displacement;
    unsigned index;
    unsigned width;
    unsigned shift;

    if (memory->base == X86ASM_REG_RIP || memory->base == X86ASM_REG_EIP) {
        result += cpu->rip + (uint64_t)instruction->length;
    } else if (register_location(memory->base, &index, &width, &shift)) {
        (void)width;
        (void)shift;
        result += cpu->registers[index];
    }
    if (register_location(memory->index, &index, &width, &shift)) {
        (void)width;
        (void)shift;
        result += cpu->registers[index] * (memory->scale == 0 ? 1u : memory->scale);
    }
    *address = result;
    return true;
}

static bool memory_offset(const x86emu_memory *memory, uint64_t address,
                          size_t bytes, size_t *offset)
{
    if (memory->data == NULL || address < memory->base_address) return false;
    uint64_t relative = address - memory->base_address;
    if (relative > SIZE_MAX || (size_t)relative > memory->size) return false;
    if (bytes > memory->size - (size_t)relative) return false;
    *offset = (size_t)relative;
    return true;
}

static bool read_memory(const x86emu_cpu *cpu, uint64_t address,
                        unsigned width, uint64_t *value)
{
    size_t offset;
    unsigned bytes = width / 8;
    if (bytes == 0 || bytes > 8 || !memory_offset(&cpu->memory, address, bytes, &offset)) {
        return false;
    }
    uint64_t result = 0;
    for (unsigned i = 0; i < bytes; ++i) result |= (uint64_t)cpu->memory.data[offset + i] << (8u * i);
    *value = result;
    return true;
}

static bool write_memory(x86emu_cpu *cpu, uint64_t address,
                         unsigned width, uint64_t value)
{
    size_t offset;
    unsigned bytes = width / 8;
    if (bytes == 0 || bytes > 8 || !memory_offset(&cpu->memory, address, bytes, &offset)) {
        return false;
    }
    for (unsigned i = 0; i < bytes; ++i) cpu->memory.data[offset + i] = (uint8_t)(value >> (8u * i));
    return true;
}

static bool read_register(const x86emu_cpu *cpu, x86asm_register reg,
                          uint64_t *value)
{
    unsigned index;
    unsigned width;
    unsigned shift;
    if (!register_location(reg, &index, &width, &shift) || index >= 16) return false;
    *value = (cpu->registers[index] >> shift) & width_mask(width);
    return true;
}

static bool write_register(x86emu_cpu *cpu, x86asm_register reg, uint64_t value)
{
    unsigned index;
    unsigned width;
    unsigned shift;
    if (!register_location(reg, &index, &width, &shift) || index >= 16) return false;
    value &= width_mask(width);
    if (width == 32) cpu->registers[index] = value;
    else if (width == 64) cpu->registers[index] = value;
    else if (width == 16) cpu->registers[index] = (cpu->registers[index] & ~UINT64_C(0xffff)) | value;
    else if (width == 8) {
        uint64_t mask = UINT64_C(0xff) << shift;
        cpu->registers[index] = (cpu->registers[index] & ~mask) | (value << shift);
    }
    return true;
}

static bool vector_location(x86asm_register reg, unsigned *index, unsigned *bytes)
{
    if (reg >= X86ASM_REG_XMM0 && reg <= X86ASM_REG_XMM15) {
        *index = (unsigned)(reg - X86ASM_REG_XMM0);
        *bytes = 16;
        return true;
    }
    if (reg >= X86ASM_REG_YMM0 && reg <= X86ASM_REG_YMM15) {
        *index = (unsigned)(reg - X86ASM_REG_YMM0);
        *bytes = 32;
        return true;
    }
    if (reg >= X86ASM_REG_ZMM0 && reg <= X86ASM_REG_ZMM31) {
        *index = (unsigned)(reg - X86ASM_REG_ZMM0);
        *bytes = 64;
        return true;
    }
    return false;
}

static bool argument_is_vector(const x86emu_cpu *cpu, const x86asm_argument *argument)
{
    unsigned index;
    unsigned bytes;
    (void)cpu;
    return argument->kind == X86ASM_ARG_REGISTER && vector_location(argument->value.reg, &index, &bytes);
}

static bool read_vector_argument(const x86emu_cpu *cpu,
                                 const x86asm_instruction *instruction,
                                 const x86asm_argument *argument, unsigned bytes,
                                 uint8_t *value)
{
    if (argument->kind == X86ASM_ARG_REGISTER) {
        unsigned index;
        unsigned register_bytes;
        if (!vector_location(argument->value.reg, &index, &register_bytes) || bytes > register_bytes) return false;
        memcpy(value, cpu->vector_registers[index], bytes);
        return true;
    }
    if (argument->kind == X86ASM_ARG_MEMORY) {
        uint64_t address;
        size_t offset;
        if (!memory_address(cpu, instruction, &argument->value.memory, &address) ||
            !memory_offset(&cpu->memory, address, bytes, &offset)) return false;
        memcpy(value, cpu->memory.data + offset, bytes);
        return true;
    }
    return false;
}

static bool read_scalar_vector_argument(const x86emu_cpu *cpu,
                                        const x86asm_instruction *instruction,
                                        const x86asm_argument *argument, unsigned bytes,
                                        uint8_t *value)
{
    if (argument->kind == X86ASM_ARG_REGISTER) {
        unsigned index;
        unsigned register_bytes;
        if (!vector_location(argument->value.reg, &index, &register_bytes) || bytes > register_bytes) return false;
        memcpy(value, cpu->vector_registers[index], bytes);
        return true;
    }
    if (argument->kind == X86ASM_ARG_MEMORY) {
        uint64_t address;
        size_t offset;
        if (!memory_address(cpu, instruction, &argument->value.memory, &address) ||
            !memory_offset(&cpu->memory, address, bytes, &offset)) return false;
        memcpy(value, cpu->memory.data + offset, bytes);
        return true;
    }
    return false;
}

static void zero_vector_upper_width(x86emu_cpu *cpu, const x86asm_argument *argument, unsigned written_bytes)
{
    unsigned index;
    unsigned register_bytes;
    if (argument->kind == X86ASM_ARG_REGISTER && vector_location(argument->value.reg, &index, &register_bytes) && written_bytes <= 64u) {
        memset(cpu->vector_registers[index] + written_bytes, 0, 64u - written_bytes);
    }
}

static void zero_vector_upper_128(x86emu_cpu *cpu, const x86asm_argument *argument)
{
    unsigned index;
    if (argument->kind == X86ASM_ARG_REGISTER && argument->value.reg >= X86ASM_REG_XMM0 && argument->value.reg <= X86ASM_REG_XMM15) {
        index = (unsigned)(argument->value.reg - X86ASM_REG_XMM0);
        memset(cpu->vector_registers[index] + 16u, 0, 48u);
    }
}

static bool write_vector_argument(x86emu_cpu *cpu,
                                  const x86asm_instruction *instruction,
                                  const x86asm_argument *argument, unsigned bytes,
                                  const uint8_t *value)
{
    if (argument->kind == X86ASM_ARG_REGISTER) {
        unsigned index;
        unsigned register_bytes;
        if (!vector_location(argument->value.reg, &index, &register_bytes) || bytes > register_bytes) return false;
        memcpy(cpu->vector_registers[index], value, bytes);
        if (bytes < register_bytes) memset(cpu->vector_registers[index] + bytes, 0, register_bytes - bytes);
        return true;
    }
    if (argument->kind == X86ASM_ARG_MEMORY) {
        uint64_t address;
        size_t offset;
        if (!memory_address(cpu, instruction, &argument->value.memory, &address) ||
            !memory_offset(&cpu->memory, address, bytes, &offset)) return false;
        memcpy(cpu->memory.data + offset, value, bytes);
        return true;
    }
    return false;
}

static void vector_bitwise(uint8_t *out, const uint8_t *left, const uint8_t *right,
                            unsigned bytes, x86asm_opcode opcode)
{
    for (unsigned i = 0; i < bytes; ++i) {
        if (opcode == X86ASM_OP_VAND) out[i] = left[i] & right[i];
        else if (opcode == X86ASM_OP_VANDN) out[i] = (uint8_t)(~left[i]) & right[i];
        else if (opcode == X86ASM_OP_VOR) out[i] = left[i] | right[i];
        else out[i] = left[i] ^ right[i];
    }
}

static void vector_add_i32(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 4) {
        uint32_t a;
        uint32_t b;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        uint32_t sum = a + b;
        memcpy(out + i, &sum, sizeof(sum));
    }
}

static void vector_sub_i32(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 4) {
        uint32_t a;
        uint32_t b;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        uint32_t difference = a - b;
        memcpy(out + i, &difference, sizeof(difference));
    }
}

static void vector_add_i8(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; ++i) out[i] = (uint8_t)(left[i] + right[i]);
}

static void vector_sub_i8(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; ++i) out[i] = (uint8_t)(left[i] - right[i]);
}

static void vector_add_i16(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 2) {
        uint16_t a;
        uint16_t b;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        uint16_t sum = (uint16_t)(a + b);
        memcpy(out + i, &sum, sizeof(sum));
    }
}

static void vector_sub_i16(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 2) {
        uint16_t a;
        uint16_t b;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        uint16_t difference = (uint16_t)(a - b);
        memcpy(out + i, &difference, sizeof(difference));
    }
}

static void vector_add_i64(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 8) {
        uint64_t a;
        uint64_t b;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        uint64_t sum = a + b;
        memcpy(out + i, &sum, sizeof(sum));
    }
}

static void vector_sub_i64(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 8) {
        uint64_t a;
        uint64_t b;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        uint64_t difference = a - b;
        memcpy(out + i, &difference, sizeof(difference));
    }
}

static uint64_t vector_load_le(const uint8_t *bytes, unsigned count);
static void vector_store_le(uint8_t *bytes, unsigned count, uint64_t value);

static void vector_mul_words(uint8_t *out, const uint8_t *left, const uint8_t *right,
                             unsigned bytes, x86asm_opcode opcode)
{
    for (unsigned i = 0; i < bytes; i += 2u) {
        uint16_t a = (uint16_t)vector_load_le(left + i, 2);
        uint16_t b = (uint16_t)vector_load_le(right + i, 2);
        uint16_t product;
        if (opcode == X86ASM_OP_PMULLW) {
            product = (uint16_t)((uint32_t)a * (uint32_t)b);
        } else if (opcode == X86ASM_OP_PMULHW) {
            int32_t signed_product = (int32_t)(int16_t)a * (int32_t)(int16_t)b;
            product = (uint16_t)(((uint32_t)signed_product) >> 16);
        } else {
            product = (uint16_t)(((uint32_t)a * (uint32_t)b) >> 16);
        }
        vector_store_le(out + i, 2, product);
    }
}

static void vector_mul_signed_dwords_low(uint8_t *out, const uint8_t *left,
                                           const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 4u) {
        int64_t a = (int64_t)(int32_t)(uint32_t)vector_load_le(left + i, 4u);
        int64_t b = (int64_t)(int32_t)(uint32_t)vector_load_le(right + i, 4u);
        uint32_t low = (uint32_t)(a * b);
        vector_store_le(out + i, 4u, (uint64_t)low);
    }
}

static void vector_mul_u32_even(uint8_t *out, const uint8_t *left,
                                const uint8_t *right, unsigned bytes)
{
    memset(out, 0, bytes);
    for (unsigned i = 0; i < bytes; i += 8u) {
        uint64_t a = vector_load_le(left + i, 4);
        uint64_t b = vector_load_le(right + i, 4);
        vector_store_le(out + i, 8, a * b);
    }
}

static void vector_saturating_add_sub(uint8_t *out, const uint8_t *left, const uint8_t *right,
                                       unsigned bytes, unsigned element_bytes,
                                       bool signed_mode, bool subtract)
{
    for (unsigned i = 0; i < bytes; i += element_bytes) {
        uint32_t a = (uint32_t)vector_load_le(left + i, element_bytes);
        uint32_t b = (uint32_t)vector_load_le(right + i, element_bytes);
        uint32_t unsigned_result;
        if (!signed_mode) {
            uint32_t maximum = element_bytes == 1u ? UINT32_C(255) : UINT32_C(65535);
            if (subtract) unsigned_result = a < b ? 0u : a - b;
            else unsigned_result = a + b > maximum ? maximum : a + b;
        } else {
            int32_t signed_a = element_bytes == 1u ? (int32_t)(int8_t)a : (int32_t)(int16_t)a;
            int32_t signed_b = element_bytes == 1u ? (int32_t)(int8_t)b : (int32_t)(int16_t)b;
            int32_t signed_result = subtract ? signed_a - signed_b : signed_a + signed_b;
            int32_t minimum = element_bytes == 1u ? -128 : -32768;
            int32_t maximum = element_bytes == 1u ? 127 : 32767;
            if (signed_result < minimum) signed_result = minimum;
            if (signed_result > maximum) signed_result = maximum;
            unsigned_result = (uint32_t)(uint16_t)(int16_t)signed_result;
        }
        vector_store_le(out + i, element_bytes, unsigned_result);
    }
}

static void vector_abs_signed(uint8_t *out, const uint8_t *input, unsigned bytes, unsigned element_bytes)
{
    uint64_t mask = (UINT64_C(1) << (element_bytes * 8u)) - UINT64_C(1);
    uint64_t sign = UINT64_C(1) << (element_bytes * 8u - 1u);
    for (unsigned i = 0; i < bytes; i += element_bytes) {
        uint64_t value = vector_load_le(input + i, element_bytes) & mask;
        if ((value & sign) != 0u) value = (~value + UINT64_C(1)) & mask;
        vector_store_le(out + i, element_bytes, value);
    }
}

static void vector_sign(uint8_t *out, const uint8_t *input, const uint8_t *control, unsigned bytes, unsigned element_bytes)
{
    uint64_t mask = (UINT64_C(1) << (element_bytes * 8u)) - UINT64_C(1);
    uint64_t sign = UINT64_C(1) << (element_bytes * 8u - 1u);
    for (unsigned i = 0; i < bytes; i += element_bytes) {
        uint64_t value = vector_load_le(input + i, element_bytes) & mask;
        uint64_t selector = vector_load_le(control + i, element_bytes) & mask;
        uint64_t result = selector == 0u ? 0u : ((selector & sign) != 0u ? ((~value + UINT64_C(1)) & mask) : value);
        vector_store_le(out + i, element_bytes, result);
    }
}

static void vector_horizontal_add(uint8_t *out, const uint8_t *left, const uint8_t *right,
                                   unsigned bytes, unsigned element_bytes, bool saturate)
{
    for (unsigned lane = 0; lane < bytes; lane += 16u) {
        unsigned elements = 16u / element_bytes;
        unsigned outputs = elements / 2u;
        for (unsigned source_index = 0; source_index < 2u; ++source_index) {
            const uint8_t *source = source_index == 0u ? left + lane : right + lane;
            unsigned output_base = lane + source_index * outputs * element_bytes;
            for (unsigned i = 0; i < outputs; ++i) {
                uint64_t raw_a = vector_load_le(source + (2u * i) * element_bytes, element_bytes);
                uint64_t raw_b = vector_load_le(source + (2u * i + 1u) * element_bytes, element_bytes);
                int64_t a = element_bytes == 2u ? (int64_t)(int16_t)(uint16_t)raw_a : (int64_t)(int32_t)(uint32_t)raw_a;
                int64_t b = element_bytes == 2u ? (int64_t)(int16_t)(uint16_t)raw_b : (int64_t)(int32_t)(uint32_t)raw_b;
                int64_t sum = a + b;
                if (saturate) {
                    if (sum > INT64_C(32767)) sum = INT64_C(32767);
                    if (sum < -INT64_C(32768)) sum = -INT64_C(32768);
                }
                uint64_t encoded = element_bytes == 2u ? (uint64_t)(uint16_t)(int16_t)sum : (uint64_t)(uint32_t)(int32_t)sum;
                vector_store_le(out + output_base + i * element_bytes, element_bytes, encoded);
            }
        }
    }
}

static void vector_horizontal_sub(uint8_t *out, const uint8_t *left, const uint8_t *right,
                                   unsigned bytes, unsigned element_bytes, bool saturate)
{
    for (unsigned lane = 0; lane < bytes; lane += 16u) {
        unsigned elements = 16u / element_bytes;
        unsigned outputs = elements / 2u;
        for (unsigned source_index = 0; source_index < 2u; ++source_index) {
            const uint8_t *source = source_index == 0u ? left + lane : right + lane;
            unsigned output_base = lane + source_index * outputs * element_bytes;
            for (unsigned i = 0; i < outputs; ++i) {
                uint64_t raw_a = vector_load_le(source + (2u * i) * element_bytes, element_bytes);
                uint64_t raw_b = vector_load_le(source + (2u * i + 1u) * element_bytes, element_bytes);
                int64_t a = element_bytes == 2u ? (int64_t)(int16_t)(uint16_t)raw_a : (int64_t)(int32_t)(uint32_t)raw_a;
                int64_t b = element_bytes == 2u ? (int64_t)(int16_t)(uint16_t)raw_b : (int64_t)(int32_t)(uint32_t)raw_b;
                int64_t difference = a - b;
                if (saturate) {
                    if (difference > INT64_C(32767)) difference = INT64_C(32767);
                    if (difference < -INT64_C(32768)) difference = -INT64_C(32768);
                }
                uint64_t encoded = element_bytes == 2u ? (uint64_t)(uint16_t)(int16_t)difference : (uint64_t)(uint32_t)(int32_t)difference;
                vector_store_le(out + output_base + i * element_bytes, element_bytes, encoded);
            }
        }
    }
}

static void vector_madd_unsigned_signed_bytes(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 2u) {
        int32_t sum = (int32_t)left[i] * (int32_t)(int8_t)right[i] + (int32_t)left[i + 1u] * (int32_t)(int8_t)right[i + 1u];
        if (sum > 32767) sum = 32767;
        if (sum < -32768) sum = -32768;
        vector_store_le(out + i, 2u, (uint64_t)(uint16_t)(int16_t)sum);
    }
}

static void vector_madd_signed_words(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 4u) {
        int64_t a = (int64_t)(int16_t)(uint16_t)vector_load_le(left + i, 2u);
        int64_t b = (int64_t)(int16_t)(uint16_t)vector_load_le(left + i + 2u, 2u);
        int64_t c = (int64_t)(int16_t)(uint16_t)vector_load_le(right + i, 2u);
        int64_t d = (int64_t)(int16_t)(uint16_t)vector_load_le(right + i + 2u, 2u);
        uint32_t result = (uint32_t)(a * c + b * d);
        vector_store_le(out + i, 4u, (uint64_t)result);
    }
}

static void vector_horizontal_minpos_unsigned_words(uint8_t *out, const uint8_t *input)
{
    uint16_t minimum = (uint16_t)vector_load_le(input, 2u);
    unsigned index = 0u;
    for (unsigned i = 1u; i < 8u; ++i) {
        uint16_t value = (uint16_t)vector_load_le(input + i * 2u, 2u);
        if (value < minimum) { minimum = value; index = i; }
    }
    memset(out, 0, 16u);
    vector_store_le(out, 2u, minimum);
    vector_store_le(out + 2u, 2u, (uint64_t)index);
}

static void vector_blend_bytes(uint8_t *out, const uint8_t *left, const uint8_t *right, const uint8_t *mask, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; ++i) out[i] = (mask[i] & UINT8_C(0x80)) != 0u ? right[i] : left[i];
}

static void vector_blend_masked_elements(uint8_t *out, const uint8_t *left,
                                         const uint8_t *right, const uint8_t *mask,
                                         unsigned bytes, unsigned element_bytes)
{
    for (unsigned i = 0; i < bytes; i += element_bytes) {
        const uint8_t *source = (mask[i + element_bytes - 1u] & UINT8_C(0x80)) != 0u ? right : left;
        memcpy(out + i, source + i, element_bytes);
    }
}

static void vector_blend_words(uint8_t *out, const uint8_t *left, const uint8_t *right,
                               unsigned bytes, uint8_t immediate)
{
    memcpy(out, left, bytes);
    for (unsigned lane = 0; lane < bytes; lane += 16u) {
        for (unsigned word = 0; word < 8u; ++word) {
            if ((immediate & (uint8_t)(UINT8_C(1) << word)) != 0u) memcpy(out + lane + 2u * word, right + lane + 2u * word, 2u);
        }
    }
}

static void vector_align_right(uint8_t *out, const uint8_t *left, const uint8_t *right,
                               unsigned bytes, uint8_t immediate)
{
    for (unsigned lane = 0; lane < bytes; lane += 16u) {
        for (unsigned i = 0; i < 16u; ++i) {
            unsigned source_index = (unsigned)immediate + i;
            if (source_index < 16u) out[lane + i] = right[lane + source_index];
            else if (source_index < 32u) out[lane + i] = left[lane + source_index - 16u];
            else out[lane + i] = 0u;
        }
    }
}

static void vector_extend_integer_elements(uint8_t *out, const uint8_t *input, unsigned output_bytes, unsigned source_bytes, unsigned destination_bytes, bool sign_extend)
{
    unsigned count = output_bytes / destination_bytes;
    for (unsigned i = 0; i < count; ++i) {
        uint64_t raw = vector_load_le(input + i * source_bytes, source_bytes);
        uint64_t value;
        if (sign_extend && source_bytes == 1u) value = (uint64_t)(int64_t)(int8_t)raw;
        else if (sign_extend && source_bytes == 2u) value = (uint64_t)(int64_t)(int16_t)(uint16_t)raw;
        else if (sign_extend && source_bytes == 4u) value = (uint64_t)(int64_t)(int32_t)(uint32_t)raw;
        else value = raw;
        vector_store_le(out + i * destination_bytes, destination_bytes, value);
    }
}

static bool vector_extension_parameters(x86asm_opcode opcode, unsigned *source_bytes, unsigned *destination_bytes, bool *sign_extend)
{
    switch (opcode) {
    case X86ASM_OP_PMOVSXBW: case X86ASM_OP_VPMOVSXBW: *source_bytes = 1u; *destination_bytes = 2u; *sign_extend = true; return true;
    case X86ASM_OP_PMOVSXBD: case X86ASM_OP_VPMOVSXBD: *source_bytes = 1u; *destination_bytes = 4u; *sign_extend = true; return true;
    case X86ASM_OP_PMOVSXBQ: case X86ASM_OP_VPMOVSXBQ: *source_bytes = 1u; *destination_bytes = 8u; *sign_extend = true; return true;
    case X86ASM_OP_PMOVSXWD: case X86ASM_OP_VPMOVSXWD: *source_bytes = 2u; *destination_bytes = 4u; *sign_extend = true; return true;
    case X86ASM_OP_PMOVSXWQ: case X86ASM_OP_VPMOVSXWQ: *source_bytes = 2u; *destination_bytes = 8u; *sign_extend = true; return true;
    case X86ASM_OP_PMOVSXDQ: case X86ASM_OP_VPMOVSXDQ: *source_bytes = 4u; *destination_bytes = 8u; *sign_extend = true; return true;
    case X86ASM_OP_PMOVZXBW: case X86ASM_OP_VPMOVZXBW: *source_bytes = 1u; *destination_bytes = 2u; *sign_extend = false; return true;
    case X86ASM_OP_PMOVZXBD: case X86ASM_OP_VPMOVZXBD: *source_bytes = 1u; *destination_bytes = 4u; *sign_extend = false; return true;
    case X86ASM_OP_PMOVZXBQ: case X86ASM_OP_VPMOVZXBQ: *source_bytes = 1u; *destination_bytes = 8u; *sign_extend = false; return true;
    case X86ASM_OP_PMOVZXWD: case X86ASM_OP_VPMOVZXWD: *source_bytes = 2u; *destination_bytes = 4u; *sign_extend = false; return true;
    case X86ASM_OP_PMOVZXWQ: case X86ASM_OP_VPMOVZXWQ: *source_bytes = 2u; *destination_bytes = 8u; *sign_extend = false; return true;
    case X86ASM_OP_PMOVZXDQ: case X86ASM_OP_VPMOVZXDQ: *source_bytes = 4u; *destination_bytes = 8u; *sign_extend = false; return true;
    default: return false;
    }
}

static void vector_multiply_even_signed_dwords(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 8u) {
        int64_t a = (int64_t)(int32_t)(uint32_t)vector_load_le(left + i, 4u);
        int64_t b = (int64_t)(int32_t)(uint32_t)vector_load_le(right + i, 4u);
        int64_t product = a * b;
        vector_store_le(out + i, 8u, (uint64_t)product);
    }
}

static void vector_minmax(uint8_t *out, const uint8_t *left, const uint8_t *right,
                          unsigned bytes, unsigned element_bytes, bool signed_mode, bool maximum)
{
    for (unsigned i = 0; i < bytes; i += element_bytes) {
        uint64_t a = vector_load_le(left + i, element_bytes);
        uint64_t b = vector_load_le(right + i, element_bytes);
        bool choose_right;
        if (signed_mode) {
            int64_t signed_a;
            int64_t signed_b;
            if (element_bytes == 1u) {
                signed_a = (int64_t)(int8_t)a;
                signed_b = (int64_t)(int8_t)b;
            } else if (element_bytes == 2u) {
                signed_a = (int64_t)(int16_t)a;
                signed_b = (int64_t)(int16_t)b;
            } else if (element_bytes == 4u) {
                signed_a = (int64_t)(int32_t)(uint32_t)a;
                signed_b = (int64_t)(int32_t)(uint32_t)b;
            } else {
                signed_a = (int64_t)a;
                signed_b = (int64_t)b;
            }
            choose_right = maximum ? signed_b > signed_a : signed_b < signed_a;
        } else choose_right = maximum ? b > a : b < a;
        vector_store_le(out + i, element_bytes, choose_right ? b : a);
    }
}

static void vector_average(uint8_t *out, const uint8_t *left, const uint8_t *right,
                           unsigned bytes, unsigned element_bytes)
{
    for (unsigned i = 0; i < bytes; i += element_bytes) {
        uint32_t a = (uint32_t)vector_load_le(left + i, element_bytes);
        uint32_t b = (uint32_t)vector_load_le(right + i, element_bytes);
        vector_store_le(out + i, element_bytes, (a + b + 1u) / 2u);
    }
}

static void vector_sad_bytes(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    memset(out, 0, bytes);
    for (unsigned lane = 0; lane < bytes; lane += 8u) {
        uint64_t sum = 0;
        for (unsigned i = 0; i < 8u; ++i) {
            uint8_t a = left[lane + i];
            uint8_t b = right[lane + i];
            sum += a > b ? (uint64_t)(a - b) : (uint64_t)(b - a);
        }
        vector_store_le(out + lane, 8, sum);
    }
}

static void vector_shuffle_bytes(uint8_t *out, const uint8_t *source,
                                 const uint8_t *control, unsigned bytes)
{
    for (unsigned lane = 0; lane < bytes; lane += 16u) {
        for (unsigned i = 0; i < 16u; ++i) {
            uint8_t selector = control[lane + i];
            out[lane + i] = (selector & 0x80u) != 0 ? 0u : source[lane + (selector & 0x0Fu)];
        }
    }
}

static void vector_shuffle_words(uint8_t *out, const uint8_t *source,
                                 unsigned bytes, uint8_t immediate, bool high_half)
{
    memcpy(out, source, bytes);
    for (unsigned lane = 0; lane < bytes; lane += 16u) {
        unsigned base = lane + (high_half ? 8u : 0u);
        for (unsigned i = 0; i < 4u; ++i) {
            unsigned selected = ((unsigned)immediate >> (2u * i)) & 3u;
            memcpy(out + base + 2u * i, source + base + 2u * selected, 2u);
        }
    }
}

static uint64_t vector_sign_mask(const uint8_t *source, unsigned bytes, unsigned element_bytes)
{
    uint64_t mask = 0;
    unsigned elements = bytes / element_bytes;
    for (unsigned i = 0; i < elements; ++i) {
        if ((source[i * element_bytes + element_bytes - 1u] & 0x80u) != 0) mask |= UINT64_C(1) << i;
    }
    return mask;
}

static uint64_t vector_byte_sign_mask(const uint8_t *source, unsigned bytes)
{
    uint64_t mask = 0;
    for (unsigned i = 0; i < bytes; ++i) {
        if ((source[i] & 0x80u) != 0) mask |= UINT64_C(1) << i;
    }
    return mask;
}

static int32_t clamp_i32(int64_t value, int32_t minimum, int32_t maximum)
{
    if (value < (int64_t)minimum) return minimum;
    if (value > (int64_t)maximum) return maximum;
    return (int32_t)value;
}

static void vector_unpack(uint8_t *out, const uint8_t *left, const uint8_t *right,
                          unsigned element_bytes, bool high_half)
{
    unsigned elements = 8u / element_bytes;
    unsigned source_base = high_half ? elements : 0u;
    for (unsigned i = 0; i < elements; ++i) {
        memcpy(out + (2u * i) * element_bytes,
               left + (source_base + i) * element_bytes, element_bytes);
        memcpy(out + (2u * i + 1u) * element_bytes,
               right + (source_base + i) * element_bytes, element_bytes);
    }
}

static void vector_pack(uint8_t *out, const uint8_t *left, const uint8_t *right,
                        x86asm_opcode opcode)
{
    unsigned input_element_bytes = opcode == X86ASM_OP_PACKSSDW ? 4u : 2u;
    unsigned input_elements = 16u / input_element_bytes;
    unsigned output_element_bytes = opcode == X86ASM_OP_PACKSSDW ? 2u : 1u;
    for (unsigned source = 0; source < 2u; ++source) {
        const uint8_t *input = source == 0u ? left : right;
        for (unsigned i = 0; i < input_elements; ++i) {
            uint64_t raw = vector_load_le(input + i * input_element_bytes, input_element_bytes);
            int32_t signed_value = input_element_bytes == 2u ? (int32_t)(int16_t)raw : (int32_t)(int32_t)(uint32_t)raw;
            int32_t packed;
            if (opcode == X86ASM_OP_PACKUSWB) packed = clamp_i32((int64_t)signed_value, 0, 255);
            else if (output_element_bytes == 1u) packed = clamp_i32((int64_t)signed_value, -128, 127);
            else packed = clamp_i32((int64_t)signed_value, -32768, 32767);
            vector_store_le(out + (source * input_elements + i) * output_element_bytes,
                            output_element_bytes, (uint32_t)packed);
        }
    }
}

static void vector_unpack_lanes(uint8_t *out, const uint8_t *left, const uint8_t *right,
                                unsigned bytes, unsigned element_bytes, bool high_half)
{
    for (unsigned lane = 0; lane < bytes; lane += 16u) {
        vector_unpack(out + lane, left + lane, right + lane, element_bytes, high_half);
    }
}

static void vector_pack_lanes(uint8_t *out, const uint8_t *left, const uint8_t *right,
                              unsigned bytes, x86asm_opcode opcode)
{
    for (unsigned lane = 0; lane < bytes; lane += 16u) {
        vector_pack(out + lane, left + lane, right + lane, opcode);
    }
}

static void vector_add_f32(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 4) {
        float a;
        float b;
        float sum;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        sum = a + b;
        memcpy(out + i, &sum, sizeof(sum));
    }
}

static void vector_arith_f32(uint8_t *out, const uint8_t *left, const uint8_t *right,
                             unsigned bytes, x86asm_opcode opcode)
{
    for (unsigned i = 0; i < bytes; i += 4u) {
        float a;
        float b;
        float value;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        if (opcode == X86ASM_OP_VSUBPS || opcode == X86ASM_OP_SUBPS) value = a - b;
        else if (opcode == X86ASM_OP_VMULPS || opcode == X86ASM_OP_MULPS) value = a * b;
        else value = a / b;
        memcpy(out + i, &value, sizeof(value));
    }
}

static void vector_compare_equal_elements(uint8_t *out, const uint8_t *left,
                                          const uint8_t *right, unsigned bytes,
                                          unsigned element_bytes)
{
    for (unsigned i = 0; i < bytes; i += element_bytes) {
        bool equal = memcmp(left + i, right + i, element_bytes) == 0;
        memset(out + i, equal ? UINT8_MAX : 0, element_bytes);
    }
}

static void vector_compare_greater_signed(uint8_t *out, const uint8_t *left,
                                          const uint8_t *right, unsigned bytes,
                                          unsigned element_bytes)
{
    for (unsigned i = 0; i < bytes; i += element_bytes) {
        uint64_t a = vector_load_le(left + i, element_bytes);
        uint64_t b = vector_load_le(right + i, element_bytes);
        int64_t signed_a;
        int64_t signed_b;
        if (element_bytes == 1u) {
            signed_a = (int64_t)(int8_t)a;
            signed_b = (int64_t)(int8_t)b;
        } else if (element_bytes == 2u) {
            signed_a = (int64_t)(int16_t)a;
            signed_b = (int64_t)(int16_t)b;
        } else if (element_bytes == 4u) {
            signed_a = (int64_t)(int32_t)(uint32_t)a;
            signed_b = (int64_t)(int32_t)(uint32_t)b;
        } else {
            signed_a = (int64_t)a;
            signed_b = (int64_t)b;
        }
        memset(out + i, signed_a > signed_b ? UINT8_MAX : 0, element_bytes);
    }
}

static uint64_t vector_load_le(const uint8_t *bytes, unsigned count)
{
    uint64_t value = 0;
    for (unsigned i = 0; i < count; ++i) value |= (uint64_t)bytes[i] << (8u * i);
    return value;
}

static void vector_store_le(uint8_t *bytes, unsigned count, uint64_t value)
{
    for (unsigned i = 0; i < count; ++i) bytes[i] = (uint8_t)(value >> (8u * i));
}

static void vector_shift_elements(uint8_t *out, const uint8_t *input,
                                  unsigned bytes, unsigned element_bytes,
                                  unsigned count, bool left, bool arithmetic)
{
    unsigned element_bits = element_bytes * 8u;
    uint64_t element_mask = width_mask(element_bits);
    for (unsigned offset = 0; offset < bytes; offset += element_bytes) {
        uint64_t value = vector_load_le(input + offset, element_bytes);
        uint64_t shifted;
        if (count == 0u) shifted = value;
        else if (count >= element_bits) {
            shifted = !left && arithmetic && (value & (UINT64_C(1) << (element_bits - 1u))) ? element_mask : 0;
        } else if (left) {
            shifted = (value << count) & element_mask;
        } else if (arithmetic && (value & (UINT64_C(1) << (element_bits - 1u))) != 0) {
            shifted = (value >> count) | (element_mask ^ width_mask(element_bits - count));
        } else {
            shifted = value >> count;
        }
        vector_store_le(out + offset, element_bytes, shifted);
    }
}

static void vector_shift_variable(uint8_t *out, const uint8_t *input,
                                   const uint8_t *counts, unsigned bytes,
                                   unsigned element_bytes, bool left, bool arithmetic)
{
    unsigned element_bits = element_bytes * 8u;
    uint64_t element_mask = width_mask(element_bits);
    for (unsigned offset = 0; offset < bytes; offset += element_bytes) {
        uint64_t value = vector_load_le(input + offset, element_bytes);
        uint64_t count = vector_load_le(counts + offset, element_bytes);
        uint64_t shifted;
        if (count >= element_bits) {
            shifted = !left && arithmetic && (value & (UINT64_C(1) << (element_bits - 1u))) ? element_mask : 0;
        } else if (count == 0u) shifted = value;
        else if (left) shifted = (value << count) & element_mask;
        else if (arithmetic && (value & (UINT64_C(1) << (element_bits - 1u))) != 0) {
            shifted = (value >> count) | (element_mask ^ width_mask(element_bits - (unsigned)count));
        } else shifted = value >> count;
        vector_store_le(out + offset, element_bytes, shifted);
    }
}

static void vector_shift_bytes(uint8_t *out, const uint8_t *input,
                               unsigned bytes, unsigned count, bool left)
{
    memset(out, 0, bytes);
    if (count >= bytes) return;
    if (left) memcpy(out + count, input, bytes - count);
    else memcpy(out, input + count, bytes - count);
}

static void vector_shift_bytes_lanes(uint8_t *out, const uint8_t *input,
                                      unsigned bytes, unsigned count, bool left)
{
    for (unsigned lane = 0; lane < bytes; lane += 16u) {
        vector_shift_bytes(out + lane, input + lane, 16, count, left);
    }
}

static void vector_shuffle_dwords(uint8_t *out, const uint8_t *input,
                                   unsigned bytes, uint8_t control)
{
    for (unsigned lane = 0; lane < bytes; lane += 16u) {
        uint32_t source[4];
        uint32_t result[4];
        memcpy(source, input + lane, sizeof(source));
        unsigned control_value = (unsigned)control;
        for (unsigned i = 0; i < 4; ++i) {
            unsigned source_index = (control_value >> (2u * i)) & 3u;
            result[i] = source[source_index];
        }
        memcpy(out + lane, result, sizeof(result));
    }
}

static void vector_blend(uint8_t *out, const uint8_t *left, const uint8_t *right,
                           unsigned bytes, unsigned element_bytes, uint8_t mask)
{
    unsigned elements = bytes / element_bytes;
    for (unsigned i = 0; i < elements; ++i) {
        const uint8_t *source = (mask & (uint8_t)(1u << i)) != 0 ? right : left;
        memcpy(out + i * element_bytes, source + i * element_bytes, element_bytes);
    }
}

static void vector_minmax_f32(uint8_t *out, const uint8_t *left, const uint8_t *right,
                               unsigned bytes, bool maximum)
{
    for (unsigned i = 0; i < bytes; i += 4u) {
        float a;
        float b;
        float value;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        if (isnan(a) || isnan(b)) value = b;
        else if (maximum ? a > b : a < b) value = a;
        else value = b;
        memcpy(out + i, &value, sizeof(value));
    }
}

static void vector_minmax_f64(uint8_t *out, const uint8_t *left, const uint8_t *right,
                               unsigned bytes, bool maximum)
{
    for (unsigned i = 0; i < bytes; i += 8u) {
        double a;
        double b;
        double value;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        if (isnan(a) || isnan(b)) value = b;
        else if (maximum ? a > b : a < b) value = a;
        else value = b;
        memcpy(out + i, &value, sizeof(value));
    }
}

static bool fp_compare_predicate(double a, double b, unsigned predicate)
{
    bool unordered = isnan(a) || isnan(b);
    bool equal = !unordered && a == b;
    switch (predicate & 31u) {
    case 0: case 16: return equal;
    case 1: case 17: return !unordered && a < b;
    case 2: case 18: return !unordered && a <= b;
    case 3: case 19: return unordered;
    case 4: case 20: return !equal;
    case 5: case 21: return unordered || a >= b;
    case 6: case 22: return unordered || a > b;
    case 9: case 25: return unordered || a < b;
    case 10: case 26: return unordered || a <= b;
    case 7: case 23: return !unordered;
    case 8: case 24: return unordered || equal;
    case 11: case 27: return false;
    case 12: case 28: return !unordered && !equal;
    case 13: case 29: return !unordered && a >= b;
    case 14: case 30: return !unordered && a > b;
    case 15: case 31: return true;
    default: return false;
    }
}

static void vector_compare_fp(uint8_t *out, const uint8_t *left, const uint8_t *right,
                              unsigned bytes, unsigned element_bytes, unsigned predicate)
{
    for (unsigned i = 0; i < bytes; i += element_bytes) {
        double a;
        double b;
        uint64_t mask = 0;
        if (element_bytes == 4u) {
            float af;
            float bf;
            memcpy(&af, left + i, sizeof(af));
            memcpy(&bf, right + i, sizeof(bf));
            a = af;
            b = bf;
            mask = fp_compare_predicate(a, b, predicate) ? UINT64_C(0xFFFFFFFF) : 0;
            uint32_t result = (uint32_t)mask;
            memcpy(out + i, &result, sizeof(result));
        } else {
            memcpy(&a, left + i, sizeof(a));
            memcpy(&b, right + i, sizeof(b));
            mask = fp_compare_predicate(a, b, predicate) ? UINT64_MAX : 0;
            memcpy(out + i, &mask, sizeof(mask));
        }
    }
}

static void vector_add_f64(uint8_t *out, const uint8_t *left, const uint8_t *right, unsigned bytes)
{
    for (unsigned i = 0; i < bytes; i += 8) {
        double a;
        double b;
        double sum;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        sum = a + b;
        memcpy(out + i, &sum, sizeof(sum));
    }
}

static void vector_arith_f64(uint8_t *out, const uint8_t *left, const uint8_t *right,
                             unsigned bytes, x86asm_opcode opcode)
{
    for (unsigned i = 0; i < bytes; i += 8u) {
        double a;
        double b;
        double value;
        memcpy(&a, left + i, sizeof(a));
        memcpy(&b, right + i, sizeof(b));
        if (opcode == X86ASM_OP_VSUBPD || opcode == X86ASM_OP_SUBPD) value = a - b;
        else if (opcode == X86ASM_OP_VMULPD || opcode == X86ASM_OP_MULPD) value = a * b;
        else value = a / b;
        memcpy(out + i, &value, sizeof(value));
    }
}

static void multiply_unsigned(uint64_t left, uint64_t right, unsigned width,
                               uint64_t *low, uint64_t *high)
{
    if (width < 64) {
        uint64_t product = left * right;
        *low = product & width_mask(width);
        *high = product >> width;
        return;
    }
    uint64_t left_low = (uint32_t)left;
    uint64_t left_high = left >> 32;
    uint64_t right_low = (uint32_t)right;
    uint64_t right_high = right >> 32;
    uint64_t p00 = left_low * right_low;
    uint64_t p01 = left_low * right_high;
    uint64_t p10 = left_high * right_low;
    uint64_t p11 = left_high * right_high;
    uint64_t result = p00;
    uint64_t carry = 0;
    uint64_t addend = p01 << 32;
    uint64_t old = result;
    result += addend;
    if (result < old) ++carry;
    addend = p10 << 32;
    old = result;
    result += addend;
    if (result < old) ++carry;
    *low = result;
    *high = p11 + (p01 >> 32) + (p10 >> 32) + carry;
}

static uint64_t implicit_accumulator(const x86emu_cpu *cpu, unsigned width)
{
    if (width == 8) return cpu->registers[X86EMU_RAX] & UINT64_C(0xff);
    return cpu->registers[X86EMU_RAX] & width_mask(width);
}

static uint64_t implicit_accumulator_high(const x86emu_cpu *cpu, unsigned width)
{
    if (width == 8) return (cpu->registers[X86EMU_RAX] >> 8) & UINT64_C(0xff);
    return cpu->registers[X86EMU_RDX] & width_mask(width);
}

static bool write_implicit_accumulator(x86emu_cpu *cpu, unsigned width, uint64_t value)
{
    if (width == 8) return write_register(cpu, X86ASM_REG_AL, value);
    if (width == 16) return write_register(cpu, X86ASM_REG_AX, value);
    if (width == 32) return write_register(cpu, X86ASM_REG_EAX, value);
    return write_register(cpu, X86ASM_REG_RAX, value);
}

static bool write_implicit_accumulator_high(x86emu_cpu *cpu, unsigned width, uint64_t value)
{
    if (width == 8) return write_register(cpu, X86ASM_REG_AH, value);
    if (width == 16) return write_register(cpu, X86ASM_REG_DX, value);
    if (width == 32) return write_register(cpu, X86ASM_REG_EDX, value);
    return write_register(cpu, X86ASM_REG_RDX, value);
}

static bool divide_unsigned_128(uint64_t high, uint64_t low, uint64_t divisor,
                                uint64_t *quotient, uint64_t *remainder);

static bool signed_magnitude(uint64_t value, unsigned width, bool *negative, uint64_t *magnitude)
{
    uint64_t mask = width_mask(width);
    uint64_t sign = UINT64_C(1) << (width - 1);
    uint64_t masked = value & mask;
    *negative = (masked & sign) != 0;
    *magnitude = *negative ? ((~masked + 1u) & mask) : masked;
    return true;
}

static bool signed_divide_128(uint64_t high, uint64_t low, uint64_t divisor,
                              unsigned width, uint64_t *quotient, uint64_t *remainder)
{
    bool divisor_negative;
    uint64_t divisor_magnitude;
    bool numerator_negative;
    uint64_t high_magnitude;
    uint64_t low_magnitude;
    uint64_t unsigned_quotient;
    uint64_t unsigned_remainder;
    uint64_t mask = width_mask(width);
    if (!signed_magnitude(divisor, width, &divisor_negative, &divisor_magnitude) || divisor_magnitude == 0) return false;
    if ((high & mask) == 0) {
        numerator_negative = false;
        high_magnitude = 0;
        low_magnitude = low & mask;
    } else {
        numerator_negative = (high & (UINT64_C(1) << (width - 1))) != 0;
        if (!numerator_negative) {
            high_magnitude = high & mask;
            low_magnitude = low & mask;
        } else {
            low_magnitude = (~low + 1u) & mask;
            high_magnitude = (~high + (low_magnitude == 0 ? 1u : 0u)) & mask;
        }
    }
    if (!divide_unsigned_128(high_magnitude, low_magnitude, divisor_magnitude,
                             &unsigned_quotient, &unsigned_remainder)) return false;
    if (unsigned_quotient > mask) return false;
    if (numerator_negative != divisor_negative) {
        uint64_t signed_limit = UINT64_C(1) << (width - 1);
        if (unsigned_quotient > signed_limit) return false;
        *quotient = (~unsigned_quotient + 1u) & mask;
    } else {
        if (unsigned_quotient >= (UINT64_C(1) << (width - 1))) return false;
        *quotient = unsigned_quotient;
    }
    *remainder = numerator_negative ? ((~unsigned_remainder + 1u) & mask) : unsigned_remainder;
    return true;
}

static bool divide_unsigned_128(uint64_t high, uint64_t low, uint64_t divisor,
                                uint64_t *quotient, uint64_t *remainder)
{
    if (divisor == 0 || high >= divisor) return false;
    uint64_t rem = 0;
    uint64_t result = 0;
    for (int bit = 127; bit >= 0; --bit) {
        uint64_t incoming = bit >= 64 ? (high >> (unsigned)(bit - 64)) & 1u
                                      : (low >> (unsigned)bit) & 1u;
        bool doubled_overflow = rem > (UINT64_MAX - incoming) / 2u;
        bool subtract = doubled_overflow || (rem * 2u + incoming >= divisor);
        if (subtract) {
            if (doubled_overflow) rem = rem - (divisor - rem) + incoming;
            else rem = rem * 2u + incoming - divisor;
            if (bit < 64) result |= UINT64_C(1) << (unsigned)bit;
        } else {
            rem = rem * 2u + incoming;
        }
    }
    *quotient = result;
    *remainder = rem;
    return true;
}

static bool read_argument(const x86emu_cpu *cpu,
                          const x86asm_instruction *instruction,
                          const x86asm_argument *argument, unsigned width,
                          uint64_t *value)
{
    if (argument->kind == X86ASM_ARG_REGISTER) return read_register(cpu, argument->value.reg, value);
    if (argument->kind == X86ASM_ARG_IMMEDIATE) {
        *value = (uint64_t)argument->value.immediate & width_mask(width);
        return true;
    }
    if (argument->kind == X86ASM_ARG_RELATIVE) {
        *value = (uint64_t)(int64_t)argument->value.relative;
        return true;
    }
    if (argument->kind == X86ASM_ARG_MEMORY) {
        uint64_t address;
        if (!memory_address(cpu, instruction, &argument->value.memory, &address)) return false;
        return read_memory(cpu, address, width, value);
    }
    return false;
}

static bool write_argument(x86emu_cpu *cpu,
                           const x86asm_instruction *instruction,
                           const x86asm_argument *argument, unsigned width,
                           uint64_t value)
{
    if (argument->kind == X86ASM_ARG_REGISTER) return write_register(cpu, argument->value.reg, value);
    if (argument->kind == X86ASM_ARG_MEMORY) {
        uint64_t address;
        if (!memory_address(cpu, instruction, &argument->value.memory, &address)) return false;
        return write_memory(cpu, address, width, value);
    }
    return false;
}

static void set_flag(x86emu_cpu *cpu, uint64_t flag, bool enabled)
{
    if (enabled) cpu->rflags |= flag;
    else cpu->rflags &= ~flag;
}

static bool get_flag(const x86emu_cpu *cpu, uint64_t flag)
{
    return (cpu->rflags & flag) != 0;
}

static bool even_parity(uint8_t value)
{
    value ^= value >> 4;
    value ^= value >> 2;
    value ^= value >> 1;
    return (value & 1) == 0;
}

static void set_logic_flags(x86emu_cpu *cpu, uint64_t result, unsigned width)
{
    uint64_t masked = result & width_mask(width);
    set_flag(cpu, X86EMU_FLAG_CF, false);
    set_flag(cpu, X86EMU_FLAG_OF, false);
    set_flag(cpu, X86EMU_FLAG_ZF, masked == 0);
    set_flag(cpu, X86EMU_FLAG_SF, (masked >> (width - 1)) != 0);
    set_flag(cpu, X86EMU_FLAG_PF, even_parity((uint8_t)masked));
}

static uint64_t add_values(x86emu_cpu *cpu, uint64_t left, uint64_t right,
                           unsigned width, bool carry_in)
{
    uint64_t mask = width_mask(width);
    uint64_t a = left & mask;
    uint64_t b = right & mask;
    uint64_t carry = carry_in ? 1u : 0u;
    uint64_t result = (a + b + carry) & mask;
    bool carry_out = width == 64 ? result < a || (carry != 0 && result == a) : (a + b + carry) > mask;
    bool sign = (UINT64_C(1) << (width - 1));
    bool overflow = ((~(a ^ b) & (a ^ result)) & sign) != 0;
    set_flag(cpu, X86EMU_FLAG_CF, carry_out);
    set_flag(cpu, X86EMU_FLAG_OF, overflow);
    set_flag(cpu, X86EMU_FLAG_AF, ((a ^ b ^ result) & UINT64_C(0x10)) != 0);
    set_flag(cpu, X86EMU_FLAG_ZF, result == 0);
    set_flag(cpu, X86EMU_FLAG_SF, (result & sign) != 0);
    set_flag(cpu, X86EMU_FLAG_PF, even_parity((uint8_t)result));
    return result;
}

static uint64_t subtract_values(x86emu_cpu *cpu, uint64_t left, uint64_t right,
                                unsigned width, bool borrow_in)
{
    uint64_t mask = width_mask(width);
    uint64_t a = left & mask;
    uint64_t b = right & mask;
    uint64_t borrow = borrow_in ? 1u : 0u;
    uint64_t result = (a - b - borrow) & mask;
    uint64_t rhs = (b + borrow) & mask;
    bool borrow_out = a < b || (borrow != 0 && a == b);
    uint64_t sign = UINT64_C(1) << (width - 1);
    bool overflow = (((a ^ rhs) & (a ^ result)) & sign) != 0;
    set_flag(cpu, X86EMU_FLAG_CF, borrow_out);
    set_flag(cpu, X86EMU_FLAG_OF, overflow);
    set_flag(cpu, X86EMU_FLAG_AF, ((a ^ b ^ result) & UINT64_C(0x10)) != 0);
    set_flag(cpu, X86EMU_FLAG_ZF, result == 0);
    set_flag(cpu, X86EMU_FLAG_SF, (result & sign) != 0);
    set_flag(cpu, X86EMU_FLAG_PF, even_parity((uint8_t)result));
    return result;
}

static bool condition_holds(const x86emu_cpu *cpu, x86asm_opcode opcode)
{
    bool cf = get_flag(cpu, X86EMU_FLAG_CF);
    bool zf = get_flag(cpu, X86EMU_FLAG_ZF);
    bool sf = get_flag(cpu, X86EMU_FLAG_SF);
    bool of = get_flag(cpu, X86EMU_FLAG_OF);
    bool pf = get_flag(cpu, X86EMU_FLAG_PF);
    switch (opcode) {
    case X86ASM_OP_JO: case X86ASM_OP_CMOVO: case X86ASM_OP_SETO: return of;
    case X86ASM_OP_JNO: case X86ASM_OP_CMOVNO: case X86ASM_OP_SETNO: return !of;
    case X86ASM_OP_JB: case X86ASM_OP_CMOVB: case X86ASM_OP_SETB: return cf;
    case X86ASM_OP_JAE: case X86ASM_OP_CMOVAE: case X86ASM_OP_SETAE: return !cf;
    case X86ASM_OP_JE: case X86ASM_OP_CMOVE: case X86ASM_OP_SETE: return zf;
    case X86ASM_OP_JNE: case X86ASM_OP_CMOVNE: case X86ASM_OP_SETNE: return !zf;
    case X86ASM_OP_JBE: case X86ASM_OP_CMOVBE: case X86ASM_OP_SETBE: return cf || zf;
    case X86ASM_OP_JA: case X86ASM_OP_CMOVA: case X86ASM_OP_SETA: return !cf && !zf;
    case X86ASM_OP_JS: case X86ASM_OP_CMOVS: case X86ASM_OP_SETS: return sf;
    case X86ASM_OP_JNS: case X86ASM_OP_CMOVNS: case X86ASM_OP_SETNS: return !sf;
    case X86ASM_OP_JP: case X86ASM_OP_CMOVP: case X86ASM_OP_SETP: return pf;
    case X86ASM_OP_JNP: case X86ASM_OP_CMOVNP: case X86ASM_OP_SETNP: return !pf;
    case X86ASM_OP_JL: case X86ASM_OP_CMOVL: case X86ASM_OP_SETL: return sf != of;
    case X86ASM_OP_JGE: case X86ASM_OP_CMOVGE: case X86ASM_OP_SETGE: return sf == of;
    case X86ASM_OP_JLE: case X86ASM_OP_CMOVLE: case X86ASM_OP_SETLE: return zf || sf != of;
    case X86ASM_OP_JG: case X86ASM_OP_CMOVG: case X86ASM_OP_SETG: return !zf && sf == of;
    default: return false;
    }
}

static uint64_t byte_swap_width(uint64_t value, unsigned width)
{
    unsigned bytes = width / 8;
    uint64_t result = 0;
    for (unsigned i = 0; i < bytes; ++i) result |= ((value >> (8u * i)) & UINT64_C(0xff)) << (8u * (bytes - 1u - i));
    return result;
}

static uint64_t shift_value(x86emu_cpu *cpu, x86asm_opcode opcode,
                            uint64_t value, uint64_t count, unsigned width)
{
    uint64_t mask = width_mask(width);
    unsigned limit = width == 64 ? 63u : 31u;
    unsigned amount = (unsigned)count & limit;
    uint64_t masked = value & mask;
    if (amount == 0) return masked;
    uint64_t sign = UINT64_C(1) << (width - 1);
    uint64_t result;
    if (opcode == X86ASM_OP_SHL) {
        set_flag(cpu, X86EMU_FLAG_CF, ((masked >> (width - amount)) & 1u) != 0);
        result = (masked << amount) & mask;
        set_flag(cpu, X86EMU_FLAG_OF, amount == 1 && (((result & sign) != 0) != get_flag(cpu, X86EMU_FLAG_CF)));
    } else if (opcode == X86ASM_OP_SHR) {
        set_flag(cpu, X86EMU_FLAG_CF, ((masked >> (amount - 1u)) & 1u) != 0);
        result = masked >> amount;
        set_flag(cpu, X86EMU_FLAG_OF, amount == 1 && (masked & sign) != 0);
    } else {
        set_flag(cpu, X86EMU_FLAG_CF, ((masked >> (amount - 1u)) & 1u) != 0);
        result = (uint64_t)((int64_t)((masked ^ sign) - sign) >> amount) & mask;
        set_flag(cpu, X86EMU_FLAG_OF, false);
    }
    set_flag(cpu, X86EMU_FLAG_ZF, result == 0);
    set_flag(cpu, X86EMU_FLAG_SF, (result & sign) != 0);
    set_flag(cpu, X86EMU_FLAG_PF, even_parity((uint8_t)result));
    return result;
}

static x86emu_error stack_push(x86emu_cpu *cpu, uint64_t value)
{
    cpu->registers[X86EMU_RSP] -= X86EMU_STACK_WIDTH;
    if (!write_memory(cpu, cpu->registers[X86EMU_RSP], 64, value)) {
        cpu->registers[X86EMU_RSP] += X86EMU_STACK_WIDTH;
        return X86EMU_ERR_MEMORY;
    }
    return X86EMU_OK;
}

static x86emu_error stack_pop(x86emu_cpu *cpu, uint64_t *value)
{
    if (!read_memory(cpu, cpu->registers[X86EMU_RSP], 64, value)) return X86EMU_ERR_MEMORY;
    cpu->registers[X86EMU_RSP] += X86EMU_STACK_WIDTH;
    return X86EMU_OK;
}

static bool has_breakpoint(const x86emu_cpu *cpu, uint64_t address)
{
    for (size_t i = 0; i < cpu->breakpoint_count; ++i) {
        if (cpu->breakpoints[i] == address) return true;
    }
    return false;
}

static bool instruction_has_prefix(const x86asm_instruction *instruction, x86asm_prefix prefix)
{
    for (unsigned i = 0; i < sizeof(instruction->prefixes) / sizeof(instruction->prefixes[0]); ++i) {
        if ((instruction->prefixes[i] & 0xff) == prefix) return true;
    }
    return false;
}

static void update_string_index(x86emu_cpu *cpu, unsigned index, uint64_t amount)
{
    if (get_flag(cpu, X86EMU_FLAG_DF)) cpu->registers[index] -= amount;
    else cpu->registers[index] += amount;
}

static bool read_string_element(const x86emu_cpu *cpu, uint64_t address,
                                unsigned width, uint64_t *value)
{
    return read_memory(cpu, address, width, value);
}

static bool write_string_element(x86emu_cpu *cpu, uint64_t address,
                                 unsigned width, uint64_t value)
{
    return write_memory(cpu, address, width, value);
}

static x86emu_error map_decode_error(x86asm_error error)
{
    switch (error) {
    case X86ASM_OK: return X86EMU_OK;
    case X86ASM_ERR_TRUNCATED: return X86EMU_ERR_MEMORY;
    case X86ASM_ERR_INVALID_MODE: return X86EMU_ERR_BAD_ARGUMENT;
    case X86ASM_ERR_UNRECOGNIZED: return X86EMU_ERR_DECODE;
    default: return X86EMU_ERR_DECODE;
    }
}

void x86emu_init(x86emu_cpu *cpu, x86emu_memory memory, uint64_t entry)
{
    if (cpu == NULL) return;
    memset(cpu, 0, sizeof(*cpu));
    cpu->memory = memory;
    cpu->rip = entry;
    cpu->rflags = UINT64_C(2);
    cpu->last_error = X86EMU_OK;
}

uint64_t x86emu_get_register(const x86emu_cpu *cpu, unsigned index)
{
    if (cpu == NULL || index >= 16) return 0;
    return cpu->registers[index];
}

void x86emu_set_register(x86emu_cpu *cpu, unsigned index, uint64_t value)
{
    if (cpu != NULL && index < 16) cpu->registers[index] = value;
}

void x86emu_set_interrupt_handler(x86emu_cpu *cpu, x86emu_interrupt_handler handler)
{
    if (cpu != NULL) cpu->interrupt_handler = handler;
}

void x86emu_set_system_handler(x86emu_cpu *cpu, x86emu_system_handler handler)
{
    if (cpu != NULL) cpu->system_handler = handler;
}

bool x86emu_add_breakpoint(x86emu_cpu *cpu, uint64_t address)
{
    if (cpu == NULL || cpu->breakpoint_count >= X86EMU_MAX_BREAKPOINTS || has_breakpoint(cpu, address)) return false;
    cpu->breakpoints[cpu->breakpoint_count++] = address;
    return true;
}

bool x86emu_remove_breakpoint(x86emu_cpu *cpu, uint64_t address)
{
    if (cpu == NULL) return false;
    for (size_t i = 0; i < cpu->breakpoint_count; ++i) {
        if (cpu->breakpoints[i] == address) {
            cpu->breakpoints[i] = cpu->breakpoints[--cpu->breakpoint_count];
            return true;
        }
    }
    return false;
}

const char *x86emu_error_string(x86emu_error error)
{
    switch (error) {
    case X86EMU_OK: return "ok";
    case X86EMU_ERR_BAD_ARGUMENT: return "bad argument";
    case X86EMU_ERR_MEMORY: return "memory access fault";
    case X86EMU_ERR_ARITHMETIC: return "arithmetic fault";
    case X86EMU_ERR_PRIVILEGED: return "privileged instruction";
    case X86EMU_ERR_DECODE: return "instruction decode failure";
    case X86EMU_ERR_UNSUPPORTED: return "unsupported instruction";
    case X86EMU_ERR_BREAKPOINT: return "breakpoint";
    case X86EMU_ERR_INTERRUPT: return "unhandled interrupt";
    case X86EMU_ERR_STEP_LIMIT: return "step limit reached";
    default: return "unknown emulator error";
    }
}

x86emu_error x86emu_step(x86emu_cpu *cpu)
{
    if (cpu == NULL || cpu->memory.data == NULL) return X86EMU_ERR_BAD_ARGUMENT;
    if (cpu->halted) return X86EMU_OK;
    if (has_breakpoint(cpu, cpu->rip)) return X86EMU_ERR_BREAKPOINT;

    size_t offset;
    if (!memory_offset(&cpu->memory, cpu->rip, 1, &offset)) return X86EMU_ERR_MEMORY;
    x86asm_error decode_error = x86asm_decode(cpu->memory.data + offset,
                                               cpu->memory.size - offset, 64,
                                               &cpu->last_instruction);
    if (decode_error != X86ASM_OK) {
        cpu->last_error = map_decode_error(decode_error);
        return cpu->last_error;
    }
    x86emu_error result = X86EMU_OK;

    const x86asm_instruction *instruction = &cpu->last_instruction;
    uint64_t next_rip = cpu->rip + (uint64_t)instruction->length;
    unsigned width = instruction->data_size == 0 ? 64u : (unsigned)instruction->data_size;
    uint64_t left, right, value;
    bool condition;

    switch (instruction->opcode) {
    case X86ASM_OP_NOP:
        cpu->rip = next_rip;
        break;
    case X86ASM_OP_PUSHF:
        result = stack_push(cpu, cpu->rflags | UINT64_C(2));
        if (result == X86EMU_OK) cpu->rip = next_rip;
        break;
    case X86ASM_OP_POPF:
        result = stack_pop(cpu, &value);
        if (result == X86EMU_OK) {
            uint64_t writable = X86EMU_FLAG_CF | X86EMU_FLAG_PF | X86EMU_FLAG_AF |
                                X86EMU_FLAG_ZF | X86EMU_FLAG_SF | X86EMU_FLAG_TF |
                                X86EMU_FLAG_IF | X86EMU_FLAG_DF | X86EMU_FLAG_OF;
            cpu->rflags = (cpu->rflags & ~writable) | (value & writable) | UINT64_C(2);
            cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_CLI:
    case X86ASM_OP_STI:
    case X86ASM_OP_HLT:
        result = X86EMU_ERR_PRIVILEGED;
        break;
    case X86ASM_OP_MOVSB: case X86ASM_OP_MOVSW: case X86ASM_OP_MOVSD: case X86ASM_OP_MOVSQ:
    case X86ASM_OP_STOSB: case X86ASM_OP_STOSW: case X86ASM_OP_STOSD: case X86ASM_OP_STOSQ:
    case X86ASM_OP_LODSB: case X86ASM_OP_LODSW: case X86ASM_OP_LODSD: case X86ASM_OP_LODSQ:
    case X86ASM_OP_CMPSB: case X86ASM_OP_CMPSW: case X86ASM_OP_CMPSD: case X86ASM_OP_CMPSQ:
    case X86ASM_OP_SCASB: case X86ASM_OP_SCASW: case X86ASM_OP_SCASD: case X86ASM_OP_SCASQ: {
        unsigned element_bytes = (unsigned)instruction->data_size / 8u;
        bool is_movs = instruction->opcode >= X86ASM_OP_MOVSB && instruction->opcode <= X86ASM_OP_MOVSQ;
        bool is_stos = instruction->opcode >= X86ASM_OP_STOSB && instruction->opcode <= X86ASM_OP_STOSQ;
        bool is_lods = instruction->opcode >= X86ASM_OP_LODSB && instruction->opcode <= X86ASM_OP_LODSQ;
        bool is_cmps = instruction->opcode >= X86ASM_OP_CMPSB && instruction->opcode <= X86ASM_OP_CMPSQ;
        bool is_scas = instruction->opcode >= X86ASM_OP_SCASB && instruction->opcode <= X86ASM_OP_SCASQ;
        bool repeated = instruction_has_prefix(instruction, X86ASM_PREFIX_REP) ||
                        instruction_has_prefix(instruction, X86ASM_PREFIX_REPN);
        bool repeat_equal = instruction_has_prefix(instruction, X86ASM_PREFIX_REP);
        uint64_t iterations = repeated ? cpu->registers[X86EMU_RCX] : 1;
        if (element_bytes == 0 || element_bytes > 8) result = X86EMU_ERR_UNSUPPORTED;
        else {
            while (iterations != 0) {
                uint64_t source_value = 0;
                uint64_t destination_value = 0;
                if (is_movs || is_cmps || is_lods) {
                    if (!read_string_element(cpu, cpu->registers[X86EMU_RSI], element_bytes * 8u, &source_value)) {
                        result = X86EMU_ERR_MEMORY;
                        break;
                    }
                }
                if (is_movs) {
                    if (!write_string_element(cpu, cpu->registers[X86EMU_RDI], element_bytes * 8u, source_value)) {
                        result = X86EMU_ERR_MEMORY;
                        break;
                    }
                    update_string_index(cpu, X86EMU_RSI, element_bytes);
                    update_string_index(cpu, X86EMU_RDI, element_bytes);
                } else if (is_stos) {
                    source_value = implicit_accumulator(cpu, element_bytes * 8u);
                    if (!write_string_element(cpu, cpu->registers[X86EMU_RDI], element_bytes * 8u, source_value)) {
                        result = X86EMU_ERR_MEMORY;
                        break;
                    }
                    update_string_index(cpu, X86EMU_RDI, element_bytes);
                } else if (is_lods) {
                    if (!write_implicit_accumulator(cpu, element_bytes * 8u, source_value)) {
                        result = X86EMU_ERR_MEMORY;
                        break;
                    }
                    update_string_index(cpu, X86EMU_RSI, element_bytes);
                } else if (is_cmps || is_scas) {
                    if (is_scas) source_value = implicit_accumulator(cpu, element_bytes * 8u);
                    if (!read_string_element(cpu, cpu->registers[X86EMU_RDI], element_bytes * 8u, &destination_value)) {
                        result = X86EMU_ERR_MEMORY;
                        break;
                    }
                    (void)subtract_values(cpu, source_value, destination_value, element_bytes * 8u, false);
                    if (is_cmps) update_string_index(cpu, X86EMU_RSI, element_bytes);
                    update_string_index(cpu, X86EMU_RDI, element_bytes);
                    if (repeated && ((repeat_equal && !get_flag(cpu, X86EMU_FLAG_ZF)) ||
                                     (!repeat_equal && get_flag(cpu, X86EMU_FLAG_ZF)))) {
                        --cpu->registers[X86EMU_RCX];
                        --iterations;
                        break;
                    }
                }
                if (repeated) --cpu->registers[X86EMU_RCX];
                --iterations;
            }
            if (result == X86EMU_OK) cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_VMOVLPS:
    case X86ASM_OP_VMOVHPS:
    case X86ASM_OP_VMOVLPD:
    case X86ASM_OP_VMOVHPD: {
        bool high = instruction->opcode == X86ASM_OP_VMOVHPS || instruction->opcode == X86ASM_OP_VMOVHPD;
        bool store = instruction->arguments[0].kind == X86ASM_ARG_MEMORY;
        unsigned lane_offset = high ? 8u : 0u;
        uint8_t vector_value[16];
        uint64_t scalar = 0;
        if (store) {
            if (!read_vector_argument(cpu, instruction, &instruction->arguments[1], 16, vector_value)) result = X86EMU_ERR_MEMORY;
            else {
                memcpy(&scalar, vector_value + lane_offset, sizeof(scalar));
                if (!write_argument(cpu, instruction, &instruction->arguments[0], 64, scalar)) result = X86EMU_ERR_MEMORY;
                else cpu->rip = next_rip;
            }
        } else if (!read_vector_argument(cpu, instruction, &instruction->arguments[1], 16, vector_value) ||
                   !read_argument(cpu, instruction, &instruction->arguments[2], 64, &scalar)) result = X86EMU_ERR_MEMORY;
        else {
            memcpy(vector_value + lane_offset, &scalar, sizeof(scalar));
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, vector_value)) result = X86EMU_ERR_MEMORY;
            else {
                zero_vector_upper_128(cpu, &instruction->arguments[0]);
                cpu->rip = next_rip;
            }
        }
        break;
    }
    case X86ASM_OP_MOVLPS:
    case X86ASM_OP_MOVHPS:
    case X86ASM_OP_MOVLPD:
    case X86ASM_OP_MOVHPD: {
        bool high = instruction->opcode == X86ASM_OP_MOVHPS || instruction->opcode == X86ASM_OP_MOVHPD;
        bool store = instruction->arguments[0].kind == X86ASM_ARG_MEMORY;
        unsigned lane_offset = high ? 8u : 0u;
        uint8_t vector_value[16];
        uint64_t scalar = 0;
        if (store) {
            if (!read_vector_argument(cpu, instruction, &instruction->arguments[1], 16, vector_value)) result = X86EMU_ERR_MEMORY;
            else {
                memcpy(&scalar, vector_value + lane_offset, sizeof(scalar));
                if (!write_argument(cpu, instruction, &instruction->arguments[0], 64, scalar)) result = X86EMU_ERR_MEMORY;
                else cpu->rip = next_rip;
            }
        } else if (!read_argument(cpu, instruction, &instruction->arguments[1], 64, &scalar) ||
                   !read_vector_argument(cpu, instruction, &instruction->arguments[0], 16, vector_value)) result = X86EMU_ERR_MEMORY;
        else {
            memcpy(vector_value + lane_offset, &scalar, sizeof(scalar));
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, vector_value)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_MOVD:
    case X86ASM_OP_MOVQ:
    case X86ASM_OP_VMOVD:
    case X86ASM_OP_VMOVQ: {
        unsigned transfer_bytes = width / 8u;
        if (argument_is_vector(cpu, &instruction->arguments[0])) {
            uint64_t scalar = 0;
            uint8_t source_value[16];
            uint8_t vector_value[16] = { 0 };
            bool read_ok;
            if (argument_is_vector(cpu, &instruction->arguments[1])) {
                read_ok = read_vector_argument(cpu, instruction, &instruction->arguments[1], transfer_bytes, source_value);
                if (read_ok) memcpy(&scalar, source_value, transfer_bytes);
            } else {
                read_ok = read_argument(cpu, instruction, &instruction->arguments[1], width, &scalar);
            }
            if (!read_ok) result = X86EMU_ERR_MEMORY;
            else {
                memcpy(vector_value, &scalar, transfer_bytes);
                if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, vector_value)) result = X86EMU_ERR_MEMORY;
                else {
                    if (instruction->opcode == X86ASM_OP_VMOVD || instruction->opcode == X86ASM_OP_VMOVQ) zero_vector_upper_128(cpu, &instruction->arguments[0]);
                    cpu->rip = next_rip;
                }
            }
        } else {
            uint8_t vector_value[16];
            uint64_t scalar = 0;
            if (!read_vector_argument(cpu, instruction, &instruction->arguments[1], 16, vector_value)) result = X86EMU_ERR_MEMORY;
            else {
                memcpy(&scalar, vector_value, transfer_bytes);
                if (!write_argument(cpu, instruction, &instruction->arguments[0], width, scalar)) result = X86EMU_ERR_MEMORY;
                else cpu->rip = next_rip;
            }
        }
        break;
    }
    case X86ASM_OP_VMOVDQA:
    case X86ASM_OP_VMOVDQU: {
        unsigned vector_bytes = width / 8u;
        uint8_t vector_value[32];
        if (vector_bytes > sizeof(vector_value) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[1], vector_bytes, vector_value) ||
            !write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, vector_value)) result = X86EMU_ERR_MEMORY;
        else { zero_vector_upper_width(cpu, &instruction->arguments[0], vector_bytes); cpu->rip = next_rip; }
        break;
    }
    case X86ASM_OP_MOVNTDQ:
    case X86ASM_OP_VMOVNTDQ: {
        unsigned vector_bytes = instruction->opcode == X86ASM_OP_MOVNTDQ ? 16u : width / 8u;
        uint8_t vector_value[32];
        if (vector_bytes > sizeof(vector_value) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[1], vector_bytes, vector_value) ||
            !write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, vector_value)) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    }
    case X86ASM_OP_MOVNTDQA:
    case X86ASM_OP_VMOVNTDQA:
    case X86ASM_OP_LDDQU:
    case X86ASM_OP_VLDDQU: {
        bool legacy = instruction->opcode == X86ASM_OP_MOVNTDQA || instruction->opcode == X86ASM_OP_LDDQU;
        unsigned vector_bytes = legacy ? 16u : width / 8u;
        uint8_t vector_value[32];
        if (vector_bytes > sizeof(vector_value) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[1], vector_bytes, vector_value) ||
            !write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, vector_value)) result = X86EMU_ERR_MEMORY;
        else { if (!legacy) zero_vector_upper_width(cpu, &instruction->arguments[0], vector_bytes); cpu->rip = next_rip; }
        break;
    }
    case X86ASM_OP_MOVDQA:
    case X86ASM_OP_MOVDQU:
    case X86ASM_OP_MOVUPS:
    case X86ASM_OP_MOVUPD: {
        uint8_t vector_value[16];
        if (!read_vector_argument(cpu, instruction, &instruction->arguments[1], 16, vector_value) ||
            !write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, vector_value)) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    }
    case X86ASM_OP_ADDSS:
    case X86ASM_OP_SUBSS:
    case X86ASM_OP_MULSS:
    case X86ASM_OP_DIVSS:
    case X86ASM_OP_MINSS:
    case X86ASM_OP_MAXSS:
    case X86ASM_OP_ADDSD:
    case X86ASM_OP_SUBSD:
    case X86ASM_OP_MULSD:
    case X86ASM_OP_DIVSD:
    case X86ASM_OP_MINSD:
    case X86ASM_OP_MAXSD: {
        unsigned scalar_bytes = (instruction->opcode == X86ASM_OP_ADDSS || instruction->opcode == X86ASM_OP_SUBSS || instruction->opcode == X86ASM_OP_MULSS || instruction->opcode == X86ASM_OP_DIVSS || instruction->opcode == X86ASM_OP_MINSS || instruction->opcode == X86ASM_OP_MAXSS) ? 4u : 8u;
        uint8_t destination[16];
        uint8_t source[8];
        if (!read_vector_argument(cpu, instruction, &instruction->arguments[0], 16, destination) ||
            !read_scalar_vector_argument(cpu, instruction, &instruction->arguments[1], scalar_bytes, source)) result = X86EMU_ERR_MEMORY;
        else {
            if (scalar_bytes == 4u) {
                float a, b, r;
                memcpy(&a, destination, sizeof(a)); memcpy(&b, source, sizeof(b));
                if (instruction->opcode == X86ASM_OP_ADDSS) r = a + b; else if (instruction->opcode == X86ASM_OP_SUBSS) r = a - b; else if (instruction->opcode == X86ASM_OP_MULSS) r = a * b; else if (instruction->opcode == X86ASM_OP_DIVSS) r = a / b; else if (isnan(a) || isnan(b) || a == b) r = b; else r = instruction->opcode == X86ASM_OP_MINSS ? (a < b ? a : b) : (a > b ? a : b);
                memcpy(destination, &r, sizeof(r));
            } else {
                double a, b, r;
                memcpy(&a, destination, sizeof(a)); memcpy(&b, source, sizeof(b));
                if (instruction->opcode == X86ASM_OP_ADDSD) r = a + b; else if (instruction->opcode == X86ASM_OP_SUBSD) r = a - b; else if (instruction->opcode == X86ASM_OP_MULSD) r = a * b; else if (instruction->opcode == X86ASM_OP_DIVSD) r = a / b; else if (isnan(a) || isnan(b) || a == b) r = b; else r = instruction->opcode == X86ASM_OP_MINSD ? (a < b ? a : b) : (a > b ? a : b);
                memcpy(destination, &r, sizeof(r));
            }
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, destination)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_ADDPS:
    case X86ASM_OP_SUBPS:
    case X86ASM_OP_MULPS:
    case X86ASM_OP_DIVPS:
    case X86ASM_OP_MINPS:
    case X86ASM_OP_MAXPS:
    case X86ASM_OP_CMPPS:
    case X86ASM_OP_ADDPD:
    case X86ASM_OP_SUBPD:
    case X86ASM_OP_MULPD:
    case X86ASM_OP_DIVPD:
    case X86ASM_OP_MINPD:
    case X86ASM_OP_MAXPD:
    case X86ASM_OP_CMPPD:
    case X86ASM_OP_XORPS: {
        uint8_t left_vector[16];
        uint8_t right_vector[16];
        uint8_t output_vector[16];
        if (!read_vector_argument(cpu, instruction, &instruction->arguments[0], 16, left_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[1], 16, right_vector)) result = X86EMU_ERR_MEMORY;
        else {
            if (instruction->opcode == X86ASM_OP_ADDPS) vector_add_f32(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_SUBPS || instruction->opcode == X86ASM_OP_MULPS || instruction->opcode == X86ASM_OP_DIVPS) vector_arith_f32(output_vector, left_vector, right_vector, 16, instruction->opcode);
            else if (instruction->opcode == X86ASM_OP_MINPS || instruction->opcode == X86ASM_OP_MAXPS) vector_minmax_f32(output_vector, left_vector, right_vector, 16, instruction->opcode == X86ASM_OP_MAXPS);
            else if (instruction->opcode == X86ASM_OP_CMPPS) vector_compare_fp(output_vector, left_vector, right_vector, 16, 4u, (unsigned)instruction->arguments[2].value.immediate);
            else if (instruction->opcode == X86ASM_OP_ADDPD) vector_add_f64(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_SUBPD || instruction->opcode == X86ASM_OP_MULPD || instruction->opcode == X86ASM_OP_DIVPD) vector_arith_f64(output_vector, left_vector, right_vector, 16, instruction->opcode);
            else if (instruction->opcode == X86ASM_OP_MINPD || instruction->opcode == X86ASM_OP_MAXPD) vector_minmax_f64(output_vector, left_vector, right_vector, 16, instruction->opcode == X86ASM_OP_MAXPD);
            else if (instruction->opcode == X86ASM_OP_CMPPD) vector_compare_fp(output_vector, left_vector, right_vector, 16, 8u, (unsigned)instruction->arguments[2].value.immediate);
            else vector_bitwise(output_vector, left_vector, right_vector, 16, X86ASM_OP_VXOR);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, output_vector)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_PMULLD:
    case X86ASM_OP_VPMULLD: {
        bool legacy = instruction->opcode == X86ASM_OP_PMULLD;
        unsigned vector_bytes = legacy ? 16u : width / 8u;
        uint8_t left_vector[32];
        uint8_t right_vector[32];
        uint8_t output_vector[32];
        unsigned left_index = legacy ? 0u : 1u;
        unsigned right_index = legacy ? 1u : 2u;
        if (vector_bytes > sizeof(output_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[left_index], vector_bytes, left_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[right_index], vector_bytes, right_vector)) result = X86EMU_ERR_MEMORY;
        else {
            vector_mul_signed_dwords_low(output_vector, left_vector, right_vector, vector_bytes);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, output_vector)) result = X86EMU_ERR_MEMORY;
            else { if (!legacy) zero_vector_upper_width(cpu, &instruction->arguments[0], vector_bytes); cpu->rip = next_rip; }
        }
        break;
    }
    case X86ASM_OP_VZEROUPPER:
        for (unsigned i = 0; i < 32; ++i) memset(cpu->vector_registers[i] + 16, 0, 48);
        cpu->rip = next_rip;
        break;
    case X86ASM_OP_VMOVUPS:
    case X86ASM_OP_VMOVUPD: {
        unsigned vector_bytes = width / 8;
        uint8_t vector_value[64];
        if (vector_bytes > sizeof(vector_value) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[1], vector_bytes, vector_value) ||
            !write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, vector_value)) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    }
    case X86ASM_OP_MOVSS:
    case X86ASM_OP_MOVSD_SCALAR:
    case X86ASM_OP_VMOVSS:
    case X86ASM_OP_VMOVSD: {
        unsigned scalar_bytes = (instruction->opcode == X86ASM_OP_MOVSS || instruction->opcode == X86ASM_OP_VMOVSS) ? 4u : 8u;
        uint8_t scalar_value[8];
        if (instruction->opcode == X86ASM_OP_MOVSS || instruction->opcode == X86ASM_OP_MOVSD_SCALAR) {
            if (instruction->arguments[0].kind == X86ASM_ARG_MEMORY) {
                if (!read_scalar_vector_argument(cpu, instruction, &instruction->arguments[1], scalar_bytes, scalar_value) || !write_vector_argument(cpu, instruction, &instruction->arguments[0], scalar_bytes, scalar_value)) result = X86EMU_ERR_MEMORY;
                else cpu->rip = next_rip;
            } else if (!read_scalar_vector_argument(cpu, instruction, &instruction->arguments[1], scalar_bytes, scalar_value)) result = X86EMU_ERR_MEMORY;
            else {
                uint8_t destination[16] = { 0 };
                if (instruction->arguments[1].kind == X86ASM_ARG_REGISTER && !read_vector_argument(cpu, instruction, &instruction->arguments[0], 16, destination)) result = X86EMU_ERR_MEMORY;
                else {
                    memcpy(destination, scalar_value, scalar_bytes);
                    if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, destination)) result = X86EMU_ERR_MEMORY;
                    else cpu->rip = next_rip;
                }
            }
        } else if (instruction->arguments[0].kind == X86ASM_ARG_MEMORY) {
            if (!read_scalar_vector_argument(cpu, instruction, &instruction->arguments[1], scalar_bytes, scalar_value) || !write_vector_argument(cpu, instruction, &instruction->arguments[0], scalar_bytes, scalar_value)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        } else if (instruction->arguments[2].kind == X86ASM_ARG_NONE) {
            if (!read_scalar_vector_argument(cpu, instruction, &instruction->arguments[1], scalar_bytes, scalar_value)) result = X86EMU_ERR_MEMORY;
            else {
                uint8_t destination[16] = { 0 };
                memcpy(destination, scalar_value, scalar_bytes);
                if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, destination)) result = X86EMU_ERR_MEMORY;
                else { zero_vector_upper_128(cpu, &instruction->arguments[0]); cpu->rip = next_rip; }
            }
        } else {
            uint8_t first[16];
            if (!read_vector_argument(cpu, instruction, &instruction->arguments[1], 16, first) || !read_scalar_vector_argument(cpu, instruction, &instruction->arguments[2], scalar_bytes, scalar_value)) result = X86EMU_ERR_MEMORY;
            else {
                memcpy(first, scalar_value, scalar_bytes);
                if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, first)) result = X86EMU_ERR_MEMORY;
                else { zero_vector_upper_128(cpu, &instruction->arguments[0]); cpu->rip = next_rip; }
            }
        }
        break;
    }
    case X86ASM_OP_BLENDPS:
    case X86ASM_OP_BLENDPD:
    case X86ASM_OP_VBLENDPS:
    case X86ASM_OP_VBLENDPD:
    case X86ASM_OP_VPBLENDD: {
        bool legacy = instruction->opcode == X86ASM_OP_BLENDPS || instruction->opcode == X86ASM_OP_BLENDPD;
        unsigned vector_bytes = legacy ? 16u : width / 8u;
        unsigned element_bytes = instruction->opcode == X86ASM_OP_BLENDPS || instruction->opcode == X86ASM_OP_VBLENDPS || instruction->opcode == X86ASM_OP_VPBLENDD ? 4u : 8u;
        unsigned left_index = legacy ? 0u : 1u;
        unsigned right_index = legacy ? 1u : 2u;
        unsigned immediate_index = legacy ? 2u : 3u;
        uint8_t left_vector[64];
        uint8_t right_vector[64];
        uint8_t output_vector[64];
        uint64_t immediate_value;
        if (vector_bytes > sizeof(output_vector) || instruction->arguments[immediate_index].kind != X86ASM_ARG_IMMEDIATE ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[left_index], vector_bytes, left_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[right_index], vector_bytes, right_vector) ||
            !read_argument(cpu, instruction, &instruction->arguments[immediate_index], 8u, &immediate_value)) result = X86EMU_ERR_MEMORY;
        else {
            vector_blend(output_vector, left_vector, right_vector, vector_bytes, element_bytes, (uint8_t)immediate_value);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, output_vector)) result = X86EMU_ERR_MEMORY;
            else { if (!legacy) zero_vector_upper_width(cpu, &instruction->arguments[0], vector_bytes); cpu->rip = next_rip; }
        }
        break;
    }
    case X86ASM_OP_PSHUFD:
    case X86ASM_OP_PSHUFLW:
    case X86ASM_OP_PSHUFHW:
    case X86ASM_OP_VPSHUFD:
    case X86ASM_OP_VPSHUFLW:
    case X86ASM_OP_VPSHUFHW: {
        unsigned vector_bytes = instruction->opcode == X86ASM_OP_PSHUFD || instruction->opcode == X86ASM_OP_PSHUFLW || instruction->opcode == X86ASM_OP_PSHUFHW ? 16u : width / 8u;
        uint8_t input[64];
        uint8_t output[64];
        uint64_t immediate;
        if (vector_bytes > sizeof(input) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[1], vector_bytes, input) ||
            !read_argument(cpu, instruction, &instruction->arguments[2], 8, &immediate)) result = X86EMU_ERR_MEMORY;
        else {
            if (instruction->opcode == X86ASM_OP_PSHUFD || instruction->opcode == X86ASM_OP_VPSHUFD) vector_shuffle_dwords(output, input, vector_bytes, (uint8_t)immediate);
            else vector_shuffle_words(output, input, vector_bytes, (uint8_t)immediate, instruction->opcode == X86ASM_OP_PSHUFHW || instruction->opcode == X86ASM_OP_VPSHUFHW);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, output)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_PTEST:
    case X86ASM_OP_VPTEST: {
        unsigned vector_bytes = instruction->opcode == X86ASM_OP_PTEST ? 16u : width / 8u;
        uint8_t destination[64];
        uint8_t source[64];
        bool and_zero = true;
        bool and_not_zero = true;
        if (vector_bytes > sizeof(destination) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, destination) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[1], vector_bytes, source)) result = X86EMU_ERR_MEMORY;
        else {
            for (unsigned i = 0; i < vector_bytes; ++i) {
                if ((source[i] & destination[i]) != 0u) and_zero = false;
                if ((source[i] & (uint8_t)~destination[i]) != 0u) and_not_zero = false;
            }
            set_flag(cpu, X86EMU_FLAG_ZF, and_zero);
            set_flag(cpu, X86EMU_FLAG_CF, and_not_zero);
            set_flag(cpu, X86EMU_FLAG_OF, false);
            set_flag(cpu, X86EMU_FLAG_AF, false);
            set_flag(cpu, X86EMU_FLAG_PF, false);
            set_flag(cpu, X86EMU_FLAG_SF, false);
            cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_MOVMSKPS:
    case X86ASM_OP_MOVMSKPD:
    case X86ASM_OP_PMOVMSKB:
    case X86ASM_OP_VMOVMSKPS:
    case X86ASM_OP_VMOVMSKPD:
    case X86ASM_OP_VPMOVMSKB: {
        bool legacy_mask = instruction->opcode == X86ASM_OP_MOVMSKPS || instruction->opcode == X86ASM_OP_MOVMSKPD || instruction->opcode == X86ASM_OP_PMOVMSKB;
        bool byte_mask = instruction->opcode == X86ASM_OP_PMOVMSKB || instruction->opcode == X86ASM_OP_VPMOVMSKB;
        unsigned vector_bytes = legacy_mask ? 16u : width / 8u;
        unsigned element_bytes = instruction->opcode == X86ASM_OP_MOVMSKPD || instruction->opcode == X86ASM_OP_VMOVMSKPD ? 8u : 4u;
        uint8_t input[64];
        uint64_t mask;
        unsigned destination_width = legacy_mask && width == 64u ? 64u : 32u;
        if (vector_bytes > sizeof(input) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[1], vector_bytes, input)) result = X86EMU_ERR_MEMORY;
        else {
            mask = byte_mask ? vector_byte_sign_mask(input, vector_bytes) : vector_sign_mask(input, vector_bytes, element_bytes);
            if (!write_argument(cpu, instruction, &instruction->arguments[0], destination_width, mask)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_VPCMPEQB:
    case X86ASM_OP_VPCMPEQW:
    case X86ASM_OP_VPCMPEQD:
    case X86ASM_OP_VPCMPEQQ:
    case X86ASM_OP_PCMPGTB:
    case X86ASM_OP_PCMPGTW:
    case X86ASM_OP_PCMPGTD:
    case X86ASM_OP_VPCMPGTB:
    case X86ASM_OP_VPCMPGTW:
    case X86ASM_OP_VPCMPGTD:
    case X86ASM_OP_VPCMPGTQ: {
        bool legacy_signed_compare = instruction->opcode == X86ASM_OP_PCMPGTB || instruction->opcode == X86ASM_OP_PCMPGTW || instruction->opcode == X86ASM_OP_PCMPGTD;
        unsigned vector_bytes = legacy_signed_compare ? 16u : width / 8u;
        unsigned element_bytes = instruction->opcode == X86ASM_OP_VPCMPEQB || instruction->opcode == X86ASM_OP_PCMPGTB || instruction->opcode == X86ASM_OP_VPCMPGTB ? 1u :
                                 (instruction->opcode == X86ASM_OP_VPCMPEQW || instruction->opcode == X86ASM_OP_PCMPGTW || instruction->opcode == X86ASM_OP_VPCMPGTW ? 2u :
                                  (instruction->opcode == X86ASM_OP_VPCMPEQD || instruction->opcode == X86ASM_OP_PCMPGTD || instruction->opcode == X86ASM_OP_VPCMPGTD ? 4u : 8u));
        unsigned left_argument = legacy_signed_compare ? 0u : 1u;
        unsigned right_argument = legacy_signed_compare ? 1u : 2u;
        uint8_t left_vector[64];
        uint8_t right_vector[64];
        uint8_t output_vector[64];
        if (vector_bytes > sizeof(output_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[left_argument], vector_bytes, left_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[right_argument], vector_bytes, right_vector)) result = X86EMU_ERR_MEMORY;
        else {
            if (legacy_signed_compare || instruction->opcode == X86ASM_OP_VPCMPGTB || instruction->opcode == X86ASM_OP_VPCMPGTW || instruction->opcode == X86ASM_OP_VPCMPGTD || instruction->opcode == X86ASM_OP_VPCMPGTQ) {
                vector_compare_greater_signed(output_vector, left_vector, right_vector, vector_bytes, element_bytes);
            } else {
                vector_compare_equal_elements(output_vector, left_vector, right_vector, vector_bytes, element_bytes);
            }
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, output_vector)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_PSLLW:
    case X86ASM_OP_PSLLD:
    case X86ASM_OP_PSLLQ:
    case X86ASM_OP_PSRLW:
    case X86ASM_OP_PSRLD:
    case X86ASM_OP_PSRLQ:
    case X86ASM_OP_PSRAW:
    case X86ASM_OP_PSRAD:
    case X86ASM_OP_PSLLDQ:
    case X86ASM_OP_PSRLDQ: {
        uint8_t input[16];
        uint8_t output[16];
        if (!read_vector_argument(cpu, instruction, &instruction->arguments[0], 16, input) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], 8, &right)) result = X86EMU_ERR_MEMORY;
        else {
            unsigned count = (unsigned)right & 0xFFu;
            if (instruction->opcode == X86ASM_OP_PSLLDQ) vector_shift_bytes(output, input, 16, count, true);
            else if (instruction->opcode == X86ASM_OP_PSRLDQ) vector_shift_bytes(output, input, 16, count, false);
            else {
                unsigned element_bytes = instruction->opcode == X86ASM_OP_PSLLQ || instruction->opcode == X86ASM_OP_PSRLQ ? 8u :
                                         instruction->opcode == X86ASM_OP_PSLLD || instruction->opcode == X86ASM_OP_PSRLD || instruction->opcode == X86ASM_OP_PSRAD ? 4u : 2u;
                bool left_shift = instruction->opcode == X86ASM_OP_PSLLW || instruction->opcode == X86ASM_OP_PSLLD || instruction->opcode == X86ASM_OP_PSLLQ;
                bool arithmetic = instruction->opcode == X86ASM_OP_PSRAW || instruction->opcode == X86ASM_OP_PSRAD;
                vector_shift_elements(output, input, 16, element_bytes, count, left_shift, arithmetic);
            }
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, output)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_PHMINPOSUW:
    case X86ASM_OP_VPHMINPOSUW: {
        uint8_t input[16];
        uint8_t output[16];
        if (!read_vector_argument(cpu, instruction, &instruction->arguments[1], 16u, input)) result = X86EMU_ERR_MEMORY;
        else {
            vector_horizontal_minpos_unsigned_words(output, input);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16u, output)) result = X86EMU_ERR_MEMORY;
            else { if (instruction->opcode == X86ASM_OP_VPHMINPOSUW) zero_vector_upper_128(cpu, &instruction->arguments[0]); cpu->rip = next_rip; }
        }
        break;
    }
    case X86ASM_OP_PINSRB:
    case X86ASM_OP_PINSRW:
    case X86ASM_OP_PINSRD:
    case X86ASM_OP_PINSRQ:
    case X86ASM_OP_VPINSRB:
    case X86ASM_OP_VPINSRW:
    case X86ASM_OP_VPINSRD:
    case X86ASM_OP_VPINSRQ: {
        bool legacy = instruction->opcode == X86ASM_OP_PINSRB || instruction->opcode == X86ASM_OP_PINSRW || instruction->opcode == X86ASM_OP_PINSRD || instruction->opcode == X86ASM_OP_PINSRQ;
        unsigned source_index = legacy ? 1u : 2u;
        unsigned immediate_index = legacy ? 2u : 3u;
        unsigned element_bytes = instruction->opcode == X86ASM_OP_PINSRB || instruction->opcode == X86ASM_OP_VPINSRB ? 1u : (instruction->opcode == X86ASM_OP_PINSRW || instruction->opcode == X86ASM_OP_VPINSRW ? 2u : (instruction->opcode == X86ASM_OP_PINSRQ || instruction->opcode == X86ASM_OP_VPINSRQ ? 8u : 4u));
        unsigned index_mask = element_bytes == 1u ? 15u : (element_bytes == 2u ? 7u : (element_bytes == 4u ? 3u : 1u));
        uint8_t destination[16];
        uint64_t scalar;
        uint64_t immediate;
        if (!read_vector_argument(cpu, instruction, legacy ? &instruction->arguments[0] : &instruction->arguments[1], 16u, destination) ||
            !read_argument(cpu, instruction, &instruction->arguments[source_index], element_bytes * 8u, &scalar) ||
            !read_argument(cpu, instruction, &instruction->arguments[immediate_index], 8u, &immediate)) result = X86EMU_ERR_MEMORY;
        else {
            unsigned element_index = (unsigned)immediate & index_mask;
            vector_store_le(destination + element_index * element_bytes, element_bytes, scalar);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16u, destination)) result = X86EMU_ERR_MEMORY;
            else { if (!legacy) zero_vector_upper_128(cpu, &instruction->arguments[0]); cpu->rip = next_rip; }
        }
        break;
    }
    case X86ASM_OP_PEXTRB:
    case X86ASM_OP_PEXTRW:
    case X86ASM_OP_PEXTRD:
    case X86ASM_OP_PEXTRQ:
    case X86ASM_OP_VPEXTRB:
    case X86ASM_OP_VPEXTRW:
    case X86ASM_OP_VPEXTRD:
    case X86ASM_OP_VPEXTRQ: {
        unsigned element_bytes = instruction->opcode == X86ASM_OP_PEXTRB || instruction->opcode == X86ASM_OP_VPEXTRB ? 1u : (instruction->opcode == X86ASM_OP_PEXTRW || instruction->opcode == X86ASM_OP_VPEXTRW ? 2u : (instruction->opcode == X86ASM_OP_PEXTRQ || instruction->opcode == X86ASM_OP_VPEXTRQ ? 8u : 4u));
        unsigned index_mask = element_bytes == 1u ? 15u : (element_bytes == 2u ? 7u : (element_bytes == 4u ? 3u : 1u));
        uint8_t source[16];
        uint64_t immediate;
        uint64_t scalar;
        if (!read_vector_argument(cpu, instruction, &instruction->arguments[1], 16u, source) ||
            !read_argument(cpu, instruction, &instruction->arguments[2], 8u, &immediate)) result = X86EMU_ERR_MEMORY;
        else {
            unsigned element_index = (unsigned)immediate & index_mask;
            scalar = vector_load_le(source + element_index * element_bytes, element_bytes);
            unsigned destination_width = instruction->arguments[0].kind == X86ASM_ARG_REGISTER ? 64u : element_bytes * 8u;
            if (!write_argument(cpu, instruction, &instruction->arguments[0], destination_width, scalar)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_PALIGNR:
    case X86ASM_OP_VPALIGNR: {
        bool legacy = instruction->opcode == X86ASM_OP_PALIGNR;
        unsigned vector_bytes = legacy ? 16u : width / 8u;
        unsigned left_index = legacy ? 0u : 1u;
        unsigned right_index = legacy ? 1u : 2u;
        unsigned immediate_index = legacy ? 2u : 3u;
        uint8_t left_vector[32];
        uint8_t right_vector[32];
        uint8_t output_vector[32];
        uint64_t immediate_value;
        if (vector_bytes > sizeof(left_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[left_index], vector_bytes, left_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[right_index], vector_bytes, right_vector) ||
            !read_argument(cpu, instruction, &instruction->arguments[immediate_index], 8u, &immediate_value)) result = X86EMU_ERR_MEMORY;
        else {
            vector_align_right(output_vector, left_vector, right_vector, vector_bytes, (uint8_t)immediate_value);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, output_vector)) result = X86EMU_ERR_MEMORY;
            else { if (!legacy) zero_vector_upper_width(cpu, &instruction->arguments[0], vector_bytes); cpu->rip = next_rip; }
        }
        break;
    }
    case X86ASM_OP_PBLENDW:
    case X86ASM_OP_VPBLENDW: {
        bool legacy = instruction->opcode == X86ASM_OP_PBLENDW;
        unsigned vector_bytes = legacy ? 16u : width / 8u;
        unsigned left_index = legacy ? 0u : 1u;
        unsigned right_index = legacy ? 1u : 2u;
        unsigned immediate_index = legacy ? 2u : 3u;
        uint8_t left_vector[32];
        uint8_t right_vector[32];
        uint8_t output_vector[32];
        uint64_t immediate_value;
        if (vector_bytes > sizeof(left_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[left_index], vector_bytes, left_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[right_index], vector_bytes, right_vector) ||
            !read_argument(cpu, instruction, &instruction->arguments[immediate_index], 8u, &immediate_value)) result = X86EMU_ERR_MEMORY;
        else {
            vector_blend_words(output_vector, left_vector, right_vector, vector_bytes, (uint8_t)immediate_value);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, output_vector)) result = X86EMU_ERR_MEMORY;
            else { if (!legacy) zero_vector_upper_width(cpu, &instruction->arguments[0], vector_bytes); cpu->rip = next_rip; }
        }
        break;
    }
    case X86ASM_OP_PBLENDVB:
    case X86ASM_OP_VPBLENDVB:
    case X86ASM_OP_BLENDVPS:
    case X86ASM_OP_BLENDVPD:
    case X86ASM_OP_VBLENDVPS:
    case X86ASM_OP_VBLENDVPD: {
        bool legacy = instruction->opcode == X86ASM_OP_PBLENDVB || instruction->opcode == X86ASM_OP_BLENDVPS || instruction->opcode == X86ASM_OP_BLENDVPD;
        bool byte_blend = instruction->opcode == X86ASM_OP_PBLENDVB || instruction->opcode == X86ASM_OP_VPBLENDVB;
        unsigned vector_bytes = legacy ? 16u : width / 8u;
        unsigned element_bytes = byte_blend ? 1u : (instruction->opcode == X86ASM_OP_BLENDVPD || instruction->opcode == X86ASM_OP_VBLENDVPD ? 8u : 4u);
        const x86asm_argument *left_argument = legacy ? &instruction->arguments[0] : &instruction->arguments[1];
        const x86asm_argument *right_argument = legacy ? &instruction->arguments[1] : &instruction->arguments[2];
        const x86asm_argument *mask_argument = legacy ? &instruction->arguments[2] : &instruction->arguments[3];
        uint8_t left_vector[32];
        uint8_t right_vector[32];
        uint8_t mask_vector[32];
        uint8_t output_vector[32];
        if (vector_bytes > sizeof(left_vector) ||
            !read_vector_argument(cpu, instruction, left_argument, vector_bytes, left_vector) ||
            !read_vector_argument(cpu, instruction, right_argument, vector_bytes, right_vector) ||
            !read_vector_argument(cpu, instruction, mask_argument, vector_bytes, mask_vector)) result = X86EMU_ERR_MEMORY;
        else {
            if (byte_blend) vector_blend_bytes(output_vector, left_vector, right_vector, mask_vector, vector_bytes);
            else vector_blend_masked_elements(output_vector, left_vector, right_vector, mask_vector, vector_bytes, element_bytes);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, output_vector)) result = X86EMU_ERR_MEMORY;
            else { if (!legacy) zero_vector_upper_width(cpu, &instruction->arguments[0], vector_bytes); cpu->rip = next_rip; }
        }
        break;
    }
    case X86ASM_OP_PMOVSXBW:
    case X86ASM_OP_PMOVSXBD:
    case X86ASM_OP_PMOVSXBQ:
    case X86ASM_OP_PMOVSXWD:
    case X86ASM_OP_PMOVSXWQ:
    case X86ASM_OP_PMOVSXDQ:
    case X86ASM_OP_PMOVZXBW:
    case X86ASM_OP_PMOVZXBD:
    case X86ASM_OP_PMOVZXBQ:
    case X86ASM_OP_PMOVZXWD:
    case X86ASM_OP_PMOVZXWQ:
    case X86ASM_OP_PMOVZXDQ:
    case X86ASM_OP_VPMOVSXBW:
    case X86ASM_OP_VPMOVSXBD:
    case X86ASM_OP_VPMOVSXBQ:
    case X86ASM_OP_VPMOVSXWD:
    case X86ASM_OP_VPMOVSXWQ:
    case X86ASM_OP_VPMOVSXDQ:
    case X86ASM_OP_VPMOVZXBW:
    case X86ASM_OP_VPMOVZXBD:
    case X86ASM_OP_VPMOVZXBQ:
    case X86ASM_OP_VPMOVZXWD:
    case X86ASM_OP_VPMOVZXWQ:
    case X86ASM_OP_VPMOVZXDQ: {
        unsigned source_element_bytes;
        unsigned destination_element_bytes;
        bool sign_extend;
        bool is_vex = instruction->opcode >= X86ASM_OP_VPMOVSXBW && instruction->opcode <= X86ASM_OP_VPMOVZXDQ;
        unsigned output_bytes = is_vex ? width / 8u : 16u;
        unsigned source_bytes;
        uint8_t input[64] = { 0 };
        uint8_t output[64] = { 0 };
        if (!vector_extension_parameters(instruction->opcode, &source_element_bytes, &destination_element_bytes, &sign_extend)) result = X86EMU_ERR_UNSUPPORTED;
        else {
            source_bytes = (output_bytes / destination_element_bytes) * source_element_bytes;
            if (source_bytes > sizeof(input) || output_bytes > sizeof(output) ||
                !read_vector_argument(cpu, instruction, &instruction->arguments[1], source_bytes, input)) result = X86EMU_ERR_MEMORY;
            else {
                vector_extend_integer_elements(output, input, output_bytes, source_element_bytes, destination_element_bytes, sign_extend);
                if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], output_bytes, output)) result = X86EMU_ERR_MEMORY;
                else { if (is_vex) zero_vector_upper_width(cpu, &instruction->arguments[0], output_bytes); cpu->rip = next_rip; }
            }
        }
        break;
    }
    case X86ASM_OP_PSHUFB:
    case X86ASM_OP_VPSHUFB: {
        unsigned vector_bytes = instruction->opcode == X86ASM_OP_PSHUFB ? 16u : width / 8u;
        const x86asm_argument *source_argument = instruction->opcode == X86ASM_OP_PSHUFB ? &instruction->arguments[0] : &instruction->arguments[1];
        const x86asm_argument *control_argument = instruction->opcode == X86ASM_OP_PSHUFB ? &instruction->arguments[1] : &instruction->arguments[2];
        uint8_t source[64];
        uint8_t control[64];
        uint8_t output[64];
        if (vector_bytes > sizeof(output) ||
            !read_vector_argument(cpu, instruction, source_argument, vector_bytes, source) ||
            !read_vector_argument(cpu, instruction, control_argument, vector_bytes, control)) result = X86EMU_ERR_MEMORY;
        else {
            vector_shuffle_bytes(output, source, control, vector_bytes);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, output)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_PCMPEQB:
    case X86ASM_OP_PCMPEQW:
    case X86ASM_OP_PCMPEQD:
    case X86ASM_OP_PADDB:
    case X86ASM_OP_PADDW:
    case X86ASM_OP_PADDD:
    case X86ASM_OP_PSUBB:
    case X86ASM_OP_PSUBW:
    case X86ASM_OP_PSUBD:
    case X86ASM_OP_PMULLW:
    case X86ASM_OP_PMULHW:
    case X86ASM_OP_PMULHUW:
    case X86ASM_OP_PMULUDQ:
    case X86ASM_OP_PADDUSB:
    case X86ASM_OP_PADDUSW:
    case X86ASM_OP_PADDSB:
    case X86ASM_OP_PADDSW:
    case X86ASM_OP_PSUBUSB:
    case X86ASM_OP_PSUBUSW:
    case X86ASM_OP_PSUBSB:
    case X86ASM_OP_PSUBSW:
    case X86ASM_OP_PMINUB:
    case X86ASM_OP_PMAXUB:
    case X86ASM_OP_PMINSB:
    case X86ASM_OP_PMAXSB:
    case X86ASM_OP_PMINUW:
    case X86ASM_OP_PMAXUW:
    case X86ASM_OP_PMINSD:
    case X86ASM_OP_PMAXSD:
    case X86ASM_OP_PMINUD:
    case X86ASM_OP_PMAXUD:
    case X86ASM_OP_PMINSW:
    case X86ASM_OP_PMAXSW:
    case X86ASM_OP_PABSB:
    case X86ASM_OP_PABSW:
    case X86ASM_OP_PABSD:
    case X86ASM_OP_PSIGNB:
    case X86ASM_OP_PSIGNW:
    case X86ASM_OP_PSIGND:
    case X86ASM_OP_PHADDW:
    case X86ASM_OP_PHADDD:
    case X86ASM_OP_PHADDSW:
    case X86ASM_OP_PHSUBW:
    case X86ASM_OP_PHSUBD:
    case X86ASM_OP_PHSUBSW:
    case X86ASM_OP_PMADDUBSW:
    case X86ASM_OP_PMADDWD:
    case X86ASM_OP_PMULDQ:
    case X86ASM_OP_PAVGB:
    case X86ASM_OP_PAVGW:
    case X86ASM_OP_PSADBW:
    case X86ASM_OP_PUNPCKLBW:
    case X86ASM_OP_PUNPCKLWD:
    case X86ASM_OP_PUNPCKLDQ:
    case X86ASM_OP_PUNPCKHBW:
    case X86ASM_OP_PUNPCKHWD:
    case X86ASM_OP_PUNPCKHDQ:
    case X86ASM_OP_PACKSSWB:
    case X86ASM_OP_PACKSSDW:
    case X86ASM_OP_PACKUSWB:
    case X86ASM_OP_PAND:
    case X86ASM_OP_POR:
    case X86ASM_OP_PXOR: {
        uint8_t left_vector[16];
        uint8_t right_vector[16];
        uint8_t output_vector[16];
        if (!read_vector_argument(cpu, instruction, &instruction->arguments[0], 16, left_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[1], 16, right_vector)) result = X86EMU_ERR_MEMORY;
        else {
            if (instruction->opcode == X86ASM_OP_PADDB) vector_add_i8(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PADDW) vector_add_i16(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PADDD) vector_add_i32(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PSUBB) vector_sub_i8(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PSUBW) vector_sub_i16(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PSUBD) vector_sub_i32(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PMULUDQ) vector_mul_u32_even(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PMULLW || instruction->opcode == X86ASM_OP_PMULHW || instruction->opcode == X86ASM_OP_PMULHUW) vector_mul_words(output_vector, left_vector, right_vector, 16, instruction->opcode);
            else if (instruction->opcode == X86ASM_OP_PADDUSB) vector_saturating_add_sub(output_vector, left_vector, right_vector, 16, 1, false, false);
            else if (instruction->opcode == X86ASM_OP_PADDUSW) vector_saturating_add_sub(output_vector, left_vector, right_vector, 16, 2, false, false);
            else if (instruction->opcode == X86ASM_OP_PADDSB) vector_saturating_add_sub(output_vector, left_vector, right_vector, 16, 1, true, false);
            else if (instruction->opcode == X86ASM_OP_PADDSW) vector_saturating_add_sub(output_vector, left_vector, right_vector, 16, 2, true, false);
            else if (instruction->opcode == X86ASM_OP_PSUBUSB) vector_saturating_add_sub(output_vector, left_vector, right_vector, 16, 1, false, true);
            else if (instruction->opcode == X86ASM_OP_PSUBUSW) vector_saturating_add_sub(output_vector, left_vector, right_vector, 16, 2, false, true);
            else if (instruction->opcode == X86ASM_OP_PSUBSB) vector_saturating_add_sub(output_vector, left_vector, right_vector, 16, 1, true, true);
            else if (instruction->opcode == X86ASM_OP_PSUBSW) vector_saturating_add_sub(output_vector, left_vector, right_vector, 16, 2, true, true);
            else if (instruction->opcode == X86ASM_OP_PMINUB) vector_minmax(output_vector, left_vector, right_vector, 16, 1, false, false);
            else if (instruction->opcode == X86ASM_OP_PMAXUB) vector_minmax(output_vector, left_vector, right_vector, 16, 1, false, true);
            else if (instruction->opcode == X86ASM_OP_PMINSB) vector_minmax(output_vector, left_vector, right_vector, 16, 1, true, false);
            else if (instruction->opcode == X86ASM_OP_PMAXSB) vector_minmax(output_vector, left_vector, right_vector, 16, 1, true, true);
            else if (instruction->opcode == X86ASM_OP_PMINUW) vector_minmax(output_vector, left_vector, right_vector, 16, 2, false, false);
            else if (instruction->opcode == X86ASM_OP_PMAXUW) vector_minmax(output_vector, left_vector, right_vector, 16, 2, false, true);
            else if (instruction->opcode == X86ASM_OP_PMINSD) vector_minmax(output_vector, left_vector, right_vector, 16, 4, true, false);
            else if (instruction->opcode == X86ASM_OP_PMAXSD) vector_minmax(output_vector, left_vector, right_vector, 16, 4, true, true);
            else if (instruction->opcode == X86ASM_OP_PMINUD) vector_minmax(output_vector, left_vector, right_vector, 16, 4, false, false);
            else if (instruction->opcode == X86ASM_OP_PMAXUD) vector_minmax(output_vector, left_vector, right_vector, 16, 4, false, true);
            else if (instruction->opcode == X86ASM_OP_PMINSW) vector_minmax(output_vector, left_vector, right_vector, 16, 2, true, false);
            else if (instruction->opcode == X86ASM_OP_PMAXSW) vector_minmax(output_vector, left_vector, right_vector, 16, 2, true, true);
            else if (instruction->opcode == X86ASM_OP_PABSB) vector_abs_signed(output_vector, right_vector, 16, 1);
            else if (instruction->opcode == X86ASM_OP_PABSW) vector_abs_signed(output_vector, right_vector, 16, 2);
            else if (instruction->opcode == X86ASM_OP_PABSD) vector_abs_signed(output_vector, right_vector, 16, 4);
            else if (instruction->opcode == X86ASM_OP_PSIGNB) vector_sign(output_vector, left_vector, right_vector, 16, 1);
            else if (instruction->opcode == X86ASM_OP_PSIGNW) vector_sign(output_vector, left_vector, right_vector, 16, 2);
            else if (instruction->opcode == X86ASM_OP_PSIGND) vector_sign(output_vector, left_vector, right_vector, 16, 4);
            else if (instruction->opcode == X86ASM_OP_PHADDW) vector_horizontal_add(output_vector, left_vector, right_vector, 16, 2, false);
            else if (instruction->opcode == X86ASM_OP_PHADDD) vector_horizontal_add(output_vector, left_vector, right_vector, 16, 4, false);
            else if (instruction->opcode == X86ASM_OP_PHADDSW) vector_horizontal_add(output_vector, left_vector, right_vector, 16, 2, true);
            else if (instruction->opcode == X86ASM_OP_PHSUBW) vector_horizontal_sub(output_vector, left_vector, right_vector, 16, 2, false);
            else if (instruction->opcode == X86ASM_OP_PHSUBD) vector_horizontal_sub(output_vector, left_vector, right_vector, 16, 4, false);
            else if (instruction->opcode == X86ASM_OP_PHSUBSW) vector_horizontal_sub(output_vector, left_vector, right_vector, 16, 2, true);
            else if (instruction->opcode == X86ASM_OP_PMADDUBSW) vector_madd_unsigned_signed_bytes(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PMADDWD) vector_madd_signed_words(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PMULDQ) vector_multiply_even_signed_dwords(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PAVGB) vector_average(output_vector, left_vector, right_vector, 16, 1);
            else if (instruction->opcode == X86ASM_OP_PAVGW) vector_average(output_vector, left_vector, right_vector, 16, 2);
            else if (instruction->opcode == X86ASM_OP_PSADBW) vector_sad_bytes(output_vector, left_vector, right_vector, 16);
            else if (instruction->opcode == X86ASM_OP_PUNPCKLBW) vector_unpack(output_vector, left_vector, right_vector, 1, false);
            else if (instruction->opcode == X86ASM_OP_PUNPCKLWD) vector_unpack(output_vector, left_vector, right_vector, 2, false);
            else if (instruction->opcode == X86ASM_OP_PUNPCKLDQ) vector_unpack(output_vector, left_vector, right_vector, 4, false);
            else if (instruction->opcode == X86ASM_OP_PUNPCKHBW) vector_unpack(output_vector, left_vector, right_vector, 1, true);
            else if (instruction->opcode == X86ASM_OP_PUNPCKHWD) vector_unpack(output_vector, left_vector, right_vector, 2, true);
            else if (instruction->opcode == X86ASM_OP_PUNPCKHDQ) vector_unpack(output_vector, left_vector, right_vector, 4, true);
            else if (instruction->opcode == X86ASM_OP_PACKSSWB || instruction->opcode == X86ASM_OP_PACKSSDW || instruction->opcode == X86ASM_OP_PACKUSWB) vector_pack(output_vector, left_vector, right_vector, instruction->opcode);
            else if (instruction->opcode == X86ASM_OP_PCMPEQB) vector_compare_equal_elements(output_vector, left_vector, right_vector, 16, 1);
            else if (instruction->opcode == X86ASM_OP_PCMPEQW) vector_compare_equal_elements(output_vector, left_vector, right_vector, 16, 2);
            else if (instruction->opcode == X86ASM_OP_PCMPEQD) vector_compare_equal_elements(output_vector, left_vector, right_vector, 16, 4);
            else if (instruction->opcode == X86ASM_OP_PAND) vector_bitwise(output_vector, left_vector, right_vector, 16, X86ASM_OP_VAND);
            else if (instruction->opcode == X86ASM_OP_POR) vector_bitwise(output_vector, left_vector, right_vector, 16, X86ASM_OP_VOR);
            else vector_bitwise(output_vector, left_vector, right_vector, 16, X86ASM_OP_VXOR);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, output_vector)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_VPSLLW:
    case X86ASM_OP_VPSLLD:
    case X86ASM_OP_VPSLLQ:
    case X86ASM_OP_VPSRLW:
    case X86ASM_OP_VPSRLD:
    case X86ASM_OP_VPSRLQ:
    case X86ASM_OP_VPSRAW:
    case X86ASM_OP_VPSRAD:
    case X86ASM_OP_VPSLLDQ:
    case X86ASM_OP_VPSRLDQ: {
        unsigned vector_bytes = width / 8u;
        uint8_t input[64];
        uint8_t output[64];
        if (vector_bytes > sizeof(input) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[1], vector_bytes, input) ||
            !read_argument(cpu, instruction, &instruction->arguments[2], 8, &right)) result = X86EMU_ERR_MEMORY;
        else {
            unsigned count = (unsigned)right & 0xFFu;
            if (instruction->opcode == X86ASM_OP_VPSLLDQ) vector_shift_bytes_lanes(output, input, vector_bytes, count, true);
            else if (instruction->opcode == X86ASM_OP_VPSRLDQ) vector_shift_bytes_lanes(output, input, vector_bytes, count, false);
            else {
                unsigned element_bytes = instruction->opcode == X86ASM_OP_VPSLLQ || instruction->opcode == X86ASM_OP_VPSRLQ ? 8u :
                                         instruction->opcode == X86ASM_OP_VPSLLD || instruction->opcode == X86ASM_OP_VPSRLD || instruction->opcode == X86ASM_OP_VPSRAD ? 4u : 2u;
                bool left_shift = instruction->opcode == X86ASM_OP_VPSLLW || instruction->opcode == X86ASM_OP_VPSLLD || instruction->opcode == X86ASM_OP_VPSLLQ;
                bool arithmetic = instruction->opcode == X86ASM_OP_VPSRAW || instruction->opcode == X86ASM_OP_VPSRAD;
                vector_shift_elements(output, input, vector_bytes, element_bytes, count, left_shift, arithmetic);
            }
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, output)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_VADDSS:
    case X86ASM_OP_VSUBSS:
    case X86ASM_OP_VMULSS:
    case X86ASM_OP_VDIVSS:
    case X86ASM_OP_VMINSS:
    case X86ASM_OP_VMAXSS:
    case X86ASM_OP_VADDSD:
    case X86ASM_OP_VSUBSD:
    case X86ASM_OP_VMULSD:
    case X86ASM_OP_VDIVSD:
    case X86ASM_OP_VMINSD:
    case X86ASM_OP_VMAXSD: {
        unsigned scalar_bytes = (instruction->opcode == X86ASM_OP_VADDSS || instruction->opcode == X86ASM_OP_VSUBSS || instruction->opcode == X86ASM_OP_VMULSS || instruction->opcode == X86ASM_OP_VDIVSS || instruction->opcode == X86ASM_OP_VMINSS || instruction->opcode == X86ASM_OP_VMAXSS) ? 4u : 8u;
        uint8_t first[16];
        uint8_t source[8];
        if (!read_vector_argument(cpu, instruction, &instruction->arguments[1], 16, first) ||
            !read_scalar_vector_argument(cpu, instruction, &instruction->arguments[2], scalar_bytes, source)) result = X86EMU_ERR_MEMORY;
        else {
            if (scalar_bytes == 4u) {
                float a, b, r;
                memcpy(&a, first, sizeof(a)); memcpy(&b, source, sizeof(b));
                if (instruction->opcode == X86ASM_OP_VADDSS) r = a + b; else if (instruction->opcode == X86ASM_OP_VSUBSS) r = a - b; else if (instruction->opcode == X86ASM_OP_VMULSS) r = a * b; else if (instruction->opcode == X86ASM_OP_VDIVSS) r = a / b; else if (isnan(a) || isnan(b) || a == b) r = b; else r = instruction->opcode == X86ASM_OP_VMINSS ? (a < b ? a : b) : (a > b ? a : b);
                memcpy(first, &r, sizeof(r));
            } else {
                double a, b, r;
                memcpy(&a, first, sizeof(a)); memcpy(&b, source, sizeof(b));
                if (instruction->opcode == X86ASM_OP_VADDSD) r = a + b; else if (instruction->opcode == X86ASM_OP_VSUBSD) r = a - b; else if (instruction->opcode == X86ASM_OP_VMULSD) r = a * b; else if (instruction->opcode == X86ASM_OP_VDIVSD) r = a / b; else if (isnan(a) || isnan(b) || a == b) r = b; else r = instruction->opcode == X86ASM_OP_VMINSD ? (a < b ? a : b) : (a > b ? a : b);
                memcpy(first, &r, sizeof(r));
            }
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], 16, first)) result = X86EMU_ERR_MEMORY;
            else { zero_vector_upper_128(cpu, &instruction->arguments[0]); cpu->rip = next_rip; }
        }
        break;
    }
    case X86ASM_OP_VXORPS:
    case X86ASM_OP_VXOR:
    case X86ASM_OP_VAND:
    case X86ASM_OP_VANDN:
    case X86ASM_OP_VOR:
    case X86ASM_OP_VADDPS:
    case X86ASM_OP_VADDPD:
    case X86ASM_OP_VMINPS:
    case X86ASM_OP_VMAXPS:
    case X86ASM_OP_VCMPPS:
    case X86ASM_OP_VSUBPS:
    case X86ASM_OP_VMULPS:
    case X86ASM_OP_VDIVPS:
    case X86ASM_OP_VSUBPD:
    case X86ASM_OP_VMINPD:
    case X86ASM_OP_VMAXPD:
    case X86ASM_OP_VCMPPD:
    case X86ASM_OP_VMULPD:
    case X86ASM_OP_VDIVPD:
    case X86ASM_OP_VPADDB:
    case X86ASM_OP_VPADDW:
    case X86ASM_OP_VPADDD:
    case X86ASM_OP_VPSUBB:
    case X86ASM_OP_VPSUBD:
    case X86ASM_OP_VPSUBW:
    case X86ASM_OP_VPADDQ:
    case X86ASM_OP_VPSUBQ:
    case X86ASM_OP_VPMULLW:
    case X86ASM_OP_VPMULHW:
    case X86ASM_OP_VPMULHUW:
    case X86ASM_OP_VPMULUDQ:
    case X86ASM_OP_VPADDUSB:
    case X86ASM_OP_VPADDUSW:
    case X86ASM_OP_VPADDSB:
    case X86ASM_OP_VPADDSW:
    case X86ASM_OP_VPSUBUSB:
    case X86ASM_OP_VPSUBUSW:
    case X86ASM_OP_VPSUBSB:
    case X86ASM_OP_VPSUBSW:
    case X86ASM_OP_VPMINUB:
    case X86ASM_OP_VPMAXUB:
    case X86ASM_OP_VPMINSW:
    case X86ASM_OP_VPMAXSW:
    case X86ASM_OP_VPMINSB:
    case X86ASM_OP_VPMAXSB:
    case X86ASM_OP_VPMINUW:
    case X86ASM_OP_VPMAXUW:
    case X86ASM_OP_VPMINSD:
    case X86ASM_OP_VPMAXSD:
    case X86ASM_OP_VPMINUD:
    case X86ASM_OP_VPMAXUD:
    case X86ASM_OP_VPABSB:
    case X86ASM_OP_VPABSW:
    case X86ASM_OP_VPABSD:
    case X86ASM_OP_VPSIGNB:
    case X86ASM_OP_VPSIGNW:
    case X86ASM_OP_VPSIGND:
    case X86ASM_OP_VPHADDW:
    case X86ASM_OP_VPHADDD:
    case X86ASM_OP_VPHADDSW:
    case X86ASM_OP_VPHSUBW:
    case X86ASM_OP_VPHSUBD:
    case X86ASM_OP_VPHSUBSW:
    case X86ASM_OP_VPMADDUBSW:
    case X86ASM_OP_VPMADDWD:
    case X86ASM_OP_VPMULDQ:
    case X86ASM_OP_VPAVGB:
    case X86ASM_OP_VPAVGW:
    case X86ASM_OP_VPSADBW:
    case X86ASM_OP_VPUNPCKLBW:
    case X86ASM_OP_VPUNPCKLWD:
    case X86ASM_OP_VPUNPCKLDQ:
    case X86ASM_OP_VPUNPCKHBW:
    case X86ASM_OP_VPUNPCKHWD:
    case X86ASM_OP_VPUNPCKHDQ:
    case X86ASM_OP_VPACKSSWB:
    case X86ASM_OP_VPACKSSDW:
    case X86ASM_OP_VPACKUSWB:
    case X86ASM_OP_VPSLLVD:
    case X86ASM_OP_VPSRLVD:
    case X86ASM_OP_VPSRAVD:
    case X86ASM_OP_VPSLLVQ:
    case X86ASM_OP_VPSRLVQ: {
        unsigned vector_bytes = width / 8;
        uint8_t left_vector[64];
        uint8_t right_vector[64];
        uint8_t output_vector[64];
        bool legacy_two_operand = instruction->arguments[2].kind == X86ASM_ARG_NONE;
        if (vector_bytes > sizeof(output_vector) ||
            !read_vector_argument(cpu, instruction,
                                  legacy_two_operand ? &instruction->arguments[0] : &instruction->arguments[1],
                                  vector_bytes, left_vector) ||
            !read_vector_argument(cpu, instruction, &instruction->arguments[legacy_two_operand ? 1 : 2],
                                  vector_bytes, right_vector)) result = X86EMU_ERR_MEMORY;
        else {
            if (instruction->opcode == X86ASM_OP_VPABSB) vector_abs_signed(output_vector, right_vector, vector_bytes, 1);
            else if (instruction->opcode == X86ASM_OP_VPABSW) vector_abs_signed(output_vector, right_vector, vector_bytes, 2);
            else if (instruction->opcode == X86ASM_OP_VPABSD) vector_abs_signed(output_vector, right_vector, vector_bytes, 4);
            else if (instruction->opcode == X86ASM_OP_VPSIGNB) vector_sign(output_vector, left_vector, right_vector, vector_bytes, 1);
            else if (instruction->opcode == X86ASM_OP_VPSIGNW) vector_sign(output_vector, left_vector, right_vector, vector_bytes, 2);
            else if (instruction->opcode == X86ASM_OP_VPSIGND) vector_sign(output_vector, left_vector, right_vector, vector_bytes, 4);
            else if (instruction->opcode == X86ASM_OP_VPHADDW) vector_horizontal_add(output_vector, left_vector, right_vector, vector_bytes, 2, false);
            else if (instruction->opcode == X86ASM_OP_VPHADDD) vector_horizontal_add(output_vector, left_vector, right_vector, vector_bytes, 4, false);
            else if (instruction->opcode == X86ASM_OP_VPHADDSW) vector_horizontal_add(output_vector, left_vector, right_vector, vector_bytes, 2, true);
            else if (instruction->opcode == X86ASM_OP_VPHSUBW) vector_horizontal_sub(output_vector, left_vector, right_vector, vector_bytes, 2, false);
            else if (instruction->opcode == X86ASM_OP_VPHSUBD) vector_horizontal_sub(output_vector, left_vector, right_vector, vector_bytes, 4, false);
            else if (instruction->opcode == X86ASM_OP_VPHSUBSW) vector_horizontal_sub(output_vector, left_vector, right_vector, vector_bytes, 2, true);
            else if (instruction->opcode == X86ASM_OP_VPMADDUBSW) vector_madd_unsigned_signed_bytes(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPMADDWD) vector_madd_signed_words(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPMULDQ) vector_multiply_even_signed_dwords(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPADDB) vector_add_i8(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPADDW) vector_add_i16(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPSUBB) vector_sub_i8(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPSUBW) vector_sub_i16(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VADDPS) vector_add_f32(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VSUBPS || instruction->opcode == X86ASM_OP_VMULPS || instruction->opcode == X86ASM_OP_VDIVPS) vector_arith_f32(output_vector, left_vector, right_vector, vector_bytes, instruction->opcode);
            else if (instruction->opcode == X86ASM_OP_VMINPS || instruction->opcode == X86ASM_OP_VMAXPS) vector_minmax_f32(output_vector, left_vector, right_vector, vector_bytes, instruction->opcode == X86ASM_OP_VMAXPS);
            else if (instruction->opcode == X86ASM_OP_VCMPPS) vector_compare_fp(output_vector, left_vector, right_vector, vector_bytes, 4u, (unsigned)instruction->arguments[3].value.immediate);
            else if (instruction->opcode == X86ASM_OP_VADDPD) vector_add_f64(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VSUBPD || instruction->opcode == X86ASM_OP_VMULPD || instruction->opcode == X86ASM_OP_VDIVPD) vector_arith_f64(output_vector, left_vector, right_vector, vector_bytes, instruction->opcode);
            else if (instruction->opcode == X86ASM_OP_VMINPD || instruction->opcode == X86ASM_OP_VMAXPD) vector_minmax_f64(output_vector, left_vector, right_vector, vector_bytes, instruction->opcode == X86ASM_OP_VMAXPD);
            else if (instruction->opcode == X86ASM_OP_VCMPPD) vector_compare_fp(output_vector, left_vector, right_vector, vector_bytes, 8u, (unsigned)instruction->arguments[3].value.immediate);
            else if (instruction->opcode == X86ASM_OP_VPADDD) vector_add_i32(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPSUBD) vector_sub_i32(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPADDQ) vector_add_i64(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPSUBQ) vector_sub_i64(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPMULUDQ) vector_mul_u32_even(output_vector, left_vector, right_vector, vector_bytes);
            else if (instruction->opcode == X86ASM_OP_VPMULLW || instruction->opcode == X86ASM_OP_VPMULHW || instruction->opcode == X86ASM_OP_VPMULHUW) {
                x86asm_opcode scalar_opcode = instruction->opcode == X86ASM_OP_VPMULLW ? X86ASM_OP_PMULLW :
                                               instruction->opcode == X86ASM_OP_VPMULHW ? X86ASM_OP_PMULHW : X86ASM_OP_PMULHUW;
                vector_mul_words(output_vector, left_vector, right_vector, vector_bytes, scalar_opcode);
            } else if (instruction->opcode == X86ASM_OP_VPADDUSB || instruction->opcode == X86ASM_OP_VPADDUSW || instruction->opcode == X86ASM_OP_VPADDSB || instruction->opcode == X86ASM_OP_VPADDSW ||
                       instruction->opcode == X86ASM_OP_VPSUBUSB || instruction->opcode == X86ASM_OP_VPSUBUSW || instruction->opcode == X86ASM_OP_VPSUBSB || instruction->opcode == X86ASM_OP_VPSUBSW) {
                unsigned element_bytes = (instruction->opcode == X86ASM_OP_VPADDUSW || instruction->opcode == X86ASM_OP_VPADDSW || instruction->opcode == X86ASM_OP_VPSUBUSW || instruction->opcode == X86ASM_OP_VPSUBSW) ? 2u : 1u;
                bool signed_mode = instruction->opcode == X86ASM_OP_VPADDSB || instruction->opcode == X86ASM_OP_VPADDSW || instruction->opcode == X86ASM_OP_VPSUBSB || instruction->opcode == X86ASM_OP_VPSUBSW;
                bool subtract = instruction->opcode == X86ASM_OP_VPSUBUSB || instruction->opcode == X86ASM_OP_VPSUBUSW || instruction->opcode == X86ASM_OP_VPSUBSB || instruction->opcode == X86ASM_OP_VPSUBSW;
                vector_saturating_add_sub(output_vector, left_vector, right_vector, vector_bytes, element_bytes, signed_mode, subtract);
            } else if (instruction->opcode == X86ASM_OP_VPMINUB || instruction->opcode == X86ASM_OP_VPMAXUB || instruction->opcode == X86ASM_OP_VPMINSW || instruction->opcode == X86ASM_OP_VPMAXSW ||
                       instruction->opcode == X86ASM_OP_VPMINSB || instruction->opcode == X86ASM_OP_VPMAXSB || instruction->opcode == X86ASM_OP_VPMINUW || instruction->opcode == X86ASM_OP_VPMAXUW ||
                       instruction->opcode == X86ASM_OP_VPMINSD || instruction->opcode == X86ASM_OP_VPMAXSD || instruction->opcode == X86ASM_OP_VPMINUD || instruction->opcode == X86ASM_OP_VPMAXUD) {
                unsigned element_bytes = (instruction->opcode == X86ASM_OP_VPMINSD || instruction->opcode == X86ASM_OP_VPMAXSD || instruction->opcode == X86ASM_OP_VPMINUD || instruction->opcode == X86ASM_OP_VPMAXUD) ? 4u :
                                         (instruction->opcode == X86ASM_OP_VPMINUW || instruction->opcode == X86ASM_OP_VPMAXUW || instruction->opcode == X86ASM_OP_VPMINSW || instruction->opcode == X86ASM_OP_VPMAXSW) ? 2u : 1u;
                bool signed_mode = instruction->opcode == X86ASM_OP_VPMINSB || instruction->opcode == X86ASM_OP_VPMAXSB || instruction->opcode == X86ASM_OP_VPMINSW || instruction->opcode == X86ASM_OP_VPMAXSW || instruction->opcode == X86ASM_OP_VPMINSD || instruction->opcode == X86ASM_OP_VPMAXSD;
                bool maximum = instruction->opcode == X86ASM_OP_VPMAXUB || instruction->opcode == X86ASM_OP_VPMAXSW || instruction->opcode == X86ASM_OP_VPMAXSB || instruction->opcode == X86ASM_OP_VPMAXUW || instruction->opcode == X86ASM_OP_VPMAXSD || instruction->opcode == X86ASM_OP_VPMAXUD;
                vector_minmax(output_vector, left_vector, right_vector, vector_bytes, element_bytes, signed_mode, maximum);
            } else if (instruction->opcode == X86ASM_OP_VPAVGB || instruction->opcode == X86ASM_OP_VPAVGW) {
                vector_average(output_vector, left_vector, right_vector, vector_bytes, instruction->opcode == X86ASM_OP_VPAVGW ? 2u : 1u);
            } else if (instruction->opcode == X86ASM_OP_VPSADBW) {
                vector_sad_bytes(output_vector, left_vector, right_vector, vector_bytes);
            } else if (instruction->opcode == X86ASM_OP_VPUNPCKLBW || instruction->opcode == X86ASM_OP_VPUNPCKLWD || instruction->opcode == X86ASM_OP_VPUNPCKLDQ ||
                       instruction->opcode == X86ASM_OP_VPUNPCKHBW || instruction->opcode == X86ASM_OP_VPUNPCKHWD || instruction->opcode == X86ASM_OP_VPUNPCKHDQ) {
                unsigned element_bytes = (instruction->opcode == X86ASM_OP_VPUNPCKLBW || instruction->opcode == X86ASM_OP_VPUNPCKHBW) ? 1u :
                                         (instruction->opcode == X86ASM_OP_VPUNPCKLWD || instruction->opcode == X86ASM_OP_VPUNPCKHWD) ? 2u : 4u;
                bool high_half = instruction->opcode == X86ASM_OP_VPUNPCKHBW || instruction->opcode == X86ASM_OP_VPUNPCKHWD || instruction->opcode == X86ASM_OP_VPUNPCKHDQ;
                vector_unpack_lanes(output_vector, left_vector, right_vector, vector_bytes, element_bytes, high_half);
            } else if (instruction->opcode == X86ASM_OP_VPACKSSWB || instruction->opcode == X86ASM_OP_VPACKSSDW || instruction->opcode == X86ASM_OP_VPACKUSWB) {
                x86asm_opcode scalar_opcode = instruction->opcode == X86ASM_OP_VPACKSSWB ? X86ASM_OP_PACKSSWB :
                                               instruction->opcode == X86ASM_OP_VPACKSSDW ? X86ASM_OP_PACKSSDW : X86ASM_OP_PACKUSWB;
                vector_pack_lanes(output_vector, left_vector, right_vector, vector_bytes, scalar_opcode);
            } else if (instruction->opcode == X86ASM_OP_VPSLLVD || instruction->opcode == X86ASM_OP_VPSRLVD || instruction->opcode == X86ASM_OP_VPSRAVD) {
                vector_shift_variable(output_vector, left_vector, right_vector, vector_bytes, 4,
                                      instruction->opcode == X86ASM_OP_VPSLLVD, instruction->opcode == X86ASM_OP_VPSRAVD);
            } else if (instruction->opcode == X86ASM_OP_VPSLLVQ || instruction->opcode == X86ASM_OP_VPSRLVQ) {
                vector_shift_variable(output_vector, left_vector, right_vector, vector_bytes, 8,
                                      instruction->opcode == X86ASM_OP_VPSLLVQ, false);
            } else vector_bitwise(output_vector, left_vector, right_vector, vector_bytes, instruction->opcode);
            if (!write_vector_argument(cpu, instruction, &instruction->arguments[0], vector_bytes, output_vector)) result = X86EMU_ERR_MEMORY;
            else { zero_vector_upper_width(cpu, &instruction->arguments[0], vector_bytes); cpu->rip = next_rip; }
        }
        break;
    }
    case X86ASM_OP_MOVBE:
        if (!read_argument(cpu, instruction, &instruction->arguments[1], width, &right) ||
            !write_argument(cpu, instruction, &instruction->arguments[0], width,
                            byte_swap_width(right, width))) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    case X86ASM_OP_MOV:
        if (!read_argument(cpu, instruction, &instruction->arguments[1], width, &right) ||
            !write_argument(cpu, instruction, &instruction->arguments[0], width, right)) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    case X86ASM_OP_MOVSX:
    case X86ASM_OP_MOVZX: {
        unsigned source_width = ((instruction->encoded_opcode & 0xffu) == 0xB6 ||
                                 (instruction->encoded_opcode & 0xffu) == 0xBE) ? 8u : 16u;
        if (!read_argument(cpu, instruction, &instruction->arguments[1], source_width, &right)) result = X86EMU_ERR_MEMORY;
        else {
            if (instruction->opcode == X86ASM_OP_MOVSX) {
                if (source_width == 8) right = (uint64_t)(int64_t)(int8_t)right;
                else right = (uint64_t)(int64_t)(int16_t)right;
            }
            if (!write_argument(cpu, instruction, &instruction->arguments[0], width, right)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_LEA:
        if (instruction->arguments[1].kind != X86ASM_ARG_MEMORY ||
            !memory_address(cpu, instruction, &instruction->arguments[1].value.memory, &value) ||
            !write_argument(cpu, instruction, &instruction->arguments[0], width, value)) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    case X86ASM_OP_ADD:
    case X86ASM_OP_ADC:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], width, &right)) result = X86EMU_ERR_MEMORY;
        else {
            value = add_values(cpu, left, right, width, instruction->opcode == X86ASM_OP_ADC && get_flag(cpu, X86EMU_FLAG_CF));
            if (!write_argument(cpu, instruction, &instruction->arguments[0], width, value)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_SUB:
    case X86ASM_OP_SBB:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], width, &right)) result = X86EMU_ERR_MEMORY;
        else {
            value = subtract_values(cpu, left, right, width, instruction->opcode == X86ASM_OP_SBB && get_flag(cpu, X86EMU_FLAG_CF));
            if (!write_argument(cpu, instruction, &instruction->arguments[0], width, value)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_AND:
    case X86ASM_OP_OR:
    case X86ASM_OP_XOR:
    case X86ASM_OP_TEST:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], width, &right)) result = X86EMU_ERR_MEMORY;
        else {
            if (instruction->opcode == X86ASM_OP_AND || instruction->opcode == X86ASM_OP_TEST) value = left & right;
            else if (instruction->opcode == X86ASM_OP_OR) value = left | right;
            else value = left ^ right;
            set_logic_flags(cpu, value, width);
            if (instruction->opcode != X86ASM_OP_TEST && !write_argument(cpu, instruction, &instruction->arguments[0], width, value)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_CMP:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], width, &right)) result = X86EMU_ERR_MEMORY;
        else { (void)subtract_values(cpu, left, right, width, false); cpu->rip = next_rip; }
        break;
    case X86ASM_OP_INC:
    case X86ASM_OP_DEC:
    case X86ASM_OP_NEG:
    case X86ASM_OP_NOT:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left)) result = X86EMU_ERR_MEMORY;
        else {
            bool old_cf = get_flag(cpu, X86EMU_FLAG_CF);
            if (instruction->opcode == X86ASM_OP_INC) value = add_values(cpu, left, 1, width, false);
            else if (instruction->opcode == X86ASM_OP_DEC) value = subtract_values(cpu, left, 1, width, false);
            else if (instruction->opcode == X86ASM_OP_NEG) value = subtract_values(cpu, 0, left, width, false);
            else value = (~left) & width_mask(width);
            if (instruction->opcode == X86ASM_OP_INC || instruction->opcode == X86ASM_OP_DEC) set_flag(cpu, X86EMU_FLAG_CF, old_cf);
            if (!write_argument(cpu, instruction, &instruction->arguments[0], width, value)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_ROL:
    case X86ASM_OP_ROR:
    case X86ASM_OP_RCL:
    case X86ASM_OP_RCR:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], 8, &right)) result = X86EMU_ERR_MEMORY;
        else {
            unsigned count_mask = width == 64u ? 63u : 31u;
            unsigned count = (unsigned)right & count_mask;
            if (instruction->opcode == X86ASM_OP_RCL || instruction->opcode == X86ASM_OP_RCR) {
                count %= width + 1u;
            }
            value = left & width_mask(width);
            if (count == 0) cpu->rip = next_rip;
            else {
                bool carry = get_flag(cpu, X86EMU_FLAG_CF);
                for (unsigned i = 0; i < count; ++i) {
                    if (instruction->opcode == X86ASM_OP_ROL || instruction->opcode == X86ASM_OP_RCL) {
                        bool outgoing = ((value >> (width - 1u)) & 1u) != 0;
                        value = ((value << 1u) & width_mask(width)) | (instruction->opcode == X86ASM_OP_RCL ? (carry ? 1u : 0u) : (value >> (width - 1u)));
                        carry = outgoing;
                    } else {
                        bool outgoing = (value & 1u) != 0;
                        value = (value >> 1u) | ((instruction->opcode == X86ASM_OP_RCR ? (carry ? 1u : 0u) : (value & 1u)) << (width - 1u));
                        carry = outgoing;
                    }
                }
                set_flag(cpu, X86EMU_FLAG_CF, carry);
                if (count == 1u) {
                    bool msb = ((value >> (width - 1u)) & 1u) != 0;
                    bool next = width > 1u && ((value >> (width - 2u)) & 1u) != 0;
                    set_flag(cpu, X86EMU_FLAG_OF, msb ^ (instruction->opcode == X86ASM_OP_ROL || instruction->opcode == X86ASM_OP_RCL ? carry : next));
                }
                if (!write_argument(cpu, instruction, &instruction->arguments[0], width, value)) result = X86EMU_ERR_MEMORY;
                else cpu->rip = next_rip;
            }
        }
        break;
    case X86ASM_OP_SHLD:
    case X86ASM_OP_SHRD:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], width, &right) ||
            !read_argument(cpu, instruction, &instruction->arguments[2], 8, &value)) result = X86EMU_ERR_MEMORY;
        else {
            unsigned count_mask = width == 64u ? 63u : 31u;
            unsigned count = (unsigned)value & count_mask;
            if (count == 0) cpu->rip = next_rip;
            else {
                uint64_t mask = width_mask(width);
                if (instruction->opcode == X86ASM_OP_SHLD) {
                    value = ((left << count) | (right >> (width - count))) & mask;
                    set_flag(cpu, X86EMU_FLAG_CF, ((left >> (width - count)) & 1u) != 0);
                    if (count == 1u) set_flag(cpu, X86EMU_FLAG_OF,
                                               ((value >> (width - 1u)) & 1u) != ((cpu->rflags & X86EMU_FLAG_CF) != 0));
                } else {
                    value = ((left >> count) | (right << (width - count))) & mask;
                    set_flag(cpu, X86EMU_FLAG_CF, ((left >> (count - 1u)) & 1u) != 0);
                    if (count == 1u) set_flag(cpu, X86EMU_FLAG_OF,
                                               ((left >> (width - 1u)) & 1u) != ((value >> (width - 1u)) & 1u));
                }
                set_flag(cpu, X86EMU_FLAG_ZF, value == 0);
                set_flag(cpu, X86EMU_FLAG_SF, ((value >> (width - 1u)) & 1u) != 0);
                set_flag(cpu, X86EMU_FLAG_PF, even_parity((uint8_t)value));
                if (!write_argument(cpu, instruction, &instruction->arguments[0], width, value)) result = X86EMU_ERR_MEMORY;
                else cpu->rip = next_rip;
            }
        }
        break;
    case X86ASM_OP_SHL:
    case X86ASM_OP_SHR:
    case X86ASM_OP_SAR:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], 8, &right)) result = X86EMU_ERR_MEMORY;
        else {
            value = shift_value(cpu, instruction->opcode, left, right, width);
            if (!write_argument(cpu, instruction, &instruction->arguments[0], width, value)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_XADD:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], width, &right)) result = X86EMU_ERR_MEMORY;
        else {
            value = add_values(cpu, left, right, width, false);
            if (!write_argument(cpu, instruction, &instruction->arguments[0], width, value) ||
                !write_argument(cpu, instruction, &instruction->arguments[1], width, left)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_XCHG:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], width, &right) ||
            !write_argument(cpu, instruction, &instruction->arguments[0], width, right) ||
            !write_argument(cpu, instruction, &instruction->arguments[1], width, left)) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    case X86ASM_OP_BSWAP:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !write_argument(cpu, instruction, &instruction->arguments[0], width, byte_swap_width(left, width))) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    case X86ASM_OP_CLC:
        set_flag(cpu, X86EMU_FLAG_CF, false); cpu->rip = next_rip; break;
    case X86ASM_OP_STC:
        set_flag(cpu, X86EMU_FLAG_CF, true); cpu->rip = next_rip; break;
    case X86ASM_OP_CMC:
        set_flag(cpu, X86EMU_FLAG_CF, !get_flag(cpu, X86EMU_FLAG_CF)); cpu->rip = next_rip; break;
    case X86ASM_OP_CLD:
        set_flag(cpu, X86EMU_FLAG_DF, false); cpu->rip = next_rip; break;
    case X86ASM_OP_STD:
        set_flag(cpu, X86EMU_FLAG_DF, true); cpu->rip = next_rip; break;
    case X86ASM_OP_LAHF:
        value = (cpu->rflags & (X86EMU_FLAG_SF | X86EMU_FLAG_ZF | X86EMU_FLAG_AF | X86EMU_FLAG_PF | X86EMU_FLAG_CF)) | 0x02u;
        if (!write_register(cpu, X86ASM_REG_AH, value >> 0)) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    case X86ASM_OP_SAHF:
        if (!read_register(cpu, X86ASM_REG_AH, &value)) result = X86EMU_ERR_MEMORY;
        else {
            set_flag(cpu, X86EMU_FLAG_SF, (value & 0x80) != 0);
            set_flag(cpu, X86EMU_FLAG_ZF, (value & 0x40) != 0);
            set_flag(cpu, X86EMU_FLAG_AF, (value & 0x10) != 0);
            set_flag(cpu, X86EMU_FLAG_PF, (value & 0x04) != 0);
            set_flag(cpu, X86EMU_FLAG_CF, (value & 0x01) != 0);
            cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_BSF:
    case X86ASM_OP_BSR:
        if (!read_argument(cpu, instruction, &instruction->arguments[1], width, &right)) result = X86EMU_ERR_MEMORY;
        else if (right == 0) {
            set_flag(cpu, X86EMU_FLAG_ZF, true);
            cpu->rip = next_rip;
        } else {
            unsigned bit = 0;
            if (instruction->opcode == X86ASM_OP_BSF) {
                while (((right >> bit) & 1u) == 0) ++bit;
            } else {
                bit = width - 1u;
                while (((right >> bit) & 1u) == 0) --bit;
            }
            set_flag(cpu, X86EMU_FLAG_ZF, false);
            if (!write_argument(cpu, instruction, &instruction->arguments[0], width, bit)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_BT:
    case X86ASM_OP_BTS:
    case X86ASM_OP_BTR:
    case X86ASM_OP_BTC: {
        uint64_t bit_index;
        uint64_t bit_mask;
        uint64_t target;
        uint64_t address;
        unsigned bit;
        unsigned index_width = instruction->arguments[1].kind == X86ASM_ARG_REGISTER ? width : 8u;
        if (!read_argument(cpu, instruction, &instruction->arguments[1], index_width, &bit_index)) result = X86EMU_ERR_MEMORY;
        else if (instruction->arguments[0].kind == X86ASM_ARG_REGISTER) {
            if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &target)) result = X86EMU_ERR_MEMORY;
            else {
                bit = (unsigned)(bit_index % width);
                bit_mask = UINT64_C(1) << bit;
                set_flag(cpu, X86EMU_FLAG_CF, (target & bit_mask) != 0);
                if (instruction->opcode == X86ASM_OP_BTS) target |= bit_mask;
                else if (instruction->opcode == X86ASM_OP_BTR) target &= ~bit_mask;
                else if (instruction->opcode == X86ASM_OP_BTC) target ^= bit_mask;
                if (instruction->opcode == X86ASM_OP_BT ||
                    write_argument(cpu, instruction, &instruction->arguments[0], width, target)) cpu->rip = next_rip;
                else result = X86EMU_ERR_MEMORY;
            }
        } else if (instruction->arguments[0].kind == X86ASM_ARG_MEMORY &&
                   memory_address(cpu, instruction, &instruction->arguments[0].value.memory, &address)) {
            uint64_t byte_offset = (bit_index / width) * (width / 8u);
            bit = (unsigned)(bit_index % width);
            if (address > UINT64_MAX - byte_offset) result = X86EMU_ERR_MEMORY;
            else if (!read_memory(cpu, address + byte_offset, width, &target)) result = X86EMU_ERR_MEMORY;
            else {
                bit_mask = UINT64_C(1) << bit;
                set_flag(cpu, X86EMU_FLAG_CF, (target & bit_mask) != 0);
                if (instruction->opcode == X86ASM_OP_BTS) target |= bit_mask;
                else if (instruction->opcode == X86ASM_OP_BTR) target &= ~bit_mask;
                else if (instruction->opcode == X86ASM_OP_BTC) target ^= bit_mask;
                if (instruction->opcode == X86ASM_OP_BT || write_memory(cpu, address + byte_offset, width, target)) cpu->rip = next_rip;
                else result = X86EMU_ERR_MEMORY;
            }
        } else result = X86EMU_ERR_MEMORY;
        break;
    }
    case X86ASM_OP_CMPXCHG16B: {
        uint64_t address;
        uint64_t memory_low;
        uint64_t memory_high;
        uint64_t expected_low = cpu->registers[X86EMU_RAX];
        uint64_t expected_high = cpu->registers[X86EMU_RDX];
        if (instruction->arguments[0].kind != X86ASM_ARG_MEMORY ||
            !memory_address(cpu, instruction, &instruction->arguments[0].value.memory, &address) ||
            address > UINT64_MAX - 8u ||
            !read_memory(cpu, address, 64, &memory_low) ||
            !read_memory(cpu, address + 8u, 64, &memory_high)) result = X86EMU_ERR_MEMORY;
        else if (memory_low == expected_low && memory_high == expected_high) {
            if (!write_memory(cpu, address, 64, cpu->registers[X86EMU_RBX]) ||
                !write_memory(cpu, address + 8u, 64, cpu->registers[X86EMU_RCX])) result = X86EMU_ERR_MEMORY;
            else {
                set_flag(cpu, X86EMU_FLAG_ZF, true);
                cpu->rip = next_rip;
            }
        } else {
            cpu->registers[X86EMU_RAX] = memory_low;
            cpu->registers[X86EMU_RDX] = memory_high;
            set_flag(cpu, X86EMU_FLAG_ZF, false);
            cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_CMPXCHG8B: {
        uint64_t address;
        uint64_t current;
        uint64_t expected = ((cpu->registers[X86EMU_RDX] & UINT64_C(0xffffffff)) << 32) |
                            (cpu->registers[X86EMU_RAX] & UINT64_C(0xffffffff));
        uint64_t replacement = ((cpu->registers[X86EMU_RCX] & UINT64_C(0xffffffff)) << 32) |
                               (cpu->registers[X86EMU_RBX] & UINT64_C(0xffffffff));
        if (instruction->arguments[0].kind != X86ASM_ARG_MEMORY ||
            !memory_address(cpu, instruction, &instruction->arguments[0].value.memory, &address) ||
            !read_memory(cpu, address, 64, &current)) result = X86EMU_ERR_MEMORY;
        else if (current == expected) {
            if (!write_memory(cpu, address, 64, replacement)) result = X86EMU_ERR_MEMORY;
            else {
                set_flag(cpu, X86EMU_FLAG_ZF, true);
                cpu->rip = next_rip;
            }
        } else {
            if (!write_register(cpu, X86ASM_REG_EAX, current & UINT64_C(0xffffffff)) ||
                !write_register(cpu, X86ASM_REG_EDX, current >> 32)) result = X86EMU_ERR_MEMORY;
            else {
                set_flag(cpu, X86EMU_FLAG_ZF, false);
                cpu->rip = next_rip;
            }
        }
        break;
    }
    case X86ASM_OP_CMPXCHG:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &left) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], width, &right)) result = X86EMU_ERR_MEMORY;
        else {
            uint64_t accumulator = implicit_accumulator(cpu, width);
            (void)subtract_values(cpu, accumulator, left, width, false);
            if ((accumulator & width_mask(width)) == (left & width_mask(width))) {
                set_flag(cpu, X86EMU_FLAG_ZF, true);
                if (!write_argument(cpu, instruction, &instruction->arguments[0], width, right)) result = X86EMU_ERR_MEMORY;
            } else {
                set_flag(cpu, X86EMU_FLAG_ZF, false);
                if (!write_implicit_accumulator(cpu, width, left)) result = X86EMU_ERR_MEMORY;
            }
            if (result == X86EMU_OK) cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_MUL: {
        uint64_t divisor;
        uint64_t low_product;
        uint64_t high_product;
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &divisor)) result = X86EMU_ERR_MEMORY;
        else {
            multiply_unsigned(implicit_accumulator(cpu, width), divisor, width,
                               &low_product, &high_product);
            if (!write_implicit_accumulator(cpu, width, low_product) ||
                !write_implicit_accumulator_high(cpu, width, high_product)) result = X86EMU_ERR_MEMORY;
            else {
                set_flag(cpu, X86EMU_FLAG_CF, high_product != 0);
                set_flag(cpu, X86EMU_FLAG_OF, high_product != 0);
                cpu->rip = next_rip;
            }
        }
        break;
    }
    case X86ASM_OP_DIV:
    case X86ASM_OP_IDIV: {
        uint64_t divisor;
        uint64_t dividend_low = implicit_accumulator(cpu, width);
        uint64_t dividend_high = implicit_accumulator_high(cpu, width);
        uint64_t quotient;
        uint64_t remainder;
        bool valid;
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &divisor)) result = X86EMU_ERR_MEMORY;
        else if (instruction->opcode == X86ASM_OP_DIV) {
            valid = divide_unsigned_128(dividend_high, dividend_low, divisor, &quotient, &remainder);
            if (!valid) result = X86EMU_ERR_ARITHMETIC;
            else if (!write_implicit_accumulator(cpu, width, quotient) ||
                     !write_implicit_accumulator_high(cpu, width, remainder)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        } else {
            valid = signed_divide_128(dividend_high, dividend_low, divisor, width, &quotient, &remainder);
            if (!valid) result = X86EMU_ERR_ARITHMETIC;
            else if (!write_implicit_accumulator(cpu, width, quotient) ||
                     !write_implicit_accumulator_high(cpu, width, remainder)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    }
    case X86ASM_OP_IMUL:
        if (instruction->arguments[1].kind == X86ASM_ARG_NONE) {
            uint64_t operand;
            uint64_t low_product;
            uint64_t high_product;
            bool left_negative;
            bool right_negative;
            uint64_t left_magnitude;
            uint64_t right_magnitude;
            bool product_negative;
            if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &operand)) result = X86EMU_ERR_MEMORY;
            else {
                (void)signed_magnitude(implicit_accumulator(cpu, width), width, &left_negative, &left_magnitude);
                (void)signed_magnitude(operand, width, &right_negative, &right_magnitude);
                multiply_unsigned(left_magnitude, right_magnitude, width, &low_product, &high_product);
                product_negative = left_negative != right_negative;
                if (product_negative) {
                    uint64_t old_low = low_product;
                    low_product = ~low_product + 1u;
                    high_product = (~high_product + (low_product == 0 && old_low != 0 ? 1u : 0u)) & width_mask(width);
                }
                if (!write_implicit_accumulator(cpu, width, low_product) ||
                    !write_implicit_accumulator_high(cpu, width, high_product)) result = X86EMU_ERR_MEMORY;
                else {
                    bool sign_extension = (low_product & (UINT64_C(1) << (width - 1))) != 0 ? high_product == width_mask(width) : high_product == 0;
                    set_flag(cpu, X86EMU_FLAG_CF, !sign_extension);
                    set_flag(cpu, X86EMU_FLAG_OF, !sign_extension);
                    cpu->rip = next_rip;
                }
            }
        } else if (!read_argument(cpu, instruction, &instruction->arguments[1], width, &right) ||
                   !read_argument(cpu, instruction, &instruction->arguments[0], width, &left)) result = X86EMU_ERR_MEMORY;
        else {
            uint64_t product_low;
            uint64_t product_high;
            uint64_t left_magnitude;
            uint64_t right_magnitude;
            bool left_negative;
            bool right_negative;
            bool product_negative;
            bool overflow;
            (void)signed_magnitude(left, width, &left_negative, &left_magnitude);
            (void)signed_magnitude(right, width, &right_negative, &right_magnitude);
            multiply_unsigned(left_magnitude, right_magnitude, width, &product_low, &product_high);
            product_negative = left_negative != right_negative;
            if (product_negative) {
                uint64_t old_low = product_low;
                product_low = ~product_low + 1u;
                product_high = ~product_high + (product_low == 0 && old_low != 0 ? 1u : 0u);
            }
            overflow = (product_low & (UINT64_C(1) << (width - 1))) != 0
                       ? product_high != width_mask(width)
                       : product_high != 0;
            set_flag(cpu, X86EMU_FLAG_CF, overflow);
            set_flag(cpu, X86EMU_FLAG_OF, overflow);
            if (!write_argument(cpu, instruction, &instruction->arguments[0], width, product_low)) result = X86EMU_ERR_MEMORY;
            else cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_ENTER: {
        uint64_t frame_size;
        uint64_t nesting;
        uint64_t frame_pointer;
        if (!read_argument(cpu, instruction, &instruction->arguments[0], 16, &frame_size) ||
            !read_argument(cpu, instruction, &instruction->arguments[1], 8, &nesting)) result = X86EMU_ERR_MEMORY;
        else if ((nesting & 31u) != 0) {
            uint64_t old_base = cpu->registers[X86EMU_RBP];
            frame_pointer = cpu->registers[X86EMU_RSP] - X86EMU_STACK_WIDTH;
            result = stack_push(cpu, old_base);
            if (result == X86EMU_OK) {
                for (uint64_t level = 1; level < (nesting & 31u); ++level) {
                    cpu->registers[X86EMU_RBP] -= X86EMU_STACK_WIDTH;
                    result = stack_push(cpu, cpu->registers[X86EMU_RBP]);
                    if (result != X86EMU_OK) break;
                }
            }
            if (result == X86EMU_OK) result = stack_push(cpu, frame_pointer);
            if (result == X86EMU_OK) {
                cpu->registers[X86EMU_RBP] = frame_pointer;
                cpu->registers[X86EMU_RSP] -= frame_size;
                cpu->rip = next_rip;
            }
        } else {
            frame_pointer = cpu->registers[X86EMU_RSP] - X86EMU_STACK_WIDTH;
            result = stack_push(cpu, cpu->registers[X86EMU_RBP]);
            if (result == X86EMU_OK) {
                cpu->registers[X86EMU_RBP] = frame_pointer;
                cpu->registers[X86EMU_RSP] -= frame_size;
                cpu->rip = next_rip;
            }
        }
        break;
    }
    case X86ASM_OP_LEAVE:
        cpu->registers[X86EMU_RSP] = cpu->registers[X86EMU_RBP];
        result = stack_pop(cpu, &value);
        if (result == X86EMU_OK) {
            cpu->registers[X86EMU_RBP] = value;
            cpu->rip = next_rip;
        }
        break;
    case X86ASM_OP_PUSH:
        if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &value)) result = X86EMU_ERR_MEMORY;
        else { result = stack_push(cpu, value); if (result == X86EMU_OK) cpu->rip = next_rip; }
        break;
    case X86ASM_OP_POP:
        result = stack_pop(cpu, &value);
        if (result == X86EMU_OK && !write_argument(cpu, instruction, &instruction->arguments[0], width, value)) result = X86EMU_ERR_MEMORY;
        if (result == X86EMU_OK) cpu->rip = next_rip;
        break;
    case X86ASM_OP_CALL:
        if (instruction->arguments[0].kind == X86ASM_ARG_RELATIVE) value = next_rip + (uint64_t)(int64_t)instruction->arguments[0].value.relative;
        else if (!read_argument(cpu, instruction, &instruction->arguments[0], width, &value)) result = X86EMU_ERR_MEMORY;
        if (result == X86EMU_OK) { result = stack_push(cpu, next_rip); if (result == X86EMU_OK) cpu->rip = value; }
        break;
    case X86ASM_OP_RET:
        result = stack_pop(cpu, &value);
        if (result == X86EMU_OK) {
            if (instruction->arguments[0].kind == X86ASM_ARG_IMMEDIATE) {
                cpu->registers[X86EMU_RSP] += (uint64_t)instruction->arguments[0].value.immediate;
            }
            cpu->rip = value;
        }
        break;
    case X86ASM_OP_JMP:
        if (instruction->arguments[0].kind == X86ASM_ARG_RELATIVE) cpu->rip = next_rip + (uint64_t)(int64_t)instruction->arguments[0].value.relative;
        else if (read_argument(cpu, instruction, &instruction->arguments[0], width, &value)) cpu->rip = value;
        else result = X86EMU_ERR_MEMORY;
        break;
    case X86ASM_OP_JA: case X86ASM_OP_JAE: case X86ASM_OP_JB: case X86ASM_OP_JBE:
    case X86ASM_OP_JE: case X86ASM_OP_JG: case X86ASM_OP_JGE: case X86ASM_OP_JL:
    case X86ASM_OP_JLE: case X86ASM_OP_JNE: case X86ASM_OP_JNO: case X86ASM_OP_JNP:
    case X86ASM_OP_JNS: case X86ASM_OP_JO: case X86ASM_OP_JP: case X86ASM_OP_JS:
        if (condition_holds(cpu, instruction->opcode) && instruction->arguments[0].kind == X86ASM_ARG_RELATIVE)
            cpu->rip = next_rip + (uint64_t)(int64_t)instruction->arguments[0].value.relative;
        else cpu->rip = next_rip;
        break;
    case X86ASM_OP_LOOP:
    case X86ASM_OP_LOOPE:
    case X86ASM_OP_LOOPNE:
    case X86ASM_OP_JCXZ:
        if (instruction->opcode != X86ASM_OP_JCXZ) --cpu->registers[X86EMU_RCX];
        condition = instruction->opcode == X86ASM_OP_JCXZ ? cpu->registers[X86EMU_RCX] == 0 : cpu->registers[X86EMU_RCX] != 0;
        if (instruction->opcode == X86ASM_OP_LOOPE) condition = condition && get_flag(cpu, X86EMU_FLAG_ZF);
        if (instruction->opcode == X86ASM_OP_LOOPNE) condition = condition && !get_flag(cpu, X86EMU_FLAG_ZF);
        if (condition) cpu->rip = next_rip + (uint64_t)(int64_t)instruction->arguments[0].value.relative;
        else cpu->rip = next_rip;
        break;
    case X86ASM_OP_CMOVA: case X86ASM_OP_CMOVAE: case X86ASM_OP_CMOVB: case X86ASM_OP_CMOVBE:
    case X86ASM_OP_CMOVE: case X86ASM_OP_CMOVG: case X86ASM_OP_CMOVGE: case X86ASM_OP_CMOVL:
    case X86ASM_OP_CMOVLE: case X86ASM_OP_CMOVNE: case X86ASM_OP_CMOVNO: case X86ASM_OP_CMOVNP:
    case X86ASM_OP_CMOVNS: case X86ASM_OP_CMOVO: case X86ASM_OP_CMOVP: case X86ASM_OP_CMOVS:
        condition = condition_holds(cpu, instruction->opcode);
        if (condition && (!read_argument(cpu, instruction, &instruction->arguments[1], width, &value) ||
                          !write_argument(cpu, instruction, &instruction->arguments[0], width, value))) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    case X86ASM_OP_SETA: case X86ASM_OP_SETAE: case X86ASM_OP_SETB: case X86ASM_OP_SETBE:
    case X86ASM_OP_SETE: case X86ASM_OP_SETG: case X86ASM_OP_SETGE: case X86ASM_OP_SETL:
    case X86ASM_OP_SETLE: case X86ASM_OP_SETNE: case X86ASM_OP_SETNO: case X86ASM_OP_SETNP:
    case X86ASM_OP_SETNS: case X86ASM_OP_SETO: case X86ASM_OP_SETP: case X86ASM_OP_SETS:
        if (!write_argument(cpu, instruction, &instruction->arguments[0], 8,
                            condition_holds(cpu, instruction->opcode) ? 1 : 0)) result = X86EMU_ERR_MEMORY;
        else cpu->rip = next_rip;
        break;
    case X86ASM_OP_INT:
        if (instruction->arguments[0].kind != X86ASM_ARG_IMMEDIATE) result = X86EMU_ERR_INTERRUPT;
        else if (instruction->arguments[0].value.immediate == 3 && cpu->break_on_int3) result = X86EMU_ERR_BREAKPOINT;
        else if (cpu->interrupt_handler != NULL && cpu->interrupt_handler(cpu, (uint8_t)instruction->arguments[0].value.immediate, cpu->user_data)) cpu->rip = next_rip;
        else result = X86EMU_ERR_INTERRUPT;
        break;
    case X86ASM_OP_SYSCALL:
    case X86ASM_OP_SYSENTER:
    case X86ASM_OP_SYSRET:
    case X86ASM_OP_SYSEXIT:
        if (cpu->system_handler != NULL) {
            cpu->rip = next_rip;
            if (!cpu->system_handler(cpu, instruction->opcode, cpu->user_data)) result = X86EMU_ERR_PRIVILEGED;
        } else result = X86EMU_ERR_PRIVILEGED;
        break;
    case X86ASM_OP_UD2:
        result = X86EMU_ERR_DECODE;
        break;
    default:
        result = X86EMU_ERR_UNSUPPORTED;
        break;
    }

    if (result != X86EMU_OK) {
        cpu->last_error = result;
        return result;
    }
    cpu->steps++;
    return X86EMU_OK;
}

x86emu_error x86emu_run(x86emu_cpu *cpu, uint64_t max_steps)
{
    if (cpu == NULL) return X86EMU_ERR_BAD_ARGUMENT;
    for (uint64_t i = 0; i < max_steps; ++i) {
        x86emu_error error = x86emu_step(cpu);
        if (error != X86EMU_OK) return error;
    }
    cpu->last_error = X86EMU_ERR_STEP_LIMIT;
    return cpu->last_error;
}
