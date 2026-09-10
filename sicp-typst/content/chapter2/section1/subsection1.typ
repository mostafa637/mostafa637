// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Example: Arithmetic Operations for Rational Numbers], label-name: <sec:rationals>)

Suppose we want to do
#idx("arithmetic", sub: "on rational numbers")
#idx("rational number(s)", sub: "arithmetic operations on")
#idx("rational-number arithmetic")
arithmetic with rational numbers. We want to be
able to add, subtract, multiply, and divide them and to test whether
two rational numbers are equal.

Let us begin by assuming that we already have a way of constructing a
rational number from a numerator and a denominator. We also assume
that, given a rational number, we have a way of extracting (or
selecting) its numerator and its denominator. Let us further assume
that the constructor and selectors are available as
functions:

- #py("make_rat(")$n$#py(",")$d$#py(")") #idx("makerat") returns the rational number whose numerator is the integer $n$ and whose denominator is the integer $d$.
- #py("numer(")$x$#py(")") #idx("numer") returns the numerator of the rational number $x$.
- #py("denom(")$x$#py(")") #idx("denom") returns the denominator of the rational number $x$.

We are using here a powerful strategy of synthesis:
#idx("wishful thinking")
#emph[wishful thinking]. We haven't yet said how a rational number
is represented, or how the
functions
#py("numer"), #py("denom"), and
#py("make_rat")
should be implemented. Even so, if we did have these three
functions,
we could then add, subtract, multiply, divide, and test equality by using
the following relations:

$ mat(delim: #none, frac(n_(1), d_(1))+frac(n_(2), d_(2)), =, frac(n_(1)d_(2)+n_(2)d_(1), d_(1)d_(2)); frac(n_(1), d_(1))-frac(n_(2), d_(2)), =, frac(n_(1)d_(2)-n_(2)d_(1), d_(1)d_(2)); frac(n_(1), d_(1)) dot.op frac(n_(2), d_(2)), =, frac(n_(1)n_(2), d_(1)d_(2)); frac(n_(1)/d_(1), n_(2)/d_(2)), =, frac(n_(1)d_(2), d_(1)n_(2)); frac(n_(1), d_(1)), =, frac(n_(2), d_(2)) space upright("if and only if") space space space n_(1)d_(2) space = space n_(2)d_(1)) $

We can express these rules as
functions:
#idx("addrat", decl: true)#idx("subrat", decl: true)#idx("mulrat", decl: true)#idx("divrat", decl: true)#idx("equalrat", decl: true)
#snippet(```python
def add_rat(x, y):
    return make_rat(numer(x) * denom(y) + numer(y) * denom(x),
                    denom(x) * denom(y))
def sub_rat(x, y):
    return make_rat(numer(x) * denom(y) - numer(y) * denom(x),
                    denom(x) * denom(y))
def mul_rat(x, y):
    return make_rat(numer(x) * numer(y),
                    denom(x) * denom(y))
def div_rat(x, y):
    return make_rat(numer(x) * denom(y),
                    denom(x) * numer(y))
def equal_rat(x, y):
    return numer(x) * denom(y) == numer(y) * denom(x)
```)

Now we have the operations on rational numbers defined in terms of the
selector and constructor
functions
#py("numer"), #py("denom"), and
#py("make_rat").
But we haven't yet defined these. What we need is some way to glue
together a numerator and a denominator to form a rational number.

#subheading([Pairs])

To enable us to implement the concrete level of our data abstraction, our
Python environment
provides a compound structure called a
#idx("pair(s)")
#emph[pair], which can be constructed with the
primitive function

#idx("pair (primitive function)")
#py("pair").
This
function
takes two arguments and returns a compound data object that contains the
two arguments as parts. Given a pair, we can extract the parts using the
primitive
functions
#idx("head (primitive function)")

#py("head")
and
#idx("tail (primitive function)")

#py("tail").
Thus, we can use
#py("pair"),
#py("head"),
and
#py("tail")
as follows:

#snippet(```python
x = pair(1, 2)
```)

#snippet(```python
print(head(x))
```)

#output(```python
print(head(x))
```)

#snippet(```python
print(tail(x))
```)

#output(```python
print(tail(x))
```)

Notice that a pair is a data object that can be given a name and
manipulated, just like a primitive data object. Moreover,
#py("pair")
can be used to form pairs whose elements are pairs, and so on:

#snippet(```python
x = pair(1, 2)

y = pair(3, 4)

z = pair(x, y)
```)

#snippet(```python
print(head(head(z)))
```)

#output(```python
print(head(head(z)))
```)

#snippet(```python
print(head(tail(z)))
```)

#output(```python
print(head(tail(z)))
```)

In section @sec:hierarchical-data we will see how this
ability to combine pairs means that pairs can be used as general-purpose
building blocks to create all sorts of complex data structures. The single
compound-data primitive #emph[pair], implemented by the
functions
#py("pair"),
#py("head"),
and
#py("tail"),
is the only glue we need. Data objects constructed from pairs are called
#idx("list structure")
#idx("data", sub: "list-structured")
#emph[list-structured] data.

#subheading([Representing rational numbers])

Pairs offer a natural way to complete the
#idx("rational number(s)", sub: "represented as pairs")
rational-number system.
Simply represent a rational number as a pair of two integers: a numerator
and a denominator. Then
#py("make_rat"),
#py("numer"), and #py("denom")
are readily implemented as follows:#footnote[Another way to define the selectors and constructor is

#snippet(```python
make_rat = pair
numer = head
denom = tail
```)

The first definition associates the name
#py("make_rat")
with the value of the expression
#py("pair"),
which is the primitive
function
that constructs pairs. Thus
#py("make_rat")
and
#py("pair")
are names for the same primitive constructor.

Defining selectors and constructors in this way is efficient: Instead of
#py("make_rat")
#emph[calling]
#py("pair"),
#py("make_rat")
#emph[is]
#py("pair"),
so there is only one
function
called, not two, when
#py("make_rat")
is called. On the other hand, doing this defeats debugging aids that
trace
function
calls or put breakpoints on
function
calls:
You may want to watch
#py("make_rat")
being called, but you certainly don't want to watch every call to
#py("pair").

We have chosen not to use this style of definition in this book.
#anchor(<foot:proc-def-style>)]
#idx("makerat", decl: true)#idx("numer", decl: true)#idx("denom", decl: true)
#snippet(```python
def make_rat(n, d): return pair(n, d)

def numer(x): return head(x)

def denom(x): return tail(x)
```)

Also, in order to display the results of our computations, we can
#idx("rational number(s)", sub: "printing")
print rational numbers by printing the numerator, a slash, and the

denominator.
We use the primitive function
#idx("str (primitive function)")

#py("str") to turn any value (here
a number) into a string. The operator
#idx("string(s)", sub: "concatenation")
#idx("concatenating strings")
#idx("+", sub: "as string concatenation operator")

#py("+") in Python is
#idx("overloaded operator +")
#emph[overloaded]; it can be applied to two numbers or to two strings,
and in the latter case it returns the result of #emph[concatenating]
the two strings.

#idx("printrat", decl: true)
#snippet(```python
def print_rat(x):
    print(str(numer(x)) + " / " + str(denom(x)))
```)

Now we can try our rational-number
functions:

#snippet(```python
one_half = make_rat(1, 2)

print(print_rat(one_half))
```)

#output(```python
one_half = make_rat(1, 2)

print(print_rat(one_half))
```)

#snippet(```python
one_third = make_rat(1, 3)
```)

#snippet(```python
print(print_rat(add_rat(one_half, one_third)))
```)

#output(```python
print(print_rat(add_rat(one_half, one_third)))
```)

#snippet(```python
print(print_rat(mul_rat(one_half, one_third)))
```)

#output(```python
print(print_rat(mul_rat(one_half, one_third)))
```)

#snippet(```python
print(print_rat(add_rat(one_third, one_third)))
```)

#output(```python
print(print_rat(add_rat(one_third, one_third)))
```)

As the final example shows, our rational-number implementation does not
#idx("rational number(s)", sub: "reducing to lowest terms")
#idx("reducing to lowest terms")
reduce rational numbers to lowest terms. We can remedy this by changing
#py("make_rat").
If we have a
#py("gcd")
function
like the one in section @sec:gcd that produces
#idx("greatest common divisor", sub: "used in rational-number arithmetic")
the greatest common divisor of two integers, we can use
#py("gcd") to reduce the numerator and the
denominator to lowest terms before constructing the pair:

#idx("makerat", sub: "reducing to lowest terms", decl: true)
#snippet(```python
def make_rat(n, d):
    g = gcd(n, d)
    return pair(n // g, d // g)
```)

Now we have

#snippet(```python
print_rat(add_rat(one_third, one_third))
```)

#output(```python
print_rat(add_rat(one_third, one_third))
```)

as desired. This modification was accomplished by changing the constructor
#py("make_rat") without changing any of the
functions
(such as
#py("add_rat")
and
#py("mul_rat"))
that implement the actual operations.

#exercise(label-name: <ex:2_1>, [
Define a better version of
#py("make_rat")
that handles both positive and negative arguments.
The function #py("make_rat")
should normalize the sign so that if the rational number is positive, both
the numerator and denominator are positive, and if the rational number is
negative, only the numerator is negative.
])

#idx("arithmetic", sub: "on rational numbers")
#idx("rational number(s)", sub: "arithmetic operations on")
#idx("rational-number arithmetic")
