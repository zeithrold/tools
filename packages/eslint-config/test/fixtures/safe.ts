export function describe(value: unknown): string {
  if (typeof value === 'string') {
    return value
  }
  return 'unknown'
}

export const ids = [
  1,
  2,
  3,
] as const
