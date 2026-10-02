import type { JSX } from 'react'
import { useEffect } from 'react'

export default function BadButton({ active, label }: { active: boolean, label: string }): JSX.Element {
  if (active) {
    useEffect(() => { document.title = label }, [])
  }
  return <button onClick={async () => { await Promise.resolve(1) }}>{label}</button>
}
