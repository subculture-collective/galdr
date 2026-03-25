# Galdr Design System

## Brand direction

**Brand:** Galdr  
**Meaning:** Norse incantation / spellcraft  
**Positioning:** Professional customer-health intelligence with a dark, edgy, ritual-tech aesthetic.

This system blends:

- `26-industrial` for tactile product seriousness and mechanical confidence
- `19-minimal-dark` for premium nocturnal polish and readability
- `14-bold-typography` accents for controlled punk energy

## Experience principles

1. **Ritual over noise** — the UI should feel intentional and “cast,” never chaotic.
2. **Dark by default** — depth from layered charcoals, not pure black.
3. **Sharp confidence** — cards/buttons carry strong edges and subtle hard shadows.
4. **Professional legibility** — AA+ contrast and clean hierarchy at all breakpoints.
5. **Measured motion** — short, deliberate transitions (150–250ms), no flashy gimmicks.

## Core design tokens

### Color tokens

- `--galdr-bg`: `#0b0b12`
- `--galdr-bg-elevated`: `#12121a`
- `--galdr-surface`: `#171724`
- `--galdr-surface-soft`: `#1f1f2e`
- `--galdr-border`: `#2d2d40`
- `--galdr-fg`: `#f2f2f8`
- `--galdr-fg-muted`: `#a8a8bc`
- `--galdr-accent`: `#8b5cf6` (arcane violet)
- `--galdr-accent-2`: `#22d3ee` (frost-cyan)
- `--galdr-danger`: `#f43f5e`
- `--galdr-success`: `#34d399`

### Typography

- **Display + UI:** `Space Grotesk`, `Inter`, `system-ui`, sans-serif
- **Technical labels:** `JetBrains Mono`, monospace
- Headings: weight `700`
- Body: weight `400–500`
- Labels/chips: uppercase + tracking `0.08em`

### Shape and depth

- Radius: `10px`, `14px`, `18px`
- Border: 1px, high-contrast dark edge
- Shadow style:
  - ambient: `0 10px 30px rgba(0,0,0,0.35)`
  - accent glow: `0 0 0 1px rgba(139,92,246,0.35), 0 0 24px rgba(139,92,246,0.18)`

## Component language

### Buttons

- **Primary:** violet gradient fill, bright text, slight glow on hover
- **Secondary:** elevated dark surface with subtle border and hover brighten
- States:
  - hover: `translateY(-1px)`
  - active: `translateY(1px)` + reduced shadow
  - focus-visible: 2px accent ring

### Cards / panels

- Elevated dark surfaces with a soft top highlight and hard-ish outer edge
- Optional “rune rail” accent (2px violet strip) for featured modules

### Inputs

- Dark recessed field
- Strong focus ring with accent and no outline suppression without replacement

### Status treatment

- healthy: emerald
- at-risk: amber/violet blend
- critical: rose
- Status should use icon + text, not color only.

## Layout and spacing

- Container max width: `1280px` (`max-w-7xl`)
- Section rhythm: `py-16` mobile, `py-24` desktop
- Grid spacing: `gap-4` to `gap-8`
- First paint pages to align first: Landing, Pricing, Auth, App shell

## Motion and interaction

- Default transition: `200ms ease`
- Elevation on hover should remain subtle (no dramatic scale)
- Respect `prefers-reduced-motion`

## Accessibility guardrails

- Minimum text contrast: 4.5:1
- Visible keyboard focus for all interactive controls
- Semantic elements over clickable `<div>` wrappers
- Touch targets >= 44px

## Implementation rollout

### Phase 1 (foundation)

1. Global tokens + typography in `web/src/index.css`
2. Global focus ring / motion-reduction defaults
3. Shared utility classes (`galdr-panel`, `galdr-card`, `galdr-button-*`)

### Phase 2 (brand surfaces)

1. Rebrand core UI strings to **Galdr**
2. Landing page + marketing sections refactor
3. Sidebar/app shell visual alignment

### Phase 3 (full convergence)

1. Auth + onboarding screens styling alignment
2. Pricing/SEO/legal pages finish
3. Chart color/token normalization and hardcoded color cleanup

## Acceptance criteria

- No new hardcoded one-off hex values in components where a token exists
- All primary CTAs use shared Galdr button patterns
- Landing, pricing, auth, and app shell all visually belong to one system
- Keyboard-only navigation has clear focus indicators on every route