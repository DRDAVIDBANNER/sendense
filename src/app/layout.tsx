import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { QueryProvider } from '@/lib/queryClient'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'MigrateKit OSSEA - Migration Dashboard',
  description: 'VMware to OSSEA Migration Management Interface',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={inter.className} suppressHydrationWarning>
        <QueryProvider>
          {children}
        </QueryProvider>
      </body>
    </html>
  )
}