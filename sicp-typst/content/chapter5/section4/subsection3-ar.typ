// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([الكتل والإسنادات والإعلانات], label-name: <sec:block-assign-def-evaluation>)

#subheading([الكتل])

#idx("explicit-control evaluator for Python", sub: "blocks")
#idx("explicit-control evaluator for Python", sub: "declarations")

يُقيّم جسم الكتلة بالنسبة للبيئة الحالية الممتدة بإطار يربط جميع الأسماء المحلية بالقيمة #py("\"*unassigned*\""). نستخدم مؤقتًا المسجّل #py("val") لحفظ قائمة كل المتغيرات المُعلنة في الكتلة، وهي القائمة التي نحصل عليها بالمسح الخارج للإعلانات (#en[scanning out declarations]) عبر
#idx("scanning out declarations", sub: "in explicit-control evaluator")
#py("scan_out_declarations")
من القسم @sec:core-of-evaluator. ونفترض أن الدالتين #py("scan_out_declarations") و #py("list_of_unassigned") متاحتان كعمليتَي آلة.#footnote[تقترح الحاشية السفلية @foot:syntax-transformer أن التنفيذ الفعلي سيجري تحويلات البناء قبل تنفيذ البرنامج. وفي نفس السياق، يجب استخراج الأسماء المُعلنة في الكتل في خطوة معالجة مسبقة بدلاً من استخراجها في كل مرة يُقيّم فيها كتلة.]
#idx("evblock", decl: true)
#syntax("
\"ev_block\",
  assign(\"comp\", list(op(\"block_body\"), reg(\"comp\"))),
  assign(\"val\", list(op(\"scan_out_declarations\"), reg(\"comp\"))),

  save(\"comp\"),    // حتى نتمكن من استخدامه لحفظ قيم *unassigned* مؤقتاً
  assign(\"comp\", list(op(\"list_of_unassigned\"), reg(\"val\"))),
  assign(\"env\", list(op(\"extend_environment\"),
                     reg(\"val\"), reg(\"comp\"), reg(\"env\"))),
  restore(\"comp\"), // جسم الكتلة
  go_to(label(\"eval_dispatch\")),
	  ")

#subheading([الإسنادات والإعلانات])

الإسنادات
#idx("explicit-control evaluator for Python", sub: "assignments")
تتم معالجتها بواسطة #py("ev_assignment")، التي يتم الوصول إليها من #py("eval_dispatch") مع تعبير الإسناد في #py("comp"). يُقيّم الكود عند #py("ev_assignment") أولاً جزء القيمة من التعبير ثم يثبت القيمة الجديدة في البيئة.
يُفترض أن الدالة #py("assign_symbol_value") مريحة كعملية آلة.
#idx("evassignment", decl: true)
#snippet(```python
"ev_assignment",
  assign("unev", list(op("assignment_symbol"), reg("comp"))),
  save("unev"), // حفظ المتغير للاحقًا
  assign("comp", list(op("assignment_value_expression"), reg("comp"))),
  save("env"),
  save("continue"),
  assign("continue", label("ev_assignment_install")),
  go_to(label("eval_dispatch")), // تقييم قيمة الإسناد
"ev_assignment_install",
  restore("continue"),
  restore("env"),
  restore("unev"),
  perform(list(op("assign_symbol_value"),
               reg("unev"), reg("val"), reg("env"))),
  go_to(reg("continue")),
```)

الإعلانات
#idx("explicit-control evaluator for Python", sub: "declarations")
عن المتغيرات والثوابت تُعالج بطريقة مماثلة.
لاحظ أنه في حين أن قيمة الإسناد هي القيمة التي تم إسنادها، فإن قيمة الإعلان هي #py("undefined"). ويعالَج ذلك بضبط #py("val") على #py("undefined") قبل المتابعة.
وكما فعلنا في المُقيِّم فوق الدائري (#en[metacircular])، نحوِّل تعريفَ الدالة إلى إعلان ثابتٍ مكافئٍ قيمةُ تعبيره تعبيرُ لامدا. ويحدث ذلك عند #py("ev_function_definition")، التي تُجري التحويل في موضعِه داخل #py("comp") ثم تنتقل إلى #py("ev_declaration").

#idx("evfunctiondefinition", decl: true)#idx("evdeclaration", decl: true)
#snippet(```python
"ev_function_definition",
  assign("comp",
         list(op("function_decl_to_constant_decl"), reg("comp"))),
"ev_declaration",
  assign("unev", list(op("declaration_symbol"), reg("comp"))),
  save("unev"), // حفظ الاسم المُعلن
  assign("comp",
         list(op("declaration_value_expression"), reg("comp"))),
  save("env"),
  save("continue"),
  assign("continue", label("ev_declaration_assign")),
  go_to(label("eval_dispatch")), // تقييم قيمة الإعلان
"ev_declaration_assign",
  restore("continue"),
  restore("env"),
  restore("unev"),
  perform(list(op("assign_symbol_value"),
               reg("unev"), reg("val"), reg("env"))),
  assign("val", constant(undefined)),
  go_to(reg("continue")),
```)

#exercise(label-name: <ex:derived-expressions>, [
وسّع المُقيِّم للتعامل مع
#idx("derived component", sub: "adding to explicit-control evaluator")
#idx("explicit-control evaluator for Python", sub: "derived components")
#idx("explicit-control evaluator for Python", sub: "syntactic forms (additional)")
حلقات #py("while")، عن طريق ترجمتها إلى تطبيقات لدالة #py("while_loop")، كما هو موضح في التمرين @ex:while_loop.
يمكنك لصق إعلان الدالة #py("while_loop") أمام برامج المستخدم.
يمكنك "الغش" بافتراض أن محول البناء #py("while_to_application") متاح كعملية آلة. ارجع إلى التمرين @ex:while_loop لمناقشة ما إذا كان هذا النهج يعمل إذا سُمح بتعليمات #py("return") و #py("break") و #py("continue") داخل حلقة while. إذا لم يكن كذلك، كيف يمكنك تعديل مُقيِّم التحكم الصريح لتشغيل البرامج مع حلقات while التي تتضمن هذه التعليمات؟
])

#exercise(label-name: <ex:5_26>, [
عدل المُقيِّم بحيث يستخدم
#idx("explicit-control evaluator for Python", sub: "normal-order evaluation")
#idx("normal-order evaluation", sub: "in explicit-control evaluator")
تقييم الترتيب العادي، استناداً إلى المُقيِّم الكسول في القسم @sec:lazy-evaluation.
])
