// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([العَوْدِيَّة والتكرار الخطّيّان], label-name: <sec:recursion-and-iteration>)

#idx("iterative process", sub: "recursive process vs.")
#idx("recursive process", sub: "iterative process vs.")

نبدأ بالنظر في
#idx("factorial")
دالة المضروب، المحدَّدة بـ:

$ mat(delim: #none, n!, =, n dot.op (n-1) dot.op (n-2) dots.c 3 dot.op 2 dot.op 1) $

هناك طرق عديدة لحساب المضروب. إحدى الطرق هي الاستفادة من
ملاحظة أنّ $n!$ يساوي
$n$ مضروباً في $(n-1)!$ لأي
عدد صحيح موجب $n$:

$ mat(delim: #none, n!, =, n dot.op lr([ (n-1) dot.op (n-2) dots.c 3 dot.op 2 dot.op 1 ]) = n dot.op (n-1)!) $

وبذلك، يمكننا حساب $n!$ بحساب
$(n-1)!$ وضرب الناتج في $n$. وإذا أضفنا الشرط بأنّ 1!
يساوي 1،
تُترجَم هذه الملاحظة مباشرة إلى
دالة حاسوبية:
#idx("factorial", sub: "linear recursive version", decl: true)
#snippet(```python
def factorial(n):
    return 1 if n == 1 else n * factorial(n - 1)
```)

يمكننا استخدام نموذج الاستبدال من
القسم @sec:substitution-model لمشاهدة هذه
#idx("substitution model of function application", sub: "shape of process")
الدالة أثناء عملها لحساب !6، كما هو موضح في
الشكل @fig:recursive-factorial-javascript.

والآن لنأخذ منظوراً مختلفاً لحساب المضروب. يمكننا وصف قاعدة لحساب $n!$
من خلال تحديد أننا نضرب أولاً 1 في 2، ثم نضرب الناتج في 3،
ثم في 4، وهكذا حتى نصل إلى $n$.
وبشكل أكثر رسمية، نحافظ على حاصل ضرب تراكمي، جنباً إلى جنب مع عدّاد
يعُدّ من 1 حتى $n$. ويمكننا وصف الحساب بالقول إنّ العدّاد وحاصل الضرب
يتغيران بالتزامن من خطوة إلى الخطوة التالية وفقاً للقاعدة:

$ mat(delim: #none,
upright("product"), arrow.l, upright("counter") dot.op upright("product");
upright("counter"), arrow.l, upright("counter") + 1) $

ومع اشتراط أنّ $n!$ هو قيمة
حاصل الضرب عندما يتجاوز العدّاد $n$.

ومرة أخرى، يمكننا إعادة صياغة وصفنا كـ
دالة
لحساب المضروب:#footnote[في برنامج حقيقي، سنستخدم على الأرجح البنية المكتفية ذاتياً (الكتلية) المقتَرحة في القسم السابق لإخفاء تعريف #py("fact_iter"):

#snippet(```python
def factorial(n):
    def iterate(product, counter):
        return (product
                if counter > n
                else iterate(counter * product, counter + 1))
    return iterate(1, 1)
```)

تجنّبنا القيام بذلك هنا لتقليل عدد الأشياء التي يجب التفكير فيها دفعة واحدة.]<foot:block-structured-factorial>
#idx("factorial", sub: "linear iterative version", decl: true)
#snippet(```python
def factorial(n):
    return fact_iter(1, 1, n)

def fact_iter(product, counter, max_count):
    return (product
            if counter > max_count
            else fact_iter(counter * product, counter + 1, max_count))
```)

وكما سبق، يمكننا استخدام نموذج الاستبدال لتصور العملية

#sicp-figure(image("/images/img_javascript/ch1-Z-G-7.svg", width: 70%), caption: [عملية عَوْدِيَّة خطية لحساب !6.], label-name: <fig:recursive-factorial-javascript>)

#sicp-figure(image("/images/img_javascript/ch1-Z-G-10.svg", width: 70%), caption: [عملية تكرارية خطية لحساب $6!$.], label-name: <fig:iterative-factorial-javascript>)

لحساب $6!$، كما هو موضح في
الشكل @fig:iterative-factorial-javascript.

قَارِن بين العمليتين. من وجهة نظر واحدة، تبدوان غير مختلفتين تقريباً.
كلاهما تحسبان نفس الدالة الرياضية على نفس المجال، وكل منهما تتطلب عدداً من الخطوات يتناسب مع $n$ لحساب $n!$. بل إنّ كلا العمليتين تنفّذان نفس المتتالية من عمليات الضرب، وتحصلان على نفس المتتالية
من نواتج الضرب الجزئية. ومن ناحية أخرى، حين نأخذ في الاعتبار
#idx("shape of a process")
#idx("process", sub: "shape of")
«أشكال» العمليتين، نجد أنهما تتطوران بطريقتين مختلفتين تماماً.

تأمَّل العملية الأولى. يكشف نموذج الاستبدال عن شكل من التمدد يليه انكماش،
مشار إليه بالسهم في
الشكل @fig:recursive-factorial-javascript.
يحدث التمدد حين تبني العملية سلسلة من
#idx("deferred operations")
#emph[العمليات المؤجَّلة] (في هذه الحالة، سلسلة من عمليات الضرب).
ويحدث الانكماش حين تُنفَّذ العمليات بالفعل. ويُسمّى هذا النوع من العمليات،
المميَّز بسلسلة من العمليات المؤجَّلة،
#idx("recursive process")
#idx("process", sub: "recursive")
#emph[عملية عَوْدِيَّة]. يتطلب إجراء هذه العملية أن يحتفظ المُفسِّر بمعلومات
عن العمليات التي سيتعيّن أداؤها لاحقاً. وفي حساب $n!$، فإنّ طول سلسلة
الضرب المؤجَّل، وبالتالي مقدار المعلومات المطلوبة لمتابعتها،
#idx("linear growth")
ينمو خطياً مع $n$ (يتناسب مع
$n$)، تماماً كعدد الخطوات.
وتُسمّى مثل هذه العملية
#idx("recursive process", sub: "linear")
#idx("linear recursive process")
#idx("process", sub: "linear recursive")
#emph[عملية عَوْدِيَّة خطية].

وبالمقابل، لا تنمو العملية الثانية ولا تنكمش. في كل خطوة، كل ما نحتاج إلى متابعته، لأي $n$، هو القيم الحالية للأسماء
#py("product")، و#py("counter")،
و#py("max_count").
ونسمي هذه
#idx("iterative process")
#idx("process", sub: "iterative")
#emph[عملية تكرارية]. بوجه عام، العملية التكرارية هي عملية يمكن تلخيص حالتها بعدد ثابت من
#idx("state variable")
#emph[متغيرات الحالة]، جنباً إلى جنب مع قاعدة ثابتة تصف كيفية تحديث متغيرات الحالة مع انتقال العملية من حالة إلى حالة، وشرط انتهاء (اختياري) يحدد الظروف التي يجب أن تنتهي عندها العملية. وفي حساب $n!$، فإنّ عدد الخطوات المطلوبة ينمو خطياً مع $n$.
وتُسمّى مثل هذه العملية
#idx("iterative process", sub: "linear")
#idx("linear iterative process")
#idx("process", sub: "linear iterative")
#emph[عملية تكرارية خطية].

يمكن رؤية التباين بين العمليتين بطريقة أخرى.
في الحالة التكرارية، توفر متغيرات الحالة وصفاً كاملاً لحالة العملية عند أي نقطة. وإذا أوقفنا الحساب بين الخطوات، فإنّ كل ما نحتاج إلى القيام به لاستئناف الحساب هو تزويد المُفسِّر بقيم متغيرات الحالة الثلاثة. وليس الأمر كذلك مع العملية العَوْدِيَّة. في هذه الحالة، هناك بعض المعلومات «الخفية» الإضافية، يحتفظ بها المُفسِّر ولا تحتويها متغيرات الحالة، والتي تشير إلى «مكان العملية» في تنفيذ سلسلة العمليات المؤجَّلة. وكلما طالت السلسلة، وجب الاحتفاظ بمعلومات أكثر.#footnote[حين نناقش تنفيذ الدوال على آلات السجلات في الفصل @chap:reg، سنرى أنّ أي عملية تكرارية يمكن تحقيقها «في العتاد» كآلة تحتوي على مجموعة ثابتة من السجلات وبدون ذاكرة مساعدة. وفي المقابل، فإنّ تحقيق عملية عَوْدِيَّة يتطلب آلة تستخدم هيكل بيانات مساعداً يُعرَف بـ
#idx("stack")
#emph[المكدس (#en[stack])].]
#idx("substitution model of function application", sub: "shape of process")

عند المقارنة بين التكرار والعَوْدِيَّة، يجب أن نكون حريصين على عدم الخلط بين مفهوم
#idx("recursive function", sub: "recursive process vs.")
#idx("recursive process", sub: "recursive function vs.")
#emph[العملية] العَوْدِيَّة ومفهوم #emph[الدالة] العَوْدِيَّة.
حين نحدد دالة بأنها عَوْدِيَّة، فإننا نشير إلى حقيقة نحوية مفادها أنّ تعريف الدالة يشير (سواء بشكل مباشر أو غير مباشر) إلى الدالة نفسها. ولكن حين نحدد عملية بأنها تتبع نمطاً، ولنقل، عَوْدِيّاً خطياً، فإننا نتحدث عن كيفية تطور العملية، وليس عن بناء الجملة (النحو) لكتابة الدالة. قد يبدو من المربك أننا نشير إلى دالة عَوْدِيَّة مثل #py("fact_iter") بأنها تُنشئ عملية تكرارية. ومع ذلك، فإنّ العملية تكرارية حقاً: حالتها ملتقطة بالكامل بواسطة متغيرات حالتها الثلاثة، ولا يحتاج المُفسِّر إلا لمتابعة ثلاثة أسماء فقط من أجل تنفيذ العملية.

أحد الأسباب التي قد تجعل التمييز بين العملية والدالة مربكاً هو أنّ معظم تنفيذات اللغات الشائعة (بما في ذلك #idx("C", sub: "recursive functions in") #en[C]، و#idx("Java, recursive functions in") #en[Java]، و#idx("Python, recursive functions in") #en[Python])
مُصمَّمة بطريقة تجعل تفسير أي دالة عَوْدِيَّة يستهلك مقداراً من الذاكرة ينمو مع عدد استدعاءات الدالة، حتى عندما تكون العملية الموصوفة، من حيث المبدأ، تكرارية.
ونتيجة لذلك، فإنّ هذه اللغات لا يمكنها وصف العمليات التكرارية إلا باللجوء إلى «بنى تكرار» ذات غرض خاص
#idx("looping constructs")
مثل
$mono("do")$، و
$mono("repeat")$، و
$mono("until")$، و
$mono("for")$، و
$mono("while")$.
 وتنفيذ بايثون الذي سننظر فيه في الفصل @chap:reg لا يشترك في هذا العيب. فهو سينفذ العملية التكرارية في مساحة ثابتة، حتى لو وُصِفَت العملية التكرارية بواسطة دالة عَوْدِيَّة.

ويُسمّى التنفيذ الذي يتمتع بهذه الخاصية
#idx("tail recursion")
#emph[ذيلِيّ العَوْدِيَّة (#en[tail-recursive])].#footnote[عُرِفَت العَوْدِيَّة الذيلية لفترة طويلة كحيلة لتحسين المترجمات. وقدم كارت هويت
#idx("Hewitt, Carl Eddie")
(1977) أساساً دلالياً متماسكاً للعَوْدِيَّة الذيلية، وشرحها من حيث نموذج «تمرير الرسائل» للحساب الذي سنناقشه في الفصل @chap:state. وإلهاماً من هذا، قام جيرالد جاي سسمان و
#idx("Steele, Guy Lewis Jr.")
غاي لويس ستيل جونيور (انظر Steele 1975) بإنشاء مُفسِّر ذيلي العَوْدِيَّة لـ #en[Scheme]. وأظهر ستيل لاحقاً كيف أنّ العَوْدِيَّة الذيلية هي نتيجة للطريقة الطبيعية لتجميع استدعاءات الدوال
#idx("Sussman, Gerald Jay")
(Steele 1977).
ويتطلب معيار #en[IEEE] لـ #en[Scheme] أن تكون تنفيذات #en[Scheme]
#idx("tail recursion", sub: "in Scheme")
#idx("tail recursion", sub: "in Python")
#idx("Scheme", sub: "tail recursion in")
#idx("Python", sub: "tail recursion in")
ذيلية العَوْدِيَّة. لا يحدد مرجع لغة بايثون العَوْدِيَّة الذيلية بطريقة أو بأخرى. في حين أنّ معظم تنفيذات بايثون ليست ذيلية العَوْدِيَّة، فإننا نفترض في هذا الكتاب تنفيذاً يمتلك هذه الخاصية.]
مع التنفيذ ذيلي العَوْدِيَّة،
#idx("iterative process", sub: "implemented by function call")
يمكن التعبير عن التكرار باستخدام آلية استدعاء الدوال العادية، بحيث تكون بنى التكرار الخاصة مفيدة فقط كـ
#idx("syntactic sugar", sub: "looping constructs as")
سُكَّر نحوي.#footnote[يستكشف التمرين @ex:while_loop حلقات #py("while") في بايثون كسُكَّر نحوي للدوال التي تؤدي إلى عمليات تكرارية. تتميز اللغة الكاملة بايثون، مثل اللغات التقليدية الأخرى، بوفرة من الأشكال النحوية، والتي يمكن التعبير عنها جميعاً بشكل أكثر انتظاماً في لغة #en[Lisp].]

#idx("iterative process", sub: "recursive process vs.")
#idx("recursive process", sub: "iterative process vs.")

#exercise(label-name: <ex:addition-procedures>, [
كل من الدالتين التاليتين تحدّد طريقة لإضافة عددين صحيحين موجبين بدلالة الدالتين
#py("inc")، التي تزيد معطيتها بمقدار 1،
و#py("dec")، التي تنقص معطيتها بمقدار 1.

#snippet(```python
def plus(a, b):
    return b if a == 0 else inc(plus(dec(a), b))
```)

#snippet(```python
def plus(a, b):
    return b if a == 0 else plus(dec(a), inc(b))
```)

باستخدام نموذج الاستبدال، وُضِّح العملية الناتجة عن كل دالة في تقييم
#py("plus(4, 5)").
هل هذه العمليات تكرارية أم عَوْدِيَّة؟
])

#exercise(label-name: <ex:1_10>, [
الدالة التالية تحسب دالة رياضية تُسمّى
#idx("Ackermann's function")
#idx("function (mathematical)", sub: "Ackermann's")
دالة أكرمان #en[(Ackermann's function)].

#snippet(```python
def A(x, y):
    return (0 if y == 0
            else 2 * y if x == 0
            else 2 if y == 1
            else A(x - 1, A(x, y - 1)))
```)

ما الذي تطبعه التعليمات التالية؟

#snippet(```python
print(A(1, 10))
```)

#snippet(```python
print(A(2, 4))
```)

#snippet(```python
print(A(3, 3))
```)

تأمل الدوال التالية، حيث #py("A") هي الدالة المحدَّدة أعلاه:

#snippet(```python
def f(n):
    return A(0, n)

def g(n):
    return A(1, n)

def h(n):
    return A(2, n)

def k(n):
    return 5 * n * n
```)

أعطِ تعريفات رياضية موجزة للدوال التي تحسبها الدوال
#py("f")، و#py("g")، و#py("h") لِقَيِم $n$ الصحيحة الموجبة. على سبيل المثال،
$k(n)$ تحسب
$5n^(2)$.
])
