// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([نموذج الآلة], label-name: <sec:machine-model>)

يتم تمثيل نموذج الآلة الذي تنشئه #py("make_machine") كدالة ذات حالة محليّة باستخدام تقنيات تمرير الرسائل التي تم تطويرها في الفصل @chap:state. لبناء هذا النموذج، تبدأ #py("make_machine") باستدعاء الدالة #py("make_new_machine") لبناء أجزاء نموذج الآلة المشتركة بين جميع آلات المسجّلات. نموذج الآلة الأساسي هذا الذي تبنيه #py("make_new_machine") هو في الجوهر حاوية لبعض المسجّلات ومكدس، إلى جانب آلية تنفيذ تعالج تعليمات وحدة التحكم واحدة تلو الأخرى.

ثم تقوم الدالة #py("make_machine") بتوسيع هذا النموذج الأساسي (عن طريق إرسال رسائل إليه) لتضمين المسجّلات والعمليات ووحدة التحكم للآلة المعينة المحددة. أولاً، تقوم بتخصيص مسجّل في الآلة الجديدة لكل اسم من أسماء المسجّلات المحددة وتثبيت العمليات المحددة في الآلة. ثم تستخدم #idx("assembler") #emph[مُجَمِّعًا] (#en[assembler]) (الموصوف أدناه في القسم @sec:assembler) لتحويل قائمة وحدة التحكم إلى تعليمات للآلة الجديدة وتثبيتها كتسلسل تعليمات الآلة.
تُرجع الدالة #py("make_machine") نموذج الآلة المعدل كقيمة لها.

#idx("makemachine", decl: true)
#snippet(```python
def make_machine(register_names, ops, controller):
    machine = make_new_machine()
    for_each(lambda register_name: machine("allocate_register")(register_name),
             register_names)
    machine("install_operations")(ops)
    machine("install_instruction_sequence")(assemble(controller, machine))
    return machine
```)

#subheading([المسجّلات])

#idx("register(s)", sub: "representing")

سنمثل المسجّل كدالة ذات حالة محليّة، كما في الفصل @chap:state. تنشئ الدالة #py("make_register") مسجّلاً يحمل قيمة يمكن الوصول إليها أو تغييرها:
#idx("makeregister", decl: true)
#snippet(```python
def make_register(name):
    contents = "*unassigned*"
    def dispatch(message):
        def set_value(value):
            nonlocal contents
            contents = value
        return (contents if message == "get"
                else set_value if message == "set"
                else error("unknown request -- make_register", message))
    return dispatch
```)

تُستخدم الدوال التالية للوصول إلى المسجّلات:
#idx("getcontents", decl: true)#idx("setcontents", decl: true)
#snippet(```python
def get_contents(register):
    return register("get")
def set_contents(register, value):
    return register("set")(value)
```)

#subheading([المكدس])

#idx("stack", sub: "representing")

يمكننا أيضًا تمثيل المكدس كدالة ذات حالة محليّة. تنشئ الدالة #py("make_stack") مكدسًا تتكون حالته المحليّة من قائمة بالعناصر الموجودة على المكدس. يتقبل المكدس طلبات لـ #py("push") عنصر على المكدس، ولـ #py("pop") العنصر العلوي من المكدس وإرجاعه، ولـ #py("initialize") المكدس ليصبح فارغًا.

#idx("makestack", decl: true)
#snippet(```python
def make_stack():
    stack = None
    def push(x):
        nonlocal stack
        stack = pair(x, stack)
        return "done"
    def pop():
        nonlocal stack
        if is_null(stack):
            error("empty stack -- pop")
        else:
            top = head(stack)
            stack = tail(stack)
            return top
    def initialize():
        nonlocal stack
        stack = None
        return "done"
    def dispatch(message):
        return (push if message == "push"
                else pop() if message == "pop"
                else initialize() if message == "initialize"
                else error("unknown request -- stack", message))
    return dispatch
```)

تُستخدم الدوال التالية للوصول إلى المكدسات:
#idx("pop", decl: true)#idx("push", decl: true)
#snippet(```python
def pop(stack):
    return stack("pop")
def push(stack, value):
    return stack("push")(value)
```)

#subheading([الآلة الأساسية])

تنشئ الدالة #py("make_new_machine")، الموضحة في الشكل @fig:make-new-machine، كائنًا تتكون حالته المحليّة من مكدس، وتسلسل تعليمات فارغ في البداية، وقائمة عمليات تحتوي في البداية على عملية لـ
#idx("initializestack operation in register machine")
تهيئة المكدس، و
#idx("register table, in simulator")
#emph[جدول مسجّلات] يحتوي في البداية على مسجّلين، باسم
#idx("flag register")
#py("flag") و
#idx("pc register")
#py("pc")
#idx("program counter")
(اختصارًا لـ "عداد البرنامج" - #en[program counter]). تقوم الدالة الداخلية #py("allocate_register") بإضافة إدخالات جديدة إلى جدول المسجّلات، وتبحث الدالة الداخلية #py("lookup_register") عن المسجّلات في الجدول.

#sicp-figure([#snippet(```python
def make_new_machine():
    pc = make_register("pc")
    flag = make_register("flag")
    stack = make_stack()
    the_instruction_sequence = None
    the_ops = llist(llist("initialize_stack", lambda: stack("initialize")))
    register_table = llist(llist("pc", pc), llist("flag", flag))
    def allocate_register(name):
        nonlocal register_table
        if is_undefined(assoc(name, register_table)):
            register_table = pair(llist(name, make_register(name)),
                                  register_table)
        else:
            error("multiply defined register", name)
        return "register allocated"
    def lookup_register(name):
        val = assoc(name, register_table)
        return (error("unknown register", name) if is_undefined(val)
                else head(tail(val)))
    def execute():
        insts = get_contents(pc)
        if is_null(insts):
            return "done"
        else:
            inst_execution_fun(head(insts))()
            return execute()
    def dispatch(message):
        def start():
            set_contents(pc, the_instruction_sequence)
            return execute()
        def install_instruction_sequence(seq):
            nonlocal the_instruction_sequence
            the_instruction_sequence = seq
        def install_operations(ops):
            nonlocal the_ops
            the_ops = append(the_ops, ops)
        return (start() if message == "start"
                else install_instruction_sequence
                     if message == "install_instruction_sequence"
                else allocate_register if message == "allocate_register"
                else lookup_register if message == "get_register"
                else install_operations if message == "install_operations"
                else stack if message == "stack"
                else the_ops if message == "operations"
                else error("unknown request -- machine", message))
    return dispatch
```)], caption: [الدالة #idx("makenewmachine", decl: true) #py("make_new_machine") تنفذ نموذج الآلة الأساسي.], label-name: <fig:make-new-machine>)

يُستخدم المسجّل #py("flag") للتحكم في التفريع في الآلة المحاكاة. تُعيّن تعليمات #py("test") محتويات #py("flag") إلى نتيجة الاختبار (صحيح أو خطأ). وتحدد تعليمات #py("branch") ما إذا كانت ستتفرع أم لا عن طريق فحص محتويات #py("flag").

يحدد المسجّل #py("pc") تسلسل التعليمات أثناء تشغيل الآلة. يتم تنفيذ هذا التسلسل بواسطة الدالة الداخلية #py("execute"). في نموذج المحاكاة، كل تعليمة آلة هي بنية بيانات تتضمن دالة بدون وسائط، تسمى
#idx("instruction execution function")
#idx("execution function", sub: "in register-machine simulator")
#emph[دالة تنفيذ التعليمة]، بحيث يؤدي استدعاء هذه الدالة إلى محاكاة تنفيذ التعليمة. أثناء تشغيل المحاكاة، يشير #py("pc") إلى المكان في تسلسل التعليمات الذي يبدأ بالتعليمة التالية المُراد تنفيذها.
#idx("execute")
تحصل الدالة #py("execute") على تلك التعليمة، وتنفذها عن طريق استدعاء دالة تنفيذ التعليمة، وتكرر هذه الدورة حتى لا يتبقى المزيد من التعليمات للتنفيذ (أي حتى يشير #py("pc") إلى نهاية تسلسل التعليمات).

كجزء من تشغيلها، تقوم كل دالة تنفيذ تعليمة بتعديل #py("pc") للإشارة إلى التعليمة التالية المُراد تنفيذها. تغير التعليمتان #py("branch") و #py("go_to") المسجّل #py("pc") ليشار إلى الوجهة الجديدة. تقوم جميع التعليمات الأخرى ببساطة بقدُم #py("pc")، مما يجعله يشير إلى التعليمة التالية في التسلسل. لاحظ أن كل استدعاء لـ #py("execute") يستدعي #py("execute") مرة أخرى، ولكن هذا لا ينشئ حلقة لا نهائية لأن تشغيل دالة تنفيذ التعليمة يغير محتويات #py("pc").

تُرجع الدالة #py("make_new_machine") دالة توجيه تنفذ الوصول عبر تمرير الرسائل إلى الحالة الداخلية. لاحظ أن بدء الآلة يتحقق عن طريق ضبط #py("pc") على بداية تسلسل التعليمات واستدعاء #py("execute").

من أجل الراحة، نقدم واجهة بديلة لعملية #py("start") للآلة، بالإضافة إلى دوال لضبط وفحص محتويات المسجّل، كما هو محدد في بداية القسم @sec:simulator:
#idx("start register machine", decl: true)#idx("getregistercontents", decl: true)#idx("setregistercontents", decl: true)
#snippet(```python
def start(machine):
    return machine("start")
def get_register_contents(machine, register_name):
    return get_contents(get_register(machine, register_name))
def set_register_contents(machine, register_name, value):
    set_contents(get_register(machine, register_name), value)
    return "done"
```)

تستخدم هذه الدوال (والعديد من الدوال في القسمين @sec:assembler و @sec:ex-proc) ما يلي للبحث عن المسجّل ذي الاسم المعطى في آلة معينة:
#idx("getregister", decl: true)
#snippet(```python
def get_register(machine, reg_name):
    return machine("get_register")(reg_name)
```)
