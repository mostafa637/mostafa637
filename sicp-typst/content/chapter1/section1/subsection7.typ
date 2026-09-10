// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Example: Square Roots by Newton's Method], label-name: <sec:sqrt>)

#idx("function (mathematical)", sub: "Python function vs.")
Functions,
as introduced above, are much like ordinary mathematical functions. They
specify a value that is determined by one or more parameters. But there
is an important difference between mathematical functions and computer
functions.
Computer functions
must be effective.

As a case in point, consider the problem of computing square
roots. We can define the square-root function as

$ sqrt(x) space =upright(" the ")y u p r i g h t(" such that ")y gt.eq 0upright(" and ") y^(2) space = space x $

This describes a perfectly legitimate mathematical function. We could
use it to recognize whether one number is the square root of another, or
to derive facts about square roots in general. On the other hand, the
definition does not describe a
computer function.
Indeed, it tells us almost nothing about how to actually find the square
root of a given number. It will not help matters to rephrase this
definition in
pseudo-Python:

#syntax("
def sqrt(x):
    return the y ", $mono("with")$, " y >= 0 and square(y) == x
      ")

This only begs the question.

The contrast between
mathematical function and computer function
is a reflection of the general distinction between describing properties of
things and describing how to do things, or, as it is sometimes referred to,
the distinction between
#idx("declarative vs. imperative knowledge")
#idx("imperative vs. declarative knowledge")
declarative knowledge and imperative knowledge. In
#idx("mathematics", sub: "computer science vs.")
#idx("computer science", sub: "mathematics vs.")
mathematics we are usually concerned with declarative (what is)
descriptions, whereas in computer science we are usually concerned
with imperative (how to) descriptions.#footnote[Declarative and
imperative descriptions are intimately related, as indeed are
mathematics and computer science. For instance, to say that the
answer produced by a program is
#idx("correctness of a program")
"correct" is to make a declarative statement about the program.
There is a large amount of research aimed at establishing techniques for
#idx("proving programs correct")
proving that programs are correct, and much of the technical difficulty of
this subject has to do with negotiating the transition between imperative
statements (from which programs are constructed) and declarative statements
(which can be used to deduce things).
In a related vein, programming language designers have explored so-called #idx("programming language", sub: "very high-level") #idx("very high-level language") very high-level languages, in which one actually programs in terms of declarative statements.
The idea is to make interpreters sophisticated
enough so that, given "what is" knowledge specified by the
programmer, they can generate "how to" knowledge automatically.
This cannot be done in general, but there are important areas where progress
has been made. We shall revisit this idea in chapter @chap:meta.]
#idx("function (mathematical)", sub: "Python function vs.")

How does one compute
#idx("square root")
#idx("Newton's method", sub: "for square roots")
square roots? The most common way is to use
Newton's method of successive approximations, which says that whenever
we have a guess $y$ for the value of the square
root of a number $x$, we can perform a simple
manipulation to get a better guess (one closer to the actual square root)
by averaging $y$ with
$x/y$.#footnote[This square-root algorithm is
actually a special case of Newton's method, which is a general
technique for finding roots of equations. The square-root algorithm itself
was developed by Heron of
#idx("Heron of Alexandria")
Alexandria in the first century CE. We will see how to express
the general Newton's method as a
Python function
in section @sec:proc-returned-values.]
For example, we can compute the square root of 2 as follows. Suppose our
initial guess is 1:

$ mat(delim: #none, upright("Guess"), upright("Quotient"), upright("Average"); 1, ( frac(2, 1) = 2), ( frac((2+1), 2) = 1.5); 1.5, ( frac(2, 1.5) = 1.3333), ( frac((1.3333+1.5), 2) = 1.4167); 1.4167, ( frac(2, 1.4167) = 1.4118), ( frac((1.4167+1.4118), 2) = 1.4142); 1.4142, dots.h, dots.h) $

Continuing this process, we obtain better and better approximations to the
square root.

Now let's formalize the process in terms of functions. We start with
a value for the
#idx("radicand")
radicand (the number whose square root we are trying to compute) and a value
for the guess. If the guess is good enough for our purposes, we are done;
if not, we must repeat the process with an improved guess. We write this
basic strategy as a
function:

#snippet(```python
def sqrt_iter(guess, x):
    return (guess
            if is_good_enough(guess, x)
            else sqrt_iter(improve(guess, x), x))
```)

A guess is improved by averaging it with the quotient of the radicand and
the old guess:

#snippet(```python
def improve(guess, x):
    return average(guess, x / guess)
```)

where
#idx("average", decl: true)
#snippet(```python
def average(x, y):
    return (x + y) / 2
```)

We also have to say what we mean by "good enough." The
following will do for illustration, but it is not really a very good
test. (See exercise @ex:ex-sqrt-end-test.)
The idea is to improve the answer until it is close enough so that its
square differs from the radicand by less than a predetermined
tolerance (here 0.001):#footnote[We will usually give
#idx("predicate", sub: "naming convention for")
#idx("naming conventions", sub: "isfor predicates")
#idx("is, in predicate names", sort: "is")
predicates names starting with #py("is_"), to help us remember that they
are predicates.]

#snippet(```python
def is_good_enough(guess, x):
    return abs(square(guess) - x) < 0.001
```)

Finally, we need a way to get started. For instance, we can always guess
that the square root of any number
is 1:

#idx("sqrt", decl: true)
#snippet(```python
def sqrt(x):
    return sqrt_iter(1, x)
```)

If we type these
definitions
to the interpreter, we can use #py("sqrt")
just as we can use any
function:

#snippet(```python
print(sqrt(9))
```)

#output(```python
print(sqrt(9))
```)

#snippet(```python
print(sqrt(100 + 37))
```)

#output(```python
print(sqrt(100 + 37))
```)

#snippet(```python
print(sqrt(sqrt(2) + sqrt(3)))
```)

#output(```python
print(sqrt(sqrt(2) + sqrt(3)))
```)

#idx("square root")#idx("Newton's method", sub: "for square roots")
#snippet(```python
print(square(sqrt(1000)))
```)

#output(```python
print(square(sqrt(1000)))
```)

The #py("sqrt") program also illustrates that the
simple
#idx("iterative process", sub: "implemented by function call") functional
language we have introduced so far is sufficient for writing any purely
numerical program that one could write in, say, C or Pascal. This might
seem surprising, since we have not included in our language any iterative
#idx("looping constructs")
(looping) constructs that direct the computer to do something over and over
again.
The function #py("sqrt_iter"),
on the other hand, demonstrates how iteration can be accomplished using no
special construct other than the ordinary ability to call a
function.#footnote[Readers who are worried about the efficiency issues involved in using
function
calls to implement iteration should note the remarks on "tail recursion" in
section @sec:recursion-and-iteration.]
#idx("iterative process", sub: "implemented by function call")

#exercise([
Alyssa P. Hacker doesn't see why #py("if")
#idx("syntactic form", sub: "need for")
#idx("conditional expression", sub: "why a syntactic form")
needs to be provided as a syntactic form.
"Why can't I just define it as an ordinary conditional function whose application works just like conditional expressions?"
she asks.#footnote[As a Lisp hacker from the original #emph[Structure and Interpretation of Computer Programs], Alyssa prefers a simpler, more uniform
syntax.]
Alyssa's friend Eva Lu Ator claims this can indeed be
done, and she defines a #py("conditional")
function as follows:

#snippet(```python
def conditional(predicate, then_clause, else_clause):
    return then_clause if predicate else else_clause
```)

Eva demonstrates the program for Alyssa:

#snippet(```python
print(conditional(2 == 3, 0, 5))
```)

#output(```python
print(conditional(2 == 3, 0, 5))
```)

#snippet(```python
print(conditional(1 == 1, 0, 5))
```)

#output(```python
print(conditional(1 == 1, 0, 5))
```)

Delighted, Alyssa uses
#py("conditional") to rewrite the square-root
program:

#snippet(```python
def sqrt_iter(guess, x):
    return conditional(is_good_enough(guess, x),
                       guess,
                       sqrt_iter(improve(guess, x),
                                 x))
```)

What happens when Alyssa attempts to use this to compute square roots?
Explain.
#anchor(<ex:new-if>)
])

#exercise(label-name: <ex:ex-sqrt-end-test>, [
The
#py("is_good_enough")
test used in computing square roots will not be very effective for finding
the square roots of very small numbers. Also, in real computers, arithmetic
operations are almost always performed with limited precision. This makes
our test inadequate for very large numbers. Explain these statements, with
examples showing how the test fails for small and large numbers. An
alternative strategy for implementing
#py("is_good_enough")
is to watch how #py("guess") changes from one
iteration to the next and to stop when the change is a very small fraction
of the guess. Design a square-root
function
that uses this kind of end test. Does this work better for small and
large numbers?
])

#exercise(label-name: <ex:cube-root-newton>, [
Newton's method for
#idx("cube root", sub: "by Newton's method")
#idx("Newton's method", sub: "for cube roots")
cube roots is based on the fact that if
$y$ is an
approximation to the cube root of $x$, then a better approximation is
given by the value

$ frac(x/y^(2)+2y, 3) $

Use this formula to implement a cube-root
function
analogous to the square-root
function.
(In section @sec:proc-returned-values we will see how to
implement Newton's method in general as an abstraction of these
square-root and cube-root
functions.)
])
