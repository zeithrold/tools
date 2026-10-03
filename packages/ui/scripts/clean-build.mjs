import { rm } from 'node:fs/promises'

// Clear only this package's reproducible output so removed modules cannot survive a rebuild or pack.
await rm(new URL('../dist/', import.meta.url), { recursive: true, force: true })
