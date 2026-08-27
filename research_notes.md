# نتائج فحص x86asm

المصدر الرسمي هو مستودع [golang/arch](https://github.com/golang/arch)، والمسار المعني هو `x86/x86asm`. توثيق الحزمة الرسمي موجود في [pkg.go.dev](https://pkg.go.dev/golang.org/x/arch/x86/x86asm)، ويعرّف الحزمة بأنها منفّذ لفك ترميز تعليمات x86، وتقبل `Decode` أنماط 16 و32 و64 بت.

تم استنساخ المستودع محلياً إلى `/home/ubuntu/x86asm-c99`، والنسخة الحالية هي commit `a79bd56a4b0298c1b35848a43595f2b26277c97e` بتاريخ 2026-08-25 من المستودع. ملفات التنفيذ الأساسية هي `inst.go`, `decode.go`, `gnu.go`, `intel.go`, `plan9x.go`, `tables.go`, `avx.go`, و`avx_tables.go`، بينما ملفات الاختبار ليست مطلوبة في المكتبة النهائية.

الحجم التقريبي لملفات التنفيذ 1.1 MB، منها جداول مولدة كبيرة: `tables.go` نحو 271 KB و`avx_tables.go` نحو 646 KB. خوارزمية فك الترميز table-driven وتستخدم برنامج bytecode في `decoder`، مع جداول AVX منفصلة. الواجهة العامة تشمل `Decode`, وأنواع التعليمات والوسائط والسجلات والذاكرة، إضافة إلى تنسيقات Intel وGNU وPlan 9.

تصميم المنفذ المقترح: مكتبة C99 مستقلة بلا Go، مع بادئة عامة `x86asm_` لمحاكاة مساحة أسماء C؛ ملف واجهة `x86asm.h`، تنفيذ أساسي `x86asm.c`، جداول ثابتة منفصلة، واختبارات/مثال بناء. سيتم تمثيل اتحاد الوسائط في C بدلاً من واجهة Go، مع دوال تهيئة وتنسيق آمنة تعتمد على buffer يمرره المستدعي.

ملاحظة مهمة: التحويل الكامل لجميع تعليمات x86/AVX يتطلب نقل الجداول المولدة، وليس من المناسب إسقاط هذه الجداول أو استبدالها بمحلل جزئي دون التصريح بذلك للمستخدم.

## متابعة التحويل اليدوي

تم إصلاح خريطة سجلات byte مع REX، بما في ذلك AH/CH/DH/BH مقابل SPL/BPL/SIL/DIL، وإضافة REX.R/REX.X/REX.B في مسارات ModR/M وSIB، ودعم address-size override داخل read_modrm. أضيفت خرائط accumulator، XADD، XCHG، BSWAP، shift groups، LOOP/LOOPE/LOOPNE/JCXZ، وتعليمات الأعلام الأساسية.

تمت إضافة سجلات ZMM0–ZMM31 ودعم EVEX أساسي لـ VADDPS/VADDPD/VXORPS/VPADDD/VMOVUPS/VMOVUPD، مع اختبار VPADDD ZMM. كما تمت إضافة مسار XOP ثلاثي البايت لـ AMD VPCMOV في XOP map 8 وفق جدول Linux opcode المأخوذ من AMD APM Volume 3 Appendix A، واختبار `8f e8 78 a2 c0`.

آخر تحقق: `make check CFLAGS='-std=c99 -Wall -Wextra -Wpedantic -Wconversion -Wshadow -O2'` نجح لكل اختبارات decoder والمحاكي.

## آخر توسعة

أضيف دعم SSE legacy لـ MOVUPS وADDPS وXORPS بصيغة معاملين destructive، مع ربطه بمخزن XMM/YMM/ZMM في المحاكي. أضيف دعم VEX map 0F38 للعمليات المنطقية AVX2: VAND وVANDN وVOR وVXOR، مع تصحيح mandatory prefix لـ VPADDD وفق ترميز VEX الصحيح.

أضيفت أيضًا صيغ MOV moffs A0–A3 وTEST accumulator A8/A9، وتعليمات SYSCALL وSYSRET وSYSENTER وSYSEXIT وUD2 كتعليمات decoded صراحة. أضيفت اختبارات تنفيذية لـ VADDPS وVMOVUPS باستخدام ذاكرة مستعارة مملوكة للمستخدم.

آخر تحقق بعد هذه التعديلات: `make check CFLAGS='-std=c99 -Wall -Wextra -Wpedantic -Wconversion -Wshadow -O2'` نجح لكل اختبارات decoder والمحاكي.

ملاحظة تنفيذية: formatter يطبع arguments حتى الوسيط ذي kind NONE؛ لذلك صيغ VEX ذات immediate مثل VPSHUFD/VBLEND يجب أن تستخدم arguments[0..2] فقط، مع immediate في arguments[2] بعد حذف vvvv الضمني. كما أن VEX C4 يحتاج طباعة البادئات الثلاثة C4/p0/p1 دون opcode.

ملاحظة formatter: `format_instruction` يطبع arguments بالترتيب حتى أول `X86ASM_ARG_NONE`. لذلك يجب ترتيب صيغ VEX ذات immediate في arguments بصورة متجاورة. في VPSHUFD وVBLEND يكون الترتيب dest, source, immediate؛ لا يجوز ترك خانة vvvv فارغة بين source وimmediate.

## توسعة SIMD الأخيرة

تمت إضافة VBLENDPS وVBLENDPD من خريطة VEX 0F3A بصيغة immediate، وتصحيح طباعة بادئة C4 ذات الثلاثة bytes. كما أضيف VPCMPEQB من 0F 74 بصيغة VEX، مع تنفيذ مقارنة packed bytes وإرجاع masks بقيمة 0xFF أو 0x00 لكل byte. اختبارات decoder والمحاكي نجحت بعد تصحيح ترتيب operands وموضع immediate.

## توسعة arithmetic SIMD

أضيف VPSUBD من VEX opcode FA في خريطة 0F، مع helper يطرح packed 32-bit integers modulo 2^32. أضيف اختبار decoder واختبار تنفيذي يستخدم arrays من uint32_t بدل float، ونجحت اختبارات البناء والمحاكي.

## حزمة SSE2 وinteger الأخيرة

أضيفت يدويًا إلى decoder والمحاكي تعليمات SSE2 التالية: `MOVD` و`MOVQ` بين GPR وXMM، `MOVDQA` و`MOVDQU`، `PADD*` و`PSUB*` بعروض byte/word/dword، `PCMPEQB` و`PCMPEQW` و`PCMPEQD`، `PAND` و`POR` و`PXOR`، shifts packed `PSLLW/D/Q` و`PSRLW/D/Q` و`PSRAW/D` و`PSLLDQ` و`PSRLDQ`، وضرب `PMULLW` و`PMULHW` و`PMULHUW` و`PMULUDQ`. أضيفت أيضًا `SHLD` و`SHRD`، `ROL` و`ROR` و`RCL` و`RCR`، و`CMPXCHG16B` إلى جانب `CMPXCHG8B`.

التنفيذ يستخدم buffers محلية ثابتة و`memcpy` لتجنب مشاكل alignment وeffective type، ولا يخصص أو يحرر ذاكرة المستخدم. عولج overflow المكتشف في PMULLW تحت UBSan، وأضيف رفض صريح للـ borrowed memory ذي data=NULL. نجحت اختبارات C99 الصارمة بعد إصلاح اختبارات count وModR/M، ونجحت اختبارات ASan/UBSan دون runtime errors؛ لا يزال يلزم إعادة تشغيل Sanitizer بعد آخر تعديل warning-clean قبل الإصدار النهائي.

هذه الإضافات لا تعني نقل جداول Go كاملة؛ ما زال decoder يدويًا جزئيًا، وما زالت مجموعات x87 وMMX الواسعة وAVX-512 الكاملة وAMD XOP/FMA4/3DNow/SVM غير مكتملة أو decode-only حسب التعليمات.

## مقارنات AVX2 signed packed

أضيفت `VPCMPGTB` و`VPCMPGTW` و`VPCMPGTD` في VEX map1، و`VPCMPGTQ` في VEX map2 مع W=1. التنفيذ يحول كل عنصر إلى النوع signed الموافق، ويكتب all-ones عند تحقق المقارنة وzero خلاف ذلك. اختبارات decoder والتنفيذ تغطي sign-bit للحالات 8/16/32/64-bit على YMM، ونجحت جولة strict وASan/UBSan بعد الإضافة.

## AVX2 min/max الموسعة

أضيفت `VPMINSB` و`VPMAXSB` و`VPMINUW` و`VPMAXUW` و`VPMINSD` و`VPMAXSD` و`VPMINUD` و`VPMAXUD` من VEX map2، مع تعميم helper المقارنة على عناصر 8/16/32/64-bit. اختبارات التنفيذ تستخدم قيم sign-bit لتفريق signed عن unsigned، ونجحت جولة strict وASan/UBSan دون أخطاء.

## مرجع PSHUFB

وفق مرجع Intel [PSHUFB](https://www.felixcloutier.com/x86/pshufb)، فإن `PSHUFB` legacy هو `66 0F 38 00 /r` بمعاملين destructive، بينما `VPSHUFB` يستخدم VEX.128/256. في صيغة VEX يكون الترتيب `dest=ModR/M.reg` و`src1=VEX.vvvv` و`src2=ModR/M.rm`. كل 128-bit lane يُخلط مستقلًا؛ bit 7 في كل control byte يكتب zero، وإلا تختار low nibble byte من source lane. سيُحافظ التنفيذ اليدوي على هذه lane semantics ولن يفعّل EVEX masks.

المصدر الخارجي المستخدم للتحقق: [Intel PSHUFB reference](https://www.felixcloutier.com/x86/pshufb).

## packed floating-point arithmetic

تمت مراجعة مراجع Intel لـ `SUBPS` و`MULPS` و`DIVPS` و`ADDPD`. تستخدم الصيغ legacy opcodes `0F 5C/59/5E` للـ single و`66 0F 5C/59/5E` للـ double، بينما تستخدم صيغ VEX نفس opcode في map1 مع `VEX.vvvv` كمصدر أول و`ModR/M.rm` كمصدر ثانٍ. ستُضاف arithmetic add/sub/mul/div فقط في هذه الدفعة؛ لا يُفترض دعم EVEX masking أو تقريبات غير موثقة.

المراجع: [Intel SUBPS](https://www.felixcloutier.com/x86/subps)، [Intel MULPS](https://www.felixcloutier.com/x86/mulps)، [Intel DIVPS](https://www.felixcloutier.com/x86/divps)، [Intel ADDPD](https://www.felixcloutier.com/x86/addpd).


## توسعة AVX2 packed multiply وshift

تم التحقق من encodings عبر GNU assembler و`objdump` ثم نقلها يدويًا إلى decoder: `VPMULLW` و`VPMULHW` و`VPMULHUW` و`VPMULUDQ`، إضافة إلى `VPSLLW/D/Q` و`VPSRLW/D/Q` و`VPSRAW/D` و`VPSLLDQ` و`VPSRLDQ`. في صيغ VEX الخاصة بالـ immediate shifts يكون `VEX.vvvv` هو الوجهة، و`ModR/M.rm` هو المصدر، بينما `ModR/M.reg` يحدد امتداد العملية؛ لذلك فُصل هذا المسار عن العمليات الثلاثية العامة. نُفذت `VPSLLDQ/VPSRLDQ` داخل كل 128-bit lane كما تقتضي AVX2، لا عبر YMM ككتلة واحدة.

أضيفت اختبارات decoder والتنفيذ لـ XMM/YMM، ونجحت اختبارات C99 الصارمة بعد إصلاح توقعات shift qword وإعادة تحميل المصادر بين التعليمات. ما زال هذا مسار AVX2 جزئيًا، وليس دعمًا عامًا لكل AVX/AVX-512.


## توسعة AVX2 saturating وmin/max

أضيفت يدويًا خرائط وتنفيذ `VPADDUSB` و`VPADDUSW` و`VPADDSB` و`VPADDSW`، و`VPSUBUSB` و`VPSUBUSW` و`VPSUBSB` و`VPSUBSW`، و`VPMINUB` و`VPMAXUB` و`VPMINSW` و`VPMAXSW`، و`VPAVGB` و`VPAVGW` و`VPSADBW`. أُعيد استخدام helpers SSE2 بعد تعميمها على 128/256-bit، مع saturation signed/unsigned منفصل وPSADBW داخل كل 64-bit lane. أضيفت اختبارات decoder والتنفيذ على YMM، ونجحت اختبارات C99 الصارمة بعد تصحيح مدخلات حالات saturation.

## packed word/dword shuffle

تؤكد مراجع Intel أن `PSHUFD` يستخدم `66 0F 70 /r ib` ويدوّر doublewords داخل كل 128-bit lane، وأن `VPSHUFD` بصيغة VEX.128/256 يستخدم operand المصدر من `ModR/M.r/m` وimm8، مع استقلال كل lane. كما أن `PSHUFLW` يستخدم `F2 0F 70 /r ib` و`PSHUFHW` يستخدم `F3 0F 70 /r ib`، وتنسخ الصيغ high/low words غير المعاد ترتيبها إلى النصف المقابل داخل كل 128-bit lane. ستدعم الدفعة decoder/executor لـ PSHUFD وPSHUFLW وPSHUFHW ونسخ VEX.128/256 فقط، دون EVEX masking أو MMX.

المراجع: [Intel PSHUFD](https://www.felixcloutier.com/x86/pshufd)، [Intel PSHUFLW](https://www.felixcloutier.com/x86/pshuflw)، [Intel PSHUFHW](https://www.felixcloutier.com/x86/pshufhw).


## تنفيذ packed shuffle

أضيفت `PSHUFD` و`PSHUFLW` و`PSHUFHW` legacy من opcode `0F 70 /r ib` مع prefixes `66/F2/F3`، وأضيفت صيغ `VPSHUFD` و`VPSHUFLW` و`VPSHUFHW` لـ VEX.128/256 في map1. التنفيذ ينسخ المصدر أولًا في صيغ word shuffle ثم يبدل low أو high four words داخل كل 128-bit lane؛ أما dword shuffle فيختار أربعة dwords مستقلة لكل lane بحسب imm8. لم تُفعّل صيغ EVEX masked أو MMX. اختبارات decoder والتنفيذ وASan/UBSan نجحت بعد إزالة تعريف helper مكرر.


## legacy packed signed comparisons

يراجع هذا النقل مرجع Intel الموحّد لـ `PCMPGTB/PCMPGTW/PCMPGTD`: الصيغ XMM هي `66 0F 64/65/66 /r`، والنتيجة all-ones أو zero لكل عنصر signed، مع operands destructive في legacy. صيغ VEX المقابلة تستخدم `VEX.vvvv` و`ModR/M.rm`، بينما لا تُفعّل هذه الشجرة صيغ EVEX mask registers أو MMX.

المصدر: [Intel PCMPGTB/PCMPGTW/PCMPGTD](https://www.felixcloutier.com/x86/pcmpgtb:pcmpgtw:pcmpgtd).


## mask extraction

تؤكد مراجع Intel أن `MOVMSKPS` يستخدم `0F 50 /r` ويستخرج sign bit من كل float، وأن `MOVMSKPD` يستخدم `66 0F 50 /r` ويستخرج sign bit من كل double. صيغ VEX.128/256 تزيد عدد العناصر إلى 4/8 للـ single و2/4 للـ double، مع تصفير البتات العليا للوجهة. أما `PMOVMSKB` فيستخدم `66 0F D7 /r` لـ XMM، و`VPMOVMSKB` يستخدم VEX map1 opcode `D7` ويخرج mask بعرض 16 أو 32 bit من أعلى bit لكل byte. هذه الدفعة لا تفعّل MMX أو EVEX، وتتحقق من أن ModR/M.MOD يجب أن يكون register source، وأن الوجهة GPR لا vector.

المراجع: [Intel MOVMSKPS](https://www.felixcloutier.com/x86/movmskps)، [Intel MOVMSKPD](https://www.felixcloutier.com/x86/movmskpd)، [Intel PMOVMSKB](https://www.felixcloutier.com/x86/pmovmskb).


## نتيجة mask extraction

أضيفت يدويًا `MOVMSKPS` و`MOVMSKPD` و`PMOVMSKB` legacy، ونظائر `VMOVMSKPS` و`VMOVMSKPD` و`VPMOVMSKB` بصيغ VEX.128/256. التنفيذ يقرأ مصدر XMM/YMM، يضع sign-bit أو أعلى bit لكل byte في mask منخفض داخل GPR، ويكتب الوجهة بعرض 32 بت مع التصفير العلوي، مع احترام عرض المصدر في صيغ VEX. أضيفت اختبارات decoder والتنفيذ لترتيب bits على XMM وYMM، ونجحت strict وASan/UBSan. لم يُفعّل مسار MMX القديم ولا EVEX mask extraction.


## PTEST وVPTEST

مرجع Intel يحدد `PTEST` كـ `66 0F 38 17 /r` و`VPTEST` كـ VEX map2 opcode `17` مع `pp=66`، بعرض XMM أو YMM. لا تُعدّل الوجهة؛ تُحسب ZF من `(src AND dest)==0` وCF من `(src AND NOT dest)==0`، وتُصفّر OF وAF وPF وSF. VEX.vvvv محجوز ويجب أن يكون 1111؛ هذه الدفعة ستنفذ صيغ register/memory التي يدعمها نموذج operand الحالي، دون EVEX.

المرجع: https://www.felixcloutier.com/x86/ptest


## packed FP MIN/MAX

أضيفت `MINPS` و`MAXPS` و`MINPD` و`MAXPD` legacy، ونظائر VEX `VMINPS` و`VMAXPS` و`VMINPD` و`VMAXPD` بعرضي XMM/YMM. يختار التنفيذ source2 عند وجود NaN في أي من المصدرين، ويختار source2 عند تساوي الصفرين للحفاظ على signed-zero، ثم يستعمل المقارنة العادية للحالات finite المتبقية. أضيفت اختبارات decoder والتنفيذ تشمل NaN وsigned-zero وNDS operand order، ونجحت strict وASan/UBSan. EVEX masking/broadcasting غير مفعّل.

المراجع: https://www.felixcloutier.com/x86/minps ، https://www.felixcloutier.com/x86/maxps ، https://www.felixcloutier.com/x86/minpd ، https://www.felixcloutier.com/x86/maxpd


## CMPPS وCMPPD

تحدد مراجع Intel أن `CMPPS` هو `0F C2 /r ib` و`CMPPD` هو `66 0F C2 /r ib`، بينما تستعمل صيغ VEX opcode `C2` مع `pp=0` للـ single و`pp=1` للـ double، وتكون الوجهة `ModR/M.reg` والمصدر الأول `VEX.vvvv` والمصدر الثاني `ModR/M.rm` ثم imm8. في legacy تُستعمل bits 2:0 من imm8، وفي VEX bits 4:0؛ القيم الأعلى من 7 في VEX تدعم predicates الإضافية. نتيجة كل عنصر packed هي all-ones أو zero بعرض العنصر، مع ordered/unordered حسب وجود NaN. هذه الدفعة ستبدأ بتمثيل instruction منفصل واحد لكل single/double مع حفظ imm8، وتنفيذ مجموعة predicates كاملة في VEX وذات القيم 0..7 في legacy، دون EVEX opmask.

المراجع: https://www.felixcloutier.com/x86/cmpps ، https://www.felixcloutier.com/x86/cmppd


## CMPPS/CMPPD وVCMPPS/VCMPPD

أضيفت `CMPPS` و`CMPPD` legacy مع imm8 مقيد إلى bits 2:0، ونظائر `VCMPPS` و`VCMPPD` VEX مع imm8 إلى bits 4:0. التنفيذ يدعم ordered/unordered predicates 0..31، ويخرج all-ones أو zero بعرض كل عنصر، ويستعمل `isnan` لاكتشاف unordered دون تحويلات aliasing أو تخصيص ذاكرة. اختبارات decoder تغطي صيغ legacy وVEX وترتيب NDS، واختبارات التنفيذ تغطي equality وunordered وgreater-than وNaN، ونجحت strict وASan/UBSan. EVEX opmask وfloating-point exception signaling التفصيلي غير مفعّلين؛ الإصدار يعيد semantics النتيجة الأساسية فقط.

المراجع: https://www.felixcloutier.com/x86/cmpps ، https://www.felixcloutier.com/x86/cmppd


## scalar SSE/AVX arithmetic

تحدد مراجع Intel `ADDSS/SUBSS` بصيغة legacy `F3 0F 58/5C /r`، و`ADDSD/SUBSD` بصيغة `F2 0F 58/5C /r`، مع VEX `VADDSS/VSUBSS` و`VADDSD/VSUBSD`. في legacy يتغير العنصر الأدنى فقط وتبقى بقية XMM كما هي؛ في VEX.128 يُنسخ الجزء الأعلى من المصدر الأول إلى الوجهة ثم تُصفّر البتات الأعلى من 128. هذه الدفعة ستقتصر على add/sub scalar، مع memory scalar operands، دون EVEX masks أو rounding controls.

المراجع: https://www.felixcloutier.com/x86/addss ، https://www.felixcloutier.com/x86/subss ، https://www.felixcloutier.com/x86/addsd ، https://www.felixcloutier.com/x86/subsd


## scalar ADD/SUB

أضيفت `ADDSS` و`SUBSS` و`ADDSD` و`SUBSD` legacy، ونظائر VEX `VADDSS` و`VSUBSS` و`VADDSD` و`VSUBSD`. في legacy يُحدّث العنصر الأدنى داخل destination مع إبقاء بقية XMM، وفي VEX تُنسخ بقية XMM من المصدر الأول (`VEX.vvvv`) بينما تُحدّث قيمة العنصر الأدنى. يدعم executor مصدر register أو memory scalar بعرض 4 أو 8 بايت عبر reader مستقل يتحقق من حدود borrowed memory. أضيفت اختبارات decoder والتنفيذ لصيغ single/double، ونجحت strict وASan/UBSan. EVEX masks وrounding controls غير مفعّلة.

المراجع: https://www.felixcloutier.com/x86/addss ، https://www.felixcloutier.com/x86/subss ، https://www.felixcloutier.com/x86/addsd ، https://www.felixcloutier.com/x86/subsd


## scalar MUL/DIV

تستخدم `MULSS/DIVSS` prefixes `F3` مع opcodes `59/5E`، وتستخدم `MULSD/DIVSD` prefixes `F2` مع opcodes `59/5E`. صيغ VEX تقابلها `VMULSS/VDIVSS` و`VMULSD/VDIVSD` مع المصدر الأول في `VEX.vvvv` والمصدر الثاني في ModR/M.r/m. legacy يحدّث العنصر الأدنى ويحافظ على بقية XMM، وVEX.128 ينسخ الجزء الأعلى من المصدر الأول إلى الوجهة. سيُستكمل الربط مع scalar add/sub الموجود، دون EVEX masks أو rounding controls.

المراجع: https://www.felixcloutier.com/x86/mulss ، https://www.felixcloutier.com/x86/divss ، https://www.felixcloutier.com/x86/mulsd ، https://www.felixcloutier.com/x86/divsd


## حزمة scalar MUL/DIV وتدقيق VEX.128

أضيفت `MULSS` و`DIVSS` و`MULSD` و`DIVSD` legacy، ونظائر VEX `VMULSS` و`VDIVSS` و`VMULSD` و`VDIVSD`. صيغ single تستخدم prefix `F3`، وصيغ double تستخدم `F2`، مع opcode `59` للضرب و`5E` للقسمة. صيغ VEX تستخدم `VEX.vvvv` مصدرًا أولًا وModR/M.r/m مصدرًا ثانيًا؛ memory metadata يحدد 4 بايت لـ single و8 بايت لـ double.

ينفذ المحاكي العمليات عبر `float`/`double` مع `memcpy` للقراءة والكتابة، ويحافظ على بقية XMM في legacy ونسخها من المصدر الأول في VEX. أضيفت اختبارات decoder لجميع الصيغ واختبار semantics مستقل للـ legacy register paths. كما أضيف اختبار VEX scalar يثبت مسح bytes 16..63 من الحالة الفيزيائية، وأضيف helper داخلي لتنفيذ upper-zeroing لمسار VEX.128 scalar. نجحت جولة strict وجولة ASan/UBSan، ثم أزيلت artifacts عبر `make clean`.

المراجع: [MULSS](https://www.felixcloutier.com/x86/mulss)، [DIVSS](https://www.felixcloutier.com/x86/divss)، [MULSD](https://www.felixcloutier.com/x86/mulsd)، [DIVSD](https://www.felixcloutier.com/x86/divsd).


## مرجع MOVSS وMOVSD

تستخدم `MOVSS` prefix `F3` مع opcode `0F 10/11`، وتستخدم `MOVSD` prefix `F2` مع opcode `0F 10/11`. في legacy register-to-register يُنقل العنصر الأدنى مع إبقاء بقية XMM، بينما memory load يمسح الجزء الأعلى من XMM حتى 128 بت، وmemory store يكتب scalar فقط. صيغ VEX register-register لها ثلاثة operands: الوجهة، المصدر الأول في `VEX.vvvv` الذي يحدد الجزء الأعلى، والمصدر الثاني في ModR/M.reg أو rm بحسب صيغة `10/11`; أما VEX memory load/store فتستخدم operandين scalar مع `VEX.vvvv` محجوزًا ويجب أن يكون 1111، وVEX.L يجب أن يكون صفرًا. VEX.128 يمسح الحالة الفيزيائية فوق 128 بت.

المراجع: [MOVSS](https://www.felixcloutier.com/x86/movss)، [MOVSD](https://www.felixcloutier.com/x86/movsd).


## حزمة MOVSS/MOVSD scalar

أضيفت opcodes منفصلة `MOVSS` و`MOVSD_SCALAR` لتجنب تعارض اسم `MOVSD` الخاص بتعليمات السلاسل، مع `VMOVSS` و`VMOVSD`. يدعم decoder صيغ legacy register-register وmemory load/store، وصيغ VEX register merge وmemory load/store على map1 opcode `10/11`. في صيغ VEX memory يُرفض `VEX.vvvv` غير المحجوز، ويُرفض `VEX.L=1` لهذه التعليمات. executor يحافظ على الجزء الأعلى في legacy register moves، يمسح الجزء الأعلى من XMM في legacy memory loads، وينفذ VEX source merge مع مسح upper physical state.

أضيفت اختبارات formatter/decoder، واختبارات semantics لـ register merge وmemory load/store وVEX scalar. نجحت strict build دون warnings وASan/UBSan، ثم أزيلت artifacts عبر `make clean`. ما زال يلزم لاحقًا توسيع اختبار MOVSD memory ورفض جميع الحقول المحجوزة في صيغ VEX ذات store على نحو مستقل قبل إصدار جديد.

المراجع: [MOVSS](https://www.felixcloutier.com/x86/movss)، [MOVSD](https://www.felixcloutier.com/x86/movsd).


## reserved VEX fields

تؤكد مراجع Intel أن `VMOVMSKPS` و`VMOVMSKPD` و`VPMOVMSKB` تتطلب `VEX.vvvv=1111b`، وأن `VPTEST` يتطلب الشرط نفسه. صيغ mask extraction تحتاج أيضًا ModR/M.MOD=11 لأن المصدر سجل vector، بينما يمكن لـ VPTEST قبول memory source. هذه القيود مستقلة عن zero-extension للوجهة، ويجب أن يعيد decoder خطأً صريحًا عند مخالفتها بدل قبول encoding تقريبي.

المراجع: [MOVMSKPS](https://www.felixcloutier.com/x86/movmskps)، [MOVMSKPD](https://www.felixcloutier.com/x86/movmskpd)، [PMOVMSKB](https://www.felixcloutier.com/x86/pmovmskb)، [PTEST](https://www.felixcloutier.com/x86/ptest).


## تشديد VEX reserved fields

شُدّد decoder بحيث ترفض `VMOVMSKPS` و`VMOVMSKPD` و`VPMOVMSKB` إذا كان `VEX.vvvv` غير 1111، مع إبقاء شرط ModR/M.MOD=11 لهذه التعليمات، بينما ظل `VPTEST` يستخدم الرفض الخاص به. أضيفت ثلاث حالات `check_error` تغطي vvvv غير الصحيح، ونجحت اختبارات decoder والمحاكي تحت strict. لا تزال تغطية EVEX reserved fields خارج نطاق هذا المسار، كما أن الحقول الخاصة بتعليمات VEX الأخرى تحتاج تدقيقًا تدريجيًا لا تعميمًا غير موثق.


## VCMP predicate validation

تؤكد صفحة Intel الموحدة لـ `CMPPS` أن VEX/EVEX يستخدم bits 4:0 من imm8، بينما bits 5:7 محجوزة؛ legacy يستخدم bits 2:0 وتكون bits 3:7 محجوزة. كما تؤكد أن VEX.128 يمسح الحالة فوق 128 بت. لذلك سيُرفض imm8 ذو أي bit من 5 إلى 7 في `VCMPPS/VCMPPD` بدل الاقتصار على mask منخفضة فقط.

المصدر: [Intel CMPPS](https://www.felixcloutier.com/x86/cmpps).


## VEX validation الإضافي

أضيف رفض صريح لـ `VEX.L=1` في `VMOVSS` و`VMOVSD` scalar لأن Intel يوصي ويقيد هذه الصيغ بعرض VEX.128 فقط. كما أضيف رفض bits 5:7 في imm8 الخاص بـ `VCMPPS/VCMPPD`، إذ تحدد VEX bits 4:0 فقط predicate. اختبارات `check_error` تغطي `VMOVSS` ذي L غير الصحيح و`VCMPPS` ذي high reserved bit. نجحت strict build دون warnings وASan/UBSan، ثم نُظفت artifacts.


## scalar MIN/MAX

تستخدم `MINSS/MAXSS` prefix `F3` وopcode `5D/5F`، وتستخدم `MINSD/MAXSD` prefix `F2` مع opcode `5D/5F`. في legacy يتصرف destination كمصدر أول ويحافظ على بقية XMM؛ في VEX يكون المصدر الأول في `VEX.vvvv`، ويُنسخ الجزء الأعلى من المصدر الأول إلى الوجهة مع upper-zeroing فوق 128 بت. طبقًا لمراجع Intel، يفوز المصدر الثاني عند NaN أو عند تساوي ±0، ولذلك لا تكفي المقارنة العادية في C؛ يلزم اختبار `isnan` واختيار source2 صراحةً.

المراجع: [MINSS](https://www.felixcloutier.com/x86/minss)، [MAXSS](https://www.felixcloutier.com/x86/maxss)، [MINSD](https://www.felixcloutier.com/x86/minsd)، [MAXSD](https://www.felixcloutier.com/x86/maxsd).


## الدفعة التالية: PABS/VPABS

اختيرت عائلة `PABSB/PABSW/PABSD` و`VPABSB/VPABSW/VPABSD` كدفعة SSSE3/AVX/AVX2 مترابطة. تعمل النسخ legacy على XMM بعرض 128 بت، بينما تعمل VEX.128/256 على XMM/YMM مع zeroing للحالة فوق العرض، و`VEX.vvvv` محجوز ويجب أن يساوي 1111. ستُنفذ absolute value لمدخلات signed byte/word/dword باستخدام عمليات unsigned محددة، مع إبقاء دعم PABSQ/EVEX خارج هذه الدفعة لعدم وجود نموذج EVEX عام كافٍ.

المصدر: [Intel PABSB/PABSW/PABSD/PABSQ](https://www.felixcloutier.com/x86/pabsb:pabsw:pabsd:pabsq).


## PABS/VPABS implementation result

أضيفت `PABSB/PABSW/PABSD` legacy على XMM و`VPABSB/VPABSW/VPABSD` على XMM/YMM. decoder يضبط memory width إلى 16 بايت في legacy وإلى 16/32 بايت حسب VEX.L، ويرفض `VEX.vvvv` غير 1111 في VPABS. executor يستخدم absolute value unsigned آمنة لعناصر signed byte/word/dword، بما في ذلك INT_MIN، ويطبق VEX upper-zeroing للحالة الفيزيائية فوق العرض المكتوب. أضيفت اختبارات decoder/formatter وحالات semantics للأنواع الثلاثة، ونجحت strict build دون warnings وASan/UBSan. بقي `PABSQ` وEVEX masked forms خارج هذه الدفعة عمدًا.


## PSIGN/VPSIGN

تؤكد Intel أن `PSIGNB/W/D` يحدّث destination destructive: إذا كان control signed سالبًا يُنَفّذ negate لقيمة destination، وإذا كان صفرًا تصبح القيمة صفرًا، وإذا كان موجبًا تبقى destination. صيغ VEX تستخدم `dest=ModR/M.reg` و`src1=VEX.vvvv` و`src2=ModR/M.rm`، وتطبق sign control من src2 على src1. VEX.128 يمسح الحالة فوق 128 بت، وVEX.L=1 محجوز/غير صالح للتعليمات scalar? في هذه العائلة VEX.L=1 هو شرط VPABS؟ مراجع PSIGN تثبت أن L=1 غير صالح للنسخ 128، بينما النسخ 256 مدعومة؛ سيُقبل L حسب العرض في decoder.

المصدر: [Intel PSIGNB/PSIGNW/PSIGND](https://www.felixcloutier.com/x86/psignb:psignw:psignd).


## PHADD/VPHADD

تعمل `PHADDW` و`PHADDD` و`PHADDSW` على جمع كل زوج متجاور من عناصر destination ثم المصدر، وتعبئة النتائج في النصفين المتتاليين من الوجهة. legacy XMM صيغة destructive بوسيطين، بينما VEX صيغة NDS بثلاثة operands: `dest=reg` و`src1=VEX.vvvv` و`src2=rm`. في VEX.256 تُجرى العملية داخل كل 128-bit lane دون عبور حدود lane، ثم تُكتب النتائج إلى YMM. `PHADDSW` يستخدم signed-word saturation؛ PHADDW/PHADDD لا يغيران الأعلام ولا يبلغان overflow. أستبعد MMX وEVEX من الدفعة الحالية.

المصادر: [Intel PHADDW/PHADDD](https://www.felixcloutier.com/x86/phaddw:phaddd)، [Intel PHADDSW](https://www.felixcloutier.com/x86/phaddsw).


## PHSUB/VPHSUB

تعمل `PHSUBW` و`PHSUBD` بطرح العنصر الأعلى في كل زوج من العنصر الأدنى داخل source وdestination، ثم تعبئة النتائج في النصفين المتتاليين. `PHSUBSW` نفس الترتيب مع signed-word saturation. صيغ legacy XMM destructive بوسيطين، وصيغ VEX NDS بثلاثة operands، وVEX.256 يحافظ على حدود 128-bit lanes. أُبقيت صيغ MMX خارج المنفذ الحالي، ولا تُنفذ تعليمات غير مربوطة بالـexecutor.

المصادر: [Intel PHSUBW/PHSUBD](https://www.felixcloutier.com/x86/phsubw:phsubd)، [Intel PHSUBSW](https://www.felixcloutier.com/x86/phsubsw).


## PMADDUBSW/VPMADDUBSW

تضرب التعليمة كل byte unsigned من destination/source1 في byte signed المناظر من source2، ثم تجمع كل زوج متجاور من النواتج الوسيطة وتُشبع الناتج إلى signed word. legacy XMM صيغة destructive بوسيطين، وVEX صيغة NDS بثلاثة operands؛ VEX.128/256 تصفر الحالة الفيزيائية فوق العرض. صيغ MMX وEVEX masked خارج هذه الدفعة.

المصدر: [Intel PMADDUBSW](https://www.felixcloutier.com/x86/pmaddubsw).


## PMADDWD/VPMADDWD

تضرب `PMADDWD` كل زوج من signed 16-bit words المتناظرة، ثم تجمع كل حاصلين إلى signed 32-bit dword. لا توجد saturation؛ النتيجة تلتف modulo 32-bit، مع حالة موثقة خاصة عندما تكون القيمتان في الزوجين `0x8000` حيث النتيجة `0x80000000`. legacy XMM destructive، وVEX.128/256 NDS بثلاثة operands، مع zeroing للحالة الفيزيائية فوق العرض في VEX.

المصدر: [Intel PMADDWD](https://www.felixcloutier.com/x86/pmaddwd).


## PMULDQ/VPMULDQ

تستخدم `PMULDQ` و`VPMULDQ` عناصر signed 32-bit ذات الفهارس الزوجية فقط من كل source، أي dword رقم 0 و2 في XMM، و0 و2 و4 و6 في YMM، وتنتج signed 64-bit qwords. legacy XMM destructive، بينما VEX NDS بثلاثة operands مع zeroing فوق 128 أو 256 بحسب العرض. المصدر الذاكري يُقرأ بعرض المتجه الكامل رغم استخدام العناصر الزوجية فقط. صيغ EVEX masked وMMX خارج الدفعة.

المصدر: [Intel PMULDQ](https://www.felixcloutier.com/x86/pmuldq).


## PMOVSX/PMOVZX

أضيفت صيغ `PMOVSXBW/BD/BQ/WD/WQ/SXDQ` و`PMOVZXBW/BD/BQ/WD/WQ/DQ` legacy، ونسخ `VPMOV*` VEX.128/VEX.256. التحويلات sign/zero extend من source XMM أو memory بالعرض المحدد إلى destination XMM/YMM؛ في VEX.256 يبقى source register XMM بينما يكون destination YMM، و`VEX.vvvv` محجوز ويُرفض عند عدم ترميزه كـ `1111`. اختبرت الدفعة جميع درجات التحويل، signed/unsigned boundaries، legacy memory، VEX.256 register وmemory، وupper-zeroing، مع نجاح strict وASan/UBSan. صيغ EVEX masked وMMX خارج النطاق.

المراجع: [Intel PMOVSX](https://www.felixcloutier.com/x86/pmovsx)، [Intel PMOVZX](https://www.felixcloutier.com/x86/pmovzx).


## PBLENDVB/VPBLENDVB

تستخدم `PBLENDVB` legacy المصدر الوجهة نفسه، والمصدر الثاني `xmm2/m128`، والقناع الضمني `XMM0`؛ يُنسخ كل byte من source عند كون MSB في byte القناع مساويًا 1، وإلا تبقى قيمة الوجهة. صيغ VEX تستخدم `VEX.vvvv` كمصدر أول، و`ModR/M.r/m` كمصدر ثانٍ، وتشفّر سجل القناع في nibble العالي من imm8، مع تجاهل nibble المنخفض. `VEX.W=1` و`VEX.L=1` غير صالحين للنسخة VEX.128؛ نسخة VEX.256 تستخدم YMM للمصدرين والقناع، وVEX.128 تمسح الحالة الفيزيائية فوق 128 بت. legacy opcode هو `66 0F 38 10 /r`، وVEX opcode هو `VEX.128/256.66.0F3A 4C /r ib`.

المصدر: [Intel PBLENDVB](https://www.felixcloutier.com/x86/pblendvb).


## PHMINPOSUW/VPHMINPOSUW

تبحث التعليمة عن أصغر unsigned 16-bit word بين ثمانية عناصر في source XMM، وتضع القيمة في word الأدنى وindex من 0 إلى 7 في bits 16..18، مع اختيار أول index عند التساوي وتصفير بقية destination. `PHMINPOSUW` legacy destructive من source XMM أو m128، بينما `VPHMINPOSUW` VEX.128 له source XMM أو m128، و`VEX.vvvv=1111` و`L=0` محجوزان/مطلوبان. legacy يحافظ على الحالة الفيزيائية فوق 128، وVEX يمسحها.

المصدر: [Intel PHMINPOSUW](https://www.felixcloutier.com/x86/phminposuw).


## نتيجة PHMINPOSUW

أضيفت `PHMINPOSUW` و`VPHMINPOSUW` يدويًا مع minimum unsigned word، وأول index عند التساوي، وكتابة القيمة في word الأدنى وindex في word التالي وتصفير بقية destination. اختبرت الصيغ legacy وVEX.128، register وmemory، حفظ الحالة فوق 128 في legacy ومسحها في VEX، ورفض VEX.L=1. اجتازت الدفعة strict وASan/UBSan، ولا تتضمن صيغ EVEX أو MMX.


## PBLENDW/VPBLENDW

تختار `PBLENDW` كل word من المصدر الثاني أو تترك destination legacy كما هو بحسب bit مطابق في imm8؛ legacy destructive على XMM. صيغ `VPBLENDW` تستخدم VEX.vvvv كمصدر أول وModR/M.r/m كمصدر ثانٍ، وتستخدم imm8 كقناع word من 8 bits؛ VEX.128 تمسح الحالة الفيزيائية فوق 128، وVEX.256 تعمل على YMM. النسخ EVEX غير داخلة في هذه الدفعة.

المصدر: [Intel PBLENDW](https://www.felixcloutier.com/x86/pblendw).

## PALIGNR/VPALIGNR

مرجع Intel يعرّف `PALIGNR` legacy كـ destructive operation تجمع destination العالي مع source المنخفض ثم تزحزح composite بمقدار `imm8 * 8` وتستخرج 128 bits؛ القيم الأكبر من 32 للـ128-bit تعطي صفرًا. `VPALIGNR` تستخدم VEX.vvvv كمصدر أول وModR/M.r/m كمصدر ثانٍ، وتعيد zeroing فوق 128 في VEX.128. في VEX.256 تُطبّق العملية مستقلاً على كل 128-bit lane باستخدام نفس imm8، وليس على composite واحد بعرض 512 bits. ستبقى صيغ MMX وEVEX masked خارج هذه الدفعة.

المصدر: [Intel PALIGNR](https://www.felixcloutier.com/x86/palignr).

## PINSR/PEXTR

تستخدم عائلة `PINSRB/PINSRD/PINSRQ` خريطة `66 0F 3A` مع opcodes `20/22` وimm8، وتُبقي legacy destination XMM مع اختيار العنصر من imm8[3:0] للـbyte، imm8[1:0] للـdword، وimm8[0] للـqword. صيغ VEX.128 تستخدم `VEX.vvvv` كمصدر XMM أول، وModR/M.r/m كمصدر GPR أو memory، وتتطلب VEX.L=0؛ VEX.W يحدد qword في opcode 22، بينما byte form تعامل W1 كصيغة byte في 64-bit mode.

تستخدم `PEXTRB/PEXTRD/PEXTRQ` opcodes `14/16` من الخريطة نفسها، مع الوجهة ModR/M.r/m والـsource XMM في ModR/M.reg وimm8. في VEX.128 يكون VEX.vvvv محجوزًا ويجب أن يكون 1111، وVEX.L=0. الوجهة register تُصفّر البتات العليا، أما memory فتكتب حجم العنصر فقط. ستنفذ الدفعة XMM وGPR/memory دون EVEX أو MMX.

المراجع: [Intel PINSRB/PINSRD/PINSRQ](https://www.felixcloutier.com/x86/pinsrb:pinsrd:pinsrq)، [Intel PEXTRB/PEXTRD/PEXTRQ](https://www.felixcloutier.com/x86/pextrb:pextrd:pextrq)، و[Intel PEXTRW](https://www.felixcloutier.com/x86/pextrw).

## PMULLD/VPMULLD

تستخدم `PMULLD` ترميز `66 0F 38 40 /r` وتضرب أربعة signed dword عناصر في XMM destination/source، مع تخزين أقل 32 bit من كل حاصل وضمان بقاء upper physical XMM/ZMM legacy دون تغيير. تستخدم `VPMULLD` الخريطة نفسها مع `VEX.128/256.66.0F38.WIG 40 /r`، حيث يكون المصدر الأول `VEX.vvvv` والمصدر الثاني ModR/M.r/m، وتُصفّر الحالة الفيزيائية فوق 128 أو 256 bit حسب L. صيغ EVEX/PMULLQ خارج نطاق هذه الدفعة.

مرجع Intel/Felix: https://www.felixcloutier.com/x86/pmulld:pmullq

## Legacy PMIN/PMAX SSE4.1

مراجع Intel/Felix تثبت أن الصيغ legacy الإضافية في خريطة `66 0F 38` هي: `PMINSB=38`, `PMINSD=39`, `PMINUW=3A`, `PMINUD=3B`, `PMAXSB=3C`, `PMAXSD=3D`, `PMAXUW=3E`, و`PMAXUD=3F`. كلها XMM-to-XMM أو XMM-to-m128، وتختار الحد الأدنى/الأقصى لكل عنصر signed أو unsigned حسب الاسم، مع إبقاء upper physical vector state دون تغيير في legacy. هذه الدفعة لا تضيف MMX أو EVEX؛ التنفيذ سيعيد استخدام helper minmax الموجود مع element widths 1/2/4.

المراجع: https://www.felixcloutier.com/x86/pminsb:pminsw، https://www.felixcloutier.com/x86/pminsd:pminsq، https://www.felixcloutier.com/x86/pminub:pminuw، https://www.felixcloutier.com/x86/pmaxsb:pmaxsw:pmaxsd:pmaxsq، https://www.felixcloutier.com/x86/pmaxub:pmaxuw، https://www.felixcloutier.com/x86/pmaxud:pmaxuq

## BLENDVPS/VBLENDVPS وBLENDVPD/VBLENDVPD

تستخدم الصيغ legacy `BLENDVPS` و`BLENDVPD` خريطة `66 0F 38` مع opcode `14` و`15`، وتكون الوجهة destructive والـmask ضمنيًا XMM0. تستخدم نظائر VEX خريطة `VEX.128/256.66.0F3A.W0` مع opcode `4A` للـsingle و`4B` للـdouble؛ المصدر الأول هو `VEX.vvvv`، والمصدر الثاني ModR/M.r/m، ويُحدد mask register من imm8[7:4] بينما imm8[3:0] مهمل. تختار BLENDVPS dword lanes وBLENDVPD qword lanes وفق أعلى bit في كل عنصر، وتُصفّر VEX.128 upper physical state. صيغ EVEX خارج النطاق.

المراجع: https://www.felixcloutier.com/x86/blendvps و https://www.felixcloutier.com/x86/blendvpd

## BLENDPS/BLENDPD legacy

تستخدم `BLENDPS` ترميز `66 0F 3A 0C /r ib` وتختار أربعة dword lanes وفق imm8[3:0]، بينما تستخدم `BLENDPD` ترميز `66 0F 3A 0D /r ib` وتختار qword lanes وفق imm8[1:0]. كلتاهما destructive legacy XMM-to-XMM أو memory، مع بقاء upper physical vector state دون تغيير. نظائر VEX الموجودة مسبقًا تستخدم نفس opcodes بصيغة NDS وتدعم VEX.128/VEX.256.

المراجع: https://www.felixcloutier.com/x86/blendps و https://www.felixcloutier.com/x86/blendpd

## VPBLENDD

مرجع Intel/Felix يعرّف `VPBLENDD` بصيغ `VEX.128/256.66.0F3A.W0 02 /r ib`، مع المصدر الأول من `VEX.vvvv` والمصدر الثاني من ModR/M.r/m، واختيار dword lanes وفق imm8[7:0]. تدعم VEX.128 مصدر XMM أو m128 وتصفّر upper physical state، بينما تدعم VEX.256 مصدر YMM أو m256. لا يورد المرجع صفحة legacy `PBLENDD` مستقلة في هذا المسار؛ لذلك لن يُضاف PBLENDD legacy بالتخمين، وستقتصر هذه الدفعة على `VPBLENDD` الموثق. VEX.W=1 خارج الصيغة.

المرجع: https://www.felixcloutier.com/x86/vpblendd

## MOVNTDQA/VMOVNTDQA

يستخدم `MOVNTDQA` الترميز `66 0F 38 2A /r` لتحميل m128 إلى XMM، وتستخدم `VMOVNTDQA` الترميز `VEX.128/256.66.0F38.WIG 2A /r` لتحميل m128 أو m256 إلى XMM أو YMM، مع `VEX.vvvv=1111` ورفض ModR/M register source. يشترط العتاد محاذاة 16 أو 32 بايت وفق العرض، لكن هذا المنفذ user-mode لا يحاكي #GP أو أنواع الذاكرة/cache؛ يطبق تحميلًا عاديًا مع فحص حدود borrowed memory، ويطبق upper-zeroing في VEX.128/256 كما يلزم للحالة الفيزيائية المدعومة.

المرجع: https://www.felixcloutier.com/x86/movntdqa

## LDDQU/VLDDQU

تستخدم `LDDQU` الترميز `F2 0F F0 /r` لتحميل 128-bit من memory غير المحاذاة إلى XMM، وتستخدم `VLDDQU` ترميز `VEX.128/256.F2.0F.WIG F0 /r` لتحميل 128 أو 256-bit إلى XMM/YMM. في VEX يكون vvvv محجوزًا ويجب أن يساوي 1111، والمصدر memory-only. في هذا المنفذ تُنفذ كتحميلات عادية غير محاذاة داخل borrowed memory؛ لا تُحاكى تفاصيل cache-line protocol أو #AC المحتمل، وتُطبق upper-zeroing لصيغ VEX.128/256 بينما legacy يحافظ على upper state.

المرجع: https://www.felixcloutier.com/x86/lddqu

## MOVNTDQ/VMOVNTDQ

يستخدم `MOVNTDQ` الترميز `66 0F E7 /r` لتخزين XMM إلى m128، وتستخدم `VMOVNTDQ` الترميز `VEX.128/256.66.0F.WIG E7 /r` لتخزين XMM أو YMM إلى m128 أو m256. وجهة ModR/M.r/m memory-only، ومصدر ModR/M.reg، وVEX.vvvv محجوز=1111 وVEX.L يحدد العرض في صيغ VEX. في نموذج user-mode تُنفذ العملية ككتابة borrowed memory عادية؛ لا تُحاكى cache hints أو #GP alignment أو SFENCE/MFENCE ordering، مع فحص الحدود ومنع الكتابة خارج الذاكرة.

المرجع: https://www.felixcloutier.com/x86/movntdq

## VMOVDQA/VMOVDQU

تعرّف مراجع Intel صيغ VEX للتحميل والتخزين vector بعرض 128/256. في `VMOVDQA` تكون صيغة VEX.128/256.66.0F.WIG بالـopcodes 6F للتحميل و7F للتخزين، مع memory alignment مطلوب في العتاد، بينما `VMOVDQU` تستخدم VEX.128/256.F3.0F.WIG بالـopcodes 6F/7F وتسمح بمصدر أو وجهة memory غير محاذاة. في الصيغ VEX يكون vvvv محجوزًا=1111، وتُصفّر الأجزاء العليا وفق طول VEX. هذا المنفذ سيضيف صيغ VEX فقط عند الحاجة، ويعامل alignment/cache كقيود خارج user-mode مع بقاء borrowed bounds إلزامية.

المراجع: https://www.felixcloutier.com/x86/movdqa:vmovdqa32:vmovdqa64 و https://www.felixcloutier.com/x86/movdqu:vmovdqu8:vmovdqu16:vmovdqu32:vmovdqu64

## إصلاح legacy MOVDQA/MOVDQU store ordering

يحدد مرجع Intel/Felix أن `66 0F 6F /r MOVDQA xmm1, xmm2/m128` تحميل، وأن `66 0F 7F /r MOVDQA xmm2/m128, xmm1` تخزين؛ وبالمثل يستخدم `F3 0F 6F/7F` لـ`MOVDQU`. جدول Op/En يضع الوجهة القابلة للكتابة في ModR/M.reg للـ6F، وفي ModR/M.r/m للـ7F. لذلك يجب أن يعكس decoder arguments عند opcode 7F بدل معاملته كتحميل، بينما يظل نموذج المحاكي borrowed-memory approximation ولا يفرض محاذاة #GP.

المرجع: https://www.felixcloutier.com/x86/movdqa:vmovdqa32:vmovdqa64

## VMOVD وVMOVQ

يحدد مرجع Intel/Felix صيغ AVX التالية: `VEX.128.66.0F.W0 6E /r VMOVD xmm1, r32/m32`، و`VEX.128.66.0F.W1 6E /r VMOVQ xmm1, r64/m64`، و`VEX.128.66.0F.W0 7E /r VMOVD r32/m32, xmm1`، و`VEX.128.66.0F.W1 7E /r VMOVQ r64/m64, xmm1`. الصيغ VEX.128 فقط؛ `VEX.L=1` غير صالح لهذه التعليمات. `VEX.vvvv` محجوز ويجب أن يكون 1111. عند النقل إلى XMM، يُكتب الجزء المنخفض 32 أو 64 بت وتُصفّر الحالة العليا؛ وعند النقل إلى GPR يكتب الوجهة scalar بعرض W المناسب. سيبقى النموذج user-mode خاليًا من alignment/cache approximations، مع bounds checking للذاكرة borrowed.

المراجع: https://www.felixcloutier.com/x86/movd:movq و https://uops.info/html-instr/VMOVD_XMM_R32.html

## دفعة MOVQ XMM↔XMM وmemory

يحدد مرجع Intel/Felix صيغ SSE2 `F3 0F 7E /r MOVQ xmm1, xmm2/m64` للتحميل إلى XMM، وصيغة `66 0F D6 /r MOVQ xmm2/m64, xmm1` للتخزين من XMM. في صيغة XMM الوجهة يُنقل low quadword وتُصفّر high quadword من XMM، بينما التخزين إلى memory يكتب 64 بت فقط. صيغ VEX المقابلة مختلفة: `VEX.128.F3.0F.WIG 7E` للـXMM source/destination، و`VEX.128.66.0F.WIG D6` لاتجاه التخزين؛ وكلاهما يصفّر upper physical state فوق 128 بت عند وجهة XMM، وvvvv محجوز وVEX.L=1 غير صالح.

تُستبعد `MOVDQ2Q` و`MOVQ2DQ` من هذه الدفعة لأنهما يتطلبان MMX registers وانتقال x87/MMX state، بينما واجهة المحاكي الحالية لا تحتوي MMX register file ولا تدّعي محاكاة ذلك state. سيُنفذ أولًا نطاق MOVQ XMM↔XMM وmemory فقط، مع borrowed bounds وعدم فرض alignment faults.

المراجع: https://www.felixcloutier.com/x86/movq و https://www.felixcloutier.com/x86/movdq2q

## MOVLPS/MOVHPS وMOVLPD/MOVHPD partial XMM moves

يحدد مرجع Intel/Felix أن `MOVLPS` يستخدم `0F 12 /r` للتحميل من m64 إلى low 64 bits من XMM، و`0F 13 /r` للتخزين من low 64 bits إلى m64. `MOVLPD` يستخدم `66 0F 12 /r` و`66 0F 13 /r` بالسلوك نفسه. هذه legacy forms memory-only؛ لا تُقبل register-to-register أو memory-to-memory. عند load legacy تُحفظ high 64 bits من XMM، وكذلك physical bytes فوق 128 بت.

يستخدم `MOVHPS` و`MOVHPD` opcodes `0F 16/17` و`66 0F 16/17`، ويحمّلان أو يخزّنان high 64 bits مع إبقاء الجزء الآخر وفق المرجع. ستبدأ الدفعة بصيغ legacy memory-only فقط، ولن تُضاف VEX merge forms لأن لها operand ثالثًا VEX.vvvv وسلوكًا مختلفًا يحتاج مسارًا منفصلًا. في نموذج user-mode تُنفذ m64 كـborrowed memory بعرض 8 بايت مع bounds checking، دون ادعاء محاكاة FP exceptions أو alignment faults.

المراجع: https://www.felixcloutier.com/x86/movlps و https://www.felixcloutier.com/x86/movlpd
