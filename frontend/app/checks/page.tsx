import Link from "next/link";
import {
  SignedIn,
  SignedOut,
  SignInButton,
  SignOutButton,
  UserButton,
} from "@clerk/nextjs";

import { ApiError, listChecks } from "@/lib/api/client";
import type { Check } from "@/lib/api/schemas";
import { isClerkBackendEnabled, isClerkUiEnabled } from "@/lib/clerk-config";
import { requireBackendAccessToken } from "@/lib/server/backend-auth";
import { tw, fonts, inlineStyles, statusStyles } from "@/lib/theme";
import {
  ScanLines,
  WindowChrome,
  PageHeading,
  ErrorBanner,
  Footer,
  PulseLogo,
} from "@/app/components/ui";

import { pauseCheckAction, resumeCheckAction } from "./actions";

export const dynamic = "force-dynamic";

const DEFAULT_LIMIT = 25;
const MAX_LIMIT = 200;

type PageSearchParams = {
  error?: string;
  limit?: string;
  offset?: string;
};

type PageProps = {
  searchParams: Promise<PageSearchParams>;
};

function parseIntParam(rawValue: string | undefined, fallback: number, max: number) {
  if (!rawValue) return fallback;
  const parsed = Number.parseInt(rawValue, 10);
  if (Number.isNaN(parsed)) return fallback;
  return Math.max(0, Math.min(parsed, max));
}

function formatDate(value: string | null | undefined) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function checksHref(limit: number, offset: number) {
  return `/checks?${new URLSearchParams({ limit: String(limit), offset: String(offset) })}`;
}

export default async function ChecksPage({ searchParams }: PageProps) {
  const params = await searchParams;
  const limit = parseIntParam(params.limit, DEFAULT_LIMIT, MAX_LIMIT);
  const offset = parseIntParam(params.offset, 0, 10_000);

  let checks: Check[] = [];
  let loadingError = "";

  if (!isClerkBackendEnabled) {
    loadingError = "Clerk is not configured. Set NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY and CLERK_SECRET_KEY.";
  } else {
    try {
      const token = await requireBackendAccessToken();
      const response = await listChecks(token, { limit, offset });
      checks = response.items;
    } catch (error) {
      if (error instanceof ApiError) {
        loadingError = `${error.message} (status ${error.status})`;
      } else if (error instanceof Error) {
        loadingError = error.message;
      } else {
        loadingError = "Unable to load checks.";
      }
    }
  }

  const prevOffset = Math.max(0, offset - limit);
  const nextOffset = offset + limit;

  return (
    <div className={tw.page}>
      <ScanLines />

      {/* Navigation */}
      <nav className={tw.navWrapper}>
        <PulseLogo asLink />
        <div className={tw.navLinks}>
          <Link href="/docs" className={tw.navLink}>[docs]</Link>
          <a href="#" className={tw.navLink}>[github]</a>
          <span className={tw.navActive}>[checks]</span>
          {isClerkUiEnabled ? (
            <>
              <SignedIn>
                <UserButton />
              </SignedIn>
              <SignedOut>
                <SignInButton mode="modal">
                  <button type="button" className={`${tw.navActive} transition hover:text-green-300`}>
                    [login]
                  </button>
                </SignInButton>
              </SignedOut>
            </>
          ) : null}
        </div>
      </nav>

      <main className="relative z-10 mx-auto flex max-w-5xl flex-col gap-6 px-6 pb-12">
        <PageHeading breadcrumb="pulse / checks">
          Checks <span className={tw.accent}>Dashboard</span>
        </PageHeading>

        {/* Auth section */}
        <section className={tw.cardOverflow}>
          <WindowChrome label="auth" />
          <div className="p-5">
            <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
              <p className="text-slate-400">
                <span className="text-green-500">$ </span>
                {isClerkBackendEnabled
                  ? "Authenticated with Clerk. Backend API calls use Clerk JWTs server-side."
                  : "Clerk is not configured in this environment."}
              </p>
              {isClerkBackendEnabled ? (
                <SignedIn>
                  <SignOutButton>
                    <button type="button" className={tw.btnDanger}>
                      [sign-out]
                    </button>
                  </SignOutButton>
                </SignedIn>
              ) : null}
            </div>
          </div>
        </section>

        {params.error && <ErrorBanner message={params.error} />}
        {loadingError && <ErrorBanner message={loadingError} />}

        {/* Checks table */}
        <section className={tw.cardOverflow}>
          <WindowChrome label={`checks — ${checks.length} results`} />

          {/* Pagination bar */}
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 px-5 py-3">
            <p className="text-xs text-slate-500">
              <span className="text-green-500">$</span> showing {checks.length} checks
              {offset > 0 && <span className="text-slate-600"> (offset {offset})</span>}
            </p>
            <div className="flex items-center gap-2 text-xs">
              {offset > 0 ? (
                <Link href={checksHref(limit, prevOffset)} className={tw.paginationLink}>
                  [prev]
                </Link>
              ) : (
                <span className={tw.paginationDisabled}>[prev]</span>
              )}
              <Link href={checksHref(limit, nextOffset)} className={tw.paginationLink}>
                [next]
              </Link>
            </div>
          </div>

          {/* Table */}
          <div className="overflow-x-auto">
            <table className="min-w-full border-collapse text-sm">
              <thead>
                <tr className={tw.tableHeader}>
                  <th className="px-5 py-3 font-medium">Name</th>
                  <th className="px-5 py-3 font-medium">Status</th>
                  <th className="px-5 py-3 font-medium">Interval</th>
                  <th className="px-5 py-3 font-medium">Next Due</th>
                  <th className="px-5 py-3 font-medium">Last Ping</th>
                  <th className="px-5 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {checks.map((check) => {
                  const sc = statusStyles[check.status as keyof typeof statusStyles] ?? statusStyles.paused;
                  return (
                    <tr key={check.id} className={tw.tableRow}>
                      <td className="px-5 py-3">
                        <div className="flex items-center gap-2.5">
                          <div
                            className={`h-2 w-2 shrink-0 rounded-full ${sc.dot}`}
                            style={check.status === "up" ? inlineStyles.dotPulse : undefined}
                          />
                          <div>
                            <Link href={`/checks/${check.id}`} className="font-medium text-white transition hover:text-green-300">
                              {check.name}
                            </Link>
                            <p className="mt-0.5 text-xs text-slate-700">{check.id}</p>
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-3">
                        <span className={`inline-flex rounded border px-2 py-0.5 text-xs font-bold uppercase ${sc.badge}`}>
                          {check.status}
                        </span>
                      </td>
                      <td className="px-5 py-3 text-slate-400">
                        {check.expected_interval_seconds}s
                        <span className="text-slate-700"> + </span>
                        {check.grace_seconds}s
                        <span className="text-slate-600"> grace</span>
                      </td>
                      <td className="px-5 py-3 text-slate-400">{formatDate(check.next_due_at)}</td>
                      <td className="px-5 py-3 text-slate-400">{formatDate(check.last_ping_at)}</td>
                      <td className="px-5 py-3">
                        <div className="flex items-center gap-2">
                          {check.status === "paused" ? (
                            <form action={resumeCheckAction}>
                              <input type="hidden" name="check_id" value={check.id} />
                              <input type="hidden" name="offset" value={String(offset)} />
                              <input type="hidden" name="limit" value={String(limit)} />
                              <button type="submit" className={tw.btnActionGreen}>[resume]</button>
                            </form>
                          ) : (
                            <form action={pauseCheckAction}>
                              <input type="hidden" name="check_id" value={check.id} />
                              <input type="hidden" name="offset" value={String(offset)} />
                              <input type="hidden" name="limit" value={String(limit)} />
                              <button type="submit" className={tw.btnActionYellow}>[pause]</button>
                            </form>
                          )}
                          <Link href={`/checks/${check.id}`} className={tw.btnActionSlate}>
                            [details]
                          </Link>
                        </div>
                      </td>
                    </tr>
                  );
                })}
                {checks.length === 0 && (
                  <tr>
                    <td colSpan={6} className="px-5 py-14 text-center">
                      <p className="text-slate-600">
                        <span className="text-green-500">$</span> No checks found for this account.
                      </p>
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>
      </main>

      <Footer links={[{ label: "home", href: "/" }, { label: "docs", href: "/docs" }, { label: "github", href: "#" }]} />
    </div>
  );
}
