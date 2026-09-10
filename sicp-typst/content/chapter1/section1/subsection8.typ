// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Functions as Black-Box Abstractions], label-name: <sec:black-box>)

The function #py("sqrt")
is our first example of a process defined by a set of mutually
defined functions.
Notice that the
definition of #py("sqrt_iter")
is
#idx("recursive function", sub: "recursive function definition")
#emph[recursive]; that is, the
function
is defined in terms of itself. The idea of being able to define a
function
in terms of itself may be disturbing; it may seem unclear how such a
"circular" definition could make sense at all, much less
specify a well-defined process to be carried out by a computer. This will
be addressed more carefully in
section @sec:procedures-and-processes. But first
let's consider some other important points illustrated by the
#py("sqrt") example.

Observe that the problem of computing square roots breaks up naturally
into a number of subproblems:
#idx("program", sub: "structure of")
how to tell whether a guess is good
enough, how to improve a guess, and so on. Each of these tasks is
accomplished by a separate
function.
The entire #py("sqrt") program can be viewed as a
cluster of
functions (shown in figure @fig:sqrt-decomposition_new)
that mirrors the decomposition of the problem into subproblems.

#sicp-figure(image("/images/img_javascript/ch1-Z-G-6.svg", width: 70%), caption: [Functional decomposition of the #py("sqrt") program.], label-name: <fig:sqrt-decomposition_new>)

The importance of this
#idx("decomposition of program into parts")
decomposition strategy is not simply that one
is dividing the program into parts. After all, we could take any
large program and divide it into parts—the first ten lines, the next
ten lines, the next ten lines, and so on. Rather, it is crucial that
each
function
accomplishes an identifiable task that can be used as a module in defining
other
functions.

For example, when we define the
#py("is_good_enough") function
in terms of #py("square"), we are able to
regard the #py("square")
function
as a
#idx("black box")
"black box." We are not at that moment concerned with
#emph[how] the
function
computes its result, only with the fact #emph[that] it computes the
square. The details of how the square is computed can be suppressed,
to be considered at a later time. Indeed, as far as the
#py("is_good_enough") function
is concerned, #py("square") is not quite a
function
but rather an abstraction of a
function,
a so-called
#idx("functional abstraction") #idx("abstraction", sub: "functional") #emph[functional abstraction].
At this level of abstraction, any
function
that computes the square is equally good.

Thus, considering only the values they return, the following two
functions
squaring a number should be indistinguishable. Each takes a numerical
argument and produces the square of that number as the value.#footnote[It
is not even clear which of these
functions
is a more efficient implementation. This depends upon the hardware
available. There are machines for which the "obvious"
implementation is the less efficient one. Consider a machine that has
extensive tables of logarithms and antilogarithms stored in a very
efficient manner.]

#snippet(```python
def square(x): return x * x
```)

#snippet(```python
def square(x):
    return math_exp(double(math_log(x)))

def double(x): return x + x
```)

So a
function
should be able to suppress detail. The users of the
function
may not have written the
function
themselves, but may have obtained it from another programmer as a
black box. A user should not need to know how the
function
is implemented in order to use it.

#subheading([Local names])

#idx("local name")

One detail of a
function's
implementation that should not matter to the user of the
function
is the implementer's choice of names for the
function's parameters.
Thus, the following
functions
should not be distinguishable:

#snippet(```python
def square(x): return x * x
```)

#snippet(```python
def square(y): return y * y
```)

This principle—that the meaning of a
function
should be independent of the parameter names used by its
author—seems on the surface to be self-evident, but its
consequences are profound. The simplest consequence is that the
parameter names of a
function
must be local to the body of the
function.
For example, we used #py("square")
in the definition of #py("is_good_enough")
in our square-root
function:

#snippet(```python
def is_good_enough(guess, x):
    return abs(square(guess) - x) < 0.001
```)

The intention of the author of
#py("is_good_enough")
is to determine if the square of the first argument is within a given
tolerance of the second argument. We see that the author of
#py("is_good_enough")
used the name #py("guess") to refer to the
first argument and #py("x") to refer to the
second argument. The argument of #py("square")
is #py("guess"). If the author of
#py("square") used #py("x")
(as above) to refer to that argument, we see that the
#py("x") in
#py("is_good_enough")
must be a different #py("x") than the one
in #py("square"). Running the
function
#py("square") must not affect the value
of #py("x") that is used by
#py("is_good_enough"),
because that value of #py("x") may be needed by
#py("is_good_enough")
after #py("square") is done computing.

If the parameters were not local to the bodies of their respective
functions,
then the parameter #py("x") in
#py("square") could be confused with the parameter
#py("x") in
#py("is_good_enough"),
and the behavior of
#py("is_good_enough")
would depend upon which version of #py("square")
we used. Thus, #py("square") would not be the
black box we desired.

A
#idx("parameters", sub: "names of")
#idx("name", sub: "of a parameter")
parameter of a function
has a very special role in the
function definition,
in that it doesn't matter what name the

parameter has. Such a name is called
#idx("bound name") #idx("name", sub: "bound") #emph[bound], and we say that the function definition
#idx("bind")
#emph[binds] its
parameters.
The meaning of a
function definition is unchanged if a bound name
is consistently renamed throughout the
definition.#footnote[The
concept of consistent renaming is actually subtle and difficult to
define formally. Famous logicians have made embarrassing errors
here.]
If a
name
is not bound, we say that it is
#idx("free name") #idx("name", sub: "free")
#emph[free]. The set of
statements
for which a binding
defines
a name is called the
#idx("scope of a name") #idx("name", sub: "scope of")
#emph[scope] of that name. In a
function definition, the bound names
declared as the
#idx("parameters", sub: "scope of") #idx("scope of a name", sub: "function's parameters")
parameters of the function
have the body of the
function
as their scope.

In the
definition of #py("is_good_enough")
above,
#py("guess") and
#py("x") are
bound variables
but

#py("abs")
and #py("square") are free.
The meaning of
#py("is_good_enough")
should be independent of the names we choose for
#py("guess") and
#py("x") so long as they are distinct and
different from

#py("abs")
and #py("square"). (If we renamed
#py("guess") to
#py("abs") we would have introduced a bug by
#idx("capturing a free name") #idx("bug", sub: "capturing a free name") #idx("free name", sub: "capturing")
#emph[capturing] the variable
#py("abs").
It would have changed from free to bound.) The meaning of
#py("is_good_enough")
is not independent of the names of its free variables,
however. It surely depends upon the fact
(external to this definition)
that the name #py("abs") refers to a function
for computing the absolute value of a number.
The function #py("is_good_enough")
will compute a different function if we substitute
#py("math_cos") (the primitive cosine function)
for #py("abs") in its
definition.
#idx("local name")

#subheading([Internal definitions and block structure])

#anchor(<sec:block-structure>)

We have one kind of name isolation available to us so far:
The parameters of a function
are local to the body of the
function.
The square-root program illustrates another way in which we would like to
control the use of names.
#idx("program", sub: "structure of")
The existing program consists of separate
functions:

#snippet(```python
def sqrt(x):
    return sqrt_iter(1, x)

def sqrt_iter(guess, x):
    return (guess
            if is_good_enough(guess, x)
            else sqrt_iter(improve(guess, x), x))

def is_good_enough(guess, x):
    return abs(square(guess) - x) < 0.001

def improve(guess, x):
    return average(guess, x / guess)
```)

The problem with this program is that the only
function
that is important to users of #py("sqrt") is
#py("sqrt"). The other
functions
(#py("sqrt_iter"), #py("is_good_enough"),
and #py("improve")) only clutter up their minds.
They may not
define any other function
called
#py("is_good_enough")
as part of another program to work together
with the square-root program, because #py("sqrt")
needs it. The problem is especially severe in the construction of large
systems by many separate programmers. For example, in the construction
of a large library of numerical
functions,
many numerical functions are computed as successive approximations and
thus might have
functions
named
#py("is_good_enough")
and #py("improve") as auxiliary
functions.
We would like to localize the
subfunctions,
hiding them inside #py("sqrt") so that
#py("sqrt") could coexist with other
successive approximations, each having its own private
#py("is_good_enough") function.
To make this possible, we allow a
function
to have
#idx("block structure")
#idx("internal definition")
internal definitions that are local to that
function.
For example, in the square-root problem we can write

#snippet(```python
def sqrt(x):
    def is_good_enough(guess, x):
        return abs(square(guess) - x) < 0.001
    def improve(guess, x):
        return average(guess, x / guess)
    def sqrt_iter(guess, x):
        return (guess
                if is_good_enough(guess, x)
                else sqrt_iter(improve(guess, x), x))
    return sqrt_iter(1, x)
```)

The body of a function definition forms a #emph[block]; definitions inside it are local to the function. #idx("block") #idx("syntactic forms", sub: "block")
Such nesting of
definitions,
called #emph[block structure], is basically the right solution to the
simplest name-packaging problem. But there is a better idea lurking here.
In addition to internalizing the
definitions of the auxiliary functions,
we can simplify them. Since #py("x") is bound in the
definition
of #py("sqrt"), the
functions
#py("is_good_enough"),
#py("improve"), and
#py("sqrt_iter"), which are defined internally to
#py("sqrt"), are in the scope of
#py("x"). Thus, it is not necessary to pass
#py("x") explicitly to each of these
functions.
Instead, we allow #py("x") to be a free
#idx("internal definition", sub: "free name in") #idx("free name", sub: "in internal definition")
name
in the internal
definitions,
as shown below. Then #py("x") gets its value from
the argument with which the enclosing
function
#py("sqrt") is called. This discipline is called
#idx("lexical scoping")
#emph[lexical scoping].#footnote[Lexical scoping dictates that free
names in a function
are taken to refer to bindings made by enclosing
function definitions;
that is, they are looked up in
#idx("environment", sub: "lexical scoping and")
the environment in which the
function was defined.
We will see how this works in detail in chapter @chap:state when we
study environments and the detailed behavior of the interpreter.]
#idx("sqrt", sub: "block structured", decl: true)
#snippet(```python
def sqrt(x):
    def is_good_enough(guess):
        return abs(square(guess) - x) < 0.001
    def improve(guess):
        return average(guess, x / guess)
    def sqrt_iter(guess):
        return (guess
                if is_good_enough(guess)
                else sqrt_iter(improve(guess)))
    return sqrt_iter(1)
```)

We will use block structure extensively to help us break up large programs
into tractable pieces.#footnote[Embedded
definitions must come first in a function
body.
#idx("internal definition", sub: "position of")
The management is not responsible for the consequences of running programs
that intertwine
definition
and use; see also
footnote @foot:tdz
in section @sec:lambda.]<foot:management>
The idea of block structure originated with the programming language
#idx("Algol", sub: "block structure")
Algol 60\. It appears in most advanced programming languages and is an
important tool for helping to organize the construction of large programs.
#idx("program", sub: "structure of")
#idx("block structure")
#idx("internal definition")
