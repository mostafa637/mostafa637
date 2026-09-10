// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([البنى الهرمية], label-name: <sec:trees>)

#idx("data", sub: "hierarchical")
#idx("hierarchical data structures")
#idx("tree", sub: "represented as pairs")
#idx("pair(s)", sub: "used to represent tree")

يتعمم تمثيل المتتاليات باستخدام القوائم المترابطة بشكل طبيعي لتمثيل المتتاليات التي قد تكون عناصرها هي نفسها متتاليات. فعلى سبيل المثال، يمكننا اعتبار الكائن
#py("[[1, [2, None]], [3, [4, None]]]")
المُنشأ بوساطة

#snippet(```python
pair(llist(1, 2), llist(3, 4))
```)

قائمة مترابطة ذات ثلاثة عناصر، أولها هو نفسه قائمة مترابطة،
#py("[1, [2, None]]").
ويُظهر الشكل @fig:cons-of-2-lists تمثيل هذه البنية باستخدام الأزواج.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-15.svg", width: 70%), caption: [البنية المكونة بوساطة #py("pair(llist(1, 2), llist(3, 4))").], label-name: <fig:cons-of-2-lists>)

هناك طريقة أخرى للتفكير في المتتاليات التي تكون عناصرها متتاليات وهي اعتبارها #emph[أشجاراً] (#en[trees]). وتكون عناصر المتتالية هي فروع (#en[branches]) الشجرة، والعناصر التي هي نفسها متتاليات تكون أشجاراً فرعية (#en[subtrees]).
ويُظهر الشكل @fig:list-as-tree البنية الموجودة في
الشكل @fig:cons-of-2-lists منظورا إليها كشجرة.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-16.svg", width: 70%), caption: [بنية القائمة المترابطة في الشكل @fig:cons-of-2-lists منظورا إليها كشجرة.], label-name: <fig:list-as-tree>)

التعاود (#en[recursion])
#idx("recursion", sub: "in working with trees")
أداة طبيعية للتعامل مع بنى الأشجار، حيث يمكننا غالباً اختزال العمليات على الأشجار إلى عمليات على فروعها، والتي تختزل بدورها إلى عمليات على فروع الفروع، وهكذا حتى نصل إلى أوراق (#en[leaves]) الشجرة. وكأمثلة على ذلك، قارن دالة
#py("length")
من القسم @sec:sequences مع دالة
#idx("countleaves")
#idx("tree", sub: "counting leaves of")
#py("count_leaves")،
والتي ترجع العدد الإجمالي لأوراق الشجرة:

#snippet(```python
x = pair(llist(1, 2), llist(3, 4))
```)

#snippet(```python
print(length(x))
```)

#output(```python
print(length(x))
```)

#snippet(```python
print(count_leaves(x))
```)

#output(```python
print(count_leaves(x))
```)

#snippet(```python
print_llist(llist(x, x))
```)

#output(```python
print_llist(llist(x, x))
```)

#snippet(```python
print(length(llist(x, x)))
```)

#output(```python
print(length(llist(x, x)))
```)

#snippet(```python
print(count_leaves(llist(x, x)))
```)

#output(```python
print(count_leaves(llist(x, x)))
```)

لتنفيذ
#py("count_leaves")،
تذكر الخطة التعاودية لحساب
#py("length"):

- طول (#py("length")) القائمة المترابطة #py("x") هو 1 زائد طول (#py("length")) ذيل (#py("tail")) #py("x").
- طول (#py("length")) القائمة المترابطة الفارغة هو 0.

تتشابه الدالة #py("count_leaves") مع ذلك. فالقيمة بالنسبة للقائمة المترابطة الفارغة هي نفسها:

- #py("count_leaves") للقائمة المترابطة الفارغة هي 0.

ولكن في خطوة التخفيض، حيث ننتزع رأس (#py("head")) القائمة المترابطة، يجب أن نأخذ في الاعتبار أن الرأس (#py("head")) نفسه قد يكون شجرة نحتاج إلى عد أوراقها. وبالتالي، فإن خطوة التخفيض المناسبة هي:

- #py("count_leaves") لشجرة #py("x") هي #py("count_leaves") لرأس (#py("head")) #py("x") زائد #py("count_leaves") لذيل (#py("tail")) #py("x").

أخيراً، من خلال أخذ الرؤوس (#py("head")s)، نصل إلى الأوراق الفعلية، لذا نحتاج إلى حالة أساسية أخرى:

- #py("count_leaves") للورقة هي 1.

للمساعدة في كتابة دوال تعاودية على الأشجار،
توفر بيئة #en[Python] الخاصة بنا المحمول الأولي
#idx("ispair (primitive function)")

#py("is_pair")،
والذي يفحص ما إذا كان معامله زوجاً. إليك الدالة الكاملة:#footnote[ترتيب المحمولين مهم، لأن #py("None") يحقق #py("is_none") وهو ليس زوجاً أيضاً.]
#idx("countleaves", decl: true)
#snippet(```python
def count_leaves(x):
    return (0 if is_none(x)
            else 1 if not is_pair(x)
            else count_leaves(head(x)) + count_leaves(tail(x)))
```)

#exercise(label-name: <ex:nested-list>, [
نفترض أننا قيمنا التعبير
#py("llist(1, llist(2, llist(3, 4)))").
أعطِ النتيجة المطبوعة بواسطة المفسر، وبنية الصندوق والمؤشر المقابلة لها، وتفسير ذلك كشجرة (كما في الشكل @fig:list-as-tree).

#anchor(<ex:2_24>)
])

#exercise(label-name: <ex:2_25>, [
أعطِ تركيبات من الرؤوس (#py("head")s) واللذيول (#py("tail")s) التي ستستخرج 7 من كل قائمة من القوائم المترابطة التالية، المعطاة بترميز القوائم المترابطة:

#snippet(```python
llist(1, 3, llist(5, 7), 9)

llist(llist(7))

llist(1, llist(2, llist(3, llist(4, llist(5, llist(6, 7))))))
```)
])

#exercise(label-name: <ex:2_26>, [
نفترض أننا عرفنا #py("x") و #py("y") ليكون قمعين (قائمتين مترابطتين):

#snippet(```python
x = llist(1, 2, 3)

y = llist(4, 5, 6)
```)

ما هي نتيجة تقييم كل من التعبيرات التالية، بالترميز الصندوقي وبترميز القائمة المترابطة؟

#snippet(```python
append(x, y)
```)

#snippet(```python
pair(x, y)
```)

#snippet(```python
llist(x, y)
```)
])

#exercise(label-name: <ex:2_27>, [
عدّل دالة #py("reverse") من التمرين @ex:reverse لإنتاج دالة
#idx("deepreverse")
#idx("tree", sub: "reversing at all levels")
#py("deep_reverse")
تأخذ قائمة مترابطة كمعامل وترجع كقيمتها القائمة المترابطة مع عكس عناصرها وعكس جميع القوائم الفرعية معها أيضاً بشكل عميق. على سبيل المثال،

#snippet(```python
x = llist(llist(1, 2), llist(3, 4))
```)

#snippet(```python
print_llist(x)
```)

#output(```python
print_llist(x)
```)

#snippet(```python
print_llist(reverse(x))
```)

#output(```python
print_llist(reverse(x))
```)

#snippet(```python
print_llist(deep_reverse(x))
```)

#output(```python
print_llist(deep_reverse(x))
```)
])

#exercise(label-name: <ex:fringe>, [
اكتب دالة
#idx("fringe")
#idx("tree", sub: "fringe of")
#py("fringe")
تأخذ شجرة كمعامل (ممثلة كقائمة مترابطة) وترجع قائمة مترابطة تكون عناصرها جميع أوراق الشجرة مرتبة من اليسار إلى اليمين. على سبيل المثال،

#snippet(```python
x = llist(llist(1, 2), llist(3, 4))
```)

#snippet(```python
print_llist(fringe(x))
```)

#output(```python
print_llist(fringe(x))
```)

#snippet(```python
print_llist(fringe(llist(x, x)))
```)

#output(```python
print_llist(fringe(llist(x, x)))
```)
])

#exercise(label-name: <ex:mobile>, [
يتكون المنشأ المعلق الثنائي (#idx("mobile") #en[binary mobile]) من فرعين: فرع أيسر وفرع أيمن. كل فرع هو قضيب ذو طول معين، يعلق منه إما وزن أو منشأ معلق ثنائي آخر. يمكننا تمثيل المنشأ المعلق الثنائي باستخدام البيانات المركبة عن طريق بنائه من فرعين (على سبيل المثال، باستخدام #py("llist")):

#snippet(```python
def make_mobile(left, right):
    return llist(left, right)
```)

يتم بناء الفرع من طول (#py("length")) (يجب أن يكون رقماً) جنباً إلى جنب مع بنية (#py("structure"))، والتي قد تكون إما رقماً (يمثل وزناً بسيطاً) أو منشأ معلقاً آخر:

#snippet(```python
def make_branch(length, structure):
    return llist(length, structure)
```)

+ اكتب دوال الاختيار المقابلة #py("left_branch") و #py("right_branch")، والتي ترجع فروع المنشأ المعلق، و #py("branch_length") و #py("branch_structure")، والتي ترجع مكونات الفرع.
+ باستخدام دوال الاختيار الخاصة بك، عرف دالة #py("total_weight") ترجع الوزن الإجمالي للمنشأ المعلق.
+ يُقال عن المنشأ المعلق إنه #idx("balanced mobile") #emph[متوازن] إذا كان عزم الدوران الناتِج عن فرعه العلوي الأيسر يساوي عزم الدوران الناتِج عن فرعه العلوي الأيمن (أي إذا كان طول القضيب الأيسر مضروباً في الوزن المعلق منه يساوي الجداء المقابل للجانب الأيمن) وكان كل من المناشئ المعلقة الفرعية المعلقة من فروعه متوازناً. صمم محمولاً يفحص ما إذا كان المنشأ المعلق الثنائي متوازناً.
+ نفترض أننا غيرنا تمثيل المناشئ المعلقة بحيث تكون البواني: #snippet(```python def make_mobile(left, right): return pair(left, right) def make_branch(length, structure): return pair(length, structure) ```) كم يتوجب عليك تغيير برامجك للتحويل إلى التمثيل الجديد؟
])

#idx("data", sub: "hierarchical")
#idx("hierarchical data structures")
#idx("tree", sub: "represented as pairs")
#idx("pair(s)", sub: "used to represent tree")

#subheading([التطبيق الخرائطي على الأشجار])

#idx("tree", sub: "mapping over")
#idx("mapping", sub: "over trees")

تماماً كما تُعدّ #py("map") تجريداً قوياً للتعامل مع المتتاليات، فإن #py("map") جنباً إلى جنب مع التعاود تُعدّ تجريداً قوياً للتعامل مع الأشجار. على سبيل المثال، الدالة
#py("scale_tree")،
المماثلة لـ #py("scale_linked_list") من القسم @sec:sequences، تأخذ كمعاملات عاملاً عددياً وشجرة أوراقها أعداد. وترجع شجرة بالشكل نفسه، حيث يتم ضرب كل عدد في المعامل. والخطة التعاودية لـ
#py("scale_tree")
مشابهة لخطة
#py("count_leaves"):
#idx("scaletree", decl: true)
#snippet(```python
def scale_tree(tree, factor):
    return (None if is_none(tree)
            else tree * factor if not is_pair(tree)
            else pair(scale_tree(head(tree), factor),
                      scale_tree(tail(tree), factor)))
```)

#snippet(```python
print_llist(scale_tree(llist(1, llist(2, llist(3, 4), 5), llist(6, 7)),
                       10))
```)

#output(```python
print_llist(scale_tree(llist(1, llist(2, llist(3, 4), 5), llist(6, 7)),
                       10))
```)

طريقة أخرى لتنفيذ #py("scale_tree") هي اعتبار الشجرة كمتتالية من الأشجار الفرعية واستخدام #py("map").
نطبق #py("map") على المتتالية، مغيرين مقياس كل شجرة فرعية بدورها، ونرجع القائمة المترابطة للنتائج. وفي الحالة الأساسية، حيث تكون الشجرة ورقة، نضرب ببساطة في المعامل:

#idx("scaletree", decl: true)
#snippet(```python
def scale_tree(tree, factor):
    return map(lambda sub_tree: (scale_tree(sub_tree, factor)
                                 if is_pair(sub_tree)
                                 else sub_tree * factor),
               tree)
```)

يمكن تنفيذ العديد من عمليات الأشجار بمجموعات مماثلة من عمليات المتتاليات والتعاود.

#exercise(label-name: <ex:square-tree>, [
أعلن عن دالة
#py("square_tree")
مماثلة لدالة
#py("square_linked_list")
من التمرين @ex:square-list. أي ينبغي لـ
#py("square_tree")
أن تتصرف على النحو التالي:

#snippet(```python
print_llist(square_tree(llist(1,
                        llist(2, llist(3, 4), 5),
                        llist(6, 7))))
```)

#output(```python
print_llist(square_tree(llist(1,
                        llist(2, llist(3, 4), 5),
                        llist(6, 7))))
```)

أعلن عن #py("square_tree") مباشرةً (أي دون استخدام أي دوال عليا) وأيضاً باستخدام #py("map") والتعاود.
])

#exercise(label-name: <ex:tree-map>, [
جرد إجابتك للتمرين @ex:square-tree لإنتاج دالة
#idx("treemap")
#py("tree_map")
بخاصية تسمح بالإعلان عن #py("square_tree") كـ:

#snippet(```python
def square_tree(tree): return tree_map(square, tree)
```)
])

#exercise(label-name: <ex:2_32>, [
يمكننا تمثيل
#idx("set", sub: "subsets of")
مجموعة كقائمة مترابطة من العناصر المتميزة، ويمكننا تمثيل مجموعة جميع المجموعات الجزئية لمجموعة كقائمة مترابطة من القوائم المترابطة. على سبيل المثال، إذا كانت المجموعة هي
#py("llist(1, 2, 3)")،
فإن مجموعة كل المجموعات الجزئية هي
#snippet(```python llist(None, llist(3), llist(2), llist(2, 3), llist(1), llist(1, 3), llist(1, 2), llist(1, 2, 3)) ```)
أكمل الإعلان التالي لدالة تنشئ مجموعة المجموعات الجزئية لمجموعة وأعطِ شرحاً واضحاً لسبب عملها:
#idx("subsets of a set", decl: true)
#syntax("
def subsets(s):
    if is_none(s):
        return llist(None)
    else:
        rest = subsets(tail(s))
        return append(rest, map(", metaphrase[??], ", rest))
      ")
])

#idx("tree", sub: "mapping over")
#idx("mapping", sub: "over trees")
