// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Lexical Addressing], label-name: <sec:lexical-addressing>)

#idx("compiler for Python", sub: "lexical addressing")
#idx("lexical addressing")

One of the most common optimizations performed by compilers is the
optimization of
name
lookup. Our compiler, as we have implemented it so far, generates code that
uses the
#py("lookup_symbol_value")
operation of the evaluator machine.
This searches for a
name
by comparing
it
with each
name
that is currently bound, working frame
by frame outward through the runtime environment. This search can be
expensive if the frames are deeply nested or if there are many
names.
For example, consider the problem of looking up the value
of #py("x") while evaluating the expression
#py("x * y * z")
in an application of the
function of five arguments
that is returned by

#snippet(```python
((x, y) =>
   (a, b, c, d, e) =>
     ((y, z) => x * y * z)(a * b * x, c + d + x))(3, 4)
```)

Each time
#py("lookup_symbol_value")
searches for #py("x"), it must determine that
the symbol
#py("\"x\"") is not equal to #py("\"y\"") or #py("\"z\"") (in the first frame), nor to #py("\"a\""), #py("\"b\""), #py("\"c\""), #py("\"d\""), or #py("\"e\"") (in the second frame).

Because our language is
#idx("lexical scoping", sub: "environment structure and")
lexically scoped, the runtime environment for any
component
will have a
structure that parallels the lexical structure of the program in which
the
component
appears.
Thus, the compiler can know, when it analyzes the
above expression,
that each time the
function
is applied the
binding for #py("x")
in
#py("x * y * z")
will be found two frames out from the
current frame and will be the first
binding
in that frame.

We can exploit this fact by inventing a new kind of
name-lookup
operation,
#py("lexical_address_lookup"),
that takes as arguments an environment and a
#idx("lexical addressing", sub: "lexical address")
#emph[lexical address] that
consists of two numbers: a #emph[frame number], which specifies how many
frames to pass over, and a #emph[displacement number], which specifies
how many
bindings
to pass over in that frame.
#idx("lexicaladdresslookup")
The operation #py("lexical_address_lookup")
will produce the value of the
name
stored at that lexical address
relative to the current environment. If we add the
#py("lexical_address_lookup")
operation to our machine, we can make the compiler generate code that
references
names
using this operation, rather than
#py("lookup_symbol_value").
Similarly, our compiled code can use a new
#idx("lexicaladdressassign")
#py("lexical_address_assign")
operation instead of
#py("assign_symbol_value").

With lexical addressing, there is no need to include any
symbolic references to names in the object code,
and frames do not need to include symbols at run time.

In order to generate such code, the compiler must be able to determine
the lexical address of a
name
it is about to compile a reference
to. The lexical address of a
name
in a program depends on where
one is in the code. For example, in the following program, the
address of #py("x") in expression
$e_(1)$ is (2,0)—two frames back
and the first
name
in the frame. At that point
#py("y") is at
address (0,0) and #py("c") is at address (1,2).
In expression
$e_(2)$,
#py("x") is at (1,0),
#py("y") is at (1,1), and
#py("c") is at (0,2).

#syntax("
((x, y) =>
   (a, b, c, d, e) =>
     ((y, z) => ", $e_(1)$, ")(", $e_(2)$, ", c + d + x))(3, 4);
      ")

One way for the compiler to produce code that uses lexical addressing
is to maintain a data structure called a
#idx("compile-time environment")
#emph[compile-time environment]. This keeps track of which
bindings
will be at which
positions in which frames in the runtime environment when a
particular
name-access
operation is executed. The compile-time
environment is a list of frames, each containing a list of
symbols.
There will be no values associated with the symbols, since values are not computed at compile time. (Exercise @ex:constant-optimisations will change this, as an optimization for constants.)
The compile-time
environment becomes an additional argument to
#py("compile") and is
passed along to each code generator. The top-level call to
#py("compile") uses
a compile-time-environment that includes the names of all primitive functions and primitive values.
When
the body of a lambda expression
is compiled,
#py("compile_lambda_body")
extends the compile-time environment by a frame containing the
function's
parameters, so that the

body is compiled with that extended environment.

Similarly, when
the body of a block
is compiled,
#py("compile_block")
extends the compile-time environment by a frame containing the
#idx("compiler for Python", sub: "scanning out internal declarations")
scanned-out local names of the body.

At each point in the compilation,
#py("compile_name")
and
#py("compile_assignment_declaration")
use the compile-time
environment in order to generate the appropriate lexical addresses.

Exercises @ex:lexical-address-start
through @ex:impl-lex-addr describe how to
complete this sketch of the lexical-addressing strategy in order to
incorporate lexical lookup into the compiler.
Exercises @ex:lexical-constants
and @ex:constant-optimisations
describe other uses for	the compile-time environment.

#idx("compiler for Python", sub: "lexical addressing")
#idx("lexical addressing")

#exercise(label-name: <ex:lexical-address-start>, [
Write a
function
#idx("lexicaladdresslookup")
#idx("lexicaladdressassign")
#py("lexical_address_lookup")
that implements the new lookup operation. It should take two
arguments—a lexical address and a runtime environment—and
return the value of the
name
stored at the specified lexical address.
The function #py("lexical_address_lookup")
should signal an error if the value
of the name
is the
string #py("\"*unassigned*\"").

Also write a
function
#py("lexical_address_assign")
that implements the operation that changes the value
of the name
at a specified lexical address.

#anchor(<ex:lexical-address-lookup>)
])

#exercise(label-name: <ex:5_43>, [
Modify the compiler to maintain the
#idx("compile-time environment")
compile-time environment as
described above. That is, add a compile-time-environment argument to
#py("compile") and the various code generators, and
extend it in
#py("compile_lambda_body") and #py("compile_block").
])

#exercise(label-name: <ex:find-variable>, [
Write a
function
#py("find_symbol")
that takes as arguments a
symbol
and a
#idx("compile-time environment")
compile-time environment and
returns the lexical address of the
symbol
with respect to that
environment. For example, in the program fragment that is shown above, the
compile-time environment during the compilation of expression
$e_(1)$ is

#snippet(```python
list(list("y", "z"),
     list("a", "b", "c", "d", "e"),
     list("x", "y"))
```)

The function #py("find_symbol")
should produce

#snippet(```python
find_symbol("c", list(list("y", "z"),
                      list("a", "b", "c", "d", "e"),
                      list("x", "y")));
```)

#output(```python
find_symbol("c", list(list("y", "z"),
                      list("a", "b", "c", "d", "e"),
                      list("x", "y")));
```)

#snippet(```python
find_symbol("x", list(list("y", "z"),
                      list("a", "b", "c", "d", "e"),
                      list("x", "y")));
```)

#output(```python
find_symbol("x", list(list("y", "z"),
                      list("a", "b", "c", "d", "e"),
                      list("x", "y")));
```)

#snippet(```python
find_symbol("w", list(list("y", "z"),
                      list("a", "b", "c", "d", "e"),
                      list("x", "y")));
```)

#output(```python
find_symbol("w", list(list("y", "z"),
                      list("a", "b", "c", "d", "e"),
                      list("x", "y")));
```)
])

#exercise(label-name: <ex:impl-lex-addr>, [
Using
#py("find_symbol")
from exercise @ex:find-variable,
rewrite
#py("compile_assignment_declaration") and #py("compile_name")
to output lexical-address instructions.

In cases where #py("find_symbol")
returns #py("\"not found\"")
(that is, where the name is not in the compile-time environment),
you should report a compile-time error.

Test the modified compiler on a few simple cases, such as the nested
lambda
combination at the beginning of this section.
])

#exercise(label-name: <ex:lexical-constants>, [
In Python, an attempt to assign a new value to a name that is declared
as a
#idx("constant (in Python)", sub: "detecting assignment to")
constant leads to an error.
Exercise @ex:mutable shows how to
detect such errors at run time. With the techniques presented in this
section, we can detect attempts to assign a new value to a constant
#emph[at compile time]. For this purpose, extend the functions
#py("compile_lambda_body") and
#py("compile_block")
to record in the compile-time environment whether a name is declared as a variable (using
#py("let") or as a parameter), or as
a constant (using
#py("const")
or
#py("function")).
Modify #py("compile_assignment")
to report an appropriate error when it detects an
assignment to a	constant.
])

#exercise(label-name: <ex:constant-optimisations>, [
Knowledge about constants at compile time opens the door to many
optimizations that allow us to generate more efficient object code. In
addition to the extension of the
#idx("compile-time environment")
compile-time environment in
exercise @ex:lexical-constants to indicate names
declared as constants, we may store the
value of a constant if it is known at compile time, or other information
that can help us optimize the code.

+ A constant declaration such as #py("const") #meta("name") #py("=") #meta("literal")#py(";") allows us to replace all occurrences of #meta("name") within the scope of the declaration by #meta("literal") so that #meta("name") doesn't have to be looked up in the runtime environment. This optimization is called #emph[constant propagation]. Use an extended compile-time environment to store literal constants, and modify #py("compile_name") to use the stored constant in the generated #py("assign") instruction instead of the #py("lookup_symbol_value") operation.
+ Function definition is a derived component that expands to constant declaration. Let us assume that the names of primitive functions in the global environment are also considered constants. If we further extend our compile-time environment to keep track of which names refer to compiled functions and which ones to primitive functions, we can move the test that checks whether a function is compiled or primitive from run time to compile time. This makes the object code more efficient because it replaces a test that must be performed once per function application in the generated code by one that is performed by the compiler. Using such an extended compile-time environment, modify #py("compile_function_call") so that if it can be determined at compile time whether the called function is compiled or primitive, only the instructions in the #py("compiled_branch") or the #py("primitive_branch") are generated.
+ Replacing constant names with their literal values as in part (a) paves the way for another optimization, namely replacing applications of primitive functions to literal values with the compile-time computed result. This optimization, called #emph[constant folding], replaces expressions such as #py("40 + 2") by #py("42") by performing the addition in the compiler. Extend the compiler to perform constant folding for arithmetic operations on numbers and for string concatenation. #anchor(<foot:ex-ast-transformations>)
])
