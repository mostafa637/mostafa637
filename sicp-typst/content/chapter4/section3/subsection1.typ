// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Search and #py("amb")], label-name: <sec:amb>)

To extend
Python
to support nondeterminism, we introduce a new
syntactic form
#idx("amb", decl: true)
called #py("amb").#footnote[The idea of
#py("amb") for nondeterministic programming was
first described in 1961 by
#idx("McCarthy, John")
John McCarthy (see
McCarthy 1967).]
The expression
#py("amb(")$e_(1), space e_(2), dots.h , e_(n)$#py(")")
returns the value of one of the $n$ expressions
$e_(i)$ "ambiguously." For example,
the expression

#snippet(```python
llist(amb(1, 2, 3), amb("a", "b"))
```)

can have six possible values:

#snippet(```python
llist(1, "a") llist(1, "b") llist(2, "a")
llist(2, "b") llist(3, "a") llist(3, "b")
```)

An #py("amb") expression
with a single choice produces an ordinary (single) value.

An #py("amb") expression
with no choices—the expression
#py("amb()")—is
#idx("failure, in nondeterministic computation")
an expression with no acceptable values. Operationally, we can think of
#py("amb()")
as an expression that when evaluated causes the computation to
"fail": The computation aborts and no value is produced.
Using this idea, we can express the requirement that a particular predicate
expression #py("p") must be true as follows:

#idx("require", decl: true)
#snippet(```python
def require(p):
    if  not  p:
        amb()
    else:
        pass
```)

With #py("amb") and
#py("require"), we can implement the
#py("an_element_of") function
used above:

#idx("anelementof", decl: true)
#snippet(```python
def an_element_of(items):
    require( not is_none(items))
    return amb(head(items), an_element_of(tail(items)))
```)

An application of #py("an_element_of")
fails if the list is empty. Otherwise it ambiguously returns either the
first element of the list or an element chosen from the rest of the list.

We can also express infinite ranges of choices. The following
function
potentially returns any integer greater than or equal to some
given $n$:

#idx("anintegerstartingfrom", decl: true)
#snippet(```python
def an_integer_starting_from(n):
    return amb(n, an_integer_starting_from(n + 1))
```)

This is like the stream
function #py("integers_starting_from")
described in section @sec:infinite-streams, but with an
important difference: The stream
function
returns an object that represents the sequence of all integers beginning
with $n$, whereas the
#py("amb")
function
returns a single integer.#footnote[In actuality, the distinction between
nondeterministically returning a single choice and returning all choices
depends somewhat on our point of view. From the perspective of the code
that uses the value, the nondeterministic choice returns a single value.
From the perspective of the programmer designing the code, the
nondeterministic choice potentially returns all possible values, and the
computation branches so that each value is investigated
separately.]

Abstractly, we can imagine that evaluating an
#py("amb") expression causes
#idx("time", sub: "in nondeterministic computing")
time to split into
branches, where the computation continues on each branch with one of the
possible values of the expression. We say that
#py("amb") represents a
#idx("nondeterministic choice point")
#emph[nondeterministic choice point]. If we had a machine with a
sufficient number of processors that could be dynamically allocated, we
could implement the search in a straightforward way. Execution would
proceed as in a sequential machine, until an #py("amb")
expression is encountered. At this point, more processors would be allocated
and initialized to continue all of the parallel executions implied by the
choice. Each processor would proceed sequentially as if it were the only
choice, until it either terminates by encountering a failure, or it further
subdivides, or it finishes.#footnote[One might object that this is a
hopelessly inefficient mechanism. It might require millions of processors
to solve some easily stated problem this way, and most of the time most
of those processors would be idle. This objection should be taken in
the context of history. Memory used to be considered just such an
expensive commodity.
#idx("memory", sub: "in 1965")
In 1965 a megabyte of RAM cost about \$400,000. Now every personal
computer has many gigabytes of RAM, and most of the time most of that RAM is
unused. It is hard to underestimate the cost of mass-produced
electronics.]

On the other hand, if we have a machine that can execute only one process
(or a few concurrent processes), we must consider the alternatives
#idx("failure, in nondeterministic computation", sub: "searching and")
sequentially. One could imagine modifying an evaluator to pick at random a
branch to follow whenever it encounters a choice point. Random choice,
however, can easily lead to failing values. We might try running the
evaluator over and over, making random choices and hoping to find a
non-failing value, but it is better to
#idx("systematic search")
#idx("search", sub: "systematic")
#emph[systematically search] all possible execution paths. The
#py("amb") evaluator that we will develop and work
with in this section implements a systematic search as follows: When the
evaluator encounters an application of #py("amb"), it
initially selects the first alternative. This selection may itself lead to
a further choice. The evaluator will always initially choose the first
alternative at each choice point. If a choice results in a failure, then
the evaluator
#idx("automagically")
automagically#footnote[Automagically: "Automatically, but in a way which, for some reason (typically because it is too complicated, or too ugly, or perhaps even too trivial), the speaker doesn't feel like explaining."
#idx("Steele, Guy Lewis Jr.")
(Steele 1983,
#idx("Raymond, Eric")
Raymond 1996)]
#idx("backtracking")
#emph[backtracks] to the most recent choice point and tries the next
alternative. If it runs out of alternatives at any choice point, the
evaluator will back up to the previous choice point and resume from there.
This process leads to a search strategy known as
#idx("depth-first search")
#idx("search", sub: "depth-first")
#emph[depth-first search] or
#idx("chronological backtracking")
#emph[chronological backtracking].#footnote[The integration of
#idx("automatic search", sub: "history of")
automatic search strategies
into programming languages has had a long and checkered history. The first
suggestions that nondeterministic algorithms might be elegantly encoded in a
programming language with search and automatic backtracking came from
#idx("Floyd, Robert")
Robert Floyd (1967).
#idx("Hewitt, Carl Eddie")
Carl Hewitt (1969) invented a programming language called
#idx("Planner")
Planner that explicitly supported automatic chronological backtracking,
providing for a built-in depth-first search strategy.
#idx("Sussman, Gerald Jay")
#idx("Winograd, Terry")
#idx("Charniak, Eugene")
Sussman, Winograd, and Charniak (1971) implemented a subset of this language,
called
#idx("MicroPlanner")
MicroPlanner, which was used to support work in problem solving and robot
planning. Similar ideas, arising from logic and theorem proving, led to the
genesis in Edinburgh and Marseille of the elegant language
#idx("Prolog")
Prolog (which we will discuss in
section @sec:logic-programming). After sufficient
frustration with automatic search,
#idx("McDermott, Drew")
#idx("Sussman, Gerald Jay")
McDermott and Sussman (1972) developed a language called
#idx("Conniver")
Conniver, which included mechanisms for placing the search strategy under
programmer control. This proved unwieldy, however, and
#idx("Sussman, Gerald Jay")
#idx("Stallman, Richard M.")
Sussman and Stallman (1975) found a more tractable approach while
investigating methods of symbolic analysis for electrical circuits. They
developed a nonchronological backtracking scheme that was based on tracing
out the logical dependencies connecting facts, a technique that has come to
be known as
#idx("dependency-directed backtracking")
#emph[dependency-directed backtracking]. Although their method was
complex, it produced reasonably efficient programs because it did little
redundant search.
#idx("Doyle, Jon")
Doyle (1979) and
#idx("McAllester, David Allen")
McAllester (1978, 1980)
generalized and clarified the methods of Stallman and Sussman, developing a
new paradigm for formulating search that is now called
#idx("truth maintenance")
#emph[truth maintenance].
Many problem-solving systems
use some form of truth-maintenance system as a substrate. See
#idx("Forbus, Kenneth D.")
#idx("de Kleer, Johan")
Forbus and de Kleer 1993 for a discussion of elegant
ways to build truth-maintenance systems and applications using truth
maintenance.
#idx("Zabih, Ramin")
#idx("McAllester, David Allen")
#idx("Chapman, David")
Zabih, McAllester, and Chapman 1987 describes a
#idx("Scheme", sub: "nondeterministic extension of")
nondeterministic extension to Scheme that is based on
#py("amb"); it is similar to the interpreter described
in this section, but more sophisticated, because it uses dependency-directed
backtracking rather than chronological
backtracking.
#idx("Winston, Patrick Henry")
Winston 1992 gives an introduction to
both kinds of backtracking.]<foot:backtrack>

#subheading([Driver loop])

The
#idx("driver loop", sub: "in nondeterministic evaluator")
driver loop for the #py("amb") evaluator has some
unusual properties. It reads
a program
and prints the value of the
first non-failing execution, as in the
#py("prime_sum_pair")
example shown above. If we want to see the value of the next successful
execution, we can ask the interpreter to backtrack and attempt to generate a
second non-failing execution.

This is signaled by typing
#idx("retry", decl: true)
#py("retry").
If any other input except #py("retry")
is given, the interpreter will start a new problem, discarding the
unexplored alternatives in the previous problem.

Here is a sample interaction:

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
prime_sum_pair(llist(1, 3, 5, 8), llist(20, 35, 110))
```)

#output(```python
prime_sum_pair(llist(1, 3, 5, 8), llist(20, 35, 110))
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
retry
```)

#output(```python
retry
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
retry
```)

#output(```python
retry
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
retry
```)

#output(```python
retry
```)

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
prime_sum_pair(llist(19, 27, 30), llist(11, 36, 58))
```)

#output(```python
prime_sum_pair(llist(19, 27, 30), llist(11, 36, 58))
```)

#exercise(label-name: <ex:amb-pythag-triples>, [
Write a
function #py("an_integer_between")
that returns an integer between two given bounds. This can be used to
implement a
function
that finds
#idx("Pythagorean triples", sub: "with nondeterministic programs")
#idx("nondeterministic programs", sub: "Pythagorean triples")
Pythagorean triples, i.e., triples of integers
$(i,j,k)$ between the given bounds such
that $i lt.eq j$ and
$i^(2) + j^(2) =k^(2)$, as follows:

#snippet(```python
def a_pythogorean_triple_between(low, high):
    i = an_integer_between(low, high)
    j = an_integer_between(i, high)
    k = an_integer_between(j, high)
    require(i * i + j * j == k * k)
    return llist(i, j, k)
```)
])

#exercise(label-name: <ex:amb-pythag-triples_2>, [
Exercise @ex:stream-pythagorean-triples discussed how to
generate the stream of #emph[all]
#idx("Pythagorean triples", sub: "with nondeterministic programs")
#idx("nondeterministic programs", sub: "Pythagorean triples")
Pythagorean triples, with no upper bound
on the size of the integers to be searched. Explain why simply replacing
#py("an_integer_between")
by
#py("an_integer_starting_from")
in the
function
in
exercise @ex:amb-pythag-triples is not an adequate way to
generate arbitrary Pythagorean triples. Write a
function
that actually will accomplish this. (That is, write a
function
for which repeatedly typing
#py("retry")
would in principle eventually generate all Pythagorean triples.)
])

#exercise(label-name: <ex:amb-pythag-triples_3>, [
Ben Bitdiddle claims that the following method for generating
#idx("Pythagorean triples", sub: "with nondeterministic programs")
#idx("nondeterministic programs", sub: "Pythagorean triples")
Pythagorean
triples is more efficient than the one in
exercise @ex:amb-pythag-triples. Is he correct?
(Hint: Consider the number of possibilities that must be explored.)

#snippet(```python
def a_pythagorean_triple_between(low, high):
    i = an_integer_between(low, high)
    hsq = high * high
    j = an_integer_between(i, high)
    ksq = i * i + j * j
    require(hsq >= ksq)
    k = math_sqrt(ksq)
    require(is_integer(k))
    return llist(i, j, k)
```)
])
