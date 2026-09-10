// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([تشغيل المُقيِّم كـ برنامج], label-name: <sec:running-eval>)

#idx("metacircular evaluator for Python", sub: "running")

بالنظر إلى المـُقيِّم، لدينا في أيدينا وصف (مـُعبر عنه بـ Python) لـ العملية التي تـُقيَّم بها عبارات وتعبيرات Python. وإحدى مزايا التعبير عن المـُقيِّم كـ برنامج هي أنه يمكننا تشغيل البرنامج. وهذا يعطينا، مشغلاً داخل Python، نموذجاً عاملاً لكيفية تقييم Python نفسها للتعبيرات. ويمكن أن يخدم هذا كـ إطار عمل للتجريب بـ قواعد التقييم، كما سنفعل لاحقاً في هذا الفصل.

يـُختزل برنامج المـُقيِّم الخاص بنا التعبيرات في النهاية إلى تطبيق
#idx("metacircular evaluator for Python", sub: "primitive functions")
الدوال الأوليّة.
ولذلك، فإن كل ما نحتاجه لتشغيل المـُقيِّم هو إنشاء آلية تستدعي نظام Python الأساسي لنمذجة تطبيق الدوال الأوليّة.

يجب أن يكون هناك ارتباط لكل اسم دالة أوليّة ومعامل، بحيث عندما تـُقيّم
#py("evaluate")
تعبير الدالة لـ تطبيق دالة أوليّة، ستعثر على كائن لترفق به #py("apply"). وهكذا نـُعد
#idx("metacircular evaluator for Python", sub: "global environment")
#idx("global environment", sub: "in metacircular evaluator")
بيئة عالمية تربط كائنات فريدة بـ أسماء الدوال الأوليّة والمعاملات التي يمكن أن تظهر في التعبيرات التي سنقوّم بتقييمها.
#idx("symbol(s)", sub: "in global environment")

تتضمن البيئة العالمية أيضاً ارتباطات لـ
#idx("metacircular evaluator for Python", sub: "None")
#py("None")
وأسماء أخرى،
بحيث يمكن استخدامها كـ ثوابت في التعبيرات المراد تقييمها.

#idx("setupenvironment", decl: true)
#snippet(```python
def setup_environment():
    return extend_environment(append(primitive_function_symbols, primitive_constant_symbols), append(primitive_function_objects, primitive_constant_values), the_empty_environment)
```)

#idx("theglobalenvironment", decl: true)
#snippet(```python
the_global_environment = setup_environment()
```)

لا يهم كيف نمثل كائنات الدوال الأوليّة، طالما أن #py("apply") يمكنها التعرف عليها وتطبيقها باستخدام الدوال #py("is_primitive_function") و #py("apply_primitive_function"). ولقد اخترنا تمثيل الدالة الأوليّة كـ قائمة تبدأ بالسلسلة النصية #py("\"primitive\"") وتحتوي على دالة في JavaScript الأساسية تنفذ تلك الدالة الأوليّة.
#idx("isprimitivefunction", decl: true)#idx("primitiveimplementation", decl: true)
#snippet(```python
def is_primitive_function(fun):
    return is_tagged_list(fun, "primitive")

def primitive_implementation(fun):
    return head(tail(fun))
```)

ستحصل الدالة #py("setup_environment") على أسماء الدوال الأوليّة ودوال التنفيذ من قائمة:#footnote[أي دالة معرفة في Python الأساسية يمكن استخدامها كـ دالة أوليّة للمـُقيِّم دائري التجريد. ولا يلزم أن يكون اسم الدالة الأوليّة المـُثبتة في المـُقيِّم هو نفسه اسم تنفيذها في Python الأساسية؛ والأسماء هي نفسها هنا لأن المـُقيِّم دائري التجريد ينفذ Python نفسها.
وهكذا، على سبيل المثال، كان بإمكاننا وضع #py("llist(\"first\", head)") أو #py("llist(\"square\", lambda x: x * x)") في قائمة #py("primitive_functions").]
#idx("primitivefunctionsymbols", decl: true)#idx("primitivefunctionobjects", decl: true)
#syntax("
primitive_functions = llist(llist(\"head\",    head             ),
                             llist(\"tail\",    tail             ),
                             llist(\"pair\",    pair             ),
                             llist(\"is_null\", is_none          ),
                             llist(\"+\",       lambda x, y: x + y  ),
                             ", metaphrase[more primitive functions], "
                            )

primitive_function_symbols = \\
    map(lambda f: head(f), primitive_functions)

primitive_function_objects = \\
    map(lambda f: llist(\"primitive\", head(tail(f))),
        primitive_functions)
      ")

وبشكل مماثل للدوال الأوليّة، نُعرّف ثوابت أوليّة أخرى تـُثبَّت في البيئة العالمية بواسطة الدالة #py("setup_environment").

#syntax("
primitive_constants = llist(llist(\"undefined\", None),
                             llist(\"math_PI\",   math_pi)
                             ", metaphrase[more primitive constants], "
                            )

primitive_constant_symbols = \\
    map(lambda c: head(c), primitive_constants)

primitive_constant_values = \\
    map(lambda c: head(tail(c)), primitive_constants)
	  ")

لتطبيق دالة أوليّة، نكتفي بتطبيق دالة التنفيذ على الوسائط، باستخدام نظام Python الأساسي:#footnote[#anchor(<foot:vector-array>)
طريقة #py("apply") في Python تتوقع وسائط الدالة في #emph[متجه] (#en[vector]). (تـُسمى المتجهات "مصفوفات" في Python).
وبالتالي، تـُحول #py("arglist") إلى متجه—وهنا باستخدام حلقة طالما (انظر التمرين @ex:while_loop):
#idx("applyinunderlyingjavascript", decl: true)#idx("apply (primitive method)")
#syntax("
def apply_in_underlying_javascript(prim, arglist):
    arg_vector = []
    # empty vector
    i = 0
    while  not is_none(arglist):
        arg_vector[i] = head(arglist)
        # store value at index ", $mono("i")$, "
        i = i + 1
        arglist = tail(arglist)
    return prim.apply(prim, arg_vector)
    # ", $mono("apply")$, " is accessed via ", $mono("prim")$)

استخدمنا أيضاً #py("apply_in_underlying_javascript") للإعلان عن الدالة #py("apply_generic") في القسم @sec:data-directed.]
#idx("applyprimitivefunction", decl: true)
#snippet(```python
def apply_primitive_function(fun, arglist):
    return apply_in_underlying_javascript(primitive_implementation(fun), arglist)
```)

#idx("metacircular evaluator for Python", sub: "primitive functions")

للتسهيل في تشغيل المـُقيِّم دائري التجريد، نوفر
#idx("metacircular evaluator for Python", sub: "driver loop")
#idx("driver loop", sub: "in metacircular evaluator")
#emph[حلقة المحرك] (#en[driver loop]) التي تنماذج حلقة القراءة والتقييم والطباعة لنظام JavaScript الأساسي. حيث تطبع
#idx("prompts")
#idx("prompts", sub: "metacircular evaluator")
#emph[محثاً] (#en[prompt]) وتقرأ برنامج دخل كـ سلسلة نصية.
وتـُحول السلسلة النصية للبرنامج إلى تمثيل قائمة مـُعنونة للعبارة كما هو موضح في القسم @sec:representing-expressions—وهي عملية تـُسمى الإعراب وتـُنجز بواسطة الدالة الأوليّة #py("parse").
ونسفق كل نتيجة مطبوعة بـ #emph[محث خرج] وذلك للتمييز بين قيمة البرنامج والخرج الآخر الذي قد يـُطبع. وتحصل حلقة المحرك على بيئة البرنامج السابقة كـ وسيط.
وكما هو موضح في نهاية القسم @sec:env-internal-def، فإن حلقة المحرك تعامل البرنامج كما لو كان في كتلة: حيث تفحص الإعلانات، وتوسع البيئة المعطاة بـ إطار يحتوي على ارتباط لكل اسم بـ #py("\"*unassigned*\"")، وتُقيّم البرنامج بالنسبة للبيئة المـُوسعة، والتي تـُمَرَّر بعد ذلك كـ وسيط للتكرار التالي لحلقة المحرك.
#idx("driverloop", sub: "for metacircular evaluator", decl: true)
#snippet(```python
input_prompt = "M-evaluate input: "
output_prompt = "M-evaluate value: "
def driver_loop(env):
    input = user_read(input_prompt)
    if is_none(input):
        print("evaluator terminated")
    else:
        program = parse(input)
        locals = scan_out_declarations(program)
        unassigneds = list_of_unassigned(locals)
        program_env = extend_environment(locals, unassigneds, env)
        output = evaluate(program, program_env)
        user_print(output_prompt, output)
        return driver_loop(program_env)
```)

نستخدم دالة #py("prompt") في Python لطلب وقراءة السلسلة النصية للدخل من المستخدم:
#idx("userread", decl: true)
#snippet(```python
def user_read(prompt_string):
    return prompt(prompt_string)
```)

ترجع الدالة
#idx("prompt (primitive function)")
#py("prompt")
#py("null") عندما يلغي المستخدم الدخل. ونستخدم دالة طباعة خاصة #py("user_print")، لتجنب طباعة جزء البيئة لـ دالة مركبة، والذي قد يكون قائمة طويلة جداً (أو قد يحتوي على دورات).
#idx("userprint", decl: true)
#snippet(```python
def user_print(string, object):
    def prepare(object):
        return "< compound-function >" if is_compound_function(object) else "< primitive-function >" if is_primitive_function(object) else pair(prepare(head(object)), prepare(tail(object))) if is_pair(object) else object
    print(string + " " + stringify(prepare(object)))
```)

الآن كل ما نحتاجه لتشغيل المـُقيِّم هو تهيئة البيئة العالمية وبدء حلقة المحرك. وإليك تفاعلاً نموذجياً:

#snippet(```python
the_global_environment = setup_environment()
driver_loop(the_global_environment)
```)

#prompt(```python
M-evaluate input:
```)

#snippet(```python
def append(xs, ys):
    return ys if is_none(xs) else pair(head(xs), append(tail(xs), ys))
```)

#output(```python
def append(xs, ys):
    return ys if is_none(xs) else pair(head(xs), append(tail(xs), ys))
```)

#prompt(```python
M-evaluate input:
```)

#snippet(```python
append(llist("a", "b", "c"), llist("d", "e", "f"))
```)

#output(```python
append(llist("a", "b", "c"), llist("d", "e", "f"))
```)

#exercise(label-name: <ex:mceval-map>, [
تُجري كل من إيفا لو آتور ورئيسها لويس ريسونر تجارب على المـُقيِّم دائري التجريد. حيث تكتب إيفا تعريف #py("map")، وتُشغل بعض البرامج الاختيارية التي تستخدمه. وتعمل بشكل جيد. وفي المقابل، قام لويس بتثبيت نسخة النظام لـ #py("map") كـ دالة أوليّة للمـُقيِّم دائري التجريد. وعندما يحاول استخدامها، تسوء الأمور بشكل رهيب. اشرح لماذا تفشل #py("map") الخاصة بـ لويس في حين تعمل الخاصة بـ إيفا.
])

#idx("metacircular evaluator for Python", sub: "running")
