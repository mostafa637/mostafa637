# نتيجة بناء GTK Hello Android

تم بناء تطبيق GTK4 بلغة C يعرض رسالة `Hello World` داخل نافذة Android.

## الناتج

- الحزمة: `com.example.gtkhelloworld`
- النوع: Debug APK موقّع تلقائيًا للتجربة
- المعمارية المرفقة: `arm64-v8a`
- الحجم التقريبي: 2.8 MB
- التحقق: APK Signature Scheme v2 ناجح

## بيئة البناء

- GTK 4.23.3
- Android SDK Platform 35/36
- Android NDK 27.2.12479018
- Meson
- Gradle 9.3.1
- Pixiewood

## التثبيت

```bash
adb install -r gtk-hello-android-debug.apk
```

## إعادة البناء

بعد تثبيت Android SDK وNDK وPixiewood، تُهيأ ملفات cross-compilation ثم تُنفذ مرحلتا generate وbuild حسب تعليمات Pixiewood. قد تحتاج إلى إنشاء أيقونة Android داخل مجلد الموارد إذا لم يكن Android Studio مثبتًا.

هذا APK مخصص للاختبار، وليس إصدار Release موقّعًا بمفتاح نشر Google Play.
