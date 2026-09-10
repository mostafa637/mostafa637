#import "@preview/pyrunner:0.3.0" as py

#let run-and-capture(code-raw) = {
  let text-code = if type(code-raw) == str { code-raw } else { code-raw.text }
  let lines = text-code.split("\n").map(line => "    " + line).join("\n")
  let wrapper = "import sys, io\n_buf = io.StringIO()\nsys.stdout = _buf\ntry:\n" + lines + "\nfinally:\n    sys.stdout = sys.__stdout__\n_buf.getvalue()"
  py.block(raw(wrapper, lang: "python"))
}

#let my-code = ```python
def factorial(n):
    return 1 if n <= 1 else n * factorial(n - 1)

print("factorial(5) =", factorial(5))
for i in range(1, 6):
    print(f"2^{i} =", 2**i)
```

== الكود
#my-code

== الناتج المُحسَب عبر Pyrunner
#let out = run-and-capture(my-code)
#raw(out)
