// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Deductive Information Retrieval], label-name: <sec:deductive-info-retrieval>)

#idx("query language")

Logic programming excels in providing interfaces to
#idx("data base", sub: "logic programming and")
data bases for
information retrieval. The query language we shall implement in this
chapter is designed to be used in this way.

In order to illustrate what the query system does, we will show how it
can be used to manage the data base of personnel records for
#idx("Gargle")
Gargle, a thriving high-technology company in the
Boston area. The language provides pattern-directed access to
personnel information and can also take advantage of general rules in
order to make logical deductions.

#subheading([A sample data base])

#idx("query language", sub: "data base")

#idx("data base", sub: "Gargle personnel")
The personnel data base for Gargle
contains
#idx("assertion")
#emph[assertions] about company personnel. Here is the
information about Ben Bitdiddle, the resident computer wizard:

#snippet(```python
address(llist("Bitdiddle", "Ben"),
        llist("Slumerville", llist("Ridge", "Road"), 10))
job(llist("Bitdiddle", "Ben"), llist("computer", "wizard"))
salary(llist("Bitdiddle", "Ben"), 122000)
```)

Assertions look just like function applications in Python, but
they actually represent information in the data base. The first
symbols—here #py("address"),
#py("job") and
#py("salary")—describe the
#emph[kind of information] contained in the respective assertion, and
the "arguments" are lists or primitive values such as strings
and numbers. The first symbols do not need to be declared, as do constants
or variables in Python; their scope is global.

As resident wizard, Ben is in charge of the company's computer
division, and he supervises two programmers and one technician. Here
is the information about them:

#snippet(```python
address(llist("Hacker", "Alyssa", "P"),
        llist("Cambridge", llist("Mass", "Ave"), 78))
job(llist("Hacker", "Alyssa", "P"), llist("computer", "programmer"))
salary(llist("Hacker", "Alyssa", "P"), 81000)
supervisor(llist("Hacker", "Alyssa", "P"), llist("Bitdiddle", "Ben"))

address(llist("Fect", "Cy", "D"),
        llist("Cambridge", llist("Ames", "Street"), 3))
job(llist("Fect", "Cy", "D"), llist("computer", "programmer"))
salary(llist("Fect", "Cy", "D"), 70000)
supervisor(llist("Fect", "Cy", "D"), llist("Bitdiddle", "Ben"))

address(llist("Tweakit", "Lem", "E"),
        llist("Boston", llist("Bay", "State", "Road"), 22))
job(llist("Tweakit", "Lem", "E"), llist("computer", "technician"))
salary(llist("Tweakit", "Lem", "E"), 51000)
supervisor(llist("Tweakit", "Lem", "E"), llist("Bitdiddle", "Ben"))
```)

There is also a programmer trainee, who is supervised by Alyssa:

#snippet(```python
address(llist("Reasoner", "Louis"),
        llist("Slumerville", llist("Pine", "Tree", "Road"), 80))
job(llist("Reasoner", "Louis"),
         llist("computer", "programmer", "trainee"))
salary(llist("Reasoner", "Louis"), 62000)
supervisor(llist("Reasoner", "Louis"), llist("Hacker", "Alyssa", "P"))
```)

All these people are in the computer division, as indicated by the
word
#py("\"computer\"")
as the first item in their job
descriptions.

Ben is a high-level employee. His supervisor is the company's big
wheel himself:

#snippet(```python
supervisor(llist("Bitdiddle", "Ben"), llist("Warbucks", "Oliver"))

address(llist("Warbucks", "Oliver"),
        llist("Swellesley", llist("Top", "Heap", "Road")))
job(llist("Warbucks", "Oliver"), llist("administration", "big", "wheel"))
salary(llist("Warbucks", "Oliver"), 314159)
```)

Besides the computer division supervised by Ben, the company has an
accounting division, consisting of a chief accountant and his
assistant:

#snippet(```python
address(llist("Scrooge", "Eben"),
        llist("Weston", llist("Shady", "Lane"), 10))
job(llist("Scrooge", "Eben"), llist("accounting", "chief", "accountant"))
salary(llist("Scrooge", "Eben"), 141421)
supervisor(llist("Scrooge", "Eben"), llist("Warbucks", "Oliver"))

address(llist("Cratchit", "Robert"),
        llist("Allston", llist("N", "Harvard", "Street"), 16))
job(llist("Cratchit", "Robert"), llist("accounting", "scrivener"))
salary(llist("Cratchit", "Robert"), 26100)
supervisor(llist("Cratchit", "Robert"), llist("Scrooge", "Eben"))
```)

There is also
an administrative assistant
for the big wheel:

#snippet(```python
address(llist("Aull", "DeWitt"),
        llist("Slumerville", llist("Onion", "Square"), 5))
job(llist("Aull", "DeWitt"), llist("administration", "assistant"))
salary(llist("Aull", "DeWitt"), 42195)
supervisor(llist("Aull", "DeWitt"), llist("Warbucks", "Oliver"))
```)

The data base also contains assertions about which kinds of jobs can
be done by people holding other kinds of jobs. For instance, a
computer wizard can do the jobs of both a computer programmer and a
computer technician:

#snippet(```python
can_do_job(llist("computer", "wizard"),
           llist("computer", "programmer"))
can_do_job(llist("computer", "wizard"),
           llist("computer", "technician"))
```)

A computer programmer could fill in for a trainee:

#snippet(```python
can_do_job(llist("computer", "programmer"),
           llist("computer", "programmer", "trainee"))
```)

#idx("administrative assistant, importance of")
Also, as is well known,

#snippet(```python
can_do_job(llist("administration", "assistant"),
           llist("administration", "big", "wheel"))
```)

#idx("data base", sub: "Gargle personnel")
#idx("query language", sub: "data base")

#subheading([Simple queries])

#idx("simple query")

The query language allows users to retrieve information from the data
base by posing queries in response to the system's prompt.
For example, to find all computer programmers one can say

#prompt(```python
Query input:
```)

#snippet(```python
job($x, llist("computer", "programmer"))
```)

The system will respond with the following items:

#output(```python
job($x, llist("computer", "programmer"))
```)

The input query specifies that we are looking for entries in the data
base that match a certain
#idx("pattern")
#emph[pattern].

In this example, the pattern
specifies #py("job") as the kind of information
that we are looking for. The first item can be
anything, and the second is the literal list
#py("llist(\"computer\", \"programmer\")").
The "anything" that can be the first item in the matching
assertion is specified by a
#idx("pattern variable")
#emph[pattern variable],
#py("$x"). As pattern variables, we use
#idx("naming conventions", sub: "$ for pattern variables")
#idx("$, pattern variables starting with", sort: "0a4")
Python names that start with a dollar sign.
We will see below why
it is useful to specify names for pattern variables rather than just
putting a single symbol such as #py("$")
into patterns to represent "anything."

The system responds to a simple query by showing all entries in the data
base that match the specified pattern.

A pattern can have more than one variable. For example, the query

#snippet(```python
address($x, $y)
```)

will list all the employees' addresses.

A pattern can have no variables, in which case the query simply
determines whether that pattern is an entry in the data base. If so,
there will be one match; if not, there will be no matches.

The same pattern variable can appear more than once in a query,
specifying that the same "anything" must appear in each
position. This is why variables have names. For example,

#snippet(```python
supervisor($x, $x)
```)

finds all people who supervise themselves (though there are no
such assertions in our sample data base).

The query

#snippet(```python
job($x, llist("computer", $type))
```)

matches all job entries whose
second
item is a two-element list whose
first item is
#py("\"computer\""):

#snippet(```python
job(llist("Bitdiddle", "Ben"), llist("computer", "wizard"))
job(llist("Hacker", "Alyssa", "P"), llist("computer", "programmer"))
job(llist("Fect", "Cy", "D"), llist("computer", "programmer"))
job(llist("Tweakit", "Lem", "E"), llist("computer", "technician"))
```)

This same pattern does #emph[not] match

#snippet(```python
job(llist("Reasoner", "Louis"),
    llist("computer", "programmer", "trainee"))
```)

because the second item in the assertion is a list of three
elements, and the pattern's second item specifies that there
should be two elements. If we wanted to change the pattern so that the
second item could be any
list beginning with
#py("\"computer\""),
we could
specify

#snippet(```python
job($x, pair("computer", $type))
```)

For example,

#snippet(```python
pair("computer", $type)
```)

matches the data

#snippet(```python
llist("computer", "programmer", "trainee")
```)

with
#py("$type")
as
#py("llist(\"programmer\", \"trainee\")").
It also matches the data

#snippet(```python
llist("computer", "programmer")
```)

with
#py("$type")
as
#py("llist(\"programmer\")"),
and matches the data

#snippet(```python
llist("computer")
```)

with
#py("$type")
as the empty
list, #py("None").

#idx("pattern")

We can describe the query language's processing of simple queries as
follows:

- The system finds all assignments to variables in the query pattern that #idx("satisfy a pattern (simple query)") #emph[satisfy] the pattern—that is, all sets of values for the variables such that if the pattern variables are #idx("instantiate a pattern") #emph[instantiated with] (replaced by) the values, the result is in the data base.
- The system responds to the query by listing all instantiations of the query pattern with the variable assignments that satisfy it.

Note that if the pattern has no variables, the query reduces to a
determination of whether that pattern is in the data base. If so, the
empty assignment, which assigns no values to variables, satisfies that
pattern for that data base.

#exercise(label-name: <ex:4_53>, [
Give simple queries that retrieve the following information from the
data base:

+ all people supervised by Ben Bitdiddle;
+ the names and jobs of all people in the accounting division;
+ the names and addresses of all people who live in Slumerville.
])

#idx("simple query")

#subheading([Compound queries])

#idx("compound query")

Simple queries form the primitive operations of the query language.
In order to form compound operations, the query language provides
means of combination. One thing that makes the query language a logic
programming language is that the means of combination mirror the means
of combination used in forming logical expressions:
#py("and"), #py("or"), and
#py("not").

We can use
#idx("and (query language)")
#py("and") as follows to find the addresses
of all the computer programmers:

#snippet(```python
and(job($person, llist("computer", "programmer")),
    address($person, $where))
```)

The resulting output is

#output(```python
and(job($person, llist("computer", "programmer")),
    address($person, $where))
```)

In general,

#syntax("
and(", meta("query"), $""_(1)$, ", ", meta("query"), $""_(2)$, ", ", $dots.h$, ", ", meta("query"), $""_(n))$)

is
#idx("satisfy a compound query")
satisfied by all sets of values for the pattern variables that
simultaneously satisfy
#meta("query")$""_(1), dots.h ,$ #meta("query")$""_(n)$.

As for simple queries, the system processes a compound query by
finding all assignments to the pattern variables that satisfy the
query, then displaying instantiations of the query with those values.

Another means of constructing compound queries is through
#idx("or (query language)")
#py("or"). For example,

#snippet(```python
or(supervisor($x, llist("Bitdiddle", "Ben")),
   supervisor($x, llist("Hacker", "Alyssa", "P")))
```)

will find all employees supervised by Ben Bitdiddle or Alyssa P.
Hacker:

#output(```python
or(supervisor($x, llist("Bitdiddle", "Ben")),
   supervisor($x, llist("Hacker", "Alyssa", "P")))
```)

In general,

#syntax("
or(", meta("query"), $""_(1)$, ", ", meta("query"), $""_(2)$, ", ", $dots.h$, ", ", meta("query"), $""_(n)$, ")
	  ")

is satisfied by all sets of values for the pattern variables that
satisfy at least one of
#meta("query")$""_(1) dots.h$ #meta("query")$""_(n)$.

Compound queries can also be formed with
#idx("not (query language)")
#py("not").
For example,

#snippet(```python
and(supervisor($x, llist("Bitdiddle", "Ben")),
    not(job($x, llist("computer", "programmer"))))
```)

finds all people supervised by Ben Bitdiddle who are not computer
programmers. In general,

#syntax("
not(", meta("query"), $""_(1)$, ")
	  ")

is satisfied by all assignments to the pattern variables that do not
satisfy
#meta("query")$""_(1)$.#footnote[Actually, this description of
#py("not") is valid only for simple cases. The real
behavior of #py("not") is more complex. We will
examine #py("not")'s peculiarities
in sections @sec:how-query-works
and @sec:math-logic.]

The final combining form starts with
#idx("javascriptpredicate (query language)")
#py("javascript_predicate") and
contains a Python predicate. In general,

#syntax("
javascript_predicate(", meta("predicate"), ")
	  ")

will be satisfied by assignments to the pattern variables
in the #meta("predicate") for which the
instantiated
#meta("predicate") is true.
For example, to find all people whose salary is greater than
\$50,000 we could write#footnote[A query should use
#py("javascript_predicate")
only to perform an operation not
provided in the query language. In particular,
#py("javascript_predicate")
should not be used to
#idx("query language", sub: "equality testing in")
test equality (since that is what the matching in the
query language is designed to do) or inequality (since that can
be done with the #py("same") rule shown
below).]

#snippet(```python
and(salary($person, $amount), javascript_predicate($amount > 50000))
```)

#idx("satisfy a compound query")

#exercise(label-name: <ex:4_54>, [
Formulate compound queries that retrieve the following information:

+ the names of all people who are supervised by Ben Bitdiddle, together with their addresses;
+ all people whose salary is less than Ben Bitdiddle's, together with their salary and Ben Bitdiddle's salary;
+ all people who are supervised by someone who is not in the computer division, together with the supervisor's name and job.
])

#idx("compound query")

#subheading([Rules])

#idx("rule (query language)")

In addition to primitive queries and compound queries, the query
language provides means for
#idx("query language", sub: "abstraction in")
abstracting queries. These are given by
#emph[rules]. The rule
#idx("livesnear (rule)", decl: true)
#snippet(```python
rule(lives_near($person_1, $person_2),
     and(address($person_1, pair($town, $rest_1)),
         address($person_2, pair($town, $rest_2)),
         not(same($person_1, $person_2))))
```)

specifies that two people live near each other if they live in the
same town. The final #py("not") clause prevents the
rule from saying that all people live near themselves. The
#py("same") relation is defined by a very simple
rule:#footnote[Notice that we do not need #py("same")
in order to make two things be the same: We just use the same pattern
variable for each—in effect, we have one thing instead of two things
in the first place. For example, see
#py("$town")
in the
#py("lives_near")
rule and
#py("$middle_manager")
in the
#py("wheel")
rule below.
The #py("same") relation
is useful when we want to force two things to be
different, such as
#py("$person_1")
and
#py("$person_2")
in the
#py("lives_near")
rule. Although using the same pattern variable in two
parts of a query forces the same value to appear in both places, using
different pattern variables does not force different values to appear.
(The values assigned to different pattern variables may be the same or
different.)]
#idx("same (rule)", decl: true)
#snippet(```python
rule(same($x, $x))
```)

The following rule declares that a person is a "wheel" in an
organization if he supervises someone who is in turn a supervisor:
#idx("wheel (rule)", decl: true)
#snippet(```python
rule(wheel($person),
     and(supervisor($middle_manager, $person),
         supervisor($x, $middle_manager)))
```)

The general form of a rule is

#syntax("
rule(", meta("conclusion"), ", ", meta("body"), ")
	  ")

where #meta("conclusion") is a pattern and
#meta("body") is any query.#footnote[We will
also allow
#idx("rule (query language)", sub: "without body")
rules without bodies, as in
#py("same"), and we will interpret such a rule to
mean that the rule conclusion is satisfied by any values of the
variables.]

We can think of a rule as representing a large (even
infinite) set of assertions, namely all instantiations of the rule conclusion
with variable assignments that satisfy the rule body. When we described
simple queries (patterns), we said that an assignment to variables satisfies
a pattern if the instantiated pattern is in the data base. But the pattern
needn't be explicitly in the data base as an assertion. It
can be an
#idx("assertion", sub: "implicit")
implicit assertion implied by a rule. For example, the
query

#snippet(```python
lives_near($x, llist("Bitdiddle", "Ben"))
```)

results in

#output(```python
lives_near($x, llist("Bitdiddle", "Ben"))
```)

To find all computer programmers who live near Ben Bitdiddle, we can
ask

#snippet(```python
and(job($x, llist("computer", "programmer")),
    lives_near($x, llist("Bitdiddle", "Ben")))
```)

As in the case of compound
functions,
rules can be used as parts of other rules (as we saw with the
#py("lives_near")
rule above) or even be defined
#idx("recursion", sub: "in rules")
recursively. For instance, the rule
#idx("outrankedby (rule)", decl: true)
#snippet(```python
rule(outranked_by($staff_person, $boss),
     or(supervisor($staff_person, $boss),
        and(supervisor($staff_person, $middle_manager),
            outranked_by($middle_manager, $boss))))
```)

says that a staff person is outranked by a boss in the organization if
the boss is the person's supervisor or (recursively) if the

person's supervisor is outranked by the boss.

#exercise(label-name: <ex:4_55>, [
Define a rule that says that person 1 can replace person 2 if either
person 1 does the same job as person 2 or someone who does person 1's
job can also do person 2's job, and if person 1 and person 2
are not the same person. Using your rule, give queries that find the
following:

+ all people who can replace Cy D. Fect;
+ all people who can replace someone who is being paid more than they are, together with the two salaries.
])

#exercise(label-name: <ex:4_56>, [
Define a rule that says that a person is a "big shot" in a
division if the person works in the division but does not have a supervisor
who works in the division.
])

#exercise(label-name: <ex:4_57>, [
Ben Bitdiddle has missed one meeting too many. Fearing that his habit of
forgetting meetings could cost him his job, Ben decides to do something about
it. He adds all the weekly meetings of the firm to the Gargle data base
by asserting the following:

#snippet(```python
meeting("accounting", llist("Monday", "9am"))
meeting("administration", llist("Monday", "10am"))
meeting("computer", llist("Wednesday", "3pm"))
meeting("administration", llist("Friday", "1pm"))
```)

Each of the above assertions is for a meeting of an entire division.
Ben also adds an entry for the company-wide meeting that spans all the
divisions. All of the company's employees attend this meeting.

#snippet(```python
meeting("whole-company", llist("Wednesday", "4pm"))
```)

+ On Friday morning, Ben wants to query the data base for all the meetings that occur that day. What query should he use?
+ Alyssa P. Hacker is unimpressed. She thinks it would be much more useful to be able to ask for her meetings by specifying her name. So she designs a rule that says that a person's meetings include all #py("\"whole-company\"") meetings plus all meetings of that person's division. Fill in the body of Alyssa's rule. #syntax(" rule(meeting_time(", $mono("$")$, "person, ", $mono("$")$, "day_and_time), ", meta("rule"), "-", meta("body"), ") ")
+ Alyssa arrives at work on Wednesday morning and wonders what meetings she has to attend that day. Having defined the above rule, what query should she make to find this out?
])

#exercise(label-name: <ex:lives-near>, [
By giving the query
#idx("livesnear (rule)")
#snippet(```python
lives_near($person, llist("Hacker", "Alyssa", "P"))
```)

Alyssa P. Hacker is able to find people who live near her, with whom
she can ride to work. On the other hand, when she tries to find all
pairs of people who live near each other by querying

#snippet(```python
lives_near($person_1, $person_2)
```)

she notices that each pair of people who live near each other is
listed twice; for example,

#snippet(```python
lives_near(llist("Hacker", "Alyssa", "P"), llist("Fect", "Cy", "D"))
lives_near(llist("Fect", "Cy", "D"), llist("Hacker", "Alyssa", "P"))
```)

Why does this happen?
Is there a way to find a list of people who live near each other, in
which each pair appears only once? Explain.
])

#subheading([Logic as programs])

#idx("query language", sub: "logical deductions")

We can regard a rule as a kind of logical implication: #emph[If] an
assignment of values to pattern variables satisfies the body, #emph[then] it satisfies the conclusion. Consequently, we can regard the
query language as having the ability to perform #emph[logical deductions] based upon the rules. As an example, consider the
#py("append") operation described at the beginning of
section @sec:logic-programming. As we said,
#py("append") can be characterized by the following
two rules:

- For any list #py("y"), the empty list and #py("y") #py("append") to form #py("y").
- For any #py("u"), #py("v"), #py("y"), and #py("z"), #py("pair(u, v)") and #py("y") #py("append") to form #py("pair(u, z)") if #py("v") and #py("y") #py("append") to form #py("z").

To express this in our query language, we define two rules for a relation

#snippet(```python
append_to_form(x, y, z)
```)

which we can interpret to mean "#py("x") and #py("y") #py("append") to form #py("z")":
#idx("appendtoform (rules)", decl: true)
#snippet(```python
rule(append_to_form(None, $y, $y))

rule(append_to_form(pair($u, $v), $y, pair($u, $z)),
     append_to_form($v, $y, $z))
```)

The first rule has
#idx("rule (query language)", sub: "without body")
no body, which means that the conclusion holds for
any value of
#py("$y").
Note how the second rule makes use of
#py("pair") to name the
head
and
tail
of a list.

Given these two rules, we can formulate queries that compute the
#py("append") of two lists:

#prompt(```python
Query input:
```)

#snippet(```python
append_to_form(llist("a", "b"), llist("c", "d"), $z)
```)

#output(```python
append_to_form(llist("a", "b"), llist("c", "d"), $z)
```)

What is more striking, we can use the same rules to ask the question
"Which list, when #py("append")ed to #py("llist(\"a\", \"b\")"), yields #py("llist(\"a\", \"b\", \"c\", \"d\")")?"
This is done as follows:

#prompt(```python
Query input:
```)

#snippet(```python
append_to_form(llist("a", "b"), $y, llist("a", "b", "c", "d"))
```)

#output(```python
append_to_form(llist("a", "b"), $y, llist("a", "b", "c", "d"))
```)

We can

ask for all pairs of lists that
#py("append") to form
#py("llist(\"a\", \"b\", \"c\", \"d\")"):

#prompt(```python
Query input:
```)

#snippet(```python
append_to_form($x, $y, llist("a", "b", "c", "d"))
```)

#output(```python
append_to_form($x, $y, llist("a", "b", "c", "d"))
```)

The query system may seem to exhibit quite a bit of intelligence in
using the rules to deduce the answers to the queries above. Actually,
as we will see in the next section, the system is following a
well-determined algorithm in unraveling the rules. Unfortunately,
although the system works impressively in the
#py("append") case, the general methods may break down
in more complex cases, as we will see
in section @sec:math-logic.

#exercise(label-name: <ex:next-to>, [
The following rules implement a
#py("next_to_in")
relation that finds adjacent elements of a list:
#idx("nexttoin (rules)", decl: true)
#snippet(```python
rule(next_to_in($x, $y, pair($x, pair($y, $u))))

rule(next_to_in($x, $y, pair($v, $z)),
     next_to_in($x, $y, $z))
```)

What will the response be to the following queries?

#snippet(```python
next_to_in($x, $y, llist(1, llist(2, 3), 4))

next_to_in($x, 1, llist(2, 1, 3, 1))
```)
])

#exercise(label-name: <ex:last-pair-rules>, [
Define rules to implement the
#idx("lastpair", sub: "rules")
#py("last_pair")
operation of exercise @ex:last,
which returns a list
containing the last element of a nonempty list. Check your rules on

the following queries:

- #py("last_pair(llist(3), $x)")
- #py("last_pair(llist(1, 2, 3), $x)")
- #py("last_pair(llist(2, $x), llist(3))")

Do your rules work correctly on queries such as
#py("last_pair($x, llist(3))")?
])

#exercise(label-name: <ex:genesis>, [
The following data base (see Genesis 4) traces the genealogy of the
descendants of
#idx("Ada")#idx("Genesis")
Ada back to Adam, by way of Cain:

#snippet(```python
son("Adam", "Cain")
son("Cain", "Enoch")
son("Enoch", "Irad")
son("Irad", "Mehujael")
son("Mehujael", "Methushael")
son("Methushael", "Lamech")
wife("Lamech", "Ada")
son("Ada", "Jabal")
son("Ada", "Jubal")
```)

Formulate rules such as "If #emph[S] is the son of #emph[F], and #emph[F] is the son of #emph[G], then #emph[S] is the grandson of #emph[G]" and "If #emph[W] is the wife of #emph[M], and #emph[S] is the son of #emph[W], then #emph[S] is the son of #emph[M]" (which was supposedly more true in biblical times than
today) that will enable the query system to find the grandson of Cain; the
sons of Lamech; the grandsons of Methushael.
(See
exercise @ex:great-grandson
for some rules to deduce more complicated relationships.)
])

#idx("rule (query language)")
#idx("query language", sub: "logical deductions")
#idx("query language")
