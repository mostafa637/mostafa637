// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([أمثلة على البرامج غير الحتمية], label-name: <sec:amb-examples>)

صف القسم @sec:amb-implementation تنفيذ مُقَيِّم #py("amb"). ولكن أولاً، نعطي بعض الأمثلة حول كيفية استخدامه. وتتمثل ميزة البرمجة غير الحتمية في أننا نتمكن من إخفاء تفاصيل كيفية إجراء البحث، وبذلك نعبر عن برامجنا بمستوى أعلى من
#idx("تجريد", sub: "البحث في البرمجة غير الحتمية")
التجريد.

#subheading([أحاجي المنطق])

#idx("أحاجي", sub: "أحاجي المنطق")
#idx("أحاجي المنطق")
#idx("برامج غير حتمية", sub: "أحاجي المنطق")

اللغز التالي (المأخوذ والمعدل من
#idx("Dinesman, Howard P.")
#en[Dinesman 1968])
هو نموذج لفئة كبيرة من أحاجي المنطق البسيطة:

#blockquote[شركة البرمجيات
#idx("Gargle")
#en[Gargle] تتوسع، و#en[Alyssa] و#en[Ben] و#en[Cy] و#en[Lem] و#en[Louis] يتنتقلون إلى صف من خمسة مكاتب خاصة في مبنى جديد. أليسا لا تنتقل إلى المكتب الأخير. وبن لا ينتقل إلى المكتب الأول. وساي لا يأخذ المكتب الأول ولا المكتب الأخير. وليـم ينتقل إلى مكتب بعد مكتب بن. ومكتب لويس ليس مجاورًا لمكتب ساي. ومكتب ساي ليس مجاورًا لمكتب بن. مَن ينتقل إلى أي مكتب؟]

يمكننا تحديد مَن ينتقل إلى أي مكتب بطريقة مباشرة عن طريق تسديد جميع الاحتمالات وفرض القيود المعطاة:#footnote[يستخدم برنامجنا الدالة التالية لتحديد ما إذا كانت عناصر القائمة متميزة:

#idx("distinct", decl: true)
#snippet(```python
def distinct(items):
    return True if is_none(items) else True if is_none(tail(items)) else distinct(tail(items)) if is_none(member(head(items), tail(items))) else False
```)]
#idx("officemove", decl: true)
#snippet(```python
def office_move():
    alyssa = amb(1, 2, 3, 4, 5)
    ben = amb(1, 2, 3, 4, 5)
    cy = amb(1, 2, 3, 4, 5)
    lem = amb(1, 2, 3, 4, 5)
    louis = amb(1, 2, 3, 4, 5)
    require(distinct(llist(alyssa, ben, cy, lem, louis)))
    require(alyssa != 5)
    require(ben != 1)
    require(cy != 5)
    require(cy != 1)
    require(lem > ben)
    require(abs(louis - cy) != 1)
    require(abs(cy - ben) != 1)
    return llist(llist("alyssa", alyssa), llist("ben", ben), llist("cy", cy), llist("lem", lem), llist("louis", louis))
```)

تقييم التعبير
#py("office_move()")
ينتج النتيجة

#snippet(```python
llist(llist("alyssa", 3), llist("ben", 2), llist("cy", 4),
     llist("lem", 5), llist("louis", 1))
```)

على الرغم من أن هذه الدالة البسيطة تعمل، إلا أنها بطيئة جدًا. وتناقش التمارين @ex:better-office-move1 و @ex:better-office-move2 بعض التحسينات الممكنة.

#exercise(label-name: <ex:office_move_1>, [
عدّل دالة انتقال المكاتب لإسقاط الشرط القائل بأن مكتب لويس ليس مجاورًا لمكتب ساي. كم عدد الحلول لهذه الأحجية المعدلة؟
])

#exercise(label-name: <ex:better-office-move1>, [
هل يغير ترتيب القيود في دالة انتقال المكاتب الإجابة؟ وهل يؤثر على الوقت المستغرق للعثور على إجابة؟ إذا كنت تعتقد أن هذا يفرق، فاعرض برنامجًا أسرع مُحَصَّلاً من البرنامج المعطى عن طريق إعادة ترتيب القيود. وإذا كنت تعتقد أن هذا لا يفرق، فقدم حجتك.
])

#exercise(label-name: <ex:better-office-move2>, [
في مشكلة انتقال المكاتب، كم عدد مجموعات التعيينات الموجودة للأشخاص في المكاتب، قبل وبعد الشرط القائل بأن تعيينات المكاتب متميزة؟ إن توليد جميع التعيينات الممكنة للأشخاص في المكاتب ثم ترك الأمر للتتبع التراجعي لاستبعادها أمر غير كفء للغاية. على سبيل المثال، تعتمد معظم القيود على اسم أو اسمين فقط من أسماء الأشخاص والمكاتب، وبالتالي يمكن فرضها قبل تحديد المكاتب لجميع الأشخاص. اكتب ووضح دالة غير حتمية أكثر كفاءة بكثير تحل هذه المشكلة بناءً على توليد الاحتمالات التي لم تُستبعد بالفعل بواسطة القيود السابقة فقط.
])

#exercise(label-name: <ex:office_move_4>, [
اكتب برنامج
#idx("برمجة غير حتمية مقابل برمجة بايثون")
#en[Python] عاديًا لحل أحجية انتقال المكاتب.
])

#exercise(label-name: <ex:liars>, [
حل أحجية «الكاذبين» التالية (المأخوذة والمعدلة من
#idx("Phillips, Hubert")
#en[Phillips 1934]):

#blockquote[يلتقي أليسا وساي وإيفا وليم ولويس لتناول غداء عمل في مطعم الخدمة المتوسطة. وتصل وجباتهم واحدة تلو الأخرى، بعد وقت طويل من تقديم طلباتهم. ولتسلية بن، الذي ينتظر عودتهم إلى المكتب لحضور اجتماع، يقرر كل منهم الإدلاء بعبارة واحدة صادقة وعبارة واحدة كاذبة حول وجباتهم:

- أليسا: «وصلت وجبة ليم ثانية. ووصلت وجبتي ثالثة.»
- ساي: «وصلت وجبتي أولاً. ووصلت وجبة إيفا ثانية.»
- إيفا: «وصلت وجبتي ثالثة، ووصلت وجبة المسكين ساي أخيرًا.»
- ليم: «وصلت وجبتي ثانية. ووصلت وجبة لويس رابعة.»
- لويس: «وصلت وجبتي رابعة. ووصلت وجبة أليسا أولاً.»

ما هو الترتيب الحقيقي الذي تلقى به رواد المطعم الخمسة وجباتهم؟]
])

#exercise(label-name: <ex:checking>, [
استخدم مُقَيِّم #py("amb") لحل الأحجية التالية (المأخوذة والمعدلة من
#idx("Phillips, Hubert")
#en[Phillips 1961]):

#blockquote[يختار كل من أليسا وبن وساي وإيفا ولويس فصلًا مختلفًا من كتاب #en[SICP JS] ويحلون جميع التمارين في ذلك الفصل. فيحل لويس تمارين فصل «الدوال»، وأليسا تمارين فصل «البيانات»، وساي تمارين فصل «الحالة». ويقررون فحص عمل بعضهم البعض، وتتطوع أليسا لفحص التمارين في فصل «التجريد فوق اللغوي». ويقوم بن بحل التمارين في فصل «آلات المسجلات» ويفحصها لويس. والشخص الذي يفحص التمارين في فصل «الدوال» يحل التمارين التي تفحصها إيفا. مَن يفحص التمارين في فصل «البيانات»؟]

حاول كتابة البرنامج بحيث يعمل بكفاءة (انظر التمرين @ex:better-office-move2). وحدد أيضًا كم عدد الحلول الموجودة إذا لم يُقَل لنا إن أليسا تفحص التمارين في فصل «التجريد فوق اللغوي».
])

#idx("أحاجي", sub: "أحاجي المنطق")
#idx("أحاجي المنطق")
#idx("برامج غير حتمية", sub: "أحاجي المنطق")

#exercise(label-name: <ex:queens_amb>, [
وصف التمرين @ex:8queens
#idx("شطرنج، أحجية الملكات الثماني")
#idx("أحجية الملكات الثماني")
#idx("أحاجي", sub: "أحجية الملكات الثماني")
#idx("برمجة غير حتمية مقابل برمجة بايثون")
«أحجية الملكات الثماني» المتمثلة في وضع الملكات على رقعة الشطرنج بحيث لا تهاجم أي منهن الأخرى. اكتب برنامجًا غير حتمي لحل هذه الأحجية.
])

#subheading([تحليل اللغة الطبيعية])

#idx("تحليل اللغة الطبيعية")
#idx("برامج غير حتمية", sub: "تحليل اللغة الطبيعية")

تبدأ البرامج المصممة لقبول اللغة الطبيعية كمدخلات عادةً بـ #emph[إعراب] المدخلات (أو تحليلها نحويًا)، أي مطابقة المدخلات مع هيكل نحوي ما. على سبيل المثال، قد نحاول التعرف على الجمل البسيطة المكونة من أداة متبوعة باسم متبوع بفعل، مثل "The cat eats" («القط يأكل»). ولتحقيق مثل هذا التحليل، يجب أن نكون قادرين على تحديد أجزاء الكلام للكلمات الفردية. ويمكننا البدء ببعض القوائم التي تصنف الكلمات المختلفة:#footnote[نستخدم هنا العرف القائل بأن العنصر الأول في كل قائمة يحدد جزء الكلام لباقي الكلمات في القائمة.]
#idx("nouns", decl: true)#idx("verbs", decl: true)#idx("articles", decl: true)
#snippet(```python
nouns = llist("noun", "student", "professor", "cat", "class")
verbs = llist("verb", "studies", "lectures", "eats", "sleeps")
articles = llist("article", "the", "a")
```)

ونحتاج أيضًا إلى
#idx("نحو")
#emph[قواعد نحوية] (#en[grammar])، أي مجموعة من القواعد التي تصف كيفية تركيب العناصر النحوية من عناصر أبسط. قد تنص قواعد نحوية بسيطة جدًا على أن الجملة تتكون دائمًا من جزءين — مركب اسمي متبوعًا بفعل — وأن المركب الاسمي يتكون من أداة متبوعة باسم. ومع هذه القواعد النحوية، يتم تحليل الجملة "The cat eats" على النحو التالي:

#snippet(```python
llist("sentence",
     llist("noun-phrase", llist("article", "the"), llist("noun", "cat"),
     llist("verb", "eats"))
```)

#idx("parse...")

يمكننا توليد هذا التحليل باستخدام برنامج بسيط يملك دوالاً منفصلة لكل قاعدة من القواعد النحوية. ولتحليل جملة، نحدد جزءيها المكونين ونرجع قائمة بهذين العنصرين، مَوسُومَين بالرمز #py("sentence"):

#snippet(```python
def parse_sentence():
    return llist("sentence", parse_noun_phrase(), parse_word(verbs))
```)

وبالمثل، يتم تحليل المركب الاسمي عن طريق العثور على أداة متبوعة باسم:

#snippet(```python
def parse_noun_phrase():
    return llist("noun-phrase", parse_word(articles), parse_word(nouns))
```)

في أدنى مستوى، يتلخص التحليل في التحقق المتكرر من أن الكلمة التالية التي لم تُحَلَّل بعد هي عضو في قائمة الكلمات لجزء الكلام المطلوب. ولتنفيذ ذلك، نحتفظ بمتغير عام #py("not_yet_parsed")، وهو المدخل الذي لم يُحَلَّل بعد. وفي كل مرة نتحقق فيها من كلمة، نشترط أن تكون #py("not_yet_parsed") غير فارغة وأن تبدأ بكلمة من القائمة المحددة. وإذا كان الأمر كذلك، فإننا نزيل تلك الكلمة من #py("not_yet_parsed") ونرجع الكلمة جنبًا إلى جنب مع جزء الكلام الخاص بها (الذي نجد في رأس القائمة):#footnote[لاحظ أن #py("parse_word") تستخدم الإسناد لتعديل قائمة المدخلات التي لم تُحَلَّل بعد. ولكي يعمل هذا، يجب على مُقَيِّم #py("amb") إلغاء آثار الإسنادات عندما يتراجع.]

#snippet(```python
def parse_word(word_list):
    require( not is_none(not_yet_parsed))
    require( not is_none(member(head(not_yet_parsed), tail(word_list))))
    found_word = head(not_yet_parsed)
    not_yet_parsed = tail(not_yet_parsed)
    return llist(head(word_list), found_word)
```)

لبدء التحليل، كل ما نحتاج إلى فعله هو تعيين #py("not_yet_parsed") لتكون الإدخال بالكامل، ومحاولة تحليل جملة، والتحقق من عدم بفي أي شيء:

#snippet(```python
not_yet_parsed = None
```)

#snippet(```python
def parse_input(input):
    not_yet_parsed = input
    sent = parse_sentence()
    require(is_none(not_yet_parsed))
    return sent
```)

يمكننا الآن تجربة المحلل والتحقق من أنه يعمل لجملة الاختبار البسيطة لدينا:

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
parse_input(llist("the",  "cat",  "eats"))
```)

#output(```python
parse_input(llist("the", "cat", "eats"))
```)

مُقَيِّم #py("amb") مفيد هنا لأنه من المريح التعبير عن قيود التحليل بمساعدة #py("require"). والبحث التلقائي والتتبع التراجعي يؤديان إلى نتائج مثمرة حقًا عندما نأخذ بالاعتبار قواعد نحوية أكثر تعقيدًا حيث توجد خيارات لكيفية تفكيك الوحدات.

لنضف إلى قواعدنا النحوية قائمة بحروف الجر:
#idx("prepositions", decl: true)
#snippet(```python
prepositions = llist("prep", "for", "to", "in", "by", "with")
```)

ونعرف الجملة الجرية (مثل "for the cat") على أنها حرف جر متبوع بمركب اسمي:

#snippet(```python
def parse_prepositional_phrase():
    return llist("prep-phrase", parse_word(prepositions), parse_noun_phrase())
```)

الآن يمكننا تعريف الجملة على أنها مركب اسمي متبوع بمركب فعلي، حيث يمكن أن يكون المركب الفعلي إما فعلًا أو مركبًا فعليًا ممتدًا بجملة جرية:#footnote[لاحظ أن هذا التعريف عودي — فقد يتبع الفعل أي عدد من الجمل الجرية.]

#snippet(```python
def parse_sentence():
    return llist("sentence", parse_noun_phrase(), parse_verb_phrase())
def parse_verb_phrase():
    def maybe_extend(verb_phrase):
        return amb(verb_phrase, maybe_extend(llist("verb-phrase", verb_phrase, parse_prepositional_phrase())))
    return maybe_extend(parse_word(verbs))
```)

وبينما نحن بصدد ذلك، يمكننا أيضًا تفصيل تعريف المركبات الاسمية للسماح بأشياء مثل "a cat in the class". وما كنا نسميه مركبًا اسميًا، سنسميه الآن مركبًا اسميًا بسيطًا، وسيكون المركب الاسمي الآن إما مركبًا اسميًا بسيطًا أو مركبًا اسميًا ممتدًا بجملة جرية:

#snippet(```python
def parse_simple_noun_phrase():
    return llist("simple-noun-phrase", parse_word(articles), parse_word(nouns))
def parse_noun_phrase():
    def maybe_extend(noun_phrase):
        return amb(noun_phrase, maybe_extend(llist("noun-phrase", noun_phrase, parse_prepositional_phrase())))
    return maybe_extend(parse_simple_noun_phrase())
```)

#idx("parse...")

تتيح لنا قواعدنا النحوية الجديدة تحليل جمل أكثر تعقيدًا. على سبيل المثال

#snippet(```python
parse_input(llist("the", "student", "with", "the", "cat",
                 "sleeps", "in", "the", "class"))
```)

ينتج

#snippet(```python
llist("sentence",
     llist("noun-phrase",
          llist("simple-noun-phrase",
               llist("article", "the"), llist("noun", "student")),
          llist("prep-phrase", llist("prep", "with"),
               llist("simple-noun-phrase",
                    llist("article", "the"),
                    llist("noun", "cat")))),
     llist("verb-phrase",
          llist("verb", "sleeps"),
          llist("prep-phrase", llist("prep", "in"),
               llist("simple-noun-phrase",
                    llist("article", "the"),
                    llist("noun", "class")))))
```)

لاحظ أن مدخلاً معينًا قد تكون له أكثر من إعراب قانوني واحد. ففي الجملة "The professor lectures to the student with the cat"، قد يكون الأستاذ يحاضر ومعه القط، أو أن الطالب هو مَن يملك القط. ويجد برنامجنا غير الحتمي كلا الاحتمالين:

#snippet(```python
parse_input(llist("the", "professor", "lectures",
                 "to", "the", "student", "with", "the", "cat"))
```)

ينتج

#snippet(```python
llist("sentence",
     llist("simple-noun-phrase",
          llist("article", "the"), llist("noun", "professor")),
     llist("verb-phrase",
          llist("verb-phrase",
               llist("verb", "lectures"),
               llist("prep-phrase", llist("prep", "to"),
                    llist("simple-noun-phrase",
                    llist("article", "the"),
            llist("noun", "student")))),
          llist("prep-phrase", llist("prep", "with"),
               llist("simple-noun-phrase",
                    llist("article", "the"),
                    llist("noun", "cat")))))
```)

وطلب إعادة المحاولة من المُقَيِّم ينتج

#snippet(```python
llist("sentence",
     llist("simple-noun-phrase",
          llist("article", "the"), llist("noun", "professor")),
     llist("verb-phrase",
          llist("verb", "lectures"),
          llist("prep-phrase", llist("prep", "to"),
               llist("noun-phrase",
                    llist("simple-noun-phrase",
                         llist("article", "the"),
                         llist("noun", "student")),
                    llist("prep-phrase", llist("prep", "with"),
                         llist("simple-noun-phrase",
                              llist("article", "the"),
                              llist("noun", "cat")))))))
```)

#idx("برامج غير حتمية", sub: "تحليل اللغة الطبيعية")

#exercise(label-name: <ex:five_ways>, [
مع القواعد النحوية المعطاة أعلاه، يمكن تحليل الجملة التالية بخمس طرق مختلفة: "The professor lectures to the student in the class with the cat." اعطِ الإعرابات الخمسة ووضح الفروق الدقيقة في المعنى بينها.
])

#exercise(label-name: <ex:ordered_parsing>, [
#idx("مُقَيِّم غير حتمي", sub: "ترتيب تقييم الوسائط")
المُقَيِّمات في الأقسام @sec:mc-eval و @sec:lazy-evaluation لا تحدد الترتيب الذي تُقَيَّم به تعبيرات الوسائط. وسنرى أن مُقَيِّم #py("amb") يُقَيِّمها من اليسار إلى اليمين. وضح لماذا لن يعمل برنامج التحليل الخاص بنا إذا قُيِّمت تعبيرات الوسائط بترتيب آخر.
])

#exercise(label-name: <ex:louis_verb_phrase>, [
يقترح #en[Louis Reasoner] أنه، بما أن المركب الفعلي هو إما فعل أو مركب فعلي متبوع بجملة جرية، فمن الأسهل بكثير التصريح عن الدالة #py("parse_verb_phrase") على النحو التالي (وبالمثل للمركبات الاسمية):

#snippet(```python
def parse_verb_phrase():
    return amb(parse_word(verbs), llist("verb-phrase", parse_verb_phrase(), parse_prepositional_phrase()))
```)

هل يعمل هذا؟ وهل يغير سلوك البرنامج إذا بدلنا ترتيب التعبيرات في #py("amb")؟
])

#exercise(label-name: <ex:complex_sentences>, [
وسع القواعد النحوية المعطاة أعلاه للتعامل مع جمل أكثر تعقيدًا. على سبيل المثال، يمكنك توسيع المركبات الاسمية والفعالية لتشمل الصفات والظروف، أو يمكنك التعامل مع الجمل المركبة.#footnote[يمكن أن يصبح هذا النوع من القواعد النحوية معقدًا بشكل تعسفي، ولكنه مجرد لعبة بالنسبة لـ
#idx("تحليل اللغة الطبيعية", sub: "الفهم الحقيقي للغة مقابل محلل ألعاب")
فهم اللغة الحقيقي. يتطلب فهم اللغة الطبيعية الحقيقي بواسطة الحاسوب مزيجًا متطورًا من التحليل النحوي وتفسير المعنى. من ناحية أخرى، حتى محللات الألعاب يمكن أن تكون مفيدة في دعم لغات أوامر مرنة لبرامج مثل أنظمة استرجاع المعلومات. يناقش #idx("Winston, Patrick Henry") #en[Winston 1992] النهج الحسابية للفهم الحقيقي للغة وكذلك تطبيقات القواعد النحوية البسيطة للغات الأوامر.]
])

#exercise(label-name: <ex:sentence-generate>, [
تهتم #en[Alyssa P. Hacker] بدرجة أكبر بـ
#idx("توليد الجمل")
توليد جمل مثيرة للاهتمام مقارنة بإعرابها. وتحاجج بأنه عن طريق تغيير الدالة #py("parse_word") بحيث تتجاهل «جملة المدخلات» وبدلاً من ذلك تنجح دائمًا وتولد كلمة مناسبة، يمكننا استخدام البرامج التي بنيناها للتحليل لإجراء التوليد بدلاً من ذلك. نفذ فكرة أليسا، واعرض أول نصف دزينة أو نحو ذلك من الجمل الموَلَّدة.#footnote[على الرغم من أن فكرة أليسا تعمل بشكل جيد (وبسيطة بشكل مدهش)، إلا أن الجمل التي تولدها ممله نوعًا ما — فهي لا تعاين الجمل الممكنة لهذه اللغة بطريقة مثيرة للاهتمام للغاية. وفي الواقع، فإن القواعد النحوية عودية للغاية في العديد من الأماكن، وتقنية أليسا «تسقط في» إحدى هذه العوديات وتتعطل. انظر التمرين @ex:ramb لطريقة التعامل مع هذا.]
])

#idx("تحليل اللغة الطبيعية")
#idx("حساب غير حتمي")
