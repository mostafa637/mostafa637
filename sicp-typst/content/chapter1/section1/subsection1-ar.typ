// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([التعبيرات], label-name: <sec:expressions>)

إحدى الطرق السهلة للبدء في البرمجة هي فحص بعض التفاعلات النموذجية مع مُفسِّر
لغة بايثون.

تكتب
#emph[تعليمة]
وتمرّرها إلى المُفسِّر الذي
#emph[يُقيِّم] تلك
#idx("statement") التعليمة.

نوع من التعليمات التي قد تكتبها هو
#emph[التعبير].
#idx("number(s)", sub: "in Python")
#idx("primitive expression", sub: "number")
#idx("expression")
نوع من التعبيرات الأوّلية هو العدد.

(بدقة أكثر، التعبير الذي تكتبه يتكوّن من الأرقام التي تمثّل العدد في الأساس 10.)

إذا قدّمت لبايثون البرنامج

#snippet(```python
486
```)

سيستجيب المُفسِّر بطباعة — لا شيء.
لرؤية نتيجة تقييم #py("486")،
نحتاج لتطبيق الدالة الأوّلية #py("print")
عليه، باستخدام الترميز الرياضي المعتاد لتطبيق الدوال

#snippet(```python
print(486)
```)

فينتج#footnote[في جميع أنحاء هذا الكتاب،
نُميّز
#idx("notation in this book", sub: "slanted characters for interpreter response")
بين المُدخَل الذي يكتبه المستخدم وأي نص يطبعه المُفسِّر بعرض الأخير بحروف مائلة.]

#output(```python
print(486)
```)

يمكن دمج التعبيرات التي تمثّل أعداداً مع
عوامل
(مثل
#idx("+", sub: "as numeric addition operator")

#py("+")
#idx("arithmetic", sub: "operators for")
#idx("* (multiplication operator)", sort: "-1")

أو #py("*")) لتشكيل
#idx("compound expression")
#idx("operator combination")
تعبير مركّب يمثّل تطبيق دالة أوّلية مقابلة على تلك الأعداد. مثلاً:

#snippet(```python
print(137 + 349)
```)

#output(```python
print(137 + 349)
```)

#snippet(```python
print(1000 - 334)
```)

#output(```python
print(1000 - 334)
```)

#snippet(```python
print(5 * 99)
```)

#output(```python
print(5 * 99)
```)

#idx("/ (division operator)")
#snippet(```python
print(10 / 4)
```)

#output(```python
print(10 / 4)
```)

#snippet(```python
print(2.7 + 10)
```)

#output(```python
print(2.7 + 10)
```)

تُسمّى التعبيرات مثل #py("137 + 349")، التي تحتوي على تعبيرات أخرى كمكوّنات،
#emph[تركيبات].
#idx("combination")
التركيبات المُشكَّلة من رمز
#idx("operator of a combination")
#idx("operator combination")
#emph[عامل] في المنتصف، وتعبيرات
#idx("operands of a combination")
#emph[معاملات] على يساره ويمينه، تُسمّى
#emph[تركيبات العوامل].
#idx("value", sub: "of an expression")
تُحصَل قيمة تركيبة العوامل بتطبيق الدالة المحدّدة بالعامل على الوسائط التي هي
قيم المعاملات.

يُعرف اصطلاح وضع العامل بين المعاملات بالترميز
#idx("infix operator")
#idx("infix notation")
#emph[الوسطي (infix)]. وهو يتبع الترميز الرياضي الذي تعرفه على الأرجح من
المدرسة والحياة اليومية. وكما في الرياضيات، يمكن أن تكون تركيبات العوامل
#emph[متداخلة]، أي يمكن أن تحتوي على معاملات
#idx("nested operator combinations")
هي ذاتها تركيبات عوامل:

#snippet(```python
print((3 * 5) + (10 - 6))
```)

#output(```python
print((3 * 5) + (10 - 6))
```)

كالمعتاد،
#idx("parentheses", sub: "to group operator combinations")
تُستخدم الأقواس لتجميع تركيبات العوامل لتجنّب الغموض. تتبع بايثون أيضاً
الاصطلاحات المعتادة حين تُحذف الأقواس: الضرب والقسمة يرتبطان أقوى من الجمع
والطرح. مثلاً:

#snippet(```python
3 * 5 + 10 / 2
```)

تعني

#snippet(```python
(3 * 5) + (10 / 2)
```)

نقول إنّ #py("*") و
#py("/") لهما
#idx("precedence", sub: "of operators")
#emph[أسبقية أعلى]
من #py("+") و
#py("-"). تُقرأ تسلسلات الجمع والطرح من اليسار إلى اليمين، وكذلك تسلسلات
الضرب والقسمة. وبالتالي:
#idx("-", sub: "as numeric subtraction operator")
#snippet(```python
1 - 5 / 2 * 4 + 3
```)

تعني

#snippet(```python
(1 - ((5 / 2) * 4)) + 3
```)

نقول إنّ العوامل
#py("+") و
#py("-") و
#py("*") و
#py("/") هي
#idx("associativity", sub: "of operators")
#idx("left-associative")
#emph[يسارية الارتباط].

لا يوجد حدّ (من حيث المبدأ) لعمق هذا التداخل ولا للتعقيد الكلي للتعبيرات التي
يمكن لمُفسِّر بايثون تقييمها. نحن البشر من قد يرتبك بتعبيرات لا تزال بسيطة
نسبياً مثل

#snippet(```python
3 * (2 * 4 + (3 + 5)) + ((10 - 7) + 6)
```)

التي سيُقيّمها المُفسِّر بسهولة إلى 57. يمكننا مساعدة أنفسنا بكتابة مثل هذا
التعبير بالصيغة

#snippet(```python
(3 * (2 * 4 + (3 + 5))
 +
 ((10 - 7) + 6))
```)

للفصل البصري بين المكوّنات الرئيسية للتعبير.#footnote[الأقواس الإضافية ضرورية
إذا أردنا نشر تعبير على عدة أسطر ولم تكن هناك أقواس محيطة بالفعل.]

حتى مع التعبيرات المعقّدة، يعمل المُفسِّر دائماً بنفس الدورة الأساسية: يقرأ
تعليمة يكتبها المستخدم، ويُقيّم التعليمة، ويطبع نتيجة أي تطبيقات لـ #py("print").
يُعبَّر عن هذا النمط من التشغيل غالباً بالقول إنّ المُفسِّر يعمل في
#idx("read-evaluate-print loop") #idx("interpreter", sub: "read-evaluate-print loop") #emph[حلقة اقرأ-قيِّم-اطبع].
لاحظ مع ذلك أنه من الضروري توجيه المُفسِّر صراحة لطباعة قيمة التعبير.
