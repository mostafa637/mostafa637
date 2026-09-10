// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Linear Recursion and Iteration], label-name: <sec:recursion-and-iteration>)

#idx("iterative process", sub: "recursive process vs.")
#idx("recursive process", sub: "iterative process vs.")

We begin by considering the
#idx("factorial")
factorial function, defined by

$ mat(delim: #none, n!, =, n dot.op (n-1) dot.op (n-2) dots.c 3 dot.op 2 dot.op 1) $

There are many ways to compute factorials. One way is to make use of
the observation that $n!$ is equal to
$n$ times $(n-1)!$ for
any positive integer $n$:

$ mat(delim: #none, n!, =, n dot.op lr([ (n-1) dot.op (n-2) dots.c 3 dot.op 2 dot.op 1 ]) = n dot.op (n-1)!) $

Thus, we can compute $n!$ by computing
$(n-1)!$ and multiplying the
result by $n$. If we add the stipulation that 1!
is equal to 1,
this observation translates directly into a
computer function:
#idx("factorial", sub: "linear recursive version", decl: true)
#snippet(```python
def factorial(n):
    return 1 if n == 1 else n * factorial(n - 1)
```)

We can use the substitution model of
section @sec:substitution-model to watch this
#idx("substitution model of function application", sub: "shape of process")
function in action computing 6!, as shown in
figure @fig:recursive-factorial-javascript.

Now let's take a different perspective on computing factorials. We
could describe a rule for computing $n!$ by
specifying that we first multiply 1 by 2, then multiply the result by 3,
then by 4, and so on until we reach $n$.
More formally, we maintain a running product, together with a counter
that counts from 1 up to $n$. We can describe
the computation by saying that the counter and the product simultaneously
change from one step to the next according to the rule

$ mat(delim: #none,
upright("product"), arrow.l, upright("counter") dot.op upright("product");
upright("counter"), arrow.l, upright("counter") + 1) $

and stipulating that $n!$ is the value of the
product when the counter exceeds $n$.

Once again, we can recast our description as a
function
for computing
factorials:#footnote[In a real program we would probably use the
block structure introduced in the last section to hide the
definition of #py("fact_iter"):

#snippet(```python
def factorial(n):
    def iterate(product, counter):
        return (product
                if counter > n
                else iterate(counter * product, counter + 1))
    return iterate(1, 1)
```)

We avoided doing this here so as to minimize the number of things to
think about at
once.]<foot:block-structured-factorial>
#idx("factorial", sub: "linear iterative version", decl: true)
#snippet(```python
def factorial(n):
    return fact_iter(1, 1, n)

def fact_iter(product, counter, max_count):
    return (product
            if counter > max_count
            else fact_iter(counter * product, counter + 1, max_count))
```)

As before, we can use the substitution model to visualize the process

#sicp-figure(image("/images/img_javascript/ch1-Z-G-7.svg", width: 70%), caption: [A linear recursive process for computing 6!.], label-name: <fig:recursive-factorial-javascript>)

#sicp-figure(image("/images/img_javascript/ch1-Z-G-10.svg", width: 70%), caption: [A linear iterative process for computing $6!$.], label-name: <fig:iterative-factorial-javascript>)

of computing $6!$, as shown in
figure @fig:iterative-factorial-javascript.

Compare the two processes. From one point of view, they seem hardly
different at all. Both compute the same mathematical function on the
same domain, and each requires a number of steps proportional to
$n$
to compute $n!$. Indeed, both processes even
carry out the same sequence of multiplications, obtaining the same sequence
of partial products. On the other hand, when we consider the
#idx("shape of a process")
#idx("process", sub: "shape of")
"shapes" of the two processes, we find that they evolve quite
differently.

Consider the first process. The substitution model reveals a shape of
expansion followed by contraction, indicated by the arrow in
figure @fig:recursive-factorial-javascript.
The expansion occurs as the process builds up a chain of
#idx("deferred operations")
#emph[deferred operations] (in this case, a chain of multiplications).
The contraction occurs as the operations are actually performed. This
type of process, characterized by a chain of deferred operations, is called a
#idx("recursive process")
#idx("process", sub: "recursive")
#emph[recursive process]. Carrying out this process requires that the
interpreter keep track of the operations to be performed later on. In the
computation of $n!$, the length of the chain of
deferred multiplications, and hence the amount of information needed to
keep track of it,
#idx("linear growth")
grows linearly with $n$ (is proportional to
$n$), just like the number of steps.
Such a process is called a
#idx("recursive process", sub: "linear")
#idx("linear recursive process")
#idx("process", sub: "linear recursive")
#emph[linear recursive process].

By contrast, the second process does not grow and shrink. At each
step, all we need to keep track of, for any $n$,
are the current values of the
names
#py("product"), #py("counter"),
and
#py("max_count").
We call this an
#idx("iterative process")
#idx("process", sub: "iterative")
#emph[iterative process]. In general, an iterative process is one whose
state can be summarized by a fixed number of
#idx("state variable")
#emph[state variables], together with a fixed rule that describes how
the state variables should be updated as the process moves from state to
state and an (optional) end test that specifies conditions under which the
process should terminate. In computing $n!$, the
number of steps required grows linearly with $n$.
Such a process is called a
#idx("iterative process", sub: "linear")
#idx("linear iterative process")
#idx("process", sub: "linear iterative")
#emph[linear iterative process].

The contrast between the two processes can be seen in another way.
In the iterative case, the state variables provide a complete description of
the state of the process at any point. If we stopped the computation between
steps, all we would need to do to resume the computation is to supply the
interpreter with the values of the three state variables. Not so with the
recursive process. In this case there is some additional
"hidden" information, maintained by the interpreter and not
contained in the state variables, which indicates "where the process is" in negotiating the chain of deferred operations. The longer the
chain, the more information must be maintained.#footnote[When we discuss the
implementation of
functions
on register machines in chapter @chap:reg, we will see that any iterative
process can be realized "in hardware" as a machine that has a
fixed set of registers and no auxiliary memory. In contrast, realizing a
recursive process requires a machine that uses an
auxiliary data structure known as a
#idx("stack")
#emph[stack].]
#idx("substitution model of function application", sub: "shape of process")

In contrasting iteration and recursion, we must be careful not to
confuse the notion of a
#idx("recursive function", sub: "recursive process vs.")
#idx("recursive process", sub: "recursive function vs.")
recursive #emph[process] with the notion of a recursive
#emph[function].
When we describe a
function
as recursive, we are referring to the syntactic fact that the
function definition
refers (either directly or indirectly) to the
function
itself. But when we describe a process as following a pattern that is, say,
linearly recursive, we are speaking about how the process evolves, not
about the syntax of how a
function
is written. It may seem disturbing that we refer to a recursive
function
such as
#py("fact_iter")
as generating an iterative process. However, the process really is
iterative: Its state is captured completely by its three state variables,
and an interpreter need keep track of only three
names
in order to execute the process.

One reason that the distinction between process and
function
may be confusing is that most implementations of common languages
(including #idx("C", sub: "recursive functions in") C, #idx("Java, recursive functions in") Java, and #idx("Python, recursive functions in") Python)
are designed in such a way that the interpretation of
any recursive
function
consumes an amount of memory that grows with the number of
function
calls, even when the process described is, in principle, iterative.
As a consequence, these languages can describe iterative processes only
by resorting to special-purpose
#idx("looping constructs")
"looping constructs" such as
$mono("do")$,
$mono("repeat")$,
$mono("until")$,
$mono("for")$, and
$mono("while")$.
The implementation of
Python
we shall consider in chapter @chap:reg does not share this defect. It will
execute an iterative process in constant space, even if the iterative
process is described by a recursive
function.

An implementation with this property is called
#idx("tail recursion")
#emph[tail-recursive].#footnote[Tail recursion has long been
known as a compiler optimization trick. A coherent semantic basis for
tail recursion was provided by
#idx("Hewitt, Carl Eddie")
Carl Hewitt (1977), who explained it in
#idx("message passing", sub: "tail recursion and")
terms of the "message-passing" model of computation that we
shall discuss in chapter @chap:state. Inspired by this, Gerald Jay Sussman
and
#idx("Steele, Guy Lewis Jr.")
Guy Lewis Steele Jr. (see Steele 1975)
constructed a tail-recursive interpreter for Scheme. Steele later showed
how tail recursion is a consequence of the natural way to compile
function calls
#idx("Sussman, Gerald Jay")
(Steele 1977).
The IEEE standard for Scheme requires that Scheme implementations
#idx("tail recursion", sub: "in Scheme")
#idx("tail recursion", sub: "in Python")
#idx("Scheme", sub: "tail recursion in")
#idx("Python", sub: "tail recursion in")
be tail-recursive. The Python Language Reference does not specify tail recursion, one way or the other. While most implementations of Python are not tail-recursive, we assume in this book an implementation that is.]
With a tail-recursive implementation,
#idx("iterative process", sub: "implemented by function call")
iteration can be expressed using the ordinary function
call mechanism, so that special iteration constructs are useful only as
#idx("syntactic sugar", sub: "looping constructs as")
syntactic sugar.#footnote[Exercise @ex:while_loop
explores Python's while loops as syntactic
sugar for functions that give rise to iterative processes.
The full language Python, like other conventional languages,
features a plethora of syntactic
forms, all of which can be expressed more uniformly in the
language Lisp.]

#idx("iterative process", sub: "recursive process vs.")
#idx("recursive process", sub: "iterative process vs.")

#exercise(label-name: <ex:addition-procedures>, [
Each of the following two
functions
defines a method for adding two positive integers in terms of the
functions
#py("inc"), which increments its argument by 1,
and #py("dec"), which decrements its argument by 1.

#snippet(```python
def plus(a, b):
    return b if a == 0 else inc(plus(dec(a), b))
```)

#snippet(```python
def plus(a, b):
    return b if a == 0 else plus(dec(a), inc(b))
```)

Using the substitution model, illustrate the process generated by each
function
in evaluating
#py("plus(4, 5)").
Are these processes iterative or recursive?
])

#exercise(label-name: <ex:1_10>, [
The following
function
computes a mathematical function called
#idx("Ackermann's function")
#idx("function (mathematical)", sub: "Ackermann's")
Ackermann's function.

#snippet(```python
def A(x, y):
    return (0 if y == 0
            else 2 * y if x == 0
            else 2 if y == 1
            else A(x - 1, A(x, y - 1)))
```)

What is printed by the following statements?

#snippet(```python
print(A(1, 10))
```)

#snippet(```python
print(A(2, 4))
```)

#snippet(```python
print(A(3, 3))
```)

Consider the following
functions,
where #py("A") is the
function defined
above:

#snippet(```python
def f(n):
    return A(0, n)

def g(n):
    return A(1, n)

def h(n):
    return A(2, n)

def k(n):
    return 5 * n * n
```)

Give concise mathematical definitions for the functions computed by
the
functions
#py("f"), #py("g"), and
#py("h") for positive integer values of
$n$. For example,
$k(n)$ computes
$5n^(2)$.
])
