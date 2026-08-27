# ملاحظات الإصدار

هذا الإصدار هو **منفذ يدوي جزئي وواسع** من `golang.org/x/arch/x86/x86asm` إلى C99، مع محاكي x86_64 بنمط user-mode. لا يدّعي تكافؤًا كاملًا مع جداول Go الأصلية ولا يدّعي أنه محاكي نظام أو محاكي x86 كامل.

## المكونات

يحتوي `c99/` على decoder وformatter وواجهة المحاكي واختبارات decoder والتنفيذ. يستخدم decoder بادئات `x86asm_` و`X86ASM_`، بينما تستخدم طبقة المحاكاة بادئات `x86emu_`. الذاكرة في `x86emu_memory` borrowed ويملكها المستدعي؛ لا ينفذ المحاكي `malloc` أو `calloc` أو `realloc` أو `free` ولا يحرر الذاكرة ضمنيًا.

يشمل النطاق المتحقق حزمًا واسعة من integer/control/stack/flags/strings/atomics، وSSE2/SSSE3 المتجهية، وAVX/AVX2 integer وpacked floating-point arithmetic المدعومة، بما في ذلك `PSHUFB/VPSHUFB` مع zeroing و128-bit lane semantics، وعمليات packed single/double add/sub/mul/div/min/max، و`CMPPS/CMPPD` و`VCMPPS/VCMPPD` مع predicates ordered/unordered وimm8، و`PTEST/VPTEST` مع flags، إضافة إلى حزم saturation وcompare وshuffle/unpack/pack التي توثقها `PORT_README.md` و`research_notes.md`. أضيفت scalar `ADDSS/SUBSS/MULSS/DIVSS` و`ADDSD/SUBSD/MULSD/DIVSD`، ونقل `MOVSS/MOVSD` مع صيغ VEX register-merge وmemory، و`MINSS/MAXSS/MINSD/MAXSD` مع قواعد NaN وsigned-zero الخاصة بالمصدر الثاني. كما أضيفت `PABSB/PABSW/PABSD` و`VPABSB/VPABSW/VPABSD` على XMM/YMM مع upper-zeroing وreserved-vvvv validation، و`PSIGNB/PSIGNW/PSIGND` و`VPSIGNB/W/D` مع قواعد control الموجب/السالب/الصفر وsource ordering، و`PHADDW/PHADDD/PHADDSW` و`VPHADD*` مع lane-local semantics وsigned-word saturation، و`PHSUBW/PHSUBD/PHSUBSW` و`VPHSUB*` مع اتجاه الطرح الصحيح وsigned-word saturation، و`PMADDUBSW` و`VPMADDUBSW` مع ضرب unsigned/signed bytes وجمع الأزواج وتشبع signed-word، و`PMADDWD` و`VPMADDWD` مع ضرب signed words وجمع الأزواج إلى signed dword وwrap semantics الموثقة، و`PMULDQ` و`VPMULDQ` مع استخدام even signed dwords وإنتاج signed qwords، و`PMOVSXBW/BD/BQ/WD/WQ/SXDQ` و`PMOVZXBW/BD/BQ/WD/WQ/DQ` مع نسخ `VPMOV*` والتحويلات الصحيحة بين byte/word/dword/qword، بما في ذلك memory widths وreserved VEX.vvvv، و`PBLENDVB` و`VPBLENDVB` مع MSB mask من XMM0 الضمني أو mask register المشفر في imm8، وregister/memory tests، و`PBLENDW` و`VPBLENDW` مع word mask من imm8 وlegacy destructive/VEX source ordering وlane-local semantics، و`PALIGNR` و`VPALIGNR` بإزاحة byte عبر concatenation وlane-local VEX.256، و`PHMINPOSUW` و`VPHMINPOSUW` لاختيار unsigned word الأدنى وكتابة القيمة مع أول index، مع اختبارات ties وmemory وupper preservation/zeroing. تبقى صيغ MMX وEVEX masked لهذه العائلات خارج النطاق. شُددت أيضًا بعض reserved VEX fields: `vvvv` في mask extraction و`L` في scalar VEX وhigh imm8 bits في VCMP. لا تشمل دفعة PABS الحالية `PABSQ` أو صيغ EVEX masked/broadcast.

## التحقق

تم تشغيل البناء والاختبارات في 27 أغسطس 2026 بالتكوينين التاليين:

```sh
cd c99
make clean && make check \
  CFLAGS='-std=c99 -Wall -Wextra -Wpedantic -Wconversion -Wshadow -O2'

make clean && make check \
  CFLAGS='-std=c99 -Wall -Wextra -Wpedantic -Wconversion -Wshadow -O1 -g -fsanitize=address,undefined -fno-omit-frame-pointer' \
  LDFLAGS='-fsanitize=address,undefined'
```

نجح `test_x86asm` و`test_x86emu` في الجولتين. بعد التحقق نُظفت نواتج البناء من شجرة الإصدار.

## الحدود المعروفة

لا يضم هذا الإصدار نقلًا آليًا لكل جداول Go، ولا يغطي جميع legacy x86 أو SSE/AVX/AVX-512 أو EVEX. دعم EVEX محدود جدًا. لا يوجد تنفيذ x87 أو MMX واسع. حالات AMD XOP مثل `VPROTB` و`BEXTR` محدودة، و`VPCMOV` decode-only لأن تمثيل operands الحالي لا يمثل صيغة AMD ذات الأربعة operands بصورة سليمة؛ لذلك لا يُنفذ بالتخمين. كما لا توجد FMA4 أو 3DNow أو SVM كاملة، وتبقى التعليمات المفكوكة بلا semantics صريحة غير مدعومة بدل إرجاع نتائج تقريبية.

ينبغي استخدام الإصدار كقاعدة C99 قابلة للمراجعة والتوسعة، لا كبديل drop-in كامل للحزمة الرسمية أو كمحاكي يصلح لتشغيل نظام تشغيل.

## المراجع

- [Intel x86 instruction reference](https://www.felixcloutier.com/x86/)
- [PSHUFB](https://www.felixcloutier.com/x86/pshufb)
- [SUBPS](https://www.felixcloutier.com/x86/subps)
- [MULPS](https://www.felixcloutier.com/x86/mulps)
- [DIVPS](https://www.felixcloutier.com/x86/divps)
- [ADDPD](https://www.felixcloutier.com/x86/addpd)
- [MULSS](https://www.felixcloutier.com/x86/mulss)
- [DIVSS](https://www.felixcloutier.com/x86/divss)
- [MULSD](https://www.felixcloutier.com/x86/mulsd)
- [DIVSD](https://www.felixcloutier.com/x86/divsd)
- [MOVSS](https://www.felixcloutier.com/x86/movss)
- [MOVSD](https://www.felixcloutier.com/x86/movsd)
- [MINSS](https://www.felixcloutier.com/x86/minss)
- [MAXSS](https://www.felixcloutier.com/x86/maxss)
- [MINSD](https://www.felixcloutier.com/x86/minsd)
- [MAXSD](https://www.felixcloutier.com/x86/maxsd)
- [PBLENDW](https://www.felixcloutier.com/x86/pblendw)
- [PALIGNR](https://www.felixcloutier.com/x86/palignr)
- [CMPPS](https://www.felixcloutier.com/x86/cmpps)
- [CMPPD](https://www.felixcloutier.com/x86/cmppd)
- [PTEST](https://www.felixcloutier.com/x86/ptest)
- [AMD64 Architecture Programmer’s Manual](https://www.amd.com/system/files/TechDocs/24593.pdf)


### دفعة PINSR/PEXTR

أضيفت `PINSRB/PINSRW/PINSRD/PINSRQ` و`PEXTRB/PEXTRW/PEXTRD/PEXTRQ` بصيغ legacy، مع نظائر VEX.128 `VPINSR*` و`VPEXTR*`. يدعم decoder مصادر GPR وmemory في الإدخال، ووجهات GPR وmemory في الاستخراج، مع REX.W لصيغ qword، وأقنعة imm8 الصحيحة لكل حجم عنصر. يتحقق VEX decoder من VEX.L=0، ومن VEX.vvvv المحجوز في صيغ VPEXTR، بينما لا يشمل التنفيذ EVEX أو MMX.

تغطي الاختبارات register/memory لكل byte وword وdword وqword، وimm8 masking، وتصفير GPR عند الاستخراج، والحفاظ على upper XMM في legacy ومسحه في VEX.128، وفشل borrowed memory bounds مع بقاء RIP دون تغيير. اجتازت الدفعة strict C99 وبوابة ASan/UBSan دون تحذيرات أو أخطاء.


### دفعة PMULLD/VPMULLD

أضيفت `PMULLD` legacy و`VPMULLD` بصيغ VEX.128 وVEX.256 من خريطة `0F38`, opcode `40`. ينفذ المحاكي ضرب signed dword ويحتفظ بأقل 32 bit من كل حاصل، مع source ordering الصحيح في VEX، وmemory source بالعرض المطابق، وحفظ upper physical state في legacy ومسحه في VEX فوق VL. لا تشمل الدفعة `PMULLQ` أو EVEX masking.

تغطي الاختبارات قيمًا موجبة وسالبة، overflow مع low-product، legacy XMM upper preservation، VEX.128/VEX.256 upper-zeroing، ومصدر memory بعرض 256 bit. اجتازت الدفعة strict C99 وASan/UBSan دون تحذيرات أو أخطاء.


### دفعة legacy PMIN/PMAX SSE4.1

أضيفت صيغ XMM legacy `PMINSB` و`PMAXSB` و`PMINUW` و`PMAXUW` و`PMINSD` و`PMAXSD` و`PMINUD` و`PMAXUD` من خريطة `66 0F 38`, opcodes `38..3F`. ينفذ المحاكي min/max لكل byte أو word أو dword مع signedness الصحيحة، ويترك upper physical XMM state محفوظًا كما تتطلب صيغة legacy. لا تشمل الدفعة MMX أو EVEX.

تغطي الاختبارات signed boundaries، unsigned high-bit values، register وmemory source، وupper preservation. اجتازت الدفعة strict C99 وASan/UBSan دون تحذيرات أو أخطاء.


### دفعة BLENDVPS/BLENDVPD

أضيفت `BLENDVPS` و`BLENDVPD` legacy من خريطة `66 0F 38` مع XMM0 mask ضمني، ونظائر `VBLENDVPS` و`VBLENDVPD` من خريطة `VEX.128/256.66.0F3A` مع mask register من imm8[7:4]. ينفذ المحاكي اختيار dword أو qword lane وفق أعلى bit في mask، ويحافظ على البتات كما هي، ويدعم memory source وVEX upper-zeroing دون تحويل floating-point غير آمن. صيغ EVEX خارج النطاق.

تغطي الاختبارات legacy implicit mask، VEX mask register، single/double، VEX.128/VEX.256، memory source، وlegacy upper preservation. اجتازت الدفعة strict C99 وASan/UBSan دون تحذيرات أو أخطاء.


### دفعة BLENDPS/BLENDPD legacy

أضيفت صيغ `BLENDPS` و`BLENDPD` legacy SSE4.1 من خريطة `66 0F 3A`، مع imm8 lane mask وdestructive destination. يختار `BLENDPS` dword lanes وفق imm8[3:0] ويختار `BLENDPD` qword lanes وفق imm8[1:0]، ويدعم التنفيذ register/memory source مع الحفاظ على upper physical XMM state. لا تشمل الدفعة EVEX.

تغطي الاختبارات register وmemory، masks جزئية، وupper preservation. اجتازت strict C99 وASan/UBSan دون تحذيرات أو أخطاء.


### دفعة VPBLENDD

أضيفت `VPBLENDD` بصيغ VEX.128 وVEX.256 من opcode `66 0F 3A 02`, مع imm8 dword lane mask، المصدر الأول من `VEX.vvvv`، والمصدر الثاني من ModR/M.r/m. يدعم التنفيذ register وmemory source، ويطبق upper-zeroing في VEX.128. لم تُضف صيغة `PBLENDD` legacy بالتخمين لأن المرجع المستخدم لهذه الدفعة يثبت VPBLENDD VEX فقط.

تغطي الاختبارات masks جزئية، register/memory، VEX.128/VEX.256، وupper-zeroing. اجتازت strict C99 وASan/UBSan دون تحذيرات أو أخطاء.


### دفعة MOVNTDQA/VMOVNTDQA

أضيفت `MOVNTDQA` legacy و`VMOVNTDQA` VEX.128/VEX.256 كتحميلات memory-only بعرض 128 أو 256 بت. يرفض decoder register source وVEX.vvvv غير المحجوز، ويطبق executor تحميلًا عاديًا في نموذج user-mode مع حدود borrowed memory؛ لا يحاكي cache protocol أو memory types أو #GP alignment. تُحفظ upper state في legacy وتُصفّر في VEX.

تغطي الاختبارات 128/256-bit loads، RAX-based memory، upper preservation/zeroing، وفشل حدود الذاكرة المستعارة. اجتازت الدفعة strict C99 وASan/UBSan دون تحذيرات أو أخطاء.


### دفعة LDDQU/VLDDQU

أضيفت `LDDQU` legacy و`VLDDQU` بصيغ VEX.128/VEX.256 كتحميلات memory-only غير محاذاة بعرض 128 أو 256 بت. يفحص decoder مصدر الذاكرة وVEX.vvvv المحجوز، وينفذ المحاكي التحميل كعملية عادية داخل borrowed memory؛ لا يحاكي cache-line protocol أو #AC أو السلوك الميكروي الإضافي. يحافظ legacy على upper XMM ويطبق VEX upper-zeroing.

تغطي الاختبارات العناوين غير المحاذاة، RAX addressing، صيغ 128/256-bit، upper preservation/zeroing، وفشل حدود الذاكرة المستعارة. اجتازت الدفعة strict C99 وASan/UBSan دون تحذيرات أو أخطاء.


### دفعة MOVNTDQ/VMOVNTDQ

أضيفت `MOVNTDQ` legacy و`VMOVNTDQ` بصيغ VEX.128/VEX.256 كـstores memory-only بعرض 128 أو 256 بت. يطبق decoder source vector register وdestination memory، ويجري التحقق من VEX.vvvv المحجوز وVEX.L. في نموذج user-mode ينفذ المحاكي الكتابة كعملية borrowed-memory عادية؛ لا يحاكي cache hints أو ordering fences أو #GP alignment.

تغطي الاختبارات register source، RAX-based memory، VEX.128/VEX.256، والتحقق من فشل bounds دون تغيير RIP. اجتازت الدفعة strict C99 وASan/UBSan دون تحذيرات أو أخطاء.


### دفعة VMOVDQA/VMOVDQU

أضيفت `VMOVDQA` و`VMOVDQU` بصيغ VEX.128/VEX.256، مع opcode `6F` للتحميل و`7F` للتخزين وترميز `VEX.pp` الصحيح (`66` للأولى و`F3` للثانية). يدعم decoder register/memory ordering الصحيح ويتحقق من `VEX.vvvv=1111` المحجوز. ينفذ المحاكي النقل عبر borrowed memory، ويطبق VEX upper-zeroing فوق VL، من دون ادعاء محاكاة alignment faults أو cache protocol.

تغطي الاختبارات register moves، VEX.128/VEX.256 loads وstores، source ordering، upper-zeroing، عدم الكتابة خارج عرض store، وفشل borrowed bounds مع بقاء RIP. اجتازت الدفعة strict C99 وASan/UBSan دون تحذيرات أو أخطاء، ثم نُظفت نواتج البناء. لا يمثل هذا السجل إصدار release جديدًا.


### إصلاح MOVDQA/MOVDQU legacy memory stores

صُحح decoder لصيغ `MOVDQA` و`MOVDQU` legacy ذات opcode `7F` كي يضع الوجهة في ModR/M.r/m والمصدر في ModR/M.reg، كما يحدد Intel. أضيفت اختبارات decoder وexecutor للـloads والـstores بعرض 128 بت، بما في ذلك `MOVDQU` بعناوين غير محاذاة، preservation للـupper XMM، وفشل borrowed bounds مع بقاء RIP دون تغيير. لا يفرض نموذج user-mode alignment fault اصطناعيًا على `MOVDQA`.

اجتازت الدفعة strict C99 وASan/UBSan دون تحذيرات أو أخطاء، ثم نُظفت نواتج البناء. هذا تحديث تغطية وليس رقم إصدار أو archive جديدًا.


### دفعة VMOVD/VMOVQ

أضيفت `VMOVD` و`VMOVQ` بصيغ VEX.128 بين GPR أو memory وXMM، باستخدام opcode `6E` للنقل إلى XMM و`7E` للنقل من XMM. يختار `VEX.W=0/1` عرض 32/64 بت، ويُرفض `VEX.L=1` و`VEX.vvvv` غير المحجوز، مع دعم VEX.B/R للسجلات الممتدة. ينفذ المحاكي zero-extension إلى XMM مع upper-zeroing فوق 128 بت، وعمليات GPR بعرضها الصحيح، ضمن borrowed memory bounds ودون allocators.

تغطي اختبارات decoder/executor الاتجاهين، memory widths، VEX.W/L/vvvv، R8D/R8 وXMM9، upper physical state، وفشل bounds مع بقاء RIP دون تغيير. اجتازت الدفعة strict C99 وASan/UBSan دون تحذيرات أو أخطاء، ثم نُظفت نواتج البناء. هذا تحديث تغطية وليس release أو archive جديدًا.


### دفعة MOVQ/VMOVQ XMM-form

أضيفت صيغ `MOVQ` legacy `F3 0F 7E` و`66 0F D6` للتعامل بين XMM وXMM أو m64، وصيغ `VMOVQ` VEX.128 المقابلة. يطبق executor نقل low 64 bits، وتصفير high 64 bits عند وجهة XMM، مع preservation للـphysical upper bytes في legacy وupper-zeroing في VEX. أضيفت memory width وoperand ordering وVEX.L/vvvv validation المناسبة.

تغطي الاختبارات decoder/executor register وmemory load/store، source ordering، aligned/borrowed bounds، legacy upper preservation، VEX upper-zeroing، وعدم تغيير RIP عند فشل الذاكرة. بقيت `MOVDQ2Q` و`MOVQ2DQ` خارج النطاق لاعتمادهما على MMX/x87 state غير الموجود في المحاكي. اجتازت الدفعة strict C99 وASan/UBSan، ثم نُظفت نواتج البناء. لا يمثل هذا السجل release أو archive جديدًا.


### دفعة partial XMM moves

أضيفت صيغ `MOVLPS` و`MOVHPS` و`MOVLPD` و`MOVHPD` legacy memory-only بعرض 64 بت. يدعم decoder opcodes `12/13/16/17` مع prefix `66` عند الحاجة، ويرفض register-to-register. ينفذ executor تحميل أو تخزين low/high quadword مع الحفاظ على الجزء غير المنقول والـphysical upper state في XMM، ويستخدم borrowed memory bounds دون محاكاة alignment أو FP exceptions.

تغطي الاختبارات decoder/executor للعائلات الأربع، اتجاهات load/store، preservation للأجزاء الأخرى، الكتابة بعرض 8 بايت، وفشل bounds مع بقاء RIP. لم تُضف VEX merge forms في هذه الدفعة لاختلاف operand layout وسلوك merge. اجتازت الدفعة strict C99 وASan/UBSan، ثم نُظفت نواتج البناء. لا يمثل هذا السجل release أو archive جديدًا.


### دفعة VEX partial XMM merge

أضيفت صيغ `VMOVLPS` و`VMOVHPS` و`VMOVLPD` و`VMOVHPD` VEX.128. تدعم loads ثلاثة operands مع source XMM من `VEX.vvvv` وm64، وتدعم stores memory-only مع `VEX.vvvv=1111`. يطبق decoder التحقق من VEX.L وترتيب operands، ويرفض register-to-register وvvvv غير المحجوز في stores.

ينفذ executor دمج low/high 64-bit مع source XMM، ويصفر الحالة الفيزيائية فوق 128 بت عند destination XMM، بينما تخزن stores ثمانية بايت فقط. تغطي الاختبارات العائلات الأربع والـmerge source ordering وupper-zeroing وL/vvvv/mod validation وborrowed bounds. اجتازت الدفعة strict C99 وASan/UBSan، ثم نُظفت نواتج البناء. لا يمثل هذا السجل release أو archive جديدًا.
