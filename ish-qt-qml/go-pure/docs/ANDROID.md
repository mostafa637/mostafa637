# Android build plan

يستخدم هذا الفرع Gio لإنشاء واجهة Android من مصدر Go واحد. لا توجد ملفات Qt أو Wails أو WebView في هذا المسار.

## البناء

بعد تثبيت Android SDK/NDK وضبط `ANDROID_HOME` و`ANDROID_NDK_HOME`:

```bash
cd go-pure
go install gioui.org/cmd/gogio@v0.8.0
make android
```

ينشئ `gogio` ملف APK ويضيف manifest الخاص بـ Gio. يجب تثبيت `gogio` في PATH قبل تشغيل الهدف `android`.

## طبقة الجلسة

يبدأ النموذج الأولي جلسة host PTY من `internal/platform`. قبل إنتاج APK، يجب إضافة تنفيذ Android خلف `session.Session`. سيكون هذا التنفيذ أحد مسارين واضحين:

1. **iSH core backend:** adapter إلى `CoreSession.c` عبر cgo/NDK، وهو المسار المطلوب لتشغيل rootfs وkernel iSH.
2. **Android PTY backend:** تنفيذ PTY خاص بالمنصة إذا كان التطبيق سيشغل shell Android، وليس iSH rootfs.

لا ينبغي أن تختار الواجهة بين هذه المسارات. يتم الاختيار في `internal/app` عبر factory، بينما تبقى `internal/ui` و`internal/terminal` مستقلتين عن Android.

## lifecycle

يجب أن يكون `Session.Close` آمنًا عند تكراره، وأن تتوقف goroutines قبل تدمير نافذة Gio. يجب إرسال `Resize` عند تغير حجم النافذة، وإيقاف القراءة عند `DestroyEvent`. وهذا الفصل يمنع بقاء goroutine تشير إلى نافذة أو renderer بعد إغلاق Android Activity.
