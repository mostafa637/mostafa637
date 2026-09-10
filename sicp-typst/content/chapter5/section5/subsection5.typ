// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([An Example of Compiled Code], label-name: <sec:compiled-code>)

#idx("compiler for Python", sub: "example compilation")
#idx("factorial", sub: "compilation of")

Now that we have seen all the elements of the compiler, let us examine
an example of compiled code to see how things fit together. We will
compile the
declaration
of a recursive
#py("factorial")
function
by passing as first argument to #py("compile") the result of applying #py("parse") to a string representation of the program (here using #idx("` (back quote)", sort: "''") #idx("quotation marks", sub: "back quotes") #idx("back quotes") #idx("string(s)", sub: "typed over multiple lines") back quotes #py("`")$dots.h$#py("`"), which work like single and double quotation marks but allow the string to span multiple lines):

#snippet(```python
compile(parse(`
function factorial(n) {
    return n === 1
           ? 1
           : factorial(n - 1) * n;
}
              `),
        "val",
        "next");
```)

We have specified that the value of the
declaration
should be placed in the #py("val") register.
We don't care what the compiled
code does after executing the
declaration,
so our choice of
#py("\"next\"")
as the linkage
descriptor is arbitrary.

The function #py("compile") determines that it was given a function definition, so it transforms it to a constant declaration and then
calls
#py("compile_declaration"). This compiles
code to compute the value to be assigned (targeted to
#py("val")), followed by code to install the
declaration,
followed by code to put the value of the
declaration (which is the value #py("undefined"))
into the target register, followed finally by the linkage code.
The #py("env") register
is preserved around the computation of the
value, because it is needed in order to install the
declaration.
Because
the linkage is
#py("\"next\""),
there is no linkage code
in this case. The skeleton of the compiled code is thus

#syntax(metaphrase[save #py("env") if modified by code to compute value], "
", metaphrase[compilation of declaration value, target #py("val"), linkage #py("\"next\"")], "
", metaphrase[restore #py("env") if saved above], "
perform(list(op(\"assign_symbol_value\"),
             constant(\"factorial\"),
             reg(\"val\"),
             reg(\"env\"))),
assign(\"val\", constant(undefined))
      ")

The expression that is

compiled to produce the value for the
name
#py("factorial")
is a
lambda
expression whose value is the
function
that computes factorials.
The function #py("compile")
handles this
by calling
#py("compile_lambda_expression"),
which compiles the
function
body, labels it as a new entry point, and generates the instruction that
will combine the
function
body at the new entry point with the runtime environment and assign the
result to #py("val"). The sequence then skips around
the compiled
function
code, which is inserted at this point. The
function
code itself begins by extending the
function's declaration
environment by a frame that binds the

parameter #py("n") to the
function
argument. Then comes the actual
function
body. Since this code for the value of the
name
doesn't modify the #py("env") register, the
optional #py("save")
and #py("restore") shown above aren't
generated. (The
function
code at
#py("entry1")
isn't executed at this point,
so its use of #py("env") is irrelevant.)
Therefore, the skeleton for the compiled code becomes

#syntax($mono(" ")mono(" ")$, "assign(\"val\", list(op(\"make_compiled_function\"),
                     label(\"entry1\"),
                     reg(\"env\"))),
  go_to(label(\"after_lambda2\")),
\"entry1\",
  assign(\"env\", list(op(\"compiled_function_env\"), reg(\"fun\"))),
  assign(\"env\", list(op(\"extend_environment\"),
                     constant(list(\"n\")),
                     reg(\"argl\"),
                     reg(\"env\"))),
  ", metaphrase[compilation of function body], "
\"after_lambda2\",
  perform(list(op(\"assign_symbol_value\"),
               constant(\"factorial\"),
               reg(\"val\"),
               reg(\"env\"))),
  assign(\"val\", constant(undefined))
      ")

A
function
body is always compiled (by
#py("compile_lambda_body"))

with target #py("val") and linkage
#py("\"next\"").
The
body
in this case consists of
a single
return statement:#footnote[Because of the #py("append_return_undefined") in #py("compile_lambda_body"), the body actually consists of a sequence with two return statements. However, the dead-code check in #py("compile_sequence") will stop after the compilation of the first return statement, so the body effectively consists of only a single return statement.]

#snippet(```python
return n === 1
       ? 1
       : factorial(n - 1) * n;
```)

The function
#py("compile_return_statement")
generates code to revert the stack using the marker and to restore
the #py("continue") register,
and then compiles the return
expression with target #py("val") and linkage
#py("\"return\""), because
its value is to be returned from the function.

The return expression is a conditional expression, for which #py("compile_conditional")
generates code that first computes the predicate (targeted to
#py("val")), then checks the result and branches
around the true branch if the predicate is false.
Registers #py("env")
and #py("continue")
are preserved around the predicate code, since they may be needed for the
rest of the
conditional
expression.

The

true and false branches are both
compiled with target #py("val") and linkage
#py("\"return\"").
(That is, the value of the conditional,
which is the value computed by either of its branches, is the value of the
function.)

#syntax($mono(" ")mono(" ")$, "revert_stack_to_marker(),
  restore(\"continue\"),
  ", metaphrase[save #py("continue"), #py("env") if modified by predicate and needed by branches], "
  ", metaphrase[compilation of predicate, target #py("val"), linkage #py("\"next\"")], "
  ", metaphrase[restore #py("continue"), #py("env") if saved above], "
  test(list(op(\"is_falsy\"), reg(\"val\"))),
  branch(label(\"false_branch4\")),
\"true_branch3\",
  ", metaphrase[compilation of true branch, target #py("val"), linkage #py("\"return\"")], "
\"false_branch4\",
  ", metaphrase[compilation of false branch, target #py("val"), linkage #py("\"return\"")], "
\"after_cond5\",
      ")

The predicate
#py("n === 1")
is a
function application (after transformation of the operator combination).
This looks up the
function expression (the symbol #py("\"===\""))
and places this value in
#py("fun").
It then assembles the arguments #py("1") and the value
of #py("n") into #py("argl").
Then it tests whether
#py("fun")
contains a primitive or a compound
function,
and dispatches to a primitive branch or a compound branch accordingly.
Both branches resume at the
#py("after_call")
label.
The compound branch must set up #py("continue") to jump past the primitive branch and push a marker to the stack to match the revert operation in the compiled return statement of the function.
The requirements to preserve registers around the evaluation of the
function and argument expressions
don't result in
any saving of registers, because in this case those evaluations don't
modify the registers in question.

#syntax($mono(" ")mono(" ")$, "assign(\"fun\", list(op(\"lookup_symbol_value\"),
                     constant(\"===\"), reg(\"env\"))),
  assign(\"val\", constant(1)),
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  assign(\"val\", list(op(\"lookup_symbol_value\"),
                     constant(\"n\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch6\")),
\"compiled_branch7\",
  assign(\"continue\", label(\"after_call8\")),
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch6\",
  assign(\"val\", list(op(\"apply_primitive_function\"),
                     reg(\"fun\"),
                     reg(\"argl\"))),
\"after_call8\",
      ")

The true branch, which is the constant 1, compiles (with target
#py("val") and linkage
#py("\"return\""))
to

#syntax($mono(" ")mono(" ")$, "assign(\"val\", constant(1)),
  go_to(reg(\"continue\")),
      ")

The code for the false branch is another
function
call, where the
function
is the value of the symbol
#py("\"*\""),
and the arguments
are #py("n") and the result of another
function
call (a call to #py("factorial")).
Each of these calls sets up
#py("fun")
and #py("argl") and its own primitive
and compound branches. Figure @fig:comp-factorial1
shows the complete compilation of the
declaration
of the #py("factorial")
function.
Notice that the possible #py("save") and
#py("restore") of
#py("continue") and
#py("env") around the predicate, shown above,
are in fact generated, because these registers are modified by the
function
call in the predicate and needed for the
function
call and the
#py("\"return\"")
linkage in the branches.

#sicp-figure([#syntax("
// construct the function and skip over the code for the function body
  assign(\"val\", list(op(\"make_compiled_function\"),
                     label(\"entry1\"), reg(\"env\"))),
  go_to(label(\"after_lambda2\")),
\"entry1\",                           // calls to ", $mono("factorial")$, " will enter here
  assign(\"env\", list(op(\"compiled_function_env\"), reg(\"fun\"))),
  assign(\"env\", list(op(\"extend_environment\"), constant(list(\"n\")),
                     reg(\"argl\"), reg(\"env\"))),
// begin actual function body
  revert_stack_to_marker(),         // starts with a return statement
  restore(\"continue\"),
  save(\"continue\"),                 // preserve registers across predicate
  save(\"env\"),
// compute ", $mono("n === 1")$, "
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"===\"), reg(\"env\"))),
  assign(\"val\", constant(1)),
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  assign(\"val\", list(op(\"lookup_symbol_value\"), constant(\"n\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch6\")),
\"compiled_branch7\",
  assign(\"continue\", label(\"after_call8\")),
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch6\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
\"after_call8\",                      // ", $mono("val")$, " now contains result of ", $mono("n === 1")$, "
  restore(\"env\"),
  restore(\"continue\"),
  test(list(op(\"is_falsy\"), reg(\"val\"))),
  branch(label(\"false_branch4\")),
\"true_branch3\",                     // return 1
  assign(\"val\", constant(1)),
  go_to(reg(\"continue\")),
\"false_branch4\",
// compute and return ", $mono("factorial(n - 1) * n")$, "
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"*\"), reg(\"env\"))),
  save(\"continue\"),
  save(\"fun\"),                      // save ", $mono("*")$, " function
  assign(\"val\", list(op(\"lookup_symbol_value\"), constant(\"n\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  save(\"argl\"),                     // save partial argument list for ", $mono("*")$, "
// compute ", $mono("factorial(n - 1)")$, " which is the other argument for ", $mono("*")$, "
  assign(\"fun\", list(op(\"lookup_symbol_value\"),
                     constant(\"factorial\"), reg(\"env\"))),
  save(\"fun\"),                      // save ", $mono("factorial")$, " function
    ")], caption: [Compilation of the declaration of the #py("factorial") function (continued on next page).], label-name: <fig:comp-factorial1>)

#sicp-figure([#syntax("
// compute ", $mono("n - 1")$, " which is the argument for ", $mono("factorial")$, "
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"-\"), reg(\"env\"))),
  assign(\"val\", constant(1)),
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  assign(\"val\", list(op(\"lookup_symbol_value\"), constant(\"n\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch10\")),
\"compiled_branch11\",
  assign(\"continue\", label(\"after_call12\")),
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch10\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
\"after_call12\",                     // ", $mono("val")$, " now contains result of ", $mono("n - 1")$, "
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  restore(\"fun\"),                   // restore ", $mono("factorial")$, "
// apply ", $mono("factorial")$, "
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch14\")),
\"compiled_branch15\",
  assign(\"continue\", label(\"after_call16\")),
  save(\"continue\"),                 // set up for compiled function $-$
  push_marker_to_stack(),           //   return in function will restore stack
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch14\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
\"after_call16\",                     // ", $mono("val")$, " now contains result of ", $mono("factorial(n - 1)")$, "
  restore(\"argl\"),                  // restore partial argument list for ", $mono("*")$, "
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  restore(\"fun\"),                   // restore ", $mono("*")$, "
  restore(\"continue\"),
// apply ", $mono("*")$, " and return its value
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch18\")),
\"compiled_branch19\", // note that a compound function here is called tail-recursively
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch18\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
  go_to(reg(\"continue\")),
\"after_call20\",
\"after_cond5\",
\"after_lambda2\",
// assign the function to the name ", $mono("factorial")$, "
  perform(list(op(\"assign_symbol_value\"),
               constant(\"factorial\"), reg(\"val\"), reg(\"env\"))),
  assign(\"val\", constant(undefined))
    ")], caption: [(continued)], label-name: <fig:continued_1>)

#idx("compiler for Python", sub: "example compilation")

#exercise(label-name: <ex:5_36>, [
Consider the following declaration of a factorial
function,
which is slightly different from the one given above:

#snippet(```python
function factorial_alt(n) {
    return n === 1
           ? 1
           : n * factorial_alt(n - 1);
}
```)

Compile this
function
and compare the resulting code with that produced for
#py("factorial"). Explain any differences you find.
Does either program execute more efficiently than the other?
])

#exercise(label-name: <ex:compiled-fact>, [
Compile the
#idx("iterative process", sub: "recursive process vs.")
#idx("recursive process", sub: "iterative process vs.")
iterative factorial
function

#snippet(```python
function factorial(n) {
    function iter(product, counter) {
        return counter > n
               ? product
               : iter(product * counter, counter + 1);
    }
    return iter(1, 1);
}
```)

Annotate the resulting code, showing the essential difference between
the code for iterative and recursive versions of
#py("factorial") that makes one process build up
stack space and the other run in constant stack space.
])

#exercise(label-name: <ex:compiled-code>, [
What
program
was compiled to produce the code shown in
figure @fig:compilation-example1?
])

#sicp-figure([#syntax($mono(" ")mono(" ")$, "assign(\"val\", list(op(\"make_compiled_function\"),
                     label(\"entry1\"), reg(\"env\"))),
\"entry1\"
  assign(\"env\", list(op(\"compiled_function_env\"), reg(\"fun\"))),
  assign(\"env\", list(op(\"extend_environment\"),
                     constant(list(\"x\")), reg(\"argl\"), reg(\"env\"))),
  revert_stack_to_marker(),
  restore(\"continue\"),
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"+\"), reg(\"env\"))),
  save(\"continue\"),
  save(\"fun\"),
  save(\"env\"),
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"g\"), reg(\"env\"))),
  save(\"fun\"),
  assign(\"fun\", list(op(\"lookup_symbol_value\"), constant(\"+\"), reg(\"env\"))),
  assign(\"val\", constant(2)),
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  assign(\"val\", list(op(\"lookup_symbol_value\"), constant(\"x\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch3\")),
\"compiled_branch4\"
  assign(\"continue\", label(\"after_call5\")),
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch3\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
\"after_call5\",
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  restore(\"fun\"),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch7\")),
\"compiled_branch8\",
  assign(\"continue\", label(\"after_call9\")),
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch7\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
\"after_call9\",
  assign(\"argl\", list(op(\"list\"), reg(\"val\"))),
  restore(\"env\"),
  assign(\"val\", list(op(\"lookup_symbol_value\"), constant(\"x\"), reg(\"env\"))),
  assign(\"argl\", list(op(\"pair\"), reg(\"val\"), reg(\"argl\"))),
  restore(\"fun\"),
  restore(\"continue\"),
  test(list(op(\"is_primitive_function\"), reg(\"fun\"))),
  branch(label(\"primitive_branch11\")),
    ")], caption: [An example of compiler output (continued on next page). See exercise @ex:compiled-code.], label-name: <fig:compilation-example1>)

#sicp-figure([#syntax("
\"compiled_branch12\",
  save(\"continue\"),
  push_marker_to_stack(),
  assign(\"val\", list(op(\"compiled_function_entry\"), reg(\"fun\"))),
  go_to(reg(\"val\")),
\"primitive_branch11\",
  assign(\"val\", list(op(\"apply_primitive_function\"), reg(\"fun\"), reg(\"argl\"))),
  go_to(reg(\"continue\")),
\"after_call13\",
\"after_lambda2\",
  perform(list(op(\"assign_symbol_value\"),
               constant(\"f\"), reg(\"val\"), reg(\"env\"))),
  assign(\"val\", constant(undefined))
    ")], caption: [(continued)], label-name: <fig:continued_2>)

#idx("factorial", sub: "compilation of")

#exercise(label-name: <ex:5_39>, [
What
#idx("order of evaluation", sub: "in compiler")
#idx("compiler for Python", sub: "order of argument evaluation")
order of evaluation does our compiler produce for
arguments of an application?
Is it left-to-right (as mandated by the ECMAScript specification), right-to-left, or some other order?
Where in the compiler is this order determined? Modify the compiler
so that it produces some other order of evaluation. (See the
discussion of order of evaluation for the explicit-control evaluator
in section @sec:eceval-core.) How does changing the
order of
argument
evaluation affect the efficiency of the code that
constructs the argument list?
])

#exercise(label-name: <ex:5_40>, [
One way to understand the compiler's
#idx("compiler for Python", sub: "stack usage")
#idx("preserving")
#py("preserving") mechanism for
optimizing stack usage is to see what extra operations would
be generated if we did not use this idea. Modify
#py("preserving") so
that it always generates the #py("save") and
#py("restore") operations.
Compile some simple expressions and identify the unnecessary stack
operations that are generated.
Compare the code to that generated with the
#py("preserving") mechanism intact.
])

#exercise(label-name: <ex:open-code>, [
Our compiler is clever about avoiding unnecessary stack operations,
but it is not clever at all when it comes to compiling calls to the primitive
functions
of the language in terms of the primitive operations
supplied by the machine. For example, consider how much code is
compiled to compute
#py("a + 1"):
The code sets up an argument list in #py("argl"), puts
the primitive addition
function
(which it finds by looking up the symbol
#py("\"+\"")
in the environment) into
#py("fun"),
and tests whether the
function
is primitive or compound. The
compiler always generates code to perform the test, as well as code
for primitive and compound branches (only one of which will be executed).
We have not shown the part of the controller that implements
primitives, but we presume that these instructions make use of
primitive arithmetic operations in the machine's data paths. Consider
how much less code would be generated if the compiler could
#idx("compiler for Python", sub: "open coding of primitives")
#idx("open coding of primitives")
#emph[open-code] primitives—that is, if it could generate code to
directly use these primitive machine operations. The expression
#py("a + 1")
might be compiled into something as simple as#footnote[We have used
the same symbol #py("+") here to denote both the
source-language
function
and the machine operation. In general there will not be a
one-to-one correspondence between primitives of the source language
and primitives of the machine.]

#snippet(```python
assign("val", list(op("lookup_symbol_value"), constant("a"), reg("env"))),
assign("val", list(op("+"), reg("val"), constant(1)))
```)

In this exercise we will extend our compiler to support open coding of
selected primitives. Special-purpose code will be generated for calls to these primitive
functions
instead of the general
function-application
code. In order to support this, we will augment
our machine with special argument registers
#py("arg1") and #py("arg2").
The primitive arithmetic operations of the machine will take their
inputs from #py("arg1") and
#py("arg2"). The results may be put into
#py("val"), #py("arg1"), or
#py("arg2").

The compiler must be able to recognize the application of an
open-coded primitive in the source program. We will augment the
dispatch in the #py("compile")
function
to recognize the names of these primitives in addition to the
syntactic forms it currently recognizes.
For each
syntactic
form our compiler has a code
generator. In this exercise we will construct a family of code generators
for the open-coded primitives.

+ The open-coded primitives, unlike the syntactic forms, all need their argument expressions evaluated. Write a code generator #py("spread_arguments") for use by all the open-coding code generators. The function #py("spread_arguments") should take a list of argument expressions and compile the given argument expressions targeted to successive argument registers. Note that an argument expression may contain a call to an open-coded primitive, so argument registers will have to be preserved during argument-expression evaluation.
+ The Python operators #py("==="), #py("*"), #py("-"), and #py("+"), among others, are implemented in the register machine as primitive functions and are referred to in the global environment with the symbols #py("\"===\""), #py("\"*\""), #py("\"-\""), and #py("\"+\""). In Python, it is not possible to redeclare these names, because they do not meet the syntactic restrictions for names. This means it is safe to open-code them. For each of the primitive functions #py("==="), #py("*"), #py("-"), and #py("+"), write a code generator that takes an application with a function expression that names that function, together with a target and a linkage descriptor, and produces code to spread the arguments into the registers and then perform the operation targeted to the given target with the given linkage. Make #py("compile") dispatch to these code generators.
+ Try your new compiler on the #py("factorial") example. Compare the resulting code with the result produced without open coding.
])
