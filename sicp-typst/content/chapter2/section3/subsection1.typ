// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Strings])

#anchor(<sec:strings>)

#idx("string(s)")

So far, we have used strings in order to display messages,
using the functions #py("display") and
#py("error") (as for example in
exercise @ex:search-for-primes).
We can form compound data using strings and have
linked lists such as
#idx("Wrigstad, Tobias, daughter of")
#idx("Henz, Martin, children of")

#syntax("
llist(\"a\", \"b\", \"c\", \"d\")
llist(23, 45, 17)
llist(llist(\"Jakob\", 27), llist(\"Lova\", 9), llist(\"Luisa\", 24))
          ")

In order to distinguish strings from names, we surround them
#idx("quotation marks", sub: "double")
#idx("\" (double quote)")
with double quotation marks. For example, the Python expression
#py("z") denotes the value of the
name #py("z"), whereas the Python
expression #py("\"z\"") denotes a string
that consists of a single character, namely the last letter in the
English alphabet in lower case.

Via quotation marks, we can distinguish between strings and names:

#snippet(```python
a = 1
b = 2
```)

#snippet(```python
print(llist(a, b))
```)

#output(```python
print(llist(a, b))
```)

#snippet(```python
print(llist("a", "b"))
```)

#output(```python
print(llist("a", "b"))
```)

#snippet(```python
print(llist("a", b))
```)

#output(```python
print(llist("a", b))
```)

In section @sec:conditionals, we introduced
#py("==") and
#py("!=")
as primitive predicates on numbers.
#idx("equality", sub: "of strings")
#idx("==", sub: "as string comparison operator")

#idx("!=", sub: "as string comparison operator", sort: ";4")

From now
on, we shall allow two
strings as operands of
#py("==") and
#py("!="). The predicate
#py("==")
returns True if and only
if the two strings are the same, and
#py("!=")
returns True if and only
if the two strings are not the same.#footnote[We can consider two strings to be "the same" if they
consist of the same characters in the same order. Such a definition
skirts a deep issue that we are not yet ready to address: the meaning
of "sameness" in a programming language. We will return
to this in chapter @chap:state
(section @sec:costs-of-assignment).]
Using #py("=="), we can implement
a useful function called #py("member").
This takes two arguments: a string and a linked list of strings or
a number and a linked list of numbers.
If the first argument is
not contained in the linked list (i.e., is not
#py("==") to any item in the linked list),
then #py("member") returns
#py("None"). Otherwise, it returns the
sublist of the linked list beginning with the first occurrence of the
string or number:
#idx("member", decl: true)
#snippet(```python
def member(item, x):
    return (None if is_none(x)
            else x if item == head(x)
            else member(item, tail(x)))
```)

For example, the value of

#snippet(```python
print(member("apple", llist("pear", "banana", "prune")))
```)

is #py("None"), whereas the value of

#snippet(```python
print(member("apple", llist("x", "y", "apple", "pear")))
```)

is #py("llist(\"apple\", \"pear\")").

#anchor(<ex:equal->)
Two linked lists are said to be
#idx("equal")
#idx("equality", sub: "of linked lists")
#idx("structural equality")
#idx("equality", sub: "structural")
#idx("equality", sub: "of numbers")
#idx("equality", sub: "of strings")
#idx("linked list", sub: "equality of")

#idx("==", sub: "as general comparison operator")
#emph[equal]
if they contain equal elements arranged in the same order, and
Python's #py("==") operator supports this notion
of #emph[structural equality].
For example,

#snippet(```python
llist("this", "is", "a", "linked", "list") == llist("this", "is", "a", "linked", "list")
```)

is True, but

#snippet(```python
llist("this", "is", "a", "linked", "list") == llist("this", llist("is", "a"), "linked", "list")
```)

#idx("number(s)", sub: "equality of")
#idx("string(s)", sub: "equality of")
is False. To be more precise, Python defines the equality operator
#py("==") recursively in terms of the
basic #py("==") equality of numbers and
strings by
saying that #py("a") and
#py("b") are equal
if they are both strings or
both numbers and they are equal,
or if they are both pairs such
that #py("head(a)") is equal to
#py("head(b)") and
#py("tail(a)") is equal to
#py("tail(b)"). The
#py("member") function above
uses #py("==") and therefore
tests for structural equality.

#exercise(label-name: <ex:2_53>, [
What is the result of evaluating each of the
following expressions, in box notation and linked-list notation?

#snippet(```python
llist("a", "b", "c")
```)

#snippet(```python
llist(llist("george"))
```)

#snippet(```python
tail(llist(llist("x1", "x2"), llist("y1", "y2")))
```)

#snippet(```python
tail(head(llist(llist("x1", "x2"), llist("y1", "y2"))))
```)

#snippet(```python
member("red", llist("blue", "shoes", "yellow", "socks"))
```)

#snippet(```python
member("red", llist("red", "shoes", "blue", "socks"))
```)
])

#exercise([
Implement a function #py("equal")
that behaves exactly like #py("=="),
such that #py("equal") applies
#py("==") only to numbers and strings.
])

#exercise(label-name: <ex:double-quotation>, [
The Python interpreter reads the characters after a double
#idx("\" (double quote)")
quotation mark #py("\"") until it finds
another double quotation mark. All characters between the two are part
of the string, excluding the double quotation marks themselves. But what
if we want a string to contain double quotation marks? For this
purpose, Python also allows
#idx("quotation marks", sub: "single")
#idx("' (single quote)")
#emph[single] quotation marks
to delimit strings, as for example in
#py("'say your name aloud'").
Within singly-quoted strings, we can use double quotation marks, and
vice versa, so
#py("'say \"your name\" aloud'") and
#py("\"say 'your name' aloud\"") are valid
strings that have different characters at positions 4 and 14, if we
start counting at 0. Depending on the font in use, two single
quotation marks might not be easily distinguishable from a double
quotation mark. Can you spot which is which and work out the value of
the following expression?

#snippet(```python
print('"' == "")
```)
])

#idx("string(s)")
