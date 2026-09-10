// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Sequences as Conventional Interfaces], label-name: <sec:sequences-conventional-interfaces>)

#idx("sequence(s)", sub: "as conventional interface")
#idx("conventional interface", sub: "sequence as")

In working with compound data, we've stressed how data abstraction
permits us to design programs without becoming enmeshed in the details
of data representations, and how abstraction preserves for us the
flexibility to experiment with alternative representations. In this
section, we introduce another powerful design principle for working
with data structures—the use of #emph[conventional interfaces].

In section @sec:higher-order-procedures we saw how
program abstractions, implemented as higher-order
functions,
can capture common patterns in programs that deal with numerical data. Our
ability to formulate analogous operations for working with compound data
depends crucially on the style in which we manipulate our data structures.
Consider, for example, the following
function,
analogous to the
#py("count_leaves")
function
of section @sec:trees, which takes a tree as argument
and computes the sum of the squares of the leaves that are odd:

#idx("sumoddsquares", decl: true)
#snippet(```python
def sum_odd_squares(tree):
    return (0 if is_none(tree)
            else (square(tree) if is_odd(tree)
                  else 0) if not is_pair(tree)
            else sum_odd_squares(head(tree)) +
                 sum_odd_squares(tail(tree)))
```)

On the surface, this
function
is very different from the following one, which constructs a linked list of all
the even Fibonacci numbers
$(upright("Fib"))(k)$, where
$k$ is less than or equal to a given integer
$n$:
#idx("evenfibs", decl: true)
#snippet(```python
def even_fibs(n):
    def next(k):
        if k > n:
            return None
        else:
            f = fib(k)
            return (pair(f, next(k + 1)) if is_even(f)
                    else next(k + 1))
    return next(0)
```)

Despite the fact that these two
functions
are structurally very different, a more abstract description of the two
computations reveals a great deal of similarity. The first program

- enumerates the leaves of a tree;
- filters them, selecting the odd ones;
- squares each of the selected ones; and
- accumulates the results using #py("+"), starting with 0.

The second program

- enumerates the integers from 0 to $n$;
- computes the Fibonacci number for each integer;
- filters them, selecting the even ones; and
- accumulates the results using #py("pair"), starting with the empty linked list.

#sicp-figure(image("/images/img_javascript/ch2-Z-G-17.svg", width: 70%), caption: [The signal-flow plans for the functions #py("sum_odd_squares") (top) and #py("even_fibs") (bottom) reveal the commonality between the two programs.], label-name: <fig:signal-flow-plans>)

A signal-processing engineer would find it natural to conceptualize these
processes in terms of
#idx("signal-processing view of computation")
#idx("signal-flow diagram")
signals flowing through a cascade of stages, each of
which implements part of the program plan, as shown in
figure @fig:signal-flow-plans.
In
#py("sum_odd_squares"),
we begin with an
#idx("enumerator")
#emph[enumerator], which generates a "signal" consisting of
the leaves of a given tree. This signal is passed through a
#idx("filter")
#emph[filter], which eliminates all but the odd elements. The resulting
signal is in turn passed through a
#idx("mapping", sub: "as a transducer")
#emph[map], which is a "transducer" that applies the
#py("square")
function
to each element. The output of the map is then fed to an
#idx("accumulator")
#emph[accumulator], which combines the elements using
#py("+"),
starting from an initial 0. The plan for
#py("even_fibs")
is analogous.

Unfortunately, the two
function definitions
above fail to exhibit this signal-flow structure. For instance, if we
examine the
#py("sum_odd_squares")
function,
we find that the enumeration is implemented partly by the
#py("is_none")
and
#py("is_pair")
tests and partly by the tree-recursive structure of the
function.
Similarly, the accumulation is found partly in the tests and partly in the
addition used in the recursion. In general, there are no distinct parts of
either
function
that correspond to the elements in the signal-flow description. Our two
functions
decompose the computations in a different way, spreading the enumeration
over the program and mingling it with the map, the filter, and the
accumulation. If we could organize our programs to make the signal-flow
structure manifest in the
functions
we write, this would increase the conceptual clarity of the resulting
program.

#subheading([Sequence Operations])

#anchor(<sec:sequence-operations>)
#idx("sequence(s)", sub: "operations on")

The key to organizing programs so as to more clearly reflect the
signal-flow structure is to concentrate on the "signals" that
flow from one stage in the process to the next. If we represent these
signals as linked lists, then we can use linked-list operations to implement the
processing at each of the stages. For instance, we can implement the
mapping stages of the signal-flow diagrams using the
#py("map")
function
from section @sec:sequences:

#snippet(```python
print_llist(map(square, llist(1, 2, 3, 4, 5)))
```)

#output(```python
print_llist(map(square, llist(1, 2, 3, 4, 5)))
```)

Filtering a sequence to select only those elements that satisfy a given
predicate is accomplished by
#idx("filter", decl: true)
#snippet(```python
def filter(predicate, sequence):
    return (None if is_none(sequence)
            else pair(head(sequence),
                      filter(predicate, tail(sequence)))
            if predicate(head(sequence))
            else filter(predicate, tail(sequence)))
```)

For example,

#snippet(```python
print_llist(filter(is_odd, llist(1, 2, 3, 4, 5)))
```)

#output(```python
print_llist(filter(is_odd, llist(1, 2, 3, 4, 5)))
```)

Accumulations can be implemented by
#idx("reduce", decl: true)
#snippet(```python
def reduce(op, initial, sequence):
    return (initial if is_none(sequence)
            else op(head(sequence),
                    reduce(op, initial, tail(sequence))))
```)

#snippet(```python
print(reduce(plus, 0, llist(1, 2, 3, 4, 5)))
```)

#output(```python
print(reduce(plus, 0, llist(1, 2, 3, 4, 5)))
```)

#snippet(```python
print(reduce(times, 1, llist(1, 2, 3, 4, 5)))
```)

#output(```python
print(reduce(times, 1, llist(1, 2, 3, 4, 5)))
```)

#snippet(```python
print_llist(reduce(pair, None, llist(1, 2, 3, 4, 5)))
```)

#output(```python
print_llist(reduce(pair, None, llist(1, 2, 3, 4, 5)))
```)

All that remains to implement signal-flow diagrams is to enumerate the
sequence of elements to be processed. For
#py("even_fibs"),
we need to generate the sequence of integers in a given range, which we
can do as follows:
#idx("enumerateinterval", decl: true)
#snippet(```python
def enumerate_interval(low, high):
    return (None if low > high
            else pair(low,
                      enumerate_interval(low + 1, high)))
```)

#snippet(```python
print_llist(enumerate_interval(2, 7))
```)

#output(```python
print_llist(enumerate_interval(2, 7))
```)

To enumerate the leaves of a tree, we can use#footnote[This is, in fact,
precisely the
#idx("fringe", sub: "as a tree enumeration")
#py("fringe")
function
from exercise @ex:fringe. Here we've renamed it
to emphasize that it is part of a family of general sequence-manipulation
functions.]
#idx("tree", sub: "enumerating leaves of")#idx("enumeratetree", decl: true)
#snippet(```python
def enumerate_tree(tree):
    return (None if is_none(tree)
            else llist(tree) if not is_pair(tree)
            else append(enumerate_tree(head(tree)),
                        enumerate_tree(tail(tree))))
```)

#snippet(```python
print_llist(enumerate_tree(llist(1, llist(2, llist(3, 4)), 5)))
```)

#output(```python
print_llist(enumerate_tree(llist(1, llist(2, llist(3, 4)), 5)))
```)

Now we can reformulate
#py("sum_odd_squares")
and
#py("even_fibs")
as in the signal-flow diagrams. For
#py("sum_odd_squares"),
we enumerate the sequence of leaves of the tree, filter this to keep only
the odd numbers in the sequence, square each element, and sum the results:
#idx("sumoddsquares", decl: true)
#snippet(```python
def sum_odd_squares(tree):
    return reduce(plus,
                  0,
                  map(square,
                      filter(is_odd,
                             enumerate_tree(tree))))
```)

For
#py("even_fibs"),
we enumerate the integers from 0 to $n$, generate
the Fibonacci number for each of these integers, filter the resulting
sequence to keep only the even elements, and accumulate the results
into a linked list:
#idx("evenfibs", decl: true)
#snippet(```python
def even_fibs(n):
    return reduce(pair,
                  None,
                  filter(is_even,
                         map(fib,
                             enumerate_interval(0, n))))
```)

The value of expressing programs as sequence operations is that this
helps us make program designs that are modular, that is, designs that
are constructed by combining relatively independent pieces. We can
encourage modular design by providing a library of standard components
together with a conventional interface for connecting the components
in flexible ways.

Modular construction
#idx("modularity")
#idx("sequence(s)", sub: "as source of modularity")
is a powerful strategy for controlling complexity in
engineering design. In real signal-processing applications, for example,
designers regularly build systems by cascading elements selected from
standardized families of filters and transducers. Similarly, sequence
operations provide a library of standard program elements that we can mix
and match. For instance, we can reuse pieces from the
#py("sum_odd_squares")
and
#py("even_fibs")
functions
in a program that constructs a linked list of the squares of the first
$n+1$ Fibonacci numbers:

#snippet(```python
def list_fib_squares(n):
    return reduce(pair,
                  None,
                  map(square,
                      map(fib,
                          enumerate_interval(0, n))))
```)

#snippet(```python
print_llist(list_fib_squares(10))
```)

#output(```python
print_llist(list_fib_squares(10))
```)

We can rearrange the pieces and use them in computing the product of the
squares of the odd integers in a sequence:

#snippet(```python
def product_of_squares_of_odd_elements(sequence):
    return reduce(times,
                  1,
                  map(square,
                      filter(is_odd, sequence)))
```)

#snippet(```python
print(product_of_squares_of_odd_elements(llist(1, 2, 3, 4, 5)))
```)

#output(```python
print(product_of_squares_of_odd_elements(llist(1, 2, 3, 4, 5)))
```)

We can also formulate conventional data-processing applications in terms of
sequence operations. Suppose we have a sequence of personnel records and
we want to find the salary of the highest-paid programmer. Assume that we
have a selector #py("salary") that returns the salary
of a record, and a predicate
#py("is_programmer")
that tests if a record is for a programmer. Then we can write

#snippet(```python
def salary_of_highest_paid_programmer(records):
    return reduce(max,
                  0,
                  map(salary,
                      filter(is_programmer, records)))
```)

These examples give just a hint of the vast range of operations that
can be expressed as sequence operations.#footnote[#idx("Waters, Richard C.")
Richard Waters (1979) developed a program that automatically analyzes
traditional
#idx("Fortran")
Fortran programs, viewing them in terms of maps, filters, and accumulations.
He found that fully 90 percent of the code in the Fortran Scientific
Subroutine Package fits neatly into this paradigm. One of the reasons
for the success of Lisp as a programming language is that linked lists provide a
standard medium for expressing ordered collections so that they can be
manipulated using higher-order operations. Many modern languages, such as
Python, have learned this lesson.]

Sequences, implemented here as linked lists, serve as a conventional interface
that permits us to combine processing modules. Additionally, when we
uniformly represent structures as sequences, we have localized the
data-structure dependencies in our programs to a small number of sequence
operations. By changing these, we can experiment with alternative
representations of sequences, while leaving the overall design of our
programs intact. We will exploit this capability in
section @sec:streams, when we generalize the
sequence-processing paradigm to admit infinite sequences.

#exercise(label-name: <ex:2_33>, [
Fill in the missing expressions to complete the following definitions of
some basic linked-list-manipulation operations as accumulations:

#idx("length", sub: "as accumulation")#idx("map", sub: "as accumulation")#idx("append", sub: "as accumulation")
#syntax("
def map(f, sequence):
    return reduce(lambda x, y: ", metaphrase[??], ",
                      None, sequence)
def append(seq1, seq2):
    return reduce(pair, ", metaphrase[??], ", ", metaphrase[??], ")
def length(sequence):
    return reduce(", metaphrase[??], ", 0, sequence)
      ")
])

#exercise(label-name: <ex:horner>, [
Evaluating a
polynomial in $x$ at a given value
of $x$ can be formulated as an accumulation.
We evaluate the polynomial

$ a_(n) x^(n) +a_(n-1)x^(n-1)+ dots.c + a_(1) x+a_(0) $

using a well-known algorithm called
#idx("polynomial(s)", sub: "evaluating with Horner's rule")
#idx("Horner's rule")
#emph[Horner's rule], which structures the computation as

$ lr(( dots.c (a_(n) x+a_(n-1))x+ dots.c +a_(1) )) x+a_(0) $

In other words, we start with $a_(n)$, multiply
by $x$, add $a_(n-1)$,
multiply by $x$, and so on, until we reach
$a_(0)$.#footnote[According to
#idx("Knuth, Donald E.")
Knuth (1997b), this rule was formulated by
#idx("Horner, W. G.", sort: "Horner")
W. G. Horner early in the nineteenth century, but the method was actually
used by Newton over a hundred years earlier. Horner's rule evaluates
the polynomial using fewer additions and multiplications than does the
straightforward method of first computing
$a_(n) x^(n)$, then adding
$a_(n-1)x^(n-1)$, and so on. In fact, it is
possible to prove that any algorithm for evaluating arbitrary polynomials
must use at least as many additions and multiplications as does
Horner's rule, and thus Horner's rule is an
#idx("algorithm", sub: "optimal")
#idx("optimality", sub: "of Horner's rule")
optimal algorithm for polynomial evaluation. This was proved (for the
number of additions) by
#idx("Ostrowski, A. M.")
A. M. Ostrowski in a 1954 paper that essentially founded the modern study
of optimal algorithms. The analogous statement for multiplications was
proved by
#idx("Pan, V. Y.")
V. Y. Pan in 1966. The book by
#idx("Borodin, Alan")
#idx("Munro, Ian")
Borodin and Munro (1975)
provides an overview of these and other results about optimal
algorithms.]
Fill in the following template to produce a
function
that evaluates a polynomial using Horner's rule. Assume that the
coefficients of the polynomial are arranged in a sequence, from
$a_(0)$ through
$a_(n)$.

#syntax("
def horner_eval(x, coefficient_sequence):
    return reduce(lambda this_coeff, higher_terms: ", metaphrase[??], ",
                  0,
                  coefficient_sequence)
      ")

For example, to compute $1+3x+5x^(3)+x^(5)$ at
$x=2$ you would evaluate

#snippet(```python
print(horner_eval(2, llist(1, 3, 0, 5, 0, 1)))
```)
])

#exercise(label-name: <ex:countleaves-as-accumulation>, [
Redefine
#py("count_leaves")
from section @sec:trees as an accumulation:
#idx("countleaves", sub: "as accumulation")
#syntax("
def count_leaves(t):
    return reduce(", metaphrase[??], ", ", metaphrase[??], ", map(", metaphrase[??], ", ", metaphrase[??], "))
          ")
])

#exercise(label-name: <ex:accumulate-n>, [
The
function
#py("reduce_n")
is similar to
#py("reduce")
except that it takes as its third argument a sequence of sequences, which
are all assumed to have the same number of elements. It applies the
designated accumulation
function
to combine all the first elements of the sequences, all the second elements
of the sequences, and so on, and returns a sequence of the results. For
instance, if #py("s") is a sequence containing four
sequences

#snippet(```python
llist(llist(1, 2, 3), llist(4, 5, 6), llist(7, 8, 9), llist(10, 11, 12))
```)

then the value of
#py("reduce_n(plus, 0, s)")
should be the sequence
#py("llist(22, 26, 30)").
Fill in the missing expressions in the following definition of
#py("reduce_n"):
#idx("reducen")
#syntax("
def reduce_n(op, init, seqs):
    return (None if is_none(head(seqs))
            else pair(reduce(op, init, ", metaphrase[??], "),
                      reduce_n(op, init, ", metaphrase[??], ")))
      ")
])

#exercise(label-name: <ex:matrix-ops>, [
Suppose we represent vectors $v=(v_(i))$ as
#idx("matrix, represented as sequence")
#idx("vector (mathematical)", sub: "represented as sequence")
#idx("vector (mathematical)", sub: "operations on")
sequences of numbers, and matrices $m=(m_(i j))$
as sequences of vectors (the rows of the matrix). For example, the matrix

$ lr([ mat(delim: #none, 1, 2, 3, 4; 4, 5, 6, 6; 6, 7, 8, 9) ]) $

is represented as the following sequence:

#snippet(```python
llist(llist(1, 2, 3, 4),
      llist(4, 5, 6, 6),
      llist(6, 7, 8, 9))
```)

With this representation, we can use sequence operations to concisely
express the basic matrix and vector operations. These operations
(which are described in any book on matrix algebra) are the following:

#sicp-table(columns: 2, [#py("dot_product(")$v$#py(",")$w$#py(")")], [returns the sum $sum_(i)v_(i) w_(i)$;], [#py("matrix_times_vector(")$m$#py(",")$v$#py(")")], [returns the vector $t$, where $t_(i) =sum_(j)m_(i j)v_(j)$;], [#py("matrix_times_matrix(")$m$#py(",")$n$#py(")")], [returns the matrix $p$, where $p_(i j)=sum_(k) m_(i k)n_(k j)$;], [#py("transpose(")$m$#py(")")], [returns the matrix $n$, where $n_(i j)=m_(j i)$.])

We can define the dot product as#footnote[This definition uses
the function #py("reduce_n") from exercise @ex:accumulate-n.]
#idx("dotproduct", decl: true)
#snippet(```python
def dot_product(v, w):
    return reduce(plus, 0, reduce_n(times, 1, llist(v, w)))
```)

Fill in the missing expressions in the following
functions
for computing the other matrix operations. (The
function
#py("reduce_n") is declared in
exercise @ex:accumulate-n.)
#idx("matrixtimesvector")#idx("matrixtimesmatrix")#idx("transpose a matrix")
#syntax("
def matrix_times_vector(m, v):
    return map(", metaphrase[??], ", m)
def transpose(mat):
    return reduce_n(", metaphrase[??], ", ", metaphrase[??], ", mat)
def matrix_times_matrix(m, n):
    cols = transpose(n)
    return map(", metaphrase[??], ", m)
      ")
])

#exercise(label-name: <ex:fold-right-left>, [
The
#idx("reduce", sub: "same as foldright")
#idx("foldright")
#py("reduce")
function
is also known as
#py("fold_right"),
because it combines the first element of the sequence with the result
of combining all the elements to the right. There is also a
#py("fold_left"),
which is similar to
#py("fold_right"),
except that it combines elements working in the opposite direction:
#idx("foldleft", decl: true)
#snippet(```python
def fold_left(op, initial, sequence):
    def iter(result, rest):
        return (result if is_none(rest)
                else iter(op(result, head(rest)),
                          tail(rest)))
    return iter(initial, sequence)
```)

What are the values of

#snippet(```python
print(fold_right(divide, 1, llist(1, 2, 3)))
```)

#snippet(```python
print(fold_left(divide, 1, llist(1, 2, 3)))
```)

#snippet(```python
print(fold_right(llist, None, llist(1, 2, 3)))
```)

#snippet(```python
print(fold_left(llist, None, llist(1, 2, 3)))
```)

Give a property that
#py("op")
should satisfy to guarantee that
#py("fold_right")
and
#py("fold_left")
will produce the same values for any sequence.
])

#exercise(label-name: <ex:2_39>, [
Complete the following definitions of #py("reverse")
#idx("reverse", sub: "as folding")
(exercise @ex:reverse) in terms of
#py("fold_right")
and
#py("fold_left")
from exercise @ex:fold-right-left:

#syntax("
def reverse(sequence):
    return fold_right(lambda x, y: ", metaphrase[??], ", None, sequence)
      ")

#syntax("
def reverse(sequence):
    return fold_left(lambda x, y: ", metaphrase[??], ", None, sequence)
      ")
])

#idx("sequence(s)", sub: "operations on")

#subheading([Nested Mappings])

#anchor(<sec:nested-mappings>)
#idx("mapping", sub: "nested")

We can extend the sequence paradigm to include many computations that are
commonly expressed using nested loops.#footnote[This approach to nested
mappings was shown to us by
#idx("Turner, David")
David Turner, whose languages
#idx("KRC")
KRC and
#idx("Miranda")
Miranda provide elegant formalisms for dealing with these constructs. The
examples in this section (see also
exercise @ex:8queens) are adapted from Turner 1981.
In section @sec:exploiting-streams, we'll see
how this approach generalizes to infinite sequences.]
Consider this problem: Given a positive integer
$n$, find all ordered pairs of distinct positive
integers $i$ and $j$,
where $1 lt.eq j < i lt.eq n$, such that
$i +j$ is prime. For example, if
$n$ is 6, then the pairs are the following:

$ mat(delim: #none, i, 2, 3, 4, 4, 5, 6, 6; j, 1, 2, 1, 3, 2, 1, 5; i+j, 3, 5, 5, 7, 7, 7, 11) $

A natural way to organize this computation is to generate the sequence
of all ordered pairs of positive integers less than or equal to
$n$, filter to select those pairs whose sum is
prime, and then, for each pair $(i, j)$ that
passes through the filter, produce the triple
$(i, j, i+j)$.

Here is a way to generate the sequence of pairs: For each integer
$i lt.eq n$, enumerate the integers
$j < i$, and for each such
$i$ and $j$
generate the pair $(i, j)$. In terms of
sequence operations, we map along the sequence
#py("enumerate_interval(1, n)").
For each $i$ in this sequence, we map along the
sequence
#py("enumerate_interval(1, i - 1)").
For each $j$ in this latter sequence, we
generate the pair
#py("llist(i, j)").
This gives us a sequence of pairs for each $i$.
Combining all the sequences for all the $i$ (by
accumulating with #py("append")) produces the
required sequence of pairs:#footnote[We're representing a pair here
as a linked list of two elements rather than as
an ordinary pair.
Thus, the "pair" $(i, j)$ is
represented as
#py("llist(i, j)"),
not
#py("pair(i, j)").]

#snippet(```python
print(reduce(append,
             None,
             map(lambda i: map(lambda j: llist(i, j),
                               enumerate_interval(1, i - 1)),
                 enumerate_interval(1, n))))
```)

The combination of mapping and accumulating with
#py("append") is so common in this sort of program
that we will isolate it as a separate
function:
#idx("flatmap", decl: true)
#snippet(```python
def flatmap(f, seq):
    return reduce(append, None, map(f, seq))
```)

Now filter this sequence of pairs to find those whose sum is prime. The
filter predicate is called for each element of the sequence; its argument
is a pair and it must extract the integers from the pair. Thus, the
predicate to apply to each element in the sequence is

#snippet(```python
def is_prime_sum(pair):
    return is_prime(head(pair) + head(tail(pair)))
```)

Finally, generate the sequence of results by mapping over the filtered
pairs using the following
function,
which constructs a triple consisting of the two elements of the pair along
with their sum:

#snippet(```python
def make_pair_sum(pair):
    return llist(head(pair), head(tail(pair)),
                 head(pair) + head(tail(pair)))
```)

Combining all these steps yields the complete
function:
#idx("primesumpairs", decl: true)
#snippet(```python
def prime_sum_pairs(n):
    return map(make_pair_sum,
               filter(is_prime_sum,
                  flatmap(lambda i: map(lambda j: llist(i, j),
                                        enumerate_interval(1, i - 1)),
                          enumerate_interval(1, n))))
```)

Nested mappings are also useful for sequences other than those that
enumerate intervals. Suppose we wish to generate all the
#idx("set", sub: "permutations of")
#idx("permutations of a set")
permutations
of a set $S$; that is, all the ways of ordering
the items in the set. For instance, the permutations of
$\{1, 2, 3\}$ are
$\{1, 2, 3\}$,
$\{ 1, 3, 2\}$,
$\{2, 1, 3\}$,
$\{ 2, 3, 1\}$,
$\{ 3, 1, 2\}$, and
$\{ 3, 2, 1\}$. Here is a plan for generating
the permutations of $S$: For each item
$x$ in $S$,
recursively generate the sequence of permutations of
$S-x$,#footnote[The set
$S-x$ is the set of all elements of
$S$, excluding
$x$.] and adjoin
$x$ to the front of each one. This yields, for
each $x$ in $S$, the
sequence of permutations of $S$ that begin
with $x$. Combining these sequences for
all $x$ gives all the permutations
of $S$:#footnote[#idx("# (for comments in programs)", sort: "0a5")
#idx("number sign (# for comments in programs)")
#idx("comments in programs")
#idx("program", sub: "comments in")
The character #py("#") in Python programs is
used to introduce #emph[comments]. Everything from
#py("#")
to the end of the line is ignored by the interpreter. In this book we
don't use many comments; we try to make our programs self-documenting
by using descriptive names.]
#idx("permutations of a set", sub: "permutations", decl: true)

#snippet(```python
def permutations(s):
    return (llist(None)       # sequence containing empty set
            if is_none(s)     # empty set?
            else flatmap(lambda x: map(lambda p: pair(x, p),
                                       permutations(remove(x, s))),
                         s))
```)

Notice how this strategy reduces the problem of generating permutations of
$S$ to the problem of generating the
permutations of sets with fewer elements than
$S$. In the terminal case, we work our way down
to the empty linked list, which represents a set of no elements. For this, we
generate
#py("llist(None)"),
which is a sequence with one item, namely the set with no elements. The
#py("remove")
function
used in #py("permutations") returns all the items in
a given sequence except for a given item. This can be expressed as a
simple filter:
#idx("remove", decl: true)
#snippet(```python
def remove(item, sequence):
    return filter(lambda x: x != item,
                  sequence)
```)

#exercise(label-name: <ex:2_40>, [
Write a
function
#idx("uniquepairs")
#py("unique_pairs")
that, given an integer $n$, generates the
sequence of pairs $(i, j)$ with
$1 lt.eq j < i lt.eq n$. Use
#py("unique_pairs")
to simplify the definition of
#py("prime_sum_pairs")
given above.
])

#exercise(label-name: <ex:2_41>, [
Write a
function
to find all ordered triples of distinct positive integers
$i$, $j$,
and $k$ less than or equal to a given
integer $n$ that sum to a given integer
$s$.
])

#sicp-figure(image("/images/img_original/ch2-Z-G-23.svg", width: 70%), caption: [A solution to the eight-queens puzzle.], label-name: <fig:8queens>)

#exercise(label-name: <ex:8queens>, [
The
#idx("eight-queens puzzle")
#idx("chess, eight-queens puzzle")
#idx("puzzles", sub: "eight-queens puzzle")
"eight-queens puzzle" asks how to place eight queens on a
chessboard so that no queen is in check from any other (i.e., no two
queens are in the same row, column, or diagonal). One possible solution
is shown in figure @fig:8queens. One way to solve the
puzzle is to work across the board, placing a queen in each column.
Once we have placed $k-1$ queens, we must place
the $k$th queen in a position where it does not
check any of the queens already on the board. We can formulate this
approach recursively: Assume that we have already generated the sequence
of all possible ways to place $k-1$ queens in
the first $k-1$ columns of the board. For
each of these ways, generate an extended set of positions by placing a
queen in each row of the $k$th column. Now
filter these, keeping only the positions for which the queen in the
$k$th column is safe with respect to the other
queens. This produces the sequence of all ways to place
$k$ queens in the first
$k$ columns. By continuing this process, we
will produce not only one solution, but all solutions to the puzzle.

We implement this solution as a
function
#py("queens"),
which returns a sequence of all solutions to the problem of placing
$n$ queens on an
$n times n$ chessboard.
The function #py("queens")
has an internal
function
#py("queens_cols")
that returns the sequence of all ways to place queens in the first
$k$ columns of the board.

#idx("queens", decl: true)
#snippet(```python
def queens(board_size):
    def queen_cols(k):
        return (llist(empty_board) if k == 0
                else filter(lambda positions: is_safe(k, positions),
                            flatmap(lambda rest_of_queens:
                                      map(lambda new_row:
                                            adjoin_position(new_row, k,
                                                            rest_of_queens),
                                          enumerate_interval(1, board_size)),
                                    queen_cols(k - 1))))
    return queen_cols(board_size)
```)

In this
function
#py("rest_of_queens")
is a way to place $k-1$ queens in the first
$k-1$ columns, and
#py("new_row")
is a proposed row in which to place the queen for the
$k$th column. Complete the program by
implementing the representation for sets of board positions, including the
function
#py("adjoin_position"),
which adjoins a new row-column position to a set of positions, and
#py("empty_board"),
which represents an empty set of positions. You must also write the
function
#py("is_safe"),
which determines for a set of positions whether the queen in the
$k$th column is safe with respect to the others.
(Note that we need only check whether the new queen is safe—the
other queens are already guaranteed safe with respect to each other.)
])

#exercise(label-name: <ex:2_43>, [
Louis Reasoner is having a terrible time doing
exercise @ex:8queens. His
#py("queens")
function
seems to work, but it runs extremely slowly. (Louis never does manage to
wait long enough for it to solve even the
$6 times 6$ case.) When Louis asks Eva Lu Ator
for help, she points out that he has interchanged the order of the nested
mappings in the
#py("flatmap"),
writing it as

#snippet(```python
flatmap(lambda new_row:
          map(lambda rest_of_queens:
                adjoin_position(new_row, k, rest_of_queens),
              queen_cols(k - 1)),
        enumerate_interval(1, board_size))
```)

Explain why this interchange makes the program run slowly. Estimate
how long it will take Louis's program to solve the eight-queens
puzzle, assuming that the program in
exercise @ex:8queens solves the puzzle in time
$T$.
])

#idx("sequence(s)", sub: "as conventional interface")
#idx("conventional interface", sub: "sequence as")
#idx("mapping", sub: "nested")
