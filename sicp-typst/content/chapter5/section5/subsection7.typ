// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Interfacing Compiled Code to the Evaluator], label-name: <sec:interfacing-compiled-code>)

#idx("compiler for Python", sub: "interfacing to evaluator")
#idx("compiler for Python", sub: "running compiled code")
#idx("explicit-control evaluator for Python", sub: "modified for compiled code")

We have not yet explained how to load compiled code into the evaluator
machine or how to run it. We will assume that the explicit-control-evaluator
machine has been defined as in
section @sec:running-evaluator, with the additional
operations specified in footnote @foot:compiler-ops (section @sec:compiling-components).
We will implement a
function
#idx("compileandgo")
#py("compile_and_go")
that compiles a
Python program,
loads the resulting object code into the evaluator machine,
and causes the machine to run the code in the
evaluator global environment, print the result, and
enter the evaluator's driver loop. We will also modify the evaluator
so that interpreted
components
can call compiled
functions
as well as interpreted ones. We can then put a compiled
function
into the machine and use the
evaluator to call it:

#snippet(```python
compile_and_go(parse(`
function factorial(n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
                     `));
```)

#output(```python
compile_and_go(parse(`
function factorial(n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
                     `));
```)

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
factorial(5);
```)

#output(```python
factorial(5);
```)

To allow the evaluator to handle compiled
functions
(for example,
to evaluate the call to #py("factorial") above),
we need to change the code at
#py("apply_dispatch")
(section @sec:procedure-application) so that it
recognizes compiled
functions
(as distinct from compound or primitive
functions)
and transfers control directly to the entry point of the
compiled code:#footnote[Of course, compiled
functions
as well as interpreted
functions
are compound (nonprimitive). For compatibility with the terminology used
in the explicit-control evaluator, in this section we will use
"compound" to mean interpreted (as opposed to
compiled).]
#idx("applydispatch", sub: "modified for compiled code", decl: true)#idx("compiledapply", decl: true)
#snippet(```python
"apply_dispatch",
  test(list(op("is_primitive_function"), reg("fun"))),
  branch(label("primitive_apply")),
  test(list(op("is_compound_function"), reg("fun"))),
  branch(label("compound_apply")),
  test(list(op("is_compiled_function"), reg("fun"))),
  branch(label("compiled_apply")),
  go_to(label("unknown_function_type")),

"compiled_apply",
  push_marker_to_stack(),
  assign("val", list(op("compiled_function_entry"), reg("fun"))),
  go_to(reg("val")),
```)

At
#py("compiled_apply"), as at
#py("compound_apply"), we push a marker to the stack
so that a return statement in the compiled function
can revert the stack to this state.

Note that there is no save of
#py("continue") at
#py("compiled_apply") before
the marking of the stack, because the evaluator was
arranged so that at
#py("apply_dispatch"), the
continuation would be at the top of the stack.

To enable us to run some compiled code when we start the evaluator
machine, we add a #py("branch") instruction at
the beginning of the evaluator machine, which causes the machine to
go to a new entry point if the #py("flag") register
is set.#footnote[Now that the evaluator machine starts
with a #py("branch"), we must always initialize the
#py("flag") register before starting the evaluator
machine. To start the machine at its ordinary
read-evaluate-print
loop, we
could use
#idx("starteceval", decl: true)
#snippet(```python
function start_eceval() {
    set_register_contents(eceval, "flag", false);
    return start(eceval);
}
```)]

#syntax($mono(" ")mono(" ")$, "branch(label(\"external_entry\")), // branches if flag is set
\"read_evaluate_print_loop\",
  perform(list(op(\"initialize_stack\"))),
  ", $dots.h$)

The code at #py("external_entry")
assumes that the machine is started with #py("val")
containing the location of an instruction sequence that puts a result into
#py("val") and ends with
#py("go_to(reg(\"continue\"))").
Starting at this entry point jumps to the location designated
by #py("val"), but first assigns
#py("continue") so that execution will return to
#py("print_result"),
which prints the value in #py("val") and then goes to
the beginning of the evaluator's
read-evaluate-print
loop.#footnote[Since
a compiled
function
is an object that the system may try to print, we also modify the system
print operation
#py("user_print")
(from section @sec:running-eval) so that it will not
attempt to print the components of a compiled
function:
#idx("userprint", sub: "modified for compiled code", decl: true)
#snippet(```python
function user_print(string, object) {
    function prepare(object) {
        return is_compound_function(object)
               ? "< compound function >"
               : is_primitive_function(object)
               ? "< primitive function >"
               : is_compiled_function(object)
               ? "< compiled function >"
               : is_pair(object)
               ? pair(prepare(head(object)),
                      prepare(tail(object)))
               : object;
    }
    display(string + " " + stringify(prepare(object)));
}
```)]
#idx("externalentry", decl: true)
#snippet(```python
"external_entry",
  perform(list(op("initialize_stack"))),
  assign("env", list(op("get_current_environment"))),
  assign("continue", label("print_result")),
  go_to(reg("val")),
```)

#idx("explicit-control evaluator for Python", sub: "modified for compiled code")

Now we can use the following
function
to compile a
function definition,
execute the compiled code, and run the
read-evaluate-print
loop so
we can try the
function.
Because we want the compiled code to
proceed
to the location in
#py("continue") with its result in
#py("val"), we compile the
program
with a target of #py("val") and a
linkage of
#py("\"return\"").
In order to transform the
object code produced by the compiler into executable instructions
for the evaluator register machine, we use the
function
#py("assemble") from the
register-machine simulator
(section @sec:assembler).

For the interpreted program to refer to the names that
are declared at top level in the compiled program, we
#idx("scanning out declarations", sub: "in compiler")
scan out the top-level names and
extend the global environment by binding these names to
#py("\"*unassigned*\""),
knowing that the compiled code will assign them
the correct values.

We then initialize
the #py("val") register to point to the list
of instructions, set the
#py("flag") so that the evaluator will go to
#py("external_entry"),
and start the evaluator.

#idx("compileandgo", decl: true)
#snippet(```python
function compile_and_go(program) {
    const instrs = assemble(instructions(compile(program,
                                                 "val", "return")),
                            eceval);
    const toplevel_names = scan_out_declarations(program);
    const unassigneds = list_of_unassigned(toplevel_names);
    set_current_environment(extend_environment(
                               toplevel_names,
                               unassigneds,
                               the_global_environment));
    set_register_contents(eceval, "val", instrs);
    set_register_contents(eceval, "flag", true);
    return start(eceval);
}
```)

If we have set up
#idx("compiler for Python", sub: "monitoring performance (stack use) of compiled code")
stack monitoring, as at the end of
section @sec:running-evaluator, we can examine the
stack usage of compiled code:

#snippet(```python
compile_and_go(parse(`
function factorial(n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
                     `));
```)

#output(```python
compile_and_go(parse(`
function factorial(n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
                     `));
```)

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
factorial(5);
```)

#output(```python
factorial(5);
```)

Compare
#idx("compiler for Python", sub: "explicit-control evaluator vs.")
this example with the evaluation of
#py("factorial(5)")
using the interpreted version of the same
function,
shown at the end of section @sec:running-evaluator.
The interpreted version required 151 pushes and a maximum stack depth of 28.
This illustrates the optimization that results from our compilation strategy.

#subheading([Interpretation and compilation])

#idx("interpreter", sub: "compiler vs.")
#idx("compiler", sub: "interpreter vs.")

With the programs in this section, we can now experiment with the
alternative execution strategies of interpretation and
compilation.#footnote[We can do even better by extending the compiler
to allow compiled code to call interpreted
functions.
See exercise @ex:compiled-call-interpreted.]
An interpreter raises the machine to the level of the user program; a
compiler lowers the user program to the level of the machine language.
We can regard the
Python
language (or any programming language) as a
coherent family of abstractions erected on the machine language.
Interpreters are good for interactive program development and
debugging because the steps of program execution are organized in
terms of these abstractions, and are therefore more intelligible
to the programmer.
Compiled code can execute faster, because the steps of program execution
are organized in terms of the machine language, and the compiler is free
to make optimizations that cut across the higher-level
abstractions.#footnote[Independent of the strategy of execution, we
incur significant overhead if we insist that
#idx("error handling", sub: "in compiled code")
errors encountered in
execution of a user program be detected and signaled, rather than being
allowed to kill the system or produce wrong answers. For example, an
out-of-bounds array reference can be detected by checking the validity
of the reference before performing it. The overhead of checking,
however, can be many times the cost of the array reference itself, and
a programmer should weigh speed against safety in determining whether
such a check is desirable. A good compiler should be able to produce
code with such checks, should avoid redundant checks, and should allow
programmers to control the extent and type of error checking in the
compiled code.

Compilers for popular languages, such as
#idx("C", sub: "error handling")
C and C++,
put hardly any error-checking operations into
running code, so as to make things run as fast as possible. As a
result, it falls to programmers to explicitly provide error checking.
Unfortunately, people often neglect to do this, even in
critical applications where speed is not a constraint. Their programs
lead fast and dangerous lives. For example, the notorious
#idx("Internet Worm")
"Worm"
that paralyzed the Internet in 1988 exploited the
#idx("UNIX")
UNIX$""^(upright("TM"))$
operating system's failure to check whether the input buffer has
overflowed in the finger daemon. (See
#idx("Spafford, Eugene H.")
Spafford 1989.)]

The alternatives of interpretation and compilation also lead to
different strategies for
#idx("porting a language")
porting languages to new computers. Suppose
that we wish to implement
Python
for a new machine. One strategy is
to begin with the explicit-control evaluator of
section @sec:eceval
and translate its instructions to instructions for the
new machine. A different strategy is to begin with the compiler and
change the code generators so that they generate code for the new
machine. The second strategy allows us to run any
Python
program on the new machine by first compiling it with the compiler running
on our
original Python system, and linking it with a compiled version of the runtime
library.#footnote[Of course, with either the interpretation or the
compilation strategy we must also implement for the new machine storage
allocation, input and output, and all the various operations that we took
as "primitive" in our discussion of
the evaluator and compiler. One strategy for minimizing work here is
to write as many of these operations as possible in
Python
and then compile them for the new machine. Ultimately, everything reduces
to a small kernel (such as garbage collection and the mechanism for
applying actual machine primitives) that is hand-coded for the new
machine.] Better yet, we can compile the compiler itself, and run
this on the new machine to compile other
Python programs.#footnote[This strategy leads to amusing tests of correctness of
the compiler, such as checking
whether the compilation of a program on the new machine, using the
compiled compiler, is identical with the
compilation of the program on the original
Python
system. Tracking down the source of differences is fun but often
frustrating, because the results are extremely sensitive to minuscule
details.] Or we can compile one of the interpreters of
section @sec:mc-eval to produce an interpreter that
runs on the new machine.

#exercise(label-name: <ex:measure-factorial-ratio>, [
By
#idx("compiler for Python", sub: "monitoring performance (stack use) of compiled code")
#idx("factorial", sub: "stack usage, compiled")
comparing the stack operations used by compiled code to the stack
operations used by the evaluator for the same computation, we can
determine the extent to which the compiler optimizes use of the stack,
both in speed (reducing the total number of stack operations) and in
space (reducing the maximum stack depth). Comparing this optimized
stack use to the performance of a special-purpose machine for the same
computation gives some indication of the quality of the compiler.

+ Exercise @ex:rec-fact asked you to determine, as a function of $n$, the number of pushes and the maximum stack depth needed by the evaluator to compute $n!$ using the recursive factorial function given above. Exercise @ex:measure-fact asked you to do the same measurements for the special-purpose factorial machine shown in figure @fig:fact-machine. Now perform the same analysis using the compiled #py("factorial") function. Take the ratio of the number of pushes in the compiled version to the number of pushes in the interpreted version, and do the same for the maximum stack depth. Since the number of operations and the stack depth used to compute $n!$ are linear in $n$, these ratios should approach constants as $n$ becomes large. What are these constants? Similarly, find the ratios of the stack usage in the special-purpose machine to the usage in the interpreted version. Compare the ratios for special-purpose versus interpreted code to the ratios for compiled versus interpreted code. You should find that the special-purpose machine is much more efficient than the compiled code, since the hand-tailored controller code should be much better than what is produced by our rudimentary general-purpose compiler.
+ Can you suggest improvements to the compiler that would help it generate code that would come closer in performance to the hand-tailored version?
])

#exercise(label-name: <ex:measure-fib-ratio>, [
Carry out an analysis like the one in
exercise @ex:measure-factorial-ratio to determine the
effectiveness of compiling the tree-recursive
#idx("compiler for Python", sub: "monitoring performance (stack use) of compiled code")
#idx("fib", sub: "stack usage, compiled")
Fibonacci
function

#snippet(```python
function fib(n) {
    return n < 2 ? n : fib(n - 1) + fib(n - 2);
}
```)

compared to the effectiveness of using the special-purpose Fibonacci machine
of figure @fig:fib-machine. (For measurement of the
interpreted performance, see exercise @ex:rec-fib.)
For Fibonacci, the time resource used is not linear in
$n$; hence the ratios of stack operations will not
approach a limiting value that is independent of
$n$.
])

#exercise(label-name: <ex:compiled-call-interpreted>, [
This section described how to modify the explicit-control evaluator so
that interpreted code can call compiled
functions.
Show how to modify the compiler so that compiled
functions
can call not only primitive
functions
and compiled
functions,
but interpreted
functions
as well. This requires modifying
#py("compile_function_call")
to handle the case of compound (interpreted)
functions.
Be sure to handle all the same #py("target") and
#py("linkage") combinations
as in
#py("compile_fun_appl").
To do the actual
function
application,
the code needs to jump to the evaluator's
#py("compound_apply")
entry point. This label cannot be directly referenced in object code
(since the assembler requires that all labels referenced by the
code it is assembling be defined there), so we will add a register
called #py("compapp") to the evaluator machine to
hold this entry point, and add an instruction to initialize it:

#syntax($mono(" ")mono(" ")$, "assign(\"compapp\", label(\"compound_apply\")),
  branch(label(\"external_entry\")),     // branches if flag is set
\"read_evaluate_print_loop\",
  ", $dots.h$)

To test your code, start by
declaring
a
function
#py("f") that calls a
function
#py("g"). Use
#py("compile_and_go")
to compile the
declaration
of #py("f")
and start the evaluator. Now, typing at the evaluator,
declare #py("g") and try to call #py("f").
])

#exercise(label-name: <ex:5_51>, [
The
#py("compile_and_go")
interface implemented in this section is
awkward, since the compiler can be called only once (when the
evaluator machine is started). Augment the compiler–interpreter
interface by providing a
#idx("compileandrun")
#py("compile_and_run")
primitive that can be called from within the explicit-control evaluator
as follows:

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
compile_and_run(parse(`
function factorial(n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
                      `));
```)

#output(```python
compile_and_run(parse(`
function factorial(n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
                      `));
```)

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
factorial(5)
```)

#output(```python
factorial(5)
```)
])

#exercise(label-name: <ex:read-compile-execute>, [
As an alternative to using the explicit-control evaluator's
read-evaluate-print
loop, design a register machine that performs a
read-compile-execute-print loop. That is, the machine should run a
loop that reads
a program,
compiles it, assembles and
executes the resulting code, and prints the result. This is easy to
run in our simulated setup, since we can arrange to call the
functions
#py("compile") and
#py("assemble") as "register-machine operations."
])

#exercise(label-name: <ex:5_53>, [
Use the compiler to compile the
#idx("metacircular evaluator for Python", sub: "compilation of")
metacircular evaluator of
section @sec:mc-eval and run this program using the
register-machine simulator.

Because the parser takes a string as input, you will need to
convert the program into a string. The simplest way to do this is
to use the back quotes (#py("`")),
as we have done for the example inputs to
#py("compile_and_go") and
#py("compile_and_run").

The resulting interpreter will run very slowly because of the multiple
levels of interpretation, but getting all the details to work is an
instructive exercise.
])

#exercise(label-name: <ex:interp-in-C>, [
Develop a rudimentary implementation of
Python
in
#idx("C", sub: "Python interpreter written in")
C (or some other low-level language of your choice) by translating the
explicit-control evaluator of section @sec:eceval
into C. In order to run this code you will need to also
provide appropriate storage-allocation routines and other runtime
support.
])

#exercise(label-name: <ex:compiler-in-C>, [
As a counterpoint to exercise @ex:interp-in-C, modify
the compiler so that it compiles
Python functions
into sequences of
#idx("C", sub: "compiling Python into")
#idx("C", sub: "Python interpreter written in")
#idx("metacircular evaluator for Python", sub: "compilation of")
C instructions. Compile the metacircular evaluator of
section @sec:mc-eval to produce a
Python
interpreter written in C.
])

#idx("compiler for Python", sub: "interfacing to evaluator")
#idx("compiler for Python", sub: "running compiled code")
#idx("compiler for Python")
