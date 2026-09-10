// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#section([محاكي آلة المسجّلات], label-name: <sec:simulator>)

#idx("register machine", sub: "simulator")
#idx("register-machine simulator")
#idx("simulation", sub: "of register machine")

من أجل اكتساب فهم جيد لتصميم آلات المسجّلات، يجب علينا اختبار الآلات التي نصممها لمعرفة ما إذا كانت تعمل كما هو متوقع. إحدى الطرق لاختبار التصميم هي المحاكاة اليدوية لتشغيل وحدة التحكم، كما في التمرين @ex:hand-sim. لكن هذا أمر مرهق للغاية بالنسبة للآلات كافة باستثناء أبسطها. في هذا القسم نقوم ببناء محاكٍ للآلات الموصوفة بلغة آلة المسجّلات. المحاكي هو برنامج #en[Python] يحتوي على أربع دوال واجهة:
الأولى تستخدم وصفًا لآلة مسجّلات لبناء نموذج للآلة (بنية بيانات تتوافق أجزاؤها مع أجزاء الآلة المراد محاكاتها)، والدوال الثلاث الأخرى تسمح لنا بمحاكاة الآلة من خلال التعامل مع النموذج:

- #py("make_machine(")#meta("register-names")#py(",")#meta("operations")#py(",")#meta("controller")#py(")") #idx("makemachine") \ يبني ويُرجع نموذجًا للآلة مع المسجّلات والعمليات ووحدة التحكم المحددة.
- #py("set_register_contents(")#meta("machine-model")#py(",")#meta("register-name")#py(",")#meta("value")#py(")") #idx("setregistercontents") \ يخزن قيمة في مسجّل محاكى في الآلة المعطاة.
- #py("get_register_contents(")#meta("machine-model")#py(",")#meta("register-name")#py(")") #idx("getregistercontents") \ يُرجع محتويات مسجّل محاكى في الآلة المعطاة.
- #py("start(")#meta("machine-model")#py(")") #idx("start register machine") \ يحاكي تنفيذ الآلة المعطاة، بدءًا من بداية تسلسل وحدة التحكم والتوقف عند الوصول إلى نهاية التسلسل.

كمثال على كيفية استخدام هذه الدوال، يمكننا تعريف #py("gcd_machine") ليكون نموذجًا لآلة GCD الخاصة بالقسم @sec:register-machine-language كما يلي:

#idx("gcd", sub: "register machine for")#idx("gcdmachine", decl: true)
#snippet(```python
gcd_machine = make_machine(
    llist("a", "b", "t"),
    llist(llist("rem", lambda a, b: a % b),
         llist("=", lambda a, b: a == b)),
    llist("test_b",
           test(llist(op("="), reg("b"), constant(0))),
           branch(label("gcd_done")),
           assign("t", llist(op("rem"), reg("a"), reg("b"))),
           assign("a", reg("b")),
           assign("b", reg("t")),
           go_to(label("test_b")),
         "gcd_done"))
```)

الوسيط الأول لـ #py("make_machine") هو قائمة بأبناء المسجّلات. الوسيط التالي هو جدول (قائمة من قوائم ذات عنصرين) يقرن كل اسم عملية بدالة #en[Python] تنفذ العملية (أي تنتج نفس قيمة الإخراج عند إعطائها نفس قيم الإدخال). الوسيط الأخير يحدد وحدة التحكم كقائمة من التسميات وتعليمات الآلة، كما في القسم @sec:designing-register-machines.

لحساب القواسم المشتركة الكبرى (GCDs) باستخدام هذه الآلة، نحدد مسجّلات الإدخال، ونبدأ الآلة، ونفحص النتيجة عند إنهاء المحاكاة:

#snippet(```python
set_register_contents(gcd_machine, "a", 206)
```)

#output(```python
set_register_contents(gcd_machine, "a", 206)
```)

#snippet(```python
set_register_contents(gcd_machine, "b", 40)
```)

#output(```python
set_register_contents(gcd_machine, "b", 40)
```)

#snippet(```python
start(gcd_machine)
```)

#output(```python
start(gcd_machine)
```)

#snippet(```python
get_register_contents(gcd_machine, "a")
```)

#output(```python
get_register_contents(gcd_machine, "a")
```)

سيعمل هذا الحساب بشكل أبطأ بكثير من دالة #py("gcd") المكتوبة بلغة #en[Python]، لأننا سنحاكي تعليمات الآلة منخفضة المستوى، مثل #py("assign")، بواسطة عمليات أكثر تعقيدًا بكثير.

#exercise(label-name: <ex:use-simulator>, [
استخدم المحاكي لاختبار الآلات التي صممتها في التمرين @ex:design-reg-machines.
])

#include "../../chapter5/section2/subsection1-ar.typ"

#include "../../chapter5/section2/subsection2-ar.typ"

#include "../../chapter5/section2/subsection3-ar.typ"

#include "../../chapter5/section2/subsection4-ar.typ"
