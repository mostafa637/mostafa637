# إعادة كتابة iSH بلغة Go وGio

هذا مجلد إعادة كتابة مستقلة لتطبيق iSH بلغة Go. الواجهة تستخدم [Gio](https://gioui.org/) بدل Qt/QML، وفك تعليمات x86-64 يستخدم `golang.org/x/arch/x86/x86asm`. بناء LLVM IR يستخدم [llir/llvm](https://github.com/llir/llvm)، وهي مكتبة مكتوبة بالكامل بـGo للتعامل مع LLVM IR.

## نقطة مهمة حول code generation

`llir/llvm` مكتوبة بالكامل بـGo، لكنها تمثل وتكتب LLVM IR ولا تحتوي backend مستقلًا لتحويل IR إلى machine code. لذلك لا يوجد encoder مخصص في هذا المشروع. أداة `cmd/ish-go-translator` تكتب IR بواسطة `llir/llvm`، ثم تستدعي `llc-18`، وهو backend LLVM الجاهز، لإخراج object code للمضيف. هذا يفصل front-end Go عن backend LLVM، ويتجنب إعادة تنفيذ x86/AArch64 encoder.

## المكونات الحالية

| المجلد | الوظيفة |
|---|---|
| `cmd/ish-go-gui` | نافذة Gio مع terminal UTF-8، محرر IME شفاف، شريط ملحقات iSH، ArrowPad، وصفحات Settings/Files/About. |
| `cmd/ish-go-rootfs` | مستورد `root.tar.gz` إلى `base/data` و`base/meta.db` داخل مجلد بيانات التطبيق الخاص. |
| `cmd/ish-go-translator` | CLI لفك bytes x86-64، بناء IR، واستدعاء `llc` اختياريًا. |
| `internal/session` | جلسة iSH Core/Asbestos حقيقية عبر cgo، مع fallback Linux الاختباري وPTY عند الحاجة. |
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

ينتج workflow APK موقّعًا لكل من `arm64-v8a` و`x86_64`، ويتحقق من توقيع كل APK باستخدام `apksigner`. كما ينزل APK x86_64 إلى AVD يعمل على Linux مع KVM، يثبته، يشغّل Activity، ينتظر جاهزية جلسة Alpine الحقيقية، يتحقق من marker الإصدار، يفتح Settings من زر `infoLight`، يعود إلى الطرفية، ويلتقط screenshots وتشخيصات AVD.

آخر تشغيل Go/Gio ناجح موثق هو [run 33114995383](https://github.com/mostafa637/mostafa637/actions/runs/33114995383) على commit `6f3a4f535ac1cdd3be7c87c1bd00fa0c5f9dd259`؛ ونجح أيضاً run الدفع [33114991706](https://github.com/mostafa637/mostafa637/actions/runs/33114991706) لنفس SHA. أثبتا نجاح Linux، وبناء APKين موقّعين، وتشغيل AVD x86_64 على Linux + KVM، وجاهزية Alpine i386 الحقيقية، وmarker `Alpine release 3.19.0`، ووجود السجلين `Terminal -> Settings` و`Settings -> iSH`، مع سجل crash فارغ. لقطة Settings المرفقة بالـartifact تعرض فعلياً grouped Settings مع Back toolbar وAppearance وExternal Keyboard وFilesystems وUpgrade Repositories وAbout.

بصمات SHA-256 للـartifacts التي رفعها run 33114995383 هي:

| Artifact | SHA-256 |
|---|---|
| `ish-go-arm64-v8a.apk` | `c5ca95ad47e5409282008fae494bf823431503b9446f45debf5820bab0ccf313` |
| `ish-go-x86_64.apk` | `1972c6bf0422ee44d913479caba75cb4dbc7ccc63d886b28d8b044e66b11b7b8` |
| `ish-go-linux-x86_64` | `b181cb0d0582c31ba33fa336d09418a3976e198c526d47f108b0233061d6430e` |

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

هذه نسخة Go/Gio قابلة للبناء تحتوي على terminal UTF-8، جلسة iSH Core/Asbestos حقيقية خلف cgo، فك rootfs بعد التثبيت داخل private app data، شريط ملحقات مستندًا إلى Terminal.storyboard، ArrowPad بسلوك السحب والتكرار المستند إلى `ArrowBarButton.m`، وصفحات Settings/Appearance/External Keyboard/Filesystems/Browse Files/About مع زر Back داخل toolbar. أثبت run 33112614240 أن Alpine i386 يبدأ على Android x86_64 وأن `cat /etc/alpine-release` يعطي `3.19.0`.

صفحات Settings الحالية هي طبقة parity أولية وليست نقلًا وظيفيًا كاملًا لكل شاشات iSH iOS. التنقل والصفوف وBrowse Files تعمل، ونموذج `UserPreferences` يحفظ JSON داخل مجلد التطبيق الخاص ويربط Blink Cursor وCursor Style وTheme وFont Size وColor Scheme بالـrenderer، كما تحفظ صفوف External Keyboard قيمها وتعرضها عند العودة. ما تزال إدارة roots الحقيقية، import/export/delete، تنفيذ Upgrade Repositories، وفتح روابط About بحاجة إلى استكمال؛ لذلك لا تُعد الصفحات نقلًا وظيفيًا كاملاً بعد.

يبقى هدف المشروع الحفاظ على core C الأصلي (iSH/Asbestos/fakefs/SQLite) عبر cgo بدل استبداله بـhost shell على Android. أما مترجم x86asm/LLVM في هذا المجلد فهو أداة تجريبية مستقلة ولا يدّعي ترجمة x86-64 كاملة.
