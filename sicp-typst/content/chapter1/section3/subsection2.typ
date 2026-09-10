// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Constructing Functions using Lambda Expressions], label-name: <sec:lambda>)

In using #py("sum") as in
section @sec:procedures-as-parameters, it seems
terribly awkward to have to define trivial functions such as
#py("pi_term") and
#py("pi_next") just so we can use them as
arguments to our higher-order function. Rather than define
#py("pi_next") and
#py("pi_term"), it would be more convenient
to have a way to directly specify "the function that returns its input incremented by 4" and "the function that returns the reciprocal of its input times its input plus 2." We can do this
by introducing the #emph[lambda expression] as a syntactic form for
creating functions.
Using lambda expressions, we can describe what we want as

#snippet(```python
lambda x: x + 4
```)

and

#snippet(```python
lambda x: 1 / (x * (x + 2))
```)

Then we can express our #py("pi_sum") function
without
defining any auxiliary functions:
#idx("pisum", sub: "with lambda expression", decl: true)
#snippet(```python
def pi_sum(a, b):
    return sum(lambda x: 1 / (x * (x + 2)),
               a,
               lambda x: x + 4,
               b)
```)

Again using
a lambda expression,
we can write the #py("integral")
function
without having to
define the auxiliary function
#py("add_dx"):

#idx("integral", sub: "with lambda expression", decl: true)
#snippet(```python
def integral(f, a, b, dx):
    return sum(f,
               a + dx / 2,
               lambda x: x + dx,
               b) * dx
```)

In general, lambda expressions are used to create functions in the
same way as function definitions,

#idx("lambda expression")
#idx("syntactic forms", sub: "lambda expression")
#idx("lambda expression", sub: "function definition vs.")
#idx("function definition", sub: "lambda expression vs.")

except that no name is specified for the function and the
#idx("parentheses", sub: "around parameters of lambda expression")
parentheses around the parameters and the
#py("return") keyword are omitted.

#syntax("
lambda ", meta("parameters"), ": ", meta("expression"))

The resulting function is just as much a function
as one that is created using a function definition statement.
#idx("function definition", sub: "lambda expression vs.")
#idx("lambda expression", sub: "function definition vs.")
The only difference is that it has not been associated with any name in the
environment.

In fact,

#snippet(```python
plus4 = lambda x: x + 4
```)

is equivalent to
#idx("lambda expression", sub: "function definition vs.")
#idx("function definition", sub: "lambda expression vs.")

#snippet(```python
def plus4(x):
    return x + 4
```)

We can read a lambda expression as follows:

$ mat(delim: #none, mono(bold("lambda") "x"), mono(":"), mono("x"), mono("+"), mono("4"); arrow.t, arrow.t, arrow.t, arrow.t, arrow.t; mono("The function of an argument" "x"), "that results in", "the value", "plus", "4.") $

Like any expression that has a
function
#idx("lambda expression", sub: "as function expression of application") #idx("function expression", sub: "lambda expression as")
as its value, a
lambda
expression can be used as the function expression in an application
 such as

#snippet(```python
print((lambda x, y, z: x + y + square(z))(1, 2, 3))
```)

#output(```python
print((lambda x, y, z: x + y + square(z))(1, 2, 3))
```)

or, more generally, in any context where we would normally use a
function
name.#footnote[It would be clearer and less intimidating to people learning
Python
if a
term
more obvious than
#emph[lambda expression], such as #emph[function definition expression],
were used. But the convention is
very firmly entrenched, not just for Lisp and Scheme but also for Python, Java and other languages, no doubt partly due to the influence of the Scheme editions of this book. #idx("Scheme", sub: "use of lambda in")
The notation is adopted from the
#idx("λ calculus (lambda calculus)", sort: "0l")
#idx("λ calculus (lambda calculus)", sort: "lambda")
$lambda$ calculus, a
mathematical formalism introduced by the mathematical logician
#idx("Church, Alonzo")
Alonzo Church (1941). Church developed the
$lambda$ calculus to provide a rigorous
foundation for studying the notions of
function and function application. The
$lambda$ calculus has become a basic
tool for mathematical investigations of the
semantics of programming languages.]
Note that a #py("lambda") expression has #idx("precedence", sub: "of lambda expression") #idx("lambda expression", sub: "precedence of") lower precedence than function application and thus the #idx("parentheses", sub: "around lambda expression") parentheses around the lambda expression are necessary here.

#subheading([Using declaration assignment to create local variables])

#idx("local name")

Another use of
#py("lambda")
is in creating local variables.
We often need local variables in our functions other than those that have been bound as parameters.
For example, suppose we wish to compute the function

$ mat(delim: #none, f(x, y), =, x(1 + x y)^(2) +y (1 - y) + (1 + x y)(1 - y)) $

which we could also express as

$ mat(delim: #none, a, =, 1+x y; b, =, 1-y; f(x, y), =, x a^(2) +y b + a b) $

In writing a
function
to compute $f$, we would like to include as
local variables
not only $x$ and $y$
but also the names of intermediate quantities like
$a$ and $b$. One way
to accomplish this is to use an auxiliary
function
to bind the local variables:

#snippet(```python
def f(x, y):
    def f_helper(a, b):
        return x * square(a) + y * b + a * b
    return f_helper(1 + x * y, 1 - y)
```)

Of course, we could use a
#py("lambda")
expression to specify an anonymous
function
for binding our local variables.
The body of
#py("f")
then becomes a single call to that
function:

#snippet(```python
def f_2(x, y):
    return (lambda a, b:
            x * square(a) + y * b + a * b)(1 + x * y, 1 - y)
```)

A more convenient way to declare local variables is by using
declaration assignments within the body of the function.
Using a declaration assignment, the function

can be written as

#snippet(```python
def f_3(x, y):
    a = 1 + x * y
    b = 1 - y
    return x * square(a) + y * b + a * b
```)

Variables that are declared with declaration assignments
inside a function have the
body of the immediately surrounding function as their scope.#footnote[#idx("declaration", sub: "use of name before")
Note that a name declared in a function cannot be used before the
declaration is fully evaluated, regardless of whether the same name is
declared outside the function. Thus in the program below, the
attempt to use the #py("a") declared
at the top level
to provide a value for the calculation of
the #py("b") declared in
#py("f") cannot work.

#snippet(```python
a = 1
def f(x):
    b = a + x
    a = 5
    return a + b
f(10)
```)

The program
leads to an error, because the #py("a") in
#py("a + x") is used before its declaration
is evaluated. We will return to this program in
section @sec:internal-definitions
(exercise @ex:simultaneous-def), after we learn
more about evaluation.]<foot:tdz>
$""^(,)$
#footnote[The substitution
model can be expanded to say that for a declaration assignment, the value of the
expression after #py("=")
is substituted for the name before
#py("=")
in the rest of the function body (after the declaration), similar to the
substitution of arguments for parameters in the evaluation of a
function application.]
#idx("local name")

#subheading([Conditional statements])

We have seen that it is often useful to declare variables that are local to
function definitions. When functions become big, we should
keep the computation associated with the variables as restricted as possible.
Consider for example #py("expmod") in
exercise @ex:louis-fast-prime.

#snippet(```python
def expmod(base, exp, m):
    return (1 if exp == 0
            else (expmod(base, exp // 2, m)
                  * expmod(base, exp // 2, m)) % m if is_even(exp)
            else (base * expmod(base, exp - 1, m)) % m)
```)

This function is unnecessarily inefficient, because it contains two
identical calls:

#snippet(```python
expmod(base, exp // 2, m)
```)

While this can be easily fixed in this example using the
#py("square") function, this is not so easy
in general. Without using #py("square"),
we would be tempted to introduce a local name for the expression as
follows:

#snippet(```python
def expmod(base, exp, m):
    half_exp = expmod(base, exp // 2, m)
    return (1 if exp == 0
            else (half_exp * half_exp) % m if is_even(exp)
            else (base * expmod(base, exp - 1, m)) % m)
```)

#idx("conditional statement", sub: "need for")
This would make the function not just inefficient, but actually
nonterminating! The problem is that the declaration assignment appears
outside the conditional expression, which means that it is executed even
when the base case #py("exp == 0") is met.
To avoid this situation, we provide for
#idx("conditional statement")
#idx("syntactic forms", sub: "conditional statement")
#idx("if (keyword)", sort: "if")
#idx("else (keyword)", sort: "else")
#idx("keywords", sub: "if")
#idx("keywords", sub: "else")
#idx("predicate", sub: "of conditional statement")
#idx("conditional statement", sub: "predicate, consequent, and alternative of")
#emph[conditional statements], and allow return
statements to appear in the branches of the statement. Using a
conditional statement, we can write the function
#py("expmod") as follows:

#snippet(```python
def expmod(base, exp, m):
    if exp == 0:
        return 1
    else:
        if is_even(exp):
            half_exp = expmod(base, exp // 2, m)
            return (half_exp * half_exp) % m
        else:
            return (base * expmod(base, exp - 1, m)) % m
```)

The simplest form of a conditional statement is

#syntax("
if ", meta("predicate"), ":
    ", meta("consequent-statements"), "
else:
    ", meta("alternative-statements"))

As for a conditional expression, the interpreter first evaluates the
#meta("predicate"). If it evaluates to true,
the interpreter evaluates the
#idx("consequent", sub: "of conditional statement")
#idx("conditional statement", sub: "consequent statements of")
#meta("consequent-statements") in sequence, and if it
evaluates to false, the interpreter evaluates
#idx("alternative", sub: "of conditional statement")
#idx("conditional statement", sub: "alternative statements of")
the #meta("alternative-statements") in sequence. Evaluation of a return
statement returns from the surrounding function, ignoring any
statements in the sequence
#idx("sequence of statements", sub: "in conditional statement")
after the return statement and any statements after the conditional statement.
#idx("conditional statement")

Python provides the keyword
#py("elif") to avoid deeply nested
#py("else: if") statements. Using
#py("elif"),
we can write the #py("expmod") function as follows:

#snippet(```python
def expmod(base, exp, m):
    if exp == 0:
        return 1
    elif is_even(exp):
        half_exp = expmod(base, exp // 2, m)
        return (half_exp * half_exp) % m
    else:
        return (base * expmod(base, exp - 1, m)) % m
```)

#exercise(label-name: <ex:1_34>, [
Suppose we
define the function

#snippet(```python
def f(g):
    return g(2)
```)

Then we have

#snippet(```python
print(f(square))
```)

#output(```python
print(f(square))
```)

#snippet(```python
print(f(lambda z: z * (z + 1)))
```)

#output(```python
print(f(lambda z: z * (z + 1)))
```)

What happens if we (perversely) ask the interpreter to evaluate the
application #py("f(f)")?
Explain.
])
