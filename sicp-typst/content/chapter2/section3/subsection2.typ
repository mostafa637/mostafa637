// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Example: Symbolic Differentiation], label-name: <sec:symbolic-differentiation>)

#idx("differentiation", sub: "symbolic")
#idx("symbolic differentiation")
#idx("algebraic expression", sub: "differentiating")

As an illustration of symbol manipulation and a further illustration
of data abstraction, consider the design of a
function
that performs symbolic differentiation of algebraic expressions. We would
like the
function
to take as arguments an algebraic expression and a variable and to return
the derivative of the expression with respect to the variable. For example,
if the arguments to the
function
are $a x^(2) + b x +c$
and $x$, the
function
should return $2a x+b$. Symbolic differentiation
is of special historical significance in
the programming language Lisp.#footnote[The original version of this book used the programming language Scheme, a dialect of Lisp.]
It was one of the
motivating examples behind the development of a computer language for
symbol manipulation. Furthermore, it marked the beginning of the line of
research that led to the development of powerful systems for symbolic
mathematical work, which are
today routinely used by applied mathematicians and physicists.

In developing the symbolic-differentiation program, we will follow the same
strategy of data abstraction that we followed in developing the
rational-number system of section @sec:rationals. That
is, we will first define a differentiation algorithm that operates on
abstract objects such as "sums," "products," and
"variables" without worrying about how these are to be
represented. Only afterward will we address the representation problem.

#subheading([The differentiation program with abstract data])

To
keep things simple, we will consider a very simple
symbolic-differentiation program that handles expressions that are built up
using only the operations of addition and multiplication with two
arguments. Differentiation of any such expression can be carried out by
applying the following
#idx("differentiation", sub: "rules for")
reduction rules:

$ mat(delim: #none, frac(d c, d x), =, 0 upright("for ca constant or a variable different from x"); frac(d x, d x), =, 1; frac(d(u+v), d x), =, frac(d u, d x)+frac(d v, d x); frac(d(u v), d x), =, u l r(( frac(d v, d x) ))+v l r(( frac(d u, d x) ))) $

Observe that the latter two rules are recursive in nature. That is, to
obtain the derivative of a sum we first find the derivatives of the terms
and add them. Each of the terms may in turn be an expression that needs to
be decomposed. Decomposing into smaller and smaller pieces will eventually
produce pieces that are either constants or variables, whose derivatives
will be either $0$ or
$1$.

To embody these rules in a
function
we indulge in a little
#idx("wishful thinking")
wishful thinking, as we did in designing the rational-number implementation.
If we had a means for representing algebraic expressions, we should be able
to tell whether an expression is a sum, a product, a constant, or a
variable. We should be able to extract the parts of an expression. For a
sum, for example, we want to be able to extract the addend (first term) and
the augend (second term). We should also be able to construct expressions
from parts. Let us assume that we already have
functions
to implement the following selectors, constructors, and predicates:

#sicp-table(columns: 2, [#py("is_variable(e)")], [Is #py("e") a variable?], [#py("is_same_variable(v1, v2)")], [Are #py("v1") and #py("v2") the same variable?], [#py("is_sum(e)")], [Is #py("e") a sum?], [#py("addend(e)")], [Addend of the sum #py("e").], [#py("augend(e)")], [Augend of the sum #py("e").], [#py("make_sum(a1, a2)")], [Construct the sum of #py("a1") and #py("a2").], [#py("is_product(e)")], [Is #py("e") a product?], [#py("multiplier(e)")], [Multiplier of the product #py("e").], [#py("multiplicand(e)")], [Multiplicand of the product #py("e").], [#py("make_product(m1, m2)")], [Construct the product of #py("m1") and #py("m2").])

Using these, and the primitive predicate
#idx("isnumber (primitive function)")

#py("is_number"),
which identifies numbers, we can express the differentiation rules as the
following
function:
#idx("deriv (symbolic)", decl: true)
#snippet(```python
def deriv(exp, variable):
    return (0
            if is_number(exp)
            else (1 if is_same_variable(exp, variable) else 0)
            if is_variable(exp)
            else make_sum(deriv(addend(exp), variable),
                          deriv(augend(exp), variable))
            if is_sum(exp)
            else make_sum(make_product(multiplier(exp),
                                       deriv(multiplicand(exp),
                                             variable)),
                  make_product(deriv(multiplier(exp),
                                             variable),
                                       multiplicand(exp)))
            if is_product(exp)
            else error("unknown expression type -- deriv", exp))
```)

This #py("deriv")
function
incorporates the complete differentiation algorithm. Since it is expressed
in terms of abstract data, it will work no matter how we choose to
represent algebraic expressions, as long as we design a proper set of
selectors and constructors. This is the issue we must address next.

#subheading([Representing algebraic expressions])

#idx("algebraic expression", sub: "representing")

We can imagine many ways to use
linked list
structure to represent algebraic
expressions. For example, we could use
linked lists
of symbols that mirror the
usual algebraic notation, representing $a x+b$ as
#py("llist(\"a\", \"*\", \"x\", \"+\", \"b\")").
However, it will be more convenient if we reflect the mathematical structure of the expression in the Python value representing it; that is, to represent $a x+b$ as #py("llist(\"+\", llist(\"*\", \"a\", \"x\"), \"b\")"). Placing a binary operator in front of its operands is called #idx("prefix notation") #emph[prefix notation], in contrast with the infix notation introduced in section @sec:expressions. With prefix notation, our
data representation for the differentiation problem is as follows:

- The variables are just strings. They are identified by the primitive predicate #idx("isstring (primitive function)") #py("is_string"): #idx("isvariable", sub: "for algebraic expressions", decl: true) #snippet(```python def is_variable(x): return is_string(x) ```)
- Two variables are the same if the strings representing them are equal: #idx("issamevariable", decl: true) #snippet(```python def is_same_variable(v1, v2): return is_variable(v1) and is_variable(v2) and v1 == v2 ```)
- Sums and products are constructed as linked lists: #idx("makesum", decl: true)#idx("makeproduct", decl: true) #snippet(```python def make_sum(a1, a2): return llist("+", a1, a2) def make_product(m1, m2): return llist("*", m1, m2) ```)
- A sum is a linked list whose first element is the string #py("\"+\""): #idx("issum", decl: true) #snippet(```python def is_sum(x): return is_pair(x) and head(x) == "+" ```)
- The addend is the second item of the sum linked list: #idx("addend", decl: true) #snippet(```python def addend(s): return head(tail(s)) ```)
- The augend is the third item of the sum linked list: #idx("augend", decl: true) #snippet(```python def augend(s): return head(tail(tail(s))) ```)
- A product is a linked list whose first element is the string #py("\"*\""): #idx("isproduct", decl: true) #snippet(```python def is_product(x): return is_pair(x) and head(x) == "*" ```)
- The multiplier is the second item of the product linked list: #idx("multiplier", sub: "selector", decl: true) #snippet(```python def multiplier(s): return head(tail(s)) ```)
- The multiplicand is the third item of the product linked list: #idx("multiplicand", decl: true) #snippet(```python def multiplicand(s): return head(tail(tail(s))) ```)

Thus, we need only combine these with the algorithm as embodied by
#py("deriv") in order to have a working
symbolic-differentiation program. Let us look at some examples of its
behavior:

#snippet(```python
print_llist(deriv(llist("+", "x", 3), "x"))
```)

#output(```python
print_llist(deriv(llist("+", "x", 3), "x"))
```)

#snippet(```python
print_llist(deriv(llist("*", "x", "y"), "x"))
```)

#output(```python
print_llist(deriv(llist("*", "x", "y"), "x"))
```)

#snippet(```python
print_llist(deriv(llist("*", llist("*", "x", "y"),
                             llist("+", "x", 3)), "x"))
```)

#output(```python
print_llist(deriv(llist("*", llist("*", "x", "y"),
                             llist("+", "x", 3)), "x"))
```)

The program produces answers that are correct; however, they are
unsimplified. It is true that

$ mat(delim: #none, frac(d(x y), d x), =, x dot.op 0+1 dot.op y) $

but we would like the program to know that
$x dot.op 0 = 0$,
$1 dot.op y = y$, and
$0+y = y$. The answer for the second example
should have been simply #py("y"). As the
third example shows, this becomes a serious issue when the expressions are
complex.

Our difficulty is much like the one we encountered with the rational-number
implementation:
#idx("algebraic expression", sub: "simplifying")
#idx("simplification of algebraic expressions")
we haven't reduced answers to simplest form. To
accomplish the rational-number reduction, we needed to change only the
constructors and the selectors of the implementation. We can adopt a similar strategy here. We won't change #py("deriv") at
all. Instead, we will change
#py("make_sum")
so that if both summands are numbers,
#py("make_sum")
will add them and return their sum. Also, if one of the summands is 0,
then
#py("make_sum")
will return the other summand.

#idx("makesum", decl: true)
#snippet(```python
def make_sum(a1, a2):
    return (a2
            if number_equal(a1, 0)
            else a1
            if number_equal(a2, 0)
            else a1 + a2
            if is_number(a1) and is_number(a2)
            else llist("+", a1, a2))
```)

This uses the
function
#py("number_equal"),
which checks whether an expression is equal to a given number:

#idx("numberequal", decl: true)
#snippet(```python
def number_equal(exp, num):
    return is_number(exp) and exp == num
```)

Similarly, we will change
#py("make_product")
to build in the rules that 0 times anything is 0 and 1 times anything is
the thing itself:

#idx("makeproduct", decl: true)
#snippet(```python
def make_product(m1, m2):
    return (0
            if number_equal(m1, 0) or number_equal(m2, 0)
            else m2
            if number_equal(m1, 1)
            else m1
            if number_equal(m2, 1)
            else m1 * m2
            if is_number(m1) and is_number(m2)
            else llist("*", m1, m2))
```)

Here is how this version works on our three examples:

#snippet(```python
print(deriv(llist("+", "x", 3), "x"))
```)

#output(```python
print(deriv(llist("+", "x", 3), "x"))
```)

#snippet(```python
print(deriv(llist("*", "x", "y"), "x"))
```)

#output(```python
print(deriv(llist("*", "x", "y"), "x"))
```)

#snippet(```python
print_llist(deriv(llist("*", llist("*", "x", "y"),
                             llist("+", "x", 3)), "x"))
```)

#output(```python
print_llist(deriv(llist("*", llist("*", "x", "y"),
                             llist("+", "x", 3)), "x"))
```)

Although this is quite an improvement, the third example shows that there
is still a long way to go before we get a program that puts expressions
into a form that we might agree is "simplest." The problem
of algebraic simplification is complex because, among other reasons, a
form that may be simplest for one purpose may not be for another.
#idx("algebraic expression", sub: "simplifying")

#exercise(label-name: <ex:deriv-exponentiation>, [
Show how to extend the basic differentiator to handle more kinds of
expressions.
#idx("differentiation", sub: "rules for")
For instance, implement the differentiation rule

$ mat(delim: #none, frac(d(u^(n)), d x), =, nu^(n-1)lr(( frac(d u, d x) ))) $

by adding a new clause to the #py("deriv") program
and defining appropriate
functions
#py("is_exp"),
#py("base"), #py("exponent"),
and
#py("make_exp").
(You may use
the string #py("\"**\"")
to denote exponentiation.) Build in the rules that anything raised to the
power 0 is 1 and anything raised to the power 1 is the thing itself.
])

#exercise(label-name: <ex:2_57>, [
Extend the differentiation program to handle sums and products of arbitrary
numbers of (two or more) terms. Then the last example above could be
expressed as

#snippet(```python
deriv(llist("*", "x", "y", llist("+", "x", 3)), "x")
```)

Try to do this by changing only the
representation for sums and products, without changing the
#py("deriv")
function
at all. For example, the #py("addend") of a sum would
be the first term, and the #py("augend") would be the
sum of the rest of the terms.
])

#exercise(label-name: <ex:2_58>, [
Suppose we want to modify the differentiation program so that it works
with ordinary mathematical notation, in which
#py("\"+\"")
and
#py("\"*\"")
are
#idx("infix notation", sub: "prefix notation vs.")
#idx("prefix notation", sub: "infix notation vs.")
infix rather than prefix operators. Since the differentiation program
is defined in terms of abstract data, we can modify it to work with
different representations of expressions solely by changing the predicates,
selectors, and constructors that define the representation of the algebraic
expressions on which the differentiator is to operate.

+ Show how to do this in order to differentiate algebraic expressions presented in infix form, as in this example: #snippet(```python llist("x", "+", llist(3, "*", llist("x", "+", llist("y", "+", 2)))) ```) To simplify the task, assume that #py("\"+\"") and #py("\"*\"") always take two arguments and that expressions are fully parenthesized.
+ The problem becomes substantially harder if we allow a notation closer to ordinary infix notation, which omits unnecessary parentheses and assumes that multiplication has higher precedence than addition, as in this example: #snippet(```python llist("x", "+", 3, "*", llist("x", "+", "y", "+", 2)) ```) Can you design appropriate predicates, selectors, and constructors for this notation such that our derivative program still works?
])

#idx("algebraic expression", sub: "representing")
#idx("differentiation", sub: "symbolic")
#idx("symbolic differentiation")
#idx("algebraic expression", sub: "differentiating")
