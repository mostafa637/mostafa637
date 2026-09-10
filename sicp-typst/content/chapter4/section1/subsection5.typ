// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Data as Programs], label-name: <sec:data-as-programs>)

#idx("program", sub: "as data")
#idx("data", sub: "as program")

#sicp-figure(image("/images/img_original/ch4-Z-G-2.svg", width: 70%), caption: [The factorial program, viewed as an abstract machine.], label-name: <fig:fact-proc-as-machine>)

In thinking about a
Python
program that evaluates
Python statements and
expressions, an analogy might be helpful. One operational view of the
meaning of a program is that a
#idx("program", sub: "as abstract machine")
program is a description of an abstract (perhaps infinitely large) machine.
For example, consider the familiar program to compute factorials:

#snippet(```python
def factorial(n):
    return 1 if n == 1 else factorial(n - 1) * n
```)

We may regard this program as the description of a
#idx("factorial", sub: "as an abstract machine")
machine containing
parts that decrement, multiply, and test for equality, together with a
two-position switch and another factorial machine. (The factorial
machine is infinite because it contains another factorial machine
within it.) Figure @fig:fact-proc-as-machine is a flow
diagram for the factorial machine, showing how the parts are wired together.

In a similar way, we can regard the evaluator as a very special
#idx("evaluator", sub: "as abstract machine")
machine that takes as input a description of a machine. Given this
input, the evaluator configures itself to emulate the machine
described. For example, if we feed our evaluator the definition of
#py("factorial"), as shown in
figure @fig:eval-factorial,
the evaluator will be able to compute factorials.

#sicp-figure(image("/images/img_javascript/ch4-Z-G-3.svg", width: 70%), caption: [The evaluator emulating a factorial machine.], label-name: <fig:eval-factorial>)

From this perspective, our evaluator is seen to be a
#idx("evaluator", sub: "as universal machine")
#idx("universal machine")
#emph[universal machine].
It mimics other machines when these are described as
Python
programs.#footnote[The fact that the machines are described in
Python
is inessential. If we give our evaluator a
Python
program that behaves as an evaluator for some other language, say C, the
Python
evaluator will emulate the C evaluator, which in turn can emulate any
machine described as a C program. Similarly, writing a
Python
evaluator in C produces a C program that can execute any
Python
program. The deep idea here is that any evaluator can emulate any other.
Thus, the notion of "what can in principle be computed"
(ignoring practicalities of time and memory required) is independent of the
language or the computer, and instead reflects an underlying notion of
#idx("computability")
#emph[computability]. This was first demonstrated in a clear way by
#idx("Turing, Alan M.")
Alan M. Turing (1912–1954), whose 1936 paper laid the foundations
for theoretical
#idx("computer science")
computer science. In the paper, Turing presented a simple computational
model—now known as a
#idx("Turing machine")
#emph[Turing machine]—and argued that any "effective process" can be formulated as a program for such a machine. (This
argument is known as the
#idx("Church–Turing thesis")
#emph[Church–Turing thesis].) Turing then implemented a universal machine,
i.e., a Turing machine that behaves as an evaluator for Turing-machine
programs. He used this framework to demonstrate that there are well-posed
problems that cannot be computed by Turing machines (see
exercise @ex:halting-theorem), and so by implication
cannot be formulated as "effective processes." Turing went on
to make fundamental contributions to practical computer science as well.
For example, he invented the idea of
#idx("program", sub: "structured with subroutines")
structuring programs using general-purpose subroutines. See
#idx("Hodges, Andrew")
Hodges 1983 for a biography of Turing.]
This is striking. Try to imagine an analogous evaluator for electrical
circuits. This would be a circuit that takes as input a signal encoding the
plans for some other circuit, such as a filter. Given this input, the
circuit evaluator would then behave like a filter with the same description.
Such a universal electrical circuit is almost unimaginably complex. It is
remarkable that the program evaluator is a rather simple
program.#footnote[Some people find it counterintuitive that an evaluator,
which is implemented by a relatively simple
function,
can emulate programs that are more complex than the evaluator itself. The
existence of a universal evaluator machine is a deep and wonderful property
of computation.
#idx("recursion theory")
#emph[Recursion theory], a branch of mathematical logic, is concerned with
logical limits of computation.
#idx("Hofstadter, Douglas R.")
Douglas Hofstadter's beautiful book #emph[Gödel, Escher, Bach] (1979) explores some of these ideas.]

Another striking aspect of the evaluator is that it acts as a bridge between
the data objects that are manipulated by our programming language and the
programming language itself. Imagine that the evaluator program
(implemented in Python)
is running, and that a user is typing
programs
to the evaluator and
observing the results. From the perspective of the user, an input
program
such as
#py("x * x;")
is
a program
in the programming language, which the evaluator should
execute.
From the perspective of the evaluator, however, the program is simply a string or—after parsing—a tagged-list representation that is to be manipulated according to a well-defined set of rules.

That the
#idx("Python", sub: "eval in", decl: true)
#idx("eval (primitive function in Python)")

user's programs are the evaluator's data need not
be a source of confusion. In fact, it is sometimes convenient to ignore
this distinction, and to give the user the ability to explicitly
evaluate a string as a Python statement, using Python's
primitive function #py("eval")
that takes as argument a string. It parses the string
and—provided that it is syntactically correct—evaluates the
resulting representation in the environment in which
#py("eval") is applied. Thus,

#snippet(```python
eval("5 * 5;");
```)

and

#snippet(```python
evaluate(parse("5 * 5;"), the_global_environment)
```)

will both return 25.#footnote[Note that
#py("eval")
may not be available in the Python environment that you are
using, or its use may be restricted for security reasons.]

#idx("program", sub: "as data")
#idx("data", sub: "as program")

#exercise(label-name: <ex:halting-theorem>, [
Given a one-argument
function
#py("f")
and an object #py("a"),
#py("f")
is said to "halt" on
#py("a") if evaluating the expression
#py("f(a)")
returns a value (as opposed to terminating with an error message or running
forever).
#idx("halting problem")
Show that it is impossible to write a
function
#py("halts")
that correctly determines whether
#py("f")
halts on
#py("a") for any
function
#py("f")
and object #py("a").
Use the following reasoning: If you had such a
function
#py("halts"),
you could implement the following program:

#snippet(```python
def run_forever():
    return run_forever()

def strange(f):
    return run_forever() if halts(f, f) else "halted"
```)

Now consider evaluating the expression
#py("strange(strange)")
and show that any possible outcome (either halting or running forever)
violates the intended behavior of
#py("halts").#footnote[Although
we stipulated that
#py("halts")
is given a
function
object, notice that this reasoning still applies even if
#py("halts")
can gain access to the
function's
text and its environment.
#idx("Turing, Alan M.")
This is Turing's celebrated
#idx("Halting Theorem")
#emph[Halting Theorem], which gave the
first clear example of a
#idx("noncomputable")#idx("computability")
#emph[noncomputable] problem, i.e., a well-posed
task that cannot be carried out as a computational
function.]
])

#idx("metacircular evaluator for Python")
