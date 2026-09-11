// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([الدوال كتجريدات صندوق أسود], label-name: <sec:black-box>)

الدالة #py("sqrt") هي أول مثال لدينا على عملية مُعرَّفة بمجموعة من الدوال
المُعرَّفة بشكل متبادل. لاحظ أنّ تعريف #py("sqrt_iter")
#idx("recursive function", sub: "recursive function definition")
#emph[عودي]؛ أي أنّ الدالة مُعرَّفة بدلالة ذاتها. قد تكون فكرة القدرة على
تعريف دالة بدلالة ذاتها مقلقة؛ وقد يبدو غير واضح كيف يمكن لمثل هذا التعريف
«الدائري» أن يكون منطقياً أصلاً، ناهيك عن تحديد عملية محدّدة جيداً ينفّذها
حاسوب. سنتناول هذا بعناية أكبر في القسم @sec:procedures-and-processes. لكن
أولاً لنتأمّل بعض النقاط المهمة الأخرى التي يوضّحها مثال #py("sqrt").

لاحظ أنّ مشكلة حساب الجذور التربيعية تنقسم طبيعياً إلى عدد من المشكلات الفرعية:
#idx("program", sub: "structure of")
كيف نحدّد ما إذا كان التخمين جيداً كفاية، وكيف نحسّن التخمين، وهكذا. كل مهمة
من هذه المهام تُنجَز بدالة منفصلة. يمكن النظر إلى برنامج #py("sqrt") بأكمله
كتجمّع من الدوال (كما في الشكل @fig:sqrt-decomposition_new) يعكس تحليل المشكلة
إلى مشكلات فرعية.

#sicp-figure-ar(image("/images/img_javascript/ch1-Z-G-6.svg", width: 70%), caption: [التحليل الدالّي لبرنامج #py("sqrt").], label-name: <fig:sqrt-decomposition_new>)

أهمية استراتيجية
#idx("decomposition of program into parts")
التحليل هذه ليست ببساطة أننا نقسّم البرنامج إلى أجزاء. ففي النهاية، يمكننا أخذ
أي برنامج كبير وتقسيمه إلى أجزاء — أول عشرة أسطر، ثم العشرة التالية، وهكذا.
بل الحاسم هو أنّ كل دالة تُنجز مهمة محدّدة يمكن استخدامها كوحدة في تعريف
دوال أخرى.

مثلاً، حين نُعرّف الدالة #py("is_good_enough") بدلالة #py("square")، نستطيع
اعتبار الدالة #py("square")
#idx("black box")
«صندوقاً أسود». لا نهتم في تلك اللحظة بـ #emph[كيف] تحسب الدالة نتيجتها، بل
فقط بـ #emph[أنها] تحسب المربع. يمكن إخفاء تفاصيل كيفية حساب المربع لتُعالَج
في وقت لاحق. في الواقع، بالنسبة لدالة #py("is_good_enough")، ليست #py("square")
دالة تماماً بل تجريد لدالة، ما يُسمّى
#idx("functional abstraction") #idx("abstraction", sub: "functional") #emph[تجريداً دالّياً].
عند هذا المستوى من التجريد، أي دالة تحسب المربع تكون جيدة بنفس القدر.

وهكذا، بالنظر فقط إلى القيم التي تُرجعها، يجب أن تكون الدالتان التاليتان
لتربيع عدد غير قابلتين للتمييز. كل منهما تأخذ وسيطاً عددياً وتُنتج مربع ذلك
العدد كقيمة.#footnote[ليس واضحاً حتى أيهما تنفيذ أكفأ. هذا يعتمد على العتاد
المتاح. هناك آلات يكون فيها التنفيذ «البديهي» هو الأقل كفاءة. تأمّل آلة لديها
جداول واسعة من اللوغاريتمات ومضاداتها مخزّنة بطريقة فعّالة جداً.]

#snippet(```python
def square(x): return x * x
```)

#snippet(```python
def square(x):
    return math_exp(double(math_log(x)))

def double(x): return x + x
```)

إذاً يجب أن تكون الدالة قادرة على إخفاء التفاصيل. مستخدمو الدالة قد لا يكونون
كتبوها بأنفسهم بل حصلوا عليها من مبرمج آخر كصندوق أسود. لا ينبغي أن يحتاج
المستخدم لمعرفة كيفية تنفيذ الدالة لاستخدامها.

#subheading([الأسماء المحلية])

#idx("local name")

تفصيل من تنفيذ الدالة لا ينبغي أن يهمّ مستخدم الدالة هو اختيار المنفّذ لأسماء
معلمات الدالة. وهكذا يجب ألا تكون الدالتان التاليتان قابلتين للتمييز:

#snippet(```python
def square(x): return x * x
```)

#snippet(```python
def square(y): return y * y
```)

هذا المبدأ — أنّ معنى الدالة يجب أن يكون مستقلاً عن أسماء المعلمات التي
استخدمها مؤلفها — يبدو على السطح بديهياً، لكن عواقبه عميقة. أبسط عاقبة هي أنّ
أسماء معلمات الدالة يجب أن تكون محلية لجسم الدالة. مثلاً، استخدمنا #py("square")
في تعريف #py("is_good_enough") في دالة الجذر التربيعي:

#snippet(```python
def is_good_enough(guess, x):
    return abs(square(guess) - x) < 0.001
```)

نيّة مؤلف #py("is_good_enough") هي تحديد ما إذا كان مربع الوسيط الأول ضمن
تسامح معيّن من الوسيط الثاني. نرى أنّ المؤلف استخدم الاسم #py("guess") للإشارة
إلى الوسيط الأول و #py("x") للإشارة إلى الثاني. وسيط #py("square") هو
#py("guess"). لو استخدم مؤلف #py("square") الاسم #py("x") (كما أعلاه) للإشارة
إلى وسيطه، نرى أنّ #py("x") في #py("is_good_enough") يجب أن يكون #py("x")
مختلفاً عن ذلك في #py("square"). تشغيل الدالة #py("square") يجب ألا يؤثر على
قيمة #py("x") المستخدمة في #py("is_good_enough")، لأنّ تلك القيمة قد تكون
مطلوبة بعد انتهاء #py("square") من الحساب.

لو لم تكن المعلمات محلية لأجسام دوالها، لأمكن الخلط بين المعلمة #py("x") في
#py("square") والمعلمة #py("x") في #py("is_good_enough")، ولاعتمد سلوك
#py("is_good_enough") على أي نسخة من #py("square") استخدمنا. وعندها لن تكون
#py("square") الصندوق الأسود الذي أردناه.

#idx("parameters", sub: "names of")
#idx("name", sub: "of a parameter")
للمعلمة في الدالة دور خاص جداً في تعريف الدالة، إذ لا يهمّ ما اسمه. يُسمّى
هذا الاسم
#idx("bound name") #idx("name", sub: "bound") #emph[مقيَّداً]، ونقول إنّ تعريف الدالة
#idx("bind")
#emph[يُقيّد] معلماته. معنى تعريف الدالة لا يتغيّر إذا أُعيدت تسمية اسم
مقيَّد بشكل متّسق في جميع أنحاء التعريف.#footnote[مفهوم إعادة التسمية المتّسقة
في الواقع دقيق وصعب التعريف شكلياً. ارتكب منطقيون مشهورون أخطاء محرجة هنا.]
إذا لم يكن الاسم مقيَّداً نقول إنه
#idx("free name") #idx("name", sub: "free")
#emph[حر]. مجموعة التعليمات التي يُعرِّف فيها التقييد اسماً تُسمّى
#idx("scope of a name") #idx("name", sub: "scope of")
#emph[نطاق] ذلك الاسم. في تعريف الدالة، الأسماء المقيَّدة المُصرَّح بها
#idx("parameters", sub: "scope of") #idx("scope of a name", sub: "function's parameters")
كمعلمات للدالة نطاقها هو جسم الدالة.

في تعريف #py("is_good_enough") أعلاه، #py("guess") و #py("x") متغيّران
مقيَّدان لكن #py("abs") و #py("square") حرّان. يجب أن يكون معنى
#py("is_good_enough") مستقلاً عن الأسماء التي نختارها لـ #py("guess") و #py("x")
طالما أنها متمايزة ومختلفة عن #py("abs") و #py("square"). (لو أعدنا تسمية
#py("guess") إلى #py("abs") لكنّا أدخلنا خطأ برمجياً بـ
#idx("capturing a free name") #idx("bug", sub: "capturing a free name") #idx("free name", sub: "capturing")
#emph[التقاط] المتغيّر #py("abs"). لكان تحوّل من حر إلى مقيَّد.) لكنّ معنى
#py("is_good_enough") ليس مستقلاً عن أسماء متغيّراته الحرة. فهو يعتمد حتماً على
حقيقة (خارجية عن هذا التعريف) أنّ الاسم #py("abs") يشير إلى دالة لحساب القيمة
المطلقة لعدد. ستحسب الدالة #py("is_good_enough") دالة مختلفة لو استبدلنا
#py("math_cos") (دالة جيب التمام الأوّلية) بـ #py("abs") في تعريفها.
#idx("local name")

#subheading([التعريفات الداخلية وبنية الكتل])

#anchor(<sec:block-structure>)

لدينا حتى الآن نوع واحد من عزل الأسماء: معلمات الدالة محلية لجسم الدالة.
يوضّح برنامج الجذر التربيعي طريقة أخرى نرغب فيها بالتحكم في استخدام الأسماء.
#idx("program", sub: "structure of")
يتكوّن البرنامج الحالي من دوال منفصلة:

#snippet(```python
def sqrt(x):
    return sqrt_iter(1, x)

def sqrt_iter(guess, x):
    return (guess
            if is_good_enough(guess, x)
            else sqrt_iter(improve(guess, x), x))

def is_good_enough(guess, x):
    return abs(square(guess) - x) < 0.001

def improve(guess, x):
    return average(guess, x / guess)
```)

مشكلة هذا البرنامج أنّ الدالة الوحيدة المهمة لمستخدمي #py("sqrt") هي
#py("sqrt"). الدوال الأخرى (#py("sqrt_iter") و #py("is_good_enough")
و #py("improve")) تشوّش أذهانهم فقط. قد لا يستطيعون تعريف دالة أخرى باسم
#py("is_good_enough") كجزء من برنامج آخر يعمل مع برنامج الجذر التربيعي، لأنّ
#py("sqrt") تحتاجها. المشكلة حادة بشكل خاص في بناء أنظمة كبيرة بواسطة عدة
مبرمجين منفصلين. مثلاً، في بناء مكتبة كبيرة من الدوال العددية، تُحسب كثير من
الدوال العددية كتقريبات متتالية وقد يكون لديها دوال باسم #py("is_good_enough")
و #py("improve") كدوال مساعدة. نريد جعل الدوال الفرعية محلية، وإخفاءها داخل
#py("sqrt") حتى تتعايش #py("sqrt") مع تقريبات متتالية أخرى، لكل منها دالة
#py("is_good_enough") خاصة بها. لتحقيق ذلك، نسمح للدالة بأن يكون لها تعريفات
#idx("block structure")
#idx("internal definition")
داخلية محلية لتلك الدالة. مثلاً، في مشكلة الجذر التربيعي يمكننا كتابة

#snippet(```python
def sqrt(x):
    def is_good_enough(guess, x):
        return abs(square(guess) - x) < 0.001
    def improve(guess, x):
        return average(guess, x / guess)
    def sqrt_iter(guess, x):
        return (guess
                if is_good_enough(guess, x)
                else sqrt_iter(improve(guess, x), x))
    return sqrt_iter(1, x)
```)

يُشكّل جسم تعريف الدالة #emph[كتلة]؛ والتعريفات داخله محلية للدالة. #idx("block") #idx("syntactic forms", sub: "block")
هذا التداخل في التعريفات المسمّى #emph[بنية الكتل] هو أساساً الحل الصحيح لأبسط
مشكلة تغليف أسماء. لكن هناك فكرة أفضل تكمن هنا. بالإضافة إلى تداخل تعريفات
الدوال المساعدة، يمكننا تبسيطها. بما أنّ #py("x") مقيَّد في تعريف #py("sqrt")،
فإنّ الدوال #py("is_good_enough") و #py("improve") و #py("sqrt_iter") المُعرَّفة
داخلياً في #py("sqrt") تقع في نطاق #py("x"). وبالتالي ليس ضرورياً تمرير #py("x")
صراحة لكل من هذه الدوال. بدلاً من ذلك نسمح لـ #py("x") بأن يكون اسماً
#idx("internal definition", sub: "free name in") #idx("free name", sub: "in internal definition")
حراً في التعريفات الداخلية كما هو مبيّن أدناه. عندها يأخذ #py("x") قيمته من
الوسيط الذي استُدعيت به الدالة المحيطة #py("sqrt"). يُسمّى هذا الانضباط
#idx("lexical scoping")
#emph[النطاق المعجمي].#footnote[يفرض النطاق المعجمي أنّ الأسماء الحرة في دالة
تشير إلى ربط مُنشأ بواسطة تعريفات دوال محيطة؛ أي يُبحث عنها في
#idx("environment", sub: "lexical scoping and")
البيئة التي عُرِّفت فيها الدالة. سنرى كيف يعمل هذا بالتفصيل في الفصل @chap:state
حين ندرس البيئات والسلوك التفصيلي للمُفسِّر.]
#idx("sqrt", sub: "block structured", decl: true)
#snippet(```python
def sqrt(x):
    def is_good_enough(guess):
        return abs(square(guess) - x) < 0.001
    def improve(guess):
        return average(guess, x / guess)
    def sqrt_iter(guess):
        return (guess
                if is_good_enough(guess)
                else sqrt_iter(improve(guess)))
    return sqrt_iter(1)
```)

سنستخدم بنية الكتل بكثافة لمساعدتنا في تقسيم البرامج الكبيرة إلى أجزاء قابلة
للإدارة.#footnote[يجب أن تأتي التعريفات المُضمَّنة أولاً في جسم الدالة.
#idx("internal definition", sub: "position of")
الإدارة غير مسؤولة عن عواقب تشغيل برامج تُخلط فيها التعريفات والاستخدام؛ انظر
أيضاً الحاشية @foot:tdz في القسم @sec:lambda.]<foot:management>
نشأت فكرة بنية الكتل مع لغة البرمجة
#idx("Algol", sub: "block structure")
#en[Algol 60]. وهي تظهر في معظم لغات البرمجة المتقدّمة وهي أداة مهمة للمساعدة
في تنظيم بناء البرامج الكبيرة.
#idx("program", sub: "structure of")
#idx("block structure")
#idx("internal definition")
