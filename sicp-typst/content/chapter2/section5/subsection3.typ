// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Example: Symbolic Algebra], label-name: <sec:symbolic-algebra>)

#idx("symbolic algebra")

The manipulation of symbolic algebraic expressions is a complex
process that illustrates many of the hardest problems that occur in
the design of large-scale systems. An
#idx("algebraic expression")
algebraic expression, in
general, can be viewed as a hierarchical structure, a tree of
operators applied to operands. We can construct algebraic expressions
by starting with a set of primitive objects, such as constants and
variables, and combining these by means of algebraic operators, such
as addition and multiplication. As in other languages, we form
abstractions that enable us to refer to compound objects in simple
terms. Typical abstractions in symbolic algebra are ideas such as
linear combination, polynomial, rational function, or trigonometric
function. We can regard these as compound "types," which are
often useful for directing the processing of expressions. For example, we
could describe the expression

$ x^(2) thin sin (y^(2)+1)+x thin cos 2y+ cos (y^(3) -2y^(2)) $

as a polynomial in $x$ with coefficients that
are trigonometric functions of polynomials in
$y$ whose coefficients are integers.

We will not attempt to develop a complete algebraic-manipulation
system here. Such systems are exceedingly complex programs, embodying
deep algebraic knowledge and elegant algorithms. What we will do is
look at a simple but important part of algebraic manipulation: the
arithmetic of polynomials. We will illustrate the kinds of decisions
the designer of such a system faces, and how to apply the ideas of
abstract data and generic operations to help organize this effort.

#subheading([Arithmetic on polynomials])

#idx("polynomial(s)")
#idx("polynomial arithmetic")

Our first task in designing a system for performing arithmetic on
polynomials is to decide just what a polynomial is. Polynomials are
normally defined relative to certain variables (the
#idx("indeterminate of a polynomial")
#idx("polynomial(s)", sub: "indeterminate of")
#emph[indeterminates] of the polynomial). For simplicity, we will
restrict ourselves to polynomials having just one indeterminate
#idx("univariate polynomial")
#idx("polynomial(s)", sub: "univariate")
#emph[(univariate polynomials]).#footnote[On the other hand, we will
allow polynomials whose coefficients are themselves polynomials in other
variables. This will give us essentially the same representational
power as a full multivariate system, although it does lead to coercion
problems, as discussed below.] We will define a polynomial to
be a sum of terms, each of which is either a coefficient, a power of the
indeterminate, or a product of a coefficient and a power of the
indeterminate. A coefficient is defined as an algebraic expression
that is not dependent upon the indeterminate of the polynomial. For
example,

$ 5x^(2) +3x +7 $

is a simple polynomial in $x$, and

$ (y^(2) +1)x^(3) +(2y)x+1 $

is a polynomial in $x$ whose coefficients are
polynomials in $y$.

Already we are skirting some thorny issues. Is the first of these
polynomials the same as the polynomial
$5y^(2) +3y +7$, or not? A reasonable answer
might be "yes, if we are considering a polynomial purely as a mathematical function, but no, if we are considering a polynomial to be a syntactic form." The second polynomial is algebraically equivalent
to a polynomial in $y$ whose coefficients are
polynomials in $x$. Should our system recognize
this, or not? Furthermore, there are other ways to represent a
polynomial—for example, as a product of factors, or (for a
univariate polynomial) as the set of roots, or as a listing of the values
of the polynomial at a specified set of points.#footnote[For univariate
polynomials, giving the value of a polynomial at a given set of points can
be a particularly good representation. This makes polynomial arithmetic
extremely simple. To obtain, for example, the sum of two polynomials
represented in this way, we need only add the values of the
polynomials at corresponding points. To transform back to a more
familiar representation, we can use the
#idx("Lagrange interpolation formula")
Lagrange interpolation formula, which shows how to recover the coefficients
of a polynomial of degree $n$ given the values
of the polynomial at $n+1$ points.]
We can finesse these questions by deciding that in our
algebraic-manipulation system a "polynomial" will be a
particular syntactic form, not its underlying mathematical meaning.

Now we must consider how to go about doing arithmetic on polynomials.
In this simple system, we will consider only addition and
multiplication. Moreover, we will insist that two polynomials to be
combined must have the same indeterminate.

We will approach the design of our system by following the familiar
discipline of data abstraction. We will represent polynomials using a
data structure called a
#idx("poly")
#emph[poly], which consists of a variable and a
#idx("term list of polynomial")
collection of terms. We assume that we have selectors
#py("variable") and
#py("term_list")
that extract those parts from a poly and a constructor
#py("make_poly")
that assembles a poly from a given variable and a term list.
A variable will be just a
string,
so we can use the
#idx("issamevariable")
#py("is_same_variable")
function
of section @sec:symbolic-differentiation to compare
variables.
The following
functions
define
#idx("polynomial arithmetic", sub: "addition")
#idx("polynomial arithmetic", sub: "multiplication")
addition and multiplication of polys:
#idx("addpoly", decl: true)#idx("mulpoly", decl: true)
#snippet(```python
def add_poly(p1, p2):
    return (make_poly(variable(p1),
                      add_terms(term_list(p1), term_list(p2)))
            if is_same_variable(variable(p1), variable(p2))
            else error("polys not in same var -- add_poly", llist(p1, p2)))
def mul_poly(p1, p2):
    return (make_poly(variable(p1),
                      mul_terms(term_list(p1), term_list(p2)))
            if is_same_variable(variable(p1), variable(p2))
            else error("polys not in same var -- mul_poly", llist(p1, p2)))
```)

To incorporate polynomials into our generic arithmetic system, we need
to supply them with type tags. We'll use the tag
#py("\"polynomial\""),
and install appropriate operations on tagged polynomials in the operation
table.

We'll embed all our code in an installation function
for the polynomial package,
similar to the installation functions in
section @sec:generic-arithmetic-operators:
#idx("package", sub: "polynomial")#idx("polynomial package")#idx("polynomial arithmetic", sub: "interfaced to generic arithmetic system")#idx("installpolynomialpackage", decl: true)#idx("makepoly", decl: true)#idx("variable", decl: true)#idx("termlist", decl: true)
#syntax("
def install_polynomial_package():
    # internal functions
    # representation of poly
    def make_poly(variable, term_list):
        return pair(variable, term_list)
    def variable(p): return head(p)
    def term_list(p): return tail(p)
    ", metaphrase[functions #py("is_same_variable") and #py("is_variable") from section 2.3.2], "

    # representation of terms and term lists
    ", metaphrase[functions #py("adjoin_term...coeff") from text below], "

    def add_poly(p1, p2): ...
    ", metaphrase[functions used by #py("add_poly")], "
    def mul_poly(p1, p2): ...
    ", metaphrase[functions used by #py("mul_poly")], "

    # interface to rest of the system
    def tag(p): return attach_tag(\"polynomial\", p)
    put(\"add\", llist(\"polynomial\", \"polynomial\"),
        lambda p1, p2: tag(add_poly(p1, p2)))
    put(\"mul\", llist(\"polynomial\", \"polynomial\"),
        lambda p1, p2: tag(mul_poly(p1, p2)))
    put(\"make\", \"polynomial\",
        lambda variable, terms: tag(make_poly(variable, terms)))
    return \"done\"
	    ")

Polynomial addition is performed termwise. Terms of the same order
(i.e., with the same power of the indeterminate) must be combined.
This is done by forming a new term of the same order whose coefficient
is the sum of the coefficients of the addends. Terms in one addend
for which there are no terms of the same order in the other addend are
simply accumulated into the sum polynomial being constructed.

In order to manipulate term lists, we will assume that we have a
constructor
#idx("theemptytermlist")
#py("the_empty_termlist")
that returns an empty term list and a constructor
#idx("adjointerm")
#py("adjoin_term")
that adjoins a new term to a term list. We will also assume that we have
a predicate
#idx("isemptytermlist")
#py("is_empty_termlist")
that tells if a given term list is empty, a selector
#idx("firstterm")
#py("first_term")
that extracts the highest-order term from a term list, and a selector
#idx("restterms")
#py("rest_terms")
that returns all but the highest-order term. To manipulate terms,
we will suppose that we have a constructor
#idx("maketerm")
#py("make_term")
that constructs a term with given order and coefficient, and selectors
#idx("order")
#py("order") and
#idx("coeff")
#py("coeff") that return, respectively, the order
and the coefficient of the term. These operations allow us to consider
both terms and term lists as data abstractions, whose concrete
representations we can worry about separately.

Here is the
function
that constructs the term list for the sum of two
polynomials;#footnote[This operation is very much like the ordered
#py("union_set")
operation we developed in exercise @ex:union-set.
In fact, if we think of the terms of the polynomial as a set ordered
according to the power of the indeterminate, then the program that
produces the term list for a sum is almost identical to
#py("union_set").]

note that we slightly extend the syntax of
#idx("conditional statement", sub: "conditional instead of alternative block")
conditional statements described in
section @sec:lambda by admitting another conditional
statement in place of the block following
#py("else"):

#idx("addterms", decl: true)
#snippet(```python
def add_terms(L1, L2):
    if is_empty_termlist(L1):
        return L2
    elif is_empty_termlist(L2):
        return L1
    else:
        t1 = first_term(L1)
        t2 = first_term(L2)
        return (adjoin_term(t1, add_terms(rest_terms(L1), L2))
                if order(t1) > order(t2)
                else adjoin_term(t2, add_terms(L1, rest_terms(L2)))
                if order(t1) < order(t2)
                else adjoin_term(make_term(order(t1),
                                           add(coeff(t1), coeff(t2))),
                                 add_terms(rest_terms(L1),
                                           rest_terms(L2))))
```)

The most important point to note here is that we used the generic addition
function
#idx("add (generic)", sub: "used for polynomial coefficients")
#py("add") to add together the coefficients of the
terms being combined. This has powerful consequences, as we will see below.

In order to multiply two term lists, we multiply each term of the first
linked list by all the terms of the other linked list, repeatedly using
#py("mul_term_by_all_terms"),
which multiplies a given term by all terms in a given term list. The
resulting term lists (one for each term of the first list) are accumulated
into a sum. Multiplying two terms forms a term whose order is the sum of
the orders of the factors and whose coefficient is the product of the
coefficients of the factors:
#idx("multerms", decl: true)
#snippet(```python
def mul_terms(L1, L2):
    return (the_empty_termlist
            if is_empty_termlist(L1)
            else add_terms(mul_term_by_all_terms(
                                  first_term(L1), L2),
                           mul_terms(rest_terms(L1), L2)))
def mul_term_by_all_terms(t1, L):
    if is_empty_termlist(L):
        return the_empty_termlist
    else:
        t2 = first_term(L)
        return adjoin_term(
                   make_term(order(t1) + order(t2),
                             mul(coeff(t1), coeff(t2))),
                   mul_term_by_all_terms(t1, rest_terms(L)))
```)

This is really all there is to polynomial addition and multiplication.
Notice that, since we operate on terms using the generic
functions
#idx("add (generic)", sub: "used for polynomial coefficients")
#idx("mul (generic)", sub: "used for polynomial coefficients")
#py("add") and #py("mul"),
our polynomial package is automatically able to handle any type of
coefficient that is known about by the generic arithmetic package.
If we include a
#idx("coercion", sub: "in polynomial arithmetic")
coercion mechanism such as one of those discussed in
section @sec:combining-data-of-different-types,
then we also are automatically able to handle operations on
polynomials of different coefficient types, such as

$ (lr([ 3x^(2) +(2+3i)x+7 ]) dot.op lr([ x^(4) +frac(2, 3)x^(2) +(5+3i) ])) $

Because we installed the polynomial addition and multiplication
functions
#py("add_poly")
and
#py("mul_poly")
in the generic arithmetic system as the #py("add")
and #py("mul") operations for type
#py("polynomial"), our system is also automatically
able to handle polynomial operations such as

$ (lr([ (y+1)x^(2) +(y^(2) +1)x+(y-1) ]) dot.op lr([ (y-2)x+(y^(3) +7) ])) $

The reason is that when the system tries to combine coefficients, it
will dispatch through #py("add") and
#py("mul"). Since the coefficients are themselves
polynomials (in $y$), these will be combined
using
#py("add_poly")
and
#py("mul_poly").
The result is a kind of
#idx("data-directed recursion")
#idx("recursion", sub: "data-directed")
"data-directed recursion" in which, for example, a call to
#py("mul_poly")
will result in recursive calls to
#py("mul_poly")
in order to multiply the coefficients. If the coefficients of the
coefficients were themselves polynomials (as might be used to represent
polynomials in three variables), the data direction would ensure that the
system would follow through another level of recursive calls, and so on
through as many levels as the structure of the data dictates.#footnote[To
make this work completely smoothly, we should also add to our generic
arithmetic system the ability to coerce a "number" to a
polynomial by regarding it as a polynomial of degree zero whose coefficient
is the number. This is necessary if we are going to perform operations
such as

$ (lr([ x^(2) +(y+1)x+5 ])+ lr([ x^(2) +2x+1 ])) $

which requires adding the coefficient $y+1$ to
the coefficient 2.]
#idx("polynomial arithmetic", sub: "addition")
#idx("polynomial arithmetic", sub: "multiplication")

#subheading([Representing term lists])

#idx("term list of polynomial", sub: "representing")

Finally, we must confront the job of implementing a good
representation for term lists. A term list is, in effect, a set of
coefficients keyed by the order of the term. Hence, any of the
methods for representing sets, as discussed in
section @sec:representing-sets, can be applied to this
task. On the other hand, our
functions
#py("add_terms") and #py("mul_terms")
always access term lists sequentially from highest to lowest order.
Thus, we will use some kind of ordered linked-list representation.

How should we structure the list that represents a term list? One
consideration is the "density" of the polynomials we intend
to manipulate. A polynomial is said to be
#idx("dense polynomial")
#idx("polynomial(s)", sub: "dense")
#emph[dense] if it has nonzero coefficients in terms of most orders.
If it has many zero terms it is said to be
#idx("sparse polynomial")
#idx("polynomial(s)", sub: "sparse")
#emph[sparse]. For example,

$ A: x^(5) +2x^(4) +3x^(2) -2x -5 $

is a dense polynomial, whereas

$ B: x^(100) +2x^(2) +1 $

is sparse.

The term list of a dense polynomial is most efficiently represented as a linked list of the coefficients.
For example,
the polynomial
$A$ above would be nicely represented as
#py("llist(1, 2, 0, 3, -2, -5)").
The order of a term in this representation is the length of the sublist
beginning with that term's coefficient, decremented by 1.#footnote[In
these polynomial examples, we assume that we have implemented the generic
arithmetic system using the type mechanism suggested in
exercise @ex:internal-type-system. Thus, coefficients
that are ordinary numbers will be represented as the numbers themselves
rather than as pairs whose
#py("head")
is the
string #py("\"python_number\"").]
This would be a terrible representation for a sparse polynomial such as
$B$: There would be a giant linked list of zeros
punctuated by a few lonely nonzero terms. A more reasonable representation
of the term list of a sparse polynomial is as a linked list of the nonzero terms,
where each term is a linked list containing the order of the term and the
coefficient for that order. In such a scheme, polynomial
$B$ is efficiently represented as
#py("llist(llist(100, 1), llist(2, 2), llist(0, 1))").
As most polynomial manipulations are performed on sparse polynomials, we
will use this method. We will assume that term lists are represented as
linked lists of terms, arranged from highest-order to lowest-order term. Once we
have made this decision, implementing the selectors and constructors for
terms and term lists is straightforward:#footnote[Although we are assuming
that term lists are ordered, we have implemented
#py("adjoin_term")
to simply
adjoin the new term to the front of the existing term list.
We can get away with this so
long as we guarantee that the
functions
(such as
#py("add_terms"))
that use
#py("adjoin_term")
always call it with a higher-order term than appears in the linked list. If we
did not want to make such a guarantee, we could have implemented
#py("adjoin_term")
to be similar to the
#py("adjoin_set")
constructor for the ordered linked-list
representation of sets
(exercise @ex:adjoin-set).]<foot:adjoin-term>
#idx("adjointerm", decl: true)#idx("theemptytermlist", decl: true)#idx("firstterm", decl: true)#idx("restterms", decl: true)#idx("isemptytermlist", decl: true)#idx("maketerm", decl: true)#idx("order", decl: true)#idx("coeff", decl: true)
#snippet(```python
def adjoin_term(term, term_list):
    return (term_list
            if is_equal_to_zero(coeff(term))
            else pair(term, term_list))

the_empty_termlist = None

def first_term(term_list): return head(term_list)

def rest_terms(term_list): return tail(term_list)

def is_empty_termlist(term_list): return is_none(term_list)

def make_term(order, coeff): return llist(order, coeff)

def order(term): return head(term)

def coeff(term): return head(tail(term))
```)

where
#py("is_equal_to_zero")
is as defined in exercise @ex:-zero-. (See also
exercise @ex:adjoin-term below.)

Users of the polynomial package will create (tagged) polynomials by means
of the
function:

#idx("makepolynomial", decl: true)
#snippet(```python
def make_polynomial(variable, terms):
    return get("make", "polynomial")(variable, terms)
```)

#exercise(label-name: <ex:adjoin-term>, [
Install
#idx("isequaltozero (generic)", sub: "for polynomials")
#idx("zero test (generic)", sub: "for polynomials")
#py("is_equal_to_zero")
for polynomials in the generic arithmetic package. This will allow
#py("adjoin_term")
to work for polynomials with coefficients that are themselves polynomials.
])

#exercise(label-name: <ex:sub-poly>, [
Extend the polynomial system to include
#idx("polynomial arithmetic", sub: "subtraction")
subtraction of polynomials.
(Hint: You may find it helpful to define a generic negation operation.)
])

#exercise(label-name: <ex:2_89>, [
Declare functions
that implement the term-list representation described above as
appropriate for dense polynomials.
])

#exercise(label-name: <ex:2_90>, [
Suppose we want to have a polynomial system that is efficient for both
sparse and dense polynomials. One way to do this is to allow both
kinds of term-list representations in our system. The situation is
analogous to the complex-number example of
section @sec:multiple-reps, where we allowed both
rectangular and polar representations. To do this we must distinguish
different types of term lists and make the operations on term lists
generic. Redesign the polynomial system to implement this generalization.
This is a major effort, not a local change.
])

#idx("term list of polynomial", sub: "representing")

#exercise(label-name: <ex:-terms>, [
A univariate polynomial can be divided by another one to produce a
#idx("polynomial arithmetic", sub: "division")
polynomial quotient and a polynomial remainder. For example,

$ mat(delim: #none, frac(x^(5)-1, x^(2) -1), =, x^(3) +x, space upright("remainder ")x-1) $

Division can be performed via long division.
That is, divide the highest-order term of the dividend by
the highest-order term of the divisor. The result is the first term of the
quotient. Next, multiply the result by the divisor, subtract that
from the dividend, and produce the rest of the answer by recursively
dividing the difference by the divisor. Stop when the order of the
divisor exceeds the order of the dividend and declare the dividend to
be the remainder. Also, if the dividend ever becomes zero, return
zero as both quotient and remainder.

We can design a
#idx("divpoly")
#py("div_poly")
function
on the model of
#py("add_poly")
and
#py("mul_poly").
The
function
checks to see if the two polys have the same variable. If so,
#py("div_poly")
strips off the variable and passes the problem to
#py("div_terms"),
which performs the division operation on term lists.
The function #py("div_poly")
finally reattaches the variable to the result supplied by
#py("div_terms").
It is convenient to design
#py("div_terms")
to compute both the quotient and the remainder of a division.
The function #py("div_terms")
can take two term lists as arguments and return a linked list of the quotient
term list and the remainder term list.

Complete the following definition of
#py("div_terms")
by filling in the missing
parts.
Use this to implement
#py("div_poly"),
which takes two polys as arguments and returns a linked list of the quotient and
remainder polys.
#idx("divterms", decl: true)
#syntax("
def div_terms(L1, L2):
    if is_empty_termlist(L1):
        return llist(the_empty_termlist, the_empty_termlist)
    else:
        t1 = first_term(L1)
        t2 = first_term(L2)
        if order(t2) > order(t1):
            return llist(the_empty_termlist, L1)
        else:
            new_c = div(coeff(t1), coeff(t2))
            new_o = order(t1) - order(t2)
            rest_of_result = ", metaphrase[compute rest of result recursively], "
            ", metaphrase[form and return complete result])
])

#subheading([Hierarchies of types in symbolic algebra])

#idx("hierarchy of types", sub: "in symbolic algebra")
#idx("polynomial(s)", sub: "hierarchy of types")
#idx("type(s)", sub: "hierarchy in symbolic algebra")

Our polynomial system illustrates how objects of one type
(polynomials) may in fact be complex objects that have objects of many
different types as parts. This poses no real difficulty in defining
generic operations. We need only install appropriate generic operations
for performing the necessary manipulations of the parts of the
compound types. In fact, we saw that polynomials form a kind of
"recursive data abstraction," in that parts of a polynomial may
themselves be polynomials. Our generic operations and our
data-directed programming style can handle this complication without
much trouble.

On the other hand, polynomial algebra is a system for which the data
types cannot be naturally arranged in a tower. For instance, it is
possible to have polynomials in $x$ whose
coefficients are polynomials in $y$. It is also
possible to have polynomials in $y$ whose
coefficients are polynomials in $x$. Neither of
these types is "above" the other in any natural way, yet it is
often necessary to add together elements from each set. There are several
ways to do this. One possibility is to convert one polynomial to the type
of the other by expanding and rearranging terms so that both polynomials
have the same principal variable. One can impose a towerlike structure on
this by ordering the variables and thus always converting any polynomial
to a
#idx("canonical form, for polynomials")
#idx("polynomial(s)", sub: "canonical form")
"canonical form" with the highest-priority variable
dominant and the lower-priority variables buried in the coefficients.
This strategy works fairly well, except that the conversion may expand
a polynomial unnecessarily, making it hard to read and perhaps less
efficient to work with. The tower strategy is certainly not natural
for this domain or for any domain where the user can invent new types
dynamically using old types in various combining forms, such as
trigonometric functions, power series, and integrals.

It should not be surprising that controlling
#idx("coercion", sub: "in algebraic manipulation")
coercion is a serious problem in the design of large-scale
algebraic-manipulation systems. Much of the complexity of such systems is
concerned with relationships among diverse types. Indeed, it is fair to
say that we do not yet completely understand coercion. In fact, we do not
yet completely understand the concept of a data type. Nevertheless, what
we know provides us with powerful structuring and modularity principles to
support the design of large systems.

#exercise(label-name: <ex:2_92>, [
By imposing an ordering on variables, extend the polynomial package so
that addition and multiplication of polynomials works for polynomials
in different variables. (This is not easy!)
])

#idx("hierarchy of types", sub: "in symbolic algebra")
#idx("polynomial(s)", sub: "hierarchy of types")
#idx("type(s)", sub: "hierarchy in symbolic algebra")

#subheading([Extended exercise: Rational functions])

#idx("rational function")
#idx("function (mathematical)", sub: "rational")
#idx("polynomial arithmetic", sub: "rational functions")

We can extend our generic arithmetic system to include #emph[rational functions]. These are "fractions" whose numerator and
denominator are polynomials, such as

$ frac(x+1, x^(3) -1) $

The system should be able to add, subtract, multiply, and divide
rational functions, and to perform such computations as

$ mat(delim: #none, frac(x+1, x^(3) -1)+frac(x, x^(2) -1), =, frac(x^(3) +2x^(2) +3x +1, x^(4) + x^(3) -x-1)) $

(Here the sum has been simplified by removing common factors.
Ordinary "cross multiplication" would have produced a
fourth-degree polynomial over a fifth-degree polynomial.)

If we modify our rational-arithmetic package so that it uses generic
operations, then it will do what we want, except for the problem
of reducing fractions to lowest terms.

#exercise(label-name: <ex:make-rat-poly>, [
Modify the rational-arithmetic package to use generic operations, but
change
#py("make_rat")
so that it does not attempt to reduce fractions to lowest terms. Test
your system by calling
#py("make_rational")
on two polynomials to produce a rational function

#snippet(```python
p1 = make_polynomial("x", llist(make_term(2, 1), make_term(0, 1)))
p2 = make_polynomial("x", llist(make_term(3, 1), make_term(0, 1)))
rf = make_rational(p2, p1)
```)

Now add #py("rf") to itself, using
#py("add"). You will observe that this addition
function
does not reduce fractions to lowest terms.
])

We can reduce polynomial fractions to lowest terms using the same idea
we used with integers: modifying
#py("make_rat")
to divide both the numerator and the denominator by their greatest common
divisor. The notion of
#idx("greatest common divisor", sub: "of polynomials")
#idx("polynomial arithmetic", sub: "greatest common divisor")
"greatest common divisor" makes sense for polynomials. In
fact, we can compute the GCD of two polynomials using essentially the
same Euclid's Algorithm that works for integers.#footnote[The fact
that
#idx("Euclid's Algorithm", sub: "for polynomials")
#idx("polynomial arithmetic", sub: "Euclid's Algorithm")
Euclid's Algorithm works for polynomials is formalized in algebra
by saying that polynomials form a kind of algebraic domain called a
#idx("Euclidean ring")
#idx("measure in a Euclidean ring")
#emph[Euclidean ring]. A Euclidean ring is a domain that admits
addition, subtraction, and commutative multiplication, together with a
way of assigning to each element $x$ of the
ring a positive integer
"measure" $m(x)$ with the
properties that $m(x y) gt.eq m(x)$ for any nonzero
$x$ and $y$ and that,
given any $x$ and $y$,
there exists a $q$ such that
$y=q x+r$ and either
$r=0$ or
$m(r) < m(x)$. From an abstract point of
view, this is what is needed to prove that Euclid's Algorithm works.
For the domain of integers, the measure $m$ of an
integer is the absolute value of the integer itself. For the domain of
polynomials, the measure of a polynomial is its degree.] The
integer version is

#snippet(```python
def gcd(a, b):
    return (a
            if b == 0
            else gcd(b, a % b))
```)

Using this, we could make the obvious modification to define a GCD
operation that works on term lists:
#idx("gcdterms", decl: true)
#snippet(```python
def gcd_terms(a, b):
    return (a
            if is_empty_termlist(b)
            else gcd_terms(b, remainder_terms(a, b)))
```)

where
#py("remainder_terms")
picks out the remainder component of the list returned by the term-list
division operation
#py("div_terms")
that was implemented in exercise @ex:-terms.

#exercise(label-name: <ex:remainder-terms>, [
Using
#py("div_terms"),
implement the
function
#idx("remainderterms")
#py("remainder_terms")
and use this to define
#py("gcd_terms")
as above. Now write a
function
#idx("greatest common divisor", sub: "generic")
#py("gcd_poly")
that computes the polynomial GCD of two polys. (The
function
should signal an error if the two polys are not in the same variable.)
Install in the system a generic operation
#py("greatest_common_divisor")
that reduces to
#py("gcd_poly")
for polynomials and to ordinary #py("gcd") for
ordinary numbers. As a test, try

#snippet(```python
p1 = make_polynomial("x", llist(make_term(4, 1), make_term(3, -1),
                                make_term(2, -2), make_term(1, 2)))
p2 = make_polynomial("x", llist(make_term(3, 1), make_term(1, -1)))
greatest_common_divisor(p1, p2)
```)

and check your result by hand.
])

#exercise(label-name: <ex:gcd-of-polys>, [
Define $P_(1)$,
$P_(2)$, and
$P_(3)$ to be the polynomials

#sicp-table(columns: 2, [$P_(1)$:], [$x^(2) - 2x + 1$], [$P_(2)$:], [$11x^(2) + 7$], [$P_(3)$:], [$13x + 5$])

Now define $Q_(1)$ to be the product of
$P_(1)$ and $P_(2)$ and
$Q_(2)$ to be the product of
$P_(1)$ and $P_(3)$, and
use
#py("greatest_common_divisor")
(exercise @ex:remainder-terms) to compute the GCD of
$Q_(1)$ and $Q_(2)$.
Note that the answer is not the same as $P_(1)$.
This example introduces noninteger operations into the computation, causing
difficulties with the GCD
algorithm.#footnote[In Python, division of integers can produce limited-precision decimal numbers, and thus we may fail to get a valid divisor.]
To understand what is happening, try tracing
#py("gcd_terms")
while computing the GCD or try performing the division by hand.
])

We can solve the problem exhibited in
exercise @ex:gcd-of-polys if
we use the following modification of the GCD algorithm (which really
works only in the case of polynomials with integer coefficients).
Before performing any polynomial division in the GCD computation, we
multiply the dividend by an integer constant factor, chosen to
guarantee that no fractions will arise during the division process.
Our answer will thus differ from the actual GCD by an integer constant
factor, but this does not matter in the case of reducing rational
functions to lowest terms; the GCD will be used to divide both the
numerator and denominator, so the integer constant factor will cancel
out.

More precisely, if $P$ and
$Q$ are polynomials, let
$O_(1)$ be the order of
$P$ (i.e., the order of the largest term of
$P$) and let $O_(2)$
be the order of $Q$. Let
$c$ be the leading coefficient of
$Q$. Then it can be shown that, if we multiply
$P$ by the
#idx("integerizing factor")
#emph[integerizing factor]
$c^(1+O_(1) -O_(2))$, the resulting polynomial
can be divided by $Q$ by using the
#py("div_terms")
algorithm without introducing any fractions. The operation of multiplying
the dividend by this constant and then dividing is sometimes called the
#idx("pseudodivision of polynomials")
#emph[pseudodivision] of $P$ by
$Q$. The remainder of the division is
called the
#idx("pseudoremainder of polynomials")
#emph[pseudoremainder].

#exercise(label-name: <ex:pseudoremainder-terms>, [
+ Implement the function #py("pseudoremainder_terms"), which is just like #py("remainder_terms") except that it multiplies the dividend by the integerizing factor described above before calling #py("div_terms"). Modify #py("gcd_terms") to use #py("pseudoremainder_terms"), and verify that #py("greatest_common_divisor") now produces an answer with integer coefficients on the example in exercise @ex:gcd-of-polys.
+ The GCD now has integer coefficients, but they are larger than those of $P_(1)$. Modify #py("gcd_terms") so that it removes common factors from the coefficients of the answer by dividing all the coefficients by their (integer) greatest common divisor.
])

#idx("polynomial arithmetic", sub: "greatest common divisor")
#idx("rational function", sub: "reducing to lowest terms")
#idx("reducing to lowest terms")

Thus, here is how to reduce a rational function to lowest terms:

- Compute the GCD of the numerator and denominator, using the version of #py("gcd_terms") from exercise @ex:pseudoremainder-terms.
- When you obtain the GCD, multiply both numerator and denominator by the same integerizing factor before dividing through by the GCD, so that division by the GCD will not introduce any noninteger coefficients. As the factor you can use the leading coefficient of the GCD raised to the power $1+O_(1) -O_(2)$, where $O_(2)$ is the order of the GCD and $O_(1)$ is the maximum of the orders of the numerator and denominator. This will ensure that dividing the numerator and denominator by the GCD will not introduce any fractions.
- The result of this operation will be a numerator and denominator with integer coefficients. The coefficients will normally be very large because of all of the integerizing factors, so the last step is to remove the redundant factors by computing the (integer) greatest common divisor of all the coefficients of the numerator and the denominator and dividing through by this factor.

#exercise(label-name: <ex:reduce-poly>, [
+ Implement this algorithm as a function #py("reduce_terms") that takes two term lists #py("n") and #py("d") as arguments and returns a linked list #py("nn"), #py("dd"), which are #py("n") and #py("d") reduced to lowest terms via the algorithm given above. Also write a function #py("reduce_poly"), analogous to #py("add_poly"), that checks to see if the two polys have the same variable. If so, #py("reduce_poly") strips off the variable and passes the problem to #py("reduce_terms"), then reattaches the variable to the two term lists supplied by #py("reduce_terms").
+ Define a function analogous to #py("reduce_terms") that does what the original #py("make_rat") did for integers: #snippet(```python def reduce_integers(n, d): g = gcd(n, d) return llist(n // g, d // g) ```) and define #py("reduce") as a generic operation that calls #py("apply_generic") to dispatch either to #py("reduce_poly") (for #py("polynomial") arguments) or to #py("reduce_integers") (for #py("python_number") arguments). You can now easily make the rational-arithmetic package reduce fractions to lowest terms by having #py("make_rat") call #py("reduce") before combining the given numerator and denominator to form a rational number. The system now handles rational expressions in either integers or polynomials. To test your program, try the example at the beginning of this extended exercise: #snippet(```python p1 = make_polynomial("x", llist(make_term(1, 1), make_term(0, 1))) p2 = make_polynomial("x", llist(make_term(3, 1), make_term(0, -1))) p3 = make_polynomial("x", llist(make_term(1, 1))) p4 = make_polynomial("x", llist(make_term(2, 1), make_term(0, -1))) rf1 = make_rational(p1, p2) rf2 = make_rational(p3, p4) add(rf1, rf2) ```) See if you get the correct answer, correctly reduced to lowest terms.
])

The GCD computation is at the heart of any system that does operations
on rational functions. The algorithm used above, although
mathematically straightforward, is extremely slow. The slowness is
due partly to the large number of division operations and partly to
the enormous size of the intermediate coefficients generated by the
pseudodivisions.
#idx("rational function", sub: "reducing to lowest terms")
#idx("reducing to lowest terms")
One of the active areas in the development of
algebraic-manipulation systems is the design of better algorithms for
computing polynomial GCDs.#footnote[One extremely efficient and
elegant method for computing
#idx("polynomial arithmetic", sub: "greatest common divisor")
#idx("polynomial arithmetic", sub: "probabilistic algorithm for GCD")
#idx("probabilistic algorithm")
#idx("algorithm", sub: "probabilistic")
polynomial GCDs was discovered by
#idx("Zippel, Richard E.")
Richard Zippel (1979). The method is a probabilistic algorithm, as is the
fast test for primality that we discussed in chapter @chap:fun.
Zippel's book (1993) describes this method, together with other ways
to compute polynomial GCDs.]
#idx("rational function")
#idx("function (mathematical)", sub: "rational")
#idx("polynomial arithmetic", sub: "rational functions")
#idx("symbolic algebra")
#idx("polynomial(s)")
#idx("polynomial arithmetic")
