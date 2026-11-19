import { useEffectEvent, useLayoutEffect, useReducer, useState } from 'react'
import { toast } from 'sonner'

interface DebugRequest {
  json: string
  traceId: string
  spanId: string
}

const emptyRequest: DebugRequest = { json: '{\n  \n}', traceId: '', spanId: '' }

// The cleanup also guards transports that finish after cancellation.
export function startDefaultRequest(
  load: (signal: AbortSignal) => Promise<DebugRequest>,
  apply: (request: DebugRequest) => void,
  fail: (error: unknown) => void,
) {
  const controller = new AbortController()
  const { signal } = controller
  void load(signal).then(
    (request) => {
      if (!signal.aborted) apply(request)
    },
    (error: unknown) => {
      if (!signal.aborted) fail(error)
    },
  )
  return () => controller.abort()
}

export function useDebugDefaultRequest<T>(
  selection: T | null,
  load: (selection: T, signal: AbortSignal) => Promise<DebugRequest>,
  onLoaded: () => void,
) {
  const [request, setRequest] = useState(emptyRequest)
  const [loading, setLoading] = useState(false)
  const [revision, reload] = useReducer((value: number) => value + 1, 0)
  const didLoad = useEffectEvent(onLoaded)

  // A new selection starts with a blank editor. A manual reset of the same
  // selection keeps the current draft until its replacement is ready.
  useLayoutEffect(() => { setRequest(emptyRequest) }, [selection])
  useLayoutEffect(() => {
    setLoading(selection !== null)
    if (selection === null) return
    return startDefaultRequest(
      (signal) => load(selection, signal),
      (defaults) => {
        setRequest(defaults)
        setLoading(false)
        didLoad()
      },
      (error) => {
        setLoading(false)
        toast.error(error instanceof Error ? error.message : 'Request failed')
      },
    )
  }, [selection, load, revision])

  function setJson(json: string) {
    setRequest((current) => ({ ...current, json }))
  }

  function reset() {
    if (selection === null) setRequest(emptyRequest)
    else reload()
  }

  return { ...request, setJson, loading, reset }
}
