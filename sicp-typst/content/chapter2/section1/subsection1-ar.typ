// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([مثال: العمليات الحسابية للأعداد الناطقة], label-name: <sec:rationals>)

#idx("arithmetic", sub: "on rational numbers")
#idx("rational number(s)", sub: "arithmetic operations on")
#idx("rational-number arithmetic")

افترض أننا نود إجراء عمليات حسابية مع الأعداد الناطقة (الكسرية). ونريد التمكن من جمعها وطرحها وضربها وقسمتها واختبار ما إذا كان عددان ناطقان متساويين.

لنبدأ بفرض أننا نمتلك بالفعل طريقة لبناء عدد ناطق من بسط ومقام. ونفترض أيضاً أنه مع إعطاء عدد ناطق، لدينا طريقة لاستخراج (أو تحديد) بسطه ومقامه. ولنفترض كذلك أنّ المُنشِئ والمحدّدات متاحة كدوال:

- #py("make_rat(")$n$#py(",")$d$#py(")") #idx("makerat") ترجع العدد الناطق الذي بسطه العدد الصحيح $n$ ومقامه العدد الصحيح $d$.
- #py("numer(")$x$#py(")") #idx("numer") ترجع بسط العدد الناطق $x$.
- #py("denom(")$x$#py(")") #idx("denom") ترجع مقام العدد الناطق $x$.

نحن نستخدم هنا استراتيجية تركيب قوية:
#idx("wishful thinking")
#emph[التفكير بالتمني (#en[wishful thinking])]. فلم نذكر بعد كيف يُثمَّل العدد الناطق، أو كيف يجب تنفيذ الدوال #py("numer") و #py("denom") و #py("make_rat"). ومع ذلك، لو كنا نملك هذه الدوال الثلاث، فبإمكاننا عندئذ الجمع والطرح والضرب والقسمة واختبار المساواة باستخدام العلاقات التالية:

$ mat(delim: #none, frac(n_(1), d_(1))+frac(n_(2), d_(2)), =, frac(n_(1)d_(2)+n_(2)d_(1), d_(1)d_(2)); frac(n_(1), d_(1))-frac(n_(2), d_(2)), =, frac(n_(1)d_(2)-n_(2)d_(1), d_(1)d_(2)); frac(n_(1), d_(1)) dot.op frac(n_(2), d_(2)), =, frac(n_(1)n_(2), d_(1)d_(2)); frac(n_(1)/d_(1), n_(2)/d_(2)), =, frac(n_(1)d_(2), d_(1)n_(2)); frac(n_(1), d_(1)), =, frac(n_(2), d_(2)) space upright("إذا وفقط إذا كان") space space space n_(1)d_(2) space = space n_(2)d_(1)) $

يمكننا التعبير عن هذه القواعد كدوال:
#idx("addrat", decl: true)#idx("subrat", decl: true)#idx("mulrat", decl: true)#idx("divrat", decl: true)#idx("equalrat", decl: true)
#snippet(```python
def add_rat(x, y):
    return make_rat(numer(x) * denom(y) + numer(y) * denom(x),
                    denom(x) * denom(y))
def sub_rat(x, y):
    return make_rat(numer(x) * denom(y) - numer(y) * denom(x),
                    denom(x) * denom(y))
def mul_rat(x, y):
    return make_rat(numer(x) * numer(y),
                    denom(x) * denom(y))
def div_rat(x, y):
    return make_rat(numer(x) * denom(y),
                    denom(x) * numer(y))
def equal_rat(x, y):
    return numer(x) * denom(y) == numer(y) * denom(x)
```)

الآن لدينا العمليات على الأعداد الناطقة مُحدَّدة بدلالة دوال المحدّدات والمُنشِئ #py("numer") و #py("denom") و #py("make_rat"). لكننا لم نحدد هذه الدوال بعد. وما نحتاجه هو طريقة ما للصق البسط والمقام معاً لتشكيل عدد ناطق.

#subheading([الأزواج])

لتمكيننا من تنفيذ المستوى الملموس لتجريد البيانات لدينا، توفر بيئة بايثون بنية مركبة تُسمّى
#idx("pair(s)")
#emph[زوجاً (#en[pair])]\، والتي يمكن بناؤها باستخدام الدالة الأولية

#idx("pair (primitive function)")
#py("pair").
تأخذ هذه الدالة معطيين وترجع كائن بيانات مركب يحتوي على المعطيين كأجزاء. ومع إعطاء زوج، يمكننا استخراج الأجزاء باستخدام الدالتين الأوليتين
#idx("head (primitive function)")
#py("head")
و
#idx("tail (primitive function)")
#py("tail").
وهكذا، يمكننا استخدام #py("pair") و #py("head") و #py("tail") كما يلي:

#snippet(```python
x = pair(1, 2)
```)

#snippet(```python
print(head(x))
```)

#output(```python
print(head(x))
```)

#snippet(```python
print(tail(x))
```)

#output(```python
print(tail(x))
```)

لاحظ أنّ الزوج كائن بيانات يمكن إعطاؤه اسماً والتعامل معه، تماماً مثل كائن البيانات الأولي. وعلاوة على ذلك، يمكن استخدام #py("pair") لتشكيل أزواج تكون عناصرها أزواجاً، وهكذا:

#snippet(```python
x = pair(1, 2)

y = pair(3, 4)

z = pair(x, y)
```)

#snippet(```python
print(head(head(z)))
```)

#output(```python
print(head(head(z)))
```)

#snippet(```python
print(head(tail(z)))
```)

#output(```python
print(head(tail(z)))
```)

في القسم @sec:hierarchical-data سنرى كيف تعني هذه القدرة على دمج الأزواج أنه يمكن استخدام الأزواج كـلبنات بناء عامة الغرض لإنشاء جميع أنواع هياكل البيانات المعقدة. وتعتبر الآلية الأولية الوحيدة للبيانات المركبة #emph[الزوج]، المنفذة بالدوال #py("pair") و #py("head") و #py("tail")، هي الغراء الوحيد الذي نحتاجه. وتُسمّى كائنات البيانات المنشأة من الأزواج
#idx("list structure")
#idx("data", sub: "list-structured")
#emph[بيانات ذات بنية قائمة (#en[list-structured data])].

#subheading([تمثيل الأعداد الناطقة])

توفر الأزواج طريقة طبيعية لإكمال نظام الأعداد الناطقة.
ببساطة، مَثِّل العدد الناطق كـ زوج من عددين صحيحين: بسط ومقام. وعندئذ تُنفَّذ #py("make_rat") و #py("numer") و #py("denom") بسهولة كما يلي:#footnote[طريقة أخرى لتعريف المحدّدات والمُنشِئ هي:

#snippet(```python
make_rat = pair
numer = head
denom = tail
```)

التعريف الأول يربط الاسم #py("make_rat") بقيمة التعبير #py("pair")، وهو الدالة الأولية التي تبني الأزواج. وبالتالي فإنّ #py("make_rat") و #py("pair") هما اسمان للمُنشِئ الأولي نفسه.

وتعريف المحدّدات والمُنشِئات بهذه الطريقة كفؤ: فبدلاً من استدعاء #py("pair") داخل #py("make_rat")، فإنّ #py("make_rat") هي نفسها #py("pair")، ولذا هناك دالة واحدة فقط تُستدعى، وليس دالتان، عند استدعاء #py("make_rat"). ومن ناحية أخرى، فإن هذا يضر بأدوات التنقيح التي تتتبع استدعاءات الدوال أو تضع نقاط توقف على استدعاءات الدوال: فقد ترغب في مشاهدة استدعاء #py("make_rat")، لكنك بالتأكيد لا تريد مشاهدة كل استدعاء لـ #py("pair").

ولقد اخترنا عدم استخدام هذا النمط من التعريفات في هذا الكتاب.
#anchor(<foot:proc-def-style>)]
#idx("makerat", decl: true)#idx("numer", decl: true)#idx("denom", decl: true)
#snippet(```python
def make_rat(n, d): return pair(n, d)

def numer(x): return head(x)

def denom(x): return tail(x)
```)

أيضاً، من أجل عرض نتائج حساباتنا، يمكننا
#idx("rational number(s)", sub: "printing")
طباعة الأعداد الناطقة بطباعة البسط، ثم خط مائل، ثم المقام.
ونستخدم الدالة الأولية
#idx("str (primitive function)")
#py("str") لتحويل أي قيمة (هنا عدد) إلى سلسلة نصية. والمعامل
#idx("string(s)", sub: "concatenation")
#idx("concatenating strings")
#idx("+", sub: "as string concatenation operator")
#py("+") في بايثون
#idx("overloaded operator +")
#emph[مُحمَّل بأكثر من معنى]؛ حيث يمكن تطبيقه على عددين أو على سلسلتين نصيتين، وفي الحالة الأخيرة يرجع نتيجة #emph[ربط (دمج)] السلسلتين النصيتين.

#idx("printrat", decl: true)
#snippet(```python
def print_rat(x):
    print(str(numer(x)) + " / " + str(denom(x)))
```)

الآن يمكننا تجربة دوال الأعداد الناطقة لدينا:

#snippet(```python
one_half = make_rat(1, 2)

print(print_rat(one_half))
```)

#output(```python
one_half = make_rat(1, 2)

print(print_rat(one_half))
```)

#snippet(```python
one_third = make_rat(1, 3)
```)

#snippet(```python
print(print_rat(add_rat(one_half, one_third)))
```)

#output(```python
print(print_rat(add_rat(one_half, one_third)))
```)

#snippet(```python
print(print_rat(mul_rat(one_half, one_third)))
```)

#output(```python
print(print_rat(mul_rat(one_half, one_third)))
```)

#snippet(```python
print(print_rat(add_rat(one_third, one_third)))
```)

#output(```python
print(print_rat(add_rat(one_third, one_third)))
```)

كما يوضح المثال الأخير، فإنّ تنفيذنا للأعداد الناطقة لا
#idx("rational number(s)", sub: "reducing to lowest terms")
#idx("reducing to lowest terms")
يختزل الأعداد الناطقة إلى أبسط صورة. ويمكننا علاج ذلك بتغيير #py("make_rat").
فإذا كانت لدينا دالة #py("gcd") مثل تلك الموجودة في القسم @sec:gcd والتي تنتج
#idx("greatest common divisor", sub: "used in rational-number arithmetic")
القاسم المشترك الأعظم لعددين صحيحين، فيمكننا استخدام #py("gcd") لاختزال البسط والمقام إلى أبسط صورة قبل بناء الزوج:

#idx("makerat", sub: "reducing to lowest terms", decl: true)
#snippet(```python
def make_rat(n, d):
    g = gcd(n, d)
    return pair(n // g, d // g)
```)

الآن لدينا:

#snippet(```python
print_rat(add_rat(one_third, one_third))
```)

#output(```python
print_rat(add_rat(one_third, one_third))
```)

كما هو مطلوب. وقُدِّم هذا التعديل بتغيير المُنشِئ #py("make_rat") دون تغيير أي من الدوال (مثل #py("add_rat") و #py("mul_rat")) التي تنفذ العمليات الفعلية.

#exercise(label-name: <ex:2_1>, [
عرف نسخة أفضل من #py("make_rat") تتعامل مع كل من المعطيات الموجبة والسالبة.
ويجب أن تقوم الدالة #py("make_rat") بمعايرة الإشارة بحيث إذا كان العدد الناطق موجباً، يكون كل من البسط والمقام موجباً، وإذا كان العدد الناطق سالباً، يكون البسط فقط سالباً.
])

#idx("arithmetic", sub: "on rational numbers")
#idx("rational number(s)", sub: "arithmetic operations on")
#idx("rational-number arithmetic")
