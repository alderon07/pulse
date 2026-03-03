import type { Metadata } from "next";
import { ClerkProvider } from "@clerk/nextjs";
import {
  Geist,
  Geist_Mono,
  JetBrains_Mono,
  Syne,
} from "next/font/google";
import "./globals.css";
import { hasClerkPublishableKey } from "@/lib/clerk-config";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains",
  subsets: ["latin"],
  display: "swap",
});

const syne = Syne({
  variable: "--font-syne",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "Pulse — Cron Monitoring",
  description:
    "Real-time heartbeat monitoring for your cron jobs. Get alerted instantly when scheduled tasks miss their check-in.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const content = (
    <body
      className={`${geistSans.variable} ${geistMono.variable} ${jetbrainsMono.variable} ${syne.variable} antialiased`}
    >
      {children}
    </body>
  );

  const publishableKey = process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;

  return (
    <html lang="en">
      {hasClerkPublishableKey && publishableKey ? (
        <ClerkProvider publishableKey={publishableKey}>{content}</ClerkProvider>
      ) : (
        content
      )}
    </html>
  );
}
