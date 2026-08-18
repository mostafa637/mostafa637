# SQLite في Pure Go

## القرار

يستخدم فرع Pure Go مكتبة `modernc.org/sqlite` خلف واجهة `database/sql`. يوفّر المشروع driver متوافقًا مع `database/sql` ومبنيًا من SQLite بلغة Go دون CGo، لذلك لا يحتاج إلى ربط `libsqlite3` أو compiler C داخل طبقة Go.

## سبب الاختيار

هذا الاختيار ينسجم مع هدف Pure Go ويجعل طبقة storage مستقلة عن Gio وgritty وPTY. كما أن المكتبة توفر واجهة virtual tables في `modernc.org/sqlite/vtab` إذا احتاج fakefs لاحقًا إلى ربط جداول افتراضية.

## قيد Android

توثيق الحزمة يعرض المنصات الموثقة في صفحة Go الحالية، لكن نجاح Android النهائي يعتمد على إصدار Go وهدف `GOOS/GOARCH` ونسخة toolchain. لذلك يجب اختبار Android في CI بدل اعتبار نجاح Linux ضمانًا كافيًا.

## مراجع

1. https://pkg.go.dev/modernc.org/sqlite — التوثيق الرسمي للحزمة، الإصدار المنشور v1.56.0 بتاريخ 2026-08-03، ووصفها كمنفذ SQLite دون CGo.
2. https://modernc.org/sqlite — المستودع الأساسي والتوثيق الرسمي وشرح virtual tables وgenerated sources.
