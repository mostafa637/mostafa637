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
