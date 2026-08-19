# ELF32 DT_NEEDED fixture

This directory contains a tiny i386 dynamic executable and one shared object. The executable enters `_start` directly, resolves `foo` through an i386 `R_386_JMP_SLOT` relocation, and exits through `int 0x80` with the return value from `foo`. `libfoo.so` returns `42` and has the SONAME `libfoo.so`.

The checked-in binaries are used by `internal/core/kernel/exec_test.go`. They do not depend on a host libc, an interpreter, or CGo; the guest loader must resolve `/lib/libfoo.so` from FakeFS and patch the PLT/GOT before execution.

To rebuild on a host with LLVM's i386 target:

```sh
clang --target=i386-linux-gnu -c -fPIC libfoo.s -o libfoo.o
ld.lld -m elf_i386 -shared -soname libfoo.so -o libfoo.so libfoo.o
clang --target=i386-linux-gnu -c main.s -o main.o
ld.lld -m elf_i386 -e _start --no-dynamic-linker --no-as-needed -L. -lfoo -o main main.o
rm -f main.o libfoo.o
```

The main executable is intentionally ET_EXEC so its direct `_start` entry can initialize the GOT base without requiring a compiler-generated i386 PIC thunk. The loader still exercises `DT_NEEDED`, cross-object symbol resolution, `R_386_JMP_SLOT`, and guest execution.
