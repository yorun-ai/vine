import {
  VrpcInvokeError,
  createVrpcClient,
  getClientInstanceId,
} from '@yorun-ai/vrpc/client'
import { toast } from 'sonner'

export const vrpcClient = createVrpcClient({
  prefixUrl: '/api/invoke',
  clientInfo: {
    clientName: 'vine.hub.dashboard',
    clientVersion: '0.0.1',
    clientInstanceId: getClientInstanceId(),
  },
})

vrpcClient.use({
  onError: (error) => {
    console.error('VRPC Error:', error)

    if (error instanceof VrpcInvokeError) {
      if (error.status === 401) {
        // TODO: Handle unauthorized error, e.g., redirect to login page
        console.warn('Unauthorized access - please log in.')
      } else {
        toast.error(error.message)
      }
    }
  },
})
