// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Expressions], label-name: <sec:expressions>)

One easy way to get started at programming is to examine some typical
interactions with an interpreter for the
Python language.

You type
a #emph[statement],
and pass it to the interpreter, which then
#emph[evaluates] that
#idx("statement") statement.

One kind of statement you might type is an
#emph[expression].
#idx("number(s)", sub: "in Python")
#idx("primitive expression", sub: "number")
#idx("expression")
One kind of primitive expression is a number.

(More precisely, the expression that you type consists of the numerals that
represent the number in base 10.)

If you present
Python with the program

#snippet(```python
486
```)

the interpreter will respond by printing—nothing.
To see the result of evaluating #py("486"),
we need to apply the primitive function #py("print")
to it, using the usual mathematical notation of function application

#snippet(```python
print(486)
```)

resulting in#footnote[Throughout this book,
we distinguish
#idx("notation in this book", sub: "slanted characters for interpreter response")
between the input typed by
the user and any text printed by the interpreter by showing the
latter in slanted characters.]

#output(```python
print(486)
```)

Expressions representing numbers may be combined with
operators
(such
#idx("+", sub: "as numeric addition operator")

as #py("+")
#idx("arithmetic", sub: "operators for")
#idx("* (multiplication operator)", sort: "-1")

or #py("*")) to form a
#idx("compound expression")
#idx("operator combination")
compound expression that represents the
application of a corresponding primitive
function to those numbers. For example,

#snippet(```python
print(137 + 349)
```)

#output(```python
print(137 + 349)
```)

#snippet(```python
print(1000 - 334)
```)

#output(```python
print(1000 - 334)
```)

#snippet(```python
print(5 * 99)
```)

#output(```python
print(5 * 99)
```)

#idx("/ (division operator)")
#snippet(```python
print(10 / 4)
```)

#output(```python
print(10 / 4)
```)

#snippet(```python
print(2.7 + 10)
```)

#output(```python
print(2.7 + 10)
```)

Expressions such as #py("137 + 349"), which contain other expressions
as components, are called #emph[combinations].
#idx("combination")
Combinations that are formed by an
#idx("operator of a combination")
#idx("operator combination")
#emph[operator] symbol in the middle, and
#idx("operands of a combination")
#emph[operand] expressions to the left and right of it,
are called
#emph[operator combinations].
#idx("value", sub: "of an expression")
The value of an operator combination is
obtained by applying the function specified by the operator to the
arguments that are the values of the operands.

The convention of placing the operator between the operands is
known as
#idx("infix operator")
#idx("infix notation")
#emph[infix notation]. It follows the mathematical notation that
you are most likely familiar with from school and everyday life.
As in mathematics, operator combinations can be #emph[nested], that
is, they can have operands that
#idx("nested operator combinations")
themselves are operator combinations:

#snippet(```python
print((3 * 5) + (10 - 6))
```)

#output(```python
print((3 * 5) + (10 - 6))
```)

As usual,
#idx("parentheses", sub: "to group operator combinations")
parentheses are used to group operator combinations in order
to avoid ambiguities. Python also follows the usual conventions
when parentheses are omitted: multiplication and division bind more
strongly than addition and subtraction. For example,

#snippet(```python
3 * 5 + 10 / 2
```)

stands for

#snippet(```python
(3 * 5) + (10 / 2)
```)

We say that #py("*") and
#py("/") have
#idx("precedence", sub: "of operators")
#emph[higher precedence]
than #py("+") and
#py("-"). Sequences of additions and
subtractions are read from left to right, as are sequences of
multiplications and divisions. Thus,
#idx("-", sub: "as numeric subtraction operator")
#snippet(```python
1 - 5 / 2 * 4 + 3
```)

stands for

#snippet(```python
(1 - ((5 / 2) * 4)) + 3
```)

We say that the operators
#py("+"),
#py("-"),
#py("*") and
#py("/") are
#idx("associativity", sub: "of operators")
#idx("left-associative")
#emph[left-associative].

There is no limit (in principle) to the depth of such nesting and to the
overall complexity of the expressions that the Python interpreter
can evaluate. It is we humans who might get confused by still relatively
simple expressions such as

#snippet(```python
3 * (2 * 4 + (3 + 5)) + ((10 - 7) + 6)
```)

which the interpreter would readily evaluate to be 57. We can help
ourselves by writing such an expression in the form

#snippet(```python
(3 * (2 * 4 + (3 + 5))
 +
 ((10 - 7) + 6))
```)

to visually separate the major components of the expression.#footnote[The additional parentheses are necessary,
if we want to spread an expression over multiple lines and there
are no surrounding parentheses already.]

Even with complex expressions, the interpreter always operates in the
same basic cycle: It reads
a statement typed by the user,
evaluates the
statement,
and prints the result of any applications of #py("print").
This mode of operation is often expressed by saying
that the interpreter runs in a
#idx("read-evaluate-print loop") #idx("interpreter", sub: "read-evaluate-print loop") #emph[read-evaluate-print loop].
Observe, however, that it is necessary to explicitly instruct the interpreter to print the value of the expression.
