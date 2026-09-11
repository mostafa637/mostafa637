// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([تمثيل الطوابير], label-name: <sec:queues>)

#idx("queue")

تتيح لنا دالتا التعديل #py("set_head") و #py("set_tail") من استخدام الأزواج لبناء بنيات بيانات لا يمكن بناؤها باستخدام #py("pair") و #py("head") و #py("tail") وحدها. ويوضح هذا القسم كيفية استخدام الأزواج لتمثيل بنية بيانات تُسمى الطابور (#en[queue] / صف الانتظار). وسوف يوضح القسم @sec:tables كيفية تمثيل بنيات بيانات تُسمى الجداول.

الطابور (#en[queue]) هو تسلسل تُدرج فيه العناصر عند أحد الأطراف (يُسمى
#idx("queue", sub: "rear of")
#emph[خلفية] الطابور) وتُحذف من الطرف الآخر (
#idx("queue", sub: "front of")
#emph[مقدمة] الطابور).
ويُظهر الشكل @fig:queue-ops طابوراً خالياً في البداية تُدرج فيه العناصر #py("a") و #py("b"). ثم يُزال #py("a")، وتُدرج #py("c") و #py("d")، ويُزال #py("b"). وبما أن العناصر تُزال دائماً بالترتيب الذي أُدرجت به، يُسمى الطابور أحياناً بـ
#idx("FIFO buffer")
ذاكرة مؤقتة من نوع #emph[FIFO] (الأول دخولاً، الأول خروجاً - #en[first in, first out]).

#sicp-figure([#sicp-table(columns: 2, [العملية], [الطابور الناتج], [#py("q = make_queue()")], [], [#py("insert_queue(q, \"a\")")], [#py("a")], [#py("insert_queue(q, \"b\")")], [#py("a b")], [#py("delete_queue(q)")], [#py("b")], [#py("insert_queue(q, \"c\")")], [#py("b c")], [#py("insert_queue(q, \"d\")")], [#py("b c d")], [#py("delete_queue(q)")], [#py("c d")])], caption: [عمليات الطابور.], label-name: <fig:queue-ops>)

ومن حيث
#idx("data abstraction", sub: "for queue")
#idx("queue", sub: "operations on")
تجريد البيانات، يمكننا اعتبار الطابور معرَّفاً بمجموعة العمليات التالية:

- منشئ: \ #idx("makequeue") #py("make_queue()") \ يُرجع طابوراً خالياً (طابوراً لا يحتوي على عناصر).
- محمول: \ #idx("isemptyqueue") #py("is_empty_queue(")#meta("queue")#py(")") \ يختبر ما إذا كان الطابور خالياً.
- محدد: \ #idx("frontqueue") #py("front_queue(")#meta("queue")#py(")") \ يُرجع الكائن الموجود في مقدمة الطابور، ويُصدر خطأ إذا كان الطابور خاليًا؛ وهو لا يعدل الطابور.
- دالتا تعديل: \ #py("insert_queue(")#meta("queue")#py(",")#meta("item")#py(")") \ يُدرج #idx("insertqueue") العنصر عند خلفية الطابور ويُرجع الطابور المعدل كقيمته. #py("delete_queue(")#meta("queue")#py(")") \ يُزيل #idx("deletequeue") العنصر من مقدمة الطابور ويُرجع الطابور المعدل كقيمته، ويُصدر خطأ إذا كان الطابور خاليًا قبل الحذف.

وبما أن الطابور عبارة عن تسلسل من العناصر، فقد يمثل بالتأكيد كقائمة عادية؛ فتكون مقدمة الطابور هي #py("head") القائمة، وإدراج عنصر في الطابور يكافئ إلحاق عنصر جديد عند نهاية القائمة، وحذف عنصر من الطابور يكون مجرد أخذ #py("tail") القائمة. ومع ذلك، فإن هذا التمثيل غير كفء، لأننا لإدراج عنصر يجب أن نفحص القائمة حتى نصل إلى النهاية. وبما أن الطريقة الوحيدة لدينا لفحص القائمة هي عمليات #py("tail") المتتالية، فإن هذا الفحص يتطلب $Theta(n)$ من الخطوات لقائمة تحتوي على $n$ من العناصر. وتتغلب تعديل بسيط على تمثيل القائمة على هذا العيب عن طريق السماح لتنفيذ عمليات الطابور بحيث تتطلب $Theta(1)$ من الخطوات؛ أي بحيث يكون عدد الخطوات المطلوبة مستقلاً عن طول الطابور.

تنشأ الصعوبة في تمثيل القائمة من الحاجة إلى الفحص للعثور على نهاية القائمة. والسبب في حاجتنا للفحص هو أنه على الرغم من أن الطريقة القياسية لتمثيل القائمة كسلسلة من الأزواج توفر لنا بسهولة مؤشراً إلى بداية القائمة، إلا أنها لا تعطينا مؤشراً يمكن الوصول إليه بسهولة إلى النهاية. والتعديل الذي يتجنب هذا العيب هو تمثيل الطابور كقائمة، إلى جانب مؤشر إضافي يشير إلى الزوج الأخير في القائمة. وبهذه الطريقة، عندما نذهب لإدراج عنصر، يمكننا فحص المؤشر الخلفي وبالتالي تجنب فحص القائمة.

وبالتالي، يُمثل الطابور كزوج من المؤشرات، #py("front_ptr") و #py("rear_ptr")، اللذين يشيران على التوالي إلى الزوجين الأول والأخير في قائمة عادية. وبما أننا نود أن يكون الطابور كائناً محدداً، يمكننا استخدام #py("pair") لجمع المؤشرين الاثنين. وهكذا، سيكون الطابور نفسه هو #py("pair") للمؤشرين الاثنين.
ويوضح الشكل @fig:queue-pointers هذا التمثيل.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-19.svg", width: 70%), caption: [تنفيذ الطابور كقائمة مع مؤشري المقدمة والخلفية.], label-name: <fig:queue-pointers>)

ولتعريف عمليات الطابور نستخدم الدوال التالية، والتي تمكننا من اختيار وتعديل مؤشري المقدمة والخلفية للطابور:

#idx("frontptr", decl: true)#idx("rearptr", decl: true)#idx("setfrontptr", decl: true)#idx("setrearptr", decl: true)
#snippet(```python
def front_ptr(queue):
    return head(queue)

def rear_ptr(queue):
    return tail(queue)

def set_front_ptr(queue, item):
    set_head(queue, item)

def set_rear_ptr(queue, item):
    set_tail(queue, item)
```)

والآن يمكننا تنفيذ عمليات الطابور الفعلية. وسوف نعتبر الطابور خالياً إذا كان مؤشر مقدمته هو القائمة الخالية:

#idx("isemptyqueue", decl: true)
#snippet(```python
def is_empty_queue(queue):
    return is_none(front_ptr(queue))
```)

يُرجع المنشئ #py("make_queue")، كطابور خالي في البداية، زوجاً كلاً من #py("head") و #py("tail") الخاصين به هما القائمة الخالية:

#idx("makequeue", decl: true)
#snippet(```python
def make_queue():
    return pair(None, None)
```)

ولاختيار العنصر عند مقدمة الطابور، نُرجع #py("head") الزوج المشار إليه بوساطة مؤشر المقدمة:

#idx("frontqueue", decl: true)
#snippet(```python
def front_queue(queue):
    return (error("front_queue called with an empty queue", queue)
            if is_empty_queue(queue)
            else head(front_ptr(queue)))
```)

ولإدراج عنصر في طابور، نتبع الطريقة التي توضح نتيجتها في الشكل @fig:queue-insert.
ننشئ أولاً زوجاً جديداً يكون #py("head") الخاص به هو العنصر المراد إدراجه ويكون #py("tail") الخاص به هو القائمة الخالية. وإذا كان الطابور خالياً في البداية، نحدد مؤشري المقدمة والخلفية للطابور إلى هذا الزوج الجديد. بخلاف ذلك، نعدل الزوج الأخير في الطابور ليثير إلى الزوج الجديد، ونحدد أيضاً مؤشر الخلفية إلى الزوج الجديد.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-20.svg", width: 70%), caption: [نتيجة استخدام #py("insert_queue(q, \"d\")") على الطابور في الشكل @fig:queue-pointers.], label-name: <fig:queue-insert>)

#idx("insertqueue", decl: true)
#snippet(```python
def insert_queue(queue, item):
    new_pair = pair(item, None)
    if is_empty_queue(queue):
        set_front_ptr(queue, new_pair)
        set_rear_ptr(queue, new_pair)
    else:
        set_tail(rear_ptr(queue), new_pair)
        set_rear_ptr(queue, new_pair)
    return queue
```)

ولحذف العنصر عند مقدمة الطابور، نعدل مجرد مؤشر المقدمة بحيث يشير الآن إلى العنصر الثاني في الطابور، والذي يمكن العثور عليه باتباع مؤشر #py("tail") للعنصر الأول (انظر الشكل @fig:queue-delete):#footnote[إذا كان العنصر الأول هو العنصر الأخير في الطابور، فسيكون مؤشر المقدمة هو القائمة الخالية بعد الحذف، مما يحدد الطابور كخالي؛ ولا داعي للقلق بشأن تحديث مؤشر الخلفية، والذي سيظل يشير إلى العنصر المحذوف، لأن #py("is_empty_queue") تنظر فقط إلى مؤشر المقدمة.]

#sicp-figure(image("/images/img_javascript/ch3-Z-G-21.svg", width: 70%), caption: [نتيجة استخدام #py("delete_queue(q)") على الطابور في الشكل @fig:queue-insert.], label-name: <fig:queue-delete>)

#idx("deletequeue", decl: true)
#snippet(```python
def delete_queue(queue):
    if is_empty_queue(queue):
        error("delete_queue called with an empty queue", queue)
    else:
        set_front_ptr(queue, tail(front_ptr(queue)))
        return queue
```)

#exercise(label-name: <ex:3_21>, [
يقرر بن بتديدل اختبار تنفيذ الطابور الموصوف أعلاه. فيكتب الدوال في مفسر بايثون ويشرع في تجربتها:

#snippet(```python
q1 = make_queue()
```)

#snippet(```python
print(insert_queue(q1, "a"))
```)

#output(```python
print(insert_queue(q1, "a"))
```)

#snippet(```python
print(insert_queue(q1, "b"))
```)

#output(```python
print(insert_queue(q1, "b"))
```)

#snippet(```python
print(delete_queue(q1))
```)

#output(```python
print(delete_queue(q1))
```)

#snippet(```python
print(delete_queue(q1))
```)

#output(```python
print(delete_queue(q1))
```)

يشكو بن قائلاً: "الأمر كله خاطئ! تظهر استجابة المفسر أن العنصر الأخير أُدرج في الطابور مرتين. وعندما أحذف كلا العنصرين، فإن #py("b") الثاني لا يزال هناك، فالطابور ليس خالياً، على الرغم من أنه يفترض أن يكون كذلك." وتفترض إيفا لو آتور أن بن أسيء فهم ما يحدث. فتشرح قائلة: "ليست المسألة أن العناصر تدخل في الطابور مرتين. بل فقط أن طابعة بايثون القياسية لا تعرف كيفية فهم تمثيل الطابور. وإذا كنت تريد رؤية الطابور مطبوعاً بشكل صحيح، فستتعين عليك تعريف دالة الطباعة الخاصة بك للأسطور." اشرح ما تتحدث عنه إيفا لو. وبشكل خاص، أظهر لماذا تنتج أمثلة بن النتائج المطبوعة التي أنتجتها.
ودالة
#idx("printqueue")
#py("print_queue")
تأخذ طابوراً كدخل وتطبع تسلسل العناصر في الطابور.
])

#exercise(label-name: <ex:3_22>, [
بدلاً من تمثيل الطابور كزوج من المؤشرات، يمكننا بناء الطابور كدالة
#idx("queue", sub: "functional implementation of")
ذات حالة محلية. وستتكون الحالة المحلية من مؤشرات إلى بداية ونهاية قائمة عادية. وبالتالي، فإن دالة #py("make_queue") ستأخذ الشكل

#syntax("
def make_queue():
    front_ptr = ", $dots.h$, "
    rear_ptr = ", $dots.h$, "
    ", metaphrase[تصريحات الدوال الداخلية], "
    def dispatch(m): ", $dots.h$, "
    return dispatch
      ")

أكمل تعريف #py("make_queue") وقدم تنفيذات لعمليات الطابور باستخدام هذا التمثيل.
])

#exercise(label-name: <ex:deque>, [
الطابور المزدوج (#emph[deque] -
#idx("queue", sub: "double-ended")
#idx("deque")
"الطابور ذو النهايتين") هو تسلسل يمكن إدراج وحذف العناصر فيه إما عند المقدمة أو عند الخلفية.
والعمليات على السطور المزدوجة هي المنشئ #py("make_deque")، والمحمول #py("is_empty_deque")، والمحددات #py("front_deque") و #py("rear_deque")، ودوال التعديل #py("front_insert_deque") و #py("front_delete_deque") و #py("rear_insert_deque") و #py("rear_delete_deque").
أظهر كيفية تمثيل السطور المزدوجة باستخدام الأزواج، وقدم تنفيذات للعمليات.#footnote[احرص على عدم جعل المفسر يحاول طباعة بنية تحتوي على دورات. (انظر التمرين @ex:make-cycle).]
ويجب أن تتطلب جميع العمليات $Theta(1)$ من الخطوات.
])

#idx("queue")
