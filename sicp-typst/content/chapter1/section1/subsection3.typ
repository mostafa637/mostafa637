// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Evaluating Operator Combinations], label-name: <sec:evaluating-combinations>)

#idx("operator combination", sub: "evaluation of")
#idx("evaluation", sub: "of operator combination")

One of our goals in this chapter is to isolate issues about thinking
procedurally. As a case in point, let us consider that, in evaluating
operator
combinations, the interpreter is itself following a procedure.

- To evaluate an operator combination, do the following:

  + Evaluate the operand expressions of the combination.
  + Apply the function that is denoted by the operator to the arguments that are the values of the operands.

Even this simple rule illustrates some important points about
processes in general. First, observe that the first step dictates
that in order to accomplish the evaluation process for a
combination we must first perform the evaluation process on each
operand of the combination. Thus, the evaluation rule is
#idx("recursion")
#emph[recursive] in nature;
that is, it includes, as one of its steps, the need to invoke the rule
itself.

Notice how succinctly the idea of recursion can be used to express
#idx("recursion", sub: "expressing complicated process")
what, in the case of a deeply nested combination, would otherwise be
viewed as a rather complicated process. For example, evaluating

#snippet(```python
(2 + 4 * 6) * (3 + 12)
```)

requires that the evaluation rule be applied to four different
combinations. We can obtain a picture of this process by
representing the combination in the form of a
#idx("operator combination", sub: "as a tree")
#idx("tree", sub: "combination viewed as")
tree, as shown in
figure @fig:tree-comb.
Each combination is represented by a
#idx("node of a tree")
node with
#idx("branch of a tree")
branches corresponding to the operator and the
operands of the combination stemming from it.
The
#idx("terminal node of a tree")
terminal nodes (that is, nodes with
no branches stemming from them) represent either operators or numbers.
Viewing evaluation in terms of the tree, we can imagine that the
values of the operands percolate upward, starting from the terminal
nodes and then combining at higher and higher levels. In general, we
shall see that recursion is a very powerful technique for dealing with
hierarchical, treelike objects. In fact, the "percolate values upward" form of the evaluation rule is an example of a general kind
of process known as
#idx("tree accumulation")
#emph[tree accumulation].

#sicp-figure(image("/images/img_javascript/ch1-Z-G-1.svg", width: 70%), caption: [Tree representation, showing the value of each subexpression.], label-name: <fig:tree-comb>)

Next, observe that the repeated application of the first step brings
us to the point where we need to evaluate, not combinations, but
primitive expressions such as
numerals or names.
We take care of the primitive cases
#idx("primitive expression", sub: "evaluation of")
#idx("evaluation", sub: "of primitive expression")
by stipulating that

- the values of numerals are the numbers that they name, and
- the values of names are the objects associated with those names in the environment.

The key point to
notice is the role of the
#idx("environment", sub: "as context for evaluation")
environment in determining the meaning of
the
names
in expressions. In an interactive language such as
Python,
it is meaningless to speak of the value of an expression such as
#py("x + 1")
without specifying any information about the environment
that would provide a meaning for the
name #py("x").
As we shall see in chapter 3, the general notion of
the environment as providing a context in which evaluation takes place
will play an important role in our understanding of program execution.
#idx("operator combination", sub: "evaluation of") #idx("evaluation", sub: "of operator combination")

Notice that the
evaluation rule given above does not handle
#idx("declaration assignment", sub: "why a syntactic form") declaration assignments.
For instance, evaluating
#py("x = 3")
does not apply
an operator #py("=")
to two arguments, one
of which is the value of the
name
#py("x") and the other of which is
3, since the purpose of the
declaration assignment
is precisely to associate
#py("x") with a value.
(That is,
#py("x = 3")
is not
a combination.)

The shape of the declaration assignment
instructs the Python
interpreter to treat the statement in a special way. Each such
#idx("syntactic form")
#emph[syntactic form] has its own evaluation rule. The
various kinds of statements and expressions (each with its associated
evaluation rule) constitute the
#idx("syntax", sub: "of a programming language")
syntax of the programming language.
