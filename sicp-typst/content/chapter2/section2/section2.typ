// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Hierarchical Data and the Closure Property], label-name: <sec:hierarchical-data>)

As we have seen, pairs provide a primitive "glue" that we can
use to construct compound data objects.
Figure @fig:first-box-and-pointer_js
shows a standard way to visualize a
#idx("pair(s)", sub: "box-and-pointer notation for")
pair—in this case, the pair formed by
#py("pair(1, 2)").

In this representation, which is called
#idx("box-and-pointer notation")
#emph[box-and-pointer notation], each compound object is shown as a
#idx("pointer", sub: "in box-and-pointer notation")
#emph[pointer] to a box. The box for a pair
has two parts, the left part containing the head of the pair and the
right part containing the tail.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-11.svg", width: 70%), caption: [Box-and-pointer representation of #py("pair(1, 2)").], label-name: <fig:first-box-and-pointer_js>)

We have already seen that
#py("pair")
can be used to combine not only numbers but pairs as well. (You made use
of this fact, or should have, in doing
exercises @ex:segments1
and @ex:rectangles.) As a consequence, pairs provide
a universal building block from which we can construct all sorts of data
structures.
Figure @fig:box-and-pointer-two-ways_js
shows two ways to use pairs to combine the numbers 1, 2, 3, and 4.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-12.svg", width: 70%), caption: [Two ways to combine 1, 2, 3, and 4 using pairs.], label-name: <fig:box-and-pointer-two-ways_js>)

The ability to create pairs whose elements are pairs is the essence of
linked-list structure's importance as a representational tool. We refer to
this ability as the
#idx("closure", sub: "closure property of pair")
#idx("pair (primitive function)", sub: "closure property of")
#emph[closure property] of
#py("pair").
In general, an operation for combining data objects satisfies the closure
property if the results of combining things with that operation can
themselves be combined using the same operation.#footnote[The use of the
word
#idx("closure", sub: "in abstract algebra")
"closure" here comes from abstract algebra, where a set of
elements is said to be
closed under an operation if applying the operation
to elements in the set produces an element that is again an element of the
set. The
programming languages
community also (unfortunately) uses the word "closure" to
describe a totally unrelated concept: A closure
is an implementation technique for representing
functions with free names.
We do not use the word "closure" in this second sense in this
book.]
Closure is the key to power in any means of combination because it permits
us to create
#idx("hierarchical data structures")
#idx("data", sub: "hierarchical")
#emph[hierarchical] structures—structures made up of parts, which
themselves are made up of parts, and so on.

From the outset of chapter @chap:fun, we've made essential use of
closure in dealing with
functions,
because all but the very simplest programs rely on the fact that the
elements of a combination can themselves be combinations. In this section,
we take up the consequences of closure for compound data. We describe some
conventional techniques for using pairs to represent sequences and trees,
and we exhibit a graphics language that illustrates closure in a vivid
way.

#include "../../chapter2/section2/subsection1.typ"

#include "../../chapter2/section2/subsection2.typ"

#include "../../chapter2/section2/subsection3.typ"

#include "../../chapter2/section2/subsection4.typ"
