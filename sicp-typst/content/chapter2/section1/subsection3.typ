// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([What Is Meant by Data?], label-name: <sec:data->)

#idx("data", sub: "meaning of")

We began the rational-number implementation in
section @sec:rationals by implementing the
rational-number operations
#py("add_rat"),
#py("sub_rat"),
and so on in terms of three unspecified
functions:
#py("make_rat"),
#py("numer"), and
#py("denom"). At that point, we could think of the
operations as being defined in terms of data objects—numerators,
denominators, and rational numbers—whose behavior was specified
by the latter three
functions.

But exactly what is meant by #emph[data]? It is not enough to say
"whatever is implemented by the given selectors and constructors." Clearly, not every arbitrary set of three
functions
can serve as an appropriate basis for the rational-number
implementation. We need to guarantee that,
#idx("makerat", sub: "axiom for")
#idx("numer", sub: "axiom for")
#idx("denom", sub: "axiom for")
if we construct a rational number #py("x") from a
pair of integers #py("n") and
#py("d"), then extracting the
#py("numer") and the
#py("denom") of #py("x") and
dividing them should yield the same result as dividing
#py("n") by #py("d"). In
other words,
#py("make_rat"),
#py("numer"), and
#py("denom") must satisfy the condition that, for
any integer #py("n") and any nonzero
integer #py("d"), if #py("x") is
#py("make_rat(n, d)"),
then

$ mat(delim: #none, frac(mono("numer")(mono("x")), mono("denom")(mono("x"))), =, frac(mono("n"), mono("d"))) $

In fact, this is the only condition
#py("make_rat"),
#py("numer"), and
#py("denom") must fulfill in order to form a
suitable basis for a rational-number representation. In general, we can
think of data as defined by some collection of selectors and
constructors, together with specified conditions that these
functions
must fulfill in order to be a valid
representation.#footnote[Surprisingly, this idea is very difficult to
formulate rigorously. There are two approaches to giving such a
formulation. One, pioneered by
#idx("Hoare, Charles Antony Richard")
C. A. R. Hoare (1972), is known as the method of
#idx("data", sub: "abstract models for")
#idx("abstract models for data")
#emph[abstract models]. It formalizes the
"functions plus conditions"
specification as outlined in the rational-number example above. Note
that the condition on the rational-number representation was stated in
terms of facts about integers (equality and division). In general,
abstract models define new kinds of data objects in terms of previously
defined types of data objects. Assertions about data objects can
therefore be checked by reducing them to assertions about previously
defined data objects. Another approach, introduced by
#idx("Zilles, Stephen N.")
Zilles at MIT, by
#idx("Goguen, Joseph")
Goguen,
#idx("Thatcher, James W.")
Thatcher,
#idx("Wagner, Eric G.")
Wagner, and
#idx("Wright, Jesse B.")
Wright at IBM (see Thatcher, Wagner, and Wright 1978), and by
#idx("Guttag, John Vogel")
Guttag at Toronto (see Guttag 1977),
is called
#idx("data", sub: "algebraic specification for")
#idx("algebraic specification for data")
#emph[algebraic specification]. It regards the
"functions"
as elements of an abstract algebraic system whose behavior is
specified by axioms that correspond to our "conditions,"
and uses the techniques of abstract algebra to check assertions about
data objects. Both methods are surveyed in the paper by
#idx("Liskov, Barbara Huberman")
Liskov and Zilles
(1975).]

#idx("data", sub: "functional representation of") #idx("functional representation of data")
This point of view can serve to define not only
"high-level" data objects, such as rational numbers, but
lower-level objects as well.
Consider the notion of a
#idx("pair(s)", sub: "functional representation of")
pair, which we used in order to define our
rational numbers. We never actually said what a pair was, only that
the language supplied
functions
#py("pair"),
#py("head"),
and
#py("tail")
for operating on pairs. But the only thing we need to know about these
three operations
is that if we glue two objects together using
#py("pair")
we can retrieve the objects using
#py("head")
and
#py("tail").
#idx("pair (primitive function)", sub: "axiom for")
#idx("head (primitive function)", sub: "axiom for")
#idx("tail (primitive function)", sub: "axiom for")
#idx("pair(s)", sub: "axiomatic definition of")
That is, the operations satisfy the condition that, for any objects
#py("x") and #py("y"), if
#py("z") is
#py("pair(x, y)")
then
#py("head(z)")
is #py("x") and
#py("tail(z)")
is #py("y"). Indeed, we mentioned that these three
functions
are included as primitives in our language. However, any triple of
functions
that satisfies the above condition can be used as the basis for
implementing pairs. This point is illustrated strikingly by the fact
that we could implement
#py("pair"),
#py("head"),
and
#py("tail")
without using any data structures at all but only using
functions.
Here are the definitions:#footnote[The function #idx("error (primitive function)", sub: "optional second argument") #py("error") introduced in section @sec:proc-general-methods takes as optional second argument a string that gets displayed before the first argument—for example, if #py("m") is 42: #output(```python Error in line 7: argument not 0 or 1 -- pair: 42 ```)]
#idx("pair (primitive function)", sub: "functional implementation of", decl: true)#idx("head (primitive function)", sub: "functional implementation of", decl: true)#idx("tail (primitive function)", sub: "functional implementation of", decl: true)
#snippet(```python
def pair(x, y):
    def dispatch(m):
        return (x if m == 0
                else y if m == 1
                else error("argument not 0 or 1 -- pair", m))
    return dispatch
def head(z): return z(0)

def tail(z): return z(1)
```)

This use of
functions
corresponds to nothing like our intuitive notion of what data should be.
Nevertheless, all we need to do to show that this is a valid way to
represent pairs is to verify that these
functions
satisfy the condition given above.

The subtle point to notice is that the value returned by
#py("pair(x, y)")
is a
function—namely
the internally defined
function
#py("dispatch"), which takes one argument and returns
either #py("x") or #py("y")
depending on whether the argument is 0 or 1. Correspondingly,
#py("head(z)")
is defined to apply #py("z") to 0. Hence, if
#py("z") is the
function
formed by
#py("pair(x, y)"),
then #py("z") applied to 0 will yield
#py("x"). Thus, we have shown that
#py("head(pair(x, y))")
yields #py("x"), as desired. Similarly,
#py("tail(pair(x, y))")
applies the
function
returned by
#py("pair(x, y)")
to 1, which returns #py("y").
Therefore, this
functional
implementation of pairs is a valid
implementation, and if we access pairs using only
#py("pair"),
#py("head"),
and
#py("tail")
we cannot distinguish this implementation from one that uses
"real" data structures.

The point of exhibiting the
functional
representation of pairs is not that our language works this way
(an efficient implementation of pairs might use Python's native #emph[list] data structure)
but that it could work this way. The
functional
representation, although obscure, is a perfectly adequate way to represent
pairs, since it fulfills the only conditions that pairs need to fulfill.
This example also demonstrates that the ability to manipulate
functions
as objects automatically provides the ability to represent compound data.
This may seem a curiosity now, but
functional
representations of data will play a central role in our programming
repertoire. This style of programming is often called
#idx("message passing")
#emph[message passing], and we will be using it as a basic tool in
chapter @chap:state when we address the issues of modeling and simulation.

#exercise(label-name: <ex:lambda-cons>, [
Here is an alternative
functional
representation of pairs. For this
representation, verify that
#py("head(pair(x, y))")
yields #py("x") for any objects
#py("x") and #py("y").
#idx("pair (primitive function)", sub: "functional implementation of", decl: true)#idx("head (primitive function)", sub: "functional implementation of", decl: true)#idx("tail (primitive function)", sub: "functional implementation of")
#snippet(```python
def pair(x, y):
    return lambda m: m(x, y)
def head(z):
    return z(lambda p, q: p)
```)

What is the corresponding definition of
#idx("tail (primitive function)", sub: "functional implementation of")
#py("tail")?
(Hint: To verify that this works, make use of the substitution model of
section @sec:substitution-model.)
])

#exercise(label-name: <ex:2_5>, [
Show that we can represent pairs of nonnegative integers using only
numbers and arithmetic operations if we represent the pair
$a$ and $b$ as the
integer that is the product $2^(a) 3^(b)$. Give the
corresponding definitions of the
functions
#py("pair"),
#py("head"),
and
#py("tail").
])

#idx("pair(s)", sub: "functional representation of")

#exercise(label-name: <ex:church-numerals>, [
In case representing pairs as
functions
(exercise @ex:lambda-cons)
wasn't mind-boggling enough, consider that, in a language that can
manipulate
functions,
we can get by without numbers (at least insofar as nonnegative integers
are concerned) by implementing 0 and the operation of adding 1 as

#snippet(```python
zero = lambda f: lambda x: x

def add_1(n):
    return lambda f: lambda x: f(n(f)(x))
```)

This representation is known as
#idx("Church numerals")
#emph[Church numerals], after its inventor,
#idx("Church, Alonzo")
Alonzo Church, the logician who invented the
$lambda$ calculus.

Define #py("one") and #py("two")
directly (not in terms of #py("zero") and
#py("add_1")).
(Hint: Use substitution to evaluate
#py("add_1(zero)")).
Give a direct definition of the addition
function #py("plus")
(not in terms of repeated application of
#py("add_1")).
])

#idx("data", sub: "functional representation of") #idx("functional representation of data") #idx("data", sub: "meaning of")
