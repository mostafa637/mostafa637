// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Separating Syntactic Analysis from Execution], label-name: <sec:separating-analysis>)

#idx("syntactic analysis, separated from execution", sub: "in metacircular evaluator")
#idx("analyzing evaluator")
#idx("metacircular evaluator for Python", sub: "analyzing version")

The evaluator implemented above is simple, but it is very
#idx("metacircular evaluator for Python", sub: "efficiency of")
#idx("efficiency", sub: "of evaluation")
inefficient, because the syntactic analysis of
components
is interleaved
with their execution. Thus if a program is executed many times, its
syntax is analyzed many times. Consider, for example, evaluating
#py("factorial(4)") using the following definition of
#py("factorial"):

#snippet(```python
def factorial(n):
    return 1 if n == 1 else factorial(n - 1) * n
```)

Each time #py("factorial") is called, the evaluator
must determine that the body is
a conditional
expression and extract the predicate. Only then can it evaluate the
predicate and dispatch on its value. Each time it evaluates the expression
#py("factorial(n - 1) * n"),
or the subexpressions
#py("factorial(n - 1)")
and
#py("n - 1"),
the evaluator must perform the case analysis in
#py("evaluate")
to determine that the expression is an application, and must extract
its function expression and argument expressions.
This analysis is expensive.
Performing it repeatedly is wasteful.

We can transform the evaluator to be significantly more efficient by
arranging things so that syntactic analysis is performed only
once.#footnote[This technique is an integral part of the compilation
process, which we shall discuss in chapter @chap:reg. Jonathan Rees wrote
a Scheme
interpreter like this in about 1982 for the T project
#idx("Rees, Jonathan A.")
#idx("Adams, Norman I., IV")
(Rees and Adams 1982).
#idx("Feeley, Marc")
Marc Feeley 1986
(see also
#idx("Lapalme, Guy")
Feeley and Lapalme 1987)
independently invented this technique
in his master's thesis.] We split
#py("evaluate"),
which takes
a component
and an environment, into two parts. The
function
#py("analyze") takes only the
component.
It performs the syntactic
analysis and returns a new
function, the
#idx("execution function", sub: "in analyzing evaluator")
#emph[execution function], that
encapsulates the work to be done in executing the analyzed
component.
The execution
function
takes an environment as its
argument and completes the evaluation. This saves work because
#py("analyze") will be called only once on
a component,
while the execution
function
may be called many times.

With the separation into analysis and execution,
#py("evaluate")
now becomes

#idx("evaluate (metacircular)", sub: "analyzing version", decl: true)
#snippet(```python
def evaluate(component, env):
    return analyze(component)(env)
```)

The result of calling #py("analyze") is the execution
function
to be applied to the environment. The #py("analyze")
function
is the same case analysis as performed by the original
#py("evaluate")
of section @sec:core-of-evaluator, except that the
functions
to which we dispatch perform only analysis, not full evaluation:

#idx("analyze", sub: "metacircular", decl: true)
#snippet(```python
def analyze(component):
    return analyze_literal(component) if is_literal(component) else analyze_name(component) if is_name(component) else analyze_application(component) if is_application(component) else analyze(operator_combination_to_application(component)) if is_operator_combination(component) else analyze_conditional(component) if is_conditional(component) else analyze_lambda_expression(component) if is_lambda_expression(component) else analyze_sequence(sequence_statements(component)) if is_sequence(component) else analyze_block(component) if is_block(component) else analyze_return_statement(component) if is_return_statement(component) else analyze(function_decl_to_constant_decl(component)) if is_function_definition(component) else analyze_declaration(component) if is_declaration(component) else analyze_assignment(component) if is_assignment(component) else error("unknown syntax -- analyze", component)
```)

#idx("analyze...", sub: "metacircular")

Here is the simplest syntactic analysis
function, which handles literal expressions.
It returns an execution
function
that ignores its environment argument and just returns the
value of the literal:

#snippet(```python
def analyze_literal(component):
    return lambda env: (literal_value(component))
```)

Looking up
the value of a name
must still be done in the execution phase, since this depends upon knowing
the environment.#footnote[There is, however, an important part of the
search for a name
that #emph[can] be done as part of the syntactic analysis.
As we will show in section @sec:lexical-addressing,
one can determine the position in the environment structure where the
value of the variable will be found, thus obviating the need to scan the
environment for the entry that matches the variable.]

#snippet(```python
def analyze_name(component):
    return lambda env: (lookup_symbol_value(symbol_of_name(component), env))
```)

To analyze an application, we analyze the
function expression and argument expressions
and construct an execution function that calls the
execution function of the function expression
(to obtain the actual function to be applied) and the
argument-expression execution functions (to obtain the actual arguments).
We then pass these to
#py("execute_application"),
which is the analog of #py("apply") in
section @sec:core-of-evaluator.
The function #py("execute_application")
differs from #py("apply") in that the
function body for a compound function
has already been analyzed, so there is no need to do further analysis.
Instead, we just call the execution function
for the body on the extended environment.

#idx("executeapplication", sub: "metacircular", decl: true)
#snippet(```python
def analyze_application(component):
    ffun = analyze(function_expression(component))
    afuns = map(analyze, arg_expressions(component))
    return lambda env: (execute_application(ffun(env), map(lambda afun: (afun(env)), afuns)))
def execute_application(fun, args):
    if is_primitive_function(fun):
        return apply_primitive_function(fun, args)
    elif is_compound_function(fun):
        result = function_body(fun) (extend_environment(function_parameters(fun), args, function_environment(fun)))
        return return_value_content(result) if is_return_value(result) else None
    else:
        error("unknown function type -- execute_application", fun)
```)

For conditionals,
we extract and analyze the predicate, consequent, and alternative at
analysis time.

#snippet(```python
def analyze_conditional(component):
    pfun = analyze(conditional_predicate(component))
    cfun = analyze(conditional_consequent(component))
    afun = analyze(conditional_alternative(component))
    return lambda env: (cfun(env) if is_truthy(pfun(env)) else afun(env))
```)

Analyzing a
lambda
expression also achieves a major gain in efficiency: We analyze the
lambda body only once, even though
functions resulting from evaluation of the
lambda expression
may be applied many times.

#snippet(```python
def analyze_lambda_expression(component):
    params = lambda_parameter_symbols(component)
    bfun = analyze(lambda_body(component))
    return lambda env: (make_function(params, bfun, env))
```)

Analysis of a sequence of
statements is more
involved.#footnote[See exercise @ex:analyze-sequence for
some insight into the processing of sequences.] Each
statement
in the sequence is analyzed, yielding an execution
function. These execution functions
are combined to produce an execution
function that takes an environment as argument and sequentially
calls each individual execution
function with the environment as argument.

#snippet(```python
def analyze_sequence(stmts):
    def sequentially(fun1, fun2):
        def the_sequence(env):
            fun1_val = fun1(env)
            return (fun1_val
                    if is_return_value(fun1_val)
                    else fun2(env))
        return the_sequence
    def loop(first_fun, rest_funs):
        return (first_fun
                if is_none(rest_funs)
                else loop(sequentially(first_fun, head(rest_funs)),
                          tail(rest_funs)))
    funs = map(analyze, stmts)
    return ((lambda env: None)
            if is_none(funs)
            else loop(head(funs), tail(funs)))
```)

The body of a
block is scanned only once for local declarations.
The bindings are installed in the environment when
the execution function for the block is called.

#snippet(```python
def analyze_block(component):
    body = block_body(component)
    bfun = analyze(body)
    locals = scan_out_declarations(body)
    unassigneds = list_of_unassigned(locals)
    return lambda env: (bfun(extend_environment(locals, unassigneds, env)))
```)

For return statements, we analyze the return expression.
The execution function for the return statement simply calls
the execution function for the return expression and wraps
the result in a return value.

#snippet(```python
def analyze_return_statement(component):
    rfun = analyze(return_expression(component))
    return lambda env: (make_return_value(rfun(env)))
```)

The function #py("analyze_assignment")
must defer actually setting the variable until the execution, when the
environment has been supplied. However, the fact that the
assignment-value expression
can be analyzed (recursively) during analysis is a major gain in
efficiency, because the
assignment-value expression
will now be analyzed only once. The same holds true for
constant and variable declarations.

#snippet(```python
def analyze_assignment(component):
    symbol = assignment_symbol(component)
    vfun = analyze(assignment_value_expression(component))
    def the_assignment(env):
        value = vfun(env)
        assign_symbol_value(symbol, value, env)
        return value
    return the_assignment

def analyze_declaration(component):
    symbol = declaration_symbol(component)
    vfun = analyze(declaration_value_expression(component))
    def the_declaration(env):
        assign_symbol_value(symbol, vfun(env), env)
        return None
    return the_declaration
```)

Our new evaluator uses the same data structures, syntax
functions,
and
runtime support functions
as in sections @sec:representing-expressions,
 @sec:eval-data-structures,
and @sec:running-eval.

#idx("analyze...", sub: "metacircular")

#exercise([
Extend the evaluator in this section to support
#idx("while loop", sub: "implementing in analyzing evaluator")
while loops.
(See exercise @ex:while_loop.)
])

#exercise(label-name: <ex:analyze-sequence>, [
Alyssa P. Hacker doesn't understand why
#idx("analyze...", sub: "metacircular")
#py("analyze_sequence")
needs to be so complicated. All the other analysis
functions
are straightforward transformations of the corresponding evaluation
functions
(or
#py("evaluate")
clauses) in
section @sec:core-of-evaluator.
She expected
#py("analyze_sequence")
to look like this:

#snippet(```python
def analyze_sequence(stmts):
    def execute_sequence(funs, env):
        if is_none(funs):
            return None
        elif is_none(tail(funs)):
            return head(funs)(env)
        else:
            head_val = head(funs)(env)
            return head_val if is_return_value(head_val) else execute_sequence(tail(funs), env)
    funs = map(analyze, stmts)
    return lambda env: (execute_sequence(funs, env))
```)

Eva Lu Ator explains to Alyssa that the version in the text does more of the
work of evaluating a sequence at analysis time. Alyssa's
sequence-execution function,
rather than having the calls to the individual execution
functions
built in, loops through the
functions
in order to call them: In effect, although the individual
statements
in the sequence have been analyzed, the sequence itself has not been.

Compare the two versions of
#py("analyze_sequence").
For example, consider the common case (typical of
function
bodies) where the sequence has just one
statement.
What work will the
execution
function
produced by Alyssa's program do? What about the execution
function
produced by the program in the text above? How do the two versions compare
for a sequence with two expressions?
])

#exercise(label-name: <ex:meta_speed>, [
Design and carry out some experiments to compare the speed of the original
metacircular evaluator with the version in this section. Use your results
to estimate the fraction of time that is spent in analysis versus execution
for various
functions.
])

#idx("syntactic analysis, separated from execution", sub: "in metacircular evaluator")
#idx("analyzing evaluator")
#idx("metacircular evaluator for Python", sub: "analyzing version")
