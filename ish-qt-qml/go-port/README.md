# إعادة كتابة iSH بلغة Go وGio

هذا مجلد إعادة كتابة مستقلة لتطبيق iSH بلغة Go. الواجهة تستخدم [Gio](https://gioui.org/) بدل Qt/QML، وفك تعليمات x86-64 يستخدم `golang.org/x/arch/x86/x86asm`. بناء LLVM IR يستخدم [llir/llvm](https://github.com/llir/llvm)، وهي مكتبة مكتوبة بالكامل بـGo للتعامل مع LLVM IR.

## نقطة مهمة حول code generation

`llir/llvm` مكتوبة بالكامل بـGo، لكنها تمثل وتكتب LLVM IR ولا تحتوي backend مستقلًا لتحويل IR إلى machine code. لذلك لا يوجد encoder مخصص في هذا المشروع. أداة `cmd/ish-go-translator` تكتب IR بواسطة `llir/llvm`، ثم تستدعي `llc-18`، وهو backend LLVM الجاهز، لإخراج object code للمضيف. هذا يفصل front-end Go عن backend LLVM، ويتجنب إعادة تنفيذ x86/AArch64 encoder.

## المكونات الحالية

| المجلد | الوظيفة |
|---|---|
| `cmd/ish-go-gui` | نافذة Gio مع جلسة PTY، مخرجات shell، محرر إدخال، وشريط ملحقات iSH. |
| `cmd/ish-go-rootfs` | مستورد `root.tar.gz` إلى `base/data` و`base/meta.db`. |
| `cmd/ish-go-translator` | CLI لفك bytes x86-64، بناء IR، واستدعاء `llc` اختياريًا. |
| `internal/session` | جلسة shell عبر `os/exec` و`github.com/creack/pty/v2`. |
| `internal/rootfs` | مستورد fakefs Pure Go يستخدم `modernc.org/sqlite`، مع root path وmetadata وhardlink. |
| `internal/x86translate` | مرحلة decode باستخدام `x86asm`. |
| `internal/ir` | lowering صريح ومحدود إلى LLVM IR. |

## البناء والاختبار المحلي

```sh
go test ./...
go vet ./...
go build ./cmd/ish-go-gui ./cmd/ish-go-rootfs ./cmd/ish-go-translator
```

لتشغيل واجهة Linux، يجب أن تكون جلسة رسومية متاحة:

```sh
go run ./cmd/ish-go-gui
```

ولتوليد تطبيق Android عبر Gio، يجب تثبيت Android SDK/NDK و`gogio` وفق [توثيق Gio Android](https://gioui.org/doc/install/android)، ثم:

```sh
gogio -target android -arch arm64 -o ish-go-arm64.apk ./cmd/ish-go-gui
gogio -target android -arch amd64 -o ish-go-x86_64.apk ./cmd/ish-go-gui
```

## بناء GitHub Actions

يوجد workflow في `.github/workflows/build-go-gio.yml` يبني ويختبر المشروع على GitHub. يشمل ذلك اختبارات Go و`go vet`، وفحص تنسيق Go، وتجربة lowering من x86-64 إلى LLVM IR ثم إخراج object code عبر `llc-18`، وبناء executable لـLinux.

ينتج workflow APK موقّعًا لكل من `arm64-v8a` و`x86_64`، ويتحقق من توقيع كل APK باستخدام `apksigner`. كما ينزل APK x86_64 إلى AVD يعمل على Linux مع KVM، يثبته، يشغّل Activity، يتحقق من عملية التطبيق، ويلتقط screenshot.

آخر تشغيل ناجح موثق هو [run 32841449796](https://github.com/mostafa637/mostafa637/actions/runs/32841449796) على commit `3944fd26f5e4c5efca197fc71dc9e2df4eeb989b`. Artifacts هذا التشغيل هي executable Linux، وAPK `arm64-v8a`، وAPK `x86_64`، ولقطة شاشة وتشخيصات AVD.

## مستورد rootfs

المستورِد يثبت البنية التي يتوقعها fakefs:

```sh
go run ./cmd/ish-go-rootfs \
  -archive /path/to/root.tar.gz \
  -base /path/to/app-private-data
```

ينشئ `data/` و`meta.db` داخل staging، يسجل الجذر عند غيابه، يحفظ stat blob بطول 16 بايت، ينشئ hardlink فعليًا ويربطه بنفس inode في SQLite، ثم يستبدل `data` و`meta.db` دون حذف مجلد التطبيق الأب. بعد ذلك ينفذ `PRAGMA integrity_check` ويتحقق من وجود الجذر و`/bin/sh`.

## مثال LLVM

```sh
go run ./cmd/ish-go-translator \
  -hex 48c7c0010000004883c002c3 \
  > sample.ll
llc-18 -filetype=obj -o sample.o sample.ll
```

العينة تعني تقريبًا:

```asm
mov rax, 1
add rax, 2
ret
```

الـlowering الحالي يدعم فقط `NOP/PAUSE` و`MOV RAX, imm` و`ADD/SUB/XOR RAX, imm` و`RET`. أي instruction أخرى تُرفض بخطأ صريح. هذا مقصود لمنع إعطاء انطباع زائف بأن `x86asm` وحدها تنفذ ترجمة كاملة لـx86-64.

## الحالة والقيود

هذه نسخة Go/Gio قابلة للبناء تحتوي على واجهة terminal وجلسة PTY ومستورد rootfs ومترجم x86asm إلى LLVM IR. جلسة Gio الحالية تشغل shell المضيف عبر PTY، وليست بعد بديلًا كاملًا لـAsbestos داخل iSH. نقل Asbestos وجميع syscalls وfakefs إلى Go يتطلب مشروعًا مستقلًا واسعًا أو إبقاء core C خلف ABI مؤقتة؛ لا يمكن ادعاء اكتمال port كامل قبل نقل هذه المكونات واختبارها على Linux وAndroid.

كما أن دعم شريط الملحقات في Gio ما زال يحتاج استكمال parity مع iSH الأصلي، ومن ذلك سلوك Control sticky وبعض تفاصيل الإدخال والتخطيط. اختبار AVD الحالي يثبت APK ويشغّل الواجهة ويتحقق من عملية التطبيق، ولا يثبت بعد تشغيل نواة iSH/Asbestos كاملة داخل Android.
