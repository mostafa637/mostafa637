// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([The Rules for Evaluation], label-name: <sec:env-model-rules>)

#idx("environment model of evaluation", sub: "rules for evaluation")

The overall specification of how the interpreter
#idx("function application", sub: "environment model of")
#idx("environment model of evaluation", sub: "function application")
evaluates a
function application
remains the same as when we first introduced it in
section @sec:compound-procedures:

- To evaluate an application:

  + Evaluate the subexpressions of the application.#footnote[Assignment introduces a subtlety into step 1 of the evaluation rule. As shown in exercise @ex:order-of-evaluation, the presence of assignment allows us to write expressions that will produce different values depending on the order in which the subexpressions in a combination are evaluated. To eliminate such ambiguities, #idx("order of evaluation", sub: "in Python") Python specifies left-to-right evaluation of the subexpressions of combinations and of the argument expressions of applications.]
  + Apply the value of the function subexpression to the values of the argument subexpressions.

The environment model of evaluation replaces the substitution model in
specifying what it means to apply a compound
function
to arguments.

In the environment model of evaluation, a
function
is always a pair consisting of some code and a pointer to an environment.
Functions
are created in one way only: by evaluating a
lambda
expression.

This produces a
function
whose code is obtained from the text of the
lambda
expression and whose environment is the environment in which the
lambda
expression was evaluated to produce the
function.
For example, consider the
function definition
#idx("square", sub: "in environment model")

#snippet(```python
def square(x):
    return x * x
```)

evaluated in the
program
environment. The
function definition
syntax is
equivalent to
an underlying implicit
lambda
expression. It would have been equivalent to have used#footnote[Footnote #text(fill: red)[?] in chapter 1
mentions subtle differences between the two in full Python, which
we will ignore in this book.]

#snippet(```python
square = lambda x: x * x
```)

which evaluates
#py("lambda x: x * x")
and binds #py("square") to the resulting value, all
in the
program
environment.

Figure @fig:evaluating-square
shows the result of evaluating this
#idx("declaration", sub: "environment model of")
declaration statement.

The global environment encloses the program environment. To reduce
clutter, after this figure we will not display the global environment
(as it is always the same), but we are reminded of its existence by the
pointer from the program environment upward.

The
function
object is a pair whose code specifies that the
function
has one

parameter, namely #py("x"), and a
function
body
#py("return x * x").
The environment part of the
function
is a pointer to the program environment, since that is the environment in
which the
lambda
expression was evaluated to produce the
function.
A new binding, which associates the
function
object with the
name
#py("square"), has been added
to the program frame.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-3.svg", width: 70%), caption: [Environment structure produced by evaluating #py("def square(x): return x * x") in the program environment.], label-name: <fig:evaluating-square>)

In general, function definitions and
declaration assignments
add bindings to frames.
Assignment is forbidden on constants, so our environment model
needs to distinguish names that refer to constants
from names that refer to variables. We indicate that
a name is a constant by writing an equal sign after the colon
that follows the name.
We consider function definitions as equivalent to constant
declarations;#footnote[We mentioned in
footnote #text(fill: red)[?]
in chapter 1
that the full Python language allows assignment to
names that are declared with function definitions.]
observe the equal signs after the colons in
figure @fig:evaluating-square.

Now that we have seen how
functions
are created, we can describe how
functions
are applied. The environment model specifies: To apply a
function
to arguments, create a new environment containing a frame that binds the
parameters to the values of the arguments. The enclosing environment of
this frame is the environment specified by the
function.
Now, within this new environment, evaluate the
function
body.

To show how this rule is followed,
figure @fig:square5-eval
illustrates the environment structure created by evaluating the
expression
#py("square(5)")
in the
program
environment, where #py("square") is the
function
generated in
figure @fig:evaluating-square.
Applying the
function
results in the creation of a new environment, labeled E1 in the figure, that
begins with a frame in which #py("x"), the

parameter for the
function,
is bound to the argument 5.
Note that name #py("x") in environment E1 is followed by a colon with no equal sign, which indicates that the parameter #py("x") is treated as a variable.#footnote[This example does not make use of the fact that the parameter #py("x") is a variable, but recall the function #py("make_withdraw") in section @sec:local-state-variables, which relied on its parameter being a variable.]
The pointer leading upward from this frame shows that the
frame's enclosing environment is the
program
environment. The
program
environment is chosen here, because this is the environment that is
indicated as part of the #py("square")
function
object. Within E1, we evaluate the body of the
function,
#py("return x * x").
Since the value of #py("x") in E1 is 5, the result is
#py("5 * 5"),
or 25\.
#idx("square", sub: "in environment model")

#sicp-figure(image("/images/img_javascript/ch3-Z-G-4.svg", width: 70%), caption: [Environment created by evaluating #py("square(5)") in the program environment.], label-name: <fig:square5-eval>)

The environment model of
function
application can be summarized by two
rules:

- A function object is applied to a set of arguments by constructing a frame, binding the parameters of the function to the arguments of the call, and then evaluating the body of the function in the context of the new environment constructed. The new frame has as its enclosing environment the environment part of the function object being applied. The result of the application is the result of evaluating the return expression of the first return statement encountered while evaluating the function body.
- A function is created by evaluating a lambda expression relative to a given environment. The resulting #idx("lambda expression", sub: "value of") function object is a pair consisting of the text of the lambda expression and a pointer to the environment in which the function was created.

#idx("assignment", sub: "evaluation of")
Finally, we specify the behavior of reassignment, the operation that
forced us to introduce the environment model in the first place.
Evaluating the reassignment statement
#meta("name") #py("=") #meta("value")
in some environment
locates the binding of the name in the environment.
That is, one finds the first frame in the environment that contains a
binding for the name and changes
that binding to reflect the new
value of the variable.
If the name is unbound in the environment, then
the assignment signals a
#py("\"variable undeclared\"") error.

These evaluation rules, though considerably more complex than the
substitution model, are still reasonably straightforward. Moreover,
the evaluation model, though abstract, provides a correct description
of how the interpreter evaluates expressions. In chapter @chap:meta we shall
see how this model can serve as a blueprint for implementing a working
interpreter. The following sections elaborate the details of the
model by analyzing some illustrative programs.
#idx("environment model of evaluation", sub: "rules for evaluation")
