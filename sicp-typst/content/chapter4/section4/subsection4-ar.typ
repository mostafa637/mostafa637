// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#subsection([تنفيذ نظام الاستعلام], label-name: <sec:implementing-the-query-system>)

وصف القسم @sec:how-query-works كيفية عمل نظام الاستعلام. والآن نملأ التفاصيل بتقديم تنفيذ كامل للنظام.

#subsubsection([حلقة المحرك], label-name: <sec:query-driver>)

تقرأ
#idx("حلقة المحرك", sub: "في مفسر الاستعلام")
#idx("مفسر الاستعلام", sub: "حلقة المحرك")
حلقة المحرك لنظام الاستعلام تعبيرات المدخلات مكررة. وإذا كان التعبير قاعدة أو تقريرًا جازمًا يُرَاد إضافته إلى قاعدة البيانات، تُضاف المعلومات. وإلا يُفْتَرَض أن التعبير استعلام. ويمرر المحرك هذا الاستعلام إلى #py("evaluate_query") جنبًا إلى جنب مع تدفق إطارات أولي يتكون من إطار فارغ واحد. ونتيجة التقييم هي تدفق إطارات متولدة بواسطة استيفاء الاستعلام بقيم المتغيرات الموجودة في قاعدة البيانات. وتُستخدم هذه الإطارات لتشكيل تدفق جديد يتكون من نسخ الاستعلام الأصلي التي تُجَسَّد فيها المتغيرات بالقيم المعطاة بواسطة تدفق الإطارات، ويُعْرَض هذا التدفق النهائي:

#idx("محثات", sub: "مفسر الاستعلام")#idx("querydriverloop", decl: true)
#snippet(```python
input_prompt = "Query input:"
output_prompt = "Query results:"
def query_driver_loop():
    input = user_read(input_prompt) + ";"
    if is_none(input):
        print("evaluator terminated")
    else:
        expression = parse(input)
        query = convert_to_query_syntax(expression)
        if is_assertion(query):
            add_rule_or_assertion(assertion_body(query))
            print("Assertion added to data base.")
        else:
            print(output_prompt)
            display_stream(stream_map(lambda frame: (unparse(instantiate_expression(expression, frame))), evaluate_query(query, singleton_stream(None))))
        return query_driver_loop()
```)

وهنا، كما هو الحال في المُقيِّمات الأخرى في هذا الفصل، نستخدم
#idx("بناء جملة تجريدي", sub: "في مفسر الاستعلام")
#idx("parse", sub: "في مفسر الاستعلام")
#py("parse") لتحويل مكوّن من لغة الاستعلام يُعْطَى كسلسلة نصية إلى تمثيل بناء جملة #en[Python]. (ونلحق فاصلة منقوطة بسلسلة تعبيرات المدخلات لأن #py("parse") تتوقع عبارة).
ثم نحول تمثيل بناء الجملة أكثر إلى مستوى مفاهيمي مناسب لنظام الاستعلام باستخدام #py("convert_to_query_syntax")، المصرَّح عنها في القسم @sec:query-syntax جنبًا إلى جنب مع المحمول #py("is_assertion") والمحدد #py("assertion_body").
والدالة #py("add_rule_or_assertion") مصرَّح عنها في القسم @sec:query-db.
وتُستخدم الإطارات الناتجة عن تقييم الاستعلام لتجسيد تمثيل بناء الجملة، وتُلْغَى الفهرسة النحوية للنتيجة إلى سلسلة نصية للطباعة. والدالتان
#idx("مفسر الاستعلام", sub: "تجسيد")
#idx("instantiateexpression")
#py("instantiate_expression") و
#idx("unparse", sub: "في مفسر الاستعلام")
#py("unparse") مصرَّح عنهما في القسم @sec:query-syntax.

#idx("مفسر الاستعلام", sub: "تجسيد")
#idx("مفسر الاستعلام", sub: "حلقة المحرك")

#subsubsection([المُقيِّم], label-name: <sec:query-eval>)

الدالة
#idx("مفسر الاستعلام", sub: "مُقَيِّم الاستعلام")
#py("evaluate_query")،
المستدعاة بواسطة #py("query_driver_loop")، هي المُقيِّم الأساسي لنظام الاستعلام. وتأخذ كمدخلات استعلامًا وتدفق إطارات، وترجع تدفق إطارات مُمَدَّدة.
وتحدد الأشكال
النحوية بواسطة
#idx("برمجة موجهة بالبيانات", sub: "في مفسر الاستعلام")
توجيه موجه بالبيانات باستخدام #py("get") و#py("put")، تمامًا كما فعلنا في تنفيذ العمليات العامة في الفصل @chap:data. وأي استعلام لا يُحَدَّد كشكل نحوي يُفْتَرَض أنه استعلام بسيط، ليعالج بواسطة #py("simple_query").
#idx("evaluatequery", decl: true)
#snippet(```python
def evaluate_query(query, frame_stream):
    qfun = get(type(query), "evaluate_query")
    return simple_query(query, frame_stream) if is_none(qfun) else qfun(contents(query), frame_stream)
```)

والدالتان #py("type") و#py("contents")، المُعَرَّفتان في القسم @sec:query-syntax، تنفذان بناء الجملة التجريدي للأشكال النحوية.

#subheading([استعلامات بسيطة])

#idx("استعلام بسيط", sub: "معالجة")

تتعامل الدالة
#py("simple_query")
مع الاستعلامات البسيطة. وتأخذ كوسائط استعلامًا بسيطًا (نمطًا) جنبًا إلى جنب مع تدفق إطارات، وترجع التدفق المشكل عن طريق تمديد كل إطار بجميع مطابقات قاعدة البيانات للاستعلام.
#idx("simplequery", decl: true)
#snippet(```python
def simple_query(query_pattern, frame_stream):
    return stream_flatmap(lambda frame: (stream_append_delayed(find_assertions(query_pattern, frame), lambda : (apply_rules(query_pattern, frame)))), frame_stream)
```)

لكل إطار في تدفق المدخلات، نستخدم #py("find_assertions") (القسم @sec:query-match) لمطابقة النمط مقابل جميع التقريرات الجازمة في قاعدة البيانات، مما ينتج تدفق إطارات مُمَدَّدة، ونستخدم #py("apply_rules") (القسم @sec:query-unify) لتطبيق جميع القواعد الممكنة، مما ينتج تدفق إطارات مُمَدَّدة آخر.
وتُدْمَج هاتان التدفقان (باستخدام #py("stream_append_delayed")، القسم @sec:query-streams) لإنشاء تدفق بجميع الطرق التي يمكن بها استيفاء النمط المعطى متسقًا مع الإطار الأصلي (انظر التمرين @ex:q-why-not-append).
وتُدْمَج التدفقات للإطارات الفردية للمدخلات باستخدام #py("stream_flatmap") (القسم @sec:query-streams) لتشكيل تدفق ضخم واحد بجميع الطرق التي يمكن بها تمديد أي من الإطارات في تدفق المدخلات الأصلي لإنتاج مطابقة مع النمط المعطى.

#idx("استعلام بسيط", sub: "معالجة")

#subheading([استعلامات مركبة])

#idx("استعلام مركب", sub: "معالجة")

نتعامل مع استعلامات #idx("and (لغة الاستعلام)", sub: "تقييم الـ") #py("and") كما هو موضح في الشكل @fig:query-and بالدالة #py("conjoin")، التي تأخذ كمدخلات المقترنات وتدفق الإطارات وترجع تدفق الإطارات المُمَدَّدة. أولاً، تعالج #py("conjoin") تدفق الإطارات للعثور على تدفق جميع توسيعات الإطارات الممكنة التي تستوفي الاستعلام الأول في الإقتران. ثم، باستخدام هذا كتدفق إطارات جديد، تطبق عوديًا #py("conjoin") على باقي الاستعلامات.
#idx("conjoin", decl: true)
#snippet(```python
def conjoin(conjuncts, frame_stream):
    return frame_stream if is_empty_conjunction(conjuncts) else conjoin(rest_conjuncts(conjuncts), evaluate_query(first_conjunct(conjuncts), frame_stream))
```)

والعبارة

#snippet(```python
put("and", "evaluate_query", conjoin)
```)

تعد #py("evaluate_query") للتوجيه إلى #py("conjoin") عند اعتراض #py("and").

ونتعامل مع استعلامات #idx("or (لغة الاستعلام)", sub: "تقييم الـ") #py("or") بنفس الطريقة، كما هو موضح في الشكل @fig:query-or.
وتُحْسَب تدفقات المخرجات للمفصلات المختلفة لـ #py("or") بشكل منفصل وتُدْمَج باستخدام الدالة #py("interleave_delayed") من القسم @sec:query-streams. (انظر التمارين @ex:q-why-not-append و @ex:q-why-interleave).
#idx("disjoin", decl: true)
#snippet(```python
def disjoin(disjuncts, frame_stream):
    return None if is_empty_disjunction(disjuncts) else interleave_delayed(evaluate_query(first_disjunct(disjuncts), frame_stream), lambda : (disjoin(rest_disjuncts(disjuncts), frame_stream)))
put("or", "evaluate_query", disjoin)
```)

المحمولات والمحددات لتمثيل المقترنات والمفصلات مُعْطَاة في القسم @sec:query-syntax.

#subheading([المرشحات])

الشكل النحوي لـ #idx("not (لغة الاستعلام)", sub: "تقييم الـ") #py("not") يُعَالَج بالطريقة الموضحة خطوطها العريضة في القسم @sec:how-query-works. فنحاول تمديد كل إطار في تدفق المدخلات لـ استيفاء الاستعلام الذي يُنْفَى، ونضمن إطارًا معينًا في تدفق المخرجات فقط إذا لم يكن بالإمكان تمديده.

#idx("negate", decl: true)
#snippet(```python
def negate(exps, frame_stream):
    return stream_flatmap(lambda frame: (singleton_stream(frame) if is_none(evaluate_query(negated_query(exps), singleton_stream(frame))) else None), frame_stream)
put("not", "evaluate_query", negate)
```)

والشكل النحوي لـ #idx("javascriptpredicate (لغة الاستعلام)", sub: "تقييم الـ") #py("javascript_predicate") هو مرشح مماثل لـ #py("not").

ويُستخدم كل إطار في التدفق لتجسيد المتغيرات في المحمول، ويُقَيَّم المحمول المُجَسَّد، وتُصَفَّى الإطارات التي يُقَيَّم المحمول من أجلها إلى خادع من تدفق المدخلات.
ويُقَيَّم المحمول المُجَسَّد باستخدام #py("evaluate") من القسم @sec:mc-eval مع #py("the_global_environment") وبالتالي يمكنه التعامل مع أي تعبير #en[Python]، طالما أن جميع متغيرات الأنماط مُمَثَّلة ومُجَسَّدة قبل التقييم.

#idx("javascriptpredicate (مفسر الاستعلام)", decl: true)
#snippet(```python
def javascript_predicate(exps, frame_stream):
    return stream_flatmap(lambda frame: (singleton_stream(frame) if evaluate(instantiate_expression(javascript_predicate_expression(exps), frame), the_global_environment) else None), frame_stream)
put("javascript_predicate", "evaluate_query", javascript_predicate)
```)

والشكل النحوي #py("always_true") يوفر استعلامًا يُسْتَوْفَى دائمًا. فهو يتجاهل محتوياته (عادة فارغة) ويمرر ببساطة جميع الإطارات في تدفق المدخلات.
والمحدد #py("rule_body") (القسم @sec:query-syntax) يستخدم #py("always_true")
#idx("قاعدة (لغة الاستعلام)", sub: "بدون متن")
لتوفير متون للقواعد التي عُرِّفت بدون متون (أي القواعد التي تُسْتَوْفَى متونها دائمًا).
#idx("alwaystrue", decl: true)
#snippet(```python
def always_true(ignore, frame_stream):
    return frame_stream
put("always_true", "evaluate_query", always_true)
```)

المحددات التي تحدد بناء جملة #py("not") و#py("javascript_predicate") مُعْطَاة في القسم @sec:query-syntax.

#idx("استعلام مركب", sub: "معالجة")
#idx("مفسر الاستعلام", sub: "مُقَيِّم الاستعلام")

#subsubsection([العثور على التقريرات الجازمة بمطابقة الأنماط], label-name: <sec:query-match>)

الدالة #py("find_assertions")،
#idx("مفسر الاستعلام", sub: "مطابقة الأنماط")
#idx("مطابقة الأنماط", sub: "تنفيذ")
المستدعاة بواسطة #py("simple_query") (القسم @sec:query-eval)، تأخذ كمدخلات نمطًا وإطارًا. وترجع تدفق إطارات، يمدد كل منها الإطار المعطى بمطابقة قاعدة بيانات للنمط المعطى. وتستخدم #py("fetch_assertions") (القسم @sec:query-db) للحصول على تدفق بجميع التقريرات الجازمة في قاعدة البيانات التي ينبغي التحقق منها لمطابقتها مع النمط والإطار. والسبب في استخدام #py("fetch_assertions") هنا هو أنه يمكننا غالبًا تطبيق اختبارات بسيطة ستزيل العديد من الإدخالات في قاعدة البيانات من مجمع المرشحين لمطابقة ناجحة. وسيعمل النظام مع ذلك إذا ألغينا #py("fetch_assertions") وفحصنا ببساطة تدفق جميع التقريرات الجازمة في قاعدة البيانات، لكن الحساب سيكون أقل كفاءة لأننا سنحتاج إلى إجراء استدعاءات أكثر بكثير للمطابق.
#idx("findassertions", decl: true)
#snippet(```python
def find_assertions(pattern, frame):
    return stream_flatmap(lambda datum: (check_an_assertion(datum, pattern, frame)), fetch_assertions(pattern, frame))
```)

وتأخذ الدالة #py("check_an_assertion") كوسائط كائنَ بيانات (تقريراً جازماً)، ونمطًا، وإطارًا وترجع إما تدفقًا من عنصر واحد يحتوي على الإطار المُمَدَّد وإما #py("None") إذا فشلت المطابقة.
#idx("checkanassertion", decl: true)
#snippet(```python
def check_an_assertion(assertion, query_pat, query_frame):
    match_result = pattern_match(query_pat, assertion, query_frame)
    return None if match_result == "failed" else singleton_stream(match_result)
```)

ومطابق الأنماط الأساسي يرجع إما السلسلة النصية #py("\"failed\"") وإما توسيعًا للإطار المعطى. والفكرة الأساسية للمطابق هي التحقق من النمط مقابل البيانات، عنصرًا تلو الآخر، مجمعًا الرابطات لمتغيرات النمط. وإذا كان النمط وكائن البيانات متماثلين، تنجح المطابقة ونرجع إطار الرابطات المجمع حتى الآن. وإلا، إذا كان النمط متغيرًا (يتم التحقق منه بواسطة الدالة #py("is_variable") المصرَّح عنها في القسم @sec:query-syntax)، نمدد الإطار الحالي عن طريق ربط المتغير بالبيانات، طالما أن هذا متسق مع الرابطات الموجودة بالفعل في الإطار. وإذا كان النمط والبيانات كلاهما أزواجًا، نطابق (عوديًا) رأس النمط مقابل رأس البيانات لإنتاج إطار؛ وفي هذا الإطار نطابق بعد ذلك ذيل النمط مقابل ذيل البيانات. وإذا لم تكن أي من هذه الحالات قابلة للتطبيق، تفشل المطابقة ونرجع السلسلة النصية #py("\"failed\"").
#idx("patternmatch", decl: true)
#snippet(```python
def pattern_match(pattern, data, frame):
    return "failed" if frame == "failed" else frame if pattern == data else extend_if_consistent(pattern, data, frame) if is_variable(pattern) else pattern_match(tail(pattern), tail(data), pattern_match(head(pattern), head(data), frame)) if is_pair(pattern) and is_pair(data) else "failed"
```)

إليك الدالة التي تمدد إطارًا عن طريق إضافة ربط جديد، إذا كان هذا متسقًا مع الرابطات الموجودة بالفعل في الإطار:

#idx("extendifconsistent", decl: true)
#snippet(```python
def extend_if_consistent(variable, data, frame):
    binding = binding_in_frame(variable, frame)
    return extend(variable, data, frame) if is_none(binding) else pattern_match(binding_value(binding), data, frame)
```)

إذا لم يكن هناك ربط للمتغير في الإطار، نضيف ببساطة ربط المتغير بالبيانات. وإلا نطابق، في الإطار، البيانات مقابل قيمة المتغير في الإطار. وإذا كانت القيمة المخزنة تحتوي على ثوابت فقط، كما يجب أن تكون إذا أُعْطِيَت أثناء مطابقة الأنماط بواسطة #py("extend_if_consistent")، فإن المطابقة تختبر ببساطة ما إذا كانت القيمة المخزنة والقيمة الجديدة هما نفسيهما. وإذا كان الأمر كذلك، ترجع الإطار غير المعدل؛ وإذا لم يكن كذلك، ترجع إشارة فشل. ومع ذلك، قد تحتوي القيمة المخزنة على متغيرات أنماط إذا تم تخزينها أثناء التوحيد (انظر القسم @sec:query-unify). والمطابقة العودية للنمط المخزن مقابل البيانات الجديدة ستضيف أو تتحقق من الرابطات لـ المتغيرات في هذا النمط. على سبيل المثال، افترض أن لدينا إطارًا فيه #py("$x") مرتبطة بـ #py("llist(\"f\", $y)") و#py("$y") غير مرتبطة، ونرغب في تزويد هذا الإطار بـ ربط لـ #py("$x") بـ #py("llist(\"f\", \"b\")"). فنبحث عن #py("$x") ونجد أنها مرتبطة بـ #py("llist(\"f\", $y)"). ويقودنا هذا لمطابقة #py("llist(\"f\", $y)") مقابل القيمة الجديدة المقترحة #py("llist(\"f\", \"b\")") في الإطار نفسه. وفي النهاية تمدد هذه المطابقة الإطار عن طريق إضافة ربط لـ #py("$y") بـ #py("\"b\""). ويظل المتغير #py("$x") مرتبطًا بـ #py("llist(\"f\", $y)"). ونحن لا نعدل أبدًا ربطًا مخزنًا ولا نخزن أبدًا أكثر من ربط واحد لمتغير معين.

والدوال المستخدمة بواسطة #py("extend_if_consistent") لمعالجة الرابطات مُعَرَّفة في القسم @sec:query-bindings.

#idx("مفسر الاستعلام", sub: "مطابقة الأنماط")
#idx("مطابقة الأنماط", sub: "تنفيذ")

#subsubsection([القواعد والتوحيد], label-name: <sec:query-unify>)

#idx("قاعدة (لغة الاستعلام)", sub: "تطبيق الـ")

الدالة #py("apply_rules") هي نظيرة القواعد لـ #py("find_assertions") (القسم @sec:query-match). وتأخذ كمدخلات نمطًا وإطارًا، وتملك تدفق إطارات توسيع عن طريق تطبيق القواعد من قاعدة البيانات. والدالة #py("stream_flatmap") ترسم خرائط #py("apply_a_rule") لأسفل تدفق القواعد القابلة للتطبيق محتملًا (المحددة بواسطة #py("fetch_rules")، القسم @sec:query-db) وتدمج تدفقات الإطارات الناتجة.

#idx("applyrules", decl: true)
#snippet(```python
def apply_rules(pattern, frame):
    return stream_flatmap(lambda rule: (apply_a_rule(rule, pattern, frame)), fetch_rules(pattern, frame))
```)

وتطبق الدالة #py("apply_a_rule") قاعدة باستخدام الطريقة الموضحة خطوطها العريضة في القسم @sec:how-query-works. فهي تمدد أولًا إطار وسائطها عن طريق توحيد نتيجة القاعدة مع النمط في الإطار المعطى. وإذا نجح هذا، تقيم متن القاعدة في هذا الإطار الجديد.

وقبل حدوث أي من هذا، يعيد البرنامج تسمية جميع المتغيرات في القاعدة بأسماء جديدة فريدة. والسبب في ذلك هو منع المتغيرات لتطبيقات القواعد المختلفة من أن تختلط ببعضها البعض. على سبيل المثال، إذا كانت قاعدتان تستخدمان متغيرًا يسمى #py("$x")، فقد تضيف كل واحدة منهما ربطًا لـ #py("$x") إلى الإطار عند تطبيقها. وهاتان الـ #py("$x") ليس بينهما أي علاقة ببعضهما البعض، ولا ينبغي أن ننخدع بالتفكير في أن الربطين يجب أن يكونا متسقين. وبدلاً من إعادة تسمية المتغيرات، كان بإمكاننا ابتكار بنية بيئة أكثر ذكاءً؛ ومع ذلك، فإن نهج إعادة التسمية الذي اخترناه هنا هو الأحدث والمباشر، حتى لو لم يكن الأكثر كفاءة. (انظر التمرين @ex:query-local-names). إليك دالة #py("apply_a_rule"):
#idx("applyarule", decl: true)
#snippet(```python
def apply_a_rule(rule, query_pattern, query_frame):
    clean_rule = rename_variables_in(rule)
    unify_result = unify_match(query_pattern, conclusion(clean_rule), query_frame)
    return None if unify_result == "failed" else evaluate_query(rule_body(clean_rule), singleton_stream(unify_result))
```)

والمحددتان #py("rule_body") و#py("conclusion") اللتان تستخرجان أجزاء القاعدة مُعَرَّفتان في القسم @sec:query-syntax.

ونولد أسماء متغيرات فريدة عن طريق ربط معرف فريد (مثل عدد) مع كل تطبيق قاعدة ودمج هذا المعرف مع أسماء المتغيرات الأصلية. على سبيل المثال، إذا كان معرف تطبيق القاعدة هو 7، فقد نغير كل #py("$x") في القاعدة إلى #py("$x_7") وكل #py("$y") في القاعدة إلى #py("$y_7").
(والدالتان #py("make_new_variable") و#py("new_rule_application_id") مدرجتان مع دوال بناء الجملة في القسم @sec:query-syntax.)
#idx("renamevariablesin", decl: true)
#snippet(```python
def rename_variables_in(rule):
    rule_application_id = new_rule_application_id()
    def tree_walk(exp):
        return make_new_variable(exp, rule_application_id) if is_variable(exp) else pair(tree_walk(head(exp)), tree_walk(tail(exp))) if is_pair(exp) else exp
    return tree_walk(rule)
```)

#idx("قاعدة (لغة الاستعلام)", sub: "تطبيق الـ")

و
#idx("مفسر الاستعلام", sub: "التوحيد")
#idx("توحيد", sub: "تنفيذ")
خوارزمية التوحيد مُنَفَّذة كدالة تأخذ كمدخلات نمطين وإطارًا وترجع إما الإطار المُمَدَّد وإما السلسلة النصية #py("\"failed\"").
والموحد يشبه مطابق الأنماط باستثناء أنه متناظر — فالمتغيرات مسموح بها على كلا جانبي المطابقة. والدالة #py("unify_match") هي أساسًا نفس #py("pattern_match")، باستثناء وجود بند إضافي (موسوم بـ \*\*\* أدناه) للتعامل مع الحالة التي يكون فيها الكائن الموجود على الجانب الأيمن للمطابقة متغيرًا.
#idx("unifymatch", decl: true)
#snippet(```python
def unify_match(p1, p2, frame):
    return "failed" if frame == "failed" else frame if p1 == p2 else extend_if_possible(p1, p2, frame) if is_variable(p1) else extend_if_possible(p2, p1, frame) if is_variable(p2) else unify_match(tail(p1), tail(p2), unify_match(head(p1), head(p2), frame)) if is_pair(p1) and is_pair(p2) else "failed"
```)

في التوحيد، كما هو الحال في مطابقة الأنماط أحادية الجانب، نريد قبول التوسيع المقترح للإطار فقط إذا كان متسقًا مع الرابطات الموجودة. والدالة #py("extend_if_possible") المستخدمة في التوحيد هي نفس الدالة #py("extend_if_consistent") المستخدمة في مطابقة الأنماط باستثناء فحصين خاصين، موسومين بـ \*\*\* في البرنامج أدناه. ففي الحالة الأولى، إذا كان المتغير الذي نحول مطابقته غير مرتبط، ولكن القيمة التي نحاول مطابقته بها هي نفسها متغير (مختلف)، فمن الضروري الفحص لمعرفة ما إذا كانت القيمة مرتبطة، وإذا كان الأمر كذلك، فمطابقة قيمتها. وإذا كان كلا الطرفين للمطابقة غير مرتبطين، فيمكننا ربط أي منهما بالآخر.

وفحص التداخل الثاني يتناول محاولات ربط متغير بنمط يتضمن ذلك المتغير. ومثل هذا الموقف قد يحدث كلما تكرر متغير في كلا النمطين. تأمل، على سبيل المثال، توحيد النمطين #py("llist($x, $x)") و#py("llist($y,") $⟨$#meta("expression") #meta("involving") #py("$y")$⟩$#py(")") في إطار يكون فيه كل من #py("$x") و#py("$y") غير مرتبطين. في البداية تُطَابَق #py("$x") مقابل #py("$y")، مما يصنع ربطًا لـ #py("$x") بـ #py("$y").
وبعد ذلك، تُطَابَق نفس #py("$x") مقابل التعبير المعطى المتضمن #py("$y"). وبما أن #py("$x") مرتبطة بالفعل بـ #py("$y")، ينتج عن ذلك مطابقة #py("$y") مقابل التعبير. وإذا فكرنا في الموحد كمسستكشف لمجموعة قيم لمتغيرات النمط تجعل الأنماط متطابقة، فإن هذه الأنماط تقتضي تعليمات لإيجاد #py("$y") بحيث تكون #py("$y") مساوية للتعبير المتضمن #py("$y"). ونحن نرفض مثل هذه الرابطات؛ وتتعرف المحمول #py("depends_on") على هذه الحالات.#footnote[عمومًا، توحيد #py("$y") مع تعبير يتضمن #py("$y") سيتطلب قدرتنا على العثور على
#idx("نقطة ثابتة", sub: "التوحيد و")
نقطة ثابتة للمعادلة #py("$y") $= ⟨$#meta("expression") #meta("involving") #py("$y")$⟩$.
ومن الممكن أحيانًا تشكيل تعبير نحويًا يبدو أنه الحل. على سبيل المثال، يبدو أن #py("$y") $=$ #py("llist(\"f\", $y)") يملك النقطة الثابتة #py("llist(\"f\", llist(\"f\", llist(\"f\",") … #py(")))")، والتي يمكننا إنتاجها بالبدء بالتعبير #py("llist(\"f\", $y)") والاستبدال المتكرر لـ #py("llist(\"f\", $y)") محل #py("$y").
ولسوء الحظ، ليست كل معادلة من هذا القبيل تملك نقطة ثابتة ذات معنى. والقضايا التي تنشأ هنا تشبه قضايا معالجة
#idx("متسلسلة لانهائية")
المتسلسلات اللانهائية في الرياضيات. على سبيل المثال، نعلم أن 2 هو حل المعادلة $y = 1 + y/2$.
وبالبدء بالتعبير $1 + y/2$ والاستبدال المتكرر لـ $1 + y/2$ محل $y$ نحصل على

$ 2 space = space y space = space 1 + y/2 space = space 1 + (1+y/2)/2 space = space 1 + 1/2 + y/4 space = space dots.c , $

مما يؤدي إلى

$ 2 space = space 1 + 1/2 + 1/4 + 1/8 + dots.c . $

ومع ذلك، إذا جربنا المعالجة نفسها بالبدء بملاحظة أن $-1$ هو حل المعادلة $y space = space 1 + 2y$، فنحصل على

$ -1 space = space y space = space 1 + 2y space = space 1 + 2(1 + 2y) space = space 1 + 2 + 4y space = space dots.c , $

مما يؤدي إلى

$ -1 space = space 1 + 2 + 4 + 8 + dots.c . $

وعلى الرغم من أن المعالجات الصورية المستخدمة في اشتقاق هاتين المعادلتين متطابقة، إلا أن النتيجة الأولى تأكيد صحيح عن المتسلسلات اللانهائية ولكن النتيجة الثانية ليست كذلك. وبالمثل، بالنسبة لنتائج التوحيد الخاصة بنا، فإن التفكير باستخدام تعبير مبني نحويًا بشكل تعسفي قد يؤدي إلى أخطاء.

ومع ذلك، تتيح معظم أنظمة البرمجة المنطقية اليوم المراجع الدائرية، عن طريق قبول بنية البيانات الدائرية كنتيجة للمطابقة. ويبرر ذلك نظريًا باستخدام #emph[الأشجار الناطقة] (#en[rational trees])
#idx("Jaffar, Joxan")
#idx("Stuckey, Peter J.")
#idx("شجرة", sub: "ناطقة")
#idx("شجرة ناطقة")
(#en[Jaffar and Stuckey 1986]).
وقبول بنية البيانات الدائرية يتيح بيانات ذاتية المرجعية، مثل بنية بيانات الموظف التي تشير إلى صاحب العمل، الذي يشير بدوره إلى الموظف.]
ومن ناحية أخرى، لا نريد رفض محاولات ربط متغير بنفسه. على سبيل المثال، تأمل توحيد #py("llist($x, $x)") و#py("llist($y, $y)"). المحاولة الثانية لربط #py("$x") بـ #py("$y") تطابق #py("$y") (القيمة المخزنة لـ #py("$x")) مقابل #py("$y") (القيمة الجديدة لـ #py("$x")). ويتكفل بهذا بند #py("equal") في #py("unify_match").

#idx("extendifpossible", decl: true)
#snippet(```python
def extend_if_possible(variable, value, frame):
    binding = binding_in_frame(variable, frame)
    if  not  is_none(binding):
        return unify_match(binding_value(binding), value, frame)
    elif is_variable(value):
        binding = binding_in_frame(value, frame)
        return unify_match(variable, binding_value(binding), frame) if not is_none(binding) else extend(variable, value, frame)
    elif depends_on(value, variable, frame):
        return "failed"
    else:
        return extend(variable, value, frame)
```)

والدالة #py("depends_on") هي محمول يختبر ما إذا كان التعبير المقترح ليكون قيمة متغير النمط يعتمد على المتغير. ويجب القيام بذلك بالنسبة للإطار الحالي لأن التعبير قد يحتوي على ظهورات لمتغير يملك بالفعل قيمة تعتمد على متغير الاختبار الخاص بنا. وبنية #py("depends_on") عبارة عن مرور عودي بسيط في شجرة نلغي فيه قيم المتغيرات كلما كان ذلك ضروريًا.
#idx("dependson", decl: true)
#snippet(```python
def depends_on(expression, variable, frame):
    def tree_walk(e):
        if is_variable(e):
            if variable == e:
                return True
            else:
                b = binding_in_frame(e, frame)
                return False if is_none(b) else tree_walk(binding_value(b))
        else:
            return tree_walk(head(e)) or tree_walk(tail(e)) if is_pair(e) else False
    return tree_walk(expression)
```)

#idx("مفسر الاستعلام", sub: "التوحيد")
#idx("توحيد", sub: "تنفيذ")

#subsubsection([صيانة قاعدة البيانات], label-name: <sec:query-db>)

إحدى المشكلات المهمة في تصميم لغات البرمجة المنطقية هي ترتيب الأمور بحيث يُفْحَص أقل عدد ممكن من إدخالات قاعدة البيانات غير ذات الصلة
#idx("مفسر الاستعلام", sub: "قاعدة بيانات")
#idx("قاعدة بيانات", sub: "فهرسة")
#idx("فهرسة قاعدة بيانات")
عند التحقق من نمط معين. ولهذا الغرض، سنمثل التقرير الجازم كقائمة رأسها سلسلة نصية تمثل نوع معلومات التقرير الجازم. ونخزن التقريرات الجازمة في تدفقات منفصلة، تدفق لكل نوع معلومات، في جدول مفهرس حسب النوع. ولجلب تقرير جازم قد يطابق نمطًا، نرجع (ليتم اختباره باستخدام المطابق) جميع التقريرات الجازمة المخزنة التي لها نفس الرأس (نفس نوع المعلومات). ويمكن للطرق الأكثر ذكاءً الاستفادة أيضًا من المعلومات الموجودة في الإطار. ونحن نتجنب بناء معاييرنا للفهرسة داخل البرنامج؛ وبدلاً من ذلك نستدعي محمولات ومحددات تجسد معاييرنا.
#idx("fetchassertions", decl: true)
#snippet(```python
def fetch_assertions(pattern, frame):
    return get_indexed_assertions(pattern)
def get_indexed_assertions(pattern):
    return get_stream(index_key_of(pattern), "assertion-stream")
```)

والدالة #py("get_stream") تبحث عن تدفق في الجدول وترجع تدفقًا فارغًا إذا لم يكن هناك شيء مخزن هناك.

#snippet(```python
def get_stream(key1, key2):
    s = get(key1, key2)
    return None if is_none(s) else s
```)

وتُخَزَّن القواعد بشكل مشابه، باستخدام رأس نتيجة القاعدة. والنمط يمكنه مطابقة القواعد التي تتشارك نتائجها نفس الرأس. وبالتالي، عند جلب القواعد التي قد تطابق نمطًا، نجلب جميع القواعد التي تملك نتائجها نفس رأس النمط.

#idx("fetchrules", decl: true)
#snippet(```python
def fetch_rules(pattern, frame):
    return get_indexed_rules(pattern)
def get_indexed_rules(pattern):
    return get_stream(index_key_of(pattern), "rule-stream")
```)

والدالة #py("add_rule_or_assertion") تُستخدَم بواسطة #py("query_driver_loop") لإضافة التقريرات الجازمة والقواعد إلى قاعدة البيانات. ويُخَزَّن كل عنصر في الفهرس.
#idx("addruleorassertion", decl: true)
#snippet(```python
def add_rule_or_assertion(assertion):
    return add_rule(assertion) if is_rule(assertion) else add_assertion(assertion)
def add_assertion(assertion):
    store_assertion_in_index(assertion)
    return "ok"
def add_rule(rule):
    store_rule_in_index(rule)
    return "ok"
```)

ولتخزين تقرير جازم أو قاعدة فعليًا، نخزنه في التدفق المناسب.

#snippet(```python
def store_assertion_in_index(assertion):
    key = index_key_of(assertion)
    current_assertion_stream = get_stream(key, "assertion-stream")
    put(key, "assertion-stream", pair(assertion, lambda : (current_assertion_stream)))
def store_rule_in_index(rule):
    pattern = conclusion(rule)
    key = index_key_of(pattern)
    current_rule_stream = get_stream(key, "rule-stream")
    put(key, "rule-stream", pair(rule, lambda : (current_rule_stream)))
```)

والمفتاح الذي يُخَزَّن تحته نمط ما (تقرير جازم أو نتيجة قاعدة) في الجدول هو السلسلة النصية التي يبتدئ بها.

#snippet(```python
def index_key_of(pattern):
    return head(pattern)
```)

#idx("مفسر الاستعلام", sub: "قاعدة بيانات")

#subsubsection([عمليات التدفقات], label-name: <sec:query-streams>)

#idx("مفسر الاستعلام", sub: "عمليات التدفقات")

يستخدم نظام الاستعلام بعض عمليات التدفقات التي لم تُعْرَض في الفصل @chap:state.

والدالتان #py("stream_append_delayed") و#py("interleave_delayed") تشبهان تمامًا #py("stream_append") و#py("interleave") (القسم @sec:exploiting-streams)، باستثناء أنهما تأخذان وسيطًا مؤجَّلًا (مثل دالة #py("integral") في القسم @sec:streams-and-delayed-evaluation). ويؤجل هذا التكرار في بعض الحالات (انظر التمرين @ex:q-why-not-append).
#idx("streamappenddelayed", decl: true)#idx("interleavedelayed", decl: true)
#snippet(```python
def stream_append_delayed(s1, delayed_s2):
    return delayed_s2() if is_none(s1) else pair(head(s1), lambda : (stream_append_delayed(stream_tail(s1), delayed_s2)))
def interleave_delayed(s1, delayed_s2):
    return delayed_s2() if is_none(s1) else pair(head(s1), lambda : (interleave_delayed(delayed_s2(), lambda : (stream_tail(s1)))))
```)

والدالة #py("stream_flatmap")، المستخدَمة في جميع أنحاء مُقَيِّم الاستعلام لرسم خرائط دالة ما عبر تدفق إطارات ودمج تدفقات الإطارات الناتجة، هي نظيرة التدفقات لدالة #py("flatmap") المُمَكَّنة للقوائم الاعتيادية في القسم @sec:nested-mappings. وعلى عكس #py("flatmap") الاعتيادية، فإننا نجمع التدفقات بعملية تداخل، بدلاً من مجرد إلحاقها (انظر التمارين @ex:q-why-interleave و @ex:q-why-delay).
#idx("streamflatmap", decl: true)#idx("flattenstream", decl: true)
#snippet(```python
def stream_flatmap(fun, s):
    return flatten_stream(stream_map(fun, s))
def flatten_stream(stream):
    return None if is_none(stream) else interleave_delayed(head(stream), lambda : (flatten_stream(stream_tail(stream))))
```)

كما يستخدم المُقيِّم الدالة البسيطة التالية لتوليد تدفق يتكون من عنصر واحد:
#idx("singletonstream", decl: true)
#snippet(```python
def singleton_stream(x):
    return pair(x, lambda : (None))
```)

#idx("مفسر الاستعلام", sub: "عمليات التدفقات")

#subsubsection([دوال بناء جملة الاستعلام والتجسيد], label-name: <sec:query-syntax>)

#idx("مفسر الاستعلام", sub: "بناء جملة لغة الاستعلام")
#idx("تمثيل خاص بلغة الاستعلام")

رأينا في القسم @sec:query-driver أن حلقة المحرك تحول أولاً سلسلة نصية للمدخلات إلى تمثيل بناء جملة #en[Python]. وتعتبر المدخلات مصممة لتبدو مثل تعبير #en[Python] بحيث يمكننا استخدام الدالة #py("parse") من القسم @sec:representing-expressions وأيضًا لدعم تدوين #en[Python] في #py("javascript_predicate"). على سبيل المثال،

#snippet(```python
parse('job($x, list("computer", "wizard"));')
```)

ينتج عنه

#output(```python
parse('job($x, list("computer", "wizard"));')
```)

ويشير الوسام #py("\"application\"") إلى أنه من الناحية النحوية، سيُعَامَل الاستعلام كتطبيق دالة في #en[JavaScript].
والدالة #py("unparse") تحول بناء الجملة مجددًا إلى سلسلة نصية:

#snippet(```python
unparse(parse('job($x, list("computer", "wizard"));'))
```)

#output(```python
unparse(parse('job($x, list("computer", "wizard"));'))
```)

وفي معالج الاستعلام، افترضنا تمثيلاً خاصًا بلغة الاستعلام للتقريرات الجازمة والقواعد والاستعلامات.
والدالة #py("convert_to_query_syntax") تحول تمثيل بناء الجملة إلى ذلك التمثيل. وبباستخدام نفس المثال،

#snippet(```python
convert_to_query_syntax(parse('job($x, list("computer", "wizard"));'))
```)

ينتج عنه

#output(```python
convert_to_query_syntax(parse('job($x, list("computer", "wizard"));'))
```)

ودوال نظام الاستعلام مثل #py("add_rule_or_assertion") في القسم @sec:query-db و#py("evaluate_query") في القسم @sec:query-eval تعمل على التمثيل الخاص بلغة الاستعلام باستخدام محددات ومحمولات مثل #py("type") و#py("contents") و#py("is_rule") و#py("first_conjunct") المصرَّح عنها أدناه.
ويوضح الشكل @fig:syntax-abstraction-lp موانع التجريد الثلاثة
#idx("موانع التجريد", sub: "في لغة الاستعلام")
المستخدمة بواسطة نظام الاستعلام وكيف تجسرها دوال التحويل #py("parse") و#py("unparse") و#py("convert_to_query_syntax").

#sicp-figure(image("/images/img_javascript/ch4-syntax-abstraction-logic-programming.svg", width: 70%), caption: [تجريد بناء الجملة في نظام الاستعلام.], label-name: <fig:syntax-abstraction-lp>)

#subheading([التعامل مع متغيرات الأنماط])

يُسْتَخْدَم المحمول #py("is_variable") على التمثيل الخاص بلغة الاستعلام أثناء معالجة الاستعلام وعلى تمثيل بناء جملة #en[Python] أثناء التجسيد للتعرف على الأسماء التي تبدأ بعلامة الدولار.
#idx("مفسر الاستعلام", sub: "تمثيل متغير النمط")
#idx("متغير نمط", sub: "تمثيل الـ")
ونفترض وجود دالة #py("char_at") ترجع سلسلة نصية تحتوي فقط على المحرف للسلسلة المعطاة في الموضع المعطى.#footnote[الطريقة الفعلية للحصول على السلسلة التي تحتوي على المحرف الأول لسلسلة #py("s") في #en[JavaScript] هي #py("s.charAt(0)").]
#idx("isvariable", sub: "في نظام الاستعلام", decl: true)
#snippet(```python
def is_variable(exp):
    return is_name(exp) and char_at(symbol_of_name(exp), 0) == "$"
```)

وتُبْنَى المتغيرات الفريدة أثناء تطبيق القاعدة (في القسم @sec:query-unify) عن طريق الدوال التالية.
والمعرف الفريد لتطبيق قاعدة هو عدد، يزداد في كل مرة تُطَبَّق فيها قاعدة.
#footnote[إنشاء متغيرات جديدة بدمج السلاسل النصية والتعرف على المتغيرات عن طريق التحقق من محرفها الأول أثناء معالجة الاستعلام ينطوي على هدر نوعًا ما. والحل الأكثر كفاءة وسيّضع وسمًا منفصلاً لمتغيرات الأنماط في التمثيل الخاص بلغة الاستعلام وسيستخدم بناء الأزواج بدلاً من دمج السلاسل لإنشاء متغيرات جديدة. وقد اخترنا الحل الأقل كفاءة لتبسيط العرض.]
#idx("makenewvariable", decl: true)
#snippet(```python
rule_counter = 0
def new_rule_application_id():
    rule_counter = rule_counter + 1
    return rule_counter
def make_new_variable(variable, rule_application_id):
    return make_name(symbol_of_name(variable) + "_" + stringify(rule_application_id))
```)

#subheading([الدالة #py("convert_to_query_syntax")])

تُحَوِّل الدالة #py("convert_to_query_syntax")
عوديًا
#idx("تمثيل خاص بلغة الاستعلام", sub: "تحويل بناء جملة بايثون إلى")
تمثيل بناء جملة #en[Python] إلى التمثيل الخاص بلغة الاستعلام عن طريق تبسيط التقريرات الجازمة والقواعد والاستعلامات بحيث يصبح رمز الاسم في تعبير الدالة لتطبيق ما وسمًا، باستثناء أنه إذا كان الرمز هو #py("\"pair\"") أو #py("\"list\"")، فإن زوج أو قائمة #en[Python] (بدون وسم) يُبْنَى. وهذا يعني أن #py("convert_to_query_syntax") تُفَسِّر تطبيقات مُنْشِئات #py("pair") و#py("list") أثناء التحويل، ودوال المعالجة مثل #py("pattern_match") في القسم @sec:query-match و#py("unify_match") في القسم @sec:query-unify يمكنها العمل مباشرة على الأزواج والقوائم المقصودة بدلاً من العمل على تمثيل بناء الجملة المتولد بواسطة المحلل.
وقائمة «الوسائط» (من عنصر واحد) لـ #py("javascript_predicate") تظل غير معالجة، كما هو موضح أدناه.
والمتغير يظل دون تغيير، والتعبير الحرفي يُبَسَّط إلى القيمة الأولية التي يحتوي عليها.
#idx("converttoquerysyntax", decl: true)
#snippet(```python
def convert_to_query_syntax(exp):
    if is_application(exp):
        function_symbol = symbol_of_name(function_expression(exp))
        if function_symbol == "javascript_predicate":
            return pair(function_symbol, arg_expressions(exp))
        else:
            processed_args = map(convert_to_query_syntax, arg_expressions(exp))
            return pair(head(processed_args), head(tail(processed_args))) if function_symbol == "pair" else processed_args if function_symbol == "list" else pair(function_symbol, processed_args)
    elif is_variable(exp):
        return exp
    else:
        return literal_value(exp)
```)

واستثناء هذا المعالجة هو #py("javascript_predicate").
فبما أن تمثيل بناء جملة #en[Python] المُجَسَّد لتعبير محموله يُمرَّر إلى #py("evaluate") في القسم @sec:core-of-evaluator، يجب أن يظل تمثيل بناء الجملة الأصلي القادم من #py("parse") سليمًا في التمثيل الخاص بلغة الاستعلام للتعبير.
وفي هذا المثال من القسم @sec:deductive-info-retrieval

#snippet(```python
and(salary($person, $amount), javascript_predicate($amount > 50000))
```)

تنتج #py("convert_to_query_syntax") بنية بيانات يُدْمَج فيها تمثيل بناء جملة #en[Python] في تمثيل خاص بلغة الاستعلام:

#output(```python
and(salary($person, $amount), javascript_predicate($amount > 50000))
```)

ومن أجل تقييم التعبير الفرعي #py("javascript_predicate") لذلك الاستعلام المعالج، تدعو الدالة #py("javascript_predicate") في القسم @sec:query-eval الدالة #py("instantiate_expression") (أدناه) على تمثيل بناء جملة #en[Python] المدمج لـ #py("$amount > 50000") لاستبدال المتغير #py("list(\"name\", \"$amount\")") بـ تعبير حرفي، على سبيل المثال #py("list(\"literal\", 70000)")، يمثل القيمة الأولية التي تُربَط بها #py("$amount")، هنا 70000.
ومُقَيِّم #en[Python] يمكنه تقييم المحمول المُجَسَّد، الذي يمثل الآن #py("70000 > 50000").

#subheading([تجسيد تعبير])

#idx("مفسر الاستعلام", sub: "تجسيد")

تدعو الدالة #py("javascript_predicate") في القسم @sec:query-eval وحلقة المحرك في القسم @sec:query-driver الدالة #py("instantiate_expression") على تعبير ما للحصول على نسخة يستبدل فيها أي متغير في التعبير بقيمته في إطار معطى.
وتستخدم تعبيرات المدخلات والنتيجة تمثيل بناء جملة #en[Python]، لذا فإن أي قيمة تنتج عن تجسيد متغير تحتاج إلى التحويل من شكلها في الربط إلى تمثيل بناء جملة #en[Python].
#idx("instantiateexpression", decl: true)
#snippet(```python
def instantiate_expression(expression, frame):
    return convert(instantiate_term(expression, frame)) if is_variable(expression) else pair(instantiate_expression(head(expression), frame), instantiate_expression(tail(expression), frame)) if is_pair(expression) else expression
```)

وتأخذ الدالة #py("instantiate_term") متغيرًا، أو زوجًا، أو قيمة أولية كوسيط أول وإطارًا كوسيط ثانٍ وتستبدل عوديًا المتغيرات في الوسيط الأول بقيمها في الإطار حتى يتم الوصول إلى قيمة أولية أو متغير غير مرتبط.
وعندما تواجه العملية زوجًا، يُبْنَى زوج جديد تكون أجزاؤه هي النسخ المُجَسَّدة للأجزاء الأصلية.
على سبيل المثال، إذا كانت #py("$x") مرتبطة بالزوج $[mono("$y"), 5]$ في إطار $f$ كنتيجة للتوحيد، وكانت #py("$y") مرتبطة بدورها بـ 3، فإن نتيجة تطبيق #py("instantiate_term") على #py("list(\"name\", \"$x\")") و$f$ هي الزوج $[3, 5]$.
#idx("instantiateterm", decl: true)
#syntax("
def instantiate_term(term, frame):
    if is_variable(term):
        binding = binding_in_frame(term, frame)
        return term if is_none(binding) else instantiate_term(binding_value(binding), frame)
    elif is_pair(term):
        return pair(instantiate_term(head(term), frame), instantiate_term(tail(term), frame))
    else:
        return term
")

وتُبْنِي الدالة #py("convert") تمثيل بناء جملة #en[Python] لمتغير، أو زوج، أو قيمة أولية مرجعة بواسطة #py("instantiate_term").
والزوج في الأصل يصبح تطبيقًا لـ مُنْشِئ الأزواج في #en[Python] والقيمة الأولية تصبح تعبيرًا حرفيًا.
#idx("convert", decl: true)
#syntax("
def convert(term):
    return term if is_variable(term) else make_application(make_name(\"pair\"), llist(convert(head(term)), convert(tail(term)))) if is_pair(term) else make_literal(term)
")

ولكشف هذه الدوال الثلاث، تأمل ما يحدث عندما يُعَالَج الاستعلام

#snippet(```python
job($x, llist("computer", "wizard"))
```)

الذي أُعْطِيَ تمثيل بناء جملة #en[Python] الخاص به في بداية القسم @sec:query-syntax، بواسطة حلقة المحرك.
لنقل إن إطارًا $g$ من تدفق النتيجة يربط المتغير #py("$x") بالزوج $[mono("\"Bitdiddle\""), mono("$y")]$ والمتغير #py("$y") بالزوج $[mono("\"Ben\""), mono("null")]$، فإن

#syntax("
instantiate_term(llist(\"name\", \"$\\$$x\"), ", meta("g"), ")
	    ")

يرجع القائمة

#output(```python
job($x, llist("computer", "wizard"))
```)

والتي تحولها #py("convert") إلى

#output(```python
job($x, llist("computer", "wizard"))
```)

ونتيجة تطبيق #py("instantiate_expression") على تمثيل بناء جملة #en[Python] للاستعلام والإطار $g$ هي:

#output(```python
job($x, llist("computer", "wizard"))
```)

وتلغي حلقة المحرك الفهرسة النحوية لهذا التمثيل وتعرضه كالتالي:

#output(```python
job($x, llist("computer", "wizard"))
```)

#idx("مفسر الاستعلام", sub: "تجسيد")

#subheading([الدالة #py("unparse")])

فيما يلي إعلانات المحمولات والمحدات للأشكال النحوية #py("and") و #py("or") و #py("not") و #py("javascript_predicate") (القسم @sec:query-eval):
#idx("isemptyconjunction", decl: true)#idx("firstconjunct", decl: true)#idx("restconjuncts", decl: true)#idx("isemptydisjunction", decl: true)#idx("firstdisjunct", decl: true)#idx("restdisjuncts", decl: true)#idx("negatedquery", decl: true)#idx("javascriptpredicateexpression", decl: true)
#snippet(```python
def is_empty_conjunction(exps):
    return is_none(exps)

def first_conjunct(exps):
    return head(exps)

def rest_conjuncts(exps):
    return tail(exps)
def is_empty_disjunction(exps):
    return is_none(exps)

def first_disjunct(exps):
    return head(exps)

def rest_disjuncts(exps):
    return tail(exps)
def negated_query(exps):
    return head(exps)
def javascript_predicate_expression(exps):
    return head(exps)
```)

الدوال الثلاث التالية تُحدد التمثيل الخاص بلغة الاستعلام للقواعد:
#idx("isrule", decl: true)#idx("conclusion", decl: true)#idx("rulebody", decl: true)
#snippet(```python
def is_rule(assertion):
    return is_tagged_list(assertion, "rule")
def conclusion(rule):
    return head(tail(rule))

def rule_body(rule):
    return llist("always_true") if is_none(tail(tail(rule))) else head(tail(tail(rule)))
```)

#idx("مفسر الاستعلام", sub: "بناء جملة لغة الاستعلام")
#idx("تمثيل خاص بلغة الاستعلام")

#subsubsection([الإطارات والروابط], label-name: <sec:query-bindings>)

#idx("مفسر الاستعلام", sub: "الإطار")
#idx("إطار (مفسر الاستعلام)", sub: "التمثيل")

تُمَثَّل الإطارات كقوائم من الروابط، والتي هي أزواج من متغير وقيمة:
#idx("makebinding", decl: true)#idx("bindingvariable", decl: true)#idx("bindingvalue", decl: true)#idx("bindinginframe", decl: true)#idx("extend", decl: true)
#snippet(```python
def make_binding(variable, value):
    return pair(variable, value)
def binding_variable(binding):
    return head(binding)
def binding_value(binding):
    return tail(binding)
def binding_in_frame(variable, frame):
    return assoc(variable, frame)
def extend(variable, value, frame):
    return pair(make_binding(variable, value), frame)
```)

#idx("مفسر الاستعلام", sub: "الإطار")

#exercise(label-name: <ex:q-why-not-append>, [
يتساءل لويس ريزونر لماذا أُدْرِجَت الدالتان #py("simple_query") و #py("disjoin") (القسم @sec:query-eval) باستخدام تعبيرات مؤجلة بدلاً من تعريفها كالتالي:

#idx("simplequery", sub: "بدون تعبير مؤجل", decl: true)#idx("disjoin", sub: "بدون تعبير مؤجل", decl: true)
#snippet(```python
def simple_query(query_pattern, frame_stream):
    return stream_flatmap(lambda frame: (stream_append(find_assertions(query_pattern, frame), apply_rules(query_pattern, frame))), frame_stream)
def disjoin(disjuncts, frame_stream):
    return None if is_empty_disjunction(disjuncts) else interleave(evaluate_query(first_disjunct(disjuncts), frame_stream), disjoin(rest_disjuncts(disjuncts), frame_stream))
```)

هل يمكنك إعطاء أمثلة على استعلامات قد تُؤدي فيها هذه التعاريف الأبسط إلى سلوك غير مرغوب فيه؟
])

#exercise(label-name: <ex:q-why-interleave>, [
لماذا تقوم #py("disjoin") و #py("stream_flatmap") بتداخل التدفقات بدلاً من مجرد إلحاقها؟ أعطِ أمثلة تُوضّح سبب عمل التداخل بشكل أفضل. (تلميح: لماذا استخدمنا #py("interleave") في القسم @sec:exploiting-streams؟)
])

#exercise(label-name: <ex:q-why-delay>, [
لماذا تستخدم #py("flatten_stream") تعبيرًا مؤجلاً في جسمها؟ ما الذي قد يكون خاطئًا لو عُرِّفت على النحو التالي:

#snippet(```python
def flatten_stream(stream):
    return None if is_none(stream) else interleave(head(stream), flatten_stream(stream_tail(stream)))
```)
])

#exercise(label-name: <ex:4_71>, [
تقترح أليسا ب. هاكر استخدام نسخة أسهل من #idx("streamflatmap") #py("stream_flatmap") في #py("negate") و #py("javascript_predicate") و #py("find_assertions"). وهي تلاحظ أن الدالة التي تُطبَّق على تدفق الإطارات في هذه الحالات تُنتج دائمًا إما تدفقًا فارغًا أو تدفقًا أحاديًا، لذا لا داعي للتداخل عند تجميع هذه التدفقات.

+ املأ التعبيرات المفقودة في برنامج أليسا. #syntax(" function simple_stream_flatmap(fun, s) { return simple_flatten(stream_map(fun, s)) } function simple_flatten(stream) { return stream_map(", metaphrase[??], ", stream_filter(", metaphrase[??], ", stream)) } ")
+ هل يتغير سلوك نظام الاستعلام إذا قمنا بتغييره بهذه الطريقة؟
])

#exercise(label-name: <ex:unique>, [
نَفِّذ للغة الاستعلام شكلًا نحويًا يُسمّى #idx("لغة الاستعلام", sub: "توسيع لـ") #idx("استعلام مركب", sub: "معالجة") #idx("unique (لغة الاستعلام)") #py("unique").
ينبغي أن تنجح تطبيقات #py("unique") إذا كان هناك عنصر واحد فقط في قاعدة البيانات يستوفي استعلامًا مُحدّدًا. على سبيل المثال،

#snippet(```python
unique(job($x, llist("computer", "wizard")))
```)

ينبغي أن يطبع تدفقًا ذو عنصر واحد

#snippet(```python
unique(job(llist("Bitdiddle", "Ben"), llist("computer", "wizard")))
```)

نظراً لأن بن هو ساحر الحاسوب الوحيد، و

#snippet(```python
unique(job($x, llist("computer", "programmer")))
```)

ينبغي أن يطبع التدفق الفارغ، نظراً لوجود أكثر من مبرمج حاسوب واحد. علاوة على ذلك،

#snippet(```python
and(job($x, $j), unique(job($anyone, $j)))
```)

ينبغي أن يعرض جميع الوظائف التي يشغلها شخص واحد فقط، والأشخاص الذين يشغلونها.

هناك جزءان لتنفيذ #py("unique"). الأول هو كتابة دالة تتعامل مع هذا الشكل النحوي، والثاني هو جعل #py("evaluate_query") تُوجّه إلى تلك الدالة. الجزء الثاني بسيط، لأن #py("evaluate_query") تُجري توجيهها بطريقة موجهة بالبيانات. إذا كانت دالتك تُسمّى #py("uniquely_asserted")، فكل ما عليك فعله هو

#snippet(```python
put("unique", "evaluate_query", uniquely_asserted)
```)

وسوف توجّه #py("evaluate_query") إلى هذه الدالة لكل استعلام يكون #py("type") (رأسه) هو السلسلة النصية #py("\"unique\"").

المسألة الحقيقية هي كتابة الدالة #py("uniquely_asserted"). ينبغي أن تأخذ هذه الدالة كمدخل #py("contents") (ذيل) استعلام #py("unique")، بالإضافة إلى تدفق من الإطارات. ولكل إطار في التدفق، ينبغي أن تستخدم #py("evaluate_query") لإيجاد تدفق جميع التوسيعات للإطار التي تستوفي الاستعلام المعطى. ويجب استبعاد أي تدفق لا يحتوي على عنصر واحد بالضبط. ويجب إرجاع التدفقات المتبقية ليتم تجميعها في تدفق واحد كبير وهو نتيجة استعلام #py("unique"). هذا مشابه لتنفيذ الشكل النحوي #py("not").

اختبر تطبيقك من خلال صياغة استعلام يعرض جميع الأشخاص الذين يشرفون على شخص واحد بالضبط.
])

#exercise(label-name: <ex:q-exponential-and>, [
تطبيقنا لـ #py("and") كتركيب متسلسل للاستعلامات #idx("مفسر الاستعلام", sub: "تحسينات لـ") #idx("and (لغة الاستعلام)", sub: "تقييم") #idx("استعلام مركب", sub: "معالجة") (الشكل @fig:query-and) أنيق، ولكنه غير كفء لأننا في معالجة الاستعلام الثاني لـ #py("and") يجب أن نفحص قاعدة البيانات لكل إطار يُنتجه الاستعلام الأول. إذا كانت قاعدة البيانات تحتوي على $N$ عنصرًا، وكان استعلام نموذجي يُنتج عددًا من إطارات المخرجات يتناسب مع $N$ (مثلاً $N/k$)، فإن فحص قاعدة البيانات لكل إطار يُنتجه الاستعلام الأول سيتطلب $N^(2)/k$ استدعاءً لمُطابق الأنماط.
وهناك نهج آخر وهو معالجة بندي #py("and") بشكل منفصل، ثم البحث عن جميع أزواج إطارات المخرجات المتوافقة. إذا كان كل استعلام يُنتج $N/k$ إطار مخرجات، فإن هذا يعني أنه يجب علينا إجراء $N^(2)/k^(2)$ فحص توافق—أي أقل بعامل $k$ من عدد المطابقات المطلوبة في طريقتنا الحالية.

ابتكر تنفيذًا لـ #py("and") يستخدم هذه الاستراتيجية. يجب عليك تنفيذ دالة تأخذ إطارين كمدخلات، وتفحص ما إذا كانت الروابط في الإطارين متوافقة، وإذا كان الأمر كذلك، تُنتج إطارًا يدمج مجموعتي الروابط. هذه العملية مشابهة للتوحيد.
])

#exercise(label-name: <ex:not-query-filter>, [
في القسم @sec:math-logic رأينا أن #idx("مفسر الاستعلام", sub: "تحسينات لـ") #idx("مفسر الاستعلام", sub: "مشكلات مع not و javascriptpredicate") #idx("استعلام مركب", sub: "معالجة") #idx("not (لغة الاستعلام)", sub: "تقييم") #py("not") و #idx("javascriptpredicate (لغة الاستعلام)", sub: "تقييم") #py("javascript_predicate") يمكن أن يسببا إعطاء لغة الاستعلام إجابات "خاطئة" إذا أُطبّقت عمليات التصفية هذه على إطارات تظل فيها المتغيرات غير مرتبطة. ابتكر طريقة لإصلاح هذا القصور. إحدى الأفكار هي إجراء التصفية بطريقة "مؤجلة" عن طريق إلحاق "وعد" بالتصفية بالإطار يُوفّى فقط عندما تُربط متغيرات كافية لجعل العملية ممكنة. يمكننا الانتظار لإجراء التصفية حتى تُنفّذ جميع العمليات الأخرى. ومع ذلك، من أجل الكفاءة، نود إجراء التصفية في أقرب وقت ممكن من أجل تقليل عدد الإطارات المتوسطة المُولَّدة.
])

#exercise(label-name: <ex:query-lang-amb>, [
أعد تصميم لغة الاستعلام كبرنامج #idx("مفسر الاستعلام", sub: "كبرنامج غير محدد") غير محدد ليتم تنفيذه باستخدام المُقيِّم في القسم @sec:nondeterministic-evaluation، بدلاً من كونه عملية تدفقية. في هذا النهج، سيعطي كل استعلام إجابة واحدة (بدلاً من تدفق كل الإجابات) ويمكن للمستخدم كتابة #py("retry") لرؤية المزيد من الإجابات. ينبغي أن تجد أن الكثير من الآليات التي بنيناها في هذا القسم تُستبدل بالبحث غير المحدد والتراجع. ومع ذلك، ستجد على الأرجح أن لغتك الجديدة للاستعلام بها اختلافات دقيقة في السلوك عن تلك المُطبَّقة هنا. هل يمكنك إيجاد أمثلة تُوضّح هذا الاختلاف؟
])

#exercise(label-name: <ex:query-local-names>, [
عندما نفّذنا مُفسِّر #en[Python] في القسم @sec:mc-eval، رأينا كيفية استخدام البيئات المحلية لتجنب #idx("بيئة", sub: "إعادة تسمية مقابل") #idx("مفسر الاستعلام", sub: "مُفسِّر Python مقابل") تضارب الأسماء بين وسائط الدوال. على سبيل المثال، في تقييم

#snippet(```python
def square(x):
    return x * x
def sum_of_squares(x, y):
    return square(x) + square(y)
sum_of_squares(3, 4)
```)

لا يوجد خلط بين #py("x") في #py("square") و #py("x") في #py("sum_of_squares")، لأننا نُقيّم جسم كل دالة في بيئة صُمِّمت خصيصًا لتحتوي على روابط للأسماء المحلية. وفي نظام الاستعلام، استخدمنا استراتيجية مختلفة لتجنب تضارب الأسماء في تطبيق القواعد. في كل مرة نُطبّق فيها قاعدة، نُعيد تسمية المتغيرات بأسماء جديدة نضمن أن تكون فريدة. والاستراتيجية المماثلة لـ مُفسِّر #en[Python] هي الاستغناء عن البيئات المحلية وإعادة تسمية المتغيرات في جسم الدالة في كل مرة نُطبّق فيها الدالة.

نَفِّذ للغة الاستعلام طريقة تطبيق قواعد تستخدم البيئات بدلاً من إعادة التسمية. وانظر ما إذا كان بإمكانك البناء على بنية البيئة الخاصة بك لإنشاء تراكيب في لغة الاستعلام للتعامل مع الأنظمة الكبيرة، مثل نظير القاعدة لـ #idx("بنية الكتلة", sub: "في لغة الاستعلام") #idx("بيئة", sub: "في مفسر الاستعلام") #idx("مفسر الاستعلام", sub: "بنية البيئة في") #idx("قاعدة (لغة الاستعلام)", sub: "تطبيق") الدوال ذات بنية الكتلة. هل يمكنك ربط أي من هذا بمسألة إجراء الاستنباطات في سياق ما (مثلاً، "إذا افترضت أن $P$ صحيحة، ففسأكون قادرًا على استنباط $A$ و $B$.") كطريقة لحل المشكلات؟ (هذه المسألة مفتوحة المصدر).
])
