#!/usr/bin/env python3
"""Генерация иконки CodePic.

Создаёт в assets/icons/:
  icon.png            — 1024x1024, прозрачный фон (источник для fyne-cross)
  icon.ico            — многоразмерный 16..256 (для Windows)
  hicolor/<N>x<N>.png — набор PNG для freedesktop (Linux)

Требуется Pillow:  pip install Pillow
"""

from PIL import Image, ImageDraw
import os

# Основной размер (рекомендация Fyne: 1024x1024)
BASE = 1024

# Куда писать результат
OUT = "assets/icons"

# --- Геометрия (координаты 1024x1024) ---
CARD_X, CARD_Y = 130, 130
CARD_W, CARD_H = 764, 764
CARD_RADIUS    = 110

DOT_RADIUS  = 36
DOT_Y       = 342
DOT_X_START = 280
DOT_SPACING = 90

LINE_HEIGHT  = 60
LINE_RADIUS  = 30
LINE_X       = 280
LINE_Y_START = 470
LINE_GAP     = 100
LINE_WIDTHS  = [380, 480, 260]  # три строки разной длины

# --- Цвета (палитра Dracula) ---
CARD_BG      = (40, 42, 54, 255)      # #282a36
DOT_RED      = (255, 95, 86, 255)     # #ff5f56
DOT_YELLOW   = (255, 189, 46, 255)    # #ffbd2e
DOT_GREEN    = (39, 201, 63, 255)     # #27c93f
CODE_PURPLE  = (189, 147, 249, 255)   # #bd93f9
CODE_GREEN   = (80, 250, 123, 255)    # #50fa7b
CODE_YELLOW  = (241, 250, 140, 255)   # #f1fa8c


def draw_icon(size: int) -> Image.Image:
    """Рисует иконку заданного размера (квадратную, RGBA)."""
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    s = size / BASE
    def r(v): return int(round(v * s))

    # 1. Карточка
    draw.rounded_rectangle(
        (r(CARD_X), r(CARD_Y), r(CARD_X + CARD_W), r(CARD_Y + CARD_H)),
        radius=r(CARD_RADIUS),
        fill=CARD_BG,
    )

    # 2. Три точки в шапке
    for i, color in enumerate([DOT_RED, DOT_YELLOW, DOT_GREEN]):
        cx = r(DOT_X_START + i * DOT_SPACING)
        cy = r(DOT_Y)
        rad = r(DOT_RADIUS)
        draw.ellipse((cx - rad, cy - rad, cx + rad, cy + rad), fill=color)

    # 3. Строки «кода» с подсветкой
    for i, (width, color) in enumerate(zip(
        LINE_WIDTHS, [CODE_PURPLE, CODE_GREEN, CODE_YELLOW]
    )):
        y = r(LINE_Y_START + i * LINE_GAP)
        draw.rounded_rectangle(
            (r(LINE_X), y, r(LINE_X + width), y + r(LINE_HEIGHT)),
            radius=r(LINE_RADIUS),
            fill=color,
        )

    return img


def main():
    # Создаём дерево assets/icons/hicolor
    hicolor = os.path.join(OUT, "hicolor")
    os.makedirs(hicolor, exist_ok=True)

    # 1. Основной PNG — источник для fyne-cross
    draw_icon(BASE).save(os.path.join(OUT, "icon.png"))
    print(f"✓ {OUT}/icon.png ({BASE}x{BASE})")

    # 2. Многоразмерный ICO для Windows
    ico_sizes = [16, 24, 32, 48, 64, 128, 256]
    images = [draw_icon(s) for s in ico_sizes]
    images[0].save(
        os.path.join(OUT, "icon.ico"),
        format="ICO",
        sizes=[(s, s) for s in ico_sizes],
        append_images=images[1:],
    )
    print(f"✓ {OUT}/icon.ico ({', '.join(map(str, ico_sizes))})")

    # 3. Набор PNG для freedesktop
    for s in [16, 22, 24, 32, 48, 64, 128, 256]:
        draw_icon(s).save(os.path.join(hicolor, f"{s}x{s}.png"))
    print(f"✓ {OUT}/hicolor/ (freedesktop sizes)")


if __name__ == "__main__":
    main()