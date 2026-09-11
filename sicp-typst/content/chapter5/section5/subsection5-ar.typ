// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([مثال على الكود المُترجَم], label-name: <sec:compiled-code>)

#idx("compiler for Python", sub: "example compilation")
#idx("factorial", sub: "compilation of")

الآن وقد رأينا جميع عناصر المُترجِم، دعنا نفحص مثالاً للكود المُترجَم لنرى كيفية توافق الأشياء معًا. سنترجم إعلان دالة #py("factorial") العودية عن طريق تمرير نتيجة تطبيق #py("parse") على تمثيل سلسلي للبرنامج كوسيط أول لـ #py("compile") (هنا باستخدام علامات الباك تك #py("`")$dots.h$#py("`")، والتي تعمل مثل علامات التنصيص الفردية والمزدوجة ولكنها تسمح للسلسلة بالامتداد عبر أسطر متعددة):

#snippet(```python
compile(parse(`
function factorial(n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
              `),
        "val",
        "next");
```)

لقد حددنا أن قيمة الإعلان يجب أن توضع في المسجّل #py("val").
ولا نهتم بما يفعله الكود المُترجَم بعد تنفيذ الإعلان، لذا فإن اختيارنا لـ #py("\"next\"") كواصف ارتباط هو أمر عشوائي.

تحدد الدالة #py("compile") أنها أُعطيت تعريف دالة، لذا تحوله إلى إعلان ثابت ثم تستدعي #py("compile_declaration"). هذا يترجم كودًا لحساب القيمة المراد إسنادها (موجهًا إلى #py("val"))، متبوعًا بكود لتثبيت الإعلان، متبوعًا بكود لوضع قيمة الإعلان (وهي القيمة #py("undefined")) في المسجّل المستهدف، متبوعًا أخيرًا بكود الارتباط.
يُحفظ المسجّل #py("env") حول حساب القيمة، لأنه يلزم لتثبيت الإعلان.
ولأن الارتباط هو #py("\"next\"")، فلا يوجد كود ارتباط في هذه الحالة. وبالتالي فإن هيكل الكود المُترجَم هو:

#syntax(metaphrase[حفظ #py("env") إذا تم تعديله بواسطة كود حساب القيمة], "
", metaphrase[ترجمة قيمة الإعلان، الهدف #py("val")، الارتباط #py("\"next\"")], "
", metaphrase[استعادة #py("env") إذا تم حفظه أعلاه], "
perform(list(op(\"assign_symbol_value\"),
             constant(\"factorial\"),
             reg(\"val\"),
             reg(\"env\"))),
assign(\"val\", constant(undefined))
      ")

التعبير الذي يتم ترجمته لإنتاج القيمة للاسم #py("factorial") هو تعبير لامدا قيمته هي الدالة التي تحسب المضروبات.
تتعامل الدالة #py("compile") مع هذا باستدعاء #py("compile_lambda_expression")، والتي تترجم جسم الدالة، وتظلله كنقطة إدخال جديدة، وتولد التعليمة التي ستدمج جسم الدالة عند نقطة الإدخال الجديدة مع بيئة وقت التشغيل وتسند النتيجة إلى #py("val"). ثم يتخطي التسلسل حول كود الدالة المُترجَم، والذي يُدرج عند هذه النقطة. يبدأ كود الدالة نفسه بتوسيع بيئة إعلان الدالة بإطار يربط المعلمة #py("n") بوسيط الدالة. ثم يأتي جسم الدالة الفعلي. نظرًا لأن هذا الكود لقيمة الاسم لا يعدل المسجّل #py("env")، فإن عمليتي الحفظ والاستعادة الاختياريتين الموضحتين أعلاه لا تُولدان. (الكود الخاص بالدالة عند #py("entry1") لا يُنفذ عند هذه النقطة، لذا فإن استخدامه لـ #py("env") غير ذي صلة).
لذلك، فإن الهيكل للكود المُترجَم يصبح:

#syntax($mono(" ")mono(" ")$, "assign(\"val\", list(op(\"make_compiled_function\"),
                     label(\"entry1\"),
                     reg(\"env\"))),
  go_to(label(\"after_lambda2\")),
\"entry1\",
  assign(\"env\", list(op(\"compiled_function_env\"), reg(\"fun\"))),
  assign(\"env\", list(op(\"extend_environment\"),
                     constant(list(\"n\")),
                     reg(\"argl\"),
                     reg(\"env\"))),
  ", metaphrase[ترجمة جسم الدالة], "
\"after_lambda2\",
  perform(list(op(\"assign_symbol_value\"),
               constant(\"factorial\"),
               reg(\"val\"),
               reg(\"env\"))),
  assign(\"val\", constant(undefined))
      ")

جسم الدالة يُترجم دائمًا (بواسطة #py("compile_lambda_body")) مع الهدف #py("val") والارتباط #py("\"next\"").
ويتكون الجسم في هذه الحالة من تعليمة إرجاع واحدة:#footnote[بسبب #py("append_return_undefined") في #py("compile_lambda_body")، فإن الجسم يتكون في الواقع من تسلسل مع تعليمتي إرجاع. ومع ذلك، فإن فحص الكود الميت في #py("compile_sequence") سيتوقف بعد ترجمة تعليمة الإرجاع الأولى، لذا فإن الجسم يتكون فعليًا من تعليمة إرجاع واحدة فقط.]

#snippet(```python
return n === 1
       ? 1
       : factorial(n - 1) * n;
```)

تولد الدالة #py("compile_return_statement") كودًا للتراجع عن المكدس باستخدام العلامة واستعادة المسجّل #py("continue")، ثم تترجم تعبير الإرجاع مع الهدف #py("val") والارتباط #py("\"return\"")، لأن قيمته هي المراد إرجاعها من الدالة.

تعبير الإرجاع هو تعبير شرطي، تولد له الدالة #py("compile_conditional") كودًا يحسب أولاً المحمول (الموجه إلى #py("val"))، ثم يفحص النتيجة ويتفرع حول فرع الصحة إذا كان المحمول خاطئاً.
يُحفظ المسجّلان #py("env") و #py("continue") حول كود المحمول، لأنهما قد يلزمان لباقي التعبير الشرطي.

يُترجم كلا فرعي الصحة والخطأ مع الهدف #py("val") والارتباط #py("\"return\"").
(أي أن قيمة الشرطي، وهي القيمة المحسوبة بواسطة أي من فرعيه، هي قيمة الدالة.)

#syntax($mono(" ")mono(" ")$, "revert_stack_to_marker(),
  restore(\"continue\"),
  ", metaphrase[حفظ #py("continue") و #py("env") إذا تم تعديلهما بواسطة المحمول وطلبهما الفروع], "
  ", metaphrase[ترجمة المحمول، الهدف #py("val")، الارتباط #py("\"next\"")], "
  ", metaphrase[استعادة #py("continue") و #py("env") إذا تم حفظهما أعلاه], "
  test(list(op(\"is_falsy\"), reg(\"val\"))),
  branch(label(\"false_branch4\")),
\"true_branch3\",
  ", metaphrase[ترجمة فرع الصحة، الهدف #py("val")، الارتباط #py("\"return\"")], "
\"false_branch4\",
  ", metaphrase[ترجمة فرع الخطأ، الهدف #py("val")، الارتباط #py("\"return\"")], "
\"after_cond5\",
      ")

المحمول #py("n === 1") هو تطبيق دالة (بعد تحويل تركيب المُعامل).
هذا يبحث عن تعبير الدالة (الرمز #py("\"===\"")) ويضع هذه القيمة في #py("fun").
ثم يجمع الوسيط #py("1") وقيمة #py("n") في #py("argl").
ثم يختبر ما إذا كانت #py("fun") تحتوي على دالة أوّلية أم دالة مركبة، ويوجه إلى فرع أوّلي أو فرع مركب بناءً على ذلك.
يستأنف كلا الفرعين عند التسمية #py("after_call").
يجب على الفرع المركب إعداد #py("continue") للقفز إلى ما بعد الفرع الأوّلي ودفع علامة إلى المكدس لمطابقة عملية التراجع في تعليمة الإرجاع المُترجَمة للدالة.
متطلبات حفظ المسجّلات حول تقييم تعبيرات الدالة والوسائط لا تتسبب في أي حفظ للمسجّلات، لأن التقييمات في هذه الحالة لا تعدل المسجّلات المعنية.

#syntax($mono(" ")mono(" ")$, "assign(\"fun\", list(op(\"lookup_symbol_value\"),
                     constant(\"===\"), reg(\"env\"))),
  assign(\"val\", constant(1)),
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  assign(\"val\", list(op(\"lookup_symbol_value\"),
                     constant(\"n\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch6\")),
\"compiled_branch7\",
  assign(\"continue\", label(\"after_call8\")),
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch6\",
  assign(\"val\", list(op(\"apply_primitive_function\"),
                     reg(\"fun\"),
                     reg(\"argl\"))),
\"after_call8\",
      ")

فرع الصحة، وهو الثابت 1، يُترجم (مع الهدف #py("val") والارتباط #py("\"return\"")) إلى:

#syntax($mono(" ")mono(" ")$, "assign(\"val\", constant(1)),
  go_to(reg(\"continue\")),
      ")

الكود لفرع الخطأ هو استدعاء دالة آخر، حيث الدالة هي قيمة الرمز #py("\"*\"")، والوسائط هي #py("n") ونتيجة استدعاء دالة آخر (استدعاء لـ #py("factorial")).
كل من هذه الاستدعاءات يقيم #py("fun") و #py("argl") وفرعيه الأوّلي والمركب الخاصين به. يُظهر الشكل @fig:comp-factorial1 الترجمة الكاملة لإعلان دالة #py("factorial").
لاحظ أن الـ #py("save") والـ #py("restore") المحتملين لـ #py("continue") و #py("env") حول المحمول، الموضحين أعلاه، يُولدان في الواقع، لأن هذين المسجّلين يُعدلان بواسطة استدعاء الدالة في المحمول ويُلزمان لاستدعاء الدالة وارتباط #py("\"return\"") في الفروع.

#sicp-figure([#syntax("
// بناء الدالة والتخطي حول الكود لجسم الدالة
  assign(\"val\", list(op(\"make_compiled_function\"),
                     label(\"entry1\"), reg(\"env\"))),
  go_to(label(\"after_lambda2\")),
\"entry1\",                           // الاستدعاءات لـ ", $mono("factorial")$, " ستدخل هنا
  assign(\"env\", list(op(\"compiled_function_env\"), reg(\"fun\"))),
  assign(\"env\", list(op(\"extend_environment\"), constant(list(\"n\")),
                     reg(\"argl\"), reg(\"env\"))),
// بداية جسم الدالة الفعلي
  revert_stack_to_marker(),         // يبدأ بتعليمة إرجاع
  restore(\"continue\"),
  save(\"continue\"),                 // حفظ المسجّلات عبر المحمول
  save(\"env\"),
// حساب ", $mono("n === 1")$, "
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"===\"), reg(\"env\"))),
  assign(\"val\", constant(1)),
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  assign(\"val\", list(op(\"lookup_symbol_value\"), constant(\"n\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch6\")),
\"compiled_branch7\",
  assign(\"continue\", label(\"after_call8\")),
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch6\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
\"after_call8\",                      // ", $mono("val")$, " يحتوي الآن على نتيجة ", $mono("n === 1")$, "
  restore(\"env\"),
  restore(\"continue\"),
  test(list(op(\"is_falsy\"), reg(\"val\"))),
  branch(label(\"false_branch4\")),
\"true_branch3\",                     // return 1
  assign(\"val\", constant(1)),
  go_to(reg(\"continue\")),
\"false_branch4\",
// حساب وإرجاع ", $mono("factorial(n - 1) * n")$, "
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"*\"), reg(\"env\"))),
  save(\"continue\"),
  save(\"fun\"),                      // حفظ دالة ", $mono("*")$, "
  assign(\"val\", list(op(\"lookup_symbol_value\"), constant(\"n\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  save(\"argl\"),                     // حفظ قائمة الوسائط الجزئية لـ ", $mono("*")$, "
// حساب ", $mono("factorial(n - 1)")$, " والذي هو الوسيط الآخر لـ ", $mono("*")$, "
  assign(\"fun\", list(op(\"lookup_symbol_value\"),
                     constant(\"factorial\"), reg(\"env\"))),
  save(\"fun\"),                      // حفظ دالة ", $mono("factorial")$, "
    ")], caption: [ترجمة إعلان دالة #py("factorial") (متابعة في الصفحة التالية).], label-name: <fig:comp-factorial1>)

#sicp-figure([#syntax("
// حساب ", $mono("n - 1")$, " والذي هو الوسيط لـ ", $mono("factorial")$, "
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"-\"), reg(\"env\"))),
  assign(\"val\", constant(1)),
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  assign(\"val\", list(op(\"lookup_symbol_value\"), constant(\"n\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch10\")),
\"compiled_branch11\",
  assign(\"continue\", label(\"after_call12\")),
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch10\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
\"after_call12\",                     // ", $mono("val")$, " يحتوي الآن على نتيجة ", $mono("n - 1")$, "
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  restore(\"fun\"),                   // استعادة ", $mono("factorial")$, "
// تطبيق ", $mono("factorial")$, "
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch14\")),
\"compiled_branch15\",
  assign(\"continue\", label(\"after_call16\")),
  save(\"continue\"),                 // الإعداد للدالة المُترجَمة $-$
  push_marker_to_stack(),           //   الإرجاع في الدالة سيعيد المكدس
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch14\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
\"after_call16\",                     // ", $mono("val")$, " يحتوي الآن على نتيجة ", $mono("factorial(n - 1)")$, "
  restore(\"argl\"),                  // استعادة قائمة الوسائط الجزئية لـ ", $mono("*")$, "
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  restore(\"fun\"),                   // استعادة ", $mono("*")$, "
  restore(\"continue\"),
// تطبيق ", $mono("*")$, " وإرجاع قيمته
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch18\")),
\"compiled_branch19\", // لاحظ أن الدالة المركبة هنا تُستدعى عوديًا ذيليًا
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch18\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
  go_to(reg(\"continue\")),
\"after_call20\",
\"after_cond5\",
\"after_lambda2\",
// إسناد الدالة لـ الاسم ", $mono("factorial")$, "
  perform(list(op(\"assign_symbol_value\"),
               constant(\"factorial\"), reg(\"val\"), reg(\"env\"))),
  assign(\"val\", constant(undefined))
    ")], caption: [(متابعة)], label-name: <fig:continued_1>)

#idx("compiler for Python", sub: "example compilation")

#exercise(label-name: <ex:5_36>, [
ضع في اعتبارك الإعلان التالي لدالة مضروب، وهو مختلف قليلاً عن الإعلان المعطى أعلاه:

#snippet(```python
function factorial_alt(n) {
    return n === 1
           ? 1
           : n * factorial_alt(n - 1);
}
```)

ترجم هذه الدالة وقارن الكود الناتج مع ذلك المُنتج لـ #py("factorial"). اشرح أي اختلافات تجدها. هل ينفذ أي من البرنامجين بكفاءة أكبر من الآخر؟
])

#exercise(label-name: <ex:compiled-fact>, [
ترجم دالة المضروب
#idx("iterative process", sub: "recursive process vs.")
#idx("recursive process", sub: "iterative process vs.")
التكرارية:

#snippet(```python
function factorial(n) {
    function iter(product, counter) {
        return counter > n
               ? product
               : iter(product * counter, counter + 1);
    }
    return iter(1, 1);
}
```)

أضف حواشي للكود الناتج، مظهرًا الفرق الأساسي بين الكود للنسختين التكرارية والعودية لـ #py("factorial") والذي يجعل إحدى العمليتين تبني مساحة مكدس والأخرى تعمل في مساحة مكدس ثابتة.
])

#exercise(label-name: <ex:compiled-code>, [
ما هو البرنامج الذي تُمّ تجميعه لإنتاج الكود الموضح في الشكل @fig:compilation-example1؟
])

#sicp-figure([#syntax($mono(" ")mono(" ")$, "assign(\"val\", list(op(\"make_compiled_function\"),
                     label(\"entry1\"), reg(\"env\"))),
\"entry1\"
  assign(\"env\", list(op(\"compiled_function_env\"), reg(\"fun\"))),
  assign(\"env\", list(op(\"extend_environment\"),
                     constant(list(\"x\")), reg(\"argl\"), reg(\"env\"))),
  revert_stack_to_marker(),
  restore(\"continue\"),
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"+\"), reg(\"env\"))),
  save(\"continue\"),
  save(\"fun\"),
  save(\"env\"),
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"g\"), reg(\"env\"))),
  save(\"fun\"),
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"+\"), reg(\"env\"))),
  assign(\"val\", constant(2)),
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  assign(\"val\", list(op(\"lookup_symbol_value\"), constant(\"x\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch3\")),
\"compiled_branch4\"
  assign(\"continue\", label(\"after_call5\")),
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch3\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
\"after_call5\",
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  restore(\"fun\"),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch7\")),
\"compiled_branch8\",
  assign(\"continue\", label(\"after_call9\")),
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch7\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
\"after_call9\",
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  restore(\"env\"),
  assign(\"val\", list(op(\"lookup_symbol_value\"), constant(\"x\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  restore(\"fun\"),
  restore(\"continue\"),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch11\")),
    ")], caption: [مثال على مخرج المُترجِم (متابعة في الصفحة التالية). انظر التمرين @ex:compiled-code.], label-name: <fig:compilation-example1>)

#sicp-figure([#syntax("
\"compiled_branch12\",
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch11\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
  go_to(reg(\"continue\")),
\"after_call13\",
\"after_lambda2\",
  perform(list(op(\"assign_symbol_value\"),
               constant(\"f\"), reg(\"val\"), reg(\"env\"))),
  assign(\"val\", constant(undefined))
    ")], caption: [(متابعة)], label-name: <fig:continued_2>)

#idx("factorial", sub: "compilation of")

#exercise(label-name: <ex:5_39>, [
ما هو
#idx("order of evaluation", sub: "in compiler")
#idx("compiler for Python", sub: "order of argument evaluation")
ترتيب التقييم الذي ينتجه مُترجِمنا لوسائط تطبيق ما؟
هل هو من اليسار إلى اليمين (كما تقتضي مواصفات ECMAScript)، أم من اليمين إلى اليسار، أم ترتيب آخر؟
أين يحدد هذا الترتيب في المُترجِم؟ عدل المُترجِم بحيث ينتج ترتيب تقييم آخر. (انظر مناقشة ترتيب التقييم لمُقيِّم التحكم الصريح في القسم @sec:eceval-core). كيف يؤثر تغيير ترتيب تقييم الوسائط على كفاءة الكود الذي يبني قائمة الوسائط؟
])

#exercise(label-name: <ex:5_40>, [
إحدى الطرق لفهم آلية
#idx("compiler for Python", sub: "stack usage")
#idx("preserving")
#py("preserving") للمُترجِم لتحسين استخدام المكدس هي رؤية العمليات الإضافية التي كانت ستُولد إذا لم نستخدم هذه الفكرة. عدل #py("preserving") بحيث تولد دائمًا عمليتي #py("save") و #py("restore").
ترجم بعض التعبيرات البسيطة وحدد عمليات المكدس غير الضرورية المُولدة.
قارن الكود بذلك المُنتج مع بقاء آلية #py("preserving") سليمة.
])

#exercise(label-name: <ex:open-code>, [
مُترجِمنا ذكي بشأن تجنب عمليات المكدس غير الضرورية، ولكنه ليس ذكيًا على الإطلاق عندما يتعلق الأمر بترجمة استدعاءات الدوال الأوّلية في اللغة بدلالة العمليات الأوّلية الموفرة بواسطة الآلة. على سبيل المثال، ضع في اعتبارك كمية الكود المُترجَم لحساب #py("a + 1"):
يعد الكود قائمة وسائط في #py("argl")، ويضع دالة الجمع الأوّلية (التي يجدها بالبحث عن الرمز #py("\"+\"") في البيئة) في #py("fun")، ويختبر ما إذا كانت الدالة أوّلية أم مركبة. يولد المُترجِم دائمًا كودًا لتنفيذ الاختبار، بالإضافة إلى كود للفرعين الأوّلي والمركب (واحدهما فقط سُينفذ).
لم نُظهر الجزء من وحدة التحكم الذي ينفذ الأوّليات، لكننا نفترض أن هذه التعليمات تستخدم عمليات الحساب الأوّلية في مسارات بيانات الآلة. ضع في اعتبارك كم كان كود أقل سيُولد إذا تمكن المُترجِم من
#idx("compiler for Python", sub: "open coding of primitives")
#idx("open coding of primitives")
#emph[الترميز المفتوح] (#en[open-code]) للأوّليات—أي، إذا استطاع توليد كود لاستخدام عمليات الآلة الأوّلية هذه مباشرة. التعبير #py("a + 1") قد يُترجم إلى شيء بسيط كـ:#footnote[استخدمنا نفس الرمز #py("+") هنا للإشارة إلى دالة اللغة المصدر وعملية الآلة كلاهما. بشكل عام لن يكون هناك تناظر واحد لواحد بين أوّليات اللغة المصدر وأوّليات الآلة.]

#snippet(```python
assign("val", list(op("lookup_symbol_value"), constant("a"), reg("env"))),
assign("val", list(op("+"), reg("val"), constant(1)))
```)

في هذا التمرين سنوسع مُترجِمنا لدعم الترميز المفتوح لأوّليات محددة. سُيولد كود مخصص الأغراض لاستدعاءات هذه الدوال الأوّلية بدلاً من كود تطبيق الدوال العام. ولدعم ذلك، سنعزز آلتنا بمسجّلات وسائط خاصة #py("arg1") و #py("arg2").
تأخذ عمليات الحساب الأوّلية للآلة مدخلاتها من #py("arg1") و #py("arg2"). وقد توضع النتائج في #py("val") أو #py("arg1") أو #py("arg2").

يجب أن يكون المُترجِم قادرًا على التعرف على تطبيق أوّلية ذات ترميز مفتوح في البرنامج المصدر. سنعزز التوجيه في الدالة #py("compile") للتعرف على أسماء هذه الأوّليات بالإضافة إلى الأشكال النحوية التي يتعرف عليها حاليًا.
لكل شكل نحوي يملك مُترجِمنا مولد كود. في هذا التمرين سنبني عائلة من مولدات الكود للأوّليات ذات الترميز المفتوح.

+ الأوّليات ذات الترميز المفتوح، على عكس الأشكال النحوية، تحتاج جميعها إلى تقييم تعبيرات وسائطها. اكتب مولد كود #py("spread_arguments") للاستخدام بواسطة جميع مولدات كود الترميز المفتوح. ينبغي أن تأخذ الدالة #py("spread_arguments") قائمة بتعبيرات الوسائط وتترجم تعبيرات الوسائط المعطاة موجهة إلى مسجّلات وسائط متتالية. لاحظ أن تعبير الوسيط قد يحتوي على استدعاء لأوّلية ذات ترميز مفتوح، لذا يجب حفظ مسجّلات الوسائط أثناء تقييم تعبير الوسيط.
+ معامِلات بايثون #py("===") و #py("*") و #py("-") و #py("+")، بين أمور أخرى، تُنفذ في آلة المسجّلات كدوال أوّلية ويُشار إليها في البيئة العالمية بالرموز #py("\"===\"") و #py("\"*\"") و #py("\"-\"") و #py("\"+\""). في بايثون، لا يمكن إعادة إعلان هذه الأسماء، لأنها لا تستوفي القيود النحوية للأسماء. هذا يعني أنه من الآمن ترميزها بشكل مفتوح. لكل من الدوال الأوّلية #py("===") و #py("*") و #py("-") و #py("+")، اكتب مولد كود تأخذ تطبيقًا مع تعبير دالة يسمي تلك الدالة، جنبًا إلى جنب مع هدف وواصف ارتباط، وتُنتج كودًا لنشر الوسائط في المسجّلات ثم تنفيذ العملية موجهة إلى الهدف المعطى مع الارتباط المعطى. اجعل #py("compile") توجه إلى مولدات الكود هذه.
+ جرب مُترجِمك الجديد على مثال #py("factorial"). قارن الكود الناتج مع النتيجة المُنتجة بدون ترميز مفتوح.
])
