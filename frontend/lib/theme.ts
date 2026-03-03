import type { CSSProperties } from "react";

// ─── Font Stacks ────────────────────────────────────────────
export const fonts = {
  mono: { fontFamily: "var(--font-jetbrains)" } satisfies CSSProperties,
  display: { fontFamily: "var(--font-syne)" } satisfies CSSProperties,
} as const;

// ─── Inline Styles (non-Tailwind) ───────────────────────────
export const inlineStyles = {
  scanLines: {
    background:
      "repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(34,197,94,0.1) 2px, rgba(34,197,94,0.1) 4px)",
  } satisfies CSSProperties,
  blinkCaret: {
    animation: "blink-caret 1s step-end infinite",
  } satisfies CSSProperties,
  dotPulse: {
    animation: "dot-pulse 2s ease-in-out infinite",
  } satisfies CSSProperties,
} as const;

// ─── Tailwind Class Strings ─────────────────────────────────
export const tw = {
  // Page-level
  page: "min-h-screen bg-black text-slate-200",
  scanOverlay: "pointer-events-none fixed inset-0 z-20 opacity-[0.03]",

  // Surfaces
  card: "rounded border border-slate-800 bg-[#0a0a0a]",
  cardOverflow: "overflow-hidden rounded border border-slate-800 bg-[#0a0a0a]",
  cardHover:
    "rounded border border-slate-800 bg-[#0a0a0a] transition hover:border-green-500/30",

  // Terminal window chrome
  windowBar: "flex items-center gap-2 border-b border-slate-800 px-4 py-2.5",
  dotClose: "h-2.5 w-2.5 rounded-full bg-[#ff5f57]",
  dotMinimize: "h-2.5 w-2.5 rounded-full bg-[#febc2e]",
  dotMaximize: "h-2.5 w-2.5 rounded-full bg-[#28c840]",
  dotCloseLg: "h-3 w-3 rounded-full bg-[#ff5f57]",
  dotMinimizeLg: "h-3 w-3 rounded-full bg-[#febc2e]",
  dotMaximizeLg: "h-3 w-3 rounded-full bg-[#28c840]",
  windowLabel: "ml-3 text-xs text-slate-600",

  // Buttons
  btnPrimary:
    "rounded border border-green-500 bg-green-500/10 px-6 py-3 text-sm font-medium text-green-400 transition hover:bg-green-500/20",
  btnSecondary:
    "rounded border border-slate-800 px-6 py-3 text-sm text-slate-500 transition hover:border-slate-700 hover:text-slate-300",
  btnDanger:
    "rounded border border-red-500/30 px-3 py-1.5 text-xs font-medium text-red-400 transition hover:bg-red-500/10",
  btnActionGreen:
    "rounded border border-green-500/30 px-2.5 py-1 text-xs font-medium text-green-400 transition hover:bg-green-500/10",
  btnActionYellow:
    "rounded border border-yellow-500/30 px-2.5 py-1 text-xs font-medium text-yellow-400 transition hover:bg-yellow-500/10",
  btnActionSlate:
    "rounded border border-slate-700 px-2.5 py-1 text-xs font-medium text-slate-300 transition hover:border-green-500/30 hover:text-green-300",

  // Navigation
  navWrapper:
    "relative z-10 mx-auto flex max-w-5xl items-center justify-between px-6 py-6",
  navLinks: "flex items-center gap-6 text-sm text-slate-500",
  navLink: "transition hover:text-green-400",
  navActive: "text-green-400",

  // Typography
  heading: "font-extrabold uppercase tracking-tight text-white",
  breadcrumb: "text-sm text-slate-600",
  body: "text-sm leading-relaxed text-slate-300",
  muted: "text-sm text-slate-500",
  accent: "text-green-400",

  // Table
  tableHeader:
    "border-b border-slate-800 text-left text-xs uppercase tracking-wider text-slate-600",
  tableRow:
    "border-b border-slate-800/50 transition hover:bg-white/[0.02]",

  // Error
  errorBanner:
    "rounded border border-red-500/30 bg-red-500/5 px-4 py-3 text-sm text-red-400",

  // Footer
  footer: "mx-auto max-w-5xl border-t border-slate-800 px-6 py-6",
  footerInner:
    "flex flex-col items-center justify-between gap-3 text-xs text-slate-600 md:flex-row",

  // Divider
  divider: "border-t border-dashed border-slate-800",

  // Pagination
  paginationLink:
    "rounded border border-slate-800 px-3 py-1 text-slate-400 transition hover:border-green-500/30 hover:text-green-400",
  paginationDisabled:
    "rounded border border-slate-800/50 px-3 py-1 text-slate-700",
} as const;

// ─── Status Colors ──────────────────────────────────────────
export type CheckStatus = "up" | "late" | "down" | "paused";

export const statusStyles = {
  up: {
    dot: "bg-green-400",
    badge: "text-green-400 border-green-500/30 bg-green-500/10",
    text: "text-green-400",
    border: "border-green-500/30",
  },
  late: {
    dot: "bg-yellow-400",
    badge: "text-yellow-400 border-yellow-500/30 bg-yellow-500/10",
    text: "text-yellow-400",
    border: "border-yellow-500/30",
  },
  down: {
    dot: "bg-red-400",
    badge: "text-red-400 border-red-500/30 bg-red-500/10",
    text: "text-red-400",
    border: "border-red-500/30",
  },
  paused: {
    dot: "bg-slate-500",
    badge: "text-slate-400 border-slate-500/30 bg-slate-500/10",
    text: "text-slate-400",
    border: "border-slate-500/30",
  },
} as const satisfies Record<
  CheckStatus,
  { dot: string; badge: string; text: string; border: string }
>;
