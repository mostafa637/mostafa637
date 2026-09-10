// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([The Core of the Evaluator])

#anchor(<sec:core-of-evaluator>)

#sicp-figure(image("/images/img_javascript/ch4-Z-G-1.svg", width: 70%), caption: [The #idx("metacircular evaluator for Python", sub: "evaluate–apply cycle") #py("evaluate")–#py("apply") cycle exposes the essence of a computer language.], label-name: <fig:eval-apply>)

The evaluation
#idx("metacircular evaluator for Python", sub: "evaluate and apply")
process can be described as the interplay between two
functions:
#py("evaluate")
and #py("apply").

#subheading([The function #py("evaluate")])

The function #py("evaluate")
takes as arguments
a program #emph[component]—a statement or expression#footnote[There is no need to distinguish between statements and expressions in our evaluator. For example, we do not differentiate between expressions and expression statements; we represent them identically and consequently they are handled in the same way by the #py("evaluate") function. Similarly, our evaluator does not enforce Python's syntactic restriction that statements cannot appear inside expressions other than lambda expressions.]—and an environment.
It classifies the
component
and directs its evaluation.
The function #py("evaluate")
is structured as a case analysis of the syntactic type of the
component
to be evaluated. In order to keep the
function
general, we express
the determination of the type of
a component
abstractly, making no
commitment to any particular
#idx("metacircular evaluator for Python", sub: "component representation")
representation for the various types of
components.
Each type of
component
has a
#emph[syntax predicate]
that tests for it and an abstract means for selecting its parts. This
#idx("metacircular evaluator for Python", sub: "data abstraction in")
#idx("abstract syntax", sub: "in metacircular evaluator")
#emph[abstract syntax]
makes it easy to see how we can change the syntax of the language by
using the same evaluator, but with a different collection of syntax
functions.

#subsubheading([Primitive expressions])

- For #idx("expression", sub: "literal") #idx("literal expression") literal expressions, such as numbers, #py("evaluate") returns their value.
- The function #py("evaluate") must look up names in the environment to find their values.

#subsubheading([Combinations])

- For a function application, #py("evaluate") must recursively evaluate the function expression and the argument expressions of the application. The resulting function and arguments are passed to #py("apply"), which handles the actual function application.
- An operator combination is transformed into a function application and then evaluated.

#subsubheading([Syntactic forms])

- A conditional expression or statement requires special processing of its parts, so as to evaluate the consequent if the predicate is true, and otherwise to evaluate the alternative.
- A lambda expression must be transformed into an applicable function by packaging together the parameters and body specified by the lambda expression with the environment of the evaluation.
- A sequence of statements requires evaluating its components in the order in which they appear.
- A block requires evaluating its body in a new environment that reflects all names declared within the block.
- A return statement must produce a value that becomes the result of the function call that gave rise to the evaluation of the return statement.
- A function definition is transformed into a constant declaration and then evaluated.
- A constant or variable declaration or an assignment must call #py("evaluate") recursively to compute the new value to be associated with the name being declared or assigned. The environment must be modified to reflect the new value of the name.

Here is the declaration of
#py("evaluate"):

#idx("evaluate (metacircular)", decl: true)
#snippet(```python
def evaluate(component, env):
  if is_literal(component):
    return literal_value(component)
  elif is_name(component):
    return lookup_symbol_value(symbol_of_name(component), env)
  elif is_application(component):
    return apply(evaluate(function_expression(component), env),
                          llist_of_values(arg_expressions(component),
                                          env))
  elif is_operator_combination(component):
    return evaluate(operator_combination_to_application(component),
                    env)
  elif is_conditional(component):
    return eval_conditional(component, env)
  elif is_lambda_expression(component):
    return make_function(lambda_parameter_symbols(component),
                         make_return_statement(lambda_body(component)),
                         env)
  elif is_sequence(component):
    return eval_sequence(sequence_statements(component), env)
  elif is_return_statement(component):
    return eval_return_statement(component, env)
  elif is_function_definition(component):
    return evaluate(function_def_to_assignment(component), env)
  elif is_assignment(component):
    return eval_assignment(component, env)
  else:    return error("unknown syntax -- evaluate", component)
```)

For clarity,
#py("evaluate")
has been implemented as a
#idx("data-directed programming", sub: "case analysis vs.")
#idx("case analysis", sub: "data-directed programming vs.")
case analysis using
conditional statements.
The disadvantage of this is that our
function
handles only a few distinguishable types of
statements and
expressions, and no new ones can be defined without editing the
declaration of #py("evaluate").
In most
interpreter
implementations, dispatching on the type of
a component
is done in a data-directed style. This allows a user to add new types of
components that #py("evaluate")
can distinguish, without modifying the
declaration of #py("evaluate")
itself. (See exercise @ex:data-directed-eval.)

The representation of names is handled by the syntax abstractions. Internally,
the evaluator uses strings to represent names, and we refer to such strings as
#idx("symbol(s)", sub: "representing names in metacircular evaluator")
#emph[symbols]. The function
#py("symbol_of_name") used in
#py("evaluate") extracts from a
name the symbol by which it is represented.

#subheading([Apply])

The function #py("apply")
takes two arguments, a
function
and a linked list of arguments to which the
function
should be applied.
The function #py("apply")
classifies
functions
into two kinds: It calls
#idx("applyprimitivefunction")
#py("apply_primitive_function")
to apply primitives; it applies compound
functions
by evaluating the body of the function.
The environment for the evaluation of the body of a compound
function
is constructed by extending the base environment carried by the
function
to include a frame that binds the parameters of the
function
to the arguments to which the
function
is to be applied.
Here is the
declaration
of #py("apply"):
#idx("apply (metacircular)", decl: true)
#snippet(```python
def apply(fun, arguments):
    if is_primitive_function(fun):
        return apply_primitive_function(fun, arguments)
    elif is_compound_function(fun):
        body = function_body(fun)
        locals = local_variables(body)
    unassigneds = llist_of_unassigned(locals)
        result = evaluate(body,
                          extend_environment(
                              append(function_parameters(fun),
                         locals),
                              append(arguments,
                         unassigneds),
                              function_environment(fun)))
        return (return_value_content(result)
                if is_return_value(result)
                else None)
    else:
        return error("unknown function type -- apply", fun)
```)

In order to return a value, a Python function needs to evaluate a
#idx("metacircular evaluator for Python", sub: "return value")
#idx("return value", sub: "representation in metacircular evaluator")
return statement. If a function terminates without evaluating a return
statement, the value
#idx("return value", sub: "None as")
#py("None") is returned.
To distinguish the two cases, the evaluation of a return statement
will wrap the result of evaluating its return expression into a
#emph[return value]. If
the evaluation of the function body yields such a return value, the content
of the return value is retrieved; otherwise the value
#py("None") is returned.#footnote[This test is a deferred operation, and thus our evaluator will give rise to a
recursive process even if the interpreted program should give rise to an
iterative process according to the description in
section @sec:recursion-and-iteration. In other
words, our metacircular evaluator implementation of Python is
#idx("apply (metacircular)", sub: "tail recursion and")
#idx("metacircular evaluator for Python", sub: "tail recursion and")
#idx("tail recursion", sub: "metacircular evaluator and")
not tail-recursive.
Sections @sec:tail-recursion-return
and @sec:compiling-combinations
show how to achieve tail recursion using a register machine.]<foot:apply>

The function #py("scan_out_declarations")
#idx("scanning out declarations", sub: "in metacircular evaluator")
collects a list of all symbols representing names declared in the body.
It uses
#py("declaration_symbol")
to retrieve the symbol that represents the name
from the declaration statements it finds.
#idx("scanoutdeclarations", decl: true)
#snippet(```python
def scan_out_declarations(component):
    if is_sequence(component):
        return reduce(append,
                      None,
                      map(scan_out_declarations,
                          sequence_statements(component)))
    elif is_declaration(component):
        return llist(declaration_symbol(component))
    else:
        return None
```)

We ignore declarations that are nested in function definitions,
because the evaluation of that function definition will take care of them.

#subheading([Function arguments])

When
#py("evaluate")
processes a
function
application, it uses
#py("llist_of_values")
to produce the list of arguments to which the
function
is to be applied.
The function #py("llist_of_values")
takes as an argument the
argument expressions of the application.
It evaluates each
argument expression
and returns a
list of the corresponding values:#footnote[We chose to implement #py("llist_of_values") using the #idx("metacircular evaluator for Python", sub: "higher-order functions in") #idx("higher-order functions", sub: "in metacircular evaluator") higher-order function #py("map"), and we will use common higher-order functions in other places as well. However, the evaluator can be implemented without any use of higher-order functions (and thus could be written in a language that doesn't have higher-order functions), even though the language that it supports will include higher-order functions. For example, #py("llist_of_values") can be written without #py("map") as follows: #idx("llistofvalues", sub: "without higher-order functions", decl: true) #snippet(```python def llist_of_values(exps, env): if is_null(exps): return None else: pair(evaluate(head(exps), env), llist_of_values(tail(exps), env))) ```)]<foot:mceval-higher-order>
#idx("llistofvalues", decl: true)
#snippet(```python
def llist_of_values(exps, env):
    return map(lambda arg: evaluate(arg, env), exps)
```)

#subheading([Conditionals])

The function #py("eval_conditional")
evaluates the predicate part of
a conditional component
in the given environment. If the result is true,
the consequent is evaluated, otherwise the alternative is evaluated:
#idx("evalconditional (metacircular)", decl: true)
#snippet(```python
def eval_conditional(component, env):
    if is_truthy(evaluate(conditional_predicate(component), env)):
        return (evaluate(conditional_consequent(component), env)
    else: return evaluate(conditional_alternative(component), env))
```)

Note that the evaluator does not need to distinguish between conditional expressions and conditional statements.

The use of
#idx("istruthy")
#idx("truthiness")
#py("is_truthy")
in
#py("eval_conditional")
#idx("metacircular evaluator for Python", sub: "implemented language vs. implementation language")
highlights the issue of the connection between an implemented language and
an implementation language. The
#py("conditional_predicate")
is evaluated in the language being implemented and thus yields a value in
that language. The interpreter predicate
#py("is_truthy")
translates that value into a value that can be tested by the
conditional expression
in the implementation language: The metacircular representation of truth
might not be the same as that of the underlying
Python.#footnote[In this case, the language being implemented and the
implementation language are the same. Contemplation of the meaning of
#py("is_truthy")
here yields
#idx("consciousness, expansion of")
expansion of consciousness without the abuse of
substance.]

#subheading([Sequences])

The function #py("eval_sequence")
is used by #py("evaluate")
to evaluate a sequence of statements at the top level, in a
function body, or in branch of a conditional statement.
It takes as arguments a sequence of statements and an
environment, and evaluates the statements in the order in which they
occur. The value returned is #py("None"),
except that if the evaluation of any statement in the sequence yields
a return value, that value is returned and the subsequent statements are
ignored.
#idx("evalsequence", decl: true)
#snippet(```python
def eval_sequence(stmts, env):
    for_each(lambda stmt: evaluate(stmt, env), stmts)
    return None
```)

#subheading([Return statements])

The function #py("eval_return_statement")
is used to evaluate
#idx("return statement", sub: "handling in metacircular evaluator")
return statements. As seen in
#py("apply") and
the evaluation
of sequences, the result of evaluation of a return statement
needs to be identifiable so that the evaluation of a function
body can return immediately, even if there are statements
after the return statement. For this purpose,
the evaluation of a return statement wraps the result of
evaluating the return expression in a return value object.#footnote[#idx("metacircular evaluator for Python", sub: "tail recursion and")
#idx("tail recursion", sub: "metacircular evaluator and")
The application of the function
#py("make_return_value") to the result
of evaluating the return expression creates a deferred operation, in
addition to the deferred operation created by
#py("apply"). See
footnote @foot:apply for details.]
#idx("evalreturnstatement", decl: true)
#snippet(```python
def eval_return_statement(component, env):
    return make_return_value(evaluate(return_expression(component),
                                      env))
```)

#subheading([Assignments and declarations])

The
function #py("eval_assignment")
handles assignments to
names. (To simplify the presentation of our evaluator, we are allowing assignment not just to variables but also—erroneously—to constants. Exercise @ex:mutable explains how we could distinguish constants from variables and prevent assignment to constants.)
The function #py("eval_assignment") calls #py("evaluate") on the value expression to find the value to be assigned and calls #py("assignment_symbol") to retrieve the symbol that represents the name from the assignment. The function #py("eval_assignment") transmits the symbol and the value to #py("assign_symbol_value") to be installed in the designated environment. The evaluation of an assignment returns the value that was assigned.
#idx("evalassignment", decl: true)
#snippet(```python
def eval_assignment(component, env):
    value = evaluate(assignment_value_expression(component), env)
    assign_symbol_value(assignment_symbol(component), value, env)
    return value
```)

Constant and variable declarations are both recognized by the
#py("is_declaration") syntax predicate.
They are treated in a manner similar to
assignments, because #py("eval_block")
has already bound their symbols to #py("\"*unassigned*\"")
in the current environment.
Their evaluation replaces #py("\"*unassigned*\"")
with the result of evaluating the value expression.
#idx("evaldeclaration", decl: true)
#snippet(```python
def eval_declaration(component, env):
    assign_symbol_value(
        declaration_symbol(component),
        evaluate(declaration_value_expression(component), env),
        env)
    return None
```)

The result of evaluating the body of a function is determined by
return statements, and therefore the return value
#py("undefined") in
#py("eval_declaration") only
matters when the declaration occurs at the top level,
outside of any function body. Here we use the return value
#py("undefined") to simplify
the presentation; exercise @ex:value_producing
describes the real result of evaluating top-level components
in Python.

#anchor(<foot:value_producing_2>)

#idx("metacircular evaluator for Python", sub: "evaluate and apply")

#exercise(label-name: <ex:arg-eval-order>, [
Notice that we cannot tell whether the metacircular evaluator
#idx("order of evaluation", sub: "in metacircular evaluator")
#idx("metacircular evaluator for Python", sub: "order of argument evaluation")
evaluates argument expressions from left to right or from right to left.
Its evaluation order is inherited from the underlying JavaScript:
If the arguments to #py("pair") in
#py("map") are evaluated from
left to	right, then #py("llist_of_values")
will evaluate argument expressions from left to right; and if the
arguments to #py("pair") are
evaluated from right to left, then
#py("llist_of_values") will evaluate
argument expressions from right to left.

Write a version of #py("llist_of_values")
that evaluates argument expressions from left to right regardless of the
order of evaluation in the underlying JavaScript. Also write a version of
#py("llist_of_values") that evaluates
argument expressions from right to left.
])
