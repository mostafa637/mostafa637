// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../lib/sicp.typ": *

#chapter([Building Abstractions with Functions], label-name: <chap:fun>)

#epigraph([The acts of the mind, wherein it exerts its power over simple ideas,
are chiefly these three: 1. Combining several simple ideas into one
compound one, and thus all complex ideas are made. 2. The second is
bringing two ideas, whether simple or complex, together, and setting
them by one another so as to take a view of them at once, without
uniting them into one, by which it gets all its ideas of relations.
3\. The third is separating them from all other ideas that accompany
them in their real existence: this is called abstraction, and thus all
its general ideas are made.], author: [John Locke], title: [An Essay Concerning Human Understanding], date: [1690])

We are about to study the idea of a
#emph[computational process].
#idx("process")
#idx("computational process")
Computational processes are abstract beings that inhabit computers.
As they evolve, processes manipulate other abstract things called
#emph[data].
#idx("data")
The evolution of a process is directed by a pattern of rules
called a
#idx("program")
#emph[program].
People create programs to direct processes.
In effect, we conjure the spirits of the computer with our spells.

A computational process is indeed much like a sorcerer's idea of a
spirit. It cannot be seen or touched. It is not composed of matter
at all. However, it is very real. It can perform intellectual work.
It can answer questions. It can affect the world by disbursing money
at a bank or by controlling a robot arm in a factory. The programs we
use to conjure processes are like a sorcerer's spells. They are
carefully composed from symbolic expressions in arcane and esoteric
#emph[programming languages]
#idx("programming language")
that prescribe the tasks we want our
processes to perform.

A computational process, in a correctly working computer, executes
programs precisely and accurately. Thus, like the sorcerer's
apprentice, novice programmers must learn to understand and to
anticipate the consequences of their conjuring. Even small errors
(usually called #idx("bug") #emph[bugs])

in programs can have complex and unanticipated consequences.

Fortunately, learning to program is considerably less dangerous than
learning sorcery, because the spirits we deal with are conveniently
contained in a secure way. Real-world programming, however,
requires care, expertise, and wisdom. A small bug in a computer-aided
design program, for example, can lead to the catastrophic collapse of
an airplane or a dam or the self-destruction of an industrial robot.

Master software engineers have the ability to organize programs so
that they can be reasonably sure that the resulting processes will
perform the tasks intended. They can visualize the behavior of their
systems in advance. They know how to structure programs so that
unanticipated problems do not lead to catastrophic consequences, and
when problems do arise, they can
#idx("debug")
#emph[debug]
their programs. Well-designed
computational systems, like well-designed automobiles or nuclear
reactors, are designed in a modular manner, so that the parts can be
constructed, replaced, and debugged separately.

#subheading([Programming in Python])

We need an appropriate language for describing processes, and
we will use for this purpose the programming language
Python. Just as our everyday thoughts are usually expressed in
our natural language (such as English, German, or Chinese),
and descriptions of quantitative phenomena are expressed with
mathematical notations, our procedural thoughts will be
expressed in Python.
#idx("Python", sub: "history of")
Python was conceived in late 1989 by
#idx("van Rossum, Guido", sort: "Rossum")
#idx("CWI (Centrum Wiskunde en Informatica)")
Guido van Rossum at Centrum Wiskunde & Informatica (CWI)
in Amsterdam, and first released in 1991. Van Rossum designed
#idx("Amoeba operating system")
it as a high-level scripting language for the Amoeba
distributed operating system—a language for the kind
of administrative tasks that were too intricate for shell
scripts yet not large enough to be worth writing out as a
full, compiled program. From the outset Python was meant to
cooperate with software written in other languages, and it
became especially well known for its ability to serve as a
glue that binds together components written in C. The name
"Python" is not a reference to the snake but to
the British comedy troupe Monty Python's Flying Circus, and is
#idx("Python Software Foundation")
a registered trademark of the Python Software Foundation.

Despite its inception as a language for scripting and gluing
together software components, Python is a general-purpose
programming language.
#idx("interpreter")
A Python interpreter is a machine that
carries out processes described in the Python language. The
#idx("van Rossum, Guido", sort: "Rossum")
first Python interpreter was implemented by van Rossum at CWI;
written in C, it remains the reference implementation of the
#idx("CPython")
#idx("Python", sub: "CPython as reference implementation")
language and is known today as CPython. Python's most direct
#idx("ABC")
ancestor is ABC, a teaching language that van Rossum had
helped develop at CWI, from which Python took such traits as
the use of indentation to express block structure and a set of
rich built-in data types; further influences came from
#idx("Modula-3")
Modula-3 and, for much of its syntax and its implementation,
from C. Unlike the original version of this book, which used
#idx("Scheme", sub: "dialect of Lisp")
the Lisp dialect Scheme, Python does not descend from the Lisp
tradition. It nonetheless provides the features on which this
book most depends—first-class functions, lexical scoping, and
dynamic typing.

Unlike Java and C, which are normally compiled to a
lower-level language before they run, a Python program is
first translated into an intermediate form called
#idx("bytecode")
#emph[bytecode],
#idx("virtual machine, interpreting Python")
which is then #emph[interpreted] by a virtual machine; CPython works
in exactly this way, and alternative implementations such as
#idx("PyPy")
PyPy exist as well. Python is not overseen by a formal
standards body. Instead the language is defined by its
reference implementation together with the
#idx("Python Language Reference (PLR)")
#idx("PLR (Python Language Reference)")
#emph[Python Language Reference],
and it evolves through a community process
#idx("Python Enhancement Proposal (PEP)")
organized around design documents called #emph[Python Enhancement Proposals], or PEPs.
#idx("Python", sub: "versions 2 and 3")
The language has had two major lines:
Python 2, which appeared in 2000, and Python 3, which appeared
in 2008 and deliberately broke backward compatibility with its
predecessor; the two coexisted for more than a decade, until
support for Python 2 ended in January 2020 (PEP 373). This
book uses Python 3\.

#idx("Python", sub: "efficiency of")
#idx("efficiency", sub: "of Python")
The growing use of Python for tasks that demand high
performance—above all in scientific computing and data
analysis—encouraged work on executing Python programs
efficiently. Interpreting bytecode, as CPython does, is rarely
the fastest way to carry out a computation.
Two responses have proved important. The first,
and in practice the more consequential, is to keep the program
itself in Python while delegating its numerically intensive
parts to libraries written in lower-level languages such as C
and Fortran; the widely used
#idx("NumPy")
NumPy library is the canonical
example, and this strategy relies on exactly the cooperation
with C described earlier.
#idx("Just-In-Time (JIT) compilation")
The second is Just-In-Time (JIT)
compilation, in which the frequently executed portions of a
program are translated into native machine code as the program
runs. PyPy has long applied
this technique to good effect, and as of this writing (2026)
CPython has begun to incorporate a JIT compiler of its own,
though it remains experimental and its performance and future
are still being worked out. Python is today among the most
widely used programming languages, prominent in fields ranging
from web development and automation to scientific computing,
data science, machine learning, and the teaching of
programming.

Python possesses a set of features that make it an excellent
medium for studying important programming constructs and data
structures and for relating them to the linguistic features
that support them. Its lexically scoped first-class functions
give direct access to functional abstraction; that abstraction
is expressed chiefly through the function-definition statement
#py("def"), with the more restricted
#py("lambda") expression available for
short anonymous functions—a division of labor that echoes the
distinction between #py("define") and
#py("lambda") in Scheme itself.
Python's dynamic typing allows the adaptation to
remain close to the Scheme original throughout the book. Above
and beyond these considerations, programming in Python is
great fun.

#include "../chapter1/section1/section1.typ"

#include "../chapter1/section2/section2.typ"

#include "../chapter1/section3/section3.typ"
