// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Functions as General Methods], label-name: <sec:proc-general-methods>)

#idx("higher-order functions", sub: "function as general method")

We introduced compound
functions
in section @sec:compound-procedures as a mechanism for
abstracting patterns of numerical operations so as to make them independent
of the particular numbers involved. With higher-order
functions,
such as
the #py("integral")
function
of section @sec:procedures-as-parameters, we began to
see a more powerful kind of abstraction:
functions
used to express general methods of computation, independent of the
particular functions involved. In this section we discuss two more elaborate
examples—general methods for finding zeros and fixed points of
functions—and show how these methods can be expressed directly as
functions.

#subheading([Finding roots of equations by the half-interval method])

The
#idx("half-interval method")
#emph[half-interval method] is a simple but powerful technique for
finding roots of an equation $f(x)=0$, where
$f$ is a continuous function. The idea is that,
if we are given points $a$ and
$b$ such that
$f(a) < 0 < f(b)$, then
$f$ must have at least one zero between
$a$ and $b$. To locate
a zero, let $x$ be the average of
$a$ and $b$ and
compute $f(x)$. If
$f(x) > 0$, then
$f$ must have a zero between
$a$ and $x$. If
$f(x) < 0$, then
$f$ must have a zero between
$x$ and $b$.
Continuing in this way, we can identify smaller and smaller intervals on
which $f$ must have a zero. When we reach a
point where the interval is small enough, the process stops. Since the
interval of uncertainty is reduced by half at each step of the process, the
maximal number of steps required grows as
$Theta ( log ( L/T))$, where
$L$ is the length of the original interval and
$T$ is the error tolerance (that is, the size of
the interval we will consider "small enough").
Here is a
function that implements this strategy:

#idx("search", decl: true)
#snippet(```python
def search(f, neg_point, pos_point):
    midpoint = average(neg_point, pos_point)
    if close_enough(neg_point, pos_point):
        return midpoint
    else:
        test_value = f(midpoint)
        if positive(test_value):
            return search(f, neg_point, midpoint)
        elif negative(test_value):
            return search(f, midpoint, pos_point)
        else: return midpoint
```)

We assume that we are initially given the function
$f$ together with points at which its values are
negative and positive. We first compute the midpoint of the two given
points. Next we check to see if the given interval is small enough, and if
so we simply return the midpoint as our answer. Otherwise, we compute as a
test value the value of $f$ at the midpoint. If
the test value is positive, then we continue the process with a new interval
running from the original negative point to the midpoint. If the test value
is negative, we continue with the interval from the midpoint to the positive
point. Finally, there is the possibility that the test value is 0, in
which case the midpoint is itself the root we are searching for.

To test whether the endpoints are "close enough" we can use a
function
similar to the one used in section @sec:sqrt for
computing square roots:#footnote[We have used 0.001 as a representative
"small" number to indicate a tolerance for the acceptable error
in a calculation. The appropriate tolerance for a real calculation depends
upon the problem to be solved and the limitations of the computer and the
algorithm. This is often
a very subtle consideration, requiring help from a
#idx("numerical analyst")
numerical analyst or some
other kind of magician.]

#snippet(```python
def close_enough(x, y):
    return abs(x - y) < 0.001
```)

The function #py("search")
is awkward to use directly, because we can accidentally give it points at
which $f$'s values do not have the required
sign, in which case we get a wrong answer. Instead we will use
#py("search") via the following
function,
which checks to see which of the endpoints has a negative function value and
which has a positive value, and calls the #py("search")
function
accordingly. If the function has the same sign on the two given points, the
half-interval method cannot be used, in which case the
function
signals an error.#footnote[This
can be accomplished using
#idx("error (primitive function)")

#py("error"),
which takes as argument a string that is printed as error message along with the number of the program line that gave rise to the call of #py("error").]

#snippet(```python
def half_interval_method(f, a, b):
    a_value = f(a)
    b_value = f(b)
    if negative(a_value) and positive(b_value):
        return search(f, a, b)
    elif negative(b_value) and positive(a_value):
        return search(f, b, a)
    else: error("values are not of opposite sign")
```)

The following example uses the

#idx("half-interval method", sub: "halfintervalmethod")

#idx("π (pi)", sub: "approximation with half-interval method", sort: "pi")
half-interval method to approximate
$pi$ as the root between 2 and 4 of
$sin thin x = 0$:

#snippet(```python
print(half_interval_method(math_sin, 2, 4))
```)

#output(```python
print(half_interval_method(math_sin, 2, 4))
```)

Here is another example, using the half-interval method to search for a root
of the equation $x^(3) - 2x - 3 = 0$ between 1
and 2:

#snippet(```python
print(half_interval_method(lambda x: x * x * x - 2 * x - 3, 1, 2))
```)

#output(```python
print(half_interval_method(lambda x: x * x * x - 2 * x - 3, 1, 2))
```)

#idx("half-interval method")

#subheading([Finding fixed points of functions])

A number $x$ is called a
#idx("fixed point")
#idx("function (mathematical)", sub: "fixed point of")
#emph[fixed point] of a
function $f$ if $x$
satisfies the equation $f(x)=x$. For some
functions $f$ we can locate a fixed point by
beginning with an initial guess and applying $f$
repeatedly,

$ f(x), space f(f(x)), space f(f(f(x))), space dots.h $

until the value does not change very much. Using this idea, we can devise a
function
#py("fixed_point")
that takes as inputs a function and an initial guess and produces an
approximation to a fixed point of the function. We apply the function
repeatedly until we find two successive values whose difference is less
than some prescribed tolerance:

#idx("fixedpoint", decl: true)
#snippet(```python
tolerance = 0.00001
def fixed_point(f, first_guess):
    def close_enough(x, y):
        return abs(x - y) < tolerance
    def try_with(guess):
        next = f(guess)
        return next if close_enough(guess, next) else try_with(next)
    return try_with(first_guess)
```)

For example, we can use this method to approximate the fixed point of the
#idx("fixed point", sub: "of cosine")
#idx("cosine", sub: "fixed point of")
#idx("mathcos (primitive function)")

cosine function, starting with 1 as an initial approximation:#footnote[To obtain a
#idx("fixed point", sub: "computing with calculator")
#idx("calculator, fixed points with")
fixed point of cosine on a calculator,
set it to radians mode and then repeatedly press the
$cos$
button until the value does not change any longer.]

#snippet(```python
print(fixed_point(math_cos, 1))
```)

#output(```python
print(fixed_point(math_cos, 1))
```)

Similarly, we can find a solution to the equation
$y= sin y + cos y$:
#idx("mathsin (primitive function)")

#snippet(```python
print(fixed_point(lambda y: math_sin(y) + math_cos(y), 1))
```)

#output(```python
print(fixed_point(lambda y: math_sin(y) + math_cos(y), 1))
```)

The fixed-point process is reminiscent of the process we used for finding
square roots in section @sec:sqrt. Both are based on
the idea of repeatedly improving a guess until the result satisfies some
criterion. In fact, we can readily formulate the
#idx("fixed point", sub: "square root as")
square-root computation as a fixed-point search. Computing the square root
of some number $x$ requires finding a
$y$ such that
$y^(2) = x$. Putting this equation into the
equivalent form $y = x/y$, we recognize that we
are looking for a fixed point of the function#footnote[$arrow.r.bar$
#idx("↦ notation for mathematical function", sort: "0a2")
#idx("function (mathematical)", sub: "↦ notation for")
(pronounced "maps to") is the mathematician's way of
writing
lambda expressions.
$y arrow.r.bar x/y$ means
#py("lambda y: x / y"),
that is, the function whose value at $y$ is
$x/y$.]
$y arrow.r.bar x/y$, and we can therefore try to
compute square roots as
#idx("sqrt", sub: "as fixed point", decl: true)
#snippet(```python
def sqrt(x):
    return fixed_point(lambda y: x / y, 1)
```)

Unfortunately, this fixed-point search does not converge. Consider an
initial guess $y_(1)$. The next guess is
$y_(2) = x/y_(1)$ and the next guess is
$y_(3) = x/y_(2) = x/(x/y_(1)) = y_(1)$. This results
in an infinite loop in which the two guesses
$y_(1)$ and $y_(2)$ repeat
over and over, oscillating about the answer.

One way to control such oscillations is to prevent the guesses from changing
so much. Since the answer is always between our guess
$y$
and $x/y$, we can make a new guess that is not as
far from $y$ as $x/y$
by averaging $y$ with
$x/y$, so that the next guess after
$y$ is
$frac(1, 2)(y+x/y)$ instead of
$x/y$. The process of making such a sequence of
guesses is simply the process of looking for a fixed point of
$y arrow.r.bar frac(1, 2)(y+x/y)$:

#snippet(```python
def sqrt(x):
    return fixed_point(lambda y: average(y, x / y), 1)
```)

(Note that $y=frac(1, 2)(y+x/y)$ is a simple
transformation of the equation $y=x/y$; to derive
it, add $y$ to both sides of the equation and
divide by 2.)

With this modification, the square-root
function
works. In fact, if we unravel the definitions, we can see that the sequence
of approximations to the square root generated here is precisely the same as
the one generated by our original square-root
function
of section @sec:sqrt. This approach of averaging
successive approximations to a solution, a technique we call
#idx("average damping")
#emph[average damping], often aids the convergence of fixed-point searches.
#idx("fixed point")
#idx("function (mathematical)", sub: "fixed point of")

#exercise(label-name: <ex:1_35>, [
Show that the
#idx("golden ratio", sub: "as fixed point")
#idx("fixed point", sub: "golden ratio as")
golden ratio $phi$
(section @sec:tree-recursion) is a fixed point of the
transformation $x arrow.r.bar 1 + 1/x$, and use this
fact to compute $phi$ by means of the
#py("fixed_point") function.
])

#exercise(label-name: <ex:log-fixed-point>, [
Modify
#py("fixed_point")
so that it prints the sequence of approximations it generates, using the
primitive function
#py("print")
Then find a solution to
$x^(x) = 1000$ by finding a fixed point of
$x arrow.r.bar log (1000)/ log (x)$.
#idx("mathlog (primitive function)") (Use the primitive function #py("math_log"), which computes natural logarithms.)
Compare the number of steps this takes with and without average damping.
(Note that you cannot start
#py("fixed_point") with a guess of 1, as this would cause division by
$log (1)=0$.)
])

#exercise(label-name: <ex:continued-fractions>, [
An infinite
#idx("continued fraction")
#emph[continued fraction] is an expression of the form

$ mat(delim: #none, f, =, (frac(N_(1), D_(1)+ frac(N_(2), D_(2)+ frac(N_(3), D_(3)+ dots.c ))))) $

As an example, one can show that the infinite continued fraction
expansion with the $N_(i)$ and the
$D_(i)$ all equal to 1 produces
$1/ phi$, where
$phi$ is the
#idx("continued fraction", sub: "golden ratio as")
#idx("golden ratio", sub: "as continued fraction")
golden ratio (described in
section @sec:tree-recursion). One way to approximate
an infinite continued fraction is to truncate the expansion after a given
number of terms. Such a truncation—a so-called
#emph[$k$-term finite continued fraction]—has the form

$ (frac(N_(1), D_(1) + frac(N_(2), dots.down + frac(N_(K), D_(K))))) $

+ Suppose that #py("n") and #py("d") are functions of one argument (the term index $i$) that return the $N_(i)$ and $D_(i)$ of the terms of the continued fraction. Define a function #py("cont_frac") such that evaluating #py("cont_frac(n, d, k)") computes the value of the $k$-term finite continued fraction. Check your function by approximating $1/ phi$ using #snippet(```python print(cont_frac(lambda i: 1, lambda i: 1, k)) ```) for successive values of #py("k"). How large must you make #py("k") in order to get an approximation that is accurate to 4 decimal places?
+ If your #py("cont_frac") function generates a recursive process, write one that generates an iterative process. If it generates an iterative process, write one that generates a recursive process.
])

#exercise(label-name: <ex:1_38>, [
In 1737, the Swiss mathematician
#idx("Euler, Leonhard")
Leonhard Euler published a memoir
#emph[De Fractionibus Continuis], which included a
#idx("continued fraction", sub: "e as")
#idx("e", sub: "as continued fraction", sort: "e")
continued fraction expansion for $e-2$, where
$e$ is the base of the natural logarithms. In
this fraction, the $N_(i)$ are all 1, and the
$D_(i)$ are successively
1, 2, 1, 1, 4, 1, 1, 6, 1, 1, 8, …. Write a program that uses your
#py("cont_frac") function
from exercise @ex:continued-fractions to approximate
$e$, based on Euler's expansion.
])

#exercise(label-name: <ex:1_39>, [
A continued fraction representation of the tangent function was
published in 1770 by the German mathematician
#idx("Lambert, J.H.")
#idx("continued fraction", sub: "tangent as")
#idx("tangent", sub: "as continued fraction")
J.H. Lambert:

$ mat(delim: #none, tan x, =, (frac(x, 1- frac(x^(2), 3- frac(x^(2), 5- frac(x^(2), dots.down )))))) $

where $x$ is in radians.
Define a function #py("tan_cf(x, k)")
that computes an approximation to the tangent function based on
Lambert's formula.
As in exercise @ex:continued-fractions, #py("k") specifies the number of terms to compute.
])

#idx("higher-order functions", sub: "function as general method")
