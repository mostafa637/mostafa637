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

تم التحقق من التشغيل الفعلي عبر GitHub Actions في التشغيل `32902130382` للالتزام `b5c5a3c0c8e00ccb7b8ab43d5c423a713ffc2760` على فرع `ish-qt-qml`. نجحت اختبارات Linux وLLVM/Gio، وبناءا Android الموقّعان للمعماريتين `arm64-v8a` و`x86_64`، واختبار Android x86_64 AVD باستخدام Linux وKVM.

أثبت اختبار AVD أن `GioActivity` بقيت في حالة `RESUMED` ومركّزة، وأن rootfs فُك داخل مجلد التطبيق الخاص `/data/user/0/org.ish.go/files/ish-rootfs`. سجّل iSH Core نجاح `after-main` و`after-devices`، ثم ظهر داخل الطرفية ناتج `/etc/alpine-release` بالقيمة `3.19.0`، كما سُجّلت الرسالة `iSH Alpine smoke marker received; Alpine release 3.19.0` في logcat. لم يسجّل artifact الخاص بالـAVD أي crash.

هذه النتيجة تثبت تشغيل Alpine i386 الحقيقي عبر محرك iSH/Asbestos المضمّن خلف طبقة cgo، وليست تشغيل `/system/bin/sh` الخاص بالمضيف. كما أن gritty مستخدم لمعالجة parser/buffer الخاصة بالطرفية، وليس بديلًا عن محرك Alpine.

لإعادة التحقق من التوقيع، نفّذ workflow خطوة `apksigner verify --verbose` ونجحت للـAPKين؛ التوقيع الموثق هو APK Signature Scheme v3 بموقّع واحد. بصمات SHA-256 للـartifacts في ذلك التشغيل هي:

| ABI | SHA-256 |
| --- | --- |
| `arm64-v8a` | `f143ac2fc2f851f411198e63f6c3299da8368264ce2662711cbe3a6bea4780b6` |
| `x86_64` | `0f65880e80271c40772330e76d43d6a1ed620d5e7334b6b2501c224a666bdbc1` |
