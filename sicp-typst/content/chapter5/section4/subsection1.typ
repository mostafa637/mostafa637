// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([The Dispatcher and Basic Evaluation], label-name: <sec:eceval-core>)

#idx("explicit-control evaluator for Python", sub: "controller")

The central element in the evaluator is the sequence of instructions beginning at
#py("eval_dispatch").
This corresponds to the
#py("evaluate")
function
of the metacircular evaluator described in section @sec:core-of-evaluator. When the controller starts at
#py("eval_dispatch"),
it evaluates the
component
specified by
#py("comp")
in the environment specified by #py("env"). When evaluation is
complete, the controller will go to the entry point stored in
#py("continue"), and the #py("val")
register will hold the value of the
component.
As with the metacircular
#py("evaluate"),
the structure of
#py("eval_dispatch")
is a case analysis on the syntactic type of the
component
to be evaluated.#footnote[In our controller, the dispatch is written as a sequence of
#py("test") and #py("branch")
instructions. Alternatively, it could have been written in a data-directed
style, which avoids
the need to perform sequential tests and
facilitates
the definition of new
component
types.]
#idx("evaldispatch", decl: true)
#snippet(```python
"eval_dispatch",
  test(list(op("is_literal"), reg("comp"))),
  branch(label("ev_literal")),
  test(list(op("is_name"), reg("comp"))),
  branch(label("ev_name")),
  test(list(op("is_application"), reg("comp"))),
  branch(label("ev_application")),
  test(list(op("is_operator_combination"), reg("comp"))),
  branch(label("ev_operator_combination")),
  test(list(op("is_conditional"), reg("comp"))),
  branch(label("ev_conditional")),
  test(list(op("is_lambda_expression"), reg("comp"))),
  branch(label("ev_lambda")),
  test(list(op("is_sequence"), reg("comp"))),
  branch(label("ev_sequence")),
  test(list(op("is_block"), reg("comp"))),
  branch(label("ev_block")),
  test(list(op("is_return_statement"), reg("comp"))),
  branch(label("ev_return")),
  test(list(op("is_function_definition"), reg("comp"))),
  branch(label("ev_function_definition")),
  test(list(op("is_declaration"), reg("comp"))),
  branch(label("ev_declaration")),
  test(list(op("is_assignment"), reg("comp"))),
  branch(label("ev_assignment")),
  go_to(label("unknown_component_type")),
```)

#subheading([Evaluating simple expressions])

#idx("explicit-control evaluator for Python", sub: "expressions with no subexpressions to evaluate")

Numbers and strings,
names,
and
lambda
expressions have no subexpressions to be evaluated. For these, the evaluator simply
places the correct value in the #py("val") register and
continues execution at the entry point specified by
#py("continue"). Evaluation of simple expressions is performed
by the following controller code:
#idx("evliteral", decl: true)#idx("evname", decl: true)#idx("evlambda", decl: true)
#snippet(```python
"ev_literal",
  assign("val", list(op("literal_value"), reg("comp"))),
  go_to(reg("continue")),

"ev_name",
  assign("val", list(op("symbol_of_name"), reg("comp"), reg("env"))),
  assign("val", list(op("lookup_symbol_value"),
                     reg("val"), reg("env"))),
  go_to(reg("continue")),

"ev_lambda",
  assign("unev", list(op("lambda_parameter_symbols"), reg("comp"))),
  assign("comp", list(op("lambda_body"), reg("comp"))),
  assign("val", list(op("make_function"),
                     reg("unev"), reg("comp"), reg("env"))),
  go_to(reg("continue")),
```)

Observe how
#py("ev_lambda")
uses the
#py("unev")
and
#py("comp")
registers to hold the parameters and body of the lambda expression so
that they can be passed to the
#py("make_function")
operation, along with the environment in #py("env").

#idx("explicit-control evaluator for Python", sub: "expressions with no subexpressions to evaluate")

#subheading([Conditionals])

As with the metacircular evaluator, syntactic forms are handled by selectively
evaluating fragments of the component. For a
#idx("explicit-control evaluator for Python", sub: "conditionals")
conditional, we must evaluate the
predicate and decide, based on the value of predicate, whether to evaluate the
consequent or the alternative.

Before evaluating the predicate, we save the conditional itself, which is in
#py("comp"), so that we can later extract the
consequent or alternative. To evaluate the predicate expression, we move it to
the #py("comp") register and go to
#py("eval_dispatch"). The environment in the
#py("env") register is already the correct one in
which to evaluate the predicate. However, we save
#py("env") because we will need it later to
evaluate the consequent or the alternative. We set up
#py("continue") so that evaluation will resume at
#py("ev_conditional_decide") after the predicate
has been evaluated. First, however, we save the old value of
#py("continue"), which we will need later in order
to return to the evaluation of the statement that is waiting for the value of
the conditional.
#idx("evconditional", decl: true)
#snippet(```python
"ev_conditional",
  save("comp"), // save conditional for later
  save("env"),
  save("continue"),
  assign("continue", label("ev_conditional_decide")),
  assign("comp", list(op("conditional_predicate"), reg("comp"))),
  go_to(label("eval_dispatch")), // evaluate the predicate
```)

When we resume at #py("ev_conditional_decide") after
evaluating the predicate, we test whether it was true or false
and, depending on the result, place either the consequent or the alternative in
#py("comp") before going to
#py("eval_dispatch").#footnote[In this chapter, we will use the function
#idx("isfalsy", sub: "why used in explicit-control evaluator", decl: true)
#py("is_falsy") to test the value of the predicate.
This allows us to write the consequent and alternative branches in the same
order as in a conditional, and simply fall through to the consequent
branch when the predicate holds. The function #py("is_falsy")
is declared as the opposite of the
#py("is_truthy") function used to test predicates of
conditionals in section @sec:core-of-evaluator.]
Notice that restoring
#py("env") and
#py("continue") here sets up
#py("eval_dispatch") to have the correct environment
and to continue at the right place to receive the value of the conditional.

#snippet(```python
"ev_conditional_decide",
  restore("continue"),
  restore("env"),
  restore("comp"),
  test(list(op("is_falsy"), reg("val"))),
  branch(label("ev_conditional_alternative")),
"ev_conditional_consequent",
  assign("comp", list(op("conditional_consequent"), reg("comp"))),
  go_to(label("eval_dispatch")),
"ev_conditional_alternative",
  assign("comp", list(op("conditional_alternative"), reg("comp"))),
  go_to(label("eval_dispatch")),
```)

#subheading([Sequence Evaluation])

#idx("explicit-control evaluator for Python", sub: "sequences of statements")

The portion of the explicit-control evaluator beginning at
#py("ev_sequence"), which handles
sequences of statements, is analogous to the metacircular evaluator's
#py("eval_sequence") function.

The entries at #py("ev_sequence_next") and
#py("ev_sequence_continue") form a loop that
successively evaluates each statement in a sequence.
The list of unevaluated
statements is kept in #py("unev").
At #py("ev_sequence") we place the sequence of
statements to be evaluated in #py("unev"). If the
sequence is empty, we set #py("val") to #py("undefined") and jump to
#py("continue") via
#py("ev_sequence_empty"). Otherwise we start the
sequence-evaluation loop, first saving the value of #py("continue") on the stack, because
the #py("continue") register will be used for local flow of control in the loop, and the original
value is needed for continuing after the statement sequence. Before evaluating
each statement, we check to see if there are additional statements to be evaluated
in the sequence. If so, we save the rest of the unevaluated statements (held in
#py("unev")) and the environment in which these must
be evaluated (held in #py("env")) and call
#py("eval_dispatch") to evaluate the statement,
which has been placed in #py("comp"). The
two saved registers are restored after this evaluation, at
#py("ev_sequence_continue").

The final statement in the sequence is handled differently, at the entry point
#py("ev_sequence_last_statement"). Since there are
no more statements to be evaluated after this one, we need not save
#py("unev") or
#py("env") before going to
#py("eval_dispatch"). The value of the whole
sequence is the value of the last statement, so after the evaluation of the last
statement there is nothing left to do except continue at the entry point that was
saved at #py("ev_sequence").
Rather than setting up #py("continue") to arrange
for #py("eval_dispatch") to return here and then
restoring #py("continue") from the stack and
continuing at that entry point, we restore
#py("continue") from the stack before going to
#py("eval_dispatch"), so that
#py("eval_dispatch") will continue at that entry
point after evaluating the statement.

#idx("evsequence", decl: true)
#snippet(```python
"ev_sequence",
  assign("unev", list(op("sequence_statements"), reg("comp"))),
  test(list(op("is_empty_sequence"), reg("unev"))),
  branch(label("ev_sequence_empty")),
  save("continue"),
"ev_sequence_next",
  assign("comp", list(op("first_statement"), reg("unev"))),
  test(list(op("is_last_statement"), reg("unev"))),
  branch(label("ev_sequence_last_statement")),
  save("unev"),
  save("env"),
  assign("continue", label("ev_sequence_continue")),
  go_to(label("eval_dispatch")),
"ev_sequence_continue",
  restore("env"),
  restore("unev"),
  assign("unev", list(op("rest_statements"), reg("unev"))),
  go_to(label("ev_sequence_next")),
"ev_sequence_last_statement",
  restore("continue"),
  go_to(label("eval_dispatch")),

"ev_sequence_empty",
  assign("val", constant(undefined)),
  go_to(reg("continue")),
```)

Unlike #py("eval_sequence") in the metacircular
evaluator, #py("ev_sequence") does not need to check whether a return statement was
evaluated so as to terminate the sequence evaluation. The "explicit control" in this evaluator allows a return statement to jump directly to
the continuation of the current function application without resuming the
sequence evaluation. Thus sequence evaluation does not need to be concerned
with returns, or even be aware of the existence of return statements in the
language. Because a return statement jumps out of the sequence-evaluation code,
the restores of saved registers at #py("ev_sequence_continue")
won't be executed. We will see later how the return statement removes these values from the stack.

#idx("explicit-control evaluator for Python", sub: "sequences of statements")
