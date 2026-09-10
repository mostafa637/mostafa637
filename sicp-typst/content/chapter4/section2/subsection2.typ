// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([An Interpreter with Lazy Evaluation])

In this section we will implement a normal-order language that is
the same as
Python
except that compound
functions
are non-strict in each argument. Primitive
functions
will still be strict. It is not difficult to modify the evaluator of
section @sec:core-of-evaluator so that the language it
interprets behaves this way. Almost all the required changes center around
function
application.

The basic idea is that, when applying a
function,
the interpreter must determine which arguments are to be evaluated and which
are to be delayed. The delayed arguments are not evaluated; instead, they
are transformed into objects called
#idx("thunk")
#emph[thunk]s.#footnote[The word #emph[thunk] was invented by an informal
#idx("thunk", sub: "origin of name")
working group that was discussing the implementation of call-by-name
#idx("Algol", sub: "thunks")
in Algol 60. They observed that most of the analysis of ("thinking about") the expression could be done at compile time; thus, at run
time, the expression would already have been "thunk" about
#idx("Ingerman, Peter")
(Ingerman et al. 1960).]
The thunk must contain the information required to produce the value
of the argument when it is needed, as if it had been evaluated at
the time of the application. Thus, the thunk must contain the
argument expression and the environment in
which the
function
application is being evaluated.

The process of evaluating the expression in a thunk is called
#idx("forcing", sub: "of thunk")
#idx("thunk", sub: "forcing")
#emph[forcing].#footnote[This is analogous to the
forcing of
the delayed objects that were introduced in chapter @chap:state to
represent streams. The critical difference between what we are
doing here and what we did in chapter @chap:state is that we are building
delaying and forcing into the evaluator, and thus making this uniform
and automatic throughout the language.]
In general, a thunk will be forced only when its value is needed:
when it is passed to a primitive
function
that will use the value of the thunk; when it is the value of a predicate of
a conditional; and when it is the value of
a function expression
that is about to be
applied as a
function.
One design choice we have available is whether or not to
#idx("memoization", sub: "of thunks")
#emph[memoize] thunks, similar to the optimization for streams in
section @sec:delayed-lists. With memoization, the first
time a thunk is forced, it stores the value that is computed. Subsequent
forcings simply return the stored value without repeating the computation.
We'll make our interpreter memoize, because this is more efficient for
many applications. There are tricky considerations here,
however.#footnote[Lazy evaluation combined with memoization is sometimes
referred to as
#idx("call-by-need argument passing")
#emph[call-by-need] argument passing, in contrast to
#emph[call-by-name] argument passing.
#idx("call-by-name argument passing")
(Call-by-name, introduced in
#idx("Algol", sub: "call-by-name argument passing")
Algol 60, is similar to non-memoized lazy
evaluation.) As language designers, we can build our evaluator to memoize,
not to memoize, or leave this an option for programmers
(exercise @ex:user-controlled-strictness). As you might
expect from chapter @chap:state, these choices raise issues that become both
subtle and confusing in the presence of assignments. (See
exercises @ex:delay-side-effects
and @ex:memoize-or-not.)
An excellent article by
#idx("Clinger, William")
Clinger (1982) attempts to clarify the
multiple dimensions of confusion that arise here.]

#idx("thunk")

#subheading([Modifying the evaluator])

The main difference between the lazy evaluator and the one in
section @sec:mc-eval is in the handling of
function
applications in
#py("evaluate")
and
#py("apply").

#idx("evaluate (lazy)")
The #py("is_application") clause of #idx("evaluate (lazy)") #py("evaluate") becomes

#snippet(```python
: is_application(component)
? apply(actual_value(function_expression(component), env),
        arg_expressions(component), env)
```)

This is almost the same as the
#py("is_application")
clause of
#py("evaluate")
in section @sec:core-of-evaluator. For lazy evaluation,
however, we call #py("apply") with the
argument
expressions, rather than the arguments produced by evaluating them. Since
we will need the environment to construct thunks if the arguments are to be
delayed, we must pass this as well.  We still evaluate the
function expression,
because
#py("apply") needs the actual
function
to be applied in order to dispatch on its type (primitive versus compound)
and apply it.

Whenever we need the actual value of an expression, we use

#idx("actualvalue", decl: true)
#snippet(```python
def actual_value(exp, env):
    return force_it(evaluate(exp, env))
```)

instead of just
#py("evaluate"),
so that if the expression's value is a thunk, it will be forced.

Our new version of #py("apply") is also almost the
same as the version in section @sec:core-of-evaluator.
The difference is that
#py("evaluate")
has passed in unevaluated
argument
expressions: For primitive
functions
(which are strict), we evaluate all the arguments before applying the
primitive; for compound
functions
(which are non-strict) we delay all the
arguments before applying the
function.

#idx("apply (lazy)", decl: true)
#snippet(```python
def apply(fun, args, env):
    if is_primitive_function(fun):
        return apply_primitive_function(fun, list_of_arg_values(args, env))
    elif is_compound_function(fun):
        result = evaluate(function_body(fun), extend_environment(function_parameters(fun), list_of_delayed_args(args, env), function_environment(fun)))
        return return_value_content(result) if is_return_value(result) else None
    else:
        error("unknown function type -- apply", fun)
```)

The
functions
that process the arguments are just like
#py("list_of_values")
from section @sec:core-of-evaluator,
except that
#py("list_of_delayed_args")
delays the arguments instead of evaluating them, and
#py("list_of_arg_values")
uses
#py("actual_value")
instead of
#py("evaluate"):
#idx("listofargvalues", decl: true)#idx("listofdelayedargs", decl: true)
#snippet(```python
def list_of_arg_values(exps, env):
    return map(lambda exp: (actual_value(exp, env)), exps)
def list_of_delayed_args(exps, env):
    return map(lambda exp: (delay_it(exp, env)), exps)
```)

The other place we must change the evaluator is in the handling of
conditionals,
where we must use
#py("actual_value")
instead of
#py("evaluate")
to get the value of the predicate
expression before testing whether it is true or false:

#idx("evalconditional (lazy)", decl: true)
#snippet(```python
def eval_conditional(component, env):
    return evaluate(conditional_consequent(component), env) if is_truthy(actual_value(conditional_predicate(component), env)) else evaluate(conditional_alternative(component), env)
```)

Finally, we must change the
#idx("driver loop", sub: "in lazy evaluator")
#py("driver_loop")
function
(from section @sec:running-eval) to use
#py("actual_value")
instead of
#py("evaluate"),
so that if a delayed value is propagated back to the
read-evaluate-print loop,
it will be forced before being printed.
We also change the prompts to indicate that
this is the lazy evaluator:
#idx("prompts", sub: "lazy evaluator")#idx("driverloop", sub: "for lazy evaluator", decl: true)
#snippet(```python
input_prompt = "L-evaluate input: "
output_prompt = "L-evaluate value: "
def driver_loop(env):
    input = user_read(input_prompt)
    if is_none(input):
        print("evaluator terminated")
    else:
        program = parse(input)
        locals = scan_out_declarations(program)
        unassigneds = list_of_unassigned(locals)
        program_env = extend_environment(locals, unassigneds, env)
        output = actual_value(program, program_env)
        user_print(output_prompt, output)
        return driver_loop(program_env)
```)

With these changes made, we can start the evaluator and test it. The
successful evaluation of the
#py("try_me")
expression
discussed in section @sec:evaluation-order indicates
that the interpreter is performing lazy evaluation:

#snippet(```python
the_global_environment = setup_environment()
driver_loop(the_global_environment)
```)

#prompt(```python
L-evaluate input:
```)

#snippet(```python
def try_me(a, b):
    return 1 if a == 0 else b
```)

#output(```python
def try_me(a, b):
    return 1 if a == 0 else b
```)

#prompt(```python
L-evaluate input:
```)

#snippet(```python
try_me(0, head(None))
```)

#output(```python
try_me(0, head(None))
```)

#subheading([Representing thunks])

#idx("thunk", sub: "implementation of")

Our evaluator must arrange to create thunks when
functions
are applied to arguments and to force these thunks later. A thunk must
package an expression together with the environment, so that the argument
can be produced later. To force the thunk, we simply extract the expression
and environment from the thunk and evaluate the expression in the
environment. We use
#py("actual_value")
rather than
#py("evaluate")
so that in case the value of the expression is itself a thunk, we will force
that, and so on, until we reach something that is not a thunk:
#idx("forceit", decl: true)
#snippet(```python
def force_it(obj):
    return actual_value(thunk_exp(obj), thunk_env(obj)) if is_thunk(obj) else obj
```)

One easy way to package an expression with an environment is to make a list
containing the expression and the environment. Thus, we create a thunk as
follows:
#idx("delayit", decl: true)
#snippet(```python
def delay_it(exp, env):
    return llist("thunk", exp, env)
def is_thunk(obj):
    return is_tagged_list(obj, "thunk")
def thunk_exp(thunk):
    return head(tail(thunk))

def thunk_env(thunk):
    return head(tail(tail(thunk)))
```)

Actually, what we want for our interpreter is not quite this, but
rather thunks that have been memoized.

When a thunk is forced, we will turn it into an evaluated thunk by replacing
the stored expression with its value and changing the
#py("thunk") tag so that it can be recognized as
already evaluated.#footnote[Notice that we also erase the
#py("env") from the thunk once the expression's
value has been computed. This makes no difference in the values returned by
the interpreter. It does help save space, however, because removing the
reference from the thunk to the #py("env") once it is
no longer needed allows this structure to be
#idx("garbage collection", sub: "memoization and")
#idx("memoization", sub: "garbage collection and")
#emph[garbage-collected] and its space
recycled, as we will discuss in
section @sec:storage-allocation.

Similarly, we could have allowed unneeded environments in the memoized
delayed objects of section @sec:delayed-lists
to be garbage-collected, by having
#py("memo")
do something like
#py("fun = None;")
to discard the
function #py("fun")
(which includes the environment in which the
lambda expression that makes up the tail of the stream
was evaluated) after storing its
value.]
#idx("forceit", sub: "memoized version", decl: true)
#snippet(```python
def is_evaluated_thunk(obj):
    return is_tagged_list(obj, "evaluated_thunk")
def thunk_value(evaluated_thunk):
    return head(tail(evaluated_thunk))

def force_it(obj):
    if is_thunk(obj):
        result = actual_value(thunk_exp(obj), thunk_env(obj))
        set_head(obj, "evaluated_thunk")
        set_head(tail(obj), result)
        set_tail(tail(obj), None)
        return result
    elif is_evaluated_thunk(obj):
        return thunk_value(obj)
    else:
        return obj
```)

Notice that the same
#py("delay_it")
function
works both with and
without memoization.#idx("thunk", sub: "implementation of")

#exercise(label-name: <ex:delay-side-effects>, [
Suppose we type in the following
declarations
to the lazy evaluator:

#snippet(```python
count = 0
def id(x):
    global count
    count = count + 1
    return x
```)

Give the missing values in the following sequence of interactions, and explain
your answers.#footnote[This exercise demonstrates that the interaction between
lazy evaluation and side effects can be very confusing. This is just what you
might expect from the discussion in chapter @chap:state.]

#snippet(```python
w = id(id(10))
```)

#prompt(```python
L-evaluate input:
```)

#syntax("
count
    ")

#output(```python
w = id(id(10))
```)

#prompt(```python
L-evaluate input:
```)

#syntax("
w
    ")

#output(```python
w = id(id(10))
```)

#prompt(```python
L-evaluate input:
```)

#syntax("
count
    ")

#output(```python
w = id(id(10))
```)
])

#exercise(label-name: <ex:force-operator>, [
The function #py("evaluate")
uses
#py("actual_value")
rather than
#py("evaluate")
to evaluate the
function expression
before passing it to
#py("apply"), in order to force the value of the
function expression.
Give an example that demonstrates the need for this forcing.
])

#exercise(label-name: <ex:memoize-or-not>, [
Exhibit a program that you would expect to run much more slowly without
memoization than with memoization. Also, consider the following
interaction, where the #py("id")
function
is defined as in exercise @ex:delay-side-effects and
#py("count") starts at 0:

#snippet(```python
def square(x):
    return x * x
```)

#prompt(```python
L-evaluate input:
```)

#syntax("
square(id(10))
      ")

#output(```python
def square(x):
    return x * x
```)

#prompt(```python
L-evaluate input:
```)

#syntax("
count
      ")

#output(```python
def square(x):
    return x * x
```)

Give the responses both when the evaluator memoizes and when it does not.
])

#exercise(label-name: <ex:force-sequence>, [
Cy D. Fect, a reformed C programmer, is worried that some side effects
may never take place, because the lazy evaluator doesn't force the
statements in a sequence.
Since the value of a statement in a sequence
may not be used (the statement may be there only for
its effect, such as assigning to a variable or printing), there may be
no subsequent use of this value (e.g., as an argument to a primitive
function) that will cause it to be forced.
Cy thus thinks that when
evaluating sequences, we must force all statements in the sequence.
He proposes to modify
#py("evaluate_sequence")
from section @sec:core-of-evaluator to use
#py("actual_value")
rather than
#py("evaluate"):

#snippet(```python
def eval_sequence(stmts, env):
    if is_empty_sequence(stmts):
        return None
    elif is_last_statement(stmts):
        return actual_value(first_statement(stmts), env)
    else:
        first_stmt_value = actual_value(first_statement(stmts), env)
        if is_return_value(first_stmt_value):
            return first_stmt_value
        else:
            return eval_sequence(rest_statements(stmts), env)
```)

+ Ben Bitdiddle thinks Cy is wrong. He shows Cy the #py("for_each") function described in exercise @ex:for-each, which gives an important example of a sequence with side effects: #idx("foreach", decl: true) #snippet(```python def for_each(fun, items): if is_none(items): return "done" else: fun(head(items)) for_each(fun, tail(items)) ```) He claims that the evaluator in the text (with the original #py("eval_sequence")) handles this correctly: #prompt(```python L-evaluate input: ```) #snippet(```python for_each(print, llist(57, 321, 88)) ```) #output(```python 57 321 88 L-evaluate value: "done" ```) Explain why Ben is right about the behavior of #py("for_each").
+ Cy agrees that Ben is right about the #py("for_each") example, but says that that's not the kind of program he was thinking about when he proposed his change to #py("eval_sequence"). He declares the following two functions in the lazy evaluator: #snippet(```python def f1(x): x = pair(x, llist(2)) return x def f2(x): def f(e): e return x return f(x = pair(x, llist(2))) ```) What are the values of #py("f1(1)") and #py("f2(1)") with the original #py("eval_sequence")? What would the values be with Cy's proposed change to #py("eval_sequence")?
+ Cy also points out that changing #py("eval_sequence") as he proposes does not affect the behavior of the example in part a. Explain why this is true.
+ How do you think sequences ought to be treated in the lazy evaluator? Do you like Cy's approach, the approach in the text, or some other approach?
])

#exercise(label-name: <ex:user-controlled-strictness>, [
The approach taken in this section is somewhat unpleasant, because it
makes an incompatible change to
Python.
It might be nicer to implement lazy evaluation as an
#idx("upward compatibility")
#emph[upward-compatible extension], that is, so that ordinary
Python
programs will work as before. We can do this by
introducing optional parameter declaration as a new syntactic form inside function
declarations to let the user control whether or not arguments are to be
delayed. While we're at it, we may as well also give the user the
choice between delaying with and without memoization. For example, the
declaration

#syntax("
def f(a, b, c, d):
    parameters(\"strict\", \"lazy\", \"strict\", \"lazy_memo\")
    ", $dots.h$)

would define #py("f") to be a
function
of four arguments, where the first and third arguments are evaluated when the
function
is called, the second argument is delayed, and the fourth argument is both
delayed and memoized.

You can assume that the parameter declaration is always
the first statement in the body of a function definition,
and if it is omitted, all parameters are strict.
Thus, ordinary function definition
will produce the same behavior as ordinary Python,
while adding the
#py("\"lazy_memo\"")
declaration to each parameter of every compound	function
will produce the behavior of the lazy evaluator defined in this section.
Design and implement the changes required to produce such an extension to
Python. The #py("parse") function will
treat parameter declarations as	function applications, so you need to
modify #py("apply") to dispatch to your
implementation of the new syntactic form.

You must also arrange for
#py("evaluate")
or #py("apply") to determine when arguments are to be
delayed, and to force or delay arguments accordingly, and you must arrange
for forcing to memoize or not, as appropriate.
])

#idx("lazy evaluator")
