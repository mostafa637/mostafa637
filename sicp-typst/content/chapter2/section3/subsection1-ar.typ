// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([السلاسل النصية])

#anchor(<sec:strings>)

#idx("string(s)")

حتى الآن، استخدمنا السلاسل النصية من أجل عرض الرسائل، باستخدام الدالتين #py("display") و #py("error") (كما في التمرين @ex:search-for-primes على سبيل المثال).
ويمكننا تشكيل بيانات مركبة باستخدام السلاسل النصية وأن نملك قوائم مترابطة مثل:
#idx("Wrigstad, Tobias, daughter of")
#idx("Henz, Martin, children of")

#syntax("
llist(\"a\", \"b\", \"c\", \"d\")
llist(23, 45, 17)
llist(llist(\"Jakob\", 27), llist(\"Lova\", 9), llist(\"Luisa\", 24))
          ")

وللتمييز بين السلاسل النصية والأسماء، نحيطها
#idx("quotation marks", sub: "double")
#idx("\" (double quote)")
بعلامات تنصيص مزدوجة. فعلى سبيل المثال، يدل التعبير في #en[Python] #py("z") على قيمة الاسم #py("z")، في حين يدل التعبير في #en[Python] #py("\"z\"") على سلسلة نصية تتكون من محرف واحد، وهو الحرف الأخير في الأبجدية الإنجليزية بالحرف الصغير.

عن طريق علامات التنصيص، يمكننا التمييز بين السلاسل النصية والأسماء:

#snippet(```python
a = 1
b = 2
```)

#snippet(```python
print(llist(a, b))
```)

#output(```python
print(llist(a, b))
```)

#snippet(```python
print(llist("a", "b"))
```)

#output(```python
print(llist("a", "b"))
```)

#snippet(```python
print(llist("a", b))
```)

#output(```python
print(llist("a", b))
```)

في القسم @sec:conditionals قدمنا #py("==") و #py("!=") كمحمولات أولية على الأعداد.
#idx("equality", sub: "of strings")
#idx("==", sub: "as string comparison operator")

#idx("!=", sub: "as string comparison operator", sort: ";4")

ومن الآن فصاعداً، سنسمح بسلسلتين نصيتين كوسيطتين لـ #py("==") و #py("!="). يرجع المحمول #py("==") #py("True") إذا وفقط إذا كانت السلسلتان النصيتان متماثلتين، ويرجع #py("!=") #py("True") إذا وفقط إذا لم تكن السلسلتان النصيتان متماثلتين.#footnote[يمكننا اعتبار سلسلتين نصيتين "متماثلتين" إذا كانتا تتكونان من المحارف نفسها وبالترتيب نفسه. ومثل هذا التعريف يتجاوز مسألة عميقة لسنا مستعدين للتعامل معها بعد: معنى "التماثل" في لغة البرمجة. وسوف نعود إلى هذا في الفصل @chap:state (القسم @sec:costs-of-assignment).]
باستخدام #py("==")، يمكننا تنفيذ دالة مفيدة تسمى #py("member").
تأخذ هذه الدالة وسيطين: سلسلةً نصية وقائمة مترابطة من السلاسل النصية، أو عدداً وقائمة مترابطة من الأعداد.
وإذا كان الوسيط الأول غير محتوى في القائمة المترابطة (أي ليس مساوياً بـ #py("==") لأي عنصر في القائمة المترابطة)، فإن #py("member") ترجع #py("None"). وخلاف ذلك، ترجع القائمة الفرعية من القائمة المترابطة بدءاً من التواجد الأول للسلسلة النصية أو العدد:
#idx("member", decl: true)
#snippet(```python
def member(item, x):
    return (None if is_none(x)
            else x if item == head(x)
            else member(item, tail(x)))
```)

فعلى سبيل المثال، قيمة

#snippet(```python
print(member("apple", llist("pear", "banana", "prune")))
```)

هي #py("None")، في حين أن قيمة

#snippet(```python
print(member("apple", llist("x", "y", "apple", "pear")))
```)

هي #py("llist(\"apple\", \"pear\")").

#anchor(<ex:equal->)
يُقال عن قائمتين مترابطتين أنهما
#idx("equal")
#idx("equality", sub: "of linked lists")
#idx("structural equality")
#idx("equality", sub: "structural")
#idx("equality", sub: "of numbers")
#idx("equality", sub: "of strings")
#idx("linked list", sub: "equality of")

#idx("==", sub: "as general comparison operator")
#emph[متساويتان]
إذا كانتا تحتويان على عناصر متساوية مرتبة بالترتيب نفسه، ويدعم عامل #py("==") في #en[Python] هذا المفهوم من #emph[المساواة الهيكلية] (#en[structural equality]).
فعلى سبيل المثال،

#snippet(```python
llist("this", "is", "a", "linked", "list") == llist("this", "is", "a", "linked", "list")
```)

هو #py("True")، ولكن

#snippet(```python
llist("this", "is", "a", "linked", "list") == llist("this", llist("is", "a"), "linked", "list")
```)

#idx("number(s)", sub: "equality of")
#idx("string(s)", sub: "equality of")
هو #py("False"). ولنكون أكثر دقة، تعرّف #en[Python] عامل المساواة #py("==") تعاودياً بدلالة المساواة الأساسية #py("==") للأعداد والسلاسل النصية بالقول إن #py("a") و #py("b") متساويان إذا كانا كلاهما سلسلتين نصيتين أو كلاهما عددين وهما متساويان، أو إذا كانا كلاهما زوجين بحيث #py("head(a)") مساوٍ لـ #py("head(b)") و #py("tail(a)") مساوٍ لـ #py("tail(b)"). وتستخدم دالة #py("member") أعلاه #py("==") وبالتالي تفحص المساواة الهيكلية.

#exercise(label-name: <ex:2_53>, [
ما هي نتيجة تقييم كل من التعبيرات التالية، بالترميز الصندوقي وبترميز القائمة المترابطة؟

#snippet(```python
llist("a", "b", "c")
```)

#snippet(```python
llist(llist("george"))
```)

#snippet(```python
tail(llist(llist("x1", "x2"), llist("y1", "y2")))
```)

#snippet(```python
tail(head(llist(llist("x1", "x2"), llist("y1", "y2"))))
```)

#snippet(```python
member("red", llist("blue", "shoes", "yellow", "socks"))
```)

#snippet(```python
member("red", llist("red", "shoes", "blue", "socks"))
```)
])

#exercise([
نفذ دالة #py("equal") تتصرف مثل #py("==") تماماً، بحيث تطبق #py("equal") العامل #py("==") فقط على الأعداد والسلاسل النصية.
])

#exercise(label-name: <ex:double-quotation>, [
يقرأ مفسر #en[Python] المحارف بعد علامة التنصيص المزدوجة #idx("\" (double quote)") #py("\"") حتى يجد علامة تنصيص مزدوجة أخرى. وجميع المحارف بين العلامتين هي جزء من السلسلة النصية، باستثناء علامتي التنصيص المزدوجتين أنفسهما. ولكن ماذا لو أردنا أن تحتوي سلسلة نصية على علامات تنصيص مزدوجة؟ لهذا الغرض، تسمح #en[Python] أيضاً بـ #idx("quotation marks", sub: "single") #idx("' (single quote)") علامات التنصيص #emph[المفردة] لتحديد السلاسل النصية، كما في #py("'say your name aloud'") على سبيل المثال.
وداخل السلاسل النصية ذات المحارف المفردة، يمكننا استخدام علامات تنصيص مزدوجة، والعكس بالعكس، لذا فإن #py("'say \"your name\" aloud'") و #py("\"say 'your name' aloud\"") هما سلسلتان نصيتان صالحتان تملكان محارف مختلفة في الموضعين 4 و 14 إذا بدأنا العد من 0. وبناءً على الخط المستخدم، قد لا يسهل تمييز علامتي تنصيص مفردتين عن علامة تنصيص مزدوجة. هل يمكنك اكتشاف أيها أيهما واستنتاج قيمة التعبير التالي؟

#snippet(```python
print('"' == "")
```)
])

#idx("string(s)")
