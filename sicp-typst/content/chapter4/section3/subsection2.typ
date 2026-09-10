// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Examples of Nondeterministic Programs], label-name: <sec:amb-examples>)

Section @sec:amb-implementation describes the
implementation of the #py("amb") evaluator. First,
however, we give some examples of how it can be used. The advantage of
nondeterministic programming is that we can suppress the details of how
search is carried out, thereby
expressing our programs at a higher level of
#idx("abstraction", sub: "of search in nondeterministic programming")
abstraction.

#subheading([Logic Puzzles])

#idx("puzzles", sub: "logic puzzles")
#idx("logic puzzles")
#idx("nondeterministic programs", sub: "logic puzzles")

The following puzzle (adapted from
#idx("Dinesman, Howard P.")
Dinesman 1968)
is typical of a large class of simple logic puzzles:

#blockquote[The software company
#idx("Gargle")
Gargle is expanding, and Alyssa, Ben, Cy, Lem, and Louis
are moving into a row of five private offices in a
new building. Alyssa does not move into the last office. Ben does not
move into the first office. Cy takes neither the first nor the last office.
Lem moves into an office after Ben's. Louis's office is not next to
Cy's. Cy's office is not next to Ben's. Who moves into which office?]

We can determine who moves into which office in a straightforward way by
enumerating all the possibilities and imposing the given
restrictions:#footnote[Our program uses the following
function
to determine if the elements of a list are distinct:

#idx("distinct", decl: true)
#snippet(```python
def distinct(items):
    return True if is_none(items) else True if is_none(tail(items)) else distinct(tail(items)) if is_none(member(head(items), tail(items))) else False
```)]
#idx("officemove", decl: true)
#snippet(```python
def office_move():
    alyssa = amb(1, 2, 3, 4, 5)
    ben = amb(1, 2, 3, 4, 5)
    cy = amb(1, 2, 3, 4, 5)
    lem = amb(1, 2, 3, 4, 5)
    louis = amb(1, 2, 3, 4, 5)
    require(distinct(llist(alyssa, ben, cy, lem, louis)))
    require(alyssa != 5)
    require(ben != 1)
    require(cy != 5)
    require(cy != 1)
    require(lem > ben)
    require(abs(louis - cy) != 1)
    require(abs(cy - ben) != 1)
    return llist(llist("alyssa", alyssa), llist("ben", ben), llist("cy", cy), llist("lem", lem), llist("louis", louis))
```)

Evaluating the expression
#py("office_move()")
produces the result

#snippet(```python
llist(llist("alyssa", 3), llist("ben", 2), llist("cy", 4),
     llist("lem", 5), llist("louis", 1))
```)

Although this simple
function
works, it is very slow.
Exercises @ex:better-office-move1
and @ex:better-office-move2 discuss some possible
improvements.

#exercise(label-name: <ex:office_move_1>, [
Modify the office-move
function
to omit the requirement that Louis's office is not next to Cy's.
How many solutions are there to this modified puzzle?
])

#exercise(label-name: <ex:better-office-move1>, [
Does the order of the restrictions in the office-move
function
affect the answer? Does it affect the time to find an answer? If you
think it matters, demonstrate a faster program obtained from the given
one by reordering the restrictions. If you think it does not matter,
argue your case.
])

#exercise(label-name: <ex:better-office-move2>, [
In the office move problem, how many sets of assignments are
there of people to offices, both before and after the requirement that
office assignments be distinct? It is very inefficient to generate all
possible assignments of people to offices and then leave it to
backtracking to eliminate them. For example, most of the restrictions
depend on only one or two of the person-office
names,
and can thus be imposed before offices have been selected for all the people.
Write and demonstrate a much more efficient nondeterministic
function
that solves this problem based upon generating only those possibilities that
are not already ruled out by previous restrictions.
])

#exercise(label-name: <ex:office_move_4>, [
Write an ordinary
#idx("nondeterministic programming vs. Python programming")
Python
program to solve the office move puzzle.
])

#exercise(label-name: <ex:liars>, [
Solve the following "Liars" puzzle (adapted from
#idx("Phillips, Hubert")
Phillips 1934):

#blockquote[Alyssa, Cy, Eva, Lem, and Louis meet for a business lunch at SoSoService.
Their meals arrive one after the other, a considerable time after they
placed their orders. To entertain Ben, who expects them back at the office
for a meeting, they decide to each make one true statement and one false
statement about their orders:

- Alyssa: "Lem's meal arrived second. Mine arrived third."
- Cy: "Mine arrived first. Eva's arrived second."
- Eva: "Mine arrived third, and poor Cy's arrived last."
- Lem: "Mine arrived second. Louis's arrived fourth."
- Louis: "Mine arrived fourth. Alyssa's meal arrived first."

What was the real order in which the five diners received their meals?]
])

#exercise(label-name: <ex:checking>, [
Use the #py("amb") evaluator to solve the following
puzzle (adapted from
#idx("Phillips, Hubert")
Phillips 1961):

#blockquote[Alyssa, Ben, Cy, Eva, and Louis each pick a different chapter of SICP JS
and solve all the exercises in that chapter.
Louis solves the exercises in the "Functions" chapter,
Alyssa the ones in the "Data" chapter, and
Cy the ones in the "State" chapter.
They decide to check each other's work, and
Alyssa volunteers to check the exercises in the "Meta" chapter.
The exercises in the "Register Machines" chapter are solved by Ben
and checked by Louis.
The person who checks the exercises in the "Functions" chapter
solves the exercises that are checked by Eva.
Who checks the exercises in the	"Data" chapter?]

Try to write the program so that it runs efficiently (see
exercise @ex:better-office-move2). Also determine
how many solutions there are if we are not told that Alyssa checks the
exercises in the "Meta" chapter.
])

#idx("puzzles", sub: "logic puzzles")
#idx("logic puzzles")
#idx("nondeterministic programs", sub: "logic puzzles")

#exercise(label-name: <ex:queens_amb>, [
Exercise @ex:8queens described the
#idx("chess, eight-queens puzzle")
#idx("eight-queens puzzle")
#idx("puzzles", sub: "eight-queens puzzle")
#idx("nondeterministic programming vs. Python programming")
"eight-queens puzzle" of placing queens on a chessboard so that
no two attack each other. Write a nondeterministic program to solve this
puzzle.
])

#subheading([Parsing natural language])

#idx("parsing natural language")
#idx("nondeterministic programs", sub: "parsing natural language")

Programs designed to accept natural language as input usually start by
attempting to #emph[parse] the input, that is, to match the input
against some grammatical structure. For example, we might try to
recognize simple sentences consisting of an article followed by a noun
followed by a verb, such as "The cat eats." To accomplish
such an analysis, we must be able to identify the parts of speech of
individual words. We could start with some lists that classify various
words:#footnote[Here we use the convention that the first element of each
list designates the part of speech for the rest of the words in the
list.]
#idx("nouns", decl: true)#idx("verbs", decl: true)#idx("articles", decl: true)
#snippet(```python
nouns = llist("noun", "student", "professor", "cat", "class")
verbs = llist("verb", "studies", "lectures", "eats", "sleeps")
articles = llist("article", "the", "a")
```)

We also need a
#idx("grammar")
#emph[grammar], that is, a set of rules describing how
grammatical elements are composed from simpler elements. A very
simple grammar might stipulate that a sentence always consists of two
pieces—a noun phrase followed by a verb—and that a noun
phrase consists of an article followed by a noun. With this grammar, the
sentence "The cat eats" is parsed as follows:

#snippet(```python
llist("sentence",
     llist("noun-phrase", llist("article", "the"), llist("noun", "cat"),
     llist("verb", "eats"))
```)

#idx("parse...")

We can generate such a parse with a simple program that has separate
functions
for each of the grammatical rules. To parse a sentence, we identify its
two constituent pieces and return a list of these two elements, tagged with
the symbol #py("sentence"):

#snippet(```python
def parse_sentence():
    return llist("sentence", parse_noun_phrase(), parse_word(verbs))
```)

A noun phrase, similarly, is parsed by finding an article followed by a
noun:

#snippet(```python
def parse_noun_phrase():
    return llist("noun-phrase", parse_word(articles), parse_word(nouns))
```)

At the lowest level, parsing boils down to repeatedly checking that
the next
not-yet-parsed
word is a member of the list of words for the
required part of speech. To implement this, we maintain a global
variable
#py("not_yet_parsed"),
which is the input that has not yet been parsed. Each time we check a word,
we require that
#py("not_yet_parsed")
must be nonempty and that it should begin with a word from the designated
list. If so, we remove that word from
#py("not_yet_parsed")
and return the word together with its part of speech (which is found at
the head of the list):#footnote[Notice that
#py("parse_word")
uses
assignment
to modify the
not-yet-parsed
input list. For this to work, our
#py("amb") evaluator must undo the effects of
assignments
when it backtracks.]

#snippet(```python
def parse_word(word_list):
    require( not is_none(not_yet_parsed))
    require( not is_none(member(head(not_yet_parsed), tail(word_list))))
    found_word = head(not_yet_parsed)
    not_yet_parsed = tail(not_yet_parsed)
    return llist(head(word_list), found_word)
```)

To start the parsing, all we need to do is set
#py("not_yet_parsed")
to be
the entire input, try to parse a sentence, and check that nothing is
left over:

#snippet(```python
not_yet_parsed = None
```)

#snippet(```python
def parse_input(input):
    not_yet_parsed = input
    sent = parse_sentence()
    require(is_none(not_yet_parsed))
    return sent
```)

We can now try the parser and verify that it works for our simple test
sentence:

#prompt(```python
amb-evaluate input:
```)

#snippet(```python
parse_input(llist("the",  "cat",  "eats"))
```)

#output(```python
parse_input(llist("the",  "cat",  "eats"))
```)

The #py("amb") evaluator is useful here because it is
convenient to express the parsing constraints with the aid of
#py("require"). Automatic search and backtracking
really pay off, however, when we consider more complex grammars where there
are choices for how the units can be decomposed.

Let's add to our grammar a list of prepositions:
#idx("prepositions", decl: true)
#snippet(```python
prepositions = llist("prep", "for", "to", "in", "by", "with")
```)

and define a prepositional phrase (e.g., "for the cat") to be
a preposition followed by a noun phrase:

#snippet(```python
def parse_prepositional_phrase():
    return llist("prep-phrase", parse_word(prepositions), parse_noun_phrase())
```)

Now we can define a sentence to be a noun phrase followed by a verb
phrase, where a verb phrase can be either a verb or a verb phrase
extended by a prepositional phrase:#footnote[Observe that this
definition is recursive—a verb may be followed by any number
of prepositional phrases.]

#snippet(```python
def parse_sentence():
    return llist("sentence", parse_noun_phrase(), parse_verb_phrase())
def parse_verb_phrase():
    def maybe_extend(verb_phrase):
        return amb(verb_phrase, maybe_extend(llist("verb-phrase", verb_phrase, parse_prepositional_phrase())))
    return maybe_extend(parse_word(verbs))
```)

While we're at it, we can also elaborate the definition of noun
phrases to permit such things as "a cat in the class." What
we used to call a noun phrase, we'll now call a simple noun phrase,
and a noun phrase will now be either a simple noun phrase or a noun phrase
extended by a prepositional phrase:

#snippet(```python
def parse_simple_noun_phrase():
    return llist("simple-noun-phrase", parse_word(articles), parse_word(nouns))
def parse_noun_phrase():
    def maybe_extend(noun_phrase):
        return amb(noun_phrase, maybe_extend(llist("noun-phrase", noun_phrase, parse_prepositional_phrase())))
    return maybe_extend(parse_simple_noun_phrase())
```)

#idx("parse...")

Our new grammar lets us parse more complex sentences. For example

#snippet(```python
parse_input(llist("the", "student", "with", "the", "cat",
                 "sleeps", "in", "the", "class"))
```)

produces

#snippet(```python
llist("sentence",
     llist("noun-phrase",
          llist("simple-noun-phrase",
               llist("article", "the"), llist("noun", "student")),
          llist("prep-phrase", llist("prep", "with"),
               llist("simple-noun-phrase",
                    llist("article", "the"),
                    llist("noun", "cat")))),
     llist("verb-phrase",
          llist("verb", "sleeps"),
          llist("prep-phrase", llist("prep", "in"),
               llist("simple-noun-phrase",
                    llist("article", "the"),
                    llist("noun", "class")))))
```)

Observe that a given input may have more than one legal parse. In the
sentence "The professor lectures to the student with the cat,"
it may be that the professor is lecturing with the cat, or that the student
has the cat. Our nondeterministic program finds both possibilities:

#snippet(```python
parse_input(llist("the", "professor", "lectures",
                 "to", "the", "student", "with", "the", "cat"))
```)

produces

#snippet(```python
llist("sentence",
     llist("simple-noun-phrase",
          llist("article", "the"), llist("noun", "professor")),
     llist("verb-phrase",
          llist("verb-phrase",
               llist("verb", "lectures"),
               llist("prep-phrase", llist("prep", "to"),
                    llist("simple-noun-phrase",
                    llist("article", "the"),
            llist("noun", "student")))),
          llist("prep-phrase", llist("prep", "with"),
               llist("simple-noun-phrase",
                    llist("article", "the"),
                    llist("noun", "cat")))))
```)

Asking the evaluator to retry yields

#snippet(```python
llist("sentence",
     llist("simple-noun-phrase",
          llist("article", "the"), llist("noun", "professor")),
     llist("verb-phrase",
          llist("verb", "lectures"),
          llist("prep-phrase", llist("prep", "to"),
               llist("noun-phrase",
                    llist("simple-noun-phrase",
                         llist("article", "the"),
                         llist("noun", "student")),
                    llist("prep-phrase", llist("prep", "with"),
                         llist("simple-noun-phrase",
                              llist("article", "the"),
                              llist("noun", "cat")))))))
```)

#idx("nondeterministic programs", sub: "parsing natural language")

#exercise(label-name: <ex:five_ways>, [
With the grammar given above, the following sentence can be parsed in five
different ways: "The professor lectures to the student in the class with the cat." Give the five parses and explain the differences in
shades of meaning among them.
])

#exercise(label-name: <ex:ordered_parsing>, [
The
#idx("nondeterministic evaluator", sub: "order of argument evaluation")
evaluators in sections @sec:mc-eval and
@sec:lazy-evaluation do not determine what order
argument expressions
are
evaluated in. We will see that the #py("amb") evaluator
evaluates them from left to right. Explain why our parsing program
wouldn't work if the
argument expressions
were evaluated in some other order.
])

#exercise(label-name: <ex:louis_verb_phrase>, [
Louis Reasoner suggests that, since a verb phrase is either a verb or
a verb phrase followed by a prepositional phrase, it would be much more
straightforward to
declare
the
function
#py("parse_verb_phrase")
as follows (and similarly for noun phrases):

#snippet(```python
def parse_verb_phrase():
    return amb(parse_word(verbs), llist("verb-phrase", parse_verb_phrase(), parse_prepositional_phrase()))
```)

Does this work? Does the program's behavior change if we interchange
the order of expressions in the #py("amb")?
])

#exercise(label-name: <ex:complex_sentences>, [
Extend the grammar given above to handle more complex sentences. For
example, you could extend noun phrases and verb phrases to include adjectives
and adverbs, or you could handle compound sentences.#footnote[This kind of
grammar can become arbitrarily complex, but it
is only a
#idx("parsing natural language", sub: "real language understanding vs. toy parser")
toy as far as real language understanding is concerned.
Real natural-language understanding by computer requires an elaborate
mixture of syntactic analysis and interpretation of meaning. On the
other hand, even toy parsers can be useful in supporting flexible
command languages for programs such as information-retrieval systems.
#idx("Winston, Patrick Henry")
Winston 1992 discusses computational approaches to
real language understanding and also the applications of simple grammars
to command languages.]
])

#exercise(label-name: <ex:sentence-generate>, [
Alyssa P. Hacker is more interested in
#idx("generating sentences")
generating interesting sentences
than in parsing them. She reasons that by simply changing the
function #py("parse_word")
so that it ignores the "input sentence" and instead always
succeeds and generates an appropriate word, we can use the programs we had
built for parsing to do generation instead. Implement Alyssa's idea,
and show the first half-dozen or so sentences generated.#footnote[Although
Alyssa's idea works just fine (and is surprisingly simple), the
sentences that it generates are a bit boring—they don't
sample the possible sentences of this language in a very interesting way.
In fact, the grammar is highly recursive in many places, and
Alyssa's technique "falls into" one of these recursions
and gets stuck. See exercise @ex:ramb for a way to deal
with this.]
])

#idx("parsing natural language")
#idx("nondeterministic computing")
