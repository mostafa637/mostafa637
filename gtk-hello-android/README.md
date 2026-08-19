# GTK Hello World for Android

هذا مشروع صغير بلغة **C** يستخدم **GTK4** لعرض نافذة تحتوي على النص `Hello World`. أُعدّ المشروع بحيث يمكن اختباره على Linux، ثم تحويله إلى حزمة Android باستخدام GTK Android Builder المعروف باسم **Pixiewood**.

## الملفات

| الملف | الغرض |
|---|---|
| `src/main.c` | كود التطبيق بلغة C وإنشاء نافذة GTK والنص المعروض. |
| `meson.build` | تعريف مشروع Meson وربط GTK4 وتحديد نوع هدف Android. |
| `pixiewood.xml` | وصف بناء حزمة Android واعتمادياتها ومعمارياتها. |
| `data/com.example.GtkHelloWorld.metainfo.xml` | بيانات التطبيق بصيغة AppStream. |
| `data/icon.svg` | أيقونة SVG بسيطة للتطبيق. |

## اختبار سريع على Linux

تحتاج إلى تثبيت GTK4 وMeson وNinja أولًا. بعد ذلك نفّذ:

```bash
meson setup build
meson compile -C build
./build/gtk-hello-android
```

## بناء APK على Android

يتطلب البناء وجود GTK Android Builder/Pixiewood وبيئة Android SDK/NDK. من جذر المشروع نفّذ التسلسل التالي:

```bash
pixiewood prepare pixiewood.xml
pixiewood generate
pixiewood build
```

قد تحتاج إلى تمرير مسارات Meson أو Android SDK/NDK وفق إعداد بيئتك، ويمكن عرض الخيارات المتاحة عبر:

```bash
pixiewood --help
```

## ملاحظات

يستخدم البرنامج GTK4 وواجهة `GtkApplication`. يتطلب هدف Meson الخيار `android_exe_type: 'application'`، ولذلك يوصى باستخدام Meson 1.9 أو أحدث عند البناء لأندرويد. دعم GTK على Android ما يزال مسارًا متقدمًا/تجريبيًا مقارنة بتطبيقات Android الأصلية، لذا قد تختلف خطوات الإعداد باختلاف نسخة GTK وPixiewood وAndroid NDK.

## الترخيص

يمكنك تعديل الترخيص في ملف AppStream وفي ملفات المشروع بما يناسبك. المثال الحالي يضع قيمة MIT لبيانات المشروع.
