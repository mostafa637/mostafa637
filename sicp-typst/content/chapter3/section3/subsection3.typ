// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Representing Tables], label-name: <sec:tables>)

#idx("table")

When we studied various ways of representing sets in chapter @chap:data, we
mentioned in section @sec:representing-sets the task of
maintaining a table of records
#idx("key of a record", sub: "in a table")
indexed by identifying keys. In the
implementation of data-directed programming in
section @sec:data-directed, we made extensive use of
two-dimensional tables, in which information is stored and retrieved
using two keys. Here we see how to build tables as mutable list
structures.

We first consider a
#idx("table", sub: "one-dimensional")
one-dimensional table, in which each value is
stored under a single key. We implement the table as a list of
records, each of which is implemented as a pair consisting of a key
and the associated value. The records are glued together to form a
list by pairs whose
#py("head")s
point to successive records. These gluing pairs are called the
#idx("table", sub: "backbone of")
#emph[backbone] of the table. In order to have a place that we can
change when we add a new record to the table, we build the table as a
#idx("headed list")
#idx("list(s)", sub: "headed")
#emph[headed list]. A headed list has a special backbone pair at the
beginning, which holds a dummy "record"—in this case
the arbitrarily chosen
string #py("\"*table*\"").
Figure @fig:table
shows the box-and-pointer diagram for the table

#snippet(```python
a: 1
b: 2
c: 3
```)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-22.svg", width: 70%), caption: [A table represented as a headed list.], label-name: <fig:table>)

To extract information from a table we use the
#py("lookup")
function,
which takes a key as argument and returns the associated value (or
#py("None")
if
there is no value stored under that key).
The function #py("lookup")
is defined in terms of the #py("assoc") operation,
which expects a key and a list of records as arguments. Note that
#py("assoc") never sees the dummy record.
The function #py("assoc")
returns the record that has the given key as its
#py("head").#footnote[Because #py("assoc") uses
#py("equal"), it can recognize keys that
are strings, numbers, or list structure.]
The function #py("lookup")
then checks to see that the resulting record returned by
#py("assoc") is not
#py("None"),
and returns the value (the
#py("tail"))
of the record.

#idx("lookup", sub: "in one-dimensional table", decl: true)#idx("assoc", decl: true)
#snippet(```python
def lookup(key, table):
    record = assoc(key, tail(table))
    return (None
            if is_none(record)
            else tail(record))

def assoc(key, records):
    return (None
            if is_none(records)
            else head(records)
            if key == head(head(records))
            else assoc(key, tail(records)))
```)

To insert a value in a table under a specified key, we first use
#py("assoc") to see if there is already a record in
the table with this key. If not, we form a new record by
#py("pair")ing
the key with the value, and insert this at the head of the table's
list of records, after the dummy record. If there already is a record with
this key, we set the
#py("tail")
of this record to the designated new value. The header of the table
provides us with a fixed location to modify in order to insert the new
record.#footnote[Thus, the first backbone pair is the object that represents
the table "itself"; that is, a pointer to the table is a
pointer to this pair. This same backbone pair always starts the table.
If we did not arrange things in this way,
#py("insert")
would have to return a new value for the start of the table
when it added a new record.]
#idx("insert", sub: "in one-dimensional table", decl: true)
#snippet(```python
def insert(key, value, table):
    record = assoc(key, tail(table))
    if is_none(record):
        set_tail(table,
                 pair(pair(key, value), tail(table)))
    else:
        set_tail(record, value)
    return "ok"
```)

To construct a new table, we simply create a list containing
just the string #py("\"*table*\""):
#idx("maketable", sub: "one-dimensional table", decl: true)
#snippet(```python
def make_table():
    return llist("*table*")
```)

#idx("table", sub: "one-dimensional")

#subheading([Two-dimensional tables])

#idx("table", sub: "two-dimensional")

In a two-dimensional table, each value is indexed by two keys. We can
construct such a table as a one-dimensional table in which each key
identifies a subtable.
Figure @fig:2dtable
shows the box-and-pointer diagram for the table

#snippet(```python
"math":
    "+":  43
    "-":  45
    "*":  42
"letters":
    "a":  97
    "b":  98
```)

which has two subtables. (The subtables don't need a special header
string,
since the key that identifies the subtable serves this purpose.)

#sicp-figure(image("/images/img_javascript/ch3-Z-G-23.svg", width: 70%), caption: [A two-dimensional table.], label-name: <fig:2dtable>)

When we look up an item, we use the first key to identify the correct
subtable. Then we use the second key to identify the record within the
subtable.

#idx("lookup", sub: "in two-dimensional table", decl: true)
#snippet(```python
def lookup(key_1, key_2, table):
    subtable = assoc(key_1, tail(table))
    if is_none(subtable):
        return None
    else:
        record = assoc(key_2, tail(subtable))
        return (None
                if is_none(record)
                else tail(record))
```)

To insert a new item under a pair of keys, we use
#py("assoc") to see if there is a subtable stored
under the first key. If not, we build a new subtable containing the single
record
(#py("key_2"),
#py("value")) and insert it into the table under the
first key. If a subtable already exists for the first key, we insert the
new record into this subtable, using the insertion method for
one-dimensional tables described above:
#idx("insert", sub: "in two-dimensional table", decl: true)
#snippet(```python
def insert(key_1, key_2, value, table):
    subtable = assoc(key_1, tail(table))
    if is_none(subtable):
        set_tail(table,
                 pair(llist(key_1, pair(key_2, value)), tail(table)))
    else:
        record = assoc(key_2, tail(subtable))
        if is_none(record):
            set_tail(subtable,
                     pair(pair(key_2, value), tail(subtable)))
        else:
            set_tail(record, value)
    return "ok"
```)

#idx("table", sub: "two-dimensional")

#subheading([Creating local tables])

#idx("table", sub: "local")

The #py("lookup") and
#py("insert")
operations defined above take the table as an argument. This enables us to
use programs that access more than one table. Another way to deal with
multiple tables is to have separate #py("lookup") and
#py("insert")
functions
for each table. We can do this by representing a table procedurally, as an
object that maintains an internal table as part of its local state. When
sent an appropriate message, this "table object" supplies the
function
with which to operate on the internal table. Here is a generator for
two-dimensional tables represented in this fashion:
#idx("maketable", sub: "message-passing implementation", decl: true)
#snippet(```python
def make_table():
    local_table = llist("*table*")
    def lookup(key_1, key_2):
        subtable = assoc(key_1, tail(local_table))
        if is_none(subtable):
            return None
        else:
            record = assoc(key_2, tail(subtable))
            return (None
                    if is_none(record)
                    else tail(record))
    def insert(key_1, key_2, value):
        subtable = assoc(key_1, tail(local_table))
        if is_none(subtable):
            set_tail(local_table,
                     pair(llist(key_1, pair(key_2, value)),
                          tail(local_table)))
        else:
            record = assoc(key_2, tail(subtable))
            if is_none(record):
                set_tail(subtable,
                         pair(pair(key_2, value), tail(subtable)))
            else:
                set_tail(record, value)
    def dispatch(m):
        return (lookup if m == "lookup"
                else insert if m == "insert"
                else error("unknown operation -- table", m))
    return dispatch
```)

Using
#py("make_table"),
we could
#idx("operation-and-type table", sub: "implementing")
implement the #py("get") and
#py("put") operations used in
section @sec:data-directed for data-directed
programming, as follows:

#idx("get", decl: true)#idx("put", decl: true)
#snippet(```python
operation_table = make_table()
get = operation_table("lookup")
put = operation_table("insert")
```)

The function #py("get")
takes as arguments two keys, and #py("put") takes
as arguments two keys and a value. Both operations access the same
local table, which is encapsulated within the object created by the
call to
#py("make_table").
#idx("table", sub: "local")

#exercise(label-name: <ex:numeric-keys>, [
In the table implementations above, the keys are
#idx("table", sub: "testing equality of keys")
#idx("key of a record", sub: "testing equality of")
tested for equality using
#py("equal")
(called by #py("assoc")). This is not always the
appropriate test. For instance, we might have a table with numeric keys in
which we don't need an exact match to the number we're looking
up, but only a number within some tolerance of it. Design a table
constructor
#py("make_table")
that takes as an argument a
#py("same_key")
function
that will be used to test "equality" of keys.
The function #py("make_table")
should return a #py("dispatch")
function
that can be used to access appropriate
#py("lookup") and
#py("insert")
functions
for a local table.
])

#exercise(label-name: <ex:3_25>, [
Generalizing one- and two-dimensional tables, show how to implement a
table in which values are stored under an
#idx("table", sub: "n-dimensional")
arbitrary number of keys and
different values may be stored under different numbers of keys.
The
#py("lookup") and
#py("insert")
functions
should take as input a list of keys used to access the table.
])

#exercise(label-name: <ex:3_26>, [
To search a table as implemented above, one needs to scan through the
list of records. This is basically the unordered list representation of
section @sec:representing-sets. For large tables, it
may be more efficient to structure the table in a different manner.
Describe a table implementation where the (key, value) records are organized
using a
#idx("binary tree", sub: "table structured as")
#idx("table", sub: "represented as binary tree vs. unordered list")
binary tree, assuming that keys can be ordered in some way
(e.g., numerically or alphabetically). (Compare
exercise @ex:set-lookup-binary-tree of chapter @chap:data.)
])

#exercise(label-name: <ex:memoization>, [
#emph[Memoization]
#idx("memoization")
#idx("tabulation")
#idx("table", sub: "used to store computed values")

(also called #emph[tabulation]) is a technique that
enables a
function
to record, in a local table, values that have previously been computed.
This technique can make a vast difference in the performance of a program.
A memoized
function
maintains a table in which values of previous calls are stored
using as keys the arguments that produced the values. When the
memoized
function
is asked to compute a value, it first checks the table to see if the value
is already there and, if so, just returns that value. Otherwise, it
computes the new value in the ordinary way and stores this in the table.
As an example of memoization, recall from
section @sec:tree-recursion the exponential process for
computing Fibonacci numbers:

#snippet(```python
def fib(n):
    return (0
            if n == 0
            else 1
            if n == 1
            else fib(n - 1) + fib(n - 2))
```)

The memoized version of the same
function
is

#idx("fib", sub: "with memoization", decl: true)#idx("memofib", decl: true)
#snippet(```python
memo_fib = memoize(lambda n: (0
                              if n == 0
                              else 1
                              if n == 1
                              else memo_fib(n - 1) +
                                   memo_fib(n - 2)))
```)

where the memoizer is defined as
#idx("memoize", decl: true)
#snippet(```python
def memoize(f):
    table = make_table()
    def memoized(x):
        previously_computed_result = lookup(x, table)
        if is_none(previously_computed_result):
            result = f(x)
            insert(x, result, table)
            return result
        else:
            return previously_computed_result
    return memoized
```)

Draw an environment diagram to analyze the computation of
#py("memo_fib(3)").
Explain why
#py("memo_fib")
computes the $n$th Fibonacci number in a number
of steps proportional to $n$. Would the scheme
still work if we had simply defined
#py("memo_fib")
to be
#py("memoize(fib)")?
])

#idx("table")
