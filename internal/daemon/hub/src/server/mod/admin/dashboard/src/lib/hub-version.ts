import { useQuery } from '@tanstack/react-query'
import { vrpcClient } from '@/config/vrpc-client'
import { createAdminApiService } from '@/skeled/admin'

const service = createAdminApiService(vrpcClient)

export function useHubVersion() {
  return useQuery({
    queryKey: ['hub-version'],
    queryFn: () => service.version(null),
    staleTime: Infinity,
  })
}
