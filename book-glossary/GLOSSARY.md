# جدول المصطلحات الموحَّد (صارم) — النسخة الثانية عشرة


## قاعدة الصرامة

1. كل مصطلح إنجليزي ← مقابل عربي ثابت.
2. اختلاف المصطلح الإنجليزي ⇐ لا يُعطى نفس المقابل العربي.
3. `Job` = مهمة | `Task` = مهمة فرعية.
4. `Errant` = شاردة | `Runaway` = جامحة.

> صيغة السطر في هذا الملف: المصطلح الإنجليزي، ثم مسافتان أو أكثر، ثم المقابل
> العربي. يعتمد `tools/extract_terms.py` هذه الصيغة، فلا تُدخِل مسافتين
> متتاليتين داخل المصطلح نفسه.

---

## القواعد التحريرية العليا (النسخة 12)

هذه القواعد تسبق أي مدخل في الجدول عند التعارض، والمرجع فيها هو السياق
العلمي لوظيفة المصطلح في الكتاب نفسه:

1. **لا ترجمة بلا سياق.** لا يُفرض مقابل واحد على مصطلح عبر كل السياقات؛
   يُعتمد مقابل الجدول في سياقه المقصود، وما عدا ذلك بديل سياقي يُوثَّق.
2. **Argument / Parameter** — التمييز المقصود في الكتاب:
   - `Argument` = **وسيط** (جمعه: وسائط) — ما يُمرَّر عند الاستدعاء.
   - `Parameter` = **معلمة** (جمعها: معلمات) — الاسم في تعريف الدالة.
   - «معامل» و«معطى» بدائل سياقية وليست الافتراض؛ ولا بد من النظر قبل
     أي توحيد. ويبقى «معامل» بمعنى *coefficient* (معاملات كثيرات الحدود)
     مصطلحًا مختلفًا لا يُستبدل أبدًا، و«معطى» الواردة صفةً («عدد معطى»)
     ليست مصطلحًا أصلًا.
3. **Power** = **قوة** في المعنى الرياضي (x raised to a power)، ولا
   تُخلط بـ«طاقة» إلا في سياق فيزيائي/عتادي صريح.
4. **Operator** = **عامل** في السياق الرياضي/البرمجي (عنصر التعبير)،
   ويُخصَّص «مشغّل» للمشغّل البشري في سياق أنظمة التشغيل.
5. **Accuracy** = **دقة** عمومًا؛ ولا تُفهم «صحة» إلا إذا فرض السياق
   ذلك — والاستثناء الموثَّق هو ثنائية دقة/صحة (Accuracy/Precision)
   لمؤقّت الجدولة في OSTEP، حيث يحتاج المصطلحان مقابلين متمايزين.
6. **لا نقل بين الكتب.** لا يُنقل المقابل الاصطلاحي من كتاب إلى آخر
   (OSTEP ↔ SICP) لمجرد تطابق الكلمة الإنجليزية؛ الوظيفة العلمية
   للمصطلح في سياق كل كتاب هي المرجع. الجدول قاموس خاص بالمشروع
   وسياقاته العلمية، وليس قاموسًا إنجليزيًا عامًا. والمداخل الخاصة
   بكتاب بعينه تُوسَم بلوحة كتاب: `[OSTEP]` أو `[SICP]` — ولا يُطبَّق
   المدخل الموسوم إلا في سياق كتابه.

---

## القسم الأول: المفاهيم الثلاثة الرئيسية للكتاب

```
Virtualization                              الافتراضية
virtual                                     افتراضي
virtualized                                 مُنفَّذ بأسلوب افتراضي / مُقدَّم افتراضيًا / جرى افتراضه حسب السياق
virtualization                              الافتراضية
virtualized registers                       السجلات المُقدَّمة افتراضيًا
Concurrency                                 التزامن
Persistence                                 ثبات البيانات
```

## القسم الثاني: نظام التشغيل — المفاهيم الأساسية

```
Operating System (OS)                       نظام التشغيل
OS Subsystem                                نظام فرعي لنظام التشغيل
Kernel                                      النواة
Microkernel                                 نواة مصغّرة
Monolithic Kernel                           النواة المتجانسة
Virtual Machine                             آلة افتراضية
Virtual Machine Monitor                     مراقب الآلة الافتراضية
Resource Manager                            مدير الموارد
Standard Library                            مكتبة قياسية
Supervisor                                  المشرف
Master Control Program                      برنامج التحكم الرئيسي
```

## القسم الثالث: الافتراضية والذاكرة

```
Virtualizing Memory                         افتراضية الذاكرة
Virtualizing the CPU                        افتراضية المعالج
Virtual Memory                              الذاكرة الافتراضية
Virtual Memory System                       نظام الذاكرة الافتراضية
Virtual Address Space                       فضاء العناوين الافتراضي
Address Space                               فضاء العناوين
Virtual Address                             عنوان افتراضي
Physical Address                            عنوان فيزيائي
Physical Memory                             الذاكرة الفيزيائية
Abstraction                                 التجريد
Illusion                                    وهم
Address-Space Randomization (ASLR)          عشوائية فضاء العناوين
Base/Bounds                                 الأساس والحدود
```

## القسم الرابع: مكوّنات فضاء العناوين وأنواع الذاكرة

```
Code                                        الكود
Code Segment                                جزء الكود
Stack                                       المكدس
Heap                                        الكومة
Stack Memory                                ذاكرة المكدس
Heap Memory                                 ذاكرة الكومة
Automatic Memory                            الذاكرة التلقائية
Anonymous Memory Region                     منطقة ذاكرة مجهولة
Swap Space                                  مساحة المبادلة
Program's Break                             فاصل البرنامج
Static Variables                            متغيرات ساكنة
Local Variables                             متغيرات محلية
Parameters                                  معلمات
Return Values                               قيم الإرجاع
Function Call Chain                         سلسلة استدعاءات الدوال
Dynamic Allocation                          تخصيص ديناميكي
Allocate / Allocation                       يُخصِّص / تخصيص
Deallocate / Free                           يُلغي تخصيص / يُحرر
malloc()                                    malloc()
new                                         new
Pointer                                     مؤشر
Dereference                                 فك المرجعية (للمؤشر)
Cast / Casting                              تحويل / تحويل النوع
Data Structures                              هياكل البيانات
Array                                       مصفوفة
String                                      سلسلة نصية
```

## القسم الخامس: أخطاء الذاكرة وإدارتها

```
Automatic Memory Management                 الإدارة التلقائية للذاكرة
Garbage Collector                           جامع القمامة
Memory Leak                                 تسرب الذاكرة
Segmentation Fault (Segfault)               خطأ تقسيم
Fragmentation                               التجزئه
Buffer Overflow                             طفح المخزن المؤقت
Uninitialized Read                          قراءة غير مهيأة
Dangling Pointer                            مؤشر متدلي
Double Free                                 تحرير مزدوج
Invalid Free                                تحرير غير صالح
```

## القسم السادس: العمليات — أساسيات

```
Process                                     عملية
Running Program                             البرنامج الجاري
Thread                                      خيط
Threads                                     خيوط
Multi-threaded Program                      برنامج متعدد الخيوط
Context Switch                              تبديل السياق
I/O (Input/Output)                          إدخال / إخراج
CPU                                         وحدة المعالجة المركزية (CPU)
CPU Utilization                             نسبة استغلال المعالج
PID (Process ID)                            معرّف العملية
```

## القسم السابع: التعليمات ونموذج الحوسبة

```
Von Neumann Model                           نموذج فون نيومان
Fetch (instruction)                         جلب (تعليمة)
Decode (instruction)                        فكّ ترميز (تعليمة)
Execute (instruction)                       تنفيذ (تعليمة)
Load (instruction)                          تحميل (تعليمة)
Store (instruction)                         تخزين (تعليمة)
Instruction Fetch                           جلب تعليمة
Register                                    مسجّل
Registers                                   مسجّلات
Program Counter (PC)                        عدّاد البرنامج
General-Purpose Registers                   مسجّلات الأغراض العامة
Atomic (execution)                          ذرّي (تنفيذ)
```

## القسم الثامن: أنماط نظام التشغيل التاريخية

```
Multiprogramming                            البرمجة المتعددة
Time Sharing                                المشاركة الزمنية
Batch Computing                             الحوسبة الدُّفعية
Batch Processing                            المعالجة الدُّفعية
Interactivity                               التفاعلية
Mainframe                                   الحاسوب المركزي
Minicomputer                                الحاسوب الصغير
Personal Computer (PC)                      الحاسوب الشخصي
Operator                                    عامل (رياضي/برمجي)
Operator (human) [OSTEP]                    المشغّل (البشري)
```

## القسم التاسع: أهداف تصميم نظام التشغيل

```
Transparency                                الشفافية
Efficiency                                  الكفاءة
Protection                                  الحماية
Isolation                                   العزل
Reliability                                 الموثوقية
Security                                    الأمان
Energy-Efficiency                           كفاءة الطاقة
Mobility                                    التنقّلية
Performance                                 الأداء
Overhead                                    عبء إضافي
Overheads                                   أعباء إضافية
Trade-off                                   مفاضلة
Trade-offs                                  مفاضلات
Mechanism                                   آلية
Mechanisms                                  آليات
Policy                                      سياسة
Policies                                    سياسات
```

## القسم العاشر: العتاد والبنية التحتية

```
Hardware                                    العتاد
Hardware Support                            دعم العتاد
Hardware Privilege Level                    مستوى امتياز العتاد
Translation Lookaside Buffer (TLB)          مخزن الترجمة المؤقت
TLB miss                                    إخفاق في مخزن الترجمة المؤقت
TLB hit                                     إصابة في مخزن الترجمة المؤقت
DRAM                                        DRAM (ذاكرة ديناميكية)
Hard Disk Drive (HDD)                       قرص صلب
Solid-State Drive (SSD)                     قرص الحالة الصلبة
I/O Device                                  جهاز إدخال/إخراج
Disk                                        قرص
Disks                                       أقراص
RAID                                        RAID
Interrupt                                   مقاطعة
Interrupts                                  مقاطعات
Network                                     الشبكة
Packet                                      حزمة (بيانات)
```

**صيغ `TLB` حسب السياق:**

| الصيغة | المقابل |
|---|---|
| أوّل ظهور | «مخزن الترجمة المؤقت (Translation Lookaside Buffer، TLB)» |
| مفرد معرَّف | مخزن الترجمة المؤقت / المخزن |
| نكرة | مخزن ترجمة مؤقت |
| جمع | مخازن الترجمة المؤقتة |

## القسم الحادي عشر: استدعاءات النظام والامتياز

```
System Call                                 استدعاء النظام
Procedure Call                              استدعاء الإجراء
Library Call                                استدعاء مكتبة
API (Application Programming Interface)     واجهة برمجة التطبيقات
Trap (instruction)                          المصيدة (تعليمة)
Trap Handler                                معالج المصيدة
Return-from-Trap                            العودة من المصيدة
User Mode                                   وضع المستخدم
Kernel Mode                                 وضع النواة
Privilege Level                             مستوى الامتياز
open()                                      open()
read()                                      read()
write()                                     write()
close()                                     close()
exit()                                      exit()
getpid()                                    getpid()
```

## القسم الثاني عشر: نظام الملفات والتخزين الدائم

```
File System                                 نظام الملفات
File                                        ملف
Files                                       ملفات
Persistent Storage                          تخزين دائم
Volatile (storage)                          متطاير (تخزين)
Journaling                                  التدوين اليومي
Copy-on-Write                               النسخ عند الكتابة
B-Tree                                      شجرة B
Device Driver                               برنامج تشغيل الجهاز
Error Code                                  رمز الخطأ
```

## القسم الثالث عشر: التزامن والبرمجة المتعددة الخيوط

```
Concurrency Problem                         مشكلة التزامن
Race Condition                              حالة تسابق
Shared Memory                               ذاكرة مشتركة
Memory Protection                           حماية الذاكرة
Primitive                                   أوليّة
Primitives                                  أوّليات
Atomic Operation                            عملية ذرّية
volatile (keyword)                          volatile (كلمة محجوزة)
worker() routine                            روتين worker()
```

## القسم الرابع عشر: أنظمة تشغيل تاريخية وحديثة

```
UNIX                                        يونكس
Linux                                       لينكس
DOS (Disk Operating System)                 نظام تشغيل الأقراص (دوس)
Mac OS                                      نظام تشغيل ماك (Mac OS)
macOS                                       نظام تشغيل ماك (macOS)
Windows NT                                  ويندوز NT
Android                                     أندرويد
BSD (Berkeley Systems Distribution)         توزيعة أنظمة بيركلي (BSD)
Multics                                     مولتيكس
GNU                                         GNU
Open-Source Software                        برمجيات مفتوحة المصدر
Shell                                       الشِل
Pipe                                        أنبوب
Pipes                                       أنابيب
grep                                        grep
wc (word count)                             wc (عدّ الكلمات)
Distributed System                          نظام موزَّع
```

## القسم الخامس عشر: مصطلحات التمثيل والقيم

```
Hexadecimal                                 النظام الست عشري
Arbitrary (address)                         اعتباطي
Sparse (address space)                      متخلخل
Contiguous                                  متصل
Convention                                  اصطلاح
Lower Addresses                             العناوين الأصغر
Higher Addresses                            العناوين الأكبر
Free Space                                  المساحة الحرة
Page                                        صفحة
Pages                                       صفحات
```

## القسم السادس عشر: أدوات لينكس والواجبات العملية

```
free (tool)                                 free (أداة)
pmap (tool)                                 pmap (أداة)
ps auxw                                     ps auxw
gcc (compiler)                              gcc (مترجم)
Python                                      Python
Terminal Window                             نافذة طرفية
Command-line Argument                       معامل سطر الأوامر
Megabyte (MB)                               ميجابايت
Simulation                                  محاكاة
Latency                                     زمن الاستجابة
Background Job                              مهمة في الخلفية
Debugger                                    مُنقّح
Symbol Information                          معلومات الرموز
valgrind / memcheck                         فالجريند / ميم-شيك
process-run.py                              process-run.py
fork.py                                     fork.py
scheduler.py                                scheduler.py
mlfq.py                                     mlfq.py
lottery.py                                  lottery.py
multi.py                                    multi.py
```

## القسم السابع عشر: مصطلحات التعلّم والدراسة

```
Professor                                   الأستاذ
Student                                     الطالب
Lecture                                     محاضرة
Lectures                                    محاضرات
Notes                                       ملاحظات
Homework                                    واجب
Homeworks                                   واجبات
Project                                     مشروع
Projects                                    مشاريع
Exam                                        امتحان
Real Code                                   شيفرة حقيقية
Real Problems                               مشكلات حقيقية
Material                                    المادة (الدراسية)
Basics                                      الأساسيات
Narrative                                   السرد
Dialogue                                    حوار
Dialogues                                   حوارات
```

## القسم الثامن عشر: الفصل 3 — حوار حول الافتراضية (إضافات)

```
Application                                 تطبيق
Applications                                تطبيقات
Resource                                    مورد
Resources                                   موارد
Entity                                      كيان
virtual (adjective)                         افتراضي (صفة)
physical (adjective)                        مادى (صفة)
Virtual CPU                                 معالج افتراضي
Virtual CPUs                                معالجات افتراضية
Physical CPU                                معالج مادى
(to) virtualize (verb)                      يجعل افتراضياً
```

> ملاحظة على `physical`: يمكن استخدام «فيزيائي» إذا كان له علاقة بالفيزياء.

## القسم التاسع عشر: الفصل 4 — التجريد: العملية (إضافات)

```
Process API                                 واجهة برمجة تطبيقات العملية
Create (process)                            إنشاء (عملية)
Destroy (process)                           تدمير (عملية)
Exit (process)                              خروج (عملية)
Kill (a process)                            قتل (عملية)
Runaway Process                             عملية جامحة
Errant Process                              عملية شاردة
Errant Program                              برنامج شارد
Suspend (a process)                         تعليق (عملية)
Resume (a process)                          استئناف (عملية)
Wait (for a process)                        انتظار (عملية)
Status (process)                            حالة (عملية)
Machine State                               حالة الآلة
Instruction Pointer (IP)                    مؤشر التعليمة (IP)
Stack Pointer                               مؤشر المكدس
Frame Pointer                               مؤشر الإطار
Register Context                            سياق المسجّلات
Process State                               حالة العملية
Process States                              حالات العملية
Running (state)                             قيد التشغيل
Ready (state)                               جاهزة
Blocked (state)                             محظورة
blocking                                    حاجب
non-blocking                                غير حاجب
Scheduled                                   مُجدوَل
Descheduled                                 مُزال من الجدولة
State Transition                            انتقال حالة
State Transitions                           انتقالات الحالة
Trace (of execution/state)                  تتبّع
Wake (a process)                            إيقاظ (عملية)
Program Loading                             تحميل البرنامج
Executable Format                           صيغة تنفيذية
Executable Formats                          صيغ تنفيذية
Eager Loading                               تحميل دَفعة واحدة
Lazy Loading                                تحميل كسول
On-demand (loading)                         تحميل عند الطلب
Paging                                      التقسيم إلى صفحات
Swapping                                    المبادلة
Run-time Stack                              مكدس وقت التشغيل
Entry Point                                 نقطة الدخول
Argument                                    وسيط
Arguments                                   وسائط
File Descriptor                             واصف ملف
File Descriptors                            واصفات ملفات
Standard Input                              الإدخال القياسي
Standard Output                             الإخراج القياسي
Standard Error                              الخطأ القياسي
Process List                                قائمة العمليات
Task List                                   قائمة المهام الفرعية
Process Control Block (PCB)                 كتلة تحكّم العملية (PCB)
Process Descriptor                          واصف العملية
Parent Process                              العملية الأب
Child Process                               العملية الابن
Return Code                                 رمز الإرجاع
Zombie State                                حالة الزومبي
xv6                                         xv6
Kernel Stack                                مكدس النواة
Trap Frame                                  إطار المصيدة
```

## القسم العشرون: الفصل 5 — فاصل: واجهة برمجة العمليات (إضافات)

```
Interlude                                   فاصل
Interludes                                  فواصل
Practical (aspects)                         الجوانب العملية
fork()                                      fork()
exec()                                      exec()
execvp()                                    execvp()
execl()                                     execl()
execle()                                    execle()
execlp()                                    execlp()
execv()                                     execv()
execvpe()                                   execvpe()
wait()                                      wait()
waitpid()                                   waitpid()
kill()                                      kill()
signal()                                    signal()
Signal                                      إشارة
Signals                                     إشارات
Process Group                               مجموعة عمليات
SIGINT                                      SIGINT
SIGTSTP                                     SIGTSTP
control-c                                   control-c
control-z                                   control-z
fg (shell built-in)                         fg
Built-in Command                            أمر مدمج
Deterministic                               حتمي
Non-determinism                             اللاحتمية
Prompt                                      موجّه (Prompt)
Command                                     أمر
Redirection                                 إعادة توجيه
Output Redirection                          إعادة توجيه الإخراج
Input/Output Redirection                    إعادة توجيه الدخل/الخرج
STDIN_FILENO                                STDIN_FILENO
STDOUT_FILENO                               STDOUT_FILENO
STDERR_FILENO                               STDERR_FILENO
pipe()                                      pipe()
In-kernel (pipe)                            داخل النواة (pipe)
Queue                                       طابور (بيانات)
man pages                                   صفحات الدليل (man pages)
RTFM                                        RTFM
User                                        المستخدم
Credentials                                 بيانات الاعتماد
Login                                       تسجيل الدخول
Password                                    كلمة مرور
Superuser                                   المستخدم المتميز
root                                        الجذر (root)
shutdown                                    shutdown
ps                                          ps
top                                         top
killall                                     killall
Process Tree                                شجرة العمليات
Orphaned Process                            عملية يتيمة
Random Seed                                 بذرة عشوائية
Flag (command-line option)                  خيار (Option)
Action                                      إجراء
Actions                                     إجراءات
spawn()                                     spawn()
```

## القسم الحادي والعشرون: الفصل 6 — الآلية: التنفيذ المباشر المحدود

```
Limited Direct Execution (LDE)              التنفيذ المباشر المحدود
Direct Execution                            التنفيذ المباشر
Direct Execution Protocol                   بروتوكول التنفيذ المباشر
Control                                     التحكم
Restricted Operation                        عملية مقيدة
Restricted Operations                       عمليات مقيدة
Restricted Instruction                      تعليمة مقيدة
Restricted Instructions                     تعليمات مقيدة
Privileged Operation                        عملية ذات امتيازات
Privileged Operations                       عمليات ذات امتيازات
Privileged Instruction                      تعليمة ذات امتيازات
Privileged Instructions                     تعليمات ذات امتيازات
Exception                                   استثناء
Illegal Operation                           عملية غير مشروعة
Illegal Instruction                         تعليمة غير مشروعة
Protected Control Transfer                  نقل التحكم المحمي
Trap Table                                  جدول المصائد
System-call Number                          رقم استدعاء نظام
Calling Convention                          اتفاقية الاستدعاء
C Library                                   مكتبة C
Assembly (language)                         لغة التجميع

Assembler → مُجَمِّع: البرنامج الذي يحوّل لغة التجميع إلى شفرة آلة.
assemble → يُجَمِّع
assembly → تجميع بحسب السياق.
Boot Time                                   وقت التمهيد
Boot Sequence                               تسلسل التمهيد
Reboot                                      إعادة تشغيل الجهاز
Cooperative (approach)                      نهج تعاوني
Non-cooperative (approach)                  نهج غير تعاوني
Cooperative Preemption                      الإيقاف الطوعي
yield (system call)                         استدعاء yield (التنازل)
Infinite Loop                               حلقة لا نهائية
Misbehaving process                         عملية سيئة السلوك
Rogue process                               عملية مارقة
Timer                                       مؤقت
Timer Device                                جهاز مؤقت
Timer Interrupt                             مقاطعة المؤقت
Interrupt Timer                             مؤقت المقاطعات
Interrupt Handler                           معالج مقاطعة
Syscall Handler                             معالج استدعاءات النظام
Disable Interrupts                          تعطيل المقاطعات
Lost Interrupts                             ضياع المقاطعات
Save Registers                              حفظ المسجّلات
Restore Registers                           استعادة المسجّلات
Kernel Stack Pointer                        مؤشر مكدس النواة
Flags (register)                            الأعلام (مسجل)
Lock                                        قفل
Locks                                       أقفال
Locking Scheme                              مخطط أقفال
Locking Schemes                             مخططات أقفال
Multiprocessor(s)                           نظام متعدد المعالجات
Memory Bandwidth                            عرض نطاق الذاكرة
Memory-intensive                            كثيف الذاكرة
Benchmark                                   معيار قياس
lmbench                                     lmbench
Microsecond (µs)                            ميكروثانية
Millisecond (ms)                            مللي ثانية
MHz                                         ميجاهرتز
GHz                                         جيجاهرتز
Precision (of timer) [OSTEP]                دقة (المؤقت)
Accuracy                                    دقة
Accuracy (of timer) [OSTEP]                 صحة (المؤقت)
gettimeofday()                              gettimeofday()
rdtsc                                       rdtsc
Null System Call                            استدعاء نظام صفري
0-byte read                                 قراءة 0-بايت
UNIX Socket(s)                              مقابس يونكس (sockets)
Bind (process to CPU)                       ربط عملية بمعالج
sched_setaffinity()                         sched_setaffinity()
```

> في هذا الفصل يظهر `Mechanism` بصيغة «آلية (Mechanism)»؛ المعتمد في القسم
> التاسع هو «آلية» — انظر تقرير التعارضات.

## القسم الثاني والعشرون: الفصل 7 — الجدولة: مقدمة

```
Scheduling                                  الجدولة
Scheduling Policy                           سياسة الجدولة
Scheduling Discipline                       منهج جدولة
Scheduling Disciplines                      مناهج جدولة
Workload                                    عبء العمل
Workload Assumptions                        افتراضات عبء العمل
Scheduling Metric                           مقياس الجدولة
Metric                                      مقياس
Turnaround Time                             وقت الإنجاز
Completion Time                             وقت الاكتمال
Arrival Time                                وقت الوصول
Average Turnaround Time                     متوسط وقت الإنجاز
Response Time                               وقت الاستجابة
First Run Time (Tfirstrun)                  وقت أول تشغيل
Fairness                                    العدالة
Jain's Fairness Index                       مؤشر جاين للعدالة
First In, First Out (FIFO)                  الوارد أولاً يُخدم أولاً (FIFO)
First Come, First Served (FCFS)             من يأتي أولاً يُخدم أولاً (FCFS)
Convoy Effect                               تأثير القافلة
Shortest Job First (SJF)                    المهمة الأقصر أولاً (SJF)
Shortest Time-to-Completion First (STCF)    الوقت الأقصر للاكتمال أولاً (STCF)
Preemptive Shortest Job First (PSJF)        المهمة الأقصر أولاً — استباقي (PSJF)
Preemptive (scheduler)                      استباقي (مُجدوِل)
Non-preemptive (scheduler)                  غير استباقي (مُجدوِل)
(to) Preempt                                يستبق (يُقاطِع)
Round Robin (RR)                            جولة روبن (RR)
Time Slice                                  شريحة زمنية
Scheduling Quantum                          كمّيّة الجدولة
Quantum Length                              طول الكمّيّة
Time-slicing                                التقطيع الزمني
Timer-interrupt Period                      فترة مقاطعة المؤقت
Amortization                                الإطفاء (توزيع الكلفة)
CPU Cache(s)                                مخابئ المعالج (Caches)
Branch Predictor(s)                         متنبئات التفرّع
On-chip (hardware)                          عتاد على الشريحة
(to) Flush (caches/TLBs)                    تفريغ (المخازن/‏TLBs)
Overlap                                     التداخل
Utilization                                 معدل الاستخدام
Resource Utilization                        معدل استخدام الموارد
CPU Burst                                   دفعة معالج
Interactive (job/process)                   تفاعلي (مهمة/عملية)
CPU-intensive                               كثيف الاستخدام للمعالج
Oracle (omniscient scheduler)               عرّافة (مُجدوِل عارف بالغيب)
Omniscient                                  عارف بالغيب
Total Wait Time                             وقت الانتظار الإجمالي
Ready Queue                                 طابور الجاهزية
Run Queue                                   طابور التشغيل
Job                                         مهمة
Jobs                                        مهام
Task                                        مهمة فرعية
Tasks                                       مهام فرعية
```

## القسم الثالث والعشرون: الفصل 8 — الجدولة: طابور التغذية الراجعة متعدد المستويات

```
Multi-Level Feedback Queue (MLFQ)           طابور التغذية الراجعة متعدد المستويات (MLFQ)
Multi-level                                 متعدد المستويات
Feedback                                    التغذية الراجعة
Queue Level(s)                              مستوى/مستويات الطابور
Priority                                    أولوية
Priority Level                              مستوى أولوية
High Priority                               أولوية عالية
Low Priority                                أولوية منخفضة
Topmost Queue                               الطابور العلوي
Higher Queue                                الطابور الأعلى
Lower Queue                                 الطابور الأدنى
Priority Adjustment                         تعديل الأولوية
Observed Behavior                           السلوك المُلاحَظ
History (of a job)                          تاريخ المهمة
Predict (future behavior)                   التنبّؤ (بالسلوك المستقبلي)
Relinquish (the CPU)                        التخلّي عن المعالج
CPU-bound                                   مقيّد بالمعالج
I/O-intensive                               كثيف الإدخال/الإخراج
Allotment (time allotment)                  الحصّة الزمنية
Allotment Reset                             إعادة ضبط الحصّة الزمنية
Time Period S                               فترة زمنية S
Priority Boost                              تعزيز الأولوية (رفع دوري)
(to) Boost Priorities                       يعزّز الأولويات
(to) Promote (priority)                     يُرقّي الأولوية
(to) Demote (priority)                      يُنزّل الأولوية
Periodic (boost)                            دوري (تعزيز)
Starvation                                  التجويع
(to) Starve                                 يُجوّع
Make Progress                               إحراز تقدّم
Gaming (the scheduler)                      التلاعب بالمُجدوِل
Gaming Tolerance                            مقاومة التلاعب
Anti-gaming                                 مضاد للتلاعب
Fair Share                                  النصيب العادل
Unfair Share                                نصيب غير عادل
(to) Monopolize (the CPU)                   احتكار المعالج
Secure From Attack                          محصّن ضد الهجمات
Security Concern                            مصدر قلق أمني
Policy Enforcement                          إنفاذ السياسة
Parameterize (a scheduler)                  تعيين معلمات المُجدوِل
Parameter(s)                                معلمة / معلمات
Tuning                                      ضبط المعلمات
Configuration File                          ملف تكوين
Default Values                              قيم افتراضية
Voo-doo Constants                           ثوابت الفودو
Ousterhout's Law                            قانون أوسترهوت
System Administrator                        مسؤول النظام
Time-Sharing Scheduling Class (TS)          فئة جدولة المشاركة الزمنية (TS)
Table (of scheduling parameters)            جدول (معلمات الجدولة)
Formula(e)                                  صيغة/صيغ رياضية
Decay-Usage Scheduling                      جدولة الاستخدام المتلاشي
Decay (over time)                           تلاشي (بمرور الوقت)
Advice / Hints                              تلميحات (نصيحة)
nice (utility)                              nice
madvise                                     madvise
Memory Manager                              مدير الذاكرة
Informed Prefetching                        الجلب المسبق المستنير
Caching                                     التخزين المؤقت
Compatible Time-Sharing System (CTSS)       نظام المشاركة الزمنية المتوافق (CTSS)
ACM (Association for Computing Machinery)   رابطة آلات الحوسبة (ACM)
Turing Award                                جائزة تورنغ
```

## القسم الرابع والعشرون: الفصل 9 — الجدولة: الحصّة النسبية

```
Proportional-share Scheduler                مُجدوِل الحصّة النسبية
Proportional-share Scheduling               جدولة الحصّة النسبية
Fair-share Scheduler                        مُجدوِل الحصّة العادلة
Fair-share Scheduling                       جدولة الحصّة العادلة
Share (of CPU time)                         حصّة (من وقت المعالج)
Percentage                                  نسبة مئوية
Lottery Scheduling                          جدولة اليانصيب
Lottery (event)                             يانصيب (حدث)
Ticket                                      تذكرة
Tickets                                     تذاكر
Total Tickets                               إجمالي التذاكر
Winning Ticket                              تذكرة فائزة
Ticket Range                                مجال التذاكر
Probabilistic                               احتمالي
Probabilistically                           احتمالياً
Deterministically                           حتميّاً
Probabilistic Correctness                   صحّة احتمالية
Guarantee                                   ضمان
Randomness                                  العشوائية
Random (adjective)                          عشوائي (صفة)
Randomized (approach)                       عشوائي (نهج مُصرَّح بالعشوائية)
Pseudo-random                               شبه عشوائي
Random Number Generator                     مُولِّد أرقام عشوائية
Corner Case                                 حالة طرفية
Corner Cases                                حالات طرفية
Worst Case                                  حالة أسوأ
Worst-case Performance                      أداء في الحالة الأسوأ
LRU (replacement policy)                    LRU (سياسة الأقل استخداماً مؤخراً)
Replacement Policy                          سياسة الاستبدال
Cyclic-sequential Workload                  عبء عمل تتابعي دوري
Ticket Mechanism(s)                         آليات التذاكر
Ticket Currency                             عملة التذاكر
Currency Conversion                         تحويل العملة
Global Currency                             العملة العامة
Ticket Transfer                             نقل التذاكر
Ticket Inflation                            تضخيم التذاكر
Greedy (process)                            عملية جشعة
Client/Server Setting                       بيئة عميل/خادم
Request                                     طلب
Implementation                              التنفيذ
Data Structure                              هيكل بيانات
List (data structure)                       قائمة (هيكل بيانات)
Ordered List                                قائمة مرتّبة
Sorted Order                                ترتيب مُفرَز
Traverse (a list)                           اجتياز (قائمة)
List Iterations                             تكرارات اجتياز القائمة
Fairness Metric (F)                         مقياس عدالة (F)
Job Length (R)                              طول المهمة (R)
Trial(s)                                    تجربة/تجارب
Simulator                                   مُحاكي
Ticket-assignment Problem                   مشكلة تخصيص التذاكر
Stride Scheduling                           جدولة الخطوة
Stride                                      خطوة (Stride)
Inverse Proportion                          تناسب عكسي
Pass Value                                  قيمة المرور
Global Progress                             التقدّم الكلّي
No Global State                             انعدام الحالة العامة
Global Variable                             متغيّر عام
Scheduling Cycle                            دورة جدولة
remove_min(queue)                           remove_min(queue)
insert(queue, curr)                         insert(queue, curr)
Completely Fair Scheduler (CFS)             المُجدوِل العادل تماماً (CFS)
Fair Division (of CPU)                      قسمة عادلة (للمعالج)
Virtual Runtime                             وقت تشغيل افتراضي
vruntime                                    vruntime
Physical (real) Time                        وقت فيزيائي (حقيقي)
Control Parameter(s)                        معلمة/معلمات تحكّم
sched_latency                               sched_latency
min_granularity                             min_granularity
Nice Level                                  مستوى اللُّطف (nice level)
nice (parameter)                            nice (معلمة)
Weight                                      وزن
prio_to_weight                              prio_to_weight
Default (nice=0)                            افتراضي (nice=0)
Weighted Round-Robin                        جولة روبن مُرجَّحة
Dynamic Time Slices                         شرائح زمنية ديناميكية
Periodic Timer Interrupt                    مقاطعة مؤقّت دورية
Fixed Time Interval(s)                      فترات زمنية ثابتة
Red-Black Tree                              شجرة حمراء-سوداء
Balanced Tree                               شجرة متوازنة
Insertion                                   إدراج
Deletion                                    حذف
Logarithmic Time                            زمن لوغاريتمي
Linear Time                                 زمن خطّي
Runnable (process)                          قابلة للتشغيل (عملية)
Sleeping (process)                          نائمة (عملية)
Wake Up (a process)                         استيقاظ (عملية)
Catch Up (with others)                      اللحاق بالركب
Heuristic(s)                                استدلال/استدلالات
Scalable                                    قابل للتوسّع
Scalability                                 قابلية التوسّع
Datacenter                                  مركز بيانات
Scheduler Efficiency                        كفاءة المُجدوِل
Scheduler Overhead                          عبء المُجدوِل الإضافي
CPU Cycle(s)                                دورات المعالج
Cloud                                       السحابة
Virtualized Datacenter                      مركز بيانات افتراضي
Hypervisor(s)                               مُشرف/مُشرفو الأجهزة الافتراضية
Guest Operating System                      نظام تشغيل ضيف
ESX Server                                  ESX Server
VMWare                                      VMWare
VM (Virtual Machine)                        آلة افتراضية
```

## القسم الخامس والعشرون: الفصل 10 — جدولة المعالجات المتعددة (متقدم)

```
Multiprocessor Scheduling                   جدولة المعالجات المتعددة
Multiprocessor Scheduling (Advanced)        جدولة المعالجات المتعددة (متقدم)
Multiprocessor System(s)                    نظام/أنظمة متعددة المعالجات
Single CPU                                  وحدة معالجة مركزية واحدة
Single-CPU Hardware                         عتاد أحادي المعالج
Multi-CPU Hardware                          عتاد متعدد المعالجات
Processor                                   معالج (Processor)
Processors                                  معالجات (Processors)
Multicore Processor                         معالج متعدد الأنوية
CPU Core                                    نواة وحدة المعالجة المركزية
CPU Cores                                   أنوية وحدة المعالجة المركزية
Chip                                        شريحة
Power                                       قوة (رياضية) / طاقة (فيزيائية)
Parallel                                    توازٍ
Parallelism                                 التوازي
Parallelization                             الموازاة
Multiprocessor Architecture                 معمارية المعالجات المتعددة
Computer Architecture                       معمارية الحاسوب
Graduate (course)                           مساق دراسات عليا
Cache                                       ذاكرة تخزين مؤقت
Caches                                      ذواكر تخزين مؤقت
Hardware Cache                              ذاكرة تخزين مؤقت عتادية
Hardware Caches                             ذواكر تخزين مؤقت عتادية
CPU Cache                                   ذاكرة تخزين مؤقت للمعالج
CPU Caches                                  ذواكر تخزين مؤقت للمعالج
Cache Hierarchy                             تسلسل هرمي لذواكر التخزين المؤقت
Main Memory                                 الذاكرة الرئيسية
Memory                                      ذاكرة
Locality                                    المحلية
Temporal Locality                           المحلية الزمنية
Spatial Locality                            المحلية المكانية
Load Instruction                            تعليمة تحميل
Explicit Load Instruction                   تعليمة تحميل صريحة
Address                                     عنوان
Data Item                                   عنصر بيانات
Bus                                         ناقل
Bus-based System                            نظام معتمد على الناقل
Cache Coherence                             ترابط ذاكرة التخزين المؤقت
Coherence Protocol                          بروتوكول الترابط
Cache Coherence Protocol                    بروتوكول ترابط ذاكرة التخزين المؤقت
Single Shared Memory                        ذاكرة مشتركة واحدة
Bus Snooping                                التطفّل على الناقل
Monitor (memory accesses)                   مراقِب (عمليات الوصول إلى الذاكرة)
Monitor                                     مراقِب
Monitors                                    مراقِبات
monitor lock                                قفل المراقِب
monitor synchronization                     مزامنة المراقِب
Invalidate (a cache line/copy)              يُبطل (نسخة/سطر)
Update (a cache line/copy)                  يُحدّث (نسخة/سطر)
Write-back Cache                            ذاكرة تخزين مؤقت ذات كتابة مؤجّلة
Write-back                                  كتابة مؤجّلة
Synchronization                             المزامنة
Mutual Exclusion                            الاستبعاد المتبادل
Mutual Exclusion Primitive                  أوليّة استبعاد متبادل
Mutual Exclusion Primitives                 أوليات استبعاد متبادل
locking                                     القفل
lock-free                                   بلا أقفال
wait                                        انتظار
wait-free                                   بلا انتظار
livelock                                    الجمود الحيّ
Lock-free Data Structure                    هيكل بيانات بلا أقفال
Lock-free Data Structures                   هياكل بيانات بلا أقفال
Deadlock                                    الجمود
Shared Data                                 بيانات مشتركة
Shared Data Structure                       هيكل بيانات مشترك
Shared Queue                                طابور مشترك
Linked List                                 قائمة مترابطة
Mutex                                       قفل الاستبعاد المتبادل
pthread_mutex_t                             pthread_mutex_t
lock()                                      lock()
unlock()                                    unlock()
Lock Contention                             تزاحم على القفل
Synchronization Overhead                    عبء المزامنة الإضافي
Lock Overhead                               عبء الأقفال الإضافي
Cache Affinity                              ألفة ذاكرة التخزين المؤقت
Affinity Mechanism                          آلية الألفة
Affinity Fairness                           عدالة الألفة
Single-Queue Scheduling                     الجدولة بطابور واحد
Single-Queue Multiprocessor Scheduling      الجدولة بطابور واحد للمعالجات المتعددة
SQMS                                        SQMS
Multi-Queue Scheduling                      الجدولة بطوابير متعددة
Multi-Queue Multiprocessor Scheduling       الجدولة بطوابير متعددة للمعالجات المتعددة
MQMS                                        MQMS
Load Imbalance                              عدم توازن الحمل
Load Balance                                توازن الحمل
Migration                                   هجرة
Continuous Migration                        هجرة مستمرة
Work Stealing                               سرقة العمل
Source Queue                                طابور مصدر
Target Queue                                طابور هدف
Peek (at another queue)                     نظرة خاطفة (على طابور آخر)
Threshold                                   عتبة
Black Art                                   فنّ أسود
Linux Multiprocessor Schedulers             مُجدوِلات لينكس للمعالجات المتعددة
O(1) Scheduler                              مُجدوِل O(1)
BF Scheduler                                مُجدوِل BF
BFS                                         BFS
Priority-based Scheduler                    مُجدوِل معتمد على الأولوية
Earliest Eligible Virtual Deadline First (EEVDF)  أقرب موعد افتراضي مؤهّل أولاً (EEVDF)
Eligible                                    مؤهّل
Virtual Deadline                            موعد افتراضي
Centralized Scheduler                       مُجدوِل مركزي
Per-CPU (scheduling)                        (جدولة) لكل CPU
Working Set                                 مجموعة عمل
Working-set Size                            حجم مجموعة العمل
Cache Size                                  حجم ذاكرة التخزين المؤقت
Warm Rate                                   معدل التسخين
Warmup Time                                 وقت التسخين
Warm (cache)                                دافئة (ذاكرة تخزين مؤقت)
Cold (cache)                                باردة (ذاكرة تخزين مؤقت)
Trace                                       تتبّع
Tick                                        نبضة (tick)
Tick-by-tick Trace                          تتبّع نبضة بنبضة
Time Left                                   الوقت المتبقي
Super-linear Speedup                        التسريع الفائق للخطّي
Speedup                                     تسريع
```

> `deadlock` وردت في المصدر مرتين: «الجمود» و«الجمود / التعطّل المتبادل بحسب
> السياق» — انظر تقرير التعارضات.

---

## سجلّ نقاط صرامة مهمّة (تحديث النسخة 11)

- **Job مقابل Task**
  - `Job` ← مهمة
  - `Task` ← مهمة فرعية
- **Runaway مقابل Errant**
  - `Runaway` ← جامحة (تستهلك الوقت بلا توقف)
  - `Errant` ← شاردة (تخطئ في المكان أو الصلاحيات)

---

*نهاية الجدول — النسخة الثانية عشرة (الصارمة)*
