// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Evaluating Function Applications], label-name: <sec:tail-recursion-return>)

#anchor(<sec:evaluating-function-applications>)

#idx("explicit-control evaluator for Python", sub: "function application")
#idx("explicit-control evaluator for Python", sub: "combinations")

A function application is specified by a combination containing a function
expression and argument expressions. The function expression is a subexpression
whose value is a function, and the argument expressions are subexpressions whose
values are the arguments to which the function should be applied. The metacircular
#py("evaluate") handles applications by calling
itself recursively to evaluate each element of the combination, and then passing
the results to #py("apply"), which performs the
actual function application. The explicit-control evaluator does the same thing;
these recursive calls are implemented by
#py("go_to") instructions, together with use of the
stack to save registers that will be restored after the recursive call returns.
Before each call we will be
#idx("explicit-control evaluator for Python", sub: "stack usage")
careful to identify which registers must be saved
(because their values will be needed later).#footnote[This is an important but
subtle point in translating algorithms from a procedural language, such as
Python, to a register-machine language. As an alternative to saving only what
is needed, we could save all the registers (except
#py("val")) before each recursive call. This is
called a
#idx("framed-stack discipline")
#idx("stack", sub: "framed")
#emph[framed-stack] discipline.
This would work but might save more registers than necessary; this could be an
important consideration in a system where stack operations are expensive. Saving
registers whose contents will not be needed later may also hold on to useless data
that could otherwise be garbage-collected, freeing space to be reused.]

As in the metacircular evaluator, operator combinations are transformed into
applications of primitive functions corresponding to the operators. This takes
place at #py("ev_operator_combination"), which
performs this transformation in place in #py("comp")
and falls through to
#py("ev_application").#footnote[We assume that the syntax transformer
#py("operator_combination_to_application") is
available as a machine operation. In an actual implementation built from
scratch, we would use our explicit-control evaluator to interpret a Python
program that performs source-level transformations like this one and
#py("function_decl_to_constant_decl")
in a syntax phase that runs before execution.]<foot:syntax-transformer>

We begin the evaluation of an application by evaluating the function expression to
produce a function, which will later be applied to the evaluated argument
expressions. To evaluate the function expression, we move it to the
#py("comp") register and go to
#py("eval_dispatch"). The environment in the
#py("env") register is already the correct one in
which to evaluate the function expression. However, we save
#py("env") because we will need it later to evaluate
the argument expressions. We also extract the argument expressions into
#py("unev") and save this on the stack. We set up
#py("continue") so that
#py("eval_dispatch") will resume at
#py("ev_appl_did_function_expression") after the
function expression has been evaluated. First, however, we save the old value of
#py("continue"), which tells the controller where to
continue after the application.

#idx("evoperatorcombination", decl: true)#idx("evapplication", decl: true)
#snippet(```python
"ev_operator_combination",
  assign("comp", list(op("operator_combination_to_application"),
                      reg("comp"), reg("env"))),
"ev_application",
  save("continue"),
  save("env"),
  assign("unev", list(op("arg_expressions"), reg("comp"))),
  save("unev"),
  assign("comp", list(op("function_expression"), reg("comp"))),
  assign("continue", label("ev_appl_did_function_expression")),
  go_to(label("eval_dispatch")),
```)

#idx("explicit-control evaluator for Python", sub: "argument evaluation")

Upon returning from evaluating the function expression, we proceed to evaluate the
argument expressions of the application and to accumulate the resulting arguments
in a list, held in #py("argl"). (This is like the
evaluation of a sequence of statements, except that we collect the values.) First
we restore the unevaluated argument expressions and the environment. We initialize
#py("argl") to an empty list. Then we assign to the
#py("fun") register the function that was produced
by evaluating the function expression. If there are no argument expressions, we go
directly to #py("apply_dispatch"). Otherwise we save
#py("fun") on the stack and start the
argument-evaluation loop:#footnote[We add to the evaluator data-structure
functions in section @sec:eval-data-structures the following
two functions for manipulating argument lists:
#idx("emptyarglist", decl: true)#idx("adjoinarg", decl: true)
#snippet(```python
function empty_arglist() { return null; }

function adjoin_arg(arg, arglist) {
    return append(arglist, list(arg));
}
```)

We also make use of an additional syntax function to test for the last argument expression
in an application:
#idx("islastargumentexpression", decl: true)
#snippet(```python
function is_last_argument_expression(arg_expression) {
    return is_null(tail(arg_expression));
}
```)]

#snippet(```python
"ev_appl_did_function_expression",
  restore("unev"), // the argument expressions
  restore("env"),
  assign("argl", list(op("empty_arglist"))),
  assign("fun", reg("val")), // the function
  test(list(op("is_null"), reg("unev"))),
  branch(label("apply_dispatch")),
  save("fun"),
```)

Each cycle of the argument-evaluation loop evaluates an argument expression from
the list in #py("unev") and accumulates the result
into #py("argl"). To evaluate an argument
expression, we place it in the #py("comp") register
and go to #py("eval_dispatch"), after setting
#py("continue") so that execution will resume with
the argument-accumulation phase. But first we save the arguments accumulated so
far (held in #py("argl")), the environment (held in
#py("env")), and the remaining argument expressions
to be evaluated (held in #py("unev")). A special
case is made for the evaluation of the last argument expression, which is handled
at #py("ev_appl_last_arg").

#snippet(```python
"ev_appl_argument_expression_loop",
  save("argl"),
  assign("comp", list(op("head"), reg("unev"))),
  test(list(op("is_last_argument_expression"), reg("unev"))),
  branch(label("ev_appl_last_arg")),
  save("env"),
  save("unev"),
  assign("continue", label("ev_appl_accumulate_arg")),
  go_to(label("eval_dispatch")),
```)

When an argument expression has been evaluated, the value is accumulated into the
list held in #py("argl"). The argument expression is
then removed from the list of unevaluated argument expressions in
#py("unev"), and the argument-evaluation loop
continues.

#snippet(```python
"ev_appl_accumulate_arg",
  restore("unev"),
  restore("env"),
  restore("argl"),
  assign("argl", list(op("adjoin_arg"), reg("val"), reg("argl"))),
  assign("unev", list(op("tail"), reg("unev"))),
  go_to(label("ev_appl_argument_expression_loop")),
```)

Evaluation of the last argument expression is handled differently, as is the last
statement in a sequence. There is no need to save the environment or the list of
unevaluated argument expressions before going to
#py("eval_dispatch"), since they will not be
required after the last argument expression is evaluated. Thus, we return from the
evaluation to a special entry point
#py("ev_appl_accum_last_arg"), which restores the
argument list, accumulates the new argument, restores the saved function, and goes
off to perform the application.#footnote[The optimization of treating the last
argument expression specially is known as
#idx("evlis tail recursion")
#emph[evlis tail recursion] (see
#idx("Wand, Mitchell")
Wand 1980).
We could be somewhat more efficient in the argument evaluation loop if we made
evaluation of the first argument expression a special case too. This would permit
us to postpone initializing #py("argl") until after
evaluating the first argument expression, so as to avoid saving
#py("argl") in this case. The compiler in
section @sec:compilation performs this optimization. (Compare
the #py("construct_arglist") function of
section @sec:compiling-combinations.)]

#snippet(```python
"ev_appl_last_arg",
  assign("continue", label("ev_appl_accum_last_arg")),
  go_to(label("eval_dispatch")),
"ev_appl_accum_last_arg",
  restore("argl"),
  assign("argl", list(op("adjoin_arg"), reg("val"), reg("argl"))),
  restore("fun"),
  go_to(label("apply_dispatch")),
```)

The details of the argument-evaluation loop determine the
#idx("order of evaluation", sub: "in explicit-control evaluator")
order in which the
interpreter evaluates the argument expressions of a combination (e.g., left to
right or right to left—see exercise @ex:order-of-evaluation).
This order is not determined by the metacircular evaluator, which inherits its
control structure from the underlying JavaScript in which it is implemented.#footnote[The order of argument-expression evaluation by the function
#py("list_of_values") in the metacircular
evaluator is determined by the order of evaluation of the arguments to
#py("pair"), which is used to construct the
argument list.
#idx("order of evaluation", sub: "in metacircular evaluator")
The version of
#py("list_of_values") in footnote @foot:mceval-higher-order of section @sec:mc-eval
calls #py("pair") directly; the version in the
text uses #py("map"), which calls
#py("pair"). (See exercise @ex:arg-eval-order.)]

Because we use #py("head") in #py("ev_appl_argument_expression_loop")
to extract successive argument expressions from #py("unev")
and #py("tail") at
#py("ev_appl_accumulate_arg") to extract the rest of the argument expressions,
the explicit-control evaluator will
evaluate the argument expressions of a combination in left-to-right order,
as required by the ECMAScript specification.

#idx("explicit-control evaluator for Python", sub: "argument evaluation")

#subheading([Function Application])

#anchor(<sec:procedure-application>)

The entry point #py("apply_dispatch") corresponds to
the #py("apply") function of the metacircular
evaluator. By the time we get to
#py("apply_dispatch"), the
#py("fun") register contains the function to apply
and #py("argl") contains the list of evaluated
arguments to which it must be applied. The saved value of
#py("continue") (originally passed to
#py("eval_dispatch") and saved at
#py("ev_application")), which tells where to return
with the result of the function application, is on the stack. When the application
is complete, the controller transfers to the entry point specified by the saved
#py("continue"), with the result of the application
in #py("val"). As with the metacircular
#py("apply"), there are two cases to consider.
Either the function to be applied is a primitive or it is a compound function.

#idx("applydispatch", decl: true)
#snippet(```python
"apply_dispatch",
  test(list(op("is_primitive_function"), reg("fun"))),
  branch(label("primitive_apply")),
  test(list(op("is_compound_function"), reg("fun"))),
  branch(label("compound_apply")),
  go_to(label("unknown_function_type")),
```)

We assume that each
#idx("explicit-control evaluator for Python", sub: "primitive functions")
primitive is implemented so as to obtain its arguments from
#py("argl") and place its result in
#py("val"). To specify how the machine handles
primitives, we would have to provide a sequence of controller instructions to
implement each primitive and arrange for
#py("primitive_apply") to dispatch to the
instructions for the primitive identified by the contents of
#py("fun"). Since we are interested in the structure
of the evaluation process rather than the details of the primitives, we will
instead just use an #py("apply_primitive_function")
operation that applies the function in fun to the arguments in
#py("argl"). For the purpose of simulating the
evaluator with the simulator of section @sec:simulator we use
the function #py("apply_primitive_function"), which
calls on the underlying JavaScript system to perform the application, just as we
did for the metacircular evaluator in section @sec:core-of-evaluator. After computing the value of the primitive
application, we restore #py("continue") and go to
the designated entry point.
#idx("primitiveapply", decl: true)
#snippet(```python
"primitive_apply",
  assign("val", list(op("apply_primitive_function"),
                     reg("fun"), reg("argl"))),
  restore("continue"),
  go_to(reg("continue")),
```)

The sequence of instructions labeled
#py("compound_apply") specifies the application of
#idx("explicit-control evaluator for Python", sub: "compound functions")
compound functions. To apply a compound function, we proceed in a way similar to
what we did in the metacircular evaluator. We construct a frame that binds the
function's parameters to the arguments, use this frame to extend the environment
carried by the function, and evaluate in this extended environment the body of the
function.

At this point the compound function is in register
#py("fun") and its arguments are in
#py("argl"). We extract the function's parameters
into #py("unev") and its environment into
#py("env"). We then replace the environment in
#py("env") with the environment constructed by
extending it with bindings of the parameters to the given arguments. We then
extract the body of the function into #py("comp").
The natural next step would be to restore the saved
#py("continue") and proceed to
#py("eval_dispatch") to evaluate the body and go to
the restored continuation with the result in
#py("val"), as is done for the last statement of a
sequence. But there is a complication!

The complication has two aspects. One is that
at any point in the evaluation of the body, a
#idx("explicit-control evaluator for Python", sub: "return statements")
return statement may require the
function to return the value of the return expression as the value of the body.
But a return statement may be nested arbitrarily deeply in the body; so the stack
at the moment the return statement is encountered is not necessarily the stack
that is needed for a return from the function. One way to make it possible to
adjust the stack for the return is to put a #emph[marker] on the stack that can be
found by the return code. This is implemented by the
#idx("register-machine language", sub: "pushmarkertostack")
#idx("pushmarkertostack (in register machine)")
#py("push_marker_to_stack") instruction. The return
code can then use the
#idx("register-machine language", sub: "revertstacktomarker")
#idx("revertstacktomarker (in register machine)")
#py("revert_stack_to_marker")
instruction to restore the stack to the place indicated by the marker before
evaluating the return expression.#footnote[The special instructions
#py("push_marker_to_stack") and
#py("revert_stack_to_marker") are not strictly
necessary and could be implemented by explicitly pushing and popping a marker
value onto and off the stack. Anything that could not be confused with a value in
the program can be used as a marker. See exercise @ex:push_marker_to_stack1.]

The other aspect of the complication is that if the evaluation of the body
terminates without executing a return statement, the value of the body must be
#py("undefined"). To handle this, we set up the
#py("continue") register to point to the entry point
#py("return_undefined") before going off to
#py("eval_dispatch") to evaluate the body. If a
return statement is not encountered during evaluation of the body, evaluation of
the body will continue at #py("return_undefined").
#idx("compoundapply", decl: true)
#snippet(```python
"compound_apply",
  assign("unev", list(op("function_parameters"), reg("fun"))),
  assign("env", list(op("function_environment"), reg("fun"))),
  assign("env", list(op("extend_environment"),
                     reg("unev"), reg("argl"), reg("env"))),
  assign("comp", list(op("function_body"), reg("fun"))),
  push_marker_to_stack(),
  assign("continue", label("return_undefined")),
  go_to(label("eval_dispatch")),
```)

The only places in the interpreter where the
#py("env") register is assigned a new value are
#py("compound_apply") and
#py("ev_block") (section @sec:block-assign-def-evaluation). Just as in the metacircular evaluator,
the new environment for evaluation of a function body is constructed from the
environment carried by the function, together with the argument list and the
corresponding list of names to be bound.

#idx("explicit-control evaluator for Python", sub: "function application")
#idx("explicit-control evaluator for Python", sub: "combinations")

When a return statement is evaluated at
#py("ev_return"), we use the
#py("revert_stack_to_marker") instruction to restore
the stack to its state at the beginning of the function call by removing all
values from the stack down to and including the marker. As a consequence,
#py("restore(\"continue\")") will restore the
continuation of the function call, which was saved at
#py("ev_application"). We then proceed to evaluate
the return expression, whose result will be placed in
#py("val") and thus be the value returned from the
function when we continue after the evaluation of the return expression.

#idx("evreturn", decl: true)
#snippet(```python
"ev_return",
  revert_stack_to_marker(),
  restore("continue"),
  assign("comp", list(op("return_expression"), reg("comp"))),
  go_to(label("eval_dispatch")),
```)

If no return statement is encountered during evaluation of the function body,
#idx("return value", sub: "undefined as")
evaluation continues at #py("return_undefined"), the
continuation that was set up at
#py("compound_apply"). To return
#py("undefined") from the function, we put
#py("undefined") into
#py("val") and go to the entry point that was put
onto the stack at #py("ev_application"). Before we
can restore that continuation from the stack, however, we must remove the marker
that was saved at #py("compound_apply").

#idx("returnundefined", decl: true)
#snippet(```python
"return_undefined",
  revert_stack_to_marker(),
  restore("continue"),
  assign("val", constant(undefined)),
  go_to(reg("continue")),
```)

#subheading([Return Statements and Tail Recursion])

#idx("explicit-control evaluator for Python", sub: "tail recursion")

#idx("tail recursion", sub: "explicit-control evaluator and")
#idx("return statement", sub: "handling in explicit-control evaluator")

In chapter @chap:fun we said that the process described by a
function
such as

#snippet(```python
function sqrt_iter(guess, x) {
    return is_good_enough(guess, x)
           ? guess
           : sqrt_iter(improve(guess, x), x);
}
```)

is an iterative process. Even though the
function
is syntactically recursive (defined in terms of itself), it is not logically
necessary for an evaluator to save information in passing from one call to
#py("sqrt_iter")
to the next.#footnote[We saw in
section @sec:designing-register-machines how to
implement such a process with a register machine that had no stack; the
state of the process was stored in a fixed set of registers.] An
evaluator that can execute a
function
such as
#py("sqrt_iter")
without requiring increasing storage as the
function
continues to call itself is called a
#idx("tail-recursive evaluator")
#idx("metacircular evaluator for Python", sub: "tail recursion and")
#idx("tail recursion", sub: "metacircular evaluator and")
#idx("return statement", sub: "tail recursion and")
#emph[tail-recursive] evaluator.

The metacircular implementation of the evaluator in chapter @chap:meta isn't
tail-recursive. It implements a return statement as a
constructor of a return value object containing the value to be
returned and inspects the result of a function call to see whether it is
such an object. If the evaluation of a function body produces a return value
object, the return value of the function is the contents of that object;
otherwise, the return value is
#py("undefined"). Both the construction of the
return value object and the eventual inspection of the result of the
function call are deferred operations, which lead to an accumulation of
information on the stack.

Our explicit-control evaluator #emph[is] tail-recursive, because it does not need to wrap up
return values for inspection and thus avoids the buildup of stack from deferred operations.
At #py("ev_return"), in order to evaluate the expression that
computes the return value of a function, we transfer directly to
#py("eval_dispatch") with nothing more on the stack
than right before the function call. We accomplish this by undoing any saves to
the stack by the function (which are useless because we are returning) using
#py("revert_stack_to_marker"). Then, rather than arranging
for #py("eval_dispatch") to come back here and #emph[then]
restoring #py("continue") from the stack and
continuing at that entry point, we restore
#py("continue") from the stack #emph[before] going to
#py("eval_dispatch") so that
#py("eval_dispatch") will continue at that entry
point after evaluating the expression. Finally, we transfer to
#py("eval_dispatch") without saving any information
on the stack. Thus, when we proceed to evaluate a return expression, the stack is
the same as just before the call to the function whose return value we are about
to compute. Hence, evaluating a return expression—even if it is a function call
(as in #py("sqrt_iter"), where the conditional
expression reduces to a call to
#py("sqrt_iter"))—will not cause any information to
accumulate on the stack.#footnote[This implementation of tail recursion is one
variety of a well-known optimization technique used by many compilers. In
compiling a function that ends with a function call, one can replace the call by a
jump to the called function's entry point. Building this strategy into the
interpreter, as we have done in this section, provides the optimization uniformly
throughout the language.]

If we did not think to take advantage of the fact that it is unnecessary to
hold on to the useless information on the stack while evaluating a return
expression, we might have taken the straightforward approach of evaluating
the return expression, coming back to restore the stack, and finally
continuing at
the entry point that is waiting for the result of the function call:

#syntax("
\"ev_return\",  // alternative implementation: not tail-recursive
  assign(\"comp\", list(op(\"return_expression\"), reg(\"comp\"))),
  assign(\"continue\", label(\"ev_restore_stack\")),
  go_to(label(\"eval_dispatch\")),
\"ev_restore_stack\",
  revert_stack_to_marker(),    // undo saves in current function
  restore(\"continue\"),         // undo save at ", $mono("ev_application")$, "
  go_to(reg(\"continue\")),
          ")

This may seem like a minor change to our previous code for evaluation of
return statements:

The only difference is that we delay undoing any register saves to the stack until after the evaluation of the return expression.

The interpreter will still give the same value for any expression. But this change
is fatal to the tail-recursive implementation, because we must now come back after
evaluating the return expression in order
to undo the (useless) register saves.

These extra saves will accumulate during a nest of
function
calls.

Consequently, processes such as
#py("sqrt_iter")
will require space proportional to the number of iterations rather than requiring
constant space.

This difference can be significant. For example,
#idx("iterative process", sub: "implemented by function call")
with tail recursion, an infinite loop can be expressed using only the
function-call and return mechanisms:

#snippet(```python
function count(n) {
    display(n);
    return count(n + 1);
}
```)

Without tail recursion, such a
function
would eventually run out of stack space, and expressing a true iteration
would require some control mechanism other than
function
call.

Note that our Python implementation requires the use of
#idx("tail recursion", sub: "return statement necessary for")
#py("return") in order to be tail-recursive.
Because the undoing of the register saves takes place at
#py("ev_return"), removing
#py("return") from the
#py("count") function above will cause it to
eventually run out of stack space. This explains the use of
#py("return") in the infinite driver loops in
chapter @chap:meta.

#idx("explicit-control evaluator for Python", sub: "return statements")

#exercise(label-name: <ex:missing-return>, [
Explain how the
#idx("explicit-control evaluator for Python", sub: "tail recursion")
#idx("tail recursion", sub: "explicit-control evaluator and")
stack builds up if #py("return") is
removed from #py("count"):

#snippet(```python
function count(n) {
    display(n);
    count(n + 1);
}
```)
])

#exercise(label-name: <ex:push_marker_to_stack1>, [
Implement the equivalent of #py("push_marker_to_stack") by
using #py("save") at
#py("compound_apply") to store a special marker
value on the stack. Implement the equivalent of
#py("revert_stack_to_marker") at
#py("ev_return") and
#py("return_undefined") as a loop that repeatedly
performs a #py("restore") until it hits the
marker. Note that this will require restoring a value to a register other than the one it
was saved from. (Although we are careful to avoid that in our evaluator, our stack
implementation actually allows it. See exercise @ex:stack-behavior.)
This is necessary because the only way to pop from the stack is by
restoring to a register.

Hint: You will need to create a unique constant to serve as the marker, for
example with #py("const marker = list(\"marker\")").
Because #py("list") creates a new pair, it cannot be
#py("===") to anything else on the stack.
])

#exercise(label-name: <ex:push_marker_to_stack2>, [
Implement
#idx("register-machine language", sub: "pushmarkertostack")
#idx("pushmarkertostack (in register machine)")
#py("push_marker_to_stack") and
#idx("register-machine language", sub: "revertstacktomarker")
#idx("revertstacktomarker (in register machine)")
#py("revert_stack_to_marker") as register-machine
instructions, following the implementation of
#py("save") and
#py("restore") in section @sec:ex-proc. Add functions
#py("push_marker") and
#py("pop_marker") to access stacks, mirroring the
implementation of #py("push") and
#py("pop") in section @sec:machine-model. Note that you do not
need to actually insert a marker into the stack. Instead, you can add a local
state variable to the stack model to keep track of the position of the last #py("save")
before each #py("push_marker_to_stack").
If you choose to put a marker on the stack, see the hint in
exercise @ex:push_marker_to_stack1.
])

#idx("tail recursion", sub: "explicit-control evaluator and")

#idx("explicit-control evaluator for Python", sub: "tail recursion")
