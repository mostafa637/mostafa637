// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([How the Query System Works], label-name: <sec:how-query-works>)

#idx("query interpreter", sub: "overview")

In section @sec:implementing-the-query-system we will
present an implementation of the query interpreter as a collection of
functions.
In this section we give an overview that explains the general
structure of the system independent of low-level implementation
details. After describing the implementation of the interpreter, we
will be in a position to understand some of its limitations and some
of the subtle ways in which the query language's logical operations
differ from the operations of mathematical logic.

It should be apparent that the query evaluator must perform some kind
of search in order to match queries against facts and rules in the
data base. One way to do this would be to implement the query system
as a nondeterministic program, using the #py("amb")
evaluator of section @sec:nondeterministic-evaluation
(see exercise @ex:query-lang-amb). Another possibility
is to manage the search with the aid of streams. Our implementation follows
this second approach.

The query system is organized around two central operations, called
#emph[pattern matching] and #emph[unification]. We first describe
pattern matching and explain how this operation, together with the
organization of information in terms of streams of frames, enables us
to implement both simple and compound queries. We next discuss
unification, a generalization of pattern matching needed to implement
rules. Finally, we show how the entire query interpreter fits
together through a
function
that classifies
queries
in a manner analogous to the way
#py("evaluate")
classifies expressions for the interpreter described in
section @sec:mc-eval.

#subheading([Pattern matching])

#idx("query interpreter", sub: "pattern matching")
#idx("pattern matching")

A #emph[pattern matcher] is a program that tests whether some datum
fits a specified pattern. For example, the
datum #py("llist(llist(\"a\", \"b\"), \"c\", llist(\"a\", \"b\"))")
matches the pattern
#py("llist($x, \"c\", $x)")
with the pattern variable
#py("$x")
bound to
#py("llist(\"a\", \"b\")").
The same
data list
matches the pattern
#py("llist($x, $y, $z)")
with
#py("$x")
and
#py("$z")
both bound to
#py("llist(\"a\", \"b\")")
and
#py("$y")
bound to
#py("\"c\"").
It also matches the pattern
#py("llist(llist($x, $y), \"c\", llist($x, $y))")
with
#py("$x")
bound to
#py("\"a\"")
and
#py("$y")
bound to
#py("\"b\"").
However, it does not match the pattern
#py("llist($x, \"a\", $y)"),
since that pattern specifies a list whose second element is the
string #py("\"a\"").

The pattern matcher used by the query system takes as inputs a
pattern, a datum, and a
#idx("query interpreter", sub: "frame")
#idx("frame (query interpreter)")
#emph[frame] that specifies bindings for
various pattern variables. It checks whether the datum matches the
pattern in a way that is consistent with the bindings already in the
frame. If so, it returns the given frame augmented by any bindings
that may have been determined by the match. Otherwise, it indicates
that the match has failed.

Using the pattern
#py("llist($x, $y, $x)")
to match
#py("llist(\"a\", \"b\", \"a\")")
given an empty frame, for example,
will return a frame specifying that
#py("$x")
is bound to
#py("\"a\"")
and
#py("$y")
is bound to
#py("\"b\"").
Trying the match with the same pattern, the same datum, and a frame
specifying that
#py("$y")
is bound to
#py("\"a\"")
will fail. Trying the match with the same pattern, the same datum, and a
frame in which
#py("$y")
is bound to
#py("\"b\"")
and
#py("$x")
is unbound will return the given frame augmented by a binding of
#py("$x")
to #py("\"a\"").

The pattern matcher is all the mechanism that is needed to process
#idx("simple query", sub: "processing")
simple
queries that don't involve rules. For instance, to process the query

#snippet(```python
job($x, llist("computer", "programmer"))
```)

we scan through all assertions in the data base and select those that
match the pattern with respect to an initially empty frame. For each
match we find, we use the frame returned by the match to instantiate
the pattern with a value for
#py("$x").

#idx("query interpreter", sub: "pattern matching")
#idx("pattern matching")

#subheading([Streams of frames])

The testing of patterns against frames is organized through the use of
#idx("query interpreter", sub: "streams of frames")
#idx("stream(s)", sub: "used in query interpreter")
streams. Given a single frame, the matching process runs through the
data-base entries one by one. For each data-base entry, the matcher
generates either a special symbol indicating that the match has failed
or an extension to the frame. The results for all the data-base
entries are collected into a stream, which is passed through a filter
to weed out the failures. The result is a stream of all the frames
that extend the given frame via a match to some assertion in the data
base.#footnote[Because matching is generally very
#idx("efficiency", sub: "of data-base access")
expensive, we would
like to avoid applying the full matcher to every element of the data
base. This is usually arranged by breaking up the process into a
fast, coarse match and the final match. The coarse match filters the
data base to produce a small set of candidates for the final match.
With care, we can arrange our data base so that some of the work of
coarse matching can be done when the data base is constructed rather
then when we want to select the candidates. This is called
#idx("data base", sub: "indexing")
#idx("indexing a data base")
#emph[indexing] the data base. There is a vast technology built around
data-base-indexing schemes. Our implementation, described in
section @sec:implementing-the-query-system, contains a
simpleminded form of such an optimization.]

In our system, a query takes an input stream of frames and performs
the above matching operation for every frame in the stream, as
indicated in
figure @fig:query-stream.
That is, for
each frame in the input stream, the query generates a new stream consisting
of all extensions to that frame by matches to assertions in the data base.
All these streams are then combined to form one huge stream, which contains
all possible extensions of every frame in the input stream. This stream is
the output of the query.

#sicp-figure(image("/images/img_javascript/Fig4.4a.std.svg", width: 70%), caption: [A query processes a stream of frames.], label-name: <fig:query-stream>)

To answer a
#idx("simple query", sub: "processing")
simple query, we use the query with an input stream
consisting of a single empty frame. The resulting output stream
contains all extensions to the empty frame (that is, all answers to
our query). This stream of frames is then used to generate a stream
of copies of the original query pattern with the variables
instantiated by the values in each frame, and this is the stream that
is finally printed.

#subheading([Compound queries])

#idx("compound query", sub: "processing")

The real elegance of the stream-of-frames implementation is evident
when we deal with compound queries. The processing of compound
queries makes use of the ability of our matcher to demand that a match
be consistent with a specified frame. For example, to handle the
#idx("and (query language)", sub: "evaluation of")
#py("and") of two queries, such as

#snippet(```python
and(can_do_job($x, llist("computer", "programmer", "trainee")),
    job($person, $x))
```)

(informally, "Find all people who can do the job of a computer programmer trainee"), we first find all entries that match the
pattern

#snippet(```python
can_do_job($x, llist("computer", "programmer", "trainee"))
```)

This produces a stream of frames, each of which contains a binding for
#py("$x").
Then for each frame in the stream we find all entries that
match

#snippet(```python
job($person, $x)
```)

in a way that is consistent with the given binding for
#py("$x").
Each such match will produce a frame containing bindings for
#py("$x")
and
#py("$person").
The #py("and") of two queries can be viewed as a series
combination of the two component queries, as shown in
figure @fig:query-and.
The frames that pass through the
first query filter are filtered and further extended by the second query.

#sicp-figure(image("/images/img_javascript/Fig4.5a.std.svg", width: 70%), caption: [The #py("and") combination of two queries is produced by operating on the stream of frames in series.], label-name: <fig:query-and>)

Figure @fig:query-or
shows the analogous method for
computing the
#idx("or (query language)", sub: "evaluation of")
#py("or") of two queries as a parallel
combination of the two component queries. The input stream of frames is
extended separately by each query. The two resulting streams are then
merged to produce the final output stream.

#sicp-figure(image("/images/img_javascript/Fig4.6a.std.svg", width: 70%), caption: [The #py("or") combination of two queries is produced by operating on the stream of frames in parallel and merging the results.], label-name: <fig:query-or>)

Even from this high-level description, it is apparent that the
processing of compound queries can be slow.
#idx("efficiency", sub: "of query processing")
For example, since a query may produce more than one output frame for each
input frame, and each query in an #py("and") gets its
input frames from the previous query, an #py("and")
query could, in the worst case, have to perform a number of matches that is
exponential in the number of queries (see
exercise @ex:q-exponential-and).#footnote[But this kind
of exponential explosion is not common in #py("and")
queries because the added conditions tend to reduce rather than expand
the number of frames produced.]
Though systems for handling only simple queries are quite practical, dealing
with complex queries is extremely difficult.#footnote[There is a large
literature on data-base-management systems that is concerned with how to
handle complex queries efficiently.]

From the stream-of-frames viewpoint, the
#idx("not (query language)", sub: "evaluation of")
#py("not") of
some query acts as a filter that removes all frames for which the query can
be satisfied. For instance, given the pattern

#snippet(```python
not(job($x, llist("computer", "programmer")))
```)

we attempt, for each frame in the input stream, to produce extension
frames that satisfy
#py("job($x, llist(\"computer\", \"programmer\"))").
We remove from the input stream all frames for which such extensions exist.
The result is a stream consisting of only those frames in which the binding
for
#py("$x")
does not satisfy
#py("job($x, llist(\"computer\", \"programmer\"))").
For example, in processing the query

#snippet(```python
and(supervisor($x, $y),
    not(job($x, llist("computer", "programmer"))))
```)

the first clause will generate frames with bindings for
#py("$x")
and
#py("$y").
The #py("not") clause will then filter these by
removing all frames in which the binding for
#py("$x")
satisfies the restriction that
#py("$x")
is a computer programmer.#footnote[There is a subtle difference between this
filter implementation of #py("not") and the usual
meaning of #py("not") in mathematical logic. See
section @sec:math-logic.]

The
#idx("javascriptpredicate (query language)", sub: "evaluation of")
#py("javascript_predicate") syntactic form
is implemented as a similar filter on frame streams. We use each frame in
the stream to instantiate any variables in the pattern, then apply the
Python
predicate. We remove from the input stream all frames for which the
predicate fails.

#idx("compound query", sub: "processing")

#subheading([Unification])

#idx("query interpreter", sub: "unification")
#idx("unification")

In order to handle rules in the query language, we must be able to
find the rules whose conclusions match a given query pattern. Rule
conclusions are like assertions except that they can contain
variables, so we will need a generalization of pattern
matching—called #emph[unification]—in which both the
"pattern" and the "datum" may contain variables.

A unifier takes two patterns, each containing constants and variables,
and determines whether it is possible to assign values to the
variables that will make the two patterns equal. If so, it returns a
frame containing these bindings. For example, unifying
#py("llist($x, \"a\", $y)")
and
#py("llist($y, $z, \"a\")")
will specify a frame in which
#py("$x"),
#py("$y"),
and
#py("$z")
must all be bound to
#py("\"a\"").
On the other hand, unifying
#py("llist($x, $y, \"a\")")
and
#py("llist($x, \"b\", $y)")
will fail, because there is no value for
#py("$y")
that can make the two patterns equal. (For the second elements of the
patterns to be equal,
#py("$y")
would have to be
#py("\"b\"")
however, for the third elements to be equal,
#py("$y")
would have to be
#py("\"a\"").)
The unifier used in the query system, like the pattern matcher, takes a
frame as input and performs unifications that are consistent with this frame.

The unification algorithm is the most technically difficult part of
the query system. With complex patterns, performing unification may
seem to require deduction.
To unify
#snippet(```python llist($x, $x) ```)
and
#snippet(```python llist(llist("a", $y, "c"), llist("a", "b", $z)) ```)
for example,
the algorithm must infer that
#py("$x")
should be
#py("llist(\"a\", \"b\", \"c\")"),
#py("$y")
should be
#py("\"b\""),
and
#py("$z")
should be
#py("\"c\"").
We may think of this process as solving a set of equations among the pattern
components. In general, these are simultaneous equations, which may require
substantial manipulation to solve.#footnote[In one-sided pattern matching,
all the equations that contain pattern variables are explicit and already
solved for the unknown (the pattern variable).] For example,
unifying
#py("llist($x, $x)")
and
#py("llist(llist(\"a\", $y, \"c\"), llist(\"a\", \"b\", $z))")
may be thought of as specifying the simultaneous equations

$ mat(delim: #none, mono("$x"), =, mono("llist(\"a\", $y, \"c\")"); mono("$x"), =, mono("llist(\"a\", \"b\", $z)")) $

These equations imply that

$ mat(delim: #none, mono("llist(\"a\", $y, \"c\")"), =, mono("llist(\"a\", \"b\", $z)")) $

which in turn implies that

$ mat(delim: #none, mono("\"a\""), =, mono("\"a\""), mono("$y"), =, mono("\"b\""), mono("\"c\""), =, mono("$z")) $

and hence that

$ mat(delim: #none, mono("$x"), =, mono("llist(\"a\", \"b\", \"c\")")) $

In a successful pattern match, all pattern variables become bound, and
the values to which they are bound contain only constants. This is
also true of all the examples of unification we have seen so far.
#idx("pattern matching", sub: "unification vs.")
#idx("unification", sub: "pattern matching vs.")
In general, however, a successful unification may not completely
determine the variable values; some variables may remain unbound and
others may be bound to values that contain variables.

Consider the unification of
#py("llist($x, \"a\")")
and
#py("llist(llist(\"b\", $y), $z)").
We can deduce that
#py("$x")
$=$
#py("llist(\"b\", $y)")
and
#py("\"a\"")
$=$
#py("$z"),
but we cannot further solve for
#py("$x")
or
#py("$y").
The unification doesn't fail, since it is certainly possible to make
the two patterns equal by assigning values to
#py("$x")
and
#py("$y").
Since this match in no way restricts the values
#py("$y")
can take on, no binding for
#py("$y")
is put into the result frame. The match does, however, restrict the value of
#py("$x").
Whatever value
#py("$y")
has,
#py("$x")
must be
#py("llist(\"b\", $y)").
A binding of
#py("$x")
to the pattern
#py("llist(\"b\", $y)")
is thus put into the frame. If a value for
#py("$y")
is later determined and added to the frame (by a pattern match or
unification that is required to be consistent with this frame), the
previously bound
#py("$x")
will refer to this value.#footnote[Another way to think of unification is
that it generates the most general pattern that is a specialization of the
two input patterns.
This means that the unification of #py("llist($x, \"a\")")
and
#py("llist(llist(\"b\", $y), $z)")
is
#py("llist(llist(\"b\", $y), \"a\")")
and
that
the unification of
#py("llist($x, \"a\", $y)")
and
#py("llist($y, $z, \"a\")"),
discussed above, is
#py("llist(\"a\", \"a\", \"a\")").
For our implementation, it is more convenient to think of the result
of unification as a frame rather than a pattern.]

#idx("query interpreter", sub: "unification")
#idx("unification")

#subheading([Applying rules])

#idx("rule (query language)", sub: "applying")

Unification is the key to the component of the query system that makes
inferences from rules. To see how this is accomplished, consider
processing a query that involves applying a rule, such as

#snippet(```python
lives_near($x, llist("Hacker", "Alyssa", "P"))
```)

To process this query, we first use the ordinary pattern-match
function
described above to see if there are any assertions in the data base that
match this pattern. (There will not be any in this case, since our data
base includes no direct assertions about who lives near whom.) The next
step is to attempt to unify the query pattern with the conclusion of each
rule. We find that the pattern unifies with the conclusion of the rule

#snippet(```python
rule(lives_near($person_1, $person_2),
     and(address($person_1, pair($town, $rest_1)),
         address($person_2, llist($town, $rest_2)),
         not(same($person_1, $person_2))))
```)

resulting in a frame specifying that
#py("$x") should be bound to (have the same value as) #py("$person_1") and that #py("$person_2") is bound to #py("llist(\"Hacker\", \"Alyssa\", \"P\")").
Now, relative to this frame, we evaluate the compound query given by the body
of the rule. Successful matches will extend this frame by providing a
binding for
#py("$person_1"),
and consequently a value for
#py("$x"),
which we can use to instantiate the original query pattern.

In general, the query evaluator uses the following method to apply a
rule when trying to establish a query pattern in a frame that
specifies bindings for some of the pattern variables:

- Unify the query with the conclusion of the rule to form, if successful, an extension of the original frame.
- Relative to the extended frame, evaluate the query formed by the body of the rule.

Notice how similar this is to the method for applying a
function
in the
#idx("query interpreter", sub: "Python interpreter vs.")
#py("evaluate")/#py("apply")
evaluator for
Python:

- Bind the function's parameters to its arguments to form a frame that extends the original function environment.
- Relative to the extended environment, evaluate the expression formed by the body of the function.

The similarity between the two evaluators should come as no surprise.
Just as
function
definitions are the means of abstraction in
Python,
rule definitions are the means of abstraction in the query language.
In each case, we unwind the abstraction by creating appropriate
bindings and evaluating the rule or
function
body relative to these.

#idx("rule (query language)", sub: "applying")

#subheading([Simple queries])

#idx("simple query", sub: "processing")

We saw earlier in this section how to evaluate simple queries in the
absence of rules. Now that we have seen how to apply rules, we can
describe how to evaluate simple queries by using both rules and
assertions.

Given the query pattern and a stream of frames, we produce, for each
frame in the input stream, two streams:

- a stream of extended frames obtained by matching the pattern against all assertions in the data base (using the pattern matcher), and
- a stream of extended frames obtained by applying all possible rules (using the unifier).#footnote[Since unification is a #idx("pattern matching", sub: "unification vs.") #idx("unification", sub: "pattern matching vs.") generalization of matching, we could simplify the system by using the unifier to produce both streams. Treating the easy case with the simple matcher, however, illustrates how matching (as opposed to full-blown unification) can be useful in its own right.]

Appending these two streams produces a stream that consists of all the
ways that the given pattern can be satisfied consistent with the
original frame. These streams (one for each frame in the input
stream) are now all combined to form one large stream, which therefore
consists of all the ways that any of the frames in the original input
stream can be extended to produce a match with the given pattern.

#idx("simple query", sub: "processing")

#subheading([The query evaluator and the driver loop])

Despite the complexity of the underlying matching operations, the
system is organized much like an
#idx("query interpreter", sub: "query evaluator")
#idx("query interpreter", sub: "Python interpreter vs.")
evaluator for any language. The
function
that coordinates the matching operations is called
#idx("evaluatequery")
#py("evaluate_query"),
and it plays a role analogous to that of the
#py("evaluate")
function
for
Python.
The function #py("evaluate_query")
takes as inputs a query and a stream of frames. Its output is a stream of
frames, corresponding to successful matches to the query pattern, that
extend some frame in the input stream, as indicated in
figure @fig:query-stream.
Like
#py("evaluate"),
#py("evaluate_query")
classifies the different types of expressions (queries) and dispatches to an
appropriate
function
for each. There is a
function
for each
syntactic
form
(#py("and"), #py("or"),
#py("not"), and
#py("javascript_predicate"))
and one for simple queries.

The
#idx("driver loop", sub: "in query interpreter")
#idx("query interpreter", sub: "driver loop")
driver loop, which is analogous to the
#py("driver_loop")
function
for the other evaluators in this chapter, reads queries
typed by the user.
For each query, it calls
#py("evaluate_query")
with the query and a stream that consists of a single empty frame. This
will produce the stream of all possible matches (all possible extensions to
the empty frame). For each frame in the resulting stream, it instantiates
the original query using the values of the variables found in the frame.
This stream of instantiated queries is then printed.#footnote[The reason we
use
#idx("stream(s)", sub: "used in query interpreter")
#idx("query interpreter", sub: "streams of frames")
streams (rather than lists) of frames is that the
recursive application of rules can generate infinite numbers of values that
satisfy a query. The delayed evaluation embodied in streams is crucial
here: The system will print responses one by one as they are generated,
regardless of whether there are a finite or infinite number of
responses.]

The driver also checks for the special command
#idx("assert (query interpreter)")
#idx("query interpreter", sub: "adding rule or assertion")
#py("assert"),
which signals that the input is not a query but rather an assertion or rule
to be added to the data base. For instance,
#idx("query interpreter", sub: "overview")
#snippet(```python
assert(job(llist("Bitdiddle", "Ben"), llist("computer", "wizard")))

assert(rule(wheel($person),
            and(supervisor($middle_manager, $person),
                supervisor($x, $middle_manager))))
```)
