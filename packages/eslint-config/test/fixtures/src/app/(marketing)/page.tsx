import type { JSX } from 'react'

export function generateMetadata(): { title: string } {
  return { title: 'Strict page' }
}

export function generateViewport(): { width: string } {
  return { width: 'device-width' }
}

export function generateStaticParams(): { slug: string }[] {
  return []
}

export default function Page(): JSX.Element {
  return <main>Strict page</main>
}
