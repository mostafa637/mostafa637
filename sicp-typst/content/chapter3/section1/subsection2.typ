// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([The Benefits of Introducing Reassignment], label-name: <sec:benefits-of-assignment>)

#idx("reassignment", sub: "benefits of")

As we shall see, introducing
reassignment
into our programming language
leads us into a thicket of difficult conceptual issues. Nevertheless,
viewing systems as
#idx("object(s)", sub: "benefits of modeling with")
collections of objects with local state is a
powerful technique for maintaining a
#idx("modularity", sub: "through modeling with objects")
modular design. As a simple
example, consider the design of a
function
#py("rand") that, whenever
it is called, returns an integer chosen at random.

It is not at all clear what is meant by "chosen at random."
What we presumably want is for successive calls to
#idx("random-number generator")
#py("rand") to produce a sequence of numbers that has
statistical properties of uniform distribution. We will not discuss methods
for generating suitable sequences here. Rather, let us assume that we have a
function
#py("rand_update")
that has the property that if we start with a given number
$x_(1)$ and form

#syntax($x_(2)$, " = rand_update(", $x_(1)$, ")
", $x_(3)$, " = rand_update(", $x_(2)$, ")
      ")

then the sequence of values
$x_(1), x_(2), x_(3), dots.h$, will have the desired
statistical properties.#footnote[One common way to implement
#py("rand_update")
is to use the rule that $x$ is updated to
$a x+b$ modulo $m$,
where $a$, $b$, and
$m$ are appropriately chosen integers.
Chapter @chap:state of
#idx("Knuth, Donald E.")
Knuth 1997b includes an extensive
discussion of techniques for generating sequences of random numbers and
establishing their statistical properties. Notice that the
#py("rand_update")
function
computes a mathematical function: Given the same input twice, it
produces the same output. Therefore, the number sequence produced by
#py("rand_update")
certainly is not "random," if by "random" we
insist that each number in the sequence is unrelated to the preceding
number. The relation between "real randomness" and so-called
#idx("pseudo-random sequence")
#emph[pseudo-random] sequences, which are produced by well-determined
computations and yet have suitable statistical properties, is a
complex question involving difficult issues in mathematics and
philosophy.
#idx("Kolmogorov, A. N.")
Kolmogorov,
#idx("Solomonoff, Ray")
Solomonoff, and
#idx("Chaitin, Gregory")
Chaitin have made great
progress in clarifying these issues; a discussion can be found in
Chaitin 1975.]

We can implement #py("rand") as a
function
with a local state variable #py("x") that is
initialized to some fixed value
#py("random_init").
Each call to #py("rand") computes
#py("rand_update")
of the current value of #py("x"), returns this as the
random number, and also stores this as the new value of
#py("x").

#idx("rand", decl: true)
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

Of course, we could generate the same sequence of random numbers
without using
reassignment
by simply calling
#py("rand_update")
directly. However, this would mean that any part of our program that used
random numbers would have to explicitly remember the current value of
#py("x") to be passed as an argument to
#py("rand_update").
To realize what an annoyance this would be, consider using random numbers
to implement a technique called
#idx("Monte Carlo simulation")
#idx("random-number generator", sub: "in Monte Carlo simulation")
#emph[Monte Carlo simulation].

The Monte Carlo method consists of choosing sample experiments at random
from a large set and then making deductions on the basis of the
probabilities estimated from tabulating the results of those experiments.
For example, we can approximate
#idx("π (pi)", sub: "Dirichlet estimate for", sort: "pi")
$pi$ using the fact that
$6/ pi ^(2)$ is the probability that two integers
chosen at random will have no factors in common; that is, that their
greatest common divisor will be 1.#footnote[This theorem is due to G.
#idx("Dirichlet, Peter Gustav Lejeune")
Lejeune Dirichlet. See section 4.5.2 of
#idx("Knuth, Donald E.")
Knuth 1997b for a discussion and a proof.]
To obtain the approximation to $pi$, we perform
a large number of experiments. In each experiment we choose two integers at
random and perform a test
#idx("greatest common divisor", sub: "used to estimate π")
to see if their GCD is 1. The fraction of times that the test is passed
gives us our estimate of $6/ pi ^(2)$, and from this
we obtain our approximation to $pi$.

The heart of our program is a
function
#py("monte_carlo"),
which takes as arguments the number of times to try an experiment, together
with the experiment, represented as a no-argument
function
that will return either true or false each time it is run.
The function #py("monte_carlo")
runs the experiment for the designated number of trials and returns a
number telling the fraction of the trials in which the experiment was
found to be true.

#idx("estimatepi", decl: true)#idx("dirichlettest", decl: true)#idx("montecarlo", decl: true)
#snippet(```python
def estimate_pi(trials):
    return math_sqrt(6 / monte_carlo(trials, dirichlet_test))

def dirichlet_test():
    return gcd(rand(), rand()) == 1

def monte_carlo(trials, experiment):
    def iter(trials_remaining, trials_passed):
        return (trials_passed / trials
                if trials_remaining == 0
                else iter(trials_remaining - 1, trials_passed + 1)
                if experiment()
                else iter(trials_remaining - 1, trials_passed))
    return iter(trials, 0)
```)

Now let us try the same computation using
#py("rand_update")
directly rather than #py("rand"), the way we would be
forced to proceed if we did not use
reassignment
to model local state:
#idx("estimatepi", decl: true)
#snippet(```python
def estimate_pi(trials):
    return math_sqrt(6 / random_gcd_test(trials, random_init))

def random_gcd_test(trials, initial_x):
    def iter(trials_remaining, trials_passed, x):
        x1 = rand_update(x)
        x2 = rand_update(x1)
        return (trials_passed / trials
                if trials_remaining == 0
                else iter(trials_remaining - 1, trials_passed + 1, x2)
                if gcd(x1, x2) == 1
                else iter(trials_remaining - 1, trials_passed, x2))
    return iter(trials, 0, initial_x)
```)

While the program is still simple, it betrays some painful breaches of
modularity. In our first version of the program, using
#py("rand"), we can express the Monte Carlo method
directly as a general
#py("monte_carlo")
function
that takes as an argument an arbitrary
#py("experiment")
function.
In our second version of the program, with no local state for the
random-number generator,
#py("random_gcd_test")
must explicitly manipulate the random numbers
#py("x1") and #py("x2") and
recycle #py("x2") through the iterative loop as the
new input to
#py("rand_update").
This explicit handling of the random numbers intertwines the structure of
accumulating test results with the fact that our particular experiment uses
two random numbers, whereas other Monte Carlo experiments might use one
random number or three. Even the top-level
function
#py("estimate_pi")
has to be concerned with supplying an initial random number. The fact that
the random-number generator's insides are leaking out into other parts
of the program makes it difficult for us to isolate the Monte Carlo idea so
that it can be applied to other tasks. In the first version of the program,
reassignment
encapsulates the state of the random-number generator within the
#py("rand")
function,
so that the details of random-number generation remain independent of the
rest of the program.

The general phenomenon illustrated by the Monte Carlo example is this: From
the point of view of one part of a complex process, the other parts appear
to change with time. They have hidden time-varying local state. If we wish
to write computer programs whose structure reflects this decomposition, we
make computational objects (such as bank accounts and random-number
generators) whose behavior changes with time. We model state with local
state variables, and we model the changes of state with
reassignments
to those
variables.

It is tempting to conclude this discussion by saying that, by introducing
reassignment
and the technique of hiding state in local variables, we are able
to structure systems in a more modular fashion than if all state had to be
manipulated explicitly, by passing additional parameters. Unfortunately,
as we shall see, the story is not so simple.

#exercise(label-name: <ex:monte-carlo-integration>, [
#emph[Monte Carlo integration]
#idx("Monte Carlo integration")
#idx("π (pi)", sub: "approximation with Monte Carlo integration", sort: "pi")
#idx("definite integral", sub: "estimated with Monte Carlo simulation")
is a method of estimating definite
integrals by means of Monte Carlo simulation. Consider computing the
area of a region of space described by a predicate
$P(x, y)$ that is true for points
$(x, y)$ in the region and false for points not
in the region. For example, the region contained within a circle of radius
$3$ centered at
$(5, 7)$ is described by the predicate that tests
whether $(x-5)^(2) + (y-7)^(2) lt.eq 3^(2)$. To estimate
the area of the region described by such a predicate, begin by choosing a
rectangle that contains the region. For example, a rectangle with diagonally
opposite corners at $(2, 4)$ and
$(8, 10)$ contains the circle above. The desired
integral is the area of that portion of the rectangle that lies in the
region. We can estimate the integral by picking, at random, points
$(x, y)$ that lie in the rectangle, and testing
$P(x, y)$ for each point to determine whether the
point lies in the region. If we try this with many points, then the fraction
of points that fall in the region should give an estimate of the proportion
of the rectangle that lies in the region. Hence, multiplying this fraction
by the area of the entire rectangle should produce an estimate of the
integral.

Implement Monte Carlo integration as a
function
#idx("estimateintegral")
#py("estimate_integral")
that takes as arguments a predicate #py("P"), upper
and lower bounds #py("x1"),
#py("x2"), #py("y1"), and
#py("y2") for the rectangle, and the number of trials
to perform in order to produce the estimate. Your
function
should use the same
#py("monte_carlo")
function
that was used above to estimate $pi$. Use your
#py("estimate_integral")
to produce an estimate of $pi$ by measuring the
area of a unit circle.

You will find it useful to have a
function
that returns a number chosen at random from a given range. The following
#py("random_in_range")
function
implements this in terms of the
#py("math_random") function
used in section @sec:primality, which returns a
nonnegative number less
than 1\.
#idx("randominrange", decl: true)
#snippet(```python
def random_in_range(low, high):
    range = high - low
    return low + math_random() * range
```)
])

#exercise(label-name: <ex:random-with-reset>, [
It is useful to be able to
#idx("random-number generator", sub: "with reset")
#idx("rand", sub: "with reset")
reset a random-number generator to produce
a sequence starting from a given value. Design a new
#py("rand")
function
that is called with an argument that is either the
string #py("\"generate\"") or the string #py("\"reset\"")
and behaves as follows:
#py("rand(\"generate\")")
produces a new random number;
#py("rand(\"reset\")(")#meta("new-value")#py(")")
resets the internal state variable to the designated #meta("new-value"). Thus, by resetting the
state, one can generate repeatable sequences. These are very handy to have
when testing and debugging programs that use random numbers.
])

#idx("reassignment", sub: "benefits of")
