"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import {
  createAlertChannel,
  patchAlertChannel,
  patchCheck,
  pauseCheck,
  resumeCheck,
  rotateCheckToken,
} from "@/lib/api/client";
import { alertChannelTypeSchema } from "@/lib/api/schemas";
import { requireBackendAccessToken } from "@/lib/server/backend-auth";
import { toActionErrorMessage } from "@/lib/server/action-utils";

function detailPath(checkId: string, error?: string) {
  if (!error) {
    return `/checks/${checkId}`;
  }

  return `/checks/${checkId}?error=${encodeURIComponent(error)}`;
}

function requiredField(formData: FormData, key: string, label: string) {
  const raw = formData.get(key);
  const value = typeof raw === "string" ? raw.trim() : "";

  if (!value) {
    throw new Error(`${label} is required`);
  }

  return value;
}

function parseEnabledValue(rawValue: string) {
  if (rawValue === "true") {
    return true;
  }
  if (rawValue === "false") {
    return false;
  }

  throw new Error("enabled must be true or false");
}

function parseStrictPositiveInt(rawValue: string, label: string) {
  const parsed = Number.parseInt(rawValue, 10);
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(`${label} must be greater than 0`);
  }
  return parsed;
}

function parseNonNegativeInt(rawValue: string, label: string) {
  const parsed = Number.parseInt(rawValue, 10);
  if (Number.isNaN(parsed) || parsed < 0) {
    throw new Error(`${label} must be greater than or equal to 0`);
  }
  return parsed;
}

function parseScheduleMode(rawValue: string) {
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

function revalidateCheckPaths(checkId: string) {
  revalidatePath("/checks");
  revalidatePath(`/checks/${checkId}`);
}

export async function pauseCheckFromDetailAction(formData: FormData) {
  const checkId = requiredField(formData, "check_id", "check_id");

  try {
    const token = await requireBackendAccessToken();
    await pauseCheck(token, checkId);
  } catch (error) {
    redirect(detailPath(checkId, toActionErrorMessage(error)));
  }

  revalidateCheckPaths(checkId);
  redirect(detailPath(checkId));
}

export async function resumeCheckFromDetailAction(formData: FormData) {
  const checkId = requiredField(formData, "check_id", "check_id");

  try {
    const token = await requireBackendAccessToken();
    await resumeCheck(token, checkId);
  } catch (error) {
    redirect(detailPath(checkId, toActionErrorMessage(error)));
  }

  revalidateCheckPaths(checkId);
  redirect(detailPath(checkId));
}

export async function rotateTokenAction(formData: FormData) {
  const checkId = requiredField(formData, "check_id", "check_id");

  try {
    const token = await requireBackendAccessToken();
    await rotateCheckToken(token, checkId);
  } catch (error) {
    redirect(detailPath(checkId, toActionErrorMessage(error)));
  }

  revalidateCheckPaths(checkId);
  redirect(detailPath(checkId));
}

export async function updateScheduleAction(formData: FormData) {
  const checkId = requiredField(formData, "check_id", "check_id");

  try {
    const scheduleMode = parseScheduleMode(requiredField(formData, "schedule_mode", "schedule_mode"));
    let expectedIntervalSeconds: number | undefined;
    let graceSeconds: number | undefined;
    if (scheduleMode === "manual") {
      expectedIntervalSeconds = parseStrictPositiveInt(
        requiredField(formData, "expected_interval_seconds", "expected_interval_seconds"),
        "expected_interval_seconds"
      );
      graceSeconds = parseNonNegativeInt(
        requiredField(formData, "grace_seconds", "grace_seconds"),
        "grace_seconds"
      );
    } else {
      expectedIntervalSeconds = parseOptionalStrictPositiveInt(
        formData.get("expected_interval_seconds"),
        "expected_interval_seconds"
      );
      graceSeconds = parseOptionalNonNegativeInt(formData.get("grace_seconds"), "grace_seconds");
    }

    const token = await requireBackendAccessToken();
    await patchCheck(token, checkId, {
      schedule_mode: scheduleMode,
      expected_interval_seconds: expectedIntervalSeconds,
      grace_seconds: graceSeconds,
    });
  } catch (error) {
    redirect(detailPath(checkId, toActionErrorMessage(error)));
  }

  revalidateCheckPaths(checkId);
  redirect(detailPath(checkId));
}

export async function createAlertChannelAction(formData: FormData) {
  const checkId = requiredField(formData, "check_id", "check_id");

  try {
    const type = requiredField(formData, "channel_type", "channel_type");
    const target = requiredField(formData, "target", "target");

    const parsedType = alertChannelTypeSchema.safeParse(type);
    if (!parsedType.success) {
      throw new Error("channel_type must be email or webhook");
    }

    const token = await requireBackendAccessToken();
    await createAlertChannel(token, {
      type: parsedType.data,
      target,
    });
  } catch (error) {
    redirect(detailPath(checkId, toActionErrorMessage(error)));
  }

  revalidateCheckPaths(checkId);
  redirect(detailPath(checkId));
}

export async function toggleAlertChannelAction(formData: FormData) {
  const checkId = requiredField(formData, "check_id", "check_id");

  try {
    const channelId = requiredField(formData, "channel_id", "channel_id");
    const enabled = parseEnabledValue(requiredField(formData, "enabled", "enabled"));

    const token = await requireBackendAccessToken();
    await patchAlertChannel(token, channelId, { enabled });
  } catch (error) {
    redirect(detailPath(checkId, toActionErrorMessage(error)));
  }

  revalidateCheckPaths(checkId);
  redirect(detailPath(checkId));
}
