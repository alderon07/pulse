import Link from "next/link";
import {
  SignedIn,
  SignedOut,
  SignInButton,
  UserButton,
} from "@clerk/nextjs";
import { notFound } from "next/navigation";

import {
  ApiError,
  getCheck,
  listAlertChannels,
  listCheckEvents,
} from "@/lib/api/client";
import type { AlertChannel, Check, Event } from "@/lib/api/schemas";
import { isClerkBackendEnabled, isClerkUiEnabled } from "@/lib/clerk-config";
import { requireBackendAccessToken } from "@/lib/server/backend-auth";
import { ScheduleModeFields } from "@/app/checks/components/schedule-mode-fields";

import {
  createAlertChannelAction,
  pauseCheckFromDetailAction,
  resumeCheckFromDetailAction,
  rotateTokenAction,
  toggleAlertChannelAction,
  updateScheduleAction,
} from "./actions";

export const dynamic = "force-dynamic";

const EVENT_LIMIT = 30;
const CHANNEL_LIMIT = 100;

type PageProps = {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ error?: string }>;
};

function formatDate(value: string | null | undefined) {
  if (!value) {
    return "—";
  }

  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(new Date(value));
}

function statusColor(status: Check["status"]) {
  switch (status) {
    case "up":
      return "text-green-400 border-green-500/30 bg-green-500/10";
    case "late":
      return "text-yellow-400 border-yellow-500/30 bg-yellow-500/10";
    case "down":
      return "text-red-400 border-red-500/30 bg-red-500/10";
    default:
      return "text-slate-400 border-slate-500/30 bg-slate-500/10";
  }
}

function formatOptionalNumber(value: number | null | undefined) {
  if (value === null || value === undefined) {
    return "—";
  }

  return value.toLocaleString("en-US");
}

function successText(value: boolean | null | undefined) {
  if (value === null || value === undefined) {
    return "—";
  }

  return value ? "true" : "false";
}

export default async function CheckDetailPage({ params, searchParams }: PageProps) {
  const { id } = await params;
  const query = await searchParams;

  let check: Check | null = null;
  let events: Event[] = [];
  let channels: AlertChannel[] = [];
  let loadingError = "";

  if (!isClerkBackendEnabled) {
    loadingError = "Clerk is not configured. Set NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY and CLERK_SECRET_KEY.";
  } else {
    try {
      const token = await requireBackendAccessToken();

      const [checkResult, eventsResult, channelsResult] = await Promise.allSettled([
        getCheck(token, id),
        listCheckEvents(token, id, EVENT_LIMIT),
        listAlertChannels(token, { limit: CHANNEL_LIMIT, offset: 0 }),
      ]);

      if (checkResult.status === "rejected") {
        if (checkResult.reason instanceof ApiError && checkResult.reason.status === 404) {
          notFound();
        }

        throw checkResult.reason;
      }

      check = checkResult.value;

      if (eventsResult.status === "fulfilled") {
        events = eventsResult.value.items;
      }

      if (channelsResult.status === "fulfilled") {
        channels = channelsResult.value.items;
      }
    } catch (error) {
      if (error instanceof ApiError) {
        loadingError = `${error.message} (status ${error.status})`;
      } else if (error instanceof Error) {
        loadingError = error.message;
      } else {
        loadingError = "Failed to load check details.";
      }
    }
  }

  if (!check) {
    const fallbackMessage = query.error ?? (loadingError || "Check not found");

    return (
      <main className="mx-auto flex min-h-screen w-full max-w-4xl flex-col gap-4 px-6 py-12 text-slate-200">
        <Link href="/checks" className="text-green-400 transition hover:text-green-300">{"<"} back to checks</Link>
        <div className="rounded border border-red-500/30 bg-red-500/5 px-4 py-3 text-sm text-red-400">
          <span className="font-bold">!</span> {fallbackMessage}
        </div>
      </main>
    );
  }

  return (
    <div className="min-h-screen bg-black text-slate-200">
      <div
        className="pointer-events-none fixed inset-0 z-20 opacity-[0.03]"
        style={{
          background:
            "repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(34,197,94,0.1) 2px, rgba(34,197,94,0.1) 4px)",
        }}
      />

      <nav className="relative z-10 mx-auto flex max-w-5xl items-center justify-between px-6 py-6">
        <Link href="/checks" className="text-green-400 transition hover:text-green-300">
          {"<"} back /checks
        </Link>
        {isClerkUiEnabled ? (
          <>
            <SignedIn>
              <UserButton />
            </SignedIn>
            <SignedOut>
              <SignInButton mode="modal">
                <button type="button" className="text-green-400 transition hover:text-green-300">
                  [login]
                </button>
              </SignInButton>
            </SignedOut>
          </>
        ) : null}
      </nav>

      <main className="relative z-10 mx-auto flex max-w-5xl flex-col gap-6 px-6 pb-12">
        <header className="space-y-3">
          <p className="text-sm text-slate-600">{">"} pulse / checks / {check.id}</p>
          <div className="flex flex-wrap items-center gap-3">
            <h1
              className="text-2xl font-extrabold uppercase tracking-tight text-white md:text-4xl"
              style={{ fontFamily: "var(--font-syne)" }}
            >
              {check.name}
            </h1>
            <span className={`inline-flex rounded border px-2 py-0.5 text-xs font-bold uppercase ${statusColor(check.status)}`}>
              {check.status}
            </span>
          </div>
        </header>

        {query.error && (
          <div className="rounded border border-red-500/30 bg-red-500/5 px-4 py-3 text-sm text-red-400">
            <span className="font-bold">!</span> {query.error}
          </div>
        )}
        {loadingError && (
          <div className="rounded border border-yellow-500/30 bg-yellow-500/5 px-4 py-3 text-sm text-yellow-300">
            <span className="font-bold">!</span> {loadingError}
          </div>
        )}

        <section className="grid gap-4 md:grid-cols-2">
          <article className="rounded border border-slate-800 bg-[#0a0a0a] p-5">
            <p className="text-xs uppercase tracking-wide text-slate-500">Configuration</p>
            <dl className="mt-4 space-y-2 text-sm">
              <div className="flex justify-between gap-3">
                <dt className="text-slate-500">Interval</dt>
                <dd className="text-slate-200">{check.expected_interval_seconds}s</dd>
              </div>
              <div className="flex justify-between gap-3">
                <dt className="text-slate-500">Grace</dt>
                <dd className="text-slate-200">{check.grace_seconds}s</dd>
              </div>
              <div className="flex justify-between gap-3">
                <dt className="text-slate-500">Schedule mode</dt>
                <dd className="text-slate-200">
                  {check.schedule_mode ?? "manual"}
                  {check.schedule_mode === "auto" && check.interval_sample_count !== undefined
                    ? ` (${check.interval_sample_count} samples)`
                    : ""}
                </dd>
              </div>
              <div className="flex justify-between gap-3">
                <dt className="text-slate-500">Next due</dt>
                <dd className="text-slate-200">{formatDate(check.next_due_at)}</dd>
              </div>
              <div className="flex justify-between gap-3">
                <dt className="text-slate-500">Last ping</dt>
                <dd className="text-slate-200">{formatDate(check.last_ping_at)}</dd>
              </div>
            </dl>
            <form action={updateScheduleAction} className="mt-4 grid gap-2 border-t border-slate-800 pt-4 sm:grid-cols-3">
              <input type="hidden" name="check_id" value={check.id} />
              <ScheduleModeFields
                defaultMode={(check.schedule_mode ?? "manual") as "manual" | "auto"}
                defaultIntervalSeconds={check.expected_interval_seconds}
                defaultGraceSeconds={check.grace_seconds}
                labelClassName="block text-xs text-slate-500"
              />
              <div className="sm:col-span-3">
                <button
                  type="submit"
                  className="rounded border border-cyan-500/30 px-3 py-1.5 text-xs font-medium text-cyan-300 transition hover:bg-cyan-500/10"
                >
                  [save-timing]
                </button>
              </div>
            </form>
          </article>

          <article className="rounded border border-slate-800 bg-[#0a0a0a] p-5">
            <p className="text-xs uppercase tracking-wide text-slate-500">Token</p>
            <p className="mt-3 break-all rounded border border-slate-800 bg-black px-3 py-2 text-xs text-green-300">
              {check.token}
            </p>
            <div className="mt-4 flex flex-wrap gap-2">
              {check.status === "paused" ? (
                <form action={resumeCheckFromDetailAction}>
                  <input type="hidden" name="check_id" value={check.id} />
                  <button
                    type="submit"
                    className="rounded border border-green-500/30 px-3 py-1.5 text-xs font-medium text-green-400 transition hover:bg-green-500/10"
                  >
                    [resume]
                  </button>
                </form>
              ) : (
                <form action={pauseCheckFromDetailAction}>
                  <input type="hidden" name="check_id" value={check.id} />
                  <button
                    type="submit"
                    className="rounded border border-yellow-500/30 px-3 py-1.5 text-xs font-medium text-yellow-400 transition hover:bg-yellow-500/10"
                  >
                    [pause]
                  </button>
                </form>
              )}
              <form action={rotateTokenAction}>
                <input type="hidden" name="check_id" value={check.id} />
                <button
                  type="submit"
                  className="rounded border border-sky-500/30 px-3 py-1.5 text-xs font-medium text-sky-400 transition hover:bg-sky-500/10"
                >
                  [rotate-token]
                </button>
              </form>
            </div>
          </article>
        </section>

        <section className="overflow-hidden rounded border border-slate-800 bg-[#0a0a0a]">
          <div className="border-b border-slate-800 px-4 py-2.5 text-xs text-slate-600">events (latest {EVENT_LIMIT})</div>
          <div className="overflow-x-auto">
            <table className="min-w-full border-collapse text-sm">
              <thead>
                <tr className="border-b border-slate-800 text-left text-xs uppercase tracking-wider text-slate-600">
                  <th className="px-4 py-3 font-medium">Time</th>
                  <th className="px-4 py-3 font-medium">Type</th>
                  <th className="px-4 py-3 font-medium">Duration (ms)</th>
                  <th className="px-4 py-3 font-medium">Output</th>
                  <th className="px-4 py-3 font-medium">Success</th>
                </tr>
              </thead>
              <tbody>
                {events.map((event) => (
                  <tr key={event.id} className="border-b border-slate-800/50">
                    <td className="px-4 py-2.5 text-slate-300">{formatDate(event.received_at)}</td>
                    <td className="px-4 py-2.5 text-slate-300">{event.type}</td>
                    <td className="px-4 py-2.5 text-slate-400">{formatOptionalNumber(event.duration_ms)}</td>
                    <td className="px-4 py-2.5 text-slate-400">{formatOptionalNumber(event.output_size)}</td>
                    <td className="px-4 py-2.5 text-slate-400">{successText(event.success)}</td>
                  </tr>
                ))}
                {events.length === 0 && (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-slate-600">
                      No events yet for this check.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <article className="overflow-hidden rounded border border-slate-800 bg-[#0a0a0a] lg:col-span-2">
            <div className="border-b border-slate-800 px-4 py-2.5 text-xs text-slate-600">alert channels</div>
            <div className="overflow-x-auto">
              <table className="min-w-full border-collapse text-sm">
                <thead>
                  <tr className="border-b border-slate-800 text-left text-xs uppercase tracking-wider text-slate-600">
                    <th className="px-4 py-3 font-medium">Type</th>
                    <th className="px-4 py-3 font-medium">Target</th>
                    <th className="px-4 py-3 font-medium">Enabled</th>
                    <th className="px-4 py-3 font-medium">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {channels.map((channel) => (
                    <tr key={channel.id} className="border-b border-slate-800/50">
                      <td className="px-4 py-2.5 text-slate-300">{channel.type}</td>
                      <td className="px-4 py-2.5 text-slate-400">{channel.target}</td>
                      <td className="px-4 py-2.5 text-slate-300">{channel.enabled ? "yes" : "no"}</td>
                      <td className="px-4 py-2.5">
                        <form action={toggleAlertChannelAction}>
                          <input type="hidden" name="check_id" value={check.id} />
                          <input type="hidden" name="channel_id" value={channel.id} />
                          <input type="hidden" name="enabled" value={channel.enabled ? "false" : "true"} />
                          <button
                            type="submit"
                            className="rounded border border-slate-700 px-2.5 py-1 text-xs text-slate-300 transition hover:border-green-500/30 hover:text-green-300"
                          >
                            {channel.enabled ? "[disable]" : "[enable]"}
                          </button>
                        </form>
                      </td>
                    </tr>
                  ))}
                  {channels.length === 0 && (
                    <tr>
                      <td colSpan={4} className="px-4 py-8 text-center text-slate-600">
                        No alert channels configured.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </article>

          <article className="rounded border border-slate-800 bg-[#0a0a0a] p-4">
            <p className="text-xs uppercase tracking-wide text-slate-500">add channel</p>
            <form action={createAlertChannelAction} className="mt-3 space-y-3">
              <input type="hidden" name="check_id" value={check.id} />
              <label className="block text-xs text-slate-500">
                type
                <select
                  name="channel_type"
                  className="mt-1.5 w-full rounded border border-slate-800 bg-black px-3 py-2 text-sm text-slate-200 outline-none transition focus:border-green-500/50"
                  defaultValue="email"
                >
                  <option value="email">email</option>
                  <option value="webhook">webhook</option>
                </select>
              </label>
              <label className="block text-xs text-slate-500">
                target
                <input
                  name="target"
                  required
                  placeholder="ops@example.com or https://..."
                  className="mt-1.5 w-full rounded border border-slate-800 bg-black px-3 py-2 text-sm text-slate-200 outline-none transition focus:border-green-500/50"
                />
              </label>
              <button
                type="submit"
                className="w-full rounded border border-green-500/30 bg-green-500/10 px-3 py-2 text-xs font-medium text-green-400 transition hover:bg-green-500/20"
              >
                [create-channel]
              </button>
            </form>
          </article>
        </section>
      </main>
    </div>
  );
}
