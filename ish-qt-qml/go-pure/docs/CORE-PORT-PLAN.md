# خطة إعادة كتابة iSH core إلى Go

## النطاق

الهدف هو نقل طبقات iSH الخاصة إلى Go تدريجيًا، مع إبقاء النسخة C قابلة للبناء للمقارنة حتى يثبت البديل الجديد. استُبدلت SQLite المضمّنة بطبقة `modernc.org/sqlite` Pure Go، بينما تُعزل حسابات x87 ذات 80-bit خلف غلاف `jenska/float` في `internal/core/emu/fpu`.

## جرد المصدر الحالي

| الطبقة | الملفات | الأسطر التقريبية | قرار النقل |
|---|---:|---:|---|
| `asbestos` | 25 | 4,484 | نقل تدريجي إلى `internal/core/cpu` بعد تثبيت نموذج الذاكرة والسجلات |
| `emu` | 16 | 4,420 | نقل إلى `internal/core/emu` مع اختبارات instruction-level |
| `kernel` | 45 | 8,656 | تقسيم إلى `internal/core/kernel` و`internal/core/syscall` و`internal/core/signal` |
| `fs` | 49 | 9,522 | تقسيم إلى `internal/core/fs` و`internal/core/tty` و`internal/core/proc` |
| `util` | 11 | 749 | نقل حسب الاعتماد إلى `internal/core/util` |
| `app/core` | 4 | 602 | إعادة كتابة session وboot وPTY في `internal/core/session` |
| SQLite المضمّن | 2 | 283,998 | لا يُنقل؛ يُستبدل بـ`modernc.org/sqlite` خلف `internal/core/storage` |

## حدود الحزم المقترحة

```text
go-pure/
├── cmd/ishgo/
├── internal/core/
│   ├── cpu/          # Asbestos: registers, memory, execution state
│   ├── emu/          # x86 instruction and floating-point support
│   ├── kernel/       # task, syscall, signal, time, memory
│   ├── fs/           # inode, fd, path, proc, real/fake filesystem
│   ├── tty/          # tty, pty, poll and terminal semantics
│   ├── session/      # boot, launch, resolver and lifecycle
│   └── backend.go    # replaceable backend boundary
├── internal/terminal/
├── internal/ui/
└── internal/platform/
```

## ترتيب النقل

يبدأ العمل بمكوّنات لا تعتمد على نظام التشغيل: أنواع السجلات والذاكرة، ثم parser للتعليمات واختبارات المقارنة. بعد ذلك تُنقل طبقة kernel الأساسية والمهام وعمليات الذاكرة، ثم filesystem وfd وtty. في المرحلة الأخيرة تُنقل session وboot وتُربط Gio بها.

كل مرحلة يجب أن تملك اختبارات Go مستقلة، واختبار مقارنة مع النسخة C عندما يكون الناتج قابلًا للرصد. لا يجوز حذف C core قبل أن ينجح shell smoke وrootfs وPTY وPython في النسختين.

## معيار التكافؤ

يُعتبر المكوّن منقولًا عندما يمرّ على الأقل باختبارات: إنشاء الجلسة، prompt `/ #`، الكتابة والقراءة، CR/LF، تغيير حجم tty، Ctrl-C، إيقاف session، وعمليات `apk add python3` و`python3 --version` داخل rootfs. يجب تسجيل أي اختلاف متعمد في `docs/PORTING-NOTES.md`.
