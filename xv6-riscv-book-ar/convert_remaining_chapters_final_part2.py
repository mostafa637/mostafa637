import os

typst_dir = "/home/user/mostafa637/xv6-riscv-book-ar/typst"

# 9. sched_ar.typ
sched_typ = """#import "listings.typ": *

= المجدول وتنقيل السياق < CH:SCHED >

يتطلب تنفيذ مفهوم *التعددية المشاركة* (Multitasking) تحويل وحدة المعالجة المركزية (CPU) بين العمليات المختلفة لإعطاء الانطباع بأن جميع البرامج تعمل بالتوازي.
تُعرف عملية تبديل المعالج من عملية إلى أخرى بـ *تبديل السياق* (context switch).

يقدم هذا الفصل كيفية تبديل السياق في xv6، وهيكلة دالة المجدول (Scheduler)، وحالات العملية، والتفاعل بين المجدول وأقفال العمليات.

== تبديل السياق

يتطلب تحويل المعالج من عملية أولى إلى عملية ثانية حفظ حالة السجلات العتادية الخاصة بالعملية الأولى لاسترجاعها لاحقًا، وتحميل السجلات المحفوظة الخاصة بالعملية الثانية.

في xv6، لا يتم التبديل المباشر بين عملية ومستقلة، بل يتم التبديل بين:
1. سياق العملية الأولى إلى سياق المعالج المخصص في النواة (Kernel Scheduler Context).
2. اختيار عملية قابلة للتنفيذ.
3. سياق المعالج المخصص إلى سياق العملية الثانية.

تُحفظ السجلات المطلوبة لتبديل السياق داخل هيكل بيانات C يُدعى #lstinline("struct context") المعرف في #lstinline("kernel/proc.h") :
#lstlisting[
struct context {
  uint64 ra;  // Return Address
  uint64 sp;  // Stack Pointer

  // سجلات المحافظة (Callee-saved registers)
  uint64 s0;
  uint64 s1;
  uint64 s2;
  uint64 s3;
  uint64 s4;
  uint64 s5;
  uint64 s6;
  uint64 s7;
  uint64 s8;
  uint64 s9;
  uint64 s10;
  uint64 s11;
};
]

تُنفذ دالة التجميع #lstinline("swtch(struct context *old, struct context *new)") في #lstinline("kernel/swtch.S") الحفظ والاسترجاع الفعلي للسجلات على مكدس النواة:
1. تحفظ السجلات الكائنة في المعالج الحالي داخل الهيكل المومأ إليه بـ #lstinline("old") .
2. تحمل القيم المحفوظة في الهيكل #lstinline("new") إلى سجلات المعالج.
3. تعود بالتعليمة #lstinline("ret") مستخدمة القيمة المحدثة في سجل العنوان القادم #lstinline("ra") .

== الكود البرمجي: المجدول

تنفذ كل نوات معالجة في xv6 حلقة مجدول رئيسة داخل الدالة #lstinline("scheduler()") المحددة في #lstinline("kernel/proc.c") :
#lstlisting[
void
scheduler(void)
{
  struct proc *p;
  struct cpu *c = mycpu();
  
  c->proc = 0;
  for(;;){
    // تفعيل المقاطعات لتجنب الجمود أثناء الانتظار
    intr_on();

    for(p = proc; p < &proc[NPROC]; p++) {
      acquire(&p->lock);
      if(p->state == RUNNABLE) {
        // تحويل حالة العملية إلى جارية
        p->state = RUNNING;
        c->proc = p;

        swtch(&c->context, &p->context);

        // تمت العودة من العملية
        c->proc = 0;
      }
      release(&p->lock);
    }
  }
}
]

عندما ترغب عملية بالإنهاء أو التخلي عن وحدة المعالجة (مثل دعوة #lstinline("yield()") أو الدخول في النوم عبر #lstinline("sleep()") )، تقوم بدعوة #lstinline("sched()") :
تتحقق #lstinline("sched()") من شروط الأمان (حيازة #lstinline("p->lock") ، تعطيل المقاطعات التعددية)، ثم تنفذ #lstinline("swtch(&p->context, &c->context)") للعودة فورًا إلى حلقة #lstinline("scheduler()") الخاصة بالمعالج.

== تتبع تبديل السياق

تتبع مسار التبديل الكامل بين العملية A والعملية B:
1. تكون العملية A منفذة لكود مستخدم وتحدث مقاطعة مؤقت زمنية فتنتقل إلى #lstinline("usertrap()") .
2. تستدعي #lstinline("usertrap()") الدالة #lstinline("yield()") .
3. تحوز #lstinline("yield()") قفل العملية #lstinline("p->lock") ، وتغير حالة العملية إلى #lstinline("RUNNABLE") ، ثم تدعو #lstinline("sched()") .
4. تنفذ #lstinline("sched()") الدالة #lstinline("swtch(&p->context, &c->context)") فتنتقل النواة إلى المجدول عند التعليمة التالية لـ #lstinline("swtch") داخل #lstinline("scheduler()") .
5. يحرر المجدول قفل العملية A، ويبحث في مصفوفة العمليات حتى يجد العملية B القابلة للتنفيذ ( #lstinline("RUNNABLE") ).
6. يحوز المجدول قفل العملية B، ويغير حالتها إلى #lstinline("RUNNING") ، ويدعو #lstinline("swtch(&c->context, &p->context)") .
7. تعود #lstinline("swtch") بالمعالج إلى المكان الذي توقفت فيه العملية B سابقًا.

== التبديل وقفل العملية (p->lock)

يلعب القفل #lstinline("p->lock") دورًا محوريًا أثناء تبديل السياق:
- تحوز العملية القفل في الدالة #lstinline("yield()") أو #lstinline("sleep()") *قبل* تغيير حالتها و*قبل* دعوة #lstinline("swtch()") .
- يظل القفل محوزًا أثناء تنفيذ #lstinline("swtch()") !
- يقوم المجدول (في النواة المستقبِلة) بتحرير القفل #lstinline("release(&p->lock)") *بعد* انتهاء #lstinline("swtch()") .

يضمن هذا النموذج حماية حالة العملية ومنع نوات معالجة أخرى من اختيار نفس العملية والتنفيذ عليها في نفس اللحظة (حالة تسابق على تنفيذ نفس العملية).

== العالم الحقيقي

يستخدم xv6 المجدول بالدوران العام (Round-Robin Scheduler) لسهولة فهمه وبساطته. بينما تستخدم أنظمة التشغيل الحديثة مجدولات متعددة المستويات ومجهزة بالأولويات المقسمة، مثل مجدول العدالة الكاملة (Completely Fair Scheduler - CFS) في Linux، ومجدولات التغذية الراجعة متعددة المستويات (MLFQ).

== تمارين

1. تتبع القيم المخزنة في السجلين #lstinline("sp") و #lstinline("ra") خلال القفز بين #lstinline("swtch") والمجدول.
2. قم بتنفيذ مجدول يعتمد على أولوية العملية (Priority-based Scheduler) بدلاً من الدوران العام في xv6.
"""

with open(os.path.join(typst_dir, "sched_ar.typ"), "w", encoding="utf-8") as f:
    f.write(sched_typ)

# 10. sleep_ar.typ
sleep_typ = """#import "listings.typ": *

= تنسيق وإشارات التزامن عبر النوم واليقظة < CH:SLEEP >

تحتاج العمليات داخل النواة غالبًا إلى الانتظار لحدوث حدث معين (مثل قراءة بيانات من القرص الصلب، انتظار إدخال من الشاشة، أو انتظار انتهاء عملية ابن عبر #lstinline("wait()") ).

إذا انتظرت العملية باستخدام حلقة تكرارية نشطة (Spin-waiting)، فإنها تستنزف موارد المعالج دون جدوى.
يوفر xv6 آلية التنسيق *النوم واليقظة* (Sleep and Wakeup - أو أسلوب الحراس والشرطية)، التي تسمح للعملية بالتعليق والدخول في حالة النوم حتى يقوم جزء آخر من النواة بإيقاظها عند تحقق الشرط.

يقدم هذا الفصل آليات النوم واليقظة، مشكلة الاستيقاظ المفقود (Lost Wakeup)، وكيفية تنفيذ الأقفال النوامة، واستدعاءات النظام #lstinline("wait()") و #lstinline("exit()") و #lstinline("kill()") .

== النوم واليقظة والاستيقاظ المفقود

تعتمد الآلية البسيطة للتنسيق على وجود عنوان انتسساب أو قناة انتظار تُدعى *قناة النوم* (chan / sleep channel):
- #lstinline("sleep(chan, lock)") : تضع العملية الحالية في حالة النوم على العنوان #lstinline("chan") وتتخلى عن المعالج.
- #lstinline("wakeup(chan)") : توقظ كافة العمليات المنتظرة أو النائمة على العنوان #lstinline("chan") وتغير حالتها إلى #lstinline("RUNNABLE") .

لتجنب مشكلة *الاستيقاظ المفقود* (Lost Wakeup):
تحدث هذه المشكلة إذا قُطعت العملية أثناء فحص الشرط وقبل النوم مباشرةً، فتقوم دالة المقاطعة بفرز #lstinline("wakeup()") ولم تجد أي عملية نائمة بعد. بعد ذلك تنام العملية إلى الأبد.

لتجنب ذلك، تتطلب #lstinline("sleep()") حيازة قفل الشرط (Condition Lock) الممرر إليها:
تقوم #lstinline("sleep()") بحيازة قفل العملية الداخلي #lstinline("p->lock") ثم تحرير قفل الشرط الممرر بشكل ذري قبل دعوة #lstinline("sched()") .

== الكود البرمجي: تنفيذ النوم واليقظة

تنفذ الدالة #lstinline("sleep()") في #lstinline("kernel/proc.c") بالخطوات المحددة التالية:
#lstlisting[
void
sleep(void *chan, struct spinlock *lk)
{
  struct proc *p = myproc();
  
  // حيازة p->lock لحماية حالة العملية
  acquire(&p->lock);
  
  // تحرير قفل الشرط الممرر
  release(lk);

  // وضع القناة وتعديل الحالة إلى SLEEPING
  p->chan = chan;
  p->state = SLEEPING;

  sched();

  // بعد الاستيقاظ: تنظيف القناة وتحرير p->lock
  p->chan = 0;
  release(&p->lock);

  // إعادة حيازة قفل الشرط الأصلي
  acquire(lk);
}
]

وتقوم الدالة #lstinline("wakeup(chan)") بتمشيط جدول العمليات بحثًا عن العمليات الممتلكة لـ #lstinline("p->state == SLEEPING") والمطابقة لـ #lstinline("p->chan == chan") ، وتغير حالتها فورًا إلى #lstinline("RUNNABLE") .

== الكود البرمجي: أقفال النوم (Sleep-locks)

تُبنى أقفال النوم فوق آلية #lstinline("sleep()") و #lstinline("wakeup()") لحماية الموارد التي يتطلب حيازتها الانتظار لفترات طويلة:
#lstlisting[
void
acquiresleep(struct sleeplock *lk)
{
  acquire(&lk->lk);
  while (lk->locked) {
    sleep(lk, &lk->lk);
  }
  lk->locked = 1;
  lk->pid = myproc()->pid;
  release(&lk->lk);
}
]

== الكود البرمجي: wait و exit و kill

تتفاعل آليات النوم واليقظة لدعم دورة حياة العمليات:
- #lstinline("exit(status)") : تُستدعى لتدمير العملية الحالية. تغير حالة العملية إلى #lstinline("ZOMBIE") ، وتمرر أطفال العملية إلى العملية الأولى ( #lstinline("init") )، وتوقظ الأب المنتظر عبر #lstinline("wakeup(p->parent)") ثم تدعو #lstinline("sched()") دون العودة مطلقًا.
- #lstinline("wait(status)") : تبحث عن أي عملية ابن حالتها #lstinline("ZOMBIE") . إذا وجدتها، تنظف ذاكرة الابن ومكدسه وإطار مصيدته وترجع معرّف الابن #lstinline("pid") . وإذا كان للعملية أبناء ما زالوا يشتغلون، تنام العملية على عنوانها الذاتي #lstinline("sleep(p, &p->lock)") حتى ينهي أحدهم.
- #lstinline("kill(pid)") : تعين الراية #lstinline("p->killed = 1") للعملية المستهدفة وتوقظها فورًا إذا كانت نائمة. عند خروج العملية من المصيدة أو استدعاء النظام التالي، تفحص الراية وتنهي تنفيذها فورًا عبر #lstinline("exit(-1)") .

== العالم الحقيقي

تُعرف آليات النوم واليقظة في أدبيات أنظمة التشغيل بـ *المراقِبات* (Monitors) أو متغبرات الشرط (Condition Variables).
تستخدم الأنظمة الإنتاجية هياكل بيانات متقدمة تُدعى سلاسل الانتظار (Wait Queues) لربط القنوات بقوائم انتظار سريعة لتجنب المرور على جميع العمليات في النظام كما يكتفي xv6.

== تمارين

1. اشرح بالتفصيل كيف تمنع حيازة #lstinline("p->lock") و قفل الشرط حدوث مشكلة «الاستيقاظ المفقود».
2. قم بتعديل #lstinline("wakeup()") في xv6 لتستند على مصفوفة قوائم انتظار لكل قناة نوم لزيادة السرعة.
"""

with open(os.path.join(typst_dir, "sleep_ar.typ"), "w", encoding="utf-8") as f:
    f.write(sleep_typ)

print("sched_ar and sleep_ar converted.")
