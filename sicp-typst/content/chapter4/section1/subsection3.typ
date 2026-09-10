// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Evaluator Data Structures], label-name: <sec:eval-data-structures>)

In addition to defining the
representation of components,
the evaluator implementation must also define the data structures that the
evaluator manipulates internally, as part of the execution of a
program, such as the representation of
functions
and environments and the representation of true and false.

#subheading([Testing of predicates])

#idx("metacircular evaluator for Python", sub: "representation of true and false")

In order to limit the predicate expressions of conditionals to proper
predicates (expressions that evaluate to a boolean value) as we do throughout
this book, we insist here that the function
#idx("metacircular evaluator for Python", sub: "representation of true and false")
#py("is_truthy") gets applied only to
boolean values, and we accept only the boolean value
#py("true") to be truthy.
The opposite of
#py("is_truthy") is called
#py("is_falsy").#footnote[Conditionals

in full Python accept #emph[any] value, not just a boolean,
as the result of evaluating the
"predicate" expression. Python's notion of
truthiness and falsiness is captured by the following variants of
#py("is_truthy") and
#py("is_falsy"):
#idx("truthiness")
#idx("falsiness")
#idx("isboolean")#idx("istruthy", sub: "full Python version", decl: true)#idx("isfalsy", sub: "full Python version", decl: true)
#snippet(```python
def is_truthy(x):
    return not is_falsy(x)

def is_falsy(x):
    return (is_boolean(x) and not x) or (is_number(x) and (x == 0 or x != x)) or (is_string(x) and x == "") or is_null(x) or is_undefined(x)
```)

The test #py("x != x") is not a typo;
the only Python value for which
#py("x != x") yields true is the value
#py("NaN") ("Not a Number"),
#idx("NaN, not a typo")
which is considered to be a falsy number (also not a typo), along with 0\.
The numerical value
#py("NaN") is the result of certain
arithmetic border cases such as
#py("0 / 0").

The terms "truthy" and "falsy" were coined
by
#idx("good parts of Python")
#idx("Python", sub: "good parts")
#idx("Crockford, Douglas")
Douglas Crockford, one of whose books
(Crockford 2008) inspired this JavaScript adaptation.]<foot:truthy>

#idx("istruthy", decl: true)#idx("isfalsy", decl: true)
#snippet(```python
def is_truthy(x):
    return x if is_boolean(x) else error("boolean expected, received", x)
def is_falsy(x):
    return not is_truthy(x)
```)

#subheading([Representing functions])

To handle primitives, we assume that we have available the following
#idx("metacircular evaluator for Python", sub: "representation of functions")
functions:

- #py("apply_primitive_function(")#meta("fun")#py(",") #meta("args")#py(")") #idx("applyprimitivefunction") applies the given primitive function to the argument values in the list #meta("args") and returns the result of the application.
- #py("is_primitive_function(")#meta("fun")#py(")") #idx("isprimitivefunction") tests whether #meta("fun") is a primitive function.

These mechanisms for handling primitives are further described in
section @sec:running-eval.

Compound
functions
are constructed from parameters,
function
bodies, and environments using the constructor
#py("make_function"):
#idx("makefunction", decl: true)#idx("iscompoundfunction", decl: true)#idx("functionparameters", decl: true)#idx("functionbody", decl: true)#idx("functionenvironment", decl: true)
#snippet(```python
def make_function(parameters, body, env):
    return llist("compound_function", parameters, body, env)
def is_compound_function(f):
    return is_tagged_list(f, "compound_function")
def function_parameters(f):
    return llist_ref(f, 1)

def function_body(f):
    return llist_ref(f, 2)

def function_environment(f):
    return llist_ref(f, 3)
```)

#subheading([Representing return values])

We saw in section @sec:core-of-evaluator that the
evaluation of a sequence terminates when a return statement
is encountered, and that the evaluation of a function application needs
to return the value #py("None") if
the evaluation of the function body does not encounter a
return statement. In order to recognize that a value resulted from a
#idx("return value", sub: "representation in metacircular evaluator")
return statement, we introduce #emph[return values] as evaluator data
structures.
#idx("makereturnvalue", decl: true)#idx("isreturnvalue", decl: true)#idx("returnvaluecontent", decl: true)
#snippet(```python
def make_return_value(content):
    return llist("return_value", content)
def is_return_value(value):
    return is_tagged_list(value, "return_value")
def return_value_content(value):
    return head(tail(value))
```)

#subheading([Operations on Environments])

#anchor(<sec:operations-on-environments>)

The evaluator needs operations for
#idx("metacircular evaluator for Python", sub: "environment operations")
#idx("symbol(s)", sub: "in environment operations")
manipulating environments. As explained
in section @sec:environment-model, an environment is a
sequence of frames, where each frame is a table of bindings that associate
symbols
with their corresponding values. We use the following operations for
manipulating environments:

- #py("lookup_symbol_value(")#meta("symbol")#py(",") #meta("env")#py(")") #idx("lookupsymbolvalue") returns the value that is bound to #meta("symbol") in the environment #meta("env"), or signals an error if #meta("symbol") is unbound.
- #py("extend_environment(")#meta("symbols")#py(",") #meta("values")#py(",") #meta("base-env")#py(")") #idx("extendenvironment") returns a new environment, consisting of a new frame in which the symbols in the list #meta("symbols") are bound to the corresponding elements in the list #meta("values"), where the enclosing environment is the environment #meta("base-env").
- #py("assign_symbol_value(")#meta("symbol")#py(",") #meta("value")#py(",") #meta("env")#py(")") #idx("assignsymbolvalue") finds the innermost frame of #meta("env") in which #meta("symbol") is bound, and changes that frame so that #meta("symbol") is now bound to #meta("value"), or signals an error if #meta("symbol") is unbound.

To implement these operations we
#idx("metacircular evaluator for Python", sub: "representation of environments")
represent an environment as a list of
frames. The enclosing environment of an environment is the
#py("tail")
of the list. The empty environment is simply the empty list.
#idx("enclosingenvironment", decl: true)#idx("firstframe", decl: true)#idx("theemptyenvironment", decl: true)
#snippet(```python
def enclosing_environment(env):
    return tail(env)

def first_frame(env):
    return head(env)

the_empty_environment = None
```)

Each frame of an environment is represented as a pair of lists: a list
of the
names
bound in that frame and a list of the associated
values.#footnote[Frames are not really a data
abstraction:
The function #py("assign_symbol_value") below uses #py("set_head") to directly modify the values in a frame.
The purpose of the frame
functions
is to make the environment-manipulation
functions
easy to read.]

#idx("makeframe", decl: true)#idx("framesymbols", decl: true)#idx("framevalues", decl: true)
#snippet(```python
def make_frame(symbols, values):
    return pair(symbols, values)

def frame_symbols(frame):
    return head(frame)

def frame_values(frame):
    return tail(frame)
```)

To extend an environment by a new frame that associates
symbols
with values, we make a frame consisting of the list of
symbols
and the list of values, and we adjoin this to the environment. We signal
an error if the number of
symbols
does not match the number of values.
#idx("extendenvironment", decl: true)
#snippet(```python
def extend_environment(symbols, vals, base_env):
    return pair(make_frame(symbols, vals), base_env) if length(symbols) == length(vals) else error("too many arguments supplied" if length(symbols) < length(vals) else "too few arguments supplied", pair(symbols, vals))
```)

This is used by #py("apply") in section @sec:core-of-evaluator to bind the parameters of a function to its arguments.

To look up a
symbol
in an environment, we scan the list of
symbols
in the first frame. If we find the desired
symbol,
we return the corresponding element in the list of values. If we do not
find the
symbol
in the current frame, we search the enclosing environment, and so on.
If we reach the empty environment, we signal an
#py("\"unbound name\"")
error.
#idx("lookupsymbolvalue", decl: true)
#snippet(```python
def lookup_symbol_value(symbol, env):
    def env_loop(env):
        def scan(symbols, vals):
            return env_loop(enclosing_environment(env)) if is_none(symbols) else head(vals) if symbol == head(symbols) else scan(tail(symbols), tail(vals))
        if env == the_empty_environment:
            error("unbound name", symbol)
        else:
            frame = first_frame(env)
            return scan(frame_symbols(frame), frame_values(frame))
    return env_loop(env)
```)

To assign
a new value to a symbol in a specified environment, we scan
for the symbol, just as in
#py("lookup_symbol_value"),
and change the corresponding value when we find it.

#idx("assignsymbolvalue", decl: true)
#snippet(```python
def assign_symbol_value(symbol, val, env):
    def env_loop(env):
        def scan(symbols, vals):
            return env_loop(enclosing_environment(env)) if is_none(symbols) else set_head(vals, val) if symbol == head(symbols) else scan(tail(symbols), tail(vals))
        if env == the_empty_environment:
            error("unbound name -- assignment", symbol)
        else:
            frame = first_frame(env)
            return scan(frame_symbols(frame), frame_values(frame))
    return env_loop(env)
```)

The method described here is only one of many plausible ways to represent
environments. Since we used
#idx("metacircular evaluator for Python", sub: "data abstraction in")
data abstraction to isolate the rest of the
evaluator from the detailed choice of representation, we could change the
environment representation if we wanted to. (See
exercise @ex:alternate-frame-representation.) In a
production-quality
Python
system, the speed of the evaluator's environment
operations—especially that of
symbol
lookup—has a major
impact on the performance of the system. The representation described here,
although conceptually simple, is not efficient and would not ordinarily be
used in a production system.#footnote[The drawback of this representation (as
well as the variant in
exercise @ex:alternate-frame-representation) is that the
evaluator may have to search through many frames in order to find the binding
for a given variable.
(Such an approach is referred to as
#idx("deep binding")
#idx("binding", sub: "deep")
#emph[deep binding].) One way to avoid
this inefficiency is to make use of a strategy called
#emph[lexical addressing], which will be discussed in
section @sec:lexical-addressing.]

#idx("metacircular evaluator for Python", sub: "representation of environments")

#exercise(label-name: <ex:alternate-frame-representation>, [
Instead of representing a frame as a pair of lists, we can represent a frame
as a list of bindings, where each binding is a symbol-value pair. Rewrite the
environment operations to use this alternative representation.
])

#exercise(label-name: <ex:4_10>, [
The
functions
#py("lookup_symbol_value") and #py("assign_symbol_value")
can be expressed in terms of
a more abstract function
for traversing the environment structure.
Define an abstraction that captures the common pattern and redefine the two functions in terms of this abstraction.
])

#exercise(label-name: <ex:mutable>, [
Our language distinguishes constants from variables by using
different keywords—#py("const")
and #py("let")—and prevents
assignment to constants. However, our interpreter
does not make use of this distinction; the function
#py("assign_symbol_value") will happily
assign a new value to a given symbol, regardless whether it is declared
as a constant or a variable.
#idx("constant (in Python)", sub: "detecting assignment to")
Correct this flaw by calling the function
#py("error") whenever an attempt is
made to use a constant on the left-hand side of an assignment.
You may proceed as follows:

- Introduce predicates #py("is_constant_declaration") and #py("is_variable_declaration") that allow you to distinguish the two kinds. As shown in section @sec:representing-expressions, #py("parse") distinguishes them by using the tags #py("\"constant_declaration\"") and #py("\"variable_declaration\"").
- Change #py("scan_out_declarations") and (if necessary) #py("extend_environment") such that constants are distinguishable from variables in the frames in which they are bound.
- Change #py("assign_symbol_value") such that it checks whether the given symbol has been declared as a variable or as a constant, and in the latter case signals an error that assignment operations are not allowed on constants.
- Change #py("eval_declaration") such that when it encounters a constant declaration, it calls a new function, #py("assign_constant_value"), which does not perform the check that you introduced in #py("assign_symbol_value").
- If necessary, change #py("apply") to ensure that assignment to function parameters remains possible.
])

#exercise(label-name: <ex:access_unassigned>, [
+ Python's specification requires an implementation to signal a runtime error upon an attempt to access the value of a name before its declaration is evaluated (see the end of section @sec:env-internal-def). To achieve this behavior in the evaluator, #idx("lookupsymbolvalue", sub: "for scanned-out declarations") change #py("lookup_symbol_value") to signal an error if the value it finds is #py("\"*unassigned*\"").
+ Similarly, we must not assign a new value to a variable if we have not evaluated its #py("let") declaration yet. Change the evaluation of assignment such that assignment to a variable declared with #py("let") signals an error in this case.
])

#exercise(label-name: <ex:var_js>, [
Prior to ECMAScript 2015's strict mode that we are using in this book,
Python variables
worked quite differently from Scheme variables, which would have made
this adaptation to Python considerably less compelling.

+ Before ECMAScript 2015, the only way to declare a local variable in Python was using the keyword #py("var") instead of the keyword #py("let"). The scope of variables declared with #py("var") is the entire body of the immediately surrounding function definition or lambda expression, rather than just the immediately enclosing block. Modify #py("scan_out_declarations") and #py("eval_block") such that names declared with #py("const") and #py("let") follow the scoping rules of #py("var").
+ When not in strict mode, Python permits undeclared names to appear to the left of the #py("=") in assignments. Such an assignment adds the new binding to the global environment. Modify the function #py("assign_symbol_value") to make assignment behave this way. The strict mode, which forbids such assignments, was introduced in Python in order to make programs more secure. What security issue is addressed by preventing assignment from adding bindings to the global environment?
])
