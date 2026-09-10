// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Tree Recursion], label-name: <sec:tree-recursion>)

#idx("tree-recursive process")
#idx("process", sub: "tree-recursive")
#idx("recursive process", sub: "tree")

Another common pattern of computation is called #emph[tree recursion].
As an example, consider computing the sequence of
#idx("Fibonacci numbers")
Fibonacci numbers,
in which each number is the sum of the preceding two:

$ 0, 1, 1, 2, 3, 5, 8, 13, 21, dots.h $

In general, the Fibonacci numbers can be defined by the rule

$ mat(delim: #none, upright("Fib")(n), =, cases(0 quad upright("if n=0"), 1 quad upright("if n=1"), upright("Fib")(n-1)+upright("Fib")(n-2) quad upright("otherwise"))) . $

We can immediately translate this definition into a recursive
function
for computing Fibonacci numbers:
#idx("fib", sub: "tree-recursive version", decl: true)
#snippet(```python
def fib(n):
    return (0 if n == 0
            else 1 if n == 1
            else fib(n - 1) + fib(n - 2))
```)

#sicp-figure(image("/images/img_javascript/ch1-Z-G-13.svg", width: 70%), caption: [The tree-recursive process generated in computing #py("fib(5)").], label-name: <fig:fib-tree>)

Consider the pattern of this computation. To compute
#py("fib(5)"),
we compute
#py("fib(4)")
and
#py("fib(3)").
To compute
#py("fib(4)"),
we compute
#py("fib(3)")
and
#py("fib(2)").
In general, the evolved process looks like a tree, as shown in
figure @fig:fib-tree.
Notice that the branches split into
two at each level (except at the bottom); this reflects the fact that the
#py("fib")
function
calls itself twice each time it is invoked.

This
function
is instructive as a prototypical tree recursion, but it is a terrible way to
compute Fibonacci numbers because it does so much redundant computation.
Notice in
figure @fig:fib-tree
that the entire
computation of
#py("fib(3)")—almost half the work—is
duplicated. In fact, it is not hard to show that the number of times the
function
will compute
#py("fib(1)")
or
#py("fib(0)")
(the number of leaves in the above tree, in general) is precisely
$upright("Fib")(n+1)$. To get an idea of how
bad this is, one can show that the value of
$upright("Fib")(n)$
#idx("exponential growth", sub: "of tree-recursive Fibonacci-number computation")
grows exponentially with $n$. More precisely
(see exercise @ex:fib-proof),
$upright("Fib")(n)$ is the closest integer to
$phi ^(n) /sqrt(5)$, where

$ mat(delim: #none, phi, =, (1+sqrt(5))/2, approx, 1.6180) $

is the
#idx("golden ratio")
#emph[golden ratio], which satisfies the equation

$ mat(delim: #none, phi ^(2), =, phi + 1) $

Thus, the process uses a number of steps that grows exponentially with the
input. On the other hand, the space required grows only linearly with the
input, because we need keep track only of which nodes are above us in the
tree at any point in the computation. In general, the number of steps
required by a tree-recursive process will be proportional to the number of
nodes in the tree, while the space required will be proportional to the
maximum depth of the tree.

We can also formulate an iterative process for computing the Fibonacci
numbers. The idea is to use a pair of integers $a$
and $b$, initialized to
$upright("Fib")(1)=1$ and
$upright("Fib")(0)=0$, and to repeatedly apply the
simultaneous transformations

$ mat(delim: #none, a, arrow.l, a+b; b, arrow.l, a) $

It is not hard to show that, after applying this transformation
$n$ times, $a$ and
$b$ will be equal, respectively, to
$upright("Fib")(n+1)$ and
$upright("Fib")(n)$. Thus, we can compute
Fibonacci numbers iteratively using the
function
#idx("fib", sub: "linear iterative version", decl: true)
#snippet(```python
def fib(n):
    return fib_iter(1, 0, n)

def fib_iter(a, b, count):
    return b if count == 0 else fib_iter(a + b, a, count - 1)
```)

This second method for computing $upright("Fib")(n)$
is a linear iteration. The difference in number of steps required by the two
methods—one linear in $n$, one growing as
fast as $upright("Fib")(n)$ itself—is
enormous, even for small inputs.

One should not conclude from this that tree-recursive processes are useless.
When we consider processes that operate on hierarchically structured data
rather than numbers, we will find that tree recursion is a natural and
powerful tool.#footnote[An example of this was hinted at in
section @sec:evaluating-combinations: The interpreter
itself evaluates expressions using a tree-recursive process.] But
even in numerical operations, tree-recursive processes can be useful in
helping us to understand and design programs. For instance, although the
first
#py("fib")
function
is much less efficient than the second one, it is more straightforward,
being little more than a translation into
Python
of the definition of the Fibonacci sequence. To formulate the iterative
algorithm required noticing that the computation could be recast as an
iteration with three state variables.

#subheading([Example: Counting change])

#idx("counting change")

It takes only a bit of cleverness to come up with the iterative Fibonacci
algorithm. In contrast, consider the following problem:
How many different ways can we make change of
\$1.00 (100 cents),
given half-dollars, quarters, dimes, nickels, and pennies
(50 cents, 25 cents, 10 cents, 5 cents, and 1 cent, respectively)?
More generally, can
we write a
function
to compute the number of ways to change any given amount of money?

This problem has a simple solution as a recursive
function.
Suppose we think of the types of coins available as arranged in some order.
Then the following relation holds:

#blockquote[The number of ways to change amount $a$ using
$n$ kinds of coins equals

- the number of ways to change amount $a$ using all but the first kind of coin, plus
- the number of ways to change amount $a-d$ using all $n$ kinds of coins, where $d$ is the denomination of the first kind of coin.]

To see why this is true, observe that the ways to make change can be divided
into two groups: those that do not use any of the first kind of coin, and
those that do. Therefore, the total number of ways to make change for some
amount is equal to the number of ways to make change for the amount without
using any of the first kind of coin, plus the number of ways to make change
assuming that we do use the first kind of coin. But the latter number is
equal to the number of ways to make change for the amount that remains after
using a coin of the first kind.

Thus, we can recursively reduce the problem of changing a given amount to
problems of changing smaller amounts or using fewer kinds of coins. Consider
this reduction rule carefully, and convince yourself that we can use it to
describe an algorithm if we specify the following degenerate
cases:#footnote[For example, work through in detail how the reduction rule
applies to the problem of making change for 10 cents using pennies and
nickels.]

- If $a$ is exactly 0, we should count that as 1 way to make change.
- If $a$ is less than 0, we should count that as 0 ways to make change.
- If $n$ is 0, we should count that as 0 ways to make change.

We can easily translate this description into a recursive
function:

#idx("countchange", decl: true)
#snippet(```python
def count_change(amount):
    return cc(amount, 5)

def cc(amount, kinds_of_coins):
    return (1 if amount == 0
            else 0 if amount < 0 or kinds_of_coins == 0
            else cc(amount, kinds_of_coins - 1)
                 + cc(amount - first_denomination(kinds_of_coins),
                      kinds_of_coins))

def first_denomination(kinds_of_coins):
    return (1 if kinds_of_coins == 1
            else 5 if kinds_of_coins == 2
            else 10 if kinds_of_coins == 3
            else 25 if kinds_of_coins == 4
            else 50 if kinds_of_coins == 5
            else 0)
```)

(The
#py("first_denomination") function
takes as input the number of kinds of coins available and returns the
denomination of the first kind. Here we are thinking of the coins as
arranged in order from largest to smallest, but any order would do as well.)
We can now answer our original question about changing a dollar:

#snippet(```python
print(count_change(100))
```)

#output(```python
print(count_change(100))
```)

The function #py("count_change")
generates a tree-recursive process with redundancies similar to those in
our first implementation of #py("fib").

On the other hand, it is not
obvious how to design a better algorithm for computing the result, and we
leave this problem as a challenge. The observation that a
#idx("efficiency", sub: "of tree-recursive process")
tree-recursive process may be highly inefficient but often easy to specify
and understand has led people to propose that one could get the best of both
worlds by designing a "smart compiler" that could transform
tree-recursive
functions
into more efficient
functions
that compute the same result.#footnote[One approach to coping with redundant
computations is to arrange matters so that we automatically construct a
table of values as they are computed. Each time we are asked to apply the
function
to some argument, we first look to see if the value is already stored in the
table, in which case we avoid performing the redundant computation. This
strategy, known as
#idx("tabulation")
#emph[tabulation] or
#idx("memoization")
#emph[memoization], can be implemented in a
straightforward way. Tabulation can sometimes be used to transform processes
that require an exponential number of steps
(such as #py("count_change"))
into processes whose space and time requirements grow linearly with the
input. See exercise @ex:memoization.]
#idx("tree-recursive process")
#idx("process", sub: "tree-recursive")
#idx("recursive process", sub: "tree")
#idx("counting change")

#exercise(label-name: <ex:1_11>, [
A function $f$ is defined by the
rules
$f(n)=n$ if $n < 3$
and $f(n)=(f(n-1))+2f(n-2)+3f(n-3)$ if
$n gt.eq 3$. Write a
Python function
that computes $f$ by means of a recursive process.
Write a
function
that computes $f$ by means of an iterative
process.
])

#exercise(label-name: <ex:1_12>, [
The following pattern of numbers is called
#idx("Pascal's triangle")
#emph[Pascal's triangle].

$ ( mat(delim: #none, , , , , 1, , , , ; , , , 1, , 1, , , ; , , 1, , 2, , 1, , ; , 1, , 3, , 3, , 1, ; 1, , 4, , 6, , 4, , 1; , , , , dots.h, , , , )) $

The numbers at the edge of the triangle are all 1, and each number inside
the triangle is the sum of the two numbers above it.#footnote[The elements
of Pascal's triangle are called the #emph[binomial coefficients],
because the $n$th row consists of
#idx("binomial coefficients")
the coefficients of the terms in the expansion of
$(x+y)^(n)$. This pattern for computing the
coefficients
appeared in
#idx("Pascal, Blaise", sort: "Pascal")
Blaise Pascal's 1653 seminal work on probability theory,
#emph[Traité du triangle arithmétique].
According to
#idx("Edwards, Anthony William Fairbank")
Edwards (2019), the same pattern appears
in the works of
the eleventh-century Persian mathematician
#idx("Al-Karaji")
Al-Karaji,
in the works of the twelfth-century Hindu mathematician
#idx("Bhaskara")
Bhaskara, and
in the works of the
thirteenth-century Chinese mathematician
#idx("Yang Hui")
Yang Hui.]
Write a
function
that computes elements of Pascal's triangle by means of a recursive
process.
])

#exercise(label-name: <ex:fib-proof>, [
Prove that $upright("Fib")(n)$ is the closest
integer to $phi ^(n)/sqrt(5)$, where
$phi = (1+sqrt(5))/2$.

Hint: Use induction and the
definition of the Fibonacci numbers to prove that
$upright("Fib")(n)=( phi ^(n)- psi ^(n))/sqrt(5)$,
where
$psi = (1-sqrt(5))/2$.
])
