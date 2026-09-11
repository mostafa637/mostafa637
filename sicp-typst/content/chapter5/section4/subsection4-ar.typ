// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([تشغيل المُقيِّم], label-name: <sec:running-evaluator>)

#idx("explicit-control evaluator for Python", sub: "running")

مع تنفيذ مُقيِّم التحكم الصريح نصل إلى نهاية التطوير الذي بدأ في الفصل @chap:fun، والذي استكشفنا فيه نماذج دقيقة متتالية لعملية التقييم
#idx("models of evaluation")
#idx("evaluation", sub: "models of").
بدأنا بنموذج التعويض غير الرسمي نسبياً، ثم مددناه في الفصل @chap:state إلى نموذج البيئة، والذي مكننا من التعامل مع الحالة والتغيير. في المُقيِّم فوق الدائري (#en[metacircular]) في الفصل @chap:meta، استخدمنا #en[Python] نفسها كلغة لجعل بنية البيئة المبنية أثناء تقييم مكون أكثر صراحة.
الآن، مع آلات المسجّلات، أخذنا نظرة فاحصة على آليات المُقيِّم لإدارة التخزين، وتمرير الوسائط، والتحكم. في كل مستوى جديد من الوصف، كان علينا إثارة قضايا وحل غموض لم يكن ظاهراً في المعالجة السابقة الأقل دقة للتقييم. لفهم سلوك مُقيِّم التحكم الصريح، يمكننا محاكاته ومراقبة أدائه.

سنثبت
#idx("explicit-control evaluator for Python", sub: "driver loop")
#idx("driver loop", sub: "in explicit-control evaluator")
حلقة مشغل (#en[driver loop]) في آلة المُقيِّم الخاصة بنا. وتلعب هذه دور الدالة #py("driver_loop") في القسم @sec:running-eval. سيكرر المُقيِّم طباعة محث، وقراءة برنامج، وتقييم البرنامج بالذهاب إلى #py("eval_dispatch")، وطباعة النتيجة.
إذا لم يتم إدخال أي شيء عند المحث، نقفز إلى التسمية #py("evaluator_done")، وهي نقطة الإدخال الأخيرة في وحدة التحكم.
تشكل التعليمات التالية بداية تسلسل وحدة التحكم لمُقيِّم التحكم الصريح:#footnote[نفترض هنا أن #py("user_read") و #py("parse") والعمليات الطباعية المختلفة متاحة كعمليات آلة أولية، وهو أمر مفيد لمُحاكاتنا، ولكنه غير واقعي تمامًا في الممارسة العملية. هذه في الواقع عمليات معقدة للغاية. في الممارسة العملية، فإن القراءة والطباعة ستُنفذ باستخدام عمليات إدخال وإخراج منخفضة المستوى مثل نقل الأحرف المنفردة من وإلى جهاز.]
#idx("prompts", sub: "explicit-control evaluator")#idx("readevaluateprintloop", decl: true)#idx("printresult", decl: true)
#syntax("
\"read_evaluate_print_loop\",
  perform(list(op(\"initialize_stack\"))),
  assign(\"comp\", list(op(\"user_read\"),
                      constant(\"EC-evaluate input:\"))),
  assign(\"comp\", list(op(\"parse\"), reg(\"comp\"))),
  test(list(op(\"is_null\"), reg(\"comp\"))),
  branch(label(\"evaluator_done\")),
  assign(\"env\", list(op(\"get_current_environment\"))),
  assign(\"val\", list(op(\"scan_out_declarations\"), reg(\"comp\"))),
  save(\"comp\"),    // حتى نتمكن من استخدامه لحفظ قيم *unassigned* مؤقتاً
  assign(\"comp\", list(op(\"list_of_unassigned\"), reg(\"val\"))),
  assign(\"env\", list(op(\"extend_environment\"),
                     reg(\"val\"), reg(\"comp\"), reg(\"env\"))),
  perform(list(op(\"set_current_environment\"), reg(\"env\"))),
  restore(\"comp\"), // البرنامج
  assign(\"continue\", label(\"print_result\")),
  go_to(label(\"eval_dispatch\")),
\"print_result\",
  perform(list(op(\"user_print\"),
               constant(\"EC-evaluate value:\"), reg(\"val\"))),
  go_to(label(\"read_evaluate_print_loop\")),
  ")

نخزن البيئة الحالية، وهي البيئة العالمية في البداية، في المتغير #py("current_environment") ونحدثها في كل مرة حول الحلقة لتذكر الإعلانات السابقة.
العمليتان #py("get_current_environment") و #py("set_current_environment") تحصلان ببساطة وتضبطان هذا المتغير.
#idx("getcurrentenvironment", decl: true)#idx("setcurrentenvironment", decl: true)
#snippet(```python
let current_environment = the_global_environment;

function get_current_environment() {
    return current_environment;
}

function set_current_environment(env) {
    current_environment = env;
}
```)

عندما نرافق
#idx("error handling", sub: "in explicit-control evaluator")
#idx("explicit-control evaluator for Python", sub: "error handling")
خطأً في دالة (مثل خطأ "نوع دالة غير معروف" المشار إليه عند #py("apply_dispatch"))، نطبع رسالة خطأ ونعود إلى حلقة المشغل.#footnote[هناك أخطاء أخرى نود أن يتعامل معها المُفسِّر، لكنها ليست بهذه البساطة. انظر التمرين @ex:interp-errors.]
#idx("unknowncomponenttype", decl: true)#idx("unknownfunctiontype", decl: true)#idx("signalerror", decl: true)
#syntax("
\"unknown_component_type\",
  assign(\"val\", constant(\"unknown syntax\")),
  go_to(label(\"signal_error\")),

\"unknown_function_type\",
  restore(\"continue\"), // تنظيف المكدس (من ", $mono("apply_dispatch")$, ")
  assign(\"val\", constant(\"unknown function type\")),
  go_to(label(\"signal_error\")),

\"signal_error\",
  perform(list(op(\"user_print\"),
               constant(\"EC-evaluator error:\"), reg(\"val\"))),
  go_to(label(\"read_evaluate_print_loop\")),
      ")

لأغراض المحاكاة، نُهيئ المكدس في كل مرة عبر حلقة المشغل، نظرًا لأنه قد لا يكون فارغًا بعد أن يقاطع خطأ (مثل اسم غير مُعلن) تقييمًا.#footnote[كان بإمكاننا إجراء تهيئة المكدس فقط بعد الأخطاء، لكن القيام بذلك في حلقة المشغل سيكون مريحًا لمراقبة أداء المُقيِّم، كما هو موضح أدناه.]

#idx("explicit-control evaluator for Python", sub: "controller")

إذا قمنا بتجميع جميع أجزاء الكود المقدمة في الأقسام من @sec:eceval-core إلى @sec:running-evaluator، فيمكننا إنشاء
#idx("explicit-control evaluator for Python", sub: "machine model")
نموذج آلة مُقيِّم يمكننا تشغيله باستخدام محاكي آلة المسجّلات في القسم @sec:simulator.

#idx("eceval", decl: true)
#syntax("
const eceval = make_machine(list(\"comp\", \"env\", \"val\", \"fun\",
                                 \"argl\", \"continue\", \"unev\"),
                            eceval_operations,
                            list(\"read_evaluate_print_loop\",
                                 ", metaphrase[وحدة تحكم الآلة بالكامل كما هي معطاة أعلاه], "
                                 \"evaluator_done\"));
      ")

يجب علينا تعريف دوال #en[Python] لمُحاكاة العمليات المستخدمة كأوليات بواسطة المُقيِّم. هذه هي نفس الدوال التي استخدمناها للمُقيِّم فوق الدائري في القسم @sec:mc-eval، جنبًا إلى جنب مع عدد قليل من الدوال الإضافية المحددة في الحواشي السفلية في جميع أنحاء القسم @sec:eceval.

#syntax("
const eceval_operations = list(list(\"is_literal\", is_literal),
                               ", $⟨italic("قائمة") med thin italic("كاملة") med thin italic("من") med italic("العمليات") med thin italic("لآلة") med thin italic("eceval")⟩$, ");
      ")

أخيرًا، يمكننا تهيئة البيئة العالمية وتشغيل المُقيِّم:
#idx("theglobalenvironment", decl: true)
#snippet(```python
const the_global_environment = setup_environment();
start(eceval);
```)

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
function append(x, y) {
    return is_null(x)
           ? y
           : pair(head(x), append(tail(x), y));
}
```)

#output(```python
function append(x, y) {
    return is_null(x)
           ? y
           : pair(head(x), append(tail(x), y));
}
```)

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
append(list("a", "b", "c"), list("d", "e", "f"));
```)

#output(```python
append(list("a", "b", "c"), list("d", "e", "f"));
```)

بالطبع، تقييم البرامج بهذه الطريقة سيتطلب وقتًا أطول بكثير مما لو كنا قد أدخلناها مباشرة في #en[Python]، بسبب المستويات المتعددة للمحاكاة المعنية. تُقيّم برامجنا بواسطة آلة مُقيِّم التحكم الصريح، والتي تُحاكى بواسطة برنامج #en[Python]، والذي يُقيّم هو نفسه بواسطة مُفسِّر #en[Python].

#idx("explicit-control evaluator for Python", sub: "running")

#subheading([مراقبة أداء المُقيِّم])

#idx("explicit-control evaluator for Python", sub: "monitoring performance (stack use)")

يمكن أن تكون المحاكاة أداة قوية لتوجيه تنفيذ المُقيِّمات.
#idx("simulation", sub: "as machine-design tool")
تسمح المحاكاة بسهولة ليس فقط لاستكشاف التنوعات في تصميم آلة المسجّلات ولكن أيضًا لمراقبة أداء المُقيِّم المحاكى. على سبيل المثال، أحد العوامل المهمة في الأداء هو مدى كفاءة المُقيِّم في استخدام المكدس. يمكننا ملاحظة عدد عمليات المكدس المطلوبة لتقييم برامج مختلفة عن طريق تعريف آلة مسجّلات المُقيِّم باستخدام النسخة من المحاكي التي تجمع إحصائيات حول استخدام المكدس (القسم @sec:monitor)، وإضافة تعليمة عند نقطة إدخال #py("print_result") للمُقيِّم لطباعة الإحصائيات:
#idx("printresult", sub: "monitored-stack version", decl: true)
#syntax("
\"print_result\",
  perform(list(op(\"print_stack_statistics\"))), // التعليمة المضافة
  // الباقي هو نفسه كما كان من قبل
  perform(list(op(\"user_print\"),
               constant(\"EC-evaluate value:\"), reg(\"val\"))),
  go_to(label(\"read_evaluate_print_loop\")),
      ")

تبدو التفاعلات مع المُقيِّم الآن كالتالي:

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
function factorial (n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
```)

#output(```python
function factorial (n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
```)

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
factorial(5);
```)

#output(```python
factorial(5);
```)

لاحظ أن حلقة مشغل المُقيِّم تعيد تهيئة المكدس عند بداية كل تفاعل، بحيث ترجع الإحصائيات المطبوعة فقط إلى عمليات المكدس المستخدمة لتقييم البرنامج السابق.

#exercise(label-name: <ex:tail-rec-fact>, [
استخدم المكدس الخاضع للمراقبة لاستكشاف خاصية
#idx("explicit-control evaluator for Python", sub: "tail recursion")
#idx("tail recursion", sub: "explicit-control evaluator and")
العودية الذيلية للمُقيِّم (القسم @sec:tail-recursion-return). ابدأ المُقيِّم وعرّف دالة #py("factorial") التكرارية
#idx("factorial", sub: "stack usage, interpreted")
من القسم @sec:recursion-and-iteration:

#snippet(```python
function factorial(n) {
    function iter(product, counter) {
        return counter > n
               ? product
               : iter(counter * product,
                      counter + 1);
    }
    return iter(1, 1);
}
```)

شغل الدالة مع بعض القيم الصغيرة لـ $n$. سجل أقصى عمق للمكدس وعدد عمليات الدفع المطلوبة لحساب $n!$ لكل من هذه القيم.

+ ستجد أن العمق الأقصى المطلوب لتقييم $n!$ يستقل عن $n$. ما هو هذا العمق؟
+ حدد من بياناتك صيغة بدلالة $n$ لإجمالي عدد عمليات الدفع المستخدمة في تقييم $n!$ لأي $n \ge 1$. لاحظ أن عدد العمليات المستخدمة هو دالة خطية في $n$ وبالتالي يتحدد بثابتين.
])

#exercise(label-name: <ex:rec-fact>, [
للمقارنة مع التمرين @ex:tail-rec-fact، استكشف سلوك الدالة التالية لحساب
#idx("factorial", sub: "stack usage, interpreted")
المضروب عودياً:

#snippet(```python
function factorial(n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
```)

عن طريق تشغيل هذه الدالة مع المكدس الخاضع للمراقبة، حدد، كدالة في $n$، العمق الأقصى للمكدس وإجمالي عدد عمليات الدفع المستخدمة في تقييم $n!$ لـ $n \ge 1$. (مرة أخرى، ستكون هذه الدوال خطية). لخص تجاربك بملء الجدول التالي بالتعبيرات المناسبة بدلالة $n$:

#blockquote[$ mat(delim: #none, , "العمق الأقصى", "عدد عمليات الدفع"; "المضروب العودي", , ; "المضروب التكراري", , ) $]

العمق الأقصى هو مقياس لمقدار المساحة المستخدمة بواسطة المُقيِّم في إجراء الحساب، وعدد عمليات الدفع يتناسب جيداً مع الوقت المطلوب.
])

#exercise(label-name: <ex:5_29>, [
عدل تعريف المُقيِّم بتغيير #py("ev_return") كما هو موضح في القسم @sec:tail-recursion-return بحيث لا يعود المُقيِّم
#idx("explicit-control evaluator for Python", sub: "tail recursion")
#idx("tail recursion", sub: "explicit-control evaluator and")
عودياً ذيلياً. أعد تشغيل تجاربك من التمرينين @ex:tail-rec-fact و @ex:rec-fact لإظهار أن كلا نسختي دالة #py("factorial") تطلب الآن مساحة تنمو خطياً مع مدخلاتها.
])

#exercise(label-name: <ex:rec-fib>, [
راقب عمليات المكدس في حساب فيبوناتشي العودي الشجري:
#idx("fib", sub: "stack usage, interpreted")
#idx("fib", sub: "tree-recursive version", decl: true)
#snippet(```python
function fib(n) {
    return n < 2 ? n : fib(n - 1) + fib(n - 2);
}
```)

+ أعطِ صيغة بدلالة $n$ للعمق الأقصى للمكدس المطلوب لحساب $"Fib"(n)$ لـ $n \ge 2$. تلميح: في القسم @sec:tree-recursion جادلنا بأن المساحة المستخدمة بواسطة هذه العملية تنمو خطياً مع $n$.
+ أعطِ صيغة لإجمالي عدد عمليات الدفع المستخدمة لحساب $"Fib"(n)$ لـ $n \ge 2$. يجب أن تجد أن عدد عمليات الدفع (والذي يتناسب جيداً مع الوقت المستخدم) ينمو أسياً مع $n$. تلميح: لتكن $S(n)$ عدد عمليات الدفع المستخدمة في حساب $"Fib"(n)$. يجب أن تكون قادرًا على الجدال بأن هناك صيغة تعبر عن $S(n)$ بدلالة $S(n-1)$ و $S(n-2)$ وبعض الثابت غير المباشر $k$ المستقل عن $n$. أعطِ الصيغة، واذكر ما هو $k$. ثم أظهر أنه يمكن التعبير عن $S(n)$ ك$a "Fib"(n+1) + b$ وأعطِ قيم $a$ و $b$.
])

#idx("explicit-control evaluator for Python", sub: "monitoring performance (stack use)")

#exercise(label-name: <ex:interp-errors>, [
مُقيِّمنا حاليًا يمسك ويصدر إشارات لـ نوعين فقط من
#idx("error handling", sub: "in explicit-control evaluator")
#idx("explicit-control evaluator for Python", sub: "error handling")
الأخطاء—أنواع المكونات غير المعروفة وأنواع الدوال غير المعروفة. الأخطاء الأخرى ستخرجنا من حلقة القراءة والتقييم والطباعة للمُقيِّم.
عندما نشغل المُقيِّم باستخدام محاكي آلة المسجّلات، يتم إمساك هذه الأخطاء بواسطة نظام #en[Python] الأساسي. هذا يشبه تعطل الحاسوب عندما يرتكب برنامج مستخدم خطأً.#footnote[يتجلى هذا، على سبيل المثال، في صورة "ذعر النواة" (#en[kernel panic]) أو "شاشة الموت الزرقاء" أو حتى إعادة التشغيل. إعادة التشغيل التلقائي هو نهج عادة ما يُستخدم في الهواتف والأجهزة اللوحية. معظم أنظمة التشغيل الحديثة تقوم بعمل جيد في منع برامج المستخدم من التسبب في تعطل الآلة بأكملها.]
إنه مشروع كبير لجعل نظام خطأ حقيقي يعمل، ولكن الأمر يستحق الجهد لفهم ما هو معني هنا.

+ الأخطاء التي تحدث في عملية التقييم، مثل محاولة الوصول إلى اسم غير مربوط، يمكن إمساكها بتغيير عملية البحث لجعلها تُرجع كود حالة متميز، والذي لا يمكن أن يكون قيمة محتملة لأي اسم مستخدم. يمكن للمُقيِّم اختبار كود الحالة هذا ثم القيام بما يلزم للانتقال إلى #py("signal_error"). أوجد جميع الأماكن في المُقيِّم حيث يكون مثل هذا التغيير ضروريًا وأصلحها. هذا عمل كثير.
+ الأسوأ من ذلك بكثير هو مشكلة التعامل مع الأخطاء التي يصدر إشارة بها تطبيق الدوال الأولية مثل محاولة القسمة على صفر أو محاولة استخراج الـ #py("head") لسلسلة نصية. في نظام مكتوب بشكل احترافي وعالي الجودة، يتم فحص كل تطبيق أولي للأمان كجزء من الأولية. على سبيل المثال، يمكن لكل استدعاء لـ #py("head") فحص ما إذا كان الوسيط زوجًا أولاً. إذا لم يكن الوسيط زوجًا، فسيرجع التطبيق كود حالة متميز إلى المُقيِّم، والذي سيرفع الفشل عندئذٍ. يمكننا الترتيب لذلك في محاكي آلة المسجّلات الخاص بنا عن طريق جعل كل دالة أولية تفحص القابلية للتطبيق وتُرجع كود حالة متميز مناسب عند الفشل. عندئذٍ يمكن لكود #py("primitive_apply") في المُقيِّم فحص كود الحالة والانتقال إلى #py("signal_error") إذا لزم الأمر. ابنِ هذه البنية واجعلها تعمل. هذا مشروع رئيسي.
])

#idx("explicit-control evaluator for Python")
