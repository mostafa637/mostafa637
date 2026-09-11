// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#section([مُقيِّم التحكم الصريح], label-name: <sec:eceval>)

#sicp-figure(image("/images/img_original/chip.png", width: 40%), caption: [تنفيذ شريحة سيليكون لمُقيِّم لغة مخطط (Scheme).], label-name: <fig:Scheme-chip>)

#idx("explicit-control evaluator for Python")

في القسم @sec:designing-register-machines رأينا كيفية تحويل برامج #en[Python] البسيطة إلى أوصاف لآلات مسجّلات. سنجري الآن هذا التحويل على برنامج أكثر تعقيدًا، وهو المُقيِّم فوق الدائري (#en[metacircular]) في الأقسام من @sec:core-of-evaluator إلى @sec:running-eval، والذي يوضح كيفية وصف سلوك مُفسِّر #en[Python] بدلالة الدالتين #py("evaluate") و #py("apply").
#emph[مُقيِّم التحكم الصريح] (#en[explicit-control evaluator]) الذي نطبقه في هذا القسم يوضح كيفية وصف آليات استدعاء الدوال وتمرير الوسائط الأساسية المستخدمة في عملية التقييم بدلالة العمليات على المسجّلات والمكدسات. بالإضافة إلى ذلك، يمكن لمُقيِّم التحكم الصريح أن يخدم كتنفيذ لمُفسِّر #en[Python]، مكتوب بلغة تشبه إلى حد كبير لغة الآلة الأصلية للحواسيب التقليدية. يمكن تنفيذ المُقيِّم بواسطة محاكي آلة المسجّلات في القسم @sec:simulator. بدلاً من ذلك، يمكن استخدامه كنقطة بداية لبناء تنفيذ بلغة الآلة لمُقيِّم #en[Python]، أو حتى آلة خاصة الأغراض
#idx("Scheme chip")
#idx("integrated-circuit implementation of Scheme")
#idx("chip implementation of Scheme")
#idx("Scheme", sub: "integrated-circuit implementation of")
لتقييم برامج #en[Python].
يُظهر الشكل @fig:Scheme-chip تنفيذًا عتاديًا كشريحة سيليكون تعمل كمُقيِّم لـ Scheme، وهي اللغة المستخدمة بدلاً من بايثون في الطبعة الأصلية من هذا الكتاب. بدأ مصممو الشريحة بمواصفات مسار البيانات ووحدة التحكم لآلة مسجّلات تشبه المُقيِّم الموصوف في هذا القسم واستخدموا برامج أتمتة التصميم لبناء تخطيط الدائرة المدمجة.#footnote[انظر
#idx("Batali, John Dean")
باتالي وآخرين 1982 لمزيد من المعلومات حول الشريحة والطريقة التي تم تصميمها بها.]

#subheading([المسجّلات والعمليات])

#idx("explicit-control evaluator for Python", sub: "data paths")
#idx("explicit-control evaluator for Python", sub: "operations")

في تصميم مُقيِّم التحكم الصريح، يجب علينا تحديد العمليات المراد استخدامها في آلة المسجّلات الخاصة بنا. وصفنا المُقيِّم فوق الدائري بدلالة البناء التجريدي، باستخدام دوال مثل #py("is_literal") و #py("make_function"). في تنفيذ آلة المسجّلات، يمكننا توسيع هذه الدوال إلى تسلسلات من عمليات ذاكرة بنية القائمة الأوّلية، وتنفيذ هذه العمليات على آلة المسجّلات الخاصة بنا. ومع ذلك، فإن هذا سيجعل مُقيِّمنا طويلاً جداً، مما يطمس البنية الأساسية بالتفاصيل. لتوضيح العرض، سنضمّن كعمليات أوّلية لآلة المسجّلات دوال البناء المعطاة في القسم @sec:representing-expressions والدوال لتمثيل البيئات وبيانات وقت التشغيل الأخرى المعطاة في القسمين @sec:eval-data-structures و @sec:running-eval.
من أجل تحديد مُقيِّم بالكامل يمكن برمجته بلغة آلة منخفضة المستوى أو تنفيذه في العتاد، سنستبدل هذه العمليات بعمليات أكثر أوّلية، باستخدام تنفيذ بنية القائمة الموصوف في القسم @sec:storage-allocation.

تتضمن آلة مسجّلات مُقيِّم #en[Python] الخاصة بنا مكدسًا وثمانية مسجّلات:
#idx("explicit-control evaluator for Python", sub: "registers")
#idx("comp register")
#py("comp")،
#idx("env register")
#py("env")،
#idx("val register")
#py("val")،
#idx("continue register", sub: "in explicit-control evaluator")
#py("continue")،
#idx("fun register")
#py("fun")،
#idx("argl register")
#py("argl")، و
#idx("unev register")
#py("unev").
يُستخدم المسجّل #py("comp") لحفظ المكون (#en[component]) المراد تقييمه، ويحتوي #py("env") على البيئة التي سيتم التقييم فيها. في نهاية التقييم، يحتوي #py("val") على القيمة التي تم الحصول عليها بتقييم المكون في البيئة المحددة. يُستخدم المسجّل #py("continue") لتنفيذ العودية، كما هو موضح في القسم @sec:stack-recursion. (يحتاج المُقيِّم إلى استدعاء نفسه عوديًا، لأن تقييم مكون يتطلب تقييم مكوناته الفرعية). تُستخدم المسجّلات #py("fun") و #py("argl") و #py("unev") في تقييم تطبيقات الدوال.

لن نقدم مخطط مسار بيانات لإظهار كيفية توصيل المسجّلات والعمليات للمُقيِّم، ولن نقدم القائمة الكاملة لعمليات الآلة. هذه ضمنية في وحدة التحكم للمُقيِّم، والتي سيتم تقديمها بالتفصيل.

#idx("explicit-control evaluator for Python", sub: "data paths")

#include "../../chapter5/section4/subsection1-ar.typ"

#include "../../chapter5/section4/subsection2-ar.typ"

#include "../../chapter5/section4/subsection3-ar.typ"

#include "../../chapter5/section4/subsection4-ar.typ"
