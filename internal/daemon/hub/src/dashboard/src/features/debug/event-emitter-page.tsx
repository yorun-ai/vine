import * as React from 'react'
import { json } from '@codemirror/lang-json'
import CodeMirror from '@uiw/react-codemirror'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { Copy, Loader2, RotateCcw, Search, Send } from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import { vrpcClient } from '@/config/vrpc-client'
import { cn } from '@/lib/utils'
import {
  createEventDebugService,
  type EventDebugEventItem,
} from '@/skeled'

const eventDebugService = createEventDebugService(vrpcClient)
const jsonExtensions = [json()]
const defaultEventJson = '{\n  \n}'
const storageEventJsonPrefix = 'vine.hub.debug.eventEmitter.eventJson.'
const storageEventSkelNameKey = 'vine.hub.debug.eventEmitter.eventSkelName'

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function eventKey(event: EventDebugEventItem) {
  return `${event.eventSkelName}:${event.schemaHash}`
}

function eventJsonStorageKey(eventSkelName: string, schemaHash: string) {
  return `${storageEventJsonPrefix}${eventSkelName}:${schemaHash}`
}

function readStorage(key: string) {
  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

function writeStorage(key: string, value: string) {
  try {
    window.localStorage.setItem(key, value)
  } catch {
    // Ignore private-mode or quota failures; the emit itself already ran.
  }
}

function eventEmitterPath(eventSkelName?: string) {
  const base = '/debug/event-emitter'
  if (!eventSkelName) {
    return base
  }
  return `${base}/${encodeURIComponent(eventSkelName)}`
}

function selectedPathParts(pathname: string) {
  const prefix = '/debug/event-emitter'
  if (pathname !== prefix && !pathname.startsWith(`${prefix}/`)) {
    return { eventSkelName: null, isCurrent: false }
  }
  const parts = pathname
    .slice(prefix.length)
    .split('/')
    .filter(Boolean)
    .map((part) => decodeURIComponent(part))
  return {
    eventSkelName: parts[0] ?? null,
    isCurrent: true,
  }
}

interface SelectCardTextProps {
  description?: string
  placeholder?: string
  title?: string
}

function SelectCardText({
  description,
  placeholder,
  title,
}: SelectCardTextProps) {
  return (
    <span
      className={cn(
        'grid min-h-10 min-w-0 flex-1 content-center text-left',
        title && 'gap-0.5',
      )}
    >
      <span
        className={cn(
          'truncate text-sm font-semibold',
          title ? 'text-foreground' : 'text-muted-foreground',
        )}
      >
        {title || placeholder}
      </span>
      {title ? (
        <span
          className={cn(
            'min-h-4 truncate font-mono text-xs text-muted-foreground',
            !description && 'invisible',
          )}
        >
          {description || 'placeholder'}
        </span>
      ) : null}
    </span>
  )
}

function SelectCardItem({
  description,
  title,
}: Required<Pick<SelectCardTextProps, 'description' | 'title'>>) {
  return (
    <span className="grid min-w-0 flex-1 gap-0.5">
      <span className="truncate text-sm font-semibold text-foreground">
        {title}
      </span>
      <span className="truncate font-mono text-xs text-muted-foreground">
        {description}
      </span>
    </span>
  )
}

export function EventEmitterPage() {
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const pathSelection = React.useMemo(
    () => selectedPathParts(pathname),
    [pathname],
  )
  const [events, setEvents] = React.useState<Array<EventDebugEventItem>>([])
  const [eventQuery, setEventQuery] = React.useState('')
  const [eventSelectOpen, setEventSelectOpen] = React.useState(false)
  const eventSearchInputRef = React.useRef<HTMLInputElement>(null)
  const [selectedEvent, setSelectedEvent] =
    React.useState<EventDebugEventItem | null>(null)
  const [eventJson, setEventJson] = React.useState(defaultEventJson)
  const [traceId, setTraceId] = React.useState('')
  const [spanId, setSpanId] = React.useState('')
  const [loadingEvents, setLoadingEvents] = React.useState(true)
  const [loadingDefaultRequest, setLoadingDefaultRequest] =
    React.useState(false)
  const [emitting, setEmitting] = React.useState(false)
  const [result, setResult] = React.useState('No event emitted yet.')

  const selectedEventKey = selectedEvent ? eventKey(selectedEvent) : null
  const filteredEvents = React.useMemo(() => {
    const keyword = eventQuery.trim().toLowerCase()
    if (keyword === '') {
      return events
    }
    return events.filter((item) =>
      item.eventSkelName.toLowerCase().includes(keyword),
    )
  }, [eventQuery, events])

  const navigateToSelection = React.useCallback(
    (eventSkelName?: string) => {
      void navigate({ to: eventEmitterPath(eventSkelName) })
    },
    [navigate],
  )

  const focusEventSearchInput = React.useCallback(() => {
    window.requestAnimationFrame(() => {
      eventSearchInputRef.current?.focus()
    })
  }, [])

  React.useEffect(() => {
    if (!eventSelectOpen) {
      return
    }
    focusEventSearchInput()
  }, [eventQuery, eventSelectOpen, focusEventSearchInput])

  const loadDefaultEmitRequest = React.useCallback(async () => {
    if (selectedEvent === null) {
      setTraceId('')
      setSpanId('')
      setEventJson(defaultEventJson)
      return
    }

    setLoadingDefaultRequest(true)
    try {
      const nextRequest = await eventDebugService.buildDefaultEmitRequest({
        eventSkelName: selectedEvent.eventSkelName,
        schemaHash: selectedEvent.schemaHash,
      })
      const savedEventJson = readStorage(
        eventJsonStorageKey(
          selectedEvent.eventSkelName,
          selectedEvent.schemaHash,
        ),
      )
      setTraceId(nextRequest.traceId)
      setSpanId(nextRequest.spanId)
      setEventJson(savedEventJson ?? nextRequest.eventJson)
      setResult('No event emitted yet.')
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoadingDefaultRequest(false)
    }
  }, [selectedEvent])

  React.useEffect(() => {
    let ignore = false
    setLoadingEvents(true)
    void eventDebugService
      .listEvents(null)
      .then((nextEvents) => {
        if (!ignore) {
          setEvents(nextEvents)
        }
      })
      .catch((error: unknown) => {
        if (!ignore) {
          toast.error(getErrorMessage(error))
        }
      })
      .finally(() => {
        if (!ignore) {
          setLoadingEvents(false)
        }
      })
    return () => {
      ignore = true
    }
  }, [])

  React.useEffect(() => {
    const preferredEventSkelName =
      pathSelection.eventSkelName ?? readStorage(storageEventSkelNameKey)
    const nextEvent =
      events.find(
        (item) => item.eventSkelName === preferredEventSkelName,
      ) ??
      events[0] ??
      null
    setSelectedEvent((current) => {
      if (
        current === nextEvent ||
        (current !== null &&
          nextEvent !== null &&
          eventKey(current) === eventKey(nextEvent))
      ) {
        return current
      }
      return nextEvent
    })
  }, [events, pathSelection.eventSkelName])

  React.useEffect(() => {
    if (selectedEvent === null) {
      return
    }
    if (!pathSelection.isCurrent) {
      return
    }
    writeStorage(storageEventSkelNameKey, selectedEvent.eventSkelName)
    if (pathSelection.eventSkelName !== selectedEvent.eventSkelName) {
      navigateToSelection(selectedEvent.eventSkelName)
    }
  }, [
    navigateToSelection,
    pathSelection.eventSkelName,
    pathSelection.isCurrent,
    selectedEvent,
  ])

  React.useEffect(() => {
    void loadDefaultEmitRequest()
  }, [loadDefaultEmitRequest])

  const emitEvent = React.useCallback(async () => {
    if (selectedEvent === null) {
      return
    }

    setEmitting(true)
    try {
      await eventDebugService.emitEvent({
        request: {
          eventSkelName: selectedEvent.eventSkelName,
          schemaHash: selectedEvent.schemaHash,
          eventJson,
          traceId: traceId.trim() === '' ? null : traceId.trim(),
          spanId: spanId.trim() === '' ? null : spanId.trim(),
        },
      })
      writeStorage(
        eventJsonStorageKey(
          selectedEvent.eventSkelName,
          selectedEvent.schemaHash,
        ),
        eventJson,
      )
      setResult('Event message published.')
      toast.success('Event emitted')
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setEmitting(false)
    }
  }, [eventJson, selectedEvent, spanId, traceId])

  const copyText = React.useCallback(async (value: string, message: string) => {
    try {
      await navigator.clipboard.writeText(value)
      toast.success(message)
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }, [])

  return (
    <div className="flex h-full min-h-0 flex-col bg-background">
      <section className="shrink-0 border-b border-border px-3 py-2">
        <div className="grid items-start gap-3 lg:grid-cols-[minmax(260px,1fr)_auto]">
          <label className="grid min-w-0 gap-1.5">
            <span className="text-xs font-medium text-muted-foreground">
              Event
            </span>
            <Select
              open={eventSelectOpen}
              onOpenChange={setEventSelectOpen}
              value={selectedEventKey ?? undefined}
              onValueChange={(value) => {
                const nextEvent = events.find((item) => eventKey(item) === value)
                if (nextEvent) {
                  setSelectedEvent(nextEvent)
                  setEventSelectOpen(false)
                  navigateToSelection(nextEvent.eventSkelName)
                }
              }}
              disabled={loadingEvents || events.length === 0}
            >
              <SelectTrigger className="h-auto w-full rounded-lg border-transparent bg-primary/[0.05] px-3 py-2.5 hover:bg-primary/[0.07] focus-visible:border-primary/30">
                <SelectCardText
                  title={selectedEvent?.eventSkelName}
                  description={selectedEvent?.schemaHash}
                  placeholder={loadingEvents ? 'Loading events' : 'Select event'}
                />
              </SelectTrigger>
              <SelectContent
                align="start"
                alignItemWithTrigger={false}
                className="p-0"
                onPointerEnter={focusEventSearchInput}
                onPointerMove={focusEventSearchInput}
                header={
                  <div className="sticky top-0 z-10 border-b border-border bg-popover p-2">
                    <div className="relative">
                      <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        ref={eventSearchInputRef}
                        value={eventQuery}
                        className="h-8 pl-8"
                        placeholder="Search event"
                        onChange={(event) => setEventQuery(event.target.value)}
                        onKeyDownCapture={(event) => event.stopPropagation()}
                        onKeyDown={(event) => event.stopPropagation()}
                      />
                    </div>
                  </div>
                }
              >
                {filteredEvents.length === 0 ? (
                  <div className="px-3 py-4 text-sm text-muted-foreground">
                    No events found.
                  </div>
                ) : (
                  filteredEvents.map((item) => (
                    <SelectItem
                      key={eventKey(item)}
                      value={eventKey(item)}
                      className={cn(
                        'rounded-lg border px-3 py-2.5 pr-8 hover:border-primary/30 hover:bg-primary/[0.06] focus:border-primary/30 focus:bg-primary/[0.06]',
                        selectedEventKey === eventKey(item)
                          ? 'border-primary/30 bg-primary/[0.06]'
                          : 'border-transparent',
                      )}
                    >
                      <SelectCardItem
                        title={item.eventSkelName}
                        description={item.schemaHash}
                      />
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>
          </label>

          <div className="flex items-center justify-end gap-2 pt-8">
            <Button
              type="button"
              variant="outline"
              className="h-10 min-w-28"
              disabled={loadingDefaultRequest}
              onClick={() => void loadDefaultEmitRequest()}
            >
              {loadingDefaultRequest ? (
                <Loader2 className="animate-spin" />
              ) : (
                <RotateCcw />
              )}
              Reset
            </Button>
            <Button
              type="button"
              className="h-10 min-w-28"
              disabled={
                selectedEvent === null ||
                emitting ||
                loadingDefaultRequest
              }
              onClick={() => void emitEvent()}
            >
              {emitting ? <Loader2 className="animate-spin" /> : <Send />}
              Send
            </Button>
          </div>
        </div>
      </section>

      <main className="grid min-h-0 flex-1 grid-rows-[minmax(0,1fr)_minmax(160px,0.45fr)]">
        <section className="flex min-h-0 flex-col border-b border-border">
          <div className="flex h-11 shrink-0 items-center gap-6 border-b border-border px-4">
            <h2 className="text-sm font-semibold text-foreground">Request</h2>
            {traceId || spanId ? (
              <div className="flex min-w-0 items-center gap-4 font-mono text-xs text-muted-foreground">
                <span className="truncate">traceId={traceId || '-'}</span>
                <span className="truncate">spanId={spanId || '-'}</span>
              </div>
            ) : null}
          </div>
          <div className="min-h-0 flex-1 p-3">
            <div className="flex h-full min-h-0 flex-col gap-1.5">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-foreground">
                  Event
                </span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-7 px-2 text-xs"
                  onClick={() => void copyText(eventJson, 'Event copied')}
                >
                  <Copy />
                  Copy
                </Button>
              </div>
              <CodeMirror
                value={eventJson}
                extensions={jsonExtensions}
                onChange={setEventJson}
                basicSetup={{
                  autocompletion: true,
                  bracketMatching: true,
                  closeBrackets: true,
                  foldGutter: true,
                  highlightActiveLine: true,
                  highlightActiveLineGutter: true,
                  lineNumbers: true,
                }}
                height="100%"
                className="min-h-0 flex-1 overflow-hidden rounded-md border border-input bg-background text-[13px] [&_.cm-editor]:h-full [&_.cm-scroller]:h-full [&_.cm-theme]:h-full"
                theme="light"
              />
            </div>
          </div>
        </section>

        <section className="flex min-h-0 flex-col">
          <div className="flex h-11 shrink-0 items-center border-b border-border px-4">
            <h2 className="text-sm font-semibold text-foreground">Result</h2>
          </div>
          <div className="min-h-0 flex-1 p-3">
            <pre className="scrollbar-reserved h-full overflow-auto rounded-md border border-input bg-muted/20 p-3 font-mono text-[13px] leading-5 text-muted-foreground">
              {result}
            </pre>
          </div>
        </section>
      </main>
    </div>
  )
}
