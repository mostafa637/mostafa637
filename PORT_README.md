# x86asm-c99

منفذ يدوي ونظيف لجزء عملي من واجهة `x86asm` إلى **C99** مستقل. بما أن لغة C لا تملك مساحات أسماء، تبدأ جميع الرموز العامة بالبادئة `x86asm_` أو `X86ASM_` لتفادي تعارض الأسماء.

> المصدر المرجعي هو حزمة `golang.org/x/arch/x86/x86asm` من مستودع [golang/arch](https://github.com/golang/arch/tree/master/x86/x86asm)، وهي حزمة لفك ترميز تعليمات x86 وتدعم أوضاع 16 و32 و64 بت وفق توثيقها الرسمي على [pkg.go.dev](https://pkg.go.dev/golang.org/x/arch/x86/x86asm).

## ما الذي نُقل يدوياً؟

أُعيد تصميم نموذج البيانات بدلاً من محاكاة واجهة Go. يمثّل `x86asm_argument` الوسيط كاتحاد C99 صريحاً بأنواع register وmemory وimmediate وrelative، وتم فصل القراءة متعددة البايتات، فك ModR/M وSIB، اختيار السجلات، معالجة البادئات، وفئة التعليمات في دوال قصيرة قابلة للمراجعة.

يدعم المنفذ الحالي فك ترميز مجموعة موسعة في أوضاع 16 و32 و64 بت، تشمل `MOV`, `LEA`, `ADD`, `ADC`, `SUB`, `SBB`, `AND`, `OR`, `XOR`, `CMP`, `TEST`, `IMUL`, `MUL`, `DIV`, `IDIV`, `NEG`, `NOT`, `INC`, `DEC`, `MOVSX`, `MOVZX`, `PUSH`, `POP`, `CALL`, `JMP`, `RET`, القفزات الشرطية و`LOOP` و`JCXZ`، `INT`, `NOP`, `XADD`, `XCHG`, `BSWAP`, `SHL`, `SHR`, `SAR`, `SHLD`, `SHRD`, `ROL`, `ROR`, `RCL`, `RCR`، وتعليمات الأعلام الأساسية. كما يدعم REX.R/X/B، ModR/M وSIB وaddress-size override وRIP-relative memory، مع دعم يدوي أساسي لـ VEX وEVEX. تشمل مجموعة AVX/AVX2 المنفذة `VMOVUPS` و`VMOVUPD` و`VADDPS` و`VADDPD` و`VSUBPS` و`VMULPS` و`VDIVPS` و`VSUBPD` و`VMULPD` و`VDIVPD`، و`VPADD*` و`VPSUB*` الصحيحة، وعمليات saturation `VPADDUSB/USW/SB/SW` و`VPSUBUSB/USW/SB/SW`، و`VPMINUB` و`VPMAXUB` و`VPMINSW` و`VPMAXSW` و`VPMINSB` و`VPMAXSB` و`VPMINUW` و`VPMAXUW` و`VPMINSD` و`VPMAXSD` و`VPMINUD` و`VPMAXUD` و`VPAVGB` و`VPAVGW` و`VPSADBW`، إضافة إلى `VPSHUFB` بعزل 128-bit lanes وzeroing عند control bit7، و`VPSHUFD` و`VPSHUFLW` و`VPSHUFHW` بإعادة ترتيب مستقلة لكل 128-bit lane، و`VPUNPCK*` و`VPACKSSWB` و`VPACKSSDW` و`VPACKUSWB`، و`VBLENDPS/PD` و`VPCMPEQ*` و`PCMPGTB/W/D` و`VPCMPGTB/W/D/Q` signed وعمليات المنطق والإزاحة والضرب المتجهية المدعومة. توجد خريطة EVEX محدودة جدًا، مثل `VPADDD` بعرض ZMM، ولا تعني وجود دعم عام لـ AVX-512. يحفظ المحاكي سجلات XMM/YMM/ZMM. أما AMD XOP فـ`VPROTB` و`BEXTR` decode-only جزئيًا، و`VPCMOV` ممثل حاليًا في decoder فقط بسبب نموذج operands غير المكتمل، ولا يُنفذ بالتخمين. وتشمل طبقة SSE2/SSSE3 legacy المنفذة `MOVD`, `MOVQ`, `MOVDQA`, `MOVDQU`, `MOVUPD`, `ADDPS`, `SUBPS`, `MULPS`, `DIVPS`, `ADDPD`, `SUBPD`, `MULPD`, `DIVPD`, `PADD*`, `PSUB*`, `PAND`, `POR`, `PXOR`, `PSLL*`, `PSRL*`, `PSRA*`, `PSLLDQ`, `PSRLDQ`, `PCMPEQ*`, `PMULLW`, `PMULHW`, `PMULHUW`, `PMULUDQ`, وعمليات saturation/min-max/average/`PSADBW`، و`PSHUFD` و`PSHUFLW` و`PSHUFHW` و`PUNPCK*` و`PACKSSWB` و`PACKSSDW` و`PACKUSWB` و`PSHUFB` و`PABSB/PABSW/PABSD` مع نسخ `VPABS*` XMM/YMM، و`PSIGNB/PSIGNW/PSIGND` مع نسخ `VPSIGN*` XMM/YMM، و`PHADDW/PHADDD/PHADDSW` مع نسخ `VPHADD*` XMM/YMM وجمع lane-local وsigned-word saturation، و`PHSUBW/PHSUBD/PHSUBSW` مع نسخ `VPHSUB*` XMM/YMM وطرح lane-local وsigned-word saturation، و`PMADDUBSW` مع `VPMADDUBSW` لضرب unsigned/signed bytes وجمع الأزواج مع signed-word saturation، و`PMADDWD` مع `VPMADDWD` لضرب signed words وجمع كل زوج إلى signed dword مع wrap semantics الموثقة، و`PMULDQ` مع `VPMULDQ` لضرب even signed dwords وإنتاج signed qwords مع صيغ XMM/YMM وmemory، و`PMOVSXBW/BD/BQ/WD/WQ/SXDQ` و`PMOVZXBW/BD/BQ/WD/WQ/DQ` مع نسخ `VPMOV*` على XMM/YMM، وsign/zero extension للأحجام الستة مع memory widths الدقيقة وreserved VEX.vvvv، و`PBLENDVB` و`VPBLENDVB` بانتقاء bytes وفق MSB لقناع XMM0 الضمني أو mask register المشفر في imm8، و`PBLENDW` و`VPBLENDW` باختيار words وفق bits imm8، مع destructive legacy وsource ordering وlane-local VEX، و`PALIGNR` و`VPALIGNR` بإزاحة byte عبر concatenation، مع تكرار العملية داخل كل 128-bit lane في VEX.256، و`PHMINPOSUW` و`VPHMINPOSUW` لاختيار أصغر unsigned word وكتابة القيمة مع أول index في destination، إضافة إلى `VPCMPGTB/W/D/Q` signed بعروضها المتجهية المدعومة، و`MOVMSKPS` و`MOVMSKPD` و`PMOVMSKB` بصيغ legacy و`VMOVMSKPS` و`VMOVMSKPD` و`VPMOVMSKB` بصيغ VEX.128/256 لاستخراج sign masks وbyte masks إلى GPR، و`PTEST`/`VPTEST` مع تحديث ZF وCF ومسح الأعلام الحسابية المحددة، و`MINPS`/`MAXPS` و`MINPD`/`MAXPD` ونظائر VEX `VMINPS`/`VMAXPS` و`VMINPD`/`VMAXPD` مع معالجة source2 عند NaN واختيار source2 عند تساوي الصفرين، و`CMPPS`/`CMPPD` ونظائر VEX `VCMPPS`/`VCMPPD` مع imm8 predicates ordered/unordered ونتائج masks packed، وتعليمات scalar `ADDSS`/`SUBSS`/`MULSS`/`DIVSS`/`ADDSD`/`SUBSD`/`MULSD`/`DIVSD`، ونقل `MOVSS`/`MOVSD` scalar مع صيغ VMOVSS/VMOVSD register-merge وmemory load/store، وتعليمات scalar `MINSS`/`MAXSS`/`MINSD`/`MAXSD` ونظائر VEX، مع قاعدة Intel التي تجعل المصدر الثاني هو الناتج عند NaN أو تساوي ±0. تحافظ legacy register moves على بقية XMM، وتمسح memory loads الجزء الأعلى من XMM، بينما تنفذ صيغ VEX source merge وupper-zeroing في النموذج الفيزيائي. جميع قراءات وكتابات scalar memory تمر عبر borrowed memory التي يملكها المستدعي. مسار VEX.128 scalar يمسح أيضًا الحالة الفيزيائية فوق 128 بت. كما نُفذت `CMPXCHG8B` و`CMPXCHG16B` مع المسارات الذرية الممثلة داخل نموذج user-mode.

تدعم دفعة PBLENDW صيغ SSE4.1 legacy وAVX/AVX2 VEX.128/VEX.256، وتكرر قناع imm8 على كل 128-bit lane في VEX.256. وتدعم دفعة PALIGNR صيغ SSSE3 legacy وAVX/AVX2 VEX.128/VEX.256، مع zero عند imm8 أكبر من نافذة 32 byte وzeroing للحالة الفيزيائية العليا في صيغ VEX. لا تشمل دفعة PABS الحالية `PABSQ` أو صيغ EVEX masked/broadcast؛ كما لا تشمل دفعة PSIGN صيغ MMX أو EVEX، ولا تشمل دفعة PHADD أو PHSUB أو PMADDUBSW صيغ MMX أو EVEX. تظل هذه خارج النطاق المنفذ حتى تتوفر معالجة EVEX وopmask/MMX كاملة.

هذا الإصدار **ليس ترجمة آلية كاملة لكل 1.1 MB من حزمة Go**، ولا يدّعي تغطية كل تعليمات AVX/AVX-512 أو كل تعليمات AMD القديمة مثل XOP/FMA4/3DNow. تغطية decoder الكاملة تتطلب نقل الجداول المولدة الكبيرة وخوارزميات AVX/EVEX وAMD كاملة، مع اختبارات مطابقة واسعة. عند الحاجة إلى تكافؤ شامل، يجب توسيع `x86asm_opcode` والجداول واختبارات المطابقة قبل استخدامه كبديل كامل للحزمة الأصلية.

## البناء

```sh
cd c99
make check
```

يتطلب البناء مترجم C يدعم C99 و`make`. يمكن تغيير المترجم أو الخيارات هكذا:

```sh
make CC=gcc CFLAGS='-std=c99 -Wall -Wextra -Wpedantic -O2' check

ولفحص الذاكرة والسلوك غير المعرّف:

```sh
make clean
make check CFLAGS='-std=c99 -Wall -Wextra -Wpedantic -O1 -g -fsanitize=address,undefined -fno-omit-frame-pointer' LDFLAGS='-fsanitize=address,undefined'
```

## مثال استخدام

```c
#include "x86asm.h"
#include <stdio.h>

int main(void)
{
    const uint8_t code[] = { 0x48, 0x8B, 0x44, 0x8B, 0x10 };
    x86asm_instruction instruction;
    char text[128];

    x86asm_error error = x86asm_decode(code, sizeof(code), 64, &instruction);
    if (error != X86ASM_OK) {
        fprintf(stderr, "decode failed: %s\n", x86asm_error_string(error));
        return 1;
    }

    x86asm_format_default(&instruction, text, sizeof(text));
    printf("%s\n", text);
    return 0;
}
```

الناتج المتوقع هو:

```text
48 mov rax, [rbx+4*rcx+0x10]
```

## محاكي user-mode

يحتوي `x86emu.h` و`x86emu.c` على نواة محاكي user-mode مرتبطة بالـ decoder. يحتفظ المحاكي بمؤشر إلى buffer الذاكرة وحجمه فقط؛ **لا يخصص الذاكرة ولا يحررها ولا يملكها**. يجب على المستدعي إبقاء buffer صالحًا طوال عمر CPU وتحريره بنفسه بعد انتهاء الاستخدام. يدعم المحاكي حاليًا السجلات العامة، RIP وRFLAGS، stack، الذاكرة المستعارة، نقاط التوقف، CALL/RET، الفروع، الحساب الصحيح، الأعلام، السلاسل مع REP/DF، الذرية الثنائية، وعمليات SIMD على XMM/YMM ضمن القائمة الموثقة، مع lane semantics صريحة لعمليات AVX2 التي تعمل على 128-bit lanes. لا يوجد تنفيذ واسع لـ x87/MMX أو AVX-512. تعليمات syscall/sysenter والمسارات privileged لا تُحاكى بصلاحيات وهمية؛ تمر عبر hook أو تعيد خطأً صريحًا. أما التعليمات التي يفكها decoder ولا تملك semantics منفذة بعد فتعيد `X86EMU_ERR_UNSUPPORTED` بدل نتيجة تقريبية.

## ملاحظات التصميم

الدوال التي تكتب نصاً تعيد الطول المطلوب دون احتساب المحرف الصفري، وتقبل حجماً صفرياً للاستعلام عن الطول. لا تعتمد المكتبة على Go أو `libopcodes` أو Capstone أو أي مكتبة خارجية. كما أن فك الترميز لا يستخدم مؤشرات عامة قابلة للتعديل، وتُصفّر بنية النتيجة قبل بدء العملية.

## الترخيص والمصدر

يرتبط الكود الأصلي بترخيص BSD-3-Clause الخاص بمشروع Go. يجب مراجعة ملف الترخيص الأصلي قبل إعادة توزيع نسخة موسعة أو إضافة جداول من المصدر الأصلي. هذه النسخة اليدوية تحتفظ بمرجع المصدر وتفصل بوضوح بين المنطق المعاد تصميمه والبيانات التي لم تُنقل بعد.

## المراجع

[1]: https://github.com/golang/arch/tree/master/x86/x86asm "المستودع الرسمي لملفات x86asm"
[2]: https://pkg.go.dev/golang.org/x/arch/x86/x86asm "توثيق حزمة x86asm"


## دفعة PINSR وPEXTR

أضيفت صيغ SSE4.1 legacy وAVX VEX.128 لـ`PINSRB/PINSRW/PINSRD/PINSRQ` و`PEXTRB/PEXTRW/PEXTRD/PEXTRQ` مع namespace العام المعتاد. تختار تعليمات الإدخال العنصر من imm8، وتقرأ المصدر scalar من GPR أو memory، بينما تختار تعليمات الاستخراج العنصر من XMM وتكتب إلى GPR أو memory. تُطبَّق أقنعة index الصحيحة للـbyte والـword والـdword والـqword، وتُصفّر الوجهة GPR عند الاستخراج، وتبقى upper XMM الفيزيائية محفوظة في legacy ويمسحها VEX.128 في الإدخال. صيغ EVEX وMMX غير مشمولة، و`VPINSR*` و`VPEXTR*` لا تسمح بـVEX.L=1؛ كما يتحقق decoder من أن VEX.vvvv محجوز في صيغ الاستخراج.

اختبرت الدفعة بصيغ register وmemory وبأحجام العناصر الأربعة، مع imm8 values تتجاوز عدد العناصر للتحقق من masking، واختبار preservation/zeroing للحالة الفيزيائية، وحالة borrowed memory ناقصة تعيد `X86EMU_ERR_MEMORY` دون كتابة خارج المجال. اجتازت بوابة strict وبوابة ASan/UBSan.


## دفعة PMULLD

أضيفت `PMULLD` legacy و`VPMULLD` بصيغ VEX.128/VEX.256. تضرب كل صيغة زوج signed dword متناظرًا، وتكتب أقل 32 bit من حاصل الضرب لكل عنصر؛ لا تُدّعى هنا تغطية `PMULLQ` أو EVEX masking. تحافظ صيغة legacy على upper physical vector state، بينما تصفّر صيغ VEX الجزء الأعلى فوق 128 أو 256 bit. يدعم decoder مصدر register أو memory بالعرض الصحيح، وتغطي الاختبارات نتائج موجبة وسالبة وحالات overflow/low-product وmemory وupper preservation/zeroing.

اجتازت دفعة PMULLD بوابة strict C99 وبوابة ASan/UBSan، مع بقاء الذاكرة borrowed وعدم إضافة أي allocator إلى المحاكي.


## صيغ legacy PMIN/PMAX الإضافية

أضيفت صيغ SSE4.1 legacy الناقصة `PMINSB` و`PMAXSB` و`PMINUW` و`PMAXUW` و`PMINSD` و`PMAXSD` و`PMINUD` و`PMAXUD` من خريطة `66 0F 38`. يقارن التنفيذ كل عنصر بعرضه الصحيح، ويفصل بدقة بين signed وunsigned، ويحتفظ بالنتيجة في XMM destination مع بقاء upper physical vector state دون تغيير. لا تشمل هذه الدفعة MMX أو EVEX؛ نظائر VEX لهذه العائلة كانت موجودة مسبقًا.

تغطي الاختبارات الحدود السالبة والموجبة، حالات signed/unsigned التي تختلف فيها النتيجة، register وmemory operands، والحفاظ على upper XMM. اجتازت الدفعة strict C99 وASan/UBSan.


## دفعة BLENDVPS وBLENDVPD

أضيفت `BLENDVPS` و`BLENDVPD` بصيغ legacy، مع `VBLENDVPS` و`VBLENDVPD` بصيغ VEX.128/VEX.256. تستخدم الصيغ legacy XMM0 ضمنيًا كقناع destructive، بينما تستخدم صيغ VEX المصدر الأول من `VEX.vvvv` والمصدر الثاني من ModR/M.r/m، ويحدد imm8[7:4] سجل القناع؛ النصف الأدنى من imm8 مهمل. يختار التنفيذ dword lanes في single وqword lanes في double وفق bit7 من العنصر المقابل في mask، وينسخ البتات كما هي دون type-punning float. يحفظ legacy upper state ويمسح VEX الجزء الأعلى فوق VL، ولا تشمل الدفعة EVEX.

تغطي الاختبارات mask XMM0 implicit، mask register صريحًا، single/double، VEX.128/VEX.256، ومصدر memory، وقد اجتازت strict C99 وASan/UBSan.


## BLENDPS وBLENDPD legacy

أضيفت صيغ SSE4.1 legacy `BLENDPS` و`BLENDPD` من خريطة `66 0F 3A`، مع destructive destination وimm8 lane mask. تختار `BLENDPS` عناصر dword الأربعة وفق imm8[3:0]، وتختار `BLENDPD` عنصري qword وفق imm8[1:0]؛ يدعم decoder مصدر XMM أو memory بعرض 128-bit، ويحافظ executor على upper physical state في صيغ legacy. نظائر VEX immediate كانت موجودة في المنفذ، ولم تُضف صيغ EVEX.

تغطي الاختبارات masks جزئية، register وmemory source، واختبارًا صريحًا للحفاظ على upper XMM. اجتازت الدفعة strict C99 وASan/UBSan.


## VPBLENDD

أضيفت `VPBLENDD` بصيغ VEX.128 وVEX.256 من خريطة `66 0F 3A 02`، مع المصدر الأول من `VEX.vvvv` والمصدر الثاني من ModR/M.r/m، واختيار dword lanes وفق imm8[7:0]. يدعم التنفيذ مصادر register وmemory بعرض 128 أو 256 بت، ويصفّر upper physical state عند VEX.128 ويطبق source ordering غير الهدمي. لم تُضف `PBLENDD` legacy لأن المرجع المستخدم لهذه الدفعة يثبت صيغ VPBLENDD VEX فقط؛ لا يوجد ادعاء تغطية صيغ غير موثقة بالتخمين.

تغطي الاختبارات VEX.128 وVEX.256، masks جزئية، memory source، وupper-zeroing، وقد اجتازت strict C99 وASan/UBSan.


## MOVNTDQA وVMOVNTDQA

أضيفت `MOVNTDQA` legacy لتحميل m128 إلى XMM، و`VMOVNTDQA` بصيغ VEX.128/VEX.256 لتحميل m128 أو m256 إلى XMM أو YMM. يتطلب decoder مصدر memory فقط ويرفض register source وVEX.vvvv غير المحجوز، بينما ينفذ المحاكي في user-mode تحميلًا عاديًا لأن cache protocol وmemory types و#GP ليست جزءًا من نموذج المحاكاة. تُفحص حدود borrowed memory، ويحافظ legacy على upper state ويطبق VEX upper-zeroing.

محاذاة العناوين المطلوبة في العتاد موثقة كقيد خارج نموذج هذا المنفذ؛ لا يدّعي التنفيذ محاكاة أخطاء المحاذاة العتادية. تغطي الاختبارات 128/256-bit memory loads، RAX addressing، upper preservation/zeroing، وفشل الذاكرة المستعارة الناقصة. اجتازت الدفعة strict C99 وASan/UBSan.


## LDDQU وVLDDQU

أضيفت `LDDQU` legacy بالترميز `F2 0F F0 /r` و`VLDDQU` بصيغ VEX.128/VEX.256 بالترميز `F2 0F F0 /r`. التعليمات memory-only وتقرأ 16 أو 32 بايتًا من borrowed memory دون اشتراط محاذاة في نموذج المنفذ، مع فحص حدود الذاكرة. ينفذ user-mode المحاكي هذه التعليمات كتحميلات عادية غير محاذاة؛ لا يحاكي تفاصيل cache-line protocol أو الازدواج المحتمل للقراءات أو #AC. يحافظ legacy على upper XMM state، بينما يطبق VEX upper-zeroing للحالة الفيزيائية.

تغطي الاختبارات عناوين غير محاذاة، صيغ 128/256-bit، RAX addressing، upper preservation/zeroing، وفشل الذاكرة المستعارة الناقصة. اجتازت الدفعة strict C99 وASan/UBSan.


## MOVNTDQ وVMOVNTDQ

أضيفت `MOVNTDQ` legacy بالترميز `66 0F E7 /r` و`VMOVNTDQ` بصيغ VEX.128/VEX.256 لتخزين XMM أو YMM في borrowed memory بعرض 128 أو 256 بت. يفرض decoder destination memory-only وsource vector register، ويرفض VEX.vvvv غير المحجوز ويطبق VEX.L لاختيار العرض. ينفذ user-mode المحاكي الكتابة كعملية عادية؛ لا يحاكي cache hints أو SFENCE/MFENCE ordering أو #GP alignment، لكنه يفحص حدود borrowed memory ولا يكتب خارجها.

تغطي الاختبارات register source، RAX addressing، 128/256-bit stores، وفشل حدود الذاكرة المستعارة. اجتازت الدفعة strict C99 وASan/UBSan.


## VMOVDQA وVMOVDQU

أضيفت `VMOVDQA` و`VMOVDQU` بصيغ VEX.128 وVEX.256 من خريطة `0F`، حيث يستخدم `66` صيغة aligned لـ`VMOVDQA` ويستخدم `F3` صيغة unaligned لـ`VMOVDQU`، مع opcode `6F` للتحميل و`7F` للتخزين. يدعم decoder register وmemory operands مع ترتيب الوجهة/المصدر الصحيح، ويتحقق من أن `VEX.vvvv=1111` لأنه حقل محجوز في هذه الصيغ.

ينفذ المحاكي النقل عبر helpers المتجه والذاكرة المستعارة، ويصفّر الحالة الفيزيائية فوق 128 أو 256 بت في صيغ VEX.128/VEX.256. لا يحاكي نموذج user-mode محاذاة العتاد أو #GP/#AC أو cache protocol؛ لذلك لا تُحوّل صفة aligned في `VMOVDQA` إلى فحص محاذاة اصطناعي، مع بقاء فحص bounds للذاكرة إلزاميًا. تغطي الاختبارات register moves، memory loads/stores، VEX.128/VEX.256، upper-zeroing، وعدم الكتابة خارج 128-bit store، وفشل borrowed-memory bounds. اجتازت الدفعة strict C99 وASan/UBSan.


## إصلاح MOVDQA وMOVDQU legacy memory stores

صُحح مسار `MOVDQA` و`MOVDQU` legacy عند opcode `7F`: أصبحت الوجهة ModR/M.r/m والمصدر ModR/M.reg كما في جدول Intel، بدل تفسير store كأنه load. يدعم المسار الآن اختبارات load وstore من/إلى XMM وذاكرة borrowed بعرض 128 بت، مع الحفاظ على upper physical XMM state في الصيغ legacy. يظل `MOVDQU` مناسبًا لعناوين غير محاذاة داخل نموذج user-mode، بينما لا يفرض المنفذ خطأ محاذاة اصطناعيًا على `MOVDQA`; فحص bounds مستقل وإلزامي.

تغطي الاختبارات `MOVDQA` و`MOVDQU` memory loads وstores، عناوين aligned وunaligned، preservation للجزء الأعلى من XMM، وعدم تغيير RIP عند فشل borrowed-memory bounds. اجتازت الدفعة strict C99 وASan/UBSan.


## VMOVD وVMOVQ

أضيفت صيغ AVX `VMOVD` و`VMOVQ` بصيغة VEX.128 بين GPR أو memory وXMM. يستخدم `VMOVD` الترميز `VEX.128.66.0F.W0 6E/7E` بعرض 32 بت، بينما يستخدم `VMOVQ` الترميز `VEX.128.66.0F.W1 6E/7E` بعرض 64 بت. يدعم opcode `6E` النقل إلى XMM وopcode `7E` النقل من XMM، مع register وmemory operands، وسجلات GPR الممتدة التي يحددها VEX.B/R. يرفض decoder `VEX.L=1` و`VEX.vvvv` غير المحجوز، ويرفض VMOVQ في mode غير 64-bit.

ينفذ المحاكي zero-extension إلى الجزء المنخفض من XMM ثم يصفر الحالة الفيزيائية فوق 128 بت عند النقل إلى XMM، ويكتب GPR بعرض 32 أو 64 بت عند النقل العكسي. تستخدم memory operands الذاكرة borrowed مع فحص bounds؛ لا توجد ownership أو allocator داخل المحاكي. تغطي الاختبارات الاتجاهين، memory بعرض 32/64 بت، VEX.W/L/vvvv، R8D/R8 وXMM9، upper-zeroing، وعدم تغيير RIP عند فشل الحدود. اجتازت الدفعة strict C99 وASan/UBSan.


## MOVQ وVMOVQ بين XMM والذاكرة

أضيفت صيغ `MOVQ` legacy ذات XMM operands: `F3 0F 7E /r` للتحميل من XMM أو m64 إلى XMM، و`66 0F D6 /r` للتخزين من XMM إلى XMM أو m64. كما أضيفت صيغ `VMOVQ` VEX.128 المقابلة: `F3 0F 7E` و`66 0F D6`. ينقل المسار low 64 bits، ويصفر high 64 bits من XMM عند وجهة XMM؛ في legacy تبقى physical bytes فوق 128 بت محفوظة، بينما VEX يصفرها فوق 128 بت.

يدعم decoder ترتيب operands الصحيح والـmemory width البالغ 8 بايت، ويتحقق من VEX.L وVEX.vvvv في صيغ VEX. تدعم الاختبارات register/memory load/store، source ordering، preservation legacy، upper-zeroing VEX، وفشل borrowed bounds دون تغيير RIP. لا يشمل هذا التحديث `MOVDQ2Q` أو `MOVQ2DQ` لأنهما يتطلبان MMX register file وx87/MMX state غير الموجودين في نموذج المحاكي، ولا تُدّعى محاكاتهما.


## MOVLPS/MOVHPS وMOVLPD/MOVHPD

أضيفت صيغ legacy memory-only الأربع لنقل نصف XMM بعرض 64 بت: `MOVLPS` (`0F 12/13`)، `MOVHPS` (`0F 16/17`)، `MOVLPD` (`66 0F 12/13`) و`MOVHPD` (`66 0F 16/17`). يحمّل opcode الأول من m64 إلى low أو high quadword، بينما يخزّن opcode الثاني نصف XMM المحدد إلى m64. يرفض decoder register-to-register لهذه الصيغ، لأن المرجع يعرّفها memory-only.

تحافظ loads legacy على النصف الآخر من XMM وعلى physical bytes فوق 128 بت، وتكتب stores ثمانية بايت فقط. الذاكرة borrowed مع فحص bounds، ولا تُفرض محاذاة أو floating-point exception model اصطناعيًا. لم تُضف صيغ VEX merge لهذه العائلة بعد، لأنها تحتاج operand VEX.vvvv إضافيًا وسلوك merge مختلفًا عن legacy.
