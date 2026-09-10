// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Formulating Abstractions with Higher-Order Functions], label-name: <sec:higher-order-procedures>)

We have seen that
functions
are, in effect, abstractions that describe compound operations on
numbers independent of the particular numbers. For example, when we
define
#idx("cube", decl: true)
#snippet(```python
def cube(x):
    return x * x * x
```)

we are not talking about the cube of a particular number, but rather
about a method for obtaining the cube of any number. Of course we could
get along without ever
defining this function,
by always writing expressions such as

#snippet(```python
3 * 3 * 3
x * x * x
y * y * y
```)

and never mentioning #py("cube") explicitly. This
would place us at a serious disadvantage, forcing us to work always at
the level of the particular operations that happen to be primitives in
the language (multiplication, in this case) rather than in terms of
higher-level operations. Our programs would be able to compute cubes,
but our language would lack the ability to express the concept of cubing.
One of the things we should demand from a powerful programming language
is the ability to build abstractions by assigning names to common
patterns and then to work in terms of the abstractions directly.
Functions
provide this ability. This is why all but the most primitive
programming languages include mechanisms for
defining functions.

Yet even in numerical processing we will be severely limited in our
ability to create abstractions if we are restricted to
functions
whose parameters must be numbers. Often the same programming pattern
will be used with a number of different
functions.
To express such patterns as concepts, we will need to construct
functions
that can accept
functions
as arguments or return
functions
as values.
Functions
that manipulate
functions
are called
#idx("higher-order functions")
#emph[higher-order functions.]
This section shows how higher-order
functions
can serve as powerful abstraction mechanisms, vastly increasing the
expressive power of our language.

#include "../../chapter1/section3/subsection1.typ"

#include "../../chapter1/section3/subsection2.typ"

#include "../../chapter1/section3/subsection3.typ"

#include "../../chapter1/section3/subsection4.typ"
