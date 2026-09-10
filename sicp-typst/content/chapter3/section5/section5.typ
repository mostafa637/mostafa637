// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Streams], label-name: <sec:streams>)

#idx("stream(s)")

We've gained a good understanding of assignment as a tool in modeling,
as well as an appreciation of the complex problems that assignment
raises. It is time to ask whether we could have gone about things in a
different way, so as to avoid some of these problems. In this
section, we explore an alternative approach to modeling state, based
on data structures called #emph[streams]. As we shall see, streams can
mitigate some of the complexity of modeling state.

Let's step back and review where this complexity comes from. In an
attempt to model real-world phenomena, we made some apparently
reasonable decisions: We modeled real-world objects with local state
by computational objects with local variables. We identified time
variation in the real world with time variation in the computer. We
implemented the time variation of the states of the model objects in
the computer with assignments to the local variables of the model
objects.

Is there another approach? Can we avoid identifying time in the
computer with time in the modeled world? Must we make the model
change with time in order to model phenomena in a changing world?
Think about the issue in terms of mathematical functions. We can
describe the time-varying behavior of a quantity
$x$ as a function of time
$x(t)$.
If we concentrate on $x$ instant by instant,
we think of it as a changing quantity. Yet if we concentrate on the entire
time history of values, we do not emphasize change—the function
itself does not change.#footnote[Physicists sometimes adopt this view by
introducing the
#idx("world line of a particle")
"world lines" of particles as a device for reasoning about
motion. We've also already mentioned
(section @sec:sequences-conventional-interfaces) that
this is the natural way to think about signal-processing systems. We will
explore applications of streams to signal processing in
section @sec:exploiting-streams.]

If time is measured in discrete steps, then we can model a time function as
a (possibly infinite) sequence. In this section, we will see how to
model change in terms of sequences that represent the time histories
of the systems being modeled. To accomplish this, we introduce new
data structures called #emph[streams]. From an abstract point of view,
a stream is simply a sequence. However, we will find that the
straightforward implementation of streams as lists (as in
section @sec:sequences) doesn't fully reveal
the power of stream processing. As an alternative, we introduce the
technique of
#idx("delayed evaluation")
#emph[delayed evaluation], which enables us to represent
very large (even infinite) sequences as streams.

Stream processing lets us model systems that have state without ever
using assignment or mutable data. This has important implications,
both theoretical and practical, because we can build models that avoid
the drawbacks inherent in introducing assignment. On the other hand,
the stream framework raises difficulties of its own, and the question
of which modeling technique leads to more modular and more easily
maintained systems remains open.

#include "../../chapter3/section5/subsection1.typ"

#include "../../chapter3/section5/subsection2.typ"

#include "../../chapter3/section5/subsection3.typ"

#include "../../chapter3/section5/subsection4.typ"

#include "../../chapter3/section5/subsection5.typ"
