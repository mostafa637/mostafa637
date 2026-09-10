// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Functions as Returned Values], label-name: <sec:proc-returned-values>)

#idx("higher-order functions", sub: "function as returned value")

The above examples demonstrate how the ability to pass
functions
as arguments significantly enhances the expressive power of our programming
language. We can achieve even more expressive power by creating
functions
whose returned values are themselves
functions.

We can illustrate this idea by looking again at the fixed-point example
described at the end of
section @sec:proc-general-methods. We formulated a new
version of the square-root
function
as a fixed-point search, starting with the observation that
$sqrt(x)$ is a fixed-point of the function
$y arrow.r.bar x/y$. Then we used average damping to
make the approximations converge. Average damping is a useful general
technique in itself. Namely, given a
function $f$, we consider the function
whose value at $x$ is equal to the average of
$x$ and $f(x)$.

We can express the idea of average damping by means of the following
function:
#idx("averagedamp", decl: true)
#snippet(```python
def average_damp(f):
    return lambda x: average(x, f(x))
```)

The function #py("average_damp")
takes as its argument a
function
#py("f") and returns as its value a
function
(produced by the lambda expression)
that, when applied to a number #py("x"), produces the
average of #py("x") and
#py("f(x)").
For example, applying
#py("average_damp")
to the #py("square")
function
produces a
function
whose value at some number $x$ is the average of
$x$ and $x^(2)$.
Applying this resulting
function
to 10 returns the average of 10 and 100, or 55:#footnote[Observe that this
is an application whose function expression is itself
#idx("function application", sub: "as function expression of application")
#idx("function expression", sub: "application as")
an application. Exercise @ex:a-plus-abs-b already
demonstrated the ability to form such applications, but that was only a toy
example. Here we begin to see the real need for such
applications—when applying a function
that is obtained as the value returned by a higher-order function.]

#snippet(```python
print(average_damp(square)(10))
```)

#output(```python
print(average_damp(square)(10))
```)

Using
#py("average_damp"),
we can reformulate the
#idx("fixed point", sub: "square root as")
square-root
function
as follows:
#idx("sqrt", sub: "as fixed point", decl: true)
#snippet(```python
def sqrt(x):
    return fixed_point(average_damp(lambda y: x / y), 1)
```)

Notice how this formulation makes explicit the three ideas in the method:
fixed-point search, average damping, and the function
$y arrow.r.bar x/y$. It is instructive to compare
this formulation of the square-root method with the original version given
in section @sec:sqrt. Bear in mind that these
functions
express the same process, and notice how much clearer the idea becomes when
we express the process in terms of these abstractions. In general, there
are many ways to formulate a process as a
function.
Experienced programmers know how to choose
process
formulations that are particularly perspicuous, and where useful elements of
the process are exposed as separate entities that can be reused in other
applications. As a simple example of reuse, notice that the cube root of
$x$ is a fixed point of the function
$y arrow.r.bar x/y^(2)$, so we can immediately
generalize our square-root
function
to one that extracts
#idx("cube root", sub: "as fixed point")
#idx("fixed point", sub: "cube root as")
cube roots:#footnote[See exercise @ex:nth-roots
for a further generalization.]
#idx("cuberoot", decl: true)
#idx("cuberoot", decl: true)
#snippet(```python
def cube_root(x):
    return fixed_point(average_damp(lambda y: x / square(y)), 1)
```)

#subheading([Newton's method])

When we first introduced the square-root
function,
in section @sec:sqrt, we mentioned that this was a
special case of
#idx("Newton's method", sub: "for differentiable functions")
#emph[Newton's method]. If
$x arrow.r.bar g(x)$ is a differentiable function,
then a solution of the equation $g(x)=0$ is a
fixed point of the function $x arrow.r.bar f(x)$ where

$ mat(delim: #none, f(x), =, x - frac(g(x), D g(x))) $

and $D g(x)$ is the derivative of
$g$ evaluated at $x$.
#idx("fixed point", sub: "in Newton's method")
Newton's method is the use of the fixed-point method we saw above to
approximate a solution of the equation by finding a fixed point of the
function $f$.#footnote[Elementary calculus books
usually describe Newton's method in terms of the sequence of
approximations $x_(n+1)=x_(n)-g(x_(n))/D g(x_(n))$.
Having language for talking about processes and using the idea of fixed
points simplifies the description of the method.]
For many functions $g$ and for sufficiently good
initial guesses for $x$, Newton's method
converges very rapidly to a solution of
$g(x)=0$.#footnote[Newton's method does not
always converge to an answer, but it can be shown that in favorable cases
each iteration doubles the number-of-digits accuracy of the approximation
to the solution. In such cases,
#idx("Newton's method", sub: "half-interval method vs.")
#idx("half-interval method", sub: "Newton's method vs.")
Newton's method will converge much more rapidly than the half-interval
method.]

In order to implement Newton's method as a
function,
we must first express the idea of
#idx("derivative of a function")
#idx("function (mathematical)", sub: "derivative of")
#idx("differentiation", sub: "numerical")
derivative. Note that
"derivative," like average damping, is something that
transforms a function into another function. For instance, the derivative
of the function $x arrow.r.bar x^(3)$ is the function
$x arrow.r.bar 3x^(2)$. In general, if
$g$ is a function and
$d x$ is a small number, then the derivative
$D g$ of $g$ is the
function whose value at any number $x$ is given
(in the limit of small $d x$) by

$ mat(delim: #none, D g(x), =, frac(g(x+d x) - g(x), d x)) $

Thus, we can express the idea of derivative (taking
$d x$ to be, say, 0.00001) as the
function

#snippet(```python
def deriv(g):
    return lambda x: (g(x + dx) - g(x)) / dx
```)

along with the
definition

#snippet(```python
dx = 0.00001
```)

Like
#idx("deriv (numerical)", decl: true)
#py("average_damp"),
#idx("deriv (numerical)", decl: true)
#py("deriv") is a
function
that takes a
function
as argument and returns a
function
as value. For example, to approximate the derivative of
$x arrow.r.bar x^(3)$ at 5 (whose exact value is 75)
we can evaluate
#idx("cube", decl: true)
#snippet(```python
def cube(x):
    return x * x * x

print(deriv(cube)(5))
```)

#output(```python
def cube(x):
    return x * x * x

print(deriv(cube)(5))
```)

With the aid of #py("deriv"), we can express
Newton's method as a fixed-point process:

#idx("newtontransform", decl: true)#idx("newtonsmethod", decl: true)
#snippet(```python
def newton_transform(g):
    return lambda x: x - g(x) / deriv(g)(x)
def newtons_method(g, guess):
    return fixed_point(newton_transform(g), guess)
```)

The
#py("newton_transform")
function
expresses the formula at the beginning of this section, and
#py("newtons_method")
is readily defined in terms of this. It takes as arguments a
function
that computes the function for which we want to find a zero, together with
an initial guess. For instance, to find the
square root of $x$, we can use
#idx("Newton's method", sub: "for square roots")
Newton's
method to find a zero of the function
$y arrow.r.bar y^(2)-x$ starting with an initial guess
of 1.#footnote[For finding square roots, Newton's method converges
rapidly to the correct solution from any starting point.]
This provides yet another form of the square-root
function:
#idx("sqrt", sub: "with Newton's method", decl: true)#idx("sqrt", sub: "as fixed point", decl: true)
#snippet(```python
def sqrt(x):
    return newtons_method(lambda y: square(y) - x, 1)
```)

#idx("Newton's method", sub: "for differentiable functions")

#subheading([Abstractions and first-class functions])

We've seen two ways to express the square-root computation as an
instance of a more general method, once as a fixed-point search and once
using Newton's method. Since Newton's method was itself
expressed as a fixed-point process, we actually saw two ways to compute
square roots as fixed points. Each method begins with a function and finds a
#idx("fixed point", sub: "of transformed function")
fixed point of some transformation of the function. We can express this
general idea itself as a
function:
#idx("fixedpointoftransform", decl: true)
#snippet(```python
def fixed_point_of_transform(g, transform, guess):
    return fixed_point(transform(g), guess)
```)

This very general
function
takes as its arguments a
function
#py("g")
that computes some function, a
function
that transforms #py("g"), and an initial guess.
The returned result is a fixed point of the transformed function.

Using this abstraction, we can recast the first square-root computation
#idx("fixed point", sub: "square root as")
from this section (where we look for a fixed point of the average-damped
version of $y arrow.r.bar x/y$) as an instance of
this general method:
#idx("sqrt", sub: "as fixed point", decl: true)
#snippet(```python
def sqrt(x):
    return fixed_point_of_transform(lambda y: x / y,
                                    average_damp,
                                    1)
```)

Similarly, we can express the second square-root computation from this
section (an instance of
#idx("Newton's method", sub: "for square roots")
Newton's method that finds a fixed point of
the Newton transform of $y arrow.r.bar y^(2)-x$) as
#idx("sqrt", sub: "as fixed point", decl: true)#idx("sqrt", sub: "with Newton's method", decl: true)
#snippet(```python
def sqrt(x):
    return fixed_point_of_transform(lambda y: square(y) - x,
                                    newton_transform,
                                    1)
```)

We began section @sec:higher-order-procedures with the
observation that compound
functions
are a crucial abstraction mechanism, because they permit us to express
general methods of computing as explicit elements in our programming
language. Now we've seen how higher-order
functions
permit us to manipulate these general methods to create further abstractions.

As programmers, we should be alert to opportunities to identify the
underlying abstractions in our programs and to build upon them and
generalize them to create more powerful abstractions. This is not to say
that one should always write programs in the most abstract way possible;
expert programmers know how to choose the level of abstraction appropriate
to their task. But it is important to be able to think in terms of these
abstractions, so that we can be ready to apply them in new contexts. The
significance of higher-order
functions
is that they enable us to represent these abstractions explicitly as
elements in our programming language, so that they can be handled just
like other computational elements.

In general, programming languages impose restrictions on the ways in which
computational elements can be manipulated. Elements with the fewest
restrictions are said to have
#idx("first-class elements in language")
#emph[first-class] status. Some of the "rights and privileges" of first-class elements are:#footnote[The notion of
first-class status of programming-language
elements is due to the British computer scientist
#idx("Strachey, Christopher")
Christopher Strachey (1916–1975).]

- They may be referred to using names.
- They may be passed as arguments to functions.
- They may be returned as the results of functions.
- They may be included in data structures.#footnote[We'll see examples of this after we introduce data structures in chapter @chap:data.]

Python,
#idx("Python", sub: "first-class functions in")

like other high-level
programming languages, awards
functions
full first-class status. This poses challenges for efficient
implementation, but the resulting gain in expressive power is
enormous.#footnote[The major implementation cost of first-class
functions
is that allowing
functions
to be returned as values requires reserving storage for a
function's free names
even while the
function
is not executing.
In the Python implementation we will study in section @sec:mc-eval, these names are stored in the function's environment.]

#exercise(label-name: <ex:1_40>, [
Define a function
#py("cubic") that can be used together with the
#py("newtons_method")
function
in expressions of the form

#snippet(```python
newtons_method(cubic(a, b, c), 1)
```)

to approximate zeros of the cubic
$x^(3) +a x^(2) +b x +c$.
])

#exercise(label-name: <ex:1_41>, [
Define a function
#py("double") that takes a
function
of one argument as argument and returns a
function
that applies the original
function
twice. For example, if #py("inc") is a
function
that adds 1 to its argument, then
#py("double(inc)")
should be a
function
that adds 2. What value is printed by

#snippet(```python
print(double(double(double))(inc)(5))
```)
])

#exercise(label-name: <ex:compose>, [
Let $f$ and $g$ be
two one-argument functions. The
#idx("composition of functions")
#idx("function (mathematical)", sub: "composition of")
#emph[composition]
$f$ after $g$ is
defined to be the function $x arrow.r.bar f(g(x))$.
Define a function
#py("compose") that implements composition. For
example, if #py("inc") is a
function
that adds 1 to its argument,

#snippet(```python
print(compose(square, inc)(6))
```)

#output(```python
print(compose(square, inc)(6))
```)
])

#exercise(label-name: <ex:repeated>, [
If $f$ is a numerical function and
#idx("function (mathematical)", sub: "repeated application of")
$n$ is a positive integer, then we can form the
$n$th
#idx("function (mathematical)", sub: "repeated application of")
repeated application of
$f$, which is defined to be the function whose
value at $x$ is
$f(f( dots.h (f(x)) dots.h ))$. For example, if
$f$ is the function
$x arrow.r.bar x+1$, then the
$n$th repeated application of
$f$ is the function
$x arrow.r.bar x+n$. If
$f$ is the operation of squaring a number, then
the $n$th repeated application of
$f$ is the function that raises its argument to
the $2^(n)$th power. Write a
function
that takes as inputs a
function
that computes $f$ and a positive integer
$n$ and returns the
function
that computes the $n$th repeated application of
$f$. Your
function
should be able to be used as follows:

#snippet(```python
print(repeated(square, 2)(5))
```)

#output(```python
print(repeated(square, 2)(5))
```)

Hint: You may find it convenient to use
#py("compose") from
exercise @ex:compose.
])

#exercise(label-name: <ex:smooth>, [
The idea of
#idx("function (mathematical)", sub: "smoothing of")
#idx("smoothing a function")
#emph[smoothing] a function is an important concept in
#idx("signal processing", sub: "smoothing a function")
signal processing. If $f$ is a function and
$d x$ is some small number, then the smoothed
version of $f$ is the function whose value at a
point $x$ is the average of
$f(x-d x)$, $f(x)$, and
$f(x+d x)$. Write a
function
#py("smooth") that takes as input a
function
that computes $f$ and returns a
function
that computes the smoothed $f$. It is sometimes
valuable to repeatedly smooth a function (that is, smooth the smoothed
function, and so on) to obtained the #emph[$n$-fold smoothed function]. Show how to generate the
$n$-fold smoothed function of any given function
using #py("smooth") and
#py("repeated") from
exercise @ex:repeated.

#anchor(<ex:1_44>)
])

#exercise(label-name: <ex:nth-roots>, [
We saw in section @sec:proc-general-methods that
attempting to compute square roots by naively finding a fixed point of
$y arrow.r.bar x/y$ does not converge, and that this
can be fixed by average damping. The same method works for finding cube
roots as fixed points of the average-damped
$y arrow.r.bar x/y^(2)$. Unfortunately, the process
does not work for
#idx("fourth root, as fixed point")
#idx("fixed point", sub: "fourth root as")
fourth roots—a single average damp is not enough to make a
fixed-point search for $y arrow.r.bar x/y^(3)$
converge. On the other hand, if we average-damp twice (i.e., use the
average damp of the average damp of
$y arrow.r.bar x/y^(3)$) the fixed-point search does
converge. Do some experiments to determine how many average damps are
required to compute
#idx("nth root, as fixed point", sort: "nth")
#idx("fixed point", sub: "nth root as")
$n$th roots as a fixed-point search based upon
repeated average damping of $y arrow.r.bar x/y^(n-1)$.
Use this to implement a simple
function
for computing $n$th roots using
#py("fixed_point"),
#py("average_damp"),
and the #py("repeated")
function
of exercise @ex:repeated. Assume that any arithmetic
operations you need are available as primitives.
])

#exercise(label-name: <ex:1_46>, [
Several of the numerical methods described in this chapter are instances
of an extremely general computational strategy known as
#idx("iterative improvement")
#idx("sqrt", sub: "as iterative improvement")
#idx("fixedpoint", sub: "as iterative improvement")
#idx("fixed point", sub: "as iterative improvement")
#emph[iterative improvement]. Iterative improvement says that, to compute something,
we start with an initial guess for the answer, test if the guess is good
enough, and otherwise improve the guess and continue the process using the
improved guess as the new guess. Write a
function
#py("iterative_improve")
that takes two
functions
as arguments: a method for telling whether a guess is good enough and a
method for improving a guess.
The function #py("iterative_improve")
should return as its value a
function
that takes a guess as argument and keeps improving the guess until it is
good enough. Rewrite the #py("sqrt")
function
of section @sec:sqrt and the
#py("fixed_point")
function
of section @sec:proc-general-methods in terms of
#py("iterative_improve").
])

#idx("higher-order functions", sub: "function as returned value")
