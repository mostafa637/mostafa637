// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Data-Directed Programming and Additivity], label-name: <sec:data-directed>)

#idx("data-directed programming")
#idx("additivity")

The general strategy of checking the type of a datum and calling an
appropriate
function
is called
#idx("modularity", sub: "through dispatching on type")
#idx("dispatching", sub: "on type")
#idx("type(s)", sub: "dispatching on")
#emph[dispatching on type]. This is a powerful strategy for obtaining
modularity in system design. On the other hand, implementing the dispatch
as in section @sec:manifest-types has two significant
weaknesses. One weakness is that the generic interface
functions
(#py("real_part"),
#py("imag_part"),
#py("magnitude"), and
#py("angle")) must know about all the different
representations. For instance, suppose we wanted to incorporate a new
representation for complex numbers into our complex-number system. We
would need to identify this new representation with a type, and then add a
clause to each of the generic interface
functions
to check for the new type and apply the appropriate selector for that
representation.

Another weakness of the technique is that even though the individual
representations can be designed separately, we must guarantee that no two
functions
in the entire system have the same name. This is why Ben and Alyssa had
to change the names of their original
functions
from section @sec:representations-complex-numbers.

The issue underlying both of these weaknesses is that the technique for
implementing generic interfaces is not #emph[additive]. The person
implementing the generic selector
functions
must modify those
functions
each time a new representation is installed, and the people
interfacing the individual representations must modify their
code to avoid name conflicts. In each of these cases, the changes
that must be made to the code are straightforward, but they must be
made nonetheless, and this is a source of inconvenience and error.
This is not much of a problem for the complex-number system as it
stands, but suppose there were not two but hundreds of different
representations for complex numbers. And suppose that there were many
generic selectors to be maintained in the abstract-data interface.
Suppose, in fact, that no one programmer knew all the interface
functions
or all the representations. The problem is real and must
be addressed in such programs as
large-scale data-base-management systems.

What we need is a means for modularizing the system design even
further. This is provided by the programming technique known as #emph[data-directed programming]. To understand how data-directed
programming works, begin with the observation that whenever we deal
with a set of generic operations that are common to a set of
different types we are, in effect, dealing with a two-dimensional
table that contains the possible operations on one axis and the
possible types on the other axis. The entries in the table are the
functions
that implement each operation for each type of argument presented.
In the complex-number system developed in the previous section, the
correspondence between operation name, data type, and actual
function
was spread out among the various conditional clauses in the generic
interface
functions.
But the same information could have been organized in a table, as shown in
figure @fig:operator-table.

Data-directed programming is the technique of designing programs to work
with such a
#idx("table", sub: "for data-directed programming")
table directly. Previously, we implemented the mechanism that
interfaces the complex-arithmetic code with the two representation packages
as a set of
functions
that each perform an explicit dispatch on type. Here we will implement the
interface as a single
function
that looks up the combination of the operation name and argument type in
the table to find the correct
function
to apply, and then applies it to the contents of the argument. If we do
this, then to add a new representation package to the system we need not
change any existing
functions;
we need only add new entries to the table.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-63.svg", width: 59%), caption: [Table of operations for the complex-number system.], label-name: <fig:operator-table>)

To implement this plan, assume that we have two
functions,
#py("put") and #py("get"), for
manipulating the
#idx("operation-and-type table")
operation-and-type table:

- #idx("put") #py("put(")#meta("op")#py(",")#meta("type")#py(",")#meta("item")#py(")") \ installs the #meta("item") in the table, indexed by the #meta("op") and the #meta("type").
- #idx("get") #py("get(")#meta("op")#py(",")#meta("type")#py(")") \ looks up the #meta("op"), #meta("type") entry in the table and returns the item found there. If no item is found, #py("get") returns a unique primitive value that is referred to by the keyword #idx("None (keyword)") #py("None") and recognized by the primitive predicate #idx("isnone (primitive function)") #py("is_none").

For now, we can assume that #py("put") and
#py("get") are included in our language. In
chapter @chap:state (section @sec:tables) we will see
how to implement these and other operations for manipulating tables.

Here is how data-directed programming can be used in the complex-number
system. Ben, who developed the rectangular representation, implements his
code just as he did originally. He defines a collection of
functions
or a
#idx("package")
#idx("package", sub: "rectangular representation")
#idx("rectangular package")
#emph[package], and interfaces these to the rest of the system by adding
entries to the table that tell the system how to operate on rectangular
numbers. This is accomplished by calling the following
function:

#idx("installrectangularpackage", decl: true)
#snippet(```python
def install_rectangular_package():
    # internal functions
    def real_part(z): return head(z)
    def imag_part(z): return tail(z)
    def make_from_real_imag(x, y): return pair(x, y)
    def magnitude(z):
        return math_sqrt(square(real_part(z)) + square(imag_part(z)))
    def angle(z):
        return math_atan2(imag_part(z), real_part(z))
    def make_from_mag_ang(r, a):
        return pair(r * math_cos(a), r * math_sin(a))

    # interface to the rest of the system
    def tag(x): return attach_tag("rectangular", x)
    put("real_part", llist("rectangular"), real_part)
    put("imag_part", llist("rectangular"), imag_part)
    put("magnitude", llist("rectangular"), magnitude)
    put("angle", llist("rectangular"), angle)
    put("make_from_real_imag", "rectangular",
        lambda x, y: tag(make_from_real_imag(x, y)))
    put("make_from_mag_ang", "rectangular",
        lambda r, a: tag(make_from_mag_ang(r, a)))
    return "done"
```)

Notice that the internal
functions
here are the same
functions
from section @sec:representations-complex-numbers that
Ben wrote when he was working in isolation. No changes are necessary in
order to interface them to the rest of the system. Moreover, since these
function definitions
are internal to the installation
function,
Ben needn't worry about name conflicts with other
functions
outside the rectangular package. To interface these to the rest of the
system, Ben installs his
#py("real_part")
function
under the operation name
#py("real_part")
and the type
#py("llist(\"rectangular\")"),
and similarly for the other selectors.#footnote[We use the
linked list
#py("llist(\"rectangular\")")
rather than the
string #py("\"rectangular\"")
to allow for the possibility of operations with multiple arguments, not
all of the same type.] The interface also defines the
constructors to be used by the external system.#footnote[The type the
constructors are installed under needn't be a
linked list
because a
constructor is always used to make an object of one particular
type.] These are identical to Ben's internally defined
constructors, except that they attach the tag.

Alyssa's
#idx("package", sub: "polar representation")
#idx("polar package")
polar package is analogous:
#idx("installpolarpackage", decl: true)
#snippet(```python
def install_polar_package():
    # internal functions
    def magnitude(z): return head(z)
    def angle(z): return tail(z)
    def make_from_mag_ang(r, a): return pair(r, a)
    def real_part(z):
        return magnitude(z) * math_cos(angle(z))
    def imag_part(z):
        return magnitude(z) * math_sin(angle(z))
    def make_from_real_imag(x, y):
        return pair(math_sqrt(square(x) + square(y)),
                    math_atan2(y, x))

    # interface to the rest of the system
    def tag(x): return attach_tag("polar", x)
    put("real_part", llist("polar"), real_part)
    put("imag_part", llist("polar"), imag_part)
    put("magnitude", llist("polar"), magnitude)
    put("angle", llist("polar"), angle)
    put("make_from_real_imag", "polar",
        lambda x, y: tag(make_from_real_imag(x, y)))
    put("make_from_mag_ang", "polar",
        lambda r, a: tag(make_from_mag_ang(r, a)))
    return "done"
```)

Even though Ben and Alyssa both still use their original
functions
defined with the same names as each other's (e.g.,
#py("real_part")),
these declarations are now internal to different
functions
(see section @sec:block-structure), so there is no name
conflict.

The complex-arithmetic selectors access the table by means of a general
"operation"
function
called
#py("apply_generic"),
which applies a generic operation to some arguments.
The function #py("apply_generic")
looks in the table under the name of the operation and the types of the
arguments and applies the resulting
function
if one is present:#footnote[The function
#py("apply_generic")
uses the function
#idx("applyinunderlyingjavascript")
#py("apply_in_underlying_javascript")
given in section @sec:running-eval

(footnote @foot:vector-array),

which takes two arguments, a function and a linked list, and
applies the function, using the elements in the linked list as arguments.

For example,

#snippet(```python
apply_in_underlying_javascript(sum_of_squares, llist(1, 3))
```)

returns 10.]

#idx("applygeneric", decl: true)
#snippet(```python
def apply_generic(op, args):
    type_tags = map(type_tag, args)
    fun = get(op, type_tags)
    return (apply_in_underlying_javascript(fun, map(contents, args))
            if not is_none(fun)
            else error("no method for these types -- apply_generic",
                       llist(op, type_tags)))
```)

Using
#py("apply_generic"),
we can define our generic selectors as follows:
#idx("realpart", sub: "data-directed", decl: true)#idx("imagpart", sub: "data-directed", decl: true)#idx("magnitude", sub: "data-directed", decl: true)#idx("angle", sub: "data-directed", decl: true)
#snippet(```python
def real_part(z): return apply_generic("real_part", llist(z))

def imag_part(z): return apply_generic("imag_part", llist(z))

def magnitude(z): return apply_generic("magnitude", llist(z))

def angle(z): return apply_generic("angle", llist(z))
```)

Observe that these do not change at all if a new representation is
added to the system.

We can also extract from the table the constructors to be used by the
programs external to the packages in making complex numbers from real and
imaginary parts and from magnitudes and angles. As in
section @sec:manifest-types, we construct rectangular
numbers whenever we have real and imaginary parts, and polar numbers
whenever we have magnitudes and angles:
#idx("makefromrealimag", decl: true)#idx("makefrommagang", decl: true)
#snippet(```python
def make_from_real_imag(x, y):
    return get("make_from_real_imag", "rectangular")(x, y)
def make_from_mag_ang(r, a):
    return get("make_from_mag_ang", "polar")(r, a)
```)

#exercise(label-name: <ex:data-directed-differentiation>, [
Section @sec:symbolic-differentiation described a
program that performs
#idx("symbolic differentiation")
#idx("differentiation", sub: "symbolic")
symbolic differentiation:

#snippet(```python
def deriv(exp, variable):
    return (0
            if is_number(exp)
            else (1 if is_same_variable(exp, variable) else 0)
            if is_variable(exp)
            else make_sum(deriv(addend(exp), variable),
                          deriv(augend(exp), variable))
            if is_sum(exp)
            else make_sum(make_product(multiplier(exp),
                                       deriv(multiplicand(exp), variable)),
                          make_product(deriv(multiplier(exp), variable),
                                       multiplicand(exp)))
            if is_product(exp)
            # more rules can be added here
            else error("unknown expression type -- deriv", exp))
```)

#snippet(```python
print(deriv(llist("*", llist("*", "x", "y"), llist("+", "x", 4)), "x"))
```)

#output(```python
print(deriv(llist("*", llist("*", "x", "y"), llist("+", "x", 4)), "x"))
```)

We can regard this program as performing a dispatch on the type of the
expression to be differentiated. In this situation the
"type tag" of the datum is the algebraic operator symbol
(such as "+")
and the operation being performed is
#py("deriv"). We can transform this program into
data-directed style by rewriting the basic derivative
function
as
#idx("deriv (symbolic)", sub: "data-directed", decl: true)
#snippet(```python
def deriv(exp, variable):
    return (0
            if is_number(exp)
            else (1 if is_same_variable(exp, variable) else 0)
            if is_variable(exp)
            else get("deriv", operator(exp))(operands(exp), variable))
def operator(exp): return head(exp)

def operands(exp): return tail(exp)
```)

+ Explain what was done above. Why can't we assimilate the predicates #py("is_number") and #py("is_variable") into the data-directed dispatch?
+ Write the functions for derivatives of sums and products, and the auxiliary code required to install them in the table used by the program above.
+ Choose any additional differentiation rule that you like, such as the one for exponents (exercise @ex:deriv-exponentiation), and install it in this data-directed system.
+ In this simple algebraic manipulator the type of an expression is the algebraic operator that binds it together. Suppose, however, we indexed the functions in the opposite way, so that the dispatch line in #py("deriv") looked like #snippet(```python get(operator(exp), "deriv")(operands(exp), variable) ```) What corresponding changes to the derivative system are required?
])

#exercise(label-name: <ex:2_74>, [
Insatiable
#idx("data base", sub: "Insatiable Enterprises personnel")
Enterprises, Inc., is a highly decentralized conglomerate company
consisting of a large number of independent divisions located all over the
world. The company's computer facilities have just been
interconnected by means of a clever network-interfacing scheme that makes
the entire network appear to any user to be a single computer.
Insatiable's president, in her first attempt to exploit the ability
of the network to extract administrative information from division files,
is dismayed to discover that, although all the division files have been
implemented as data structures in
Python,
the particular data structure used varies from division to division. A
meeting of division managers is hastily called to search for a strategy to
integrate the files that will satisfy headquarters' needs while
preserving the existing autonomy of the divisions.

Show how such a strategy can be implemented with
#idx("data base", sub: "data-directed programming and")
data-directed programming.
As an example, suppose that each division's personnel records consist
of a single file, which contains a set of records keyed on
employees' names. The structure of the set varies from division to
division. Furthermore, each employee's record is itself a set
(structured differently from division to division) that contains
information keyed under identifiers such as
#py("address") and
#py("salary"). In particular:

+ Implement for headquarters a #py("get_record") function that retrieves a specified employee's record from a specified personnel file. The function should be applicable to any division's file. Explain how the individual divisions' files should be structured. In particular, what type information must be supplied?
+ Implement for headquarters a #py("get_salary") function that returns the salary information from a given employee's record from any division's personnel file. How should the record be structured in order to make this operation work?
+ Implement for headquarters a #py("find_employee_record") function. This should search all the divisions' files for the record of a given employee and return the record. Assume that this function takes as arguments an employee's name and a linked list of all the divisions' files.
+ When Insatiable takes over a new company, what changes must be made in order to incorporate the new personnel information into the central system?
])

#idx("data-directed programming")
#idx("additivity")

#subheading([Message passing])

#idx("message passing")

The key idea of data-directed programming is to handle generic operations
in programs by dealing explicitly with operation-and-type tables, such as
the table in
figure @fig:operator-table.
The style of programming we used in
section @sec:manifest-types organized the required
dispatching on type by having each operation take care of its own
dispatching. In effect, this decomposes the operation-and-type table into
rows, with each generic operation
function
representing a row of the table.

An alternative implementation strategy is to decompose the table into
columns and, instead of using "intelligent operations" that
dispatch on data types, to work with "intelligent data objects" that dispatch on operation names. We can do this by
arranging things so that a data object, such as a rectangular number, is
represented as a
function
that takes as input the required operation name and performs the operation
indicated. In such a discipline,
#py("make_from_real_imag")
could be written as
#idx("makefromrealimag", sub: "message-passing", decl: true)
#snippet(```python
def make_from_real_imag(x, y):
    def dispatch(op):
        return (x
                if op == "real_part"
                else y
                if op == "imag_part"
                else math_sqrt(square(x) + square(y))
                if op == "magnitude"
                else math_atan2(y, x)
                if op == "angle"
                else error("unknown op -- make_from_real_imag", op))
    return dispatch
```)

The corresponding
#py("apply_generic")
function,
which applies a generic operation to an argument, now simply feeds the
operation's name to the data object and lets the object do the
work:#footnote[One limitation of this organization is it permits only
generic
functions
of one argument.]
#idx("applygeneric", sub: "with message passing", decl: true)
#snippet(```python
def apply_generic(op, arg): return head(arg)(op)
```)

Note that the value returned by
#py("make_from_real_imag")
is a
function—the internal
#py("dispatch")
function.
This is the
function
that is invoked when
#py("apply_generic")
requests an operation to be performed.

This style of programming is called #emph[message passing]. The name
comes from the image that a data object is an entity that receives the
requested operation name as a "message." We have already seen
an example of message passing in section @sec:data-,
where we saw how
#py("pair"),
#py("head"),
and
#py("tail")
could be defined with no data objects but only
functions.
Here we see that message passing is not a mathematical trick but a useful
technique for organizing systems with generic operations. In the remainder
of this chapter we will continue to use data-directed programming, rather
than message passing, to discuss generic arithmetic operations. In
chapter @chap:state we will return to message passing, and we will see that
it can be a powerful tool for structuring simulation programs.

#exercise(label-name: <ex:2_75>, [
Implement the constructor
#idx("makefrommagang", sub: "message-passing")
#py("make_from_mag_ang")
in message-passing style. This
function
should be analogous to the
#py("make_from_real_imag")
function
given above.
])

#exercise(label-name: <ex:extend-generic>, [
As a large system with generic operations evolves, new types of data
objects or new operations may be needed. For each of the three
strategies—generic operations with explicit
#idx("dispatching", sub: "comparing different styles")
dispatch, data-directed
style, and message-passing-style—describe the changes that must be
made to a system in order to add new types or new operations. Which
organization would be most appropriate for a system in which new types must
often be added? Which would be most appropriate for a system in which new
operations must often be added?
])
