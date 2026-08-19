# خطة توسيع المحاكي إلى x86_64

## نتائج التدقيق

الحزمة `internal/core/cpu` الحالية مبنية حول نموذج i386 مستقل: ثمانية سجلات `uint32`، و`EIP` و`EFLAGS` بعرض 32 بت، وعناوين الذاكرة من نوع `Address` المبني على 32 بت، وdecoder يستدعي `x86asm.Decode(..., 32)` ثم يرفض كل instruction لا تحمل `DataSize==32`. كما أن `Instruction` و`Operand` يحتويان على حقول بعرض 32 بت، لذلك لا يصح توسيعها في مكانها دون تغيير دلالات i386 الحالية.

حالة FPU وMMX/XMM موجودة داخل `MachineState` لكنها ليست نموذجاً كافياً لمعالج 64 بت؛ سجلات XMM الحالية ثمانية فقط، بينما نمط long mode يحتاج إلى ستة عشر XMM على الأقل، وسجلات GPR إضافية مثل R8–R15، وRIP وRFLAGS بعرض 64 بت، وFS/GS base بعرض 64 بت.

طبقة ELF الحالية تقبل `ELFCLASS32` و`EM_386` فقط، وloader/stack/TLS/syscall dispatcher مبنية على ABI i386. لذلك ستضاف واجهات ELF64 وABI x86_64 منفصلة مع إبقاء مسار i386 سليماً، ثم يُدمجان عبر اختيار architecture داخل guest lifecycle.

## قرار التصميم

سيُنشأ نموذج `MachineState64` و`Instruction64` وdecoder/executor خاصان بـx86_64 بدلاً من إعادة استخدام أنواع i386 بعروض ناقصة. ستُشارك طبقات الذاكرة وعمليات effective-address حيث يمكن ذلك، لكن كل adapter سيتحقق صراحة من `DataSize` و`AddrSize` وRex semantics. سيبقى `x86asm` المصدر الوحيد لتفكيك التعليمات.

التنفيذ سيبدأ بالنواة اللازمة لتشغيل كود ELF64 عادي: register moves، arithmetic/logical، compare/test، shifts/rotates، branches/call/ret، stack، sign/zero extension، multiply/divide، byte-register semantics، REX.W، RIP-relative addressing، وREX.R/B/X. بعد ذلك تُضاف تعليمات atomic/SSE الضرورية وABI/syscalls وELF64/TLS/dynamic linking.

لا يُستخدم CGo. كل اختبار يُشغّل عبر `CGO_ENABLED=0 go test ./internal/core/...`، ثم `go vet` و`make build` قبل كل commit مستقل.

## الفجوات المعروفة

| الطبقة | الوضع الحالي | المطلوب لـx86_64 |
|---|---|---|
| CPU state | i386 فقط | GPR 64-bit، RIP/RFLAGS، R8–R15، XMM0–XMM15، FS/GS base 64-bit |
| decoder | `x86asm.Decode(..., 32)` | decode mode 64، REX/operand-size/address-size، RIP-relative |
| executor | قيم وعناوين `uint32` | عمليات 8/16/32/64، sign extension، canonical address checks |
| ELF | ELF32/EM_386 | ELF64/EM_X86_64، PT_LOAD، PIE، stack وauxv |
| ABI | i386 int 0x80 | Linux x86_64 `syscall`، RDI/RSI/RDX/R10/R8/R9 وRAX |
| TLS | i386 GS/TLS | FS-base/arch_prctl وTLS64 |
| SIMD | XMM0–XMM7 في state فقط | XMM0–XMM15 وsubset SSE2 الضروري |
| UI/guest | architecture-agnostic جزئياً | اختيار architecture وتوجيه lifecycle إلى CPU/ELF/ABI المناسب |

## تحليل asbestos JIT في iSH

مرجع iSH لا يستخدم interpreter بسيطاً فقط؛ محرك `asbestos` يبني `fiber_block` من نقطة IP، ويولد سلسلة gadgets أثناء `gen_step` حتى تصل التعليمة إلى نهاية كتلة التحكم بالتدفق أو إلى حد الصفحة، ثم يخزن الكتلة في hash حسب عنوان البداية ويربط الكتل اللاحقة عبر jump/return chaining. عند تعديل صفحات الذاكرة، تُبطل الكتل المرتبطة بالصفحات وتُؤجل عملية التحرير عبر jetsam حتى لا تُحرر كتلة ما زالت قيد التنفيذ. حلقة `cpu_run_to_interrupt` تحتفظ بـsmall direct-mapped cache محلي، وتنسخ حالة CPU إلى frame، وتشغّل الكتل حتى interrupt أو timer/poke، ثم تعيد الحالة إلى المضيف.

الـassembly في `gadgets-x86_64/entry.S` يعتمد على ABI خاص لحفظ سجلات المضيف، تحميل حالة guest، ثم تشغيل gadget stream، وهو غير قابل للنقل حرفياً إلى Pure Go لأن Go لا يقدم استدعاء function pointer إلى bytes مولدة بصورة آمنة ومحمولة على Linux وAndroid. لذلك سيحافظ التنفيذ الجديد على **نفس دورة JIT** وخصائصها الدلالية، لكن ستكون `CompiledBlock` سلسلة عمليات Go مترجمة مسبقاً من `x86asm.Inst`، وتُنفذ عبر dispatch مباشر داخل frame. هذا هو البديل Pure Go لـgadget stream؛ وهو يسمح لاحقاً بإضافة backend native اختياري دون تغيير cache أو invalidation أو ABI.

| مفهوم asbestos | نظيره Pure Go المقترح |
|---|---|
| `gen_start/gen_step/gen_end` | `CompileBlock64` يفكك بـ`x86asm.Decode(code, 64)` ويجمع `Op64` حتى branch/syscall/page limit |
| `fiber_block` | `CompiledBlock64` يحوي start/end، page set، operations، branch targets، generation |
| hash + local cache | `BlockCache64` مع map عالمي وdirect-mapped per-run cache |
| gadget execution | `Op64.Execute(frame)` أو closures typed، مع عدم السماح بامتدادات غير موثقة |
| `jump_ip`/return chaining | روابط منطقية إلى block targets مع invalidation عند تغير generation |
| `asbestos_invalidate_range` | `InvalidateRange64(start,end)` يحذف/يعلّم الكتل المتأثرة بالصفحات |
| jetsam/RCU grace period | invalidated list تُحرر بعد انتهاء run frame الحالي |
| `fiber_enter/exit` | `RunFrame64` ينسخ/يعيد حالة CPU ويعيد `Interrupt64` |
| poke/timer | `Poke` ذري/قناة lifecycle + حد دورات كما في iSH |
| TLB helpers | memory translation methods مع صلاحيات read/write/execute وfault metadata |

يجب ألا يُسمى هذا backend native machine-code JIT قبل إضافة backend مستقل يولد وينفذ code pages؛ النسخة الأولى هي **block JIT Pure Go** وتحافظ على behavior/cache/invalidation الخاص بـiSH بلا CGo.
