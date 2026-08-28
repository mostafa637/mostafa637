from pathlib import Path

import cairosvg
from PIL import Image, ImageDraw

root = Path(__file__).resolve().parents[1]
icon_dir = root / "android-qt" / "assets" / "ui" / "icons"

for source in sorted(icon_dir.glob("*.svg")):
    target = source.with_suffix(".png")
    cairosvg.svg2png(url=str(source), write_to=str(target), output_width=128, output_height=128)
    print(target.relative_to(root))

# These four iSH iOS accessory glyphs are not available consistently in
# Android fonts. Draw the original symbol shapes into PNGs so runtime rendering
# does not depend on font coverage or a missing-glyph box.
S = 4
SIZE = 128


def canvas(color):
    return Image.new("RGBA", (SIZE * S, SIZE * S), (0, 0, 0, 0)), color


def line(draw, points, color, width=7):
    draw.line([(int(x * S), int(y * S)) for x, y in points], fill=color,
              width=width * S, joint="curve")


def ellipse(draw, box, color, width=7):
    draw.ellipse(tuple(int(v * S) for v in box), outline=color, width=width * S)


def rect(draw, box, color, width=7, radius=8):
    draw.rounded_rectangle(tuple(int(v * S) for v in box), radius=radius * S,
                           outline=color, width=width * S)


def make_tab(color):
    image, color = canvas(color)
    draw = ImageDraw.Draw(image)
    line(draw, [(16, 64), (108, 64)], color)
    line(draw, [(78, 34), (108, 64), (78, 94)], color)
    line(draw, [(110, 18), (110, 110)], color)
    return image


def make_control(color):
    image, color = canvas(color)
    draw = ImageDraw.Draw(image)
    line(draw, [(18, 92), (64, 26), (110, 92)], color)
    return image


def make_escape(color):
    image, color = canvas(color)
    draw = ImageDraw.Draw(image)
    rect(draw, (58, 24, 112, 104), color)
    line(draw, [(72, 64), (18, 64)], color)
    line(draw, [(18, 64), (40, 42)], color)
    line(draw, [(18, 64), (40, 86)], color)
    return image


def make_info(color):
    image, color = canvas(color)
    draw = ImageDraw.Draw(image)
    ellipse(draw, (18, 18, 110, 110), color)
    draw.ellipse((60 * S, 34 * S, 68 * S, 42 * S), fill=color)
    line(draw, [(64, 54), (64, 92)], color, width=8)
    return image


makers = {"tab": make_tab, "control": make_control, "escape": make_escape,
          "info": make_info}
for mode, color in (("light", (28, 28, 30, 255)), ("dark", (245, 245, 247, 255))):
    for name, maker in makers.items():
        target = icon_dir / f"{name}-{mode}.png"
        image = maker(color).resize((SIZE, SIZE), Image.Resampling.LANCZOS)
        image.save(target, optimize=True)
        print(target.relative_to(root))
