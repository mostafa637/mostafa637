// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([مثال: الجذور التربيعية بطريقة نيوتن], label-name: <sec:sqrt>)

#idx("function (mathematical)", sub: "Python function vs.")
الدوال، كما قُدّمت أعلاه، تشبه كثيراً الدوال الرياضية العادية. فهي تحدّد قيمة
يحدّدها معامل واحد أو أكثر. لكن هناك فرق مهم بين الدوال الرياضية ودوال الحاسوب.
يجب أن تكون دوال الحاسوب فعّالة.

كحالة توضيحية، تأمّل مشكلة حساب الجذور التربيعية. يمكننا تعريف دالة الجذر
التربيعي كالتالي:

$ sqrt(x) space =upright(" the ")y u p r i g h t(" such that ")y gt.eq 0upright(" and ") y^(2) space = space x $

يصف هذا دالة رياضية مشروعة تماماً. يمكننا استخدامها للتعرّف على ما إذا كان عدد
هو الجذر التربيعي لعدد آخر، أو لاستنتاج حقائق عن الجذور التربيعية بشكل عام.
لكن من ناحية أخرى، لا يصف التعريف دالة حاسوب. بل لا يخبرنا بأي شيء تقريباً عن
كيفية إيجاد الجذر التربيعي لعدد معطى فعلاً. لن يفيد إعادة صياغة هذا التعريف
بـ #en[Python] زائفة:

#syntax("\
def sqrt(x):
    return the y ", $mono("with")$, " y >= 0 and square(y) == x
      ")

هذا يطرح السؤال فحسب.

التناقض بين الدالة الرياضية ودالة الحاسوب هو انعكاس للتمييز العام بين وصف خصائص
الأشياء ووصف كيفية فعل الأشياء، أو كما يُشار إليه أحياناً، التمييز بين
#idx("declarative vs. imperative knowledge")
#idx("imperative vs. declarative knowledge")
المعرفة التقريرية والمعرفة الأمرية. في
#idx("mathematics", sub: "computer science vs.")
#idx("computer science", sub: "mathematics vs.")
الرياضيات نهتم عادة بالأوصاف التقريرية (ما هو)، بينما في علوم الحاسوب نهتم عادة
بالأوصاف الأمرية (كيف يُفعل).#footnote[الأوصاف التقريرية والأمرية مترابطة بشكل
وثيق، كما أنّ الرياضيات وعلوم الحاسوب كذلك. مثلاً، القول بأنّ الإجابة التي
ينتجها برنامج
#idx("correctness of a program")
«صحيحة» هو تصريح تقريري عن البرنامج. هناك قدر كبير من البحث يهدف إلى ترسيخ
تقنيات
#idx("proving programs correct")
لإثبات صحة البرامج، وكثير من الصعوبة التقنية في هذا الموضوع تتعلق بالتفاوض
في الانتقال بين التصريحات الأمرية (التي تُبنى منها البرامج) والتصريحات التقريرية
(التي يمكن استخدامها لاستنتاج الأشياء). وفي سياق مماثل، استكشف مصممو لغات
البرمجة ما يُسمّى #idx("programming language", sub: "very high-level") #idx("very high-level language") لغات عالية المستوى جداً، حيث يبرمج المرء فعلياً بتصريحات تقريرية. الفكرة هي جعل
المُفسِّرات متطوّرة بما يكفي لتتمكّن، بإعطائها معرفة «ما هو» المحدّدة من
المبرمج، من توليد معرفة «كيف يُفعل» تلقائياً. لا يمكن فعل هذا بشكل عام، لكن
هناك مجالات مهمة أُحرز فيها تقدّم. سنعيد زيارة هذه الفكرة في الفصل @chap:meta.]
#idx("function (mathematical)", sub: "Python function vs.")

كيف يحسب المرء
#idx("square root")
#idx("Newton's method", sub: "for square roots")
الجذور التربيعية؟ الطريقة الأكثر شيوعاً هي استخدام طريقة نيوتن للتقريبات
المتتالية، التي تقول أنه كلما كان لدينا تخمين $y$ لقيمة الجذر التربيعي لعدد
$x$، يمكننا إجراء معالجة بسيطة للحصول على تخمين أفضل (أقرب للجذر التربيعي
الفعلي) بحساب متوسط $y$ و $x\/y$.#footnote[خوارزمية الجذر التربيعي هذه هي في
الواقع حالة خاصة من طريقة نيوتن، وهي تقنية عامة لإيجاد جذور المعادلات.
طوّر الخوارزمية ذاتها هيرون
#idx("Heron of Alexandria")
الإسكندراني في القرن الأول الميلادي. سنرى كيف نعبّر عن طريقة نيوتن العامة كدالة
#en[Python] في القسم @sec:proc-returned-values.]
مثلاً، يمكننا حساب الجذر التربيعي لـ 2 كالتالي. لنفترض أنّ تخميننا الأولي هو 1:

$ mat(delim: #none, upright("التخمين"), upright("الخارج"), upright("المتوسط"); 1, ( frac(2, 1) = 2), ( frac((2+1), 2) = 1.5); 1.5, ( frac(2, 1.5) = 1.3333), ( frac((1.3333+1.5), 2) = 1.4167); 1.4167, ( frac(2, 1.4167) = 1.4118), ( frac((1.4167+1.4118), 2) = 1.4142); 1.4142, dots.h, dots.h) $

بمواصلة هذه العملية نحصل على تقريبات أفضل فأفضل للجذر التربيعي.

الآن لنُشكلن العملية بدوال. نبدأ بقيمة
#idx("radicand")
للمجذور (العدد الذي نحاول حساب جذره التربيعي) وقيمة للتخمين. إذا كان التخمين
جيداً كفاية لأغراضنا فقد انتهينا؛ وإلا يجب تكرار العملية بتخمين محسّن.
نكتب هذه الاستراتيجية الأساسية كدالة:

#snippet(```python
def sqrt_iter(guess, x):
    return (guess
            if is_good_enough(guess, x)
            else sqrt_iter(improve(guess, x), x))
```)

يُحسَّن التخمين بحساب متوسطه مع خارج قسمة المجذور على التخمين القديم:

#snippet(```python
def improve(guess, x):
    return average(guess, x / guess)
```)

حيث
#idx("average", decl: true)
#snippet(```python
def average(x, y):
    return (x + y) / 2
```)

يجب أيضاً تحديد ما نعنيه بـ «جيد كفاية». ما يلي يصلح للتوضيح لكنه ليس اختباراً
جيداً فعلاً (انظر التمرين @ex:ex-sqrt-end-test). الفكرة هي تحسين الإجابة حتى
تكون قريبة بما يكفي بحيث يختلف مربعها عن المجذور بأقل من تسامح محدّد مسبقاً
(هنا 0.001):#footnote[سنعطي عادة
#idx("predicate", sub: "naming convention for")
#idx("naming conventions", sub: "isfor predicates")
#idx("is, in predicate names", sort: "is")
أسماء المحمولات بادئة #py("is_") لمساعدتنا على تذكّر أنها محمولات.]

#snippet(```python
def is_good_enough(guess, x):
    return abs(square(guess) - x) < 0.001
```)

أخيراً، نحتاج طريقة للبدء. مثلاً، يمكننا دائماً تخمين أنّ الجذر التربيعي لأي
عدد هو 1:

#idx("sqrt", decl: true)
#snippet(```python
def sqrt(x):
    return sqrt_iter(1, x)
```)

إذا كتبنا هذه التعريفات في المُفسِّر، يمكننا استخدام #py("sqrt") كأي دالة:

#snippet(```python
print(sqrt(9))
```)

#output(```python
print(sqrt(9))
```)

#snippet(```python
print(sqrt(100 + 37))
```)

#output(```python
print(sqrt(100 + 37))
```)

#snippet(```python
print(sqrt(sqrt(2) + sqrt(3)))
```)

#output(```python
print(sqrt(sqrt(2) + sqrt(3)))
```)

#idx("square root")#idx("Newton's method", sub: "for square roots")
#snippet(```python
print(square(sqrt(1000)))
```)

#output(```python
print(square(sqrt(1000)))
```)

يوضّح برنامج #py("sqrt") أيضاً أنّ اللغة
#idx("iterative process", sub: "implemented by function call")
الدالّية البسيطة التي قدّمناها حتى الآن كافية لكتابة أي برنامج عددي بحت يمكن
كتابته بلغة #en[C] أو #en[Pascal] مثلاً. قد يبدو هذا مفاجئاً لأننا لم نضمّن في
لغتنا أي بنى
#idx("looping constructs")
تكرارية (حلقات) توجّه الحاسوب لفعل شيء مراراً. لكن الدالة #py("sqrt_iter")
توضّح كيف يمكن إنجاز التكرار دون أي بنية خاصة سوى القدرة العادية على استدعاء
دالة.#footnote[القرّاء القلقون بشأن قضايا الكفاءة المتعلقة باستخدام استدعاءات
الدوال لتنفيذ التكرار يجب أن يلاحظوا الملاحظات عن «العودية الذيلية» في
القسم @sec:recursion-and-iteration.]
#idx("iterative process", sub: "implemented by function call")

#exercise([
لا ترى #en[Alyssa P. Hacker] لماذا يجب أن يكون #py("if")
#idx("syntactic form", sub: "need for")
#idx("conditional expression", sub: "why a syntactic form")
شكلاً صياغياً. «لماذا لا أستطيع تعريفه كدالة شرطية عادية يعمل تطبيقها تماماً
كالتعبيرات الشرطية؟» تسأل.#footnote[بصفتها مخترقة #en[Lisp] من كتاب #emph[بنية
وتفسير برامج الحاسوب] الأصلي، تفضّل #en[Alyssa] صياغة أبسط وأكثر انتظاماً.]
تدّعي صديقتها #en[Eva Lu Ator] أنّ هذا ممكن فعلاً، وتُعرّف دالة
#py("conditional") كالتالي:

#snippet(```python
def conditional(predicate, then_clause, else_clause):
    return then_clause if predicate else else_clause
```)

تُوضّح #en[Eva] البرنامج لـ #en[Alyssa]:

#snippet(```python
print(conditional(2 == 3, 0, 5))
```)

#output(```python
print(conditional(2 == 3, 0, 5))
```)

#snippet(```python
print(conditional(1 == 1, 0, 5))
```)

#output(```python
print(conditional(1 == 1, 0, 5))
```)

سعيدة بالنتيجة، تستخدم #en[Alyssa] #py("conditional") لإعادة كتابة برنامج
الجذر التربيعي:

#snippet(```python
def sqrt_iter(guess, x):
    return conditional(is_good_enough(guess, x),
                       guess,
                       sqrt_iter(improve(guess, x),
                                 x))
```)

ماذا يحدث حين تحاول #en[Alyssa] استخدام هذا لحساب الجذور التربيعية؟ اشرح.
#anchor(<ex:new-if>)
])

#exercise(label-name: <ex:ex-sqrt-end-test>, [
اختبار #py("is_good_enough") المستخدم في حساب الجذور التربيعية لن يكون فعّالاً
جداً لإيجاد الجذور التربيعية للأعداد الصغيرة جداً. كذلك في الحواسيب الحقيقية
تُجرى العمليات الحسابية دائماً تقريباً بدقة محدودة. هذا يجعل اختبارنا غير كافٍ
للأعداد الكبيرة جداً. اشرح هذه التصريحات مع أمثلة تُظهر كيف يفشل الاختبار
للأعداد الصغيرة والكبيرة. استراتيجية بديلة لتنفيذ #py("is_good_enough") هي
مراقبة كيف يتغيّر #py("guess") من تكرار لآخر والتوقف حين يكون التغيير جزءاً
صغيراً جداً من التخمين. صمّم دالة جذر تربيعي تستخدم هذا النوع من اختبار الإنهاء.
هل تعمل بشكل أفضل للأعداد الصغيرة والكبيرة؟
])

#exercise(label-name: <ex:cube-root-newton>, [
تعتمد طريقة نيوتن
#idx("cube root", sub: "by Newton's method")
#idx("Newton's method", sub: "for cube roots")
للجذور التكعيبية على حقيقة أنه إذا كان $y$ تقريباً للجذر التكعيبي لـ $x$، فإنّ
تقريباً أفضل يُعطى بالقيمة

$ frac(x/y^(2)+2y, 3) $

استخدم هذه الصيغة لتنفيذ دالة جذر تكعيبي مماثلة لدالة الجذر التربيعي.
(في القسم @sec:proc-returned-values سنرى كيف ننفّذ طريقة نيوتن بشكل عام
كتجريد لدالتي الجذر التربيعي والتكعيبي هاتين.)
])
