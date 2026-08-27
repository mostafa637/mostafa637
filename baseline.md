# خط أساس النسخة الحالية

بتاريخ 2026-08-25، نجح البناء والاختبار باستخدام:

```text
-std=c99 -Wall -Wextra -Wpedantic -Wconversion -Wshadow -O2
```

الحالات الناجحة: MOV immediate، ADD register/register، MOV مع SIB وdisplacement، وJNE short. لا توجد تحذيرات ترجمة في هذا الخط الأساسي.
