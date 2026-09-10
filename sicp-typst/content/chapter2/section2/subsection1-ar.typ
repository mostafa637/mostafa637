// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([تمثيل المتتاليات], label-name: <sec:sequences>)

إحدى البنى المفيدة التي يمكننا بناؤها باستخدام الأزواج هي
#idx("sequence(s)")
#idx("sequence(s)", sub: "represented by pairs")
#idx("pair(s)", sub: "used to represent sequence")
#emph[المتتالية] (#en[sequence])—وهي مجموعة مرتبة من كائنات البيانات. وهناك بالطبع طرق عديدة لتمثيل المتتاليات باستخدام الأزواج. وتستعرض إحدى الطرق المباشرة للغاية في
الشكل @fig:sequence-of-pairs،
حيث تُمثل المتتالية 1، 2، 3، 4 كسلسلة من الأزواج. يكون
#py("head") (رأس)
كل زوج هو العنصر المقابل في السلسلة، ويكون
#py("tail") (ذيل)
الزوج هو الزوج التالي في السلسلة. ويدل
#py("tail")
الزوج الأخير على نهاية المتتالية،
والتي تُمثل في مخططات الصندوق والمؤشر بقطر مائل
#idx("box-and-pointer notation", sub: "end-of-linked-list marker")
وفي البرامج بقيمة
#idx("keywords", sub: "None")
#idx("None (keyword)", sub: "as end-of-linked-list marker")
#idx("end-of-linked-list marker")
#en[Python] الأولية #py("None").
وتُبنى المتتالية بأكملها عبر عمليات
#py("pair")
المتداخلة:

#snippet(```python
pair(1,
     pair(2,
          pair(3,
               pair(4, None))))
```)

#sicp-figure(image("/images/img_javascript/ch2-Z-G-13.svg", width: 70%), caption: [المتتالية 1، 2، 3، 4 ممثلة كسلسلة من الأزواج.], label-name: <fig:sequence-of-pairs>)

تُسمى مثل هذه السلسلة من الأزواج، المكونة من تطبيقات متداخلة لـ
#py("pair")،
#idx("linked list") #emph[قائمة مترابطة] (#en[linked list])،
وتوفر بيئة #en[Python] الخاصة بنا إجراءً أولياً يسمى
#idx("llist (primitive function)")

#py("llist")
للمساعدة في بناء القوائم
المترابطة.#footnote[في هذا الكتاب، نستخدم المصطلح
#emph[قائمة مترابطة]
لنعني سلسلة من الأزواج تنتهي بمؤشر نهاية القائمة المترابطة.
وعلى النقيض من ذلك، يشير المصطلح
#idx("linked-list structure", sub: "linked list vs.")
#idx("linked list(s)", sub: "linked-list structure vs.")
#emph[بنية القائمة المترابطة]
إلى أي بنية بيانات مكونة من أزواج، وليس فقط إلى
القوائم المترابطة.]
يمكن إنتاج المتتالية أعلاه بوساطة
#idx("llist (primitive function)")

#py("llist(1, 2, 3, 4)").
وبشكل عام، فإن

#syntax("
llist(", meta("a"), $""_(1)$, ", ", meta("a"), $""_(2)$, ", ", $dots.h$, ", ", meta("a"), $""_(n)$, ")
      ")

يكافئ

#syntax("
pair(", meta("a"), $""_(1)$, ", pair(", meta("a"), $""_(2)$, ", pair(", $dots.h$, ", pair(", meta("a"), $""_(n)$, ", None)", $dots.h$, ")))
      ")

يطبع المفسر الخاص بنا الأزواج باستخدام تمثيل نصي لمخططات الصندوق والمؤشر نسميه #emph[الترميز الصندوقي] (#en[box notation]).
#idx("linked list", sub: "printed representation of")
#idx("[ , ] (box notation for pairs)", sort: "0a21")
#idx("box notation for pairs")
#idx("pair(s)", sub: "box notation for")
#idx("notation in this book", sub: "box notation for data")
نتيجة #py("pair(1, 2)")
تُطبع كـ #py("[1, 2]")، وكائن البيانات في الشكل @fig:sequence-of-pairs
يُطبع كـ
#py("[1, [2, [3, [4, None]]]]"):

#snippet(```python
one_through_four = llist(1, 2, 3, 4)
```)

#snippet(```python
print(one_through_four)
```)

#output(```python
print(one_through_four)
```)

يمكننا التفكير في
#idx("linked list", sub: "manipulation with head, tail, and pair")
#idx("head (primitive function)", sub: "as linked-list operation")
#py("head")
على أنه يختار العنصر الأول في القائمة المترابطة،
وفي
#idx("tail (primitive function)", sub: "as linked-list operation")
#py("tail")
على أنه يختار مكون القائمة المترابطة المكون من جميع العناصر ما عدا العنصر الأول.
ويمكن استخدام التطبيقات المتداخلة لـ
#py("head")
و
#py("tail")
لاستخراج العناصر الثانية والثالثة واللاحقة في القائمة المترابطة.
ويقوم الإجراء الباني
#idx("pair (primitive function)", sub: "as linked-list operation")
#py("pair")
بإنشاء قائمة مترابطة مثل القائمة الأصلية، ولكن مع إضافة عنصر إضافي في
البداية.

#snippet(```python
print(head(one_through_four))
```)

#output(```python
print(head(one_through_four))
```)

#snippet(```python
print(tail(one_through_four))
```)

#output(```python
print(tail(one_through_four))
```)

#snippet(```python
print(head(tail(one_through_four)))
```)

#output(```python
print(head(tail(one_through_four)))
```)

#snippet(```python
print(pair(10, one_through_four))
```)

#output(```python
print(pair(10, one_through_four))
```)

#snippet(```python
print(pair(5, one_through_four))
```)

#output(```python
print(pair(5, one_through_four))
```)

يمكن اعتبار القيمة #py("None")، المستخدمة لإنهاء سلسلة الأزواج، كمتتالية لا تحتوي على عناصر، أي
#idx("empty linked list")
#idx("None (keyword)", sub: "as empty linked list")
#emph[القائمة المترابطة الفارغة].#footnote[تُستخدم القيمة
#py("None") في #en[Python] لأغراض متعدّدة، كما سنرى في الفصل @chap:state.]

يصعب أحياناً قراءة الترميز الصندوقي. وفي هذا الكتاب، عندما نريد الإشارة إلى طبيعة القائمة المترابطة لبنية بيانات ما، فإننا سنستخدم
#idx("notation in this book", sub: "linked-list notation for data")
#idx("linked-list notation for data")
#emph[ترميز القائمة المترابطة] البديل: وحيثما أمكن، يستخدم ترميز القائمة المترابطة تطبيقات لـ #py("llist") التي يؤدي تقييمها إلى البنية المطلوبة. فعلى سبيل المثال، بدلاً من الترميز الصندوقي

#output(```python
print(pair(5, one_through_four))
```)

نكتب

#output(```python
print(pair(5, one_through_four))
```)

في ترميز القائمة المترابطة.#footnote[توفر بيئة #en[Python] الخاصة بنا دالة أولية
#py("print_llist")
تعمل مثل الدالة الأولية
#py("print")، باستثناء أنها تستخدم ترميز القائمة المترابطة بدلاً من الترميز الصندوقي.]

#subheading([عمليات القوائم المترابطة])

#idx("linked list", sub: "operations on")
#idx("linked list", sub: "techniques for manipulating")

إن استخدام الأزواج لتمثيل متتاليات العناصر كقوائم مترابطة يترافق مع تقنيات برمجة تقليدية للتلاعب بالقوائم المترابطة عن طريق
#idx("walking down a linked list with tail") #idx("linked list", sub: "walking down with tail") استخدام #py("tail") متتالياً للمرور على عناصر القائمة المترابطة.
فعلى سبيل المثال، الدالة
#idx("linked list", sub: "nth element of")
#py("llist_ref")
تأخذ كمعاملات قائمة مترابطة ورقماً $n$
وترجع العنصر $n$ في القائمة المترابطة.
ومن المعتاد ترقيم عناصر القائمة المترابطة بدءاً من 0. وطريقة حساب
#py("llist_ref")
هي كما يلي:

- بالنسبة لـ $n=0$، يجب أن ترجع #py("llist_ref") #py("head") القائمة المترابطة.
- خلاف ذلك، يجب أن ترجع #py("llist_ref") العنصر $(n-1)$ من #py("tail") القائمة المترابطة.

#idx("llistref", decl: true)
#snippet(```python
def llist_ref(items, n):
    return (head(items) if n == 0
            else llist_ref(tail(items), n - 1))
```)

#snippet(```python
squares = llist(1, 4, 9, 16, 25)

print(llist_ref(squares, 3))
```)

#output(```python
squares = llist(1, 4, 9, 16, 25)

print(llist_ref(squares, 3))
```)

غالباً ما نمر على القائمة المترابطة بأكملها. وللمساعدة في ذلك، تتضمن بيئة #en[Python] الخاصة بنا محمولاً أولياً
#idx("isnone (primitive function)")

#idx("empty linked list", sub: "recognizing with isnone")
#idx("None (keyword)", sub: "recognizing with isnone")
#py("is_none")،
والذي يفحص ما إذا كان معامله هو القائمة المترابطة الفارغة.
وتوضح الدالة
#idx("length")
#idx("linked list", sub: "length of")
#py("length")، التي ترجع عدد العناصر في القائمة المترابطة، هذا النمط النموذجي للاستخدام:
#idx("length", sub: "recursive version", decl: true)
#snippet(```python
def length(items):
    return (0 if is_none(items)
            else 1 + length(tail(items)))
```)

#snippet(```python
odds = llist(1, 3, 5, 7)

print(length(odds))
```)

#output(```python
odds = llist(1, 3, 5, 7)

print(length(odds))
```)

تنفذ الدالة #py("length") خطة تكرارية بسيطة (تعاودية). وخطوة التخفيض هي:

- طول (#py("length")) أي قائمة مترابطة هو 1 زائد طول (#py("length")) ذيل (#py("tail")) القائمة المترابطة.

ويتم تطبيق ذلك متتالياً حتى نصل إلى الحالة الأساسية:

- طول (#py("length")) القائمة المترابطة الفارغة هو 0.

يمكننا أيضاً حساب #py("length") بأسلوب تكراري:
#idx("length", sub: "iterative version", decl: true)
#snippet(```python
def length(items):
    def length_iter(a, count):
        return (count if is_none(a)
                else length_iter(tail(a), count + 1))
    return length_iter(items, 0)
```)

هناك تقنية برمجة تقليدية أخرى وهي
#idx("constructing a linked list with pair") #idx("linked list", sub: "constructing with pair") #idx("adjoining to a linked list with pair") #idx("linked list", sub: "adjoining to with pair") بناء قائمة مترابطة للنتيجة عن طريق إلحاق العناصر بمقدمة القائمة المترابطة باستخدام #py("pair") أثناء المرور عبر القائمة المترابطة باستخدام #py("tail")،
كما في الدالة
#idx("linked list", sub: "combining with append")
#py("append")، التي تأخذ قائمتين مترابطتين كمعاملات وتدمج عناصرهما لإنشاء قائمة مترابطة جديدة:

#snippet(```python
print_llist(append(squares, odds))
```)

#output(```python
print_llist(append(squares, odds))
```)

#snippet(```python
print_llist(append(odds, squares))
```)

#output(```python
print_llist(append(odds, squares))
```)

يتم تنفيذ الدالة #py("append") أيضاً باستخدام خطة تعاودية.
لإلحاق القائمتين المترابطتين #py("list1") و #py("list2")، افعل ما يلي:

- إذا كانت #py("list1") هي القائمة المترابطة الفارغة، فإن النتيجة هي فقط #py("list2").
- خلاف ذلك، ألحق #py("tail") القائمة #py("list1") بـ #py("list2")، وألحق #py("head") القائمة #py("list1") بالنتيجة:

#idx("append", decl: true)
#snippet(```python
def append(list1, list2):
    return (list2 if is_none(list1)
            else pair(head(list1), append(tail(list1), list2)))
```)

#exercise(label-name: <ex:last>, [
عرف دالة
#idx("lastpair")
#idx("linked list", sub: "last pair of")
#py("last_pair")
ترجع القائمة المترابطة التي تحتوي فقط على العنصر الأخير من قائمة مترابطة معطاة (غير فارغة):

#snippet(```python
print_llist(last_pair(llist(23, 72, 149, 34)))
```)

#output(```python
print_llist(last_pair(llist(23, 72, 149, 34)))
```)
])

#exercise(label-name: <ex:reverse>, [
عرف دالة
#idx("reverse")
#idx("linked list", sub: "reversing")
#py("reverse")
تأخذ قائمة مترابطة كمعامل وترجع قائمة مترابطة تحتوي على العناصر نفسها ولكن بترتيب معكوس:

#snippet(```python
print_llist(reverse(llist(1, 4, 9, 16, 25)))
```)

#output(```python
print_llist(reverse(llist(1, 4, 9, 16, 25)))
```)
])

#exercise(label-name: <ex:2_19>, [
تأمل برنامج عد الفئات النقدية (#idx("counting change")) في القسم @sec:tree-recursion. سيكون من الجيد التمكن من تغيير العملة المستخدمة في البرنامج بسهولة، لكي نتمكن من حساب عدد طرق فك الجنيه الإسترليني، على سبيل المثال. وكما كُتب البرنامج، فإن المعرفة بالعملة تتوزع جزئياً في الدالة
#py("first_denomination")
وجزئياً في الدالة
#py("count_change")
(التي تعرف أن هناك خمسة أنواع من قطع النقد الأمريكية).
وسيكون من الأفضل التمكن من تقديم قائمة مترابطة من القطع النقدية ليتم استخدامها لفك المبلغ.

نريد إعادة كتابة الدالة #py("cc") بحيث يكون معاملها الثاني قائمة مترابطة من قيم القطع النقدية بدلاً من عدد صحيح يحدد القطع المستخدمة. يمكن أن يكون لدينا حينئذ قوائم مترابطة تعرّف كل نوع من العملات:

#snippet(```python
us_coins = llist(50, 25, 10, 5, 1)
uk_coins = llist(100, 50, 20, 10, 5, 2, 1)
```)

يمكننا بعد ذلك استدعاء #py("cc") كما يلي:

#snippet(```python
print(cc(100, us_coins))
```)

#output(```python
print(cc(100, us_coins))
```)

للقيام بذلك، سيتطلب الأمر تغيير برنامج #py("cc") نوعاً ما. وسيكون له الشكل نفسه، لكنه سيتعامل مع معامله الثاني بشكل مختلف، على النحو التالي:

#snippet(```python
def cc(amount, coin_values):
    return (1 if amount == 0
            else 0 if amount < 0 or no_more(coin_values)
            else cc(amount, except_first_denomination(coin_values)) +
                 cc(amount - first_denomination(coin_values), coin_values))
```)

عرف الدوال #py("first_denomination")، و #py("except_first_denomination")، و #py("no_more") بدلالة العمليات الأولية على بنى القوائم المترابطة. هل يؤثر ترتيب القائمة المترابطة #py("coin_values") على النتيجة التي ينتجها #py("cc")؟ ولماذا أو لمَ لا؟
])

#exercise(label-name: <ex:2_20>, [
في وجود الدوال العليا، ليس من الضروري تماماً للدوال أن تمتلك معاملات متعددة؛ إذ يمكن لمعامل واحد أن يكفي. وإذا كان لدينا دالة مثل #py("plus") تتطلب طبيعياً معاملين، يمكننا كتابة تنويعة للدالة نمرر إليها المعاملات واحداً تلو الآخر. ويمكن أن يؤدي تطبيق التنويعة على المعامل الأول إلى إرجاع دالة يمكننا تطبيقها بعد ذلك على المعامل الثاني، وهكذا. وهذه الممارسة—المسماة
#idx("currying")
#emph[الكَرِينة] (#en[currying]) والمسمّاة باسم عالم الرياضيات والمنطق الأمريكي
#idx("Curry, Haskell Brooks")
#en[Haskell Brooks Curry]—شائعة جداً في لغات البرمجة مثل
#idx("Haskell")
#en[Haskell] و
#idx("Ocaml")
#en[OCaml]. وفي #en[Python]، تبدو النسخة المكرّنة من #py("plus") كما يلي.

#snippet(```python
def plus_curried(x):
    return lambda y: x + y
```)

اكتب دالة #py("brooks") تأخذ دالة مكرّنة كمعامل أول وقائمة مترابطة من المعاملات كمعامل ثانٍ، حيث تُطبّق الدالة المكرّنة عليها واحداً تلو الآخر بالترتيب المعطى. فعلى سبيل المثال، ينبغي أن يكون لاستدعاء #py("brooks") التالي التأثير نفسه لـ #py("plus_curried(3)(4)"):

#snippet(```python
print(brooks(plus_curried, llist(3, 4)))
```)

#output(```python
print(brooks(plus_curried, llist(3, 4)))
```)

وبما أننا في صدد ذلك، فلنكرّن الدالة #py("brooks") نفسها! اكتب دالة #py("brooks_curried") يمكن تطبيقها كما يلي:

#snippet(```python
print(brooks_curried(llist(plus_curried, 3, 4)))
```)

#output(```python
print(brooks_curried(llist(plus_curried, 3, 4)))
```)

مع هذه الدالة #py("brooks_curried")، ما هي نتائج تقييم العبارتين التاليتين؟

#snippet(```python
brooks_curried(llist(brooks_curried,
                     llist(plus_curried, 3, 4)))
```)

#snippet(```python
brooks_curried(llist(brooks_curried,
                     llist(brooks_curried,
                           llist(plus_curried, 3, 4))))
```)
])

#idx("linked list", sub: "operations on")
#idx("linked list", sub: "techniques for manipulating")

#subheading([التطبيق الخرائطي على القوائم المترابطة])

#idx("linked list", sub: "mapping over")
#idx("mapping", sub: "over linked lists")

إحدى العمليات المفيدة للغاية هي تطبيق تحويل ما على كل عنصر في قائمة مترابطة وإنشاء قائمة مترابطة من النتائج. فعلى سبيل المثال، تقوم الدالة التالية بتغيير مقياس كل رقم في قائمة مترابطة بمعامل معين:
#idx("scalelinkedlist", decl: true)
#snippet(```python
def scale_linked_list(items, factor):
    return (None if is_none(items)
            else pair(head(items) * factor,
                      scale_linked_list(tail(items), factor)))
```)

#snippet(```python
print(scale_linked_list(llist(1, 2, 3, 4, 5), 10))
```)

#output(```python
print(scale_linked_list(llist(1, 2, 3, 4, 5), 10))
```)

يمكننا تجريد هذه الفكرة العامة وصياغتها كنمط مشترك مُعبر عنه كدالة عليا، تماماً كما في القسم @sec:higher-order-procedures. وتسمى الدالة العليا هنا #py("map").
تأخذ الدالة #py("map") كمعاملات دالةً ذات معامل واحد وقائمةً مترابطة، وترجع قائمة مترابطة من النتائج الناتجة عن تطبيق الدالة على كل عنصر في القائمة المترابطة:
#idx("map", decl: true)
#snippet(```python
def map(fun, items):
    return (None if is_none(items)
            else pair(fun(head(items)),
                      map(fun, tail(items))))
```)

#snippet(```python
print(map(abs, llist(-10, 2.5, -11.6, 17)))
```)

#output(```python
print(map(abs, llist(-10, 2.5, -11.6, 17)))
```)

#snippet(```python
print(map(lambda x: x * x, llist(1, 2, 3, 4)))
```)

#output(```python
print(map(lambda x: x * x, llist(1, 2, 3, 4)))
```)

الان يمكننا إعطاء تعريف جديد لـ #py("scale_linked_list") بدلالة #py("map"):
#idx("scalelinkedlist", decl: true)
#snippet(```python
def scale_linked_list(items, factor):
    return map(lambda x: x * factor, items)
```)

تعتبر الدالة #py("map") بنية هامة، ليس فقط لأنها تجسد نمطاً شائعاً، بل لأنها تؤسس مستوى أعلى من التجريد في التعامل مع القوائم المترابطة.
في التعريف الأصلي لـ #py("scale_linked_list")، يلفت البناء التعاودي للبرنامج الانتباه إلى معالجة عناصر القائمة المترابطة عنصراً عنصر.
بينما تعريف #py("scale_linked_list") بدلالة #py("map") يحجب ذلك المستوى من التفاصيل ويؤكد على أن تغيير المقياس يحول قائمة مترابطة من العناصر إلى قائمة مترابطة من النتائج. الفرق بين التعريفين ليس أن الحاسوب يقوم بعملية مختلفة (فهو لا يفعل)، بل أننا نفكر في العملية بشكل مختلف. وفي الواقع، تساعد #py("map") في إنشاء حاجز تجريد يعزل تنفيذ الدوال التي تحول القوائم المترابطة عن تفاصيل كيفية استخراج وتجميع عناصر القائمة المترابطة. ومثل الحواجز المعروضة في الشكل @fig:abstraction-barriers، يمنحنا هذا التجريد المرونة لتغيير التفاصيل منخفضة المستوى لكيفية تنفيذ المتتاليات، مع الحفاظ على الإطار المفاهيمي للعمليات التي تحول المتتاليات إلى متتاليات.
ويتوسع القسم @sec:sequences-conventional-interfaces في هذا الاستخدام للمتتاليات كإطار لتنظيم البرامج.

#exercise(label-name: <ex:square-list>, [
تأخذ الدالة #py("square_linked_list") قائمة مترابطة من الأعداد كمعامل وترجع قائمة مترابطة من مربعات تلك الأعداد.

#snippet(```python
print(square_linked_list(llist(1, 2, 3, 4)))
```)

#output(```python
print(square_linked_list(llist(1, 2, 3, 4)))
```)

إليك تعريفين مختلفين لـ #py("square_linked_list"). أكملهما بملء التعبيرات المفقودة:

#syntax("
def square_linked_list(items):
    return (None if is_none(items)
            else pair(", metaphrase[??], ", ", metaphrase[??], "))
      ")

#syntax("
def square_linked_list(items):
    return map(", metaphrase[??], ", ", metaphrase[??], ")
      ")
])

#exercise(label-name: <ex:iter-square-list>, [
يحاول #en[Louis Reasoner] إعادة كتابة الدالة الأولى لـ #py("square_linked_list") من التمرين @ex:square-list بحيث تنشئ عملية تكرارية:

#snippet(```python
def square_linked_list(items):
    def iter(things, answer):
        return (answer if is_none(things)
                else iter(tail(things),
                          pair(square(head(things)),
                               answer)))
    return iter(items, None)
```)

لسوء الحظ، فإن تعريف #py("square_linked_list") بهذه الطريقة ينتج قائمة مترابطة للنتيجة بترتيب معكوس للترتيب المطلوب.
لماذا؟

ثم يحاول #en[Louis] إصلاح خلله عن طريق المبادلة بين معاملي #py("pair"):

#snippet(```python
def square_linked_list(items):
    def iter(things, answer):
        return (answer if is_none(things)
                else iter(tail(things),
                          pair(answer,
                               square(head(things)))))
    return iter(items, None)
```)

هذا لا يعمل أيضاً. اشرح ذلك.
])

#exercise(label-name: <ex:for-each>, [
الدالة
#idx("foreach")
#py("for_each")
مشابهة لـ
#py("map").
فهي تأخذ كمعاملات دالة وقائمة مترابطة من العناصر. ومع ذلك، بدلاً من تشكيل قائمة مترابطة من النتائج، تقوم #py("for_each") فقط بتطبيق الدالة على كل عنصر بدورها، من اليسار إلى اليمين. والقيم المعادة عن طريق تطبيق الدالة على العناصر لا تُستخدم على الإطلاق—تُستخدم #py("for_each") مع الدوال التي تؤدي إجراءً، مثل الطباعة. فعلى سبيل المثال،

#snippet(```python
print(for_each(lambda x: print(x), llist(57, 321, 88)))
```)

#output(```python
print(for_each(lambda x: print(x), llist(57, 321, 88)))
```)

يمكن أن تكون القيمة المعادة من استدعاء #py("for_each") (غير الموضحة أعلاه) أي شيء اعتباطي، مثل #py("True"). قدم تنفيذاً لـ #py("for_each").
])

#idx("linked list", sub: "mapping over")
#idx("mapping", sub: "over linked lists")
