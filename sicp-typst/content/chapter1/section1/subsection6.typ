// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Conditional Expressions and Predicates], label-name: <sec:conditionals>)

The expressive power of the class of
functions
that we can define at this point is very limited, because we have no way to
make tests and to perform different operations depending on the result of a
test.

For instance, we cannot define a function that computes the
#idx("absolute value")
absolute value of a number by testing whether the number is nonnegative
and taking different actions in each case according to the rule

$ mat(delim: #none, |x|, =, cases(x quad upright("if x gt.eq 0"), -x quad upright("otherwise"))) . $

This construct is a
#idx("case analysis")
#emph[case analysis] and can be written
in Python using a #emph[conditional expression] as
#idx("abs", decl: true)
#snippet(```python
def abs(x):
    return x if x >= 0 else - x
```)

which could be expressed in English as "Return $x$, if $x$ is greater than or equal to zero; otherwise return $- x$."
The general form of a conditional expression is

#syntax(meta("consequent-expression"), " if ", meta("predicate"), " else ", meta("alternative-expression"))

The keyword #py("if") in conditional
#idx("conditional expression")
#idx("syntactic forms", sub: "conditional expression")
#idx("if else expression")
#idx("predicate", sub: "of conditional expression")
#idx("True (keyword)", sort: "true")
#idx("False (keyword)", sort: "false")
#idx("keywords", sub: "True")
#idx("keywords", sub: "False")
#idx("expression", sub: "boolean")
#idx("boolean values (True, False)")
expressions is followed by a
#meta("predicate")—that is,
an expression whose value is interpreted as either true or false.#footnote["Interpreted as either #idx("true") #idx("false") #idx("#t", decl: true) #idx("true", decl: true) #idx("#f", decl: true) #idx("false", decl: true) true or false"
means this:
In Python, there are two distinguished values that are
denoted by the constants
#py("True") and
#py("False").
When the interpreter checks a predicate's value, it interprets
#py("False") as false and
#py("True") as true. Python considers
any value to be either true or false, but in this book we only
use these two.]
The keyword #py("if")
is preceded by the
#meta("consequent-expression"),
and followed by the #py("else") keyword
and finally by the #meta("alternative-expression").

To
#idx("evaluation", sub: "of conditional expression")
#idx("conditional expression", sub: "evaluation of")
evaluate a conditional expression,
the interpreter starts by evaluating the
#meta("predicate")
of the expression. If the
#meta("predicate")
evaluates to true, the interpreter evaluates the
#idx("consequent", sub: "of conditional expression")
#meta("consequent-expression") and returns its value as the value of the conditional.
If the #meta("predicate")
evaluates to false, it evaluates the
#idx("alternative", sub: "of conditional expression")
#meta("alternative-expression") and returns its value as the value of the
conditional.#footnote[#idx("conditional expression", sub: "non-boolean value as predicate")
Conditionals

in full Python accept any value, not just the boolean values 1 and 0, as the result of evaluating
the #meta("predicate") expression (see footnote @foot:truthy
in section @sec:eval-data-structures for details). The programs in this book
use only boolean values as predicates of conditionals.]<foot:any-value-as-predicate>

The word
#idx("predicate")
#emph[predicate] is used for operators and functions that
return true or false, as well as for expressions that
evaluate to true or false. The absolute-value function
#py("abs") makes use of the
#idx(">= (numeric comparison operator)")

#idx("number(s)", sub: "comparison of")
#idx("number(s)", sub: "equality of")
primitive predicate #py(">="),
an operator that takes two numbers as arguments and tests whether the
first number is greater than or equal to the second number, returning
true or false accordingly.

If we prefer to handle the zero case separately, we can specify the function
that computes the absolute value of a number by writing

$ mat(delim: #none, |x|, =, cases(x quad upright("if x > 0"), 0 quad upright("if x = 0"), -x quad upright("otherwise"))) . $

In Python, we express a case analysis with multiple cases by nesting
conditional expressions as alternative expressions inside other conditional expressions:
#idx("abs", decl: true)
#snippet(```python
def abs(x):
    return x if x > 0 else 0 if x == 0 else - x
```)

Parentheses are not needed around the alternative expression
#py("0 if x == 0 else - x"), because
the conditional-expression syntactic form
#idx("conditional expression", sub: "as alternative of conditional expression")
#idx("associativity", sub: "of conditional expression")
#idx("conditional expression", sub: "right-associativity of")
#idx("right-associative")
is right-associative.
The general form of a
#idx("case analysis", sub: "general")
case analysis is

#syntax(meta("e"), $""_(1)$, " if ", meta("p"), $""_(1)$, " else ", meta("e"), $""_(2)$, " if ", meta("p"), $""_(2)$, " ", $dots.c$, " ", meta("e"), $""_(n)$, " if ", meta("p"), $""_(n)$, " else ", meta("final-alternative-expression"))

We call a
consequent expression
$e_(i)$
together with its
predicate $p_(i)$
a
#idx("clause of a case analysis")
#idx("predicate", sub: "of clause")
#emph[clause]. A case analysis
can be seen as a sequence of clauses, followed by a final
alternative expression.
#idx("case analysis", sub: "as sequence of clauses")
According to the evaluation of conditional expressions,
a case analysis is evaluated by first evaluating
the predicate #meta("p")$""_(1)$.
If its value is false, then #meta("p")$""_(2)$
is evaluated.
If #meta("p")$""_(2)$'s
value is also false, then #meta("p")$""_(3)$
is evaluated. This process continues until a predicate is
found whose value is true, in which case the interpreter returns the
value of the corresponding
#idx("consequent", sub: "of clause")
consequent expression
#meta("e")
of the clause
as the value of the case analysis.
If none of the
#meta("p")'s
is found to be true, the value of the case analysis
is the value of the final alternative expression.

In addition to primitive predicates such as
#idx("> (numeric comparison operator)")
#idx("< (numeric comparison operator)", sort: ">=0")
#idx("<= (numeric comparison operator)", sort: ">=1")
#idx("==", sub: "as numeric equality operator")
#idx("!=", sub: "as numeric comparison operator", sort: ";4")

#idx("equality", sub: "of numbers")
#py(">="),
#py(">"),
#py("<"),
#py("<="),
#py("=="), and
#py("!=") that are applied to
numbers,#footnote[For now, we restrict these operators to number
arguments. In sections @sec:strings
and @sec:mutable-list-structure, we shall
generalize the equality and inequality predicates
#py("==") and
#py("!=").]
there are logical composition operations, which enable us to construct
compound predicates. The three most frequently used are these:

- #meta("expression")$""_(1)$ #py("and") #meta("expression")$""_(2)$\ This operation expresses #idx("syntactic sugar", sub: "and and || as") #idx("and (logical conjunction)") #idx("and (logical conjunction)", sub: "evaluation of") #idx("syntactic forms", sub: "logical conjunction (and)") #idx("logical conjunction") #idx("conjunction") #idx("evaluation", sub: "of and") #emph[logical conjunction], meaning roughly the same as the English word "and." We assume#footnote[This assumption is justified by the restriction mentioned in footnote @foot:any-value-as-predicate. Full Python needs to consider the case where the result of evaluating #meta("expression")$""_(1)$ is neither true nor false.] this syntactic form to be syntactic sugar#footnote[Syntactic forms that are simply convenient alternative surface structures for things that can be written in more uniform ways are sometimes called #emph[syntactic sugar], to use a phrase coined by #idx("Landin, Peter") #idx("syntactic sugar") Peter Landin.] for\ #meta("expression")$""_(2)$ #py("if") #meta("expression")$""_(1)$ #py("else") #py("False").
- #meta("expression")$""_(1)$ #py("or") #meta("expression")$""_(2)$\ This operation expresses #idx("or (logical disjunction)") #idx("or (logical disjunction)", sub: "evaluation of") #idx("syntactic forms", sub: "logical disjunction (or)") #idx("logical disjunction") #idx("disjunction") #idx("evaluation", sub: "of or") #emph[logical disjunction], meaning roughly the same as the English word "or." We assume this syntactic form to be syntactic sugar for\ #py("True") #py("if") #meta("expression")$""_(1)$ #py("else") #meta("expression")$""_(2)$.
- #py("not") #meta("expression")\ This operation expresses #idx("not (logical negation operator)") #idx("negation", sub: "logical (not)") #emph[logical negation], meaning roughly the same as the English word "not." The value of the expression is true when #meta("expression") evaluates to false, and false when #meta("expression") evaluates to true.

Notice that #py("and") and
#py("or") are syntactic forms,
not operators;
#idx("and (logical conjunction)", sub: "why a syntactic form")
#idx("or (logical disjunction)", sub: "why a syntactic form")
their right-hand
expression is not always evaluated. The operator
#py("not"), on the other hand,
follows the evaluation rule of section
@sec:evaluating-combinations.
It is a #emph[unary] operator, which means that it takes only
one argument, whereas the arithmetic operators and primitive predicates
discussed so far
are #emph[binary], taking two arguments. The operator
#py("not") precedes its argument;
we call it a
#idx("-", sub: "as numeric negation operator")

#idx("negation", sub: "numeric (-)")
#idx("binary operator")
#idx("unary operator")
#idx("prefix operator")
#emph[prefix operator]. Another prefix operator is
the numeric negation operator, an example of
which is the expression #py("- x")
in the #py("abs") functions above.

As an example of how these predicates are used, the condition that a
number $x$ be in the range
$5 < x < 10$ may be expressed as

#snippet(```python
x > 5 and x < 10
```)

The syntactic form
#py("and")
has lower precedence than the comparison operators
#py(">")
and #py("<"), and
the conditional-expression syntactic form
$dots.c$#py("if")$dots.c$#py("else")$dots.c$
has lower precedence than any other operator we have encountered so far,
a property we used in
the #py("abs") functions above.

As another example, we can define
a predicate to test whether one number is

greater than or equal to another as

#snippet(```python
def greater_or_equal(x, y):
    return x > y or x == y
```)

or alternatively as

#snippet(```python
def greater_or_equal(x, y):
    return not (x < y)
```)

The function #py("greater_or_equal"),
when applied to two numbers, behaves the same as the operator
#py(">="). Unary operators have
#idx("precedence", sub: "of unary operators")
higher precedence than binary operators, which makes the
parentheses in this example necessary.

#exercise(label-name: <ex:1_1>, [
Below is a sequence of
statements.
What is the result printed by the interpreter in response to each
statement?
Assume that the sequence is to be evaluated in the order
in which it is presented.

#snippet(```python
print(10)
```)

#snippet(```python
print(5 + 3 + 4)
```)

#snippet(```python
print(9 - 1)
```)

#snippet(```python
print(6 / 2)
```)

#snippet(```python
print(2 * 4 + (4 - 6))
```)

#snippet(```python
a = 3
```)

#snippet(```python
b = a + 1
```)

#snippet(```python
print(a + b + a * b)
```)

#snippet(```python
print(a == b)
```)

#snippet(```python
print(b if b > a and b < a * b else a)
```)

#snippet(```python
print(6 if a == 4 else 6 + 7 + a if b == 4 else 25)
```)

#snippet(```python
print(2 + (b if b < a else a))
```)

#snippet(```python
print((a if a > b else b if a < b else -1) * (a + 1))
```)

The parentheses around the conditional expressions in the last two statements
are necessary because the
#idx("conditional expression", sub: "as operand of operator combination")
conditional-expression syntactic form has lower
#idx("precedence", sub: "of conditional expression")
#idx("conditional expression", sub: "precedence of")
precedence than the arithmetic operators
#py("+") and
#py("*").
])

#exercise(label-name: <ex:1_2>, [
Translate the following expression into
Python

$frac(5+4+lr(( 2-lr(( 3-(6+frac(4, 5)) )) )), 3 (6-2) (2-7))$
])

#exercise(label-name: <ex:1_3>, [
Define a function
that takes three numbers as arguments and returns
the sum of the squares of the two larger numbers.
])

#exercise(label-name: <ex:a-plus-abs-b>, [
Observe that our model of evaluation allows for
#idx("function application", sub: "compound expression as function expression of")
#idx("compound expression", sub: "as function expression of application")
#idx("function expression", sub: "compound expression as")
applications
whose function expressions are compound expressions. Use this observation
to describe the behavior of #py("a_plus_abs_b"):

#snippet(```python
def plus(a, b): return a + b

def minus(a, b): return a - b

def a_plus_abs_b(a, b):
    return (plus if b >= 0 else minus)(a, b)
```)
])

#exercise(label-name: <ex:normal-order-vs-appl-order-test>, [
Ben Bitdiddle has invented a test to determine whether the interpreter
he is faced with is using
#idx("normal-order evaluation", sub: "applicative order vs.")
#idx("applicative-order evaluation", sub: "normal order vs.")
applicative-order evaluation or normal-order
evaluation. He
defines the following two functions:

#snippet(```python
def p(): return p()

def test(x, y):
    return 0 if x == 0 else y
```)

Then he evaluates the
statement

#snippet(```python
test(0, p())
```)

What behavior will Ben observe with an interpreter that uses
applicative-order evaluation? What behavior will he observe with an
interpreter that uses normal-order evaluation? Explain your answer.
#idx("normal-order evaluation", sub: "of conditional expressions")
#idx("conditional expression", sub: "normal-order evaluation of")
(Assume that the evaluation rule for
conditional expressions
is the same whether the interpreter is using normal or applicative order:
The predicate expression is evaluated first, and the result determines
whether to evaluate the consequent or the alternative expression.)
])
