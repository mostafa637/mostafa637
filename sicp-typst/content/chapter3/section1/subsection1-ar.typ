// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([متغيرات الحالة المحلية], label-name: <sec:local-state-variables>)

#idx("local state variable")
#idx("state variable", sub: "local")

لتوضيح ما نعنيه بوجود كائن حسابي ذي
#idx("object(s)", sub: "with time-varying state")
حالة متغيرة بمرور الوقت (#en[time-varying state])، دعونا ننمذج موقف سحب الأموال من
#idx("bank account")
حساب بنكي. سنفعل ذلك باستخدام الدالة
#py("withdraw")، والتي تأخذ كوسيط الـ
#py("amount") المراد سحبه.
إذا كان هناك ما يكفي من المال في الحساب لاستيعاب السحب، فإن #py("withdraw") يجب أن ترجع الرصيد المتبقي بعد السحب. بخلاف ذلك، يجب أن ترجع #py("withdraw") الرسالة #emph[أموال غير كافية] (#en[Insufficient funds]). على سبيل المثال، إذا بدأنا بـ 100 دولار في الحساب، فيجب أن نحصل على التسلسل التالي من الاستجابات باستخدام #py("withdraw"):

#snippet(```python
print(withdraw(25))
```)

#output(```python
print(withdraw(25))
```)

#snippet(```python
print(withdraw(25))
```)

#output(```python
print(withdraw(25))
```)

#snippet(```python
print(withdraw(60))
```)

#output(```python
print(withdraw(60))
```)

#snippet(```python
print(withdraw(15))
```)

#output(```python
print(withdraw(15))
```)

لاحظ أن التعبير
#py("withdraw(25)")،
عند تقييمه مرتين، ينتج قيمين مختلفين. هذا نوع جديد من السلوك لدالة.
حتى الآن، يمكن اعتبار جميع دوال بايثون لدينا كمواصفات لحساب دوال رياضية. استدعاء دالة يحسب قيمة الدالة المطبقة على الوسائط المعطاة، ويؤدي استدعاءان لنفس الدالة بنفس الوسائط دائماً إلى نفس النتيجة.#footnote[في الواقع، هذا ليس صحيحاً تماماً. أحد الاستثناءات كان
#idx("mathrandom (primitive function)", sub: "reassignment needed for")
#idx("random-number generator")
مُولّد الأرقام العشوائية في القسم @sec:primality. وتضمن استثناء آخر
#idx("operation-and-type table", sub: "assignment needed for")
جداول العمليات/الأنواع التي قدمناها في القسم @sec:data-directed، حيث تعتمد قيم استدعاءين لـ #py("get") بنفس الوسائط على استدعاءات متداخلة لـ #py("put"). من ناحية أخرى، حتى نقدم عملية إعادة الإسناد (#en[reassignment])، ليس لدينا طريقة لإنشاء مثل هذه الدوال بأنفسنا.]

لتنفيذ #py("withdraw")، يمكننا استخدام متغير #py("balance") للإشارة إلى رصيد الأموال في الحساب وتعريف #py("withdraw") كدالة تصل إلى #py("balance").
تتحقق الدالة #py("withdraw") لترى ما إذا كان #py("balance") كافياً بمقدار #py("amount") المطلوب. إذا كان الأمر كذلك، فإن #py("withdraw") تقوم بإنقاص #py("balance") بمقدار #py("amount") وترجع القيمة الجديدة لـ #py("balance"). بخلاف ذلك، ترجع #py("withdraw") الرسالة #emph[أموال غير كافية]. إليك تصريحات #py("balance") و #py("withdraw"):
#idx("withdraw", decl: true)
#snippet(```python
balance = 100

def withdraw(amount):
    global balance
    if balance >= amount:
        balance = balance - amount
        return balance
    else:
        return "Insufficient funds"
```)

يتم إنجاز إنقاص #py("balance") عن طريق الإسناد

#snippet(```python
balance = balance - amount
```)

تبدو هذه التعليمة كإسناد تصريحي، لكنها لا تصرح عن متغير جديد. التصريح العام (#en[global declaration])

#snippet(```python
global balance
```)

يضمن أن الاسم #py("balance") مصرح به مسبقاً في جسم الدالة #py("withdraw") ويشير إلى المتغير المصرح به عاماً #py("balance").
#idx("reassignment")
#idx("reassignment", sub: "reassignment statement")
#idx("variable", sub: "reassignment to")
#idx("syntactic forms", sub: "reassignment")
#idx("=", sort: "=")
يُسمى الإسناد الذي يأخذ الشكل

#syntax(meta("name"), " = ", meta("new-value"))

حيث يكون #meta("name") مصرحاً به مسبقاً، بـ #emph[إعادة الإسناد] (#en[reassignment]).

تغير إعادة الإسناد #meta("name") بحيث تصبح قيمته هي النتيجة التي يتم الحصول عليها بتقييم #meta("new-value"). في الحالة المطروحة، نحن نغير #py("balance") بحيث تصبح قيمته الجديدة هي نتيجة طرح #py("amount") من القيمة السابقة لـ #py("balance").#footnote[تبدو تعليمات إعادة الإسناد وإسنادات التصريح متشابهة ويجب ألا يتم الخلط بينها وبين
#idx("assignment", sub: "equality test vs.")
تعبيرات اختبار المساواة التي تأخذ الشكل #syntax(meta("expression_1"), " == ", meta("expression_2")) والتي تُقيّم كـ #py("True") إذا كان #meta("expression")$""_(1)$ يُقيّم إلى نفس القيمة مثل #meta("expression")$""_(2)$ وكـ #py("False") بخلاف ذلك.]

تستخدم الدالة #py("withdraw") أيضاً
#idx("sequence of statements")
#emph[تسلسلاً من التعليمات] (#en[sequence of statements]) للتسبب في تقييم تعليمتين في الحالة التي يكون فيها اختبار #py("if") صحيحاً: أولاً إنقاص #py("balance") ثم إرجاع قيمة #py("balance"). بشكل عام، يؤدي تنفيذ تسلسل

#syntax(meta("stmt"), $""_(1)$, " ", meta("stmt"), $""_(2) dots.h$, meta("stmt"), $""_(n)$)

إلى تقييم التعليمات #meta("stmt")$""_(1)$ إلى #meta("stmt")$""_(n)$ بالتسلسل.#footnote[لقد استخدمنا بالفعل
#idx("sequence of statements", sub: "in block")
تسلسلات ضمنياً في برامجنا، لأنه في بايثون، يمكن أن تحتوي كتلة جسم الدالة على تسلسل من تعريفات الدوال متبوعة بتعليمة إرجاع، وليس فقط تعليمة إرجاع واحدة، كما نوقش في القسم @sec:block-structure.]

على الرغم من أن #py("withdraw") تعمل كما هو مطلوب، إلا أن المتغير #py("balance") يمثل مشكلة. كما هو محدد أعلاه، #py("balance") هو اسم مُعرّف في بيئة البرنامج وهو قابل للوصول بحرية ليتم فحصه أو تعديله بواسطة أي دالة. سيكون من الأفضل بكثير لو تمكنا بطريقة ما من جعل #py("balance") داخلياً بالنسبة لـ #py("withdraw")، بحيث تكون #py("withdraw") هي الدالة الوحيدة التي يمكنها الوصول إلى #py("balance") بشكل مباشر ولا يمكن لأي دالة أخرى الوصول إلى #py("balance") إلا بشكل غير مباشر (من خلال استدعاءات #py("withdraw")). من شأن هذا أن ينمذج بشكل أكثر دقة فكرة أن #py("balance") هو متغير حالة محلي تستخدمه #py("withdraw") لتتبع حالة الحساب.

يمكننا جعل #py("balance") داخلياً لـ #py("withdraw") بإعادة كتابة التعريف على النحو التالي:

#idx("newwithdraw", decl: true)
#snippet(```python
def make_withdraw_balance_100():
    balance = 100
    def withdraw(amount):
        nonlocal balance
        if balance >= amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    return withdraw

new_withdraw = make_withdraw_balance_100()
```)

ما فعلناه هنا هو استخدام إسناد تصريحي لإنشاء بيئة بمتغير محلي #py("balance")، مرتبط بالقيمة الأولية 100. داخل هذه البيئة المحلية، نستخدم تصريح دالة لإنشاء دالة #py("withdraw") تأخذ #py("amount") كوسيط وتتصرف—عند إرجاعها كنتيجة لتقييم جسم الدالة #py("make_withdraw_balance_100")—بنفس الطريقة تماماً كدالة #py("withdraw") السابقة لدينا، ولكن متغيرها #py("balance") لا يمكن الوصول إليه بواسطة أي دالة أخرى.#footnote[في مصطلحات لغات البرمجة، يُقال إن المتغير #py("balance")
#idx("encapsulated name")
#idx("name", sub: "encapsulated")
#emph[مُغلف] (#en[encapsulated]) داخل الدالة #py("new_withdraw").
يعكس التغليف (#en[Encapsulation]) مبدأ تصميم النظم العام المعروف بـ
#idx("hiding principle")
#idx("modularity", sub: "hiding principle")
#emph[مبدأ الإخفاء] (#en[hiding principle]): يمكن للمرء أن يجعل النظام أكثر نمطية وقوة من خلال حماية أجزاء النظام من بعضها البعض؛ أي من خلال توفير الوصول إلى المعلومات فقط لتلك الأجزاء من النظام التي لديها "حاجة للمعرفة".]

تواجه الدالة #py("withdraw") المتداخلة مشكلة مشابهة لمشكلة الدالة #py("withdraw") العامة السابقة:
الإسناد
`balance = balance - amount`
يجب أن يشير إلى المتغير `balance` المصرح به خارج الدالة #py("withdraw")،
ولا ينبغي أن يعيد التصريح عن الاسم `balance`. ومع ذلك، الآن الاسم `balance` ليس اسماً عاماً، ولكنه اسم مصرح به في الدالة المحيطة `make_withdraw_balance_100`.
تستخدم بايثون تصريحات `nonlocal` في هذا الموقف.
يشير التصريح `nonlocal balance` في الدالة `withdraw` المتداخلة إلى أن أي إسناد للاسم `balance` داخل الدالة `withdraw` يشير إلى المتغير `balance` المصرح به خارج الدالة ولكن ليس عاماً.#footnote[على غرار التصريحات العامة (#en[global])، لا يمتد تصريح `nonlocal` إلى الدوال المتداخلة داخل الدالة التي يظهر فيها تصريح `nonlocal`. لكي تتمكن هذه الدوال المتداخلة من إعادة الإسناد للمتغير الـ `nonlocal`، ستحتاج هي الأخرى إلى التصريح عن المتغير على أنه `nonlocal`.]

دمج عمليات إعادة الإسناد مع إسنادات التصريح هو تقنية البرمجة العامة التي سنستخدمها لبناء كائنات حسابية ذات حالة محلية. لسوء الحظ، يثير استخدام هذه التقنية مشكلة خطيرة: عندما قدمنا الدوال لأول مرة، قدمنا أيضاً نموذج الاستبدال للتقييم (القسم @sec:substitution-model) لتوفير تفسير لما يعنيه تطبيق الدالة. قلنا إن تطبيق دالة يكون جسمها عبارة عن تعليمة إرجاع (`return`) يجب أن يُفسر على أنه تقييم لتعبير الإرجاع للدالة مع استبدال المعاملات بقيمها. بالنسبة للدوال ذات الأجسام الأكثر تعقيداً، نحتاج إلى تقييم الجسم بأكمله مع استبدال المعاملات بقيمها.
تكمن المشكلة في أنه، بمجرد تقديمنا للإسناد في لغتنا، لم يعد الاستبدال نموذجاً كافياً لتطبيق الدالة. (سنرى لماذا هذا صحيح في القسم @sec:costs-of-assignment). ونتيجة لذلك، ليس لدينا من الناحية الفنية في هذه المرحلة أي طريقة لفهم سبب تصرف الدالة #py("new_withdraw") كما زُعم أعلاه. من أجل فهم دالة مثل #py("new_withdraw") حقاً، سنحتاج إلى تطوير نموذج جديد لتطبيق الدالة. في القسم @sec:environment-model سنقدم مثل هذا النموذج، مع شرح لتعليمات إعادة الإسناد وإسنادات التصريح. أولاً، ومع ذلك، نفحص بعض التغييرات على الفكرة التي أنشأتها #py("new_withdraw").

الدالة التالية، #py("make_withdraw")، تنشئ "معالجات سحب". يحدد المعامل #py("balance") في #py("make_withdraw") المبلغ الأولي للمال في الحساب.#footnote[على عكس #py("make_withdraw_balance_100") أعلاه، لا يتعين علينا استخدام إسناد تصريحي لجعل #py("balance") متغيراً محلياً، نظراً لأن المعاملات هي بالفعل محلية. سيصبح هذا أكثر وضوحاً بعد مناقشة نموذج البيئة للتقييم في القسم @sec:environment-model. (انظر أيضاً التمرين @ex:local-state-variable).]<foot:make_withdraw>
#idx("makewithdraw", decl: true)
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

يمكن استخدام الدالة #py("make_withdraw") على النحو التالي لإنشاء كائنين #py("W1") و #py("W2"):

#snippet(```python
W1 = make_withdraw(100)
W2 = make_withdraw(100)
```)

#snippet(```python
print(W1(50))
```)

#output(```python
print(W1(50))
```)

#snippet(```python
print(W2(70))
```)

#output(```python
print(W2(70))
```)

#snippet(```python
print(W2(40))
```)

#output(```python
print(W2(40))
```)

#snippet(```python
print(W1(40))
```)

#output(```python
print(W1(40))
```)

لاحظ أن #py("W1") و #py("W2") كائنان مستقلان تماماً، ولكل منهما متغير حالة محلي خاص به #py("balance"). لا تؤثر عمليات السحب من أحدهما على الآخر.

يمكننا أيضاً إنشاء كائنات تتعامل مع
#idx("deposit message for bank account", decl: true)
الودائع بالإضافة إلى عمليات السحب، وبالتالي يمكننا تمثيل حسابات بنكية بسيطة. إليك دالة ترجع "كائن حساب بنكي" برصيد أولي محدد:
#idx("makeaccount", decl: true)
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
                else error("unknown request -- make_account", m))
    return dispatch
```)

تقوم كل مكالمة لـ #py("make_account") بإعداد بيئة تحتوي على متغير حالة محلي #py("balance"). داخل هذه البيئة، تُعرّف #py("make_account") دوال #py("deposit") و #py("withdraw") التي تصل إلى #py("balance")، ودالة إضافية #py("dispatch") تأخذ "رسالة" كدخل وترجع إحدى الدالتين المحليتين. الدالة #py("dispatch") نفسها يتم إرجاعها كقيمة تمثل كائن الحساب البنكي. هذا بالضبط هو أسلوب
#idx("message passing", sub: "in bank account")
#emph[تمرير الرسائل] (#en[message-passing]) في البرمجة الذي رأيناه في القسم @sec:data-directed، على الرغم من أننا نستخدمه هنا جنباً إلى جنب مع القدرة على تعديل المتغيرات المحلية.

يمكن استخدام الدالة #py("make_account") على النحو التالي:

#snippet(```python
acc = make_account(100)
```)

#snippet(```python
print(acc("withdraw")(50))
```)

#output(```python
print(acc("withdraw")(50))
```)

#snippet(```python
print(acc("withdraw")(60))
```)

#output(```python
print(acc("withdraw")(60))
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

ترجع كل مكالمة لـ #py("acc") الدالة المعرفة محلياً #py("deposit") أو #py("withdraw")، والتي يتم تطبيقها بعد ذلك على الـ #py("amount") المحدد. كما كان الحال مع #py("make_withdraw")، فإن مكالمة أخرى لـ #py("make_account")

#snippet(```python
acc2 = make_account(100)
```)

ستنتج كائن حساب منفصل تماماً، يحافظ على #py("balance") المحلي الخاص به.

#exercise(label-name: <ex:make-accumulator>, [
#idx("accumulator")
#emph[المُراكِم] (#en[accumulator]) هو دالة يتم استدعاؤها مراراً وتكراراً بوسيط رقمي واحد وتقوم بتراكم وسائطها في مجموع. في كل مرة يتم استدعاؤها، ترجع المجموع المتراكم حالياً. اكتب دالة
#idx("makeaccumulator")
#py("make_accumulator")
تولد مُراكمات، يحافظ كل منها على مجموع مستقل. يجب أن يحدد الدخل لـ #py("make_accumulator") القيمة الأولية للمجموع؛ على سبيل المثال:

#snippet(```python
a = make_accumulator(5)
```)

#snippet(```python
print(a(10))
```)

#output(```python
print(a(10))
```)

#snippet(```python
print(a(10))
```)

#output(```python
print(a(10))
```)
])

#exercise(label-name: <ex:make-monitored>, [
في تطبيقات اختبار البرمجيات، من المفيد أن تكون قادراً على حساب عدد المرات التي يتم فيها استدعاء دالة معينة أثناء سير العملية الحسابية. اكتب دالة
#idx("makemonitored")
#idx("monitored function")

#py("make_monitored")
تأخذ كدخل دالة #py("f")، والتي بدورها تأخذدخلاً واحداً. النتيجة المُرجعة بواسطة #py("make_monitored") هي دالة ثالثة، لنقل #py("mf")، والتي تتبع عدد المرات التي تم استدعاؤها من خلال الاحتفاظ بعداد داخلي. إذا كان الدخل لـ #py("mf") هو السلسلة النصية #py("\"how many calls\"")، فإن #py("mf") ترجع قيمة العداد. إذا كان الدخل هو السلسلة النصية #py("\"reset count\"")، فإن #py("mf") تعيد ضبط العداد إلى الصفر. لأي دخل آخر، ترجع #py("mf") نتيجة استدعاء #py("f") على ذلك الدخل وتقوم بزيادة العداد. على سبيل المثال، يمكننا إنشاء نسخة مراقبة من دالة #py("sqrt"):

#snippet(```python
s = make_monitored(math_sqrt)
```)

#snippet(```python
print(s(100))
```)

#output(```python
print(s(100))
```)

#snippet(```python
print(s("how many calls"))
```)

#output(```python
print(s("how many calls"))
```)
])

#exercise(label-name: <ex:password-protection>, [
قم بتعديل دالة #py("make_account") بحيث تنشئ
#idx("bank account", sub: "password-protected")
#idx("password-protected bank account")
حسابات محمية بكلمة مرور. أي أن #py("make_account") يجب أن تأخذ سلسلة نصية كوسيط إضافي، كما في

#snippet(```python
acc = make_account(100, "secret password")
```)

يجب أن يعالج كائن الحساب الناتج الطلب فقط إذا كان مصحوباً بكلمة المرور التي تم إنشاء الحساب بها، وبخلاف ذلك يجب أن يرجع شكوى:

#snippet(```python
print(acc("secret password", "withdraw")(40))
```)

#output(```python
print(acc("secret password", "withdraw")(40))
```)

#snippet(```python
print(acc("some other password", "deposit")(40))
```)

#output(```python
print(acc("some other password", "deposit")(40))
```)
])

#exercise(label-name: <ex:3_4>, [
قم بتعديل دالة #py("make_account") الخاصة بالتمرين @ex:password-protection عن طريق إضافة متغير حالة محلي آخر بحيث إذا تم الوصول إلى الحساب أكثر من سبع مرات متتالية بكلمة مرور غير صحيحة، فإنه يستدعي دالة #py("call_the_cops").
])

#idx("local state variable")
#idx("state variable", sub: "local")
