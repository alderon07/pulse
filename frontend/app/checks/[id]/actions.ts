"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import {
  createAlertChannel,
  patchAlertChannel,
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
