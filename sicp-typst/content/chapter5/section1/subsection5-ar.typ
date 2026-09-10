// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([ملخص التعليمات], label-name: <sec:instruction-summary>)

#idx("register-machine language", sub: "instructions")
#idx("register-machine language")

تعليمة وحدة التحكم في لغة آلة المسجّلات الخاصة بنا لها أحد الأشكال التالية، حيث كل
#meta("input")$""_(i)$ هو

#idx("register-machine language", sub: "reg")
#idx("register-machine language", sub: "constant")
#py("reg(")#meta("register-name")#py(")")
أو
#py("constant(")#meta("constant-value")#py(")").

تم تقديم هذه التعليمات في القسم @sec:register-machine-language:
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

تم تقديم استخدام المسجّلات لحفظ التسميات في القسم @sec:subroutines:

#syntax("
assign(", meta("register-name"), ", label(", meta("label-name"), "))

go_to(reg(", meta("register-name"), "))
      ")

تم تقديم تعليمات استخدام المكدس في القسم @sec:stack-recursion:
#idx("register-machine language", sub: "save")#idx("register-machine language", sub: "restore")
#syntax("
save(", meta("register-name"), ")

restore(", meta("register-name"), ")
      ")

النوع الوحيد من
#idx("register-machine language", sub: "constant")
#idx("constant (in register machine)", sub: "syntax of")
#meta("constant-value")
الذي رأيناه حتى الآن هو الرقم، ولكن لاحقًا سنستخدم أيضًا السلاسل النصية (#en[strings]) والقوائم (#en[lists]).

على سبيل المثال،
#py("constant(\"abc\")")
هو سلسلة نصية #py("\"abc\"")، و
#py("constant(null)")
هو القائمة الفارغة، و
#py("constant(llist(\"a\", \"b\", \"c\"))")
هو القائمة
#py("llist(\"a\", \"b\", \"c\")").

#idx("register machine", sub: "design of")
#idx("register-machine language")
