// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Representations for Complex Numbers], label-name: <sec:representations-complex-numbers>)

#idx("complex numbers", sub: "rectangular vs. polar form")

We will develop a system that performs arithmetic operations on complex
numbers as a simple but unrealistic example of a program that uses generic
operations. We begin by discussing two plausible representations for
complex numbers as ordered pairs: rectangular form (real part and imaginary
part) and polar form (magnitude and angle).#footnote[In actual computational
systems, rectangular form is preferable to polar form most of the time
because of
#idx("roundoff error")
roundoff errors in conversion between rectangular and polar form. This is
why the complex-number example is unrealistic. Nevertheless, it provides a
clear illustration of the design of a system using generic operations and a
good introduction to the more substantial systems to be developed later in
this chapter.] Section @sec:manifest-types
will show how both representations can be made to coexist in a single
system through the use of type tags and generic operations.

Like rational numbers, complex numbers are naturally represented as ordered
pairs. The set of complex numbers can be thought of as a two-dimensional
space with two orthogonal axes, the "real" axis and the
"imaginary" axis. (See
figure @fig:complex-plane.) From this point of view,
the complex number $z=x+i y$ (where
$i^(2) =-1$) can be thought of as the point in
the plane whose real coordinate is $x$ and whose
imaginary coordinate is $y$. Addition of complex
numbers reduces in this representation to addition of coordinates:

$ mat(delim: #none, upright("Real-part")(z_(1)+z_(2)), =, upright("Real-part")(z_(1))+upright("Real-part")(z_(2)); upright("Imaginary-part")(z_(1) +z_(2)), =, upright("Imaginary-part")(z_(1))+upright("Imaginary-part")(z_(2))) $

When multiplying complex numbers, it is more natural to think in terms
of representing a complex number in polar form, as a magnitude and an
angle ($r$ and $A$
in figure @fig:complex-plane). The product of two
complex numbers is the vector obtained by stretching one complex number by
#idx("complex numbers", sub: "rectangular vs. polar form")
the length of the other and then rotating it through the angle of the other:

$ mat(delim: #none, upright("Magnitude")(z_(1) dot.op z_(2)), =, upright("Magnitude")(z_(1)) dot.op upright("Magnitude")(z_(2)); upright("Angle")(z_(1) dot.op z_(2)), =, upright("Angle")(z_(1))+upright("Angle")(z_(2))) $

Thus, there are two different representations for complex numbers,
which are appropriate for different operations. Yet, from the
viewpoint of someone writing a program that uses complex numbers, the
principle of data abstraction suggests that all the operations for
manipulating complex numbers should be available regardless of which
representation is used by the computer. For example, it is often
useful to be able to find the magnitude of a complex number that is
specified by rectangular coordinates. Similarly, it is often useful
to be able to determine the real part of a complex number that is
specified by polar coordinates.

#sicp-figure(image("/images/img_original/ch2-Z-G-59.svg", width: 70%), caption: [Complex numbers as points in the plane.], label-name: <fig:complex-plane>)

To design such a system, we can follow the same
#idx("data abstraction")
data-abstraction strategy we followed in designing the rational-number
package in section @sec:rationals. Assume that the
operations on complex numbers are implemented in terms of four selectors:
#py("real_part"),
#py("imag_part"),
#py("magnitude"),
and #py("angle"). Also assume that we have two
functions
for constructing complex numbers:
#py("make_from_real_imag")
returns a complex number with specified real and imaginary parts, and
#py("make_from_mag_ang")
returns a complex number with specified magnitude and angle. These
functions
have the property that, for any complex number
#py("z"), both

#snippet(```python
make_from_real_imag(real_part(z), imag_part(z))
```)

and

#snippet(```python
make_from_mag_ang(magnitude(z), angle(z))
```)

produce complex numbers that are equal to #py("z").

Using these constructors and selectors, we can implement arithmetic on
complex numbers using the "abstract data" specified by the
constructors and selectors, just as we did for rational numbers in
section @sec:rationals. As shown in the formulas
above, we can add and subtract complex numbers in terms of real and
imaginary parts while multiplying and dividing complex numbers in terms of
magnitudes and angles:
#idx("addcomplex", decl: true)#idx("subcomplex", decl: true)#idx("mulcomplex", decl: true)#idx("divcomplex", decl: true)
#snippet(```python
def add_complex(z1, z2):
    return make_from_real_imag(real_part(z1) + real_part(z2),
                               imag_part(z1) + imag_part(z2))
def sub_complex(z1, z2):
    return make_from_real_imag(real_part(z1) - real_part(z2),
                               imag_part(z1) - imag_part(z2))
def mul_complex(z1, z2):
    return make_from_mag_ang(magnitude(z1) * magnitude(z2),
                             angle(z1) + angle(z2))
def div_complex(z1, z2):
    return make_from_mag_ang(magnitude(z1) / magnitude(z2),
                             angle(z1) - angle(z2))
```)

To complete the complex-number package, we must choose a representation and
we must implement the constructors and selectors in terms of primitive
numbers and primitive
linked-list structure. There are two obvious ways to do
this: We can represent a complex number in "rectangular form"
as a pair (real part, imaginary part) or in "polar form" as a
pair (magnitude, angle). Which shall we choose?

In order to make the different choices concrete, imagine that there are two
programmers, Ben Bitdiddle and Alyssa P. Hacker, who are independently
designing representations for the complex-number system.
Ben chooses to represent
#idx("complex numbers", sub: "rectangular representation")
complex numbers in rectangular form. With this
choice, selecting the real and imaginary parts of a complex number is
straightforward, as is constructing a complex number with given real and
imaginary parts. To find the magnitude and the angle, or to construct a
complex number with a given magnitude and angle, he uses the trigonometric
relations

$ mat(delim: #none, x, =, r space cos A, , r, =, sqrt(x^(2) +y^(2)); y, =, r space sin A, , A, =, arctan (y,x)) $

which relate the real and imaginary parts ($x$,
$y$) to the magnitude and the angle
$(r, A)$.#footnote[The arctangent function
referred to
here,
computed by Python's #idx("arctangent") #idx("mathatan2 (primitive function)") #py("math_atan2") function,
is defined so as to take two arguments
$y$ and $x$
and to return the angle whose tangent is $y/x$.
The signs of the arguments determine the quadrant of the angle.]
Ben's representation is therefore given by the following selectors
and constructors:
#idx("realpart", sub: "rectangular representation", decl: true)#idx("imagpart", sub: "rectangular representation", decl: true)#idx("magnitude", sub: "rectangular representation", decl: true)#idx("angle", sub: "rectangular representation", decl: true)#idx("makefromrealimag", sub: "rectangular representation", decl: true)#idx("makefrommagang", sub: "rectangular representation", decl: true)
#snippet(```python
def real_part(z): return head(z)

def imag_part(z): return tail(z)

def magnitude(z):
    return math_sqrt(square(real_part(z)) + square(imag_part(z)))
def angle(z):
    return math_atan2(imag_part(z), real_part(z))
def make_from_real_imag(x, y): return pair(x, y)

def make_from_mag_ang(r, a):
    return pair(r * math_cos(a), r * math_sin(a))
```)

Alyssa, in contrast, chooses to represent complex numbers in
#idx("complex numbers", sub: "polar representation")
polar form.

For her, selecting the magnitude and angle is straightforward, but she has
to use the
#idx("trigonometric relations")
trigonometric relations to obtain the real and imaginary parts.
Alyssa's representation is:
#idx("realpart", sub: "polar representation", decl: true)#idx("imagpart", sub: "polar representation", decl: true)#idx("magnitude", sub: "polar representation", decl: true)#idx("angle", sub: "polar representation", decl: true)#idx("makefromrealimag", sub: "polar representation", decl: true)#idx("makefrommagang", sub: "polar representation", decl: true)
#snippet(```python
def real_part(z):
    return magnitude(z) * math_cos(angle(z))
def imag_part(z):
    return magnitude(z) * math_sin(angle(z))
def magnitude(z): return head(z)

def angle(z): return tail(z)

def make_from_real_imag(x, y):
    return pair(math_sqrt(square(x) + square(y)),
                math_atan2(y, x))
def make_from_mag_ang(r, a): return pair(r, a)
```)

The discipline of data abstraction ensures that the same implementation of
#py("add_complex"),
#py("sub_complex"),
#py("mul_complex"),
and
#py("div_complex")
will work with either Ben's representation or Alyssa's
representation.
