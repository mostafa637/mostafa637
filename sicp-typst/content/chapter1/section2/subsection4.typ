// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Exponentiation], label-name: <sec:exponentiation>)

#idx("exponentiation")

Consider the problem of computing the exponential of a given number.
We would like a
function
that takes as arguments a base $b$ and a positive
integer exponent $n$ and
computes $b^(n)$. One way to do this is via
the recursive definition

$ mat(delim: #none, b^(n), =, b dot.op b^(n-1); b^(0), =, 1) $

which translates readily into the
function
#idx("expt", sub: "linear recursive version", decl: true)
#snippet(```python
def expt(b, n):
    return 1 if n == 0 else b * expt(b, n - 1)
```)

This is a linear recursive process, which requires
$Theta (n)$ steps and
$Theta (n)$ space. Just as with factorial, we
can readily formulate an equivalent linear iteration:
#idx("expt", sub: "linear iterative version", decl: true)
#snippet(```python
def expt(b, n):
    return expt_iter(b, n, 1)

def expt_iter(b, counter, product):
    return (product
            if counter == 0
            else expt_iter(b, counter - 1, b * product))
```)

This version requires $Theta (n)$ steps and
$Theta (1)$ space.

We can compute exponentials in fewer steps by using
#idx("successive squaring")
successive squaring.
For instance, rather than computing $b^(8)$ as

$ b dot.op (b dot.op (b dot.op (b dot.op (b dot.op (b dot.op (b dot.op b)))))) $

we can compute it using three multiplications:

$ mat(delim: #none, b^(2), =, b dot.op b; b^(4), =, b^(2) dot.op b^(2); b^(8), =, b^(4) dot.op b^(4)) $

This method works fine for exponents that are powers of 2. We can also take
advantage of successive squaring in computing exponentials in general if we
use the rule

$ mat(delim: #none, b^(n), =, (b^(n/2))^(2), thin upright("if") space n space upright("is even"); b^(n), =, b dot.op b^(n-1), upright("if") space n space upright("is odd")) $

We can express this method as a
function, where the operator #idx("integer division") #idx("// (integer division operator)") #py("//") denotes #emph[integer division], which discards any fractional part of the quotient:
#idx("fastexpt", decl: true)
#snippet(```python
def fast_expt(b, n):
    return (1 if n == 0
            else square(fast_expt(b, n // 2)) if is_even(n)
            else b * fast_expt(b, n - 1))
```)

where the predicate to test whether an integer is even is defined in terms
of the
#idx("remainder", sub: "after integer division") #idx("\"% (remainder operator)", sort: "///") operator #py("%"), which computes the remainder after integer division,
by
#idx("iseven", decl: true)
#snippet(```python
def is_even(n):
    return n % 2 == 0
```)

The process evolved by
#py("fast_expt")
#idx("order of growth", sub: "logarithmic")
#idx("logarithmic growth")
grows logarithmically with $n$ in both space and
number of steps. To see this, observe that computing
$b^(2n)$ using
#py("fast_expt")
requires only one more multiplication than computing
$b^(n)$. The size of the exponent we can compute
therefore doubles (approximately) with every new multiplication we are
allowed. Thus, the number of multiplications required for an exponent of
$n$ grows about as fast as the logarithm of
$n$ to the base 2. The process has
$Theta ( log n)$ growth.#footnote[More precisely,
the number of multiplications required is equal to 1 less than the log
base 2 of $n$, plus the number of ones in the
binary representation of $n$. This total is
always less than twice the log base 2 of $n$.
The arbitrary constants $k_(1)$ and
$k_(2)$ in the definition of order notation imply
that, for a logarithmic process, the base to which logarithms are taken does
not matter, so all such processes are described as
$Theta ( log n)$.]

The difference between $Theta ( log n)$ growth
and $Theta (n)$ growth becomes striking as
$n$ becomes large. For example,
#py("fast_expt")
for $n=1000$ requires only 14
multiplications.#footnote[You may wonder why anyone would care about raising
numbers to the 1000th power. See
section @sec:primality.]
It is also possible to use the idea of successive squaring to devise an
iterative algorithm that computes exponentials with a logarithmic number of
steps (see exercise @ex:iter-expon-pro), although, as is
often the case with iterative algorithms, this is not written down so
straightforwardly as the recursive algorithm.#footnote[This iterative
algorithm is ancient. It appears in the
#idx("Chandah-sutra")
#emph[Chandah-sutra] by
#idx("Pingala, Áchárya")
Áchárya, written before 200 BCE.
See
#idx("Knuth, Donald E.")
Knuth 1997b, section 4.6.3, for a full discussion
and analysis of this and other methods of exponentiation.]
#idx("exponentiation")

#exercise(label-name: <ex:iter-expon-pro>, [
Design a
function
that evolves an iterative exponentiation process that uses successive
squaring and uses a logarithmic number of steps, as does
#py("fast_expt").
(Hint: Using the observation that
$(b^(n/2))^(2) =(b^(2))^(n/2)$, keep, along with the
exponent $n$ and the base
$b$, an additional state variable
$a$, and define the state transformation in such
a way that the product $a b^(n)$ is unchanged from
state to state. At the beginning of the process
$a$ is taken to be 1, and the answer is given by
the value of $a$ at the end of the process. In
general, the technique of defining an
#idx("invariant quantity of an iterative process")
#emph[invariant quantity] that remains unchanged from state to state is a
powerful way to think about the
design of
#idx("iterative process", sub: "design of algorithm")
iterative algorithms.)
])

#exercise(label-name: <ex:add-expon>, [
The exponentiation algorithms in this section are based on performing
exponentiation by means of repeated multiplication. In a similar way,
one can perform integer multiplication by means of repeated addition.
The following multiplication
function
(in which it is assumed that our language can only add, not multiply) is
analogous to the #py("expt")
function:

#snippet(```python
def times(a, b):
    return 0 if b == 0 else a + times(a, b - 1)
```)

This algorithm takes a number of steps that is linear in
#py("b"). Now suppose we include, together with
addition,
the functions
#py("double"), which doubles an
integer, and #py("halve"), which divides an (even)
integer by 2. Using these, design a multiplication
function
analogous to
#py("fast_expt")
that uses a logarithmic number of steps.
])

#exercise(label-name: <ex:it-pro-mult-int>, [
Using the results of exercises @ex:iter-expon-pro
and @ex:add-expon, devise a
function
that generates an iterative process for multiplying two integers in terms
of adding, doubling, and halving and uses a logarithmic number of
steps.#footnote[This
algorithm, which is sometimes known as the
#idx("Russian peasant method of multiplication")
#idx("multiplication by Russian peasant method")
"Russian peasant method" of multiplication, is ancient. Examples of its use are
found in the
#idx("Rhind Papyrus")
Rhind Papyrus, one of the two oldest mathematical documents in existence,
written about 1700 BCE (and copied from an even
older document) by an Egyptian scribe named
#idx("A'h-mose", sort: "Ahmose")
A'h-mose.]
])

#exercise(label-name: <ex:1_19>, [
There is a clever algorithm for computing the Fibonacci numbers in a
#idx("fib", sub: "logarithmic version", decl: true)
logarithmic number of steps. Recall the transformation of the state
variables $a$ and
$b$ in the
#py("fib_iter")
process of section @sec:tree-recursion:
$a arrow.l a+b$ and
$b arrow.l a$. Call this transformation
$T$, and observe that applying
$T$ over
and over again $n$ times, starting with 1 and 0,
produces the pair $upright("Fib")(n+1)$ and
$upright("Fib")(n)$. In other words, the
Fibonacci numbers are produced by applying
$T^(n)$, the $n$th
power of the transformation $T$, starting with
the pair $(1,0)$. Now consider
$T$ to be the special case of
$p=0$ and $q=1$ in
a family of transformations $T_(p q)$, where
$T_(p q)$ transforms the pair
$(a,b)$ according to
$a arrow.l b q+a q+a p$ and
$b arrow.l b p+a q$. Show that if we apply such
a transformation $T_(p q)$ twice, the effect is
the same as using a single transformation
$T_(p'q')$ of the same form, and compute
$p'$ and $q'$ in
terms of $p$
and $q$. This gives us an explicit way
to square these transformations, and thus we can compute
$T^(n)$ using successive squaring, as in the
#py("fast_expt")
function.
Put this all together to complete the following
function,
which runs in a logarithmic number of steps:#footnote[This exercise was
suggested

by
#idx("Stoy, Joseph E.")
Joe Stoy, based on an example in
#idx("Kaldewaij, Anne")
Kaldewaij 1990.]

#syntax("
def fib(n):
    return fib_iter(1, 0, 0, 1, n)

def fib_iter(a, b, p, q, count):
    return (b
            if count == 0
            else fib_iter(a,
                          b,
                          ", metaphrase[??], ",           # compute p'
                          ", metaphrase[??], ",           # compute q'
                          count // 2)
            if is_even(count)
            else fib_iter(b * q + a * q + a * p,
                          b * p + a * q,
                          p,
                          q,
                          count - 1))
	")
])
