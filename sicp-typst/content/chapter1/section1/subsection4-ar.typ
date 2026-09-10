// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([الدوال المركّبة], label-name: <sec:compound-procedures>)

لقد حدّدنا في بايثون بعض العناصر التي يجب أن تظهر في أي لغة برمجة قوية:

- الأعداد والعمليات الحسابية هي بيانات ودوال أوّلية.
- تداخل التركيبات يوفّر وسيلة لتركيب العمليات.
- إسنادات التصريح التي تربط الأسماء بالقيم توفّر وسيلة محدودة للتجريد.

الآن سنتعلّم عن
#idx("def", sub: "function definition")
#emph[تعريفات الدوال]،
وهي تقنية تجريد أقوى بكثير يمكن بها إعطاء عملية مركّبة اسماً ثم الإشارة إليها
كوحدة.

نبدأ بفحص كيفية التعبير عن فكرة «التربيع». قد نقول: «لتربيع شيء ما، خذه
مضروباً في نفسه.»

يُعبَّر عن هذا في لغتنا كالتالي:
#idx("square", decl: true)#idx("def (keyword)")#idx("keywords", sub: "def")
#snippet(```python
def square(x): return x * x
```)

يمكننا فهم هذا بالطريقة التالية:

$ mat(delim: #none, mono(bold("def")), mono("square("), mono("x"), mono(")" mono(":")), mono(bold("return")), mono("x"), mono("*"), mono("x"), ; arrow.t, arrow.t, arrow.t, , , arrow.t, arrow.t, arrow.t, ; "لكي", "تُربِّع", "شيئاً", , "خذه", "مضروباً", "في", "نفسه", ) $

لدينا هنا
#idx("compound function") #emph[دالة مركّبة]
أُعطيت الاسم #py("square"). تمثّل الدالة عملية ضرب شيء في نفسه. الشيء المراد
ضربه يُعطى اسماً محلياً، #py("x")، الذي يلعب نفس الدور الذي يلعبه الضمير في
اللغة الطبيعية.
#idx("naming", sub: "of functions") #idx("function", sub: "naming (with def)") #idx("function", sub: "creating with def") #idx("function definition") #idx("definition", sub: "of function")
تقييم التعريف يُنشئ هذه الدالة المركّبة ويربطها بالاسم
#idx("syntactic forms", sub: "function definition")
#idx("def")
#idx("function definition")
#idx("declaration", sub: "of function (def)")
#py("square").#footnote[لاحظ أنّ هناك عمليتين مختلفتين مدمجتين هنا: نحن نُنشئ
الدالة، ونعطيها الاسم #py("square"). من الممكن بل من المهم أن نتمكّن من فصل
هذين المفهومين — إنشاء دوال دون تسميتها، وإعطاء أسماء لدوال أُنشئت بالفعل.
سنرى كيف نفعل هذا في القسم @sec:lambda.]

أبسط شكل لتعريف دالة هو

#syntax("\
def ", meta("name"), "(", meta("parameters"), "): return ", meta("expression"))

#idx("name", sub: "of a function") الاسم #meta("name")
هو رمز يُربط بتعريف الدالة في البيئة.#footnote[في جميع أنحاء هذا الكتاب
#idx("notation in this book", sub: "italic symbols in expression syntax")
#idx("syntax", sub: "of expressions, describing")
سنصف الصياغة العامة للتعبيرات باستخدام رموز مائلة — مثل #meta("name") — للدلالة
على «الفتحات» في التعبير التي يجب ملؤها عند الاستخدام الفعلي.]
#idx("parameters")
المعاملات #meta("parameters")
هي الأسماء المستخدمة داخل جسم الدالة للإشارة إلى الوسائط المقابلة للدالة.

الكلمة #py("def") هي #emph[كلمة محجوزة] في بايثون. تحمل الكلمات المحجوزة معنىً
خاصاً ولا يمكن استخدامها كأسماء. تُوجّه الكلمة المحجوزة في مكوّن برنامج مُفسِّرَ
بايثون لمعاملة المكوّن كشكل صياغي بقاعدة تقييم خاصة به.
تُجمع #meta("parameters")
#idx("parentheses", sub: "in function definition")
#idx("parentheses", sub: "in function definition")
داخل أقواس وتُفصل بفواصل، كما ستكون في تطبيق الدالة المُعرَّفة.
#idx("return statement")
#idx("return value")
#idx("return (keyword)")
#idx("syntactic forms", sub: "return statement")
#idx("keywords", sub: "return")
في أبسط شكل،
#idx("body of a function")
#emph[جسم] تعريف الدالة هو
#emph[تعليمة إرجاع] واحدة،#footnote[بشكل
#idx("sequence of statements", sub: "in function body")
أعم، يمكن أن يكون جسم الدالة تسلسلاً من التعليمات. في هذه الحالة يُقيّم المُفسِّر
كل تعليمة في التسلسل بالترتيب حتى تحدّد تعليمة إرجاع قيمة تطبيق الدالة.]
تتكوّن من الكلمة المحجوزة #py("return") متبوعة بـ #emph[تعبير الإرجاع]
الذي سيُنتج قيمة تطبيق الدالة حين تُستبدل المعاملات بالوسائط الفعلية التي
تُطبَّق عليها الدالة.
#idx("def", sub: "function definition")

بعد تعريف #py("square")، يمكننا الآن استخدامها في تعبير #emph[تطبيق دالة]:

#snippet(```python
square(21)
```)

تطبيقات الدوال هي — بعد تركيبات العوامل — النوع الثاني من تركيب التعبيرات في
تعبيرات أكبر نصادفه. الشكل العام لتطبيق دالة هو

#syntax(meta("function-expression"), "(", meta("argument-expressions"), ")
          ")

حيث
#idx("function expression")
#meta("function-expression") تحدّد الدالة المراد تطبيقها على
#idx("argument(s)")
#meta("argument-expressions") المفصولة بفواصل.
لتقييم تطبيق دالة، يتبع المُفسِّر
#idx("evaluation", sub: "of function application")
#idx("function application", sub: "evaluation of")
إجراءً مشابهاً جداً لإجراء تركيبات العوامل الموصوف في القسم @sec:evaluating-combinations.

- لتقييم تطبيق دالة، افعل ما يلي:

  + قيِّم التعبيرات الفرعية للتطبيق، أي تعبير الدالة وتعبيرات الوسائط.
  + طبِّق الدالة التي هي قيمة تعبير الدالة على قيم تعبيرات الوسائط.

ينطبق الإجراء ذاته على تطبيقات الدالة الأوّلية #py("print") التي صادفناها
بالفعل في القسم @sec:expressions.

#snippet(```python
print(square(2 + 5))
```)

#output(```python
print(square(2 + 5))
```)

هنا، تعبير وسيط #py("print") هو تعبير مركّب — تعبير التطبيق
#py("square(2 + 5)")، الذي تعبير وسيطه هو ذاته تعبير مركّب — تركيبة العوامل
#py("2 + 5").

#snippet(```python
print(square(square(3)))
```)

#output(```python
print(square(square(3)))
```)

بالطبع يمكن تداخل تعبيرات تطبيق الدوال أكثر.

يمكننا أيضاً استخدام #py("square") كوحدة بناء في تعريف دوال أخرى. مثلاً، يمكن
التعبير عن $x^(2) +y^(2)$ كـ

#snippet(```python
square(x) + square(y)
```)

يمكننا بسهولة تعريف دالة #py("sum_of_squares") التي، بإعطائها عددين كوسيطين،
تُنتج مجموع مربعيهما:
#idx("sumofsquares", decl: true)
#snippet(```python
def sum_of_squares(x, y):
    return square(x) + square(y)
```)

للوضوح، يمكننا بدء سطر جديد بعد النقطتين، وفي هذه الحالة يجب إزاحة جسم
الدالة.#footnote[عدد الحروف المستخدمة في الإزاحة مرن لكن يجب أن يكون متسقاً
في جميع أنحاء جسم الدالة. في هذا الكتاب نستخدم غالباً إزاحة بأربعة حروف.]

#snippet(```python
print(sum_of_squares(3, 4))
```)

#output(```python
print(sum_of_squares(3, 4))
```)

الآن يمكننا استخدام #py("sum_of_squares") كوحدة بناء في إنشاء دوال أخرى:

#snippet(```python
def f(a):
    return sum_of_squares(a + 1, a * 2)
```)

#snippet(```python
print(f(5))
```)

#output(```python
print(f(5))
```)

بالإضافة إلى الدوال المركّبة، توفّر أي بيئة بايثون
#idx("primitive function")
دوالاً أوّلية مدمجة في المُفسِّر أو مُحمَّلة من مكتبات.
#idx("Python environment used in this book")
إلى جانب الدوال الأوّلية التي توفّرها العوامل والدالة الأوّلية #py("print")،
تتضمّن بيئة بايثون المستخدمة في هذا الكتاب دوالاً أوّلية إضافية مثل الدالة
#idx("mathlog (primitive function)")
#py("math_log") التي تحسب اللوغاريتم الطبيعي لوسيطها. تُستخدم هذه الدوال
الأوّلية الإضافية بنفس طريقة
#idx("compound function", sub: "used like primitive function")
الدوال المركّبة تماماً؛ تقييم التطبيق #py("print(math_log(1))") يُظهر العدد 0.0.
في الواقع، لا يمكن معرفة ما إذا كانت #py("square") مدمجة في المُفسِّر أو
مُحمَّلة من مكتبة أو مُعرَّفة كدالة مركّبة بالنظر إلى تعريف
#py("sum_of_squares") المعطى أعلاه.
