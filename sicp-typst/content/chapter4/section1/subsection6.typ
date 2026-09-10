// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Internal Declarations], label-name: <sec:internal-definitions>)

#idx("block structure")

#idx("internal declaration", sub: "scope of name")
#idx("scope of a name", sub: "internal declaration")

In Python, the scope of a declaration
is the entire block that immediately surrounds the declaration,
not just the portion of the block starting at the point where
the declaration occurs.
This section takes a closer look at this design choice.

Let us revisit the pair of mutually recursive functions
#py("is_even") and
#py("is_odd") from
Section @sec:env-internal-def,
declared locally
in the body of a function #py("f").

#syntax("
def f(x):
    def is_even(n):
        return True if n == 0 else is_odd(n - 1)
    def is_odd(n):
        return False if n == 0 else is_even(n - 1)
    return is_even(x)
")

Our intention here is that the name
#py("is_odd")
in the body of the function #py("is_even")
should refer to the function #py("is_odd")
that is declared after #py("is_even").
The scope of the name #py("is_odd") is the
entire body block of #py("f"), not just the portion of
the body of #py("f") starting at the point where
the declaration of #py("is_odd")
occurs. Indeed, when we consider that
#py("is_odd") is itself defined in terms of
#py("is_even")—so that
#py("is_even") and
#py("is_odd") are mutually recursive
functions—we see that the only	satisfactory interpretation of
the two declarations is to regard them as if the names
#py("is_even") and
#py("is_odd")
were being added to the environment simultaneously. More generally, in
block structure, the scope of a local name is the entire block
in which the declaration is evaluated.

The evaluation of blocks in the metacircular evaluator of
section @sec:core-of-evaluator achieves such
a simultaneous scope for local names by
#idx("scanning out declarations", sub: "in metacircular evaluator")
#idx("internal declaration", sub: "scanning out")
scanning out the declarations in the block and extending the current
environment with a frame containing bindings for all the declared
names before evaluating the declarations. Thus the new environment
in which the block body is evaluated already contains
bindings for
#py("is_even") and
#py("is_odd"), and any occurrence
of one of these names refers to the correct binding. Once their
declarations are evaluated,
these names are bound to their declared values, namely function
objects that have the extended environment as their environment
part. Thus, for example,
by the time
#py("is_even")
gets applied in the body of
#py("f"), its environment
already contains the correct binding for the symbol
#py("is_odd"), and
the evaluation of the name
#py("is_odd") in the body of
#py("is_even") retrieves the correct
value.

#exercise(label-name: <ex:4_16>, [
Consider the function #py("f_3")
of section @sec:lambda:

#snippet(```python
def f_3(x, y):
    a = 1 + x * y
    b = 1 - y
    return x * square(a) + y * b + a * b
```)

+ Draw a diagram of the environment in effect during evaluation of the return expression of #py("f_3").
+ When evaluating a function application, the evaluator creates two frames: one for the parameters and one for the names declared #emph[directly] in the function's body block, as opposed to in an inner block. Since all these names have the same scope, an implementation could combine the two frames. Change the evaluator such that the evaluation of the body block does not create a new frame. You may assume that this will not result in duplicate names in the frame (exercise @ex:directly justifies this).
])

#exercise(label-name: <ex:hoisting>, [
Eva Lu Ator is writing programs in which
function definitions and other statements are interleaved.
She needs to make sure that the declarations are evaluated before
the functions are applied. She complains: "Why can't the evaluator take care of this chore, and #idx("hoisting of function definitions") #idx("function definition", sub: "hoisting of") hoist all function declarations to the beginning of the block in which they appear? Function definitions outside of blocks should be hoisted to the beginning of the program."

+ Modify the evaluator following Eva's suggestion.
+ The designers of Python decided to follow Eva's approach. Discuss this decision.
+ In addition, the designers of Python decided to allow the name declared by a function definition to be reassigned using assignment. Modify your solution accordingly and discuss this decision.
])

#exercise(label-name: <ex:lambda_calculus>, [
Recursive functions are obtained in a
#idx("recursive function", sub: "specifying without declaration")
roundabout way in our
interpreter: First declare the name that will refer to the recursive
function and assign to it the special value
#py("\"*unassigned*\""); then define the
recursive function in the scope of that name; and finally assign the
defined function to the name. By the time the recursive function gets
applied, any occurrences of the name in the body properly refer to
the recursive function. Amazingly, it is possible to specify recursive
functions without using declarations or assignment. The following program computes
10 factorial by applying a recursive
#idx("factorial", sub: "without declaration or assignment")
factorial function:#footnote[This example illustrates a programming
trick for formulating recursive functions without using assignment. The
most general trick of this sort is the
#idx("Y operator", sort: "Y")
#idx("Scheme", sub: "Y operator written in")
$Y$
#emph[operator], which can be used to give a "pure $lambda$-calculus" implementation of
recursion. (See
#idx("Stoy, Joseph E.")#idx("Gabriel, Richard P.")
Stoy 1977 for details on the lambda
calculus, and Gabriel 1988 for an exposition of the
$Y$ operator in the language
Scheme.)]

#snippet(```python
(lambda n: ((lambda fact: (fact(fact, n))) (lambda ft, k: (1 if k == 1 else k * ft(ft, k - 1)))))(10)
```)

+ Check (by evaluating the expression) that this really does compute factorials. Devise an analogous expression for computing Fibonacci numbers.
+ Consider the function #py("f") given above: #syntax(" def f(x): def is_even(n): return True if n == 0 else is_odd(n - 1) def is_odd(n): return False if n == 0 else is_even(n - 1) return is_even(x) ") Fill in the missing expressions to complete an alternative declaration of #py("f"), which has no internal function definitions: #syntax(" def f(x): return (lambda is_even, is_odd: is_even(is_even, is_odd, x))( lambda is_ev, is_od, n: True if n == 0 else is_od(", metaphrase[??], ", ", metaphrase[??], ", ", metaphrase[??], "), lambda is_ev, is_od, n: False if n == 0 else is_ev(", metaphrase[??], ", ", metaphrase[??], ", ", metaphrase[??], ")) ")
])

#subheading([Sequential Declaration Processing])

#idx("sequential declaration processing vs. scanning out")
#idx("scanning out declarations", sub: "sequential declaration processing vs.")
#anchor(<add_binding_to_frame>)
The design of our evaluator of
section @sec:core-of-evaluator imposes a
runtime burden on the evaluation of blocks: It needs to scan
the body of the block for locally declared names, extend the
current environment with a new frame that binds those names, and evaluate the
block body in this extended environment. Alternatively, the evaluation
of a block could extend the current environment with an empty frame.
The evaluation of each declaration in the block body would then add
a new binding to that frame.
To implement this design, we first simplify
#py("eval_block"):

#snippet(```python
def eval_block(component, env):
    body = block_body(component)
    return evaluate(body, extend_environment(None, None, env))
```)

The function
#py("eval_declaration") can no
longer assume that the environment already has a binding for
the name.
Instead of using
#py("assign_symbol_value") to
change an existing binding, it calls a new function,
#py("add_binding_to_frame"), to
add to the first frame of the environment a binding of the name
to the value of the value expression.

#snippet(```python
def eval_declaration(component, env):
    add_binding_to_frame(declaration_symbol(component), evaluate(declaration_value_expression(component), env), first_frame(env))
    return None
def add_binding_to_frame(symbol, value, frame):
    set_head(frame, pair(symbol, head(frame)))
    set_tail(frame, pair(value, tail(frame)))
```)

With sequential declaration processing, the scope of a
declaration is no longer the entire block that immediately surrounds
the declaration, but rather just the portion of the block starting at
the point where the declaration occurs.
Although we no longer have simultaneous scope, sequential
declaration processing
will evaluate calls to the function
#py("f") at the beginning of this section
correctly, but for an
"accidental" reason: Since the declarations
of the internal functions come first, no calls to these functions
will be evaluated until all of them have been declared. Hence,
#py("is_odd") will have been declared by the time
#py("is_even") is executed. In fact,
sequential declaration processing
will give the same result as our scanning-out-names evaluator in
section @sec:core-of-evaluator
for any function
in which the
#idx("internal declaration", sub: "restrictions on")
internal declarations come first in a body and evaluation of the value
expressions for the declared names doesn't actually use any of
the declared names.
Exercise @ex:simultaneous-def shows
an example of a function that doesn't
obey these restrictions, so that the alternative evaluator isn't
equivalent to our scanning-out-names evaluator.

Sequential declaration processing is more efficient and easier to
implement than scanning out names.
However, with sequential processing, the
declaration to which a name refers may depend on the order in which
the statements in a block are evaluated.
In exercise @ex:simultaneous-def, we see that
views may differ as to whether that is desirable.

#idx("sequential declaration processing vs. scanning out")
#idx("scanning out declarations", sub: "sequential declaration processing vs.")

#exercise(label-name: <ex:simultaneous-def>, [
Ben Bitdiddle, Alyssa P. Hacker, and Eva Lu Ator are arguing about
the desired result of evaluating the program

#snippet(```python
a = 1
def f(x):
    b = a + x
    a = 5
    return a + b
f(10)
```)

Ben asserts that the result should be obtained using the sequential
processing of declarations:
#py("b") is declared to be 11, then
#py("a") is declared to be 5, so the result is 16.
Alyssa objects that mutual recursion requires the simultaneous scope
rule for internal function definitions, and that it is unreasonable to
treat function names differently from other names. Thus, she argues for
the mechanism implemented in
section @sec:core-of-evaluator. This would
lead to #py("a") being unassigned at the time that
the value for #py("b") is to be computed. Hence,
in Alyssa's view the function should produce an error. Eva has a
third opinion. She says that if the declarations of
#py("a") and #py("b")
are truly meant to be simultaneous, then the value 5 for
#py("a") should be used in evaluating
#py("b"). Hence, in Eva's view
#py("a") should be 5,
#py("b") should be 15, and the result should be 20.
Which (if any) of these	viewpoints do you support? Can you devise a way
to implement internal declarations so that they behave as Eva
prefers?#footnote[The designers of Python support Alyssa on the
following grounds: Eva is in principle correct—the declarations
should be regarded as simultaneous. But it seems difficult to implement
a general, efficient mechanism that does what Eva requires. In the
absence of such a mechanism, it is better to generate an error in the
difficult cases of simultaneous declarations (Alyssa's notion) than
to produce an incorrect answer (as Ben would have it).]
])

#idx("block structure")
#idx("internal declaration", sub: "scope of name")
#idx("scope of a name", sub: "internal declaration")
