// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Monitoring Machine Performance], label-name: <sec:monitor>)

#idx("register machine", sub: "monitoring performance")

Simulation is useful not only for verifying the correctness of a
proposed machine design but also for measuring the machine's
#idx("simulation", sub: "for monitoring performance of register machine")
performance. For example, we can install in our simulation program a
"meter" that measures the number of stack operations used in a
computation. To do this, we modify our simulated stack to keep track
of the number of times registers are saved on the stack and the
maximum depth reached by the stack, and add a message to the stack's
interface that prints the statistics, as shown below.
We also add an operation to the basic machine model to print the
stack statistics, by initializing
#py("the_ops")
in
#py("make_new_machine")
to
#idx("initializestack operation in register machine")#idx("printstackstatistics operation in register machine")
#snippet(```python
llist(llist("initialize_stack", lambda: stack("initialize")),
     llist("print_stack_statistics", lambda: stack("print_statistics")))
```)

Here is the new version of
#py("make_stack"):
#idx("makestack", sub: "with monitored stack", decl: true)
#snippet(```python
def make_stack():
    stack = None
    number_pushes = 0
    max_depth = 0
    current_depth = 0
    def push(x):
        nonlocal stack, number_pushes, current_depth, max_depth
        stack = pair(x, stack)
        number_pushes = number_pushes + 1
        current_depth = current_depth + 1
        max_depth = math_max(current_depth, max_depth)
        return "done"
    def pop():
        nonlocal stack, current_depth
        if is_null(stack):
            error("empty stack -- pop")
        else:
            top = head(stack)
            stack = tail(stack)
            current_depth = current_depth - 1
            return top
    def initialize():
        nonlocal stack, number_pushes, max_depth, current_depth
        stack = None
        number_pushes = 0
        max_depth = 0
        current_depth = 0
        return "done"
    def print_statistics():
        display("total pushes = " + stringify(number_pushes))
        display("maximum depth = " + stringify(max_depth))
    def dispatch(message):
        return (push if message == "push"
                else pop() if message == "pop"
                else initialize() if message == "initialize"
                else print_statistics() if message == "print_statistics"
                else error("unknown request -- stack", message))
    return dispatch
```)

Exercises @ex:instruction-count
through @ex:breakpoints
describe other useful monitoring and debugging features that can be
added to the register-machine simulator.

#exercise(label-name: <ex:measure-fact>, [
Measure the number of pushes and the maximum stack depth required to
compute
#idx("factorial", sub: "stack usage, register machine")
$n!$ for various small values of
$n$ using the factorial
machine shown in figure @fig:fact-machine. From your
data determine formulas in terms of $n$ for the
total number of push operations and the maximum stack depth used in
computing $n!$ for any
$n > 1$. Note that each of these is a linear
function of $n$ and is thus determined by two
constants. In order to get the statistics printed, you will have to augment
the factorial machine with instructions to initialize the stack and print
the statistics. You may want to also modify the machine so that it
repeatedly reads a value for $n$, computes the
factorial, and prints
the result (as we did for the GCD machine in
figure @fig:gcd-with-io), so that you will not have to
repeatedly invoke
#py("get_register_contents"),
#py("set_register_contents"),
and
#py("start").
])

#exercise(label-name: <ex:instruction-count>, [
Add
#idx("instruction counting")
#emph[instruction counting]
to the register machine simulation.
That is, have the machine model keep track of the number of
instructions executed. Extend the machine model's interface to
accept a new message that prints the value of the instruction count and
resets the count to zero.
])

#exercise(label-name: <ex:reg-machine-instruction-trace>, [
Augment the simulator to provide for
#idx("instruction tracing")
#idx("tracing", sub: "instruction execution")
#emph[instruction tracing].
That is, before each instruction is executed, the simulator should print
the  instruction. Make the machine model accept
#py("trace_on")
and
#py("trace_off")
messages to turn tracing on and off.
])

#exercise(label-name: <ex:5_16>, [
Extend the instruction tracing of
exercise @ex:reg-machine-instruction-trace so that
before printing an instruction, the simulator prints any labels that
immediately precede that instruction in the controller sequence. Be
careful to do this in a way that does not interfere with instruction
counting (exercise @ex:instruction-count).
You will have to make the simulator retain the necessary label information.
])

#exercise(label-name: <ex:5_17>, [
Modify the
#py("make_register")
function
of section @sec:machine-model so that registers can be
#idx("register(s)", sub: "tracing")
#idx("tracing", sub: "register assignment")
traced. Registers should accept messages that turn tracing on and off. When
a register is traced, assigning a value to the register should print the
name of the register, the old contents of the register, and the new contents
being assigned. Extend the interface to the machine model to permit you to
turn tracing on and off for designated machine registers.
])

#exercise(label-name: <ex:breakpoints>, [
Alyssa P. Hacker wants a
#idx("breakpoint")
#emph[breakpoint] feature in the simulator to help her debug her machine
designs. You have been hired to install this feature for her. She wants to
be able to specify a place in the controller sequence where the simulator
will stop and allow her to examine the state of the machine. You are to
implement a
function

#syntax("
set_breakpoint(", meta("machine"), ", ", meta("label"), ", ", meta("n"), ")
      ")

that sets a breakpoint just before the $n$th
instruction after the given label. For example,

#snippet(```python
set_breakpoint(gcd_machine, "test_b", 4)
```)

installs a breakpoint in
#py("gcd_machine")
just before the assignment to register #py("a").
When the simulator reaches the breakpoint it should print the label and the
offset of the breakpoint and stop executing instructions. Alyssa can then
use
#py("get_register_contents")
and
#py("set_register_contents")
to manipulate the state of the simulated machine. She should then be able
to continue execution by saying

#syntax("
proceed_machine(", meta("machine"), ")
      ")

She should also be able to remove a specific breakpoint by means of

#syntax("
cancel_breakpoint(", meta("machine"), ", ", meta("label"), ", ", meta("n"), ")
      ")

or to remove all breakpoints by means of

#syntax("
cancel_all_breakpoints(", meta("machine"), ")
      ")
])

#idx("register machine", sub: "simulator")
#idx("register-machine simulator")
#idx("simulation", sub: "of register machine")
#idx("register machine", sub: "monitoring performance")
