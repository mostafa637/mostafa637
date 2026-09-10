// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([ترجمة التطبيقات وتعليمات الإرجاع], label-name: <sec:compiling-combinations>)

#idx("compiler for Python", sub: "function applications")
#idx("compiler for Python", sub: "combinations")

جوهر عملية الترجمة هو ترجمة تطبيقات الدوال. الكود لـ تطبيق المُترجَم مع هدف وارتباط معينين يملك الشكل:

#syntax(metaphrase[ترجمة تعبير الدالة، الهدف #py("fun")، الارتباط #py("\"next\"")], "
", metaphrase[تقييم تعبيرات الوسائط وبناء قائمة الوسائط في #py("argl")], "
", metaphrase[ترجمة استدعاء الدالة مع الهدف والارتباط المعطيين])

قد يلزم حفظ واستعادة المسجّلات #py("env") و #py("fun") و #py("argl") أثناء تقييم تعبيرات الدالة والوسائط.
لاحظ أن هذا هو المكان الوحيد في المُترجِم الذي يُحدد فيه هدف آخر غير #py("val").

الكود المطلوب يُولد بواسطة #py("compile_application").
تترجم هذه عوديًا تعبير الدالة، لإنتاج كود يضع الدالة المراد تطبيقها في #py("fun")، وتترجم تعبيرات الوسائط، لإنتاج كود يُقيّم تعبيرات الوسائط الفردية للتطبيق. تُجمع تسلسلات التعليمات لتعبيرات الوسائط (بواسطة #py("construct_arglist")) مع الكود الذي يبني قائمة الوسائط في #py("argl")، ويُجمع كود قائمة الوسائط الناتج مع كود الدالة والكود الذي ينفذ استدعاء الدالة (المُنتج بواسطة #py("compile_function_call")).
في إلحاق تسلسلات الكود، يجب حفظ المسجّل #py("env") حول تقييم تعبير الدالة (نظرًا لأن تقييم تعبير الدالة قد يعدل #py("env")، والذي سيلزم لتقييم تعبيرات الوسائط)، ويجب حفظ المسجّل #py("fun") حول بناء قائمة الوسائط (نظرًا لأن تقييم تعبيرات الوسائط قد يعدل #py("fun")، والذي سيلزم لتطبيق الدالة الفعلي). يجب أيضًا حفظ المسجّل #py("continue") طوال الوقت، نظرًا لأنه يلزم للارتباط في استدعاء الدالة.
#idx("compileapplication", decl: true)
#snippet(```python
function compile_application(exp, target, linkage) {
    const fun_code = compile(function_expression(exp), "fun", "next");
    const argument_codes = map(arg => compile(arg, "val", "next"),
                               arg_expressions(exp));
    return preserving(list("env", "continue"),
                      fun_code,
                      preserving(list("fun", "continue"),
                          construct_arglist(argument_codes),
                          compile_function_call(target, linkage)));
}
```)

الكود لبناء قائمة الوسائط سُيقيّم كل تعبير وسيط في #py("val") ثم يجمع تلك القيمة مع قائمة الوسائط التي تُجمع في #py("argl") باستخدام #py("pair").
نظرًا لأننا نلحق الوسائط في مقدمة #py("argl") بالتتابع، يجب أن نبدأ بالوسيط الأخير وننتهي بالأول، بحيث تظهر الوسائط بترتيب من الأول إلى الأخير في القائمة الناتجة.
بدلاً من إضاعة تعليمة بتهيئتها #py("argl") إلى القائمة الفارغة للإعداد لهذا التسلسل من التقييمات، نجعل تسلسل الكود الأول يبني #py("argl") الأولي.
الشكل العام لبناء قائمة الوسائط هو بالتالي كما يلي:

#syntax(metaphrase[ترجمة الوسيط الأخير، موجه إلى #py("val")], "
assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
", metaphrase[ترجمة الوسيط التالي، موجه إلى #py("val")], "
assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
", $dots.h$, "
", metaphrase[ترجمة الوسيط الأول، موجه إلى #py("val")], "
assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
      ")

يجب حفظ المسجّل #py("argl") حول كل تقييم وسيط باستثناء الأول (حتى لا تُفقد الوسائط المجمعة حتى الآن)، ويجب حفظ #py("env") حول كل تقييم وسيط باستثناء الأخير (للاستخدام بواسطة تقييمات الوسائط اللاحقة).

ترجمة كود الوسائط هذا دقيقة بعض الشيء، بسبب المعاملة الخاصة لتعبير الوسيط الأول المراد تقييمه والحاجة إلى حفظ #py("argl") و #py("env") في أماكن مختلفة.
تأخذ الدالة #py("construct_arglist") كوسائط الكود الذي يُقيّم تعبيرات الوسائط الفردية.
إذا لم تكن هناك تعبيرات وسائط على الإطلاق، فإنها تُصنع ببساطة التعليمة:

#snippet(```python
assign(argl, constant(null))
```)

بخلاف ذلك، تنشئ #py("construct_arglist") كودًا يُهيئ #py("argl") مع الوسيط الأخير، وتُلحق كودًا يُقيّم باقي الوسائط ويضمها إلى #py("argl") بالتتابع. من أجل معالجة الوسائط من الأخير إلى الأول، يجب أن نعكس قائمة تسلسلات كود الوسائط من الترتيب الموفر بواسطة #py("compile_application").
#idx("constructarglist", decl: true)
#snippet(```python
function construct_arglist(arg_codes) {
    if (is_null(arg_codes)) {
        return make_instruction_sequence(null, list("argl"),
                   list(assign("argl", constant(null))));
    } else {
        const rev_arg_codes = reverse(arg_codes);
        const code_to_get_last_arg =
            append_instruction_sequences(
                head(rev_arg_codes),
                make_instruction_sequence(list("val"), list("argl"),
                    list(assign("argl",
                                list(op("list"), reg("val"))))));
        return is_null(tail(rev_arg_codes))
               ? code_to_get_last_arg
               : preserving(list("env"),
                     code_to_get_last_arg,
                     code_to_get_rest_args(tail(rev_arg_codes)));
    }
}
function code_to_get_rest_args(arg_codes) {
    const code_for_next_arg =
        preserving(list("argl"),
            head(arg_codes),
            make_instruction_sequence(list("val", "argl"), list("argl"),
                list(assign("argl", list(op("pair"),
                                         reg("val"), reg("argl"))))));
    return is_null(tail(arg_codes))
           ? code_for_next_arg
           : preserving(list("env"),
                        code_for_next_arg,
                        code_to_get_rest_args(tail(arg_codes)));
}
```)

#subheading([تطبيق الدوال])

بعد تقييم عناصر تطبيق الدالة، يجب على الكود المُترجَم تطبيق الدالة في #py("fun") على الوسائط في #py("argl"). ينفذ الكود التوجيه نفسه بالأساس مثل الدالة #py("apply") في المُقيِّم ما فوق الدائري في القسم @sec:core-of-evaluator أو نقطة الإدخال #py("apply_dispatch") في مُقيِّم التحكم الصريح في القسم @sec:procedure-application. حيث يفحص ما إذا كانت الدالة المراد تطبيقها دالة أوّلية أم دالة مُترجَمة.
بالنسبة للدالة الأوّلية، يستخدم #py("apply_primitive_function")؛ وسنرى قريباً كيف يتعامل مع الدوال المُترجَمة.
كود تطبيق الدالة يملك الشكل التالي:

#syntax($mono(" ")mono(" ")$, "test(list(op(\"primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch\")),
\"compiled_branch\",
  ", metaphrase[الكود لتطبيق الدالة المُترجَمة مع الهدف المعطى والارتباط المناسب], "
\"primitive_branch\",
  assign(", meta("target"), ",
         list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
  ", metaphrase[الارتباط], "
\"after_call\"
      ")

لاحظ أنه يجب على الفرع المُترجَم التخطي حول الفرع الأوّلي. لذلك، إذا كان الارتباط لاستدعاء الدالة الأصلي هو #py("\"next\"")، يجب أن يستخدم الفرع المركب ارتباطًا يقفز إلى تسمية تم إدراجها بعد الفرع الأوّلي.
(هذا يشبه الارتباط المستخدم لفرع الصحة في #py("compile_conditional").)
#idx("compilefunctioncall", decl: true)
#snippet(```python
function compile_function_call(target, linkage) {
    const primitive_branch = make_label("primitive_branch");
    const compiled_branch = make_label("compiled_branch");
    const after_call = make_label("after_call");
    const compiled_linkage = linkage === "next" ? after_call : linkage;
    return append_instruction_sequences(
        make_instruction_sequence(list("fun"), null,
            list(test(list(op("is_primitive_function"), reg("fun"))),
                 branch(label(primitive_branch)))),
        append_instruction_sequences(
            parallel_instruction_sequences(
                append_instruction_sequences(
                    compiled_branch,
                    compile_fun_appl(target, compiled_linkage)),
                append_instruction_sequences(
                    primitive_branch,
                    end_with_linkage(linkage,
                        make_instruction_sequence(list("fun", "argl"),
                                                  list(target),
                            list(assign(
                                   target,
                                   list(op("apply_primitive_function"),
                                        reg("fun"), reg("argl")))))))),
            after_call));
}
```)

الفرعان الأوّلي والمركب، مثل فرعي الصحة والخطأ في #py("compile_conditional")، يُلحقان باستخدام #py("parallel_instruction_sequences") بدلاً من #py("append_instruction_sequences") العادية، لأنهما لن يُنفذا تتابعيًا.

#subheading([تطبيق الدوال المُترجَمة])

التعامل مع تطبيق الدالة والإرجاع هو الجزء الأكثر دقة في المُترجِم.
تملك الدالة المُترجَمة (كما بُنيت بواسطة #py("compile_lambda_expression")) نقطة إدخال، وهي تسمية تحدد أين يبدأ الكود الخاص بالدالة. الكود عند نقطة الإدخال هذه يحسب نتيجة في #py("val") وينتهي بتنفيذ التعليمات من تعليمة إرجاع مُترجَمة.

يستخدم الكود الخاص بتطبيق الدالة المُترجَمة المكدس بنفس الطريقة مثل مُقيِّم التحكم الصريح (القسم @sec:procedure-application):
قبل القفز إلى نقطة إدخال الدالة المُترجَمة، يحفظ مواصلة استدعاء الدالة على المكدس، متبوعة بعلامة تسمح بالتراجع عن المكدس إلى الحالة التي سبقت الاستدعاء مباشرة مع وجود المواصلة في الأعلى.

#syntax($mono(" ")mono(" ")$, "// الإعداد للإرجاع من الدالة
  save(\"continue\"),
  push_marker_to_stack(),
  // القفز إلى نقطة إدخال الدالة
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
        ")

ترجمة تعليمة إرجاع (مع #py("compile_return_statement")) تُولد كودًا مقابلًا للتراجع عن المكدس واستعادة والقفز إلى #py("continue").

#syntax($mono(" ")mono(" ")$, "revert_stack_to_marker(),
  restore(\"continue\"),
  ", metaphrase[تقييم تعبير الإرجاع وتخزين النتيجة في #py("val")], "
  go_to(reg(\"continue\")), // كود ارتباط ", $mono("\"return\"")$, "
         ")

ما لم تدخل دالة في حلقة لا نهائية، فإنها ستنتهي بتنفيذ كود الإرجاع أعلاه، الناتج إما عن تعليمة إرجاع في البرنامج أو تعليمة مدرجة بواسطة #py("compile_lambda_body")
#idx("return value", sub: "undefined as")
لإرجاع #py("undefined").#footnote[نظرًا لأن تنفيذ جسم دالة ينتهي دائمًا بإرجاع، فلا داعي هنا لآلية مثل نقطة الإدخال #py("return_undefined") من القسم @sec:procedure-application.]

الكود المباشر لتطبيق دالة مُترجَمة مع هدف وارتباط معينين سيقوم بإعداد #py("continue") لتجعل الدالة تعود إلى تسمية محليّة بدلاً من الارتباط النهائي، لنسخ قيمة الدالة من #py("val") إلى المسجّل المستهدف إذا لزم الأمر. وسيبدو كالتالي إذا كان الارتباط تسمية:

#syntax($mono(" ")mono(" ")$, "assign(\"continue\", label(\"fun_return\")), // أين يجب أن تعود الدالة
  save(\"continue\"),       // ستُستعاد بواسطة الدالة
  push_marker_to_stack(), // يسمح للدالة بالتراجع عن المكدس للعثور على ", $mono("fun_return")$, "
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),    // تتراجع في النهاية عن المكدس، وتستعيد وتقفز إلى ", $mono("continue")$, "
\"fun_return\",             // الدالة تعود إلى هنا
  assign(", $italic("target")$, ", reg(\"val\")), // مُضمن إذا لم يكن الهدف ", $mono("val")$, "
  go_to(label(", meta("linkage"), ")),   // كود الارتباط
      ")

أو سيبدو هكذا—حافطًا مواصلة المستدعي عند البداية من أجل استعادتها والذهاب إليها عند النهاية—إذا كان الارتباط #py("\"return\"") (أي إذا كان التطبيق في تعليمة إرجاع وكانت قيمته هي النتيجة المراد إرجاعها):

#syntax($mono(" ")mono(" ")$, "save(\"continue\"),       // حفظ مواصلة المستدعي
  assign(\"continue\", label(\"fun_return\")), // أين يجب أن تعود الدالة
  save(\"continue\"),       // ستُستعاد بواسطة الدالة
  push_marker_to_stack(), // يسمح للدالة بالتراجع عن المكدس للعثور على ", $mono("fun_return")$, "
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),    // تتراجع في النهاية عن المكدس، وتستعيد وتقفز إلى ", $mono("continue")$, "
\"fun_return\",             // الدالة تعود إلى هنا
  assign(", $italic("target")$, ", reg(\"val\")), // مُضمن إذا لم يكن الهدف ", $mono("val")$, "
  restore(\"continue\"),    // استعادة مواصلة المستدعي
  go_to(reg(\"continue\")), // كود الارتباط
      ")

هذا الكود يُعد #py("continue") بحيث تعود الدالة إلى التسمية #py("fun_return") ويقفز إلى نقطة إدخال الدالة. الكود عند #py("fun_return") ينقل نتيجة الدالة من #py("val") إلى المسجّل المستهدف (إذا لزم الأمر) ثم يقفز إلى الموقع المحدد بواسطة الارتباط.
(الارتباط دائمًا هو #py("\"return\"") أو تسمية، لأن #py("compile_function_call") يستبدل ارتباط #py("\"next\"") لفرع الدالة المركبة بتسمية #py("after_call").)

قبل القفز إلى نقطة إدخال الدالة، نحفظ #py("continue") وننفذ #py("push_marker_to_stack()") لتمكين الدالة من العودة إلى الموقع المقصود في البرنامج مع المكدس المتوقع. تعليمات #py("revert_stack_to_marker()") و #py("restore(\"continue\")") المقابلة تُولد بواسطة #py("compile_return_statement") لكل تعليمة إرجاع في جسم الدالة.#footnote[في مكان آخر في المُترجِم، جميع عمليات حفظ واستعادة المسجّلات تُولد بواسطة #py("preserving") لحفظ قيمة مسجّل عبر تسلسل من التعليمات بحفظها قبل تلك التعليمات واستعادتها بعد ذلك—على سبيل المثال عبر تقييم محمول شرطي. ولكن هذه الآلية لا يمكنها توليد تعليمات لحفظ واستعادة #py("continue") لتطبيق دالة والإرجاع المقابل، لأن هذه تترجم بشكل منفصل وليست متجاورة. بدلاً من ذلك، يجب توليد عمليات الحفظ والاستعادة هذه صراحة بواسطة #py("compile_fun_appl") و #py("compile_return_statement").]

في الواقع، إذا لم يكن الهدف هو #py("val")، فإن ما سبق هو الكود الذي سيولده مُترجِمنا بالضبط.#footnote[في الواقع، نرفع إشارة خطأ عندما لا يكون الهدف هو #py("val") والارتباط هو #py("\"return\"")، نظرًا لأن المكان الوحيد الذي نطلب فيه ارتباط #py("\"return\"") هو في ترجمة تعبيرات الإرجاع، واتفاقيتنا هي أن الدوال تُرجع قيمها في #py("val").]
عادة، ومع ذلك، فإن الهدف هو #py("val") (المرة الوحيدة التي يحدد فيها المُترجِم مسجّلاً مختلفاً هي عند استهداف تقييم تعبير دالة إلى #py("fun"))، لذا توضع نتيجة الدالة مباشرة في المسجّل المستهدف ولا داعي للقفز إلى موقع خاص ينسخها. بدلاً من ذلك نُبسط الكود بإعداد #py("continue") بحيث "تعد" الدالة المستدعاة مباشرة إلى المكان المحدد بواسطة ارتباط المستدعي:

#syntax(metaphrase[إعداد #py("continue") للارتباط ودفع العلامة], "
assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
go_to(reg(\"val\")),
      ")

إذا كان الارتباط تسمية، نقوم بإعداد #py("continue") بحيث تواصل الدالة عند تلك التسمية. (أي أن الـ #py("go_to(reg(\"continue\"))") التي تنتهي بها الدالة المستدعاة تصبح مكافئة لـ #py("go_to(label(")#meta("linkage")#py("))") عند #py("fun_return") أعلاه.)

#syntax("
assign(\"continue\", label(", meta("linkage"), ")),
save(\"continue\"),
push_marker_to_stack(),
assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
go_to(reg(\"val\")),
      ")

إذا كان الارتباط #py("\"return\"")، فلا نحتاج إلى إسناد #py("continue"): فهو يحمل بالفعل الموقع المطلوب.
(أي أن الـ #py("go_to(reg(\"continue\"))") التي تنتهي بها الدالة المستدعاة تذهب مباشرة إلى المكان الذي كانت ستذهب إليه الـ #py("go_to(reg(\"continue\"))") عند #py("fun_return").)

#snippet(```python
save("continue"),
push_marker_to_stack(),
assign("val", list(op("compiled_function_entry"), reg("fun"))),
go_to(reg("val")),
```)

مع هذا التنفيذ لارتباط #py("\"return\"")، يولد المُترجِم
#idx("return statement", sub: "tail recursion and")
#idx("compiler for Python", sub: "tail-recursive code generated by")
#idx("tail recursion", sub: "compiler and")
كودًا عوديًا ذيليًا.
استدعاء دالة في تعليمة إرجاع تكون قيمتها هي النتيجة المراد إرجاعها يجري نقلاً مباشرًا، دون حفظ معلومات غير ضرورية على المكدس.

افترض بدلاً من ذلك أننا تعاملنا مع حالة استدعاء دالة مع ارتباط #py("\"return\"") وهدف #py("val") بنفس الطريقة كما بالنسبة لهدف ليس #py("val"). كان هذا يدمر العودية الذيلية. كان نظامنا سيظل يُرجع استدعاء الدالة. ولكن في كل مرة نستدعي فيها دالة، كنا سنحفظ #py("continue") ونعود بعد الاستدعاء لـ إلغاء الحفظ (غير المفيد). كانت عمليات الحفظ الإضافية هذه ستتراكم أثناء تداخل استدعاءات الدوال.#footnote[جعل المُترجِم يولد كودًا عوديًا ذيليًا هو أمر مرغوب فيه، خاصة في النمط الوظيفي. ومع ذلك، فإن المُترجِمات للغات الشائعة، بما في ذلك C و C++، لا تفعل ذلك دائمًا، وبالتالي لا يمكن لهذه اللغات تمثيل العمليات التكرارية بدلالة استدعاء الدالة وحده. الصعوبة مع العودية الذيلية في هذه اللغات هي أن تنفيذاتها تستخدم المكدس لتخزين وسائط الدوال والأسماء المحلية بالإضافة إلى عناوين العودة. تنفيذات بايثون الموصوفة في هذا الكتاب تخزن الوسائط والأسماء في الذاكرة لتجميع المهملات. والسبب في استخدام المكدس للأسماء والوسائط هو أنه يتجنب الحاجة إلى تجميع المهملات في اللغات التي لن تتطلب ذلك بخلاف ذلك، ويُعتقد عموماً أنه أكثر كفاءة. يمكن للمُترجِمات المتطورة، في الواقع، استخدام المكدس للوسائط دون تدمير العودية الذيلية. (انظر هانسون 1990 لوصف ذلك). هناك أيضًا بعض النقاش حول ما إذا كان تخصيص المكدس أكثر كفاءة في الواقع من تجميع المهملات في المقام الأول، ولكن التفاصيل يبدو أنها تعتمد على نقاط دقيقة في بنية الحاسوب. (انظر أبل 1987 وميلر وروتساس 1994 لوجهات نظر متعارضة حول هذه المسألة).]

تولد الدالة #py("compile_fun_appl") كود تطبيق الدالة أعلاه بالنظر في أربع حالات، اعتمادًا على ما إذا كان الهدف للاستدعاء هو #py("val") وما إذا كان الارتباط هو #py("\"return\"").
لاحظ أن تسلسلات التعليمات تُعلن أنها تعدل جميع المسجّلات، نظرًا لأن تنفيذ جسم الدالة يمكن أن يغير المسجّلات بطرق عشوائية.#footnote[الثابت
#idx("compiler for Python", sub: "register use")
#py("all_regs")
مربوط بقائمة أسماء جميع المسجّلات:
#idx("allregs (compiler)", decl: true)
#snippet(```python
const all_regs = list("env", "fun", "val", "argl", "continue");
```)]

#idx("compilefunappl", decl: true)
#syntax("
function compile_fun_appl(target, linkage) {
    const fun_return = make_label(\"fun_return\");
    return target === \"val\" && linkage !== \"return\"
           ? make_instruction_sequence(list(\"fun\"), all_regs,
                 list(assign(\"continue\", label(linkage)),
                      save(\"continue\"),
                      push_marker_to_stack(),
                      assign(\"val\", list(op(\"compiled_function_entry\"),
                                         reg(\"fun\"))),
                      go_to(reg(\"val\"))))
           : target !== \"val\" && linkage !== \"return\"
           ? make_instruction_sequence(list(\"fun\"), all_regs,
                 list(assign(\"continue\", label(fun_return)),
                      save(\"continue\"),
                      push_marker_to_stack(),
                      assign(\"val\", list(op(\"compiled_function_entry\"),
                                         reg(\"fun\"))),
                      go_to(reg(\"val\")),
                      fun_return,
                      assign(target, reg(\"val\")),
                      go_to(label(linkage))))
           : target === \"val\" && linkage === \"return\"
           ? make_instruction_sequence(list(\"fun\", \"continue\"),
                                       all_regs,
                 list(save(\"continue\"),
                      push_marker_to_stack(),
                      assign(\"val\", list(op(\"compiled_function_entry\"),
                                         reg(\"fun\"))),
                      go_to(reg(\"val\"))))
           : // ", $mono("target !== \"val\" && linkage === \"return\"")$, "
             error(target, \"return linkage, target not val -- compile\");
}
      ")

#idx("compiler for Python", sub: "function applications")
#idx("compiler for Python", sub: "combinations")

أوضحنا كيفية توليد كود ارتباط عودي ذيلي لتطبيق دالة عندما يكون الارتباط هو #py("\"return\"")—أي عندما يكون التطبيق في تعليمة إرجاع وقيمته هي النتيجة المراد إرجاعها. وبالمثل، كما تم شرحه في القسم @sec:procedure-application، فإن آلية علامة المكدس المستخدمة هنا (وفي مُقيِّم التحكم الصريح) للاستدعاء والإرجاع تُنتج سلوكًا عوديًا ذيليًا في تلك الحالة فقط. هاتان الناحيتان من الكود المُولد لتطبيق الدالة تتحدان لضمان أنه عندما تنتهي دالة بإرجاع قيمة استدعاء دالة، فإنه لا يتراكم أي مكدس.

#subheading([ترجمة تعليمات الإرجاع])

الكود لـ
#idx("compiler for Python", sub: "return statements")
#idx("return statement", sub: "handling in compiler")
تعليمة إرجاع ياخذ الشكل التالي، بغض النظر عن الارتباط والهدف المعطيين:

#syntax("
revert_stack_to_marker(),
restore(\"continue\"),   // حُفظ بواسطة ", $mono("compile_fun_appl")$, "
", metaphrase[تقييم تعبير الإرجاع وتخزين النتيجة في #py("val")], "
go_to(reg(\"continue\")) // كود ارتباط ", $mono("\"return\"")$, "
          ")

تعليمات التراجع عن المكدس باستخدام العلامة ثم استعادة #py("continue") تناظر التعليمات المـُولدة بواسطة #py("compile_fun_appl") لحفظ #py("continue") وتظليل المكدس.
القفزة النهائية إلى #py("continue") يُولدها استخدام الارتباط #py("\"return\"") عند ترجمة تعبير الإرجاع.
الدالة #py("compile_return_statement") مختلفة عن جميع مولدات الكود الأخرى في أنها تتجاهل وسائط الهدف والارتباط—فهي تترجم دائمًا تعبير الإرجاع مع الهدف #py("val") والارتباط #py("\"return\"").
#idx("compilereturnstatement", decl: true)
#snippet(```python
function compile_return_statement(stmt, target, linkage) {
    return append_instruction_sequences(
               make_instruction_sequence(null, list("continue"),
                   list(revert_stack_to_marker(),
                        restore("continue"))),
               compile(return_expression(stmt), "val", "return"));
}
```)
