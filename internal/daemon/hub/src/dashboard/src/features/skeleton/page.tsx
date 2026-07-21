import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { ChevronDown, Loader2, RefreshCw, Search } from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  ResizableListHandle,
  useReservedScrollbar,
  useResizableListPanel,
} from '@/components/ui/resizable-list-panel'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Skeleton } from '@/components/ui/skeleton'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import type {
  SkeletonData,
  SkeletonServiceItem,
  SkeletonWebItem,
} from '@/skeled'

import { SkeletonItemBadges, SkeletonItemDetails } from './details'
import {
  buildTypeDefinitionIndex,
  displayItemName,
  getErrorMessage,
  itemRouteHash,
  itemValues,
  itemVersionKey,
  skeletonDomainHref,
  skeletonItemHref,
  skeletonConfigs,
  skeletonRouteConfig,
  skeletonService,
  splitDomainSkelName,
} from './model'
import type { SkeletonItem, SkeletonKind } from './model'

const skeletonItemsByKind = new Map<SkeletonKind, Array<SkeletonItem>>()
let cachedTypeDefinitions: Array<SkeletonData> | null = null
const SKELETON_LIST_WIDTH_STORAGE_KEY = 'vinehub_skeleton_list_width_v2'
const SKELETON_LIST_DEFAULT_WIDTH = 352

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

function skeletonListItemDomId(kind: SkeletonKind, skelName: string) {
  return `skeleton-list-item:${kind}:${skelName}`
}

function skeletonVersionDomId(
  kind: SkeletonKind,
  skelName: string,
  schemaHash: string,
) {
  return `skeleton-list-version:${kind}:${skelName}:${schemaHash}`
}

interface SkeletonItemVersionGroup {
  skelName: string
  main: SkeletonItem
  versions: Array<SkeletonItem>
  items: Array<SkeletonItem>
}

function buildSkeletonItemVersionGroups(items: Array<SkeletonItem>) {
  const groupsBySkelName = new Map<string, SkeletonItemVersionGroup>()
  for (const item of items) {
    let group = groupsBySkelName.get(item.skelName)
    if (!group) {
      group = { skelName: item.skelName, main: item, versions: [], items: [] }
      groupsBySkelName.set(item.skelName, group)
    }
    group.items.push(item)
    if (item.isMain) {
      group.main = item
    }
  }

  const groups = [...groupsBySkelName.values()]
  for (const group of groups) {
    group.items.sort(compareSkeletonItems)
    if (!group.main.isMain) {
      group.main = group.items[0]
    }
    group.versions = group.items.filter((item) => item !== group.main)
  }
  groups.sort((a, b) => a.main.skelName.localeCompare(b.main.skelName))
  return groups
}

function compareSkeletonItems(a: SkeletonItem, b: SkeletonItem) {
  if (a.isMain !== b.isMain) {
    return a.isMain ? -1 : 1
  }
  return b.schemaHash.localeCompare(a.schemaHash)
}

function getSkeletonListBadge(item: SkeletonItem) {
  if ('lifecycle' in item && item.lifecycle) {
    return item.lifecycle
  }
  if ('pub' in item && item.pub) {
    return 'public'
  }
  if ('enum' in item && item.enum) {
    return 'Enum'
  }
  return null
}

export function SkeletonPage({ kind }: { kind: SkeletonKind }) {
  const { t, tText } = useLocale()
  const config = skeletonConfigs[kind]
  const routeConfig = skeletonRouteConfig[kind]
  const Icon = config.icon
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname.replace(/\/$/, ''),
  })
  const routeItem = useRouterState({
    select: (state) => {
      const prefix = `${routeConfig.listPath}/`
      const currentPathname = state.location.pathname

      if (!currentPathname.startsWith(prefix)) {
        return { skelName: undefined, schemaHash: undefined }
      }

      const parts = currentPathname
        .slice(prefix.length)
        .replace(/\/$/, '')
        .split('/')
      const encodedSkelName = parts[0]

      return {
        skelName: encodedSkelName
          ? decodeURIComponent(encodedSkelName)
          : undefined,
        schemaHash: parts[1] ? decodeURIComponent(parts[1]) : undefined,
      }
    },
  })
  const routeSkelName = routeItem.skelName
  const routeSchemaHash = routeItem.schemaHash
  const [items, setItems] = React.useState<Array<SkeletonItem>>(
    () => skeletonItemsByKind.get(kind) ?? [],
  )
  const [typeDefinitions, setTypeDefinitions] = React.useState<
    Array<SkeletonData>
  >(() => cachedTypeDefinitions ?? [])
  const [query, setQuery] = React.useState('')
  const [expandedVersionGroups, setExpandedVersionGroups] = React.useState<
    Record<string, boolean>
  >({})
  const [loading, setLoading] = React.useState(
    () => !skeletonItemsByKind.has(kind),
  )
  const listPanel = useResizableListPanel({
    defaultWidth: SKELETON_LIST_DEFAULT_WIDTH,
    storageKey: SKELETON_LIST_WIDTH_STORAGE_KEY,
  })
  const handleListScroll = useReservedScrollbar()
  const scrollHideTimers = React.useRef(new WeakMap<Element, number>())

  const loadItems = React.useCallback(async () => {
    const hasCachedItems = skeletonItemsByKind.has(kind)
    setLoading(!hasCachedItems)
    try {
      if (kind === 'data') {
        const dataItems = (await config.load()) as Array<SkeletonData>
        skeletonItemsByKind.set(kind, dataItems)
        cachedTypeDefinitions = dataItems
        setItems(dataItems)
        setTypeDefinitions(dataItems)
        return
      }

      if (kind === 'actors') {
        const [nextItems, dataItems] = await Promise.all([
          config.load(),
          skeletonService.listData(null),
        ])
        skeletonItemsByKind.set(kind, nextItems)
        skeletonItemsByKind.set('data', dataItems)
        cachedTypeDefinitions = dataItems
        setItems(nextItems)
        setTypeDefinitions(dataItems)
        return
      }

      const [nextItems, dataItems] = await Promise.all([
        config.load(),
        skeletonService.listData(null),
      ])
      skeletonItemsByKind.set(kind, nextItems)
      skeletonItemsByKind.set('data', dataItems)
      cachedTypeDefinitions = dataItems
      setItems(nextItems)
      setTypeDefinitions(dataItems)
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoading(false)
    }
  }, [config, kind])

  React.useEffect(() => {
    void loadItems()
  }, [loadItems])

  const handleScrollAreaScroll = React.useCallback(
    (event: React.UIEvent<HTMLElement>) => {
      const target = event.currentTarget
      target.dataset.scrolling = 'true'

      const currentTimer = scrollHideTimers.current.get(target)
      if (currentTimer !== undefined) {
        window.clearTimeout(currentTimer)
      }

      const nextTimer = window.setTimeout(() => {
        delete target.dataset.scrolling
        scrollHideTimers.current.delete(target)
      }, 900)

      scrollHideTimers.current.set(target, nextTimer)
    },
    [],
  )

  const filteredItems = React.useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (keyword === '') {
      return items
    }
    return items.filter((item) =>
      itemValues(item).some((value) => value.toLowerCase().includes(keyword)),
    )
  }, [items, query])
  const filteredGroups = React.useMemo(
    () => buildSkeletonItemVersionGroups(filteredItems),
    [filteredItems],
  )
  const allGroups = React.useMemo(
    () => buildSkeletonItemVersionGroups(items),
    [items],
  )
  React.useEffect(() => {
    if (!routeSkelName || !routeSchemaHash) {
      return
    }
    const groupKey = `${kind}:${routeSkelName}`
    setExpandedVersionGroups((current) =>
      current[groupKey] ? current : { ...current, [groupKey]: true },
    )
  }, [kind, routeSchemaHash, routeSkelName])
  React.useEffect(() => {
    if (!routeSkelName) {
      return
    }
    window.requestAnimationFrame(() => {
      const targetId = routeSchemaHash
        ? skeletonVersionDomId(kind, routeSkelName, routeSchemaHash)
        : skeletonListItemDomId(kind, routeSkelName)
      document.getElementById(targetId)?.scrollIntoView({
        block: 'nearest',
        inline: 'nearest',
      })
    })
  }, [filteredGroups, kind, routeSchemaHash, routeSkelName])
  const selectedItem = React.useMemo(
    () =>
      items.find(
        (item) =>
          item.skelName === routeSkelName &&
          (routeSchemaHash ? item.schemaHash === routeSchemaHash : item.isMain),
      ) ?? null,
    [items, routeSchemaHash, routeSkelName],
  )
  const selectedVersionGroup = React.useMemo(
    () =>
      selectedItem
        ? (allGroups.find(
            (group) => group.skelName === selectedItem.skelName,
          ) ?? null)
        : null,
    [allGroups, selectedItem],
  )
  const typeIndex = React.useMemo(
    () => buildTypeDefinitionIndex(typeDefinitions),
    [typeDefinitions],
  )
  const relatedServices = React.useMemo(() => {
    if (kind !== 'actors' || !selectedItem || !('services' in selectedItem)) {
      return []
    }
    return selectedItem.services
  }, [kind, selectedItem])
  const relatedWebs = React.useMemo(() => {
    if (kind !== 'actors' || !selectedItem || !('webs' in selectedItem)) {
      return []
    }
    return selectedItem.webs
  }, [kind, selectedItem])

  const navigateToItem = React.useCallback(
    (item: SkeletonItem, replace = false) => {
      const schemaHash = itemRouteHash(item)
      void navigate({
        to: schemaHash ? routeConfig.detailVersionPath : routeConfig.detailPath,
        params: schemaHash
          ? { skelName: item.skelName, schemaHash }
          : { skelName: item.skelName },
        replace,
      })
    },
    [navigate, routeConfig.detailPath, routeConfig.detailVersionPath],
  )
  const navigateToSkeletonItem = React.useCallback(
    (item: SkeletonItem, itemKind: SkeletonKind) => {
      const targetRouteConfig = skeletonRouteConfig[itemKind]
      const schemaHash = itemRouteHash(item)
      void navigate({
        to: schemaHash
          ? targetRouteConfig.detailVersionPath
          : targetRouteConfig.detailPath,
        params: schemaHash
          ? { skelName: item.skelName, schemaHash }
          : { skelName: item.skelName },
      })
    },
    [navigate],
  )
  React.useEffect(() => {
    if (
      loading ||
      pathname !== routeConfig.listPath ||
      routeSkelName ||
      filteredGroups.length === 0
    ) {
      return
    }
    navigateToItem(filteredGroups[0].main, true)
  }, [
    filteredGroups,
    loading,
    navigateToItem,
    pathname,
    routeConfig.listPath,
    routeSkelName,
  ])
  const navigateToTypeDefinition = React.useCallback(
    (item: SkeletonItem) => {
      navigateToSkeletonItem(item, 'data')
    },
    [navigateToSkeletonItem],
  )
  const navigateToActorDefinition = React.useCallback(
    (skelName: string, schemaHash?: string) => {
      void navigate({
        to: schemaHash
          ? '/skeleton/actor/$skelName/$schemaHash'
          : '/skeleton/actor/$skelName',
        params: schemaHash ? { skelName, schemaHash } : { skelName },
      })
    },
    [navigate],
  )
  const navigateToServiceDefinition = React.useCallback(
    (item: SkeletonServiceItem) => {
      navigateToSkeletonItem(item, 'services')
    },
    [navigateToSkeletonItem],
  )
  const navigateToDataDefinition = React.useCallback(
    (item: SkeletonData) => {
      navigateToSkeletonItem(item, 'data')
    },
    [navigateToSkeletonItem],
  )
  const navigateToWebDefinition = React.useCallback(
    (item: SkeletonWebItem) => {
      navigateToSkeletonItem(item, 'webs')
    },
    [navigateToSkeletonItem],
  )
  const navigateToDomainDefinition = React.useCallback(
    (item: SkeletonItem) => {
      void navigate({
        to: '/skeleton/domain/$domain',
        params: { domain: item.domain },
      })
    },
    [navigate],
  )

  return (
    <section className="flex h-[calc(100dvh-3.5rem)] flex-col overflow-hidden bg-white">
      <div
        className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[var(--list-panel-width)_minmax(0,1fr)]"
        style={listPanel.gridStyle}
      >
        <aside className="relative flex min-h-0 flex-col border-b border-border/70 lg:border-r lg:border-b-0">
          <div className="border-b border-border/70 p-4">
            <div className="relative">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={query}
                className="pl-8"
                placeholder={t('common.searchSkelName')}
                onChange={(event) => setQuery(event.target.value)}
              />
            </div>
            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>
                {t('common.itemCount').replace(
                  '{count}',
                  String(filteredGroups.length),
                )}
              </span>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-7"
                  title={t('action.refreshList')}
                  onClick={() => void loadItems()}
                  disabled={loading}
                >
                  {loading ? (
                    <Loader2 className="size-3.5 animate-spin" />
                  ) : (
                    <RefreshCw className="size-3.5" />
                  )}
                </Button>
              </div>
            </div>
          </div>

          <div
            className="scrollbar-reserved min-h-0 flex-1 overflow-auto py-2 pr-1 pl-2"
            onScroll={handleListScroll}
          >
            {loading ? (
              <div className="space-y-2">
                {Array.from({ length: 6 }).map((_, index) => (
                  <Skeleton key={index} className="h-16 w-full" />
                ))}
              </div>
            ) : filteredGroups.length === 0 ? (
              <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                {items.length === 0
                  ? tText(config.emptyTitle)
                  : t('skeleton.noMatch')}
              </div>
            ) : (
              <div className="min-w-0 space-y-1">
                {filteredGroups.map((group) => {
                  const item = group.main
                  const isSelected =
                    item.skelName === routeSkelName &&
                    (routeSchemaHash
                      ? item.schemaHash === routeSchemaHash
                      : item.isMain)
                  const listBadge = getSkeletonListBadge(item)
                  const groupKey = `${kind}:${group.skelName}`
                  const isExpanded = Boolean(expandedVersionGroups[groupKey])

                  return (
                    <div key={group.skelName} className="grid min-w-0 gap-1">
                      <div className="relative min-w-0">
                        {group.versions.length > 0 ? (
                          <button
                            type="button"
                            className="absolute top-2.5 right-2 z-10 inline-flex h-6 shrink-0 items-center gap-1 rounded-full border bg-background px-2 text-xs font-medium transition-colors hover:border-primary/40 hover:bg-primary/[0.04] hover:text-primary"
                            title={
                              isExpanded
                                ? t('action.collapseVersions')
                                : t('action.expandVersions')
                            }
                            onClick={(event) => {
                              event.preventDefault()
                              event.stopPropagation()
                              setExpandedVersionGroups((current) => ({
                                ...current,
                                [groupKey]: !current[groupKey],
                              }))
                            }}
                          >
                            <span>{t('version.multiple')}</span>
                            <ChevronDown
                              className={cn(
                                'size-3.5 transition-transform',
                                !isExpanded && '-rotate-90',
                              )}
                            />
                          </button>
                        ) : null}
                        <a
                          id={skeletonListItemDomId(kind, item.skelName)}
                          href={skeletonItemHref(item, kind)}
                          onClick={(event) => {
                            if (shouldUseBrowserNavigation(event)) {
                              return
                            }
                            event.preventDefault()
                            navigateToItem(item)
                          }}
                          className={cn(
                            'relative flex w-full min-w-0 flex-col gap-1 rounded-lg border px-3 py-2.5 text-left transition-colors',
                            group.versions.length > 0 ? 'pr-24' : '',
                            isSelected
                              ? 'border-primary/30 bg-primary/[0.06]'
                              : 'border-transparent hover:bg-primary/[0.05]',
                          )}
                        >
                          <span
                            className={cn(
                              'flex min-w-0 items-center gap-2 text-sm font-medium',
                              isSelected ? 'text-primary' : 'text-foreground',
                            )}
                          >
                            <span className="truncate">
                              {displayItemName(item)}
                            </span>
                            {listBadge ? (
                              <Badge variant="outline" className="shrink-0">
                                {listBadge}
                              </Badge>
                            ) : null}
                          </span>
                          <span className="truncate text-xs text-muted-foreground">
                            {item.skelName}
                          </span>
                        </a>
                      </div>
                      {isExpanded && group.versions.length > 0 ? (
                        <div
                          className={cn(
                            'ml-3 grid min-w-0 gap-1 border-l pl-2',
                            group.versions.some(
                              (version) =>
                                version.skelName === routeSkelName &&
                                version.schemaHash === routeSchemaHash,
                            ) && 'border-amber-300',
                          )}
                        >
                          {group.versions.map((version) => {
                            const versionSelected =
                              version.skelName === routeSkelName &&
                              version.schemaHash === routeSchemaHash
                            const versionBadge = getSkeletonListBadge(version)
                            return (
                              <a
                                key={itemVersionKey(version)}
                                id={skeletonVersionDomId(
                                  kind,
                                  version.skelName,
                                  version.schemaHash,
                                )}
                                href={skeletonItemHref(version, kind)}
                                onClick={(event) => {
                                  if (shouldUseBrowserNavigation(event)) {
                                    return
                                  }
                                  event.preventDefault()
                                  navigateToItem(version)
                                }}
                                className={cn(
                                  'grid min-w-0 gap-1 rounded-md border px-3 py-2 text-left transition-colors',
                                  versionSelected
                                    ? 'border-amber-300 bg-amber-50'
                                    : 'border-transparent hover:bg-amber-50/60',
                                )}
                              >
                                <div className="flex min-w-0 items-center gap-2">
                                  <div className="flex min-w-0 flex-1 items-center gap-2">
                                    <span
                                      className={cn(
                                        'truncate text-sm font-medium',
                                        versionSelected
                                          ? 'text-amber-700'
                                          : 'text-foreground',
                                      )}
                                    >
                                      {displayItemName(version)}
                                    </span>
                                    {versionBadge ? (
                                      <Badge
                                        variant="outline"
                                        className="shrink-0"
                                      >
                                        {versionBadge}
                                      </Badge>
                                    ) : null}
                                  </div>
                                  <Badge
                                    variant="outline"
                                    className="shrink-0 border-amber-300 bg-amber-50 text-amber-700"
                                  >
                                    {version.schemaHash}
                                  </Badge>
                                </div>
                                <span className="truncate font-mono text-xs text-muted-foreground">
                                  {version.skelName}
                                </span>
                              </a>
                            )
                          })}
                        </div>
                      ) : null}
                    </div>
                  )
                })}
              </div>
            )}
          </div>
          <ResizableListHandle
            defaultWidth={SKELETON_LIST_DEFAULT_WIDTH}
            label={t('skeleton.resizeList')}
            panel={listPanel}
          />
        </aside>

        <main className="min-h-0 overflow-hidden">
          {!routeSkelName && !loading ? (
            <div className="flex h-full min-h-[24rem] items-center justify-center text-sm text-muted-foreground">
              {t('skeleton.selectOne')}
            </div>
          ) : loading ? (
            <div className="space-y-4 p-6">
              <Skeleton className="h-8 w-56" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-[28rem] w-full" />
            </div>
          ) : selectedItem ? (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b border-border/70 px-6 py-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <Icon className="size-4 shrink-0 text-primary" />
                      <h2 className="min-w-0 truncate text-base font-semibold">
                        {displayItemName(selectedItem)}
                      </h2>
                      <SkeletonItemBadges
                        item={selectedItem}
                        showVersion={false}
                      />
                      {!selectedItem.isMain ? (
                        <Badge
                          variant="outline"
                          className="border-amber-300 bg-amber-50 text-amber-700"
                        >
                          {selectedItem.schemaHash}
                        </Badge>
                      ) : selectedVersionGroup &&
                        selectedVersionGroup.versions.length > 0 ? (
                        <Popover>
                          <PopoverTrigger
                            className="inline-flex h-6 items-center gap-1 rounded-full border bg-background px-2 text-xs font-medium transition-colors hover:border-primary/40 hover:bg-primary/[0.04] hover:text-primary"
                            title={t('action.viewVersions')}
                          >
                            <span>{t('version.multiple')}</span>
                            <ChevronDown className="size-3.5" />
                          </PopoverTrigger>
                          <PopoverContent
                            align="start"
                            className="w-52 gap-1 p-1.5"
                          >
                            {selectedVersionGroup.versions.map((version) => {
                              const versionSelected =
                                version.schemaHash === selectedItem.schemaHash
                              return (
                                <a
                                  key={itemVersionKey(version)}
                                  href={skeletonItemHref(version, kind)}
                                  onClick={(event) => {
                                    if (shouldUseBrowserNavigation(event)) {
                                      return
                                    }
                                    event.preventDefault()
                                    navigateToItem(version)
                                  }}
                                  className={cn(
                                    'flex items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors',
                                    versionSelected
                                      ? 'bg-amber-50 text-amber-700'
                                      : 'hover:bg-primary/[0.05]',
                                  )}
                                >
                                  <span className="font-mono text-xs">
                                    {version.schemaHash}
                                  </span>
                                  {versionSelected ? (
                                    <Badge
                                      variant="outline"
                                      className="border-amber-300 bg-amber-50 text-amber-700"
                                    >
                                      {t('action.current')}
                                    </Badge>
                                  ) : null}
                                </a>
                              )
                            })}
                          </PopoverContent>
                        </Popover>
                      ) : null}
                    </div>
                    <p className="mt-2 truncate font-mono text-xs text-muted-foreground">
                      {(() => {
                        const { domainPart, restPart } = splitDomainSkelName(
                          selectedItem.domain,
                          selectedItem.skelName,
                        )
                        return (
                          <>
                            <a
                              href={skeletonDomainHref(selectedItem)}
                              className="font-mono text-primary underline-offset-2 hover:underline"
                              onClick={(event) => {
                                if (shouldUseBrowserNavigation(event)) {
                                  return
                                }
                                event.preventDefault()
                                navigateToDomainDefinition(selectedItem)
                              }}
                            >
                              {domainPart}
                            </a>
                            {restPart}
                          </>
                        )
                      })()}
                    </p>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => void loadItems()}
                    disabled={loading}
                  >
                    <RefreshCw />
                    {t('action.refresh')}
                  </Button>
                </div>
                {selectedItem.description ? (
                  <p className="mt-2 text-sm text-muted-foreground">
                    {selectedItem.description}
                  </p>
                ) : null}
              </div>

              <div
                className="scrollbar-reserved min-h-0 flex-1 overflow-y-auto p-6 pr-4"
                onScroll={handleScrollAreaScroll}
              >
                <SkeletonItemDetails
                  item={selectedItem}
                  kind={kind}
                  typeIndex={typeIndex}
                  relatedServices={relatedServices}
                  relatedWebs={relatedWebs}
                  onTypeClick={navigateToTypeDefinition}
                  onActorClick={navigateToActorDefinition}
                  onServiceClick={navigateToServiceDefinition}
                  onDataClick={navigateToDataDefinition}
                  onWebClick={navigateToWebDefinition}
                />
              </div>
            </div>
          ) : (
            <div className="flex h-full min-h-[24rem] items-center justify-center text-sm text-muted-foreground">
              {t('skeleton.notFound')}
            </div>
          )}
        </main>
      </div>
    </section>
  )
}
