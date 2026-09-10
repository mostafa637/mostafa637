// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Abstraction Barriers], label-name: <sec:abstraction-barriers>)

#idx("abstraction barriers")
Before continuing with more examples of compound data and data
abstraction, let us consider some of the issues raised by the
rational-number example. We defined the rational-number operations in
terms of a constructor
#py("make_rat")
and selectors #py("numer") and
#py("denom"). In general, the underlying idea of data
abstraction is to identify for each type of data object a basic set of
operations in terms of which all manipulations of data objects of that type
will be expressed, and then to use only those operations in manipulating the
data.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-6.svg", width: 65%), caption: [Data-abstraction barriers in the rational-number package.], label-name: <fig:abstraction-barriers>)

We can envision the structure of the rational-number system as
shown in
figure @fig:abstraction-barriers.
The horizontal lines represent #emph[abstraction barriers] that isolate
different "levels" of the system. At each level, the barrier
separates the programs (above) that use the data abstraction from the
programs (below) that implement the data abstraction. Programs that
use rational numbers manipulate them solely in terms of the
functions
supplied "for public use" by the rational-number package:
#py("add_rat"),
#py("sub_rat"),
#py("mul_rat"),
#py("div_rat"),
and
#py("equal_rat").
These, in turn, are implemented solely in terms of the
#idx("constructor", sub: "as abstraction barrier")
constructor and
#idx("selector", sub: "as abstraction barrier")
selectors
#py("make_rat"),
#py("numer"), and #py("denom"),
which themselves are implemented in terms of pairs. The details of how
pairs are implemented are irrelevant to the rest of the rational-number
package so long as pairs can be manipulated by the use of
#py("pair"),
#py("head"),
and
#py("tail").
In effect,
functions
at each level are the interfaces that define the abstraction barriers and
connect the different levels.

This simple idea has many advantages. One advantage is that it makes
programs much easier to maintain and to modify. Any complex data
structure can be represented in a variety of ways with the primitive
data structures provided by a programming language. Of course, the
choice of representation influences the programs that operate on it;
thus, if the representation were to be changed at some later time, all
such programs might have to be modified accordingly. This task could
be time-consuming and expensive in the case of large programs unless
the dependence on the representation were to be confined by design to
a very few program modules.

For example, an alternate way to address the problem of
#idx("rational number(s)", sub: "reducing to lowest terms")
#idx("reducing to lowest terms")
#idx("makerat", decl: true)
#idx("denom", sub: "reducing to lowest terms", decl: true)
#idx("numer", sub: "reducing to lowest terms", decl: true)
reducing rational
numbers to lowest terms is to perform the reduction whenever we
access the parts of a rational number, rather than when we construct
it. This leads to different constructor and selector
functions:

#snippet(```python
def make_rat(n, d):
    return pair(n, d)
def numer(x):
    g = gcd(head(x), tail(x))
    return head(x) // g
def denom(x):
    g = gcd(head(x), tail(x))
    return tail(x) // g
```)

The difference between this implementation and the previous one lies in when
we compute the #py("gcd"). If in our typical use of
rational numbers we access the numerators and denominators of the same
rational numbers many times, it would be preferable to compute the
#py("gcd") when the rational numbers are constructed.
If not, we may be better off waiting until access time to compute the
#py("gcd"). In any case, when we change from one
representation to the other, the
functions
#py("add_rat"),
#py("sub_rat"),
and so on do not have to be modified at all.

Constraining the dependence on the representation to a few interface
functions
helps us design programs as well as modify them, because it allows us to
maintain the flexibility to consider alternate implementations. To continue
with our simple example, suppose we are designing a rational-number package
and we can't decide initially whether to perform the
#py("gcd") at construction time or at selection time.
The data-abstraction methodology gives us a way to defer that decision
without losing the ability to make progress on the rest of the system.

#exercise(label-name: <ex:segments1>, [
Consider the problem of representing
#idx("line segment", sub: "represented as pair of points")
line segments in a plane. Each segment is represented as a pair of points:
a starting point and an ending point.
Declare
a constructor
#idx("makesegment")
#py("make_segment")
and selectors
#idx("startsegment")
#py("start_segment")
and
#idx("endsegment")
#py("end_segment")
that define the representation of segments in
terms of points. Furthermore, a point
#idx("point, represented as a pair")
can be represented as a pair
of numbers: the $x$ coordinate and the
$y$ coordinate. Accordingly, specify a
constructor
#idx("makepoint")
#py("make_point")
and selectors
#py("x_point")
and
#py("y_point")
that define this representation. Finally, using your selectors and
constructors,
declare a function
#idx("midpointsegment")
#py("midpoint_segment")
that takes a line segment as argument and returns its midpoint (the point
whose coordinates are the average of the coordinates of the endpoints).
To try your
functions,
you'll need a way to print points:
#idx("printpoint", decl: true)
#snippet(```python
def print_point(p):
    print("(" + str(x_point(p)) + ", "
              + str(y_point(p)) + ")")
```)
])

#exercise(label-name: <ex:rectangles>, [
Implement a representation for
#idx("rectangle, representing")
rectangles in a plane. (Hint: You may want to
make use of exercise @ex:segments1.) In terms of your
constructors and selectors, create
functions
that compute the perimeter and the area of a given rectangle. Now implement
a different representation for rectangles. Can you design your system with
suitable abstraction barriers, so that the same perimeter and area
functions
will work using either representation?
])

#idx("abstraction barriers")
