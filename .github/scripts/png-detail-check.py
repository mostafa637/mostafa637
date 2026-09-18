#!/usr/bin/env python3
"""Decide whether a screenshot actually shows something, without PIL or ImageMagick.

Why this exists: the emulator job's first rendered frame can be pure black. Android's
GPU emulation (swiftshader_indirect) needs a second or more per frame on a 2-core
runner - the run log had "EGL_emulation: app_time_stats: avg=3730.97ms" - so a
screencap taken right after the app logged its markers can legitimately be empty.
"19 checks passed" while the screenshot was black was exactly the kind of green that
proves nothing, so the pass/fail decision now includes one pixel-level question: does
this frame contain more than one colour?

The threshold is not "more than one colour": the black first frame measures two
colours (the window plus the system gesture pill), so that passes. What separates "the
app painted" from "nothing was painted" is whether any light reached the framebuffer -
the terminal is white-on-black, so a real frame has both many colours and a lit share -
hence the pair: MIN_COLORS and MIN_LIT.

Only what screencap produces is supported: 8-bit RGB/RGBA, non-interlaced, no
prediction other than the standard PNG filters. Anything else reports "undecodable"
rather than guessing - an honest unknown beats a wrong pass.
"""
import struct
import sys
import zlib

MAX_COLORS = 4096  # bounds the set while scanning; anything past "few" is enough
MIN_COLORS = 16    # a black frame with system chrome measures ~2-6
MIN_LIT = 0.5      # percent of sampled pixels bright enough to be UI, not noise
LIT_FLOOR = 40       # 8-bit channel value; #222222 (the key bar) already counts


def chunks(data):
    if data[:8] != b"\x89PNG\r\n\x1a\n":
        raise ValueError("not a PNG")
    pos = 8
    while pos + 8 <= len(data):
        length = struct.unpack_from(">I", data, pos)[0]
        kind = data[pos + 4:pos + 8]
        payload = data[pos + 8:pos + 8 + length]
        yield kind, payload
        pos += 12 + length
        if kind == b"IEND":
            return


def paeth(a, b, c):
    p = a + b - c
    pa, pb, pc = abs(p - a), abs(p - b), abs(p - c)
    if pa <= pb and pa <= pc:
        return a
    return b if pb <= pc else c


def main(path):
    with open(path, "rb") as fh:
        data = fh.read()
    width = height = depth = color_type = interlace = None
    idat = bytearray()
    for kind, payload in chunks(data):
        if kind == b"IHDR":
            width, height, depth, color_type, _comp, _mode, interlace = struct.unpack(">IIBBBBB", payload)
        elif kind == b"IDAT":
            idat += payload
        elif kind == b"IEND":
            break
    if interlace or depth != 8 or color_type not in (2, 6):
        print("undecodable: not 8-bit non-interlaced RGB(A)")
        return 2
    channels = 3 if color_type == 2 else 4
    stride = width * channels
    raw = zlib.decompress(bytes(idat))
    if len(raw) < (stride + 1) * height:
        print("undecodable: truncated pixel data")
        return 2

    colors = set()
    lit = 0
    sampled = 0
    prev = bytearray(stride)
    pos = 0
    rows_sampled = 0
    step = max(1, height // 64)  # 64 rows is plenty to tell blank from rendered
    for y in range(height):
        ft = raw[pos]
        pos += 1
        line = bytearray(raw[pos:pos + stride])
        pos += stride
        if ft == 1:
            for i in range(channels, stride):
                line[i] = (line[i] + line[i - channels]) & 0xFF
        elif ft == 2:
            for i in range(stride):
                line[i] = (line[i] + prev[i]) & 0xFF
        elif ft == 3:
            for i in range(stride):
                left = line[i - channels] if i >= channels else 0
                line[i] = (line[i] + ((left + prev[i]) >> 1)) & 0xFF
        elif ft == 4:
            for i in range(stride):
                left = line[i - channels] if i >= channels else 0
                upleft = prev[i - channels] if i >= channels else 0
                line[i] = (line[i] + paeth(left, prev[i], upleft)) & 0xFF
        elif ft != 0:
            print("undecodable: unknown filter %d" % ft)
            return 2
        if y % step == 0:
            rows_sampled += 1
            for i in range(0, stride, channels):
                r, g, b = line[i], line[i + 1], line[i + 2]
                colors.add(bytes((r, g, b)))
                if max(r, g, b) >= LIT_FLOOR:
                    lit += 1
                sampled += 1
        prev = line
    share = 100.0 * lit / max(sampled, 1)
    verdict = "rendered" if len(colors) >= MIN_COLORS and share >= MIN_LIT else "BLANK"
    print("colors=%d lit=%.2f%% (floor %d) rows_sampled=%d pixels=%d detail=%s"
          % (len(colors), share, LIT_FLOOR, rows_sampled, sampled, verdict))
    return 0 if verdict == "rendered" else 1


if __name__ == "__main__":
    try:
        sys.exit(main(sys.argv[1]))
    except Exception as exc:  # never let a decoder surprise look like an app failure
        print("undecodable: %s" % exc)
        sys.exit(2)
