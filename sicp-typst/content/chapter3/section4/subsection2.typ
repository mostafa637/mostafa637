// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Mechanisms for Controlling Concurrency])

#idx("concurrency", sub: "mechanisms for controlling")

We've seen that the difficulty in dealing with concurrent
threads
is rooted in the need to consider the interleaving of the order of events
in the different
threads.
For example, suppose we have two
threads,
one with three ordered events $(a,b,c)$
and one with three ordered events $(x,y,z)$.
If the two
threads
run concurrently, with no constraints on how their execution is
interleaved, then there are 20 different possible orderings for the events
that are consistent with the individual orderings for the two
threads:

$ mat(delim: #none, (a,b,c,x,y,z), (a,x,b,y,c,z), (x,a,b,c,y,z), (x,a,y,z,b,c); (a,b,x,c,y,z), (a,x,b,y,z,c), (x,a,b,y,c,z), (x,y,a,b,c,z); (a,b,x,y,c,z), (a,x,y,b,c,z), (x,a,b,y,z,c), (x,y,a,b,z,c); (a,b,x,y,z,c), (a,x,y,b,z,c), (x,a,y,b,c,z), (x,y,a,z,b,c); (a,x,b,c,y,z), (a,x,y,z,b,c), (x,a,y,b,z,c), (x,y,z,a,b,c)) $

As programmers designing this system, we would have to consider the
effects of each of these 20 orderings and check that each behavior is
acceptable. Such an approach rapidly becomes unwieldy as the numbers of
threads and events increase.

A more practical approach to the design of concurrent systems is to
devise general mechanisms that allow us to constrain the interleaving
of concurrent
threads
so that we can be sure that the program
behavior is correct. Many mechanisms have been developed for this
purpose. In this section, we describe one of them, the #emph[serializer].

#subheading([Serializing access to shared state])

#idx("serializer")

Serialization implements the following idea:
Threads
will execute concurrently, but there will be certain collections of
functions
that cannot be executed concurrently. More precisely, serialization
creates distinguished sets of
functions
such that only one execution of a
function
in each serialized set is permitted to happen at a time. If some
function
in the set is being executed, then a
thread
that attempts to execute any
function
in the set will be forced to wait
until the first execution has finished.

We can use serialization to control access to shared variables.
For example, if we want to update a shared variable based on the
previous value of that variable, we put the access to the previous
value of the variable and the assignment of the new value to the
variable in the same
function.
We then ensure that no other
function
that assigns to the variable can run concurrently with this
function
by serializing all of these
functions
with the same serializer. This guarantees that the value of the
variable cannot be changed between an access and the corresponding
assignment.

#subheading([Serializers])

To make the above mechanism more concrete, suppose that we have
extended
Python
to include a
function
called
#idx("concurrentexecute") #py("concurrent_execute"):

#syntax("
concurrent_execute(", $f_(1)$, ", ", $f_(2)$, ", ", $dots.h$, ", ", $f_(k)$, ")
      ")

Each $f$ must be a function of no arguments. The function #py("concurrent_execute") creates a separate thread for each $f$, which applies $f$ (to no arguments).
These
threads
all run concurrently.#footnote[The function #py("concurrent_execute") is
not part of the Python standard, but the examples in this section
can be implemented in ECMAScript 2020.]

As an example of how this is used, consider

#snippet(```python
x = 10

def t1():
    global x
    x = x * x

def t2():
    global x
    x = x + 1

concurrent_execute(t1, t2)
```)

This creates two concurrent
threads—$T_(1)$, which sets
#py("x") to #py("x") times
#py("x"), and $T_(2)$,
which increments #py("x"). After execution is
complete, #py("x") will be left with one of five
possible values, depending on the interleaving of the events of
$T_(1)$ and $T_(2)$:

#sicp-table(columns: 2, [101:], [$T_(1)$ sets #py("x") to 100 and then $T_(2)$ increments #py("x") to 101.], [121:], [$T_(2)$ increments #py("x") to 11 and then $T_(1)$ sets #py("x") to #py("x") times #py("x").], [110:], [$T_(2)$ changes #py("x") from 10 to 11 between the two times that $T_(1)$], [], [accesses the value of #py("x") during the evaluation of #py("x * x").], [11:], [$T_(2)$ accesses #py("x"), then $T_(1)$ sets #py("x") to 100, then $T_(2)$ sets #py("x").], [100:], [$T_(1)$ accesses #py("x") (twice), then $T_(2)$ sets #py("x") to 11, then $T_(1)$ sets #py("x").])

We can constrain the concurrency by using serialized
functions,
which are created by #emph[serializers]. Serializers are constructed by
#py("make_serializer"),
whose implementation is given below. A serializer takes a
function
as argument and returns a serialized
function
that behaves like the original
function.
All calls to a given serializer return serialized
functions
in the same set.

Thus, in contrast to the example above, executing

#snippet(```python
x = 10

s = make_serializer()

def t1():
    global x
    x = x * x

def t2():
    global x
    x = x + 1

concurrent_execute(s(t1), s(t2))
```)

can produce only two possible values for
#py("x"), 101 or 121.
The other possibilities are eliminated, because the execution of
$T_(1)$ and $T_(2)$ cannot be interleaved.

Here is a version of the
#py("make_account")
function
from section @sec:local-state-variables,
where the deposits and withdrawals have been
#idx("bank account", sub: "serialized")
serialized:

#idx("makeaccount", sub: "with serialization", decl: true)
#snippet(```python
def make_account(balance):
    def withdraw(amount):
        nonlocal balance
        if balance > amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    def deposit(amount):
        nonlocal balance
        balance = balance + amount
        return balance
    protect = make_serializer()
    def dispatch(m):
        return (protect(withdraw) if m == "withdraw"
                else protect(deposit) if m == "deposit"
                else balance if m == "balance"
                else error("unknown request -- make_account", m))
    return dispatch
```)

With this implementation, two
threads
cannot be withdrawing from or
depositing into a single account concurrently. This eliminates the source
of the error illustrated in figure @fig:bank-access,
where Peter changes the account balance between the times when Paul accesses
the balance to compute the new value and when Paul actually performs the
assignment. On the other hand, each account has its own serializer,
so that deposits and withdrawals for different accounts can proceed
concurrently.

#exercise(label-name: <ex:3_39>, [
Which of the five possibilities in the
concurrent
execution shown above remain if we instead serialize execution as follows:

#snippet(```python
x = 10

s = make_serializer()

def t1():
    global x
    x = s(lambda: x * x)()

def t2():
    global x
    x = x + 1

concurrent_execute(t1, s(t2))
```)
])

#exercise(label-name: <ex:3_40>, [
Give all possible values of #py("x")
that can result from executing

#snippet(```python
x = 10

def t1():
    global x
    x = x * x

def t2():
    global x
    x = x * x * x

concurrent_execute(t1, t2)
```)

 Which of these possibilities remain if we instead use serialized
functions:

#snippet(```python
x = 10

s = make_serializer()

def t1():
    global x
    x = x * x

def t2():
    global x
    x = x * x * x

concurrent_execute(s(t1), s(t2))
```)
])

#exercise(label-name: <ex:bensconcern>, [
Ben Bitdiddle worries that it would be better to implement the bank
account as follows (where the commented line has been changed):
#idx("makeaccount", sub: "with serialization", decl: true)
#snippet(```python
def make_account(balance):
    def withdraw(amount):
        nonlocal balance
        if balance > amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    def deposit(amount):
        nonlocal balance
        balance = balance + amount
        return balance
    protect = make_serializer()
    def dispatch(m):
        return (protect(withdraw) if m == "withdraw"
                else protect(deposit) if m == "deposit"
                else protect(lambda: balance)() if m == "balance"  # serialized
                else error("unknown request -- make_account", m))
    return dispatch
```)

because allowing unserialized access to the bank balance can
result in anomalous behavior. Do you agree? Is there any
scenario that demonstrates Ben's concern?
])

#exercise(label-name: <ex:3_42>, [
Ben Bitdiddle suggests that it's a waste of time to
create a new serialized
function
in response to every #py("withdraw") and
#py("deposit") message. He says that
#py("make_account")
could be changed so that the calls to
#py("protect")
are done outside the #py("dispatch")
function.
That is, an account would return the same serialized
function
(which was created at the same time as the account) each time
it is asked for a withdrawal
function.
#idx("makeaccount", sub: "with serialization", decl: true)
#snippet(```python
def make_account(balance):
    def withdraw(amount):
        nonlocal balance
        if balance > amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    def deposit(amount):
        nonlocal balance
        balance = balance + amount
        return balance
    protect = make_serializer()
    protect_withdraw = protect(withdraw)
    protect_deposit = protect(deposit)
    def dispatch(m):
        return (protect_withdraw if m == "withdraw"
                else protect_deposit if m == "deposit"
                else balance if m == "balance"
                else error("unknown request -- make_account", m))
    return dispatch
```)

Is this a safe change to make? In particular, is there any difference
in what concurrency is allowed by these two versions of
#py("make_account")?
])

#idx("serializer")

#subheading([Complexity of using multiple shared resources])

#idx("serializer", sub: "with multiple shared resources")
#idx("shared resources")

Serializers provide a powerful abstraction that helps isolate the
complexities of concurrent programs so that they can be dealt with
carefully and (hopefully) correctly. However, while using serializers
is relatively straightforward when there is only a single shared
resource (such as a single bank account), concurrent programming can
be treacherously difficult when there are multiple shared resources.

To illustrate one of the difficulties that can arise, suppose we wish to
#idx("bank account", sub: "exchanging balances")
swap the balances in two bank accounts. We access each account to find
the balance, compute the difference between the balances, withdraw this
difference from one account, and deposit it in the other account.
We could implement this as
follows:#footnote[We have simplified #py("exchange")
by exploiting the fact that our #py("deposit")
message accepts negative amounts. (This is a serious bug in our banking
system!)]

#idx("exchange", decl: true)
#snippet(```python
def exchange(account1, account2):
    difference = account1("balance") - account2("balance")
    account1("withdraw")(difference)
    account2("deposit")(difference)
```)

This
function
works well when only a single
thread
is trying to do the exchange. Suppose, however, that Peter and Paul both
have access to accounts $a_(1)$,
$a_(2)$, and $a_(3)$, and
that Peter exchanges $a_(1)$ and
$a_(2)$ while Paul concurrently exchanges
$a_(1)$ and $a_(3)$.
Even with account deposits and withdrawals
serialized for individual accounts (as in the
#py("make_account")
function
shown above in this section), #py("exchange") can
still produce incorrect results. For example, Peter might compute the
difference in the balances for $a_(1)$ and
$a_(2)$, but then Paul might change the balance in
$a_(1)$ before Peter is able to complete the
exchange.#footnote[If the account balances start out as \$10,
\$20, and \$30, then after any number of concurrent exchanges,
the balances should still be \$10, \$20, and \$30 in
some order. Serializing the deposits to individual accounts is not
sufficient to guarantee this. See
exercise @ex:exchange-bug.]
For correct behavior, we must arrange for the
#py("exchange")
function
to lock out any other concurrent accesses to the accounts during the
entire time of the exchange.

One way we can accomplish this is by using both accounts' serializers
to serialize the entire #py("exchange")
function.
To do this, we will arrange for access to an account's serializer.
Note that we are deliberately breaking the modularity of the bank-account
object by exposing the serializer. The following version of
#py("make_account")
is identical to the original version given in
section @sec:local-state-variables, except that a
serializer is provided to protect the balance variable, and the serializer
is exported via message passing:
#idx("makeaccountandserializer", decl: true)
#snippet(```python
def make_account_and_serializer(balance):
    def withdraw(amount):
        nonlocal balance
        if balance > amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    def deposit(amount):
        nonlocal balance
        balance = balance + amount
        return balance
    balance_serializer = make_serializer()
    return lambda m: (withdraw if m == "withdraw"
                      else deposit if m == "deposit"
                      else balance if m == "balance"
                      else balance_serializer if m == "serializer"
                      else error("unknown request -- make_account", m))
```)

We can use this to do serialized deposits and withdrawals. However,
unlike our earlier serialized account, it is now the responsibility of
each user of bank-account objects to explicitly manage the
serialization, for example as
follows:#footnote[Exercise @ex:export-serializer
investigates why deposits and withdrawals are no longer automatically
serialized by the account.]

#idx("deposit, with external serializer", decl: true)
#snippet(```python
def deposit(account, amount):
    s = account("serializer")
    d = account("deposit")
    s(d(amount))
```)

Exporting the serializer in this way gives us enough flexibility to
implement a serialized exchange program. We simply serialize the original
#py("exchange")
function
with the serializers for both accounts:

#idx("serializedexchange", decl: true)
#snippet(```python
def serialized_exchange(account1, account2):
    serializer1 = account1("serializer")
    serializer2 = account2("serializer")
    serializer1(serializer2(exchange))(account1, account2)
```)

#exercise(label-name: <ex:exchange-bug>, [
Suppose that the balances in three accounts start out as \$10,
\$20, and \$30, and that multiple
threads
run, exchanging the balances in the accounts. Argue that if the
threads
are run sequentially,
after any number of concurrent exchanges, the account balances should be
\$10, \$20, and \$30 in some order.
Draw a timing diagram like the one in
figure @fig:bank-access to
show how this condition can be violated if the exchanges are
implemented using the first version of the account-exchange program in
this section. On the other hand, argue that even with this
#py("exchange") program, the sum of the balances
in the accounts will be preserved. Draw a timing diagram to show how
even this condition would be violated if we did not serialize the
transactions on individual accounts.
])

#exercise(label-name: <ex:3_44>, [
Consider the problem of
#idx("bank account", sub: "transferring money")
transferring an amount from one account to
another. Ben Bitdiddle claims that this can be accomplished with the
following
function,
even if there are multiple people concurrently
transferring money among multiple accounts, using any account
mechanism that serializes deposit and withdrawal transactions, for
example, the version of
#py("make_account")
in the text above.

#snippet(```python
def transfer(from_account, to_account, amount):
    from_account("withdraw")(amount)
    to_account("deposit")(amount)
```)

Louis Reasoner claims that there is a problem here, and that we need
to use a more sophisticated method, such as the one required for
dealing with the exchange problem. Is Louis right? If not, what is
the essential difference between the transfer problem and the exchange
problem? (You should assume that the balance in
#py("from_account")
is at least #py("amount").)
])

#exercise(label-name: <ex:export-serializer>, [
Louis Reasoner thinks our bank-account system is unnecessarily complex
and error-prone now that deposits and withdrawals aren't
automatically serialized. He suggests that
#py("make_account_and_serializer")
should have exported the serializer
(for use by such functions as #py("serialized_exchange"))
in addition to (rather than instead of) using it to serialize accounts and
deposits as
#py("make_account")
did. He proposes to redefine accounts as follows:

#snippet(```python
def make_account_and_serializer(balance):
    def withdraw(amount):
        nonlocal balance
        if balance > amount:
            balance = balance - amount
            return balance
        else:
            return "Insufficient funds"
    def deposit(amount):
        nonlocal balance
        balance = balance + amount
        return balance
    balance_serializer = make_serializer()
    return lambda m: (balance_serializer(withdraw) if m == "withdraw"
                      else balance_serializer(deposit) if m == "deposit"
                      else balance if m == "balance"
                      else balance_serializer if m == "serializer"
                      else error("unknown request -- make_account", m))
```)

Then deposits are handled as with the original
#py("make_account"):

#snippet(```python
def deposit(account, amount):
    account("deposit")(amount)
```)

Explain what is wrong with Louis's reasoning. In particular,
consider what happens when
#py("serialized_exchange")
is called.
])

#idx("serializer", sub: "with multiple shared resources")
#idx("shared resources")

#subheading([Implementing serializers])

#idx("serializer", sub: "implementing")

We implement serializers in terms of a more primitive synchronization
mechanism called a
#idx("mutex")
#emph[mutex]. A mutex is an object that supports two
operations—the mutex can be
#idx("acquire a mutex")
#emph[acquired], and the mutex can be
#idx("release a mutex")
#emph[released]. Once a mutex has been acquired, no other acquire
operations on that mutex may proceed until the mutex is
released.#footnote[The term "mutex" is an abbreviation for
#idx("mutual exclusion")
#emph[mutual exclusion]. The general problem of arranging a mechanism
that permits concurrent
threads
to safely share resources is called the mutual exclusion problem. Our
mutex is a simple variant of the
#idx("semaphore")
#emph[semaphore] mechanism (see
exercise @ex:semaphore), which was introduced in the
#idx("THE Multiprogramming System")
"THE" Multiprogramming System developed at the
#idx("Technological University of Eindhoven")
Technological University of Eindhoven and named for the university's
initials in Dutch
#idx("Dijkstra, Edsger Wybe")
(Dijkstra 1968a). The acquire and
release operations were originally called
#idx("P operation on semaphore", sort: "P")
#idx("V operation on semaphore", sort: "V")
P and V, from the Dutch
words #emph[passeren] (to pass) and #emph[vrijgeven] (to release), in
reference to the semaphores used on railroad systems. Dijkstra's
classic exposition (1968b) was one of the first to clearly present the
issues of concurrency control, and showed how to use semaphores to
handle a variety of concurrency problems.]
In our implementation, each serializer has an associated mutex. Given a
function #py("f"),
the serializer returns a
function
that acquires the mutex, runs
#py("f"),
and then releases the mutex. This ensures that only one of the
functions
produced by the serializer can be running at once, which is
precisely the serialization property that we need to guarantee.

To apply serializers to functions that take an arbitrary number of arguments,
we use Python's #emph[rest] parameter and #emph[spread] syntax.
#idx("... (rest parameter and spread syntax)", sort: "0a3")
#idx("vector (data structure)", sub: "used in spread and rest parameter syntax")
#idx("rest parameter and spread syntax")
#idx("spread and rest parameter syntax")

#idx("argument(s)", sub: "arbitrary number of")
The #py("...") in front of the
parameter #py("args") collects
the rest (here all) of the arguments of any call of the function
into a #emph[vector] data structure.
The
#py("...") in front of
#py("args") in
the application
#py("f(...args)")
spreads the elements of
#py("args") so that
they become separate arguments of
#py("f").

#idx("makeserializer", decl: true)
#snippet(```python
def make_serializer():
    mutex = make_mutex()
    def serializer(f):
        def serialized_f(*args):
            mutex("acquire")
            val = f(*args)
            mutex("release")
            return val
        return serialized_f
    return serializer
```)

The mutex is a mutable object (here we'll use a one-element list,
which we'll refer to as a
#idx("cell, in serializer implementation")
#emph[cell]) that can hold the value true or false. When the value is
false, the mutex is available to be acquired. When the value is true, the
mutex is unavailable, and any
thread
that attempts to acquire the mutex must wait.

Our mutex constructor
#py("make_mutex")
begins by initializing the cell contents to false. To acquire the mutex,
we test the cell. If the mutex is available, we set the cell contents to
true and proceed. Otherwise, we wait in a loop, attempting to acquire over
and over again, until we find that the mutex is available.#footnote[In most
time-shared operating systems,
threads
that are
#idx("blocked process")
blocked by a mutex do
not waste time
#idx("busy-waiting")
"busy-waiting" as above. Instead, the system
schedules another
thread
to run while the first is waiting, and the blocked
thread
is awakened when the mutex becomes available.]
To release the mutex, we set the cell contents to false.
#idx("makemutex", decl: true)
#snippet(```python
def make_mutex():
    cell = llist(False)
    def the_mutex(m):
        return ((the_mutex("acquire")  # retry
                 if test_and_set(cell)
                 else True)
                if m == "acquire"
                else clear(cell) if m == "release"
                else error("unknown request -- mutex", m))
    return the_mutex

def clear(cell):
    set_head(cell, False)
```)

The function #py("test_and_set")
tests the cell and returns the result of the test. In addition, if the
test was false,
#py("test_and_set")
sets the cell contents to true before returning false. We can express this
behavior as the following
function:
#idx("testandset", decl: true)
#snippet(```python
def test_and_set(cell):
    if head(cell):
        return True
    else:
        set_head(cell, True)
        return False
```)

However, this implementation of
#py("test_and_set")
does not suffice as it stands. There is a crucial subtlety here, which is
the essential place where concurrency control enters the system: The
#py("test_and_set")
operation must be performed
#idx("atomic requirement for testandset")
#emph[atomically]. That is, we must guarantee that, once a
thread
has tested the cell and found it to be false, the cell contents will
actually be set to true before any other
thread
can test the cell. If we do not make this guarantee, then the mutex can
fail in a way similar to the bank-account failure in
figure @fig:bank-access. (See
exercise @ex:atomic-test-and-set.)

The actual implementation of
#py("test_and_set")
depends on the details of how our system runs concurrent
threads.
For example, we might be executing concurrent
threads
on a sequential processor using a
#idx("time slicing")
time-slicing mechanism that cycles through the
threads,
permitting each
thread
to run for a short time before interrupting it
and moving on to the next
thread.
In that case,
#py("test_and_set")
can work by disabling time slicing during the testing and
setting.
Alternatively, multiprocessing computers provide instructions that
support atomic operations directly in hardware.#footnote[There are many
variants of such
#idx("atomic operations supported in hardware")
instructions—including test-and-set, test-and-clear, swap,
compare-and-exchange, load-reserve, and store-conditional—whose
design must be carefully matched to the machine's
processor–memory interface. One issue that arises here is to
determine what happens if two
threads
attempt to acquire the same resource at exactly the same time by using such
an instruction. This requires some mechanism for making a decision about
which
thread
gets control. Such a mechanism is called an
#idx("arbiter")
#emph[arbiter]. Arbiters usually boil down to some sort of hardware
device. Unfortunately, it is possible to prove that one cannot physically
construct a fair arbiter that works 100% of the time unless one
allows the arbiter an arbitrarily long time to make its decision.
The fundamental phenomenon here was originally observed by the
fourteenth-century French philosopher
#idx("Buridan, Jean")
Jean Buridan in his commentary on
#idx("Aristotle's De caelo (Buridan's commentary on)")
Aristotle's #emph[De caelo]. Buridan argued that a perfectly rational
#idx("dog, perfectly rational behavior of")
dog placed between two equally attractive sources of food will starve to
death, because it is incapable of deciding which to go to first.]

#exercise(label-name: <ex:atomic-test-and-set>, [
Suppose that we implement
#py("test_and_set")
using an ordinary
function
as shown in the text, without attempting to make the operation atomic.
Draw a timing diagram like the one in
figure @fig:bank-access to demonstrate how the mutex
implementation can fail by allowing two
threads
to acquire the mutex at the same time.
])

#exercise(label-name: <ex:semaphore>, [
A semaphore
#idx("semaphore", sub: "of size n")
(of size $n$) is a generalization of
a mutex. Like a mutex, a semaphore supports acquire and release operations,
but it is more general in that up to $n$
threads
can acquire it
concurrently. Additional
threads
that attempt to acquire the semaphore must wait for release operations.
Give implementations of semaphores

+ in terms of mutexes
+ in terms of atomic #py("test_and_set") operations.
])

#idx("serializer", sub: "implementing")

#subheading([Deadlock])

#idx("concurrency", sub: "deadlock")
#idx("deadlock")

Now that we have seen how to implement serializers, we can see
that account exchanging still has a problem, even with the
#py("serialized_exchange")
function
above.
Imagine that Peter attempts to exchange $a_(1)$
with $a_(2)$ while Paul concurrently attempts to
exchange $a_(2)$ with
$a_(1)$. Suppose that Peter's
thread
reaches the point where it has entered a serialized
function
protecting $a_(1)$ and, just after that,
Paul's
thread
enters a serialized
function
protecting $a_(2)$. Now Peter cannot proceed (to
enter a serialized
function
protecting $a_(2)$) until Paul exits the serialized
function
protecting $a_(2)$. Similarly, Paul cannot proceed
until Peter exits the serialized
function
protecting $a_(1)$. Each
thread
is stalled forever, waiting for the other. This situation is called a
#emph[deadlock]. Deadlock is always a danger in systems that provide
concurrent access to multiple shared resources.

One way to avoid the
#idx("deadlock", sub: "avoidance")
deadlock in this situation is to give each account a
unique identification number and rewrite
#py("serialized_exchange")
so that a
thread
will always attempt to enter a
function
protecting the lowest-numbered account first. Although this method works
well for the exchange problem, there are other situations that require more
sophisticated deadlock-avoidance techniques, or where deadlock cannot
be avoided at all. (See exercises @ex:deadlock-avoid
and @ex:deadlock-cannot-avoid.)#footnote[The general
technique for avoiding
#idx("deadlock", sub: "recovery")
deadlock by numbering the
shared resources and acquiring them in order is due to
#idx("Havender, J.")
Havender (1968). Situations where deadlock cannot be
avoided require #emph[deadlock-recovery] methods, which entail having
threads
"back out" of the deadlocked state and try again.
Deadlock-recovery mechanisms are widely used in
data-base-management systems, a topic that is treated in detail in
#idx("Gray, Jim")
#idx("Reuter, Andreas")
Gray and Reuter 1993.]

#exercise(label-name: <ex:deadlock-avoid>, [
Explain in detail why the
#idx("serializedexchange", sub: "with deadlock avoidance")
deadlock-avoidance method described above,
(i.e., the accounts are numbered, and each
thread
attempts to acquire the smaller-numbered account first) avoids
deadlock in the exchange problem. Rewrite
#py("serialized_exchange")
to incorporate this idea. (You will also need to modify
#py("make_account")
so that each account is created with a number, which can be accessed by
sending an appropriate message.)
])

#exercise(label-name: <ex:deadlock-cannot-avoid>, [
Give a scenario where the deadlock-avoidance mechanism described
above does not work. (Hint: In the exchange problem, each
thread
knows in advance which accounts it will need to get access to. Consider a
situation where a
thread
must get access to some shared resources before it can know which additional
shared resources it will require.)
])

#idx("concurrency", sub: "deadlock")
#idx("deadlock")

#subheading([Concurrency, time, and communication])

We've seen how programming concurrent systems requires controlling
the ordering of events when different
threads
access shared state, and we've seen how to achieve this control
through judicious use of serializers. But the problems of concurrency
lie deeper than this, because, from a fundamental point of view, it's
not always clear what is meant by "shared state."

Mechanisms such as
#py("test_and_set")
require
threads
to examine a global shared flag at arbitrary times. This is problematic
and inefficient to implement in modern high-speed processors, where
due to optimization techniques such as pipelining and cached memory,
the contents of memory may not be in a consistent state at every instant.
In
some
multiprocessing systems, therefore, the serializer paradigm
is being supplanted by
other
approaches to concurrency
control.#footnote[One such alternative to serialization is called
#idx("barrier synchronization")
#emph[barrier synchronization]. The programmer permits concurrent
threads
to execute as they please, but establishes certain synchronization points
("barriers") through which no
thread
can proceed until all the
threads
have reached the barrier.
Some
processors provide machine instructions
that permit programmers to establish synchronization points at places where
consistency is required. The
#idx("PowerPC")
PowerPC$""^(upright("TM"))$, for example, includes
for this purpose two instructions called
#idx("SYNC")
SYNC and
#idx("EIEIO")
EIEIO (Enforced In-order Execution of Input/Output).]

The problematic aspects of shared state also arise in large, distributed
systems. For instance, imagine a distributed banking system where
individual branch banks maintain local values for bank balances and
periodically compare these with values maintained by other branches. In
such a system the value of "the account balance" would be
undetermined, except right after synchronization. If Peter deposits money
in an account he holds jointly with Paul, when should we say that the
account balance has changed—when the balance in the local branch
changes, or not until after the synchronization? And if Paul accesses the
account from a different branch, what are the reasonable constraints to
place on the banking system such that the behavior is
"correct"? The only thing that might matter for correctness
is the behavior observed by Peter and Paul individually and the
"state" of the account immediately after synchronization.
Questions about the "real" account balance or the order of
events between synchronizations may be irrelevant or
meaningless.#footnote[This may seem like a strange point of view, but there
are
systems that work this way.
#idx("credit-card accounts, international")
International charges to credit-card accounts,
for example, are normally cleared on a per-country basis, and the charges
made in different countries are periodically reconciled. Thus the account
balance may be different in different countries.]

The basic phenomenon here is that synchronizing different
threads,
establishing shared state, or imposing an order on events requires
communication among the
threads.
#idx("time", sub: "communication and")
In essence, any notion of time in concurrency control must be intimately
tied to communication.#footnote[For distributed systems, this perspective
was pursued by
#idx("Lamport, Leslie")
Lamport (1978), who showed how to use communication to establish
"global clocks" that can be used to establish orderings on
events in distributed systems.] It is intriguing that a similar
connection between time and communication also arises in the
#idx("relativity, theory of")
Theory of Relativity, where the speed of light (the fastest signal that can
be used to synchronize events) is a fundamental constant relating time and
space. The complexities we encounter in dealing with time and state in our
computational models may in fact mirror a fundamental complexity of
the physical universe.

#idx("concurrency")
#idx("concurrency", sub: "mechanisms for controlling")
