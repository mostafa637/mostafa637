// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Compilation], label-name: <sec:compilation>)

#idx("compiler")

The explicit-control evaluator of
section @sec:eceval is a
register machine whose controller interprets
Python
programs. In this
section we will see how to run
Python
programs on a register machine whose controller is not a
Python
interpreter.

The explicit-control evaluator machine is
#idx("universal machine", sub: "explicit-control evaluator as")
#idx("explicit-control evaluator for Python", sub: "as universal machine")
universal—it
can carry out any computational process that can be described in
Python.
The
evaluator's controller orchestrates the use of its data
paths to perform the desired computation. Thus, the
evaluator's data paths are universal: They are sufficient
to perform any computation we desire, given an appropriate
controller.#footnote[This is a theoretical statement. We are
not claiming
that the evaluator's data paths are a particularly convenient or
efficient set of data paths for a general-purpose computer. For example,
they are not very good for implementing high-performance floating-point
calculations or calculations that intensively manipulate bit
vectors.]

Commercial
#idx("universal machine", sub: "general-purpose computer as")
#idx("general-purpose computer, as universal machine")
general-purpose computers are
register machines organized
around a collection of registers and operations that constitute
an efficient and convenient universal set of data paths.
The controller for a general-purpose machine is an interpreter for
a register-machine language like the one we have been using. This
language is called the
#idx("native language of machine")
#emph[native language] of the machine, or simply
#idx("machine language")
#emph[machine language]. Programs written in machine language are
sequences of instructions that use the machine's data paths.
For example, the
#idx("explicit-control evaluator for Python", sub: "as machine-language program")
explicit-control evaluator's instruction sequence
can be thought of as a machine-language program for a general-purpose
computer rather than as the controller for a specialized interpreter
machine.

#idx("compiler", sub: "interpreter vs.")
#idx("interpreter", sub: "compiler vs.")

There are two common strategies for bridging the gap between
higher-level languages and register-machine languages.
The explicit-control evaluator illustrates the
strategy of interpretation. An interpreter written in the native
language of a machine configures the machine to execute programs
written in a language (called the
#idx("source language")
#emph[source language]) that may
differ from the native language of the machine performing the
evaluation. The primitive
functions
of the source language are implemented as a library of subroutines written
in the native language of the given machine. A program to be interpreted
(called the
#idx("source program")
#emph[source program]) is represented as a data structure. The interpreter
traverses this data structure, analyzing the source program. As it
does so, it simulates the intended behavior of the source program by
calling appropriate primitive subroutines from the library.

In this section, we explore the alternative strategy of #emph[compilation]. A compiler for a given source language and machine
translates a source program into an equivalent program (called the
#idx("object program")
#emph[object program]) written in the machine's native language.
The compiler that we implement in this section translates programs written in
Python
into sequences of instructions to be executed using the explicit-control
evaluator machine's data paths.#footnote[Actually, the machine that
runs compiled code can be simpler than the interpreter machine, because we
#idx("compiler for Python", sub: "register use")
won't use the
#py("comp")
and
#py("unev") registers. The interpreter
used these to hold pieces of unevaluated
components.
With the
compiler, however, these
components
get built into the
compiled code that the register machine will run. For the same
reason,
#idx("compiler for Python", sub: "machine-operation use")
we don't need the machine operations that deal with
component
syntax. But compiled code will use a few additional machine
operations (to represent compiled
function
objects) that didn't
appear in the explicit-control evaluator machine.]

Compared with interpretation, compilation can provide a great increase
in the efficiency of program execution, as we will explain below in
the overview of the compiler.
On the other hand, an interpreter
provides a more powerful environment for interactive program
development and debugging, because the source program being executed
is available at run time to be examined and modified. In addition,
because the entire library of primitives is present, new programs can
be constructed and added to the system during debugging.

In view of the complementary advantages of compilation and
interpretation, modern
program-development environments
pursue a mixed
strategy.
These systems
are generally organized so that interpreted
functions
and compiled
functions
can call each other.
This enables a programmer to compile those parts of a
program that are assumed to be debugged, thus gaining the efficiency
advantage of compilation, while retaining the interpretive mode of execution
for those parts of the program that are in the flux of interactive
development and
debugging.#footnote[Language implementations often delay the compilation of program parts even when they are assumed to be debugged, until there is enough evidence that compiling them would lead to an overall efficiency advantage. The evidence is obtained at run time by monitoring the number of times the program parts are being interpreted. This technique is called #emph[just-in-time compilation].]
In section @sec:interfacing-compiled-code, after
we have implemented the compiler, we will show how to interface it
with our interpreter to produce an integrated
interpreter-compiler

system.

#idx("compiler", sub: "interpreter vs.")
#idx("interpreter", sub: "compiler vs.")
#idx("compiler")

#subheading([An overview of the compiler])

#idx("compiler for Python")
#idx("compiler for Python", sub: "explicit-control evaluator vs.")

Our compiler is much like our interpreter, both in its structure and in
the function it performs. Accordingly, the mechanisms used by the
compiler for analyzing
components
will be similar to those used by
the interpreter. Moreover, to make it easy to interface compiled and
interpreted code, we will design the compiler to generate code that
obeys the same conventions of
#idx("compiler for Python", sub: "register use")
register usage as the interpreter: The
environment will be kept in the #py("env") register,
argument lists will be accumulated in #py("argl"), a
function
to be applied will be in
#py("fun"),
functions
will return their answers in #py("val"),
and the location to which a
function
should return will be kept in
#py("continue").
In general, the compiler translates a source program into an object
program that performs essentially the same register operations as
would the interpreter in evaluating the same source program.

#idx("compiler for Python", sub: "efficiency")

This description suggests a strategy for implementing a rudimentary
compiler: We traverse the
component
in the same way the
interpreter does. When we encounter a register instruction that the
interpreter would perform in evaluating the
component,
we do not
execute the instruction but instead accumulate it into a sequence. The
resulting sequence of instructions will be the object code. Observe
the
#idx("efficiency", sub: "of compilation")
efficiency advantage of compilation over interpretation. Each
time the interpreter evaluates
a component—for example,
#py("f(96, 22)")—it
performs the work of classifying the
component
(discovering that this is a
function
application) and
testing for the end of the
list of argument expressions (discovering that there are two argument expressions).
With a
compiler, the
component
is analyzed only once, when the
instruction sequence is generated at compile time. The object code
produced by the compiler contains only the instructions that evaluate
the
function expression and the two argument expressions,
assemble the argument list, and apply the
function (in #py("fun"))
to the arguments (in #py("argl")).

This is the same kind of optimization we implemented in the
#idx("compiler for Python", sub: "analyzing evaluator vs.")
analyzing evaluator of section @sec:separating-analysis.
But there are further opportunities to gain efficiency in compiled code.
As the interpreter runs, it follows a process that must be applicable
to any
component
in the language. In contrast, a given segment of
compiled code is meant to execute some particular
component.
This can make a big difference, for example in the use of the
stack to save registers. When the interpreter evaluates
a component,
it must be prepared for any contingency. Before evaluating a
subcomponent,
the interpreter saves all registers that will be needed later, because the
subcomponent
might require an arbitrary evaluation.
A compiler, on the other hand, can exploit the structure of the particular
component
it is processing to generate code that avoids
unnecessary stack operations.

As a case in point, consider the
application #py("f(96, 22)").
Before the interpreter evaluates the
function expression of the application,
it prepares
for this evaluation by saving the registers containing the
argument expressions
and the environment, whose values will be needed later. The interpreter then
evaluates the
function expression
to obtain the result in
#py("val"), restores the saved registers, and finally
moves the result from #py("val") to
#py("fun").
However, in the particular expression we
are dealing with, the
function expression
is the
name
#py("f"), whose evaluation is
accomplished by the machine operation
#py("lookup_symbol_value"),
which does not alter any registers. The compiler that we implement in
this section will take advantage of this fact and generate code that
evaluates the
function expression
using the instruction

#snippet(```python
assign("fun",
       list(op("lookup_symbol_value"), constant("f"), reg("env")))
```)

where the argument to #py("lookup_symbol_value") is extracted at compile time from the parser's representation of #py("f(96, 22)").
This code not only avoids the unnecessary saves and
restores but also assigns the value of the lookup directly to
#py("fun"),
whereas the interpreter would obtain the
result in #py("val") and then move this to
#py("fun").

A compiler can also optimize access to the environment. Having
analyzed the code, the compiler can

know in which frame
the value of a particular name
will be located and access that frame directly,
rather than performing the
#py("lookup_symbol_value")
search. We will discuss how to implement such
lexical addressing
in
section @sec:lexical-addressing. Until then, however,
we will focus on the kind of register and stack optimizations described
above. There are many other optimizations that can be performed by a
compiler, such as coding primitive operations "in line" instead
of using a general #py("apply") mechanism (see
exercise @ex:open-code); but we will not emphasize these
here. Our main goal in this section is to illustrate the compilation process
in a simplified (but still interesting) context.

#idx("compiler for Python", sub: "efficiency")
#idx("compiler for Python", sub: "explicit-control evaluator vs.")

#include "../../chapter5/section5/subsection1.typ"

#include "../../chapter5/section5/subsection2.typ"

#include "../../chapter5/section5/subsection3.typ"

#include "../../chapter5/section5/subsection4.typ"

#include "../../chapter5/section5/subsection5.typ"

#include "../../chapter5/section5/subsection6.typ"

#include "../../chapter5/section5/subsection7.typ"
