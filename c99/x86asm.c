#include "x86asm.h"

#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>

#define X86ASM_MAX_INSTRUCTION_LENGTH 15

static const char *const register_names[X86ASM_REG_COUNT] = {
    [X86ASM_REG_NONE] = "",
    [X86ASM_REG_AL] = "al", [X86ASM_REG_CL] = "cl",
    [X86ASM_REG_DL] = "dl", [X86ASM_REG_BL] = "bl",
    [X86ASM_REG_AH] = "ah", [X86ASM_REG_CH] = "ch",
    [X86ASM_REG_DH] = "dh", [X86ASM_REG_BH] = "bh",
    [X86ASM_REG_SPL] = "spl", [X86ASM_REG_BPL] = "bpl",
    [X86ASM_REG_SIL] = "sil", [X86ASM_REG_DIL] = "dil",
    [X86ASM_REG_R8B] = "r8b", [X86ASM_REG_R9B] = "r9b",
    [X86ASM_REG_R10B] = "r10b", [X86ASM_REG_R11B] = "r11b",
    [X86ASM_REG_R12B] = "r12b", [X86ASM_REG_R13B] = "r13b",
    [X86ASM_REG_R14B] = "r14b", [X86ASM_REG_R15B] = "r15b",
    [X86ASM_REG_AX] = "ax", [X86ASM_REG_CX] = "cx",
    [X86ASM_REG_DX] = "dx", [X86ASM_REG_BX] = "bx",
    [X86ASM_REG_SP] = "sp", [X86ASM_REG_BP] = "bp",
    [X86ASM_REG_SI] = "si", [X86ASM_REG_DI] = "di",
    [X86ASM_REG_R8W] = "r8w", [X86ASM_REG_R9W] = "r9w",
    [X86ASM_REG_R10W] = "r10w", [X86ASM_REG_R11W] = "r11w",
    [X86ASM_REG_R12W] = "r12w", [X86ASM_REG_R13W] = "r13w",
    [X86ASM_REG_R14W] = "r14w", [X86ASM_REG_R15W] = "r15w",
    [X86ASM_REG_EAX] = "eax", [X86ASM_REG_ECX] = "ecx",
    [X86ASM_REG_EDX] = "edx", [X86ASM_REG_EBX] = "ebx",
    [X86ASM_REG_ESP] = "esp", [X86ASM_REG_EBP] = "ebp",
    [X86ASM_REG_ESI] = "esi", [X86ASM_REG_EDI] = "edi",
    [X86ASM_REG_R8D] = "r8d", [X86ASM_REG_R9D] = "r9d",
    [X86ASM_REG_R10D] = "r10d", [X86ASM_REG_R11D] = "r11d",
    [X86ASM_REG_R12D] = "r12d", [X86ASM_REG_R13D] = "r13d",
    [X86ASM_REG_R14D] = "r14d", [X86ASM_REG_R15D] = "r15d",
    [X86ASM_REG_RAX] = "rax", [X86ASM_REG_RCX] = "rcx",
    [X86ASM_REG_RDX] = "rdx", [X86ASM_REG_RBX] = "rbx",
    [X86ASM_REG_RSP] = "rsp", [X86ASM_REG_RBP] = "rbp",
    [X86ASM_REG_RSI] = "rsi", [X86ASM_REG_RDI] = "rdi",
    [X86ASM_REG_R8] = "r8", [X86ASM_REG_R9] = "r9",
    [X86ASM_REG_R10] = "r10", [X86ASM_REG_R11] = "r11",
    [X86ASM_REG_R12] = "r12", [X86ASM_REG_R13] = "r13",
    [X86ASM_REG_R14] = "r14", [X86ASM_REG_R15] = "r15",
    [X86ASM_REG_IP] = "ip", [X86ASM_REG_EIP] = "eip",
    [X86ASM_REG_RIP] = "rip", [X86ASM_REG_ES] = "es",
    [X86ASM_REG_CS] = "cs", [X86ASM_REG_SS] = "ss",
    [X86ASM_REG_DS] = "ds", [X86ASM_REG_FS] = "fs",
    [X86ASM_REG_GS] = "gs",
    [X86ASM_REG_XMM0] = "xmm0", [X86ASM_REG_XMM1] = "xmm1",
    [X86ASM_REG_XMM2] = "xmm2", [X86ASM_REG_XMM3] = "xmm3",
    [X86ASM_REG_XMM4] = "xmm4", [X86ASM_REG_XMM5] = "xmm5",
    [X86ASM_REG_XMM6] = "xmm6", [X86ASM_REG_XMM7] = "xmm7",
    [X86ASM_REG_XMM8] = "xmm8", [X86ASM_REG_XMM9] = "xmm9",
    [X86ASM_REG_XMM10] = "xmm10", [X86ASM_REG_XMM11] = "xmm11",
    [X86ASM_REG_XMM12] = "xmm12", [X86ASM_REG_XMM13] = "xmm13",
    [X86ASM_REG_XMM14] = "xmm14", [X86ASM_REG_XMM15] = "xmm15",
    [X86ASM_REG_YMM0] = "ymm0", [X86ASM_REG_YMM1] = "ymm1",
    [X86ASM_REG_YMM2] = "ymm2", [X86ASM_REG_YMM3] = "ymm3",
    [X86ASM_REG_YMM4] = "ymm4", [X86ASM_REG_YMM5] = "ymm5",
    [X86ASM_REG_YMM6] = "ymm6", [X86ASM_REG_YMM7] = "ymm7",
    [X86ASM_REG_YMM8] = "ymm8", [X86ASM_REG_YMM9] = "ymm9",
    [X86ASM_REG_YMM10] = "ymm10", [X86ASM_REG_YMM11] = "ymm11",
    [X86ASM_REG_YMM12] = "ymm12", [X86ASM_REG_YMM13] = "ymm13",
    [X86ASM_REG_YMM14] = "ymm14", [X86ASM_REG_YMM15] = "ymm15",
    [X86ASM_REG_ZMM0] = "zmm0", [X86ASM_REG_ZMM1] = "zmm1",
    [X86ASM_REG_ZMM2] = "zmm2", [X86ASM_REG_ZMM3] = "zmm3",
    [X86ASM_REG_ZMM4] = "zmm4", [X86ASM_REG_ZMM5] = "zmm5",
    [X86ASM_REG_ZMM6] = "zmm6", [X86ASM_REG_ZMM7] = "zmm7",
    [X86ASM_REG_ZMM8] = "zmm8", [X86ASM_REG_ZMM9] = "zmm9",
    [X86ASM_REG_ZMM10] = "zmm10", [X86ASM_REG_ZMM11] = "zmm11",
    [X86ASM_REG_ZMM12] = "zmm12", [X86ASM_REG_ZMM13] = "zmm13",
    [X86ASM_REG_ZMM14] = "zmm14", [X86ASM_REG_ZMM15] = "zmm15",
    [X86ASM_REG_ZMM16] = "zmm16", [X86ASM_REG_ZMM17] = "zmm17",
    [X86ASM_REG_ZMM18] = "zmm18", [X86ASM_REG_ZMM19] = "zmm19",
    [X86ASM_REG_ZMM20] = "zmm20", [X86ASM_REG_ZMM21] = "zmm21",
    [X86ASM_REG_ZMM22] = "zmm22", [X86ASM_REG_ZMM23] = "zmm23",
    [X86ASM_REG_ZMM24] = "zmm24", [X86ASM_REG_ZMM25] = "zmm25",
    [X86ASM_REG_ZMM26] = "zmm26", [X86ASM_REG_ZMM27] = "zmm27",
    [X86ASM_REG_ZMM28] = "zmm28", [X86ASM_REG_ZMM29] = "zmm29",
    [X86ASM_REG_ZMM30] = "zmm30", [X86ASM_REG_ZMM31] = "zmm31",
    [X86ASM_REG_K0] = "k0", [X86ASM_REG_K1] = "k1",
    [X86ASM_REG_K2] = "k2", [X86ASM_REG_K3] = "k3",
    [X86ASM_REG_K4] = "k4", [X86ASM_REG_K5] = "k5",
    [X86ASM_REG_K6] = "k6", [X86ASM_REG_K7] = "k7"
};

static const char *const opcode_names[X86ASM_OP_COUNT] = {
    [X86ASM_OP_AAA] = "aaa", [X86ASM_OP_ADC] = "adc",
    [X86ASM_OP_ADD] = "add", [X86ASM_OP_ADDPS] = "addps", [X86ASM_OP_AND] = "and",
    [X86ASM_OP_BEXTR] = "bextr", [X86ASM_OP_BSWAP] = "bswap",
    [X86ASM_OP_BSF] = "bsf", [X86ASM_OP_BSR] = "bsr",
    [X86ASM_OP_BT] = "bt", [X86ASM_OP_BTS] = "bts",
    [X86ASM_OP_BTR] = "btr", [X86ASM_OP_BTC] = "btc",
    [X86ASM_OP_CLC] = "clc", [X86ASM_OP_CMC] = "cmc", [X86ASM_OP_CLD] = "cld",
    [X86ASM_OP_CALL] = "call", [X86ASM_OP_CMP] = "cmp",
    [X86ASM_OP_CMPXCHG] = "cmpxchg", [X86ASM_OP_CMPXCHG8B] = "cmpxchg8b",
    [X86ASM_OP_CMPXCHG16B] = "cmpxchg16b",
    [X86ASM_OP_DEC] = "dec", [X86ASM_OP_DIV] = "div",
    [X86ASM_OP_IDIV] = "idiv", [X86ASM_OP_IMUL] = "imul",
    [X86ASM_OP_INC] = "inc", [X86ASM_OP_INT] = "int",
    [X86ASM_OP_JA] = "ja", [X86ASM_OP_JAE] = "jae",
    [X86ASM_OP_JB] = "jb", [X86ASM_OP_JBE] = "jbe",
    [X86ASM_OP_JCXZ] = "jcxz", [X86ASM_OP_JE] = "je", [X86ASM_OP_JG] = "jg",
    [X86ASM_OP_JGE] = "jge", [X86ASM_OP_JL] = "jl",
    [X86ASM_OP_JLE] = "jle", [X86ASM_OP_JMP] = "jmp",
    [X86ASM_OP_JNE] = "jne", [X86ASM_OP_JNO] = "jno",
    [X86ASM_OP_JNP] = "jnp", [X86ASM_OP_JNS] = "jns",
    [X86ASM_OP_LOOP] = "loop", [X86ASM_OP_LOOPE] = "loope", [X86ASM_OP_LOOPNE] = "loopne",
    [X86ASM_OP_JO] = "jo", [X86ASM_OP_JP] = "jp",
    [X86ASM_OP_JS] = "js",
    [X86ASM_OP_CMOVA] = "cmova", [X86ASM_OP_CMOVAE] = "cmovae",
    [X86ASM_OP_CMOVB] = "cmovb", [X86ASM_OP_CMOVBE] = "cmovbe",
    [X86ASM_OP_CMOVE] = "cmove", [X86ASM_OP_CMOVG] = "cmovg",
    [X86ASM_OP_CMOVGE] = "cmovge", [X86ASM_OP_CMOVL] = "cmovl",
    [X86ASM_OP_CMOVLE] = "cmovle", [X86ASM_OP_CMOVNE] = "cmovne",
    [X86ASM_OP_CMOVNO] = "cmovno", [X86ASM_OP_CMOVNP] = "cmovnp",
    [X86ASM_OP_CMOVNS] = "cmovns", [X86ASM_OP_CMOVO] = "cmovo",
    [X86ASM_OP_CMOVP] = "cmovp", [X86ASM_OP_CMOVS] = "cmovs",
    [X86ASM_OP_LAHF] = "lahf", [X86ASM_OP_LEA] = "lea", [X86ASM_OP_LEAVE] = "leave",
    [X86ASM_OP_MOV] = "mov", [X86ASM_OP_MOVSX] = "movsx", [X86ASM_OP_MOVBE] = "movbe",
    [X86ASM_OP_MOVD] = "movd", [X86ASM_OP_MOVQ] = "movq", [X86ASM_OP_MOVLPS] = "movlps", [X86ASM_OP_MOVHPS] = "movhps", [X86ASM_OP_MOVLPD] = "movlpd", [X86ASM_OP_MOVHPD] = "movhpd",
    [X86ASM_OP_MOVDQA] = "movdqa", [X86ASM_OP_MOVDQU] = "movdqu", [X86ASM_OP_MOVNTDQA] = "movntdqa", [X86ASM_OP_MOVNTDQ] = "movntdq", [X86ASM_OP_LDDQU] = "lddqu", [X86ASM_OP_MOVUPD] = "movupd", [X86ASM_OP_MOVSS] = "movss", [X86ASM_OP_MOVSD_SCALAR] = "movsd",
    [X86ASM_OP_SUBPS] = "subps", [X86ASM_OP_MULPS] = "mulps", [X86ASM_OP_DIVPS] = "divps",
    [X86ASM_OP_ADDPD] = "addpd", [X86ASM_OP_SUBPD] = "subpd", [X86ASM_OP_MULPD] = "mulpd", [X86ASM_OP_DIVPD] = "divpd",
    [X86ASM_OP_PADDB] = "paddb", [X86ASM_OP_PADDW] = "paddw", [X86ASM_OP_PADDD] = "paddd",
    [X86ASM_OP_PSUBB] = "psubb", [X86ASM_OP_PSUBW] = "psubw", [X86ASM_OP_PSUBD] = "psubd",
    [X86ASM_OP_PCMPEQB] = "pcmpeqb", [X86ASM_OP_PCMPEQW] = "pcmpeqw", [X86ASM_OP_PCMPEQD] = "pcmpeqd",
    [X86ASM_OP_PAND] = "pand", [X86ASM_OP_POR] = "por", [X86ASM_OP_PXOR] = "pxor",
    [X86ASM_OP_PSLLW] = "psllw", [X86ASM_OP_PSLLD] = "pslld", [X86ASM_OP_PSLLQ] = "psllq",
    [X86ASM_OP_PSRLW] = "psrlw", [X86ASM_OP_PSRLD] = "psrld", [X86ASM_OP_PSRLQ] = "psrlq",
    [X86ASM_OP_PSRAW] = "psraw", [X86ASM_OP_PSRAD] = "psrad",
    [X86ASM_OP_PSLLDQ] = "pslldq", [X86ASM_OP_PSRLDQ] = "psrldq",
    [X86ASM_OP_PMULLW] = "pmullw", [X86ASM_OP_PMULHW] = "pmulhw",
    [X86ASM_OP_PMULHUW] = "pmulhuw", [X86ASM_OP_PMULUDQ] = "pmuludq", [X86ASM_OP_PMULLD] = "pmulld",
    [X86ASM_OP_PADDUSB] = "paddusb", [X86ASM_OP_PADDUSW] = "paddusw",
    [X86ASM_OP_PADDSB] = "paddsb", [X86ASM_OP_PADDSW] = "paddsw",
    [X86ASM_OP_PSUBUSB] = "psubusb", [X86ASM_OP_PSUBUSW] = "psubusw",
    [X86ASM_OP_PSUBSB] = "psubsb", [X86ASM_OP_PSUBSW] = "psubsw",
    [X86ASM_OP_PMINUB] = "pminub", [X86ASM_OP_PMAXUB] = "pmaxub", [X86ASM_OP_PMINSB] = "pminsb", [X86ASM_OP_PMAXSB] = "pmaxsb", [X86ASM_OP_PMINUW] = "pminuw", [X86ASM_OP_PMAXUW] = "pmaxuw", [X86ASM_OP_PMINSD] = "pminsd", [X86ASM_OP_PMAXSD] = "pmaxsd", [X86ASM_OP_PMINUD] = "pminud", [X86ASM_OP_PMAXUD] = "pmaxud",
    [X86ASM_OP_PSHUFB] = "pshufb", [X86ASM_OP_PSHUFD] = "pshufd", [X86ASM_OP_PSHUFLW] = "pshuflw", [X86ASM_OP_PSHUFHW] = "pshufhw",
    [X86ASM_OP_PABSB] = "pabsb", [X86ASM_OP_PABSW] = "pabsw", [X86ASM_OP_PABSD] = "pabsd", [X86ASM_OP_PSIGNB] = "psignb", [X86ASM_OP_PSIGNW] = "psignw", [X86ASM_OP_PSIGND] = "psignd",
    [X86ASM_OP_PHADDW] = "phaddw", [X86ASM_OP_PHADDD] = "phaddd", [X86ASM_OP_PHADDSW] = "phaddsw", [X86ASM_OP_PHSUBW] = "phsubw", [X86ASM_OP_PHSUBD] = "phsubd", [X86ASM_OP_PHSUBSW] = "phsubsw", [X86ASM_OP_PMADDUBSW] = "pmaddubsw", [X86ASM_OP_PMADDWD] = "pmaddwd",     [X86ASM_OP_PMULDQ] = "pmuldq",
    [X86ASM_OP_PMOVSXBW] = "pmovsxbw", [X86ASM_OP_PMOVSXBD] = "pmovsxbd", [X86ASM_OP_PMOVSXBQ] = "pmovsxbq", [X86ASM_OP_PMOVSXWD] = "pmovsxwd", [X86ASM_OP_PMOVSXWQ] = "pmovsxwq", [X86ASM_OP_PMOVSXDQ] = "pmovsxdq",
    [X86ASM_OP_PMOVZXBW] = "pmovzxbw", [X86ASM_OP_PMOVZXBD] = "pmovzxbd", [X86ASM_OP_PMOVZXBQ] = "pmovzxbq", [X86ASM_OP_PMOVZXWD] = "pmovzxwd", [X86ASM_OP_PMOVZXWQ] = "pmovzxwq", [X86ASM_OP_PMOVZXDQ] = "pmovzxdq",     [X86ASM_OP_PBLENDVB] = "pblendvb", [X86ASM_OP_BLENDPS] = "blendps", [X86ASM_OP_BLENDPD] = "blendpd", [X86ASM_OP_BLENDVPS] = "blendvps", [X86ASM_OP_BLENDVPD] = "blendvpd", [X86ASM_OP_PBLENDW] = "pblendw", [X86ASM_OP_PALIGNR] = "palignr", [X86ASM_OP_PINSRB] = "pinsrb", [X86ASM_OP_PINSRW] = "pinsrw", [X86ASM_OP_PINSRD] = "pinsrd", [X86ASM_OP_PINSRQ] = "pinsrq", [X86ASM_OP_PEXTRB] = "pextrb", [X86ASM_OP_PEXTRW] = "pextrw", [X86ASM_OP_PEXTRD] = "pextrd", [X86ASM_OP_PEXTRQ] = "pextrq", [X86ASM_OP_PHMINPOSUW] = "phminposuw",


    [X86ASM_OP_VPSHUFB] = "vpshufb", [X86ASM_OP_VPSHUFD] = "vpshufd", [X86ASM_OP_VPSHUFLW] = "vpshuflw", [X86ASM_OP_VPSHUFHW] = "vpshufhw",
    [X86ASM_OP_VPABSB] = "vpabsb", [X86ASM_OP_VPABSW] = "vpabsw", [X86ASM_OP_VPABSD] = "vpabsd", [X86ASM_OP_VPSIGNB] = "vpsignb", [X86ASM_OP_VPSIGNW] = "vpsignw", [X86ASM_OP_VPSIGND] = "vpsignd",
    [X86ASM_OP_VPHADDW] = "vphaddw", [X86ASM_OP_VPHADDD] = "vphaddd", [X86ASM_OP_VPHADDSW] = "vphaddsw", [X86ASM_OP_VPHSUBW] = "vphsubw", [X86ASM_OP_VPHSUBD] = "vphsubd", [X86ASM_OP_VPHSUBSW] = "vphsubsw", [X86ASM_OP_VPMADDUBSW] = "vpmaddubsw", [X86ASM_OP_VPMADDWD] = "vpmaddwd",     [X86ASM_OP_VPMULDQ] = "vpmuldq",
    [X86ASM_OP_VPMOVSXBW] = "vpmovsxbw", [X86ASM_OP_VPMOVSXBD] = "vpmovsxbd", [X86ASM_OP_VPMOVSXBQ] = "vpmovsxbq", [X86ASM_OP_VPMOVSXWD] = "vpmovsxwd", [X86ASM_OP_VPMOVSXWQ] = "vpmovsxwq", [X86ASM_OP_VPMOVSXDQ] = "vpmovsxdq",
    [X86ASM_OP_VPMOVZXBW] = "vpmovzxbw", [X86ASM_OP_VPMOVZXBD] = "vpmovzxbd", [X86ASM_OP_VPMOVZXBQ] = "vpmovzxbq", [X86ASM_OP_VPMOVZXWD] = "vpmovzxwd", [X86ASM_OP_VPMOVZXWQ] = "vpmovzxwq", [X86ASM_OP_VPMOVZXDQ] = "vpmovzxdq",     [X86ASM_OP_VPBLENDVB] = "vpblendvb", [X86ASM_OP_VPBLENDD] = "vpblendd", [X86ASM_OP_VBLENDVPS] = "vblendvps", [X86ASM_OP_VBLENDVPD] = "vblendvpd", [X86ASM_OP_VPBLENDW] = "vpblendw", [X86ASM_OP_VPALIGNR] = "vpalignr", [X86ASM_OP_VPINSRB] = "vpinsrb", [X86ASM_OP_VPINSRW] = "vpinsrw", [X86ASM_OP_VPINSRD] = "vpinsrd", [X86ASM_OP_VPINSRQ] = "vpinsrq", [X86ASM_OP_VPEXTRB] = "vpextrb", [X86ASM_OP_VPEXTRW] = "vpextrw", [X86ASM_OP_VPEXTRD] = "vpextrd", [X86ASM_OP_VPEXTRQ] = "vpextrq", [X86ASM_OP_VPHMINPOSUW] = "vphminposuw",


    [X86ASM_OP_PMINSW] = "pminsw", [X86ASM_OP_PMAXSW] = "pmaxsw",
    [X86ASM_OP_PAVGB] = "pavgb", [X86ASM_OP_PAVGW] = "pavgw", [X86ASM_OP_PSADBW] = "psadbw",
    [X86ASM_OP_PUNPCKLBW] = "punpcklbw", [X86ASM_OP_PUNPCKLWD] = "punpcklwd", [X86ASM_OP_PUNPCKLDQ] = "punpckldq",
    [X86ASM_OP_PUNPCKHBW] = "punpckhbw", [X86ASM_OP_PUNPCKHWD] = "punpckhwd", [X86ASM_OP_PUNPCKHDQ] = "punpckhdq",
    [X86ASM_OP_PACKSSWB] = "packsswb", [X86ASM_OP_PACKSSDW] = "packssdw", [X86ASM_OP_PACKUSWB] = "packuswb",
    [X86ASM_OP_MOVSB] = "movsb", [X86ASM_OP_MOVSW] = "movsw",
    [X86ASM_OP_MOVSD] = "movsd", [X86ASM_OP_MOVSQ] = "movsq",
    [X86ASM_OP_STOSB] = "stosb", [X86ASM_OP_STOSW] = "stosw",
    [X86ASM_OP_STOSD] = "stosd", [X86ASM_OP_STOSQ] = "stosq",
    [X86ASM_OP_LODSB] = "lodsb", [X86ASM_OP_LODSW] = "lodsw",
    [X86ASM_OP_LODSD] = "lodsd", [X86ASM_OP_LODSQ] = "lodsq",
    [X86ASM_OP_CMPSB] = "cmpsb", [X86ASM_OP_CMPSW] = "cmpsw",
    [X86ASM_OP_CMPSD] = "cmpsd", [X86ASM_OP_CMPSQ] = "cmpsq",
    [X86ASM_OP_SCASB] = "scasb", [X86ASM_OP_SCASW] = "scasw",
    [X86ASM_OP_SCASD] = "scasd", [X86ASM_OP_SCASQ] = "scasq",
    [X86ASM_OP_MOVUPS] = "movups", [X86ASM_OP_MOVMSKPS] = "movmskps", [X86ASM_OP_MOVMSKPD] = "movmskpd", [X86ASM_OP_PMOVMSKB] = "pmovmskb", [X86ASM_OP_PTEST] = "ptest",
    [X86ASM_OP_MINPS] = "minps", [X86ASM_OP_MAXPS] = "maxps", [X86ASM_OP_MINPD] = "minpd", [X86ASM_OP_MAXPD] = "maxpd", [X86ASM_OP_CMPPS] = "cmpps", [X86ASM_OP_CMPPD] = "cmppd", [X86ASM_OP_ADDSS] = "addss", [X86ASM_OP_SUBSS] = "subss", [X86ASM_OP_MULSS] = "mulss", [X86ASM_OP_DIVSS] = "divss", [X86ASM_OP_MINSS] = "minss", [X86ASM_OP_MAXSS] = "maxss", [X86ASM_OP_ADDSD] = "addsd", [X86ASM_OP_SUBSD] = "subsd", [X86ASM_OP_MULSD] = "mulsd", [X86ASM_OP_DIVSD] = "divsd", [X86ASM_OP_MINSD] = "minsd", [X86ASM_OP_MAXSD] = "maxsd",
    [X86ASM_OP_MUL] = "mul", [X86ASM_OP_MOVZX] = "movzx", [X86ASM_OP_NEG] = "neg",
    [X86ASM_OP_NOP] = "nop", [X86ASM_OP_NOT] = "not",
    [X86ASM_OP_ENTER] = "enter", [X86ASM_OP_SAHF] = "sahf",
    [X86ASM_OP_OR] = "or", [X86ASM_OP_POP] = "pop",
    [X86ASM_OP_PUSH] = "push", [X86ASM_OP_PUSHF] = "pushf",
    [X86ASM_OP_POPF] = "popf", [X86ASM_OP_CLI] = "cli",
    [X86ASM_OP_STI] = "sti", [X86ASM_OP_HLT] = "hlt", [X86ASM_OP_RET] = "ret",
    [X86ASM_OP_SBB] = "sbb", [X86ASM_OP_SETA] = "seta",
    [X86ASM_OP_SETAE] = "setae", [X86ASM_OP_SETB] = "setb",
    [X86ASM_OP_SETBE] = "setbe", [X86ASM_OP_SETE] = "sete",
    [X86ASM_OP_SETG] = "setg", [X86ASM_OP_SETGE] = "setge",
    [X86ASM_OP_SETL] = "setl", [X86ASM_OP_SETLE] = "setle",
    [X86ASM_OP_SETNE] = "setne", [X86ASM_OP_SETNO] = "setno",
    [X86ASM_OP_SETNP] = "setnp", [X86ASM_OP_SETNS] = "setns",
    [X86ASM_OP_SETO] = "seto", [X86ASM_OP_SETP] = "setp",
    [X86ASM_OP_SETS] = "sets",     [X86ASM_OP_SHL] = "shl", [X86ASM_OP_SHR] = "shr", [X86ASM_OP_SAR] = "sar",
    [X86ASM_OP_SHLD] = "shld", [X86ASM_OP_SHRD] = "shrd",
    [X86ASM_OP_ROL] = "rol", [X86ASM_OP_ROR] = "ror",
    [X86ASM_OP_RCL] = "rcl", [X86ASM_OP_RCR] = "rcr",

    [X86ASM_OP_STC] = "stc", [X86ASM_OP_STD] = "std",
    [X86ASM_OP_SYSCALL] = "syscall", [X86ASM_OP_SYSENTER] = "sysenter",
    [X86ASM_OP_SYSRET] = "sysret", [X86ASM_OP_SYSEXIT] = "sysexit",
    [X86ASM_OP_UD2] = "ud2",
    [X86ASM_OP_SUB] = "sub",
    [X86ASM_OP_TEST] = "test", [X86ASM_OP_XADD] = "xadd",
    [X86ASM_OP_XCHG] = "xchg",
    [X86ASM_OP_XOR] = "xor", [X86ASM_OP_XORPS] = "xorps",
    [X86ASM_OP_VADDPD] = "vaddpd", [X86ASM_OP_VADDPS] = "vaddps", [X86ASM_OP_VSUBPS] = "vsubps", [X86ASM_OP_VMULPS] = "vmulps", [X86ASM_OP_VDIVPS] = "vdivps", [X86ASM_OP_VMINPS] = "vminps", [X86ASM_OP_VMAXPS] = "vmaxps",
    [X86ASM_OP_VMOVMSKPS] = "vmovmskps", [X86ASM_OP_VMOVMSKPD] = "vmovmskpd", [X86ASM_OP_VPMOVMSKB] = "vpmovmskb", [X86ASM_OP_VPTEST] = "vptest",
    [X86ASM_OP_VSUBPD] = "vsubpd", [X86ASM_OP_VMULPD] = "vmulpd", [X86ASM_OP_VDIVPD] = "vdivpd", [X86ASM_OP_VMINPD] = "vminpd", [X86ASM_OP_VMAXPD] = "vmaxpd", [X86ASM_OP_VCMPPS] = "vcmpps", [X86ASM_OP_VCMPPD] = "vcmppd", [X86ASM_OP_VADDSS] = "vaddss", [X86ASM_OP_VSUBSS] = "vsubss", [X86ASM_OP_VMULSS] = "vmulss", [X86ASM_OP_VDIVSS] = "vdivss", [X86ASM_OP_VMINSS] = "vminss", [X86ASM_OP_VMAXSS] = "vmaxss", [X86ASM_OP_VADDSD] = "vaddsd", [X86ASM_OP_VSUBSD] = "vsubsd", [X86ASM_OP_VMULSD] = "vmulsd", [X86ASM_OP_VDIVSD] = "vdivsd", [X86ASM_OP_VMINSD] = "vminsd", [X86ASM_OP_VMAXSD] = "vmaxsd",
    [X86ASM_OP_VBLENDPD] = "vblendpd", [X86ASM_OP_VBLENDPS] = "vblendps",
    [X86ASM_OP_VPADDB] = "vpaddb", [X86ASM_OP_VPADDW] = "vpaddw",
    [X86ASM_OP_VAND] = "vand", [X86ASM_OP_VANDN] = "vandn",
    [X86ASM_OP_VMOVUPD] = "vmovupd", [X86ASM_OP_VMOVUPS] = "vmovups", [X86ASM_OP_VMOVD] = "vmovd", [X86ASM_OP_VMOVQ] = "vmovq", [X86ASM_OP_VMOVDQA] = "vmovdqa", [X86ASM_OP_VMOVDQU] = "vmovdqu", [X86ASM_OP_VMOVNTDQA] = "vmovntdqa", [X86ASM_OP_VMOVNTDQ] = "vmovntdq", [X86ASM_OP_VLDDQU] = "vlddqu", [X86ASM_OP_VMOVSS] = "vmovss", [X86ASM_OP_VMOVSD] = "vmovsd",
    [X86ASM_OP_VPADDD] = "vpaddd", [X86ASM_OP_VPCMPEQB] = "vpcmpeqb",
    [X86ASM_OP_VPCMPEQW] = "vpcmpeqw", [X86ASM_OP_VPCMPEQD] = "vpcmpeqd",
    [X86ASM_OP_VPCMPGTB] = "vpcmpgtb", [X86ASM_OP_VPCMPGTW] = "vpcmpgtw",
    [X86ASM_OP_VPCMPGTD] = "vpcmpgtd", [X86ASM_OP_VPCMPGTQ] = "vpcmpgtq",
    [X86ASM_OP_VPMULLW] = "vpmullw", [X86ASM_OP_VPMULHW] = "vpmulhw",
    [X86ASM_OP_VPMULHUW] = "vpmulhuw", [X86ASM_OP_VPMULUDQ] = "vpmuludq", [X86ASM_OP_VPMULLD] = "vpmulld",
    [X86ASM_OP_VPADDUSB] = "vpaddusb", [X86ASM_OP_VPADDUSW] = "vpaddusw",
    [X86ASM_OP_VPADDSB] = "vpaddsb", [X86ASM_OP_VPADDSW] = "vpaddsw",
    [X86ASM_OP_VPSUBUSB] = "vpsubusb", [X86ASM_OP_VPSUBUSW] = "vpsubusw",
    [X86ASM_OP_VPSUBSB] = "vpsubsb", [X86ASM_OP_VPSUBSW] = "vpsubsw",
    [X86ASM_OP_VPMINUB] = "vpminub", [X86ASM_OP_VPMAXUB] = "vpmaxub",
    [X86ASM_OP_VPMINSW] = "vpminsw", [X86ASM_OP_VPMAXSW] = "vpmaxsw",
    [X86ASM_OP_VPMINSB] = "vpminsb", [X86ASM_OP_VPMAXSB] = "vpmaxsb",
    [X86ASM_OP_VPMINUW] = "vpminuw", [X86ASM_OP_VPMAXUW] = "vpmaxuw",
    [X86ASM_OP_VPMINSD] = "vpminsd", [X86ASM_OP_VPMAXSD] = "vpmaxsd",
    [X86ASM_OP_VPMINUD] = "vpminud", [X86ASM_OP_VPMAXUD] = "vpmaxud",
    [X86ASM_OP_VPAVGB] = "vpavgb", [X86ASM_OP_VPAVGW] = "vpavgw", [X86ASM_OP_VPSADBW] = "vpsadbw",
    [X86ASM_OP_VPUNPCKLBW] = "vpunpcklbw", [X86ASM_OP_VPUNPCKLWD] = "vpunpcklwd", [X86ASM_OP_VPUNPCKLDQ] = "vpunpckldq",
    [X86ASM_OP_VPUNPCKHBW] = "vpunpckhbw", [X86ASM_OP_VPUNPCKHWD] = "vpunpckhwd", [X86ASM_OP_VPUNPCKHDQ] = "vpunpckhdq",
    [X86ASM_OP_VPACKSSWB] = "vpacksswb", [X86ASM_OP_VPACKSSDW] = "vpackssdw", [X86ASM_OP_VPACKUSWB] = "vpackuswb",
    [X86ASM_OP_VPSLLW] = "vpsllw", [X86ASM_OP_VPSLLD] = "vpslld", [X86ASM_OP_VPSLLQ] = "vpsllq",
    [X86ASM_OP_VPSRLW] = "vpsrlw", [X86ASM_OP_VPSRLD] = "vpsrld", [X86ASM_OP_VPSRLQ] = "vpsrlq",
    [X86ASM_OP_VPSRAW] = "vpsraw", [X86ASM_OP_VPSRAD] = "vpsrad",
    [X86ASM_OP_VPSLLDQ] = "vpslldq", [X86ASM_OP_VPSRLDQ] = "vpsrldq",
    [X86ASM_OP_VPSLLVD] = "vpsllvd", [X86ASM_OP_VPSRLVD] = "vpsrlvd", [X86ASM_OP_VPSRAVD] = "vpsravd",
    [X86ASM_OP_VPSLLVQ] = "vpsllvq", [X86ASM_OP_VPSRLVQ] = "vpsrlvq",
    [X86ASM_OP_VPCMPEQQ] = "vpcmpeqq", [X86ASM_OP_PCMPGTB] = "pcmpgtb", [X86ASM_OP_PCMPGTW] = "pcmpgtw", [X86ASM_OP_PCMPGTD] = "pcmpgtd", [X86ASM_OP_VPCMOV] = "vpcmov",
    [X86ASM_OP_VPROTB] = "vprotb", [X86ASM_OP_VOR] = "vor", [X86ASM_OP_VPADDQ] = "vpaddq", [X86ASM_OP_VPSUBB] = "vpsubb", [X86ASM_OP_VPSUBD] = "vpsubd", [X86ASM_OP_VPSUBQ] = "vpsubq", [X86ASM_OP_VPSUBW] = "vpsubw", [X86ASM_OP_VXOR] = "vxor",
    [X86ASM_OP_VXORPS] = "vxorps",
    [X86ASM_OP_VZEROUPPER] = "vzeroupper"
};

const char *x86asm_register_name(x86asm_register reg)
{
    if (reg >= X86ASM_REG_COUNT || register_names[reg] == NULL) {
        return "?";
    }
    return register_names[reg];
}

const char *x86asm_opcode_name(x86asm_opcode opcode)
{
    if (opcode >= X86ASM_OP_COUNT || opcode_names[opcode] == NULL) {
        return "invalid";
    }
    return opcode_names[opcode];
}

const char *x86asm_error_string(x86asm_error error)
{
    switch (error) {
    case X86ASM_OK: return "ok";
    case X86ASM_ERR_INVALID_MODE: return "invalid x86 mode";
    case X86ASM_ERR_TRUNCATED: return "truncated instruction";
    case X86ASM_ERR_UNRECOGNIZED: return "unrecognized instruction";
    case X86ASM_ERR_BUFFER_TOO_SMALL: return "output buffer too small";
    default: return "unknown x86asm error";
    }
}

static uint16_t read_u16(const uint8_t *p)
{
    return (uint16_t)p[0] | ((uint16_t)p[1] << 8);
}

static uint32_t read_u32(const uint8_t *p)
{
    return (uint32_t)p[0] | ((uint32_t)p[1] << 8) |
           ((uint32_t)p[2] << 16) | ((uint32_t)p[3] << 24);
}

static uint64_t read_u64(const uint8_t *p)
{
    return (uint64_t)read_u32(p) | ((uint64_t)read_u32(p + 4) << 32);
}

static x86asm_register register_for_width(int width, unsigned index)
{
    static const x86asm_register eight[] = {
        X86ASM_REG_AL, X86ASM_REG_CL, X86ASM_REG_DL, X86ASM_REG_BL,
        X86ASM_REG_AH, X86ASM_REG_CH, X86ASM_REG_DH, X86ASM_REG_BH,
        X86ASM_REG_R8B, X86ASM_REG_R9B, X86ASM_REG_R10B, X86ASM_REG_R11B,
        X86ASM_REG_R12B, X86ASM_REG_R13B, X86ASM_REG_R14B, X86ASM_REG_R15B
    };
    static const x86asm_register sixteen[] = {
        X86ASM_REG_AX, X86ASM_REG_CX, X86ASM_REG_DX, X86ASM_REG_BX,
        X86ASM_REG_SP, X86ASM_REG_BP, X86ASM_REG_SI, X86ASM_REG_DI,
        X86ASM_REG_R8W, X86ASM_REG_R9W, X86ASM_REG_R10W, X86ASM_REG_R11W,
        X86ASM_REG_R12W, X86ASM_REG_R13W, X86ASM_REG_R14W, X86ASM_REG_R15W
    };
    static const x86asm_register thirty_two[] = {
        X86ASM_REG_EAX, X86ASM_REG_ECX, X86ASM_REG_EDX, X86ASM_REG_EBX,
        X86ASM_REG_ESP, X86ASM_REG_EBP, X86ASM_REG_ESI, X86ASM_REG_EDI,
        X86ASM_REG_R8D, X86ASM_REG_R9D, X86ASM_REG_R10D, X86ASM_REG_R11D,
        X86ASM_REG_R12D, X86ASM_REG_R13D, X86ASM_REG_R14D, X86ASM_REG_R15D
    };
    static const x86asm_register sixty_four[] = {
        X86ASM_REG_RAX, X86ASM_REG_RCX, X86ASM_REG_RDX, X86ASM_REG_RBX,
        X86ASM_REG_RSP, X86ASM_REG_RBP, X86ASM_REG_RSI, X86ASM_REG_RDI,
        X86ASM_REG_R8, X86ASM_REG_R9, X86ASM_REG_R10, X86ASM_REG_R11,
        X86ASM_REG_R12, X86ASM_REG_R13, X86ASM_REG_R14, X86ASM_REG_R15
    };
    if (index >= 16) {
        return X86ASM_REG_NONE;
    }
    switch (width) {
    case 8: return eight[index];
    case 16: return sixteen[index];
    case 32: return thirty_two[index];
    case 64: return sixty_four[index];
    default: return X86ASM_REG_NONE;
    }
}

static x86asm_register register_for_byte_code(unsigned index, uint8_t rex)
{
    static const x86asm_register legacy[8] = {
        X86ASM_REG_AL, X86ASM_REG_CL, X86ASM_REG_DL, X86ASM_REG_BL,
        X86ASM_REG_AH, X86ASM_REG_CH, X86ASM_REG_DH, X86ASM_REG_BH
    };
    static const x86asm_register rex_bytes[8] = {
        X86ASM_REG_AL, X86ASM_REG_CL, X86ASM_REG_DL, X86ASM_REG_BL,
        X86ASM_REG_SPL, X86ASM_REG_BPL, X86ASM_REG_SIL, X86ASM_REG_DIL
    };
    if (index >= 16) return X86ASM_REG_NONE;
    if (index < 8) return rex == 0 ? legacy[index] : rex_bytes[index];
    return register_for_width(8, index);
}

static x86asm_register register_for_encoding(int width, unsigned index, uint8_t rex)
{
    return width == 8 ? register_for_byte_code(index, rex) : register_for_width(width, index);
}

static void set_register(x86asm_argument *arg, x86asm_register reg)
{
    arg->kind = X86ASM_ARG_REGISTER;
    arg->value.reg = reg;
}

static void set_immediate(x86asm_argument *arg, int64_t value)
{
    arg->kind = X86ASM_ARG_IMMEDIATE;
    arg->value.immediate = value;
}

static void set_relative(x86asm_argument *arg, int32_t value)
{
    arg->kind = X86ASM_ARG_RELATIVE;
    arg->value.relative = value;
}

static void set_memory(x86asm_argument *arg, x86asm_memory memory)
{
    arg->kind = X86ASM_ARG_MEMORY;
    arg->value.memory = memory;
}

static x86asm_error read_modrm(const uint8_t *bytes, size_t length, size_t *pos,
                               int mode, int address_mode, int width, uint8_t rex, uint8_t *modrm,
                               x86asm_argument *rm_arg, x86asm_register *reg)
{
    if (*pos >= length) {
        return X86ASM_ERR_TRUNCATED;
    }

    *modrm = bytes[(*pos)++];
    unsigned mod = *modrm >> 6;
    unsigned reg_field = (*modrm >> 3) & 7;
    unsigned rm_field = *modrm & 7;
    if ((rex & 0x04) != 0) reg_field |= 8;
    if ((rex & 0x01) != 0) rm_field |= 8;
    *reg = register_for_encoding(width, reg_field, rex);

    if (mod == 3) {
        set_register(rm_arg, register_for_encoding(width, rm_field, rex));
        return X86ASM_OK;
    }

    x86asm_memory memory = { X86ASM_REG_NONE, X86ASM_REG_NONE,
                             X86ASM_REG_NONE, 0, 0 };
    bool address64 = address_mode == 64;
    bool has_sib = width != 0 && address_mode != 16 && (rm_field & 7) == 4;
    unsigned base = rm_field;
    unsigned index = 4;

    if (address_mode == 16) {
        static const x86asm_register bases[8] = {
            X86ASM_REG_BX, X86ASM_REG_BX, X86ASM_REG_BP, X86ASM_REG_BP,
            X86ASM_REG_SI, X86ASM_REG_DI, X86ASM_REG_BP, X86ASM_REG_BX
        };
        static const x86asm_register indexes[8] = {
            X86ASM_REG_SI, X86ASM_REG_DI, X86ASM_REG_SI, X86ASM_REG_DI,
            X86ASM_REG_NONE, X86ASM_REG_NONE, X86ASM_REG_NONE, X86ASM_REG_NONE
        };
        memory.base = bases[rm_field];
        memory.index = indexes[rm_field];
        memory.scale = memory.index != X86ASM_REG_NONE ? 1 : 0;
        if (mod == 0 && rm_field == 6) {
            memory.base = X86ASM_REG_NONE;
        }
        if (mod == 0 && rm_field == 6) {
            if (*pos + 2 > length) return X86ASM_ERR_TRUNCATED;
            memory.displacement = (int16_t)read_u16(bytes + *pos);
            *pos += 2;
        } else if (mod == 1) {
            if (*pos >= length) return X86ASM_ERR_TRUNCATED;
            memory.displacement = (int8_t)bytes[(*pos)++];
        } else if (mod == 2) {
            if (*pos + 2 > length) return X86ASM_ERR_TRUNCATED;
            memory.displacement = (int16_t)read_u16(bytes + *pos);
            *pos += 2;
        }
        set_memory(rm_arg, memory);
        return X86ASM_OK;
    }

    if (has_sib) {
        if (*pos >= length) return X86ASM_ERR_TRUNCATED;
        uint8_t sib = bytes[(*pos)++];
        unsigned scale = sib >> 6;
        index = (sib >> 3) & 7;
        base = sib & 7;
        memory.scale = (uint8_t)(1u << scale);
        if ((rex & 0x02) != 0 && index != 4) index |= 8;
        if (index != 4) memory.index = register_for_width(mode == 64 ? 64 : 32, index);
        if ((rex & 0x01) != 0) base |= 8;
        if (!(mod == 0 && (base & 7) == 5)) memory.base = register_for_width(mode == 64 ? 64 : 32, base);
        if (mod == 0 && (base & 7) == 5 && (rex & 0x01) == 0) {
            memory.base = address64 ? X86ASM_REG_RIP : X86ASM_REG_NONE;
        }
    } else {
        if (mod == 0 && (rm_field & 7) == 5 && (rex & 0x01) == 0) {
            memory.base = address64 ? X86ASM_REG_RIP : X86ASM_REG_NONE;
        } else {
            memory.base = register_for_width(mode == 64 ? 64 : 32, rm_field);
        }
    }

    if (mod == 0 && ((!has_sib && (rm_field & 7) == 5 && (rex & 0x01) == 0) ||
                     (has_sib && (base & 7) == 5 && (rex & 0x01) == 0))) {
        if (*pos + 4 > length) return X86ASM_ERR_TRUNCATED;
        memory.displacement = (int32_t)read_u32(bytes + *pos);
        *pos += 4;
    } else if (mod == 1) {
        if (*pos >= length) return X86ASM_ERR_TRUNCATED;
        memory.displacement = (int8_t)bytes[(*pos)++];
    } else if (mod == 2) {
        if (*pos + 4 > length) return X86ASM_ERR_TRUNCATED;
        memory.displacement = (int32_t)read_u32(bytes + *pos);
        *pos += 4;
    }

    set_memory(rm_arg, memory);
    return X86ASM_OK;
}

static x86asm_opcode condition_opcode(unsigned condition)
{
    static const x86asm_opcode conditions[16] = {
        X86ASM_OP_JO, X86ASM_OP_JNO, X86ASM_OP_JB, X86ASM_OP_JAE,
        X86ASM_OP_JE, X86ASM_OP_JNE, X86ASM_OP_JBE, X86ASM_OP_JA,
        X86ASM_OP_JS, X86ASM_OP_JNS, X86ASM_OP_JP, X86ASM_OP_JNP,
        X86ASM_OP_JL, X86ASM_OP_JGE, X86ASM_OP_JLE, X86ASM_OP_JG
    };
    return conditions[condition & 15];
}

static x86asm_opcode cmov_opcode(unsigned condition)
{
    static const x86asm_opcode operations[16] = {
        X86ASM_OP_CMOVO, X86ASM_OP_CMOVNO, X86ASM_OP_CMOVB, X86ASM_OP_CMOVAE,
        X86ASM_OP_CMOVE, X86ASM_OP_CMOVNE, X86ASM_OP_CMOVBE, X86ASM_OP_CMOVA,
        X86ASM_OP_CMOVS, X86ASM_OP_CMOVNS, X86ASM_OP_CMOVP, X86ASM_OP_CMOVNP,
        X86ASM_OP_CMOVL, X86ASM_OP_CMOVGE, X86ASM_OP_CMOVLE, X86ASM_OP_CMOVG
    };
    return operations[condition & 15];
}

static x86asm_opcode set_opcode(unsigned condition)
{
    static const x86asm_opcode operations[16] = {
        X86ASM_OP_SETO, X86ASM_OP_SETNO, X86ASM_OP_SETB, X86ASM_OP_SETAE,
        X86ASM_OP_SETE, X86ASM_OP_SETNE, X86ASM_OP_SETBE, X86ASM_OP_SETA,
        X86ASM_OP_SETS, X86ASM_OP_SETNS, X86ASM_OP_SETP, X86ASM_OP_SETNP,
        X86ASM_OP_SETL, X86ASM_OP_SETGE, X86ASM_OP_SETLE, X86ASM_OP_SETG
    };
    return operations[condition & 15];
}

static x86asm_opcode arithmetic_group_opcode(unsigned group)
{
    static const x86asm_opcode operations[8] = {
        X86ASM_OP_ADD, X86ASM_OP_OR, X86ASM_OP_ADC, X86ASM_OP_SBB,
        X86ASM_OP_AND, X86ASM_OP_SUB, X86ASM_OP_XOR, X86ASM_OP_CMP
    };
    return operations[group & 7];
}

static x86asm_opcode shift_group_opcode(unsigned group)
{
    static const x86asm_opcode operations[8] = {
        X86ASM_OP_ROL, X86ASM_OP_ROR, X86ASM_OP_RCL, X86ASM_OP_RCR,
        X86ASM_OP_SHL, X86ASM_OP_SHR, X86ASM_OP_SHL, X86ASM_OP_SAR
    };
    return group < 8u ? operations[group] : X86ASM_OP_INVALID;
}

static x86asm_register vector_register(bool ymm, unsigned index)
{
    if (index >= 16) return X86ASM_REG_NONE;
    return (x86asm_register)((unsigned)(ymm ? X86ASM_REG_YMM0 : X86ASM_REG_XMM0) + index);
}

static x86asm_register vector_register_length(unsigned length, unsigned index)
{
    if (length == 512 && index < 32) return (x86asm_register)((unsigned)X86ASM_REG_ZMM0 + index);
    return vector_register(length == 256, index);
}

static void set_vector(x86asm_argument *argument, bool ymm, unsigned index)
{
    argument->kind = X86ASM_ARG_REGISTER;
    argument->value.reg = vector_register(ymm, index);
}

static void set_vector_length(x86asm_argument *argument, unsigned length, unsigned index)
{
    argument->kind = X86ASM_ARG_REGISTER;
    argument->value.reg = vector_register_length(length, index);
}

static x86asm_error read_vex_memory(const uint8_t *bytes, size_t length, size_t *pos,
                                    int mode, uint8_t modrm, bool vex_x, bool vex_b,
                                    x86asm_memory *memory)
{
    unsigned mod = modrm >> 6;
    unsigned rm = modrm & 7;
    bool has_sib = mod != 3 && rm == 4;
    unsigned base = rm;
    unsigned index = 4;
    *memory = (x86asm_memory){ X86ASM_REG_NONE, X86ASM_REG_NONE,
                               X86ASM_REG_NONE, 0, 0 };

    if (has_sib) {
        if (*pos >= length) return X86ASM_ERR_TRUNCATED;
        uint8_t sib = bytes[(*pos)++];
        memory->scale = (uint8_t)(1u << (sib >> 6));
        index = (sib >> 3) & 7;
        base = sib & 7;
        if (!vex_x) index |= 8;
        if (index != 4) memory->index = register_for_width(mode == 64 ? 64 : 32, index);
        if (!vex_b) base |= 8;
        if (!(mod == 0 && (base & 7) == 5)) {
            memory->base = register_for_width(mode == 64 ? 64 : 32, base);
        } else {
            memory->base = mode == 64 ? X86ASM_REG_RIP : X86ASM_REG_NONE;
        }
    } else {
        if (!vex_b) rm |= 8;
        if (mod == 0 && (rm & 7) == 5) {
            memory->base = mode == 64 ? X86ASM_REG_RIP : X86ASM_REG_NONE;
        } else {
            memory->base = register_for_width(mode == 64 ? 64 : 32, rm);
        }
    }

    if (mod == 0 && ((!has_sib && (rm & 7) == 5) ||
                     (has_sib && (base & 7) == 5))) {
        if (*pos + 4 > length) return X86ASM_ERR_TRUNCATED;
        memory->displacement = (int32_t)read_u32(bytes + *pos);
        *pos += 4;
    } else if (mod == 1) {
        if (*pos >= length) return X86ASM_ERR_TRUNCATED;
        memory->displacement = (int8_t)bytes[(*pos)++];
    } else if (mod == 2) {
        if (*pos + 4 > length) return X86ASM_ERR_TRUNCATED;
        memory->displacement = (int32_t)read_u32(bytes + *pos);
        *pos += 4;
    }
    return X86ASM_OK;
}

static x86asm_error decode_sse(const uint8_t *bytes, size_t length, size_t pos,
                               int mode, uint8_t rex, uint8_t second,
                               x86asm_instruction *instruction)
{
    bool p66 = false;
    bool pf2 = false;
    bool pf3 = false;
    for (unsigned i = 0; i < 4; ++i) {
        if ((instruction->prefixes[i] & 0xFFu) == X86ASM_PREFIX_DATA16) p66 = true;
        if ((instruction->prefixes[i] & 0xFFu) == X86ASM_PREFIX_REPN) pf2 = true;
        if ((instruction->prefixes[i] & 0xFFu) == X86ASM_PREFIX_REP) pf3 = true;
    }
    if ((second == 0x12 || second == 0x13 || second == 0x16 || second == 0x17) &&
        !pf2 && !pf3) {
        bool high = second == 0x16 || second == 0x17;
        bool store = second == 0x13 || second == 0x17;
        uint8_t modrm;
        x86asm_register ignored_register;
        x86asm_argument memory_argument = { X86ASM_ARG_NONE, { 0 } };
        x86asm_error error = read_modrm(bytes, length, &pos, mode, mode, 64, rex,
                                         &modrm, &memory_argument, &ignored_register);
        if (error != X86ASM_OK) return error;
        if ((modrm >> 6) == 3u) return X86ASM_ERR_UNRECOGNIZED;
        unsigned vector_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
        if (p66) instruction->opcode = high ? X86ASM_OP_MOVHPD : X86ASM_OP_MOVLPD;
        else instruction->opcode = high ? X86ASM_OP_MOVHPS : X86ASM_OP_MOVLPS;
        instruction->mode = mode;
        instruction->address_size = mode;
        instruction->data_size = 128;
        instruction->memory_bytes = 8;
        instruction->encoded_opcode = ((uint32_t)0x0F << 8) | second;
        if (store) {
            instruction->arguments[0] = memory_argument;
            set_vector(&instruction->arguments[1], false, vector_index);
        } else {
            set_vector(&instruction->arguments[0], false, vector_index);
            instruction->arguments[1] = memory_argument;
        }
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if ((second == 0x50 && !pf2 && !pf3) || (second == 0xD7 && p66 && !pf2 && !pf3)) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        uint8_t modrm = bytes[pos++];
        if ((modrm >> 6) != 3u) return X86ASM_ERR_UNRECOGNIZED;
        unsigned destination_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
        unsigned source_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
        unsigned destination_width = (mode == 64 && (rex & 8u) != 0) ? 64u : 32u;
        instruction->opcode = second == 0xD7 ? X86ASM_OP_PMOVMSKB : (p66 ? X86ASM_OP_MOVMSKPD : X86ASM_OP_MOVMSKPS);
        instruction->mode = mode;
        instruction->address_size = mode;
        instruction->data_size = (int)destination_width;
        instruction->memory_bytes = 0;
        instruction->encoded_opcode = ((uint32_t)0x0F << 8) | second;
        set_register(&instruction->arguments[0], register_for_width((int)destination_width, destination_index));
        set_vector(&instruction->arguments[1], false, source_index);
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (second == 0x70 && (p66 || pf2 || pf3) && !(p66 && pf2) && !(p66 && pf3) && !(pf2 && pf3)) {
        uint8_t modrm;
        x86asm_register ignored_register;
        x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
        x86asm_error error = read_modrm(bytes, length, &pos, mode, mode, 64, rex,
                                         &modrm, &source, &ignored_register);
        if (error != X86ASM_OK) return error;
        unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
        if ((modrm >> 6) == 3u) {
            unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
            set_vector(&source, false, rm_index);
        }
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = p66 ? X86ASM_OP_PSHUFD : (pf2 ? X86ASM_OP_PSHUFLW : X86ASM_OP_PSHUFHW);
        instruction->mode = mode;
        instruction->address_size = mode;
        instruction->data_size = 128;
        instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
        set_vector(&instruction->arguments[0], false, reg_index);
        instruction->arguments[1] = source;
        set_immediate(&instruction->arguments[2], bytes[pos++]);
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if ((second == 0x71 || second == 0x72 || second == 0x73) && p66 && !pf2 && !pf3) {
        uint8_t modrm;
        x86asm_register ignored_register;
        x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
        x86asm_error error = read_modrm(bytes, length, &pos, mode, mode, 128, rex,
                                         &modrm, &source, &ignored_register);
        if (error != X86ASM_OK) return error;
        unsigned group = (modrm >> 3) & 7u;
        if (second == 0x71 && group != 2u && group != 4u && group != 6u) return X86ASM_ERR_UNRECOGNIZED;
        if (second == 0x72 && group != 2u && group != 4u && group != 6u) return X86ASM_ERR_UNRECOGNIZED;
        if (second == 0x73 && group != 2u && group != 3u && group != 6u && group != 7u) return X86ASM_ERR_UNRECOGNIZED;
        if (second == 0x71) instruction->opcode = group == 2u ? X86ASM_OP_PSRLW : group == 4u ? X86ASM_OP_PSRAW : X86ASM_OP_PSLLW;
        else if (second == 0x72) instruction->opcode = group == 2u ? X86ASM_OP_PSRLD : group == 4u ? X86ASM_OP_PSRAD : X86ASM_OP_PSLLD;
        else if (group == 2u) instruction->opcode = X86ASM_OP_PSRLQ;
        else if (group == 3u) instruction->opcode = X86ASM_OP_PSRLDQ;
        else if (group == 6u) instruction->opcode = X86ASM_OP_PSLLQ;
        else instruction->opcode = X86ASM_OP_PSLLDQ;
        if ((modrm >> 6) == 3u) {
            unsigned destination_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
            set_vector(&instruction->arguments[0], false, destination_index);
        } else {
            instruction->arguments[0] = source;
        }
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        set_immediate(&instruction->arguments[1], bytes[pos++]);
        instruction->mode = mode;
        instruction->address_size = mode;
        instruction->data_size = 128;
        instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
        instruction->encoded_opcode = ((uint32_t)0x0F << 8) | second;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (second == 0x7E && pf3 && !p66 && !pf2) {
        uint8_t modrm;
        x86asm_register ignored_register;
        x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
        x86asm_error error = read_modrm(bytes, length, &pos, mode, mode, 64, rex,
                                         &modrm, &source, &ignored_register);
        if (error != X86ASM_OK) return error;
        unsigned destination_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
        if ((modrm >> 6) == 3u) {
            unsigned source_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
            set_vector(&source, false, source_index);
        }
        instruction->opcode = X86ASM_OP_MOVQ;
        instruction->mode = mode;
        instruction->address_size = mode;
        instruction->data_size = 64;
        instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 8 : 0;
        instruction->encoded_opcode = ((uint32_t)0x0F << 8) | second;
        set_vector(&instruction->arguments[0], false, destination_index);
        instruction->arguments[1] = source;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (second == 0xD6 && p66 && !pf2 && !pf3) {
        uint8_t modrm;
        x86asm_register ignored_register;
        x86asm_argument destination = { X86ASM_ARG_NONE, { 0 } };
        x86asm_error error = read_modrm(bytes, length, &pos, mode, mode, 64, rex,
                                         &modrm, &destination, &ignored_register);
        if (error != X86ASM_OK) return error;
        unsigned source_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
        if ((modrm >> 6) == 3u) {
            unsigned destination_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
            set_vector(&destination, false, destination_index);
        }
        instruction->opcode = X86ASM_OP_MOVQ;
        instruction->mode = mode;
        instruction->address_size = mode;
        instruction->data_size = 64;
        instruction->memory_bytes = destination.kind == X86ASM_ARG_MEMORY ? 8 : 0;
        instruction->encoded_opcode = ((uint32_t)0x0F << 8) | second;
        instruction->arguments[0] = destination;
        set_vector(&instruction->arguments[1], false, source_index);
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if ((second == 0x6E || second == 0x7E) && !p66 && !pf2 && !pf3) {
        uint8_t modrm;
        x86asm_register reg;
        unsigned transfer_width = (rex & 8u) != 0 ? 64u : 32u;
        x86asm_argument rm_argument = { X86ASM_ARG_NONE, { 0 } };
        x86asm_error error = read_modrm(bytes, length, &pos, mode, mode, (int)transfer_width,
                                         rex, &modrm, &rm_argument, &reg);
        if (error != X86ASM_OK) return error;
        instruction->opcode = transfer_width == 64u ? X86ASM_OP_MOVQ : X86ASM_OP_MOVD;
        instruction->mode = mode;
        instruction->address_size = mode;
        instruction->data_size = (int)transfer_width;
        instruction->memory_bytes = rm_argument.kind == X86ASM_ARG_MEMORY ? (int)(transfer_width / 8u) : 0;
        instruction->encoded_opcode = ((uint32_t)0x0F << 8) | second;
        unsigned vector_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
        if (second == 0x6E) {
            set_vector(&instruction->arguments[0], false, vector_index);
            instruction->arguments[1] = rm_argument;
        } else {
            instruction->arguments[0] = rm_argument;
            set_vector(&instruction->arguments[1], false, vector_index);
        }
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if ((second == 0xC4 || second == 0xC5) && p66) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        uint8_t modrm = bytes[pos++];
        unsigned mod = modrm >> 6;
        unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
        unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
        x86asm_argument operand = { X86ASM_ARG_NONE, { 0 } };
        if (mod == 3u) set_register(&operand, register_for_encoding(second == 0xC4 ? 32 : (mode == 64 ? 64 : 32), rm_index, rex));
        else {
            x86asm_memory memory;
            x86asm_error error = read_vex_memory(bytes, length, &pos, mode, modrm, (rex & 2u) == 0, (rex & 1u) == 0, &memory);
            if (error != X86ASM_OK) return error;
            set_memory(&operand, memory);
        }
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        instruction->mode = mode;
        instruction->address_size = mode;
        instruction->data_size = 128;
        instruction->encoded_opcode = ((uint32_t)0x0F << 8) | second;
        instruction->opcode = second == 0xC4 ? X86ASM_OP_PINSRW : X86ASM_OP_PEXTRW;
        instruction->memory_bytes = operand.kind == X86ASM_ARG_MEMORY ? 2 : 0;
        if (second == 0xC4) {
            set_vector(&instruction->arguments[0], false, reg_index);
            instruction->arguments[1] = operand;
        } else {
            instruction->arguments[0] = operand;
            set_vector(&instruction->arguments[1], false, reg_index);
        }
        set_immediate(&instruction->arguments[2], bytes[pos++]);
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (pos >= length) return X86ASM_ERR_TRUNCATED;
    uint8_t modrm = bytes[pos++];
    unsigned mod = modrm >> 6;
    unsigned reg = ((modrm >> 3) & 7) | ((rex & 4) != 0 ? 8u : 0u);
    unsigned rm = (modrm & 7) | ((rex & 1) != 0 ? 8u : 0u);
    x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
    if (mod == 3) set_vector(&source, false, rm);
    else {
        x86asm_memory memory;
        x86asm_error error = read_vex_memory(bytes, length, &pos, mode, modrm,
                                             (rex & 2) == 0, (rex & 1) == 0, &memory);
        if (error != X86ASM_OK) return error;
        set_memory(&source, memory);
    }
    if ((second == 0x10 || second == 0x11) && pf3 && !p66 && !pf2) {
        instruction->opcode = X86ASM_OP_MOVSS;
    } else if ((second == 0x10 || second == 0x11) && pf2 && !p66 && !pf3) {
        instruction->opcode = X86ASM_OP_MOVSD_SCALAR;
    } else if ((second == 0x10 || second == 0x11) && !p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_MOVUPS;
    } else if ((second == 0x10 || second == 0x11) && p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_MOVUPD;
    } else if (second == 0x58 && pf3 && !p66 && !pf2) {
        instruction->opcode = X86ASM_OP_ADDSS;
    } else if (second == 0x5C && pf3 && !p66 && !pf2) {
        instruction->opcode = X86ASM_OP_SUBSS;
    } else if (second == 0x58 && pf2 && !p66 && !pf3) {
        instruction->opcode = X86ASM_OP_ADDSD;
    } else if (second == 0x5C && pf2 && !p66 && !pf3) {
        instruction->opcode = X86ASM_OP_SUBSD;
    } else if (second == 0x59 && pf3 && !p66 && !pf2) {
        instruction->opcode = X86ASM_OP_MULSS;
    } else if (second == 0x5E && pf3 && !p66 && !pf2) {
        instruction->opcode = X86ASM_OP_DIVSS;
    } else if (second == 0x5D && pf3 && !p66 && !pf2) {
        instruction->opcode = X86ASM_OP_MINSS;
    } else if (second == 0x5F && pf3 && !p66 && !pf2) {
        instruction->opcode = X86ASM_OP_MAXSS;
    } else if (second == 0x59 && pf2 && !p66 && !pf3) {
        instruction->opcode = X86ASM_OP_MULSD;
    } else if (second == 0x5E && pf2 && !p66 && !pf3) {
        instruction->opcode = X86ASM_OP_DIVSD;
    } else if (second == 0x5D && pf2 && !p66 && !pf3) {
        instruction->opcode = X86ASM_OP_MINSD;
    } else if (second == 0x5F && pf2 && !p66 && !pf3) {
        instruction->opcode = X86ASM_OP_MAXSD;
    } else if (second == 0x58 && !p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_ADDPS;
    } else if (second == 0x57 && !p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_XORPS;
    } else if (second == 0x5C && !p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_SUBPS;
    } else if (second == 0x59 && !p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_MULPS;
    } else if (second == 0x5E && !p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_DIVPS;
    } else if (second == 0x5D && !p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_MINPS;
    } else if (second == 0x5F && !p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_MAXPS;
    } else if (second == 0x58 && p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_ADDPD;
    } else if (second == 0x5C && p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_SUBPD;
    } else if (second == 0x59 && p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_MULPD;
    } else if (second == 0x5E && p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_DIVPD;
    } else if (second == 0x5D && p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_MINPD;
    } else if (second == 0x5F && p66 && !pf2 && !pf3) {
        instruction->opcode = X86ASM_OP_MAXPD;
    } else if (second == 0xC2 && !pf2 && !pf3) {
        if (p66) instruction->opcode = X86ASM_OP_CMPPD;
        else if (!p66) instruction->opcode = X86ASM_OP_CMPPS;
        else return X86ASM_ERR_UNRECOGNIZED;
    } else if (second == 0xE7 && p66 && !pf2 && !pf3) {
        if (source.kind != X86ASM_ARG_MEMORY) return X86ASM_ERR_UNRECOGNIZED;
        instruction->opcode = X86ASM_OP_MOVNTDQ;
    } else if ((second == 0x6F || second == 0x7F) && (p66 || pf3)) {
        instruction->opcode = p66 ? X86ASM_OP_MOVDQA : X86ASM_OP_MOVDQU;
    } else if (p66 && second == 0xFC) {
        instruction->opcode = X86ASM_OP_PADDB;
    } else if (p66 && second == 0xFD) {
        instruction->opcode = X86ASM_OP_PADDW;
    } else if (p66 && second == 0xFE) {
        instruction->opcode = X86ASM_OP_PADDD;
    } else if (p66 && second == 0xF8) {
        instruction->opcode = X86ASM_OP_PSUBB;
    } else if (p66 && second == 0xF9) {
        instruction->opcode = X86ASM_OP_PSUBW;
    } else if (p66 && second == 0xFA) {
        instruction->opcode = X86ASM_OP_PSUBD;
    } else if (p66 && second == 0x74) {
        instruction->opcode = X86ASM_OP_PCMPEQB;
    } else if (p66 && second == 0x75) {
        instruction->opcode = X86ASM_OP_PCMPEQW;
    } else if (p66 && second == 0x76) {
        instruction->opcode = X86ASM_OP_PCMPEQD;
    } else if (p66 && second == 0x64) {
        instruction->opcode = X86ASM_OP_PCMPGTB;
    } else if (p66 && second == 0x65) {
        instruction->opcode = X86ASM_OP_PCMPGTW;
    } else if (p66 && second == 0x66) {
        instruction->opcode = X86ASM_OP_PCMPGTD;
    } else if (p66 && second == 0xD5) {
        instruction->opcode = X86ASM_OP_PMULLW;
    } else if (p66 && second == 0xE4) {
        instruction->opcode = X86ASM_OP_PMULHUW;
    } else if (p66 && second == 0xE5) {
        instruction->opcode = X86ASM_OP_PMULHW;
    } else if (p66 && second == 0xF4) {
        instruction->opcode = X86ASM_OP_PMULUDQ;
    } else if (p66 && second == 0x60) {
        instruction->opcode = X86ASM_OP_PUNPCKLBW;
    } else if (p66 && second == 0x61) {
        instruction->opcode = X86ASM_OP_PUNPCKLWD;
    } else if (p66 && second == 0x62) {
        instruction->opcode = X86ASM_OP_PUNPCKLDQ;
    } else if (p66 && second == 0x63) {
        instruction->opcode = X86ASM_OP_PACKSSWB;
    } else if (p66 && second == 0x67) {
        instruction->opcode = X86ASM_OP_PACKUSWB;
    } else if (p66 && second == 0x68) {
        instruction->opcode = X86ASM_OP_PUNPCKHBW;
    } else if (p66 && second == 0x69) {
        instruction->opcode = X86ASM_OP_PUNPCKHWD;
    } else if (p66 && second == 0x6A) {
        instruction->opcode = X86ASM_OP_PUNPCKHDQ;
    } else if (p66 && second == 0x6B) {
        instruction->opcode = X86ASM_OP_PACKSSDW;
    } else if (p66 && second == 0xDC) {
        instruction->opcode = X86ASM_OP_PADDUSB;
    } else if (p66 && second == 0xDD) {
        instruction->opcode = X86ASM_OP_PADDUSW;
    } else if (p66 && second == 0xEC) {
        instruction->opcode = X86ASM_OP_PADDSB;
    } else if (p66 && second == 0xED) {
        instruction->opcode = X86ASM_OP_PADDSW;
    } else if (p66 && second == 0xD8) {
        instruction->opcode = X86ASM_OP_PSUBUSB;
    } else if (p66 && second == 0xD9) {
        instruction->opcode = X86ASM_OP_PSUBUSW;
    } else if (p66 && second == 0xE8) {
        instruction->opcode = X86ASM_OP_PSUBSB;
    } else if (p66 && second == 0xE9) {
        instruction->opcode = X86ASM_OP_PSUBSW;
    } else if (p66 && second == 0xDA) {
        instruction->opcode = X86ASM_OP_PMINUB;
    } else if (p66 && second == 0xDE) {
        instruction->opcode = X86ASM_OP_PMAXUB;
    } else if (p66 && second == 0xEA) {
        instruction->opcode = X86ASM_OP_PMINSW;
    } else if (p66 && second == 0xEE) {
        instruction->opcode = X86ASM_OP_PMAXSW;
    } else if (p66 && second == 0xE0) {
        instruction->opcode = X86ASM_OP_PAVGB;
    } else if (p66 && second == 0xE3) {
        instruction->opcode = X86ASM_OP_PAVGW;
    } else if (p66 && second == 0xF6) {
        instruction->opcode = X86ASM_OP_PSADBW;
    } else if (p66 && second == 0xDB) {
        instruction->opcode = X86ASM_OP_PAND;
    } else if (p66 && second == 0xEB) {
        instruction->opcode = X86ASM_OP_POR;
    } else if (p66 && second == 0xEF) {
        instruction->opcode = X86ASM_OP_PXOR;
    } else {
        return X86ASM_ERR_UNRECOGNIZED;
    }
    instruction->mode = mode;
    instruction->address_size = mode;
    instruction->data_size = 128;
    instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
    if (instruction->opcode == X86ASM_OP_MOVSS) instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 4 : 0;
    if (instruction->opcode == X86ASM_OP_MOVSD_SCALAR) instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 8 : 0;
    if (instruction->opcode == X86ASM_OP_ADDSS || instruction->opcode == X86ASM_OP_SUBSS) instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 4 : 0;
    if (instruction->opcode == X86ASM_OP_ADDSD || instruction->opcode == X86ASM_OP_SUBSD || instruction->opcode == X86ASM_OP_MULSD || instruction->opcode == X86ASM_OP_DIVSD) instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 8 : 0;
    if (instruction->opcode == X86ASM_OP_MULSS || instruction->opcode == X86ASM_OP_DIVSS || instruction->opcode == X86ASM_OP_MINSS || instruction->opcode == X86ASM_OP_MAXSS) instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 4 : 0;
    instruction->encoded_opcode = ((uint32_t)0x0F << 8) | second;
    if (second == 0x11) {
        instruction->arguments[0] = source;
        set_vector(&instruction->arguments[1], false, reg);
    } else if (second == 0x10) {
        set_vector(&instruction->arguments[0], false, reg);
        instruction->arguments[1] = source;
    } else if (instruction->opcode == X86ASM_OP_MOVNTDQ ||
               ((instruction->opcode == X86ASM_OP_MOVDQA || instruction->opcode == X86ASM_OP_MOVDQU) && second == 0x7F)) {
        instruction->arguments[0] = source;
        set_vector(&instruction->arguments[1], false, reg);
    } else {
        set_vector(&instruction->arguments[0], false, reg);
        instruction->arguments[1] = source;
    }
    if (second == 0xC2) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        if ((bytes[pos] & 0xF8u) != 0u) return X86ASM_ERR_UNRECOGNIZED;
        set_immediate(&instruction->arguments[2], bytes[pos++]);
    }
    instruction->length = (int)pos;
    return X86ASM_OK;
}

static x86asm_error decode_xop(const uint8_t *bytes, size_t length, size_t pos,
                               int mode, x86asm_instruction *instruction)
{
    if (pos + 3 >= length) return X86ASM_ERR_TRUNCATED;
    uint8_t p0 = bytes[++pos];
    uint8_t p1 = bytes[++pos];
    uint8_t opcode = bytes[++pos];
    ++pos;
    unsigned map = p0 & 0x1F;
    bool ymm = (p1 & 4) != 0;
    bool r = (p0 & 0x80) != 0;
    bool x = (p0 & 0x40) != 0;
    bool b = (p0 & 0x20) != 0;
    unsigned vvvv = (~p1 >> 3) & 0x0F;
    uint8_t rex = 0;
    if (!r) rex |= 0x04;
    if (!x) rex |= 0x02;
    if (!b) rex |= 0x01;

    if (map == 10 && opcode == 0x10) {
        uint8_t modrm;
        x86asm_register reg;
        x86asm_error error = read_modrm(bytes, length, &pos, mode, mode, mode == 64 ? 64 : 32,
                                         rex, &modrm, &instruction->arguments[1], &reg);
        if (error != X86ASM_OK) return error;
        if (pos + 4 > length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = X86ASM_OP_BEXTR;
        instruction->mode = mode;
        instruction->address_size = mode;
        instruction->data_size = mode == 64 ? 64 : 32;
        instruction->prefixes[0] = 0x8F;
        instruction->prefixes[1] = p0;
        instruction->prefixes[2] = p1;
        instruction->encoded_opcode = ((uint32_t)0x8F << 16) | ((uint32_t)p0 << 8) | p1;
        set_register(&instruction->arguments[0], reg);
        set_immediate(&instruction->arguments[2], (int32_t)read_u32(bytes + pos));
        pos += 4;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (map != 8 || (opcode != 0xA2 && opcode != 0xC0)) return X86ASM_ERR_UNRECOGNIZED;
    if (pos >= length) return X86ASM_ERR_TRUNCATED;
    uint8_t modrm = bytes[pos++];
    unsigned mod = modrm >> 6;
    unsigned reg = ((modrm >> 3) & 7) | (r ? 0u : 8u);
    unsigned rm = (modrm & 7) | (b ? 0u : 8u);
    x86asm_argument rm_argument = { X86ASM_ARG_NONE, { 0 } };
    if (mod == 3) set_vector(&rm_argument, ymm, rm);
    else {
        x86asm_memory memory;
        x86asm_error error = read_vex_memory(bytes, length, &pos, mode, modrm, x, b, &memory);
        if (error != X86ASM_OK) return error;
        set_memory(&rm_argument, memory);
    }
    instruction->opcode = opcode == 0xA2 ? X86ASM_OP_VPCMOV : X86ASM_OP_VPROTB;
    instruction->mode = mode;
    instruction->address_size = mode;
    instruction->data_size = ymm ? 256 : 128;
    instruction->memory_bytes = rm_argument.kind == X86ASM_ARG_MEMORY ? (ymm ? 32 : 16) : 0;
    instruction->prefixes[0] = 0x8F;
    instruction->prefixes[1] = p0;
    instruction->prefixes[2] = p1;
    instruction->encoded_opcode = ((uint32_t)0x8F << 16) | ((uint32_t)p0 << 8) | p1;
    set_vector(&instruction->arguments[0], ymm, reg);
    set_vector(&instruction->arguments[1], ymm, vvvv);
    instruction->arguments[2] = rm_argument;
    if (opcode == 0xC0) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        set_immediate(&instruction->arguments[3], bytes[pos++]);
    }
    instruction->length = (int)pos;
    return X86ASM_OK;
}

static x86asm_error decode_evex(const uint8_t *bytes, size_t length, size_t pos,
                                int mode, x86asm_instruction *instruction)
{
    if (pos + 4 >= length) return X86ASM_ERR_TRUNCATED;
    uint8_t p0 = bytes[++pos];
    uint8_t p1 = bytes[++pos];
    uint8_t p2 = bytes[++pos];
    uint8_t opcode = bytes[++pos];
    unsigned map = p0 & 7;
    unsigned pp = p1 & 3;
    unsigned ll = (p2 >> 5) & 3;
    unsigned length_bits = ll == 0 ? 128u : (ll == 1 ? 256u : (ll == 2 ? 512u : 0u));
    bool r = (p0 & 0x10) != 0;
    bool x = (p0 & 0x40) != 0;
    bool b = (p0 & 0x20) != 0;
    bool r2 = (p0 & 0x80) != 0;
    uint8_t vvvv = (uint8_t)(((~p1 >> 3) & 0x0F) | (((~p2 >> 3) & 1) << 4));
    if (map != 1 || length_bits == 0 || pos >= length) return X86ASM_ERR_UNRECOGNIZED;

    x86asm_opcode operation;
    if (opcode == 0x58 && pp == 0) operation = X86ASM_OP_VADDPS;
    else if (opcode == 0x58 && pp == 1) operation = X86ASM_OP_VADDPD;
    else if (opcode == 0x57 && pp == 0) operation = X86ASM_OP_VXORPS;
    else if (opcode == 0xFE && pp == 1) operation = X86ASM_OP_VPADDD;
    else if (opcode == 0x10 && (pp == 0 || pp == 1)) operation = pp == 0 ? X86ASM_OP_VMOVUPS : X86ASM_OP_VMOVUPD;
    else if (opcode == 0x11 && (pp == 0 || pp == 1)) operation = pp == 0 ? X86ASM_OP_VMOVUPS : X86ASM_OP_VMOVUPD;
    else return X86ASM_ERR_UNRECOGNIZED;

    if (pos >= length) return X86ASM_ERR_TRUNCATED;
    uint8_t modrm = bytes[++pos];
    ++pos;
    unsigned mod = modrm >> 6;
    unsigned reg = ((modrm >> 3) & 7) | (r ? 0u : 8u) | (r2 ? 0u : 16u);
    unsigned rm = (modrm & 7) | (b ? 0u : 8u);
    x86asm_argument rm_argument = { X86ASM_ARG_NONE, { 0 } };
    if (mod == 3) set_vector_length(&rm_argument, length_bits, rm);
    else {
        x86asm_memory memory;
        x86asm_error error = read_vex_memory(bytes, length, &pos, mode, modrm, x, b, &memory);
        if (error != X86ASM_OK) return error;
        set_memory(&rm_argument, memory);
    }

    instruction->opcode = operation;
    instruction->mode = mode;
    instruction->address_size = mode;
    instruction->data_size = (int)length_bits;
    instruction->memory_bytes = rm_argument.kind == X86ASM_ARG_MEMORY ? (int)(length_bits / 8) : 0;
    instruction->zeroing = (p2 & 0x80) != 0;
    instruction->broadcast = (p2 & 0x10) != 0 && rm_argument.kind == X86ASM_ARG_MEMORY;
    instruction->prefixes[0] = 0x62;
    instruction->prefixes[1] = p0;
    instruction->prefixes[2] = p1;
    instruction->prefixes[3] = p2;
    instruction->encoded_opcode = ((uint32_t)0x62 << 24) | ((uint32_t)p0 << 16) | ((uint32_t)p1 << 8) | opcode;
    if (opcode == 0x10 || opcode == 0x11) {
        if (opcode == 0x11) { instruction->arguments[0] = rm_argument; set_vector_length(&instruction->arguments[1], length_bits, reg); }
        else { set_vector_length(&instruction->arguments[0], length_bits, reg); instruction->arguments[1] = rm_argument; }
    } else {
        set_vector_length(&instruction->arguments[0], length_bits, reg);
        set_vector_length(&instruction->arguments[1], length_bits, vvvv);
        instruction->arguments[2] = rm_argument;
    }
    return instruction->length = (int)pos, X86ASM_OK;
}

static x86asm_error decode_vex(const uint8_t *bytes, size_t length, size_t pos,
                               int mode, x86asm_instruction *instruction)
{
    uint8_t prefix = bytes[pos++];
    uint8_t p = 0, l = 0, w = 0, map = 0, vvvv;
    bool r = true, x = true, b = true;

    if (prefix == 0xC5) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        uint8_t b1 = bytes[pos++];
        r = (b1 & 0x80) != 0;
        vvvv = (uint8_t)((~b1 >> 3) & 0x0F);
        l = (b1 >> 2) & 1;
        p = b1 & 3;
        map = 1;
    } else if (prefix == 0xC4) {
        if (pos + 1 >= length) return X86ASM_ERR_TRUNCATED;
        uint8_t b1 = bytes[pos++];
        uint8_t b2 = bytes[pos++];
        r = (b1 & 0x80) != 0;
        x = (b1 & 0x40) != 0;
        b = (b1 & 0x20) != 0;
        map = b1 & 0x1F;
        w = (b2 >> 7) & 1;
        vvvv = (uint8_t)((~b2 >> 3) & 0x0F);
        l = (b2 >> 2) & 1;
        p = b2 & 3;
    } else {
        return X86ASM_ERR_UNRECOGNIZED;
    }

    if ((map != 1 && map != 2 && map != 3) || pos >= length) return X86ASM_ERR_UNRECOGNIZED;
    uint8_t opcode = bytes[pos++];
    instruction->encoded_opcode = ((uint32_t)prefix << 16) | opcode;
    instruction->mode = mode;
    instruction->address_size = mode;
    instruction->data_size = l ? 256 : 128;
    instruction->prefixes[0] = prefix;
    if (prefix == 0xC4) {
        instruction->prefixes[1] = bytes[pos - 3];
        instruction->prefixes[2] = bytes[pos - 2];
    } else {
        instruction->prefixes[1] = bytes[pos - 2];
    }

    if (opcode == 0x77 && p == 0 && l == 0 && w == 0) {
        instruction->opcode = X86ASM_OP_VZEROUPPER;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    x86asm_opcode operation = X86ASM_OP_INVALID;
    bool load_store = false;
    bool scalar_move = false;
    bool scalar_vector_move = false;
    bool xmm_quad_move = false;
    bool unary_vector = false;
    bool extend_vector = false;
    bool store = false;
    bool packed_shift = false;
    bool mask_extract = false;
    bool two_vector_test = false;
    bool variable_blend = false;
    bool scalar_insert = false;
    bool scalar_extract = false;
    if (opcode == 0x58 && p == 0) operation = X86ASM_OP_VADDPS;
    else if (opcode == 0x58 && p == 1) operation = X86ASM_OP_VADDPD;
    else if (opcode == 0x58 && p == 2) operation = X86ASM_OP_VADDSS;
    else if (opcode == 0x5C && p == 2) operation = X86ASM_OP_VSUBSS;
    else if (opcode == 0x58 && p == 3) operation = X86ASM_OP_VADDSD;
    else if (opcode == 0x5C && p == 3) operation = X86ASM_OP_VSUBSD;
    else if (opcode == 0x59 && p == 2) operation = X86ASM_OP_VMULSS;
    else if (opcode == 0x5E && p == 2) operation = X86ASM_OP_VDIVSS;
    else if (opcode == 0x5D && p == 2) operation = X86ASM_OP_VMINSS;
    else if (opcode == 0x5F && p == 2) operation = X86ASM_OP_VMAXSS;
    else if (opcode == 0x59 && p == 3) operation = X86ASM_OP_VMULSD;
    else if (opcode == 0x5E && p == 3) operation = X86ASM_OP_VDIVSD;
    else if (opcode == 0x5D && p == 3) operation = X86ASM_OP_VMINSD;
    else if (opcode == 0x5F && p == 3) operation = X86ASM_OP_VMAXSD;
    else if (opcode == 0x5C && p == 0) operation = X86ASM_OP_VSUBPS;
    else if (opcode == 0x5C && p == 1) operation = X86ASM_OP_VSUBPD;
    else if (opcode == 0x59 && p == 0) operation = X86ASM_OP_VMULPS;
    else if (opcode == 0x59 && p == 1) operation = X86ASM_OP_VMULPD;
    else if (opcode == 0x5E && p == 0) operation = X86ASM_OP_VDIVPS;
    else if (opcode == 0x5E && p == 1) operation = X86ASM_OP_VDIVPD;
    else if (opcode == 0x5D && p == 0) operation = X86ASM_OP_VMINPS;
    else if (opcode == 0x5F && p == 0) operation = X86ASM_OP_VMAXPS;
    else if (opcode == 0x5D && p == 1) operation = X86ASM_OP_VMINPD;
    else if (opcode == 0x5F && p == 1) operation = X86ASM_OP_VMAXPD;
    else if (opcode == 0xC2 && p == 0) operation = X86ASM_OP_VCMPPS;
    else if (opcode == 0xC2 && p == 1) operation = X86ASM_OP_VCMPPD;
    else if (opcode == 0x57 && p == 0) operation = X86ASM_OP_VXORPS;
    else if (opcode == 0x50 && p == 0 && map == 1) { operation = X86ASM_OP_VMOVMSKPS; mask_extract = true; }
    else if (opcode == 0x50 && p == 1 && map == 1) { operation = X86ASM_OP_VMOVMSKPD; mask_extract = true; }
    else if (opcode == 0xD7 && p == 1 && map == 1) { operation = X86ASM_OP_VPMOVMSKB; mask_extract = true; }
    else if (opcode == 0x17 && p == 1 && map == 2) { operation = X86ASM_OP_VPTEST; two_vector_test = true; }
    else if (opcode == 0xFC && p == 1 && map == 1) operation = X86ASM_OP_VPADDB;
    else if (opcode == 0xFD && p == 1 && map == 1) operation = X86ASM_OP_VPADDW;
    else if (opcode == 0xFE && p == 1 && map == 1) operation = X86ASM_OP_VPADDD;
    else if (opcode == 0xF8 && p == 1 && map == 1) operation = X86ASM_OP_VPSUBB;
    else if (opcode == 0xF9 && p == 1 && map == 1) operation = X86ASM_OP_VPSUBW;
    else if (opcode == 0xFA && p == 1 && map == 1) operation = X86ASM_OP_VPSUBD;
    else if (opcode == 0xD4 && p == 1 && map == 1) operation = X86ASM_OP_VPADDQ;
    else if (opcode == 0xFB && p == 1 && map == 1) operation = X86ASM_OP_VPSUBQ;
    else if (opcode == 0x70 && p == 1 && map == 1) operation = X86ASM_OP_VPSHUFD;
    else if (opcode == 0x70 && p == 2 && map == 1) operation = X86ASM_OP_VPSHUFLW;
    else if (opcode == 0x70 && p == 3 && map == 1) operation = X86ASM_OP_VPSHUFHW;
    else if (opcode == 0x0C && p == 1 && map == 3) operation = X86ASM_OP_VBLENDPS;
    else if (opcode == 0x0D && p == 1 && map == 3) operation = X86ASM_OP_VBLENDPD;
    else if (opcode == 0x02 && p == 1 && map == 3) operation = X86ASM_OP_VPBLENDD;
    else if (opcode == 0x0E && p == 1 && map == 3) operation = X86ASM_OP_VPBLENDW;
    else if (opcode == 0x0F && p == 1 && map == 3) operation = X86ASM_OP_VPALIGNR;
    else if (opcode == 0x14 && p == 1 && map == 3) { operation = X86ASM_OP_VPEXTRB; scalar_extract = true; }
    else if (opcode == 0x16 && p == 1 && map == 3) { operation = w != 0 ? X86ASM_OP_VPEXTRQ : X86ASM_OP_VPEXTRD; scalar_extract = true; }
    else if (opcode == 0x20 && p == 1 && map == 3) { operation = X86ASM_OP_VPINSRB; scalar_insert = true; }
    else if (opcode == 0x22 && p == 1 && map == 3) { operation = w != 0 ? X86ASM_OP_VPINSRQ : X86ASM_OP_VPINSRD; scalar_insert = true; }
    else if (opcode == 0xC4 && p == 1 && map == 1) { operation = X86ASM_OP_VPINSRW; scalar_insert = true; }
    else if (opcode == 0xC5 && p == 1 && map == 1) { operation = X86ASM_OP_VPEXTRW; scalar_extract = true; }
    else if (opcode == 0x74 && p == 1 && map == 1) operation = X86ASM_OP_VPCMPEQB;
    else if (opcode == 0x75 && p == 1 && map == 1) operation = X86ASM_OP_VPCMPEQW;
    else if (opcode == 0x76 && p == 1 && map == 1) operation = X86ASM_OP_VPCMPEQD;
    else if (opcode == 0x64 && p == 1 && map == 1) operation = X86ASM_OP_VPCMPGTB;
    else if (opcode == 0x65 && p == 1 && map == 1) operation = X86ASM_OP_VPCMPGTW;
    else if (opcode == 0x66 && p == 1 && map == 1) operation = X86ASM_OP_VPCMPGTD;
    else if (opcode == 0xD5 && p == 1 && map == 1) operation = X86ASM_OP_VPMULLW;
    else if (opcode == 0xE4 && p == 1 && map == 1) operation = X86ASM_OP_VPMULHUW;
    else if (opcode == 0xE5 && p == 1 && map == 1) operation = X86ASM_OP_VPMULHW;
    else if (opcode == 0xF4 && p == 1 && map == 1) operation = X86ASM_OP_VPMULUDQ;
    else if (opcode == 0xDC && p == 1 && map == 1) operation = X86ASM_OP_VPADDUSB;
    else if (opcode == 0xDD && p == 1 && map == 1) operation = X86ASM_OP_VPADDUSW;
    else if (opcode == 0xEC && p == 1 && map == 1) operation = X86ASM_OP_VPADDSB;
    else if (opcode == 0xED && p == 1 && map == 1) operation = X86ASM_OP_VPADDSW;
    else if (opcode == 0xD8 && p == 1 && map == 1) operation = X86ASM_OP_VPSUBUSB;
    else if (opcode == 0xD9 && p == 1 && map == 1) operation = X86ASM_OP_VPSUBUSW;
    else if (opcode == 0xE8 && p == 1 && map == 1) operation = X86ASM_OP_VPSUBSB;
    else if (opcode == 0xE9 && p == 1 && map == 1) operation = X86ASM_OP_VPSUBSW;
    else if (opcode == 0xDA && p == 1 && map == 1) operation = X86ASM_OP_VPMINUB;
    else if (opcode == 0xDE && p == 1 && map == 1) operation = X86ASM_OP_VPMAXUB;
    else if (opcode == 0xEA && p == 1 && map == 1) operation = X86ASM_OP_VPMINSW;
    else if (opcode == 0xEE && p == 1 && map == 1) operation = X86ASM_OP_VPMAXSW;
    else if (opcode == 0xE0 && p == 1 && map == 1) operation = X86ASM_OP_VPAVGB;
    else if (opcode == 0xE3 && p == 1 && map == 1) operation = X86ASM_OP_VPAVGW;
    else if (opcode == 0xF6 && p == 1 && map == 1) operation = X86ASM_OP_VPSADBW;
    else if (opcode == 0x60 && p == 1 && map == 1) operation = X86ASM_OP_VPUNPCKLBW;
    else if (opcode == 0x61 && p == 1 && map == 1) operation = X86ASM_OP_VPUNPCKLWD;
    else if (opcode == 0x62 && p == 1 && map == 1) operation = X86ASM_OP_VPUNPCKLDQ;
    else if (opcode == 0x63 && p == 1 && map == 1) operation = X86ASM_OP_VPACKSSWB;
    else if (opcode == 0x67 && p == 1 && map == 1) operation = X86ASM_OP_VPACKUSWB;
    else if (opcode == 0x68 && p == 1 && map == 1) operation = X86ASM_OP_VPUNPCKHBW;
    else if (opcode == 0x69 && p == 1 && map == 1) operation = X86ASM_OP_VPUNPCKHWD;
    else if (opcode == 0x6A && p == 1 && map == 1) operation = X86ASM_OP_VPUNPCKHDQ;
    else if (opcode == 0x6B && p == 1 && map == 1) operation = X86ASM_OP_VPACKSSDW;
    else if ((opcode == 0x71 || opcode == 0x72 || opcode == 0x73) && p == 1 && map == 1) {
        operation = X86ASM_OP_VPSLLW;
        packed_shift = true;
    }
    else if (opcode == 0x1C && p == 1 && map == 2) { operation = X86ASM_OP_VPABSB; unary_vector = true; }
    else if (opcode == 0x1D && p == 1 && map == 2) { operation = X86ASM_OP_VPABSW; unary_vector = true; }
    else if (opcode == 0x1E && p == 1 && map == 2) { operation = X86ASM_OP_VPABSD; unary_vector = true; }
    else if (opcode == 0x01 && p == 1 && map == 2) operation = X86ASM_OP_VPHADDW;
    else if (opcode == 0x02 && p == 1 && map == 2) operation = X86ASM_OP_VPHADDD;
    else if (opcode == 0x03 && p == 1 && map == 2) operation = X86ASM_OP_VPHADDSW;
    else if (opcode == 0x04 && p == 1 && map == 2) operation = X86ASM_OP_VPMADDUBSW;
    else if (opcode == 0x28 && p == 1 && map == 2) operation = X86ASM_OP_VPMULDQ;
    else if (opcode == 0x2A && p == 1 && map == 2) { operation = X86ASM_OP_VMOVNTDQA; unary_vector = true; }
    else if (opcode == 0xF0 && p == 2 && map == 1) { operation = X86ASM_OP_VLDDQU; unary_vector = true; }
    else if (opcode == 0xE7 && p == 1 && map == 1) { operation = X86ASM_OP_VMOVNTDQ; load_store = true; store = true; }
    else if (opcode == 0x40 && p == 1 && map == 2) operation = X86ASM_OP_VPMULLD;
    else if (opcode == 0x4A && p == 1 && map == 3) { operation = X86ASM_OP_VBLENDVPS; variable_blend = true; }
    else if (opcode == 0x4B && p == 1 && map == 3) { operation = X86ASM_OP_VBLENDVPD; variable_blend = true; }
    else if (opcode == 0x4C && p == 1 && map == 3) { operation = X86ASM_OP_VPBLENDVB; variable_blend = true; }
    else if (opcode == 0x41 && p == 1 && map == 2) { operation = X86ASM_OP_VPHMINPOSUW; unary_vector = true; }
    else if (opcode == 0x20 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVSXBW; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x21 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVSXBD; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x22 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVSXBQ; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x23 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVSXWD; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x24 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVSXWQ; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x25 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVSXDQ; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x30 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVZXBW; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x31 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVZXBD; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x32 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVZXBQ; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x33 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVZXWD; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x34 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVZXWQ; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x35 && p == 1 && map == 2) { operation = X86ASM_OP_VPMOVZXDQ; unary_vector = true; extend_vector = true; }
    else if (opcode == 0x05 && p == 1 && map == 2) operation = X86ASM_OP_VPHSUBW;
    else if (opcode == 0x06 && p == 1 && map == 2) operation = X86ASM_OP_VPHSUBD;
    else if (opcode == 0x07 && p == 1 && map == 2) operation = X86ASM_OP_VPHSUBSW;
    else if (opcode == 0x08 && p == 1 && map == 2) operation = X86ASM_OP_VPSIGNB;
    else if (opcode == 0x09 && p == 1 && map == 2) operation = X86ASM_OP_VPSIGNW;
    else if (opcode == 0x0A && p == 1 && map == 2) operation = X86ASM_OP_VPSIGND;
    else if (opcode == 0xF5 && p == 1 && map == 1) operation = X86ASM_OP_VPMADDWD;
    else if (opcode == 0x00 && p == 1 && map == 2) operation = X86ASM_OP_VPSHUFB;
    else if (opcode == 0x29 && p == 1 && map == 2) operation = X86ASM_OP_VPCMPEQQ;
    else if (opcode == 0x37 && p == 1 && map == 2 && w != 0) operation = X86ASM_OP_VPCMPGTQ;
    else if (opcode == 0x38 && p == 1 && map == 2) operation = X86ASM_OP_VPMINSB;
    else if (opcode == 0x39 && p == 1 && map == 2) operation = X86ASM_OP_VPMINSD;
    else if (opcode == 0x3A && p == 1 && map == 2) operation = X86ASM_OP_VPMINUW;
    else if (opcode == 0x3B && p == 1 && map == 2) operation = X86ASM_OP_VPMINUD;
    else if (opcode == 0x3C && p == 1 && map == 2) operation = X86ASM_OP_VPMAXSB;
    else if (opcode == 0x3D && p == 1 && map == 2) operation = X86ASM_OP_VPMAXSD;
    else if (opcode == 0x3E && p == 1 && map == 2) operation = X86ASM_OP_VPMAXUW;
    else if (opcode == 0x3F && p == 1 && map == 2) operation = X86ASM_OP_VPMAXUD;
    else if (opcode == 0xDB && p == 1 && map == 2) operation = X86ASM_OP_VAND;
    else if (opcode == 0xDF && p == 1 && map == 2) operation = X86ASM_OP_VANDN;
    else if (opcode == 0xEB && p == 1 && map == 2) operation = X86ASM_OP_VOR;
    else if (opcode == 0xEF && p == 1 && map == 2) operation = X86ASM_OP_VXOR;
    else if (opcode == 0x47 && p == 1 && map == 2) operation = w ? X86ASM_OP_VPSLLVQ : X86ASM_OP_VPSLLVD;
    else if (opcode == 0x45 && p == 1 && map == 2) operation = w ? X86ASM_OP_VPSRLVQ : X86ASM_OP_VPSRLVD;
    else if (opcode == 0x46 && p == 1 && map == 2 && w == 0) operation = X86ASM_OP_VPSRAVD;
    else if (opcode == 0x10 && (p == 2 || p == 3)) {
        operation = p == 2 ? X86ASM_OP_VMOVSS : X86ASM_OP_VMOVSD;
        load_store = true; scalar_move = true;
    } else if (opcode == 0x11 && (p == 2 || p == 3)) {
        operation = p == 2 ? X86ASM_OP_VMOVSS : X86ASM_OP_VMOVSD;
        load_store = true; scalar_move = true; store = true;
    } else if ((opcode == 0x6E || opcode == 0x7E) && p == 1 && map == 1) {
        operation = w != 0 ? X86ASM_OP_VMOVQ : X86ASM_OP_VMOVD;
        scalar_vector_move = true;
        store = opcode == 0x7E;
    } else if (((opcode == 0x7E && p == 3) || (opcode == 0xD6 && p == 1)) && map == 1) {
        operation = X86ASM_OP_VMOVQ;
        xmm_quad_move = true;
        store = opcode == 0xD6;
    } else if ((opcode == 0x6F || opcode == 0x7F) && (p == 1 || p == 2) && map == 1) {
        operation = p == 1 ? X86ASM_OP_VMOVDQA : X86ASM_OP_VMOVDQU;
        load_store = true;
        store = opcode == 0x7F;
    } else if (opcode == 0x10 && (p == 0 || p == 1)) {
        operation = p == 0 ? X86ASM_OP_VMOVUPS : X86ASM_OP_VMOVUPD;
        load_store = true;
    } else if (opcode == 0x11 && (p == 0 || p == 1)) {
        operation = p == 0 ? X86ASM_OP_VMOVUPS : X86ASM_OP_VMOVUPD;
        load_store = true;
        store = true;
    } else {
        return X86ASM_ERR_UNRECOGNIZED;
    }

    if ((scalar_move || scalar_vector_move || xmm_quad_move) && l != 0) return X86ASM_ERR_UNRECOGNIZED;
    if (scalar_vector_move && w != 0 && mode != 64) return X86ASM_ERR_UNRECOGNIZED;
    if (xmm_quad_move && w != 0) return X86ASM_ERR_UNRECOGNIZED;
    if ((scalar_insert || scalar_extract) && l != 0) return X86ASM_ERR_UNRECOGNIZED;
    if (scalar_extract && vvvv != 15u) return X86ASM_ERR_UNRECOGNIZED;
    if ((variable_blend || operation == X86ASM_OP_VPBLENDD) && w != 0) return X86ASM_ERR_UNRECOGNIZED;
    if (operation == X86ASM_OP_VPHMINPOSUW && l != 0) return X86ASM_ERR_UNRECOGNIZED;
    if (pos >= length) return X86ASM_ERR_TRUNCATED;
    uint8_t modrm = bytes[pos++];
    unsigned mod = modrm >> 6;
    unsigned reg = ((modrm >> 3) & 7) | (r ? 0 : 8);
    unsigned rm = (modrm & 7) | (b ? 0 : 8);
    x86asm_argument rm_argument = { X86ASM_ARG_NONE, { 0 } };
    if (two_vector_test && vvvv != 15u) return X86ASM_ERR_UNRECOGNIZED;
    if (mask_extract && (mod != 3u || vvvv != 15u)) return X86ASM_ERR_UNRECOGNIZED;
    if (scalar_move && mod != 3u && vvvv != 15u) return X86ASM_ERR_UNRECOGNIZED;
    if ((scalar_vector_move || xmm_quad_move) && vvvv != 15u) return X86ASM_ERR_UNRECOGNIZED;
    if (unary_vector && vvvv != 15u) return X86ASM_ERR_UNRECOGNIZED;
    if ((operation == X86ASM_OP_VMOVNTDQA || operation == X86ASM_OP_VLDDQU || operation == X86ASM_OP_VMOVNTDQ) && mod == 3u) return X86ASM_ERR_UNRECOGNIZED;
    if ((operation == X86ASM_OP_VMOVDQA || operation == X86ASM_OP_VMOVDQU) && vvvv != 15u) return X86ASM_ERR_UNRECOGNIZED;
    if (operation == X86ASM_OP_VMOVNTDQ && vvvv != 15u) return X86ASM_ERR_UNRECOGNIZED;
    uint8_t immediate = 0;
    if (mod == 3u) {
        if (scalar_insert) {
            int source_width = operation == X86ASM_OP_VPINSRQ ? 64 : 32;
            set_register(&rm_argument, register_for_width(source_width, rm));
        } else if (scalar_extract) {
            int destination_width = operation == X86ASM_OP_VPEXTRQ ? 64 : (mode == 64 ? 64 : 32);
            set_register(&rm_argument, register_for_width(destination_width, rm));
        } else if (scalar_vector_move) {
            set_register(&rm_argument, register_for_width(w != 0 ? 64 : 32, rm));
        } else {
            set_vector(&rm_argument, l != 0, rm);
        }
    } else {
        x86asm_memory memory;
        x86asm_error error = read_vex_memory(bytes, length, &pos, mode, modrm, x, b, &memory);
        if (error != X86ASM_OK) return error;
        set_memory(&rm_argument, memory);
    }
    if (packed_shift) {
        unsigned group = (modrm >> 3) & 7u;
        if (opcode == 0x71 && group != 2u && group != 4u && group != 6u) return X86ASM_ERR_UNRECOGNIZED;
        if (opcode == 0x72 && group != 2u && group != 4u && group != 6u) return X86ASM_ERR_UNRECOGNIZED;
        if (opcode == 0x73 && group != 2u && group != 3u && group != 6u && group != 7u) return X86ASM_ERR_UNRECOGNIZED;
        if (opcode == 0x71) operation = group == 2u ? X86ASM_OP_VPSRLW : group == 4u ? X86ASM_OP_VPSRAW : X86ASM_OP_VPSLLW;
        else if (opcode == 0x72) operation = group == 2u ? X86ASM_OP_VPSRLD : group == 4u ? X86ASM_OP_VPSRAD : X86ASM_OP_VPSLLD;
        else if (group == 2u) operation = X86ASM_OP_VPSRLQ;
        else if (group == 3u) operation = X86ASM_OP_VPSRLDQ;
        else if (group == 6u) operation = X86ASM_OP_VPSLLQ;
        else operation = X86ASM_OP_VPSLLDQ;
    }
    if (packed_shift || operation == X86ASM_OP_VPSHUFD || operation == X86ASM_OP_VPSHUFLW || operation == X86ASM_OP_VPSHUFHW || operation == X86ASM_OP_VBLENDPS || operation == X86ASM_OP_VBLENDPD || operation == X86ASM_OP_VPBLENDD || operation == X86ASM_OP_VPBLENDW || operation == X86ASM_OP_VPALIGNR || operation == X86ASM_OP_VPINSRB || operation == X86ASM_OP_VPINSRW || operation == X86ASM_OP_VPINSRD || operation == X86ASM_OP_VPINSRQ || operation == X86ASM_OP_VPEXTRB || operation == X86ASM_OP_VPEXTRW || operation == X86ASM_OP_VPEXTRD || operation == X86ASM_OP_VPEXTRQ || operation == X86ASM_OP_VCMPPS || operation == X86ASM_OP_VCMPPD || variable_blend) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        immediate = bytes[pos++];
        if ((operation == X86ASM_OP_VCMPPS || operation == X86ASM_OP_VCMPPD) && (immediate & 0xE0u) != 0u) return X86ASM_ERR_UNRECOGNIZED;
    }

    instruction->opcode = operation;
    if (scalar_vector_move) instruction->data_size = w != 0 ? 64 : 32;
    if (xmm_quad_move) instruction->data_size = 64;
    if (mask_extract) {
        set_register(&instruction->arguments[0], register_for_width(32, reg));
        set_vector(&instruction->arguments[1], l != 0, rm);
    } else if (two_vector_test) {
        set_vector(&instruction->arguments[0], l != 0, reg);
        instruction->arguments[1] = rm_argument;
    } else if (unary_vector) {
        set_vector(&instruction->arguments[0], l != 0, reg);
        if (extend_vector && rm_argument.kind == X86ASM_ARG_REGISTER) set_vector(&instruction->arguments[1], false, rm);
        else instruction->arguments[1] = rm_argument;
    } else if (scalar_insert) {
        set_vector(&instruction->arguments[0], false, reg);
        set_vector(&instruction->arguments[1], false, vvvv);
        instruction->arguments[2] = rm_argument;
        set_immediate(&instruction->arguments[3], immediate);
    } else if (scalar_extract) {
        instruction->arguments[0] = rm_argument;
        set_vector(&instruction->arguments[1], false, reg);
        set_immediate(&instruction->arguments[2], immediate);
    } else if (scalar_vector_move) {
        if (store) {
            instruction->arguments[0] = rm_argument;
            set_vector(&instruction->arguments[1], false, reg);
        } else {
            set_vector(&instruction->arguments[0], false, reg);
            instruction->arguments[1] = rm_argument;
        }
    } else if (xmm_quad_move) {
        if (store) {
            instruction->arguments[0] = rm_argument;
            set_vector(&instruction->arguments[1], false, reg);
        } else {
            set_vector(&instruction->arguments[0], false, reg);
            instruction->arguments[1] = rm_argument;
        }
    } else if (scalar_move && mod == 3u) {
        if (store) {
            instruction->arguments[0] = rm_argument;
            set_vector(&instruction->arguments[1], false, vvvv);
            set_vector(&instruction->arguments[2], false, reg);
        } else {
            set_vector(&instruction->arguments[0], false, reg);
            set_vector(&instruction->arguments[1], false, vvvv);
            instruction->arguments[2] = rm_argument;
        }
    } else if (load_store) {
        if (store) {
            instruction->arguments[0] = rm_argument;
            set_vector(&instruction->arguments[1], l != 0, reg);
        } else {
            set_vector(&instruction->arguments[0], l != 0, reg);
            instruction->arguments[1] = rm_argument;
        }
    } else if (packed_shift) {
        set_vector(&instruction->arguments[0], l != 0, vvvv);
        instruction->arguments[1] = rm_argument;
        set_immediate(&instruction->arguments[2], immediate);
    } else if (operation == X86ASM_OP_VPSHUFD || operation == X86ASM_OP_VPSHUFLW || operation == X86ASM_OP_VPSHUFHW || operation == X86ASM_OP_VBLENDPS || operation == X86ASM_OP_VBLENDPD) {
        set_vector(&instruction->arguments[0], l != 0, reg);
        if (operation == X86ASM_OP_VPSHUFD || operation == X86ASM_OP_VPSHUFLW || operation == X86ASM_OP_VPSHUFHW) {
            instruction->arguments[1] = rm_argument;
            set_immediate(&instruction->arguments[2], immediate);
        } else {
            set_vector(&instruction->arguments[1], l != 0, vvvv);
            instruction->arguments[2] = rm_argument;
            set_immediate(&instruction->arguments[3], immediate);
        }
    } else if (operation == X86ASM_OP_VPBLENDD || operation == X86ASM_OP_VPBLENDW || operation == X86ASM_OP_VPALIGNR) {
        set_vector(&instruction->arguments[0], l != 0, reg);
        set_vector(&instruction->arguments[1], l != 0, vvvv);
        instruction->arguments[2] = rm_argument;
        set_immediate(&instruction->arguments[3], immediate);
    } else if (operation == X86ASM_OP_VCMPPS || operation == X86ASM_OP_VCMPPD) {
        set_vector(&instruction->arguments[0], l != 0, reg);
        set_vector(&instruction->arguments[1], l != 0, vvvv);
        instruction->arguments[2] = rm_argument;
        set_immediate(&instruction->arguments[3], immediate);
    } else if (variable_blend) {
        set_vector(&instruction->arguments[0], l != 0, reg);
        set_vector(&instruction->arguments[1], l != 0, vvvv);
        instruction->arguments[2] = rm_argument;
        set_vector(&instruction->arguments[3], l != 0, (unsigned)(immediate >> 4));
    } else {
        set_vector(&instruction->arguments[0], l != 0, reg);
        set_vector(&instruction->arguments[1], l != 0, vvvv);
        instruction->arguments[2] = rm_argument;
    }

    if (rm_argument.kind == X86ASM_ARG_MEMORY) {
        instruction->memory_bytes = l ? 32 : 16;
        if (scalar_vector_move) instruction->memory_bytes = w != 0 ? 8 : 4;
        if (xmm_quad_move) instruction->memory_bytes = 8;
        if (operation == X86ASM_OP_VADDSS || operation == X86ASM_OP_VSUBSS) instruction->memory_bytes = 4;
        if (operation == X86ASM_OP_VADDSD || operation == X86ASM_OP_VSUBSD || operation == X86ASM_OP_VMULSD || operation == X86ASM_OP_VDIVSD || operation == X86ASM_OP_VMINSD || operation == X86ASM_OP_VMAXSD) instruction->memory_bytes = 8;
        if (extend_vector) {
            unsigned opcode_bytes = opcode == 0x20 || opcode == 0x30 ? 8u : (opcode == 0x21 || opcode == 0x31 ? 4u : (opcode == 0x22 || opcode == 0x32 ? 2u : (opcode == 0x23 || opcode == 0x33 ? 8u : (opcode == 0x24 || opcode == 0x34 ? 4u : 8u))));
            instruction->memory_bytes = l ? (int)(opcode_bytes * 2u) : (int)opcode_bytes;
        }
        if (scalar_insert || scalar_extract) {
            instruction->memory_bytes = operation == X86ASM_OP_VPINSRB || operation == X86ASM_OP_VPEXTRB ? 1 : (operation == X86ASM_OP_VPINSRW || operation == X86ASM_OP_VPEXTRW ? 2 : (operation == X86ASM_OP_VPINSRQ || operation == X86ASM_OP_VPEXTRQ ? 8 : 4));
        }
        if (operation == X86ASM_OP_VMULSS || operation == X86ASM_OP_VDIVSS || operation == X86ASM_OP_VMINSS || operation == X86ASM_OP_VMAXSS || operation == X86ASM_OP_VMOVSS) instruction->memory_bytes = 4;
    }
    instruction->length = (int)pos;
    return X86ASM_OK;
}

x86asm_error x86asm_decode(const uint8_t *bytes, size_t length, int mode,
                           x86asm_instruction *instruction)
{
    size_t pos = 0;
    uint8_t rex = 0;
    uint8_t opcode_byte;
    int operand_width;
    bool operand_override = false;
    bool prefix_f2 = false;

    if (instruction == NULL || bytes == NULL) return X86ASM_ERR_UNRECOGNIZED;
    memset(instruction, 0, sizeof(*instruction));
    if (mode != 16 && mode != 32 && mode != 64) return X86ASM_ERR_INVALID_MODE;
    if (length == 0) return X86ASM_ERR_TRUNCATED;

    while (pos < length && pos < 14) {
        uint8_t prefix = bytes[pos];
        bool recognized = true;
        switch (prefix) {
        case 0xF0: case 0xF2: case 0xF3:
        case 0x2E: case 0x36: case 0x3E: case 0x26: case 0x64: case 0x65:
        case 0x66: case 0x67:
            instruction->prefixes[pos++] = (x86asm_prefix)prefix;
            if (prefix == 0x66) operand_override = true;
            if (prefix == 0xF2) prefix_f2 = true;
            break;
        default:
            recognized = false;
            break;
        }
        if (!recognized) break;
    }
    if (pos < length && bytes[pos] == 0x62) {
        return decode_evex(bytes, length, pos, mode, instruction);
    }
    if (pos < length && bytes[pos] == 0x8F && pos + 1 < length &&
        (bytes[pos + 1] & 0x1F) >= 8 && (bytes[pos + 1] & 0x1F) <= 10) {
        return decode_xop(bytes, length, pos, mode, instruction);
    }
    if (pos < length && (bytes[pos] == 0xC4 || bytes[pos] == 0xC5)) {
        return decode_vex(bytes, length, pos, mode, instruction);
    }

    if (mode == 64 && pos < length && (bytes[pos] & 0xF0) == 0x40) {
        rex = bytes[pos];
        instruction->prefixes[pos++] = rex;
    }
    if (pos >= length) return X86ASM_ERR_TRUNCATED;

    operand_width = mode == 16 ? 16 : 32;
    if (mode == 64 && (rex & 8)) operand_width = 64;
    if (operand_override) operand_width = mode == 16 ? 32 : 16;
    instruction->mode = mode;
    instruction->address_size = operand_override && mode == 64 ? 32 : mode;
    instruction->data_size = operand_width;
    opcode_byte = bytes[pos++];
    instruction->encoded_opcode = opcode_byte;

    if (mode != 64 && opcode_byte >= 0x40 && opcode_byte <= 0x47) {
        instruction->opcode = X86ASM_OP_INC;
        set_register(&instruction->arguments[0], register_for_width(operand_width, opcode_byte & 7));
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (mode != 64 && opcode_byte >= 0x48 && opcode_byte <= 0x4F) {
        instruction->opcode = X86ASM_OP_DEC;
        set_register(&instruction->arguments[0], register_for_width(operand_width, opcode_byte & 7));
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (opcode_byte >= 0x91 && opcode_byte <= 0x97) {
        instruction->opcode = X86ASM_OP_XCHG;
        set_register(&instruction->arguments[0], register_for_width(mode == 64 ? 64 : operand_width, 0));
        set_register(&instruction->arguments[1], register_for_encoding(mode == 64 ? 64 : operand_width, (opcode_byte & 7) | ((rex & 1) ? 8 : 0), rex));
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (opcode_byte >= 0xA0 && opcode_byte <= 0xA3) {
        unsigned width = (opcode_byte == 0xA0 || opcode_byte == 0xA2) ? 8u : (unsigned)operand_width;
        size_t address_bytes = instruction->address_size == 16 ? (size_t)2 : (instruction->address_size == 32 ? (size_t)4 : (size_t)8);
        if (pos + address_bytes > length) return X86ASM_ERR_TRUNCATED;
        uint64_t address = address_bytes == 2 ? read_u16(bytes + pos) : (address_bytes == 4 ? read_u32(bytes + pos) : read_u64(bytes + pos));
        pos += address_bytes;
        x86asm_memory memory = { X86ASM_REG_NONE, X86ASM_REG_NONE, X86ASM_REG_NONE, 0, (int64_t)address };
        instruction->opcode = X86ASM_OP_MOV;
        instruction->memory_bytes = (int)(width / 8);
        if (opcode_byte == 0xA0 || opcode_byte == 0xA1) {
            set_register(&instruction->arguments[0], opcode_byte == 0xA0 ? X86ASM_REG_AL : register_for_width(operand_width, 0));
            set_memory(&instruction->arguments[1], memory);
        } else {
            set_memory(&instruction->arguments[0], memory);
            set_register(&instruction->arguments[1], opcode_byte == 0xA2 ? X86ASM_REG_AL : register_for_width(operand_width, 0));
        }
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0xA8 || opcode_byte == 0xA9) {
        unsigned width = opcode_byte == 0xA8 ? 8u : (unsigned)operand_width;
        size_t immediate_size = width == 8 ? (size_t)1 : (width == 16 ? (size_t)2 : (size_t)4);
        if (pos + immediate_size > length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = X86ASM_OP_TEST;
        set_register(&instruction->arguments[0], width == 8 ? X86ASM_REG_AL : register_for_width(operand_width, 0));
        if (immediate_size == 1) set_immediate(&instruction->arguments[1], (int8_t)bytes[pos]);
        else if (immediate_size == 2) set_immediate(&instruction->arguments[1], (int16_t)read_u16(bytes + pos));
        else set_immediate(&instruction->arguments[1], (int32_t)read_u32(bytes + pos));
        pos += immediate_size;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if ((opcode_byte & 0xF8) == 0xB0) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = X86ASM_OP_MOV;
        set_register(&instruction->arguments[0], register_for_encoding(8, (opcode_byte & 7) | ((rex & 1) ? 8 : 0), rex));
        set_immediate(&instruction->arguments[1], (int8_t)bytes[pos++]);
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    switch (opcode_byte) {
    case 0x04: case 0x05: case 0x0C: case 0x0D:
    case 0x14: case 0x15: case 0x1C: case 0x1D:
    case 0x24: case 0x25: case 0x2C: case 0x2D:
    case 0x34: case 0x35: case 0x3C: case 0x3D: {
        bool byte_form = (opcode_byte & 1) == 0;
        size_t immediate_size = byte_form ? (size_t)1 : (operand_width == 64 ? (size_t)4 : (size_t)(operand_width / 8));
        if (pos + immediate_size > length) return X86ASM_ERR_TRUNCATED;
        if (opcode_byte >= 0x04 && opcode_byte <= 0x05) instruction->opcode = X86ASM_OP_ADD;
        else if (opcode_byte >= 0x0C && opcode_byte <= 0x0D) instruction->opcode = X86ASM_OP_OR;
        else if (opcode_byte >= 0x14 && opcode_byte <= 0x15) instruction->opcode = X86ASM_OP_ADC;
        else if (opcode_byte >= 0x1C && opcode_byte <= 0x1D) instruction->opcode = X86ASM_OP_SBB;
        else if (opcode_byte >= 0x24 && opcode_byte <= 0x25) instruction->opcode = X86ASM_OP_AND;
        else if (opcode_byte >= 0x2C && opcode_byte <= 0x2D) instruction->opcode = X86ASM_OP_SUB;
        else if (opcode_byte >= 0x34 && opcode_byte <= 0x35) instruction->opcode = X86ASM_OP_XOR;
        else instruction->opcode = X86ASM_OP_CMP;
        set_register(&instruction->arguments[0], register_for_width(operand_width, 0));
        if (byte_form) set_register(&instruction->arguments[0], X86ASM_REG_AL);
        if (immediate_size == 1) set_immediate(&instruction->arguments[1], (int8_t)bytes[pos]);
        else if (immediate_size == 2) set_immediate(&instruction->arguments[1], (int16_t)read_u16(bytes + pos));
        else set_immediate(&instruction->arguments[1], (int32_t)read_u32(bytes + pos));
        pos += immediate_size;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    default:
        break;
    }

    if (opcode_byte == 0x90) {
        instruction->opcode = (instruction->prefixes[0] & 0xFF) == X86ASM_PREFIX_REP
                                  ? X86ASM_OP_NOP : X86ASM_OP_NOP;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0xF5 || opcode_byte == 0xF8 || opcode_byte == 0xF9 ||
        opcode_byte == 0xFC || opcode_byte == 0xFD || opcode_byte == 0x9E || opcode_byte == 0x9F) {
        if (opcode_byte == 0xF5) instruction->opcode = X86ASM_OP_CMC;
        else if (opcode_byte == 0xF8) instruction->opcode = X86ASM_OP_CLC;
        else if (opcode_byte == 0xF9) instruction->opcode = X86ASM_OP_STC;
        else if (opcode_byte == 0xFC) instruction->opcode = X86ASM_OP_CLD;
        else if (opcode_byte == 0xFD) instruction->opcode = X86ASM_OP_STD;
        else if (opcode_byte == 0x9E) instruction->opcode = X86ASM_OP_SAHF;
        else instruction->opcode = X86ASM_OP_LAHF;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0x9C || opcode_byte == 0x9D || opcode_byte == 0xFA ||
        opcode_byte == 0xFB || opcode_byte == 0xF4) {
        if (opcode_byte == 0x9C) instruction->opcode = X86ASM_OP_PUSHF;
        else if (opcode_byte == 0x9D) instruction->opcode = X86ASM_OP_POPF;
        else if (opcode_byte == 0xFA) instruction->opcode = X86ASM_OP_CLI;
        else if (opcode_byte == 0xFB) instruction->opcode = X86ASM_OP_STI;
        else instruction->opcode = X86ASM_OP_HLT;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0xA4 || opcode_byte == 0xA5 || opcode_byte == 0xAA ||
        opcode_byte == 0xAB || opcode_byte == 0xAC || opcode_byte == 0xAD ||
        opcode_byte == 0xA6 ||
        opcode_byte == 0xA7 || opcode_byte == 0xAE || opcode_byte == 0xAF) {
        bool byte_form = opcode_byte == 0xA4 || opcode_byte == 0xA6 ||
                          opcode_byte == 0xAA || opcode_byte == 0xAC || opcode_byte == 0xAE;
        bool word_form = opcode_byte == 0xA5 || opcode_byte == 0xAB || opcode_byte == 0xAD ||
                         opcode_byte == 0xA7 || opcode_byte == 0xAF;
        unsigned element_width = byte_form ? 8u : (unsigned)operand_width;
        if (word_form && element_width == 16) {
            if (opcode_byte == 0xA5) instruction->opcode = X86ASM_OP_MOVSW;
            else if (opcode_byte == 0xAB) instruction->opcode = X86ASM_OP_STOSW;
            else if (opcode_byte == 0xAD) instruction->opcode = X86ASM_OP_LODSW;
            else if (opcode_byte == 0xA7) instruction->opcode = X86ASM_OP_CMPSW;
            else instruction->opcode = X86ASM_OP_SCASW;
        } else if (word_form && element_width == 64) {
            if (opcode_byte == 0xA5) instruction->opcode = X86ASM_OP_MOVSQ;
            else if (opcode_byte == 0xAB) instruction->opcode = X86ASM_OP_STOSQ;
            else if (opcode_byte == 0xAD) instruction->opcode = X86ASM_OP_LODSQ;
            else if (opcode_byte == 0xA7) instruction->opcode = X86ASM_OP_CMPSQ;
            else instruction->opcode = X86ASM_OP_SCASQ;
        } else if (word_form) {
            if (opcode_byte == 0xA5) instruction->opcode = X86ASM_OP_MOVSD;
            else if (opcode_byte == 0xAB) instruction->opcode = X86ASM_OP_STOSD;
            else if (opcode_byte == 0xAD) instruction->opcode = X86ASM_OP_LODSD;
            else if (opcode_byte == 0xA7) instruction->opcode = X86ASM_OP_CMPSD;
            else instruction->opcode = X86ASM_OP_SCASD;
        } else if (opcode_byte == 0xA4) instruction->opcode = X86ASM_OP_MOVSB;
        else if (opcode_byte == 0xAA) instruction->opcode = X86ASM_OP_STOSB;
        else if (opcode_byte == 0xAC) instruction->opcode = X86ASM_OP_LODSB;
        else if (opcode_byte == 0xA6) instruction->opcode = X86ASM_OP_CMPSB;
        else if (opcode_byte == 0xAE) instruction->opcode = X86ASM_OP_SCASB;
        instruction->data_size = (int)element_width;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0xC9) {
        instruction->opcode = X86ASM_OP_LEAVE;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0xC8) {
        if (pos + 3 > length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = X86ASM_OP_ENTER;
        set_immediate(&instruction->arguments[0], read_u16(bytes + pos));
        set_immediate(&instruction->arguments[1], bytes[pos + 2]);
        pos += 3;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (opcode_byte == 0xCC) {
        instruction->opcode = X86ASM_OP_INT;
        set_immediate(&instruction->arguments[0], 3);
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0xCD) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = X86ASM_OP_INT;
        set_immediate(&instruction->arguments[0], bytes[pos++]);
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0xC3 || opcode_byte == 0xCB) {
        instruction->opcode = X86ASM_OP_RET;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0xC2 || opcode_byte == 0xCA) {
        if (pos + 2 > length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = X86ASM_OP_RET;
        set_immediate(&instruction->arguments[0], read_u16(bytes + pos));
        pos += 2;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if ((opcode_byte & 0xF8) == 0x50) {
        instruction->opcode = X86ASM_OP_PUSH;
        set_register(&instruction->arguments[0], register_for_encoding(mode == 64 ? 64 : operand_width, (opcode_byte & 7) | ((rex & 1) ? 8 : 0), rex));
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if ((opcode_byte & 0xF8) == 0x58) {
        instruction->opcode = X86ASM_OP_POP;
        set_register(&instruction->arguments[0], register_for_encoding(mode == 64 ? 64 : operand_width, (opcode_byte & 7) | ((rex & 1) ? 8 : 0), rex));
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if ((opcode_byte & 0xF8) == 0xB8) {
        size_t immediate_size = operand_width == 64 ? (size_t)8 : (size_t)(operand_width / 8);
        if (pos + immediate_size > length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = X86ASM_OP_MOV;
        set_register(&instruction->arguments[0], register_for_encoding(operand_width, (opcode_byte & 7) | ((rex & 1) ? 8 : 0), rex));
        set_immediate(&instruction->arguments[1], immediate_size == 8 ? (int64_t)read_u64(bytes + pos) : (operand_width == 16 ? (int16_t)read_u16(bytes + pos) : (int32_t)read_u32(bytes + pos)));
        pos += immediate_size;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (opcode_byte >= 0xE0 && opcode_byte <= 0xE3) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        if (opcode_byte == 0xE0) instruction->opcode = X86ASM_OP_LOOPNE;
        else if (opcode_byte == 0xE1) instruction->opcode = X86ASM_OP_LOOPE;
        else if (opcode_byte == 0xE2) instruction->opcode = X86ASM_OP_LOOP;
        else instruction->opcode = X86ASM_OP_JCXZ;
        set_relative(&instruction->arguments[0], (int8_t)bytes[pos++]);
        instruction->pc_relative_bytes = 1;
        instruction->pc_relative_offset = (int)pos - 1;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0xEB || (opcode_byte >= 0x70 && opcode_byte <= 0x7F)) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = opcode_byte == 0xEB ? X86ASM_OP_JMP : condition_opcode(opcode_byte & 15);
        set_relative(&instruction->arguments[0], (int8_t)bytes[pos]);
        instruction->pc_relative_bytes = 1;
        instruction->pc_relative_offset = (int)pos;
        pos++;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }
    if (opcode_byte == 0xE8 || opcode_byte == 0xE9) {
        size_t displacement_size = mode == 16 ? 2 : 4;
        if (pos + displacement_size > length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = opcode_byte == 0xE8 ? X86ASM_OP_CALL : X86ASM_OP_JMP;
        set_relative(&instruction->arguments[0], displacement_size == 2 ? (int16_t)read_u16(bytes + pos) : (int32_t)read_u32(bytes + pos));
        instruction->pc_relative_bytes = (int)displacement_size;
        instruction->pc_relative_offset = (int)pos;
        pos += displacement_size;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (opcode_byte == 0x0F) {
        if (pos >= length) return X86ASM_ERR_TRUNCATED;
        uint8_t second = bytes[pos++];
        instruction->encoded_opcode = ((uint32_t)opcode_byte << 8) | second;
        if (second == 0x10 || second == 0x11 || second == 0x12 || second == 0x13 || second == 0x16 || second == 0x17 || second == 0x57 || second == 0x58 ||
            second == 0x50 || second == 0x59 || second == 0x5C || second == 0x5D || second == 0x5E || second == 0x5F || second == 0x70 || second == 0xC2 ||
            second == 0x6E || second == 0x7E || second == 0x6F || second == 0x7F || second == 0xD6 ||
            second == 0x71 || second == 0x72 || second == 0x73 ||
            second == 0x64 || second == 0x65 || second == 0x66 || second == 0x74 || second == 0x75 ||
            second == 0x76 || second == 0x60 || second == 0x61 || second == 0x62 || second == 0x63 ||
            second == 0x67 || second == 0x68 || second == 0x69 || second == 0x6A || second == 0x6B ||
            second == 0xD5 || second == 0xD7 || second == 0xE4 || second == 0xE5 ||
            second == 0xF4 || second == 0xDC || second == 0xDD || second == 0xEC || second == 0xED ||
            second == 0xD8 || second == 0xD9 || second == 0xE8 || second == 0xE9 ||
            second == 0xDA || second == 0xDE || second == 0xEA || second == 0xEE ||
            second == 0xE0 || second == 0xE3 || second == 0xF6 || second == 0xDB || second == 0xEB || second == 0xEF ||
            second == 0xF8 || second == 0xF9 ||
            second == 0xFA || second == 0xFC || second == 0xFD || second == 0xFE || second == 0xE7 || second == 0xC4 || second == 0xC5) {
            return decode_sse(bytes, length, pos, mode, rex, second, instruction);
        }
        if (second == 0x05 || second == 0x07 || second == 0x0B || second == 0x34 || second == 0x35) {
            if (second == 0x05) instruction->opcode = X86ASM_OP_SYSCALL;
            else if (second == 0x07) instruction->opcode = X86ASM_OP_SYSRET;
            else if (second == 0x0B) instruction->opcode = X86ASM_OP_UD2;
            else if (second == 0x34) instruction->opcode = X86ASM_OP_SYSENTER;
            else instruction->opcode = X86ASM_OP_SYSEXIT;
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xC7) {
            uint8_t modrm;
            x86asm_register ignored_register;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex,
                                             &modrm, &instruction->arguments[0], &ignored_register);
            if (error != X86ASM_OK) return error;
            if (((modrm >> 3) & 7u) != 1u || (modrm >> 6) == 3u) return X86ASM_ERR_UNRECOGNIZED;
            if ((rex & 8u) != 0) {
                instruction->opcode = X86ASM_OP_CMPXCHG16B;
                instruction->data_size = 128;
                instruction->memory_bytes = 16;
            } else {
                instruction->opcode = X86ASM_OP_CMPXCHG8B;
                instruction->data_size = 64;
                instruction->memory_bytes = 8;
            }
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second >= 0xC8 && second <= 0xCF) {
            instruction->opcode = X86ASM_OP_BSWAP;
            set_register(&instruction->arguments[0], register_for_width(mode == 64 ? 64 : operand_width, (second & 7) | ((rex & 1) != 0 ? 8u : 0u)));
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xBC || second == 0xBD) {
            x86asm_register reg;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size,
                                             operand_width, rex, &(uint8_t){0}, &instruction->arguments[1], &reg);
            if (error != X86ASM_OK) return error;
            instruction->opcode = second == 0xBC ? X86ASM_OP_BSF : X86ASM_OP_BSR;
            set_register(&instruction->arguments[0], reg);
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xA4 || second == 0xA5 || second == 0xAC || second == 0xAD) {
            uint8_t modrm;
            x86asm_register reg;
            bool immediate_count = second == 0xA4 || second == 0xAC;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size,
                                             operand_width, rex, &modrm, &instruction->arguments[0], &reg);
            if (error != X86ASM_OK) return error;
            instruction->opcode = second == 0xA4 || second == 0xA5 ? X86ASM_OP_SHLD : X86ASM_OP_SHRD;
            set_register(&instruction->arguments[1], reg);
            if (immediate_count) {
                if (pos >= length) return X86ASM_ERR_TRUNCATED;
                set_immediate(&instruction->arguments[2], bytes[pos++]);
            } else {
                set_register(&instruction->arguments[2], X86ASM_REG_CL);
            }
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xA3 || second == 0xAB || second == 0xB3 || second == 0xBB) {
            x86asm_register reg;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size,
                                             operand_width, rex, &(uint8_t){0}, &instruction->arguments[0], &reg);
            if (error != X86ASM_OK) return error;
            if (second == 0xA3) instruction->opcode = X86ASM_OP_BT;
            else if (second == 0xAB) instruction->opcode = X86ASM_OP_BTS;
            else if (second == 0xB3) instruction->opcode = X86ASM_OP_BTR;
            else instruction->opcode = X86ASM_OP_BTC;
            set_register(&instruction->arguments[1], reg);
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xBA) {
            uint8_t modrm;
            x86asm_register reg;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size,
                                             operand_width, rex, &modrm, &instruction->arguments[0], &reg);
            unsigned group = (modrm >> 3) & 7u;
            if (error != X86ASM_OK) return error;
            if (group < 4u || group > 7u || pos >= length) return X86ASM_ERR_UNRECOGNIZED;
            instruction->opcode = group == 4u ? X86ASM_OP_BT :
                                 (group == 5u ? X86ASM_OP_BTS :
                                  (group == 6u ? X86ASM_OP_BTR : X86ASM_OP_BTC));
            set_immediate(&instruction->arguments[1], bytes[pos++]);
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second >= 0x80 && second <= 0x8F) {
            size_t displacement_size = mode == 16 ? 2 : 4;
            if (pos + displacement_size > length) return X86ASM_ERR_TRUNCATED;
            instruction->opcode = condition_opcode(second & 15);
            set_relative(&instruction->arguments[0], displacement_size == 2 ? (int16_t)read_u16(bytes + pos) : (int32_t)read_u32(bytes + pos));
            instruction->pc_relative_bytes = (int)displacement_size;
            instruction->pc_relative_offset = (int)pos;
            pos += displacement_size;
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second >= 0x40 && second <= 0x4F) {
            x86asm_register reg;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, operand_width, rex, &(uint8_t){0}, &instruction->arguments[1], &reg);
            if (error != X86ASM_OK) return error;
            instruction->opcode = cmov_opcode(second & 15);
            set_register(&instruction->arguments[0], reg);
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second >= 0x90 && second <= 0x9F) {
            x86asm_register ignored_register;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 8, rex, &(uint8_t){0}, &instruction->arguments[0], &ignored_register);
            if (error != X86ASM_OK) return error;
            instruction->opcode = set_opcode(second & 15);
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0x1F) {
            x86asm_register ignored_register;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, operand_width, rex, &(uint8_t){0}, &instruction->arguments[0], &ignored_register);
            if (error != X86ASM_OK) return error;
            instruction->opcode = X86ASM_OP_NOP;
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xC0 || second == 0xC1) {
            x86asm_register reg;
            int width = second == 0xC0 ? 8 : operand_width;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, width, rex, &(uint8_t){0}, &instruction->arguments[0], &reg);
            if (error != X86ASM_OK) return error;
            instruction->opcode = X86ASM_OP_XADD;
            set_register(&instruction->arguments[1], reg);
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xB0 || second == 0xB1) {
            x86asm_register reg;
            int width = second == 0xB0 ? 8 : operand_width;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, width, rex, &(uint8_t){0}, &instruction->arguments[0], &reg);
            if (error != X86ASM_OK) return error;
            instruction->opcode = X86ASM_OP_CMPXCHG;
            set_register(&instruction->arguments[1], reg);
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xF5 && operand_override) {
            uint8_t modrm;
            x86asm_register ignored_register;
            x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
            if (error != X86ASM_OK) return error;
            unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
            unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
            if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
            instruction->opcode = X86ASM_OP_PMADDWD;
            instruction->data_size = 128;
            instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
            set_vector(&instruction->arguments[0], false, reg_index);
            instruction->arguments[1] = source;
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xAF) {
            x86asm_register reg;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, operand_width, rex, &(uint8_t){0}, &instruction->arguments[1], &reg);
            if (error != X86ASM_OK) return error;
            instruction->opcode = X86ASM_OP_IMUL;
            set_register(&instruction->arguments[0], reg);
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xB6 || second == 0xB7 || second == 0xBE || second == 0xBF) {
            x86asm_register reg;
            uint8_t modrm;
            int source_width = (second == 0xB6 || second == 0xBE) ? 8 : 16;
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, operand_width, rex, &modrm, &instruction->arguments[1], &reg);
            if (error != X86ASM_OK) return error;
            if ((modrm >> 6) == 3) {
                unsigned encoded_rm = (modrm & 7) | ((rex & 1) != 0 ? 8u : 0u);
                set_register(&instruction->arguments[1], register_for_encoding(source_width, encoded_rm, rex));
            }
            instruction->opcode = (second == 0xB6 || second == 0xB7) ? X86ASM_OP_MOVZX : X86ASM_OP_MOVSX;
            set_register(&instruction->arguments[0], reg);
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0x3A && pos < length && (bytes[pos] == 0x14 || bytes[pos] == 0x16 || bytes[pos] == 0x20 || bytes[pos] == 0x22)) {
            if (!operand_override) return X86ASM_ERR_UNRECOGNIZED;
            uint8_t third = bytes[pos++];
            uint8_t modrm;
            x86asm_register ignored_register;
            x86asm_argument operand = { X86ASM_ARG_NONE, { 0 } };
            x86asm_error error;
            if (pos >= length) return X86ASM_ERR_TRUNCATED;
            modrm = bytes[pos++];
            unsigned mod = modrm >> 6;
            unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
            unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
            bool insert = third == 0x20 || third == 0x22;
            bool byte_element = third == 0x14 || third == 0x20;
            bool qword_element = (third == 0x16 || third == 0x22) && (rex & 8u) != 0u;
            unsigned element_bytes = byte_element ? 1u : (qword_element ? 8u : (third == 0xC4 || third == 0xC5 ? 2u : 4u));
            if (mod == 3u) {
                int scalar_width = insert ? (qword_element ? 64 : 32) : (qword_element ? 64 : (mode == 64 ? 64 : 32));
                set_register(&operand, register_for_encoding(scalar_width, rm_index, rex));
            } else {
                x86asm_memory memory;
                error = read_vex_memory(bytes, length, &pos, mode, modrm, (rex & 2u) == 0, (rex & 1u) == 0, &memory);
                if (error != X86ASM_OK) return error;
                set_memory(&operand, memory);
            }
            if (pos >= length) return X86ASM_ERR_TRUNCATED;
            instruction->opcode = third == 0x14 ? X86ASM_OP_PEXTRB : (third == 0x16 ? (qword_element ? X86ASM_OP_PEXTRQ : X86ASM_OP_PEXTRD) : (third == 0x20 ? X86ASM_OP_PINSRB : (qword_element ? X86ASM_OP_PINSRQ : X86ASM_OP_PINSRD)));
            instruction->data_size = 128;
            instruction->memory_bytes = operand.kind == X86ASM_ARG_MEMORY ? (int)element_bytes : 0;
            set_vector(&instruction->arguments[insert ? 0 : 1], false, reg_index);
            if (insert) instruction->arguments[1] = operand;
            else instruction->arguments[0] = operand;
            set_immediate(&instruction->arguments[2], bytes[pos++]);
            instruction->length = (int)pos;
            (void)ignored_register;
            return X86ASM_OK;
        }
        if (second == 0x3A) {
            if (pos >= length || !operand_override) return X86ASM_ERR_UNRECOGNIZED;
            uint8_t third = bytes[pos++];
            if (third != 0x0C && third != 0x0D && third != 0x0E && third != 0x0F) return X86ASM_ERR_UNRECOGNIZED;
            uint8_t modrm;
            x86asm_register ignored_register;
            x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
            if (error != X86ASM_OK) return error;
            if (pos >= length) return X86ASM_ERR_TRUNCATED;
            unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
            unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
            if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
            instruction->opcode = third == 0x0C ? X86ASM_OP_BLENDPS : (third == 0x0D ? X86ASM_OP_BLENDPD : (third == 0x0E ? X86ASM_OP_PBLENDW : X86ASM_OP_PALIGNR));
            instruction->data_size = 128;
            instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
            set_vector(&instruction->arguments[0], false, reg_index);
            instruction->arguments[1] = source;
            set_immediate(&instruction->arguments[2], bytes[pos++]);
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0xF0 && prefix_f2) {
            uint8_t modrm;
            x86asm_register ignored_register;
            x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
            x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 128, rex, &modrm, &source, &ignored_register);
            if (error != X86ASM_OK) return error;
            if ((modrm >> 6) == 3u) return X86ASM_ERR_UNRECOGNIZED;
            unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
            instruction->opcode = X86ASM_OP_LDDQU;
            instruction->data_size = 128;
            instruction->memory_bytes = 16;
            set_vector(&instruction->arguments[0], false, reg_index);
            instruction->arguments[1] = source;
            instruction->length = (int)pos;
            return X86ASM_OK;
        }
        if (second == 0x38) {
            if (pos >= length) return X86ASM_ERR_TRUNCATED;
            uint8_t third = bytes[pos++];
            if (third == 0x2A && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 128, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                if ((modrm >> 6) == 3u) return X86ASM_ERR_UNRECOGNIZED;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                instruction->opcode = X86ASM_OP_MOVNTDQA;
                instruction->data_size = 128;
                instruction->memory_bytes = 16;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if (third == 0x17 && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size,
                                                 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) {
                    unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                    set_vector(&source, false, rm_index);
                }
                instruction->opcode = X86ASM_OP_PTEST;
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if ((third == 0x01 || third == 0x02 || third == 0x03) && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = third == 0x01 ? X86ASM_OP_PHADDW : (third == 0x02 ? X86ASM_OP_PHADDD : X86ASM_OP_PHADDSW);
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if (((third >= 0x20u && third <= 0x25u) || (third >= 0x30u && third <= 0x35u)) && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                if (third <= 0x25u) {
                    instruction->opcode = third == 0x20u ? X86ASM_OP_PMOVSXBW : (third == 0x21u ? X86ASM_OP_PMOVSXBD : (third == 0x22u ? X86ASM_OP_PMOVSXBQ : (third == 0x23u ? X86ASM_OP_PMOVSXWD : (third == 0x24u ? X86ASM_OP_PMOVSXWQ : X86ASM_OP_PMOVSXDQ))));
                } else {
                    instruction->opcode = third == 0x30u ? X86ASM_OP_PMOVZXBW : (third == 0x31u ? X86ASM_OP_PMOVZXBD : (third == 0x32u ? X86ASM_OP_PMOVZXBQ : (third == 0x33u ? X86ASM_OP_PMOVZXWD : (third == 0x34u ? X86ASM_OP_PMOVZXWQ : X86ASM_OP_PMOVZXDQ))));
                }
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? ((third == 0x20u || third == 0x30u || third == 0x23u || third == 0x33u || third == 0x25u || third == 0x35u) ? 8u : (third == 0x21u || third == 0x31u || third == 0x24u || third == 0x34u ? 4u : 2u)) : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if (third >= 0x38u && third <= 0x3Fu && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = third == 0x38u ? X86ASM_OP_PMINSB : (third == 0x39u ? X86ASM_OP_PMINSD : (third == 0x3Au ? X86ASM_OP_PMINUW : (third == 0x3Bu ? X86ASM_OP_PMINUD : (third == 0x3Cu ? X86ASM_OP_PMAXSB : (third == 0x3Du ? X86ASM_OP_PMAXSD : (third == 0x3Eu ? X86ASM_OP_PMAXUW : X86ASM_OP_PMAXUD))))));
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if (third == 0x40 && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = X86ASM_OP_PMULLD;
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if (third == 0x41 && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = X86ASM_OP_PHMINPOSUW;
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if ((third == 0x14 || third == 0x15) && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = third == 0x14 ? X86ASM_OP_BLENDVPS : X86ASM_OP_BLENDVPD;
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                set_vector(&instruction->arguments[2], false, 0u);
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if (third == 0x10 && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = X86ASM_OP_PBLENDVB;
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                set_vector(&instruction->arguments[2], false, 0u);
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if (third == 0x28 && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = X86ASM_OP_PMULDQ;
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if (third == 0x04 && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = X86ASM_OP_PMADDUBSW;
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if ((third == 0x05 || third == 0x06 || third == 0x07) && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = third == 0x05 ? X86ASM_OP_PHSUBW : (third == 0x06 ? X86ASM_OP_PHSUBD : X86ASM_OP_PHSUBSW);
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if ((third == 0x08 || third == 0x09 || third == 0x0A) && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = third == 0x08 ? X86ASM_OP_PSIGNB : (third == 0x09 ? X86ASM_OP_PSIGNW : X86ASM_OP_PSIGND);
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if ((third == 0x1C || third == 0x1D || third == 0x1E) && operand_override) {
                uint8_t modrm;
                x86asm_register reg;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, 64, rex, &modrm, &source, &reg);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) set_vector(&source, false, rm_index);
                instruction->opcode = third == 0x1C ? X86ASM_OP_PABSB : (third == 0x1D ? X86ASM_OP_PABSW : X86ASM_OP_PABSD);
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if (third == 0x00 && operand_override) {
                uint8_t modrm;
                x86asm_register ignored_register;
                x86asm_argument source = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size,
                                                 64, rex, &modrm, &source, &ignored_register);
                if (error != X86ASM_OK) return error;
                unsigned reg_index = ((modrm >> 3) & 7u) | ((rex & 4u) != 0 ? 8u : 0u);
                if ((modrm >> 6) == 3u) {
                    unsigned rm_index = (modrm & 7u) | ((rex & 1u) != 0 ? 8u : 0u);
                    set_vector(&source, false, rm_index);
                }
                instruction->opcode = X86ASM_OP_PSHUFB;
                instruction->data_size = 128;
                instruction->memory_bytes = source.kind == X86ASM_ARG_MEMORY ? 16 : 0;
                set_vector(&instruction->arguments[0], false, reg_index);
                instruction->arguments[1] = source;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
            if (third == 0xF0 || third == 0xF1) {
                uint8_t modrm;
                x86asm_register reg;
                x86asm_argument rm_argument = { X86ASM_ARG_NONE, { 0 } };
                x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size,
                                                 operand_width, rex, &modrm, &rm_argument, &reg);
                if (error != X86ASM_OK) return error;
                if ((modrm >> 6) == 3u) return X86ASM_ERR_UNRECOGNIZED;
                instruction->opcode = X86ASM_OP_MOVBE;
                if (third == 0xF0) {
                    set_register(&instruction->arguments[0], reg);
                    instruction->arguments[1] = rm_argument;
                } else {
                    instruction->arguments[0] = rm_argument;
                    set_register(&instruction->arguments[1], reg);
                }
                instruction->memory_bytes = operand_width / 8;
                instruction->length = (int)pos;
                return X86ASM_OK;
            }
        }
        return X86ASM_ERR_UNRECOGNIZED;
    }

    if (opcode_byte == 0x80 || opcode_byte == 0x81 || opcode_byte == 0x82 || opcode_byte == 0x83) {
        uint8_t modrm;
        x86asm_register ignored_register;
        int width = opcode_byte == 0x80 || opcode_byte == 0x82 ? 8 : operand_width;
        x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, width, rex, &modrm, &instruction->arguments[0], &ignored_register);
        if (error != X86ASM_OK) return error;
        instruction->opcode = arithmetic_group_opcode((modrm >> 3) & 7);
        if (opcode_byte == 0x80 || opcode_byte == 0x81 || opcode_byte == 0x82) {
            size_t immediate_size = width == 8 ? (size_t)1 : (size_t)(width / 8);
            if (pos + immediate_size > length) return X86ASM_ERR_TRUNCATED;
            set_immediate(&instruction->arguments[1], immediate_size == 1 ? (int8_t)bytes[pos] : (width == 16 ? (int16_t)read_u16(bytes + pos) : (int32_t)read_u32(bytes + pos)));
            pos += immediate_size;
        } else {
            if (pos >= length) return X86ASM_ERR_TRUNCATED;
            set_immediate(&instruction->arguments[1], (int8_t)bytes[pos++]);
        }
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (opcode_byte == 0xC6 || opcode_byte == 0xC7) {
        uint8_t modrm;
        x86asm_register ignored_register;
        int width = opcode_byte == 0xC6 ? 8 : operand_width;
        x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, width, rex, &modrm, &instruction->arguments[0], &ignored_register);
        if (error != X86ASM_OK) return error;
        if (((modrm >> 3) & 7) != 0) return X86ASM_ERR_UNRECOGNIZED;
        size_t immediate_size = width == 8 ? (size_t)1 : (size_t)(width / 8);
        if (pos + immediate_size > length) return X86ASM_ERR_TRUNCATED;
        instruction->opcode = X86ASM_OP_MOV;
        set_immediate(&instruction->arguments[1], immediate_size == 1 ? (int8_t)bytes[pos] : (width == 16 ? (int16_t)read_u16(bytes + pos) : (int32_t)read_u32(bytes + pos)));
        pos += immediate_size;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (opcode_byte == 0xF6 || opcode_byte == 0xF7) {
        uint8_t modrm;
        x86asm_register ignored_register;
        int width = opcode_byte == 0xF6 ? 8 : operand_width;
        x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, width, rex, &modrm, &instruction->arguments[0], &ignored_register);
        if (error != X86ASM_OK) return error;
        unsigned group = (modrm >> 3) & 7;
        if (group == 0) {
            instruction->opcode = X86ASM_OP_TEST;
            size_t immediate_size = width == 8 ? (size_t)1 : (size_t)(width / 8);
            if (pos + immediate_size > length) return X86ASM_ERR_TRUNCATED;
            set_immediate(&instruction->arguments[1], immediate_size == 1 ? (int8_t)bytes[pos] : (width == 16 ? (int16_t)read_u16(bytes + pos) : (int32_t)read_u32(bytes + pos)));
            pos += immediate_size;
        } else if (group == 2) {
            instruction->opcode = X86ASM_OP_NOT;
        } else if (group == 3) {
            instruction->opcode = X86ASM_OP_NEG;
        } else if (group == 4) {
            instruction->opcode = X86ASM_OP_MUL;
        } else if (group == 5) {
            instruction->opcode = X86ASM_OP_IMUL;
        } else if (group == 6) {
            instruction->opcode = X86ASM_OP_DIV;
        } else if (group == 7) {
            instruction->opcode = X86ASM_OP_IDIV;
        } else {
            return X86ASM_ERR_UNRECOGNIZED;
        }
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (opcode_byte == 0xC0 || opcode_byte == 0xC1 ||
        opcode_byte == 0xD0 || opcode_byte == 0xD1 ||
        opcode_byte == 0xD2 || opcode_byte == 0xD3) {
        uint8_t modrm;
        x86asm_register ignored_register;
        int width = (opcode_byte == 0xC0 || opcode_byte == 0xD0 || opcode_byte == 0xD2) ? 8 : operand_width;
        x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, width, rex, &modrm, &instruction->arguments[0], &ignored_register);
        if (error != X86ASM_OK) return error;
        instruction->data_size = width;
        instruction->memory_bytes = width / 8;
        instruction->opcode = shift_group_opcode((modrm >> 3) & 7);
        if (instruction->opcode == X86ASM_OP_INVALID) return X86ASM_ERR_UNRECOGNIZED;
        if (opcode_byte == 0xC0 || opcode_byte == 0xC1) {
            if (pos >= length) return X86ASM_ERR_TRUNCATED;
            set_immediate(&instruction->arguments[1], bytes[pos++]);
        } else if (opcode_byte == 0xD2 || opcode_byte == 0xD3) {
            set_register(&instruction->arguments[1], X86ASM_REG_CL);
        } else {
            set_immediate(&instruction->arguments[1], 1);
        }
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (opcode_byte == 0xFE || opcode_byte == 0xFF) {
        uint8_t modrm;
        x86asm_register ignored_register;
        int width = opcode_byte == 0xFE ? 8 : (mode == 64 ? 64 : operand_width);
        x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, width, rex, &modrm, &instruction->arguments[0], &ignored_register);
        if (error != X86ASM_OK) return error;
        unsigned group = (modrm >> 3) & 7;
        if (group == 0) instruction->opcode = X86ASM_OP_INC;
        else if (group == 1) instruction->opcode = X86ASM_OP_DEC;
        else if (opcode_byte == 0xFF && group == 2) instruction->opcode = X86ASM_OP_CALL;
        else if (opcode_byte == 0xFF && group == 4) instruction->opcode = X86ASM_OP_JMP;
        else if (opcode_byte == 0xFF && group == 6) instruction->opcode = X86ASM_OP_PUSH;
        else return X86ASM_ERR_UNRECOGNIZED;
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    if (opcode_byte == 0x88 || opcode_byte == 0x89 || opcode_byte == 0x8A || opcode_byte == 0x8B ||
        opcode_byte == 0x8D || opcode_byte == 0x86 || opcode_byte == 0x87 ||
        opcode_byte == 0x01 || opcode_byte == 0x03 ||
        opcode_byte == 0x29 || opcode_byte == 0x2B || opcode_byte == 0x31 || opcode_byte == 0x33 ||
        opcode_byte == 0x39 || opcode_byte == 0x3B) {
        x86asm_register reg;
        int width = (opcode_byte == 0x88 || opcode_byte == 0x8A) ? 8 : operand_width;
        x86asm_error error = read_modrm(bytes, length, &pos, mode, instruction->address_size, width, rex, &(uint8_t){0}, &instruction->arguments[1], &reg);
        if (error != X86ASM_OK) return error;
        if (opcode_byte == 0x8D) instruction->opcode = X86ASM_OP_LEA;
        else if (opcode_byte == 0x86 || opcode_byte == 0x87) instruction->opcode = X86ASM_OP_XCHG;
        else if ((opcode_byte & 0x38) == 0x00) instruction->opcode = X86ASM_OP_ADD;
        else if ((opcode_byte & 0x38) == 0x28) instruction->opcode = X86ASM_OP_SUB;
        else if ((opcode_byte & 0x38) == 0x30) instruction->opcode = X86ASM_OP_XOR;
        else if ((opcode_byte & 0x38) == 0x38) instruction->opcode = X86ASM_OP_CMP;
        else instruction->opcode = X86ASM_OP_MOV;
        if (opcode_byte == 0x8A || opcode_byte == 0x8B || opcode_byte == 0x03 || opcode_byte == 0x2B || opcode_byte == 0x33 || opcode_byte == 0x3B) {
            set_register(&instruction->arguments[0], reg);
        } else {
            x86asm_argument tmp = instruction->arguments[1];
            instruction->arguments[1].kind = X86ASM_ARG_NONE;
            set_register(&instruction->arguments[1], reg);
            instruction->arguments[0] = tmp;
        }
        instruction->length = (int)pos;
        return X86ASM_OK;
    }

    return X86ASM_ERR_UNRECOGNIZED;
}

typedef struct x86asm_writer {
    char *buffer;
    size_t capacity;
    size_t length;
} x86asm_writer;

static void writer_append(x86asm_writer *writer, const char *text)
{
    size_t text_length = strlen(text);
    if (writer->capacity != 0 && writer->length < writer->capacity) {
        size_t available = writer->capacity - writer->length - 1;
        size_t copied = text_length < available ? text_length : available;
        memcpy(writer->buffer + writer->length, text, copied);
        writer->buffer[writer->length + copied] = '\0';
    }
    writer->length += text_length;
}

static void writer_appendf(x86asm_writer *writer, const char *format, ...)
{
    va_list arguments;
    va_start(arguments, format);
    char local[128];
    int count = vsnprintf(local, sizeof(local), format, arguments);
    va_end(arguments);
    if (count <= 0) return;
    if ((size_t)count < sizeof(local)) {
        writer_append(writer, local);
        return;
    }
    char *dynamic = (char *)malloc((size_t)count + 1);
    if (dynamic == NULL) return;
    va_start(arguments, format);
    vsnprintf(dynamic, (size_t)count + 1, format, arguments);
    va_end(arguments);
    writer_append(writer, dynamic);
    free(dynamic);
}

static void format_argument(x86asm_writer *writer, const x86asm_argument *argument)
{
    switch (argument->kind) {
    case X86ASM_ARG_REGISTER:
        writer_append(writer, x86asm_register_name(argument->value.reg));
        break;
    case X86ASM_ARG_IMMEDIATE:
        writer_appendf(writer, "0x%llx", (unsigned long long)argument->value.immediate);
        break;
    case X86ASM_ARG_RELATIVE:
        writer_appendf(writer, ".%+d", argument->value.relative);
        break;
    case X86ASM_ARG_MEMORY: {
        const x86asm_memory *memory = &argument->value.memory;
        writer_append(writer, "[");
        if (memory->segment != X86ASM_REG_NONE) {
            writer_append(writer, x86asm_register_name(memory->segment));
            writer_append(writer, ":");
        }
        bool wrote = false;
        if (memory->base != X86ASM_REG_NONE) {
            writer_append(writer, x86asm_register_name(memory->base));
            wrote = true;
        }
        if (memory->index != X86ASM_REG_NONE) {
            if (wrote) writer_append(writer, "+");
            if (memory->scale > 1) writer_appendf(writer, "%u*", memory->scale);
            writer_append(writer, x86asm_register_name(memory->index));
            wrote = true;
        }
        if (memory->displacement != 0 || !wrote) {
            if (wrote && memory->displacement >= 0) writer_append(writer, "+");
            writer_appendf(writer, "0x%llx", (unsigned long long)memory->displacement);
        }
        writer_append(writer, "]");
        break;
    }
    default:
        break;
    }
}

static size_t format_instruction(const x86asm_instruction *instruction,
                                 char *output, size_t output_size)
{
    x86asm_writer writer = { output, output_size, 0 };
    for (size_t i = 0; i < 14 && instruction->prefixes[i] != 0; ++i) {
        if ((instruction->prefixes[i] & X86ASM_PREFIX_IMPLICIT) == 0) {
            writer_appendf(&writer, "%02x ", instruction->prefixes[i] & 0xff);
        }
    }
    writer_append(&writer, x86asm_opcode_name(instruction->opcode));
    bool first = true;
    for (size_t i = 0; i < 6; ++i) {
        if (instruction->arguments[i].kind == X86ASM_ARG_NONE) break;
        writer_append(&writer, first ? " " : ", ");
        format_argument(&writer, &instruction->arguments[i]);
        first = false;
    }
    if (output_size != 0 && output != NULL) output[output_size - 1] = '\0';
    return writer.length;
}

size_t x86asm_format_default(const x86asm_instruction *instruction,
                             char *output, size_t output_size)
{
    if (instruction == NULL) return 0;
    return format_instruction(instruction, output, output_size);
}

size_t x86asm_format_intel(const x86asm_instruction *instruction, uint64_t pc,
                           char *output, size_t output_size)
{
    (void)pc;
    return x86asm_format_default(instruction, output, output_size);
}
