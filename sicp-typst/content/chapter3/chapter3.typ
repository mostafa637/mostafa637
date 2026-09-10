// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../lib/sicp.typ": *

#chapter([Modularity, Objects, and State], label-name: <chap:state>)

#epigraph([$M epsilon tau alpha beta acute( alpha ) lambda lambda o nu ἀ nu alpha pi alpha acute( upsilon ) epsilon tau alpha iota$
\

(Even while it changes, it stands still.)
#idx("Heraclitus")], author: [Heraclitus])

#epigraph([Plus ça change, plus c'est la même chose.

#idx("Karr, Alphonse")], author: [Alphonse Karr])

The preceding chapters introduced the basic elements from which
programs are made. We saw how primitive
functions
and primitive data are combined to construct compound entities, and we
learned that abstraction is vital in helping us to cope with the complexity
of large systems. But these tools are not sufficient for designing
programs. Effective program synthesis also requires organizational
principles that can guide us in formulating the overall design of a
program. In particular, we need strategies to help us structure large
systems so that they will be
#idx("modularity")
#emph[modular], that is, so that they can
be divided "naturally" into coherent parts that can be
separately developed and maintained.

One powerful design strategy, which is particularly appropriate to the
construction of programs for
#idx("modeling", sub: "as a design strategy")
modeling physical systems, is to base the
structure of our programs on the structure of the system being
modeled. For each object in the system, we construct a corresponding
computational object. For each system action, we define a symbolic
operation in our computational model. Our hope in using this strategy
is that extending the model to accommodate new objects or new actions
will require no strategic changes to the program, only the addition of
the new symbolic analogs of those objects or actions. If we have been
successful in our system organization, then to add a new feature or
debug an old one we will have to work on only a localized part of the
system.

To a large extent, then, the way we organize a large program is
dictated by our perception of the system to be modeled. In this
chapter we will investigate two prominent organizational strategies
arising from two rather different "world views" of the
structure of systems. The first organizational strategy concentrates on
#idx("object(s)")
#emph[objects], viewing a large system as a collection of distinct objects
whose behaviors may change over time. An alternative organizational
strategy concentrates on the
#idx("stream(s)")
#emph[streams] of information that flow in
the system, much as an electrical engineer views a signal-processing
system.

Both the object-based approach and the stream-processing approach
raise significant linguistic issues in programming.
With objects, we must be concerned with how a computational object can
change and yet maintain its identity. This will force us to abandon
our old substitution model of computation
(section @sec:substitution-model) in favor of a more
mechanistic but less theoretically tractable
#idx("environment model of evaluation")
#emph[environment model] of
computation. The difficulties of dealing with objects, change, and
identity are a fundamental consequence of the need to grapple with
time in our computational models. These difficulties become even
greater when we allow the possibility of concurrent execution of
programs. The stream approach can be most fully exploited when we
decouple simulated time in our model from the order of the events that
take place in the computer during evaluation. We will accomplish this
using a technique known as
#idx("delayed evaluation")
#emph[delayed evaluation].

#include "../chapter3/section1/section1.typ"

#include "../chapter3/section2/section2.typ"

#include "../chapter3/section3/section3.typ"

#include "../chapter3/section4/section4.typ"

#include "../chapter3/section5/section5.typ"
