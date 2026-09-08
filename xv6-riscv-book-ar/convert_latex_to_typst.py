import os, glob, re

tex_dir = "/home/user/mostafa637/xv6-riscv-book-ar/tex"
typst_dir = "/home/user/mostafa637/xv6-riscv-book-ar/typst"

mapping = {
    "acks_ar.tex": "acks_ar.typ",
    "unix_ar.tex": "unix_ar.typ",
    "first_ar.tex": "first_ar.typ",
    "mem_ar.tex": "mem_ar.typ",
    "trap_ar.tex": "trap_ar.typ",
    "pgfault_ar.tex": "pgfault_ar.typ",
    "interrupt_ar.tex": "interrupt_ar.typ",
    "lock_ar.tex": "lock_ar.typ",
    "sched_ar.tex": "sched_ar.typ",
    "sleep_ar.tex": "sleep_ar.typ",
    "fs_ar.tex": "fs_ar.typ",
    "log_ar.tex": "log_ar.typ",
    "lock2_ar.tex": "lock2_ar.typ",
    "sum_ar.tex": "sum_ar.typ"
}

def latex_to_typst(text):
    # Remove documentclass/input if present
    text = re.sub(r'\\chapter\*?\{([^}]+)\}', r'= \1', text)
    text = re.sub(r'\\section\{([^}]+)\}', r'== \1', text)
    text = re.sub(r'\\subsection\{([^}]+)\}', r'=== \1', text)
    text = re.sub(r'\\label\{([^}]+)\}', r'< \1 >', text)
    text = re.sub(r'\\indextext\{([^}]+)\}', r'*\1*', text)
    text = re.sub(r'\\index\{([^}]+)\}', r'', text)
    text = re.sub(r'\\indexcode\{([^}]+)\}', r'`\1`', text)
    text = re.sub(r'\\lstinline\{([^}]+)\}', r'#lstinline("\1")', text)
    text = re.sub(r'\\textbf\{([^}]+)\}', r'*\1*', text)
    text = re.sub(r'\\textit\{([^}]+)\}', r'_\1_', text)
    text = re.sub(r'\\url\{([^}]+)\}', r'#link("\1")[\1]', text)
    text = re.sub(r'\\cite\{([^}]+)\}', r'[\1]', text)
    text = re.sub(r'\\ref\{([^}]+)\}', r'@\1', text)
    
    # lstlisting blocks converted to #lstlisting[...]
    text = re.sub(r'\\begin\{lstlisting\}', '#lstlisting[', text)
    text = re.sub(r'\\end\{lstlisting\}', ']', text)
    
    # Lists
    text = re.sub(r'\\begin\{itemize\}', '', text)
    text = re.sub(r'\\end\{itemize\}', '', text)
    text = re.sub(r'\\begin\{enumerate\}', '', text)
    text = re.sub(r'\\end\{enumerate\}', '', text)
    text = re.sub(r'\\item\s+', '- ', text)
    
    # Figures and tables
    text = re.sub(r'\\begin\{figure\}(.*?)\\end\{figure\}', r'\1', text, flags=re.DOTALL)
    text = re.sub(r'\\caption\{([^}]+)\}', r'#figure(caption: [\1])[]', text)
    text = re.sub(r'\\center', '', text)
    text = re.sub(r'\\includegraphics\[.*?\]\{([^}]+)\}', r'#image("\1")', text)
    text = re.sub(r'\\begin\{tabular\}\{.*?\}', '', text)
    text = re.sub(r'\\end\{tabular\}', '', text)
    text = re.sub(r'\\midrule', '', text)
    
    return text

for tex_file, typst_file in mapping.items():
    tex_path = os.path.join(tex_dir, tex_file)
    typst_path = os.path.join(typst_dir, typst_file)
    if os.path.exists(tex_path):
        with open(tex_path, 'r', encoding='utf-8') as f:
            content = f.read()
        converted = latex_to_typst(content)
        with open(typst_path, 'w', encoding='utf-8') as f:
            f.write(converted)
        print(f"Generated {typst_file}")

print("Typst conversion complete.")
