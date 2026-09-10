// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([The Environment Model of Evaluation], label-name: <sec:environment-model>)

#idx("environment model of evaluation")

When we introduced compound
functions
in chapter @chap:fun, we used the
#idx("substitution model of function application")
substitution model of evaluation
(section @sec:substitution-model) to define what is
meant by applying a
function
to arguments:

- To apply a compound function to arguments, evaluate the return expression of the function (more generally, the body) with each parameter replaced by the corresponding argument.

Once we admit
reassignment
into our programming language, such a
definition is no longer adequate. In particular,
section @sec:costs-of-assignment argued that, in the
presence of
reassignment,
a name cannot be considered to be merely representing a value. Rather, a name must somehow designate a "place" in which values can be stored.
In our new model of
evaluation, these places will be maintained in structures called
#idx("environment")
#emph[environments].

An environment is a sequence of
#idx("frame (environment model)")
#emph[frames]. Each frame is a table (possibly empty) of
#idx("binding")
#emph[bindings], which associate
variable names
with their corresponding
values.
(A single frame may contain at most one binding for any name.)
Each frame also has a pointer to its
#idx("enclosing environment")
#idx("environment", sub: "enclosing")
#emph[enclosing environment], unless, for the purposes of discussion, the
frame is considered to be
#idx("global frame")
#idx("frame (environment model)", sub: "global")
#emph[global]. The
#idx("name", sub: "value of") #emph[value of a name]
with respect to an environment is the value given by the binding of
the
name
in the first frame in the environment that contains a
binding for that
name.
If no frame in the sequence specifies a
binding for the
name,
then the
name
is said to be
#idx("unbound name")
#idx("name", sub: "unbound")
#emph[unbound] in the environment.

#sicp-figure(image("/images/img_javascript/ch3-Z-G-2.svg", width: 70%), caption: [A simple #idx("environment model of evaluation", sub: "environment structure") environment structure.], label-name: <fig:environment-structure>)

Figure @fig:environment-structure
shows a simple environment
structure consisting of three frames, labeled I, II, and III. In the
diagram, A, B, C, and D are pointers to environments. C and D point
to the same environment. The
names
#py("z") and
#py("x") are bound in frame II, while
#py("y") and #py("x") are bound
in frame I. The value of #py("x") in environment D
is 3. The value of #py("x") with respect to
environment B is also 3. This is determined as follows: We examine the
first frame in the sequence (frame III) and do not find a binding for
#py("x"), so we proceed to the enclosing environment
D and find the binding in frame I. On the other hand, the value of
#py("x") in environment A is 7, because the first
frame in the sequence (frame II) contains a binding of
#py("x") to 7. With respect to environment A, the
binding of #py("x") to 7 in frame II is said to
#idx("shadow a binding")
#emph[shadow] the binding of #py("x") to 3 in
frame I.

The environment is crucial to the evaluation process, because it determines
the context in which an expression should be evaluated. Indeed, one could
say that expressions in a programming language do not, in themselves, have
any meaning. Rather, an expression acquires a meaning only with respect to
some environment in which it is evaluated.

Even the interpretation of an expression as straightforward as
#py("display(1)") depends on an
understanding that one is operating in a context in which the name
#py("display") refers to the primitive function
that displays a value.

Thus, in our model of evaluation we will always speak of evaluating an
expression with respect to some environment. To describe interactions with
the	interpreter, we will suppose that there is a
#idx("global environment")
global environment, consisting of a single frame (with no enclosing
environment) that includes values for the
names
associated with the
primitive
functions.
For example, the idea that
#py("display") is the name for the primitive display function is captured by saying that the name #py("display")
is bound in the global environment to the primitive
display function.

Before we evaluate a program, we extend the global
environment with a new frame, the
#emph[program frame], resulting in the
#idx("program environment")
#emph[program environment]. We will add
the names that are declared at the top level of the
program, outside of any block, to this frame. The given program
is then	evaluated with respect to the program environment.

#include "../../chapter3/section2/subsection1.typ"

#include "../../chapter3/section2/subsection2.typ"

#include "../../chapter3/section2/subsection3.typ"

#include "../../chapter3/section2/subsection4.typ"

#include "../../chapter3/section2/subsection5.typ"
