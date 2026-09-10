// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Example: Testing for Primality], label-name: <sec:primality>)

#idx("prime number(s)", sub: "testing for")
#idx("prime number(s)")

This section describes two methods for checking the primality of an
integer $n$, one with order of growth
$Theta (sqrt(n))$, and a
"probabilistic" algorithm with order of growth
$Theta ( log n)$. The exercises at the end of
this section suggest programming projects based on these algorithms.

#subheading([Searching for divisors])

Since ancient times, mathematicians have been fascinated by problems
concerning prime numbers, and many people have worked on the problem
of determining ways to test if numbers are prime. One way
to test if a number is prime is to find the number's divisors. The
following program finds the smallest integral divisor (greater than 1)
of a given number $n$. It does this in a
straightforward way, by testing $n$ for
divisibility by successive integers starting with 2.
#idx("finddivisor", decl: true)#idx("divides", decl: true)#idx("smallestdivisor", decl: true)
#snippet(```python
def smallest_divisor(n):
    return find_divisor(n, 2)

def find_divisor(n, test_divisor):
    return (n if square(test_divisor) > n
            else test_divisor if divides(test_divisor, n)
            else find_divisor(n, test_divisor + 1))

def divides(a, b):
    return b % a == 0
```)

We can test whether a number is prime as follows:
$n$ is prime if and only if
$n$ is its own smallest divisor.
#idx("isprime", decl: true)
#snippet(```python
def is_prime(n):
    return n == smallest_divisor(n)
```)

The end test for
#py("find_divisor")
is based on the fact that if $n$ is not prime it
must have a divisor less than or equal to
$sqrt(n)$.#footnote[If
$d$ is a divisor of
$n$, then so is $n/d$.
But $d$ and $n/d$
cannot both be greater than $sqrt(n)$.]
This means that the algorithm need only test divisors between 1 and
$sqrt(n)$. Consequently, the number of steps
required to identify $n$ as prime will have order
of growth $Theta (sqrt(n))$.

#subheading([The Fermat test])

The $Theta ( log n)$ primality test is based on
a result from number theory known as
#idx("Fermat test for primality")
#idx("prime number(s)", sub: "Fermat test for")
Fermat's Little
Theorem.#footnote[Pierre
#idx("Fermat, Pierre de", sort: "Fermat")
de Fermat (1601–1665) is considered to be
the founder of modern
#idx("number theory")
number theory. He obtained many important number-theoretic results,
but he usually announced just the results, without providing his proofs.
#idx("Fermat's Little Theorem", sub: "proof", sort: "Fermats")
Fermat's Little Theorem was stated in a letter he wrote in 1640.
The first published proof was given by
#idx("Euler, Leonhard", sub: "proof of Fermat's Little Theorem")
Euler in 1736 (and an
earlier, identical proof was discovered in the unpublished manuscripts
of
#idx("Leibniz, Baron Gottfried Wilhelm von", sub: "proof of Fermat's Little Theorem")
Leibniz). The most famous of Fermat's results—known as
Fermat's Last Theorem—was jotted down in 1637 in his copy of
the book #emph[Arithmetic] (by the third-century Greek mathematician
#idx("Diophantus's Arithmetic, Fermat's copy of")
Diophantus) with the remark "I have discovered a truly remarkable proof, but this margin is too small to contain it." Finding a proof
of Fermat's Last Theorem became one of the most famous challenges in
number theory. A complete
solution was finally given in 1995 by
#idx("Wiles, Andrew")
Andrew Wiles of Princeton
University.]

#blockquote[#strong[Fermat's Little Theorem:]
#idx("Fermat's Little Theorem", sort: "Fermats")
If $n$ is a prime number and
$a$ is any positive integer less than
$n$, then $a$ raised
to the $n$th power is congruent to
$a$ modulo $n$.]

(Two numbers are said to be
#idx("congruent modulo n")
#emph[congruent modulo]
$n$ if they both have the same remainder when
divided by $n$. The remainder of a number
$a$ when divided by
$n$ is also referred to as the
#idx("remainder", sub: "modulo n")
#idx("modulo n")
#emph[remainder of] $a$ #emph[modulo]
$n$, or simply as $a$
#emph[modulo] $n$.)

If $n$ is not prime, then, in general, most of
the numbers $a < n$ will not satisfy the above
relation. This leads to the following algorithm for testing primality:
Given a number $n$, pick a
#idx("random-number generator", sub: "in primality testing")
random number $a < n$ and compute the
remainder of $a^(n)$ modulo
$n$. If the result is not equal to
$a$, then $n$ is
certainly not prime. If it is $a$, then chances
are good that $n$ is prime. Now pick another
random number $a$ and test it with the same
method. If it also satisfies the equation, then we can be even more
confident that $n$ is prime. By trying more and
more values of $a$, we can increase our
confidence in the result. This algorithm is known as the Fermat test.

To implement the Fermat test, we need a
function
that computes the
#idx("exponentiation", sub: "modulo n")
exponential of a number modulo another number:
#idx("expmod", decl: true)
#snippet(```python
def expmod(base, exp, m):
    return (1 if exp == 0
            else square(expmod(base, exp // 2, m)) % m if is_even(exp)
            else (base * expmod(base, exp - 1, m)) % m)
```)

This is very similar to the
#py("fast_expt")
function
of section @sec:exponentiation. It uses successive
squaring, so that the number of steps grows logarithmically with the
exponent.#footnote[The reduction steps in the cases where the exponent
$e$ is greater than 1 are based on the fact that,
for any integers $x$,
$y$, and $m$, we can
find the remainder of $x$ times
$y$ modulo $m$ by
computing separately the remainders of $x$ modulo
$m$ and $y$ modulo
$m$, multiplying these, and then taking the
remainder of the result modulo $m$. For
instance, in the case where $e$ is even, we
compute the remainder of $b^(e/2)$ modulo
$m$, square this, and take the remainder modulo
$m$. This technique is useful because it means
we can perform our computation without ever having to deal with numbers much
larger than $m$. (Compare
exercise @ex:Alyssas-expmod.)]

The Fermat test is performed by choosing at random a number
$a$ between 1 and
$n-1$ inclusive and checking whether the remainder
modulo $n$ of the
$n$th power of $a$ is
equal to $a$. The random number
$a$ is chosen using the
primitive function #py("random_random"),

#idx("randomrandom (primitive function)") which returns a nonnegative number less than 1. Hence, to obtain a random number between 1 and $n-1$, we multiply the return value of #py("random_random") by $n-1$, round down the result with the primitive function #idx("mathfloor (primitive function)") #py("math_floor"), and add 1:

#idx("fermattest", decl: true)
#snippet(```python
def fermat_test(n):
    def try_it(a):
        return expmod(a, n, n) == a
    return try_it(1 + math_floor(random_random() * (n - 1)))
```)

The following
function
runs the test a given number of times, as specified by a parameter. Its
value is true if the test succeeds every time, and false otherwise.
#idx("fastisprime", decl: true)
#snippet(```python
def fast_is_prime(n, times):
    return (True if times == 0
            else fast_is_prime(n, times - 1) if fermat_test(n)
            else False)
```)

#subheading([Probabilistic methods])

#idx("probabilistic algorithm")
#idx("algorithm", sub: "probabilistic")

The Fermat test differs in character from most familiar algorithms, in which
one computes an answer that is guaranteed to be correct. Here, the answer
obtained is only probably correct. More precisely, if
$n$ ever fails the Fermat test, we can be certain
that $n$ is not prime. But the fact that
$n$ passes the test, while an extremely strong
indication, is still not a guarantee that $n$ is
prime. What we would like to say is that for any number
$n$, if we perform the test enough times and find
that $n$ always passes the test, then the
probability of error in our primality test can be made as small as we like.

Unfortunately, this assertion is not quite correct. There do exist numbers
that fool the Fermat test: numbers $n$ that are
not prime and yet have the property that $a^(n)$ is
congruent to $a$ modulo
$n$ for all integers
$a < n$. Such numbers are extremely rare, so
the Fermat test is quite reliable in practice.#footnote[Numbers that fool the
Fermat test are called
#idx("Carmichael numbers")
#emph[Carmichael numbers], and little is known
about them other than that they are extremely rare. There are 255
Carmichael numbers below 100,000,000. The smallest few are 561, 1105,
1729, 2465, 2821, and 6601. In testing primality of very large
numbers chosen at random, the chance of stumbling upon a value that
fools the Fermat test is less than the chance that
#idx("cosmic radiation")
cosmic radiation will cause the computer to make an error in carrying out a
"correct" algorithm. Considering an algorithm to be inadequate
for the first reason but not for the second illustrates the difference
between
#idx("engineering vs. mathematics")
#idx("mathematics", sub: "engineering vs.")
mathematics and engineering.]<foot:carmichaelfn>
There are variations of the Fermat test that cannot be fooled. In these
tests, as with the Fermat method, one tests the primality of an integer
$n$ by choosing a random integer
$a < n$ and checking some condition that
depends upon $n$ and
$a$. (See
exercise @ex:miller-rabin for an example of such a test.)
On the other hand, in contrast to the Fermat test, one can prove that, for
any $n$, the condition does not hold for most of
the integers $a < n$ unless
$n$ is prime. Thus, if
$n$ passes the test for some random choice
of $a$, the chances are better than even
that $n$ is prime. If
$n$ passes the test for two random choices of
$a$, the chances are better than 3 out of 4 that
$n$ is prime. By running the test with more and
more randomly chosen values of $a$ we can make
the probability of error as small as we like.
#idx("Fermat test for primality")
#idx("prime number(s)", sub: "Fermat test for")

The existence of tests for which one can prove that the chance of error
becomes arbitrarily small has sparked interest in algorithms of this type,
which have come to be known as #emph[probabilistic algorithms]. There is
#idx("probabilistic algorithm")
#idx("algorithm", sub: "probabilistic")
#idx("prime number(s)")
a great deal of research activity in this area, and probabilistic algorithms
have been fruitfully applied to many fields.#footnote[One of the most
striking applications of
probabilistic prime testing has been to the field of
#idx("cryptography")
cryptography.
Although it is computationally infeasible to factor an arbitrary 300-digit number as of this writing (2021), the primality of such a number can be checked in a few seconds with the Fermat test.
This fact forms the basis of a technique for constructing
"unbreakable codes" suggested by
#idx("Rivest, Ronald L.")
Rivest,
#idx("Shamir, Adi")
Shamir, and
#idx("Adleman, Leonard")
Adleman (1977). The resulting
#idx("RSA algorithm")
#emph[RSA algorithm] has become a widely used technique for enhancing the
security of electronic communications. Because of this and related
developments, the study of
#idx("prime number(s)", sub: "cryptography and")
prime numbers, once considered the epitome of a topic in "pure"
mathematics to be studied only for its own sake, now turns out to have
important practical applications to cryptography, electronic funds transfer,
and information retrieval.]

#exercise(label-name: <ex:use-smallest-divisor>, [
Use the
#py("smallest_divisor")
function
to find the smallest divisor of each of the following numbers: 199, 1999,
19999\.
])

#exercise(label-name: <ex:search-for-primes>, [
Assume a primitive function #idx("gettime (primitive function)") #py("get_time") of no arguments that returns the number of milliseconds that have passed since 00:00:00 UTC on Thursday, 1 January, 1970.#footnote[This date is called the #idx("UNIX", sub: "epoch") #emph[UNIX epoch] and is part of the specification of functions that deal with time in the UNIX$""^(upright("TM"))$ operating system.] The following #py("timed_prime_test") function,
when called with an integer $n$, prints
$n$ and checks to see if
$n$ is prime. If $n$
is prime, the
function
prints three asterisks#footnote[The primitive function #py("print") prints its argument. Here #py("\"") #py("***") #py("\"") is a #emph[string], a sequence of characters that we pass as argument to the #py("print") function. Section @sec:strings introduces strings more thoroughly.] followed by the amount of
time used in performing the test.

#idx("print (primitive function)")#idx("timedprimetest")
#snippet(```python
def timed_prime_test(n):
    print(n)
    return start_prime_test(n, get_time())

def start_prime_test(n, start_time):
    return (report_prime(get_time() - start_time) if is_prime(n)
            else False)

def report_prime(elapsed_time):
    print(" *** ")
    print(elapsed_time)
    return True
```)

Using this
function,
write a
function
#py("search_for_primes")
that checks the primality of consecutive odd integers in a specified range.
Use your
function
to find the three smallest primes larger than 1000; larger than 10,000;
larger than 100,000; larger than 1,000,000. Note the time needed to test
each prime. Since the testing algorithm has order of growth of
$Theta (sqrt(n))$, you should expect that testing
for primes around 10,000 should take about
$sqrt(10)$ times as long as testing for primes
around 1000. Do your timing data bear this out? How well do the data for
100,000 and 1,000,000 support the $sqrt(n)$
prediction? Is your result compatible with the notion that programs on
your machine run in time proportional to the number of steps required for
the computation?
])

#exercise(label-name: <ex:better-smallest-divisor>, [
The
#idx("smallestdivisor", sub: "more efficient version")
#py("smallest_divisor")
function
shown at the start of this section does lots of needless testing: After it
checks to see if the number is divisible by 2 there is no point in checking
to see if it is divisible by any larger even numbers. This suggests that
the values used for
#py("test_divisor")
should not be 2, 3, 4, 5, 6, … but rather 2, 3, 5, 7, 9,
…. To implement this change,
define a function
#py("next_int") that returns 3 if its input is equal to 2
and otherwise returns its input plus 2\. Modify the
#py("smallest_divisor")
function
to use
#py("next_int(test_divisor)")
instead of
#py("test_divisor + 1").
With
#py("timed_prime_test")
incorporating this modified version of
#py("smallest_divisor"),
run the test for each of the 12 primes found in
exercise @ex:search-for-primes.
Since this modification halves the number of test steps, you should expect
it to run about twice as fast. Is this expectation confirmed? If not, what
is the observed ratio of the speeds of the two algorithms, and how do you
explain the fact that it is different from 2?
])

#exercise(label-name: <ex:mod-timed-prime-test>, [
Modify the
#py("timed_prime_test")
function
of exercise @ex:search-for-primes to use
#py("fast_is_prime")
(the Fermat method), and test each of the 12 primes you found in that
exercise. Since the Fermat test has
$Theta ( log n)$ growth, how would you expect
the time to test primes near 1,000,000 to compare with the time needed to
test primes near 1000? Do your data bear this out? Can you explain any
discrepancy you find?
])

#exercise(label-name: <ex:Alyssas-expmod>, [
Alyssa P. Hacker complains that we went to a lot of extra work in writing
#py("expmod"). After all, she says, since we already
know how to compute exponentials, we could have simply written
#idx("expmod", decl: true)
#snippet(```python
def expmod(base, exp, m):
    return fast_expt(base, exp) % m
```)

Is she correct?
Would this
function
serve as well for our fast prime tester? Explain.
])

#exercise(label-name: <ex:louis-fast-prime>, [
Louis Reasoner is having great difficulty doing
exercise @ex:mod-timed-prime-test.
His
#py("fast_is_prime")
test seems to run more slowly than his
#py("is_prime")
test. Louis calls his friend Eva Lu Ator over to help. When they examine
Louis's code, they find that he has rewritten the
#py("expmod")
function
to use an explicit multiplication, rather than calling
#py("square"):
#idx("expmod", decl: true)
#snippet(```python
def expmod(base, exp, m):
    return (1 if exp == 0
            else (expmod(base, exp // 2, m)
                  * expmod(base, exp // 2, m)) % m if is_even(exp)
            else (base * expmod(base, exp - 1, m)) % m)
```)

"I don't see what difference that could make,"
says Louis. "I do." says Eva. "By writing the function like that, you have transformed the $Theta ( log n)$ process into a $Theta (n)$ process." Explain.
])

#exercise(label-name: <ex:1_27>, [
Demonstrate that the
#idx("Carmichael numbers")
Carmichael numbers listed in
footnote @foot:carmichaelfn really do fool the Fermat
test. That is, write a
function
that takes an integer $n$ and tests whether
$a^(n)$ is congruent to
$a$ modulo $n$ for
every $a < n$, and try your
function
on the given Carmichael numbers.
])

#exercise(label-name: <ex:miller-rabin>, [
One variant of the Fermat test that cannot be fooled is called the
#idx("prime number(s)", sub: "Miller–Rabin test for")
#idx("Fermat test for primality", sub: "variant of")
#idx("Miller–Rabin test for primality")
#idx("Miller, Gary L.")
#idx("Rabin, Michael O.")
#emph[Miller–Rabin test] (Miller 1976;
Rabin 1980). This starts from
an alternate form of
#idx("Fermat's Little Theorem", sub: "alternate form", sort: "Fermats")
Fermat's Little Theorem, which states that if
$n$ is a prime number and
$a$ is any positive integer less than
$n$, then $a$ raised
to the $(n-1)$st power is congruent to 1
modulo $n$. To test the primality of a
number $n$ by the Miller–Rabin test, we pick a
random number $a < n$ and raise
$a$ to the $(n-1)$st
power modulo $n$ using the
#py("expmod")
function.
However, whenever we perform the squaring step in
#py("expmod"), we check to see if we have discovered a
"nontrivial square root of 1 modulo $n$,"
that is, a number not equal to 1 or $n-1$ whose
square is equal to 1 modulo $n$. It is
possible to prove that if such a nontrivial square root of 1 exists, then
$n$ is not prime. It is also possible to prove
that if $n$ is an odd number that is not prime,
then, for at least half the numbers $a < n$,
computing $a^(n-1)$ in this way will reveal a
nontrivial square root of 1 modulo $n$.
(This is why the Miller–Rabin test cannot be fooled.) Modify the
#py("expmod")
function
to signal if it discovers a nontrivial square root of 1, and use this to
implement the Miller–Rabin test with a
function
analogous to
#py("fermat_test").
Check your
function
by testing various known primes and non-primes. Hint: One convenient way to
make #py("expmod") signal is to have it return 0\.
])

#idx("prime number(s)", sub: "testing for")
