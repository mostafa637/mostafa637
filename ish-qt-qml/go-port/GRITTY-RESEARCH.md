# تقييم gritty

## المصدر

- المستودع: https://github.com/viktomas/gritty
- README الخام: https://raw.githubusercontent.com/viktomas/gritty/master/README.md
- controller: https://raw.githubusercontent.com/viktomas/gritty/main/controller/controller.go
- buffer: https://raw.githubusercontent.com/viktomas/gritty/main/buffer/buffer.go
- go.mod: https://raw.githubusercontent.com/viktomas/gritty/main/go.mod

## النتيجة

gritty هو terminal emulator مكتوب بلغة Go وواجهة Gio، وهدفه المرجعي تنفيذ VT100-ish قابل للقراءة والتعلم. بنيته تفصل بين `buffer` الذي يخزن شبكة الأحرف وخصائصها، و`parser` الذي يفسر control sequences، و`controller` الذي يربط PTY بالـbuffer وبإشارات إعادة الرسم.

الكود الحالي في gritty يبدأ أمرًا خارجيًا بواسطة `os/exec` وPTY، والـREADME يذكر صراحة أن البداية تكون مع `/bin/sh`. لذلك gritty يعالج **عرض الطرفية، ANSI/VT parsing، المؤشر، والألوان**، لكنه لا يوفر Linux kernel emulator، ولا syscall translation، ولا fakefs، ولا rootfs launcher، ولا تشغيل Alpine داخل Android دون صلاحيات chroot.

الـcontroller الحالي يعتمد على `github.com/creack/pty` وعلى `gioui.org`، ونسخة gritty المنشورة في `go.mod` تستخدم Gio `v0.2.0` وPTY `v1.1.18`، بينما مشروع Go الحالي يستخدم Gio `v0.10.2` و`creack/pty/v2`. لذلك الأنسب هو استخدام buffer/parser كمرجع أو نسخ/تكييف مكونات terminal، وليس إدخال controller القديم كاملًا دون مراجعة التوافق.

## قرار الدمج

سيتم دمج نموذج gritty في واجهة Go/Gio: تحويل خرج جلسة Alpine إلى parser ثم buffer، ورسم grid الأحرف بدل عرض النص الخام في `material.Label`. وسيبقى مصدر الجلسة منفصلًا خلف interface؛ في المرحلة الحالية يمكن أن يكون PTY للتجربة، لكن تشغيل Alpine الحقيقي يتطلب ربطه بمحرك iSH/Asbestos أو محرك syscall/emulation مكتوب بـGo. إدخال gritty وحده لا يجعل `/system/bin/sh` Alpine.
