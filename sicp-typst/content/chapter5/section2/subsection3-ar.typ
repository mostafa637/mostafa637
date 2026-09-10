// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([التعليمات ودوال تنفيذها], label-name: <sec:ex-proc>)

#idx("execution function", sub: "in register-machine simulator")
#idx("register-machine language", sub: "instructions")

يستدعي المُجَمِّع الدالة #py("make_execution_function") لإنتاج دالة التنفيذ لتعليمة وحدة التحكم.
مثل الدالة #py("analyze") في المُقيِّم في القسم @sec:separating-analysis، تقوم هذه الدالة بالتوجيه حسب نوع التعليمة لإنتاج دالة التنفيذ المناسبة.

تفاصيل دوال التنفيذ هذه تحدد معنى التعليمات الفردية في لغة آلة المسجّلات.

#idx("makeexecutionfunction", decl: true)
#snippet(```python
def make_execution_function(inst, labels, machine,
                            pc, flag, stack, ops):
    inst_type = type(inst)
    return (make_assign_ef(inst, machine, labels, ops, pc) if inst_type == "assign"
            else make_test_ef(inst, machine, labels, ops, flag, pc) if inst_type == "test"
            else make_branch_ef(inst, machine, labels, flag, pc) if inst_type == "branch"
            else make_go_to_ef(inst, machine, labels, pc) if inst_type == "go_to"
            else make_save_ef(inst, machine, stack, pc) if inst_type == "save"
            else make_restore_ef(inst, machine, stack, pc) if inst_type == "restore"
            else make_perform_ef(inst, machine, labels, ops, pc) if inst_type == "perform"
            else error("unknown instruction type -- assemble", inst))
```)

عناصر تسلسل #py("controller") المستلمة بواسطة #py("make_machine") والـمُمررة إلى #py("assemble") هي سلاسل نصية (للتسميات) وقوائم معلّمة (للتعليمات). العلامة (tag) في التعليمة هي سلسلة نصية تُعرّف نوع التعليمة، مثل #py("\"go_to\"")، والعناصر المتبقية في القائمة تحتوي على الوسائط، مثل وجهة #py("go_to").
يستخدم التوجيه في #py("make_execution_function"):
#idx("type in register machine", decl: true)
#snippet(```python
def type(instruction):
    return head(instruction)
```)

يتم بناء القوائم المعلّمة عندما يتم تقييم تعبير #py("list") الذي يمثل الوسيط الثالث لـ #py("make_machine"). كل وسيط لتلك الـ #py("list") إما أن يكون سلسلة نصية (والتي تُقيّم إلى نفسها) أو استدعاءً لـ مُنشئ (constructor) قائمة معلّمة لتعليمة. على سبيل المثال، #py("assign(\"b\", reg(\"t\"))") يستدعي المُنشئ #py("assign") مع الوسائط #py("\"b\"") ونتيجة استدعاء المُنشئ #py("reg") مع الوسيط #py("\"t\""). تحدد المُنشئات ووسائطها بناء التعليمات الفردية في لغة آلة المسجّلات. تعوض مُنشئات التعليمات والمُحددات (selectors) أدناه، إلى جانب مولدات دوال التنفيذ التي تستخدم المُحددات.

#subheading([تعليمة #py("assign")])

#idx("assign (in register machine)", sub: "simulating")

تنشئ الدالة #py("make_assign_ef") دوال تنفيذ لتعليمات #py("assign"):
#idx("makeassignef", decl: true)
#snippet(```python
def make_assign_ef(inst, machine, labels, operations, pc):
    target = get_register(machine, assign_reg_name(inst))
    value_exp = assign_value_exp(inst)
    value_fun = (make_operation_exp_ef(value_exp, machine, labels, operations)
                 if is_operation_exp(value_exp)
                 else make_primitive_exp_ef(value_exp, machine, labels))
    def execution_fun():
        set_contents(target, value_fun())
        advance_pc(pc)
    return execution_fun
```)

تنشئ الدالة #py("assign") تعليمات #py("assign").
يستخرج المُحددان #py("assign_reg_name") و #py("assign_value_exp") اسم المسجّل وتعبير القيمة من تعليمة #py("assign").

#idx("assign (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "assign", decl: true)#idx("assignregname", decl: true)#idx("assignvalueexp", decl: true)
#snippet(```python
def assign(register_name, source):
    return llist("assign", register_name, source)
def assign_reg_name(assign_instruction):
    return head(tail(assign_instruction))
def assign_value_exp(assign_instruction):
    return head(tail(tail(assign_instruction)))
```)

تبحث الدالة #py("make_assign_ef") عن اسم المسجّل بواسطة #py("get_register") لإنتاج كائن المسجّل المستهدف. يتم إرسال تعبير القيمة إلى #py("make_operation_exp_ef") إذا كانت القيمة هي نتيجة عملية، وتُرسل إلى #py("make_primitive_exp_ef") بخلاف ذلك. تقوم هذه الدوال (الموضحة أدناه) بتحليل تعبير القيمة وإنتاج دالة تنفيذ للقيمة. هذه دالة بدون وسائط، تسمى
#idx("valuefun")
#py("value_fun")، والتي سيتم تقييمها أثناء المحاكاة لإنتاج القيمة الفعلية المراد إسنادها للمسجّل. لاحظ أن عمل البحث عن اسم المسجّل وتحليل تعبير القيمة يتم مرة واحدة فقط، في وقت التجميع، وليس في كل مرة يتم فيها محاكاة التعليمة. هذا التوفير في العمل هو السبب في استخدامنا لـ دوال
#idx("syntactic analysis, separated from execution", sub: "in register-machine simulator")
التنفيذ، ويتوافق مباشرة مع التوفير في العمل الذي حصلنا عليه بفصل تحليل البرنامج عن التنفيذ في مُقيِّم القسم @sec:separating-analysis.

النتيجة المُرجعة بواسطة #py("make_assign_ef") هي دالة التنفيذ لتعليمة #py("assign"). عندما يتم استدعاء هذه الدالة (بواسطة دالة #py("execute") الخاصة بنموذج الآلة)، فإنها تضبط محتويات المسجّل المستهدف على النتيجة التي تم الحصول عليها بتنفيذ #py("value_fun"). ثم تقدم #py("pc") إلى التعليمة التالية عن طريق تشغيل الدالة:
#idx("advancepc", decl: true)
#snippet(```python
def advance_pc(pc):
    set_contents(pc, tail(get_contents(pc)))
```)

الدالة #py("advance_pc") هي إنهاء طبيعي لجميع التعليمات باستثناء #py("branch") و #py("go_to").

#subheading([التعليمات #py("test") و #py("branch") و #py("go_to")])

تتعامل الدالة #idx("test (in register machine)", sub: "simulating") #py("make_test_ef") مع تعليمات #py("test") بطريقة مماثلة. تستخرج التعبير الذي يحدد الشرط المراد اختباره وتولد دالة تنفيذ له. في وقت المحاكاة، يتم استدعاء دالة الشرط، وتُسند النتيجة إلى المسجّل #py("flag")، وتُقدم #py("pc"):
#idx("maketestef", decl: true)
#snippet(```python
def make_test_ef(inst, machine, labels, operations, flag, pc):
    condition = test_condition(inst)
    if is_operation_exp(condition):
        condition_fun = make_operation_exp_ef(condition, machine, labels, operations)
        def execution_fun():
            set_contents(flag, condition_fun())
            advance_pc(pc)
        return execution_fun
    else:
        error("bad test instruction -- assemble", inst)
```)

تنشئ الدالة #py("test") تعليمات #py("test"). يستخرج المُحدد #py("test_condition") الشرط من الاختبار.
#idx("testcondition", decl: true)#idx("test (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "test", decl: true)
#snippet(```python
def test(condition):
    return llist("test", condition)

def test_condition(test_instruction):
    return head(tail(test_instruction))
```)

تفحص دالة التنفيذ لتعليمة #py("branch") محتويات المسجّل #py("flag") وتضبط إما محتويات #py("pc") على وجهة التفريع (إذا تم أخذ التفريع) أو تقوم بتقديم #py("pc") ببساطة (إذا لم يتم أخذ التفريع). لاحظ أن الوجهة المشار إليها في تعليمة #idx("branch (in register machine)", sub: "simulating") #py("branch") يجب أن تكون تسمية، وتفرض الدالة #py("make_branch_ef") ذلك. لاحظ أيضًا أنه يتم البحث عن التسمية في وقت التجميع، وليس في كل مرة يتم فيها محاكاة تعليمة #py("branch").
#idx("makebranchef", decl: true)
#snippet(```python
def make_branch_ef(inst, machine, labels, flag, pc):
    dest = branch_dest(inst)
    if is_label_exp(dest):
        insts = lookup_label(labels, label_exp_label(dest))
        def execution_fun():
            if get_contents(flag):
                set_contents(pc, insts)
            else:
                advance_pc(pc)
        return execution_fun
    else:
        error("bad branch instruction -- assemble", inst)
```)

تنشئ الدالة #py("branch") تعليمات #py("branch"). يستخرج المُحدد #py("branch_dest") الوجهة من التفريع.
#idx("register-machine language", sub: "branch", decl: true)#idx("branch (in register machine)", sub: "instruction constructor", decl: true)#idx("branchdest", decl: true)
#snippet(```python
def branch(label):
    return llist("branch", label)

def branch_dest(branch_instruction):
    return head(tail(branch_instruction))
```)

تعليمة #idx("goto (in register machine)", sub: "simulating") #py("go_to") تشبه التفريع، باستثناء أنه يمكن تحديد الوجهة إما كتسمية أو كمسجّل، ولا يوجد شرط للفحص—يتم ضبط #py("pc") دائمًا على الوجهة الجديدة.
#idx("makegotoef", decl: true)
#snippet(```python
def make_go_to_ef(inst, machine, labels, pc):
    dest = go_to_dest(inst)
    if is_label_exp(dest):
        insts = lookup_label(labels, label_exp_label(dest))
        return lambda: set_contents(pc, insts)
    elif is_register_exp(dest):
        reg = get_register(machine, register_exp_reg(dest))
        return lambda: set_contents(pc, get_contents(reg))
    else:
        error("bad go_to instruction -- assemble", inst)
```)

تنشئ الدالة #py("go_to") تعليمات #py("go_to"). يستخرج المُحدد #py("go_to_dest") الوجهة من تعليمة #py("go_to").
#idx("goto (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "goto", decl: true)#idx("gotodest", decl: true)
#snippet(```python
def go_to(label):
    return llist("go_to", label)

def go_to_dest(go_to_instruction):
    return head(tail(go_to_instruction))
```)

#subheading([تعليمات أخرى])

تعليمات المكدس #py("save") و #py("restore") تستخدم المكدس ببساطة مع المسجّل المحدد وتقدم #py("pc"):
#idx("makesaveef", decl: true)#idx("makerestoreef", decl: true)#idx("save (in register machine)", sub: "simulating")#idx("restore (in register machine)", sub: "simulating")
#snippet(```python
def make_save_ef(inst, machine, stack, pc):
    reg = get_register(machine, stack_inst_reg_name(inst))
    def execution_fun():
        push(stack, get_contents(reg))
        advance_pc(pc)
    return execution_fun
def make_restore_ef(inst, machine, stack, pc):
    reg = get_register(machine, stack_inst_reg_name(inst))
    def execution_fun():
        set_contents(reg, pop(stack))
        advance_pc(pc)
    return execution_fun
```)

تنشئ الدالتان #py("save") و #py("restore") تعليمات #py("save") و #py("restore"). يستخرج المُحدد #py("stack_inst_reg_name") اسم المسجّل من هذه التعليمات.
#idx("save (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "save", decl: true)#idx("restore (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "restore", decl: true)#idx("stackinstregname", decl: true)
#snippet(```python
def save(reg):
    return llist("save", reg)

def restore(reg):
    return llist("restore", reg)

def stack_inst_reg_name(stack_instruction):
    return head(tail(stack_instruction))
```)

النوع الأخير من التعليمات، والذي تتعامل معه #idx("perform (in register machine)", sub: "simulating") #py("make_perform_ef")، يولد دالة تنفيذ للإجراء المراد تنفيذه. في وقت المحاكاة، يتم تنفيذ دالة الإجراء وتقديم #py("pc").
#idx("makeperformef", decl: true)
#snippet(```python
def make_perform_ef(inst, machine, labels, operations, pc):
    action = perform_action(inst)
    if is_operation_exp(action):
        action_fun = make_operation_exp_ef(action, machine, labels, operations)
        def execution_fun():
            action_fun()
            advance_pc(pc)
        return execution_fun
    else:
        error("bad perform instruction -- assemble", inst)
```)

تنشئ الدالة #py("perform") تعليمات #py("perform"). يستخرج المُحدد #py("perform_action") الإجراء من تعليمة #py("perform").
#idx("perform (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "perform", decl: true)#idx("performaction", decl: true)
#snippet(```python
def perform(action):
    return llist("perform", action)

def perform_action(perform_instruction):
    return head(tail(perform_instruction))
```)

#subheading([دوال التنفيذ للتعبيرات الفرعية])

قيمة تعبير #idx("reg (in register machine)", sub: "simulating") #py("reg")، أو #idx("label (in register machine)", sub: "simulating") #py("label")، أو #idx("constant (in register machine)", sub: "simulating") #py("constant") قد تكون مطلوبة للإسناد إلى مسجّل (#py("make_assign_ef") أعلاه) أو كمدخل لعملية (#py("make_operation_exp_ef") أدناه). تولد الدالة التالية دوال تنفيذ لإنتاج قيم لهذه التعبيرات أثناء المحاكاة:
#idx("makeprimitiveexpef", decl: true)
#snippet(```python
def make_primitive_exp_ef(exp, machine, labels):
    if is_constant_exp(exp):
        c = constant_exp_value(exp)
        return lambda: c
    elif is_label_exp(exp):
        insts = lookup_label(labels, label_exp_label(exp))
        return lambda: insts
    elif is_register_exp(exp):
        r = get_register(machine, register_exp_reg(exp))
        return lambda: get_contents(r)
    else:
        error("unknown expression type -- assemble", exp)
```)

تحدد بناء تعبيرات #py("reg") و #py("label") و #py("constant") بواسطة دوال المُنشئ التالية، إلى جانب المحمولات والمُحددات المقابلة.
#idx("reg (in register machine)", decl: true)#idx("register-machine language", sub: "reg", decl: true)#idx("isregisterexp", decl: true)#idx("registerexpreg", decl: true)#idx("constant (in register machine)", decl: true)#idx("register-machine language", sub: "constant", decl: true)#idx("isconstantexp", decl: true)#idx("constantexpvalue", decl: true)#idx("label (in register machine)", decl: true)#idx("register-machine language", sub: "label", decl: true)#idx("islabelexp", decl: true)#idx("labelexplabel", decl: true)
#snippet(```python
def reg(name):
    return llist("reg", name)

def is_register_exp(exp):
    return is_tagged_list(exp, "reg")

def register_exp_reg(exp):
    return head(tail(exp))
def constant(value):
    return llist("constant", value)

def is_constant_exp(exp):
    return is_tagged_list(exp, "constant")

def constant_exp_value(exp):
    return head(tail(exp))
def label(name):
    return llist("label", name)

def is_label_exp(exp):
    return is_tagged_list(exp, "label")

def label_exp_label(exp):
    return head(tail(exp))
```)

قد تتضمن التعليمات #py("assign") و #py("perform") و #py("test") تطبيق عملية آلة (محددة بواسطة تعبير #idx("op (in register machine)", sub: "simulating") #py("op")) على بعض المعاملات (المحددة بواسطة تعبيرات #py("reg") و #py("constant")). تُنتج الدالة التالية دالة تنفيذ لـ "تعبير عملية"—قائمة تحتوي على تعبيرات العملية والمعاملات من التعليمة:
#idx("makeoperationexpef", decl: true)
#snippet(```python
def make_operation_exp_ef(exp, machine, labels, operations):
    op = lookup_prim(operation_exp_op(exp), operations)
    afuns = map(lambda e: make_primitive_exp_ef(e, machine, labels),
                operation_exp_operands(exp))
    return lambda: apply_in_underlying_python(op, map(lambda f: f(), afuns))
```)

يتم تحديد بناء تعبيرات العمليات بواسطة:
#idx("op (in register machine)", sub: "simulating", decl: true)#idx("register-machine language", sub: "op", decl: true)#idx("isoperationexp", decl: true)#idx("operationexpop", decl: true)#idx("operationexpoperands", decl: true)
#snippet(```python
def op(name):
    return llist("op", name)

def is_operation_exp(exp):
    return is_pair(exp) and is_tagged_list(head(exp), "op")

def operation_exp_op(op_exp):
    return head(tail(head(op_exp)))

def operation_exp_operands(op_exp):
    return tail(op_exp)
```)

لاحظ أن التعامل مع تعبيرات العمليات يشبه إلى حد كبير التعامل مع تطبيقات الدوال بواسطة الدالة #py("analyze_application") في مُقيِّم القسم @sec:separating-analysis من حيث أننا نولد دالة تنفيذ لكل معامل.
في وقت المحاكاة، نستدعي دوال المعاملات ونطبق دالة #en[Python] التي تحاكي العملية على القيم الناتجة.

نحن نستخدم الدالة #py("apply_in_underlying_python")، كما فعلنا في #py("apply_primitive_function") في القسم @sec:running-eval. هذا مطلوب لتطبيق #py("op") على جميع عناصر قائمة الوسائط #py("afuns") الناتجة عن الـ #py("map") الأولى، كما لو كانت وسائط منفصلة لـ #py("op"). بدون هذا، ستكون #py("op") مقيدة بأن تكون دالة أحادية.

يتم العثور على دالة المحاكاة من خلال البحث عن اسم العملية في جدول العمليات الخاص بالآلة:
#idx("lookupprim", decl: true)
#snippet(```python
def lookup_prim(symbol, operations):
    val = assoc(symbol, operations)
    return (error("unknown operation -- assemble", symbol) if is_undefined(val)
            else head(tail(val)))
```)

#exercise(label-name: <ex:5_9>, [
التعامل مع عمليات الآلة أعلاه يسمح لها بالعمل على التسميات بالإضافة إلى الثوابت ومحتويات المسجّلات. عدل دوال معالجة التعبيرات لتفرض شرط أن العمليات لا يمكن استخدامها إلا مع المسجّلات والثوابت.
])

#exercise(label-name: <ex:stack-behavior>, [
عندما قدمنا #idx("save (in register machine)") #py("save") و #idx("restore (in register machine)") #py("restore") في القسم @sec:stack-recursion، لم نحدد ما سيحدث إذا حاولت استعادة مسجّل لم يكن آخر مسجّل تم حفظه، كما في التسلسل:

#snippet(```python
save(y)
save(x)
restore(y)
```)

هناك عدة إمكانيات معقولة لمفهوم #py("restore"):

+ تضع #py("restore(y)") في #py("y") آخر قيمة تم حفظها على المكدس، بغض النظر عن المسجّل الذي جاءت منه تلك القيمة. هذه هي الطريقة التي يتصرف بها محاكينا. أظهر كيفية الاستفادة من هذا السلوك لإلغاء تعليمة واحدة من آلة فيبوناتشي في القسم @sec:stack-recursion (الشكل @fig:fib-machine).
+ تضع #py("restore(y)") في #py("y") آخر قيمة تم حفظها على المكدس، ولكن فقط إذا تم حفظ تلك القيمة من #py("y")؛ بخلاف ذلك يصدر خطأ. عدل المحاكي ليتصرف بهذه الطريقة. سيتعين عليك تغيير #py("save") لوضع اسم المسجّل على المكدس جنبًا إلى جنب مع القيمة.
+ تضع #py("restore(y)") في #py("y") آخر قيمة تم حفظها من #py("y") بغض النظر عن المسجّلات الأخرى التي تم حفظها بعد #py("y") ولم تُسترد. عدل المحاكي ليتصرف بهذه الطريقة. سيتعين عليك ربط مكدس منفصل بكل مسجّل. يجب أن تجعل عملية #py("initialize_stack") تقوم بتهيئة جميع مكدسات المسجّلات.
])

#exercise(label-name: <ex:simulated-data-paths>, [
يمكن استخدام المحاكي للمساعدة في تحديد مسارات البيانات المطلوبة لتنفيذ آلة مع وحدة تحكم معينة. قم بتوسيع المُجَمِّع لتخزين المعلومات التالية في نموذج الآلة:

- قائمة بجميع التعليمات، مع إزالة التكرارات، مرتبة حسب نوع التعليمة (#py("assign")، و #py("go_to")، وهكذا)؛
- قائمة (بدون تكرارات) بالمسجّلات المستخدمة لحفظ نقاط الإدخال (هذه هي المسجّلات المشار إليها بواسطة تعليمات #py("go_to"))؛
- قائمة (بدون تكرارات) بالمسجّلات التي يتم حفظها أو استعادتها بواسطة #py("save") أو #py("restore")؛
- لكل مسجّل، قائمة (بدون تكرارات) بالمصادر التي تم الإسناد منها (على سبيل المثال، مصادر المسجّل #py("val") في آلة المضروب بالشكل @fig:fact-machine هي #py("constant(1)") و #py("llist(op(\"*\"), reg(\"n\"), reg(\"val\"))")).

قم بتوسيع واجهة تمرير الرسائل للآلة لتوفير الوصول إلى هذه المعلومات الجديدة. لاختبار المحلل الخاص بك، عرّف آلة فيبوناتشي من الشكل @fig:fib-machine وافحص القوائم التي قمت ببنائها.
])

#exercise(label-name: <ex:5_12>, [
عدل المحاكي بحيث يستخدم تسلسل وحدة التحكم لتحديد المسجّلات التي تمتلكها الآلة بدلاً من اشتراط قائمة المسجّلات كوسيط لـ #py("make_machine").
بدلاً من التخصيص المسبق للمسجّلات في #py("make_machine")، يمكنك تخصيصها واحدة تلو الأخرى عند رؤيتها لأول مرة أثناء تجميع التعليمات.
])

#idx("register-machine language", sub: "instructions")
#idx("execution function", sub: "in register-machine simulator")
