// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([فصل التحليل النحوي عن التنفيذ], label-name: <sec:separating-analysis>)

#idx("تحليل نحوي، مفصول عن التنفيذ", sub: "في المقييم دائري التجريد")
#idx("مُقَيِّم مُحَلِّل")
#idx("مُقَيِّم دائري التجريد لـ Python", sub: "النسخة المُحَلِّلة")

المُقَيِّم المُنَفَّذ أعلاه بسيط، ولكنه غير كفء للغاية
#idx("مُقَيِّم دائري التجريد لـ Python", sub: "كفاءة")
#idx("كفاءة", sub: "التقييم")
لأن التحليل النحوي للمكونات يتداخل
مع تنفيذها. وبالتالي، إذا عولِج برنامج مرات عديدة،
يتم تحليل بناء جملته النحوية مرات عديدة. تأمل، على سبيل المثال، تقييم
#py("factorial(4)") باستخدام التعريف التالي لـ
#py("factorial"):

#snippet(```python
def factorial(n):
    return 1 if n == 1 else factorial(n - 1) * n
```)

في كل مرة تُستدعى فيها #py("factorial")، يجب على المُقَيِّم
تحديد أن المتن هو تعبير شرطي واستخراج المحمول.
عندئذٍ فقط يمكنه تقييم المحمول والتوجيه بناءً على قيمته. وفي كل مرة يقيم فيها التعبير
#py("factorial(n - 1) * n")،
أو التعبيرات الفرعية
#py("factorial(n - 1)")
و
#py("n - 1")،
يجب على المُقَيِّم إجراء تحليل الحالات في
#py("evaluate")
لتحديد أن التعبير هو تطبيق، ويجب عليه استخراج
تعبير الدالة وتعبيرات المعاملات.
هذا التحليل مكلف.
وإجراؤه بشكل متكرر ينطوي على هدر.

يمكننا تحويل المُقَيِّم ليكون أكثر كفاءة بدرجة كبيرة عن طريق
ترتيب الأمور بحيث يتم إجراء التحليل النحوي مرة
واحدة فقط.#footnote[هذه التقنية جزء لا يتجزأ من عملية التصريف،
والتي سنناقشها في الفصل @chap:reg. كتب
#en[Jonathan Rees] مفسر
#en[Scheme] مثل هذا في عام 1982 تقريبًا لمشروع #en[T]
#idx("Rees, Jonathan A.")
#idx("Adams, Norman I., IV")
(#en[Rees and Adams 1982]).
#idx("Feeley, Marc")
اخترع #en[Marc Feeley 1986]
(انظر أيضًا
#idx("Lapalme, Guy")
#en[Feeley and Lapalme 1987])
هذه التقنية بشكل مستقل في أطروحة الماجستير الخاصة به.] نقسم
#py("evaluate")،
التي تأخذ
مكوّنًا وببيئة، إلى جزءين.
الدالة
#py("analyze") تأخذ فقط
المكوّن.
وتجري التحليل
النحوي وترجع دالة جديدة،
#idx("دالة التنفيذ", sub: "في المقييم المحلل")
#emph[دالة التنفيذ] (#en[execution function])،
التي تغلف العمل المطلوب إنجازه في تنفيذ المكوّن المُحَلَّل.
تأخذ دالة التنفيذ بيئة كمعامل لها وتكمل التقييم. وهذا يوفر العمل لأن
#py("analyze") ستُستدعى مرة واحدة فقط على
مكوّن،
بينما قد تُستدعى دالة التنفيذ مرات عديدة.

مع الفصل بين التحليل والتنفيذ،
تصبح #py("evaluate") الآن

#idx("evaluate (metacircular)", sub: "النسخة المُحَلِّلة", decl: true)
#snippet(```python
def evaluate(component, env):
    return analyze(component)(env)
```)

نتيجة استدعاء #py("analyze") هي دالة التنفيذ
التي سيتم تطبيقها على البيئة. ودالة #py("analyze")
هي نفس تحليل الحالات المُنَفَّذ بواسطة #py("evaluate")
الأصلية في القسم @sec:core-of-evaluator، باستثناء أن
الدوال التي نوجه إليها تؤدي تحليلًا فقط، وليس تقييمًا كاملاً:

#idx("analyze", sub: "metacircular", decl: true)
#snippet(```python
def analyze(component):
    return analyze_literal(component) if is_literal(component) else analyze_name(component) if is_name(component) else analyze_application(component) if is_application(component) else analyze(operator_combination_to_application(component)) if is_operator_combination(component) else analyze_conditional(component) if is_conditional(component) else analyze_lambda_expression(component) if is_lambda_expression(component) else analyze_sequence(sequence_statements(component)) if is_sequence(component) else analyze_block(component) if is_block(component) else analyze_return_statement(component) if is_return_statement(component) else analyze(function_decl_to_constant_decl(component)) if is_function_definition(component) else analyze_declaration(component) if is_declaration(component) else analyze_assignment(component) if is_assignment(component) else error("unknown syntax -- analyze", component)
```)

#idx("analyze...", sub: "metacircular")

إليك أبسط دالة تحليل نحوي،
والتي تتعامل مع التعبيرات الحرفية.
فهي ترجع دالة تنفيذ
تتجاهل معامل البيئة الخاص بها وترجع فقط
قيمة التعبير الحرفي:

#snippet(```python
def analyze_literal(component):
    return lambda env: (literal_value(component))
```)

ما زال البحث عن قيمة اسم
يجب أن يتم في مرحلة التنفيذ، نظرًا لأن هذا يعتمد على معرفة
البيئة.#footnote[ومع ذلك، هناك جزء مهم من البحث عن اسم
#emph[يمكن] القيام به كجزء من التحليل النحوي.
كما سنوضح في القسم @sec:lexical-addressing،
يمكن للمرء تحديد الموضع في بنية البيئة حيث سيتم العثور على
قيمة المتغير، مما يلغي الحاجة إلى مسح البيئة بحثًا عن الإدخال الذي يطابق المتغير.]

#snippet(```python
def analyze_name(component):
    return lambda env: (lookup_symbol_value(symbol_of_name(component), env))
```)

لتحليل تطبيق دالة، نُحَلِّل تعبير
الدالة وتعبيرات المعاملات
وننشئ دالة تنفيذ تستدعي دالة التنفيذ لتعبير الدالة
(للحصول على الدالة الفعلية المراد تطبيقها) ودوال
التنفيذ لتعبيرات المعاملات (للحصول على المعاملات الفعلية).
ثم نمرر هذه إلى
#py("execute_application")،
وهي النظيرة لـ #py("apply") في
القسم @sec:core-of-evaluator.
تختلف الدالة #py("execute_application") عن #py("apply") في أن
متن الدالة للدالة المركبة قد تم تحليله بالفعل،
لذا لا داعي لإجراء تحليل إضافي.
وبدلاً من ذلك، نستدعي فقط دالة التنفيذ للمتن على البيئة المُمَدَّدة.

#idx("executeapplication", sub: "metacircular", decl: true)
#snippet(```python
def analyze_application(component):
    ffun = analyze(function_expression(component))
    afuns = map(analyze, arg_expressions(component))
    return lambda env: (execute_application(ffun(env), map(lambda afun: (afun(env)), afuns)))
def execute_application(fun, args):
    if is_primitive_function(fun):
        return apply_primitive_function(fun, args)
    elif is_compound_function(fun):
        result = function_body(fun) (extend_environment(function_parameters(fun), args, function_environment(fun)))
        return return_value_content(result) if is_return_value(result) else None
    else:
        error("unknown function type -- execute_application", fun)
```)

بالنسبة للعبارات الشرطية،
نستخرج ونحلل المحمول، والنتيجة، والبديل في
وقت التحليل.

#snippet(```python
def analyze_conditional(component):
    pfun = analyze(conditional_predicate(component))
    cfun = analyze(conditional_consequent(component))
    afun = analyze(conditional_alternative(component))
    return lambda env: (cfun(env) if is_truthy(pfun(env)) else afun(env))
```)

تحليل تعبير
#en[lambda] يحقق أيضًا كسبًا كبيرًا في الكفاءة: نُحَلِّل
متن #en[lambda] مرة واحدة فقط، حتى لو كانت
الدوال الناتجة عن تقييم تعبير #en[lambda] قد
تُطبَّق مرات عديدة.

#snippet(```python
def analyze_lambda_expression(component):
    params = lambda_parameter_symbols(component)
    bfun = analyze(lambda_body(component))
    return lambda env: (make_function(params, bfun, env))
```)

تحليل تتابع من العبارات
أكثر تعقيدًا.#footnote[انظر التمرين @ex:analyze-sequence للحصول على
بعض الرؤى حول معالجة المتتاليات.] تُحَلَّل كل
عبارة
في التتابع، مما ينتج عنه دالة تنفيذ.
تُدمج دوال التنفيذ هذه لإنتاج دالة تنفيذ
تأخذ بيئة كمعامل وتستدعي بالتتابع كل دالة تنفيذ
فردية مع البيئة كمعامل.

#snippet(```python
def analyze_sequence(stmts):
    def sequentially(fun1, fun2):
        def the_sequence(env):
            fun1_val = fun1(env)
            return (fun1_val
                    if is_return_value(fun1_val)
                    else fun2(env))
        return the_sequence
    def loop(first_fun, rest_funs):
        return (first_fun
                if is_none(rest_funs)
                else loop(sequentially(first_fun, head(rest_funs)),
                          tail(rest_funs)))
    funs = map(analyze, stmts)
    return ((lambda env: None)
            if is_none(funs)
            else loop(head(funs), tail(funs)))
```)

يُمشَّط متن
الكتلة مرة واحدة فقط للتصريحات المحلية.
وتُثبَّت الرابطات في البيئة عندما
تُستدعى دالة التنفيذ للكتلة.

#snippet(```python
def analyze_block(component):
    body = block_body(component)
    bfun = analyze(body)
    locals = scan_out_declarations(body)
    unassigneds = list_of_unassigned(locals)
    return lambda env: (bfun(extend_environment(locals, unassigneds, env)))
```)

بالنسبة لعبارات الإرجاع، نُحَلِّل تعبير الإرجاع.
وتستدعي دالة التنفيذ لعبارة الإرجاع ببلباطة
دالة التنفيذ لتعبير الإرجاع وتغلف النتيجة في قيمة إرجاع.

#snippet(```python
def analyze_return_statement(component):
    rfun = analyze(return_expression(component))
    return lambda env: (make_return_value(rfun(env)))
```)

يجب على الدالة #py("analyze_assignment")
تأجيل التعيين الفعلي للمتغير حتى التنفيذ، عندما تكون
البيئة قد تم تزويدها. ومع ذلك، فإن حقيقة أن تعبير قيمة التعيين
يمكن تحليله (عوديًا) أثناء التحليل توفر كسبًا كبيرًا في الكفاءة، لأن
تعبير قيمة التعيين سيتم تحليله الآن مرة واحدة فقط. وينطبق الشيء نفسه على
تصريحات الثوابت والمتغيرات.

#snippet(```python
def analyze_assignment(component):
    symbol = assignment_symbol(component)
    vfun = analyze(assignment_value_expression(component))
    def the_assignment(env):
        value = vfun(env)
        assign_symbol_value(symbol, value, env)
        return value
    return the_assignment

def analyze_declaration(component):
    symbol = declaration_symbol(component)
    vfun = analyze(declaration_value_expression(component))
    def the_declaration(env):
        assign_symbol_value(symbol, vfun(env), env)
        return None
    return the_declaration
```)

يستخدم المُقَيِّم الجديد نفس بنيات البيانات، ودوال بناء الجملة،
ودوال الدعم وقت التشغيل
كما في الأقسام @sec:representing-expressions،
و @sec:eval-data-structures،
و @sec:running-eval.

#idx("analyze...", sub: "metacircular")

#exercise([
وسع المُقَيِّم في هذا القسم لدعم حلقات
#idx("حلقة while", sub: "تنفيذها في المقييم المحلل")
#py("while").
(انظر التمرين @ex:while_loop.)
])

#exercise(label-name: <ex:analyze-sequence>, [
لا تفهم #en[Alyssa P. Hacker] لماذا
#idx("analyze...", sub: "metacircular")
يحتاج #py("analyze_sequence")
إلى أن يكون معقدًا إلى هذا الحد. جميع دوال التحليل
الأخرى
هي تحويلات مباشرة لدوال التقييم
المقابلة
(أو بند #py("evaluate")) في
القسم @sec:core-of-evaluator.
وقد توقعت أن تبدو
#py("analyze_sequence")
بهذا الشكل:

#snippet(```python
def analyze_sequence(stmts):
    def execute_sequence(funs, env):
        if is_none(funs):
            return None
        elif is_none(tail(funs)):
            return head(funs)(env)
        else:
            head_val = head(funs)(env)
            return head_val if is_return_value(head_val) else execute_sequence(tail(funs), env)
    funs = map(analyze, stmts)
    return lambda env: (execute_sequence(funs, env))
```)

تشرح #en[Eva Lu Ator] لأليسا أن النسخة الموجودة في النص تقوم بالمزيد من عمل تقييم التتابع في وقت التحليل. فدالة تنفيذ التتابع الخاصة بأليسا،
بدلاً من أن تكون استدعاءات دوال التنفيذ الفردية
مبنية فيها، تقوم بالتكرار عبر
الدوال
من أجل استدعائها: وفي الواقع، على الرغم من أن العبارات الفردية
في التتابع قد حُلِّلت، فإن التتابع نفسه لم يُحَلَّل.

قَارِن بين النسختين من
#py("analyze_sequence").
على سبيل المثال، تأمل الحالة الشائعة (النموذجية في
متون الدوال) حيث يحتوي التتابع على عبارة
واحدة فقط.
ما العمل الذي ستؤديه دالة
التنفيذ
الناتجة عن برنامج أليسا؟ وماذا عن دالة التنفيذ
الناتجة عن البرنامج الموجود في النص أعلاه؟ كيف تقارن النسختان بالنسبة لتتابع يحتوي على تعبيرين؟
])

#exercise(label-name: <ex:meta_speed>, [
صمم ونفذ بعض التجارب لمقارنة سرعة المُقَيِّم دائري التجريد الأصلي
مع النسخة الموجودة في هذا القسم. استخدم نتائجك
لتخدير الكسر الزمني الذي يُقضى في التحليل مقابل التنفيذ
لمختلف الدوال.
])

#idx("تحليل نحوي، مفصول عن التنفيذ", sub: "في المقييم دائري التجريد")
#idx("مُقَيِّم مُحَلِّل")
#idx("مُقَيِّم دائري التجريد لـ Python", sub: "النسخة المُحَلِّلة")
