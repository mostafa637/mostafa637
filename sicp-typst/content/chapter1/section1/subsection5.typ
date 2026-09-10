// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([The Substitution Model for Function Application], label-name: <sec:substitution-model>)

#idx("substitution model of function application")

To evaluate a function application, the interpreter follows the process
described in section @sec:compound-procedures.

That is, the interpreter evaluates the elements of the
application
and	applies the
function
(which is the value of the
function expression of the application)
to the arguments (which are the values of the
argument expressions of the application).

We can assume that the application of primitive functions is handled by the interpreter or libraries.
For compound
functions,
the application process is as follows:

- To apply a compound function to arguments, evaluate the return expression of the function with each parameter replaced by the corresponding argument.#footnote[If the body of the function is a sequence of statements, the body is evaluated with the parameters replaced, and the value of the application is the value of the return expression of the first return statement encountered.]

To illustrate this process, let's evaluate the
application

#snippet(```python
f(5)
```)

where #py("f") is the
function defined
in section @sec:compound-procedures.
We begin by retrieving the
return expression
of #py("f"):

#snippet(```python
sum_of_squares(a + 1, a * 2)
```)

Then we replace the parameter #py("a")
by the argument 5:

#snippet(```python
sum_of_squares(5 + 1, 5 * 2)
```)

Thus the problem reduces to the evaluation of
an application
with two
arguments
and
a function expression #py("sum_of_squares").
Evaluating this
application
involves three subproblems. We must evaluate the
function expression
to get the
function
to be applied, and we must evaluate the
argument expressions
to get the arguments. Now
#py("5 + 1")
produces 6 and
#py("5 * 2")
produces 10, so we must apply the
#py("sum_of_squares") function
to 6 and 10. These values are substituted for the

parameters #py("x") and
#py("y") in the body of
#py("sum_of_squares"),
reducing the expression to

#snippet(```python
square(6) + square(10)
```)

If we use the
definition
of #py("square"), this reduces to

#snippet(```python
(6 * 6) + (10 * 10)
```)

which reduces by multiplication to

#snippet(```python
36 + 100
```)

and finally to

#snippet(```python
136
```)

The process we have just described is called the #emph[substitution model] for
function
application. It can be taken as a model that
determines the "meaning" of
function
application, insofar as the
functions
in this chapter are concerned. However, there are two
points that should be stressed:

- The purpose of the substitution is to help us think about function application, not to provide a description of how the interpreter really works. Typical interpreters do not evaluate function applications by manipulating the text of a function to substitute values for the parameters. In practice, the "substitution" is accomplished by using a local environment for the parameters. We will discuss this more fully in chapters 3 and 4 when we examine the implementation of an interpreter in detail.
- Over the course of this book, we will present a sequence of increasingly elaborate models of how interpreters work, culminating with a complete implementation of an interpreter and compiler in chapter @chap:reg. The substitution model is only the first of these models—a way to get started thinking formally about the evaluation process. In general, when #idx("modeling", sub: "in science and engineering") modeling phenomena in science and engineering, we begin with simplified, incomplete models. As we examine things in greater detail, these simple models become inadequate and must be replaced by more refined models. The substitution model is no exception. In particular, when we address in chapter @chap:state the use of functions with "mutable data," we will see that the substitution model breaks down and must be replaced by a more complicated model of function application.#footnote[Despite the simplicity of the substitution idea, it turns out to be surprisingly complicated to give a rigorous mathematical definition of the substitution process. The problem arises from the possibility of confusion between the names used for the parameters of a function and the (possibly identical) names used in the expressions to which the function may be applied. Indeed, there is a long history of erroneous definitions of #emph[substitution] in the literature of logic and programming semantics. #idx("Stoy, Joseph E.") See Stoy 1977 for a careful discussion of substitution.]

#subheading([Applicative order versus normal order])

According to the description of evaluation given in
section @sec:compound-procedures,
the interpreter first evaluates the
function
and
argument expressions
and then applies the resulting
function
to the resulting arguments. This is not the only way to perform evaluation.
An alternative evaluation model would not evaluate the
arguments
until their values were needed. Instead it would first substitute
argument
expressions for parameters until it obtained an expression involving
only
operators and primitive functions,
and would then perform the evaluation. If we
used this method, the evaluation of

#snippet(```python
f(5)
```)

would proceed according to the sequence of expansions

#snippet(```python
sum_of_squares(5 + 1, 5 * 2)

square(5 + 1)     + square(5 * 2)

(5 + 1) * (5 + 1) + (5 * 2) * (5 * 2)
```)

followed by the reductions

#snippet(```python
6    *    6    +    10   *   10

    36         +        100

              136
```)

This gives the same answer as our previous evaluation model, but the
process is different. In particular, the evaluations of
#py("5 + 1")
and
#py("5 * 2")
are each performed twice here, corresponding to the reduction of the
expression

#snippet(```python
x * x
```)

with #py("x") replaced respectively by
#py("5 + 1")
and
#py("5 * 2").

This alternative "fully expand and then reduce"
evaluation method is known as
#idx("normal-order evaluation")
#emph[normal-order evaluation], in contrast to the "evaluate the arguments and then apply" method that the interpreter actually
uses, which is called
#idx("applicative-order evaluation")
#emph[applicative-order evaluation]. It can be shown that, for
function
applications that can be modeled using substitution (including all the
functions
in the first two chapters of this book) and that yield legitimate values,
normal-order and applicative-order evaluation produce the same value.
(See exercise @ex:normal-order-vs-appl-order-test
for an instance of an "illegitimate" value where normal-order
and applicative-order evaluation do not give the same result.)

Python #idx("applicative-order evaluation", sub: "in Python") #idx("Python", sub: "applicative-order evaluation in")
uses applicative-order evaluation, partly because of the
additional efficiency obtained from avoiding multiple evaluations of
expressions such as those illustrated with
#py("5 + 1") and #py("5 * 2")
above and, more significantly, because normal-order evaluation
becomes much more complicated to deal with when we leave the realm of
functions
that can be modeled by substitution. On the other hand,
normal-order evaluation can be an extremely valuable tool, and we will
investigate some of its implications in chapters 3 and 4.#footnote[In
chapter 3 we will introduce #emph[stream processing], which is a
way of handling apparently "infinite" data structures
by incorporating a limited form of normal-order evaluation. In
section @sec:lazy-evaluation we will modify the
Python
interpreter to produce a normal-order variant of
Python.]
#idx("substitution model of function application")
