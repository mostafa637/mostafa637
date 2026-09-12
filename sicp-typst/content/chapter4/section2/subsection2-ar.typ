// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([مفسر بالتقييم الكسول])

في هذا القسم سننفذ لغة ذات ترتيب عادي تكون متطابقة مع #en[Python] باستثناء أن الدوال المركبة غير صارمة في كل وسيط. وستظل الدوال الأولية صارمة. ليس من الصعب تعديل المُقيِّم في القسم @sec:core-of-evaluator بحيث تتصرف اللغة التي يفسرها بهذه الطريقة. وتتركز معظم التغييرات المطلوبة حول تطبيق الدوال.

الفكرة الأساسية هي أنه عند تطبيق دالة، يجب على المفسر تحديد الوسائط التي يجب تقييمها والوسائط التي يجب تأجيلها. لا تُقَيَّم الوسائط المؤجلة؛ وبدلاً من ذلك، تُحوَّل إلى كائنات تسمى
#idx("thunk")
#emph[thunk]s.#footnote[كُلمة #emph[thunk] تم ابتكارها بواسطة مجموعة عمل غير رسمية كانت تناقش تنفيذ الاستدعاء بالاسم
#idx("thunk", sub: "origin of name")
#idx("Algol", sub: "thunks")
في لغة #en[Algol 60]. وقد لاحظوا أن معظم تحليل («التفكير في») التعبير يمكن إجراؤه في وقت التصريف؛ وبالتالي، في وقت التشغيل، يكون التعبير قد تم «التفكير فيه» مسبقًا (#en["thunk" about])
#idx("Ingerman, Peter")
(#en[Ingerman et al. 1960]).]
يجب أن تحتوي الـ #en[thunk] على المعلومات المطلوبة لإنتاج قيمة الوسيط عندما تدعو الحاجة، كما لو كانت قد قُيِّمت في وقت التطبيق. وبالتالي، يجب أن تحتوي الـ #en[thunk] على تعبير الوسيط والبيئة التي يُقَيَّم فيها تطبيق الدالة.

تُسمى عملية تقييم التعبير المغلَّف في الـ #en[thunk] بـ
#idx("forcing", sub: "of thunk")
#idx("thunk", sub: "forcing")
#emph[الفرض] (#en[forcing]).#footnote[هذا مماثل لفرض الكائنات المؤجلة التي قُدِّمت في الفصل @chap:state لتمثيل التدفقات. الفارق الحاسم بين ما نفعله هنا وما فعلناه في الفصل @chap:state هو أننا نبني التأجيل والفرض داخل المُقيِّم نفسه، وبذلك نجعل هذا موحدًا وتلقائيًا في جميع أنحاء اللغة.]
وعمومًا، تُفْرَض الـ #en[thunk] فقط عندما تكون هناك حاجة إلى قيمتها: عندما تُمرَّر إلى دالة أولية ستستخدم قيمة الـ #en[thunk]؛ وعندما تكون قيمة للمحمول في تعبير شرطي؛ وعندما تكون قيمة لتعبير دالة على وشك أن تُطبَّق كدالة.
أحد خيارات التصميم المتاحة لدينا هو ما إذا كنا سنقوم بـ
#idx("memoization", sub: "of thunks")
#emph[التخزين الموقّت] (#en[memoize]) للـ #en[thunks] أم لا، بشكل مشابه للتحسين الخاص بالتدفقات في القسم @sec:delayed-lists. مع التخزين الموقّت، في أول مرة تُفْرَض فيها الـ #en[thunk]، تُخَزِّن القيمة التي تم حسابها. وتسترجع الاستدعاءات اللاحقة القيمة المخزنة دون تكرار الحساب. سنجعل مفسرنا يقوم بالتخزين الموقّت، لأن هذا أكثر كفاءة للعديد من التطبيقات. ومع ذلك، هناك اعتبارات دقيقة هنا.#footnote[التقييم الكسول المقترن بالتخزين الموقّت يُشار إليه أحيانًا باسم تمرير الوسائط بـ
#idx("call-by-need argument passing")
#emph[الاستدعاء عند الحاجة] (#en[call-by-need])، على عكس تمرير الوسائط بـ
#emph[الاستدعاء بالاسم] (#en[call-by-name]).
#idx("call-by-name argument passing")
(الاستدعاء بالاسم، الذي قُدِّم في
#idx("Algol", sub: "call-by-name argument passing")
#en[Algol 60]، يشبه التقييم الكسول غير المُنَفَّذ مع تخزين موقت.) بصفتنا مصممي لغات، يمكننا بناء مُقَيِّمنا ليقوم بالتخزين الموقّت، أو ألا يقوم به، أو ترك هذا كخيار للمبرمجين (التمرين @ex:user-controlled-strictness). وكما قد تتوقع من الفصل @chap:state، فإن هذه الخيارات تثير قضايا تصبح دقيقة ومربكة في وجود الإسناد. (انظر التمارين @ex:delay-side-effects و @ex:memoize-or-not). حاول مقال ممتاز كتبه #idx("Clinger, William") #en[Clinger (1982)] توضيح الأبعاد المتعددة للارتباك التي تنشأ هنا.]

#idx("thunk")

#subheading([تعديل المُقيِّم])

الفارق الرئيسي بين المُقيِّم الكسول والمُقيِّم في القسم @sec:mc-eval هو في كيفية التعامل مع تطبيقات الدوال في #py("evaluate") و#py("apply").

#idx("evaluate (lazy)")
يصبح بند #py("is_application") لـ #idx("evaluate (lazy)") #py("evaluate") كالتالي:

#snippet(```python
: is_application(component)
? apply(actual_value(function_expression(component), env),
        arg_expressions(component), env)
```)

هذا يشبه تقريبًا بند #py("is_application") لـ #py("evaluate") في القسم @sec:core-of-evaluator. ومع ذلك، بالنسبة للتقييم الكسول، فإننا نستدعي #py("apply") مع تعبيرات الوسائط، بدلاً من الوسائط الناتجة عن تقييمها. وبما أننا سنحتاج إلى البيئة لإنشاء الـ #en[thunks] إذا كان يجب تأجيل الوسائط، فيجب علينا تمرير البيئة أيضًا. وما زلنا نُقَيِّم تعبير الدالة، لأن #py("apply") تحتاج إلى الدالة الفعلية المراد تطبيقها للتوجيه بناءً على نوعها (أولية مقابل مركبة) وتطبيقها.

كلما احتجنا إلى القيمة الفعلية لتعبير، نستخدم

#idx("actualvalue", decl: true)
#snippet(```python
def actual_value(exp, env):
    return force_it(evaluate(exp, env))
```)

بدلاً من مجرد #py("evaluate")، بحيث إذا كانت قيمة التعبير عبارة عن #en[thunk]، سيتم فرضها.

نسختنا الجديدة من #py("apply") هي أيضًا شبه متطابقة مع النسخة في القسم @sec:core-of-evaluator. الفارق هو أن #py("evaluate") مررت تعبيرات وسائط غير مُقَيَّمة: بالنسبة للدوال الأولية (وهي صارمة)، نُقَيِّم جميع الوسائط قبل تطبيق الأولية؛ وبالنسبة للدوال المركبة (وهي غير صارمة)، نؤجل جميع الوسائط قبل تطبيق الدالة.

#idx("apply (lazy)", decl: true)
#snippet(```python
def apply(fun, args, env):
    if is_primitive_function(fun):
        return apply_primitive_function(fun, list_of_arg_values(args, env))
    elif is_compound_function(fun):
        result = evaluate(function_body(fun), extend_environment(function_parameters(fun), list_of_delayed_args(args, env), function_environment(fun)))
        return return_value_content(result) if is_return_value(result) else None
    else:
        error("unknown function type -- apply", fun)
```)

الدوال التي تعالج الوسائط تشبه تمامًا #py("list_of_values") من القسم @sec:core-of-evaluator، باستثناء أن #py("list_of_delayed_args") تؤجل الوسائط بدلاً من تقييمها، و#py("list_of_arg_values") تستخدم #py("actual_value") بدلاً من #py("evaluate"):

#idx("listofargvalues", decl: true)#idx("listofdelayedargs", decl: true)
#snippet(```python
def list_of_arg_values(exps, env):
    return map(lambda exp: (actual_value(exp, env)), exps)
def list_of_delayed_args(exps, env):
    return map(lambda exp: (delay_it(exp, env)), exps)
```)

المكان الآخر الذي يجب أن نغير فيه المُقيِّم هو في التعامل مع العبارات الشرطية، حيث يجب استخدام #py("actual_value") بدلاً من #py("evaluate") للحصول على قيمة تعبير المحمول قبل اختباره ما إذا كان صحيحًا أم خادعًا:

#idx("evalconditional (lazy)", decl: true)
#snippet(```python
def eval_conditional(component, env):
    return evaluate(conditional_consequent(component), env) if is_truthy(actual_value(conditional_predicate(component), env)) else evaluate(conditional_alternative(component), env)
```)

أخيرًا، يجب أن نغير دالة
#idx("driver loop", sub: "in lazy evaluator")
#py("driver_loop") (من القسم @sec:running-eval) لتستخدم #py("actual_value") بدلاً من #py("evaluate")، بحيث إذا تم إرجاع قيمة مؤجلة إلى حلقة القراءة والتقييم والطباعة، فسيتم فرضها قبل طباعتها. ونغير المحثات أيضًا للإشارة إلى أن هذا هو المُقيِّم الكسول:

#idx("prompts", sub: "lazy evaluator")#idx("driverloop", sub: "for lazy evaluator", decl: true)
#snippet(```python
input_prompt = "L-evaluate input: "
output_prompt = "L-evaluate value: "
def driver_loop(env):
    input = user_read(input_prompt)
    if is_none(input):
        print("evaluator terminated")
    else:
        program = parse(input)
        locals = scan_out_declarations(program)
        unassigneds = list_of_unassigned(locals)
        program_env = extend_environment(locals, unassigneds, env)
        output = actual_value(program, program_env)
        user_print(output_prompt, output)
        return driver_loop(program_env)
```)

مع إجراء هذه التغييرات، يمكننا تشغيل المُقيِّم واختباره. يوضح التقييم الناجح لتعبير #py("try_me") المناقش في القسم @sec:evaluation-order أن المفسر يؤدي تقييمًا كسولاً:

#snippet(```python
the_global_environment = setup_environment()
driver_loop(the_global_environment)
```)

#prompt(```python
L-evaluate input:
```)

#snippet(```python
def try_me(a, b):
    return 1 if a == 0 else b
```)

#output(```python
def try_me(a, b):
    return 1 if a == 0 else b
```)

#prompt(```python
L-evaluate input:
```)

#snippet(```python
try_me(0, head(None))
```)

#output(```python
try_me(0, head(None))
```)

#subheading([تمثيل الـ thunks])

#idx("thunk", sub: "implementation of")

يجب أن يرتب مُقَيِّمنا لإنشاء #en[thunks] عندما تُطبَّق الدوال على الوسائط ولفرض هذه الـ #en[thunks] لاحقًا. يجب أن تُغَلِّف الـ #en[thunk] تعبيرًا جنبًا إلى جنب مع البيئة، بحيث يمكن إنتاج الوسيط لاحقًا. لفرض الـ #en[thunk]، نستخرج ببساطة التعبير والبيئة من الـ #en[thunk] ونُقَيِّم التعبير في البيئة. نستخدم #py("actual_value") بدلاً من #py("evaluate") حتى أنه في حالة ما إذا كانت قيمة التعبير هي نفسها #en[thunk]، فسنفرض ذلك، وهكذا، حتى نصل إلى شيء ليس #en[thunk]:

#idx("forceit", decl: true)
#snippet(```python
def force_it(obj):
    return actual_value(thunk_exp(obj), thunk_env(obj)) if is_thunk(obj) else obj
```)

إحدى الطرق السهلة لتغليف تعبير مع بيئة هي إنشاء قائمة تحتوي على التعبير والبيئة. وبالتالي، ننشئ #en[thunk] على النحو التالي:

#idx("delayit", decl: true)
#snippet(```python
def delay_it(exp, env):
    return llist("thunk", exp, env)
def is_thunk(obj):
    return is_tagged_list(obj, "thunk")
def thunk_exp(thunk):
    return head(tail(thunk))

def thunk_env(thunk):
    return head(tail(tail(thunk)))
```)

في الواقع، ما نريده لمفسرنا ليس هذا بالضبط، بل بالأحرى #en[thunks] تم تخزينها موقتًا.

عندما تُفْرَض الـ #en[thunk]، سنحولها إلى #en[thunk] مُقَيَّمة عن طريق استبدال التعبير المخزن بقيمته وتغيير وسام #py("thunk") بحيث يمكن التعرف عليه على أنه قُيِّم بالفعل.#footnote[لاحظ أننا نمسح أيضًا #py("env") من الـ #en[thunk] بمجرد حساب قيمة التعبير. لا يحدث هذا أي فرق في القيم التي يرجعها المفسر. ولكنه يساعد في توفير المساحة، لأن إزالة الإشارة من الـ #en[thunk] إلى #py("env") بمجرد عدم الحاجة إليها يسمح لهذه البنية بالخضوع لـ
#idx("garbage collection", sub: "memoization and")
#idx("memoization", sub: "garbage collection and")
#emph[جمع القمامة] (#en[garbage collection]) وإعادة تدوير مساحتها، كما سنناقش في القسم @sec:storage-allocation.

وبالمثل، كان بإمكاننا السماح للبيئات غير الضرورية في الكائنات المؤجلة المخزنة موقتًا من القسم @sec:delayed-lists بالخضوع لجمع القمامة، عن طريق جعل #py("memo") تفعل شيئًا مثل #py("fun = None;") للتخلص من الدالة #py("fun") (والتي تتضمن البيئة التي تم فيها تقييم تعبير #en[lambda] الذي يشكل ذيل التدفق) بعد تخزين قيمتها.]

#idx("forceit", sub: "memoized version", decl: true)
#snippet(```python
def is_evaluated_thunk(obj):
    return is_tagged_list(obj, "evaluated_thunk")
def thunk_value(evaluated_thunk):
    return head(tail(evaluated_thunk))

def force_it(obj):
    if is_thunk(obj):
        result = actual_value(thunk_exp(obj), thunk_env(obj))
        set_head(obj, "evaluated_thunk")
        set_head(tail(obj), result)
        set_tail(tail(obj), None)
        return result
    elif is_evaluated_thunk(obj):
        return thunk_value(obj)
    else:
        return obj
```)

لاحظ أن نفس الدالة #py("delay_it") تعمل سواء مع أو بدون التخزين الموقّت.#idx("thunk", sub: "implementation of")

#exercise(label-name: <ex:delay-side-effects>, [
افترض أننا نكتب التصريحات التالية للمُقَيِّم الكسول:

#snippet(```python
count = 0
def id(x):
    global count
    count = count + 1
    return x
```)

اعطِ القيم المفقودة في التتابع التالي من التفاعلات، ووضح إجاباتك:#footnote[يوضح هذا التمرين أن التفاعل بين التقييم الكسول والآثار الجانبية يمكن أن يكون مربكًا للغاية. هذا بالضبط ما قد تتوقعه من المناقشة في الفصل @chap:state.]

#snippet(```python
w = id(id(10))
```)

#prompt(```python
L-evaluate input:
```)

#syntax("
count
    ")

#output(```python
w = id(id(10))
```)

#prompt(```python
L-evaluate input:
```)

#syntax("
w
    ")

#output(```python
w = id(id(10))
```)

#prompt(```python
L-evaluate input:
```)

#syntax("
count
    ")

#output(```python
w = id(id(10))
```)
])

#exercise(label-name: <ex:force-operator>, [
تستخدم الدالة #py("evaluate") الدالة #py("actual_value") بدلاً من #py("evaluate") لتقييم تعبير الدالة قبل تمريره إلى #py("apply")، وذلك لفرض قيمة تعبير الدالة. اعطِ مثالاً يوضح الحاجة إلى هذا الفرض.
])

#exercise(label-name: <ex:memoize-or-not>, [
اعرض برنامجًا تتوقع أن يعمل أبطأ بكثير بدون تخزين موقت مقارنة مع تخزين موقت. وافترض التفاعل التالي، حيث تكون الدالة #py("id") مُعرَّفة كما في التمرين @ex:delay-side-effects وتبدأ #py("count") من 0:

#snippet(```python
def square(x):
    return x * x
```)

#prompt(```python
L-evaluate input:
```)

#syntax("
square(id(10))
      ")

#output(```python
def square(x):
    return x * x
```)

#prompt(```python
L-evaluate input:
```)

#syntax("
count
      ")

#output(```python
def square(x):
    return x * x
```)

اعطِ الإجابات عندما يقوم المُقيِّم بالتخزين الموقّت وعندما لا يقوم به.
])

#exercise(label-name: <ex:force-sequence>, [
يشعر #en[Cy D. Fect]، وهو مبرمج #en[C] سابق تعافى، بالقلق من أن بعض الآثار الجانبية قد لا تحدث أبدًا، لأن المُقيِّم الكسول لا يفرض العبارات في التتابع. وبما أن قيمة عبارة ما في تتابع قد لا تُستخدم (فقد تكون العبارة موجودة فقط لغرض أثرها، مثل الإسناد لمتغير أو الطباعة)، فقد لا يكون هناك استخدام لاحق لهذه القيمة (مثل وسيط لدالة أولية) يتسبب في فرضها. لذلك يعتقد ساين أنه عند تقييم المتتاليات، يجب فرض جميع العبارات في التتابع. ويقترح تعديل #py("evaluate_sequence") من القسم @sec:core-of-evaluator لتستخدم #py("actual_value") بدلاً من #py("evaluate"):

#snippet(```python
def eval_sequence(stmts, env):
    if is_empty_sequence(stmts):
        return None
    elif is_last_statement(stmts):
        return actual_value(first_statement(stmts), env)
    else:
        first_stmt_value = actual_value(first_statement(stmts), env)
        if is_return_value(first_stmt_value):
            return first_stmt_value
        else:
            return eval_sequence(rest_statements(stmts), env)
```)

+ يعتقد #en[Ben Bitdiddle] أن ساين على خطأ. ويوضح لساين دالة #py("for_each") الموصوفة في التمرين @ex:for-each، والتي تعطي مثالاً مهماً لتتابع يحتوي على آثار جانبية: #idx("foreach", decl: true) #snippet(```python def for_each(fun, items): if is_none(items): return "done" else: fun(head(items)) for_each(fun, tail(items)) ```) ويدعي أن المُقيِّم في النص (مع #py("eval_sequence") الأصلية) يتعامل مع هذا بشكل صحيح: #prompt(```python L-evaluate input: ```) #snippet(```python for_each(print, llist(57, 321, 88)) ```) #output(```python 57 321 88 L-evaluate value: "done" ```) وضح لماذا بن على حق بشأن سلوك #py("for_each").
+ يوافق ساين على أن بن على حق بشأن مثال #py("for_each")، لكنه يقول إن هذا ليس نوع البرامج التي كان يفكر فيها عندما اقترح تغييره على #py("eval_sequence"). ويُصرِّح بالدالتين التاليتين في المُقيِّم الكسول: #snippet(```python def f1(x): x = pair(x, llist(2)) return x def f2(x): def f(e): e return x return f(x = pair(x, llist(2))) ```) ما هي قيم #py("f1(1)") و#py("f2(1)") مع #py("eval_sequence") الأصلية؟ وماذا ستكون القيم مع تغيير ساين المقترح على #py("eval_sequence")؟
+ يشير ساين أيضًا إلى أن تغيير #py("eval_sequence") كما اقترح لا يؤثر على سلوك المثال في الجزء أ. وضح لماذا هذا صحيح.
+ كيف تعتقد أنه ينبغي التعامل مع المتتاليات في المُقيِّم الكسول؟ هل تفضل نهج ساين، أم النهج الموجود في النص، أم نهجًا آخر؟
])

#exercise(label-name: <ex:user-controlled-strictness>, [
النهج المتبع في هذا القسم مزعج إلى حد ما، لأنه يحل تغييرًا غير متوافق مع #en[Python]. وقد يكون من الأجمل تنفيذ التقييم الكسول كالتالي:
#idx("upward compatibility")
#emph[توسيع متوافق تصاعديًا] (#en[upward-compatible extension])، أي بحيث تعمل برامج #en[Python] العادية كما كانت من قبل. يمكننا القيام بذلك عن طريق تقديم تصريح بارامتر اختياري كشكل نحوي جديد داخل تصريحات الدوال للسماح للمستخدم بالتحكم فيما إذا كان ينبغي تأجيل الوسائط أم لا. وبينما نحن بصدد ذلك، يمكننا أيضًا إعطاء المستخدم الخيار بين التأجيل مع أو بدون تخزين موقت. على سبيل المثال، التصريح

#syntax("
def f(a, b, c, d):
    parameters(\"strict\", \"lazy\", \"strict\", \"lazy_memo\")
    ", $dots.h$)

سيحدد #py("f") لتقون دالة من أربعة وسائط، حيث يُقَيَّم الوسيطان الأول والثالث عند استدعاء الدالة، والوسيط الثاني يُؤَجَّل، والوسيط الرابع يُؤَجَّل ويُخَزَّن موقتًا معًا.

يمكنك افتراض أن تصريح البارامترات هو دائمًا العبارة الأولى في متن تعريف الدالة، وإذا أُهمل، فإن جميع البارامترات تكون صارمة. وبالتالي، سيتيح تعريف الدالة العادي نفس السلوك مثل #en[Python] العادية، بينما ستؤدي إضافة تصريح #py("\"lazy_memo\"") لكل بارامتر من كل دالة مركبة إلى إنتاج سلوك المُقيِّم الكسول المحدَّد في هذا القسم. صمم ونفذ التغييرات المطلوبة لإنتاج مثل هذا التوسيع لـ #en[Python]. ستتعامل دالة #py("parse") مع تصريحات البارامترات كتطبيقات دوال، لذا يلزمك تعديل #py("apply") للتوجيه إلى تنفيذك للشكل النحوي الجديد.

يجب عليك أيضًا الترتيب لـ #py("evaluate") أو #py("apply") لتحديد متى ينبغي تأجيل الوسائط، ولفرض أو تأجيل الوسائط وفقًا لذلك، ويجب عليك الترتيب للفرض ليقوم بالتخزين الموقّت أو لا، حسب الاقتضاء.
])

#idx("lazy evaluator")
