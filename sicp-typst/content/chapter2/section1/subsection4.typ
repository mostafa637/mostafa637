// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Extended Exercise: Interval Arithmetic], label-name: <sec:interval-arith>)

#idx("interval arithmetic")
#idx("arithmetic", sub: "on intervals")

Alyssa P. Hacker is designing a system to help people solve
engineering problems. One feature she wants to provide in her system
is the ability to manipulate inexact quantities (such as measured
parameters of physical devices) with known precision, so that when
computations are done with such approximate quantities the results
will be numbers of known precision.

Electrical engineers will be using Alyssa's system to compute
electrical quantities. It is sometimes necessary for them to compute
the value of a parallel equivalent resistance
$R_(p)$ of two resistors
$R_(1)$ and $R_(2)$
#idx("resistance", sub: "formula for parallel resistors")
using the formula

$ mat(delim: #none, R_(p), =, frac(1, 1/R_(1)+1/R_(2))) $

Resistance values are usually known only up to some
#idx("resistance", sub: "tolerance of resistors")
tolerance guaranteed by the manufacturer of the resistor. For example, if
you buy a resistor labeled "6.8 ohms with 10% tolerance" you can
only be sure that the resistor has a resistance between
$6.8-0.68=6.12$ and
$6.8+0.68=7.48$ ohms. Thus, if you have a
6.8-ohm 10% resistor in parallel with a 4.7-ohm
5% resistor, the resistance of the combination can range from about
2.58 ohms (if the two resistors are at the lower bounds) to about 2.97 ohms
(if the two resistors are at the upper bounds).

Alyssa's idea is to implement "interval arithmetic" as a
set of arithmetic operations for combining "intervals" (objects
that represent the range of possible values of an inexact quantity). The
result of adding, subtracting, multiplying, or dividing two intervals is
itself an interval, representing the range of the result.

Alyssa postulates the existence of an abstract object called an
"interval" that has two endpoints: a lower bound and an upper bound.
She also presumes that, given the endpoints of an interval, she can
construct the interval using the data constructor
#idx("makeinterval")
#py("make_interval").
Alyssa first writes a
function
for adding two intervals. She reasons that the minimum value the sum could
be is the sum of the two lower bounds and the maximum value it could be is
the sum of the two upper bounds:
#idx("addinterval", decl: true)
#snippet(```python
def add_interval(x, y):
    return make_interval(lower_bound(x) + lower_bound(y),
                         upper_bound(x) + upper_bound(y))
```)

Alyssa also works out the product of two intervals by finding the
minimum and the maximum of the products of the bounds and using them
as the bounds of the resulting interval.

#idx("min (primitive function)")
(The functions #py("min")
and
#idx("max (primitive function)")

#py("max")
are
primitives that find the minimum or maximum of any number of arguments.)
#idx("mulinterval", decl: true)
#snippet(```python
def mul_interval(x, y):
    p1 = lower_bound(x) * lower_bound(y)
    p2 = lower_bound(x) * upper_bound(y)
    p3 = upper_bound(x) * lower_bound(y)
    p4 = upper_bound(x) * upper_bound(y)
    return make_interval(min(p1, p2, p3, p4),
                         max(p1, p2, p3, p4))
```)

To divide two intervals, Alyssa multiplies the first by the reciprocal of
the second. Note that the bounds of the reciprocal interval are
the reciprocal of the upper bound and the reciprocal of the lower bound, in
that order.
#idx("divinterval", decl: true)
#snippet(```python
def div_interval(x, y):
    return mul_interval(x, make_interval(1 / upper_bound(y),
                                         1 / lower_bound(y)))
```)

#exercise(label-name: <ex:alyssa-interval-start>, [
Alyssa's program is incomplete because she has not specified the
implementation of the interval abstraction. Here is a definition of
the interval constructor:

#idx("makeinterval", decl: true)
#snippet(```python
def make_interval(x, y): return pair(x, y)
```)

Define selectors
#idx("upperbound")
#py("upper_bound")
and
#idx("lowerbound")
#py("lower_bound")
to complete the implementation.

#anchor(<ex:2_7>)
])

#exercise(label-name: <ex:2_8>, [
Using reasoning analogous to Alyssa's, describe how the difference
of two intervals may be computed. Define a corresponding subtraction
function,
called
#idx("subinterval")
#py("sub_interval").
])

#exercise(label-name: <ex:2_9>, [
The
#idx("width of an interval")
#emph[width] of an interval is half of the difference between its
upper and lower bounds. The width is a measure of the uncertainty of
the number specified by the interval. For some arithmetic operations
the width of the result of combining two intervals is a function only
of the widths of the argument intervals, whereas for others the width
of the combination is not a function of the widths of the argument
intervals. Show that the width of the sum (or difference) of two
intervals is a function only of the widths of the intervals being
added (or subtracted). Give examples to show that this is not true
for multiplication or division.
])

#exercise(label-name: <ex:div-interval>, [
Ben Bitdiddle, an expert systems programmer, looks over Alyssa's
shoulder and comments that it is not clear what it means to
#idx("divinterval", sub: "division by zero")
divide by an interval that spans zero. Modify Alyssa's program to
check for this condition and to signal an error if it occurs.
])

#exercise(label-name: <ex:alyssa-interval-end>, [
In passing, Ben also cryptically comments: "By testing the signs of the endpoints of the intervals, it is possible to break #idx("mulinterval", sub: "more efficient version") #py("mul_interval") into nine cases, only one of which requires more than two multiplications." Rewrite this
function
using Ben's suggestion.
])

After debugging her program, Alyssa shows it to a potential user, who
complains that her program solves the wrong problem. He wants a program
that can deal with numbers represented as a center value and an additive
tolerance; for example, he wants to work with intervals such as
$3.5 plus.minus 0.15$ rather than
$[3.35, 3.65]$. Alyssa returns to her desk and
fixes this problem by supplying an alternate constructor and alternate
selectors:
#idx("makecenterwidth", decl: true)#idx("center", decl: true)#idx("width", decl: true)
#snippet(```python
def make_center_width(c, w):
    return make_interval(c - w, c + w)
def center(i):
    return (lower_bound(i) + upper_bound(i)) / 2
def width(i):
    return (upper_bound(i) - lower_bound(i)) / 2
```)

Unfortunately, most of Alyssa's users are engineers. Real engineering
situations usually involve measurements with only a small uncertainty,
measured as the ratio of the width of the interval to the midpoint of the
interval. Engineers usually specify percentage tolerances on the parameters
of devices, as in the resistor specifications given earlier.

#exercise(label-name: <ex:make-center-percent>, [
Define a constructor
#idx("makecenterpercent")
#py("make_center_percent")
that takes a center and a percentage tolerance and produces the desired
interval. You must also define a selector
#py("percent") that produces the percentage tolerance
for a given interval. The #py("center") selector is
the same as the one shown above.
])

#exercise(label-name: <ex:interval-product>, [
Show that under the assumption of small percentage tolerances there is
a simple formula for the approximate percentage tolerance of the
product of two intervals in terms of the tolerances of the factors.
You may simplify the problem by assuming that all numbers are
positive.
])

After considerable work, Alyssa P. Hacker delivers her finished
system. Several years later, after she has forgotten all about it, she
gets a frenzied call from an irate user, Lem E. Tweakit.
It seems that Lem has
noticed that the
#idx("resistance", sub: "formula for parallel resistors")
formula for parallel resistors can be written in two
algebraically equivalent ways:

$ frac(R_(1)R_(2), R_(1)+R_(2)) $

and

$ frac(1, 1/R_(1)+1/R_(2)) $

He has written the following two programs, each of which computes the
parallel-resistors formula differently:

#snippet(```python
def par1(r1, r2):
    return div_interval(mul_interval(r1, r2),
                        add_interval(r1, r2))
def par2(r1, r2):
    one = make_interval(1, 1)
    return div_interval(one,
                        add_interval(div_interval(one, r1),
                                     div_interval(one, r2)))
```)

Lem complains that Alyssa's program gives different answers for
the two ways of computing. This is a serious complaint.

#exercise(label-name: <ex:interval-compare>, [
Demonstrate that Lem is right. Investigate the behavior of the
system on a variety of arithmetic expressions. Make some intervals
$A$ and $B$,
and use them in computing the expressions $A/A$
and $A/B$. You will get the most insight by
using intervals whose width is a small percentage of the center value.
Examine the results of the computation in center-percent form (see
exercise @ex:make-center-percent).
])

#exercise(label-name: <ex:2_15>, [
Eva Lu Ator, another user, has also noticed the different intervals
computed by different but algebraically equivalent expressions. She
says that a formula to compute with intervals using Alyssa's system
will produce tighter error bounds if it can be written in such a form
that no
name
that represents an uncertain number is repeated. Thus, she says,
#py("par2") is a "better" program for
parallel resistances than #py("par1"). Is she right?
Why?
])

#exercise(label-name: <ex:2_16>, [
Explain, in general, why equivalent algebraic expressions may lead to
different answers. Can you devise an interval-arithmetic package that
does not have this shortcoming, or is this task impossible? (Warning:
This problem is very difficult.)
])

#idx("interval arithmetic")
#idx("arithmetic", sub: "on intervals")
