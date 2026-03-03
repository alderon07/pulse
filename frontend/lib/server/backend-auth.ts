import "server-only";

import { auth } from "@clerk/nextjs/server";
import { redirect } from "next/navigation";
import { isClerkBackendEnabled } from "@/lib/clerk-config";

const JWT_TEMPLATE = process.env.CLERK_BACKEND_JWT_TEMPLATE?.trim();

export async function requireBackendAccessToken() {
  if (!isClerkBackendEnabled) {
    throw new Error(
      "Clerk is not configured. Set NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY and CLERK_SECRET_KEY."
    );
  }

  const { userId, getToken } = await auth();

  if (!userId) {
    redirect("/sign-in");
  }

  const token = JWT_TEMPLATE ? await getToken({ template: JWT_TEMPLATE }) : await getToken();

  if (!token) {
    throw new Error(
      "Unable to acquire a backend access token from Clerk. Configure CLERK_BACKEND_JWT_TEMPLATE or backend JWT verification."
    );
  }

  return token;
}
