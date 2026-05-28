// Helpers for fitting text labels inside treemap cells. The treemap in
// `allocation.ts` renders HTML divs with `overflow: hidden`, which means
// long account names — especially under zh-CN locales where CJK characters
// take ~2x the width of an ASCII char — get clipped mid-character to
// garbled prefixes like "ASSETS:H.." or "A:RECEIVE..". Rather than relying
// on the browser to truncate, we estimate the rendered text width upfront
// and either return a truncated label with an ellipsis or hide the label
// entirely when even a short prefix would not fit. The full account name
// is always visible via the existing `data-tippy-content` tooltip.
//
// The width estimate is intentionally cheap and approximate — Canvas /
// `getComputedTextLength()` would be accurate but require DOM access at
// render-time, which we want to avoid so this helper stays a pure
// function (and unit-testable under `bun:test` without happydom).

// Average character width as a fraction of the font size. ASCII / Latin
// glyphs in the default sans-serif stack hover around 0.55–0.6 em for
// regular weight; we use 0.6 to bias the estimate slightly wider so we err
// on the side of hiding labels that would otherwise clip.
const ASCII_CHAR_WIDTH_EM = 0.6;

// CJK ideographs occupy roughly one em in every font we care about. The
// Bulma `.heading` class used for treemap labels falls back to system
// fonts, so this constant works across PingFang / Noto Sans CJK / etc.
const CJK_CHAR_WIDTH_EM = 1.0;

// Below this many pixels we don't bother rendering anything — even a
// single character plus an ellipsis would be illegible. This is the
// horizontal padding the `.node` flex container effectively reserves for
// the label, not the cell width itself; we let the caller subtract any
// horizontal padding before passing the width in.
const MIN_RENDERABLE_PX = 16;

// Estimate the rendered width (in px) of `label` when typeset at the
// given font size. Treats characters outside the basic Latin / Latin-1
// supplement ranges as full-width to approximate CJK behavior — that's
// the dominant non-ASCII case in this app's locales (en, zh-CN).
export function estimateTextWidth(label: string, fontSizePx: number): number {
  let width = 0;
  for (const ch of label) {
    const code = ch.codePointAt(0) ?? 0;
    // Anything in the 0x00–0xFF range is treated as half-width, everything
    // else (CJK, fullwidth forms, emoji surrogates, …) as full-width.
    width += code <= 0xff ? ASCII_CHAR_WIDTH_EM : CJK_CHAR_WIDTH_EM;
  }
  return width * fontSizePx;
}

// Return a label that fits inside `cellWidthPx` at `fontSizePx`. If the
// label already fits, it is returned unchanged. If a truncated prefix
// plus an ellipsis would fit, that is returned. If the cell is so narrow
// that even a one-character prefix would clip (or smaller than
// `MIN_RENDERABLE_PX`), the empty string is returned and the caller
// should hide the label entirely.
export function fitTreemapLabel(label: string, cellWidthPx: number, fontSizePx: number): string {
  if (!label) return "";
  if (cellWidthPx < MIN_RENDERABLE_PX) return "";

  const fullWidth = estimateTextWidth(label, fontSizePx);
  if (fullWidth <= cellWidthPx) return label;

  const ellipsis = "…"; // single-char Unicode ellipsis
  const ellipsisWidth = estimateTextWidth(ellipsis, fontSizePx);
  const budget = cellWidthPx - ellipsisWidth;
  if (budget <= 0) return "";

  // Walk the codepoints from the start, accumulating width until adding
  // the next one would overflow. We work in codepoints (not UTF-16 units)
  // so a surrogate pair never splits mid-character.
  const chars = Array.from(label);
  let used = 0;
  let prefix = "";
  for (const ch of chars) {
    const w = estimateTextWidth(ch, fontSizePx);
    if (used + w > budget) break;
    used += w;
    prefix += ch;
  }
  if (prefix.length === 0) return "";
  return prefix + ellipsis;
}
