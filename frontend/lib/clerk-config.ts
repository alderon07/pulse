export const hasClerkPublishableKey = Boolean(process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY);
export const hasClerkSecretKey = Boolean(process.env.CLERK_SECRET_KEY);

export const isClerkUiEnabled = hasClerkPublishableKey;
export const isClerkBackendEnabled = hasClerkPublishableKey && hasClerkSecretKey;
