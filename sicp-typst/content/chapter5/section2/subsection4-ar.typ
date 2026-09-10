// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([مراقبة أداء الآلة], label-name: <sec:monitor>)

#idx("register machine", sub: "monitoring performance")

المحاكاة مفيدة ليس فقط للتحقق من صحة تصميم آلة مقترح، بل أيضًا لقياس
#idx("simulation", sub: "for monitoring performance of register machine")
أداء الآلة. على سبيل المثال، يمكننا تثبيت "عدّاد" في برنامج المحاكاة الخاص بنا يقيس عدد عمليات المكدس المستخدمة في الحساب. للقيام بذلك، نعدل مكدسنا المحاكى لتتبع عدد المرات التي يتم فيها حفظ المسجّلات على المكدس وأقصى عمق يصل إليه المكدس، ونضيف رسالة إلى واجهة المكدس تطبع الإحصائيات، كما هو موضح أدناه.
نضيف أيضًا عملية إلى نموذج الآلة الأساسي لطباعة إحصائيات المكدس، من خلال تهيئة #py("the_ops") في #py("make_new_machine") إلى:
#idx("initializestack operation in register machine")#idx("printstackstatistics operation in register machine")
#snippet(```python
llist(llist("initialize_stack", lambda: stack("initialize")),
     llist("print_stack_statistics", lambda: stack("print_statistics")))
```)

إليك النسخة الجديدة من #py("make_stack"):
#idx("makestack", sub: "with monitored stack", decl: true)
#snippet(```python
def make_stack():
    stack = None
    number_pushes = 0
    max_depth = 0
    current_depth = 0
    def push(x):
        nonlocal stack, number_pushes, current_depth, max_depth
        stack = pair(x, stack)
        number_pushes = number_pushes + 1
        current_depth = current_depth + 1
        max_depth = math_max(current_depth, max_depth)
        return "done"
    def pop():
        nonlocal stack, current_depth
        if is_null(stack):
            error("empty stack -- pop")
        else:
            top = head(stack)
            stack = tail(stack)
            current_depth = current_depth - 1
            return top
    def initialize():
        nonlocal stack, number_pushes, max_depth, current_depth
        stack = None
        number_pushes = 0
        max_depth = 0
        current_depth = 0
        return "done"
    def print_statistics():
        display("total pushes = " + stringify(number_pushes))
        display("maximum depth = " + stringify(max_depth))
    def dispatch(message):
        return (push if message == "push"
                else pop() if message == "pop"
                else initialize() if message == "initialize"
                else print_statistics() if message == "print_statistics"
                else error("unknown request -- stack", message))
    return dispatch
```)

تصف التمارين من @ex:instruction-count إلى @ex:breakpoints ميزات مراقبة وتصحيح أخطاء مفيدة أخرى يمكن إضافتها إلى محاكي آلة المسجّلات.

#exercise(label-name: <ex:measure-fact>, [
قِس عدد عمليات الدفع (#py("push")) وأقصى عمق للمكدس المطلوب لحساب
#idx("factorial", sub: "stack usage, register machine")
$n!$ لقيم صغيرة مختلفة لـ $n$ باستخدام آلة المضروب الموضحة في الشكل @fig:fact-machine. من بياناتك، حدد صيغًا بدلالة $n$ لإجمالي عدد عمليات الدفع وأقصى عمق للمكدس المستخدم في حساب $n!$ لأي $n > 1$. لاحظ أن كلًا منها دالة خطية في $n$ وبالتالي تتحدد بثابتين. من أجل طباعة الإحصائيات، سيتعين عليك تعزيز آلة المضروب بتعليمات لتهيئة المكدس وطباعة الإحصائيات. قد ترغب أيضًا في تعديل الآلة بحيث تقرأ بشكل متكرر قيمة لـ $n$، وتحسب المضروب، وتطبع النتيجة (كما فعلنا لآلة GCD في الشكل @fig:gcd-with-io)، حتى لا تضطر إلى استدعاء #py("get_register_contents") و #py("set_register_contents") و #py("start") بشكل متكرر.
])

#exercise(label-name: <ex:instruction-count>, [
أضف
#idx("instruction counting")
#emph[عدّ التعليمات] (#en[instruction counting])
إلى محاكاة آلة المسجّلات.
أي، اجعل نموذج الآلة يتتبع عدد التعليمات المنفذة. قم بتوسيع واجهة نموذج الآلة لتقبل رسالة جديدة تطبع قيمة عدّاد التعليمات وتستعيد العدّاد إلى الصفر.
])

#exercise(label-name: <ex:reg-machine-instruction-trace>, [
عزز المحاكي لتوفير
#idx("instruction tracing")
#idx("tracing", sub: "instruction execution")
#emph[تتبع التعليمات] (#en[instruction tracing]).
أي قبل تنفيذ كل تعليمة، يجب على المحاكي طباعة التعليمة. اجعل نموذج الآلة يتقبل الرسالتين #py("trace_on") و #py("trace_off") لتشغيل وإيقاف التتبع.
])

#exercise(label-name: <ex:5_16>, [
وسّع تتبع التعليمات في التمرين @ex:reg-machine-instruction-trace بحيث يقوم المحاكي قبل طباعة تعليمة بطباعة أي تسميات تسبق تلك التعليمة مباشرة في تسلسل وحدة التحكم. احرص على القيام بذلك بطريقة لا تتداخل مع عدّ التعليمات (التمرين @ex:instruction-count).
سيتعين عليك جعل المحاكي يحتفظ بمعلومات التسمية اللازمة.
])

#exercise(label-name: <ex:5_17>, [
عدل الدالة #py("make_register") في القسم @sec:machine-model بحيث يمكن
#idx("register(s)", sub: "tracing")
#idx("tracing", sub: "register assignment")
تتبع المسجّلات. يجب أن تتقبل المسجّلات رسائل لتشغيل وإيقاف التتبع. عند تتبع مسجّل، يجب أن يؤدي إسناد قيمة للمسجّل إلى طباعة اسم المسجّل، والمحتويات القديمة للمسجّل، والمحتويات الجديدة المسندة. قم بتوسيع الواجهة لنموذج الآلة لتسمح لك بتشغيل وإيقاف التتبع لمسجّلات محددة في الآلة.
])

#exercise(label-name: <ex:breakpoints>, [
تريد أليسا ب. هاكر (#en[Alyssa P. Hacker]) ميزة
#idx("breakpoint")
#emph[نقطة التوقف] (#en[breakpoint]) في المحاكي لمساعدتها في تصحيح أخطاء تصميمات آلاتها. لقد تم توظيفك لتثبيت هذه الميزة لها. تريد أن تكون قادرة على تحديد مكان في تسلسل وحدة التحكم حيث سيتوقف المحاكي ويسمح لها بفحص حالة الآلة. عليك تنفيذ دالة:

#syntax("
set_breakpoint(", meta("machine"), ", ", meta("label"), ", ", meta("n"), ")
      ")

تقوم بضبط نقطة توقف قبل التعليمة الـ $n$ مباشرة بعد التسمية المعطاة. على سبيل المثال:

#snippet(```python
set_breakpoint(gcd_machine, "test_b", 4)
```)

تثبت نقطة توقف في #py("gcd_machine") قبل الإسناد إلى المسجّل #py("a") مباشرة.
عندما يصل المحاكي إلى نقطة التوقف، يجب أن يطبع التسمية وإزاحة نقطة التوقف ويتوقف عن تنفيذ التعليمات. يمكن لأليسا بعد ذلك استخدام #py("get_register_contents") و #py("set_register_contents") للتحكم في حالة الآلة المحاكاة. يجب أن تكون قادرة بعد ذلك على مواصلة التنفيذ عن طريق إدخال:

#syntax("
proceed_machine(", meta("machine"), ")
      ")

يجب أن تكون قادرة أيضًا على إزالة نقطة توقف محددة عن طريق:

#syntax("
cancel_breakpoint(", meta("machine"), ", ", meta("label"), ", ", meta("n"), ")
      ")

أو إزالة جميع نقاط التوقف عن طريق:

#syntax("
cancel_all_breakpoints(", meta("machine"), ")
      ")
])

#idx("register machine", sub: "simulator")
#idx("register-machine simulator")
#idx("simulation", sub: "of register machine")
#idx("register machine", sub: "monitoring performance")
