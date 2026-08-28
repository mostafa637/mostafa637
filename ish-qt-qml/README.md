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

تم التحقق من التشغيل الفعلي الأحدث عبر [GitHub Actions run 33107011021](https://github.com/mostafa637/mostafa637/actions/runs/33107011021) للالتزام `4bcf4d86424f97edb25438f569926840ffb4fff0` على فرع `ish-qt-qml`. نجحت اختبارات Go وLLVM/Gio وبناء Linux، ونجح بناء APKين موقّعين للمعماريتين `arm64-v8a` و`x86_64`، كما نجح اختبار Android x86_64 AVD باستخدام Linux وKVM. يشمل هذا الإصدار أصول iSH iOS الأصلية لـPaste وHide Keyboard، و`paint.PaintOp` اللازمة لرسمها فعلياً، وسلوك ArrowPad الأصلي: عتبة سحب 20dp، إرسال الاتجاه فور تجاوزه، ثم تكرار بعد 500ms كل 100ms مع تخفيف الأسهم غير المختارة.

أثبت اختبار AVD أن `GioActivity` كانت في حالة `RESUMED` ومركّزة، وأن rootfs فُك داخل مجلد التطبيق الخاص `/data/user/0/org.ish.go/files/ish-rootfs`. سجّل iSH Core `phase=after-main result=0` و`phase=after-devices result=0` ثم `iSH Alpine session ready; rootfs and ash started`. بعد إرسال أوامر الاختبار عبر واجهة Gio ظهر داخل logcat مرتان marker `iSH Alpine smoke marker received; Alpine release 3.19.0`، وبقي `avd-logcat-crash.txt` فارغاً ولم يظهر `FATAL EXCEPTION` أو `ANR` للتطبيق. يتضمن artifact الخاص بالـAVD لقطة الشاشة النهائية و`avd-activities.txt` و`avd-logcat.txt` وملفات تشخيص التوقّف.

هذه النتيجة تثبت تشغيل Alpine i386 الحقيقي عبر محرك iSH/Asbestos المضمّن خلف طبقة cgo، وليست تشغيل `/system/bin/sh` الخاص بالمضيف. كما أن gritty مستخدم لمعالجة parser/buffer الخاصة بالطرفية، وليس بديلاً عن محرك Alpine. واجهة Gio الحالية تطابق بنية iSH الأساسية: canvas طرفية داكن edge-to-edge، إدخال مباشر إلى PTY، شريط accessory مضغوط بترتيب Tab/Control/Escape/Arrow ثم infoLight/Paste/Hide Keyboard، ومناطق لمس مستقلة للأسهم. ملفات `Paste.pdf` و`Hide Keyboard.pdf` مأخوذة من `upstream/ish-ios/app/Assets.xcassets`، وحُوّلت إلى PNG شفافة مضمّنة مع الحفاظ على الشكل الأصلي؛ أما رموز Tab/Control/Escape/Info النصية فتستخدم glyphs المصدر مع خط مضمن. ArrowPad يستخدم gesture ثنائي المحور مستلهماً `ArrowBarButton.m`، بما في ذلك السحب، التقاط الاتجاه، الإرسال الفوري، والتكرار. أما صفحات الإعدادات/الملفات المتقدمة في iSH iOS فليست مدّعاة كمطابقة كاملة في هذا المسار.

نفّذ workflow خطوة `apksigner verify --verbose` ونجحت للـAPKين؛ التوقيع الموثق هو APK Signature Scheme v3 بموقّع واحد. بصمات SHA-256 للـartifacts في التشغيل الأخير هي:

| Artifact | SHA-256 |
| --- | --- |
| `ish-go-arm64-v8a.apk` | `976c069819f24b2f055dcb64f01106145c640ef3589b04ec84980d3f8397e79b` |
| `ish-go-x86_64.apk` | `809f478bbe1016a03ee14776cdb40469b8a8f31a9c3b7573bd9973215505c74a` |
| `ish-go-linux-x86_64` | `64dac81f0fd8571fbd53b1796221824c4bb11bf0258783b2f2bd9d870234c46f` |
