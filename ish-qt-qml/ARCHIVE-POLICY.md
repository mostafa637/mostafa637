# سياسة أرشفة مشروع iSH Qt/QML

بعد كل تعديل أو commit يُنشأ أرشيف ZIP مستقل باسم يتضمن رقم commit، مثل:

```text
ish-qt-qml-<commit>-<timestamp>.zip
```

يحتوي الأرشيف على المصدر المتتبع في Git، وملف Git bundle حديثًا للاستعادة، وملف `ARCHIVE-INFO.md`، وملف SHA-256. تُستبعد مجلدات البناء المؤقتة وملفات Android الناتجة لتقليل الحجم؛ أما APKs وملفات التوزيع فتُرسل كمرفقات منفصلة عند الحاجة.

لاستعادة سجل Git من الـbundle:

```bash
git clone recovery/ish-qt-history-<commit>.bundle ish-android
cd ish-android
git log --oneline
```

تُحفظ الأرشفة خارج شجرة المصدر حتى لا يدخل ZIP داخل ZIP في التحديث التالي.
