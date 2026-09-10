// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Instruction Summary], label-name: <sec:instruction-summary>)

#idx("register-machine language", sub: "instructions")
#idx("register-machine language")

A controller instruction in our register-machine language
has one of the following forms, where each
#meta("input")$""_(i)$ is

#idx("register-machine language", sub: "reg")
#idx("register-machine language", sub: "constant")
#py("reg(")#meta("register-name")#py(")")
or
#py("constant(")#meta("constant-value")#py(")").

These instructions were introduced in
section @sec:register-machine-language:
#idx("register-machine language", sub: "goto")
#idx("register-machine language", sub: "assign")#idx("register-machine language", sub: "op")#idx("register-machine language", sub: "perform")#idx("register-machine language", sub: "test")#idx("register-machine language", sub: "branch")#idx("register-machine language", sub: "label")
#syntax("
assign(", meta("register-name"), ", reg(", meta("register-name"), "))

assign(", meta("register-name"), ", constant(", meta("constant-value"), "))

assign(", meta("register-name"), ", llist(op(", meta("operation-name"), "), ", meta("input"), $""_(1)$, ", ", $dots.h$, ", ", meta("input"), $""_(n)$, "))

perform(llist(op(", meta("operation-name"), "), ", meta("input"), $""_(1)$, ", ", $dots.h$, ", ", meta("input"), $""_(n)$, "))

test(llist(op(", meta("operation-name"), "), ", meta("input"), $""_(1)$, ", ", $dots.h$, ", ", meta("input"), $""_(n)$, "))

branch(label(", meta("label-name"), "))

go_to(label(", meta("label-name"), "))
      ")

The use of registers to hold labels was introduced in
section @sec:subroutines:

#syntax("
assign(", meta("register-name"), ", label(", meta("label-name"), "))

go_to(reg(", meta("register-name"), "))
      ")

Instructions to use the stack were introduced in
section @sec:stack-recursion:
#idx("register-machine language", sub: "save")#idx("register-machine language", sub: "restore")
#syntax("
save(", meta("register-name"), ")

restore(", meta("register-name"), ")
      ")

The only kind of
#idx("register-machine language", sub: "constant")
#idx("constant (in register machine)", sub: "syntax of")
#meta("constant-value")
we have seen so far is a number, but later we will
also use strings
and lists.

For example,
#py("constant(\"abc\")")
is the string #py("\"abc\""),
#py("constant(null)")
is the empty list, and
#py("constant(llist(\"a\", \"b\", \"c\"))")
is the list
#py("llist(\"a\", \"b\", \"c\")").

#idx("register machine", sub: "design of")
#idx("register-machine language")
