# Android diagnostics

تكتب طبقة Qt سجلًا تشخيصيًا عند بدء التطبيق، وعند أخطاء rootfs أو WebSocket أو native session، وعند رسائل Qt، كما تثبّت معالجًا لإشارات الانهيار الشائعة.

المسار الأساسي القابل للكتابة هو:

```text
/sdcard/Android/data/com.mostafa637.ishqt/files/ish-qt-errors.log
```

تحاول النسخة أيضًا الكتابة إلى:

```text
/sdcard/Download/ish-qt-errors.log
```

إذا منع Android الوصول إلى التخزين العام، يبقى المسار الأول هو المسار المتوقع. توجد أيضًا نسخة fallback في مجلد بيانات التطبيق إذا لم يتوفر التخزين الخارجي.

يمكن نسخ السجل باستخدام Android Debug Bridge:

```bash
adb pull /sdcard/Android/data/com.mostafa637.ishqt/files/ish-qt-errors.log
```

يبدأ كل سطر بتوقيت UTC. السجلات المهمة تحمل عناوين مثل `native session` و`web channel` و`rootfs preparation` و`QML session`. عند حدوث `SIGSEGV` أو `SIGABRT` أو إشارة مشابهة، يضيف المعالج سطرًا من نوع `fatal signal` قبل إعادة إرسال الإشارة للنظام.

لا يعتمد إنشاء المسار الأساسي على صلاحية تخزين عامة؛ فهو مجلد التخزين الخارجي الخاص بالتطبيق. قد تخفي بعض تطبيقات إدارة الملفات مجلد `Android/data`، وفي هذه الحالة يكون استخدام `adb pull` هو الطريقة الأكثر موثوقية.

يظهر المسار الفعلي الذي فُتح داخل خاصية QML التالية:

```text
platformServices.diagnosticLogPath
```
