// Test harness for the Calepin-backed interpreter transcripts.
//
//   calepin compile test-calepin.typ   -> executes the transcripts for real
//   typst compile test-calepin.typ     -> compiles; transcripts stay empty
//
// Transcripts are emitted as hidden Calepin chunks that publish their output
// through the store; the output string is typeset by listings.typ.
#import "lib/code.typ": output, snippet, prompt, transcript-source, transcript-options
#import "lib/listings.typ": listings

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

== سطر المُفسِّر التفاعلي

#prompt(```python
>>> print("ok")
```)

== نص المُشغِّل المُرسَل إلى calepin

#raw(transcript-source("print(\"hi\")", "_sicp_tX"), lang: "python", block: true)

== إعدادات listings للمخرجات

#repr(transcript-options)
