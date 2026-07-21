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
import {
  createServiceDebugService,
  type ServiceDebugActorItem,
  type ServiceDebugAppInstance,
  type ServiceDebugInvokeResponse,
  type ServiceDebugMethodItem,
  type ServiceDebugServiceItem,
} from '@/skeled'
import { cn } from '@/lib/utils'

const serviceDebugService = createServiceDebugService(vrpcClient)
const jsonExtensions = [json()]
const defaultParams = '{\n  \n}'
const defaultActorInfo = '{}'
const defaultTimeoutSeconds = '30'
const serviceDebugClientTimeoutPaddingMs = 1000
const autoAppInstanceValue = '__auto__'
const storageActorSkelNameKey = 'vine.hub.debug.serviceClient.actorSkelName'
const storageActorInfoPrefix = 'vine.hub.debug.serviceClient.actorInfo.'
const storageParamsPrefix = 'vine.hub.debug.serviceClient.params.'
const storageServiceSkelNameKey = 'vine.hub.debug.serviceClient.serviceSkelName'
const storageMethodSkelNameKey = 'vine.hub.debug.serviceClient.methodSkelName'
const storageTimeoutSecondsKey = 'vine.hub.debug.serviceClient.timeoutSeconds'

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function shouldRefreshAppInstancesAfterInvokeError(error: unknown) {
  const message = getErrorMessage(error).toLowerCase()
  return (
    message.includes('app instance not found') ||
    message.includes('rpc service registration not found')
  )
}

function appInstanceKey(instance: ServiceDebugAppInstance) {
  return `${instance.appName}:${instance.appInstanceId}`
}

function serviceKey(service: ServiceDebugServiceItem) {
  return `${service.serviceSkelName}:${service.schemaHash}`
}

function methodStorageKey(serviceSkelName: string, methodSkelName: string) {
  return `${storageParamsPrefix}${serviceSkelName}:${methodSkelName}`
}

function actorInfoStorageKey(actorSkelName: string) {
  return `${storageActorInfoPrefix}${actorSkelName}`
}

function formatResponse(response: ServiceDebugInvokeResponse | null) {
  if (response === null) {
    return 'No response yet.'
  }
  const body = parseJsonText(response.bodyJson)
  return typeof body === 'string' ? body : JSON.stringify(body, null, 2)
}

function parseJsonText(value: string) {
  if (value.trim() === '') {
    return null
  }
  try {
    return JSON.parse(value)
  } catch {
    return value
  }
}

function serviceDebugClientTimeoutMs(timeoutSeconds: number) {
  return timeoutSeconds * 1000 + serviceDebugClientTimeoutPaddingMs
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
    // Ignore private-mode or quota failures; the request itself already ran.
  }
}

function serviceClientPath(serviceSkelName?: string, methodSkelName?: string) {
  const base = '/debug/service-client'
  if (!serviceSkelName) {
    return base
  }
  const servicePath = `${base}/${encodeURIComponent(serviceSkelName)}`
  if (!methodSkelName) {
    return servicePath
  }
  return `${servicePath}/${encodeURIComponent(methodSkelName)}`
}

function selectedPathParts(pathname: string) {
  const prefix = '/debug/service-client'
  if (pathname !== prefix && !pathname.startsWith(`${prefix}/`)) {
    return { isCurrent: false, methodSkelName: null, serviceSkelName: null }
  }
  const parts = pathname
    .slice(prefix.length)
    .split('/')
    .filter(Boolean)
    .map((part) => decodeURIComponent(part))
  return {
    isCurrent: true,
    serviceSkelName: parts[0] ?? null,
    methodSkelName: parts[1] ?? null,
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

export function ServiceClientPage() {
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const pathSelection = React.useMemo(
    () => selectedPathParts(pathname),
    [pathname],
  )
  const [appInstances, setAppInstances] = React.useState<
    Array<ServiceDebugAppInstance>
  >([])
  const [services, setServices] = React.useState<
    Array<ServiceDebugServiceItem>
  >([])
  const [serviceQuery, setServiceQuery] = React.useState('')
  const [serviceSelectOpen, setServiceSelectOpen] = React.useState(false)
  const serviceSearchInputRef = React.useRef<HTMLInputElement>(null)
  const [methods, setMethods] = React.useState<Array<ServiceDebugMethodItem>>(
    [],
  )
  const [actors, setActors] = React.useState<Array<ServiceDebugActorItem>>([])
  const [selectedAppInstance, setSelectedAppInstance] =
    React.useState<ServiceDebugAppInstance | null>(null)
  const [selectedService, setSelectedService] =
    React.useState<ServiceDebugServiceItem | null>(null)
  const [selectedMethod, setSelectedMethod] =
    React.useState<ServiceDebugMethodItem | null>(null)
  const [selectedActorSkelName, setSelectedActorSkelName] =
    React.useState<string | null>(null)
  const [actorInfoJson, setActorInfoJson] = React.useState(defaultActorInfo)
  const [params, setParams] = React.useState(defaultParams)
  const [traceId, setTraceId] = React.useState('')
  const [spanId, setSpanId] = React.useState('')
  const [timeoutSeconds, setTimeoutSeconds] = React.useState(
    () => readStorage(storageTimeoutSecondsKey) ?? defaultTimeoutSeconds,
  )
  const [loadingServices, setLoadingServices] = React.useState(true)
  const [loadingAppInstances, setLoadingAppInstances] = React.useState(false)
  const [loadingMethods, setLoadingMethods] = React.useState(false)
  const [loadingDefaultRequest, setLoadingDefaultRequest] =
    React.useState(false)
  const [invoking, setInvoking] = React.useState(false)
  const [response, setResponse] =
    React.useState<ServiceDebugInvokeResponse | null>(null)
  const responseText = React.useMemo(() => formatResponse(response), [response])

  const selectedAppInstanceKey = selectedAppInstance
    ? appInstanceKey(selectedAppInstance)
    : autoAppInstanceValue
  const selectedServiceKey = selectedService ? serviceKey(selectedService) : null
  const selectedMethodKey = selectedMethod?.skelName ?? null
  const filteredServices = React.useMemo(() => {
    const keyword = serviceQuery.trim().toLowerCase()
    if (keyword === '') {
      return services
    }
    return services.filter((item) =>
      item.serviceSkelName.toLowerCase().includes(keyword),
    )
  }, [serviceQuery, services])
  const selectedMethodBelongsToService =
    selectedMethod !== null &&
    methods.some((method) => method.skelName === selectedMethod.skelName)

  const focusServiceSearchInput = React.useCallback(() => {
    window.requestAnimationFrame(() => {
      serviceSearchInputRef.current?.focus()
    })
  }, [])

  React.useEffect(() => {
    if (!serviceSelectOpen) {
      return
    }
    focusServiceSearchInput()
  }, [focusServiceSearchInput, serviceQuery, serviceSelectOpen])

  const navigateToSelection = React.useCallback(
    (serviceSkelName?: string, methodSkelName?: string) => {
      void navigate({ to: serviceClientPath(serviceSkelName, methodSkelName) })
    },
    [navigate],
  )

  const chooseActor = React.useCallback(
    (actorSkelName: string | null, nextActors = actors) => {
      setSelectedActorSkelName(actorSkelName)
      if (actorSkelName === null) {
        setActorInfoJson(defaultActorInfo)
        return
      }
      const actor = nextActors.find((item) => item.skelName === actorSkelName)
      setActorInfoJson(
        readStorage(actorInfoStorageKey(actorSkelName)) ??
          actor?.actorInfoJson ??
          defaultActorInfo,
      )
    },
    [actors],
  )

  const loadDefaultInvokeRequest = React.useCallback(async () => {
    if (
      selectedService === null ||
      selectedMethod === null ||
      !selectedMethodBelongsToService
    ) {
      setActors([])
      setSelectedActorSkelName(null)
      setActorInfoJson(defaultActorInfo)
      setTraceId('')
      setSpanId('')
      setParams(defaultParams)
      return
    }

    setLoadingDefaultRequest(true)
    try {
      const nextRequest =
        await serviceDebugService.buildDefaultInvokeRequest({
          serviceSkelName: selectedService.serviceSkelName,
          schemaHash: selectedService.schemaHash,
          methodSkelName: selectedMethod.skelName,
        })
      setActors(nextRequest.actors)
      setTraceId(nextRequest.traceId)
      setSpanId(nextRequest.spanId)
      setParams(
        readStorage(
          methodStorageKey(selectedService.serviceSkelName, selectedMethod.skelName),
        ) ?? nextRequest.paramsJson,
      )

      const preferredActorSkelName = readStorage(storageActorSkelNameKey)
      const defaultActorSkelName = nextRequest.actorSkelName
      const nextActorSkelName =
        preferredActorSkelName &&
        nextRequest.actors.some(
          (actor) => actor.skelName === preferredActorSkelName,
        )
          ? preferredActorSkelName
          : defaultActorSkelName
      if (nextActorSkelName === null) {
        setSelectedActorSkelName(null)
        setActorInfoJson(defaultActorInfo)
      } else {
        setSelectedActorSkelName(nextActorSkelName)
        const actor = nextRequest.actors.find(
          (item) => item.skelName === nextActorSkelName,
        )
        setActorInfoJson(
          readStorage(actorInfoStorageKey(nextActorSkelName)) ??
            actor?.actorInfoJson ??
            nextRequest.actorInfoJson,
        )
      }
      setResponse(null)
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoadingDefaultRequest(false)
    }
  }, [selectedMethod, selectedMethodBelongsToService, selectedService])

  const resetRequest = React.useCallback(() => {
    void loadDefaultInvokeRequest()
  }, [loadDefaultInvokeRequest])

  React.useEffect(() => {
    let ignore = false
    setLoadingServices(true)
    void serviceDebugService
      .listServices(null)
      .then((nextServices) => {
        if (ignore) {
          return
        }
        setServices(nextServices)
      })
      .catch((error: unknown) => {
        if (!ignore) {
          toast.error(getErrorMessage(error))
        }
      })
      .finally(() => {
        if (!ignore) {
          setLoadingServices(false)
        }
      })
    return () => {
      ignore = true
    }
  }, [])

  React.useEffect(() => {
    if (services.length === 0) {
      setSelectedService(null)
      return
    }
    const preferredServiceSkelName =
      pathSelection.serviceSkelName ?? readStorage(storageServiceSkelNameKey)
    const nextService =
      services.find(
        (item) => item.serviceSkelName === preferredServiceSkelName,
      ) ??
      selectedService ??
      services[0]
    if (!services.some((item) => serviceKey(item) === serviceKey(nextService))) {
      setSelectedService(services[0])
      return
    }
    setSelectedService(nextService)
  }, [pathSelection.serviceSkelName, selectedService, services])

  React.useEffect(() => {
    if (selectedService === null) {
      setMethods([])
      setSelectedMethod(null)
      setAppInstances([])
      setSelectedAppInstance(null)
      return
    }

    let ignore = false
    setLoadingMethods(true)
    setLoadingAppInstances(true)
    setMethods([])
    setSelectedMethod(null)
    setAppInstances([])
    setSelectedAppInstance(null)

    void serviceDebugService
      .listMethods({
        serviceSkelName: selectedService.serviceSkelName,
        schemaHash: selectedService.schemaHash,
      })
      .then((nextMethods) => {
        if (ignore) {
          return
        }
        setMethods(nextMethods)
        const preferredMethodSkelName =
          pathSelection.methodSkelName ?? readStorage(storageMethodSkelNameKey)
        const nextMethod =
          nextMethods.find(
            (item) => item.skelName === preferredMethodSkelName,
          ) ?? nextMethods[0] ?? null
        setSelectedMethod(nextMethod)
        if (
          pathSelection.isCurrent &&
          nextMethod &&
          pathSelection.serviceSkelName !== selectedService.serviceSkelName
        ) {
          navigateToSelection(selectedService.serviceSkelName, nextMethod.skelName)
        }
      })
      .catch((error: unknown) => {
        if (!ignore) {
          toast.error(getErrorMessage(error))
        }
      })
      .finally(() => {
        if (!ignore) {
          setLoadingMethods(false)
        }
      })

    void serviceDebugService
      .listServiceAppInstances({
        serviceSkelName: selectedService.serviceSkelName,
        schemaHash: selectedService.schemaHash,
      })
      .then((nextAppInstances) => {
        if (ignore) {
          return
        }
        setAppInstances(nextAppInstances)
      })
      .catch((error: unknown) => {
        if (!ignore) {
          toast.error(getErrorMessage(error))
        }
      })
      .finally(() => {
        if (!ignore) {
          setLoadingAppInstances(false)
        }
      })

    return () => {
      ignore = true
    }
  }, [
    navigateToSelection,
    pathSelection.isCurrent,
    pathSelection.methodSkelName,
    pathSelection.serviceSkelName,
    selectedService,
  ])

  React.useEffect(() => {
    if (pathSelection.isCurrent && selectedService && selectedMethod) {
      writeStorage(
        storageServiceSkelNameKey,
        selectedService.serviceSkelName,
      )
      writeStorage(storageMethodSkelNameKey, selectedMethod.skelName)
      navigateToSelection(selectedService.serviceSkelName, selectedMethod.skelName)
    }
  }, [
    navigateToSelection,
    pathSelection.isCurrent,
    selectedMethod,
    selectedService,
  ])

  React.useEffect(() => {
    void loadDefaultInvokeRequest()
  }, [loadDefaultInvokeRequest])

  const refreshAppInstances = React.useCallback(async () => {
    if (selectedService === null) {
      return
    }
    setLoadingAppInstances(true)
    try {
      const nextAppInstances =
        await serviceDebugService.listServiceAppInstances({
          serviceSkelName: selectedService.serviceSkelName,
          schemaHash: selectedService.schemaHash,
        })
      setAppInstances(nextAppInstances)
      setSelectedAppInstance((current) => {
        if (
          current !== null &&
          nextAppInstances.some(
            (item) => appInstanceKey(item) === appInstanceKey(current),
          )
        ) {
          return current
        }
        return null
      })
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoadingAppInstances(false)
    }
  }, [selectedService])

  const invokeService = React.useCallback(async () => {
    if (selectedService === null || selectedMethod === null) {
      return
    }
    const parsedTimeoutSeconds = Number(timeoutSeconds.trim())
    if (!Number.isInteger(parsedTimeoutSeconds) || parsedTimeoutSeconds <= 0) {
      toast.error('Timeout must be a positive integer seconds value')
      return
    }
    setInvoking(true)
    try {
      const nextResponse = await serviceDebugService.invokeService(
        {
          request: {
            appName: selectedAppInstance?.appName ?? null,
            appInstanceId: selectedAppInstance?.appInstanceId ?? null,
            serviceSkelName: selectedService.serviceSkelName,
            schemaHash: selectedService.schemaHash,
            methodSkelName: selectedMethod.skelName,
            paramsJson: params,
            timeoutSeconds: parsedTimeoutSeconds,
            traceId: traceId.trim() === '' ? null : traceId.trim(),
            spanId: spanId.trim() === '' ? null : spanId.trim(),
            actorSkelName:
              selectedActorSkelName === null ||
              selectedActorSkelName.trim() === ''
                ? null
                : selectedActorSkelName.trim(),
            actorInfoJson,
          },
        },
        { timeoutMs: serviceDebugClientTimeoutMs(parsedTimeoutSeconds) },
      )
      setResponse(nextResponse)
      writeStorage(
        methodStorageKey(selectedService.serviceSkelName, selectedMethod.skelName),
        params,
      )
      writeStorage(storageTimeoutSecondsKey, timeoutSeconds)
      if (selectedActorSkelName !== null && selectedActorSkelName.trim() !== '') {
        writeStorage(storageActorSkelNameKey, selectedActorSkelName)
        writeStorage(actorInfoStorageKey(selectedActorSkelName), actorInfoJson)
      }
    } catch (error) {
      toast.error(getErrorMessage(error))
      if (shouldRefreshAppInstancesAfterInvokeError(error)) {
        void refreshAppInstances()
      }
    } finally {
      setInvoking(false)
    }
  }, [
    actorInfoJson,
    params,
    selectedActorSkelName,
    selectedAppInstance,
    selectedMethod,
    selectedService,
    refreshAppInstances,
    spanId,
    timeoutSeconds,
    traceId,
  ])

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
        <div className="grid gap-3 lg:grid-cols-[minmax(260px,1.2fr)_minmax(220px,1fr)_minmax(220px,1fr)_auto]">
          <label className="grid min-w-0 gap-1.5">
            <span className="text-xs font-medium text-muted-foreground">
              Service
            </span>
            <Select
              open={serviceSelectOpen}
              onOpenChange={setServiceSelectOpen}
              value={selectedServiceKey}
              onValueChange={(value) => {
                const nextService = services.find(
                  (item) => serviceKey(item) === value,
                )
                if (nextService) {
                  setSelectedService(nextService)
                  setMethods([])
                  setSelectedMethod(null)
                  setAppInstances([])
                  setSelectedAppInstance(null)
                  setServiceSelectOpen(false)
                  navigateToSelection(nextService.serviceSkelName)
                }
              }}
              disabled={loadingServices || services.length === 0}
            >
              <SelectTrigger className="h-auto w-full rounded-lg border-transparent bg-primary/[0.05] px-3 py-2.5 hover:bg-primary/[0.07] focus-visible:border-primary/30">
                <SelectCardText
                  title={selectedService?.serviceSkelName}
                  description={selectedService?.schemaHash}
                  placeholder={
                    loadingServices ? 'Loading services' : 'Select service'
                  }
                />
              </SelectTrigger>
              <SelectContent
                align="start"
                alignItemWithTrigger={false}
                className="p-0"
                onPointerEnter={focusServiceSearchInput}
                onPointerMove={focusServiceSearchInput}
                header={
                  <div className="sticky top-0 z-10 border-b border-border bg-popover p-2">
                    <div className="relative">
                      <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        ref={serviceSearchInputRef}
                        value={serviceQuery}
                        className="h-8 pl-8"
                        placeholder="Search service"
                        onChange={(event) => setServiceQuery(event.target.value)}
                        onKeyDownCapture={(event) => event.stopPropagation()}
                        onKeyDown={(event) => event.stopPropagation()}
                      />
                    </div>
                  </div>
                }
              >
                {filteredServices.length === 0 ? (
                  <div className="px-3 py-4 text-sm text-muted-foreground">
                    No services found.
                  </div>
                ) : (
                  filteredServices.map((item) => (
                    <SelectItem
                      key={serviceKey(item)}
                      value={serviceKey(item)}
                      className={cn(
                        'rounded-lg border px-3 py-2.5 pr-8 hover:border-primary/30 hover:bg-primary/[0.06] focus:border-primary/30 focus:bg-primary/[0.06]',
                        selectedServiceKey === serviceKey(item)
                          ? 'border-primary/30 bg-primary/[0.06]'
                          : 'border-transparent',
                      )}
                    >
                      <SelectCardItem
                        title={item.serviceSkelName}
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
              Method
            </span>
            <Select
              value={selectedMethodKey}
              onValueChange={(value) => {
                const nextMethod = methods.find(
                  (item) => item.skelName === value,
                )
                if (nextMethod && selectedService) {
                  setSelectedMethod(nextMethod)
                  navigateToSelection(
                    selectedService.serviceSkelName,
                    nextMethod.skelName,
                  )
                }
              }}
              disabled={
                selectedService === null || loadingMethods || methods.length === 0
              }
            >
              <SelectTrigger className="h-auto w-full rounded-lg border-transparent bg-primary/[0.05] px-3 py-2.5 hover:bg-primary/[0.07] focus-visible:border-primary/30">
                <SelectCardText
                  title={selectedMethod?.skelName}
                  description={
                    selectedMethod ? selectedMethod.resultType || 'void' : undefined
                  }
                  placeholder={
                    loadingMethods ? 'Loading methods' : 'Select method'
                  }
                />
              </SelectTrigger>
              <SelectContent
                align="start"
                alignItemWithTrigger={false}
                className="p-0"
              >
                {methods.map((item) => (
                  <SelectItem
                    key={item.skelName}
                    value={item.skelName}
                    className={cn(
                      'rounded-lg border px-3 py-2.5 pr-8 focus:border-primary/30 focus:bg-primary/[0.06]',
                      selectedMethodKey === item.skelName
                        ? 'border-primary/30 bg-primary/[0.06]'
                        : 'border-transparent',
                    )}
                  >
                    <SelectCardItem
                      title={item.skelName}
                      description={item.resultType || 'void'}
                    />
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>

          <label className="grid min-w-0 gap-1.5">
            <span className="text-xs font-medium text-muted-foreground">
              App Instance
            </span>
            <Select
              value={selectedAppInstanceKey}
              onValueChange={(value) => {
                if (value === autoAppInstanceValue) {
                  setSelectedAppInstance(null)
                  return
                }
                const nextAppInstance = appInstances.find(
                  (item) => appInstanceKey(item) === value,
                )
                setSelectedAppInstance(nextAppInstance ?? null)
              }}
              disabled={selectedService === null || loadingAppInstances}
            >
              <SelectTrigger className="h-auto w-full rounded-lg border-transparent bg-primary/[0.05] px-3 py-2.5 hover:bg-primary/[0.07] focus-visible:border-primary/30">
                <SelectCardText
                  title={
                    selectedAppInstance
                      ? `${selectedAppInstance.appName}@${selectedAppInstance.appVersion}`
                      : 'Auto select'
                  }
                  description={
                    selectedAppInstance?.appInstanceId ??
                    'Uses the only matching instance'
                  }
                  placeholder={
                    loadingAppInstances
                      ? 'Loading app instances'
                      : 'Select app instance'
                  }
                />
              </SelectTrigger>
              <SelectContent
                align="start"
                alignItemWithTrigger={false}
                className="p-0"
              >
                <SelectItem
                  value={autoAppInstanceValue}
                  className={cn(
                    'rounded-lg border px-3 py-2.5 pr-8 focus:border-primary/30 focus:bg-primary/[0.06]',
                    selectedAppInstance === null
                      ? 'border-primary/30 bg-primary/[0.06]'
                      : 'border-transparent',
                  )}
                >
                  <SelectCardItem
                    title="Auto select"
                    description="Use the only matching app instance"
                  />
                </SelectItem>
                {appInstances.map((item) => (
                  <SelectItem
                    key={appInstanceKey(item)}
                    value={appInstanceKey(item)}
                    className={cn(
                      'rounded-lg border px-3 py-2.5 pr-8 focus:border-primary/30 focus:bg-primary/[0.06]',
                      selectedAppInstanceKey === appInstanceKey(item)
                        ? 'border-primary/30 bg-primary/[0.06]'
                        : 'border-transparent',
                    )}
                  >
                    <SelectCardItem
                      title={`${item.appName}@${item.appVersion}`}
                      description={item.appInstanceId}
                    />
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>

          <div />
        </div>
      </section>

      <main className="grid min-h-0 flex-1 grid-rows-[minmax(0,1fr)_minmax(0,1fr)]">
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
          <div className="grid min-h-0 flex-1 gap-3 p-3 lg:grid-cols-[minmax(280px,360px)_minmax(0,1fr)]">
            <section className="flex min-h-0 flex-col">
              <div className="grid shrink-0 gap-3">
                <h3 className="text-sm font-semibold text-foreground">Actor</h3>
                <label className="grid min-w-0">
                  <Select
                    value={selectedActorSkelName}
                    onValueChange={(value) => chooseActor(value)}
                    disabled={actors.length === 0}
                  >
                    <SelectTrigger className="h-9 w-full">
                      <span className="truncate text-sm">
                        {selectedActorSkelName || 'No actor required'}
                      </span>
                    </SelectTrigger>
                    <SelectContent
                      align="start"
                      alignItemWithTrigger={false}
                      className="p-0"
                    >
                      {actors.map((item) => (
                        <SelectItem key={item.skelName} value={item.skelName}>
                          {item.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </label>
                <label className="grid min-w-0 gap-1.5">
                  <span className="text-xs font-medium text-muted-foreground">
                    Timeout (s)
                  </span>
                  <Input
                    type="number"
                    min={1}
                    step={1}
                    value={timeoutSeconds}
                    className="h-9"
                    onChange={(event) => setTimeoutSeconds(event.target.value)}
                  />
                </label>
              </div>

              <div className="min-h-0 flex-1 pt-3">
                {actors.length > 0 ? (
                  <label className="flex h-full min-h-0 flex-col">
                    <CodeMirror
                      value={actorInfoJson}
                      extensions={jsonExtensions}
                      onChange={setActorInfoJson}
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
                      className="min-h-0 flex-1 overflow-hidden rounded-md border border-input bg-background text-[13px] [&_.cm-editor]:h-full [&_.cm-scroller]:h-full"
                      theme="light"
                    />
                  </label>
                ) : null}
              </div>

              <div className="flex shrink-0 items-center gap-2 pt-3">
                <Button
                  type="button"
                  variant="outline"
                  className="h-10 min-w-28"
                  disabled={loadingDefaultRequest}
                  onClick={resetRequest}
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
                    selectedService === null ||
                    selectedMethod === null ||
                    invoking ||
                    loadingDefaultRequest
                  }
                  onClick={() => void invokeService()}
                >
                  {invoking ? <Loader2 className="animate-spin" /> : <Send />}
                  Send
                </Button>
              </div>
            </section>

            <div className="flex min-h-0 flex-col gap-1.5">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-foreground">
                  Params
                </span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-7 px-2 text-xs"
                  onClick={() => void copyText(params, 'Params copied')}
                >
                  <Copy />
                  Copy
                </Button>
              </div>
              <CodeMirror
                value={params}
                extensions={jsonExtensions}
                onChange={setParams}
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
                className="min-h-0 flex-1 overflow-hidden rounded-md border border-input bg-background text-[13px] [&_.cm-editor]:h-full [&_.cm-scroller]:h-full"
                theme="light"
              />
            </div>
          </div>
        </section>

        <section className="flex min-h-0 flex-col">
          <div className="flex h-11 shrink-0 items-center gap-2 border-b border-border px-4">
            <h2 className="text-sm font-semibold text-foreground">
              Response
            </h2>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-7 px-2 text-xs"
              disabled={response === null}
              onClick={() => void copyText(responseText, 'Response copied')}
            >
              <Copy />
              Copy
            </Button>
          </div>
          <div className="min-h-0 flex-1 p-3">
            <pre className="scrollbar-reserved h-full overflow-auto rounded-md border border-input bg-muted/20 p-3 font-mono text-[13px] leading-5 text-muted-foreground">
              {responseText}
            </pre>
          </div>
        </section>
      </main>
    </div>
  )
}
