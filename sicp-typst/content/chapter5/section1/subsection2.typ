// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Abstraction in Machine Design])

#idx("abstraction", sub: "in register-machine design")

We will often define a machine to include "primitive"
operations that are actually very complex. For example, in
sections @sec:eceval and @sec:compilation
we will treat
Python's
environment manipulations as primitive. Such abstraction is valuable
because it allows us to ignore the details of parts of a machine so that we
can concentrate on other aspects of the design. The fact that we have
swept a lot of complexity under the rug, however, does not mean that a
machine design is unrealistic. We can always replace the complex
"primitives" by simpler primitive operations.

Consider the GCD machine. The machine has an instruction that computes
the remainder of the contents of registers #py("a")
and #py("b") and assigns the result to register
#py("t"). If we want to construct the GCD machine
without using a primitive remainder operation, we must specify how to
compute remainders in terms of simpler operations, such as subtraction.
Indeed, we can write a
Python function
that finds remainders in this way:

#snippet(```python
def remainder(n, d):
    return n if n < d else remainder(n - d, d)
```)

We can thus replace the remainder operation in the GCD machine's
data paths with a subtraction operation and a comparison test.
Figure @fig:gcd-machine-rem shows the data paths and
controller for the elaborated machine. The instruction

#sicp-figure(image("/images/img_original/Fig5.5b.std.svg", width: 70%), caption: [Data paths and controller for the elaborated GCD machine.], label-name: <fig:gcd-machine-rem>)

#snippet(```python
assign("t", llist(op("rem"), reg("a"), reg("b")))
```)

in the GCD controller definition is replaced by a sequence of
instructions that contains a loop, as shown in
figure @fig:gcd-machine-rem-controller.

#sicp-figure([#snippet(```python
controller(
  llist(
    "test_b",
      test(llist(op("="), reg("b"), constant(0))),
      branch(label("gcd_done")),
      assign("t", reg("a")),
    "rem_loop",
      test(llist(op("<"), reg("t"), reg("b"))),
      branch(label("rem_done")),
      assign("t", llist(op("-"), reg("t"), reg("b"))),
      go_to(label("rem_loop")),
    "rem_done",
      assign("a", reg("b")),
      assign("b", reg("t")),
      go_to(label("test_b")),
    "gcd_done"))
```)], caption: [Controller instruction sequence for the GCD machine in figure @fig:gcd-machine-rem.], label-name: <fig:gcd-machine-rem-controller>)

#exercise(label-name: <ex:sqrt-machine>, [
Design a machine to compute
#idx("sqrt", sub: "register machine for")
square roots using Newton's method, as
described in section @sec:sqrt and implemented with the following code in section @sec:block-structure:

#snippet(```python
def sqrt(x):
    def is_good_enough(guess):
        return math_abs(square(guess) - x) < 0.001
    def improve(guess):
        return average(guess, x / guess)
    def sqrt_iter(guess):
        return guess if is_good_enough(guess) else sqrt_iter(improve(guess))
    return sqrt_iter(1)
```)

Begin by assuming that
#py("is_good_enough")
and #py("improve") operations are available as
primitives. Then show how to expand these in terms of arithmetic
operations. Describe each version of the #py("sqrt")
machine design by drawing a data-path diagram and writing a controller
definition in the register-machine language.
])

#idx("abstraction", sub: "in register-machine design")
