// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([The Costs of Introducing Assignment], label-name: <sec:costs-of-assignment>)

#idx("assignment", sub: "costs of")

As we have seen,
assignment
enables us to model objects
that have local state. However, this advantage comes at a price. Our
programming language can no longer be interpreted in terms of the
substitution model of
function
application that we introduced in
section @sec:substitution-model. Moreover, no simple
model with "nice" mathematical properties can be an adequate
framework for dealing with objects and assignment in programming languages.

So long as we do not use assignments, two evaluations of the same
function
with the same arguments will produce the same result, so that
functions
can be viewed as computing mathematical functions. Programming without any
use of assignments, as we did throughout the first two chapters of this
book, is accordingly known as
#idx("functional programming")
#emph[functional programming].

#idx("substitution model of function application", sub: "inadequacy of")
To understand how assignment complicates matters, consider a simplified
version of the #py("make_withdraw")
function
of section @sec:local-state-variables that does not
bother to check for an insufficient amount:
#idx("makesimplifiedwithdraw", decl: true)
#snippet(```python
def make_simplified_withdraw(balance):
    def withdraw(amount):
        nonlocal balance
        balance = balance - amount
        return balance
    return withdraw
```)

#snippet(```python
W = make_simplified_withdraw(25)
```)

#snippet(```python
print(W(20))
```)

#output(```python
print(W(20))
```)

#snippet(```python
print(W(10))
```)

#output(```python
print(W(10))
```)

Compare this
function
with the following #py("make_decrementer")
function,
which does not use
assignment:
#idx("makedecrementer", decl: true)
#snippet(```python
def make_decrementer(balance):
    return lambda amount: balance - amount
```)

The function #py("make_decrementer")
returns a
function
that subtracts its input from a designated amount
#py("balance"), but there is no accumulated effect
over successive calls, as with
#py("make_simplified_withdraw"):

#snippet(```python
D = make_decrementer(25)
```)

#snippet(```python
print(D(20))
```)

#output(```python
print(D(20))
```)

#snippet(```python
print(D(10))
```)

#output(```python
print(D(10))
```)

We can use the substitution model to explain how
#py("make_decrementer") works. For instance, let us
analyze the evaluation of the expression

#snippet(```python
print(make_decrementer(25)(20))
```)

We first simplify the
function expression of the application
by substituting
$25$ for #py("balance") in
the body of
#py("make_decrementer").
This reduces the
expression to

#snippet(```python
(lambda amount: 25 - amount)(20)
```)

Now we apply the
function
by substituting 20 for
#py("amount") in the body of the
lambda
expression:

#snippet(```python
print(25 - 20)
```)

The final answer is 5.

Observe, however, what happens if we attempt a similar substitution analysis
with #py("make_simplified_withdraw"):

#snippet(```python
make_simplified_withdraw(25)(20)
```)

We first simplify the
function expression
by substituting 25 for
#py("balance") in
the body of
#py("make_simplified_withdraw"). This reduces the
expression
to#footnote[We don't substitute for the occurrence of
#py("balance") in the
assignment
because the name in
an assignment
is not evaluated. If we did substitute for it, we would get
#py("25 = 25 - amount"),
which makes no sense.]

#snippet(```python
def withdraw(amount):
    balance = 25 - amount
    return 25

withdraw(20)
```)

Now we apply the
function
by substituting 20 for #py("amount")
in the body of the
function definition:

#snippet(```python
balance = 25 - 20
return 25
```)

If we adhered to the substitution model, we would have to say that the
meaning of the
function
application is to first set #py("balance") to 5 and
then return 25 as the value of the expression. This gets the wrong answer.
In order to get the correct answer, we would have to somehow distinguish the
first occurrence of #py("balance") (before the effect
of the
assignment)
from the second occurrence of #py("balance")
(after the effect of the
assignment),
and the substitution model cannot do this.

The trouble here is that substitution is based ultimately on the notion that
the names in our language are essentially symbols for values.

This worked well for constants.
But a variable, whose value can change with assignment, cannot simply
be a name for a value. A variable somehow refers to a place where a
value can be stored, and the value stored at this place can change.
#idx("substitution model of function application", sub: "inadequacy of")

In section @sec:environment-model we will see how
environments play this role of "place" in our computational
model.

#subheading([Sameness and change])

#idx("sameness and change", sub: "meaning of")
#idx("change and sameness", sub: "meaning of")

The issue surfacing here is more profound than the mere breakdown of a
particular model of computation. As soon as we introduce change into
our computational models, many notions that were previously
straightforward become problematical. Consider the concept of two
things being "the same."

Suppose we call
#py("make_decrementer")
twice with the same argument to create two
functions:

#snippet(```python
D1 = make_decrementer(25)

D2 = make_decrementer(25)
```)

Are
#py("D1")
and
#py("D2")
the same? An acceptable answer is yes, because
#py("D1")
and
#py("D2")
have the same computational behavior—each is a
function
that subtracts its input from 25. In fact,
#py("D1")
could be substituted for
#py("D2")
in any computation without changing the result.

Contrast this with making two calls to
#py("make_simplified_withdraw"):

#snippet(```python
W1 = make_simplified_withdraw(25)

W2 = make_simplified_withdraw(25)
```)

Are #py("W1") and
#py("W2") the same? Surely not, because calls to
#py("W1") and #py("W2")
have distinct effects, as shown by the following sequence of interactions:

#snippet(```python
print(W1(20))
```)

#output(```python
print(W1(20))
```)

#snippet(```python
print(W1(20))
```)

#output(```python
print(W1(20))
```)

#snippet(```python
print(W2(20))
```)

#output(```python
print(W2(20))
```)

Even though #py("W1") and
#py("W2") are "equal" in the sense that
they are both created by evaluating the same expression,
#py("make_simplified_withdraw(25)"),
it is not true that
#py("W1") could be substituted for
#py("W2") in any expression without changing the
result of evaluating the expression.

A language that supports the concept that "equals can be substituted for equals" in an expression without changing the value of the
expression is said to be
#idx("referential transparency")
#idx("transparency, referential")
#idx("equality", sub: "referential transparency and")
#emph[referentially transparent]. Referential transparency is violated
when we include
assignment
in our computer language. This makes it tricky to determine when we can
simplify expressions by substituting equivalent expressions. Consequently,
reasoning about programs that use assignment becomes drastically more
difficult.

Once we forgo referential transparency, the notion of what it means for
computational objects to be "the same" becomes difficult to
capture in a formal way. Indeed, the meaning of "same" in the
real world that our programs model is hardly clear in itself. In general,
we can determine that two apparently identical objects are indeed
"the same one" only by modifying one object and then observing
whether the other object has changed in the same way. But how can we tell
if an object has "changed" other than by observing the
"same" object twice and seeing whether some property of the
object differs from one observation to the next? Thus, we cannot determine
"change" without some a priori notion of
"sameness," and we cannot determine sameness without observing
the effects of change.

As an example of how this issue arises in programming, consider the
situation where Peter and Paul have a
#idx("bank account", sub: "joint")
bank account with \$100 in
it. There is a substantial difference between modeling this as

#snippet(```python
peter_acc = make_account(100)
paul_acc = make_account(100)
```)

and modeling it as

#snippet(```python
peter_acc = make_account(100)
paul_acc = peter_acc
```)

In the first situation, the two bank accounts are distinct.
Transactions made by Peter will not affect Paul's account, and vice
versa. In the second situation, however, we have defined
#py("paul_acc")
to be #emph[the same thing] as
#py("peter_acc").
In effect, Peter and Paul now have a joint bank account, and if Peter makes
a withdrawal from
#py("peter_acc")
Paul will observe less money in
#py("paul_acc").
These two similar but distinct situations can cause confusion in building
computational models. With the shared account, in particular, it can be
especially confusing that there is one object (the bank account) that has
two different names
(#py("peter_acc") and #py("paul_acc"))
if we are searching for all the places in our program where
#py("paul_acc")
can be changed, we must remember to look also at things that change
#py("peter_acc").#footnote[The
phenomenon of a single computational object being accessed by more than one
name is known as
#idx("aliasing")
#emph[aliasing]. The joint bank account situation illustrates a very
simple example of an alias. In section @sec:mutable-data
we will see much more complex examples, such as "distinct"
compound data structures that share parts. Bugs can occur in our programs if
we forget that a change to an object may also, as a
#idx("side-effect bug")
#idx("bug", sub: "side effect with aliasing")
#idx("assignment", sub: "bugs associated with")
"side effect," change a "different" object because
the two "different" objects are actually a single object
appearing under different aliases. These so-called #emph[side-effect bugs] are so difficult to locate and to analyze that some people have
proposed that programming languages be designed in such a way as to not
allow side effects or aliasing
#idx("Lampson, Butler")
#idx("Morris, J. H.")
#idx("Schmidt, Eric")
#idx("Wadler, Philip")
(Lampson et al. 1981;
Morris, Schmidt, and Wadler 1980).]

With reference to the above remarks on "sameness" and
"change," observe that if Peter and Paul could only examine
their bank balances, and could not perform operations that changed the
balance, then the issue of whether the two accounts are distinct would be
moot. In general, so long as we never modify data objects, we can regard a
compound data object to be precisely the totality of its pieces. For
example, a rational number is determined by giving its numerator and
its denominator. But this view is no longer valid in the presence of
change, where a compound data object has an "identity" that is
something different from the pieces of which it is composed. A bank
account is still "the same" bank account even if we change the
balance by making a withdrawal; conversely, we could have two
different bank accounts with the same state information. This
complication is a consequence, not of our programming language, but of

our perception of a bank account as an object. We do not, for
example, ordinarily regard a rational number as a changeable object
with identity, such that we could change the numerator and still have
"the same" rational number.
#idx("sameness and change", sub: "meaning of")
#idx("change and sameness", sub: "meaning of")

#subheading([Pitfalls of imperative programming])

In contrast to functional programming, programming that makes extensive use
of assignment is known as
#idx("imperative programming")
#idx("programming", sub: "imperative")
#emph[imperative programming]. In addition to raising complications about
computational models, programs written in imperative style are susceptible
to bugs that cannot occur in functional programs. For example, recall the
iterative factorial program from
section @sec:recursion-and-iteration (here using a conditional statement instead of a conditional expression):

#snippet(```python
def factorial(n):
    def iter(product, counter):
        if counter > n:
            return product
        else:
            return iter(counter * product,
                        counter + 1)
    return iter(1, 1)
```)

Instead of passing arguments in the internal iterative loop, we could
adopt a more imperative style by using explicit assignment
to update the values of the variables #py("product")
and #py("counter"):
#idx("factorial", sub: "with assignment", decl: true)
#snippet(```python
def factorial(n):
    product = 1
    counter = 1
    def iter():
        nonlocal product, counter
        if counter > n:
            return product
        else:
            product = counter * product
            counter = counter + 1
            return iter()
    return iter()
```)

This does not change the results produced by the program, but it does
introduce a subtle trap. How do we decide the order of the assignments?
As it happens, the program is correct as written. But writing the
assignments in the opposite order

#snippet(```python
counter = counter + 1
product = counter * product
```)

would have produced a different,
#idx("bug", sub: "order of assignments")
#idx("assignment", sub: "bugs associated with")
incorrect result. In general, programming
with assignment forces us to carefully consider the relative orders of the
assignments to make sure that each statement is using the correct version
of the variables that have been changed. This issue simply does not arise
in functional programs.#footnote[In view of this, it is ironic that
introductory programming is most often taught in a highly imperative style.
This may be a vestige of a belief, common throughout the 1960s and 1970s,
that programs that call
functions
must inherently be less efficient than programs that perform assignments.
(Steele (1977)
#idx("Steele, Guy Lewis Jr.")
debunks this argument.) Alternatively it may reflect a view that
step-by-step assignment is easier for beginners to visualize than
function
call. Whatever the reason, it often saddles beginning programmers with
"should I set this variable before or after that one" concerns
that can complicate programming and obscure the important ideas.]

The complexity of imperative programs becomes even worse if we consider
applications in which several processes execute concurrently. We will
return to this in section @sec:time-is-of-the-essence.
First, however, we will address the issue of providing a computational
model for expressions that involve assignment, and explore the uses of
objects with local state in designing simulations.

#exercise(label-name: <ex:3_7>, [
Consider the bank account objects created by
#py("make_account"),
with the password modification described in
exercise @ex:password-protection. Suppose that our
banking system requires the ability to make
#idx("bank account", sub: "joint")
joint accounts. Define a
function
#idx("makejoint")
#py("make_joint")
that accomplishes this.
The function #py("make_joint")
should take three arguments. The first is a password-protected account.
The second argument must match the password with which the account was
defined in order for the
#py("make_joint")
operation to proceed. The third argument is a new password.
The function #py("make_joint")
is to create an additional access to the original account using the new
password. For example, if
#py("peter_acc")
is a bank account with
password
#py("\"open sesame\""),
then

#snippet(```python
paul_acc = make_joint(peter_acc, "open sesame", "rosebud")
```)

will allow one to make transactions on
#py("peter_acc")
using the name
#py("paul_acc")
and the password
#py("\"rosebud\"").
You may wish to modify your solution to
exercise @ex:password-protection to accommodate this
new feature.
])

#exercise(label-name: <ex:3_8>, [
When we defined the evaluation model in
section @sec:evaluating-combinations, we said that the
#idx("order of evaluation", sub: "in Python")
#idx("order of evaluation", sub: "assignment and")
first step in evaluating an expression is to evaluate its subexpressions.
But we never specified the order in which the subexpressions should be
evaluated (e.g., left to right or right to left).
When we introduce assignment, the order in which the operands of an operator combination are evaluated can make a difference to the result.
Define a simple
function
#py("f") such that evaluating
#py("f(0) + f(1)")
will return 0 if the
operands of #py("+")
are evaluated from left to right but will return 1 if the
operands
are evaluated from right to left.
#anchor(<ex:order-of-evaluation>)
])

#idx("assignment")
#idx("local state")
#idx("assignment", sub: "costs of")
