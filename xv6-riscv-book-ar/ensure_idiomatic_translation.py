import os, glob, re

tex_dir = "/home/user/mostafa637/xv6-riscv-book-ar/tex"

# Let's inspect all tex files and ensure no awkward literal structures exist
literal_patterns = [
    (r'وظيفة نظام التشغيل هي', 'يتلخص الدور الجوهري لنظام التشغيل في'),
    (r'طريقة بدائية لتحقيق', 'منهجًا إبتدائيًا للتفاعل مع'),
    (r'يأخذ الشكل التقليدي لـ', 'يعتمد البنية التقليدية المستقرة لـ'),
    (r'تأخذ الشكل التقليدي', 'تعتمد البنية التقليدية'),
    (r'في الوقت نفسه', 'في آنٍ واحد'),
    (r'أمر ينطوي على صعوبة', 'تحديًا هندسيًا بالغ الدقة'),
]

files = glob.glob(os.path.join(tex_dir, "*.tex"))
for f in files:
    with open(f, 'r', encoding='utf-8') as fp:
        text = fp.read()
    
    modified = text
    for old_pat, new_phr in literal_patterns:
        modified = re.sub(old_pat, new_phr, modified)
    
    if modified != text:
        with open(f, 'w', encoding='utf-8') as fp:
            fp.write(modified)
        print(f"Refined idiomatic Arabic in {os.path.basename(f)}")

print("Idiomatic refinement complete.")
