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

ينتج workflow APK موقّعًا لكل من `arm64-v8a` و`x86_64`، ويتحقق من توقيع كل APK باستخدام `apksigner`. كما ينزل APK x86_64 إلى AVD يعمل على Linux مع KVM، يثبته، يشغّل Activity، ينتظر جاهزية جلسة Alpine الحقيقية، يتحقق من marker الإصدار، يفتح Settings من زر `infoLight`، يعود إلى الطرفية، ويلتقط screenshots وتشخيصات AVD. ويحتوي أيضاً على probe مستقل لـAPK arm64: يثبته داخل صورة Android x86_64 التي توفر Native Bridge، ينتظر readiness، يحقن `echo ALP ALP` و`cat /etc/alpine-release`، ويفحص logcat ولقطة الشاشة.

آخر تشغيل Go/Gio ناجح موثق هو [run 33129833499](https://github.com/mostafa637/mostafa637/actions/runs/33129833499) على commit `82a8bad996ebddd4db89930fc5ed85948e910c7d`. أثبت نجاح Linux، وبناء APKين موقّعين، وتشغيل AVD x86_64 على Linux + KVM، وجاهزية Alpine i386 الحقيقية، وmarker `Alpine release 3.19.0`، ووجود السجلين `Terminal -> Settings` و`Settings -> iSH`، مع سجل crash فارغ. والأهم أن APK arm64 ثُبّت وشُغّل فعلياً داخل AVD Android x86_64 عبر Native Bridge؛ أثبت logcat `phase=after-main result=0` و`phase=after-devices result=0`، ثم نفّذ smoke marker وظهر `3.19.0` في الطرفية، وسجل crash فارغ. هذا يثبت أن مشكلة تشغيل arm64 ليست crash في كود التطبيق. الاختبار ليس AVD arm64 guest أصلياً؛ تشغيل guest arm64 الحقيقي تعذر على runners المتاحة بسبب قيود صورة/حزمة Android Emulator، لذلك يسجل workflow هذا القيد بدلاً من ادعاء نتيجة غير متاحة. لقطة Settings المرفقة بالـartifact تعرض grouped Settings مع Back toolbar وAppearance وExternal Keyboard وFilesystems وUpgrade Repositories وAbout، كما تعرض لقطة arm64 الطرفية Alpine 3.19.0 فعلياً.

بصمات SHA-256 للـartifacts التي رفعها run 33129833499 هي:

| Artifact | SHA-256 |
|---|---|
| `ish-go-arm64-v8a.apk` | `e70e55862814aa1efd8508a231c11daadf5747a486f126e0e034b51042b36c58` |
| `ish-go-x86_64.apk` | `a6c50e5213807560d4dc8c289fd50155c763798154cfdc1ae1c6d45f5ef19833` |
| `ish-go-linux-x86_64` | `f59878354dc28040fee7776d4a2dc707a30e3dfb5f3cf35e20a9517f8e8b15d3` |

## إصلاح AArch64 في Asbestos

كان مسار `asbestos_invalidate_range` يعيد كتابة مؤشرات القفز داخل كتل التعليمات المولّدة ثم يعيد تنفيذ هذه الكتل من دون نشر التعديل إلى instruction cache. هذا غير آمن على مضيف ذي I-cache منفصل مثل AArch64؛ فقد تبقى التعليمات القديمة في cache بعد تعديل `jump_ip`. أضيفت دالة `fiber_code_flush` التي تستدعي [`__builtin___clear_cache`](https://gcc.gnu.org/onlinedocs/gcc/Other-Builtins.html#index-_005f_005f_005fbuiltin_005f_005f_005fclear_005fcache) بعد توليد الكتلة، وبعد استعادة jump في `fiber_block_disconnect`، وبعد إعادة ربط jump في مسار `cpu_step_to_interrupt`. لا يتغير منطق guest x86 أو بنية Asbestos، وعلى x86 لا يضيف هذا التعديل مساراً عملياً لتغيير السلوك.

تم بناء `ish-core-smoke` كـAArch64 ELF static وتشغيله بواسطة [`qemu-aarch64`](https://www.qemu.org/docs/master/user/main.html) مع fakefs/SQLite وAlpine i386 الحقيقيين. نجح smoke الأساسي واختبار الضغط الذي نفّذ 100 دورة `exec` وقراءة/كتابة ملفات، وأظهر كلاهما `3.19.0` وخرجاً `0` دون segmentation fault أو assertion أو QEMU signal. كما أظهر تفكيك binary سبعة استدعاءات فعلية إلى `__clear_cache` من مسارات Asbestos.

أضيف أيضاً إصلاح دورة حياة `jetsam` من PR #2، لكن بصيغة آمنة: block الذي دخل `jetsam` عبر `asbestos_invalidate_range` يكون قد مرّ بالفعل عبر `fiber_block_disconnect`، لذلك يمنع `fiber_free_jetsam` إعادة فصل block نفسه مرتين. هذا يحافظ على إزالة المراجع قبل التحرير من دون تكرار `list_remove` أو إنقاص counters مرتين.

نجح [GitHub Actions run 33160964142](https://github.com/mostafa637/mostafa637/actions/runs/33160964142) بعد الدمج الآمن: اختبارات Go و`go vet`، بناء Linux، بناء APK arm64 وx86_64 وتوقيعهما، واختبار AVD x86_64 مع probe لـAPK arm64. كما نجح تشغيل C Core arm64 بعد هذا التعديل على QEMU مع Alpine `3.19.0` والخروج `0`. هذا لا يساوي اختبار AVD arm64 guest أصلي؛ ذلك المسار ما زال يتخطى بوضوح عند عدم توفر Android Emulator/SDK arm64 على runner.

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

هذه نسخة Go/Gio قابلة للبناء تحتوي على terminal UTF-8، جلسة iSH Core/Asbestos حقيقية خلف cgo، فك rootfs بعد التثبيت داخل private app data، شريط ملحقات مستندًا إلى Terminal.storyboard، ArrowPad بسلوك السحب والتكرار المستند إلى `ArrowBarButton.m`، وصفحات Settings/Appearance/External Keyboard/Filesystems/Browse Files/About مع زر Back داخل toolbar. أثبت run 33129833499 أن Alpine i386 يبدأ على Android x86_64 وأن `cat /etc/alpine-release` يعطي `3.19.0`، كما أثبت تشغيل APK arm64 عبر Native Bridge في المحاكي نفسه.

صفحات Settings الحالية هي طبقة parity أولية وليست نقلًا وظيفيًا كاملًا لكل شاشات iSH iOS. التنقل والصفوف وBrowse Files تعمل، ونموذج `UserPreferences` يحفظ JSON داخل مجلد التطبيق الخاص ويربط Blink Cursor وCursor Style وTheme وFont Size وColor Scheme بالـrenderer، كما تحفظ صفوف External Keyboard قيمها وتعرضها عند العودة. ما تزال إدارة roots الحقيقية، import/export/delete، تنفيذ Upgrade Repositories، وفتح روابط About بحاجة إلى استكمال؛ لذلك لا تُعد الصفحات نقلًا وظيفيًا كاملاً بعد.

يبقى هدف المشروع الحفاظ على core C الأصلي (iSH/Asbestos/fakefs/SQLite) عبر cgo بدل استبداله بـhost shell على Android. أما مترجم x86asm/LLVM في هذا المجلد فهو أداة تجريبية مستقلة ولا يدّعي ترجمة x86-64 كاملة.
