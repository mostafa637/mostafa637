// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Logic Programming], label-name: <sec:logic-programming>)

#idx("logic programming")

In chapter @chap:fun we stressed that computer science deals with
#idx("declarative vs. imperative knowledge")
#idx("imperative vs. declarative knowledge")
#idx("mathematics", sub: "computer science vs.")
#idx("computer science", sub: "mathematics vs.")
imperative
(how to) knowledge, whereas mathematics deals with declarative (what
is) knowledge. Indeed, programming languages require that the
programmer express knowledge in a form that indicates the step-by-step
methods for solving particular problems. On the other hand,
high-level languages provide, as part of the language implementation,
a substantial amount of methodological knowledge that frees
the user from concern with numerous details of how a specified
computation will progress.

Most programming languages, including
Python,
are organized around
computing the values of mathematical functions. Expression-oriented
languages
(such as Lisp, C, Python, and JavaScript)
capitalize on the
"pun" that an expression that describes the value of a
function may also be interpreted as a means of computing that value.
Because of this, most programming languages are strongly biased toward
unidirectional computations (computations with well-defined inputs and
outputs). There are, however, radically different programming languages
that relax this bias. We saw one such example in
section @sec:constraints, where the objects of
computation were arithmetic constraints. In a constraint system the
direction and the order of computation are not so well specified; in
carrying out a computation the system must therefore provide more detailed
"how to" knowledge than would be the case with an ordinary
arithmetic computation. This does not mean, however, that the user is
released altogether from the responsibility of providing imperative
knowledge. There are many constraint networks that implement the same set
of constraints, and the user must choose from the set of mathematically
equivalent networks a suitable network to specify a particular computation.

The nondeterministic program evaluator of
section @sec:nondeterministic-evaluation also moves
away from the view that programming is about constructing algorithms for
computing unidirectional functions. In a nondeterministic language,
expressions can have more than one value, and, as a result, the
computation is
dealing with
#idx("relations, computing in terms of")
relations rather than with single-valued functions. Logic
programming extends this idea by combining a relational vision of programming
with a powerful kind of symbolic pattern matching called
#emph[unification].#footnote[Logic programming has grown out of a long
#idx("logic programming", sub: "history of")
history of research in
#idx("theorem proving (automatic)")
automatic theorem proving. Early theorem-proving
programs could accomplish very little, because they exhaustively searched
the space of possible proofs. The major breakthrough that made such a
search plausible was the discovery in the early 1960s of the
#idx("unification", sub: "discovery of algorithm")
#emph[unification algorithm] and the
#idx("resolution principle")
#emph[resolution principle] (Robinson 1965).
Resolution was used, for example, by
#idx("Green, Cordell")
#idx("Raphael, Bertram")
Green and Raphael (1968) (see also Green 1969) as the
basis for a deductive question-answering system. During most of this period,
researchers concentrated on algorithms that are guaranteed to find a proof if
one exists. Such algorithms were difficult to control and to direct toward
a proof.
#idx("Hewitt, Carl Eddie")
Hewitt (1969) recognized the possibility of merging the control structure of
a programming language with the operations of a logic-manipulation system,
leading to the work in automatic search mentioned in
section @sec:amb
(footnote @foot:backtrack). At the same time that this
was being done,
#idx("Colmerauer, Alain")
Colmerauer, in Marseille, was developing rule-based systems for manipulating
natural language (see Colmerauer et al. 1973).
He invented a programming language called
#idx("Prolog")
Prolog for representing those rules.
#idx("Kowalski, Robert")
Kowalski (1973; 1979)
in Edinburgh, recognized that execution of a Prolog program could be
interpreted as proving theorems (using a proof technique called linear
#idx("resolution, Horn-clause")
Horn-clause resolution). The merging of the last two strands led to the
logic-programming movement. Thus, in assigning credit for the development
of logic programming, the French can point to Prolog's genesis at the
#idx("University of Marseille")
University of Marseille, while the British can highlight the work at the
#idx("University of Edinburgh")
University of Edinburgh. According to people at
#idx("MIT")
MIT, logic programming was developed by these groups in an attempt to figure
out what Hewitt was talking about in his brilliant but impenetrable Ph.D.
thesis. For a history of logic
programming, see
#idx("Robinson, J. A.")
Robinson 1983.]

This approach, when it works, can be a very
#idx("declarative vs. imperative knowledge", sub: "logic programming and")
#idx("imperative vs. declarative knowledge", sub: "logic programming and")
powerful way to write programs.
Part of the power comes from the fact that a single "what is"
fact can be used to solve a number of different problems that would have
different "how to" components. As an example, consider the
#idx("append", sub: "what is (rules) vs. how to (function)")
#py("append") operation, which takes two lists as
arguments and combines their elements to form a single list. In a procedural
language such as
Python,
we could define #py("append") in terms of the
basic list constructor
#py("pair"),
as we did in section @sec:sequences:

#snippet(```python
def append(x, y):
    return y if is_none(x) else pair(head(x), append(tail(x), y))
```)

This
function
can be regarded as a translation into
Python
of the following two rules, the first of which covers the case where the
first list is empty and the second of which handles the case of a nonempty
list, which is a
#py("pair")
of two parts:

- For any list #py("y"), the empty list and #py("y") #py("append") to form #py("y").
- For any #py("u"), #py("v"), #py("y"), and #py("z"), #py("pair(u, v)") and #py("y") #py("append") to form #py("pair(u, z)") if #py("v") and #py("y") #py("append") to form #py("z").#footnote[To see the correspondence between the rules and the function, let #py("x") in the function (where #py("x") is nonempty) correspond to #py("pair(u, v)") in the rule. Then #py("z") in the rule corresponds to the #py("append") of #py("tail(x)") and #py("y").]

Using the #py("append")
function,
we can answer questions such as

#blockquote[Find the #py("append") of
#py("llist(\"a\", \"b\")")
and
#py("llist(\"c\", \"d\")").]

But the same two rules are also sufficient for answering the following
sorts of questions, which the
function
can't answer:

#blockquote[Find a list #py("y")
that
#py("append")s with
#py("llist(\"a\", \"b\")")
to produce
\ #py("llist(\"a\", \"b\", \"c\", \"d\")").]

#blockquote[Find all #py("x") and #py("y")
that #py("append") to form
#py("llist(\"a\", \"b\", \"c\", \"d\")").]

In a
#idx("logic programming", sub: "logic programming languages")
#idx("programming language", sub: "logic")
logic programming language, the programmer writes an
#py("append")
"function"
by stating the two rules about #py("append") given
above.
#idx("append", sub: "what is (rules) vs. how to (function)")
"How to" knowledge is provided automatically by the
interpreter to allow this single pair of rules to be used to answer all
three types of questions about
#py("append").#footnote[This certainly does not
relieve the user of the entire problem of how to compute the answer. There
are many different mathematically equivalent sets of rules for formulating
the #py("append") relation, only some of which can be
turned into effective devices for computing in any direction. In addition,
sometimes "what is" information gives no clue
"how to" compute an answer. For example, consider the problem
of computing the $y$ such that
$y^(2) = x$.]

Contemporary logic programming languages (including the one we
implement here) have substantial deficiencies, in that their general
"how to" methods can lead them into spurious infinite loops or
other undesirable behavior. Logic programming is an active field of research
in computer science.#footnote[Interest in logic programming peaked
#idx("logic programming", sub: "history of")
#idx("logic programming", sub: "in Japan")
#idx("logic programming", sub: "computers for")
during the early 1980s when the Japanese government began an ambitious
project aimed at building superfast computers optimized to run logic
programming languages. The speed of such computers was to be measured
in LIPS (Logical Inferences Per Second) rather than the usual FLOPS
(FLoating-point Operations Per Second). Although the project
succeeded in developing hardware and software as originally planned,
the international computer industry moved in a different direction.
See
#idx("Feigenbaum, Edward")
#idx("Shrobe, Howard E.")
Feigenbaum and Shrobe 1993 for an overview evaluation
of the Japanese project. The logic programming community has also moved on
to consider relational programming based on techniques other than
simple pattern matching, such as the ability to deal with numerical
constraints such as the ones illustrated in the constraint-propagation
system of section @sec:constraints.]

#idx("declarative vs. imperative knowledge", sub: "logic programming and")
#idx("imperative vs. declarative knowledge", sub: "logic programming and")

Earlier in this chapter we explored the technology of implementing
interpreters and described the elements that are essential to an
interpreter for a
Python-like
language (indeed, to an interpreter for any conventional language). Now we
will apply these ideas to discuss an interpreter for a logic programming
language. We call this
language the
#idx("query language")
#emph[query language], because it is very useful for
retrieving information from data bases by formulating
#idx("query")
#emph[queries], or questions, expressed in the language. Even though the
query language is very different from
Python,
we will find it convenient to describe the language in terms of the same
general framework we have been using all along: as a collection of primitive
elements, together with means of combination that enable us to combine
simple elements to create more complex elements and means of abstraction
that enable us to regard complex elements as single conceptual units. An
interpreter for a logic programming language is considerably more complex
than an interpreter for a language like
Python.
Nevertheless, we will see
that our
#idx("query interpreter")
query-language interpreter contains many of the same elements
found in the interpreter of section @sec:mc-eval. In
particular, there will be an "evaluate" part that classifies
expressions according to type and an "apply" part that
implements the language's abstraction mechanism
(functions
in the case of
Python,
and #emph[rules] in the case of logic programming). Also, a central role
is played in the implementation by a frame data structure, which determines
the correspondence between symbols and their associated values. One
additional interesting aspect of our query-language implementation is
that we make substantial use of streams, which were introduced in
chapter @chap:state.

#idx("logic programming")

#include "../../chapter4/section4/subsection1.typ"

#include "../../chapter4/section4/subsection2.typ"

#include "../../chapter4/section4/subsection3.typ"

#include "../../chapter4/section4/subsection4.typ"
