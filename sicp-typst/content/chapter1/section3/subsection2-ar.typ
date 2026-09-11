// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([إنشاء الدوال باستخدام تعبيرات لامبدا], label-name: <sec:lambda>)

في استخدام #py("sum") كما في
القسم @sec:procedures-as-parameters، يبدو من المربك للغاية اضطرارنا لتعريف دوال بسيطة مثل #py("pi_term") و #py("pi_next") لمجرد استخدامها كمعطيات لدالتنا ذات الرتبة الأعلى. وبدلاً من تعريف #py("pi_next") و #py("pi_term")، سيكون من الأكثر ملاءمة إيجاد طريقة لتحديد «الدالة التي ترجع مدخلها مضافاً إليه 4» و «الدالة التي ترجع مقلوب مدخلها مضروباً في مدخلها زائد 2» مباشرة. ويمكننا القيام بذلك بتقديم #emph[تعبير لامبدا] كشكل نحوي لإنشاء الدوال.
باستخدام تعبيرات لامبدا، يمكننا وصف ما نريده كـ:

#snippet(```python
lambda x: x + 4
```)

و

#snippet(```python
lambda x: 1 / (x * (x + 2))
```)

عندئذ يمكننا التعبير عن دالة #py("pi_sum") دون تعريف أي دوال مساعدة:
#idx("pisum", sub: "with lambda expression", decl: true)
#snippet(```python
def pi_sum(a, b):
    return sum(lambda x: 1 / (x * (x + 2)),
               a,
               lambda x: x + 4,
               b)
```)

ومرة أخرى باستخدام تعبير لامبدا، يمكننا كتابة دالة #py("integral") دون الحاجة لتعريف الدالة المساعدة #py("add_dx"):

#idx("integral", sub: "with lambda expression", decl: true)
#snippet(```python
def integral(f, a, b, dx):
    return sum(f,
               a + dx / 2,
               lambda x: x + dx,
               b) * dx
```)

وبوجه عام، تُستخدَم تعبيرات لامبدا لإنشاء الدوال بنفس طريقة تعاريف الدوال،

#idx("lambda expression")
#idx("syntactic forms", sub: "lambda expression")
#idx("lambda expression", sub: "function definition vs.")
#idx("function definition", sub: "lambda expression vs.")

باستثناء أنه لا يُحدَّد اسم للدالة وتُحذَف الأقواس حول المعلمات وكلمة #py("return") المفتاحية.

#snippet(```python
lambda parameters: expression
```)

والدالة الناتجة هي دالة تماماً مثل تلك التي تُنشَأ باستخدام تعليمة تعريف الدالة.
#idx("function definition", sub: "lambda expression vs.")
#idx("lambda expression", sub: "function definition vs.")
والفرق الوحيد هو أنها لم تُربَط بأي اسم في البيئة.

وفي الواقع،

#snippet(```python
plus4 = lambda x: x + 4
```)

مكافئ لـ
#idx("lambda expression", sub: "function definition vs.")
#idx("function definition", sub: "lambda expression vs.")

#snippet(```python
def plus4(x):
    return x + 4
```)

يمكننا قراءة تعبير لامبدا كما يلي:

$ mat(delim: #none, mono(bold("lambda") "x"), mono(":"), mono("x"), mono("+"), mono("4"); arrow.t, arrow.t, arrow.t, arrow.t, arrow.t; mono("الدالة ذات المعطى" "x"), "التي تنتج", "القيمة", "زائد", "4.") $

ومثل أي تعبير له دالة
#idx("lambda expression", sub: "as function expression of application") #idx("function expression", sub: "lambda expression as")
كقيمة له، يمكن استخدام تعبير لامبدا كتعبير دالة في تطبيق مثل:

#snippet(```python
print((lambda x, y, z: x + y + square(z))(1, 2, 3))
```)

#output(```python
print((lambda x, y, z: x + y + square(z))(1, 2, 3))
```)

أو، بشكل أكثر عمومية، في أي سياق نستخدم فيه عادة اسم دالة.#footnote[كان سيكون أكثر وضوحاً وأقل إرهاباً للأشخاص الذين يتعلمون بايثون لو استُخدِم مصطلح أكثر وضوحاً من #emph[تعبير لامبدا]، مثل #emph[تعبير تعريف الدالة]. لكن العرف راسخ للغاية، ليس فقط لـ #en[Lisp] و #en[Scheme] بل أيضاً لـ #en[Python] و #en[Java] ولغات أخرى، ويرجع ذلك بلا شك جزئياً إلى تأثير طبعات #en[Scheme] من هذا الكتاب.
#idx("Scheme", sub: "use of lambda in")
وقد أُخِذ هذا الترميز من
#idx("λ calculus (lambda calculus)", sort: "0l")
#idx("λ calculus (lambda calculus)", sort: "lambda")
حساب $lambda$، وهو شكل رياضياتي قدمه المنطقي الرياضياتي
#idx("Church, Alonzo")
ألونزو تشيرش #en[(Alonzo Church)] (1941). طور تشيرش حساب $lambda$ لتوفير أساس صارم لدراسة مفاهيم الدالة وتطبيق الدالة. وأصبح حساب $lambda$ أداة أساسية للتحقيقات الرياضياتية في دلالات لغات البرمجة.]
لاحظ أنّ لتعبير #py("lambda") #idx("precedence", sub: "of lambda expression") #idx("lambda expression", sub: "precedence of") أسبقية أقل من تطبيق الدالة، وبالتالي فإنّ الأقواس #idx("parentheses", sub: "around lambda expression") حول تعبير لامبدا ضرورية هنا.

#subheading([استخدام إسناد التعيين لإنشاء متغيرات محلية])

#idx("local name")

استخدام آخر لـ #py("lambda") هو إنشاء متغيرات محلية.
غالباً ما نحتاج إلى متغيرات محلية في دوالنا غير تلك التي رُبِطَت كمعلمات.
على سبيل المثال، افترض أننا نود حساب الدالة:

$ mat(delim: #none, f(x, y), =, x(1 + x y)^(2) +y (1 - y) + (1 + x y)(1 - y)) $

والتي يمكننا التعبير عنها أيضاً كـ:

$ mat(delim: #none, a, =, 1+x y; b, =, 1-y; f(x, y), =, x a^(2) +y b + a b) $

عند كتابة دالة لحساب $f$، نودّ أن نضمّن كمتغيرات محلية ليس فقط $x$ و $y$ بل أيضاً أسماء الكميات الوسيطة مثل $a$ و $b$. وإحدى الطرق لتحقيق ذلك هي استخدام دالة مساعدة لربط المتغيرات المحلية:

#snippet(```python
def f(x, y):
    def f_helper(a, b):
        return x * square(a) + y * b + a * b
    return f_helper(1 + x * y, 1 - y)
```)

وبالطبع، كان بإمكاننا استخدام تعبير #py("lambda") لتحديد دالة مجهولة لربط متغيراتنا المحلية. وعندئذ يصبح جسم #py("f") استدعاءً واحداً لتلك الدالة:

#snippet(```python
def f_2(x, y):
    return (lambda a, b:
            x * square(a) + y * b + a * b)(1 + x * y, 1 - y)
```)

وهناك طريقة أكثر ملاءمة للتصريح عن المتغيرات المحلية وهي استخدام إسناد التعيين داخل جسم الدالة.
باستخدام إسناد التعيين، يمكن كتابة الدالة كـ:

#snippet(```python
def f_3(x, y):
    a = 1 + x * y
    b = 1 - y
    return x * square(a) + y * b + a * b
```)

المتغيرات التي يُصرَّح عنها بإسناد التعيين داخل دالة يكون جسم الدالة المحيطة مباشرة هو نطاقها.#footnote[#idx("declaration", sub: "use of name before")
لاحظ أنّ الاسم المصرَّح عنه في دالة لا يمكن استخدامه قبل تقييم التصريح بالكامل، بغض النظر عما إذا كان الاسم نفسه مصرَّحاً عنه خارج الدالة. وبالتالي في البرنامج أدناه، فإنّ محاولة استخدام #py("a") المصرَّح عنها في المستوى الأعلى لتوفير قيمة لحساب #py("b") المصرَّح عنها في #py("f") لا يمكن أن تعمل:

#snippet(```python
a = 1
def f(x):
    b = a + x
    a = 5
    return a + b
f(10)
```)

يؤدي البرنامج إلى خطأ، لأنّ #py("a") في #py("a + x") مُستخدَمة قبل تقييم تصريحها. وسنعود إلى هذا البرنامج في القسم @sec:internal-definitions (التمرين @ex:simultaneous-def)، بعد أن نتعلّم المزيد عن التقييم.]<foot:tdz>
$""^(,)$
#footnote[يمكن توسيع نموذج الاستبدال للقول إنه بالنسبة لإسناد التعيين، تُستبدَل قيمة التعبير بعد #py("=") بالاسم قبل #py("=") في بقية جسم الدالة (بعد التصريح)، بشكل مشابه لاستبدال الوسائط بالمعلمات في تقييم تطبيق الدالة.]
#idx("local name")

#subheading([التعليمات الشرطية])

لقد رأينا أنه من المفيد غالباً التصريح عن متغيرات محلية لتعاريف الدوال. وعندما تصبح الدوال كبيرة، يجب علينا إبقاء الحساب المرتبط بالمتغيرات مقيداً قدر الإمكان. تأمَّل على سبيل المثال #py("expmod") في التمرين @ex:louis-fast-prime:

#snippet(```python
def expmod(base, exp, m):
    return (1 if exp == 0
            else (expmod(base, exp // 2, m)
                  * expmod(base, exp // 2, m)) % m if is_even(exp)
            else (base * expmod(base, exp - 1, m)) % m)
```)

هذه الدالة غير كفؤة بلا داعٍ، لأنها تحتوي على استدعاءين متطابقين:

#snippet(```python
expmod(base, exp // 2, m)
```)

وفي حين يمكن إصلاح ذلك بسهولة في هذا المثال باستخدام دالة #py("square")، فإنّ هذا ليس سهلاً بوجه عام. ودون استخدام #py("square")، سنميل إلى تقديم اسم محلي للتعبير كما يلي:

#snippet(```python
def expmod(base, exp, m):
    half_exp = expmod(base, exp // 2, m)
    return (1 if exp == 0
            else (half_exp * half_exp) % m if is_even(exp)
            else (base * expmod(base, exp - 1, m)) % m)
```)

#idx("conditional statement", sub: "need for")
هذا من شأنه أن يجعل الدالة ليست فقط غير كفؤة، بل وغير منتهية بالفعل! والمشكلة هي أنّ إسناد التعيين يظهر خارج التعبير الشرطي، مما يعني أنه يُنفَّذ حتى عندما يتحقق حالة الأساس #py("exp == 0").
ولتجنب هذا الوضع، نوفر
#idx("conditional statement")
#idx("syntactic forms", sub: "conditional statement")
#idx("if (keyword)", sort: "if")
#idx("else (keyword)", sort: "else")
#idx("keywords", sub: "if")
#idx("keywords", sub: "else")
#idx("predicate", sub: "of conditional statement")
#idx("conditional statement", sub: "predicate, consequent, and alternative of")
#emph[التعليمات الشرطية]، ونسمح لتعليمات الإرجاع بالظهور في فروع التعليمة. وباستخدام تعليمة شرطية، يمكننا كتابة دالة #py("expmod") كما يلي:

#snippet(```python
def expmod(base, exp, m):
    if exp == 0:
        return 1
    else:
        if is_even(exp):
            half_exp = expmod(base, exp // 2, m)
            return (half_exp * half_exp) % m
        else:
            return (base * expmod(base, exp - 1, m)) % m
```)

أبسط شكل للتعليمة الشرطية هو:

#snippet(```python
if predicate:
    consequent-statements
else:
    alternative-statements
```)

كما هو الحال بالنسبة للتعبير الشرطي، يقيّم المُفسِّر أولاً المحمول. وإذا قُيِّم إلى صحيح (True)، يقيّم المُفسِّر
#idx("consequent", sub: "of conditional statement")
#idx("conditional statement", sub: "consequent statements of")
تعليمات النتيجة بالتتابع، وإذا قُيِّم إلى خاطئ (False)، يقيّم المُفسِّر
#idx("alternative", sub: "of conditional statement")
#idx("conditional statement", sub: "alternative statements of")
تعليمات البديل بالتتابع. وتقييم تعليمة الإرجاع يرجع من الدالة المحيطة، متجاهلاً أي تعليمات في المتتالية
#idx("sequence of statements", sub: "in conditional statement")
بعد تعليمة الإرجاع وأي تعليمات بعد التعليمة الشرطية.
#idx("conditional statement")

توفر بايثون الكلمة المفتاحية #py("elif") لتجنب تعليمات #py("else: if") المتداخلة بعمق. وباستخدام #py("elif")، يمكننا كتابة دالة #py("expmod") كما يلي:

#snippet(```python
def expmod(base, exp, m):
    if exp == 0:
        return 1
    elif is_even(exp):
        half_exp = expmod(base, exp // 2, m)
        return (half_exp * half_exp) % m
    else:
        return (base * expmod(base, exp - 1, m)) % m
```)

#exercise(label-name: <ex:1_34>, [
افترض أننا عرفنا الدالة:

#snippet(```python
def f(g):
    return g(2)
```)

عندئذ يكون لدينا:

#snippet(```python
print(f(square))
```)

#output(```python
print(f(square))
```)

#snippet(```python
print(f(lambda z: z * (z + 1)))
```)

#output(```python
print(f(lambda z: z * (z + 1)))
```)

ماذا يحدث إذا طلبنا (على سبيل العناد) من المُفسِّر تقييم التطبيق #py("f(f)")؟ اشرح ذلك.
])
