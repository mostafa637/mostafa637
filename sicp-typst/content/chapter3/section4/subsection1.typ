// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([The Nature of Time in Concurrent Systems], label-name: <sec:nature-of-time>)

#idx("time", sub: "in concurrent systems")

On the surface, time seems straightforward. It
is an ordering imposed on events.#footnote[To quote some graffiti seen
on a
building wall in Cambridge,
Massachusetts: "Time #idx("time", sub: "purpose of") is a device that was invented to keep everything from happening at once."]
For any events $A$ and
$B$,
either $A$ occurs before
$B$,
$A$ and
$B$ are simultaneous, or
$A$ occurs after
$B$. For instance,
returning to the bank account example, suppose that Peter withdraws
\$10 and Paul withdraws \$25 from a
#idx("bank account", sub: "joint, with concurrent access")
joint account that initially
contains \$100, leaving \$65 in the account. Depending on the
order of the two withdrawals, the sequence of balances in the account is
either $$100 arrow.r $90 arrow.r $65$
or $$100 arrow.r $75 arrow.r $65$.
In a computer implementation of the banking system, this changing
sequence of balances could be modeled by successive assignments to
a variable #py("balance").

In complex situations, however, such a view can be problematic.
Suppose that Peter and Paul, and other people besides, are
accessing the same bank account through a network of banking machines
distributed all over the world. The actual sequence of balances in
the account will depend critically on the detailed timing of the
accesses and the details of the communication among the machines.

This
#idx("order of events", sub: "indeterminacy in concurrent systems")
indeterminacy in the order of events can pose serious problems in
the design of concurrent systems. For instance, suppose that the
withdrawals made by Peter and Paul are implemented as two separate
threads
sharing a common variable #py("balance"), each
thread
specified by the
function
given in section @sec:local-state-variables:

#snippet(```python
def withdraw(amount):
    global balance
    if balance >= amount:
        balance = balance - amount
        return balance
    else:
        return "Insufficient funds"
```)

If the two
threads
operate independently, then Peter might test the
balance and attempt to withdraw a legitimate amount.
#idx("withdraw", sub: "problems in concurrent system")
However, Paul
might withdraw some funds in between the time that Peter checks the
balance and the time Peter completes the withdrawal, thus invalidating
Peter's test.

Things can be worse still. Consider the
statement

#snippet(```python
balance = balance - amount
```)

executed as part of each withdrawal process. This consists of three
steps: (1) accessing the value of the #py("balance")
variable; (2) computing the new balance; (3) setting
#py("balance") to this new value. If Peter and
Paul's withdrawals execute this statement concurrently, then the
two withdrawals might interleave the order in which they access
#py("balance") and set it to the new value.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-31.svg", width: 40%), caption: [Timing diagram #idx("timing diagram") showing how interleaving the order of events in two banking withdrawals can lead to an incorrect final balance.], label-name: <fig:bank-access>)

The timing diagram in
figure @fig:bank-access
depicts
an order of events where #py("balance") starts at 100,
Peter withdraws 10, Paul withdraws 25, and yet the final value of
#py("balance") is 75. As shown in the diagram,
the reason for this anomaly is that Paul's assignment of 75 to
#py("balance") is made under the assumption that
the value of #py("balance") to be decremented is 100.
That assumption, however, became invalid when Peter changed
#py("balance") to 90. This is a catastrophic
failure for the banking system, because the total amount of money in the
system is not conserved. Before the transactions, the total amount of
money was \$100\. Afterwards, Peter has \$10, Paul has
\$25, and the bank has \$75.#footnote[An even worse
failure for this system could occur if the two
assignments
attempt to change the balance simultaneously, in which case the actual data
appearing in memory might end up being a random combination of the
information being written by the two
threads.
Most computers have interlocks on
the primitive memory-write operations, which protect against such
simultaneous access. Even this seemingly simple kind of protection,
however, raises implementation challenges in the design of
multiprocessing computers, where elaborate
#idx("cache-coherence protocols")
#emph[cache-coherence]
protocols are required to ensure that the various processors will
maintain a consistent view of memory contents, despite the fact that
data may be replicated ("cached") among the different
processors to increase the speed of memory access.]

The general phenomenon illustrated
here is that several
threads
may
#idx("state", sub: "shared")
#idx("shared state")
share a common state variable. What makes this complicated is that more than one
thread
may be trying to manipulate the shared state at the same
time. For the bank account example, during each transaction, each
customer should be able to act as if the other customers did not
exist. When
customers change
the balance in a way that depends on
the balance,
they
must be able to assume that, just before the moment of
change, the balance is still what
they
thought it was.

#subheading([Correct behavior of concurrent programs])

#idx("concurrency", sub: "correctness of concurrent programs")

The above example typifies the subtle bugs that can creep into
concurrent programs. The root of this complexity lies in the
assignments to variables that are shared among the different
threads.
We already know that we must be careful in writing programs that use
assignment,
because the results of a computation depend on the order in which the
assignments occur.#footnote[The factorial program in
section @sec:costs-of-assignment illustrates this for
a single sequential
thread.]
With concurrent
threads
we must be especially careful about
assignments, because we may not be able to control the order of the
assignments made by the different
threads.
If several such changes
might be made concurrently (as with two depositors accessing a joint
account) we need some way to ensure that our system behaves correctly.
For example, in the case of withdrawals from a joint bank account, we
must ensure that money is conserved.
To make concurrent programs behave correctly, we may have to
place some restrictions on concurrent execution.

One possible restriction on concurrency would stipulate that no two
operations that change any shared state variables can occur at the same
time. This is an extremely stringent requirement. For distributed banking,
it would require the system designer to ensure that only one transaction
could proceed at a time. This would be both inefficient and overly
conservative. Figure @fig:two-shared-accounts shows
Peter and Paul sharing a bank account, where Paul has a private account
as well. The diagram illustrates two withdrawals from the shared account
(one by Peter and one by Paul) and a deposit to Paul's private
account.#footnote[The columns show the contents of Peter's wallet,
the joint account (in Bank1), Paul's wallet, and Paul's
private account (in Bank2), before and after each withdrawal (W) and
deposit (D). Peter withdraws \$10 from Bank1; Paul deposits
\$5 in Bank2, then withdraws \$25 from Bank1.]
The two withdrawals from the shared account must not be concurrent (since
both access and update the same account), and Paul's deposit and
withdrawal must not be concurrent (since both access and update the amount
in Paul's wallet). But there should be no problem permitting
Paul's deposit to his private account to proceed concurrently with
Peter's withdrawal from the shared account.

#sicp-figure(image("/images/img_original/Fig3.30.svg", width: 70%), caption: [Concurrent deposits and withdrawals from a joint account in Bank1 and a private account in Bank2.], label-name: <fig:two-shared-accounts>)

A less stringent restriction on concurrency would ensure that a
concurrent system produces the same result as if the
threads
had run sequentially in some order. There are two important aspects to this
requirement. First, it does not require the
threads
to actually run sequentially, but only to produce results that are the same
#emph[as if] they had run sequentially. For the example in
figure @fig:two-shared-accounts, the designer of the
bank account system can safely allow Paul's deposit and Peter's
withdrawal to happen concurrently, because the net result will be the same as
if the two operations had happened sequentially. Second, there may be more
than one possible "correct" result produced by a concurrent
program, because we require only that the result be the same as for
#emph[some] sequential order. For example, suppose that Peter and
Paul's joint account starts out with \$100, and Peter deposits
\$40 while Paul concurrently withdraws half the money in the account.
Then sequential execution could result in the account balance being either
\$70 or \$90 (see
exercise @ex:joint-account).#footnote[A more formal way
to express this idea is to say that concurrent programs are inherently
#idx("nondeterminism, in behavior of concurrent programs")
#emph[nondeterministic]. That is, they are described not by single-valued
functions, but by functions whose results are sets of possible values.
In section @sec:nondeterministic-evaluation we will
study a language for expressing nondeterministic computations.]<foot:nondeterministic>

There are still weaker requirements for correct execution of concurrent
programs. A program for simulating
#idx("diffusion, simulation of")
diffusion (say, the flow of heat in an object) might consist of a large
number of
threads,
each one representing a small volume of space, that update their values
concurrently. Each
thread
repeatedly changes its value to the average of its own value and its
neighbors' values. This algorithm converges to the right answer
independent of the order in which the operations are done; there is no
need for any restrictions on concurrent use of the shared values.

#exercise(label-name: <ex:joint-account>, [
Suppose that
#idx("Peter, Paul and Mary")
Peter, Paul, and Mary share a joint bank account that
initially contains \$100\. Concurrently, Peter deposits
\$10, Paul withdraws \$20, and Mary withdraws half the
money in the account, by executing the following commands:

#sicp-table(columns: 2, [Peter:], [#py("balance = balance + 10")], [Paul:], [#py("balance = balance - 20")], [Mary:], [#py("balance = balance - (balance / 2)")])

+ List all the different possible values for #py("balance") after these three transactions have been completed, assuming that the banking system forces the three threads to run sequentially in some order.
+ What are some other values that could be produced if the system allows the threads to be interleaved? Draw timing diagrams like the one in figure @fig:bank-access to explain how these values can occur.
])

#idx("time", sub: "in concurrent systems")
#idx("concurrency", sub: "correctness of concurrent programs")
