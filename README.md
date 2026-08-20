# mostafa637

## GTK4 Hello World على Android

يتضمن هذا المستودع تطبيقًا بسيطًا مكتوبًا بلغة **C** باستخدام GTK4، ويعرض نافذة تحتوي على النص `Hello World` على Android. يستخدم المشروع Meson وPixiewood لبناء حزمة Android قابلة للتثبيت، مع توفير مكتبات native للمعماريتين `arm64-v8a` و`x86_64` داخل APK واحد.

### تنزيل APK

الملف الجاهز للتثبيت هو [`gtk-hello-android/gtk-hello-android-release.apk`](gtk-hello-android/gtk-hello-android-release.apk). يمكن تثبيته على أجهزة Android AArch64، كما يمكن اختباره على محاكي Android x86_64. الحزمة موقعة بتوقيع إصدار Android، ومعرّف التطبيق هو `com.example.gtkhelloworld`.

### البناء المحلي

```bash
cd gtk-hello-android
export ANDROID_HOME=/path/to/android-sdk
export JAVA_HOME=/path/to/java-21
/path/to/pixiewood build pixiewood.xml
```

ينتج Pixiewood ملفات ABI منفصلة وملفًا عالميًا باسم `app-universal-release.apk`. يجب اعتماد الملف العالمي عند تحديث `gtk-hello-android-release.apk` في المستودع.

### اختبار GitHub Actions

يُشغّل workflow الموجود في [`.github/workflows/android-emulator.yml`](.github/workflows/android-emulator.yml) الاختبار على محاكي Android API 34 بنواة x86_64. يثبت الـ APK، يشغّل `org.gtk.android.ToplevelActivity`، يحفظ لقطة الشاشة و`logcat`، ثم ينشر آخر ملفات التشخيص في فرع `debug/emulator-latest` عند تشغيل workflow من فرع `main`.

يعالج Runtime Android بدء GTK بعد إنشاء Activity وبعد أول إطار Android، حتى لا يحجب خيط واجهة Android أثناء انتظار تهيئة GTK. هذا مهم لتجنب ظهور System UI ANR أثناء الإقلاع.

### بنية المشروع

| المسار | الغرض |
|---|---|
| `gtk-hello-android/src/main.c` | نقطة الدخول بلغة C وواجهة Hello World |
| `gtk-hello-android/src/android-java/` | Runtime وActivity الخاصة بتشغيل GTK على Android |
| `gtk-hello-android/pixiewood.xml` | اعتماديات Pixiewood والمعماريات المستهدفة |
| `.github/workflows/android-emulator.yml` | بناء/اختبار APK على GitHub Actions |
