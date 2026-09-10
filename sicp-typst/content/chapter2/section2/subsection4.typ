// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Example: A Picture Language], label-name: <sec:graphics>)

#idx("picture language")

This section presents a simple language for drawing pictures that
illustrates the power of data abstraction and closure, and also exploits
higher-order
functions
in an essential way. The language is designed to make it easy to
experiment with patterns such as the ones in
figure @fig:sqlimit-designs, which are composed of
repeated elements that are shifted and scaled.#footnote[The picture
language is based on the language
#idx("Henderson, Peter")
Peter Henderson created to construct images like
#idx("Escher, Maurits Cornelis")
M.C. Escher's "Square Limit" woodcut (see
Henderson 1982). The woodcut incorporates a repeated
scaled pattern, similar to the arrangements drawn using the
#py("square_limit")
function
in this section.] In this language, the data objects being
combined are represented as
functions
rather than as linked-list structure. Just as
#py("pair"),
which satisfies the
#idx("closure", sub: "closure property of picture-language operations")
closure property, allowed us to easily build arbitrarily complicated
linked-list structure, the operations in this language, which also satisfy the closure
property, allow us to easily build arbitrarily complicated patterns.

#sicp-figure(image("/images/img_original/2.9.svg", width: 50%), caption: [Designs generated with the picture language.], label-name: <fig:sqlimit-designs>)

#subheading([The picture language])

When we began our study of programming in
section @sec:elements-of-programming, we emphasized the
importance of describing a language by focusing on the language's
primitives, its means of combination, and its means of abstraction.
We'll follow that framework here.

Part of the elegance of this picture language is that there is only one
kind of element, called a
#idx("painter(s)")
#emph[painter]. A painter draws an image that is shifted and scaled to
fit within a designated
#idx("frame (picture language)")
parallelogram-shaped frame. For example, there's a primitive painter
we'll call #py("wave")
that makes a crude line drawing,
as shown in figure @fig:wave.

#sicp-figure(image("/images/img_original/2.10.svg", width: 100%), caption: [Images produced by the #py("wave") painter, with respect to four different frames. The frames, shown with dashed lines, are not part of the images.], label-name: <fig:wave>)

The actual shape of the drawing depends on the frame—all four
images in figure @fig:wave are produced by the same
#py("wave") painter, but with respect to four
different frames. Painters can be more elaborate than this: The primitive
painter called #py("rogers") paints a picture of
MIT's founder, William Barton Rogers, as shown in
figure @fig:rogers.#footnote[#idx("MIT", sub: "early history of")
#idx("Rogers, William Barton")
William Barton Rogers (1804–1882) was the founder and first
president of MIT. A geologist and talented teacher, he taught at
William and Mary College and at the University of Virginia. In 1859
he moved to Boston, where he had more time for research, worked on a
plan for establishing a "polytechnic institute," and
served as Massachusetts's first State Inspector of Gas Meters.

When MIT was established in 1861, Rogers was elected its first
president. Rogers espoused an ideal of "useful learning"
that was different from the university education of the time, with its
overemphasis on the classics, which, as he wrote, "stand in the way of the broader, higher and more practical instruction and discipline of the natural and social sciences." This
education was likewise to be different from narrow trade-school
education. In Rogers's words:

#blockquote[The world-enforced distinction between the practical and the
scientific worker is utterly futile, and the whole experience of
modern times has demonstrated its utter worthlessness.]

Rogers served as president of MIT until 1870, when he resigned due to
ill health. In 1878 the second president of MIT,
#idx("Runkle, John Daniel")
John Runkle, resigned under the pressure of a financial crisis
brought on by the Panic of 1873 and strain of fighting off attempts
by Harvard to take over MIT. Rogers returned to hold the office of
president until 1881.

Rogers collapsed and died while addressing MIT's graduating
class at the commencement exercises of 1882. Runkle quoted
Rogers's last words in a memorial address delivered that same
year:

#blockquote["As I stand here today and see what the Institute is, … I call to mind the beginnings of science. I remember one hundred and fifty years ago Stephen Hales published a pamphlet on the subject of illuminating gas, in which he stated that his researches had demonstrated that 128 grains of bituminous coal—"

#idx("coal, bituminous")
"Bituminous coal," these were his last words on
earth. Here he bent forward, as if consulting some notes on the
table before him, then slowly regaining an erect position, threw
up his hands, and was translated from the scene of his earthly
labors and triumphs to "the tomorrow of death,"
where the mysteries of life are solved, and the disembodied
spirit finds unending satisfaction in contemplating the new and
still unfathomable mysteries of the infinite future.]

In the words of Francis A. Walker
#idx("Walker, Francis Amasa")
(MIT's third president):

#blockquote[All his life he had borne himself most faithfully and heroically,
and he died as so good a knight would surely have wished, in
harness, at his post, and in the very part and act of public duty.]]
The four images in figure @fig:rogers
are drawn with respect to the same four frames
as the #py("wave") images in
figure @fig:wave.

#sicp-figure(image("/images/img_original/2.11.svg", width: 100%), caption: [Images of William Barton Rogers, founder and first president of MIT, painted with respect to the same four frames as in figure @fig:wave (original image courtesy MIT Museum).], label-name: <fig:rogers>)

To combine images, we use various
#idx("painter(s)", sub: "operations")
operations that construct new painters
from given painters. For example, the
#idx("beside")
#py("beside") operation takes two painters and
produces a new, compound painter that draws the first painter's image
in the left half of the frame and the second painter's image in the
right half of the frame. Similarly,
#idx("below")
#py("below") takes two painters and produces a
compound painter that draws the first painter's image below the
second painter's image. Some operations transform a single painter
to produce a new painter. For example,
#idx("flipvert")
#py("flip_vert")
takes a painter and produces a painter that draws its image upside-down, and
#idx("fliphoriz")
#py("flip_horiz")
produces a painter that draws the original painter's image
left-to-right reversed.

Figure @fig:build-up-wave shows the drawing of a
painter called #py("wave4")
that is built up in two stages starting from
#py("wave"):

#snippet(```python
wave2 = beside(wave, flip_vert(wave))
wave4 = below(wave2, wave2)
```)

In building up a complex image in this manner we are exploiting the fact
that painters are
#idx("closure", sub: "closure property of picture-language operations")
closed under the language's means of combination.
The #py("beside") or
#py("below") of two painters is itself a painter;
therefore, we can use it as an element in making more complex painters.
As with building up linked-list structure using
#py("pair"),
the closure of our data under the means of combination is crucial to the
ability to create complex structures while using only a few operations.

#sicp-figure(stack(dir: ttb, spacing: 1em, image("/images/img_original/2.12.svg", width: 50%), [#syntax("
$\\ $
wave2 =                          wave4 =
 beside(wave, flip_vert(wave))    below(wave2, wave2)

      ")]), caption: [Creating a complex figure, starting from the #py("wave") painter of figure @fig:wave.], label-name: <fig:build-up-wave>)

Once we can combine painters, we would like to be able to abstract typical
patterns of combining painters. We will implement the painter operations as
Python functions.
This means that we don't need a special abstraction mechanism in the
picture language: Since the means of combination are ordinary
Python functions,
we automatically have the capability to do anything with painter operations
that we can do with
functions.
For example, we can abstract the pattern in
#py("wave4") as
#idx("flippedpairs", decl: true)
#snippet(```python
def flipped_pairs(painter):
    painter2 = beside(painter, flip_vert(painter))
    return below(painter2, painter2)
```)

and
declare
#py("wave4") as an instance of this
pattern:

#snippet(```python
wave4 = flipped_pairs(wave)
```)

#sicp-figure(image("/images/img_javascript/ch2-Z-G-37.svg", width: 59%), caption: [Recursive plans for #py("right_split") and #py("corner_split").], label-name: <fig:split-plans>)

We can also define recursive operations. Here's one that makes
painters split and branch towards the right as shown in
figures @fig:split-plans
and
@fig:split-plans-2:
#idx("rightsplit", decl: true)
#snippet(```python
def right_split(painter, n):
    if n == 0:
        return painter
    else:
        smaller = right_split(painter, n - 1)
        return beside(painter, below(smaller, smaller))
```)

We can produce balanced patterns by branching upwards as well as towards
the right (see exercise @ex:up-split and
figures @fig:split-plans
and @fig:split-plans-2):
#idx("cornersplit", decl: true)
#snippet(```python
def corner_split(painter, n):
    if n == 0:
        return painter
    else:
        up = up_split(painter, n - 1)
        right = right_split(painter, n - 1)
        top_left = beside(up, up)
        bottom_right = below(right, right)
        corner = corner_split(painter, n - 1)
        return beside(below(painter, top_left),
                      below(bottom_right, corner))
```)

#sicp-figure(image("/images/img_javascript/2.14.svg", width: 45%), caption: [The recursive operation #py("right_split") applied to the painters #py("wave") and #py("rogers"). Combining four #py("corner_split") figures produces symmetric #py("square_limit") as shown in figure @fig:sqlimit-designs.], label-name: <fig:split-plans-2>)

By placing four copies of a
#py("corner_split")
appropriately, we obtain a pattern called
#py("square_limit"),
whose application to #py("wave") and
#py("rogers") is shown in
figure @fig:sqlimit-designs:
#idx("squarelimit", decl: true)
#snippet(```python
def square_limit(painter, n):
    quarter = corner_split(painter, n)
    half = beside(flip_horiz(quarter), quarter)
    return below(flip_vert(half), half)
```)

#exercise(label-name: <ex:up-split>, [
Declare the function
#idx("upsplit")
#py("up_split")
used by
#py("corner_split").
It is similar to
#py("right_split"),
except that it switches the roles of #py("below")
and #py("beside").

#anchor(<ex:2_44>)
])

#subheading([Higher-order operations])

#idx("painter(s)", sub: "higher-order operations")

In addition to abstracting patterns of combining painters, we can work at a
higher level, abstracting patterns of combining painter operations. That
is, we can view the painter operations as elements to manipulate and can
write means of combination for these
elements—functions
that take painter operations as arguments and create new painter operations.

For example,
#py("flipped_pairs")
and
#py("square_limit")
each arrange four copies of a painter's image in a square pattern;
they differ only in how they orient the copies. One way to abstract this
pattern of painter combination is with the following
function,
which takes four one-argument painter operations and produces a painter
operation that transforms a given painter with those four operations and
arranges the results in a square.#footnote[The painter operation returned by #py("square_of_four") consists of several statements, so it cannot be written as a lambda expression, whose body in Python must be a single expression. Instead, we use a local function declaration with the name #py("combine"). #idx("lambda expression", sub: "restricted to a single expression in Python")]<foot:lambda_with_block>
The functions #py("tl"),
#py("tr"), #py("bl"), and
#py("br") are the transformations to apply to the
top left copy, the top right copy, the bottom left copy, and the bottom
right copy, respectively.
#idx("squareoffour", decl: true)
#snippet(```python
def square_of_four(tl, tr, bl, br):
    def combine(painter):
        top = beside(tl(painter), tr(painter))
        bottom = beside(bl(painter), br(painter))
        return below(bottom, top)
    return combine
```)

Then
#py("flipped_pairs") can be defined in terms of
#py("square_of_four") as follows:#footnote[Equivalently, we could
write
#idx("flippedpairs", decl: true)
#snippet(```python
flipped_pairs = square_of_four(identity, flip_vert,
                               identity, flip_vert)
```)]
#idx("flippedpairs", decl: true)
#snippet(```python
def flipped_pairs(painter):
    combine4 = square_of_four(identity, flip_vert,
                                    identity, flip_vert)
    return combine4(painter)
```)

and
#py("square_limit")
can be expressed as#footnote[The function #py("rotate180")
rotates a painter by 180 degrees. Instead of
#py("rotate180")
we could say
#py("compose(flip_vert, flip_horiz)"),
using the
#py("compose")
function
from exercise @ex:compose.]
#idx("squarelimit", decl: true)
#snippet(```python
def square_limit(painter, n):
    combine4 = square_of_four(flip_horiz, identity,
                                    rotate180, flip_vert)
    return combine4(corner_split(painter, n))
```)

#exercise(label-name: <ex:splitting>, [
The functions #py("right_split")
and
#py("up_split")
can be expressed as instances of a general splitting operation.
Declare a function
#idx("split")
#py("split") with the property that evaluating

#snippet(```python
right_split = split(beside, below)
up_split = split(below, beside)
```)

produces
functions
#py("right_split")
and
#py("up_split") with the same behaviors as the ones already declared.
])

#subheading([Frames])

#idx("frame (picture language)")

Before we can show how to implement painters and their means of
combination, we must first consider
#idx("vector (mathematical)", sub: "in picture-language frame")
frames. A frame can be described by three vectors—an origin vector
and two edge vectors. The origin vector specifies the offset of the
frame's origin from some absolute origin in the plane, and the edge
vectors specify the offsets of the frame's corners from its origin.
If the edges are perpendicular, the frame will be rectangular.
Otherwise the frame will be a more general parallelogram.

Figure @fig:frame shows a frame and its associated
vectors. In accordance with data abstraction, we need not be specific yet
about how frames are represented, other than to say that there is a
constructor
#idx("makeframe")
#py("make_frame"),
which takes three vectors and produces a frame, and three corresponding
selectors
#idx("originframe")
#py("origin_frame"),
#idx("edge1frame")
#py("edge1_frame"),
and
#idx("edge2frame")
#py("edge2_frame")
(see exercise @ex:implement-frames).

#sicp-figure(image("/images/img_original/ch2-Z-G-42.svg", width: 70%), caption: [A frame is described by three vectors—an origin and two edges.], label-name: <fig:frame>)

We will use coordinates in the
#idx("unit square")
unit square
($0 lt.eq x, y lt.eq 1$) to specify images. With
each frame, we associate a
#idx("frame (picture language)", sub: "coordinate map")
#emph[frame coordinate map], which will be used to shift and scale images
to fit the frame. The map transforms the unit square into the frame by
mapping the vector $bold("v")=(x, y)$ to the
vector sum

$ upright("Origin(Frame)") + x dot.op upright(" Edge")_(1)upright(" (Frame)") + y dot.op upright(" Edge")_(2)upright(" (Frame)") $

For example, $(0, 0)$ is mapped to the origin of
the frame, $(1, 1)$ to the vertex diagonally
opposite the origin, and $(0.5, 0.5)$ to the
center of the frame. We can create a frame's coordinate map with
the following
function:#footnote[The function #py("frame_coord_map")
uses the vector operations described in
exercise @ex:vectors below, which we assume have been
implemented using some representation for vectors. Because of data
abstraction, it doesn't matter what this vector representation is,
so long as the vector operations behave correctly.]
#idx("framecoordmap", decl: true)
#snippet(```python
def frame_coord_map(frame):
    return lambda v: add_vect(origin_frame(frame),
                         add_vect(scale_vect(xcor_vect(v),
                                             edge1_frame(frame)),
                                  scale_vect(ycor_vect(v),
                                             edge2_frame(frame))))
```)

Observe that applying
#py("frame_coord_map")
to a frame returns a
function
that, given a vector, returns a vector. If the argument vector is in the
unit square, the result vector will be in the frame. For example,

#snippet(```python
print(frame_coord_map(a_frame)(make_vect(0, 0)))
```)

returns the same vector as

#snippet(```python
print(origin_frame(a_frame))
```)

#exercise(label-name: <ex:vectors>, [
A two-dimensional
#idx("vector (mathematical)", sub: "represented as pair")
#idx("vector (mathematical)", sub: "operations on")
vector $v$ running from the
origin to a point can be represented as a pair consisting of an
$x$-coordinate and a
$y$-coordinate. Implement a data abstraction
for vectors by giving a constructor
#idx("makevect")
#py("make_vect")
and corresponding selectors
#idx("xcorvect")
#py("xcor_vect")
and
#idx("ycorvect")
#py("ycor_vect").
In terms of your selectors and constructor, implement
functions
#idx("addvect")
#py("add_vect"),
#idx("subvect")
#py("sub_vect"),
and
#idx("scalevect")
#py("scale_vect")
that perform the operations vector addition, vector subtraction, and
multiplying a vector by a scalar:

$ mat(delim: #none, (x_(1), y_(1))+(x_(2), y_(2)), =, (x_(1)+x_(2), y_(1)+y_(2)); (x_(1), y_(1))-(x_(2), y_(2)), =, (x_(1)-x_(2), y_(1)-y_(2)); s dot.op (x, y), =, (s x, s y)) $
])

#exercise(label-name: <ex:implement-frames>, [
Here are two possible constructors for frames:
#idx("makeframe", decl: true)
#snippet(```python
def make_frame(origin, edge1, edge2):
    return llist(origin, edge1, edge2)

def make_frame(origin, edge1, edge2):
    return pair(origin, pair(edge1, edge2))
```)

For each constructor supply the appropriate selectors to produce an
implementation for frames.

#anchor(<ex:2_47>)
])

#subheading([Painters])

A painter is represented as a
#idx("painter(s)", sub: "represented as functions")
function
that, given a frame as argument, draws a particular image shifted and
scaled to fit the frame. That is to say, if
#py("p") is a painter and
#py("f") is a frame, then we produce
#py("p")'s image in
#py("f") by calling #py("p")
with #py("f") as argument.

The details of how primitive painters are implemented depend on the
particular characteristics of the graphics system and the type of image to
be drawn. For instance, suppose we have a
function
#idx("drawline")
#py("draw_line")
that draws a line on the screen between two specified points. Then we can
create painters for line drawings, such as the
#py("wave")
painter in figure @fig:wave, from linked lists of line
segments as follows:#footnote[The function #py("segments_to_painter")
uses the representation for line segments described in
exercise @ex:segments2 below. It also uses the
#py("for_each")
function
described in exercise @ex:for-each.]

#idx("segmentstopainter", decl: true)
#snippet(```python
def segments_to_painter(segment_list):
    return lambda frame:
             for_each(lambda segment:
                        draw_line(
                            frame_coord_map(frame)
                                (start_segment(segment)),
                            frame_coord_map(frame)
                                (end_segment(segment))),
                      segment_list)
```)

The segments are given using coordinates with respect to the unit square.
For each segment in the linked list, the painter transforms the segment endpoints
with the frame coordinate map and draws a line between the transformed
points.

Representing painters as
functions
erects a powerful abstraction barrier in the picture language. We can
create and intermix all sorts of primitive painters, based on a variety of
graphics capabilities. The details of their implementation do not matter.
Any
function
can serve as a painter, provided that it takes a frame as argument and
draws something scaled to fit the frame.#footnote[For example, the #py("rogers") painter of
figure @fig:rogers was constructed from a gray-level
image. For each point in a given frame, the
#py("rogers") painter determines the point in
the image that is mapped to it under the frame coordinate map, and
shades it accordingly.

By allowing different types of painters, we are capitalizing on the
abstract data idea discussed in section @sec:data-,
where we argued that a rational-number representation could be anything at
all that satisfies an appropriate condition. Here we're using the
fact that a painter can be implemented in any way at all, so long as it
draws something in the designated frame.

Section @sec:data- also showed how pairs could be
implemented as
functions.
Painters are our second example of a
functional
representation for data.]

#exercise(label-name: <ex:segments2>, [
A directed line segment in the plane can be represented as a pair of
#idx("line segment", sub: "represented as pair of vectors")
vectors—the vector running from the origin to the start-point of
the segment, and the vector running from the origin to the end-point of
the segment. Use your vector representation from
exercise @ex:vectors to define a representation for
segments with a constructor
#idx("makesegment")
#py("make_segment")
and selectors
#idx("startsegment")
#py("start_segment")
and
#idx("endsegment")
#py("end_segment").
])

#exercise(label-name: <ex:making-wave>, [
Use
#py("segments_to_painter")
to define the following primitive painters:

+ The painter that draws the outline of the designated frame.
+ The painter that draws an "X" by connecting opposite corners of the frame.
+ The painter that draws a diamond shape by connecting the midpoints of the sides of the frame.
+ The #py("wave") painter.
])

#subheading([Transforming and combining painters])

#idx("painter(s)", sub: "transforming and combining")

An operation on painters (such as
#py("flip_vert")
or #py("beside"))
works by creating a painter that invokes the original painters with respect
to frames derived from the argument frame. Thus, for example,
#py("flip_vert")
doesn't have to know how a painter works in order to flip
it—it just has to know how to turn a frame upside down: The flipped
painter just uses the original painter, but in the inverted frame.

Painter operations are based on the
function
#py("transform_painter"),
which takes as arguments a painter and information on how to transform a
frame and produces a new painter. The transformed painter, when called on
a frame, transforms the frame and calls the original painter on the
transformed frame. The arguments to
#py("transform_painter")
are points (represented as vectors) that specify the corners of the new
frame: When mapped into the frame, the first point specifies the new
frame's origin and the other two specify the ends of its edge vectors.
Thus, arguments within the unit square specify a frame contained within the
original frame.
#idx("transformpainter", decl: true)
#snippet(```python
def transform_painter(painter, origin, corner1, corner2):
    def transformed(frame):
        m = frame_coord_map(frame)
        new_origin = m(origin)
        return painter(make_frame(
                           new_origin,
                           sub_vect(m(corner1), new_origin),
                           sub_vect(m(corner2), new_origin)))
    return transformed
```)

Here's how to flip painter images vertically:
#idx("flipvert", decl: true)
#snippet(```python
def flip_vert(painter):
    return transform_painter(painter,
                             make_vect(0, 1),  # new origin
                             make_vect(1, 1),  # new end of edge1
                             make_vect(0, 0)); # new end of edge2
```)

Using
#py("transform_painter"), we can easily define new transformations. For example, we can declare a painter that shrinks its image to the upper-right quarter of the frame it is given:
#idx("shrinktoupperright", decl: true)
#snippet(```python
def shrink_to_upper_right(painter):
    return transform_painter(painter,
                             make_vect(0.5, 0.5),
                             make_vect(1, 0.5),
                             make_vect(0.5, 1))
```)

Other transformations rotate images counterclockwise by 90
degrees#footnote[The function #py("rotate90")
is a pure rotation only for square frames, because it also stretches and
shrinks the image to fit into the rotated frame.]
#idx("rotate90", decl: true)
#snippet(```python
def rotate90(painter):
    return transform_painter(painter,
                             make_vect(1, 0),
                             make_vect(1, 1),
                             make_vect(0, 0))
```)

or squash images towards the center of the frame:#footnote[The diamond-shaped images in
figures @fig:wave
and @fig:rogers were created with
#py("squash_inwards")
applied to #py("wave") and
#py("rogers").]
#idx("squashinwards", decl: true)
#snippet(```python
def squash_inwards(painter):
    return transform_painter(painter,
                             make_vect(0, 0),
                             make_vect(0.65, 0.35),
                             make_vect(0.35, 0.65))
```)

Frame transformation is also the key to
defining means of combining two or more painters.
The #py("beside")
function,
for example, takes two painters, transforms them to paint in the left and
right halves of an argument frame respectively, and produces a new,
compound painter. When the compound painter is given a frame, it calls the
first transformed painter to paint in the left half of the frame and calls
the second transformed painter to paint in the right half of the frame:
#idx("beside", decl: true)
#snippet(```python
def beside(painter1, painter2):
    split_point = make_vect(0.5, 0)
    paint_left  = transform_painter(painter1,
                                    make_vect(0, 0),
                                    split_point,
                                    make_vect(0, 1))
    paint_right = transform_painter(painter2,
                                    split_point,
                                    make_vect(1, 0),
                                    make_vect(0.5, 1))
    def painter(frame):
        paint_left(frame)
        paint_right(frame)
    return painter
```)

Observe how the painter data abstraction, and in particular the
representation of painters as
functions,
makes
#py("beside") easy to implement. The
#py("beside")
function
need not know anything about the details of the component painters other
than that each painter will draw something in its designated frame.

#exercise(label-name: <ex:rotate>, [
Declare
the transformation
#idx("fliphoriz")
#py("flip_horiz"),
which flips painters horizontally, and transformations that rotate painters
counterclockwise by 180 degrees and 270 degrees.
])

#exercise(label-name: <ex:below>, [
Declare
the
#idx("below")
#py("below") operation for painters.
The function #py("below")
takes two painters as arguments. The resulting painter, given a frame,
draws with the first painter in the bottom of the frame and with the
second painter in the top.
Define #py("below") in two different
ways—first by writing a
function
that is analogous to the
#py("beside")
function
given above, and again in terms of #py("beside") and
suitable rotation operations (from exercise @ex:rotate).
])

#subheading([Levels of language for robust design])

The picture language exploits some of the critical ideas we've
introduced about abstraction with
functions
and data. The fundamental data abstractions, painters, are implemented
using
functional
representations, which enables the language to handle different basic
drawing capabilities in a uniform way. The means of combination satisfy
the closure property, which permits us to easily build up complex designs.
Finally, all the tools for abstracting
functions
are available to us for abstracting means of combination for painters.

We have also obtained a glimpse of another crucial idea about languages and
program design. This is the approach of
#idx("stratified design")
#idx("design, stratified")
#emph[stratified design], the notion that a complex system should be
structured as a sequence of levels that are described using a sequence of
languages. Each level is constructed by combining parts that are regarded
as primitive at that level, and the parts constructed at each level are
used as primitives at the next level. The language used at each level
of a stratified design has primitives, means of combination, and means
of abstraction appropriate to that level of detail.

Stratified design pervades the engineering of complex systems. For
example, in computer engineering, resistors and transistors are
combined (and described using a language of analog circuits) to
produce parts such as and-gates and or-gates, which form the
primitives of a language for digital-circuit design.#footnote[Section @sec:circuit-simulator describes one such
language.] These parts are combined to build
processors, bus structures, and memory systems, which are in turn
combined to form computers, using languages appropriate to computer
architecture. Computers are combined to form distributed systems,
using languages appropriate for describing network interconnections,
and so on.

As a tiny example of stratification, our picture language uses primitive
elements (primitive painters) that specify points and lines to provide the
shapes of a painter like #py("rogers"). The bulk of
our description of the picture language focused on combining these
primitives, using geometric combiners such as
#py("beside") and #py("below").
We also worked at a higher level, regarding
#py("beside") and #py("below")
as primitives to be manipulated in a language whose operations, such as
#py("square_of_four"),
capture common patterns of combining geometric combiners.

Stratified design helps make programs
#idx("robustness")
#emph[robust], that is, it makes
it likely that small changes in a specification will require
correspondingly small changes in the program. For instance, suppose we
wanted to change the image based on #py("wave")
shown in figure @fig:sqlimit-designs. We could work
at the lowest level to change the detailed appearance of the
#py("wave") element; we could work at the middle
level to change the way
#py("corner_split")
replicates the #py("wave"); we could work at the
highest level to change how
#py("square_limit")
arranges the four copies of the corner. In general, each level of a
stratified design provides a different vocabulary for expressing the
characteristics of the system, and a different kind of ability to change it.

#exercise(label-name: <ex:2_52>, [
Make changes to the square limit of #py("wave")
shown in figure @fig:sqlimit-designs by working at
each of the levels described above. In particular:

+ Add some segments to the primitive #py("wave") painter of exercise @ex:making-wave (to add a smile, for example).
+ Change the pattern constructed by #py("corner_split") (for example, by using only one copy of the #py("up_split") and #py("right_split") images instead of two).
+ Modify the version of #idx("squarelimit") #py("square_limit") that uses #idx("squareoffour") #py("square_of_four") so as to assemble the corners in a different pattern. (For example, you might make the big Mr. Rogers look outward from each corner of the square.)
])

#idx("picture language")
