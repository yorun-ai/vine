import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import {
  ArrowRight,
  ChevronDown,
  Compass,
  Edit3,
  GitBranch,
  Loader2,
  RefreshCw,
  Search,
} from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  ResizableListHandle,
  useReservedScrollbar,
  useResizableListPanel,
} from '@/components/ui/resizable-list-panel'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { vrpcClient } from '@/config/vrpc-client'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import { createPortalEntryService } from '@/skeled'
import type {
  PortalEntry,
  PortalEntryAccessUpdate,
  PortalEntryRule,
  PortalRule,
} from '@/skeled'

const portalEntryService = createPortalEntryService(vrpcClient)
const PORTAL_ENTRY_LIST_DEFAULT_WIDTH = 352
const PORTAL_ENTRY_LIST_WIDTH_STORAGE_KEY = 'vinehub_portal_entry_list_width'
const portalEntrySchemes = ['http', 'https'] as const

interface PortalEntryAccessFormValue {
  scheme: string
  host: string
  port: string
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function portalEntryToAccessFormValue(
  entry: PortalEntry,
): PortalEntryAccessFormValue {
  return {
    scheme: entry.scheme,
    host: entry.host,
    port: String(entry.port),
  }
}

function portalEntryAccessFormValueToUpdate(
  value: PortalEntryAccessFormValue,
): PortalEntryAccessUpdate {
  return {
    scheme: value.scheme,
    host: value.host.trim(),
    port: Number(value.port),
  }
}

function portalEntryAddress(entry: PortalEntry) {
  return `${entry.scheme}://${entry.host || '*'}:${entry.port}`
}

function isValidPort(value: string) {
  const port = Number(value)
  return Number.isInteger(port) && port >= 0 && port <= 65535
}

function ruleTargetLabel(rule: PortalRule) {
  switch (rule.targetType) {
    case 'SITE':
      return 'Site'
    case 'PERMANENT_REDIRECT':
      return 'Permanent Redirect'
    case 'TEMPORARY_REDIRECT':
      return 'Temporary Redirect'
    default:
      return rule.targetType
  }
}

function ruleTargetValue(entryRule: PortalEntryRule) {
  const { rule, site } = entryRule
  if (rule.targetType === 'SITE') {
    return site?.name ?? rule.siteName
  }
  return rule.redirectionPattern
}

function formatRuleMatch(rule: PortalRule) {
  const pathPrefix = rule.pathPrefix || '/'
  return rule.host ? `${rule.host}${pathPrefix}` : pathPrefix
}

function portalRuleHref(rule: PortalRule) {
  return `/portal/rule/${rule.id}`
}

function portalSitePath(id: number) {
  return `/portal/site/${id}`
}

function portalEntryPath(name: string) {
  return `/portal/entry/${encodeURIComponent(name)}`
}

function portalEntryListItemDomId(name: string) {
  return `portal-entry-list-item:${encodeURIComponent(name)}`
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

function selectedEntryNameFromPath(pathname: string) {
  const match = pathname.match(/^\/portal\/entry\/(.+)$/)
  return match ? decodeURIComponent(match[1]) : null
}

function isPortalEntryPath(pathname: string) {
  return pathname === '/portal/entry' || pathname.startsWith('/portal/entry/')
}

function PortalEntryListSkeleton() {
  return (
    <div className="space-y-1">
      {Array.from({ length: 4 }).map((_, index) => (
        <div key={index} className="rounded-lg px-3 py-2.5">
          <Skeleton className="mb-2 h-4 w-32" />
          <Skeleton className="h-3 w-44" />
        </div>
      ))}
    </div>
  )
}

function PortalEntryAccessDialog({
  entry,
  open,
  onOpenChange,
  onUpdated,
}: {
  entry: PortalEntry | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onUpdated: (entry: PortalEntry) => void
}) {
  const { t } = useLocale()
  const [saving, setSaving] = React.useState(false)
  const [formValue, setFormValue] = React.useState<PortalEntryAccessFormValue>(
    () =>
      entry
        ? portalEntryToAccessFormValue(entry)
        : { scheme: 'http', host: '', port: '80' },
  )

  React.useEffect(() => {
    if (!open || entry == null) {
      return
    }
    setFormValue(portalEntryToAccessFormValue(entry))
  }, [entry, open])

  const updateField = React.useCallback(
    (field: keyof PortalEntryAccessFormValue, value: string) => {
      setFormValue((current) => ({
        ...current,
        [field]: value,
      }))
    },
    [],
  )

  const submit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      if (entry == null) {
        return
      }
      if (!isValidPort(formValue.port)) {
        toast.error(t('portalEntry.invalidPort'))
        return
      }

      setSaving(true)
      try {
        const updated = await portalEntryService.updateAccess({
          scheme: entry.scheme,
          host: entry.host,
          port: entry.port,
          update: portalEntryAccessFormValueToUpdate(formValue),
        })
        toast.success(t('portalEntry.updateSuccess'))
        onUpdated(updated)
        onOpenChange(false)
      } catch (error) {
        toast.error(getErrorMessage(error))
      } finally {
        setSaving(false)
      }
    },
    [entry, formValue, onOpenChange, onUpdated, t],
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('portalEntry.editAccessTitle')}</DialogTitle>
          <DialogDescription>
            {t('portalEntry.editAccessDescription')}
          </DialogDescription>
        </DialogHeader>
        <form className="grid gap-5" onSubmit={submit}>
          <div className="grid gap-2">
            <Label>{t('portalEntry.scheme')}</Label>
            <Select
              value={formValue.scheme}
              onValueChange={(value) => updateField('scheme', value ?? '')}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {portalEntrySchemes.map((scheme) => (
                  <SelectItem key={scheme} value={scheme}>
                    {scheme}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label>{t('portalEntry.host')}</Label>
            <Input
              value={formValue.host}
              placeholder="*"
              onChange={(event) => updateField('host', event.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label>{t('portalEntry.port')}</Label>
            <Input
              value={formValue.port}
              inputMode="numeric"
              onChange={(event) => updateField('port', event.target.value)}
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={saving}
            >
              {t('action.cancel')}
            </Button>
            <Button type="submit" disabled={saving}>
              {saving ? <Loader2 className="size-4 animate-spin" /> : null}
              {t('action.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function PortalEntryPage() {
  const { t, tText } = useLocale()
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const [entries, setEntries] = React.useState<Array<PortalEntry>>([])
  const [query, setQuery] = React.useState('')
  const [loading, setLoading] = React.useState(true)
  const [editingEntry, setEditingEntry] = React.useState<PortalEntry | null>(
    null,
  )
  const listPanel = useResizableListPanel({
    defaultWidth: PORTAL_ENTRY_LIST_DEFAULT_WIDTH,
    storageKey: PORTAL_ENTRY_LIST_WIDTH_STORAGE_KEY,
  })
  const handleListScroll = useReservedScrollbar()
  const [selectedEntryName, setSelectedEntryName] = React.useState<
    string | null
  >(() => selectedEntryNameFromPath(window.location.pathname))
  const [collapsedSections, setCollapsedSections] = React.useState<
    Record<string, boolean>
  >({})
  const loadEntries = React.useCallback(async () => {
    setLoading(true)

    try {
      const loadedEntries = await portalEntryService.list(null)
      setEntries(loadedEntries)
      return loadedEntries
    } catch (error) {
      toast.error(getErrorMessage(error))
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void loadEntries()
  }, [loadEntries])

  const visibleEntries = React.useMemo(() => {
    return entries.filter((entry) => entry.rules.length > 0)
  }, [entries])

  const filteredEntries = React.useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (keyword === '') {
      return visibleEntries
    }

    return visibleEntries.filter((entry) => {
      const values = [
        entry.name,
        entry.scheme,
        entry.host,
        String(entry.port),
        ...entry.rules.flatMap((entryRule) => [
          entryRule.site?.name ?? '',
          entryRule.site?.actorSkelName ?? '',
          entryRule.site?.webName ?? '',
          ...((entryRule.site?.rpcgwServices as Array<string> | undefined) ??
            []),
          entryRule.rule.name,
          entryRule.rule.host,
          entryRule.rule.pathPrefix,
          entryRule.rule.targetType,
          entryRule.rule.siteName,
          entryRule.rule.redirectionPattern,
        ]),
      ]

      return values.some((value) => value.toLowerCase().includes(keyword))
    })
  }, [query, visibleEntries])

  const selectedEntry = React.useMemo(
    () =>
      filteredEntries.find((entry) => entry.name === selectedEntryName) ??
      filteredEntries[0] ??
      null,
    [filteredEntries, selectedEntryName],
  )
  const selectEntry = React.useCallback(
    (name: string, replace = false) => {
      setSelectedEntryName(name)
      void navigate({ replace, to: portalEntryPath(name) })
    },
    [navigate],
  )

  const jumpToRule = React.useCallback(
    (rule: PortalRule) => {
      void navigate({ to: portalRuleHref(rule) })
    },
    [navigate],
  )

  const jumpToSite = React.useCallback(
    (entryRule: PortalEntryRule) => {
      if (entryRule.site == null) {
        return
      }
      void navigate({ to: portalSitePath(entryRule.site.id) })
    },
    [navigate],
  )

  const toggleSectionCollapsed = React.useCallback((section: string) => {
    setCollapsedSections((current) => ({
      ...current,
      [section]: !current[section],
    }))
  }, [])

  const handleEntryUpdated = React.useCallback(
    (entry: PortalEntry) => {
      void loadEntries().then(() => {
        selectEntry(entry.name, true)
      })
    },
    [loadEntries, selectEntry],
  )

  React.useEffect(() => {
    if (!isPortalEntryPath(pathname)) {
      return
    }
    setSelectedEntryName(selectedEntryNameFromPath(pathname))
  }, [pathname])

  React.useEffect(() => {
    if (!isPortalEntryPath(pathname)) {
      return
    }
    if (filteredEntries.length === 0) {
      setSelectedEntryName(null)
      return
    }

    if (!filteredEntries.some((entry) => entry.name === selectedEntryName)) {
      selectEntry(filteredEntries[0].name, true)
    }
  }, [filteredEntries, pathname, selectEntry, selectedEntryName])
  React.useEffect(() => {
    if (!selectedEntryName) {
      return
    }
    window.requestAnimationFrame(() => {
      document
        .getElementById(portalEntryListItemDomId(selectedEntryName))
        ?.scrollIntoView({
          block: 'nearest',
          inline: 'nearest',
        })
    })
  }, [filteredEntries, selectedEntryName])

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
                placeholder={t('portalEntry.searchPlaceholder')}
                onChange={(event) => setQuery(event.target.value)}
              />
            </div>
            <div className="flex items-center justify-between gap-2">
              <div className="text-xs text-muted-foreground">
                {t('portalEntry.itemCount').replace(
                  '{count}',
                  String(visibleEntries.length),
                )}
              </div>
              <div className="flex items-center gap-1">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-7"
                  title={t('action.refreshList')}
                  onClick={() => void loadEntries()}
                  disabled={loading}
                >
                  <RefreshCw
                    className={cn('size-3.5', loading && 'animate-spin')}
                  />
                </Button>
              </div>
            </div>
          </div>

          <div
            className="scrollbar-reserved min-h-0 flex-1 overflow-auto py-2 pr-1 pl-2"
            onScroll={handleListScroll}
          >
            {loading ? (
              <PortalEntryListSkeleton />
            ) : filteredEntries.length === 0 ? (
              <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                {visibleEntries.length === 0
                  ? t('portalEntry.empty')
                  : t('portalEntry.noMatch')}
              </div>
            ) : (
              <div className="space-y-1">
                {filteredEntries.map((entry) => (
                  <a
                    key={entry.name}
                    id={portalEntryListItemDomId(entry.name)}
                    href={portalEntryPath(entry.name)}
                    onClick={(event) => {
                      if (shouldUseBrowserNavigation(event)) {
                        return
                      }
                      event.preventDefault()
                      selectEntry(entry.name)
                    }}
                    className={cn(
                      'flex w-full flex-col gap-1 rounded-lg border px-3 py-2.5 text-left transition-colors',
                      selectedEntry?.name === entry.name
                        ? 'border-primary/30 bg-primary/[0.06]'
                        : 'border-transparent hover:bg-primary/[0.05]',
                    )}
                  >
                    <div className="flex min-w-0 items-center gap-2">
                      <span className="truncate text-sm font-medium">
                        {entry.name}
                      </span>
                    </div>
                    <div className="truncate font-mono text-xs text-muted-foreground">
                      {portalEntryAddress(entry)}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {t('portalEntry.ruleCount').replace(
                        '{count}',
                        String(entry.rules.length),
                      )}
                    </div>
                  </a>
                ))}
              </div>
            )}
          </div>
          <ResizableListHandle
            defaultWidth={PORTAL_ENTRY_LIST_DEFAULT_WIDTH}
            label={t('portalEntry.resizeList')}
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
          ) : selectedEntry ? (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b border-border/70 px-6 py-5">
                <div className="flex min-w-0 items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex min-w-0 items-center gap-2">
                      <Compass className="size-4 shrink-0 text-primary" />
                      <h1 className="truncate text-xl font-semibold">
                        {selectedEntry.name}
                      </h1>
                    </div>
                    <div className="mt-1 font-mono text-sm text-muted-foreground">
                      {portalEntryAddress(selectedEntry)}
                    </div>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setEditingEntry(selectedEntry)}
                  >
                    <Edit3 className="size-4" />
                    {t('action.edit')}
                  </Button>
                </div>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto">
                <div className="grid gap-5 px-6 pt-6 pb-6">
                  <section className="grid gap-2">
                    <button
                      type="button"
                      className="sticky top-0 z-20 -mx-6 flex items-center gap-2 bg-white px-6 py-2 text-left"
                      onClick={() => toggleSectionCollapsed('rules')}
                    >
                      <ChevronDown
                        className={cn(
                          'size-3.5 text-muted-foreground transition-transform',
                          collapsedSections.rules && '-rotate-90',
                        )}
                      />
                      <GitBranch className="size-4 text-primary" />
                      <h3 className="text-sm font-semibold">
                        {t('portalEntry.rulesTitle')}
                      </h3>
                      <Badge variant="outline">
                        {selectedEntry.rules.length}
                      </Badge>
                    </button>
                    <div
                      className={cn(
                        'grid gap-3',
                        collapsedSections.rules && 'hidden',
                      )}
                    >
                      {selectedEntry.rules.map((entryRule) => (
                        <div
                          key={entryRule.rule.id}
                          className="grid gap-3 rounded-lg border bg-background p-4"
                        >
                          <div className="flex min-w-0 items-start justify-between gap-3">
                            <div className="min-w-0">
                              <div className="flex min-w-0 items-center gap-2">
                                <a
                                  href={portalRuleHref(entryRule.rule)}
                                  className="truncate text-left text-sm font-semibold transition-colors hover:text-primary hover:underline"
                                  onClick={(event) => {
                                    if (shouldUseBrowserNavigation(event)) {
                                      return
                                    }
                                    event.preventDefault()
                                    jumpToRule(entryRule.rule)
                                  }}
                                >
                                  {entryRule.rule.name}
                                </a>
                              </div>
                            </div>
                            <div className="flex shrink-0 items-center gap-2">
                              <Badge variant="outline">
                                {tText(ruleTargetLabel(entryRule.rule))}
                              </Badge>
                            </div>
                          </div>
                          <div className="flex min-w-0 items-center gap-2 text-sm">
                            <span className="truncate font-mono text-muted-foreground">
                              {formatRuleMatch(entryRule.rule)}
                            </span>
                            <ArrowRight className="size-4 shrink-0 text-muted-foreground" />
                            {entryRule.site ? (
                              <a
                                href={portalSitePath(entryRule.site.id)}
                                className="truncate font-mono transition-colors hover:text-primary hover:underline"
                                onClick={(event) => {
                                  if (shouldUseBrowserNavigation(event)) {
                                    return
                                  }
                                  event.preventDefault()
                                  jumpToSite(entryRule)
                                }}
                              >
                                {ruleTargetValue(entryRule)}
                              </a>
                            ) : (
                              <span className="truncate font-mono">
                                {ruleTargetValue(entryRule)}
                              </span>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </section>
                </div>
              </div>
            </div>
          ) : (
            <Empty className="h-full rounded-none border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Compass />
                </EmptyMedia>
                <EmptyTitle>
                  {visibleEntries.length === 0
                    ? t('portalEntry.empty')
                    : t('portalEntry.noMatch')}
                </EmptyTitle>
                <EmptyDescription>
                  {t('portalEntry.emptyDescription')}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          )}
        </main>
      </div>
      <PortalEntryAccessDialog
        entry={editingEntry}
        open={editingEntry != null}
        onOpenChange={(open) => {
          if (!open) {
            setEditingEntry(null)
          }
        }}
        onUpdated={handleEntryUpdated}
      />
    </section>
  )
}
