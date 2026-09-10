// Test harness for the Calepin-backed interpreter transcripts.
//
//   calepin compile test-calepin.typ   -> executes the transcripts for real
//   typst compile test-calepin.typ     -> compiles; transcripts stay empty
//
#import "lib/code.typ": output, snippet, transcript-source, show-interpreter-outputs

#show: show-interpreter-outputs

#let program = ```python
def factorial(n):
    return 1 if n <= 1 else n * factorial(n - 1)

print("factorial(5) =", factorial(5))
for i in range(1, 6):
    print(f"2^{i} =", 2**i)
```

== الكود

#snippet(program)

== الناتج

#output(program)

== تعبير بلا طباعة (يجب ألا يظهر صندوق)

#output(```python
486
```)

== نص المُشغِّل المُرسَل إلى calepin

#raw(transcript-source("print(\"hi\")"), lang: "python", block: true)
