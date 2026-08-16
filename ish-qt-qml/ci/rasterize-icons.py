from pathlib import Path
import cairosvg

root = Path(__file__).resolve().parents[1]
icon_dir = root / "android-qt" / "assets" / "ui" / "icons"
for source in sorted(icon_dir.glob("*.svg")):
    target = source.with_suffix(".png")
    cairosvg.svg2png(url=str(source), write_to=str(target), output_width=128, output_height=128)
    print(target.relative_to(root))
