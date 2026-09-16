import {
  VrpcInvokeError,
  createVrpcClient,
  getClientInstanceId,
} from '@yorun-ai/vrpc/client'
import { toast } from 'sonner'

export const vrpcClient = createVrpcClient({
  // Hub serves this build and the Admin API on one listener, and the API answers
  // this path there. The Dashboard is the only client that reaches the API from
  // a browser, so the path is the listener's own instead of the runtime path the
  // components use between each other.
  prefixUrl: '/api/invoke',
  // This build is the first hop: Hub serves it, so the Dashboard opens the span
  // of the call instead of leaving it to a Portal entry that carried the request
  // before.
  traceMode: 'direct',
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
