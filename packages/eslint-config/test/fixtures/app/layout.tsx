import type { JSX, ReactNode } from 'react'

export const metadata = { title: 'Strict layout' }
export const viewport = { width: 'device-width', initialScale: 1 }

interface LayoutProps {
  children: ReactNode
}

export default function RootLayout({ children }: LayoutProps): JSX.Element {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
