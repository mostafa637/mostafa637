// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Running the Evaluator], label-name: <sec:running-evaluator>)

#idx("explicit-control evaluator for Python", sub: "running")

With the implementation of the explicit-control evaluator we come to
the end of a development, begun in chapter @chap:fun, in which we have
explored successively more precise
#idx("models of evaluation")
#idx("evaluation", sub: "models of")
models of the evaluation process.
We started with the relatively informal substitution model, then
extended this in chapter @chap:state to the environment model, which enabled us
to deal with state and change. In the metacircular evaluator of
chapter @chap:meta, we used
Python
itself as a language for making more
explicit the environment structure constructed during evaluation of an
component.
Now, with register machines, we have taken a close look
at the evaluator's mechanisms for storage management,
argument passing, and control. At
each new level of description, we have had to raise issues and resolve
ambiguities that were not apparent at the previous, less precise
treatment of evaluation. To understand the behavior of the
explicit-control evaluator, we can simulate it and monitor its
performance.

We will install a
#idx("explicit-control evaluator for Python", sub: "driver loop")
#idx("driver loop", sub: "in explicit-control evaluator")
driver loop in our evaluator machine. This plays
the role of the
#py("driver_loop")
function
of section @sec:running-eval. The evaluator
will repeatedly print a prompt, read
a program,
evaluate
the program
by going to
#py("eval_dispatch"),
and print the result.
If nothing is entered at the prompt, we jump to the label #py("evaluator_done"), which is the last entry point in the controller.
The following instructions form the beginning of the
explicit-control evaluator's controller sequence:#footnote[We assume
here that
#py("user_read"), #py("parse"),
and the various printing
operations are available as primitive machine operations, which is useful
for our simulation, but completely unrealistic in practice. These are
actually extremely complex operations. In practice,
reading and printing
would be
implemented using low-level input-output operations such as transferring
single characters to and from a device.]
#idx("prompts", sub: "explicit-control evaluator")#idx("readevaluateprintloop", decl: true)#idx("printresult", decl: true)
#syntax("
\"read_evaluate_print_loop\",
  perform(list(op(\"initialize_stack\"))),
  assign(\"comp\", list(op(\"user_read\"),
                      constant(\"EC-evaluate input:\"))),
  assign(\"comp\", list(op(\"parse\"), reg(\"comp\"))),
  test(list(op(\"is_null\"), reg(\"comp\"))),
  branch(label(\"evaluator_done\")),
  assign(\"env\", list(op(\"get_current_environment\"))),
  assign(\"val\", list(op(\"scan_out_declarations\"), reg(\"comp\"))),
  save(\"comp\"),    // so we can use it to temporarily hold ", $mono("*unassigned*")$, " values
  assign(\"comp\", list(op(\"list_of_unassigned\"), reg(\"val\"))),
  assign(\"env\", list(op(\"extend_environment\"),
                     reg(\"val\"), reg(\"comp\"), reg(\"env\"))),
  perform(list(op(\"set_current_environment\"), reg(\"env\"))),
  restore(\"comp\"), // the program
  assign(\"continue\", label(\"print_result\")),
  go_to(label(\"eval_dispatch\")),
\"print_result\",
  perform(list(op(\"user_print\"),
               constant(\"EC-evaluate value:\"), reg(\"val\"))),
  go_to(label(\"read_evaluate_print_loop\")),
  ")

We store the current environment, initially the global environment,
in the variable #py("current_environment")
and update it each time around the loop to remember past declarations.
The operations
#py("get_current_environment") and
#py("set_current_environment")
simply get and set this variable.
#idx("getcurrentenvironment", decl: true)#idx("setcurrentenvironment", decl: true)
#snippet(```python
let current_environment = the_global_environment;

function get_current_environment() {
    return current_environment;
}

function set_current_environment(env) {
    current_environment = env;
}
```)

When we encounter an
#idx("error handling", sub: "in explicit-control evaluator")
#idx("explicit-control evaluator for Python", sub: "error handling")
error in a
function
(such as the
"unknown function type" error
indicated at
#py("apply_dispatch")),
we print an error message and return to the driver loop.#footnote[There are
other errors that we would like the interpreter to handle, but these are not
so simple. See exercise @ex:interp-errors.]
#idx("unknowncomponenttype", decl: true)#idx("unknownfunctiontype", decl: true)#idx("signalerror", decl: true)
#syntax("
\"unknown_component_type\",
  assign(\"val\", constant(\"unknown syntax\")),
  go_to(label(\"signal_error\")),

\"unknown_function_type\",
  restore(\"continue\"), // clean up stack (from ", $mono("apply_dispatch")$, ")
  assign(\"val\", constant(\"unknown function type\")),
  go_to(label(\"signal_error\")),

\"signal_error\",
  perform(list(op(\"user_print\"),
               constant(\"EC-evaluator error:\"), reg(\"val\"))),
  go_to(label(\"read_evaluate_print_loop\")),
      ")

For the purposes of the simulation, we initialize the stack each time
through the driver loop, since it might not be empty after an error
(such as an undeclared name)
interrupts an evaluation.#footnote[We
could perform the stack initialization only after errors, but doing it in
the driver loop will be convenient for monitoring the evaluator's
performance, as described below.]

#idx("explicit-control evaluator for Python", sub: "controller")

If we combine all the code fragments presented in sections
@sec:eceval-core–@sec:running-evaluator,
we can create an
#idx("explicit-control evaluator for Python", sub: "machine model")
evaluator machine model that we can run using the
register-machine simulator of section @sec:simulator.

#idx("eceval", decl: true)
#syntax("
const eceval = make_machine(list(\"comp\", \"env\", \"val\", \"fun\",
                                 \"argl\", \"continue\", \"unev\"),
                            eceval_operations,
                            list(\"read_evaluate_print_loop\",
                                 ", metaphrase[entire machine controller as given above], "
                                 \"evaluator_done\"));
      ")

We must define
Python functions
to simulate the operations used as primitives by the evaluator. These are
the same
functions
we used for the metacircular evaluator in
section @sec:mc-eval, together with the few additional
ones defined in footnotes throughout section @sec:eceval.

#syntax("
const eceval_operations = list(list(\"is_literal\", is_literal),
                               ", $⟨italic("complete") med thin italic("list") med thin italic("of") med italic("operations") med thin italic("for") med thin italic("eceval") med thin italic("machine")⟩$, ");
      ")

Finally, we can initialize the global environment and run the evaluator:
#idx("theglobalenvironment", decl: true)
#snippet(```python
const the_global_environment = setup_environment();
start(eceval);
```)

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
function append(x, y) {
    return is_null(x)
           ? y
           : pair(head(x), append(tail(x), y));
}
```)

#output(```python
function append(x, y) {
    return is_null(x)
           ? y
           : pair(head(x), append(tail(x), y));
}
```)

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
append(list("a", "b", "c"), list("d", "e", "f"));
```)

#output(```python
append(list("a", "b", "c"), list("d", "e", "f"));
```)

Of course, evaluating
programs
in this way will take much longer
than if we had directly typed them into
Python,
because of the
multiple levels of simulation involved. Our
programs
are evaluated
by the explicit-control-evaluator machine, which is being simulated by
a
Python
program, which is itself being evaluated by the
Python
interpreter.

#idx("explicit-control evaluator for Python", sub: "running")

#subheading([Monitoring the performance of the evaluator])

#idx("explicit-control evaluator for Python", sub: "monitoring performance (stack use)")

Simulation can be a powerful tool to guide the implementation of
evaluators.
#idx("simulation", sub: "as machine-design tool")
Simulations make it easy not only to explore variations
of the register-machine design but also to monitor the performance of
the simulated evaluator. For example, one important factor in
performance is how efficiently the evaluator uses the stack. We can
observe the number of stack operations required to evaluate various
programs
by defining the evaluator register machine with the
version of the simulator that collects statistics on stack use
(section @sec:monitor), and adding an instruction at the
evaluator's
#py("print_result")
entry point to print the statistics:
#idx("printresult", sub: "monitored-stack version", decl: true)
#syntax("
\"print_result\",
  perform(list(op(\"print_stack_statistics\"))), // added instruction
  // rest is same as before
  perform(list(op(\"user_print\"),
               constant(\"EC-evaluate value:\"), reg(\"val\"))),
  go_to(label(\"read_evaluate_print_loop\")),
      ")

Interactions with the evaluator now look like this:

#prompt(```python
EC-evaluate input:
```)

#snippet(```python
function factorial (n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
```)

#output(```python
function factorial (n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
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

Note that the driver loop of the evaluator reinitializes the stack
at the start of
each interaction, so that the statistics printed will refer only to
stack operations used to evaluate the previous
program.

#exercise(label-name: <ex:tail-rec-fact>, [
Use the monitored stack to explore the
#idx("explicit-control evaluator for Python", sub: "tail recursion")
#idx("tail recursion", sub: "explicit-control evaluator and")
tail-recursive property of the
evaluator (section @sec:tail-recursion-return). Start the
evaluator and define the
#idx("factorial", sub: "stack usage, interpreted")
iterative #py("factorial")
function
from section @sec:recursion-and-iteration:

#snippet(```python
function factorial(n) {
    function iter(product, counter) {
        return counter > n
               ? product
               : iter(counter * product,
                      counter + 1);
    }
    return iter(1, 1);
}
```)

Run the
function
with some small values of $n$. Record the
maximum stack depth and the number of pushes required to compute
$n!$ for each of these values.

+ You will find that the maximum depth required to evaluate $n!$ is independent of $n$. What is that depth?
+ Determine from your data a formula in terms of $n$ for the total number of push operations used in evaluating $n!$ for any $n gt.eq 1$. Note that the number of operations used is a linear function of $n$ and is thus determined by two constants.
])

#exercise(label-name: <ex:rec-fact>, [
For comparison with exercise @ex:tail-rec-fact, explore
the behavior of the following
function
for computing
#idx("factorial", sub: "stack usage, interpreted")
factorials recursively:

#snippet(```python
function factorial(n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
```)

By running this
function
with the monitored stack, determine, as a function of
$n$, the maximum depth of the stack and the total
number of pushes used in evaluating $n!$ for
$n gt.eq 1$. (Again, these functions will be
linear.) Summarize your experiments by filling in the following table with
the appropriate expressions in terms of $n$:

#blockquote[$ mat(delim: #none, , "Maximum depth", "Number of pushes"; "Recursive factorial", , ; "Iterative factorial", , ) $]

The maximum depth is a measure of the amount of space used by the
evaluator in carrying out the computation, and the number of pushes
correlates well with the time required.
])

#exercise(label-name: <ex:5_29>, [
Modify the definition of the evaluator by changing
#py("ev_return") as described in section @sec:tail-recursion-return
so that the evaluator is no longer
#idx("explicit-control evaluator for Python", sub: "tail recursion")
#idx("tail recursion", sub: "explicit-control evaluator and")
tail-recursive. Rerun your experiments from
exercises @ex:tail-rec-fact
and @ex:rec-fact to demonstrate that both versions of
the #py("factorial")
function
now require space that grows linearly with their input.
])

#exercise(label-name: <ex:rec-fib>, [
Monitor the stack operations in the tree-recursive
#idx("fib", sub: "stack usage, interpreted")
Fibonacci computation:
#idx("fib", sub: "tree-recursive version", decl: true)
#snippet(```python
function fib(n) {
    return n < 2 ? n : fib(n - 1) + fib(n - 2);
}
```)

+ Give a formula in terms of $n$ for the maximum depth of the stack required to compute $(upright("Fib"))(n)$ for $n gt.eq 2$. Hint: In section @sec:tree-recursion we argued that the space used by this process grows linearly with $n$.
+ Give a formula for the total number of pushes used to compute $(upright("Fib"))(n)$ for $n gt.eq 2$. You should find that the number of pushes (which correlates well with the time used) grows exponentially with $n$. Hint: Let $S(n)$ be the number of pushes used in computing $(upright("Fib"))(n)$. You should be able to argue that there is a formula that expresses $S(n)$ in terms of $S(n-1)$, $S(n-2)$, and some fixed "overhead" constant $k$ that is independent of $n$. Give the formula, and say what $k$ is. Then show that $S(n)$ can be expressed as $a (upright("Fib"))(n+1) + b$ and give the values of $a$ and $b$.
])

#idx("explicit-control evaluator for Python", sub: "monitoring performance (stack use)")

#exercise(label-name: <ex:interp-errors>, [
Our evaluator currently catches and signals only two kinds of
#idx("error handling", sub: "in explicit-control evaluator")
#idx("explicit-control evaluator for Python", sub: "error handling")
errors—unknown
component
types and unknown
function
types. Other errors will take us out of the evaluator
read-evaluate-print
loop.
When we run the evaluator using the register-machine simulator, these
errors are caught by the underlying
Python
system. This is analogous
to the computer crashing when a user program makes an error.#footnote[This manifests itself as, for example, a "kernel panic" or a "blue screen of death" or even a reboot. Automatic rebooting is an approach
typically used on phones and tablets. Most modern operating systems do a
decent job of preventing user programs from causing an entire machine to
crash.]
It is a large project to
make a real error system work, but it is well worth the effort to understand
what is involved here.

+ Errors that occur in the evaluation process, such as an attempt to access an unbound name, could be caught by changing the lookup operation to make it return a distinguished condition code, which cannot be a possible value of any user name. The evaluator can test for this condition code and then do what is necessary to go to #py("signal_error"). Find all of the places in the evaluator where such a change is necessary and fix them. This is lots of work.
+ Much worse is the problem of handling errors that are signaled by applying primitive functions such as an attempt to divide by zero or an attempt to extract the #py("head") of a string. In a professionally written high-quality system, each primitive application is checked for safety as part of the primitive. For example, every call to #py("head") could first check that the argument is a pair. If the argument is not a pair, the application would return a distinguished condition code to the evaluator, which would then report the failure. We could arrange for this in our register-machine simulator by making each primitive function check for applicability and returning an appropriate distinguished condition code on failure. Then the #py("primitive_apply") code in the evaluator can check for the condition code and go to #py("signal_error") if necessary. Build this structure and make it work. This is a major project.
])

#idx("explicit-control evaluator for Python")
