# CI findings — 2026-08-15

## External references

1. Qt 6.11.1 `qmllint` documentation: https://doc.qt.io/qt-6/qtqml-tooling-qmllint.html

The documentation states that `qmllint` reports QML syntax and anti-pattern warnings, supports warning severity configuration through `.qmllint.ini` and command-line category options, and uses `MaxWarnings` to control the warning threshold. The project CI therefore keeps `--max-warnings 1000000` and disables only the `import` category while the `IshQt` module metadata is generated later by CMake.

2. Official iSH repository: https://github.com/ish-app/ish

The current official checkout used for reference is commit `7864dd60`. It contains the original Objective-C/Objective-C++ application sources under `app/`, including `Terminal.m`, `TerminalViewController.m`, `LinuxRoot.c`, and root/theme/settings code. It does not contain files named `app/core/CoreSession.c`, `CoreClipboard.c`, `CoreLocation.c`, or `deps/sqlite/sqlite3.c`; those names in the current Qt CMake/check script are custom migration paths and need to be restored or replaced with the actual source layout.

## CI observations

The latest workflow run after QML syntax fixes had successful QML lint, while Linux Desktop and both Android ABI jobs failed at `ci/check-source-completeness.sh` because the custom Qt bridge and the four custom core paths are absent. The AVD job was skipped because both Android build jobs failed.

## iSH core build reference

The official Xcode project maps the Linux bridge target `libiSHLinux` to `app/LinuxInterop.c`, `app/PasteboardDeviceLinux.c`, `app/fakefs.c`, `app/LinuxRoot.c`, `app/LinuxTTY.c`, and `app/LinuxPTY.c`. The main `libish` target contains the emulator, fake filesystem, kernel, platform, and utility C sources; `libiSHLinuxUser` contains `linux/emu_asbestos.c`. The current Qt CMake references a custom `app/core` layout and `android/app/src/main/cpp`, neither of which exists in the checked-in snapshot. The correct migration must either create a CMake equivalent of these original targets or clearly document a temporary compatibility layer; it must not compile UIKit/Objective-C sources into Qt Android.

## Runtime assets and SQLite sources

The iSH Xcode configuration defines the official root filesystem source as `https://github.com/ish-app/roots/releases/download/g00712ff0a54b2839c5aa1a8ed758003ca65357dc/appstore-apk.tar.gz`; this archive was downloaded into `android-qt/assets/rootfs/root.tar.gz` and identified as gzip data (3,139,804 bytes).

The official SQLite download page at `https://sqlite.org/download.html` lists SQLite amalgamation 3.53.4 as `https://sqlite.org/2026/sqlite-amalgamation-3530400.zip`. The downloaded ZIP passed `unzip -t`; `sqlite3.c` and `sqlite3.h` were copied to `upstream/ish-ios/deps/sqlite/` (the local SHA-256 values are recorded by the working tree). SQLite is public-domain and is being vendored to avoid Android system-library assumptions.

The iSH Meson source currently supports `platform/darwin.c` and `platform/linux.c` only; there is no `platform/android.c`. The Qt port therefore needs an Android-compatible platform source or a neutral platform implementation while retaining the iSH kernel/Asbestos sources.

## Rootfs/fakefs contract
المصدر المحلي المرجعي: `upstream/ish-ios/tools/fakefs.c` و`fakefs.h` و`fs/fake.c`. صيغة rootfs القابلة للتركيب هي مجلد يحوي `meta.db` و`data/`؛ `fs/fake.c` يثبت اسم `data` ثم يبحث عن `meta.db` بجواره. `tools/fakefs.c` يوفر `fakefs_import()` مع callback للتقدم، وهو المسار الصحيح لتحويل `root.tar.gz` إلى fakefs؛ لا يكفي استخراج tar مباشرة إلى مجلد الجذر. الأرشيف المضمّن الحالي `android-qt/assets/rootfs/root.tar.gz` هو tar عادي يحوي `/bin`, `/etc`, `/ish` وغيرها، لذلك يجب ربط `tools/fakefs.c` و`fakefs_import` أو توفير مسار import مكافئ داخل RootfsManager.

## CMake/core progress
نجح بناء Meson محليًا لـ`kernel=ish`, `engine=asbestos` بعد إضافة SQLite amalgamation bundled (`libsqlite3.a`) ونسخة `platform/android.c` من Linux platform. أضيفت أداة `tools/generate-offsets.py` لتوليد `cpu-offsets.h` من compiler assembly، وأعيدت كتابة `android-qt/CMakeLists.txt` لبناء VDSO وAsbestos وkernel وfakefs وCoreSession مباشرة على Linux/Android.

## CI run 31897543109 بعد commit 935b014

نجح `QML lint (Qt 6.11.1)`. نجح configure Linux وبدأ البناء؛ فشل هدف VDSO لأن CMake مرّر linker script مع اقتباس حرفي (`-Wl,-T,\".../vdso.lds\"`) رغم أن الملف موجود. الإصلاح المطلوب هو تمرير `-Wl,-T,${ISH_SOURCE_DIR}/vdso/vdso.lds` بلا اقتباس داخلي.

فشلت وظائف Android أثناء `find_package(Qt6)` لأن `CMAKE_PREFIX_PATH` يشير إلى Qt Android فقط، بينما Qt6Config.cmake موجود في Qt host tools. يجب توفير مساري Qt host وQt Android معًا، مثل إضافة `QT_HOST_ROOT` و`QT_ANDROID_ROOT` إلى `CMAKE_PREFIX_PATH` أو ضبط `Qt6_DIR`/`CMAKE_FIND_ROOT_PATH_MODE_PACKAGE` بما يسمح بتحميل Config من host ثم مكونات Android. أصبح مسار NDK الصريح تحت `$ANDROID_SDK_ROOT/ndk/27.2.12479018` صحيحًا.

بعد نجاح configure Linux، ظهر خطأ C++ في Qt 6.11 داخل `WebSocketTransport.h`: استخدام `QPointer<QWebSocket>` مع forward declaration فقط منع `static_cast<QObject*, QWebSocket*>` أثناء moc؛ الإصلاح هو تضمين `<QWebSocket>` في header.

## Qt CMake guidance

توثيق Qt 6.11 الرسمي يوصي عند البناء المتقاطع باستخدام toolchain الخاص بالمنصة بدل تمرير مسارات `Qt6_ROOT` أو `CMAKE_PREFIX_PATH` يدويًا. المصدر: https://doc.qt.io/qt-6/cmake-making-qt-available.html . لذلك يستخدم workflow الآن `android_x86_64/lib/cmake/Qt6/qt.toolchain.cmake` أو نظيره لـarm64، مع `QT_HOST_PATH` لأدوات Qt المضيفة.

## CI runs 31898684107, 31899082453, and 31899358661

بعد جعل `CoreSession.c` و`tty-real.c` متوافقين مع Android bionic، نجح Android `x86_64` بينما احتاج arm64 إلى توسيع فروع Asbestos الشرطية التي تجاوزت مدى `R_AARCH64_CONDBR19`. استبدلت الفروع البعيدة في `entry.S` و`memory.S` بتفرع شرطي محلي يتبعه `b` غير مشروط؛ في run `31899358661` نجح `arm64-v8a` و`x86_64` وLinux وQML lint.

ظل اختبار AVD على macOS في خطوة بدء المحاكي دون الانتقال إلى `adb` أو تثبيت APK، فأُلغي التشغيل بعد التحقق من عدم وجود سجل حي مفيد. عُدّل workflow ليبدأ `adb` صراحة، يفحص خروج emulator، يفرض حدود انتظار محددة، يطبع `emulator.log` و`adb devices` وخصائص النظام عند timeout، ويستخدم `-wipe-data` لتفادي حالة AVD قديمة.

## TaskTree note

يظهر أثناء Configure تحذير `Could NOT find Qt6TaskTree`. لم يمنع ذلك بناء Linux أو Android، ويبدو مرتبطًا باكتشاف مكوّن اختياري ضمن Qt 6.11.1 وليس بمكوّن Qt مطلوب مباشرة من التطبيق؛ سيعاد تقييمه فقط إذا تحول إلى خطأ في دورة لاحقة.

## CI run 31900560318 — AVD architecture mismatch

بعد تشغيل emulator من مجلد SDK، اختفى خطأ مكتبة Qt النسبية، لكن صورة `google_apis;arm64-v8a` فشلت على runner macOS الحالي برسالة `HVF error: HV_UNSUPPORTED` و`qemu-system-aarch64-headless: failed to initialize HVF`. توثيق Android يوضح أن تسريع VM يتطلب تطابق معمارية host وصورة النظام، كما يوضح سجل GitHub issue مماثل أن هذا الخطأ يحدث مع arm64 AVD على macOS غير مناسب.

أصبح اختبار AVD يكتشف `uname -m`، ويستخدم `x86_64` وartifact `ish-qt-android-x86_64` على host x64، أو `arm64-v8a` وartifact arm64 على host arm64. كما استُبدل خيار الرسوم deprecated `swiftshader_indirect` بـ`swiftshader`.

## CI run 31901073121 — hosted macOS has no usable HVF for this smoke test

اختيار صورة arm64 المطابقة للـrunner لم يحل المشكلة: run `31901073121` انتهى عند `HVF error: HV_UNSUPPORTED` و`qemu-system-aarch64-headless: failed to initialize HVF`. توثيق Android يذكر أن `-no-accel` مخصص لصور x86/x86_64، وأن عدم وجود hypervisor يفرض ترجمة برمجية بطيئة. لذلك يستخدم smoke test الآن APK `x86_64` وصورة `google_apis;x86_64` مع `-no-accel` ووقت إقلاع أطول، بينما يستمر بناء arm64-v8a وإنتاج APK الخاص به مستقلًا.

## CI run 31901568875 — x86_64 image also requires host match

بعد تحويل smoke test إلى x86_64، أثبت run `31901568875` أن runner هو `arm64` وأن emulator يرفض الصورة صراحةً: `Avd's CPU Architecture 'x86_64' is not supported by the QEMU2 emulator on aarch64 host. System image must match the host architecture.` لذلك عاد الاختبار إلى صورة host المطابقة، مع تمرير `-qemu -accel tcg,thread=multi` لمحاولة إجبار QEMU على ترجمة برمجية بدل HVF، ورفع مهلة الإقلاع لأن TCG أبطأ.

## CI run 31902052669 — TCG is not available for Android arm64 emulator on hosted arm64 macOS

حاول run `31902052669` تمرير `-qemu -accel tcg,thread=multi` إلى صورة `arm64-v8a`، لكنه انتهى بـ`HVF fatal error` داخل `hvf_init_vcpu` قبل الإقلاع. هذا يثبت أن نسخة Android Emulator الحالية على runner arm64 لا توفر مسار software arm64 قابلًا للاستخدام هنا.

وفق مراجع GitHub، `macos-latest` هو arm64 مع منع nested virtualization، بينما `macos-15-intel` هو runner Intel الرسمي المتاح لاختبار x86_64. نُقل AVD smoke test إلى `macos-15-intel`، حيث يستخدم صورة `google_apis;x86_64` وAPK x86_64 مع `-no-accel`; ويظل APK arm64-v8a مبنيًا ومرفوعًا كartifact مستقل.

## Android Emulator Runner on Linux + KVM

أضيف job مستقلًا باسم `Android AVD smoke test (Linux + KVM)` باستخدام `reactivecircus/android-emulator-runner@v2`. قبل تشغيل الإجراء تُضبط صلاحيات `/dev/kvm` عبر قاعدة udev الرسمية، ثم يُختبر APK `x86_64` على صورة API 35 `google_apis` مع profile `pixel_6`. هذا المسار يستخدم hardware acceleration على Ubuntu، بينما يبقى job macOS Intel منفصلًا لاختبار software AVD.

يُحمّل job ملفات APK الناتجة من `build-android` ولا يعيد بناء المشروع. بعد التثبيت يشغّل الحزمة `com.mostafa637.ishqt`، يلتقط screenshot وlogcat، ويفشل عند أخطاء startup مثل `FATAL EXCEPTION` أو `UnsatisfiedLinkError` أو فشل `QQmlApplicationEngine`.

## Linux AVD Runner shell behavior

كشف run `31903956686` أن `reactivecircus/android-emulator-runner@v2` ينفذ قيمة `script` كسلسلة أوامر منفصلة عبر `/usr/bin/sh -c`. لذلك لم تبقَ متغيرات `apk_root` و`apk` بين الأسطر، كما انقسمت كتلة `if` وأصبح `set -o pipefail` غير مدعوم. نُقلت أوامر التثبيت والتشغيل إلى `ci/run-android-avd-smoke.sh`، ويستدعيها workflow في سطر واحد: `script: bash ci/run-android-avd-smoke.sh`. سجل التشغيل نفسه أثبت أن KVM يعمل وأن AVD x86_64 يقلع في نحو 39 ثانية؛ الفشل كان في script الاختبار فقط، لا في المحاكي أو KVM.

## Linux AVD APK signing

نجح run `31904503252` في إقلاع AVD x86_64 عبر KVM خلال نحو 37 ثانية، ثم فشل التثبيت فقط لأن artifact المختار كان `android-build-release-unsigned.apk` ورسالة Android كانت `INSTALL_PARSE_FAILED_NO_CERTIFICATES`. أُضيف إلى `ci/run-android-avd-smoke.sh` اختيار APK موقّع أولًا، مع fallback يقوم بإنشاء debug keystore مؤقت، ثم `zipalign` و`apksigner sign/verify` قبل `adb install`. هذا التوقيع خاص باختبار CI ولا يغيّر APK release المنشور أو مفاتيح التوزيع.

## إزالة اختبار macOS AVD

بناءً على طلب المستخدم، أُزيل job `Android AVD smoke test (macOS)` من `.github/workflows/build-qt.yml`. يبقى اختبار AVD على Linux + KVM فقط؛ فهو المسار الذي أثبت إقلاع Android x86_64 وتثبيت/تشغيل APK بنجاح. تستمر وظيفتا بناء APK للمعماريتين `arm64-v8a` و`x86_64`، بينما تُحفظ ملاحظات macOS السابقة كسجل تاريخي لا كاختبار نشط.
