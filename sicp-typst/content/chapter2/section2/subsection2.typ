// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Hierarchical Structures], label-name: <sec:trees>)

#idx("data", sub: "hierarchical")
#idx("hierarchical data structures")
#idx("tree", sub: "represented as pairs")
#idx("pair(s)", sub: "used to represent tree")

The representation of sequences in terms of linked lists generalizes naturally to
represent sequences whose elements may themselves be sequences. For
example, we can regard the object
#py("[[1, [2, None]], [3, [4, None]]]")
constructed by

#snippet(```python
pair(llist(1, 2), llist(3, 4))
```)

as a linked list of three items, the first of which is itself a linked list,
#py("[1, [2, None]]").
Figure @fig:cons-of-2-lists
shows the representation of this structure in terms of pairs.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-15.svg", width: 70%), caption: [Structure formed by #py("pair(llist(1, 2), llist(3, 4))").], label-name: <fig:cons-of-2-lists>)

Another way to think of sequences whose elements are sequences is as
#emph[trees]. The elements of the sequence are the branches of the
tree, and elements that are themselves sequences are subtrees.
Figure @fig:list-as-tree
shows the structure in
figure @fig:cons-of-2-lists
viewed as a tree.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-16.svg", width: 70%), caption: [The linked-list structure in figure @fig:cons-of-2-lists viewed as a tree.], label-name: <fig:list-as-tree>)

Recursion
#idx("recursion", sub: "in working with trees")
is a natural tool for dealing with tree structures, since we can
often reduce operations on trees to operations on their branches, which
reduce in turn to operations on the branches of the branches, and so on,
until we reach the leaves of the tree. As an example, compare the
#py("length")
function
of section @sec:sequences with the
#idx("countleaves")
#idx("tree", sub: "counting leaves of")
#py("count_leaves")
function,
which returns the total number of leaves of a tree:

#snippet(```python
x = pair(llist(1, 2), llist(3, 4))
```)

#snippet(```python
print(length(x))
```)

#output(```python
print(length(x))
```)

#snippet(```python
print(count_leaves(x))
```)

#output(```python
print(count_leaves(x))
```)

#snippet(```python
print_llist(llist(x, x))
```)

#output(```python
print_llist(llist(x, x))
```)

#snippet(```python
print(length(llist(x, x)))
```)

#output(```python
print(length(llist(x, x)))
```)

#snippet(```python
print(count_leaves(llist(x, x)))
```)

#output(```python
print(count_leaves(llist(x, x)))
```)

To implement
#py("count_leaves"),
recall the recursive plan for computing
#py("length"):

- The #py("length") of a linked list #py("x") is 1 plus the #py("length") of the #py("tail") of #py("x").
- The #py("length") of the empty linked list is 0.

The function #py("count_leaves")
is similar. The value for the empty linked list is the same:

- #py("count_leaves") of the empty linked list is 0.

But in the reduction step, where we strip off the
#py("head")
of the linked list, we must take into account that the
#py("head")
may itself be a tree whose leaves we need to count. Thus, the appropriate
reduction step is

- #py("count_leaves") of a tree #py("x") is #py("count_leaves") of the #py("head") of #py("x") plus #py("count_leaves") of the #py("tail") of #py("x").

Finally, by taking
#py("head")s
we reach actual leaves, so we need another base case:

- #py("count_leaves") of a leaf is 1.

To aid in writing recursive
functions
on trees,
our Python environment
provides the primitive predicate
#idx("ispair (primitive function)")

#py("is_pair"),
which tests whether its argument is a pair. Here is the complete
function:#footnote[The order of the two predicates matters, since #py("None") satisfies #py("is_none") and also is not a pair.]
#idx("countleaves", decl: true)
#snippet(```python
def count_leaves(x):
    return (0 if is_none(x)
            else 1 if not is_pair(x)
            else count_leaves(head(x)) + count_leaves(tail(x)))
```)

#exercise(label-name: <ex:nested-list>, [
Suppose we evaluate the expression
#py("llist(1, llist(2, llist(3, 4)))").
Give the result printed by the interpreter, the corresponding
box-and-pointer structure, and the interpretation of this as a tree (as in
figure @fig:list-as-tree).

#anchor(<ex:2_24>)
])

#exercise(label-name: <ex:2_25>, [
Give combinations of
#py("head")s
and
#py("tail")s
that will pick 7 from each of the following
linked lists, given in linked-list notation:

#snippet(```python
llist(1, 3, llist(5, 7), 9)

llist(llist(7))

llist(1, llist(2, llist(3, llist(4, llist(5, llist(6, 7))))))
```)
])

#exercise(label-name: <ex:2_26>, [
Suppose we define #py("x") and
#py("y") to be two linked lists:

#snippet(```python
x = llist(1, 2, 3)

y = llist(4, 5, 6)
```)

What is the result of evaluating each of the following expressions, in box notation and linked-list notation?

#snippet(```python
append(x, y)
```)

#snippet(```python
pair(x, y)
```)

#snippet(```python
llist(x, y)
```)
])

#exercise(label-name: <ex:2_27>, [
Modify your
#py("reverse")
function
of exercise @ex:reverse to produce a
#idx("deepreverse")
#idx("tree", sub: "reversing at all levels")
#py("deep_reverse")
function
that takes a linked list as argument and returns as its value the linked list with its
elements reversed and with all sublists deep-reversed as well. For example,

#snippet(```python
x = llist(llist(1, 2), llist(3, 4))
```)

#snippet(```python
print_llist(x)
```)

#output(```python
print_llist(x)
```)

#snippet(```python
print_llist(reverse(x))
```)

#output(```python
print_llist(reverse(x))
```)

#snippet(```python
print_llist(deep_reverse(x))
```)

#output(```python
print_llist(deep_reverse(x))
```)
])

#exercise(label-name: <ex:fringe>, [
Write a
function
#idx("fringe")
#idx("tree", sub: "fringe of")
#py("fringe")
that takes as argument a tree (represented as a linked list) and returns a linked list
whose elements are all the leaves of the tree arranged in left-to-right
order. For example,

#snippet(```python
x = llist(llist(1, 2), llist(3, 4))
```)

#snippet(```python
print_llist(fringe(x))
```)

#output(```python
print_llist(fringe(x))
```)

#snippet(```python
print_llist(fringe(llist(x, x)))
```)

#output(```python
print_llist(fringe(llist(x, x)))
```)
])

#exercise(label-name: <ex:mobile>, [
A binary
#idx("mobile")
mobile consists of two branches, a left branch and a right
branch. Each branch is a rod of a certain length, from which hangs
either a weight or another binary mobile. We can represent a binary
mobile using compound data by constructing it from two branches (for
example, using #py("llist")):

#snippet(```python
def make_mobile(left, right):
    return llist(left, right)
```)

A branch is constructed from a #py("length") (which
must be a number) together with a #py("structure"),
which may be either a number (representing a simple weight) or another
mobile:

#snippet(```python
def make_branch(length, structure):
    return llist(length, structure)
```)

+ Write the corresponding selectors #py("left_branch") and #py("right_branch"), which return the branches of a mobile, and #py("branch_length") and #py("branch_structure"), which return the components of a branch.
+ Using your selectors, define a function #py("total_weight") that returns the total weight of a mobile.
+ A mobile is said to be #idx("balanced mobile") #emph[balanced] if the torque applied by its top-left branch is equal to that applied by its top-right branch (that is, if the length of the left rod multiplied by the weight hanging from that rod is equal to the corresponding product for the right side) and if each of the submobiles hanging off its branches is balanced. Design a predicate that tests whether a binary mobile is balanced.
+ Suppose we change the representation of mobiles so that the constructors are #snippet(```python def make_mobile(left, right): return pair(left, right) def make_branch(length, structure): return pair(length, structure) ```) How much do you need to change your programs to convert to the new representation?
])

#idx("data", sub: "hierarchical")
#idx("hierarchical data structures")
#idx("tree", sub: "represented as pairs")
#idx("pair(s)", sub: "used to represent tree")

#subheading([Mapping over trees])

#idx("tree", sub: "mapping over")
#idx("mapping", sub: "over trees")

Just as #py("map") is a powerful abstraction for
dealing with sequences, #py("map") together with
recursion is a powerful abstraction for dealing with trees. For instance,
the
#py("scale_tree")
function,
analogous to
#py("scale_linked_list")
of section @sec:sequences, takes as arguments a numeric
factor and a tree whose leaves are numbers. It returns a tree of the same
shape, where each number is multiplied by the factor. The recursive plan
for
#py("scale_tree")
is similar to the one for
#py("count_leaves"):
#idx("scaletree", decl: true)
#snippet(```python
def scale_tree(tree, factor):
    return (None if is_none(tree)
            else tree * factor if not is_pair(tree)
            else pair(scale_tree(head(tree), factor),
                      scale_tree(tail(tree), factor)))
```)

#snippet(```python
print_llist(scale_tree(llist(1, llist(2, llist(3, 4), 5), llist(6, 7)),
                       10))
```)

#output(```python
print_llist(scale_tree(llist(1, llist(2, llist(3, 4), 5), llist(6, 7)),
                       10))
```)

Another way to implement
#py("scale_tree")
is to regard the tree as a sequence of sub-trees and use
#py("map").
We map over the sequence, scaling each sub-tree in turn, and return the
linked list of results. In the base case, where the tree is a leaf, we simply
multiply by the factor:

#idx("scaletree", decl: true)
#snippet(```python
def scale_tree(tree, factor):
    return map(lambda sub_tree: (scale_tree(sub_tree, factor)
                                 if is_pair(sub_tree)
                                 else sub_tree * factor),
               tree)
```)

Many tree operations can be implemented by similar combinations of
sequence operations and recursion.

#exercise(label-name: <ex:square-tree>, [
Declare a function
#py("square_tree")
analogous to the
#py("square_linked_list")
function
of exercise @ex:square-list. That is,
#py("square_tree")
should behave as follows:

#snippet(```python
print_llist(square_tree(llist(1,
                        llist(2, llist(3, 4), 5),
                        llist(6, 7))))
```)

#output(```python
print_llist(square_tree(llist(1,
                        llist(2, llist(3, 4), 5),
                        llist(6, 7))))
```)

Declare #py("square_tree")
both directly (i.e., without using any higher-order
functions)
and also by using
#py("map") and recursion.
])

#exercise(label-name: <ex:tree-map>, [
Abstract your answer to exercise @ex:square-tree to
produce a
function
#idx("treemap")
#py("tree_map")
with the property that
#py("square_tree") could be declared as

#snippet(```python
def square_tree(tree): return tree_map(square, tree)
```)
])

#exercise(label-name: <ex:2_32>, [
We can represent a
#idx("set", sub: "subsets of")
set as a linked list of distinct elements, and we can
represent the set of all subsets of the set as a linked list of linked lists. For
example, if the set is
#py("llist(1, 2, 3)"),
then the set of all subsets is
#snippet(```python llist(None, llist(3), llist(2), llist(2, 3), llist(1), llist(1, 3), llist(1, 2), llist(1, 2, 3)) ```)
Complete the
following
declaration of a function
that generates the set of subsets of a set and give a clear explanation of
why it works:
#idx("subsets of a set", decl: true)
#syntax("
def subsets(s):
    if is_none(s):
        return llist(None)
    else:
        rest = subsets(tail(s))
        return append(rest, map(", metaphrase[??], ", rest))
      ")
])

#idx("tree", sub: "mapping over")
#idx("mapping", sub: "over trees")
