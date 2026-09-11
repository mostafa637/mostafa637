// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([التجريد في تصميم الآلات])

#idx("abstraction", sub: "in register-machine design")

غالبًا ما نحدد آلة لتشمل عمليات "أولية" تكون في الواقع معقدة جدًا. على سبيل المثال، في القسمين @sec:eceval و @sec:compilation سنتعامل مع معالجات البيئة الخاصة ببايثون كعمليات أولية. مثل هذا التجريد (abstraction) ذو قيمة لأنه يسمح لنا بتجاهل تفاصيل أجزاء من الآلة حتى نتمكن من التركيز على جوانب أخرى من التصميم. ومع ذلك، فإن حقيقة أننا أخفينا الكثير من التعقيد لا تعني أن تصميم الآلة غير واقعي. يمكننا دائمًا استبدال الـ "أوليات" المعقدة بعمليات أولية أبسط.

ضع في اعتبارك آلة GCD. تحتوي الآلة على تعليمة تحسب باقي قسمة محتويات المسجّلين #py("a") و #py("b") وتسند النتيجة إلى المسجّل #py("t"). إذا أردنا بناء آلة GCD دون استخدام عملية باقي أولية، فيجب علينا تحديد كيفية حساب البواقي بدلالة عمليات أبسط، مثل الطرح.
في الواقع، يمكننا كتابة دالة بايثون تجد البواقي بهذه الطريقة:

#snippet(```python
def remainder(n, d):
    return n if n < d else remainder(n - d, d)
```)

وبالتالي يمكننا استبدال عملية الباقي في مسارات بيانات آلة GCD بعملية طرح واختبار مقارنة.
يوضح الشكل @fig:gcd-machine-rem مسارات البيانات ووحدة التحكم للآلة المُفصلة. التعليمة

#sicp-figure(image("/images/img_original/Fig5.5b.std.svg", width: 70%), caption: [مسارات البيانات ووحدة التحكم لآلة GCD المُفصلة.], label-name: <fig:gcd-machine-rem>)

#snippet(```python
assign("t", llist(op("rem"), reg("a"), reg("b")))
```)

في تعريف وحدة تحكم GCD تُستبدل بتسلسل من التعليمات يحتوي على حلقة (loop)، كما هو موضح في الشكل @fig:gcd-machine-rem-controller.

#sicp-figure([#snippet(```python
controller(
  llist(
    "test_b",
      test(llist(op("="), reg("b"), constant(0))),
      branch(label("gcd_done")),
      assign("t", reg("a")),
    "rem_loop",
      test(llist(op("<"), reg("t"), reg("b"))),
      branch(label("rem_done")),
      assign("t", llist(op("-"), reg("t"), reg("b"))),
      go_to(label("rem_loop")),
    "rem_done",
      assign("a", reg("b")),
      assign("b", reg("t")),
      go_to(label("test_b")),
    "gcd_done"))
```)], caption: [تسلسل تعليمات وحدة التحكم لآلة GCD في الشكل @fig:gcd-machine-rem.], label-name: <fig:gcd-machine-rem-controller>)

#exercise(label-name: <ex:sqrt-machine>, [
صمم آلة لحساب #idx("sqrt", sub: "register machine for") الجذور التربيعية باستخدام طريقة نيوتن، كما هو موضح في القسم @sec:sqrt ومنفذ بواسطة الكود التالي في القسم @sec:block-structure:

#snippet(```python
def sqrt(x):
    def is_good_enough(guess):
        return math_abs(square(guess) - x) < 0.001
    def improve(guess):
        return average(guess, x / guess)
    def sqrt_iter(guess):
        return guess if is_good_enough(guess) else sqrt_iter(improve(guess))
    return sqrt_iter(1)
```)

ابدأ بافتراض أن العمليتين #py("is_good_enough") و #py("improve") متوفرتان كعمليات أولية. ثم وضح كيف يمكن توسيعها بدلالة العمليات الحسابية. صِف كل إصدار من تصميم آلة #py("sqrt") عن طريق رسم مخطط مسار بيانات وكتابة تعريف وحدة التحكم في لغة آلة المسجّلات.
])

#idx("abstraction", sub: "in register-machine design")
