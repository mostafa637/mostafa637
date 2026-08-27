# iSH Qt Android/Linux

هذا المستودع يحتوي طبقة Qt/QML لتطبيق iSH على Android وLinux. تم تفعيل Git محليًا لحفظ نقاط استعادة متتابعة.

## نقاط الاستعادة

قبل إعادة إنشاء ملفات QML وWebView حُفظ commit باسم:

```text
ef46ed8 chore: save restored baseline before QML and WebView reconstruction
```

بعد اكتمال إعادة الإنشاء يُحفظ commit ثانٍ. يمكن الرجوع إلى الحالة السابقة باستخدام:

```bash
git log --oneline
git restore --source ef46ed8 -- .
```

## الملفات المعاد إنشاؤها

تتضمن إعادة الإنشاء مكونات `IshIOSStyle`، صفحات الإعدادات، `TerminalWeb.qml`، `TerminalScrollBar.qml`، وموارد `term.html` و`term.css` و`term.js` و`qwebchannel.js`. كل صفحة ثانوية تستخدم `IOSToolBar` وبداخله زر `‹ Back` يرسل إشارة `closeRequested` إلى التنقل في `Main.qml`.

## WebView

تقوم `TerminalWeb.qml` بتحميل URL الذي يعيده `RootfsManager.terminalUrl(webChannel.url)`. عند نجاح التحميل ترسل إشارة `ready`، وعند الفشل ترسل رسالة تشخيصية. صفحة HTML تعرض رسالة ابتدائية بدل الظهور فارغة، وتدعم UTF-8 والإدخال الأساسي وWebSocket عند تمرير عنوان `ws`.

## البناء

تحتاج عملية البناء الكاملة إلى ملفات نواة iSH وAsbestos وfakefs ومصادر C++ التي يطلبها `android-qt/CMakeLists.txt`، إضافة إلى Qt 6.11.1 وAndroid SDK/NDK. إذا لم تكن هذه الملفات موجودة في النسخة المستعادة فلن يكفي QML وحده لإنتاج APK جديد.

## تحقق Android وAlpine

تم التحقق من التشغيل الفعلي الأحدث عبر [GitHub Actions run 33040885722](https://github.com/mostafa637/mostafa637/actions/runs/33040885722) للالتزام `ee81d70e5d7056ba5f877fe50030f9e4320a02ed` على فرع `ish-qt-qml`. نجحت اختبارات Go وLLVM/Gio وبناء Linux، ونجح بناء APKين موقّعين للمعماريتين `arm64-v8a` و`x86_64`، كما نجح اختبار Android x86_64 AVD باستخدام Linux وKVM. شمل هذا الإصلاح ربط أصول iSH iOS الأصلية لـPaste وHide Keyboard داخل Gio، وإضافة `paint.PaintOp` اللازمة لرسمها فعلياً، مع إبقاء readiness بعد نجاح تشغيل rootfs و`ash` الحقيقي واستخدام keyevents متسلسلة في اختبار AVD.

أثبت اختبار AVD أن `GioActivity` كانت في حالة `RESUMED` ومركّزة، وأن rootfs فُك داخل مجلد التطبيق الخاص `/data/user/0/org.ish.go/files/ish-rootfs`. سجّل iSH Core `phase=after-main result=0` و`phase=after-devices result=0` ثم `iSH Alpine session ready; rootfs and ash started`. بعد إرسال أوامر الاختبار عبر واجهة Gio ظهر داخل logcat مرتان marker `iSH Alpine smoke marker received; Alpine release 3.19.0`، وبقي `avd-logcat-crash.txt` فارغاً ولم يظهر `FATAL EXCEPTION` أو `ANR` للتطبيق. يتضمن artifact الخاص بالـAVD لقطة الشاشة النهائية و`avd-activities.txt` و`avd-logcat.txt` وملفات تشخيص التوقّف.

هذه النتيجة تثبت تشغيل Alpine i386 الحقيقي عبر محرك iSH/Asbestos المضمّن خلف طبقة cgo، وليست تشغيل `/system/bin/sh` الخاص بالمضيف. كما أن gritty مستخدم لمعالجة parser/buffer الخاصة بالطرفية، وليس بديلاً عن محرك Alpine. واجهة Gio الحالية تطابق بنية iSH الأساسية: canvas طرفية داكن edge-to-edge، إدخال مباشر إلى PTY، شريط accessory مضغوط بترتيب Tab/Control/Escape/Arrow ثم gear/Paste/Hide Keyboard، ومناطق لمس مستقلة للأسهم. ملفات `Paste.pdf` و`Hide Keyboard.pdf` مأخوذة من `upstream/ish-ios/app/Assets.xcassets`، وحُوّلت إلى PNG شفافة مضمّنة مع الحفاظ على الشكل الأصلي؛ أما رموز Tab/Control/Escape/Info النصية فتستخدم glyphs المصدر مع خط مضمن. أما صفحات الإعدادات/الملفات المتقدمة في iSH iOS فليست مدّعاة كمطابقة كاملة في هذا المسار.

نفّذ workflow خطوة `apksigner verify --verbose` ونجحت للـAPKين؛ التوقيع الموثق هو APK Signature Scheme v3 بموقّع واحد. بصمات SHA-256 للـartifacts في التشغيل الأخير هي:

| Artifact | SHA-256 |
| --- | --- |
| `ish-go-arm64-v8a.apk` | `26a8de94990a1956cc4cce702a6af308a18edfe865673daa502d84dd7a131409` |
| `ish-go-x86_64.apk` | `29c45ac743ca1a972ecfeac39fae4bb1082123be7802373400246b1de0ac8cce` |
| `ish-go-linux-x86_64` | `049e63e3997535036d8b50da6dfd0a2ff5fa1be276fed8abd46f69835ea9591b` |
