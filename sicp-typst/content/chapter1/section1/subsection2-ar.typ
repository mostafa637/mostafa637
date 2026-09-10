// ترجمة عربية — من مصادر Typst الإنجليزية.
#import "../../../lib/sicp-ar.typ": *

#subsection([التسمية والبيئة], label-name: <sec:naming>)

جانب حاسم في أي لغة برمجة هو الوسائل التي توفّرها لاستخدام
#idx("naming", sub: "of computational objects")
الأسماء للإشارة إلى الكائنات الحسابية.
نقول إنّ
#idx("primitive expression", sub: "name of variable")
الاسم يحدّد
#idx("variable")
#emph[متغيّراً]
#idx("variable", sub: "value of")
#emph[قيمته] هي الكائن.

في بايثون، نُسمّي الأشياء بـ #idx("declaration assignment") #idx("declaration", sub: "of variable") #idx("syntactic forms", sub: "declaration assignment") #emph[إسنادات التصريح].

#snippet(```python
size = 2
```)

يجعل المُفسِّر يربط القيمة 2 بالاسم #py("size").#footnote[تستخدم بايثون نفس الصياغة #meta("name") \= #meta("expression") لإعادة إسناد قيمة #meta("name") حتى لو كان الاسم قد أُسنِد إليه سابقاً بإسناد تصريح. في هذا الفصل والفصل التالي لا نستخدم هذا الخيار. يناقش الفصل @chap:state إعادة الإسناد.]
بمجرد ربط الاسم #py("size")
بالعدد 2، يمكننا الإشارة إلى القيمة 2 بالاسم:

#snippet(```python
print(size)
```)

#output(```python
print(size)
```)

#snippet(```python
print(5 * size)
```)

#output(```python
print(5 * size)
```)

إليك أمثلة أخرى على استخدام إسنادات التصريح:

#snippet(```python
pi = 3.14159
```)

#snippet(```python
radius = 10
```)

#snippet(```python
print(pi * radius * radius)
```)

#output(```python
print(pi * radius * radius)
```)

#snippet(```python
circumference = 2 * pi * radius
```)

#snippet(```python
print(circumference)
```)

#output(```python
print(circumference)
```)

إسناد #idx("means of abstraction", sub: "declaration assignment as") التصريح
هو أبسط وسائل التجريد في لغتنا، إذ يسمح لنا باستخدام أسماء بسيطة للإشارة إلى
نتائج العمليات المركّبة، مثل #py("circumference") المحسوبة أعلاه. بشكل عام،
يمكن أن تكون للكائنات الحسابية بنى معقّدة جداً، وسيكون من غير الملائم أبداً
أن نتذكّر تفاصيلها ونكرّرها في كل مرة نريد استخدامها. بل إنّ البرامج المعقّدة
تُبنى خطوة بخطوة ببناء كائنات حسابية متزايدة التعقيد. يجعل المُفسِّر بناء
البرنامج التدريجي هذا ملائماً بشكل خاص لأنه يمكن إنشاء ربط الاسم بالكائن
تدريجياً في تفاعلات متتالية. تشجّع هذه الميزة
#idx("incremental development of programs")
#idx("program", sub: "incremental development of")
التطوير والاختبار التدريجي للبرامج وهي مسؤولة إلى حدٍّ كبير عن حقيقة أنّ
#idx("program", sub: "structure of")
برنامج بايثون يتكوّن عادة من عدد كبير من الدوال البسيطة نسبياً.

يجب أن يكون واضحاً أنّ إمكانية ربط القيم بالأسماء ثم استرجاعها لاحقاً تعني
أنّ المُفسِّر يجب أن يحتفظ بنوع من الذاكرة التي تتتبّع أزواج الاسم والكائن.
تُسمّى هذه الذاكرة
#idx("environment")
#emph[البيئة]
(بدقة أكثر
#idx("program environment") #emph[بيئة البرنامج]،
لأننا سنرى لاحقاً أنّ الحساب قد يتضمّن عدداً من البيئات
المختلفة).#footnote[سيُظهر الفصل @chap:state أنّ مفهوم البيئة هذا حاسم لفهم
كيفية عمل المُفسِّر. وسيستخدم الفصل @chap:meta البيئات لتنفيذ المُفسِّرات.]
