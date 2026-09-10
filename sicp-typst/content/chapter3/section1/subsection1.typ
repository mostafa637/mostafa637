// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Local State Variables], label-name: <sec:local-state-variables>)

#idx("local state variable")
#idx("state variable", sub: "local")

To illustrate what we mean by having a computational object with
#idx("object(s)", sub: "with time-varying state")
time-varying state, let us model the situation of withdrawing money
from a
#idx("bank account")
bank account. We will do this using a
function
#py("withdraw"), which takes as argument an
#py("amount") to be withdrawn.
If there is enough money in the account to accommodate the withdrawal,
then #py("withdraw") should return the balance
remaining after the withdrawal. Otherwise,
#py("withdraw") should return the message
#emph[Insufficient funds.] For example, if we begin with \$100
in the account, we should obtain the following sequence of responses
using
#py("withdraw"):

#snippet(```python
print(withdraw(25))
```)

#output(```python
print(withdraw(25))
```)

#snippet(```python
print(withdraw(25))
```)

#output(```python
print(withdraw(25))
```)

#snippet(```python
print(withdraw(60))
```)

#output(```python
print(withdraw(60))
```)

#snippet(```python
print(withdraw(15))
```)

#output(```python
print(withdraw(15))
```)

Observe that the expression
#py("withdraw(25)"),
evaluated twice, yields different values. This is a new kind of
behavior for a
function.
Until now, all our
Python functions
could be viewed as specifications for computing mathematical functions.
A call to a
function
computed the value of the function applied to the given arguments,
and two calls to the same
function
with the same arguments always produced the same
result.#footnote[Actually, this is not quite true. One exception was the
#idx("mathrandom (primitive function)", sub: "reassignment needed for")
#idx("random-number generator")
random-number generator
in section @sec:primality. Another exception
involved the
#idx("operation-and-type table", sub: "assignment needed for")
operation/type tables we introduced in
section @sec:data-directed, where the values of two
calls to #py("get") with the same arguments
depended on intervening calls to #py("put").
On the other hand, until we introduce
reassignment,
we have no way to
create such
functions
ourselves.]

To implement #py("withdraw"), we can use a
variable #py("balance") to indicate the balance of
money in the account and define #py("withdraw")
as a
function
that accesses #py("balance").
The #py("withdraw")
function
checks to see if #py("balance") is at least as large
as the requested #py("amount"). If so,
#py("withdraw") decrements
#py("balance") by #py("amount")
and returns the new value of #py("balance"). Otherwise,
#py("withdraw") returns the #emph[Insufficient funds]
message. Here are the
declarations
of #py("balance") and
#py("withdraw"):
#idx("withdraw", decl: true)
#snippet(```python
balance = 100

def withdraw(amount):
    global balance
    if balance >= amount:
        balance = balance - amount
        return balance
    else:
        return "Insufficient funds"
```)

Decrementing #py("balance") is accomplished by the
assignment

#snippet(```python
balance = balance - amount
```)

This statement looks like a declaration assignment, but it does
not declare a new variable. The global declaration

#snippet(```python
global balance
```)

ensures that the name
#py("balance") is already
declared in the	body of the function
#py("withdraw")
and refers to the globally declared variable
#py("balance").
#idx("reassignment")
#idx("reassignment", sub: "reassignment statement")
#idx("variable", sub: "reassignment to")
#idx("syntactic forms", sub: "reassignment")
#idx("=", sort: "=")
An assignment of the form

#syntax(meta("name"), " = ", meta("new-value"))

where the #meta("name") is already declared, is called
a #emph[reassignment].

The reassignment
changes
#meta("name")
so that its value is the
result obtained by evaluating
#meta("new-value").
In the case at hand, we are changing #py("balance") so
that its new value will be the result of subtracting
#py("amount") from the previous value of
#py("balance").#footnote[Reassignment statements and declaration assignments look similar to and should not be confused with #idx("assignment", sub: "equality test vs.") expressions of the form #syntax(meta("expression_1"), " == ", meta("expression_2")) which evaluate to #py("True") if #meta("expression")$""_(1)$ evaluates to the same value as #meta("expression")$""_(2)$ and to #py("False") otherwise.]

The function #py("withdraw") also uses a
#idx("sequence of statements")
#emph[sequence of statements] to cause two statements to be evaluated
in the case where the #py("if") test is
true: first decrementing #py("balance")
and then returning the value of
#py("balance").
In general, executing a sequence

#syntax(meta("stmt"), $""_(1)$, " ", meta("stmt"), $""_(2) dots.h$, meta("stmt"), $""_(n)$)

causes the statements #meta("stmt")$""_(1)$
through
#meta("stmt")$""_(n)$ to be evaluated in
sequence.#footnote[We have already used
#idx("sequence of statements", sub: "in block")
sequences implicitly in our programs, because in
Python the body block
of a function can contain a sequence of function definitions
followed by a return statement, not
just a single return statement,
as discussed in
section @sec:block-structure.]

Although #py("withdraw") works as desired, the
variable #py("balance") presents a problem. As
specified above, #py("balance") is a name defined
in the
program environment and is freely accessible to be examined or
modified by any
function.
It would be much better if we could somehow make
#py("balance") internal to
#py("withdraw"), so that
#py("withdraw") would be the only
function
that could access #py("balance") directly and
any other
function
could access #py("balance") only indirectly
(through calls to #py("withdraw")). This would
more accurately model the notion that
#py("balance") is a local state variable used by
#py("withdraw") to keep track of the state of the
account.

We can make #py("balance") internal to
#py("withdraw") by rewriting the definition as
follows:

#idx("newwithdraw", decl: true)
#snippet(```python
def make_withdraw_balance_100():
    balance = 100
    def withdraw(amount):
        nonlocal balance
        if balance >= amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    return withdraw

new_withdraw = make_withdraw_balance_100()
```)

What we have done here is use a declaration assignment
to establish an environment with a local variable
#py("balance"), bound to the initial
value 100. Within this local environment, we use a function
declaration to
create a function
#py("withdraw")
that takes #py("amount")
as an argument and—returned as the result of evaluating the body of the
#py("make_withdraw_balance_100")
function—behaves in precisely the same way as
our previous
#py("withdraw")
function,
but its variable
#py("balance") is not accessible by any
other function.#footnote[In programming-language jargon, the variable
#py("balance") is said to be
#idx("encapsulated name")
#idx("name", sub: "encapsulated")
#emph[encapsulated] within the
#py("new_withdraw") function.
Encapsulation reflects the general system-design principle known as the
#idx("hiding principle")
#idx("modularity", sub: "hiding principle")
#emph[hiding principle]: One can
make a system more modular and robust by protecting parts of the
system from each other; that is, by providing information access only
to those parts of the system that have a "need to know."]

The nested #py("withdraw")
function faces a similar problem as the previous
global #py("withdraw") function:
The assignment
balance = balance - amount
should refer to the variable balance declared outside
of the #py("withdraw") function,
and should not redeclare the name
balance. However, now the name
balance is not a global name, but a name that is declared
in the surrounding function
make\_withdraw\_balance\_100.
Python uses nonlocal declarations in this situation.
The declaration
nonlocal balance
in the nested withdraw function indicates that any assignment
to the name balance inside the withdraw function refers
to the variable balance declared outside the function
but not globally.#footnote[Similar to global declarations,
a nonlocal declaration does not extend to functions that
are nested inside the function in which the nonlocal declaration
appears. For these nested functions to reassign the
nonlocal variable, they also need to declare the variable as nonlocal.]

Combining
reassignments with declaration assignments
is the general programming
technique we will use for constructing computational objects with
local state. Unfortunately, using this technique raises a serious
problem: When we first introduced
functions,
we also introduced the substitution model of evaluation
(section @sec:substitution-model) to provide an
interpretation of what
function
application means. We said that applying a
function whose body is a return statement
should be interpreted as evaluating the
return expression of the function
with the

parameters replaced by their values.
For functions with more complex bodies, we need to evaluate the whole body with the parameters replaced by their values.
The trouble is that,
as soon as we introduce assignment into our language, substitution is no
longer an adequate model of
function
application. (We will see why this is so in
section @sec:costs-of-assignment.) As a consequence, we
technically have at this point no way to understand why the
#py("new_withdraw")
function
behaves as claimed above. In order to really understand a
function
such as
#py("new_withdraw"),
we will need to develop a new model of
function
application. In section @sec:environment-model we will
introduce such a model, together with an explanation of
reassignment statements and declaration assignments.
First, however, we examine some variations on the theme established by
#py("new_withdraw").

The following
function, #py("make_withdraw"),
creates "withdrawal processors."
The  parameter
#py("balance") in
#py("make_withdraw")
specifies the initial amount of money in the
account.#footnote[In contrast with
#py("make_withdraw_balance_100")
above, we do not have to use
a declaration assignment
to make #py("balance") a local variable, since

parameters are already
local. This will be clearer after the discussion of the environment
model of evaluation in
section @sec:environment-model.
(See also
exercise @ex:local-state-variable.)]<foot:make_withdraw>
#idx("makewithdraw", decl: true)
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

The function #py("make_withdraw")
can be used as follows to create two objects #py("W1")
and #py("W2"):

#snippet(```python
W1 = make_withdraw(100)
W2 = make_withdraw(100)
```)

#snippet(```python
print(W1(50))
```)

#output(```python
print(W1(50))
```)

#snippet(```python
print(W2(70))
```)

#output(```python
print(W2(70))
```)

#snippet(```python
print(W2(40))
```)

#output(```python
print(W2(40))
```)

#snippet(```python
print(W1(40))
```)

#output(```python
print(W1(40))
```)

Observe that #py("W1") and
#py("W2") are completely independent objects, each
with its own local state variable #py("balance").
Withdrawals from one do not affect the other.

We can also create objects that handle
#idx("deposit message for bank account", decl: true)
deposits as well as
withdrawals, and thus we can represent simple bank accounts. Here is
a
function
that returns a "bank-account object" with a specified initial
balance:
#idx("makeaccount", decl: true)
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
                else error("unknown request -- make_account", m))
    return dispatch
```)

Each call to #py("make_account") sets up an
environment with a local state variable #py("balance").
Within this environment, #py("make_account") defines
functions
#py("deposit") and
#py("withdraw") that access
#py("balance") and an additional
function
#py("dispatch")
that takes a "message" as input and returns one of the two local
functions.
The #py("dispatch")
function
itself is returned as the value that represents the bank-account object.
This is precisely the
#idx("message passing", sub: "in bank account")
#emph[message-passing] style of programming that we saw in
section @sec:data-directed, although here we are using
it in conjunction with the ability to modify local variables.

The function #py("make_account")
can be used as follows:

#snippet(```python
acc = make_account(100)
```)

#snippet(```python
print(acc("withdraw")(50))
```)

#output(```python
print(acc("withdraw")(50))
```)

#snippet(```python
print(acc("withdraw")(60))
```)

#output(```python
print(acc("withdraw")(60))
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

Each call to #py("acc") returns the locally defined
#py("deposit") or #py("withdraw")
function,
which is then applied to the specified #py("amount").
As was the case with
#py("make_withdraw"), another call to #py("make_account")

#snippet(```python
acc2 = make_account(100)
```)

will produce a completely separate account object, which maintains its
own local #py("balance").

#exercise(label-name: <ex:make-accumulator>, [
An
#idx("accumulator")
#emph[accumulator] is a
function
that is called repeatedly with a single numeric argument and accumulates its
arguments into a sum. Each time it is called, it returns the currently
accumulated sum. Write a
function
#idx("makeaccumulator")
#py("make_accumulator")
that generates accumulators, each maintaining an independent sum. The
input to
#py("make_accumulator")
should specify the initial value of the sum; for example

#snippet(```python
a = make_accumulator(5)
```)

#snippet(```python
print(a(10))
```)

#output(```python
print(a(10))
```)

#snippet(```python
print(a(10))
```)

#output(```python
print(a(10))
```)
])

#exercise(label-name: <ex:make-monitored>, [
In software-testing applications, it is useful to be able to count the
number of times a given
function
is called during the course of a computation. Write a
function
#idx("makemonitored")
#idx("monitored function")

#py("make_monitored")
that takes as input a
function,
#py("f"), that itself takes one input. The result
returned by
#py("make_monitored")
is a third
function,
say #py("mf"), that keeps track of the number of times
it has been called by maintaining an internal counter. If the input to
#py("mf") is the
string #py("\"how many calls\""),
then #py("mf") returns the value of the counter. If
the input is the
string #py("\"reset count\""),
then #py("mf") resets the counter to zero. For any
other input, #py("mf") returns the result of calling
#py("f") on that input and increments the counter.
For instance, we could make a monitored version of the
#py("sqrt")
function:

#snippet(```python
s = make_monitored(math_sqrt)
```)

#snippet(```python
print(s(100))
```)

#output(```python
print(s(100))
```)

#snippet(```python
print(s("how many calls"))
```)

#output(```python
print(s("how many calls"))
```)
])

#exercise(label-name: <ex:password-protection>, [
Modify the
#py("make_account")
function
so that it creates
#idx("bank account", sub: "password-protected")
#idx("password-protected bank account")
password-protected accounts. That is,
#py("make_account")
should take a
string
as an additional argument, as in

#snippet(```python
acc = make_account(100, "secret password")
```)

The resulting account object should process a request only if it is
accompanied by the password with which the account was created, and
should otherwise return a complaint:

#snippet(```python
print(acc("secret password", "withdraw")(40))
```)

#output(```python
print(acc("secret password", "withdraw")(40))
```)

#snippet(```python
print(acc("some other password", "deposit")(40))
```)

#output(```python
print(acc("some other password", "deposit")(40))
```)
])

#exercise(label-name: <ex:3_4>, [
Modify the
#py("make_account")
function
of exercise @ex:password-protection by adding another
local state variable so that, if an account is accessed more than seven
consecutive times with an incorrect password, it invokes the
function
#py("call_the_cops").
])

#idx("local state variable")
#idx("state variable", sub: "local")
