// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([ما المقصود بالبيانات؟], label-name: <sec:data->)

#idx("data", sub: "meaning of")

بدأنا تنفيذ الأعداد الناطقة في القسم @sec:rationals بتنفيذ عمليات الأعداد الناطقة #py("add_rat") و #py("sub_rat") وما إلى ذلك بدلالة ثلاث دوال غير محدودة: #py("make_rat") و #py("numer") و #py("denom"). وفي تلك النقطة، كان بإمكاننا التفكير في العمليات كما لو كانت مُحدَّدة بدلالة كائنات بيانات — بسط ومقام وأعداد ناطقة — والتي حُدِّد سلوكها بالدوال الثلاث الأخيرة.

ولكن ما المقصود بالـ #emph[بيانات] بالضبط؟ لا يكفي القول «كل ما يتم تنفيذه بالمحدّدات والمُنشِئات المعطاة». فمن الواضح أن ليس أي مجموعة اختيارية من ثلاث دوال يمكن أن تعمل كأساس مناسب لتنفيذ الأعداد الناطقة. إذ نحتاج إلى ضمان أنه،
#idx("makerat", sub: "axiom for")
#idx("numer", sub: "axiom for")
#idx("denom", sub: "axiom for")
إذا بنينا عدداً ناطقاً #py("x") من زوج أعداد صحيحة #py("n") و #py("d")، فإنّ استخراج #py("numer") و #py("denom") لـ #py("x") وقسمتهما يجب أن يعطي نفس نتيجة قسمة #py("n") على #py("d"). وبعبارة أخرى، يجب أن تحقّق #py("make_rat") و #py("numer") و #py("denom") الشرط أنه، لأي عدد صحيح #py("n") وأي عدد صحيح غير صفري #py("d")، إذا كان #py("x") هو #py("make_rat(n, d)")، فإنّ:

$ mat(delim: #none, frac(mono("numer")(mono("x")), mono("denom")(mono("x"))), =, frac(mono("n"), mono("d"))) $

وفي الواقع، هذا هو الشرط الوحيد الذي يجب أن تلبيه #py("make_rat") و #py("numer") و #py("denom") من أجل تشكيل أساس مناسب لتمثيل الأعداد الناطقة. وبوجه عام، يمكننا التفكير في البيانات كما لو كانت مُحدَّدة بمجموعة ما من المحدّدات والمُنشِئات، جنباً إلى جنب مع شروط محددة يجب أن تلبيها هذه الدوال حتى تكون تمثيلاً صالحاً.#footnote[من المدهش أنّ يصعب جداً صياغة هذه الفكرة بشكل صارم. وهناك مقاربتاً لتقديم مثل هذه الصياغة. المقاربة الأولى، التي رادها
#idx("Hoare, Charles Antony Richard")
سي. أي. آر. هوار (1972)، تُعرَف بطريقة
#idx("data", sub: "abstract models for")
#idx("abstract models for data")
#emph[النماذج المجردة (#en[abstract models])]. وهي تصوغ مواصفات «الدوال زائد الشروط» كما أُوضِح في مثال الأعداد الناطقة أعلاه. لاحظ أنّ الشرط في تمثيل الأعداد الناطقة صُرِّح عنه بدلالة حقائق حول الأعداد الصحيحة (المساواة والقسمة). وبوجه عام، تحدد النماذج المجردة أنواعاً جديدة من كائنات البيانات بدلالة أنواع كائنات البيانات المحددة سابقاً. ولذا يمكن التحقق من التأكيدات حول كائنات البيانات باختزالها إلى تأكيدات حول كائنات البيانات المحددة سابقاً. والمقاربة الأخرى، التي قدمها
#idx("Zilles, Stephen N.")
زيليس في معهد ماساتشوستس للتكنولوجيا (#en[MIT])، و
#idx("Goguen, Joseph")
غوغوين، و
#idx("Thatcher, James W.")
ثاتشر، و
#idx("Wagner, Eric G.")
فاغنر، و
#idx("Wright, Jesse B.")
رايت في #en[IBM] (انظر ثاتشر وفاغنر ورايت 1978)، و
#idx("Guttag, John Vogel")
غوتاغ في تورونتو (انظر غوتاغ 1977)، تُسمّى
#idx("data", sub: "algebraic specification for")
#idx("algebraic specification for data")
#emph[المواصفات الجبرية (#en[algebraic specification])]. وتعتبر «الدوال» كعناصر لنظام جبري مجرد يُحدَّد سلوكه ببديهيات تتوافق مع «شروطنا»، وتستخدم تقنيات الجبر المجرد للتحقق من التأكيدات حول كائنات البيانات. ومُسِحَت كلتا الطريقتين في ورقة ليسكوف وزيليس
#idx("Liskov, Barbara Huberman")
(1975).]

#idx("data", sub: "functional representation of") #idx("functional representation of data")
ويمكن لوجهة النظر هذه أن تعمل ليس فقط لتحديد كائنات البيانات «رفيعة المستوى»، مثل الأعداد الناطقة، بل والكائنات منخفضة المستوى أيضاً.
تأمَّل مفهوم
#idx("pair(s)", sub: "functional representation of")
الزوج، الذي استخدمناه لتحديد أعدادنا الناطقة. لم نذكر في الواقع ما هو الزوج، بل قلنا فقط إنّ اللغة توفر الدوال #py("pair") و #py("head") و #py("tail") للعمل على الأزواج. ولكن الشيء الوحيد الذي نحتاج إلى معرفته حول هذه العمليات الثلاث هو أنه إذا ألصقنا كائنين معاً باستخدام #py("pair")، فيمكننا استعادة الكائنين باستخدام #py("head") و #py("tail").
#idx("pair (primitive function)", sub: "axiom for")
#idx("head (primitive function)", sub: "axiom for")
#idx("tail (primitive function)", sub: "axiom for")
#idx("pair(s)", sub: "axiomatic definition of")
أي أنّ العمليات تحقّق الشرط أنه، لأي كائنين #py("x") و #py("y")، إذا كان #py("z") هو #py("pair(x, y)") فإنّ #py("head(z)") هو #py("x") و #py("tail(z)") هو #py("y"). وبالفعل، ذكرنا أنّ هذه الدوال الثلاث ضُمِّنَت كأوليات في لغتنا. ومع ذلك، فإنّ أي ثلاثية من الدوال تحقّق الشرط أعلاه يمكن استخدامها كأصل لتنفيذ الأزواج. وتُوضَّح هذه النقطة بشكل مذهل بحقيقة أنه كان بإمكاننا تنفيذ #py("pair") و #py("head") و #py("tail") دون استخدام أي هياكل بيانات على الإطلاق بل فقط باستخدام الدوال.
إليك التعاريف:#footnote[الدالة #idx("error (primitive function)", sub: "optional second argument") #py("error") المقدمة في القسم @sec:proc-general-methods تأخذ كمعطى ثاني اختياري سلسلة نصية يُعرَض قبل المعطى الأول — على سبيل المثال، إذا كان #py("m") هو 42: #output(```python Error in line 7: argument not 0 or 1 -- pair: 42 ```)]
#idx("pair (primitive function)", sub: "functional implementation of", decl: true)#idx("head (primitive function)", sub: "functional implementation of", decl: true)#idx("tail (primitive function)", sub: "functional implementation of", decl: true)
#snippet(```python
def pair(x, y):
    def dispatch(m):
        return (x if m == 0
                else y if m == 1
                else error("argument not 0 or 1 -- pair", m))
    return dispatch
def head(z): return z(0)

def tail(z): return z(1)
```)

هذا الاستخدام للدوال لا يتوافق مع أي شيء يشبه فكرتنا البديهية عما يجب أن تكون عليه البيانات. ومع ذلك، كل ما نحتاج إلى القيام به لإظهار أنّ هذه طريقة صالحة لتمثيل الأزواج هو التحقق من أنّ هذه الدوال تحقّق الشرط المعطى أعلاه.

والنقطة الدقيقة التي يجب ملاحظتها هي أنّ القيمة المرجوعة من #py("pair(x, y)") هي دالة — وتحديداً الدالة المُعرَّفة داخلياً #py("dispatch")، والتي تأخذ معطى واحداً وترجع إما #py("x") أو #py("y") اعتماداً على ما إذا كان المعطى 0 أم 1. وبناءً على ذلك، يُعرَّف #py("head(z)") لتطبيق #py("z") على 0. ومن ثمّ، إذا كانت #py("z") هي الدالة المكوَّنة بواسطة #py("pair(x, y)")، فإنّ #py("z") المطبَّقة على 0 ستعطي #py("x"). وهكذا أظهرنا أنّ #py("head(pair(x, y))") يعطي #py("x")، كما هو مطلوب. وبالمثل، يُطبِّق #py("tail(pair(x, y))") الدالة المرجوعة من #py("pair(x, y)") على 1، والتي ترجع #py("y").
ولذلك، فإنّ هذا التنفيذ الدالّي للأزواج تنفيذ صالح، وإذا وصلنا إلى الأزواج باستخدام #py("pair") و #py("head") و #py("tail") فقط، فلا يمكننا تمييز هذا التنفيذ عن تنفيذ يستخدم هياكل بيانات «حقيقية».

والهدف من عرض التمثيل الدالّي للأزواج ليس أنّ لغتنا تعمل بهذه الطريقة (التنفيذ الكفء للأزواج قد يستخدم قائمة بايثون #emph[list] الأصلية) بل أنها يمكن أن تعمل بهذه الطريقة. والتمثيل الدالّي، رغم غرابته، طريقة مناسبة تماماً لتمثيل الأزواج، لأنه يلبي الشروط الوحيدة التي تحتاج الأزواج لتلبيتها. ويُظهر هذا المثال أيضاً أنّ القدرة على التعامل مع الدوال ككائنات توفر تلقائياً القدرة على تمثيل البيانات المركبة. قد يبدو هذا أمراً غريباً الآن، لكن التمثيل الدالّي للبيانات سيلعب دوراً مركزياً في حصيلتنا البرمجية. ويُسمّى هذا النمط من البرمجة غالباً
#idx("message passing")
#emph[تمرير الرسائل (#en[message passing])]\، وسنستخدمه كأداة أساسية في الفصل @chap:state عند التعامل مع قضايا النمذجة والمحاكاة.

#exercise(label-name: <ex:lambda-cons>, [
إليك تمثيلاً دالّياً بديلاً للأزواج. بالنسبة لهذا التمثيل، تحقق من أنّ #py("head(pair(x, y))") يعطي #py("x") لأي كائنين #py("x") و #py("y").
#idx("pair (primitive function)", sub: "functional implementation of", decl: true)#idx("head (primitive function)", sub: "functional implementation of", decl: true)#idx("tail (primitive function)", sub: "functional implementation of")
#snippet(```python
def pair(x, y):
    return lambda m: m(x, y)
def head(z):
    return z(lambda p, q: p)
```)

ما هو التعريف المناظر لـ
#idx("tail (primitive function)", sub: "functional implementation of")
#py("tail")؟
(إرشاد: للتحقق من أنّ هذا يعمل، استخدم نموذج الاستبدال من القسم @sec:substitution-model).
])

#exercise(label-name: <ex:2_5>, [
أظهر أنه يمكننا تمثيل أزواج الأعداد الصحيحة غير السالبة باستخدام الأعداد والعمليات الحسابية فقط إذا مثلنا الزوج $a$ و $b$ كـ عدد صحيح هو حاصل الضرب $2^(a) 3^(b)$. وأعطِ التعاريف المناظرة للدوال #py("pair") و #py("head") و #py("tail").
])

#idx("pair(s)", sub: "functional representation of")

#exercise(label-name: <ex:church-numerals>, [
في حال لم يكن تمثيل الأزواج كدوال (التمرين @ex:lambda-cons) مذهلاً بما فيه الكفاية، تأمَّل أنه، في لغة يمكنها التعامل مع الدوال، يمكننا الاستغناء عن الأعداد (على الأقل فيما يتعلق بالأعداد الصحيحة غير السالبة) بتنفيذ 0 وعملية إضافة 1 كـ:

#snippet(```python
zero = lambda f: lambda x: x

def add_1(n):
    return lambda f: lambda x: f(n(f)(x))
```)

يُعرَف هذا التمثيل بـ
#idx("Church numerals")
#emph[أعداد تشيرش (#en[Church numerals])]\، نسبة إلى مخترعها،
#idx("Church, Alonzo")
ألونزو تشيرش #en[(Alonzo Church)]، المنطقي الذي اخترع حساب $lambda$.

عرف #py("one") و #py("two") مباشرة (وليس بدلالة #py("zero") و #py("add_1")).
(إرشاد: استخدم الاستبدال لتقييم #py("add_1(zero)")).
وأعطِ تعريفاً مباشراً لدالة الجمع #py("plus") (وليس بدلالة التطبيق التكراري لـ #py("add_1")).
])

#idx("data", sub: "functional representation of") #idx("functional representation of data") #idx("data", sub: "meaning of")
