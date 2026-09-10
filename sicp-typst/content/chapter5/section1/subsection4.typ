// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Using a Stack to Implement Recursion], label-name: <sec:stack-recursion>)

#idx("stack", sub: "for recursion in register machine")
#idx("register machine", sub: "stack")
#idx("recursive process", sub: "register machine for")

With the ideas illustrated so far, we can implement any
#idx("iterative process", sub: "register machine for")
iterative
process by specifying a register machine that has a register
corresponding to each state variable of the process. The machine
repeatedly executes a controller loop, changing the contents
of the registers, until some termination condition is satisfied. At
each point in the controller sequence, the state of the machine
(representing the state of the iterative process) is completely
determined by the contents of the registers (the values of the state
variables).

Implementing
#idx("recursive process", sub: "iterative process vs.")
#idx("iterative process", sub: "recursive process vs.")
#idx("factorial", sub: "register machine for (recursive)")
recursive processes, however, requires an additional
mechanism. Consider the following recursive method for computing
factorials, which we first examined in
section @sec:recursion-and-iteration:

#snippet(```python
def factorial(n):
    return 1 if n == 1 else n * factorial(n - 1)
```)

As we see from the
function,
computing $n!$ requires computing
$(n-1)!$. Our GCD machine, modeled on the
function

#snippet(```python
def gcd(a, b):
    return a if b == 0 else gcd(b, a % b)
```)

similarly had to compute another GCD. But there is an important
difference between the #py("gcd")
function,
which reduces the original computation to a new GCD computation, and
#py("factorial"), which requires computing another
factorial as a subproblem. In GCD, the answer to the new GCD computation is
the answer to the original problem. To compute the next GCD, we simply
place the new arguments in the input registers of the GCD machine and reuse
the machine's data paths by executing the same controller sequence.
When the machine is finished solving the final GCD problem, it has completed
the entire computation.

In the case of factorial (or any recursive process) the answer to the
new factorial subproblem is not the answer to the original problem.
The value obtained for $(n-1)!$ must be
multiplied by $n$ to get the final answer. If
we try to imitate the GCD design, and solve the factorial subproblem by
decrementing the #py("n") register and rerunning the
factorial machine, we will no longer have available the old value of
#py("n") by which to multiply the result. We thus
need a second factorial machine to work on the subproblem. This second
factorial computation itself has a factorial subproblem, which
requires a third factorial machine, and so on. Since each factorial
machine contains another factorial machine within it, the total
machine contains an infinite nest of similar machines and hence cannot
be constructed from a fixed, finite number of parts.

Nevertheless, we can implement the factorial process as a register
machine if we can arrange to use the same components for each nested
instance of the machine. Specifically, the machine that computes
$n!$
should use the same components to work on the subproblem of computing
$(n-1)!$, on the subproblem for
$(n-2)!$, and so on. This is
plausible because, although the factorial process dictates that an
unbounded number of copies of the same machine are needed to perform a
computation, only one of these copies needs to be active at any given
time. When the machine encounters a recursive subproblem, it can
suspend work on the main problem, reuse the same physical parts to
work on the subproblem, then continue the suspended computation.

In the subproblem, the contents of the registers will be different
than they were in the main problem. (In this case the
#py("n") register is decremented.) In order to be
able to continue the suspended computation, the machine must save the
contents of any registers that will be needed after the subproblem is
solved so that these can be restored to continue the suspended computation.
In the case of factorial, we will save the old value of
#py("n"), to be restored when we are finished
computing the factorial of the decremented #py("n")
register.#footnote[One might argue that we don't need to save the old
#py("n"); after we decrement it and solve the
subproblem, we could simply increment it to recover the old value. Although
this strategy works for factorial, it cannot work in general, since the old
value of a register cannot always be computed from the new one.]

Since there is no a priori limit on the depth of nested
recursive calls, we may need to save an arbitrary number of register
values. These values must be restored in the reverse of the order in
which they were saved, since in a nest of recursions the last
subproblem to be entered is the first to be finished. This dictates
the use of a #emph[stack], or "last in, first out" data
structure, to save register values. We can extend the register-machine
language to include a stack by adding two kinds of instructions: Values are
placed
on the stack using a
#idx("register-machine language", sub: "save")
#idx("save (in register machine)")
#py("save") instruction and
restored from the stack using a
#idx("register-machine language", sub: "restore")
#idx("restore (in register machine)")
#py("restore")
instruction. After a sequence of values has been
#py("save")d on the stack, a sequence of
#py("restore")s will retrieve these values in reverse
order.#footnote[In section @sec:storage-allocation we
will see how to implement a stack in terms of more primitive
operations.]

With the aid of the stack, we can reuse a single copy of the factorial
machine's data paths for each factorial subproblem. There is a
similar design issue in reusing the controller sequence that operates
the data paths. To reexecute the factorial computation, the
controller cannot simply loop back to the beginning, as with
an iterative process, because after solving the
$(n-1)!$ subproblem
the machine must still multiply the result by
$n$. The controller
must suspend its computation of $n!$, solve the
$(n-1)!$ subproblem,
then continue its computation of $n!$. This
view of the factorial computation suggests the use of the subroutine
mechanism described in section @sec:subroutines, which
has the controller use a
#idx("continue register", sub: "recursion and")
#py("continue") register to transfer to the part of
the sequence that solves a subproblem and then continue where it left off on
the main problem. We can thus make a factorial subroutine that returns to
the entry point stored in the #py("continue")
register. Around each subroutine call, we save and restore
#py("continue") just as we do the
#py("n") register, since each "level" of
the factorial computation will use the same
#py("continue") register. That is, the factorial
subroutine must put a new value in #py("continue")
when it calls itself for a subproblem, but it will need the old value in
order to return to the place that called it to solve a subproblem.

#sicp-figure(stack(dir: ttb, spacing: 1em, image("/images/img_javascript/Fig5.11b.std.svg", width: 70%), [#syntax("
controller(
  llist(
      assign(\"continue\", label(\"fact_done\")),  # set up final return address
    \"fact_loop\",
      test(llist(op(\"=\"), reg(\"n\"), constant(1))),
      branch(label(\"base_case\")),
      # Set up for recursive call by saving ", $mono("n")$, " and ", $mono("continue")$, ".
      # Set up ", $mono("continue")$, " so that the computation will continue
      # at ", $mono("after_fact")$, " when the subroutine returns.
      save(\"continue\"),
      save(\"n\"),
      assign(\"n\", llist(op(\"-\"), reg(\"n\"), constant(1))),
      assign(\"continue\", label(\"after_fact\")),
      go_to(label(\"fact_loop\")),
    \"after_fact\",
      restore(\"n\"),
      restore(\"continue\"),
      assign(\"val\",                   # ", $mono("val")$, " now contains $n(n-1)!$
             llist(op(\"*\"), reg(\"n\"), reg(\"val\"))),
      go_to(reg(\"continue\")),         # return to caller
    \"base_case\",
      assign(\"val\", constant(1)),     # base case: 1! = 1
      go_to(reg(\"continue\")),         # return to caller
    \"fact_done\"))
")]), caption: [A recursive #idx("factorial", sub: "register machine for (recursive)") factorial machine.], label-name: <fig:fact-machine>)

Figure @fig:fact-machine
shows the data paths and controller for
a machine that implements the recursive
#py("factorial")
function.
The machine has a stack and three registers, called
#py("n"), #py("val"), and
#py("continue"). To simplify the data-path diagram,
we have not named the register-assignment buttons, only the stack-operation
buttons (#py("sc") and #py("sn")
to save registers, #py("rc") and
#py("rn") to restore registers). To operate the
machine, we put in register #py("n") the number whose
factorial we wish to compute and start the machine. When the machine
reaches #py("fact_done"), the computation is finished
and the answer will be found in the #py("val")
register. In the controller sequence, #py("n") and
#py("continue") are saved before each recursive call
and restored upon return from the call. Returning from a call is
accomplished by branching to the location stored in
#py("continue").
The register #py("continue")
is initialized when the machine starts so that the last return will go to
#py("fact_done"). The
#py("val")
register, which holds the result of the factorial computation, is not
saved before the recursive call, because the old contents of
#py("val") is not useful after the subroutine returns.
Only the new value, which is the value produced by the subcomputation, is
needed.

#idx("factorial", sub: "register machine for (recursive)")

Although in principle the factorial computation requires an infinite
machine, the machine in
figure @fig:fact-machine
is actually finite except for the stack, which is potentially unbounded. Any
particular physical implementation of a stack, however, will be of finite
size, and this will limit the depth of recursive calls that can be handled
by the machine. This implementation of factorial illustrates the general
strategy for realizing recursive algorithms as ordinary register machines
augmented by stacks. When a recursive subproblem is encountered, we save on
the stack the registers whose current values will be required after the
subproblem is solved, solve the recursive subproblem, then restore the saved
registers and continue execution on the main problem. The
#py("continue") register must always be saved.
Whether there are other registers that need to be saved depends on the
particular machine, since not all recursive computations need the original
values of registers that are modified during solution of the subproblem
(see exercise @ex:design-reg-machines).

#subheading([A double recursion])

Let us examine a more complex recursive process, the tree-recursive
computation of the
#idx("fib", sub: "register machine for (tree-recursive)")
Fibonacci numbers, which we introduced in
section @sec:tree-recursion:

#snippet(```python
def fib(n):
    return 0 if n == 0 else 1 if n == 1 else fib(n - 1) + fib(n - 2)
```)

Just as with factorial, we can implement the recursive Fibonacci
computation as a register machine with registers
#py("n"), #py("val"),
and #py("continue"). The machine is more complex than
the one for factorial, because there are two places in the controller
sequence where we need to perform recursive calls—once to compute
Fib$(n-1)$ and once to compute
Fib$(n-2)$. To set up for each of these calls,
we save the registers whose values will be needed later, set the
#py("n")
register to the number whose Fib we need to compute recursively
($n-1$ or $n-2$), and
assign to #py("continue") the entry point in the main
sequence to which to return (#py("afterfib_n_1") or
#py("afterfib_n_2"), respectively). We then go to
#py("fib_loop"). When we return from the
recursive call, the answer is in #py("val").
Figure @fig:fib-machine shows the controller sequence
for this machine.

#sicp-figure([#syntax("
controller(
  llist(
    assign(\"continue\", label(\"fib_done\")),
  \"fib_loop\",
    test(llist(op(\"<\"), reg(\"n\"), constant(2))),
    branch(label(\"immediate_answer\")),
    # set up to compute ", $upright("Fib")(n-1)$, "
    save(\"continue\"),
    assign(\"continue\", label(\"afterfib_n_1\")),
    save(\"n\"),                     # save old value of ", $mono("n")$, "
    assign(\"n\", llist(op(\"-\"), reg(\"n\"), constant(1))),  # clobber ", $mono("n")$, " to $n-1$
    go_to(label(\"fib_loop\")),      # perform recursive call
  \"afterfib_n_1\",                  # upon return, ", $mono("val")$, " contains ", $upright("Fib")(n-1)$, "
    restore(\"n\"),
    restore(\"continue\"),
    # set up to compute ", $upright("Fib")(n-2)$, "
    assign(\"n\", llist(op(\"-\"), reg(\"n\"), constant(2))),
    save(\"continue\"),
    assign(\"continue\", label(\"afterfib_n_2\")),
    save(\"val\"),                   # save ", $upright("Fib")(n-1)$, "
    go_to(label(\"fib_loop\")),
  \"afterfib_n_2\",                  # upon return, ", $mono("val")$, " contains ", $upright("Fib")(n-2)$, "
    assign(\"n\", reg(\"val\")),       # ", $mono("n")$, " now contains ", $upright("Fib")(n-2)$, "
    restore(\"val\"),                # ", $mono("val")$, " now contains ", $upright("Fib")(n-1)$, "
    restore(\"continue\"),
    assign(\"val\",                  # ", $upright("Fib")(n-1) + upright("Fib")(n-2)$, "
      llist(op(\"+\"), reg(\"val\"), reg(\"n\"))),
    go_to(reg(\"continue\")),        # return to caller, answer in ", $mono("val")$, "
  \"immediate_answer\",
    assign(\"val\", reg(\"n\")),       # base case: ", $upright("Fib")(n) = n$, "
    go_to(reg(\"continue\")),
  \"fib_done\"))
")], caption: [Controller for a machine to compute #idx("fib", sub: "register machine for (tree-recursive)") Fibonacci numbers.], label-name: <fig:fib-machine>)

#exercise(label-name: <ex:design-reg-machines>, [
Specify register machines that implement each of the following
functions.
For each machine, write a controller instruction sequence
and draw a diagram showing the data paths.

+ Recursive exponentiation: #idx("expt", sub: "register machine for") #snippet(```python def expt(b, n): return 1 if n == 0 else b * expt(b, n - 1) ```)
+ Iterative exponentiation: #snippet(```python def expt(b, n): def expt_iter(counter, product): return product if counter == 0 else expt_iter(counter - 1, b * product) return expt_iter(n, 1) ```)
])

#exercise(label-name: <ex:hand-sim>, [
Hand-simulate the factorial and Fibonacci machines, using some
nontrivial input (requiring execution of at least one recursive call).
Show the contents of the stack at each significant point in the
execution.
])

#exercise(label-name: <ex:5_6>, [
Ben Bitdiddle observes that the Fibonacci machine's controller sequence
has an extra #py("save") and an extra
#py("restore"), which can be removed to make a faster
machine. Where are these instructions?
])

#idx("stack", sub: "for recursion in register machine")
#idx("register machine", sub: "stack")
#idx("recursive process", sub: "register machine for")
