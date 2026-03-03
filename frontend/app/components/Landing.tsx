import Link from "next/link";
import { SignedIn, SignedOut, SignInButton, UserButton } from "@clerk/nextjs";
import { LogIn } from "lucide-react";
import { isClerkUiEnabled } from "@/lib/clerk-config";
import { tw, fonts, statusStyles } from "@/lib/theme";
import { ScanLines, WindowChrome, Footer, PulseLogo, Divider, NavItem } from "./ui";

const TERMINAL_LINES = [
  { prompt: true, text: 'curl -s https://pulse.dev/v1/ping/tok_abc123' },
  { prompt: false, text: '{"status":"ok","latency_ms":23}' },
  { prompt: true, text: "pulse checks list" },
  { prompt: false, text: "" },
  {
    prompt: false,
    text: "NAME              INTERVAL   STATUS   LAST PING",
    dim: true,
  },
  { prompt: false, text: "backup-daily      24h        UP       2 min ago", status: "up" as const },
  { prompt: false, text: "deploy-webhook    5m         UP       30s ago", status: "up" as const },
  { prompt: false, text: "report-gen        1h         LATE     1h 15m ago", status: "late" as const },
  { prompt: false, text: "sync-prod         30m        DOWN     3h ago", status: "down" as const },
];

const FEATURES = [
  { flag: "--ping", title: "One HTTP request.", desc: "GET or POST. That's your entire integration." },
  { flag: "--alert", title: "Multi-channel alerts.", desc: "Email, Slack, webhooks. Routed, not spammed." },
  { flag: "--grace", title: "No false alarms.", desc: "Configurable grace periods. Set your margin." },
  { flag: "--status", title: "Up. Late. Down.", desc: "Three states. Zero ambiguity. Always current." },
];

const CODE_EXAMPLES = [
  {
    label: "crontab",
    code: `# Monitor your backup job\n*/5 * * * * /opt/bin/backup.sh \\\\\n  && curl -fsS https://pulse.dev/v1/ping/TOKEN`,
  },
  {
    label: "docker",
    code: `# Docker healthcheck\nHEALTHCHECK --interval=60s --timeout=5s \\\\\n  CMD curl -f https://pulse.dev/v1/ping/TOKEN \\\\\n  || exit 1`,
  },
  {
    label: "systemd",
    code: `# systemd timer hook\n[Service]\nExecStartPost=/usr/bin/curl -fsS \\\\\n  https://pulse.dev/v1/ping/TOKEN`,
  },
];

const STATUS_LEGEND = [
  { status: "UP", key: "up" as const, desc: "Ping received on time" },
  { status: "LATE", key: "late" as const, desc: "Past due, within grace" },
  { status: "DOWN", key: "down" as const, desc: "Past grace period" },
  { status: "PAUSED", key: "paused" as const, desc: "Monitoring suspended" },
];

export default function Landing() {
  return (
    <div className={tw.page} style={fonts.mono}>
      <ScanLines />

      {/* Navigation */}
      <nav className={tw.navWrapper}>
        <PulseLogo />
        <div className={tw.navLinks}>
          <NavItem href="/docs" label="docs" icon="docs" />
          <NavItem href="#" label="github" icon="github" external />
          {isClerkUiEnabled ? (
            <>
              <SignedOut>
                <SignInButton mode="modal">
                  <button type="button" className={`${tw.navActive} transition hover:text-green-300`}>
                    <LogIn size={16} className="sm:hidden" />
                    <span className="hidden sm:inline">[login]</span>
                  </button>
                </SignInButton>
              </SignedOut>
              <SignedIn>
                <NavItem href="/checks" label="dashboard" icon="dashboard" />
                <UserButton />
              </SignedIn>
            </>
          ) : (
            <NavItem href="/checks" label="checks" icon="checks" />
          )}
        </div>
      </nav>

      {/* Hero */}
      <section className="relative mx-auto max-w-5xl px-6 pt-16 pb-10 md:pt-28">
        {/* Terminal window */}
        <div className={tw.cardOverflow}>
          <WindowChrome label="bash — pulse" large />
          <div className="no-scrollbar overflow-x-auto space-y-1 p-5 text-sm leading-relaxed">
            {TERMINAL_LINES.map((line, i) => (
              <div key={i} className={line.dim ? "text-slate-600" : ""}>
                {line.prompt && <span className="text-green-500">$ </span>}
                <span className={line.status ? statusStyles[line.status].text : ""}>
                  {line.text}
                </span>
              </div>
            ))}
            <div>
              <span className="text-green-500">$ </span>
              <span className="inline-block h-4 w-2 bg-green-400" style={{ animation: "blink-caret 1s step-end infinite" }} />
            </div>
          </div>
        </div>

        {/* Headline */}
        <div className="mt-14">
          <h1 className={`text-4xl md:text-7xl ${tw.heading}`} style={fonts.display}>
            Your crons.
            <br />
            <span className={tw.accent}>Monitored.</span>
          </h1>
          <p className="mt-6 max-w-lg text-base leading-relaxed text-slate-500">
            Dead-simple heartbeat monitoring for developers who ship.
            One ping. Full visibility. Zero config overhead.
          </p>
          <div className="mt-8 flex flex-wrap gap-4">
            {isClerkUiEnabled ? (
              <>
                <SignedOut>
                  <SignInButton mode="modal">
                    <button type="button" className={tw.btnPrimary}>
                      {">"} get-started
                    </button>
                  </SignInButton>
                </SignedOut>
                <SignedIn>
                  <Link href="/checks" className={tw.btnPrimary}>
                    {">"} open-dashboard
                  </Link>
                </SignedIn>
              </>
            ) : (
              <Link href="/checks" className={tw.btnPrimary}>
                {">"} open-checks
              </Link>
            )}
            <Link href="/docs" className={tw.btnSecondary}>
              {">"} read-the-docs
            </Link>
          </div>
        </div>
      </section>

      <Divider className="mx-auto max-w-5xl px-6 py-8" />

      {/* Features */}
      <section className="mx-auto max-w-5xl px-6 py-12">
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {FEATURES.map((f) => (
            <div key={f.flag} className={`${tw.cardHover} p-5`}>
              <span className="text-sm font-bold text-green-400">{f.flag}</span>
              <h3 className="mt-3 text-sm font-bold text-white">{f.title}</h3>
              <p className="mt-1.5 text-xs leading-relaxed text-slate-500">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      <Divider className="mx-auto max-w-5xl px-6 py-4" />

      {/* Integration Examples */}
      <section id="integration" className="mx-auto max-w-5xl px-6 py-12">
        <h2 className={`text-2xl md:text-4xl ${tw.heading}`} style={fonts.display}>
          Integrate in <span className={tw.accent}>seconds</span>
        </h2>
        <p className="mt-3 text-sm text-slate-500">
          Add a single line to your existing workflow. Here are some examples.
        </p>
        <div className="mt-8 grid gap-4 md:grid-cols-3">
          {CODE_EXAMPLES.map((ex) => (
            <div key={ex.label} className={tw.cardOverflow}>
              <div className="border-b border-slate-800 px-4 py-2">
                <span className="text-xs text-slate-600">{ex.label}</span>
              </div>
              <pre className="no-scrollbar overflow-x-auto p-4 text-xs leading-relaxed text-green-300/80">{ex.code}</pre>
            </div>
          ))}
        </div>
      </section>

      {/* Status legend */}
      <section className="mx-auto max-w-5xl px-6 py-12">
        <div className={`${tw.card} p-6`}>
          <h3 className={`text-lg ${tw.heading}`} style={fonts.display}>Status Model</h3>
          <div className="mt-4 grid gap-4 text-sm md:grid-cols-4">
            {STATUS_LEGEND.map((s) => (
              <div key={s.status} className={`rounded border ${statusStyles[s.key].border} ${statusStyles[s.key].text} bg-black/50 px-4 py-3`}>
                <span className="font-bold">{s.status}</span>
                <p className="mt-1 text-xs text-slate-500">{s.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="mx-auto max-w-5xl px-6 py-16">
        <div className="rounded border border-green-500/20 bg-green-500/[0.03] px-8 py-12 text-center">
          <p className="text-sm text-green-400/60">{">"} ready?</p>
          <h2 className={`mt-3 text-2xl sm:text-3xl md:text-5xl ${tw.heading}`} style={fonts.display}>
            Ship with <span className={tw.accent}>confidence</span>
          </h2>
          <p className="mx-auto mt-4 max-w-md text-sm text-slate-500">
            Free forever for small teams. No credit card. No vendor lock-in.
            Just reliable cron monitoring.
          </p>
          <Link href={isClerkUiEnabled ? "/sign-up" : "/checks"} className={`mt-8 inline-block ${tw.btnPrimary} px-8`}>
            {">"} sign-up-free
          </Link>
        </div>
      </section>

      <Footer links={[{ label: "docs", href: "/docs" }, { label: "github", href: "#" }, { label: "status", href: "#" }]} />
    </div>
  );
}
