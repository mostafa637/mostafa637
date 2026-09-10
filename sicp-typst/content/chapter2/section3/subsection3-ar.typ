// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([مثال: تمثيل المجموعات], label-name: <sec:representing-sets>)

#idx("set")

في الأمثلة السابقة، بنينا تمثيلات لنوعين من كائنات البيانات المركبة: الأعداد الكسرية والتعبيرات الجبرية. وفي أحد هذه الأمثلة، كان لدينا خيار تبسيط (تخفيض) التعبيرات إما في وقت البناء وإما في وقت الاختيار، ولكن خلاف ذلك كان اختيار تمثيل لهذه البنى بدلالة القوائم المترابطة أمراً مباشراً. وعندما ننتقل إلى تمثيل المجموعات، فإن اختيار التمثيل ليس وضاءاً إلى هذا الحد. وفي الواقع، هناك عدد من التمثيلات الممكنة، وهي تختلف تفاوتاً كبيراً عن بعضها البعض في طرق عدة.

بشكل غير رسمي، المجموعة هي مجرد تشكيلة من الكائنات المتميزة. ولإعطاء تعريف أكثر دقة، يمكننا استخدام طريقة تجريد البيانات. أي أننا نعرّف "المجموعة" عن طريق تحديد العمليات التي سيتم استخدامها على المجموعات (#idx("set", sub: "operations on")). وهذه العمليات هي
#py("union_set")،
#py("intersection_set")،
#py("is_element_of_set")،
و
#py("adjoin_set").
#idx("iselementofset")
الدالة #py("is_element_of_set") هي محمول يحدد ما إذا كان عنصر معطى عنصراً في مجموعة.
#idx("adjoinset")
وتأخذ الدالة #py("adjoin_set") كائناً ومجموعة كمعاملات وترجع مجموعة تحتوي على عناصر المجموعة الأصلية بالإضافة إلى العنصر الملحق.
#idx("unionset")
وتحسب الدالة #py("union_set") اتحاد مجموعتين، وهو المجموعة التي تحتوي على كل عنصر يظهر في أي من المعاملين.
#idx("intersectionset")
وتحسب الدالة #py("intersection_set") تقاطع مجموعتين، وهو المجموعة التي تحتوي فقط على العناصر التي تظهر في كلا المعاملين. ومن وجهة نظر تجريد البيانات، نحن أحرار في تصميم أي تمثيل ينفذ هذه العمليات بطريقة متسقة مع التفسيرات المعطاة أعلاه.#footnote[إذا أردنا أن نكون أكثر صرامة، فيمكننا تحديد "متسق مع التفسيرات المعطاة أعلاه" ليعني أن العمليات تحقق مجموعة من القواعد مثل هذه:

- لأي مجموعة #py("S") وأي كائن #py("x")، فإن #py("is_element_of_set(x, adjoin_set(x, S))") صحيح (غير رسمي: "إلحاق كائن بمجموعة ينتج عنه مجموعة تحتوي على الكائن").
- لأي مجموعتين #py("S") و #py("T") وأي كائن #py("x")، فإن #py("is_element_of_set(x, union_set(S, T))") يساوي #py("is_element_of_set(x, S) or is_element_of_set(x, T)") (غير رسمي: "عناصر #py("union_set(S, T)") هي العناصر الموجودة في #py("S") أو في #py("T")").
- لأي كائن #py("x")، فإن #py("is_element_of_set(x, None)") خطأ (غير رسمي: "لا يوجد كائن هو عنصر في المجموعة الفارغة").]
#idx("set", sub: "operations on")

#subheading([المجموعات كقوائم مترابطة غير مرتبة])

#idx("set", sub: "represented as unordered linked list")
#idx("unordered-linked-list representation of sets")

إحدى طرق تمثيل المجموعة هي كقائمة مترابطة من عناصرها لا يظهر فيها أي عنصر أكثر من مرة واحدة. وتُمثل المجموعة الفارغة بالقائمة المترابطة الفارغة.
وفي هذا التمثيل، تشبه #py("is_element_of_set") الدالة #py("member") من القسم @sec:strings.
وتستخدم #py("equal") بدلاً من #py("==") حتى لا يلزم أن تكون عناصر المجموعة أعداداً أو سلاسل نصية فقط:
#idx("iselementofset", sub: "unordered-linked-list representation", decl: true)
#snippet(```python
def is_element_of_set(x, set):
    return (False
            if is_none(set)
            else True
            if x == head(set)
            else is_element_of_set(x, tail(set)))
```)

باستخدام هذا، يمكننا كتابة #py("adjoin_set").
إذا كان الكائن المراد إلحاقه موجوداً بالفعل في المجموعة، فنرجع المجموعة فقط.
خلاف ذلك، نستخدم #py("pair") لإضافة الكائن إلى القائمة المترابطة التي تمثل المجموعة:
#idx("adjoinset", sub: "unordered-linked-list representation", decl: true)
#snippet(```python
def adjoin_set(x, set):
    return (set
            if is_element_of_set(x, set)
            else pair(x, set))
```)

بالنسبة لـ #py("intersection_set")، يمكننا استخدام استراتيجية تعاودية. فإذا كنا نعرف كيفية تشكيل تقاطع #py("set2") وذيل (#py("tail")) #py("set1")، فكل ما نحتاجه هو البت في إدراج رأس (#py("head")) #py("set1") في هذا التقاطع أم لا. ولكن هذا يعتمد على ما إذا كان #py("head(set1)") موجوداً أيضاً في #py("set2"). إليك الدالة الناتجة:
#idx("intersectionset", sub: "unordered-linked-list representation", decl: true)
#snippet(```python
def intersection_set(set1, set2):
    return (None
            if is_none(set1) or is_none(set2)
            else pair(head(set1), intersection_set(tail(set1), set2))
            if is_element_of_set(head(set1), set2)
            else intersection_set(tail(set1), set2))
```)

عند تصميم التمثيل، من المسائل التي يجب أن نهتم بها هي الكفاءة. تأمل عدد الخطوات التي تتطلبها عمليات المجموعات لدينا. وبما أنها جميعها تستخدم #py("is_element_of_set")، فإن سرعة هذه العملية تملك تأثيراً رئيساً على كفاءة تنفيذ المجموعة ككل. والآن، لفحص ما إذا كان كائن ما عنصراً في مجموعة، قد تضطر #py("is_element_of_set") إلى مسح المجموعة بأكملها. (في أسوأ الحالات، يتبين أن الكائن ليس في المجموعة.) ومن ثم، إذا كانت للمجموعة $n$ من العناصر، فإن #py("is_element_of_set") قد تستغرق ما يصل إلى $n$ من الخطوات. وبالتالي، فإن عدد الخطوات المطلوبة ينمو كـ $Theta (n)$. وعدد الخطوات التي تتطلبها #py("adjoin_set")، والتي تستخدم هذه العملية، ينمو أيضاً كـ $Theta (n)$. وبالنسبة لـ #py("intersection_set")، التي تجري فحص #py("is_element_of_set") لكل عنصر من عناصر #py("set1")، ينمو عدد الخطوات المطلوبة كحاصل ضرب أحجام المجموعات المعنية، أو $Theta (n^(2))$ لمجموعتين بحجم $n$. وسيكون الأمر نفسه بالنسبة لـ #py("union_set").

#exercise(label-name: <ex:2_59>, [
نفذ عملية #idx("unionset", sub: "unordered-linked-list representation") #py("union_set") لتمثيل القائمة المترابطة غير المرتبة للمجموعات.
])

#exercise(label-name: <ex:2_60>, [
حددنا أن المجموعة تُمثَّل كقائمة مترابطة دون تكرارات.
الآن نفترض أننا نسمح بالتكرارات. على سبيل المثال، يمكن تمثيل المجموعة $\{1,2,3\}$ كالقائمة المترابطة #py("llist(2, 3, 2, 1, 3, 2, 2)").
صمم الدوال #py("is_element_of_set") و #py("adjoin_set") و #py("union_set") و #py("intersection_set") التي تعمل على هذا التمثيل. كيف تقارن كفاءة كل واحدة منها مع الدالة المقابلة لتمثيل عدم التكرار؟ هل هناك تطبيقات قد تستخدم فيها هذا التمثيل تفضيلاً على تمثيل عدم التكرار؟
])

#idx("set", sub: "represented as unordered linked list")
#idx("unordered-linked-list representation of sets")

#subheading([المجموعات كقوائم مترابطة مرتبة])

#idx("set", sub: "represented as ordered linked list")
#idx("ordered-linked-list representation of sets")

إحدى طرق التسريع لعمليات المجموعات لدينا هي تغيير التمثيل بحيث تسرد عناصر المجموعة بترتيب تصاعدي. وللقيام بذلك، نحتاج إلى طريقة ما للمقارنة بين كائنين لنتمكن من قول أيهما أكبر. فعلى سبيل المثال، يمكننا مقارنة السلاسل النصية معجمياً، أو يمكننا الاتفاق على طريقة ما لتعيين عدد فريد لكائن ثم مقارنة العناصر عن طريق مقارنة الأعداد المقابلة. ولإبقاء مناقشتنا بسيطة، سننظر فقط في الحالة التي تكون فيها عناصر المجموعة أعداداً، حتى نتمكن من مقارنة العناصر باستخدام #py(">") و #py("<"). وسوف نمثل مجموعة من الأعداد بسرد عناصرها بترتيب تصاعدي. وبينما سمح لنا تمثيلنا الأول أعلاه بتمثيل المجموعة $\{1,3,6,10\}$ بسرد العناصر في أي ترتيب، فإن تمثيلنا الجديد لا يسمح إلا بالقائمة المترابطة #py("llist(1, 3, 6, 10)").

إحدى مزايا الترتيب تظهر في #py("is_element_of_set"):
فعند الفحص عن وجود عنصر ما، لم نعد مضطرين لمسح المجموعة بأكملها. وإذا وصلنا إلى عنصر مجموعة أكبر من العنصر الذي نبحث عنه، فإننا نعرف أن العنصر ليس في المجموعة:
#idx("iselementofset", sub: "ordered-linked-list representation", decl: true)
#syntax("
def is_element_of_set(x, set):
    return (False
            if is_none(set)
            else True
            if x == head(set)
            else False
            if x < head(set)
            # ", $mono("x > head(set)")$, "
            else is_element_of_set(x, tail(set)))
      ")

كم عدد الخطوات التي يوفره هذا؟ في أسوأ الحالات، قد يكون العنصر الذي نبحث عنه هو الأكبر في المجموعة، لذا فإن عدد الخطوات هو نفسه بالنسبة للتمثيل غير المرتب. ومن ناحية أخرى، إذا بحثنا عن عناصر بأحجام مختلفة كثيرة، فيمكننا التوقع أنه في بعض الأحيان سنكون قادرين على التوقف عن البحث عند نقطة قريبة من بداية القائمة المترابطة، وفي أحيان أخرى سنظل بحاجة إلى فحص معظم القائمة المترابطة. وفي المتوسط، ينبغي أن نتوقع الاضطرار إلى فحص نصف العناصر في المجموعة تقريباً. وبالتالي، فإن متوسط عدد الخطوات المطلوبة سيكون حوالي $n/2$.
ولا يزال هذا نمواً بمقدار $Theta (n)$، ولكنه يوفر لنا، في المتوسط، عاملاً قدره 2 في عدد الخطوات مقارنة بالتنفيذ السابق.

ونحصل على تسريع أكثر إبهاراً مع #py("intersection_set").
ففي التمثيل غير المرتب، تطلبت هذه العملية $Theta (n^(2))$ خطوة، لأننا أجرينا مسحاً كاملاً لـ #py("set2") لكل عنصر من عناصر #py("set1"). ولكن مع التمثيل المرتب، يمكننا استخدام طريقة أكثر ذكاءً. نبدأ بمقارنة العنصرين الأوليين، #py("x1") و #py("x2")، للمجموعتين. وإذا كان #py("x1") يساوي #py("x2")، فإن ذلك يعطي عنصراً في التقاطع، وبقية التقاطع هي تقاطع الذيلين (#py("tail")s) للمجموعتين. نفترض، مع ذلك، أن #py("x1") أقل من #py("x2"). وبما أن #py("x2") هو أصغر عنصر في #py("set2")، فيمكننا أن نستنتج فوراً أن #py("x1") لا يمكن أن يظهر في أي مكان في #py("set2") وبالتالي ليس في التقاطع. ومن ثم، فإن التقاطع يساوي تقاطع #py("set2") مع ذيل (#py("tail")) #py("set1"). وبالمثل، إذا كان #py("x2") أقل من #py("x1")، فإن التقاطع يُعطى بتقاطع #py("set1") مع ذيل (#py("tail")) #py("set2"). إليك الدالة:
#idx("intersectionset", sub: "ordered-linked-list representation", decl: true)
#syntax("
def intersection_set(set1, set2):
    if is_none(set1) or is_none(set2):
        return None
    else:
        x1 = head(set1)
        x2 = head(set2)
        return (pair(x1, intersection_set(tail(set1), tail(set2)))
                if x1 == x2
                else intersection_set(tail(set1), set2)
                if x1 < x2
                # ", $mono("x2 < x1")$, "
                else intersection_set(set1, tail(set2)))
      ")

لتقدير عدد الخطوات التي تتطلبها هذه العملية، لاحظ أننا نختزل في كل خطوة مشكلة التقاطع إلى حساب تقاطعات مجموعات أصغر—بإزالة العنصر الأول من #py("set1") أو #py("set2") أو كليهما. وبالتالي، فإن عدد الخطوات المطلوبة هو على الأكثر مجموع حجمي #py("set1") و #py("set2")، بدلاً من حاصل ضرب الحجمين كما في التمثيل غير المرتب. وهذا نمو بـ $Theta (n)$ بدلاً من $Theta (n^(2))$—وهو تسريع ملحوظ، حتى بالنسبة للمجموعات ذات الحجم المتوسط.

#exercise(label-name: <ex:adjoin-set>, [
قدم تنفيذاً لـ
#idx("adjoinset", sub: "ordered-linked-list representation")
#py("adjoin_set")
باستخدام التمثيل المرتب. وعلى غرار #py("is_element_of_set")، أظهر كيفية الاستفادة من الترتيب لإنتاج دالة تتطلب في المتوسط حوالي نصف عدد الخطوات مقارنة بالتمثيل غير المرتب.

#anchor(<ex:2_61>)
])

#exercise(label-name: <ex:union-set>, [
قدم تنفيذاً بـ $Theta (n)$ لـ
#idx("unionset", sub: "ordered-linked-list representation")
#py("union_set")
للمجموعات الممثلة كقوائم مترابطة مرتبة.
])

#idx("set", sub: "represented as ordered linked list")
#idx("ordered-linked-list representation of sets")

#subheading([المجموعات كأشجار ثنائية])

#idx("set", sub: "represented as binary tree")
#idx("binary tree", sub: "set represented as")
#idx("tree", sub: "binary")
#idx("binary tree")
#idx("binary search")
#idx("search", sub: "of binary tree")

يمكننا أداء ما هو أفضل من تمثيل القوائم المترابطة المرتبة عن طريق ترتيب عناصر المجموعة في شكل شجرة. تحوي كل عقدة في الشجرة عنصراً واحداً من المجموعة، يُسمى "العنصر المدخل" (#en[entry]) في تلك العقدة، ووصلة إلى كل من عقدتين أخريين (قد تكونان فارغتين). تشير الوصلة "اليسرى" إلى عناصر أصغر من العنصر الموجود في العقدة، وتشير الوصلة "اليمين" إلى عناصر أكبر من العنصر الموجود في العقدة.
يُظهر الشكل @fig:binary-tree بعض الأشجار التي تمثل المجموعة $\{1,3,5,7,9,11\}$. ويمكن تمثيل المجموعة نفسها بوساطة شجرة بعدة طرق مختلفة. والشيء الوحيد الذي نطلبه للتمثيل الصالح هو أن تكون جميع العناصر في الشجرة الفرعية اليسرى أصغر من عنصر العقدة وأن تكون جميع العناصر في الشجرة الفرعية اليمنى أكبر.

#sicp-figure(image("/images/img_original/ch2-Z-G-51.svg", width: 70%), caption: [أشجار ثنائية مختلفة تمثل المجموعة $\{ 1,3,5,7,9,11 \}$.], label-name: <fig:binary-tree>)

ميزة تمثيل الشجرة هي التالية: نفترض أننا نريد الفحص عما إذا كان العدد $x$ محتواً في مجموعة. نبدأ بمقارنة $x$ مع العنصر في العقدة العلوية. فإذا كان $x$ أقل من هذا، فإننا نعلم أننا بحاجة فقط إلى البحث في الشجرة الفرعية اليسرى؛ وإذا كان $x$ أكبر، فسنحتاج فقط إلى البحث في الشجرة الفرعية اليمنى. والآن، إذا كانت الشجرة "متوازنة"، فستكون كل واحدة من هذه الأشجار الفرعية نصف حجم الشجرة الأصلية تقريباً. وبالتالي، اختزلنا في خطوة واحدة مشكلة البحث في شجرة بحجم $n$ إلى البحث في شجرة بحجم $n/2$. وبما أن حجم الشجرة يتنصف في كل خطوة، ففي وسعنا التوقع أن عدد الخطوات اللازمة للبحث في شجرة بحجم $n$ ينمو كـ $Theta ( log n)$.#footnote[تنصيف حجم المشكلة في كل خطوة هو الخاصية المميزة للنمو اللوغاريتمي (#idx("logarithmic growth"))، كما رأينا مع خوارزمية الرفع إلى قوة السريعة في القسم @sec:exponentiation وطريقة البحث في نصف النطاق في القسم @sec:proc-general-methods.] وبالنسبة للمجموعات الكبيرة، سيكون هذا تسريعاً كبيراً مقارنة بالتمثيلات السابقة.

يمكننا تمثيل الأشجار باستخدام القوائم المترابطة (#idx("binary tree", sub: "represented with linked lists")).
ستكون كل عقدة عبارة عن قائمة مترابطة من ثلاثة عناصر: العنصر المدخل في العقدة، والشجرة الفرعية اليسرى، والشجرة الفرعية اليمنى. وستشير الشجرة الفرعية اليسرى أو اليمنى القادمة من قائمة مترابطة فارغة إلى عدم وجود شجرة فرعية متصلة هناك. ويمكننا وصف هذا التمثيل بالدوال التالية:#footnote[نحن نمثل المجموعات بدلالة الأشجار، والأشجار بدلالة القوائم المترابطة—في الواقع، تجريد بيانات مبني على تجريد بيانات. ويمكننا اعتبار الدوال #py("entry") و #py("left_branch") و #py("right_branch") و #py("make_tree") كطريقة لعزل تجريد "الشجرة الثنائية" عن الطريقة المحددة التي قد نرغب بها في تمثيل مثل هذه الشجرة بدلالة بنية القوائم المترابطة.]
#idx("entry", decl: true)#idx("leftbranch", decl: true)#idx("rightbranch", decl: true)#idx("maketree", decl: true)
#snippet(```python
def entry(tree): return head(tree)

def left_branch(tree): return head(tail(tree))

def right_branch(tree): return head(tail(tail(tree)))

def make_tree(entry, left, right):
    return llist(entry, left, right)
```)

الآن يمكننا كتابة #py("is_element_of_set") باستخدام الاستراتيجية الموصوفة أعلاه:
#idx("iselementofset", sub: "binary-tree representation", decl: true)
#syntax("
def is_element_of_set(x, set):
    return (False
            if is_none(set)
            else True
            if x == entry(set)
            else is_element_of_set(x, left_branch(set))
            if x < entry(set)
            # ", $mono("x > entry(set)")$, "
            else is_element_of_set(x, right_branch(set)))
      ")

ويتم تنفيذ إلحاق عنصر بمجموعة بطريقة مماثلة ويتطلب أيضاً $Theta ( log n)$ خطوات. ولإلحاق عنصر #py("x")، نقارن #py("x") مع عنصر العقدة لتحديد ما إذا كان ينبغي إضافة #py("x") إلى الفرع الأيمن أم إلى الفرع الأيسر، وبعد أن نلحق #py("x") بالفرع المناسب، نجمع هذا الفرع المنشأ حديثاً مع العنصر الأصلي والفرع الآخر. وإذا كان #py("x") مساوياً للعنصر المدخل، فنرجع العقدة فقط. وإذا طُلب منا إلحاق #py("x") بشجرة فارغة، فنولد شجرة تملك #py("x") كعنصر مدخل وفروعاً يمنى ويسرى فارغة. إليك الدالة:

#idx("adjoinset", sub: "binary-tree representation", decl: true)
#syntax("
def adjoin_set(x, set):
    return (make_tree(x, None, None)
            if is_none(set)
            else set
            if x == entry(set)
            else make_tree(entry(set),
                           adjoin_set(x, left_branch(set)),
                           right_branch(set))
            if x < entry(set)
            # ", $mono("x > entry(set)")$, "
            else make_tree(entry(set),
                           left_branch(set),
                           adjoin_set(x, right_branch(set))))
      ")

يعتمد الادعاء أعلاه بأنه يمكن إنجاز البحث في الشجرة في عدد لوغاريتمي من الخطوات على افتراض أن الشجرة #idx("balanced binary tree") #idx("binary tree", sub: "balanced") "متوازنة"، أي أن الشجرة الفرعية اليسرى واليمنى لكل شجرة تمتلكان تقريباً العدد نفسه من العناصر، بحيث تحوي كل شجرة فرعية حوالي نصف عناصر والديها. ولكن كيف يمكننا التأكد من أن الأشجار التي نبنيها ستكون متوازنة؟ حتى لو بدأنا بشجرة متوازنة، فإن إضافة عناصر مع #py("adjoin_set") قد تنتج نتيجة غير متوازنة. وبما أن موضع عنصر ملحق حديثاً يعتمد على كيفية مقارنة العنصر مع العناصر الموجودة بالفعل في المجموعة، فيمكننا التوقع أنه إذا أضفنا عناصر "عشوائياً"، فستتجه الشجرة إلى التوازن في المتوسط. ولكن هذا ليس ضماناً. فعلى سبيل المثال، إذا بدأنا بمجموعة فارغة وألحقنا الأعداد من 1 إلى 7 بالتتابع، فسينتهي بنا الأمر بالشجرة غير المتوازنة للغاية الموضحة في الشكل @fig:unbalanced-tree. وفي هذه الشجرة تكون جميع الأشجار الفرعية اليسرى فارغة، لذا لا تملك ميزة تفوق قائمة مترابطة مرتبة بسيطة.
وإحدى الطرق لحل هذه المشكلة هي تعريف عملية تحول شجرة اعتباطية إلى شجرة متوازنة بالعناصر نفسها. ويمكننا حينئذ إنجاز هذا التحويل بعد كل بضع عمليات #py("adjoin_set") للحفاظ على توازن مجموعتنا. وتوجد أيضاً طرق أخرى لحل هذه المشكلة، ويتضمن معظمها تصميم بنى بيانات جديدة يمكن فيها إجراء كل من البحث والإدراج في $Theta ( log n)$ خطوة.#footnote[تتضمن الأمثلة على مثل هذه البنى #idx("tree", sub: "B-tree") #idx("tree", sub: "red-black") #idx("B-tree") #idx("red-black tree") #emph[أشجار B] و #emph[أشجار الحمر والسود]. وهناك مؤلفات واسعة في بنى البيانات المخصصة لهذه المشكلة. انظر #idx("Cormen, Thomas H.") #idx("Leiserson, Charles E.") #idx("Rivest, Ronald L.") #idx("Stein, Clifford") #en[Cormen] و #en[Leiserson] و #en[Rivest] و #en[Stein] (2022).]

#exercise(label-name: <ex:tree-to-list>, [
تحول كل واحدة من الدالتين التاليتين الشجرة الثنائية إلى قائمة مترابطة (#idx("binary tree", sub: "converting to a linked list") #idx("linked list", sub: "converting a binary tree to a")).
#idx("treetolinkedlist…", decl: true)
#snippet(```python
def tree_to_linked_list_1(tree):
    return (None
            if is_none(tree)
            else append(tree_to_linked_list_1(left_branch(tree)),
                        pair(entry(tree),
                             tree_to_linked_list_1(right_branch(tree)))))
```)

#snippet(```python
def tree_to_linked_list_2(tree):
    def copy_to_linked_list(tree, result_list):
        return (result_list
                if is_none(tree)
                else copy_to_linked_list(left_branch(tree),
                         pair(entry(tree),
                              copy_to_linked_list(right_branch(tree),
                                           result_list))))
    return copy_to_linked_list(tree, None)
```)

+ هل تنتج الدالتان النتيجة نفسها لكل شجرة؟ إذا لم تكن كذلك، فكيف تختلف النتائج؟ ما هي القوائم المترابطة التي تنتجها الدالتان للأشجار في الشكل @fig:binary-tree؟
+ هل تملك الدالتان رتبة النمو نفسها في عدد الخطوات المطلوبة لتحويل شجرة متوازنة بـ $n$ من العناصر إلى قائمة مترابطة؟ وإذا لم تكن كذلك، فأيهما تنمو بشكل أبطأ؟
])

#sicp-figure(image("/images/img_original/ch2-Z-G-52.svg", width: 70%), caption: [شجرة غير متوازنة ناتجة عن إلحاق الأعداد من 1 إلى 7 بالتتابع.], label-name: <fig:unbalanced-tree>)

#exercise(label-name: <ex:list-to-tree>, [
تحول الدالة التالية #py("linked_list_to_tree") (#idx("binary tree", sub: "converting a linked list to a") #idx("linked list", sub: "converting to a binary tree")) قائمة مترابطة مرتبة إلى شجرة ثنائية متوازنة. وتأخذ الدالة المساعدة #py("partial_tree") كمعاملات عدداً صحيحاً $n$ وقائمة مترابطة من $n$ من العناصر على الأقل، وتبني شجرة متوازنة تحوي أول $n$ من عناصر القائمة المترابطة. والنتيجة المعادة من #py("partial_tree") هي زوج (مكوَّن بـ #py("pair")) يكون رأسه (#py("head")) هو الشجرة المبنية وذيله (#py("tail")) هو القائمة المترابطة من العناصر غير المضمنة في الشجرة.
#idx("linkedlisttotree", decl: true)
#snippet(```python
def linked_list_to_tree(elements):
    return head(partial_tree(elements, length(elements)))
def partial_tree(elts, n):
    if n == 0:
        return pair(None, elts)
    else:
        left_size = (n - 1) // 2
        left_result = partial_tree(elts, left_size)
        left_tree = head(left_result)
        non_left_elts = tail(left_result)
        right_size = n - (left_size + 1)
        this_entry = head(non_left_elts)
        right_result = partial_tree(tail(non_left_elts), right_size)
        right_tree = head(right_result)
        remaining_elts = tail(right_result)
        return pair(make_tree(this_entry, left_tree, right_tree),
                    remaining_elts)
```)

+ اكتب فقرة قصيرة تشرح فيها بأوضح ما يمكنك كيفية عمل #py("partial_tree"). وارسم الشجرة المنتجة بوساطة #py("linked_list_to_tree") للقائمة المترابطة #py("llist(1, 3, 5, 7, 9, 11)").
+ ما هي رتبة النمو في عدد الخطوات التي تتطلبها #py("linked_list_to_tree") لتحويل قائمة مترابطة من $n$ من العناصر؟
])

#exercise(label-name: <ex:tree-ops>, [
استخدم نتائج التمرينين @ex:tree-to-list و @ex:list-to-tree لتقديم تنفيذات بـ $Theta (n)$ لـ
#idx("unionset", sub: "binary-tree representation")
#py("union_set")
و
#idx("intersectionset", sub: "binary-tree representation")
#py("intersection_set")
للمجموعات المنفذة كـ أشجار ثنائية (متوازنة).#footnote[التمارين @ex:tree-to-list–@ex:tree-ops تعود إلى #idx("Hilfinger, Paul") #en[Paul Hilfinger].]
])

#idx("set", sub: "represented as binary tree")
#idx("binary tree", sub: "set represented as")

#subheading([المجموعات واسترجاع المعلومات])

لقد فحصنا خيارات استخدام القوائم المترابطة لتمثيل المجموعات ورأينا كيف يمكن لاختيار التمثيل لكائن بيانات أن يملك تأثيراً كبيراً على أداء البرامج التي تستخدم البيانات. وسبب آخر للتركيز على المجموعات هو أن التقنيات المناقشة هنا تظهر مراراً وتكراراً في التطبيقات التي تتضمن استرجاع المعلومات.

#idx("data base", sub: "as set of records")
#idx("set", sub: "data base as")
تأمل قاعدة بيانات تحتوي على عدد كبير من السجلات الفردية (#idx("record, in a data base"))، مثل ملفات الموظفين لشركة أو المعاملات في نظام محاسبي. يقضي نظام إدارة البيانات النموذجي مقداراً كبيراً من الوقت في الوصول إلى البيانات في السجلات أو تعديلها، وبالتالي فإنه يتطلب طريقة كفؤة للوصول إلى السجلات. ويتم ذلك عن طريق تحديد جزء من كل سجل ليخدم كـ #idx("key of a record", sub: "in a data base") #emph[مفتاح] (#en[key]) معرف. ويمكن أن يكون المفتاح أي شيء يحدد السجل بشكل فريد. بالنسبة لملف الموظفين، قد يكون رقم تعريف الموظف. وبالنسبة لنظام المحاسبة، قد يكون رقم المعاملة. ومهما كان المفتاح، فعندما نعرّف السجل كبنية بيانات، يجب أن نضمن دالة محدد اختيارات #idx("key") #py("key") تسترجع المفتاح المرتبط بسجل معطى.

والآن نمثل قاعدة البيانات كمجموعة من السجلات. ولتحديد موقع السجل ذي المفتاح المعطى، نستخدم دالة #py("lookup")، والتي تأخذ كمعاملات مفتاحاً وقاعدة بيانات، وترجع السجل الذي يحوي ذلك المفتاح، أو #py("False") إذا لم يكن هناك مثل هذا السجل.
وتُنفذ دالة #py("lookup") بالطريقة نفسها تقريباً لـ #py("is_element_of_set").
فعلى سبيل المثال، إذا كانت مجموعة السجلات منفذة كقائمة مترابطة غير مرتبة، فيمكننا استخدام:

#idx("lookup", sub: "in set of records", decl: true)
#snippet(```python
def lookup(given_key, set_of_records):
    return (False
            if is_none(set_of_records)
            else head(set_of_records)
            if given_key == key(head(set_of_records))
            else lookup(given_key, tail(set_of_records)))
```)

بالطبع، هناك طرق أفضل لتمثيل المجموعات الكبيرة من القوائم المترابطة غير المرتبة.
فأنظمة استرجاع المعلومات التي يتوجب فيها الوصول إلى السجلات "عشوائياً" تُنفذ عادةً بطريقة قائمة على الأشجار، مثل تمثيل الشجرة الثنائية المناقش سابقاً.
وفي تصميم مثل هذا النظام، يمكن لـ منهجية تجريد البيانات أن توفر مساعدة كبيرة. إذ يمكن للمصمم إنشاء تنفيذ أولي باستخدام تمثيل بسيط ومباشر مثل القوائم المترابطة غير المرتبة.
وسيكون هذا غير مناسب للنظام النهائي، ولكنه يمكن أن يكون مفيداً في توفير قاعدة بيانات "سريعة مؤقتة" يمكن فحص بقية النظام بها. ولاحقاً، يمكن تعديل تمثيل البيانات ليكون أكثر تطوراً. وإذا جرى الوصول إلى قاعدة البيانات بدلالة محددات اختيارات وبوانٍ مجردة، فإن هذا التغيير في التمثيل لن يتطلب أي تغييرات على بقية النظام.

#exercise(label-name: <ex:set-lookup-binary-tree>, [
نفذ دالة #py("lookup") للحالة التي تتكون فيها مجموعة السجلات كشجرة ثنائية، مرتبة حسب القيم العددية للمفاتيح.
])
