// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Streams Are Delayed Lists], label-name: <sec:delayed-lists>)

#idx("stream(s)", sub: "implemented as delayed lists")

As we saw in
section @sec:sequences-conventional-interfaces,
sequences can serve as standard interfaces for combining program
modules. We formulated powerful abstractions for manipulating
sequences, such as #py("map"),
#py("filter"), and
#py("accumulate"), that capture a wide variety of
operations in a manner that is both succinct and elegant.

Unfortunately, if we represent sequences as lists, this elegance is
bought at the price of severe inefficiency with respect to both the
time and space required by our computations.
When we represent manipulations on sequences as transformations
of lists, our programs must construct and copy data structures (which
may be huge) at every step of a process.

To see why this is true, let us compare two programs for computing the
sum of all the prime numbers in an interval. The first program is
written in standard iterative style:#footnote[Assume that we have a
predicate
#py("is_prime")
(e.g., as in section @sec:primality) that
tests for primality.]
#idx("sumprimes", decl: true)
#snippet(```python
def sum_primes(a, b):
    def iter(count, accum):
        return (accum
                if count > b
                else iter(count + 1, count + accum)
                if is_prime(count)
                else iter(count + 1, accum))
    return iter(a, 0)
```)

The second program performs the same computation using the sequence
operations of
section @sec:sequences-conventional-interfaces:
#idx("sumprimes", decl: true)
#snippet(```python
def sum_primes(a, b):
    return reduce(lambda x, y: x + y,
                  0,
                  filter(is_prime,
                         enumerate_interval(a, b)))
```)

In carrying out the computation, the first program needs to store only
the sum being accumulated. In contrast, the filter in the second
program cannot do any testing until
#py("enumerate_interval")
has constructed a complete list of the numbers in the interval.
The filter generates another list, which in turn is passed to
#py("accumulate") before being collapsed to form
a sum. Such large intermediate storage is not needed by the first program,
which we can think of as enumerating the interval incrementally, adding
each prime to the sum as it is generated.

The inefficiency in using lists becomes painfully apparent if we use
the sequence paradigm to compute the second prime in the interval from
10,000 to 1,000,000 by evaluating the expression

#snippet(```python
head(tail(filter(is_prime,
                 enumerate_interval(10000, 1000000))))
```)

This expression does find the second prime, but the computational overhead
is outrageous. We construct a list of almost a million integers, filter
this list by testing each element for primality, and then ignore almost
all of the result. In a more traditional programming style, we would
interleave the enumeration and the filtering, and stop when we reached
the second prime.

Streams are a clever idea that allows one to use sequence
manipulations without incurring the costs of manipulating sequences as
lists. With streams we can achieve the best of both worlds: We can
formulate programs elegantly as sequence manipulations, while attaining
the efficiency of incremental computation. The basic idea is to arrange
to construct a stream only partially, and to pass the partial
construction to the program that consumes the stream. If the consumer
attempts to access a part of the stream that has not yet been

constructed, the stream will automatically construct just enough more
of itself to produce the required part, thus preserving the illusion
that the entire stream exists. In other words, although we will write
programs as if we were processing complete sequences, we design our
stream implementation to automatically and transparently interleave
the construction of the stream with its use.

To accomplish this, we will construct streams using pairs,
with the first item of the stream in the head of the pair.
However, rather than placing the value of the rest of the stream
#idx("promise to evaluate")
into the tail of the pair, we will put there a "promise"
to compute the rest if it is ever requested.
If we have a data item
#py("h") and a stream
#py("t"), we construct a stream
whose head is
#py("h") and whose tail is
#py("t") by evaluating
#py("pair(h, lambda: t)")—the
tail
#py("t") of a stream is
"wrapped" in a function of no arguments,
#idx("delayed expression")
so that its evaluation will be #emph[delayed].
#idx("empty stream")
#idx("stream(s)", sub: "empty")
The empty stream is
#py("null"), the same as the empty list.

To access the first data item of a nonempty stream,
we simply select the
#py("head") of the pair, as with a list.
But to access the tail of a stream, we need to evaluate the
delayed expression.
For convenience, we define
#idx("streamtail", decl: true)
#snippet(```python
def stream_tail(stream):
    return tail(stream)()
```)

This selects the tail of the pair and applies the function
found there to obtain the next pair of the stream
(or
#py("null") if the tail of the stream
is empty)—in effect,
#idx("forcing", sub: "tail of stream")
#emph[forcing] the function in the
tail of the pair to fulfill its promise.
#idx("stream(s)", sub: "implemented as delayed lists")

We can make and use streams, in just the same way as we can make
and use lists, to represent aggregate data arranged in a sequence. In
particular, we can build stream analogs of the list operations from
chapter @chap:data, such as #py("list_ref"),
#py("map"), and
#py("for_each"):#footnote[This should
bother you. The fact that we are defining such similar functions
for streams and lists indicates that we are missing some underlying
abstraction. Unfortunately, in order to exploit this abstraction, we
will need to exert finer control over the process of evaluation than we
can at present. We will discuss this point further at the end of
section @sec:streams-and-delayed-evaluation.
In section @sec:lazy-evaluation, we'll
develop a framework that unifies lists and streams.]
#idx("streamref", decl: true)#idx("streammap", decl: true)#idx("streamforeach", decl: true)
#snippet(```python
def stream_ref(s, n):
    return (head(s)
            if n == 0
            else stream_ref(stream_tail(s), n - 1))

def stream_map(f, s):
    return (None
            if is_none(s)
            else pair(f(head(s)),
                      lambda: stream_map(f, stream_tail(s))))

def stream_for_each(fun, s):
    if is_none(s):
        return True
    else:
        fun(head(s))
        return stream_for_each(fun, stream_tail(s))
```)

The function
#py("stream_for_each") is useful for
viewing streams:
#idx("displaystream", decl: true)
#snippet(```python
def display_stream(s):
    return stream_for_each(display, s)
```)

To make the stream implementation automatically and transparently
interleave the construction of a stream with its use, we have arranged
for the tail
of a stream to be evaluated when it is accessed by the
#py("stream_tail")
function rather than when the stream is constructed by
#py("pair").
This implementation choice is reminiscent of our discussion of rational numbers
in section @sec:abstraction-barriers, where we saw
that we can choose to implement rational numbers so that the reduction
of numerator and denominator to lowest terms is performed either at
construction time or at selection time. The two rational-number
implementations produce the same data abstraction, but the choice has
an effect on efficiency. There is a similar relationship between
streams and ordinary lists. As a data abstraction, streams are the
same as lists. The difference is the time at which the elements are
evaluated. With ordinary lists, both the
#py("head") and the
#py("tail")
are evaluated at construction time. With streams, the
#py("tail") is evaluated at selection time.

#subheading([Streams in action])

To see how this data structure behaves, let us analyze the
"outrageous" prime computation we saw above, reformulated
in terms of streams:

#snippet(```python
print(head(stream_tail(stream_filter(
                     is_prime,
                     stream_enumerate_interval(10000, 1000000)))))
```)

We will see that it does indeed work efficiently.

We begin by calling
#py("stream_enumerate_interval") with
the arguments 10,000 and 1,000,000. The function
#py("stream_enumerate_interval")
is the stream analog of
#py("enumerate_interval")
(section @sec:sequences-conventional-interfaces):

#idx("streamenumerateinterval", decl: true)
#snippet(```python
def stream_enumerate_interval(low, high):
    return (None
            if low > high
            else pair(low,
                      lambda: stream_enumerate_interval(low + 1, high)))
```)

and thus the result returned by
#py("stream_enumerate_interval"),
formed by the #py("pair"),
is#footnote[The numbers shown here do not really appear in the delayed
expression. What actually appears is the original expression, in an
environment in which the variables are bound to the appropriate numbers.
For example, #py("low + 1") with
#py("low") bound to 10,000 actually appears
where #py("10001") is shown.]

#snippet(```python
print(pair(10000, lambda: stream_enumerate_interval(10001, 1000000)))
```)

That is, #py("stream_enumerate_interval")
returns a stream represented as a pair whose
#py("head")
is 10,000 and whose #py("tail")
is a promise to enumerate more of the
interval if so requested. This stream is now filtered for primes,
using the stream analog of the #py("filter")
function
(section @sec:sequences-conventional-interfaces):
#idx("streamfilter", decl: true)
#snippet(```python
def stream_filter(pred, stream):
    return (None
            if is_none(stream)
            else pair(head(stream),
                      lambda: stream_filter(pred, stream_tail(stream)))
            if pred(head(stream))
            else stream_filter(pred, stream_tail(stream)))
```)

The function #py("stream_filter") tests the
#py("head") of the stream (which is 10,000). Since
this is not prime, #py("stream_filter")
examines the tail of its input stream. The call to
#py("stream_tail") forces evaluation of the
delayed #py("stream_enumerate_interval"),
which now returns

#snippet(```python
print(pair(10001, lambda: stream_enumerate_interval(10002, 1000000)))
```)

The function #py("stream_filter") now
looks at the #py("head") of this stream,
10,001, sees that this is not prime either, forces another
#py("stream_tail"), and so on, until
#py("stream_enumerate_interval") yields
the prime 10,007, whereupon
#py("stream_filter"), according to its
definition, returns

#snippet(```python
pair(head(stream),
     lambda: stream_filter(pred, stream_tail(stream)))
```)

which in this case is

#snippet(```python
print(pair(10007,
     lambda: stream_filter(
                 is_prime,
                 pair(10008,
                      lambda: stream_enumerate_interval(10009, 1000000)))))
```)

This result is now passed to
#py("stream_tail") in our original
expression. This forces the delayed
#py("stream_filter"),
which in turn keeps forcing the delayed
#py("stream_enumerate_interval") until it
finds the next prime, which is 10,009. Finally, the result passed to
#py("head") in our original expression is

#snippet(```python
print(pair(10009,
     lambda: stream_filter(
                 is_prime,
                 pair(10010,
                      lambda: stream_enumerate_interval(10011, 1000000)))))
```)

The function #py("head") returns 10,009, and the
computation is complete. Only as many integers were tested for
primality as were necessary to find the second prime, and the interval
was enumerated only as far as was necessary to feed the prime filter.

In general, we can think of delayed evaluation as
#idx("programming", sub: "demand-driven")
"demand-driven" programming, whereby each stage in the
stream process is activated only enough to satisfy the next stage. What
we have done is to
#idx("order of events", sub: "decoupling apparent from actual")
decouple the actual order of events in the computation from the apparent
structure of our functions. We write functions as if the streams existed
"all at once" when, in reality, the computation is
performed incrementally, as in traditional programming styles.

#subheading([An optimization])

When we construct stream pairs, we delay the evaluation of their tail
expressions by wrapping these expressions in a function. We force their
evaluation when needed, by applying the function.

This implementation suffices for streams to work as advertised, but
there is an important optimization that we shall consider where needed.
In many	applications, we end up forcing the same delayed object many
times. This can lead to serious inefficiency in recursive programs
involving streams. (See
exercise @ex:fib-stream-efficiency.)
The solution is to build delayed objects so that the first time they are
forced, they store the value that is computed. Subsequent forcings will
simply return the stored value without repeating the computation. In
other words, we implement the construction of stream pairs as a
#idx("delayed expression", sub: "memoized")
#idx("memoization", sub: "in stream tail")
memoized function similar to the one described in
exercise @ex:memoization. One way to accomplish this
is to use the following function, which takes as argument a function
(of no arguments) and returns a memoized version of the function.
The first time the memoized function is run, it saves the computed
result.	On subsequent evaluations, it simply returns
the result.#footnote[There are many possible implementations of streams
other than the one described in this section. Delayed evaluation, which
is the key to making streams practical, was inherent in
#idx("Algol", sub: "call-by-name argument passing")
Algol 60's
#idx("call-by-name argument passing")
#emph[call-by-name]
parameter-passing method. The use of this mechanism to implement
streams was first described by
#idx("Landin, Peter")
Landin (1965). Delayed evaluation for
streams was introduced into Lisp by
#idx("Friedman, Daniel P.")
#idx("Wise, David S.")
Friedman and Wise (1976). In their
implementation,
#py("cons") (the Lisp
equivalent of our
#py("pair") function)
always delays evaluating its arguments, so
that lists automatically behave as streams. The memoizing
optimization is also known as
#idx("call-by-need argument passing")
#emph[call-by-need]. The Algol community would refer to our original
delayed objects as
#idx("thunk", sub: "call-by-name")
#idx("thunk", sub: "call-by-need")
#idx("Algol", sub: "thunks")
#emph[call-by-name thunks] and to the optimized
versions as #emph[call-by-need thunks].]
#idx("memo", decl: true)
#snippet(```python
def memo(fun):
    already_run = False
    result = None
    def memoized():
        nonlocal already_run, result
        if not already_run:
            result = fun()
            already_run = True
            return result
        else:
            return result
    return memoized
```)

We can make use of #py("memo") whenever
we construct a stream pair. For example, instead of
#idx("streammap", decl: true)
#snippet(```python
def stream_map(f, s):
    return (None
            if is_none(s)
            else pair(f(head(s)),
                      lambda: stream_map(f, stream_tail(s))))
```)

we can define an optimized function
#py("stream_map")	as
follows:
#idx("streammapoptimized", decl: true)
#snippet(```python
def stream_map_optimized(f, s):
    return (None
            if is_none(s)
            else pair(f(head(s)),
                      memo(lambda:
                           stream_map_optimized(f, stream_tail(s)))))
```)

#exercise(label-name: <ex:combine-streams>, [
Declare a function #py("stream_map_2")
that takes a binary function and two streams as arguments and returns
a stream whose elements are the results of applying the function
pairwise to the corresponding elements of the argument streams.
#idx("streammap2")
#snippet(```python
def stream_map_2(f, s1, s2):
    ...
```)

Similar to #py("stream_map_optimized"),
declare a function
#py("stream_map_2_optimized") by
modifying your
#py("stream_map_2")
such that the result stream employs memoization.
])

#exercise(label-name: <ex:delayed1>, [
Note that our primitive function
#py("display") returns its argument
after displaying it.
What does the interpreter print in response to evaluating each
statement in the following sequence?#footnote[Exercises such
as @ex:delayed1 and @ex:delayed2
are valuable for testing our understanding of how delayed evaluation
works. On the other hand, intermixing delayed evaluation with
#idx("delayed evaluation", sub: "printing and")
printing—and, even worse, with assignment—is extremely
confusing, and instructors of courses on computer languages have
traditionally tormented their students with examination questions such
as the ones in this section. Needless to say, writing programs that
depend on such subtleties is
#idx("programming", sub: "odious style")
odious programming style. Part of the power of stream processing is
that it lets us ignore the order in which events actually happen in
our programs. Unfortunately, this is precisely what we cannot afford
to do in the presence of assignment, which forces us to be concerned
with time and change.]

#snippet(```python
x = stream_map(display, stream_enumerate_interval(0, 10))

stream_ref(x, 5)

stream_ref(x, 7)
```)

What does the interpreter print if
#py("stream_map_optimized")
is used instead of #py("stream_map")?

#snippet(```python
x = stream_map_optimized(display, stream_enumerate_interval(0, 10))

stream_ref(x, 5)

stream_ref(x, 7)
```)
])

#exercise(label-name: <ex:delayed2>, [
Consider the sequence of statements

#snippet(```python
sum = 0

def accum(x):
    global sum
    sum = x + sum
    return sum

seq = stream_map(accum, stream_enumerate_interval(1, 20))

y = stream_filter(is_even, seq)

z = stream_filter(lambda x: x % 5 == 0, seq)

stream_ref(y, 7)

display_stream(z)
```)

What is the value of #py("sum") after each of the
above statements is evaluated?
#idx("delayed evaluation", sub: "assignment and")
What is the printed response to	evaluating the
#py("stream_ref") and
#py("display_stream") expressions?
Would these responses differ if we had applied the function
#py("memo")
on every tail of every constructed stream pair, as suggested in the
optimization above? Explain.
])
