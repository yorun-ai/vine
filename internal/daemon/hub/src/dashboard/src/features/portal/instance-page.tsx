import { ListDetailLayout } from '@/components/ui/list-detail-layout'
import { SearchInput } from '@/components/ui/search-input'
import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { Boxes, RefreshCw, Rocket } from 'lucide-react'
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
import { Skeleton } from '@/components/ui/skeleton'
import { vrpcClient } from '@/config/vrpc-client'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import { createServiceDebugApiService } from '@/skeled/admin'
import type { ServiceDebugPortalInstance } from '@/skeled/admin'

import { filterPortalInstances } from './instance-filter'

const serviceDebugService = createServiceDebugApiService(vrpcClient)
const PORTAL_INSTANCE_LIST_DEFAULT_WIDTH = 352

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function portalInstancePath(instanceId: string) {
  return `/portal/instance/${encodeURIComponent(instanceId)}`
}

function portalInstanceListItemDomId(instanceId: string) {
  return `portal-instance-list-item:${encodeURIComponent(instanceId)}`
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
  const match = pathname.match(/^\/portal\/instance\/(.+)$/)
  return match ? decodeURIComponent(match[1]) : null
}

function isPortalInstancePath(pathname: string) {
  return (
    pathname === '/portal/instance' || pathname.startsWith('/portal/instance/')
  )
}

function PortalInstanceListSkeleton() {
  return (
    <div className="space-y-1">
      {Array.from({ length: 4 }).map((_, index) => (
        <div key={index} className="rounded-lg px-3 py-2.5">
          <Skeleton className="h-4 w-40" />
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

export function PortalInstancePage() {
  const { t } = useLocale()
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const [items, setItems] = React.useState<Array<ServiceDebugPortalInstance>>([])
  const [query, setQuery] = React.useState('')
  const [loading, setLoading] = React.useState(true)
  const [selectedInstanceId, setSelectedInstanceId] = React.useState<
    string | null
  >(() => selectedInstanceIdFromPath(window.location.pathname))

  const loadItems = React.useCallback(async () => {
    setLoading(true)
    try {
      setItems(await serviceDebugService.listPortalInstances(null))
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void loadItems()
  }, [loadItems])

  const filteredItems = React.useMemo(
    () => filterPortalInstances(items, query),
    [items, query],
  )

  const selectedItem = React.useMemo(() => {
    const item = filteredItems.find(
      (current) => current.instanceId === selectedInstanceId,
    )
    if (item) {
      return item
    }
    return filteredItems[0]
  }, [filteredItems, selectedInstanceId])

  const selectItem = React.useCallback(
    (instanceId: string, replace = false) => {
      setSelectedInstanceId(instanceId)
      void navigate({ replace, to: portalInstancePath(instanceId) })
    },
    [navigate],
  )

  React.useEffect(() => {
    if (!isPortalInstancePath(pathname)) {
      return
    }
    setSelectedInstanceId(selectedInstanceIdFromPath(pathname))
  }, [pathname])

  React.useEffect(() => {
    if (!isPortalInstancePath(pathname)) {
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
        .getElementById(portalInstanceListItemDomId(selectedInstanceId))
        ?.scrollIntoView({
          block: 'nearest',
          inline: 'nearest',
        })
    })
  }, [filteredItems, selectedInstanceId])

  return (
    <ListDetailLayout
      defaultWidth={PORTAL_INSTANCE_LIST_DEFAULT_WIDTH}
      resizeLabel={t('portalInstance.resizeList')}
      listHeader={
        <>
          <div className="relative">
            <SearchInput
              value={query}
              placeholder={t('portalInstance.searchPlaceholder')}
              onValueChange={setQuery}
            />
          </div>
          <div className="mt-3 flex items-center gap-2">
            <div className="ml-auto flex shrink-0 items-center gap-2">
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
        </>
      }
      listFooter={t('portalInstance.itemCount').replace(
        '{count}',
        String(items.length),
      )}
      list={
        loading ? (
          <PortalInstanceListSkeleton />
        ) : filteredItems.length === 0 ? (
          <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
            {items.length === 0
              ? t('portalInstance.empty')
              : t('portalInstance.noMatch')}
          </div>
        ) : (
          <div className="space-y-1">
            {filteredItems.map((item) => (
              <a
                key={item.instanceId}
                id={portalInstanceListItemDomId(item.instanceId)}
                href={portalInstancePath(item.instanceId)}
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
                  <span className="truncate font-mono text-sm font-semibold">
                    {item.instanceId}
                  </span>
                  <Badge variant="outline">
                    {item.version || t('common.noVersion')}
                  </Badge>
                </div>
              </a>
            ))}
          </div>
        )
      }
    >
      {loading ? (
        <div className="space-y-4 p-6">
          <Skeleton className="h-8 w-56" />
          <Skeleton className="h-24 w-full" />
        </div>
      ) : selectedItem ? (
        <div className="flex h-full min-h-0 flex-col">
          <div className="border-b border-border/70 px-6 py-5">
            <div className="flex min-w-0 items-center gap-2">
              <Rocket className="size-4 shrink-0 text-primary" />
              <h1 className="truncate font-mono text-xl font-semibold">
                {selectedItem.instanceId}
              </h1>
              <Badge variant="outline">
                {selectedItem.version || t('common.noVersion')}
              </Badge>
            </div>
          </div>

          <div className="scrollbar-reserved min-h-0 flex-1 overflow-y-auto p-6 pr-4">
            <section className="grid gap-3">
              <div className="flex items-center gap-2">
                <Boxes className="size-4 text-primary" />
                <h3 className="text-sm font-semibold">
                  {t('portalInstance.instanceInfo')}
                </h3>
              </div>
              <div className="grid gap-2 lg:grid-cols-2">
                <DetailRow
                  label={t('portalInstance.instanceId')}
                  value={selectedItem.instanceId}
                />
                <DetailRow
                  label={t('portalInstance.version')}
                  value={selectedItem.version || t('common.none')}
                />
              </div>
            </section>
          </div>
        </div>
      ) : (
        <Empty className="h-full rounded-none border-0">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Rocket />
            </EmptyMedia>
            <EmptyTitle>
              {items.length === 0
                ? t('portalInstance.empty')
                : t('portalInstance.noMatch')}
            </EmptyTitle>
            <EmptyDescription>
              {t('portalInstance.emptyDescription')}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
    </ListDetailLayout>
  )
}
