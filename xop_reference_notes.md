## AMD XOP verification

The reviewed XOP reference states that VPCMOV has four XMM/YMM register operands in the form `VPCMOV dest, src1, src2, selector`, with per-bit semantics `dest[i] = selector[i] ? src1[i] : src2[i]`. This means the current three-operand decode of VPCMOV is incomplete and must not be executed as if it were correct. The current implementation therefore remains decode-only for VPCMOV until its fourth selector operand is represented correctly.

Sources reviewed:

- AMD documentation URL requested by the project: https://www.amd.com/content/dam/amd/en/documents/archived-tech-docs/programmer-references/43479.pdf (the current AMD URL returned 404 during this session).
- XOP reference: https://chessprogramming.org/XOP
- Intel/X86 reference index: https://www.felixcloutier.com/x86/

The XOP page also identifies the AMD Volume 6 family of references for XOP, FMA4, and CVT16. No XOP executor was added based on guesswork.

Additional verification:

- Sandpile XOP opcode table: https://www.sandpile.org/x86/opc_xop.htm
- Local Linux opcode map: `linux_x86_opcode_map.txt`, XOP map 8h line 1128, records `VPCMOV Vx,Hx,Wx,Lx (W=0) | VPCMOV Vx,Hx,Lx,Wx (W=1)` and map 8h lines 1132-1135 records fixed-count `VPROTB/W/D/Q`.
- The table confirms that VPCMOV is a four-register operation and that operand order changes with the XOP W bit. The current C decoder's three-operand VPCMOV representation is consequently decode-only and must be redesigned before execution is enabled.
