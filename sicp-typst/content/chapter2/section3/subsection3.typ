// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Example: Representing Sets], label-name: <sec:representing-sets>)

#idx("set")

In the previous examples we built representations for two kinds of
compound data objects: rational numbers and algebraic expressions. In
one of these examples we had the choice of simplifying (reducing) the
expressions at either construction time or selection time, but other
than that the choice of a representation for these structures in terms
of
linked lists
was straightforward. When we turn to the representation of
sets, the choice of a representation is not so obvious. Indeed, there
are a number of possible representations, and they differ
significantly from one another in several ways.

Informally, a set is simply a collection of distinct objects. To give
a more precise definition we can employ the method of data
abstraction. That is, we define "set" by specifying the
#idx("set", sub: "operations on")
operations that are to be used on sets. These are
#py("union_set"),
#py("intersection_set"),
#py("is_element_of_set"),
and
#py("adjoin_set").
#idx("iselementofset")
The function #py("is_element_of_set")
is a predicate that determines whether a given element is a member of a set.
#idx("adjoinset")
The function #py("adjoin_set")
takes an object and a set as arguments and returns a set that contains the
elements of the original set and also the adjoined element.
#idx("unionset")
The function #py("union_set")
computes the union of two sets, which is the set containing each element
that appears in either argument.
#idx("intersectionset")
The function #py("intersection_set")
computes the intersection of two sets, which is the set containing only
elements that appear in both arguments. From the viewpoint of data
abstraction, we are free to design any representation that implements these
operations in a way consistent with the interpretations given
above.#footnote[If we want to be more formal, we can specify
"consistent with the interpretations given above" to mean
that the operations satisfy a collection of rules such as these:

- For any set #py("S") and any object #py("x"), #py("is_element_of_set(x, adjoin_set(x, S))") is true (informally: "Adjoining an object to a set produces a set that contains the object").
- For any sets #py("S") and #py("T") and any object #py("x"), #py("is_element_of_set(x, union_set(S, T))") is equal to #py("is_element_of_set(x, S) or is_element_of_set(x, T)") (informally: "The elements of #py("union_set(S, T)") are the elements that are in #py("S") or in #py("T")").
- For any object #py("x"), #py("is_element_of_set(x, None)") is false (informally: "No object is an element of the empty set").]
#idx("set", sub: "operations on")

#subheading([Sets as unordered linked lists])

#idx("set", sub: "represented as unordered linked list")
#idx("unordered-linked-list representation of sets")

One way to represent a set is as a
linked list
of its elements in which no
element appears more than once. The empty set is represented by the
empty
linked list.
In this representation,
#py("is_element_of_set")
is similar to the
function
#py("member") of section @sec:strings.
It uses
#py("equal")
instead of
#py("==")
so that the set elements need not be
just numbers or strings:
#idx("iselementofset", sub: "unordered-linked-list representation", decl: true)
#snippet(```python
def is_element_of_set(x, set):
    return (False
            if is_none(set)
            else True
            if x == head(set)
            else is_element_of_set(x, tail(set)))
```)

Using this, we can write
#py("adjoin_set").
If the object to be adjoined is already in the set, we just return the set.
Otherwise, we use
#py("pair")
to add the object to the
linked list
that represents the set:
#idx("adjoinset", sub: "unordered-linked-list representation", decl: true)
#snippet(```python
def adjoin_set(x, set):
    return (set
            if is_element_of_set(x, set)
            else pair(x, set))
```)

For
#py("intersection_set")
we can use a recursive strategy. If we know how to form the intersection
of #py("set2") and the
#py("tail")
of #py("set1"), we only need to decide whether to
include the
#py("head")
of #py("set1") in this. But this depends on whether
#py("head(set1)")
is also in #py("set2"). Here is the resulting
function:
#idx("intersectionset", sub: "unordered-linked-list representation", decl: true)
#snippet(```python
def intersection_set(set1, set2):
    return (None
            if is_none(set1) or is_none(set2)
            else pair(head(set1), intersection_set(tail(set1), set2))
            if is_element_of_set(head(set1), set2)
            else intersection_set(tail(set1), set2))
```)

In designing a representation, one of the issues we should be concerned
with is efficiency. Consider the number of steps required by our set
operations. Since they all use
#py("is_element_of_set"),
the speed of this operation has a major impact on the efficiency of the set
implementation as a whole. Now, in order to check whether an object is a
member of a set,
#py("is_element_of_set")
may have to scan the entire set. (In the worst case, the object turns out
not to be in the set.) Hence, if the set has
$n$ elements,
#py("is_element_of_set")
might take up to $n$ steps. Thus, the number of
steps required grows as $Theta (n)$. The number
of steps required by
#py("adjoin_set"),
which uses
this operation, also grows as $Theta (n)$. For
#py("intersection_set"),
which does an
#py("is_element_of_set")
check for each element of #py("set1"), the number of
steps required grows as the product of the sizes of the sets involved, or
$Theta (n^(2))$ for two sets of size
$n$. The same will be true of
#py("union_set").

#exercise(label-name: <ex:2_59>, [
Implement the
#idx("unionset", sub: "unordered-linked-list representation")
#py("union_set")
operation for the
unordered-linked-list
representation of sets.
])

#exercise(label-name: <ex:2_60>, [
We specified that a set would be represented as a
linked list
with no duplicates.
Now suppose we allow duplicates. For instance, the set
$\{1,2,3\}$ could be represented as the
linked list
#py("llist(2, 3, 2, 1, 3, 2, 2)").
Design
functions
#py("is_element_of_set"),
#py("adjoin_set"),
#py("union_set"),
and
#py("intersection_set")
that operate on this representation. How does the efficiency of each
compare with the corresponding
function
for the non-duplicate representation? Are there applications for which
you would use this representation in preference to the non-duplicate one?
])

#idx("set", sub: "represented as unordered linked list")
#idx("unordered-linked-list representation of sets")

#subheading([Sets as ordered linked lists])

#idx("set", sub: "represented as ordered linked list")
#idx("ordered-linked-list representation of sets")

One way to speed up our set operations is to change the representation
so that the set elements are listed in increasing order. To do this,
we need some way to compare two objects so that we can say which is
bigger. For example, we could compare
strings
lexicographically, or
we could agree on some method for assigning a unique number to an
object and then compare the elements by comparing the corresponding
numbers. To keep our discussion simple, we will consider only the
case where the set elements are numbers, so that we can compare
elements using #py(">") and
#py("<"). We will represent a set of
numbers by listing its elements in increasing order. Whereas our
first representation above allowed us to represent the set
$\{1,3,6,10\}$ by listing the elements in any
order, our new representation allows only the
linked list
#py("llist(1, 3, 6, 10)").

One advantage of ordering shows up in
#py("is_element_of_set"):
In checking for the presence of an item, we no longer have to scan the
entire set. If we reach a set element that is larger than the item we
are looking for, then we know that the item is not in the set:
#idx("iselementofset", sub: "ordered-linked-list representation", decl: true)
#syntax("
def is_element_of_set(x, set):
    return (False
            if is_none(set)
            else True
            if x == head(set)
            else False
            if x < head(set)
            # ", $mono("x > head(set)")$, "
            else is_element_of_set(x, tail(set)))
      ")

How many steps does this save? In the worst case, the item we are
looking for may be the largest one in the set, so the number of steps
is the same as for the unordered representation. On the other hand,
if we search for items of many different sizes we can expect that
sometimes we will be able to stop searching at a point near the
beginning of the
linked list
and that other times we will still need to
examine most of the
linked list.
On the average we should expect to have to
examine about half of the items in the set. Thus, the average
number of steps required will be about $n/2$.
This is still $Theta (n)$ growth, but
it does save us, on the average, a factor of 2 in number of steps over the
previous implementation.

We obtain a more impressive speedup with
#py("intersection_set").
In the unordered representation this operation required
$Theta (n^(2))$ steps, because we performed a
complete scan of #py("set2") for each element of
#py("set1"). But with the ordered representation,
we can use a more clever method. Begin by comparing the initial elements,
#py("x1") and
#py("x2"), of the two sets. If
#py("x1") equals
#py("x2"), then that gives an element of the
intersection, and the rest of the intersection is the intersection of the
#py("tail")s
of the two sets. Suppose, however, that #py("x1")
is less than #py("x2"). Since
#py("x2") is the smallest element in
#py("set2"), we can immediately conclude that
#py("x1") cannot appear anywhere in
#py("set2") and hence is not in the intersection.
Hence, the intersection is equal to the intersection of
#py("set2") with the
#py("tail")
of #py("set1"). Similarly, if
#py("x2") is less than
#py("x1"), then the intersection is given by the
intersection of #py("set1") with the
#py("tail")
of #py("set2"). Here is the
function:
#idx("intersectionset", sub: "ordered-linked-list representation", decl: true)
#syntax("
def intersection_set(set1, set2):
    if is_none(set1) or is_none(set2):
        return None
    else:
        x1 = head(set1)
        x2 = head(set2)
        return (pair(x1, intersection_set(tail(set1), tail(set2)))
                if x1 == x2
                else intersection_set(tail(set1), set2)
                if x1 < x2
                # ", $mono("x2 < x1")$, "
                else intersection_set(set1, tail(set2)))
      ")

To estimate the number of steps required by this process, observe that at
each step we reduce the intersection problem to computing intersections of
smaller sets—removing the first element from
#py("set1") or #py("set2")
or both. Thus, the number of steps required is at most the sum of the sizes
of #py("set1") and #py("set2"),
rather than the product of the sizes as with the unordered representation.
This is $Theta (n)$ growth rather than
$Theta (n^(2))$—a considerable speedup,
even for sets of moderate size.

#exercise(label-name: <ex:adjoin-set>, [
Give an implementation of

#idx("adjoinset", sub: "ordered-linked-list representation")
#py("adjoin_set")
using the ordered representation. By analogy with
#py("is_element_of_set")
show how to take advantage of the ordering to produce a
function
that requires on the average about half as many steps as with the unordered
representation.

#anchor(<ex:2_61>)
])

#exercise(label-name: <ex:union-set>, [
Give a $Theta (n)$ implementation of
#idx("unionset", sub: "ordered-linked-list representation")
#py("union_set")
for sets represented as ordered
linked lists.
])

#idx("set", sub: "represented as ordered linked list")
#idx("ordered-linked-list representation of sets")

#subheading([Sets as binary trees])

#idx("set", sub: "represented as binary tree")
#idx("binary tree", sub: "set represented as")
#idx("tree", sub: "binary")
#idx("binary tree")
#idx("binary search")
#idx("search", sub: "of binary tree")

We can do better than the
ordered-linked-list
representation by arranging the set
elements in the form of a tree. Each node of the tree holds one element of
the set, called the "entry" at that node, and a link to each
of two other (possibly empty) nodes. The "left" link points to
elements smaller than the one at the node, and the "right"
link to elements greater than the one at the node.
Figure @fig:binary-tree shows some trees that represent
the set $\{1,3,5,7,9,11\}$. The same set may be
represented by a tree in a number of different ways. The only thing we
require for a valid representation is that all elements in the left subtree
be smaller than the node entry and that all elements in the right subtree be
larger.

#sicp-figure(image("/images/img_original/ch2-Z-G-51.svg", width: 70%), caption: [Various binary trees that represent the set $\{ 1,3,5,7,9,11 \}$.], label-name: <fig:binary-tree>)

The advantage of the tree representation is this: Suppose we want to check
whether a number $x$ is contained in a set. We
begin by comparing $x$ with the entry in the
top node. If $x$ is less than this, we know
that we need only search the left subtree; if $x$
is greater, we need only search the right subtree. Now, if the tree is
"balanced," each of these subtrees will be about half the size
of the original. Thus, in one step we have reduced the problem of
searching a tree of size $n$ to searching a tree
of size $n/2$. Since the size of the tree is
halved at each step, we should expect that the number of steps needed to
search a tree of size $n$ grows as
$Theta ( log n)$.#footnote[Halving the size of
the problem at each step is the distinguishing characteristic of
#idx("logarithmic growth")
logarithmic growth, as we saw with the fast-exponentiation algorithm of
section @sec:exponentiation and the half-interval
search method of
section @sec:proc-general-methods.] For
large sets, this will be a significant speedup over the previous
representations.

We can represent trees by using
#idx("binary tree", sub: "represented with linked lists")
linked lists.
Each node will be a
linked list
of
three items: the entry at the node, the left subtree, and the right
subtree. A left or a right subtree of the empty
linked list
will indicate
that there is no subtree connected there. We can describe this
representation by the following
functions:#footnote[We
are representing sets in terms of trees, and trees in terms
of
linked lists—in
effect, a data abstraction built upon a data abstraction.
We can regard the
functions
#py("entry"),
#py("left_branch"),
#py("right_branch"),
and
#py("make_tree")
as a way of isolating the abstraction of a "binary tree" from
the particular way we might wish to represent such a tree in terms of
linked-list
structure.]
#idx("entry", decl: true)#idx("leftbranch", decl: true)#idx("rightbranch", decl: true)#idx("maketree", decl: true)
#snippet(```python
def entry(tree): return head(tree)

def left_branch(tree): return head(tail(tree))

def right_branch(tree): return head(tail(tail(tree)))

def make_tree(entry, left, right):
    return llist(entry, left, right)
```)

Now we can write
#py("is_element_of_set")
using the strategy described above:
#idx("iselementofset", sub: "binary-tree representation", decl: true)
#syntax("
def is_element_of_set(x, set):
    return (False
            if is_none(set)
            else True
            if x == entry(set)
            else is_element_of_set(x, left_branch(set))
            if x < entry(set)
            # ", $mono("x > entry(set)")$, "
            else is_element_of_set(x, right_branch(set)))
      ")

Adjoining an item to a set is implemented similarly and also requires
$Theta ( log n)$ steps. To adjoin an item
#py("x"), we compare
#py("x") with the node entry to determine whether
#py("x") should be added to the right or to the left
branch, and having adjoined
#py("x") to the appropriate branch we piece this
newly constructed branch together with the original entry and the other
branch. If #py("x") is equal to the entry, we just
return the node. If we are asked to adjoin
#py("x") to an empty tree, we generate a tree that
has #py("x") as the entry and empty right and left
branches. Here is the
function:

#idx("adjoinset", sub: "binary-tree representation", decl: true)
#syntax("
def adjoin_set(x, set):
    return (make_tree(x, None, None)
            if is_none(set)
            else set
            if x == entry(set)
            else make_tree(entry(set),
                           adjoin_set(x, left_branch(set)),
                           right_branch(set))
            if x < entry(set)
            # ", $mono("x > entry(set)")$, "
            else make_tree(entry(set),
                           left_branch(set),
                           adjoin_set(x, right_branch(set))))
      ")

The above claim that searching the tree can be performed in a logarithmic
number of steps rests on the assumption that the tree is
#idx("balanced binary tree")
#idx("binary tree", sub: "balanced")
"balanced," i.e., that the
left and the right subtree of every tree have approximately the same
number of elements, so that each subtree contains about half the
elements of its parent. But how can we be certain that the trees we
construct will be balanced? Even if we start with a balanced tree,
adding elements with
#py("adjoin_set")
may produce an unbalanced result. Since the position of a newly adjoined
element depends on how the element compares with the items already in the
set, we can expect that if we add elements "randomly" the tree
will tend to be balanced on the average. But this is not a guarantee. For
example, if we start with an empty set and adjoin the numbers 1 through 7
in sequence we end up with the highly unbalanced tree shown in
figure @fig:unbalanced-tree. In this tree all the left
subtrees are empty, so it has no advantage over a simple ordered
linked list.
One
way to solve this problem is to define an operation that transforms an
arbitrary tree into a balanced tree with the same elements. Then we can perform this transformation after every few
#py("adjoin_set")
operations to keep our set in balance. There are also other ways to solve
this problem, most of which involve designing new data structures for which
searching and insertion both can be done in
$Theta ( log n)$
steps.#footnote[Examples of such structures include
#idx("tree", sub: "B-tree")
#idx("tree", sub: "red-black")
#idx("B-tree")
#idx("red-black tree")
#emph[B-trees] and #emph[red-black trees]. There is a large literature
on data structures devoted to this problem. See
#idx("Cormen, Thomas H.")
#idx("Leiserson, Charles E.")
#idx("Rivest, Ronald L.")
#idx("Stein, Clifford")
Cormen, Leiserson, Rivest, and Stein 2022.]

#exercise(label-name: <ex:tree-to-list>, [
Each of the following two
functions
converts a
#idx("binary tree", sub: "converting to a linked list")
#idx("linked list", sub: "converting a binary tree to a")
binary tree to a
linked list.
#idx("treetolinkedlist…", decl: true)
#snippet(```python
def tree_to_linked_list_1(tree):
    return (None
            if is_none(tree)
            else append(tree_to_linked_list_1(left_branch(tree)),
                        pair(entry(tree),
                             tree_to_linked_list_1(right_branch(tree)))))
```)

#snippet(```python
def tree_to_linked_list_2(tree):
    def copy_to_linked_list(tree, result_list):
        return (result_list
                if is_none(tree)
                else copy_to_linked_list(left_branch(tree),
                         pair(entry(tree),
                              copy_to_linked_list(right_branch(tree),
                                           result_list))))
    return copy_to_linked_list(tree, None)
```)

+ Do the two functions produce the same result for every tree? If not, how do the results differ? What linked lists do the two functions produce for the trees in figure @fig:binary-tree?
+ Do the two functions have the same order of growth in the number of steps required to convert a balanced tree with $n$ elements to a linked list? If not, which one grows more slowly?
])

#sicp-figure(image("/images/img_original/ch2-Z-G-52.svg", width: 70%), caption: [Unbalanced tree produced by adjoining 1 through 7 in sequence.], label-name: <fig:unbalanced-tree>)

#exercise(label-name: <ex:list-to-tree>, [
The following
function
#py("linked_list_to_tree")
#idx("binary tree", sub: "converting a linked list to a")
#idx("linked list", sub: "converting to a binary tree")
converts an ordered
linked list
to a balanced binary tree. The helper
function
#py("partial_tree")
takes as arguments an integer $n$ and
linked list
of
at least $n$ elements and constructs a balanced
tree containing the first $n$ elements of the
linked list.
The result returned by
#py("partial_tree")
is a pair (formed with
#py("pair"))
whose
#py("head")
is the constructed tree and whose
#py("tail")
is the
linked list
of elements not included in the tree.
#idx("linkedlisttotree", decl: true)
#snippet(```python
def linked_list_to_tree(elements):
    return head(partial_tree(elements, length(elements)))
def partial_tree(elts, n):
    if n == 0:
        return pair(None, elts)
    else:
        left_size = (n - 1) // 2
        left_result = partial_tree(elts, left_size)
        left_tree = head(left_result)
        non_left_elts = tail(left_result)
        right_size = n - (left_size + 1)
        this_entry = head(non_left_elts)
        right_result = partial_tree(tail(non_left_elts), right_size)
        right_tree = head(right_result)
        remaining_elts = tail(right_result)
        return pair(make_tree(this_entry, left_tree, right_tree),
                    remaining_elts)
```)

+ Write a short paragraph explaining as clearly as you can how #py("partial_tree") works. Draw the tree produced by #py("linked_list_to_tree") for the linked list #py("llist(1, 3, 5, 7, 9, 11)").
+ What is the order of growth in the number of steps required by #py("linked_list_to_tree") to convert a linked list of $n$ elements?
])

#exercise(label-name: <ex:tree-ops>, [
Use the results of exercises @ex:tree-to-list
and @ex:list-to-tree to give
$Theta (n)$ implementations of
#idx("unionset", sub: "binary-tree representation")
#py("union_set")
and
#idx("intersectionset", sub: "binary-tree representation")
#py("intersection_set")
for sets implemented as (balanced) binary trees.#footnote[Exercises @ex:tree-to-list–@ex:tree-ops
are due to
#idx("Hilfinger, Paul")
Paul Hilfinger.]
])

#idx("set", sub: "represented as binary tree")
#idx("binary tree", sub: "set represented as")

#subheading([Sets and information retrieval])

We have examined options for using
linked lists
to represent sets and have
seen how the choice of representation for a data object can have a
large impact on the performance of the programs that use the data.
Another reason for concentrating on sets is that the techniques
discussed here appear again and again in applications involving
information retrieval.

#idx("data base", sub: "as set of records")
#idx("set", sub: "data base as")
Consider a data base containing a large number of individual records,
#idx("record, in a data base")
such as the personnel files for a company or the transactions in an
accounting system. A typical data-management system spends a large
amount of time accessing or modifying the data in the records and
therefore requires an efficient method for accessing records. This is
done by identifying a part of each record to serve as an identifying
#idx("key of a record", sub: "in a data base")
#emph[key]. A key can be anything that uniquely identifies the
record. For a personnel file, it might be an employee's ID number.
For an accounting system, it might be a transaction number. Whatever
the key is, when we define the record as a data structure we should
include a
#idx("key")
#py("key") selector
function
that retrieves the key associated with a given record.

Now we represent the data base as a set of records. To locate the record
with a given key we use a
function
#py("lookup"), which takes as arguments a key and a
data base and which returns the record that has that key, or false if there
is no such record.
The function #py("lookup")
is implemented in almost the same way as
#py("is_element_of_set").
For example, if the set of records is implemented as an unordered
linked list
we
could use

#idx("lookup", sub: "in set of records", decl: true)
#snippet(```python
def lookup(given_key, set_of_records):
    return (False
            if is_none(set_of_records)
            else head(set_of_records)
            if given_key == key(head(set_of_records))
            else lookup(given_key, tail(set_of_records)))
```)

Of course, there are better ways to represent large sets than as unordered
linked lists.
Information-retrieval systems in which records have to be
"randomly accessed" are typically implemented by a tree-based
method, such as the binary-tree representation discussed previously.
In designing such a system the methodology of data abstraction
can be a great help. The designer can create an initial implementation
using a simple, straightforward representation such as unordered
linked lists.
This will be unsuitable for the eventual system, but it can be useful in
providing a "quick and dirty" data base with which to test the
rest of the system. Later on, the data representation can be modified to
be more sophisticated. If the data base is accessed in terms of abstract
selectors and constructors, this change in representation will not require
any changes to the rest of the system.

#exercise(label-name: <ex:set-lookup-binary-tree>, [
Implement the #py("lookup")
function
for the case where the set of records is structured as a binary tree,
ordered by the numerical values of the keys.
])
