import { ApiError } from "@/lib/api/client";

export function toActionErrorMessage(error: unknown) {
  if (error instanceof ApiError) {
    return `${error.message} (status ${error.status})`;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return "Request failed";
}
