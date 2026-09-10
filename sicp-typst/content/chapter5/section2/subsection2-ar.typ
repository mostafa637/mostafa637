// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([المُجَمِّع], label-name: <sec:assembler>)

#idx("assembler")

يحول المُجَمِّع (#en[assembler]) تسلسل تعليمات وحدة التحكم للآلة إلى قائمة مقابلة من تعليمات الآلة، كل منها مع دالة التنفيذ الخاصة بها. بشكل عام، تشبه آلة التجميع إلى حد كبير المُقيِّمات التي درسنا في الفصل @chap:meta—هناك لغة إدخال (في هذه الحالة، لغة آلة المسجّلات) ويجب علينا إجراء إجراء مناسب لكل نوع من المكونات في اللغة.

تقنية إنتاج دالة تنفيذ لكل تعليمة هي بالضبط ما استخدمناه في القسم @sec:separating-analysis لتسريع المُقيِّم عن طريق فصل التحليل عن التنفيذ في وقت التشغيل. كما رأينا في الفصل @chap:meta، يمكن إجراء الكثير من
#idx("syntactic analysis, separated from execution", sub: "in register-machine simulator")
التحليل النحوي المفيد لتعبيرات #en[Python] دون معرفة القيم الفعلية للأسماء. هنا، وبشكل مماثل، يمكن إجراء الكثير من التحليل المفيد لتعبيرات لغة آلة المسجّلات دون معرفة المحتويات الفعلية لمسجّلات الآلة. على سبيل المثال، يمكننا استبدال الإشارات إلى المسجّلات بمؤشرات إلى كائنات المسجّل، ويمكننا استبدال الإشارات إلى التسميات بمؤشرات إلى المكان في تسلسل التعليمات الذي تحدده التسمية.

قبل أن يتمكن من إنتاج دوال تنفيذ التعليمات، يجب أن يعرف المُجَمِّع ما تشير إليه جميع التسميات، لذا يبدأ بفحص تسلسل وحدة التحكم لفصل التسميات عن التعليمات. أثناء فحص وحدة التحكم، يبني كلاً من قائمة التعليمات وجدولاً يربط كل تسمية بمؤشر في تلك القائمة. ثم يقوم المُجَمِّع بتعزيز قائمة التعليمات عن طريق إدراج دالة التنفيذ لكل تعليمة.

الدالة #py("assemble") هي المدخل الرئيسي للمُجَمِّع. تأخذ تسلسل وحدة التحكم ونموذج الآلة كوسائط وتُرجع تسلسل التعليمات المخزن في النموذج.
تستدعي الدالة #py("assemble") الدالة #py("extract_labels") لبناء قائمة التعليمات الأولية وجدول التسميات من وحدة التحكم المحددة. الوسيط الثاني لـ #py("extract_labels") هو دالة يتم استدعاؤها لمعالجة هذه النتائج: تستخدم هذه الدالة #py("update_insts") لإنتاج دوال تنفيذ التعليمات وإدراجها في قائمة التعليمات، وتُرجع القائمة المعدلة.
#idx("assemble", decl: true)
#snippet(```python
def assemble(controller, machine):
    def receive(insts, labels):
        update_insts(insts, labels, machine)
        return insts
    return extract_labels(controller, receive)
```)

تأخذ الدالة #py("extract_labels") قائمة #py("controller") ودالة #py("receive") كوسائط. سيتم استدعاء الدالة #py("receive") بقيمتين: (1) قائمة #py("insts") من بنى بيانات التعليمات، تحتوي كل منها على تعليمة من #py("controller")؛ و (2) جدول يسمى #py("labels")، يربط كل تسمية من #py("controller") بالموقع في القائمة #py("insts") الذي تحدده التسمية.

#idx("extractlabels", decl: true)
#snippet(```python
def extract_labels(controller, receive):
    def incorporate(insts, labels):
        next_element = head(controller)
        return (receive(insts,
                        pair(make_label_entry(next_element, insts),
                             labels))
                if is_string(next_element)
                else receive(pair(make_inst(next_element), insts),
                             labels))
    return (receive(None, None) if is_null(controller)
            else extract_labels(tail(controller), incorporate))
```)

تظعمل الدالة #py("extract_labels") عن طريق الفحص التتابعي لعناصر #py("controller") وتجميع #py("insts") و #py("labels"). إذا كان العنصر سلسلة نصية (وبالتالي تسمية)، يضاف إدخال مناسب إلى جدول #py("labels"). بخلاف ذلك، يتم تجميع العنصر في قائمة #py("insts").#footnote[استخدام دالة
#idx("receive function")
#py("receive")
هنا هو طريقة لجعل #py("extract_labels") تُرجع قيمتين فعليًا—#py("labels") و #py("insts")—دون إنشاء بنية بيانات مرارًا وتكرارًا صراحة لحملها. هناك تنفيذ بديل يُرجع زوجًا صريحًا من القيم هو:
#idx("extractlabels", decl: true)
#snippet(```python
def extract_labels(controller):
    if is_null(controller):
        return pair(None, None)
    else:
        result = extract_labels(tail(controller))
        insts = head(result)
        labels = tail(result)
        next_element = head(controller)
        return (pair(insts, pair(make_label_entry(next_element, insts), labels))
                if is_string(next_element)
                else pair(pair(make_inst(next_element), insts), labels))
```)

والذي يتم استدعاؤه بواسطة #py("assemble") كما يلي:
#idx("assemble", decl: true)
#snippet(```python
def assemble(controller, machine):
    result = extract_labels(controller)
    insts = head(result)
    labels = tail(result)
    update_insts(insts, labels, machine)
    return insts
```)

يمكنك اعتبار استخدامنا لـ #py("receive") كاستعراض لطريقة أنيقة لـ
#idx("returning multiple values")
إرجاع قيم متعددة، أو مجرد عذر لإظهار خدعة برمجية. يسمى الوسيط مثل #py("receive") الذي يعد الدالة التالية التي سيتم استدعاؤها بـ
#idx("continuation", sub: "in register-machine simulator")
"المواصلة" (#en[continuation]). تذكر أننا استخدمنا أيضًا المواصلات لتنفيذ بنية التحكم بالتتبع التراجعي في مُقيِّم #py("amb") في القسم @sec:amb-implementation.]

تعدل الدالة #py("update_insts") قائمة التعليمات، التي تحتوي في البداية فقط على تعليمات وحدة التحكم، لتشمل دوال التنفيذ المقابلة:
#idx("updateinsts", decl: true)
#snippet(```python
def update_insts(insts, labels, machine):
    pc = get_register(machine, "pc")
    flag = get_register(machine, "flag")
    stack = machine("stack")
    ops = machine("operations")
    return for_each(
        lambda inst: set_inst_execution_fun(
            inst,
            make_execution_function(inst_controller_instruction(inst),
                                    labels, machine, pc, flag, stack, ops)),
        insts)
```)

تقرن بنية بيانات تعليمة الآلة ببساطة تعليمة وحدة التحكم ودالة التنفيذ المقابلة. تكون دالة التنفيذ غير متاحة بعد عندما تنشئ #py("extract_labels") التعليمة، ويتم إدراجها لاحقًا بواسطة #py("update_insts").
#idx("makeinst", decl: true)#idx("instcontrollerinstruction", decl: true)#idx("instexecutionfun", decl: true)#idx("setinstexecutionfun", decl: true)
#snippet(```python
def make_inst(inst_controller_instruction):
    return pair(inst_controller_instruction, None)
def inst_controller_instruction(inst):
    return head(inst)
def inst_execution_fun(inst):
    return tail(inst)
def set_inst_execution_fun(inst, fun):
    set_tail(inst, fun)
```)

لا تُستخدم تعليمة وحدة التحكم بواسطة محاكينا، ولكنها مفيدة للاحتفاظ بها لتصحيح الأخطاء (انظر التمرين @ex:reg-machine-instruction-trace).

عناصر جدول التسميات هي أزواج:
#idx("makelabelentry", decl: true)
#snippet(```python
def make_label_entry(label_name, insts):
    return pair(label_name, insts)
```)

سوف يتم البحث عن الإدخالات في الجدول باستخدام:
#idx("lookuplabel", decl: true)
#snippet(```python
def lookup_label(labels, label_name):
    val = assoc(label_name, labels)
    return (error("undefined label -- assemble", label_name) if is_undefined(val)
            else tail(val))
```)

#exercise(label-name: <ex:5_8>, [
كود آلة المسجّلات التالي غامض، لأن التسمية #py("here") معرفة أكثر من مرة:

#snippet(```python
"start",
  go_to(label("here")),
"here",
  assign("a", constant(3)),
  go_to(label("there")),
"here",
  assign("a", constant(4)),
  go_to(label("there")),
"there",
```)

مع المحاكي كما هو مكتوب، ماذا ستكون محتويات المسجّل #py("a") عندما يصل التحكم إلى #py("there")؟ عدل الدالة #py("extract_labels") بحيث يصدر المُجَمِّع خطأً إذا تم استخدام نفس اسم التسمية للإشارة إلى موقعين مختلفين.
])

#idx("assembler")
