// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([جوهر المُقيِّم])

#anchor(<sec:core-of-evaluator>)

#sicp-figure(image("/images/img_javascript/ch4-Z-G-1.svg", width: 70%), caption: [تكشف دورة #idx("metacircular evaluator for Python", sub: "evaluate–apply cycle") #py("evaluate")–#py("apply") عن جوهر لغة الحاسوب.], label-name: <fig:eval-apply>)

يمكن وصف عملية
#idx("metacircular evaluator for Python", sub: "evaluate and apply")
التقييم بأنها تفاعل بين دالتين:
#py("evaluate")
و #py("apply").

#subheading([الدالة #py("evaluate")])

تأخذ الدالة #py("evaluate")
كوسائط
#emph[مكون] برنامج—عبارة أو تعبير#footnote[لا داعي للتمييز بين العبارات والتعبيرات في مـُقيِّمنا. على سبيل المثال، لا نفرق بين التعبيرات وعبارات التعبير؛ فنحن نمثلها على نحو مطابق وبالتالي يـُتعامل معها بالطريقة نفسها بواسطة الدالة #py("evaluate"). وبالمثل، لا يفرض مـُقيِّمنا القيد النحوي لـ Python بأن العبارات لا يمكن أن تظهر داخل التعبيرات بخلاف تعبيرات دالة غير مسماة (#en[lambda]).]—وبيئة.
وتصنف
المكون
وتوجه تقييمه.
الدالة #py("evaluate")
مـُهيكلة كتحليل حالات للنوع النحوي لـ
المكون
المراد تقييمه. وللحفاظ على عمومية
الدالة،
نعبر عن تحديد نوع
المكون
تجريدياً، دون إجراء أي
#idx("metacircular evaluator for Python", sub: "component representation")
التزام بتمثيل معين للأنواع المختلفة لـ
المكونات.
ولكل نوع من
المكونات
#emph[دالة شرطية نحوية]
تختبره ووسيلة تجريدية لاختيار أجزائه. وهذا
#idx("metacircular evaluator for Python", sub: "data abstraction in")
#idx("abstract syntax", sub: "in metacircular evaluator")
#emph[البناء النحوي التجريدي]
يجعل من السهل رؤية كيف يمكننا تغيير البناء النحوي للغة باستخدام المـُقيِّم نفسه، ولكن مع مجموعة مختلفة من الدوال النحوية.

#subsubheading([التعبيرات الأوليّة])

- بالنسبة لـ #idx("expression", sub: "literal") #idx("literal expression") التعبيرات الحرفية، مثل الأعداد، ترجع #py("evaluate") قيمتها.
- يجب على الدالة #py("evaluate") البحث عن الأسماء في البيئة للعثور على قيمها.

#subsubheading([التركيبات])

- بالنسبة لـ تطبيق الدالة، يجب على #py("evaluate") تقييم تعبير الدالة وتعبيرات الوسائط للتطبيق عودياً. وتـُمَرَّر الدالة والوسائط الناتجة إلى #py("apply")، والتي تتعامل مع تطبيق الدالة الفعلي.
- تـُحوَّل تركيبات المعاملات إلى تطبيق دالة ثم تـُقيَّم.

#subsubheading([الأشكال النحوية])

- يتطلب التعبير أو العبارة الشرطية معالجة خاصة لأجزائها، حتى تـُقيَّم النتيجة إذا كانت الدالة الشرطية صحيحة، وإلا تـُقيَّم البديلة.
- يجب تحويل تعبير الدالة غير المسماة (#en[lambda]) إلى دالة قابلة للتطبيق بـ حزم البارامترات والجسم المحددين بـ تعبير #en[lambda] مع بيئة التقييم.
- يتطلب تسلسل العبارات تقييم مكوناته بالترتيب الذي تظهر به.
- تتطلب الكتلة تقييم جسمها في بيئة جديدة تعكس جميع الأسماء المـُعلنة داخل الكتلة.
- يجب أن تنتج عبارة الإرجاع قيمة تصبح نتيجة استدعاء الدالة الذي أدى إلى تقييم عبارة الإرجاع.
- يـُحوَّل تعريف الدالة إلى إعلان ثابت ثم يـُقيَّم.
- يجب على إعلان الثابت أو المتغير أو الإسناد استدعاء #py("evaluate") عودياً لحساب القيمة الجديدة المراد ربطها بالاسم الجاري إعلانه أو إسناده. ويجب تعديل البيئة لتعكس القيمة الجديدة للاسم.

إليك إعلان
#py("evaluate"):

#idx("evaluate (metacircular)", decl: true)
#snippet(```python
def evaluate(component, env):
  if is_literal(component):
    return literal_value(component)
  elif is_name(component):
    return lookup_symbol_value(symbol_of_name(component), env)
  elif is_application(component):
    return apply(evaluate(function_expression(component), env),
                          llist_of_values(arg_expressions(component),
                                          env))
  elif is_operator_combination(component):
    return evaluate(operator_combination_to_application(component),
                    env)
  elif is_conditional(component):
    return eval_conditional(component, env)
  elif is_lambda_expression(component):
    return make_function(lambda_parameter_symbols(component),
                         make_return_statement(lambda_body(component)),
                         env)
  elif is_sequence(component):
    return eval_sequence(sequence_statements(component), env)
  elif is_return_statement(component):
    return eval_return_statement(component, env)
  elif is_function_definition(component):
    return evaluate(function_def_to_assignment(component), env)
  elif is_assignment(component):
    return eval_assignment(component, env)
  else:    return error("unknown syntax -- evaluate", component)
```)

للتوضيح،
تـُمّ تنفيذ #py("evaluate")
كـ
#idx("data-directed programming", sub: "case analysis vs.")
#idx("case analysis", sub: "data-directed programming vs.")
تحليل حالات باستخدام
العبارات الشرطية.
والعيب في هذا هو أن دالتنا
تتعامل مع عدد قليل فقط من الأنواع القابلة للتمييز لـ
العبارات و
التعبيرات، ولا يمكن تعريف أنواع جديدة دون تعديل إعلان #py("evaluate").
في معظم
تنفيذات المـُفسِّرات،
يتم التوجيه حسب نوع
المكون
بأسلوب موجه بالبيانات. هذا يسمح للمستخدم بـ إضافة أنواع جديدة من المكونات التي يمكن لـ #py("evaluate") تمييزها، دون تعديل إعلان #py("evaluate") نفسه. (انظر التمرين @ex:data-directed-eval.)

يـُتعامل مع تمثيل الأسماء بواسطة تجريدات البناء النحوي. داخلياً،
يستخدم المـُقيِّم السلاسل النصية لتمثيل الأسماء، ونشير إلى مثل هذه السلاسل النصية بـ
#idx("symbol(s)", sub: "representing names in metacircular evaluator")
#emph[الرموز] (#en[symbols]). والدالة
#py("symbol_of_name") المستخدمة في
#py("evaluate") تستخرج من
الاسم الرمز الذي يـُمثَّل به.

#subheading([التطبيق (#en[Apply])])

تأخذ الدالة #py("apply")
وسيطين،
دالة
وقائمة مترابطة من الوسائط التي يجب تطبيق
الدالة
عليها.
تصنف الدالة #py("apply")
الدوال
إلى نوعين: حيث تستدعي
#idx("applyprimitivefunction")
#py("apply_primitive_function")
لتطبيق الدوال الأوليّة؛ وتطبق الدوال
المركبة
بتقييم جسم الدالة.
والبيئة لتقييم جسم دالة
مركبة
تـُبنى بـ توسيع البيئة الأساسية المحمولة بواسطة
الدالة
لتتضمن إطاراً يربط بارامترات
الدالة
بالوسائط التي يجب تطبيق
الدالة
عليها.
إليك
إعلان
#py("apply"):
#idx("apply (metacircular)", decl: true)
#snippet(```python
def apply(fun, arguments):
    if is_primitive_function(fun):
        return apply_primitive_function(fun, arguments)
    elif is_compound_function(fun):
        body = function_body(fun)
        locals = local_variables(body)
    unassigneds = llist_of_unassigned(locals)
        result = evaluate(body,
                          extend_environment(
                              append(function_parameters(fun),
                         locals),
                              append(arguments,
                         unassigneds),
                              function_environment(fun)))
        return (return_value_content(result)
                if is_return_value(result)
                else None)
    else:
        return error("unknown function type -- apply", fun)
```)

من أجل إرجاع قيمة، تحتاج دالة Python إلى تقييم
#idx("metacircular evaluator for Python", sub: "return value")
#idx("return value", sub: "representation in metacircular evaluator")
عبارة إرجاع. وإذا انتهت دالة دون تقييم عبارة إرجاع، فإن القيمة
#idx("return value", sub: "None as")
#py("None") تـُرجع.
وللتمييز بين الحالتين، فإن تقييم عبارة الإرجاع سيغلف نتيجة تقييم تعبير إرجاعها في
#emph[قيمة إرجاع]. وإذا أدى تقييم جسم الدالة إلى مثل قيمة الإرجاع هذه، فيتم استرداد محتوى قيمة الإرجاع؛ وإلا تـُرجع القيمة
#py("None").#footnote[هذا الاختبار عملية مؤجلة، وبالتالي فإن مـُقيِّمنا سيعطي مساراً لعملية
عودية حتى لو كان البرنامج المـُفسَّر سيعطي مساراً لعملية
تكرارية وفقاً للوصف في
القسم @sec:recursion-and-iteration. وبعبارة أخرى، فإن تنفيذنا للمـُقيِّم دائري التجريد لـ Python ليس
#idx("apply (metacircular)", sub: "tail recursion and")
#idx("metacircular evaluator for Python", sub: "tail recursion and")
#idx("tail recursion", sub: "metacircular evaluator and")
عودي الذيل (#en[tail-recursive]).
توضح القسمان @sec:tail-recursion-return
و @sec:compiling-combinations
كيفية تحقيق العودية الذيلية باستخدام آلة مسجلات.]<foot:apply>

الدالة #py("scan_out_declarations")
#idx("scanning out declarations", sub: "in metacircular evaluator")
تجمع قائمة بجميع الرموز التي تمثل الأسماء Mـُعلنة في الجسم.
وتستخدم
#py("declaration_symbol")
لاسترداد الرمز الذي يمثل الاسم
من عبارات الإعلان التي تعثر عليها.
#idx("scanoutdeclarations", decl: true)
#snippet(```python
def scan_out_declarations(component):
    if is_sequence(component):
        return reduce(append,
                      None,
                      map(scan_out_declarations,
                          sequence_statements(component)))
    elif is_declaration(component):
        return llist(declaration_symbol(component))
    else:
        return None
```)

نحن نتجاهل الإعلانات المتداخلة في تعاريف الدوال، لأن تقييم تعريف الدالة يتكفل بها.

#subheading([وسائط الدوال])

عندما تعالج
#py("evaluate")
تطبيق
دالة، فإنها تستخدم
#py("llist_of_values")
لإنتاج قائمة الوسائط التي يجب تطبيق
الدالة
عليها.
تأخذ الدالة #py("llist_of_values")
كوسيط تعبيرات
الوسائط للتطبيق.
وتقُيم كل
تعبير وسيط
وترجع
قائمة بالقيم المقابلة:#footnote[اخترنا تنفيذ #py("llist_of_values") باستخدام #idx("metacircular evaluator for Python", sub: "higher-order functions in") #idx("higher-order functions", sub: "in metacircular evaluator") الدالة عالية الرتبة #py("map")، وسنستخدم الدوال عالية الرتبة الشائعة في أماكن أخرى أيضاً. ومع ذلك، يمكن تنفيذ المـُقيِّم دون أي استخدام لـ الدوال عالية الرتبة (وبالتالي يمكن كتابته بلغة لا تحتوي على دوال عالية الرتبة)، حتى لو كانت اللغة التي يدعمها تتضمن دوالاً عالية الرتبة. على سبيل المثال، يمكن كتابة #py("llist_of_values") دون #py("map") كما يلي: #idx("llistofvalues", sub: "without higher-order functions", decl: true) #snippet(```python def llist_of_values(exps, env): if is_null(exps): return None else: pair(evaluate(head(exps), env), llist_of_values(tail(exps), env))) ```)]<foot:mceval-higher-order>
#idx("llistofvalues", decl: true)
#snippet(```python
def llist_of_values(exps, env):
    return map(lambda arg: evaluate(arg, env), exps)
```)

#subheading([الشرطيات])

تُقيّم الدالة #py("eval_conditional")
الجزء الشرطي لـ
مكون شرطي
في البيئة المعطاة. وإذا كانت النتيجة صحيحة،
يـُقيَّم النتيجة، وإلا تـُقيَّم البديلة:
#idx("evalconditional (metacircular)", decl: true)
#snippet(```python
def eval_conditional(component, env):
    if is_truthy(evaluate(conditional_predicate(component), env)):
        return (evaluate(conditional_consequent(component), env)
    else: return evaluate(conditional_alternative(component), env))
```)

لاحظ أن المـُقيِّم لا يحتاج إلى التمييز بين التعبيرات الشرطية والعبارات الشرطية.

استخدام
#idx("istruthy")
#idx("truthiness")
#py("is_truthy")
في
#py("eval_conditional")
#idx("metacircular evaluator for Python", sub: "implemented language vs. implementation language")
يسلط الضوء على قضية الصلة بين اللغة المـُنفذة
ولغة التنفيذ. فـالـ
#py("conditional_predicate")
يـُقيَّم في اللغة الجاري تنفيذها وبالتالي يعطي قيمة في
تلك اللغة. ودالة الشرط المـُفسِّرة
#py("is_truthy")
تترجم تلك القيمة إلى قيمة يمكن اختبارها بـ التعبير
الشرطي
في لغة التنفيذ: وقد لا يكون التمثيل دائري التجريد للحقيقة هو نفسه مثل ذلك الخاص بـ Python الأساسية.#footnote[في هذه الحالة، اللغة الجاري تنفيذها ولغة التنفيذ هما نفسيهما. وتأمل معنى #py("is_truthy") هنا يعطي #idx("consciousness, expansion of") توسيعاً للوعي دون إساءة استخدام المواد.]

#subheading([التسلسلات])

تـُستخدم الدالة #py("eval_sequence")
بواسطة #py("evaluate")
لتقييم تسلسل العبارات في المستوى الأعلى، أو في جسم دالة، أو في فرع عبارة شرطية.
وتأخذ كوسائط تسلسلاً من العبارات وبيئة، وتُقيّم العبارات بالترتيب الذي تحدث به. والقيمة المرجعة هي #py("None")،
باستثناء أنه إذا أدى تقييم أي عبارة في التسلسل إلى قيمة إرجاع، فإن تلك القيمة تـُرجع وتـُتجاهل العبارات اللاحقة.
#idx("evalsequence", decl: true)
#snippet(```python
def eval_sequence(stmts, env):
    for_each(lambda stmt: evaluate(stmt, env), stmts)
    return None
```)

#subheading([عبارات الإرجاع])

تـُستخدم الدالة #py("eval_return_statement")
لتقييم
#idx("return statement", sub: "handling in metacircular evaluator")
عبارات الإرجاع. وكما رأينا في
#py("apply") وتقييم
التسلسلات، فإن نتيجة تقييم عبارة الإرجاع يجب أن تكون قابلة للتمييز حتى يتمكن تقييم جسم الدالة من الإرجاع فوراً، حتى لو كانت هناك عبارات بعد عبارة الإرجاع. ولهذا الغرض،
فإن تقييم عبارة الإرجاع يغلف نتيجة تقييم تعبير الإرجاع في كائن قيمة إرجاع.#footnote[#idx("metacircular evaluator for Python", sub: "tail recursion and")
#idx("tail recursion", sub: "metacircular evaluator and")
تطبيق الدالة #py("make_return_value") على نتيجة تقييم تعبير الإرجاع ينشئ عملية مؤجلة، بالإضافة إلى العملية المؤجلة المنشأة بواسطة #py("apply"). انظر الحاشية @foot:apply للتفاصيل.]
#idx("evalreturnstatement", decl: true)
#snippet(```python
def eval_return_statement(component, env):
    return make_return_value(evaluate(return_expression(component),
                                      env))
```)

#subheading([الإسنادات والإعلانات])

تتعامل
الدالة #py("eval_assignment")
مع الإسنادات لـ
الأسماء. (لتسطير عرض مـُقيِّمنا، فنحن نسمح بالإسناد ليس فقط للمتغيرات ولكن أيضاً—على نحو خاطئ—للثوابت. يوضح التمرين @ex:mutable كيف يمكننا التمييز بين الثوابت والمتغيرات ومنع الإسناد للثوابت.)
تستدعي الدالة #py("eval_assignment") الدالة #py("evaluate") على تعبير القيمة للعثور على القيمة المراد إسنادها وتستدعي #py("assignment_symbol") لاسترداد الرمز الذي يمثل الاسم من الإسناد. وتنقل الدالة #py("eval_assignment") الرمز والقيمة إلى #py("assign_symbol_value") ليتم تثبيتها في البيئة المحددة. ويرجع تقييم الإسناد القيمة التي تـُمّ إسنادها.
#idx("evalassignment", decl: true)
#snippet(```python
def eval_assignment(component, env):
    value = evaluate(assignment_value_expression(component), env)
    assign_symbol_value(assignment_symbol(component), value, env)
    return value
```)

تـُعرف كل من إعلانات الثوابت والمتغيرات بواسطة الدالة الشرطية النحوية
#py("is_declaration").
وتـُعامل بطريقة مماثلة لـ الإسنادات، لأن #py("eval_block")
قد ربطت بالفعل رموزها بـ #py("\"*unassigned*\"")
في البيئة الحالية.
وتقييمها يستبدل #py("\"*unassigned*\"")
بنتيجة تقييم تعبير القيمة.
#idx("evaldeclaration", decl: true)
#snippet(```python
def eval_declaration(component, env):
    assign_symbol_value(
        declaration_symbol(component),
        evaluate(declaration_value_expression(component), env),
        env)
    return None
```)

تتحدد نتيجة تقييم جسم دالة بـ عبارات الإرجاع، وبالتالي فإن قيمة الإرجاع
#py("undefined") في
#py("eval_declaration") تهم فقط عندما يحدث الإعلان في المستوى الأعلى،
خارج أي جسم دالة. وهنا نستخدم قيمة الإرجاع
#py("undefined") لتسطير العرض؛ ويصف التمرين @ex:value_producing النتيجة الحقيقية لتقييم المكونات في المستوى الأعلى في Python.

#anchor(<foot:value_producing_2>)

#idx("metacircular evaluator for Python", sub: "evaluate and apply")

#exercise(label-name: <ex:arg-eval-order>, [
لاحظ أنه لا يمكننا معرفة ما إذا كان المـُقيِّم دائري التجريد
#idx("order of evaluation", sub: "in metacircular evaluator")
#idx("metacircular evaluator for Python", sub: "order of argument evaluation")
يُقيّم تعبيرات الوسائط من اليسار إلى اليمين أم من اليمين إلى اليسار.
فـ ترتيب تقييمه مـُوروث من JavaScript الأساسية:
فإذا كانت الوسائط لـ #py("pair") في
#py("map") تـُقيَّم من
اليسار إلى اليمين، فإن #py("llist_of_values")
ستُقيّم تعبيرات الوسائط من اليسار إلى اليمين؛ وإذا كانت
الوسائط لـ #py("pair")
تـُقيَّم من اليمين إلى اليسار، فإن
#py("llist_of_values") ستُقيّم
تعبيرات الوسائط من اليمين إلى اليسار.

اكتب نسخة من #py("llist_of_values")
تُقيّم تعبيرات الوسائط من اليسار إلى اليمين بغض النظر عن
ترتيب التقييم في JavaScript الأساسية. واكتب أيضاً نسخة من
#py("llist_of_values") تُقيّم
تعبيرات الوسائط من اليمين إلى اليسار.
])
