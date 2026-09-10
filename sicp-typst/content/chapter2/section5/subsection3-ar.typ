// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([مثال: الجبر الرمزي], label-name: <sec:symbolic-algebra>)

#idx("symbolic algebra")
#idx("algebra", sub: "symbolic")

تُعد معالجة التعبيرات الجبرية الرمزية عملية معقدة توضح العديد من القضايا المتعلقة بتجريد البيانات. وتتراوح الأنظمة المصممة لمعالجة التعبيرات الجبرية من المعالجات القائمة على القواعد البسيطة، إلى المجمعات القائمة على تبسيط التعبيرات الخاصة، وصولاً إلى الأنظمة العامة المعقدة للغاية. وفي هذا القسم ننظر إلى سياق محدود ولكنه مهم لحسابيات متعددي الحدود (#en[polynomials]). وتلك مهمة حسابية شائعة، وتعتمد الخوارزميات المتقدمة في الجبر الحاسوبي بشكل مكثف على معالجة متعددي الحدود.

#idx("polynomial arithmetic")
#idx("arithmetic", sub: "of polynomials")

سنقوم ببناء نظام يؤدي العمليات الحسابية على متعددي الحدود. وهدفنا الأساسي هو توضيح كيفية دمج متعددي الحدود في نظامنا الحسابي العام، مع الحفاظ على التجريد البياناتي السليم.

=== حسابيات متعددي الحدود

#idx("polynomial arithmetic", sub: "definition of polynomial")
#idx("polynomial(s)", sub: "definition of")

تُعرّف حدودية متعددي الحدود (#en[polynomial]) عادةً بدلالة متغير غير محدد (أو متغير مستقل - #idx("indeterminate") #emph[indeterminate]) يُرمز له بالرمز $x$. وتُكتب الحدودية كـ مجموع من الحدود (#en[terms]):
$ a_n x^n + a_(n-1) x^(n-1) + ... + a_1 x + a_0 $
حيث $a_n, a_(n-1), ..., a_0$ هي المعاملات (#idx("coefficient") #emph[coefficients])، و $n$ هي الدرجة أو الرتبة (#emph[order])، و $x^k$ هي القوى المتتالية للمتغير.

في هذا القسم سننظر فقط في #idx("univariate polynomial") #idx("polynomial(s)", sub: "univariate") #emph[متعددي الحدود أحادي المتغير] (#en[univariate polynomials])، أي تلك التي تحتوي على متغير مستقل واحد.

لتنفيذ الحسابيات على متعددي الحدود، يجب أن نتفق على التجريد الذي يمثل حدودية ما. الحدودية تنتمي إلى نوع بيانات معين ويكون لها متغير محدد وقائمة من الحدود. وسوف نستخدم الوسم #py("polynomial") لتمثيل هذا النوع.

نعرف المنشئات والمحددات لمتعددي الحدود كما يلي:

#snippet(```python
def make_polynomial(variable, term_list):
    return contents(apply_generic("make_polynomial",
                                  llist(variable, term_list)))

def variable(p):
    return apply_generic("variable", llist(p))

def term_list(p):
    return apply_generic("term_list", llist(p))
```)

وسوف نستخدم الرموز لتمثيل المتغيرات، مثل الرمز #py("\"x\""). ونستخدم الدالة #py("is_same_variable") للتحقق مما إذا كان متغيران متطابقين:

#snippet(```python
def is_same_variable(v1, v2):
    return is_variable(v1) and is_variable(v2) and v1 == v2

def is_variable(x):
    return is_string(x)
```)

يمكننا إضافة حدوديتين إذا كانتا في المتغير نفسه، وذلك بجمع حدودهما المناظرة. وإذا كانت الحدوديتان في متغيرين مختلفين، يمكننا إما إطلاق خطأ أو تحويلهما إلى متغير مشترك إذا كانت هناك أولوية محددة بين المتغيرات.

نعرف الدالة العامية #py("add_poly") والجمع الكلي لمتعددي الحدود:

#snippet(```python
def add_poly(p1, p2):
    if is_same_variable(variable(p1), variable(p2)):
        return make_polynomial(variable(p1),
                               add_terms(term_list(p1),
                                         term_list(p2)))
    else:
        error("Polys not in same var -- ADD_POLY", llist(p1, p2))

def mul_poly(p1, p2):
    if is_same_variable(variable(p1), variable(p2)):
        return make_polynomial(variable(p1),
                               mul_terms(term_list(p1),
                                         term_list(p2)))
    else:
        error("Polys not in same var -- MUL_POLY", llist(p1, p2))
```)

نثبت هذه العمليات في النظام العام:

#snippet(```python
def install_polynomial_package():
    # Internal procedures
    def make_poly(variable, term_list):
        return llist(variable, term_list)
    def var(p): return head(p)
    def tag_term_list(p): return head(tail(p))

    def add_poly(p1, p2):
        if is_same_variable(var(p1), var(p2)):
            return make_poly(var(p1),
                             add_terms(tag_term_list(p1),
                                       tag_term_list(p2)))
        else:
            error("Polys not in same var -- ADD_POLY", llist(p1, p2))

    def mul_poly(p1, p2):
        if is_same_variable(var(p1), var(p2)):
            return make_poly(var(p1),
                             mul_terms(tag_term_list(p1),
                                       tag_term_list(p2)))
        else:
            error("Polys not in same var -- MUL_POLY", llist(p1, p2))

    # Interface to rest of the system
    def tag(p): return attach_tag("polynomial", p)
    put("add", llist("polynomial", "polynomial"),
        lambda p1, p2: tag(add_poly(p1, p2)))
    put("mul", llist("polynomial", "polynomial"),
        lambda p1, p2: tag(mul_poly(p1, p2)))
    put("make_polynomial", "polynomial",
        lambda variable, term_list: tag(make_poly(variable, term_list)))
    return "done"
```)

=== تمثيل قوائم الحدود

الآن نحتاج إلى تنفيذ العمليات #py("add_terms") و #py("mul_terms") على قوائم الحدود (#en[term lists]). تدمج #py("add_terms") قائمتين من الحدود لتشكيل قائمة حدود حاصل الجمع، وتضرب #py("mul_terms") كل حد من القائمة الأولى في كل حد من القائمة الثانية وتجمع النتائج.

#idx("term list")
#idx("polynomial arithmetic", sub: "term lists")

نعرف إجراءات إنشاء واستخراج الحدود:

#snippet(```python
def make_term(order, coeff):
    return llist(order, coeff)

def order(term):
    return head(term)

def coeff(term):
    return head(tail(term))
```)

تأخذ #py("add_terms") قائمتين مرتبتين من الحدود وتدمجهما:

#snippet(```python
def add_terms(L1, L2):
    if is_empty_termlist(L1):
        return L2
    elif is_empty_termlist(L2):
        return L1
    else:
        t1 = first_term(L1)
        t2 = first_term(L2)
        if order(t1) > order(t2):
            return adjoin_term(t1, add_terms(rest_terms(L1), L2))
        elif order(t1) < order(t2):
            return adjoin_term(t2, add_terms(L1, rest_terms(L2)))
        else:
            return adjoin_term(make_term(order(t1),
                                           add(coeff(t1), coeff(t2))),
                               add_terms(rest_terms(L1),
                                         rest_terms(L2)))
```)

لاحظ أننا نستخدم العمليات الحسابية العامة #py("add") لجمع المعاملات. هذا يتيح لنظامنا معالجة متعددي حدود مع معاملات يمكن أن تكون أعداداً عادية، أو أعداداً كسرية، أو أعداداً مركبة، أو حتى متعددي حدود آخرين!

ولضرب قائمتين من الحدود:

#snippet(```python
def mul_terms(L1, L2):
    if is_empty_termlist(L1):
        return the_empty_termlist()
    else:
        return add_terms(mul_term_by_all_terms(first_term(L1), L2),
                         mul_terms(rest_terms(L1), L2))

def mul_term_by_all_terms(t1, L):
    if is_empty_termlist(L):
        return the_empty_termlist()
    else:
        t2 = first_term(L)
        return adjoin_term(
            make_term(order(t1) + order(t2), mul(coeff(t1), coeff(t2))),
            mul_term_by_all_terms(t1, rest_terms(L)))
```)

لاحظ استخدام #py("mul") لضرب المعاملات و #py("+") لجمع الرتب.

==== قوائم الحدود الكثيفة والضئيلة

#idx("dense polynomial")
#idx("sparse polynomial")
#idx("polynomial(s)", sub: "dense vs. sparse")

هناك طريقتان رئيسيتان لتمثيل قوائم الحدود:
1. #emph[الكثيفة] (#en[dense]): تمثيل قوائم الحدود بقائمة من المعاملات فقط مرتبة حسب الرتبة تنازلياً.
2. #emph[الضئيلة] (#en[sparse]): تمثيل قوائم الحدود بقائمة من الأزواج `(order, coeff)` فقط للحدود غير الصفرية.

على سبيل المثال، الحدودية $x^5 + 2x + 1$:
- التمثيل الضئيل: `[(5, 1), (1, 2), (0, 1)]`
- التمثيل الكثيف: `[1, 0, 0, 0, 2, 1]`

تكون قوائم الحدود الضئيلة أكثر كفاءة بالنسبة لمتعددي الحدود ذات الدرجات العالية التي تحتوي على العديد من المعاملات الصفرية، مثل $x^(100) + 1$.

نعرف محددات ومنشئات قائمة الحدود الضئيلة كالتالي:

#snippet(```python
def adjoin_term(term, term_list):
    if is_equal(coeff(term), 0):
        return term_list
    else:
        return pair(term, term_list)

def the_empty_termlist(): return nil
def first_term(term_list): return head(term_list)
def rest_terms(term_list): return tail(term_list)
def is_empty_termlist(term_list): return is_null(term_list)
```)

#exercise(label-name: <ex:poly-negation>, [
#idx("negation of polynomials")
عَرِّف الطرح لمتعددي الحدود بدلالة جمع المتناظرات السلبية لمتعددي الحدود. أضف العملية العامة #py("sub") لمتعددي الحدود إلى الحزمة.
])

#exercise(label-name: <ex:poly-zero>, [
#idx("zero test for polynomials")
عَرِّف فحص الصفر العام #py("is_zero") لمتعددي الحدود.
])

#exercise(label-name: <ex:dense-sparse-poly>, [
صمِّم واجهة لتمثيل قوائم الحدود الكثيفة والضئيلة بحيث يمكن استخدام كليهما في حزمة متعددي الحدود ذاتها.
])

=== قسمة متعددي الحدود

#idx("polynomial arithmetic", sub: "division")
#idx("division", sub: "of polynomials")

يمكننا قسمة حدودية على أخرى لإنتاج حدودية ناتج القسمة وحزمة باقي القسمة. الخوارزمية هي خوارزمية القسمة المطولة التقليدية لمتعددي الحدود:

#snippet(```python
def div_terms(L1, L2):
    if is_empty_termlist(L1):
        return llist(the_empty_termlist(), the_empty_termlist())
    else:
        t1 = first_term(L1)
        t2 = first_term(L2)
        if order(t1) < order(t2):
            return llist(the_empty_termlist(), L1)
        else:
            new_c = div(coeff(t1), coeff(t2))
            new_o = order(t1) - order(t2)
            rest_of_result = div_terms(
                sub_terms(L1,
                          mul_term_by_all_terms(make_term(new_o, new_c),
                                                L2)),
                L2)
            return llist(adjoin_term(make_term(new_o, new_c),
                                     head(rest_of_result)),
                         head(tail(rest_of_result)))
```)

تأخذ #py("div_terms") قائمتين من الحدود وتُرجع قائمة تحتوي على قائمة حدود ناتج القسمة وقائمة حدود باقي القسمة.

#exercise(label-name: <ex:poly-div>, [
استخدم #py("div_terms") لتحديد #py("div_poly") والعملية العامة #py("div") لمتعددي الحدود.
])

=== القاسم المشترك الأعظم لمتعددي الحدود

#idx("polynomial arithmetic", sub: "greatest common divisor")
#idx("greatest common divisor", sub: "polynomials")

يمكننا حساب القاسم المشترك الأعظم (#en[GCD]) لمتعددي حدود باستخدام خوارزمية إقليدس المناظرة للأعداد الصحيحة:

#snippet(```python
def gcd_terms(a, b):
    if is_empty_termlist(b):
        return a
    else:
        return gcd_terms(b, remainder_terms(a, b))
```)

حيث #py("remainder_terms") تُرجع باقي قسمة #py("a") على #py("b").

#exercise(label-name: <ex:remainder-terms>, [
باستخدام #py("div_terms")، نفّذ الدالة #idx("remainderterms")#py("remainder_terms") واستخدمها لتعريف #py("gcd_terms") كما هو موضح أعلاه. ثم اكتب دالة #idx("greatest common divisor", sub: "generic")#py("gcd_poly") تحسب القاسم المشترك الأعظم لمتعددي حدود. واثبّت عملية عامة #py("greatest_common_divisor") تؤول إلى #py("gcd_poly") بالنسبة لمتعددي الحدود وإلى #py("gcd") العادي بالنسبة للأعداد. واختبر برنامجك على:

#snippet(```python
p1 = make_polynomial("x", llist(make_term(4, 1), make_term(3, -1),
                                make_term(2, -2), make_term(1, 2)))
p2 = make_polynomial("x", llist(make_term(3, 1), make_term(1, -1)))
greatest_common_divisor(p1, p2)
```)
تحقق من النتيجة يدوياً.
])

#exercise(label-name: <ex:gcd-of-polys>, [
لتكن $P_1$ و $P_2$ و $P_3$ متعددي الحدود التالية:

#sicp-table(columns: 2, [$P_1$:], [$x^2 - 2x + 1$], [$P_2$:], [$11x^2 + 7$], [$P_3$:], [$13x + 5$])

الآن لتكن $Q_1$ حاصل ضرب $P_1$ و $P_2$، ولتكن $Q_2$ حاصل ضرب $P_1$ و $P_3$. استخدم #py("greatest_common_divisor") (التمرين @ex:remainder-terms) لحساب GCD لـ $Q_1$ و $Q_2$. لاحظ أن النتيجة ليست مساوية لـ $P_1$. يُدخل هذا المثال عمليات غير صحيحة (كسرية) في الحسابات، مما يسبب صعوبات لخوارزمية GCD.#footnote[في Python، قد تؤدي قسمة الأعداد الصحيحة إلى أعداد عشرية ذات دقة محدودة، وبالتالي قد نفشل في الحصول على مقسوم عليه صحيح.]
لِفهم ما يحدث، جرب تتبع #py("gcd_terms") أثناء حساب GCD أو حاول إجراء القسمة يدوياً.
])

يمكننا حل المشكلة المعروضة في التمرين @ex:gcd-of-polys باستخدام التعديل التالي على خوارزمية GCD (والذي يعمل حقاً فقط في حالة متعددي الحدود ذات المعاملات الصحيحة).
قبل إجراء أي قسمة متعددي حدود في حساب GCD، نضرب المقسوم في عامل ثابت صحيح، يُختار لضمان عدم ظهور أي كسور أثناء عملية القسمة. وبالتالي فإن إجابتنا ستختلف عن GCD الفعلي بمقدار عامل ثابت صحيح، ولكن هذا لا يهم في حالة اختزال الدوال الكسرية إلى أدن طواحينها؛ حيث سيُستخدم GCD لقسمة البسط والمقام معاً، وبالتالي سيلتغي العامل الثابت الصحيح.

وبشكل أكثر دقة، إذا كان $P$ و $Q$ متعددي حدود، لتكن $O_1$ رتبة $P$ (أي رتبة أكبر حد في $P$) ولتكن $O_2$ رتبة $Q$. وليكن $c$ المعامل الرئيسي لـ $Q$. فيمكن إثبات أنه إذا ضربنا $P$ في
#idx("integerizing factor")
#emph[معامل المعالجة الصحيحة]
$c^(1+O_1 - O_2)$، فإن الحدودية الناتجة يمكن قسمتها على $Q$ باستخدام الخوارزمية
#py("div_terms")
دون إدخال أي كسور. وتُسمى عملية ضرب المقسوم في هذا الثابت ثم القسمة أحياناً بـ
#idx("pseudodivision of polynomials")
#emph[القسمة الزائفة] لـ $P$ على $Q$. ويسمى باقي القسمة بـ
#idx("pseudoremainder of polynomials")
#emph[الباقي الزائف].

#exercise(label-name: <ex:pseudoremainder-terms>, [
+ نفّذ الدالة #py("pseudoremainder_terms")، والتي تشبه #py("remainder_terms") تماماً باستثناء أنها تضرب المقسوم في معامل المعالجة الصحيحة الموصوف أعلاه قبل استدعاء #py("div_terms"). عدّل #py("gcd_terms") لاستخدام #py("pseudoremainder_terms")، وتحقق من أن #py("greatest_common_divisor") ينتج الآن إجابة بمعاملات صحيحة في المثال الموجود في التمرين @ex:gcd-of-polys.
+ أصبح لـ GCD الآن معاملات صحيحة، ولكنها أكبر من معاملات $P_1$. عدّل #py("gcd_terms") بحيث يزيل العوامل المشتركة من معاملات الإجابة عن طريق قسمة جميع المعاملات على القاسم المشترك الأعظم (الصحيح) لها.
])

#idx("polynomial arithmetic", sub: "greatest common divisor")
#idx("rational function", sub: "reducing to lowest terms")
#idx("reducing to lowest terms")

وهكذا، إليك كيفية اختزال دالة كسرية إلى أدنى حدودها:

- احسب GCD للبسط والمقام، باستخدام نسخة #py("gcd_terms") من التمرين @ex:pseudoremainder-terms.
- عند الحصول على GCD، اضرب كلاً من البسط والمقام بنفس معامل المعالجة الصحيحة قبل القسمة على GCD، حتى لا تؤدي القسمة على GCD إلى إدخال أي معاملات غير صحيحة. وكعامل يمكنك استخدام المعامل الرئيسي لـ GCD مرفوعاً للقوة $1+O_1 - O_2$، حيث $O_2$ هي رتبة GCD و $O_1$ هي القيمة العظمى لرتبتي البسط والمقام. سيضمن هذا أن قسمة البسط والمقام على GCD لن تدخل أي كسور.
- ستكون نتيجة هذه العملية بسطاً ومقاماً بمعاملات صحيحة. ستكون المعاملات عادة ضخمة جداً بسبب كل عوامل المعالجة الصحيحة، لذا فإن الخطوة الأخيرة هي إزالة العوامل الزائدة عن طريق حساب القاسم المشترك الأعظم (الصحيح) لجميع معاملات البسط والمقام والقسمة على هذا العامل.

#exercise(label-name: <ex:reduce-poly>, [
+ نفّذ هذه الخوارزمية كدالة #py("reduce_terms") تأخذ قائمتي حدود #py("n") و #py("d") كمعاملات وتُرجع قائمة مرتبطة #py("nn"), #py("dd") تمثل #py("n") و #py("d") مختزلتين إلى أدنى الحدود عبر الخوارزمية الموضحة أعلاه. واكتب أيضاً دالة #py("reduce_poly")، ممثالة لـ #py("add_poly")، تتحقق مما إذا كان لمتعددي الحدود المتغير نفسه. إذا كان الأمر كذلك، تُجرد #py("reduce_poly") المتغير وتمرر المسألة إلى #py("reduce_terms")، ثم تعيد إرفاق المتغير بقائمتي الحدود المقدمتين من #py("reduce_terms").
+ عرّف دالة مماثلة لـ #py("reduce_terms") تقوم بما كانت تفعله #py("make_rat") الأصلية للأعداد الصحيحة: #snippet(```python def reduce_integers(n, d): g = gcd(n, d) return llist(n // g, d // g) ```) وعرّف #py("reduce") كعملية عامة تستدعي #py("apply_generic") للتوجيه إما إلى #py("reduce_poly") (مع معاملات من النوع #py("polynomial")) أو إلى #py("reduce_integers") (مع معاملات من النوع #py("python_number")). يمكنك الآن بسهولة جعل حزمة الحسابيات الكسرية تختزل الكسور إلى أدنى حدودها من خلال جعل #py("make_rat") تستدعي #py("reduce") قبل دمج البسط والمقام المعطيين لتشكيل عدد كسرية. يتعامل النظام الآن مع التعبيرات الكسرية إما في الأعداد الصحيحة أو متعددي الحدود. لاختبار برنامجك، جرب المثال في بداية هذا التمرين الممتد: #snippet(```python p1 = make_polynomial("x", llist(make_term(1, 1), make_term(0, 1))) p2 = make_polynomial("x", llist(make_term(3, 1), make_term(0, -1))) p3 = make_polynomial("x", llist(make_term(1, 1))) p4 = make_polynomial("x", llist(make_term(2, 1), make_term(0, -1))) rf1 = make_rational(p1, p2) rf2 = make_rational(p3, p4) add(rf1, rf2) ```) انظر ما إذا كنت ستحصل على الإجابة الصحيحة مختزلة بشكل صحيح إلى أدنى حدودها.
])

تُعد حسابات GCD في قلب أي نظام يجري عمليات على الدوال الكسرية. والتقنية المستخدمة أعلاه، على الرغم من بساطتها الرياضياتية، إلا أنها بطيئة للغاية. ويرجع البطء جزئياً إلى العدد الكبير من عمليات القسمة وجزئياً إلى الحجم الهائل للمعاملات الوسيطة الناتجة عن القسمات الزائفة.
#idx("rational function", sub: "reducing to lowest terms")
#idx("reducing to lowest terms")
تعد خوارزميات حساب GCD لمتعددي الحدود من المجالات النشطة في تطوير أنظمة المعالجة الجبرية.#footnote[اكتشف ريتشارد زيبيل (#idx("Zippel, Richard E.") #en[Richard Zippel], 1979) طريقة فائقة الكفاءة والأناقة لحساب
#idx("polynomial arithmetic", sub: "greatest common divisor")
#idx("polynomial arithmetic", sub: "probabilistic algorithm for GCD")
#idx("probabilistic algorithm")
#idx("algorithm", sub: "probabilistic")
GCD لمتعددي الحدود. وتعد هذه الطريقة خوارزمية احتمالية (#en[probabilistic algorithm])، مثل الفحص السريع للأولية الذي ناقشناه في الفصل @chap:fun. ويصف كتاب زيبيل (1993) هذه الطريقة إلى جانب طرق أخرى لحساب GCD لمتعددي الحدود.]
#idx("rational function")
#idx("function (mathematical)", sub: "rational")
#idx("polynomial arithmetic", sub: "rational functions")
#idx("symbolic algebra")
#idx("polynomial(s)")
#idx("polynomial arithmetic")
