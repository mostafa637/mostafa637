// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([الذاكرة كأشعة], label-name: <sec:memory-as-vectors>)

يمكن التفكير في الذاكرة الحاسوبية التقليدية كمصفوفة من الفجوات (cubbyholes)، يمكن لكل منها أن تحتوي على قطعة من المعلومات. كل فجوة لها اسم فريد، يسمى
#idx("address")
#emph[العنوان] (#en[address]) أو
#idx("location")
#emph[الموقع] (#en[location]). توفر أنظمة الذاكرة النموذجية عمليتين أوّليتين: إحداهما تسترد البيانات المخزنة في موقع محدد والأخرى تسند بيانات جديدة إلى موقع محدد. يمكن زيادة عناوين الذاكرة لدعم الوصول التتابع لمجموعة من الفجوات. بشكل عام، تتطلب العديد من عمليات البيانات المهمة التعامل مع عناوين الذاكرة كبيانات، والتي يمكن تخزينها في مواقع الذاكرة ومعالجتها في مسجّلات الآلة. تمثيل بنية القائمة هو أحد تطبيقات
#idx("address arithmetic")
#idx("arithmetic", sub: "address arithmetic")
#emph[حساب العناوين] (#en[address arithmetic]) هذا.

لنمذجة ذاكرة الحاسوب، نستخدم نوعًا جديدًا من بنية البيانات يسمى
#idx("vector (data structure)")
#emph[شعاع] (#en[vector]). تجريديًا، الشعاع هو كائن بيانات مركب يمكن الوصول إلى عناصره الفردية عن طريق دليل صحيح (#en[integer index]) في مقدار من الوقت يستقل عن الدليل.#footnote[يمكننا تمثيل الذاكرة كقوائم من العناصر. ومع ذلك، فإن وقت الوصول لن يكون مستقلاً عن الدليل، لأن الوصول إلى العنصر الـ $n$ في القائمة يتطلب $n-1$ من عمليات #py("tail").] من أجل وصف عمليات الذاكرة، نستخدم دالتين لمعالجة الأشعة:#footnote[كما ذكرنا في القسم @sec:running-eval (الحاشية السفلية @foot:vector-array)، يدعم بايثون الأشعة كبنى بيانات ويسميها "مصفوفات". نستخدم المصطلح #emph[شعاع] في هذا الكتاب لأنه المصطلح الأكثر شيوعًا. يمكن تنفيذ دوال الأشعة أعلاه بسهولة باستخدام دعم المصفوفات الأوّلي في بايثون.]

- #py("vector_ref(")#meta("vector")#py(",")#meta("n")#py(")") #idx("vectorref (primitive function)") تُرجع العنصر الـ #meta("n") في الشعاع.
- #py("vector_set(")#meta("vector")#py(",")#meta("n")#py(",")#meta("value")#py(")") #idx("vectorset (primitive function)") تضبط العنصر الـ $n$ في الشعاع على القيمة المحددة.

على سبيل المثال، إذا كان #py("v") شعاعًا، فإن
#py("vector_ref(v, 5)")
يحصل على الإدخال الخامس في الشعاع #py("v") و
#py("vector_set(v, 5, 7)")
يغير قيمة الإدخال الخامس للشعاع #py("v") إلى 7.#footnote[من أجل الاكتمال، ينبغي أن نحدد عملية #py("make_vector") التي تبني الأشعة. ومع ذلك، في التطبيق الحالي سنستخدم الأشعة فقط لنمذجة التقسيمات الثابتة لذاكرة الحاسوب.] بالنسبة لذاكرة الحاسوب، يمكن تنفيذ هذا الوصول من خلال استخدام حساب العناوين لدمج #emph[عنوان أساسي] (#en[base address]) يحدد موقع بداية الشعاع في الذاكرة مع #emph[دليل] (#en[index]) يحدد الإزاحة لعنصر معين في الشعاع.

#subheading([تمثيل البيانات])

#idx("pair(s)", sub: "represented using vectors")
#idx("list structure", sub: "represented using vectors")

#sicp-figure(image("/images/img_javascript/Fig5.14b.std.svg", width: 70%), caption: [تمثيلات "الصندوق والسهم" وشعاع الذاكرة للقائمة #py("list(list(1, 2), 3, 4)").], label-name: <fig:box-and-pointer-memory>)

يمكننا استخدام الأشعة لتنفيذ بنى الأزواج الأساسية المطلوبة لذاكرة ذات بنية قائمة. دعنا نتخيل أن ذاكرة الحاسوب مقسمة إلى شعاعين:
#idx("theheads", sub: "vector")
#py("the_heads")
و
#idx("thetails", sub: "vector")
#py("the_tails").
سنمثل بنية القائمة كما يلي: المؤشر إلى زوج هو دليل في الشعاعين. الـ
#py("head")
للزوج هو الإدخال في
#py("the_heads")
ذو الدليل المحدد، والـ
tail
للزوج هو الإدخال في
#py("the_tails")
ذو الدليل المحدد. نحتاج أيضًا إلى تمثيل لكائنات أخرى غير الأزواج (مثل الأرقام والسلاسل النصية) وطريقة للتمييز بين نوع من البيانات وآخر. هناك طرق عديدة لتحقيق ذلك، ولكنها تختزل جميعًا إلى استخدام
#idx("typed pointer")
#idx("pointer", sub: "typed")
#emph[مؤشرات معلّمة النوع] (#en[typed pointers])، أي توسيع مفهوم "المؤشر" لتضمين معلومات حول نوع البيانات.#footnote[هذه هي بالضبط فكرة
#idx("tagged data")
#idx("data", sub: "tagged")
"البيانات المعلّمة" (#en[tagged data]) نفسها التي قدمناها في الفصل @chap:data للتعامل مع العمليات العامة. هنا، ومع ذلك، يتم تضمين أنواع البيانات عند مستوى الآلة الأوّلي بدلاً من بنائها من خلال استخدام القوائم.

قد تُفرّغ معلومات النوع بعدة طرق، اعتمادًا على تفاصيل الآلة التي سيتم تنفيذ نظام #en[Python] عليها. ستعتمد كفاءة تنفيذ برامج #en[Python] بشكل قوي على كيفية اختيار ذلك بذكاء، ولكن من الصعب صياغة قواعد تصميم عامة للاختيارات الجيدة. الطريقة الأكثر مباشرة لتنفيذ المؤشرات معلّمة النوع هي تخصيص مجموعة ثابتة من البتات في كل مؤشر لتكون
#idx("type field")
#emph[حقل نوع] (#en[type field]) يفرّغ نوع البيانات. تشمل الأسئلة المهمة التي يجب معالجتها في تصميم مثل هذا التمثيل ما يلي: كم عدد بتات النوع المطلوبة؟ ما مدى كبر أدلة الأشعة التي يجب أن تكون؟ ما مدى كفاءة استخدام تعليمات الآلة الأوّلية لمعالجة حقول النوع للمؤشرات؟ يُقال عن الآلات التي تتضمن أجهزة خاصة للتعامل الكفء مع حقول النوع إنها تملك
#idx("tagged architecture")
#emph[بنى معلّمة] (#en[tagged architectures]).] يُمكّن نوع البيانات النظام من التمييز بين مؤشر إلى زوج (والذي يتكون من نوع البيانات "زوج" ودليل في أشعة الذاكرة) والمؤشرات إلى أنواع أخرى من البيانات (والتي تتكون من نوع بيانات آخر وأي شيء يُستخدم لتمثيل البيانات من ذلك النوع). يُعتبر كائنا بيانات أنهما متطابقان
(#py("==="))
إذا كانت مؤشراتهما متطابقة.
يُوضح الشكل @fig:box-and-pointer-memory استخدام هذه الطريقة لتمثيل
#py("list(list(1, 2), 3, 4)")،
والذي يظهر أيضًا مخطط الصندوق والسهم الخاص به. نستخدم بادئات الأحرف للإشارة إلى معلومات نوع البيانات. وبالتالي، فإن المؤشر إلى الزوج ذي الدليل 5 يُشار إليه بـ #py("p5")، والقائمة الفارغة يُشار إليها بالمؤشر #py("e0")، والمؤشر إلى الرقم 4 يُشار إليه بـ #py("n4"). في مخطط الصندوق والسهم، أشرنا في الأسفل على اليسار من كل زوج إلى دليل الشعاع الذي يحدد أين تُمكث
#py("head")
و
#py("tail")
للزوج. قد تحتوي المواقع الفارغة في
#py("the_heads")
و
#py("the_tails")
على أجزاء من بنى قوائم أخرى (ليست ذات اهتمام هنا).

قد يتكون المؤشر إلى رقم، مثل #py("n4")، من نوع يشير إلى بيانات عددية بالإضافة إلى التمثيل الفعلي للرقم 4.#footnote[هذا القرار بشأن تمثيل الأرقام يحدد ما إذا كان
#idx("===", sub: "as numeric equality operator")
#idx("equality", sub: "of numbers")
#idx("number(s)", sub: "equality of")
#py("===")،
الذي يختبر المساواة في المؤشرات، يمكن استخدامه لاختبار المساواة في الأرقام. إذا كان المؤشر يحتوي على الرقم نفسه، فإن الأرقام المتساوية سيكون لها نفس المؤشر. ولكن إذا كان المؤشر يحتوي على دليل موقع يتم فيه تخزين الرقم، فإن الأرقام المتساوية ستضمن الحصول على مؤشرات متساوية فقط إذا حرصنا على عدم تخزين نفس الرقم في أكثر من موقع واحد.]
للتعامل مع الأرقام الكبيرة جدًا بحيث لا يمكن تمثيلها في المقدار الثابت من المساحة المخصصة لمؤشر واحد، يمكننا استخدام نوع بيانات
#idx("bignum")
#idx("number(s)", sub: "bignum")
#emph[عدد ضخم] (#en[bignum]) متميز، يحدد له المؤشر قائمة يتم فيها تخزين أجزاء الرقم.#footnote[هذا تمامًا مثل كتابة رقم كـ تسلسل من الأرقام، باستثناء أن كل "رقم" هو رقم بين 0 وأكبر رقم يمكن تخزينه في مؤشر واحد.]

السلسلة النصية
#idx("string(s)", sub: "representation of")
قد تُمثَّل كمؤشر معلّم النوع يحدد تسلسلاً من الأحرف التي تشكل التمثيل المطبوع للسلسلة. يبني المحلل مثل هذا التسلسل عندما يواجه سلسلة نصية صريحة، والمُعامل التجميعي للسلاسل النصية #py("+") والدوال الأوّلية المنتجة للسلاسل مثل
#py("stringify")
تبني مثل هذا التسلسل.
نظرًا لأننا نريد أن يتم التعرف على حالتين من السلسلة النصية كـ "نفس" السلسلة بواسطة
#py("===") ونريد أن تكون
#idx("===", sub: "as string comparison operator")
#idx("equality", sub: "of strings")
#py("===")
اختبارًا بسيطًا لمساواة المؤشرات، يجب أن نضمن أنه إذا رأى النظام نفس السلسلة مرتين، فإنه سيستخدم نفس المؤشر (إلى نفس تسلسل الأحرف) لتمثيل كلا الوقوعين. لتحقيق ذلك، يحتفظ النظام بجدول يسمى
#idx("string pool")
#emph[مجمع السلاسل النصية] (#en[string pool])،
لجميع السلاسل التي واجهها على الإطلاق. عندما يكون النظام على وشك بناء سلسلة، فإنه يفحص مجمع السلاسل لمعرفة ما إذا كان قد رأى نفس السلسلة من قبل. إذا لم يكن قد رأى ذلك، فإنه يبني سلسلة جديدة (مؤشر معلّم النوع إلى تسلسل أحرف جديد) ويدخل هذا المؤشر في مجمع السلاسل. إذا كان النظام قد رأى السلسلة من قبل، فإنه يُرجع مؤشر السلسلة المخزن في مجمع السلاسل. تسمى عملية استبدال السلاسل بمؤشرات فريدة بـ
#idx("interning strings")
#idx("string(s)", sub: "interning")
#emph[تأطير السلاسل النصية] (#en[string interning]).

#subheading([تنفيذ عمليات القائمة الأوّلية])

#anchor(<sec:impl-list-ops>)

بالنظر إلى مخطط التمثيل أعلاه، يمكننا استبدال كل عملية قائمة "أوّلية" لآلة مسجّلات بعملية شعاع أوّلية واحدة أو أكثر. سنستخدم مسجّلين،
#idx("theheads", sub: "register")
#py("the_heads")
و
#idx("thetails", sub: "register")
#py("the_tails")،
لتحديد أشعة الذاكرة، وسنفترض أن
#py("vector_ref")
و
#py("vector_set")
متاحان كـ عمليات أوّلية. نفترض أيضًا أن العمليات العددية على المؤشرات (مثل زيادة مؤشر، أو استخدام مؤشر زوج لدليل شعاع، أو إضافة عددين) تستخدم فقط جزء الدليل من المؤشر معلّم النوع.

على سبيل المثال، يمكننا جعل آلة المسجّلات تدعم التعليمات
#idx("head (primitive function)", sub: "implemented with vectors")#idx("tail (primitive function)", sub: "implemented with vectors")
#syntax("
assign(", meta("reg"), $""_(1)$, ", list(op(\"head\"), reg(", meta("reg"), $""_(2)$, ")))

assign(", meta("reg"), $""_(1)$, ", list(op(\"tail\"), reg(", meta("reg"), $""_(2)$, ")))
      ")

إذا قمنا بتنفيذ هذه، على التوالي، كـ:

#syntax("
assign(", meta("reg"), $""_(1)$, ", list(op(\"vector_ref\"), reg(\"the_heads\"), reg(", meta("reg"), $""_(2)$, ")))

assign(", meta("reg"), $""_(1)$, ", list(op(\"vector_ref\"), reg(\"the_tails\"), reg(", meta("reg"), $""_(2)$, ")))
      ")

التعليمات:
#idx("sethead (primitive function)", sub: "implemented with vectors")#idx("settail (primitive function)", sub: "implemented with vectors")
#syntax("
perform(list(op(\"set_head\"), reg(", meta("reg"), $""_(1)$, "), reg(", meta("reg"), $""_(2)$, ")))

perform(list(op(\"set_tail\"), reg(", meta("reg"), $""_(1)$, "), reg(", meta("reg"), $""_(2)$, ")))
      ")

تُنفذ كـ:

#syntax("
perform(list(op(\"vector_set\"), reg(\"the_heads\"), reg(", meta("reg"), $""_(1)$, "), reg(", meta("reg"), $""_(2)$, ")))

perform(list(op(\"vector_set\"), reg(\"the_tails\"), reg(", meta("reg"), $""_(1)$, "), reg(", meta("reg"), $""_(2)$, ")))
      ")

يتم إجراء العملية #idx("pair (primitive function)", sub: "implemented with vectors") #py("pair") عن طريق تخصيص دليل غير مستخدم وتخزين وسائط #py("pair") في #py("the_heads") و #py("the_tails") عند موقع الشعاع ذي الدليل المذكور. نفترض أن هناك مسجّلاً خاصًا،
#idx("free register")
#py("free")، يحمل دائمًا مؤشر زوج يحتوي على الدليل المتاح التالي، وأنه يمكننا زيادة جزء الدليل لهذا المؤشر للعثور على الموقع الحر التالي.#footnote[هناك طرق أخرى للعثور على التخزين الحر. على سبيل المثال، يمكننا ربط جميع الأزواج غير المستخدمة معًا في
#idx("free list")
#emph[قائمة حرة] (#en[free list]). مواقعنا الحر متتالية (وبالتالي يمكن الوصول إليها عن طريق زيادة مؤشر) لأننا نستخدم مُجمّع مهملات مُضغِّطًا، كما سنرى في القسم @sec:gc.]
على سبيل المثال، التعليمة:

#syntax("
assign(", meta("reg"), $""_(1)$, ", list(op(\"pair\"), reg(", meta("reg"), $""_(2)$, "), reg(", meta("reg"), $""_(3)$, ")))
      ")

تُنفذ كـ التسلسل التالي من عمليات الأشعة:#footnote[هذا بالأساس هو تنفيذ #py("pair") بدلالة #py("set_head") و #py("set_tail")، كما هو موضح في القسم @sec:mutable-list-structure. العملية #py("get_new_pair") المستخدمة في ذلك التنفيذ تتحقق هنا بواسطة المؤشر #py("free").]

#syntax("
perform(list(op(\"vector_set\"),
             reg(\"the_heads\"), reg(\"free\"), reg(", meta("reg"), $""_(2)$, "))),
perform(list(op(\"vector_set\"),
             reg(\"the_tails\"), reg(\"free\"), reg(", meta("reg"), $""_(3)$, "))),
assign(", meta("reg"), $""_(1)$, ", reg(\"free\")),
assign(\"free\", list(op(\"+\"), reg(\"free\"), constant(1)))
      ")

عملية #py("===")

#syntax("
list(op(\"===\"), reg(", meta("reg"), $""_(1)$, "), reg(", meta("reg"), $""_(2)$, "))
      ")

تختبر ببساطة المساواة في جميع الحقول في المسجّلات، والمحمولات مثل
#idx("ispair (primitive function)", sub: "implemented with typed pointers")
#py("is_pair")،
#idx("isnull (primitive function)", sub: "implemented with typed pointers")
#py("is_null")،
#idx("isstring (primitive function)", sub: "implemented with typed pointers")
#py("is_string")،
و
#idx("isnumber (primitive function)", sub: "implemented with typed pointers")
#py("is_number")
تحتاج فقط إلى فحص حقل النوع.

#subheading([تنفيذ المكدسات])

#idx("stack", sub: "representing")

على الرغم من أن آلات المسجّلات الخاصة بنا تستخدم المكدسات، إلا أننا لا نحتاج إلى القيام بأي شيء خاص هنا، نظرًا لأنه يمكن نمذجة المكدسات بدلالة القوائم. يمكن أن يكون المكدس قائمة بالقيم المحفوظة، يشير إليها مسجّل خاص
#py("the_stack").
وبالتالي، يمكن تنفيذ #py("save(")#meta("reg")#py(")") كـ:
#idx("save (in register machine)", sub: "implementing")
#syntax("
assign(\"the_stack\", list(op(\"pair\"), reg(", meta("reg"), "), reg(\"the_stack\")))
      ")

وبالمثل، يمكن تنفيذ #py("restore(")#meta("reg")#py(")") كـ:
#idx("restore (in register machine)", sub: "implementing")
#syntax("
assign(", meta("reg"), ", list(op(\"head\"), reg(\"the_stack\")))
assign(\"the_stack\", list(op(\"tail\"), reg(\"the_stack\")))
      ")

ويمكن تنفيذ #py("perform(list(op(\"initialize_stack\")))") كـ:

#syntax("
assign(\"the_stack\", constant(null))
      ")

يمكن توسيع هذه العمليات بشكل أكبر بدلالة عمليات الأشعة المعطاة أعلاه. ومع ذلك، في بنى الحواسيب التقليدية، يكون من المزايا عادةً تخصيص المكدس كشُعاع منفصل. عندئذٍ، يمكن تحقيق الدفع والاستعادة من المكدس عن طريق زيادة أو إنقاص دليل في ذلك الشعاع.

#exercise(label-name: <ex:5_19>, [
ارسم تمثيل الصندوق والسهم وتمثيل شعاع الذاكرة (كما في الشكل @fig:box-and-pointer-memory) لبنية القائمة الناتجة عن:

#snippet(```python
const x = pair(1, 2);
const y = list(x, x);
```)

مع كون المؤشر #py("free") في البداية #py("p1"). ما هي القيمة النهائية لـ #py("free")$thin$؟ ما هي المؤشرات التي تمثل قيم #py("x") و #py("y")؟
])

#exercise(label-name: <ex:count-leaves-machine>, [
نفذ آلات مسجّلات للدوال التالية #idx("countleaves", sub: "as register machine").
افترض أن عمليات ذاكرة بنية القائمة متاحة كـ أوّليات آلة.

+ العودية #py("count_leaves"): #snippet(```python function count_leaves(tree) { return is_null(tree) ? 0 : ! is_pair(tree) ? 1 : count_leaves(head(tree)) + count_leaves(tail(tree)); } ```)
+ العودية #py("count_leaves") مع عدّاد صريح: #snippet(```python function count_leaves(tree) { function count_iter(tree, n) { return is_null(tree) ? n : ! is_pair(tree) ? n + 1 : count_iter(tail(tree), count_iter(head(tree), n)); } return count_iter(tree, 0); } ```)
])

#exercise(label-name: <ex:5_21>, [
قدم التمرين @ex:append من القسم @sec:mutable-list-structure دالة #py("append") تضيف قائمتين لتشكيل قائمة جديدة ودالة #py("append_mutator") تربط قائمتين معًا. صمم آلة مسجّلات لتنفيذ كل من هذه الدوال #idx("append", sub: "as register machine") #idx("appendmutator", sub: "as register machine").
افترض أن عمليات ذاكرة بنية القائمة متاحة كـ عمليات أوّلية.
])

#idx("pair(s)", sub: "represented using vectors")
#idx("list structure", sub: "represented using vectors")
