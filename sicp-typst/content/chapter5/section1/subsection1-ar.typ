// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([لغة لوصف آلات المسجّلات], label-name: <sec:register-machine-language>)

#idx("register machine", sub: "language for describing")

تعتبر مخططات مسار البيانات ووحدة التحكم كافية لتمثيل الآلات البسيطة مثل GCD، ولكنها غير عملية لوصف الآلات الكبيرة مثل مُفسِّر (interpreter) بايثون. لجعل من الممكن التعامل مع الآلات المعقدة، سنقوم بإنشاء لغة تعرض، في شكل نصي، جميع المعلومات المقدمة بواسطة مخططات مسار البيانات ووحدة التحكم. سنبدأ بتدوين (notation) يعكس المخططات بشكل مباشر.

نُعرّف مسارات البيانات لآلة ما عن طريق وصف المسجّلات والعمليات. لوصف مسجّل، نمنحه اسمًا ونحدد الأزرار التي تتحكم في الإسناد إليه. نعطي كل زر من هذه الأزرار اسمًا ونحدد مصدر البيانات التي تدخل المسجّل تحت تحكم الزر. (المصدر هو مسجّل، أو ثابت، أو عملية). لوصف عملية، نمنحها اسمًا ونحدد مدخلاتها (مسجّلات أو ثوابت).

نُعرّف وحدة التحكم لآلة ما على أنها تسلسل من #idx("register-machine language", sub: "instructions") #emph[التعليمات (instructions)] مع #idx("register-machine language", sub: "label") #emph[تسميات (labels)] تحدد #idx("register-machine language", sub: "entry point") #emph[نقاط الإدخال (entry points)] في التسلسل. التعليمة هي واحدة مما يلي:

- اسم زر مسار البيانات الذي يجب الضغط عليه لإسناد قيمة لمسجّل. (يتوافق هذا مع مربع في مخطط وحدة التحكم.)
- تعليمة #idx("test (in register machine)") #idx("register-machine language", sub: "test") #py("test")، والتي تنفذ اختبارًا محددًا.
- تفريع شرطي (conditional branch) #idx("register-machine language", sub: "branch") #idx("branch (in register machine)") #idx("register-machine language", sub: "label") #idx("label (in register machine)") (تعليمة #py("branch")) إلى موقع مُشار إليه بواسطة تسمية وحدة تحكم، بناءً على نتيجة الاختبار السابق. (يتوافق الاختبار والتفريع معًا مع معين في مخطط وحدة التحكم.) إذا كان الاختبار خاطئًا، يجب أن تستمر وحدة التحكم مع التعليمة التالية في التسلسل. بخلاف ذلك، يجب أن تستمر وحدة التحكم مع التعليمة التي تلي التسمية.
- تفريع غير شرطي (unconditional branch) #idx("register-machine language", sub: "goto") #idx("goto (in register machine)") (تعليمة #py("go_to")) تُسمّي تسمية وحدة تحكم لاستئناف التنفيذ عندها.

#sicp-figure(image("/images/img_original/Fig5.2.std.svg", width: 70%), caption: [وحدة التحكم لآلة GCD.], label-name: <fig:gcd-controller>)

تبدأ الآلة في بداية تسلسل تعليمات وحدة التحكم وتتوقف عندما يصل التنفيذ إلى نهاية التسلسل. باستثناء عندما يُغيّر التفريع تدفق التحكم، يتم تنفيذ التعليمات بالترتيب الذي تم سردها به.

يوضح الشكل @fig:gcd-machine-spec آلة GCD الموصوفة بهذه الطريقة. هذا المثال يلمح فقط إلى عمومية هذه الأوصاف، نظرًا لأن آلة GCD حالة بسيطة جدًا: يحتوي كل مسجّل على زر واحد فقط، ويتم استخدام كل زر واختبار مرة واحدة فقط في وحدة التحكم.

لسوء الحظ، من الصعب قراءة مثل هذا الوصف. لكي نفهم تعليمات وحدة التحكم، يجب علينا الرجوع باستمرار إلى تعريفات أسماء الأزرار وأسماء العمليات، ولكي نفهم ما تفعله الأزرار، قد نضطر إلى الرجوع إلى تعريفات أسماء العمليات. وبالتالي، سنقوم بتحويل تدويننا لدمج المعلومات من أوصاف مسار البيانات ووحدة التحكم حتى نراها كلها معًا.

للحصول على هذا الشكل من الوصف، سنستبدل أسماء الأزرار والعمليات العشوائية بتعريفات سلوكها. أي أنه بدلاً من القول (في وحدة التحكم) "اضغط على الزر #py("t<-r")" والقول بشكل منفصل (في مسارات البيانات) "يسند الزر #py("t<-r") قيمة عملية #py("rem") إلى المسجّل #py("t")" و"مدخلات عملية #py("rem") هي محتويات المسجّلين #idx("register-machine language", sub: "assign") #idx("assign (in register machine)") #idx("register-machine language", sub: "op") #idx("op (in register machine)") #idx("register-machine language", sub: "reg") #idx("reg (in register machine)") #py("a") و #py("b")"، سنقول (في وحدة التحكم) "اضغط على الزر الذي يُسند للمسجّل #py("t") قيمة عملية #py("rem") على محتويات المسجّلين #py("a") و #py("b")."
بالمثل، بدلاً من القول (في وحدة التحكم) "قم بتنفيذ اختبار #py("=")" والقول بشكل منفصل (في مسارات البيانات) "يعمل اختبار #py("=") على محتويات المسجّل #py("b") والثابت 0"، سنقول "قم بتنفيذ اختبار #py("=") على #idx("register-machine language", sub: "constant") #idx("constant (in register machine)") محتويات المسجّل #py("b") والثابت 0." سنحذف وصف مسار البيانات، مع ترك تسلسل وحدة التحكم فقط. وبالتالي، يتم وصف آلة GCD على النحو التالي:

#snippet(```python
controller(
  llist(
    "test_b",
      test(llist(op("="), reg("b"), constant(0))),
      branch(label("gcd_done")),
      assign("t", llist(op("rem"), reg("a"), reg("b"))),
      assign("a", reg("b")),
      assign("b", reg("t")),
      go_to(label("test_b")),
    "gcd_done"))
```)

هذا الشكل من الوصف أسهل في القراءة من النوع الموضح في الشكل @fig:gcd-machine-spec، ولكنه يحتوي أيضًا على عيوب:

- إنه أكثر إسهابًا للآلات الكبيرة، حيث تتكرر الأوصاف الكاملة لعناصر مسار البيانات كلما تم ذكر العناصر في تسلسل تعليمات وحدة التحكم. (لا تمثل هذه مشكلة في مثال GCD، لأنه يتم استخدام كل عملية وزر مرة واحدة فقط.) علاوة على ذلك، فإن تكرار أوصاف مسار البيانات يحجب البنية الفعلية لمسار البيانات للآلة؛ ليس من الواضح بالنسبة لآلة كبيرة عدد المسجّلات والعمليات والأزرار الموجودة وكيفية ترابطها.
- لأن تعليمات وحدة التحكم في تعريف الآلة تبدو وكأنها تعبيرات بايثون، فمن السهل أن ننسى أنها ليست تعبيرات بايثون تعسفية. إنها يمكن أن تدون فقط عمليات الآلة القانونية. على سبيل المثال، يمكن للعمليات أن تعمل بشكل مباشر فقط على الثوابت ومحتويات المسجّلات، وليس على نتائج العمليات الأخرى.

على الرغم من هذه العيوب، سنستخدم لغة آلة المسجّلات هذه في جميع أنحاء هذا الفصل، لأننا سنكون أكثر اهتمامًا بفهم وحدات التحكم من فهم العناصر والوصلات في مسارات البيانات. ومع ذلك، يجب أن نضع في اعتبارنا أن تصميم مسار البيانات أمر بالغ الأهمية في تصميم الآلات الحقيقية.

#exercise(label-name: <ex:iterative-fact-2>, [
استخدم لغة آلة المسجّلات لوصف #idx("factorial", sub: "register machine for (iterative)") آلة المضروب التكرارية في التمرين @ex:iterative-fact.
])

#subheading([أفعال (Actions)])

#idx("actions, in register machine")
#idx("register machine", sub: "actions")

دعونا نعدل آلة GCD بحيث يمكننا كتابة الأرقام التي نريد إيجاد القاسم المشترك الأكبر لها وطباعة الإجابة.
لن نناقش كيفية صنع آلة يمكنها القراءة والطباعة، ولكننا سنفترض (كما نفعل عندما نستخدم #py("prompt") و #py("display") في بايثون) أنها متوفرة كعمليات أوّلية.#footnote[يخفي هذا الافتراض قدرًا كبيرًا من التعقيد. يتطلب تنفيذ القراءة والطباعة جهدًا كبيرًا، على سبيل المثال للتعامل مع ترميزات الأحرف للغات المختلفة.]

عملية #idx("prompt operation in register machine") #py("prompt") تشبه العمليات التي استخدمناها في أنها تُنتج قيمة يمكن تخزينها في مسجّل. لكن #py("prompt") لا تأخذ مدخلات من أي مسجّلات؛ تعتمد قيمتها على شيء يحدث خارج أجزاء الآلة التي نصممها. سنسمح لعمليات آلتنا بأن يكون لها مثل هذا السلوك، وبالتالي سنقوم برسم وتدوين استخدام #py("prompt") تمامًا كما نفعل مع أي عملية أخرى تحسب قيمة.

#sicp-figure([#snippet(```python
data_paths(
  registers(
    llist(
      pair(name("a"),
           buttons(name("a<-b"), source(register("b")))),
      pair(name("b"),
           buttons(name("b<-t"), source(register("t")))),
      pair(name("t"),
           buttons(name("t<-r"), source(operation("rem")))))),
  operations(
    llist(
      pair(name("rem"),
           inputs(register("a"), register("b"))),
      pair(name("="),
           inputs(register("b"), constant(0))))))

controller(
  llist(
    "test_b",                     # label
      test("="),                  # test
      branch(label("gcd_done")),  # conditional branch
      "t<-r",                     # button push
      "a<-b",                     # button push
      "b<-t",                     # button push
      go_to(label("test_b")),     # unconditional branch
    "gcd_done"))                  # label
```)], caption: [مواصفات لآلة GCD.], label-name: <fig:gcd-machine-spec>)

من ناحية أخرى، تختلف عملية #idx("display operation in register machine") #py("display") عن العمليات التي استخدمناها بطريقة أساسية: فهي لا تُنتج قيمة إخراج لتخزينها في مسجّل. على الرغم من أن لها تأثيرًا، إلا أن هذا التأثير لا ينصب على جزء من الآلة التي نصممها. سنشير إلى هذا النوع من العمليات على أنه #emph[فعل (action)]. سنمثل الفعل في مخطط مسار البيانات تمامًا كما نمثل عملية تحسب قيمة — كشبه منحرف يحتوي على اسم الفعل. تشير الأسهم إلى مربع الفعل من أي مدخلات (مسجّلات أو ثوابت). كما نربط زرًا بالفعل. الضغط على الزر يجعل الفعل يحدث. لجعل وحدة التحكم تضغط على زر فعل نستخدم نوعًا جديدًا من التعليمات يسمى #idx("register-machine language", sub: "perform") #idx("perform (in register machine)") #py("perform"). وبالتالي، يتم تمثيل فعل طباعة محتويات المسجّل #py("a") في تسلسل وحدة التحكم بواسطة التعليمة:

#snippet(```python
perform(llist(op("display"), reg("a")))
```)

يوضح الشكل @fig:gcd-with-io مسارات البيانات ووحدة التحكم لآلة GCD الجديدة. بدلاً من إيقاف الآلة بعد طباعة الإجابة، جعلناها تبدأ من جديد، بحيث تقرأ بشكل متكرر زوجًا من الأرقام، وتحسب القاسم المشترك الأكبر لها، وتطبع النتيجة.
هذه البنية تشبه حلقات التشغيل (driver loops) التي استخدمناها في مُفسِّرات الفصل @chap:meta.

#sicp-figure(stack(dir: ttb, spacing: 1em, image("/images/img_javascript/Fig5.4c.std.svg", width: 70%), [#snippet(```python
controller(
  llist(
    "gcd_loop",
      assign("a", llist(op("prompt"))),
      assign("b", llist(op("prompt"))),
    "test_b",
      test(llist(op("="), reg("b"), constant(0))),
      branch(label("gcd_done")),
      assign("t", llist(op("rem"), reg("a"), reg("b"))),
      assign("a", reg("b")),
      assign("b", reg("t")),
      go_to(label("test_b")),
    "gcd_done",
      perform(llist(op("display"), reg("a"))),
      go_to(label("gcd_loop"))))
```)]), caption: [آلة GCD التي تقرأ المدخلات وتطبع النتائج.], label-name: <fig:gcd-with-io>)

#idx("register machine", sub: "language for describing")
#idx("actions, in register machine")
#idx("register machine", sub: "actions")
