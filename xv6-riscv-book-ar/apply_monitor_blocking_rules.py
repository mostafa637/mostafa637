import os, glob, re, json

glossary_md = "/home/user/mostafa637/xv6-riscv-book-ar/glossary/GLOSSARY.md"
termbase_json = "/home/user/mostafa637/xv6-riscv-book-ar/glossary/termbase.json"

# Update GLOSSARY.md
with open(glossary_md, 'a', encoding='utf-8') as f:
    f.write("\n\n---\n\n### قواعد مصطلحات الحجب والمراقب (Blocking & Monitors Rules)\n\n")
    f.write("| المصطلح الإنجليزي | المقابل العربي الصارم |\n")
    f.write("|---|---|\n")
    f.write("| **blocking** | حاجب |\n")
    f.write("| **non-blocking** | غير حاجب |\n")
    f.write("| **blocked** | محظور |\n")
    f.write("| **Monitor** | مراقِب |\n")
    f.write("| **Monitors** | مراقِبات |\n")
    f.write("| **monitor lock** | قفل المراقِب |\n")
    f.write("| **monitor synchronization** | مزامنة المراقِب |\n")

# Update termbase.json
with open(termbase_json, 'r', encoding='utf-8') as f:
    tb = json.load(f)

tb["terms"]["blocking"] = "حاجب"
tb["terms"]["non-blocking"] = "غير حاجب"
tb["terms"]["blocked"] = "محظور"
tb["terms"]["Monitor"] = "مراقِب"
tb["terms"]["Monitors"] = "مراقِبات"
tb["terms"]["monitor lock"] = "قفل المراقِب"
tb["terms"]["monitor synchronization"] = "مزامنة المراقِب"

with open(termbase_json, 'w', encoding='utf-8') as f:
    json.dump(tb, f, ensure_ascii=False, indent=2)

print("Updated termbase and glossary with blocking/monitor rules.")
