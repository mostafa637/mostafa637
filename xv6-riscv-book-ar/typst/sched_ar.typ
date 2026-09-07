#import "listings.typ": *

= المجدول وتنقيل السياق
<CH:SCHED>

يتطلب تنفيذ مفهوم *التعددية المشاركة* (Multitasking) تحويل وحدة المعالجة المركزية (CPU) بين العمليات المختلفة لإعطاء الانطباع بأن جميع البرامج تعمل بالتوازي.
تُعرف عملية تبديل المعالج من عملية إلى أخرى بـ *تبديل السياق* (Context Switch).

يقدم هذا الفصل كيفية تبديل السياق في #lstinline("xv6")، وهيكلة دالة المجدول (Scheduler)، وحالات العملية، والتفاعل بين المجدول وأقفال العمليات.

== تعدد المشاركة
<sec:multiplexing>

تحقق النواة التعددية المشاركة عبر فصل المعالجة الفيزيائية عن حالة العملية الافتراضية.
عندما تنتهي شريحة الوقت للعملية الحالية أو تتوقف منتظرة الإدخال والإخراج، تحفظ النواة سياق العملية وترتقي بعملية أخرى لتستغل المعالج.

== نظرة عامة على تبديل السياق
<sec:context_switch_overview>

يتطلب تحويل المعالج من عملية أولى إلى عملية ثانية حفظ حالة السجلات العتادية الخاصة بالعملية الأولى لاسترجاعها لاحقًا، وتحميل السجلات المحفوظة الخاصة بالعملية الثانية.

في #lstinline("xv6")، لا يتم التبديل المباشر بين عملية ومستقلة، بل يتم التبديل بين:
+ سياق العملية الأولى إلى سياق المعالج المخصص في النواة (Kernel Scheduler Context).
+ اختيار عملية قابلة للتنفيذ.
+ سياق المعالج المخصص إلى سياق العملية الثانية.

== الكود البرمجي: تبديل السياق
<sec:code_context_switching>

تُحفظ السجلات المطلوبة لتبديل السياق داخل هيكل بيانات C يُدعى #lstinline("struct context") المعرف في #lstinline("kernel/proc.h"):
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
+ تحفظ السجلات الكائنة في المعالج الحالي داخل الهيكل المومأ إليه بـ #lstinline("old").
+ تحمل القيم المحفوظة في الهيكل #lstinline("new") إلى سجلات المعالج.
+ تعود بالتعليمة #lstinline("ret") مستخدمة القيمة المحدثة في سجل العنوان القادم #lstinline("ra").

== الكود البرمجي: المجدول
<sec:code_scheduling>

تنفذ كل نوات معالجة في #lstinline("xv6") حلقة مجدول رئيسة داخل الدالة #lstinline("scheduler()") المحددة في #lstinline("kernel/proc.c"):
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

عندما ترغب عملية بالإنهاء أو التخلي عن وحدة المعالجة (مثل دعوة #lstinline("yield()") أو الدخول في النوم عبر #lstinline("sleep()"))، تقوم بدعوة #lstinline("sched()"):
تتحقق #lstinline("sched()") من شروط الأمان (حيازة #lstinline("p->lock")، تعطيل المقاطعات التعددية)، ثم تنفذ #lstinline("swtch(\&p->context, \&c->context)") للعودة فورًا إلى حلقة #lstinline("scheduler()") الخاصة بالمعالج.

== الكود البرمجي: mycpu و myproc
<sec:code_mycpu_myproc>

تسترجع النواة معرّف وحدة المعالجة الحالية عبر السجل العتادي #lstinline("tp") (Thread Pointer)، حيث تعيد الدالة #lstinline("mycpu()") مؤشر هيكل المعالج الحالي #lstinline("struct cpu")، بينما تعيد الدالة #lstinline("myproc()") مؤشر العملية المربوطة بالمعالج الحالية.

== العالم الحقيقي
<sec:real_world_sched>

يستخدم #lstinline("xv6") المجدول بالدوران العام (Round-Robin Scheduler) لسهولة فهمه وبساطته. بينما تستخدم أنظمة التشغيل الحديثة مجدولات متعددة المستويات ومجهزة بالأولويات المقسمة، مثل مجدول العدالة الكاملة (Completely Fair Scheduler - CFS) في Linux، ومجدولات التغذية الراجعة متعددة المستويات (MLFQ).

== تمارين
<sec:exercises_sched>

+ تتبع القيم المخزنة في السجلين #lstinline("sp") و #lstinline("ra") خلال القفز بين #lstinline("swtch") والمجدول.
+ قم بتنفيذ مجدول يعتمد على أولوية العملية (Priority-based Scheduler) بدلاً من الدوران العام في #lstinline("xv6").