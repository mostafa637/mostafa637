// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([مثال: التفاضل الرمزي], label-name: <sec:symbolic-differentiation>)

#idx("differentiation", sub: "symbolic")
#idx("symbolic differentiation")
#idx("algebraic expression", sub: "differentiating")

كتوضيح للتلاعب بالرموز وكارتقاء آخر بتجريد البيانات، تأمل تصميم دالة تؤدي التفاضل الرمزي للتعبيرات الجبرية. نود أن تأخذ الدالة كمعاملات تعبيراً جبرياً ومتغيراً وأن ترجع مشتقة التعبير بالنسبة للمتغير. فعلى سبيل المثال، إذا كانت المعاملات للدالة هي $a x^(2) + b x + c$ و $x$، فيجب أن ترجع الدالة $2a x + b$. والتفاضل الرمزي له أهمية تاريخية خاصة في لغة البرمجة #en[Lisp].#footnote[استخدمت النسخة الأصلية من هذا الكتاب لغة البرمجة #en[Scheme]، وهي لهجة من #en[Lisp].]
وكان أحد الأمثلة المحفزة وراء تطوير لغة حاسوب للتلاعب بالرموز. علاوة على ذلك، مثل ذلك بداية خط البحث الذي أدى إلى تطوير أنظمة قوية للأعمال الرياضية الرمزية، والتي يستخدمها الرياضيون التطبيقيون والفيزيائيون اليوم بشكل روتيني.

عند تطوير برنامج التفاضل الرمزي، سنتبع استراتيجية تجريد البيانات نفسها التي اتبعناها عند تطوير نظام الأعداد الكسرية في القسم @sec:rationals. أي أننا سنعرف أولاً خوارزمية تفاضل تعمل على كائنات مجردة مثل "المجاميع" و "الجداءات" و "المتغيرات" دون القلق بشأن كيفية تمثيلها. وبعد ذلك فقط سنتناول مسألة التمثيل.

#subheading([برنامج التفاضل بالبيانات المجردة])

لإبقاء الأمور بسيطة، سننظر في برنامج تفاضل رمزي بسيط للغاية يتعامل مع التعبيرات المبنية باستخدام عمليتي الجمع والضرب بمعاملين فقط. ويمكن إجراء التفاضل لأي تعبير من هذا القبيل عن طريق تطبيق قواعد التخفيض التالية (#idx("differentiation", sub: "rules for")):

$ mat(delim: #none, frac(d c, d x), =, 0 upright("لكل ثابت c أو متغير مختلف عن x") ; frac(d x, d x), =, 1 ; frac(d(u+v), d x), =, frac(d u, d x)+frac(d v, d x) ; frac(d(u v), d x), =, u l r(( frac(d v, d x) ))+v l r(( frac(d u, d x) ))) $

لاحظ أن القاعتين الأخيرتين تعاوديتان بطبيعتهما. أي أنه للحصول على مشتقة مجموع، نجد أولاً مشتقات الحدود ونجمعها. وقد يكون كل حد بدوره تعبيراً يلزمه التفكيك. والتفكيك إلى قطع أصغر وأصغر سينتج في النهاية قطعاً إما ثوابت وإما متغيرات، والتي ستكون مشتقاتها إما $0$ وإما $1$.

لتجسيد هذه القواعد في دالة، نلجأ إلى القليل من #idx("wishful thinking") #emph[التفكير المتمني] (#en[wishful thinking])، كما فعلنا في تصميم تنفيذ الأعداد الكسرية. فلو كانت لدينا وسيلة لتمثيل التعبيرات الجبرية، فسينبغي أن نكون قادرين على تحديد ما إذا كان التعبير مجموعاً أم جداءً أم ثابتاً أم متغيراً. وينبغي أن نكون قادرين على استخراج أجزاء التعبير. بالنسبة للمجموع، على سبيل المثال، نريد التمكن من استخراج الحد المضاف الأول (#py("addend")) والحد المضاف الثاني (#py("augend")). وينبغي أن نكون قادرين أيضاً على بناء تعبيرات من الأجزاء. لنفترض أن لدينا بالفعل دوال لتنفيذ محددات الاختيارات والبواني والمحمولات التالية:

#sicp-table(columns: 2, [#py("is_variable(e)")], [هل #py("e") متغير؟], [#py("is_same_variable(v1, v2)")], [هل #py("v1") و #py("v2") هما المتغير نفسه؟], [#py("is_sum(e)")], [هل #py("e") مجموع؟], [#py("addend(e)")], [الحد المضاف الأول للمجموع #py("e").], [#py("augend(e)")], [الحد المضاف الثاني للمجموع #py("e").], [#py("make_sum(a1, a2)")], [بناء مجموع #py("a1") و #py("a2").], [#py("is_product(e)")], [هل #py("e") جداء؟], [#py("multiplier(e)")], [المضروب في الجداء #py("e").], [#py("multiplicand(e)")], [المضروب فيه في الجداء #py("e").], [#py("make_product(m1, m2)")], [بناء جداء #py("m1") و #py("m2").])

باستخدام هذه، والمحمول الأولي #idx("isnumber (primitive function)") #py("is_number") الذي يحدد الأعداد، يمكننا التعبير عن قواعد التفاضل كالدالة التالية:
#idx("deriv (symbolic)", decl: true)
#snippet(```python
def deriv(exp, variable):
    return (0
            if is_number(exp)
            else (1 if is_same_variable(exp, variable) else 0)
            if is_variable(exp)
            else make_sum(deriv(addend(exp), variable),
                          deriv(augend(exp), variable))
            if is_sum(exp)
            else make_sum(make_product(multiplier(exp),
                                       deriv(multiplicand(exp),
                                             variable)),
                  make_product(deriv(multiplier(exp),
                                             variable),
                                       multiplicand(exp)))
            if is_product(exp)
            else error("unknown expression type -- deriv", exp))
```)

تتضمن دالة #py("deriv") هذه خوارزمية التفاضل الكاملة. وبما أنها معبر عنها بدلالة البيانات المجردة، فسوف تعمل بغض النظر عن كيفية اختيارنا لتمثيل التعبيرات الجبرية، طالما أننا نصمم مجموعة مناسبة من محددات الاختيارات والبواني. وهذه هي المسألة التي يجب أن نتناولها بعد ذلك.

#subheading([تمثيل التعبيرات الجبرية])

#idx("algebraic expression", sub: "representing")

يمكننا تخيل طرق عديدة لاستخدام بنية القوائم المترابطة لتمثيل التعبيرات الجبرية. فعلى سبيل المثال، يمكننا استخدام قوائم مترابطة من الرموز تعكس الترميز الجبري المعتاد، ممثلين $a x + b$ كـ
#py("llist(\"a\", \"*\", \"x\", \"+\", \"b\")").
ومع ذلك، سيكون من الأكثر ملاءمة أن نعكس البنية الرياضية للتعبير في قيمة #en[Python] التي تمثله؛ أي تمثيل $a x + b$ ك#py("llist(\"+\", llist(\"*\", \"a\", \"x\"), \"b\")"). وتسمى ممارسة وضع العامل الثنائي أمام معاملاته بـ #idx("prefix notation") #emph[الترميز البادئي] (#en[prefix notation])، على النقيض من الترميز الإقحامي المعتاد المألوف. ومع الترميز البادئي، يكون تمثيل بياناتنا لمشكلة التفاضل كما يلي:

- المتغيرات هي مجرد سلاسل نصية. ويتم التعرف عليها بوساطة المحمول الأولي #idx("isstring (primitive function)") #py("is_string"): #idx("isvariable", sub: "for algebraic expressions", decl: true) #snippet(```python def is_variable(x): return is_string(x) ```)
- المتغيران هما المتغير نفسه إذا كانت السلاسل النصية الممثلة لهما متساوية: #idx("issamevariable", decl: true) #snippet(```python def is_same_variable(v1, v2): return is_variable(v1) and is_variable(v2) and v1 == v2 ```)
- المجاميع والجداءات تُبنى كقوائم مترابطة: #idx("makesum", decl: true)#idx("makeproduct", decl: true) #snippet(```python def make_sum(a1, a2): return llist("+", a1, a2) def make_product(m1, m2): return llist("*", m1, m2) ```)
- المجموع هو قائمة مترابطة عنصرها الأول هو السلسلة النصية #py("\"+\""): #idx("issum", decl: true) #snippet(```python def is_sum(x): return is_pair(x) and head(x) == "+" ```)
- الحد المضاف الأول هو العنصر الثاني من قائمة المجموع المترابطة: #idx("addend", decl: true) #snippet(```python def addend(s): return head(tail(s)) ```)
- الحد المضاف الثاني هو العنصر الثالث من قائمة المجموع المترابطة: #idx("augend", decl: true) #snippet(```python def augend(s): return head(tail(tail(s))) ```)
- الجداء هو قائمة مترابطة عنصرها الأول هو السلسلة النصية #py("\"*\""): #idx("isproduct", decl: true) #snippet(```python def is_product(x): return is_pair(x) and head(x) == "*" ```)
- المضروب هو العنصر الثاني من قائمة الجداء المترابطة: #idx("multiplier", sub: "selector", decl: true) #snippet(```python def multiplier(s): return head(tail(s)) ```)
- المضروب فيه هو العنصر الثالث من قائمة الجداء المترابطة: #idx("multiplicand", decl: true) #snippet(```python def multiplicand(s): return head(tail(tail(s))) ```)

وهكذا، نحتاج فقط إلى دمج هذه مع الخوارزمية التجريدية المنفذة بوساطة #py("deriv") للحصول على برنامج تفاضل رمزي يعمل. لننظر في بعض الأمثلة على سلوكه:

#snippet(```python
print_llist(deriv(llist("+", "x", 3), "x"))
```)

#output(```python
print_llist(deriv(llist("+", "x", 3), "x"))
```)

#snippet(```python
print_llist(deriv(llist("*", "x", "y"), "x"))
```)

#output(```python
print_llist(deriv(llist("*", "x", "y"), "x"))
```)

#snippet(```python
print_llist(deriv(llist("*", llist("*", "x", "y"),
                             llist("+", "x", 3)), "x"))
```)

#output(```python
print_llist(deriv(llist("*", llist("*", "x", "y"),
                             llist("+", "x", 3)), "x"))
```)

ينتج البرنامج إجابات صحيحة؛ ومع ذلك، فهي غير مبسطة. فمن الصحيح أن:

$ mat(delim: #none, frac(d(x y), d x), =, x dot.op 0+1 dot.op y) $

ولكننا نود أن يعرف البرنامج أن $x dot.op 0 = 0$ و $1 dot.op y = y$ و $0 + y = y$. وكان ينبغي أن تكون الإجابة للمثال الثاني مجرد #py("y"). وكما يظهر المثال الثالث، يصبح هذا مشكلة خطيرة عندما تكون التعبيرات معقدة.

صعوبتنا تشبه كثيراً الصعوبة التي واجهناها في تنفيذ الأعداد الكسرية: #idx("algebraic expression", sub: "simplifying") #idx("simplification of algebraic expressions") لم نختزل الإجابات إلى أبسط صورة. ولإنجاز تخفيض الأعداد الكسرية، احتجنا فقط إلى تغيير البواني ومحددات الاختيارات في التنفيذ. ويمكننا اعتماد استراتيجية مماثلة هنا. فلن نغير #py("deriv") على الإطلاق. وبدلاً من ذلك، سنغير #py("make_sum") بحيث إذا كان كلا المضافين أعداداً، فإن #py("make_sum") سجمعهما وترجع مجموعهما. وأيضاً، إذا كان أحد المضافين 0، فإن #py("make_sum") سترجع المضاف الآخر.

#idx("makesum", decl: true)
#snippet(```python
def make_sum(a1, a2):
    return (a2
            if number_equal(a1, 0)
            else a1
            if number_equal(a2, 0)
            else a1 + a2
            if is_number(a1) and is_number(a2)
            else llist("+", a1, a2))
```)

يستخدم هذا الدالة #py("number_equal")، والتي تفحص ما إذا كان التعبير مساوياً لعدد معطى:

#idx("numberequal", decl: true)
#snippet(```python
def number_equal(exp, num):
    return is_number(exp) and exp == num
```)

وبالمثل، سنغير #py("make_product") لتدمج القواعد التي تقول إن 0 مضروباً في أي شيء هو 0 وأن 1 مضروباً في أي شيء هو الشيء نفسه:

#idx("makeproduct", decl: true)
#snippet(```python
def make_product(m1, m2):
    return (0
            if number_equal(m1, 0) or number_equal(m2, 0)
            else m2
            if number_equal(m1, 1)
            else m1
            if number_equal(m2, 1)
            else m1 * m2
            if is_number(m1) and is_number(m2)
            else llist("*", m1, m2))
```)

إليك كيفية عمل هذه النسخة على أمثلتنا الثلاثة:

#snippet(```python
print(deriv(llist("+", "x", 3), "x"))
```)

#output(```python
print(deriv(llist("+", "x", 3), "x"))
```)

#snippet(```python
print(deriv(llist("*", "x", "y"), "x"))
```)

#output(```python
print(deriv(llist("*", "x", "y"), "x"))
```)

#snippet(```python
print_llist(deriv(llist("*", llist("*", "x", "y"),
                             llist("+", "x", 3)), "x"))
```)

#output(```python
print_llist(deriv(llist("*", llist("*", "x", "y"),
                             llist("+", "x", 3)), "x"))
```)

على الرغم من أن هذا تحسن كبير، فإن المثال الثالث يظهر أنه لا يزال هناك طريق طويل قبل الحصول على برنامج يضع التعبيرات في شكل قد نتفق على أنه "الأبسط". ومشكلة التبسيط الجبري معقدة لأنه، من بين أسباب أخرى، قد لا يكون الشكل الذي قد يكون الأبسط لغرض ما هو الأبسط لغرض آخر.
#idx("algebraic expression", sub: "simplifying")

#exercise(label-name: <ex:deriv-exponentiation>, [
أظهر كيفية توسيع مفرّق المشتقات الأساسي ليتعامل مع أنواع أكثر من التعبيرات.
#idx("differentiation", sub: "rules for")
على سبيل المثال، نفذ قاعدة التفاضل:

$ mat(delim: #none, frac(d(u^(n)), d x), =, n u^(n-1)lr(( frac(d u, d x) ))) $

عن طريق إضافة بند جديد إلى برنامج #py("deriv") وتعريف الدوال المناسبة #py("is_exp") و #py("base") و #py("exponent") و #py("make_exp").
(يمكنك استخدام السلسلة النصية #py("\"**\"") للإشارة إلى الرفع إلى قوة.) وادمج القواعد التي تقول إن أي شيء مرفوع للقوة 0 هو 1 وأي شيء مرفوع للقوة 1 هو الشيء نفسه.
])

#exercise(label-name: <ex:2_57>, [
وسّع برنامج التفاضل للتعامل مع مجاميع وجداءات عدد اعتباطي من الحدود (حدين أو أكثر). وعندئذ يمكن التعبير عن المثال الأخير أعلاه كـ:

#snippet(```python
deriv(llist("*", "x", "y", llist("+", "x", 3)), "x")
```)

حاول القيام بذلك عن طريق تغيير تمثيل المجاميع والجداءات فقط، دون تغيير دالة #py("deriv") على الإطلاق. فعلى سبيل المثال، سيكون #py("addend") المجموع هو الحد الأول، ويكون #py("augend") هو مجموع بقية الحدود.
])

#exercise(label-name: <ex:2_58>, [
نفترض أننا نريد تعديل برنامج التفاضل بحيث يعمل مع الترميز الرياضي المعتاد، حيث تكون #py("\"+\"") و #py("\"*\"") من العوامل الإقحامية بدلاً من البادئية (#idx("infix notation", sub: "prefix notation vs.") #idx("prefix notation", sub: "infix notation vs.")). وبما أن برنامج التفاضل معرف بدلالة البيانات المجردة، يمكننا تعديله ليعمل مع تمثيلات مختلفة للتعبيرات فقط عن طريق تغيير المحمولات ومحددات الاختيارات والبواني التي تعرّف تمثيل التعبيرات الجبرية التي سيعمل عليها مفرّق المشتقات.

+ أظهر كيفية القيام بذلك من أجل تفاضل التعبيرات الجبرية المقدمة في شكل إقحامي، كما في هذا المثال: #snippet(```python llist("x", "+", llist(3, "*", llist("x", "+", llist("y", "+", 2)))) ```) لتبسيط المهمة، افترض أن #py("\"+\"") و #py("\"*\"") تأخذان دائماً معاملين وأن التعبيرات محاطة بالأقواس بالكامل.
+ تتفاقم الصعوبة بشكل كبير إذا سمحنا بترميز أقرب إلى الترميز الإقحامي المعتاد، والذي يحذف الأقواس غير الضرورية ويفترض أن الضرب له أسبقية أعلى من الجمع، كما في هذا المثال: #snippet(```python llist("x", "+", 3, "*", llist("x", "+", "y", "+", 2)) ```) هل يمكنك تصميم محمولات ومحددات اختيارات وبوانٍ مناسبة لهذا الترميز بحيث لا يزال برنامج المشتقة الخاص بنا يعمل؟
])

#idx("algebraic expression", sub: "representing")
#idx("differentiation", sub: "symbolic")
#idx("symbolic differentiation")
#idx("algebraic expression", sub: "differentiating")
