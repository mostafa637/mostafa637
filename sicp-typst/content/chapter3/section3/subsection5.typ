// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Propagation of Constraints], label-name: <sec:constraints>)

#idx("propagation of constraints")
#idx("constraint(s)", sub: "propagation of")

Computer programs are traditionally organized as
one-directional computations, which perform operations on prespecified
arguments to produce desired outputs. On the other hand, we often
model systems in terms of relations among quantities. For example, a
mathematical model of a mechanical structure might include the
information that the deflection $d$ of a metal
rod is related to the force $F$ on the rod, the
length $L$ of the rod, the cross-sectional
area $A$, and the elastic modulus
$E$ via the equation

$ mat(delim: #none, d A E, =, F L) $

Such an equation is not one-directional. Given any four of the
quantities, we can use it to compute the fifth. Yet translating the
equation into a traditional computer language would force us to choose
one of the quantities to be computed in terms of the other four.
Thus, a
function
for computing the area $A$ could not be used to
compute the deflection $d$, even though the
computations of $A$ and
$d$ arise from the same
equation.#footnote[Constraint propagation first appeared in the incredibly
forward-looking
#idx("SKETCHPAD")
SKETCHPAD system of
#idx("Sutherland, Ivan")
Ivan Sutherland (1963). A beautiful constraint-propagation system based
on the
#idx("Smalltalk")
Smalltalk language was developed by
#idx("Borning, Alan")
Alan Borning (1977) at
#idx("Xerox Palo Alto Research Center")
Xerox Palo Alto Research Center. Sussman, Stallman, and Steele
applied constraint propagation to electrical circuit analysis
#idx("Sussman, Gerald Jay")
#idx("Stallman, Richard M.")
(Sussman and Stallman 1975;
#idx("Steele, Guy Lewis Jr.")
Sussman and Steele 1980).
#idx("TK!Solver")
TK!Solver
#idx("Konopasek, Milos")
#idx("Jayaraman, Sundaresan")
(Konopasek and Jayaraman 1984)
is an extensive modeling environment based on constraints.]

In this section, we sketch the design of a language that enables us to work
in terms of
#idx("relations, computing in terms of")
relations themselves. The primitive elements of the language
are
#idx("primitive constraints")
#idx("constraint(s)", sub: "primitive")
#emph[primitive constraints], which state that certain relations hold
between quantities. For example,
#py("adder(a, b, c)")
specifies that the quantities $a$,
$b$, and $c$ must be
related by the equation $a+b=c$,
#py("multiplier(x, y, z)")
expresses the constraint $x y = z$, and
#py("constant(3.14, x)")
says that the value of $x$ must be 3.14.

Our language provides a means of combining primitive constraints in order to
express more complex relations. We combine constraints by constructing
#idx("constraint network")
#emph[constraint networks], in which constraints are joined by
#idx("connector(s), in constraint system")
#emph[connectors]. A connector is an object that "holds" a
value that may participate in one or more constraints. For example, we know
that the relationship between Fahrenheit and Celsius temperatures is

$ mat(delim: #none, 9C, =, 5(F - 32)) $

Such a constraint can be thought of as a network consisting of primitive
adder, multiplier, and constant constraints
(figure @fig:constraint). In the figure, we see on the
left a multiplier box with three terminals, labeled
$m_(1)$, $m_(2)$, and
$p$. These connect the multiplier to the rest of
the network as follows:
The $m_(1)$ terminal is linked to a connector
$C$, which will hold the Celsius temperature.
The $m_(2)$ terminal is linked to a connector
$w$, which is also linked to a constant box that
holds 9. The $p$ terminal, which the multiplier
box constrains to be the product of $m_(1)$ and
$m_(2)$, is linked to the
$p$ terminal of another multiplier box, whose
$m_(2)$ is connected to a constant 5 and whose
$m_(1)$ is connected to one of the terms in a sum.

#sicp-figure(image("/images/img_original/ch3-Z-G-30.svg", width: 70%), caption: [The relation $9C = 5(F - 32)$ expressed as a constraint network.], label-name: <fig:constraint>)

Computation by such a network proceeds as follows: When a connector is
given a value (by the user or by a constraint box to which it is
linked), it awakens all of its associated constraints (except for the
constraint that just awakened it) to inform them that it has a value.
Each awakened constraint box then polls its connectors to see if there
is enough information to determine a value for a connector. If so,
the box sets that connector, which then awakens all of its associated
constraints, and so on. For instance, in conversion between
Celsius and Fahrenheit, $w$,
$x$, and $y$ are
immediately set by the constant boxes to \$9\$, \$5\$, and \$32\$, respectively. The
connectors awaken the multipliers and the adder, which determine that there
is not enough information to proceed. If the user (or some other part of
the network) sets $C$ to a value (say 25), the
leftmost multiplier will be awakened, and it will set
$u$ to $25 dot.op 9=225$.
Then $u$ awakens the second multiplier, which sets
$v$ to \$45\$, and $v$
awakens the adder, which sets $F$ to \$77\$.

#subheading([Using the constraint system])

To use the constraint system to carry out the temperature computation
outlined above, we first call the constructor
#py("make_connector")
to create two connectors,
#py("C") and #py("F"),
and then link them in an appropriate network:

#snippet(```python
C = make_connector()
F = make_connector()
print(celsius_fahrenheit_converter(C, F))
```)

#output(```python
C = make_connector()
F = make_connector()
print(celsius_fahrenheit_converter(C, F))
```)

The
function
that creates the network is defined as follows:
#idx("celsiusfahrenheitconverter", decl: true)
#snippet(```python
def celsius_fahrenheit_converter(c, f):
    u = make_connector()
    v = make_connector()
    w = make_connector()
    x = make_connector()
    y = make_connector()
    multiplier(c, w, u)
    multiplier(v, x, u)
    adder(v, y, f)
    constant(9, w)
    constant(5, x)
    constant(32, y)
    return "ok"
```)

This
function
creates the internal connectors #py("u"),
#py("v"), #py("w"),
#py("x"), and #py("y"), and
links them as shown in figure @fig:constraint using the
primitive constraint constructors #py("adder"),
#py("multiplier"), and
#py("constant"). Just as with the digital-circuit
simulator of section @sec:circuit-simulator, expressing
these combinations of primitive elements in terms of
functions
automatically provides our language with a means of abstraction for compound
objects.

To watch the network in action, we can place probes on the connectors
#py("C") and #py("F"), using a
#py("probe")
function
similar to the one we used to monitor wires in
section @sec:circuit-simulator. Placing a probe on a
connector will cause a message to be printed whenever the connector is
given a value:

#snippet(```python
probe("Celsius temp", C)
probe("Fahrenheit temp", F)
```)

Next we set the value of #py("C") to 25. (The third
argument to
#py("set_value")
tells #py("C") that this directive comes from the
#py("user").)

#snippet(```python
print(set_value(C, 25, "user"))
```)

#output(```python
print(set_value(C, 25, "user"))
```)

The probe on #py("C") awakens and reports the value.
#py("C") also
propagates its value through the network as described above. This
sets #py("F") to 77, which is reported by the probe
on #py("F").

Now we can try to set #py("F") to a new value, say 212:

#snippet(```python
set_value(F, 212, "user")
```)

#output(```python
set_value(F, 212, "user")
```)

The connector complains that it has sensed a contradiction: Its value
is 77, and someone is trying to set it to 212. If we really want to
reuse the network with new values, we can tell
#py("C") to forget its old value:

#snippet(```python
print(forget_value(C, "user"))
```)

#output(```python
print(forget_value(C, "user"))
```)

#py("C") finds that the
#py("\"user\""),
who set its value originally, is now retracting that value, so
#py("C") agrees to lose its value, as shown by the
probe, and informs the rest of the network of this fact. This information
eventually propagates to #py("F"), which now finds
that it has no reason for continuing to believe that its own
value is 77\. Thus, #py("F") also
gives up its value, as shown by the probe.

Now that #py("F") has no value, we are free to set it
to 212:

#snippet(```python
print(set_value(F, 212, "user"))
```)

#output(```python
print(set_value(F, 212, "user"))
```)

This new value, when propagated through the network, forces
#py("C") to have a value of 100, and this is
registered by the probe on #py("C"). Notice that the
very same network is being used to compute #py("C")
given #py("F") and to compute
#py("F") given #py("C").
This nondirectionality of computation is the distinguishing feature of
constraint-based systems.

#subheading([Implementing the constraint system])

The constraint system is implemented via procedural objects with local
state, in a manner very similar to the digital-circuit simulator of
section @sec:circuit-simulator. Although the primitive
objects of the constraint system are somewhat more complex, the overall
system is simpler, since there is no concern about agendas and logic delays.

The basic
#idx("connector(s), in constraint system", sub: "operations on")
operations on connectors are the following:

- #py("has_value(")#meta("connector")#py(")") #idx("hasvalue") \ tells whether the connector has a value.
- #py("get_value(")#meta("connector")#py(")") #idx("getvalue") \ returns the connector's current value.
- #py("set_value(")#meta("connector")#py(",")#meta("new-value")#py(",") #meta("informant")#py(")") #idx("setvalue") \ indicates that the informant is requesting the connector to set its value to the new value.
- #py("forget_value(")#meta("connector")#py(",") #meta("retractor")#py(")") #idx("forgetvalue") \ tells the connector that the retractor is requesting it to forget its value.
- #py("connect(")#meta("connector")#py(",") #meta("new-constraint")#py(")") #idx("connect") \ tells the connector to participate in the new constraint.

The connectors communicate with the constraints by means of the
functions
#py("inform_about_value"),
which tells the given constraint that the connector has a value, and
#py("inform_about_no_value"),
which tells the constraint that the connector has lost its value.

#py("Adder") constructs an adder constraint among
summand connectors #py("a1") and
#py("a2") and a #py("sum")
connector. An adder is implemented as a
function
with local state (the
function
#py("me") below):
#idx("adder (primitive constraint)", decl: true)
#snippet(```python
def adder(a1, a2, sum):
    def process_new_value():
        if has_value(a1) and has_value(a2):
            set_value(sum, get_value(a1) + get_value(a2), me)
        elif has_value(a1) and has_value(sum):
            set_value(a2, get_value(sum) - get_value(a1), me)
        elif has_value(a2) and has_value(sum):
            set_value(a1, get_value(sum) - get_value(a2), me)
        else:
            pass
    def process_forget_value():
        forget_value(sum, me)
        forget_value(a1, me)
        forget_value(a2, me)
        process_new_value()
    def me(request):
        if request == "I have a value.":
            process_new_value()
        elif request == "I lost my value.":
            process_forget_value()
        else:
            error("unknown request -- adder", request)
    connect(a1, me)
    connect(a2, me)
    connect(sum, me)
    return me
```)

The function #py("adder")
connects the new adder to the designated
connectors and returns it as its value. The
function
#py("me"), which represents the adder, acts as a
dispatch to the local
functions.
The following
"syntax interfaces" (see
footnote @foot:object-syntax in
section @sec:circuit-simulator) are used in conjunction
with the dispatch:
#idx("informaboutvalue", decl: true)#idx("informaboutnovalue", decl: true)
#snippet(```python
def inform_about_value(constraint):
    return constraint("I have a value.")

def inform_about_no_value(constraint):
    return constraint("I lost my value.")
```)

The adder's local
function
#py("process_new_value")
is called when the adder is informed that one of its connectors has a value.
The adder first checks to see if both #py("a1") and
#py("a2") have values. If so, it tells
#py("sum") to set its value to the sum of the two
addends. The #py("informant") argument to
#py("set_value")
is #py("me"), which is the adder object itself. If
#py("a1") and #py("a2") do not
both have values, then the adder checks to see if perhaps
#py("a1") and #py("sum") have
values. If so, it sets #py("a2") to the difference of
these two. Finally, if #py("a2") and
#py("sum") have values, this gives the adder enough
information to set #py("a1"). If the adder is told
that one of its connectors has lost a value, it requests that all of its
connectors now lose their values. (Only those values that were set by
this adder are actually lost.) Then it runs
#py("process_new_value").
The reason for this last step is that one or more connectors may still
have a value (that is, a connector may have had a value that was not
originally set by the adder), and these values may need to be
propagated back through the adder.

A multiplier is very similar to an adder. It will set its
#py("product") to 0 if either of the factors is 0,
even if the other factor is not known.
#idx("multiplier", sub: "primitive constraint", decl: true)
#snippet(```python
def multiplier(m1, m2, product):
    def process_new_value():
        if ((has_value(m1) and get_value(m1) == 0)
                or (has_value(m2) and get_value(m2) == 0)):
            set_value(product, 0, me)
        elif has_value(m1) and has_value(m2):
            set_value(product, get_value(m1) * get_value(m2), me)
        elif has_value(product) and has_value(m1):
            set_value(m2, get_value(product) / get_value(m1), me)
        elif has_value(product) and has_value(m2):
            set_value(m1, get_value(product) / get_value(m2), me)
        else:
            pass
    def process_forget_value():
        forget_value(product, me)
        forget_value(m1, me)
        forget_value(m2, me)
        process_new_value()
    def me(request):
        if request == "I have a value.":
            process_new_value()
        elif request == "I lost my value.":
            process_forget_value()
        else:
            error("unknown request -- multiplier", request)
    connect(m1, me)
    connect(m2, me)
    connect(product, me)
    return me
```)

A #py("constant") constructor simply sets the value of
the designated connector. Any
#py("\"I have a value.\"")
or
#py("\"I lost my value.\"")
message sent to the constant box will produce an error.
#idx("constant (primitive constraint)", decl: true)
#snippet(```python
def constant(value, connector):
    def me(request):
        error("unknown request -- constant", request)
    connect(connector, me)
    set_value(connector, value, me)
    return me
```)

Finally, a probe prints a message about the setting or unsetting of
the designated connector:

#idx("probe", sub: "in constraint system", decl: true)
#snippet(```python
def probe(name, connector):
    def print_probe(value):
        display("Probe: " + name + " = " + str(value))
    def process_new_value():
        print_probe(get_value(connector))
    def process_forget_value():
        print_probe("?")
    def me(request):
        return (process_new_value()
                if request == "I have a value."
                else process_forget_value()
                if request == "I lost my value."
                else error("unknown request -- probe", request))
    connect(connector, me)
    return me
```)

#subheading([Representing connectors])

#idx("connector(s), in constraint system", sub: "representing")

A connector is represented as a procedural object with local state variables
#py("value"), the current value of the connector;
#py("informant"), the object that set the
connector's value; and #py("constraints"),
a list of the constraints in which the connector participates.
#idx("makeconnector", decl: true)
#snippet(```python
def make_connector():
    value = False
    informant = False
    constraints = None
    def set_my_value(newval, setter):
        nonlocal value, informant
        if not has_value(me):
            value = newval
            informant = setter
            return for_each_except(setter,
                                   inform_about_value,
                                   constraints)
        elif value != newval:
            error("contradiction", llist(value, newval))
        else:
            return "ignored"
    def forget_my_value(retractor):
        nonlocal informant
        if retractor is informant:
            informant = False
            return for_each_except(retractor,
                                   inform_about_no_value,
                                   constraints)
        else:
            return "ignored"
    def connect(new_constraint):
        nonlocal constraints
        if is_none(member(new_constraint, constraints)):
            constraints = pair(new_constraint, constraints)
        else:
            pass
        if has_value(me):
            inform_about_value(new_constraint)
        else:
            pass
        return "done"
    def me(request):
        if request == "has_value":
            return informant is not False
        elif request == "value":
            return value
        elif request == "set_value":
            return set_my_value
        elif request == "forget":
            return forget_my_value
        elif request == "connect":
            return connect
        else:
            error("unknown operation -- connector", request)
    return me
```)

The connector's local
function
#py("set_my_value")
is called when there is a request to set the connector's value. If
the connector does not currently have a value, it will set its value and
remember as #py("informant") the constraint that
requested the value to be set.#footnote[The
#py("setter") might not be a constraint. In our
temperature example, we used
#py("\"user\"")
as the
#py("setter").] Then the connector will
notify all of its participating constraints except the constraint that
requested the value to be set. This is accomplished using the following
iterator, which applies a designated
function
to all items in a list except a given one:
#idx("foreachexcept", decl: true)
#snippet(```python
def for_each_except(exception, fun, list):
    def loop(items):
        if is_none(items):
            return "done"
        elif head(items) is exception:
            return loop(tail(items))
        else:
            fun(head(items))
            return loop(tail(items))
    return loop(list)
```)

If a connector is asked to forget its value, it runs
#py("forget_my_value"), a local function that
first checks to make sure that the request is coming from the same
object that set the value originally. If so, the connector informs its
associated constraints about the loss of the value.

The local
function
#py("connect") adds the designated new constraint
to the list of constraints if it is not already in that
list.
Then, if the connector has a value, it informs the new constraint of this
fact.

The connector's
function
#py("me") serves as a dispatch to the other internal
functions
and also represents the connector as an object. The following
functions
provide a syntax interface for the dispatch:
#idx("hasvalue", decl: true)#idx("getvalue", decl: true)#idx("setvalue", decl: true)#idx("forgetvalue", decl: true)#idx("connect", decl: true)
#snippet(```python
def has_value(connector):
    return connector("has_value")

def get_value(connector):
    return connector("value")

def set_value(connector, new_value, informant):
    return connector("set_value")(new_value, informant)

def forget_value(connector, retractor):
    return connector("forget")(retractor)

def connect(connector, new_constraint):
    return connector("connect")(new_constraint)
```)

#exercise(label-name: <ex:3_33>, [
Using primitive multiplier, adder, and constant constraints, define a
function
#idx("averager (constraint)")
#py("averager") that takes three connectors
#py("a"), #py("b"),
and #py("c") as inputs and establishes the
constraint that the value of
#py("c") is the average of the values of
#py("a") and #py("b").
])

#exercise(label-name: <ex:squarer-constraint>, [
Louis Reasoner wants to build a
#idx("squarer (constraint)")
squarer, a constraint device with two
terminals such that the value of connector
#py("b") on the second
terminal will always be the square of the value
#py("a") on the first
terminal. He proposes the following simple device made from a
multiplier:

#snippet(```python
def squarer(a, b):
    return multiplier(a, a, b)
```)

There is a serious flaw in this idea. Explain.
])

#exercise(label-name: <ex:3_35>, [
Ben Bitdiddle tells Louis that one way to avoid the trouble in
exercise @ex:squarer-constraint is to define a
#idx("squarer (constraint)")
squarer as a new primitive constraint. Fill in the missing
portions in Ben's outline for a
function
to implement such a constraint:

#syntax("
def squarer(a, b):
    def process_new_value():
        if has_value(b):
            if get_value(b) < 0:
                error(\"square less than 0 -- squarer\", get_value(b))
            else:
                ", meta("alternative_1"), "
        else:
            ", meta("alternative_2"), "
    def process_forget_value():
        ", meta("body_1"), "
    def me(request):
        ", meta("body_2"), "
    ", meta("statements"), "
    return me
      ")
])

#exercise(label-name: <ex:3_36>, [
Suppose we evaluate the following sequence of
statements
in the
program
environment:

#snippet(```python
a = make_connector()
b = make_connector()
set_value(a, 10, "user")
```)

At some time during evaluation of the
#py("set_value"),
the following expression from the connector's local
function
is evaluated:

#snippet(```python
for_each_except(setter, inform_about_value, constraints)
```)

Draw an environment diagram showing the environment in which the above
expression is evaluated.
])

#exercise(label-name: <ex:3_37>, [
The
#py("celsius_fahrenheit_converter")
function
is cumbersome when
compared with a more expression-oriented style of definition, such as
#idx("celsiusfahrenheitconverter", sub: "expression-oriented", decl: true)
#snippet(```python
def celsius_fahrenheit_converter(x):
    return cplus(cmul(cdiv(cv(9), cv(5)), x), cv(32))

C = make_connector()
F = celsius_fahrenheit_converter(C)
```)

Here
#py("cplus"),
#py("cmul"),
etc. are the "constraint"
versions of the arithmetic operations. For example,
#py("cplus")
takes two connectors as arguments and returns a connector that is
related to these by an adder constraint:

#snippet(```python
def cplus(x, y):
    z = make_connector()
    adder(x, y, z)
    return z
```)

Define analogous
functions
#py("cminus"),
#py("cmul"),
#py("cdiv"),
and
#py("cv")
(constant value) that enable us to define compound constraints as in
the converter example above.#footnote[The
#idx("expression-oriented vs. imperative programming style")
#idx("imperative vs. expression-oriented programming style")
expression-oriented format
is convenient because it avoids the need to name the intermediate
expressions in a computation. Our original formulation of the
constraint language is cumbersome in the same way that many languages
are cumbersome when dealing with operations on compound data. For
example, if we wanted to compute the product
$((a+b)) dot.op ((c+d))$, where the
variables represent vectors, we could work in
"imperative style,"
using
functions
that set the values of designated vector arguments
but do not themselves return vectors as values:

#snippet(```python
v_sum("a", "b", temp1)
v_sum("c", "d", temp2)
v_prod(temp1, temp2, answer)
```)

Alternatively, we could deal with expressions, using
functions
that return vectors as values, and thus avoid
explicitly mentioning #py("temp1") and
#py("temp2"):

#snippet(```python
answer = v_prod(v_sum("a", "b"), v_sum("c", "d"))
```)

Since
Python
allows us to return compound objects as values of
functions,
we can transform our imperative-style constraint language
into an expression-oriented style as shown in this exercise.

Given the advantage of the
expression-oriented format, one might ask if there is any reason to
have implemented the system in imperative style, as we did in this
section. One reason is that the non-expression-oriented constraint
language provides a handle on constraint objects (e.g., the value of
the #py("adder")
function)
as well as on connector objects. This is
useful if we wish to extend the system with new operations that
communicate with constraints directly rather than only indirectly via
operations on connectors. Although it is easy to implement the
expression-oriented style in terms of the imperative implementation,
it is very difficult to do the converse.]
])

#idx("propagation of constraints")
#idx("constraint(s)", sub: "propagation of")
