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
  createTaskDebugService,
  type TaskDebugTaskItem,
  type TaskDebugTriggerItem,
} from '@/skeled'

const taskDebugService = createTaskDebugService(vrpcClient)
const jsonExtensions = [json()]
const defaultArgumentsJson = '{\n  \n}'
const storageArgumentsPrefix = 'vine.hub.debug.taskLauncher.arguments.'
const storageTaskSkelNameKey = 'vine.hub.debug.taskLauncher.taskSkelName'
const storageTriggerSkelNameKey =
  'vine.hub.debug.taskLauncher.triggerSkelName'

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function taskKey(task: TaskDebugTaskItem) {
  return `${task.taskSkelName}:${task.schemaHash}`
}

function argumentsStorageKey(
  taskSkelName: string,
  schemaHash: string,
  triggerSkelName: string,
) {
  return `${storageArgumentsPrefix}${taskSkelName}:${schemaHash}:${triggerSkelName}`
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
    // Ignore private-mode or quota failures; the launch itself already ran.
  }
}

function taskLauncherPath(taskSkelName?: string, triggerSkelName?: string) {
  const base = '/debug/task-launcher'
  if (!taskSkelName) {
    return base
  }
  const taskPath = `${base}/${encodeURIComponent(taskSkelName)}`
  if (!triggerSkelName) {
    return taskPath
  }
  return `${taskPath}/${encodeURIComponent(triggerSkelName)}`
}

function selectedPathParts(pathname: string) {
  const prefix = '/debug/task-launcher'
  if (pathname !== prefix && !pathname.startsWith(`${prefix}/`)) {
    return { isCurrent: false, taskSkelName: null, triggerSkelName: null }
  }
  const parts = pathname
    .slice(prefix.length)
    .split('/')
    .filter(Boolean)
    .map((part) => decodeURIComponent(part))
  return {
    isCurrent: true,
    taskSkelName: parts[0] ?? null,
    triggerSkelName: parts[1] ?? null,
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

export function TaskLauncherPage() {
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const pathSelection = React.useMemo(
    () => selectedPathParts(pathname),
    [pathname],
  )
  const [tasks, setTasks] = React.useState<Array<TaskDebugTaskItem>>([])
  const [taskQuery, setTaskQuery] = React.useState('')
  const [taskSelectOpen, setTaskSelectOpen] = React.useState(false)
  const taskSearchInputRef = React.useRef<HTMLInputElement>(null)
  const [triggers, setTriggers] = React.useState<Array<TaskDebugTriggerItem>>(
    [],
  )
  const [selectedTask, setSelectedTask] =
    React.useState<TaskDebugTaskItem | null>(null)
  const [selectedTrigger, setSelectedTrigger] =
    React.useState<TaskDebugTriggerItem | null>(null)
  const [argumentsJson, setArgumentsJson] =
    React.useState(defaultArgumentsJson)
  const [traceId, setTraceId] = React.useState('')
  const [spanId, setSpanId] = React.useState('')
  const [loadingTasks, setLoadingTasks] = React.useState(true)
  const [loadingTriggers, setLoadingTriggers] = React.useState(false)
  const [loadingDefaultRequest, setLoadingDefaultRequest] =
    React.useState(false)
  const [launching, setLaunching] = React.useState(false)
  const [result, setResult] = React.useState('No task launched yet.')

  const selectedTaskKey = selectedTask ? taskKey(selectedTask) : null
  const selectedTriggerKey = selectedTrigger?.skelName ?? null
  const filteredTasks = React.useMemo(() => {
    const keyword = taskQuery.trim().toLowerCase()
    if (keyword === '') {
      return tasks
    }
    return tasks.filter((item) =>
      item.taskSkelName.toLowerCase().includes(keyword),
    )
  }, [taskQuery, tasks])
  const selectedTriggerBelongsToTask =
    selectedTrigger !== null &&
    triggers.some((trigger) => trigger.skelName === selectedTrigger.skelName)

  const navigateToSelection = React.useCallback(
    (taskSkelName?: string, triggerSkelName?: string) => {
      void navigate({ to: taskLauncherPath(taskSkelName, triggerSkelName) })
    },
    [navigate],
  )

  const focusTaskSearchInput = React.useCallback(() => {
    window.requestAnimationFrame(() => {
      taskSearchInputRef.current?.focus()
    })
  }, [])

  React.useEffect(() => {
    if (!taskSelectOpen) {
      return
    }
    focusTaskSearchInput()
  }, [focusTaskSearchInput, taskQuery, taskSelectOpen])

  const loadDefaultLaunchRequest = React.useCallback(async () => {
    if (selectedTask === null || !selectedTriggerBelongsToTask) {
      setTraceId('')
      setSpanId('')
      setArgumentsJson(defaultArgumentsJson)
      return
    }

    setLoadingDefaultRequest(true)
    try {
      const nextRequest = await taskDebugService.buildDefaultLaunchRequest({
        taskSkelName: selectedTask.taskSkelName,
        schemaHash: selectedTask.schemaHash,
        triggerSkelName: selectedTrigger.skelName,
      })
      const savedArgumentsJson = readStorage(
        argumentsStorageKey(
          selectedTask.taskSkelName,
          selectedTask.schemaHash,
          selectedTrigger.skelName,
        ),
      )
      setTraceId(nextRequest.traceId)
      setSpanId(nextRequest.spanId)
      setArgumentsJson(savedArgumentsJson ?? nextRequest.argumentsJson)
      setResult('No task launched yet.')
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoadingDefaultRequest(false)
    }
  }, [selectedTask, selectedTrigger, selectedTriggerBelongsToTask])

  React.useEffect(() => {
    let ignore = false
    setLoadingTasks(true)
    void taskDebugService
      .listTasks(null)
      .then((nextTasks) => {
        if (!ignore) {
          setTasks(nextTasks)
        }
      })
      .catch((error: unknown) => {
        if (!ignore) {
          toast.error(getErrorMessage(error))
        }
      })
      .finally(() => {
        if (!ignore) {
          setLoadingTasks(false)
        }
      })
    return () => {
      ignore = true
    }
  }, [])

  React.useEffect(() => {
    const preferredTaskSkelName =
      pathSelection.taskSkelName ?? readStorage(storageTaskSkelNameKey)
    const nextTask =
      tasks.find((item) => item.taskSkelName === preferredTaskSkelName) ??
      tasks[0] ??
      null
    setSelectedTask((current) => {
      if (
        current === nextTask ||
        (current !== null &&
          nextTask !== null &&
          taskKey(current) === taskKey(nextTask))
      ) {
        return current
      }
      return nextTask
    })
  }, [pathSelection.taskSkelName, tasks])

  React.useEffect(() => {
    if (selectedTask === null) {
      setTriggers([])
      setSelectedTrigger(null)
      return
    }

    let ignore = false
    setLoadingTriggers(true)
    setTriggers([])
    setSelectedTrigger(null)
    void taskDebugService
      .listTriggers({
        taskSkelName: selectedTask.taskSkelName,
        schemaHash: selectedTask.schemaHash,
      })
      .then((nextTriggers) => {
        if (!ignore) {
          setTriggers(nextTriggers)
        }
      })
      .catch((error: unknown) => {
        if (!ignore) {
          toast.error(getErrorMessage(error))
        }
      })
      .finally(() => {
        if (!ignore) {
          setLoadingTriggers(false)
        }
      })
    return () => {
      ignore = true
    }
  }, [selectedTask])

  React.useEffect(() => {
    const preferredTriggerSkelName =
      pathSelection.triggerSkelName ?? readStorage(storageTriggerSkelNameKey)
    const nextTrigger =
      triggers.find(
        (item) => item.skelName === preferredTriggerSkelName,
      ) ??
      triggers[0] ??
      null
    setSelectedTrigger((current) => {
      if (
        current === nextTrigger ||
        current?.skelName === nextTrigger?.skelName
      ) {
        return current
      }
      return nextTrigger
    })
  }, [pathSelection.triggerSkelName, triggers])

  React.useEffect(() => {
    if (
      !pathSelection.isCurrent ||
      selectedTask === null ||
      selectedTrigger === null ||
      !selectedTriggerBelongsToTask
    ) {
      return
    }
    writeStorage(storageTaskSkelNameKey, selectedTask.taskSkelName)
    writeStorage(storageTriggerSkelNameKey, selectedTrigger.skelName)
    if (
      pathSelection.taskSkelName !== selectedTask.taskSkelName ||
      pathSelection.triggerSkelName !== selectedTrigger.skelName
    ) {
      navigateToSelection(selectedTask.taskSkelName, selectedTrigger.skelName)
    }
  }, [
    navigateToSelection,
    pathSelection.isCurrent,
    pathSelection.taskSkelName,
    pathSelection.triggerSkelName,
    selectedTask,
    selectedTrigger,
    selectedTriggerBelongsToTask,
  ])

  React.useEffect(() => {
    void loadDefaultLaunchRequest()
  }, [loadDefaultLaunchRequest])

  const launchTask = React.useCallback(async () => {
    if (selectedTask === null || selectedTrigger === null) {
      return
    }

    setLaunching(true)
    try {
      await taskDebugService.launchTask({
        request: {
          taskSkelName: selectedTask.taskSkelName,
          schemaHash: selectedTask.schemaHash,
          triggerSkelName: selectedTrigger.skelName,
          argumentsJson,
          traceId: traceId.trim() === '' ? null : traceId.trim(),
          spanId: spanId.trim() === '' ? null : spanId.trim(),
        },
      })
      writeStorage(
        argumentsStorageKey(
          selectedTask.taskSkelName,
          selectedTask.schemaHash,
          selectedTrigger.skelName,
        ),
        argumentsJson,
      )
      setResult('Task message published.')
      toast.success('Task launched')
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLaunching(false)
    }
  }, [argumentsJson, selectedTask, selectedTrigger, spanId, traceId])

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
        <div className="grid items-start gap-3 lg:grid-cols-[minmax(260px,1fr)_minmax(260px,1fr)_auto]">
          <label className="grid min-w-0 gap-1.5">
            <span className="text-xs font-medium text-muted-foreground">
              Task
            </span>
            <Select
              open={taskSelectOpen}
              onOpenChange={setTaskSelectOpen}
              value={selectedTaskKey ?? undefined}
              onValueChange={(value) => {
                const nextTask = tasks.find((item) => taskKey(item) === value)
                if (nextTask) {
                  setSelectedTask(nextTask)
                  setTriggers([])
                  setSelectedTrigger(null)
                  setTaskSelectOpen(false)
                  navigateToSelection(nextTask.taskSkelName)
                }
              }}
              disabled={loadingTasks || tasks.length === 0}
            >
              <SelectTrigger className="h-auto w-full rounded-lg border-transparent bg-primary/[0.05] px-3 py-2.5 hover:bg-primary/[0.07] focus-visible:border-primary/30">
                <SelectCardText
                  title={selectedTask?.taskSkelName}
                  description={selectedTask?.schemaHash}
                  placeholder={loadingTasks ? 'Loading tasks' : 'Select task'}
                />
              </SelectTrigger>
              <SelectContent
                align="start"
                alignItemWithTrigger={false}
                className="p-0"
                onPointerEnter={focusTaskSearchInput}
                onPointerMove={focusTaskSearchInput}
                header={
                  <div className="sticky top-0 z-10 border-b border-border bg-popover p-2">
                    <div className="relative">
                      <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        ref={taskSearchInputRef}
                        value={taskQuery}
                        className="h-8 pl-8"
                        placeholder="Search task"
                        onChange={(event) => setTaskQuery(event.target.value)}
                        onKeyDownCapture={(event) => event.stopPropagation()}
                        onKeyDown={(event) => event.stopPropagation()}
                      />
                    </div>
                  </div>
                }
              >
                {filteredTasks.length === 0 ? (
                  <div className="px-3 py-4 text-sm text-muted-foreground">
                    No tasks found.
                  </div>
                ) : (
                  filteredTasks.map((item) => (
                    <SelectItem
                      key={taskKey(item)}
                      value={taskKey(item)}
                      className={cn(
                        'rounded-lg border px-3 py-2.5 pr-8 hover:border-primary/30 hover:bg-primary/[0.06] focus:border-primary/30 focus:bg-primary/[0.06]',
                        selectedTaskKey === taskKey(item)
                          ? 'border-primary/30 bg-primary/[0.06]'
                          : 'border-transparent',
                      )}
                    >
                      <SelectCardItem
                        title={item.taskSkelName}
                        description={item.schemaHash}
                      />
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>
          </label>

          <label className="grid min-w-0 gap-1.5">
            <span className="text-xs font-medium text-muted-foreground">
              Trigger
            </span>
            <Select
              value={selectedTriggerKey ?? undefined}
              onValueChange={(value) => {
                const nextTrigger = triggers.find(
                  (item) => item.skelName === value,
                )
                if (nextTrigger) {
                  setSelectedTrigger(nextTrigger)
                  navigateToSelection(
                    selectedTask?.taskSkelName,
                    nextTrigger.skelName,
                  )
                }
              }}
              disabled={
                selectedTask === null ||
                loadingTriggers ||
                triggers.length === 0
              }
            >
              <SelectTrigger className="h-auto w-full rounded-lg border-transparent bg-primary/[0.05] px-3 py-2.5 hover:bg-primary/[0.07] focus-visible:border-primary/30">
                <SelectCardText
                  title={selectedTrigger?.skelName}
                  description={selectedTrigger?.description ?? undefined}
                  placeholder={
                    loadingTriggers ? 'Loading triggers' : 'Select trigger'
                  }
                />
              </SelectTrigger>
              <SelectContent
                align="start"
                alignItemWithTrigger={false}
                className="p-0"
              >
                {triggers.map((item) => (
                  <SelectItem
                    key={item.skelName}
                    value={item.skelName}
                    className={cn(
                      'rounded-lg border px-3 py-2.5 pr-8 hover:border-primary/30 hover:bg-primary/[0.06] focus:border-primary/30 focus:bg-primary/[0.06]',
                      selectedTriggerKey === item.skelName
                        ? 'border-primary/30 bg-primary/[0.06]'
                        : 'border-transparent',
                    )}
                  >
                    <SelectCardItem
                      title={item.skelName}
                      description={item.description ?? item.inputDescription ?? ''}
                    />
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>

          <div className="flex items-center justify-end gap-2 pt-8">
            <Button
              type="button"
              variant="outline"
              className="h-10 min-w-28"
              disabled={loadingDefaultRequest}
              onClick={() => void loadDefaultLaunchRequest()}
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
                selectedTask === null ||
                selectedTrigger === null ||
                launching ||
                loadingDefaultRequest
              }
              onClick={() => void launchTask()}
            >
              {launching ? <Loader2 className="animate-spin" /> : <Send />}
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
                  Arguments
                </span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-7 px-2 text-xs"
                  onClick={() =>
                    void copyText(argumentsJson, 'Arguments copied')
                  }
                >
                  <Copy />
                  Copy
                </Button>
              </div>
              <CodeMirror
                value={argumentsJson}
                extensions={jsonExtensions}
                onChange={setArgumentsJson}
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
