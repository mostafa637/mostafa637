// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([تطبيق الدوال البسيطة], label-name: <sec:env-apply-proc>)

#idx("environment model of evaluation", sub: "function-application example")
#idx("function application", sub: "environment model of")

#idx("sumofsquares", sub: "in environment model")
عندما قدمنا نموذج الاستبدال في القسم @sec:substitution-model أوضحنا كيف يُقَيَّم التطبيق #py("f(5)") إلى 136، بفرض تعاريف الدوال التالية:

#snippet(```python
def square(x):
    return x * x

def sum_of_squares(x, y):
    return square(x) + square(y)

def f(a):
    return sum_of_squares(a + 1, a * 2)
```)

يمكننا تحليل المثال نفسه باستخدام نموذج البيئة.
يُظهر الشكل @fig:sum-squares كائنات الدوال الثلاثة المنشأة بتقييم تعاريف #py("f") و #py("square") و #py("sum_of_squares") في بيئة البرنامج. وكل كائن دالة يتكون من بعض الشفرات، إلى جانب مؤشر إلى بيئة البرنامج.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-5.svg", width: 70%), caption: [كائنات الدوال في إطار البرنامج.], label-name: <fig:sum-squares>)

وفي الشكل @fig:f5-eval نرى بنية البيئة المنشأة بتقييم التعبير #py("f(5)").
ينشئ الاستدعاء لـ #py("f") بيئة جديدة، E1، تبدأ بإطار يُربط فيه #py("a")، معلمة #py("f")، بالوسيط 5. وفي E1، نقوم بتقييم جسم #py("f"):

#snippet(```python
return sum_of_squares(a + 1, a * 2)
```)

ولتقييم تعليمة الإرجاع، نقوم أولاً بتقييم التعبيرات الفرعية لتعبير الإرجاع.
التعبير الفرعي الأول، #py("sum_of_squares")، له قيمة هي كائن دالة. (لاحظ كيف تم العثور على هذه القيمة: ننظر أولاً في الإطار الأول لـ E1، والذي لا يحتوي على رابط لـ #py("sum_of_squares"). ثم ننتقل إلى البيئة المحيطة، أي بيئة البرنامج، ونجد الرابط الموضح في الشكل @fig:sum-squares.)
أما التعبيران الفرعيان الآخران فيتم تقييمهما بتطبيق العمليات الأولية #py("+") و #py("*") لتقييم التركيبتين #py("a + 1") و #py("a * 2") للحصول على 6 و 10 على التوالي.

والآن نطبق كائن الدالة #py("sum_of_squares") على الوسيطين 6 و 10. ينتج عن هذا بيئة جديدة، E2، يُربط فيها المعلمان #py("x") و #py("y") بالوسيطين. وداخل E2 نقوم بتقييم التعليمة #snippet(```python return square(x) + square(y) ```)
يقودنا هذا إلى تقييم #py("square(x)")، حيث يُعثر على #py("square") في إطار البرنامج و #py("x") تساوي 6. ومرة أخرى، نُنشئ بيئة جديدة، E3، يُربط فيها #py("x") بـ 6، وداخلها نقوم بتقييم جسم #py("square")، وهو #py("return x * x").
وأيضاً كجزء من تطبيق #py("sum_of_squares")، يجب أن نقيم التعبير الفرعي #py("square(y)")، حيث #py("y") تساوي 10. ينشئ هذا الاستدعاء الثاني لـ #py("square") بيئة أخرى، E4، يُربط فيها #py("x")، معلمة #py("square")، بـ 10. وداخل E4 يجب أن نقيم #py("return x * x").

والنقطة المهمة التي يجب ملاحظتها هي أن كل استدعاء لـ #py("square") ينشئ بيئة جديدة تحتوي على رابط لـ #py("x"). ونستطيع أن نرى هنا كيف تعمل الأطر المختلفة على إبقاء المتغيرات المحلية المختلفة المسماة جميعها بـ #py("x") منفصلة. لاحظ أن كل إطار ينشئه #py("square") يشير إلى بيئة البرنامج، حيث إن هذه هي البيئة المشار إليها بوساطة كائن الدالة #py("square").

بعد تقييم التعبيرات الفرعية، تُرجع النتائج. وتُجمع القيم الناتجة عن الاستدعائين لـ #py("square") بواسطة #py("sum_of_squares")، وتُرجع هذه النتيجة بوساطة #py("f").
وبما أن تركيزنا هنا ينصب على بنيات البيئة، فلن نطيل الحديث في كيفية تمرير هذه القيم المُرجعة من استدعاء إلى آخر؛ ومع ذلك، يُعد هذا جانباً مهماً من عملية التقييم، وسنعود إليه بالتفصيل في الفصل @chap:reg.
#idx("sumofsquares", sub: "in environment model")

#exercise(label-name: <ex:factorial-env-model>, [
في القسم @sec:recursion-and-iteration استخدمنا نموذج الاستبدال لتحليل دالتين
#idx("recursive process", sub: "iterative process vs.")
#idx("iterative process", sub: "recursive process vs.")
لحساب
#idx("factorial", sub: "environment structure in evaluating")
العاملي، نسخة تكرارية عودية

#snippet(```python
def factorial(n):
    return (1
            if n == 1
            else n * factorial(n - 1))
```)

ونسخة تكرارية محددة

#snippet(```python
def factorial(n):
    return fact_iter(1, 1, n)

def fact_iter(product, counter, max_count):
    return (product
            if counter > max_count
            else fact_iter(counter * product,
                           counter + 1,
                           max_count))
```)

وضّح بنيات البيئة المنشأة بتقييم #py("factorial(6)") باستخدام كل نسخة من دالة #py("factorial").#footnote[لن يوضح نموذج البيئة ادعاءنا في القسم @sec:recursion-and-iteration بأن المفسر يمكنه تنفيذ دالة مثل #py("fact_iter") بمساحة ثابتة باستخدام العودية الذيلية. وسوف نناقش
#idx("environment model of evaluation", sub: "tail recursion and")
#idx("tail recursion", sub: "environment model of evaluation and")
العودية الذيلية عندما نتعامل مع بنية التحكم للمفسر في القسم @sec:eceval.]
])

#idx("environment model of evaluation", sub: "function-application example")
#idx("function application", sub: "environment model of")

#sicp-figure(image("/images/img_javascript/ch3-Z-G-6.svg", width: 70%), caption: [البيئات المنشأة بتقييم #py("f(5)") باستخدام الدوال الموضحة في الشكل @fig:sum-squares.], label-name: <fig:f5-eval>)
