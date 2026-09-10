// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([ترجمة المكونات], label-name: <sec:compiling-components>)

في هذا القسم والقسم التالي ننفذ مولدات الكود التي توجه إليها الدالة #py("compile").

#subheading([ترجمة كود الارتباط])

بشكل عام، ينتهي مخرج كل مولد كود بتعليمات—مُولدة بواسطة
#idx("compiler for Python", sub: "linkage code")
الدالة #py("compile_linkage")—تنفذ الارتباط المطلوب. إذا كان الارتباط #py("\"return\"") فإنه يجب علينا توليد التعليمة #py("go_to(reg(\"continue\"))").
هذا يحتاج إلى المسجّل #py("continue") ولا يعدل أي مسجّلات.
إذا كان الارتباط #py("\"next\"")، فلا داعي لتضمين أي تعليمات إضافية. بخلاف ذلك، يكون الارتباط تسمية، ونولد #py("go_to") إلى تلك التسمية، وهي تعليمة لا تحتاج أو تعدل أي مسجّلات.
#idx("compilelinkage", decl: true)
#snippet(```python
function compile_linkage(linkage) {
    return linkage === "return"
           ? make_instruction_sequence(list("continue"), null,
                                       list(go_to(reg("continue"))))
           : linkage === "next"
           ? make_instruction_sequence(null, null, null)
           : make_instruction_sequence(null, null,
                                       list(go_to(label(linkage))));
}
```)

يُلحق كود الارتباط بتسلسل تعليمات بواسطة #py("preserving") للمسجّل #py("continue")، نظرًا لأن ارتباط #py("\"return\"") سيتطلب المسجّل #py("continue"):
إذا كان تسلسل التعليمات المعطى يعدل #py("continue") وكود الارتباط يحتاجه، سيتم حفظ #py("continue") واستعادته.
#idx("endwithlinkage", decl: true)
#snippet(```python
function end_with_linkage(linkage, instruction_sequence) {
    return preserving(list("continue"),
                      instruction_sequence,
                      compile_linkage(linkage));
}
```)

#subheading([ترجمة المكونات البسيطة])

مولدات الكود لـ
#idx("compiler for Python", sub: "literals")
#idx("compiler for Python", sub: "names")
التعبيرات الصريحة والأسماء تبني تسلسلات تعليمات تسند القيمة المطلوبة إلى المسجّل المستهدف ثم تواصل كما هو محدد بواسطة واصف الارتباط.

تستخرج القيمة الصريحة في وقت الترجمة من المكون المراد ترجمته وتوضع في الجزء الثابت من تعليمة #py("assign"). بالنسبة للاسم، تُولد تعليمة لاستخدام عملية #py("lookup_symbol_value") عند تشغيل البرنامج المُترجَم، للبحث عن القيمة المقترنة برمز في البيئة الحالية. مثل القيمة الصريحة، يُستخرج الرمز في وقت الترجمة من المكون المراد ترجمته. وبالتالي فإن #py("symbol_of_name(component)") تنفذ مرة واحدة فقط، عندما يتم ترجمة البرنامج، ويظهر الرمز كـ ثابت في تعليمة #py("assign").

#idx("compileliteral", decl: true)#idx("compilename", decl: true)
#snippet(```python
function compile_literal(component, target, linkage) {
    const literal = literal_value(component);
    return end_with_linkage(linkage,
               make_instruction_sequence(null, list(target),
                   list(assign(target, constant(literal)))));
}
function compile_name(component, target, linkage) {
    const symbol = symbol_of_name(component);
    return end_with_linkage(linkage,
               make_instruction_sequence(list("env"), list(target),
                   list(assign(target,
                               list(op("lookup_symbol_value"),
                                    constant(symbol),
                                    reg("env"))))));
}
```)

تعليما الإسناد هاتان تعدلان المسجّل المستهدف، والتعليمة التي تبحث عن رمز تحتاج إلى المسجّل #py("env").

الإسنادات و
#idx("compiler for Python", sub: "assignments")
#idx("compiler for Python", sub: "declarations")
الإعلانات
تتم معالجتها بنفس الطريقة تقريبًا كما في المُفسِّر.
الدالة #py("compile_assignment_declaration") تولد عوديًا كودًا يحسب القيمة المراد اقترانها بالرمز وتُلحق به تسلسل تعليمات من تعليمتين يثبت القيمة المقترنة بالرمز في البيئة ويسند قيمة المكون بأكمله (القيمة المسندة بالنسبة للإسناد أو #py("undefined") بالنسبة للإعلان) إلى المسجّل المستهدف. الترجمة العودية تملك الهدف #py("val") والارتباط #py("\"next\"") بحيث يضع الكود نتيجته في #py("val") ويواصل مع الكود المـُلحق بعده. ويتم الإلحاق مع حفظ #py("env")، نظرًا لأن البيئة تلزم لتحديث اقتران الرمز والقيمة وكان من الممكن أن يكون الكود لحساب القيمة عبارة عن ترجمة لتعبير معقد قد يعدل المسجّلات بطرق عشوائية.
#idx("compileassignment", decl: true)#idx("compiledeclaration", decl: true)
#snippet(```python
function compile_assignment(component, target, linkage) {
    return compile_assignment_declaration(
               assignment_symbol(component),
               assignment_value_expression(component),
               reg("val"),
               target, linkage);
}

function compile_declaration(component, target, linkage) {
    return compile_assignment_declaration(
               declaration_symbol(component),
               declaration_value_expression(component),
               constant(undefined),
               target, linkage);
}
function compile_assignment_declaration(
             symbol, value_expression, final_value,
             target, linkage) {
    const get_value_code = compile(value_expression, "val", "next");
    return end_with_linkage(linkage,
               preserving(list("env"),
                   get_value_code,
                   make_instruction_sequence(list("env", "val"),
                                             list(target),
                       list(perform(list(op("assign_symbol_value"),
                                         constant(symbol),
                                         reg("val"),
                                         reg("env"))),
                            assign(target, final_value)))));
}
```)

تسلسل التعليمتين المـُلحق يتطلب #py("env") و #py("val") ويعدل الهدف. لاحظ أنه على الرغم من أننا نحفظ #py("env") لهذا التسلسل، إلا أننا لا نحفظ #py("val")، لأن #py("get_value_code") صُمم ليضع نتيجته صراحة في #py("val") لاستخدامها بواسطة هذا التسلسل.
(في الواقع، إذا حفظنا #py("val")، فسنقع في خلل، لأن هذا سيتسبب في استعادة المحتويات السابقة لـ #py("val") بعد تشغيل #py("get_value_code") مباشرة.)

#subheading([ترجمة الشرطيات])

الكود الخاص بـ
#idx("compiler for Python", sub: "conditionals")
الشرطي المُترجَم مع هدف وارتباط معينين يملك الشكل:

#syntax(metaphrase[ترجمة المحمول، الهدف #py("val")، الارتباط #py("\"next\"")], "
  test(list(op(\"is_falsy\"), reg(\"val\"))),
  branch(label(\"false_branch\")),
\"true_branch\",
  ", metaphrase[ترجمة النتيجة مع الهدف المعطى والارتباط المعطى أو #py("after_cond")], "
\"false_branch\",
  ", metaphrase[ترجمة البديل مع الهدف والارتباط المعطيين], "
\"after_cond\"
      ")

لتوليد هذا الكود، نترجم المحمول والنتيجة والبديل، ونجمع الكود الناتج مع تعليمات لاختبار نتيجة المحمول ومع تسميات مُولدة حديثاً لتظليل فرعي الصحة والخطأ ونهاية الشرطي.#footnote[لا يمكننا استخدام التسميات #py("true_branch") و #py("false_branch") و #py("after_cond") كما هو موضح أعلاه فقط، لأنه قد يكون هناك أكثر من شرطي واحد في البرنامج.
#idx("compiler for Python", sub: "label generation")
يستخدم المُترجِم الدالة #py("make_label") لتوليد التسميات.
تأخذ الدالة #py("make_label") سلسلة نصية كوسيط وتُرجع سلسلة نصية جديدة تبدأ بالسلسلة المعطاة. على سبيل المثال، الاستدعاءات المتتالية لـ #py("make_label(\"a\")") تُرجع #py("\"a1\"") و #py("\"a2\"") وهكذا.
يمكن تنفيذ الدالة #py("make_label") بشكل مشابه لتوليد أسماء المتغيرات الفريدة في لغة الاستعلام، كما يلي:
#idx("makelabel", decl: true)
#snippet(```python
let label_counter = 0;

function new_label_number() {
    label_counter = label_counter + 1;
    return label_counter;
}
function make_label(string) {
    return string + stringify(new_label_number());
}
```)]
في هذا الترتيب الكودي، يجب أن نتفرع حول فرع الصحة إذا كان الاختبار خاطئاً. التعقيد الطفيف الوحيد هو في كيفية التعامل مع ارتباط فرع الصحة. إذا كان الارتباط للشرطي هو #py("\"return\"") أو تسمية، فإن فرعي الصحة والخطأ كلاهما سيستخدمان نفس هذا الارتباط. إذا كان الارتباط هو #py("\"next\"")، فإن فرع الصحة ينتهي بقفزة حول كود فرع الخطأ إلى التسمية في نهاية الشرطي.
#idx("compileconditional", decl: true)
#snippet(```python
function compile_conditional(component, target, linkage) {
    const t_branch = make_label("true_branch");
    const f_branch = make_label("false_branch");
    const after_cond = make_label("after_cond");
    const consequent_linkage =
            linkage === "next" ? after_cond : linkage;
    const p_code = compile(conditional_predicate(component),
                           "val", "next");
    const c_code = compile(conditional_consequent(component),
                           target, consequent_linkage);
    const a_code = compile(conditional_alternative(component),
                           target, linkage);
    return preserving(list("env", "continue"),
             p_code,
             append_instruction_sequences(
               make_instruction_sequence(list("val"), null,
                 list(test(list(op("is_falsy"), reg("val"))),
                      branch(label(f_branch)))),
               append_instruction_sequences(
                 parallel_instruction_sequences(
                   append_instruction_sequences(t_branch, c_code),
                   append_instruction_sequences(f_branch, a_code)),
                 after_cond)));
}
```)

يُحفظ المسجّل #py("env") حول كود المحمول لأنه قد يلزم لفرعي الصحة والخطأ، ويُحفظ #py("continue") لأنه قد يلزم لكود الارتباط في تلك الفروع.
الكود لفرعي الصحة والخطأ (واللذان لا يُنفذان تتابعيًا) يُلحق باستخدام مجمع خاص #py("parallel_instruction_sequences") موصف في القسم @sec:combining-instruction-sequences.

#subheading([ترجمة التسلسلات])

ترجمة
#idx("compiler for Python", sub: "sequences of statements")
تسلسلات العبارات تناظر تقييمها في مُقيِّم التحكم الصريح مع استثناء واحد: إذا ظهرت تعليمة إرجاع في أي مكان في تسلسل، فإننا نعاملها كما لو كانت العبارة الأخيرة.
تُترجم كل عبارة في التسلسل—العبارة الأخيرة (أو تعليمة إرجاع) مع الارتباط المخصص للتسلسل، والعبارات الأخرى مع الارتباط #py("\"next\"") (لتنفيذ باقي التسلسل). تُلحق تسلسلات التعليمات للعبارات الفردية لتشكيل تسلسل تعليمات واحد، بحيث يُحفظ #py("env") (المطلوب لباقي التسلسل) و #py("continue") (المطلوب محتملاً للارتباط في نهاية التسلسل).#footnote[سيلزم المسجّل #py("continue") لارتباط #py("\"return\"")، والذي يمكن أن ينتج عن ترجمة بواسطة #py("compile_and_go") (القسم @sec:interfacing-compiled-code).]
#idx("compilesequence", decl: true)
#snippet(```python
function compile_sequence(seq, target, linkage) {
    return is_empty_sequence(seq)
           ? compile_literal(make_literal(undefined), target, linkage)
           : is_last_statement(seq) ||
                 is_return_statement(first_statement(seq))
           ? compile(first_statement(seq), target, linkage)
           : preserving(list("env", "continue"),
                 compile(first_statement(seq), target, "next"),
                 compile_sequence(rest_statements(seq),
                                  target, linkage));
}
```)

معاملة تعليمة الإرجاع كما لو كانت العبارة الأخيرة في تسلسل تتجنب ترجمة أي "كود ميت" (#en[dead code]) بعد تعليمة الإرجاع لا يمكن تنفيذه أبدًا.
إزالة فحص #py("is_return_statement") لا تغير سلوك البرنامج الهدف؛ ومع ذلك، هناك العديد من الأسباب لعدم ترجمة الكود الميت، والتي تخرج عن نطاق هذا الكتاب (الأمان، وقت الترجمة، حجم الكود الهدف، إلخ)، والعديد من المُترجِمات تعطي تحذيرات للكود الميت.#footnote[مُترجِمنا لا يكتشف كل الكود الميت. على سبيل المثال، التعليمة الشرطية التي ينتهي كلا فرعي النتيجة والبديل فيها بتعليمة إرجاع لن تمنع ترجمة العبارات اللاحقة. انظر التمرينين @ex:dead-code و @ex:append_return_undefined.]<foot:dead-code>

#subheading([ترجمة الكتل])

تُترجم
#idx("compiler for Python", sub: "blocks")
الكتلة بسبق تعليمة #py("assign") لجسم الكتلة المُترجَم. يوسع الإسناد البيئة الحالية بإطار يربط الأسماء المُعلنة في الكتلة بالقيمة #py("\"*unassigned*\""). هذه العملية تتطلب وتعدل المسجّل #py("env").
#idx("compileblock", decl: true)#idx("scanning out declarations", sub: "in compiler")
#snippet(```python
function compile_block(stmt, target, linkage) {
    const body = block_body(stmt);
    const locals = scan_out_declarations(body);
    const unassigneds = list_of_unassigned(locals);
    return append_instruction_sequences(
               make_instruction_sequence(list("env"), list("env"),
                   list(assign("env", list(op("extend_environment"),
                                           constant(locals),
                                           constant(unassigneds),
                                           reg("env"))))),
               compile(body, target, linkage));
}
```)

#subheading([ترجمة تعبيرات لامدا])

تعبيرات لامدا
#idx("compiler for Python", sub: "lambda expressions")
تبني الدوال.
يجب أن يكون للكود الهدف لـ تعبير لامدا الشكل:

#syntax(metaphrase[بناء كائن الدالة وإسناده للمسجّل المستهدف], "
", metaphrase[الارتباط])

عندما نترجم تعبير لامدا، نولد أيضًا الكود لـ جسم الدالة. وعلى الرغم من أن الجسم لن يُنفذ عند وقت بناء الدالة، فمن المريح إدراج الكود للجسم بعد الكود الخاص بتعبير لامدا مباشرة.
إذا كان الارتباط لتعبير لامدا تسمية أو #py("\"return\"")، فهذا مناسب. ولكن إذا كان الارتباط #py("\"next\"")، فسنحتاج إلى التخطي حول الكود الخاص بجسم الدالة باستخدام ارتباط يقفز إلى تسمية تم إدراجها بعد الجسم. وبالتالي يملك الكود الهدف الشكل:

#syntax(metaphrase[بناء كائن الدالة وإسناده للمسجّل المستهدف], "
", metaphrase[الكود للارتباط المعطى], " ", meta("أو"), " go_to(label(\"after_lambda\"))
", metaphrase[ترجمة جسم الدالة], "
\"after_lambda\"
	  ")

تولد الدالة #py("compile_lambda_expression") الكود لبناء كائن الدالة متبوعًا بالكود لجسم الدالة. سيتم بناء كائن الدالة في وقت التشغيل بدمج البيئة الحالية (البيئة عند نقطة الإعلان) مع نقطة الإدخال لجسم الدالة المُترجَم (تسمية مُولدة حديثاً).#footnote[نحتاج إلى عمليات آلة لتنفيذ بنية بيانات لتمثيل الدوال المُترجَمة، بشكل يناظر البنية للدوال المركبة الموصوفة في القسم @sec:eval-data-structures:
#idx("makecompiledfunction", decl: true)#idx("iscompiledfunction", decl: true)#idx("compiledfunctionentry", decl: true)#idx("compiledfunctionenv", decl: true)
#snippet(```python
function make_compiled_function(entry, env) {
    return list("compiled_function", entry, env);
}
function is_compiled_function(fun) {
    return is_tagged_list(fun, "compiled_function");
}
function compiled_function_entry(c_fun) {
    return head(tail(c_fun));
}
function compiled_function_env(c_fun) {
    return head(tail(tail(c_fun)));
}
```)]<foot:compiler-ops>
#idx("compilelambdaexpression", decl: true)
#snippet(```python
function compile_lambda_expression(exp, target, linkage) {
    const fun_entry = make_label("entry");
    const after_lambda = make_label("after_lambda");
    const lambda_linkage =
            linkage === "next" ? after_lambda : linkage;
    return append_instruction_sequences(
               tack_on_instruction_sequence(
                   end_with_linkage(lambda_linkage,
                       make_instruction_sequence(list("env"),
                                                 list(target),
                           list(assign(target,
                                    list(op("make_compiled_function"),
                                         label(fun_entry),
                                         reg("env")))))),
                   compile_lambda_body(exp, fun_entry)),
               after_lambda);
}
```)

تستخدم الدالة #py("compile_lambda_expression") المجمع الخاص #py("tack_on_instruction_sequence") (من القسم @sec:combining-instruction-sequences) بدلاً من #py("append_instruction_sequences") لإلحاق جسم الدالة بكود تعبير لامدا، لأن الجسم ليس جزءًا من تسلسل التعليمات الذي سيتم تنفيذه عند الدخول في التسلسل المركب؛ بل إنه في التسلسل فقط لأن ذلك كان مكانًا مريحًا لوضعه فيه.

تبني الدالة #py("compile_lambda_body") الكود لجسم الدالة. يبدأ هذا الكود بتسمية لنقطة الإدخال. ثم تأتي التعليمات التي ستتسبب في تبديل بيئة التقييم في وقت التشغيل إلى البيئة الصحيحة لتقييم جسم الدالة—أي بيئة الدالة، ممتدة لتشمل روابط المعاملات بالوسائط التي تم استدعاء الدالة بها. بعد ذلك يأتي الكود لجسم الدالة، معززاً لضمان انتهائه بتعليمة إرجاع.

يُترجم الجسم المعزز مع الهدف #py("val") بحيث توضع قيمة إرجاعه في #py("val"). واصف الارتباط الـمُمرر إلى هذه الترجمة غير ذي صلة، حيث سيتم تجاهله.#footnote[جسم الدالة المعزز هو تسلسل ينتهي بتعليمة إرجاع.
ترجمة تسلسل من العبارات تستخدم الارتباط #py("\"next\"") لجميع عباراتها المكونة باستثناء الأخيرة، والتي تستخدم لها الارتباط المعطى.
في هذه الحالة، العبارة الأخيرة هي تعليمة إرجاع، وكما سنرى في القسم @sec:compiling-combinations، فإن تعليمة الإرجاع تستخدم دائمًا واصف الارتباط #py("\"return\"") لتعبير إرجاعها. وبالتالي فإن جميع أجسام الدوال ستنتهي بارتباط #py("\"return\"")، وليس الـ #py("\"next\"") الذي نمرره كوسيط ارتباط لـ #py("compile") في #py("compile_lambda_body").]
نظرًا لأن وسيط الارتباط مطلوب، فإننا نختار #py("\"next\"") بشكل عشوائي.

#idx("compilelambdabody", decl: true)
#snippet(```python
function compile_lambda_body(exp, fun_entry) {
    const params  = lambda_parameter_symbols(exp);
    return append_instruction_sequences(
        make_instruction_sequence(list("env", "fun", "argl"),
                                  list("env"),
            list(fun_entry,
                 assign("env",
                        list(op("compiled_function_env"),
                             reg("fun"))),
                 assign("env",
                        list(op("extend_environment"),
                             constant(params),
                             reg("argl"),
                             reg("env"))))),
        compile(append_return_undefined(lambda_body(exp)),
                "val", "next"));
}
```)

لضمان انتهاء جميع الدوال بتنفيذ تعليمة إرجاع، تُلحق #py("compile_lambda_body") بجسم لامدا تعليمة إرجاع تعبير إرجاعها هو القيمة الصريحة #py("undefined").
وللقيام بذلك، تستخدم الدالة #py("append_return_undefined")، والتي تبني التمثيل المعلّم في قائمة (من القسم @sec:representing-expressions) لتسلسل يتكون من الجسم وعبارة #py("return undefined;").

#snippet(```python
function append_return_undefined(body) {
    return list("sequence", list(body,
                                 list("return_statement",
                                      list("literal", undefined))));
}
```)

هذا التحويل البسيط لأجسام لامدا هو طريقة ثالثة لضمان أن الدالة التي لا ترجع صراحة تملك قيمة الإرجاع
#idx("return value", sub: "undefined as")
#py("undefined").
في المُقيِّم ما فوق الدائري، استخدمنا كائن قيمة إرجاع، والذي لعب دورًا أيضًا في إيقاف تقييم التسلسل.
في مُقيِّم التحكم الصريح، واصلت الدوال التي لم ترجع صراحة إلى نقطة إدخال تخزن #py("undefined") في #py("val").
انظر التمرين @ex:append_return_undefined لطريقة أكثر أنق لـ معالجة إدراج تعليمات الإرجاع.
#anchor(<sec:append_return_undefined>)

#exercise(label-name: <ex:dead-code>, [
أشارت الحاشية السفلية @foot:dead-code إلى أن المُترجِم لا يكتشف جميع حالات
#idx("compiler for Python", sub: "dead code analysis")
الكود الميت. ما الذي يتطلبه الأمر من مُترجِم لاكتشاف جميع حالات الكود الميت؟

تلميح: تعتمد الإجابة على كيفية تعريفنا للكود الميت. أحد التعاريف الممكنة (والمفيدة) هو "الكود الذي يلي تعليمة إرجاع في تسلسل"—ولكن ماذا عن الكود في فرع النتيجة لـ #py("if (false)")$dots.h$ أو الكود الذي يلي استدعاءً لـ #py("run_forever()") في التمرين @ex:halting-theorem؟
])

#exercise(label-name: <ex:append_return_undefined>, [
التصميم الحالي لـ #py("append_return_undefined") خام بعض الشيء: فهو يلحق دائمًا #py("return undefined;") بجسم لامدا، حتى لو كانت هناك تعليمة إرجاع بالفعل في كل مسار تنفيذ للجسم. أعد كتابة #py("append_return_undefined") بحيث تدرج #py("return undefined;") في نهاية تلك المسارات فقط التي لا تحتوي على تعليمة إرجاع. اختبر حلك على الدوال أدناه، مستبدلاً أي تعبيرات لـ $e_(1)$ و $e_(2)$ وأي عبارات (غير إرجاع) لـ $s_(1)$ و $s_(2)$.
في #py("t")، يجب إضافة تعليمة إرجاع إما عند كلا الـ #py("(*)") أو عند #py("(**)") فقط.
في #py("w") و #py("h")، يجب إضافة تعليمة إرجاع عند إحدى الـ #py("(*)")'s.
في #py("m")، لا يجب إضافة أي تعليمة إرجاع.

#syntax("
function t(b) {   function w(b) {    function m(b) {    function h(b1, b2) {
   if (b) {          if (b) {           if (b) {           if (b1) {
      ", $s_(1)$, "              return ", $e_(1);$, "         return ", $e_(1);$, "         return ", $e_(1)$, ";
      (*)                                                  } else {
   } else {          } else {           } else {              if (b2) {
      ", $s_(2)$, "               ", $s_(1)$, "               return ", $e_(2)$, ";             ", $s_(1)$, "
      (*)               (*)                                      (*)
   }                 }                  }                     } else {
   (**)              (*)                                         return ", $e_(2)$, ";
}                 }                  }                        }
                                                               (*)
                                                            }
                                                            (*)
                                                         }
	  ")
])
