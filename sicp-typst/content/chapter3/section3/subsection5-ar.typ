// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([انتشار القيود], label-name: <sec:constraints>)

#idx("propagation of constraints")
#idx("constraint(s)", sub: "propagation of")

تُنظم برامج الحاسوب تقليدياً كحسابات أُحادية الاتجاه، تجري العمليات على وسائط محددة مسبقاً لإنتاج المخرجات المطلوبة. ومن ناحية أخرى، غالباً ما نمذج الأنظمة بدلالة العلاقات بين الكميات. على سبيل المثال، قد يتضمن النموذج الرياضي لهيكل ميكانيكي معلومات تفيد بأن انحراف $d$ قضيب معدني مرتبط بالقوة $F$ على القضيب، وطول $L$ القضيب، ومساحة المقطع العرضي $A$، وموديول المرونة $E$ عبر المعادلة

$ d A E = F L $

ومثل هذه المعادلة ليست أُحادية الاتجاه. فمع إعطاء أي أربع من الكميات، يمكننا استخدامها لحساب الخامسة. ومع ذلك، فإن ترجمة المعادلة إلى لغة حاسوب تقليدية ستجبرنا على اختيار واحدة من الكميات لتُحسب بدلالة الأربع الأخرى.
وبالتالي، فإن دالة لحساب المساحة $A$ لا يمكن استخدامها لحساب الانحراف $d$، على الرغم من أن حسابات $A$ و $d$ تنشأ من المعادلة نفسها.#footnote[ظهر انتشار القيود لأول مرة في نظام
#idx("SKETCHPAD")
SKETCHPAD التطلعي بشكل مذهل لـ
#idx("Sutherland, Ivan")
إيفان سذرلاند (#en[Ivan Sutherland 1963]). وطور نظام انتشار قيود جميل يعتمد على لغة
#idx("Smalltalk")
Smalltalk بوساطة
#idx("Borning, Alan")
ألان بورنينغ (#en[Alan Borning 1977]) في مركز
#idx("Xerox Palo Alto Research Center")
Xerox Palo Alto للأبحاث. وطبق سوسمان وستالمان وستيل انتشار القيود على تحليل الدوائر الكهربائية
#idx("Sussman, Gerald Jay")
#idx("Stallman, Richard M.")
(#en[Sussman and Stallman 1975; Sussman and Steele 1980]).
#idx("Steele, Guy Lewis Jr.")
#idx("TK!Solver")
و
#idx("Konopasek, Milos")
#idx("Jayaraman, Sundaresan")
TK!Solver (#en[Konopasek and Jayaraman 1984]) بيئة نمذجة واسعة تعتمد على القيود.]

ونرسم في هذا القسم تصميم لغة تمكننا من العمل بدلالة
#idx("relations, computing in terms of")
العلاقات نفسها. والعناصر الأولية للغة هي
#idx("primitive constraints")
#idx("constraint(s)", sub: "primitive")
#emph[القيود الأولية] (#en[primitive constraints])، والتي تنص على أن علاقات معينة تتواجد بين الكميات. على سبيل المثال، فإن
#py("adder(a, b, c)")
تحدد أن الكميات $a$ و $b$ و $c$ يجب أن ترتبط بالمعادلة $a + b = c$، وتصل
#py("multiplier(x, y, z)")
عن القيد $x y = z$، وتحدد
#py("constant(3.14, x)")
أن قيمة $x$ يجب أن تكون 3.14.

وتوفر لغتنا وسيلة لتركيب القيود الأولية من أجل التعبير عن علاقات أكثر تعقيداً. ونحن نركب القيود بإنشاء
#idx("constraint network")
#emph[شبكات القيود] (#en[constraint networks])، والتي تُربط فيها القيود بوساطة
#idx("connector(s), in constraint system")
#emph[الموصلات] (#en[connectors]). والموصل هو كائن "يحمل" قيمة قد تشارك في قيد واحد أو أكثر. على سبيل المثال، نعلم أن العلاقة بين درجات الحرارة بالفهرنهايت والسيلزيوس هي

$ 9C = 5(F - 32) $

ويمكن التفكير في مثل هذا القيد كشبكة تتكون من قيود البوابة المضيفة والمضاعفة والثابت الأولية (الشكل @fig:constraint). ونرى في الشكل على اليسار صندوق مضاعف بثلاث محطات، مسمومة بـ $m_1$ و $m_2$ و $p$. وتصل هذه المضاعف ببقية الشبكة كما يلي:
محطة $m_1$ ترتبط بموصل $C$ يحمل درجة الحرارة بالسيلزيوس.
ومحطة $m_2$ ترتبط بموصل $w$ مرتبط أيضاً بصندوق ثابت يحمل القيمة 9. ومحطة $p$، التي يفرض صندوق المضاعف أن تكون حاصل ضرب $m_1$ و $m_2$، ترتبط بمحطة $p$ لصندوق مضاعف آخر، والذي يرتبط $m_2$ الخاص به بالثابت 5 ويرتبط $m_1$ الخاص به بأحد الحدود في المجموع.

#sicp-figure(image("/images/img_original/ch3-Z-G-30.svg", width: 70%), caption: [العلاقة $9C = 5(F - 32)$ معبراً عنها كشبكة قيود.], label-name: <fig:constraint>)

ويستمر الحساب بوساطة مثل هذه الشبكة كما يلي: عندما يُعطى موصل قيمة (بوساطة المستخدم أو بوساطة صندوق قيد مرتبط به)، فإنه يوقظ جميع القيود المرتبطة به (باستثناء القيد الذي أيقظه للتو) لإبلاغها بأن لديه قيمة.
ثم يفحص كل صندوق قيد مستيقظ موصلاته لرؤية ما إذا كانت هناك معلومات كافية لتحديد قيمة لموصل ما. وإذا كان الأمر كذلك، يحدد الصندوق ذلك الموصل، والذي يوقظ بدوره جميع القيود المرتبطة به، وهكذا. على سبيل المثال، في التحويل بين السيلزيوس والفهرنهايت، يُحدد $w$ و $x$ و $y$ فوراً بوساطة الصناديق الثابتة إلى 9 و 5 و 32 على التوالي. وتوقظ الموصلات المضاعفات والمضيف، والتي تحدد أنه لا توجد معلومات كافية للمتابعة. وإذا حدد المستخدم (أو جزء آخر من الشبكة) $C$ إلى قيمة (مثلاً 25)، سيستيقظ المضاعف الأيسر، وسيحدد $u$ إلى $25 dot.op 9 = 225$.
ثم يوقظ $u$ المضاعف الثاني، والذي يحدد $v$ إلى 45، ويوقظ $v$ المضيف، والذي يحدد $F$ إلى 77.

=== استخدام نظام القيود

لاستخدام نظام القيود لإجراء حساب درجة الحرارة الموضح أعلاه، ننادي أولاً المنشئ #py("make_connector") لإنشاء موصلين، #py("C") و #py("F")، ثم نربطهما في شبكة مناسبة:

#snippet(```python
C = make_connector()
F = make_connector()
print(celsius_fahrenheit_converter(C, F))
```)

#output(```python
C = make_connector()
F = make_connector()
print(celsius_fahrenheit_converter(C, F))
```)

تُعرف الدالة التي تنشئ الشبكة كما يلي:
#idx("celsiusfahrenheitconverter", decl: true)
#snippet(```python
def celsius_fahrenheit_converter(c, f):
    u = make_connector()
    v = make_connector()
    w = make_connector()
    x = make_connector()
    y = make_connector()
    multiplier(c, w, u)
    multiplier(v, x, u)
    adder(v, y, f)
    constant(9, w)
    constant(5, x)
    constant(32, y)
    return "ok"
```)

تنشئ هذه الدالة الموصلات الداخلية #py("u") و #py("v") و #py("w") و #py("x") و #py("y")، وتصلها كما هو موضح في الشكل @fig:constraint باستخدام منشئات القيود الأولية #py("adder") و #py("multiplier") و #py("constant"). وتصوير هذه التركيبات من العناصر الأولية بدلالة الدوال يوفر للغتنا تلقائياً وسيلة تجريد للكائنات المركبة، كما هو الحال مع محاكي الدوائر الرقمية في القسم @sec:circuit-simulator.

ولمشاهدة الشبكة في حالة عمل، يمكننا وضع مجسات على الموصلين #py("C") و #py("F")، باستخدام دالة #py("probe") مشابهة لتلك التي استخدمناها لمراقبة الأسلاك في القسم @sec:circuit-simulator. ووضع مجس على موصل سيتسبب في طباعة رسالة كلما أُعطي الموصل قيمة:

#snippet(```python
probe("Celsius temp", C)
probe("Fahrenheit temp", F)
```)

وبعد ذلك نحدد قيمة #py("C") إلى 25. (والوسيط الثالث لـ #py("set_value") يخبر #py("C") أن التوجيه يأتي من #py("user").)

#snippet(```python
print(set_value(C, 25, "user"))
```)

#output(```python
print(set_value(C, 25, "user"))
```)

يستيقظ المجس على #py("C") ويبلغ عن القيمة.
وينشر #py("C") أيضاً قيمته عبر الشبكة كما هو موضح أعلاه. ويحدد هذا #py("F") إلى 77، والذي يُبلغ عنه بوساطة المجس على #py("F").

والآن يمكننا محاولة تحديد #py("F") إلى قيمة جديدة، مثلاً 212:

#snippet(```python
set_value(F, 212, "user")
```)

#output(```python
set_value(F, 212, "user")
```)

يشكو الموصل من أنه استشعر تناقضاً: فقيمته هي 77، وشخص ما يحاول تحديدها إلى 212. وإذا كنا نريد حقاً إعادة استخدام الشبكة بقيم جديدة، فيمكننا إخبار #py("C") بأن ينسى قيمته القديمة:

#snippet(```python
print(forget_value(C, "user"))
```)

#output(```python
print(forget_value(C, "user"))
```)

يجد #py("C") أن الـ #py("\"user\"")، الذي حدد قيمته أصلاً، يتراجع الآن عن تلك القيمة، فيوافق #py("C") على فقدان قيمته، كما يُظهر المجس، ويبلغ بقية الشبكة بهذه الحقيقة. وتنتشر هذه المعلومات في النهاية إلى #py("F")، والذي يجد الآن أنه ليس لديه سبب للاستمرار في الاعتقاد بأن قيمته الخاصة هي 77. وبالتالي، فإن #py("F") يتخلى أيضاً عن قيمته، كما يُظهر المجس.

والآن بعد أن أصبح #py("F") بدون قيمة، فنحن أحرار في تحديدها إلى 212:

#snippet(```python
print(set_value(F, 212, "user"))
```)

#output(```python
print(set_value(F, 212, "user"))
```)

وهذه القيمة الجديدة، عند انتشارها عبر الشبكة، تجبر #py("C") على امتلاك قيمة 100، ويُسجل هذا بوساطة المجس على #py("C"). ولاحظ أن الشبكة ذاتها تُستخدم لحساب #py("C") بفرض #py("F") ولحساب #py("F") بفرض #py("C").
وهذا عدم توجيه الحساب هو الميزة المميزة للأنظمة القائمة على القيود.

=== تنفيذ نظام القيود

يُنفذ نظام القيود عبر كائنات إجرائية بـ حالة محلية، بطريقة مشابهة جداً لمحاكي الدوائر الرقمية في القسم @sec:circuit-simulator. وعلى الرغم من أن الكائنات الأولية لنظام القيود أكثر تعقيداً نوعاً ما، إلا أن النظام العام أبسط، حيث لا يوجد اهتمام بالأجندات والتأخيرات المنطقية.

والعمليات الأساسية
#idx("connector(s), in constraint system", sub: "operations on")
على الموصلات هي التالية:

- #py("has_value(")#meta("connector")#py(")") #idx("hasvalue") \ تخبر ما إذا كان للموصل قيمة.
- #py("get_value(")#meta("connector")#py(")") #idx("getvalue") \ تُرجع القيمة الحالية للموصل.
- #py("set_value(")#meta("connector")#py(",")#meta("new-value")#py(",") #meta("informant")#py(")") #idx("setvalue") \ تشير إلى أن المخبر يطلب من الموصل تحديد قيمته إلى القيمة الجديدة.
- #py("forget_value(")#meta("connector")#py(",") #meta("retractor")#py(")") #idx("forgetvalue") \ تخبر الموصل أن المتراجع يطلب منه نسيان قيمته.
- #py("connect(")#meta("connector")#py(",") #meta("new-constraint")#py(")") #idx("connect") \ تخبر الموصل بالمشاركة في القيد الجديد.

وتتواصل الموصلات مع القيود بوساطة الدوال #py("inform_about_value")، التي تخبر القيد المعطى أن الموصل لديه قيمة، و #py("inform_about_no_value")، التي تخبر القيد أن الموصل قد فقد قيمته.

ويبني المنشئ #py("Adder") قيد مضيف بين موصلات المجموع الجزيئي #py("a1") و #py("a2") وموصل المجموع الكلي #py("sum"). ويُنفذ المضيف كدالة بالحالة المحلية (الدالة #py("me") أدناه):
#idx("adder (primitive constraint)", decl: true)
#snippet(```python
def adder(a1, a2, sum):
    def process_new_value():
        if has_value(a1) and has_value(a2):
            set_value(sum, get_value(a1) + get_value(a2), me)
        elif has_value(a1) and has_value(sum):
            set_value(a2, get_value(sum) - get_value(a1), me)
        elif has_value(a2) and has_value(sum):
            set_value(a1, get_value(sum) - get_value(a2), me)
        else:
            pass
    def process_forget_value():
        forget_value(sum, me)
        forget_value(a1, me)
        forget_value(a2, me)
        process_new_value()
    def me(request):
        if request == "I have a value.":
            process_new_value()
        elif request == "I lost my value.":
            process_forget_value()
        else:
            error("unknown request -- adder", request)
    connect(a1, me)
    connect(a2, me)
    connect(sum, me)
    return me
```)

تصل الدالة #py("adder") المضيف الجديد بالموصلات المحددة وتُرجعه كقيمته. والدالة #py("me")، التي تمثل المضيف، تعمل كموجه للدوال المحلية.
وتُستخدم "واجهات بناء الجمل" التالية (انظر الحاشية في القسم @sec:circuit-simulator) بالتزامن مع التوجيه:
#idx("informaboutvalue", decl: true)#idx("informaboutnovalue", decl: true)
#snippet(```python
def inform_about_value(constraint):
    return constraint("I have a value.")

def inform_about_no_value(constraint):
    return constraint("I lost my value.")
```)

تُستدعى دالة المضيف المحلية #py("process_new_value") عندما يُبلغ المضيف بأن أحد موصلاته لديه قيمة.
ويفحص المضيف أولاً لرؤية ما إذا كان كل من #py("a1") و #py("a2") لديهما قيم. وإذا كان الأمر كذلك، يخبر #py("sum") بتحديد قيمته إلى مجموع الحدين المضافين. والوسيط #py("informant") لـ #py("set_value") هو #py("me")، والذي هو كائن المضيف نفسه. وإذا لم يكن لكل من #py("a1") و #py("a2") قيم، يختبر المضيف ما إذا كان لـ #py("a1") و #py("sum") قيم. وإذا كان الأمر كذلك، يحدد #py("a2") إلى الفرق بينهما. وأخيراً، إذا كان لـ #py("a2") و #py("sum") قيم، فإن هذا يعطي المضيف معلومات كافية لتحديد #py("a1"). وإذا أُخبر المضيف بأن أحد موصلاته قد فقد قيمة، فإنه يطلب أن تفقد جميع موصلاته قيمها الآن. (والقيم التي حددها هذا المضيف هي فقط التي تُفقد في الواقع). ثم يشغل #py("process_new_value").
والسبب في هذه الخطوة الأخيرة هو أن موصلاً واحداً أو أكثر قد تظل لديه قيمة (أي أن الموصل قد يكون لديه قيمة لم تحدد أصلاً بوساطة المضيف)، وهذه القيم قد تحتاجه للانتشار عكسياً عبر المضيف.

والمضاعف مشابهاً جداً للمضيف. وسيحدد حاصل ضربه #py("product") إلى 0 إذا كان أي من العاملين مساوياً لـ 0، حتى لو لم يكن العامل الآخر معروفاً.
#idx("multiplier", sub: "primitive constraint", decl: true)
#snippet(```python
def multiplier(m1, m2, product):
    def process_new_value():
        if ((has_value(m1) and get_value(m1) == 0)
                or (has_value(m2) and get_value(m2) == 0)):
            set_value(product, 0, me)
        elif has_value(m1) and has_value(m2):
            set_value(product, get_value(m1) * get_value(m2), me)
        elif has_value(product) and has_value(m1):
            set_value(m2, get_value(product) / get_value(m1), me)
        elif has_value(product) and has_value(m2):
            set_value(m1, get_value(product) / get_value(m2), me)
        else:
            pass
    def process_forget_value():
        forget_value(product, me)
        forget_value(m1, me)
        forget_value(m2, me)
        process_new_value()
    def me(request):
        if request == "I have a value.":
            process_new_value()
        elif request == "I lost my value.":
            process_forget_value()
        else:
            error("unknown request -- multiplier", request)
    connect(m1, me)
    connect(m2, me)
    connect(product, me)
    return me
```)

ويحدد منشئ الثابت #py("constant") مجرد قيمة الموصل المحدد. وأي رسالة #py("\"I have a value.\"") أو #py("\"I lost my value.\"") تُرسل إلى صندوق الثابت ستنتج خطأ.
#idx("constant (primitive constraint)", decl: true)
#snippet(```python
def constant(value, connector):
    def me(request):
        error("unknown request -- constant", request)
    connect(connector, me)
    set_value(connector, value, me)
    return me
```)

وأخيراً، يطبع المجس رسالة حول تحديد أو إلغاء تحديد الموصل المحدد:

#idx("probe", sub: "in constraint system", decl: true)
#snippet(```python
def probe(name, connector):
    def print_probe(value):
        display("Probe: " + name + " = " + str(value))
    def process_new_value():
        print_probe(get_value(connector))
    def process_forget_value():
        print_probe("?")
    def me(request):
        return (process_new_value()
                if request == "I have a value."
                else process_forget_value()
                if request == "I lost my value."
                else error("unknown request -- probe", request))
    connect(connector, me)
    return me
```)

=== تمثيل الموصلات

#idx("connector(s), in constraint system", sub: "representing")

يُمثل الموصل ككائن إجرائي بمتغيرات حالة محلية: #py("value")، القيمة الحالية للموصل؛ و #py("informant")، الكائن الذي حدد قيمة الموصل؛ و #py("constraints")، قائمة القيود التي يشارك فيها الموصل.
#idx("makeconnector", decl: true)
#snippet(```python
def make_connector():
    value = False
    informant = False
    constraints = None
    def set_my_value(newval, setter):
        nonlocal value, informant
        if not has_value(me):
            value = newval
            informant = setter
            return for_each_except(setter,
                                   inform_about_value,
                                   constraints)
        elif value != newval:
            error("contradiction", llist(value, newval))
        else:
            return "ignored"
    def forget_my_value(retractor):
        nonlocal informant
        if retractor is informant:
            informant = False
            return for_each_except(retractor,
                                   inform_about_no_value,
                                   constraints)
        else:
            return "ignored"
    def connect(new_constraint):
        nonlocal constraints
        if is_none(member(new_constraint, constraints)):
            constraints = pair(new_constraint, constraints)
        else:
            pass
        if has_value(me):
            inform_about_value(new_constraint)
        else:
            pass
        return "done"
    def me(request):
        if request == "has_value":
            return informant is not False
        elif request == "value":
            return value
        elif request == "set_value":
            return set_my_value
        elif request == "forget":
            return forget_my_value
        elif request == "connect":
            return connect
        else:
            error("unknown operation -- connector", request)
    return me
```)

تُستدعى الدالة المحلية للموصل #py("set_my_value") عندما يكون هناك طلب لتحديد قيمة الموصل. وإذا لم تكن للموصل قيمة حالية، فإنه سيحدد قيمته ويتذكر القيد ك#py("informant") الذي طلب تحديد القيمة.#footnote[قد لا يكون #py("setter") قيداً. وفي مثال درجة الحرارة لدينا، استخدمنا #py("\"user\"") ك#py("setter").] ثم يبلغ الموصل جميع القيود المشاركة باستثناء القيد الذي طلب تحديد القيمة. ويتحقق هذا باستخدام المكرر التالي، الذي يطبق دالة محدودة على كافة العناصر في قائمة باستثناء عنصر معطى:
#idx("foreachexcept", decl: true)
#snippet(```python
def for_each_except(exception, fun, list):
    def loop(items):
        if is_none(items):
            return "done"
        elif head(items) is exception:
            return loop(tail(items))
        else:
            fun(head(items))
            return loop(tail(items))
    return loop(list)
```)

وإذا طُلب من الموصل نسيان قيمته، فإنه يشغل #py("forget_my_value")، وهي دالة محلية تفحص أولاً للتأكد من أن الطلب يأتي من الكائن نفسه الذي حدد القيمة أصلاً. وإذا كان الأمر كذلك، يبلغ الموصل القيود المرتبطة به بفقدان القيمة.

وتضيف الدالة المحلية #py("connect") القيد الجديد المحدد إلى قائمة القيود إذا لم يكن موجوداً بالفعل في تلك القائمة.
ثم إذا كانت للموصل قيمة، يبلغ القيد الجديد بهذه الحقيقة.

وتخدم الدالة #py("me") للموصل كموجه للدوال الداخلية الأخرى وتمثل أيضاً الموصل ككائن. وتوفر الدوال التالية واجهة بناء جمل للتوجيه:
#idx("hasvalue", decl: true)#idx("getvalue", decl: true)#idx("setvalue", decl: true)#idx("forgetvalue", decl: true)#idx("connect", decl: true)
#snippet(```python
def has_value(connector):
    return connector("has_value")

def get_value(connector):
    return connector("value")

def set_value(connector, new_value, informant):
    return connector("set_value")(new_value, informant)

def forget_value(connector, retractor):
    return connector("forget")(retractor)

def connect(connector, new_constraint):
    return connector("connect")(new_constraint)
```)

#exercise(label-name: <ex:3_33>, [
باستخدام قيود المضاعف، والمضيف، والثابت الأولية، عرّف دالة
#idx("averager (constraint)")
#py("averager") تأخذ ثلاثة موصلات #py("a") و #py("b") و #py("c") كمدخلات وتؤسس القيد بأن قيمة #py("c") هي المتوسط لقيمتي #py("a") و #py("b").
])

#exercise(label-name: <ex:squarer-constraint>, [
يريد لويس ريزونر بناء
#idx("squarer (constraint)")
جهاز تربيحي، وهو جهاز قيد بمحطتين بحيث تكون قيمة الموصل #py("b") على المحطة الثانية دائماً مربع قيمة #py("a") على المحطة الأولى. ويقترح الجهاز البسيط التالي المصنوع من مضاعف:

#snippet(```python
def squarer(a, b):
    return multiplier(a, a, b)
```)

هناك عيب خطير في هذه الفكرة. اشرحه.
])

#exercise(label-name: <ex:3_35>, [
يخبر بن بتديدل لويس أن إحدى الطرق لتجنب المشكلة في التمرين @ex:squarer-constraint هي تعريف
#idx("squarer (constraint)")
جهاز تربيعي كقيد أولي جديد. املأ الأجزاء المفقودة في مخطط بن لدالة تنفذ مثل هذا القيد:

#syntax("
def squarer(a, b):
    def process_new_value():
        if has_value(b):
            if get_value(b) < 0:
                error(\"square less than 0 -- squarer\", get_value(b))
            else:
                ", meta("alternative_1"), "
        else:
            ", meta("alternative_2"), "
    def process_forget_value():
        ", meta("body_1"), "
    def me(request):
        ", meta("body_2"), "
    ", meta("statements"), "
    return me
      ")
])

#exercise(label-name: <ex:3_36>, [
افترض أننا قمنا بتقييم تسلسل التعليمات التالي في بيئة البرنامج:

#snippet(```python
a = make_connector()
b = make_connector()
set_value(a, 10, "user")
```)

وفي وقت ما أثناء تقييم #py("set_value")، يُقَيَّم التعبير التالي من الدالة المحلية للموصل:

#snippet(```python
for_each_except(setter, inform_about_value, constraints)
```)

ارسم مخطط بيئة يوضح البيئة التي يُقَيَّم فيها التعبير أعلاه.
])

#exercise(label-name: <ex:3_37>, [
تُعد دالة #py("celsius_fahrenheit_converter") ثقيلة بالمقارنة مع أسلوب تعريف أكثر توجهاً للتعبير، مثل
#idx("celsiusfahrenheitconverter", sub: "expression-oriented", decl: true)
#snippet(```python
def celsius_fahrenheit_converter(x):
    return cplus(cmul(cdiv(cv(9), cv(5)), x), cv(32))

C = make_connector()
F = celsius_fahrenheit_converter(C)
```)

وهنا #py("cplus") و #py("cmul") وإلخ هي نسخ "القيد" من العمليات الحسابية. على سبيل المثال، تأخذ #py("cplus") موصلين كوسائط وتُرجع موصلاً مرتبطاً بهما بقيد مضيف:

#snippet(```python
def cplus(x, y):
    z = make_connector()
    adder(x, y, z)
    return z
```)

عرّف دوالاً مماثلة #py("cminus") و #py("cmul") و #py("cdiv") و #py("cv") (قيمة ثابتة) تمكننا من تعريف قيود مركبة كما في مثال المحول أعلاه.#footnote[إن التنسيق الموجه بالتعبير
#idx("expression-oriented vs. imperative programming style")
#idx("imperative vs. expression-oriented programming style")
مريح لأنه يتجنب الحاجة لتسمية التعبيرات الوسيطة في الحساب. وصياغتنا الأصلية للغة القيود ثقيلة بالطريقة نفسها التي تكون بها العديد من اللغات ثقيلة عند التعامل مع العمليات على البيانات المركبة. على سبيل المثال، إذا أردنا حساب حاصل الضرب $((a+b)) dot.op ((c+d))$، حيث تمثل المتغيرات متجهات، فيمكننا العمل بـ "أسلوب أمري"، باستخدام دوال تفرض قيم وسائط متجهات محدودة ولكنها لا تُرجع كقيم متجهات بأنفسها:

#snippet(```python
v_sum("a", "b", temp1)
v_sum("c", "d", temp2)
v_prod(temp1, temp2, answer)
```)

وبدلاً من ذلك، كان بإمكاننا التعامل مع التعبيرات، باستخدام دوال تُرجع متجهات كقيم، وبالتالي تجنب الإشارة صراحة إلى #py("temp1") و #py("temp2"):

#snippet(```python
answer = v_prod(v_sum("a", "b"), v_sum("c", "d"))
```)

وبما أن بايثون تسمح لنا بـ إرجاع كائنات مركبة كقيم للدوال، فيمكننا تحويل لغة القيود بأسلوبها الأمري إلى أسلوب موجه بالتعبير كما هو موضح في هذا التمرين.

وبفرض ميزة التنسيق الموجه بالتعبير، قد يسأل المرء ما إذا كان هناك سبب لتنفيذ النظام بأسلوب أمري، كما فعلنا في هذا القسم. أحد الأسباب هو أن لغة القيود غير الموجهة بالتعبير توفر مقبضاً على كائنات القيود (مثلاً، قيمة دالة #py("adder")) وكذلك على كائنات الموصلات. وهذا مفيد إذا كنا نرغب في توسيع النظام بعمليات جديدة تتواصل مع القيود مباشرة بدلاً من التواصل غير المباشر عبر العمليات على الموصلات فقط. وعلى الرغم من أنه من السهل تنفيذ الأسلوب الموجه بالتعبير بدلالة التنفيذ الأمري، إلا أنه من الصعب جداً العكس.]
])

#idx("propagation of constraints")
#idx("constraint(s)", sub: "propagation of")
