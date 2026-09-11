// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([التدفقات اللانهائية], label-name: <sec:infinite-streams>)

#idx("infinite stream(s)")

رأينا كيفية دعم وهم التعامل مع التدفقات ككيانات كاملة على الرغم من أننا في الواقع نحسب فقط القدر الذي نحتاجه للوصول إليه من التدفق. يمكننا استغلال هذه التقنية لتمثيل التسلسلات بكفاءة كتدفقات، حتى لو كانت التسلسلات طويلة جداً. وما هو أكثر إثارة للإعجاب، أنه يمكننا استخدام التدفقات لتمثيل تسلسلات لانهائية الطول. على سبيل المثال، فكر في التعريف التالي لتدفق الأعداد الصحيحة الموجبة:

#idx("integersstartingfrom", decl: true)
#snippet(```python
def integers_starting_from(n):
    return pair(n, lambda: integers_starting_from(n + 1))
```)

#idx("integers (infinite stream)", decl: true)
#snippet(```python
integers = integers_starting_from(1)
```)

هذا منطقي لأن #py("integers") سيكون زوجاً رأسه (#py("head")) هو 1 وذيله (#py("tail")) هو وعد بإنتاج الأعداد الصحيحة بدءاً من 2. هذا تدفق لانهائي الطول، ولكن في أي وقت معين يمكننا فحص جزء منتهٍ فقط منه. وبالتالي، فإن برامجنا لن تعثر أبداً على أن التدفق اللانهائي بأكمله ليس موجوداً.

باستخدام #py("integers")، يمكننا تعريف تدفقات لانهائية أخرى، مثل تدفق الأعداد الصحيحة التي لا تقبل القسمة على 7:

#idx("isdivisible", decl: true)
#snippet(```python
def is_divisible(x, y):
    return x % y == 0
```)

#snippet(```python
no_sevens = stream_filter(lambda x: not is_divisible(x, 7),
                          integers)
```)

ثم يمكننا العثور على أعداد صحيحة لا تقبل القسمة على 7 ببساطة عن طريق الوصول إلى عناصر هذا التدفق:

#snippet(```python
print(stream_ref(no_sevens, 100))
```)

#output(```python
print(stream_ref(no_sevens, 100))
```)

وبالمثل مع #py("integers")، يمكننا تعريف التدفق اللانهائي لأعداد فيبوناتشي:

#idx("fibs (infinite stream)", decl: true)
#snippet(```python
def fibgen(a, b):
    return pair(a, lambda: fibgen(b, a + b))

fibs = fibgen(0, 1)
```)

الثابت #py("fibs") هو زوج رأسه (#py("head")) هو 0 وذيله (#py("tail")) هو وعد بتقييم #py("fibgen(1, 1)").
وعندما نُقيّم هذا الاستدعاء المؤجل لـ #py("fibgen(1, 1)")، فإنه سينتج زوجاً رأسه (#py("head")) هو 1 وذيله (#py("tail")) هو وعد بتقييم #py("fibgen(1, 2)")، وهكذا.

لإلقاء نظرة على تدفق لانهائي أكثر إثارة، يمكننا تعميم مثال #py("no_sevens") لبناء التدفق اللانهائي للأعداد الأولية، باستخدام طريقة #emph[الغربلة] (#en[sieving]).#footnote[تُذكرنا هذه الطريقة بـ
#idx("prime number(s)", sub: "Eratosthenes's sieve for")
#idx("sieve of Eratosthenes")
#emph[غربال إراتوستينس] القديم، المسمى باسم
#idx("Eratosthenes")
إراتوستينس، وهو عالم رياضيات يوناني سكندري من القرن الثالث قبل الميلاد. يجد غرباله الأعداد الأولية عن طريق الشطب التكراري لمضاعفات كل عدد أولي بدوره؛ وعلى الرغم من قدمه، إلا أنه شكل الأساس لـ "غربالات" عتادية لأغراض خاصة كانت، حتى سبعينيات القرن العشرين، أكثر الأدوات كفاءة لتحديد موقع الأعداد الأولية الكبيرة. ومنذ ذلك الحين، تُمّ استبدال هذه الطرق بنواتج التقنيات الاحتمالية المناقشة في القسم @sec:primality. لكن الطريقة المعتمدة على التدفق المعطاة هنا ليست في الواقع غربال إراتوستينس: كما توضح ميليسا أو نيل في O'Neill 2009، فهي تختبر كل مرشح للقسمة على الأعداد الأولية التي تُمّ العثور عليها حتى الآن—وهو شكل من أشكال #emph[القسمة التجريبية]—بدلاً من شطب المضاعفات، وتعد أقل كفاءة بكثير من الغربال الأصلي.]
نبدأ بالأعداد الصحيحة بدءاً من 2، وهو العدد الأولي الأول. وللحصول على باقي الأعداد الأولية، نبدأ بتصفية مضاعفات 2 من باقي الأعداد الصحيحة. هذا يترك تدفقاً يبدأ بـ 3، وهو العدد الأولي التالي. الآن نصفي مضاعفات 3 من باقي هذا التدفق. هذا يترك تدفقاً يبدأ بـ 5، وهو العدد الأولي التالي، وهكذا. بعبارة أخرى، نبني الأعداد الأولية بواسطة عملية غربلة، موصوفة كما يلي: لغربلة تدفق S، نُشكل تدفقاً عنصره الأول هو العنصر الأول من S والحاصل الحصول على باقيه بتصفية جميع مضاعفات العنصر الأول من S خارج باقي S وغربلة النتيجة. يُوصف هذه العملية بسهولة بدلالة عمليات التدفق:

#idx("primes (infinite stream)", decl: true)#idx("sieve of Eratosthenes", sub: "sieve", decl: true)
#snippet(```python
def sieve(stream):
    return pair(head(stream),
                lambda: sieve(stream_filter(
                                  lambda x: not is_divisible(x, head(stream)),
                                  stream_tail(stream))))

primes = sieve(integers_starting_from(2))
```)

الآن للعثور على عدد أولي معين، نحتاج فقط إلى طلبه:

#snippet(```python
print(stream_ref(primes, 50))
```)

#output(```python
print(stream_ref(primes, 50))
```)

من المثير للاهتمام تأمل نظام معالجة الإشارات المستقر بواسطة #py("sieve")، والموضح في
#idx("Henderson, Peter", sub: "Henderson diagram")
"مخطط هندرسون" في الشكل @fig:primesieve.#footnote[قمنا بتسمية هذه الأشكال باسم
#idx("Henderson, Peter")
بيتر هندرسون، الذي كان أول شخص يرينا مخططات من هذا النوع كطريقة للتفكير في معالجة التدفقات.] يتغذى تدفق الدخل في "مُفكك أزواج" يفصل العنصر الأول من التدفق عن باقي التدفق.
يُستخدم العنصر الأول لبناء مرشِّح القابلية للقسمة، يمر منه ما تبقى، ويُغذى خرج المرشح إلى صندوق غربال آخر. ثم يُضم العنصر الأول الأصلي إلى خرج الغربال الداخلي لتشكيل تدفق الخرج.
وهكذا، ليس التدفق لانهائياً فحسب، بل إن معالج الإشارات لانهائي أيضاً، لأن الغربال يحتوي على غربال داخله.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-35.svg", width: 57%), caption: [غربال الأعداد الأولية من منظور نظام معالجة الإشارات. يمثل كل خط متصل تدفقاً من القيم المُنقولة. ويدل الخط المنقط من #py("head") إلى #py("pair") و #py("filter") على أن هذه قيمة واحدة وليست تدفقاً.], label-name: <fig:primesieve>)

#subheading([تعريف التدفقات ضمنياً])

#idx("stream(s)", sub: "implicit definition")

تُمّ تعريف تدفقات #py("integers") و #py("fibs") أعلاه بتحديد دوال "مولدة" تحسب عناصر التدفق صراحة واحداً تلو الآخر. والطريقة البديلة لتحديد التدفقات هي الاستفادة من التقييم المؤجل لتعريف التدفقات ضمنياً. على سبيل المثال، تحدد العبارة التالية التدفق #py("ones") ليكون تدفقاً لانهائياً من الآحاد:

#idx("ones (infinite stream)", decl: true)
#snippet(```python
ones = pair(1, lambda: ones)
```)

يعمل هذا مثل إعلان دالة عودية: #py("ones") هو زوج رأسه (#py("head")) هو 1 وذيله (#py("tail")) هو وعد بتقييم #py("ones"). وتقييم الذيل يعطينا مرة أخرى 1 ووعداً بتقييم #py("ones")، وهكذا.

يمكننا القيام بأشياء أكثر إثارة للاهتمام عن طريق تداول التدفقات بعمليات مثل #py("add_streams")، والتي تنتج المجموع عنصراً بعنصر لتدفقين معطيين:#footnote[تستخدم هذه الدالة #py("stream_map_2") من التمرين @ex:combine-streams.]

#idx("addstreams", decl: true)
#snippet(```python
def add_streams(s1, s2):
    return stream_map_2(lambda x1, x2: x1 + x2, s1, s2)
```)

الآن يمكننا تعريف الأعداد الصحيحة كما يلي:

#idx("integers (infinite stream)", sub: "implicit definition", decl: true)
#snippet(```python
integers = pair(1, lambda: add_streams(ones, integers))
```)

يحدد هذا #py("integers") ليكون تدفقاً عنصره الأول هو 1 وباقيه هو مجموع #py("ones") و #py("integers").
وبالتالي، فإن العنصر الثاني من #py("integers") هو 1 زائد العنصر الأول من #py("integers")، أو 2؛ والعنصر الثالث من #py("integers") هو 1 زائد العنصر الثاني من #py("integers")، أو 3؛ وهكذا. يعمل هذا التعريف لأنه في أي نقطة، تُمّ توليد ما يكفي من تدفق #py("integers") بحيث يمكننا إعادته إلى التعريف لإنتاج العدد الصحيح التالي.

يمكننا تعريف أعداد فيبوناتشي بالأسلوب نفسه:

#idx("fibs (infinite stream)", sub: "implicit definition", decl: true)
#snippet(```python
fibs = pair(0,
            lambda: pair(1,
                         lambda: add_streams(stream_tail(fibs),
                                             fibs)))
```)

يقول هذا التعريف إن #py("fibs") هو تدفق يبدأ بـ 0 و 1، بحيث يمكن توليد باقي التدفق بإضافة #py("fibs") إلى نفسه مُزاحاً بمقدار موضع واحد:

$ mat(delim: #none, , , 1, 1, 2, 3, 5, 8, 13, 21, dots.h, =, mono("stream")mono("_")mono("tail(fibs)"); , , 0, 1, 1, 2, 3, 5, 8, 13, dots.h, =, mono("fibs"); 0, 1, 1, 2, 3, 5, 8, 13, 21, 34, dots.h, =, mono("fibs")) $

الدالة #py("scale_stream") مفيدة أيضاً في صياغة مثل تعاريف التدفقات هذه. حيث تضرب كل عنصر في التدفق بثابت معطى:

#idx("scalestream", decl: true)
#snippet(```python
def scale_stream(stream, factor):
    return stream_map(lambda x: x * factor,
                      stream)
```)

على سبيل المثال،

#snippet(```python
double = pair(1, lambda: scale_stream(double, 2))
```)

ينتج تدفق قوى 2:
$1, 2, 4, 8, 16, 32,$ ….

يمكن إعطاء تعريف بديل لتدفق الأعداد الأولية بالبدء بالأعداد الصحيحة وتصفيتها باختبار أوليتها. سنحتاج إلى العدد الأولي الأول، 2، للبدء:

#idx("primes (infinite stream)", sub: "implicit definition", decl: true)
#snippet(```python
primes = pair(2,
              lambda: stream_filter(is_prime,
                                    integers_starting_from(3)))
```)

هذا التعريف ليس مستقيماً وبسيطاً كما يبدو، لأننا سنختبر ما إذا كان العدد $n$ أولياً بالتحقق مما إذا كان $n$ يقبل القسمة على عدد أولي (وليس فقط على أي عدد صحيح) أقل من أو يساوي $sqrt(n)$:

#idx("isprime", decl: true)
#snippet(```python
def is_prime(n):
    def iter(ps):
        return (True
                if square(head(ps)) > n
                else False
                if is_divisible(n, head(ps))
                else iter(stream_tail(ps)))
    return iter(primes)
```)

هذا تعريف عودي، حيث تُمّ تعريف #py("primes") بدلالة دالة شرطية #py("is_prime")، والتي تستخدم نفسها تدفق #py("primes").
والسبب في عمل هذه الدالة هو أنه في أي نقطة، تُمّ توليد ما يكفي من تدفق #py("primes") لاختبار أولية الأعداد التي نحتاج إلى فحصها بعد ذلك. أي أنه لكل $n$ نختبر أوليته، إما أن $n$ ليس أولياً (وفي هذه الحالة توجد بالفعل أعداد أولية تُمّ توليدها تقسمه) أو أن $n$ أولي (وفي هذه الحالة توجد بالفعل أعداد أولية تُمّ توليدها—أي أعداد أولية أقل من $n$—أكبر من $sqrt(n)$).#footnote[هذه النقطة الأخيرة دقيقة للغاية وتعتمد على حقيقة أن $p_(n+1) lt.eq p_(n)^(2)$. (هنا، يمثل $p_(k)$ العدد الأولي الـ $k$). تقديرات مثل هذه صعبة التحديد للغاية. يوضح إثبات اقليدس القديم
#idx("Euclid's proof of infinite number of primes")
لأنه توجد أعداد لانهائية من الأعداد الأولية أن
$p_(n+1) lt.eq p_(1) p_(2) thin dots.c thin thin p_(n) +1$،
ولم تُثبت أي نتيجة أفضل بشكل جوهري حتى عام 1851، عندما أثبت عالم الرياضيات الروسي
#idx("Chebyshev, Pafnutii L'vovich")
بافنوتي تشيبيشيف
أن $p_(n+1) lt.eq 2p_(n)$ لكل $n$. وتُعرف هذه النتيجة، التي تُمّ تخمينها في الأصل عام 1845، باسم
#idx("Bertrand's Hypothesis")
#emph[فرضية برتراند]. يمكن العثور على الإثبات في القسم 22.3 من
#idx("Hardy, Godfrey Harold")
#idx("Wright, E. M.")
هاردي ورايت 1960.]
#idx("stream(s)", sub: "implicit definition")

#exercise(label-name: <ex:without_running>, [
دون تشغيل البرنامج، صف عناصر التدفق المعرف بواسطة:

#snippet(```python
s = pair(1, lambda: add_streams(s, s))
```)
])

#exercise(label-name: <ex:element_wise_product>, [
عرِّف دالة
#idx("mulstreams")
#idx("infinite stream(s)", sub: "of factorials")
#idx("factorial", sub: "infinite stream")
#py("mul_streams")،
مماثلة لـ #py("add_streams")، تنتج الحاصل ضرب عنصراً بعنصر لتدفقَي دخلها. استخدم هذه الدالة جنباً إلى جنب مع تدفق #py("integers") لإكمال التعريف التالي للتدفق الذي عنصره الـ $n$ (بالعد من 0) هو مضروب $n+1$:

#syntax("
factorials = pair(1, lambda: mul_streams(", metaphrase[??], ", ", metaphrase[??], "))
      ")
])

#exercise(label-name: <ex:partial-sums>, [
عرِّف دالة
#idx("partialsums")
#py("partial_sums")
تأخذ كوسيطة تدفقاً $S$ وترجع التدفق الذي عناصره هي
$S_(0), S_(0)+S_(1), S_(0)+S_(1)+S_(2),$ ….
على سبيل المثال، ينبغي أن يكون
#py("partial_sums(integers)")
هو التدفق $1, 3, 6, 10, 15, dots.h$.
])

#exercise(label-name: <ex:merge>, [
مشكلة شهيرة، طرحها لأول مرة
#idx("Hamming, Richard Wesley")
ريتشارد هالمينغ، هي تعداد، بترتيب تصاعدي ودون تكرارات، لجميع الأعداد الصحيحة الموجبة التي ليس لها عوامل أولية غير 2 أو 3 أو 5. إحدى الطرق الواضحة للقيام بذلك هي ببساطة اختبار كل عدد صحيح بدوره لمعرفة ما إذا كان له أي عوامل غير 2 و 3 و 5. لكن هذا غير فعال للغاية، لأنه مع كبر الأعداد الصحيحة، فإن عدداً أقل فأقل منها يفي بالشرط. كبديل، دعنا نسمي التدفق المطلوب من الأعداد
#py("S") ونلاحظ الحقائق التالية عنه.

- #py("S") يبدأ بـ 1.
- عناصر #py("scale_stream(S, 2)") هي أيضاً عناصر من #py("S").
- الشيء نفسه صحيح بالنسبة لـ #py("scale_stream(S, 3)") و #py("scale_stream(S, 5)").
- هذه هي جميع عناصر #py("S").

الآن كل ما علينا فعله هو دمج العناصر من هذه المصادر. ولهذا نُعرّف دالة
#idx("infinite stream(s)", sub: "merging")
#py("merge") تدمج تدفقين مرتبين في تدفق نتيجة مرتب واحد، مُستبعدةً التكرارات:

#idx("merge", decl: true)
#snippet(```python
def merge(s1, s2):
    if is_none(s1):
        return s2
    elif is_none(s2):
        return s1
    else:
        s1head = head(s1)
        s2head = head(s2)
        return (pair(s1head, lambda: merge(stream_tail(s1), s2))
                if s1head < s2head
                else pair(s2head, lambda: merge(s1, stream_tail(s2)))
                if s1head > s2head
                else pair(s1head, lambda: merge(stream_tail(s1), stream_tail(s2))))
```)

ثم يمكن بناء التدفق المطلوب باستخدام #py("merge")، كما يلي:

#syntax("
S = pair(1, lambda: merge(", metaphrase[??], ", ", metaphrase[??], "))
      ")

املأ التعبيرات المفقودة في الأماكن المحددة بـ #metaphrase[??] أعلاه.
])

#exercise(label-name: <ex:fib-stream-efficiency>, [
كم عدد عمليات الإضافة التي تُجرى عندما نحسب عدد فيبوناتشي الـ $n$ باستخدام إعلان #py("fibs") القائم على الدالة #py("add_streams")؟ بين أن هذا العدد أكبر أسياً من عدد عمليات الإضافة المنفذة إذا كانت #py("add_streams") قد استخدمت الدالة #py("stream_map_2_optimized") الموصوفة في التمرين @ex:combine-streams، وإذا كنا قد أعلنا عن #py("fibs") كما يلي: #snippet(```python fibs = pair(0, memo(lambda: pair(1, memo(lambda: add_streams(stream_tail(fibs), fibs))))) ```)
])

#exercise(label-name: <ex:quotient>, [
أعط تفسيراً للتدفق المحسوب بواسطة الدالة:

#snippet(```python
def expand(num, den, radix):
    return pair(math_trunc((num * radix) / den),
                lambda: expand((num * radix) % den, den, radix))
```)

حيث #idx("mathtrunc (primitive function)") #py("math_trunc") تتجاهل الجزء الكسري لوسيطها، وهنا باقي القسمة.
ما هي العناصر المتتالية الناتجة عن #py("expand(1, 7, 10)")؟
ما الناتج عن #py("expand(3, 8, 10)")؟
])

#exercise(label-name: <ex:powerseries>, [
في القسم @sec:symbolic-algebra رأينا كيفية تنفيذ نظام حساب الجبر كثير الحدود بتمثيل كثرات الحدود كقوائم من الحدود. وبطريقة مماثلة، يمكننا العمل مع
#idx("power series, as stream")
#idx("infinite stream(s)", sub: "representing power series")
#idx("eˣ, power series for", sort: "e")
#idx("cosine", sub: "power series for")
#idx("sine", sub: "power series for")
#emph[متسلسلات القوى]، مثل

$ mat(delim: #none, e^(x), =, 1+x+frac(x^(2), 2)+frac(x^(3), 3 dot.op 2) +frac(x^(4), 4 dot.op 3 dot.op 2)+ dots.c ,; cos x, =, 1-frac(x^(2), 2)+frac(x^(4), 4 dot.op 3 dot.op 2)- dots.c ,; sin x, =, x-frac(x^(3), 3 dot.op 2) +frac(x^(5), 5 dot.op 4 dot.op 3 dot.op 2)- dots.c ,) $

الممثلة كتدفقات لانهائية.
سنمثل المتسلسلة
$a_(0) + a_(1) x + a_(2) x^(2) + a_(3) x^(3) + dots.c$
كالتدفق الذي عناصره هي المعاملات
$a_(0), a_(1), a_(2), a_(3),$ ….

+ تكامل
#idx("integral", sub: "of a power series") #idx("power series, as stream", sub: "integrating")
المتسلسلة $a_(0) + a_(1) x + a_(2) x^(2) + a_(3) x^(3) + dots.c$ هو المتسلسلة $ c + a_(0) x + frac(1, 2)a_(1) x^(2) + frac(1, 3)a_(2) x^(3) + frac(1, 4)a_(3) x^(4) + dots.c $ حيث $c$ هو أي ثابت. عرِّف دالة #idx("integrateseries") #py("integrate_series") تأخذ كدخل تدفقاً $a_(0), a_(1), a_(2), dots.h$ يمثل متسلسلة قوى وترجع التدفق $a_(0), frac(1, 2)a_(1), frac(1, 3)a_(2), dots.h$ لمعاملات الحدود غير الثابتة لتكامل المتسلسلة. (بما أن النتيجة ليس لها حد ثابت، فإنها لا تمثل متسلسلة قوى؛ وعندما نستخدم #py("integrate_series")، سنستخدم #py("pair") لربط الثابت المناسب ببداية التدفق.)
+ الدالة $x arrow.r.bar e^(x)$ هي المشتقة الخاصة بها. هذا يعني أن $e^(x)$ وتكامل $e^(x)$ هما المتسلسلة نفسها، باستثناء الحد الثابت، وهو $e^(0) = 1$. وبناءً على ذلك، يمكننا توليد متسلسلة $e^(x)$ ك#snippet(```python exp_series = pair(1, lambda: integrate_series(exp_series)) ```) بين كيفية توليد المتسلسلة لـ الجيب وجيب التمام، بدءاً من الحقائق أن مشتقة الجيب هي جيب التمام ومشتقة جيب التمام هي سالب الجيب: #syntax(" cosine_series = pair(1, ", metaphrase[??], ") sine_series = pair(0, ", metaphrase[??], ") ")
])

#exercise(label-name: <ex:mul-series>, [
مع تمثيل
#idx("power series, as stream", sub: "adding")
#idx("power series, as stream", sub: "multiplying")
#idx("arithmetic", sub: "on power series")
#idx("mulseries")
متسلسلات القوى كتدفقات من المعاملات كما في التمرين @ex:powerseries، يتُمّ تنفيذ إضافة المتسلسلات بواسطة #py("add_streams").
أكمل إعلان الدالة التالية لضرب المتسلسلات:

#syntax("
def mul_series(s1, s2):
    pair(", metaphrase[??], ", lambda: add_streams(", metaphrase[??], ", ", metaphrase[??], "))
      ")

يمكنك اختبار دالتك عن طريق التحقق من أن $sin^(2) x + cos^(2) x = 1$، باستخدام المتسلسلات من التمرين @ex:powerseries.
])

#exercise(label-name: <ex:invert-unit-series>, [
لتكن $S$ متسلسلة قوى (التمرين @ex:powerseries) حدها الثابت هو 1. افترض أننا نريد العثور على متسلسلة القوى $1/S$، أي المتسلسلة $X$ بحيث تكون $S dot.op X= 1$.
اكتب $S=1+S_(R)$ حيث $S_(R)$ هي جزء $S$ بعد الحد الثابت. ثم يمكننا حل $X$ كما يلي:

$ mat(delim: #none, S dot.op X, =, 1; (1+S_(R)) dot.op X, =, 1; X + S_(R) dot.op X, =, 1; X, =, 1 - S_(R) dot.op X) $

بعبارة أخرى، $X$ هي متسلسلة القوى التي حدها الثابت هو 1 وحدودها ذات الرتب الأعلى تُعطى بسالب $S_(R)$ مضروباً في $X$.
استخدم هذه الفكرة لكتابة دالة #py("invert_unit_series") تحسب $1/S$ لمتسلسلة قوى $S$ ذات حد ثابت 1. ستحتاج إلى استخدام #py("mul_series") من التمرين @ex:mul-series.
])

#exercise(label-name: <ex:diving_power_series>, [
استخدم نتائج التمارين @ex:mul-series و @ex:invert-unit-series لتحديد دالة
#idx("power series, as stream", sub: "dividing")
#idx("arithmetic", sub: "on power series")
#idx("divseries")
#py("div_series")
تقسم متسلسلي قوى.
ينبغي أن تعمل الدالة #py("div_series") لأي متسلسلتين، بشرط أن تبدأ متسلسلة المقام بحد ثابت غير صفري. (إذا كان للمقام حد ثابت صفري، فيجب أن تشير #py("div_series") إلى خطأ.) بين كيفية استخدام #py("div_series") جنباً إلى جنب مع نتيجة التمرين @ex:powerseries لتوليد متسلسلة القوى لـ
#idx("tangent", sub: "power series for")
الظل (#en[tangent]).
])

#idx("infinite stream(s)")
