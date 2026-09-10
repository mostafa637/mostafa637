// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([The Assembler], label-name: <sec:assembler>)

#idx("assembler")

The assembler transforms the sequence of controller
instructions
for a machine into a corresponding list of machine instructions, each with its
execution
function.
Overall, the assembler is much like the evaluators we studied in
chapter @chap:meta—there is an input language (in this case, the
register-machine language) and we must perform an appropriate action for each
type of component in the language.

The technique of producing an execution
function
for each instruction is just what we used in
section @sec:separating-analysis to speed
up the evaluator by separating analysis from runtime execution. As we
saw in chapter @chap:meta, much useful
#idx("syntactic analysis, separated from execution", sub: "in register-machine simulator")
analysis of
Python
expressions could
be performed without knowing the actual values of
names.
Here, analogously, much useful analysis of register-machine-language
expressions can be performed without knowing the actual contents of
machine registers. For example, we can replace references to
registers by pointers to the register objects, and we can
replace references to labels by pointers to the place in the
instruction sequence that the label designates.

Before it can generate the instruction execution
functions,
the assembler must know what all the labels refer to, so it begins by
scanning the controller sequence to separate the labels from the
instructions. As it scans the controller, it constructs both a list of
instructions and a table that associates each label with a pointer
into that list. Then the assembler augments the instruction list by
inserting the execution
function
for each instruction.

The #py("assemble")
function
is the main entry to the assembler. It takes the controller
sequence and the
machine model as arguments and returns the instruction sequence to be stored
in the model.
The function #py("assemble")
calls
#py("extract_labels")
to build the initial instruction list and label table from the supplied
controller. The second argument
to
#py("extract_labels")
is a
function
to be called to process these results: This
function
uses
#py("update_insts")
to generate the instruction execution
functions
and insert them into the instruction list, and returns the modified list.
#idx("assemble", decl: true)
#snippet(```python
def assemble(controller, machine):
    def receive(insts, labels):
        update_insts(insts, labels, machine)
        return insts
    return extract_labels(controller, receive)
```)

The function #py("extract_labels") takes
a list #py("controller")
and a function #py("receive")
as arguments. The function #py("receive") will
be called with two values: (1) a list #py("insts")
of instruction data structures, each containing an instruction from
#py("controller"); and (2) a table called
#py("labels"), which associates each label from
#py("controller") with the position in the list
#py("insts") that the label designates.

#idx("extractlabels", decl: true)
#snippet(```python
def extract_labels(controller, receive):
    def incorporate(insts, labels):
        next_element = head(controller)
        return (receive(insts,
                        pair(make_label_entry(next_element, insts),
                             labels))
                if is_string(next_element)
                else receive(pair(make_inst(next_element), insts),
                             labels))
    return (receive(None, None) if is_null(controller)
            else extract_labels(tail(controller), incorporate))
```)

The function #py("extract_labels")
works by sequentially scanning the elements of the
#py("controller")
and accumulating the
#py("insts") and the
#py("labels"). If an element is a
string
(and thus a label) an appropriate entry is added to the
#py("labels") table. Otherwise the element is
accumulated onto the #py("insts")
list.#footnote[Using the
#idx("receive function")
#py("receive")
function
here is a way to get
#py("extract_labels")
to effectively return two
values—#py("labels") and
#py("insts")—without explicitly making a
compound data structure to hold them. An alternative implementation, which
returns an explicit pair of values, is
#idx("extractlabels", decl: true)
#snippet(```python
def extract_labels(controller):
    if is_null(controller):
        return pair(None, None)
    else:
        result = extract_labels(tail(controller))
        insts = head(result)
        labels = tail(result)
        next_element = head(controller)
        return (pair(insts, pair(make_label_entry(next_element, insts), labels))
                if is_string(next_element)
                else pair(pair(make_inst(next_element), insts), labels))
```)

which would be called by #py("assemble") as follows:
#idx("assemble", decl: true)
#snippet(```python
def assemble(controller, machine):
    result = extract_labels(controller)
    insts = head(result)
    labels = tail(result)
    update_insts(insts, labels, machine)
    return insts
```)

You can consider our use of #py("receive") as
demonstrating an elegant way to
#idx("returning multiple values")

return multiple values, or simply an excuse
to show off a programming trick. An argument like
#py("receive") that is the next
function
to be invoked is called a
#idx("continuation", sub: "in register-machine simulator")
"continuation." Recall that we
also used continuations to implement the backtracking control
structure in the #py("amb") evaluator in
section @sec:amb-implementation.]

The function #py("update_insts")
modifies the instruction list, which initially contains only
the controller instructions,
to include the corresponding execution
functions:
#idx("updateinsts", decl: true)
#snippet(```python
def update_insts(insts, labels, machine):
    pc = get_register(machine, "pc")
    flag = get_register(machine, "flag")
    stack = machine("stack")
    ops = machine("operations")
    return for_each(
        lambda inst: set_inst_execution_fun(
            inst,
            make_execution_function(inst_controller_instruction(inst),
                                    labels, machine, pc, flag, stack, ops)),
        insts)
```)

The machine instruction data structure simply pairs the
controller
instruction  with the corresponding execution
function.
The execution
function
is not yet available when
#py("extract_labels")
constructs the instruction, and is inserted later by
#py("update_insts").
#idx("makeinst", decl: true)#idx("instcontrollerinstruction", decl: true)#idx("instexecutionfun", decl: true)#idx("setinstexecutionfun", decl: true)
#snippet(```python
def make_inst(inst_controller_instruction):
    return pair(inst_controller_instruction, None)
def inst_controller_instruction(inst):
    return head(inst)
def inst_execution_fun(inst):
    return tail(inst)
def set_inst_execution_fun(inst, fun):
    set_tail(inst, fun)
```)

The
controller instruction
is not used by our simulator, but is handy to keep
around for debugging (see
exercise @ex:reg-machine-instruction-trace).

Elements of the label table are pairs:
#idx("makelabelentry", decl: true)
#snippet(```python
def make_label_entry(label_name, insts):
    return pair(label_name, insts)
```)

Entries will be looked up in the table with
#idx("lookuplabel", decl: true)
#snippet(```python
def lookup_label(labels, label_name):
    val = assoc(label_name, labels)
    return (error("undefined label -- assemble", label_name) if is_undefined(val)
            else tail(val))
```)

#exercise(label-name: <ex:5_8>, [
The following register-machine code is ambiguous, because the label
#py("here") is defined more than once:

#snippet(```python
"start",
  go_to(label("here")),
"here",
  assign("a", constant(3)),
  go_to(label("there")),
"here",
  assign("a", constant(4)),
  go_to(label("there")),
"there",
```)

With the simulator as written, what will the contents of register
#py("a") be when control reaches
#py("there")? Modify the
#py("extract_labels")
function
so that the assembler will signal an error if the same label
name is used to indicate two different locations.
])

#idx("assembler")
