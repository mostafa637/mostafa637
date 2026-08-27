CC ?= cc
CFLAGS ?= -std=c99 -Wall -Wextra -Wpedantic -Wconversion -Wshadow -O2

all: test_x86asm test_x86emu

libx86asm.a: x86asm.o
	ar rcs $@ $^

x86asm.o: x86asm.c x86asm.h
	$(CC) $(CFLAGS) -c x86asm.c -o $@

test_x86asm: x86asm.o test_x86asm.o
	$(CC) $(CFLAGS) $^ -o $@

test_x86asm.o: test_x86asm.c x86asm.h
	$(CC) $(CFLAGS) -c test_x86asm.c -o $@

x86emu.o: x86emu.c x86emu.h x86asm.h
	$(CC) $(CFLAGS) -c x86emu.c -o $@

test_x86emu: x86asm.o x86emu.o test_x86emu.o
	$(CC) $(CFLAGS) $^ -o $@

test_x86emu.o: test_x86emu.c x86emu.h x86asm.h
	$(CC) $(CFLAGS) -c test_x86emu.c -o $@

check: test_x86asm test_x86emu
	./test_x86asm
	./test_x86emu

clean:
	rm -f *.o *.a test_x86asm test_x86emu

.PHONY: all check clean
