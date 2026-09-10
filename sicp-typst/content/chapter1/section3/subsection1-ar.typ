// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([الدوال كـمعطيات], label-name: <sec:procedures-as-parameters>)

#idx("higher-order functions", sub: "function as argument")

تأمَّل الدوال الثلاث التالية.
الأولى تحسب مجموع الأعداد الصحيحة من #py("a") حتى #py("b"):
#idx("sumintegers", decl: true)
#snippet(```python
def sum_integers(a, b):
    return 0 if a > b else a + sum_integers(a + 1, b)
```)

والثانية تحسب مجموع مكعبات الأعداد الصحيحة في النطاق المعطى:
#idx("sumcubes", decl: true)
#snippet(```python
def sum_cubes(a, b):
    return 0 if a > b else cube(a) + sum_cubes(a + 1, b)
```)

والثالثة تحسب مجموع متتالية من الحدود في المتسلسلة:

$ frac(1, 1 dot.op 3)+frac(1, 5 dot.op 7)+frac(1, 9 dot.op 11)+ dots.c $

والتي تتقارب نحو $pi /8$ (ببطء شديد):#footnote[هذه المتسلسلة،
#idx("π (pi)", sub: "Leibniz's series for", sort: "pi")
#idx("Leibniz, Baron Gottfried Wilhelm von", sub: "series for π")
التي تُكتَب عادة بالشكل المكافئ
$frac( pi , 4) = 1-frac(1, 3)+frac(1, 5)-frac(1, 7)+ dots.c$،
تعود إلى لايبنتز. وسنرى كيفية استخدامها كأساس لبعض الحيل العددية المتقدمة في القسم @sec:exploiting-streams.]
#idx("pisum", decl: true)
#snippet(```python
def pi_sum(a, b):
    return 0 if a > b else 1 / (a * (a + 2)) + pi_sum(a + 4, b)
```)

تشترك هذه الدوال الثلاث بوضوح في نمط أسايسي مشترك. فهي متطابقة في معظمها، وتختلف فقط في اسم الدالة، والدالة المطبَّقة على #py("a") لحساب الحد المراد إضافته، والدالة التي تقدم القيمة التالية لـ #py("a"). وكان بإمكاننا إنشاء كل من هذه الدوال بملء الفراغات في القالب نفسه:

#snippet(```python
def name(a, b):
    return (0 if a > b
            else term(a) + name(next(a), b))
```)

إن وجود مثل هذا النمط المشترك دلالة قوية على وجود
#idx("abstraction", sub: "common pattern and")
تجريد مفيد ينتظر الخروج إلى السطح. وبالفعل، حدد علماء الرياضيات منذ زمن طويل تجريد
#idx("series, summation of")
#idx("summation of a series")
#idx("Σ (sigma) notation", sort: "sigma")
#idx("Σ (sigma) notation", sort: "0s")
#emph[جمع المتسلسلات] واخترعوا «ترميز سيجما»، على سبيل المثال:

$ mat(delim: #none, sum_(n=a)^(b) space f(n), =, f(a)+ dots.c +f(b)) $

للتعيير عن هذا المفهوم. وتكمن قوة ترميز سيجما في أنه يتيح لعلماء الرياضيات التعامل مع مفهوم الجمع ذاته بدلاً من التعامل فقط مع مجاميع محددة — على سبيل المثال، صياغة نتائج عامة حول المجاميع تكون مستقلة عن المتسلسلة المحددة الجاري جمعها.

وبالمثل، كمصممي برامج، نودّ أن تكون لغتنا قوية بدرجة كافية بحيث يمكننا كتابة دالة تعبر عن مفهوم الجمع ذاته بدلاً من كتابة دوال فقط تحسب مجاميع محددة. ويمكننا القيام بذلك بسهولة في لغتنا الوظيفية بآخذ القالب المشترك المعروض أعلاه وتحويل «الفراغات» إلى معاملات:
#idx("sum", decl: true)
#snippet(```python
def sum(term, a, next, b):
    return 0 if a > b else term(a) + sum(term, next(a), next, b)
```)

لاحظ أنّ #py("sum") تأخذ كمعطيات الحدين الأدنى والأعلى #py("a") و #py("b") جنباً إلى جنب مع الدالتين #py("term") و #py("next"). ويمكننا استخدام #py("sum") تماماً كما نستخدم أي دالة. على سبيل المثال، يمكننا استخدامها (جنباً إلى جنب مع دالة #py("inc") التي تزيد معطيتها بمقدار 1) لتعريف #py("sum_cubes"):

#idx("inc", decl: true)#idx("sumcubes", sub: "with higher-order functions", decl: true)
#snippet(```python
def inc(n):
    return n + 1
def sum_cubes(a, b):
    return sum(cube, a, inc, b)
```)

باستخدام هذا، يمكننا حساب مجموع مكعبات الأعداد الصحيحة من 1 إلى 10:

#snippet(```python
print(sum_cubes(1, 10))
```)

#output(```python
print(sum_cubes(1, 10))
```)

وبمساعدة دالة التماثل (المطابقة) لحساب الحد، يمكننا تعريف #py("sum_integers") بدلالة #py("sum"):
#idx("identity", decl: true)
#snippet(```python
def identity(x):
    return x
```)

#idx("sumintegers", sub: "with higher-order functions", decl: true)
#snippet(```python
def sum_integers(a, b):
    return sum(identity, a, inc, b)
```)

ثم يمكننا جمع الأعداد الصحيحة من 1 إلى 10:

#snippet(```python
print(sum_integers(1, 10))
```)

#output(```python
print(sum_integers(1, 10))
```)

ويمكننا أيضاً تعريف #py("pi_sum") بنفس الطريقة:#footnote[لاحظ أننا استخدمنا البنية المكتفية ذاتياً (الكتلية) (القسم @sec:black-box) لضم تعاريف #py("pi_next") و #py("pi_term") داخل #py("pi_sum")، نظرًا لأنه من غير المرجح أن تكون هذه الدوال مفيدة لأي غرض آخر. وسنرى كيفية التخلص منها تماماً في القسم @sec:lambda.]
#idx("pisum", sub: "with higher-order functions", decl: true)
#snippet(```python
def pi_sum(a, b):
    def pi_term(x):
        return 1 / (x * (x + 2))
    def pi_next(x):
        return x + 4
    return sum(pi_term, a, pi_next, b)
```)

باستخدام هذه الدوال، يمكننا حساب تقريب لـ $pi$:

#snippet(```python
print(8 * pi_sum(1, 1000))
```)

#output(```python
print(8 * pi_sum(1, 1000))
```)

وبمجرد الحصول على #py("sum")، يمكننا استخدامها كلبنة بناء في صياغة المزيد من المفاهيم. على سبيل المثال، يمكن تقريب
#idx("definite integral")
التكامل المحدد لدالة $f$ بين الحدين $a$ و $b$ عديدياً باستخدام الصيغة:

$ mat(delim: #none, integral_(a)^(b)f, =, lr([ thin f l r(( a+frac(d x, 2) )) thin + thin f l r(( a+d x+frac(d x, 2) )) thin + thin f l r(( a+2d x+frac(d x, 2) )) thin + thin dots.c ]) d x) $

لقيم صغيرة لـ $d x$. ويمكننا التعبير عن ذلك مباشرة كدالة:
#idx("integral", decl: true)
#snippet(```python
def integral(f, a, b, dx):
    def add_dx(x):
        return x + dx
    return sum(f, a + dx / 2, add_dx, b) * dx
```)

#snippet(```python
print(integral(cube, 0, 1, 0.01))
```)

#output(```python
print(integral(cube, 0, 1, 0.01))
```)

#snippet(```python
print(integral(cube, 0, 1, 0.001))
```)

#output(```python
print(integral(cube, 0, 1, 0.001))
```)

(القيمة الدقيقة لتكامل #py("cube") بين 0 و 1 هي 1/4).

#exercise(label-name: <ex:simpsons-rule>, [
قاعدة سيمبسون هي طريقة أكثر دقة للتكامل العددي من

#idx("Simpson's Rule for numerical integration")
الطريقة الموضحة أعلاه. باستخدام قاعدة سيمبسون، يُقرَّب تكامل دالة $f$ بين $a$ و $b$ كـ:

$ frac(h, 3)[ y_(0) +4y_(1) +2y_(2) +4y_(3) +2y_(4) + dots.c +2y_(n-2) +4y_(n-1)+y_(n) ] $

حيث $h=(b-a)/n$، لعدد صحيح زوجي $n$، و $y_(k) =f(a+k h)$. (زيادة $n$ تزيد من دقة التقريب).
عرف دالة تأخذ كمعطيات $f$ و $a$ و $b$ و $n$ وترجع قيمة التكامل، المنسوبة باستخدام قاعدة سيمبسون. واستخدم دالتك لمكاملة #py("cube") بين 0 و 1 (مع $n=100$ و $n=1000$)، وقارن النتائج بتلك الخاصة بدالة #py("integral") الموضحة أعلاه.

#anchor(<ex:1_29>)
])

#idx("definite integral")

#exercise(label-name: <ex:1_30>, [
تُنشئ دالة
#idx("sum", sub: "iterative version")
#py("sum")
أعلاه عَوْدِيَّة خطية. ويمكن إعادة كتابة الدالة بحيث يُنفَّذ الجمع تكرارياً. أظهر كيفية القيام بذلك بملء التعبيرات المفقودة في التعريف التالي:

#snippet(```python
def sum(term, a, next, b):
    def iterate(a, result):
        return (??
                if ??
                else iterate(??, ??))
    return iterate(??, ??)
```)
])

#exercise(label-name: <ex:product>, [
+ دالة #py("sum") هي مجرد التجريد الأكثر بساطة من عدد هائل من التجريدات المماثلة التي يمكن التقاطها كدوال من رتبة أعلى.#footnote[الهدف من التمارين @ex:product–@ex:filtered-accumulate هو إظهار القوة التعبيرية التي تُحقَّق باستخدام تجريد مناسب لدمج العديد من العمليات التي تبدو متباينة. ومع ذلك، ورغم أنّ التجميع الترشيحي والتصفية فكرتان أنيقتان، فإنّ أيدينا مقيدة نوعاً ما في استخدامها في هذه النقطة نظرًا لأنه ليس لدينا بعد هياكل بيانات لتوفير وسائل تجميع مناسبة لهذه التجريدات. وسنعود إلى هذه الأفكار في القسم @sec:sequences-conventional-interfaces عندما نوضح كيفية استخدام #emph[المتتاليات] كواجهات لدمج المرشِّحات والمجمِّعات لبناء تجريدات أكثر قوة. وسنرى هناك كيف تصبح هذه الطرق قوية وأنيقة لتصميم البرامج.] اكتب دالة مناظرة تُسمّى #idx("product") #py("product") ترجع حاصل ضرب قيم دالة عند نقاط عبر نطاق معطى. وأظهر كيفية تعريف #idx("factorial", sub: "with higher-order functions") #py("factorial") بدلالة #py("product"). واستخدم أيضاً #py("product") لحساب تقريبات لـ #idx("π (pi)", sub: "Wallis's formula for", sort: "pi") $pi$ باستخدام الصيغة:#footnote[اكتُشِفَت هذه الصيغة بواسطة عالم الرياضيات الإنجليزي في القرن السابع عشر #idx("Wallis, John") جون واليس #en[(John Wallis)].] $ mat(delim: #none, frac( pi , 4), =, frac(2 dot.op 4 dot.op 4 dot.op 6 dot.op 6 dot.op 8 dots.c , 3 dot.op 3 dot.op 5 dot.op 5 dot.op 7 dot.op 7 dots.c )) $
+ إذا كانت دالة #py("product") لديك تُنشئ عملية عَوْدِيَّة، فاكتب دالة تُنشئ عملية تكرارية. وإذا كانت تُنشئ عملية تكرارية، فاكتب دالة تُنشئ عملية عَوْدِيَّة.
])

#exercise(label-name: <ex:accumulate>, [
+ أظهر أنّ #py("sum") و #py("product") #idx("sum", sub: "as accumulation") #idx("product", sub: "as accumulation") (التمرين @ex:product) كلاهما حالتان خاصتان لمفهوم أكثر عمومية يُسمّى #idx("accumulate") #py("accumulate") يدمج مجموعة من الحدود، باستخدام دالة تجميع عامة: #snippet(```python accumulate(combiner, null_value, term, a, next, b) ```) تأخذ الدالة #py("accumulate") كمعطيات نفس مواصفات الحد والنطاق كما في #py("sum") و #py("product")، جنباً إلى جنب مع دالة #py("combiner") (ذات معطيين) تحدد كيفية دمج الحد الحالي مع تجميع الحدود السابقة و #py("null_value") التي تحدد القيمة الأساسية التي تُستخدَم عند نفاد الحدود. اكتب #py("accumulate") وأظهر كيف يمكن تعريف كل من #py("sum") و #py("product") كاستدعاءات بسيطة لـ #py("accumulate").
+ إذا كانت دالة #py("accumulate") لديك تُنشئ عملية عَوْدِيَّة، فاكتب دالة تُنشئ عملية تكرارية. وإذا كانت تُنشئ عملية تكرارية، فاكتب دالة تُنشئ عملية عَوْدِيَّة.
])

#exercise(label-name: <ex:filtered-accumulate>, [
يمكنك الحصول على نسخة أكثر عمومية من
#idx("filteredaccumulate")
#py("accumulate")
(التمرين @ex:accumulate)
بتقديم مفهوم
#idx("filter")
#emph[المرشِّح (#en[filter])] على الحدود المراد دمجها. أي، ادمج فقط تلك الحدود المشتقة من قيم في النطاق والتي تحقق شرطاً محدداً. التجريد الناتج
#idx("filteredaccumulate")
#py("filtered_accumulate")
يأخذ نفس معطيات #py("accumulate")، جنباً إلى جنب مع محمول إضافي ذي معطى واحد يحدد المرشِّح. اكتب #py("filtered_accumulate") كدالة. وأظهر كيفية التعبير عما يلي باستخدام #py("filtered_accumulate"):

+ مجموع مربعات الأعداد الأولية في الفاصل الزمني $a$ إلى $b$ (بفرض أن لديك محمول #py("is_prime") مكتوباً بالفعل)
+ حاصل ضرب جميع الأعداد الصحيحة الموجبة الأقل من $n$ والتي هي #idx("relatively prime") أولية فيما بينها وبين $n$ (أي، جميع الأعداد الصحيحة الموجبة $i < n$ بحيث $upright("GCD")(i,n)=1$).
])

#idx("higher-order functions", sub: "function as argument")
