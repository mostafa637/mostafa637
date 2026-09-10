// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Maintaining the Illusion of Infinite Memory], label-name: <sec:gc>)

#idx("garbage collection")

The representation method outlined in
section @sec:memory-as-vectors solves the problem of
implementing list structure, provided that we have an infinite amount of
memory. With a real computer we will eventually run out of free space in
which to construct new pairs.#footnote[This may not be true eventually,
because memories may get large enough so that it would be impossible
to run out of free memory in the lifetime of the computer. For
example, there are about
$3 times 10^(16)$ nanoseconds
in a year, so if we were to
#py("pair")
once per
nanosecond
we would need about
$10^(18)$
cells of memory to build a machine
that could operate for 30 years without running out of memory. That much
memory seems absurdly large by today's standards, but it is not
physically impossible. On the other hand, processors are getting faster and
modern computers have increasingly
large numbers of processors operating in
parallel on a single memory, so it may be possible to use up memory much
faster than we have postulated.] However, most of the pairs
generated in a typical computation are used only to hold intermediate
results. After these
results are accessed, the pairs are no longer needed—they are #emph[garbage]. For instance, the computation

#snippet(```python
reduce((x, y) => x + y,
       0,
       filter(is_odd, enumerate_interval(0, n)))
```)

constructs two lists: the enumeration and the result of filtering
the enumeration. When the accumulation is complete, these lists are
no longer needed, and the allocated memory can be reclaimed. If we
can arrange to collect all the garbage periodically, and if this turns
out to recycle memory at about the same rate at which we construct new
pairs, we will have preserved the illusion that there is an infinite
amount of memory.

In order to recycle pairs, we must have a way to determine which
allocated pairs are not needed (in the sense that their contents can
no longer influence the future of the computation). The method we
shall examine for accomplishing this is known as #emph[garbage collection]. Garbage collection is based on the observation that, at
any moment in
an interpretation based on list-structured memory,
the only objects that can
affect the future of the computation are those that can be reached by
some succession of
#py("head")
and
#py("tail")
operations starting from the pointers that are currently in the machine
registers.#footnote[We assume here that the stack is represented as a list
as described in section @sec:memory-as-vectors, so that
items on the stack are accessible via the pointer in the stack
register.] Any memory cell that is not so accessible may be
recycled.

There are many ways to perform garbage collection. The method we
shall examine here is called
#idx("stop-and-copy garbage collector")
#idx("garbage collector", sub: "stop-and-copy")
#emph[stop-and-copy]. The basic idea is to divide memory into two
halves: "working memory" and "free memory." When
#py("pair")
constructs pairs, it allocates these in working memory. When working memory
is full, we perform garbage collection by locating all the useful pairs in
working memory and copying these into consecutive locations in free memory.
(The useful pairs are located by tracing all the
#py("head") and
#py("tail")
pointers, starting with the machine registers.) Since we do not copy the
garbage, there will presumably be additional free memory that we can
use to allocate new pairs. In addition, nothing in the working memory
is needed, since all the useful pairs in it have been copied. Thus,
if we interchange the roles of working memory and free memory, we can
continue processing; new pairs will be allocated in the new working
memory (which was the old free memory). When this is full, we can
copy the useful pairs into the new free memory (which was the old
working memory).#footnote[This idea was invented and first implemented
by
#idx("Minsky, Marvin Lee")
Minsky, as part of the implementation of
#idx("Lisp", sub: "on DEC PDP-1")
Lisp for the PDP-1 at the
#idx("MIT", sub: "Research Laboratory of Electronics")
MIT Research Laboratory of Electronics. It was further developed by
#idx("Fenichel, Robert")
#idx("Yochelson, Jerome C.")
Fenichel and Yochelson (1969) for use in the Lisp implementation for the
#idx("Multics time-sharing system")
Multics time-sharing system. Later,
#idx("Baker, Henry G., Jr.")
Baker (1978) developed a "real-time" version of the method,
which does not require the computation to stop during garbage collection.
Baker's idea was extended by
#idx("Hewitt, Carl Eddie")
Hewitt,
#idx("Lieberman, Henry")
Lieberman, and
#idx("Moon, David A.")
Moon (see Lieberman and Hewitt 1983) to take
advantage of the fact that some structure is more volatile
and other structure is more permanent.

An alternative commonly used garbage-collection technique is the
#idx("mark-sweep garbage collector")
#idx("garbage collector", sub: "mark-sweep")
#emph[mark-sweep] method. This consists of tracing all the structure
accessible from the machine registers and marking each pair we reach.
We then scan all of memory, and any location that is unmarked is
"swept up" as garbage and made available for reuse. A full
discussion of the mark-sweep method can be found in
#idx("Allen, John")
Allen 1978.

The Minsky-Fenichel-Yochelson algorithm is the dominant algorithm in
use for large-memory systems because it examines only the useful part
of memory. This is in contrast to mark-sweep, in which the sweep
phase must check all of memory. A second advantage of stop-and-copy
is that it is a
#idx("compacting garbage collector")
#idx("garbage collector", sub: "compacting")
#emph[compacting] garbage collector. That is, at the
end of the garbage-collection phase the useful data will have been
moved to consecutive memory locations, with all garbage pairs
compressed out. This can be an extremely important performance
consideration in machines with virtual memory, in which accesses to
widely separated memory addresses may require extra paging
operations.]

#subheading([Implementation of a stop-and-copy garbage collector])

We now use our register-machine language to describe the stop-and-copy
algorithm in more detail. We will assume that there is a register
called
#idx("root register")
#py("root") that contains a pointer to a structure
that eventually points at all accessible data. This can be arranged by
storing the contents of all the machine registers in a preallocated list
pointed at by #py("root") just before starting
garbage collection.#footnote[This list of
registers does not
include
the registers used by the storage-allocation
system: #py("root"),
#py("the_heads"),
#py("the_tails"),
and the other registers that will be introduced in this section.]
We also assume that, in addition to the current working memory, there is
free memory available into which we can copy the useful data. The current
working memory consists of vectors whose base addresses are in
registers called
#idx("theheads", sub: "register")
#py("the_heads")
and
#idx("thetails", sub: "register")
#py("the_tails"),
and the free memory is in registers called
#idx("newheads register")
#py("new_heads")
and
#idx("newtails register")
#py("new_tails").

Garbage collection is triggered when we exhaust the free cells in the
current working memory, that is, when a
#py("pair")
operation attempts to increment the #py("free")
pointer beyond the end of the memory vector. When the garbage-collection
process is complete, the #py("root") pointer will
point into the new memory, all objects accessible from the
#py("root") will have been moved to the new memory,
and the #py("free") pointer will indicate the next
place in the new memory where a new pair can be allocated. In addition,
the roles of working memory and new memory will have been
interchanged—new pairs will be constructed in the new memory,
beginning at the place indicated by #py("free"), and
the (previous) working memory will be available as the new memory for the
next garbage collection.
Figure @fig:memory-reconfig
shows the arrangement of memory just before and just after garbage
collection.

#sicp-figure(image("/images/img_javascript/Fig5.15c.std.svg", width: 70%), caption: [Reconfiguration of memory by the garbage-collection process.], label-name: <fig:memory-reconfig>)

The state of the garbage-collection process is controlled by
maintaining two pointers:
#idx("free register")
#py("free") and
#idx("scan register")
#py("scan"). These are initialized to point to the
beginning of the new memory. The algorithm begins by relocating the pair
pointed at by #py("root") to the beginning of the new
memory. The pair is copied, the #py("root") pointer
is adjusted to point to the new location, and the
#py("free") pointer is incremented. In addition, the
old location of the pair is marked to show that its contents have been
moved. This marking is done as follows: In the
#py("head")
position, we place a special tag that signals that this is an already-moved
object. (Such an object is traditionally called a
#idx("broken heart")
#emph[broken heart].)#footnote[The term
#emph[broken heart] was coined by
#idx("Cressey, David")
David Cressey, who wrote a garbage collector
for
#idx("MDL")
#idx("Lisp", sub: "MDL dialect of")
MDL, a dialect of Lisp developed at MIT during the early 1970s.]
In the
#py("tail")
position we place a
#idx("forwarding address")
#emph[forwarding address] that points at the location to which the object
has been moved.

After relocating the root, the garbage collector enters its basic
cycle. At each step in the algorithm, the
#py("scan") pointer
(initially pointing at the relocated root) points at a pair that has
been moved to the new memory but whose
#py("head")
and
#py("tail")
pointers still refer to objects in the old memory. These objects are each
relocated, and the #py("scan") pointer is incremented.
To relocate an object (for example, the object indicated by the
#py("head")
pointer of the pair we are scanning) we check to see if the object has
already been moved (as indicated by the presence of a broken-heart tag
in the
#py("head")
position of the object). If the object has not
already been moved, we copy it to the place indicated by
#py("free"),
update #py("free"), set up a broken heart at the
object's old location, and update the pointer to the object (in this
example, the
#py("head")
pointer of the pair we are scanning) to point
to the new location. If the object has already been moved, its
forwarding address (found in the
#py("tail")
position of the broken heart) is substituted for the pointer in the pair
being scanned. Eventually, all accessible objects will have been moved and
scanned, at which point the #py("scan") pointer will
overtake the #py("free") pointer and the process will
terminate.

We can specify the stop-and-copy algorithm as a sequence of instructions for
a register machine. The basic step of relocating an object is accomplished
by a subroutine called
#py("relocate_old_result_in_new").
This subroutine gets its argument, a pointer to the object to be relocated,
from a register named
#idx("old register")
#py("old"). It relocates the designated object
(incrementing #py("free") in the process),
puts a pointer to the relocated object into a register called
#idx("new register")
#py("new"), and returns by branching to the entry
point stored in the register
#py("relocate_continue").
To begin garbage collection, we invoke this subroutine to relocate the
#py("root") pointer, after initializing
#py("free") and #py("scan").
When the relocation of #py("root") has been
accomplished, we install the new pointer as the new
#py("root") and enter the main loop of the garbage
collector.

#snippet(```python
"begin_garbage_collection",
  assign("free", constant(0)),
  assign("scan", constant(0)),
  assign("old", reg("root")),
  assign("relocate_continue", label("reassign_root")),
  go_to(label("relocate_old_result_in_new")),
"reassign_root",
  assign("root", reg("new")),
  go_to(label("gc_loop")),
```)

In the main loop of the garbage collector we must determine whether
there are any more objects to be scanned. We do this by testing
whether the #py("scan") pointer is coincident with
the #py("free") pointer. If the pointers are equal,
then all accessible objects have been relocated, and we branch to
#py("gc_flip"),
which cleans things up so that we can continue the interrupted computation.
If there are still pairs to be scanned, we call the relocate subroutine to
relocate the
#py("head")
of the next pair (by placing the
#py("head")
pointer in #py("old")). The
#py("relocate_continue")
register is set up so that the subroutine will return to update the
#py("head")
pointer.

#snippet(```python
"gc_loop",
  test(list(op("==="), reg("scan"), reg("free"))),
  branch(label("gc_flip")),
  assign("old", list(op("vector_ref"), reg("new_heads"), reg("scan"))),
  assign("relocate_continue", label("update_head")),
  go_to(label("relocate_old_result_in_new")),
```)

At
#py("update_head"),
we modify the
#py("head")
pointer of the pair being scanned, then proceed to relocate the
#py("tail")
of the pair. We return to
#py("update_tail")
when that relocation has been accomplished. After relocating and updating
the
#py("tail"),
we are finished scanning that pair, so we continue with the main loop.

#snippet(```python
"update_head",
  perform(list(op("vector_set"),
               reg("new_heads"), reg("scan"), reg("new"))),
  assign("old", list(op("vector_ref"),
                     reg("new_tails"), reg("scan"))),
  assign("relocate_continue", label("update_tail")),
  go_to(label("relocate_old_result_in_new")),

"update_tail",
  perform(list(op("vector_set"),
               reg("new_tails"), reg("scan"), reg("new"))),
  assign("scan", list(op("+"), reg("scan"), constant(1))),
  go_to(label("gc_loop")),
```)

The subroutine
#py("relocate_old_result_in_new")
relocates objects as follows: If the object to be relocated (pointed at by
#py("old")) is not a pair, then we return the same
pointer to the object unchanged (in #py("new")).
(For example, we may be scanning a pair whose
#py("head")
is the number 4. If we represent the
#py("head")
by #py("n4"), as described in
section @sec:impl-list-ops, then we want the
"relocated"
#py("head")
pointer to still be #py("n4").) Otherwise, we
must perform the relocation. If the
#py("head")
position of the pair to be relocated contains a broken-heart tag, then the
pair has in fact already been moved, so we retrieve the forwarding address
(from the
#py("tail")
position of the broken heart) and return this in
#py("new"). If the pointer in
#py("old") points at a yet-unmoved pair, then we move
the pair to the first free cell in new memory (pointed at by
#py("free")) and set up the broken heart by storing a
broken-heart tag and forwarding address at the old location.
The subroutine #py("relocate_old_result_in_new")
uses a register
#idx("oldht register") #py("oldht")
to hold the
#py("head")
or the
#py("tail")
of the object pointed at by #py("old").#footnote[The
garbage collector uses the low-level predicate
#py("is_pointer_to_pair")
instead of the list-structure
#py("is_pair")
operation because in a real system there might be various things
that are treated as pairs for garbage-collection purposes.
For example,

a
function
object may be implemented as a special kind of
"pair" that doesn't satisfy the
#py("is_pair")
predicate.
For simulation purposes,
#py("is_pointer_to_pair")
can be implemented as
#py("is_pair").]

#syntax("
\"relocate_old_result_in_new\",
  test(list(op(\"is_pointer_to_pair\"), reg(\"old\"))),
  branch(label(\"pair\")),
  assign(\"new\", reg(\"old\")),
  go_to(reg(\"relocate_continue\")),
\"pair\",
  assign(\"oldht\", list(op(\"vector_ref\"),
                       reg(\"the_heads\"), reg(\"old\"))),
  test(list(op(\"is_broken_heart\"), reg(\"oldht\"))),
  branch(label(\"already_moved\")),
  assign(\"new\", reg(\"free\")),     // new location for pair
  // Update ", $mono("free")$, " pointer
  assign(\"free\", list(op(\"+\"), reg(\"free\"), constant(1))),
  // Copy the head and tail to new memory
  perform(list(op(\"vector_set\"),
               reg(\"new_heads\"), reg(\"new\"),
               reg(\"oldht\"))),
  assign(\"oldht\", list(op(\"vector_ref\"),
                       reg(\"the_tails\"), reg(\"old\"))),
  perform(list(op(\"vector_set\"),
               reg(\"new_tails\"), reg(\"new\"),
               reg(\"oldht\"))),
  // Construct the broken heart
  perform(list(op(\"vector_set\"),
               reg(\"the_heads\"), reg(\"old\"),
               constant(\"broken_heart\"))),
  perform(list(op(\"vector_set\"),
               reg(\"the_tails\"), reg(\"old\"),
               reg(\"new\"))),
  go_to(reg(\"relocate_continue\")),
\"already_moved\",
  assign(\"new\", list(op(\"vector_ref\"),
                     reg(\"the_tails\"), reg(\"old\"))),
  go_to(reg(\"relocate_continue\")),
      ")

At the very end of the garbage collection process, we interchange the
role of old and new memories by interchanging pointers: interchanging
#py("the_heads")
with
#py("new_heads"),
and
#py("the_tails")
with
#py("new_tails").
We will then be ready to perform another garbage
collection the next time memory runs out.

#snippet(```python
"gc_flip",
  assign("temp", reg("the_tails")),
  assign("the_tails", reg("new_tails")),
  assign("new_tails", reg("temp")),
  assign("temp", reg("the_heads")),
  assign("the_heads", reg("new_heads")),
  assign("new_heads", reg("temp"))
```)

#idx("list-structured memory")
#idx("memory", sub: "list-structured")
#idx("garbage collection")
#idx("stop-and-copy garbage collector")
#idx("garbage collector", sub: "stop-and-copy")
