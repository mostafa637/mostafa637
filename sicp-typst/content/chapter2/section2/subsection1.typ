// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Representing Sequences], label-name: <sec:sequences>)

One of the useful structures we can build with pairs is a
#idx("sequence(s)")
#idx("sequence(s)", sub: "represented by pairs")
#idx("pair(s)", sub: "used to represent sequence")
#emph[sequence]—an ordered collection of data objects. There
are, of course, many ways to represent sequences in terms of pairs. One
particularly straightforward representation is illustrated in
figure @fig:sequence-of-pairs,
where the sequence 1, 2, 3, 4 is represented as a chain of pairs. The
#py("head")
of each pair is the
corresponding item in the chain, and the
#py("tail")
of the pair is the next pair in the chain. The
#py("tail")
of the final pair signals the end of the
sequence,
represented in box-and-pointer
diagrams as a diagonal line
#idx("box-and-pointer notation", sub: "end-of-linked-list marker")
and in programs as
#idx("keywords", sub: "None")
#idx("None (keyword)", sub: "as end-of-linked-list marker")
#idx("end-of-linked-list marker")
Python's primitive value #py("None").
The entire sequence is constructed by nested
#py("pair")
operations:

#snippet(```python
pair(1,
     pair(2,
          pair(3,
               pair(4, None))))
```)

#sicp-figure(image("/images/img_javascript/ch2-Z-G-13.svg", width: 70%), caption: [The sequence 1, 2, 3, 4 represented as a chain of pairs.], label-name: <fig:sequence-of-pairs>)

Such a sequence of pairs, formed by nested
#py("pair") applications,
is called a
#idx("linked list") #emph[linked list],
and
our Python environment
provides a primitive called
#idx("llist (primitive function)")

#py("llist")
to help in constructing
linked
lists.#footnote[In this book, we use
#emph[linked list]
to mean a chain of
pairs terminated by the end-of-linked-list marker.
In contrast, the term
#idx("linked-list structure", sub: "linked list vs.")
#idx("linked list(s)", sub: "linked-list structure vs.")
#emph[linked-list structure]
refers to any data structure made out of pairs,
not just to
linked lists.]
The above sequence could be produced by
#idx("llist (primitive function)")

#py("llist(1, 2, 3, 4)").
In general,

#syntax("
llist(", meta("a"), $""_(1)$, ", ", meta("a"), $""_(2)$, ", ", $dots.h$, ", ", meta("a"), $""_(n)$, ")
      ")

is equivalent to

#syntax("
pair(", meta("a"), $""_(1)$, ", pair(", meta("a"), $""_(2)$, ", pair(", $dots.h$, ", pair(", meta("a"), $""_(n)$, ", None)", $dots.h$, ")))
      ")

Our interpreter prints pairs using a textual representation of
box-and-pointer diagrams that we call #emph[box notation].
#idx("linked list", sub: "printed representation of")
#idx("[ , ] (box notation for pairs)", sort: "0a21")
#idx("box notation for pairs")
#idx("pair(s)", sub: "box notation for")
#idx("notation in this book", sub: "box notation for data")
The result of #py("pair(1, 2)")
is printed as #py("[1, 2]"), and
the data object in figure @fig:sequence-of-pairs
is printed as
#py("[1, [2, [3, [4, None]]]]"):

#snippet(```python
one_through_four = llist(1, 2, 3, 4)
```)

#snippet(```python
print(one_through_four)
```)

#output(```python
print(one_through_four)
```)

We can think of
#idx("linked list", sub: "manipulation with head, tail, and pair")
#idx("head (primitive function)", sub: "as linked-list operation")
#py("head")
as selecting the first item in the
linked list,
and of
#idx("tail (primitive function)", sub: "as linked-list operation")
#py("tail")
as selecting the
linked-list component
consisting of all but the first item. Nested
applications of
#py("head")
and
#py("tail")
can be used to extract the second, third, and subsequent items in the
linked list.
The constructor
#idx("pair (primitive function)", sub: "as linked-list operation")
#py("pair")
makes a
linked list
like the original one, but with an additional item at the
beginning.

#snippet(```python
print(head(one_through_four))
```)

#output(```python
print(head(one_through_four))
```)

#snippet(```python
print(tail(one_through_four))
```)

#output(```python
print(tail(one_through_four))
```)

#snippet(```python
print(head(tail(one_through_four)))
```)

#output(```python
print(head(tail(one_through_four)))
```)

#snippet(```python
print(pair(10, one_through_four))
```)

#output(```python
print(pair(10, one_through_four))
```)

#snippet(```python
print(pair(5, one_through_four))
```)

#output(```python
print(pair(5, one_through_four))
```)

The value #py("None"), used to terminate
the chain of pairs, can be thought of as a sequence of no elements, the
#idx("empty linked list")
#idx("None (keyword)", sub: "as empty linked list")
#emph[empty linked list].#footnote[The value
#py("None") is used in Python for
various purposes, as we shall see in chapter @chap:state.]

Box notation is sometimes difficult to read. In this book, when we want to
indicate the
linked-list nature of a data structure, we will employ the
alternative
#idx("notation in this book", sub: "linked-list notation for data")
#idx("linked-list notation for data")
#emph[linked-list notation]: Whenever possible, linked-list notation uses
applications
of #py("llist") whose evaluation would result in the
desired structure. For example, instead of the box notation

#output(```python
print(pair(5, one_through_four))
```)

we write

#output(```python
print(pair(5, one_through_four))
```)

in linked-list notation.#footnote[Our Python environment provides
a primitive function
#py("print_llist")
that works like the primitive function
#py("print"), except that
it uses linked-list notation instead of box notation.]

#subheading([Linked-list operations])

#idx("linked list", sub: "operations on")
#idx("linked list", sub: "techniques for manipulating")

The use of pairs to represent sequences of elements as
linked lists
is accompanied
by conventional programming techniques for manipulating
linked lists
by
successively
#idx("walking down a linked list with tail") #idx("linked list", sub: "walking down with tail") using #py("tail") to walk down the linked lists.
For example, the
function
#idx("linked list", sub: "nth element of")
#py("llist_ref")
takes as arguments a
linked list
and a number $n$ and
returns the $n$th item of the
linked list.
It is
customary to number the elements of the
linked list
beginning with 0. The method
for computing
#py("llist_ref")
is the following:

- For $n=0$, #py("llist_ref") should return the #py("head") of the linked list.
- Otherwise, #py("llist_ref") should return the $(n-1)$st item of the #py("tail") of the linked list.

#idx("llistref", decl: true)
#snippet(```python
def llist_ref(items, n):
    return (head(items) if n == 0
            else llist_ref(tail(items), n - 1))
```)

#snippet(```python
squares = llist(1, 4, 9, 16, 25)

print(llist_ref(squares, 3))
```)

#output(```python
squares = llist(1, 4, 9, 16, 25)

print(llist_ref(squares, 3))
```)

Often we
walk down the whole linked list.
To aid in this,
our Python environment
includes a primitive
predicate
#idx("isnone (primitive function)")

#idx("empty linked list", sub: "recognizing with isnone")
#idx("None (keyword)", sub: "recognizing with isnone")
#py("is_none"),
which tests whether its argument is the empty
linked list.
The
function
#idx("length")
#idx("linked list", sub: "length of")
#py("length"), which returns the number of items in a
linked list,
illustrates this typical pattern of use:
#idx("length", sub: "recursive version", decl: true)
#snippet(```python
def length(items):
    return (0 if is_none(items)
            else 1 + length(tail(items)))
```)

#snippet(```python
odds = llist(1, 3, 5, 7)

print(length(odds))
```)

#output(```python
odds = llist(1, 3, 5, 7)

print(length(odds))
```)

The #py("length")
function
implements a simple recursive plan. The reduction step is:

- The #py("length") of any linked list is 1 plus the #py("length") of the #py("tail") of the linked list.

This is applied successively until we reach the base case:

- The #py("length") of the empty linked list is 0.

We could also compute #py("length") in an iterative
style:
#idx("length", sub: "iterative version", decl: true)
#snippet(```python
def length(items):
    def length_iter(a, count):
        return (count if is_none(a)
                else length_iter(tail(a), count + 1))
    return length_iter(items, 0)
```)

Another conventional programming technique is to
#idx("constructing a linked list with pair") #idx("linked list", sub: "constructing with pair") #idx("adjoining to a linked list with pair") #idx("linked list", sub: "adjoining to with pair") construct an answer linked list by adjoining elements to the front of the linked list with #py("pair") while walking down a linked list using #py("tail"),
as in the
function
#idx("linked list", sub: "combining with append")
#py("append"), which takes two linked lists as arguments and combines their elements to make a new linked list:

#snippet(```python
print_llist(append(squares, odds))
```)

#output(```python
print_llist(append(squares, odds))
```)

#snippet(```python
print_llist(append(odds, squares))
```)

#output(```python
print_llist(append(odds, squares))
```)

The function #py("append")
is also implemented using a recursive plan.
To append linked lists #py("list1") and #py("list2"), do the following:

- If #py("list1") is the empty linked list, then the result is just #py("list2").
- Otherwise, append the #py("tail") of #py("list1") and #py("list2"), and adjoin the #py("head") of #py("list1") to the result:

#idx("append", decl: true)
#snippet(```python
def append(list1, list2):
    return (list2 if is_none(list1)
            else pair(head(list1), append(tail(list1), list2)))
```)

#exercise(label-name: <ex:last>, [
Define a
function
#idx("lastpair")
#idx("linked list", sub: "last pair of")
#py("last_pair")
that returns the
linked list
that contains only the last element of a given
(nonempty)
linked list:

#snippet(```python
print_llist(last_pair(llist(23, 72, 149, 34)))
```)

#output(```python
print_llist(last_pair(llist(23, 72, 149, 34)))
```)
])

#exercise(label-name: <ex:reverse>, [
Define a
function
#idx("reverse")
#idx("linked list", sub: "reversing")
#py("reverse")
that takes a
linked list
as argument and
returns a
linked list
of the same elements in reverse order:

#snippet(```python
print_llist(reverse(llist(1, 4, 9, 16, 25)))
```)

#output(```python
print_llist(reverse(llist(1, 4, 9, 16, 25)))
```)
])

#exercise(label-name: <ex:2_19>, [
Consider the
#idx("counting change")
change-counting program of
section @sec:tree-recursion. It would be nice to be
able to easily change the currency used by the program, so that we could
compute the number of ways to change a British pound, for example. As
the program is written, the knowledge of the currency is distributed
partly into the
function
#py("first_denomination")
and partly into the
function
#py("count_change")
(which knows
that there are five kinds of U.S. coins).
It would be nicer
to be able to supply a
linked list
of coins to be used for making change.

We want to rewrite the
function
#py("cc") so that its second argument
is a
linked list
of
the values of the coins to use rather than an integer specifying which
coins to use. We could then have
linked lists
that defined each kind of
currency:

#snippet(```python
us_coins = llist(50, 25, 10, 5, 1)
uk_coins = llist(100, 50, 20, 10, 5, 2, 1)
```)

We could then call #py("cc") as follows:

#snippet(```python
print(cc(100, us_coins))
```)

#output(```python
print(cc(100, us_coins))
```)

To do this will require changing the program
#py("cc") somewhat. It will still have the same
form, but it will access its second argument differently, as follows:

#snippet(```python
def cc(amount, coin_values):
    return (1 if amount == 0
            else 0 if amount < 0 or no_more(coin_values)
            else cc(amount, except_first_denomination(coin_values)) +
                 cc(amount - first_denomination(coin_values), coin_values))
```)

Define the
functions
#py("first_denomination"),
#py("except_first_denomination"),
and
#py("no_more")
in terms of primitive operations on
linked list
structures. Does the order of
the
linked list
#py("coin_values")
affect the answer produced by #py("cc")?
Why or why not?
])

#exercise(label-name: <ex:2_20>, [
In the presence of higher-order functions, it is not strictly necessary
for functions to have multiple parameters; one would
suffice. If we have a function such as
#py("plus") that naturally requires two
arguments, we could write a variant of the function to which we pass
the arguments one at a time. An application of the variant to the
first argument could return a function that we can then apply to the
second argument, and so on. This practice—called
#idx("currying")
#emph[currying] and named after the American mathematician and
logician
#idx("Curry, Haskell Brooks")
Haskell Brooks Curry—is quite common in programming
languages such as
#idx("Haskell")
Haskell and
#idx("Ocaml")
OCaml. In Python, a curried
version of #py("plus") looks as follows.

#snippet(```python
def plus_curried(x):
    return lambda y: x + y
```)

Write a function #py("brooks") that
takes a curried function as first argument and as second argument a linked list
of arguments to which the curried function is then applied, one by one,
in the given order. For example, the following application of
#py("brooks") should have the
same effect as
#py("plus_curried(3)(4)"):

#snippet(```python
print(brooks(plus_curried, llist(3, 4)))
```)

#output(```python
print(brooks(plus_curried, llist(3, 4)))
```)

While we are at it, we might as well curry the function
#py("brooks")! Write a function
#py("brooks_curried") that can be applied
as follows:

#snippet(```python
print(brooks_curried(llist(plus_curried, 3, 4)))
```)

#output(```python
print(brooks_curried(llist(plus_curried, 3, 4)))
```)

With this function #py("brooks_curried"),
what are the results of evaluating the following two statements?

#snippet(```python
brooks_curried(llist(brooks_curried,
                     llist(plus_curried, 3, 4)))
```)

#snippet(```python
brooks_curried(llist(brooks_curried,
                     llist(brooks_curried,
                           llist(plus_curried, 3, 4))))
```)
])

#idx("linked list", sub: "operations on")
#idx("linked list", sub: "techniques for manipulating")

#subheading([Mapping over linked lists])

#idx("linked list", sub: "mapping over")
#idx("mapping", sub: "over linked lists")

One extremely useful operation is to apply some transformation to each
element in a
linked list
and generate the
linked list
of results. For instance, the
following
function
scales each number in a
linked list
by a given factor:
#idx("scalelinkedlist", decl: true)
#snippet(```python
def scale_linked_list(items, factor):
    return (None if is_none(items)
            else pair(head(items) * factor,
                      scale_linked_list(tail(items), factor)))
```)

#snippet(```python
print(scale_linked_list(llist(1, 2, 3, 4, 5), 10))
```)

#output(```python
print(scale_linked_list(llist(1, 2, 3, 4, 5), 10))
```)

We can abstract this general idea and capture it as a common pattern
expressed as a higher-order
function,
just as in section @sec:higher-order-procedures. The
higher-order
function
here is called #py("map").
The function #py("map")
takes as arguments a
function
of one argument and a
linked list,
and returns a
linked list
of the results produced by
applying the
function
to each element in the
linked list:
#idx("map", decl: true)
#snippet(```python
def map(fun, items):
    return (None if is_none(items)
            else pair(fun(head(items)),
                      map(fun, tail(items))))
```)

#snippet(```python
print(map(abs, llist(-10, 2.5, -11.6, 17)))
```)

#output(```python
print(map(abs, llist(-10, 2.5, -11.6, 17)))
```)

#snippet(```python
print(map(lambda x: x * x, llist(1, 2, 3, 4)))
```)

#output(```python
print(map(lambda x: x * x, llist(1, 2, 3, 4)))
```)

Now we can give a new definition of
#py("scale_linked_list")
in terms of
#py("map"):
#idx("scalelinkedlist", decl: true)
#snippet(```python
def scale_linked_list(items, factor):
    return map(lambda x: x * factor, items)
```)

The function #py("map")
is an important construct, not only because it captures a common pattern,
but because it establishes a higher level of abstraction in dealing with
linked lists.
In the original definition of
#py("scale_linked_list"),
the recursive structure of the program draws attention to the
element-by-element processing of the
linked list.
Defining
#py("scale_linked_list")
in terms of
#py("map")
suppresses that level of
detail and emphasizes that scaling transforms a
linked list
of elements to a
linked list
of results. The difference between the two definitions is not that the
computer is performing a different process (it isn't) but that we
think about the process differently. In effect,
#py("map")
helps establish an abstraction barrier
that isolates the implementation of
functions
that transform linked lists from the details of how the elements of the linked list are
extracted and combined. Like the barriers shown in
figure @fig:abstraction-barriers,
this abstraction gives us the flexibility to change the low-level details
of how sequences are implemented, while preserving the conceptual framework
of operations that transform sequences to sequences.
Section @sec:sequences-conventional-interfaces expands
on this use of sequences as a framework for organizing programs.

#exercise(label-name: <ex:square-list>, [
The
function
#py("square_linked_list")
takes a
linked list
of numbers as argument and returns a
linked list
of the squares of
those numbers.

#snippet(```python
print(square_linked_list(llist(1, 2, 3, 4)))
```)

#output(```python
print(square_linked_list(llist(1, 2, 3, 4)))
```)

Here are two different definitions of
#py("square_linked_list").
Complete both of them by filling in the missing expressions:

#syntax("
def square_linked_list(items):
    return (None if is_none(items)
            else pair(", metaphrase[??], ", ", metaphrase[??], "))
      ")

#syntax("
def square_linked_list(items):
    return map(", metaphrase[??], ", ", metaphrase[??], ")
      ")
])

#exercise(label-name: <ex:iter-square-list>, [
Louis Reasoner tries to rewrite the first
#py("square_linked_list")
function
of exercise @ex:square-list so that it evolves an
iterative process:

#snippet(```python
def square_linked_list(items):
    def iter(things, answer):
        return (answer if is_none(things)
                else iter(tail(things),
                          pair(square(head(things)),
                               answer)))
    return iter(items, None)
```)

Unfortunately, defining
#py("square_linked_list")
this way produces the answer
linked list
in the reverse order of the one desired.
Why?

Louis then tries to fix his bug by interchanging the arguments to
#py("pair"):

#snippet(```python
def square_linked_list(items):
    def iter(things, answer):
        return (answer if is_none(things)
                else iter(tail(things),
                          pair(answer,
                               square(head(things)))))
    return iter(items, None)
```)

This doesn't work either. Explain.
])

#exercise(label-name: <ex:for-each>, [
The
function
#idx("foreach")
#py("for_each")
is similar to
#py("map")
It takes as arguments a
function
and a
linked list
of elements. However, rather than forming a
linked list
of the
results,
#py("for_each")
just applies the
function
to each of the elements in turn, from left to right. The values returned by
applying the
function
to the elements are not used
at all—#py("for_each")
is used with
functions
that perform an action, such as printing. For example,

#snippet(```python
print(for_each(lambda x: print(x), llist(57, 321, 88)))
```)

#output(```python
print(for_each(lambda x: print(x), llist(57, 321, 88)))
```)

The value returned by the call to
#py("for_each")
(not illustrated above) can be something arbitrary, such as true. Give an
implementation of
#py("for_each").
])

#idx("linked list", sub: "mapping over")
#idx("mapping", sub: "over linked lists")
