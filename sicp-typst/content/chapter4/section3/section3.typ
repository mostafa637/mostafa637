// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Nondeterministic Computing], label-name: <sec:nondeterministic-evaluation>)

#idx("nondeterministic computing")

In this section, we extend the
Python
evaluator to support a
programming paradigm called #emph[nondeterministic computing] by
building into the evaluator a facility to support
#idx("automatic search")
automatic search.
This is a much more profound change to the language than the
introduction of lazy evaluation in
section @sec:lazy-evaluation.

Nondeterministic computing, like stream processing, is useful for
#idx("nondeterministic programming vs. Python programming")
"generate and test" applications. Consider the task of
starting with two lists of positive integers and finding a pair of
integers—one from the first list and one from the second
list—whose sum is prime. We saw how to handle this with finite
sequence operations in section @sec:nested-mappings and
with infinite streams in section @sec:exploiting-streams.
Our approach was to generate the sequence of all possible pairs and filter
these to select the pairs whose sum is prime. Whether we actually generate
the entire sequence of pairs first as in chapter @chap:data, or interleave the
generating and filtering as in chapter @chap:state, is immaterial to the
essential image of how the computation is organized.

The nondeterministic approach evokes a different image. Imagine simply
that we choose (in some way) a number from the first list and a number
from the second list and require (using some mechanism) that their
#idx("nondeterministic programs", sub: "pairs with prime sums")
sum be prime. This is expressed by the following
function:
#idx("primesumpair", decl: true)
#snippet(```python
def prime_sum_pair(list1, list2):
    a = an_element_of(list1)
    b = an_element_of(list2)
    require(is_prime(a + b))
    return llist(a, b)
```)

It might seem as if this
function
merely restates the problem,
rather than specifying a way to solve it. Nevertheless, this is a
legitimate nondeterministic program.#footnote[We assume that we have
previously defined a
function
#py("is_prime")
that tests whether numbers are prime. Even with
#py("is_prime")
defined, the
#py("prime_sum_pair")
function
may look suspiciously like the unhelpful
"pseudo-Python"
attempt to define the square-root function, which we described at the
beginning of section @sec:sqrt. In fact, a square-root
function
along those lines can actually be formulated as a nondeterministic program.
By incorporating a search mechanism into the evaluator, we are eroding the
#idx("declarative vs. imperative knowledge", sub: "nondeterministic computing and")
#idx("imperative vs. declarative knowledge", sub: "nondeterministic computing and")
distinction between purely declarative descriptions and imperative
specifications of how to compute answers. We'll go even farther in
this direction in
section @sec:logic-programming.]

The key idea here is that
components
in a nondeterministic language
can have more than one possible value. For instance,
#py("an_element_of")
might return any element of the given list. Our nondeterministic program
evaluator will work by automatically choosing a possible value and keeping
track of the choice. If a subsequent requirement is not met, the evaluator
will try a different choice, and it will keep trying new choices until the
evaluation succeeds, or until we run out of choices. Just as the lazy
evaluator freed the programmer from the details of how values are delayed
and forced, the nondeterministic program evaluator will free the programmer
from the details of how choices are made.

It is instructive to contrast the different images of
#idx("time", sub: "in nondeterministic computing")
time evoked by
nondeterministic evaluation and stream processing. Stream processing
uses lazy evaluation to decouple the time when the stream of possible
answers is assembled from the time when the actual stream elements are
produced. The evaluator supports the illusion that all the possible
answers are laid out before us in a timeless sequence. With
nondeterministic evaluation,
a component
represents the exploration
of a set of possible worlds, each determined by a set of choices.
Some of the possible worlds lead to dead ends, while others have
useful values. The nondeterministic program evaluator supports the
illusion that time branches, and that our programs have different
possible execution histories. When we reach a dead end, we can
revisit a previous choice point and proceed along a different branch.

The nondeterministic program evaluator implemented below is called the
#py("amb") evaluator because it is based on
a new syntactic form
called #py("amb"). We can type the above
declaration
of
#py("prime_sum_pair")
at the #py("amb") evaluator driver loop (along with
declarations
of
#py("is_prime"),
#py("an_element_of"),
and #py("require")) and run the
function
as follows:

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
prime_sum_pair(llist(1, 3, 5, 8), llist(20, 35, 110))
```)

#output(```python
prime_sum_pair(llist(1, 3, 5, 8), llist(20, 35, 110))
```)

The value returned was obtained after the evaluator repeatedly chose
elements from each of the lists, until a successful choice was made.

Section @sec:amb introduces
#py("amb") and explains how it supports nondeterminism
through the evaluator's automatic search mechanism.
Section @sec:amb-examples presents examples of
nondeterministic programs, and
section @sec:amb-implementation gives the details of how
to implement the #py("amb") evaluator by modifying the
ordinary
Python
evaluator.

#include "../../chapter4/section3/subsection1.typ"

#include "../../chapter4/section3/subsection2.typ"

#include "../../chapter4/section3/subsection3.typ"
