// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([التصريحات الداخلية], label-name: <sec:env-internal-def>)

#idx("block structure", sub: "in environment model")
#idx("environment model of evaluation", sub: "internal declarations")
#idx("internal declaration", sub: "in environment model")

نتناول في هذا القسم تقييم أجسام الدوال أو الكتل الأخرى (مثل فروع التعليمات الشرطية) التي تحتوي على تصريحات.
تفتح كل كتلة نطاقاً جديداً للأسماء المصرح بها في الكتلة.
ولتقييم كتلة في بيئة معطاة، نوسع تلك البيئة بإطار جديد يحتوي على جميع الأسماء المصرح بها مباشرة (أي خارج الكتل المتداخلة) في جسم الكتلة، ثم نقيم الجسم في البيئة المنشأة حديثاً.

قدم القسم @sec:black-box الفكرة القائلة بأن الدوال يمكن أن تحتوي على تصريحات داخلية، مما يؤدي إلى بنية كتلة كما في
#idx("sqrt", sub: "in environment model")
دالة حساب الجذور التربيعية التالية:

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

يمكننا الآن استخدام نموذج البيئة لرؤية سبب تصرف هذه التصريحات الداخلية كما هو مرغوب فيه.
يُظهر الشكل @fig:sqrt-internal النقطة في تقييم التعبير #py("sqrt(2)") حيث أُستدعيت الدالة الداخلية #py("is_good_enough") لأول مرة مع كون #py("guess") مساوياً لـ 1.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-12.svg", width: 70%), caption: [دالة #py("sqrt") بتصريحات داخلية.], label-name: <fig:sqrt-internal>)

لاحظ بنية البيئة.
الاسم #py("sqrt") مرتبط في بيئة البرنامج بكائن دالة بيئته المرتبطة هي بيئة البرنامج. وعندما أُستدعيت #py("sqrt")، أُنشئت بيئة جديدة، E1، تابعة لبيئة البرنامج، يُربط فيها المعامل #py("x") بـ 2. ثم أُقيِّم جسم #py("sqrt") في E1.

هذا الجسم عبارة عن كتلة بتعاريف دوال محلية ولذلك أُوُسِّعت E1 بإطار جديد لتلك التصريحات، مما نتج عنه البيئة الجديدة E2. ثم أُقيِّم جسم الكتلة في E2. وبما أن التعليمة الأولى في الجسم هي

#snippet(```python
def is_good_enough(guess):
    return abs(square(guess) - x) < 0.001
```)

فإن تقييم هذا التصريح أنشأ الدالة #py("is_good_enough") في البيئة E2.

ولنكون أكثر دقة، فإن الاسم #py("is_good_enough") في الإطار الأول لـ E2 أُربط بكائن دالة بيئته المرتبطة هي E2.

وبالمثل، أُشئت #py("improve") و #py("sqrt_iter") كدوال في E2.
وللاختصار، يُظهر الشكل @fig:sqrt-internal فقط كائن الدالة لـ #py("is_good_enough").

وبعد تعريف الدوال المحلية، أُقيِّم التعبير #py("sqrt_iter(1)")، لا يزال في البيئة E2.
لذلك أُستدعي كائن الدالة المرتبط بـ #py("sqrt_iter") في E2 مع كون 1 كـ وسيط. ينشئ هذا بيئة E3 يُربط فيها #py("guess")، معامل #py("sqrt_iter")، بـ 1.
وتستدعي الدالة #py("sqrt_iter") بدورها #py("is_good_enough") بقيمة #py("guess") (من E3) كـ وسيط لـ #py("is_good_enough").
أنشأ هذا بيئة أخرى، E4، يُربط فيها #py("guess") (معامل #py("is_good_enough")) بـ 1. وعلى الرغم من أن كل من #py("sqrt_iter") و #py("is_good_enough") لديهما معامل مسمى بـ #py("guess")، إلا أنهما متغيران محليان متميزان يقعان في إطارين مختلفين.

أيضاً، لكل من E3 و E4 البيئة E2 كبيئة محيطة بهما، لأن كلاً من الدالتين #py("sqrt_iter") و #py("is_good_enough") لديهما E2 كجزء البيئة الخاص بهما.

وإحدى نتائج هذا هي أن الاسم #py("x") الذي يظهر في جسم #py("is_good_enough") سيشير إلى رابط #py("x") الذي يظهر في E1، وهو قيمة #py("x") التي أُستدعيت بها دالة #py("sqrt") الأصلية.
#idx("sqrt", sub: "in environment model")

وبالتالي فإن نموذج البيئة يشرح الخصائص الرئيسية التي تجعل تعاريف الدوال المحلية تقنية مفيدة لجعل البرامج نمطية:

- لا تتداخل أسماء الدوال المحلية مع الأسماء الخارجية بالنسبة للدالة المحيطة، لأن أسماء الدوال المحلية ستكون مرتبطة في الإطار الذي تنشئه الكتلة عند تقييمها، بدلاً من الارتباط في بيئة البرنامج.
- يمكن للدوال المحلية الوصول إلى وسائط الدالة المحيطة، بمجرد استخدام أسماء المعاملات كـ أسماء حرة. وذلك لأن جسم الدالة المحلية يُقيّم في بيئة تابعة لبيئة التقييم للدالة المحيطة.

#exercise(label-name: <ex:two-accounts>, [
في القسم @sec:env-local-state رأينا كيف يصف نموذج البيئة سلوك الدوال ذات الحالة المحلية. والآن رأينا كيف تعمل التصريحات الداخلية.
#idx("environment model of evaluation", sub: "message passing")
#idx("message passing", sub: "environment model and")
تحتوي دالة تمرير الرسائل النموذجية على كلا الجانبين. فكر في دالة
#idx("bank account")
الحساب البنكي من القسم @sec:local-state-variables:
#idx("makeaccount", sub: "in environment model")
#snippet(```python
def make_account(balance):
    def withdraw(amount):
        nonlocal balance
        if balance >= amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    def deposit(amount):
        nonlocal balance
        balance = balance + amount
        return balance
    def dispatch(m):
        return (withdraw if m == "withdraw"
                else deposit if m == "deposit"
                else error("Unknown request: make_account", m))
    return dispatch
```)

وضّح بنية البيئة الموّلدة بوساطة تسلسل التفاعلات

#snippet(```python
acc = make_account(50)
```)

#snippet(```python
print(acc("deposit")(40))
```)

#output(```python
print(acc("deposit")(40))
```)

#snippet(```python
print(acc("withdraw")(60))
```)

#output(```python
print(acc("withdraw")(60))
```)

أين يُحتفظ بالحالة المحلية لـ #py("acc")؟
نفترض أننا عرّفنا حساباً آخر

#snippet(```python
acc2 = make_account(100)
```)

كيف تُحفظ الحالات المحلية للحسابين متميزة؟ وما هي أجزاء بنية البيئة المشتركة بين #py("acc") و #py("acc2")؟
])

=== المزيد عن الكتل

كما رأينا، فإن نطاق الأسماء المصرح بها في #py("sqrt") هو جسم #py("sqrt") بأكمله. يشرح هذا سبب عمل
#idx("mutual recursion")
#idx("recursion", sub: "mutual")
#emph[العودية المتبادلة] (#en[mutual recursion])، كما في هذه الطريقة (المسرفة جداً) للتحقق مما إذا كان عدد صحيح غير سالب زوجياً.

#syntax("
def f(x):
    def is_even(n):
        return (True
                if n == 0
                else is_odd(n - 1))
    def is_odd(n):
        return (False
                if n == 0
                else is_even(n - 1))
    return is_even(x)
      ")

في الوقت الذي تُستدعى فيه #py("is_even") أثناء استدعاء لـ #py("f")، يبدو مخطط البيئة مثل ذلك الموضح في الشكل @fig:sqrt-internal عند استدعاء #py("sqrt_iter"). وتكون الدالتان #py("is_even") و #py("is_odd") مرتبطتين في E2 بكائنين داليين يشيران إلى E2 كالبيئة التي يُقَيَّم فيها الاستدعاء لتلك الدوال. وبالتالي فإن #py("is_odd") في جسم #py("is_even") تشير إلى الدالة الصحيحة.
وعلى الرغم من أن #py("is_odd") معرّفة بعد #py("is_even")، فإنه لا يختلف ذلك عن كيفية إشارة الاسم #py("improve") والاسم #py("sqrt_iter") نفسه إلى الدوال الصحيحة في جسم #py("sqrt_iter").

ومزودين بطريقة للتعامل مع التصريحات داخل الكتل، يمكننا إعادة زيارة تصريحات الأسماء عند المستوى الأعلى. في القسم @sec:env-model-rules، رأينا أن الأسماء المصرح بها عند المستوى الأعلى تُضاف إلى إطار البرنامج. والشرح الأفضل هو أن البرنامج بأكمله يُوضع في كتلة ضمنية، تُقيّم في البيئة العامة.
ثم تعامل معاملة الكتل الموصوفة أعلاه المستوى الأعلى:
تُوُسَّع البيئة العامة بإطار يحتوي على روابط جميع الأسماء المصرح بها في الكتلة الضمنية. وذلك الإطار هو إطار البرنامج والبيئة الناتجة هي
#idx("program environment")
بيئة البرنامج.

قلنا إن جسم الكتلة يُقيّم في بيئة تحتوي على جميع الأسماء المصرح بها مباشرة في جسم الكتلة.
ويُوضع الاسم المصرح به محلياً في البيئة عند دخول الكتلة، ولكن بدون قيمة مرتبطة. وتقييم تصريحه أثناء تقييم جسم الكتلة يسند حينها للاسم نتيجة تقييم التعبير إلى يمين #py("=")، كما لو كان التصريح إسناداً. وبما أن إضافة الاسم إلى البيئة منفصلة عن تقييم التصريح، والكتلة بأكملها تقع في نطاق الاسم، فإن برنامجاً خاطئاً قد يحاول
#idx("declaration", sub: "use of name before")
الوصول إلى قيمة اسم قبل تقييم تصريحه؛ وتقييم اسم غير مسند يرسل خطأ.#footnote[يشرح هذا سبب خطأ البرنامج في الحاشية في الفصل 1.
يُسمى الوقت بين إنشاء الرابط لاسم وتقييم تصريح الاسم بـ
#idx("temporal dead zone (TDZ)")
#idx("TDZ (temporal dead zone)")
#emph[منطقة الموت الزمنية] (#en[temporal dead zone - TDZ]).]<foot:tdz_explained>

#idx("environment model of evaluation")
#idx("block structure", sub: "in environment model")
#idx("environment model of evaluation", sub: "internal declarations")
#idx("internal declaration", sub: "in environment model")
