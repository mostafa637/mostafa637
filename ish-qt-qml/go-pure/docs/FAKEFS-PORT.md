# مرحلة نقل fakefs وCoreSession

## ما نُقل إلى Go

أُضيفت حزمة `internal/core/fs` كطبقة filesystem Pure Go فوق `internal/core/storage`. تبقى بايتات الملفات والدلائل في rootfs حقيقي على نظام التشغيل، بينما تُخزن هوية inode وحقول `mode` و`uid` و`gid` و`rdev` في `meta.db` عبر `modernc.org/sqlite`. هذا يطابق فصل iSH الأصلي بين realfs ومعلومات fakefs [1].

يعرّف `IshStat` الحقول الأربعة نفسها الموجودة في `struct ish_stat` الأصلي، وتغطي API الحالية عمليات `Stat` و`Lstat` و`OpenFile` و`Create` و`Mkdir` و`Symlink` و`Readlink` و`Link` و`Unlink` و`Rename` و`SetAttr` و`Truncate` و`ReadDir` وقراءة/كتابة الملفات. كما توفر `FS()` محولًا إلى `io/fs.FS` حتى يمكن استخدام rootfs مع أدوات Go القياسية.

## bootstrap

يستخدم `BootstrapMetadata` عملية `filepath.WalkDir` لإدخال metadata للعناصر الموجودة في rootfs التي لا تملك سجلًا بعد. لا يعيد كتابة السجلات الموجودة، ولذلك يمكن تشغيله بعد انقطاع التهيئة أو عند بدء كل جلسة. القيم الافتراضية الحالية للـbootstrap هي uid/gid صفر، بما يطابق حساب root الأولي في iSH؛ أما صلاحيات الملف ونوعه فتُستنتج من `os.FileMode`.

## CoreSession

أُضيفت `internal/core/session.CoreSession` لتملك دورة حياة fakefs وPTY في كائن واحد ينفذ عقد `internal/session.Session`. عند ضبط `ISH_ROOTFS` يمر entrypoint عبر `core.GoFactory` وينشئ `CoreSession`; وعند عدم ضبطه يبقى host PTY متاحًا فقط لتطوير واجهة Linux دون rootfs.

> هذا هو جسر تشغيل مرحلي، وليس ادعاءً بأن shell أصبح يعمل داخل CPU emulator أو أن rootfs صار chroot/namespace معزولًا.

الـPTY الحالي يشغّل shell المضيف لتثبيت lifecycle وresize وقراءة/كتابة الطرفية. المرحلة التالية هي استبدال حقل PTY بمحرك `internal/core/kernel` و`internal/core/cpu` الذي ينفذ iSH userspace، مع إبقاء نفس واجهة `Session` وبدون تعديل Gio أو gritty.

## الاختبارات

تغطي الاختبارات دورة metadata والملف والدليل، hard links، rename لشجرة، symlink وreadlink، path traversal، واجهة `io/fs`، bootstrap rootfs، وPTY lifecycle. شغّلت الحزمة كذلك مع `CGO_ENABLED=0`؛ اعتماد Gio الرسومي يظل منفصلًا عن core وقد يحتاج دعم Vulkan/المنصة عند بناء التطبيق الكامل.

## المراجع

[1]: https://github.com/ish-app/ish/blob/master/fs/fake.c "iSH fakefs implementation"
[2]: https://github.com/ish-app/ish/blob/master/fs/fake-db.h "iSH fakefs database API and ish_stat definition"
