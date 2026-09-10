// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Is Logic Programming Mathematical Logic?], label-name: <sec:math-logic>)

#idx("query language", sub: "mathematical logic vs.")
#idx("logic programming", sub: "mathematical logic vs.")

The means of combination used in the query language may at first seem
identical to the operations #py("and"),
#py("or"), and #py("not") of
mathematical logic, and the application of query-language rules is in
fact accomplished through a legitimate method of
#idx("inference, method of")
inference.#footnote[That a particular method of inference is
legitimate is not a trivial assertion. One must prove that if one
starts with true premises, only true conclusions can be derived. The
method of inference represented by rule applications is
#idx("modus ponens", sort: "modus")
#emph[modus ponens],
the familiar method of inference that says that if #emph[A] is
true and #emph[A implies B] is true, then we may conclude that #emph[B]
is true.] This identification of the query language with
mathematical logic is not really valid, though, because the query language
provides a
#idx("control structure")
#emph[control structure] that interprets the logical statements
procedurally. We can often take advantage of this control structure.
For example, to find all of the supervisors of programmers we could
formulate a query in either of two logically equivalent forms:

#snippet(```python
and(job($x, llist("computer", "programmer")),
    supervisor($x, $y))
```)

or

#snippet(```python
and(supervisor($x, $y),
    job($x, llist("computer", "programmer")))
```)

If a company has
#idx("bureaucracy")
many more supervisors than programmers,
it is better to use the first form rather than the second,
because the data base must be scanned for each intermediate result
(frame) produced by the first clause of the #py("and").

The aim of logic programming is to provide the programmer with
techniques for decomposing a computational problem into two separate
problems:
#idx("declarative vs. imperative knowledge", sub: "logic programming and")
#idx("imperative vs. declarative knowledge", sub: "logic programming and")
"what" is to be computed, and "how" this
should be computed. This is accomplished by selecting a subset of the
statements of mathematical logic that is powerful enough to be able to
describe anything one might want to compute, yet weak enough to have a
controllable procedural interpretation. The intention here is that,
on the one hand, a program specified in a logic programming language
should be an effective program that can be carried out by a computer.
Control ("how" to compute) is effected by using the order of
evaluation of the language. We should be able to arrange the order of
clauses and the order of subgoals within each clause so that the
computation is done in an order deemed to be effective and efficient.
At the same time, we should be able to view the result of the
computation ("what" to compute) as a simple consequence of the
laws of logic.

Our query language can be regarded as just such a procedurally
interpretable subset of mathematical logic. An assertion represents a
simple fact (an atomic proposition). A rule represents the
implication that the rule conclusion holds for those cases where the
rule body holds. A rule has a natural procedural interpretation: To
establish the conclusion of the rule, establish the body of the rule.
Rules, therefore, specify computations. However, because rules can
also be regarded as statements of mathematical logic, we can justify any
"inference" accomplished by a logic program by asserting that
the same result could be obtained by working entirely within
mathematical logic.#footnote[We must qualify this statement by
agreeing that, in speaking of the "inference" accomplished
by a logic program, we assume that the computation terminates.
Unfortunately, even this qualified statement is false for our
implementation of the query language (and also false for programs in
Prolog and most other current logic programming languages) because of
our use of #py("not") and
#py("javascript_predicate").
As we will describe below, the #py("not") implemented
in the query language is not always consistent with the
#py("not") of mathematical logic, and
#py("javascript_predicate")
introduces additional complications. We could implement a language
consistent with mathematical logic by simply removing
#py("not") and
#py("javascript_predicate")
from the language and agreeing to write programs using only simple queries,
#py("and"), and #py("or").
However, this would greatly restrict the expressive power of the language.
One of the major concerns of research in logic programming was to find ways
to achieve more consistency with mathematical logic without unduly
sacrificing expressive power.]

#subheading([Infinite loops])

#idx("query interpreter", sub: "infinite loops")

A consequence of the procedural interpretation of logic programs is
that it is possible to construct hopelessly inefficient programs for
solving certain problems. An extreme case of inefficiency occurs when
the system falls into infinite loops in making deductions. As a
simple example, suppose we are setting up a data base of famous
marriages, including
#idx("Mouse, Minnie and Mickey")
#snippet(```python
assert(married("Minnie", "Mickey"))
```)

If we now ask

#snippet(```python
married("Mickey", $who)
```)

we will get no response, because the system doesn't know that if
$A$ is married to $B$,
then $B$ is married to
$A$. So we assert the rule

#snippet(```python
assert(rule(married($x, $y),
            married($y, $x)))
```)

and again query

#snippet(```python
married("Mickey", $who)
```)

Unfortunately, this will drive the system into an infinite loop, as
follows:

- The system finds that the #py("married") rule is applicable; that is, the rule conclusion #py("married($x, $y)") unifies with the query pattern #py("married(\"Mickey\", $who)") to produce a frame in which #py("$x") is bound to #py("\"Mickey\"") and #py("$y") is bound to #py("$who"). So the interpreter proceeds to evaluate the rule body #py("married($y, $x)") in this frame—in effect, to process the query #py("married($who, \"Mickey\")").
- One answer, #py("married(\"Minnie\", \"Mickey\")"), appears directly as an assertion in the data base.
- The #py("married") rule is also applicable, so the interpreter again evaluates the rule body, which this time is equivalent to #py("married(\"Mickey\", $who)").

The system is now in an infinite loop. Indeed, whether the system
will find the simple answer
#py("married(\"Minnie\", \"Mickey\")")
before it goes into the loop depends on implementation details concerning the
order in which the system checks the items in the data base. This is a very
simple example of the kinds of loops that can occur. Collections of
interrelated rules can lead to loops that are much harder to anticipate, and
the appearance of a loop can depend on the order of clauses in an
#py("and") (see
exercise @ex:query-simple-loop) or on low-level details
concerning the order in which the system processes queries.#footnote[This is
not a problem of the logic but one of the procedural interpretation of the
logic provided by our interpreter. We could write an interpreter that would
not fall into a loop here. For example, we could enumerate all the proofs
derivable from our assertions and our rules in a breadth-first rather than a
depth-first order. However, such a system makes it more difficult to take
advantage of the order of deductions in our programs. One attempt to
build sophisticated control into such a program is described in
#idx("de Kleer, Johan")
de Kleer et al. 1977.
Another technique, which does not lead to such serious control problems, is
to put in special knowledge, such as detectors for particular kinds of loops
(exercise @ex:query-loop-detector). However, there can
be no general scheme for reliably preventing a system from going down
infinite paths in performing deductions. Imagine a diabolical rule of
the form "To show $P(x)$ is true, show that $P(f(x))$ is true," for some suitably
chosen function $f$.]

#idx("query interpreter", sub: "infinite loops")

#subheading([Problems with #py("not")])

#idx("query interpreter", sub: "problems with not and javascriptpredicate")

Another quirk in the query system concerns
#idx("not (query language)")
#py("not").
Given the data base of
section @sec:deductive-info-retrieval, consider the
following two queries:

#snippet(```python
and(supervisor($x, $y),
    not(job($x, llist("computer", "programmer"))))

and(not(job($x, llist("computer", "programmer"))),
    supervisor($x, $y))
```)

These two queries do not produce the same result. The first query
begins by finding all entries in the data base that match
#py("supervisor($x, $y)"),
and then filters the resulting frames by removing the ones in which the
value of
#py("$x")
satisfies
#py("job($x,llist(\"computer\", \"programmer\"))").
The second query begins by filtering the
incoming frames to remove those that can satisfy
#py("job($x, llist(\"computer\",\"programmer\"))").
Since the only incoming frame is empty, it checks the data base
for
patterns that satisfy
#py("job($x, llist(\"computer\", \"programmer\"))").
Since there generally are entries of this form, the
#py("not") clause filters out the empty frame and
returns an empty stream of frames. Consequently, the entire compound query
returns an empty stream.

The trouble is that our implementation of #py("not")
really is meant to serve as a filter on values for the variables. If a
#py("not") clause is processed with a frame in which
some of the variables remain unbound (as does
#py("$x")
in the example above), the system will produce unexpected results. Similar
problems occur with the use of
#idx("javascriptpredicate (query language)")
#py("javascript_predicate")—the
Python predicate can't work if some of its variables are unbound.
See exercise @ex:not-query-filter.

There is also a much more serious way in which the
#py("not") of the query language differs from the
#py("not") of mathematical logic. In logic, we
interpret the statement "not $P$" to
mean that $P$ is not true. In the query system,
however, "not $P$" means that
$P$ is not deducible from the knowledge in the
data base. For example, given the personnel data base of
section @sec:deductive-info-retrieval, the system would
happily deduce all sorts of #py("not") statements,
such as that Ben Bitdiddle is not a baseball fan, that it is not raining
outside, and that $2 + 2$
is not 4.#footnote[Consider the query
#py("not(baseball_fan(llist(\"Bitdiddle\", \"Ben\")))").
The system finds that
#py("baseball_fan(llist(\"Bitdiddle\", \"Ben\"))")
is not in the data base, so the empty frame does not satisfy the pattern and
is not filtered out of the initial stream of frames. The result of the
query is thus the empty frame, which is used to instantiate the input query
to produce
#py("not(baseball_fan(llist(\"Bitdiddle\", \"Ben\")))").] In other
words, the #py("not") of logic programming languages
reflects the so-called
#idx("closed world assumption")
#emph[closed world assumption] that all relevant information has been
included in the data base.#footnote[A discussion and justification of this
treatment of #py("not") can be found in the article
#idx("negation as failure")
"Negation as Failure" by
#idx("Clark, Keith L.")
Clark (1978).]

#idx("query interpreter", sub: "problems with not and javascriptpredicate")

#exercise(label-name: <ex:query-simple-loop>, [
Louis Reasoner mistakenly deletes the
#idx("outrankedby (rule)")
#py("outranked_by")
rule (section @sec:deductive-info-retrieval) from the
data base. When he realizes this, he quickly reinstalls it. Unfortunately,
he makes a slight change in the rule, and types it in as

#snippet(```python
rule(outranked_by($staff_person, $boss),
     or(supervisor($staff_person, $boss),
        and(outranked_by($middle_manager, $boss),
            supervisor($staff_person, $middle_manager))))
```)

Just after Louis types this information into the system, DeWitt
Aull comes by to find out who outranks Ben Bitdiddle. He issues
the query

#snippet(```python
outanked_by(llist("Bitdiddle", "Ben"), $who)
```)

After answering, the system goes into an infinite loop. Explain why.
])

#exercise(label-name: <ex:multiple-query>, [
Cy D. Fect, looking forward to the day when he will rise in the
organization, gives a query to find all the wheels (using the
#idx("wheel (rule)")
#py("wheel") rule of
section @sec:deductive-info-retrieval):

#snippet(```python
wheel($who)
```)

To his surprise, the system responds

#output(```python
wheel($who)
```)

Why is Oliver Warbucks listed four times?
])

#exercise(label-name: <ex:4_64>, [
Ben has been
#idx("query language", sub: "extensions to")
generalizing the query system to provide statistics about the
company. For example, to find the total salaries of all the computer
programmers one will be able to say

#snippet(```python
sum($amount,
    and(job($x, llist("computer", "programmer")),
        salary($x, $amount)))
```)

In general, Ben's new system allows expressions of the form

#syntax("
accumulation_function(", meta("variable"), ",
                      ", meta("query"), "-", meta("pattern"), ")
      ")

where
#py("accumulation_function")
can be things like #py("sum"),
#py("average"), or
#py("maximum").
Ben reasons that it should be a cinch to implement this. He will simply
feed the query pattern to
#py("evaluate_query").
This will produce a stream of frames. He will then pass this stream through
a mapping function that extracts the value of the designated variable from
each frame in the stream and feed the resulting stream of values to the
accumulation function. Just as Ben completes the implementation and is
about to try it out, Cy walks by, still puzzling over the
#py("wheel") query result in
exercise @ex:multiple-query. When Cy shows Ben the
system's response, Ben groans, "Oh, no, my simple accumulation scheme won't work!"

What has Ben just realized? Outline a method he can use to salvage the
situation.
])

#exercise(label-name: <ex:query-loop-detector>, [
Devise a
#idx("query interpreter", sub: "improvements to")
#idx("query interpreter", sub: "infinite loops")
way to install a loop detector in the query system so as to
avoid the kinds of simple loops illustrated in the text and in
exercise @ex:query-simple-loop. The general idea is
that the system should maintain some sort of history of its current chain of
deductions and should not begin processing a query that it is already
working on. Describe what kind of information (patterns and frames)
is included in this history, and how the check should be made. (After
you study the details of the query-system implementation in
section @sec:implementing-the-query-system, you may
want to modify the system to include your loop detector.)
])

#exercise(label-name: <ex:4_66>, [
Define rules to implement the
#idx("reverse", sub: "rules")
#py("reverse") operation
of exercise @ex:reverse, which returns a list containing
the same elements as a given list
in reverse order.
(Hint: Use
#py("append_to_form").)
Can your rules answer both
the query #py("reverse(llist(1, 2, 3), $x)")
and the query #py("reverse($x, llist(1, 2, 3))")?
])

#exercise(label-name: <ex:great-grandson>, [
Let us modify the data base and the rules of
exercise @ex:genesis to add
"great" to a grandson relationship. This should enable the
system to deduce that Irad is the great-grandson of Adam, or that Jabal
and Jubal are the great-great-great-great-great-grandsons of Adam.

+ Change the assertions in the data base such that there is only one kind of relationship information, namely #py("related"). The first item then describes the relationship. Thus, instead of #py("son(\"Adam\", \"Cain\")"), you would write #py("related(\"son\", \"Adam\", \"Cain\")"). Represent the fact about Irad, for example, as #snippet(```python related(llist("great", "grandson"), "Adam", "Irad") ```)
+ Write rules that determine if a list ends in the word #py("\"grandson\"").
+ Use this to express a rule that allows one to derive the relationship #snippet(```python llist(pair("great", $rel), $x, $y) ```) where #py("$rel") is a list ending in #py("\"grandson\"").
+ Check your rules on the queries #py("related(list(\"great\", \"grandson\"), $g, $ggs)") and #py("related($relationship, \"Adam\", \"Irad\")").
])

#idx("query language", sub: "mathematical logic vs.")
#idx("logic programming", sub: "mathematical logic vs.")
