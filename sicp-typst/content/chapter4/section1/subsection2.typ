// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Representing Components], label-name: <sec:representing-expressions>)

#idx("metacircular evaluator for Python", sub: "component representation")
#idx("metacircular evaluator for Python", sub: "syntax of evaluated language")
#idx("parsing Python")

Programmers write programs as text, i.e. sequences of characters, entered
in a programming environment or a text editor. To run our evaluator, we need
to start with a representation of this program text as a Python value.
In section @sec:strings we introduced strings to represent
text. We would like to evaluate programs such as
#py("\"size = 2\\n5 * size\"")
from section @sec:naming.
Unfortunately, such program text does not provide enough structure to
the evaluator. In this example, the program parts
#py("\"size = 2\"") and
#py("\"5 * size\"") look similar, but carry
very different meanings. Abstract syntax functions such as
#py("declaration_value_expression") would be
difficult
and error-prone to implement by examining the program text.
In this section, we therefore
introduce a function
#idx("parse")
#py("parse") that translates program text
to a #emph[tagged-list representation], reminiscent of
the tagged data of section @sec:manifest-types.
For example, the application of
#py("parse") to the program string
above produces a data structure that
reflects the structure of the program: a sequence consisting
of a constant declaration associating the name
#py("size") with the value 2
and a multiplication.

#snippet(```python
parse("size = 2\n5 * size")
```)

#output(```python
parse("size = 2\n5 * size")
```)

The syntax functions used by the evaluator access the tagged-list
representation produced by
#py("parse").

The evaluator is reminiscent of the
#idx("metacircular evaluator for Python", sub: "symbolic differentiation and")
symbolic differentiation program
discussed in section @sec:symbolic-differentiation.
Both programs operate on symbolic
data.
In both programs, the
result of operating on
an object
is determined by
operating recursively on the pieces of the
object
and combining
the results in a way that depends on the type of the
object.
In both programs we used
#idx("data abstraction")
data abstraction to decouple the general rules
of operation from the details of how
the objects
are represented. In
the differentiation program this meant that the same differentiation
function
could deal with algebraic expressions in prefix form, in
infix form, or in some other form. For the evaluator, this means that
the syntax of the language being evaluated is determined solely by #py("parse") and the functions that classify and extract pieces of the tagged lists produced by #py("parse").

#sicp-figure(image("/images/img_javascript/ch4-parse-abstraction.svg", width: 70%), caption: [Syntax abstraction in the evaluator.], label-name: <fig:parse-abstraction>)

Figure @fig:parse-abstraction depicts the
#idx("abstraction barriers", sub: "in representing Python syntax")
abstraction barrier
formed by the syntax predicates and selectors,
which interface the evaluator to the tagged-list representation of programs,
which in turn is separated from the string representation by
#py("parse"). Below we
describe the parsing of program components and list the
corresponding syntax predicates and selectors, as well as
constructors if they are needed.

#idx("parsing Python")

#subheading([Literal expression])

Literal expressions

#idx("literal expression", sub: "parsing of")
are parsed into tagged lists with
tag #py("\"literal\"") and
the actual value.

$ mat(delim: #none, lt.double space italic("literal")-italic("expression") space gt.double, =, mono("list(\"literal\", ")italic("value")mono(")")) $

where #meta("value") is
the Python value represented by the
#meta("literal-expression") string.
Here $lt.double space italic("literal")-italic("expression") space gt.double$ denotes the
result of parsing the string #meta("literal-expression").

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

The syntax predicate for literal expressions is
#py("is_literal").
#idx("isliteral", decl: true)
#snippet(```python
def is_literal(component):
    return is_tagged_list(component, "literal")
```)

It is defined in terms of the function
#py("is_tagged_list"),
which identifies lists that begin with a designated
string:
#idx("istaggedlist", decl: true)
#snippet(```python
def is_tagged_list(component, the_tag):
    return is_pair(component) and head(component) == the_tag
```)

The second element of the list that results from parsing a literal expression
is its actual Python value.
The selector for retrieving the value is
#py("literal_value").
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

In the rest of this section, we just list the syntax predicates and selectors,
and omit their declarations if they just access the obvious list elements.

We provide a constructor for literals, which will come in handy:
#idx("makeliteral", decl: true)
#snippet(```python
def make_literal(value):
    return llist("literal", value)
```)

#subheading([Names])

The tagged-list representation for

#idx("name", sub: "parsing of")
#idx("symbol(s)", sub: "in parsing of names")
names includes the tag #py("\"name\"") as first
element and the string representing the name as second element.

$ mat(delim: #none, lt.double space italic("name") space gt.double, =, mono("list(\"name\", ")italic("symbol")mono(")")) $

where #meta("symbol") is a string
that contains the characters that make up the
#meta("name") as written in the program.

The syntax predicate for names is
#idx("isname")
#py("is_name").

The symbol is accessed using the selector
#idx("symbolofname")
#py("symbol_of_name").

We provide a constructor for names, to be used by
#py("operator_combination_to_application"):
#idx("makename", decl: true)
#snippet(```python
def make_name(symbol):
    return llist("name", symbol)
```)

#subheading([Expression statements])

We do not need to distinguish between expressions and

#idx("expression statement", sub: "parsing of")
expression statements.
Consequently,
#py("parse") can ignore the difference
between the two kinds of components:

$ mat(delim: #none, lt.double space italic("expression")mono(";") space gt.double, =, lt.double space italic("expression") space gt.double) $

#subheading([Function applications])

Function applications

#idx("function application", sub: "parsing of")
are parsed as follows:

#syntax($lt.double space$, meta("fun-expr"), "(", meta("arg-expr"), $""_(1)$, ", ", $dots.h$, ", ", meta("arg-expr"), $""_(n)$, ")", $space gt.double$, " =
     llist(\"application\",
          ", $lt.double space$, meta("fun-expr"), $space gt.double$, ",
          llist(", $lt.double space$, meta("arg-expr"), $""_(1) med gt.double$, ", ", $dots.h$, ", ", $lt.double space$, meta("arg-expr"), $""_(n) med gt.double$, "))
	  ")

We declare
#idx("isapplication")
#py("is_application")
as the syntax predicate and
#idx("functionexpression")
#idx("argexpressions")
#py("function_expression") and
#py("arg_expressions") as the selectors.

We add a constructor for function applications, to be used by
#py("operator_combination_to_application"):
#idx("makeapplication", decl: true)
#snippet(```python
def make_application(function_expression, argument_expressions):
    return llist("application", function_expression, argument_expressions)
```)

#subheading([Conditionals])

Conditional expressions

#idx("conditional expression", sub: "parsing of")

#idx("conditional statement", sub: "parsing of")
are parsed as follows:

#syntax($lt.double space$, meta("predicate"), " ? ", meta("consequent-expression"), " : ", meta("alternative-expression"), $space gt.double$, " =
        llist(\"conditional_expression\",
             ", $lt.double space$, meta("predicate"), $space gt.double$, ",
             ", $lt.double space$, meta("consequent-expression"), $space gt.double$, ",
             ", $lt.double space$, meta("alternative-expression"), $space gt.double$, ")
	  ")

Similarly, conditional statements are parsed as follows:

#syntax($lt.double space$, "if (", meta("predicate"), ") ", meta("consequent-block"), " else ", meta("alternative-block"), $space gt.double$, " =
        llist(\"conditional_statement\",
             ", $lt.double space$, meta("predicate"), $space gt.double$, ",
             ", $lt.double space$, meta("consequent-block"), $space gt.double$, ",
             ", $lt.double space$, meta("alternative-block"), $space gt.double$, ")
	  ")

The syntax predicate
#idx("isconditional")
#py("is_conditional")
returns true for both kinds of conditionals, and the selectors
#idx("conditionalpredicate")
#py("conditional_predicate"),
#idx("conditionalconsequent")
#py("conditional_consequent"), and
#idx("conditionalalternative")
#py("conditional_alternative")
can be applied to both kinds.

#subheading([Lambda expressions])

A lambda expression

#idx("lambda expression", sub: "parsing of")
whose body is an expression is parsed as if the
body consisted of a block containing a single return statement whose
return expression is the body of the lambda expression.

#syntax($lt.double space$, "(", meta("name"), $""_(1)$, ", ", $dots.h$, ", ", meta("name"), $""_(n)$, ") => ", meta("expression"), $space gt.double$, " =
    ", $lt.double space$, "(", meta("name"), $""_(1)$, ", ", $dots.h$, ", ", meta("name"), $""_(n)$, ") => { return ", meta("expression"), "; }", $space gt.double$)

A lambda expression whose body is a block is parsed as follows:

#syntax($lt.double space$, "(", meta("name"), $""_(1)$, ", ", $dots.h$, ", ", meta("name"), $""_(n)$, ") => ", meta("block"), $space gt.double$, " =
     llist(\"lambda_expression\",
          llist(", $lt.double space$, meta("name"), $""_(1) med gt.double$, ", ", $dots.h$, ", ", $lt.double space$, meta("name"), $""_(n) med gt.double$, "),
          ", $lt.double space$, meta("block"), $space gt.double$, ")
	  ")

The syntax predicate is
#idx("islambdaexpression")
#py("is_lambda_expression")
and the selector for the body of the lambda expression is
#idx("lambdabody")
#py("lambda_body").
The selector for the parameters, called
#py("lambda_parameter_symbols"),
additionally extracts the symbols from the names.
#idx("lambdaparametersymbols", decl: true)
#snippet(```python
def lambda_parameter_symbols(component):
    return map(symbol_of_name, head(tail(component)))
```)

The function
#py("function_decl_to_constant_decl")
needs a constructor for lambda expressions:
#idx("makelambdaexpression", decl: true)
#snippet(```python
def make_lambda_expression(parameters, body):
    return llist("lambda_expression", parameters, body)
```)

#subheading([Sequences])

A sequence statement

#idx("sequence of statements", sub: "parsing of")
packages a sequence of statements into a
single statement. A sequence of statements is parsed as follows:

#syntax($lt.double space$, meta("statement"), $""_(1)$, " ", $dots.c$, " ", meta("statement"), $""_(n) med gt.double$, " =
     llist(\"sequence\", llist(", $lt.double space$, meta("statement"), $""_(1) med gt.double$, ", ", $dots.h$, ", ", $lt.double space$, meta("statement"), $""_(n) med gt.double$, "))
	  ")

The syntax predicate is
#idx("issequence")
#py("is_sequence") and
the selector is #py("sequence_statements").
We retrieve the first of a list of statements using
#py("first_statement") and
the remaining statements using
#py("rest_statements"). We test
whether the list is empty using the predicate
#py("is_empty_sequence") and
whether it contains only one element
using the predicate
#py("is_last_statement").#footnote[These
selectors for a list of statements are not intended
as a data abstraction.
They are introduced as mnemonic names for the
basic list operations in order to make it easier to understand the
explicit-control evaluator in
section @sec:eceval.]<foot:mceval-abstraction>
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

#subheading([Blocks])

Blocks

#idx("block", sub: "parsing of")
are parsed as follows:#footnote[A parser implementation may
decide to represent a block by just its statement sequence if none
of the statements of the sequence are declarations, or to represent
a sequence with only one statement by just that statement. The language
processors in this chapter and in chapter 5 do not depend on
these decisions.]

$ mat(delim: #none, lt.double space mono(\{) space italic("statements") space mono(\}) space gt.double, =, mono("list(\"block\",") space lt.double space italic("statements") space gt.double mono(")")) $

Here #meta("statements") refers to a sequence of
statements, as shown above.
The syntax predicate is
#idx("isblock")
#py("is_block")
and the selector is
#idx("blockbody")
#py("block_body").

#subheading([Return statements])

Return statements

#idx("return statement", sub: "parsing of")
are parsed as follows:

$ mat(delim: #none, lt.double space bold(mono("return")) space italic("expression") mono(";") space gt.double, =, mono("list(\"return_statement\",") space lt.double space italic("expression") space gt.double mono(")")) $

The syntax predicate and selector are, respectively,
#idx("isreturnstatement")
#py("is_return_statement")
and
#idx("returnexpression")
#py("return_expression").

#subheading([Assignments])

Assignments

#idx("assignment", sub: "parsing of")
are parsed as follows:

$ mat(delim: #none, lt.double med italic("name") space mono("=") space italic("expression") med gt.double, =, mono("list(\"assignment\",") space lt.double med italic("name") med gt.double mono(", ") lt.double med italic("expression") med gt.double mono(")")) $

The syntax predicate is
#idx("isassignment")
#py("is_assignment")
and the selectors are
#py("assignment_symbol")
and
#idx("assignmentvalueexpression")
#py("assignment_value_expression").
The symbol is wrapped in a tagged list representing the name, and thus
#py("assignment_symbol") needs to
unwrap it.
#idx("assignmentsymbol", decl: true)
#snippet(```python
def assignment_symbol(component):
    return head(tail(head(tail(component))))
```)

#subheading([Assignments and function definitions])

assignments

#idx("assignment", sub: "parsing of")

#idx("variable", sub: "assignment, parsing of")
are parsed as follows:

#syntax($lt.double space$, meta("name"), "$ = $", meta("expression"), $space gt.double$, " =
     llist(\"assignment\", ", $lt.double space$, meta("name"), $space gt.double$, ", ", $lt.double space$, meta("expression"), $space gt.double$, ")
	  ")

The selectors
#py("assignment_symbol") and
#py("assignment_value_expression") apply to both
kinds.
#idx("assignmentsymbol", decl: true)#idx("assignmentvalueexpression", decl: true)
#snippet(```python
def assignment_symbol(component):
    return symbol_of_name(head(tail(component)))
def assignment_value_expression(component):
    return head(tail(tail(component)))
```)

The function
#py("function_def_to_assignment")
needs a constructor for constant declarations:
#idx("makeassignment", decl: true)
#snippet(```python
def make_assignment(name, value_expression):
    return llist("assignment", name, value_expression)
```)

Function definitions

#idx("function definition", sub: "parsing of")
are parsed as follows:

#syntax($lt.double space$, "function ", meta("name"), "(", meta("name"), $""_(1)$, ", ", $dots.h$, " ", meta("name"), $""_(n)$, ") ", meta("block"), $space gt.double$, " =
    llist(\"function_definition\",
         ", $lt.double space$, meta("name"), $space gt.double$, ",
         llist(", $lt.double space$, meta("name"), $""_(1) med gt.double$, ", ", $dots.h$, ", ", $lt.double space$, meta("name"), $""_(n) med gt.double$, "),
         ", $lt.double space$, meta("block"), $space gt.double$, ")
	  ")

The syntax predicate
#idx("isfunctiondefinition")
#py("is_function_definition")
recognizes these.
The selectors are
#idx("functiondefinitionname")
#py("function_definition_name"),
#idx("functiondefinitionparameters")
#py("function_definition_parameters"), and
#idx("functiondefinitionbody")
#py("function_definition_body").

The syntax predicate
#py("is_declaration")
returns true for all three kinds of declarations.
#idx("isdeclaration", decl: true)
#snippet(```python
def is_declaration(component):
    return is_tagged_list(component, "constant_declaration") or is_tagged_list(component, "variable_declaration") or is_tagged_list(component, "function_definition")
```)

The selectors
#py("declaration_symbol") and
#py("declaration_value_expression") apply to all
three kinds.
#idx("declarationsymbol", decl: true)#idx("declarationvalueexpression", decl: true)
#snippet(```python
def declaration_symbol(component):
    return symbol_of_name(head(tail(component)))
def declaration_value_expression(component):
    return head(tail(tail(component)))
```)

The function
#py("function_decl_to_constant_decl")
needs a constructor for constant declarations:
#idx("makeconstantdeclaration", decl: true)
#snippet(```python
def make_constant_declaration(name, value_expression):
    return llist("constant_declaration", name, value_expression)
```)

#subheading([Derived components])

#idx("derived components in evaluator")
#idx("syntactic form", sub: "as derived component")
#idx("metacircular evaluator for Python", sub: "derived components")
#idx("metacircular evaluator for Python", sub: "syntactic forms as derived components")

Some
#idx("syntactic form", sub: "as derived component")
syntactic forms in our language can be defined in terms of
components involving other syntactic forms, rather than being
implemented directly.
One example is
#idx("function definition", sub: "as derived component")
#idx("derived components in evaluator", sub: "function definition")
function definition, which
#py("evaluate")
transforms into a constant declaration whose
value expression is a lambda expression.#footnote[In
actual Python, there are subtle differences between the two
forms; see footnote #text(fill: red)[?]
in chapter 1.
Exercise @ex:hoisting addresses these differences.]
#idx("functiondecltoconstantdecl", decl: true)
#snippet(```python
def function_decl_to_constant_decl(component):
    return make_constant_declaration( function_definition_name(component), make_lambda_expression( function_definition_parameters(component), function_definition_body(component)))
```)

Implementing the evaluation of function definitions
in this	way simplifies the evaluator because it reduces the number of
syntactic forms for which the evaluation process must be explicitly specified.

Similarly, we define

#idx("operator combination", sub: "parsing of")
operator combinations in terms of
function applications.
Operator combinations are unary or binary and carry their operator symbol as second element
in the tagged-list representation:

#syntax($lt.double space$, meta("unary-operator"), " ", meta("expression"), $space gt.double$, " =
     llist(\"unary_operator_combination\",
          \"", meta("unary-operator"), "\",
          llist(", $lt.double space$, meta("expression"), $space gt.double$, "))
	  ")

where #meta("unary-operator") is
#py("!") (for logical negation) or
#py("-unary") (for numeric negation), and

#syntax($lt.double space$, meta("expression"), $""_(1)$, " ", meta("binary-operator"), " ", meta("expression"), $""_(2) med gt.double$, " =
     llist(\"binary_operator_combination\",
          \"", meta("binary-operator"), "\",
          llist(", $lt.double space$, meta("expression"), $""_(1) med gt.double$, ", ", $lt.double space$, meta("expression"), $""_(2) med gt.double$, "))
	  ")

where #meta("binary-operator") is
#py("+"),
#py("-"),
#py("*"),
#py("/"),
#py("%"),
#py("==="),
#py("!=="),
#py(">"),
#py("<"),
#py(">=") or
#py("<=").
The syntax predicates are
#py("is_operator_combination"),
#py("is_unary_operator_combination"), and
#py("is_binary_operator_combination"),
and the selectors are
#py("operator_symbol"),
#py("first_operand"), and
#py("second_operand").

The evaluator uses
#py("operator_combination_to_application")
to transform an
#idx("operator combination", sub: "as function application")
#idx("operator combination", sub: "as derived component")
#idx("derived components in evaluator", sub: "operator combination")
operator combination into a function application whose
function expression is the name of the operator:
#idx("operatorcombinationtoapplication", decl: true)
#snippet(```python
def operator_combination_to_application(component):
    operator = operator_symbol(component)
    return make_application(make_name(operator), llist(first_operand(component))) if is_unary_operator_combination(component) else make_application(make_name(operator), llist(first_operand(component), second_operand(component)))
```)

Components (such as function definitions and operator combinations) that we
choose
to implement as syntactic transformations are called
#idx("derived component")
#emph[derived components]. Logical composition operations are also
derived components (see exercise @ex:eval-and-or).

#idx("metacircular evaluator for Python", sub: "component representation")
#idx("metacircular evaluator for Python", sub: "syntax of evaluated language")
#idx("metacircular evaluator for Python", sub: "derived components")
#idx("metacircular evaluator for Python", sub: "syntactic forms as derived components")
#idx("derived components in evaluator")
#idx("syntactic form", sub: "as derived component")

#exercise(label-name: <ex:parse>, [
The inverse of #py("parse")
is called
#idx("unparse", sub: "as inverse of parse")
#py("unparse"). It takes
as argument a
tagged list as produced by #py("parse")
and returns a string that adheres to Python notation.

+ Write a function #py("unparse") by following the structure of #py("evaluate") (without the environment parameter), but producing a string that represents the given component, rather than evaluating it. Recall from section @sec:circuit-simulator that the operator #py("+") can be applied to two strings to concatenate them and that the primitive function #py("stringify") turns values such as 1.5, true, #py("None") and #py("None") into strings. Take care to respect operator precedences by surrounding the strings that result from unparsing operator combinations with parentheses (always or whenever necessary).
+ Your #py("unparse") function will come in handy when solving later exercises in this section. Improve #py("unparse") by adding #py("\"") #py("\"") (space) and #py("\"\\n\"") (newline) characters to the result string, to follow the #idx("indentation") indentation style used in the Python programs of this book. Adding such #idx("whitespace characters") whitespace characters to (or removing them from) a program text in order to make the text easier to read is called #idx("pretty-printing") #emph[pretty-printing].
])

#exercise(label-name: <ex:data-directed-eval>, [
Rewrite
#py("evaluate") so that the
dispatch is done in
#idx("data-directed programming", sub: "in metacircular evaluator")
#idx("metacircular evaluator for Python", sub: "data-directed evaluate")
#idx("evaluate (metacircular)", sub: "data-directed")
data-directed style. Compare this with the
data-directed differentiation function of
exercise @ex:data-directed-differentiation. (You may
use the tag of the tagged-list representation as the type of the components.)
])

#exercise(label-name: <ex:eval-and-or>, [
Recall from section @sec:conditionals that the
#idx("metacircular evaluator for Python", sub: "syntactic forms (additional)")
#idx("metacircular evaluator for Python", sub: "&& (logical conjunction)")
#idx("metacircular evaluator for Python", sub: "|| (logical disjunction)")
#idx("&& (logical conjunction)", sub: "implementing in metacircular evaluator", sort: ";1")
#idx("&& (logical conjunction)", sub: "as derived component", sort: ";1")
#idx("|| (logical disjunction)", sub: "implementing in metacircular evaluator", sort: ";2")
#idx("|| (logical disjunction)", sub: "as derived component", sort: ";2")
logical composition operations
#py("&&")
and
#py("||")
are syntactic sugar for conditional expressions:
The logical conjunction
$italic("expression")_(1)$ #py("&&")
$italic("expression")_(2)$
is syntactic sugar for
$italic("expression")_(1)$ #py("?")
$italic("expression")_(2)$ #py(":")
#py("false"), and
the logical disjunction
$italic("expression")_(1)$
#py("||")
$italic("expression")_(2)$
is syntactic sugar for
$italic("expression")_(1)$ #py("?")
#py("true") #py(":")
$italic("expression")_(2)$.

They are
#idx("&& (logical conjunction)", sub: "parsing of", sort: ";1")

#idx("|| (logical disjunction)", sub: "parsing of", sort: ";2")
parsed as follows:

#syntax($lt.double space$, meta("expression"), $""_(1)$, " ", meta("logical-operation"), " ", meta("expression"), $""_(2) med gt.double$, " =
    llist(\"logical_composition\",
         \"", meta("logical"), "-", meta("operation"), "\",
         llist(", $lt.double space$, meta("expression"), $""_(1) med gt.double$, ", ", $lt.double space$, meta("expression"), $""_(2) med gt.double$, "))
	  ")

where #meta("logical-operation") is
#py("&&") or
#py("||").
Install
#py("&&") and
#py("||")
as new syntactic forms for the evaluator
by declaring appropriate syntax functions and evaluation functions
#py("eval_and") and
#py("eval_or"). Alternatively, show how to
implement
#py("&&") and
#py("||")
as derived components.
])

#exercise(label-name: <ex:directly>, [
+ In Python, lambda expressions must not have #idx("parameters", sub: "duplicate") #idx("duplicate parameters") #idx("metacircular evaluator for Python", sub: "preventing duplicate parameters") duplicate parameters. The evaluator in section @sec:core-of-evaluator does not check for this.

  - Modify the evaluator so that any attempt to apply a function with duplicate parameters signals an error.
  - Implement a #py("verify") function that checks whether any lambda expression in a given program contains duplicate parameters. With such a function, we could check the entire program before we pass it to #py("evaluate").

  In order to implement this check in an evaluator for Python, which of these two approaches would you prefer? Why?
+ In Python, the parameters of a lambda expression must be distinct from #idx("metacircular evaluator for Python", sub: "parameters distinct from local names") #idx("parameters", sub: "distinct from local names") #idx("internal declaration", sub: "names distinct from parameters") the names declared #emph[directly] in the body block of the lambda expression (as opposed to in an inner block). Use your preferred approach above to check for this as well.
])

#exercise([
The language Scheme includes a variant of
#idx("metacircular evaluator for Python", sub: "syntactic forms (additional)")
#idx("metacircular evaluator for Python", sub: "let* (Scheme variant of let)")
#idx("let* (Scheme variant of let)")
#idx("Scheme", sub: "let* in")
#py("let") called
#py("let*"). We could approximate
the behavior of
#py("let*") in Python by stipulating
that a
#py("let*") declaration implicitly
introduces a new block whose body includes the declaration and all
subsequent statements of the statement sequence in which the
declaration occurs. For example, the program

#snippet(```python
let* x = 3
let* y = x + 2
let* z = x + y + 5
print(x * z)
```)

displays 39 and could be seen as a shorthand for

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

+ Write a program in such an extended Python language that behaves differently when some occurrences of the keyword #py("let") are replaced with #py("let*").
+ Introduce #py("let*") as a new syntactic form by designing a suitable tagged-list representation and writing a parse rule. Declare a syntax predicate and selectors for the tagged-list representation.
+ Assuming that #py("parse") implements your new rule, write a #py("let_star_to_nested_let") function that transforms any occurrence of #py("let*") in a given program as described above. We could then evaluate a program #py("p") in the extended language by running #py("evaluate(let_star_to_nested_let(p))").
+ As an alternative, consider implementing #py("let*") by adding to #py("evaluate") a clause that recognizes the new syntactic form and calls a function #py("eval_let_star_declaration"). Why does this approach not work?
])

#exercise(label-name: <ex:while_loop>, [
Python supports
#idx("metacircular evaluator for Python", sub: "syntactic forms (additional)")
#idx("while loop", sub: "implementing in metacircular evaluator")
#idx("metacircular evaluator for Python", sub: "while loop")
#idx("syntactic forms", sub: "while loop")
#idx("while (keyword)")
#idx("keywords", sub: "while")
#emph[while loops] that execute a given
statement repeatedly. Specifically,

#syntax("
while (", meta("predicate"), ") { ", meta("body"), " }
	  ")

evaluates the #meta("predicate"), and
if the result is true, evaluates the
#meta("body")
and then evaluates
the whole while loop again.
Once the #meta("predicate") evaluates
to false, the
while loop terminates.

For example, recall the imperative-style version of the iterative
factorial function from section @sec:costs-of-assignment:

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

We can formulate the same algorithm using a while loop as follows:
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

While loops are parsed as follows:

#syntax($lt.double space$, "while (", meta("predicate"), ") ", meta("block"), $space gt.double$, " =
        llist(\"while_loop\", ", $lt.double space$, meta("predicate"), $space gt.double$, ", ", $lt.double space$, meta("block"), $space gt.double$, ")
	      ")

+ Declare a syntax predicate and selectors to handle while loops.
+ Declare a function #py("while_loop") that takes as arguments a predicate and a body—each represented by a function of no arguments—and simulates the behavior of the while loop. The #py("factorial") function would then look as follows: #snippet(```python def factorial(n): product = 1 counter = 1 def pred(): return counter <= n def body(): nonlocal product, counter product = counter * product counter = counter + 1 while_loop(pred, body) return product ```) Your function #py("while_loop") should generate an iterative process (see section @sec:recursion-and-iteration).
+ Install while loops as a derived component by defining a transformation function #py("while_to_application") that makes use of your function #py("while_loop").
+ What problem arises with this approach for implementing while loops, when the programmer decides within the body of the loop to return from the function that contains the loop?
+ Change your approach to address the problem. How about directly installing while loops for the evaluator, using a function #py("eval_while")?
+ Following this direct approach, implement a #idx("syntactic forms", sub: "break statement") #idx("break (keyword)") #idx("keywords", sub: "break") #py("break;") statement that immediately terminates the loop in which it is evaluated.
+ Implement a #idx("syntactic forms", sub: "continue statement") #idx("continue (keyword)", sort: "continue") #idx("keywords", sub: "continue") #strong[#raw("continue")]#py(";") statement that terminates only the loop iteration in which it is evaluated, and continues with evaluating the while loop predicate.
])

#exercise(label-name: <ex:value_producing>, [
The result of evaluating the body of a function is determined by
its return statements.
Following up on footnote @foot:value_producing_2
and the evaluation of declarations in
section @sec:core-of-evaluator,
this exercise addresses the question of what should be the result of
#idx("value", sub: "of a program")
#idx("program", sub: "value of")
#idx("statement", sub: "value-producing and non-value-producing")
#idx("metacircular evaluator for Python", sub: "value of program at top level")
evaluating a Python program that consists of a sequence of
statements (declarations, blocks, expression statements, and conditional
statements) #emph[outside of] any function body.

For such a program, Python
statically
distinguishes between #emph[value-producing] and
#emph[non-value-producing statements]. (Here
"statically" means that
we can make the distinction by #emph[inspecting] the program
rather than by running it.)
All declarations are
non-value-producing, and all
expression statements and conditional statements are
value-producing.
The value of an expression statement is the value of the expression.
The value of a conditional statement is the value of the branch that
gets executed, or the value
#py("None") if that branch is
not value-producing.
A block is value-producing if its body (sequence of statements)
is value-producing, and then its value is the value of its body.
A sequence is value-producing if any of
its component statements is value-producing, and then its value is
the value of its #emph[last] value-producing component statement.
Finally, if the whole
program	is not value-producing, its value is the value
#py("None").

+ According to this specification, what are the values of the following four programs? #snippet(```python 1; 2; 3 1; { if (true) {} else { 2; } } 1; const x = 2 1; { let x = 2; { x = x + 3; } } ```)
+ Modify the evaluator to adhere to this specification.
])
