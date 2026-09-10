// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Streams as Lazy Lists], label-name: <sec:lazy-cons>)

#idx("stream(s)", sub: "implemented as lazy lists")
#idx("lazy list")
#idx("list(s)", sub: "lazy")
#idx("lazy pair")
#idx("pair(s)", sub: "lazy")

In section @sec:delayed-lists, we showed how to
implement streams as delayed lists.
#idx("delayed expression", sub: "lazy evaluation and")
We used a #idx("lambda expression", sub: "lazy evaluation and") lambda expression
to construct a
#idx("promise to evaluate", sub: "lazy evaluation and")
"promise" to compute the
tail
of a stream, without actually fulfilling that promise until later.

We were forced to create streams as a new kind of data object similar
but not identical to lists, and this required us to reimplement many
ordinary list operations (#py("map"),
#py("append"), and so on) for use with streams.

With lazy evaluation, streams and lists can be identical, so there is
no need for
separate list and stream operations. All we need to do is to arrange matters
so that
#py("pair")
is non-strict. One way to accomplish this is to extend the lazy evaluator
to allow for non-strict primitives, and to implement
#py("pair")
as one of these. An easier way is to recall
(section @sec:data-) that there is no fundamental need
to implement
#py("pair")
as a primitive at all. Instead, we can represent
#idx("pair(s)", sub: "functional representation of")
pairs as
functions:#footnote[This
is the
functional
representation described in exercise @ex:lambda-cons.
Essentially any
functional
representation (e.g., a message-passing implementation) would do as well.
Notice that we can install these definitions in the lazy evaluator simply by
typing them at the driver loop. If we had originally included
#py("pair"),
#py("head"),
and
#py("tail")
as primitives in the global environment, they will be redefined. (Also see
exercises @ex:lazy-list-input
and @ex:lazy-list-printing.)]

#idx("pair (primitive function)", sub: "functional implementation of", decl: true)#idx("head (primitive function)", sub: "functional implementation of", decl: true)#idx("tail (primitive function)", sub: "functional implementation of", decl: true)
#snippet(```python
def pair(x, y):
    return lambda m: (m(x, y))
def head(z):
    return z(lambda p, q: (p))
def tail(z):
    return z(lambda p, q: (q))
```)

In terms of these basic operations, the standard definitions of the list
operations will work with infinite lists (streams) as well as finite ones,
and the stream operations can be implemented as list operations. Here are
some examples:

#idx("listref", decl: true)#idx("map", decl: true)#idx("scalelist", decl: true)#idx("addlists", decl: true)#idx("ones (infinite stream)", sub: "lazy-list version", decl: true)#idx("integers (infinite stream)", sub: "lazy-list version", decl: true)
#snippet(```python
def llist_ref(items, n):
    return head(items) if n == 0 else llist_ref(tail(items), n - 1)
def map(fun, items):
    return None if is_none(items) else pair(fun(head(items)), map(fun, tail(items)))
def scale_list(items, factor):
    return map(lambda x: (x * factor), items)
def add_lists(list1, list2):
    return list2 if is_none(list1) else list1 if is_none(list2) else pair(head(list1) + head(list2), add_lists(tail(list1), tail(list2)))
ones = pair(1, ones)
integers = pair(1, add_lists(ones, integers))
```)

#prompt(```python
L-evaluate input:
```)

#snippet(```python
llist_ref(integers, 17)
```)

#output(```python
llist_ref(integers, 17)
```)

Note that these lazy lists are even lazier than the streams of
chapter @chap:state: The
head
of the list, as well as the
tail,
is delayed.#footnote[This permits us to create delayed versions of more
general kinds of
list structures, not just sequences.
#idx("Hughes, R. J. M.")
Hughes 1990
discusses some
applications of
#idx("lazy tree")#idx("tree", sub: "lazy")
"lazy trees."]
In fact, even accessing the
#py("head")
or
#py("tail")
of a lazy pair need not force the value of a list element. The value will be
forced only when it is really needed—e.g., for use as the argument
of a primitive, or to be printed as an answer.

Lazy pairs also help with the problem that arose with streams in
section @sec:streams-and-delayed-evaluation, where we
found that formulating stream models of systems with loops may require us to
sprinkle our programs with
additional lambda expressions for #idx("delayed evaluation", sub: "explicit vs. automatic") #idx("delayed expression", sub: "explicit vs. automatic") delays, beyond the ones required to construct a stream pair.
With lazy evaluation, all arguments to
functions
are delayed uniformly. For instance, we can implement
functions
to integrate lists and solve differential equations as we originally
intended in section @sec:streams-and-delayed-evaluation:

#idx("integral", sub: "lazy-list version", decl: true)#idx("solve differential equation", sub: "lazy-list version", decl: true)
#snippet(```python
def integral(integrand, initial_value, dt):
    int = pair(initial_value, add_lists(scale_list(integrand, dt), int))
    return int
def solve(f, y0, dt):
    y = integral(dy, y0, dt)
    dy = map(f, y)
    return y
```)

#prompt(```python
L-evaluate input:
```)

#snippet(```python
llist_ref(solve(lambda x: (x), 1, 0.001), 1000)
```)

#output(```python
llist_ref(solve(lambda x: (x), 1, 0.001), 1000)
```)

#exercise(label-name: <ex:lazier>, [
Give some examples that illustrate the difference between the streams
of chapter @chap:state and the "lazier" lazy lists described in
this section. How can you take advantage of this extra laziness?
])

#exercise(label-name: <ex:lazy-list-input>, [
Ben Bitdiddle tests the lazy list implementation given above by
evaluating the expression

#snippet(```python
head(llist("a", "b", "c"))
```)

To his surprise, this produces an error. After some thought, he realizes
that the "lists" obtained
from the primitive #py("list") function
are different from the lists manipulated by the new definitions of
#py("pair"),
#py("head"),
and
#py("tail").
Modify
the evaluator such that applications of the primitive #py("list") function
typed at the driver loop will produce true lazy lists.
])

#exercise(label-name: <ex:lazy-list-printing>, [
Modify the driver loop for the evaluator so that lazy pairs and lists will
print in some reasonable way. (What are you going to do about infinite
lists?) You may also need to modify the representation of lazy pairs so
that the evaluator can identify them in order to print them.
])

#idx("delayed evaluation", sub: "in lazy evaluator")
#idx("stream(s)", sub: "implemented as lazy lists")
#idx("lazy list")
#idx("list(s)", sub: "lazy")
#idx("lazy pair")
#idx("pair(s)", sub: "lazy")
