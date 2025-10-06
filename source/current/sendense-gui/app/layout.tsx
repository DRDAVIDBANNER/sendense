import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import "./globals.css";
import { Sidebar } from "@/components/layout/Sidebar";
import { ErrorBoundary } from "@/components/common/ErrorBoundary";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Sendense - Universal Backup Platform",
  description: "Enterprise-grade backup and replication management",
};

const queryClient = new QueryClient();

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased bg-background text-foreground`}
      >
        <QueryClientProvider client={queryClient}>
          <ErrorBoundary>
            <div className="flex h-screen">
              <Sidebar />
              <main className="flex-1 overflow-auto">
                {children}
              </main>
            </div>
          </ErrorBoundary>
        </QueryClientProvider>
      </body>
    </html>
  );
}
