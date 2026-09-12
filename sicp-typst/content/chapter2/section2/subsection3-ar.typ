// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([المتتاليات كواجهات تقليدية], label-name: <sec:sequences-conventional-interfaces>)

#idx("sequence(s)", sub: "as conventional interface")
#idx("conventional interface", sub: "sequence as")

عند العمل مع البيانات المركبة، شددنا على كيف أن تجريد البيانات يسمح لنا بتصميم البرامج دون الانغماس في تفاصيل تمثيلات البيانات، وكيف يحافظ التجريد لنا على المرونة للتجريب بتمثيلات بديلة. وفي هذا القسم، نقدم مبدأ تصميم قوياً آخر للعمل مع بنى البيانات—استخدام #emph[الواجهات التقليدية] (#en[conventional interfaces]).

في القسم @sec:higher-order-procedures رأينا كيف يمكن لتجريدات البرامج، المنفذة كدوال عليا، أن تلتقط الأنماط الشائعة في البرامج التي تتعامل مع البيانات العددية. وتعتمد قدرتنا على صياغة عمليات مماثلة للعمل مع البيانات المركبة اعتماداً حاسماً على الأسلوب الذي نعالج به بنى البيانات لدينا.
تأمل، على سبيل المثال، الدالة التالية، المماثلة لدالة #py("count_leaves") من القسم @sec:trees، والتي تأخذ شجرةً كوسيط وتحسب مجموع مربعات الأوراق الفردية:

#idx("sumoddsquares", decl: true)
#snippet(```python
def sum_odd_squares(tree):
    return (0 if is_none(tree)
            else (square(tree) if is_odd(tree)
                  else 0) if not is_pair(tree)
            else sum_odd_squares(head(tree)) +
                 sum_odd_squares(tail(tree)))
```)

على السطح، تتفاوت هذه الدالة تفاوتاً كبيراً عن الدالة التالية، والتي تبني قائمة مترابطة من جميع أعداد فيبوناتشي الزوجية $(upright("Fib"))(k)$، حيث $k$ أقل من أو يساوي عدداً صحيحاً معطى $n$:
#idx("evenfibs", decl: true)
#snippet(```python
def even_fibs(n):
    def next(k):
        if k > n:
            return None
        else:
            f = fib(k)
            return (pair(f, next(k + 1)) if is_even(f)
                    else next(k + 1))
    return next(0)
```)

على الرغم من أن هاتين الدالتين مختلفتين هيكلياً للغاية، إلا أن الوصف الأكثر تجريداً للحسابين يكشف عن قدر كبير من التشابه. فالبرنامج الأول:

- يُعدّد أوراق الشجرة؛
- يُرشحها، مقتطفاً الفردية منها؛
- يربّع كل واحدة من الأوراق المختارة؛ و
- يجمّع النتائج باستخدام #py("+")، بدءاً من 0.

والبرنامج الثاني:

- يُعدّد الأعداد الصحيحة من 0 إلى $n$؛
- يحسب عدد فيبوناتشي لكل عدد صحيح؛
- يُرشحها، مقتطفاً الزوجية منها؛ و
- يجمّع النتائج باستخدام #py("pair")، بدءاً من القائمة المترابطة الفارغة.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-17.svg", width: 70%), caption: [تكشف خطط سريان الإشارة للدالتين #py("sum_odd_squares") (في الأعلى) و #py("even_fibs") (في الأسفل) عن القاسم المشترك بين البرنامجين.], label-name: <fig:signal-flow-plans>)

سيجد مهندس معالجة الإشارات أنه من الطبيعي تصوير هذه العمليات بدلالة إشارات تتدفق عبر سلسلة من المراحل (#idx("signal-processing view of computation") #idx("signal-flow diagram"))، حيث تنفذ كل مرحلة جزءاً من خطة البرنامج، كما هو موضح في
الشكل @fig:signal-flow-plans.
في
#py("sum_odd_squares")،
نبدأ بـ
#idx("enumerator")
#emph[مُعدِّد] (#en[enumerator])، والذي يولد "إشارة" تتكون من أوراق شجرة معطاة. وتُمَرَّر هذه الإشارة عبر
#idx("filter")
#emph[مرشح] (#en[filter])، والذي يستبعد العناصر باستثناء الفردية منها. وتُمرَّر الإشارة الناتجة بدورها عبر
#idx("mapping", sub: "as a transducer")
#emph[تطبيق خرائطي] (#en[map])، وهو "محول إشارة" (#en[transducer]) يطبق دالة #py("square") على كل عنصر. ثم تُمَدّ مخرجات الخريطة إلى
#idx("accumulator")
#emph[مُجمِّع] (#en[accumulator])، والذي يدمج العناصر باستخدام #py("+")، بدءاً من 0 كقيمة أولية. وخطة #py("even_fibs") مماثلة لذلك.

لسوء الحظ، يفشل تعريفا الدالتين أعلاه في إظهار بنية سريان الإشارة هذه. فعلى سبيل المثال، إذا فحصنا دالة #py("sum_odd_squares")، نجد أن التعداد ينفذ جزئياً عن طريق فحوصات #py("is_none") و #py("is_pair") وجزئياً عن طريق البنية التعاودية للشجرة. وبالمثل، نجد التجميع جزئياً في الفحوصات وجزئياً في الإضافة المستخدمة في التعاود. وبشكل عام، لا تظهر أجزاء متميزة في أي من الدالتين تتوافق مع العناصر الموجودة في وصف سريان الإشارة. إذ تفكك الدالتان الحسابات بطريقة مختلفة، مما يوزع التعداد عبر البرنامج ويخلطه مع الخريطة والمرشح والتجميع. وإذا كان بإمكاننا تنظيم برامجنا لجعل بنية سريان الإشارة ظاهرة في الدوال التي نكتبها، فإن هذا سيزيد من الوضوح المفاهيمي للبرنامج الناتج.

#subheading([عمليات المتتاليات])

#anchor(<sec:sequence-operations>)
#idx("sequence(s)", sub: "operations on")

إن المفتاح لتنظيم البرامج بحيث تعكس بنية سريان الإشارة بشكل أكثر وضوحاً هو التركيز على "الإشارات" التي تتدفق من مرحلة إلى أخرى في العملية. وإذا مثلنا هذه الإشارات كقوائم مترابطة، فيمكننا حينئذٍ استخدام عمليات القوائم المترابطة لتنفيذ المعالجة في كل مرحلة من المراحل. فعلى سبيل المثال، يمكننا تنفيذ مراحل التطبيق الخرائطي لمخططات سريان الإشارة باستخدام دالة #py("map") من القسم @sec:sequences:

#snippet(```python
print_llist(map(square, llist(1, 2, 3, 4, 5)))
```)

#output(```python
print_llist(map(square, llist(1, 2, 3, 4, 5)))
```)

ويتحقق ترشيح متتالية لاختيار العناصر التي تحقق محمولاً معيناً بوساطة:
#idx("filter", decl: true)
#snippet(```python
def filter(predicate, sequence):
    return (None if is_none(sequence)
            else pair(head(sequence),
                      filter(predicate, tail(sequence)))
            if predicate(head(sequence))
            else filter(predicate, tail(sequence)))
```)

على سبيل المثال،

#snippet(```python
print_llist(filter(is_odd, llist(1, 2, 3, 4, 5)))
```)

#output(```python
print_llist(filter(is_odd, llist(1, 2, 3, 4, 5)))
```)

ويمكن تنفيذ التجميعات بوساطة:
#idx("reduce", decl: true)
#snippet(```python
def reduce(op, initial, sequence):
    return (initial if is_none(sequence)
            else op(head(sequence),
                    reduce(op, initial, tail(sequence))))
```)

#snippet(```python
print(reduce(plus, 0, llist(1, 2, 3, 4, 5)))
```)

#output(```python
print(reduce(plus, 0, llist(1, 2, 3, 4, 5)))
```)

#snippet(```python
print(reduce(times, 1, llist(1, 2, 3, 4, 5)))
```)

#output(```python
print(reduce(times, 1, llist(1, 2, 3, 4, 5)))
```)

#snippet(```python
print_llist(reduce(pair, None, llist(1, 2, 3, 4, 5)))
```)

#output(```python
print_llist(reduce(pair, None, llist(1, 2, 3, 4, 5)))
```)

كل ما يتبقى لتنفيذ مخططات سريان الإشارة هو تعداد متتالية العناصر المراد معالجتها. بالنسبة لـ #py("even_fibs")، نحتاج إلى توليد متتالية الأعداد الصحيحة في نطاق معين، وهو ما يمكننا القيام به كما يلي:
#idx("enumerateinterval", decl: true)
#snippet(```python
def enumerate_interval(low, high):
    return (None if low > high
            else pair(low,
                      enumerate_interval(low + 1, high)))
```)

#snippet(```python
print_llist(enumerate_interval(2, 7))
```)

#output(```python
print_llist(enumerate_interval(2, 7))
```)

ولتعداد أوراق شجرة، يمكننا استخدام#footnote[هذه هي، في الواقع، دالة
#idx("fringe", sub: "as a tree enumeration")
#py("fringe")
بالظبط من التمرين @ex:fringe. وهنا أعدنا تسميتها للتأكيد على أنها جزء من عائلة من دوال التلاعب بالمتتاليات العامة.]
#idx("tree", sub: "enumerating leaves of")#idx("enumeratetree", decl: true)
#snippet(```python
def enumerate_tree(tree):
    return (None if is_none(tree)
            else llist(tree) if not is_pair(tree)
            else append(enumerate_tree(head(tree)),
                        enumerate_tree(tail(tree))))
```)

#snippet(```python
print_llist(enumerate_tree(llist(1, llist(2, llist(3, 4)), 5)))
```)

#output(```python
print_llist(enumerate_tree(llist(1, llist(2, llist(3, 4)), 5)))
```)

يمكننا الآن إعادة صياغة #py("sum_odd_squares") و #py("even_fibs") كما في مخططات سريان الإشارة. بالنسبة لـ #py("sum_odd_squares")، نُعدّد متتالية أوراق الشجرة، ونرشحها للإبقاء على الأعداد الفردية فقط في المتتالية، ونربّع كل عنصر، ونجمع النتائج:
#idx("sumoddsquares", decl: true)
#snippet(```python
def sum_odd_squares(tree):
    return reduce(plus,
                  0,
                  map(square,
                      filter(is_odd,
                             enumerate_tree(tree))))
```)

وبالنسبة لـ #py("even_fibs")، نُعدّد الأعداد الصحيحة من 0 إلى $n$، ونولد عدد فيبوناتشي لكل عدد من هذه الأعداد الصحيحة، ونرشح المتتالية الناتجة للإبقاء على العناصر الزوجية فقط، ونجمع النتائج في قائمة مترابطة:
#idx("evenfibs", decl: true)
#snippet(```python
def even_fibs(n):
    return reduce(pair,
                  None,
                  filter(is_even,
                         map(fib,
                             enumerate_interval(0, n))))
```)

قيمة التعبير عن البرامج كعمليات متتاليات هي أن هذا يساعدنا على جعل تصاميم البرامج نمطية (#en[modular])، أي تصاميم مبنية عن طريق تجميع أجزاء مستقلة نسبياً. ويمكننا تشجيع التصميم النمطي من خلال توفير مكتبة من المكونات القياسية جنباً إلى جنب مع واجهة تقليدية لربط المكونات بطرق مرنة.

البناء النمطي (#idx("modularity") #idx("sequence(s)", sub: "as source of modularity")) استراتيجية قوية للتحكم في التعقيد في التصميم الهندسي. ففي تطبيقات معالجة الإشارات الحقيقية، على سبيل المثال، يبني المصممون الأنظمة بانتظام من خلال تسلسل عناصر مختارة من عائلات موحدة من المرشحات ومحولات الإشارة. وبالمثل، توفر عمليات المتتاليات مكتبة من عناصر البرامج القياسية التي يمكننا خلطها والمطابقة بينها. فعلى سبيل المثال، يمكننا إعادة استخدام أجزاء من الدالتين #py("sum_odd_squares") و #py("even_fibs") في برنامج يبني قائمة مترابطة من مربعات أول $n+1$ من أعداد فيبوناتشي:

#snippet(```python
def list_fib_squares(n):
    return reduce(pair,
                  None,
                  map(square,
                      map(fib,
                          enumerate_interval(0, n))))
```)

#snippet(```python
print_llist(list_fib_squares(10))
```)

#output(```python
print_llist(list_fib_squares(10))
```)

ويمكننا إعادة ترتيب الأجزاء واستخدامها في حساب حاصل ضرب مربعات الأعداد الصحيحة الفردية في متتالية:

#snippet(```python
def product_of_squares_of_odd_elements(sequence):
    return reduce(times,
                  1,
                  map(square,
                      filter(is_odd, sequence)))
```)

#snippet(```python
print(product_of_squares_of_odd_elements(llist(1, 2, 3, 4, 5)))
```)

#output(```python
print(product_of_squares_of_odd_elements(llist(1, 2, 3, 4, 5)))
```)

ويمكننا أيضاً صياغة تطبيقات معالجة البيانات التقليدية بدلالة عمليات المتتاليات. نفترض أن لدينا متتالية من سجلات الموظفين ونريد العثور على راتب المبرمج الأعلى أجراً. لنفترض أن لدينا محدداً #py("salary") يرجع راتب السجل، ومحمولاً #py("is_programmer") يفحص ما إذا كان السجل لمبرمج. يمكننا حينئذٍ كتابة:

#snippet(```python
def salary_of_highest_paid_programmer(records):
    return reduce(max,
                  0,
                  map(salary,
                      filter(is_programmer, records)))
```)

تعطي هذه الأمثلة مجرد تلميح عن النطاق الواسع من العمليات التي يمكن التعبير عنها كعمليات متتاليات.#footnote[#idx("Waters, Richard C.")
طور #en[Richard Waters] عام (1979) برمجية تُحلل تلقائياً برامج #idx("Fortran") #en[Fortran] التقليدية، ناظرةً إليها بدلالة الخرائط والمرشحات والتجميعات. وقد وجد أن 90 بالمئة كاملة من الشفرة في حزمة البرامج الفرعية العلمية لـ #en[Fortran] تتناسب بدقة مع هذا النموذج. وأحد أسباب نجاح لغة #en[Lisp] كلغة برمجة هو أن القوائم المترابطة توفر وسيطاً قياسياً للتعبير عن المجموعات المرتبة بحيث يمكن التلاعب بها باستخدام دوال عليا. وقد تعلمت العديد من اللغات الحديثة، مثل #en[Python]، هذا الدرس.]

تؤدي المتتاليات، المنفذة هنا كقوائم مترابطة، دور الواجهة التقليدية التي تسمح لنا بدمج وحدات المعالجة. بالإضافة إلى ذلك، عندما نمثل البنى بشكل موحد كمتتاليات، نكون قد حصرنا اعتماديات بنية البيانات في برامجنا في عدد صغير من عمليات المتتاليات. ومن خلال تغيير هذه العمليات، يمكننا التجريب بتمثيلات بديلة للمتتاليات، مع ترك التصميم العام لبرامجنا سليماً. وسوف نستغل هذه القدرة في القسم @sec:streams، عندما نعمم نموذج معالجة المتتاليات ليقبل المتتاليات اللانهائية.

#exercise(label-name: <ex:2_33>, [
أكمل التعبيرات الناقصة لإتمام التعاريف التالية لبعض عمليات التلاعب الأساسية بالقوائم المترابطة كتجميعات:

#idx("length", sub: "as accumulation")#idx("map", sub: "as accumulation")#idx("append", sub: "as accumulation")
#syntax("
def map(f, sequence):
    return reduce(lambda x, y: ", metaphrase[??], ",
                      None, sequence)
def append(seq1, seq2):
    return reduce(pair, ", metaphrase[??], ", ", metaphrase[??], ")
def length(sequence):
    return reduce(", metaphrase[??], ", 0, sequence)
      ")
])

#exercise(label-name: <ex:horner>, [
يمكن صياغة تقييم حدودية في $x$ عند قيمة معطاة لـ $x$ كعملية تجميع.
نحن نقيم الحدودية

$ a_(n) x^(n) +a_(n-1)x^(n-1)+ dots.c + a_(1) x+a_(0) $

باستخدام خوارزمية معروفة تسمى
#idx("polynomial(s)", sub: "evaluating with Horner's rule")
#idx("Horner's rule")
#emph[قاعدة هورنر] (#en[Horner's rule])، والتي تبني الحساب كالتالي:

$ lr(( dots.c (a_(n) x+a_(n-1))x+ dots.c +a_(1) )) x+a_(0) $

وبعبارة أخرى، نبدأ بـ $a_(n)$، ونضرب في $x$، ونضيف $a_(n-1)$، ونضرب في $x$، وهكذا، حتى نصل إلى $a_(0)$.#footnote[وفقاً لـ
#idx("Knuth, Donald E.")
#en[Knuth] (1997b)، صيغت هذه القاعدة بواسطة
#idx("Horner, W. G.", sort: "Horner")
#en[W. G. Horner] في أوائل القرن التاسع عشر، ولكن هذه الطريقة استخدمها #en[Newton] بالفعل قبل أكثر من مائة عام من ذلك. تقيم قاعدة هورنر الحدودية باستخدام إضافات وعمليات ضرب أقل مما تفعله الطريقة المباشرة لحساب $a_(n) x^(n)$ أولاً، ثم إضافة $a_(n-1)x^(n-1)$، وهكذا. وفي الواقع، من الممكن إثبات أن أي خوارزمية لتقييم حدودية اعتباطية يجب أن تستخدم على الأقل نفس عدد الإضافات وعمليات الضرب التي تستخدمها قاعدة هورنر، وبالتالي فإن قاعدة هورنر هي خوارزمية
#idx("algorithm", sub: "optimal")
#idx("optimality", sub: "of Horner's rule")
مثلى لتقييم الحدوديات. أُثبت هذا (بالنسبة لعدد الإضافات) بواسطة
#idx("Ostrowski, A. M.")
#en[A. M. Ostrowski] في ورقة عام 1954 أسست بشكل أساسي للدراسة الحديثة للخوارزميات المثلى. وأُثبت التعبير المماثل لعمليات الضرب بواسطة
#idx("Pan, V. Y.")
#en[V. Y. Pan] في عام 1966. ويوفر كتاب
#idx("Borodin, Alan")
#idx("Munro, Ian")
#en[Borodin] و #en[Munro] (1975) نظرة عامة على هذه النتائج وغيرها حول الخوارزميات المثلى.]
أكمل القالب التالي لإنتاج دالة تقيم حدودية باستخدام قاعدة هورنر. افترض أن معاملات الحدودية مرتبة في متتالية، من $a_(0)$ إلى $a_(n)$.

#syntax("
def horner_eval(x, coefficient_sequence):
    return reduce(lambda this_coeff, higher_terms: ", metaphrase[??], ",
                  0,
                  coefficient_sequence)
      ")

فعلى سبيل المثال، لحساب $1+3x+5x^(3)+x^(5)$ عند $x=2$ فإنك تقيم:

#snippet(```python
print(horner_eval(2, llist(1, 3, 0, 5, 0, 1)))
```)
])

#exercise(label-name: <ex:countleaves-as-accumulation>, [
أعد تعريف #py("count_leaves") من القسم @sec:trees كعملية تجميع:
#idx("countleaves", sub: "as accumulation")
#syntax("
def count_leaves(t):
    return reduce(", metaphrase[??], ", ", metaphrase[??], ", map(", metaphrase[??], ", ", metaphrase[??], "))
          ")
])

#exercise(label-name: <ex:accumulate-n>, [
الدالة #py("reduce_n") مشابهة لـ #py("reduce") باستثناء أنها تأخذ كوسيط ثالث متتالية من المتتاليات، والتي يُفترض أن لها جميعاً عدد العناصر نفسه. وتطبق دالة التجميع المحددة لدمج كل العناصر الأولى في المتتاليات، وكل العناصر الثانية في المتتاليات، وهكذا، وترجع متتالية من النتائج. فعلى سبيل المثال، إذا كانت #py("s") متتالية تحتوي على أربع متتاليات:

#snippet(```python
llist(llist(1, 2, 3), llist(4, 5, 6), llist(7, 8, 9), llist(10, 11, 12))
```)

فإن قيمة #py("reduce_n(plus, 0, s)") ينبغي أن تكون المتتالية #py("llist(22, 26, 30)").
أكمل التعبيرات الناقصة في التعريف التالي لـ #py("reduce_n"):
#idx("reducen")
#syntax("
def reduce_n(op, init, seqs):
    return (None if is_none(head(seqs))
            else pair(reduce(op, init, ", metaphrase[??], "),
                      reduce_n(op, init, ", metaphrase[??], ")))
      ")
])

#exercise(label-name: <ex:matrix-ops>, [
نفترض أننا نمثل المتجهات $v=(v_(i))$ كمتتاليات أعداد (#idx("matrix, represented as sequence") #idx("vector (mathematical)", sub: "represented as sequence") #idx("vector (mathematical)", sub: "operations on"))، والمصفوفات $m=(m_(i j))$ كمتتاليات من المتجهات (صفوف المصفوفة). فعلى سبيل المثال، المصفوفة:

$ lr([ mat(delim: #none, 1, 2, 3, 4; 4, 5, 6, 6; 6, 7, 8, 9) ]) $

ممثلة كالمتتالية التالية:

#snippet(```python
llist(llist(1, 2, 3, 4),
      llist(4, 5, 6, 6),
      llist(6, 7, 8, 9))
```)

بهذا التمثيل، يمكننا استخدام عمليات المتتاليات للتعبير بإيجاز عن عمليات المصفوفات والمتجهات الأساسية. وهذه العمليات (الموصوفة في أي كتاب عن جبر المصفوفات) هي التالية:

#sicp-table(columns: 2, [#py("dot_product(")$v$#py(",")$w$#py(")")], [ترجع المجموع $sum_(i)v_(i) w_(i)$؛], [#py("matrix_times_vector(")$m$#py(",")$v$#py(")")], [ترجع المتجه $t$، حيث $t_(i) =sum_(j)m_(i j)v_(j)$؛], [#py("matrix_times_matrix(")$m$#py(",")$n$#py(")")], [ترجع المصفوفة $p$، حيث $p_(i j)=sum_(k) m_(i k)n_(k j)$؛], [#py("transpose(")$m$#py(")")], [ترجع المصفوفة $n$، حيث $n_(i j)=m_(j i)$.])

يمكننا تعريف الجداء السلمي كالتالي:#footnote[يستخدم هذا التعريف الدالة #py("reduce_n") من التمرين @ex:accumulate-n.]
#idx("dotproduct", decl: true)
#snippet(```python
def dot_product(v, w):
    return reduce(plus, 0, reduce_n(times, 1, llist(v, w)))
```)

أكمل التعبيرات الناقصة في الدوال التالية لحساب عمليات المصفوفة الأخرى. (الدالة #py("reduce_n") مُعلَن عنها في التمرين @ex:accumulate-n.)
#idx("matrixtimesvector")#idx("matrixtimesmatrix")#idx("transpose a matrix")
#syntax("
def matrix_times_vector(m, v):
    return map(", metaphrase[??], ", m)
def transpose(mat):
    return reduce_n(", metaphrase[??], ", ", metaphrase[??], ", mat)
def matrix_times_matrix(m, n):
    cols = transpose(n)
    return map(", metaphrase[??], ", m)
      ")
])

#exercise(label-name: <ex:fold-right-left>, [
تُعرِّف دالة #py("reduce") (#idx("reduce", sub: "same as foldright") #idx("foldright")) أيضاً باسم #py("fold_right")، لأنها تدمج العنصر الأول من المتتالية مع نتيجة دمج جميع العناصر على اليمين. وهناك أيضاً دالة #py("fold_left")، والتي تشبه #py("fold_right")، باستثناء أنها تدمج العناصر في الاتجاه المعاكس:
#idx("foldleft", decl: true)
#snippet(```python
def fold_left(op, initial, sequence):
    def iter(result, rest):
        return (result if is_none(rest)
                else iter(op(result, head(rest)),
                          tail(rest)))
    return iter(initial, sequence)
```)

ما هي قيم التعبيرات التالية:

#snippet(```python
print(fold_right(divide, 1, llist(1, 2, 3)))
```)

#snippet(```python
print(fold_left(divide, 1, llist(1, 2, 3)))
```)

#snippet(```python
print(fold_right(llist, None, llist(1, 2, 3)))
```)

#snippet(```python
print(fold_left(llist, None, llist(1, 2, 3)))
```)

أعطِ خاصية يجب أن تحققها #py("op") لضمان أن #py("fold_right") و #py("fold_left") ستنتجان القيم نفسها لأي متتالية.
])

#exercise(label-name: <ex:2_39>, [
أكمل التعاريف التالية لـ #py("reverse") (#idx("reverse", sub: "as folding")) (التمرين @ex:reverse) بدلالة #py("fold_right") و #py("fold_left") من التمرين @ex:fold-right-left:

#syntax("
def reverse(sequence):
    return fold_right(lambda x, y: ", metaphrase[??], ", None, sequence)
      ")

#syntax("
def reverse(sequence):
    return fold_left(lambda x, y: ", metaphrase[??], ", None, sequence)
      ")
])

#idx("sequence(s)", sub: "operations on")

#subheading([التطبيقات الخرائطية المتداخلة])

#anchor(<sec:nested-mappings>)
#idx("mapping", sub: "nested")

يمكننا توسيع نموذج المتتاليات ليشمل العديد من الحسابات التي يُعبر عنها شائعاً باستخدام الحلقات المتداخلة.#footnote[أُظهر لنا هذا النهج للتطبيقات الخرائطية المتداخلة بواسطة #idx("Turner, David") #en[David Turner]، الذي توفر لغتاه #idx("KRC") #en[KRC] و #idx("Miranda") #en[Miranda] أساليب صورية أنيقة للتعامل مع هذه البنى. والأمثلة في هذا القسم (انظر أيضاً التمرين @ex:8queens) مقتبسة من #en[Turner 1981]. وفي القسم @sec:exploiting-streams، سنرى كيف يتعمم هذا النهج على المتتاليات اللانهائية.]
تأمل هذه المشكلة: مع إعطاء عدد صحيح موجب $n$، أوجد جميع الأزواج المرتبة من الأعداد الصحيحة الموجبة المتميزة $i$ و $j$، حيث $1 lt.eq j < i lt.eq n$، بحيث يكون $i + j$ عدداً أولياً. على سبيل المثال، إذا كانت $n$ هي 6، فإن الأزواج هي التالية:

$ mat(delim: #none, i, 2, 3, 4, 4, 5, 6, 6; j, 1, 2, 1, 3, 2, 1, 5; i+j, 3, 5, 5, 7, 7, 7, 11) $

هناك طريقة طبيعية لتنظيم هذا الحساب وهي توليد متتالية جميع الأزواج المرتبة من الأعداد الصحيحة الموجبة الأقل من أو تساوي $n$، ثم الترشيح لاختيار تلك الأزواج التي يكون مجموعها أولياً، ثم لكل زوج $(i, j)$ يمر عبر المرشح، إنتاج الثلاثية $(i, j, i+j)$.

إليك طريقة لتوليد متتالية الأزواج: لكل عدد صحيح $i lt.eq n$، عَدّد الأعداد الصحيحة $j < i$، ولكل $i$ و $j$ من هذا القبيل، ولّد الزوج $(i, j)$. بدلالة عمليات المتتاليات، نطبق #py("map") على طول المتتالية #py("enumerate_interval(1, n)").
ولكل $i$ في هذه المتتالية، نطبق #py("map") على طول المتتالية #py("enumerate_interval(1, i - 1)").
ولكل $j$ في هذه المتتالية الأخيرة، نولد الزوج #py("llist(i, j)").
يعطينا هذا متتالية من الأزواج لكل $i$.
ودمج جميع المتتاليات لكل القيم $i$ (عن طريق التجميع مع #py("append")) ينشئ متتالية الأزواج المطلوبة:#footnote[نحن نمثل الزوج هنا كقائمة مترابطة ذات عنصرين بدلاً من زوج عادي. وبالتالي فإن "الزوج" $(i, j)$ ممثل ك#py("llist(i, j)")، وليس #py("pair(i, j)").]

#snippet(```python
print(reduce(append,
             None,
             map(lambda i: map(lambda j: llist(i, j),
                               enumerate_interval(1, i - 1)),
                 enumerate_interval(1, n))))
```)

إن دمج الخريطة والتجميع مع #py("append") أمر شائع جداً في هذا النوع من البرامج لدرجة أننا سنعزله كدالة منفصلة:
#idx("flatmap", decl: true)
#snippet(```python
def flatmap(f, seq):
    return reduce(append, None, map(f, seq))
```)

الآن رشح متتالية الأزواج هذه للعثور على الأزواج التي يكون مجموعها أولياً. يُستدعى محمول المرشِّح لكل عنصر في المتتالية؛ وسيطه زوج ويجب عليه استخراج الأعداد الصحيحة من الزوج. وبالتالي، فإن المحمول المراد تطبيقه على كل عنصر في المتتالية هو:

#snippet(```python
def is_prime_sum(pair):
    return is_prime(head(pair) + head(tail(pair)))
```)

أخيراً، ولّد متتالية النتائج عن طريق تطبيق الخريطة على الأزواج المرشحة باستخدام الدالة التالية، والتي تبني ثلاثية تتكون من عنصري الزوج جنباً إلى جنب مع مجموعهما:

#snippet(```python
def make_pair_sum(pair):
    return llist(head(pair), head(tail(pair)),
                 head(pair) + head(tail(pair)))
```)

يؤدي دمج كل هذه الخطوات إلى الدالة الكاملة:
#idx("primesumpairs", decl: true)
#snippet(```python
def prime_sum_pairs(n):
    return map(make_pair_sum,
               filter(is_prime_sum,
                  flatmap(lambda i: map(lambda j: llist(i, j),
                                        enumerate_interval(1, i - 1)),
                          enumerate_interval(1, n))))
```)

التطبيقات الخرائطية المتداخلة مفيدة أيضاً لمتتاليات غير تلك التي تعدّد النطاقات. نفترض أننا نرغب في توليد جميع تباديل (#idx("set", sub: "permutations of") #idx("permutations of a set")) مجموعة $S$؛ أي جميع طرق ترتيب العناصر في المجموعة. فعلى سبيل المثال، تباديل $\{1, 2, 3\}$ هي $\{1, 2, 3\}$، $\{ 1, 3, 2\}$، $\{2, 1, 3\}$، $\{ 2, 3, 1\}$، $\{ 3, 1, 2\}$، و $\{ 3, 2, 1\}$. إليك خطة لتوليد تباديل $S$: لكل عنصر $x$ في $S$، ولّد تعاودياً متتالية تباديل $S-x$،#footnote[المجموعة $S-x$ هي مجموعة جميع عناصر $S$ مع استبعاد $x$.] وألحق $x$ بمقدمة كل واحدة منها. يمنحنا هذا، لكل $x$ في $S$، متتالية تباديل $S$ التي تبدأ بـ $x$. يمنحنا دمج هذه المتتاليات لجميع القيم $x$ جميع تباديل $S$:#footnote[#idx("# (for comments in programs)", sort: "0a5") #idx("number sign (# for comments in programs)") #idx("comments in programs") #idx("program", sub: "comments in") يُستخدم الرمز #py("#") في برامج #en[Python] لإدخال #emph[التعليقات]. ويتم تجاهل كل شيء من #py("#") إلى نهاية السطر بوساطة المفسر. وفي هذا الكتاب لا نستخدم الكثير من التعليقات؛ ونحاول جعل برامجنا موثقة ذاتياً باستخدام أسماء توضيحية.]
#idx("permutations of a set", sub: "permutations", decl: true)

#snippet(```python
def permutations(s):
    return (llist(None)       # sequence containing empty set
            if is_none(s)     # empty set?
            else flatmap(lambda x: map(lambda p: pair(x, p),
                                       permutations(remove(x, s))),
                         s))
```)

لاحظ كيف يختزل هذا النهج مشكلة توليد تباديل $S$ إلى مشكلة توليد تباديل مجموعات تحتوي على عناصر أقل من $S$. وفي الحالة الطرفية، نصل إلى القائمة المترابطة الفارغة، والتي تمثل مجموعة بلا عناصر. ولهذا، نولد #py("llist(None)")، وهو متتالية بعنصر واحد، وهو المجموعة بلا عناصر. وترجع دالة #py("remove") المستخدمة في #py("permutations") جميع العناصر في متتالية معطاة باستثناء عنصر معطى. ويمكن التعبير عن هذا كمرشح بسيط:
#idx("remove", decl: true)
#snippet(```python
def remove(item, sequence):
    return filter(lambda x: x != item,
                  sequence)
```)

#exercise(label-name: <ex:2_40>, [
اكتب دالة
#idx("uniquepairs")
#py("unique_pairs")
تولد، عند إعطائها عدداً صحيحاً $n$، متتالية الأزواج $(i, j)$ حيث $1 lt.eq j < i lt.eq n$. استخدم #py("unique_pairs") لتبسيط تعريف #py("prime_sum_pairs") المعطى أعلاه.
])

#exercise(label-name: <ex:2_41>, [
اكتب دالة للعثور على جميع الثلاثيات المرتبة من الأعداد الصحيحة الموجبة المتميزة $i$ و $j$ و $k$ الأقل من أو تساوي عدداً صحيحاً معطى $n$ والتي مجموعها يساوي عدداً صحيحاً معطى $s$.
])

#sicp-figure(image("/images/img_original/ch2-Z-G-23.svg", width: 70%), caption: [حل للغز الملكات الثماني.], label-name: <fig:8queens>)

#exercise(label-name: <ex:8queens>, [
يسأل "لغز الملكات الثماني" (#idx("eight-queens puzzle") #idx("chess, eight-queens puzzle") #idx("puzzles", sub: "eight-queens puzzle")) عن كيفية وضع ثماني ملكات على رقعة شطرنج بحيث لا تقع أي ملكة تحت التهديد من أي ملكة أخرى (أي لا توجد ملكتان في الصف أو العمود أو القطر نفسه). يُعرض أحد الحلول الممكنة في الشكل @fig:8queens. إحدى طرق حل اللغز هي العمل عبر الرقعة، بوضع ملكة في كل عمود.
وبمجرد وضع $k-1$ من الملكات، يجب وضع الملكة $k$ في موضع لا تهدد فيه أي من الملكات الموجودة بالفعل على الرقعة. ويمكننا صياغة هذا النهج تعاودياً: نفترض أننا ولدنا بالفعل متتالية جميع الطرق الممكنة لوضع $k-1$ من الملكات في أول $k-1$ من أعمدة الرقعة. ولكل من هذه الطرق، ولّد مجموعة موسعة من المواضع عن طريق وضع ملكة في كل صف من العمود $k$. والآن رشح هذه المواضع، مقتطفاً المواضع التي تكون فيها الملكة في العمود $k$ آمنة بالنسبة للملكات الأخرى. ينتج عن ذلك متتالية جميع الطرق لوضع $k$ من الملكات في أول $k$ من الأعمدة. وبالاستمرار في هذه العملية، لن ننتج حلاً واحداً فحسب، بل جميع الحلول للغز.

ننفذ هذا الحل كدالة #py("queens")، والتي ترجع متتالية جميع الحلول لمشكلة وضع $n$ من الملكات على رقعة شطرنج $n times n$.
تمتلك الدالة #py("queens") دالة داخلية #py("queens_cols") ترجع متتالية جميع الطرق لوضع الملكات في أول $k$ من أعمدة الرقعة.

#idx("queens", decl: true)
#snippet(```python
def queens(board_size):
    def queen_cols(k):
        return (llist(empty_board) if k == 0
                else filter(lambda positions: is_safe(k, positions),
                            flatmap(lambda rest_of_queens:
                                      map(lambda new_row:
                                            adjoin_position(new_row, k,
                                                            rest_of_queens),
                                          enumerate_interval(1, board_size)),
                                    queen_cols(k - 1))))
    return queen_cols(board_size)
```)

في هذه الدالة، #py("rest_of_queens") هي طريقة لوضع $k-1$ من الملكات في أول $k-1$ من الأعمدة، و #py("new_row") هو صف مقترح لوضع الملكة فيه للعمود $k$. أكمل البرنامج بتنفيذ التمثيل لمجموعات مواضع الرقعة، بما في ذلك الدالة #py("adjoin_position")، التي تلحق موضع صف-عمود جديداً بمجموعة مواضع، و #py("empty_board")، التي تمثل مجموعة مواضع فارغة. وعليك أيضاً كتابة الدالة #py("is_safe")، التي تحدد لمجموعة مواضع ما إذا كانت الملكة في العمود $k$ آمنة بالنسبة للملكات الأخرى.
(لاحظ أننا نحتاج فقط إلى الفحص عما إذا كانت الملكة الجديدة آمنة—فالملكات الأخرى مضمون بالفعل أنها آمنة بالنسبة لبعضها البعض.)
])

#exercise(label-name: <ex:2_43>, [
يواجه #en[Louis Reasoner] وقتاً عصيباً في أداء التمرين @ex:8queens. تبدو دالته #py("queens") تعمل، لكنها تعمل ببطء شديد. (لا ينجح #en[Louis] أبداً في الانتظار لفترة كافية لحل حتى حالة $6 times 6$). وعندما يطلب #en[Louis] المساعدة من #en[Eva Lu Ator]، تشير إلى أنه قد بادَل ترتيب التطبيقات الخرائطية المتداخلة في #py("flatmap")، كاتباً إياها كالتالي:

#snippet(```python
flatmap(lambda new_row:
          map(lambda rest_of_queens:
                adjoin_position(new_row, k, rest_of_queens),
              queen_cols(k - 1)),
        enumerate_interval(1, board_size))
```)

اشرح سبب إبطاء هذه المبادلة للبرنامج. قدِّر كم سيستغرق برنامج #en[Louis] لحل لغز الملكات الثماني، بافتراض أن البرنامج في التمرين @ex:8queens يحل اللغز في زمن قدره $T$.
])

#idx("sequence(s)", sub: "as conventional interface")
#idx("conventional interface", sub: "sequence as")
#idx("mapping", sub: "nested")
