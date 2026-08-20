# iSH Pure Go / Gio

هذا المجلد هو فرع إعادة كتابة مستقل لواجهة iSH باستخدام **Pure Go + Gio**. لا يستخدم Qt/QML أو Wails أو WebView. يعتمد renderer الطرفية على `github.com/viktomas/gritty` مع إبقاء PTY وiSH core خلف واجهات مستقلة.

## الحالة الحالية

النسخة الحالية هي **الهيكل الأولي القابل للترجمة**. تحتوي على نافذة Gio، toolbar أولي بأسلوب iSH، نموذج طرفية يستخدم parser/buffer من gritty، وPTY محلي لاختبار Linux. لم يتم بعد ربط iSH C core بالـ Go عبر cgo، وهذا مقصود حتى تبقى الحدود المعمارية واضحة.

## الهيكل

```text
cmd/ishgo/             نقطة تشغيل التطبيق
internal/app/          تركيب الخدمات وحلقة أحداث Gio
internal/ui/           toolbar وterminal renderer وأحداث الإدخال
internal/terminal/     gritty parser/buffer وsnapshot قابل للرسم
internal/session/      عقدة Session العامة
internal/platform/     تنفيذ PTY الخاص بالمنصة
internal/core/         مكان adapter iSH C core في المرحلة التالية
assets/                الأيقونات والموارد المرئية
docs/                  التصميم والقرارات المعمارية
tests/                 اختبارات التكامل وsmoke
```

## المتطلبات

يتطلب البناء المكتبي Go 1.26 أو أحدث، وحزم التطوير الخاصة بـ XKB وWayland وVulkan وEGL وX11 عند البناء على Ubuntu. يتطلب Gio Android SDK 31 أو أحدث عند إنشاء APK باستخدام `gogio`.

## البناء على Linux

```bash
cd go-pure
go mod tidy
go test ./...
go run ./cmd/ishgo
```

يمكن تحديد shell للاختبار المحلي:

```bash
ISH_SHELL=/bin/bash go run ./cmd/ishgo
```

## Z.AI الاختياري

يوجد عميل Pure Go اختياري في `internal/ai/zai` لطلبات Chat Completions المتوافقة مع واجهة Z.AI الرسمية. لا يدخل هذا العميل في مسار iSH core أو Wazero افتراضيًا، ولا يحتاج إلى CGO. عند الحاجة، تُضبط المتغيرات التالية خارج المستودع:

```bash
export ZAI_API_KEY=...
export ZAI_MODEL=glm-5.3
export ZAI_BASE_URL=https://api.z.ai/api/paas/v4/
```

يستخدم العميل `POST /chat/completions` مع `Authorization: Bearer`، ويُختبر محليًا عبر `httptest` دون إرسال بيانات إلى خدمة خارجية. راجع [ملاحظات تكامل Z.AI](docs/zai_integration_notes.md) و[دليل OpenAI الرسمي لـZ.AI](https://docs.z.ai/guides/develop/openai/python) قبل تفعيل اتصال حقيقي.

## Android

سيُضاف ملف manifest واسم الحزمة وسكربت `gogio` في مرحلة Android. لا ينبغي نسخ إعدادات Qt أو WebView إلى هذا الفرع. يجب أن يبقى backend الخاص بالـ PTY أو iSH core خلف `internal/session.Session`، لأن إنشاء PTY Android يحتاج تنفيذًا منفصلًا عن host shell.

## دمج iSH core

الواجهة المستقبلية المطلوبة هي:

```go
type CoreSession struct { /* C handle and lifecycle state */ }

func (s *CoreSession) Start(ctx context.Context, cols, rows int) error
func (s *CoreSession) Output() <-chan []byte
func (s *CoreSession) Write([]byte) error
func (s *CoreSession) Resize(cols, rows int) error
func (s *CoreSession) Close() error
```

بعد إضافة هذا adapter، سيستخدم التطبيق `CoreSession` بدل `platform.PTYSession` دون تغيير `internal/ui` أو `internal/terminal`.
