// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([The Elements of Programming], label-name: <sec:elements-of-programming>)

#idx("programming", sub: "elements of")

A powerful programming language is more than just a means for
instructing a computer to perform tasks. The language also serves as
a framework within which we organize our ideas about processes. Thus,
when we describe a language, we should pay particular attention to the
means that the language provides for combining simple ideas to form
more complex ideas. Every powerful language has three mechanisms for
accomplishing this:

- #strong[primitive expressions], #idx("primitive expression") which represent the simplest entities the language is concerned with,
- #strong[means of combination], by #idx("means of combination") #idx("combination, means of") which compound elements are built from simpler ones, and
- #strong[means of abstraction], #idx("means of abstraction") by which compound elements can be named and manipulated as units.

In programming, we deal with two kinds of elements:
functions
and
#idx("data")
data. (Later we will discover that they are really not so distinct.)
Informally, data is "stuff" that we want to manipulate, and
functions
are descriptions of the rules for manipulating the data.
Thus, any powerful programming language should be able to describe
primitive data and primitive
functions
and should have methods for
combining and abstracting
functions
and data.

In this chapter we will deal only with simple
#idx("numerical data")
#idx("data", sub: "numerical")
numerical data so that
we can focus on the rules for building
functions.#footnote[The

characterization of numbers as "simple data" is a barefaced
bluff. In fact, the treatment of numbers is one of the trickiest and most
confusing aspects of any programming language. Some typical issues
involved are these:
#idx("integer(s)")
#idx("real number")
#idx("number(s)", sub: "integer vs. real number")
Some computer systems distinguish #emph[integers], such as 2,
from #emph[real numbers], such as 2.71. Is the real number
2.00 different from the integer 2? Are the arithmetic operations
used for integers the same as the operations used for real numbers?
Does 6 divided by 2 produce 3, or 3.0? How large a number can we
represent? How many decimal places of accuracy can we represent?
Is the range of integers the same as the range of real numbers?
#idx("numerical analysis")
#idx("roundoff error")
#idx("truncation error")
Above and beyond these questions, of course, lies a collection of
issues concerning roundoff and truncation errors—the
entire science of numerical analysis. Since our focus in this
book is on large-scale program design rather than on numerical
techniques, we are going to ignore these problems. The numerical
examples in this chapter will exhibit the usual roundoff behavior
that one observes when using arithmetic operations that preserve
a limited number of decimal places of accuracy in noninteger
operations.]<foot:number-representation>
In later chapters we will see that
these same rules allow us to build
functions
to manipulate compound data as well.
#idx("programming", sub: "elements of")

#include "../../chapter1/section1/subsection1.typ"

#include "../../chapter1/section1/subsection2.typ"

#include "../../chapter1/section1/subsection3.typ"

#include "../../chapter1/section1/subsection4.typ"

#include "../../chapter1/section1/subsection5.typ"

#include "../../chapter1/section1/subsection6.typ"

#include "../../chapter1/section1/subsection7.typ"

#include "../../chapter1/section1/subsection8.typ"
