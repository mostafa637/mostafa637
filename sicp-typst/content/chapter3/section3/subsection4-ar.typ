// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([محاكي للدوائر الرقمية], label-name: <sec:circuit-simulator>)

#idx("digital-circuit simulation")

إن تصميم الأنظمة الرقمية المعقدة، مثل الحاسوب، نشاط هندسي مهم. وتُبنى الأنظمة الرقمية بربط العناصر البسيطة متبادلة الربط. وعلى الرغم من أن سلوك هذه العناصر الفردية بسيط، إلا أن شبكات منها يمكن أن تمتلك سلوكاً معقداً للغاية. والمحاكاة الحاسوبية لتصاميم الدوائر المقترحة هي أداة مهمة يستخدمها مهندسو الأنظمة الرقمية. وفي هذا القسم نصمم نظاماً لإجراء المحاكاة المنطقية الرقمية. ويمثل هذا النظام نوعاً من البرامج يُسمى
#idx("event-driven simulation")
#idx("simulation", sub: "event-driven")
#emph[محاكاة مدفوعة بالأحداث] (#en[event-driven simulation])، والتي تثير فيها الأفعال ("الأحداث") أحداثاً أخرى تحدث في وقت لاحق، والتي تثير بدورها أحداثاً أكثر، وهكذا.

وسيتكون نموذجنا الحسابي للدارة من كائنات تتوافق مع المكونات الأولية التي تُبنى منها الدارة. وتوجد هناك
#idx("wire, in digital circuit")
#emph[الأسلاك] (#en[wires])، التي تحمل
#idx("signal, digital")
#idx("digital signal")
#emph[إشارات رقمية] (#en[digital signals]). وقد تمتلك الإشارة الرقمية في أي لحظة قيمة واحدة فقط من قيمتين ممكنتين، 0 و 1. وتوجد أيضاً أنواع مختلفة من
#idx("function box, in digital circuit")
#emph[صناديق الدوال] الرقمية، التي تربط الأسلاك التي تحمل إشارات الدخل بأسماء أسلاك الخرج الأخرى. وتنتج هذه الصناديق إشارات خرج مكسوبة من إشارات دخلها. وتتأخر إشارة الخرج
#idx("delay, in digital circuit", sort: "delay")
بمقدار زمن يعتمد على نوع صندوق الدالة. على سبيل المثال، فإن
#idx("inverter")
#emph[العاكس] (#en[inverter]) هو صندوق دالة أولي يعكس دخله. وإذا تغيرت إشارة الدخل إلى عاكس إلى 0، فإنه بعد مرور زمن #emph[تأخير العاكس] سيغير العاكس إشارة خرجه إلى 1. وإذا تغيرت إشارة الدخل إلى عاكس إلى 1، فإنه بعد مرور زمن #emph[تأخير العاكس] سيغير العاكس إشارة خرجه إلى 0. ونحن نرسم العاكس رمزياً كما في الشكل @fig:logic-gates. وبوابة
#idx("and-gate")
#emph[بوابة وَ] (#en[and-gate])، الموضحة أيضاً في الشكل @fig:logic-gates، هي صندوق دالة أولي بدخلين وخرج واحد. وهي تقود إشارة خرجها إلى قيمة تكون
#idx("logical and (digital logic)")
#emph[وَ المنطقية] للدخل. أي أنه إذا أصبحت كلتا إشارتي دخلها 1، فإنه بعد مرور زمن #emph[تأخير بوابة وَ] ستجبر بوابة وَ إشارة خرجها على أن تكون 1؛ وبخلاف ذلك سيكون الخرج 0. وبوابة
#idx("or-gate")
#emph[بوابة أَوْ] (#en[or-gate]) هي صندوق دالة أولي مشابه بدخلين يقود إشارة خرجه إلى قيمة تكون
#idx("logical or (digital logic)")
#emph[أَوْ المنطقية] للدخل. أي أن الخرج سيصبح 1 إذا كانت إشارة دخل واحدة على الأقل 1؛ وبخلاف ذلك سيصبح الخرج 0.

#sicp-figure(image("/images/img_original/ch3-Z-G-24.svg", width: 70%), caption: [دوال أولية في محاكي المنطق الرقمي.], label-name: <fig:logic-gates>)

ويمكننا ربط الدوال الأولية معاً لبناء دوال أكثر تعقيداً. ولتحقيق ذلك نربط أصلك مخرجات بعض صناديق الدوال بمدخلات صناديق دوال أخرى. على سبيل المثال، فإن دارة
#idx("half-adder")
#idx("adder", sub: "half")
#emph[جامع النصف] (#en[half-adder]) الموضحة في الشكل @fig:half-adder تتكون من بوابة أَوْ، وبوابتي وَ، وعاكس. وتأخذ إشارتي دخل، $A$ و $B$، ولها إشارتا خرج، $S$ و $C$.
وسيصبح $S$ مساوياً لـ 1 عندما تكون إشارة واحدة فقط من $A$ و $B$ مساوية لـ 1، وسيصبح $C$ مساوياً لـ 1 عندما تكون كل من $A$ و $B$ مساوية لـ 1. ونستطيع أن نرى من الشكل أنه بسبب التأخيرات المتضمنة، قد تتولد المخرجات في أوقات مختلفة. وتنشأ العديد من الصعوبات في تصميم الدوائر الرقمية من هذه الحقيقة.

#sicp-figure(image("/images/img_original/ch3-Z-G-25.svg", width: 70%), caption: [دارة جامع النصف.], label-name: <fig:half-adder>)

وسوف نبني الآن برنامجاً لنمذجة دوائر المنطق الرقمي التي نرغب في دراستها. وسوف يبني البرنامج كائنات حسابية تنمذج الأسلاك، والتي "ستحمل" الإشارات. وصناديق الدوال ستُنَمذج بوساطة دوال تفرض العلاقات الصحيحة بين الإشارات.

وسيكون أحد العناصر الأساسية في المحاكاة لدينا دالة
#idx("makewire")
#py("make_wire")،
التي تبني الأسلاك. على سبيل المثال، يمكننا بناء ستة أسلاك كما يلي:

#snippet(```python
a = make_wire()
b = make_wire()
c = make_wire()
d = make_wire()
e = make_wire()
s = make_wire()
```)

ونحن نربط صندوق دالة بمجموعة من الأسلاك باستدعاء دالة تبني ذلك النوع من الصناديق. والوسائط لدالة المنشئ هي الأسلاك المراد ربطها بالصندوق. على سبيل المثال، بفرض أنه يمكننا بناء بوابات وَ، وبوابات أَوْ، وعواكس، يمكننا ربط جامع النصف الموضح في الشكل @fig:half-adder معاً:

#snippet(```python
print(or_gate(a, b, d))
```)

#output(```python
print(or_gate(a, b, d))
```)

#snippet(```python
print(and_gate(a, b, c))
```)

#output(```python
print(and_gate(a, b, c))
```)

#snippet(```python
print(inverter(c, e))
```)

#output(```python
print(inverter(c, e))
```)

#snippet(```python
print(and_gate(d, e, s))
```)

#output(```python
print(and_gate(d, e, s))
```)

والأفضل من ذلك أنه يمكننا تسمية هذه العملية صراحة بتعريف دالة
#py("half_adder")
تبني هذه الدارة، بفرض الأسلاك الخارجية الأربعة المراد ربطها بجامع النصف:

#idx("half-adder", sub: "halfadder", decl: true)
#snippet(```python
def half_adder(a, b, s, c):
    d = make_wire()
    e = make_wire()
    or_gate(a, b, d)
    and_gate(a, b, c)
    inverter(c, e)
    and_gate(d, e, s)
    return "ok"
```)

وميزة إجراء هذا التعريف هي أنه يمكننا استخدام #py("half_adder") نفسها كلَبِنة بناء في إنشاء دوائر أكثر تعقيداً. ويُظهر الشكل @fig:full-adder، على سبيل المثال،
#idx("full-adder")
#idx("adder", sub: "full")
#emph[جامعاً كلياً] (#en[full-adder]) يتكون من جامعي نصف وبوابة أَوْ.#footnote[الجامع الكلي هو عنصر دارة أساسي يُستخدم في جمع عددين ثنائيين. وهنا $A$ و $B$ هما البتان عند الموقعين المناظرين في العددين المراد جمعهما، و $C_(italic("in"))$ هو بت الحمل من الجمع موقعاً واحداً إلى اليمين. وتنتج الدارة $italic("SUM")$، وهو بت المجموع في الموقع المناظر، و $C_(italic("out"))$، وهو بت الحمل المراد نشره إلى اليسار.] ويمكننا بناء جامع كلي كما يلي:

#idx("full-adder", sub: "fulladder", decl: true)
#snippet(```python
def full_adder(a, b, c_in, sum, c_out):
    s = make_wire()
    c1 = make_wire()
    c2 = make_wire()
    half_adder(b, c_in, s, c1)
    half_adder(a, s, sum, c2)
    or_gate(c1, c2, c_out)
    return "ok"
```)

وبعد تعريف #py("full_adder") كدالة، يمكننا الآن استخدامها كلَبِنة بناء لإنشاء دوائر أكثر تعقيداً. (على سبيل المثال، انظر التمرين @ex:ripple-carry.)

في الجوهر، يقدم محاكينا الأدوات لبناء لغة دوائر. وإذا اعتمدنا المنظور العام على اللغات الذي اقتربنا به من دراسة بايثون في القسم @sec:elements-of-programming، فيمكننا القول إن صناديق الدوال الأولية تشكل العناصر الأولية للغة، وإن ربط الصناديق معاً يوفر وسيلة التركيب، وإن تحديد أنماط الربط كدوال يخدم كوسيلة للتجريد.

=== صناديق الدوال الأولية

صناديق الدوال الأولية
#idx("digital-circuit simulation", sub: "primitive function boxes")
تنفذ "القوى" التي يؤثر بها تغير الإشارة على سلك ما على الإشارات في أسلاك أخرى. ولبناء صناديق الدوال، نستخدم العمليات التالية على الأسلاك:

- #py("get_signal(")#meta("wire")#py(")") #idx("getsignal") \ تُرجع القيمة الحالية للإشارة على السلك.
- #py("set_signal(")#meta("wire")#py(",") #meta("new-value")#py(")"): #idx("setsignal") \ تغير قيمة الإشارة على السلك إلى القيمة الجديدة.
- #py("add_action(")#meta("wire")#py(",") #meta("function-of-no-arguments")#py(")"): #idx("addaction") \ تؤكد أن الدالة المحددة يجب تشغيلها كلما تغيرت قيمة الإشارة على السلك. ومثل هذه الدوال هي الوسائل التي تُنقل بها التغييرات في قيمة الإشارة على السلك إلى أسلاك أخرى.

وبالإضافة إلى ذلك، سنستخدم دالة
#idx("afterdelay")
#py("after_delay")
تأخذ تأخيراً زمنياً ودالة ليتم تشغيلها وتنفذ الدالة المعطاة بعد التأخير المعطى.

وباستخدام هذه الدوال، يمكننا تعريف دوال المنطق الرقمي الأولية. ولربط دخل بخرج عبر عاكس، نستخدم #py("add_action") لربط سلك الدخل بدالة سيتم تشغيلها كلما تغيرت قيمة الإشارة على سلك الدخل.
وتحسب الدالة #py("logical_not") لإشارة الدخل، ثم بعد مرور زمن #py("inverter_delay") واحد، تضبط إشارة الخرج لتكون هذه القيمة الجديدة:
#idx("inverter", sub: "inverter", decl: true)#idx("logicalnot", decl: true)
#snippet(```python
def inverter(input, output):
    def invert_input():
        new_value = logical_not(get_signal(input))
        after_delay(inverter_delay,
                    lambda: set_signal(output, new_value))
    add_action(input, invert_input)
    return "ok"

def logical_not(s):
    return (1
            if s == 0
            else 0
            if s == 1
            else error("invalid signal", s))
```)

#sicp-figure(image("/images/img_original/ch3-Z-G-26.svg", width: 70%), caption: [دارة الجامع الكلي.], label-name: <fig:full-adder>)

وبوابة وَ هي أكثر تعقيداً بقليل. فدالة الفعل يجب تشغيلها إذا تغير أي من الدخلين للبوابة. وهي تحسب #py("logical_and") (باستخدام دالة مماثلة لـ #py("logical_not")) لقيم الإشارات على أسلاك الدخل وتعد تغييراً للقيمة الجديدة ليحدث على سلك الخرج بعد مرور زمن #py("and_gate_delay") واحد.
#idx("and-gate", sub: "andgate", decl: true)
#snippet(```python
def and_gate(a1, a2, output):
    def and_action_function():
        new_value = logical_and(get_signal(a1),
                                get_signal(a2))
        after_delay(and_gate_delay,
                    lambda: set_signal(output, new_value))
    add_action(a1, and_action_function)
    add_action(a2, and_action_function)
    return "ok"
```)

#exercise(label-name: <ex:3_28>, [
عرّف
#idx("or-gate", sub: "orgate")
بوابة أَوْ كصندوق دالة أولي. ويجب أن يكون منشئك #py("or_gate") مشابهاً لـ #py("and_gate").
])

#exercise(label-name: <ex:3_29>, [
الطريقة الأخرى لبناء بوابة
#idx("or-gate", sub: "orgate")
أَوْ هي كجهاز منطق رقمي مركب، يُبنى من بوابات وَ وعواكس. عرّف دالة #py("or_gate") تحقق ذلك. وما هو وقت التأخير لبوابة أَوْ بدلالة #py("and_gate_delay") و #py("inverter_delay")؟
])

#exercise(label-name: <ex:ripple-carry>, [
يُظهر الشكل @fig:ripple-carry
#idx("ripple-carry adder")
#idx("adder", sub: "ripple-carry")
#emph[جامع الحمل المتسلسل] (#en[ripple-carry adder]) المكون من ربط $n$ من الجامعات الكلية متتالية.
وهذا هو أبسط أشكال الجامع المتوازي لجمع عددين ثنائيين من $n$ بت.
والمدخلات $A_1, A_2, A_3, ..., A_n$ و $B_1, B_2, B_3, ..., B_n$ هما العددان الثنائيان المراد جمعهما (كل من $A_k$ و $B_k$ هو 0 أو 1). وتنتج الدارة $S_1, S_2, S_3, ..., S_n$، بتات المجموع الـ $n$، و $C$، الحمل الناتج عن الجمع. اكتب دالة #py("ripple_carry_adder") تولد هذه الدارة. ويجب أن تأخذ الدالة كوسائط ثلاث قوائم من $n$ من الأسلاك لكل منها—الـ $A_k$، والـ $B_k$، والـ $S_k$—وأيضاً سلكاً آخر $C$.
والعيب الرئيسي لجامع الحمل المتسلسل هو الحاجة للانتظار حتى تنتشر إشارات الحمل. فما هو التأخير المطلوب للحصول على الخرج الكامل من جامع حمل متسلسل بـ $n$ بت، معبراً عنه بدلالة التأخيرات لبوابات وَ، وبوابات أَوْ، والعواكس؟
])

#sicp-figure(image("/images/img_original/ch3-Z-G-27.svg", width: 70%), caption: [جامع حمل متسلسل للأعداد بـ $n$ بت.], label-name: <fig:ripple-carry>)

#idx("digital-circuit simulation", sub: "primitive function boxes")

=== تمثيل الأسلاك

سيكون السلك
#idx("digital-circuit simulation", sub: "representing wires")
في المحاكاة لدينا كائناً حسابياً بمتغيري حالة محليين: #py("signal_value") (يُؤخذ أولياً ك0) ومجموعة من #py("action_functions") سيتم تشغيلها عندما تغير الإشارة قيمتها. وننفذ السلك، باستخدام أسلوب
#idx("message passing", sub: "in digital-circuit simulation")
تمرير الرسائل، كمجموعة من الدوال المحلية إلى جانب دالة #py("dispatch") تختار العملية المحلية المناسبة، تماماً كما فعلنا مع كائن الحساب البنكي البسيط في القسم @sec:local-state-variables:
#idx("makewire", decl: true)
#snippet(```python
def make_wire():
    signal_value = 0
    action_functions = None
    def set_my_signal(new_value):
        nonlocal signal_value
        if signal_value != new_value:
            signal_value = new_value
            return call_each(action_functions)
        else:
            return "done"
    def accept_action_function(fun):
        nonlocal action_functions
        action_functions = pair(fun, action_functions)
        fun()
    def dispatch(m):
        return (signal_value if m == "get_signal"
                else set_my_signal if m == "set_signal"
                else accept_action_function if m == "add_action"
                else error("unknown operation -- wire", m))
    return dispatch
```)

تختبر الدالة المحلية #py("set_my_signal") ما إذا كانت قيمة الإشارة الجديدة تغير الإشارة على السلك. وإذا كان الأمر كذلك، فإنها تشغل كل من دوال الفعل، باستخدام الدالة التالية #py("call_each")، والتي تستدعي كل عنصر من العناصر في قائمة من الدوال بدون وسائط:
#idx("calleach", decl: true)
#snippet(```python
def call_each(functions):
    if is_none(functions):
        return "done"
    else:
        head(functions)()
        return call_each(tail(functions))
```)

تضيف الدالة المحلية #py("accept_action_function") الدالة المعطاة إلى قائمة الدوال المراد تشغيلها، ثم تشغل الدالة الجديدة مرة واحدة. (انظر التمرين @ex:accept-action.)

ومع إعداد دالة #py("dispatch") المحلية كما هو محدد، يمكننا تقديم الدوال التالية للوصول إلى العمليات المحلية على الأسلاك:#footnote[هذه الدوال مجرد سكر بناء جمل يسمح لنا بـ
#idx("syntactic sugar", sub: "function vs. data as")
#idx("syntax interface")
استخدام بناء الجمل الوظيفي العادي للوصول إلى الدوال المحلية للكائنات. ومن المدهش أنه يمكننا تبديل دور "الدوال" و "البيانات" بهذه الطريقة البسيطة. على سبيل المثال، إذا كتبنا #py("wire(\"get_signal\")") فنحن نفكر في #py("wire") كدالة تُستدعى مع الرسالة #py("\"get_signal\"") كدخل. وبدلاً من ذلك، فإن كتابة #py("get_signal(wire)") تشجعنا على التفكير في #py("wire") ككائن بيانات هو الدخل لدالة #py("get_signal"). والحقيقة في هذا الأمر هي أنه، في لغة يمكننا فيها التعامل مع الدوال ككائنات، لا يوجد فارق أساسي بين "الدوال" و "البيانات"، ويمكننا اختيار سكر بناء الجمل لدينا للسماح لنا بالبرمجة بأي أسلوب نختاره.]<foot:object-syntax>
#idx("getsignal", decl: true)#idx("setsignal", decl: true)#idx("addaction", decl: true)
#snippet(```python
def get_signal(wire):
    return wire("get_signal")

def set_signal(wire, new_value):
    return wire("set_signal")(new_value)

def add_action(wire, action_function):
    return wire("add_action")(action_function)
```)

والأسلاك، التي تمتلك إشارات متغيرة بمرور الوقت وقد ترتبط تدريجياً بأجهزة، هي نموذجية للكائنات القابلة للتغيير. ونحن نمذجناها كدوال بمتغيرات حالة محلية تُعدل بالإسناد. وعندما يُنشأ سلك جديد، تُخصص مجموعة جديدة من متغيرات الحالة و تُبنى دالة #py("dispatch") جديدة وتُرجع، الملتقطة للبيئة بمتغيرات الحالة الجديدة.

وتُشارك الأسلاك بين الأجهزة المختلفة التي رُبطت بها. وبالتالي، فإن التغيير المجري بالتفاعل مع جهاز واحد سيؤثر على جميع الأجهزة الأخرى المربوطة بالسلك. وينقل السلك التغيير إلى جيرانه باستدعاء دوال الفعل المحدثة له عندما أُقيمت الاتصالات.

#idx("digital-circuit simulation", sub: "representing wires")

=== أجندة المواعيد

#idx("digital-circuit simulation", sub: "agenda")

الشيء الوحيد المطلوب لإكمال المحاكي هو #py("after_delay").
والفكرة هنا هي أننا نحافظ على بنية بيانات، تُسمى #emph[الأجندة] (#en[agenda])، تحتوي على جدول زمني للأشياء المراد فعلها.
وتُعرف العمليات التالية للأجندة:

- #py("make_agenda()"): #idx("makeagenda") \ تُرجع أجندة خالية جديدة.
- #py("is_empty_agenda(")#meta("agenda")#py(")") #idx("isemptyagenda") \ تكون صحيحة إذا كانت الأجندة المحددة خالية.
- #py("first_agenda_item(")#meta("agenda")#py(")") #idx("firstagendaitem") \ تُرجع العنصر الأول في الأجندة.
- #py("remove_first_agenda_item(")#meta("agenda")#py(")") #idx("removefirstagendaitem") \ تعدل الأجندة بإزالة العنصر الأول.
- #py("add_to_agenda(")#meta("time")#py(",") #meta("action")#py(",") #meta("agenda")#py(")") #idx("addtoagenda") \ تعدل الأجندة بإضافة دالة الفعل المعطاة ليتم تشغيلها في الوقت المحدد.
- #py("current_time(")#meta("agenda")#py(")") #idx("currenttime") \ تُرجع الوقت الحالي للمحاكاة.

والأجندة الخاصة التي نستخدمها يُرمز لها بـ #py("the_agenda").
وتضيف الدالة #py("after_delay") عناصر جديدة إلى #py("the_agenda"):
#idx("afterdelay", decl: true)
#snippet(```python
def after_delay(delay, action):
    add_to_agenda(delay + current_time(the_agenda),
                  action,
                  the_agenda)
```)

وتُقاد المحاكاة بوساطة الدالة #py("propagate")، التي تنفذ كل دالة في #py("the_agenda") بالتسلسل.

وبشكل عام، مع تشغيل المحاكاة، ستُضاف عناصر جديدة إلى الأجندة، وتستمر #py("propagate") في المحاكاة طالما توجد عناصر في الأجندة:
#idx("propagate", decl: true)
#snippet(```python
def propagate():
    if is_empty_agenda(the_agenda):
        return "done"
    else:
        first_item = first_agenda_item(the_agenda)
        first_item()
        remove_first_agenda_item(the_agenda)
        return propagate()
```)

#idx("digital-circuit simulation", sub: "agenda")

=== عينة من المحاكاة

#idx("digital-circuit simulation", sub: "sample simulation")
#idx("half-adder", sub: "simulation of")

تُظهر الدالة التالية، التي تضع "مسباراً" (#en[probe]) على سلك، المحاكي في حالة عمل. ويخبر المسبار السلك بأنه كلما تغيرت قيمة إشارته، يجب عليه طباعة قيمة الإشارة الجديدة، إلى جانب الوقت الحالي واسم يحدد السلك.
#idx("probe", sub: "in digital-circuit simulator", decl: true)
#snippet(```python
def probe(name, wire):
    add_action(wire,
               lambda: display(name + " " +
                               str(current_time(the_agenda)) +
                               ", new value = " +
                               str(get_signal(wire))))
```)

نبدأ بتهيئة الأجندة وتحديد التأخيرات لصناديق الدوال الأولية:

#snippet(```python
the_agenda = make_agenda()
inverter_delay = 2
and_gate_delay = 3
or_gate_delay = 5
```)

الآن نحدد أربعة أسلاك، مع وضع مجسات على اثنين منها:

#snippet(```python
input_1 = make_wire()
input_2 = make_wire()
sum = make_wire()
carry = make_wire()

probe("sum", sum)
```)

#output(```python
input_1 = make_wire()
input_2 = make_wire()
sum = make_wire()
carry = make_wire()

probe("sum", sum)
```)

#snippet(```python
probe("carry", carry)
```)

#output(```python
probe("carry", carry)
```)

وبعد ذلك نربط الأسلاك في دارة جامع نصف (كما في الشكل @fig:half-adder)، ونحدد الإشارة على #py("input_1") إلى 1، ونشغل المحاكاة:

#snippet(```python
print(half_adder(input_1, input_2, sum, carry))
```)

#output(```python
print(half_adder(input_1, input_2, sum, carry))
```)

#snippet(```python
print(set_signal(input_1, 1))
```)

#output(```python
print(set_signal(input_1, 1))
```)

#snippet(```python
print(propagate())
```)

#output(```python
print(propagate())
```)

تتغير إشارة #py("sum") إلى 1 في الوقت 8.
ونحن الآن على بعد ثماني وحدات زمنية من بداية المحاكاة.
وعند هذه النقطة، يمكننا تحديد الإشارة على #py("input_2") إلى 1 والسماح للقيم بالانتشار:

#snippet(```python
print(set_signal(input_2, 1))
```)

#output(```python
print(set_signal(input_2, 1))
```)

#snippet(```python
print(propagate())
```)

#output(```python
print(propagate())
```)

تتغير إشارة #py("carry") إلى 1 في الوقت 11 وتتغير إشارة #py("sum") إلى 0 في الوقت 16.

#idx("digital-circuit simulation", sub: "sample simulation")
#idx("half-adder", sub: "simulation of")

#exercise(label-name: <ex:accept-action>, [
تحدد الدالة الداخلية #py("accept_action_function") المعرفة في #idx("makewire") #py("make_wire") أنه عند إضافة دالة فعل جديدة إلى سلك، تُشغل الدالة فوراً. اشرح سبب ضرورة هذه التهيئة. وبشكل خاص، تتبع مثال جامع النصف في الفقرات أعلاه واذكر كيف كانت ستختلف استجابة النظام إذا كنا قد عرفنا #py("accept_action_function") كالتالي:

#snippet(```python
def accept_action_function(fun):
    nonlocal action_functions
    action_functions = pair(fun, action_functions)
```)
])

=== تنفيذ الأجندة

#idx("digital-circuit simulation", sub: "agenda implementation")

أخيراً، نقدم تفاصيل بنية بيانات الأجندة، التي تحتفظ بالدوال المجدولة للتنفيذ المستقبلي.

وتتكون الأجندة من
#idx("time segment, in agenda")
#emph[شرائح زمنية] (#en[time segments]). وكل شريحة زمنية هي زوج يتكون من رقم (الوقت) و
#idx("queue", sub: "in simulation agenda")
طابور (انظر التمرين @ex:agenda-list) يحتفظ بالدوال المجدولة ليتم تشغيلها خلال تلك الشريحة الزمنية.
#idx("maketimesegment", decl: true)#idx("segmenttime", decl: true)#idx("segmentqueue", decl: true)
#snippet(```python
def make_time_segment(time, queue):
    return pair(time, queue)

def segment_time(s):
    return head(s)

def segment_queue(s):
    return tail(s)
```)

وتتألف الأجندة من شرائح زمنية، نتعامل معها باستخدام عمليات الطابور الموصوفة في القسم @sec:queues.

والأجندة نفسها هي جدول أحادي البعد
#idx("table", sub: "used in simulation agenda")
من الشرائح الزمنية. وهو يختلف عن الجداول الموصوفة في القسم @sec:tables في أن الشرائح ستكون مرتبة حسب ترتيب زيادة الوقت. وبالإضافة إلى ذلك، فإننا نخزن
#idx("current time, for simulation agenda")
#emph[الوقت الحالي] (أي وقت الفعل الأخير الذي عولج) عند رأس الأجندة. والأجندة المنشأة حديثاً ليس لها شرائح زمنية ووقتها الحالي 0:#footnote[الأجندة هي قائمة مروَّسة، مثل الجداول في القسم @sec:tables، ولكن بما أن القائمة مروَّسة بالوقت، فلا نحتاج لـ مروِّس وهمي إضافي (مثل السلسلة النصية #py("\"*table*\"") المستخدمة مع الجداول).]
#idx("makeagenda", decl: true)#idx("currenttime", decl: true)#idx("setcurrenttime", decl: true)#idx("segments", decl: true)#idx("setsegments", decl: true)#idx("firstsegment", decl: true)#idx("restsegments", decl: true)
#snippet(```python
def make_agenda():
    return llist(0)

def current_time(agenda):
    return head(agenda)

def set_current_time(agenda, time):
    set_head(agenda, time)

def segments(agenda):
    return tail(agenda)

def set_segments(agenda, segs):
    set_tail(agenda, segs)

def first_segment(agenda):
    return head(segments(agenda))

def rest_segments(agenda):
    return tail(segments(agenda))
```)

تكون الأجندة خالية إذا لم تكن لها شرائح زمنية:
#idx("isemptyagenda", decl: true)
#snippet(```python
def is_empty_agenda(agenda):
    return is_none(segments(agenda))
```)

ولإضافة فعل إلى أجندة، نفحص أولاً ما إذا كانت الأجندة خالية. وإذا كان الأمر كذلك، ننشئ شريحة زمنية للفعل ونسندها في الأجندة. بخلاف ذلك، نفحص الأجندة، فاحصين وقت كل شريحة. وإذا وجدنا شريحة لوقتنا المعين، نضيف الفعل إلى الطابور المرتبط بها. وإذا وصلنا إلى وقت أحدث من الوقت المعين لنا، ندرج شريحة زمنية جديدة في الأجندة قبلها مباشرة. وإذا وصلنا إلى نهاية الأجندة، يجب أن ننشئ شريحة زمنية جديدة عند النهاية.
#idx("addtoagenda", decl: true)
#snippet(```python
def add_to_agenda(time, action, agenda):
    def belongs_before(segs):
        return is_none(segs) or time < segment_time(head(segs))
    def make_new_time_segment(time, action):
        q = make_queue()
        insert_queue(q, action)
        return make_time_segment(time, q)
    def add_to_segments(segs):
        if segment_time(head(segs)) == time:
            insert_queue(segment_queue(head(segs)), action)
        else:
            rest = tail(segs)
            if belongs_before(rest):
                set_tail(segs, pair(make_new_time_segment(time, action),
                                    tail(segs)))
            else:
                add_to_segments(rest)
    segs = segments(agenda)
    if belongs_before(segs):
        set_segments(agenda,
                     pair(make_new_time_segment(time, action), segs))
    else:
        add_to_segments(segs)
```)

الدالة التي تزيل العنصر الأول من الأجندة تحذف العنصر عند مقدمة الطابور في الشريحة الزمنية الأولى. وإذا جعل هذا الحذف الشريحة الزمنية خالية، فإننا نزيلها من قائمة الشرائح:#footnote[لاحظ أن التعليمة الشرطية في هذه الدالة تحتوي على تعليمة #idx("statement", sub: "pass", decl: true) #idx("pass statement", decl: true) #py("pass") كتعليمتها البديلة التي لا تفعل شيئاً.]<foot:one-armed>
#idx("removefirstagendaitem", decl: true)
#snippet(```python
def remove_first_agenda_item(agenda):
    q = segment_queue(first_segment(agenda))
    delete_queue(q)
    if is_empty_queue(q):
        set_segments(agenda, rest_segments(agenda))
    else:
        pass
```)

يُعثر على عنصر الأجندة الأول عند مقدمة الطابور في الشريحة الزمنية الأولى. وكلما استخرجنا عنصراً، نحدث أيضاً الوقت الحالي:#footnote[وبهذه الطريقة، سيكون الوقت الحالي دائماً هو وقت الفعل المعالج مؤخراً. وتخزين هذا الوقت عند رأس الأجندة يضمن أنه سيظل متاحاً حتى لو أُزيلت الشريحة الزمنية المرتبطة به.]
#idx("firstagendaitem", decl: true)
#snippet(```python
def first_agenda_item(agenda):
    if is_empty_agenda(agenda):
        error("agenda is empty -- first_agenda_item")
    else:
        first_seg = first_segment(agenda)
        set_current_time(agenda, segment_time(first_seg))
        return front_queue(segment_queue(first_seg))
```)

#exercise(label-name: <ex:agenda-list>, [
تُحفظ الدوال المراد تشغيلها خلال كل شريحة زمنية للأجندة في طابور.
وبالتالي، فإن الدوال لكل شريحة تُستدعى بالترتيب الذي أُضيفت به إلى الأجندة (الأول دخولاً، الأول خروجاً). اشرح سبب وجوب استخدام هذا الترتيب. وبشكل خاص، تتبع سلوك بوابة وَ التي تتغير مدخلاتها من 0,1 إلى 1,0 في الشريحة نفسها واذكر كيف كان سيتختلف السلوك إذا كنا قد خزنّا دوال الشريحة في قائمة عادية، محددين ومزيلين الدوال فقط عند المقدمة (الأخير دخولاً، الأول خروجاً).
])

#idx("digital-circuit simulation")
#idx("digital-circuit simulation", sub: "agenda implementation")
