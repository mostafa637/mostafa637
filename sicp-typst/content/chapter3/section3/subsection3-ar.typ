// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([تمثيل الجداول], label-name: <sec:tables>)

#idx("table")

عندما درسنا الطرق المختلفة لتمثيل المجموعات في الفصل @chap:data، ذكرنا في القسم @sec:representing-sets مهمة الاحتفاظ بـ جدول سجلات
#idx("key of a record", sub: "in a table")
مفهرسة بمفاتيح تعريفية. وفي تنفيذ البرمجة الموجهة بالبيانات في القسم @sec:data-directed، استخدمنا بكثرة الجداول ثنائية الأبعاد، التي تخزن وتسترجع فيها المعلومات باستخدام مفتاحين. وهنا نرى كيفية بناء الجداول كبنيات قائمة قابلة للتغيير.

ننظر أولاً في
#idx("table", sub: "one-dimensional")
جدول أحادي البعد، تُخزن فيه كل قيمة تحت مفتاح واحد. وننفذ الجدول كقائمة سجلات، يُنفذ كل منها كزوج يتكون من مفتاح والقيمة المرتبطة به. وتُرتبط السجلات معاً لتشكل قائمة بوساطة أزواج تشير #py("head")s الخاصة بها إلى السجلات المتتالية. وتُسمى هذه الأزواج المرتبطة بـ
#idx("table", sub: "backbone of")
#emph[العمود الفقري] (#en[backbone]) للجدول. وللحصول على مكان يمكننا تغييره عندما نضيف سجلاً جديداً إلى الجدول، نبني الجدول كالتالي:
#idx("headed list")
#idx("list(s)", sub: "headed")
#emph[قائمة مروَّسة] (#en[headed list]). والقائمة المروَّسة تحتوي على زوج عمود فقري خاص عند البداية، يحمل "سجلاً" وهمياً—في هذه الحالة
السلسلة النصية #py("\"*table*\"") المختارة عشوائياً.
ويُظهر الشكل @fig:table مخطط الصناديق والمؤشرات للجدول

#snippet(```python
a: 1
b: 2
c: 3
```)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-22.svg", width: 70%), caption: [جدول ممثل كقائمة مروَّسة.], label-name: <fig:table>)

ولاستخراج المعلومات من جدول نستخدم الدالة
#py("lookup")،
والتي تأخذ مفتاحاً كوسيط وتُرجع القيمة المرتبطة به (أو
#py("None")
إذا لم تكن هناك قيمة مخزنة تحت ذلك المفتاح).
وتُعرف الدالة #py("lookup") بدلالة عملية #py("assoc")،
والتي تتوقع مفتاحاً وقائمة سجلات كوسائط. لاحظ أن
#py("assoc") لا ترى السجل الوهمي أبداً.
وتُرجع الدالة #py("assoc") السجل الذي يحتوي على المفتاح المعطى ك#py("head") الخاص به.#footnote[بما أن #py("assoc") تستخدم
#py("equal")، فيمكنها التعرف على المفاتيح التي تكون سلاسل نصية، أو أعداداً، أو بنية قائمة.]
ثم تتحقق الدالة #py("lookup") لترى ما إذا كان السجل الناتج المُرجع بوساطة #py("assoc") ليس
#py("None")،
وتُرجع القيمة (#py("tail")) للسجل.

#idx("lookup", sub: "in one-dimensional table", decl: true)#idx("assoc", decl: true)
#snippet(```python
def lookup(key, table):
    record = assoc(key, tail(table))
    return (None
            if is_none(record)
            else tail(record))

def assoc(key, records):
    return (None
            if is_none(records)
            else head(records)
            if key == head(head(records))
            else assoc(key, tail(records)))
```)

ولإدراج قيمة في جدول تحت مفتاح محدد، نستخدم أولاً
#py("assoc") لرؤية ما إذا كان هناك سجل بالفعل في الجدول بهذا المفتاح. وإذا لم يكن كذلك، نشكل سجلاً جديداً بربط #py("pair") المفتاح والقيمة، ونسند هذا عند رأس قائمة سجلات الجدول، بعد السجل الوهمي. وإذا كان هناك سجل بالفعل بهذا المفتاح، نحدد #py("tail") هذا السجل للقيمة الجديدة المحددة. ويوفر لنا رأس الجدول موقعاً ثابتاً للتعديل من أجل إدراج السجل الجديد.#footnote[وبالتالي، فإن أول زوج عمود فقري هو الكائن الذي يمثل الجدول "نفسه"؛ أي أن المؤشر إلى الجدول هو مؤشر إلى هذا الزوج. وهذا الزوج نفسه يبدأ الجدول دائماً. وإذا لم نرتّب الأمور بهذه الطريقة، فإن #py("insert") كانت ستحتاج لـ إرجاع قيمة جديدة لبداية الجدول عندما تضيف سجلاً جديداً.]
#idx("insert", sub: "in one-dimensional table", decl: true)
#snippet(```python
def insert(key, value, table):
    record = assoc(key, tail(table))
    if is_none(record):
        set_tail(table,
                 pair(pair(key, value), tail(table)))
    else:
        set_tail(record, value)
    return "ok"
```)

ولبناء جدول جديد، ننشئ مجرد قائمة تحتوي فقط على السلسلة النصية #py("\"*table*\""):
#idx("maketable", sub: "one-dimensional table", decl: true)
#snippet(```python
def make_table():
    return llist("*table*")
```)

#idx("table", sub: "one-dimensional")

#subheading([الجداول ثنائية الأبعاد])

#idx("table", sub: "two-dimensional")

في الجدول ثنائي الأبعاد، تُفهرس كل قيمة بمفتاحين. ويمكننا بناء مثل هذا الجدول كجدول أحادي البعد يحدد فيه كل مفتاح جدولاً فرعياً.
ويُظهر الشكل @fig:2dtable مخطط الصناديق والمؤشرات للجدول

#snippet(```python
"math":
    "+": 43
    "-": 45
    "*": 42
"letters":
    "a": 97
    "b": 98
```)

والذي يحتوي على جدولين فرعيين. (ولا تحتاج الجداول الفرعية لسلسلة نصية ترويسة خاصة، حيث إن المفتاح الذي يحدد الجدول الفرعي يخدم هذا الغرض.)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-23.svg", width: 70%), caption: [جدول ثنائي الأبعاد.], label-name: <fig:2dtable>)

وعندما نبحث عن عنصر، نستخدم المفتاح الأول لتحديد الجدول الفرعي الصحيح. ثم نستخدم المفتاح الثاني لتحديد السجل داخل الجدول الفرعي.

#idx("lookup", sub: "in two-dimensional table", decl: true)
#snippet(```python
def lookup(key_1, key_2, table):
    subtable = assoc(key_1, tail(table))
    if is_none(subtable):
        return None
    else:
        record = assoc(key_2, tail(subtable))
        return (None
                if is_none(record)
                else tail(record))
```)

ولإدراج عنصر جديد تحت زوج من المفاتيح، نستخدم #py("assoc") لرؤية ما إذا كان هناك جدول فرعي مخزن تحت المفتاح الأول. وإذا لم يكن كذلك، نبني جدولاً فرعياً جديداً يحتوي على السجل المالي الوحيد (#py("key_2"), #py("value")) ونسنده في الجدول تحت المفتاح الأول. وإذا كان هناك جدول فرعي موجود بالفعل للمفتاح الأول، ندرج السجل الجديد في هذا الجدول الفرعي، باستخدام طريقة الإدراج للجداول أحادية البعد الموصوفة أعلاه:
#idx("insert", sub: "in two-dimensional table", decl: true)
#snippet(```python
def insert(key_1, key_2, value, table):
    subtable = assoc(key_1, tail(table))
    if is_none(subtable):
        set_tail(table,
                 pair(llist(key_1, pair(key_2, value)), tail(table)))
    else:
        record = assoc(key_2, tail(subtable))
        if is_none(record):
            set_tail(subtable,
                     pair(pair(key_2, value), tail(subtable)))
        else:
            set_tail(record, value)
    return "ok"
```)

#idx("table", sub: "two-dimensional")

#subheading([إنشاء الجداول المحلية])

#idx("table", sub: "local")

تأخذ العمليتان #py("lookup") و #py("insert") المحدادتان أعلاه الجدول كوسيط. وهذا يمكننا من استخدام البرامج التي تصل إلى أكثر من جدول واحد. والطريقة الأخرى للتعامل مع الجداول المتعددة هي أن تكون لدينا دوال #py("lookup") و #py("insert") منفصلة لكل جدول. ويمكننا فعل ذلك بتمثيل الجدول إجرائياً، ككائن يحافظ على جدول داخلي كجزء من حالته المحلية. وعند إرسال رسالة مناسبة، يقدم "كائن الجدول" هذا الدالة لتشغيلها على الجدول الداخلي. وإليك مولد للجداول ثنائية الأبعاد الممثلة بهذه الطريقة:
#idx("maketable", sub: "message-passing implementation", decl: true)
#snippet(```python
def make_table():
    local_table = llist("*table*")
    def lookup(key_1, key_2):
        subtable = assoc(key_1, tail(local_table))
        if is_none(subtable):
            return None
        else:
            record = assoc(key_2, tail(subtable))
            return (None
                    if is_none(record)
                    else tail(record))
    def insert(key_1, key_2, value):
        subtable = assoc(key_1, tail(local_table))
        if is_none(subtable):
            set_tail(local_table,
                     pair(llist(key_1, pair(key_2, value)),
                          tail(local_table)))
        else:
            record = assoc(key_2, tail(subtable))
            if is_none(record):
                set_tail(subtable,
                         pair(pair(key_2, value), tail(subtable)))
            else:
                set_tail(record, value)
    def dispatch(m):
        return (lookup if m == "lookup"
                else insert if m == "insert"
                else error("unknown operation -- table", m))
    return dispatch
```)

باستخدام #py("make_table")، كان بإمكاننا
#idx("operation-and-type table", sub: "implementing")
تنفيذ العمليتين #py("get") و #py("put") المستخدمتين في القسم @sec:data-directed للبرمجة الموجهة بالبيانات، كما يلي:

#idx("get", decl: true)#idx("put", decl: true)
#snippet(```python
operation_table = make_table()
get = operation_table("lookup")
put = operation_table("insert")
```)

تأخذ الدالة #py("get") مفتاحين كوسائط، وتأخذ #py("put") مفتاحين وقيمة كوسائط. وتصل كلا العمليتين إلى الجدول المحلي نفسه، والمغلف داخل الكائن المنشأ بالاستدعاء لـ #py("make_table").
#idx("table", sub: "local")

#exercise(label-name: <ex:numeric-keys>, [
في تنفيذات الجداول أعلاه، تُختبر المفاتيح
#idx("table", sub: "testing equality of keys")
#idx("key of a record", sub: "testing equality of")
للمساواة باستخدام #py("equal") (المستدعاة بوساطة #py("assoc")). وهذا ليس دائماً الاختبار المناسب. على سبيل المثال، قد يكون لدينا جدول بمفاتيح رقمية لا نحتاج فيها إلى تطابق دقيق مع الرقم الذي نبحث عنه، ولكن فقط رقم داخل تفاوت معين منه. صمم منشئ جدول #py("make_table") يأخذ كوسيط دالة #py("same_key") تُستخدم لاختبار "مساواة" المفاتيح.
ويجب أن تُرجع الدالة #py("make_table") دالة #py("dispatch") يمكن استخدامها للوصول إلى دوال #py("lookup") و #py("insert") المناسبة لجدول محلي.
])

#exercise(label-name: <ex:3_25>, [
بالتعميم على الجداول أحادية وثنائية الأبعاد، أظهر كيفية تنفيذ جدول تُخزن فيه القيم تحت
#idx("table", sub: "n-dimensional")
عدد عشوائي من المفاتيح وقد تُخزن قيم مختلفة تحت أعداد مختلفة من المفاتيح.
ويجب أن تأخذ الدالتان #py("lookup") و #py("insert") كدخل قائمة مفاتيح تُستخدم للوصول إلى الجدول.
])

#exercise(label-name: <ex:3_26>, [
لبحث جدول كما هو منفذ أعلاه، يحتاج المرء إلى الفحص عبر قائمة السجلات. وهذا في الأساس هو تمثيل القائمة غير المرتبة في القسم @sec:representing-sets. وللجداول الكبيرة، قد يكون من الأكثر كفاءة هيكلة الجدول بطريقة مختلفة.
صف تنفيذ جدول تُنظم فيه سجلات (المفتاح، القيمة) باستخدام
#idx("binary tree", sub: "table structured as")
#idx("table", sub: "represented as binary tree vs. unordered list")
شجرة ثنائية، بفرض أن المفاتيح يمكن ترتيبها بطريقة ما (على سبيل المثال، عددياً أو أبجدياً). (قارن التمرين @ex:set-lookup-binary-tree في الفصل @chap:data.)
])

#exercise(label-name: <ex:memoization>, [
#emph[التذكر] (#en[Memoization] -
#idx("memoization")
#idx("tabulation")
#idx("table", sub: "used to store computed values")
وتُسمى أيضاً #emph[الجدولة] - #en[tabulation]) هي تقنية تمكن دالة من تسجيل القيم التي حُسبت سابقاً في جدول محلي.
ويمكن أن تحدث هذه التقنية فارقاً شاسعاً في أداء البرنامج.
وتحافظ الدالة المحفوظة بالتذكر على جدول تُخزن فيه قيم الاستدعاءات السابقة باستخدام الوسائط التي أنتجت القيم كمفاتيح. وعندما يُطلب من الدالة المحفوظة بالتذكر حساب قيمة، فإنها تفحص الجدول أولاً لرؤية ما إذا كانت القيمة موجودة بالفعل، وإذا كان الأمر كذلك، تُرجع تلك القيمة فقط. بخلاف ذلك، تحسب القيمة الجديدة بالطريقة العادية وتخزنها في الجدول.
وكمثال على التذكر، تذكر من القسم @sec:tree-recursion العملية الأسية لحساب أعداد فيبوناتشي:

#snippet(```python
def fib(n):
    return (0
            if n == 0
            else 1
            if n == 1
            else fib(n - 1) + fib(n - 2))
```)

والنسخة المحفوظة بالتذكر من الدالة نفسها هي

#idx("fib", sub: "with memoization", decl: true)#idx("memofib", decl: true)
#snippet(```python
memo_fib = memoize(lambda n: (0
                              if n == 0
                              else 1
                              if n == 1
                              else memo_fib(n - 1) +
                                   memo_fib(n - 2)))
```)

حيث يُعرف المذكر كالتالي:
#idx("memoize", decl: true)
#snippet(```python
def memoize(f):
    table = make_table()
    def memoized(x):
        previously_computed_result = lookup(x, table)
        if is_none(previously_computed_result):
            result = f(x)
            insert(x, result, table)
            return result
        else:
            return previously_computed_result
    return memoized
```)

ارسم مخطط بيئة لتحليل حساب #py("memo_fib(3)").
واشرح لماذا تحسب #py("memo_fib") عدد فيبوناتشي الـ $n$ في عدد من الخطوات يتناسب مع $n$. وهل ستظل الخطة تعمل إذا عرّفنا ببساطة #py("memo_fib") لتكون #py("memoize(fib)")؟
])

#idx("table")
