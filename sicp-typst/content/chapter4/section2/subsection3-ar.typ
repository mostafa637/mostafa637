// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([التدفقات كقوائم كسولة], label-name: <sec:lazy-cons>)

#idx("stream(s)", sub: "implemented as lazy lists")
#idx("lazy list")
#idx("list(s)", sub: "lazy")
#idx("lazy pair")
#idx("pair(s)", sub: "lazy")

في القسم @sec:delayed-lists، أظهرنا كيفية تنفيذ التدفقات كقوائم مؤجلة.
#idx("delayed expression", sub: "lazy evaluation and")
واستخدمنا #idx("lambda expression", sub: "lazy evaluation and") تعبير #en[lambda] لإنشاء
#idx("promise to evaluate", sub: "lazy evaluation and")
«وعد» لحساب ذيل التدفق، دون الوفاء بهذا الوعد فعليًا إلا لاحقًا.

وكنا مضطرين لإنشاء التدفقات كنوع جديد من كائنات البيانات المشابهة للقوائم دون أن تكون مطابقة لها، وهذا يتطلب منا إعادة تنفيذ العديد من عمليات القوائم الاعتيادية (#py("map")، و#py("append")، وما إلى ذلك) لاستخدامها مع التدفقات.

ومع التقييم الكسول، يمكن أن تكون التدفقات والقوائم متطابقة، لذا لا داعي لعمليات منفصلة للقوائم والتدفقات. كل ما نحتاج إلى فعله هو ترتيب الأمور بحيث تكون #py("pair") غير صارمة. إحدى الطرق لتحقيق ذلك هي توسيع المُقيِّم الكسول للسماح بأوليات غير صارمة، وتنفيذ #py("pair") كإحدى هذه الأوليات. والطريقة الأسهل هي أن نتذكر (القسم @sec:data-) أنه لا يوجد احتياج أساسي لتنفيذ #py("pair") كدالة أولية على الإطلاق. وبدلاً من ذلك، يمكننا تمثيل
#idx("pair(s)", sub: "functional representation of")
الأزواج كدوال:#footnote[هذا هو التمثيل الدالي الموصوف في التمرين @ex:lambda-cons. وحقيقةً، سيفيد أي تمثيل دالي (مثل تنفيذ يمرر الرسائل). لاحظ أنه يمكننا تثبيت هذه التعاريف في المُقيِّم الكسول ببساطة عن طريق كتابتها في حلقة المحرك. وإذا كنا قد ضمنّا في الأصل #py("pair") و#py("head") و#py("tail") كأوليات في البيئة العالمية، فستُعاد إعادة تعريفها. (انظر أيضًا التمارين @ex:lazy-list-input و @ex:lazy-list-printing).]

#idx("pair (primitive function)", sub: "functional implementation of", decl: true)#idx("head (primitive function)", sub: "functional implementation of", decl: true)#idx("tail (primitive function)", sub: "functional implementation of", decl: true)
#snippet(```python
def pair(x, y):
    return lambda m: (m(x, y))
def head(z):
    return z(lambda p, q: (p))
def tail(z):
    return z(lambda p, q: (q))
```)

بدلالة هذه العمليات الأساسية، ستعمل التعاريف المعيارية لعمليات القوائم مع القوائم اللانهائية (التدفقات) وكذلك القوائم المتناهية، ويمكن تنفيذ عمليات التدفقات كعمليات قوائم. إليك بعض الأمثلة:

#idx("listref", decl: true)#idx("map", decl: true)#idx("scalelist", decl: true)#idx("addlists", decl: true)#idx("ones (infinite stream)", sub: "lazy-list version", decl: true)#idx("integers (infinite stream)", sub: "lazy-list version", decl: true)
#snippet(```python
def llist_ref(items, n):
    return head(items) if n == 0 else llist_ref(tail(items), n - 1)
def map(fun, items):
    return None if is_none(items) else pair(fun(head(items)), map(fun, tail(items)))
def scale_list(items, factor):
    return map(lambda x: (x * factor), items)
def add_lists(list1, list2):
    return list2 if is_none(list1) else list1 if is_none(list2) else pair(head(list1) + head(list2), add_lists(tail(list1), tail(list2)))
ones = pair(1, ones)
integers = pair(1, add_lists(ones, integers))
```)

#prompt(```python
L-evaluate input:
```)

#snippet(```python
llist_ref(integers, 17)
```)

#output(```python
llist_ref(integers, 17)
```)

لاحظ أن هذه القوائم الكسولة هي أكثر كسلًا حتى من تدفقات الفصل @chap:state: رأس القائمة، بالإضافة إلى ذيلها، مؤجَّل.#footnote[يتيح لنا هذا إنشاء نسخ مؤجلة من أنواع أكثر عمومية من بنيات القوائم، وليس فقط المتتاليات. يناقش #idx("Hughes, R. J. M.") #en[Hughes 1990] بعض تطبيقات
#idx("lazy tree")#idx("tree", sub: "lazy")
«الأشجار الكسولة».]
وفي الواقع، حتى الوصول إلى #py("head") أو #py("tail") لزوج كسول لا يتطلب فرض قيمة عنصر القائمة. فستُفْرَض القيمة فقط عندما تكون هناك حاجة إليها حقًا — مثل استغلالها كوسيط لدالة أولية، أو لطباعتها كإجابة.

تساعد الأزواج الكسولة أيضًا في المشكلة التي نشأت مع التدفقات في القسم @sec:streams-and-delayed-evaluation، حيث وجدنا أن صياغة نماذج التدفق للأنظمة ذات الحلقات قد تتطلب منا نثر تعبيرات #en[lambda] إضافية لـ #idx("delayed evaluation", sub: "explicit vs. automatic") #idx("delayed expression", sub: "explicit vs. automatic") التأجيل في برامجنا، بالإضافة إلى تلك المطلوبة لبناء زوج تدفق. ومع التقييم الكسول، يُؤَجَّل جميع الوسائط للدوال بشكل موحد. على سبيل المثال، يمكننا تنفيذ دوال لمكاملة القوائم وحل المعادلات التفاضلية كما قصدنا أصلاً في القسم @sec:streams-and-delayed-evaluation:

#idx("integral", sub: "lazy-list version", decl: true)#idx("solve differential equation", sub: "lazy-list version", decl: true)
#snippet(```python
def integral(integrand, initial_value, dt):
    int = pair(initial_value, add_lists(scale_list(integrand, dt), int))
    return int
def solve(f, y0, dt):
    y = integral(dy, y0, dt)
    dy = map(f, y)
    return y
```)

#prompt(```python
L-evaluate input:
```)

#snippet(```python
llist_ref(solve(lambda x: (x), 1, 0.001), 1000)
```)

#output(```python
llist_ref(solve(lambda x: (x), 1, 0.001), 1000)
```)

#exercise(label-name: <ex:lazier>, [
اعطِ بعض الأمثلة التي توضح الفارق بين تدفقات الفصل @chap:state والقوائم الكسولة «الأكثر كسلاً» الموصوفة في هذا القسم. كيف يمكنك الاستفادة من هذا الكسل الإضافي؟
])

#exercise(label-name: <ex:lazy-list-input>, [
يختبر #en[Ben Bitdiddle] تنفيذ القائمة الكسولة المعطاة أعلاه عن طريق تقييم التعبير

#snippet(```python
head(llist("a", "b", "c"))
```)

ولدهشته، ينتج عن هذا خطأ. وبعد بعض التفكير، يدرك أن «القوائم» المحصول عليها من دالة #py("list") الأولية مختلفة عن القوائم التي تتعامل معها التعاريف الجديدة لـ #py("pair") و#py("head") و#py("tail"). عدّل المُقيِّم بحيث تنتج تطبيقات دالة #py("list") الأولية المكتوبة في حلقة المحرك قوائم كسولة حقيقية.
])

#exercise(label-name: <ex:lazy-list-printing>, [
عدّل حلقة المحرك للمُقَيِّم بحيث تُطبَع الأزواج والقوائم الكسولة بطريقة معقولة. (ماذا ستفعل بشأن القوائم اللانهائية؟) قد تحتاج أيضًا إلى تعديل تمثيل الأزواج الكسولة بحيث يستطيع المُقيِّم التعرف عليها من أجل طباعتها.
])

#idx("delayed evaluation", sub: "in lazy evaluator")
#idx("stream(s)", sub: "implemented as lazy lists")
#idx("lazy list")
#idx("list(s)", sub: "lazy")
#idx("lazy pair")
#idx("pair(s)", sub: "lazy")
