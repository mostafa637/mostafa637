// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../lib/sicp.typ": *

#chapter([Computing with Register Machines], label-name: <chap:reg>)

#epigraph([My aim is to show that the heavenly machine is not a kind of divine,
live being, but a kind of clockwork (and he who believes that a clock
has soul attributes the maker's glory to the work), insofar as nearly
all the manifold motions are caused by a most simple and material
force, just as all motions of the clock are caused by a single weight.

#idx("Kepler, Johannes")], author: [Johannes Kepler], date: [letter to Herwart von Hohenburg, 1605])

We began this book by studying processes and by describing processes
in terms of
functions
written in
Python.
To explain the meanings of these
functions,
we used a succession of models of evaluation: the
substitution model of chapter @chap:fun, the environment model of
chapter @chap:state, and the metacircular evaluator of chapter @chap:meta. Our
examination of the metacircular evaluator, in particular, dispelled much of
the mystery of how
Python-like languages are interpreted.
But even the metacircular evaluator leaves important questions
unanswered, because it fails to elucidate the mechanisms of control in a
Python
system. For instance, the evaluator does not explain how the
evaluation of a subexpression manages to return a value to the
expression that uses this value.
Also, the evaluator does not explain how some recursive functions can generate iterative processes (that is, be evaluated using constant space) whereas other recursive functions will generate recursive processes.#footnote[With our metacircular evaluator, a recursive function always gives rise to a recursive process, even when the process should be iterative according to the distinction of section @sec:recursion-and-iteration. See footnote @foot:apply in section @sec:core-of-evaluator.] This chapter addresses both of these issues.

We
will describe processes in terms of the step-by-step
operation of a traditional computer. Such a computer, or
#idx("register machine")
#emph[register machine], sequentially executes
#emph[instructions] that
manipulate the contents of a fixed set of storage elements called
#idx("register(s)")
#emph[registers]. A typical register-machine instruction applies a
primitive operation to the contents of some registers and assigns the
result to another register. Our descriptions of processes executed by
register machines will look very much like "machine-language"
programs for traditional computers. However, instead of focusing on
the machine language of any particular computer, we will examine
several
Python
functions
and design a specific register machine to
execute each
function.
Thus, we will approach our task from the
perspective of a hardware architect rather than that of a
machine-language computer programmer. In designing register machines,
we will develop mechanisms for implementing important programming
constructs such as recursion. We will also present a language for
describing designs for register machines. In
section @sec:simulator we will
implement a
Python
program that uses these descriptions to simulate the machines we design.

Most of the primitive operations of our register machines are very
simple. For example, an operation might add the numbers fetched from
two registers, producing a result to be stored into a third register.
Such an operation can be performed by easily described hardware. In
order to deal with list structure, however, we will also use the
memory operations
#py("head"),
#py("tail"),
and
#py("pair"),
which require an elaborate storage-allocation mechanism. In
section @sec:storage-allocation we study their
implementation in terms of more elementary operations.

In section @sec:eceval, after we have accumulated
experience formulating simple
functions
as register machines, we will design a
machine that carries out the algorithm described by the metacircular
evaluator of section @sec:mc-eval. This will fill in
the gap in our understanding of how
Python programs
are interpreted, by providing an explicit model for the mechanisms of
control in the evaluator.
In section @sec:compilation we will study a simple
compiler that translates
Python
programs into sequences of instructions that can be executed directly with
the registers and operations of the evaluator register machine.

#include "../chapter5/section1/section1.typ"

#include "../chapter5/section2/section2.typ"

#include "../chapter5/section3/section3.typ"

#include "../chapter5/section4/section4.typ"

#include "../chapter5/section5/section5.typ"
