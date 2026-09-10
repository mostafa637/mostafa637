// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Generic Arithmetic Operations], label-name: <sec:generic-arithmetic-operators>)

#idx("generic arithmetic operations")

The task of designing generic arithmetic operations is analogous to that of
designing the generic complex-number operations. We would like, for
instance, to have a generic addition
function
#py("add") that acts like ordinary primitive addition
#py("+") on ordinary numbers, like
#py("add_rat")
on rational numbers, and like
#py("add_complex")
on complex numbers. We can implement #py("add"), and
the other generic arithmetic operations, by following the same strategy we
used in section @sec:data-directed to implement the
generic selectors for complex numbers. We will attach a type tag to each
kind of number and cause the generic
function
to dispatch to an appropriate package according to the data type of its
arguments.

The generic arithmetic
functions
are defined as follows:
#idx("add (generic)", decl: true)#idx("sub (generic)", decl: true)#idx("mul (generic)", decl: true)#idx("div (generic)", decl: true)
#snippet(```python
def add(x, y): return apply_generic("add", llist(x, y))

def sub(x, y): return apply_generic("sub", llist(x, y))

def mul(x, y): return apply_generic("mul", llist(x, y))

def div(x, y): return apply_generic("div", llist(x, y))
```)

We begin
by installing a package for handling
#idx("number(s)", sub: "in generic arithmetic system")
#idx("ordinary numbers (in generic arithmetic system)")
#emph[ordinary] numbers,
that is, the primitive numbers of our language. We

tag these
with the
string #py("\"python_number\"").
The arithmetic operations in this package are the primitive arithmetic
functions
(so there is no need to define extra
functions
to handle the untagged numbers). Since these operations each take two
arguments, they are installed in the table keyed by the
linked list
#py("llist(\"python_number\", \"python_number\")"):
#idx("package", sub: "Python-number")#idx("pythonnumber package")#idx("installpythonnumberpackage", decl: true)
#snippet(```python
def install_python_number_package():
    def tag(x):
        return attach_tag("python_number", x)
    put("add", llist("python_number", "python_number"),
        lambda x, y: tag(x + y))
    put("sub", llist("python_number", "python_number"),
        lambda x, y: tag(x - y))
    put("mul", llist("python_number", "python_number"),
        lambda x, y: tag(x * y))
    put("div", llist("python_number", "python_number"),
        lambda x, y: tag(x / y))
    put("make", "python_number",
        lambda x: tag(x))
    return "done"
```)

Users of the
Python-number package
will create (tagged) ordinary numbers by means of the
function:

#idx("makepythonnumber", decl: true)
#snippet(```python
def make_python_number(n):
    return get("make", "python_number")(n)
```)

Now that the framework of the generic arithmetic system is in place,
we can readily include new kinds of numbers. Here is a package that
performs rational arithmetic. Notice that, as a benefit of
additivity, we can use without modification the rational-number code
from section @sec:rationals as the internal
functions
in the package:
#idx("package", sub: "rational-number")#idx("rational package")#idx("rational-number arithmetic", sub: "interfaced to generic arithmetic system")#idx("installrationalpackage", decl: true)#idx("makerational", decl: true)
#snippet(```python
def install_rational_package():
    # internal functions
    def numer(x): return head(x)
    def denom(x): return tail(x)
    def make_rat(n, d):
        g = gcd(n, d)
        return pair(n // g, d // g)
    def add_rat(x, y):
        return make_rat(numer(x) * denom(y) + numer(y) * denom(x),
                        denom(x) * denom(y))
    def sub_rat(x, y):
        return make_rat(numer(x) * denom(y) - numer(y) * denom(x),
                        denom(x) * denom(y))
    def mul_rat(x, y):
        return make_rat(numer(x) * numer(y),
                        denom(x) * denom(y))
    def div_rat(x, y):
        return make_rat(numer(x) * denom(y),
                        denom(x) * numer(y))
    # interface to rest of the system
    def tag(x):
        return attach_tag("rational", x)
    put("add", llist("rational", "rational"),
        lambda x, y: tag(add_rat(x, y)))
    put("sub", llist("rational", "rational"),
        lambda x, y: tag(sub_rat(x, y)))
    put("mul", llist("rational", "rational"),
        lambda x, y: tag(mul_rat(x, y)))
    put("div", llist("rational", "rational"),
        lambda x, y: tag(div_rat(x, y)))
    put("make", "rational",
        lambda n, d: tag(make_rat(n, d)))
    return "done"

def make_rational(n, d):
    return get("make", "rational")(n, d)
```)

We can install a similar package to handle complex numbers, using the tag
#py("\"complex\"").
In creating the package, we extract from the table the operations
#py("make_from_real_imag")
and
#py("make_from_mag_ang")
that were defined by the rectangular and polar packages.
#idx("additivity")
Additivity permits us to use, as the internal operations, the same
#py("add_complex"),
#py("sub_complex"),
#py("mul_complex"),
and
#py("div_complex")
functions
from section @sec:representations-complex-numbers.
#idx("package", sub: "complex-number")#idx("complex package")#idx("complex-number arithmetic", sub: "interfaced to generic arithmetic system")#idx("installcomplexpackage", decl: true)
#snippet(```python
def install_complex_package():
    # imported functions from rectangular and polar packages
    def make_from_real_imag(x, y):
        return get("make_from_real_imag", "rectangular")(x, y)
    def make_from_mag_ang(r, a):
        return get("make_from_mag_ang", "polar")(r, a)
    # internal functions
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
    # interface to rest of the system
    def tag(z): return attach_tag("complex", z)
    put("add", llist("complex", "complex"),
        lambda z1, z2: tag(add_complex(z1, z2)))
    put("sub", llist("complex", "complex"),
        lambda z1, z2: tag(sub_complex(z1, z2)))
    put("mul", llist("complex", "complex"),
        lambda z1, z2: tag(mul_complex(z1, z2)))
    put("div", llist("complex", "complex"),
        lambda z1, z2: tag(div_complex(z1, z2)))
    put("make_from_real_imag", "complex",
        lambda x, y: tag(make_from_real_imag(x, y)))
    put("make_from_mag_ang", "complex",
        lambda r, a: tag(make_from_mag_ang(r, a)))
    return "done"
```)

Programs outside the complex-number package can construct complex
numbers either from real and imaginary parts or from magnitudes and
angles. Notice how the underlying
functions,
originally defined in the rectangular and polar packages, are exported to
the complex package, and exported from there to the outside world.

#idx("makecomplexfromrealimag", decl: true)#idx("makecomplexfrommagang", decl: true)
#snippet(```python
def make_complex_from_real_imag(x, y):
    return get("make_from_real_imag", "complex")(x, y)
def make_complex_from_mag_ang(r, a):
    return get("make_from_mag_ang", "complex")(r, a)
```)

What we have here is a
#idx("type tag", sub: "two-level")
two-level tag system. A typical complex number,
such as $3+4i$ in rectangular form, would be
represented as shown in
figure @fig:complex-number-structure.
The outer tag
(#py("\"complex\""))
is used to direct the number to the complex package. Once within the
complex package, the next tag
(#py("\"rectangular\""))
is used to direct the number to the rectangular package. In a large and
complicated system there might be many levels, each interfaced with the
next by means of generic operations. As a data object is passed
"downward," the outer tag that is used to direct it to the
appropriate package is stripped off (by applying
#py("contents")) and the next level of tag (if any)
becomes visible to be used for further dispatching.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-65.svg", width: 70%), caption: [Representation of $3+4i$ in rectangular form.], label-name: <fig:complex-number-structure>)

In the above packages, we used
#py("add_rat"),
#py("add_complex"),
and the other arithmetic
functions
exactly as originally written. Once these declarations are internal to
different installation
functions,
however, they no longer need names that are distinct from each other:
we could simply name them #py("add"),
#py("sub"), #py("mul"), and
#py("div") in both packages.

#exercise(label-name: <ex:2_77>, [
Louis Reasoner tries to evaluate the expression
#py("magnitude(z)")
where
#py("z") is the object shown in
figure @fig:complex-number-structure.
To his surprise, instead of the answer $5$
he gets an error message from
#py("apply_generic"),
saying there is no method for the operation
#py("magnitude") on the types
#py("llist(\"complex\")").
He shows this interaction to Alyssa P. Hacker, who says "The problem is that the complex-number selectors were never defined for #py("\"complex\"") numbers, just for #py("\"polar\"") and #py("\"rectangular\"") numbers. All you have to do to make this work is add the following to the #py("complex") package:"

#snippet(```python
put("real_part", llist("complex"), real_part)
put("imag_part", llist("complex"), imag_part)
put("magnitude", llist("complex"), magnitude)
print(put("angle", llist("complex"), angle))
```)

Describe in detail why this works. As an example, trace through all the
functions
called in evaluating the expression
#py("magnitude(z)")
where #py("z") is the object shown in
figure @fig:complex-number-structure.
In particular, how many times is
#py("apply_generic")
invoked? What
function
is dispatched to in each case?
])

#exercise(label-name: <ex:internal-type-system>, [
The internal
#idx("Python", sub: "internal type system")
#idx("data types", sub: "in Python")
#idx("isnumber (primitive function)", sub: "data types and")
#idx("isstring (primitive function)", sub: "data types and")
#idx("attachtag", sub: "using Python data types")
#idx("typetag", sub: "using Python data types")
#idx("contents", sub: "using Python data types")
functions
in the
#py("python_number")
package are essentially nothing more than calls to the primitive
functions
#py("+"), #py("-"), etc. It
was not possible to use the primitives of the language directly because our
type-tag system requires that each data object have a type attached to it.
In fact, however, all
Python
implementations do have a type system, which they use internally. Primitive
predicates such as
#py("is_string")
and
#py("is_number")
determine whether data objects have particular types. Modify the
definitions of
#py("type_tag"),
#py("contents"), and
#py("attach_tag")
from section @sec:manifest-types so that our generic
system takes advantage of
Python's
internal type system. That is to say, the system should work as before
except that ordinary numbers should be represented simply as
Python
numbers rather than as pairs whose
#py("head")
is the
string #py("\"python_number\"").
])

#exercise(label-name: <ex:equ->, [
Define a generic equality predicate
#idx("isequal (generic predicate)")
#idx("equality", sub: "in generic arithmetic system")
#py("is_equal")
that tests the equality of two numbers, and install it in the generic
arithmetic package. This operation should work for ordinary numbers,
rational numbers, and complex numbers.
])

#exercise(label-name: <ex:-zero->, [
Define a generic predicate
#idx("isequaltozero (generic)")
#idx("zero test (generic)")
#py("is_equal_to_zero")
that tests if its argument is zero, and install it in the generic
arithmetic package. This operation should work for ordinary numbers,
rational numbers, and complex numbers.
])

#idx("generic arithmetic operations")
