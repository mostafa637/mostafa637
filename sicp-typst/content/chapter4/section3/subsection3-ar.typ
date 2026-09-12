// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([تنفيذ مُقَيِّم #py("amb")], label-name: <sec:amb-implementation>)

#idx("مُقَيِّم غير حتمي")

قد يرجع تقييم برنامج #en[Python] اعتيادي قيمة، أو قد لا ينتهي أبدًا، أو قد يطلق خطأً.
أما في #en[Python] غير الحتمية، فإن تقييم برنامج ما قد ينتج عنه بالإضافة إلى ذلك اكتشاف طريق مسدود، وفي هذه الحالة يجب أن يتراجع التقييم إلى نقطة اختيار سابقة. ويتعقد تفسير #en[Python] غير الحتمية بسبب هذه الحالة الإضافية.

سنبني مُقَيِّم #py("amb") لـ #en[Python] غير الحتمية عن طريق تعديل
#idx("مُقَيِّم مُحَلِّل", sub: "كأساس للمقييم غير الحتمي")
المُقيِّم المُحَلِّل من القسم @sec:separating-analysis.#footnote[اخترنا تنفيذ المُقيِّم الكسول في القسم @sec:lazy-evaluation كتعديل للمُقَيِّم دائري التجريد الاعتيادي من القسم @sec:core-of-evaluator. وعلى النقيض من ذلك، سنبني مُقَيِّم #py("amb") على المُقيِّم المُحَلِّل من القسم @sec:separating-analysis، لأن دوال التنفيذ في ذلك المُقيِّم توفر إطارًا مريحًا لتنفيذ التتبع التراجعي.] وكما هو الحال في المُقيِّم المُحَلِّل، يتم تقييم المكوّن عن طريق استدعاء
#idx("executionfunction", sub: "في المُقيِّم غير الحتمي")
دالة تنفيذ ناتجة عن تحليل ذلك المكوّن. وسيكون الفارق بين تفسير #en[Python] العادية وتفسير #en[Python] غير الحتمية كليًا في دوال التنفيذ.

#subheading([دوال التنفيذ والمتابعات])

#idx("متابعة", sub: "في المُقيِّم غير الحتمي")

تذكر أن
#idx("execution function", sub: "في المُقيِّم غير الحتمي")
دوال التنفيذ للمُقَيِّم الاعتيادي تأخذ وسيطًا واحدًا: بيئة التنفيذ. وعلى النقيض من ذلك، تأخذ دوال التنفيذ في مُقَيِّم #py("amb") ثلاثة وسائط: البيئة، ودالتان تُسميان #emph[دوال المتابعة] (#en[continuation functions]).
وينتهي تقييم المكوّن باستدعاء إحدى هاتين المتابعتين: فإذا نتج عن التقييم قيمة، تُستدعى
#idx("متابعة النجاح (مقييم غير حتمي)")
#emph[متابعة النجاح] (#en[success continuation]) بتلك القيمة؛ وإذا نتج عن التقييم اكتشاف طريق مسدود، تُستدعى
#idx("متابعة الفشل (مقييم غير حتمي)")
#emph[متابعة الفشل] (#en[failure continuation]). وإنشاء ودعوة المتابعات المناسبة هما الآلية التي ينفذ بها المُقيِّم غير الحتمي التتبع التراجعي.

وظيفة متابعة النجاح هي استقبال قيمة والمضي قدمًا في الحساب. وجنبًا إلى جنب مع تلك القيمة، تُمرَّر إلى متابعة النجاح متابعة فشل أخرى، ليتم استدعاؤها لاحقًا إذا أدى استخدام تلك القيمة إلى طريق مسدود.

ووظيفة متابعة الفشل هي تجربة فرع آخر للعملية غير الحتمية. وجوهر اللغة غير الحتمية يكمن في حقيقة أن المكونات قد تمثل خيارات بين البدائل. ويجب أن يستمر تقييم مثل هذا المكوّن باستخدام أحد البدائل المشار إليها، حتى لو لم يكن معروفًا مسبقًا أي الخيارات يؤدي إلى نتائج مقبولة. وللتعامل مع هذا، يختار المُقيِّم أحد البدائل ويمرر هذه القيمة إلى متابعة النجاح. وجنبًا إلى جنب مع هذه القيمة، ينشئ المُقيِّم ويمرر متابعة فشل يمكن استدعاؤها لاحقًا لاختيار بديل مختلف.

يُطْلَق الفشل أثناء التقييم (أي تُستدعى متابعة الفشل) عندما يرفض برنامج المستخدم صراحةً خط الهجوم الحالي (على سبيل المثال، قد ينتج عن استدعاء #py("require") تنفيذ #py("amb()")، وهو تعبير يفشل دائمًا — انظر القسم @sec:amb). وتتسبب متابعة الفشل المتاحة في تلك النقطة في أن تختار أحدث نقطة اختيار بديلًا آخر. وإذا لم تكن هناك بدائل أخرى للنظر فيها عند نقطة الاختيار تلك، يُطْلَق فشل عند نقطة اختيار سابقة، وهكذا. كما تُستدعى متابعات الفشل بواسطة حلقة المحرك رداً على طلب #py("retry")، للعثور على قيمة أخرى للبرنامج.

بالإضافة إلى ذلك، إذا حدثت عملية ذات أثر جانبي (مثل الإسناد لمتغير) على فرع من العملية الناتجة عن اختيار ما، فقد يكون من الضروري، عندما تجد العملية طريقًا مسدودًا، إلغاء الأثر الجانبي قبل إجراء اختيار جديد. ويتحقق ذلك عن طريق جعل عملية الأثر الجانبي تنتج متابعة فشل تلغي الأثر الجانبي وتعمم الفشل.

باختصار، تُنشأ متابعات الفشل بواسطة:

- تعبيرات #py("amb") — لتوفير آلية لإجراء خيارات بديلة إذا أدى الاختيار الحالي الذي أجراه تعبير #py("amb") إلى طريق مسدود؛
- المحرك عالي المستوى — لتوفير آلية للإبلاغ عن الفشل عند نفاد الخيارات؛
- الإسنادات — لاعتراض الأخطاء وإلغاء الإسنادات أثناء التتبع التراجعي.

ويُبدَأ الفشل فقط عند مواجهة طريق مسدود. ويحدث هذا:

- إذا نفذ برنامج المستخدم #py("amb()")
- إذا كتب المستخدم #py("retry") في المحرك عالي المستوى.

تُستدعى متابعات الفشل أيضًا أثناء معالجة الفشل:

- عندما تنتهي متابعة الفشل المنشأة بواسطة إسناد من إلغاء أثر جانبي، فإنها تدعو متابعة الفشل التي اعترضتها، من أجل تعميم الفشل إلى نقطة الاختيار التي أدت إلى هذا الإسناد أو إلى المستوى الأعلى.
- عندما تنفد الخيارات من متابعة الفشل لـ #py("amb")، فإنها تدعو متابعة الفشل التي أُعْطِيَت أصلاً لـ #py("amb")، من أجل تعميم الفشل إلى نقطة الاختيار السابقة أو إلى المستوى الأعلى.

#idx("متابعة", sub: "في المُقيِّم غير الحتمي")

#subheading([بنية المُقيِّم])

دوال بناء الجملة وتمثيل البيانات لمُقَيِّم #py("amb")، وأيضًا دالة
#idx("analyze", sub: "nondeterministic")
#py("analyze") الأساسية، متطابقة مع تلك الموجودة في المُقيِّم في القسم @sec:separating-analysis، باستثناء حقيقة أننا نحتاج إلى دوال بناء جملة إضافية للتعرف على الشكل النحوي #py("amb"):
#idx("isamb", decl: true)
#snippet(```python
def is_amb(component):
    return is_tagged_list(component, "application") and is_name(function_expression(component)) and symbol_of_name(function_expression(component)) == "amb"
def amb_choices(component):
    return arg_expressions(component)
```)

نستمر في استخدام دالة التحليل في القسم @sec:representing-expressions، والتي لا تدعم #py("amb") كشكل نحوي وتتعامل بدلاً من ذلك مع #py("amb(") $dots.h$ #py(")") كتطبيق دالة. وتضمن الدالة #py("is_amb") أنه كلما ظهر الاسم #py("amb") كتعبير دالة لتطبيق ما، فإن المُقيِّم يتعامل مع «التطبيق» كنقطة اختيار غير حتمية.#footnote[مع هذا التعامل، لم يعد #py("amb") اسمًا بنطاق سليم. ولتجنب الارتباك، يجب أن نتمتنع عن التصريح عن #py("amb") كاسم في برامجنا غير الحتمية.]

يجب علينا أيضًا إضافة بند في التوجيه في #py("analyze") للتعرف على مثل هذه التعبيرات وتوليد دالة تنفيذ مناسبة:

#syntax($dots.h$, "
: is_amb(component)
? analyze_amb(component)
: is_application(component)
", $dots.h$)

الدالة عالية المستوى #py("ambeval") (المشابهة لنسخة #py("evaluate") المعطاة في القسم @sec:separating-analysis) تُحَلِّل المكوّن المعطى وتطبق دالة التنفيذ الناتجة على البيئة المعطاة، جنبًا إلى جنب مع متابعتين معطاتين:

#idx("ambeval", decl: true)
#snippet(```python
def ambeval(component, env, succeed, fail):
    return analyze(component)(env, succeed, fail)
```)

متابعة النجاح
#idx("متابعة النجاح (مقييم غير حتمي)")
#idx("متابعة", sub: "في المُقيِّم غير الحتمي")
هي دالة من وسيطين: القيمة المحصول عليها حديثًا ومتابعة فشل أخرى لاستخدامها إذا أدت تلك القيمة إلى فشل لاحق.
ومتابعة
#idx("متابعة الفشل (مقييم غير حتمي)")
الفشل هي دالة بلا وسائط. لذا فإن الشكل العام لـ
#idx("execution function", sub: "في المُقيِّم غير الحتمي")
دالة التنفيذ هو

#syntax("
# ", $mono("succeed") thin$, " is ", $mono("lambda value, fail:") dots.h$, "
# ", $mono("fail") thin$, " is ", $mono("lambda:") dots.h$, "
lambda env, succeed, fail: ", $dots.h$)

على سبيل المثال، تنفيذ

#syntax("
ambeval(", meta("component"), ",
        the_global_environment,
        lambda value, fail: value,
        lambda: \"failed\")
	  ")

سيحاول تقييم المكوّن المعطى وسيرجع إما قيمة المكوّن (إذا نجح التقييم) أو السلسلة النصية #py("\"failed\"") (إذا فشل التقييم).
واستدعاء #py("ambeval") في حلقة المحرك الموضحة أدناه يستخدم دوال متابعة أكثر تعقيدًا بكثير، والتي تواصل الحلقة وتدعم طلب #py("retry").

معظم تعقيد مُقَيِّم #py("amb") ينتج عن آليات تمرير المتابعات بينما تستدعي دوال التنفيذ بعضها البعض. وأثناء استعراض الكود التالي، يجب مقارنة كل دالة من دوال التنفيذ مع الدالة المقابلة للمُقَيِّم الاعتيادي المعطاة في القسم @sec:separating-analysis.

#subheading([التعبيرات البسيطة])

#idx("analyze...", sub: "nondeterministic")

دوال التنفيذ لأبسط أنواع التعبيرات هي نفسها أساسًا للمُقَيِّم الاعتيادي، باستثناء الحاجة إلى إدارة المتابعات. وتكتفي دوال التنفيذ بالنجاح بقيمة التعبير، ممررة متابعة الفشل التي أُعْطِيَت لها.

#snippet(```python
def analyze_literal(component):
    return lambda env, succeed, fail: (succeed(literal_value(component), fail))
```)

#snippet(```python
def analyze_name(component):
    return lambda env, succeed, fail: (succeed(lookup_symbol_value(symbol_of_name(component), env), fail))
```)

#snippet(```python
def analyze_lambda_expression(component):
    params = lambda_parameter_symbols(component)
    bfun = analyze(lambda_body(component))
    return lambda env, succeed, fail: (succeed(make_function(params, bfun, env), fail))
```)

لاحظ أن البحث عن اسم «ينجح» دائمًا.
#idx("فشل، في الحساب غير الحتمي", sub: "خطأ مقابل")
فإذا فشلت #py("lookup_symbol_value") في العثور على الاسم، فإنها تطلق خطأً كالمعتاد. ومثل هذا «الفشل» يشير إلى خطأ برلمجي — مرجع لاسم غير مربوط — ولا يشير إلى أنه ينبغي علينا تجربة اختيار غير حتمي آخر بدلاً من الاختيار الذي يُجَرَّب حاليًا.

#subheading([العبارات الشرطية والمتتاليات])

تُعَالَج العبارات الشرطية بطريقة مشابهة للمُقَيِّم الاعتيادي. فدالة التنفيذ المنشأة بواسطة #py("analyze_conditional") تستدعي دالة تنفيذ المحمول #py("pfun") مع متابعة نجاح تفرض ما إذا كانت قيمة المحمول صحيحة وتستمر لتنفيذ إما النتيجة أو البديل. وإذا فشل تنفيذ #py("pfun")، تُستدعى متابعة الفشل الأصلية للتعبير الشرطي.

#syntax("
def analyze_conditional(component):
    pfun = analyze(conditional_predicate(component))
    cfun = analyze(conditional_consequent(component))
    afun = analyze(conditional_alternative(component))
    return lambda env, succeed, fail: (pfun(env, lambda pred_value, fail2: (cfun(env, succeed, fail2) if is_truthy(pred_value) else afun(env, succeed, fail2)), fail))
")

وتُعَالَج المتتاليات بنفس الطريقة كما في المُقيِّم السابق، باستثناء الترتيبات في الدالة الفرعية #py("sequentially") المطلوبة لتمرير المتابعات. أي، لتنفيذ #py("a") تتابعيًا ثم #py("b")، نستدعي #py("a") مع متابعة نجاح تستدعي #py("b").

#syntax("
def analyze_sequence(stmts):
    def sequentially(a, b):
        return lambda env, succeed, fail: (a(env, lambda a_value, fail2: (succeed(a_value, fail2) if is_return_value(a_value) else b(env, succeed, fail2)), fail))
    def loop(first_fun, rest_funs):
        return first_fun if is_none(rest_funs) else loop(sequentially(first_fun, head(rest_funs)), tail(rest_funs))
    funs = map(analyze, stmts)
    return lambda env: (None) if is_none(funs) else loop(head(funs), tail(funs))
")

#subheading([التصريحات والإسنادات])

التصريحات حالة أخرى يجب أن نتكبد فيها بعض العناء لإدارة المتابعات، لأنه من الضروري تقييم تعبير قيمة التصريح قبل التصريح الفعلي عن الاسم الجديد. ولتحقيق ذلك، تُستدعى دالة تنفيذ قيمة التصريح #py("vfun") مع البيئة ومتابعة النجاح ومتابعة الفشل. وإذا نجح تنفيذ #py("vfun")، ونال قيمة #py("val") للاسم المصرَّح عنه، يُصرَّح عن الاسم ويُنْشَر النجاح:

#snippet(```python
def analyze_declaration(component):
    symbol = declaration_symbol(component)
    vfun = analyze(declaration_value_expression(component))
    def the_declaration(env, succeed, fail):
        def success(val, fail2):
            assign_symbol_value(symbol, val, env)
            return succeed(None, fail2)
        return vfun(env, success, fail)
    return the_declaration
```)

الإسنادات
#idx("متابعة الفشل (مقييم غير حتمي)", sub: "منشأة بواسطة الإسناد")
أكثر إثارة للاهتمام. فهذا هو المكان الأول الذي نستخدم فيه المتابعات حقًا، بدلاً من مجرد تمريرها. فدالة التنفيذ للإسنادات تبدأ مثل دالة التصريحات. فهي تحاول أولاً الحصول على القيمة الجديدة المراد إسنادها للاسم. وإذا فشل تقييم #py("vfun") هذا، يفشل الإسناد.

ومع ذلك، إذا نجحت #py("vfun")، ومضينا لإجراء الإسناد، يجب أن نأخذ بالاعتبار احتمال أن هذا الفرع من الحساب قد يفشل لاحقًا، وهو ما سيتطلب منا التراجع عن الإسناد. وبالتالي، يجب أن نرتب لإلغاء الإسناد كجزء من عملية التتبع التراجعي.#footnote[لم نكن نقلق بشأن إلغاء التصريحات، لأننا نفترض أنه لا يمكن استخدام اسم قبل تقييم تصريحه،
#idx("تصريح داخلي", sub: "في المُقيِّم غير الحتمي")
لذا فإن قيمته السابقة لا تهم.]

يتحقق ذلك عن طريق إعطاء #py("vfun") متابعة نجاح (الموسومة بالتعليق "\*1\*" أدناه) تحفظ القيمة القديمة للمتغير قبل إسناد القيمة الجديدة للمتغير والمضي قدمًا من الإسناد. ومتابعة الفشل التي تُمرَّر جنبًا إلى جنب مع قيمة الإسناد (الموسومة بالتعليق "\*2\*" أدناه) تستعيد القيمة القديمة للمتغير قبل مواصلة الفشل. أي، يوفر الإسناد الناجح متابعة فشل ستعترض أي فشل لاحق؛ وأي فشل كان يستدعي #py("fail2") لولا ذلك يدعو هذه الدالة بدلاً من ذلك، لإلغاء الإسناد قبل استدعاء #py("fail2") فعليًا.

#snippet(```python
def analyze_assignment(component):
    symbol = assignment_symbol(component)
    vfun = analyze(assignment_value_expression(component))
    def the_assignment(env, succeed, fail):
        def success(val, fail2):              # *1*
            old_value = lookup_symbol_value(symbol, env)
            assign_symbol_value(symbol, val, env)
            def restore():                    # *2*
                assign_symbol_value(symbol, old_value, env)
                return fail2()
            return succeed(val, restore)
        return vfun(env, success, fail)
    return the_assignment
```)

#subheading([عبارات الإرجاع والكتل])

تحليل عبارات الإرجاع مباشر. يُحَلَّل تعبير الإرجاع لإنتاج دالة تنفيذ. وتدعو دالة التنفيذ لعبارة الإرجاع دالة التنفيذ تلك مع متابعة نجاح تغلف قيمة الإرجاع في كائن قيمة إرجاع وتمررها إلى متابعة النجاح الأصلية.

#snippet(```python
def analyze_return_statement(component):
    rfun = analyze(return_expression(component))
    return lambda env, succeed, fail: (rfun(env, lambda val, fail2: (succeed(make_return_value(val), fail2)), fail))
```)

ودالة التنفيذ للكتل تدعو دالة تنفيذ المتن على بيئة مُمَدَّدة، دون تغيير متابعات النجاح أو الفشل.

#snippet(```python
def analyze_block(component):
    body = block_body(component)
    locals = scan_out_declarations(body)
    unassigneds = list_of_unassigned(locals)
    bfun = analyze(body)
    return lambda env, succeed, fail: (bfun(extend_environment(locals, unassigneds, env), succeed, fail))
```)

#subheading([تطبيقات الدوال])

دالة التنفيذ للتطبيقات لا تحتوي على أفكار جديدة باستثناء التعقيد التقني لإدارة المتابعات. وينشأ هذا التعقيد في #py("analyze_application")، بسبب الحاجة إلى تتبع متابعات النجاح والفشل بينما نُقَيِّم تعبيرات الوسائط. وتستخدم دالة #py("get_args") لتقييم قائمة تعبيرات الوسائط، بدلاً من #py("map") بسيطة كما في المُقيِّم الاعتيادي.

#snippet(```python
def analyze_application(component):
    ffun = analyze(function_expression(component))
    afuns = map(analyze, arg_expressions(component))
    return lambda env, succeed, fail: (ffun(env, lambda fun, fail2: (get_args(afuns, env, lambda args, fail3: (execute_application(fun, args, succeed, fail3)), fail2)), fail))
```)

#idx("analyze...", sub: "nondeterministic")

في #py("get_args")، لاحظ كيف يتحقق المرور عبر قائمة دوال التنفيذ #py("afun") وبناء قائمة #py("args") الناتجة عن طريق استدعاء كل #py("afun") في القائمة مع متابعة نجاح تدعو عوديًا #py("get_args"). ولكل استدعاء عودي لـ #py("get_args") متابعة نجاح قيمتها هي القائمة الجديدة الناتجة عن استخدام #py("pair") لإضافة الوسيط المحصول عليه حديثًا إلى قائمة الوسائط المتراكمة:

#syntax("
def get_args(afuns, env, succeed, fail):
    return succeed(None, fail) if is_none(afuns) else head(afuns)(env, lambda arg, fail2: (get_args(tail(afuns), env, lambda args, fail3: (succeed(pair(arg, args), fail3)), fail2)), fail)
")

وتطبيق الدالة الفعلي، الذي تُؤَدِّيه #py("execute_application")، يُنْجَز بنفس الطريقة للمُقَيِّم الاعتيادي، باستثناء الحاجة إلى إدارة المتابعات.
#idx("executeapplication", sub: "nondeterministic", decl: true)
#snippet(```python
def execute_application(fun, args, succeed, fail):
    return succeed(apply_primitive_function(fun, args), fail) if is_primitive_function(fun) else function_body(fun)(extend_environment(function_parameters(fun), args, function_environment(fun)), lambda body_result, fail2: (succeed(return_value_content(body_result) if is_return_value(body_result) else None, fail2)), fail) if is_compound_function(fun) else error("unknown function type - execute_application", fun)
```)

#subheading([تقييم تعبيرات #py("amb")])

الشكل النحوي #py("amb") هو العنصر الأساسي في اللغة غير الحتمية. وهنا نرى جوهر عملية التفسير والسبب في تتبع المتابعات. فدالة التنفيذ لـ #py("amb") تٌعَرِّف حلقة #py("try_next") التي تتنقل عبر دوال التنفيذ لجميع القيم الممكنة لتعبير #py("amb"). وتُستدعى كل دالة تنفيذ مع
#idx("متابعة الفشل (مقييم غير حتمي)", sub: "منشأة بواسطة amb")
متابعة فشل ستجرب البديل التالي. وعندما لا تكون هناك بدائل أخرى للتجربة، يفشل تعبير #py("amb") بالكامل.
#idx("analyzeamb", decl: true)
#snippet(```python
def analyze_amb(component):
    cfuns = map(analyze, amb_choices(component))
    def _lambda_1(env, succeed, fail):
        def try_next(choices):
            return fail() if is_none(choices) else head(choices)(env, succeed, lambda : (try_next(tail(choices))))
        return try_next(cfuns)
    return _lambda_1
```)

#subheading([حلقة المحرك])

#idx("حلقة المحرك", sub: "في المُقيِّم غير الحتمي")

حلقة المحرك لمُقَيِّم #py("amb") معقدة، بسبب الآلية التي تتيح للمستخدم إمكانية إعادة المحاولة عند تقييم برنامج ما.
ويستخدم المحرك دالة تُسمى #py("internal_loop")، والتي تأخذ كوسيط دالة
#idx("متابعة الفشل (مقييم غير حتمي)", sub: "منشأة بواسطة حلقة المحرك")
#py("retry"). والقصد هو أن يؤدي استدعاء #py("retry") إلى المضي قدمًا إلى البديل التالي غير المُجَرَّب في التقييم غير الحتمي.
وتدعو الدالة #py("internal_loop") إما #py("retry") رداً على كتابة المستخدم لـ #py("retry") في حلقة المحرك، وإما تبدأ تقييمًا جديدًا عن طريق استدعاء #py("ambeval").

ومتابعة الفشل لهذا الاستدعاء لـ #py("ambeval") تُبْلِغ المستخدم بأنه لا توجد قيم أخرى وتستدعي مجددًا حلقة المحرك.

ومتابعة النجاح للاستدعاء لـ #py("ambeval") أكثر دقة. فنطبع القيمة المحصول عليها ثم نستدعي مجددًا الحلقة الداخلية مع دالة #py("retry") ستكون قادرة على تجربة البديل التالي. ودالة #py("next_alternative") هذه هي الوسيط الثاني الذي يُمرَّر إلى متابعة النجاح. وعادة، نفكر في هذا الوسيط الثاني كمتابعة فشل لاستخدامها إذا فشل فرع التقييم الحالي لاحقًا. ولكن في هذه الحالة، أتممنا تقييمًا ناجحًا، لذا يمكننا استدعاء فرع البديل «الفاشل» من أجل البحث عن تقييمات ناجحة إضافية.

#idx("محثات", sub: "المُقيِّم غير الحتمي")#idx("driverloop", sub: "للمقييم غير الحتمي", decl: true)
#snippet(```python
input_prompt = "amb-evaluate input:"
output_prompt = "amb-evaluate value:"

def driver_loop(env):
    def internal_loop(retry):
        input = user_read(input_prompt)
        if is_none(input):
            print("evaluator terminated")
        elif input == "retry":
            return retry()
        else:
            print("Starting a new problem")
            program = parse(input)
            locals = scan_out_declarations(program)
            unassigneds = list_of_unassigned(locals)
            program_env = extend_environment(
                               locals, unassigneds, env)
            def on_success(val, next_alternative):
                user_print(output_prompt, val)
                return internal_loop(next_alternative)
            def on_failure():
                print("There are no more values of")
                print(input)
                return driver_loop(program_env)
            return ambeval(program, program_env, on_success, on_failure)
    def no_problem():
        print("There is no current problem")
        return driver_loop(env)
    return internal_loop(no_problem)
```)

الاستدعاء الأولي لـ #py("internal_loop") يستخدم دالة #py("retry") تشتكي من عدم وجود مشكلة حالية وتستأنف حلقة المحرك. وهذا هو السلوك الذي سيحدث إذا كتب المستخدم #py("retry") عندما لا يكون هناك تقييم قيد التقدم.

ونبدأ حلقة المحرك كالمعتاد، عن طريق تهيئة البيئة العالمية وتمريرها كالبيئة المحيطة بالتكرار الأول لـ #py("driver_loop").

#snippet(```python
the_global_environment = setup_environment()
driver_loop(the_global_environment)
```)

#exercise(label-name: <ex:ramb>, [
نفذ شكلًا نحويًا جديدًا #py("ramb") يشبه #py("amb") باستثناء أنه يبحث في البدائل بترتيب عشوائي، بدلاً من اليسار إلى اليمين. وضح كيف يمكن أن يساعد هذا في مشكلة أليسا في التمرين @ex:sentence-generate.
])

#exercise(label-name: <ex:permanent-set>, [
غير تنفيذ الإسناد بحيث لا يلتغى عند الفشل. على سبيل المثال، يمكننا اختيار عنصرين متميزين من قائمة وحساب عدد المحاولات المطلوبة لإجراء اختيار ناجح على النحو التالي:

#snippet(```python
count = 0
x = an_element_of("a", "b", "c")
y = an_element_of("a", "b", "c")
count = count + 1
require(x != y)
llist(x, y, count)
```)

#output(```python
count = 0
x = an_element_of("a", "b", "c")
y = an_element_of("a", "b", "c")
count = count + 1
require(x != y)
llist(x, y, count)
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
retry
```)

#output(```python
retry
```)

ما القيم التي كانت ستُعرض إذا كنا قد استخدمنا المعنى الأصلي للإسناد بدلاً من الإسناد الدائم؟
])

#exercise(label-name: <ex:if-fail>, [
سنعتدي بشكل مريع على بناء الجملة للعبارات الشرطية، عن طريق تنفيذ تركيبة بالشكل التالي:

#syntax("
if (evaluation_succeeds_take) { ", meta("statement"), " } else { ", meta("alternative"), " }
	  ")

تتيح هذه التركيبة للمستخدم التقاط فشل عبارة ما. فهي تقيم العبارة كالمعتاد وترجع كالمعتاد إذا نجح التقييم. وإذا فشل التقييم، فسيتم تقييم العبارة البديلة المعطاة، كما في المثال التالي:

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
if evaluation_succeeds_take:
    x = an_element_of(llist(1, 3, 5))
    require(is_even(x))
    x
else:
    "all odd"
```)

#output(```python
if evaluation_succeeds_take:
    x = an_element_of(llist(1, 3, 5))
    require(is_even(x))
    x
else:
    "all odd"
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
if evaluation_succeeds_take:
    x = an_element_of(llist(1, 3, 5, 8))
    require(is_even(x))
    x
else:
    "all odd"
```)

#output(```python
if evaluation_succeeds_take:
    x = an_element_of(llist(1, 3, 5, 8))
    require(is_even(x))
    x
else:
    "all odd"
```)

نفذ هذه التركيبة عن طريق توسيع مُقَيِّم #py("amb"). إرشاد: تظهر الدالة #py("is_amb") كيفية الاعتداء على بناء جملة #en[Python] الموجود من أجل تنفيذ شكل نحوي جديد.
])

#exercise(label-name: <ex:combine_permanent_if_fail>, [
مع النوع الجديد من الإسناد كما هو موصوف في التمرين @ex:permanent-set والتركيبة #syntax(" if (evaluation_succeeds_take) { ", $dots.h$, " } else { ", $dots.h$, " } ") كما في التمرين @ex:if-fail، ماذا ستكون نتيجة تقييم

#snippet(```python
pairs = None
if evaluation_succeeds_take:
    p = prime_sum_pair(llist(1, 3, 5, 8), llist(20, 35, 110))
    pairs = pair(p, pairs)
    amb()
else:
    pairs
```)
])

#exercise(label-name: <ex:require_special>, [
إذا لم نكن قد أدركنا أن
#idx("require", sub: "كشكل نحوي")
#py("require") يمكن تنفيذها كدالة اعتيادية تستخدم #py("amb")، ليتم تعريفها بواسطة المستخدم كجزء من برنامج غير حتمي، لكنا اضطررنا لتنفيذها كشكل نحوي. وهذا سيتطلب دوال بناء جملة

#snippet(```python
def is_require(component):
    return is_tagged_list(component, "require")
def require_predicate(component):
    return head(tail(component))
```)

وبندًا جديدًا في التوجيه في #py("analyze")

#snippet(```python
: is_require(component)
? analyze_require(component)
```)

وكذلك الدالة #py("analyze_require") التي تتعامل مع تعبيرات #py("require"). أكمل التعريف التالي لـ #py("analyze_require").

#syntax("
def analyze_require(component):
    pfun = analyze(require_predicate(component))
    return lambda env, succeed, fail: (pfun(env, lambda pred_value, fail2: (", metaphrase[??], " if ", metaphrase[??], " else succeed(\"ok\", fail2)), fail))
      ")
])

#idx("مُقَيِّم غير حتمي")
