import { useQuery } from '@tanstack/react-query'
import { vrpcClient } from '@/config/vrpc-client'
import { createAdminApiService } from '@/skeled/admin'

const service = createAdminApiService(vrpcClient)

export function useConfigAccess() {
  const query = useQuery({
    queryKey: ['hub-config-readonly'],
    queryFn: () => service.readOnly(null),
    refetchOnWindowFocus: true,
  })
  return { readOnly: query.data !== false, loading: query.isPending, error: query.isError }
}
