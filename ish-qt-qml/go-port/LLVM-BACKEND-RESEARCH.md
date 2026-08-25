# LLVM backend research

## القرار
لن يُكتب مولّد كود آلة مخصص داخل المشروع. سيُستخدم LLVM backend الجاهز عبر bindings Go، بينما تُستخدم `golang.org/x/arch/x86/x86asm` لفك تعليمات x86_64 فقط.

## المكتبات

| الاستخدام | المكتبة | الملاحظة |
|---|---|---|
| فك تعليمات x86_64 | `golang.org/x/arch/x86/x86asm` | توفر `Decode(src, 64)` وتمثيل `Inst` و`Args` وصيغ Intel/GNU/Go؛ لا تولد كود آلة. |
| LLVM من Go | `tinygo.org/x/go-llvm` | bindings إلى LLVM مثبت على النظام، وتدعم إصدارات LLVM 14–20 بحسب README الحالي، مع build tags مثل `llvm20`. |
| backend الجاهز | LLVM TargetMachine أو ORC/MCJIT من خلال go-llvm | LLVM يتولى تحسين LLVM IR وتوليد object/machine code أو JIT؛ لا نعيد تنفيذ instruction encoder داخل Go. |

## قيود مهمة

`go-llvm` يستخدم cgo ويربط LLVM النظامي، لذلك المقصود بـPure Go هنا هو أن طبقة التطبيق والمترجم مكتوبة بـGo، وليس أن LLVM نفسه أو الربط معه خالٍ من C/C++. يلزم تثبيت LLVM development package مناسب لكل منصة، أو بناء LLVM مخصصًا مع `byollvm` وضبط CFLAGS/LDFLAGS.

`x86asm` decoder لا يترجم تلقائيًا كل تعليمات x86 إلى LLVM IR. يجب أن يقتصر front-end الأول على مجموعة تعليمات مدعومة ومعلنة، أو استخدام decoder/translator جاهز آخر إذا كان الهدف توافقًا كاملًا مع x86_64. أما توليد كود الآلة بعد تكوين IR فسيتم حصريًا بواسطة LLVM TargetMachine/ORC.

## المصادر

1. https://github.com/tinygo-org/go-llvm — Go bindings to system LLVM، وإصدارات LLVM المدعومة وbuild tags.
2. https://pkg.go.dev/golang.org/x/arch/x86/x86asm — توثيق فك تعليمات x86، `Decode`, `Inst`, `Args` والصيغ النصية.
3. https://pkg.go.dev/tinygo.org/x/go-llvm — توثيق API Go لـLLVM، بما فيه Builder وTarget/JIT APIs.

## Gio

Gio هو إطار immediate-mode مكتوبًا بـGo ويدعم Linux وAndroid ومنصات أخرى. دورة التطبيق تعتمد على `app.Window` وقراءة الأحداث من `Window.Event()`، بينما تُحفظ حالة widgets في Go وتُرسم في كل دورة. لذلك ستكون طبقة الواجهة الجديدة Gio مستقلة عن QML/WebView، مع مكوّن terminal يرسم النص والحالة، وشريط ملحقات يستخدم widgets وgestures من Gio.

المصدر الرسمي: https://gioui.org/ — Gio cross-platform immediate-mode GUI.
المصدر الرسمي: https://pkg.go.dev/gioui.org/app — دورة نافذة Gio والأحداث.
المصدر الرسمي: https://gioui.org/doc/architecture/widget — widgets وحالة الإدخال.
