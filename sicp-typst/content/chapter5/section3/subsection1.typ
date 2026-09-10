// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Memory as Vectors], label-name: <sec:memory-as-vectors>)

A conventional computer memory can be thought of as an array of
cubbyholes, each of which can contain a piece of information. Each
cubbyhole has a unique name, called its
#idx("address")
#emph[address] or
#idx("location")
#emph[location]. Typical memory systems provide two primitive operations:
one that fetches the data stored in a specified location and one that
assigns new data to a specified location. Memory addresses can be
incremented to support sequential access to some set of the
cubbyholes. More generally, many important data operations require
that memory addresses be treated as data, which can be stored in
memory locations and manipulated in machine registers. The
representation of list structure is one application of such
#idx("address arithmetic")
#idx("arithmetic", sub: "address arithmetic")
#emph[address arithmetic].

To model computer memory, we use a new kind of data structure called a
#idx("vector (data structure)")
#emph[vector]. Abstractly, a vector is a compound data object whose
individual elements can be accessed by means of an integer index in an
amount of time that is independent of the index.#footnote[We could represent
memory as lists of items. However, the access time would then not be
independent of the index, since accessing the
$n$th element of a list requires
$n-1$
#py("tail")
operations.] In order to describe memory operations, we use two
functions
for manipulating vectors:#footnote[As mentioned in section @sec:running-eval (footnote @foot:vector-array), Python supports vectors as data structures and calls them "arrays." We use the term #emph[vector] in this book, as it is the more common terminology. The vector functions above are easily implemented using Python's primitive array support.]

- #py("vector_ref(")#meta("vector")#py(",")#meta("n")#py(")") #idx("vectorref (primitive function)") returns the #meta("n")th element of the vector.
- #py("vector_set(")#meta("vector")#py(",")#meta("n")#py(",")#meta("value")#py(")") #idx("vectorset (primitive function)") sets the $n$th element of the vector to the designated value.

For example, if #py("v") is a vector, then
#py("vector_ref(v, 5)")
gets the fifth entry in the vector #py("v") and
#py("vector_set(v, 5, 7)")
changes the value of the fifth entry of the vector
#py("v")
to 7.#footnote[For completeness, we should specify a
#py("make_vector")
operation that constructs vectors. However, in the present application we
will use vectors only to model fixed divisions of the computer
memory.] For computer memory, this access can be implemented
through the use of address arithmetic to combine a #emph[base address]
that specifies the beginning location of a vector in memory with an
#emph[index] that specifies the offset of a particular element of the
vector.

#subheading([Representing data])

#idx("pair(s)", sub: "represented using vectors")
#idx("list structure", sub: "represented using vectors")

#sicp-figure(image("/images/img_javascript/Fig5.14b.std.svg", width: 70%), caption: [Box-and-pointer and memory-vector representations of the list #py("list(list(1, 2), 3, 4)").], label-name: <fig:box-and-pointer-memory>)

We can use vectors to implement the basic pair structures required for a
list-structured memory. Let us imagine that computer memory is divided into
two vectors:
#idx("theheads", sub: "vector")
#py("the_heads")
and
#idx("thetails", sub: "vector")
#py("the_tails").
We will represent list structure as follows: A pointer to a pair is an index
into the two vectors. The
#py("head")
of the pair is the entry in
#py("the_heads")
with the designated index, and the
tail
of the pair is the entry in
#py("the_tails")
with the designated index. We also need a representation for objects other
than pairs (such as numbers and
strings)
and a way to distinguish one kind of data from another. There are many
methods of accomplishing this, but they all reduce to using
#idx("typed pointer")
#idx("pointer", sub: "typed")
#emph[typed pointers], that is, to extending the notion of
"pointer" to include information on data type.#footnote[This is
precisely the same
#idx("tagged data")
#idx("data", sub: "tagged")
"tagged data" idea we introduced in chapter @chap:data for
dealing with generic operations. Here, however, the data types are
included at the primitive machine level rather than constructed
through the use of lists.

Type information may be encoded in
a variety of ways, depending on the details of the machine on which the
Python
system is to be implemented. The execution efficiency of
Python
programs will be strongly dependent on how cleverly this choice is made, but
it is difficult to formulate general design rules for good choices. The
most straightforward way to implement typed pointers is to allocate a fixed
set of bits in each pointer to be a
#idx("type field")
#emph[type field] that encodes the data type. Important questions to be
addressed in designing such a representation include the following:
How many type bits are required? How large must the vector indices
be? How efficiently can the primitive machine instructions be used to
manipulate the type fields of pointers? Machines that include special
hardware for the efficient handling of type fields are said to have
#idx("tagged architecture")
#emph[tagged architectures].] The data type enables the system to
distinguish a pointer to a pair (which consists of the "pair"
data type and an index into the memory vectors) from pointers to other
kinds of data (which consist of some other data type and whatever is
being used to represent data of that type). Two data objects are
#idx("===", sub: "as equality of pointers")
considered to be the same
(#py("==="))
if their pointers are identical.
Figure @fig:box-and-pointer-memory
illustrates the use of this method to represent
#py("list(list(1, 2), 3, 4)"),
whose box-and-pointer diagram is also shown. We use letter prefixes to
denote the data-type information. Thus, a pointer to the pair with
index 5 is denoted #py("p5"), the empty list
is denoted by the pointer #py("e0"), and a pointer to
the number 4 is denoted #py("n4"). In the
box-and-pointer diagram, we have indicated at the lower left of each pair
the vector index that specifies where the
#py("head")
and
#py("tail")
of the pair are stored. The blank locations in
#py("the_heads")
and
#py("the_tails")

may contain parts of other list structures (not of interest here).

A pointer to a number, such as #py("n4"),
might consist of a type indicating numeric data together with the
actual representation of the number 4.#footnote[This decision on the
representation of numbers determines whether
#idx("===", sub: "as numeric equality operator")
#idx("equality", sub: "of numbers")
#idx("number(s)", sub: "equality of")
#py("==="),
which tests equality of pointers, can be used to test for equality of
numbers. If the pointer contains the number itself, then equal numbers will
have the same pointer. But if the pointer contains the index of a location
where the number is stored, equal numbers will be guaranteed to have
equal pointers only if we are careful never to store the same number
in more than one location.]
To deal with numbers that are too large to be represented in the fixed
amount of space allocated for a single pointer, we could use a distinct
#idx("bignum")
#idx("number(s)", sub: "bignum")
#emph[bignum] data type, for which the pointer designates a list in which
the parts of the number are stored.#footnote[This is just like writing a
number as a sequence of digits, except that each "digit" is a
number between 0 and the largest number that can be stored in a single
pointer.]

A string
#idx("string(s)", sub: "representation of")
might be represented as a typed pointer that designates a
sequence of the characters that form the string's printed
representation. The parser constructs such a sequence
when it encounters a string literal, and the
string-concatenation operator #py("+") and
string-producing
primitive functions such as
#py("stringify")
construct such a sequence.
Since we want two instances of a string to
be recognized as the "same" string by
#py("===") and we want
#idx("===", sub: "as string comparison operator")
#idx("equality", sub: "of strings")
#py("===")
to
be a simple test for equality of pointers, we must ensure that if the
system sees the same string twice, it will use the same pointer (to
the same sequence of characters) to represent both occurrences. To
accomplish this, the system maintains a table, called the
#idx("string pool")
#emph[string pool],
of all the strings it has ever encountered. When the system
is about to construct a string, it checks the string pool to see if it has ever
before seen the same string. If it has not, it
constructs a new string (a typed pointer to a new
character sequence) and enters this pointer in the string pool. If the
system has seen the string before, it returns the string pointer
stored in the string pool. This process of replacing strings by unique
pointers is called
#idx("interning strings")
#idx("string(s)", sub: "interning")
#emph[string interning].

#subheading([Implementing the primitive list operations])

#anchor(<sec:impl-list-ops>)

Given the above representation scheme, we can replace each
"primitive" list operation of a register machine with one or
more primitive vector operations. We will use two registers,
#idx("theheads", sub: "register")
#py("the_heads")
and
#idx("thetails", sub: "register")
#py("the_tails"),
to identify the memory vectors, and will
assume that
#py("vector_ref")
and
#py("vector_set")
are available as primitive operations. We also assume that numeric
operations on pointers (such as incrementing a pointer, using a pair pointer
to index a vector, or adding two numbers) use only the index portion of
the typed pointer.

For example, we can make a register machine support the instructions
#idx("head (primitive function)", sub: "implemented with vectors")#idx("tail (primitive function)", sub: "implemented with vectors")
#syntax("
assign(", meta("reg"), $""_(1)$, ", list(op(\"head\"), reg(", meta("reg"), $""_(2)$, ")))

assign(", meta("reg"), $""_(1)$, ", list(op(\"tail\"), reg(", meta("reg"), $""_(2)$, ")))
      ")

if we implement these, respectively, as

#syntax("
assign(", meta("reg"), $""_(1)$, ", list(op(\"vector_ref\"), reg(\"the_heads\"), reg(", meta("reg"), $""_(2)$, ")))

assign(", meta("reg"), $""_(1)$, ", list(op(\"vector_ref\"), reg(\"the_tails\"), reg(", meta("reg"), $""_(2)$, ")))
      ")

The instructions
#idx("sethead (primitive function)", sub: "implemented with vectors")#idx("settail (primitive function)", sub: "implemented with vectors")
#syntax("
perform(list(op(\"set_head\"), reg(", meta("reg"), $""_(1)$, "), reg(", meta("reg"), $""_(2)$, ")))

perform(list(op(\"set_tail\"), reg(", meta("reg"), $""_(1)$, "), reg(", meta("reg"), $""_(2)$, ")))
      ")

are implemented as

#syntax("
perform(list(op(\"vector_set\"), reg(\"the_heads\"), reg(", meta("reg"), $""_(1)$, "), reg(", meta("reg"), $""_(2)$, ")))

perform(list(op(\"vector_set\"), reg(\"the_tails\"), reg(", meta("reg"), $""_(1)$, "), reg(", meta("reg"), $""_(2)$, ")))
      ")

The operation #idx("pair (primitive function)", sub: "implemented with vectors") #py("pair")
is performed by allocating an unused index and storing the arguments to
#py("pair")
in
#py("the_heads")
and
#py("the_tails")
at that indexed vector position. We presume that there is a special
register,
#idx("free register")
#py("free"), that always holds a pair pointer
containing the next available index, and that we can increment the index
part of that pointer to find the next free location.#footnote[There are
other ways of finding free storage. For example, we could link together
all the unused pairs into a
#idx("free list")
#emph[free list]. Our free locations are consecutive (and hence can be
accessed by incrementing a pointer) because we are using a compacting
garbage collector, as we will see in
section @sec:gc.]
For example, the instruction

#syntax("
assign(", meta("reg"), $""_(1)$, ", list(op(\"pair\"), reg(", meta("reg"), $""_(2)$, "), reg(", meta("reg"), $""_(3)$, ")))
      ")

is implemented as the following sequence of vector
operations:#footnote[This is essentially the implementation of
#py("pair")
in terms of
#py("set_head")
and
#py("set_tail"),
as described in section @sec:mutable-list-structure.
The operation
#py("get_new_pair")
used in that implementation is realized here by the
#py("free") pointer.]

#syntax("
perform(list(op(\"vector_set\"),
             reg(\"the_heads\"), reg(\"free\"), reg(", meta("reg"), $""_(2)$, "))),
perform(list(op(\"vector_set\"),
             reg(\"the_tails\"), reg(\"free\"), reg(", meta("reg"), $""_(3)$, "))),
assign(", meta("reg"), $""_(1)$, ", reg(\"free\")),
assign(\"free\", list(op(\"+\"), reg(\"free\"), constant(1)))
      ")

The
#py("===")
operation

#syntax("
list(op(\"===\"), reg(", meta("reg"), $""_(1)$, "), reg(", meta("reg"), $""_(2)$, "))
      ")

simply tests the equality of all fields in the registers, and
predicates such as
#idx("ispair (primitive function)", sub: "implemented with typed pointers")
#py("is_pair"),
#idx("isnull (primitive function)", sub: "implemented with typed pointers")
#py("is_null"),
#idx("isstring (primitive function)", sub: "implemented with typed pointers")
#py("is_string"),
and
#idx("isnumber (primitive function)", sub: "implemented with typed pointers")
#py("is_number")
need only check the type field.

#subheading([Implementing stacks])

#idx("stack", sub: "representing")

Although our register machines use stacks, we need do nothing special
here, since stacks can be modeled in terms of lists. The stack can be

a list of the saved values, pointed to by a special register
#py("the_stack").
Thus,
#py("save(")#meta("reg")#py(")")
can be implemented as
#idx("save (in register machine)", sub: "implementing")
#syntax("
assign(\"the_stack\", list(op(\"pair\"), reg(", meta("reg"), "), reg(\"the_stack\")))
      ")

Similarly,
#py("restore(")#meta("reg")#py(")")
can be implemented as
#idx("restore (in register machine)", sub: "implementing")
#syntax("
assign(", meta("reg"), ", list(op(\"head\"), reg(\"the_stack\")))
assign(\"the_stack\", list(op(\"tail\"), reg(\"the_stack\")))
      ")

and
#py("perform(list(op(\"initialize_stack\")))")
can be implemented as

#syntax("
assign(\"the_stack\", constant(null))
      ")

These operations can be further expanded in terms of the vector
operations given above. In conventional computer architectures,
however, it is usually advantageous to allocate the stack as a
separate vector. Then pushing and popping the stack can be
accomplished by incrementing or decrementing an index into that
vector.

#exercise(label-name: <ex:5_19>, [
Draw the box-and-pointer representation and the memory-vector representation
(as in figure @fig:box-and-pointer-memory)
of the list structure produced by

#snippet(```python
const x = pair(1, 2);
const y = list(x, x);
```)

with the #py("free") pointer initially
#py("p1"). What is the final value of
#py("free")$thin$? What
pointers represent the values of #py("x") and
#py("y")?
])

#exercise(label-name: <ex:count-leaves-machine>, [
Implement register machines for the following
#idx("countleaves", sub: "as register machine")
functions.
Assume that the list-structure memory operations are available as
machine primitives.

+ Recursive #py("count_leaves"): #snippet(```python function count_leaves(tree) { return is_null(tree) ? 0 : ! is_pair(tree) ? 1 : count_leaves(head(tree)) + count_leaves(tail(tree)); } ```)
+ Recursive #py("count_leaves") with explicit counter: #snippet(```python function count_leaves(tree) { function count_iter(tree, n) { return is_null(tree) ? n : ! is_pair(tree) ? n + 1 : count_iter(tail(tree), count_iter(head(tree), n)); } return count_iter(tree, 0); } ```)
])

#exercise(label-name: <ex:5_21>, [
Exercise @ex:append of
section @sec:mutable-list-structure
presented an
#py("append")
function
that appends two lists to form a new list and an
#py("append_mutator") function
that splices two lists together. Design a register machine to
#idx("append", sub: "as register machine")
#idx("appendmutator", sub: "as register machine")
implement
each of these
functions.
Assume that the list-structure memory operations are
available as primitive operations.
])

#idx("pair(s)", sub: "represented using vectors")
#idx("list structure", sub: "represented using vectors")
