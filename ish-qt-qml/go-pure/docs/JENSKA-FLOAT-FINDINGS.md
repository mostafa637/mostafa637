# قرار استخدام `github.com/jenska/float`

## الخلاصة

تم استخدام `github.com/jenska/float` في طبقة `internal/core/emu/fpu` فقط، لأنه تنفيذ Pure Go لحسابات IEEE 754 extended double بصيغة 80-bit، وهي الوظيفة التي يحتاجها مسار محاكاة x87 في iSH. لم يُستخدم داخل `internal/core/storage`، ولم يُستخدم بديلًا عن `modernc.org/sqlite`؛ فطبقة fakefs تخزن سجلات inode ثابتة من أربعة حقول `uint32` ولا تحتاج حسابات فاصلة عائمة.

المصدر الرسمي يصف المكتبة بأنها تنفيذ Pure Go لحسابات 80-bit مشتق من SoftFloat، وموجه لمحاكاة Motorola M68881/M68882 FPU [1]. كما أن عقد iSH الأصلي يضع النوع `float80` وعمليات `f80_add` و`f80_mul` وغيرها داخل مجلد `emu`، ما يجعل هذا هو موضع الدمج الصحيح [2].

## الدمج المنفذ

أضيف غلاف محدود في `internal/core/emu/fpu/value.go`. الغلاف يخفي النوع الخارجي `X80` عن بقية core ويوفر التحويلات والعمليات الأساسية، مثل الجمع والطرح والضرب والقسمة والباقي والجذر والمقارنة والتصنيف والتحويل إلى `float64` و`int64`. هذا التصميم يسمح لاحقًا باستكمال عقد `float80.h` تدريجيًا من دون ربط filesystem أو kernel أو Gio بمكتبة الفاصلة العائمة.

أُضيفت اختبارات في `value_test.go` للتحقق من الحساب والتصنيف والمقارنة، وتشمل اختبارًا انحداريًا للعبارة `1/3*3`.

## الإصلاح المحلي الضروري

اختبارًا مباشرًا للإصدار `v1.0.0` كشف أن تنفيذ `Mul` كان يقارن كلمة المنتج unsigned بالعدد صفر (`0 < zSig0`) بدل اختبار بت الإشارة الأعلى. لذلك كان المنتج غير الصفري يُزاح دائمًا تقريبًا، ونتيجة `1/3*3` ظهرت `0.5` بدل `1`. اختبارات المكتبة الأصلية تمر، لكنها لا تغطي هذا السيناريو.

لذلك أُضيفت نسخة محلية من مصدر المكتبة في `third_party/jenska/float`، مع الحفاظ على اسم الوحدة والإصدار الرسميين عبر:

```go
replace github.com/jenska/float => ./third_party/jenska/float
```

والتعديل الوحيد المقصود في مسار الضرب يختبر:

```go
zSig0&0x8000000000000000 == 0
```

بدل المقارنة العددية. لا ينبغي إزالة هذا الإصلاح أو العودة إلى المصدر البعيد قبل رفع إصلاح upstream والتحقق من تكافؤ FPU باختبارات instruction-level.

## النتيجة العملية

| المكوّن | القرار | السبب |
|---|---|---|
| `internal/core/emu/fpu` | يستخدم الغلاف المحلي لـ`jenska/float` | موضعه يطابق طبقة x87/80-bit في iSH |
| `internal/core/storage` | لا يستخدم `jenska/float` | التخزين يعتمد على integers وblob ثابتة |
| `internal/core/storage` | يستخدم `modernc.org/sqlite` | SQLite Pure Go وبدون embedded C أو CGo |
| `internal/ui` و`internal/terminal` | لا يستوردان أيًا منهما | الحفاظ على حدود Gio والطبقات |

هذا الدمج لا يعني أن كامل FPU أو كامل iSH core أصبح منقولًا؛ بل يثبت أول حد Pure Go قابل للاختبار، ويترك استكمال سجلات CPU وتعليمات x87 وعقد `float80.h` للمراحل التالية.

## المراجع

[1]: https://github.com/jenska/float "jenska/float — 80-bit IEEE 754 extended double precision library for Go"
[2]: https://github.com/ish-app/ish "iSH upstream repository and emulator source"
