// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([تمثيلات الأعداد المركبة], label-name: <sec:representations-complex-numbers>)

#idx("complex numbers", sub: "rectangular vs. polar form")

سنطور نظاماً يؤدي عمليات حسابية على الأعداد المركبة كمثال بسيط ولكنه غير واقعي لبرنامج يستخدم عمليات عامة. نبدأ بمناقشة تمثيلين معقولين للأعداد المركبة كأزواج مرتبة: الشكل المستطيلي (الجزء الحقيقي والجزء التخيلي) والشكل القطبي (السعة والزاوية).#footnote[في الأنظمة الحسابية الفعلية، يُفضل الشكل المستطيلي على الشكل القطبي معظم الوقت بسبب #idx("roundoff error") أخطاء التدوير في التحويل بين الشكلين المستطيلي والقطبي. ولهذا السبب يعتبر مثال العدد المركب غير واقعي. ومع ذلك، فإنه يوفر توضيحاً واضحاً لتصميم نظام يستخدم عمليات عامة ومقدمة جيدة للأنظمة الأكثر جوهرية المراد تطويرها لاحقاً في هذا الفصل.] ووسيبين القسم @sec:manifest-types كيف يمكن جعل كلا التمثيلين يتعايشان في نظام واحد من خلال استخدام وسوم الأنواع والعمليات العامة.

مثل الأعداد الكسرية، تُمثَّل الأعداد المركبة طبيعياً كأزواج مرتبة. ويمكن التفكير في مجموعة الأعداد المركبة كمساحة ثنائية الأبعاد بحورين متعامدين، المحور "الحقيقي" والمحور "التخيلي" (انظر الشكل @fig:complex-plane). ومن وجهة النظر هذه، يمكن التفكير في العدد المركب $z=x+i y$ (حيث $i^(2) = -1$) كالنقطة في المستوى التي إحداثيها الحقيقي هو $x$ وإحداثيها التخيلي هو $y$. ويختزل جمع الأعداد المركبة في هذا التمثيل إلى جمع الإحداثيات:

$ mat(delim: #none, upright("Real-part")(z_(1)+z_(2)), =, upright("Real-part")(z_(1))+upright("Real-part")(z_(2)); upright("Imaginary-part")(z_(1) +z_(2)), =, upright("Imaginary-part")(z_(1))+upright("Imaginary-part")(z_(2))) $

عند ضرب الأعداد المركبة، يكون من الطبيعي أكثر التفكير بدلالة تمثيل العدد المركب في الشكل القطبي، كالسعة والزاوية ($r$ و $A$ في الشكل @fig:complex-plane). والجداء لعددين مركبين هو المتجه المحصول عليه بتمديد عدد مركب بطول الآخر ثم تدويره بزاوية الآخر (#idx("complex numbers", sub: "rectangular vs. polar form")):

$ mat(delim: #none, upright("Magnitude")(z_(1) dot.op z_(2)), =, upright("Magnitude")(z_(1)) dot.op upright("Magnitude")(z_(2)); upright("Angle")(z_(1) dot.op z_(2)), =, upright("Angle")(z_(1))+upright("Angle")(z_(2))) $

وبالتالي، هناك تمثيلان مختلفان للأعداد المركبة، مناسبان لعمليات مختلفة. ومع ذلك، من وجهة نظر شخص يكتب برمجية تستخدم الأعداد المركبة، يقترح مبدأ تجريد البيانات أن جميع العمليات للتلاعب بالأعداد المركبة ينبغي أن تكون متاحة بغض النظر عن التمثيل المستخدم بوساطة الحاسوب. فعلى سبيل المثال، من المفيد غالباً التمكن من العثور على سعة عدد مركب محدد بـ إحداثيات مستطيلية. وبالمثل، من المفيد غالباً التمكن من تحديد الجزء الحقيقي لعدد مركب محدد بـ إحداثيات قطبية.

#sicp-figure(image("/images/img_original/ch2-Z-G-59.svg", width: 70%), caption: [الأعداد المركبة كنقاط في المستوى.], label-name: <fig:complex-plane>)

لتصميم مثل هذا النظام، يمكننا اتباع استراتيجية #idx("data abstraction") تجريد البيانات نفسها التي اتبعناها في تصميم حزمة الأعداد الكسرية في القسم @sec:rationals. لنفترض أن العمليات على الأعداد المركبة منفذة بدلالة أربعة محددات اختيارات:
#py("real_part")،
#py("imag_part")،
#py("magnitude")،
و #py("angle"). ولنفترض أيضاً أن لدينا دالتين لبناء الأعداد المركبة:
ترجع #py("make_from_real_imag") عدداً مركباً بـ أجزاء حقيقية وتخيلية محددة، وترجع
#py("make_from_mag_ang") عدداً مركباً بـ سعة وزاوية محددتين. وتملك هذه الدوال خاصية أنه، لأي عدد مركب #py("z")، فإن كلاً من:

#snippet(```python
make_from_real_imag(real_part(z), imag_part(z))
```)

و

#snippet(```python
make_from_mag_ang(magnitude(z), angle(z))
```)

ينتجان أعداداً مركبة مساوية لـ #py("z").

باستخدام هذه البواني ومحددات الاختيارات، يمكننا تنفيذ الحساب على الأعداد المركبة باستخدام "البيانات المجردة" المحددة بوساطة البواني ومحددات الاختيارات، تماماً كما فعلنا للأعداد الكسرية في القسم @sec:rationals. وكما هو موضح في الصيغ أعلاه، يمكننا جمع وطرح الأعداد المركبة بدلالة الأجزاء الحقيقية والتخيلية بينما نضرب ونقسم الأعداد المركبة بدلالة السعات والزوايا:
#idx("addcomplex", decl: true)#idx("subcomplex", decl: true)#idx("mulcomplex", decl: true)#idx("divcomplex", decl: true)
#snippet(```python
def add_complex(z1, z2):
    return make_from_real_imag(real_part(z1) + real_part(z2),
                               imag_part(z1) + imag_part(z2))
def sub_complex(z1, z2):
    return make_from_real_imag(real_part(z1) - real_part(z2),
                               imag_part(z1) - imag_part(z2))
def mul_complex(z1, z2):
    return make_from_mag_ang(magnitude(z1) * magnitude(z2),
                             angle(z1) + angle(z2))
def div_complex(z1, z2):
    return make_from_mag_ang(magnitude(z1) / magnitude(z2),
                             angle(z1) - angle(z2))
```)

ولإكمال حزمة الأعداد المركبة، يجب أن نختار تمثيلاً ويجب أن ننفذ البواني ومحددات الاختيارات بدلالة الأعداد الأولية وبنية القوائم المترابطة الأولية. وهناك طريقان واضحان للقيام بذلك: إذ يمكننا تمثيل العدد المركب في "الشكل المستطيلي" كزوج (الجزء الحقيقي، الجزء التخيلي) أو في "الشكل القطبي" كزوج (السعة، الزاوية). فأيهما نختار؟

ولجعل الخيارات المختلفة ملموسة، تخيل أن هناك مبرمجين، #en[Ben Bitdiddle] و #en[Alyssa P. Hacker]، اللذين يصممان بشكل مستقل تمثيلات لنظام الأعداد المركبة.
يختار #en[Ben] تمثيل الأعداد المركبة في الشكل المستطيلي (#idx("complex numbers", sub: "rectangular representation")). ومع هذا الاختيار، فإن اختيار الأجزاء الحقيقية والتخيلية لعدد مركب أمر مباشر، وكما هو الحال في بناء عدد مركب بـ أجزاء حقيقية وتخيلية معطاة. ولإيجاد السعة والزاوية، أو لبناء عدد مركب بـ سعة وزاوية معطاتين، فإنه يستخدم العلاقات المثلثية:

$ mat(delim: #none, x, =, r space cos A, , r, =, sqrt(x^(2) +y^(2)); y, =, r space sin A, , A, =, arctan (y,x)) $

والتي تربط الأجزاء الحقيقية والتخيلية ($x$, $y$) بالسعة والزاوية $(r, A)$.#footnote[دالة الظل العكسي (#idx("arctangent") #idx("mathatan2 (primitive function)")) المعشار إليها هنا، والمحسوبة بوساطة دالة #en[Python] #py("math_atan2")، معرّفة بحيث تأخذ وسيطين $y$ و $x$ وترجع الزاوية التي ظلها $y/x$. وتحدد إشارات الوسيطين ربع الزاوية.]
وبالتالي يُعطى تمثيل #en[Ben] بوساطة محددات الاختيارات والبواني التالية:
#idx("realpart", sub: "rectangular representation", decl: true)#idx("imagpart", sub: "rectangular representation", decl: true)#idx("magnitude", sub: "rectangular representation", decl: true)#idx("angle", sub: "rectangular representation", decl: true)#idx("makefromrealimag", sub: "rectangular representation", decl: true)#idx("makefrommagang", sub: "rectangular representation", decl: true)
#snippet(```python
def real_part(z): return head(z)

def imag_part(z): return tail(z)

def magnitude(z):
    return math_sqrt(square(real_part(z)) + square(imag_part(z)))
def angle(z):
    return math_atan2(imag_part(z), real_part(z))
def make_from_real_imag(x, y): return pair(x, y)

def make_from_mag_ang(r, a):
    return pair(r * math_cos(a), r * math_sin(a))
```)

وعلى النقيض من ذلك، تختار #en[Alyssa] تمثيل الأعداد المركبة في الشكل القطبي (#idx("complex numbers", sub: "polar representation")).

وبالنسبة لها، فإن اختيار السعة والزاوية أمر مباشر، ولكن يتوجب عليها استخدام العلاقات المثلثية (#idx("trigonometric relations")) للحصول على الأجزاء الحقيقية والتخيلية.
وتمثيل #en[Alyssa] هو:
#idx("realpart", sub: "polar representation", decl: true)#idx("imagpart", sub: "polar representation", decl: true)#idx("magnitude", sub: "polar representation", decl: true)#idx("angle", sub: "polar representation", decl: true)#idx("makefromrealimag", sub: "polar representation", decl: true)#idx("makefrommagang", sub: "polar representation", decl: true)
#snippet(```python
def real_part(z):
    return magnitude(z) * math_cos(angle(z))
def imag_part(z):
    return magnitude(z) * math_sin(angle(z))
def magnitude(z): return head(z)

def angle(z): return tail(z)

def make_from_real_imag(x, y):
    return pair(math_sqrt(square(x) + square(y)),
                math_atan2(y, x))
def make_from_mag_ang(r, a): return pair(r, a)
```)

يضمن انضباط تجريد البيانات أن التنفيذ نفسه لـ #py("add_complex") و #py("sub_complex") و #py("mul_complex") و #py("div_complex") سيعمل إما مع تمثيل #en[Ben] وإما مع تمثيل #en[Alyssa].
