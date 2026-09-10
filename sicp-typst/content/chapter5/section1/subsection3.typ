// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Subroutines], label-name: <sec:subroutines>)

#idx("register machine", sub: "subroutine")
#idx("subroutine in register machine")

When designing a machine to perform a computation, we would often
prefer to arrange for components to be shared by different parts of
the computation rather than duplicate the components. Consider a
machine that includes two GCD computations—one that finds the GCD of
the contents of registers #py("a") and
#py("b") and one that finds the
GCD of the contents of registers #py("c") and
#py("d"). We might start
by assuming we have a primitive #py("gcd") operation,
then expand the two instances of #py("gcd") in terms
of more primitive operations.
Figure @fig:gcd-machine-1
shows just the GCD portions of the resulting machine's data paths,
without showing how they connect to the rest of the machine. The figure
also shows the corresponding portions of the machine's controller
sequence.

#sicp-figure(image("/images/img_javascript/Fig5.7b.std.svg", width: 70%), caption: [Portions of the data paths and controller sequence for a machine with two GCD computations.], label-name: <fig:gcd-machine-1>)

This machine has two remainder operation boxes and two boxes for
testing equality. If the duplicated components are complicated, as is the
remainder box, this will not be an economical way to build the
machine. We can avoid duplicating the data-path components by using
the same components for both GCD computations, provided that doing so
will not affect the rest of the larger machine's computation. If the
values in registers #py("a") and
#py("b") are not needed by the time the
controller gets to #py("gcd_2") (or if these values
can be moved to other registers for safekeeping), we can change the machine
so that it uses registers #py("a") and
#py("b"), rather than registers
#py("c") and #py("d"), in
computing the second GCD as well as the first. If we do this, we obtain the
controller sequence shown in
figure @fig:gcd-machine-2.

We have removed the duplicate data-path components (so that the data paths
are again as in figure @fig:gcd-machine), but the
controller now has two GCD sequences that differ only in their entry-point
labels. It would be better to replace these two sequences by branches to a
single sequence—a #py("gcd")
#emph[subroutine]—at the end of which we branch back to the
correct place in the main instruction sequence. We can accomplish this as
follows: Before branching to #py("gcd"), we place a
distinguishing value (such as 0 or 1) into a special register,
#idx("continue register")
#py("continue"). At the end of the
#py("gcd") subroutine we return either to
#py("after_gcd_1") or to #py("after_gcd_2"), depending
on the value of the #py("continue") register.
Figure @fig:gcd-machine-2cont shows the relevant portion
of the resulting controller sequence, which includes only a single copy of
the #py("gcd") instructions.

#sicp-figure([#syntax("
\"gcd_1\",
  test(llist(op(\"=\"), reg(\"b\"), constant(0))),
  branch(label(\"after_gcd_1\")),
  assign(\"t\", llist(op(\"rem\"), reg(\"a\"), reg(\"b\"))),
  assign(\"a\", reg(\"b\")),
  assign(\"b\", reg(\"t\")),
  go_to(label(\"gcd_1\")),
\"after_gcd_1\",
  ", $dots.v$, "
\"gcd_2\",
  test(llist(op(\"=\"), reg(\"b\"), constant(0))),
  branch(label(\"after_gcd_2\")),
  assign(\"t\", llist(op(\"rem\"), reg(\"a\"), reg(\"b\"))),
  assign(\"a\", reg(\"b\")),
  assign(\"b\", reg(\"t\")),
  go_to(label(\"gcd_2\")),
\"after_gcd_2\"
	")], caption: [Portions of the controller sequence for a machine that uses the same data-path components for two different GCD computations.], label-name: <fig:gcd-machine-2>)

#sicp-figure([#syntax("
\"gcd\",
  test(llist(op(\"=\"), reg(\"b\"), constant(0))),
  branch(label(\"gcd_done\")),
  assign(\"t\", llist(op(\"rem\"), reg(\"a\"), reg(\"b\"))),
  assign(\"a\", reg(\"b\")),
  assign(\"b\", reg(\"t\")),
  go_to(label(\"gcd\")),
\"gcd_done\",
  test(llist(op(\"=\"), reg(\"continue\"), constant(0))),
  branch(label(\"after_gcd_1\")),
  go_to(label(\"after_gcd_2\")),
  ", $dots.v$, "
  # Before branching to ", $mono("gcd")$, " from the first place where
  # it is needed, we place 0 in the ", $mono("continue")$, " register
  assign(\"continue\", constant(0)),
  go_to(label(\"gcd\")),
\"after_gcd_1\",
  ", $dots.v$, "
  # Before the second use of ", $mono("gcd")$, ", we place 1 in the ", $mono("continue")$, " register
  assign(\"continue\", constant(1)),
  go_to(label(\"gcd\")),
\"after_gcd_2\"
	")], caption: [Using a #py("continue") register to avoid the duplicate controller sequence in figure @fig:gcd-machine-2.], label-name: <fig:gcd-machine-2cont>)

This is a reasonable approach for handling small problems, but it would be
awkward if there were many instances of GCD computations in the controller
sequence. To decide where to continue executing after the
#py("gcd") subroutine, we would need tests in the data
paths and branch instructions in the controller for all the places that use
#py("gcd"). A more powerful method for implementing
subroutines is to have the #py("continue") register
hold the label of the entry point in the controller sequence at which
execution should continue when the subroutine is finished. Implementing this
strategy requires a new kind of connection between the data paths and the
controller of a register machine: There must be a way to assign to a
register a label in the controller sequence in such a way that this value
can be fetched from the register and used to continue execution at the
designated entry point.

To reflect this ability, we will extend the
#idx("assign (in register machine)", sub: "storing label in register")
#py("assign")
instruction of the register-machine language to allow a register to be
assigned as value a label from the controller sequence (as a special
kind of constant). We will also extend the
#idx("goto (in register machine)", sub: "destination in register") #py("go_to")
instruction to allow execution to continue at the entry point described by
the contents of a register rather than only at an entry point described by
a constant label. Using these new constructs we can terminate the
#py("gcd") subroutine with a branch to the location
stored in the #py("continue") register. This leads
to the controller sequence shown in
figure @fig:gcd-mach-2labels.

#sicp-figure([#syntax("
\"gcd\",
  test(llist(op(\"=\"), reg(\"b\"), constant(0))),
  branch(label(\"gcd_done\")),
  assign(\"t\", llist(op(\"rem\"), reg(\"a\"), reg(\"b\"))),
  assign(\"a\", reg(\"b\")),
  assign(\"b\", reg(\"t\")),
  go_to(label(\"gcd\")),
\"gcd_done\",
  go_to(reg(\"continue\")),
  ", $dots.v$, "
  # Before calling ", $mono("gcd")$, ", we assign to ", $mono("continue")$, "
  # the label to which ", $mono("gcd")$, " should return.
  assign(\"continue\", label(\"after_gcd_1\")),
  go_to(label(\"gcd\")),
\"after_gcd_1\",
  ", $dots.v$, "
  # Here is the second call to ", $mono("gcd")$, ", with a different continuation.
  assign(\"continue\", label(\"after_gcd_2\")),
  go_to(label(\"gcd\")),
\"after_gcd_2\"
	")], caption: [Assigning labels to the #py("continue") register simplifies and generalizes the strategy shown in figure @fig:gcd-machine-2cont.], label-name: <fig:gcd-mach-2labels>)

A machine with more than one subroutine could use multiple
continuation registers (e.g., #py("gcd_continue"),
#py("factorial_continue")) or we could have all
subroutines share a single
#py("continue") register. Sharing is more economical,
but we must be careful if we have a subroutine
(#py("sub1")) that calls another subroutine
(#py("sub2")). Unless
#py("sub1") saves the contents of
#py("continue") in some other register before setting
up #py("continue") for the call to
#py("sub2"), #py("sub1") will
not know where to go when it is finished. The mechanism developed in the
next section to handle recursion also provides a better solution to this
problem of nested subroutine calls.

#idx("register machine", sub: "subroutine")
#idx("subroutine in register machine")
