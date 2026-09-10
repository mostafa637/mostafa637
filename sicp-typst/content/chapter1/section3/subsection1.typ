// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Functions as Arguments], label-name: <sec:procedures-as-parameters>)

#idx("higher-order functions", sub: "function as argument")

Consider the following three
functions.
The first computes the sum of the integers from
#py("a") through #py("b"):
#idx("sumintegers", decl: true)
#snippet(```python
def sum_integers(a, b):
    return 0 if a > b else a + sum_integers(a + 1, b)
```)

The second computes the sum of the cubes of the integers in the given range:
#idx("sumcubes", decl: true)
#snippet(```python
def sum_cubes(a, b):
    return 0 if a > b else cube(a) + sum_cubes(a + 1, b)
```)

The third computes the sum of a sequence of terms in the series

$ frac(1, 1 dot.op 3)+frac(1, 5 dot.op 7)+frac(1, 9 dot.op 11)+ dots.c $

which converges to $pi /8$ (very
slowly):#footnote[This series,
#idx("π (pi)", sub: "Leibniz's series for", sort: "pi")
#idx("Leibniz, Baron Gottfried Wilhelm von", sub: "series for π")
usually written in the equivalent form
$frac( pi , 4) = 1-frac(1, 3)+frac(1, 5)-frac(1, 7)+ dots.c$,
is due to Leibniz. We'll see how to use this as the basis for some
fancy numerical tricks in
section @sec:exploiting-streams.]
#idx("pisum", decl: true)
#snippet(```python
def pi_sum(a, b):
    return 0 if a > b else 1 / (a * (a + 2)) + pi_sum(a + 4, b)
```)

These three
functions
clearly share a common underlying pattern. They are for the most part
identical, differing only in the name of the
function,
the function of #py("a") used to compute the term to
be added, and the function that provides the next value of
#py("a"). We could generate each of the
functions
by filling in slots in the same template:

#syntax("
def ", meta("name"), "(a, b):
    return (0 if a > b
            else ", meta("term"), "(a) + ", meta("name"), "(", meta("next"), "(a), b))
      ")

The presence of such a common pattern is strong evidence that there is a
useful
#idx("abstraction", sub: "common pattern and")
abstraction waiting to be brought to the surface. Indeed,
mathematicians long ago identified the abstraction of
#idx("series, summation of")
#idx("summation of a series")
#idx("Σ (sigma) notation", sort: "sigma")
#idx("Σ (sigma) notation", sort: "0s")
#emph[summation of a series] and invented "sigma notation," for example

$ mat(delim: #none, sum_(n=a)^(b) space f(n), =, f(a)+ dots.c +f(b)) $

to express this concept. The power of sigma notation is that it allows
mathematicians to deal with the concept of summation itself rather than only
with particular sums—for example, to formulate general results about
sums that are independent of the particular series being summed.

Similarly, as program designers, we would like our language to be powerful
enough so that we can write a
function
that expresses the concept of summation itself rather than only
functions
that compute particular sums. We can do so readily in our
functional
language by taking the common template shown above and transforming the
"slots" into
parameters:
#idx("sum", decl: true)
#snippet(```python
def sum(term, a, next, b):
    return 0 if a > b else term(a) + sum(term, next(a), next, b)
```)

Notice that #py("sum") takes as its arguments the
lower and upper bounds #py("a") and
#py("b") together with the
functions
#py("term") and #py("next").
We can use #py("sum") just as we would any
function.
For example, we can use it (along with a
function
#py("inc") that increments its argument by 1) to define
#py("sum_cubes"):

#idx("inc", decl: true)#idx("sumcubes", sub: "with higher-order functions", decl: true)
#snippet(```python
def inc(n):
    return n + 1
def sum_cubes(a, b):
    return sum(cube, a, inc, b)
```)

Using this, we can compute the sum of the cubes of the integers from 1 to 10:

#snippet(```python
print(sum_cubes(1, 10))
```)

#output(```python
print(sum_cubes(1, 10))
```)

With the aid of an identity
function
to compute the term, we can define
#py("sum_integers")
in terms of #py("sum"):
#idx("identity", decl: true)
#snippet(```python
def identity(x):
    return x
```)

#idx("sumintegers", sub: "with higher-order functions", decl: true)
#snippet(```python
def sum_integers(a, b):
    return sum(identity, a, inc, b)
```)

Then we can add up the integers from 1 to 10:

#snippet(```python
print(sum_integers(1, 10))
```)

#output(```python
print(sum_integers(1, 10))
```)

We can also
define #py("pi_sum")
in the same way:#footnote[Notice that we have used block structure
(section @sec:black-box) to embed the
definitions of #py("pi_next")
and
#py("pi_term")
within
#py("pi_sum"),
since these
functions
are unlikely to be useful for any other purpose. We will see how to get rid
of them altogether in section @sec:lambda.]
#idx("pisum", sub: "with higher-order functions", decl: true)
#snippet(```python
def pi_sum(a, b):
    def pi_term(x):
        return 1 / (x * (x + 2))
    def pi_next(x):
        return x + 4
    return sum(pi_term, a, pi_next, b)
```)

Using these
functions,
we can compute an approximation to $pi$:

#snippet(```python
print(8 * pi_sum(1, 1000))
```)

#output(```python
print(8 * pi_sum(1, 1000))
```)

Once we have #py("sum"), we can use it as a building
block in formulating further concepts. For instance, the
#idx("definite integral")
definite integral of a function $f$ between the
limits $a$ and $b$ can
be approximated numerically using the formula

$ mat(delim: #none, integral_(a)^(b)f, =, lr([ thin f l r(( a+frac(d x, 2) )) thin + thin f l r(( a+d x+frac(d x, 2) )) thin + thin f l r(( a+2d x+frac(d x, 2) )) thin + thin dots.c ]) d x) $

for small values of $d x$. We can express this
directly as a
function:
#idx("integral", decl: true)
#snippet(```python
def integral(f, a, b, dx):
    def add_dx(x):
        return x + dx
    return sum(f, a + dx / 2, add_dx, b) * dx
```)

#snippet(```python
print(integral(cube, 0, 1, 0.01))
```)

#output(```python
print(integral(cube, 0, 1, 0.01))
```)

#snippet(```python
print(integral(cube, 0, 1, 0.001))
```)

#output(```python
print(integral(cube, 0, 1, 0.001))
```)

(The exact value of the integral of #py("cube") between
0 and 1 is 1/4.)

#exercise(label-name: <ex:simpsons-rule>, [
Simpson's Rule is a more accurate method of numerical integration than

#idx("Simpson's Rule for numerical integration")
the method illustrated above. Using Simpson's Rule, the integral of a
function $f$ between
$a$ and $b$ is
approximated as

$ frac(h, 3)[ y_(0) +4y_(1) +2y_(2) +4y_(3) +2y_(4) + dots.c +2y_(n-2) +4y_(n-1)+y_(n) ] $

where $h=(b-a)/n$, for some even integer
$n$, and
$y_(k) =f(a+k h)$. (Increasing
$n$ increases the accuracy of the approximation.)
Define a function
that takes as arguments $f$,
$a$, $b$, and
$n$ and returns the value of the integral,
computed using Simpson's Rule. Use your
function
to integrate #py("cube") between 0 and 1 (with
$n=100$ and $n=1000$),
and compare the results to those of the #py("integral")
function
shown above.

#anchor(<ex:1_29>)
])

#idx("definite integral")

#exercise(label-name: <ex:1_30>, [
The
#idx("sum", sub: "iterative version")
#py("sum")
function
above generates a linear recursion. The
function
can be rewritten so that the sum is performed iteratively. Show how to do
this by filling in the missing expressions in the following
definition:

#syntax("
def sum(term, a, next, b):
    def iterate(a, result):
        return (", metaphrase[??], "
                if ", metaphrase[??], "
                else iterate(", metaphrase[??], ", ", metaphrase[??], "))
    return iterate(", metaphrase[??], ", ", metaphrase[??], ")
      ")
])

#exercise(label-name: <ex:product>, [
+ The #py("sum") function is only the simplest of a vast number of similar abstractions that can be captured as higher-order functions.#footnote[The intent of exercises @ex:product–@ex:filtered-accumulate is to demonstrate the expressive power that is attained by using an appropriate abstraction to consolidate many seemingly disparate operations. However, though accumulation and filtering are elegant ideas, our hands are somewhat tied in using them at this point since we do not yet have data structures to provide suitable means of combination for these abstractions. We will return to these ideas in section @sec:sequences-conventional-interfaces when we show how to use #emph[sequences] as interfaces for combining filters and accumulators to build even more powerful abstractions. We will see there how these methods really come into their own as a powerful and elegant approach to designing programs.] Write an analogous function called #idx("product") #py("product") that returns the product of the values of a function at points over a given range. Show how to define #idx("factorial", sub: "with higher-order functions") #py("factorial") in terms of #py("product"). Also use #py("product") to compute approximations to #idx("π (pi)", sub: "Wallis's formula for", sort: "pi") $pi$ using the formula#footnote[This formula was discovered by the seventeenth-century English mathematician #idx("Wallis, John") John Wallis.] $ mat(delim: #none, frac( pi , 4), =, frac(2 dot.op 4 dot.op 4 dot.op 6 dot.op 6 dot.op 8 dots.c , 3 dot.op 3 dot.op 5 dot.op 5 dot.op 7 dot.op 7 dots.c )) $
+ If your #py("product") function generates a recursive process, write one that generates an iterative process. If it generates an iterative process, write one that generates a recursive process.
])

#exercise(label-name: <ex:accumulate>, [
+ Show that #py("sum") and #py("product") #idx("sum", sub: "as accumulation") #idx("product", sub: "as accumulation") (exercise @ex:product) are both special cases of a still more general notion called #idx("accumulate") #py("accumulate") that combines a collection of terms, using some general accumulation function: #snippet(```python accumulate(combiner, null_value, term, a, next, b) ```) The function #py("accumulate") takes as arguments the same term and range specifications as #py("sum") and #py("product"), together with a #py("combiner") function (of two arguments) that specifies how the current term is to be combined with the accumulation of the preceding terms and a #py("null_value") that specifies what base value to use when the terms run out. Write #py("accumulate") and show how #py("sum") and #py("product") can both be defined as simple calls to #py("accumulate").
+ If your #py("accumulate") function generates a recursive process, write one that generates an iterative process. If it generates an iterative process, write one that generates a recursive process.
])

#exercise(label-name: <ex:filtered-accumulate>, [
You can
obtain an even more general version of
#idx("filteredaccumulate")
#py("accumulate")
(exercise @ex:accumulate)
by introducing the notion of a
#idx("filter")
#emph[filter] on the terms to be combined. That is, combine only those
terms derived from values in the range that satisfy a specified condition.
The resulting
#idx("filteredaccumulate")
#py("filtered_accumulate")
abstraction takes the same arguments as accumulate, together with an
additional predicate of one argument that specifies the filter. Write
#py("filtered_accumulate")
as a
function.
Show how to express the following using
#py("filtered_accumulate"):

+ the sum of the squares of the prime numbers in the interval $a$ to $b$ (assuming that you have an #py("is_prime") predicate already written)
+ the product of all the positive integers less than $n$ that are #idx("relatively prime") relatively prime to $n$ (i.e., all positive integers $i < n$ such that $upright("GCD")(i,n)=1$).
])

#idx("higher-order functions", sub: "function as argument")
