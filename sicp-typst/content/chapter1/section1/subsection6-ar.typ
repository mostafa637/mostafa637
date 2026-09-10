// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([التعبيرات الشرطية والمحمولات], label-name: <sec:conditionals>)

القوة التعبيرية لصنف الدوال التي يمكننا تعريفها حتى هذه النقطة محدودة جداً،
لأننا لا نملك طريقة لإجراء اختبارات وتنفيذ عمليات مختلفة بحسب نتيجة الاختبار.

مثلاً، لا يمكننا تعريف دالة تحسب
#idx("absolute value")
القيمة المطلقة لعدد باختبار ما إذا كان العدد غير سالب واتخاذ إجراءات مختلفة
في كل حالة وفق القاعدة

$ mat(delim: #none, |x|, =, cases(x quad upright("if x gt.eq 0"), -x quad upright("otherwise"))) . $

هذه البنية هي
#idx("case analysis")
#emph[تحليل حالات] ويمكن كتابتها في #en[Python] باستخدام #emph[تعبير شرطي] كالتالي:
#idx("abs", decl: true)
#snippet(```python
def abs(x):
    return x if x >= 0 else - x
```)

ويمكن التعبير عنه بالعربية: «أرجع $x$ إذا كان $x$ أكبر من أو يساوي الصفر؛
وإلا أرجع $- x$.»
الشكل العام للتعبير الشرطي هو

#syntax(meta("consequent-expression"), " if ", meta("predicate"), " else ", meta("alternative-expression"))

الكلمة المحجوزة #py("if") في التعبيرات
#idx("conditional expression")
#idx("syntactic forms", sub: "conditional expression")
#idx("if else expression")
#idx("predicate", sub: "of conditional expression")
#idx("True (keyword)", sort: "true")
#idx("False (keyword)", sort: "false")
#idx("keywords", sub: "True")
#idx("keywords", sub: "False")
#idx("expression", sub: "boolean")
#idx("boolean values (True, False)")
الشرطية يتبعها
#meta("predicate") — أي تعبير تُفسَّر قيمته على أنها صحيحة أو خاطئة.#footnote[«تُفسَّر على أنها #idx("true") #idx("false") #idx("#t", decl: true) #idx("true", decl: true) #idx("#f", decl: true) #idx("false", decl: true) صحيحة أو خاطئة»
تعني: في #en[Python] توجد قيمتان مميزتان يُشار إليهما بالثابتين
#py("True") و #py("False").
حين يتحقق المُفسِّر من قيمة محمول، يُفسِّر #py("False") على أنها خطأ
و #py("True") على أنها صواب. تعتبر #en[Python] أي قيمة إما صحيحة أو خاطئة،
لكننا في هذا الكتاب نستخدم هاتين القيمتين فقط.]
تسبق الكلمة المحجوزة #py("if")
#meta("consequent-expression")،
وتتبعها الكلمة المحجوزة #py("else")
ثم #meta("alternative-expression").

#idx("evaluation", sub: "of conditional expression")
#idx("conditional expression", sub: "evaluation of")
لتقييم تعبير شرطي، يبدأ المُفسِّر بتقييم #meta("predicate") التعبير.
إذا قُيِّم #meta("predicate") إلى صحيح، يُقيّم المُفسِّر
#idx("consequent", sub: "of conditional expression")
#meta("consequent-expression") ويُرجع قيمته كقيمة للتعبير الشرطي.
وإذا قُيِّم #meta("predicate") إلى خطأ، يُقيّم
#idx("alternative", sub: "of conditional expression")
#meta("alternative-expression") ويُرجع قيمته كقيمة للتعبير الشرطي.#footnote[#idx("conditional expression", sub: "non-boolean value as predicate")
التعبيرات الشرطية في #en[Python] الكاملة تقبل أي قيمة، وليس فقط القيم المنطقية 1 و 0، كنتيجة لتقييم
تعبير #meta("predicate") (انظر الحاشية @foot:truthy
في القسم @sec:eval-data-structures للتفاصيل). تستخدم برامج هذا الكتاب
القيم المنطقية فقط كمحمولات للتعبيرات الشرطية.]<foot:any-value-as-predicate>

الكلمة
#idx("predicate")
#emph[محمول] تُستخدم للعوامل والدوال التي تُرجع صحيحاً أو خاطئاً، وكذلك
للتعبيرات التي تُقيَّم إلى صحيح أو خطأ. تستخدم دالة القيمة المطلقة #py("abs")
#idx(">= (numeric comparison operator)")
#idx("number(s)", sub: "comparison of")
#idx("number(s)", sub: "equality of")
المحمول الأوّلي #py(">=")، وهو عامل يأخذ عددين كوسيطين ويختبر ما إذا كان
العدد الأول أكبر من أو يساوي الثاني، ويُرجع صحيحاً أو خاطئاً وفقاً لذلك.

إذا فضّلنا معالجة حالة الصفر بشكل منفصل، يمكننا تحديد الدالة التي تحسب القيمة
المطلقة لعدد بكتابة

$ mat(delim: #none, |x|, =, cases(x quad upright("if x > 0"), 0 quad upright("if x = 0"), -x quad upright("otherwise"))) . $

في #en[Python]، نعبّر عن تحليل حالات متعددة بتداخل التعبيرات الشرطية كتعبيرات
بديلة داخل تعبيرات شرطية أخرى:
#idx("abs", decl: true)
#snippet(```python
def abs(x):
    return x if x > 0 else 0 if x == 0 else - x
```)

الأقواس ليست ضرورية حول التعبير البديل
#py("0 if x == 0 else - x") لأنّ الشكل الصياغي للتعبير الشرطي
#idx("conditional expression", sub: "as alternative of conditional expression")
#idx("associativity", sub: "of conditional expression")
#idx("conditional expression", sub: "right-associativity of")
#idx("right-associative")
يميني الارتباط.
الشكل العام
#idx("case analysis", sub: "general")
لتحليل الحالات هو

#syntax(meta("e"), $""_(1)$, " if ", meta("p"), $""_(1)$, " else ", meta("e"), $""_(2)$, " if ", meta("p"), $""_(2)$, " ", $dots.c$, " ", meta("e"), $""_(n)$, " if ", meta("p"), $""_(n)$, " else ", meta("final-alternative-expression"))

نسمّي تعبير النتيجة $e_(i)$ مع محموله $p_(i)$
#idx("clause of a case analysis")
#idx("predicate", sub: "of clause")
#emph[فقرة]. يمكن النظر إلى تحليل الحالات
#idx("case analysis", sub: "as sequence of clauses")
كتسلسل من الفقرات متبوع بتعبير بديل نهائي.
وفق تقييم التعبيرات الشرطية، يُقيَّم تحليل الحالات بتقييم المحمول
#meta("p")$""_(1)$ أولاً. إذا كانت قيمته خاطئة، يُقيَّم #meta("p")$""_(2)$.
إذا كانت قيمة #meta("p")$""_(2)$ خاطئة أيضاً، يُقيَّم #meta("p")$""_(3)$.
تستمر هذه العملية حتى يُوجَد محمول قيمته صحيحة، وعندها يُرجع المُفسِّر قيمة
#idx("consequent", sub: "of clause")
تعبير النتيجة #meta("e") المقابل للفقرة كقيمة لتحليل الحالات.
إذا لم يُوجَد أي محمول صحيح، تكون قيمة تحليل الحالات هي قيمة التعبير البديل
النهائي.

بالإضافة إلى المحمولات الأوّلية مثل
#idx("> (numeric comparison operator)")
#idx("< (numeric comparison operator)", sort: ">=0")
#idx("<= (numeric comparison operator)", sort: ">=1")
#idx("==", sub: "as numeric equality operator")
#idx("!=", sub: "as numeric comparison operator", sort: ";4")
#idx("equality", sub: "of numbers")
#py(">=") و #py(">") و #py("<") و #py("<=") و #py("==") و #py("!=")
التي تُطبَّق على الأعداد،#footnote[حالياً نقصر هذه العوامل على وسائط عددية.
في القسمين @sec:strings و @sec:mutable-list-structure سنعمّم محمولات
المساواة وعدم المساواة #py("==") و #py("!=").]
توجد عمليات تركيب منطقي تمكّننا من بناء محمولات مركّبة. أكثر ثلاث استخداماً هي:

- #meta("expression")$""_(1)$ #py("and") #meta("expression")$""_(2)$\ تعبّر هذه العملية عن #idx("syntactic sugar", sub: "and and || as") #idx("and (logical conjunction)") #idx("and (logical conjunction)", sub: "evaluation of") #idx("syntactic forms", sub: "logical conjunction (and)") #idx("logical conjunction") #idx("conjunction") #idx("evaluation", sub: "of and") #emph[الاقتران المنطقي]، بمعنى مقارب للكلمة العربية «و». نفترض#footnote[هذا الافتراض مبرّر بالقيد المذكور في الحاشية @foot:any-value-as-predicate. #en[Python] الكاملة تحتاج لمعالجة الحالة التي تكون فيها نتيجة تقييم #meta("expression")$""_(1)$ ليست صحيحة ولا خاطئة.] أنّ هذا الشكل الصياغي سكّر صياغي#footnote[الأشكال الصياغية التي هي مجرد بنى سطحية بديلة ملائمة لأشياء يمكن كتابتها بطرق أكثر انتظاماً تُسمّى أحياناً #emph[سكّراً صياغياً]، باستخدام عبارة صاغها #idx("Landin, Peter") #idx("syntactic sugar") بيتر لاندن.] لـ\ #meta("expression")$""_(2)$ #py("if") #meta("expression")$""_(1)$ #py("else") #py("False").
- #meta("expression")$""_(1)$ #py("or") #meta("expression")$""_(2)$\ تعبّر هذه العملية عن #idx("or (logical disjunction)") #idx("or (logical disjunction)", sub: "evaluation of") #idx("syntactic forms", sub: "logical disjunction (or)") #idx("logical disjunction") #idx("disjunction") #idx("evaluation", sub: "of or") #emph[الانفصال المنطقي]، بمعنى مقارب للكلمة العربية «أو». نفترض أنّ هذا الشكل الصياغي سكّر صياغي لـ\ #py("True") #py("if") #meta("expression")$""_(1)$ #py("else") #meta("expression")$""_(2)$.
- #py("not") #meta("expression")\ تعبّر هذه العملية عن #idx("not (logical negation operator)") #idx("negation", sub: "logical (not)") #emph[النفي المنطقي]، بمعنى مقارب للكلمة العربية «ليس». قيمة التعبير صحيحة حين يُقيَّم #meta("expression") إلى خطأ، وخاطئة حين يُقيَّم إلى صحيح.

لاحظ أنّ #py("and") و #py("or") أشكال صياغية وليسا عاملين؛
#idx("and (logical conjunction)", sub: "why a syntactic form")
#idx("or (logical disjunction)", sub: "why a syntactic form")
تعبيرهما الأيمن لا يُقيَّم دائماً. أما العامل #py("not") فيتبع قاعدة التقييم
في القسم @sec:evaluating-combinations. وهو عامل #emph[أحادي]، بمعنى أنه يأخذ
وسيطاً واحداً فقط، بينما العوامل الحسابية والمحمولات الأوّلية المناقَشة حتى الآن
#emph[ثنائية] تأخذ وسيطين. يسبق العامل #py("not") وسيطه؛ ونسمّيه عاملاً
#idx("-", sub: "as numeric negation operator")
#idx("negation", sub: "numeric (-)")
#idx("binary operator")
#idx("unary operator")
#idx("prefix operator")
#emph[بادئاً]. عامل بادئ آخر هو عامل النفي العددي، مثاله التعبير #py("- x")
في دوال #py("abs") أعلاه.

كمثال على كيفية استخدام هذه المحمولات، يمكن التعبير عن شرط أن يكون العدد $x$
في المدى $5 < x < 10$ كالتالي:

#snippet(```python
x > 5 and x < 10
```)

الشكل الصياغي #py("and") له أسبقية أدنى من عاملي المقارنة #py(">") و #py("<")،
والشكل الصياغي للتعبير الشرطي
$dots.c$#py("if")$dots.c$#py("else")$dots.c$
له أسبقية أدنى من أي عامل صادفناه حتى الآن، وهي خاصية استخدمناها في دوال
#py("abs") أعلاه.

كمثال آخر، يمكننا تعريف محمول لاختبار ما إذا كان عدد أكبر من أو يساوي عدداً
آخر كالتالي:

#snippet(```python
def greater_or_equal(x, y):
    return x > y or x == y
```)

أو بدلاً من ذلك

#snippet(```python
def greater_or_equal(x, y):
    return not (x < y)
```)

الدالة #py("greater_or_equal") حين تُطبَّق على عددين تسلك سلوك العامل
#py(">="). العوامل الأحادية لها
#idx("precedence", sub: "of unary operators")
أسبقية أعلى من العوامل الثنائية، مما يجعل الأقواس في هذا المثال ضرورية.

#exercise(label-name: <ex:1_1>, [
فيما يلي تسلسل من التعليمات. ما النتيجة التي يطبعها المُفسِّر استجابة لكل
تعليمة؟ افترض أنّ التسلسل يُقيَّم بالترتيب المعروض.

#snippet(```python
print(10)
```)

#snippet(```python
print(5 + 3 + 4)
```)

#snippet(```python
print(9 - 1)
```)

#snippet(```python
print(6 / 2)
```)

#snippet(```python
print(2 * 4 + (4 - 6))
```)

#snippet(```python
a = 3
```)

#snippet(```python
b = a + 1
```)

#snippet(```python
print(a + b + a * b)
```)

#snippet(```python
print(a == b)
```)

#snippet(```python
print(b if b > a and b < a * b else a)
```)

#snippet(```python
print(6 if a == 4 else 6 + 7 + a if b == 4 else 25)
```)

#snippet(```python
print(2 + (b if b < a else a))
```)

#snippet(```python
print((a if a > b else b if a < b else -1) * (a + 1))
```)

الأقواس حول التعبيرات الشرطية في آخر تعليمتين ضرورية لأنّ الشكل الصياغي للتعبير
#idx("conditional expression", sub: "as operand of operator combination")
الشرطي له
#idx("precedence", sub: "of conditional expression")
#idx("conditional expression", sub: "precedence of")
أسبقية أدنى من العاملين الحسابيين #py("+") و #py("*").
])

#exercise(label-name: <ex:1_2>, [
ترجم التعبير التالي إلى #en[Python]:

$frac(5+4+lr(( 2-lr(( 3-(6+frac(4, 5)) )) )), 3 (6-2) (2-7))$
])

#exercise(label-name: <ex:1_3>, [
عرِّف دالة تأخذ ثلاثة أعداد كوسائط وتُرجع مجموع مربعي أكبر عددين.
])

#exercise(label-name: <ex:a-plus-abs-b>, [
لاحظ أنّ نموذج التقييم لدينا يسمح بتطبيقات
#idx("function application", sub: "compound expression as function expression of")
#idx("compound expression", sub: "as function expression of application")
#idx("function expression", sub: "compound expression as")
تكون تعبيرات دوالها تعبيرات مركّبة. استخدم هذه الملاحظة لوصف سلوك
#py("a_plus_abs_b"):

#snippet(```python
def plus(a, b): return a + b

def minus(a, b): return a - b

def a_plus_abs_b(a, b):
    return (plus if b >= 0 else minus)(a, b)
```)
])

#exercise(label-name: <ex:normal-order-vs-appl-order-test>, [
اخترع #en[Ben Bitdiddle] اختباراً لتحديد ما إذا كان المُفسِّر الذي يواجهه
يستخدم
#idx("normal-order evaluation", sub: "applicative order vs.")
#idx("applicative-order evaluation", sub: "normal order vs.")
التقييم بالترتيب التطبيقي أم بالترتيب العادي. عرَّف الدالتين التاليتين:

#snippet(```python
def p(): return p()

def test(x, y):
    return 0 if x == 0 else y
```)

ثم قيَّم التعليمة

#snippet(```python
test(0, p())
```)

ما السلوك الذي سيلاحظه #en[Ben] مع مُفسِّر يستخدم التقييم بالترتيب التطبيقي؟
وما السلوك الذي سيلاحظه مع مُفسِّر يستخدم التقييم بالترتيب العادي؟ اشرح
إجابتك.
#idx("normal-order evaluation", sub: "of conditional expressions")
#idx("conditional expression", sub: "normal-order evaluation of")
(افترض أنّ قاعدة تقييم التعبيرات الشرطية هي ذاتها سواء كان المُفسِّر يستخدم
الترتيب العادي أم التطبيقي: يُقيَّم تعبير المحمول أولاً، وتحدّد النتيجة ما إذا
كان سيُقيَّم تعبير النتيجة أم التعبير البديل.)
])
