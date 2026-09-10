// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Designing Register Machines], label-name: <sec:designing-register-machines>)

#idx("register machine", sub: "design of")
#idx("register machine", sub: "data paths")
#idx("register machine", sub: "controller")
#idx("data paths for register machine")
#idx("controller for register machine")
#idx("operation", sub: "in register machine")

To design a register machine, we must design its #emph[data paths]
(registers and operations) and the #emph[controller] that sequences
these operations. To illustrate the design of a simple register
machine, let us examine Euclid's Algorithm, which is used to compute
#idx("gcd", sub: "register machine for")
the greatest common divisor (GCD) of two integers. As we saw in
section @sec:gcd,
#idx("Euclid's Algorithm")
Euclid's Algorithm can be
carried out by an iterative process, as specified by the following
function:

#snippet(```python
def gcd(a, b):
    return a if b == 0 else gcd(b, a % b)
```)

A machine to carry out this algorithm must keep track of two numbers,
$a$ and $b$, so let us
assume that these numbers are stored in two registers with those names. The
basic operations required are testing whether the contents of register
#py("b") is zero and computing the remainder of the
contents of register #py("a") divided by the contents
of register #py("b").

The remainder operation is a complex process, but assume for the moment that
we have a primitive device that computes remainders. On each cycle of the
GCD algorithm, the contents of register #py("a") must
be replaced by the contents of register #py("b"), and
the contents of #py("b") must be replaced by the
remainder of the old contents of #py("a") divided by
the old contents of #py("b"). It would be convenient
if these replacements could be done simultaneously, but in our model of
register machines we will assume that only one register can be assigned a
new value at each step. To accomplish the replacements, our machine will use
a third "temporary" register, which we call
#py("t"). (First the remainder will be placed in
#py("t"), then the contents of
#py("b") will be placed in
#py("a"), and finally the remainder stored in
#py("t") will be placed in
#py("b").)

We can illustrate the registers and operations required for this
machine by using the
#idx("data paths for register machine", sub: "data-path diagram")
#idx("register machine", sub: "data-path diagram")
data-path diagram shown in
figure @fig:gcd-machine. In this
diagram, the registers (#py("a"),
#py("b"), and #py("t")) are
represented by rectangles. Each way to assign a value to a register is
indicated by an arrow with a button—drawn as $⊗$— behind the
head, pointing from the source of data to the register.
When pushed, the button allows
the value at the source to "flow" into the designated register.
The label next to each button is the name we will use to refer to the
button. The names are arbitrary, and can be chosen to have mnemonic value
(for example, #py("a<-b") denotes pushing the
button that assigns the contents of register #py("b")
to register #py("a")). The source of data for a
register can be another register (as in the
#py("a<-b") assignment), an operation result (as in
the #py("t<-r") assignment), or a constant
(a built-in value that cannot be changed, represented in a data-path
diagram by a triangle containing the constant).

An operation that computes a value from constants and the contents
of registers is represented in a data-path diagram by a trapezoid
containing a name for the operation. For example, the box marked
#py("rem") in
figure @fig:gcd-machine represents an operation that
computes the remainder of the contents of the registers
#py("a") and #py("b") to which
it is attached. Arrows (without buttons) point from the input registers and
constants to the box, and arrows connect the operation's output value
to registers. A test is represented by a circle containing a name for the
test. For example, our GCD machine has an operation that tests whether the
contents of register #py("b") is zero. A
#idx("test operation in register machine")
#idx("register machine", sub: "test operation")
test also has arrows from its input
registers and constants, but it has no output
arrows; its value is used by the controller rather than by the data
paths. Overall, the data-path diagram shows the registers and
operations that are required for the machine and how they must be
connected. If we view the arrows as wires and the
$⊗$ buttons as switches, the data-path diagram
is very like the wiring diagram for a machine that could be constructed
from electrical components.

#sicp-figure(image("/images/img_original/Fig5.1a.std.svg", width: 70%), caption: [Data paths for a GCD machine.], label-name: <fig:gcd-machine>)

In order for the data paths to actually compute GCDs, the buttons must
be pushed in the correct sequence. We will describe this sequence in
terms of a
#idx("register machine", sub: "controller diagram")
#idx("controller for register machine", sub: "controller diagram")
controller diagram, as illustrated in
figure @fig:gcd-controller. The elements of the
controller diagram indicate how the data-path components should be operated.
The rectangular boxes in the controller diagram identify data-path buttons
to be pushed, and the arrows describe the sequencing from one step to the
next. The diamond in the diagram represents a decision. One of the two
sequencing arrows will be followed, depending on the value of the data-path
test identified in the diamond. We can interpret the controller in terms
of a physical analogy: Think of the diagram as a maze in which a marble is
rolling. When the marble rolls into a box, it pushes the data-path button
that is named by the box. When the marble rolls into a decision node (such
as the test for
#py("b")$thin =0$), it leaves
the node on the path determined by the result of the indicated test.
Taken together, the data paths and the controller completely describe
a machine for computing GCDs. We start the controller (the rolling
marble) at the place marked #py("start"), after
placing numbers in registers #py("a") and
#py("b"). When the controller reaches
#py("done"), we will find the value of the GCD in
register #py("a").

#idx("gcd", sub: "register machine for")

#exercise(label-name: <ex:iterative-fact>, [
Design a register machine to compute
#idx("factorial", sub: "register machine for (iterative)")
factorials using the iterative
algorithm specified by the following
function.
Draw data-path and
controller diagrams for this machine.

#snippet(```python
def factorial(n):
    def iter(product, counter):
        return product if counter > n else iter(counter * product, counter + 1)
    return iter(1, 1)
```)
])

#idx("data paths for register machine")
#idx("controller for register machine")
#idx("register machine", sub: "data paths")
#idx("register machine", sub: "controller")
#idx("operation", sub: "in register machine")

#include "../../chapter5/section1/subsection1.typ"

#include "../../chapter5/section1/subsection2.typ"

#include "../../chapter5/section1/subsection3.typ"

#include "../../chapter5/section1/subsection4.typ"

#include "../../chapter5/section1/subsection5.typ"
