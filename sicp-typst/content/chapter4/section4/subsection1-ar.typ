// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([استرجاع المعلومات الاستنباطي], label-name: <sec:deductive-info-retrieval>)

#idx("query language")

تتفوق البرمجة المنطقية في توفير واجهات لـ
#idx("data base", sub: "logic programming and")
قواعد البيانات لاسترجاع المعلومات. ولغة الاستعلام التي سننفذها في هذا الفصل صُمِّمَت لاستخدامها بهذه الطريقة.

ولكشف ما يفعله نظام الاستعلام، سنوضح كيف يمكن استخدامه لإدارة قاعدة بيانات السجلات الوظيفية للـ
#idx("Gargle")
شركة #en[Gargle]، وهي شركة تكنولوجيا حديثة ومزدهرة في منطقة بوسطن. وتوفر اللغة وصولاً موجهًا بالأنماط إلى معلومات الموظفين، ويمكنها أيضًا الاستفادة من القواعد العامة لإجراء استنباطات منطقية.

#subheading([قاعدة بيانات عينة])

#idx("query language", sub: "data base")

#idx("data base", sub: "Gargle personnel")
تحتوي قاعدة بيانات موظفي شركة #en[Gargle] على
#idx("assertion")
#emph[تقريرات جازمة] (#en[assertions]) حول موظفي الشركة. إليك المعلومات عن بن بيتيدل، الساحر البرمجي المقيم:

#snippet(```python
address(llist("Bitdiddle", "Ben"),
        llist("Slumerville", llist("Ridge", "Road"), 10))
job(llist("Bitdiddle", "Ben"), llist("computer", "wizard"))
salary(llist("Bitdiddle", "Ben"), 122000)
```)

تبدو التقريرات الجازمة تمامًا مثل تطبيقات الدوال في #en[Python]، ولكنها تمثل في الواقع معلومات في قاعدة البيانات. وتصف الرموز الأولى — هنا #py("address") و#py("job") و#py("salary") — #emph[نوع المعلومات] المحتواة في التقرير الجازم المعني، والوسائط عبارة عن قوائم أو قيم أولية مثل السلاسل النصية والأعداد. ولا يلزم التصريح عن الرموز الأولى، كما هو الحال مع الثوابت أو المتغيرات في #en[Python]؛ فنطاقها عام.

وبصفته الساحر المقيم، يتولى بن مسؤولية قسم الحاسوب بالشركة، ويشرف على مبرمجين واثنين وفني واحد. إليك المعلومات عنهم:

#snippet(```python
address(llist("Hacker", "Alyssa", "P"),
        llist("Cambridge", llist("Mass", "Ave"), 78))
job(llist("Hacker", "Alyssa", "P"), llist("computer", "programmer"))
salary(llist("Hacker", "Alyssa", "P"), 81000)
supervisor(llist("Hacker", "Alyssa", "P"), llist("Bitdiddle", "Ben"))

address(llist("Fect", "Cy", "D"),
        llist("Cambridge", llist("Ames", "Street"), 3))
job(llist("Fect", "Cy", "D"), llist("computer", "programmer"))
salary(llist("Fect", "Cy", "D"), 70000)
supervisor(llist("Fect", "Cy", "D"), llist("Bitdiddle", "Ben"))

address(llist("Tweakit", "Lem", "E"),
        llist("Boston", llist("Bay", "State", "Road"), 22))
job(llist("Tweakit", "Lem", "E"), llist("computer", "technician"))
salary(llist("Tweakit", "Lem", "E"), 51000)
supervisor(llist("Tweakit", "Lem", "E"), llist("Bitdiddle", "Ben"))
```)

وهناك أيضًا مبرمج متدرب تشرف عليه أليسا:

#snippet(```python
address(llist("Reasoner", "Louis"),
        llist("Slumerville", llist("Pine", "Tree", "Road"), 80))
job(llist("Reasoner", "Louis"),
         llist("computer", "programmer", "trainee"))
salary(llist("Reasoner", "Louis"), 62000)
supervisor(llist("Reasoner", "Louis"), llist("Hacker", "Alyssa", "P"))
```)

وجميع هؤلاء الأشخاص يعملون في قسم الحاسوب، كما يُشار إلى ذلك بالكلمة #py("\"computer\"") كأول عنصر في أوصاف وظائفهم.

وبن موظف عالي المستوى. ومشرفه هو المدير الكسير بالشركة نفسه:

#snippet(```python
supervisor(llist("Bitdiddle", "Ben"), llist("Warbucks", "Oliver"))

address(llist("Warbucks", "Oliver"),
        llist("Swellesley", llist("Top", "Heap", "Road")))
job(llist("Warbucks", "Oliver"), llist("administration", "big", "wheel"))
salary(llist("Warbucks", "Oliver"), 314159)
```)

بالإضافة إلى قسم الحاسوب الذي يشرف عليه بن، تضم الشركة قسم المحاسبة، ويتكون من رئيس المحاسبين ومساعده:

#snippet(```python
address(llist("Scrooge", "Eben"),
        llist("Weston", llist("Shady", "Lane"), 10))
job(llist("Scrooge", "Eben"), llist("accounting", "chief", "accountant"))
salary(llist("Scrooge", "Eben"), 141421)
supervisor(llist("Scrooge", "Eben"), llist("Warbucks", "Oliver"))

address(llist("Cratchit", "Robert"),
        llist("Allston", llist("N", "Harvard", "Street"), 16))
job(llist("Cratchit", "Robert"), llist("accounting", "scrivener"))
salary(llist("Cratchit", "Robert"), 26100)
supervisor(llist("Cratchit", "Robert"), llist("Scrooge", "Eben"))
```)

وهناك أيضًا مساعد إداري للمدير الكسير:

#snippet(```python
address(llist("Aull", "DeWitt"),
        llist("Slumerville", llist("Onion", "Square"), 5))
job(llist("Aull", "DeWitt"), llist("administration", "assistant"))
salary(llist("Aull", "DeWitt"), 42195)
supervisor(llist("Aull", "DeWitt"), llist("Warbucks", "Oliver"))
```)

تحتوي قاعدة البيانات أيضًا على تقريرات جازمة حول أنواع الوظائف التي يمكن أن يقوم بها أشخاص يشغلون أنواعًا أخرى من الوظائف. على سبيل المثال، يمكن لساحر الحاسوب القيام بوظائف كل من مبرمج الحاسوب وفني الحاسوب:

#snippet(```python
can_do_job(llist("computer", "wizard"),
           llist("computer", "programmer"))
can_do_job(llist("computer", "wizard"),
           llist("computer", "technician"))
```)

ويمكن لمبرمج الحاسوب أن يحل محل متدرب:

#snippet(```python
can_do_job(llist("computer", "programmer"),
           llist("computer", "programmer", "trainee"))
```)

#idx("administrative assistant, importance of")
وكما هو معروف أيضًا،

#snippet(```python
can_do_job(llist("administration", "assistant"),
           llist("administration", "big", "wheel"))
```)

#idx("data base", sub: "Gargle personnel")
#idx("query language", sub: "data base")

#subheading([استعلامات بسيطة])

#idx("simple query")

تتيح لغة الاستعلام للمستخدمين استرجاع المعلومات من قاعدة البيانات عن طريق طرح استعلامات رداً على محث النظام.
على سبيل المثال، للعثور على جميع مبرمجي الحاسوب، يمكن للمرء كتابة

#prompt(```python
Query input:
```)

#snippet(```python
job($x, llist("computer", "programmer"))
```)

وسيرد النظام بالعناصر التالية:

#output(```python
job($x, llist("computer", "programmer"))
```)

يحدد استعلام المدخلات أننا نبحث عن إدخالات في قاعدة البيانات تطابق
#idx("pattern")
#emph[نمطًا] (#en[pattern]) معينًا.

في هذا المثال، يحدد النمط #py("job") كنوع المعلومات التي نبحث عنها. ويمكن أن يكون العنصر الأول أي شيء، والعنصر الثاني هو القائمة الحرفية #py("llist(\"computer\", \"programmer\")").
و«أي شيء» الذي يمكن أن يكون العنصر الأول في التقرير الجازم المطابق يتحدد بواسطة
#idx("pattern variable")
#emph[متغير نمط] (#en[pattern variable])،
#py("$x"). كمتغيرات أنماط، نستخدم
#idx("naming conventions", sub: "$ for pattern variables")
#idx("$, pattern variables starting with", sort: "0a4")
أسماء #en[Python] التي تبدأ بعلامة الدولار.
وسنرى أدناه لماذا من المفيد تحديد أسماء لمتغيرات الأنماط بدلاً من مجرد وضع رمز واحد مثل #py("$") في الأنماط لتمثيل «أي شيء».

يرد النظام على استعلام بسيط عن طريق إظهار جميع الإدخالات في قاعدة البيانات التي تطابق النمط المحدد.

ويمكن أن يكون للنمط أكثر من متغير واحد. على سبيل المثال، الاستعلام

#snippet(```python
address($x, $y)
```)

سيسرد جميع عناوين الموظفين.

ويمكن ألا يحتوي النمط على متغيرات، وفي هذه الحالة يحدد الاستعلام ببساطة ما إذا كان هذا النمط إدخالاً في قاعدة البيانات أم لا. وإذا كان الأمر كذلك، فستكون هناك مطابقة واحدة؛ وإذا لم يكن كذلك، فلن تكون هناك مطابقات.

وقد يظهر نفس متغير النمط أكثر من مرة في استعلام، محددًا أنه يجب أن يظهر نفس «أي شيء» في كل موضع. ولهذا السبب تمتلك المتغيرات أسماءً. على سبيل المثال،

#snippet(```python
supervisor($x, $x)
```)

يجد جميع الأشخاص الذين يشرفون على أنفسهم (على الرغم من عدم وجود تقريرات جازمة من هذا القبيل في قاعدة بياناتنا العينة).

والاستعلام

#snippet(```python
job($x, llist("computer", $type))
```)

يطابق جميع إدخالات الوظائف التي يكون عنصرها الثاني عبارة عن قائمة من عنصرين عنصرها الأول هو #py("\"computer\""):

#snippet(```python
job(llist("Bitdiddle", "Ben"), llist("computer", "wizard"))
job(llist("Hacker", "Alyssa", "P"), llist("computer", "programmer"))
job(llist("Fect", "Cy", "D"), llist("computer", "programmer"))
job(llist("Tweakit", "Lem", "E"), llist("computer", "technician"))
```)

وهذا النمط نفسه #emph[لا] يطابق

#snippet(```python
job(llist("Reasoner", "Louis"),
    llist("computer", "programmer", "trainee"))
```)

لأن العنصر الثاني في التقرير الجازم عبارة عن قائمة من ثلاثة عناصر، وعنصر النمط الثاني يحدد أنه يجب أن يكون هناك عنصران. وإذا أردنا تغيير النمط بحيث يمكن أن يكون العنصر الثاني أي قائمة تبدأ بـ #py("\"computer\"")، فيمكننا تحديد

#snippet(```python
job($x, pair("computer", $type))
```)

على سبيل المثال،

#snippet(```python
pair("computer", $type)
```)

يطابق البيانات

#snippet(```python
llist("computer", "programmer", "trainee")
```)

مع كون #py("$type") هي #py("llist(\"programmer\", \"trainee\")").
كما يطابق البيانات

#snippet(```python
llist("computer", "programmer")
```)

مع كون #py("$type") هي #py("llist(\"programmer\")")،
ويطابق البيانات

#snippet(```python
llist("computer")
```)

مع كون #py("$type") هي القائمة الفارغة، #py("None").

#idx("pattern")

يمكننا وصف معالجة لغة الاستعلام للاستعلامات البسيطة على النحو التالي:

- يجد النظام جميع التعيينات للمتغيرات في نمط الاستعلام التي #idx("satisfy a pattern (simple query)") #emph[تستوفي] النمط — أي جميع مجموعات القيم للمتغيرات بحيث إذا تم #idx("instantiate a pattern") #emph[تجسيد] متغيرات النمط بـ (استبدالها بـ) القيم، تكون النتيجة في قاعدة البيانات.
- يرد النظام على الاستعلام عن طريق سرد جميع تجسيدات نمط الاستعلام مع تعيينات المتغيرات التي تستوفيه.

لاحظ أنه إذا لم يكن بالنمط أي متغيرات، فإن الاستعلام يختزل إلى تحديد ما إذا كان هذا النمط موجودًا في قاعدة البيانات أم لا. وإذا كان الأمر كذلك، فإن التعيين الفارغ، الذي لا يعين أي قيم للمتغيرات، يستوفي ذلك النمط لقاعدة البيانات تلك.

#exercise(label-name: <ex:4_53>, [
اعطِ استعلامات بسيطة تسترجع المعلومات التالية من قاعدة البيانات:

+ جميع الأشخاص الذين يشرف عليهم بن بيتيدل؛
+ أسماء ووظائف جميع الأشخاص في قسم المحاسبة؛
+ أسماء وعناوين جميع الأشخاص الذين يعيشون في سلوميرفيل.
])

#idx("simple query")

#subheading([استعلامات مركبة])

#idx("compound query")

تشكل الاستعلامات البسيطة العمليات الأولية للغة الاستعلام. ولتشكيل عمليات مركبة، توفر لغة الاستعلام وسائل تركيب. وأحد الأشياء التي تجعل لغة الاستعلام لغة برمجة منطقية هو أن وسائل التركيب تعكس وسائل التركيب المستخدمة في تشكيل التعبيرات المنطقية: #py("and") و#py("or") و#py("not").

يمكننا استخدام
#idx("and (query language)")
#py("and") على النحو التالي للعثور على عناوين جميع مبرمجي الحاسوب:

#snippet(```python
and(job($person, llist("computer", "programmer")),
    address($person, $where))
```)

والمخرجات الناتجة هي

#output(```python
and(job($person, llist("computer", "programmer")),
    address($person, $where))
```)

وعمومًا،

#syntax("
and(", meta("query"), $""_(1)$, ", ", meta("query"), $""_(2)$, ", ", $dots.h$, ", ", meta("query"), $""_(n))$)

تُستوفى بواسطة جميع مجموعات القيم لمتغيرات النمط التي
#idx("satisfy a compound query")
تستوفي في آنٍ واحد #meta("query")$""_(1), dots.h ,$ #meta("query")$""_(n)$.

وكما هو الحال في الاستعلامات البسيطة، يعالج النظام استعلامًا مركبًا عن طريق إيجاد جميع التعيينات لمتغيرات النمط التي تستوفي الاستعلام، ثم عرض تجسيدات الاستعلام بهذه القيم.

طريقة أخرى لبناء استعلامات مركبة هي من خلال
#idx("or (query language)")
#py("or"). على سبيل المثال،

#snippet(```python
or(supervisor($x, llist("Bitdiddle", "Ben")),
   supervisor($x, llist("Hacker", "Alyssa", "P")))
```)

سيجد جميع الموظفين الذين يشرف عليهم بن بيتيدل أو أليسا بي هاكر:

#output(```python
or(supervisor($x, llist("Bitdiddle", "Ben")),
   supervisor($x, llist("Hacker", "Alyssa", "P")))
```)

وعمومًا،

#syntax("
or(", meta("query"), $""_(1)$, ", ", meta("query"), $""_(2)$, ", ", $dots.h$, ", ", meta("query"), $""_(n)$, ")
	  ")

تُستوفى بواسطة جميع مجموعات القيم لمتغيرات النمط التي تستوفي واحدًا على الأقل من #meta("query")$""_(1) dots.h$ #meta("query")$""_(n)$.

كما يمكن تشكيل استعلامات مركبة مع
#idx("not (query language)")
#py("not").
على سبيل المثال،

#snippet(```python
and(supervisor($x, llist("Bitdiddle", "Ben")),
    not(job($x, llist("computer", "programmer"))))
```)

يجد جميع الأشخاص الذين يشرف عليهم بن بيتيدل والذين ليسوا مبرمجي حاسوب. وعمومًا،

#syntax("
not(", meta("query"), $""_(1)$, ")
	  ")

تُستوفى بواسطة جميع التعيينات لمتغيرات النمط التي لا تستوفي #meta("query")$""_(1)$.#footnote[في الواقع، هذا الوصف لـ #py("not") صحيح فقط للحالات البسيطة. والسلوك الحقيقي لـ #py("not") أكثر تعقيدًا. وسندرس الخصائص الدقيقة لـ #py("not") في الأقسام @sec:how-query-works و @sec:math-logic.]

الشكل التركيبي النهائي يبدأ بـ
#idx("javascriptpredicate (query language)")
#py("javascript_predicate") ويحتوي على محمول #en[Python]. وعمومًا،

#syntax("
javascript_predicate(", meta("predicate"), ")
	  ")

سيُستوفى بواسطة التعيينات لمتغيرات النمط في #meta("predicate") التي يكون فيها #meta("predicate") المُجَسَّد صحيحًا.
على سبيل المثال، للعثور على جميع الأشخاص الذين يتقاضون راتبًا أكبر من 50,000 دولار يمكننا كتابة#footnote[يجب أن يدمج الاستعلام #py("javascript_predicate") فقط لإجراء عملية غير متوفرة في لغة الاستعلام. وعلى وجه الخصوص، يجب ألا يُستخدم #py("javascript_predicate") لـ
#idx("query language", sub: "equality testing in")
اختبار المساواة (نظرًا لأن المطابقة في لغة الاستعلام صُمِّمَت للقيام بذلك) أو عدم المساواة (نظرًا لأن ذلك يمكن القيام به مع قاعدة #py("same") الموضحة أدناه).]

#snippet(```python
and(salary($person, $amount), javascript_predicate($amount > 50000))
```)

#idx("satisfy a compound query")

#exercise(label-name: <ex:4_54>, [
صغ استعلامات مركبة تسترجع المعلومات التالية:

+ أسماء جميع الأشخاص الذين يشرف عليهم بن بيتيدل، جنبًا إلى جنب مع عناوينهم؛
+ جميع الأشخاص الذين يقل راتبهم عن راتب بن بيتيدل، جنبًا إلى جنب مع راتبهم وراتب بن بيتيدل؛
+ جميع الأشخاص الذين يشرف عليهم شخص ليس في قسم الحاسوب، جنبًا إلى جنب مع اسم المشرف ووظيفته.
])

#idx("compound query")

#subheading([القواعد])

#idx("rule (query language)")

بالإضافة إلى الاستعلامات البسيطة والاستعلامات المركبة، توفر لغة الاستعلام وسائل لـ
#idx("query language", sub: "abstraction in")
تجريد الاستعلامات. وهذه تُعْطَى بواسطة #emph[القواعد] (#en[rules]). والقاعدة
#idx("livesnear (rule)", decl: true)
#snippet(```python
rule(lives_near($person_1, $person_2),
     and(address($person_1, pair($town, $rest_1)),
         address($person_2, pair($town, $rest_2)),
         not(same($person_1, $person_2))))
```)

تحدد أن شخصين يعيشان بالقرب من بعضهما البعض إذا كانا يعيشان في نفس البلدة. والبند النهائي #py("not") يمنع القاعدة من القول إن جميع الأشخاص يعيشون بالقرب من أنفسهم. وعلاقة #py("same") مُعرَّفة بقاعدة بسيطة جدًا:#footnote[لاحظ أننا لا نحتاج إلى #py("same") من أجل جعل شيئين متماثلين: فنحن نستخدم نفس متغير النمط لكل منهما — وفي الواقع، لدينا شيء واحد بدلاً من شيئين في المقام الأول. على سبيل المثال، انظر #py("$town") في قاعدة #py("lives_near") و#py("$middle_manager") في قاعدة #py("wheel") أدناه. وعلاقة #py("same") مفيدة عندما نريد إجبار شيئين على أن يكونا مختلفين، مثل #py("$person_1") و#py("$person_2") في قاعدة #py("lives_near"). وعلى الرغم من أن استخدام نفس متغير النمط في جزءين من استعلام يجبر نفس القيمة على الظهور في كلا المكانين، إلا أن استخدام متغيرات نمط مختلفة لا يجبر ظهور قيم مختلفة. (قد تكون القيم المعينة لمتغيرات نمط مختلفة هي نفسها أو مختلفة).]
#idx("same (rule)", decl: true)
#snippet(```python
rule(same($x, $x))
```)

وتُصرِّح القاعدة التالية بأن الشخص يكون مديرًا كبيرًا («مدير كسير») في مؤسسة إذا كان يشرف على شخص هو بدوره مشرف:
#idx("wheel (rule)", decl: true)
#snippet(```python
rule(wheel($person),
     and(supervisor($middle_manager, $person),
         supervisor($x, $middle_manager)))
```)

والشكل العام للقاعدة هو

#syntax("
rule(", meta("conclusion"), ", ", meta("body"), ")
	  ")

حيث #meta("conclusion") هو نمط و#meta("body") هو أي استعلام.#footnote[سنسمح أيضًا بـ
#idx("rule (query language)", sub: "without body")
قواعد بدون متون، كما في #py("same")، وسنفسر مثل هذه القاعدة على أنها تعني أن نتيجة القاعدة مستوفاة بأي قيم للمتغيرات.]

يمكننا التفكير في القاعدة على أنها تمثل مجموعة كبيرة (أو حتى لانهائية) من التقريرات الجازمة، وهي جميع تجسيدات نتيجة القاعدة مع تعيينات المتغيرات التي تستوفي متن القاعدة. وعندما وصفنا الاستعلامات البسيطة (الأنماط)، قلنا إن تعيين المتغيرات يستوفي نمطًا ما إذا كان النمط المُجَسَّد في قاعدة البيانات. لكن لا يلزم أن يكون النمط صراحة في قاعدة البيانات كتقرير جازم. فقد يكون
#idx("assertion", sub: "implicit")
تقريرًا جازمًا ضمنيًا تقتضيه قاعدة ما. على سبيل المثال، الاستعلام

#snippet(```python
lives_near($x, llist("Bitdiddle", "Ben"))
```)

ينتج عنه

#output(```python
lives_near($x, llist("Bitdiddle", "Ben"))
```)

وللعثور على جميع مبرمجي الحاسوب الذين يعيشون بالقرب من بن بيتيدل، يمكننا السؤال عن

#snippet(```python
and(job($x, llist("computer", "programmer")),
    lives_near($x, llist("Bitdiddle", "Ben")))
```)

وكما هو الحال في الدوال المركبة، يمكن استخدام القواعد كأجزاء من قواعد أخرى (كما رأينا مع قاعدة #py("lives_near") أعلاه) أو حتى تُعَرَّف
#idx("recursion", sub: "in rules")
عوديًا. على سبيل المثال، القاعدة
#idx("outrankedby (rule)", decl: true)
#snippet(```python
rule(outranked_by($staff_person, $boss),
     or(supervisor($staff_person, $boss),
        and(supervisor($staff_person, $middle_manager),
            outranked_by($middle_manager, $boss))))
```)

تقول إن الموظف يفوقه رئيس في المؤسسة إذا كان الرئيس هو مشرف الموظف أو (عوديًا) إذا كان مشرف الموظف يفوقه الرئيس.

#exercise(label-name: <ex:4_55>, [
عرّف قاعدة تقول إن الشخص 1 يمكنه استبدال الشخص 2 إذا كان إما الشخص 1 يؤدي نفس وظيفة الشخص 2 وإما شخص يؤدي وظيفة الشخص 1 يمكنه أيضًا القيام بوظيفة الشخص 2، وإذا لم يكن الشخص 1 والشخص 2 هما نفس الشخص. وباستخدام قاعدتك، قدم استعلامات تجد ما يلي:

+ جميع الأشخاص الذين يمكنهم استبدال ساي دي فيكت؛
+ جميع الأشخاص الذين يمكنهم استبدال شخص يتقاضى راتبًا أكبر منهم، جنبًا إلى جنب مع الراتبين.
])

#exercise(label-name: <ex:4_56>, [
عرّف قاعدة تقول إن الشخص هو «شخصية مهمة» في قسم ما إذا كان الشخص يعمل في القسم ولكن ليس لديه مشرف يعمل في القسم نفسه.
])

#exercise(label-name: <ex:4_57>, [
فات بن بيتيدل اجتماع واحد أكثر مما ينبغي. ولخوفه من أن عادة نسيان الاجتماعات قد تكلفه وظيفته، يقرر بن فعل شيء حيال ذلك. فيضيف جميع الاجتماعات الأسبوعية للشركة إلى قاعدة بيانات #en[Gargle] عن طريق جزم ما يلي:

#snippet(```python
meeting("accounting", llist("Monday", "9am"))
meeting("administration", llist("Monday", "10am"))
meeting("computer", llist("Wednesday", "3pm"))
meeting("administration", llist("Friday", "1pm"))
```)

كل من التقريرات الجازمة أعلاه خاص باجتماع لقسم كامل. كما يضيف بن إدخالاً لاجتماع الشركة ككل الذي يضم جميع الأقسام. ويحضر جميع موظفي الشركة هذا الاجتماع:

#snippet(```python
meeting("whole-company", llist("Wednesday", "4pm"))
```)

+ صباح الجمعة، يريد بن الاستعلام عن قاعدة البيانات لجميع الاجتماعات التي تحدث في ذلك اليوم. ما الاستعلام الذي ينبغي له استخدامه؟
+ أليسا بي هاكر غير متأثرة بـ ذلك. فهي تعتقد أنه سيكون من الأجدى بكثير أن تكون قادرة على السؤال عن اجتماعاتها عن طريق تحديد اسمها. لذا تصمم قاعدة تقول إن اجتماعات الشخص تشمل جميع اجتماعات #py("\"whole-company\"") بالإضافة إلى جميع اجتماعات قسم ذلك الشخص. املأ متن قاعدة أليسا. #syntax(" rule(meeting_time(", $mono("$")$, "person, ", $mono("$")$, "day_and_time), ", meta("rule"), "-", meta("body"), ") ")
+ تصل أليسا إلى العمل صباح الأربعاء وتتساءل عن الاجتماعات التي يتعين عليها حضورها في ذلك اليوم. بعد تعريف القاعدة أعلاه، ما الاستعلام الذي ينبغي لها إجراؤه لمعرفة ذلك؟
])

#exercise(label-name: <ex:lives-near>, [
عن طريق تقديم الاستعلام
#idx("livesnear (rule)")
#snippet(```python
lives_near($person, llist("Hacker", "Alyssa", "P"))
```)

تستطيع أليسا بي هاكر العثور على الأشخاص الذين يعيشون بالقرب منها، والذين يمكنها الركوب معهم إلى العمل. من ناحية أخرى، عندما تحاول العثور على جميع أزواج الأشخاص الذين يعيشون بالقرب من بعضهم البعض عن طريق الاستعلام عن

#snippet(```python
lives_near($person_1, $person_2)
```)

تلاحظ أن كل زوج من الأشخاص الذين يعيشون بالقرب من بعضهم البعض مُدْرَج مرتين؛ على سبيل المثال،

#snippet(```python
lives_near(llist("Hacker", "Alyssa", "P"), llist("Fect", "Cy", "D"))
lives_near(llist("Fect", "Cy", "D"), llist("Hacker", "Alyssa", "P"))
```)

لماذا يحدث هذا؟ وهل هناك طريقة للعثور على قائمة الأشخاص الذين يعيشون بالقرب من بعضهم البعض، بحيث يظهر كل زوج مرة واحدة فقط؟ وضح.
])

#subheading([المنطق كبرامج])

#idx("query language", sub: "logical deductions")

يمكننا اعتبار القاعدة نوعًا من الاقتضاء المنطقي: #emph[إذا] كان تعيين القيم لمتغيرات النمط يستوفي المتن، #emph[فإنه] يستوفي النتيجة. ونتيجة لذلك، يمكننا اعتبار لغة الاستعلام ذات قدرة على إجراء #emph[استنباطات منطقية] بناءً على القواعد. وكمثال على ذلك، تأمل عملية #py("append") الموصوفة في بداية القسم @sec:logic-programming. وكما قلنا، يمكن توصيف #py("append") بالقاعدتين التاليتين:

- لأي قائمة #py("y")، تُشَكِّل القائمة الفارغة و#py("y") بتطبيق #py("append") القائمة #py("y").
- لأي #py("u") و#py("v") و#py("y") و#py("z")، يُشَكِّل #py("pair(u, v)") و#py("y") بتطبيق #py("append") القائمة #py("pair(u, z)") إذا كان #py("v") و#py("y") يُشَكِّلان بتطبيق #py("append") القائمة #py("z").

وللتعبير عن هذا في لغة الاستعلام الخاصة بنا، نعرّف قاعدتين لعلاقة

#snippet(```python
append_to_form(x, y, z)
```)

والتي يمكننا تفسيرها على أنها تعني «#py("x") و#py("y") يتوحدان بتطبيق #py("append") لتشكيل #py("z")»:
#idx("appendtoform (rules)", decl: true)
#snippet(```python
rule(append_to_form(None, $y, $y))

rule(append_to_form(pair($u, $v), $y, pair($u, $z)),
     append_to_form($v, $y, $z))
```)

القاعدة الأولى ليس لها
#idx("rule (query language)", sub: "without body")
متن، مما يعني أن النتيجة صحيحة لأي قيمة لـ #py("$y").
ولاحظ كيف تستخدم القاعدة الثانية #py("pair") لتسمية رأس وذيل قائمة ما.

وبإعطاء هاتين القاعدتين، يمكننا صياغة استعلامات تحسب حاصل #py("append") لقائمتين:

#prompt(```python
Query input:
```)

#snippet(```python
append_to_form(llist("a", "b"), llist("c", "d"), $z)
```)

#output(```python
append_to_form(llist("a", "b"), llist("c", "d"), $z)
```)

وما هو أكثر إذهالاً، يمكننا استخدام نفس القواعد لطرح السؤال «ما هي القائمة التي عند تطبيق #py("append") لها مع #py("llist(\"a\", \"b\")") تعطي #py("llist(\"a\", \"b\", \"c\", \"d\")")؟» ويتم ذلك على النحو التالي:

#prompt(```python
Query input:
```)

#snippet(```python
append_to_form(llist("a", "b"), $y, llist("a", "b", "c", "d"))
```)

#output(```python
append_to_form(llist("a", "b"), $y, llist("a", "b", "c", "d"))
```)

ويمكننا السؤال عن جميع أزواج القوائم التي تتحد بتطبيق #py("append") لتشكيل #py("llist(\"a\", \"b\", \"c\", \"d\")"):

#prompt(```python
Query input:
```)

#snippet(```python
append_to_form($x, $y, llist("a", "b", "c", "d"))
```)

#output(```python
append_to_form($x, $y, llist("a", "b", "c", "d"))
```)

قد يبدو نظام الاستعلام مظهراً لقدر كبير من الذكاء في استخدام القواعد لاستنباط الإجابات على الاستعلامات أعلاه. وفي الواقع، كما سنرى في القسم التالي، يتبع النظام خوارزمية محددة جيدا في كشف القواعد. ولسوء الحظ، على الرغم من أن النظام يعمل بشكل مثير للإعجاب في حالة #py("append")، إلا أن الطرق العامة قد تنهار في الحالات الأكثر تعقيدًا، كما سنرى في القسم @sec:math-logic.

#exercise(label-name: <ex:next-to>, [
تنظر القواعد التالية في تنفيذ علاقة #py("next_to_in") تجد العناصر المتجاورة في قائمة ما:
#idx("nexttoin (rules)", decl: true)
#snippet(```python
rule(next_to_in($x, $y, pair($x, pair($y, $u))))

rule(next_to_in($x, $y, pair($v, $z)),
     next_to_in($x, $y, $z))
```)

ماذا ستكون الاستجابة للاستعلامات التالية؟

#snippet(```python
next_to_in($x, $y, llist(1, llist(2, 3), 4))

next_to_in($x, 1, llist(2, 1, 3, 1))
```)
])

#exercise(label-name: <ex:last-pair-rules>, [
عرّف قواعد لتنفيذ عملية
#idx("lastpair", sub: "rules")
#py("last_pair")
الموجودة في التمرين @ex:last، والتي ترجع قائمة تحتوي على العنصر الأخير لقائمة غير فارغة. واختبر قواعدك على الاستعلامات التالية:

- #py("last_pair(llist(3), $x)")
- #py("last_pair(llist(1, 2, 3), $x)")
- #py("last_pair(llist(2, $x), llist(3))")

هل تعمل قواعدك بشكل صحيح على استعلامات مثل #py("last_pair($x, llist(3))")؟
])

#exercise(label-name: <ex:genesis>, [
تتتبع قاعدة البيانات التالية (انظر التكوين 4) نسب سلالة
#idx("Ada")#idx("Genesis")
عادا بالعودة إلى آدم، عن طريق قايين:

#snippet(```python
son("Adam", "Cain")
son("Cain", "Enoch")
son("Enoch", "Irad")
son("Irad", "Mehujael")
son("Mehujael", "Methushael")
son("Methushael", "Lamech")
wife("Lamech", "Ada")
son("Ada", "Jabal")
son("Ada", "Jubal")
```)

صغ قواعد مثل «إذا كان #emph[S] هو ابن #emph[F]، و#emph[F] هو ابن #emph[G]، فإن #emph[S] هو حفيد #emph[G]» و«إذا كانت #emph[W] هي زوجة #emph[M]، و#emph[S] هو ابن #emph[W]، فإن #emph[S] هو ابن #emph[M]» تمكن نظام الاستعلام من إيجاد حفيد قايين؛ وأبناء لامك؛ وأحفاد متوشائيل.
(انظر التمرين @ex:great-grandson لبعض القواعد لاستنباط علاقات أكثر تعقيدًا.)
])

#idx("rule (query language)")
#idx("query language", sub: "logical deductions")
#idx("query language")
