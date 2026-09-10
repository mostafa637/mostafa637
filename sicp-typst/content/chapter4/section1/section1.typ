// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([The Metacircular Evaluator], label-name: <sec:mc-eval>)

#idx("metacircular evaluator for Python")

Our evaluator for
Python
will be implemented as a
Python
program. It may
seem circular to think about evaluating
Python
programs using an evaluator that is itself implemented in
Python.
However, evaluation is a process, so it is appropriate to describe the
evaluation process using
Python,
which, after all, is our tool for describing processes.#footnote[Even so,
there will remain important aspects of the evaluation process that are not
elucidated by our evaluator. The most important of these are the detailed
mechanisms by which
functions
call other
functions
and return values to their callers. We will address these issues in
chapter @chap:reg, where we take a closer look at the evaluation process by
implementing the evaluator as a simple register machine.]
An evaluator that is written in the same language
that it evaluates is said to be
#idx("metacircular evaluator")
#idx("evaluator", sub: "metacircular")
#emph[metacircular].

The metacircular evaluator is essentially a
Python
formulation of the
#idx("environment model of evaluation", sub: "metacircular evaluator and")
#idx("metacircular evaluator for Python", sub: "environment model of evaluation in")
environment model of evaluation described in
section @sec:environment-model.
Recall that the model specifies the evaluation of function application in two basic steps:

+ To evaluate a function application, evaluate the subexpressions and then apply the value of the function subexpression to the values of the argument subexpressions.
+ To apply a compound function to a set of arguments, evaluate the body of the function in a new environment. To construct this environment, extend the environment part of the function object by a frame in which the parameters of the function are bound to the arguments to which the function is applied.

These two rules describe the essence of the evaluation process, a basic
#idx("metacircular evaluator for Python", sub: "evaluate–apply cycle")
cycle in which
statements and
expressions to be evaluated in environments are reduced to
functions
to be applied to arguments, which in turn are reduced to new
statements and
expressions to be evaluated in new environments, and so on, until we get
down to
names,
whose values are looked up in the environment, and to
operators and primitive functions,
which are applied directly (see
figure @fig:eval-apply).#footnote[If we grant ourselves
the ability to apply primitives,
then what remains for us to implement in the evaluator? The
#idx("metacircular evaluator for Python", sub: "job of")
job of the
evaluator is not to specify the primitives of the language, but rather to
provide the connective tissue—the means of combination and the means
of abstraction—that binds a collection of primitives to form a
language. Specifically:

- The evaluator enables us to deal with nested expressions. For example, although simply applying primitives would suffice for evaluating the expression #py("2 * 6"), it is not adequate for handling #py("2 * (1 + 5)"). As far as the operator #py("*") is concerned, its arguments must be numbers, and it would choke if we passed it the expression #py("1 + 5") as an argument. One important role of the evaluator is to choreograph composition so that #py("1 + 5") is reduced to 6 before being passed as an argument to #py("*").
- The evaluator allows us to use names. For example, the addition operator has no way to deal with expressions such as #py("x + 1"). We need an evaluator to keep track of names and obtain their values before invoking the operators.
- The evaluator allows us to define compound functions. This involves knowing how to use these functions in evaluating expressions and providing a mechanism that enables functions to accept arguments.
- The evaluator provides the other syntactic forms of the language such as conditionals and blocks.]
This evaluation cycle will be embodied by the interplay between the two
critical
functions
in the evaluator,
#py("evaluate")
and #py("apply"), which are described in
section @sec:core-of-evaluator
(see figure @fig:eval-apply).

The implementation of the evaluator will depend upon functions that define the #emph[syntax] of the statements and expressions to be evaluated. We will use #idx("metacircular evaluator for Python", sub: "data abstraction in") data abstraction to make the evaluator independent of the representation of the language. For example, rather than committing to a choice that an assignment is to be represented by a string beginning with a name followed by #py("="), we use an abstract predicate #py("is_assignment") to test for an assignment, and we use abstract selectors #py("assignment_symbol") and #py("assignment_value_expression") to access the parts of an assignment. The data abstraction layers presented in section @sec:representing-expressions will allow the evaluator to remain independent of concrete syntactic issues, such as the keywords of the interpreted language, and of the choice of data structures that represent the program components.
There are also
operations, described in
section @sec:eval-data-structures, that specify the
representation of
functions
and environments. For example,
#py("make_function")
constructs compound
functions,
#py("lookup_symbol_value")
accesses the values of
names,
and
#py("apply_primitive_function")
applies a primitive
function
to a given list of arguments.

#include "../../chapter4/section1/subsection1.typ"

#include "../../chapter4/section1/subsection2.typ"

#include "../../chapter4/section1/subsection3.typ"

#include "../../chapter4/section1/subsection4.typ"

#include "../../chapter4/section1/subsection5.typ"

#include "../../chapter4/section1/subsection6.typ"

#include "../../chapter4/section1/subsection7.typ"
