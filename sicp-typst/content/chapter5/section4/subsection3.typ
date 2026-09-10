// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Blocks, Assignments, and Declarations], label-name: <sec:block-assign-def-evaluation>)

#subheading([Blocks])

#idx("explicit-control evaluator for Python", sub: "blocks")
#idx("explicit-control evaluator for Python", sub: "declarations")

The body of a block is evaluated with respect to the current environment
extended by a frame that binds all local names to the value
#py("\"*unassigned*\""). We temporarily
make use of the #py("val") register to
hold the list of all variables declared in the block, which is obtained
by
#idx("scanning out declarations", sub: "in explicit-control evaluator")
#py("scan_out_declarations")
from section @sec:core-of-evaluator. The functions
#py("scan_out_declarations") and
#py("list_of_unassigned") are assumed
to be available as machine operations.#footnote[Footnote @foot:syntax-transformer
suggests that an actual implementation would perform
syntax transformations before program execution.
In the same vein, names declared in blocks should be scanned out
in a preprocessing step rather than each time a block is evaluated.]
#idx("evblock", decl: true)
#syntax("
\"ev_block\",
  assign(\"comp\", list(op(\"block_body\"), reg(\"comp\"))),
  assign(\"val\", list(op(\"scan_out_declarations\"), reg(\"comp\"))),

  save(\"comp\"),    // so we can use it to temporarily hold ", $mono("*unassigned*")$, " values
  assign(\"comp\", list(op(\"list_of_unassigned\"), reg(\"val\"))),
  assign(\"env\", list(op(\"extend_environment\"),
                     reg(\"val\"), reg(\"comp\"), reg(\"env\"))),
  restore(\"comp\"), // the block body
  go_to(label(\"eval_dispatch\")),
	  ")

#subheading([Assignments and declarations])

Assignments
#idx("explicit-control evaluator for Python", sub: "assignments")
are handled by
#py("ev_assignment"),
reached from
#py("eval_dispatch")
with the assignment expression in #py("comp"). The
code at
#py("ev_assignment")
first evaluates the value part of the expression and then installs the new
value in the environment.
The function #py("assign_symbol_value")
is assumed to be available as a machine operation.
#idx("evassignment", decl: true)
#snippet(```python
"ev_assignment",
  assign("unev", list(op("assignment_symbol"), reg("comp"))),
  save("unev"), // save variable for later
  assign("comp", list(op("assignment_value_expression"), reg("comp"))),
  save("env"),
  save("continue"),
  assign("continue", label("ev_assignment_install")),
  go_to(label("eval_dispatch")), // evaluate assignment value
"ev_assignment_install",
  restore("continue"),
  restore("env"),
  restore("unev"),
  perform(list(op("assign_symbol_value"),
               reg("unev"), reg("val"), reg("env"))),
  go_to(reg("continue")),
```)

Declarations
#idx("explicit-control evaluator for Python", sub: "declarations")
of variables and constants are handled in a similar way.
Note that whereas the value of an assignment is the value that was assigned,
the value of a declaration is
#py("undefined"). This is handled by
setting #py("val") to
#py("undefined") before continuing.
As in the metacircular evaluator, we transform a function definition
into a constant declaration whose value expression is a lambda expression. This happens at
#py("ev_function_definition"), which makes the
transformation in place in #py("comp") and
falls through to #py("ev_declaration").

#idx("evfunctiondefinition", decl: true)#idx("evdeclaration", decl: true)
#snippet(```python
"ev_function_definition",
  assign("comp",
         list(op("function_decl_to_constant_decl"), reg("comp"))),
"ev_declaration",
  assign("unev", list(op("declaration_symbol"), reg("comp"))),
  save("unev"), // save declared name
  assign("comp",
         list(op("declaration_value_expression"), reg("comp"))),
  save("env"),
  save("continue"),
  assign("continue", label("ev_declaration_assign")),
  go_to(label("eval_dispatch")), // evaluate declaration value
"ev_declaration_assign",
  restore("continue"),
  restore("env"),
  restore("unev"),
  perform(list(op("assign_symbol_value"),
               reg("unev"), reg("val"), reg("env"))),
  assign("val", constant(undefined)),
  go_to(reg("continue")),
```)

#exercise(label-name: <ex:derived-expressions>, [
Extend the evaluator to handle
#idx("derived component", sub: "adding to explicit-control evaluator")
#idx("explicit-control evaluator for Python", sub: "derived components")
#idx("explicit-control evaluator for Python", sub: "syntactic forms (additional)")
while loops, by translating them to applications of a function
#py("while_loop"), as shown in
exercise @ex:while_loop.
You can paste the declaration of the function
#py("while_loop") in front of user programs.
You may "cheat" by assuming that the syntax transformer
#py("while_to_application") is available
as a machine operation. Refer to
exercise @ex:while_loop to discuss whether
this approach works if return, break, and continue statements are allowed
inside the while loop. If not,
how can you modify the explicit-control evaluator to run programs
with while loops that include these statements?
])

#exercise(label-name: <ex:5_26>, [
Modify the evaluator so that it uses
#idx("explicit-control evaluator for Python", sub: "normal-order evaluation")
#idx("normal-order evaluation", sub: "in explicit-control evaluator")
normal-order evaluation,
based on the lazy evaluator of
section @sec:lazy-evaluation.
])
