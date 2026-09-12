// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([البحث و#py("amb")], label-name: <sec:amb>)

لتوسيع #en[Python] لدعم عدم الحتمية، نقدم شكلًا نحويًا جديدًا
#idx("amb", decl: true)
يُسمى #py("amb").#footnote[فكرة #py("amb") للبرمجة غير الحتمية وصفها لأول مرة عام 1961
#idx("McCarthy, John")
#en[John McCarthy] (انظر #en[McCarthy 1967]).]
التعبير
#py("amb(")$e_(1), space e_(2), dots.h , e_(n)$#py(")")
يرجع قيمة أحد التعبيرات الـ $n$ المسماة $e_(i)$ بشكل «التباس». على سبيل المثال، التعبير

#snippet(```python
llist(amb(1, 2, 3), amb("a", "b"))
```)

يمكن أن تترتب عليه ست قيم ممكنة:

#snippet(```python
llist(1, "a") llist(1, "b") llist(2, "a")
llist(2, "b") llist(3, "a") llist(3, "b")
```)

تعبير #py("amb") باختيار واحد ينتج عنه قيمة عادية (واحدة).

وتعبير #py("amb") بدون خيارات — التعبير
#py("amb()") — هو
#idx("failure, in nondeterministic computation")
تعبير ليس له قيم مقبولة. من الناحية التشغيلية، يمكننا التفكير في #py("amb()") كتعبير يتسبب عند تقييمه في «فشل» الحساب: حيث يُجهَض الحساب ولا تُنتَج أي قيمة. وباستخدام هذه الفكرة، يمكننا التعبير عن الشرط القائل بأن تعبير محمول معين #py("p") يجب أن يكون صحيحًا على النحو التالي:

#idx("require", decl: true)
#snippet(```python
def require(p):
    if not p:
        amb()
    else:
        pass
```)

باستخدام #py("amb") و#py("require")، يمكننا تنفيذ الدالة #py("an_element_of") المستخدمة أعلاه:

#idx("anelementof", decl: true)
#snippet(```python
def an_element_of(items):
    require( not is_none(items))
    return amb(head(items), an_element_of(tail(items)))
```)

يفشل تطبيق #py("an_element_of") إذا كانت القائمة فارغة. وإلا فإنه يرجع إما العنصر الأول في القائمة أو عنصرًا مأخوذًا من باقي القائمة بشكل التباسي.

يمكننا أيضًا التعبير عن نطاقات لانهائية من الخيارات. فالدالة التالية قد ترجع أي عدد صحيح أكبر من أو يساوي عددًا معطى $n$:

#idx("anintegerstartingfrom", decl: true)
#snippet(```python
def an_integer_starting_from(n):
    return amb(n, an_integer_starting_from(n + 1))
```)

هذا يشبه دالة التدفق #py("integers_starting_from") الموصوفة في القسم @sec:infinite-streams، ولكن مع فارق مهم: دالة التدفق ترجع كائنًا يمثل تتابع جميع الأعداد الصحيحة المبتدئة من $n$، بينما ترجع دالة #py("amb") عددًا صحيحًا واحدًا.#footnote[في الواقع، التمييز بين إرجاع اختيار واحد بشكل غير حتمي وإرجاع جميع الخيارات يعتمد إلى حد ما على وجهة نظرنا. من منظور الكود الذي يستخدم القيمة، فإن الاختيار غير الحتمي يرجع قيمة واحدة. ومن منظور المبرمج الذي يصمم الكود، فإن الاختيار غير الحتمي يرجّح إرجاع جميع القيم الممكنة، ويتفرع الحساب بحيث يُستكشَف كل قيمة بشكل منفصل.]

مجردًا، يمكننا تخيل أن تقييم تعبير #py("amb") يتسبب في
#idx("time", sub: "in nondeterministic computing")
انقسام الزمن إلى فروع، حيث يستمر الحساب في كل فرع باستخدام إحدى القيم الممكنة للتعبير. ونقول إن #py("amb") تمثل
#idx("nondeterministic choice point")
#emph[نقطة اختيار غير حتمية]. وإذا كان لدينا آلة بعدد كافٍ من المعالجات التي يمكن تخصيصها ديناميكيًا، فيمكننا تنفيذ البحث بطريقة مباشرة. وسيستمر التنفيذ كما هو الحال في آلة تتابعية، حتى يتم اعتراض تعبير #py("amb"). وعند هذه النقطة، يتم تخصيص المزيد من المعالجات وتهيئتها لمواصلة جميع التنفيذات المتوازية التي تنطوي عليها الاختيارات. وسيستمر كل معالج تتابعيًا كما لو كان الاختيار الوحيد، حتى ينتهي إما باعتراض فشل، أو يزيد من التقسيم الفرعي، أو ينتهي.#footnote[قد يعترض المرء بأن هذه آلية غير كفء على نحو يائس. قد تتطلب ملايين المعالجات لحل مشكلة منقوشة بسهولة بهذه الطريقة، وفي معظم الأوقات يكون معظم المعالجات خاملًا. وينبغي أخذ هذا الاعتراض في سياق التاريخ. كانت الذاكرة تُعتبر بضاعة باهظة الثمن بهذه الشاكلة.
#idx("memory", sub: "in 1965")
في عام 1965 كانت تكلفة ميغابايت من ذاكرة الوصول العشوائي RAM نحو 400,000 دولار. والآن يمتلك كل حاسوب شخصي عدة غيغابايت من RAM، وفي معظم الأوقات يكون معظم ذاكرة RAM غير مستخدم. ومن الصعب التقليل من تكلفة الإلكترونيات المنتجة بكميات كبيرة.]

من ناحية أخرى، إذا كان لدينا آلة يمكنها تنفيذ عملية واحدة فقط (أو بضع عمليات متزامنة)، فيجب أن نأخذ البدائل
#idx("failure, in nondeterministic computation", sub: "searching and")
تتابعيًا. يمكن للمرء تخيل تعديل المُقيِّم لاختيار فرع عشوائي لاتباعه كلما واجه نقطة اختيار. ولكن الاختيار العشوائي قد يؤدي بسهولة إلى قيم فاشلة. قد نحاول تشغيل المُقيِّم مرارًا وتكرارًا، متبعين خيارات عشوائية وآملين في العثور على قيمة غير فاشلة، ولكن من الأفضل إجراء
#idx("systematic search")
#idx("search", sub: "systematic")
#emph[بحث منتظم] لكل مسارات التنفيذ الممكنة. ومُقَيِّم #py("amb") الذي سنطوره ونعمل معه في هذا القسم ينفذ بحثًا منتظمًا كالتالي: عندما يواجه المُقيِّم تطبيقًا لـ #py("amb")، فإنه يختار في البداية البديل الأول. وقد يؤدي هذا الاختيار بحد ذاته إلى اختيار إضافي. وسيختار المُقيِّم دائمًا في البداية البديل الأول عند كل نقطة اختيار. وإذا نتج عن اختيار ما فشل، فإن المُقيِّم
#idx("automagically")
#idx("backtracking")
#emph[يتراجع] تلقائيًا وسحريًا#footnote[سحريًا وتلقائيًا (#en[automagically]): «تلقائياً، ولكن بطريقة لا يرغب المتحدث، لسببٍ ما (عادةً لأنها معقدة أكثر من اللازم، أو قبيحة، أو ربما حتى تافهة للغاية)، في شرحها.»
#idx("Steele, Guy Lewis Jr.")
(#en[Steele 1983]،
#idx("Raymond, Eric")
#en[Raymond 1996])] إلى أحدث نقطة اختيار ويجرب البديل التالي.
وإذا نَفَدَت البدائل عند أي نقطة اختيار، فسيتراجع المُقيِّم إلى نقطة الاختيار السابقة ويستأنف من هناك. وتؤدي هذه العملية إلى استراتيجية بحث تُعرف باسم
#idx("depth-first search")
#idx("search", sub: "depth-first")
#emph[البحث بالعمق أولاً] (#en[depth-first search]) أو
#idx("chronological backtracking")
#emph[التتبع التراجعي الزمني] (#en[chronological backtracking]).#footnote[إن دمج استراتيجيات البحث التلقائي
#idx("automatic search", sub: "history of")
في لغات البرمجة كان له تاريخ طويل ومتقلب. جاءت الاقتراحات الأولى لإمكانية ترميز الخوارزميات غير الحتمية بأناقة في لغة برمجة مع بحث وتتبع تراجعي تلقائي من #idx("Floyd, Robert") #en[Robert Floyd (1967)]. واخترع #idx("Hewitt, Carl Eddie") #en[Carl Hewitt (1969)] لغة برمجة تُسمى #idx("Planner") #en[Planner] ساندت صراحة التتبع التراجعي الزمني التلقائي، مجهزة باستراتيجية بحث بالعمق أولاً كجزء بنيوي. ونفذ #idx("Sussman, Gerald Jay") #idx("Winograd, Terry") #idx("Charniak, Eugene") #en[Sussman] و#en[Winograd] و#en[Charniak (1971)] مجموعة فرعية من هذه اللغة يُطلق عليها #idx("MicroPlanner") #en[MicroPlanner]، والتي استُخدمت لدعم العمل في حل المشكلات والتخطيط للروبوتات. وأدت أفكار مماثلة، ناتجة عن المنطق وإثبات النظريات، إلى نشأة لغة #idx("Prolog") #en[Prolog] الأنيقة في إدنبرة ومارسيليا (والتي سنناقشها في القسم @sec:logic-programming). وبعد إحباط كافٍ من البحث التلقائي، طور #idx("McDermott, Drew") #idx("Sussman, Gerald Jay") #en[McDermott] و#en[Sussman (1972)] لغة تُسمى #idx("Conniver") #en[Conniver]، والتي تضمنت آليات لوضع استراتيجية البحث تحت سيطرة المبرمج. لكن هذا ثبت أنه غير عملي، وجد #idx("Sussman, Gerald Jay") #idx("Stallman, Richard M.") #en[Sussman] و#en[Stallman (1975)] نهجًا أكثر سهولة في الانقياد أثناء البحث في طرق التحليل الرمزي للدوائر الكهربائية. وطورا مخطط تتبع تراجعي غير زمني كان قائمًا على تتبع التبعيات المنطقية المترابطة بالحقائق، وهي تقنية أصبحت يُطلق عليها اسم #idx("dependency-directed backtracking") #emph[التتبع التراجعي الموجه بالتبعيات]. وعلى الرغم من أن طريقتهم كانت معقدة، إلا أنها أنتجت برامج كفؤة بدرجة معقولة لأنها لم تقم ببحث زائد قليل الفائدة. وعمم #idx("Doyle, Jon") #en[Doyle (1979)] و#idx("McAllester, David Allen") #en[McAllester (1978, 1980)] ونقحا طرق #en[Stallman] و#en[Sussman]، مظهرين نمطًا جديدًا لصياغة البحث يُسمى الآن #idx("truth maintenance") #emph[صيانة الحقيقة] (#en[truth maintenance]). وتستخدم العديد من أنظمة حل المشكلات شكلاً من أشكال نظام صيانة الحقيقة كطبقة أساسية. انظر #idx("Forbus, Kenneth D.") #idx("de Kleer, Johan") #en[Forbus and de Kleer 1993] لمناقشة الطرق الأنيقة لبناء أنظمة صيانة الحقيقة والتطبيقات التي تستخدمها. ويصف #idx("Zabih, Ramin") #idx("McAllester, David Allen") #idx("Chapman, David") #en[Zabih, McAllester, and Chapman 1987] توسيعًا
#idx("Scheme", sub: "nondeterministic extension of")
غير حتمي لـ #en[Scheme] يقوم على #py("amb")؛ وهو يشبه المفسر الموصوف في هذا القسم ولكنه أكثر تطورًا، لأنه يستخدم التتبع التراجعي الموجه بالتبعيات بدلاً من التتبع التراجعي الزمني. ويقدم #idx("Winston, Patrick Henry") #en[Winston 1992] مقدمة لكلا نوعي التتبع التراجعي.]<foot:backtrack>

#subheading([حلقة المحرك])

تمتلك
#idx("driver loop", sub: "in nondeterministic evaluator")
حلقة المحرك لمُقَيِّم #py("amb") بعض الخصائص غير العادية. فهي تقرأ برنامجًا وتطبع قيمة أول تنفيذ غير فاشل، كما في مثال #py("prime_sum_pair") الموضح أعلاه. وإذا أردنا رؤية قيمة التنفيذ الناجح التالي، فيمكننا طلب التراجع من المفسر ومحاولة إنتاج تنفيذ ثانٍ غير فاشل.

يُشار إلى ذلك بكتابة
#idx("retry", decl: true)
#py("retry").
وإذا أُعْطِيَ أي إدخال آخر بخلاف #py("retry")، فسيبدأ المفسر مشكلة جديدة، ملقيًا بالبدائل غير المستكشفة في المشكلة السابقة.

إليك تفاعلاً نموذجيًا:

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
prime_sum_pair(llist(1, 3, 5, 8), llist(20, 35, 110))
```)

#output(```python
prime_sum_pair(llist(1, 3, 5, 8), llist(20, 35, 110))
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
retry
```)

#output(```python
retry
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
retry
```)

#output(```python
retry
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
retry
```)

#output(```python
retry
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
prime_sum_pair(llist(19, 27, 30), llist(11, 36, 58))
```)

#output(```python
prime_sum_pair(llist(19, 27, 30), llist(11, 36, 58))
```)

#exercise(label-name: <ex:amb-pythag-triples>, [
اكتب دالة #py("an_integer_between") ترجع عددًا صحيحًا بين حدين معطَيَيْن. يمكن استخدام هذا لتنفيذ دالة تجد
#idx("Pythagorean triples", sub: "with nondeterministic programs")
#idx("nondeterministic programs", sub: "Pythagorean triples")
ثلاثيات فيثاغورس، أي ثلاثيات من الأعداد الصحيحة $(i, j, k)$ بين الحدين المعطَيَيْن بحيث تكون $i lt.eq j$ و $i^(2) + j^(2) = k^(2)$، على النحو التالي:

#snippet(```python
def a_pythogorean_triple_between(low, high):
    i = an_integer_between(low, high)
    j = an_integer_between(i, high)
    k = an_integer_between(j, high)
    require(i * i + j * j == k * k)
    return llist(i, j, k)
```)
])

#exercise(label-name: <ex:amb-pythag-triples_2>, [
ناقش التمرين @ex:stream-pythagorean-triples كيفية توليد تدفق #emph[جميع]
#idx("Pythagorean triples", sub: "with nondeterministic programs")
#idx("nondeterministic programs", sub: "Pythagorean triples")
ثلاثيات فيثاغورس، دون وجود حد أعلى لحجم الأعداد الصحيحة المراد البحث عنها. وضح لماذا لا تُعد بساطة استبدال #py("an_integer_between") بـ #py("an_integer_starting_from") في الدالة الموجودة في التمرين @ex:amb-pythag-triples طريقة كافية لتوليد ثلاثيات فيثاغورس عشوائية. اكتب دالة تحقق ذلك بالفعل. (أي اكتب دالة يؤدي فيها إدخال #py("retry") مرارًا وتكرارًا إلى توليد جميع ثلاثيات فيثاغورس مبدئيًا من حيث المبدأ.)
])

#exercise(label-name: <ex:amb-pythag-triples_3>, [
يدعي #en[Ben Bitdiddle] أن الطريقة التالية لتوليد
#idx("Pythagorean triples", sub: "with nondeterministic programs")
#idx("nondeterministic programs", sub: "Pythagorean triples")
ثلاثيات فيثاغورس أكثر كفاءة من تلك الموجودة في التمرين @ex:amb-pythag-triples. هل هو محق؟ (إرشاد: تأمل عدد الاحتمالات التي يجب استكشافها.)

#snippet(```python
def a_pythagorean_triple_between(low, high):
    i = an_integer_between(low, high)
    hsq = high * high
    j = an_integer_between(i, high)
    ksq = i * i + j * j
    require(hsq >= ksq)
    k = math_sqrt(ksq)
    require(is_integer(k))
    return llist(i, j, k)
```)
])
