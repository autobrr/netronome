# Converting a qui theme to a netronome theme

`convert-theme.mjs` is a **one-shot migration tool**, not a build step. Run it once
per theme, look at the result, tune it by hand, commit the tuned file to the private
themes repo. Nothing in netronome's build ever calls it.

```sh
node scripts/convert-theme.mjs <input.css> <output.css> [--id nord] [--extras]
```

Plain Node ESM, no dependencies, no install step.

```sh
node scripts/convert-theme.mjs ../qui/web/src/themes/premium/nord.css nord.css
node scripts/convert-theme.mjs ../qui/web/src/themes/premium/synthwave.css synthwave.css --id synthwave
```

- `--id` sets the CSS class suffix (`:root.theme-<id>`). Defaults to the output
  filename, so `nord.css` gives `theme-nord`. The id must match the filename in the
  themes repo, because `web/src/config/themes.ts` derives the theme id from the
  filename.
- `--extras` additionally emits `--color-red-500`, `--color-emerald-400/500`,
  `--color-amber-400`, `--color-purple-400`, `--color-indigo-500` when the source
  has a colour in the matching hue range. **Off by default and usually right to
  leave off** — status colours keeping their default meaning is intentional. Turn
  it on only for themes where a green success badge would look absurd.

## Where the output goes

Converted themes are committed to the **private themes repo**,
`autobrr/qui-premium-themes`, in the `netronome/` subdirectory. They are **not**
committed to this repo — `web/src/themes/premium/` is populated at release time and
is expected to be absent in source builds.

The only theme CSS that lives in this repo is `web/src/themes/__dev__/debug.css`, a
deliberately garish fixture so the ramp-override engine stays exercisable without a
`THEMES_REPO_TOKEN`.

## Input and output

Input is a qui theme: a metadata header, `:root { ... }` for light, `.dark { ... }`
for dark, in shadcn semantic tokens. Any `@theme inline { ... }` block is stripped —
it is derived, not source of truth.

Output is a netronome theme: the same metadata header, then two blocks that redefine
Tailwind's colour ramp.

```css
/* @name: Nord
 * @description: An arctic, north-bluish color palette.
 * @premium: true
 */

:root.theme-nord      { --color-white; --color-gray-50…950; --color-blue-50…950 }
:root.theme-nord.dark { …the same 25 vars… }
```

Both blocks always carry all 25 vars, including netronome's extra `--color-gray-815`
and `--color-gray-850` steps. The script refuses to write a file that is missing one
or whose ramps are not monotonic.

Everything stays in `oklch()` from end to end — valid CSS, and no colour-space maths
to get wrong. The flip side: **a source var that is not `oklch()` is an error**, not
a guess. Convert it in the source theme first. Very light or very saturated results
may sit outside sRGB; browsers gamut-map them, which is fine but shifts them
slightly from the printed numbers.

## Mapping

| netronome var | light block ← | dark block ← |
| --- | --- | --- |
| `--color-white` | `--background` | `--foreground`, nudged brighter |
| `gray-50` | interpolated | interpolated |
| `gray-100` | `--muted` / `--secondary` / `--card` | `--foreground` |
| `gray-200` | `--border` / `--input` | interpolated |
| `gray-300` | interpolated | interpolated |
| `gray-400` | interpolated | `--muted-foreground` |
| `gray-500` | `--muted-foreground` | interpolated |
| `gray-600` | interpolated | interpolated |
| `gray-700` | interpolated | `--border` / `--input` |
| `gray-800` | interpolated | `--card` / `--popover` |
| `gray-815`, `gray-850` | interpolated | interpolated |
| `gray-900` | `--foreground` | `--background` |
| `gray-950` | `--foreground`, darker | `--background`, darker |
| `blue-500` | `--primary` | `--primary` |
| `blue-*` | ladder walked from `--primary` | ladder walked from `--primary` |

Intermediate steps are interpolated in OKLCH between the anchors on either side.
The interpolation axis is netronome's own default gray ramp (zinc plus the 815/850
steps), so a converted ramp keeps the same perceptual cadence as the default theme
rather than being a flat two-point lerp. Anchoring on `--card` / `--muted` /
`--border` is what keeps the theme's character — Nord's dark card really is a
distinctly lighter blue-gray than its background, and a pure lerp would flatten that
out.

The blue ramp pins `--primary` at `blue-500` exactly and walks the canonical Tailwind
blue lightness ladder outward, keeping the primary's hue throughout and scaling
chroma by the canonical profile so tints and shades taper. Each side is rescaled
independently so a very light or very dark primary compresses one end instead of
clipping it.

## The dark block is not the light block inverted

This is the part that is easy to get wrong, so here is the reasoning.

The tempting move is to mirror the ramp in dark mode — make `gray-50` the darkest and
`gray-950` the lightest, on the theory that dark mode is "upside down". **That is
wrong**, and the reason is that a Tailwind step number encodes an *absolute*
lightness, not a semantic role. `dark:bg-gray-850` and `text-gray-900` sit in the same
stylesheet and both resolve against whatever ramp is active.

What actually changes between the two blocks is **which end of the ramp is doing the
work**:

- **Light block:** the light end is surface (`bg-white`, `bg-gray-50`, `bg-gray-100`),
  the dark end is text (`text-gray-900`, `text-gray-700`, `text-gray-600`).
- **Dark block:** the dark end is surface (`dark:bg-gray-800` ×81, `dark:bg-gray-850`
  ×54, `dark:bg-gray-900` ×23), the light end is text (`dark:text-gray-400` ×282,
  `dark:text-gray-300` ×78, `dark:text-gray-100` ×14).

So both blocks run light→dark, 50 to 950. What differs is where the source theme's
background and foreground get *anchored*: in light, background lands near the top of
the ramp and foreground near the bottom; in dark, foreground lands near the top and
background near the bottom. The ramp is re-anchored, not reversed.

Two consequences worth stating explicitly:

1. **`--color-white` stays light in the dark block.** netronome uses
   `dark:text-white` 143 times for headings and `dark:bg-white` zero times. Setting
   it to a dark value — which a naive "invert everything" pass does — makes every
   heading in the app vanish. It is anchored to the dark `--foreground`, nudged a
   little brighter so `gray-50` has somewhere to sit.
2. **The blue ramp is not reversed either.** `dark:text-blue-400` (×66) has to be a
   bright link on a dark surface while `dark:bg-blue-900/20` has to be a dark navy
   chip and `bg-blue-500` (×50, no dark variant at all) has to stay a solid button
   fill. That is exactly stock Tailwind blue semantics, so the ladder keeps its
   direction and only its anchor colour changes.

The dark surface steps deserve one note. netronome's default packs five steps into
the dark end — `gray-800` (raised card) > `gray-815` > `gray-850` (panel) >
`gray-900` (page) > `gray-950` (deepest). The converter anchors `800` to the source's
`--card`, `900` to `--background`, and interpolates `815`/`850` between them, which
holds for every qui theme checked because shadcn themes always make the dark card
lighter than the dark background.

## Warnings

The script writes the file and warns; it does not block. Warnings are the to-do list
for the tuning pass.

- `no chromatic accent, keeping netronome blue` — the source `--primary` is neutral
  (common in minimal/mono themes). Driving the blue ramp from it would turn every
  link, button and chart line gray, so the script falls back to netronome's own blue
  after trying `--ring`, `--accent` and the chart colours. Pick a hue by hand if the
  theme deserves one.
- `<fg> on <bg> is only N L apart` — a high-traffic text/surface pair has less
  lightness separation than the same pair in netronome's default ramp. Nord trips
  this on `blue-600` on white, because Nord's primary is a pale arctic blue that
  makes for washed-out light-mode links. Darken `blue-600`/`blue-700` by hand.

## The hand-tuning pass

The converter gets a theme to "plausible", not to "shipped". Before committing:

1. Load it in the app and toggle dark mode. `web/src/config/themes.ts` picks up any
   `.css` under `web/src/themes/premium/`, so dropping the file there is enough to
   try it locally.
2. Check the crowded dark surfaces — `gray-800` / `gray-815` / `gray-850` / `gray-900`
   stack up in nested cards and can end up indistinguishable when the source theme
   has card and background close together.
3. Check light-mode `gray-100` vs `gray-200`. Many shadcn themes set `--muted` and
   `--border` to the same colour; the monotonicity pass separates them by the bare
   minimum, which reads as flat.
4. Check the swatches. The picker samples `gray-950`, `gray-800`, `blue-500`,
   `emerald-500`, `gray-100` from the *last* block in the file, i.e. the dark one.
5. Resolve every warning above, or decide it is acceptable and move on.
