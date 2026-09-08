import os, glob, re

tex_dir = "/home/user/mostafa637/xv6-riscv-book-ar/tex"
glossary_md = "/home/user/mostafa637/xv6-riscv-book-ar/glossary/GLOSSARY.md"
termbase_json = "/home/user/mostafa637/xv6-riscv-book-ar/glossary/termbase.json"

# Update mem_ar.tex
mem_path = os.path.join(tex_dir, "mem_ar.tex")
with open(mem_path, 'r', encoding='utf-8') as f:
    text = f.read()

old_tlb = r"لتسريع ترجمة العناوين وتجنب الوصول المكرر للذاكرة، يحتوي المعالج على ذاكرة مخبئية عتادية تُسمى \\indextext\{مخبأ ترجمة العناوين\} \(TLB - Translation Lookaside Buffer\)\. وعند تعديل جدول الصفحات، يجب على النواة فرز ومسح الـ TLB باستخدام تعليمة \\lstinline\{sfence\.vma\}\."

new_tlb = r"لتسريع ترجمة العناوين وتجنب الوصول المكرر للذاكرة، يحتوي المعالج على عتاد خاص يُسمى «مخزن الترجمة المؤقت (Translation Lookaside Buffer، TLB)». وعند تعديل جدول الصفحات، يجب على النواة فرز ومسح مخزن الترجمة المؤقت باستخدام تعليمة \lstinline{sfence.vma}."

text = re.sub(r'مخبأ ترجمة العناوين', 'مخزن الترجمة المؤقت', text)
text = re.sub(r'مخابئ ترجمة العناوين', 'مخازن الترجمة المؤقتة', text)
text = re.sub(r'تقليل ضغط الـ TLB', 'تقليل الضغط على مخزن الترجمة المؤقت', text)
text = text.replace("حتي يحتوي المعالج على ذاكرة مخبئية عتادية تُسمى \\indextext{مخبأ ترجمة العناوين} (TLB - Translation Lookaside Buffer).", "يحتوي المعالج على «مخزن الترجمة المؤقت (Translation Lookaside Buffer، TLB)».")
text = re.sub(r'يحتوي المعالج على ذاكرة مخبئية عتادية تُسمى.*?تعديل جدول الصفحات', r'يحتوي المعالج على «مخزن الترجمة المؤقت (Translation Lookaside Buffer، TLB)». وعند تعديل جدول الصفحات', text, flags=re.DOTALL)

with open(mem_path, 'w', encoding='utf-8') as f:
    f.write(text)

print("Updated mem_ar.tex with exact TLB specification.")
