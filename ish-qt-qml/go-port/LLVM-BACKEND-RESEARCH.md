# LLVM backend وواجهة الترجمة

## القرار النهائي

لن يُكتب مولّد كود آلة مخصص داخل المشروع. تستخدم طبقة Go مكتبة `github.com/llir/llvm` لبناء LLVM IR وكتابته، ثم تستدعي أداة LLVM الجاهزة `llc-18` لتحويل ذلك الـIR إلى object code للمضيف. وبذلك يبقى توليد كود الآلة مسؤولية backend LLVM الرسمي، لا مسؤولية encoder جديد داخل هذا المشروع.

يُستخدم `golang.org/x/arch/x86/x86asm` لفك تعليمات x86-64 إلى تمثيل Go (`x86asm.Inst` و`Args`) قبل مرحلة lowering المحدودة إلى LLVM IR.

## مكوّنات المسار

| المرحلة | التنفيذ | الحدود الحالية |
|---|---|---|
| فك x86-64 | `golang.org/x/arch/x86/x86asm` | decoder؛ لا ينشئ LLVM IR ولا كود آلة تلقائيًا. |
| بناء LLVM IR | `github.com/llir/llvm` (`llir/llvm`) | مكتبة Pure Go لتمثيل وكتابة LLVM IR؛ ليست backend ولا تتضمن TargetMachine مستقلًا. |
| توليد object/machine code | `llc-18` من LLVM | backend جاهز خارجي يُستدعى من CLI؛ لا يوجد code generator يدوي في Go. |

## معنى Pure Go في هذا التصميم

`llir/llvm` مكتوبة بالكامل بلغة Go ولا تحتاج إلى cgo لبناء LLVM IR. أما `llc-18` فهو برنامج LLVM مستقل يُثبت في بيئة البناء ويُستدعى كعملية خارجية. لذلك فإن طبقة الترجمة التي يملكها المشروع Pure Go، بينما يعتمد إخراج كود الآلة على تثبيت LLVM backend الجاهز، وهو المقصود بمتطلب استخدام مكتبة Go مع backend جاهز بدل إعادة تنفيذ code generation.

هذا التصميم مختلف عن استخدام `tinygo.org/x/go-llvm`: ذلك المشروع عبارة عن bindings إلى LLVM النظامي ويتطلب عادةً cgo ومكتبات تطوير LLVM، ولم يعد جزءًا من التنفيذ الحالي. لا ينبغي توثيق TargetMachine أو ORC/MCJIT عبر go-llvm كأنها مستخدمة هنا.

## نطاق lowering الحالي

لا يدّعي `x86asm` أو هذا المشروع توافقًا كاملًا مع x86-64. طبقة `internal/ir` تقبل مجموعة صغيرة ومعلنة وقابلة للاختبار: `NOP` و`PAUSE`، و`MOV RAX, imm`، و`ADD/SUB/XOR RAX, imm`، و`RET`. تُرفض التعليمات الأخرى بخطأ صريح بدل إنتاج IR غير صحيح أو الإيحاء بدعم كامل.

تُختبر العينة التالية في CI:

```sh
go run ./cmd/ish-go-translator \
  -hex 48c7c0010000004883c002c3 \
  -llc llc-18 \
  -o /tmp/ish-go-sample.o
```

وتحوّل تقريبًا:

```asm
mov rax, 1
add rax, 2
ret
```

## Gio وAndroid CI

واجهة `cmd/ish-go-gui` مستقلة عن Qt/QML وتستخدم Gio. تُغلّف Android عبر `gogio`، بينما يثبت workflow حزم Android SDK/NDK ويوقّع APK الناتج ثم يتحقق من التوقيع. يختبر CI Linux، وينتج APK لـ`arm64-v8a` و`x86_64`، ويشغّل APK x86_64 على AVD Linux مع KVM ويلتقط لقطة شاشة.

نجح التشغيل المثبت في commit `3944fd26f5e4c5efca197fc71dc9e2df4eeb989b` ضمن run [32841449796](https://github.com/mostafa637/mostafa637/actions/runs/32841449796)، بما في ذلك وظائف Linux وAndroid للمعماريتين واختبار AVD.

## مراجع

1. [llir/llvm — LLVM IR in Go](https://github.com/llir/llvm)
2. [x86asm package documentation](https://pkg.go.dev/golang.org/x/arch/x86/x86asm)
3. [LLVM llc command documentation](https://llvm.org/docs/CommandGuide/llc.html)
4. [Gio Android installation](https://gioui.org/doc/install/android)
5. [gogio command documentation](https://pkg.go.dev/gioui.org/cmd/gogio)
6. [Gio application package](https://pkg.go.dev/gioui.org/app)
7. [tinygo-org/go-llvm — bindings reference only](https://github.com/tinygo-org/go-llvm)
