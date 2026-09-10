// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#section([Lazy Evaluation], label-name: <sec:lazy-evaluation>)

#idx("lazy evaluator")
#idx("delayed evaluation", sub: "in lazy evaluator")

Now that we have an evaluator expressed as a
Python
program, we can experiment with alternative choices in
#idx("programming language", sub: "design of")
#idx("embedded language, language design using")
language design
simply by modifying the evaluator. Indeed, new languages are often
invented by first writing an evaluator that embeds the new language
within an existing high-level language. For example, if we wish to
discuss some aspect of a proposed modification to
Python
with another member of the
Python
community, we can supply an evaluator that embodies
the change. The recipient can then experiment with the new
evaluator and send back comments as further modifications. Not only
does the high-level implementation base make it easier to test and
debug the evaluator; in addition, the embedding enables the designer
to snarf#footnote[Snarf: "To grab, especially a large document or file for the purpose of using it either with or without the owner's permission." Snarf down: "To snarf, sometimes with the connotation of absorbing, processing, or understanding."
(These definitions were
#idx("snarf")
snarfed from
#idx("Steele, Guy Lewis Jr.")
Steele et al. 1983.
See also
#idx("Raymond, Eric")
Raymond 1996.)] features
from the underlying language, just as our embedded
Python
evaluator uses primitives and control structure from the underlying
Python.
Only later (if ever) need the designer go to the trouble of building a
complete implementation in a low-level language or in hardware. In
this section and the next we explore some variations on
Python
that provide significant additional expressive power.

#include "../../chapter4/section2/subsection1.typ"

#include "../../chapter4/section2/subsection2.typ"

#include "../../chapter4/section2/subsection3.typ"
