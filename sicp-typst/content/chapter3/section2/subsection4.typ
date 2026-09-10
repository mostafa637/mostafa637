// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Internal Declarations], label-name: <sec:env-internal-def>)

#idx("block structure", sub: "in environment model")
#idx("environment model of evaluation", sub: "internal declarations")
#idx("internal declaration", sub: "in environment model")

In this section we handle the evaluation of function bodies or other
blocks (such as the branches of conditional statements) that contain
declarations.
Each block opens a new scope for names declared in the block.
In order to evaluate a block in a given environment, we extend that
environment by a new frame that contains all names declared directly
(that is, outside of nested blocks) in the body of the block and
then evaluate the body in the newly constructed environment.

Section @sec:black-box introduced the idea that
functions
can have internal
declarations,
thus leading to a block structure as in the
#idx("sqrt", sub: "in environment model")
following
function
to compute square roots:

#snippet(```python
def sqrt(x):
    def is_good_enough(guess):
        return abs(square(guess) - x) < 0.001
    def improve(guess):
        return average(guess, x / guess)
    def sqrt_iter(guess):
        return (guess
                if is_good_enough(guess)
                else sqrt_iter(improve(guess)))
    return sqrt_iter(1)
```)

Now we can use the environment model to see why these internal
declarations
behave as desired.
Figure @fig:sqrt-internal
shows the point in the evaluation of the expression
#py("sqrt(2)")
where the internal
function
#py("is_good_enough")
has been called for the first time with
#py("guess") equal to 1\.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-12.svg", width: 70%), caption: [The #py("sqrt") function with internal declarations.], label-name: <fig:sqrt-internal>)

Observe the structure of the environment.
The name #py("sqrt") is bound in the program environment
to a
function
object whose associated environment is the
program
environment. When #py("sqrt") was called, a new
environment, E1, was formed, subordinate to the
program
environment, in which the parameter #py("x") is bound
to 2\. The body of #py("sqrt") was then
evaluated in E1.

That body is a block with local
function definitions and therefore E1 was extended with a new frame for
those declarations, resulting in the new environment E2. The body
of the block was then evaluated in E2. Since the first statement
in the body is

#snippet(```python
def is_good_enough(guess):
    return abs(square(guess) - x) < 0.001
```)

evaluating this declaration created the function
#py("is_good_enough")
in the environment E2.

To be more precise,
the name #py("is_good_enough")
in the first frame of E2 was bound to a function
object whose associated environment is E2.

Similarly,
#py("improve") and
#py("sqrt_iter")
were defined as
functions in E2.
For conciseness,
figure @fig:sqrt-internal
shows only the
function
object for
#py("is_good_enough").

After the local
functions
were defined, the expression
#py("sqrt_iter(1)")
was evaluated, still in environment
E2.
So the
function
object bound to
#py("sqrt_iter") in E2 was called with 1 as an argument. This created an environment E3 in which
#py("guess"), the parameter of
#py("sqrt_iter"),
is bound to 1.
The function #py("sqrt_iter")
in turn called
#py("is_good_enough")
with the value of #py("guess")
(from E3) as the argument for #py("is_good_enough").
This set up another environment,
E4, in which #py("guess") (the parameter of #py("is_good_enough"))
is bound to 1. Although
#py("sqrt_iter")
and
#py("is_good_enough")
both have a parameter named #py("guess"), these are two
distinct local variables located in different frames.

Also, E3 and E4 both have E2 as their enclosing environment, because the
#py("sqrt_iter")
and
#py("is_good_enough") functions
both have E2 as their environment part.

One consequence of this is that the
name
#py("x") that appears in the body of
#py("is_good_enough")
will reference the binding of #py("x") that appears in
E1, namely the value of #py("x") with which the
original #py("sqrt")
function
was called.
#idx("sqrt", sub: "in environment model")

The environment model thus explains the two key properties that make local
function definitions
a useful technique for modularizing programs:

- The names of the local functions do not interfere with names external to the enclosing function, because the local function names will be bound in the frame that the block creates when it is evaluated, rather than being bound in the program environment.
- The local functions can access the arguments of the enclosing function, simply by using parameter names as free names. This is because the body of the local function is evaluated in an environment that is subordinate to the evaluation environment for the enclosing function.

#exercise(label-name: <ex:two-accounts>, [
In section @sec:env-local-state we saw how the
environment model described the behavior of
functions
with local state. Now we have seen how internal
declarations
work.
#idx("environment model of evaluation", sub: "message passing")
#idx("message passing", sub: "environment model and")
A typical message-passing
function
contains both of these aspects. Consider the
#idx("bank account")
bank account
function
of section @sec:local-state-variables:
#idx("makeaccount", sub: "in environment model")
#snippet(```python
def make_account(balance):
    def withdraw(amount):
        nonlocal balance
        if balance >= amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    def deposit(amount):
        nonlocal balance
        balance = balance + amount
        return balance
    def dispatch(m):
        return (withdraw if m == "withdraw"
                else deposit if m == "deposit"
                else error("Unknown request: make_account", m))
    return dispatch
```)

Show the environment structure generated by the sequence of
interactions

#snippet(```python
acc = make_account(50)
```)

#snippet(```python
print(acc("deposit")(40))
```)

#output(```python
print(acc("deposit")(40))
```)

#snippet(```python
print(acc("withdraw")(60))
```)

#output(```python
print(acc("withdraw")(60))
```)

Where is the local state for #py("acc") kept?
Suppose we define another account

#snippet(```python
acc2 = make_account(100)
```)

How are the local states for the two accounts kept distinct? Which parts
of the environment structure are shared between
#py("acc") and #py("acc2")?
])

#subheading([More about blocks])

As we saw, the scope of the names declared in
#py("sqrt") is the whole body of
#py("sqrt"). This explains why
#idx("mutual recursion")
#idx("recursion", sub: "mutual")
#emph[mutual recursion] works, as in this (quite
wasteful) way of checking whether a nonnegative
integer is even.

#syntax("
def f(x):
    def is_even(n):
        return (True
                if n == 0
                else is_odd(n - 1))
    def is_odd(n):
        return (False
                if n == 0
                else is_even(n - 1))
    return is_even(x)
      ")

At the time when
#py("is_even") is called during a call to
#py("f"), the environment diagram looks
like the one in figure @fig:sqrt-internal when
#py("sqrt_iter") is called. The functions
#py("is_even") and
#py("is_odd") are bound in E2 to function objects
that point to E2 as the environment in which to evaluate calls to those
functions. Thus
#py("is_odd") in the body of
#py("is_even") refers to the right function.
Although
#py("is_odd")
is defined after
#py("is_even"),
this is no different from how in the body of
#py("sqrt_iter")
the name
#py("improve")
and the name
#py("sqrt_iter")
itself refer to the right functions.

Equipped with a way to handle declarations within blocks, we can
revisit declarations of names at the top level. In
section @sec:env-model-rules, we saw
that the names declared at the top level are added to the program
frame. A better explanation is that the whole program is placed in
an implicit block, which is evaluated in the global environment.
The treatment of blocks described above then handles the top
level:
The global environment is extended by a frame that contains the
bindings of all names declared in the implicit block. That frame is
the program frame and the resulting
environment is the
#idx("program environment")
program environment.

We said that a block's body is evaluated in an environment that
contains all names declared directly in the body of the block.
A locally declared name is put into the environment when the block is
entered, but without an associated value. The evaluation of its
declaration during evaluation of the block body then assigns to the
name the result of evaluating the expression to the right of the
#py("="), as if the declaration were
an assignment. Since the addition of the name to the environment is
separate from the evaluation of the declaration, and the whole block
is in the scope of the name, an erroneous program could attempt to
#idx("declaration", sub: "use of name before")
access the value of a name before its declaration is evaluated;
the evaluation of an unassigned name signals an error.#footnote[This explains why the program in
footnote @foot:tdz of chapter 1 goes wrong.
The time between creating the binding for a name and evaluating
the declaration of the name is called the
#idx("temporal dead zone (TDZ)")
#idx("TDZ (temporal dead zone)")
#emph[temporal dead zone] (TDZ).]<foot:tdz_explained>

#idx("environment model of evaluation")
#idx("block structure", sub: "in environment model")
#idx("environment model of evaluation", sub: "internal declarations")
#idx("internal declaration", sub: "in environment model")
