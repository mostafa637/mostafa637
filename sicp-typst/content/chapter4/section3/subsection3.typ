// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Implementing the #py("amb") Evaluator], label-name: <sec:amb-implementation>)

#idx("nondeterministic evaluator")

The evaluation of an ordinary
Python program
may return a value, may never terminate, or may signal an error.
In nondeterministic
Python
the evaluation of
a program
may in addition result in the discovery of
a dead end, in which case evaluation must backtrack to a previous choice
point. The interpretation of nondeterministic
Python
is complicated by this extra case.

We will construct the #py("amb") evaluator for
nondeterministic
Python
by modifying the
#idx("analyzing evaluator", sub: "as basis for nondeterministic evaluator")
analyzing evaluator of
section @sec:separating-analysis.#footnote[We chose to
implement the lazy evaluator in
section @sec:lazy-evaluation as a modification of the
ordinary metacircular evaluator of
section @sec:core-of-evaluator. In contrast, we will
base the #py("amb") evaluator on the analyzing
evaluator of section @sec:separating-analysis, because
the execution
functions
in that evaluator provide a convenient framework for implementing
backtracking.] As in the analyzing evaluator, evaluation of
a component
is accomplished by calling an
#idx("executionfunction", sub: "in nondeterministic evaluator")
execution
function
produced by analysis of that
component.
The difference between the interpretation of ordinary
Python
and the interpretation of nondeterministic
Python
will be entirely
in the execution
functions.

#subheading([Execution functions and continuations])

#idx("continuation", sub: "in nondeterministic evaluator")

Recall that the
#idx("execution function", sub: "in nondeterministic evaluator")
execution
functions
for the ordinary evaluator take one argument: the environment of execution.
In contrast, the execution
functions
in the #py("amb") evaluator take three arguments:
the environment, and two
functions
called
#emph[continuation functions].
The evaluation of
a component
will finish by calling one of these two
continuations: If the evaluation results in a value, the
#idx("success continuation (nondeterministic evaluator)")
#emph[success continuation] is called with that value; if the evaluation
results in the discovery of a dead end, the
#idx("failure continuation (nondeterministic evaluator)")
#emph[failure continuation] is called. Constructing and calling
appropriate continuations is the mechanism by which the nondeterministic
evaluator implements backtracking.

It is the job of the success continuation to receive a value and proceed
with the computation. Along with that value, the success continuation is
passed another failure continuation, which is to be called subsequently if
the use of that value leads to a dead end.

It is the job of the failure continuation to try another branch of the
nondeterministic process. The essence of the nondeterministic
language is in the fact that
components
may represent choices among
alternatives. The evaluation of such
a component
must proceed with
one of the indicated alternative choices, even though it is not known
in advance which choices will lead to acceptable results. To deal
with this, the evaluator picks one of the alternatives and passes this
value to the success continuation. Together with this value, the
evaluator constructs and passes along a failure continuation that can
be called later to choose a different alternative.

A failure is triggered during evaluation (that is, a failure
continuation is called) when a user program explicitly rejects the
current line of attack (for example, a call to
#py("require") may result in execution of
#py("amb()"),
an expression that always
fails—see section @sec:amb). The failure
continuation in hand at that point will cause the most recent choice point
to choose another alternative. If there are no more alternatives to be
considered at that choice point, a failure at an earlier choice point
is triggered, and so on. Failure continuations are also invoked by
the driver loop in response to a
#py("retry")
request, to find another value of the
program.

In addition, if a side-effect operation (such as assignment to a
variable) occurs on a branch of the process resulting from a choice,
it may be necessary, when the process finds a dead end, to undo the
side effect before making a new choice. This is accomplished by
having the side-effect operation produce a failure continuation that
undoes the side effect and propagates the failure.

In summary, failure continuations are constructed by

- #py("amb") expressions—to provide a mechanism to make alternative choices if the current choice made by the #py("amb") expression leads to a dead end;
- the top-level driver—to provide a mechanism to report failure when the choices are exhausted;
- assignments—to intercept failures and undo assignments during backtracking.

Failures are initiated only when a dead end is encountered. This occurs

- if the user program executes #py("amb()")
- if the user types #py("retry") at the top-level driver.

Failure continuations are also called during processing of a failure:

- When the failure continuation created by an assignment finishes undoing a side effect, it calls the failure continuation it intercepted, in order to propagate the failure back to the choice point that led to this assignment or to the top level.
- When the failure continuation for an #py("amb") runs out of choices, it calls the failure continuation that was originally given to the #py("amb"), in order to propagate the failure back to the previous choice point or to the top level.

#idx("continuation", sub: "in nondeterministic evaluator")

#subheading([Structure of the evaluator])

The syntax- and data-representation
functions
for the #py("amb") evaluator, and also the basic
#idx("analyze", sub: "nondeterministic")
#py("analyze")
function,
are identical to those in the evaluator of
section @sec:separating-analysis, except for the fact
that we need additional syntax
functions
to recognize
the #py("amb") syntactic form:
#idx("isamb", decl: true)
#snippet(```python
def is_amb(component):
    return is_tagged_list(component, "application") and is_name(function_expression(component)) and symbol_of_name(function_expression(component)) == "amb"
def amb_choices(component):
    return arg_expressions(component)
```)

We continue to use the parse function of
section @sec:representing-expressions, which
doesn't support #py("amb") as a syntactic
form and instead treats #py("amb(")
$dots.h$
#py(")") as
a function application. The function
#py("is_amb") ensures that
whenever the name
#py("amb") appears as the function
expression of an application, the evaluator treats the
"application" as
a nondeterministic choice point.#footnote[With this treatment,
#py("amb") is no longer a name with
proper scoping.	To avoid confusion, we must
refrain from declaring #py("amb") as a
name in our nondeterministic programs.]

We must also add to the dispatch in #py("analyze") a
clause that will recognize
such expressions and generate an appropriate execution function:

#syntax($dots.h$, "
: is_amb(component)
? analyze_amb(component)
: is_application(component)
", $dots.h$)

The top-level
function
#py("ambeval") (similar to the version of
#py("evaluate")
given in section @sec:separating-analysis) analyzes the
given
component
and applies the resulting execution
function
to the given environment, together with two given continuations:

#idx("ambeval", decl: true)
#snippet(```python
def ambeval(component, env, succeed, fail):
    return analyze(component)(env, succeed, fail)
```)

A success
#idx("success continuation (nondeterministic evaluator)")
#idx("continuation", sub: "in nondeterministic evaluator")
continuation is a
function
of two arguments: the value just obtained and another failure continuation to
be used if that value leads to a subsequent failure. A
#idx("failure continuation (nondeterministic evaluator)")
failure continuation
is a
function
of no arguments. So
the general form of an
#idx("execution function", sub: "in nondeterministic evaluator")
execution
function
is

#syntax("
# ", $mono("succeed") thin$, " is ", $mono("lambda value, fail:") dots.h$, "
# ", $mono("fail") thin$, " is ", $mono("lambda:") dots.h$, "
lambda env, succeed, fail: ", $dots.h$)

For example, executing

#syntax("
ambeval(", meta("component"), ",
        the_global_environment,
        lambda value, fail: value,
        lambda: \"failed\")
	  ")

will attempt to evaluate the given
component
and will return either the
component's
value (if the evaluation succeeds) or the
string #py("\"failed\"")
(if the evaluation fails).
The call to #py("ambeval") in the driver loop shown
below uses much more complicated continuation
functions,
which continue the loop and support the
#py("retry")
request.

Most of the complexity of the #py("amb") evaluator
results from the mechanics of passing the continuations around as the
execution
functions
call each other. In going through the following code, you should compare
each of the execution
functions
with the corresponding
function
for the ordinary evaluator given in
section @sec:separating-analysis.

#subheading([Simple expressions])

#idx("analyze...", sub: "nondeterministic")

The execution
functions
for the simplest kinds of expressions are
essentially the same as those for the ordinary evaluator, except for the
need to manage the continuations. The execution
functions
simply succeed with the value of the expression, passing along the failure
continuation that was passed to them.

#snippet(```python
def analyze_literal(component):
    return lambda env, succeed, fail: (succeed(literal_value(component), fail))
```)

#snippet(```python
def analyze_name(component):
    return lambda env, succeed, fail: (succeed(lookup_symbol_value(symbol_of_name(component), env), fail))
```)

#snippet(```python
def analyze_lambda_expression(component):
    params = lambda_parameter_symbols(component)
    bfun = analyze(lambda_body(component))
    return lambda env, succeed, fail: (succeed(make_function(params, bfun, env), fail))
```)

Notice that looking up a
name
always "succeeds."
#idx("failure, in nondeterministic computation", sub: "bug vs.")
If
#py("lookup_symbol_value")
fails to find the
name,
it signals an
error, as usual. Such a "failure" indicates a program
bug—a reference to an unbound
name
it is not an indication
that we should try another nondeterministic choice instead of the one that
is currently being tried.

#subheading([Conditionals and sequences])

Conditionals are also handled in a similar way as in the ordinary
evaluator. The execution
function
generated by
#py("analyze_conditional")
invokes the predicate execution
function #py("pfun")
with a success continuation that checks whether the predicate value is true
and goes on to execute either the consequent or the alternative. If the
execution of
#py("pfun")
fails, the original failure continuation for
the
conditional
expression is called.

#syntax("
def analyze_conditional(component):
    pfun = analyze(conditional_predicate(component))
    cfun = analyze(conditional_consequent(component))
    afun = analyze(conditional_alternative(component))
    return lambda env, succeed, fail: (pfun(env, lambda pred_value, fail2: (cfun(env, succeed, fail2) if is_truthy(pred_value) else afun(env, succeed, fail2)), fail))
")

Sequences are also handled in the same way as in the previous
evaluator, except for the machinations in the
subfunction
#py("sequentially") that are required for passing the
continuations. Namely, to sequentially execute #py("a")
and then #py("b"), we call
#py("a") with a success continuation that calls
#py("b").

#syntax("
def analyze_sequence(stmts):
    def sequentially(a, b):
        return lambda env, succeed, fail: (a(env, lambda a_value, fail2: (succeed(a_value, fail2) if is_return_value(a_value) else b(env, succeed, fail2)), fail))
    def loop(first_fun, rest_funs):
        return first_fun if is_none(rest_funs) else loop(sequentially(first_fun, head(rest_funs)), tail(rest_funs))
    funs = map(analyze, stmts)
    return lambda env: (None) if is_none(funs) else loop(head(funs), tail(funs))
")

#subheading([Declarations and assignments])

Declarations
are another case where we must go to some trouble to
manage the continuations, because it is necessary to evaluate the
declaration-value expression before actually declaring the new name.
To accomplish this, the
declaration-value
execution
function #py("vfun")
is called with the environment, a success continuation, and the
failure continuation. If the execution of
#py("vfun")
succeeds, obtaining a value #py("val") for the
declared name, the name is declared and the success is propagated:

#snippet(```python
def analyze_declaration(component):
    symbol = declaration_symbol(component)
    vfun = analyze(declaration_value_expression(component))
    def the_declaration(env, succeed, fail):
        def success(val, fail2):
            assign_symbol_value(symbol, val, env)
            return succeed(None, fail2)
        return vfun(env, success, fail)
    return the_declaration
```)

Assignments
#idx("failure continuation (nondeterministic evaluator)", sub: "constructed by assignment")
are more interesting. This is the first place where we
really use the continuations, rather than just passing them around.
The execution
function
for assignments starts out like the one for
declarations.
It first attempts
to obtain the new value to be assigned to the
name.
If this evaluation of
#py("vfun")
fails, the assignment fails.

If
#py("vfun")
succeeds, however, and we go on to make the assignment, we must consider the
possibility that this branch of the computation might later fail, which will
require us to backtrack out of the assignment. Thus, we must arrange to
undo the assignment as part of the backtracking process.#footnote[We
didn't worry about undoing declarations, since we assume that a name can't be used prior to the evaluation of its declaration, #idx("internal declaration", sub: "in nondeterministic evaluator") so its previous value doesn't matter.]

This is accomplished by giving
#py("vfun")
a success continuation (marked with the comment "\*1\*" below)
that saves the old value of the variable before assigning the new value to
the variable and proceeding from the assignment. The failure continuation
that is passed along with the value of the assignment (marked with the
comment "\*2\*" below) restores the old value of the variable
before continuing the failure. That is, a successful assignment provides a
failure continuation that will intercept a subsequent failure; whatever
failure would otherwise have called #py("fail2") calls
this
function
instead, to undo the assignment before actually calling
#py("fail2").

#snippet(```python
def analyze_assignment(component):
    symbol = assignment_symbol(component)
    vfun = analyze(assignment_value_expression(component))
    def the_assignment(env, succeed, fail):
        def success(val, fail2):              # *1*
            old_value = lookup_symbol_value(symbol, env)
            assign_symbol_value(symbol, val, env)
            def restore():                    # *2*
                assign_symbol_value(symbol, old_value, env)
                return fail2()
            return succeed(val, restore)
        return vfun(env, success, fail)
    return the_assignment
```)

#subheading([Return statements and blocks])

Analyzing return statements is straightforward.
The return expression is analyzed to produce an execution function.
The execution function for the return statement calls that execution
function with a success continuation that wraps the return value
in a return value object and passes it to the original success continuation.

#snippet(```python
def analyze_return_statement(component):
    rfun = analyze(return_expression(component))
    return lambda env, succeed, fail: (rfun(env, lambda val, fail2: (succeed(make_return_value(val), fail2)), fail))
```)

The execution function for blocks calls the body's execution
function on an extended environment, without changing success or
failure continuations.

#snippet(```python
def analyze_block(component):
    body = block_body(component)
    locals = scan_out_declarations(body)
    unassigneds = list_of_unassigned(locals)
    bfun = analyze(body)
    return lambda env, succeed, fail: (bfun(extend_environment(locals, unassigneds, env), succeed, fail))
```)

#subheading([Function applications])

The execution
function
for applications contains no new ideas except for the technical complexity
of managing the continuations. This complexity arises in
#py("analyze_application"),
due to the need to keep track of the success and failure continuations as
we evaluate the
argument expressions.
We use a
function #py("get_args")
to evaluate the list of
argument expressions,
rather than a simple
#py("map") as in the ordinary evaluator.

#snippet(```python
def analyze_application(component):
    ffun = analyze(function_expression(component))
    afuns = map(analyze, arg_expressions(component))
    return lambda env, succeed, fail: (ffun(env, lambda fun, fail2: (get_args(afuns, env, lambda args, fail3: (execute_application(fun, args, succeed, fail3)), fail2)), fail))
```)

#idx("analyze...", sub: "nondeterministic")

In #py("get_args"), notice how walking down the list of #py("afun") execution functions and constructing the resulting list of #py("args") is accomplished by calling each #py("afun") in the list with a success continuation that recursively calls #py("get_args").
Each of these recursive calls to
#py("get_args")
has a success continuation whose value is the
new list resulting from using #py("pair") to adjoin the newly obtained argument to the list of accumulated arguments:

#syntax("
def get_args(afuns, env, succeed, fail):
    return succeed(None, fail) if is_none(afuns) else head(afuns)(env, lambda arg, fail2: (get_args(tail(afuns), env, lambda args, fail3: (succeed(pair(arg, args), fail3)), fail2)), fail)
")

The actual
function
application, which is performed by
#py("execute_application"),
is accomplished in the same way as for the ordinary evaluator, except for
the need to manage the continuations.
#idx("executeapplication", sub: "nondeterministic", decl: true)
#snippet(```python
def execute_application(fun, args, succeed, fail):
    return succeed(apply_primitive_function(fun, args), fail) if is_primitive_function(fun) else function_body(fun)(extend_environment(function_parameters(fun), args, function_environment(fun)), lambda body_result, fail2: (succeed(return_value_content(body_result) if is_return_value(body_result) else None, fail2)), fail) if is_compound_function(fun) else error("unknown function type - execute_application", fun)
```)

#subheading([Evaluating #py("amb") expressions])

The #py("amb")
syntactic
form is the key element in the nondeterministic language. Here we see the
essence of the interpretation process and the reason for keeping track of
the continuations. The execution
function
for #py("amb") defines a loop
#py("try_next")
that cycles through the execution
functions
for all the possible values of the #py("amb")
expression. Each execution
function
is called with a
#idx("failure continuation (nondeterministic evaluator)", sub: "constructed by amb")
failure continuation that will try the next one. When
there are no more alternatives to try, the entire
#py("amb") expression fails.
#idx("analyzeamb", decl: true)
#snippet(```python
def analyze_amb(component):
    cfuns = map(analyze, amb_choices(component))
    def _lambda_1(env, succeed, fail):
        def try_next(choices):
            return fail() if is_none(choices) else head(choices)(env, succeed, lambda : (try_next(tail(choices))))
        return try_next(cfuns)
    return _lambda_1
```)

#subheading([Driver loop])

#idx("driver loop", sub: "in nondeterministic evaluator")

The driver loop for the #py("amb") evaluator is
complex, due to the mechanism that permits the user to retry in evaluating
a program.
The driver uses a
function
called
#py("internal_loop"),
which takes as argument a
function
#idx("failure continuation (nondeterministic evaluator)", sub: "constructed by driver loop")
#py("retry").
The intent is that calling
#py("retry")
should go on to the next untried alternative in the nondeterministic
evaluation.
The function #py("internal_loop")
either calls
#py("retry")
in response to the user typing
#py("retry")
at the driver loop, or else starts a new evaluation by calling
#py("ambeval").

The failure continuation for this call to
#py("ambeval")

informs the user that there are no more values and reinvokes the driver
loop.

The success continuation for the call to #py("ambeval")

is more subtle. We print the obtained value and then
reinvoke the internal loop
with a
#py("retry")
function
that will be able to try the next alternative. This
#py("next_alternative")
function
is the second argument that was passed to the success continuation.
Ordinarily, we think of this second argument as a failure continuation to
be used if the current evaluation branch later fails. In this case,
however, we have completed a successful evaluation, so we can invoke the
"failure" alternative branch in order to search for additional
successful evaluations.

#idx("prompts", sub: "nondeterministic evaluator")#idx("driverloop", sub: "for nondeterministic evaluator", decl: true)
#snippet(```python
input_prompt = "amb-evaluate input:"
output_prompt =  "amb-evaluate value:"

def driver_loop(env):
    def internal_loop(retry):
        input = user_read(input_prompt)
        if is_none(input):
            print("evaluator terminated")
        elif input == "retry":
            return retry()
        else:
            print("Starting a new problem")
            program = parse(input)
            locals = scan_out_declarations(program)
            unassigneds = list_of_unassigned(locals)
            program_env = extend_environment(
                              locals, unassigneds, env)
            def on_success(val, next_alternative):
                user_print(output_prompt, val)
                return internal_loop(next_alternative)
            def on_failure():
                print("There are no more values of")
                print(input)
                return driver_loop(program_env)
            return ambeval(program, program_env, on_success, on_failure)
    def no_problem():
        print("There is no current problem")
        return driver_loop(env)
    return internal_loop(no_problem)
```)

The initial call to
#py("internal_loop")
uses a
#py("retry")
function
that complains that there is no current problem and restarts the driver loop.
This is the behavior that will happen if the user types
#py("retry")
when there is no evaluation in progress.

We start the driver loop as usual, by setting up the global environment
and passing it as the enclosing environment for the first iteration of
#py("driver_loop").

#snippet(```python
the_global_environment = setup_environment()
driver_loop(the_global_environment)
```)

#exercise(label-name: <ex:ramb>, [
Implement a new
syntactic
form #py("ramb") that is like
#py("amb") except that it searches alternatives in a
random order, rather than from left to right. Show how this can help with
Alyssa's problem in exercise @ex:sentence-generate.
])

#exercise(label-name: <ex:permanent-set>, [
Change the implementation of assignment so that it
is not undone upon failure. For example, we can choose two distinct
elements from a list and count the number of trials required to make a
successful choice as follows:

#snippet(```python
count = 0
x = an_element_of("a", "b", "c")
y = an_element_of("a", "b", "c")
count = count + 1
require(x != y)
llist(x, y, count)
```)

#output(```python
count = 0
x = an_element_of("a", "b", "c")
y = an_element_of("a", "b", "c")
count = count + 1
require(x != y)
llist(x, y, count)
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
retry
```)

#output(```python
retry
```)

What values would have been displayed if we had used
the original meaning of assignment rather than
permanent assignment?
])

#exercise(label-name: <ex:if-fail>, [
We shall horribly abuse the syntax for conditional statements, by
implementing a construct of the following form:

#syntax("
if (evaluation_succeeds_take) { ", meta("statement"), " } else { ", meta("alternative"), " }
	  ")

The construct permits the user to catch the failure of a
statement. It evaluates the statement as usual and returns
as usual if the evaluation succeeds. If the evaluation fails,
however, the given alternative statement
is evaluated, as in the following example:

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
if evaluation_succeeds_take:
    x = an_element_of(llist(1, 3, 5))
    require(is_even(x))
    x
else:
    "all odd"
```)

#output(```python
if evaluation_succeeds_take:
    x = an_element_of(llist(1, 3, 5))
    require(is_even(x))
    x
else:
    "all odd"
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
if evaluation_succeeds_take:
    x = an_element_of(llist(1, 3, 5, 8))
    require(is_even(x))
    x
else:
    "all odd"
```)

#output(```python
if evaluation_succeeds_take:
    x = an_element_of(llist(1, 3, 5, 8))
    require(is_even(x))
    x
else:
    "all odd"
```)

Implement this construct by extending the
#py("amb")
evaluator. Hint: The function
#py("is_amb") shows
how to abuse the existing Python syntax in order to implement
a new syntactic form.
])

#exercise(label-name: <ex:combine_permanent_if_fail>, [
With
the new kind of assignment
as described in exercise @ex:permanent-set and
the construct #syntax(" if (evaluation_succeeds_take) { ", $dots.h$, " } else { ", $dots.h$, " } ")
as in exercise @ex:if-fail, what will be the result of
evaluating

#snippet(```python
pairs = None
if evaluation_succeeds_take:
    p = prime_sum_pair(llist(1, 3, 5, 8), llist(20, 35, 110))
    pairs = pair(p, pairs)
    amb()
else:
    pairs
```)
])

#exercise(label-name: <ex:require_special>, [
If we had not realized that
#idx("require", sub: "as a syntactic form")
#py("require") could be
implemented as an ordinary
function
that uses #py("amb"), to be defined by the user as
part of a nondeterministic program, we would have had to implement it
as a
syntactic
form. This would require syntax
functions

#snippet(```python
def is_require(component):
    return is_tagged_list(component, "require")
def require_predicate(component):
    return head(tail(component))
```)

and a new clause in the dispatch in #py("analyze")

#snippet(```python
: is_require(component)
? analyze_require(component)
```)

as well the
function
#py("analyze_require")
that handles #py("require")
expressions. Complete the following definition of
#py("analyze_require").

#syntax("
def analyze_require(component):
    pfun = analyze(require_predicate(component))
    return lambda env, succeed, fail: (pfun(env, lambda pred_value, fail2: (", metaphrase[??], " if ", metaphrase[??], " else succeed(\"ok\", fail2)), fail))
      ")
])

#idx("nondeterministic evaluator")
