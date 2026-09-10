// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Normal Order and Applicative Order], label-name: <sec:evaluation-order>)

#idx("normal-order evaluation", sub: "applicative order vs.")
#idx("applicative-order evaluation", sub: "normal order vs.")

In section @sec:elements-of-programming, where we began
our discussion of models of evaluation, we noted that
Python
is an #emph[applicative-order] language, namely, that all the arguments to
Python functions
are evaluated when the
function
is applied. In contrast, #emph[normal-order] languages delay evaluation of
function
arguments until the actual argument values are needed. Delaying evaluation of
function
arguments until the last possible moment (e.g., until they are required by a
primitive operation) is called
#idx("lazy evaluation")
#emph[lazy evaluation].#footnote[The difference between the
"lazy" terminology and the "normal-order"
terminology is somewhat fuzzy. Generally, "lazy" refers to the
mechanisms of particular evaluators, while "normal-order"
refers to the semantics of languages, independent of any particular
evaluation strategy. But this is not a hard-and-fast distinction, and the
two terminologies are often used interchangeably.] Consider the
function

#snippet(```python
def try_me(a, b):
    return 1 if a == 0 else b
```)

Evaluating
#py("try_me(0, head(None));") signals
an error in
Python.
With lazy evaluation, there would be no error. Evaluating the
statement
would return 1, because the argument
#py("head(None)")
would never be evaluated.

An example that exploits lazy evaluation is the
declaration
of a
function
#py("unless")

#snippet(```python
def unless(condition, usual_value, exceptional_value):
    return exceptional_value if condition else usual_value
```)

that can be used in
statements
such as

#snippet(```python
unless(is_none(xs), head(xs), print("error: xs should not be null"))
```)

This won't work in an applicative-order language because both the
usual value and the exceptional value will be evaluated before
#py("unless") is called (compare
exercise @ex:new-if). An advantage of lazy evaluation is
that some
functions,
such as #py("unless"), can do useful computation
even if evaluation of some of their arguments would produce errors or
would not terminate.

If the body of a
function
is entered before an argument has been evaluated we say that the
function
is
#idx("non-strict")
#emph[non-strict] in that argument. If the argument is evaluated before
the body of the
function
is entered we say that the
function
is
#idx("strict")
#emph[strict] in that
argument.#footnote[The "strict" versus "non-strict"
terminology means essentially the same as
"applicative-order" versus "normal-order," except
that it refers to individual
functions
and arguments rather than to the language as a whole. At a conference on
programming languages you might hear someone say, "The normal-order language #idx("Hassle") Hassle has certain strict primitives. Other functions take their arguments by lazy evaluation."]
In a purely applicative-order language, all
functions
are strict in each argument. In a purely normal-order language, all compound
functions
are non-strict in each argument, and primitive
functions
may be either strict or non-strict. There are also languages (see
exercise @ex:user-controlled-strictness) that give
programmers detailed control over the strictness of the
functions
they define.

A striking example of a
function
that can usefully be made non-strict is
#py("pair")
(or, in general, almost any constructor for data structures).
One can do useful computation, combining elements to form
data structures and operating on the resulting data structures,
even if the values of the elements are not known. It makes perfect
sense, for instance, to compute the length of a list without knowing
the values of the individual elements in the list. We will exploit
this idea in section @sec:lazy-cons to implement the
streams of chapter @chap:state as lists formed of non-strict
pairs.

#exercise(label-name: <ex:ordinary>, [
Suppose that (in ordinary applicative-order Python) we define
#py("unless") as shown above and then define
#py("factorial") in terms
of #py("unless") as

#snippet(```python
def factorial(n):
    return unless(n == 1, n * factorial(n - 1), 1)
```)

What happens if we attempt to evaluate
#py("factorial(5)")?
Will our
functions
work in a normal-order language?
])

#exercise(label-name: <ex:unless-syntactic-form>, [
Ben Bitdiddle and Alyssa P. Hacker
#idx("syntactic form", sub: "function vs.")

disagree over the importance of lazy
evaluation for implementing things such as
#py("unless"). Ben points out that it's possible
to implement #py("unless") in applicative order as a
syntactic
form. Alyssa counters that, if one did that,
#py("unless") would be merely syntax, not a
function
that could be used in conjunction with higher-order
functions.
Fill in the details on both sides of the argument.
Show how to implement #py("unless") as a derived component (like operator combination), by catching in #py("evaluate") applications whose function expression is the name #py("unless"). Give an example of a situation where it might be useful to have #py("unless") available as a function, rather than as a syntactic form.
])

#idx("normal-order evaluation", sub: "applicative order vs.")
#idx("applicative-order evaluation", sub: "normal order vs.")
