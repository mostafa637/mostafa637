// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([التدفقات والتقييم المؤجل], label-name: <sec:streams-and-delayed-evaluation>)

#idx("stream(s)", sub: "delayed evaluation and")
#idx("delayed evaluation", sub: "streams and")

تظهر الدالة #py("integral") في نهاية القسم السابق كيف يمكننا استخدام التدفقات لنمذجة أنظمة معالجة الإشارات التي تحتوي على
#idx("feedback loop, modeled with streams")
حلقات تغذية راجعة. تُمثل حلقة التغذية الراجعة للجامع الموضحة في الشكل @fig:integral بالحقيقة القائلة إن
#idx("integral", sub: "need for delayed evaluation")
التدفق الداخلي لـ #py("integral")،
#py("integ")،
تُمّ تعريفه بدلالة نفسه:

#snippet(```python
integ = pair(initial_value,
             lambda: add_streams(scale_stream(integrand, dt),
                                 integ))
```)

تعتمد قدرة المفسر على التعامل مع مثل هذا التعريف الضمني على التأخير الناتج عن تغليف الاستدعاء لـ #py("add_streams") في تعبير دالة غير مسمى (#en[lambda]).
ودون هذا التأخير، لن يتمكن المفسر من بناء #py("integ") قبل تقييم الاستدعاء لـ #py("add_streams")، والذي سيتطلب أن يكون #py("integ") معرفاً بالفعل.
وبشكل عام، فإن مثل هذا التأخير حاسم لاستخدام التدفقات لنمذجة أنظمة معالجة الإشارات التي تحتوي على حلقات. ودون تأخير، سيتعين صياغة نماذجنا بحيث تُقيَّم مدخلات أي مكون معالجة إشارات بشكل كامل قبل إمكانية إنتاج الخرج. وكان هذا ليمنع الحلقات.

لسوء الحظ، قد تتطلب نماذج التدفق للأنظمة ذات الحلقات استخدامات للتأخير تتجاوز نمط برمجة التدفق المعروض حتى الآن. على سبيل المثال، يوضح الشكل @fig:analog-computer نظام معالجة إشارات لحل
#idx("differential equation")
المعادلة التفاضلية $d y/d t=f(y)$ حيث $f$ دالة معطاة. يوضح الشكل مكون تعيين، يطبق $f$ على إشارة دخله، مرتبطاً في حلقة تغذية راجعة بمُكامل بطريقة مشابهة جداً لتلك الخاصة بدوائر الحاسوب التناظري التي تُستخدم بالفعل لحل مثل هذه المعادلات.

#sicp-figure(image("/images/img_original/ch3-Z-G-52.svg", width: 70%), caption: ["دائرة حاسوب تناظري" #idx("analog computer") تحل المعادلة $d y/d t = f(y)$.], label-name: <fig:analog-computer>)

بفرض أننا أُعطينا قيمة أولية $y_(0)$ لـ $y$، يمكننا محاولة نمذجة هذا النظام باستخدام الدالة:
#idx("solve differential equation", decl: true)
#snippet(```python
def solve(f, y0, dt):
    y = integral(dy, y0, dt)
    dy = stream_map(f, y)
    return y
```)

هذه الدالة لا تعمل، لأنه في السطر الأول من #py("solve")، يتطلب الاستدعاء لـ #py("integral") أن يكون الدخل #py("dy") معرفاً، وهو ما لا يحدث حتى السطر الثاني من #py("solve").

ومن ناحية أخرى، فإن القصد من تعريفنا منطقي، لأنه يمكننا، من حيث المبدأ، البدء في توليد تدفق #py("y") دون معرفة #py("dy").
وبالفعل، يمكن لـ #py("integral") والعديد من عمليات التدفق الأخرى توليد جزء من الإجابة مع إعطاء معلومات جزئية فقط عن الوسائط.
بالنسبة لـ #py("integral")، فإن العنصر الأول لتدفق الخرج هو #py("initial_value") المحدد. وبالتالي، يمكننا توليد العنصر الأول لتدفق الخرج دون تقييم دالة التكامل #py("dy"). وبمجرد أن نعرف العنصر الأول لـ #py("y")، فإن #py("stream_map") في السطر الثاني من #py("solve") يمكنها البدء في العمل لتوليد العنصر الأول لـ #py("dy")، والذي سينتج العنصر التالي لـ #py("y")، وهكذا.

للإستفادة من هذه الفكرة، سنعيد تعريف #py("integral") لتتوقع أن يكون تدفق دالة التكامل
#idx("delayed argument")
#idx("argument(s)", sub: "delayed")
#idx("delayed expression", sub: "explicit")
#emph[وسيطاً مؤجلاً].
ستجبر الدالة #py("integral") تقييم دالة التكامل فقط عندما يتطلب الأمر توليد أكثر من العنصر الأول لتدفق الخرج:

#idx("integral", sub: "with delayed argument", decl: true)
#snippet(```python
def integral(delayed_integrand, initial_value, dt):
    integ = pair(initial_value,
                 lambda: add_streams(
                             scale_stream(delayed_integrand(), dt),
                             integ))
    return integ
```)

الآن يمكننا تنفيذ دالة #py("solve") لدينا بتأجيل تقييم #py("dy") في الإعلان عن #py("y"):
#idx("solve differential equation", decl: true)
#snippet(```python
def solve(f, y0, dt):
    y = integral(lambda: dy, y0, dt)
    dy = stream_map(f, y)
    return y
```)

وبشكل عام، يجب على كل مستدعي لـ #py("integral") الآن تأجيل وسيط دالة التكامل. ويمكننا توضيح أن الدالة #py("solve") تعمل بتقريب
#idx("e", sub: "as solution to differential equation", sort: "e")
$e approx 2.718$ بحساب القيمة عند $y=1$ لحل المعادلة التفاضلية $d y/d t=y$ مع الشرط الأولي $y(0)=1$:#footnote[للاكتكمال في وقت معقول، يتطلب هذا الحساب استخدام تحسين التحفيظ من القسم @sec:delayed-lists في #py("integral") وفي الدالة #py("add_streams") المستخدمة في #py("integral") (باستخدام الدالة #py("stream_map_2_optimized") كما هو مقترح في التمرين @ex:fib-stream-efficiency).]

#snippet(```python
print(stream_ref(solve(lambda y: y, 1, 0.001), 1000))
```)

#output(```python
print(stream_ref(solve(lambda y: y, 1, 0.001), 1000))
```)

#exercise(label-name: <ex:integral>, [
الدالة #py("integral") المستخدمة أعلاه كانت مماثلة للتعريف "الضمني" لـ التدفق اللانهائي للأعداد الصحيحة في القسم @sec:infinite-streams. وبدلاً من ذلك، يمكننا إعطاء تعريف لـ #py("integral") يكون أكثر شبه بـ #py("integers-starting-from") (أيضاً في القسم @sec:infinite-streams):
#idx("integral", decl: true)
#snippet(```python
def integral(integrand, initial_value, dt):
    return pair(initial_value,
                None
                if is_none(integrand)
                else integral(stream_tail(integrand),
                              dt * head(integrand) + initial_value,
                              dt))
```)

عند استخدامها في أنظمة ذات حلقات، تواجه هذه الدالة المشكلة نفسها التي تواجهها نسختنا الأصلية من #py("integral").
عدل الدالة بحيث تتوقع #py("integrand") كوسيط مؤجل وبالتالي يمكن استخدامها في الدالة #py("solve") الموضحة أعلاه.
])

#exercise(label-name: <ex:2nd-order>, [
فكر في مشكلة تصميم نظام معالجة إشارات لدراسة المتجانسة
#idx("differential equation", sub: "second-order")
المعادلة التفاضلية الخطية من الرتبة الثانية:

$ mat(delim: #none, frac(d^(2) y, d t^(2))-a f r a c(d y, d t)-b y, =, 0) $

تدفق الخرج، الذي ينماذج $y$، يُوَلَّد بواسطة شبكة تحتوي على حلقة. هذا لأن قيمة $d^(2)y/d t^(2)$ تعتمد على قيم $y$ و $d y/d t$ وتحدّد كِلا هاتين القيمتين بتكامل $d^(2)y/d t^(2)$. المخطط الذي نرغب في ترميزه موضح في الشكل @fig:2nd-order. اكتب دالة #py("solve_2nd") تأخذ كوسائط الثوابت $a$ و $b$ و $d t$ والقيم الأولية $y_(0)$ و $d y_(0)$ لـ $y$ و $d y/d t$ وتولد تدفق القيم المتتالية لـ $y$.

#sicp-figure(image("/images/img_original/ch3-Z-G-53.svg", width: 70%), caption: [مخطط سريان الإشارة لحل معادلة تفاضلية خطية من الرتبة الثانية.], label-name: <fig:2nd-order>)
])

#exercise(label-name: <ex:3_79>, [
عمم دالة
#idx("differential equation", sub: "second-order")
#py("solve_2nd") من التمرين @ex:2nd-order بحيث يمكن استخدامها لحل المعادلات التفاضلية العامة من الرتبة الثانية $d^(2) y/d t^(2)=f(d y/d t, thin y)$.
])

#sicp-figure(image("/images/img_original/ch3-Z-G-58.svg", width: 70%), caption: [دائرة RLC على التوالي.], label-name: <fig:series-rlc>)

#exercise(label-name: <ex:rlc_circuit>, [
تتكون #emph[دائرة RLC على التوالي]
#idx("RLC circuit")
#idx("circuit", sub: "modeled with streams")
#idx("electrical circuits, modeled with streams")
من مقاومة، ومكثف، ومحث موصلة على التوالي، كما هو موضح في الشكل @fig:series-rlc. إذا كانت $R$ و $L$ و $C$ هي المقاومة، والمحثية، والسعة، فإن العلاقات بين الجهد ($v$) والتيار ($i$) للمكونات الثلاثة تُوصف بالمعادلات:

$ mat(delim: #none, v_(R), =, i_(R) R; v_(L), =, L f r a c(d i_(L), d t); i_(C), =, C f r a c(d v_(C), d t)) $

وتملي توصيلات الدائرة العلاقات:

$ mat(delim: #none, i_(R), =, i_(L)=-i_(C); v_(C), =, v_(L)+v_(R)) $

ويوضح دمج هذه المعادلات أن حالة الدائرة (الملخصة بـ $v_(C)$، الجهد عبر المكثف، و $i_(L)$، التيار في المحث) تُوصف بـ زوج المعادلات التفاضلية:

$ mat(delim: #none, frac(d v_(C), d t), =, -frac(i_(L), C); frac(d i_(L), d t), =, frac(1, L)v_(C)-frac(R, L)i_(L)) $

مخطط سريان الإشارة الذي يمثل هذا النظام من المعادلات التفاضلية موضح في الشكل @fig:rlc-signal-flow.

اكتب دالة #py("RLC") تأخذ كوسائط المعاملات $R$ و $L$ و $C$ للدائرة والزيادة الزمنية $d t$. وبطريقة مماثلة للدالة #py("RC") من التمرين @ex:rc-circuit، ينبغي أن تنتج #py("RLC") دالة تأخذ القيم الأولية لمتغيرات الحالة، $v_(C_(0))$ و $i_(L_(0))$، وتنتج زوجاً (باستخدام #py("pair")) لـ تدفقَي الحالات $v_(C)$ و $i_(L)$. وباستخدام #py("RLC")، ولد زوج التدفقات الذي ينماذج سلوك دائرة RLC على التوالي بـ $R = 1$ أوم، و $C= 0.2$ فاراد، و $L = 1$ هنري، و $d t = 0.1$ ثانية، والقيم الأولية $i_(L_(0)) = 0$ أمبير و $v_(C_(0)) = 10$ فولت.
])

#idx("stream(s)", sub: "delayed evaluation and")
#idx("delayed evaluation", sub: "streams and")

#subheading([تقييم الترتيب العادي])

#idx("normal-order evaluation", sub: "delayed evaluation and")
#idx("delayed evaluation", sub: "normal-order evaluation and")

توضح الأمثلة في هذا القسم كيف يوفر التقييم المؤجل مرونة برمجة كبيرة، ولكن الأمثلة نفسها تظهر أيضاً كيف يمكن لهذا أن يجعل برامجنا أكثر تعقيداً. فـدالتنا الجديدة #py("integral")، على سبيل المثال، تمنحنا القوة لنمذجة الأنظمة ذات الحلقات، ولكن يجب علينا الآن تذكر أن #py("integral") ينبغي استدعاؤها بـ دالة تكامل مؤجلة، ويجب أن تكون كل دالة تستخدم #py("integral") على دراية بذلك.
وفي الواقع، لقد أنشأنا فئتين من الدوال: الدوال العادية والدوال التي تأخذ وسائط مؤجلة. وبشكل عام، فإن إنشاء فئات منفصلة من الدوال يجبرنا على إنشاء فئات منفصلة من الدوال عالية الرتبة أيضاً.#footnote[هذا انعكاس صغير، في Python، للصعوبات التي واجهتها
#idx("higher-order functions", sub: "static typing and") #idx("data types", sub: "in statically typed languages") #idx("Pascal, lack of higher-order functions in") #idx("statically typed language") #idx("programming language", sub: "statically typed") لغات البرمجة ذات الكتابة الثابتة المبكرة مثل باسكال في التعامل مع الدوال عالية الرتبة.
في هذه اللغات، كان على المبرمج تحديد أنواع البيانات للوسائط ونتيجة كل دالة: عدد، قيمة منطقية، تسلسل، وهكذا. وبالتالي، لم نكن لنتمكن من التعبير عن تجريد مثل "تطبيق دالة معطاة #py("fun") على جميع العناصر في تسلسل" بواسطة دالة واحدة عالية الرتبة مثل #py("stream_map").
بل إننا سنحتاج إلى دالة تعيين مختلفة لكل تركيبة مختلفة من أنواع بيانات الوسائط والنتائج التي قد تُحدد لـ #py("fun").
إن الحفاظ على مفهوم عملي لـ "نوع البيانات" في وجود الدوال عالية الرتبة يثير العديد من القضايا الصعبة. وتتضح إحدى الطرق للتعامل مع هذه المشكلة باللغة
#idx("ML")
ML
#idx("Gordon, Michael")
#idx("Milner, Robin")
#idx("Wadsworth, Christopher")
(جوردون، وميلنر، وواذسوورث 1979)، والتي تتضمن "أنواع بياناتها المتعددة الأشكال بارامترياً" قوالب للتحويلات عالية الرتبة بين أنواع البيانات. علاوة على ذلك، فإن أنواع البيانات لمعظم الدوال في ML لا يُعلن عنها المبرمج صراحة أبداً. وبدلاً من ذلك، تتضمن ML آلية
#idx("type-inferencing mechanism")
#emph[استنتاج الأنواع] تستخدم المعلومات في البيئة لاستنتاج أنواع البيانات للدوال المعرفة حديثاً.
واليوم، تطورت لغات البرمجة ذات الكتابة الثابتة لتدعم عادةً شكلاً من أشكال استنتاج الأنواع بالإضافة إلى التعددية الشكلية البارامترية، بدرجات متغيرة من القوة. #idx("Haskell") #idx("type(s)", sub: "polymorphic") #idx("polymorphic types") وتجمع هاسكل بين نظام أنواع معبر واستنتاج أنواع قوي.]

إحدى الطرق لتجنب الحاجة إلى فئتين مختلفين من الدوال هي جعل جميع الدوال تأخذ وسائط مؤجلة. ويمكننا تبني نموذج تقييم تُأجَّل فيه جميع الوسائط للدوال تلقائياً وتُجبر الوسائط فقط عندما تكون مطلوبة بالفعل (على سبيل المثال، عندما تتطلبها عملية أولية). وسيحول هذا لغتنا لاستخدام تقييم الترتيب العادي، والذي وصفناه لأول مرة عندما قدمنا نموذج الاستبدال للتقييم في القسم @sec:substitution-model.
إن التحويل إلى تقييم الترتيب العادي يوفر طريقة موحدة وأنيقة لتبسيط استخدام التقييم المؤجل، وسيكون هذا استراتيجية طبيعية نتبناها إذا كنا نهتم فقط بمعالجة التدفقات. في القسم @sec:lazy-evaluation، وبعد أن ندرس المُقيِّم، سنرى كيفية تحويل لغتنا بهذه الطريقة بالضبط.
لسوء الحظ، فإن تضمين التأخيرات في استدعاءات الدوال يلحق الضرر بقدرتنا على تصميم برامج تعتمد على ترتيب الأحداث، مثل البرامج التي تستخدم الإسناد، أو تعدل البيانات، أو تجري دخلاً أو خرجاً.
حتى التأخير الواحد في ذيل زوج يمكن أن يسبب إرباكاً كبيراً، كما يتضح من التمارين @ex:delayed1 و @ex:delayed2.
وعلى حد علم أي شخص، فإن القابلية للتغير والتقييم المؤجل لا يختلطان جيداً في لغات البرمجة.

#idx("normal-order evaluation", sub: "delayed evaluation and")
#idx("delayed evaluation", sub: "normal-order evaluation and")

#sicp-figure(image("/images/img_original/ch3-Z-G-59.svg", width: 70%), caption: [مخطط سريان الإشارة لحل دائرة RLC على التوالي.], label-name: <fig:rlc-signal-flow>)
