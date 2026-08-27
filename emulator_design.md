# تصميم محاكي x86_64

مرجع التعليمات الأساسي هو [Felix Cloutier x86 reference](https://www.felixcloutier.com/x86/)، وهو مرجع مشتق من Intel SDM ويعرض دلالة التعليمات وصيغها. يذكر المرجع نفسه أنه ليس بديلاً مثالياً عن Intel SDM، لذلك سيُستخدم لتصميم الاختبارات وفهم semantics، مع تجنب الادعاء بأنه يغطي كل تفاصيل الامتيازات والاستثناءات.

الإصدار الأول سيكون محاكيًا user-mode تعليميًا، لا virtual machine كاملة ولا hypervisor. سيمتلك 16 سجلاً عاماً بعرض 64 بت، وRIP، وRFLAGS، وذاكرة خطية يملكها المستدعي، مع تنفيذ خطوة واحدة أو عدد من الخطوات، ونقاط توقف، وأخطاء وصول للذاكرة.

مجموعة التنفيذ الأولى ستغطي التعليمات التي يستطيع decoder الحالي إنتاجها بشكل واضح: MOV وLEA، ADD وADC وSUB وSBB وAND وOR وXOR، CMP وTEST، INC وDEC وNEG وNOT، IMUL وMUL وDIV وIDIV بصيغها الأساسية، PUSH وPOP، CALL وRET، JMP وJcc، CMOVcc وSETcc، NOP وINT3. لن تُنفَّذ التعليمات privileged أو floating-point أو string أو AVX داخل مسار التنفيذ الأول؛ يمكن إضافة SIMD لاحقاً كمرحلة مستقلة.

سيستخدم المحاكي decoder الحالي كطبقة فك ترميز، ثم يحول x86asm arguments إلى قيم lvalue/rvalue عبر دوال صغيرة. يجب أن تكون كل عمليات القراءة والكتابة محدودة بحجم الذاكرة، وأن تُحدّث الأعلام الحسابية وفق عرض العملية لا وفق عرض C الفعلي.

## مراجع AMD الإضافية

دليل AMD الرسمي **AMD64 Technology: Volume 6: 128-Bit and 256-Bit XOP and FMA4 Instructions**, الإصدار 3.04، يصف صيغ XOP وFMA4 وVPCOM وVPCMOV وVPPERM وVPHADD/VPHsub وVFMA وغيرها، ويذكر أن مجموعة AMD64 مقسمة إلى general-purpose وsystem و128-bit media و64-bit media وx87 وXOP media subsets. المصدر: https://www.amd.com/content/dam/amd/en/documents/archived-tech-docs/programmer-references/43479.pdf

مرجع Felix Cloutier هو نسخة مشتقة آليًا من Intel SDM، ويذكر صراحةً أنه غير مثالي؛ لذلك ستبقى دقة التنفيذ النهائية مرتبطة بحالات الاختبار وبمراجعة AMD/Intel manuals. المصدر: https://www.felixcloutier.com/x86/

امتدادات AMD لا تعني أن كل تعليماتها user-mode: بعض SVM/system instructions تحتاج حالة امتيازات وvirtualization state، ولذلك يجب أن يعيد decoder اسم العملية مع تصنيف واضح، بينما يرفض executor تنفيذها إذا لم يوفّر نموذج CPU المطلوب.
