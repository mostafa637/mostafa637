// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([CSE Machine], label-name: <sec:cse-machine>)

#idx("CSE machine")

The environment model as presented so far focuses on how functions can refer
to their parameters, locally declared names, and names that are declared
outside the function. We achieve this by evaluating statements and expressions
with respect to a #emph[current environment]. It does not specify how
we keep track of environments as computation proceeds. For example, when we
evaluate an expression #py("f(x) + y"), we
need to evaluate #py("x") in the current
environment, establish as the new current environment the environment of
#py("f") extended by a binding of its
parameter to the value of #py("x"), and
evaluate the body of #py("f") in this
extended environment. But what environment should we use for evaluating
#py("y") after
#py("f") returns?
In this section, we extend the

#subheading([Evaluating arithmetic expressions])

Exercise @ex:3_8 shows that the presence of
assignments makes the result of a program depend on the order in which
the operands of an operator combination are evaluated. To remove
ambiguities that arise from this, the Python standard specifies
left-to-right evaluation of operands.

As an example, consider the evaluation of the arithmetic expression statement

#snippet(```python
1 + (2 * 3)
```)

The expression is decomposed into its operands
#py("1") and
#py("2 * 3"), followed by the
#emph[instruction] to add their results.

#idx("CSE machine")
