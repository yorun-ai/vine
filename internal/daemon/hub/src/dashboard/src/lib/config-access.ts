import { useQuery } from '@tanstack/react-query'
import { vrpcClient } from '@/config/vrpc-client'
import { createMaintenanceApiService } from '@/skeled/admin'

const service = createMaintenanceApiService(vrpcClient)

export function useConfigAccess() {
  const query = useQuery({
    queryKey: ['hub-config-readonly'],
    queryFn: () => service.configReadOnly(null),
    refetchOnWindowFocus: true,
  })
  return { readOnly: query.data !== false, loading: query.isPending, error: query.isError }
}
