// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([التدفقات هي قوائم مؤجلة], label-name: <sec:delayed-lists>)

#idx("stream(s)", sub: "implemented as delayed lists")

كما رأينا في القسم @sec:sequences-conventional-interfaces، يمكن للتسلسلات أن تُستخدم كواجهات قياسية لربط وحدات البرنامج. وقمنا بصياغة تجريدات قوية للتعامل مع التسلسلات، مثل #py("map")، و #py("filter")، و #py("accumulate")، والتي تلتقط مجموعة واسعة من العمليات بطريقة موجزة وأنيقة في الوقت نفسه.

لسوء الحظ، إذا قمنا بتمثيل التسلسلات كقوائم، فإن هذه الأناقة تأتي على حساب عدم كفاءة شديدة فيما يتعلق بالوقت والمساحة اللذين تتطلبهما حساباتنا.
عندما نمثل التعاملات على التسلسلات كتحويلات للقوائم، يجب على برامجنا إنشاء نسخ لهياكل البيانات (التي قد تكون ضخمة) في كل خطوة من خطوات العملية.

لرؤية سبب صحة ذلك، دعنا نقارن بين برنامجين لحساب مجموع جميع الأعداد الأولية في فترة محددة. يُكتب البرنامج الأول بأسلوب تكراري قياسي:#footnote[افترض أن لدينا دالة شرطية #py("is_prime") (مثلاً، كما في القسم @sec:primality) تختبر ما إذا كان العدد أولياً.]
#idx("sumprimes", decl: true)
#snippet(```python
def sum_primes(a, b):
    def iter(count, accum):
        return (accum
                if count > b
                else iter(count + 1, count + accum)
                if is_prime(count)
                else iter(count + 1, accum))
    return iter(a, 0)
```)

يجري البرنامج الثاني الحساب نفسه باستخدام عمليات التسلسل من القسم @sec:sequences-conventional-interfaces:
#idx("sumprimes", decl: true)
#snippet(```python
def sum_primes(a, b):
    return reduce(lambda x, y: x + y,
                  0,
                  filter(is_prime,
                         enumerate_interval(a, b)))
```)

عند إجراء الحساب، يلزَم البرنامج الأول تخزين المجموع الذي يتجمع فقط. وعلى العكس من ذلك، فإن التصفية في البرنامج الثاني لا يمكنها إجراء أي اختبار حتى تقوم دالة #py("enumerate_interval") بإنشاء قائمة كاملة بالأعداد في الفترة.
تُنتج التصفية قائمة أخرى، والتي تُمَرَّر بدورها إلى #py("accumulate") قبل أن تُقلَّص لتشكيل المجموع. مثل هذا التخزين الوسيط الكببر ليس مطلوباً في البرنامج الأول، الذي يمكننا التفكير فيه على أنه يُعدد الفترة تدريجياً، مُضيفاً كل عدد أولي إلى المجموع فور توليده.

تصبح عدم الكفاءة في استخدام القوائم واضحة بشكل مؤلم إذا استخدمنا نموذج التسلسل لحساب العدد الأولي الثاني في الفترة من 10,000 إلى 1,000,000 عن طريق تقييم التعبير:

#snippet(```python
head(tail(filter(is_prime,
                 enumerate_interval(10000, 1000000))))
```)

يعثر هذا التعبير بالفعل على العدد الأولي الثاني، لكن العبء الحسابي يكون باهظاً للغاية. نحن ننشئ قائمة لقرابة مليون عدد صحيح، ونُصفي هذه القائمة باختبار كل عنصر لمعرفة أوليته، ثم نتجاهل النتيجة بأكملها تقريباً. في أسلوب البرمجة الأكثر تقليدية، سنقوم بتداخل التعداد والتصفية، ونلتزم بالتوقف عند الوصول إلى العدد الأولي الثاني.

التدفقات فكرة ذكية تسمح للمرء باستخدام التلاعب بالتسلسلات دون تكبد تكاليف التلاعب بالتسلسلات كقوائم. باستخدام التدفقات، يمكننا تحقيق أفضل ما في العالمين: يمكننا صياغة البرامج بأناقة كتلاعب بالتسلسلات، مع تحقيق كفاءة الحساب التزايدي. الفكرة الأساسية هي الترتيب لإنشاء تدفق جزئياً فقط، وتمرير البناء الجزئي إلى البرنامج الذي يستهلك التدفق. إذا حاول المستهلك الوصول إلى جزء من التدفق لم يُبْنَ بعد، فسيقوم التدفق تلقائياً بإنشاء ما يكفي فقط من نفسه لإنتاج الجزء المطلوب، وبذلك يحافظ على وهم وجود التدفق بأكمله. بعبارة أخرى، على الرغم من أننا سنكتب البرامج كما لو كنا نعالج تسلسلات كاملة، فإننا نصمم تنفيذ التدفق ليتداخل تلقائياً وبشفافية بين بناء التدفق واستخدامه.

ولتحقيق ذلك، سنبني التدفقات باستخدام الأزواج، مع وضع العنصر الأول من التدفق في رأس (#py("head")) الزوج.
ومع ذلك، بدلاً من وضع قيمة باقي التدفق
#idx("promise to evaluate")
في ذيل (#py("tail")) الزوج، سنضع هناك "وعداً" لحساب الباقي إذا تُمّ طلبه على الإطلاق.
إذا كان لدينا عنصر بيانات #py("h") وتدفق #py("t")، فإننا نبني تدفقاً رأسه #py("h") وذيله #py("t") عن طريق تقييم
#py("pair(h, lambda: t)")—حيث يكون الذيل
#py("t") للتدفق "مغلفاً" في دالة بلا وسائط،
#idx("delayed expression")
بحيث يكون تقييمه #emph[مؤجلاً].
#idx("empty stream")
#idx("stream(s)", sub: "empty")
التدفق الفارغ هو #py("null")، تماماً مثل القائمة الفارغة.

للوصول إلى عنصر البيانات الأول لتدفق غير فارغ، نحدد ببساطة رأس (#py("head")) الزوج، كما هو الحال مع القائمة.
ولكن للوصول إلى ذيل التدفق، نحتاج إلى تقييم التعبير المؤجل.
للتسهيل، نُعرّف:
#idx("streamtail", decl: true)
#snippet(```python
def stream_tail(stream):
    return tail(stream)()
```)

يحدد هذا ذيل الزوج ويطبق الدالة الموجودة هناك للحصول على الزوج التالي من التدفق
(أو #py("null") إذا كان ذيل التدفق فارغاً)—مما يُجبر في الواقع
#idx("forcing", sub: "tail of stream")
الدالة الموجودة في ذيل الزوج على الوفاء بوعدها.
#idx("stream(s)", sub: "implemented as delayed lists")

يمكننا إنشاء التدفقات واستخدامها بنفس الطريقة التي ننشئ بها القوائم ونستخدمها، لتمثيل البيانات المجمعة المرتبة في تسلسل. على وجه الخصوص، يمكننا بناء نظائر التدفق لعمليات القوائم من الفصل @chap:data، مثل #py("list_ref")، و #py("map")، و #py("for_each"):#footnote[ينبغي أن يزعجك هذا. حقيقة أننا نُعرّف دوالاً متماثلة جداً للتدفقات والقوائم تشير إلى أننا نفتقد تجريداً أساسياً ما. لسوء الحظ، للاستفادة من هذا التجريد، سيتعين علينا ممارسة تحكم أدق في عملية التقييم مما يمكننا في الوقت الحالي. سنناقش هذه النقطة أكثر في نهاية القسم @sec:streams-and-delayed-evaluation.
في القسم @sec:lazy-evaluation، سنطور إطار عمل يوحّد القوائم والتدفقات.]
#idx("streamref", decl: true)#idx("streammap", decl: true)#idx("streamforeach", decl: true)
#snippet(```python
def stream_ref(s, n):
    return (head(s)
            if n == 0
            else stream_ref(stream_tail(s), n - 1))

def stream_map(f, s):
    return (None
            if is_none(s)
            else pair(f(head(s)),
                      lambda: stream_map(f, stream_tail(s))))

def stream_for_each(fun, s):
    if is_none(s):
        return True
    else:
        fun(head(s))
        return stream_for_each(fun, stream_tail(s))
```)

الدالة #py("stream_for_each") مفيدة لـعرض التدفقات:
#idx("displaystream", decl: true)
#snippet(```python
def display_stream(s):
    return stream_for_each(display, s)
```)

لجعل تنفيذ التدفق يتداخل تلقائياً وبشفافية بين بناء التدفق واستخدامه، قمنا بالترتيب لتقييم ذيل التدفق عندما يتم الوصول إليه بواسطة الدالة #py("stream_tail") بدلاً من الوقت الذي يتم فيه بناء التدفق بواسطة #py("pair").
يُذكرنا اختيار التنفيذ هذا بمناقشتنا للأعداد الكسرية في القسم @sec:abstraction-barriers، حيث رأينا أنه يمكننا اختيار تنفيذ الأعداد الكسرية بحيث يتم اختيار اختزال البسط والمقام إلى أردأ صورة إما في وقت البناء أو في وقت الاختيار. ينتج عن تنفيذي الأعداد الكسرية التجريد للبيانات نفسه، لكن الاختيار يؤثر على الكفاءة. وهناك علاقة مماثلة بين التدفقات والقوائم العادية. كتجريد للبيانات، فإن التدفقات هي نفسها القوائم. والفرق هو الوقت الذي تُقيَّم فيه العناصر. فمع القوائم العادية، يُقيَّم كل من #py("head") و #py("tail") في وقت البناء. أما مع التدفقات، فيُقيَّم #py("tail") في وقت الاختيار.

#subheading([التدفقات في عملها])

لرؤية كيفية تصرف هيكل البيانات هذا، دعنا نحلل حساب الأعداد الأولية "الباهظ" الذي رأيناه أعلاه، مع إعادته بدلالة التدفقات:

#snippet(```python
print(head(stream_tail(stream_filter(
                     is_prime,
                     stream_enumerate_interval(10000, 1000000)))))
```)

سنرى أنه يعمل بالفعل بكفاءة.

نبدأ باستدعاء #py("stream_enumerate_interval") بالوسائط 10,000 و 1,000,000. الدالة #py("stream_enumerate_interval") هي نظير التدفق للدالة #py("enumerate_interval")
(القسم @sec:sequences-conventional-interfaces):

#idx("streamenumerateinterval", decl: true)
#snippet(```python
def stream_enumerate_interval(low, high):
    return (None
            if low > high
            else pair(low,
                      lambda: stream_enumerate_interval(low + 1, high)))
```)

وبالتالي فإن النتيجة التي ترجعها #py("stream_enumerate_interval")، المشكلة بواسطة #py("pair")، هي:#footnote[الأعداد المعروضة هنا لا تظهر بالفعل في التعبير المؤجل. ما يظهر في الواقع هو التعبير الأصلي، في بيئة ترتبط فيها المتغيرات بالأعداد المناسبة. على سبيل المثال، #py("low + 1") مع ربط #py("low") بـ 10,000 يظهر بالفعل في المكان الذي تظهر فيه #py("10001").]

#snippet(```python
print(pair(10000, lambda: stream_enumerate_interval(10001, 1000000)))
```)

أي أن #py("stream_enumerate_interval") ترجع تدفقاً ممثلاً كزوج رأسه #py("head") هو 10,000 وذيله #py("tail") هو وعد بتعداد المزيد من الفترة إذا طُلِب منها ذلك. يتم الآن تصفية هذا التدفق للأعداد الأولية، باستخدام نظير التدفق للدالة #py("filter")
(القسم @sec:sequences-conventional-interfaces):
#idx("streamfilter", decl: true)
#snippet(```python
def stream_filter(pred, stream):
    return (None
            if is_none(stream)
            else pair(head(stream),
                      lambda: stream_filter(pred, stream_tail(stream)))
            if pred(head(stream))
            else stream_filter(pred, stream_tail(stream)))
```)

تختبر الدالة #py("stream_filter") رأس (#py("head")) التدفق (وهو 10,000). وبما أن هذا ليس أولياً، تدرس #py("stream_filter") ذيل تدفق دخلها. ويجبر استدعاء #py("stream_tail") تقييم #py("stream_enumerate_interval") المؤجل، والذي يرجع الآن:

#snippet(```python
print(pair(10001, lambda: stream_enumerate_interval(10002, 1000000)))
```)

تنظر الدالة #py("stream_filter") الآن إلى رأس (#py("head")) هذا التدفق، 10,001، وترى أنه ليس أولياً أيضاً، فتجبر استدعاء #py("stream_tail") آخر، وهكذا، حتى تعطي #py("stream_enumerate_interval") العدد الأولي 10,007، وعندها ترجع #py("stream_filter")، ووفقاً لتعريفها:

#snippet(```python
pair(head(stream),
     lambda: stream_filter(pred, stream_tail(stream)))
```)

والذي يكون في هذه الحالة:

#snippet(```python
print(pair(10007,
     lambda: stream_filter(
                 is_prime,
                 pair(10008,
                      lambda: stream_enumerate_interval(10009, 1000000)))))
```)

تُمَرَّر هذه النتيجة الآن إلى #py("stream_tail") في تعبيرنا الأصلي. هذا يجبر #py("stream_filter") المؤجل، والذي يجبر بدوره #py("stream_enumerate_interval") المؤجل حتى يعثر على العدد الأولي التالي، وهو 10,009. وأخيراً، فإن النتيجة التي تُمَرَّر إلى #py("head") في تعبيرنا الأصلي هي:

#snippet(```python
print(pair(10009,
     lambda: stream_filter(
                 is_prime,
                 pair(10010,
                      lambda: stream_enumerate_interval(10011, 1000000)))))
```)

ترجع الدالة #py("head") القيمة 10,009، ويكتمل الحساب. تُمّ اختبار الأعداد الصحيحة فقط للأولية بقدر ما كان ضرورياً للعثور على العدد الأولي الثاني، وتُمّ تعداد الفترة فقط بقدر ما كان ضرورياً لتغذية مرشح الأعداد الأولية.

بشكل عام، يمكننا التفكير في التقييم المؤجل على أنه برمجة
#idx("programming", sub: "demand-driven")
"مدفوعة بالطلب" (#en[demand-driven])، حيث تُنشّط كل مرحلة في عملية التدفق بقدر ما يكفي فقط لتلبية المرحلة التالية. ما فعلناه هو
#idx("order of events", sub: "decoupling apparent from actual")
فصل الترتيب الفعلي للأحداث في الحساب عن الهيكل الظاهري لدوالنا. نحن نكتب الدوال كما لو أن التدفقات توجد "دفعة واحدة" في حين أنه في الواقع، يُجرى الحساب تدريجياً، كما في أساليب البرمجة التقليدية.

#subheading([تحسين])

عندما نبني أزواج التدفقات، فإننا نؤجل تقييم تعبيرات ذيولها عن طريق تغليف هذه التعبيرات في دالة. ونقوم بإجبار تقييمها عند الحاجة، بتطبيق الدالة.

هذا التنفيذ يكفي لعمل التدفقات كما هو معلن، ولكن هناك تحسين مهم سنأخذه في الاعتبار عند الحاجة. في العديد من التطبيقات، ننتهي بإجبار الكائن المؤجل نفسه عدة مرات. هذا يمكن أن يؤدي إلى عدم كفاءة خطيرة في البرامج العودية التي تتضمن التدفقات. (انظر التمرين @ex:fib-stream-efficiency.)
والحل هو بناء كائنات مؤجلة بحيث أنه في أول مرة تُجبر فيها، تخزن القيمة التي تُمّ حسابها. وسترجع عمليات الإجبار اللاحقة ببساطة القيمة المخزنة دون تكرار الحساب. بعبارة أخرى، ننفذ بناء أزواج التدفقات كـدالة
#idx("delayed expression", sub: "memoized")
#idx("memoization", sub: "in stream tail")
مُحفظة تشبه تلك الموصوفة في التمرين @ex:memoization. إحدى الطرق لتحقيق ذلك هي استخدام الدالة التالية، والتي تأخذ كوسيطة دالة (بلا وسائط) وترجع نسخة مُحفظة من الدالة.
في أول مرة تُشغل فيها الدالة المُحفظة، تحفظ النتيجة المحسوبة. وفي التقييمات اللاحقة، ترجع ببساطة النتيجة.#footnote[هناك العديد من التنفيذات المحتملة للتدفقات غير تلك الموصوفة في هذا القسم. كان التقييم المؤجل، وهو المفتاح لجعل التدفقات عملية، متأصلاً في طريقة تمرير الوسائط
#idx("Algol", sub: "call-by-name argument passing")
#idx("call-by-name argument passing")
#emph[بالاسم] (#en[call-by-name]) في #en[Algol 60]. كان استخدام هذه الآلية لتنفيذ التدفقات قد وُصف لأول مرة بواسطة
#idx("Landin, Peter")
لاندين (1965). وتُمّ تقديم التقييم المؤجل للتدفقات في لسب (#en[Lisp]) بواسطة
#idx("Friedman, Daniel P.")
#idx("Wise, David S.")
فريدمان ووايز (1976). في تنفيذهما، كان
#py("cons") (المكافئ في لسب لدالتنا #py("pair")) يؤجل دائمًا تقييم وسائطه، بحيث تتصرف القوائم تلقائياً كتدفقات. ويُعرف تحسين التحفيظ أيضاً باسم
#idx("call-by-need argument passing")
#emph[التمرير بالحاجة] (#en[call-by-need]). وسوف يشير مجتمع ألغول إلى كائناتنا المؤجلة الأصلية على أنها
#idx("thunk", sub: "call-by-name")
#idx("thunk", sub: "call-by-need")
#idx("Algol", sub: "thunks")
#emph[ثنكات التمرير بالاسم] وإلى النسخ المُحسنة على أنها #emph[ثنكات التمرير بالحاجة].]
#idx("memo", decl: true)
#snippet(```python
def memo(fun):
    already_run = False
    result = None
    def memoized():
        nonlocal already_run, result
        if not already_run:
            result = fun()
            already_run = True
            return result
        else:
            return result
    return memoized
```)

يمكننا الاستفادة من #py("memo") كلما بنينا زوج تدفق. على سبيل المثال، بدلاً من
#idx("streammap", decl: true)
#snippet(```python
def stream_map(f, s):
    return (None
            if is_none(s)
            else pair(f(head(s)),
                      lambda: stream_map(f, stream_tail(s))))
```)

يمكننا تعريف دالة مُحسنة #py("stream_map") كما يلي:
#idx("streammapoptimized", decl: true)
#snippet(```python
def stream_map_optimized(f, s):
    return (None
            if is_none(s)
            else pair(f(head(s)),
                      memo(lambda:
                           stream_map_optimized(f, stream_tail(s)))))
```)

#exercise(label-name: <ex:combine-streams>, [
أعلن عن دالة #py("stream_map_2") تأخذ دالة ثنائية وتدفقين كوسائط وترجع تدفقاً عناصره هي نتائج تطبيق الدالة زوجياً على العناصر المقابلة في تدفقات الوسائط.
#idx("streammap2")
#snippet(```python
def stream_map_2(f, s1, s2):
    ...
```)

على غرار #py("stream_map_optimized")، أعلن عن دالة #py("stream_map_2_optimized") بتعديل #py("stream_map_2") لديك بحيث يستخدم تدفق النتيجة التحفيظ.
])

#exercise(label-name: <ex:delayed1>, [
لاحظ أن دالتنا الأوليّة #py("display") ترجع وسيطها بعد عرضه.
ما الذي يطبعه المفسر رداً على تقييم كل عبارة في التسلسل التالي؟#footnote[تمارين مثل @ex:delayed1 و @ex:delayed2 قيمة لاختبار فهمنا لكيفية عمل التقييم المؤجل. ومن ناحية أخرى، فإن خلط التقييم المؤجل مع
#idx("delayed evaluation", sub: "printing and")
الطباعة—والأسوأ من ذلك، مع الإسناد—أمر مربك للغاية، وقد عذب مدرسو دورات لغات الحاسوب طلابهم تقليدياً بأسئلة الامتحانات مثل تلك الموجودة في هذا القسم. وغني عن القول، إن كتابة البرامج التي تعتمد على مثل هذه الدقائق هي
#idx("programming", sub: "odious style")
أسلوب برمجة بغيض. جزء من قوة معالجة التدفقات هو أنها تسمح لنا بتجاهل الترتيب الذي تحدث به الأحداث بالفعل في برامجنا. لسوء الحظ، هذا بالضبط ما لا يمكننا تحمله في وجود الإسناد، مما يجبرنا على الاهتمام بالوقت والتغير.]

#snippet(```python
x = stream_map(display, stream_enumerate_interval(0, 10))

stream_ref(x, 5)

stream_ref(x, 7)
```)

ما الذي يطبعه المفسر إذا تُمّ استخدام #py("stream_map_optimized") بدلاً من #py("stream_map")؟

#snippet(```python
x = stream_map_optimized(display, stream_enumerate_interval(0, 10))

stream_ref(x, 5)

stream_ref(x, 7)
```)
])

#exercise(label-name: <ex:delayed2>, [
فكر في تسلسل العبارات:

#snippet(```python
sum = 0

def accum(x):
    global sum
    sum = x + sum
    return sum

seq = stream_map(accum, stream_enumerate_interval(1, 20))

y = stream_filter(is_even, seq)

z = stream_filter(lambda x: x % 5 == 0, seq)

stream_ref(y, 7)

display_stream(z)
```)

ما هي قيمة #py("sum") بعد تقييم كل عبارة من العبارات أعلاه؟
#idx("delayed evaluation", sub: "assignment and")
ما الاستجابة المطبوعة لـ تقييم التعبيرين #py("stream_ref") و #py("display_stream")؟
هل تختلف هذه الاستجابات إذا طبقنا الدالة #py("memo") على كل ذيل لكل زوج تدفق مبني، كما هو مقترح في التحسين أعلاه؟ اشرح.
])
