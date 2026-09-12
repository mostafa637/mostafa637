// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([البرمجة الموجهة بالبيانات والتجمعية], label-name: <sec:data-directed>)

#idx("data-directed programming")
#idx("additivity")

تسمى الاستراتيجية العامة لفحص نوع عنصر بيانات واستدعاء دالة مناسبة بـ #idx("modularity", sub: "through dispatching on type") #idx("dispatching", sub: "on type") #idx("type(s)", sub: "dispatching on") #emph[الإرسال بناءً على النوع] (#en[dispatching on type]). وهذه استراتيجية قوية لتحقيق النمطية في تصميم النظام. ومن ناحية أخرى، فإن تنفيذ الإرسال كما في القسم @sec:manifest-types يملك نقطتي ضعف رئيسيتين. إحدى نقطتي الضعف هي أن دوال الواجهة العامة (#py("real_part") و #py("imag_part") و #py("magnitude") و #py("angle")) يجب أن تكون على علم بجميع التمثيلات المختلفة. فعلى سبيل المثال، نفترض أننا أردنا دمج تمثيل جديد للأعداد المركبة في نظام الأعداد المركبة لدينا. فسنحتاج إلى تحديد هذا التمثيل الجديد بنوع، ثم إضافة بند إلى كل دالة من دوال الواجهة العامة لفحص النوع الجديد وتطبيق المحدد المناسب لذلك التمثيل.

ونقطة ضعف أخرى لهذه التقنية هي أنه على الرغم من أنه يمكن تصميم التمثيلات الفردية بشكل منفصل، يجب أن نضمن عدم وجود دالتين في النظام بأكمله تملكان الاسم نفسه. ولهذا السبب كان على #en[Ben] و #en[Alyssa] تغيير أسماء دالهما الأصلية من القسم @sec:representations-complex-numbers.

والمسألة الكامنة وراء كلتا نقطتي الضعف هاتين هي أن تقنية تنفيذ الواجهات العامة ليست #emph[تجمعية] (#en[additive]). إذ يجب على الشخص الذي ينفذ دوال المحددات العامة تعديل تلك الدوال في كل مرة يُثبَّت فيها تمثيل جديد، ويجب على الأشخاص الذين يربطون التمثيلات الفردية تعديل الشفرة الخاصة بهم لتجنب تعارض الأسماء. وفي كل حالة من هذه الحالات، تكون التغييرات التي يجب إجراؤها على الشفرة مباشرة، ولكن يجب إجراؤها على أي حال، وتُعدّ هذه مصدراً للإزعاج والخطأ. ولا تُمثل هذه مشكلة كبيرة لنظام الأعداد المركبة كما هو قائم الآن، ولكن نفترض أنه لم يكن هناك تمثيلان بل مئات التمثيلات المختلفة للأعداد المركبة. ونفترض أنه كانت هناك العديد من المحددات العامة المراد صيانتها في واجهة البيانات المجردة. ونفترض، في الواقع، أنه لم يكن أي مبرمج على علم بجميع دوال الواجهة أو جميع التمثيلات. والمشكلة حقيقية ويجب معالجتها في برامج مثل أنظمة إدارة قواعد البيانات واسعة النطاق.

وما نحتاجه هو وسيلة لتقسيم تصميم النظام إلى وحدات نمطية بشكل أكبر. وتوفر ذلك تقنية البرمجة المعروفة بـ #emph[البرمجة الموجهة بالبيانات] (#en[data-directed programming]). ولفهم كيفية عمل البرمجة الموجهة بالبيانات، نبدأ بالملاحظة القائلة بأننا كلما تعاملنا مع مجموعة من العمليات العامة الشائعة لمجموعة من الأنواع المختلفة، فإننا، في الواقع، نتعامل مع جدول ثنائي الأبعاد يحتوي على العمليات الممكنة على أحد المحورين والأنواع الممكنة على المحور الآخر. وإدخالات الجدول هي الدوال التي تنفذ كل عملية لكل نوع من الوسائط المعطاة. وفي نظام الأعداد المركبة المطور في القسم السابق، كان التناظر بين اسم العملية، ونوع البيانات، والدالة الفعلية موزعاً بين البنود الشرطية المختلفة في دوال الواجهة العامة. ولكن كان من الممكن تنظيم المعلومات نفسها في جدول، كما هو موضح في
الشكل @fig:operator-table.

البرمجة الموجهة بالبيانات هي تقنية تصميم برامج تعمل مع مثل هذا الجدول (#idx("table", sub: "for data-directed programming")) مباشرة. في السابق، نفذنا الآلية التي تربط شفرة حساب الأعداد المركبة بحزمتي التمثيل، كمجموعة من الدوال تؤدي كل واحدة منها إرسالًا صريحاً بناءً على النوع. وهنا سننفذ الواجهة كدالة واحدة تبحث عن التركيبة بين اسم العملية ونوع الوسيط في الجدول للعثور على الدالة الصحيحة المراد تطبيقها، ثم تطبقها على محتويات الوسيط. وإذا قمنا بذلك، فلإضافة حزمة تمثيل جديدة إلى النظام لا نحتاج إلى تغيير أي دوال قائمة؛ بل نحتاج فقط إلى إضافة إدخالات جديدة إلى الجدول.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-63.svg", width: 59%), caption: [جدول العمليات لنظام الأعداد المركبة.], label-name: <fig:operator-table>)

لتنفيذ هذه الخطة، نفترض أن لدينا دالتين، #py("put") و #py("get")، للتلاعب بـ جدول العمليات والأنواع (#idx("operation-and-type table")):

- #idx("put") #py("put(")#meta("op")#py(",")#meta("type")#py(",")#meta("item")#py(")") \ يثبت #meta("item") في الجدول، مفهرساً بـ #meta("op") و #meta("type").
- #idx("get") #py("get(")#meta("op")#py(",")#meta("type")#py(")") \ يبحث عن الإدخال #meta("op"), #meta("type") في الجدول ويرجع العنصر الموجود هناك. وإذا لم يُعثر على أي عنصر، فإن #py("get") ترجع قيمة أولية فريدة يُشار إليها بالكلمة المفتاحية #idx("None (keyword)") #py("None") ويتعرف عليها المحمول الأولي #idx("isnone (primitive function)") #py("is_none").

في الوقت الحالي، يمكننا افتراض أن #py("put") و #py("get") متضمنتان في لغتنا. وفي الفصل @chap:state (القسم @sec:tables) سنرى كيفية تنفيذ هذه العمليات وغيرها للتلاعب بالجداول.

إليك كيفية استخدام البرمجة الموجهة بالبيانات في نظام الأعداد المركبة. ينفذ #en[Ben]، الذي طور التمثيل المستطيلي، شفرته تماماً كما فعل أصلًا. إذ يعرّف مجموعة من الدوال أو #idx("package") #idx("package", sub: "rectangular representation") #idx("rectangular package") #emph[حزمة]، ويربط هذه ببقية النظام عن طريق إضافة إدخالات إلى الجدول تخبر النظام بكيفية العمل على الأعداد المستطيلية. ويتم ذلك عن طريق استدعاء الدالة التالية:

#idx("installrectangularpackage", decl: true)
#snippet(```python
def install_rectangular_package():
    # internal functions
    def real_part(z): return head(z)
    def imag_part(z): return tail(z)
    def make_from_real_imag(x, y): return pair(x, y)
    def magnitude(z):
        return math_sqrt(square(real_part(z)) + square(imag_part(z)))
    def angle(z):
        return math_atan2(imag_part(z), real_part(z))
    def make_from_mag_ang(r, a):
        return pair(r * math_cos(a), r * math_sin(a))

    # interface to the rest of the system
    def tag(x): return attach_tag("rectangular", x)
    put("real_part", llist("rectangular"), real_part)
    put("imag_part", llist("rectangular"), imag_part)
    put("magnitude", llist("rectangular"), magnitude)
    put("angle", llist("rectangular"), angle)
    put("make_from_real_imag", "rectangular",
        lambda x, y: tag(make_from_real_imag(x, y)))
    put("make_from_mag_ang", "rectangular",
        lambda r, a: tag(make_from_mag_ang(r, a)))
    return "done"
```)

لاحظ أن الدوال الداخلية هنا هي الدوال نفسها من القسم @sec:representations-complex-numbers التي كتبها #en[Ben] عندما كان يعمل في عزلة. ولا يلزم إجراء أي تغييرات لربطها ببقية النظام. علاوة على ذلك، وبما أن تعاريف الدوال هذه داخلية لدالة التثبيت، فلا يتوجب على #en[Ben] القلق بشأن تعارض الأسماء مع دوال أخرى خارج الحزمة المستطيلية. ولربط هذه ببقية النظام، يثبت #en[Ben] دالة #py("real_part") الخاصة به تحت اسم العملية #py("real_part") والنوع #py("llist(\"rectangular\")")، وبالمثل للمحددات الأخرى.#footnote[نستخدم القائمة المترابطة #py("llist(\"rectangular\")") بدلاً من السلسلة النصية #py("\"rectangular\"") للسماح بإمكانية وجود عمليات بوسائط متعددة، ليست جميعها من النوع نفسه.] وتعرِف الواجهة أيضاً المُنشِئات المراد استخدامها بوساطة النظام الخارجي.#footnote[النوع الذي تُثبَّت المُنشِئات تحته لا يلزم أن يكون قائمة مترابطة لأن المُنشِئ يُستخدم دائماً لصنع كائن من نوع محدد واحد.] وهذه متطابقة مع المُنشِئات التي عرّفها #en[Ben] داخلياً، باستثناء أنها ترفق الوسم.

وحزمة #en[Alyssa] القطبية (#idx("package", sub: "polar representation") #idx("polar package")) مماثلة:
#idx("installpolarpackage", decl: true)
#snippet(```python
def install_polar_package():
    # internal functions
    def magnitude(z): return head(z)
    def angle(z): return tail(z)
    def make_from_mag_ang(r, a): return pair(r, a)
    def real_part(z):
        return magnitude(z) * math_cos(angle(z))
    def imag_part(z):
        return magnitude(z) * math_sin(angle(z))
    def make_from_real_imag(x, y):
        return pair(math_sqrt(square(x) + square(y)),
                    math_atan2(y, x))

    # interface to the rest of the system
    def tag(x): return attach_tag("polar", x)
    put("real_part", llist("polar"), real_part)
    put("imag_part", llist("polar"), imag_part)
    put("magnitude", llist("polar"), magnitude)
    put("angle", llist("polar"), angle)
    put("make_from_real_imag", "polar",
        lambda x, y: tag(make_from_real_imag(x, y)))
    put("make_from_mag_ang", "polar",
        lambda r, a: tag(make_from_mag_ang(r, a)))
    return "done"
```)

على الرغم من أن كلًا من #en[Ben] و #en[Alyssa] ما زالا يستخدما دالهما الأصلية المعرفة بالأسماء نفسها لدى كلٍّ منهما (مثل #py("real_part"))، فإن هذه الإعلانات أصبحت الآن داخلية لدوال مختلفة (انظر القسم @sec:block-structure)، لذا لا يوجد تعارض في الأسماء.

تصل محددات حساب الأعداد المركبة إلى الجدول عن طريق دالة "عملية" العامة المسماة #py("apply_generic")، والتي تطبق عمليةً عامة على بعض الوسائط. تبحث الدالة #py("apply_generic") في الجدول تحت اسم العملية وأنواع الوسائط وتطبق الدالة الناتجة إذا كانت موجودة:#footnote[تستخدم الدالة #py("apply_generic") الدالة #idx("applyinunderlyingjavascript") #py("apply_in_underlying_javascript") المعطاة في القسم @sec:running-eval (الحاشية @foot:vector-array)، والتي تأخذ وسيطين، دالةً وقائمةً مترابطة، وتطبق الدالة باستخدام العناصر الموجودة في القائمة المترابطة كوسائط. فعلى سبيل المثال، #snippet(```python apply_in_underlying_javascript(sum_of_squares, llist(1, 3)) ```) ترجع 10.]

#idx("applygeneric", decl: true)
#snippet(```python
def apply_generic(op, args):
    type_tags = map(type_tag, args)
    fun = get(op, type_tags)
    return (apply_in_underlying_javascript(fun, map(contents, args))
            if not is_none(fun)
            else error("no method for these types -- apply_generic",
                       llist(op, type_tags)))
```)

باستخدام #py("apply_generic")، يمكننا تعريف محدداتنا العامة كما يلي:
#idx("realpart", sub: "data-directed", decl: true)#idx("imagpart", sub: "data-directed", decl: true)#idx("magnitude", sub: "data-directed", decl: true)#idx("angle", sub: "data-directed", decl: true)
#snippet(```python
def real_part(z): return apply_generic("real_part", llist(z))

def imag_part(z): return apply_generic("imag_part", llist(z))

def magnitude(z): return apply_generic("magnitude", llist(z))

def angle(z): return apply_generic("angle", llist(z))
```)

لاحظ أن هذه لا تتغير على الإطلاق إذا تم إضافة تمثيل جديد إلى النظام.

ويمكننا أيضاً استخراج المُنشِئات من الجدول لتُستخدم بوساطة البرامج الخارجية عن الحزم في صنع أعداد مركبة من أجزاء حقيقية وتخيلية ومن سعات وزوايا. وكما في القسم @sec:manifest-types، نبني الأعداد المستطيلية كلما كانت لدينا أجزاء حقيقية وتخيلية، والأعداد القطبية كلما كانت لدينا سعات وزوايا:
#idx("makefromrealimag", decl: true)#idx("makefrommagang", decl: true)
#snippet(```python
def make_from_real_imag(x, y):
    return get("make_from_real_imag", "rectangular")(x, y)
def make_from_mag_ang(r, a):
    return get("make_from_mag_ang", "polar")(r, a)
```)

#exercise(label-name: <ex:data-directed-differentiation>, [
وصف القسم @sec:symbolic-differentiation برنامجاً يؤدي تفاضلاً رمزياً (#idx("symbolic differentiation") #idx("differentiation", sub: "symbolic")):

#snippet(```python
def deriv(exp, variable):
    return (0
            if is_number(exp)
            else (1 if is_same_variable(exp, variable) else 0)
            if is_variable(exp)
            else make_sum(deriv(addend(exp), variable),
                          deriv(augend(exp), variable))
            if is_sum(exp)
            else make_sum(make_product(multiplier(exp),
                                       deriv(multiplicand(exp), variable)),
                          make_product(deriv(multiplier(exp), variable),
                                       multiplicand(exp)))
            if is_product(exp)
            # more rules can be added here
            else error("unknown expression type -- deriv", exp))
```)

#snippet(```python
print(deriv(llist("*", llist("*", "x", "y"), llist("+", "x", 4)), "x"))
```)

#output(```python
print(deriv(llist("*", llist("*", "x", "y"), llist("+", "x", 4)), "x"))
```)

يمكننا اعتبار هذا البرنامج مؤدّياً لإرسالٍ بناءً على نوع التعبير المراد تفاضله. وفي هذه الحالة يكون "وسم النوع" للبيانات هو رمز العامل الجبري (مثل "+") والعملية التي تُؤدَّى هي #py("deriv"). ويمكننا تحويل هذا البرنامج إلى أسلوب موجه بالبيانات عن طريق إعادة كتابة دالة المشتقة الأساسية كالتالي:
#idx("deriv (symbolic)", sub: "data-directed", decl: true)
#snippet(```python
def deriv(exp, variable):
    return (0
            if is_number(exp)
            else (1 if is_same_variable(exp, variable) else 0)
            if is_variable(exp)
            else get("deriv", operator(exp))(operands(exp), variable))
def operator(exp): return head(exp)

def operands(exp): return tail(exp)
```)

+ اشرح ما تم القيام به أعلاه. لماذا لا يمكننا استيعاب المحمولين #py("is_number") و #py("is_variable") في الإرسال الموجَّه بالبيانات؟
+ اكتب الدوال لمشتقات المجاميع والجداءات، والشفرة المساعدة المطلوبة لتثبيتها في الجدول المستخدم بوساطة البرنامج أعلاه.
+ اختر أي قاعدة تفاضل إضافية تحبها، مثل قاعدة الأسس (التمرين @ex:deriv-exponentiation)، وثبتها في هذا النظام الموجه بالبيانات.
+ في هذا المعالِج الجبري البسيط، يكون نوع التعبير هو العامل الجبري الذي يربطه معاً. ولكن نفترض أننا فهرسنا الدوال بالطريقة المعاكسة، بحيث بدا سطر الإرسال في #py("deriv") ك#snippet(```python get(operator(exp), "deriv")(operands(exp), variable) ```) ما هي التغييرات المقابلة لنظام المشتقة المطلوبة؟
])

#exercise(label-name: <ex:2_74>, [
شركة #idx("data base", sub: "Insatiable Enterprises personnel") #en[Insatiable Enterprises, Inc.] هي شركة تكتل لا مركزية للغاية تتكون من عدد كبير من الأقسام المستقلة المنتشرة في جميع أنحاء العالم. وقد تم للتو توصيل المرافق الحاسوبية للشركة بواسطة نظام واجهة شبكية ذكي يجعل الشبكة بأكملها تبدو لأي مستخدم وكأنها حاسوب واحد. ويشعر رئيس شركة #en[Insatiable]، في محاولتها الأولى لاستغلال قدرة الشبكة على استخراج المعلومات الإدارية من ملفات الأقسام، بالفزع عندما تكتشف أنه على الرغم من أن جميع ملفات الأقسام قد نُفذت كبنى بيانات في #en[Python]، فإن بنية البيانات المحددة المستخدمة تختلف من قسم إلى آخر. ويُعقد اجتماع لمديري الأقسام عاجلاً للبحث عن استراتيجية لدمج الملفات تلبي احتياجات المقر الرئيسي مع الحفاظ على الاستقلالية الحالية للأقسام.

أظهر كيفية تنفيذ مثل هذه الاستراتيجية باستخدام #idx("data base", sub: "data-directed programming and") البرمجة الموجهة بالبيانات.
كمثال، نفترض أن سجلات الموظفين لكل قسم تتكون من ملف واحد يحتوي على مجموعة من السجلات المفهرسة بأسماء الموظفين. وتختلف بنية المجموعة من قسم إلى آخر. علاوة على ذلك، فإن سجل كل موظف هو نفسه مجموعة (مبنية بشكل مختلف من قسم إلى آخر) تحتوي على معلومات مفهرسة تحت معرفات مثل #py("address") و #py("salary"). وعلى وجه الخصوص:

+ نفذ للمقر الرئيسي دالة #py("get_record") ترجع سجل موظف محدد من ملف موظفين محدد. وينبغي أن تكون الدالة قابلة للتطبيق على ملف أي قسم. واشرح كيف ينبغي تنظيم ملفات الأقسام الفردية. وعلى وجه الخصوص، ما هي معلومات النوع التي يجب تقديمها؟
+ نفذ للمقر الرئيسي دالة #py("get_salary") ترجع معلومات الراتب من سجل موظف معطى من ملف موظفين لأي قسم. وكيف ينبغي تنظيم السجل لجعل هذه العملية تعمل؟
+ نفذ للمقر الرئيسي دالة #py("find_employee_record"). وينبغي لهذه الدالة البحث في جميع ملفات الأقسام عن سجل موظف معطى وإرجاع السجل. وافترض أن هذه الدالة تأخذ كوسيطين اسمَ موظف وقائمة مترابطة لجميع ملفات الأقسام.
+ عندما تستحوذ شركة #en[Insatiable] على شركة جديدة، ما هي التغييرات التي يجب إجراؤها لدمج معلومات الموظفين الجدد في النظام المركزي؟
])

#idx("data-directed programming")
#idx("additivity")

#subheading([تمرير الرسائل])

#idx("message passing")

الفكرة الرئيسية للبرمجة الموجهة بالبيانات هي التعامل مع العمليات العامة في البرامج عن طريق التعامل صراحة مع جداول العمليات والأنواع، مثل الجدول في الشكل @fig:operator-table.
وأسلوب البرمجة الذي استخدمناه في القسم @sec:manifest-types نظّم الإرسال المطلوب بناءً على النوع عن طريق جعل كل عملية تتولى أمر الإرسال الخاص بها. وفي الواقع، يقوم هذا بتفكيك جدول العمليات والأنواع إلى صفوف، حيث تمثل كل دالة عملية عامة صفاً من الجدول.

واستراتيجية التنفيذ البديلة هي تفكيك الجدول إلى أعمدة، وبدلاً من استخدام "عمليات ذكية" تُرسل بناءً على أنواع البيانات، العمل مع "كائنات بيانات ذكية" تُرسل بناءً على أسماء العمليات. ويمكننا القيام بذلك عن طريق ترتيب الأمور بحيث يُمثَّل كائن البيانات، مثل العدد المستطيلي، كدالة تأخذ كمدخل اسم العملية المطلوبة وتؤدي العملية المشار إليها. وفي مثل هذا الانضباط، يمكن كتابة #py("make_from_real_imag") كالتالي:
#idx("makefromrealimag", sub: "message-passing", decl: true)
#snippet(```python
def make_from_real_imag(x, y):
    def dispatch(op):
        return (x
                if op == "real_part"
                else y
                if op == "imag_part"
                else math_sqrt(square(x) + square(y))
                if op == "magnitude"
                else math_atan2(y, x)
                if op == "angle"
                else error("unknown op -- make_from_real_imag", op))
    return dispatch
```)

ودالة #py("apply_generic") المقابلة، والتي تطبق عمليةً عامة على وسيط، تغذي الآن ببساطة اسم العملية لكائن البيانات وتترك الكائن يقوم بالعمل:#footnote[إحدى قيود هذا التنظيم هي أنه يسمح فقط بالدوال العامة ذات الوسيط الواحد.]
#idx("applygeneric", sub: "with message passing", decl: true)
#snippet(```python
def apply_generic(op, arg): return head(arg)(op)
```)

لاحظ أن القيمة المعادة من #py("make_from_real_imag") هي دالة—الدالة الداخلية #py("dispatch"). وهذه هي الدالة التي تُستدعى عندما تطلب #py("apply_generic") أداء عملية ما.

يسمى أسلوب البرمجة هذا #emph[تمرير الرسائل] (#en[message passing]). ويأتي الاسم من الصورة القائلة بأن كائن البيانات هو كيان يستقبل اسم العملية المطلوبة ك"رسالة". وقد رأينا بالفعل مثالاً على تمرير الرسائل في القسم @sec:data-، حيث رأينا كيف يمكن تعريف #py("pair") و #py("head") و #py("tail") دون كائنات بيانات ولكن فقط مع الدوال. وهنا نرى أن تمرير الرسائل ليس خدعة رياضية بل تقنية مفيدة لتنظيم الأنظمة ذات العمليات العامة. وفي بقية هذا الفصل سنستمر في استخدام البرمجة الموجهة بالبيانات، بدلاً من تمرير الرسائل، لمناقشة العمليات الحسابية العامة. وفي الفصل @chap:state سنعود إلى تمرير الرسائل، وسنرى أنها يمكن أن تكون أداة قوية لتنظيم برامج المحاكاة.

#exercise(label-name: <ex:2_75>, [
نفذ المُنشِئ #idx("makefrommagang", sub: "message-passing") #py("make_from_mag_ang") بأسلوب تمرير الرسائل. وينبغي أن تكون هذه الدالة مماثلة لدالة #py("make_from_real_imag") المعطاة أعلاه.
])

#exercise(label-name: <ex:extend-generic>, [
مع تطور نظام كبير ذي عمليات عامة، قد يلزم أنواع جديدة من كائنات البيانات أو عمليات جديدة. ولكل من الاستراتيجيات الثلاث—العمليات العامة ذات الإرسال الصريح (#idx("dispatching", sub: "comparing different styles"))، والأسلوب الموجه بالبيانات، وأسلوب تمرير الرسائل—صِف التغييرات التي يجب إجراؤها على نظام ما من أجل إضافة أنواع جديدة أو عمليات جديدة. وأي تنظيم سيكون الأكثر ملاءمة لنظام يجب أن تُضاف فيه أنواع جديدة غالباً؟ وأيها سيكون الأكثر ملاءمة لنظام يجب أن تُضاف فيه عمليات جديدة غالباً؟
])
