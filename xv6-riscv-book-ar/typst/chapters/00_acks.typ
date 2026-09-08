#import "../template/listings.typ": *

= التصدير والشكر والتقدير
<CH:ACKS>

هذا النص عبارة عن مسودة مخصصة لمقرر دراسي في نظم التشغيل.
وهو يشرح المفاهيم الأساسية لنظم التشغيل من خلال دراسة نواة تطبيقية نموذجية تُسمى `xv6`.
صُممت `xv6` استناداً إلى الإصدار السادس من نظام تشغيل Unix (v6) المطور من قِبل دينيس ريتشي وكن ثومبسون.
وتتبع `xv6` البنية والأسلوب العام للإصدار السادس بمرونة، إلا أنها مُنفذة بلغة ANSI C ومخصصة لمعالجات RISC-V متعددة النوى.

يُوصى بقراءة هذا النص جنباً إلى جنب مع الشفرة المصدرية لنظام `xv6`؛ وهو منهج مستوحى من تعليقات جون لايونز على الإصدار السادس من UNIX؛ ويحتوي النص على روابط فائقة للشفرة المصدرية عبر #link("https://github.com/mit-pdos/xv6-riscv").
انظر #link("https://pdos.csail.mit.edu/6.1810") للحصول على إشارات إضافية للموارد المتاحة على الشبكة لكل من v6 و `xv6`، بما في ذلك العديد من الواجبات المعملية التي تستخدم `xv6`.

لقد استخدمنا هذا النص في المقررين 6.828 و 6.1810، وهما مقررا نظم التشغيل بمعهد ماساتشوستس للتكنولوجيا (MIT).
ونتوجه بالشكر لأعضاء هيئة التدريس، والمساعدين التعليميين، والطلاب في تلك المقررات الذين ساهموا جميعاً بشكل مباشر أو غير مباشر في تطوير `xv6`.
وبشكل خاص، نود أن نشكر كلاً من أدام بيلاي، وأوستن كليمنتس، ونيكولاي زيلدوفيتش.
وأخيراً، نود شكر كل من تواصل معنا عبر البريد الإلكتروني للإبلاغ عن أخطاء في النص أو تقديم اقتراحات للتحسين:
Abutalib Aghayev, Sebastian Boehm, brandb97, Anton Burtsev, Raphael Carvalho, Tej Chajed, Brendan Davidson, Rasit Eskicioglu, Color Fuzzy, Wojciech Gac, Giuseppe, Tao Guo, Haibo Hao, Naoki Hayama, Chris Henderson, Robert Hilderman, Eden Hochbaum, Wolfgang Keller, Paweł Kraszewski, Henry Laih, Jin Li, Austin Liew, `lyazj@github.com`, Pavan Maddamsetti, Jacek Masiulaniec, Michael McConville, m3hm00d, Mes0903, miguelgvieira, Mark Morrissey, Muhammed Mourad, Harry Pan, Harry Porter, pr3pony, Siyuan Qian, Zhefeng Qiao, Askar Safin, Salman Shah, Huang Sha, Vikram Shenoy, Adeodato Simó, Ruslan Savchenko, Pawel Szczurko, Warren Toomey, tyfkda, tzerbib, unicornx, Vanush Vaswani, Chen Wang, Xi Wang, and Zou Chang Wei, Sam Whitlock, Qiongsi Wu, LucyShawYang, `ykf1114@gmail.com`, and Meng Zhou.

إذا لاحظت أي أخطاء أو كانت لديك اقتراحات للتحسين، يُرجى إرسال بريد إلكتروني إلى فرانس كاشوك وروبيرت موريس عبر (`kaashoek,rtm@csail.mit.edu`).
