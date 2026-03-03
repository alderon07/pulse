"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import { createCheck, pauseCheck, resumeCheck } from "@/lib/api/client";
import { requireBackendAccessToken } from "@/lib/server/backend-auth";
import { toActionErrorMessage } from "@/lib/server/action-utils";

function parsePositiveInt(rawValue: FormDataEntryValue | null, fallback: number) {
  if (typeof rawValue !== "string") {
    return fallback;
  }

  const parsed = Number.parseInt(rawValue, 10);
  if (Number.isNaN(parsed) || parsed < 0) {
    return fallback;
  }

  return parsed;
}

function requiredField(formData: FormData, key: string, label: string) {
  const raw = formData.get(key);
  const value = typeof raw === "string" ? raw.trim() : "";

  if (!value) {
    throw new Error(`${label} is required`);
  }

  return value;
}

function parseStrictPositiveInt(rawValue: FormDataEntryValue | null, label: string) {
  if (typeof rawValue !== "string") {
    throw new Error(`${label} is required`);
  }

  const parsed = Number.parseInt(rawValue, 10);
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(`${label} must be greater than 0`);
  }

  return parsed;
}

function parseNonNegativeInt(rawValue: FormDataEntryValue | null, label: string) {
  if (typeof rawValue !== "string") {
    throw new Error(`${label} is required`);
  }

  const parsed = Number.parseInt(rawValue, 10);
  if (Number.isNaN(parsed) || parsed < 0) {
    throw new Error(`${label} must be greater than or equal to 0`);
  }

  return parsed;
}

function parseScheduleMode(rawValue: FormDataEntryValue | null) {
  if (typeof rawValue !== "string") {
    return "auto" as const;
  }
  const mode = rawValue.trim().toLowerCase();
  if (mode === "manual" || mode === "auto") {
    return mode;
  }
  throw new Error("schedule_mode must be manual or auto");
}

function parseOptionalStrictPositiveInt(rawValue: FormDataEntryValue | null, label: string) {
  if (typeof rawValue !== "string" || rawValue.trim() === "") {
    return undefined;
  }
  return parseStrictPositiveInt(rawValue, label);
}

function parseOptionalNonNegativeInt(rawValue: FormDataEntryValue | null, label: string) {
  if (typeof rawValue !== "string" || rawValue.trim() === "") {
    return undefined;
  }
  return parseNonNegativeInt(rawValue, label);
}

function checksPathFromFormData(formData: FormData) {
  const limit = parsePositiveInt(formData.get("limit"), 25);
  const offset = parsePositiveInt(formData.get("offset"), 0);

  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });

  return `/checks?${params.toString()}`;
}

function checksPathWithError(path: string, error: string) {
  const url = new URL(path, "https://pulse.local");
  url.searchParams.set("error", error);
  return `${url.pathname}?${url.searchParams.toString()}`;
}

export async function createCheckAction(formData: FormData) {
  const path = checksPathFromFormData(formData);

  try {
    const name = requiredField(formData, "name", "name");
    const scheduleMode = parseScheduleMode(formData.get("schedule_mode"));
    let expectedIntervalSeconds: number | undefined;
    let graceSeconds: number | undefined;

    if (scheduleMode === "manual") {
      expectedIntervalSeconds = parseStrictPositiveInt(
        formData.get("expected_interval_seconds"),
        "expected_interval_seconds"
      );
      graceSeconds = parseNonNegativeInt(formData.get("grace_seconds"), "grace_seconds");
    } else {
      expectedIntervalSeconds = parseOptionalStrictPositiveInt(
        formData.get("expected_interval_seconds"),
        "expected_interval_seconds"
      );
      graceSeconds = parseOptionalNonNegativeInt(formData.get("grace_seconds"), "grace_seconds");
    }

    const token = await requireBackendAccessToken();
    const created = await createCheck(token, {
      name,
      schedule_mode: scheduleMode,
      expected_interval_seconds: expectedIntervalSeconds,
      grace_seconds: graceSeconds,
    });

    revalidatePath("/checks");
    revalidatePath(`/checks/${created.id}`);
    redirect(`/checks/${created.id}`);
  } catch (error) {
    const message = toActionErrorMessage(error);
    redirect(checksPathWithError(path, message));
  }
}

export async function pauseCheckAction(formData: FormData) {
  const checkId = formData.get("check_id");
  const path = checksPathFromFormData(formData);

  if (typeof checkId !== "string" || !checkId.trim()) {
    redirect(`${path}&error=Missing+check+ID`);
  }

  try {
    const token = await requireBackendAccessToken();
    await pauseCheck(token, checkId);
  } catch (error) {
    const message = encodeURIComponent(toActionErrorMessage(error));
    redirect(`${path}&error=${message}`);
  }

  revalidatePath("/checks");
  redirect(path);
}

export async function resumeCheckAction(formData: FormData) {
  const checkId = formData.get("check_id");
  const path = checksPathFromFormData(formData);

  if (typeof checkId !== "string" || !checkId.trim()) {
    redirect(`${path}&error=Missing+check+ID`);
  }

  try {
    const token = await requireBackendAccessToken();
    await resumeCheck(token, checkId);
  } catch (error) {
    const message = encodeURIComponent(toActionErrorMessage(error));
    redirect(`${path}&error=${message}`);
  }

  revalidatePath("/checks");
  redirect(path);
}
