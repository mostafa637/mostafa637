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

تم التحقق من التشغيل الفعلي الأحدث عبر [GitHub Actions run 32963752976](https://github.com/mostafa637/mostafa637/actions/runs/32963752976) للالتزام `50e6f04b740c71ee0fda876d3f9a213530c7848f` على فرع `ish-qt-qml`. نجحت اختبارات Go وLLVM/Gio وبناء Linux، ونجح بناء APKين موقّعين للمعماريتين `arm64-v8a` و`x86_64`، كما نجح اختبار Android x86_64 AVD باستخدام Linux وKVM. شمل هذا الإصلاح readiness بعد نجاح تشغيل rootfs و`ash` الحقيقي، واستخدم اختبار AVD حقن keyevents متسلسلة بدلاً من `adb input text` الذي كان يقطّع الأوامر.

أثبت اختبار AVD أن `GioActivity` كانت في حالة `RESUMED` ومركّزة، وأن rootfs فُك داخل مجلد التطبيق الخاص `/data/user/0/org.ish.go/files/ish-rootfs`. سجّل iSH Core `phase=after-main result=0` و`phase=after-devices result=0` ثم `iSH Alpine session ready; rootfs and ash started`. بعد إرسال أوامر الاختبار عبر واجهة Gio ظهر داخل logcat مرتان marker `iSH Alpine smoke marker received; Alpine release 3.19.0`، وبقي `avd-logcat-crash.txt` فارغاً ولم يظهر `FATAL EXCEPTION` أو `ANR` للتطبيق. يتضمن artifact الخاص بالـAVD لقطة الشاشة النهائية و`avd-activities.txt` و`avd-logcat.txt` وملفات تشخيص التوقّف.

هذه النتيجة تثبت تشغيل Alpine i386 الحقيقي عبر محرك iSH/Asbestos المضمّن خلف طبقة cgo، وليست تشغيل `/system/bin/sh` الخاص بالمضيف. كما أن gritty مستخدم لمعالجة parser/buffer الخاصة بالطرفية، وليس بديلاً عن محرك Alpine. واجهة Gio الحالية تطابق بنية iSH الأساسية: canvas طرفية داكن edge-to-edge، إدخال مباشر إلى PTY، شريط accessory مضغوط بترتيب Tab/Control/Escape/Arrow ثم gear/Paste/Hide Keyboard، ومناطق لمس مستقلة للأسهم. أما صفحات الإعدادات/الملفات المتقدمة في iSH iOS فليست مدّعاة كمطابقة كاملة في هذا المسار.

نفّذ workflow خطوة `apksigner verify --verbose` ونجحت للـAPKين؛ التوقيع الموثق هو APK Signature Scheme v3 بموقّع واحد. بصمات SHA-256 للـartifacts في التشغيل الأخير هي:

| Artifact | SHA-256 |
| --- | --- |
| `ish-go-arm64-v8a.apk` | `bf5708cdc989717fbc5b516232de98862558113e750fa0667942d5b6c9d88f7d` |
| `ish-go-x86_64.apk` | `7ee22015275973459ea6048cb72ee4827326b6745afd0a20e1defd5fb246faa4` |
| `ish-go-linux-x86_64` | `a9d480a1149e8981f324068445c2f8c37f730a2ae1fd6258733f09a8e4e0a678` |
