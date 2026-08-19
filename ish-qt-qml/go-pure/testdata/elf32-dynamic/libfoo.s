.section .text
.globl foo
.type foo,@function
foo:
	mov $42, %eax
	ret
.size foo, .-foo
