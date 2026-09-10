// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([استغلال نموذج التدفقات], label-name: <sec:exploiting-streams>)

يمكن أن تكون التدفقات ذات التقييم المؤجل أداة نمذجة قوية، حيث توفر العديد من فوائد الحالة المحلية والإسناد. علاوة على ذلك، فإنها تتجنب بعض التعقيدات النظرية التي ترافق إدخال الإسناد إلى لغة البرمجة.

يمكن أن تكون مقاربة التدفقات كاشفة لأنها تسمح لنا ببناء أنظمة ذات حدود
#idx("modularity", sub: "streams and")
نمطية مختلفة عن الأنظمة المنظمة حول الإسناد لمتغيرات الحالة. على سبيل المثال، يمكننا التفكير في السلسلة الزمنية بأكملها (أو الإشارة) كـ محط اهتمام، بدلاً من قيم متغيرات الحالة في اللحظات الفردية. وهذا يجعل من المريح دمج ومقارنة مكونات الحالة من لحظات مختلفة.

#subheading([صياغة التكرارات كعمليات تدفق])

#idx("iterative process", sub: "as a stream process")

في القسم @sec:recursion-and-iteration، قدمنا العمليات التكرارية، والتي تستمر عن طريق تحديث متغيرات الحالة. ونعلم الآن أنه يمكننا تمثيل الحالة كـ تدفق قيم "خالٍ من الزمان" بدلاً من مجموعة من المتغيرات ليتم تحديثها. دعنا نتبنى وجهة النظر هذه في إعادة زيارة دالة الجذر التربيعي من القسم @sec:sqrt. تذكر أن الفكرة هي توليد تسلسل من التخمينات الأفضل فـ الأفضل للجذر التربيعي لـ $x$ بتطبيق الدالة التي تحسن التخمينات مراراً وتكراراً:

#snippet(```python
def sqrt_improve(guess, x):
    return average(guess, x / guess)
```)

في دالتنا الأصلية
#idx("square root", sub: "stream of approximations")
#py("sqrt")،
جعلنا هذه التخمينات تكون القيم المتتالية لمتغير الحالة. وبدلاً من ذلك، يمكننا توليد التدفق اللانهائي للتخمينات، بدءاً بتخمين أولي قدره 1:
#idx("sqrtstream", decl: true)
#snippet(```python
def sqrt_stream(x):
    return pair(1, lambda: stream_map(lambda guess: sqrt_improve(guess, x),
                                      sqrt_stream(x)))
```)

#snippet(```python
print(display_stream(sqrt_stream(2)))
```)

#output(```python
print(display_stream(sqrt_stream(2)))
```)

يمكننا توليد المزيد والمزيد من حدود التدفق للحصول على تخمينات أفضل وأفضل. وإذا أردنا، يمكننا كتابة دالة تستمر في توليد الحدود حتى تصبح الإجابة جيدة بدرجة كافية.
(انظر التمرين @ex:stream-limit.)

تكرار آخر يمكننا التعامل معه بالطريقة نفسها هو توليد تقريب لـ
#idx("π (pi)", sub: "Leibniz's series for", sort: "pi")
#idx("Leibniz, Baron Gottfried Wilhelm von", sub: "series for π")
#idx("π (pi)", sub: "stream of approximations", sort: "pi")
#idx("series, summation of", sub: "with streams")
#idx("summation of a series", sub: "with streams")
#idx("infinite stream(s)", sub: "to sum a series")
$pi$، بناءً على المتسلسلة المتناوبة التي رأيناها في القسم @sec:procedures-as-parameters:

$ mat(delim: #none, frac( pi , 4), =, 1-frac(1, 3)+frac(1, 5)-frac(1, 7)+ dots.c) $

نولد أولاً تدفق الحدود المجمعة للمتسلسلة (مقلوبات الأعداد الفردية، بإشارات متناوبة). ثم نأخذ تدفق مجاميع المزيد والمزيد من الحدود (باستخدام الدالة #py("partial_sums") من التمرين @ex:partial-sums) ونقيس النتيجة بـ 4:
#idx("pistream", decl: true)
#snippet(```python
def pi_summands(n):
    return pair(1 / n, lambda: stream_map(lambda x: -x, pi_summands(n + 2)))

pi_stream = scale_stream(partial_sums(pi_summands(1)), 4)
```)

#snippet(```python
print(display_stream(pi_stream))
```)

#output(```python
print(display_stream(pi_stream))
```)

يعطينا هذا تدفقاً من تقريبات أفضل وأفضل لـ $pi$، على الرغم من أن التقريبات تتطابق بطيئة نوعاً ما. تحصر ثمانية حدود من التسلسل قيمة $pi$ بين 3.284 و 3.017.

حتى الآن، استخدامنا لمقاربة تدفق الحالات لا يختلف كثيراً عن تحديث متغيرات الحالة. لكن التدفقات تمنحنا فرصة للقيام ببعض الحيل الإثارة. على سبيل المثال، يمكننا تحويل تدفق بـ
#idx("series, summation of", sub: "accelerating sequence of approximations")
#idx("sequence accelerator")
#emph[مُسرّع تسلسل] يحول تسلسلاً من التقريبات إلى تسلسل جديد يتطابق مع القيمة نفسها التي يتطابق معها التسلسل الأصلي، ولكن بسرعة أكبر.

أحد هذه المُسرّعات، ويعود إلى عالم الرياضيات السويسري من القرن الثامن عشر
#idx("Euler, Leonhard", sub: "series accelerator")
ليونارد أويلر (#en[Leonhard Euler])، يعمل بشكل جيد مع التسلسلات التي هي مجاميع جزئية لمتسلسلات متناوبة (متسلسلات ذات حدود متناوبة الإشارات). في تقنية أويلر، إذا كان $S_(n)$ هو الحد الـ $n$ لتسلسل المجموع الأصلي، فإن التسلسل المـُسرّع يحتوي على الحدود:

$ S_(n+1) - frac((S_(n+1)-S_(n))^(2), S_(n-1)-2S_(n)+S_(n+1)) $

وهكذا، إذا تـُمّ تمثيل التسلسل الأصلي كـ تدفق قيم، فإن التسلسل المـُتحوّل يـُعطى بـ
#idx("eulertransform", decl: true)
#syntax("
def euler_transform(s):
    s0 = stream_ref(s, 0)     # ", $S_(n-1)$, "
    s1 = stream_ref(s, 1)     # ", $S_(n)$, "
    s2 = stream_ref(s, 2)     # ", $S_(n+1)$, "
    return pair(s2 - square(s2 - s1) / (s0 + (-2) * s1 + s2),
                memo(lambda: euler_transform(stream_tail(s))))
      ")

لاحظ أننا نستخدم تحسين التحفيظ من القسم @sec:delayed-lists، لأننا فيما يلي سنعتمد على التقييم التكراري للتدفق الناتج.

يمكننا توضيح تسريع أويلر بتسلسل تقريباتنا لـ $pi$:

#snippet(```python
print(display_stream(euler_transform(pi_stream)))
```)

#output(```python
print(display_stream(euler_transform(pi_stream)))
```)

والأفضل من ذلك، يمكننا تسريع التسلسل المـُسرّع، وتسريعه عودياً، وهكذا. أي أننا ننشئ تدفق تدفقات (هيكل سنسميه
#idx("tableau")
#emph[لوحة] (#en[tableau])) يكون فيه كل تدفق هو التحويل للتدفق السابق:
#idx("maketableau", decl: true)
#snippet(```python
def make_tableau(transform, s):
    return pair(s, lambda: make_tableau(transform, transform(s)))
```)

تأخذ اللوحة الشكل:

$ mat(delim: #none, s_(00), s_(01), s_(02), s_(03), s_(04), dots.h; , s_(10), s_(11), s_(12), s_(13), dots.h; , , s_(20), s_(21), s_(22), dots.h; , , , , dots.h, ) $

وأخيراً، نـُشكّل تسلسلاً بأخذ الحد الأول في كل صف من اللوحة:
#idx("acceleratedsequence", decl: true)
#snippet(```python
def accelerated_sequence(transform, s):
    return stream_map(head, make_tableau(transform, s))
```)

يمكننا توضيح هذا النوع من "التسريع الفائق" لتسلسل $pi$:

#snippet(```python
print(display_stream(accelerated_sequence(euler_transform, pi_stream)))
```)

#output(```python
print(display_stream(accelerated_sequence(euler_transform, pi_stream)))
```)

النتيجة مبهرة. أخذ ثمانية حدود من التسلسل يعطي القيمة الصحيحة لـ $pi$ لـ 14 منزلة عشرية.
وإذا كنا قد استخدمنا تسلسل $pi$ الأصلي فقط، فسنحتاج إلى الحساب على مرتبة $10^(13)$ حد (أي توسيع المتسلسلة إلى حد تكون فيه الحدود الفردية أقل من $10^(-13)$) للحصول على تلك الدقة!
#idx("π (pi)", sub: "stream of approximations", sort: "pi")

كان بإمكاننا تنفيذ تقنيات التسريع هذه دون استخدام التدفقات. ولكن صياغة التدفق أنيقة ومريحة بشكل خاص لأن تسلسل الحالات بأكمله متاح لنا كـ هيكل بيانات يمكن التلاعب به بمجموعة موحدة من العمليات.

#exercise(label-name: <ex:stream-internal-def>, [
لويس ريسونر (#en[Louis Reasoner]) ليس سعيداً بأداء التدفق الناتج عن الدالة #py("sqrt_stream") ويحاول تحسينه باستخدام التحفيظ:

#snippet(```python
def sqrt_stream_optimized(x):
    return pair(1,
                memo(lambda: stream_map(lambda guess:
                                        sqrt_improve(guess, x),
                                        sqrt_stream_optimized(x))))
```)

بينما تقترح أليسا ب. هاكر (#en[Alyssa P. Hacker]):

#snippet(```python
def sqrt_stream_optimized_2(x):
    guesses = pair(1,
                   memo(lambda: stream_map(lambda guess:
                                           sqrt_improve(guess, x),
                                           guesses)))
    return guesses
```)

وتدعي أن نسخة لويس أقل كفاءة بكثير من نسختها، لأنها تجري حسابات فائضة. اشرح إجابة أليسا.
وهل ستكون مقاربة أليسا دون تحفيظ أكثر كفاءة من #py("sqrt_stream") الأصلية؟
])

#exercise(label-name: <ex:stream-limit>, [
اكتب دالة
#idx("streamlimit")
#py("stream_limit") تأخذ كوسائط تدفقاً وعدداً (التسامح). وينبغي أن تفحص التدفق حتى تعثر على عنصرين متتاليين يختلفان في القيمة المطلقة بمقدار أقل من التسامح، وترجع العنصر الثاني منهما. وباستخدام هذا، يمكننا حساب الجذور التربيعية حتى تسامح معطى بـ
#idx("sqrt", sub: "as stream limit", decl: true)
#snippet(```python
def sqrt(x, tolerance):
    return stream_limit(sqrt_stream(x), tolerance)
```)
])

#exercise(label-name: <ex:3_64>, [
استخدم المتسلسلة

$ mat(delim: #none, ln 2, =, 1-frac(1, 2)+frac(1, 3)-frac(1, 4)+ dots.c) $

لحساب ثلاثة تسلسلات من التقريبات للوغاريتم الطبيعي لـ 2،
#idx("logarithm, approximating 2")
بالطريقة نفسها التي فعلناها أعلاه لـ $pi$.
ما مدى سرعة تقارب هذه التسلسلات؟
])

#idx("iterative process", sub: "as a stream process")

#subheading([التدفقات اللانهائية للأزواج])

#idx("pair(s)", sub: "infinite stream of")
#idx("infinite stream(s)", sub: "of pairs")
#idx("mapping", sub: "nested")

في القسم @sec:nested-mappings، رأينا كيف يتعامل نموذج التسلسل مع الحلقات المتداخلة التقليدية كـ عمليات معرفة على تسلسلات من الأزواج. وإذا قمنا بتعميم هذه التقنية على التدفقات اللانهائية، فإنه يمكننا كتابة برامج لا تـُمثل بسهولة كـ حلقات، لأن "الحلقة" يجب أن تمتد عبر مجموعة لانهائية.

على سبيل المثال، افترض أننا نريد تعميم دالة
#idx("primesumpairs", sub: "infinite stream")
#py("prime_sum_pairs") من القسم @sec:nested-mappings لإنتاج تدفق الأزواج لـ #emph[جميع] الأعداد الصحيحة $(i,j)$ مع $i lt.eq j$ بحيث يكون $i+j$ أولياً. إذا كانت #py("int_pairs") هي تسلسل جميع أزواج الأعداد الصحيحة $(i,j)$ مع $i lt.eq j$، فإن تدفقنا المطلوب هو ببساطة#footnote[كما في القسم @sec:sequences-conventional-interfaces، نمثل زوجاً من الأعداد الصحيحة كـ قائمة بدلاً من زوج.]

#snippet(```python
print(stream_filter(lambda pair: is_prime(head(pair) + head(tail(pair))),
              int_pairs))
```)

مشكلتنا إذاً هي إنتاج التدفق #py("int_pairs").
وبشكل أكثر عمومية، افترض أن لدينا تدفقين $S = (S_(i))$ و $T = (T_(j))$، وتخيل المصفوفة المستطيلة اللانهائية:

$ mat(delim: #none, (S_(0),T_(0)), (S_(0),T_(1)), (S_(0), T_(2)), dots.h; (S_(1),T_(0)), (S_(1),T_(1)), (S_(1), T_(2)), dots.h; (S_(2),T_(0)), (S_(2),T_(1)), (S_(2), T_(2)), dots.h; dots.h, , , ) $

نرغب في توليد تدفق يحتوي على جميع الأزواج في المصفوفة التي تقع على القطر أو فوقه، أي الأزواج:

$ mat(delim: #none, (S_(0),T_(0)), (S_(0),T_(1)), (S_(0), T_(2)), dots.h; , (S_(1),T_(1)), (S_(1), T_(2)), dots.h; , , (S_(2), T_(2)), dots.h; , , , dots.h) $

(إذا أخذنا كِلا $S$ و $T$ ليكون التدفق للأعداد الصحيحة، فإن هذا سيكون تدفقنا المطلوب #py("int_pairs").)

سَمِّ التدفق العام للأزواج #py("pairs(S, T)")، واعتبره مكوناً من ثلاثة أجزاء: الزوج $(S_(0),T_(0))$، وباقي الأزواج في الصف الأول، والأزواج المتبقية:#footnote[انظر التمرين @ex:pairs-array لبعض الرؤية حول سبب اختيارنا لهذا التفكيك.]

$ mat(delim: #none, (S_(0),T_(0)), (S_(0),T_(1)), (S_(0), T_(2)), dots.h; , (S_(1),T_(1)), (S_(1), T_(2)), dots.h; , , (S_(2), T_(2)), dots.h; , , , dots.h) $

لاحظ أن القطعة الثالثة في هذا التفكيك (الأزواج التي ليست في الصف الأول) هي (عودياً) الأزواج المشكلة من #py("stream_tail(S)") و #py("stream_tail(T)").
لاحظ أيضاً أن القطعة الثانية (باقي الصف الأول) هي:

#snippet(```python
stream_map(lambda x: llist(head(s), x),
           stream_tail(t))
```)

وهكذا يمكننا تشكيل تدفق الأزواج لدينا كما يلي:

#syntax("
def pairs(s, t):
    return pair(llist(head(s), head(t)),
                lambda: ", meta("combine-in-some-way"), "(
                            stream_map(lambda x: llist(head(s), x),
                                       stream_tail(t)),
                            pairs(stream_tail(s), stream_tail(t))))
      ")

لإكمال الدالة، يجب أن نختار طريقة ما لـ
#idx("infinite stream(s)", sub: "merging")
دمج التدفقين الداخليين. إحدى الأفكار هي استخدام نظير التدفق للدالة #py("append") من القسم @sec:sequences:

#idx("streamappend", decl: true)
#snippet(```python
def stream_append(s1, s2):
    return (s2
            if is_none(s1)
            else pair(head(s1),
                      lambda: stream_append(stream_tail(s1), s2)))
```)

هذا غير مناسب للتدفقات اللانهائية، لأنه يأخذ جميع العناصر من التدفق الأول قبل إدراج التدفق الثاني. وعلى وجه الخصوص، إذا حاولنا توليد جميع أزواج الأعداد الصحيحة الموجبة باستخدام:

#snippet(```python
print(pairs(integers, integers))
```)

فإن تدفق نتائجنا سيمر أولاً عبر جميع الأزواج ذات العدد الصحيح الأول المساوي لـ 1، وبالتالي لن ينتج أبداً أزواجاً بأي قيمة أخرى للعدد الصحيح الأول.

للتعامل مع التدفقات اللانهائية، نحتاج إلى ابتكار ترتيب دمج يضمن الوصول إلى كل عنصر في النهاية إذا تركنا برنامجنا يعمل لوقت كافٍ. وطريقة أنيقة لتحقيق ذلك تكون بالدالة #py("interleave") التالية:#footnote[البيان الدقيق للخاصية المطلوبة في ترتيب الدمج هو كما يلي: يجب أن تكون هناك دالة $f$ بوسيطين بحيث أن الزوج المقابل لـ العنصر $i$ من التدفق الأول والعنصر $j$ من التدفق الثاني سيظهر كـ عنصر رقم $f(i,j)$ في تدفق الخرج. حيلة استخدام #py("interleave") لتحقيق ذلك عرضها لنا
#idx("Turner, David")
ديفيد تيرنر، الذي استخدمها في اللغة
#idx("KRC")
KRC (تيرنر 1981).]

#idx("interleave", decl: true)
#snippet(```python
def interleave(s1, s2):
    return (s2
            if is_none(s1)
            else pair(head(s1),
                      lambda: interleave(s2, stream_tail(s1))))
```)

بما أن #py("interleave") تأخذ عناصر بالتناوب من التدفقين، فإن كل عنصر من التدفق الثاني سيجد طريقه في النهاية إلى التدفق المتداخل، حتى لو كان التدفق الأول لانهائياً.

يمكننا بالتالي توليد التدفق المطلوب للأزواج كـ
#idx("pairs", decl: true)
#snippet(```python
def pairs(s, t):
    return pair(llist(head(s), head(t)),
                lambda: interleave(stream_map(lambda x: llist(head(s), x),
                                              stream_tail(t)),
                                   pairs(stream_tail(s),
                                         stream_tail(t))))
```)

#exercise(label-name: <ex:stream-pair-order>, [
افحص التدفق #py("pairs(integers, integers)").
هل يمكنك إبداء أي تعليقات عامة حول الترتيب الذي تـُوضع به الأزواج في التدفق؟ على سبيل المثال، كم عدداً تقريبياً من الأزواج تسبق الزوج (1,100)؟ والزوج (99,100)؟ والزوج (100,100)؟ (إذا كان بإمكانك تقديم بيانات رياضية دقيقة هنا، فهذا أفضل بكثير. لكن لا تتردد في تقديم إجابات وصفية أكثر إذا وجدت نفسك عالقاً.)
])

#exercise(label-name: <ex:3_67>, [
عدل الدالة #py("pairs") بحيث تنتج #py("pairs(integers, integers)") تدفق #emph[جميع] أزواج الأعداد الصحيحة $(i,j)$ (دون الشرط $i lt.eq j$). إرشاد: ستحتاج إلى إدخال تدفق إضافي.
])

#exercise(label-name: <ex:pairs-array>, [
يعتقد لويس ريسونر (#en[Louis Reasoner]) أن بناء تدفق الأزواج من ثلاثة أجزاء معقد بلا داعٍ. وبدلاً من فصل الزوج $(S_(0),T_(0))$ عن باقي الأزواج في الصف الأول، يقترح العمل مع الصف الأول بأكمله، كما يلي:

#snippet(```python
def pairs(s, t):
    return interleave(stream_map(lambda x: llist(head(s), x),
                                 t),
                      pair(stream_tail(s), stream_tail(t)))
```)

هل يعمل هذا؟ فكر فيما يحدث إذا قمنا بتقييم #py("pairs(integers, integers)") باستخدام تعريف لويس لـ #py("pairs").
])

#exercise(label-name: <ex:stream-pythagorean-triples>, [
اكتب دالة #py("triples") تأخذ ثلاثة تدفقات لانهائية، $S$ و $T$ و $U$، وتنتج تدفق الثلاثيات $(S_(i),T_(j),U_(k))$ بحيث يكون $i lt.eq j lt.eq k$. استخدم #py("triples") لتوليد تدفق جميع
#idx("Pythagorean triples", sub: "with streams")
ثلاثيات فيثاغورس للأعداد الصحيحة الموجبة، أي الثلاثيات $(i,j,k)$ بحيث يكون $i lt.eq j$ و $i^(2) + j^(2) =k^(2)$.
])

#exercise(label-name: <ex:weighted-pairs>, [
سيكون من الجيد أن نكون قادرين على توليد
#idx("infinite stream(s)", sub: "merging")
تدفقات تظهر فيها الأزواج بترتيب مفيد، بدلاً من الترتيب الذي ينتج عن عملية تداخل خاصة. يمكننا استخدام تقنية تشبه دالة #py("merge") من التمرين @ex:merge، إذا قمنا بتحديد طريقة للقول إن زوجاً من الأعداد الصحيحة "أقل من" آخر. إحدى الطرق للقيام بذلك هي تحديد "دالة وزن" $W(i,j)$ والنص على أن $(i_(1),j_(1))$ أقل من $(i_(2),j_(2))$ إذا كان $W(i_(1),j_(1)) < W(i_(2),j_(2))$. اكتب دالة
#idx("mergeweighted")
#py("merge_weighted") تشبه #py("merge")، باستثناء أن #py("merge_weighted") تأخذ وسيطاً إضافياً #py("weight")، وهو دالة تحسب وزن زوج ما، وتـُستخدم لتحديد الترتيب الذي ينبغي أن تظهر به العناصر في التدفق المدمج الناتج.#footnote[سنطلب أن تكون دالة الوزن بحيث يزداد وزن زوج ما كلما تحركنا على طول صف أو أسفل عمود في مصفوفة الأزواج.] وباستخدام هذا، عمم #py("pairs") إلى دالة #py("weighted_pairs") تأخذ تدفقين، إلى جانب دالة تحسب دالة وزن، وتولد تدفق الأزواج، مرتبة وفقاً للوزن. استخدم دالتك لتوليد:

+ تدفق جميع أزواج الأعداد الصحيحة الموجبة $(i,j)$ مع $i lt.eq j$ مرتبة وفقاً للمجموع $i + j$
+ تدفق جميع أزواج الأعداد الصحيحة الموجبة $(i,j)$ مع $i lt.eq j$، حيث لا يقبل أي من $i$ أو $j$ القسمة على 2 أو 3 أو 5، وتـُرتب الأزواج وفقاً للمجموع $2 i + 3 j + 5 i j$.
])

#exercise(label-name: <ex:ramanujan-nums>, [
الأعداد التي يمكن التعبير عنها كـ مجموع مكعبين بأكثر من طريقة تـُسمى أحياناً
#idx("Ramanujan numbers")
#emph[أعداد رامانوجان] (#en[Ramanujan numbers])، تكريماً لـ عالم الرياضيات سرينيفاسا رامانوجان.#footnote[للاقتباس من نعي ج. هـ. هاردي لـ
#idx("Hardy, Godfrey Harold")
#idx("Ramanujan, Srinivasa")
رامانوجان (هاردي 1921): "كان السيد ليتلوود (على ما أعتقد) هو من لاحظ أن 'كل عدد صحيح موجب كان صديقاً له.' أتذكر أنني ذهبت لرؤيته مرة عندما كان مريضاً في بوتني. ركبت في سيارة أجرة رقم 1729، ولاحظت أن الرقم بدا لي مملاً نوعاً ما، وأنني آمل ألا يكون نذير شؤم. فأجاب 'لا، إنه عدد ممتع للغاية؛ إنه أصغر عدد يمكن التعبير عنه كـ مجموع مكعبين بطريقتين مختلفتين.'" حيلة استخدام الأزواج الموزونة لتوليد أعداد رامانوجان عرضها علينا
#idx("Leiserson, Charles E.")
تشارلز ليسرسون.] توفر التدفقات المرتبة للأزواج حلاً أنيقاً لمشكلة حساب هذه الأعداد. وللعثور على عدد يمكن كتابته كـ مجموع مكعبين بطريقتين مختلفتين، نحتاج فقط إلى توليد تدفق أزواج الأعداد الصحيحة $(i,j)$ الموزونة وفقاً للمجموع $i^(3) + j^(3)$ (انظر التمرين @ex:weighted-pairs)، ثم البحث في التدفق عن زوجين متتاليين بنفس الوزن. اكتب دالة لتوليد أعداد رامانوجان. أول عدد من هذا القبيل هو 1,729. ما هي الأعداد الخمسة التالية؟
])

#exercise(label-name: <ex:3_72>, [
بطريقة مماثلة للتمرين @ex:ramanujan-nums ولد تدفق جميع الأعداد التي يمكن كتابتها كـ مجموع مربعين بثلاث طرق مختلفة (موضحاً كيف يمكن كتابتها كذلك).
])

#idx("pair(s)", sub: "infinite stream of")
#idx("infinite stream(s)", sub: "of pairs")
#idx("mapping", sub: "nested")

#subheading([التدفقات كإشارات])

#idx("signal processing", sub: "stream model of")
#idx("infinite stream(s)", sub: "to model signals")

بدأنا مناقشتنا للتدفقات بوصفها نظائر حسابية لـ "الإشارات" في أنظمة معالجة الإشارات.
وفي الواقع، يمكننا استخدام التدفقات لنمذجة أنظمة معالجة الإشارات بطريقة مباشرة للغاية، بمثيل قيم إشارة عند فترات زمنية متتالية كـ عناصر متتالية لـ تدفق. على سبيل المثال، يمكننا تنفيذ
#idx("integrator, for signals")
#emph[مُكامل] (#en[integrator]) أو
#emph[جامع] يحسب المجموع لـ تدفق دخل $x=(x_(i))$، وقيمة أولية $C$، وزيادة صغيرة $d t$:

$ mat(delim: #none, S_(i), =, C +sum_(j=1)^(i) x_(j) thin d t) $

ويرجع تدفق القيم $S=(S_(i))$.
دالة #py("integral") التالية تـُذكرنا بـ تعريف "الأسلوب الضمني" لـ تدفق الأعداد الصحيحة (القسم @sec:infinite-streams):

#idx("integral", decl: true)
#snippet(```python
def integral(integrand, initial_value, dt):
    integ = pair(initial_value,
                 lambda: add_streams(scale_stream(integrand, dt),
                                     integ))
    return integ
```)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-49.svg", width: 54%), caption: [الدالة #py("integral") من منظور نظام معالجة الإشارات.], label-name: <fig:integral>)

الشكل @fig:integral هو صورة لنظام معالجة إشارات يتوافق مع الدالة #py("integral").
يـُقاس تدفق الدخل بـ $d t$ ويـُمَرَّر عبر جامع، يـُمَرَّر خرجه عائداً عبر الجامع نفسه.
وينعكس الإسناد الذاتي في تعريف #py("integ") في الشكل بواسطة حلقة التغذية الراجعة التي تربط خرج الجامع بأحد دخليه.

#exercise(label-name: <ex:rc-circuit>, [
يمكننا نمذجة الدوائر الكهربائية باستخدام التدفقات لتمثيل قيم التيارات أو الجهود عند تسلسل من الأوقات. على سبيل المثال، افترض أن لدينا
#idx("RC circuit")
#idx("circuit", sub: "modeled with streams")
#idx("electrical circuits, modeled with streams")
#emph[دائرة RC] تتكون من مقاومة قيمتها $R$ ومكثف سعته $C$ على التوالي. استجابة الجهد $v$ للدائرة لتيار مـُحقن $i$ تتحدد بالصيغة في الشكل @fig:rc، والذي يـُعرض هيكله بـ مخطط سريان الإشارة المرفق.

اكتب دالة #py("RC") تنمذج هذه الدائرة.
ينبغي أن تأخذ #py("RC") كدخل قيم $R$ و $C$ و $d t$ وترجع دالة تأخذ كدخل تدفقاً يمثل التيار $i$ وقيمة أولية لـ جهد المكثف $v_(0)$ وتنتج كخرج تدفق الجهود $v$. على سبيل المثال، ينبغي أن تكون قادراً على استخدام #py("RC") لنمذجة دائرة RC بـ $R = 5$ أوم، و $C = 1$ فاراد، وخطوة زمنية 0.5 ثانية بتقييم #py("const RC1 = RC(5, 1, 0.5)").
يحدد هذا #py("RC1") كـ دالة تأخذ تدفقاً يمثل التسلسل الزمني للتيارات وجهاً أولياً للمكثف وتنتج تدفق الخرج للجهود.
])

#exercise(label-name: <ex:zero-crossing>, [
تصمم أليسا ب. هاكر نظاماً لمعالجة الإشارات القادمة من مستشعرات فيزيائية. وإحدى الميزات المهمة التي ترغب في إنتاجها هي إشارة تصف
#idx("signal processing", sub: "zero crossings of a signal")
#idx("zero crossings of a signal")
#emph[تقاطعات الصفر] لإشارة الدخل. أي أن الإشارة الناتجة يجب أن تكون $+1$ كلما تغيرت إشارة الدخل من سالبة إلى موجبة، و $-1$ كلما تغيرت إشارة الدخل من موجبة إلى سالبة، و 0 خلاف ذلك. (افترض أن إشارة الدخل 0 هي موجبة.) على سبيل المثال، إشارة دخل نموذجية مع إشارة تقاطع الصفر المرتبطة بها ستكون:

#syntax($dots.h$, " 1  2  1.5  1  0.5  -0.1  -2  -3  -2  -0.5  0.2  3  4 ", $dots.h$, "
", $dots.h med$, "  0  0    0  0    0     -1  0   0   0     0    1  0  0 ", $dots.h$)

في نظام أليسا، تـُمثل الإشارة القادمة من المستشعر كـ تدفق #py("sense_data") وتدفق #py("zero_crossings") هو التدفق المقابل لـ تقاطعات الصفر. تكتب أليسا أولاً دالة #py("sign_change_detector") تأخذ قيمتين كوسائط وتقارن إشارات القيم لإنتاج $0$ أو $1$ أو $-1$ مناسب.
ثم تبني تدفق تقاطعات الصفر الخاص بها كما يلي:

#snippet(```python
def make_zero_crossings(input_stream, last_value):
    return pair(sign_change_detector(head(input_stream), last_value),
                lambda: make_zero_crossings(stream_tail(input_stream),
                                            head(input_stream)))

zero_crossings = make_zero_crossings(sense_data, 0)
```)

تمر رئيسة أليسا، إيفا لو آتور (#en[Eva Lu Ator])، وتقترح أن هذا البرنامج مكافئ تقريباً للبرنامج التالي، الذي يستخدم الدالة #py("stream_map_2") من التمرين @ex:combine-streams:

#syntax("
zero_crossings = stream_map_2(sign_change_detector,
                              sense_data,
                              ", meta("expression"), ")
      ")

أكمل البرنامج بتوفير #meta("expression") المحدد.
])

#exercise(label-name: <ex:zero-crossing-2>, [
لسوء الحظ، فإن كاشف تقاطعات الصفر لـ أليسا في
#idx("signal processing", sub: "zero crossings of a signal")
#idx("zero crossings of a signal")
#idx("signal processing", sub: "smoothing a signal")
#idx("smoothing a signal")
التمرين @ex:zero-crossing يثبت أنه غير كافٍ، لأن الإشارة المليئة بالضوضاء من المستشعر تؤدي إلى تقاطعات صفر زائفة. يقترح ليم إي. تويكيت (#en[Lem E. Tweakit])، وهو متخصص عتاد، أن تقوم أليسا بتنعيم الإشارة لتصفية الضوضاء قبل استخراج تقاطعات الصفر. تأخذ أليسا بنصيحته وتحدد استخراج تقاطعات الصفر من الإشارة المبنية بمتوسط كل قيمة من بيانات المستشعر مع القيمة السابقة. وتشرح المشكلة لمساعدها، لويس ريسونر، الذي يحاول تنفيذ الفكرة، مـُعدلاً برنامج أليسا كما يلي:

#snippet(```python
def make_zero_crossings(input_stream, last_value):
    avpt = (head(input_stream) + last_value) / 2
    return pair(sign_change_detector(avpt, last_value),
                lambda: make_zero_crossings(stream_tail(input_stream),
                                            avpt))
```)

هذا لا ينفذ خطة أليسا بشكل صحيح.
اعثر على الخلل الذي أدخله لويس واصلحه دون تغيير هيكل البرنامج. (إرشاد: ستحتاج إلى زيادة عدد الوسائط لـ #py("make_zero_crossings").)
])

#sicp-figure(image("/images/img_original/ch3-Z-G-51.svg", width: 70%), caption: [دائرة RC و #idx("signal-flow diagram") مخطط سريان الإشارة المرتبط بها.], label-name: <fig:rc>)

#exercise(label-name: <ex:3_76>, [
لدى إيفا لو آتور نقد لمقاربة لويس في التمرين @ex:zero-crossing-2.
#idx("signal processing", sub: "zero crossings of a signal")
#idx("zero crossings of a signal")
#idx("signal processing", sub: "smoothing a signal")
#idx("smoothing a signal")
البرنامج الذي كتبه ليس نمطياً، لأنه يخلط بين عملية التنعيم واستخراج تقاطعات الصفر. على سبيل المثال، ينبغي ألا يضطر المـُستخرج إلى التغيير إذا وجدت أليسا طريقة أفضل لتهيئة إشارة الدخل الخاصة بها. ساعد لويس بكتابة دالة #py("smooth") تأخذ تدفقاً كـ دخل وتنتج تدفقاً يكون فيه كل عنصر هو متوسط عنصرين متتاليين في تدفق الدخل. ثم استخدم #py("smooth") كـ مكون لتنفيذ كاشف تقاطعات الصفر بأسلوب أكثر نمطية.
])

#idx("signal processing", sub: "stream model of")
#idx("infinite stream(s)", sub: "to model signals")
