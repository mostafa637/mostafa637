// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Streams and Delayed Evaluation], label-name: <sec:streams-and-delayed-evaluation>)

#idx("stream(s)", sub: "delayed evaluation and")
#idx("delayed evaluation", sub: "streams and")

The #py("integral")
function
at the end of the preceding section shows how we can use streams to model
signal-processing systems that contain
#idx("feedback loop, modeled with streams")
feedback loops. The feedback loop for the adder shown in
figure @fig:integral is modeled by the fact that
#idx("integral", sub: "need for delayed evaluation")
#py("integral")'s
internal stream
#py("integ")
is defined in terms of itself:

#snippet(```python
integ = pair(initial_value,
             lambda: add_streams(scale_stream(integrand, dt),
                                 integ))
```)

The interpreter's ability to deal with such an implicit definition
depends on the delay resulting from wrapping the call to
#py("add_streams") in a lambda expression.
Without this delay, the interpreter could not
construct #py("integ") before evaluating the call
to #py("add_streams"), which would require
that #py("integ") already be defined.
In general, such a delay is crucial for using streams to model
signal-processing systems that contain loops. Without a delay,
our models would have to be formulated so that the inputs to any
signal-processing component would be fully evaluated before the output
could be produced. This would outlaw loops.

Unfortunately, stream models of systems with loops may require uses of a
delay beyond the stream programming pattern seen so far. For instance,
figure @fig:analog-computer shows a
signal-processing system for solving the
#idx("differential equation")
differential equation $d y/d t=f(y)$ where
$f$ is a given function. The figure shows a
mapping component, which applies $f$ to its
input signal, linked in a feedback loop to an integrator in a manner
very similar to that of the analog computer circuits that are actually
used to solve such equations.

#sicp-figure(image("/images/img_original/ch3-Z-G-52.svg", width: 70%), caption: [An #idx("analog computer") "analog computer circuit" that solves the equation $d y/d t = f(y)$.], label-name: <fig:analog-computer>)

Assuming we are given an initial value $y_(0)$ for
$y$, we could try to model this system using the
function
#idx("solve differential equation", decl: true)
#snippet(```python
def solve(f, y0, dt):
    y = integral(dy, y0, dt)
    dy = stream_map(f, y)
    return y
```)

This
function
does not work, because in the first line of
#py("solve") the call to
#py("integral") requires that the input
#py("dy") be defined, which does not happen until the
second line of #py("solve").

On the other hand, the intent of our definition does make sense, because we
can, in principle, begin to generate the #py("y")
stream without knowing #py("dy").
Indeed, #py("integral") and many other stream operations can generate part of the answer given only partial information about the arguments.
For #py("integral"), the first element of the output
stream is the specified #py("initial_value"). Thus,
we can generate the first element of the output stream without evaluating
the integrand #py("dy"). Once we know the first
element of #py("y"), the
#py("stream_map")
in the second line of #py("solve") can begin working
to generate the first element of #py("dy"), which will
produce the next element of #py("y"), and so on.

To take advantage of this idea, we will redefine
#py("integral") to expect the integrand stream to be a
#idx("delayed argument")
#idx("argument(s)", sub: "delayed")
#idx("delayed expression", sub: "explicit")
#emph[delayed argument].
The function #py("integral") will force
the integrand to be evaluated only when it is required to generate more than
the first element of the output stream:

#idx("integral", sub: "with delayed argument", decl: true)
#snippet(```python
def integral(delayed_integrand, initial_value, dt):
    integ = pair(initial_value,
                 lambda: add_streams(
                             scale_stream(delayed_integrand(), dt),
                             integ))
    return integ
```)

Now we can implement our #py("solve")
function
by delaying the evaluation of #py("dy") in the
declaration of
#py("y"):
#idx("solve differential equation", decl: true)
#snippet(```python
def solve(f, y0, dt):
    y = integral(lambda: dy, y0, dt)
    dy = stream_map(f, y)
    return y
```)

In general, every caller of #py("integral") must now
delay
the integrand argument. We can demonstrate that the
#py("solve")
function
works by approximating
#idx("e", sub: "as solution to differential equation", sort: "e")
$e approx 2.718$ by computing the value at
$y=1$ of the solution to the differential
equation $d y/d t=y$ with initial condition
$y(0)=1$:#footnote[To complete in reasonable time, this calculation requires the use of the memoization optimization from section @sec:delayed-lists in #py("integral") and in the function #py("add_streams") used in #py("integral") (using the function #py("stream_map_2_optimized") as suggested in exercise @ex:fib-stream-efficiency).]

#snippet(```python
print(stream_ref(solve(lambda y: y, 1, 0.001), 1000))
```)

#output(```python
print(stream_ref(solve(lambda y: y, 1, 0.001), 1000))
```)

#exercise(label-name: <ex:integral>, [
The #py("integral")
function
used above was analogous to the "implicit" definition of the
infinite stream of integers in
section @sec:infinite-streams. Alternatively, we can
give a definition of #py("integral") that is more
like #py("integers-starting-from") (also in
section @sec:infinite-streams):
#idx("integral", decl: true)
#snippet(```python
def integral(integrand, initial_value, dt):
    return pair(initial_value,
                None
                if is_none(integrand)
                else integral(stream_tail(integrand),
                              dt * head(integrand) + initial_value,
                              dt))
```)

When used in systems with loops, this
function
has the same problem
as does our original version of #py("integral").
Modify the
function
so that it expects the #py("integrand") as a
delayed argument and hence can be used in the
#py("solve")
function
shown above.
])

#exercise(label-name: <ex:2nd-order>, [
Consider the problem of designing a signal-processing system to study
the homogeneous
#idx("differential equation", sub: "second-order")
second-order linear differential equation

$ mat(delim: #none, frac(d^(2) y, d t^(2))-a f r a c(d y, d t)-b y, =, 0) $

The output stream, modeling $y$, is generated by
a network that contains a loop. This is because the value of
$d^(2)y/d t^(2)$ depends upon the values of
$y$ and $d y/d t$ and
both of these are determined by integrating
$d^(2)y/d t^(2)$. The diagram we would like to
encode is shown in figure @fig:2nd-order. Write a
function #py("solve_2nd")
that takes as arguments the constants $a$,
$b$, and $d t$ and the
initial values $y_(0)$ and
$d y_(0)$ for $y$ and
$d y/d t$ and generates the stream of successive
values of $y$.

#sicp-figure(image("/images/img_original/ch3-Z-G-53.svg", width: 70%), caption: [Signal-flow diagram for the solution to a second-order linear differential equation.], label-name: <fig:2nd-order>)
])

#exercise(label-name: <ex:3_79>, [
Generalize the
#idx("differential equation", sub: "second-order")
#py("solve_2nd") function
of exercise @ex:2nd-order so that it can be used to
solve general second-order differential equations
$d^(2) y/d t^(2)=f(d y/d t, thin y)$.
])

#sicp-figure(image("/images/img_original/ch3-Z-G-58.svg", width: 70%), caption: [A series RLC circuit.], label-name: <fig:series-rlc>)

#exercise(label-name: <ex:rlc_circuit>, [
A #emph[series RLC circuit]
#idx("RLC circuit")
#idx("circuit", sub: "modeled with streams")
#idx("electrical circuits, modeled with streams")
consists of a resistor, a capacitor, and an
inductor connected in series, as shown in
figure @fig:series-rlc. If
$R$, $L$, and
$C$ are the resistance, inductance, and
capacitance, then the relations between voltage
($v$) and current
($i$) for the three components are described
by the equations

$ mat(delim: #none, v_(R), =, i_(R) R; v_(L), =, L f r a c(d i_(L), d t); i_(C), =, C f r a c(d v_(C), d t)) $

and the circuit connections dictate the relations

$ mat(delim: #none, i_(R), =, i_(L)=-i_(C); v_(C), =, v_(L)+v_(R)) $

Combining these equations shows that the state of the circuit (summarized by
$v_(C)$, the voltage across the capacitor, and
$i_(L)$, the current in the inductor) is
described by the pair of differential equations

$ mat(delim: #none, frac(d v_(C), d t), =, -frac(i_(L), C); frac(d i_(L), d t), =, frac(1, L)v_(C)-frac(R, L)i_(L)) $

The signal-flow diagram representing this system of differential equations
is shown in figure @fig:rlc-signal-flow.

Write a
function
#py("RLC") that takes as arguments the parameters
$R$, $L$, and
$C$ of the circuit and the time increment
$d t$. In a manner similar to that of the
#py("RC")
function
of exercise @ex:rc-circuit,
#py("RLC") should produce a
function
that takes the initial values of the state variables,
$v_(C_(0))$ and
$i_(L_(0))$, and produces a pair
(using #py("pair"))
of the streams of states $v_(C)$ and
$i_(L)$. Using #py("RLC"),
generate the pair of streams that models the behavior of a series RLC
circuit with $R = 1$ ohm,
$C= 0.2$ farad,
$L = 1$ henry,
$d t = 0.1$ second, and initial values
$i_(L_(0)) = 0$ amps and
$v_(C_(0)) = 10$ volts.
])

#idx("stream(s)", sub: "delayed evaluation and")
#idx("delayed evaluation", sub: "streams and")

#subheading([Normal-order evaluation])

#idx("normal-order evaluation", sub: "delayed evaluation and")
#idx("delayed evaluation", sub: "normal-order evaluation and")

The examples in this section illustrate how
delayed evaluation
provides great programming flexibility, but the same examples also show how
this can make our programs more complex. Our new
#py("integral")
function,
for instance, gives us the power to model systems with loops, but we must
now remember that #py("integral") should be called
with a delayed integrand, and every
function
that uses #py("integral") must be aware of this.
In effect, we have created two classes of
functions:
ordinary
functions
and
functions
that take delayed arguments. In general, creating separate classes of
functions
forces us to create separate classes of higher-order
functions
as well.#footnote[This is a small reflection, in
Python,
of the difficulties that
#idx("higher-order functions", sub: "static typing and") #idx("data types", sub: "in statically typed languages") #idx("Pascal, lack of higher-order functions in") #idx("statically typed language") #idx("programming language", sub: "statically typed") early statically
typed languages such as Pascal
had
in coping with higher-order
functions.
In
these
languages, the programmer
had to
specify the data types of the
arguments and the result of each
function:
number, logical value, sequence, and so on. Consequently, we could not
express an abstraction such as "map a given function #py("fun") over all the elements in a sequence" by a single higher-order
function
such as
#py("stream_map").
Rather, we would need a different mapping
function
for each different combination of argument and result data types that might
be specified for a
#py("fun").
Maintaining a practical notion of "data type" in the presence
of higher-order
functions
raises many difficult issues. One way of dealing with this problem is
illustrated by the language
#idx("ML")
ML
#idx("Gordon, Michael")
#idx("Milner, Robin")
#idx("Wadsworth, Christopher")
(Gordon, Milner, and Wadsworth 1979),
whose
"parametrically polymorphic data types"
include templates for
higher-order transformations between data types. Moreover, data types for
most
functions
in ML are never explicitly declared by the programmer. Instead, ML
includes a
#idx("type-inferencing mechanism")
#emph[type-inferencing] mechanism that uses information in the environment
to deduce the data types for newly defined
functions.
Today, statically typed programming languages have evolved to typically support some form of type inference as well as parametric polymorphism, with varying degrees of power. #idx("Haskell") #idx("type(s)", sub: "polymorphic") #idx("polymorphic types") Haskell couples an expressive type system with powerful type inference.]

One way to avoid the need for two different classes of
functions
is to make all
functions
take delayed arguments. We could adopt a model of evaluation in which all
arguments to
functions
are automatically delayed and arguments are forced only when they are
actually needed (for example, when they are required by a primitive
operation). This would transform our language to use normal-order
evaluation, which we first described when we introduced the substitution
model for evaluation in section @sec:substitution-model.
Converting to normal-order evaluation provides a uniform and elegant way to
simplify the use of delayed evaluation, and this would be a natural strategy
to adopt if we were concerned only with stream processing. In
section @sec:lazy-evaluation, after we have studied the
evaluator, we will see how to transform our language in just this way.
Unfortunately, including delays in
function
calls wreaks havoc with our ability to design programs that depend on the
order of events, such as programs that use assignment, mutate data, or
perform input or output.
Even a single delay in the tail of a pair can cause great confusion, as illustrated by exercises @ex:delayed1 and @ex:delayed2.
As far as anyone knows, mutability and delayed evaluation do not mix well
in programming
languages.

#idx("normal-order evaluation", sub: "delayed evaluation and")
#idx("delayed evaluation", sub: "normal-order evaluation and")

#sicp-figure(image("/images/img_original/ch3-Z-G-59.svg", width: 70%), caption: [A signal-flow diagram for the solution to a series RLC circuit.], label-name: <fig:rlc-signal-flow>)
