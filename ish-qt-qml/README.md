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

تم التحقق من التشغيل الفعلي النهائي عبر [GitHub Actions run 32937182527](https://github.com/mostafa637/mostafa637/actions/runs/32937182527) للالتزام `a2c57759544b43a495a4a5cda75d20ab28445e3d` على فرع `ish-qt-qml`. نجحت اختبارات Go وLLVM/Gio وبناء Linux، ونجح بناء APKين موقّعين للمعماريتين `arm64-v8a` و`x86_64`، كما نجح اختبار Android x86_64 AVD باستخدام Linux وKVM. هذا التشغيل يتضمن إصلاح readiness الذي ينتظر بدء rootfs و`ash` الحقيقي قبل إرسال أوامر smoke.

أثبت اختبار AVD أن `GioActivity` كانت في حالة `RESUMED` ومركّزة، وأن rootfs فُك داخل مجلد التطبيق الخاص `/data/user/0/org.ish.go/files/ish-rootfs`. سجّل iSH Core أولاً `iSH Alpine session ready; rootfs and ash started`، ثم ظهر داخل logcat marker `iSH Alpine smoke marker received; Alpine release 3.19.0` بعد إرسال أوامر الاختبار، مع بقاء سجل crash فارغاً. يتضمن artifact الخاص بالـAVD لقطة الشاشة النهائية و`avd-activities.txt` و`avd-logcat.txt` وملفات تشخيص التوقّف.

هذه النتيجة تثبت تشغيل Alpine i386 الحقيقي عبر محرك iSH/Asbestos المضمّن خلف طبقة cgo، وليست تشغيل `/system/bin/sh` الخاص بالمضيف. كما أن gritty مستخدم لمعالجة parser/buffer الخاصة بالطرفية، وليس بديلاً عن محرك Alpine. واجهة Gio الحالية تطابق بنية iSH الأساسية: canvas طرفية داكن، إدخال مباشر إلى PTY، شريط accessory مضغوط بترتيب Tab/Control/Escape/Arrow ثم gear/Paste/Hide Keyboard، ومناطق لمس مستقلة للأسهم. أما صفحات الإعدادات/الملفات المتقدمة في iSH iOS فليست مدّعاة كمطابقة كاملة في هذا المسار.

نفّذ workflow خطوة `apksigner verify --verbose` ونجحت للـAPKين؛ التوقيع الموثق هو APK Signature Scheme v3 بموقّع واحد. بصمات SHA-256 للـartifacts في التشغيل النهائي هي:

| ABI | SHA-256 |
| --- | --- |
| `arm64-v8a` | `83beab4158281709d058283e067b0672bef7e5aa23bcfdb702c0e123f4fbd9e5` |
| `x86_64` | `f9d122e1e1c9afde3530b828717198746fd96b4cc67510cdf692ea9f760e38c4` |
