// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([A Register-Machine Simulator], label-name: <sec:simulator>)

#idx("register machine", sub: "simulator")
#idx("register-machine simulator")
#idx("simulation", sub: "of register machine")

In order to gain a good understanding of the design of register machines,
we must test the machines we design to see if they perform as expected.
One way to test a design is to hand-simulate the operation of the
controller, as in exercise @ex:hand-sim. But this is
extremely tedious for all but the simplest machines. In this section we
construct a simulator for machines described in the register-machine
language. The simulator is a
Python
program with
four interface
functions.
The first uses a description of a register
machine to construct a model of the machine (a data structure whose
parts correspond to the parts of the machine to be simulated), and the
other three allow us to simulate the machine by manipulating the
model:

- #py("make_machine(")#meta("register-names")#py(",")#meta("operations")#py(",")#meta("controller")#py(")") #idx("makemachine") \ constructs and returns a model of the machine with the given registers, operations, and controller.
- #py("set_register_contents(")#meta("machine-model")#py(",")#meta("register-name")#py(",")#meta("value")#py(")") #idx("setregistercontents") \ stores a value in a simulated register in the given machine.
- #py("get_register_contents(")#meta("machine-model")#py(",")#meta("register-name")#py(")") #idx("getregistercontents") \ returns the contents of a simulated register in the given machine.
- #py("start(")#meta("machine-model")#py(")") #idx("start register machine") \ simulates the execution of the given machine, starting from the beginning of the controller sequence and stopping when it reaches the end of the sequence.

As an example of how these
functions
are used, we can define
#py("gcd_machine")
to be a model of the GCD machine
of section @sec:register-machine-language as follows:

#idx("gcd", sub: "register machine for")#idx("gcdmachine", decl: true)
#snippet(```python
gcd_machine = make_machine(
    llist("a", "b", "t"),
    llist(llist("rem", lambda a, b: a % b),
         llist("=", lambda a, b: a == b)),
    llist("test_b",
           test(llist(op("="), reg("b"), constant(0))),
           branch(label("gcd_done")),
           assign("t", llist(op("rem"), reg("a"), reg("b"))),
           assign("a", reg("b")),
           assign("b", reg("t")),
           go_to(label("test_b")),
         "gcd_done"))
```)

The first argument to
#py("make_machine")
is a list of register names. The next argument is a table (a list of
two-element lists) that pairs each operation name with a
Python function
that implements the operation (that is, produces the same output value
given the same input values). The last argument specifies the controller
as a list of labels and machine instructions, as in
section @sec:designing-register-machines.

To compute GCDs with this machine, we set the input registers, start the
machine, and examine the result when the simulation terminates:

#snippet(```python
set_register_contents(gcd_machine, "a", 206)
```)

#output(```python
set_register_contents(gcd_machine, "a", 206)
```)

#snippet(```python
set_register_contents(gcd_machine, "b", 40)
```)

#output(```python
set_register_contents(gcd_machine, "b", 40)
```)

#snippet(```python
start(gcd_machine)
```)

#output(```python
start(gcd_machine)
```)

#snippet(```python
get_register_contents(gcd_machine, "a")
```)

#output(```python
get_register_contents(gcd_machine, "a")
```)

This computation will run much more slowly than a
#py("gcd")
function
written in
Python,
because we will simulate low-level machine instructions, such as
#py("assign"), by much more complex operations.

#exercise(label-name: <ex:use-simulator>, [
Use the simulator to test the machines you designed in
exercise @ex:design-reg-machines.
])

#include "../../chapter5/section2/subsection1.typ"

#include "../../chapter5/section2/subsection2.typ"

#include "../../chapter5/section2/subsection3.typ"

#include "../../chapter5/section2/subsection4.typ"
