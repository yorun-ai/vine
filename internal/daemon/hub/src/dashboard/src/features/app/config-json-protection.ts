import { EditorState, StateField } from '@uiw/react-codemirror'
import { isConfigValueChange } from './config-json-document.ts'
import type { ConfigValueRange } from './config-json-document.ts'

export function createConfigValueProtection(initialRanges: Array<ConfigValueRange>) {
  const ranges = StateField.define({
    create: () => initialRanges,
    update: (current, transaction) => current.map((range) => ({
      ...range,
      blockFrom: range.blockFrom === undefined ? undefined : transaction.changes.mapPos(range.blockFrom, -1),
      from: transaction.changes.mapPos(range.from, -1),
      to: transaction.changes.mapPos(range.to, 1),
    })),
  })
  const filter = EditorState.transactionFilter.of((transaction) => {
    if (!transaction.docChanged) {
      return transaction
    }
    let allowed = true
    transaction.changes.iterChanges((from, to) => {
      if (!isConfigValueChange(transaction.startState.field(ranges), from, to)) {
        allowed = false
      }
    })
    return allowed ? transaction : []
  })

  return { ranges, extensions: [ranges, filter] }
}
