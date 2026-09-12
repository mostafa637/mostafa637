// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([بنية القوائم القابلة للتغيير], label-name: <sec:mutable-list-structure>)

#idx("mutable data objects", sub: "list structure")
#idx("list structure", sub: "mutable")
#idx("mutable data objects", sub: "pairs")
#idx("pair(s)", sub: "mutable")

يمكن استخدام العمليات الأساسية على الأزواج—#py("pair") و #py("head") و #py("tail")—لبناء بنية قائمة واختيار أجزاء من بنية القائمة، ولكنها غير قادرة على تعديل بنية القائمة. وينطبق الشيء نفسه على عمليات القوائم التي استخدمناها حتى الآن، مثل #py("append") و #py("list")، حيث يمكن تعريفها بدلالة #py("pair") و #py("head") و #py("tail").
ولتعديل بنيات القوائم نحتاج إلى عمليات جديدة.

ودالتا التعديل الأصليتان للأزواج هما
#idx("sethead (primitive function)")
#py("set_head")
و
#idx("settail (primitive function)")
#py("set_tail").
تأخذ الدالة #py("set_head") وسيطين، الأول منهما يجب أن يكون زوجاً. وتعدل هذا الزوج، مستبدلة مؤشر #py("head") بمؤشر إلى الوسيط الثاني لـ #py("set_head").#footnote[تُرجع الدالتان #py("set_head") و #py("set_tail") القيمة #py("None").
#idx("sethead (primitive function)", sub: "value of")
#idx("settail (primitive function)", sub: "value of")
ويجب استخدامهما لتأثيرهما فقط.]

كمثال، افترض أن #py("x") مرتبط بـ #py("llist(llist(\"a\", \"b\"), \"c\", \"d\")") و #py("y") بـ #py("llist(\"e\", \"f\")") كما هو موضح في الشكل @fig:two-lists.
إن تقييم التعبير #py("set_head(x, y)") يغير الزوج الذي ارتبط به #py("x")، مسبدلاً مؤشر #py("head") بقيمة #py("y"). وتُظهر نتيجة العملية في الشكل @fig:set-car.
وقد أُعدلت البنية #py("x") وهي الآن تكافئ #py("llist(llist(\"e\", \"f\"), \"c\", \"d\")").
والأزواج التي تمثل القائمة #py("llist(\"a\", \"b\")")، التي حددها المؤشر المستبدل، أصبحت الآن منفصلة عن البنية الأصلية.#footnote[نرى من هذا أن عمليات التعديل على القوائم يمكن أن تنشئ "قمامة" لا تنتمي إلى أي بنية يمكن الوصول إليها. وسوف نرى في القسم @sec:gc أن أنظمة إدارة الذاكرة في بايثون تتضمن
#idx("garbage collection", sub: "mutation and")
#emph[جامع قمامة] (#en[garbage collector])، يحدد ويعيد استخدام مساحة الذاكرة التي تستخدمها الأزواج غير المطلوبة.]

#sicp-figure(image("/images/img_javascript/ch3-Z-G-13.svg", width: 70%), caption: [القائمتان #py("x"): #py("llist(llist(\"a\", \"b\"), \"c\", \"d\")") و #py("y"): #py("llist(\"e\", \"f\")").], label-name: <fig:two-lists>)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-14.svg", width: 70%), caption: [تأثير #py("set_head(x, y)") على القائمتين في الشكل @fig:two-lists.], label-name: <fig:set-car>)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-15.svg", width: 70%), caption: [تأثير #py("z = pair(y, tail(x))") على القائمتين في الشكل @fig:two-lists.], label-name: <fig:list-cons>)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-16.svg", width: 70%), caption: [تأثير #py("set_tail(x, y)") على القائمتين في الشكل @fig:two-lists.], label-name: <fig:set-cdr>)

قارن الشكل @fig:set-car بالشكل @fig:list-cons، الذي يوضح نتيجة تنفيذ

#snippet(```python
z = pair(y, tail(x))
```)

مع كون #py("x") و #py("y") مرتبطين بالقائمتين الأصليتين في الشكل @fig:two-lists.
الاسم #py("z") مرتبط الآن بزوج جديد أُنشئ بوساطة عملية #py("pair")؛ والقائمة التي ارتبط بها #py("x") تظل غير متبدلة.

وعملية #py("set_tail") تشبه #py("set_head").
والفارق الوحيد هو أن مؤشر #py("tail") للزوج، بدلاً من مؤشر #py("head")، هو الذي يُستبدل. وتأثير تنفيذ #py("set_tail(x, y)") على القائمتين في الشكل @fig:two-lists موضح في الشكل @fig:set-cdr.
وهنا استُبدل مؤشر #py("tail") لـ #py("x") بمؤشر إلى #py("llist(\"e\", \"f\")").
وأيضاً، فإن القائمة #py("llist(\"c\", \"d\")")، التي كانت تُشكل #py("tail") لـ #py("x")، أصبحت منفصلة عن البنية.

تبني الدالة #py("pair") بنية قائمة جديدة بإنشاء أزواج جديدة، بينما تعدل #py("set_head") و #py("set_tail") الأزواج الموجودة.
وفي الواقع، كان بإمكاننا
#idx("pair (primitive function)", sub: "implemented with mutators")
تنفيذ #py("pair") بدلالة دالتي التعديل، إلى جانب دالة #py("get_new_pair")، التي تُرجع زوجاً جديداً ليس جزءاً من أي بنية قائمة موجودة.
فنحصل على الزوج الجديد، ونحدد مؤشري #py("head") و #py("tail") الخاصين به إلى الكائنين المحددين، ونُرجع الزوج الجديد كنتيجة لـ #py("pair").#footnote[سيُظهر القسم @sec:memory-as-vectors كيف يمكن لنظام إدارة الذاكرة تنفيذ #py("get_new_pair").]

#idx("pair (primitive function)", sub: "implemented with mutators", decl: true)
#snippet(```python
def pair(x, y):
    fresh = get_new_pair()
    set_head(fresh, x)
    set_tail(fresh, y)
    return fresh
```)

#exercise(label-name: <ex:append>, [
أُدخلت دالة إلحاق القوائم التالية في القسم @sec:sequences:

#idx("append", decl: true)

#snippet(```python
def append(x, y):
    return (y
            if is_none(x)
            else pair(head(x), append(tail(x), y)))
```)

تشكل الدالة #py("append") قائمة جديدة بإرفاق عناصر #py("x") متتالية إلى مقدمة #py("y").
ودالة
#idx("append", sub: "appendmutator vs.")
#py("append_mutator")
تشبه #py("append")، ولكنها دالة تعديل لا دالة إنشاء. وهي تُلحق القوائم بربطها معاً، وتعديل الزوج الأخير لـ #py("x") بحيث يصبح #py("tail") الخاص به الآن هو #py("y"). (من الخطأ استدعاء #py("append_mutator") مع قائمة خالية #py("x").)
#idx("appendmutator", decl: true)
#snippet(```python
def append_mutator(x, y):
    set_tail(last_pair(x), y)
    return x
```)

وهنا #py("last_pair") هي دالة تُرجع الزوج الأخير في وسيطها:

#idx("lastpair", decl: true)
#snippet(```python
def last_pair(x):
    return (x
            if is_none(tail(x))
            else last_pair(tail(x)))
```)

فكر في التفاعل

#snippet(```python
x = llist("a", "b")
```)

#snippet(```python
y = llist("c", "d")
```)

#snippet(```python
z = append(x, y)
```)

#snippet(```python
print(z)
```)

#output(```python
print(z)
```)

#snippet(```python
print(tail(x))
```)

#output(```python
print(tail(x))
```)

#snippet(```python
w = append_mutator(x, y)
```)

#snippet(```python
print(w)
```)

#output(```python
print(w)
```)

#snippet(```python
print(tail(x))
```)

#output(```python
print(tail(x))
```)

ما هي الاستجابات المفقودة مكان #meta("response")؟
ارسم مخططات الصناديق والمؤشرات لشرح إجابتك.
])

#exercise(label-name: <ex:make-cycle>, [
فكر في دالة
#idx("cycle in list")
#py("make_cycle")
التالية، والتي تستخدم دالة #py("last_pair") المحددة في التمرين @ex:append:
#idx("makecycle", decl: true)
#snippet(```python
def make_cycle(x):
    set_tail(last_pair(x), x)
    return x
```)

ارسم مخطط الصناديق والمؤشرات الذي يوضح البنية #py("z") المنشأة بـ

#snippet(```python
z = make_cycle(llist("a", "b", "c"))
```)

ماذا يحدث إذا حاولنا حساب #py("last_pair(z)")؟
])

#exercise(label-name: <ex:mystery>, [
الدالة التالية مفيدة للغاية، على الرغم من غموضها:
#idx("mystery", decl: true)
#snippet(```python
def mystery(x):
    def loop(x, y):
        if is_none(x):
            return y
        else:
            temp = tail(x)
            set_tail(x, y)
            return loop(temp, x)
    return loop(x, None)
```)

تستخدم الدالة #py("loop") الاسم "المؤقت" #py("temp") للابتعاد بالقيمة القديمة لـ #py("tail") لـ #py("x")، حيث إن #py("set_tail") في السطر التالي يدمر #py("tail").
اشرح ما تفعله #py("mystery") بشكل عام. نفترض أن #py("v") معرفة بـ

#snippet(```python
v = llist("a", "b", "c", "d")
```)

ارسم مخطط الصناديق والمؤشرات الذي يمثل القائمة التي ارتبط بها #py("v"). ونفترض أننا قمنا بتقييم

#snippet(```python
w = mystery(v)
```)

ارسم مخططات الصناديق والمؤشرات التي توضح البنيتين #py("v") و #py("w") بعد تقييم هذا البرنامج.
ما الذي سيُطبع كقيمتين لـ #py("v") و #py("w")؟
])

#idx("mutable data objects", sub: "list structure")
#idx("list structure", sub: "mutable")
#idx("mutable data objects", sub: "pairs")
#idx("pair(s)", sub: "mutable")

#subheading([التشارك والهوية])

#idx("data", sub: "shared")
#idx("shared data")

ذكرنا في القسم @sec:costs-of-assignment المسائل النظرية لـ
#idx("sameness and change", sub: "shared data and")
#idx("change and sameness", sub: "shared data and")
"التماثل" و "التغير" التي يثيرها تقديم الإسناد. وتظهر هذه المسائل عملياً عندما تكون الأزواج الفردية #emph[مشاركة] بين كائنات بيانات مختلفة.
على سبيل المثال، فكر في البنية المشكلة بـ

#snippet(```python
x = llist("a", "b")
z1 = pair(x, x)
```)

كما هو موضح في الشكل @fig:identity1، فإن #py("z1") هو زوج يشير كل من #py("head") و #py("tail") الخاصين به إلى الزوج نفسه #py("x"). وتشارك #py("x") بوساطة #py("head") و #py("tail") لـ #py("z1") هو نتيجة للطريقة المباشرة التي تنفذ بها #py("pair"). وبشكل عام، فإن استخدام #py("pair") لبناء القوائم ينتج عنه بنية متبادلة الربط من الأزواج تُشارك فيها العديد من الأزواج الفردية بوساطة بنيات عديدة مختلفة.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-17.svg", width: 70%), caption: [القائمة #py("z1") المشكلة بـ #py("pair(x, x)").], label-name: <fig:identity1>)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-18.svg", width: 70%), caption: [القائمة #py("z2") المشكلة بـ #py("pair(llist(\"a\", \"b\"), llist(\"a\", \"b\"))").], label-name: <fig:identity2>)

وعلى النقيض من الشكل @fig:identity1، يُظهر الشكل @fig:identity2 البنية المنشأة بـ

#snippet(```python
z2 = pair(llist("a", "b"), llist("a", "b"))
```)

في هذه البنية، الأزواج في القائمتين #py("llist(\"a\", \"b\")") متميزة، على الرغم من أنها تحتوي على السلاسل النصية نفسها.#footnote[الزوجان متميزان لأن كل استدعاء لـ #py("pair") يُرجع زوجاً جديداً. والسلاسل النصية
#idx("string(s)", sub: "uniqueness of")
"نفسها" بفي المعنى أنها بيانات أولية (مثل الأرقام تماماً) تتكون من الأحرف نفسها بالترتيب نفسه. وبما أن بايثون لا توفر طريقة لتعديل سلسلة نصية، فإن أي تشارك قد يقرر مصممو مفسر بايثون تنفيذه للسلاسل النصية يكون غير قابل للاكتشاف. ونحن نعتبر البيانات الأولية مثل الأعداد والقيما البولية والسلاسل النصية #emph[متطابقة] إذا وفقط إذا كانت #emph[غير قابلة للتمييز].]<foot:symbol-sharing>

وعند التفكير فيها كقائمة، فإن كلاً من #py("z1") و #py("z2") تمثلان القائمة "نفسها":

#snippet(```python
print(llist(llist("a", "b"), "a", "b"))
```)

وبشكل عام، فإن التشارك غير قابل للاكتشاف تماماً إذا عملنا على القوائم باستخدام #py("pair") و #py("head") و #py("tail") فقط.
ولكن إذا سمحنا بتعديل بنية القائمة، يصبح التشارك مهماً. وكمثال على الفارق الذي يمكن أن يحدثه التشارك، فكر في الدالة التالية، التي تعدل #py("head") البنية المطبقة عليها:

#snippet(```python
def set_to_wow(x):
    set_head(head(x), "wow")
    return x
```)

على الرغم من أن #py("z1") و #py("z2") هما البنية "نفسها"، إلا أن تطبيق #py("set_to_wow") عليهما يعطي نتائج مختلفة. فمع #py("z1")، يغير تعديل #py("head") أيضاً #py("tail")، لأن #py("head") و #py("tail") في #py("z1") هما الزوج نفسه. ومع #py("z2")، فإن #py("head") و #py("tail") متميزان، لذا فإن #py("set_to_wow") تعدل #py("head") فقط:

#snippet(```python
print(z1)
```)

#output(```python
print(z1)
```)

#snippet(```python
print(set_to_wow(z1))
```)

#output(```python
print(set_to_wow(z1))
```)

#snippet(```python
print(z2)
```)

#output(```python
print(z2)
```)

#snippet(```python
print(set_to_wow(z2))
```)

#output(```python
print(set_to_wow(z2))
```)

إحدى الطرق لاكتشاف التشارك في بنيات القوائم هي استخدام المحمول الأولي
#idx("is", sub: "as equality of pointers", decl: true)
#py("is").
وعند تطبيقه على قيمتين غير أوليين، يختبر
#py("x is y")
ما إذا كان #py("x") و #py("y") هما الكائن نفسه (أي ما إذا كان #py("x") و #py("y") متساويين كمؤشرات).

وبالتالي، مع #py("z1") و #py("z2") كما هما معرفان في الشكلين @fig:identity1 و @fig:identity2، يكون #py("head(z1) is tail(z1)") صحيحاً و #py("head(z2) is tail(z2)") خاطئاً.

وكما سيتضح في الأقسام التالية، يمكننا استغلال التشارك لتوسيع حصيلة بنيات البيانات التي يمكن تمثيلها بوساطة الأزواج بشكل كبير. ومن ناحية أخرى، يمكن أن يكون التشارك
#idx("mutable data objects", sub: "shared data")
خطيراً أيضاً، حيث إن التعديلات المجراة على بنيات ستؤثر أيضاً على بنيات أخرى تتشارك الأجزاء المعدلة. وتعديل العمليات #py("set_head") و #py("set_tail") يجب أن يُستخدم بحذر؛ فما لم تكن لدينا معرفة جيدة بكيفية تشارك كائنات البيانات لدينا، قد يكون للتعديل نتائج غير متوقعة.#footnote[تعكس دقة التعامل مع تشارك كائنات البيانات القابلة للتغيير المسائل الأساسية لـ "التماثل" و "التغير" التي أُثيرت في القسم @sec:costs-of-assignment. وذكرنا هناك أن قبول التغير في لغتنا يتطلب أن يكون للكائن المركب "هوية" هي شيء مختلف عن الأجزاء التي يتكون منها. وفي بايثون، نعتبر هذه "الهوية" هي الجودة التي يختبرها #py("is")، أي المساواة في المؤشرات. وبما أنه في معظم تنفيذات بايثون يكون المؤشر في الأساس عنوان ذاكرة، فإننا "نحل مسألة" تحديد هوية الكائنات باشتراط أن كائن البيانات "نفسه" هو المعلومات المخزنة في مجموعة معينة من مواقع الذاكرة في الحاسوب. وهذا يكفي لبرامج بايثون البسيطة، ولكنه ليس طريقة عامة لحل مسألة "التماثل" في النماذج الحسابية.]

#exercise(label-name: <ex:3_15>, [
ارسم مخططات الصناديق والمؤشرات لشرح تأثير #py("set_to_wow") على البنيتين #py("z1") و #py("z2") أعلاه.
])

#exercise(label-name: <ex:count-pairs>, [
يقرر بن بتديدل كتابة دالة لحساب عدد الأزواج في أي بنية قائمة.
ويعلل ذلك: "الأمر سهل. عدد الأزواج في أي بنية هو العدد في #py("head") بالإضافة إلى العدد في #py("tail") بالإضافة إلى واحد إضافي لعد الزوج الحالي." لذلك يكتب بن الدالة التالية:
#idx("countpairs", decl: true)
#snippet(```python
def count_pairs(x):
    return (0
            if not is_pair(x)
            else count_pairs(head(x)) +
                 count_pairs(tail(x)) +
                 1)
```)

أظهر أن هذه الدالة غير صحيحة. وبشكل خاص، ارسم مخططات الصناديق والمؤشرات التي تمثل بنيات قوائم مكونة من ثلاثة أزواج بالظبط والتي ترجع لها دالة بن 3؛ وترجع 4؛ وترجع 7؛ ولا ترجع أبداً على الإطلاق.
])

#exercise(label-name: <ex:count-pairs2>, [
ابتكر نسخة صحيحة من دالة #py("count_pairs") للتمرين @ex:count-pairs ترجع عدد الأزواج المتميزة في أي بنية. (تلميح: تنقل في البنية، مع الحفاظ على بنية بيانات مساعدة تُستخدم لتتبع الأزواج التي حُسبت بالفعل.)
])

#exercise(label-name: <ex:find-cycle>, [
اكتب دالة تفحص قائمة وتحدد
#idx("cycle in list", sub: "detecting")
ما إذا كانت تحتوي على دورة، أي ما إذا كان البرنامج الذي حاول العثور على نهاية القائمة عن طريق أخذ #py("tail")s متتالية سيدخل في حلقة لانهائية. أنشأ التمرين @ex:make-cycle مثل هذه القوائم.
])

#exercise(label-name: <ex:3_19>, [
أعد التمرين @ex:find-cycle باستخدام خوارزمية تستهلك مساحة ثابتة فقط. (يتطلب هذا فكرة ذكية جداً.)
])

#idx("data", sub: "shared")
#idx("shared data")

#subheading([التعديل هو مجرد إسناد])

#idx("mutable data objects", sub: "functional representation of")
#idx("mutable data objects", sub: "implemented with assignment")
#idx("pair(s)", sub: "functional representation of")
#idx("functional representation of data", sub: "mutable data")

عندما قدمنا البيانات المركبة، لاحظنا في القسم @sec:data- أنه يمكن تمثيل الأزواج بشكل خالص بدلالة الدوال:
#idx("pair (primitive function)", sub: "functional implementation of", decl: true)#idx("head (primitive function)", sub: "functional implementation of", decl: true)#idx("tail (primitive function)", sub: "functional implementation of", decl: true)
#snippet(```python
def pair(x, y):
    def dispatch(m):
        return (x
                if m == "head"
                else y
                if m == "tail"
                else error("undefined operation -- pair", m))
    return dispatch

def head(z):
    return z("head")

def tail(z):
    return z("tail")
```)

والملاحظة نفسها صحيحة بالنسبة للبيانات القابلة للتغيير. يمكننا تنفيذ كائنات البيانات القابلة للتغيير كدوال باستخدام الإسناد والحالة المحلية. على سبيل المثال، يمكننا توسيع تنفيذ الأزواج أعلاه للتعامل مع #py("set_head") و #py("set_tail") بطريقة مماثلة للطريقة التي نفذنا بها الحسابات البنكية باستخدام #py("make_account") في القسم @sec:local-state-variables:

#idx("pair (primitive function)", sub: "functional implementation of", decl: true)#idx("head (primitive function)", sub: "functional implementation of", decl: true)#idx("tail (primitive function)", sub: "functional implementation of", decl: true)#idx("sethead (primitive function)", sub: "functional implementation of", decl: true)#idx("settail (primitive function)", sub: "functional implementation of", decl: true)
#snippet(```python
def pair(x, y):
    def set_x(v):
        nonlocal x
        x = v
    def set_y(v):
        nonlocal y
        y = v
    return lambda m: (x
                      if m == "head"
                      else y
                      if m == "tail"
                      else set_x
                      if m == "set_head"
                      else set_y
                      if m == "set_tail"
                      else error("undefined operation -- pair", m))

def head(z):
    return z("head")

def tail(z):
    return z("tail")

def set_head(z, new_value):
    z("set_head")(new_value)
    return z

def set_tail(z, new_value):
    z("set_tail")(new_value)
    return z
```)

إن الإسناد هو كل ما يلزم، نظرياً، لتبرير سلوك البيانات القابلة للتغيير. وبمجرد أن نقبل الإسناد في لغتنا، فإننا نثير كل المسائل، ليس فقط للإسناد، ولكن للبيانات القابلة للتغيير بشكل عام.#footnote[ومن ناحية أخرى، من وجهة نظر التنفيذ، يتطلب الإسناد منا تعديل البيئة، والتي هي في حد ذاتها بنية بيانات قابلة للتغيير. وبالتالي، فإن الإسناد والتعديل متكافئان: يمكن تنفيذ كل منهما بدلالة الآخر.]

#exercise(label-name: <ex:cons-with-assignment>, [
ارسم مخططات البيئة لتوضيح تقييم تسلسل التعليمات

#snippet(```python
x = pair(1, 2)
z = pair(x, x)
set_head(tail(z), 17)
```)

#snippet(```python
print(head(x))
```)

#output(```python
print(head(x))
```)

باستخدام التنفيذ الوظيفي للأزواج المعطى أعلاه. (قارن التمرين @ex:two-accounts.)
])

#idx("mutable data objects", sub: "functional representation of")
#idx("mutable data objects", sub: "implemented with assignment")
#idx("pair(s)", sub: "functional representation of")
#idx("functional representation of data", sub: "mutable data")
#idx("mutable data objects")
