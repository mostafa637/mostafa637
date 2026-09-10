// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Instructions and Their Execution Functions], label-name: <sec:ex-proc>)

#idx("execution function", sub: "in register-machine simulator")
#idx("register-machine language", sub: "instructions")

The assembler calls
#py("make_execution_function")
to generate the execution
function
for a controller instruction.
Like the #py("analyze")
function
in the evaluator of section @sec:separating-analysis,
this dispatches on the type of instruction to generate the appropriate
execution
function.

The details of these execution functions determine the
meaning of the individual instructions in the register-machine language.

#idx("makeexecutionfunction", decl: true)
#snippet(```python
def make_execution_function(inst, labels, machine,
                            pc, flag, stack, ops):
    inst_type = type(inst)
    return (make_assign_ef(inst, machine, labels, ops, pc) if inst_type == "assign"
            else make_test_ef(inst, machine, labels, ops, flag, pc) if inst_type == "test"
            else make_branch_ef(inst, machine, labels, flag, pc) if inst_type == "branch"
            else make_go_to_ef(inst, machine, labels, pc) if inst_type == "go_to"
            else make_save_ef(inst, machine, stack, pc) if inst_type == "save"
            else make_restore_ef(inst, machine, stack, pc) if inst_type == "restore"
            else make_perform_ef(inst, machine, labels, ops, pc) if inst_type == "perform"
            else error("unknown instruction type -- assemble", inst))
```)

The elements of the #py("controller") sequence
received by #py("make_machine") and passed
to #py("assemble") are strings (for
labels) and tagged lists (for instructions). The tag in an instruction
is a string that identifies the instruction type, such as
#py("\"go_to\""), and the remaining elements
of the list contains the arguments, such as the destination of the
#py("go_to").
The dispatch in #py("make_execution_function") uses
#idx("type in register machine", decl: true)
#snippet(```python
def type(instruction):
    return head(instruction)
```)

The tagged lists are constructed when the
#py("list") expression that is the third
argument to #py("make_machine") is
evaluated. Each argument to that
#py("list") is either a string (which
evaluates to itself) or a call to a constructor for an instruction
tagged list. For example, #py("assign(\"b\", reg(\"t\"))") calls the constructor
#py("assign") with arguments
#py("\"b\"") and the result of calling the
constructor #py("reg") with the argument
#py("\"t\""). The constructors and their
arguments determine the syntax of the individual instructions in the
register-machine language. The instruction constructors and selectors
are shown below, along with the execution-function generators that use
the selectors.

#subheading([The instruction #py("assign")])

#idx("assign (in register machine)", sub: "simulating")

The
#py("make_assign_ef")
function
makes execution functions for #py("assign")
instructions:
#idx("makeassignef", decl: true)
#snippet(```python
def make_assign_ef(inst, machine, labels, operations, pc):
    target = get_register(machine, assign_reg_name(inst))
    value_exp = assign_value_exp(inst)
    value_fun = (make_operation_exp_ef(value_exp, machine, labels, operations)
                 if is_operation_exp(value_exp)
                 else make_primitive_exp_ef(value_exp, machine, labels))
    def execution_fun():
        set_contents(target, value_fun())
        advance_pc(pc)
    return execution_fun
```)

The function #py("assign") constructs
#py("assign") instructions.
The selectors #py("assign_reg_name") and
#py("assign_value_exp") extract the register name
and value expression from an #py("assign") instruction.

#idx("assign (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "assign", decl: true)#idx("assignregname", decl: true)#idx("assignvalueexp", decl: true)
#snippet(```python
def assign(register_name, source):
    return llist("assign", register_name, source)
def assign_reg_name(assign_instruction):
    return head(tail(assign_instruction))
def assign_value_exp(assign_instruction):
    return head(tail(tail(assign_instruction)))
```)

The function #py("make_assign_ef") looks up the register name
with
#py("get_register")
to produce the target register object. The value expression is passed to
#py("make_operation_exp_ef")
if the value is the result of an operation, and
it is passed
to
#py("make_primitive_exp_ef")
otherwise. These
functions
(shown below)
analyze
the value expression and produce an execution
function
for the value. This is a
function
of no arguments, called
#idx("valuefun")
#py("value_fun"),
which will be evaluated during the simulation to produce the actual
value to be assigned to the register. Notice that the work of looking
up the register name and
analyzing
the value expression is performed
just once, at assembly time, not every time the instruction is
simulated. This saving of work is the reason we use execution
#idx("syntactic analysis, separated from execution", sub: "in register-machine simulator")
functions,
and corresponds directly to the saving in work we obtained by separating
program analysis from execution in the evaluator of
section @sec:separating-analysis.

The result returned by
#py("make_assign_ef")
is the execution
function
for the #py("assign") instruction. When this
function
is called (by the machine model's #py("execute")
function),
it sets the contents of the target register to the result obtained by
executing
#py("value_fun").
Then it advances the #py("pc") to the next instruction
by running the
function
#idx("advancepc", decl: true)
#snippet(```python
def advance_pc(pc):
    set_contents(pc, tail(get_contents(pc)))
```)

The function #py("advance_pc")
is the normal termination for all instructions except
#py("branch") and
#py("go_to").

#subheading([The instructions #py("test"), #py("branch"), and #py("go_to")])

The function #idx("test (in register machine)", sub: "simulating") #py("make_test_ef")
handles #py("test") instructions in a similar way.
It extracts the expression that specifies the condition to be tested and
generates an execution
function
for it. At simulation time, the
function
for the condition is called, the result is assigned to the
#py("flag") register, and the
#py("pc") is advanced:
#idx("maketestef", decl: true)
#snippet(```python
def make_test_ef(inst, machine, labels, operations, flag, pc):
    condition = test_condition(inst)
    if is_operation_exp(condition):
        condition_fun = make_operation_exp_ef(condition, machine, labels, operations)
        def execution_fun():
            set_contents(flag, condition_fun())
            advance_pc(pc)
        return execution_fun
    else:
        error("bad test instruction -- assemble", inst)
```)

The function
#py("test") constructs
#py("test") instructions. The selector
#py("test_condition") extracts the condition
from a test.
#idx("testcondition", decl: true)#idx("test (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "test", decl: true)
#snippet(```python
def test(condition):
    return llist("test", condition)

def test_condition(test_instruction):
    return head(tail(test_instruction))
```)

The execution
function
for a #py("branch") instruction checks the contents of
the #py("flag") register and either sets the contents
of the #py("pc") to the branch destination (if the
branch is taken) or else just advances the #py("pc")
(if the branch is not taken). Notice that the indicated destination in a
#idx("branch (in register machine)", sub: "simulating")
#py("branch") instruction must be a label, and the
#py("make_branch_ef")
function
enforces this. Notice also that the label is looked up at assembly time,
not each time the #py("branch") instruction is
simulated.
#idx("makebranchef", decl: true)
#snippet(```python
def make_branch_ef(inst, machine, labels, flag, pc):
    dest = branch_dest(inst)
    if is_label_exp(dest):
        insts = lookup_label(labels, label_exp_label(dest))
        def execution_fun():
            if get_contents(flag):
                set_contents(pc, insts)
            else:
                advance_pc(pc)
        return execution_fun
    else:
        error("bad branch instruction -- assemble", inst)
```)

The function #py("branch")
constructs #py("branch") instructions. The
selector
#py("branch_dest") extracts
the destination from a branch.
#idx("register-machine language", sub: "branch", decl: true)#idx("branch (in register machine)", sub: "instruction constructor", decl: true)#idx("branchdest", decl: true)
#snippet(```python
def branch(label):
    return llist("branch", label)

def branch_dest(branch_instruction):
    return head(tail(branch_instruction))
```)

A
#idx("goto (in register machine)", sub: "simulating")
#py("go_to")
instruction is similar to a branch, except that the destination may be
specified either as a label or as a register, and there is no condition to
check—the #py("pc") is always set to the
new destination.
#idx("makegotoef", decl: true)
#snippet(```python
def make_go_to_ef(inst, machine, labels, pc):
    dest = go_to_dest(inst)
    if is_label_exp(dest):
        insts = lookup_label(labels, label_exp_label(dest))
        return lambda: set_contents(pc, insts)
    elif is_register_exp(dest):
        reg = get_register(machine, register_exp_reg(dest))
        return lambda: set_contents(pc, get_contents(reg))
    else:
        error("bad go_to instruction -- assemble", inst)
```)

The function #py("go_to") constructs
#py("go_to") instructions. The selector
#py("go_to_dest") extracts the destination from a
#py("go_to") instruction.
#idx("goto (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "goto", decl: true)#idx("gotodest", decl: true)
#snippet(```python
def go_to(label):
    return llist("go_to", label)

def go_to_dest(go_to_instruction):
    return head(tail(go_to_instruction))
```)

#subheading([Other instructions])

The stack instructions
#py("save") and #py("restore")
simply use the stack with the designated register and advance the
#py("pc"):
#idx("makesaveef", decl: true)#idx("makerestoreef", decl: true)#idx("save (in register machine)", sub: "simulating")#idx("restore (in register machine)", sub: "simulating")
#snippet(```python
def make_save_ef(inst, machine, stack, pc):
    reg = get_register(machine, stack_inst_reg_name(inst))
    def execution_fun():
        push(stack, get_contents(reg))
        advance_pc(pc)
    return execution_fun
def make_restore_ef(inst, machine, stack, pc):
    reg = get_register(machine, stack_inst_reg_name(inst))
    def execution_fun():
        set_contents(reg, pop(stack))
        advance_pc(pc)
    return execution_fun
```)

The functions #py("save") and
#py("restore") construct
#py("save") and #py("restore") instructions. The
selector
#py("stack_inst_reg_name")
extracts the register name from such instructions.
#idx("save (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "save", decl: true)#idx("restore (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "restore", decl: true)#idx("stackinstregname", decl: true)
#snippet(```python
def save(reg):
    return llist("save", reg)

def restore(reg):
    return llist("restore", reg)

def stack_inst_reg_name(stack_instruction):
    return head(tail(stack_instruction))
```)

The final instruction type, handled by
#idx("perform (in register machine)", sub: "simulating")
#py("make_perform_ef"),
generates an execution
function
for the action to be performed. At simulation time, the action
function
is executed and the #py("pc") advanced.
#idx("makeperformef", decl: true)
#snippet(```python
def make_perform_ef(inst, machine, labels, operations, pc):
    action = perform_action(inst)
    if is_operation_exp(action):
        action_fun = make_operation_exp_ef(action, machine, labels, operations)
        def execution_fun():
            action_fun()
            advance_pc(pc)
        return execution_fun
    else:
        error("bad perform instruction -- assemble", inst)
```)

The function #py("perform")
constructs #py("perform") instructions. The
selector
#py("perform_action") extracts
the action from a #py("perform") instruction.
#idx("perform (in register machine)", sub: "instruction constructor", decl: true)#idx("register-machine language", sub: "perform", decl: true)#idx("performaction", decl: true)
#snippet(```python
def perform(action):
    return llist("perform", action)

def perform_action(perform_instruction):
    return head(tail(perform_instruction))
```)

#subheading([Execution functions for subexpressions])

The value of a
#idx("reg (in register machine)", sub: "simulating")
#py("reg"),
#idx("label (in register machine)", sub: "simulating")
#py("label"), or
#idx("constant (in register machine)", sub: "simulating")
#py("constant")
expression may be needed for assignment to a register
(#py("make_assign_ef"), above)
or for input to an operation
(#py("make_operation_exp_ef"),
below). The following
function
generates execution
functions
to produce values for these expressions during the simulation:
#idx("makeprimitiveexpef", decl: true)
#snippet(```python
def make_primitive_exp_ef(exp, machine, labels):
    if is_constant_exp(exp):
        c = constant_exp_value(exp)
        return lambda: c
    elif is_label_exp(exp):
        insts = lookup_label(labels, label_exp_label(exp))
        return lambda: insts
    elif is_register_exp(exp):
        r = get_register(machine, register_exp_reg(exp))
        return lambda: get_contents(r)
    else:
        error("unknown expression type -- assemble", exp)
```)

The syntax of #py("reg"),
#py("label"), and #py("constant")
expressions is determined by the following constructor functions, along with
corresponding predicates and selectors.
#idx("reg (in register machine)", decl: true)#idx("register-machine language", sub: "reg", decl: true)#idx("isregisterexp", decl: true)#idx("registerexpreg", decl: true)#idx("constant (in register machine)", decl: true)#idx("register-machine language", sub: "constant", decl: true)#idx("isconstantexp", decl: true)#idx("constantexpvalue", decl: true)#idx("label (in register machine)", decl: true)#idx("register-machine language", sub: "label", decl: true)#idx("islabelexp", decl: true)#idx("labelexplabel", decl: true)
#snippet(```python
def reg(name):
    return llist("reg", name)

def is_register_exp(exp):
    return is_tagged_list(exp, "reg")

def register_exp_reg(exp):
    return head(tail(exp))
def constant(value):
    return llist("constant", value)

def is_constant_exp(exp):
    return is_tagged_list(exp, "constant")

def constant_exp_value(exp):
    return head(tail(exp))
def label(name):
    return llist("label", name)

def is_label_exp(exp):
    return is_tagged_list(exp, "label")

def label_exp_label(exp):
    return head(tail(exp))
```)

The instructions #py("assign"), #py("perform"), and #py("test")
may include the application of a machine operation (specified by an
#idx("op (in register machine)", sub: "simulating")
#py("op") expression) to some operands (specified
by #py("reg") and
#py("constant")
expressions). The following
function
produces an execution
function
for an "operation expression"—a list containing the
operation and operand expressions from the instruction:
#idx("makeoperationexpef", decl: true)
#snippet(```python
def make_operation_exp_ef(exp, machine, labels, operations):
    op = lookup_prim(operation_exp_op(exp), operations)
    afuns = map(lambda e: make_primitive_exp_ef(e, machine, labels),
                operation_exp_operands(exp))
    return lambda: apply_in_underlying_python(op, map(lambda f: f(), afuns))
```)

The syntax of operation expressions is determined by
#idx("op (in register machine)", sub: "simulating", decl: true)#idx("register-machine language", sub: "op", decl: true)#idx("isoperationexp", decl: true)#idx("operationexpop", decl: true)#idx("operationexpoperands", decl: true)
#snippet(```python
def op(name):
    return llist("op", name)

def is_operation_exp(exp):
    return is_pair(exp) and is_tagged_list(head(exp), "op")

def operation_exp_op(op_exp):
    return head(tail(head(op_exp)))

def operation_exp_operands(op_exp):
    return tail(op_exp)
```)

Observe that the treatment of operation expressions is very much like
the treatment of
function
applications by the
#py("analyze_application")
function
in the evaluator of section @sec:separating-analysis in
that we generate an execution
function
for each operand.
At simulation time, we call the operand functions and apply the Python function
that simulates the operation to the resulting values.

We make use of the function
#py("apply_in_underlying_python"), as we did
in #py("apply_primitive_function") in
section @sec:running-eval. This is needed to apply
#py("op") to all elements of the argument list
#py("afuns")
produced by the first #py("map"),
as if they were separate arguments to
#py("op"). Without this,
#py("op") would have been restricted to be a unary
function.

The simulation
function
is found by looking up the operation name in the operation table for the
machine:
#idx("lookupprim", decl: true)
#snippet(```python
def lookup_prim(symbol, operations):
    val = assoc(symbol, operations)
    return (error("unknown operation -- assemble", symbol) if is_undefined(val)
            else head(tail(val)))
```)

#exercise(label-name: <ex:5_9>, [
The treatment of machine operations above permits them to operate
on labels as well as on constants and the contents of registers.
Modify the expression-processing
functions
to enforce the condition that operations can be used only with registers
and constants.
])

#exercise(label-name: <ex:stack-behavior>, [
When we introduced
#idx("save (in register machine)")
#py("save") and
#idx("restore (in register machine)")
 in
section @sec:stack-recursion, we didn't specify
what would happen if you tried to restore a register that was not the last
one saved, as in the sequence

#snippet(```python
save(y)
save(x)
restore(y)
```)

There are several reasonable possibilities for the meaning of
#py("restore"):

+ #py("restore(y)") puts into #py("y") the last value saved on the stack, regardless of what register that value came from. This is the way our simulator behaves. Show how to take advantage of this behavior to eliminate one instruction from the Fibonacci machine of section @sec:stack-recursion (figure @fig:fib-machine).
+ #py("restore(y)") puts into #py("y") the last value saved on the stack, but only if that value was saved from #py("y"); otherwise, it signals an error. Modify the simulator to behave this way. You will have to change #py("save") to put the register name on the stack along with the value.
+ #py("restore(y)") puts into #py("y") the last value saved from #py("y") regardless of what other registers were saved after #py("y") and not restored. Modify the simulator to behave this way. You will have to associate a separate stack with each register. You should make the #py("initialize_stack") operation initialize all the register stacks.
])

#exercise(label-name: <ex:simulated-data-paths>, [
The simulator can be used to help determine the data paths required
for implementing a machine with a given controller. Extend
the assembler to store the following information in the machine model:

- a list of all instructions, with duplicates removed, sorted by instruction type (#py("assign"), #py("go_to"), and so on);
- a list (without duplicates) of the registers used to hold entry points (these are the registers referenced by #py("go_to") instructions);
- a list (without duplicates) of the registers that are #py("save")d or #py("restore")d;
- for each register, a list (without duplicates) of the sources from which it is assigned (for example, the sources for register #py("val") in the factorial machine of figure @fig:fact-machine are #py("constant(1)") and #py("llist(op(\"*\"), reg(\"n\"), reg(\"val\"))")).

Extend the message-passing interface to the machine to provide access to
this new information. To test your analyzer, define the Fibonacci machine
from figure @fig:fib-machine and examine the lists you
constructed.
])

#exercise(label-name: <ex:5_12>, [
Modify the simulator so that it uses the controller sequence to determine
what registers the machine has rather than requiring a list of registers as
an argument to
#py("make_machine").
Instead of preallocating the registers in
#py("make_machine"),
you can allocate them one at a time when they are first seen during assembly
of the instructions.
])

#idx("register-machine language", sub: "instructions")
#idx("execution function", sub: "in register-machine simulator")
