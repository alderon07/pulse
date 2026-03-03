"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import { pauseCheck, resumeCheck } from "@/lib/api/client";
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

function checksPathFromFormData(formData: FormData) {
  const limit = parsePositiveInt(formData.get("limit"), 25);
  const offset = parsePositiveInt(formData.get("offset"), 0);

  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });

  return `/checks?${params.toString()}`;
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
