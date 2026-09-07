// listings-languages.typ — Language definitions and syntax highlight maps for C and RISC-V assembly.

#let listings-language-aliases = (
  "c": "C",
  "C": "C",
  "riscv": "RISC-V",
  "RISC-V": "RISC-V",
  "asm": "RISC-V",
  "assembly": "RISC-V",
  "cpp": "C",
)

#let listings-driver-names = ("C", "RISC-V", "Python", "Bash")

#let listings-language(name) = {
  if name == none { return none }
  let lower-name = lower(str(name))
  if lower-name == "c" or lower-name == "cpp" or lower-name == "c++" {
    return (
      name: "C",
      keywords: ("if", "else", "while", "for", "return", "struct", "void", "int", "char", "unsigned", "static", "extern", "typedef", "sizeof", "switch", "case", "default", "break", "continue"),
      types: ("uint", "uint64", "uint32", "uint16", "uint8", "pde_t", "pte_t", "pagetable_t", "proc", "spinlock", "sleeplock", "stat", "inode"),
      comments: ("//", "/*"),
      strings: ("\"", "'"),
    )
  } else if lower-name == "riscv" or lower-name == "asm" or lower-name == "assembly" {
    return (
      name: "RISC-V",
      keywords: ("add", "addi", "sub", "lui", "auipc", "jal", "jalr", "beq", "bne", "blt", "bge", "ld", "sd", "lw", "sw", "csrr", "csrw", "csrrw", "csrrs", "csrrc", "ecall", "sret", "mret"),
      types: ("zero", "ra", "sp", "gp", "tp", "t0", "t1", "t2", "s0", "s1", "a0", "a1", "a2", "a3", "a4", "a5", "satp", "stvec", "sepc", "sstatus", "scause"),
      comments: ("#", "//"),
      strings: ("\"", "'"),
    )
  }
  return (name: str(name), keywords: (), types: (), comments: (), strings: ())
}
