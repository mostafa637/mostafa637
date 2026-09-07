# جدول المصطلحات الموحَّد (صارم) — النسخة الحادية عشرة
### المصطلحات المعتمدة لكتاب أنظمة التشغيل ومقرر xv6 RISC-V

تم تحديث هذا المعجم ليلتزم بـ **جدول المصطلحات الموحَّد (صارم) — النسخة الحادية عشرة**:

#### القواعد الصارمة:
1. كل مصطلح إنجليزي → مقابل عربي ثابت.
2. اختلاف المصطلح الإنجليزي ⇒ لا يُعطى نفس المقابل العربي.
3. **Job** = مهمة | **Task** = مهمة فرعية.
4. **Errant** = شاردة | **Runaway** = جامحة.

---

### جدول المصطلحات المعتمدة:

| المصطلح الإنجليزي | المقابل العربي الصارم |
|---|---|
| **Virtualization** | الافتراضية |
| **Concurrency** | التزامن |
| **Persistence** | ثبات البيانات |
| **Operating System (OS)** | نظام التشغيل |
| **Kernel** | النواة |
| **Microkernel** | نواة مصغّرة |
| **Monolithic Kernel** | النواة المتجانسة |
| **Virtual Memory** | الذاكرة الافتراضية |
| **Virtual Address Space** | فضاء العناوين الافتراضي |
| **Address Space** | فضاء العناوين |
| **Virtual Address** | عنوان افتراضي |
| **Physical Address** | عنوان فيزيائي |
| **Physical Memory** | الذاكرة الفيزيائية |
| **Code Segment** | جزء الكود |
| **Stack** | المكدس |
| **Heap** | الكومة |
| **Swap Space** | مساحة المبادلة |
| **Memory Leak** | تسرب الذاكرة |
| **Segmentation Fault** | خطأ تقسيم |
| **Fragmentation** | التجزئه |
| **Buffer Overflow** | طفح المخزن المؤقت |
| **Process** | عملية |
| **Thread** | خيط |
| **Threads** | خيوط |
| **Context Switch** | تبديل السياق |
| **System Call** | استدعاء النظام |
| **Trap (instruction)** | المصيدة (تعليمة) |
| **Trap Handler** | معالج المصيدة |
| **User Mode** | وضع المستخدم |
| **Kernel Mode** | وضع النواة |
| **Supervisor** | المشرف |
| **File System** | نظام الملفات |
| **Persistent Storage** | تخزين دائم |
| **Journaling** | التدوين اليومي |
| **Copy-on-Write** | النسخ عند الكتابة |
| **Device Driver** | برنامج تشغيل الجهاز |
| **Race Condition** | حالة تسابق |
| **Atomic Operation** | عملية ذرّية |
| **Shell** | الشِل |
| **Pipe / Pipes** | أنبوب / أنابيب |
| **Page / Pages** | صفحة / صفحات |
| **Paging** | التقسيم إلى صفحات |
| **Swapping** | المبادلة |
| **Errant Process / Program** | عملية شاردة / برنامج شارد |
| **Runaway Process** | عملية جامحة |
| **Instruction Pointer (IP)** | مؤشر التعليمة (IP) |
| **Stack Pointer** | مؤشر المكدس |
| **Frame Pointer** | مؤشر الإطار |
| **Process Control Block (PCB)** | كتلة تحكّم العملية (PCB) |
| **Process Descriptor** | واصف العملية |
| **Parent Process** | العملية الأب |
| **Child Process** | العملية الابن |
| **Zombie State** | حالة الزومبي |
| **Kernel Stack** | مكدس النواة |
| **Trap Frame** | إطار المصيدة |
| **Limited Direct Execution (LDE)**| التنفيذ المباشر المحدود |
| **Restricted Operation / Instruction** | عملية مقيدة / تعليمة مقيدة |
| **Privileged Operation / Instruction** | عملية ذات امتيازات / تعليمة ذات امتيازات |
| **Exception** | استثناء |
| **Timer Interrupt** | مقاطعة المؤقت |
| **Interrupt Handler** | معالج مقاطعة |
| **Scheduling** | الجدولة |
| **Turnaround Time** | وقت الإنجاز |
| **Response Time** | وقت الاستجابة |
| **Round Robin (RR)** | جولة روبن (RR) |
| **Time Slice** | شريحة زمنية |
| **Multi-Level Feedback Queue (MLFQ)** | طابور التغذية الراجعة متعدد المستويات (MLFQ) |
| **Completely Fair Scheduler (CFS)** | المُجدوِل العادل تماماً (CFS) |
| **Job / Jobs** | مهمة / مهام |
| **Task / Tasks** | مهمة فرعية / مهام فرعية |
| **Mutual Exclusion** | الاستبعاد المتبادل |
| **Lock-free Data Structure** | هيكل بيانات خالٍ من الأقفال |
| **Deadlock** | الجمود |
| **Work Stealing** | سرقة العمل |

---

### قواعد مصطلح مخزن الترجمة المؤقت (TLB - Translation Lookaside Buffer)

| الصيغة | المقابل العربي الصارم |
|---|---|
| **أوّل ظهور** | «مخزن الترجمة المؤقت (Translation Lookaside Buffer، TLB)» |
| **مفرد معرَّف** | مخزن الترجمة المؤقت / المخزن |
| **نكرة** | مخزن ترجمة مؤقت |
| **جمع** | مخازن الترجمة المؤقتة |
| **TLB miss / hit** | إخفاق / إصابة في مخزن الترجمة المؤقت |


---

### قواعد مصطلحات الحجب والمراقب (Blocking & Monitors Rules)

| المصطلح الإنجليزي | المقابل العربي الصارم |
|---|---|
| **blocking** | حاجب |
| **non-blocking** | غير حاجب |
| **blocked** | محظور |
| **Monitor** | مراقِب |
| **Monitors** | مراقِبات |
| **monitor lock** | قفل المراقِب |
| **monitor synchronization** | مزامنة المراقِب |
