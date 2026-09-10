// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Structure of the Compiler], label-name: <sec:compiler-structure>)

#idx("compiler for Python", sub: "structure of")

In section @sec:separating-analysis we modified our
original metacircular interpreter to separate
#idx("compiler for Python", sub: "analyzing evaluator vs.")
analysis from execution. We
analyzed each
component
to produce an execution
function
that took an environment as argument and performed the required operations.
In our compiler, we will do essentially the same analysis. Instead of
producing execution
functions,
however, we will generate sequences of instructions to be run by our
register machine.

The
function
#py("compile") is the top-level dispatch in the
compiler. It corresponds to the #py("evaluate")
function
of section @sec:core-of-evaluator, the
#py("analyze")
function
of section @sec:separating-analysis, and the
#py("eval_dispatch")
entry point of the explicit-control-evaluator in
section @sec:eceval-core. The compiler, like the
interpreters, uses the
#idx("compiler for Python", sub: "expression-syntax functions")
component-syntax functions
defined in
section @sec:representing-expressions.#footnote[Notice,
however, that our compiler is a
Python
program, and the syntax
functions
that it uses to manipulate expressions are the actual
Python functions
used with the metacircular evaluator. For the explicit-control evaluator, in
contrast, we assumed that equivalent syntax operations were available
as operations for the register machine. (Of course, when we simulated
the register machine in
Python,
we used the actual
Python functions
in our register machine simulation.)]
The function #py("compile") performs a case analysis on the
syntactic type of the
component
to be compiled. For
each type of
component,
it dispatches to a
specialized
#idx("code generator")
#emph[code generator]:
#idx("compile", decl: true)
#snippet(```python
function compile(component, target, linkage) {
    return is_literal(component)
           ? compile_literal(component, target, linkage)
           : is_name(component)
           ? compile_name(component, target, linkage)
           : is_application(component)
           ? compile_application(component, target, linkage)
           : is_operator_combination(component)
           ? compile(operator_combination_to_application(component),
                     target, linkage)
           : is_conditional(component)
           ? compile_conditional(component, target, linkage)
           : is_lambda_expression(component)
           ? compile_lambda_expression(component, target, linkage)
           : is_sequence(component)
           ? compile_sequence(sequence_statements(component),
                              target, linkage)
           : is_block(component)
           ? compile_block(component, target, linkage)
           : is_return_statement(component)
           ? compile_return_statement(component, target, linkage)
           : is_function_definition(component)
           ? compile(function_decl_to_constant_decl(component),
                     target, linkage)
           : is_declaration(component)
           ? compile_declaration(component, target, linkage)
           : is_assignment(component)
           ? compile_assignment(component, target, linkage)
           : error(component, "unknown component type -- compile");
}
```)

#subheading([Targets and linkages])

The function #py("compile")
and the code generators that it calls
take two
#idx("code generator", sub: "arguments of")
arguments in addition to the
component
to compile. There is a
#idx("target register")
#emph[target], which specifies the register in which the compiled code is
to return the value of the
component.
There is also a
#idx("linkage descriptor")
#emph[linkage descriptor], which describes how the code resulting from the
compilation of the
component
should proceed when it has finished its
execution. The linkage descriptor can require the code to do one of
the following three things:

- proceed to the next instruction in sequence (this is specified by the linkage descriptor #idx("next (linkage descriptor)") #py("\"next\"")),
- jump to the current value of the #py("continue") register as part of returning from a function call (this is specified by the linkage descriptor #idx("return (linkage descriptor)", sort: "return") #py("\"return\"")), or
- jump to a named entry point (this is specified by using the designated label as the linkage descriptor).

For example, compiling the
literal
#py("5")

with a target of the #py("val")
register and a linkage of
#py("\"next\"")
should produce
the instruction

#snippet(```python
assign("val", constant(5))
```)

Compiling the same expression with a linkage of
#py("\"return\"")
should produce the instructions

#snippet(```python
assign("val", constant(5)),
go_to(reg("continue"))
```)

In the first case, execution will continue with the next instruction
in the sequence. In the second case,
we will jump to whatever entry point is stored in the #py("continue") register.
In both cases, the value of the expression will be placed into
the target #py("val") register.

Our compiler uses the #py("\"return\"") linkage when compiling
the return expression of a return statement.
Just as in the explicit-control evaluator, returning from a function call happens in three steps:

+ reverting the stack to the marker and restoring #py("continue") (which holds a continuation set up at the beginning of the function call)
+ computing the return value and placing it in #py("val")
+ jumping to the entry point in #py("continue")

Compilation of a return statement explicitly generates code for reverting the stack and restoring #py("continue").
The return expression is compiled with target #py("val") and linkage #py("\"return\"")
so that the generated code for computing the return value places the return value in #py("val") and ends by
jumping to #py("continue").

#subheading([Instruction sequences and stack usage])

#anchor(<sec:instruction-sequences>)

#idx("instruction sequence")

Each code generator returns an
#idx("code generator", sub: "value of")
#emph[instruction sequence] containing
the object code it has generated for the
component.
Code generation for a
compound component
is accomplished by combining the output from simpler code
generators for
subcomponents,
just as evaluation of a
compound component
is accomplished by evaluating the
subcomponents.

The simplest method for combining instruction sequences is a
function
called
#idx("appendinstructionsequences")
#py("append_instruction_sequences"),
which takes as arguments two instruction sequences
that are to be
executed
sequentially. It
appends them and returns the combined sequence.
That is, if $s e q_(1)$ and
$s e q_(2)$ are sequences of instructions, then
evaluating

#syntax("
      append_instruction_sequences(", meta("seq"), $""_(1)$, ", ", meta("seq"), $""_(2)$, ")
      ")

produces the sequence

#syntax(meta("seq"), $""_(1)$, "
", meta("seq"), $""_(2)$)

Whenever registers might need to be saved, the compiler's code
generators use
#idx("preserving")
#py("preserving"), which is a more subtle method for
combining instruction sequences.
The function #py("preserving")
takes three arguments: a set of registers and two instruction sequences that
are to be executed sequentially. It appends the sequences in such a way
that the contents of each register in the set is preserved over the
execution of the first sequence, if this is needed for the execution of the
second sequence. That is, if the first sequence modifies the register
and the second sequence actually needs the register's original
contents, then #py("preserving") wraps a
#py("save") and a #py("restore")
of the register around the first sequence before appending the sequences.
Otherwise, #py("preserving") simply returns the
appended instruction sequences. Thus, for example,

#syntax("
      preserving(list(", meta("reg"), $""_(1)$, ", ", meta("reg"), $""_(2)$, "), ", meta("seq"), $""_(1)$, ", ", meta("seq"), $""_(2)$, ")
      ")

produces one of the following four sequences of instructions, depending on
how
#meta("seq")$""_(1)$ and
#meta("seq")$""_(2)$ use
#meta("reg")$""_(1)$ and
#meta("reg")$""_(2)$:

$ mat(delim: #none, italic("seq")_(1), mono("save(")italic("reg")_(1)mono("),"), mono("save(")italic("reg")_(2)mono("),"), mono("save(")italic("reg")_(2)mono("),"); italic("seq")_(2), italic("seq")_(1), italic("seq")_(1), mono("save(")italic("reg")_(1)mono("),"); , mono("restore(")italic("reg")_(1)mono("),"), mono("restore(")italic("reg")_(2)mono("),"), italic("seq")_(1); , italic("seq")_(2), italic("seq")_(2), mono("restore(")italic("reg")_(1)mono("),"); , , , mono("restore(")italic("reg")_(2)mono("),"); , , , italic("seq")_(2)) $

By using #py("preserving") to combine instruction
sequences the compiler avoids unnecessary
#idx("compiler for Python", sub: "stack usage")
stack operations. This also
isolates the details of whether or not to generate
#py("save") and #py("restore")
instructions within the #py("preserving")
function,
separating them from the concerns that arise in writing each of the
individual code generators.
In fact no #py("save") or
#py("restore") instructions are explicitly
produced by the code
generators, except that the code for calling a function saves #py("continue") and the code for returning from a function restores it: These corresponding #py("save") and #py("restore") instructions are explicitly generated by different calls to #py("compile"), not as a matched pair by #py("preserving") (as we will see in section @sec:compiling-combinations).

In principle, we could represent an instruction sequence simply as a
list of instructions.
The function #py("append_instruction_sequences")
could then combine instruction sequences by performing an ordinary list
#py("append"). However,
#py("preserving") would then be a complex operation,
because it would have to analyze each instruction sequence to
determine how the sequence uses its registers.
The function #py("preserving")
would be inefficient as well as complex, because it would have to
analyze each of its instruction sequence arguments, even though these
sequences might themselves have been constructed by calls to
#py("preserving"), in which case their parts would
have already been analyzed. To avoid such repetitious analysis we will
associate with each instruction sequence some information about its register
use. When we construct a basic instruction sequence we
will provide this information explicitly,
and the
functions
that combine instruction sequences will derive
register-use information for the combined sequence from the
information associated with the sequences being combined.

An instruction sequence will contain three pieces of information:

- the set of registers that must be initialized before the instructions in the sequence are executed (these registers are said to be #emph[needed] by the sequence),
- the set of registers whose values are modified by the instructions in the sequence, and
- the actual instructions in the sequence.

We will represent an instruction sequence as a list of its three
parts. The constructor for instruction sequences is thus
#idx("makeinstructionsequence", decl: true)

#snippet(```python
function make_instruction_sequence(needs, modifies, instructions) {
    return list(needs, modifies, instructions);
}
```)

For example, the two-instruction
sequence that looks up the value of the
symbol #py("\"x\"")
in the current environment,
assigns the result to #py("val"),
and then proceeds to the continuation,
requires registers #py("env") and
#py("continue") to have been initialized, and
modifies register #py("val").
This sequence would therefore be constructed as

#snippet(```python
make_instruction_sequence(list("env", "continue"), list("val"),
    list(assign("val",
                list(op("lookup_symbol_value"), constant("x"),
                     reg("env"))),
         go_to(reg("continue"))));
```)

The
functions
for combining instruction sequences are shown in
section @sec:combining-instruction-sequences.

#idx("instruction sequence")
#idx("compiler for Python", sub: "structure of")

#exercise(label-name: <ex:comp-optimize>, [
In evaluating a
#idx("compiler for Python", sub: "stack usage")
#idx("preserving")
function
application, the explicit-control evaluator always saves and restores
the #py("env") register around the evaluation of the
function expression,
saves and restores #py("env") around the
evaluation of each
argument expression
(except the final one), saves and restores
#py("argl") around the evaluation of each
argument expression,
and saves and restores
#py("fun")
around the
evaluation of the
argument-expression
sequence. For each of the following
applications,
say which of these #py("save") and
#py("restore") operations are superfluous and
thus could be eliminated by the compiler's
#py("preserving") mechanism:

#snippet(```python
f("x", "y")

f()("x", "y")

f(g("x"), y)

f(g("x"), "y")
```)
])

#exercise(label-name: <ex:5_33>, [
Using the
#idx("compiler for Python", sub: "explicit-control evaluator vs.")
#idx("explicit-control evaluator for Python", sub: "optimizations (additional)")
#py("preserving") mechanism, the compiler
will avoid saving and restoring #py("env") around the
evaluation of the
function expression of an application
in the case where the
function expression is a name.
We could also build such optimizations into the evaluator.
Indeed, the explicit-control evaluator of
section @sec:eceval already performs a similar
optimization, by treating
applications with no arguments
as a special case.

+ Extend the explicit-control evaluator to recognize as a separate class of components applications whose function expression is a name, and to take advantage of this fact in evaluating such components.
+ Alyssa P. Hacker suggests that by extending the evaluator to recognize more and more special cases we could incorporate all the compiler's optimizations, and that this would eliminate the advantage of compilation altogether. What do you think of this idea?
])
