import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import {
  Braces,
  CalendarClock,
  ChevronDown,
  Globe2,
  Loader2,
  Radio,
  RefreshCw,
  Search,
  Server,
  ShieldCheck,
  SlidersHorizontal,
  Users,
} from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  ResizableListHandle,
  useReservedScrollbar,
  useResizableListPanel,
} from '@/components/ui/resizable-list-panel'
import { Skeleton } from '@/components/ui/skeleton'
import { vrpcClient } from '@/config/vrpc-client'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import { createSkeletonService } from '@/skeled'
import type {
  SkeletonActorItem,
  SkeletonConfigItem,
  SkeletonData,
  SkeletonDomain,
  SkeletonEventItem,
  SkeletonResourceItem,
  SkeletonServiceItem,
  SkeletonTask,
  SkeletonWebItem,
} from '@/skeled'

import { skeletonItemHref } from './model'

const skeletonService = createSkeletonService(vrpcClient)
const SKELETON_DOMAIN_LIST_DEFAULT_WIDTH = 352
const SKELETON_DOMAIN_LIST_WIDTH_STORAGE_KEY =
  'vinehub_skeleton_domain_list_width_v2'

type SkeletonDomainKind =
  | 'actors'
  | 'configs'
  | 'services'
  | 'resources'
  | 'data'
  | 'webs'
  | 'tasks'
  | 'events'

type SkeletonDomainItem =
  | SkeletonActorItem
  | SkeletonConfigItem
  | SkeletonServiceItem
  | SkeletonResourceItem
  | SkeletonData
  | SkeletonWebItem
  | SkeletonTask
  | SkeletonEventItem

interface SkeletonDomainGroup {
  kind: SkeletonDomainKind
  label: string
  icon: React.ComponentType<{ className?: string }>
  detailPath: string
  detailVersionPath: string
  items: Array<SkeletonDomainItem>
}

interface SkeletonDomainVersionGroup {
  domain: string
  main: SkeletonDomain
  versions: Array<SkeletonDomain>
  items: Array<SkeletonDomain>
}

const groupDefinitions: Record<
  SkeletonDomainKind,
  {
    label: string
    icon: React.ComponentType<{ className?: string }>
    detailPath: string
    detailVersionPath: string
  }
> = {
  actors: {
    label: 'Actor',
    icon: Users,
    detailPath: '/skeleton/actor/$skelName',
    detailVersionPath: '/skeleton/actor/$skelName/$schemaHash',
  },
  configs: {
    label: 'Config',
    icon: SlidersHorizontal,
    detailPath: '/skeleton/config/$skelName',
    detailVersionPath: '/skeleton/config/$skelName/$schemaHash',
  },
  services: {
    label: 'Service',
    icon: Server,
    detailPath: '/skeleton/service/$skelName',
    detailVersionPath: '/skeleton/service/$skelName/$schemaHash',
  },
  resources: {
    label: 'Resource',
    icon: ShieldCheck,
    detailPath: '/skeleton/resource/$skelName',
    detailVersionPath: '/skeleton/resource/$skelName/$schemaHash',
  },
  data: {
    label: 'Data',
    icon: Braces,
    detailPath: '/skeleton/data/$skelName',
    detailVersionPath: '/skeleton/data/$skelName/$schemaHash',
  },
  webs: {
    label: 'Web',
    icon: Globe2,
    detailPath: '/skeleton/web/$skelName',
    detailVersionPath: '/skeleton/web/$skelName/$schemaHash',
  },
  tasks: {
    label: 'Task',
    icon: CalendarClock,
    detailPath: '/skeleton/task/$skelName',
    detailVersionPath: '/skeleton/task/$skelName/$schemaHash',
  },
  events: {
    label: 'Event',
    icon: Radio,
    detailPath: '/skeleton/event/$skelName',
    detailVersionPath: '/skeleton/event/$skelName/$schemaHash',
  },
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function displayItemName(item: SkeletonDomainItem) {
  if ('typeParameters' in item && item.typeParameters.length > 0) {
    return `${item.name}<${item.typeParameters.join(', ')}>`
  }
  return item.name
}

function getDomainGroup(
  kind: SkeletonDomainKind,
  items: Array<SkeletonDomainItem>,
): SkeletonDomainGroup {
  const definition = groupDefinitions[kind]
  return {
    kind,
    label: definition.label,
    icon: definition.icon,
    detailPath: definition.detailPath,
    detailVersionPath: definition.detailVersionPath,
    items,
  }
}

function getDomainGroups(summary: SkeletonDomain) {
  return [
    getDomainGroup('actors', summary.actors),
    getDomainGroup('configs', summary.configs),
    getDomainGroup('data', summary.data),
    getDomainGroup('services', summary.services),
    getDomainGroup('resources', summary.resources),
    getDomainGroup('webs', summary.webs),
    getDomainGroup('events', summary.events),
    getDomainGroup('tasks', summary.tasks),
  ].filter((group) => group.items.length > 0)
}

function getDomainSummaryKey(summary: SkeletonDomain) {
  return `${summary.domain}:${summary.schemaHash}`
}

function skeletonDomainListItemDomId(domain: string) {
  return `skeleton-domain-list-item:${encodeURIComponent(domain)}`
}

function skeletonDomainVersionDomId(domain: string, schemaHash: string) {
  return `skeleton-domain-list-version:${encodeURIComponent(domain)}:${encodeURIComponent(schemaHash)}`
}

function buildDomainVersionGroups(items: Array<SkeletonDomain>) {
  const groupsByDomain = new Map<string, SkeletonDomainVersionGroup>()
  for (const item of items) {
    let group = groupsByDomain.get(item.domain)
    if (!group) {
      group = { domain: item.domain, main: item, versions: [], items: [] }
      groupsByDomain.set(item.domain, group)
    }
    group.items.push(item)
    if (item.isMain) {
      group.main = item
    }
  }

  const groups = [...groupsByDomain.values()]
  for (const group of groups) {
    group.items.sort(compareDomainVersions)
    if (!group.main.isMain) {
      group.main = group.items[0]
    }
    group.versions = group.items.filter((item) => item !== group.main)
  }
  groups.sort((a, b) => a.main.domain.localeCompare(b.main.domain))
  return groups
}

function compareDomainVersions(a: SkeletonDomain, b: SkeletonDomain) {
  if (a.isMain !== b.isMain) {
    return a.isMain ? -1 : 1
  }
  return b.schemaHash.localeCompare(a.schemaHash)
}

function skeletonDomainHref(summary: SkeletonDomain) {
  const basePath = `/skeleton/domain/${encodeURIComponent(summary.domain)}`
  return !summary.isMain
    ? `${basePath}/${encodeURIComponent(summary.schemaHash)}`
    : basePath
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

export function SkeletonDomainPage() {
  const { t } = useLocale()
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname.replace(/\/$/, ''),
  })
  const routeParams = useRouterState({
    select: (state) => {
      const prefix = '/skeleton/domain/'
      const currentPathname = state.location.pathname

      if (!currentPathname.startsWith(prefix)) {
        return { domain: undefined, schemaHash: undefined }
      }

      const segments = currentPathname
        .slice(prefix.length)
        .replace(/\/$/, '')
        .split('/')
        .filter(Boolean)
      return {
        domain: segments[0] ? decodeURIComponent(segments[0]) : undefined,
        schemaHash: segments[1] ? decodeURIComponent(segments[1]) : undefined,
      }
    },
  })
  const [summaries, setSummaries] = React.useState<Array<SkeletonDomain>>([])
  const [collapsedGroups, setCollapsedGroups] = React.useState<
    Record<string, boolean>
  >({})
  const [expandedDomainGroups, setExpandedDomainGroups] = React.useState<
    Record<string, boolean>
  >({})
  const [query, setQuery] = React.useState('')
  const [loading, setLoading] = React.useState(true)
  const listPanel = useResizableListPanel({
    defaultWidth: SKELETON_DOMAIN_LIST_DEFAULT_WIDTH,
    storageKey: SKELETON_DOMAIN_LIST_WIDTH_STORAGE_KEY,
  })
  const handleListScroll = useReservedScrollbar()

  const loadItems = React.useCallback(async () => {
    setLoading(true)
    try {
      setSummaries(await skeletonService.listDomains(null))
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void loadItems()
  }, [loadItems])

  const filteredSummaries = React.useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (keyword === '') {
      return summaries
    }
    return summaries.filter((summary) =>
      summary.domain.toLowerCase().includes(keyword),
    )
  }, [query, summaries])
  const filteredDomainGroups = React.useMemo(
    () => buildDomainVersionGroups(filteredSummaries),
    [filteredSummaries],
  )
  const selectedSummary = React.useMemo(
    () =>
      summaries.find(
        (summary) =>
          summary.domain === routeParams.domain &&
          (routeParams.schemaHash
            ? summary.schemaHash === routeParams.schemaHash
            : summary.isMain),
      ) ??
      filteredSummaries.at(0) ??
      null,
    [filteredSummaries, routeParams.domain, routeParams.schemaHash, summaries],
  )

  const navigateToDomain = React.useCallback(
    (summary: SkeletonDomain, replace = false) => {
      void navigate({
        to: !summary.isMain
          ? '/skeleton/domain/$domain/$schemaHash'
          : '/skeleton/domain/$domain',
        params: !summary.isMain
          ? { domain: summary.domain, schemaHash: summary.schemaHash }
          : { domain: summary.domain },
        replace,
      })
    },
    [navigate],
  )
  React.useEffect(() => {
    const domain = routeParams.domain
    if (!domain || !routeParams.schemaHash) {
      return
    }
    setExpandedDomainGroups((current) =>
      current[domain] ? current : { ...current, [domain]: true },
    )
  }, [routeParams.domain, routeParams.schemaHash])
  React.useEffect(() => {
    if (!selectedSummary) {
      return
    }
    window.requestAnimationFrame(() => {
      const targetId = !selectedSummary.isMain
        ? skeletonDomainVersionDomId(
            selectedSummary.domain,
            selectedSummary.schemaHash,
          )
        : skeletonDomainListItemDomId(selectedSummary.domain)
      document.getElementById(targetId)?.scrollIntoView({
        block: 'nearest',
        inline: 'nearest',
      })
    })
  }, [filteredDomainGroups, selectedSummary])
  React.useEffect(() => {
    if (
      loading ||
      pathname !== '/skeleton/domain' ||
      routeParams.domain ||
      filteredDomainGroups.length === 0
    ) {
      return
    }
    navigateToDomain(filteredDomainGroups[0].main, true)
  }, [
    filteredDomainGroups,
    loading,
    navigateToDomain,
    pathname,
    routeParams.domain,
  ])
  const navigateToItem = React.useCallback(
    (group: SkeletonDomainGroup, item: SkeletonDomainItem) => {
      void navigate({
        to: !item.isMain ? group.detailVersionPath : group.detailPath,
        params: !item.isMain
          ? { skelName: item.skelName, schemaHash: item.schemaHash }
          : { skelName: item.skelName },
      })
    },
    [navigate],
  )
  const toggleGroupCollapsed = React.useCallback(
    (summary: SkeletonDomain, kind: SkeletonDomainKind) => {
      const key = `${getDomainSummaryKey(summary)}:${kind}`
      setCollapsedGroups((current) => ({
        ...current,
        [key]: !current[key],
      }))
    },
    [],
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
                placeholder={t('skeleton.domainSearchPlaceholder')}
                onChange={(event) => setQuery(event.target.value)}
              />
            </div>
            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>{t('skeleton.domainList')}</span>
              <div className="flex items-center gap-2">
                <span>
                  {t('common.itemCount').replace(
                    '{count}',
                    String(filteredDomainGroups.length),
                  )}
                </span>
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
            ) : filteredDomainGroups.length === 0 ? (
              <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                {summaries.length === 0
                  ? t('skeleton.domainEmpty')
                  : t('skeleton.domainNoMatch')}
              </div>
            ) : (
              <div className="space-y-1">
                {filteredDomainGroups.map((group) => {
                  const summary = group.main
                  const isSelected =
                    getDomainSummaryKey(summary) ===
                    (selectedSummary
                      ? getDomainSummaryKey(selectedSummary)
                      : undefined)
                  const groupExpanded = Boolean(
                    expandedDomainGroups[group.domain],
                  )

                  return (
                    <div key={group.domain} className="grid gap-1">
                      <div className="relative">
                        {group.versions.length > 0 ? (
                          <button
                            type="button"
                            className="absolute top-2.5 right-2 z-10 inline-flex h-6 items-center gap-1 rounded-full border bg-background px-2 text-xs font-medium transition-colors hover:border-primary/40 hover:bg-primary/[0.04] hover:text-primary"
                            title={
                              groupExpanded
                                ? t('action.collapseVersions')
                                : t('action.expandVersions')
                            }
                            onClick={(event) => {
                              event.preventDefault()
                              event.stopPropagation()
                              setExpandedDomainGroups((current) => ({
                                ...current,
                                [group.domain]: !current[group.domain],
                              }))
                            }}
                          >
                            <span>{t('version.multiple')}</span>
                            <ChevronDown
                              className={cn(
                                'size-3.5 transition-transform',
                                !groupExpanded && '-rotate-90',
                              )}
                            />
                          </button>
                        ) : null}
                        <a
                          id={skeletonDomainListItemDomId(summary.domain)}
                          href={skeletonDomainHref(summary)}
                          onClick={(event) => {
                            if (shouldUseBrowserNavigation(event)) {
                              return
                            }
                            event.preventDefault()
                            navigateToDomain(summary)
                          }}
                          className={cn(
                            'flex w-full flex-col gap-1 rounded-lg border px-3 py-2.5 text-left transition-colors',
                            group.versions.length > 0 ? 'pr-24' : '',
                            isSelected
                              ? 'border-primary/30 bg-primary/[0.06]'
                              : 'border-transparent hover:bg-primary/[0.05]',
                          )}
                        >
                          <span
                            className={cn(
                              'truncate text-sm font-medium',
                              isSelected ? 'text-primary' : 'text-foreground',
                            )}
                          >
                            {summary.domain}
                          </span>
                          {summary.isMultiVersion ? (
                            <span className="truncate font-mono text-xs text-muted-foreground">
                              {t('skeleton.mainVersion')} · {summary.schemaHash}
                            </span>
                          ) : null}
                        </a>
                      </div>
                      {groupExpanded && group.versions.length > 0 ? (
                        <div className="ml-3 grid gap-1 border-l pl-2">
                          {group.versions.map((version) => {
                            const versionSelected =
                              getDomainSummaryKey(version) ===
                              (selectedSummary
                                ? getDomainSummaryKey(selectedSummary)
                                : undefined)
                            return (
                              <a
                                key={getDomainSummaryKey(version)}
                                id={skeletonDomainVersionDomId(
                                  version.domain,
                                  version.schemaHash,
                                )}
                                href={skeletonDomainHref(version)}
                                onClick={(event) => {
                                  if (shouldUseBrowserNavigation(event)) {
                                    return
                                  }
                                  event.preventDefault()
                                  navigateToDomain(version)
                                }}
                                className={cn(
                                  'relative grid gap-1 rounded-md border px-3 py-2 pr-24 text-left transition-colors',
                                  versionSelected
                                    ? 'border-amber-300 bg-amber-50'
                                    : 'border-transparent hover:bg-amber-50/60',
                                )}
                              >
                                <Badge
                                  variant="outline"
                                  className="absolute top-2 right-3 border-amber-300 bg-amber-50 text-amber-700"
                                >
                                  {version.schemaHash}
                                </Badge>
                                <div className="flex min-w-0 items-center gap-2">
                                  <span
                                    className={cn(
                                      'truncate text-sm font-medium',
                                      versionSelected
                                        ? 'text-amber-700'
                                        : 'text-foreground',
                                    )}
                                  >
                                    {version.domain}
                                  </span>
                                </div>
                                <span className="truncate font-mono text-xs text-muted-foreground">
                                  {version.schemaHash}
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
            defaultWidth={SKELETON_DOMAIN_LIST_DEFAULT_WIDTH}
            label={t('skeleton.resizeDomainList')}
            panel={listPanel}
          />
        </aside>

        <main className="min-h-0 overflow-hidden">
          {loading ? (
            <div className="space-y-4 p-6">
              <Skeleton className="h-8 w-56" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-[28rem] w-full" />
            </div>
          ) : selectedSummary ? (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b border-border/70 px-6 py-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex min-w-0 items-center gap-2">
                      <h2 className="min-w-0 truncate text-base font-semibold">
                        {selectedSummary.domain}
                      </h2>
                      {!selectedSummary.isMain ? (
                        <Badge
                          variant="outline"
                          className="border-amber-300 bg-amber-50 text-amber-700"
                        >
                          {selectedSummary.schemaHash}
                        </Badge>
                      ) : selectedSummary.isMultiVersion ? (
                        <Badge variant="outline">
                          {t('skeleton.mainVersion')}
                        </Badge>
                      ) : null}
                    </div>
                    <p className="mt-2 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                      <span>
                        {t('skeleton.itemCount').replace(
                          '{count}',
                          String(selectedSummary.total),
                        )}
                      </span>
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
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto">
                <div className="grid gap-5 px-6 pt-6 pb-6">
                  {getDomainGroups(selectedSummary).map((group) => {
                    const Icon = group.icon
                    const groupKey = `${getDomainSummaryKey(selectedSummary)}:${group.kind}`
                    const isCollapsed = Boolean(collapsedGroups[groupKey])
                    return (
                      <section key={group.kind} className="grid gap-2">
                        <button
                          type="button"
                          className="sticky top-0 z-20 -mx-6 flex items-center gap-2 bg-white px-6 py-2 text-left"
                          onClick={() =>
                            toggleGroupCollapsed(selectedSummary, group.kind)
                          }
                        >
                          <ChevronDown
                            className={cn(
                              'size-3.5 text-muted-foreground transition-transform',
                              isCollapsed && '-rotate-90',
                            )}
                          />
                          <Icon className="size-4 text-primary" />
                          <h3 className="text-sm font-semibold">
                            {group.label}
                          </h3>
                          <Badge variant="outline">{group.items.length}</Badge>
                        </button>
                        <div
                          className={cn('grid gap-2', isCollapsed && 'hidden')}
                        >
                          {group.items.map((item) => (
                            <a
                              key={`${item.skelName}:${item.schemaHash}`}
                              href={skeletonItemHref(item, group.kind)}
                              onClick={(event) => {
                                if (shouldUseBrowserNavigation(event)) {
                                  return
                                }
                                event.preventDefault()
                                navigateToItem(group, item)
                              }}
                              className="grid gap-1 rounded-md border px-3 py-2.5 text-left transition-colors hover:border-primary/30 hover:bg-primary/[0.04]"
                            >
                              <span className="truncate text-sm font-medium">
                                {displayItemName(item)}
                              </span>
                              <span className="truncate font-mono text-xs text-muted-foreground">
                                {item.skelName}
                              </span>
                            </a>
                          ))}
                        </div>
                      </section>
                    )
                  })}
                </div>
              </div>
            </div>
          ) : (
            <div className="flex h-full min-h-[24rem] items-center justify-center text-sm text-muted-foreground">
              {t('skeleton.domainEmpty')}
            </div>
          )}
        </main>
      </div>
    </section>
  )
}
