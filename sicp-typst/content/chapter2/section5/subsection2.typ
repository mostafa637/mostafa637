// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Combining Data of Different Types], label-name: <sec:combining-data-of-different-types>)

We have seen how to define a unified arithmetic system that
encompasses ordinary numbers, complex numbers, rational numbers, and
any other type of number we might decide to invent, but we have
ignored an important issue. The operations we have defined so far
treat the different data types as being completely independent. Thus,
there are separate packages for adding, say, two ordinary numbers, or
two complex numbers. What we have not yet considered is the fact that
it is meaningful to define operations that cross the type boundaries,
such as the addition of a complex number to an ordinary number. We
have gone to great pains to introduce barriers between parts of our
programs so that they can be developed and understood separately. We
would like to introduce the cross-type operations in some carefully
controlled way, so that we can support them
without seriously violating our module boundaries.

One way to handle
#idx("operation", sub: "cross-type")
#idx("cross-type operations")
#idx("type(s)", sub: "cross-type operations")
cross-type operations is to design a different
function
for each possible combination of types for which the operation is valid.
For example, we could extend the complex-number package so that it
provides a
function
for adding complex numbers to ordinary numbers and installs this in the
table using the tag
#py("llist(\"complex\", \"python_number\")"):#footnote[We
also have to supply an almost identical
function
to handle the types\

#py("llist(\"python_number\", \"complex\")").]

#idx("addcomplextopythonnum", decl: true)
#snippet(```python
# to be included in the complex package
def add_complex_to_python_num(z, x):
    return make_complex_from_real_imag(real_part(z) + x, imag_part(z))
print(put("add", llist("complex", "python_number"),
    lambda z, x: tag(add_complex_to_python_num(z, x))))
```)

This technique works, but it is cumbersome. With such a system, the
cost of introducing a new type is not just the construction of the
package of
functions
for that type but also the construction and installation of the
functions
that implement the cross-type operations. This can easily be much more
code than is needed to define the operations on the type itself. The
method also undermines our ability to combine separate packages additively,
or least to limit the extent to which the implementors of the individual
packages need to take account of other packages. For instance, in the
example above, it seems reasonable that handling mixed operations on
complex numbers and ordinary numbers should be the responsibility of
the complex-number package. Combining rational numbers and complex
numbers, however, might be done by the complex package, by the rational
package, or by some third package that uses operations extracted from
these two packages. Formulating coherent policies on the division of
responsibility among packages can be an overwhelming task in designing
systems with many packages and many cross-type operations.

#subheading([Coercion])

#idx("coercion")

In the general situation of completely unrelated operations acting on
completely unrelated types, implementing explicit cross-type operations,
cumbersome though it may be, is the best that one can hope for.
Fortunately, we can usually do better by taking advantage of additional
structure that may be latent in our type system. Often the different
data types are not completely independent, and there may be ways by which
objects of one type may be viewed as being of another type. This process
is called #emph[coercion]. For example, if we are asked to
arithmetically combine an ordinary number with a complex number, we can
view the ordinary number as a complex number whose imaginary part is zero.
This transforms the problem to that of combining two complex numbers, which
can be handled in the ordinary way by the complex-arithmetic package.

In general, we can implement this idea by designing
#idx("coercion", sub: "function")
coercion
functions
that transform an object of one type into an equivalent
object of another type. Here is a typical coercion
function,
which transforms a given ordinary number to a complex number with that real
part and zero imaginary part:
#idx("pythonnumbertocomplex", decl: true)
#snippet(```python
def python_number_to_complex(n):
    return make_complex_from_real_imag(contents(n), 0)
```)

#idx("table", sub: "for coercion")
#idx("coercion", sub: "table")
We install these coercion
functions
in a special coercion table, indexed under the names of the two types:

#snippet(```python
put_coercion("python_number", "complex",
             python_number_to_complex)
```)

(We assume that there are
#py("put_coercion")
and
#py("get_coercion")
functions
available for manipulating this table.) Generally some of the slots in
the table will be empty, because it is not generally possible to coerce
an arbitrary data object of each type into all other types. For example,
there is no way to coerce an arbitrary complex number to an ordinary
number, so there will be no general
#py("complex_to_python_number")
function
included in the table.

Once the coercion table has been set up, we can handle coercion in a
uniform manner by modifying the
#py("apply_generic")
function
of section @sec:data-directed. When asked to apply an
operation, we first check whether the operation is defined for the
arguments' types, just as before. If so, we dispatch to the
function
found in the operation-and-type table. Otherwise, we try coercion. For
simplicity, we consider only the case where there are two
arguments.#footnote[See exercise @ex:multi-coercion for
generalizations.] We check the coercion table to see if objects
of the first type can be coerced to the second type. If so, we coerce the
first argument and try the operation again. If objects of the first type
cannot in general be coerced to the second type, we try the coercion the
other way around to see if there is a way to coerce the second argument to
the type of the first argument. Finally, if there is no known way to coerce
either type to the other type, we give up. Here is the
function:
#idx("applygeneric", sub: "with coercion", decl: true)

#snippet(```python
def apply_generic(op, args):
    type_tags = map(type_tag, args)
    fun = get(op, type_tags)
    if not is_none(fun):
        return apply(fun, map(contents, args))
    else:
        if length(args) == 2:
            type1 = head(type_tags)
            type2 = head(tail(type_tags))
            a1 = head(args)
            a2 = head(tail(args))
            t1_to_t2 = get_coercion(type1, type2)
            t2_to_t1 = get_coercion(type2, type1)
            return (apply_generic(op, llist(t1_to_t2(a1), a2))
                    if not is_none(t1_to_t2)
                    else apply_generic(op, llist(a1, t2_to_t1(a2)))
                    if not is_none(t2_to_t1)
                    else error("no method for these types",
                               llist(op, type_tags)))
        else:
            return error("no method for these types",
                         llist(op, type_tags))
```)

This coercion scheme has many advantages over the method of defining
explicit cross-type operations, as outlined above. Although we still
need to write coercion
functions
to relate the types (possibly $n^(2)$
functions
for a system with $n$ types), we need to write
only one
function
for each pair of types rather than a different
function
for each collection of types and each generic operation.#footnote[If we are
clever, we can usually get by with fewer than
$n^(2)$ coercion
functions.
For instance, if we know how to convert from type 1 to type 2 and from
type 2 to type 3, then we can use this knowledge to convert from type 1 to
type 3. This can greatly decrease the number of coercion
functions
we need to supply explicitly when we add a new type to the system. If we
are willing to build the required amount of sophistication into our system,
we can have it search the "graph" of relations among types and
automatically generate those coercion
functions
that can be inferred from the ones that are supplied
explicitly.] What we are counting on here is the fact that the
appropriate transformation between types depends only on the types
themselves, not on the operation to be applied.

On the other hand, there may be applications for which our coercion
scheme is not general enough. Even when neither of the objects to be
combined can be converted to the type of the other it may still be
possible to perform the operation by converting both objects to a
third type. In order to deal with such complexity and still preserve
modularity in our programs, it is usually necessary to build systems
that take advantage of still further structure in the relations among
types, as we discuss next.

#subheading([Hierarchies of types])

#idx("type(s)", sub: "hierarchy of")
#idx("hierarchy of types")

The coercion scheme presented above relied on the existence of natural
relations between pairs of types. Often there is more "global"
structure in how the different types relate to each other. For
instance, suppose we are building a generic arithmetic system to
handle integers, rational numbers, real numbers, and complex numbers.
In such a system, it is quite natural to regard an integer as a
special kind of rational number, which is in turn a special kind of
real number, which is in turn a special kind of complex number. What
we actually have is a so-called #emph[hierarchy of types], in which,
for example, integers are a
#idx("subtype")
#idx("type(s)", sub: "subtype")
#emph[subtype] of rational numbers (i.e.,
any operation that can be applied to a rational number can
automatically be applied to an integer). Conversely, we say that
rational numbers form a
#idx("supertype")
#idx("type(s)", sub: "supertype")
#emph[supertype] of integers. The particular
hierarchy we have here is of a very simple kind, in which each type
has at most one supertype and at most one subtype. Such a structure,
called a #emph[tower], is illustrated in
figure @fig:tower.

#sicp-figure(image("/images/img_original/ch2-Z-G-66.svg", width: 70%), caption: [A tower of types. #idx("tower of types") #idx("type(s)", sub: "tower of")], label-name: <fig:tower>)

If we have a tower structure, then we can greatly simplify the problem
of adding a new type to the hierarchy, for we need only specify how
the new type is embedded in the next supertype above it and how it is
the supertype of the type below it. For example, if we want to add an
integer to a complex number, we need not explicitly define a special
coercion
function
#py("integer_to_complex").
Instead, we define how an integer can be transformed into a rational
number, how a rational number is transformed into a real number, and how
a real number is transformed into a complex number. We then allow the
system to transform the integer into a complex number through these steps
and then add the two complex numbers.

#idx("type(s)", sub: "raising")
#idx("applygeneric", sub: "with tower of types")
We can redesign our
#py("apply_generic")
function
in the following way: For each type, we need to supply a
#py("promote")
function,
which "raises" objects of that type one level in the tower.
Then when the system is required to operate on objects of different types
it can successively raise the lower types until all the objects are at
the same level in the tower. (Exercises @ex:raise
and  @ex:apply-with-raise
concern the details of implementing such a strategy.)

Another advantage of a tower is that we can easily implement the notion
that every type "inherits" all operations defined on a
supertype. For instance, if we do not supply a special
function
for finding the real part of an integer, we should nevertheless expect
that
#py("real_part")
will be defined for integers by virtue of the fact that integers are a
subtype of complex numbers. In a tower, we can arrange for this to happen
in a uniform way by modifying
#py("apply_generic").
If the required operation is not directly defined for the type of the
object given, we raise the object to its supertype and try again. We thus
crawl up the tower, transforming our argument as we go, until we either
find a level at which the desired operation can be performed or hit the
top (in which case we give up).

#idx("type(s)", sub: "lowering")
Yet another advantage of a tower over a more general hierarchy is that
it gives us a simple way to "lower" a data object to the
simplest representation. For example, if we add
$2+3i$ to $4-3i$,
it would be nice to obtain the answer as the integer 6 rather than as the
complex number $6+0i$.
Exercise @ex:simplify discusses a way to implement
such a lowering operation. (The trick is that we need a general way
to distinguish those objects that can be lowered, such as
$6+0i$, from those that cannot, such as
$6+2i$.)

#subheading([Inadequacies of hierarchies])

#idx("hierarchy of types", sub: "inadequacy of")

If the data types in our system can be naturally arranged in a tower,
this greatly simplifies the problems of dealing with generic operations
on different types, as we have seen. Unfortunately, this is usually
not the case. Figure @fig:relations-among-figures
illustrates a more complex arrangement of mixed types, this one showing
relations among different types of geometric figures. We see that, in
general,
#idx("type(s)", sub: "multiple subtype and supertype")
#idx("supertype", sub: "multiple")
#idx("subtype", sub: "multiple")
a type may have more than one subtype. Triangles and quadrilaterals,
for instance, are both subtypes of polygons. In addition, a type may
have more than one supertype. For example, an isosceles right
triangle may be regarded either as an isosceles triangle or as a right
triangle. This multiple-supertypes issue is particularly thorny,
since it means that there is no unique way to "raise" a type
in the hierarchy. Finding the "correct" supertype in which
to apply an operation to an object may involve considerable searching
through the entire type network on the part of a
function
such as
#py("apply_generic").
Since there generally are multiple subtypes for a type, there is a similar
problem in coercing a value "down" the type hierarchy.
Dealing with large numbers of interrelated types while still preserving
modularity in the design of large systems is very difficult, and is an area
of much current research.#footnote[This statement, which also appears in
the first edition of this book, is just as true now as it was when we wrote
it
in 1984.
Developing a useful, general framework for expressing
the relations among different types of entities (what philosophers call
"ontology") seems intractably difficult. The main difference
between the confusion that existed
in 1984
and the confusion that
exists now is that now a variety of inadequate ontological theories have
been embodied in a plethora of correspondingly inadequate programming
languages. For example, much of the complexity of
#idx("object-oriented programming languages")
#idx("programming language", sub: "object-oriented")
object-oriented programming languages—and the subtle and confusing
differences among contemporary object-oriented
languages—centers on the treatment of generic operations on
interrelated types. Our own discussion of computational objects in
chapter @chap:state avoids these issues entirely. Readers familiar with
object-oriented programming will notice that we have much to say in
chapter @chap:state about local state, but we do not even mention
"classes" or "inheritance." In fact, we suspect
that these problems cannot be adequately addressed in terms of
computer-language design alone, without also drawing on work in knowledge
representation and automated reasoning.]

#sicp-figure(image("/images/img_original/ch2-Z-G-67.svg", width: 70%), caption: [Relations among types of geometric figures.], label-name: <fig:relations-among-figures>)

#exercise(label-name: <ex:2_81>, [
#idx("applygeneric", sub: "with coercion")
Louis Reasoner has noticed that
#py("apply_generic")
may try to coerce the arguments to each other's type even if they
already have the same type. Therefore, he reasons, we need to put
functions
in the coercion table to "coerce" arguments of each type to
their own type. For example, in addition to the
#py("python_number_to_complex")
coercion shown above, he would do:
#idx("pythonnumbertopythonnumber", decl: true)#idx("complextocomplex", decl: true)
#snippet(```python
def python_number_to_python_number(n): return n

def complex_to_complex(n): return n

put_coercion("python_number", "python_number",
             python_number_to_python_number)
put_coercion("complex", "complex", complex_to_complex)
```)

+ With Louis's coercion functions installed, what happens if #py("apply_generic") is called with two arguments of type #py("\"complex\"") or two arguments of type #py("\"python_number\"") for an operation that is not found in the table for those types? For example, assume that we've defined a generic exponentiation operation: #snippet(```python def exp(x, y): return apply_generic("exp", llist(x, y)) ```) and have put a function for exponentiation in the Python-number package but not in any other package: #syntax(" # following added to Python-number package put(\"exp\", llist(\"python_number\", \"python_number\"), lambda x, y: tag(math_pow(x, y))) # using primitive ", $mono("math_pow")$) What happens if we call #py("exp") with two complex numbers as arguments?
+ Is Louis correct that something had to be done about coercion with arguments of the same type, or does #py("apply_generic") work correctly as is?
+ Modify #py("apply_generic") so that it doesn't try coercion if the two arguments have the same type.
])

#exercise(label-name: <ex:multi-coercion>, [
#idx("applygeneric", sub: "with coercion of multiple arguments")
Show how to generalize
#py("apply_generic")
to handle coercion in the general case of multiple arguments. One
strategy is to attempt to coerce all the arguments to the type of the
first argument, then to the type of the second argument, and so on.
Give an example of a situation where this strategy (and likewise the
two-argument version given above) is not sufficiently general.
(Hint: Consider the case where there are some suitable mixed-type
operations present in the table that will not be tried.)
])

#exercise(label-name: <ex:raise>, [
#idx("type(s)", sub: "raising")
Suppose you are designing a generic arithmetic system for dealing with
the tower of types shown in figure @fig:tower:
integer, rational, real, complex. For
each type (except complex), design a
function
that raises objects of that type one level in the tower. Show how to
install a generic #py("promote") operation that will
work for each type (except complex).
])

#exercise(label-name: <ex:apply-with-raise>, [
#idx("applygeneric", sub: "with coercion by raising")
Using the #py("promote") operation of
exercise @ex:raise, modify the
#py("apply_generic")
function
so that it coerces its arguments to have the same type by the method of
successive raising, as discussed in this section. You will need to devise
a way to test which of two types is higher in the tower. Do this in a
manner that is "compatible" with the rest of the system and
will not lead to problems in adding new levels to the tower.
])

#exercise(label-name: <ex:simplify>, [
#idx("applygeneric", sub: "with coercion to simplify")
#idx("type(s)", sub: "lowering")
This section mentioned a method for "simplifying" a data object
by lowering it in the tower of types as far as possible. Design a
function
#py("drop") that accomplishes this for the tower
described in exercise @ex:raise. The key is to decide,
in some general way, whether an object can be lowered. For example, the
complex number $1.5+0i$ can be lowered as far as
#py("\"real\""),
the complex number $1+0i$ can be lowered as far
as
#py("\"integer\""),
and the complex number $2+3i$ cannot be lowered
at all. Here is a plan for determining whether an object can be lowered:
Begin by defining a generic operation #py("project")
that "pushes" an object down in the tower. For example,
projecting a complex number would involve throwing away the imaginary part.
Then a number can be dropped if, when we
#py("project") it and
#py("promote") the result back to the type we started
with, we end up with something equal to what we started with. Show how to
implement this idea in detail, by writing a
#py("drop")
function
that drops an object as far as possible. You will need to design the
various projection operations#footnote[A real number can be projected to
an integer using the
#idx("mathround (primitive function)")

#py("math_round")
primitive, which returns the closest integer
to its argument.] and install
#py("project") as a generic operation in the system.
You will also need to make use of a generic equality predicate, such as
described in exercise @ex:equ-. Finally, use
#py("drop")
to rewrite
#py("apply_generic")
from exercise @ex:apply-with-raise so that it
"simplifies" its answers.
])

#exercise(label-name: <ex:2_86>, [
Suppose we want to handle complex numbers whose real
parts, imaginary parts, magnitudes, and angles can be either ordinary
numbers, rational numbers, or other numbers we might wish to add to
the system. Describe and implement the changes to the system needed
to accommodate this. You will have to define operations such as
#py("sine") and #py("cosine")
that are generic over ordinary numbers and rational numbers.
])

#idx("coercion")
#idx("type(s)", sub: "hierarchy of")
#idx("hierarchy of types")
