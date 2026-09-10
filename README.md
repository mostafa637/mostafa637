
# SICP بصيغة Typst

تحويل كتاب *Structure and Interpretation of Computer Programs* (نسخة Python) من مصادر XML الخاصة بمشروع [source-academy/sicp](https://github.com/source-academy/sicp) إلى كود Typst منظّم في مجلدات، داخل [`sicp-typst/`](sicp-typst/).

الناتج كتاب من 491 صفحة يضم الفصول الخمسة والمقدمات والمراجع والفهرس الهجائي وقائمة التمارين، ويُبنى بأمر واحد:

```bash
cd sicp-typst
typst compile book.typ
```

تفاصيل البنية والأدوات في [`sicp-typst/README.md`](sicp-typst/README.md).
