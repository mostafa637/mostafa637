# iSH Pure Go architecture

هذا المجلد هو إعادة كتابة مستقلة لطبقة التطبيق باستخدام **Pure Go + Gio**. لا يعتمد على Qt/QML أو Wails أو WebView. الهدف هو إبقاء طبقة العرض قابلة للاختبار وإبقاء دورة حياة PTY منفصلة عن renderer.

## حدود الوحدات

| الوحدة | المسؤولية | ما لا تفعله |
|---|---|---|
| `cmd/ishgo` | نقطة تشغيل التطبيق | لا يحتوي منطق الواجهة أو PTY |
| `internal/app` | تركيب الخدمات وربطها بدورة حياة Gio | لا يرسم عناصرًا مباشرة |
| `internal/ui` | رسم toolbar والطرفية وإدارة أحداث لوحة المفاتيح | لا يفتح PTY ولا يستدعي C |
| `internal/terminal` | نموذج الشاشة، parser adapter، الألوان، cursor، resize | لا يملك process lifecycle |
| `internal/session` | عقدة جلسة عامة للكتابة والقراءة وresize والإغلاق | لا يعرف تفاصيل Gio |
| `internal/platform` | تنفيذ PTY الخاص بالمنصة | لا يحتوي عناصر UI |
| `internal/core` | إعادة كتابة iSH core تدريجيًا في Go، وتشمل cpu وsyscall وkernel وstorage وfakefs وemu/FPU وsession | لا يخلط core مع renderer ولا يستخدم CGo |
| `assets` | الأيقونات والموارد المرئية | لا يحتوي كود تشغيل |
| `tests` | اختبارات التكامل والـ smoke | لا يعتمد على ملفات build مولدة |

## مسار البيانات

```text
PTY / iSH core
      │ bytes
      ▼
internal/session
      │ output channel
      ▼
internal/terminal (gritty parser + buffer adapter)
      │ immutable snapshot
      ▼
internal/ui (Gio renderer)
      │ key events / resize
      ▼
internal/session
```

## قرار gritty

يُستخدم gritty كمكوّن parser/buffer قابل لإعادة الاستخدام، لكن لا نستخدم `gritty/controller` مباشرةً في التطبيق؛ لأن controller الأصلي ينشئ shell خاصًا به ويمتلك PTY lifecycle. التطبيق يحتاج إلى إبقاء iSH core وPTY الحقيقيين تحت تحكمه، ولذلك يوفر adapter منفصلًا بين `session.Session` و`terminal.Model`.

## قرار iSH core

هذا الفرع مخصص لإعادة كتابة iSH core في Go، وليس مجرد adapter لـC. أُنجزت طبقة التخزين، وfakefs فوق rootfs، وbootstrap metadata، وCoreSession المرحلي الذي يحافظ على عقد PTY، وحد FPU 80-bit المعزول داخل `internal/core/emu/fpu`. أضيفت الآن `internal/core/cpu` لصفحات الذاكرة وCOW وgrows-down وMachineState، و`internal/core/syscall` لـi386 syscall ABI الأولي، و`internal/core/kernel` لعملية Go تجمع الذاكرة والسجلات والsyscalls وfakefs. ما زال التنفيذ الفعلي للتعليمات، وELF loader، وجدولة المهام، وإشارات Linux خارج هذا الحد المرحلي؛ لذلك يستمر PTY المضيف كمسار نقل مؤقت في CoreSession. تبقى نسخة C في مجلد `upstream` مرجعًا للمقارنة والاختبارات، لكنها ليست اعتمادًا للبناء. لا يجوز إدخال CGo إلى `go-pure`; أما فرع Qt/QML فيبقى مستقلًا ولا يتأثر بهذا القرار.

## قواعد التنظيم

تُمنع الاستدعاءات المباشرة من `internal/ui` إلى `os/exec` أو `syscall` أو cgo. وتُمنع الوحدات الدنيا من استيراد Gio. وتُحفظ اختبارات النموذج والطرفية خارج عملية Gio حتى تعمل على Linux وCI دون شاشة.
