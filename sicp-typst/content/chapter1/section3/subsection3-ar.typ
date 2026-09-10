// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([الدوال كـطرق عامة], label-name: <sec:proc-general-methods>)

#idx("higher-order functions", sub: "function as general method")

قدّمنا الدوال المركبة في القسم @sec:compound-procedures كآلية لتجريد أنماط العمليات العددية لجعلها مستقلة عن الأعداد المحددة المعنية. ومع الدوال من رتبة أعلى، مثل دالة #py("integral") في القسم @sec:procedures-as-parameters، بدأنا نرى نوعاً أكثر قوة من التجريد: دوال تُستخدَم للتعبير عن طرق عامة للحساب، مستقلة عن الدوال المحددة المعنية. وفي هذا القسم نناقش مثالين أكثر إتقاناً — طرقاً عامة لإيجاد أصفار النقاط الثابتة للدوال — ونوضح كيف يمكن التعبير عن هذه الطرق مباشرة كدوال.

#subheading([إيجاد جذور المعادلات بطريقة تنصيف الفترات])

#idx("half-interval method")
#emph[طريقة تنصيف الفترات] هي تقنية بسيطة ولكنها قوية لإيجاد جذور معادلة $f(x)=0$، حيث $f$ دالة مستمرة. والفكرة هي أنه، إذا أُعطِينَا نقطتين $a$ و $b$ بحيث $f(a) < 0 < f(b)$، فيجب أن يكون لـ $f$ صفر واحد على الأقل بين $a$ و $b$. وتحديد موقع الصفر، لتكن $x$ متوسط $a$ و $b$ ونحسب $f(x)$. وإذا كان $f(x) > 0$، فإنه يجب أن يكون لـ $f$ صفر بين $a$ و $x$. وإذا كان $f(x) < 0$، فإنه يجب أن يكون لـ $f$ صفر بين $x$ و $b$. وبالمتابعة بهذه الطريقة، يمكننا تحديد فترات أصغر فأصغر يجب أن يكون لـ $f$ صفر فيها. وعندما نصل إلى نقطة تكون فيها الفترة صغيرة بدرجة كافية، تنتهي العملية. وبما أنّ فترة عدم اليقين تُنصَّف في كل خطوة من العملية، فإنّ أقصى عدد من الخطوات المطلوبة ينمو كـ $Theta ( log ( L/T))$، حيث $L$ هو طول الفترة الأصلية و $T$ هو التسامح مع الخطأ (أي حجم الفترة التي سنعتبرها «صغيرة بدرجة كافية»).
والدالة التالية تنفذ هذه الاستراتيجية:

#idx("search", decl: true)
#snippet(```python
def search(f, neg_point, pos_point):
    midpoint = average(neg_point, pos_point)
    if close_enough(neg_point, pos_point):
        return midpoint
    else:
        test_value = f(midpoint)
        if positive(test_value):
            return search(f, neg_point, midpoint)
        elif negative(test_value):
            return search(f, midpoint, pos_point)
        else: return midpoint
```)

نفترض أننا أُعطِينَا في البداية الدالة $f$ جنباً إلى جنب مع نقاط تكون قيمها عندها سالبة وموجبة. ونحسب أولاً نقطة المنتصف للنقطتين المعطاتين. ثم نتحقق مما إذا كانت الفترة المعطاة صغيرة بدرجة كافية، وإذا كان الأمر كذلك فنحن نرجع نقطة المنتصف كإجابتنا. وخلاف ذلك، نحسب كقيمة اختبار قيمة $f$ عند نقطة المنتصف. وإذا كانت قيمة الاختبار موجبة، نواصل العملية بفترة جديدة تبدأ من النقطة السالبة الأصلية إلى نقطة المنتصف. وإذا كانت قيمة الاختبار سالبة، نواصل بالفترة من نقطة المنتصف إلى النقطة الموجبة. وأخيراً، هناك احتمال أن تكون قيمة الاختبار 0، وفي هذه الحالة تكون نقطة المنتصف نفسها هي الجذر الذي نبحث عنه.

وللاختبار ما إذا كانت النقطتان النهايتان «قريبتين بدرجة كافية»، يمكننا استخدام دالة مشابِهة لتلك المستخدمة في القسم @sec:sqrt لحساب الجذور التربيعية:#footnote[استخدمنا 0.001 كعدد «صغير» نموذجي للإشارة إلى تسامح خطأ مقبول في الحساب. والتسامح المناسب للحساب الحقيقي يعتمد على المشكلة المراد حلها وقيود الحاسوب والخوارزمية. وغالباً ما يكون هذا اعتباراً دقيقاً للغاية، يتطلب مساعدة من
#idx("numerical analyst")
محلل عددي أو ساحر آخر.]

#snippet(```python
def close_enough(x, y):
    return abs(x - y) < 0.001
```)

الدالة #py("search") غير ملاءمة للاستخدام مباشرة، لأننا قد نعطيها بطريق الخطأ نقاطاً لا تحتوي قيم $f$ عندها على الإشارة المطلوبة، وفي هذه الحالة نحصل على إجابة خاطئة. وبدلاً من ذلك سنستخدم #py("search") عبر الدالة التالية، التي تتحقق من أي من النقطتين النهايتين تحتوي على قيمة دالة سالبة وأيها تحتوي على قيمة موجبة، وتستدعي الدالة #py("search") وفقاً لذلك. وإذا كان للدالة نفس الإشارة على النقطتين المعطاتين، فلا يمكن استخدام طريقة تنصيف الفترات، وفي هذه الحالة تعطي الدالة إشارة إلى وجود خطأ.#footnote[يمكن تحقيق ذلك باستخدام
#idx("error (primitive function)")
#py("error")،
والتي تأخذ كمعطى سلسلة نصية تُطبَع كرسالة خطأ جنباً إلى جنب مع رقم سطر البرنامج الذي أدى إلى استدعاء #py("error").]

#snippet(```python
def half_interval_method(f, a, b):
    a_value = f(a)
    b_value = f(b)
    if negative(a_value) and positive(b_value):
        return search(f, a, b)
    elif negative(b_value) and positive(a_value):
        return search(f, b, a)
    else: error("values are not of opposite sign")
```)

يستخدم المثال التالي
#idx("half-interval method", sub: "halfintervalmethod")
#idx("π (pi)", sub: "approximation with half-interval method", sort: "pi")
طريقة تنصيف الفترات لتقريب $pi$ كجذر بين 2 و 4 لـ $sin thin x = 0$:

#snippet(```python
print(half_interval_method(math_sin, 2, 4))
```)

#output(```python
print(half_interval_method(math_sin, 2, 4))
```)

وهناك مثال آخر، يطبق طريقة تنصيف الفترات للبحث عن جذر للمعادلة $x^(3) - 2x - 3 = 0$ بين 1 و 2:

#snippet(```python
print(half_interval_method(lambda x: x * x * x - 2 * x - 3, 1, 2))
```)

#output(```python
print(half_interval_method(lambda x: x * x * x - 2 * x - 3, 1, 2))
```)

#idx("half-interval method")

#subheading([إيجاد النقاط الثابتة للدوال])

يُسمّى العدد $x$
#idx("fixed point")
#idx("function (mathematical)", sub: "fixed point of")
#emph[نقطة ثابِتة] لدالة $f$ إذا حقق $x$ المعادلة $f(x)=x$. وبالنسبة لبعض الدوال $f$ يمكننا تحديد موقع نقطة ثابِتة بالبدء بتخمين أولي وتطبيق $f$ مراراً وتكراراً:

$ f(x), space f(f(x)), space f(f(f(x))), space dots.h $

حتى لا تتغير القيمة كثيراً. وباستخدام هذه الفكرة، يمكننا ابتكار دالة #py("fixed_point") تأخذ كمدخلات دالة وتخميناً أولياً وتنتج تقريباً لنقطة ثابِتة للدالة. ونطبق الدالة مراراً وتكراراً حتى نجد قيمتين متتاليتين يكون الفرق بينهما أقل من تسامح محدد:

#idx("fixedpoint", decl: true)
#snippet(```python
tolerance = 0.00001
def fixed_point(f, first_guess):
    def close_enough(x, y):
        return abs(x - y) < tolerance
    def try_with(guess):
        next = f(guess)
        return next if close_enough(guess, next) else try_with(next)
    return try_with(first_guess)
```)

على سبيل المثال، يمكننا استخدام هذه الطريقة لتقريب النقطة الثابتة لـ
#idx("fixed point", sub: "of cosine")
#idx("cosine", sub: "fixed point of")
#idx("mathcos (primitive function)")
دالة جيب التمام، بدءاً من 1 كتخمين أولي:#footnote[للحصول على
#idx("fixed point", sub: "computing with calculator")
#idx("calculator, fixed points with")
نقطة ثابتة لـ جيب التمام على آلة حاسبة، اضبطها على وضع الراديان ثم اضغط مراراً وتكراراً على زر $cos$ حتى لا تتغير القيمة بعد الآن.]

#snippet(```python
print(fixed_point(math_cos, 1))
```)

#output(```python
print(fixed_point(math_cos, 1))
```)

وبالمثل، يمكننا إيجاد حل للمعادلة $y= sin y + cos y$:
#idx("mathsin (primitive function)")

#snippet(```python
print(fixed_point(lambda y: math_sin(y) + math_cos(y), 1))
```)

#output(```python
print(fixed_point(lambda y: math_sin(y) + math_cos(y), 1))
```)

تُذكِّرنا عملية النقطة الثابتة بالعملية التي استخدمناها لإيجاد الجذور التربيعية في القسم @sec:sqrt. فكلاهما تعتمدان على فكرة تحسين التخمين مراراً وتكراراً حتى تحقق النتيجة معياراً ما. وفي الواقع، يمكننا بسهولة صياغة
#idx("fixed point", sub: "square root as")
حساب الجذر التربيعي كبحث عن نقطة ثابِتة. يتطلب حساب الجذر التربيعي لعدد $x$ إيجاد $y$ بحيث $y^(2) = x$. وبوضع هذه المعادلة في الشكل المكافئ $y = x/y$، ندرك أننا نبحث عن نقطة ثابِتة للدالة#footnote[$arrow.r.bar$
#idx("↦ notation for mathematical function", sort: "0a2")
#idx("function (mathematical)", sub: "↦ notation for")
(تُقرَأ «تنتقل إلى») هي طريقة الرياضياتيين لكتابة تعبيرات لامبدا. $y arrow.r.bar x/y$ تعني #py("lambda y: x / y")، أي الدالة التي قيمتها عند $y$ هي $x/y$.]
$y arrow.r.bar x/y$، وبالتالي يمكننا محاولة حساب الجذور التربيعية كـ:
#idx("sqrt", sub: "as fixed point", decl: true)
#snippet(```python
def sqrt(x):
    return fixed_point(lambda y: x / y, 1)
```)

ولسوء الحظ، فإنّ هذا البحث عن النقطة الثابتة لا يتقارب. تأمَّل تخميناً أولياً $y_(1)$. التخمين التالي هو $y_(2) = x/y_(1)$ والتخمين التالي هو $y_(3) = x/y_(2) = x/(x/y_(1)) = y_(1)$. وهذا يؤدي إلى حلقة لا نهائية يتكرر فيها التخمينان $y_(1)$ و $y_(2)$ مراراً وتكراراً، متجهين بالذبذبة حول الإجابة.

وإحدى الطرق للتحكم في مثل هذه الذبذبات هي منع التخمينات من التغير كثيراً. وبما أنّ الإجابة تقع دائماً بين تخميننا $y$ و $x/y$، يمكننا إنشاء تخمين جديد ليس بعيداً عن $y$ مثل $x/y$ بأخذ متوسط $y$ مع $x/y$، بحيث يكون التخمين التالي بعد $y$ هو $frac(1, 2)(y+x/y)$ بدلاً من $x/y$. وعملية إنشاء مثل هذه المتتالية من التخمينات هي ببساطة عملية البحث عن نقطة ثابِتة لـ $y arrow.r.bar frac(1, 2)(y+x/y)$:

#snippet(```python
def sqrt(x):
    return fixed_point(lambda y: average(y, x / y), 1)
```)

(لاحظ أنّ $y=frac(1, 2)(y+x/y)$ هي تحويل بسيط للمعادلة $y=x/y$؛ ولإشتقاقها، أضف $y$ إلى كلا الطرفين واقسم على 2).

مع هذا التعديل، تعمل دالة الجذر التربيعي. وفي الواقع، إذا فككنا التعاريف، يمكننا أن نرى أنّ متتالية التقريبات للجذر التربيعي المنشأة هنا هي بالضبط نفس تلك المنشأة بواسطة دالة الجذر التربيعي الأصلية في القسم @sec:sqrt. وهذه المقاربة لأخذ متوسط التقريبات المتتالية لحل ما، وهي تقنية نسميها
#idx("average damping")
#emph[إخماد المتوسط (#en[average damping])]، غالباً ما تساعد على تقارب البحث عن النقاط الثابتة.
#idx("fixed point")
#idx("function (mathematical)", sub: "fixed point of")

#exercise(label-name: <ex:1_35>, [
أظهر أنّ النسبة الذهبية $phi$
#idx("golden ratio", sub: "as fixed point")
#idx("fixed point", sub: "golden ratio as")
(القسم @sec:tree-recursion) هي نقطة ثابِتة للتحويل $x arrow.r.bar 1 + 1/x$، واستخدم هذه الحقيقة لحساب $phi$ بوساطة دالة #py("fixed_point").
])

#exercise(label-name: <ex:log-fixed-point>, [
عدل #py("fixed_point") بحيث تطبع متتالية التقريبات التي تُنشئها، باستخدام الدالة الأولية #py("print").
ثم أوجد حلاً لـ $x^(x) = 1000$ بإيجاد نقطة ثابِتة لـ $x arrow.r.bar log (1000)/ log (x)$.
#idx("mathlog (primitive function)") (استخدم الدالة الأولية #py("math_log")، التي تحسب اللوغاريتمات الطبيعية).
وقارن عدد الخطوات التي يتطلبها ذلك مع إخماد المتوسط وبدونه.
(لاحظ أنه لا يمكنك بدء #py("fixed_point") بتخمين 1، لأن هذا سيتسبب في القسمة على $log (1)=0$).
])

#exercise(label-name: <ex:continued-fractions>, [
الكسر المستمر المالانهاية
#idx("continued fraction")
#emph[كسر مستمر] هو تعبير من الشكل:

$ mat(delim: #none, f, =, (frac(N_(1), D_(1)+ frac(N_(2), D_(2)+ frac(N_(3), D_(3)+ dots.c ))))) $

كمثال، يمكن إثبات أنّ تفكيك الكسر المستمر المالانهاية الذي تكون فيه $N_(i)$ و $D_(i)$ كلها مساوية لـ 1 ينتج $1/ phi$، حيث $phi$ هي
#idx("continued fraction", sub: "golden ratio as")
#idx("golden ratio", sub: "as continued fraction")
النسبة الذهبية (الموصوفة في القسم @sec:tree-recursion). وإحدى الطرق لتقريب كسر مستمر مالانهاية هي بتتر قيم التفكيك بعد عدد معطى من الحدود. ومثل هذا البتر — المسمى #emph[كسر مستمر منتهي من $k$ من الحدود] — له الشكل:

$ (frac(N_(1), D_(1) + frac(N_(2), dots.down + frac(N_(K), D_(K))))) $

+ افترض أنّ #py("n") و #py("d") دالتان بمعطى واحد (مؤشر الحد $i$) ترجعان $N_(i)$ و $D_(i)$ لحدود الكسر المستمر. عرف دالة #py("cont_frac") بحيث يقيّم #py("cont_frac(n, d, k)") قيمة الكسر المستمر المنتهي ذي الـ $k$ من الحدود. وافحص دالتك بتقريب $1/ phi$ باستخدام #snippet(```python print(cont_frac(lambda i: 1, lambda i: 1, k)) ```) لقيم متتالية لـ #py("k"). كم يجب أن تجعل قيمة #py("k") للحصول على تقريب دقيق حتى 4 منازل عشرية؟
+ إذا كانت دالة #py("cont_frac") لديك تُنشئ عملية عَوْدِيَّة، فاكتب دالة تُنشئ عملية تكرارية. وإذا كانت تُنشئ عملية تكرارية، فاكتب دالة تُنشئ عملية عَوْدِيَّة.
])

#exercise(label-name: <ex:1_38>, [
في عام 1737، نشر عالم الرياضيات السويسري
#idx("Euler, Leonhard")
ليونارد أيلر #en[(Leonhard Euler)] مذكرات
#emph[De Fractionibus Continuis]، والتي تضمنت تفكيك كسر مستمر لـ $e-2$، حيث $e$ هي أساس اللوغاريتمات الطبيعية. وفي هذا الكسر، تكون $N_(i)$ كلها 1، وتكون $D_(i)$ بالتتابع 1، 2، 1، 1، 4، 1، 1، 6، 1، 1، 8، …. اكتب برنامجاً يستخدِم دالة #py("cont_frac") لديك من التمرين @ex:continued-fractions لتقريب $e$، بناءً على تفكيك أيلر.
])

#exercise(label-name: <ex:1_39>, [
نُشِر تمثيل كسر مستمر لدالة الظل في عام 1770 بواسطة عالم الرياضيات الألماني
#idx("Lambert, J.H.")
#idx("continued fraction", sub: "tangent as")
#idx("tangent", sub: "as continued fraction")
جيه. إتش. لامبرت #en[(J.H. Lambert)]:

$ mat(delim: #none, tan x, =, (frac(x, 1- frac(x^(2), 3- frac(x^(2), 5- frac(x^(2), dots.down )))))) $

حيث $x$ بالراديان.
عرف دالة #py("tan_cf(x, k)") تحسب تقريباً لدالة الظل بناءً على صيغة لامبرت.
كما في التمرين @ex:continued-fractions، يحدد #py("k") عدد الحدود المراد حسابها.
])

#idx("higher-order functions", sub: "function as general method")
