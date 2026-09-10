// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Modularity of Functional Programs and Modularity of Objects], label-name: <sec:functions-and-objects>)

#idx("modularity", sub: "functional programs vs. objects")
#idx("functional programming")

As we saw in section @sec:benefits-of-assignment, one of
the major benefits of introducing assignment is that we can increase the
modularity of our systems by encapsulating, or "hiding," parts
of the state of a large system within local variables. Stream models can
provide an equivalent modularity without the use of assignment. As an
illustration, we can reimplement the
#idx("π (pi)", sub: "Dirichlet estimate for", sort: "pi")
#idx("Monte Carlo simulation", sub: "stream formulation")
Monte Carlo estimation
of $pi$, which we examined in
section @sec:benefits-of-assignment, from a
stream-processing point of view.

The key modularity issue was that we wished to hide the internal state
of a random-number generator from programs that used random numbers.
We began with a
function #py("rand_update"),
whose successive values furnished our supply of random numbers, and used
this to produce a random-number generator:

#snippet(```python
def make_rand():
    x = random_init
    def rand():
        nonlocal x
        x = rand_update(x)
        return x
    return rand

rand = make_rand()
```)

In the stream formulation there is no random-number generator #emph[per se], just a stream of random numbers produced by successive calls to
#py("rand_update"):
#idx("randomnumbers (infinite stream)", decl: true)#idx("infinite stream(s)", sub: "of random numbers")
#snippet(```python
random_numbers = pair(random_init,
                      lambda: stream_map(rand_update, random_numbers))
```)

We use this to construct the stream of outcomes of the Cesàro
experiment performed on consecutive pairs in the
#py("random_numbers")
stream:
#idx("dirichletstream", decl: true)#idx("mapsuccessivepairs", decl: true)
#snippet(```python
def map_successive_pairs(f, s):
    return pair(f(head(s), head(stream_tail(s))),
                lambda: map_successive_pairs(
                            f,
                            stream_tail(stream_tail(s))))

dirichlet_stream = map_successive_pairs(lambda r1, r2: gcd(r1, r2) == 1,
                                        random_numbers)
```)

The
#py("dirichlet_stream")
is now fed to a
#py("monte_carlo")
function,
which produces a stream of estimates of probabilities. The results are then
converted into a stream of estimates of $pi$.
This version of the program doesn't need a parameter telling how many
trials to perform. Better estimates of $pi$
(from performing more experiments) are obtained by looking farther into the
#py("pi") stream:
#idx("montecarlo", sub: "infinite stream", decl: true)
#snippet(```python
def monte_carlo(experiment_stream, passed, failed):
    def next(passed, failed):
        return pair(passed / (passed + failed),
                    lambda: monte_carlo(stream_tail(experiment_stream),
                                        passed, failed))
    return (next(passed + 1, failed)
            if head(experiment_stream)
            else next(passed, failed + 1))

pi = stream_map(lambda p: math_sqrt(6 / p),
                monte_carlo(dirichlet_stream, 0, 0))
```)

There is considerable
#idx("modularity", sub: "through infinite streams")
modularity in this approach, because we still
can formulate a general
#py("monte_carlo") function
that can deal with arbitrary experiments. Yet there is no assignment or
local state.

#exercise(label-name: <ex:3_81>, [
Exercise @ex:random-with-reset discussed generalizing
the random-number generator to allow one to
#idx("random-number generator", sub: "with reset, stream version")
reset the random-number sequence
so as to produce repeatable sequences of "random" numbers.
Produce a stream formulation of this same generator that operates on an
input stream of requests to
#py("\"generate\"")
a new
random number or to
#py("\"reset\"")
the sequence to a
specified value and that produces the desired stream of random numbers.
Don't use assignment in your solution.
])

#exercise(label-name: <ex:3_82>, [
Redo exercise @ex:monte-carlo-integration on
#idx("Monte Carlo integration", sub: "stream formulation")
#idx("π (pi)", sub: "approximation with Monte Carlo integration", sort: "pi")
#idx("definite integral", sub: "estimated with Monte Carlo simulation")
Monte Carlo integration in terms of streams. The stream version of
#py("estimate_integral")
will not have an argument telling how many trials to perform. Instead, it
will produce a stream of estimates based on successively more trials.
])

#subheading([A functional-programming view of time])

#idx("time", sub: "functional programming and")
#idx("functional programming", sub: "time and")

Let us now return to the issues of objects and state that were raised
at the beginning of this chapter and examine them in a new light. We
introduced assignment and mutable objects to provide a mechanism for
modular construction of programs that model systems with state.
We constructed computational objects with local state variables and used
assignment to modify these variables. We modeled the temporal behavior of
the objects in the world by the temporal behavior of the corresponding
computational objects.

Now we have seen that streams provide an alternative way to model
objects with local state. We can model a changing quantity, such as
the local state of some object, using a stream that represents the
time history of successive states. In essence, we represent time
explicitly, using streams, so that we decouple time in our simulated
world from the sequence of events that take place during evaluation.
Indeed, because of the presence of
delayed evaluation
there may be little relation between simulated time in the model and the
order of events during the evaluation.

In order to contrast these two approaches to modeling, let us
reconsider the implementation of a "withdrawal processor" that
monitors the balance in a
#idx("bank account", sub: "stream model")
bank account. In
section @sec:costs-of-assignment we implemented a
simplified version of such a processor:
#idx("makesimplifiedwithdraw", decl: true)
#snippet(```python
def make_simplified_withdraw(balance):
    def withdraw(amount):
        nonlocal balance
        balance = balance - amount
        return balance
    return withdraw
```)

Calls to
#py("make_simplified_withdraw")
produce computational objects, each with a local state variable
#py("balance") that is decremented by successive calls
to the object. The object takes an #py("amount") as
an argument and returns the new balance. We can imagine the user of a bank
account typing a sequence of inputs to such an object and observing the
sequence of returned values shown on a display screen.

Alternatively, we can model a withdrawal processor as a
function
that takes as input a balance and a stream of amounts to withdraw and
produces the stream of successive balances in the account:
#idx("streamwithdraw", decl: true)
#snippet(```python
def stream_withdraw(balance, amount_stream):
    return pair(balance,
                lambda: stream_withdraw(balance - head(amount_stream),
                                        stream_tail(amount_stream)))
```)

The function #py("stream_withdraw")
implements a well-defined mathematical function whose output is fully
determined by its input. Suppose, however, that the input
#py("amount_stream")
is the stream of successive values typed by the user and that the resulting
stream of balances is displayed. Then, from the perspective of the user who
is typing values and watching results, the stream process has the same
behavior as the object created by
#py("make_simplified_withdraw").
However, with the stream version, there is no assignment, no local state
variable, and consequently none of the theoretical difficulties that we
encountered
#idx("state", sub: "vanishes in stream formulation")
in section @sec:costs-of-assignment. Yet the system
has state!

This is really remarkable. Even though
#py("stream_withdraw")
implements a well-defined mathematical function whose behavior does not
change, the user's perception here is one of interacting with a system
that has a changing state. One way to resolve this paradox is to realize
that it is the user's temporal existence that imposes state on the
system. If the user could step back from the interaction and think in terms
of streams of balances rather than individual transactions, the system
would appear stateless.#footnote[Similarly in physics, when we observe a
moving particle, we say that the position (state) of the particle is
changing. However, from the perspective of the particle's
#idx("world line of a particle")
world line in space-time there is no change involved.]

From the point of view of one part of a complex process, the other parts
appear to change with time. They have hidden time-varying local state. If
we wish to write programs that model this kind of natural decomposition in
our world (as we see it from our viewpoint as a part of that world) with
structures in our computer, we make computational objects that are not
functional—they must change with time. We model state with local
state variables, and we model the changes of state with assignments to
those variables. By doing this we make the time of execution of a
computation model time in the world that we are part of, and thus we
get "objects" in our computer.

Modeling with objects is powerful and intuitive, largely because this
matches the perception of interacting with a world of which we are
part. However, as we've seen repeatedly throughout this chapter,
these models raise thorny problems of constraining the order of events
and of synchronizing multiple processes. The possibility of avoiding
these problems has stimulated the development of
#idx("programming language", sub: "functional")
#idx("functional programming", sub: "functional programming languages")
#emph[functional programming languages], which do not include any
provision for assignment or mutable data. In such a language, all
functions
implement well-defined mathematical functions of their arguments,
whose behavior does not change. The functional approach is extremely
attractive for dealing with
#idx("concurrency", sub: "functional programming and")
#idx("functional programming", sub: "concurrency and")
concurrent systems.#footnote[John Backus, the
#idx("Fortran", sub: "inventor of")
inventor of Fortran, gave high
visibility to functional programming when he was awarded the ACM Turing
award in 1978. His acceptance speech
#idx("Backus, John")
(Backus 1978)
strongly advocated the functional approach. A good overview of functional
programming is given in
#idx("Henderson, Peter")
Henderson 1980 and in
#idx("Darlington, John")
#idx("Turner, David")
Darlington, Henderson, and Turner 1982.]

On the other hand, if we look closely, we can see time-related problems
creeping into functional models as well. One particularly troublesome area
arises when we wish to design interactive systems, especially ones that
model interactions between independent entities. For instance, consider once
more the implementation of a banking system that permits joint bank accounts.
In a conventional system using assignment and objects, we would model the
fact that Peter and Paul share an account by having both Peter and Paul send
their transaction requests to the same bank-account object, as we saw in
section @sec:costs-of-assignment. From the stream point
of view, where there are no "objects" #emph[per se], we have
already indicated that a bank account can be modeled as a process that
operates on a stream of transaction requests to produce a stream of
responses. Accordingly, we could model the fact that Peter and Paul have a
joint bank account by merging Peter's stream of transaction requests
with Paul's stream of requests and feeding the result to the
bank-account stream process, as shown in
figure @fig:joint-account-stream.

#sicp-figure(image("/images/img_original/ch3-Z-G-60.svg", width: 70%), caption: [A joint #idx("bank account", sub: "joint, modeled with streams") bank account, modeled by merging two streams of transaction requests.], label-name: <fig:joint-account-stream>)

The trouble with this formulation is in the notion of #emph[merge]. It
will not do to merge the two streams by simply taking alternately one
request from Peter and one request from Paul. Suppose Paul accesses
the account only very rarely. We could hardly force Peter to wait for
Paul to access the account before he could issue a second transaction.
#idx("infinite stream(s)", sub: "merging")
However such a merge is implemented, it must interleave the two
transaction streams in some way that is constrained by "real time" as perceived by Peter and Paul, in the sense that, if Peter and
Paul meet, they can agree that certain transactions were processed
before the meeting, and other transactions were processed after the
meeting.#footnote[Observe that, for any two streams, there is in general
more than one
acceptable order of interleaving. Thus, technically, "merge"
is a
#idx("nondeterminism, in behavior of concurrent programs")
#idx("infinite stream(s)", sub: "merging as a relation")
relation rather than a function—the answer is not a
deterministic function of the inputs. We already mentioned
(footnote @foot:nondeterministic) that nondeterminism
is essential when dealing with concurrency. The merge relation illustrates
the same essential nondeterminism, from the functional perspective.
In section @sec:nondeterministic-evaluation, we
will look at nondeterminism from yet another point of view.]
This is precisely the same constraint that we had to deal with in
section @sec:nature-of-time, where we found the need to
introduce explicit synchronization to ensure a "correct" order
of events in concurrent processing of objects with state. Thus, in an
attempt to support the functional style, the need to merge inputs from
different agents reintroduces the same problems that the functional style
was meant to eliminate.

We began this chapter with the goal of building computational models
whose structure matches our perception of the real world we are trying
to model. We can model the world as a collection of separate,
time-bound, interacting objects with state, or we can model the world
as a single, timeless, stateless unity. Each view has powerful
advantages, but neither view alone is completely satisfactory. A
grand unification has yet to emerge.#footnote[The object model approximates
the world by dividing it into separate pieces. The functional model does
not
#idx("modularity", sub: "along object boundaries")
modularize along object boundaries. The object model is useful when
the unshared state of the "objects" is much larger than the
state that they share. An example of a place where the object viewpoint
fails is
#idx("quantum mechanics")
quantum mechanics, where thinking of things as individual particles leads
to paradoxes and confusions. Unifying the object view with the
functional view may have little to do with programming, but rather
with fundamental epistemological issues.]

#idx("stream(s)")
#idx("modularity", sub: "functional programs vs. objects")
#idx("functional programming")
#idx("time", sub: "functional programming and")
#idx("functional programming", sub: "time and")
