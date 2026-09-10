// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Mutable List Structure], label-name: <sec:mutable-list-structure>)

#idx("mutable data objects", sub: "list structure")
#idx("list structure", sub: "mutable")
#idx("mutable data objects", sub: "pairs")
#idx("pair(s)", sub: "mutable")

The basic operations on
pairs—#py("pair"),
#py("head"),
and
#py("tail")—can
be used to construct list structure and to select parts
from list structure, but they are incapable of modifying list
structure. The same is true of the list operations we have used so
far, such as #py("append") and
#py("list"), since these can be defined in terms of
#py("pair"),
#py("head"),
and
#py("tail").
To modify list structures we need new operations.

The primitive mutators for pairs are
#idx("sethead (primitive function)")

#py("set_head")
and
#idx("settail (primitive function)")

#py("set_tail").
The function #py("set_head")
takes two arguments, the first of which must be a pair. It modifies this
pair, replacing the
#py("head")
pointer by a pointer to the second argument of
#py("set_head").#footnote[The functions #py("set_head") and
#py("set_tail") return the value
#py("None").
#idx("sethead (primitive function)", sub: "value of")
#idx("settail (primitive function)", sub: "value of")
They should be used only for their effect.]

As an example, suppose that #py("x") is bound to
#py("llist(llist(\"a\", \"b\"), \"c\", \"d\")")
and #py("y") to
#py("llist(\"e\", \"f\")")
as illustrated in
figure @fig:two-lists.
Evaluating the expression
#py("set_head(x, y)")
modifies the pair to which #py("x") is bound,
replacing its
#py("head")
by the value of #py("y"). The result of the operation
is shown in
figure @fig:set-car.
The structure #py("x") has been modified and
is now equivalent to #py("llist(llist(\"e\", \"f\"), \"c\", \"d\")").
The pairs representing the list
#py("llist(\"a\", \"b\")"),
identified by the pointer that was replaced, are now detached from the
original structure.#footnote[We see from this that mutation operations on
lists can create "garbage" that is not part of any accessible
structure. We will see in section @sec:gc that
Python
memory-management systems include a
#idx("garbage collection", sub: "mutation and")
#emph[garbage collector], which identifies and recycles the memory
space used by unneeded pairs.]

#sicp-figure(image("/images/img_javascript/ch3-Z-G-13.svg", width: 70%), caption: [Lists #py("x"): #py("llist(llist(\"a\", \"b\"), \"c\", \"d\")") and #py("y"): #py("llist(\"e\", \"f\")").], label-name: <fig:two-lists>)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-14.svg", width: 70%), caption: [Effect of #py("set_head(x, y)") on the lists in figure @fig:two-lists.], label-name: <fig:set-car>)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-15.svg", width: 70%), caption: [Effect of #py("z = pair(y, tail(x))") on the lists in figure @fig:two-lists.], label-name: <fig:list-cons>)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-16.svg", width: 70%), caption: [Effect of #py("set_tail(x, y)") on the lists in figure @fig:two-lists.], label-name: <fig:set-cdr>)

Compare
figure @fig:set-car
with
figure @fig:list-cons,
which illustrates the result of executing

#snippet(```python
z = pair(y, tail(x))
```)

with #py("x") and #py("y")
bound to the original lists of
figure @fig:two-lists.
The
name
#py("z") is now bound to a
new pair created by the
#py("pair")
operation; the list to which #py("x") is bound is
unchanged.

The
#py("set_tail")
operation is similar to
#py("set_head").
The only difference is that the
#py("tail")
pointer of the pair, rather than the
#py("head")
pointer, is replaced. The effect of executing
#py("set_tail(x, y)")
on the lists of
figure @fig:two-lists
is shown in
figure @fig:set-cdr.
Here the
#py("tail")
pointer of
#py("x") has been replaced by the pointer to
#py("llist(\"e\", \"f\")").
Also, the list
#py("llist(\"c\", \"d\")"),
which used to be the
#py("tail")
of #py("x"), is now detached from the structure.

The function #py("pair")
builds new list structure by creating new pairs,
whereas #py("set_head")
and
#py("set_tail")
modify existing pairs.
Indeed, we could
#idx("pair (primitive function)", sub: "implemented with mutators")
implement
#py("pair")
in terms of the two mutators, together with a
function
#py("get_new_pair"),
which returns a new pair that is not part of any existing list structure.
We obtain the new pair, set its
#py("head")
and
#py("tail")
pointers to the designated objects, and return the new pair as the result of
the
#py("pair").#footnote[Section @sec:memory-as-vectors
will show how a memory-management system
can implement #py("get_new_pair").]

#idx("pair (primitive function)", sub: "implemented with mutators", decl: true)
#snippet(```python
def pair(x, y):
    fresh = get_new_pair()
    set_head(fresh, x)
    set_tail(fresh, y)
    return fresh
```)

#exercise(label-name: <ex:append>, [
The following
function
for appending lists was introduced in
section @sec:sequences:

#idx("append", decl: true)

#snippet(```python
def append(x, y):
    return (y
            if is_none(x)
            else pair(head(x), append(tail(x), y)))
```)

The function #py("append")
forms a new list by successively
adjoining the elements of #py("x") to the front of #py("y").
The
function
#idx("append", sub: "appendmutator vs.")
#py("append_mutator")
is similar to #py("append"), but it is a mutator
rather than a constructor. It appends the lists by splicing them together,
modifying the final pair of #py("x") so that its
#py("tail")
is now #py("y"). (It is an error to call
#py("append_mutator")
with an empty #py("x").)
#idx("appendmutator", decl: true)
#snippet(```python
def append_mutator(x, y):
    set_tail(last_pair(x), y)
    return x
```)

Here
#py("last_pair")
is a
function
that returns the last pair in its argument:

#idx("lastpair", decl: true)
#snippet(```python
def last_pair(x):
    return (x
            if is_none(tail(x))
            else last_pair(tail(x)))
```)

Consider the interaction

#snippet(```python
x = llist("a", "b")
```)

#snippet(```python
y = llist("c", "d")
```)

#snippet(```python
z = append(x, y)
```)

#snippet(```python
print(z)
```)

#output(```python
print(z)
```)

#snippet(```python
print(tail(x))
```)

#output(```python
print(tail(x))
```)

#snippet(```python
w = append_mutator(x, y)
```)

#snippet(```python
print(w)
```)

#output(```python
print(w)
```)

#snippet(```python
print(tail(x))
```)

#output(```python
print(tail(x))
```)

What are the missing #meta("response")s?
Draw box-and-pointer diagrams to explain your answer.
])

#exercise(label-name: <ex:make-cycle>, [
Consider the following
#idx("cycle in list")
#py("make_cycle")
function,
which uses the
#py("last_pair")
function
defined in exercise @ex:append:
#idx("makecycle", decl: true)
#snippet(```python
def make_cycle(x):
    set_tail(last_pair(x), x)
    return x
```)

Draw a box-and-pointer diagram that shows the structure
#py("z") created by

#snippet(```python
z = make_cycle(llist("a", "b", "c"))
```)

What happens if we try to compute
#py("last_pair(z)")?
])

#exercise(label-name: <ex:mystery>, [
The following
function
is quite useful, although obscure:
#idx("mystery", decl: true)
#snippet(```python
def mystery(x):
    def loop(x, y):
        if is_none(x):
            return y
        else:
            temp = tail(x)
            set_tail(x, y)
            return loop(temp, x)
    return loop(x, None)
```)

The function #py("loop")
uses the "temporary"
name
#py("temp")
to hold the old value of the
#py("tail")
of #py("x"), since the
#py("set_tail")
on the next line destroys the
#py("tail").
Explain what #py("mystery") does in general. Suppose
#py("v") is defined by

#snippet(```python
v = llist("a", "b", "c", "d")
```)

Draw the box-and-pointer diagram that represents the list to which
#py("v") is bound. Suppose that we now evaluate

#snippet(```python
w = mystery(v)
```)

Draw box-and-pointer diagrams that show the structures
#py("v") and #py("w") after
evaluating this
program.
What would be printed as the values of #py("v")
and #py("w")?
])

#idx("mutable data objects", sub: "list structure")
#idx("list structure", sub: "mutable")
#idx("mutable data objects", sub: "pairs")
#idx("pair(s)", sub: "mutable")

#subheading([Sharing and identity])

#idx("data", sub: "shared")
#idx("shared data")

We mentioned in section @sec:costs-of-assignment the
theoretical issues of
#idx("sameness and change", sub: "shared data and")
#idx("change and sameness", sub: "shared data and")
"sameness" and "change"
raised by the introduction of assignment. These issues arise in practice
when individual pairs are #emph[shared] among different data objects.
For example, consider the structure formed by

#snippet(```python
x = llist("a", "b")
z1 = pair(x, x)
```)

As shown in
figure @fig:identity1,
#py("z1") is a pair whose
#py("head")
and
#py("tail")
both point to the same pair #py("x"). This sharing
of #py("x") by the
#py("head")
and
#py("tail")
of #py("z1") is a consequence of the straightforward
way in which
#py("pair")
is implemented. In general, using
#py("pair")
to construct lists will result in an interlinked structure of pairs in
which many individual pairs are shared by many different structures.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-17.svg", width: 70%), caption: [The list #py("z1") formed by #py("pair(x, x)").], label-name: <fig:identity1>)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-18.svg", width: 70%), caption: [The list #py("z2") formed by #py("pair(llist(\"a\", \"b\"), llist(\"a\", \"b\"))").], label-name: <fig:identity2>)

In contrast to
figure @fig:identity1, figure @fig:identity2
shows
the structure created by

#snippet(```python
z2 = pair(llist("a", "b"), llist("a", "b"))
```)

In this structure, the pairs in the two
#py("llist(\"a\", \"b\")")
lists are distinct, although
they contain the same strings.#footnote[The two pairs are distinct because each call to #py("pair") returns a new pair. The strings are #idx("string(s)", sub: "uniqueness of") "the same" in the sense that they are primitive data (just like numbers) that are composed of the same characters in the same order. Since Python provides no way to mutate a string, any sharing that the designers of a Python interpreter might decide to implement for strings is undetectable. We consider primitive data such as numbers, booleans, and strings to be #emph[identical] if and only if they are #emph[indistinguishable].]<foot:symbol-sharing>

When thought of as a list, #py("z1") and
#py("z2") both represent "the same" list:

#snippet(```python
print(llist(llist("a", "b"), "a", "b"))
```)

In general, sharing is completely undetectable if we operate on lists using
only
#py("pair"),
#py("head"),
and
#py("tail").
However, if we allow mutators on list structure, sharing becomes
significant. As an example of the difference that sharing can make,
consider the following
function,
which modifies the
#py("head")
of the structure to which it is applied:

#snippet(```python
def set_to_wow(x):
    set_head(head(x), "wow")
    return x
```)

Even though #py("z1") and
#py("z2") are "the same" structure,
applying
#py("set_to_wow")
to them yields different results. With #py("z1"),
altering the
#py("head")
also changes the
#py("tail"),
because in #py("z1") the
#py("head")
and the
#py("tail")
are the same pair. With #py("z2"), the
#py("head")
and
#py("tail")
are distinct, so
#py("set_to_wow")
modifies only the
#py("head"):

#snippet(```python
print(z1)
```)

#output(```python
print(z1)
```)

#snippet(```python
print(set_to_wow(z1))
```)

#output(```python
print(set_to_wow(z1))
```)

#snippet(```python
print(z2)
```)

#output(```python
print(z2)
```)

#snippet(```python
print(set_to_wow(z2))
```)

#output(```python
print(set_to_wow(z2))
```)

One way to detect sharing in list structures is to use the
#idx("is", sub: "as equality of pointers", decl: true)
primitive predicate #py("is").
When applied to two nonprimitive values,
#py("x is y")
tests whether #py("x") and
#py("y") are the same object (that is, whether
#py("x") and #py("y")
are equal as pointers).

Thus, with #py("z1") and
#py("z2") as defined in
figure @fig:identity1 and @fig:identity2,
#py("head(z1) is tail(z1)")
is true and
#py("head(z2) is tail(z2)")
is false.

As will be seen in the following sections, we can exploit sharing to
greatly extend the repertoire of data structures that can be
represented by pairs. On the other hand, sharing can also be
#idx("mutable data objects", sub: "shared data")
dangerous, since modifications made to structures will also affect
other structures that happen to share the modified parts. The mutation
operations
#py("set_head")
and
#py("set_tail")
should be used with care; unless we have a good understanding of how our
data objects are shared, mutation can have unanticipated
results.#footnote[The subtleties of dealing with sharing of mutable data
objects reflect the underlying issues of "sameness" and
"change" that were raised in
section @sec:costs-of-assignment. We mentioned there
that admitting change to our language requires that a compound object must
have an "identity" that is something different from the pieces
from which it is composed. In
Python,
we consider this "identity" to be the quality that is tested by
#py("is"),
i.e., by equality of pointers. Since in most
Python
implementations a pointer is essentially a memory address, we are
"solving the problem" of defining the identity of objects by
stipulating that a data object "itself" is the information
stored in some particular set of memory locations in the computer. This
suffices for simple
Python
programs, but is hardly a general way to resolve the issue of
"sameness" in computational models.]

#exercise(label-name: <ex:3_15>, [
Draw box-and-pointer diagrams to explain the effect of
#py("set_to_wow")
on the structures #py("z1") and
#py("z2") above.
])

#exercise(label-name: <ex:count-pairs>, [
Ben Bitdiddle decides to write a
function
to count the number of pairs in any list structure.
"It's easy," he reasons. "The number of pairs in any structure is the number in the #py("head") plus the number in the #py("tail") plus one more to count the current pair." So Ben writes the following
function
#idx("countpairs", decl: true)
#snippet(```python
def count_pairs(x):
    return (0
            if not is_pair(x)
            else count_pairs(head(x)) +
                 count_pairs(tail(x)) +
                 1)
```)

Show that this
function
is not correct. In particular, draw box-and-pointer diagrams representing
list structures made up of exactly three pairs for which Ben's
function
would return 3; return 4; return 7; never return at all.
])

#exercise(label-name: <ex:count-pairs2>, [
Devise a correct version of the
#py("count_pairs")
function
of exercise @ex:count-pairs that returns the number of
distinct pairs in any structure. (Hint: Traverse the structure, maintaining
an auxiliary data structure that is used to keep track of which pairs have
already been counted.)
])

#exercise(label-name: <ex:find-cycle>, [
Write a
function
that examines a list and
#idx("cycle in list", sub: "detecting")
determines whether it contains a cycle, that is,
whether a program that tried to find the end of the list by taking
successive
#py("tail")s
would go into an infinite loop. Exercise @ex:make-cycle
constructed such lists.
])

#exercise(label-name: <ex:3_19>, [
Redo exercise @ex:find-cycle using an algorithm that
takes only a constant amount of space. (This requires a very clever idea.)
])

#idx("data", sub: "shared")
#idx("shared data")

#subheading([Mutation is just assignment])

#idx("mutable data objects", sub: "functional representation of")
#idx("mutable data objects", sub: "implemented with assignment")
#idx("pair(s)", sub: "functional representation of")
#idx("functional representation of data", sub: "mutable data")

When we introduced compound data, we observed in
section @sec:data- that pairs can be represented purely
in terms of
functions:
#idx("pair (primitive function)", sub: "functional implementation of", decl: true)#idx("head (primitive function)", sub: "functional implementation of", decl: true)#idx("tail (primitive function)", sub: "functional implementation of", decl: true)
#snippet(```python
def pair(x, y):
    def dispatch(m):
        return (x
                if m == "head"
                else y
                if m == "tail"
                else error("undefined operation -- pair", m))
    return dispatch

def head(z):
    return z("head")

def tail(z):
    return z("tail")
```)

The same observation is true for mutable data. We can implement
mutable data objects as
functions
using assignment and local state. For instance, we can extend the above
pair implementation to handle
#py("set_head")
and
#py("set_tail")
in a manner analogous to the way we implemented bank accounts using
#py("make_account")
in section @sec:local-state-variables:

#idx("pair (primitive function)", sub: "functional implementation of", decl: true)#idx("head (primitive function)", sub: "functional implementation of", decl: true)#idx("tail (primitive function)", sub: "functional implementation of", decl: true)#idx("sethead (primitive function)", sub: "functional implementation of", decl: true)#idx("settail (primitive function)", sub: "functional implementation of", decl: true)
#snippet(```python
def pair(x, y):
    def set_x(v):
        nonlocal x
        x = v
    def set_y(v):
        nonlocal y
        y = v
    return lambda m: (x
                      if m == "head"
                      else y
                      if m == "tail"
                      else set_x
                      if m == "set_head"
                      else set_y
                      if m == "set_tail"
                      else error("undefined operation -- pair", m))

def head(z):
    return z("head")

def tail(z):
    return z("tail")

def set_head(z, new_value):
    z("set_head")(new_value)
    return z

def set_tail(z, new_value):
    z("set_tail")(new_value)
    return z
```)

Assignment is all that is needed, theoretically, to account for the
behavior of mutable data. As soon as we admit
assignment
to our language, we raise all the issues, not only of assignment, but of
mutable data in general.#footnote[On the other hand, from the viewpoint of
implementation, assignment requires us to modify the environment, which is
itself a mutable data structure. Thus, assignment and mutation are
equipotent: Each can be implemented in terms of the other.]

#exercise(label-name: <ex:cons-with-assignment>, [
Draw environment diagrams to illustrate the evaluation of the sequence
of
statements

#snippet(```python
x = pair(1, 2)
z = pair(x, x)
set_head(tail(z), 17)
```)

#snippet(```python
print(head(x))
```)

#output(```python
print(head(x))
```)

using the
functional
implementation of pairs given above. (Compare
exercise @ex:two-accounts.)
])

#idx("mutable data objects", sub: "functional representation of")
#idx("mutable data objects", sub: "implemented with assignment")
#idx("pair(s)", sub: "functional representation of")
#idx("functional representation of data", sub: "mutable data")
#idx("mutable data objects")
