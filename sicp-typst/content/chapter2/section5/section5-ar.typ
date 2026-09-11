// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp-ar.typ": *

#section([أنظمة ذات عمليات عامة], label-name: <sec:generic-operators>)

في القسم السابق، رأينا كيفية تصميم أنظمة يمكن فيها تمثيل كائنات البيانات بأكثر من طريقة واحدة. وكانت الفكرة الرئيسية هي ربط الشفرة التي تحدد عمليات البيانات بالتمثيلات المعددة بوساطة دوال واجهة عامة.
والآن سنرى كيفية استخدام الفكرة نفسها ليس فقط لتعريف عمليات عامة عبر تمثيلات مختلفة، بل وأيضاً لتعريف عمليات عامة عبر أنواع مختلفة من الوسائط (#idx("arithmetic", sub: "generic")). وقد رأينا بالفعل عدة حزم مختلفة من العمليات الحسابية: الحساب الأولي
(#py("+")، #py("-")، #py("*")، #py("/"))
المدمج في لغتنا، وحساب الأعداد الكسرية
(#py("add_rat")، #py("sub_rat")، #py("mul_rat")، #py("div_rat"))
من القسم @sec:rationals، وحساب الأعداد المركبة الذي نفذناه في القسم @sec:data-directed. وسوف نستخدم الآن تقنيات موجهة بالبيانات لبناء حزمة عمليات حسابية تتضمن جميع الحزم الحسابية التي بنيناها بالفعل.

يُظهر الشكل @fig:generic-system بنية النظام الذي سنبنيه. لاحظ #idx("abstraction barriers", sub: "in generic arithmetic system") حواجز التجريد. فمن منظور شخص يستخدم "الأعداد"، هناك دالة واحدة #py("add") تعمل على أي أعداد تُقدم لها.
والدالة #py("add") هي جزء من واجهة عامة تسمح بالوصول الموحد إلى حزم الحساب العادي، وحساب الكسريات، والحساب المركب المنفصلة بوساطة البرامج التي تستخدم الأعداد. وأي حزمة حسابية فردية (مثل الحزمة المركبة) قد يُوصل إليها بنفسها من خلال دوال عامة (مثل #py("add_complex")) تجمع بين الحزم المصممة لتمثيلات مختلفة (مثل المستطيلي والقطبي). علاوة على ذلك، فإن بنية النظام جمعية، بحيث يمكن تصميم الحزم الحسابية الفردية بشكل منفصل ودمجها لإنتاج نظام حسابي عام.
#idx("message passing")

#sicp-figure(image("/images/img_javascript/ch2-Z-G-64.svg", width: 70%), caption: [نظام (#idx("generic arithmetic operations", sub: "structure of system")) الحساب العام.], label-name: <fig:generic-system>)

#include "../../chapter2/section5/subsection1-ar.typ"

#include "../../chapter2/section5/subsection2-ar.typ"

#include "../../chapter2/section5/subsection3-ar.typ"

