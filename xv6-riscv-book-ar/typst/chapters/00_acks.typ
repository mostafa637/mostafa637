#import "../template/listings.typ": *

= التصدير والشكر والتقدير
<CH:ACKS>

هذا نص مسودة مخصص لفصل دراسي بداخل نظم التشغيل.
يشرح المفاهيم الرئيسية لنظم التشغيل عبر دراسة نواة نموذجية تُسمى `xv6`.
نُمذت `xv6` بناءً على الإصدار السادس من Unix (v6) الكائن لـ Dennis Ritchie و Ken Thompson.
تتبع `xv6` بمرونة تركيب وأسلوب v6، ولكنها تُنفذ بـ ANSI C لمعالج RISC-V متعدد النوى.

ينبغي قراءة هذا النص جنباً إلى جنب مع الشفرة المصدرية لـ `xv6`، وهو نهج مستوحى من تعليقات John Lions على الإصدار السادس من UNIX؛ ويحتوي النص على وصلات شعبية للشفرة المصدرية عند #link("https://github.com/mit-pdos/xv6-riscv").
انظر #link("https://pdos.csail.mit.edu/6.1810") لمؤشرات إضافية للموارد على الشبكة لـ v6 و `xv6`، بما بداخل ذلك عدة واجبات معملية تستخدم `xv6`.

لقد استخدمنا هذا النص بداخل المقررات 6.828 و 6.1810، فصول نظم التشغيل بـ MIT.
نشكر أعضاء هيئة التدريس، المساعدين التعليميين، والطلاب بداخل تلك الفصول الذين ساهموا جميعاً مباشرة أو غير مباشرة بـ `xv6`.
وبصفة خاصة، نود شكر Adam Belay و Austin Clements و Nickolai Zeldovich.
أخيراً نود شكر الأشخاص الذين أرسلوا لنا بريداً إلكترونياً بأخطاء بداخل النص أو اقتراحات للتحسينات: Abutalib Aghayev, Sebastian Boehm, brandb97, Anton Burtsev, Raphael Carvalho, Tej Chajed, Brendan Davidson, Rasit Eskicioglu, Color Fuzzy, Wojciech Gac, Giuseppe, Tao Guo, Haibo Hao, Naoki Hayama, Chris Henderson, Robert Hilderman, Eden Hochbaum, Wolfgang Keller, Paweł Kraszewski, Henry Laih, Jin Li, Austin Liew, `lyazj@github.com`, Pavan Maddamsetti, Jacek Masiulaniec, Michael McConville, m3hm00d, Mes0903, miguelgvieira, Mark Morrissey, Muhammed Mourad, Harry Pan, Harry Porter, pr3pony, Siyuan Qian, Zhefeng Qiao, Askar Safin, Salman Shah, Huang Sha, Vikram Shenoy, Adeodato Simó, Ruslan Savchenko, Pawel Szczurko, Warren Toomey, tyfkda, tzerbib, unicornx, Vanush Vaswani, Chen Wang, Xi Wang, and Zou Chang Wei, Sam Whitlock, Qiongsi Wu, LucyShawYang, `ykf1114@gmail.com`, and Meng Zhou.

إذا لاحظت أخطاء أو حزت اقتراحات للتحسين، يُرجى إرسال بريد إلكتروني لـ Frans Kaashoek و Robert Morris على (`kaashoek,rtm@csail.mit.edu`).
