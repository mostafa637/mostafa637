// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Naming and the Environment], label-name: <sec:naming>)

A critical aspect of a programming language is the means it provides
for using
#idx("naming", sub: "of computational objects")
names to refer to computational objects.
We say that the
#idx("primitive expression", sub: "name of variable")
name identifies a
#idx("variable")
#emph[variable]
whose
#idx("variable", sub: "value of")
#emph[value] is the object.

In Python, we name things with #idx("declaration assignment") #idx("declaration", sub: "of variable") #idx("syntactic forms", sub: "declaration assignment") #emph[declaration assignments].

#snippet(```python
size = 2
```)

causes the interpreter to associate the value 2 with the
name #py("size").#footnote[Python uses the same syntax #meta("name") \= #meta("expression") to reassign the value of a #meta("name") even when the name has already been assigned previously with a declaration assignment. In this and the next chapter, we do not use this option. Chapter @chap:state discusses reassignment.]
Once the name #py("size")
has been associated with the number 2, we can
refer to the value 2 by name:

#snippet(```python
print(size)
```)

#output(```python
print(size)
```)

#snippet(```python
print(5 * size)
```)

#output(```python
print(5 * size)
```)

Here are further examples of the use of
declaration assignments:

#snippet(```python
pi = 3.14159
```)

#snippet(```python
radius = 10
```)

#snippet(```python
print(pi * radius * radius)
```)

#output(```python
print(pi * radius * radius)
```)

#snippet(```python
circumference = 2 * pi * radius
```)

#snippet(```python
print(circumference)
```)

#output(```python
print(circumference)
```)

Declaration #idx("means of abstraction", sub: "declaration assignment as") assignment
is our language's
simplest means of abstraction, for it allows us to use simple names to
refer to the results of compound operations, such as the
#py("circumference") computed above.
In general, computational objects may have very complex
structures, and it would be extremely inconvenient to have to remember
and repeat their details each time we want to use them. Indeed,
complex programs are constructed by building, step by step,
computational objects of increasing complexity. The
interpreter makes this step-by-step program construction particularly
convenient because name-object associations can be created
incrementally in successive interactions. This feature encourages the
#idx("incremental development of programs")
#idx("program", sub: "incremental development of")
incremental development and testing of programs and is largely
responsible for the fact that a
#idx("program", sub: "structure of")
Python
program usually consists of a large
number of relatively simple
functions.

It should be clear that the possibility of associating values with
names and later retrieving them means that the interpreter must
maintain some sort of memory that keeps track of the name-object
pairs. This memory is called the
#idx("environment")
#emph[environment]
(more precisely the
#idx("program environment") #emph[program environment],
since we will see later that a
computation may involve a number of different
environments).#footnote[Chapter @chap:state will show that this notion of
environment is crucial for understanding how the interpreter works.
Chapter @chap:meta will use environments for implementing
interpreters.]
