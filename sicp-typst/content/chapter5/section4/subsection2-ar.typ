// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([تقييم تطبيقات الدوال], label-name: <sec:tail-recursion-return>)

#anchor(<sec:evaluating-function-applications>)

#idx("explicit-control evaluator for Python", sub: "function application")
#idx("explicit-control evaluator for Python", sub: "combinations")

تتحدد تطبيق الدالة بواسطة تركيب يحتوي على تعبير دالة وتعبيرات وسائط. تعبير الدالة هو تعبير فرعي قيمته دالة، وتعبيرات الوسائط هي تعبيرات فرعية قيمها هي الوسائط التي يجب تطبيق الدالة عليها. يتعامل #py("evaluate") فوق الدائري مع التطبيقات عن طريق استدعاء نفسه عوديًا لتقييم كل عنصر في التركيب، ثم تمرير النتائج إلى #py("apply")، التي تنفذ تطبيق الدالة الفعلي. يقوم مُقيِّم التحكم الصريح بنفس الشيء؛ وتُنفذ هذه الاستدعاءات العودية بواسطة تعليمات #py("go_to")، جنبًا إلى جنب مع استخدام المكدس لحفظ المسجّلات التي ستستعاد بعد عودة الاستدعاء العودي.
قبل كل استدعاء سنحرص على
#idx("explicit-control evaluator for Python", sub: "stack usage")
تحديد المسجّلات التي يجب حفظها (لأن قيمها ستلزم لاحقًا).#footnote[هذه نقطة مهمة ولكنها دقيقة في ترجمة الخوارزميات من لغة إجرائية، مثل بايثون، إلى لغة آلة مسجّلات. كبديل لحفظ ما يلزم فقط، كان بإمكاننا حفظ جميع المسجّلات (باستثناء #py("val")) قبل كل استدعاء عودي. ويسمى هذا بنظام
#idx("framed-stack discipline")
#idx("stack", sub: "framed")
#emph[المكدس المُمفصل] (#en[framed-stack]).
سيؤدي ذلك إلى النتيجة ولكنه قد يحفظ مسجّلات أكثر من اللازم؛ وقد يكون هذا اعتبارًا مهمًا في نظام تكون فيه عمليات المكدس مكلفة. قد يؤدي حفظ المسجّلات التي لن تلزم محتوياتها لاحقًا أيضًا إلى الاحتفاظ ببيانات غير مفيدة كان يمكن تجميعها كقمامة، مما يحرر مساحة لإعادة استخدامها.]

كما هو الحال في المُقيِّم فوق الدائري (#en[metacircular])، تُحول تركيبات المُعاملات إلى تطبيقات للدوال الأولية المقابلة للمُعاملات. يتم ذلك عند #py("ev_operator_combination")، والتي تنفذ هذا التحويل في مكانه في #py("comp") وتنتقل إلى #py("ev_application").#footnote[نفترض أن محول البناء #py("operator_combination_to_application") متاح كعملية آلة. في تنفيذ فعلي مبني من الصفر، سنستخدم مُقيِّم التحكم الصريح الخاص بنا لتفسير برنامج بايثون ينفذ تحويلات على مستوى المصدر مثل هذا التحويل و #py("function_decl_to_constant_decl") في مرحلة بناء تعمل قبل التنفيذ.]<foot:syntax-transformer>

نبدأ تقييم التطبيق بتقييم تعبير الدالة لإنتاج دالة، سيتم تطبيقها لاحقًا على تعبيرات الوسائط المقيّمة. لتقييم تعبير الدالة، ننقله إلى المسجّل #py("comp") وننتقل إلى #py("eval_dispatch"). البيئة في المسجّل #py("env") هي بالفعل البيئة الصحيحة التي يتم فيها تقييم تعبير الدالة. ومع ذلك، نحفظ #py("env") لأننا سنحتاجه لاحقًا لتقييم تعبيرات الوسائط. نستخرج أيضًا تعبيرات الوسائط في #py("unev") ونحفظها على المكدس. نقوم بإعداد #py("continue") بحيث يستأنف #py("eval_dispatch") عند #py("ev_appl_did_function_expression") بعد تقييم تعبير الدالة. أولاً، ومع ذلك، نحفظ القيمة القديمة لـ #py("continue")، والتي تخبر وحدة التحكم أين تواصل بعد التطبيق.

#idx("evoperatorcombination", decl: true)#idx("evapplication", decl: true)
#snippet(```python
"ev_operator_combination",
  assign("comp", list(op("operator_combination_to_application"),
                      reg("comp"), reg("env"))),
"ev_application",
  save("continue"),
  save("env"),
  assign("unev", list(op("arg_expressions"), reg("comp"))),
  save("unev"),
  assign("comp", list(op("function_expression"), reg("comp"))),
  assign("continue", label("ev_appl_did_function_expression")),
  go_to(label("eval_dispatch")),
```)

#idx("explicit-control evaluator for Python", sub: "argument evaluation")

عند العودة من تقييم تعبير الدالة، نواصل تقييم تعبيرات الوسائط للتطبيق وتجميع الوسائط الناتجة في قائمة، ممسوكة في #py("argl"). (هذا يشبه تقييم تسلسل من العبارات، باستثناء أننا نجمع القيم). أولاً نستعيد تعبيرات الوسائط غير المقيّمة والبيئة. نُهيئ #py("argl") إلى قائمة فارغة. ثم نسند إلى المسجّل #py("fun") الدالة الناتجة عن تقييم تعبير الدالة. إذا لم تكن هناك تعبيرات وسائط، ننتقل مباشرة إلى #py("apply_dispatch"). بخلاف ذلك نحفظ #py("fun") على المكدس ونبدأ حلقة تقييم الوسائط:#footnote[نضيف إلى دوال بنية البيانات في المُقيِّم في القسم @sec:eval-data-structures الدالتين التاليتين لمعالجة قوائم الوسائط:
#idx("emptyarglist", decl: true)#idx("adjoinarg", decl: true)
#snippet(```python
function empty_arglist() { return null; }

function adjoin_arg(arg, arglist) {
    return append(arglist, list(arg));
}
```)

نستخدم أيضًا دالة بناء إضافية لاختبار تعبير الوسيط الأخير في تطبيق ما:
#idx("islastargumentexpression", decl: true)
#snippet(```python
function is_last_argument_expression(arg_expression) {
    return is_null(tail(arg_expression));
}
```)]

#snippet(```python
"ev_appl_did_function_expression",
  restore("unev"), // تعبيرات الوسائط
  restore("env"),
  assign("argl", list(op("empty_arglist"))),
  assign("fun", reg("val")), // الدالة
  test(list(op("is_null"), reg("unev"))),
  branch(label("apply_dispatch")),
  save("fun"),
```)

كل دورة من حلقة تقييم الوسائط تُقيّم تعبير وسيط من القائمة في #py("unev") وتجمع النتيجة في #py("argl"). لتقييم تعبير وسيط، ننقله إلى المسجّل #py("comp") وننتقل إلى #py("eval_dispatch")، بعد ضبط #py("continue") بحيث يستأنف التنفيذ مع مرحلة تجميع الوسائط. ولكن أولاً نحفظ الوسائط المجمعة حتى الآن (الموجودة في #py("argl"))، والبيئة (الموجودة في #py("env"))، وتعبيرات الوسائط المتبقية المراد تقييمها (الموجودة في #py("unev")). تخصص حالة خاصة لتقييم تعبير الوسيط الأخير، والتي تتم معالجتها عند #py("ev_appl_last_arg").

#snippet(```python
"ev_appl_argument_expression_loop",
  save("argl"),
  assign("comp", list(op("head"), reg("unev"))),
  test(list(op("is_last_argument_expression"), reg("unev"))),
  branch(label("ev_appl_last_arg")),
  save("env"),
  save("unev"),
  assign("continue", label("ev_appl_accumulate_arg")),
  go_to(label("eval_dispatch")),
```)

عند تقييم تعبير وسيط، تُجمع القيمة في القائمة الموجودة في #py("argl"). ثم يُزال تعبير الوسيط من قائمة تعبيرات الوسائط غير المقيّمة في #py("unev")، وتستمر حلقة تقييم الوسائط.

#snippet(```python
"ev_appl_accumulate_arg",
  restore("unev"),
  restore("env"),
  restore("argl"),
  assign("argl", list(op("adjoin_arg"), reg("val"), reg("argl"))),
  assign("unev", list(op("tail"), reg("unev"))),
  go_to(label("ev_appl_argument_expression_loop")),
```)

تتم معالجة تقييم تعبير الوسيط الأخير بشكل مختلف، كما هو الحال مع العبارة الأخيرة في تسلسل. لا داعي لحفظ البيئة أو قائمة تعبيرات الوسائط غير المقيّمة قبل الانتقال إلى #py("eval_dispatch")، حيث لن يلزم وجودها بعد تقييم تعبير الوسيط الأخير. وبالتالي، نعود من التقييم إلى نقطة إدخال خاصة #py("ev_appl_accum_last_arg")، والتي تستعيد قائمة الوسائط، وتجمع الوسيط الجديد، وتستعيد الدالة المحفوظة، وتذهب لتنفيذ التطبيق.#footnote[يُعرف تحسين معاملة تعبير الوسيط الأخير بشكل خاص بـ
#idx("evlis tail recursion")
#emph[العودية الذيلية لـ evlis] (انظر
#idx("Wand, Mitchell")
واند 1980).
كان يمكننا أن نكون أكثر كفاءة إلى حد ما في حلقة تقييم الوسائط إذا جعلنا تقييم تعبير الوسيط الأول حالة خاصة أيضًا. وكان هذا سيتيح لنا تأجيل تهيئة #py("argl") حتى بعد تقييم تعبير الوسيط الأول، من أجل تجنب حفظ #py("argl") في هذه الحالة. يُنفذ المُترجِم في القسم @sec:compilation هذا التحسين. (قارن الدالة #py("construct_arglist") في القسم @sec:compiling-combinations).]

#snippet(```python
"ev_appl_last_arg",
  assign("continue", label("ev_appl_accum_last_arg")),
  go_to(label("eval_dispatch")),
"ev_appl_accum_last_arg",
  restore("argl"),
  assign("argl", list(op("adjoin_arg"), reg("val"), reg("argl"))),
  restore("fun"),
  go_to(label("apply_dispatch")),
```)

تفاصيل حلقة تقييم الوسائط تحدد
#idx("order of evaluation", sub: "in explicit-control evaluator")
الترتيب الذي يُقيّم به المُفسِّر تعبيرات الوسائط لتركيب ما (على سبيل المثال، من اليسار إلى اليمين أو من اليمين إلى اليسار—انظر التمرين @ex:order-of-evaluation).
هذا الترتيب لا يحدده المُقيِّم فوق الدائري، الذي يرث بنية التحكم الخاصة به من JavaScript الأساسية التي نُفذ بها.#footnote[يحدد ترتيب تقييم تعبير الوسيط بواسطة الدالة #py("list_of_values") في المُقيِّم فوق الدائري بواسطة ترتيب تقييم وسائط #py("pair")، والتي تُستخدم لبناء قائمة الوسائط.
#idx("order of evaluation", sub: "in metacircular evaluator")
نسخة #py("list_of_values") في الحاشية السفلية @foot:mceval-higher-order من القسم @sec:mc-eval تستدعي #py("pair") مباشرة؛ بينما النسخة في النص تستخدم #py("map")، التي تستدعي #py("pair"). (انظر التمرين @ex:arg-eval-order).]

نظرًا لأننا نستخدم #py("head") في #py("ev_appl_argument_expression_loop") لاستخراج تعبيرات الوسائط المتتالية من #py("unev") و #py("tail") عند #py("ev_appl_accumulate_arg") لاستخراج باقي تعبيرات الوسائط، فإن مُقيِّم التحكم الصريح سُيقيّم تعبيرات الوسائط لتركيب ما بترتيب من اليسار إلى اليمين، كما تقتضي مواصفات ECMAScript.

#idx("explicit-control evaluator for Python", sub: "argument evaluation")

#subheading([تطبيق الدالة])

#anchor(<sec:procedure-application>)

تناظر نقطة الإدخال #py("apply_dispatch") الدالة #py("apply") للمُقيِّم فوق الدائري. بحلول الوقت الذي نصل فيه إلى #py("apply_dispatch")، يحتوي المسجّل #py("fun") على الدالة المراد تطبيقها ويحتوي #py("argl") على قائمة الوسائط المقيّمة التي يجب تطبيق الدالة عليها. تكون القيمة المحفوظة لـ #py("continue") (والتي تم تمريرها في الأصل إلى #py("eval_dispatch") وحُفظت عند #py("ev_application"))، والتي تخبرنا أين نعود بنتيجة تطبيق الدالة، على المكدس. عند اكتمال التطبيق، تنتقل وحدة التحكم إلى نقطة الإدخال المحددة بواسطة #py("continue") المحفوظة، وتكون نتيجة التطبيق في #py("val"). كما هو الحال مع #py("apply") فوق الدائري، هناك حالتان ينبغي مراعاتهما.
إما أن تكون الدالة المراد تطبيقها أولية أو أن تكون دالة مركبة.

#idx("applydispatch", decl: true)
#snippet(```python
"apply_dispatch",
  test(list(op("is_primitive_function"), reg("fun"))),
  branch(label("primitive_apply")),
  test(list(op("is_compound_function"), reg("fun"))),
  branch(label("compound_apply")),
  go_to(label("unknown_function_type")),
```)

نفترض أن كل
#idx("explicit-control evaluator for Python", sub: "primitive functions")
أولية تُنفذ للحصول على وسائطها من #py("argl") ووضع نتيجتها في #py("val"). لتحديد كيفية التعامل الآلة مع الأوليات، كان سيتعين علينا تقديم تسلسل من تعليمات وحدة التحكم لتنفيذ كل أولية والترتيب لـ #py("primitive_apply") ليوجه إلى التعليمات الخاصة بالأولية المحددة بمحتويات #py("fun"). نظرًا لأننا مهتمون ببنية عملية التقييم بدلاً من تفاصيل الأوليات، فسنستخدم بدلاً من ذلك عملية #py("apply_primitive_function") تطبق الدالة في fun على الوسائط في #py("argl"). لأغراض محاكاة المُقيِّم باستخدام المحاكي في القسم @sec:simulator، نستخدم الدالة #py("apply_primitive_function")، التي تستدعي نظام JavaScript الأساسي لتنفيذ التطبيق، كما فعلنا للمُقيِّم فوق الدائري في القسم @sec:core-of-evaluator. بعد حساب قيمة التطبيق الأولي، نستعيد #py("continue") وننتقل إلى نقطة الإدخال المحددة.
#idx("primitiveapply", decl: true)
#snippet(```python
"primitive_apply",
  assign("val", list(op("apply_primitive_function"),
                     reg("fun"), reg("argl"))),
  restore("continue"),
  go_to(reg("continue")),
```)

تسلسل التعليمات المعنون بـ #py("compound_apply") يحدد تطبيق
#idx("explicit-control evaluator for Python", sub: "compound functions")
الدوال المركبة. لتطبيق دالة مركبة، نواصل بطريقة مماثلة لما فعلناه في المُقيِّم فوق الدائري. نبني إطارًا يربط معلمات الدالة بالوسائط، ونستخدم هذا الإطار لتوسيع البيئة المحمولة بواسطة الدالة، ونُقيّم جسم الدالة في هذه البيئة الممتدة.

في هذه النقطة تكون الدالة المركبة في المسجّل #py("fun") وتكون وسائطها في #py("argl"). نستخرج معلمات الدالة في #py("unev") وبيئتها في #py("env"). ثم نستبدل البيئة في #py("env") بالبيئة المبنية بتوسيعها بروابط المعلمات بالوسائط المعطاة. ثم نستخرج جسم الدالة في #py("comp").
ستكون الخطوة التالية الطبيعية هي استعادة #py("continue") المحفوظة والمواكبة إلى #py("eval_dispatch") لتقييم الجسم والانتقال إلى المواصلة المستعادة مع النتيجة في #py("val")، كما هو معمول به مع العبارة الأخيرة في تسلسل. ولكن هناك تعقيد!

يحتوي التعقيد على جانبين. أحدهما هو أنه عند أي نقطة في تقييم الجسم، قد تتطلب تعليمة
#idx("explicit-control evaluator for Python", sub: "return statements")
إرجاع من الدالة إرجاع قيمة تعبير الإرجاع كقيمة للجسم. ولكن قد تكون تعليمة الإرجاع متداخلة بشكل عشوائي عميق في الجسم؛ لذا فإن المكدس في اللحظة التي تظهر فيها تعليمة الإرجاع ليس بالضرورة هو المكدس المطلوب للإرجاع من الدالة. إحدى الطرق لجعل من الممكن تعديل المكدس للإرجاع هي وضع #emph[علامة] (#en[marker]) على المكدس يمكن العثور عليها بواسطة كود الإرجاع. يُنفذ هذا بواسطة التعليمة
#idx("register-machine language", sub: "pushmarkertostack")
#idx("pushmarkertostack (in register machine)")
#py("push_marker_to_stack"). يمكن لكود الإرجاع عندئذٍ استخدام التعليمة
#idx("register-machine language", sub: "revertstacktomarker")
#idx("revertstacktomarker (in register machine)")
#py("revert_stack_to_marker")
لاستعادة المكدس إلى المكان المشار إليه بواسطة العلامة قبل تقييم تعبير الإرجاع.#footnote[التعليمات الخاصة #py("push_marker_to_stack") و #py("revert_stack_to_marker") ليست ضرورية تمامًا وكان يمكن تنفيذها عن طريق دفع ودفع قيمة علامة صراحة على المكدس. أي شيء لا يمكن الخلط بينه وبين قيمة في البرنامج يمكن استخدامه كعلامة. انظر التمرين @ex:push_marker_to_stack1.]

الجانب الآخر من التعقيد هو أنه إذا انتهى تقييم الجسم دون تنفيذ تعليمة إرجاع، يجب أن تكون قيمة الجسم #py("undefined"). للتعامل مع هذا، نقوم بإعداد المسجّل #py("continue") ليقشير إلى نقطة الإدخال #py("return_undefined") قبل الانتقال إلى #py("eval_dispatch") لتقييم الجسم. إذا لم تظهر تعليمة إرجاع أثناء تقييم الجسم، فإن تقييم الجسم سيستمر عند #py("return_undefined").
#idx("compoundapply", decl: true)
#snippet(```python
"compound_apply",
  assign("unev", list(op("function_parameters"), reg("fun"))),
  assign("env", list(op("function_environment"), reg("fun"))),
  assign("env", list(op("extend_environment"),
                     reg("unev"), reg("argl"), reg("env"))),
  assign("comp", list(op("function_body"), reg("fun"))),
  push_marker_to_stack(),
  assign("continue", label("return_undefined")),
  go_to(label("eval_dispatch")),
```)

الأماكن الوحيدة في المُفسِّر التي يُسند فيها للمسجّل #py("env") قيمة جديدة هي #py("compound_apply") و #py("ev_block") (القسم @sec:block-assign-def-evaluation). تمامًا كما في المُقيِّم فوق الدائري، تُبنى البيئة الجديدة لتقييم جسم دالة من البيئة المحمولة بواسطة الدالة، جنبًا إلى جنب مع قائمة الوسائط والقائمة المقابلة من الأسماء المراد ربطها.

#idx("explicit-control evaluator for Python", sub: "function application")
#idx("explicit-control evaluator for Python", sub: "combinations")

عند تقييم تعليمة إرجاع عند #py("ev_return")، نستخدم التعليمة #py("revert_stack_to_marker") لاستعادة المكدس إلى حالته عند بداية استدعاء الدالة عن طريق إزالة جميع القيم من المكدس حتى العلامة وتضمينها. وكنتيجة لذلك، ستستعيد #py("restore(\"continue\")") مواصلة استدعاء الدالة، والتي حُفظت عند #py("ev_application"). نواصل عندئذٍ تقييم تعبير الإرجاع، وتوضع نتيجته في #py("val") وتكون بالتالي هي القيمة المُرجعة من الدالة عندما نواصل بعد تقييم تعبير الإرجاع.

#idx("evreturn", decl: true)
#snippet(```python
"ev_return",
  revert_stack_to_marker(),
  restore("continue"),
  assign("comp", list(op("return_expression"), reg("comp"))),
  go_to(label("eval_dispatch")),
```)

إذا لم تظهر تعليمة إرجاع أثناء تقييم جسم الدالة،
#idx("return value", sub: "undefined as")
فإن التقييم يستمر عند #py("return_undefined")، وهي المواصلة التي أُعدت عند #py("compound_apply"). لإرجاع #py("undefined") من الدالة، نضع #py("undefined") في #py("val") وننتقل إلى نقطة الإدخال التي وُضعت على المكدس عند #py("ev_application"). قبل أن نتمكن من استعادة تلك المواصلة من المكدس، ومع ذلك، يجب أن نزيل العلامة التي حُفظت عند #py("compound_apply").

#idx("returnundefined", decl: true)
#snippet(```python
"return_undefined",
  revert_stack_to_marker(),
  restore("continue"),
  assign("val", constant(undefined)),
  go_to(reg("continue")),
```)

#subheading([تعليمات الإرجاع والعودية الذيلية])

#idx("explicit-control evaluator for Python", sub: "tail recursion")

#idx("tail recursion", sub: "explicit-control evaluator and")
#idx("return statement", sub: "handling in explicit-control evaluator")

في الفصل @chap:fun قلنا إن العملية الموصوفة بواسطة دالة مثل:

#snippet(```python
function sqrt_iter(guess, x) {
    return is_good_enough(guess, x)
           ? guess
           : sqrt_iter(improve(guess, x), x);
}
```)

هي عملية تكرارية. على الرغم من أن الدالة عودية نحويًا (معرفة بدلالة نفسها)، إلا أنه ليس من الضروري منطقيًا أن يحفظ المُقيِّم معلومات عند الانتقال من استدعاء لـ #py("sqrt_iter") إلى الاستدعاء التالي.#footnote[رأينا في القسم @sec:designing-register-machines كيفية تنفيذ مثل هذه العملية بآلة مسجّلات ليس لها مكدس؛ حيث كانت حالة العملية مخزنة في مجموعة ثابتة من المسجّلات.] المُقيِّم الذي يمكنه تنفيذ دالة مثل #py("sqrt_iter") دون اشتراط زيادة التخزين مع مواصلة الدالة استدعاء نفسها يسمى مُقيِّمًا
#idx("tail-recursive evaluator")
#idx("metacircular evaluator for Python", sub: "tail recursion and")
#idx("tail recursion", sub: "metacircular evaluator and")
#idx("return statement", sub: "tail recursion and")
#emph[عوديًا ذيليًا] (#en[tail-recursive]).

التنفيذ فوق الدائري للمُقيِّم في الفصل @chap:meta ليس عوديًا ذيليًا. فهو ينفذ تعليمة الإرجاع كمُنشئ لكائن قيمة إرجاع يحتوي على القيمة المراد إرجاعها ويفحص نتيجة استدعاء دالة لمعرفة ما إذا كانت مثل هذا الكائن. إذا أدى تقييم جسم دالة إلى كائن قيمة إرجاع، فإن القيمة المُرجعة للدالة هي محتويات ذلك الكائن؛ بخلاف ذلك تكون القيمة المُرجعة #py("undefined"). كلا البنائين لكائن قيمة الإرجاع والفحص النهائي لنتيجة استدعاء الدالة هما عمليات مؤجلة، تؤدي إلى تراكم المعلومات على المكدس.

مُقيِّم التحكم الصريح الخاص بنا #emph[هو] عودي ذيلي، لأنه لا يحتاج إلى تغليف قيم الإرجاع للفحص وبالتالي يتجنب تراكم المكدس من العمليات المؤجلة. عند #py("ev_return")، ومن أجل تقييم التعبير الذي يحسب القيمة المُرجعة لدالة، ننقل مباشرة إلى #py("eval_dispatch") مع عدم وجود شيء على المكدس أكثر مما كان موجودًا قبل استدعاء الدالة مباشرة. ونحقق ذلك بالتراجع عن أي حفظ للمكدس بواسطة الدالة (والتي هي عديمة الفائدة لأننا نعود) باستخدام #py("revert_stack_to_marker"). ثم، بدلاً من الترتيب لـ #py("eval_dispatch") ليعود إلى هنا و#emph[ثم] استعادة #py("continue") من المكدس والتنفيذ عند نقطة الإدخال تلك، نستعيد #py("continue") من المكدس #emph[قبل] الانتقال إلى #py("eval_dispatch") بحيث يواصل #py("eval_dispatch") عند نقطة الإدخال تلك بعد تقييم التعبير. وأخيرًا، ننقل إلى #py("eval_dispatch") دون حفظ أي معلومات على المكدس. وبالتالي، عندما نواصل لتقييم تعبير إرجاع، فإن المكدس هو نفسه كما كان قبل الاستدعاء للدالة التي نحن على وشك حساب قيمتها المُرجعة مباشرة. وبالتالي، فإن تقييم تعبير إرجاع—حتى لو كان استدعاء دالة (كما في #py("sqrt_iter")، حيث يختزل التعبير الشرطي إلى استدعاء لـ #py("sqrt_iter"))—لن يتسبب في تراكم أي معلومات على المكدس.#footnote[هذا التنفيذ للعودية الذيلية هو نوع من تقنية تحسين معروفة جيدا تستخدمها العديد من المُترجِمات. في ترجمة دالة تنتهي باستدعاء دالة، يمكن للمرء استبدال الاستدعاء بفرز قفز إلى نقطة إدخال الدالة المستدعاة. بناء هذه الاستراتيجية في المُفسِّر، كما فعلنا في هذا القسم، يوفر التحسين بانتظام في جميع أنحاء اللغة.]

إذا لم نفكر في الاستفادة من حقيقة أنه ليس من الضروري الاحتفاظ بالمعلومات غير المفيدة على المكدس أثناء تقييم تعبير إرجاع، فقد نكون اتخذنا النهج المباشر لتقييم تعبير الإرجاع، والعودة لاستعادة المكدس، وأخيراً الاستمرار عند نقطة الإدخال التي تنتظر نتيجة استدعاء الدالة:

#syntax("
\"ev_return\",  // تنفيذ بديل: ليس عودياً ذيلياً
  assign(\"comp\", list(op(\"return_expression\"), reg(\"comp\"))),
  assign(\"continue\", label(\"ev_restore_stack\")),
  go_to(label(\"eval_dispatch\")),
\"ev_restore_stack\",
  revert_stack_to_marker(),    // إلغاء عمليات الحفظ في الدالة الحالية
  restore(\"continue\"),         // إلغاء عمليات الحفظ عند ", $mono("ev_application")$, "
  go_to(reg(\"continue\")),
          ")

قد يبدو هذا تغييراً طفيفاً على الكود السابق الخاص بنا لتقييم تعليمات الإرجاع:
الفرق الوحيد هو أننا نؤجل إلغاء أي عمليات حفظ مسجّلات على المكدس إلى ما بعد تقييم تعبير الإرجاع.

سيظل المُفسِّر يعطي نفس القيمة لأي تعبير. ولكن هذا التغيير كارثي للتنفيذ العودي الذيلية، لأننا يجب أن نعود الآن بعد تقييم تعبير الإرجاع من أجل إلغاء عمليات حفظ المسجّلات (غير المفيدة).

ستتراكم عمليات الحفظ الإضافية هذه أثناء تداخل استدعاءات الدوال.

وكنتيجة لذلك، فإن العمليات مثل #py("sqrt_iter") ستتطلب مساحة تناسبية مع عدد التكرارات بدلاً من طلب مساحة ثابتة.

يمكن أن يكون هذا الفرق كبيراً. على سبيل المثال،
#idx("iterative process", sub: "implemented by function call")
مع العودية الذيلية، يمكن التعبير عن حلقة لا نهائية باستخدام آليات استدعاء الدالة والإرجاع فقط:

#snippet(```python
function count(n) {
    display(n);
    return count(n + 1);
}
```)

بدون العودية الذيلية، فإن مثل هذه الدالة ستنفد في النهاية من مساحة المكدس، والتعبير عن تكرار حقيقي يتطلب آلية تحكم أخرى غير استدعاء الدالة.

لاحظ أن تنفيذ بايثون الخاص بنا يتطلب استخدام
#idx("tail recursion", sub: "return statement necessary for")
#py("return") لكي يكون عودياً ذيلياً.
لأن إلغاء عمليات حفظ المسجّلات يتم عند #py("ev_return")، فإن إزالة #py("return") من دالة #py("count") أعلاه سيتسبب في نفادها في النهاية من مساحة المكدس. يفسر هذا استخدام #py("return") في حلقات المشغل اللانهائية في الفصل @chap:meta.

#idx("explicit-control evaluator for Python", sub: "return statements")

#exercise(label-name: <ex:missing-return>, [
اشرح كيفية تراكم المكدس
#idx("explicit-control evaluator for Python", sub: "tail recursion")
#idx("tail recursion", sub: "explicit-control evaluator and")
إذا تم إزالة #py("return") من #py("count"):

#snippet(```python
function count(n) {
    display(n);
    count(n + 1);
}
```)
])

#exercise(label-name: <ex:push_marker_to_stack1>, [
نفذ ما يعادل #py("push_marker_to_stack") باستخدام #py("save") عند #py("compound_apply") لتخزين قيمة علامة خاصة على المكدس. نفذ ما يعادل #py("revert_stack_to_marker") عند #py("ev_return") و #py("return_undefined") كحلقة تنفذ #py("restore") بشكل متكرر حتى تضرب العلامة. لاحظ أن هذا سيتطلب استعادة قيمة إلى مسجّل آخر غير الذي حُفظت منه. (على الرغم من أننا نحرص على تجنب ذلك في مُقيِّمنا، فإن تنفيذ المكدس لدينا يسمح بذلك في الواقع. انظر التمرين @ex:stack-behavior).
هذا ضروري لأن الطريقة الوحيدة للاستعادة من المكدس هي الاستعادة إلى مسجّل.

تلميح: ستحتاج إلى إنشاء ثابت فريد ليخدم كعلامة، على سبيل المثال مع #py("const marker = list(\"marker\")").
نظرًا لأن #py("list") تنشئ زوجًا جديدًا، فلا يمكن أن تساهم #py("===") مع أي شيء آخر على المكدس.
])

#exercise(label-name: <ex:push_marker_to_stack2>, [
نفذ
#idx("register-machine language", sub: "pushmarkertostack")
#idx("pushmarkertostack (in register machine)")
#py("push_marker_to_stack") و
#idx("register-machine language", sub: "revertstacktomarker")
#idx("revertstacktomarker (in register machine)")
#py("revert_stack_to_marker") كتعليمات آلة مسجّلات، متبعًا تنفيذ #py("save") و #py("restore") في القسم @sec:ex-proc. أضف الدالتين #py("push_marker") و #py("pop_marker") للوصول إلى المكدسات، محاكيًا تنفيذ #py("push") و #py("pop") في القسم @sec:machine-model. لاحظ أنك لا تحتاج إلى إدراج علامة فعلية في المكدس. بدلاً من ذلك، يمكنك إضافة متغير حالة محلي إلى نموذج المكدس لتتبع موقع آخر #py("save") قبل كل #py("push_marker_to_stack").
إذا اخترت وضع علامة على المكدس، فانظر التلميح في التمرين @ex:push_marker_to_stack1.
])

#idx("tail recursion", sub: "explicit-control evaluator and")

#idx("explicit-control evaluator for Python", sub: "tail recursion")
