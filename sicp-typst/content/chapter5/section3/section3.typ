// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Storage Allocation and Garbage Collection], label-name: <sec:storage-allocation>)

#idx("list-structured memory")
#idx("memory", sub: "list-structured")

In section @sec:eceval, we will show how to implement a
Python
evaluator as a register machine. In order to simplify the discussion, we
will assume that our register machines can be equipped with a
#emph[list-structured memory], in which the basic operations for
manipulating list-structured data are primitive. Postulating the existence
of such a memory is a useful abstraction when one is focusing on the
mechanisms of control in
an
interpreter, but this does not reflect a realistic view of the actual
primitive data operations of contemporary computers. To obtain a more
complete picture of how
systems can support list-structured memory efficiently,
we must investigate how list structure can be represented in a way that is
compatible with conventional computer memories.

There are two considerations in implementing list structure. The first is
purely an issue of representation: how to represent the
"box-and-pointer" structure of

pairs, using only the storage and addressing capabilities of typical computer
memories. The second issue concerns the management of memory as a
computation proceeds. The operation of a
Python
system depends crucially on the ability to
continually create new data objects. These include objects that are
explicitly created by the
Python
functions
being interpreted as well as structures created by the interpreter itself,
such as environments and argument lists. Although the constant creation of
new data objects would pose no problem on a computer with an infinite amount
of rapidly addressable memory, computer memories are available only in
finite sizes (more's the pity).
Python
thus provide an
#idx("automatic storage allocation")
#emph[automatic storage allocation] facility to
support the illusion of an infinite memory. When a data object is no longer
needed, the memory allocated to it is automatically recycled and used to
construct new data objects. There are various techniques for providing such
automatic storage allocation. The method we shall discuss in this section
is called #emph[garbage collection].

#include "../../chapter5/section3/subsection1.typ"

#include "../../chapter5/section3/subsection2.typ"
