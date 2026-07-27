#!/usr/bin/env node
/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/**
 * One-shot migration tool: qui premium theme (shadcn semantic tokens)
 * -> netronome theme (Tailwind v4 colour-ramp override).
 *
 *   node scripts/convert-theme.mjs <input.css> <output.css> [--id nord] [--extras]
 *
 * Not a build step. Run it once per theme, eyeball the result, tune by hand.
 *
 * Everything stays in oklch() end to end, so there is no colour-space maths
 * beyond lerping three numbers. Inputs that are not oklch() are rejected with
 * a message rather than silently guessed at.
 */

import { readFileSync, writeFileSync } from "node:fs";
import { basename } from "node:path";

// ---------------------------------------------------------------- oklch ----

const parseOklch = (raw, what) => {
  const m = /oklch\(\s*([\d.]+)(%?)\s+([\d.]+)\s+([\d.]+)/i.exec(raw ?? "");
  if (!m) {
    throw new Error(
      `${what}: expected an oklch() value, got "${(raw ?? "").trim()}". ` +
        `Convert it to oklch() in the source theme first.`
    );
  }
  return {
    l: m[2] ? Number(m[1]) / 100 : Number(m[1]),
    c: Number(m[3]),
    h: Number(m[4]),
  };
};

// ---- sRGB -> OKLCH (Björn Ottosson's matrices) -----------------------------
// Themes are mostly authored in oklch(), but not exclusively, so plain rgb()
// and hex anchors have to be readable too.

const toLinear = (v) =>
  v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;

const srgbToOklch = (r, g, b) => {
  const [lr, lg, lb] = [r, g, b].map((v) => toLinear(v / 255));
  const l = Math.cbrt(0.4122214708 * lr + 0.5363325363 * lg + 0.0514459929 * lb);
  const m = Math.cbrt(0.2119034982 * lr + 0.6806995451 * lg + 0.1073969566 * lb);
  const s = Math.cbrt(0.0883024619 * lr + 0.2817188376 * lg + 0.6299787005 * lb);
  const L = 0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s;
  const A = 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s;
  const B = 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s;
  return oklabToOklch(L, A, B);
};

const oklabToOklch = (L, A, B) => ({
  l: L,
  c: Math.hypot(A, B),
  h: ((Math.atan2(B, A) * 180) / Math.PI + 360) % 360,
});

const oklchToOklab = ({ l, c, h }) => {
  const rad = (h * Math.PI) / 180;
  return [l, c * Math.cos(rad), c * Math.sin(rad)];
};

const toGamma = (v) =>
  v <= 0.0031308 ? v * 12.92 : 1.055 * v ** (1 / 2.4) - 0.055;

const oklchToSrgb = (col) => {
  const [L, A, B] = oklchToOklab(col);
  const l = (L + 0.3963377774 * A + 0.2158037573 * B) ** 3;
  const m = (L - 0.1055613458 * A - 0.0638541728 * B) ** 3;
  const s = (L - 0.0894841775 * A - 1.291485548 * B) ** 3;
  return [
    4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
    -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
    -0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s,
  ].map((v) => Math.min(1, Math.max(0, toGamma(v))));
};

/** WCAG 2.1 relative luminance + contrast ratio, for the AA check below. */
const relLuminance = (col) => {
  const [r, g, b] = oklchToSrgb(col).map((v) =>
    v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4
  );
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
};

const contrast = (a, b) => {
  const [hi, lo] = [relLuminance(a), relLuminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
};

const AA = 4.5;
const BLACK = { l: 0, c: 0, h: 0 };
const WHITE = { l: 1, c: 0, h: 0 };

// ---- general value resolution ---------------------------------------------
// Handles var() indirection, color-mix(in oklab, ...) and rgb()/hex, so
// parametric themes (qui's "variation" themes build every colour out of
// color-mix over a var) convert instead of erroring out.
//
// Alpha is deliberately dropped: these values become opaque ramp entries used
// for solid surfaces, and there is no single backdrop to composite against.

/** Split on top-level commas, ignoring commas nested inside parens. */
const splitTop = (s) => {
  const out = [];
  let depth = 0;
  let cur = "";
  for (const ch of s) {
    if (ch === "(") depth++;
    else if (ch === ")") depth--;
    if (ch === "," && depth === 0) {
      out.push(cur);
      cur = "";
    } else cur += ch;
  }
  if (cur.trim()) out.push(cur);
  return out.map((x) => x.trim());
};

/** Body of the first fn(...) call in `raw`, brace-balanced. */
const callBody = (raw, fn) => {
  const i = raw.toLowerCase().indexOf(`${fn}(`);
  if (i < 0) return null;
  let depth = 0;
  for (let j = i + fn.length; j < raw.length; j++) {
    if (raw[j] === "(") depth++;
    else if (raw[j] === ")" && --depth === 0)
      return raw.slice(i + fn.length + 1, j);
  }
  return null;
};

const NAMED = { white: [255, 255, 255], black: [0, 0, 0] };

const parseColor = (raw, what, vars = {}, depth = 0) => {
  const v = (raw ?? "").trim();
  if (!v) throw new Error(`${what}: empty value`);
  if (depth > 8) throw new Error(`${what}: var() nested too deeply`);

  // var(--x [, fallback])
  if (/^var\(/i.test(v)) {
    const [name, fallback] = splitTop(callBody(v, "var") ?? "");
    const target = vars[name?.replace(/^--/, "")];
    if (target !== undefined) return parseColor(target, what, vars, depth + 1);
    if (fallback) return parseColor(fallback, what, vars, depth + 1);
    throw new Error(`${what}: unresolved ${name}`);
  }

  // color-mix(in <space>, A [p%], B [q%])
  if (/^color-mix\(/i.test(v)) {
    const parts = splitTop(callBody(v, "color-mix") ?? "");
    if (parts.length < 3) throw new Error(`${what}: malformed color-mix`);
    const space = parts[0].replace(/\s+/g, " ").trim().toLowerCase();
    if (space !== "in oklab")
      throw new Error(
        `${what}: color-mix "${parts[0].trim()}" is not supported, only ` +
          `"in oklab". Convert it in the source theme first.`
      );
    const read = (part) => {
      const pct = /([\d.]+)%\s*$/.exec(part);
      return {
        col: parseColor(part.replace(/\s*[\d.]+%\s*$/, ""), what, vars, depth + 1),
        w: pct ? Number(pct[1]) / 100 : null,
      };
    };
    const a = read(parts[1]);
    const b = read(parts[2]);
    // Per spec, a single stated percentage implies the complement.
    const wa = a.w ?? (b.w !== null ? 1 - b.w : 0.5);
    const wb = b.w ?? 1 - wa;
    const total = wa + wb || 1;
    const [al, aa, ab] = oklchToOklab(a.col);
    const [bl, ba, bb] = oklchToOklab(b.col);
    const t = wb / total;
    return oklabToOklch(
      al + (bl - al) * t,
      aa + (ba - aa) * t,
      ab + (bb - ab) * t
    );
  }

  if (/^oklch\(/i.test(v)) return parseOklch(v, what);

  if (NAMED[v.toLowerCase()]) return srgbToOklch(...NAMED[v.toLowerCase()]);

  const hex = /^#([0-9a-f]{3}|[0-9a-f]{6})\b/i.exec(v);
  if (hex) {
    const h =
      hex[1].length === 3
        ? hex[1].split("").map((c) => parseInt(c + c, 16))
        : hex[1].match(/../g).map((c) => parseInt(c, 16));
    return srgbToOklch(...h);
  }

  const rgb = /^rgba?\(([^)]+)\)/i.exec(v);
  if (rgb) {
    const n = rgb[1]
      .replace(/\//g, " ")
      .split(/[\s,]+/)
      .filter(Boolean)
      .slice(0, 3)
      .map((x) =>
        x.endsWith("%") ? (Number(x.slice(0, -1)) / 100) * 255 : Number(x)
      );
    if (n.length < 3 || n.some((x) => !Number.isFinite(x)))
      throw new Error(`${what}: malformed rgb() "${v}"`);
    return srgbToOklch(n[0], n[1], n[2]);
  }

  throw new Error(
    `${what}: unsupported colour "${v}". Supported: oklch(), rgb(), hex, ` +
      `var(), color-mix(in oklab, ...).`
  );
};

const fmt = ({ l, c, h }) =>
  c < 0.0005
    ? `oklch(${l.toFixed(4)} 0 0)`
    : `oklch(${l.toFixed(4)} ${c.toFixed(4)} ${h.toFixed(2)})`;

const FLOOR_L = 0.02;
const clampL = (l) => Math.min(0.995, Math.max(FLOOR_L, l));
const shiftL = (col, d) => ({ ...col, l: clampL(col.l + d) });

// A near-neutral colour has no meaningful hue, so rotating towards it drags the
// ramp through unrelated hues (a warm-white foreground -> purple muted text
// crosses orange). Below HUE_EPS, borrow the other end's hue instead.
const HUE_EPS = 0.02;

// Shortest-arc hue lerp; an achromatic end borrows the other end's hue.
const mix = (a, b, t) => {
  let h;
  if (a.c < HUE_EPS && b.c < HUE_EPS) h = a.c >= b.c ? a.h : b.h;
  else if (a.c < HUE_EPS) h = b.h;
  else if (b.c < HUE_EPS) h = a.h;
  else {
    let d = ((b.h - a.h + 540) % 360) - 180;
    h = (a.h + d * t + 360) % 360;
  }
  return { l: a.l + (b.l - a.l) * t, c: a.c + (b.c - a.c) * t, h };
};

// ----------------------------------------------------------- css parsing ----

/** Slice out the {...} body of the first selector matching `re`, brace-balanced. */
const cutBlock = (css, re) => {
  const m = re.exec(css);
  if (!m) return null;
  const open = css.indexOf("{", m.index);
  if (open < 0) return null;
  let depth = 0;
  for (let i = open; i < css.length; i++) {
    if (css[i] === "{") depth++;
    else if (css[i] === "}" && --depth === 0)
      return { body: css.slice(open + 1, i), start: m.index, end: i + 1 };
  }
  return null;
};

/** Drop every @theme / @theme inline block - it is derived, not source of truth. */
const dropThemeBlocks = (css) => {
  for (;;) {
    const b = cutBlock(css, /@theme\b/);
    if (!b) return css;
    css = css.slice(0, b.start) + css.slice(b.end);
  }
};

// Names are kept as written: custom properties are case-sensitive, and the
// var() lookup in parseColor resolves against these exact keys.
const readVars = (body) => {
  const out = {};
  for (const m of body.matchAll(/--([a-zA-Z0-9-]+)\s*:\s*([^;]+);/g))
    out[m[1]] = m[2].trim();
  return out;
};

// [ \t] rather than \s: an empty "@key:" line must not capture the next line.
const meta = (css, key) =>
  css
    .match(new RegExp(`@${key}:[ \\t]*(.+)`))?.[1]
    ?.replace(/\*\/.*$/, "")
    .trim();

/** First of `names` present in `vars`, parsed. Throws naming every candidate. */
const anchor = (vars, block, ...names) => {
  const found = names.find((n) => vars[n]);
  if (!found)
    throw new Error(
      `${block} block: none of --${names.join(", --")} found in source theme`
    );
  return parseColor(vars[found], `${block} block: --${found}`, vars);
};

// -------------------------------------------------------- canonical ramps ----

// Lightness rhythm of netronome's own gray ramp (zinc + the 815/850 steps).
// Used as the interpolation axis so a converted ramp keeps the same cadence.
const GRAY = [
  ["50", 0.985], ["100", 0.967], ["200", 0.920], ["300", 0.871],
  ["400", 0.705], ["500", 0.552], ["600", 0.442], ["700", 0.370],
  ["800", 0.274], ["815", 0.256], ["850", 0.226], ["900", 0.210],
  ["950", 0.141],
];

// Tailwind v4 blue, as [step, L, C]. Only the shape matters: L positions set
// the ladder, C is used as a ratio against blue-500 so tints/shades taper.
const BLUE = [
  ["50", 0.970, 0.014], ["100", 0.932, 0.032], ["200", 0.882, 0.059],
  ["300", 0.809, 0.105], ["400", 0.707, 0.165], ["500", 0.623, 0.214],
  ["600", 0.546, 0.245], ["700", 0.488, 0.243], ["800", 0.424, 0.199],
  ["900", 0.379, 0.146], ["950", 0.282, 0.091],
];

const CANON = Object.fromEntries(GRAY);
const WHITE_CANON = 1.0; // virtual step, one notch above gray-50

const MIN_GAP = 0.006; // keeps adjacent steps distinguishable after lerping

// ------------------------------------------------------------ ramp build ----

/**
 * anchors: [{ at: canonical L, col }] in descending `at` order, covering both
 * ends. Every gray step is bracketed, so this is interpolation only - no
 * extrapolation guesswork.
 */
const buildGray = (anchors) => {
  const steps = [["white", WHITE_CANON], ...GRAY];
  const out = steps.map(([name, t]) => {
    if (t >= anchors[0].at) return [name, { ...anchors[0].col }];
    const i = anchors.findIndex((a) => a.at <= t);
    if (i <= 0) return [name, { ...anchors.at(-1).col }];
    const hi = anchors[i - 1], lo = anchors[i];
    return [name, mix(hi.col, lo.col, (hi.at - t) / (hi.at - lo.at))];
  });

  // Strictly decreasing lightness, top to bottom. Source themes routinely have
  // card == background or border == muted; without this they collapse.
  for (let i = 1; i < out.length; i++) {
    const prev = out[i - 1][1].l;
    if (out[i][1].l > prev - MIN_GAP) out[i][1].l = clampL(prev - MIN_GAP);
  }
  return out;
};

// netronome's own blue-500, for themes whose --primary carries no hue.
const FALLBACK_BLUE = { l: 0.623, c: 0.214, h: 259.81 };

/**
 * A neutral shadcn theme has a neutral --primary, and driving the blue ramp
 * from it turns every accent, link and chart line gray. Look for any hue the
 * theme does carry, else keep netronome's blue.
 */
const bluePrimary = (v, block) => {
  for (const n of ["primary", "ring", "accent", "chart-1", "chart-2", "chart-3"]) {
    if (!v[n]) continue;
    let col;
    try {
      col = parseColor(v[n], `${block}: --${n}`, v);
    } catch {
      continue; // unreadable accent, try the next candidate
    }
    if (col.c >= 0.04) return col;
  }
  console.warn(`  warn: ${block} has no chromatic accent, keeping netronome blue`);
  return FALLBACK_BLUE;
};

/**
 * primary sits at blue-500 exactly; the rest walk the canonical ladder,
 * rescaled per side so neither end clips. Hue is primary's throughout.
 */
const buildBlue = (primary) => {
  const c500 = BLUE.find(([s]) => s === "500");
  const upSpan = BLUE[0][1] - c500[1];
  const downSpan = c500[1] - BLUE.at(-1)[1];
  const up = (Math.min(0.98, primary.l + upSpan) - primary.l) / upSpan;
  const down = (primary.l - Math.max(0.12, primary.l - downSpan)) / downSpan;

  return BLUE.map(([step, l, c]) => {
    const d = l - c500[1];
    return [
      step,
      {
        l: clampL(primary.l + d * (d > 0 ? up : down)),
        c: Math.max(0, (primary.c * c) / c500[2]),
        h: primary.h,
      },
    ];
  });
};

// -------------------------------------------------------------- mapping ----

const lightAnchors = (v) => [
  // --color-white is the light-mode page/card surface (`bg-white dark:bg-gray-850`).
  { at: WHITE_CANON, col: anchor(v, "light", "background") },
  { at: CANON["100"], col: anchor(v, "light", "muted", "secondary", "card", "background") },
  { at: CANON["200"], col: anchor(v, "light", "border", "input", "muted") },
  { at: CANON["500"], col: anchor(v, "light", "muted-foreground") },
  { at: CANON["900"], col: anchor(v, "light", "foreground") },
  { at: CANON["950"], col: shiftL(anchor(v, "light", "foreground"), -0.05) },
];

const darkAnchors = (v) => {
  const fg = anchor(v, "dark", "foreground");
  const bg = anchor(v, "dark", "background");
  return [
    // Light end is TEXT here, dark end is SURFACE - see README-themes.md.
    { at: WHITE_CANON, col: shiftL(fg, 0.015) }, // `dark:text-white`, 143 uses
    { at: CANON["100"], col: fg },
    { at: CANON["400"], col: anchor(v, "dark", "muted-foreground") },
    { at: CANON["700"], col: anchor(v, "dark", "border", "input") },
    { at: CANON["800"], col: anchor(v, "dark", "card", "popover") },
    { at: CANON["900"], col: bg },
    { at: CANON["950"], col: shiftL(bg, -0.055) },
  ];
};

// Optional, opt-in: status colours keeping their default meaning is the norm.
const EXTRAS = [
  ["red-500", [[0, 45], [330, 360]]],
  ["emerald-500", [[130, 175]]],
  ["amber-400", [[60, 105]]],
  ["purple-400", [[280, 330]]],
  ["indigo-500", [[255, 285]]],
];

const buildExtras = (v) => {
  const pool = ["destructive", "chart-1", "chart-2", "chart-3", "chart-4", "chart-5", "accent", "secondary"]
    .filter((n) => v[n])
    .map((n) => {
      try {
        return parseColor(v[n], n, v);
      } catch {
        return null; // extras are opt-in; an unreadable one is not fatal
      }
    })
    .filter((c) => c && c.c > 0.04);

  const out = [];
  for (const [name, ranges] of EXTRAS) {
    const hit = pool
      .filter((c) => ranges.some(([a, b]) => c.h >= a && c.h <= b))
      .sort((a, b) => b.c - a.c)[0];
    if (!hit) continue;
    out.push([name, hit]);
    if (name === "emerald-500") out.push(["emerald-400", shiftL(hit, 0.08)]);
  }
  return out;
};

/**
 * Text drawn on an accent surface (primary buttons). netronome would otherwise
 * hardcode white, which is unreadable on the light pastel accents many of these
 * themes use. Prefer the theme's own --primary-foreground, since that is the
 * pairing its author intended; fall back to plain black/white when that pairing
 * does not actually clear AA.
 */
const onAccent = (v, accents, block) => {
  // Buttons use bg-blue-500 in light mode and dark:bg-blue-600, so whatever we
  // pick has to clear AA against every accent step it can land on.
  const worst = (fg) => Math.min(...accents.map((a) => contrast(fg, a)));

  let best = null;
  if (v["primary-foreground"]) {
    try {
      best = parseColor(v["primary-foreground"], `${block}: --primary-foreground`, v);
    } catch {
      best = null;
    }
  }
  if (best && worst(best) >= AA) return best;

  const fallback = worst(WHITE) >= worst(BLACK) ? WHITE : BLACK;
  if (best) {
    console.warn(
      `  warn: ${block} --primary-foreground only ${worst(best).toFixed(2)}:1 ` +
        `on the accent, using ${fallback === WHITE ? "white" : "black"} instead`
    );
  }
  return fallback;
};

// ----------------------------------------------------------------- emit ----

const emit = (selector, gray, blue, extras, onAcc) => {
  const line = ([n, col]) => `  --color-${n}: ${fmt(col)};`;
  const grayLines = gray.map(([n, col]) =>
    line([n === "white" ? "white" : `gray-${n}`, col])
  );
  const parts = [
    grayLines.join("\n"),
    blue.map(([s, c]) => line([`blue-${s}`, c])).join("\n"),
    line(["on-accent", onAcc]),
  ];
  if (extras.length) parts.push(extras.map(line).join("\n"));
  return `${selector} {\n${parts.join("\n\n")}\n}`;
};

// ----------------------------------------------------------------- main ----

process.on("uncaughtException", (e) => {
  console.error(`error: ${e.message}`);
  process.exit(1);
});

const raw = process.argv.slice(2);
const wantExtras = raw.includes("--extras");
const args = raw.filter((a) => a !== "--extras");
const flag = (name) => {
  const i = args.indexOf(`--${name}`);
  return i < 0 ? undefined : args[i + 1];
};
const [input, output] = args.filter(
  (a, i) => !a.startsWith("--") && !args[i - 1]?.startsWith("--")
);

if (!input || !output) {
  console.error("usage: node scripts/convert-theme.mjs <input.css> <output.css> [--id nord] [--extras]");
  process.exit(1);
}

const src = readFileSync(input, "utf8");
const id = flag("id") ?? basename(output).replace(/\.css$/, "");
const body = dropThemeBlocks(src);

const root = cutBlock(body, /(^|[\s};])\:root\b(?![.\w-])/);
const dark = cutBlock(body, /(^|[\s};])\.dark\b(?![\w-])/);
if (!root) throw new Error(`${input}: no ":root { ... }" block found`);
if (!dark) throw new Error(`${input}: no ".dark { ... }" block found`);

const lv = readVars(root.body);
const dv = readVars(dark.body);

const blocks = [
  { sel: `:root.theme-${id}`, v: lv, anchors: lightAnchors(lv) },
  { sel: `:root.theme-${id}.dark`, v: dv, anchors: darkAnchors(dv) },
].map(({ sel, v, anchors }) => {
  const gray = buildGray(anchors);
  let blue = buildBlue(bluePrimary(v, sel));
  const steps = (r) => ["500", "600"].map((s) => r.find(([n]) => n === s)[1]);

  let onAcc = onAccent(v, steps(blue), sel);

  // An accent parked at mid lightness contrasts poorly with black AND white, so
  // no text colour clears AA. Shift the whole ramp (not just 500/600, which
  // would break monotonicity) the short way out of that dead zone. Hue and
  // chroma are untouched, so the theme still reads as itself.
  const worst = (fg, r) => Math.min(...steps(r).map((a) => contrast(fg, a)));
  if (worst(onAcc, blue) < AA) {
    const dir = steps(blue)[0].l >= 0.6 ? +1 : -1; // lighten for black text, else darken
    for (let d = 0.01; d <= 0.16; d += 0.01) {
      const shifted = blue.map(([s, c]) => [s, shiftL(c, dir * d)]);
      const cand = onAccent(v, steps(shifted), `${sel} (nudged)`);
      if (worst(cand, shifted) >= AA) {
        console.warn(
          `  note: ${sel} accent nudged ${dir > 0 ? "+" : "-"}${d.toFixed(2)} L ` +
            `so button text can reach AA`
        );
        blue = shifted;
        onAcc = cand;
        break;
      }
    }
  }

  return { sel, gray, blue, onAcc, extras: wantExtras ? buildExtras(v) : [] };
});

// Self-check: the whole point of the tool is that nothing is missing and both
// ramps read in the right direction. Fail loudly rather than ship a broken file.
const REQUIRED = [
  "white",
  ...GRAY.map(([s]) => `gray-${s}`),
  ...BLUE.map(([s]) => `blue-${s}`),
];
for (const { sel, gray, blue, onAcc } of blocks) {
  // Primary buttons must be legible. This is the whole reason --color-on-accent
  // exists, so a theme that cannot satisfy it is a hard failure, not a warning.
  for (const s of ["500", "600"]) {
    const c = contrast(onAcc, blue.find(([n]) => n === s)[1]);
    if (c < AA)
      throw new Error(
        `${sel}: on-accent text is only ${c.toFixed(2)}:1 on blue-${s} (need ${AA})`
      );
  }

  const names = new Set([
    ...gray.map(([n]) => (n === "white" ? "white" : `gray-${n}`)),
    ...blue.map(([s]) => `blue-${s}`),
  ]);
  const missing = REQUIRED.filter((n) => !names.has(n));
  if (missing.length) throw new Error(`${sel}: missing ${missing.join(", ")}`);
  for (const ramp of [gray, blue])
    ramp.reduce((prev, [n, col]) => {
      // A true-black theme (kitsune: --background is oklch(0 0 0)) saturates at
      // clampL's floor, so the last steps legitimately tie. Only flag ties that
      // happen with headroom left - a real collapse.
      if (prev && col.l >= prev.l && prev.l > FLOOR_L + 1e-6)
        throw new Error(`${sel}: --color-${n} is not darker than the step above it`);
      return col;
    }, null);
}

// Advisory only. These are the highest-traffic fg/bg pairs in web/src; a thin
// gap here is exactly what the hand-tuning pass is for.
const PAIRS = [
  [0, "gray-600", "white"], [0, "gray-900", "white"],
  [0, "gray-500", "gray-50"], [0, "blue-600", "white"],
  [1, "gray-400", "gray-850"], [1, "gray-100", "gray-900"],
  [1, "gray-300", "gray-850"], [1, "blue-400", "gray-850"],
  [1, "white", "gray-850"],
];
// Threshold sits under every pair's value in netronome's default ramp (lowest
// is gray-500 on gray-50 at 0.43), so a warning means worse than the baseline.
const MIN_CONTRAST = 0.38;

for (const [i, fg, bg] of PAIRS) {
  const { sel, gray, blue } = blocks[i];
  const look = (n) =>
    (n.startsWith("blue-") ? blue : gray).find(
      ([s]) => (n === "white" ? "white" : `gray-${s}`) === n || `blue-${s}` === n
    )?.[1];
  const d = Math.abs(look(fg).l - look(bg).l);
  if (d < MIN_CONTRAST)
    console.warn(`  warn: ${sel} ${fg} on ${bg} is only ${d.toFixed(2)} L apart`);
}

const description = meta(src, "description");
const header = [
  `/* @name: ${meta(src, "name") ?? id}`,
  ...(description ? [` * @description: ${description}`] : []),
  ` * @premium: ${meta(src, "premium") ?? "true"}`,
  ` *`,
  ` * Converted from a qui theme by scripts/convert-theme.mjs. Hand-tuned after.`,
  ` */`,
].join("\n");

writeFileSync(
  output,
  `${header}\n\n${blocks.map((b) => emit(b.sel, b.gray, b.blue, b.extras, b.onAcc)).join("\n\n")}\n`
);

console.log(
  `${input} -> ${output}  (id: ${id}, ${REQUIRED.length} vars x 2 blocks${wantExtras ? " + extras" : ""})`
);
