// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Modeling with Mutable Data], label-name: <sec:mutable-data>)

#idx("mutable data objects")

Chapter @chap:data dealt with compound data as a means for constructing
computational objects that have several parts, in order to model
real-world objects that have several aspects. In that chapter we
introduced the discipline of data abstraction, according to which data
structures are specified in terms of constructors, which create data
objects, and selectors, which access the parts of compound data
objects. But we now know that there is another aspect of data that
chapter @chap:data did not address. The desire to model systems composed of
objects that have changing state leads us to the need to modify
compound data objects, as well as to construct and select from them.
In order to model compound objects with changing state, we will design
data abstractions to include, in addition to selectors and
constructors, operations called
#idx("mutator")
#emph[mutators], which modify data
objects. For instance, modeling a banking system requires us to
change account balances. Thus, a data structure for representing bank
accounts might admit an operation

#syntax("
set_balance(", meta("account"), ", ", meta("new-value"), ")
      ")

that changes the balance of the designated account to the designated
new value. Data objects for which mutators are defined are known as
#emph[mutable data objects].

Chapter @chap:data introduced pairs as a general-purpose "glue"
for synthesizing compound data. We begin this section by defining basic
mutators for pairs, so that pairs can serve as building blocks for
constructing mutable data objects. These mutators greatly enhance the
representational power of pairs, enabling us to build data structures
other than the sequences and trees that we worked with in
section @sec:hierarchical-data. We also present some
examples of simulations in which complex systems are modeled as collections
of objects with local state.

#include "../../chapter3/section3/subsection1.typ"

#include "../../chapter3/section3/subsection2.typ"

#include "../../chapter3/section3/subsection3.typ"

#include "../../chapter3/section3/subsection4.typ"

#include "../../chapter3/section3/subsection5.typ"
