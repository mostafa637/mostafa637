// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([تمثيل المكونات], label-name: <sec:representing-expressions>)

#idx("metacircular evaluator for Python", sub: "component representation")
#idx("metacircular evaluator for Python", sub: "syntax of evaluated language")
#idx("parsing Python")

يكتب المبرمجون البرامج كنص، أي تسلسلات من المحارف، تُدخل في بيئة برمجة أو محرر نصوص. ولتشغيل مُقيِّمنا، نحتاج إلى البدء بـ تمثيل لـ نص البرنامج هذا كقيمة في Python.
في القسم @sec:strings قدمنا السلاسل النصية لتمثيل النص. ونود تقييم برامج مثل
#py("\"size = 2\\n5 * size\"")
من القسم @sec:naming.
لسوء الحظ، فإن مثل نص البرنامج هذا لا يوفر هيكلاً كافياً للمُقيِّم. في هذا المثال، تبدو أجزاء البرنامج
#py("\"size = 2\"") و
#py("\"5 * size\"") متشابهة، لكنها تحمّل معاني مختلفة جداً. والدوال النحوية التجريدية مثل
#py("declaration_value_expression") ستكون صعبة وغير ضامنة للسلامة لتنفيذها بـ فحص نص البرنامج.
في هذا القسم، نقدم لذلك دالة
#idx("parse")
#py("parse") تترجم نص البرنامج إلى
#emph[تمثيل القائمة المُعنونة] (#en[tagged-list representation])، المُذكرة بالبيانات المُعنونة في القسم @sec:manifest-types.
على سبيل المثال، فإن تطبيق #py("parse") على السلسلة النصية للبرنامج أعلاه ينتج هيكل بيانات يعكس هيكل البرنامج: تسلسل يتكون من إعلان ثابت يربط الاسم
#py("size") بالقيمة 2 وعملية ضرب.

#snippet(```python
parse("size = 2\n5 * size")
```)

#output(```python
parse("size = 2\n5 * size")
```)

تصل الدوال النحوية المستخدمة بواسطة المُقيِّم إلى تمثيل القائمة المُعنونة المُنتج بواسطة
#py("parse").

يُذكر المُقيِّم بـ برنامج
#idx("metacircular evaluator for Python", sub: "symbolic differentiation and")
التفاضل الرمزي
المُناقش في القسم @sec:symbolic-differentiation.
كِلا البرنامجين يعملان على بيانات رمزية.
وفي كِلا البرنامجين، تتحدد نتيجة العمل على كائن ما بـ العمل عودياً على أجزاء الكائن ودمج النتائج بطريقة تعتمد على نوع الكائن.
في كِلا البرنامجين استخدمنا
#idx("data abstraction")
تجريد البيانات لفصل القواعد العامة للعمليات عن تفاصيل كيفية تمثيل الكائنات. في برنامج التفاضل، كان هذا يعني أن دالة التفاضل نفسها يمكنها التعامل مع التعبيرات الجبرية في البادئة، أو الوسطية، أو في شكل آخر. وبالنسبة للمُقيِّم، هذا يعني أن البناء النحوي للغة الجاري تقييمها يتحدد فقط بـ #py("parse") والدوال التي تصنف وتستخرج أجزاء القوائم المُعنونة المُنتجة بواسطة #py("parse").

#sicp-figure(image("/images/img_javascript/ch4-parse-abstraction.svg", width: 70%), caption: [تجريد البناء النحوي في المُقيِّم.], label-name: <fig:parse-abstraction>)

يوضح الشكل @fig:parse-abstraction
#idx("abstraction barriers", sub: "in representing Python syntax")
حاجز التجريد
المُشكل بواسطة الدوال الشرطية والمحددات النحوية، والتي تصل بين المُقيِّم وتمثيل القائمة المُعنونة للبرامج، والتي بدورها تُفصل عن تمثيل السلسلة النصية بواسطة #py("parse"). ونصف أدناه إعراب مكونات البرامج ونردج الدوال الشرطية والمحددات النحوية المقابلة، بالإضافة إلى المنشئات إذا كانت مطلوبة.

#idx("parsing Python")

#subheading([التعبير الحرفي])

التعبيرات الحرفية

#idx("literal expression", sub: "parsing of")
تُعرب إلى قوائم مُعنونة بالعنوان #py("\"literal\"") والقيمة الفعلية.

$ mat(delim: #none, lt.double space italic("literal")-italic("expression") space gt.double, =, mono("list(\"literal\", ")italic("value")mono(")")) $

حيث #meta("value") هي قيمة Python الممثلة بـ السلسلة النصية لـ #meta("literal-expression").
وهنا يرمز $lt.double space italic("literal")-italic("expression") space gt.double$ إلى نتيجة إعراب السلسلة النصية #meta("literal-expression").

#snippet(```python
parse("1;")
```)

#output(```python
parse("1;")
```)

#snippet(```python
parse("'hello world';")
```)

#output(```python
parse("'hello world';")
```)

#snippet(```python
parse("null;")
```)

#output(```python
parse("null;")
```)

الدالة الشرطية النحوية للتعبيرات الحرفية هي
#py("is_literal").
#idx("isliteral", decl: true)
#snippet(```python
def is_literal(component):
    return is_tagged_list(component, "literal")
```)

تُمّ تعريفها بدلالة الدالة #py("is_tagged_list")، والتي تحدد القوائم التي تبدأ بالسلسلة النصية المحددة:
#idx("istaggedlist", decl: true)
#snippet(```python
def is_tagged_list(component, the_tag):
    return is_pair(component) and head(component) == the_tag
```)

العنصر الثاني في القائمة الناتجة عن إعراب تعبير حرفي هو قيمته الفعلية في Python.
والمحدد لاسترداد القيمة هو #py("literal_value").
#idx("literalvalue", decl: true)
#snippet(```python
def literal_value(component):
    return head(tail(component))
```)

#snippet(```python
literal_value(parse("null;"))
```)

#output(```python
literal_value(parse("null;"))
```)

في باقي هذا القسم، نكتفي بإدراج الدوال الشرطية والمحددات النحوية، ونحذف إعلاناتها إذا كانت مجرد وصول إلى عناصر القائمة الواضحة.

نوفر منشئاً للحرفيات، والذي سيكون مفيداً:
#idx("makeliteral", decl: true)
#snippet(```python
def make_literal(value):
    return llist("literal", value)
```)

#subheading([الأسماء])

يتضمن تمثيل القائمة المُعنونة لـ

#idx("name", sub: "parsing of")
#idx("symbol(s)", sub: "in parsing of names")
الأسماء العنوان #py("\"name\"") كعنصر أول والسلسلة النصية التي تمثل الاسم كعنصر ثاني.

$ mat(delim: #none, lt.double space italic("name") space gt.double, =, mono("list(\"name\", ")italic("symbol")mono(")")) $

حيث #meta("symbol") هي سلسلة نصية تحتوي على المحارف التي تشكل #meta("name") كما هي مكتوبة في البرنامج.

الدالة الشرطية النحوية للأسماء هي
#idx("isname")
#py("is_name").

يُوصل إلى الرمز باستخدام المحدد
#idx("symbolofname")
#py("symbol_of_name").

نوفر منشئاً للأسماء، لاستخدامه بواسطة #py("operator_combination_to_application"):
#idx("makename", decl: true)
#snippet(```python
def make_name(symbol):
    return llist("name", symbol)
```)

#subheading([عبارات التعبير])

لا نحتاج إلى التمييز بين التعبيرات و

#idx("expression statement", sub: "parsing of")
عبارات التعبير.
وبالتالي، فإن #py("parse") يمكنها تجاهل الفرق بين النوعين من المكونات:

$ mat(delim: #none, lt.double space italic("expression")mono(";") space gt.double, =, lt.double space italic("expression") space gt.double) $

#subheading([تطبيقات الدوال])

تطبيقات الدوال

#idx("function application", sub: "parsing of")
تُعرب كما يلي:

#syntax($lt.double space$, meta("fun-expr"), "(", meta("arg-expr"), $""_(1)$, ", ", $dots.h$, ", ", meta("arg-expr"), $""_(n)$, ")", $space gt.double$, " =
     llist(\"application\",
          ", $lt.double space$, meta("fun-expr"), $space gt.double$, ",
          llist(", $lt.double space$, meta("arg-expr"), $""_(1) med gt.double$, ", ", $dots.h$, ", ", $lt.double space$, meta("arg-expr"), $""_(n) med gt.double$, "))
	  ")

نعلن عن #idx("isapplication") #py("is_application") كدالة شرطية نحوية وعن #idx("functionexpression") #idx("argexpressions") #py("function_expression") و #py("arg_expressions") كمحددات.

نضيف منشئاً لتطبيقات الدوال، لاستخدامه بواسطة #py("operator_combination_to_application"):
#idx("makeapplication", decl: true)
#snippet(```python
def make_application(function_expression, argument_expressions):
    return llist("application", function_expression, argument_expressions)
```)

#subheading([الشرطيات])

التعبيرات الشرطية

#idx("conditional expression", sub: "parsing of")
#idx("conditional statement", sub: "parsing of")
تُعرب كما يلي:

#syntax($lt.double space$, meta("predicate"), " ? ", meta("consequent-expression"), " : ", meta("alternative-expression"), $space gt.double$, " =
        llist(\"conditional_expression\",
             ", $lt.double space$, meta("predicate"), $space gt.double$, ",
             ", $lt.double space$, meta("consequent-expression"), $space gt.double$, ",
             ", $lt.double space$, meta("alternative-expression"), $space gt.double$, ")
	  ")

وبالمثل، تُعرب العبارات الشرطية كما يلي:

#syntax($lt.double space$, "if (", meta("predicate"), ") ", meta("consequent-block"), " else ", meta("alternative-block"), $space gt.double$, " =
        llist(\"conditional_statement\",
             ", $lt.double space$, meta("predicate"), $space gt.double$, ",
             ", $lt.double space$, meta("consequent-block"), $space gt.double$, ",
             ", $lt.double space$, meta("alternative-block"), $space gt.double$, ")
	  ")

الدالة الشرطية النحوية
#idx("isconditional")
#py("is_conditional")
ترجع صح لكلا النوعين من الشرطيات، والمحددات
#idx("conditionalpredicate")
#py("conditional_predicate")،
#idx("conditionalconsequent")
#py("conditional_consequent")، و
#idx("conditionalalternative")
#py("conditional_alternative")
يمكن تطبيقها على كلا النوعين.

#subheading([تعبيرات الدوال غير المسمات])

تعبير الدالة غير المسمات (#en[lambda])

#idx("lambda expression", sub: "parsing of")
الذي جسمه تعبير يُعرب كما لو أن الجسم يتكون من كتلة تحتوي على عبارة إرجاع واحدة يكون تعبير إرجاعها هو جسم تعبير #en[lambda].

#syntax($lt.double space$, "(", meta("name"), $""_(1)$, ", ", $dots.h$, ", ", meta("name"), $""_(n)$, ") => ", meta("expression"), $space gt.double$, " =
    ", $lt.double space$, "(", meta("name"), $""_(1)$, ", ", $dots.h$, ", ", meta("name"), $""_(n)$, ") => { return ", meta("expression"), "; }", $space gt.double$)

وتعبير #en[lambda] الذي جسمه كتلة يُعرب كما يلي:

#syntax($lt.double space$, "(", meta("name"), $""_(1)$, ", ", $dots.h$, ", ", meta("name"), $""_(n)$, ") => ", meta("block"), $space gt.double$, " =
     llist(\"lambda_expression\",
          llist(", $lt.double space$, meta("name"), $""_(1) med gt.double$, ", ", $dots.h$, ", ", $lt.double space$, meta("name"), $""_(n) med gt.double$, "),
          ", $lt.double space$, meta("block"), $space gt.double$, ")
	  ")

الدالة الشرطية النحوية هي
#idx("islambdaexpression")
#py("is_lambda_expression")
والمحدد لجسم تعبير #en[lambda] هو
#idx("lambdabody")
#py("lambda_body").
المحدد للبارامترات، المسمى
#py("lambda_parameter_symbols")،
يستخرج بالإضافة إلى ذلك الرموز من الأسماء.
#idx("lambdaparametersymbols", decl: true)
#snippet(```python
def lambda_parameter_symbols(component):
    return map(symbol_of_name, head(tail(component)))
```)

تحتاج الدالة #py("function_decl_to_constant_decl") منشئاً لتعبيرات #en[lambda]:
#idx("makelambdaexpression", decl: true)
#snippet(```python
def make_lambda_expression(parameters, body):
    return llist("lambda_expression", parameters, body)
```)

#subheading([التسلسلات])

عبارة التسلسل

#idx("sequence of statements", sub: "parsing of")
تغلف تسلسلاً من العبارات في عبارة واحدة. ويُعرب تسلسل العبارات كما يلي:

#syntax($lt.double space$, meta("statement"), $""_(1)$, " ", $dots.c$, " ", meta("statement"), $""_(n) med gt.double$, " =
     llist(\"sequence\", llist(", $lt.double space$, meta("statement"), $""_(1) med gt.double$, ", ", $dots.h$, ", ", $lt.double space$, meta("statement"), $""_(n) med gt.double$, "))
	  ")

الدالة الشرطية النحوية هي
#idx("issequence")
#py("is_sequence") و
المحدد هو #py("sequence_statements").
ونسترد الأولى من قائمة العبارات باستخدام
#py("first_statement") والعبارات المتبقية باستخدام
#py("rest_statements"). ونختبر ما إذا كانت القائمة فارغة باستخدام الدالة الشرطية
#py("is_empty_sequence") وما إذا كانت تحتوي على عنصر واحد فقط
باستخدام الدالة الشرطية
#py("is_last_statement").#footnote[هذه المحددات لقائمة العبارات ليست مقصودة كتجريد للبيانات.
بل تُمّ تقديمها كأسماء تذكيرية لعمليات القائمة الأساسية لجعل فهم المُقيِّم صريح التحكم في القسم @sec:eceval أسهل.]<foot:mceval-abstraction>
#idx("firststatement", decl: true)#idx("reststatements", decl: true)#idx("isemptysequence", decl: true)#idx("islaststatement", decl: true)
#snippet(```python
def first_statement(stmts):
    return head(stmts)

def rest_statements(stmts):
    return tail(stmts)

def is_empty_sequence(stmts):
    return is_none(stmts)

def is_last_statement(stmts):
    return is_none(tail(stmts))
```)

#subheading([الكتل])

الكتل

#idx("block", sub: "parsing of")
تُعرب كما يلي:#footnote[قد يقرر تنفيذ المُعرب تمثيل الكتلة بـ تسلسل عباراتها فقط إذا لم تكن أي من عبارات التسلسل إعلانات، أو تمثيل تسلسل يحتوي على عبارة واحدة فقط بـ تلك العبارة. معالجات اللغة في هذا الفصل وفي الفصل 5 لا تعتمد على هذه القرارات.]

$ mat(delim: #none, lt.double space mono(\{) space italic("statements") space mono(\}) space gt.double, =, mono("list(\"block\",") space lt.double space italic("statements") space gt.double mono(")")) $

وهنا تشير #meta("statements") إلى تسلسل عبارات، كما هو موضح أعلاه.
الدالة الشرطية النحوية هي
#idx("isblock")
#py("is_block")
والمحدد هو
#idx("blockbody")
#py("block_body").

#subheading([عبارات الإرجاع])

عبارات الإرجاع

#idx("return statement", sub: "parsing of")
تُعرب كما يلي:

$ mat(delim: #none, lt.double space bold(mono("return")) space italic("expression") mono(";") space gt.double, =, mono("list(\"return_statement\",") space lt.double space italic("expression") space gt.double mono(")")) $

الدالة الشرطية النحوية والمحدد هما، على التوالي،
#idx("isreturnstatement")
#py("is_return_statement")
و
#idx("returnexpression")
#py("return_expression").

#subheading([الإسنادات])

الإسنادات

#idx("assignment", sub: "parsing of")
تُعرب كما يلي:

$ mat(delim: #none, lt.double med italic("name") space mono("=") space italic("expression") med gt.double, =, mono("list(\"assignment\",") space lt.double med italic("name") med gt.double mono(", ") lt.double med italic("expression") med gt.double mono(")")) $

الدالة الشرطية النحوية هي
#idx("isassignment")
#py("is_assignment")
والمحددات هي
#py("assignment_symbol")
و
#idx("assignmentvalueexpression")
#py("assignment_value_expression").
والرمز مُغلف في قائمة مُعنونة تمثل الاسم، وبالتالي فإن
#py("assignment_symbol") تحتاج إلى فك تغليفه.
#idx("assignmentsymbol", decl: true)
#snippet(```python
def assignment_symbol(component):
    return head(tail(head(tail(component))))
```)

#subheading([الإسنادات وتعاريف الدوال])

الإسنادات

#idx("assignment", sub: "parsing of")
#idx("variable", sub: "assignment, parsing of")
تُعرب كما يلي:

#syntax($lt.double space$, meta("name"), "$ = $", meta("expression"), $space gt.double$, " =
     llist(\"assignment\", ", $lt.double space$, meta("name"), $space gt.double$, ", ", $lt.double space$, meta("expression"), $space gt.double$, ")
	  ")

المحددات
#py("assignment_symbol") و
#py("assignment_value_expression") تُطبق على كِلا النوعين.
#idx("assignmentsymbol", decl: true)#idx("assignmentvalueexpression", decl: true)
#snippet(```python
def assignment_symbol(component):
    return symbol_of_name(head(tail(component)))
def assignment_value_expression(component):
    return head(tail(tail(component)))
```)

تحتاج الدالة
#py("function_def_to_assignment")
منشئاً لإعلانات الثوابت:
#idx("makeassignment", decl: true)
#snippet(```python
def make_assignment(name, value_expression):
    return llist("assignment", name, value_expression)
```)

تعاريف الدوال

#idx("function definition", sub: "parsing of")
تُعرب كما يلي:

#syntax($lt.double space$, "function ", meta("name"), "(", meta("name"), $""_(1)$, ", ", $dots.h$, " ", meta("name"), $""_(n)$, ") ", meta("block"), $space gt.double$, " =
    llist(\"function_definition\",
         ", $lt.double space$, meta("name"), $space gt.double$, ",
         llist(", $lt.double space$, meta("name"), $""_(1) med gt.double$, ", ", $dots.h$, ", ", $lt.double space$, meta("name"), $""_(n) med gt.double$, "),
         ", $lt.double space$, meta("block"), $space gt.double$, ")
	  ")

الدالة الشرطية النحوية
#idx("isfunctiondefinition")
#py("is_function_definition")
تتعرف على هذه.
والمحددات هي
#idx("functiondefinitionname")
#py("function_definition_name")،
#idx("functiondefinitionparameters")
#py("function_definition_parameters")، و
#idx("functiondefinitionbody")
#py("function_definition_body").

الدالة الشرطية النحوية
#py("is_declaration")
ترجع صح لجميع الأنواع الثلاثة من الإعلانات.
#idx("isdeclaration", decl: true)
#snippet(```python
def is_declaration(component):
    return is_tagged_list(component, "constant_declaration") or is_tagged_list(component, "variable_declaration") or is_tagged_list(component, "function_definition")
```)

المحددات
#py("declaration_symbol") و
#py("declaration_value_expression") تُطبق على جميع
الأنواع الثلاثة.
#idx("declarationsymbol", decl: true)#idx("declarationvalueexpression", decl: true)
#snippet(```python
def declaration_symbol(component):
    return symbol_of_name(head(tail(component)))
def declaration_value_expression(component):
    return head(tail(tail(component)))
```)

تحتاج الدالة
#py("function_decl_to_constant_decl")
منشئاً لإعلانات الثوابت:
#idx("makeconstantdeclaration", decl: true)
#snippet(```python
def make_constant_declaration(name, value_expression):
    return llist("constant_declaration", name, value_expression)
```)

#subheading([المكونات المشتقة])

#idx("derived components in evaluator")
#idx("syntactic form", sub: "as derived component")
#idx("metacircular evaluator for Python", sub: "derived components")
#idx("metacircular evaluator for Python", sub: "syntactic forms as derived components")

بعض الأشكال النحوية
#idx("syntactic form", sub: "as derived component")
في لغتنا يمكن تعريفها بدلالة مكونات تتضمن أشكالاً نحوية أخرى، بدلاً من أن تُنفذ مباشرة.
ومثال على ذلك هو
#idx("function definition", sub: "as derived component")
#idx("derived components in evaluator", sub: "function definition")
تعريف الدالة، والذي تحوله #py("evaluate") إلى إعلان ثابت يكون تعبير قيمته هو تعبير #en[lambda].#footnote[في Python الفعلية، هناك فروق دقيقة بين الشكلين. التمارين تعالج هذه الفروق.]
#idx("functiondecltoconstantdecl", decl: true)
#snippet(```python
def function_decl_to_constant_decl(component):
    return make_constant_declaration( function_definition_name(component), make_lambda_expression( function_definition_parameters(component), function_definition_body(component)))
```)

تنفيذ تقييم تعاريف الدوال بهذه الطريقة يبسط المُقيِّم لأنه يقلل من عدد الأشكال النحوية التي يجب تحديد عملية تقييمها صراحة.

وبالمثل، نُعرّف

#idx("operator combination", sub: "parsing of")
تركيبات المعاملات بدلالة تطبيقات الدوال.
تركيبات المعاملات أحادية أو ثنائية وتحمل رمز معاملها كعنصر ثاني في تمثيل القائمة المُعنونة:

#syntax($lt.double space$, meta("unary-operator"), " ", meta("expression"), $space gt.double$, " =
     llist(\"unary_operator_combination\",
          \"", meta("unary-operator"), "\",
          llist(", $lt.double space$, meta("expression"), $space gt.double$, "))
	  ")

حيث #meta("unary-operator") هو
#py("!") (للنفي المنطقي) أو
#py("-unary") (للنفي العددي)، و

#syntax($lt.double space$, meta("expression"), $""_(1)$, " ", meta("binary-operator"), " ", meta("expression"), $""_(2) med gt.double$, " =
     llist(\"binary_operator_combination\",
          \"", meta("binary-operator"), "\",
          llist(", $lt.double space$, meta("expression"), $""_(1) med gt.double$, ", ", $lt.double space$, meta("expression"), $""_(2) med gt.double$, "))
	  ")

حيث #meta("binary-operator") هو
#py("+")،
#py("-")،
#py("*")،
#py("/")،
#py("%")،
#py("===")،
#py("!==")،
#py(">")،
#py("<")،
#py(">=") أو
#py("<=").
الدوال الشرطية النحوية هي
#py("is_operator_combination")،
#py("is_unary_operator_combination")، و
#py("is_binary_operator_combination")،
والمحددات هي
#py("operator_symbol")،
#py("first_operand")، و
#py("second_operand").

يستخدم المُقيِّم
#py("operator_combination_to_application")
لتحديد
#idx("operator combination", sub: "as function application")
#idx("operator combination", sub: "as derived component")
#idx("derived components in evaluator", sub: "operator combination")
تركيبة المعامل إلى تطبيق دالة يكون تعبير دالتها هو اسم المعامل:
#idx("operatorcombinationtoapplication", decl: true)
#snippet(```python
def operator_combination_to_application(component):
    operator = operator_symbol(component)
    return make_application(make_name(operator), llist(first_operand(component))) if is_unary_operator_combination(component) else make_application(make_name(operator), llist(first_operand(component), second_operand(component)))
```)

المكونات (مثل تعاريف الدوال وتركيبات المعاملات) التي نختار تنفيذها كتحويلات نحوية تُسمى
#idx("derived component")
#emph[المكونات المشتقة]. وتراكيب التركيب المنطقي هي أيضاً مكونات مشتقة (انظر التمرين @ex:eval-and-or).

#idx("metacircular evaluator for Python", sub: "component representation")
#idx("metacircular evaluator for Python", sub: "syntax of evaluated language")
#idx("metacircular evaluator for Python", sub: "derived components")
#idx("metacircular evaluator for Python", sub: "syntactic forms as derived components")
#idx("derived components in evaluator")
#idx("syntactic form", sub: "as derived component")

#exercise(label-name: <ex:parse>, [
العكس لـ #py("parse")
يُسمى
#idx("unparse", sub: "as inverse of parse")
#py("unparse"). ويأخذ كوسيط قائمة مُعنونة كما تنتجها #py("parse")
ويرجع سلسلة نصية تلتزم بـ تدوين Python.

+ اكتب دالة #py("unparse") باتباع هيكل #py("evaluate") (دون وسيط البيئة)، ولكن بإنتاج سلسلة نصية تمثل المكون المعطى، بدلاً من تقييمه. تذكر من القسم @sec:circuit-simulator أن المعامل #py("+") يمكن تطبيقه على سلسلتين نصيتين لربطهما وأن الدالة الأوليّة #py("stringify") تحول القيم مثل 1.5 و صح و #py("None") إلى سلاسل نصية. احرص على احترام أسبقيات المعاملات بـ إحاطة السلاسل النصية الناتجة عن إلغاء إعراب تركيبات المعاملات بالأقواس (دائماً أو عندما يكون ذلك ضرورياً).
+ دالتك #py("unparse") ستكون مفيدة عند حل التمارين اللاحقة في هذا القسم. حسن #py("unparse") بـ إضافة المحارف #py("\" \"") (مسافة) و #py("\"\\n\"") (سطر جديد) إلى السلسلة النصية الناتجة، لاتباع أسلوب المحاذاة #idx("indentation") المستخدَم في برامج Python في هذا الكتاب. وإضافة مثل محارف المسافات البيضاء #idx("whitespace characters") هذه إلى (أو إزالتها من) نص البرنامج لجعل النص أسهل في القراءة تُسمى #idx("pretty-printing") #emph[الطباعة الجميلة].
])

#exercise(label-name: <ex:data-directed-eval>, [
أعد كتابة #py("evaluate") بحيث تُجرى عملية التوجيه بـ أسلوب موجه بالبيانات
#idx("data-directed programming", sub: "in metacircular evaluator")
#idx("metacircular evaluator for Python", sub: "data-directed evaluate")
#idx("evaluate (metacircular)", sub: "data-directed")
قارن هذا بـ دالة التفاضل الموجهة بالبيانات في التمرين @ex:data-directed-differentiation. (يمكنك استخدام عنوان تمثيل القائمة المُعنونة كنوع المكونات.)
])

#exercise(label-name: <ex:eval-and-or>, [
تذكر من القسم @sec:conditionals أن عمليات التركيب المنطقي
#idx("metacircular evaluator for Python", sub: "syntactic forms (additional)")
#idx("metacircular evaluator for Python", sub: "&& (logical conjunction)")
#idx("metacircular evaluator for Python", sub: "|| (logical disjunction)")
#idx("&& (logical conjunction)", sub: "implementing in metacircular evaluator", sort: ";1")
#idx("&& (logical conjunction)", sub: "as derived component", sort: ";1")
#idx("|| (logical disjunction)", sub: "implementing in metacircular evaluator", sort: ";2")
#idx("|| (logical disjunction)", sub: "as derived component", sort: ";2")
#py("&&") و #py("||") هي سكر نحوي للتعبيرات الشرطية:
فالعطف المنطقي $italic("expression")_(1)$ #py("&&") $italic("expression")_(2)$ هو سكر نحوي لـ $italic("expression")_(1)$ #py("?") $italic("expression")_(2)$ #py(":") #py("false")، و
الفصل المنطقي $italic("expression")_(1)$ #py("||") $italic("expression")_(2)$ هو سكر نحوي لـ $italic("expression")_(1)$ #py("?") #py("true") #py(":") $italic("expression")_(2)$.

وتُعرب كـ
#idx("&& (logical conjunction)", sub: "parsing of", sort: ";1")
#idx("|| (logical disjunction)", sub: "parsing of", sort: ";2")
يلي:

#syntax($lt.double space$, meta("expression"), $""_(1)$, " ", meta("logical-operation"), " ", meta("expression"), $""_(2) med gt.double$, " =
    llist(\"logical_composition\",
         \"", meta("logical"), "-", meta("operation"), "\",
         llist(", $lt.double space$, meta("expression"), $""_(1) med gt.double$, ", ", $lt.double space$, meta("expression"), $""_(2) med gt.double$, "))
	  ")

حيث #meta("logical-operation") هو #py("&&") أو #py("||").
ثبت #py("&&") و #py("||") كأشكال نحوية جديدة للمُقيِّم بـ إعلان دوال نحوية ودوال تقييم مناسبة #py("eval_and") و #py("eval_or"). وبدلاً من ذلك، بين كيفية تنفيذ #py("&&") و #py("||") كمكونات مشتقة.
])

#exercise(label-name: <ex:directly>, [
+ في Python، يجب ألا تحتوي تعبيرات #en[lambda] على #idx("parameters", sub: "duplicate") #idx("duplicate parameters") #idx("metacircular evaluator for Python", sub: "preventing duplicate parameters") بارامترات مكررة. المُقيِّم في القسم @sec:core-of-evaluator لا يفحص ذلك.

  - عدل المُقيِّم بحيث تعطي أي محاولة لتطبيق دالة بـ بارامترات مكررة إشارة خطأ.
  - نفذ دالة #py("verify") تفحص ما إذا كان أي تعبير #en[lambda] في برنامج معطى يحتوي على بارامترات مكررة. ومع مثل هذه الدالة، يمكننا فحص البرنامج بأكمله قبل أن نمرره إلى #py("evaluate").

  من أجل تنفيذ هذا الفحص في مُقيِّم لـ Python، أي من هاتين المقاربتين تفضل؟ ولماذا؟
+ في Python، يجب أن تكون بارامترات تعبير #en[lambda] متميزة عن #idx("metacircular evaluator for Python", sub: "parameters distinct from local names") #idx("parameters", sub: "distinct from local names") #idx("internal declaration", sub: "names distinct from parameters") الأسماء Mُعلنة #emph[مباشرة] في كتلة الجسم لتعبير #en[lambda] (عكس الكتلة الداخلية). استخدم مقاربتك المفضلة أعلاه للفحص عن هذا أيضاً.
])

#exercise([
تتضمن لغة Scheme تنويعاً لـ
#idx("metacircular evaluator for Python", sub: "syntactic forms (additional)")
#idx("metacircular evaluator for Python", sub: "let* (Scheme variant of let)")
#idx("let* (Scheme variant of let)")
#idx("Scheme", sub: "let* in")
#py("let") يُسمى #py("let*"). وكان بإمكاننا تقريب سلوك #py("let*") في Python بـ النص على أن إعلان #py("let*") يقدم ضمناً كتلة جديدة يحتوي جسمها على الإعلان وجميع العبارات اللاحقة لتسلسل العبارات الذي يحدث فيه الإعلان. على سبيل المثال، فإن البرنامج

#snippet(```python
let* x = 3
let* y = x + 2
let* z = x + y + 5
print(x * z)
```)

يعرض 39 وكان يمكن رؤيته كاختصار لـ

#snippet(```python
{
  let x = 3
  {
    let y = x + 2
    {
      let z = x + y + 5
      print(x * z)
    }
  }
}
```)

+ اكتب برنامجاً في هذه اللغة المُوسعة لـ Python يتصرف بشكل مختلف عندما تُستبدل بعض الوقوعات للكلمة المفتاحية #py("let") بـ #py("let*").
+ قدم #py("let*") كشكل نحوي جديد بـ تصميم تمثيل قائمة مُعنونة مناسب وكتابة قاعدة إعراب. وأعلن عن دالة شرطية نحوية ومحددات لتمثيل القائمة المُعنونة.
+ بفرض أن #py("parse") تنفذ قاعدتك الجديدة، اكتب دالة #py("let_star_to_nested_let") تحول أي وقوع لـ #py("let*") في برنامج معطى كما هو موضح أعلاه. ويمكننا حينها تقييم برنامج #py("p") في اللغة المُوسعة بـ تشغيل #py("evaluate(let_star_to_nested_let(p))").
+ كبديل، فكر في تنفيذ #py("let*") بـ إضافة بند إلى #py("evaluate") يتعرف على الشكل النحوي الجديد ويستدعي دالة #py("eval_let_star_declaration"). لماذا لا تعمل هذه المقاربة؟
])

#exercise(label-name: <ex:while_loop>, [
تدعم Python
#idx("metacircular evaluator for Python", sub: "syntactic forms (additional)")
#idx("while loop", sub: "implementing in metacircular evaluator")
#idx("metacircular evaluator for Python", sub: "while loop")
#idx("syntactic forms", sub: "while loop")
#idx("while (keyword)")
#idx("keywords", sub: "while")
#emph[حلقات طالما] (#en[while loops]) التي تنفذ عبارة معطاة تكرارياً. وعلى وجه التحديد،

#syntax("
while (", meta("predicate"), ") { ", meta("body"), " }
	  ")

تُقيّم #meta("predicate")، وإذا كانت النتيجة صحيحة، تُقيّم #meta("body") ثم تُقيّم حلقة طالما بأكملها مرة أخرى. وبمجرد تقييم #meta("predicate") إلى خطأ، تنتهي حلقة طالما.

على سبيل المثال، تذكر النسخة ذات الأسلوب الأمرّي لـ دالة المضرب التكرارية من القسم @sec:costs-of-assignment:

#snippet(```python
def factorial(n):
    product = 1
    counter = 1
    def iter():
        if counter > n:
            return product
        else:
            product = counter * product
            counter = counter + 1
            return iter()
    return iter()
```)

يمكننا صياغة الخوارزم نفسه باستخدام حلقة طالما كما يلي:
#idx("factorial", sub: "with while loop", decl: true)
#snippet(```python
def factorial(n):
    product = 1
    counter = 1
    while counter <= n:
        product = counter * product
        counter = counter + 1
    return product
```)

تُعرب حلقات طالما كما يلي:

#syntax($lt.double space$, "while (", meta("predicate"), ") ", meta("block"), $space gt.double$, " =
        llist(\"while_loop\", ", $lt.double space$, meta("predicate"), $space gt.double$, ", ", $lt.double space$, meta("block"), $space gt.double$, ")
	      ")

+ أعلن عن دالة شرطية نحوية ومحددات للتعامل مع حلقات طالما.
+ أعلن عن دالة #py("while_loop") تأخذ كوسائط دالة شرطية وجسماً—كل منهما ممثل بـ دالة بلا وسائط—وتحاكي سلوك حلقة طالما. وستبدو دالة #py("factorial") حينها كما يلي: #snippet(```python def factorial(n): product = 1 counter = 1 def pred(): return counter <= n def body(): nonlocal product, counter product = counter * product counter = counter + 1 while_loop(pred, body) return product ```) ينبغي لدالتك #py("while_loop") توليد عملية تكرارية (انظر القسم @sec:recursion-and-iteration).
+ ثبت حلقات طالما كمكون مشتق بـ تعريف دالة تحويل #py("while_to_application") تستخدم دالتك #py("while_loop").
+ ما المشكلة التي تنشأ مع هذه المقاربة لتنفيذ حلقات طالما، عندما يقرر المبرمج داخل جسم الحلقة الإرجاع من الدالة التي تحتوي على الحلقة؟
+ غير مقاربتك لمعالجة المشكلة. ماذا عن تثبيت حلقات طالما مباشرة للمُقيِّم، باستخدام دالة #py("eval_while")؟
+ باتباع هذه المقاربة المباشرة، نفذ عبارة #idx("syntactic forms", sub: "break statement") #idx("break (keyword)") #idx("keywords", sub: "break") #py("break;") تنهي فوراً الحلقة التي تُقيَّم فيها.
+ نفذ عبارة #idx("syntactic forms", sub: "continue statement") #idx("continue (keyword)", sort: "continue") #idx("keywords", sub: "continue") #strong[#raw("continue")]#py(";") تنهي فقط تكرار الحلقة الذي تُقيَّم فيه، وتستمر بتقييم الدالة الشرطية لحلقة طالما.
])

#exercise(label-name: <ex:value_producing>, [
تتحدد نتيجة تقييم جسم دالة بـ عبارات الإرجاع الخاصة بها.
ومتابعةً للحاشية @foot:value_producing_2 وتقييم الإعلانات في القسم @sec:core-of-evaluator، يتناول هذا التمرين سؤال ما الذي يجب أن يكون نتيجة
#idx("value", sub: "of a program")
#idx("program", sub: "value of")
#idx("statement", sub: "value-producing and non-value-producing")
#idx("metacircular evaluator for Python", sub: "value of program at top level")
تقييم برنامج Python يتكون من تسلسل عبارات (إعلانات، كتل، عبارات تعبير، وعبارات شرطية) #emph[خارج] أي جسم دالة.

بالنسبة لمثل هذا البرنامج، تميز Python استاتيكياً بين #emph[العبارات المُنتجة للقيمة] و #emph[العبارات غير المُنتجة للقيمة]. (هنا "استاتيكياً" تعني أنه يمكننا إجراء التمييز بـ #emph[فحص] البرنامج بدلاً من تشغيله.)
جميع الإعلانات هي غير منتجة للقيمة، وجميع عبارات التعبير والعبارات الشرطية هي منتجة للقيمة.
وقيمة عبارة التعبير هي قيمة التعبير.
وقيمة العبارة الشرطية هي قيمة الفرع الذي يُنفَّذ، أو القيمة #py("None") إذا كان هذا الفرع غير منتج للقيمة.
والكتلة منتجة للقيمة إذا كان جسمها (تسلسل العبارات) منتجاً للقيمة، وحينها تكون قيمتها هي قيمة جسمها.
والتسلسل منتج للقيمة إذا كانت أي من عبارات مكوناته منتجة للقيمة، وحينها تكون قيمتها هي قيمة عبارة مكوناته المُنتجة للقيمة #emph[الأخيرة].
وأخيراً، إذا لم يكن البرنامج بأكمله منتجاً للقيمة، فإن قيمته هي القيمة #py("None").

+ وفقاً لهذا التحديد، ما هي قيم البرامج الأربعة التالية؟ #snippet(```python 1; 2; 3 1; { if (true) {} else { 2; } } 1; const x = 2 1; { let x = 2; { x = x + 3; } } ```)
+ عدل المُقيِّم للالتزام بهذا التحديد.
])
