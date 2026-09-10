// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Infinite Streams], label-name: <sec:infinite-streams>)

#idx("infinite stream(s)")

We have seen how to support the illusion of manipulating streams
as complete entities even though, in actuality, we compute only
as much of the stream as we need to access. We can exploit this
technique to represent sequences efficiently as streams, even if the
sequences are very long. What is more striking, we can use streams to
represent sequences that are infinitely long. For instance, consider
the following definition of the stream of positive integers:

#idx("integersstartingfrom", decl: true)
#snippet(```python
def integers_starting_from(n):
    return pair(n, lambda: integers_starting_from(n + 1))
```)

#idx("integers (infinite stream)", decl: true)
#snippet(```python
integers = integers_starting_from(1)
```)

This makes sense because #py("integers") will be a
pair whose
#py("head")
is 1 and whose
#py("tail")
is a promise to produce the integers beginning with 2. This is an infinitely
long stream, but in any given time we can examine only a finite portion of
it. Thus, our programs will never know that the entire infinite stream is
not there.

Using #py("integers") we can define other infinite
streams, such as the stream of integers that are not divisible by 7:

#idx("isdivisible", decl: true)
#snippet(```python
def is_divisible(x, y):
    return x % y == 0
```)

#snippet(```python
no_sevens = stream_filter(lambda x: not is_divisible(x, 7),
                          integers)
```)

Then we can find integers not divisible by 7 simply by accessing
elements of this stream:

#snippet(```python
print(stream_ref(no_sevens, 100))
```)

#output(```python
print(stream_ref(no_sevens, 100))
```)

In analogy with #py("integers"), we can define the
infinite stream of Fibonacci numbers:

#idx("fibs (infinite stream)", decl: true)
#snippet(```python
def fibgen(a, b):
    return pair(a, lambda: fibgen(b, a + b))

fibs = fibgen(0, 1)
```)

The constant #py("fibs")
is a pair whose
#py("head")
is 0 and whose
#py("tail")
is a promise to evaluate
#py("fibgen(1, 1)").
When we evaluate this delayed
#py("fibgen(1, 1)"),
it will produce a pair whose
#py("head")
is 1 and whose
#py("tail")
is a promise to evaluate
#py("fibgen(1, 2)"),
and so on.

For a look at a more exciting infinite stream, we can generalize the
#py("no_sevens")
example to construct the infinite stream of prime
numbers, using a
#emph[sieving] method.#footnote[This method is reminiscent of the ancient
#idx("prime number(s)", sub: "Eratosthenes's sieve for")
#idx("sieve of Eratosthenes")
#emph[sieve of Eratosthenes], named for
#idx("Eratosthenes")
Eratosthenes, a third-century BCE Alexandrian Greek mathematician.
His sieve finds the primes by repeatedly crossing off the multiples of each
prime in turn; although ancient, it has formed the basis
for special-purpose hardware "sieves" that, until the 1970s,
were the
most powerful tools in existence for locating large primes. Since then,
however, these methods have been superseded by outgrowths of the
#idx("probabilistic algorithm")
probabilistic techniques discussed in
section @sec:primality. But the stream-based method
given here is not actually Eratosthenes's sieve: as Melissa E.
O'Neill shows in O'Neill 2009, it tests
each candidate for divisibility by the primes found so far—a form of
#emph[trial division]—rather than crossing off multiples, and is
substantially less efficient than the genuine sieve.]
We start with the integers beginning with 2, which is the first prime.
To get the rest of the primes, we start by filtering the multiples of
2 from the rest of the integers. This leaves a stream beginning with
3, which is the next prime. Now we filter the multiples of 3 from the
rest of this stream. This leaves a stream beginning with 5, which is
the next prime, and so on. In other words, we construct the primes by
a sieving process, described as follows: To sieve a stream
S,
form a stream whose first element is the first element of
S and
the rest of which is obtained by filtering all multiples of the
first element of S out of the rest
of S and sieving the result. This
process is readily described in terms of stream operations:

#idx("primes (infinite stream)", decl: true)#idx("sieve of Eratosthenes", sub: "sieve", decl: true)
#snippet(```python
def sieve(stream):
    return pair(head(stream),
                lambda: sieve(stream_filter(
                                  lambda x: not is_divisible(x, head(stream)),
                                  stream_tail(stream))))

primes = sieve(integers_starting_from(2))
```)

Now to find a particular prime we need only ask for it:

#snippet(```python
print(stream_ref(primes, 50))
```)

#output(```python
print(stream_ref(primes, 50))
```)

It is interesting to contemplate the signal-processing system set up
by #py("sieve"), shown in the
#idx("Henderson, Peter", sub: "Henderson diagram")
"Henderson diagram" in
figure @fig:primesieve.#footnote[We have named these
figures after
#idx("Henderson, Peter")
Peter Henderson, who was the first person to show us diagrams of this sort
as a way of thinking about stream processing.] The input stream feeds into an
"un#py("pair")er"
that separates the first element of the stream from the rest of the stream.
The first element is used to construct a divisibility filter, through
which the rest is passed, and the output of the filter is fed to
another sieve box. Then the original first element is
adjoined to the output of the internal sieve to form the output stream.
Thus, not only is the stream infinite, but the signal processor is also
infinite, because the sieve contains a sieve within it.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-35.svg", width: 57%), caption: [The prime sieve viewed as a signal-processing system. Each solid line represents a stream of values being transmitted. The dashed line from the #py("head") to the #py("pair") and the #py("filter") indicates that this is a single value rather than a stream.], label-name: <fig:primesieve>)

#subheading([Defining streams implicitly])

#idx("stream(s)", sub: "implicit definition")

The #py("integers") and
#py("fibs") streams above were defined by specifying
"generating"
functions
that explicitly compute the stream elements one by one. An alternative way
to specify streams is to take advantage of delayed evaluation to define
streams implicitly. For example, the following
statement
defines the
stream #py("ones") to be an infinite stream of ones:

#idx("ones (infinite stream)", decl: true)
#snippet(```python
ones = pair(1, lambda: ones)
```)

This works much like the declaration of a recursive
function:
#py("ones") is a pair whose
#py("head")
is 1 and whose
#py("tail")
is a promise to evaluate #py("ones"). Evaluating the
#py("tail")
gives us again a 1 and a promise to evaluate
#py("ones"), and so on.

We can do more interesting things by manipulating streams with
operations such as
#py("add_streams"),
which produces the elementwise sum of two given streams:#footnote[This uses the function
#py("stream_map_2")
from exercise @ex:combine-streams.]

#idx("addstreams", decl: true)
#snippet(```python
def add_streams(s1, s2):
    return stream_map_2(lambda x1, x2: x1 + x2, s1, s2)
```)

Now we can define the integers as follows:

#idx("integers (infinite stream)", sub: "implicit definition", decl: true)
#snippet(```python
integers = pair(1, lambda: add_streams(ones, integers))
```)

This defines #py("integers") to be a stream whose
first element is 1 and the rest of which is the sum of
#py("ones") and #py("integers").
Thus, the second element of #py("integers") is 1 plus
the first element of #py("integers"), or 2; the third
element of #py("integers") is 1 plus the second
element of #py("integers"), or 3; and so on. This
definition works because, at any point, enough of the
#py("integers") stream has been generated so that we
can feed it back into the definition to produce the next integer.

We can define the Fibonacci numbers in the same style:

#idx("fibs (infinite stream)", sub: "implicit definition", decl: true)
#snippet(```python
fibs = pair(0,
            lambda: pair(1,
                         lambda: add_streams(stream_tail(fibs),
                                             fibs)))
```)

This definition says that #py("fibs") is a stream
beginning with 0 and 1, such that the rest of the stream can be generated
by adding #py("fibs") to itself shifted by one place:

$ mat(delim: #none, , , 1, 1, 2, 3, 5, 8, 13, 21, dots.h, =, mono("stream")mono("_")mono("tail(fibs)"); , , 0, 1, 1, 2, 3, 5, 8, 13, dots.h, =, mono("fibs"); 0, 1, 1, 2, 3, 5, 8, 13, 21, 34, dots.h, =, mono("fibs")) $

The function #py("scale_stream") is also useful
in formulating such stream definitions. This multiplies each item in a
stream by a given constant:

#idx("scalestream", decl: true)
#snippet(```python
def scale_stream(stream, factor):
    return stream_map(lambda x: x * factor,
                      stream)
```)

For example,

#snippet(```python
double = pair(1, lambda: scale_stream(double, 2))
```)

produces the stream of powers of 2:
$1, 2, 4, 8, 16, 32,$ ….

An alternate definition of the stream of primes can be given by
starting with the integers and filtering them by testing for
primality. We will need the first prime, 2, to get started:

#idx("primes (infinite stream)", sub: "implicit definition", decl: true)
#snippet(```python
primes = pair(2,
              lambda: stream_filter(is_prime,
                                    integers_starting_from(3)))
```)

This definition is not so straightforward as it appears, because we will
test whether a number $n$ is prime by checking
whether $n$ is divisible by a prime (not by just
any integer) less than or equal to $sqrt(n)$:

#idx("isprime", decl: true)
#snippet(```python
def is_prime(n):
    def iter(ps):
        return (True
                if square(head(ps)) > n
                else False
                if is_divisible(n, head(ps))
                else iter(stream_tail(ps)))
    return iter(primes)
```)

This is a recursive definition, since #py("primes")
is defined in terms of the
#py("is_prime")
predicate, which itself uses the #py("primes") stream.
The reason this
function
works is that, at any point, enough of the
#py("primes") stream has been generated to test the
primality of the numbers we need to check next. That is, for every
$n$ we test for primality, either
$n$ is not prime (in which case there is a prime
already generated that divides it) or $n$ is
prime (in which case there is a prime already generated—i.e., a
prime less than $n$—that is greater than
$sqrt(n)$).#footnote[This last point is very
subtle and relies on the fact that $p_(n+1) lt.eq p_(n)^(2)$. (Here, $p_(k)$ denotes the
$k$th prime.) Estimates such as these are very
difficult to establish. The ancient proof by
#idx("Euclid's proof of infinite number of primes")
Euclid that there are an infinite number of primes shows that
$p_(n+1) lt.eq p_(1) p_(2) thin dots.c thin thin p_(n) +1$,
and no substantially better result was proved until 1851, when the Russian
mathematician
#idx("Chebyshev, Pafnutii L'vovich")
P. L. Chebyshev established
that $p_(n+1) lt.eq 2p_(n)$ for all
$n$. This result, originally conjectured in
1845, is known as
#idx("Bertrand's Hypothesis")
#emph[Bertrand's hypothesis]. A proof can be
found in section 22.3 of
#idx("Hardy, Godfrey Harold")
#idx("Wright, E. M.")
Hardy and Wright 1960.]
#idx("stream(s)", sub: "implicit definition")

#exercise(label-name: <ex:without_running>, [
Without running the program, describe the elements of the
stream defined by

#snippet(```python
s = pair(1, lambda: add_streams(s, s))
```)
])

#exercise(label-name: <ex:element_wise_product>, [
Define a
function
#idx("mulstreams")
#idx("infinite stream(s)", sub: "of factorials")
#idx("factorial", sub: "infinite stream")
#py("mul_streams"),
analogous to
#py("add_streams"),
that produces the elementwise product of its two input streams. Use this
together with the stream of #py("integers") to
complete the following definition of the stream whose
$n$th element (counting from 0) is
$n+1$ factorial:

#syntax("
factorials = pair(1, lambda: mul_streams(", metaphrase[??], ", ", metaphrase[??], "))
      ")
])

#exercise(label-name: <ex:partial-sums>, [
Define a
function
#idx("partialsums")
#py("partial_sums")
that takes as argument a stream $S$ and returns
the stream whose elements are
$S_(0), S_(0)+S_(1), S_(0)+S_(1)+S_(2),$ ….
For example,
#py("partial_sums(integers)")
should be the stream $1, 3, 6, 10, 15, dots.h$.
])

#exercise(label-name: <ex:merge>, [
A famous problem, first raised by
#idx("Hamming, Richard Wesley")
R. Hamming, is to enumerate, in ascending order with no repetitions, all
positive integers with no prime factors other than 2, 3, or 5. One obvious
way to do this is to simply test each integer in turn to see whether it has
any factors other than 2, 3, and 5. But this is very inefficient, since, as
the integers get larger, fewer and fewer of them fit the requirement. As
an alternative, let us call the required stream of numbers
#py("S") and notice the following facts about it.

- #py("S") begins with 1.
- The elements of #py("scale_stream(S, 2)") are also elements of #py("S").
- The same is true for #py("scale_stream(S, 3)") and #py("scale_stream(S, 5)").
- These are all the elements of #py("S").

Now all we have to do is combine elements from these sources. For this we
define a
function
#idx("infinite stream(s)", sub: "merging")
#py("merge") that combines two ordered
streams into one ordered result stream, eliminating repetitions:

#idx("merge", decl: true)
#snippet(```python
def merge(s1, s2):
    if is_none(s1):
        return s2
    elif is_none(s2):
        return s1
    else:
        s1head = head(s1)
        s2head = head(s2)
        return (pair(s1head, lambda: merge(stream_tail(s1), s2))
                if s1head < s2head
                else pair(s2head, lambda: merge(s1, stream_tail(s2)))
                if s1head > s2head
                else pair(s1head, lambda: merge(stream_tail(s1), stream_tail(s2))))
```)

Then the required stream may be constructed with
#py("merge"), as follows:

#syntax("
S = pair(1, lambda: merge(", metaphrase[??], ", ", metaphrase[??], "))
      ")

Fill in the missing expressions in the places marked
#metaphrase[??] above.
])

#exercise(label-name: <ex:fib-stream-efficiency>, [
How many additions are performed when we compute the $n$th Fibonacci number using the declaration of #py("fibs") based on the #py("add_streams") function? Show that this number is exponentially greater than the number of additions performed if #py("add_streams") had used the function #py("stream_map_2_optimized") described in exercise @ex:combine-streams, and if we had declared #py("fibs") as follows: #snippet(```python fibs = pair(0, memo(lambda: pair(1, memo(lambda: add_streams(stream_tail(fibs), fibs))))) ```)
])

#exercise(label-name: <ex:quotient>, [
Give an interpretation of the stream computed by the
function

#snippet(```python
def expand(num, den, radix):
    return pair(math_trunc((num * radix) / den),
                lambda: expand((num * radix) % den, den, radix))
```)

where #idx("mathtrunc (primitive function)") #py("math_trunc") discards the fractional part of its argument, here the remainder of the division.
What are the successive elements produced by
#py("expand(1, 7, 10)")?
What is produced by
#py("expand(3, 8, 10)")?
])

#exercise(label-name: <ex:powerseries>, [
In section @sec:symbolic-algebra we saw how to implement a
polynomial arithmetic system representing polynomials as lists of
terms. In a similar way, we can work with
#idx("power series, as stream")
#idx("infinite stream(s)", sub: "representing power series")
#idx("eˣ, power series for", sort: "e")
#idx("cosine", sub: "power series for")
#idx("sine", sub: "power series for")
#emph[power series], such as

$ mat(delim: #none, e^(x), =, 1+x+frac(x^(2), 2)+frac(x^(3), 3 dot.op 2) +frac(x^(4), 4 dot.op 3 dot.op 2)+ dots.c ,; cos x, =, 1-frac(x^(2), 2)+frac(x^(4), 4 dot.op 3 dot.op 2)- dots.c ,; sin x, =, x-frac(x^(3), 3 dot.op 2) +frac(x^(5), 5 dot.op 4 dot.op 3 dot.op 2)- dots.c ,) $

represented as infinite streams.
We will represent the series
$a_(0) + a_(1) x + a_(2) x^(2) + a_(3) x^(3) + dots.c$
as the stream whose elements are the coefficients
$a_(0), a_(1), a_(2), a_(3),$ ….

+ The #idx("integral", sub: "of a power series") #idx("power series, as stream", sub: "integrating") integral of the series $a_(0) + a_(1) x + a_(2) x^(2) + a_(3) x^(3) + dots.c$ is the series $ c + a_(0) x + frac(1, 2)a_(1) x^(2) + frac(1, 3)a_(2) x^(3) + frac(1, 4)a_(3) x^(4) + dots.c $ where $c$ is any constant. Define a function #idx("integrateseries") #py("integrate_series") that takes as input a stream $a_(0), a_(1), a_(2), dots.h$ representing a power series and returns the stream $a_(0), frac(1, 2)a_(1), frac(1, 3)a_(2), dots.h$ of coefficients of the nonconstant terms of the integral of the series. (Since the result has no constant term, it doesn't represent a power series; when we use #py("integrate_series"), we will use #py("pair") to adjoin the appropriate constant to the beginning of the stream.)
+ The function $x arrow.r.bar e^(x)$ is its own derivative. This implies that $e^(x)$ and the integral of $e^(x)$ are the same series, except for the constant term, which is $e^(0) = 1$. Accordingly, we can generate the series for $e^(x)$ as #snippet(```python exp_series = pair(1, lambda: integrate_series(exp_series)) ```) Show how to generate the series for sine and cosine, starting from the facts that the derivative of sine is cosine and the derivative of cosine is the negative of sine: #syntax(" cosine_series = pair(1, ", metaphrase[??], ") sine_series = pair(0, ", metaphrase[??], ") ")
])

#exercise(label-name: <ex:mul-series>, [
With
#idx("power series, as stream", sub: "adding")
#idx("power series, as stream", sub: "multiplying")
#idx("arithmetic", sub: "on power series")
#idx("mulseries")
power series represented as streams of coefficients as in
exercise @ex:powerseries, adding series is implemented
by
#py("add_streams").
Complete the declaration of
the following
function
for multiplying series:

#syntax("
def mul_series(s1, s2):
    pair(", metaphrase[??], ", lambda: add_streams(", metaphrase[??], ", ", metaphrase[??], "))
      ")

You can test your
function
by verifying that $sin^(2) x + cos^(2) x = 1$,
using the series from exercise @ex:powerseries.
])

#exercise(label-name: <ex:invert-unit-series>, [
Let $S$ be a power series
(exercise @ex:powerseries)
whose constant term is 1. Suppose we want to find the power series
$1/S$, that is, the series
$X$ such that
$S dot.op X= 1$.
Write $S=1+S_(R)$ where
$S_(R)$ is the part of
$S$ after the constant term. Then we can solve
for $X$ as follows:

$ mat(delim: #none, S dot.op X, =, 1; (1+S_(R)) dot.op X, =, 1; X + S_(R) dot.op X, =, 1; X, =, 1 - S_(R) dot.op X) $

In other words, $X$ is the power series whose
constant term is 1 and whose higher-order terms are given by the negative of
$S_(R)$ times $X$.
Use this idea to write a
function
#py("invert_unit_series")
that computes $1/S$ for a power series
$S$ with constant term 1. You will need to use
#py("mul_series")
from exercise @ex:mul-series.
])

#exercise(label-name: <ex:diving_power_series>, [
Use the results of exercises @ex:mul-series
and @ex:invert-unit-series
to define a
function
#idx("power series, as stream", sub: "dividing")
#idx("arithmetic", sub: "on power series")
#idx("divseries")
#py("div_series")
that divides two power series.
The function #py("div_series")
should work for any two series, provided that the denominator series begins
with a nonzero constant term. (If the denominator has a zero constant term,
then
#py("div_series")
should signal an error.) Show how to use
#py("div_series")
together with the result of exercise @ex:powerseries
to generate the power series for
#idx("tangent", sub: "power series for")
tangent.
])

#idx("infinite stream(s)")
