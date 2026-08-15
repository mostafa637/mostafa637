# ملاحظات مرجع iSH iOS وGitHub

## المصادر

- مستودع النقل الذي أرسله المستخدم: https://github.com/mostafa637/mostafa637/tree/ish-qt-qml
- مستودع iSH iOS الأصلي: https://github.com/ish-app/ish

## نتائج المطابقة

1. فرع `ish-qt-qml` يحتوي على شجرة `upstream/ish-ios` كاملة نسبيًا، وطبقة `android-qt` التي تضم Qt/QML وCMake. اسم المجلد تاريخي؛ ملف البناء يتضمن هدف Linux Desktop إلى جانب أهداف Android، لذلك لا ينبغي اعتبار النقل Android-only.

2. الأيقونات الأصلية موجودة في المرجع تحت `upstream/ish-ios/app/Icons/`، وتشمل `icon.png` و`icon1337.png` ومجموعة الأيقونات البديلة، كما توجد AppIcon PNGs تحت `upstream/ish-ios/app/Assets.xcassets/AppIcon.appiconset/`. قبل الإصلاح لم يكن في `android-qt/assets` مجلد `ui/icons`، مع أن `Main.qml` كان يطلب `qrc:/ish-assets/ui/icons/<name>-light.svg` أو `-dark.svg`.

3. أصول واجهة iSH الأصلية هي ملفات PDF في `Assets.xcassets`: `X.imageset/xmark.circle.fill.regular.large.pdf`، و`Paste.imageset/Paste.pdf`، و`Hide Keyboard.imageset/Hide Keyboard.pdf`، و`Checkbox.imageset/checkbox.pdf`. تم تحويلها إلى SVG مع الحفاظ على المسارات الأصلية، وإنشاء نسخ فاتحة وداكنة.

4. عقد rootfs في iSH الأصلي ليست مجلدًا عاديًا فقط؛ `tools/fakefs.c` ينشئ مجلدًا يحوي `meta.db` و`data/`، ويخزن metadata في SQLite وفق schema: `meta`, `stats`, `paths`, مع `PRAGMA user_version=3`. المسارات في المرجع تُطبع بصيغة تبدأ بـ`/` داخليًا، بينما `fix_path()` يزيل الشرطة الأولى عند الوصول إلى نظام الملفات الحقيقي.

5. `Roots.m` في iSH الأصلي يستورد إلى مجلد مؤقت عبر `fakefs_import()` ثم ينقله atomically إلى وجهته النهائية. هذا يمنع ترك rootfs ناقصًا عند انقطاع النسخ أو فشل الاستيراد.

6. أرشيف `android-qt/assets/rootfs/root.tar.gz` صالح gzip وtar ويحتوي على مجلدات وملفات وروابط رمزية؛ ليس fakefs جاهزًا. لذلك يجب أن ينتج المستورد Qt/QML `meta.db` و`data/` متوافقين مع `fs/fake.c`، لا مجرد فك ضغط عادي.

## ملاحظات CI

ملف `.github/workflows/build-qt.yml` يشغّل QML lint وLinux Desktop build وAndroid ABI builds، ثم Linux+KVM AVD smoke test. آخر workflow على الفرع نجح في configure وبناء Android وتحقق توقيع APK في إحدى التشغيلات، لكن لا يوجد حتى الآن اختبار runtime مستقل يثبت نجاح rootfs بعد التثبيت. يلزم إضافة تحقق للموارد وصحة rootfs، واختبار تشغيل يلتقط فشل `QQmlApplicationEngine` أو session startup.
