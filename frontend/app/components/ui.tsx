import Link from "next/link";
import { tw, fonts, inlineStyles } from "@/lib/theme";

// ─── Scan Lines ─────────────────────────────────────────────
export function ScanLines() {
  return <div className={tw.scanOverlay} style={inlineStyles.scanLines} />;
}

// ─── Terminal Window Chrome ─────────────────────────────────
export function WindowChrome({
  label,
  large,
}: {
  label: string;
  large?: boolean;
}) {
  const dot = large
    ? { c: tw.dotCloseLg, m: tw.dotMinimizeLg, x: tw.dotMaximizeLg }
    : { c: tw.dotClose, m: tw.dotMinimize, x: tw.dotMaximize };

  return (
    <div className={tw.windowBar}>
      <div className={dot.c} />
      <div className={dot.m} />
      <div className={dot.x} />
      <span className={tw.windowLabel}>{label}</span>
    </div>
  );
}

// ─── Terminal Code Block ────────────────────────────────────
export function TerminalBlock({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className={tw.cardOverflow}>
      <WindowChrome label={label} />
      <pre className="overflow-x-auto scrollbar-none p-5 text-sm leading-relaxed text-green-300/90">
        {children}
      </pre>
    </div>
  );
}

// ─── Section Heading (Syne font) ────────────────────────────
export function SectionHeading({
  id,
  size = "lg",
  children,
}: {
  id?: string;
  size?: "sm" | "lg" | "xl";
  children: React.ReactNode;
}) {
  const sizeClass =
    size === "xl"
      ? "text-3xl md:text-5xl"
      : size === "lg"
        ? "text-xl md:text-2xl"
        : "text-lg";

  return (
    <h2
      id={id}
      className={`scroll-mt-24 ${tw.heading} ${sizeClass}`}
      style={fonts.display}
    >
      {children}
    </h2>
  );
}

// ─── Page Heading (breadcrumb + title) ──────────────────────
export function PageHeading({
  breadcrumb,
  children,
}: {
  breadcrumb: string;
  children: React.ReactNode;
}) {
  return (
    <header>
      <p className={tw.breadcrumb}>{">"} {breadcrumb}</p>
      <h1
        className={`mt-2 text-2xl md:text-4xl ${tw.heading}`}
        style={fonts.display}
      >
        {children}
      </h1>
    </header>
  );
}

// ─── Error Banner ───────────────────────────────────────────
export function ErrorBanner({ message }: { message: string }) {
  return (
    <div className={tw.errorBanner}>
      <span className="font-bold">!</span> {message}
    </div>
  );
}

// ─── Divider ────────────────────────────────────────────────
export function Divider({ className }: { className?: string }) {
  return (
    <div className={className}>
      <div className={tw.divider} />
    </div>
  );
}

// ─── Pulse Logo ─────────────────────────────────────────────
export function PulseLogo({ asLink }: { asLink?: boolean }) {
  const inner = (
    <>
      {">"} pulse_
      <span style={inlineStyles.blinkCaret} className={tw.navActive}>
        █
      </span>
    </>
  );

  if (asLink) {
    return (
      <Link href="/" className={`text-lg ${tw.navActive} transition hover:text-green-300`}>
        {inner}
      </Link>
    );
  }

  return <span className={`text-lg ${tw.navActive}`}>{inner}</span>;
}

// ─── Footer ─────────────────────────────────────────────────
export function Footer({
  links,
  maxWidth = "max-w-5xl",
}: {
  links: { label: string; href: string }[];
  maxWidth?: string;
}) {
  return (
    <footer className={`relative z-10 ${maxWidth} mx-auto border-t border-slate-800 px-6 py-6`}>
      <div className={tw.footerInner}>
        <span>pulse v1.0.0</span>
        <div className="flex gap-4">
          {links.map((link) =>
            link.href.startsWith("/") ? (
              <Link key={link.href} href={link.href} className={tw.navLink}>
                [{link.label}]
              </Link>
            ) : (
              <a key={link.href} href={link.href} className={tw.navLink}>
                [{link.label}]
              </a>
            ),
          )}
        </div>
        <span>&copy; 2026 pulse</span>
      </div>
    </footer>
  );
}
