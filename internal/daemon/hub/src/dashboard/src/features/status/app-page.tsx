import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import {
  Activity,
  Boxes,
  Clock3,
  Globe2,
  Radio,
  RefreshCw,
  Search,
  Server,
} from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import {
  ResizableListHandle,
  useReservedScrollbar,
  useResizableListPanel,
} from '@/components/ui/resizable-list-panel'
import { Skeleton } from '@/components/ui/skeleton'
import { vrpcClient } from '@/config/vrpc-client'
import { cn } from '@/lib/utils'
import { createAppStatusService } from '@/skeled'
import { useLocale } from '@/i18n'
import type {
  AppStatusView,
  EventListenerRegistration,
  ServiceHandlerRegistration,
  TaskRunnerRegistration,
  WebHandlerRegistration,
} from '@/skeled'

import {
  skeletonEventHref,
  skeletonServiceHref,
  skeletonTaskHref,
  skeletonWebHref,
} from '../skeleton/model'

const appStatusService = createAppStatusService(vrpcClient)
const APP_STATUS_LIST_DEFAULT_WIDTH = 352
const APP_STATUS_LIST_WIDTH_STORAGE_KEY = 'vinehub_status_app_list_width'

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function appStatusPath(instanceId: string) {
  return `/status/app/${encodeURIComponent(instanceId)}`
}

function appStatusListItemDomId(instanceId: string) {
  return `app-status-list-item:${encodeURIComponent(instanceId)}`
}

function shouldUseBrowserNavigation(
  event: React.MouseEvent<HTMLAnchorElement>,
) {
  return (
    event.defaultPrevented ||
    event.button !== 0 ||
    event.metaKey ||
    event.altKey ||
    event.ctrlKey ||
    event.shiftKey
  )
}

function selectedInstanceIdFromPath(pathname: string) {
  const match = pathname.match(/^\/status\/app\/(.+)$/)
  return match ? decodeURIComponent(match[1]) : null
}

function isAppStatusPath(pathname: string) {
  return pathname === '/status/app' || pathname.startsWith('/status/app/')
}

function StatusListSkeleton() {
  return (
    <div className="space-y-1">
      {Array.from({ length: 5 }).map((_, index) => (
        <div key={index} className="rounded-lg px-3 py-2.5">
          <Skeleton className="mb-2 h-4 w-32" />
          <Skeleton className="h-3 w-44" />
        </div>
      ))}
    </div>
  )
}

function DetailRow({
  label,
  value,
}: {
  label: string
  value: React.ReactNode
}) {
  return (
    <div className="grid gap-1 rounded-lg border bg-background px-3 py-2.5">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="min-w-0 break-all font-mono text-sm">{value}</div>
    </div>
  )
}

function EmptySection({ label }: { label: string }) {
  const { t } = useLocale()
  return (
    <div className="py-2 text-sm text-muted-foreground">
      {t('common.emptyPrefix')} {label}
    </div>
  )
}

function CapabilityShell({
  children,
  count,
  icon,
  title,
}: {
  children: React.ReactNode
  count: number
  icon: React.ReactNode
  title: string
}) {
  return (
    <section className="grid gap-3">
      <div className="flex items-center gap-2">
        {icon}
        <h3 className="text-sm font-semibold">{title}</h3>
        <Badge variant="outline">{count}</Badge>
      </div>
      {children}
    </section>
  )
}

function HandlerRow({
  endpoint,
  href,
  onNavigate,
  schemaHash,
  skelName,
}: {
  endpoint: string
  href: string
  onNavigate: (href: string) => void
  schemaHash: string
  skelName: string
}) {
  return (
    <div className="grid gap-1 rounded-lg border bg-background px-3 py-2.5">
      <div className="flex min-w-0 items-center gap-2">
        <a
          href={href}
          onClick={(event) => {
            if (shouldUseBrowserNavigation(event)) {
              return
            }
            event.preventDefault()
            onNavigate(href)
          }}
          className="truncate text-sm font-semibold transition-colors hover:text-primary hover:underline"
        >
          {skelName}
        </a>
        <Badge variant="outline" className="font-mono">
          {schemaHash}
        </Badge>
      </div>
      <div className="truncate font-mono text-xs text-muted-foreground">
        {endpoint}
      </div>
    </div>
  )
}

function ListenerRow({
  concurrency,
  href,
  noRetry,
  onNavigate,
  schemaHash,
  skelName,
  timeoutMs,
}: {
  concurrency: number
  href: string
  noRetry: boolean
  onNavigate: (href: string) => void
  schemaHash: string
  skelName: string
  timeoutMs: number
}) {
  return (
    <div className="grid gap-2 rounded-lg border bg-background px-3 py-2.5">
      <div className="flex min-w-0 items-center gap-2">
        <a
          href={href}
          onClick={(event) => {
            if (shouldUseBrowserNavigation(event)) {
              return
            }
            event.preventDefault()
            onNavigate(href)
          }}
          className="truncate text-sm font-semibold transition-colors hover:text-primary hover:underline"
        >
          {skelName}
        </a>
        <Badge variant="outline" className="font-mono">
          {schemaHash}
        </Badge>
      </div>
      <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
        <Badge variant="secondary">timeout {timeoutMs}ms</Badge>
        <Badge variant="secondary">concurrency {concurrency}</Badge>
        {noRetry ? <Badge variant="secondary">no retry</Badge> : null}
      </div>
    </div>
  )
}

function TaskRunnerRow({
  item,
  onNavigate,
}: {
  item: TaskRunnerRegistration
  onNavigate: (href: string) => void
}) {
  return (
    <div className="grid gap-2 rounded-lg border bg-background px-3 py-2.5">
      <div className="flex min-w-0 items-center gap-2">
        <a
          href={skeletonTaskHref(item.taskSkelName, item.schemaHash)}
          onClick={(event) => {
            if (shouldUseBrowserNavigation(event)) {
              return
            }
            event.preventDefault()
            onNavigate(skeletonTaskHref(item.taskSkelName, item.schemaHash))
          }}
          className="truncate text-sm font-semibold transition-colors hover:text-primary hover:underline"
        >
          {item.taskSkelName}
        </a>
        <Badge variant="outline" className="font-mono">
          {item.schemaHash}
        </Badge>
      </div>
      <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
        <Badge variant="secondary">timeout {item.timeoutMs}ms</Badge>
        <Badge variant="secondary">concurrency {item.concurrency}</Badge>
        {item.noRetry ? <Badge variant="secondary">no retry</Badge> : null}
      </div>
      {item.cronSchedulers.length > 0 ? (
        <div className="grid gap-1 border-t border-border/60 pt-2">
          {item.cronSchedulers.map((scheduler) => (
            <div
              key={`${scheduler.triggerSkelName}:${scheduler.cronExpr}`}
              className="flex min-w-0 items-center gap-2 text-xs"
            >
              <Badge variant="outline" className="shrink-0 font-mono">
                Cron
              </Badge>
              <span className="truncate font-mono text-muted-foreground">
                {scheduler.triggerSkelName} / {scheduler.cronExpr}
              </span>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  )
}

function ServiceHandlers({
  items,
  onNavigate,
}: {
  items: Array<ServiceHandlerRegistration>
  onNavigate: (href: string) => void
}) {
  const { t } = useLocale()
  const title = t('statusApp.rpcService')
  return (
    <CapabilityShell
      title={title}
      count={items.length}
      icon={<Server className="size-4 text-primary" />}
    >
      {items.length === 0 ? (
        <EmptySection label={title} />
      ) : (
        <div className="grid gap-2">
          {items.map((item) => (
            <HandlerRow
              key={`${item.serviceSkelName}:${item.schemaHash}:${item.endpoint}`}
              skelName={item.serviceSkelName}
              schemaHash={item.schemaHash}
              endpoint={item.endpoint}
              href={skeletonServiceHref(item.serviceSkelName, item.schemaHash)}
              onNavigate={onNavigate}
            />
          ))}
        </div>
      )}
    </CapabilityShell>
  )
}

function WebHandlers({
  items,
  onNavigate,
}: {
  items: Array<WebHandlerRegistration>
  onNavigate: (href: string) => void
}) {
  return (
    <CapabilityShell
      title="Web"
      count={items.length}
      icon={<Globe2 className="size-4 text-primary" />}
    >
      {items.length === 0 ? (
        <EmptySection label="Web" />
      ) : (
        <div className="grid gap-2">
          {items.map((item) => (
            <HandlerRow
              key={`${item.webSkelName}:${item.schemaHash}:${item.endpoint}`}
              skelName={item.webSkelName}
              schemaHash={item.schemaHash}
              endpoint={item.endpoint}
              href={skeletonWebHref(item.webSkelName, item.schemaHash)}
              onNavigate={onNavigate}
            />
          ))}
        </div>
      )}
    </CapabilityShell>
  )
}

function EventListeners({
  items,
  onNavigate,
}: {
  items: Array<EventListenerRegistration>
  onNavigate: (href: string) => void
}) {
  const { t } = useLocale()
  const title = t('statusApp.eventListener')
  return (
    <CapabilityShell
      title={title}
      count={items.length}
      icon={<Radio className="size-4 text-primary" />}
    >
      {items.length === 0 ? (
        <EmptySection label={title} />
      ) : (
        <div className="grid gap-2">
          {items.map((item) => (
            <ListenerRow
              key={`${item.eventSkelName}:${item.schemaHash}`}
              skelName={item.eventSkelName}
              schemaHash={item.schemaHash}
              href={skeletonEventHref(item.eventSkelName, item.schemaHash)}
              onNavigate={onNavigate}
              timeoutMs={item.timeoutMs}
              concurrency={item.concurrency}
              noRetry={item.noRetry}
            />
          ))}
        </div>
      )}
    </CapabilityShell>
  )
}

function TaskRunners({
  items,
  onNavigate,
}: {
  items: Array<TaskRunnerRegistration>
  onNavigate: (href: string) => void
}) {
  const { t } = useLocale()
  const title = t('statusApp.taskRunner')
  return (
    <CapabilityShell
      title={title}
      count={items.length}
      icon={<Clock3 className="size-4 text-primary" />}
    >
      {items.length === 0 ? (
        <EmptySection label={title} />
      ) : (
        <div className="grid gap-2">
          {items.map((item) => (
            <TaskRunnerRow
              key={`${item.taskSkelName}:${item.schemaHash}`}
              item={item}
              onNavigate={onNavigate}
            />
          ))}
        </div>
      )}
    </CapabilityShell>
  )
}

export function AppStatusPage() {
  const { t } = useLocale()
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const [items, setItems] = React.useState<Array<AppStatusView>>([])
  const [query, setQuery] = React.useState('')
  const [loading, setLoading] = React.useState(true)
  const listPanel = useResizableListPanel({
    defaultWidth: APP_STATUS_LIST_DEFAULT_WIDTH,
    storageKey: APP_STATUS_LIST_WIDTH_STORAGE_KEY,
  })
  const handleListScroll = useReservedScrollbar()
  const [selectedInstanceId, setSelectedInstanceId] = React.useState<
    string | null
  >(() => selectedInstanceIdFromPath(window.location.pathname))

  const loadItems = React.useCallback(async () => {
    setLoading(true)
    try {
      setItems(await appStatusService.list(null))
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void loadItems()
  }, [loadItems])

  const filteredItems = React.useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (keyword === '') {
      return items
    }
    return items.filter((item) => item.name.toLowerCase().includes(keyword))
  }, [items, query])

  const selectedItem = React.useMemo(() => {
    const item = filteredItems.find(
      (current) => current.instanceId === selectedInstanceId,
    )
    if (item) {
      return item
    }
    if (filteredItems.length === 0) {
      return undefined
    }
    return filteredItems[0]
  }, [filteredItems, selectedInstanceId])

  const selectItem = React.useCallback(
    (instanceId: string, replace = false) => {
      setSelectedInstanceId(instanceId)
      void navigate({ replace, to: appStatusPath(instanceId) })
    },
    [navigate],
  )

  const navigateToHref = React.useCallback(
    (href: string) => {
      void navigate({ to: href })
    },
    [navigate],
  )

  React.useEffect(() => {
    if (!isAppStatusPath(pathname)) {
      return
    }
    setSelectedInstanceId(selectedInstanceIdFromPath(pathname))
  }, [pathname])

  React.useEffect(() => {
    if (!isAppStatusPath(pathname)) {
      return
    }
    if (filteredItems.length === 0) {
      setSelectedInstanceId(null)
      return
    }
    if (
      !selectedInstanceId ||
      !filteredItems.some((item) => item.instanceId === selectedInstanceId)
    ) {
      selectItem(filteredItems[0].instanceId, true)
    }
  }, [filteredItems, pathname, selectItem, selectedInstanceId])

  React.useEffect(() => {
    if (!selectedInstanceId) {
      return
    }
    window.requestAnimationFrame(() => {
      document
        .getElementById(appStatusListItemDomId(selectedInstanceId))
        ?.scrollIntoView({
          block: 'nearest',
          inline: 'nearest',
        })
    })
  }, [filteredItems, selectedInstanceId])

  return (
    <section className="flex h-[calc(100dvh-3.5rem)] flex-col overflow-hidden bg-white">
      <div
        className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[var(--list-panel-width)_minmax(0,1fr)]"
        style={listPanel.gridStyle}
      >
        <aside className="relative flex min-h-0 flex-col border-b border-border/70 lg:border-r lg:border-b-0">
          <div className="grid gap-4 border-b border-border/70 p-4">
            <div className="relative w-full md:max-w-sm">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={query}
                className="pl-8"
                placeholder={t('statusApp.searchPlaceholder')}
                onChange={(event) => setQuery(event.target.value)}
              />
            </div>
            <div className="flex items-center justify-between gap-2">
              <div className="text-xs text-muted-foreground">
                {t('statusApp.itemCount').replace(
                  '{count}',
                  String(items.length),
                )}
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7"
                title={t('action.refreshList')}
                onClick={() => void loadItems()}
                disabled={loading}
              >
                <RefreshCw
                  className={cn('size-3.5', loading && 'animate-spin')}
                />
              </Button>
            </div>
          </div>

          <div
            className="scrollbar-reserved min-h-0 flex-1 overflow-auto py-2 pr-1 pl-2"
            onScroll={handleListScroll}
          >
            {loading ? (
              <StatusListSkeleton />
            ) : filteredItems.length === 0 ? (
              <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                {items.length === 0
                  ? t('statusApp.empty')
                  : t('statusApp.noMatch')}
              </div>
            ) : (
              <div className="space-y-1">
                {filteredItems.map((item) => (
                  <a
                    key={item.instanceId}
                    id={appStatusListItemDomId(item.instanceId)}
                    href={appStatusPath(item.instanceId)}
                    onClick={(event) => {
                      if (shouldUseBrowserNavigation(event)) {
                        return
                      }
                      event.preventDefault()
                      selectItem(item.instanceId)
                    }}
                    className={cn(
                      'grid w-full gap-1 rounded-lg border px-3 py-2.5 text-left transition-colors',
                      selectedItem?.instanceId === item.instanceId
                        ? 'border-primary/30 bg-primary/[0.06]'
                        : 'border-transparent hover:bg-primary/[0.05]',
                    )}
                  >
                    <div className="flex min-w-0 items-center justify-between gap-2">
                      <span className="truncate text-sm font-semibold">
                        {item.name}
                      </span>
                      <Badge variant="outline">
                        {item.version || t('common.noVersion')}
                      </Badge>
                    </div>
                    <div className="truncate font-mono text-xs text-muted-foreground">
                      {item.instanceId}
                    </div>
                  </a>
                ))}
              </div>
            )}
          </div>
          <ResizableListHandle
            defaultWidth={APP_STATUS_LIST_DEFAULT_WIDTH}
            label={t('statusApp.resizeList')}
            panel={listPanel}
          />
        </aside>

        <main className="min-h-0 overflow-hidden">
          {loading ? (
            <div className="space-y-4 p-6">
              <Skeleton className="h-8 w-56" />
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-24 w-full" />
            </div>
          ) : selectedItem ? (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b border-border/70 px-6 py-5">
                <div className="flex min-w-0 items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex min-w-0 items-center gap-2">
                      <Activity className="size-4 shrink-0 text-primary" />
                      <h1 className="truncate text-xl font-semibold">
                        {selectedItem.name}
                      </h1>
                      {selectedItem.version ? (
                        <Badge variant="outline">{selectedItem.version}</Badge>
                      ) : null}
                    </div>
                    <div className="mt-1 truncate font-mono text-sm text-muted-foreground">
                      {selectedItem.instanceId}
                    </div>
                  </div>
                </div>
              </div>

              <div className="scrollbar-reserved min-h-0 flex-1 overflow-y-auto p-6 pr-4">
                <div className="grid gap-6">
                  <section className="grid gap-3">
                    <div className="flex items-center gap-2">
                      <Boxes className="size-4 text-primary" />
                      <h3 className="text-sm font-semibold">
                        {t('statusApp.instanceInfo')}
                      </h3>
                    </div>
                    <div className="grid gap-2 lg:grid-cols-2">
                      <DetailRow
                        label={t('statusApp.instanceId')}
                        value={selectedItem.instanceId}
                      />
                      <DetailRow
                        label={t('statusApp.endpoint')}
                        value={selectedItem.endpoint || t('common.none')}
                      />
                      <DetailRow
                        label={t('statusApp.version')}
                        value={selectedItem.version || t('common.none')}
                      />
                    </div>
                  </section>

                  <ServiceHandlers
                    items={selectedItem.serviceHandlers}
                    onNavigate={navigateToHref}
                  />
                  <WebHandlers
                    items={selectedItem.webHandlers}
                    onNavigate={navigateToHref}
                  />
                  <EventListeners
                    items={selectedItem.eventListeners}
                    onNavigate={navigateToHref}
                  />
                  <TaskRunners
                    items={selectedItem.taskRunners}
                    onNavigate={navigateToHref}
                  />
                </div>
              </div>
            </div>
          ) : (
            <Empty className="h-full rounded-none border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Activity />
                </EmptyMedia>
                <EmptyTitle>
                  {items.length === 0
                    ? t('statusApp.empty')
                    : t('statusApp.noMatch')}
                </EmptyTitle>
                <EmptyDescription>
                  {t('statusApp.emptyDescription')}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          )}
        </main>
      </div>
    </section>
  )
}
