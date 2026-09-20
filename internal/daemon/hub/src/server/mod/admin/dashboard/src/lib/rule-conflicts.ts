import { useQuery, type QueryClient } from '@tanstack/react-query'
import { vrpcClient } from '@/config/vrpc-client'
import { conflictsByRuleId } from '@/features/portal/rule-conflicts'
import { createPortalRuleApiService } from '@/skeled/admin'
import type { PortalRuleConflict } from '@/skeled/admin'

const service = createPortalRuleApiService(vrpcClient)

// ruleConflictQueryKey names the conflicts query, so a page that changes a rule,
// an entry, or a site asks Hub for the conflicts again.
export const ruleConflictQueryKey = ['portal-rule-conflicts'] as const

export function invalidateRuleConflicts(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: ruleConflictQueryKey })
}

// useRuleConflicts lists the rules Hub reports as matching the same request. The
// request a rule matches depends on the Web mount path of its site, so Hub
// answers this question from the schemas it holds instead of at write time: a
// page marks the rules Portal cannot order on its own.
export function useRuleConflicts() {
  const query = useQuery({
    queryKey: ruleConflictQueryKey,
    queryFn: () => service.listConflicts(null),
    refetchOnWindowFocus: true,
  })
  const conflicts: PortalRuleConflict[] = query.data ?? []
  return {
    conflicts,
    byRuleId: conflictsByRuleId(conflicts),
    loading: query.isPending,
  }
}
