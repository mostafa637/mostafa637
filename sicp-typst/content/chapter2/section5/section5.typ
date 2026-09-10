// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Systems with Generic Operations], label-name: <sec:generic-operators>)

In the previous section, we saw how to design systems in which data
objects can be represented in more than one way. The key idea is to
link the code that specifies the data operations to the several
representations by means of generic interface
functions.
Now we will see how to use this same idea not only to define operations
that are generic over different representations but also to define
operations that are
#idx("arithmetic", sub: "generic")
generic over different kinds of arguments. We have
already seen several different packages of arithmetic operations: the
primitive arithmetic (#py("+"),
#py("-"), #py("*"),
#py("/")) built into our language, the
rational-number arithmetic
(#py("add_rat"),
#py("sub_rat"),
#py("mul_rat"),
#py("div_rat"))
of section @sec:rationals, and the complex-number
arithmetic that we implemented in
section @sec:data-directed. We will now use
data-directed techniques to construct a package of arithmetic operations
that incorporates all the arithmetic packages we have already constructed.

Figure @fig:generic-system
shows the structure of the system we
shall build. Notice the
#idx("abstraction barriers", sub: "in generic arithmetic system")
abstraction barriers. From the perspective
of someone using "numbers," there is a single
function
#py("add") that operates on whatever numbers are
supplied.
The function #py("add")
is part of a generic interface that allows the separate ordinary-arithmetic,
rational-arithmetic, and complex-arithmetic packages to be accessed
uniformly by programs that use numbers. Any individual arithmetic package
(such as the complex package) may itself be accessed through generic
functions
(such as
#py("add_complex"))
that combine packages designed for different representations (such as
rectangular and polar). Moreover, the structure of the system is additive,
so that one can design the individual arithmetic packages separately and
combine them to produce a generic arithmetic system.
#idx("message passing")

#sicp-figure(image("/images/img_javascript/ch2-Z-G-64.svg", width: 70%), caption: [Generic #idx("generic arithmetic operations", sub: "structure of system") arithmetic system.], label-name: <fig:generic-system>)

#include "../../chapter2/section5/subsection1.typ"

#include "../../chapter2/section5/subsection2.typ"

#include "../../chapter2/section5/subsection3.typ"
