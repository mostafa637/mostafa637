// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Running the Evaluator as a Program], label-name: <sec:running-eval>)

#idx("metacircular evaluator for Python", sub: "running")

Given the evaluator, we have in our hands a description
(expressed in Python)
of the process by which
Python statements and expressions
are evaluated. One advantage of expressing the evaluator as a program is
that we can run the program. This gives us, running within
Python,
a working model of how
Python
itself evaluates expressions. This can serve as a framework for
experimenting with evaluation rules, as we shall do later in this chapter.

Our evaluator program reduces expressions ultimately to the application of
#idx("metacircular evaluator for Python", sub: "primitive functions")
primitive
functions.
Therefore, all that we need to run the evaluator is to create a mechanism
that calls on the underlying
Python
system to model the application of primitive
functions.

There must be a binding for each primitive
function
name and operator, so that when
#py("evaluate")
evaluates the
function expression
of an application of a primitive, it will find an
object to pass to #py("apply"). We thus set up a
#idx("metacircular evaluator for Python", sub: "global environment")
#idx("global environment", sub: "in metacircular evaluator")
global environment that associates unique objects with the names of the
primitive
functions and operators
that can appear in the expressions we will be evaluating.
#idx("symbol(s)", sub: "in global environment")

The global environment also includes bindings for
#idx("metacircular evaluator for Python", sub: "None")
#py("None")
and other names,

so that they can be used as constants in expressions to be evaluated.

#idx("setupenvironment", decl: true)
#snippet(```python
def setup_environment():
    return extend_environment(append(primitive_function_symbols, primitive_constant_symbols), append(primitive_function_objects, primitive_constant_values), the_empty_environment)
```)

#idx("theglobalenvironment", decl: true)
#snippet(```python
the_global_environment = setup_environment()
```)

It does not matter how we represent primitive function objects, so long
as #py("apply") can identify and apply them using
the functions #py("is_primitive_function")
and #py("apply_primitive_function"). We
have chosen to represent a primitive function as a list beginning with
the string #py("\"primitive\"") and
containing a function in the underlying JavaScript that implements that
primitive.
#idx("isprimitivefunction", decl: true)#idx("primitiveimplementation", decl: true)
#snippet(```python
def is_primitive_function(fun):
    return is_tagged_list(fun, "primitive")

def primitive_implementation(fun):
    return head(tail(fun))
```)

The function #py("setup_environment")
will get the primitive names and implementation
functions
from a list:#footnote[Any
function
defined in the underlying
Python
can be used as a primitive for the metacircular evaluator. The name of a
primitive installed in the evaluator need not be the same as the name of its
implementation in the underlying
Python
the names are the same here because the metacircular evaluator implements
Python
itself.
Thus, for example, we could put
#py("llist(\"first\", head)")
or
#py("llist(\"square\", lambda x: x * x)")
in the list of
#py("primitive_functions").]
#idx("primitivefunctionsymbols", decl: true)#idx("primitivefunctionobjects", decl: true)
#syntax("
primitive_functions = llist(llist(\"head\",    head             ),
                             llist(\"tail\",    tail             ),
                             llist(\"pair\",    pair             ),
                             llist(\"is_null\", is_none          ),
                             llist(\"+\",       lambda x, y: x + y  ),
                             ", metaphrase[more primitive functions], "
                            )

primitive_function_symbols = \\
    map(lambda f: head(f), primitive_functions)

primitive_function_objects = \\
    map(lambda f: llist(\"primitive\", head(tail(f))),
        primitive_functions)
      ")

Similar to primitive functions, we define other primitive constants that are
installed in the global environment by the function
#py("setup_environment").

#syntax("
primitive_constants = llist(llist(\"undefined\", None),
                             llist(\"math_PI\",   math_pi)
                             ", metaphrase[more primitive constants], "
                            )

primitive_constant_symbols = \\
    map(lambda c: head(c), primitive_constants)

primitive_constant_values = \\
    map(lambda c: head(tail(c)), primitive_constants)
	  ")

To apply a
primitive function,
we simply apply the implementation
function
to the arguments, using the underlying
Python
system:#footnote[#anchor(<foot:vector-array>)
Python's #py("apply") method
expects the function arguments in a #emph[vector]. (Vectors
are called "arrays" in Python.)
Thus, the #py("arglist") is transformed into
a
#idx("vector (data structure)", sub: "for arguments of apply")
vector—here using a while
loop (see exercise @ex:while_loop):
#idx("applyinunderlyingjavascript", decl: true)#idx("apply (primitive method)")
#syntax("
def apply_in_underlying_javascript(prim, arglist):
    arg_vector = []
    # empty vector
    i = 0
    while  not is_none(arglist):
        arg_vector[i] = head(arglist)
        # store value at index ", $mono("i")$, "
        i = i + 1
        arglist = tail(arglist)
    return prim.apply(prim, arg_vector)
    # ", $mono("apply")$, " is accessed via ", $mono("prim")$)

We also made use of
#py("apply_in_underlying_javascript")
to declare the function
#py("apply_generic")
in section @sec:data-directed.]
#idx("applyprimitivefunction", decl: true)
#snippet(```python
def apply_primitive_function(fun, arglist):
    return apply_in_underlying_javascript(primitive_implementation(fun), arglist)
```)

#idx("metacircular evaluator for Python", sub: "primitive functions")

For convenience in running the metacircular evaluator, we provide a
#idx("metacircular evaluator for Python", sub: "driver loop")
#idx("driver loop", sub: "in metacircular evaluator")
#emph[driver loop] that models the read-evaluate-print loop of
the underlying JavaScript system. It prints a
#idx("prompts")
#idx("prompts", sub: "metacircular evaluator")
#emph[prompt] and reads an input program as a string.
It transforms the program string
into a tagged-list representation of the statement as described in
section @sec:representing-expressions—a
process called parsing and accomplished by the primitive function
#py("parse").
We precede each printed result by
an #emph[output prompt] so as to distinguish the value of the
program from other output that may be printed. The driver loop gets
the program environment of the previous program as argument.
As described at the end of section @sec:env-internal-def, the
driver loop treats the program as if it were in a block: It
scans out the declarations, extends the given environment by a frame
containing a binding of each name to
#py("\"*unassigned*\""), and evaluates
the program with respect to the extended environment, which
is then passed as argument to the next iteration of the driver loop.
#idx("driverloop", sub: "for metacircular evaluator", decl: true)
#snippet(```python
input_prompt = "M-evaluate input: "
output_prompt = "M-evaluate value: "
def driver_loop(env):
    input = user_read(input_prompt)
    if is_none(input):
        print("evaluator terminated")
    else:
        program = parse(input)
        locals = scan_out_declarations(program)
        unassigneds = list_of_unassigned(locals)
        program_env = extend_environment(locals, unassigneds, env)
        output = evaluate(program, program_env)
        user_print(output_prompt, output)
        return driver_loop(program_env)
```)

We use Python's #py("prompt") function
to request and read the input string from the user:
#idx("userread", decl: true)
#snippet(```python
def user_read(prompt_string):
    return prompt(prompt_string)
```)

The function
#idx("prompt (primitive function)")

#py("prompt") returns
#py("null") when the user cancels the
input. We use a special printing
function #py("user_print"),
to avoid printing the environment part of a compound
function,
which may be a very long list (or may even contain cycles).
#idx("userprint", decl: true)
#snippet(```python
def user_print(string, object):
    def prepare(object):
        return "< compound-function >" if is_compound_function(object) else "< primitive-function >" if is_primitive_function(object) else pair(prepare(head(object)), prepare(tail(object))) if is_pair(object) else object
    print(string + " " + stringify(prepare(object)))
```)

Now all we need to do to run the evaluator is to initialize the global
environment and start the driver loop. Here is a sample interaction:

#snippet(```python
the_global_environment = setup_environment()
driver_loop(the_global_environment)
```)

#prompt(```python
M-evaluate input:
```)

#snippet(```python
def append(xs, ys):
    return ys if is_none(xs) else pair(head(xs), append(tail(xs), ys))
```)

#output(```python
def append(xs, ys):
    return ys if is_none(xs) else pair(head(xs), append(tail(xs), ys))
```)

#prompt(```python
M-evaluate input:
```)

#snippet(```python
append(llist("a", "b", "c"), llist("d", "e", "f"))
```)

#output(```python
append(llist("a", "b", "c"), llist("d", "e", "f"))
```)

#exercise(label-name: <ex:mceval-map>, [
Eva Lu Ator and Louis Reasoner are each experimenting with the
metacircular evaluator. Eva types in the definition of
#py("map"), and runs some test programs that use it.
They work fine. Louis, in contrast, has installed the system version of
#py("map") as a primitive for the metacircular
evaluator. When he tries it, things go terribly wrong. Explain why
Louis's #py("map") fails even though
Eva's works.
])

#idx("metacircular evaluator for Python", sub: "running")
