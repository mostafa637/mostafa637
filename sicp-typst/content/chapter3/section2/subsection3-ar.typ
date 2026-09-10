// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([الأطر كمستودع للحالة المحلية], label-name: <sec:env-local-state>)

#idx("frame (environment model)", sub: "as repository of local state")
#idx("local state", sub: "maintained in frames")
#idx("environment model of evaluation", sub: "local state")

يمكننا الرجوع إلى نموذج البيئة لرؤية كيفية استخدام الدوال والإسناد لتمثيل كائنات ذات حالة محلية. كـ مثال، فكر في
#idx("makewithdraw", sub: "in environment model")
"معالج السحب" من القسم @sec:local-state-variables المُنشأ باستدعاء الدالة

#snippet(```python
def make_withdraw(balance):
    def withdraw(amount):
        nonlocal balance
        if balance >= amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    return withdraw
```)

دعونا نصف تقييم

#snippet(```python
W1 = make_withdraw(100)
```)

متبوعاً بـ

#snippet(```python
print(W1(50))
```)

#output(```python
print(W1(50))
```)

يُظهر الشكل @fig:make-withdraw نتيجة تصريح الدالة #py("make_withdraw") في بيئة البرنامج. هذا ينتج كائن دالة يحتوي على مؤشر إلى بيئة البرنامج. وحتى الآن، لا يختلف هذا عن الأمثلة التي رأيناها بالفعل، باستثناء أن تعبير الإرجاع في جسم الدالة هو نفسه تعبير لامدا.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-7.svg", width: 70%), caption: [نتيجة تعريف #py("make_withdraw") في بيئة البرنامج.], label-name: <fig:make-withdraw>)

والجزء المثير للاهتمام من الحساب يحدث عندما نطبق الدالة #py("make_withdraw") على وسيط:

#snippet(```python
W1 = make_withdraw(100)
```)

نبدأ، كالمعتاد، بإنشاء بيئة E1 يُربط فيها المعامل #py("balance") بالوسيط 100. وداخل هذه البيئة، نقوم بتقييم جسم #py("make_withdraw")، أي تعليمة الإرجاع التي تعبير إرجاعها هو تعبير لامدا. وينتج عن تقييم تعبير لامدا هذا كائن دالة جديد، تكون شفرته كما هو محدد بتعبير لامدا وتكون بيئته هي E1، البيئة التي أُقيِّم فيها تعبير لامدا لإنتاج الدالة.
وكائن الدالة الناتج هو القيمة المُرجعة بوساطة الاستدعاء لـ #py("make_withdraw").
ويُربط هذا بـ #py("W1") في بيئة البرنامج، حيث إن تصريح الثابت نفسه يُقيَّم في بيئة البرنامج.
ويُظهر الشكل @fig:w1 بنية البيئة الناتجة.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-8.svg", width: 70%), caption: [نتيجة تقييم #py("W1 = make_withdraw(100)").], label-name: <fig:w1>)

الآن يمكننا تحليل ما يحدث عندما يُطبق #py("W1") على وسيط:

#snippet(```python
print(W1(50))
```)

#output(```python
print(W1(50))
```)

نبدأ بإنشاء إطار يُربط فيه #py("amount")، معامل #py("W1")، بالوسيط 50. والنقطة الحاسمة التي يجب ملاحظتها هي أن هذا الإطار له كبيئته المحيطة ليس بيئة البرنامج، بل البيئة E1، لأن هذه هي البيئة المحددة بوساطة كائن الدالة #py("W1"). وداخل هذه البيئة الجديدة، نقوم بتقييم جسم الدالة:

#snippet(```python
if balance >= amount:
    balance = balance - amount
    return balance
else:
    return "Insufficient funds"
```)

بنية البيئة الناتجة موضحة في الشكل @fig:apply-w1.
يشير التعبير الجاري تقييمه إلى كل من #py("amount") و #py("balance").
سيُعثر على المتغير #py("amount") في الإطار الأول في البيئة، وسُيعثر على #py("balance") باتباع مؤشر البيئة المحيطة إلى E1.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-9.svg", width: 70%), caption: [البيئات المنشأة بتطبيق كائن الدالة #py("W1").], label-name: <fig:apply-w1>)

وعندما يُنفذ الإسناد، يتغير رابط #py("balance") في E1. وعند اكتمال الاستدعاء لـ #py("W1")، يصبح #py("balance") مساوياً لـ 50، ولا يزال الإطار الذي يحتوي على #py("balance") مُؤشراً إليه بوساطة كائن الدالة #py("W1"). والإطار الذي يربط #py("amount") (والذي نفذنا فيه الشفرة التي غيرت #py("balance")) لم يعد صالباً، حيث إن استدعاء الدالة الذي أنشأه قد انتهى، ولا توجد مؤشرات إلى ذلك الإطار من أجزاء أخرى من البيئة. وفي المرة التالية التي يُستدعى فيها #py("W1")، سينشئ هذا إطاراً جديداً يربط #py("amount") وتكون بيئته المحيطة هي E1. ونرى أن E1 يعمل كـ "المكان" الذي يحتفظ بمتغير الحالة المحلي لكائن الدالة #py("W1").
ويُظهر الشكل @fig:after-w1 الموقف بعد الاستدعاء لـ #py("W1").

#sicp-figure(image("/images/img_javascript/ch3-Z-G-10.svg", width: 70%), caption: [البيئات بعد الاستدعاء لـ #py("W1").], label-name: <fig:after-w1>)

لاحظ ما يحدث عندما ننشئ كائن "سحب" ثانياً عن طريق إجبار استدعاء آخر لـ #py("make_withdraw"):

#snippet(```python
W2 = make_withdraw(100)
```)

هذا ينتج بنية البيئة في الشكل @fig:w2، والتي تُظهر أن #py("W2") هو كائن دالة، أي زوج يحتوي على بعض الشفرات وبيئة. والبيئة E2 لـ #py("W2") أُنشئت بواسطة الاستدعاء لـ #py("make_withdraw"). وتتسع لإطار برابطه المحلي الخاص لـ #py("balance"). من ناحية أخرى، فإن #py("W1") و #py("W2") لهما الشفرة نفسها: الشفرة المحددة بتعبير لامدا في جسم #py("make_withdraw").#footnote[سواء كان #py("W1") و #py("W2") يتشاركان نفس الشفرة الفيزيائية المخزنة في الحاسوب، أم يحتفظ كل منهما بنسخة من الشفرة، فهذا تفصيل تنفيذي. بالنسبة للمفسر الذي نفذناه في الفصل @chap:meta، فإن الشفرة في الواقع مشتركة.] ونرى هنا لماذا يتصرف #py("W1") و #py("W2") ككائنين مستقلين. فالاستدعاءات لـ #py("W1") تشير إلى متغير الحالة #py("balance") المخزن في E1، بينما الاستدعاءات لـ #py("W2") تشير إلى #py("balance") المخزن في E2. وبالتالي، فإن التغييرات في الحالة المحلية لكائن ما لا تؤثر على الكائن الآخر.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-11.svg", width: 70%), caption: [استخدام #py("W2 = make_withdraw(100)") لإنشاء كائن ثانٍ.], label-name: <fig:w2>)

#exercise(label-name: <ex:local-state-variable>, [
في دالة #py("make_withdraw") يُنشأ المتغير المحلي #py("balance") كمعامل لـ #py("make_withdraw").
وكان بإمكاننا أيضاً إنشاء متغير الحالة المحلي بشكل منفصل، باستخدام ما قد نسميه
#idx("lambda expression", sub: "immediately invoked")
#idx("immediately invoked lambda expression")
#emph[تعبير لامدا المُنَفَّذ فوراً] (#en[immediately invoked lambda expression]) كما يلي:
#idx("makewithdraw", sub: "using immediately invoked lambda expression", decl: true)
#snippet(```python
def make_withdraw(initial_amount):
    def init(balance):
        def withdraw(amount):
            nonlocal balance
            if balance >= amount:
                balance = balance - amount
                return balance
            else:
                return "Insufficient funds"
        return withdraw
    return init(initial_amount)
```)

تُستدعى الدالة #py("init") فوراً بعد تقييمها. والغرض الوحيد منها هو إنشاء متغير محلي #py("balance") وإسناده أولياً إلى #py("initial_amount").

استخدم نموذج البيئة لتحليل هذه النسخة البديلة من #py("make_withdraw")، وارسم أشكالاً مثل تلك الموضحة أعلاه لتوضيح التفاعلات:

#snippet(```python
W1 = make_withdraw(100)

W1(50)

W2 = make_withdraw(100)
```)

أظهر أن النسختين من #py("make_withdraw") تنشئان كائنات بنفس السلوك. وكيف تختلف بنيات البيئة للنسختين؟
])

#idx("frame (environment model)", sub: "as repository of local state")
#idx("local state", sub: "maintained in frames")
#idx("environment model of evaluation", sub: "local state")
#idx("makewithdraw", sub: "in environment model")
