/**
 * contrast.ts — WCAG 2.1 contrast ratio utilities
 *
 * Вычисляет относительную яркость (relative luminance) и коэффициент
 * контраста между двумя цветами. Проверяет AA/AAA для текста.
 *
 * @see https://www.w3.org/WAI/WCAG21/Understanding/contrast-minimum
 */

// ── Константы порогов ─────────────────────────────────────────────

/** Минимальный контраст для обычного текста (AA) */
export const AA_TEXT = 4.5;

/** Минимальный контраст для крупного текста ≥18px или ≥14px bold (AA) */
export const AA_LARGE = 3.0;

/** Минимальный контраст для обычного текста (AAA) */
export const AAA_TEXT = 7.0;

/** Минимальный контраст для крупного текста (AAA) */
export const AAA_LARGE = 4.5;

// ── Утилиты ───────────────────────────────────────────────────────

/**
 * Преобразует hex-цвет (#rgb, #rrggbb) в массив RGB [0-1].
 */
function hexToRgb(hex: string): [number, number, number] {
    const clean = hex.replace('#', '');
    let r: number, g: number, b: number;

    if (clean.length === 3) {
        r = parseInt(clean[0] + clean[0], 16) / 255;
        g = parseInt(clean[1] + clean[1], 16) / 255;
        b = parseInt(clean[2] + clean[2], 16) / 255;
    } else if (clean.length === 6) {
        r = parseInt(clean.slice(0, 2), 16) / 255;
        g = parseInt(clean.slice(2, 4), 16) / 255;
        b = parseInt(clean.slice(4, 6), 16) / 255;
    } else {
        throw new Error(`Invalid hex color: ${hex}`);
    }

    return [r, g, b];
}

/**
 * Линеаризация sRGB-канала по стандарту WCAG 2.1.
 * https://www.w3.org/TR/WCAG21/#dfn-relative-luminance
 */
function linearize(channel: number): number {
    if (channel <= 0.04045) {
        return channel / 12.92;
    }
    return Math.pow((channel + 0.055) / 1.055, 2.4);
}

/**
 * Вычисляет относительную яркость (relative luminance) hex-цвета.
 * Значение от 0 (чёрный) до 1 (белый).
 *
 * @example relativeLuminance('#ffffff') // ≈ 1.0
 * @example relativeLuminance('#000000') // ≈ 0.0
 */
export function relativeLuminance(hex: string): number {
    const [r, g, b] = hexToRgb(hex);
    return 0.2126 * linearize(r) + 0.7152 * linearize(g) + 0.0722 * linearize(b);
}

/**
 * Вычисляет коэффициент контраста между двумя цветами по WCAG 2.1.
 * Значение от 1:1 (одинаковые цвета) до 21:1 (чёрный на белом).
 *
 * @example contrastRatio('#000000', '#ffffff') // ≈ 21.0
 */
export function contrastRatio(fg: string, bg: string): number {
    const l1 = relativeLuminance(fg);
    const l2 = relativeLuminance(bg);
    const lighter = Math.max(l1, l2);
    const darker = Math.min(l1, l2);
    return (lighter + 0.05) / (darker + 0.05);
}

/**
 * Проверяет WCAG 2.1 AA для обычного текста (≥4.5:1).
 * Для крупного текста используйте `isAALarge`.
 */
export function isAA(fg: string, bg: string): boolean {
    return contrastRatio(fg, bg) >= AA_TEXT;
}

/**
 * Проверяет WCAG 2.1 AA для крупного текста (≥3:1).
 * Крупный текст: ≥18px или ≥14px bold.
 */
export function isAALarge(fg: string, bg: string): boolean {
    return contrastRatio(fg, bg) >= AA_LARGE;
}

/**
 * Проверяет WCAG 2.1 AAA для обычного текста (≥7:1).
 */
export function isAAA(fg: string, bg: string): boolean {
    return contrastRatio(fg, bg) >= AAA_TEXT;
}

/**
 * Проверяет WCAG 2.1 AAA для крупного текста (≥4.5:1).
 */
export function isAAALarge(fg: string, bg: string): boolean {
    return contrastRatio(fg, bg) >= AAA_LARGE;
}

/**
 * Находит наилучший цвет текста из предложенных, проходящий AA.
 * Возвращает первый подходящий, либо `fallback` если ни один не прошёл.
 *
 * @example const best = bestTextColor(['#94a3b8', '#64748b', '#475569'], '#ffffff');
 * // best === '#475569' (text-slate-600 на белом: 7.3:1)
 */
export function bestTextColor(
    candidates: string[],
    bg: string,
    fallback: string = '#000000',
    threshold: number = AA_TEXT,
): string {
    for (const color of candidates) {
        if (contrastRatio(color, bg) >= threshold) {
            return color;
        }
    }
    return fallback;
}
