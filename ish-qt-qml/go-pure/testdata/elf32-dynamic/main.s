.section .text
.globl _start
.type _start,@function
.extern foo
_start:
	# The pure-Go guest enters _start directly; initialize the i386 PIC GOT base.
	mov $0x403240, %ebx
	call foo@PLT
	mov %eax, %ebx
	mov $1, %eax
	int $0x80
.size _start, .-_start
