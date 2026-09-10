// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Frames as the Repository of Local State], label-name: <sec:env-local-state>)

#idx("frame (environment model)", sub: "as repository of local state")
#idx("local state", sub: "maintained in frames")
#idx("environment model of evaluation", sub: "local state")

We can turn to the environment model to see how
functions
and assignment can be used to represent objects with local state. As an
example, consider the
#idx("makewithdraw", sub: "in environment model")
"withdrawal processor" from
section @sec:local-state-variables created by calling the
function

#snippet(```python
def make_withdraw(balance):
    def withdraw(amount):
        nonlocal balance
        if balance >= amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    return withdraw
```)

Let us describe the evaluation of

#snippet(```python
W1 = make_withdraw(100)
```)

followed by

#snippet(```python
print(W1(50))
```)

#output(```python
print(W1(50))
```)

Figure @fig:make-withdraw
shows the result of
declaring the #py("make_withdraw")
function
in the
program
environment. This produces a
function
object that contains a pointer to the
program
environment. So far, this is no different from the examples we have already
seen, except that
the return expression in the body of the function is itself a lambda expression.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-7.svg", width: 70%), caption: [Result of defining #py("make_withdraw") in the program environment.], label-name: <fig:make-withdraw>)

The interesting part of the computation happens when we apply the
function
#py("make_withdraw")
to an argument:

#snippet(```python
W1 = make_withdraw(100)
```)

We begin, as usual, by setting up an environment E1 in which the
 parameter
#py("balance") is bound to the argument 100. Within
this environment, we evaluate the body of
#py("make_withdraw"),
namely the
return statement whose return expression is a lambda expression. The evaluation of this lambda expression
constructs a new
function
object, whose code is as specified by the
lambda expression
and whose environment is E1, the environment in which the
lambda expression
was evaluated to produce the
function.
The resulting
function
object is the value returned by the call to
#py("make_withdraw").
This is bound to #py("W1") in the
program
environment, since the
constant declaration
itself is being evaluated in the
program
environment.
Figure @fig:w1
shows the resulting environment structure.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-8.svg", width: 70%), caption: [Result of evaluating #py("W1 = make_withdraw(100)").], label-name: <fig:w1>)

Now we can analyze what happens when #py("W1")
is applied to an argument:

#snippet(```python
print(W1(50))
```)

#output(```python
print(W1(50))
```)

We begin by constructing a frame in which
#py("amount"), the
 parameter of
#py("W1"), is bound to the argument 50. The crucial
point to observe is that this frame has as its enclosing environment not the
program
environment, but rather the environment E1, because this is the
environment that is specified by the #py("W1")
function
object. Within this new environment, we evaluate the body of the
function:

#snippet(```python
if balance >= amount:
    balance = balance - amount
    return balance
else:
    return "Insufficient funds"
```)

The resulting environment structure is shown in
figure @fig:apply-w1.
The expression being evaluated references
both #py("amount") and
#py("balance").
The variable #py("amount")
will be found in the first frame in the environment, and
#py("balance") will be found by following the
enclosing-environment pointer to E1.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-9.svg", width: 70%), caption: [Environments created by applying the function object #py("W1").], label-name: <fig:apply-w1>)

When the
assignment
is executed, the binding of #py("balance") in E1 is
changed. At the completion of the call to
#py("W1"), #py("balance") is 50,
and the frame that contains #py("balance") is still
pointed to by the
function
object #py("W1"). The frame that binds
#py("amount") (in which we executed the code that
changed #py("balance")) is no longer relevant, since
the
function
call that constructed it has terminated, and there are no pointers to that
frame from other parts of the environment. The next time
#py("W1") is called, this will build a new frame that
binds #py("amount") and whose enclosing environment is
E1. We see that E1 serves as the "place" that holds the local
state variable for the
function
object #py("W1").
Figure @fig:after-w1
shows the situation after the call to #py("W1").

#sicp-figure(image("/images/img_javascript/ch3-Z-G-10.svg", width: 70%), caption: [Environments after the call to #py("W1").], label-name: <fig:after-w1>)

Observe what happens when we create a second "withdraw" object
by making another call to
#py("make_withdraw"):

#snippet(```python
W2 = make_withdraw(100)
```)

This produces the environment structure of
figure @fig:w2,
which shows
that #py("W2") is a
function
object, that is, a pair with some code and an environment. The environment
E2 for #py("W2") was created by the call to
#py("make_withdraw").
It contains a frame with its own local binding for
#py("balance"). On the other hand,
#py("W1") and #py("W2") have the
same code: the code specified by the
lambda
expression in the body of
#py("make_withdraw").#footnote[Whether
#py("W1") and #py("W2") share
the same physical code stored in the computer, or whether they each keep a
copy of the code, is a detail of the implementation. For the interpreter we
implement in chapter @chap:meta, the code is in fact shared.] We see
here why #py("W1") and #py("W2")
behave as independent objects. Calls to
#py("W1") reference the state variable
#py("balance") stored in E1, whereas calls to
#py("W2") reference the
#py("balance") stored in E2. Thus, changes to the
local state of one object do not affect the other object.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-11.svg", width: 70%), caption: [Using #py("W2 = make_withdraw(100)") to create a second object.], label-name: <fig:w2>)

#exercise(label-name: <ex:local-state-variable>, [
In the
#py("make_withdraw")
function
the local variable #py("balance") is created as a
parameter of
#py("make_withdraw").
We could also create the local state variable
separately,
using
what we might call an #idx("lambda expression", sub: "immediately invoked") #idx("immediately invoked lambda expression") #emph[immediately invoked lambda expression]
as follows:
#idx("makewithdraw", sub: "using immediately invoked lambda expression", decl: true)
#snippet(```python
def make_withdraw(initial_amount):
    def init(balance):
        def withdraw(amount):
            nonlocal balance
            if balance >= amount:
                balance = balance - amount
                return balance
            else:
                return "Insufficient funds"
        return withdraw
    return init(initial_amount)
```)

The #py("init") functino
is invoked immediately after it
is evaluated. Its only purpose is to create a local variable
#py("balance") and
initialize it to #py("initial_amount").

Use the environment model to analyze this alternate version of
#py("make_withdraw"), drawing figures like the ones
above to illustrate the interactions

#snippet(```python
W1 = make_withdraw(100)

W1(50)

W2 = make_withdraw(100)
```)

Show that the two versions of
#py("make_withdraw")
create objects with the same behavior. How do the environment structures
differ for the two versions?
])

#idx("frame (environment model)", sub: "as repository of local state")
#idx("local state", sub: "maintained in frames")
#idx("environment model of evaluation", sub: "local state")
#idx("makewithdraw", sub: "in environment model")
