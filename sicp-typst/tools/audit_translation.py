#!/usr/bin/env python3
"""Audit the Arabic SICP translation against the English original and the
unified glossary (book-glossary/GLOSSARY.md).

Three checks, in order:

1. Coverage -- every English content file with an Arabic counterpart, and the
   structural parity of each pair: labels, headings, exercises, footnotes,
   snippets, index entries.
2. Terminology -- every glossary term that occurs in the English prose must
   occur (in its glossary Arabic form, modulo inflection) in the paired
   Arabic prose. Prose is estimated by stripping code fences, quoted strings,
   index keys and labels, which are layout machinery rather than prose (the
   Arabic files mix Arabic and English index keys).
3. Leftovers -- glossary terms surviving as English text inside the Arabic
   prose (outside the contexts stripped above).

The matching is deliberately conservative: Arabic is searched by normalized
stems (diacritics folded, alef/ya/ta-marbuta unified, definite article and a
common ة->ات plural stripped), so a term is only flagged when none of its
glossary renderings can be found at all. Human review is still expected --
the report lists evidence, not verdicts.

Usage:  python3 tools/audit_translation.py [report.md]
        (default report path: translation-audit.md next to this tool's root)
Exit code is 0 even with findings; the report is the deliverable.
"""
from __future__ import annotations

import re
import sys
import unicodedata
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
GLOSSARY = ROOT.parent / "book-glossary" / "GLOSSARY.md"
CONTENT = ROOT / "content"
# The references file is a bibliography; its titles stay English by design,
# so it is exempt from the terminology and leftover checks (not from coverage).
EXEMPT = {"others/97references97.typ"}

ENTRY_RE = re.compile(r"^(\S.*?)\s{2,}(\S.*)$")


# ── normalisation ────────────────────────────────────────────────────────────

def fold(s: str) -> str:
    """Fold Arabic text for matching: drop marks, unify letter variants."""
    out = []
    for ch in unicodedata.normalize("NFC", s):
        if unicodedata.category(ch) == "Mn" or ch == "\u0640":
            continue
        out.append(ch)
    s = "".join(out)
    s = s.replace("\u200f", "").replace("\u200e", "").replace("\xa0", " ")
    s = re.sub(r"[\u0622\u0623\u0625]", "\u0627", s)
    s = s.replace("\u0649", "\u064a")
    s = s.replace("\u0629", "\u0647")
    return re.sub(r"\s+", " ", s).strip()


def parse_glossary(path: Path) -> list[tuple[int, str, str]]:
    entries = []
    in_block = False
    for n, raw in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        if raw.startswith("```"):
            in_block = not in_block
            continue
        if not in_block:
            continue
        m = ENTRY_RE.match(raw.rstrip())
        if m:
            entries.append((n, m.group(1).strip(), m.group(2).strip()))
    return entries


def english_patterns(term: str) -> list[re.Pattern]:
    """Regexes for an English glossary term. Parentheses hold three kinds of
    note: grammatical tags ((to), (verb)) are dropped; plural markers ((s))
    are folded into the suffix; genuine alternates and acronym expansions
    become extra patterns. Remaining parentheses mark a context qualifier --
    the outside word is what the prose actually says, so it is the pattern."""
    tags = {"to", "verb", "noun", "adj", "adjective", "adv", "adverb", "s",
            "es", "plural", "singular", "a", "an", "the", "and", "or", "of",
            "in", "as", "e.g", "i.e"}
    t = re.sub(r"\[[A-Z]+\]", " ", term)  # book-scope tags
    t = re.sub(r"\((s|es)\)", " ", term)
    inner_alts: list[str] = []

    def grab(m: re.Match) -> str:
        inner = m.group(1).strip()
        if inner.lower() not in tags:
            inner_alts.append(inner)
        return " "

    base = re.sub(r"\s+", " ", re.sub(r"\(([^()]+)\)", grab, t)).strip()
    alts = {base} if base else set()
    for inner in inner_alts:
        # acronym expansion: "VM (Virtual Machine)" or "ACM (Association ...)"
        # single letters like "(F)"/"(R)" are too generic to search for
        if len(inner) < 2:
            continue
        if base and base.isupper() and len(base) <= 8 or not base:
            alts.add(inner)
        elif inner.isupper() and len(inner) <= 8:
            alts.add(inner)
    alts = {a for a in alts if a and re.search(r"[A-Za-z]", a)}
    out = []
    for alt in alts:
        words = [re.escape(w) for w in alt.split()]
        words[-1] += "(?:es|s)?"
        out.append(re.compile(r"\b" + r"\s+".join(words) + r"\b", re.I))
    return out


def arabic_stems(ar: str) -> set[str]:
    """Searchable stems for one Arabic rendering. Alternatives separated by
    '/' (sense splits) are searched independently; definite article and the
    common ة->ات plural are added as stem variants. For multi-word
    renderings the significant individual words are also stems, so a
    translation that keeps only the head noun still counts."""
    stems: set[str] = set()
    for alt in re.split(r"\s*/\s*", ar):
        alt = re.sub(r"\([^()]*\)", "", alt).strip()
        if not alt or not re.search(r"[\u0600-\u06FF]", alt):
            continue
        words = [w for w in re.split(r"\s+", fold(alt)) if w]
        for word in words + [" ".join(words)]:
            f = word
            if len(f) >= 3:
                stems.add(f)
            if f.startswith("ال") and len(f) > 4:
                stems.add(f[2:])
            if f.endswith("يه"):
                stems.add(f[:-2] + "يات")
            if f.endswith("اه") and len(f) > 4:
                stems.add(f[:-1] + "ت")
            if f.endswith("ات"):
                stems.add(f[:-1])
    # irregular plurals the ة->ات rule cannot produce (and would misreport
    # as missing glossary compliance): وسيط->وسائط, مهمة->مهام, قوة->قوى
    for word in list(stems):
        for singular, plural in (("وسيط", "وسائط"), ("مهمه", "مهام"), ("قوه", "قوى")):
            if word == singular:
                stems.add(plural)
            elif word.endswith(singular):
                stems.add(word[: -len(singular)] + plural)
    return {s for s in stems if len(s) >= 3}


# ── prose extraction ─────────────────────────────────────────────────────────

FENCE_RE = re.compile(r"```.*?```", re.S)
STRING_RE = re.compile(r'"(?:[^"\\]|\\.)*"')
EN_SPAN_RE = re.compile(r"#en\[[^\[\]]*\]")
LABEL_RE = re.compile(r"<[A-Za-z][A-Za-z0-9:_-]*>")
REF_RE = re.compile(r"@[A-Za-z][A-Za-z0-9:_-]*")
URL_RE = re.compile(r"https?://\S+")
COMMENT_RE = re.compile(r"//[^\n]*")
CALL_HEAD_RE = re.compile(r"#[A-Za-z][A-Za-z0-9_-]*\(")


def strip_calls(text: str) -> str:
    """Remove every #call( ... ) with balanced-paren scanning, whatever its
    arity or nesting (idx, py, syntax templates, sicp-figure(...) with nested
    layout calls, ...). Bracketed bodies that follow the call survive."""
    while True:
        m = CALL_HEAD_RE.search(text)
        if not m:
            return text
        depth, i = 1, m.end()
        while i < len(text) and depth:
            if text[i] == "(":
                depth += 1
            elif text[i] == ")":
                depth -= 1
            i += 1
        text = text[:m.start()] + " " + text[i:]


def prose(text: str) -> str:
    text = FENCE_RE.sub(" ", text)
    text = COMMENT_RE.sub(" ", text)
    text = STRING_RE.sub(" ", text)
    for _ in range(4):
        text = EN_SPAN_RE.sub(" ", text)
    text = strip_calls(text)
    text = LABEL_RE.sub(" ", text)
    text = REF_RE.sub(" ", text)
    text = URL_RE.sub(" ", text)
    return text


def pairs() -> list[tuple[Path, Path]]:
    out = []
    for en in sorted(CONTENT.rglob("*.typ")):
        if en.name.endswith("-ar.typ"):
            continue
        ar = en.with_name(en.stem + "-ar.typ")
        if ar.exists():
            out.append((en, ar))
    return out


def counts(text: str) -> dict[str, int]:
    return {
        "headings": len(re.findall(r"#(?:chapter|section|subsection|subsubsection|matter|matter-section|subheading|subsubheading)\(", text)),
        "exercises": len(re.findall(r"#exercise\(", text)),
        "footnotes": len(re.findall(r"#footnote\[", text)),
        "snippets": len(re.findall(r"#snippet\(", text)),
        "index": len(re.findall(r"#idx\(", text)),
    }


def labels_of(text: str) -> set[str]:
    return (
        set(re.findall(r"label-name: <([^>]+)>", text))
        | set(re.findall(r"\]<([^>]+)>", text))
        | set(re.findall(r"#anchor\(<([^>]+)>\)", text))
    )


# ── the audit ────────────────────────────────────────────────────────────────

def main() -> int:
    report_path = Path(sys.argv[1]) if len(sys.argv) > 1 else ROOT / "translation-audit.md"
    entries = parse_glossary(GLOSSARY)

    rep: list[str] = []
    rep.append("# مقارنة الترجمة العربية بالأصل الإنجليزي وجدول المصطلحات\n")
    rep.append("التدقيق آلي: يقارن كل ملف إنجليزي بنظيره العربي (التغطية والبنية)، ويتتبع ")
    rep.append("كل مصطلح من «جدول المصطلحات الموحَّد» حاضر في نثر الأصل — ويفتش عن مقابل ")
    rep.append("الجدول في النثر العربي (مع تجاهل الكود ومفاتيح الفهرس التي تبقى إنجليزية عمدًا). ")
    rep.append("المطابقة مرنة صرفيًا (إسقاط التشكيل، توحيد الألف/الياء/التاء المربوطة، سقوط «ال» ")
    rep.append("وتصريف «ة→ات»)، فالرايات تستدعي مراجعة بشرية لا حكمًا نهائيًا.\n")

    # ── 1. coverage ───────────────────────────────────────────────────────
    all_ar = {p for p in CONTENT.rglob("*-ar.typ")}
    paired = pairs()
    ar_set = {ar for _, ar in paired}
    orphans = sorted(all_ar - ar_set)
    missing = sorted(
        en.with_name(en.stem + "-ar.typ")
        for en, _ in paired
        if False
    )
    missing = sorted(
        en for en in sorted(CONTENT.rglob("*.typ"))
        if not en.name.endswith("-ar.typ")
        and not en.with_name(en.stem + "-ar.typ").exists()
    )

    rep.append("## ١. التغطية\n")
    rep.append(f"- أزواج ملفات (إنجليزي ↔ عربي): **{len(paired)}**")
    rep.append(f"- ملفات إنجليزية بلا مقابل عربي: **{len(missing)}** — "
               + (", ".join("`" + str(m.relative_to(CONTENT)) + "`" for m in missing) or "—"))
    rep.append(f"- ملفات عربية بلا أصل: **{len(orphans)}** — "
               + (", ".join("`" + str(o.relative_to(CONTENT)) + "`" for o in orphans) or "—"))
    rep.append("")

    label_gaps, count_gaps = [], []
    rows = []
    for en, ar in paired:
        ten, tar = en.read_text(encoding="utf-8"), ar.read_text(encoding="utf-8")
        len_l, lar = labels_of(ten), labels_of(tar)
        if len_l != lar:
            label_gaps.append((en, sorted(len_l - lar), sorted(lar - len_l)))
        cen, car = counts(ten), counts(tar)
        if cen != car:
            count_gaps.append((en, cen, car))
        rows.append((en, cen, car))
    rep.append(f"- تعارض في المرساة (labels): **{len(label_gaps)}** ملفًا")
    for en, only_en, only_ar in label_gaps:
        rep.append(f"  - `{en.relative_to(CONTENT)}`: في الإنجليزي فقط "
                   f"{only_en or '—'}؛ في العربي فقط {only_ar or '—'}")
    rep.append(f"- تعارض في العدّادات (عناوين/تمارين/حواشي/مقتطفات/فهرس): **{len(count_gaps)}** ملفًا")
    for en, cen, car in count_gaps:
        diff = {k: (cen[k], car[k]) for k in cen if cen[k] != car[k]}
        rep.append(f"  - `{en.relative_to(CONTENT)}`: " +
                   "، ".join(f"{k}: أصلي {v[0]} / مترجم {v[1]}" for k, v in sorted(diff.items())))
    rep.append("")

    # totals
    tot = defaultdict(lambda: [0, 0])
    for en, cen, car in rows:
        for k in cen:
            tot[k][0] += cen[k]
            tot[k][1] += car[k]
    rep.append("| العدّاد | الأصل | الترجمة |")
    rep.append("|---|---|---|")
    names = {"headings": "العناوين", "exercises": "التمارين", "footnotes": "الحواشي",
             "snippets": "مقتطفات الكود", "index": "مداخل الفهرس"}
    for k in ("headings", "exercises", "footnotes", "snippets", "index"):
        rep.append(f"| {names[k]} | {tot[k][0]} | {tot[k][1]} |")
    rep.append("")

    # ── 2. terminology ────────────────────────────────────────────────────
    # per-pair prose caches (Arabic prose is folded so folded stems match)
    prose_en, prose_ar = {}, {}
    for en, ar in paired:
        if en.relative_to(CONTENT).as_posix() in EXEMPT:
            continue
        prose_en[en] = prose(en.read_text(encoding="utf-8"))
        prose_ar[ar] = fold(prose(ar.read_text(encoding="utf-8")))

    # This repo's content is SICP; glossary entries scoped to another book
    # ([OSTEP]...) are out of context here (editorial rule 6).
    entries = [e for e in entries if "[OSTEP]" not in e[1]]

    latin_only, rows_t = [], []
    for line, term, ar in entries:
        if not re.search(r"[\u0600-\u06FF]", ar):
            latin_only.append((line, term, ar))
            continue
        pats = english_patterns(term)
        stems = arabic_stems(ar)
        if not pats or not stems:
            continue
        en_total = ar_total = 0
        files_with, files_hit = [], []
        for en, arf in paired:
            if en not in prose_en:
                continue
            n = sum(len(p.findall(prose_en[en])) for p in pats)
            if not n:
                continue
            files_with.append(en)
            en_total += n
            m = sum(prose_ar[arf].count(s) for s in stems)
            ar_total += m
            if m:
                files_hit.append(arf)
        if en_total:
            rows_t.append((term, ar, line, en_total, ar_total, len(files_hit), len(files_hit and files_hit) and len(files_hit)))

    rows_t.sort(key=lambda r: (-(r[3]), r[0]))
    ok = [r for r in rows_t if r[4] * 2 >= r[3]]
    weak = [r for r in rows_t if 0 < r[4] * 2 < r[3]]
    miss = [r for r in rows_t if r[4] == 0]

    rep.append("## ٢. التزام المصطلح مع الجدول\n")
    rep.append(f"مصطلحات الجدول الحاضرة في نثر الأصل: **{len(rows_t)}**؛ "
               f"منها متوافق (ظهر المقابل في ≥٥٠٪ من المواضع): **{len(ok)}**؛ "
               f"ضعيف التغطية: **{len(weak)}**؛ بلا مقابل ظاهر: **{len(miss)}**.\n")
    rep.append("### بلا مقابل ظاهر في الترجمة (مرشّحة أولى للمراجعة)\n")
    rep.append("| السطر في الجدول | المصطلح | مقابل الجدول | مواضع في الأصل |")
    rep.append("|---|---|---|---|")
    for term, ar, line, en_total, ar_total, fw, fh in miss:
        rep.append(f"| {line} | {term} | {ar} | {en_total} |")
    rep.append("")
    rep.append("### ضعيفة التغطية (المقابل ظهر في أقل من نصف المواضع)\n")
    rep.append("| السطر | المصطلح | مقابل الجدول | الأصل | ظهر المقابل |")
    rep.append("|---|---|---|---|---|")
    for term, ar, line, en_total, ar_total, fw, fh in weak:
        rep.append(f"| {line} | {term} | {ar} | {en_total} | {ar_total} |")
    rep.append("")
    rep.append("### مصطلحات اللاتينية فقط في الجدول (تبقى كما هي، لا تُدقق)\n")
    rep.append("، ".join(f"`{t}`" for _, t, _ in latin_only) or "—")
    rep.append("")

    # ── 3. leftovers ──────────────────────────────────────────────────────
    # leftover scan needs the *unfolded* prose (English inside it must stay
    # intact) plus the raw layout to tell parenthesised glosses from bare use
    raw_prose_ar = {
        ar: prose(ar.read_text(encoding="utf-8"))
        for _, ar in paired
        if _.relative_to(CONTENT).as_posix() not in EXEMPT
    }
    bare_hits, gloss_hits = [], []
    for line, term, ar in entries:
        if not re.search(r"[A-Za-z]", term):
            continue
        pats = english_patterns(term)
        for en, arf in paired:
            if en not in prose_en:
                continue
            text = raw_prose_ar[arf]
            for p in pats:
                for m in p.finditer(text):
                    before = text[:m.start()].rstrip()
                    snippet = text[max(0, m.start() - 40):m.end() + 40].replace("\n", " ")
                    in_parens = before.endswith("(")
                    (gloss_hits if in_parens else bare_hits).append((term, arf, snippet))
                    break  # one excerpt per (term, file, kind)

    def emit_leftovers(title: str, hits: list) -> None:
        by_term = defaultdict(list)
        for term, arf, snippet in hits:
            by_term[term].append((arf, snippet))
        rep.append(f"{title}\n")
        if not by_term:
            rep.append("لا شيء.\n")
            return
        rep.append("| المصطلح | الملف | السياق |")
        rep.append("|---|---|---|")
        for term in sorted(by_term, key=str.lower):
            for arf, snippet in by_term[term][:3]:
                rep.append(f"| {term} | `{arf.relative_to(CONTENT)}` | …{snippet.strip()}… |")
        rep.append("")

    rep.append("## ٣. بقايا إنجليزية للمصطلحات (خارج الكود والفهرس والمراجع)\n")
    emit_leftovers("### إنجليزية مكشوفة في النثر (مرشّحة للترجمة أو القرار)", bare_hits)
    emit_leftovers("### بين قوسين بوصفها شرحًا ثنائي اللغة (للاطلاع لا للتبديل)", gloss_hits)

    # glossary self-check pointer
    rep.append("## ٤. ملاحظة على الجدول نفسه\n")
    rep.append("`book-glossary/tools/check_glossary.py` يُبلغ عن تعارضَين من قاعدة «مقابل عربي ")
    rep.append("واحد لمصطلح إنجليزي واحد» (آلة افتراضية مشتركة بين Virtual Machine وVM، وتتبّع ")
    rep.append("مشتركة بين صيغتي Trace) — أُدرجا في الجدول عمدًا بوصفهما اختصارًا وصيغتين، والقرار ")
    rep.append("للمحرِّر. التدقيق أعلاه يستعمل كلا الصيغتين عند البحث.\n")

    report_path.write_text("\n".join(rep), encoding="utf-8")
    print(f"report: {report_path}")
    print(f"pairs={len(paired)} missing={len(missing)} orphans={len(orphans)} "
          f"label_gaps={len(label_gaps)} count_gaps={len(count_gaps)}")
    print(f"terms present in source prose: {len(rows_t)} ok={len(ok)} weak={len(weak)} missing={len(miss)}")
    print(f"leftovers: {len(bare_hits)} bare / {len(gloss_hits)} paren-gloss")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
