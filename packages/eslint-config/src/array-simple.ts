export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isLiteral(node: unknown): boolean {
  return isRecord(node) && node.type === 'Literal'
}

function isAccess(node: Record<string, unknown>): boolean {
  if (node.computed !== false || !isRecord(node.property) || node.property.type !== 'Identifier') {
    return false
  }
  if (!isRecord(node.object)) {
    return false
  }
  return node.object.type === 'Identifier'
    || (node.object.type === 'MemberExpression' && isAccess(node.object))
}

function isWrapper(node: Record<string, unknown>): boolean {
  const wrappers = [
    'TSAsExpression',
    'TSSatisfiesExpression',
    'TSNonNullExpression',
    'ChainExpression',
  ]
  return typeof node.type === 'string' && wrappers.includes(node.type) && isSimple(node.expression)
}

export function isSimple(node: unknown): boolean {
  if (!isRecord(node)) {
    return false
  }
  switch (node.type) {
    case 'Identifier':
      return true
    case 'Literal':
      return node.value === null || typeof node.value !== 'object'
    case 'UnaryExpression':
      return isLiteral(node.argument) && isSimple(node.argument)
    case 'MemberExpression':
      return isAccess(node)
    default:
      return isWrapper(node)
  }
}
