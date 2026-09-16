import type { PortalRuleConflict } from '@/skeled/admin'

// conflictsByRuleId indexes the conflicts Hub reports by both rules they name,
// so a page marks every rule Portal cannot order on its own.
export function conflictsByRuleId(conflicts: readonly PortalRuleConflict[]) {
  const byRuleId = new Map<number, PortalRuleConflict>()
  for (const conflict of conflicts) {
    byRuleId.set(conflict.ruleId, conflict)
    byRuleId.set(conflict.conflictRuleId, conflict)
  }
  return byRuleId
}
