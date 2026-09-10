// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Greatest Common Divisors], label-name: <sec:gcd>)

#idx("greatest common divisor")

The greatest common divisor (GCD) of two integers
$a$ and $b$ is defined
to be the largest integer that divides both $a$
and $b$ with no remainder. For example, the GCD
of 16 and 28 is 4. In chapter @chap:data, when we investigate how to
implement rational-number arithmetic, we will need to be able to compute
GCDs in order to reduce rational numbers to lowest terms. (To reduce a
rational number to lowest terms, we must divide both the numerator and the
denominator by their GCD. For example, 16/28 reduces to 4/7.) One way to
find the GCD of two integers is to factor them and search for common
factors, but there is a famous algorithm that is much more efficient.

#idx("Euclid's Algorithm")
The idea of the algorithm is based on the observation that, if
$r$ is the remainder when
$a$ is divided by
$b$, then the common divisors of
$a$ and $b$ are
precisely the same as the common divisors of $b$
and $r$. Thus, we can use the equation

$ mat(delim: #none, upright("GCD") (a, b), =, upright("GCD")(b, r)) $

to successively reduce the problem of computing a GCD to the problem of
computing the GCD of smaller and smaller pairs of integers. For example,

$ mat(delim: #none, upright("GCD")(206,40), =, upright("GCD")(40,6); , =, upright("GCD")(6,4); , =, upright("GCD")(4,2); , =, upright("GCD")(2,0); , =, 2) $

reduces $upright("GCD")(206, 40)$ to
$upright("GCD")(2, 0)$, which is 2. It is
possible to show that starting with any two positive integers and
performing repeated reductions will always eventually produce a pair
where the second number is 0. Then the GCD is the other
number in the pair. This method for computing the GCD is
known as #emph[Euclid's Algorithm].#footnote[Euclid's
Algorithm is so
called because it appears in Euclid's
#idx("Euclid's Elements", sort: "Euclids Elements")
#emph[Elements] (Book 7,
ca. 300 BCE). According to
#idx("Knuth, Donald E.")
Knuth (1997a), it can be considered the
oldest known nontrivial algorithm. The ancient Egyptian method of
multiplication (exercise @ex:it-pro-mult-int) is surely
older, but, as Knuth explains, Euclid's Algorithm is the oldest known
to have been presented as a general algorithm, rather than as a set of
illustrative examples.]

It is easy to express Euclid's Algorithm as a
function:
#idx("gcd", decl: true)
#snippet(```python
def gcd(a, b):
    return a if b == 0 else gcd(b, a % b)
```)

This generates an iterative process, whose number of steps grows as
the logarithm of the numbers involved.

The fact that the number of steps required by Euclid's Algorithm has
#idx("Euclid's Algorithm", sub: "order of growth")
logarithmic growth bears an interesting relation to the
#idx("Fibonacci numbers", sub: "Euclid's GCD algorithm and")
Fibonacci numbers:

#blockquote[#strong[Lamé's Theorem:]
#idx("Lamé's Theorem", sort: "Lames")
If Euclid's Algorithm
requires $k$ steps to compute the GCD of some
pair, then the smaller number in the pair must be greater than or equal
to the $k$th Fibonacci number.#footnote[This
theorem was proved in 1845 by
#idx("Lamé, Gabriel", sort: "Lame")
Gabriel Lamé, a
French mathematician and engineer known chiefly for his contributions
to mathematical physics. To prove the theorem, we consider pairs
$(a_(k) ,b_(k))$, where
$a_(k) gt.eq b_(k)$, for which Euclid's
Algorithm terminates in $k$ steps. The proof is
based on the claim that, if
$(a_(k+1), space b_(k+1)) arrow.r (a_(k), space b_(k)) arrow.r (a_(k-1), space b_(k-1))$ are three successive pairs
in the reduction process, then we must have
$b_(k+1) gt.eq b_(k) + b_(k-1)$.
To verify the claim, consider that a reduction step is defined by applying
the transformation $a_(k-1) = b_(k)$,
$b_(k-1) = upright("remainder of") space a_(k) space upright("divided by") space b_(k)$.
The second equation means that
$a_(k) = q b_(k) + b_(k-1)$ for some positive
integer $q$. And since
$q$ must be at least 1 we have
$a_(k) = q b_(k) + b_(k-1) gt.eq b_(k) + b_(k-1)$.
But in the previous reduction step we have
$b_(k+1)= a_(k)$. Therefore,
$b_(k+1) = a_(k) gt.eq b_(k) + b_(k-1)$.
This verifies the claim. Now we can prove the theorem by induction on
$k$, the number of steps that the algorithm
requires to terminate. The result is true for
$k=1$, since this merely requires that
$b$ be at least as large as
$upright("Fib")(1)=1$. Now, assume that the result
is true for all integers less than or equal to
$k$ and establish the result for
$k+1$. Let
$(a_(k+1), space b_(k+1)) arrow.r (a_(k), space b_(k)) arrow.r (a_(k-1), space b_(k-1))$ be successive pairs in the
reduction process. By our induction hypotheses, we have
$b_(k-1) gt.eq (upright("Fib"))(k-1)$ and
$b_(k) gt.eq (upright("Fib"))(k)$. Thus, applying
the claim we just proved together with the definition of the Fibonacci
numbers gives
$b_(k+1) gt.eq b_(k) + b_(k-1) gt.eq (upright("Fib"))(k) + (upright("Fib"))(k-1) = (upright("Fib"))(k+1)$, which completes
the proof of Lamé's Theorem.]]

We can use this theorem to get an order-of-growth estimate for Euclid's
Algorithm. Let $n$ be the smaller of the two
inputs to the
function.
If the process takes $k$ steps, then we must have
$n gt.eq (upright("Fib")) (k) approx phi ^(k)/sqrt(5)$.
Therefore the number of steps $k$ grows as the
logarithm (to the base $phi$) of
$n$. Hence, the order of growth is
$Theta ( log n)$.
#idx("greatest common divisor")
#idx("Euclid's Algorithm")

#exercise(label-name: <ex:gcd-process>, [
The process that a
function
generates is of course dependent on the rules used by the interpreter.
As an example, consider the iterative #py("gcd")
function
given above. Suppose we were to interpret this
function
using
#idx("normal-order evaluation", sub: "applicative order vs.")
#idx("applicative-order evaluation", sub: "normal order vs.")
normal-order evaluation, as discussed in
section @sec:substitution-model. (The
normal-order-evaluation rule for
conditional expressions
is described
in exercise @ex:normal-order-vs-appl-order-test.)
Using the substitution method (for normal order), illustrate the process
generated in evaluating
#py("gcd(206, 40)")
and indicate the #py("remainder") operations that are
actually performed. How many #py("remainder")
operations are actually performed in the normal-order evaluation of
#py("gcd(206, 40)")?
In the applicative-order evaluation?
])
