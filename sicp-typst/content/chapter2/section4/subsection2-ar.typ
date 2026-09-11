// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([البيانات الموسومة], label-name: <sec:manifest-types>)

#idx("complex numbers", sub: "represented as tagged data")
#idx("tagged data")
#idx("data", sub: "tagged")

إحدى الطرق للنظر إلى تجريد البيانات هي كتطبيق لـ #idx("principle of least commitment") #idx("least commitment, principle of") "مبدأ التزام الحد الأدنى". في تنفيذ نظام الأعداد المركبة في القسم @sec:representations-complex-numbers، يمكننا استخدام إما تمثيل #en[Ben] المستطيلي وإما تمثيل #en[Alyssa] القطبي. ويسمح لنا حاجز التجريد المكون بوساطة محددات الاختيارات والبواني بتأجيل اختيار تمثيل ملموس لكائنات بياناتنا إلى آخر لحظة ممكنة، وبالتالي الاحتفاظ بأقصى قدر من المرونة في تصميم نظامنا.

ويمكن أخذ مبدأ التزام الحد الأدنى إلى أبعد من ذلك. فإذا أردنا، فيمكننا الحفاظ على الغموض في التمثيل حتى #emph[بعد] تصميم محددات الاختيارات والبواني، واختيار استخدام كل من تمثيل #en[Ben] #emph[و] تمثيل #en[Alyssa]. وإذا تم تضمين كلا التمثيلين في نظام واحد، فسنحتاج إلى طريقة ما لتمييز البيانات في الشكل القطبي عن البيانات في الشكل المستطيلي. وإلا، إذا طُلب منا، على سبيل المثال، إيجاد #py("magnitude") للزوج $(3,4)$، فأننا لن نعرف ما إذا كان يجب الإجابة بـ 5 (بتفسير العدد في الشكل المستطيلي) أو 3 (بتفسير العدد في الشكل القطبي). والطريقة المباشرة لإنجاز هذا التمييز هي تضمين #idx("type tag") #emph[وسم نوع] (#en[type tag])—السلسلة النصية #py("\"rectangular\"") أو #py("\"polar\"")—كجزء من كل عدد مركب. وعندما نحتاج إلى التلاعب بعدد مركب، يمكننا استخدام الوسم لتحديد محدد الاختيارات المراد تطبيقه.

وللتلاعب بالبيانات الموسومة، سنفترض أن لدينا دوال #py("type_tag") و #py("contents") تستخرج من كائن البيانات الوسم والمحتويات الفعلية (الإحداثيات القطبية أو المستطيلية، في حالة العدد المركب). وسوف نفترض أيضاً دالة #py("attach_tag") تأخذ وسماً ومحتويات وتنتج كائن بيانات موسوماً. والطريقة المباشرة لتنفيذ هذا هي استخدام بنية القوائم المترابطة العادية:
#idx("attachtag", decl: true)#idx("typetag", decl: true)#idx("contents", decl: true)
#snippet(```python
def attach_tag(type_tag, contents):
    return pair(type_tag, contents)
def type_tag(datum):
    return (head(datum)
            if is_pair(datum)
            else error("bad tagged datum -- type_tag", datum))
def contents(datum):
    return (tail(datum)
            if is_pair(datum)
            else error("bad tagged datum -- contents", datum))
```)

باستخدام #py("type_tag")، يمكننا تعريف محمولين #py("is_rectangular") و #py("is_polar")، يتعرفان على الأعداد المستطيلية والقطبية، على التوالي:
#idx("isrectangular", decl: true)#idx("ispolar", decl: true)
#snippet(```python
def is_rectangular(z):
    return type_tag(z) == "rectangular"
def is_polar(z):
    return type_tag(z) == "polar"
```)

مع وسوم الأنواع، يمكن لـ #en[Ben] و #en[Alyssa] الآن تعديل الشفرة الخاصة بهما بحيث يمكن لتمثيليهما المختلفين التعايش في النظام نفسه. وكلما بنى #en[Ben] عدداً مركباً، يوسمه كمستطيلي. وكلما بنيت #en[Alyssa] عدداً مركباً، توسمه كقطبي. بالإضافة إلى ذلك، يجب على #en[Ben] و #en[Alyssa] التأكد من أن أسماء دوالهما لا تتعارض. وإحدى الطرق للقيام بذلك هي أن يلحق #en[Ben] اللاحقة #py("rectangular") باسم كل دالة من دوال التمثيل الخاصة به وأن تلحق #en[Alyssa] اللاحقة #py("polar") بأسماء دوالها. إليك تمثيل #en[Ben] المستطيلي المعدل من القسم @sec:representations-complex-numbers:
#idx("realpartrectangular", decl: true)#idx("imagpartrectangular", decl: true)#idx("magnituderectangular", decl: true)#idx("anglerectangular", decl: true)#idx("makefromrealimagrectangular", decl: true)#idx("makefrommagangrectangular", decl: true)
#snippet(```python
def real_part_rectangular(z): return head(z)

def imag_part_rectangular(z): return tail(z)

def magnitude_rectangular(z):
    return math_sqrt(square(real_part_rectangular(z)) +
                     square(imag_part_rectangular(z)))
def angle_rectangular(z):
    return math_atan2(imag_part_rectangular(z),
                      real_part_rectangular(z))
def make_from_real_imag_rectangular(x, y):
    return attach_tag("rectangular", pair(x, y))
def make_from_mag_ang_rectangular(r, a):
    return attach_tag("rectangular",
                      pair(r * math_cos(a), r * math_sin(a)))
```)

وهنا تمثيل #en[Alyssa] القطبي المعدل:
#idx("realpartpolar", decl: true)#idx("imagpartpolar", decl: true)#idx("magnitudepolar", decl: true)#idx("anglepolar", decl: true)#idx("makefromrealimagpolar", decl: true)#idx("makefrommagangpolar", decl: true)
#snippet(```python
def real_part_polar(z):
    return magnitude_polar(z) * math_cos(angle_polar(z))
def imag_part_polar(z):
    return magnitude_polar(z) * math_sin(angle_polar(z))
def magnitude_polar(z): return head(z)

def angle_polar(z): return tail(z)

def make_from_real_imag_polar(x, y):
    return attach_tag("polar",
                      pair(math_sqrt(square(x) + square(y)),
                           math_atan2(y, x)))

def make_from_mag_ang_polar(r, a):
    return attach_tag("polar", pair(r, a))
```)

#idx("selector", sub: "generic")
#idx("generic function", sub: "generic selector")
يُنفذ كل محدد اختيارات عام كدالة تفحص وسم معاملها وتستدعي الدالة المناسبة للتعامل مع البيانات من ذلك النوع. فعلى سبيل المثال، للحصول على الجزء الحقيقي لعدد مركب، يفحص #py("real_part") الوسم لتحديد ما إذا كان سيستخدم #py("real_part_rectangular") لـ #en[Ben] أم #py("real_part_polar") لـ #en[Alyssa]. وفي كلا الحالتين، نستخدم #py("contents") لاستخراج البيانات المجرّدة غير الموسومة وإرسالها إلى الدالة المستطيلية أو القطبية حسب المطلوب:
#idx("realpart", sub: "with tagged data", decl: true)#idx("imagpart", sub: "with tagged data", decl: true)#idx("magnitude", sub: "with tagged data", decl: true)#idx("angle", sub: "with tagged data", decl: true)
#snippet(```python
def real_part(z):
    return (real_part_rectangular(contents(z))
            if is_rectangular(z)
            else real_part_polar(contents(z))
            if is_polar(z)
            else error("unknown type -- real_part", z))
def imag_part(z):
    return (imag_part_rectangular(contents(z))
            if is_rectangular(z)
            else imag_part_polar(contents(z))
            if is_polar(z)
            else error("unknown type -- imag_part", z))
def magnitude(z):
    return (magnitude_rectangular(contents(z))
            if is_rectangular(z)
            else magnitude_polar(contents(z))
            if is_polar(z)
            else error("unknown type -- magnitude", z))
def angle(z):
    return (angle_rectangular(contents(z))
            if is_rectangular(z)
            else angle_polar(contents(z))
            if is_polar(z)
            else error("unknown type -- angle", z))
```)

لتنفيذ العمليات الحسابية للأعداد المركبة، يمكننا استخدام الدوال نفسها #py("add_complex") و #py("sub_complex") و #py("mul_complex") و #py("div_complex") من القسم @sec:representations-complex-numbers، لأن محددات الاختيارات التي تستدعيها هي عامة، وسوف تعمل بالتالي مع أي من التمثيلين. فعلى سبيل المثال، دالة #py("add_complex") لا تزال:

#snippet(```python
def add_complex(z1, z2):
    return make_from_real_imag(real_part(z1) + real_part(z2),
                               imag_part(z1) + imag_part(z2))
```)

أخيراً، يجب أن نختار ما إذا كنا سنبني الأعداد المركبة باستخدام تمثيل #en[Ben] أم تمثيل #en[Alyssa]. وإحدى الخيارات المعقولة هي بناء الأعداد المستطيلية كلما كانت لدينا أجزاء حقيقية وتخيلية وبناء الأعداد القطبية كلما كانت لدينا سعات وزوايا:
#idx("makefromrealimag", decl: true)#idx("makefrommagang", decl: true)
#snippet(```python
def make_from_real_imag(x, y):
    return make_from_real_imag_rectangular(x, y)
def make_from_mag_ang(r, a):
    return make_from_mag_ang_polar(r, a)
```)

#sicp-figure(image("/images/img_javascript/ch2-Z-G-62.svg", width: 70%), caption: [بنية (#idx("complex-number arithmetic", sub: "structure of system")) نظام الحساب المركب العام.], label-name: <fig:generic-complex-system>)

يملك نظام الأعداد المركبة الناتج البنية الموضحة في الشكل @fig:generic-complex-system.
وقد تم تفكيك النظام إلى ثلاثة أجزاء مستقلة نسبياً: عمليات الحساب للأعداد المركبة، وتنفيذ #en[Alyssa] القطبي، وتنفيذ #en[Ben] المستطيلي. وكان من الممكن كتابة التنفيذين القطبي والمستطيلي بواسطة #en[Ben] و #en[Alyssa] يعملان بشكل منفصل، ويمكن استخدام كلاهما كتمثيلات أساسية بواسطة مبرمج ثالث ينفذ دوال الحساب المركب بدلالة الواجهة المجردة للبناء ومحددات الاختيارات.

وبما أن كل كائن بيانات موسوم بنوعه، فإن محددات الاختيارات تعمل على البيانات بطريقة عامة (#idx("selector", sub: "generic") #idx("generic function", sub: "generic selector")). أي أن كل محدد اختيارات معرَّف ليكون له سلوك يعتمد على النوع المحدد للبيانات التي يتم تطبيقها عليها. لاحظ الآلية العامة لربط التمثيلات المنفصلة: داخل تنفيذ تمثيل معطى (على سبيل المثال، حزمة #en[Alyssa] القطبية) يكون العدد المركب عبارة عن زوج غير موسوم (السعة، الزاوية). وعندما يعمل محدد اختيارات عام على عدد من نوع #py("polar")، فإنه ينزع الوسم ويمرر المحتويات إلى شفرة #en[Alyssa]. وعلى العكس من ذلك، عندما تبني #en[Alyssa] عدداً للاستخدام العام، فإنها توسمه بنوع بحيث يمكن التعرف عليه بشكل مناسب بوساطة الدوال ذات المستوى الأعلى. ويمكن أن يكون انضباط انتزاع وإرفاق الوسوم مع تمرير كائنات البيانات من مستوى إلى مستوى استراتيجية تنظيمية مهمة، كما سنرى في القسم @sec:generic-operators.
#idx("complex numbers", sub: "represented as tagged data")
#idx("tagged data")
#idx("data", sub: "tagged")
