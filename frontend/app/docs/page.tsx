import Link from "next/link";
import type { Metadata } from "next";
import { tw, fonts, statusStyles } from "@/lib/theme";
import {
  ScanLines,
  TerminalBlock,
  SectionHeading,
  Divider,
  Footer,
  PulseLogo,
} from "@/app/components/ui";

export const metadata: Metadata = {
  title: "Pulse — Docs",
  description: "API documentation for Pulse cron monitoring.",
};

const PING_PARAMS = [
  { name: "duration_ms", type: "int", desc: "How long the job took (ms). Max 86,400,000." },
  { name: "output_size", type: "int", desc: "Size of job output in bytes. Max 1,000,000,000." },
  { name: "success", type: "bool", desc: "Whether the job succeeded. true or false." },
  { name: "state", type: "string", desc: "Job state: ok, run, fail, start, complete, pass, error." },
  { name: "msg", type: "string", desc: "Human-readable message (max 1024 chars)." },
  { name: "env", type: "string", desc: "Environment label, e.g. production (max 64 chars)." },
  { name: "metric", type: "string", desc: "Arbitrary metric string (max 256 chars)." },
];

const STATUS_MODEL = [
  { status: "UP", key: "up" as const, desc: "Ping arrived before next_due_at." },
  { status: "LATE", key: "late" as const, desc: "Past due but still within grace_seconds." },
  { status: "DOWN", key: "down" as const, desc: "Past due + grace period. Incident opened, alerts fire." },
  { status: "PAUSED", key: "paused" as const, desc: "Monitoring suspended. No transitions, no alerts." },
];

function EndpointRow({ method, path, desc, auth }: { method: string; path: string; desc: string; auth?: boolean }) {
  const methodColor =
    method === "GET" ? "text-green-400" :
    method === "POST" ? "text-yellow-400" :
    method === "PATCH" ? "text-blue-400" :
    "text-slate-400";

  return (
    <tr className={tw.tableRow}>
      <td className="px-4 py-2.5"><span className={`font-bold ${methodColor}`}>{method}</span></td>
      <td className="px-4 py-2.5 text-green-300/80">{path}</td>
      <td className="px-4 py-2.5 text-slate-400">{desc}</td>
      <td className="px-4 py-2.5 text-center">
        {auth ? <span className="text-yellow-400">●</span> : <span className="text-slate-700">—</span>}
      </td>
    </tr>
  );
}

const TOC = [
  { id: "quickstart", label: "Quickstart" },
  { id: "ping", label: "Ping Endpoint" },
  { id: "ping-params", label: "Ping Parameters" },
  { id: "status-model", label: "Status Model" },
  { id: "auth", label: "Authentication" },
  { id: "api-reference", label: "API Reference" },
  { id: "checks-api", label: "Checks API" },
  { id: "alert-channels", label: "Alert Channels" },
  { id: "integrations", label: "Integrations" },
  { id: "idempotency", label: "Idempotency" },
  { id: "rate-limits", label: "Rate Limits" },
  { id: "errors", label: "Error Format" },
];

export default function DocsPage() {
  return (
    <div className={tw.page}>
      <ScanLines />

      {/* Navigation */}
      <nav className="relative z-10 mx-auto flex max-w-6xl items-center justify-between px-6 py-6">
        <PulseLogo asLink />
        <div className={tw.navLinks}>
          <span className={tw.navActive}>[docs]</span>
          <a href="#" className={tw.navLink}>[github]</a>
          <Link href="/checks" className={tw.navLink}>[checks]</Link>
        </div>
      </nav>

      <div className="relative z-10 mx-auto flex max-w-6xl gap-10 px-6 pb-16">
        {/* Sidebar TOC */}
        <aside className="hidden w-52 shrink-0 lg:block">
          <div className="sticky top-8">
            <p className="text-xs font-bold uppercase tracking-wider text-slate-600">On this page</p>
            <nav className="mt-3 flex flex-col gap-1.5">
              {TOC.map((item) => (
                <a key={item.id} href={`#${item.id}`} className="text-xs text-slate-400 transition hover:text-green-400">
                  {item.label}
                </a>
              ))}
            </nav>
          </div>
        </aside>

        {/* Main content */}
        <main className="min-w-0 flex-1">
          <header>
            <p className={tw.breadcrumb}>{">"} pulse / docs</p>
            <h1 className={`mt-2 text-3xl md:text-5xl ${tw.heading}`} style={fonts.display}>
              Docu<span className={tw.accent}>mentation</span>
            </h1>
            <p className={`mt-4 max-w-2xl ${tw.body}`}>
              Everything you need to integrate Pulse into your infrastructure.
              Monitor cron jobs, scheduled tasks, and background workers with a single HTTP call.
            </p>
          </header>

          <Divider className="mt-6" />

          {/* ─── Quickstart ─── */}
          <section className="mt-10 space-y-5">
            <SectionHeading id="quickstart">Quickstart</SectionHeading>
            <p className={tw.body}>Three steps to start monitoring your first cron job.</p>

            <div className="space-y-4">
              <div>
                <p className="text-sm font-bold text-slate-400">
                  <span className={tw.accent}>01</span> — Create a check
                </p>
                <TerminalBlock label="create check">
{`curl -X POST https://api.pulse.dev/api/v1/checks \\
  -H "Authorization: Bearer \$TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "backup-daily",
    "expected_interval_seconds": 86400,
    "grace_seconds": 3600
  }'`}
                </TerminalBlock>
                <TerminalBlock label="response">
{`{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "backup-daily",
  "token": "tok_a1b2c3d4e5f6...",
  "expected_interval_seconds": 86400,
  "grace_seconds": 3600,
  "status": "up",
  "next_due_at": "2026-03-04T00:00:00Z"
}`}
                </TerminalBlock>
              </div>

              <div>
                <p className="text-sm font-bold text-slate-400">
                  <span className={tw.accent}>02</span> — Send a ping after your job runs
                </p>
                <TerminalBlock label="ping">
{`# Append to your cron job
/opt/bin/backup.sh && curl -fsS https://api.pulse.dev/v1/ping/tok_a1b2c3d4e5f6`}
                </TerminalBlock>
              </div>

              <div>
                <p className="text-sm font-bold text-slate-400">
                  <span className={tw.accent}>03</span> — Get alerted when something breaks
                </p>
                <p className={tw.body}>
                  If a ping doesn&apos;t arrive within the expected interval + grace period,
                  Pulse marks the check as <span className="text-red-400 font-bold">DOWN</span>,
                  opens an incident, and sends alerts to your configured channels.
                </p>
              </div>
            </div>
          </section>

          <Divider className="my-10" />

          {/* ─── Ping Endpoint ─── */}
          <section className="space-y-5">
            <SectionHeading id="ping">Ping <span className={tw.accent}>Endpoint</span></SectionHeading>
            <p className={tw.body}>
              The heartbeat endpoint. Call it after your job completes to report success.
              No authentication required — the check token in the URL is your credential.
            </p>
            <div className={`${tw.cardOverflow} p-5`}>
              <p className="text-sm">
                <span className="font-bold text-green-400">GET</span>{" "}
                <span className="text-yellow-400">|</span>{" "}
                <span className="font-bold text-yellow-400">POST</span>{" "}
                <span className="text-slate-400">/v1/ping/</span>
                <span className="text-green-300">{"{token}"}</span>
              </p>
            </div>
            <TerminalBlock label="simple GET ping">
{`curl https://api.pulse.dev/v1/ping/tok_a1b2c3d4e5f6`}
            </TerminalBlock>
            <TerminalBlock label="POST with metadata">
{`curl -X POST https://api.pulse.dev/v1/ping/tok_a1b2c3d4e5f6 \\
  -H "Content-Type: application/json" \\
  -d '{
    "duration_ms": 4230,
    "success": true,
    "msg": "Backed up 142 tables"
  }'`}
            </TerminalBlock>
            <TerminalBlock label="response">
{`{
  "ok": true,
  "status": "up",
  "next_due_at": "2026-03-04T00:00:00Z",
  "idempotent": false
}`}
            </TerminalBlock>

            <div className={`${tw.card} p-4`}>
              <p className="text-sm text-slate-400">
                <span className="font-bold text-yellow-400">!</span>{" "}
                The deprecated endpoint <span className="text-slate-300">/ping/{"{token}"}</span> still works
                but returns <span className="text-slate-300">Sunset: Fri, 26 Jun 2026</span> headers. Migrate
                to <span className="text-green-300">/v1/ping/{"{token}"}</span>.
              </p>
            </div>
          </section>

          <Divider className="my-10" />

          {/* ─── Ping Parameters ─── */}
          <section className="space-y-5">
            <SectionHeading id="ping-params">Ping <span className={tw.accent}>Parameters</span></SectionHeading>
            <p className={tw.body}>
              All parameters are optional. They can be sent as query params, headers
              (<span className="text-white">X-Duration-MS</span>, <span className="text-white">X-Success</span>, etc.),
              or in a JSON POST body. Priority: body &gt; headers &gt; query.
            </p>
            <div className={`overflow-x-auto ${tw.card}`}>
              <table className="min-w-full text-xs">
                <thead>
                  <tr className={tw.tableHeader}>
                    <th className="px-4 py-2.5 font-medium">Param</th>
                    <th className="px-4 py-2.5 font-medium">Type</th>
                    <th className="px-4 py-2.5 font-medium">Description</th>
                  </tr>
                </thead>
                <tbody>
                  {PING_PARAMS.map((p) => (
                    <tr key={p.name} className="border-b border-slate-800/50">
                      <td className="px-4 py-2.5 text-green-300">{p.name}</td>
                      <td className="px-4 py-2.5 text-yellow-400/70">{p.type}</td>
                      <td className="px-4 py-2.5 text-slate-400">{p.desc}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className={tw.muted}>
              Max body size: 16 KB. State values: <span className="text-green-300">ok</span> |{" "}
              <span className="text-green-300">run</span> |{" "}
              <span className="text-green-300">fail</span> |{" "}
              <span className="text-green-300">start</span> |{" "}
              <span className="text-green-300">complete</span> |{" "}
              <span className="text-green-300">pass</span> |{" "}
              <span className="text-green-300">error</span>
            </p>
          </section>

          <Divider className="my-10" />

          {/* ─── Status Model ─── */}
          <section className="space-y-5">
            <SectionHeading id="status-model">Status <span className={tw.accent}>Model</span></SectionHeading>
            <p className={tw.body}>Every check has one of four statuses. The worker evaluates transitions every tick.</p>
            <div className="grid gap-3 md:grid-cols-2">
              {STATUS_MODEL.map((s) => {
                const sc = statusStyles[s.key];
                return (
                  <div key={s.status} className={`${tw.card} border ${sc.border} px-4 py-3`}>
                    <span className={`font-bold ${sc.text}`}>{s.status}</span>
                    <p className="mt-1 text-sm text-slate-400">{s.desc}</p>
                  </div>
                );
              })}
            </div>
            <TerminalBlock label="status timeline">
{`time ──────────────────────────────────────────────▶

  ping          next_due_at    grace deadline
   │                │                │
   ▼                ▼                ▼
   ├── UP ──────────┤── LATE ────────┤── DOWN ──▶
                    │                │
              (still healthy)  (incident opens,
                                alerts fire)`}
            </TerminalBlock>
          </section>

          <Divider className="my-10" />

          {/* ─── Authentication ─── */}
          <section className="space-y-5">
            <SectionHeading id="auth">Authen<span className={tw.accent}>tication</span></SectionHeading>
            <p className={tw.body}>
              Management API endpoints (<span className="text-white">/api/v1/*</span>) require a JWT
              bearer token. Pulse supports two signing methods:
            </p>
            <div className="grid gap-3 md:grid-cols-2">
              <div className={`${tw.card} p-4`}>
                <p className="text-sm font-bold text-green-400">HS256</p>
                <p className="mt-2 text-sm text-slate-400">
                  Symmetric. Set <span className="text-white">JWT_HS256_SECRET</span> env var.
                  Sign your own tokens with the shared secret.
                </p>
              </div>
              <div className={`${tw.card} p-4`}>
                <p className="text-sm font-bold text-green-400">RS256 / JWKS</p>
                <p className="mt-2 text-sm text-slate-400">
                  Asymmetric. Set <span className="text-white">JWT_JWKS_URL</span> to your OIDC provider
                  (e.g. Clerk). Tokens are verified against the JWKS endpoint.
                </p>
              </div>
            </div>
            <TerminalBlock label="authenticated request">
{`curl https://api.pulse.dev/api/v1/checks \\
  -H "Authorization: Bearer eyJhbGciOiJIUzI1N..."`}
            </TerminalBlock>
            <p className={tw.muted}>
              JWT claims: <span className="text-slate-300">sub</span> (user ID, UUID or string) and{" "}
              <span className="text-slate-300">email</span>. Non-UUID subjects are mapped to a stable internal UUID.
            </p>
          </section>

          <Divider className="my-10" />

          {/* ─── API Reference ─── */}
          <section className="space-y-5">
            <SectionHeading id="api-reference">API <span className={tw.accent}>Reference</span></SectionHeading>
            <p className={tw.body}>All endpoints at a glance. Yellow dot = requires JWT auth.</p>
            <div className={`overflow-x-auto ${tw.card}`}>
              <table className="min-w-full text-xs">
                <thead>
                  <tr className={tw.tableHeader}>
                    <th className="px-4 py-2.5 font-medium">Method</th>
                    <th className="px-4 py-2.5 font-medium">Path</th>
                    <th className="px-4 py-2.5 font-medium">Description</th>
                    <th className="px-4 py-2.5 text-center font-medium">Auth</th>
                  </tr>
                </thead>
                <tbody>
                  <EndpointRow method="GET" path="/healthz" desc="Health check" />
                  <EndpointRow method="GET" path="/readyz" desc="Readiness probe" />
                  <EndpointRow method="GET" path="/v1/ping/{token}" desc="Record heartbeat (GET)" />
                  <EndpointRow method="POST" path="/v1/ping/{token}" desc="Record heartbeat with body" />
                  <EndpointRow method="GET" path="/api/v1/checks" desc="List checks" auth />
                  <EndpointRow method="POST" path="/api/v1/checks" desc="Create check" auth />
                  <EndpointRow method="GET" path="/api/v1/checks/{id}" desc="Get check" auth />
                  <EndpointRow method="PATCH" path="/api/v1/checks/{id}" desc="Update check" auth />
                  <EndpointRow method="POST" path="/api/v1/checks/{id}/pause" desc="Pause monitoring" auth />
                  <EndpointRow method="POST" path="/api/v1/checks/{id}/resume" desc="Resume monitoring" auth />
                  <EndpointRow method="POST" path="/api/v1/checks/{id}/rotate-token" desc="Rotate ping token" auth />
                  <EndpointRow method="GET" path="/api/v1/checks/{id}/events" desc="List ping events" auth />
                  <EndpointRow method="GET" path="/api/v1/alert-channels" desc="List alert channels" auth />
                  <EndpointRow method="POST" path="/api/v1/alert-channels" desc="Create alert channel" auth />
                  <EndpointRow method="PATCH" path="/api/v1/alert-channels/{id}" desc="Update alert channel" auth />
                </tbody>
              </table>
            </div>
          </section>

          <Divider className="my-10" />

          {/* ─── Checks API ─── */}
          <section className="space-y-5">
            <SectionHeading id="checks-api">Checks <span className={tw.accent}>API</span></SectionHeading>
            <div className="space-y-6">
              {[
                { label: "Create a check", terminal: "POST /api/v1/checks", code: `curl -X POST https://api.pulse.dev/api/v1/checks \\\\\n  -H "Authorization: Bearer \\$TOKEN" \\\\\n  -H "Content-Type: application/json" \\\\\n  -d '{\n    "name": "db-backup",\n    "expected_interval_seconds": 3600,\n    "grace_seconds": 300\n  }'\n\n# name                        required, string\n# expected_interval_seconds   required, int > 0\n# grace_seconds               required, int >= 0` },
                { label: "List checks", terminal: "GET /api/v1/checks", code: `curl "https://api.pulse.dev/api/v1/checks?limit=50&offset=0" \\\\\n  -H "Authorization: Bearer \\$TOKEN"\n\n# Response: { "items": [ ...checks ] }\n# limit:  1-200 (default 50)\n# offset: pagination offset` },
                { label: "Update a check", terminal: "PATCH /api/v1/checks/{id}", code: `curl -X PATCH https://api.pulse.dev/api/v1/checks/UUID \\\\\n  -H "Authorization: Bearer \\$TOKEN" \\\\\n  -H "Content-Type: application/json" \\\\\n  -d '{\n    "name": "new-name",\n    "expected_interval_seconds": 7200,\n    "grace_seconds": 600\n  }'\n\n# All fields optional. At least one required.` },
                { label: "Pause / Resume", terminal: "POST /api/v1/checks/{id}/pause | resume", code: `# Pause monitoring\ncurl -X POST https://api.pulse.dev/api/v1/checks/UUID/pause \\\\\n  -H "Authorization: Bearer \\$TOKEN"\n\n# Resume monitoring\ncurl -X POST https://api.pulse.dev/api/v1/checks/UUID/resume \\\\\n  -H "Authorization: Bearer \\$TOKEN"` },
                { label: "Rotate ping token", terminal: "POST /api/v1/checks/{id}/rotate-token", code: `curl -X POST https://api.pulse.dev/api/v1/checks/UUID/rotate-token \\\\\n  -H "Authorization: Bearer \\$TOKEN"\n\n# Returns updated check with new token.\n# Old token is immediately invalidated.` },
                { label: "List events", terminal: "GET /api/v1/checks/{id}/events", code: `curl "https://api.pulse.dev/api/v1/checks/UUID/events?limit=50" \\\\\n  -H "Authorization: Bearer \\$TOKEN"\n\n# Response: { "items": [ ...events ] }\n# Each event: type, duration_ms, output_size, success, created_at` },
              ].map((item) => (
                <div key={item.label}>
                  <p className="text-sm font-bold uppercase tracking-wider text-slate-500">{item.label}</p>
                  <TerminalBlock label={item.terminal}>{item.code}</TerminalBlock>
                </div>
              ))}
            </div>
          </section>

          <Divider className="my-10" />

          {/* ─── Alert Channels ─── */}
          <section className="space-y-5">
            <SectionHeading id="alert-channels">Alert <span className={tw.accent}>Channels</span></SectionHeading>
            <p className={tw.body}>
              Configure where alerts are sent when checks go down or recover.
              Two channel types: <span className="text-green-300">email</span> and{" "}
              <span className="text-green-300">webhook</span>.
            </p>
            <TerminalBlock label="create email channel">
{`curl -X POST https://api.pulse.dev/api/v1/alert-channels \\
  -H "Authorization: Bearer \$TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{ "type": "email", "target": "oncall@example.com" }'`}
            </TerminalBlock>
            <TerminalBlock label="create webhook channel">
{`curl -X POST https://api.pulse.dev/api/v1/alert-channels \\
  -H "Authorization: Bearer \$TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{ "type": "webhook", "target": "https://hooks.slack.com/..." }'`}
            </TerminalBlock>
            <TerminalBlock label="update channel">
{`curl -X PATCH https://api.pulse.dev/api/v1/alert-channels/UUID \\
  -H "Authorization: Bearer \$TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{ "enabled": false }'

# enabled:  bool   — toggle alerting
# target:   string — update destination`}
            </TerminalBlock>
            <div className={`${tw.card} p-5 text-sm leading-relaxed text-slate-400`}>
              <p><span className="font-bold text-green-400">Webhook security:</span> Outgoing webhook requests include an HMAC signature header for verification. Only HTTPS targets are allowed.</p>
              <p className="mt-3"><span className="font-bold text-green-400">Alert types:</span> <span className="text-red-400">down</span> (check went down), <span className="text-yellow-400">repeat</span> (still down), <span className="text-green-400">recovered</span> (back up), <span className="text-yellow-400">warning</span> (approaching grace).</p>
            </div>
          </section>

          <Divider className="my-10" />

          {/* ─── Integrations ─── */}
          <section className="space-y-5">
            <SectionHeading id="integrations">Inte<span className={tw.accent}>grations</span></SectionHeading>
            <div className="grid gap-4 md:grid-cols-2">
              <TerminalBlock label="crontab">{`# Append ping to any cron job\n*/5 * * * * /opt/bin/backup.sh \\\\\n  && curl -fsS https://api.pulse.dev/v1/ping/TOKEN`}</TerminalBlock>
              <TerminalBlock label="docker healthcheck">{`HEALTHCHECK --interval=60s --timeout=5s \\\\\n  CMD curl -f https://api.pulse.dev/v1/ping/TOKEN \\\\\n  || exit 1`}</TerminalBlock>
              <TerminalBlock label="systemd timer">{`[Service]\nExecStartPost=/usr/bin/curl -fsS \\\\\n  https://api.pulse.dev/v1/ping/TOKEN`}</TerminalBlock>
              <TerminalBlock label="bash with timing">{`#!/bin/bash\nstart=$(date +%s%N)\n/opt/bin/etl-job.sh\nend=$(date +%s%N)\nms=$(( (end - start) / 1000000 ))\n\ncurl -fsS "https://api.pulse.dev/v1/ping/TOKEN\\\\\n?duration_ms=$ms&success=true"`}</TerminalBlock>
              <TerminalBlock label="python">{`import requests, time\n\nstart = time.time()\nrun_job()\nelapsed = int((time.time() - start) * 1000)\n\nrequests.post(\n    "https://api.pulse.dev/v1/ping/TOKEN",\n    json={"duration_ms": elapsed, "success": True}\n)`}</TerminalBlock>
              <TerminalBlock label="node.js">{`const start = Date.now();\nawait runJob();\nconst ms = Date.now() - start;\n\nawait fetch(\n  "https://api.pulse.dev/v1/ping/TOKEN",\n  {\n    method: "POST",\n    headers: {"Content-Type": "application/json"},\n    body: JSON.stringify({\n      duration_ms: ms, success: true\n    })\n  }\n);`}</TerminalBlock>
            </div>
          </section>

          <Divider className="my-10" />

          {/* ─── Idempotency ─── */}
          <section className="space-y-5">
            <SectionHeading id="idempotency">Idemp<span className={tw.accent}>otency</span></SectionHeading>
            <p className={tw.body}>
              Prevent duplicate pings from retries or network issues.
              Pass the <span className="text-white">Idempotency-Key</span> header with a unique value per logical ping.
            </p>
            <TerminalBlock label="idempotent ping">
{`curl https://api.pulse.dev/v1/ping/TOKEN \\
  -H "Idempotency-Key: backup-2026-03-02-run-1"

# First call:  processes normally, idempotent: false
# Repeat call: returns cached result, idempotent: true`}
            </TerminalBlock>
            <p className={tw.muted}>
              Keys are scoped per check token and expire after a configurable TTL (server-side).
              Max key length: 128 characters.
            </p>
          </section>

          <Divider className="my-10" />

          {/* ─── Rate Limits ─── */}
          <section className="space-y-5">
            <SectionHeading id="rate-limits">Rate <span className={tw.accent}>Limits</span></SectionHeading>
            <p className={tw.body}>Ping endpoints are rate-limited per token + IP. Invalid tokens have a stricter limit.</p>
            <div className={`overflow-x-auto ${tw.card}`}>
              <table className="min-w-full text-xs">
                <thead>
                  <tr className={tw.tableHeader}>
                    <th className="px-4 py-2.5 font-medium">Scope</th>
                    <th className="px-4 py-2.5 font-medium">Limit</th>
                    <th className="px-4 py-2.5 font-medium">Notes</th>
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-b border-slate-800/50">
                    <td className="px-4 py-2.5 text-green-300">Valid ping</td>
                    <td className="px-4 py-2.5 text-slate-300">Configurable per-minute + burst</td>
                    <td className="px-4 py-2.5 text-slate-400">Keyed on token + client IP</td>
                  </tr>
                  <tr className="border-b border-slate-800/50">
                    <td className="px-4 py-2.5 text-red-400">Invalid token</td>
                    <td className="px-4 py-2.5 text-slate-300">Stricter limit</td>
                    <td className="px-4 py-2.5 text-slate-400">Keyed on client IP</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p className={tw.muted}>
              Exceeding the limit returns <span className="text-red-400">429 Too Many Requests</span>.
            </p>
          </section>

          <Divider className="my-10" />

          {/* ─── Error Format ─── */}
          <section className="space-y-5">
            <SectionHeading id="errors">Error <span className={tw.accent}>Format</span></SectionHeading>
            <p className={tw.body}>All errors return JSON with a consistent structure.</p>
            <TerminalBlock label="error response">
{`{
  "code": "Not Found",
  "message": "check not found"
}

# Common codes:
# 400 Bad Request      — invalid input
# 401 Unauthorized     — missing or invalid JWT
# 404 Not Found        — resource not found
# 429 Too Many Requests — rate limited
# 500 Internal Error   — server-side failure`}
            </TerminalBlock>
            <p className={tw.muted}>
              All responses set <span className="text-slate-300">Content-Type: application/json</span>.
              Ping endpoints also set <span className="text-slate-300">Cache-Control: no-store</span>.
            </p>
          </section>
        </main>
      </div>

      <Footer
        maxWidth="max-w-6xl"
        links={[{ label: "home", href: "/" }, { label: "checks", href: "/checks" }, { label: "github", href: "#" }]}
      />
    </div>
  );
}
