// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([بنية المُترجِم], label-name: <sec:compiler-structure>)

#idx("compiler for Python", sub: "structure of")

في القسم @sec:separating-analysis عدلنا مُفسِّرنا فوق الدائري الأصلي لفصل
#idx("compiler for Python", sub: "analyzing evaluator vs.")
التحليل عن التنفيذ. حللنا كل مكون لإنتاج دالة تنفيذ تأخذ بيئة كوسيط وتنفذ العمليات المطلوبة.
في مُترجِمنا، سنقوم بالتحليل نفسه بالأساس. ولكن بدلاً من إنتاج دوال تنفيذ، سنولد تسلسلات من التعليمات ليتم تشغيلها بواسطة آلة المسجّلات الخاصة بنا.

الدالة #py("compile") هي التوجيه الأعلى مستوى في المُترجِم. وهي تناظر الدالة #py("evaluate") في القسم @sec:core-of-evaluator، والدالة #py("analyze") في القسم @sec:separating-analysis، ونقطة الإدخال #py("eval_dispatch") لمُقيِّم التحكم الصريح في القسم @sec:eceval-core. يستخدم المُترجِم، مثل المُفسِّرات،
#idx("compiler for Python", sub: "expression-syntax functions")
دوال بناء المكونات المعرفة في القسم @sec:representing-expressions.#footnote[لاحظ، ومع ذلك، أن مُترجِمنا هو برنامج #en[Python]، ودوال البناء التي يستخدمها لمعالجة التعبيرات هي دوال #en[Python] الفعلية المستخدمة مع المُقيِّم فوق الدائري (#en[metacircular]). أما بالنسبة لمُقيِّم التحكم الصريح، على النقيض من ذلك، ففترضنا أن عمليات البناء المكافئة كانت متاحة كعمليات لآلة المسجّلات. (بالطبع، عندما حاطينا آلة المسجّلات في #en[Python]، استخدمنا دوال #en[Python] الفعلية في محاكاة آلة المسجّلات).]
تنفذ الدالة #py("compile") تحليلاً للحالات على النوع النحوي للمكون المراد ترجمته. ولكل نوع من المكونات، توجه إلى
#idx("code generator")
#emph[مولد كود] متخصص:
#idx("compile", decl: true)
#snippet(```python
function compile(component, target, linkage) {
    return is_literal(component)
           ? compile_literal(component, target, linkage)
           : is_name(component)
           ? compile_name(component, target, linkage)
           : is_application(component)
           ? compile_application(component, target, linkage)
           : is_operator_combination(component)
           ? compile(operator_combination_to_application(component),
                     target, linkage)
           : is_conditional(component)
           ? compile_conditional(component, target, linkage)
           : is_lambda_expression(component)
           ? compile_lambda_expression(component, target, linkage)
           : is_sequence(component)
           ? compile_sequence(sequence_statements(component),
                               target, linkage)
           : is_block(component)
           ? compile_block(component, target, linkage)
           : is_return_statement(component)
           ? compile_return_statement(component, target, linkage)
           : is_function_definition(component)
           ? compile(function_decl_to_constant_decl(component),
                     target, linkage)
           : is_declaration(component)
           ? compile_declaration(component, target, linkage)
           : is_assignment(component)
           ? compile_assignment(component, target, linkage)
           : error(component, "unknown component type -- compile");
}
```)

#subheading([الأهداف والارتباطات])

تأخذ الدالة #py("compile") ومولدات الكود التي تستدعيها وسيطين
#idx("code generator", sub: "arguments of")
بالإضافة إلى المكون المراد ترجمته. هناك
#idx("target register")
#emph[هدف] (#en[target])، والذي يحدد المسجّل الذي يجب أن يُرجع فيه الكود المُترجَم قيمة المكون.
وهناك أيضًا
#idx("linkage descriptor")
#emph[واصف ارتباط] (#en[linkage descriptor])، والذي يصف كيف يجب أن يواصل الكود الناتج عن ترجمة المكون بعد انتهائه من التنفيذ. يمكن لواصف الارتباط أن يتطلب من الكود القيام بآحد الأشياء الثلاثة التالية:

- المواصلة إلى التعليمة التالية في التسلسل (يحدد هذا بواسطة واصف الارتباط #idx("next (linkage descriptor)") #py("\"next\""))،
- القفز إلى القيمة الحالية للمسجّل #py("continue") كجزء من الإرجاع من استدعاء دالة (يحدد هذا بواسطة واصف الارتباط #idx("return (linkage descriptor)", sort: "return") #py("\"return\""))، أو
- القفز إلى نقطة إدخال معنونة (يحدد هذا باستخدام التسمية المحددة كواصف ارتباط).

على سبيل المثال، ترجمة التعبير الصريح #py("5") مع هدف المسجّل #py("val") وارتباط #py("\"next\"") ينبغي أن تُنتج التعليمة:

#snippet(```python
assign("val", constant(5))
```)

وترجمة نفس التعبير مع ارتباط #py("\"return\"") ينبغي أن تُنتج التعليمات:

#snippet(```python
assign("val", constant(5)),
go_to(reg("continue"))
```)

في الحالة الأولى، سيستمر التنفيذ مع التعليمة التالية في التسلسل. في الحالة الثانية، سنقفز إلى أي نقطة إدخال مخزنة في المسجّل #py("continue"). في كلا الحالتين، ستوضع قيمة التعبير في المسجّل المستهدف #py("val").

يستخدم مُترجِمنا الارتباط #py("\"return\"") عند ترجمة تعبير الإرجاع لتعليمة الإرجاع.
تمامًا كما في مُقيِّم التحكم الصريح، يحدث الإرجاع من استدعاء دالة في ثلاث خطوات:

+ التراجع عن المكدس حتى العلامة واستعادة #py("continue") (التي تحمل مواصلة أُعدت عند بداية استدعاء الدالة)
+ حساب قيمة الإرجاع ووضعها في #py("val")
+ القفز إلى نقطة الإدخال في #py("continue")

ترجمة تعليمة الإرجاع تُولد صراحة كودًا للتراجع عن المكدس واستعادة #py("continue").
يُترجم تعبير الإرجاع مع الهدف #py("val") والارتباط #py("\"return\"") بحيث يضع الكود الناتج لحساب قيمة الإرجاع قيمة الإرجاع في #py("val") وينتهي بالقفز إلى #py("continue").

#subheading([تسلسلات التعليمات واستخدام المكدس])

#anchor(<sec:instruction-sequences>)

#idx("instruction sequence")

يُرجع كل مولد كود
#idx("code generator", sub: "value of")
#emph[تسلسل تعليمات] (#en[instruction sequence]) يحتوي على الكود الهدف الذي وُلِّد للمكون.
تُولَّد الكود للمكون المركب بتجميع المخرجات من مولدات الكود الأبسط للمكونات الفرعية، تمامًا كما يُنجَز تقييم المكون المركب بتقييم المكونات الفرعية.

أبسط طريقة لتجميع تسلسلات التعليمات هي دالة تسمى
#idx("appendinstructionsequences")
#py("append_instruction_sequences")،
والتي تأخذ كوسائط تسلسلي تعليمات مراد تنفيذهما تتابعيًا. وتجمعهما وتُرجع التسلسل المركب.
أي أنه إذا كان $s e q_(1)$ و $s e q_(2)$ تسلسلين من التعليمات، فإن تقييم:

#syntax("
      append_instruction_sequences(", meta("seq"), $""_(1)$, ", ", meta("seq"), $""_(2)$, ")
      ")

يُنتج التسلسل:

#syntax(meta("seq"), $""_(1)$, "
", meta("seq"), $""_(2)$)

كلما قد تلزم الحاجة إلى حفظ المسجّلات، تستخدم مولدات كود المُترجِم
#idx("preserving")
#py("preserving")، وهي طريقة أكثر دقة لتجميع تسلسلات التعليمات.
تأخذ الدالة #py("preserving") ثلاثة وسائط: مجموعة من المسجّلات وتسلسلي تعليمات مراد تنفيذهما تتابعيًا. وتجمع التسلسلات بطريقة تُحفظ بها محتويات كل مسجّل في المجموعة عبر تنفيذ التسلسل الأول، إذا كان ذلك مطلوبًا لتنفيذ التسلسل الثاني. أي أنه إذا كان التسلسل الأول يعدل المسجّل وكان التسلسل الثاني يحتاج فعليًا إلى المحتويات الأصلية للمسجّل، فإن #py("preserving") تغلف تعليمة #py("save") وتعليمة #py("restore") للمسجّل حول التسلسل الأول قبل تجميع التسلسلات. بخلاف ذلك، تُرجع #py("preserving") ببساطة تسلسلات التعليمات المجمعة. وبالتالي، على سبيل المثال:

#syntax("
      preserving(list(", meta("reg"), $""_(1)$, ", ", meta("reg"), $""_(2)$, "), ", meta("seq"), $""_(1)$, ", ", meta("seq"), $""_(2)$, ")
      ")

تُنتج أحد تسلسلات التعليمات الأربعة التالية، اعتمادًا على كيفية استخدام #meta("seq")$""_(1)$ و #meta("seq")$""_(2)$ لـ #meta("reg")$""_(1)$ و #meta("reg")$""_(2)$:

$ mat(delim: #none, italic("seq")_(1), mono("save(")italic("reg")_(1)mono("),"), mono("save(")italic("reg")_(2)mono("),"), mono("save(")italic("reg")_(2)mono("),"); italic("seq")_(2), italic("seq")_(1), italic("seq")_(1), mono("save(")italic("reg")_(1)mono("),"); , mono("restore(")italic("reg")_(1)mono("),"), mono("restore(")italic("reg")_(2)mono("),"), italic("seq")_(1); , italic("seq")_(2), italic("seq")_(2), mono("restore(")italic("reg")_(1)mono("),"); , , , mono("restore(")italic("reg")_(2)mono("),"); , , , italic("seq")_(2)) $

باستخدام #py("preserving") لتجميع تسلسلات التعليمات، يتجنب المُترجِم
#idx("compiler for Python", sub: "stack usage")
عمليات المكدس غير الضرورية. كما يريح تفاصيل ما إذا كان سيتم توليد تعليمات #py("save") و #py("restore") أم لا داخل الدالة #py("preserving")، فاصلاً إياها عن الاهتمامات التي تظهر في كتابة كل مولد كود فردي.
في الواقع لا تُنتج أي تعليمات #py("save") أو #py("restore") صراحة بواسطة مولدات الكود، باستثناء أن الكود الخاص باستدعاء دالة يحفظ #py("continue") والكود الخاص بالإرجاع من دالة يستعيدها: تُولد تعليمات #py("save") و #py("restore") المقابلة هذه صراحة بواسطة استدعاءات مختلفة لـ #py("compile")، وليس كزوج متطابق بواسطة #py("preserving") (كما سنرى في القسم @sec:compiling-combinations).

من حيث المبدأ، كان بإمكاننا تمثيل تسلسل التعليمات ببساطة كقائمة من التعليمات.
وكان يمكن للدالة #py("append_instruction_sequences") عندئذٍ تجميع تسلسلات التعليمات عن طريق إجراء #py("append") عادي للقوائم. ومع ذلك، ستكون #py("preserving") عندئذٍ عملية معقدة، لأنها ستضطر إلى تحليل كل تسلسل تعليمات لتحديد كيفية استخدام التسلسل لمسجّلاته.
وستكون الدالة #py("preserving") غير كفءة بالإضافة إلى كونها معقدة، لأنها ستضطر إلى تحليل كل وسيط تسلسل تعليمات لديها، حتى لو كانت هذه التسلسلات قد بُنيت نفسها باستدعاءات لـ #py("preserving")، وفي هذه الحالة تكون أجزاؤها قد حُللت بالفعل. لتجنب مثل هذا التحليل المكرر، سنقرن بكل تسلسل تعليمات بعض المعلومات حول استخدام المسجّلات الخاص به. عندما نبني تسلسل تعليمات أساسيًا، سنقدم هذه المعلومات صراحة، والدوال التي تجمع تسلسلات التعليمات ستشتق معلومات استخدام المسجّلات للتسلسل المركب من المعلومات المقترنة بالتسلسلات التي يتم تجميعها.

سيحتوي تسلسل التعليمات على ثلاث قطع من المعلومات:

- مجموعة المسجّلات التي يجب تهيئتها قبل تنفيذ التعليمات في التسلسل (تسمى هذه المسجّلات #emph[مطلوبة] بواسطة التسلسل)،
- مجموعة المسجّلات التي تُعدل قيمها بواسطة التعليمات في التسلسل، و
- التعليمات الفعلية في التسلسل.

سنمثل تسلسل التعليمات كقائمة من أجزائه الثلاثة. وبالتالي فإن مُنشئ تسلسلات التعليمات هو:
#idx("makeinstructionsequence", decl: true)

#snippet(```python
function make_instruction_sequence(needs, modifies, instructions) {
    return list(needs, modifies, instructions);
}
```)

على سبيل المثال، فإن تسلسل التعليمات المكون من تعليمتين والذي يبحث عن قيمة الرمز #py("\"x\"") في البيئة الحالية، ويسند النتيجة إلى #py("val")، ثم يواصل إلى المواصلة، يتطلب تهيئة المسجّلين #py("env") و #py("continue")، ويعدل المسجّل #py("val").
وبالتالي يُبنى هذا التسلسل كالتالي:

#snippet(```python
make_instruction_sequence(list("env", "continue"), list("val"),
    list(assign("val",
                list(op("lookup_symbol_value"), constant("x"),
                     reg("env"))),
         go_to(reg("continue"))));
```)

الدوال المخصصة لتجميع تسلسلات التعليمات موضحة في القسم @sec:combining-instruction-sequences.

#idx("instruction sequence")
#idx("compiler for Python", sub: "structure of")

#exercise(label-name: <ex:comp-optimize>, [
في تقييم تطبيق
#idx("compiler for Python", sub: "stack usage")
#idx("preserving")
دالة، يحفظ مُقيِّم التحكم الصريح دائمًا ويستعيد المسجّل #py("env") حول تقييم تعبير الدالة، ويحفظ ويستعيد #py("env") حول تقييم كل تعبير وسيط (باستثناء الأخير)، ويحفظ ويستعيد #py("argl") حول تقييم كل تعبير وسيط، ويحفظ ويستعيد #py("fun") حول تقييم تسلسل تعبيرات الوسائط. لكل من التطبيقات التالية، اذكر أي من عمليات #py("save") و #py("restore") هذه زائدة عن الحاجة وبالتالي كان يمكن إلغاؤها بواسطة آلية #py("preserving") للمُترجِم:

#snippet(```python
f("x", "y")

f()("x", "y")

f(g("x"), y)

f(g("x"), "y")
```)
])

#exercise(label-name: <ex:5_33>, [
باستخدام آلية #py("preserving")، سيتجنب المُترجِم حفظ واستعادة #py("env") حول تقييم تعبير الدالة لتطبيق ما في الحالة التي يكون فيها تعبير الدالة اسمًا.
وكان بإمكاننا أيضًا بناء مثل هذه التحسينات في المُقيِّم.
في الواقع، فإن مُقيِّم التحكم الصريح في القسم @sec:eceval يجري بالفعل تحسينًا مماثلاً، بمعاملة التطبيقات بدون وسائط كحالة خاصة.

+ وسّع مُقيِّم التحكم الصريح ليتعرف كفئة متميزة من المكونات على التطبيقات التي يكون تعبير دالتها اسمًا، ولليستفيد من هذه الحقيقة في تقييم مثل هذه المكونات.
+ تقترح أليسا ب. هاكر أنه بتوسيع المُقيِّم للتعرف على المزيد والمزيد من الحالات الخاصة يمكننا تضمين جميع تحسينات المُترجِم، وأن هذا سيلغي ميزة الترجمة كليًا. ما رأيك في هذه الفكرة؟
])
