// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Tagged data], label-name: <sec:manifest-types>)

#idx("complex numbers", sub: "represented as tagged data")
#idx("tagged data")
#idx("data", sub: "tagged")

One way to view data abstraction is as an application of the
#idx("principle of least commitment")
#idx("least commitment, principle of")
"principle of least commitment." In implementing the
complex-number system in
section @sec:representations-complex-numbers, we can
use either Ben's rectangular representation or Alyssa's polar
representation. The abstraction barrier formed by the selectors and
constructors permits us to defer to the last possible moment the choice of
a concrete representation for our data objects and thus retain maximum
flexibility in our system design.

The principle of least commitment can be carried to even further extremes.
If we desire, we can maintain the ambiguity of representation even
#emph[after] we have designed the selectors and constructors, and elect
to use both Ben's representation #emph[and] Alyssa's
representation. If both representations are included in a single system,
however, we will need some way to distinguish data in polar form from data
in rectangular form. Otherwise, if we were asked, for instance, to find
the #py("magnitude") of the pair
$(3,4)$, we wouldn't know whether to
answer 5 (interpreting the number in rectangular form) or
3 (interpreting the number in polar form). A straightforward way to
accomplish this distinction is to include a
#idx("type tag")
#emph[type tag]—the
string #py("\"rectangular\"")
or
#py("\"polar\"")—as
part of each complex number. Then when we need to manipulate a complex
number we can use the tag to decide which selector to apply.

In order to manipulate tagged data, we will assume that we have
functions
#py("type_tag")
and #py("contents") that extract from a data object
the tag and the actual contents (the polar or rectangular coordinates, in
the case of a complex number). We will also postulate a
function
#py("attach_tag")
that takes a tag and contents and produces a tagged data object. A
straightforward way to implement this is to use ordinary
linked-list structure:
#idx("attachtag", decl: true)#idx("typetag", decl: true)#idx("contents", decl: true)
#snippet(```python
def attach_tag(type_tag, contents):
    return pair(type_tag, contents)
def type_tag(datum):
    return (head(datum)
            if is_pair(datum)
            else error("bad tagged datum -- type_tag", datum))
def contents(datum):
    return (tail(datum)
            if is_pair(datum)
            else error("bad tagged datum -- contents", datum))
```)

Using #py("type_tag"),
we can define predicates
#py("is_rectangular")
and
#py("is_polar"),
which recognize rectangular and polar numbers, respectively:
#idx("isrectangular", decl: true)#idx("ispolar", decl: true)
#snippet(```python
def is_rectangular(z):
    return type_tag(z) == "rectangular"
def is_polar(z):
    return type_tag(z) == "polar"
```)

With type tags, Ben and Alyssa can now modify their code so that their two
different representations can coexist in the same system. Whenever Ben
constructs a complex number, he tags it as rectangular. Whenever Alyssa
constructs a complex number, she tags it as polar. In addition, Ben and
Alyssa must make sure that the names of their
functions
do not conflict. One way to do this is for Ben to append the suffix
#py("rectangular") to the name of each of his
representation
functions
and for Alyssa to append #py("polar") to the names of
hers. Here is Ben's revised rectangular representation from
section @sec:representations-complex-numbers:
#idx("realpartrectangular", decl: true)#idx("imagpartrectangular", decl: true)#idx("magnituderectangular", decl: true)#idx("anglerectangular", decl: true)#idx("makefromrealimagrectangular", decl: true)#idx("makefrommagangrectangular", decl: true)
#snippet(```python
def real_part_rectangular(z): return head(z)

def imag_part_rectangular(z): return tail(z)

def magnitude_rectangular(z):
    return math_sqrt(square(real_part_rectangular(z)) +
                     square(imag_part_rectangular(z)))
def angle_rectangular(z):
    return math_atan2(imag_part_rectangular(z),
                      real_part_rectangular(z))
def make_from_real_imag_rectangular(x, y):
    return attach_tag("rectangular", pair(x, y))
def make_from_mag_ang_rectangular(r, a):
    return attach_tag("rectangular",
                      pair(r * math_cos(a), r * math_sin(a)))
```)

and here is Alyssa's revised polar representation:
#idx("realpartpolar", decl: true)#idx("imagpartpolar", decl: true)#idx("magnitudepolar", decl: true)#idx("anglepolar", decl: true)#idx("makefromrealimagpolar", decl: true)#idx("makefrommagangpolar", decl: true)
#snippet(```python
def real_part_polar(z):
    return magnitude_polar(z) * math_cos(angle_polar(z))
def imag_part_polar(z):
    return magnitude_polar(z) * math_sin(angle_polar(z))
def magnitude_polar(z): return head(z)

def angle_polar(z): return tail(z)

def make_from_real_imag_polar(x, y):
    return attach_tag("polar",
                      pair(math_sqrt(square(x) + square(y)),
                           math_atan2(y, x)))

def make_from_mag_ang_polar(r, a):
    return attach_tag("polar", pair(r, a))
```)

#idx("selector", sub: "generic")
#idx("generic function", sub: "generic selector")
Each generic selector is implemented as a
function
that checks the tag of its argument and calls the appropriate
function
for handling data of that type. For example, to obtain the real part of
a complex number,
#py("real_part")
examines the tag to determine whether to use Ben's
#py("real_part_rectangular")
or Alyssa's
#py("real_part_polar").
In either case, we use #py("contents") to extract the
bare, untagged datum and send this to the rectangular or polar
function
as required:
#idx("realpart", sub: "with tagged data", decl: true)#idx("imagpart", sub: "with tagged data", decl: true)#idx("magnitude", sub: "with tagged data", decl: true)#idx("angle", sub: "with tagged data", decl: true)
#snippet(```python
def real_part(z):
    return (real_part_rectangular(contents(z))
            if is_rectangular(z)
            else real_part_polar(contents(z))
            if is_polar(z)
            else error("unknown type -- real_part", z))
def imag_part(z):
    return (imag_part_rectangular(contents(z))
            if is_rectangular(z)
            else imag_part_polar(contents(z))
            if is_polar(z)
            else error("unknown type -- imag_part", z))
def magnitude(z):
    return (magnitude_rectangular(contents(z))
            if is_rectangular(z)
            else magnitude_polar(contents(z))
            if is_polar(z)
            else error("unknown type -- magnitude", z))
def angle(z):
    return (angle_rectangular(contents(z))
            if is_rectangular(z)
            else angle_polar(contents(z))
            if is_polar(z)
            else error("unknown type -- angle", z))
```)

To implement the complex-number arithmetic operations, we can use the same
functions
#py("add_complex"),
#py("sub_complex"),
#py("mul_complex"),
and
#py("div_complex")
from section @sec:representations-complex-numbers,
because the selectors they call are generic, and so will work with either
representation. For example, the
function
#py("add_complex")
is still

#snippet(```python
def add_complex(z1, z2):
    return make_from_real_imag(real_part(z1) + real_part(z2),
                               imag_part(z1) + imag_part(z2))
```)

Finally, we must choose whether to construct complex numbers using
Ben's representation or Alyssa's representation. One
reasonable choice is to construct rectangular numbers whenever we have
real and imaginary parts and to construct polar numbers whenever we have
magnitudes and angles:
#idx("makefromrealimag", decl: true)#idx("makefrommagang", decl: true)
#snippet(```python
def make_from_real_imag(x, y):
    return make_from_real_imag_rectangular(x, y)
def make_from_mag_ang(r, a):
    return make_from_mag_ang_polar(r, a)
```)

#sicp-figure(image("/images/img_javascript/ch2-Z-G-62.svg", width: 70%), caption: [Structure #idx("complex-number arithmetic", sub: "structure of system") of the generic complex-arithmetic system.], label-name: <fig:generic-complex-system>)

The resulting complex-number system has the structure shown in
figure @fig:generic-complex-system.
The system has been decomposed into three relatively independent parts: the
complex-number-arithmetic operations, Alyssa's polar implementation,
and Ben's rectangular implementation. The polar and rectangular
implementations could have been written by Ben and Alyssa working
separately, and both of these can be used as underlying representations by
a third programmer implementing the complex-arithmetic
functions
in terms of the abstract constructor/selector interface.

Since each data object is tagged with its type, the selectors operate on
the data in a
#idx("selector", sub: "generic")
#idx("generic function", sub: "generic selector")
generic manner. That is, each selector is defined to have a
behavior that depends upon the particular type of data it is applied to.
Notice the general mechanism for interfacing the separate representations:
Within a given representation implementation (say, Alyssa's polar
package) a complex number is an untyped pair (magnitude, angle). When a
generic selector operates on a number of #py("polar")
type, it strips off the tag and passes the contents on to Alyssa's
code. Conversely, when Alyssa constructs a number for general use, she
tags it with a type so that it can be appropriately recognized by the
higher-level
functions.
This discipline of stripping off and attaching tags as data objects are
passed from level to level can be an important organizational strategy,
as we shall see in section @sec:generic-operators.
#idx("complex numbers", sub: "represented as tagged data")
#idx("tagged data")
#idx("data", sub: "tagged")
