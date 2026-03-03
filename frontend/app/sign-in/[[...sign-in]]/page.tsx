import { SignIn } from "@clerk/nextjs";
import { isClerkUiEnabled } from "@/lib/clerk-config";

export default function SignInPage() {
  if (!isClerkUiEnabled) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-black p-6 text-slate-300">
        Clerk is not configured in this environment.
      </main>
    );
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-black p-6">
      <SignIn />
    </main>
  );
}
