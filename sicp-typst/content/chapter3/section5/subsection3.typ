// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Exploiting the Stream Paradigm], label-name: <sec:exploiting-streams>)

Streams with delayed evaluation can be a powerful modeling tool,
providing many of the benefits of local state and assignment.
Moreover, they avoid some of the theoretical tangles that accompany
the introduction of assignment into a programming language.

The stream approach can be illuminating because it allows us to build
systems with different
#idx("modularity", sub: "streams and")
module boundaries than systems organized around
assignment to state variables. For example, we can think of an entire
time series (or signal) as a focus of interest, rather than the values
of the state variables at individual moments. This makes it
convenient to combine and compare components of state from different
moments.

#subheading([Formulating iterations as stream processes])

#idx("iterative process", sub: "as a stream process")

In section @sec:recursion-and-iteration, we introduced
iterative processes, which proceed by updating state variables. We know now
that we can represent state as a "timeless" stream of values
rather than as a set of variables to be updated. Let's adopt this
perspective in revisiting the square-root
function
from section @sec:sqrt. Recall that the idea is to
generate a sequence of better and better guesses for the square root of
$x$ by applying over and over again the
function
that improves guesses:

#snippet(```python
def sqrt_improve(guess, x):
    return average(guess, x / guess)
```)

In our original
#idx("square root", sub: "stream of approximations")
#py("sqrt")
function,
we made these guesses be the successive values of a state variable. Instead
we can generate the infinite stream of guesses, starting with an initial
guess of 1:
#idx("sqrtstream", decl: true)
#snippet(```python
def sqrt_stream(x):
    return pair(1, lambda: stream_map(lambda guess: sqrt_improve(guess, x),
                                      sqrt_stream(x)))
```)

#snippet(```python
print(display_stream(sqrt_stream(2)))
```)

#output(```python
print(display_stream(sqrt_stream(2)))
```)

We can generate more and more terms of the stream to get better and
better guesses. If we like, we can write a
function
that keeps generating terms until the answer is good enough.
(See exercise @ex:stream-limit.)

Another iteration that we can treat in the same way is to generate an
approximation to
#idx("π (pi)", sub: "Leibniz's series for", sort: "pi")
#idx("Leibniz, Baron Gottfried Wilhelm von", sub: "series for π")
#idx("π (pi)", sub: "stream of approximations", sort: "pi")
#idx("series, summation of", sub: "with streams")
#idx("summation of a series", sub: "with streams")
#idx("infinite stream(s)", sub: "to sum a series")
$pi$, based upon the
alternating series that we saw in
section @sec:procedures-as-parameters:

$ mat(delim: #none, frac( pi , 4), =, 1-frac(1, 3)+frac(1, 5)-frac(1, 7)+ dots.c) $

We first generate the stream of summands of the series (the reciprocals
of the odd integers, with alternating signs). Then we take the stream
of sums of more and more terms (using the
#py("partial_sums") function
of exercise @ex:partial-sums) and scale the result by 4:
#idx("pistream", decl: true)
#snippet(```python
def pi_summands(n):
    return pair(1 / n, lambda: stream_map(lambda x: -x, pi_summands(n + 2)))

pi_stream = scale_stream(partial_sums(pi_summands(1)), 4)
```)

#snippet(```python
print(display_stream(pi_stream))
```)

#output(```python
print(display_stream(pi_stream))
```)

This gives us a stream of better and better approximations to
$pi$, although the approximations converge
rather slowly. Eight terms of the sequence bound the value of
$pi$ between 3.284 and 3.017.

So far, our use of the stream of states approach is not much different
from updating state variables. But streams give us an opportunity to do
some interesting tricks. For example, we can transform a stream with a
#idx("series, summation of", sub: "accelerating sequence of approximations")
#idx("sequence accelerator")
#emph[sequence accelerator] that converts a sequence of approximations to a
new sequence that converges to the same value as the original, only faster.

One such accelerator, due to the eighteenth-century Swiss mathematician
#idx("Euler, Leonhard", sub: "series accelerator")
Leonhard Euler, works well with sequences that are partial sums of
alternating series (series of terms with alternating signs). In
Euler's technique, if $S_(n)$ is the
$n$th term of the original sum sequence, then
the accelerated sequence has terms

$ S_(n+1) - frac((S_(n+1)-S_(n))^(2), S_(n-1)-2S_(n)+S_(n+1)) $

Thus, if the original sequence is represented as a stream of values,
the transformed sequence is given by
#idx("eulertransform", decl: true)
#syntax("
def euler_transform(s):
    s0 = stream_ref(s, 0)     # ", $S_(n-1)$, "
    s1 = stream_ref(s, 1)     # ", $S_(n)$, "
    s2 = stream_ref(s, 2)     # ", $S_(n+1)$, "
    return pair(s2 - square(s2 - s1) / (s0 + (-2) * s1 + s2),
                memo(lambda: euler_transform(stream_tail(s))))
      ")

Note that we make use of the memoization optimization of
section @sec:delayed-lists, because in the
following we will rely on repeated evaluation of the resulting stream.

We can demonstrate Euler acceleration with our sequence of
approximations to $pi$:

#snippet(```python
print(display_stream(euler_transform(pi_stream)))
```)

#output(```python
print(display_stream(euler_transform(pi_stream)))
```)

Even better, we can accelerate the accelerated sequence, and recursively
accelerate that, and so on. Namely, we create a stream of streams (a
structure we'll call a
#idx("tableau")
#emph[tableau]) in which each stream is the transform of the preceding one:
#idx("maketableau", decl: true)
#snippet(```python
def make_tableau(transform, s):
    return pair(s, lambda: make_tableau(transform, transform(s)))
```)

The tableau has the form

$ mat(delim: #none, s_(00), s_(01), s_(02), s_(03), s_(04), dots.h; , s_(10), s_(11), s_(12), s_(13), dots.h; , , s_(20), s_(21), s_(22), dots.h; , , , , dots.h, ) $

Finally, we form a sequence by taking the first term in each row of
the tableau:
#idx("acceleratedsequence", decl: true)
#snippet(```python
def accelerated_sequence(transform, s):
    return stream_map(head, make_tableau(transform, s))
```)

We can demonstrate this kind of "super-acceleration" of the
$pi$ sequence:

#snippet(```python
print(display_stream(accelerated_sequence(euler_transform, pi_stream)))
```)

#output(```python
print(display_stream(accelerated_sequence(euler_transform, pi_stream)))
```)

The result is impressive. Taking eight terms of the sequence yields the
correct value of $pi$ to 14 decimal places.
If we had used only the original $pi$ sequence,
we would need to compute on the order of $10^(13)$
terms (i.e., expanding the series far enough so that the individual terms
are less then $10^(-13)$) to get that much
accuracy!
#idx("π (pi)", sub: "stream of approximations", sort: "pi")

We could have implemented these acceleration techniques without using
streams. But the stream formulation is particularly elegant and convenient
because the entire sequence of states is available to us as a data structure
that can be manipulated with a uniform set of operations.

#exercise(label-name: <ex:stream-internal-def>, [
Louis Reasoner is not happy with the performance of the stream
produced by the
#py("sqrt_stream") function and
tries to optimize it using memoization:

#snippet(```python
def sqrt_stream_optimized(x):
    return pair(1,
                memo(lambda: stream_map(lambda guess:
                                        sqrt_improve(guess, x),
                                        sqrt_stream_optimized(x))))
```)

Alyssa P. Hacker instead proposes

#snippet(```python
def sqrt_stream_optimized_2(x):
    guesses = pair(1,
                   memo(lambda: stream_map(lambda guess:
                                           sqrt_improve(guess, x),
                                           guesses)))
    return guesses
```)

and claims that Louis's version is
considerably less efficient than hers, because it performs
redundant computation. Explain Alyssa's answer.
Would Alyssa's approach without memoization be more
efficient
than the original #py("sqrt_stream")?
])

#exercise(label-name: <ex:stream-limit>, [
Write a
#idx("streamlimit")
function #py("stream_limit")
that takes as arguments a stream
and a number (the tolerance). It should examine the stream until it
finds two successive elements that differ in absolute value by less
than the tolerance, and return the second of the two elements. Using
this, we could compute square roots up to a given tolerance by
#idx("sqrt", sub: "as stream limit", decl: true)
#snippet(```python
def sqrt(x, tolerance):
    return stream_limit(sqrt_stream(x), tolerance)
```)
])

#exercise(label-name: <ex:3_64>, [
Use the series

$ mat(delim: #none, ln 2, =, 1-frac(1, 2)+frac(1, 3)-frac(1, 4)+ dots.c) $

to compute three sequences of approximations to the natural logarithm of 2,
#idx("logarithm, approximating 2")
in the same way we did above for $pi$.
How rapidly do these sequences converge?
])

#idx("iterative process", sub: "as a stream process")

#subheading([Infinite streams of pairs])

#idx("pair(s)", sub: "infinite stream of")
#idx("infinite stream(s)", sub: "of pairs")
#idx("mapping", sub: "nested")

In section @sec:nested-mappings, we saw how the
sequence paradigm handles traditional nested loops as processes defined
on sequences of pairs. If we generalize this technique to infinite streams,
then we can write programs that are not easily represented as loops, because
the "looping" must range over an infinite set.

For example, suppose we want to generalize the
#idx("primesumpairs", sub: "infinite stream")
#py("prime_sum_pairs") function
of section @sec:nested-mappings to produce the stream
of pairs of #emph[all] integers $(i,j)$ with
$i lt.eq j$ such that
$i+j$
is prime. If
#py("int_pairs")
is the sequence of all pairs of integers $(i,j)$
with $i lt.eq j$, then our required stream is
simply#footnote[As in
section @sec:sequences-conventional-interfaces, we
represent a pair of integers as a list rather than a

pair.]

#snippet(```python
print(stream_filter(lambda pair: is_prime(head(pair) + head(tail(pair))),
              int_pairs))
```)

Our problem, then, is to produce the stream
#py("int_pairs").
More generally, suppose we have two streams
$S = (S_(i))$ and
$T = (T_(j))$,
and imagine the infinite rectangular array

$ mat(delim: #none, (S_(0),T_(0)), (S_(0),T_(1)), (S_(0), T_(2)), dots.h; (S_(1),T_(0)), (S_(1),T_(1)), (S_(1), T_(2)), dots.h; (S_(2),T_(0)), (S_(2),T_(1)), (S_(2), T_(2)), dots.h; dots.h, , , ) $

We wish to generate a stream that contains all the pairs in the array
that lie on or above the diagonal, i.e., the pairs

$ mat(delim: #none, (S_(0),T_(0)), (S_(0),T_(1)), (S_(0), T_(2)), dots.h; , (S_(1),T_(1)), (S_(1), T_(2)), dots.h; , , (S_(2), T_(2)), dots.h; , , , dots.h) $

(If we take both $S$ and
$T$ to be the stream of integers, then this
will be our desired stream
#py("int_pairs").)

Call the general stream of pairs
#py("pairs(S, T)"),
and consider it to be composed of three parts: the pair
$(S_(0),T_(0))$, the rest of the pairs in the first
row, and the remaining pairs:#footnote[See
exercise @ex:pairs-array for some insight into why we
chose this decomposition.]

$ mat(delim: #none, (S_(0),T_(0)), (S_(0),T_(1)), (S_(0), T_(2)), dots.h; , (S_(1),T_(1)), (S_(1), T_(2)), dots.h; , , (S_(2), T_(2)), dots.h; , , , dots.h) $

Observe that the third piece in this decomposition (pairs that are not in
the first row) is (recursively) the pairs formed from
#py("stream_tail(S)")
and
#py("stream_tail(T)").
Also note that the second piece (the rest of the first row) is

#snippet(```python
stream_map(lambda x: llist(head(s), x),
           stream_tail(t))
```)

Thus we can form our stream of pairs as follows:

#syntax("
def pairs(s, t):
    return pair(llist(head(s), head(t)),
                lambda: ", meta("combine-in-some-way"), "(
                            stream_map(lambda x: llist(head(s), x),
                                       stream_tail(t)),
                            pairs(stream_tail(s), stream_tail(t))))
      ")

In order to complete the
function,
we must choose some way to
#idx("infinite stream(s)", sub: "merging")
combine the two inner streams. One idea is to
use the stream analog of the #py("append")
function
from section @sec:sequences:

#idx("streamappend", decl: true)
#snippet(```python
def stream_append(s1, s2):
    return (s2
            if is_none(s1)
            else pair(head(s1),
                      lambda: stream_append(stream_tail(s1), s2)))
```)

This is unsuitable for infinite streams, however, because it takes all the
elements from the first stream before incorporating the second stream. In
particular, if we try to generate all pairs of positive integers using

#snippet(```python
print(pairs(integers, integers))
```)

our stream of results will first try to run through all pairs with the
first integer equal to 1, and hence will never produce pairs with any
other value of the first integer.

To handle infinite streams, we need to devise an order of combination
that ensures that every element will eventually be reached if we let
our program run long enough. An elegant way to accomplish this is
with the following #py("interleave")
function:#footnote[The
precise statement of the required property on the order of combination is
as follows: There should be a function $f$ of
two arguments such that the pair corresponding to
element $i$ of the first stream and
element $j$ of the second stream will
appear as element number $f(i,j)$ of the output
stream. The trick of using #py("interleave")
to accomplish this was shown to us by
#idx("Turner, David")
David Turner, who employed it in the language
#idx("KRC")
KRC (Turner 1981).]

#idx("interleave", decl: true)
#snippet(```python
def interleave(s1, s2):
    return (s2
            if is_none(s1)
            else pair(head(s1),
                      lambda: interleave(s2, stream_tail(s1))))
```)

Since #py("interleave") takes elements alternately
from the two streams, every element of the second stream will eventually
find its way into the interleaved stream, even if the first stream is
infinite.

We can thus generate the required stream of pairs as
#idx("pairs", decl: true)
#snippet(```python
def pairs(s, t):
    return pair(llist(head(s), head(t)),
                lambda: interleave(stream_map(lambda x: llist(head(s), x),
                                              stream_tail(t)),
                                   pairs(stream_tail(s),
                                         stream_tail(t))))
```)

#exercise(label-name: <ex:stream-pair-order>, [
Examine the stream
#py("pairs(integers, integers)").
Can you make any general comments about the order in which the pairs are
placed into the stream? For example, approximately how many pairs precede
the pair (1,100)? the pair (99,100)? the pair (100,100)? (If you can make
precise mathematical statements here, all the better. But feel free to give
more qualitative answers if you find yourself getting bogged down.)
])

#exercise(label-name: <ex:3_67>, [
Modify the #py("pairs")
function
so that
#py("pairs(integers, integers)")
will produce the stream of #emph[all] pairs of integers
$(i,j)$ (without the condition
$i lt.eq j$). Hint: You will need to
mix in an additional stream.
])

#exercise(label-name: <ex:pairs-array>, [
Louis Reasoner thinks that building a stream of pairs from three parts is
unnecessarily complicated. Instead of separating the pair
$(S_(0),T_(0))$ from the rest of the pairs in the
first row, he proposes to work with the whole first row, as follows:

#snippet(```python
def pairs(s, t):
    return interleave(stream_map(lambda x: llist(head(s), x),
                                 t),
                      pair(stream_tail(s), stream_tail(t)))
```)

Does this work? Consider what happens if we evaluate
#py("pairs(integers, integers)")
using Louis's definition of #py("pairs").
])

#exercise(label-name: <ex:stream-pythagorean-triples>, [
Write a
function
#py("triples") that takes three infinite streams,
$S$, $T$, and
$U$, and produces the stream of triples
$(S_(i),T_(j),U_(k))$ such that
$i lt.eq j lt.eq k$. Use
#py("triples") to generate the stream of all
#idx("Pythagorean triples", sub: "with streams")
Pythagorean triples of positive integers, i.e., the triples
$(i,j,k)$ such that
$i lt.eq j$ and
$i^(2) + j^(2) =k^(2)$.
])

#exercise(label-name: <ex:weighted-pairs>, [
It would be nice to be able to generate
#idx("infinite stream(s)", sub: "merging")
streams in which the pairs
appear in some useful order, rather than in the order that results
from an #emph[ad hoc] interleaving process. We can use a technique
similar to the #py("merge")
function
of exercise @ex:merge, if we define a way to say that
one pair of integers is "less than" another. One way to do
this is to define a
"weighting function"
$W(i,j)$ and stipulate that
$(i_(1),j_(1))$ is less than
$(i_(2),j_(2))$ if
$W(i_(1),j_(1)) < W(i_(2),j_(2))$. Write a
#idx("mergeweighted")
function #py("merge_weighted")
that is like #py("merge"), except that
#py("merge_weighted")
takes an additional argument #py("weight"), which is a
function
that computes the weight of a pair, and is used to determine the order in
which elements should appear in the resulting merged stream.#footnote[We
will require that the weighting function be such that the weight of a pair
increases as we move out along a row or down along a column of the array of
pairs.] Using this, generalize #py("pairs")
to a
function #py("weighted_pairs")
that takes two streams, together with a
function
that computes a weighting function, and generates the stream of pairs,
ordered according to weight. Use your
function
to generate

+ the stream of all pairs of positive integers $(i,j)$ with $i lt.eq j$ ordered according to the sum $i + j$
+ the stream of all pairs of positive integers $(i,j)$ with $i lt.eq j$, where neither $i$ nor $j$ is divisible by 2, 3, or 5, and the pairs are ordered according to the sum $2 i + 3 j + 5 i j$.
])

#exercise(label-name: <ex:ramanujan-nums>, [
Numbers that can be expressed as the sum of two cubes in more than one
way are sometimes called
#idx("Ramanujan numbers")
#emph[Ramanujan numbers], in honor of the
mathematician Srinivasa Ramanujan.#footnote[To quote from G. H.
Hardy's obituary of
#idx("Hardy, Godfrey Harold")
#idx("Ramanujan, Srinivasa")
Ramanujan (Hardy 1921): "It was Mr. Littlewood (I believe) who remarked that "every positive integer was one of his friends." I remember once going to see him when he was lying ill at Putney. I had ridden in taxi-cab No. 1729, and remarked that the number seemed to me a rather dull one, and that I hoped it was not an unfavorable omen. "No," he replied, "it is a very interesting number; it is the smallest number expressible as the sum of two cubes in two different ways."" The trick of using weighted pairs to
generate the Ramanujan numbers was shown to us by
#idx("Leiserson, Charles E.")
Charles
Leiserson.] Ordered streams of pairs provide an elegant solution
to the problem of computing these numbers. To find a number that can be
written as the sum of two cubes in two different ways, we need only generate
the stream of pairs of integers $(i,j)$ weighted
according to the sum $i^(3) + j^(3)$ (see
exercise @ex:weighted-pairs), then search the stream for
two consecutive pairs with the same weight. Write a
function
to generate the Ramanujan numbers. The first
such number is 1,729. What are the next five?
])

#exercise(label-name: <ex:3_72>, [
In a similar way to exercise @ex:ramanujan-nums generate
a stream of all numbers that can be written as the sum of two squares in
three different ways (showing how they can be so written).
])

#idx("pair(s)", sub: "infinite stream of")
#idx("infinite stream(s)", sub: "of pairs")
#idx("mapping", sub: "nested")

#subheading([Streams as signals])

#idx("signal processing", sub: "stream model of")
#idx("infinite stream(s)", sub: "to model signals")

We began our discussion of streams by describing them as computational
analogs of the "signals" in signal-processing systems.
In fact, we can use streams to model signal-processing systems in a very
direct way, representing the values of a signal at successive time
intervals as consecutive elements of a stream. For instance, we can
implement an
#idx("integrator, for signals")
#emph[integrator] or
#emph[summer] that, for an input stream
$x=(x_(i))$, an initial value $C$, and a small increment $d t$,
accumulates the sum

$ mat(delim: #none, S_(i), =, C +sum_(j=1)^(i) x_(j) thin d t) $

and returns the stream of values $S=(S_(i))$.
The following #py("integral")
function
is reminiscent of the "implicit style" definition of the
stream of integers (section @sec:infinite-streams):

#idx("integral", decl: true)
#snippet(```python
def integral(integrand, initial_value, dt):
    integ = pair(initial_value,
                 lambda: add_streams(scale_stream(integrand, dt),
                                     integ))
    return integ
```)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-49.svg", width: 54%), caption: [The #py("integral") function viewed as a signal-processing system.], label-name: <fig:integral>)

Figure @fig:integral
is a picture of a signal-processing
system that corresponds to the #py("integral")
function.
The input stream is scaled by $d t$ and passed
through an adder, whose output is passed back through the same adder.
The self-reference in the definition of
#py("integ")
is reflected in the figure by the feedback loop that
connects the output of the adder to one of the inputs.

#exercise(label-name: <ex:rc-circuit>, [
We can model electrical circuits using streams to represent the values
of currents or voltages at a sequence of times. For instance, suppose
we have an
#idx("RC circuit")
#idx("circuit", sub: "modeled with streams")
#idx("electrical circuits, modeled with streams")
#emph[RC circuit] consisting of a resistor of resistance
$R$ and a capacitor of capacitance
$C$ in series. The voltage response
$v$ of the circuit to an injected current
$i$ is determined by the formula in
figure @fig:rc, whose structure is shown by the
accompanying signal-flow diagram.

Write a
function
#py("RC") that models this circuit.
#py("RC") should take as inputs the values of
$R$, $C$, and
$d t$ and should return a
function
that takes as inputs a stream representing the current
$i$ and an initial value for the capacitor
voltage $v_(0)$ and produces as output the
stream of voltages $v$. For example, you
should be able to use #py("RC") to model an RC
circuit with $R = 5$ ohms,
$C = 1$ farad, and a 0.5-second time step by
evaluating
#py("const RC1 = RC(5, 1, 0.5)").
This defines #py("RC1") as a
function
that takes a stream representing the time sequence of currents and an
initial capacitor voltage and produces the output stream of voltages.
])

#exercise(label-name: <ex:zero-crossing>, [
Alyssa P. Hacker is designing a system to process signals coming from
physical sensors. One important feature she wishes to produce is a signal
that describes the
#idx("signal processing", sub: "zero crossings of a signal")
#idx("zero crossings of a signal")
#emph[zero crossings] of the input signal. That is,
the resulting signal should be $+1$ whenever the
input signal changes from negative to positive,
$-1$ whenever the input signal changes from
positive to negative, and 0 otherwise. (Assume that the sign of a 0 input
is positive.) For example, a typical input signal with its associated
zero-crossing signal would be

#syntax($dots.h$, " 1  2  1.5  1  0.5  -0.1  -2  -3  -2  -0.5  0.2  3  4 ", $dots.h$, "
", $dots.h med$, "  0  0    0  0    0     -1  0   0   0     0    1  0  0 ", $dots.h$)

In Alyssa's system, the signal from the sensor is represented as a
stream
#py("sense_data")
and the stream
#py("zero_crossings")
is the corresponding stream of zero crossings. Alyssa first writes a
function
#py("sign_change_detector")
that takes two values as arguments and compares the signs of the values to
produce an appropriate $0$,
$1$, or $-1$. She
then constructs her zero-crossing stream as follows:

#snippet(```python
def make_zero_crossings(input_stream, last_value):
    return pair(sign_change_detector(head(input_stream), last_value),
                lambda: make_zero_crossings(stream_tail(input_stream),
                                            head(input_stream)))

zero_crossings = make_zero_crossings(sense_data, 0)
```)

Alyssa's boss, Eva Lu Ator, walks by and suggests that this program is
approximately equivalent to the following one, which uses
the function #py("stream_map_2") from exercise @ex:combine-streams:

#syntax("
zero_crossings = stream_map_2(sign_change_detector,
                              sense_data,
                              ", meta("expression"), ")
      ")

Complete the program by supplying the indicated
#meta("expression").
])

#exercise(label-name: <ex:zero-crossing-2>, [
Unfortunately, Alyssa's
#idx("signal processing", sub: "zero crossings of a signal")
#idx("zero crossings of a signal")
#idx("signal processing", sub: "smoothing a signal")
#idx("smoothing a signal")
zero-crossing detector in
exercise @ex:zero-crossing proves to be insufficient,
because the noisy signal from the sensor leads to spurious zero crossings.
Lem E. Tweakit, a hardware specialist, suggests that Alyssa smooth the
signal to filter out the noise before extracting the zero crossings.
Alyssa takes his advice and decides to extract the zero crossings from
the signal constructed by averaging each value of the sense data with
the previous value. She explains the problem to her assistant, Louis
Reasoner, who attempts to implement the idea, altering Alyssa's
program as follows:

#snippet(```python
def make_zero_crossings(input_stream, last_value):
    avpt = (head(input_stream) + last_value) / 2
    return pair(sign_change_detector(avpt, last_value),
                lambda: make_zero_crossings(stream_tail(input_stream),
                                            avpt))
```)

This does not correctly implement Alyssa's plan.
Find the bug that Louis has installed
and fix it without changing the structure of the program. (Hint: You
will need to increase the number of arguments to
#py("make_zero_crossings").)
])

#sicp-figure(image("/images/img_original/ch3-Z-G-51.svg", width: 70%), caption: [An RC circuit and the associated #idx("signal-flow diagram") signal-flow diagram.], label-name: <fig:rc>)

#exercise(label-name: <ex:3_76>, [
Eva Lu Ator has a criticism of Louis's approach in
exercise @ex:zero-crossing-2.
#idx("signal processing", sub: "zero crossings of a signal")
#idx("zero crossings of a signal")
#idx("signal processing", sub: "smoothing a signal")
#idx("smoothing a signal")
The program he wrote is
not modular, because it intermixes the operation of smoothing with the
zero-crossing extraction. For example, the extractor should not have
to be changed if Alyssa finds a better way to condition her input
signal. Help Louis by writing a
function
#py("smooth") that takes a stream as input and
produces a stream in which each element is the average of two successive
input stream elements. Then use #py("smooth") as a
component to implement the zero-crossing detector in a more modular style.
])

#idx("signal processing", sub: "stream model of")
#idx("infinite stream(s)", sub: "to model signals")
