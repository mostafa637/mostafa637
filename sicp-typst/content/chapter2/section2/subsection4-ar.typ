// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([مثال: لغة الصور], label-name: <sec:graphics>)

#idx("picture language")

يقدم هذا القسم لغة بسيطة لرسم الصور توضح قوة تجريد البيانات وخاصية الإغلاق، وتستغل أيضاً الدوال العليا بطريقة أساسية. صُممت اللغة لتسهيل التجريب بأنماط مثل تلك الموضحة في
الشكل @fig:sqlimit-designs، والتي تتكون من عناصر مكررة تم إزاحتها وتغيير مقياسها.#footnote[تعتمد لغة الصور على اللغة التي أنشأها
#idx("Henderson, Peter")
#en[Peter Henderson] لإنشاء صور مثل المحفورة الخشبية «حد المربع» (#en[Square Limit]) لـ
#idx("Escher, Maurits Cornelis")
#en[M. C. Escher] (انظر Henderson 1982). تتضمن المحفورة الخشبية نمطاً مكرراً ومغيراً مقياسه، مشابهاً للترتيبات المترسمة باستخدام دالة
#py("square_limit")
في هذا القسم.] وفي هذه اللغة، تُمثَّل كائنات البيانات المراد دمجها كدوال بدلاً من بنية قوائم مترابطة. وكما سمح لنا الإجراء
#py("pair")،
الذي يحقق خاصية
#idx("closure", sub: "closure property of picture-language operations")
الإغلاق، ببناء بنية قوائم مترابطة معقدة اعتباطياً بسهولة، فإن العمليات في هذه اللغة، والتي تحقق أيضاً خاصية الإغلاق، تسمح لنا ببناء أنماط معقدة اعتباطياً بسهولة.

#sicp-figure(image("/images/img_original/2.9.svg", width: 50%), caption: [تصاميم مولدة بلغة الصور.], label-name: <fig:sqlimit-designs>)

#subheading([لغة الصور])

عندما بدأنا دراستنا للبرمجة في
القسم @sec:elements-of-programming، شددنا على أهمية وصف اللغة من خلال التركيز على أوليات اللغة، ووسائل دمجها، ووسائل تجريدها. وسنتبع هذا الإطار هنا.

جزء من أناقة لغة الصور هذه هو وجود نوع واحد فقط من العناصر، يُسمى
#idx("painter(s)")
#emph[رسّام] (#en[painter]). يرسم الرسّام صورة تُزاح ويُغيَّر مقياسها لتناسب إطاراً معيناً على شكل متوازي أضلاع. فعلى سبيل المثال، هناك رسّام أولي سنسميه #py("wave")
يقوم بعمل رسم خطي خام،
كما هو موضح في الشكل @fig:wave.

#sicp-figure(image("/images/img_original/2.10.svg", width: 100%), caption: [الصور المنتجة بواسطة الرسّام #py("wave")، بالنسبة لأربعة إطارات مختلفة. الإطارات، الموضحة بخطوط منقطة، ليست جزءاً من الصور.], label-name: <fig:wave>)

يعتمد الشكل الفعلي للرسم على الإطار—الصور الأربع كلها في الشكل @fig:wave ناتجة عن الرسّام #py("wave") نفسه، ولكن بالنسبة لأربعة إطارات مختلفة. ويمكن أن يكون الرسّامون أكثر إتقاناً من هذا: فالرسّام الأولي المسمى #py("rogers") يرسم صورة لمؤسس #en[MIT]، #en[William Barton Rogers]، كما هو موضح في
الشكل @fig:rogers.#footnote[#idx("MIT", sub: "early history of")
#idx("Rogers, William Barton")
كان #en[William Barton Rogers] (1804–1882) المؤسس وأول رئيس لـ #en[MIT]. كان عالماً في الجيولوجيا ومدرساً موهوباً، ودرّس في كلية ويليام وماري وفي جامعة فرجينيا. وفي عام 1859 انتقل إلى بوسطن، حيث وجد متسعاً من الوقت للبحث، وعمل على خطة لتأسيس "معهد متعدد الفنون" (معهد بوليتكنيك)، وعمل كأول مفتش عدادات غاز بالولاية في ماساتشوستس.

عندما أُسِّس معهد #en[MIT] في عام 1861، انتُخب روجرز كأول رئيس له. واعتنق روجرز مثلاً أعلى للـ "التعلم النافع" كان مختلفاً عن التعليم الجامعي في ذلك الوقت، والذي كان يبالغ في التركيز على الكلاسيكيات التي، كما كتب، "تقف في طريق التعليم الأكثر اتساعاً وأعلى وأكثر عملية وضبطاً للعلوم الطبيعية والاجتماعية." وكان هذا التعليم ليكون كذلك مختلفاً عن التعليم المهني الضيق. وعلى حد تعبير روجرز:

#blockquote[إن التمييز الذي يفرضه العالم بين العامل العلمي والعامل العملي هو تمييز باطل تماماً، وقد أثبتت تجربة الأزمنة الحديثة بأكملها عدم جدواه المطلقة.]

خدم روجرز كرئيس لـ #en[MIT] حتى عام 1870، عندما استقال بسبب سوء حالته الصحية. وفي عام 1878 استقال الرئيس الثاني لـ #en[MIT]،
#idx("Runkle, John Daniel")
#en[John Runkle]، تحت ضغط الأزمة المالية الناجمة عن ذعر عام 1873 وإجهاد صد محاولات جامعة هارفارد للاستحواذ على #en[MIT]. وعاد روجرز لشغل منصب الرئيس حتى عام 1881.

انهار روجرز وتوفي أثناء إلقاء كلمة أمام خريجي #en[MIT] في حفل التخرج عام 1882. واقتبس رانكل كلمات روجرز الأخيرة في كلمة تأبينية ألقيت في العام نفسه:

#blockquote["بينما أقف هنا اليوم وأرى ما أضحي عليه المعهد، ... أستحضر بدايات العلم. أتذكر منذ مائة وخمسين عاماً عندما نشر ستيفن هيلز كتيباً حول موضوع غاز الإضاءة، ذكر فيه أن أبحاثه أثبتت أن 128 حبة من الفحم البيتوميني—"

#idx("coal, bituminous")
"الفحم البيتوميني"، كانت هذه كلماته الأخيرة على الأرض. وهنا انحنى للأمام، وكأنه يستشير بعض الملاحظات على الطاولة أمامه، ثم استعاد ببطء وقفته المنتصبة، ورفع يديه، وانتقل من مشهد متاع الأرض وانتصاراتها إلى "غد الموت"، حيث تُحل أسرار الحياة، وتجد الروح المجردة راحة سرمدية في التأمل في الأسرار الجديدة التي لا تسبر غورها للمستقبل اللانهائي.]

وعلى حد تعبير #en[Francis A. Walker]
#idx("Walker, Francis Amasa")
(الرئيس الثالث لـ #en[MIT]):

#blockquote[طوال حياته كان يحمل نفسه بكل أمانة وبطولة، وتوفي كما كان يتمنى فارس صالح بالتأكيد، في عدته، وفي مركزه، وفي صلب عمل الواجب العام.]]
الصور الأربع في الشكل @fig:rogers
مرسومة بالنسبة للإطارات الأربعة نفسها
كما في صور #py("wave") في
الشكل @fig:wave.

#sicp-figure(image("/images/img_original/2.11.svg", width: 100%), caption: [صور ويليام بارتون روجرز (#en[William Barton Rogers])، مؤسس وأول رئيس لـ #en[MIT]، مرسومة بالنسبة للإطارات الأربعة نفسها كما في الشكل @fig:wave (الصورة الأصلية بإذن من متحف #en[MIT]).], label-name: <fig:rogers>)

لدمج الصور، نستخدم عمليات رسّامين مختلفة (#idx("painter(s)", sub: "operations")) تبني رسّامين جدداً من رسّامين معطين. على سبيل المثال، تأخذ عملية
#idx("beside")
#py("beside") رسّامين وتنتج رسّاماً مركباً جديداً يرسم صورة الرسّام الأول في النصف الأيسر من الإطار وصورة الرسّام الثاني في النصف الأيمن من الإطار. وبالمثل،
#idx("below")
تأخذ #py("below") رسّامين وتنتج رسّاماً مركباً يرسم صورة الرسّام الأول أسفل صورة الرسّام الثاني. وتحول بعض العمليات رسّاماً واحداً لإنتاج رسّام جديد. فعلى سبيل المثال،
#idx("flipvert")
تأخذ #py("flip_vert") رسّاماً وتنتج رسّاماً يرسم صورته مقلوبة رأساً على عقب، و
#idx("fliphoriz")
تنتج #py("flip_horiz") رسّاماً يرسم صورة الرسّام الأصلي معكوسة من اليسار إلى اليمين.

يُظهر الشكل @fig:build-up-wave رسم رسّام يسمى #py("wave4")
تم بناؤه على مرحلتين بدءاً من
#py("wave"):

#snippet(```python
wave2 = beside(wave, flip_vert(wave))
wave4 = below(wave2, wave2)
```)

عند بناء صورة معقدة بهذه الطريقة، نكون قد استغللنا حقيقة أن الرسّامين
#idx("closure", sub: "closure property of picture-language operations")
مغلقون تحت وسائل الدمج في اللغة.
فإن #py("beside") أو #py("below") لرسّامين هي نفسها رسّام؛ وبالتالي، يمكننا استخدامها كعنصر في صنع رسّامين أكثر تعقيداً. وكما هو الحال مع بناء بنية قوائم مترابطة باستخدام #py("pair")، فإن إغلاق بياناتنا تحت وسائل الدمج أمر حاسم للقدرة على إنشاء بنى معقدة باستخدام عدد قليل من العمليات فقط.

#sicp-figure(stack(dir: ttb, spacing: 1em, image("/images/img_original/2.12.svg", width: 50%), [#syntax("
$\\ $
wave2 =                          wave4 =
 beside(wave, flip_vert(wave))    below(wave2, wave2)

      ")]), caption: [إنشاء شكل معقد، بدءاً من الرسّام #py("wave") في الشكل @fig:wave.], label-name: <fig:build-up-wave>)

بمجرد أن نتمكن من دمج الرسّامين، نود التمكن من تجريد الأنماط النموذجية لدمج الرسّامين. وسوف ننفذ عمليات الرسّامين كدوال #en[Python].
وهذا يعني أننا لسنا بحاجة إلى آلية تجريد خاصة في لغة الصور: فبما أن وسائل الدمج هي دوال #en[Python] عادية، فلدينا تلقائياً القدرة على القيام بأي شيء مع عمليات الرسّامين نتمكن من القيام به مع الدوال.
فعلى سبيل المثال، يمكننا تجريد النمط في #py("wave4") كالتالي:
#idx("flippedpairs", decl: true)
#snippet(```python
def flipped_pairs(painter):
    painter2 = beside(painter, flip_vert(painter))
    return below(painter2, painter2)
```)

والإعلان عن #py("wave4") كنموذج لهذا النمط:

#snippet(```python
wave4 = flipped_pairs(wave)
```)

#sicp-figure(image("/images/img_javascript/ch2-Z-G-37.svg", width: 59%), caption: [الخطط التعاودية لـ #py("right_split") و #py("corner_split").], label-name: <fig:split-plans>)

يمكننا أيضاً تعريف عمليات تعاودية. إليك عملية تجعل الرسّامين ينقسمون ويتفرعون نحو اليمين كما هو موضح في الشكلين @fig:split-plans و @fig:split-plans-2:
#idx("rightsplit", decl: true)
#snippet(```python
def right_split(painter, n):
    if n == 0:
        return painter
    else:
        smaller = right_split(painter, n - 1)
        return beside(painter, below(smaller, smaller))
```)

يمكننا إنتاج أنماط متوازنة عن طريق التفرع نحو الأعلى وكذلك نحو اليمين (انظر التمرين @ex:up-split والشكلين @fig:split-plans و @fig:split-plans-2):
#idx("cornersplit", decl: true)
#snippet(```python
def corner_split(painter, n):
    if n == 0:
        return painter
    else:
        up = up_split(painter, n - 1)
        right = right_split(painter, n - 1)
        top_left = beside(up, up)
        bottom_right = below(right, right)
        corner = corner_split(painter, n - 1)
        return beside(below(painter, top_left),
                      below(bottom_right, corner))
```)

#sicp-figure(image("/images/img_javascript/2.14.svg", width: 45%), caption: [العملية التعاودية #py("right_split") مدمجة مع الرسّامين #py("wave") و #py("rogers"). ويؤدي دمج أشكال #py("corner_split") الأربعة إلى إنتاج #py("square_limit") المتماثل كما هو موضح في الشكل @fig:sqlimit-designs.], label-name: <fig:split-plans-2>)

بوضع أربع نسخ من #py("corner_split") بشكل مناسب، نحصل على نمط يسمى #py("square_limit")، والذي يظهر تطبيقه على #py("wave") و #py("rogers") في الشكل @fig:sqlimit-designs:
#idx("squarelimit", decl: true)
#snippet(```python
def square_limit(painter, n):
    quarter = corner_split(painter, n)
    half = beside(flip_horiz(quarter), quarter)
    return below(flip_vert(half), half)
```)

#exercise(label-name: <ex:up-split>, [
أعلن عن الدالة
#idx("upsplit")
#py("up_split")
المستخدمة بوساطة #py("corner_split").
وهي مشابهة لـ #py("right_split")، باستثناء أنها تبدل أدوار #py("below") و #py("beside").

#anchor(<ex:2_44>)
])

#subheading([العمليات العليا])

#idx("painter(s)", sub: "higher-order operations")

بالإضافة إلى تجريد أنماط دمج الرسّامين، يمكننا العمل على مستوى أعلى، بتجريد أنماط دمج عمليات الرسّامين. أي أنه يمكننا النظر إلى عمليات الرسّامين كعناصر للتلاعب بها ويمكننا كتابة وسائل دمج لهذه العناصر—دوال تأخذ عمليات الرسّامين كوسائط وتنشئ عمليات رسّامين جديدة.

فعلى سبيل المثال، يرتب كل من #py("flipped_pairs") و #py("square_limit") أربع نسخ من صورة الرسّام في نمط مربعي؛ ويختلفان فقط في كيفية توجيه النسخ. وإحدى الطرق لتجريد هذا النمط من دمج الرسّامين هي باستخدام الدالة التالية، التي تأخذ أربع عمليات رسّامين يأخذ كل منها وسيطًا واحدًا وتنتج عملية رسّام تحول رسّاماً معطى بتلك العمليات الأربع وترتب النتائج في مربع.#footnote[تتكون عملية الرسّام المعادة بوساطة #py("square_of_four") من عبارات متعددة، لذا لا يمكن كتابتها كتعبير لامبدا، والذي يجب أن يكون جسمه في #en[Python] تعبيراً واحداً. وبدلاً من ذلك، نستخدم إعلان دالة محلية باسم #py("combine"). #idx("lambda expression", sub: "restricted to a single expression in Python")]<foot:lambda_with_block>
والدوال #py("tl") و #py("tr") و #py("bl") و #py("br") هي التحويلات المراد تطبيقها على النسخة العلوية اليسرى، والنسخة العلوية اليمنى، والنسخة السفلية اليسرى، والنسخة السفلية اليمنى، على التوالي.
#idx("squareoffour", decl: true)
#snippet(```python
def square_of_four(tl, tr, bl, br):
    def combine(painter):
        top = beside(tl(painter), tr(painter))
        bottom = beside(bl(painter), br(painter))
        return below(bottom, top)
    return combine
```)

ثم يمكن تعريف #py("flipped_pairs") بدلالة #py("square_of_four") كما يلي:#footnote[بشكل مكافئ، كان بإمكاننا كتابة:
#idx("flippedpairs", decl: true)
#snippet(```python
flipped_pairs = square_of_four(identity, flip_vert,
                               identity, flip_vert)
```)]
#idx("flippedpairs", decl: true)
#snippet(```python
def flipped_pairs(painter):
    combine4 = square_of_four(identity, flip_vert,
                                    identity, flip_vert)
    return combine4(painter)
```)

ويمكن التعبير عن #py("square_limit") كالتالي:#footnote[تُدوِّر الدالة #py("rotate180") الرسّام بمقدار 180 درجة. وبدلاً من #py("rotate180") كان بإمكاننا قول #py("compose(flip_vert, flip_horiz)")، باستخدام دالة #py("compose") من التمرين @ex:compose.]
#idx("squarelimit", decl: true)
#snippet(```python
def square_limit(painter, n):
    combine4 = square_of_four(flip_horiz, identity,
                                    rotate180, flip_vert)
    return combine4(corner_split(painter, n))
```)

#exercise(label-name: <ex:splitting>, [
يمكن التعبير عن الدالتين #py("right_split") و #py("up_split") كحالات من عملية تقسيم عامة.
أعلن عن دالة
#idx("split")
#py("split") بخاصية أن تقييم:

#snippet(```python
right_split = split(beside, below)
up_split = split(below, beside)
```)

ينتج دوال #py("right_split") و #py("up_split") بالسلوك نفسه الذي للدالتين المُعلَن عنهما بالفعل.
])

#subheading([الإطارات])

#idx("frame (picture language)")

قبل أن نتمكن من إظهار كيفية تنفيذ الرسّامين ووسائل دمجهم، يجب أن ننظر أولاً في الإطارات (#idx("vector (mathematical)", sub: "in picture-language frame")). يمكن وصف الإطار بوساطة ثلاثة متجهات—متجه نقطة الأصل ومتجهي حافتين. يحدد متجه نقطة الأصل إزاحة نقطة أصل الإطار عن نقطة أصل مطلقة ما في المستوى، وتحدد متجهات الحواف إزاحات زوايا الإطار عن نقطة أصله.
وإذا كانت الحواف متعامدة، فسيكون الإطار مستطيلاً.
وإلا كان الإطار متوازي أضلاع أكثر عمومية.

يُظهر الشكل @fig:frame إطاراً ومتجهاته المرتبطة به. ووفقاً لتجريد البيانات، لسنا بحاجة إلى تحديد كيفية تمثيل الإطارات بعد، سوى القول بوجود باني
#idx("makeframe")
#py("make_frame")،
يأخذ ثلاثة متجهات وينتج إطاراً، وثلاثة محددات مقابلة:
#idx("originframe")
#py("origin_frame")،
#idx("edge1frame")
#py("edge1_frame")،
و
#idx("edge2frame")
#py("edge2_frame")
(انظر التمرين @ex:implement-frames).

#sicp-figure(image("/images/img_original/ch2-Z-G-42.svg", width: 70%), caption: [يوصَف الإطار بوساطة ثلاثة متجهات—نقطة الأصل وحافتان.], label-name: <fig:frame>)

سنستخدم الإحداثيات في المربع الوحدة (#idx("unit square") $0 lt.eq x, y lt.eq 1$) لتحديد الصور. ومع كل إطار، نربط #idx("frame (picture language)", sub: "coordinate map") #emph[خريطة إحداثيات الإطار]، والتي ستُستخدم لإزاحة وتغيير مقياس الصور لتناسب الإطار. تحول الخريطة المربع الوحدة إلى الإطار عن طريق تحويل المتجه $bold("v")=(x, y)$ إلى مجموع المتجهات:

$ upright("Origin(Frame)") + x dot.op upright(" Edge")_(1)upright(" (Frame)") + y dot.op upright(" Edge")_(2)upright(" (Frame)") $

فعلى سبيل المثال، يُحوَّل $(0, 0)$ إلى نقطة أصل الإطار، و $(1, 1)$ إلى الرأس المقابل بالقطر لنقطة الأصل، و $(0.5, 0.5)$ إلى مركز الإطار. ويمكننا إنشاء خريطة إحداثيات الإطار بالدالة التالية:#footnote[تستخدم الدالة #py("frame_coord_map") عمليات المتجهات الموصوفة في التمرين @ex:vectors أدناه، والتي نفترض أنه تم تنفيذها باستخدام تمثيل ما للمتجهات. وبسبب تجريد البيانات، لا يهم ما هو تمثيل المتجه هذا، طالما أن عمليات المتجهات تتصرف بشكل صحيح.]
#idx("framecoordmap", decl: true)
#snippet(```python
def frame_coord_map(frame):
    return lambda v: add_vect(origin_frame(frame),
                         add_vect(scale_vect(xcor_vect(v),
                                             edge1_frame(frame)),
                                  scale_vect(ycor_vect(v),
                                             edge2_frame(frame))))
```)

لاحظ أن تطبيق #py("frame_coord_map") على إطار يرجع دالة، عند إعطائها متجهاً، ترجع متجهاً. وإذا كان المتجه داخل المربع الوحدة، كان المتجه الناتج داخل الإطار. فعلى سبيل المثال،

#snippet(```python
print(frame_coord_map(a_frame)(make_vect(0, 0)))
```)

يرجع المتجه نفسه الذي ترجعه:

#snippet(```python
print(origin_frame(a_frame))
```)

#exercise(label-name: <ex:vectors>, [
يمكن تمثيل المتجه $v$ ثنائي الأبعاد (#idx("vector (mathematical)", sub: "represented as pair") #idx("vector (mathematical)", sub: "operations on")) الممتد من نقطة الأصل إلى نقطة كزوج يتكون من إحداثي $x$ وإحداثي $y$. نفذ تجريد بيانات للمتجهات عن طريق إعطاء الباني
#idx("makevect")
#py("make_vect")
ومحددين مقابلين:
#idx("xcorvect")
#py("xcor_vect")
و
#idx("ycorvect")
#py("ycor_vect").
وبدلالة المحددين والباني، نفذ الدوال:
#idx("addvect")
#py("add_vect")،
#idx("subvect")
#py("sub_vect")،
و
#idx("scalevect")
#py("scale_vect")
التي تؤدي عمليات جمع المتجهات، وطرح المتجهات، وضرب المتجه في كمية قياسية:

$ mat(delim: #none, (x_(1), y_(1))+(x_(2), y_(2)), =, (x_(1)+x_(2), y_(1)+y_(2)); (x_(1), y_(1))-(x_(2), y_(2)), =, (x_(1)-x_(2), y_(1)-y_(2)); s dot.op (x, y), =, (s x, s y)) $
])

#exercise(label-name: <ex:implement-frames>, [
إليك بانيان ممكنان للإطارات:
#idx("makeframe", decl: true)
#snippet(```python
def make_frame(origin, edge1, edge2):
    return llist(origin, edge1, edge2)

def make_frame(origin, edge1, edge2):
    return pair(origin, pair(edge1, edge2))
```)

لكل باني، قدِّم المحددَين المناسبَين لإنتاج تنفيذ للإطارات.

#anchor(<ex:2_47>)
])

#subheading([الرسّامون])

يُمثَّل الرسّام كدالة (#idx("painter(s)", sub: "represented as functions"))، عند إعطائها إطارًا كوسيط، ترسم صورة معينة تم إزاحتها وتغيير مقياسها لتناسب الإطار. بمعنى أنه إذا كان #py("p") رسّاماً و #py("f") إطاراً، فإننا ننتج صورة #py("p") في #py("f") عن طريق استدعاء #py("p") مع #py("f") كوسيط.

تعتمد تفاصيل كيفية تنفيذ الرسّامين الأوليين على الخصائص المحددة لنظام الرسومات ونوع الصورة المراد رسمها. فعلى سبيل المثال، نفترض أن لدينا دالة
#idx("drawline")
#py("draw_line")
ترسم خطاً على الشاشة بين نقطتين محددتين. يمكنك حينئذٍ إنشاء رسّامين لرسومات الخطوط، مثل الرسّام #py("wave") في الشكل @fig:wave، من قوائم مترابطة من القطع المستقيمة كما يلي:#footnote[تستخدم الدالة #py("segments_to_painter") تمثيل القطع المستقيمة الموصوف في التمرين @ex:segments2 أدناه. وتستخدم أيضاً دالة #py("for_each") الموصوفة في التمرين @ex:for-each.]

#idx("segmentstopainter", decl: true)
#snippet(```python
def segments_to_painter(segment_list):
    return lambda frame:
             for_each(lambda segment:
                        draw_line(
                            frame_coord_map(frame)
                                (start_segment(segment)),
                            frame_coord_map(frame)
                                (end_segment(segment))),
                      segment_list)
```)

تُعطى القطع المستقيمة باستخدام إحداثيات بالنسبة للمربع الوحدة. ولكل قطعة مستقيمة في القائمة المترابطة، يحوِّل الرسّام نقطتي نهاية القطعة بخريطة إحداثيات الإطار ويرسم خطاً بين النقطتين المحوَّلتين.

تمثيل الرسّامين كدوال يقيم حاجز تجريد قوياً في لغة الصور. فيمكننا إنشاء وخلط جميع أنواع الرسّامين الأوليين، بناءً على مجموعة متنوعة من قدرات الرسومات. ولا تهم تفاصيل تنفيذها. أي دالة يمكن أن تخدم كرسّام، بشرط أن تأخذ إطارًا كوسيط وترسم شيئاً مغيراً مقياسه ليلائم الإطار.#footnote[فعلى سبيل المثال، تم بناء الرسّام #py("rogers") في الشكل @fig:rogers من صورة متدرجة الرمادي. ولكل نقطة في إطار معطى، يحدد الرسّام #py("rogers") النقطة في الصورة التي تم رسمها إليها تحت خريطة إحداثيات الإطار، ويظللها وفقاً لذلك.

ومن خلال السماح بأنواع مختلفة من الرسّامين، فإننا نستغل فكرة البيانات المجردة المناقشة في القسم @sec:data-، حيث جادلنا بأن تمثيل الأعداد الكسرية يمكن أن يكون أي شيء على الإطلاق يحقق شرطاً مناسباً. وهنا نستخدم حقيقة أنه يمكن تنفيذ الرسّام بأي طريقة على الإطلاق، طالما أنه يرسم شيئاً في الإطار المحدد.

وأظهر القسم @sec:data- أيضاً كيف يمكن تنفيذ الأزواج كدوال. والرسّامون هم مثالنا الثاني على تمثيل دالي للبيانات.]

#exercise(label-name: <ex:segments2>, [
يمكن تمثيل قطعة مستقيمة موجهة في المستوى كزوج من المتجهات (#idx("line segment", sub: "represented as pair of vectors"))—المتجه الممتد من نقطة الأصل إلى نقطة بداية القطعة المستقيمة، والمتجه الممتد من نقطة الأصل إلى نقطة نهاية القطعة المستقيمة. استخدم تمثيل المتجهات الخاص بك من التمرين @ex:vectors لتعريف تمثيل للقطع المستقيمة مع باني
#idx("makesegment")
#py("make_segment")
ومحددَين
#idx("startsegment")
#py("start_segment")
و
#idx("endsegment")
#py("end_segment").
])

#exercise(label-name: <ex:making-wave>, [
استخدم #py("segments_to_painter") لتعريف الرسّامين الأوليين التالين:

+ الرسّام الذي يرسم الحد الخارجي للإطار المحدد.
+ الرسّام الذي يرسم حرف "X" بتوصيل الزوايا المتقابلة للإطار.
+ الرسّام الذي يرسم شكل معين بتوصيل منتصفات أضلاع الإطار.
+ الرسّام #py("wave").
])

#subheading([تحويل الرسّامين وتجميعهم])

#idx("painter(s)", sub: "transforming and combining")

تعمل العملية على الرسّامين (مثل #py("flip_vert") أو #py("beside")) عن طريق إنشاء رسّام يستدعي الرسّامين الأصليين بالنسبة للإطارات المشتقة من إطار الوسيط. وبالتالي، على سبيل المثال، لا يتعين على #py("flip_vert") معرفة كيفية عمل الرسّام من أجل قلبه—عليها فقط معرفة كيفية قلب الإطار رأساً على عقب: فالرسّام المقلوب يستخدم ببساطة الرسّام الأصلي، ولكن في الإطار المقلوب.

تعتمد عمليات الرسّامين على الدالة #py("transform_painter")، التي تأخذ كوسيطين رسّامًا ومعلومات حول كيفية تحويل إطار وتنتج رسّاماً جديداً. والرسّام المحول، عند استدعائه على إطار، يحول الإطار ويستدعي الرسّام الأصلي على الإطار المحول. والوسائط لـ #py("transform_painter") هي نقاط (ممثلة كمتجهات) تحدد زوايا الإطار الجديد: وعند إسقاطها في الإطار، تحدد النقطة الأولى نقطة أصل الإطار الجديد وتحدد النقطتان الأخريان نهايتي متجهي حافتيه. وبالتالي، فإن الوسائط داخل المربع الوحدة تحدد إطاراً محتوىً داخل الإطار الأصلي.
#idx("transformpainter", decl: true)
#snippet(```python
def transform_painter(painter, origin, corner1, corner2):
    def transformed(frame):
        m = frame_coord_map(frame)
        new_origin = m(origin)
        return painter(make_frame(
                           new_origin,
                           sub_vect(m(corner1), new_origin),
                           sub_vect(m(corner2), new_origin)))
    return transformed
```)

إليك كيفية قلب صور الرسّامين رأسياً:
#idx("flipvert", decl: true)
#snippet(```python
def flip_vert(painter):
    return transform_painter(painter,
                             make_vect(0, 1),  # new origin
                             make_vect(1, 1),  # new end of edge1
                             make_vect(0, 0)); # new end of edge2
```)

باستخدام #py("transform_painter")، يمكننا تعريف تحويلات جديدة بسهولة. على سبيل المثال، يمكننا الإعلان عن رسّام يقلص صورته إلى الربع العلوي الأيمن من الإطار الذي يُعطى له:
#idx("shrinktoupperright", decl: true)
#snippet(```python
def shrink_to_upper_right(painter):
    return transform_painter(painter,
                             make_vect(0.5, 0.5),
                             make_vect(1, 0.5),
                             make_vect(0.5, 1))
```)

وتُدوِّر تحويلات أخرى الصور عكس اتجاه عقارب الساعة بمقدار 90 درجة#footnote[الدالة #py("rotate90") هي دوران خالص للإطارات المربعة فقط، لأنها تقوم أيضاً بتمديد وتقليص الصورة لتلائم الإطار المدوَّر.]
#idx("rotate90", decl: true)
#snippet(```python
def rotate90(painter):
    return transform_painter(painter,
                             make_vect(1, 0),
                             make_vect(1, 1),
                             make_vect(0, 0))
```)

أو تعصر الصور نحو مركز الإطار:#footnote[أنشئت الصور ذات الشكل المعين في الشكلين @fig:wave و @fig:rogers بوساطة #py("squash_inwards") المطبقة على #py("wave") و #py("rogers").]
#idx("squashinwards", decl: true)
#snippet(```python
def squash_inwards(painter):
    return transform_painter(painter,
                             make_vect(0, 0),
                             make_vect(0.65, 0.35),
                             make_vect(0.35, 0.65))
```)

تحويل الإطار هو أيضاً المفتاح لتعريف وسائل دمج رسّامين أو أكثر.
فدالة #py("beside")، على سبيل المثال، تأخذ رسّامين، وتحولهما للرسم في النصفين الأيسر والأيمن من إطار الوسيط على التوالي، وتنتج رسّاماً مركباً جديداً. وعندما يُعطى الرسّام المركب إطاراً، فإنه يستدعي الرسّام المحول الأول للرسم في النصف الأيسر من الإطار ويستدعي الرسّام المحول الثاني للرسم في النصف الأيمن من الإطار:
#idx("beside", decl: true)
#snippet(```python
def beside(painter1, painter2):
    split_point = make_vect(0.5, 0)
    paint_left  = transform_painter(painter1,
                                    make_vect(0, 0),
                                    split_point,
                                    make_vect(0, 1))
    paint_right = transform_painter(painter2,
                                    split_point,
                                    make_vect(1, 0),
                                    make_vect(0.5, 1))
    def painter(frame):
        paint_left(frame)
        paint_right(frame)
    return painter
```)

لاحظ كيف يجعل تجريد بيانات الرسّام، وخاصة تمثيل الرسّامين كدوال، تنفيذ #py("beside") سهلاً. فدالة #py("beside") لا تحتاج إلى معرفة أي شيء عن تفاصيل الرسّامين المكونين سوى أن كل رسّام سيرسم شيئاً في الإطار المحدد له.

#exercise(label-name: <ex:rotate>, [
أعلن عن التحويل
#idx("fliphoriz")
#py("flip_horiz")،
الذي يقلب الرسّامين أفقياً، والتحويلات التي تُدوِّر الرسّامين عكس اتجاه عقارب الساعة بمقدار 180 درجة و 270 درجة.
])

#exercise(label-name: <ex:below>, [
أعلن عن عملية
#idx("below")
#py("below") للرسّامين.
تأخذ الدالة #py("below") رسّامين كوسيطين. والرسّام الناتج، عند إعطائه إطاراً، يرسم باستخدام الرسّام الأول في الجزء السفلي من الإطار وباستخدام الرسّام الثاني في الجزء العلوي.
عرِّف #py("below") بطريقتين مختلفين—أولاً عن طريق كتابة دالة مماثلة لدالة #py("beside") المعطاة أعلاه، ومجدداً بدلالة #py("beside") وعمليات الدوران المناسبة (من التمرين @ex:rotate).
])

#subheading([مستويات اللغة للتصميم المتين])

تستغل لغة الصور بعض الأفكار المحورية التي قدممناها حول التجريد مع الدوال والبيانات. فالتجريدات الأساسية للبيانات، الرسّامون، تُنفَّذ باستخدام تمثيلات دالية، مما يمكن اللغة من التعامل مع قدرات الرسم الأساسية المختلفة بطريقة موحدة. وتحقق وسائل الدمج خاصية الإغلاق، مما يتيح لنا بناء تصاميم معقدة بسهولة. وأخيراً، فإن جميع أدوات تجريد الدوال متاحة لنا لتجريد وسائل دمج الرسّامين.

وقد حصلنا أيضاً على لمحة عن فكرة حاسمة أخرى حول اللغات وتصميم البرامج. وهذه الفكرة هي نهج #idx("stratified design") #idx("design, stratified") #emph[التصميم المتدرج] (#en[stratified design])، وهو المفهوم القائل بأن النظام المعقد يجب أن يُبنى كمتتالية من المستويات التي توصف باستخدام متتالية من اللغات. يُبنى كل مستوى من خلال دمج أجزاء تُعتبر أولية في ذلك المستوى، وتُستخدم الأجزاء المبنية في كل مستوى كأوليات في المستوى التالي. وتملك اللغة المستخدمة في كل مستوى من مستويات التصميم المتدرج أوليات، ووسائل دمج، ووسائل تجريد مناسبة لمستوى التفاصيل هذا.

يتخلل التصميم المتدرج هندسة الأنظمة المعقدة. فعلى سبيل المثال، في هندسة الحاسوب، يُدمج بين المقاومات والترانزستورات (وتُوصَف باستخدام لغة الدوائر التناظرية) لإنتاج أجزاء مثل بوابات "و" (#en[AND]) وبوابات "أو" (#en[OR])، والتي تشكل الأوليات للغة تصميم الدوائر الرقمية.#footnote[يصف القسم @sec:circuit-simulator إحدى هذه اللغات.] وتُدمج هذه الأجزاء لبناء المعالجات، وبنى النواقل، وأنظمة الذاكرة، والتي تُدمج بدورها لتشكيل الحواسيب، باستخدام لغات مناسبة لمعمارية الحاسوب. وتُدمج الحواسيب لتشكيل أنظمة موزعة، باستخدام لغات مناسبة لوصف التوصيلات الشبكية، وهكذا.

كمثال صغير على التدرج، تستخدم لغة الصور الخاصة بنا عناصر أولية (رسّامين أوليين) تحدد النقاط والخطوط لتوفير أشكال رسّام مثل #py("rogers"). وركز الجزء الأكبر من وصفنا للغة الصور على دمج هذه الأوليات، باستخدام مدمجات هندسية مثل #py("beside") و #py("below").
وعملنا أيضاً على مستوى أعلى، معتبرين #py("beside") و #py("below") كأوليات يُتلاعب بها في لغة تلتقط عملياتها، مثل #py("square_of_four")، الأنماط الشائعة لدمج المدمجات الهندسية.

يساعد التصميم المتدرج في جعل البرامج #idx("robustness") #emph[متينة] (#en[robust])، أي أنه يجعل من المرجح أن التغييرات الصغيرة في المواصفات ستتطلب تغييرات صغيرة مقابلة في البرنامج. فعلى سبيل المثال، نفترض أننا أردنا تغيير الصورة القائمة على #py("wave") الموضحة في الشكل @fig:sqlimit-designs. كان بإمكاننا العمل عند أدنى مستوى لتغيير المظهر التفصيلي لعنصر #py("wave")؛ وكان بإمكاننا العمل عند المستوى المتوسط لتغيير الطريقة التي يكرر بها #py("corner_split") العنصر #py("wave")؛ وكان بإمكاننا العمل عند أعلى مستوى لتغيير كيفية ترتيب #py("square_limit") للنسخ الأربع للزاوية. وبشكل عام، يوفر كل مستوى من مستويات التصميم المتدرج مفردات مختلفة للتعبير عن خصائص النظام، ونوعاً مختلفاً من القدرة على تغييره.

#exercise(label-name: <ex:2_52>, [
أجرِ تغييرات على حد المربع لـ #py("wave") الموضح في الشكل @fig:sqlimit-designs بالعمل في كل من المستويات الموصوفة أعلاه. وعلى وجه الخصوص:

+ أضف بعض القطع المستقيمة إلى الرسّام الأولي #py("wave") من التمرين @ex:making-wave (لإضافة ابتسامة، على سبيل المثال).
+ غيِّر النمط المنشأ بوساطة #py("corner_split") (على سبيل المثال، باستخدام نسخة واحدة فقط من صور #py("up_split") و #py("right_split") بدلاً من نسختين).
+ عدل النسخة من #idx("squarelimit") #py("square_limit") التي تستخدم #idx("squareoffour") #py("square_of_four") لتجميع الزوايا في نمط مختلف. (على سبيل المثال، يمكنك جعل مستر روجرز الكبير ينظر للخارج من كل زاوية من زوايا المربع.)
])

#idx("picture language")
