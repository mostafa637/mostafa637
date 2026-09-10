// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([The Explicit-Control Evaluator], label-name: <sec:eceval>)

#sicp-figure(image("/images/img_original/chip.png", width: 40%), caption: [A silicon-chip implementation of an evaluator for Scheme.], label-name: <fig:Scheme-chip>)

#idx("explicit-control evaluator for Python")

In section @sec:designing-register-machines we saw how to
transform simple
Python
programs into descriptions of register
machines. We will now perform this transformation on a more complex
program, the metacircular evaluator of
sections @sec:core-of-evaluator–@sec:running-eval,
which shows how the behavior of a
Python
interpreter can be described in terms of the
functions #py("evaluate")
and #py("apply").
The #emph[explicit-control evaluator] that we develop in this section shows how the underlying
function-calling
and argument-passing mechanisms used in the
evaluation process can be described in terms of operations on
registers and stacks. In addition, the explicit-control evaluator can
serve as an implementation of a
Python
interpreter, written in a language that is very similar to the native machine
language of conventional computers. The evaluator can be executed by the
register-machine simulator of section @sec:simulator.
Alternatively, it can be used as a starting point for building a
machine-language implementation of a
Python
evaluator, or even a
#idx("Scheme chip")
#idx("integrated-circuit implementation of Scheme")
#idx("chip implementation of Scheme")
#idx("Scheme", sub: "integrated-circuit implementation of")
special-purpose machine for evaluating
Python programs.
Figure @fig:Scheme-chip shows such a hardware
implementation: a silicon chip that acts as an evaluator for
Scheme, the language used in place of Python in the original edition of this book.
The chip designers started with the data-path and controller specifications
for a register machine similar to the evaluator described in this section
and used design automation programs to construct the
integrated-circuit layout.#footnote[See
#idx("Batali, John Dean")
Batali et al. 1982 for more
information on the chip and the method by which it was designed.]

#subheading([Registers and operations])

#idx("explicit-control evaluator for Python", sub: "data paths")
#idx("explicit-control evaluator for Python", sub: "operations")

In designing the explicit-control evaluator, we must specify the
operations to be used in our register machine. We described the
metacircular evaluator in terms of abstract syntax, using
functions
such as
#py("is_literal")
and
#py("make_function").
In implementing the
register machine, we could expand these
functions
into sequences of
elementary list-structure memory operations, and implement these
operations on our register machine. However, this would make our
evaluator very long, obscuring the basic structure with
details. To clarify the presentation, we will include as primitive
operations of the register machine the syntax
functions
given in
section @sec:representing-expressions and the
functions
for representing environments and other runtime data given in
sections @sec:eval-data-structures
and @sec:running-eval.
In order to completely specify an evaluator that could be programmed
in a low-level machine language or implemented in hardware, we would
replace these operations by more elementary operations, using the
list-structure implementation we described in
section @sec:storage-allocation.

Our
Python
evaluator register machine includes a stack and seven
registers:
#idx("explicit-control evaluator for Python", sub: "registers")
#idx("comp register")
#py("comp"),
#idx("env register")
#py("env"),
#idx("val register")
#py("val"),
#idx("continue register", sub: "in explicit-control evaluator")
#py("continue"),
#idx("fun register")
#py("fun"),
#idx("argl register")
#py("argl"), and
#idx("unev register")
#py("unev").
The #py("comp") register
is used to hold the
component
to be evaluated, and #py("env") contains the environment in
which the evaluation is to be performed. At the end of an evaluation,
#py("val") contains the value obtained by evaluating the
component
in the designated environment. The #py("continue") register is
used to implement recursion, as explained in section @sec:stack-recursion. (The evaluator needs to call itself recursively, since
evaluating a component requires evaluating its
subcomponents.) The registers
#py("fun"),
#py("argl"), and #py("unev") are
used in evaluating function applications.

We will not provide a data-path diagram to show how the registers and
operations of the evaluator are connected, nor will we give the
complete list of machine operations. These are implicit in the
evaluator's controller, which will be presented in detail.

#idx("explicit-control evaluator for Python", sub: "data paths")

#include "../../chapter5/section4/subsection1.typ"

#include "../../chapter5/section4/subsection2.typ"

#include "../../chapter5/section4/subsection3.typ"

#include "../../chapter5/section4/subsection4.typ"
