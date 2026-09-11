// Arabic translation — generated from the English Typst sources.
#import "../../../lib/sicp-ar.typ": *

#subsection([تجميع تسلسلات التعليمات], label-name: <sec:combining-instruction-sequences>)

#idx("instruction sequence")

وصف هذا القسم تفاصيل كيفية تمثيل وتجميع تسلسلات التعليمات. تذكر من القسم @sec:instruction-sequences أن تسلسل التعليمات يُمثل كقائمة بالمسجّلات المطلوبة، والمسجّلات المُعدلة، والتعليمات الفعلية. سنعتبر أيضًا التسمية (سلسلة نصية) حالة متدهورة لتسلسل تعليمات، لا تحتاج أو تعدل أي مسجّلات.
لذا لتحديد المسجّلات المطلوبة والمُعدلة بواسطة تسلسلات التعليمات نستخدم المُحددات:
#idx("registersneeded", decl: true)#idx("registersmodified", decl: true)#idx("instructions", decl: true)
#snippet(```python
function registers_needed(s) {
    return is_string(s) ? null : head(s);
}
function registers_modified(s) {
    return is_string(s) ? null : head(tail(s));
}
function instructions(s) {
    return is_string(s) ? list(s) : head(tail(tail(s)));
}
```)

وللتحديد ما إذا كان تسلسل معين يحتاج أو يعدل مسجّلاً معينًا نستخدم المحمولين:
#idx("needsregister", decl: true)#idx("modifiesregister", decl: true)
#snippet(```python
function needs_register(seq, reg) {
    return ! is_null(member(reg, registers_needed(seq)));
}
function modifies_register(seq, reg) {
    return ! is_null(member(reg, registers_modified(seq)));
}
```)

بدلالة هذه المحمولات والمُحددات، يمكننا تنفيذ مجمعات تسلسلات التعليمات المختلفة المستخدمة في جميع أنحاء المُترجِم.

المجمع الأساسي هو #py("append_instruction_sequences").
تأخذ هذه كوسائط تسلسلي تعليمات مراد تنفيذهما تتابعيًا وتُرجع تسلسل تعليمات تكون عباراته هي عبارات التسلسلين مجمعة معًا.
النقطة الدقيقة هي تحديد المسجّلات المطلوبة والمُعدلة بواسطة التسلسل الناتج.
فهي تعدل تلك المسجّلات التي تُعدل بواسطة أي من التسلسلين؛
وتحتاج إلى تلك المسجّلات التي يجب تهيئتها قبل تشغيل التسلسل الأول (المسجّلات المطلوبة بواسطة التسلسل الأول)، جنبًا إلى جنب مع تلك المسجّلات المطلوبة بواسطة التسلسل الثاني والتي لم تُهيأ (تُعدل) بواسطة التسلسل الأول.

تُعطى الدالة #py("append_instruction_sequences") تسلسلي تعليمات #py("seq1") و #py("seq2") وتُرجع تسلسل التعليمات الذي تعليماته هي تعليمات #py("seq1") متبوعة بتعليمات #py("seq2")، والذي مسجّلاته المُعدلة هي تلك المسجّلات التي تُعدل بواسطة إما #py("seq1") أو #py("seq2")، والذي مسجّلاته المطلوبة هي المسجّلات المطلوبة بواسطة #py("seq1") جنبًا إلى جنب مع تلك المسجّلات المطلوبة بواسطة #py("seq2") والتي لا تُعدل بواسطة #py("seq1"). (بدلالة عمليات المجموعات، فإن المجموعة الجديدة من المسجّلات المطلوبة هي اتحاد مجموعة المسجّلات المطلوبة بواسطة #py("seq1") مع الفرق المجموعي للمسجّلات المطلوبة بواسطة #py("seq2") والمسجّلات المُعدلة بواسطة #py("seq1")). وبالتالي، فإن #py("append_instruction_sequences") تُنفذ كما يلي:
#idx("appendinstructionsequences", decl: true)
#snippet(```python
function append_instruction_sequences(seq1, seq2) {
    return make_instruction_sequence(
               list_union(registers_needed(seq1),
                          list_difference(registers_needed(seq2),
                                          registers_modified(seq1))),
               list_union(registers_modified(seq1),
                          registers_modified(seq2)),
               append(instructions(seq1), instructions(seq2)));
}
```)

تستخدم هذه الدالة بعض العمليات البسيطة لمعالجة المجموعات المُمثلة كقوائم، بشكل يشبه تمثيل المجموعات (غير المرتبة) الموصوف في القسم @sec:representing-sets:
#idx("listunion", decl: true)#idx("listdifference", decl: true)
#snippet(```python
function list_union(s1, s2) {
    return is_null(s1)
           ? s2
           : is_null(member(head(s1), s2))
           ? pair(head(s1), list_union(tail(s1), s2))
           : list_union(tail(s1), s2);
}
function list_difference(s1, s2) {
    return is_null(s1)
           ? null
           : is_null(member(head(s1), s2))
           ? pair(head(s1), list_difference(tail(s1), s2))
           : list_difference(tail(s1), s2);
}
```)

الدالة #py("preserving")، المجمع الرئيسي الثاني لتسلسلات التعليمات، تأخذ قائمة مسجّلات #py("regs") وتسلسلي تعليمات #py("seq1") و #py("seq2") مراد تنفيذهما تتابعيًا. وتُرجع تسلسل تعليمات تكون تعليماته هي تعليمات #py("seq1") متبوعة بتعليمات #py("seq2")، مع تعليمات #py("save") و #py("restore") مناسبة حول #py("seq1") لحماية المسجّلات في #py("regs") التي تُعدل بواسطة #py("seq1") ولكنها مطلوبة بواسطة #py("seq2"). لتحقيق ذلك، تنشئ #py("preserving") أولاً تسلسلاً يحتوي على عمليات الـ #py("save") المطلوبة متبوعة بتعليمات #py("seq1") متبوعة بعمليات الـ #py("restore") المطلوبة. ينطوي هذا التسلسل على احتياج المسجّلات التي تم حفظها واستعادتها بالإضافة إلى المسجّلات المطلوبة بواسطة #py("seq1")، ويعدل المسجّلات المُعدلة بواسطة #py("seq1") باستثناء تلك التي تم حفظها واستعادتها. ثم يُلحق هذا التسلسل المعزز و #py("seq2") بالطريقة العادية. تنفذ الدالة التالية هذه الاستراتيجية عوديًا، مارةً على قائمة المسجّلات المراد حفظها:
#idx("preserving", decl: true)
#snippet(```python
function preserving(regs, seq1, seq2) {
    if (is_null(regs)) {
        return append_instruction_sequences(seq1, seq2);
    } else {
        const first_reg = head(regs);
        return needs_register(seq2, first_reg) &&
               modifies_register(seq1, first_reg)
               ? preserving(tail(regs),
                     make_instruction_sequence(
                         list_union(list(first_reg),
                                    registers_needed(seq1)),
                         list_difference(registers_modified(seq1),
                                         list(first_reg)),
                         append(list(save(first_reg)),
                                append(instructions(seq1),
                                       list(restore(first_reg))))),
                     seq2)
               : preserving(tail(regs), seq1, seq2);
    }
}
```)

مجمع تسلسلات آخر، وهو #py("tack_on_instruction_sequence")، يُستخدم بواسطة #py("compile_lambda_expression") لإلحاق جسم دالة بتسلسل آخر. نظرًا لأن جسم الدالة ليس "في السطر" ليتم تنفيذه كجزء من التسلسل المركب، فإن استخدام المسجّلات الخاص به ليس له أي تأثير على استخدام المسجّلات للتسلسل الذي تضمن فيه. وبذلك نتجاهل مجموعات المسجّلات المطلوبة والمُعدلة لجسم الدالة عندما نثبته في التسلسل الآخر.
#idx("tackoninstructionsequence", decl: true)
#snippet(```python
function tack_on_instruction_sequence(seq, body_seq) {
    return make_instruction_sequence(
               registers_needed(seq),
               registers_modified(seq),
               append(instructions(seq), instructions(body_seq)));
}
```)

تستخدم الدالتان #py("compile_conditional") و #py("compile_function_call") مجمعًا خاصًا يسمى #py("parallel_instruction_sequences") لإلحاق فرعي الخيارين اللذين يليان اختبارًا. لن يُنفذ الفرعان تتابعيًا أبدًا؛ لأي تقييم معين للاختبار، سيتم الدخول في أحد الفرعين أو الآخر. ولهذا السبب، فإن المسجّلات المطلوبة بواسطة الفرع الثاني تظل مطلوبة بواسطة التسلسل المركب، حتى لو كانت هذه المكتوبة تُعدل بواسطة الفرع الأول.
#idx("parallelinstructionsequences", decl: true)
#snippet(```python
function parallel_instruction_sequences(seq1, seq2) {
    return make_instruction_sequence(
               list_union(registers_needed(seq1),
                          registers_needed(seq2)),
               list_union(registers_modified(seq1),
                          registers_modified(seq2)),
               append(instructions(seq1), instructions(seq2)));
}
```)

#idx("instruction sequence")
