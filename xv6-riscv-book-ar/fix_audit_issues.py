import os, glob, re

tex_dir = "/home/user/mostafa637/xv6-riscv-book-ar/tex"

replacements = [
    (r'فضاء العناوين الظاهري', 'فضاء العناوين الافتراضي'),
    (r'فضاء عناوين ظاهري', 'فضاء عناوين افتراضي'),
    (r'الذاكرة الظاهرية', 'الذاكرة الافتراضية'),
    (r'ذاكرة ظاهرية', 'ذاكرة افتراضية'),
    (r'عنوان ظاهري', 'عنوان افتراضي'),
    (r'عناوين ظاهرية', 'عناوين افتراضية'),
    (r'العنوان الظاهري', 'العنوان الافتراضي'),
    (r'العناوين الظاهرية', 'العناوين الافتراضية'),
    (r'النواة الأحادية', 'النواة المتجانسة'),
    (r'محرك الجهاز', 'برنامج تشغيل الجهاز'),
    (r'محركات الأجهزة', 'برامج تشغيل الأجهزة'),
    (r'المأزق المغلق', 'الجمود'),
    (r'المآزق المغلقة', 'حالات الجمود'),
]

tex_files = glob.glob(os.path.join(tex_dir, "*.tex"))
for tf in tex_files:
    with open(tf, 'r', encoding='utf-8') as f:
        text = f.read()
    
    modified = text
    for old_term, new_term in replacements:
        modified = re.sub(old_term, new_term, modified)
    
    if modified != text:
        with open(tf, 'w', encoding='utf-8') as f:
            f.write(modified)
        print(f"Updated {os.path.basename(tf)}")

print("Fix script completed.")
