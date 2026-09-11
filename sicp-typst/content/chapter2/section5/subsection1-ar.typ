// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([العمليات الحسابية العامة], label-name: <sec:generic-arithmetic-operators>)

#idx("generic arithmetic operations")

تشبه مهمة تصميم عمليات حسابية عامة مهمة تصميم عمليات الأعداد المركبة العامة. ونود، على سبيل المثال، أن تكون لدينا دالة جمع عامة #py("add") تتصرف مثل الجمع الأولي العادي #py("+") على الأعداد العادية، ومثل #py("add_rat") على الأعداد الكسرية، ومثل #py("add_complex") على الأعداد المركبة. ويمكننا تنفيذ #py("add")، والعمليات الحسابية العامة الأخرى، باتباع استراتيجية التوجيه بالبيانات نفسها التي استخدمناها في القسم @sec:data-directed لتنفيذ محددات الاختيارات العامة للأعداد المركبة. وسوف نرفق وسم نوع بكل نوع من الأعداد ونجعل الدالة العامة ترسل الإرسال إلى حزمة مناسبة وفقًا لنوع بيانات وسائطها.

وتُعرَّف الدوال الحسابية العامة كما يلي:
#idx("add (generic)", decl: true)#idx("sub (generic)", decl: true)#idx("mul (generic)", decl: true)#idx("div (generic)", decl: true)
#snippet(```python
def add(x, y): return apply_generic("add", llist(x, y))

def sub(x, y): return apply_generic("sub", llist(x, y))

def mul(x, y): return apply_generic("mul", llist(x, y))

def div(x, y): return apply_generic("div", llist(x, y))
```)

نبدأ بتثبيت حزمة للتعامل مع #idx("number(s)", sub: "in generic arithmetic system") #idx("ordinary numbers (in generic arithmetic system)") الأعداد #emph[العادية]، أي الأعداد الأولية في لغتنا.
ونوسم هذه بالسلسلة النصية #py("\"python_number\"").
والعمليات الحسابية في هذه الحزمة هي الدوال الحسابية الأولية (لذا لا توجد حاجة لتعريف دوال إضافية للتعامل مع الأعداد غير الموسومة). وبما أن هذه العمليات تأخذ كل منها وسيطين، فإنها تُثبَّت في الجدول مفهرسة بـ القائمة المترابطة #py("llist(\"python_number\", \"python_number\")"):
#idx("package", sub: "Python-number")#idx("pythonnumber package")#idx("installpythonnumberpackage", decl: true)
#snippet(```python
def install_python_number_package():
    def tag(x):
        return attach_tag("python_number", x)
    put("add", llist("python_number", "python_number"),
        lambda x, y: tag(x + y))
    put("sub", llist("python_number", "python_number"),
        lambda x, y: tag(x - y))
    put("mul", llist("python_number", "python_number"),
        lambda x, y: tag(x * y))
    put("div", llist("python_number", "python_number"),
        lambda x, y: tag(x / y))
    put("make", "python_number",
        lambda x: tag(x))
    return "done"
```)

وسينشئ مستخدمو حزمة أعداد #en[Python] أعداداً عادية (موسومة) عن طريق الدالة:

#idx("makepythonnumber", decl: true)
#snippet(```python
def make_python_number(n):
    return get("make", "python_number")(n)
```)

الآن وبعد أن أصبح إطار عمل نظام الحساب العام قائماً، يمكننا facilmente تضمين أنواع جديدة من الأعداد. إليك حزمة تؤدي الحساب الكسري. لاحظ أنه كفائدة للخاصية الجمعية، يمكننا استخدام شفرة الأعداد الكسرية من القسم @sec:rationals دون تعديل كدوال داخلية في الحزمة:
#idx("package", sub: "rational-number")#idx("rational package")#idx("rational-number arithmetic", sub: "interfaced to generic arithmetic system")#idx("installrationalpackage", decl: true)#idx("makerational", decl: true)
#snippet(```python
def install_rational_package():
    # internal functions
    def numer(x): return head(x)
    def denom(x): return tail(x)
    def make_rat(n, d):
        g = gcd(n, d)
        return pair(n // g, d // g)
    def add_rat(x, y):
        return make_rat(numer(x) * denom(y) + numer(y) * denom(x),
                        denom(x) * denom(y))
    def sub_rat(x, y):
        return make_rat(numer(x) * denom(y) - numer(y) * denom(x),
                        denom(x) * denom(y))
    def mul_rat(x, y):
        return make_rat(numer(x) * numer(y),
                        denom(x) * denom(y))
    def div_rat(x, y):
        return make_rat(numer(x) * denom(y),
                        denom(x) * numer(y))
    # interface to rest of the system
    def tag(x):
        return attach_tag("rational", x)
    put("add", llist("rational", "rational"),
        lambda x, y: tag(add_rat(x, y)))
    put("sub", llist("rational", "rational"),
        lambda x, y: tag(sub_rat(x, y)))
    put("mul", llist("rational", "rational"),
        lambda x, y: tag(mul_rat(x, y)))
    put("div", llist("rational", "rational"),
        lambda x, y: tag(div_rat(x, y)))
    put("make", "rational",
        lambda n, d: tag(make_rat(n, d)))
    return "done"

def make_rational(n, d):
    return get("make", "rational")(n, d)
```)

يمكننا تثبيت حزمة مماثلة للتعامل مع الأعداد المركبة، باستخدام الوسم #py("\"complex\"").
وفي إنشاء الحزمة، نستخرج من الجدول العمليات #py("make_from_real_imag") و #py("make_from_mag_ang") التي عُرِّفت بوساطة الحزمتين المستطيلية والقطبية.
وتسمح لنا الخاصية الجمعية (#idx("additivity")) باستخدام الدوال نفسها #py("add_complex") و #py("sub_complex") و #py("mul_complex") و #py("div_complex") من القسم @sec:representations-complex-numbers كعمليات داخلية.
#idx("package", sub: "complex-number")#idx("complex package")#idx("complex-number arithmetic", sub: "interfaced to generic arithmetic system")#idx("installcomplexpackage", decl: true)
#snippet(```python
def install_complex_package():
    # imported functions from rectangular and polar packages
    def make_from_real_imag(x, y):
        return get("make_from_real_imag", "rectangular")(x, y)
    def make_from_mag_ang(r, a):
        return get("make_from_mag_ang", "polar")(r, a)
    # internal functions
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
    # interface to rest of the system
    def tag(z): return attach_tag("complex", z)
    put("add", llist("complex", "complex"),
        lambda z1, z2: tag(add_complex(z1, z2)))
    put("sub", llist("complex", "complex"),
        lambda z1, z2: tag(sub_complex(z1, z2)))
    put("mul", llist("complex", "complex"),
        lambda z1, z2: tag(mul_complex(z1, z2)))
    put("div", llist("complex", "complex"),
        lambda z1, z2: tag(div_complex(z1, z2)))
    put("make_from_real_imag", "complex",
        lambda x, y: tag(make_from_real_imag(x, y)))
    put("make_from_mag_ang", "complex",
        lambda r, a: tag(make_from_mag_ang(r, a)))
    return "done"
```)

يمكن للبرامج خارج حزمة الأعداد المركبة بناء الأعداد المركبة إما من أجزاء حقيقية وتخيلية وإما من سعات وزوايا. لاحظ كيف أن الدوال الأساسية، المعرّفة أصلًا في الحزمتين المستطيلية والقطبية، تُمَدّ إلى الحزمة المركبة، وتُمَدّ من هناك إلى العالم الخارجي.

#idx("makecomplexfromrealimag", decl: true)#idx("makecomplexfrommagang", decl: true)
#snippet(```python
def make_complex_from_real_imag(x, y):
    return get("make_from_real_imag", "complex")(x, y)
def make_complex_from_mag_ang(r, a):
    return get("make_from_mag_ang", "complex")(r, a)
```)

ما لدينا هنا هو #idx("type tag", sub: "two-level") نظام وسوم من مستويين. فالعدد المركب النموذجي، مثل $3+4i$ في الشكل المستطيلي، سيمثل كما هو موضح في الشكل @fig:complex-number-structure.
يُستخدم الوسم الخارجي (#py("\"complex\"")) لتوجيه العدد إلى الحزمة المركبة. وبمجرد الدخول إلى الحزمة المركبة، يُستخدم الوسم التالي (#py("\"rectangular\"")) لتوجيه العدد إلى الحزمة المستطيلية. وفي نظام كبير ومعقد قد تكون هناك مستويات عديدة، يرتبط كل منها بالمستوى التالي بوساطة عمليات عامة. ومع تمرير كائن البيانات "لأسفل"، يُنتزع الوسم الخارجي المستخدَم لتوجيهه إلى الحزمة المناسبة (عن طريق تطبيق #py("contents")) ويصبح المستوى التالي من الوسم (إن وجد) مرئياً ليُستخدَم للإرسال الإضافي.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-65.svg", width: 70%), caption: [تمثيل $3+4i$ في الشكل المستطيلي.], label-name: <fig:complex-number-structure>)

في الحزم أعلاه، استخدمنا #py("add_rat") و #py("add_complex") والدوال الحسابية الأخرى تماماً كما كُتبت أصلًا. وبمجرد أن أصبحت هذه الإعلانات داخلية لدوال تثبيت مختلفة، لم تعد بحاجة لأسماء متميزة عن بعضها البعض: إذ كان بإمكاننا ببساطة تسميتها #py("add") و #py("sub") و #py("mul") و #py("div") في كلا الحزمتين.

#exercise(label-name: <ex:2_77>, [
يحاول #en[Louis Reasoner] تقييم التعبير #py("magnitude(z)") حيث #py("z") هو الكائن الموضح في الشكل @fig:complex-number-structure.
ولدهشته، بدلاً من الإجابة $5$ فإنه يحصل على رسالة خطأ من #py("apply_generic") تقول إنه لا توجد طريقة للعملية #py("magnitude") على الأنواع #py("llist(\"complex\")").
ويعرض هذا التفاعل على #en[Alyssa P. Hacker]، التي تقول: "المشكلة هي أن محددات اختيارات الأعداد المركبة لم تُعرَّف أبداً لأعداد #py("\"complex\"")، بل فقط لأعداد #py("\"polar\"") و #py("\"rectangular\""). وكل ما عليك فعله لجعل هذا يعمل هو إضافة ما يلي إلى حزمة #py("complex"):"

#snippet(```python
put("real_part", llist("complex"), real_part)
put("imag_part", llist("complex"), imag_part)
put("magnitude", llist("complex"), magnitude)
print(put("angle", llist("complex"), angle))
```)

صف بالتفصيل سبب عمل هذا. وكأمثلة، تتبع عبر جميع الدوال المستدعاة في تقييم التعبير #py("magnitude(z)") حيث #py("z") هو الكائن الموضح في الشكل @fig:complex-number-structure. وعلى وجه الخصوص، كم مرة تُستدعى #py("apply_generic")؟ وإلى أي دالة يتم الإرسال في كل حالة؟
])

#exercise(label-name: <ex:internal-type-system>, [
الدوال (#idx("Python", sub: "internal type system") #idx("data types", sub: "in Python") #idx("isnumber (primitive function)", sub: "data types and") #idx("isstring (primitive function)", sub: "data types and") #idx("attachtag", sub: "using Python data types") #idx("typetag", sub: "using Python data types") #idx("contents", sub: "using Python data types")) الداخلية في حزمة #py("python_number") ليست جوهرياً شيئاً أكثر من استدعاءات للدوال الأولية #py("+") و #py("-") إلخ. ولم يكن من الممكن استخدام أوليات اللغة مباشرة لأن نظام وسم الأنواع لدينا يتطلب أن يكون لكل كائن بيانات نوع مرفق به. ولكن في الواقع، تحتوي جميع تنفيذات #en[Python] على نظام أنواع تستخدمه داخلياً. وتحدد المحمولات الأولية مثل #py("is_string") و #py("is_number") ما إذا كانت كائنات البيانات تملك أنواعاً محددة. عدل تعاريف #py("type_tag") و #py("contents") و #py("attach_tag") من القسم @sec:manifest-types بحيث يستفيد نظامنا العام من نظام الأنواع الداخلي لـ #en[Python]. أي ينبغي للنظام أن يعمل كما كان من قبل باستثناء أنه ينبغي تمثيل الأعداد العادية ببساطة كأعداد #en[Python] بدلاً من أزواج يكون رأسها (#py("head")) السلسلة النصية #py("\"python_number\"").
])

#exercise(label-name: <ex:equ->, [
عرف محمول مساوة عاماً #idx("isequal (generic predicate)") #idx("equality", sub: "in generic arithmetic system") #py("is_equal") يفحص مساواة عددين، وثبته في حزمة الحساب العام. وينبغي أن تعمل هذه العملية للأعداد العادية والأعداد الكسرية والأعداد المركبة.
])

#exercise(label-name: <ex:-zero->, [
عرف محمولاً عاماً #idx("isequaltozero (generic)") #idx("zero test (generic)") #py("is_equal_to_zero") يفحص ما إذا كان وسيطه صفرًا، وثبته في حزمة الحساب العام. وينبغي أن تعمل هذه العملية للأعداد العادية والأعداد الكسرية والأعداد المركبة.
])

#idx("generic arithmetic operations")
