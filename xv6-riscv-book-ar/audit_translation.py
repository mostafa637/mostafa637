import os, glob, re

tex_dir = "/home/user/mostafa637/xv6-riscv-book-ar/tex"
qmd_dir = "/home/user/mostafa637/xv6-riscv-book-ar/qmd"

# Strict terms checks
checks = {
    r'الذاكرة الظاهرية': 'الذاكرة الافتراضية (OSTEP Strict Rule)',
    r'عنوان ظاهري': 'عنوان افتراضي (OSTEP Strict Rule)',
    r'فضاء العناوين الظاهري': 'فضاء العناوين الافتراضي (OSTEP Strict Rule)',
    r'نواة أحادية': 'النواة المتجانسة (OSTEP Strict Rule)',
    r'محرك الجهاز': 'برنامج تشغيل الجهاز (OSTEP Strict Rule)',
    r'المأزق المغلق': 'الجمود (OSTEP Strict Rule)',
}

issues = []

tex_files = glob.glob(os.path.join(tex_dir, "*.tex"))
for tf in tex_files:
    fname = os.path.basename(tf)
    with open(tf, 'r', encoding='utf-8') as f:
        text = f.read()
    for pattern, rule in checks.items():
        matches = re.findall(pattern, text)
        if matches:
            issues.append(f"[{fname}] Found '{pattern}' -> Should be: {rule} ({len(matches)} occurrences)")

print("=== TRANSLATION AUDIT REPORT ===")
if not issues:
    print("ALL CLEAR! No termbase violations found.")
else:
    print(f"Found {len(issues)} issues to fix:")
    for issue in issues:
        print(" - " + issue)
