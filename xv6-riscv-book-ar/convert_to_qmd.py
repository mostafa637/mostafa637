import re, os

tex_dir = "/home/user/mostafa637/xv6-riscv-book-ar/tex"
qmd_dir = "/home/user/mostafa637/xv6-riscv-book-ar/qmd"

mapping = {
    "acks_ar.tex": "index.qmd",
    "unix_ar.tex": "unix.qmd",
    "first_ar.tex": "first.qmd",
    "mem_ar.tex": "mem.qmd",
    "trap_ar.tex": "trap.qmd",
    "pgfault_ar.tex": "pgfault.qmd",
    "interrupt_ar.tex": "interrupt.qmd",
    "lock_ar.tex": "lock.qmd",
    "sched_ar.tex": "sched.qmd",
    "sleep_ar.tex": "sleep.qmd",
    "fs_ar.tex": "fs.qmd",
    "log_ar.tex": "log.qmd",
    "lock2_ar.tex": "lock2.qmd",
    "sum_ar.tex": "sum.qmd"
}

def latex_to_qmd(text):
    text = re.sub(r'\\chapter\*?\{([^}]+)\}', r'# \1', text)
    text = re.sub(r'\\section\{([^}]+)\}', r'## \1', text)
    text = re.sub(r'\\subsection\{([^}]+)\}', r'### \1', text)
    text = re.sub(r'\\label\{([^}]+)\}', r'', text)
    text = re.sub(r'\\indextext\{([^}]+)\}', r'**\1**', text)
    text = re.sub(r'\\index\{([^}]+)\}', r'', text)
    text = re.sub(r'\\indexcode\{([^}]+)\}', r'`\1`', text)
    text = re.sub(r'\\lstinline\{([^}]+)\}', r'`\1`', text)
    text = re.sub(r'\\textbf\{([^}]+)\}', r'**\1**', text)
    text = re.sub(r'\\textit\{([^}]+)\}', r'*\1*', text)
    text = re.sub(r'\\url\{([^}]+)\}', r'[\1](\1)', text)
    text = re.sub(r'\\cite\{([^}]+)\}', r'[\1]', text)
    text = re.sub(r'\\ref\{([^}]+)\}', r'[\1]', text)
    
    text = re.sub(r'\\begin\{lstlisting\}', '```c', text)
    text = re.sub(r'\\end\{lstlisting\}', '```', text)
    
    text = re.sub(r'\\begin\{itemize\}', '', text)
    text = re.sub(r'\\end\{itemize\}', '', text)
    text = re.sub(r'\\begin\{enumerate\}', '', text)
    text = re.sub(r'\\end\{enumerate\}', '', text)
    text = re.sub(r'\\item\s+', '* ', text)
    
    text = re.sub(r'\\begin\{figure\}(.*?)\\end\{figure\}', r'\1', text, flags=re.DOTALL)
    text = re.sub(r'\\caption\{([^}]+)\}', r'*\1*', text)
    text = re.sub(r'\\center', '', text)
    text = re.sub(r'\\includegraphics\[.*?\]\{([^}]+)\}', r'![](\1)', text)
    text = re.sub(r'\\begin\{tabular\}\{.*?\}', '', text)
    text = re.sub(r'\\end\{tabular\}', '', text)
    text = re.sub(r'\\midrule', '---', text)
    
    return text

for tex_file, qmd_file in mapping.items():
    tex_path = os.path.join(tex_dir, tex_file)
    qmd_path = os.path.join(qmd_dir, qmd_file)
    if os.path.exists(tex_path):
        with open(tex_path, 'r', encoding='utf-8') as f:
            content = f.read()
        converted = latex_to_qmd(content)
        with open(qmd_path, 'w', encoding='utf-8') as f:
            f.write(converted)
        print(f"Generated {qmd_file}")

print("QMD generation complete.")
