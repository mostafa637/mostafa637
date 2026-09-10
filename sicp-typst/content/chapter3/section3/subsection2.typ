// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Representing Queues], label-name: <sec:queues>)

#idx("queue")

The mutators
#py("set_head")
and
#py("set_tail")
enable us to use pairs to construct data structures that cannot be built
with
#py("pair"),
#py("head"),
and
#py("tail")
alone. This section shows how to use pairs to represent a data structure
called a queue. Section @sec:tables will show how to
represent data structures called tables.

A #emph[queue] is a sequence in which items are inserted at one end
(called the
#idx("queue", sub: "rear of")
#emph[rear] of the queue) and deleted from the other end (the
#idx("queue", sub: "front of")
#emph[front]).
Figure @fig:queue-ops
shows an initially empty queue in which the items
#py("a") and #py("b") are
inserted. Then #py("a") is removed,
#py("c") and #py("d") are
inserted, and #py("b") is removed. Because items are
always removed in the order in which they are inserted, a queue is
sometimes called a
#idx("FIFO buffer")
#emph[FIFO] (first in, first out) buffer.

#sicp-figure([#sicp-table(columns: 2, [Operation], [Resulting Queue], [#py("q = make_queue()")], [], [#py("insert_queue(q, \"a\")")], [#py("a")], [#py("insert_queue(q, \"b\")")], [#py("a b")], [#py("delete_queue(q)")], [#py("b")], [#py("insert_queue(q, \"c\")")], [#py("b c")], [#py("insert_queue(q, \"d\")")], [#py("b c d")], [#py("delete_queue(q)")], [#py("c d")])], caption: [Queue operations.], label-name: <fig:queue-ops>)

In terms of
#idx("data abstraction", sub: "for queue")
#idx("queue", sub: "operations on")
data abstraction, we can regard a queue as defined by the
following set of operations:

- a constructor: \ #idx("makequeue") #py("make_queue()") \ returns an empty queue (a queue containing no items).
- a predicate: \ #idx("isemptyqueue") #py("is_empty_queue(")#meta("queue")#py(")") \ tests if the queue is empty.
- a selector: \ #idx("frontqueue") #py("front_queue(")#meta("queue")#py(")") \ returns the object at the front of the queue, signaling an error if the queue is empty; it does not modify the queue.
- two mutators: \ #py("insert_queue(")#meta("queue")#py(",")#meta("item")#py(")") \ inserts #idx("insertqueue") the item at the rear of the queue and returns the modified queue as its value. #py("delete_queue(")#meta("queue")#py(")") \ removes #idx("deletequeue") the item at the front of the queue and returns the modified queue as its value, signaling an error if the queue is empty before the deletion.

Because a queue is a sequence of items, we could certainly represent
it as an ordinary list; the front of the queue would be the
#py("head")
of the list, inserting an item in the queue would amount to appending
a new element at the end of the list, and deleting an item from the
queue would just be taking the
#py("tail")
of the list. However, this representation is inefficient, because in order
to insert an item we must scan the list until we reach the end. Since the
only method we have for scanning a list is by successive
#py("tail")
operations, this scanning requires $Theta (n)$
steps for a list of $n$ items. A simple
modification to the list representation overcomes this disadvantage by
allowing the queue operations to be implemented so that they require
$Theta (1)$ steps; that is, so that the number
of steps needed is independent of the length of the queue.

The difficulty with the list representation arises from the need to
scan to find the end of the list. The reason we need to scan is that,
although the standard way of representing a list as a chain of pairs
readily provides us with a pointer to the beginning of the list, it
gives us no easily accessible pointer to the end. The modification
that avoids the drawback is to represent the queue as a list, together
with an additional pointer that indicates the final pair in the list.
That way, when we go to insert an item, we can consult the rear
pointer and so avoid scanning the list.

A queue is represented, then, as a pair of pointers,
#py("front_ptr")
and
#py("rear_ptr"),
which indicate, respectively, the first and last pairs in an ordinary list.
Since we would like the queue to be an identifiable object, we can use
#py("pair")
to combine the two pointers. Thus, the queue itself will be the
#py("pair")
of the two pointers.
Figure @fig:queue-pointers
illustrates this representation.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-19.svg", width: 70%), caption: [Implementation of a queue as a list with front and rear pointers.], label-name: <fig:queue-pointers>)

To define the queue operations we use the following
functions,
which enable us to select and to modify the front and rear pointers of a
queue:

#idx("frontptr", decl: true)#idx("rearptr", decl: true)#idx("setfrontptr", decl: true)#idx("setrearptr", decl: true)
#snippet(```python
def front_ptr(queue):
    return head(queue)

def rear_ptr(queue):
    return tail(queue)

def set_front_ptr(queue, item):
    set_head(queue, item)

def set_rear_ptr(queue, item):
    set_tail(queue, item)
```)

Now we can implement the actual queue operations. We will consider a
queue to be empty if its front pointer is the empty list:

#idx("isemptyqueue", decl: true)
#snippet(```python
def is_empty_queue(queue):
    return is_none(front_ptr(queue))
```)

The
#py("make_queue")
constructor returns, as an initially empty queue, a pair whose
#py("head")
and
#py("tail")
are both the empty list:

#idx("makequeue", decl: true)
#snippet(```python
def make_queue():
    return pair(None, None)
```)

To select the item at the front of the queue, we return the
#py("head")
of the pair indicated by the front pointer:

#idx("frontqueue", decl: true)
#snippet(```python
def front_queue(queue):
    return (error("front_queue called with an empty queue", queue)
            if is_empty_queue(queue)
            else head(front_ptr(queue)))
```)

To insert an item in a queue, we follow the method whose result is
indicated in
figure @fig:queue-insert.
We first create a new
pair whose
#py("head")
is the item to be inserted and whose
#py("tail")
is the empty list. If the queue was initially empty, we set the front and
rear pointers of the queue to this new pair. Otherwise, we modify the
final pair in the queue to point to the new pair, and also set the
rear pointer to the new pair.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-20.svg", width: 70%), caption: [Result of using #py("insert_queue(q, \"d\")") on the queue of figure @fig:queue-pointers.], label-name: <fig:queue-insert>)

#idx("insertqueue", decl: true)
#snippet(```python
def insert_queue(queue, item):
    new_pair = pair(item, None)
    if is_empty_queue(queue):
        set_front_ptr(queue, new_pair)
        set_rear_ptr(queue, new_pair)
    else:
        set_tail(rear_ptr(queue), new_pair)
        set_rear_ptr(queue, new_pair)
    return queue
```)

To delete the item at the front of the queue, we merely modify the
front pointer so that it now points at the second item in the queue,
which can be found by following the
#py("tail")
pointer of the first item (see
figure @fig:queue-delete):#footnote[If the first item is
the final item in the queue, the front pointer will be the empty list after
the deletion, which will mark the queue as empty; we needn't worry
about updating the rear pointer, which will still point to the deleted
item, because
#py("is_empty_queue")
looks only at the front pointer.]

#sicp-figure(image("/images/img_javascript/ch3-Z-G-21.svg", width: 70%), caption: [Result of using #py("delete_queue(q)") on the queue of figure @fig:queue-insert.], label-name: <fig:queue-delete>)

#idx("deletequeue", decl: true)
#snippet(```python
def delete_queue(queue):
    if is_empty_queue(queue):
        error("delete_queue called with an empty queue", queue)
    else:
        set_front_ptr(queue, tail(front_ptr(queue)))
        return queue
```)

#exercise(label-name: <ex:3_21>, [
Ben Bitdiddle decides to test the queue implementation described
above. He types in the
functions
to the
Python
interpreter and proceeds to try them out:

#snippet(```python
q1 = make_queue()
```)

#snippet(```python
print(insert_queue(q1, "a"))
```)

#output(```python
print(insert_queue(q1, "a"))
```)

#snippet(```python
print(insert_queue(q1, "b"))
```)

#output(```python
print(insert_queue(q1, "b"))
```)

#snippet(```python
print(delete_queue(q1))
```)

#output(```python
print(delete_queue(q1))
```)

#snippet(```python
print(delete_queue(q1))
```)

#output(```python
print(delete_queue(q1))
```)

"It's all wrong!" he complains.
"The interpreter's response shows that the last item is inserted into the queue twice. And when I delete both items, the second #py("b") is still there, so the queue isn't empty, even though it's supposed to be." Eva Lu Ator suggests
that Ben has misunderstood what is happening. "It's not that the items are going into the queue twice," she explains.
"It's just that the standard Python printer doesn't know how to make sense of the queue representation. If you want to see the queue printed correctly, you'll have to define your own print function for queues." Explain what Eva Lu is talking about. In particular,
show why Ben's examples produce the printed results that they do.
Define a
function
#idx("printqueue")
#py("print_queue")
that takes a queue as input and prints the sequence of items in the queue.
])

#exercise(label-name: <ex:3_22>, [
Instead of representing a queue as a pair of pointers, we can build a
queue as a
function
#idx("queue", sub: "functional implementation of")
with local state. The local state will consist of pointers to the
beginning and the end of an ordinary list. Thus, the
#py("make_queue")
function
will have the form

#syntax("
def make_queue():
    front_ptr = ", $dots.h$, "
    rear_ptr = ", $dots.h$, "
    ", metaphrase[declarations of internal functions], "
    def dispatch(m): ", $dots.h$, "
    return dispatch
      ")

Complete the definition of
#py("make_queue")
and provide implementations of the queue operations using this
representation.
])

#exercise(label-name: <ex:deque>, [
A #emph[deque]
#idx("queue", sub: "double-ended")
#idx("deque")
("double-ended queue") is a sequence in which
items can be inserted and deleted
either at the front or at
the rear.
Operations on deques are the constructor
#py("make_deque"),
the predicate
#py("is_empty_deque"),
selectors
#py("front_deque")
and
#py("rear_deque"),
and mutators
#py("front_insert_deque"),
#py("front_delete_deque"),
#py("rear_insert_deque"),
and
#py("rear_delete_deque").
Show how to represent deques using pairs, and give implementations of the
operations.#footnote[Be careful not to make the interpreter try to print a
structure that contains cycles. (See
exercise @ex:make-cycle.)]
All operations should be accomplished in
$Theta (1)$ steps.
])

#idx("queue")
