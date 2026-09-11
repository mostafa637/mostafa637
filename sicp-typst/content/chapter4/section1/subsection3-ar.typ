// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([هياكل بيانات المُقيِّم], label-name: <sec:eval-data-structures>)

بالإضافة إلى تعريف تمثيل المكونات، يجب أن يحدد تنفيذ المُقيِّم أيضاً هياكل البيانات التي يتعامل معها المُقيِّم داخلياً، كجزء من تنفيذ البرنامج، مثل تمثيل الدوال والبيئات وتمثيل صح و خطأ.

#subheading([اختبار الدوال الشرطية])

#idx("metacircular evaluator for Python", sub: "representation of true and false")

من أجل حصر تعبيرات الدوال الشرطية للشرطيات على الشرطيات المناسبة (التعبيرات التي تُقيَّم إلى قيمة منطقية) كما نفعل طوال هذا الكتاب، فإننا نصر هنا على أن الدالة
#idx("metacircular evaluator for Python", sub: "representation of true and false")
#py("is_truthy") تُطبق فقط على القيم المنطقية، ونقبل فقط القيمة المنطقية
#py("true") لتكون صحيحة.
والعكس لـ
#py("is_truthy") يُسمى
#py("is_falsy").#footnote[الشرطيات في Python الكاملة تتقبل #emph[أي] قيمة، وليس فقط قيمة منطقية، كنتيجة لتقييم تعبير "الدالة الشرطية". ومفهوم Python للصحة والخطأ يتحدد بالنسخ التالية من #py("is_truthy") و #py("is_falsy"):
#idx("truthiness")
#idx("falsiness")
#idx("isboolean")#idx("istruthy", sub: "full Python version", decl: true)#idx("isfalsy", sub: "full Python version", decl: true)
#snippet(```python
def is_truthy(x):
    return not is_falsy(x)

def is_falsy(x):
    return (is_boolean(x) and not x) or (is_number(x) and (x == 0 or x != x)) or (is_string(x) and x == "") or is_null(x) or is_undefined(x)
```)

الاختبار #py("x != x") ليس خطأ مطبعياً؛ فالقيمة الوحيدة في Python التي يثمر فيها #py("x != x") عن صح هي القيمة #py("NaN") ("ليس عدداً")،
#idx("NaN, not a typo")
والتي تُعتبر عدداً خاطئاً (أيضاً ليس خطأ مطبعياً)، جنباً إلى جنب مع 0.
والقيمة العددية #py("NaN") هي نتيجة بعض الحالات الحدية الحسابية مثل #py("0 / 0").

تم صياغة المصطلحات "حقاني" (#en[truthy]) و "خاطئاني" (#en[falsy]) بواسطة
#idx("good parts of Python")
#idx("Python", sub: "good parts")
#idx("Crockford, Douglas")
دوجلاس كروكفورد، والذي ألهم أحد كتبه (كروكفورد 2008) هذا التكيف.]<foot:truthy>

#idx("istruthy", decl: true)#idx("isfalsy", decl: true)
#snippet(```python
def is_truthy(x):
    return x if is_boolean(x) else error("boolean expected, received", x)
def is_falsy(x):
    return not is_truthy(x)
```)

#subheading([تمثيل الدوال])

للتعامل مع الدوال الأوليّة، نفترض أن لدينا الدوال التالية متاحة:
#idx("metacircular evaluator for Python", sub: "representation of functions")

- #py("apply_primitive_function(")#meta("fun")#py(",") #meta("args")#py(")") #idx("applyprimitivefunction") تطبق الدالة الأوليّة المعطاة على قيم الوسائط في القائمة #meta("args") وترجع نتيجة التطبيق.
- #py("is_primitive_function(")#meta("fun")#py(")") #idx("isprimitivefunction") تختبر ما إذا كانت #meta("fun") دالة أوليّة.

وهذه الآليات للتعامل مع الدوال الأوليّة موصوفة بشكل أكبر في القسم @sec:running-eval.

الدوال المركبة تُبنى من البارامترات، وأجسام الدوال، والبيئات باستخدام المنشئ #py("make_function"):
#idx("makefunction", decl: true)#idx("iscompoundfunction", decl: true)#idx("functionparameters", decl: true)#idx("functionbody", decl: true)#idx("functionenvironment", decl: true)
#snippet(```python
def make_function(parameters, body, env):
    return llist("compound_function", parameters, body, env)
def is_compound_function(f):
    return is_tagged_list(f, "compound_function")
def function_parameters(f):
    return llist_ref(f, 1)

def function_body(f):
    return llist_ref(f, 2)

def function_environment(f):
    return llist_ref(f, 3)
```)

#subheading([تمثيل قيم الإرجاع])

رأينا في القسم @sec:core-of-evaluator أن تقييم التسلسل ينتهي عند مواجهة عبارة إرجاع، وأن تقييم تطبيق الدالة يحتاج إلى إرجاع القيمة #py("None") إذا لم يواجه تقييم جسم الدالة عبارة إرجاع. وللتعرف على أن القيمة نتجت عن
#idx("return value", sub: "representation in metacircular evaluator")
عبارة إرجاع، نقدم #emph[قيم الإرجاع] كهياكل بيانات للمُقيِّم.
#idx("makereturnvalue", decl: true)#idx("isreturnvalue", decl: true)#idx("returnvaluecontent", decl: true)
#snippet(```python
def make_return_value(content):
    return llist("return_value", content)
def is_return_value(value):
    return is_tagged_list(value, "return_value")
def return_value_content(value):
    return head(tail(value))
```)

#subheading([العمليات على البيئات])

#anchor(<sec:operations-on-environments>)

يحتاج المُقيِّم إلى عمليات لـ
#idx("metacircular evaluator for Python", sub: "environment operations")
#idx("symbol(s)", sub: "in environment operations")
التعامل مع البيئات. وكما يُوضح في القسم @sec:environment-model، فإن البيئة هي تسلسل من الإطارات، حيث يكون كل إطار جدولاً من الارتباطات التي تربط الرموز بقيمها المقابلة. ونستخدم العمليات التالية للتعامل مع البيئات:

- #py("lookup_symbol_value(")#meta("symbol")#py(",") #meta("env")#py(")") #idx("lookupsymbolvalue") ترجع القيمة المربوطة بـ #meta("symbol") في البيئة #meta("env")، أو تعطي إشارة خطأ إذا كان #meta("symbol") غير مربوط.
- #py("extend_environment(")#meta("symbols")#py(",") #meta("values")#py(",") #meta("base-env")#py(")") #idx("extendenvironment") ترجع بيئة جديدة، تتكون من إطار جديد ترتبط فيه الرموز في القائمة #meta("symbols") بالعناصر المقابلة في القائمة #meta("values")، حيث البيئة المحيطة هي البيئة #meta("base-env").
- #py("assign_symbol_value(")#meta("symbol")#py(",") #meta("value")#py(",") #meta("env")#py(")") #idx("assignsymbolvalue") تعثر على الإطار الأكثر دفقاً في #meta("env") الذي يربط #meta("symbol")، وتغير ذلك الإطار بحيث يصبح #meta("symbol") الآن مربوطاً بـ #meta("value")، أو تعطي إشارة خطأ إذا كان #meta("symbol") غير مربوط.

لتنفيذ هذه العمليات
#idx("metacircular evaluator for Python", sub: "representation of environments")
نمثل البيئة كقائمة من الإطارات. والبيئة المحيطة للبيئة هي #py("tail") القائمة. والبيئة الفارغة هي ببساطة القائمة الفارغة.
#idx("enclosingenvironment", decl: true)#idx("firstframe", decl: true)#idx("theemptyenvironment", decl: true)
#snippet(```python
def enclosing_environment(env):
    return tail(env)

def first_frame(env):
    return head(env)

the_empty_environment = None
```)

يُمثَّل كل إطار في البيئة كزوج من القوائم: قائمة الأسماء المربوطة في ذلك الإطار وقائمة القيم المقابلة.#footnote[الإطارات ليست تجريداً حقيقياً للبيانات: والدالة #py("assign_symbol_value") أدناه تستخدم #py("set_head") لتعديل القيم في الإطار مباشرة. والغرض من دوال الإطار هو جعل دوال التعامل مع البيئة أسهل للقراءة.]

#idx("makeframe", decl: true)#idx("framesymbols", decl: true)#idx("framevalues", decl: true)
#snippet(```python
def make_frame(symbols, values):
    return pair(symbols, values)

def frame_symbols(frame):
    return head(frame)

def frame_values(frame):
    return tail(frame)
```)

لتوسيع بيئة بـ إطار جديد يربط الرموز بالقيم، ننشئ إطاراً يتكون من قائمة الرموز وقائمة القيم، ونضم هذا إلى البيئة. ونعطي إشارة خطأ إذا كان عدد الرموز لا يطابق عدد القيم.
#idx("extendenvironment", decl: true)
#snippet(```python
def extend_environment(symbols, vals, base_env):
    return pair(make_frame(symbols, vals), base_env) if length(symbols) == length(vals) else error("too many arguments supplied" if length(symbols) < length(vals) else "too few arguments supplied", pair(symbols, vals))
```)

يُستخدم هذا بواسطة #py("apply") في القسم @sec:core-of-evaluator لربط بارامترات الدالة بوسائطها.

للتعرف على قيمة رمز في بيئة، نفحص قائمة الرموز في الإطار الأول. فإذا عثرنا على الرمز المطلوب، نرجع العنصر المقابل في قائمة القيم. وإذا لم نعثر على الرمز في الإطار الحالي، نبحث في البيئة المحيطة، وهكذا. وإذا وصلنا إلى البيئة الفارغة، نعطي إشارة خطأ #py("\"unbound name\"").
#idx("lookupsymbolvalue", decl: true)
#snippet(```python
def lookup_symbol_value(symbol, env):
    def env_loop(env):
        def scan(symbols, vals):
            return env_loop(enclosing_environment(env)) if is_none(symbols) else head(vals) if symbol == head(symbols) else scan(tail(symbols), tail(vals))
        if env == the_empty_environment:
            error("unbound name", symbol)
        else:
            frame = first_frame(env)
            return scan(frame_symbols(frame), frame_values(frame))
    return env_loop(env)
```)

لإسناد قيمة جديدة لـ رمز في بيئة محددة، نفحص بحثاً عن الرمز، تماماً كما في #py("lookup_symbol_value")، ونغير القيمة المقابلة عندما نعثر عليه.

#idx("assignsymbolvalue", decl: true)
#snippet(```python
def assign_symbol_value(symbol, val, env):
    def env_loop(env):
        def scan(symbols, vals):
            return env_loop(enclosing_environment(env)) if is_none(symbols) else set_head(vals, val) if symbol == head(symbols) else scan(tail(symbols), tail(vals))
        if env == the_empty_environment:
            error("unbound name -- assignment", symbol)
        else:
            frame = first_frame(env)
            return scan(frame_symbols(frame), frame_values(frame))
    return env_loop(env)
```)

الطريقة الموصوفة هنا هي واحدة فقط من العديد من الطرق المعقولة لتمثيل البيئات. وبما أننا استخدمنا
#idx("metacircular evaluator for Python", sub: "data abstraction in")
تجريد البيانات لعزل باقي المُقيِّم عن الاختيار التفصيلي للتمثيل، فيمكننا تغيير تمثيل البيئة إذا أردنا. (انظر التمرين @ex:alternate-frame-representation.) وفي نظام Python بجودة الإنتاج، فإن سرعة عمليات بيئة المُقيِّم—خاصة سرعة البحث عن الرموز—لها تأثير كبير على أداء النظام. والتمثيل الموصوف هنا، على الرغم من بساطته المفهومية، ليس كفؤاً ولن يُستخدم عادة في نظام إنتاجي.#footnote[العيب في هذا التمثيل (وكذلك التنويع في التمرين @ex:alternate-frame-representation) هو أن المُقيِّم قد يضطر للبحث عبر إطارات عديدة للعثور على الارتباط لمتغير معطى. (ومثل هذه المقاربة تُسمى
#idx("deep binding")
#idx("binding", sub: "deep")
#emph[الربط العميق] (#en[deep binding])). وإحدى الطرق لتجنب عدم الكفاءة هذه هي الاستفادة من استراتيجية تُسمى #emph[العنونة المعجمية]، والتي ستُناقش في القسم @sec:lexical-addressing.]

#idx("metacircular evaluator for Python", sub: "representation of environments")

#exercise(label-name: <ex:alternate-frame-representation>, [
بدلاً من تمثيل الإطار كزوج من القوائم، يمكننا تمثيل الإطار كقائمة من الارتباطات، حيث تكون كل ارتباطة عبارة عن زوج من رمز وقيمة. أعد كتابة عمليات البيئة لاستخدام هذا التمثيل البديل.
])

#exercise(label-name: <ex:4_10>, [
يمكن التعبير عن الدوال #py("lookup_symbol_value") و #py("assign_symbol_value") بدلالة دالة أكثر تجريداً للمرور عبر هيكل البيئة.
عرف تجريداً يلتقط النمط المشترك وأعد تعريف الدالتين بدلالة هذا التجريد.
])

#exercise(label-name: <ex:mutable>, [
تميز لغتنا بين الثوابت والمتغيرات باستخدام كلمات مفتاحية مختلفة—#py("const") و #py("let")—وتمنع الإسناد للثوابت. ومع ذلك، فإن مُفسِّرنا لا يستفيد من هذا التمييز؛ فالدالة #py("assign_symbol_value") ستسند بسعادة قيمة جديدة لـ رمز معطى، بغض النظر عما إذا كان مُعلناً كثابت أو متغير.
#idx("constant (in Python)", sub: "detecting assignment to")
صحح هذا الخلل بـ استدعاء الدالة #py("error") كلما جرت محاولة لاستخدام ثابت على الجانب الأيسر لـ إسناد.
يمكنك الاستمرار كما يلي:

- قدم دالتين شرطيتين #py("is_constant_declaration") و #py("is_variable_declaration") تسمحان لك بالتمييز بين النوعين. كما هو موضح في القسم @sec:representing-expressions، فإن #py("parse") تميز بينهما باستخدام العنوانين #py("\"constant_declaration\"") و #py("\"variable_declaration\"").
- غير #py("scan_out_declarations") و (إذا كان ذلك ضرورياً) #py("extend_environment") بحيث يمكن التمييز بين الثوابت والمتغيرات في الإطارات التي تُرتبط فيها.
- غير #py("assign_symbol_value") بحيث تفحص ما إذا كان الرمز المعطى قد تُمّ إعلانه كمتغير أو كثابت، وفي الحالة الأخيرة تعطي إشارة خطأ أن عمليات الإسناد غير مسموح بها على الثوابت.
- غير #py("eval_declaration") بحيث عندما تواجه إعلان ثابت، تستدعي دالة جديدة، #py("assign_constant_value")، والتي لا تجري الفحص الذي قدمته في #py("assign_symbol_value").
- إذا كان ذلك ضرورياً، غير #py("apply") لضمان أن الإسناد لبارامترات الدوال يظل ممكناً.
])

#exercise(label-name: <ex:access_unassigned>, [
+ يتطلب تحديد Python من التنفيذ إعطاء إشارة خطأ وقت التشغيل عند محاولة الوصول إلى قيمة اسم قبل تقييم إعلانه (انظر نهاية القسم @sec:env-internal-def). ولتحقيق هذا السلوك في المُقيِّم، #idx("lookupsymbolvalue", sub: "for scanned-out declarations") غير #py("lookup_symbol_value") لإعطاء إشارة خطأ إذا كانت القيمة التي يعثر عليها هي #py("\"*unassigned*\"").
+ وبالمثل، يجب ألا نسند قيمة جديدة لـ متغير إذا لم نكن قد قمنا بتقييم إعلان #py("let") الخاص به بعد. غير تقييم الإسناد بحيث يعطي الإسناد لـ متغير مُعلن بـ #py("let") إشارة خطأ في هذه الحالة.
])

#exercise(label-name: <ex:var_js>, [
قبل وضع Strict Mode لـ ECMAScript 2015 الذي نستخدمه في هذا الكتاب، كانت متغيرات Python تعمل بشكل مختلف تماماً عن متغيرات Scheme، مما كان سيجعل هذا التكيف لـ Python أقل إقناعاً بكثير.

+ قبل ECMAScript 2015، كانت الطريقة الوحيدة لإعلان متغير محلي في Python هي استخدام الكلمة المفتاحية #py("var") بدلاً من الكلمة المفتاحية #py("let"). ونطاق المتغيرات المُعلنة بـ #py("var") هو الجسم بأكمله لـ تعريف الدالة أو تعبير #en[lambda] المحيط مباشرة، وليس فقط الكتلة المحيطة مباشرة. عدل #py("scan_out_declarations") و #py("eval_block") بحيث تتبع الأسماء المُعلنة بـ #py("const") و #py("let") قواعد نطاق #py("var").
+ عند عدم وجود Strict Mode، تسمح Python لأسماء غير مُعلنة بأن تظهر على يسار #py("=") في الإسنادات. ومثل هذا الإسناد يضيف الارتباط الجديد للبيئة العالمية. عدل الدالة #py("assign_symbol_value") لجعل الإسناد يتصرف بهذه الطريقة. وتُمّ تقديم Strict Mode، الذي يمنع مثل هذه الإسنادات، في Python لجعل البرامج أكثر أماناً. ما هي قضية الأمان المعالجة بـ منع الإسناد من إضافة ارتباطات للبيئة العالمية؟
])
