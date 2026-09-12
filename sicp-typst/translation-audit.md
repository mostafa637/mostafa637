# مقارنة الترجمة العربية بالأصل الإنجليزي وجدول المصطلحات

التدقيق آلي: يقارن كل ملف إنجليزي بنظيره العربي (التغطية والبنية)، ويتتبع 
كل مصطلح من «جدول المصطلحات الموحَّد» حاضر في نثر الأصل — ويفتش عن مقابل 
الجدول في النثر العربي (مع تجاهل الكود ومفاتيح الفهرس التي تبقى إنجليزية عمدًا). 
المطابقة مرنة صرفيًا (إسقاط التشكيل، توحيد الألف/الياء/التاء المربوطة، سقوط «ال» 
وتصريف «ة→ات»)، فالرايات تستدعي مراجعة بشرية لا حكمًا نهائيًا.

## ١. التغطية

- أزواج ملفات (إنجليزي ↔ عربي): **125**
- ملفات إنجليزية بلا مقابل عربي: **6** — `others/02foreword02.typ`, `others/03prefaces03.typ`, `others/04acknowledgements04.typ`, `others/06see06.typ`, `others/98indexpreface98.typ`, `others/99making99.typ`
- ملفات عربية بلا أصل: **0** — —

- تعارض في المرساة (labels): **0** ملفًا
- تعارض في العدّادات (عناوين/تمارين/حواشي/مقتطفات/فهرس): **5** ملفًا
  - `chapter1/section2/section2.typ`: index: أصلي 2 / مترجم 3
  - `chapter1/section2/subsection4.typ`: snippets: أصلي 5 / مترجم 6
  - `chapter1/section3/subsection1.typ`: snippets: أصلي 15 / مترجم 17
  - `chapter1/section3/subsection2.typ`: snippets: أصلي 19 / مترجم 21
  - `chapter4/section3/subsection1.typ`: index: أصلي 52 / مترجم 53

| العدّاد | الأصل | الترجمة |
|---|---|---|
| العناوين | 309 | 309 |
| التمارين | 356 | 356 |
| الحواشي | 332 | 332 |
| مقتطفات الكود | 1164 | 1169 |
| مداخل الفهرس | 3665 | 3667 |

## ٢. التزام المصطلح مع الجدول

مصطلحات الجدول الحاضرة في نثر الأصل: **188**؛ منها متوافق (ظهر المقابل في ≥٥٠٪ من المواضع): **158**؛ ضعيف التغطية: **10**؛ بلا مقابل ظاهر: **20**.

### بلا مقابل ظاهر في الترجمة (مرشّحة أولى للمراجعة)

| السطر في الجدول | المصطلح | مقابل الجدول | مواضع في الأصل |
|---|---|---|---|
| 316 | Sparse (address space) | متخلخل | 5 |
| 645 | Jobs | مهام | 3 |
| 415 | Blocked (state) | محظورة | 2 |
| 397 | Exit (process) | خروج (عملية) | 2 |
| 762 | Traverse (a list) | اجتياز (قائمة) | 2 |
| 680 | (to) Starve | يُجوّع | 1 |
| 860 | Bus | ناقل | 1 |
| 317 | Contiguous | متصل | 1 |
| 510 | Flag (command-line option) | خيار (Option) | 1 |
| 148 | I/O (Input/Output) | إدخال / إخراج | 1 |
| 224 | Interrupt | مقاطعة | 1 |
| 225 | Interrupts | مقاطعات | 1 |
| 398 | Kill (a process) | قتل (عملية) | 1 |
| 336 | Megabyte (MB) | ميجابايت | 1 |
| 869 | Monitors | مراقِبات | 1 |
| 300 | Multics | مولتيكس | 1 |
| 626 | Time-slicing | التقطيع الزمني | 1 |
| 246 | Trap (instruction) | المصيدة (تعليمة) | 1 |
| 292 | UNIX | يونكس | 1 |
| 267 | Volatile (storage) | متطاير (تخزين) | 1 |

### ضعيفة التغطية (المقابل ظهر في أقل من نصف المواضع)

| السطر | المصطلح | مقابل الجدول | الأصل | ظهر المقابل |
|---|---|---|---|---|
| 858 | Address | عنوان | 37 | 16 |
| 315 | Arbitrary (address) | اعتباطي | 21 | 7 |
| 382 | physical (adjective) | مادى (صفة) | 15 | 1 |
| 378 | Resource | مورد | 14 | 2 |
| 722 | Share (of CPU time) | حصّة (من وقت المعالج) | 11 | 1 |
| 644 | Job | مهمة | 9 | 4 |
| 318 | Convention | اصطلاح | 7 | 2 |
| 403 | Resume (a process) | استئناف (عملية) | 7 | 2 |
| 484 | Prompt | موجّه (Prompt) | 6 | 1 |
| 414 | Ready (state) | جاهزة | 4 | 1 |

### مصطلحات اللاتينية فقط في الجدول (تبقى كما هي، لا تُدقق)

`malloc()`، `new`، `RAID`، `open()`، `read()`، `write()`، `close()`، `exit()`، `getpid()`، `GNU`، `grep`، `ps auxw`، `Python`، `process-run.py`، `fork.py`، `scheduler.py`، `mlfq.py`، `lottery.py`، `multi.py`، `xv6`، `fork()`، `exec()`، `execvp()`، `execl()`، `execle()`، `execlp()`، `execv()`، `execvpe()`، `wait()`، `waitpid()`، `kill()`، `signal()`، `SIGINT`، `SIGTSTP`، `control-c`، `control-z`، `fg (shell built-in)`، `STDIN_FILENO`، `STDOUT_FILENO`، `STDERR_FILENO`، `pipe()`، `RTFM`، `shutdown`، `ps`، `top`، `killall`، `spawn()`، `lmbench`، `gettimeofday()`، `rdtsc`، `sched_setaffinity()`، `nice (utility)`، `madvise`، `remove_min(queue)`، `insert(queue, curr)`، `vruntime`، `sched_latency`، `min_granularity`، `prio_to_weight`، `ESX Server`، `VMWare`، `pthread_mutex_t`، `lock()`، `unlock()`، `SQMS`، `MQMS`، `BFS`

## ٣. بقايا إنجليزية للمصطلحات (خارج الكود والفهرس والمراجع)

### إنجليزية مكشوفة في النثر (مرشّحة للترجمة أو القرار)

| المصطلح | الملف | السياق |
|---|---|---|
| Abstraction | `chapter2/chapter2-ar.typ` | …قوية تسمى   #emph[تجريد البيانات (data abstraction)]. سنرى كيف يجعل تجريد البيانات تصميم ا… |
| ACM (Association for Computing Machinery) | `chapter3/section5/subsection5-ar.typ` | …الدالية عندما تُمّ منحه جائزة تورينغ من ACM عام 1978. وخطاب قبوله   (باكوس 1978) دا… |
| First In, First Out (FIFO) | `chapter3/section3/subsection2-ar.typ` | …ر أحياناً بـ   ذاكرة مؤقتة من نوع #emph[FIFO] (الأول دخولاً، الأول خروجاً -  ).… |
| List (data structure) | `chapter2/section1/subsection3-ar.typ` | …فء للأزواج قد يستخدم قائمة بايثون #emph[list] الأصلية) بل أنها يمكن أن تعمل بهذه الط… |
| Multics | `chapter5/section3/subsection2-ar.typ` | …خدام في تنفيذ Lisp لنظام مشاركة الوقت   Multics. لاحقًا، طور   بيكر (1978) نسخة   ( ) م… |
| Process | `chapter2/chapter2-ar.typ` | …على العمليات الحسابية (computational processes) وعلى دور الدوال في تصميم البرامج. رأين… |
| Python | `chapter2/section5/subsection1-ar.typ` | …لى حزمة  : magnitude(z) z apply_generic Python internal type system data types in Pyth… |
| Python | `chapter3/section5/subsection4-ar.typ` | …تبة أيضاً.#footnote[هذا انعكاس صغير، في Python، للصعوبات التي واجهتها           لغات ا… |
| Python | `chapter4/section1/subsection1-ar.typ` | …لمثل، لا يفرض مُقيِّمنا القيد النحوي لـ Python بأن العبارات لا يمكن أن تظهر داخل التعب… |
| Register | `chapter5/section1/subsection1-ar.typ` | …(في مسارات البيانات)  t<-r rem t  و rem register-machine language assign assign (in regi… |

### بين قوسين بوصفها شرحًا ثنائي اللغة (للاطلاع لا للتبديل)

| المصطلح | الملف | السياق |
|---|---|---|
| Abstraction | `chapter5/section1/subsection2-ar.typ` | …ببايثون كعمليات أولية. مثل هذا التجريد (abstraction) ذو قيمة لأنه يسمح لنا بتجاهل تفاصيل أج… |
| Action | `chapter5/section1/subsection1-ar.typ` | …ذا النوع من العمليات على أنه #emph[فعل (action)]. سنمثل الفعل في مخطط مسار البيانات تم… |
| Entry Point | `chapter5/section1/subsection1-ar.typ` | …ات (labels)] تحدد   #emph[نقاط الإدخال (entry points)] في التسلسل. التعليمة هي واحدة مما يلي… |
| Primitive | `chapter2/section5/subsection1-ar.typ` | …e system data types in Python isnumber (primitive function) data types and isstring (prim… |
| Primitive | `chapter5/section1/section1-ar.typ` | …في الوقت الحالي أن لدينا جهازًا أوليًا (primitive device) يحسب البواقي. في كل دورة من خوا… |

## ٤. ملاحظة على الجدول نفسه

`book-glossary/tools/check_glossary.py` يُبلغ عن تعارضَين من قاعدة «مقابل عربي 
واحد لمصطلح إنجليزي واحد» (آلة افتراضية مشتركة بين Virtual Machine وVM، وتتبّع 
مشتركة بين صيغتي Trace) — أُدرجا في الجدول عمدًا بوصفهما اختصارًا وصيغتين، والقرار 
للمحرِّر. التدقيق أعلاه يستعمل كلا الصيغتين عند البحث.
