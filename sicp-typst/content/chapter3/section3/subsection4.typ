// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([A Simulator for Digital Circuits], label-name: <sec:circuit-simulator>)

#idx("digital-circuit simulation")

Designing complex digital systems, such as computers, is an important
engineering activity. Digital systems are constructed by
interconnecting simple elements. Although the behavior of these
individual elements is simple, networks of them can have very complex
behavior. Computer simulation of proposed circuit designs is an
important tool used by digital systems engineers. In this section we
design a system for performing digital logic simulations. This system
typifies a kind of program called an
#idx("event-driven simulation")
#idx("simulation", sub: "event-driven")
#emph[event-driven simulation], in
which actions ("events") trigger further events that happen
at a later time, which in turn trigger more events, and so on.

Our computational model of a circuit will be composed of objects that
correspond to the elementary components from which the circuit is
constructed. There are
#idx("wire, in digital circuit")
#emph[wires], which carry
#idx("signal, digital")
#idx("digital signal")
#emph[digital signals]. A digital signal may at any moment have only one of two
possible values,
0 and 1. There are also various types of digital
#idx("function box, in digital circuit")
#emph[function boxes], which connect wires carrying input signals to other output
wires. Such boxes produce output signals computed from their input
signals. The output signal is
#idx("delay, in digital circuit", sort: "delay")
delayed by a time that depends on the
type of the function box. For example, an
#idx("inverter")
#emph[inverter] is a
primitive function box that inverts its input. If the
input signal to an inverter changes to 0, then one #emph[inverter-delay]
later the inverter will change its output signal to 1. If the input
signal to an inverter changes to 1, then one #emph[inverter-delay] later
the inverter will change its output signal to 0. We draw an inverter
symbolically as in figure @fig:logic-gates. An
#idx("and-gate")
#emph[and-gate],
also shown in figure @fig:logic-gates, is a primitive
function box with two inputs and one output. It drives its output signal
to a value that is the
#idx("logical and (digital logic)")
#emph[logical and] of the inputs. That is, if both
of its input signals become 1, then one #emph[and-gate-delay] time
later the and-gate will force its output signal to be 1; otherwise the
output will be 0. An
#idx("or-gate")
#emph[or-gate] is a similar two-input primitive function
box that drives its output signal to a value that is the
#idx("logical or (digital logic)")
#emph[logical or] of the inputs. That is, the output will become 1 if at least one
of the input signals is 1; otherwise the output will become 0\.

#sicp-figure(image("/images/img_original/ch3-Z-G-24.svg", width: 70%), caption: [Primitive functions in the digital logic simulator.], label-name: <fig:logic-gates>)

We can connect primitive functions together to construct more complex
functions. To accomplish this we wire the outputs of some
function boxes to the inputs of other function boxes. For example,
the
#idx("half-adder")
#idx("adder", sub: "half")
#emph[half-adder] circuit shown in
figure @fig:half-adder consists of an
or-gate, two and-gates, and an inverter. It takes two input signals,
$A$ and $B$, and has
two output signals, $S$ and $C$.
$S$ will become 1
whenever precisely one of $A$ and $B$
is 1, and $C$ will become 1 whenever
$A$ and $B$ are both 1. We can see
from the figure that, because of the
delays involved, the outputs may be generated at different times.
Many of the difficulties in the design of digital circuits arise from
this fact.

#sicp-figure(image("/images/img_original/ch3-Z-G-25.svg", width: 70%), caption: [A half-adder circuit.], label-name: <fig:half-adder>)

We will now build a program for modeling the digital logic circuits we
wish to study. The program will construct computational objects
modeling the wires, which will "hold" the signals. Function
boxes will be modeled by
functions
that enforce the correct relationships among the signals.

One basic element of our simulation will be a
function
#idx("makewire")
#py("make_wire"),
which constructs wires. For example, we can construct six wires as follows:

#snippet(```python
a = make_wire()
b = make_wire()
c = make_wire()
d = make_wire()
e = make_wire()
s = make_wire()
```)

We attach a function box to a set of wires by calling a
function
that constructs that kind of box. The arguments to the constructor
function
are the wires to be attached to the box. For example, given
that we can construct and-gates, or-gates, and inverters, we can wire
together the half-adder shown in figure @fig:half-adder:

#snippet(```python
print(or_gate(a, b, d))
```)

#output(```python
print(or_gate(a, b, d))
```)

#snippet(```python
print(and_gate(a, b, c))
```)

#output(```python
print(and_gate(a, b, c))
```)

#snippet(```python
print(inverter(c, e))
```)

#output(```python
print(inverter(c, e))
```)

#snippet(```python
print(and_gate(d, e, s))
```)

#output(```python
print(and_gate(d, e, s))
```)

Better yet, we can explicitly name this operation by defining a
function
#py("half_adder")
that constructs this circuit, given the four
external wires to be attached to the half-adder:

#idx("half-adder", sub: "halfadder", decl: true)
#snippet(```python
def half_adder(a, b, s, c):
    d = make_wire()
    e = make_wire()
    or_gate(a, b, d)
    and_gate(a, b, c)
    inverter(c, e)
    and_gate(d, e, s)
    return "ok"
```)

The advantage of making this definition is that we can use #py("half_adder") itself as a building block in creating more complex
circuits. Figure @fig:full-adder, for example, shows a
#idx("full-adder")
#idx("adder", sub: "full")
#emph[full-adder] composed of two half-adders and an or-gate.#footnote[A
full-adder is a basic circuit element used in adding two binary
numbers. Here $A$ and $B$
are the bits at corresponding positions in the
two numbers to be added, and $C_(italic("in"))$ is the
carry bit from the addition one place to the right. The circuit generates
$italic("SUM")$, which is the sum bit in the corresponding position, and
$C_(italic("out"))$, which is the
carry bit to be propagated to the left.] We can construct a
full-adder as follows:

#idx("full-adder", sub: "fulladder", decl: true)
#snippet(```python
def full_adder(a, b, c_in, sum, c_out):
    s = make_wire()
    c1 = make_wire()
    c2 = make_wire()
    half_adder(b, c_in, s, c1)
    half_adder(a, s, sum, c2)
    or_gate(c1, c2, c_out)
    return "ok"
```)

Having defined
#py("full_adder")
as a
function,
we can now use it as a building block for creating still more complex
circuits. (For example, see exercise @ex:ripple-carry.)

In essence, our simulator provides us with the tools to construct a
language of circuits. If we adopt the general perspective on
languages with which we approached the study of
Python
in section @sec:elements-of-programming,
we can say that the primitive function boxes form the primitive
elements of the language, that wiring boxes together provides a means
of combination, and that specifying wiring patterns as
functions
serves as a means of abstraction.

#subheading([Primitive function boxes])

The primitive function boxes
#idx("digital-circuit simulation", sub: "primitive function boxes")
implement the "forces" by which a
change in the signal on one wire influences the signals on other
wires. To build function boxes, we use the following operations on
wires:

- #py("get_signal(")#meta("wire")#py(")") #idx("getsignal") \ returns the current value of the signal on the wire.
- #py("set_signal(")#meta("wire")#py(",") #meta("new-value")#py(")"): #idx("setsignal") \ changes the value of the signal on the wire to the new value.
- #py("add_action(")#meta("wire")#py(",") #meta("function-of-no-arguments")#py(")"): #idx("addaction") \ asserts that the designated function should be run whenever the signal on the wire changes value. Such functions are the vehicles by which changes in the signal value on the wire are communicated to other wires.

In addition, we will make use of a
function
#idx("afterdelay")
#py("after_delay")
that takes a time delay and a
function
to be run and executes the given
function
after the given delay.

Using these
functions,
we can define the primitive digital logic functions. To connect an input
to an output through an inverter, we use
#py("add_action")
to associate with the input wire a
function
that will be run whenever the signal on the input wire changes value.
The
function
computes the
#py("logical_not")
of the input signal, and then, after one
#py("inverter_delay"),
sets the output signal to be this new value:
#idx("inverter", sub: "inverter", decl: true)#idx("logicalnot", decl: true)
#snippet(```python
def inverter(input, output):
    def invert_input():
        new_value = logical_not(get_signal(input))
        after_delay(inverter_delay,
                    lambda: set_signal(output, new_value))
    add_action(input, invert_input)
    return "ok"

def logical_not(s):
    return (1
            if s == 0
            else 0
            if s == 1
            else error("invalid signal", s))
```)

#sicp-figure(image("/images/img_original/ch3-Z-G-26.svg", width: 70%), caption: [A full-adder circuit.], label-name: <fig:full-adder>)

An and-gate is a little more complex. The action
function
must be run if
either of the inputs to the gate changes. It computes the
#py("logical_and") (using a function analogous to #py("logical_not"))
of the values of the signals on the input wires and sets up a change
to the new value to occur on the output wire after one
#py("and_gate_delay").
#idx("and-gate", sub: "andgate", decl: true)
#snippet(```python
def and_gate(a1, a2, output):
    def and_action_function():
        new_value = logical_and(get_signal(a1),
                                get_signal(a2))
        after_delay(and_gate_delay,
                    lambda: set_signal(output, new_value))
    add_action(a1, and_action_function)
    add_action(a2, and_action_function)
    return "ok"
```)

#exercise(label-name: <ex:3_28>, [
Define an
#idx("or-gate", sub: "orgate")
or-gate as a primitive function box. Your
#py("or_gate")
constructor should be similar to
#py("and_gate").
])

#exercise(label-name: <ex:3_29>, [
Another way to construct an
#idx("or-gate", sub: "orgate")
or-gate is as a compound digital logic
device, built from and-gates and inverters. Define a
function
#py("or_gate")
that accomplishes this. What is the delay time of the
or-gate in terms of
#py("and_gate_delay")
and
#py("inverter_delay")?
])

#exercise(label-name: <ex:ripple-carry>, [
Figure @fig:ripple-carry shows a
#idx("ripple-carry adder")
#idx("adder", sub: "ripple-carry")
#emph[ripple-carry adder] formed by stringing
together $n$ full-adders.
This is the simplest form of parallel adder
for adding two $n$-bit binary numbers.
The inputs $A_(1)$,
$A_(2)$,
$A_(3)$, …,
$A_(n)$ and
$B_(1)$,
$B_(2)$,
$B_(3)$, …,
$B_(n)$
are the two binary numbers to be added (each
$A_(k)$ and
$B_(k)$
is a 0 or a 1). The circuit generates
$S_(1)$,
$S_(2)$,
$S_(3)$,
…,
$S_(n)$,
the $n$ bits of the sum, and
$C$, the carry from
the addition. Write a
function
#py("ripple_carry_adder")
that generates this circuit. The
function
should take as arguments three lists of
$n$ wires each—the
$A_(k)$, the
$B_(k)$, and the
$S_(k)$—and
also another wire $C$.
The major drawback of the ripple-carry adder is the need to wait for the
carry signals to propagate. What is the delay needed to obtain the
complete output from an $n$-bit ripple-carry
adder, expressed in terms of the delays for and-gates, or-gates, and
inverters?
])

#sicp-figure(image("/images/img_original/ch3-Z-G-27.svg", width: 70%), caption: [A ripple-carry adder for $n$-bit numbers.], label-name: <fig:ripple-carry>)

#idx("digital-circuit simulation", sub: "primitive function boxes")

#subheading([Representing wires])

A wire
#idx("digital-circuit simulation", sub: "representing wires")
in our simulation will be a computational object with two local
state variables:
a #py("signal_value")
(initially taken to be 0) and a collection of
#py("action_functions")
to be run when the signal changes value. We implement the wire,
using
#idx("message passing", sub: "in digital-circuit simulation")
message-passing style, as
a collection of local
functions
together with a #py("dispatch")
function
that selects the appropriate local operation, just as we did
with the simple bank-account object in section
 @sec:local-state-variables:
#idx("makewire", decl: true)
#snippet(```python
def make_wire():
    signal_value = 0
    action_functions = None
    def set_my_signal(new_value):
        nonlocal signal_value
        if signal_value != new_value:
            signal_value = new_value
            return call_each(action_functions)
        else:
            return "done"
    def accept_action_function(fun):
        nonlocal action_functions
        action_functions = pair(fun, action_functions)
        fun()
    def dispatch(m):
        return (signal_value if m == "get_signal"
                else set_my_signal if m == "set_signal"
                else accept_action_function if m == "add_action"
                else error("unknown operation -- wire", m))
    return dispatch
```)

The local
function
#py("set_my_signal")
tests whether the new signal value changes the signal on the wire.
If so, it runs each of the action
functions,
using the following
function
#py("call_each"),
which calls each of the items in a list of no-argument
functions:
#idx("calleach", decl: true)
#snippet(```python
def call_each(functions):
    if is_none(functions):
        return "done"
    else:
        head(functions)()
        return call_each(tail(functions))
```)

The local
function
#py("accept_action_function")
adds the given
function
to the list of
functions
to be run, and then runs the new
function
once. (See exercise @ex:accept-action.)

With the local #py("dispatch")
function
set up as specified, we can
provide the following
functions
to access the local operations on
wires:#footnote[These
functions
are simply syntactic sugar that allow
#idx("syntactic sugar", sub: "function vs. data as")
#idx("syntax interface")
us to use ordinary
functional
syntax to access the local
functions
of objects. It is striking that we can interchange the role of
"functions"
and
"data" in such a simple way. For example, if we write
#py("wire(\"get_signal\")")
we think of #py("wire") as a
function
that is called with the message
#py("\"get_signal\"")
as input. Alternatively, writing
#py("get_signal(wire)")
encourages us to think of #py("wire") as a data
object that is the input to a
function
#py("get_signal").
The truth of the matter is that, in a language in which we can deal with
functions
as objects, there is no fundamental difference between
"functions"
and "data," and we can choose our syntactic sugar to allow us
to program in whatever style we choose.]<foot:object-syntax>
#idx("getsignal", decl: true)#idx("setsignal", decl: true)#idx("addaction", decl: true)
#snippet(```python
def get_signal(wire):
    return wire("get_signal")

def set_signal(wire, new_value):
    return wire("set_signal")(new_value)

def add_action(wire, action_function):
    return wire("add_action")(action_function)
```)

Wires, which have time-varying signals and may be incrementally attached to
devices, are typical of mutable objects. We have modeled them as
functions
with local state variables that are modified by assignment. When a new
wire is created, a new set of state variables is allocated (by the
#py("let") statements in
#py("make_wire"))
and a new #py("dispatch")
function
is constructed and returned, capturing
the environment with the new state variables.

The wires are shared among the various devices that have been
connected to them. Thus, a change made by an interaction with one
device will affect all the other devices attached to the wire. The
wire communicates the change to its neighbors by calling the action
functions
provided to it when the connections were established.

#idx("digital-circuit simulation", sub: "representing wires")

#subheading([The agenda])

#idx("digital-circuit simulation", sub: "agenda")

The only thing needed to complete the simulator is
#py("after_delay").
The idea here is that we maintain a data structure, called an
#emph[agenda], that contains a schedule of things to do.
The following operations are defined for agendas:

- #py("make_agenda()"): #idx("makeagenda") \ returns a new empty agenda.
- #py("is_empty_agenda(")#meta("agenda")#py(")") #idx("isemptyagenda") \ is true if the specified agenda is empty.
- #py("first_agenda_item(")#meta("agenda")#py(")") #idx("firstagendaitem") \ returns the first item on the agenda.
- #py("remove_first_agenda_item(")#meta("agenda")#py(")") #idx("removefirstagendaitem") \ modifies the agenda by removing the first item.
- #py("add_to_agenda(")#meta("time")#py(",") #meta("action")#py(",") #meta("agenda")#py(")") #idx("addtoagenda") \ modifies the agenda by adding the given action function to be run at the specified time.
- #py("current_time(")#meta("agenda")#py(")") #idx("currenttime") \ returns the current simulation time.

The particular agenda that we use is denoted by
#py("the_agenda").
The
function
#py("after_delay")
adds new elements to
#py("the_agenda"):
#idx("afterdelay", decl: true)
#snippet(```python
def after_delay(delay, action):
    add_to_agenda(delay + current_time(the_agenda),
                  action,
                  the_agenda)
```)

The simulation is driven by the function
#py("propagate"), which executes each
function on
#py("the_agenda")
in sequence.

In general, as the simulation runs, new items
will be added to the agenda, and #py("propagate")
will continue the simulation as long as there are items on the agenda:
#idx("propagate", decl: true)
#snippet(```python
def propagate():
    if is_empty_agenda(the_agenda):
        return "done"
    else:
        first_item = first_agenda_item(the_agenda)
        first_item()
        remove_first_agenda_item(the_agenda)
        return propagate()
```)

#idx("digital-circuit simulation", sub: "agenda")

#subheading([A sample simulation])

#idx("digital-circuit simulation", sub: "sample simulation")
#idx("half-adder", sub: "simulation of")

The following
function,
which places a "probe" on a wire, shows the simulator in
action. The probe tells the wire that, whenever its signal changes value,
it should print the new signal value, together with the current time and
a name that identifies the
wire.
#idx("probe", sub: "in digital-circuit simulator", decl: true)
#snippet(```python
def probe(name, wire):
    add_action(wire,
               lambda: display(name + " " +
                               str(current_time(the_agenda)) +
                               ", new value = " +
                               str(get_signal(wire))))
```)

We begin by initializing the agenda and specifying delays for the
primitive function boxes:

#snippet(```python
the_agenda = make_agenda()
inverter_delay = 2
and_gate_delay = 3
or_gate_delay = 5
```)

Now we define four wires, placing probes on two of them:

#snippet(```python
input_1 = make_wire()
input_2 = make_wire()
sum = make_wire()
carry = make_wire()

probe("sum", sum)
```)

#output(```python
input_1 = make_wire()
input_2 = make_wire()
sum = make_wire()
carry = make_wire()

probe("sum", sum)
```)

#snippet(```python
probe("carry", carry)
```)

#output(```python
probe("carry", carry)
```)

Next we connect the wires in a half-adder circuit (as in
figure @fig:half-adder), set the signal on
#py("input_1")
to 1, and run the simulation:

#snippet(```python
print(half_adder(input_1, input_2, sum, carry))
```)

#output(```python
print(half_adder(input_1, input_2, sum, carry))
```)

#snippet(```python
print(set_signal(input_1, 1))
```)

#output(```python
print(set_signal(input_1, 1))
```)

#snippet(```python
print(propagate())
```)

#output(```python
print(propagate())
```)

The #py("sum") signal changes to 1 at time 8.
We are now eight time units from the beginning of the simulation.
At this point, we can set the signal on
#py("input_2")
to 1 and allow the values to propagate:

#snippet(```python
print(set_signal(input_2, 1))
```)

#output(```python
print(set_signal(input_2, 1))
```)

#snippet(```python
print(propagate())
```)

#output(```python
print(propagate())
```)

The #py("carry") changes to 1 at time 11 and the
#py("sum") changes to 0 at time 16.

#idx("digital-circuit simulation", sub: "sample simulation")
#idx("half-adder", sub: "simulation of")

#exercise(label-name: <ex:accept-action>, [
The internal
function
#py("accept_action_function")
defined in
#idx("makewire")
#py("make_wire")
specifies that when a new action
function
is added to
a wire, the
function
is immediately run. Explain why this initialization
is necessary. In particular, trace through the half-adder example in
the paragraphs above and say how the system's response would differ
if we had defined
#py("accept_action_function")
as

#snippet(```python
def accept_action_function(fun):
    nonlocal action_functions
    action_functions = pair(fun, action_functions)
```)
])

#subheading([Implementing the agenda])

#idx("digital-circuit simulation", sub: "agenda implementation")

Finally, we give details of the agenda data structure, which holds the
functions
that are scheduled for future execution.

The agenda is made up of
#idx("time segment, in agenda")
#emph[time segments]. Each time segment is a
pair consisting of a number (the time) and a
#idx("queue", sub: "in simulation agenda")
queue (see
exercise @ex:agenda-list) that holds the
functions
that are scheduled to be run during that time segment.
#idx("maketimesegment", decl: true)#idx("segmenttime", decl: true)#idx("segmentqueue", decl: true)
#snippet(```python
def make_time_segment(time, queue):
    return pair(time, queue)

def segment_time(s):
    return head(s)

def segment_queue(s):
    return tail(s)
```)

We will operate on the time-segment queues using the queue operations
described in section @sec:queues.

The agenda itself is a one-dimensional
#idx("table", sub: "used in simulation agenda")
table of time segments. It
differs from the tables described in section @sec:tables
in that the segments will be sorted in order of increasing time. In
addition, we store the
#idx("current time, for simulation agenda")
#emph[current time] (i.e., the time of the last action
that was processed) at the head of the agenda. A newly constructed
agenda has no time segments and has a current time of 0:#footnote[The
agenda is a
#idx("headed list")
#idx("list(s)", sub: "headed")
headed list, like the tables in section @sec:tables,
but since the list is headed by the time, we do not need an additional
dummy header (such as the
#py("\"*table*\"") string
used
with tables).]
#idx("makeagenda", decl: true)#idx("currenttime", decl: true)#idx("setcurrenttime", decl: true)#idx("segments", decl: true)#idx("setsegments", decl: true)#idx("firstsegment", decl: true)#idx("restsegments", decl: true)
#snippet(```python
def make_agenda():
    return llist(0)

def current_time(agenda):
    return head(agenda)

def set_current_time(agenda, time):
    set_head(agenda, time)

def segments(agenda):
    return tail(agenda)

def set_segments(agenda, segs):
    set_tail(agenda, segs)

def first_segment(agenda):
    return head(segments(agenda))

def rest_segments(agenda):
    return tail(segments(agenda))
```)

An agenda is empty if it has no time segments:
#idx("isemptyagenda", decl: true)
#snippet(```python
def is_empty_agenda(agenda):
    return is_none(segments(agenda))
```)

To add an action to an agenda, we first check if the agenda is empty.
If so, we create a time segment for the action and install this in
the agenda. Otherwise, we scan the agenda, examining the time of each
segment. If we find a segment for our appointed time, we add the
action to the associated queue. If we reach a time later than the one
to which we are appointed, we insert a new time segment into the
agenda just before it. If we reach the end of the agenda, we must
create a new time segment at the end.
#idx("addtoagenda", decl: true)
#snippet(```python
def add_to_agenda(time, action, agenda):
    def belongs_before(segs):
        return is_none(segs) or time < segment_time(head(segs))
    def make_new_time_segment(time, action):
        q = make_queue()
        insert_queue(q, action)
        return make_time_segment(time, q)
    def add_to_segments(segs):
        if segment_time(head(segs)) == time:
            insert_queue(segment_queue(head(segs)), action)
        else:
            rest = tail(segs)
            if belongs_before(rest):
                set_tail(segs, pair(make_new_time_segment(time, action),
                                    tail(segs)))
            else:
                add_to_segments(rest)
    segs = segments(agenda)
    if belongs_before(segs):
        set_segments(agenda,
                     pair(make_new_time_segment(time, action), segs))
    else:
        add_to_segments(segs)
```)

The
function
that removes the first item from the agenda deletes the
item at the front of the queue in the first time segment. If this
deletion makes the time segment empty, we remove it from the list of
segments:#footnote[Observe that the

conditional statement in this function has a #idx("statement", sub: "pass", decl: true) #idx("pass statement", decl: true) #py("pass") statement as its alternative statement that does nothing.]<foot:one-armed>
#idx("removefirstagendaitem", decl: true)
#snippet(```python
def remove_first_agenda_item(agenda):
    q = segment_queue(first_segment(agenda))
    delete_queue(q)
    if is_empty_queue(q):
        set_segments(agenda, rest_segments(agenda))
    else:
        pass
```)

The first agenda item is found at the head of the queue in the first
time segment. Whenever we extract an item, we also update the current
time:#footnote[In this way, the current time will always be the time
of the action most recently processed. Storing this time at the head
of the agenda ensures that it will still be available even if the
associated time segment has been deleted.]
#idx("firstagendaitem", decl: true)
#snippet(```python
def first_agenda_item(agenda):
    if is_empty_agenda(agenda):
        error("agenda is empty -- first_agenda_item")
    else:
        first_seg = first_segment(agenda)
        set_current_time(agenda, segment_time(first_seg))
        return front_queue(segment_queue(first_seg))
```)

#exercise(label-name: <ex:agenda-list>, [
The
functions
to be run during each time segment of the agenda are kept in a queue.
Thus, the
functions
for each segment are called in the order in which they were added to the
agenda (first in, first out). Explain why this order must be used. In
particular, trace the behavior of an and-gate whose inputs change from
0,1 to 1,0 in the same segment and say how the behavior would differ if
we stored a segment's
functions
in an ordinary list, adding and removing
functions
only at the front (last in, first out).
])

#idx("digital-circuit simulation")
#idx("digital-circuit simulation", sub: "agenda implementation")
