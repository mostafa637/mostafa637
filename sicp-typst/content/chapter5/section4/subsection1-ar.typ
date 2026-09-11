// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([المُوجِّه والتقييم الأساسي], label-name: <sec:eceval-core>)

#idx("explicit-control evaluator for Python", sub: "controller")

العنصر المركزي في المُقيِّم هو تسلسل التعليمات الذي يبدأ عند #py("eval_dispatch").
هذا يتوافق مع الدالة #py("evaluate") للمُقيِّم فوق الدائري الموصوف في القسم @sec:core-of-evaluator. عندما تبدأ وحدة التحكم عند #py("eval_dispatch")، فإنها تُقيّم المكون المحدد بواسطة #py("comp") في البيئة المحددة بواسطة #py("env"). عند اكتمال التقييم، ستذهب وحدة التحكم إلى نقطة الإدخال المخزنة في #py("continue")، وسيحتفظ المسجّل #py("val") بقيمة المكون. كما هو الحال مع #py("evaluate") فوق الدائري، فإن بنية #py("eval_dispatch") هي تحليل للحالات على النوع النحوي للمكون المراد تقييمه.#footnote[في وحدة التحكم الخاصة بنا، يُكتب التوجيه كتسلسل من تعليمات #py("test") و #py("branch"). بدلاً من ذلك، كان يمكن كتابته بأسلوب موجه بالبيانات، مما يتجنب الحاجة إلى إجراء اختبارات تتابعية ويسهل تعريف أنواع مكونات جديدة.]
#idx("evaldispatch", decl: true)
#snippet(```python
"eval_dispatch",
  test(list(op("is_literal"), reg("comp"))),
  branch(label("ev_literal")),
  test(list(op("is_name"), reg("comp"))),
  branch(label("ev_name")),
  test(list(op("is_application"), reg("comp"))),
  branch(label("ev_application")),
  test(list(op("is_operator_combination"), reg("comp"))),
  branch(label("ev_operator_combination")),
  test(list(op("is_conditional"), reg("comp"))),
  branch(label("ev_conditional")),
  test(list(op("is_lambda_expression"), reg("comp"))),
  branch(label("ev_lambda")),
  test(list(op("is_sequence"), reg("comp"))),
  branch(label("ev_sequence")),
  test(list(op("is_block"), reg("comp"))),
  branch(label("ev_block")),
  test(list(op("is_return_statement"), reg("comp"))),
  branch(label("ev_return")),
  test(list(op("is_function_definition"), reg("comp"))),
  branch(label("ev_function_definition")),
  test(list(op("is_declaration"), reg("comp"))),
  branch(label("ev_declaration")),
  test(list(op("is_assignment"), reg("comp"))),
  branch(label("ev_assignment")),
  go_to(label("unknown_component_type")),
```)

#subheading([تقييم التعبيرات البسيطة])

#idx("explicit-control evaluator for Python", sub: "expressions with no subexpressions to evaluate")

الأرقام والسلاسل النصية والأسماء وتعبيرات لامدا (#en[lambda]) ليس لها تعبيرات فرعية لتقييمها. بالنسبة لهذه، يضع المُقيِّم القيمة الصحيحة ببساطة في المسجّل #py("val") ويواصل التنفيذ عند نقطة الإدخال المحددة بواسطة #py("continue"). يتم تقييم التعبيرات البسيطة بواسطة كود وحدة التحكم التالي:
#idx("evliteral", decl: true)#idx("evname", decl: true)#idx("evlambda", decl: true)
#snippet(```python
"ev_literal",
  assign("val", list(op("literal_value"), reg("comp"))),
  go_to(reg("continue")),

"ev_name",
  assign("val", list(op("symbol_of_name"), reg("comp"), reg("env"))),
  assign("val", list(op("lookup_symbol_value"),
                     reg("val"), reg("env"))),
  go_to(reg("continue")),

"ev_lambda",
  assign("unev", list(op("lambda_parameter_symbols"), reg("comp"))),
  assign("comp", list(op("lambda_body"), reg("comp"))),
  assign("val", list(op("make_function"),
                     reg("unev"), reg("comp"), reg("env"))),
  go_to(reg("continue")),
```)

لاحظ كيف تستخدم #py("ev_lambda") المسجّلين #py("unev") و #py("comp") لحفظ المعلمات وجسم تعبير لامدا حتى يمكن تمريرهما إلى عملية #py("make_function")، إلى جانب البيئة في #py("env").

#idx("explicit-control evaluator for Python", sub: "expressions with no subexpressions to evaluate")

#subheading([الشرطيات])

كما هو الحال مع المُقيِّم فوق الدائري (#en[metacircular])، تتم معالجة الأشكال النحوية عن طريق تقييم أجزاء من المكون بشكل انتقائي. بالنسبة لـ
#idx("explicit-control evaluator for Python", sub: "conditionals")
الشرطي، يجب علينا تقييم المحمول وتقرير ما إذا كنا سنقيم النتيجة أم البديل بناءً على قيمة المحمول.

قبل تقييم المحمول، نحفظ الشرطي نفسه الموجود في #py("comp")، حتى نتمكن لاحقًا من استخراج النتيجة أو البديل. لتقييم تعبير المحمول، ننقله إلى المسجّل #py("comp") وننتقل إلى #py("eval_dispatch"). البيئة في المسجّل #py("env") هي بالفعل البيئة الصحيحة التي يتم فيها تقييم المحمول. ومع ذلك، نحفظ #py("env") لأننا سنحتاجه لاحقًا لتقييم النتيجة أو البديل. نقوم بإعداد #py("continue") بحيث يستأنف التقييم عند #py("ev_conditional_decide") بعد تقييم المحمول. أولاً، ومع ذلك، نحفظ القيمة القديمة لـ #py("continue")، والتي سنحتاجها لاحقًا للعودة إلى تقييم العبارة التي تنتظر قيمة الشرطي.
#idx("evconditional", decl: true)
#snippet(```python
"ev_conditional",
  save("comp"), // حفظ الشرطي للاحقًا
  save("env"),
  save("continue"),
  assign("continue", label("ev_conditional_decide")),
  assign("comp", list(op("conditional_predicate"), reg("comp"))),
  go_to(label("eval_dispatch")), // تقييم المحمول
```)

عندما نستأنف عند #py("ev_conditional_decide") بعد تقييم المحمول، نختبر ما إذا كان صحيحًا أم خاطئًا وبناءً على النتيجة، نضع إما النتيجة أو البديل في #py("comp") قبل الانتقال إلى #py("eval_dispatch").#footnote[في هذا الفصل، سنستخدم الدالة
#idx("isfalsy", sub: "why used in explicit-control evaluator", decl: true)
#py("is_falsy") لاختبار قيمة المحمول.
يسمح لنا هذا بكتابة فرعي النتيجة والبديل بنفس الترتيب كما في الشرطي، والانتقال ببساطة إلى فرع النتيجة عندما يتحقق المحمول. أُعلنت الدالة #py("is_falsy") كعكس لـ الدالة #py("is_truthy") المستخدمة لاختبار محمولات الشرطيات في القسم @sec:core-of-evaluator.]
لاحظ أن استعادة #py("env") و #py("continue") هنا تُعد #py("eval_dispatch") ليمتلك البيئة الصحيحة وليستمر في المكان الصحيح لاستلام قيمة الشرطي.

#snippet(```python
"ev_conditional_decide",
  restore("continue"),
  restore("env"),
  restore("comp"),
  test(list(op("is_falsy"), reg("val"))),
  branch(label("ev_conditional_alternative")),
"ev_conditional_consequent",
  assign("comp", list(op("conditional_consequent"), reg("comp"))),
  go_to(label("eval_dispatch")),
"ev_conditional_alternative",
  assign("comp", list(op("conditional_alternative"), reg("comp"))),
  go_to(label("eval_dispatch")),
```)

#subheading([تقييم التسلسلات])

#idx("explicit-control evaluator for Python", sub: "sequences of statements")

جزء مُقيِّم التحكم الصريح الذي يبدأ عند #py("ev_sequence")، والذي يتعامل مع تسلسلات العبارات، يناظر دالة #py("eval_sequence") للمُقيِّم فوق الدائري.

الإدخالان عند #py("ev_sequence_next") و #py("ev_sequence_continue") يشكلان حلقة تُقيّم بالتتابع كل عبارة في تسلسل.
تُمكث قائمة العبارات غير المُقيِّمة في #py("unev").
عند #py("ev_sequence") نضع تسلسل العبارات المراد تقييمها في #py("unev"). إذا كان التسلسل فارغًا، نضبط #py("val") على #py("undefined") ونقفز إلى #py("continue") عبر #py("ev_sequence_empty"). بخلاف ذلك نبدأ حلقة تقييم التسلسل، أولاً بحفظ قيمة #py("continue") على المكدس، لأن المسجّل #py("continue") سيُستخدم للتدفق المحلي للتحكم في الحلقة، والقيمة الأصلية مطلوبة للاستمرار بعد تسلسل العبارات. قبل تقييم كل عبارة، نفحص ما إذا كانت هناك عبارات إضافية مراد تقييمها في التسلسل. إذا كان الأمر كذلك، نحفظ باقي العبارات غير المُقيِّمة (الموجودة في #py("unev")) والبيئة التي يجب تقييمها فيها (الموجودة في #py("env")) ونستدعي #py("eval_dispatch") لتقييم العبارة التي وضعت في #py("comp"). يستعاد المسجّلان المحفوظان بعد هذا التقييم، عند #py("ev_sequence_continue").

تتم معالجة العبارة الأخيرة في التسلسل بشكل مختلف، عند نقطة الإدخال #py("ev_sequence_last_statement"). نظرًا لعدم وجود المزيد من العبارات التي تلزم تقييمها بعد هذه العبارة، لا نحتاج إلى حفظ #py("unev") أو #py("env") قبل الانتقال إلى #py("eval_dispatch"). قيمة التسلسل بأكمله هي قيمة العبارة الأخيرة، لذا بعد تقييم العبارة الأخيرة لا يتبقى أي شيء للقيام به سوى الاستمرار عند نقطة الإدخال التي حُفظت عند #py("ev_sequence").
بدلاً من إعداد #py("continue") للترتيب لـ #py("eval_dispatch") ليعود إلى هنا ثم استعادة #py("continue") من المكدس والمواكبة عند نقطة الإدخال تلك، نستعيد #py("continue") من المكدس قبل الانتقال إلى #py("eval_dispatch")، بحيث يواصل #py("eval_dispatch") عند نقطة الإدخال تلك بعد تقييم العبارة.

#idx("evsequence", decl: true)
#snippet(```python
"ev_sequence",
  assign("unev", list(op("sequence_statements"), reg("comp"))),
  test(list(op("is_empty_sequence"), reg("unev"))),
  branch(label("ev_sequence_empty")),
  save("continue"),
"ev_sequence_next",
  assign("comp", list(op("first_statement"), reg("unev"))),
  test(list(op("is_last_statement"), reg("unev"))),
  branch(label("ev_sequence_last_statement")),
  save("unev"),
  save("env"),
  assign("continue", label("ev_sequence_continue")),
  go_to(label("eval_dispatch")),
"ev_sequence_continue",
  restore("env"),
  restore("unev"),
  assign("unev", list(op("rest_statements"), reg("unev"))),
  go_to(label("ev_sequence_next")),
"ev_sequence_last_statement",
  restore("continue"),
  go_to(label("eval_dispatch")),

"ev_sequence_empty",
  assign("val", constant(undefined)),
  go_to(reg("continue")),
```)

على عكس #py("eval_sequence") في المُقيِّم فوق الدائري، لا تحتاج #py("ev_sequence") إلى فحص ما إذا تم تقييم تعليمة إرجاع من أجل إنهاء تقييم التسلسل. "التحكم الصريح" في هذا المُقيِّم يسمح لتعليمة الإرجاع بالقفز مباشرة إلى مواصلة تطبيق الدالة الحالي دون استئناف تقييم التسلسل. وبالتالي لا يحتاج تقييم التسلسل إلى الاهتمام بالإرجاعات، أو حتى الإلمام بوجود تعليمات إرجاع في اللغة. نظرًا لأن تعليمة الإرجاع تقفز خارج كود تقييم التسلسل، فإن استعادات المسجّلات المحفوظة عند #py("ev_sequence_continue") لن تنفذ. سنرى لاحقًا كيف تزيل تعليمة الإرجاع هذه القيم من المكدس.

#idx("explicit-control evaluator for Python", sub: "sequences of statements")
