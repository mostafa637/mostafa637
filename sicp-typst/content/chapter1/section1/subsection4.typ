// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Compound Functions], label-name: <sec:compound-procedures>)

We have identified in
Python
some of the elements that must appear in any powerful programming language:

- Numbers and arithmetic operations are primitive data and functions.
- Nesting of combinations provides a means of combining operations.
- Declaration assignments that associate names with values provide a limited means of abstraction.

Now we will learn about
#idx("def", sub: "function definition")
#emph[function definitions],
a much more powerful abstraction technique by which a compound
operation can be given a name and then referred to as a unit.

We begin by examining how to express the idea of
"squaring."
We might say,
"To square something, take it times itself."

This is expressed in our language as
#idx("square", decl: true)#idx("def (keyword)")#idx("keywords", sub: "def")
#snippet(```python
def square(x): return x * x
```)

We can understand this in the following way:

$ mat(delim: #none, mono(bold("def")), mono("square("), mono("x"), mono(")" mono(":")), mono(bold("return")), mono("x"), mono("*"), mono("x"), ; arrow.t, arrow.t, arrow.t, , , arrow.t, arrow.t, arrow.t, ; "To", "square", "something,", , "take", "it", "times", "itself.", ) $

We have here a
#idx("compound function") #emph[compound function],
which has been given the name #py("square"). The
function
represents the operation of multiplying something by itself. The thing to
be multiplied is given a local name, #py("x"),
which plays the same role that a pronoun plays in natural language.
#idx("naming", sub: "of functions") #idx("function", sub: "naming (with def)") #idx("function", sub: "creating with def") #idx("function definition") #idx("definition", sub: "of function")
Evaluating the
definition
creates this compound
function
and associates it with the name
#idx("syntactic forms", sub: "function definition")
#idx("def")
#idx("function definition")
#idx("declaration", sub: "of function (def)")
#py("square").#footnote[Observe that there are two
different operations being combined here: we are creating the
function,
and we are giving
it the name #py("square"). It is possible, indeed
important, to be able to separate these two notions—to create
functions
without naming them, and to give names to
functions
that have already been created. We will see how to do this in
section @sec:lambda.]

The simplest form of a function definition
is

#syntax("
def ", meta("name"), "(", meta("parameters"), "): return ", meta("expression"))

The
#idx("name", sub: "of a function") #meta("name")
is a symbol to be associated with the
function
definition in the environment.#footnote[Throughout this book, we will
#idx("notation in this book", sub: "italic symbols in expression syntax")
#idx("syntax", sub: "of expressions, describing")
describe the general syntax of expressions by using italic
symbols—e.g.,
#meta("name")—to
denote the "slots" in the expression to be filled in
when such an expression is actually used.]
The
#idx("parameters")
#meta("parameters")
are the names used within the body of the
function
to refer to the
corresponding arguments of the
function.

The word #py("def") is a #emph[keyword] in Python. Keywords carry a particular meaning, and
thus cannot be used as names. A keyword in a program component instructs the Python
interpreter to treat the component as a syntactic form
with its own evaluation rule.
The #meta("parameters")
are grouped within
#idx("parentheses", sub: "in function definition")
#idx("parentheses", sub: "in function definition")
parentheses and separated by commas, as they will be in an application
of the function	being defined.
#idx("return statement")
#idx("return value")
#idx("return (keyword)")
#idx("syntactic forms", sub: "return statement")
#idx("keywords", sub: "return")
In the simplest form, the

#idx("body of a function")
#emph[body] of a function definition is a single
#emph[return statement],#footnote[More
#idx("sequence of statements", sub: "in function body")
generally, the body of the function can be a sequence of statements.
In this case, the interpreter evaluates each statement in the sequence
in turn until a return statement determines the value of the
function application.]
which consists of the keyword
#py("return")
followed by the #emph[return expression]
that will yield the value of the function application, when the

parameters are replaced by the actual arguments to which the function
is applied.
#idx("def", sub: "function definition")

Having defined #py("square"),
we can now use it in a
#emph[function application] expression:

#snippet(```python
square(21)
```)

Function applications are—after operator
combinations—the second kind of combination of
expressions into larger expressions that we encounter.
The general form of a function application is

#syntax(meta("function-expression"), "(", meta("argument-expressions"), ")
          ")

where the
#idx("function expression")
#meta("function-expression")
of the application specifies
the function to be applied to the comma-separated
#idx("argument(s)")
#meta("argument-expressions").
To evaluate a function application, the interpreter follows
#idx("evaluation", sub: "of function application")
#idx("function application", sub: "evaluation of")
a procedure
quite similar to the procedure for operator combinations described in
section @sec:evaluating-combinations.

- To evaluate a function application, do the following:

  + Evaluate the subexpressions of the application, namely the function expression and the argument expressions.
  + Apply the function that is the value of the function expression to the values of the argument expressions.

The same procedure applies to applications of the primitive function
#py("print") that we encountered already in
section @sec:expressions.

#snippet(```python
print(square(2 + 5))
```)

#output(```python
print(square(2 + 5))
```)

Here, the argument expression of #py("print")
is a compound expression, the application expression
#py("square(2 + 5)"), whose argument expression
is itself a compound expression,
the operator combination #py("2 + 5").

#snippet(```python
print(square(square(3)))
```)

#output(```python
print(square(square(3)))
```)

Of course function application expressions can be further nested.

We can also use #py("square")
as a building block in defining other
functions.
For example, $x^(2) +y^(2)$ can be expressed as

#snippet(```python
square(x) + square(y)
```)

We can easily
define
a
function #py("sum_of_squares")
that, given any two numbers as arguments, produces the
sum of their squares:
#idx("sumofsquares", decl: true)
#snippet(```python
def sum_of_squares(x, y):
    return square(x) + square(y)
```)

For readability, we can start a new line after the colon,
in which case the function body needs to be indented.#footnote[The
number of characters used in the indentation is flexible but needs
to be consistent throughout the function body. In this book, we mostly
use indentation by four characters.]

#snippet(```python
print(sum_of_squares(3, 4))
```)

#output(```python
print(sum_of_squares(3, 4))
```)

Now we can use
#py("sum_of_squares")
as a building block in constructing further
functions:

#snippet(```python
def f(a):
    return sum_of_squares(a + 1, a * 2)
```)

#snippet(```python
print(f(5))
```)

#output(```python
print(f(5))
```)

In addition to compound functions, any Python environment provides

#idx("primitive function")
primitive functions that are built into the interpreter or loaded
from libraries.
#idx("Python environment used in this book")
Besides the primitive functions provided by the operators
and the primitive function #py("print"),
the Python environment used in this book includes
additional primitive functions
such as the function
#idx("mathlog (primitive function)")

#py("math_log"),
which computes the natural logarithm of its argument.
These additional primitive functions are used in exactly the same way as
#idx("compound function", sub: "used like primitive function")
compound functions; evaluating the application
#py("print(math_log(1))") results in the display of the number 0.0.
Indeed, one could not tell by looking at the definition of
#py("sum_of_squares") given above whether
#py("square") was built into the
interpreter, loaded from a library, or defined as a compound function.
