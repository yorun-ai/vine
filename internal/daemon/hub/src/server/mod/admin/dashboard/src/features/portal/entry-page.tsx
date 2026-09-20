import { EnabledField } from './enabled-field'
import { invalidateRuleConflicts, useRuleConflicts } from '@/lib/rule-conflicts'
import { useConfigAccess } from '@/lib/config-access'
import { ListDetailFooter } from '@/components/ui/list-detail-layout'
import { SearchInput } from '@/components/ui/search-input'
import * as React from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import {
  ArrowRight,
  ChevronDown,
  Compass,
  Edit3,
  GitBranch,
  Loader2,
  RefreshCw,
  Plus,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { vrpcClient } from '@/config/vrpc-client'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import { createPortalEntryApiService } from '@/skeled/admin'
import type {
  PortalEntry,
  PortalEntryRule,
  PortalEntryUpdate,
  PortalRuleListItem,
} from '@/skeled/admin'

const portalEntryService = createPortalEntryApiService(vrpcClient)
const PORTAL_ENTRY_LIST_DEFAULT_WIDTH = 352
const portalEntrySchemes = ['http', 'https'] as const

interface PortalEntryFormValue {
  name: string
  enabled: boolean
  scheme: string
  host: string
  port: string
}

const newEntryFormValue: PortalEntryFormValue = {
  name: 'http:80',
  enabled: true,
  scheme: 'http',
  host: '',
  port: '80',
}

// derivePortalEntryName mirrors the name Hub derives for an entry it creates on
// its own, so a new entry starts with the label the Dashboard already showed.
function derivePortalEntryName(value: PortalEntryFormValue) {
  const host = value.host.trim()
  const port = value.port.trim() || (value.scheme === 'https' ? '443' : '80')
  return host === '' ? `${value.scheme}:${port}` : `${value.scheme}:${host}:${port}`
}

function syncDerivedEntryName(
  current: PortalEntryFormValue,
  next: PortalEntryFormValue,
) {
  const currentDerivedName = derivePortalEntryName(current)
  if (current.name.trim() === '' || current.name === currentDerivedName) {
    return { ...next, name: derivePortalEntryName(next) }
  }
  return next
}

function updatePortalEntryField(
  current: PortalEntryFormValue,
  field: keyof PortalEntryFormValue,
  value: string | boolean,
) {
  if (field === 'enabled') {
    return { ...current, enabled: value === true }
  }
  if (typeof value !== 'string') {
    return current
  }
  if (field === 'name') {
    if (value.trim() === '') {
      return { ...current, name: derivePortalEntryName(current) }
    }
    return { ...current, name: value }
  }
  return syncDerivedEntryName(current, { ...current, [field]: value })
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function portalEntryToFormValue(entry: PortalEntry): PortalEntryFormValue {
  return {
    name: entry.name,
    enabled: entry.enabled,
    scheme: entry.scheme,
    host: entry.host,
    port: String(entry.port),
  }
}

function portalEntryFormValueToUpdate(
  value: PortalEntryFormValue,
): PortalEntryUpdate {
  return {
    name: value.name.trim(),
    scheme: value.scheme,
    host: value.host.trim(),
    port: Number(value.port),
    enabled: value.enabled,
  }
}

function portalEntryAddress(entry: PortalEntry) {
  return `${entry.scheme}://${entry.host || '*'}:${entry.port}`
}

function isValidPort(value: string) {
  const port = Number(value)
  return Number.isInteger(port) && port >= 0 && port <= 65535
}

function ruleTargetLabel(rule: PortalRuleListItem) {
  switch (rule.routeType) {
    case 'SITE':
      return 'Site'
    case 'PERMANENT_REDIRECT':
      return 'Permanent Redirect'
    case 'TEMPORARY_REDIRECT':
      return 'Temporary Redirect'
    default:
      return rule.routeType
  }
}

function ruleTargetValue(entryRule: PortalEntryRule) {
  const { rule, site } = entryRule
  if (rule.routeType === 'SITE') {
    return `${site?.name ?? rule.routeSiteName} ${rule.routePathPrefix || '/'}`
  }
  return rule.routeRedirectionPattern
}

function formatRuleMatch(rule: PortalRuleListItem) {
  const pathPrefix = rule.matchPathPrefix || '/'
  return rule.matchHost ? `${rule.matchHost}${pathPrefix}` : pathPrefix
}

function portalRuleHref(rule: PortalRuleListItem) {
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

type PortalEntryFormErrors = Partial<
  Record<keyof PortalEntryFormValue, string>
>

function Field({
  children,
  error,
  label,
}: {
  children: React.ReactNode
  error?: string
  label: string
}) {
  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      {children}
      {error ? <div className="text-xs text-destructive">{error}</div> : null}
    </div>
  )
}

function PortalEntryInlineEditor({
  entry,
  saving,
  onCancel,
  onSubmit,
}: {
  entry: PortalEntry | null
  saving: boolean
  onCancel: () => void
  onSubmit: (value: PortalEntryFormValue) => Promise<void>
}) {
  const { t } = useLocale()
  const [formValue, setFormValue] = React.useState<PortalEntryFormValue>(() =>
    entry ? portalEntryToFormValue(entry) : newEntryFormValue,
  )
  const [fieldErrors, setFieldErrors] = React.useState<PortalEntryFormErrors>(
    {},
  )
  const [formError, setFormError] = React.useState<string | null>(null)

  const setField = React.useCallback(
    (field: keyof PortalEntryFormValue, value: string | boolean) => {
      setFormError(null)
      setFormValue((current) => updatePortalEntryField(current, field, value))
    },
    [],
  )

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const errors: PortalEntryFormErrors = {}
      if (formValue.name.trim() === '') {
        errors.name = t('portalEntry.nameRequired')
      }
      if (!isValidPort(formValue.port)) {
        errors.port = t('portalEntry.invalidPort')
      }
      setFieldErrors(errors)
      setFormError(null)
      if (Object.keys(errors).length > 0) {
        return
      }

      try {
        await onSubmit(formValue)
      } catch (error) {
        setFormError(getErrorMessage(error))
      }
    },
    [entry, formValue, onSubmit, t],
  )

  return (
    <form className="grid gap-5" onSubmit={handleSubmit}>
      {formError ? (
        <Alert variant="destructive">
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      <Field label={t('portalEntry.name')} error={fieldErrors.name}>
        <Input
          aria-invalid={Boolean(fieldErrors.name)}
          value={formValue.name}
          placeholder={derivePortalEntryName(formValue)}
          onChange={(event) => setField('name', event.target.value)}
        />
      </Field>
      <Field label={t('portalEntry.scheme')} error={fieldErrors.scheme}>
        <Select
          value={formValue.scheme}
          onValueChange={(value) => setField('scheme', value ?? '')}
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
      </Field>
      <Field label={t('portalEntry.host')}>
        <Input
          value={formValue.host}
          placeholder="*"
          onChange={(event) => setField('host', event.target.value)}
        />
      </Field>
      <Field label={t('portalEntry.port')} error={fieldErrors.port}>
        <Input
          value={formValue.port}
          inputMode="numeric"
          aria-invalid={Boolean(fieldErrors.port)}
          onChange={(event) => setField('port', event.target.value)}
        />
      </Field>

      <EnabledField
        id="portal-entry-enabled"
        enabled={formValue.enabled}
        onChange={(enabled) => setField('enabled', enabled)}
      />

      <div className="flex justify-end gap-2 border-t pt-4">
        <Button
          type="button"
          variant="outline"
          disabled={saving}
          onClick={onCancel}
        >
          {t('action.cancel')}
        </Button>
        <Button type="submit" disabled={saving}>
          {saving ? <Loader2 className="animate-spin" /> : null}
          {t('action.save')}
        </Button>
      </div>
    </form>
  )
}

function PortalEntryDeleteDialog({
  entry,
  open,
  deleting,
  onOpenChange,
  onConfirm,
}: {
  entry: PortalEntry | null
  open: boolean
  deleting: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}) {
  const { t } = useLocale()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('portalEntry.deleteTitle')}</DialogTitle>
          <DialogDescription>
            {t('portalEntry.deleteDescription')}
          </DialogDescription>
        </DialogHeader>
        <div className="rounded-lg border bg-muted/30 px-3 py-2 font-mono text-sm">
          {entry ? portalEntryAddress(entry) : ''}
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={deleting}
            onClick={() => onOpenChange(false)}
          >
            {t('action.cancel')}
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={deleting}
            onClick={onConfirm}
          >
            {deleting ? <Loader2 className="animate-spin" /> : <Trash2 />}
            {t('action.delete')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function PortalEntryPage() {
  const { readOnly } = useConfigAccess()
  const { byRuleId } = useRuleConflicts()
  const queryClient = useQueryClient()
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
  const [saving, setSaving] = React.useState(false)
  const [creating, setCreating] = React.useState(false)
  const [deletingEntry, setDeletingEntry] = React.useState<PortalEntry | null>(
    null,
  )
  const [deleting, setDeleting] = React.useState(false)
  const listPanel = useResizableListPanel({
    defaultWidth: PORTAL_ENTRY_LIST_DEFAULT_WIDTH,
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
      setEntries(loadedEntries.map((entry) => ({
        ...entry,
        rules: entry.rules
          .sort((a, b) =>
            b.rule.matchPathPrefix.length - a.rule.matchPathPrefix.length ||
            a.rule.name.localeCompare(b.rule.name),
          ),
      })))
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

  const filteredEntries = React.useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (keyword === '') {
      return entries
    }

    return entries.filter((entry) => {
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
          entryRule.rule.matchHost,
          entryRule.rule.matchPathPrefix,
          entryRule.rule.routeType,
          entryRule.rule.routeSiteName,
          entryRule.rule.routeRedirectionPattern,
          entryRule.rule.routePathPrefix,
        ]),
      ]

      return values.some((value) => value.toLowerCase().includes(keyword))
    })
  }, [entries, query])

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
    (rule: PortalRuleListItem) => {
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

  const handleCreate = React.useCallback(
    async (value: PortalEntryFormValue) => {
      setSaving(true)
      try {
        const created = await portalEntryService.create({
          creation: {
            name: value.name.trim(),
            scheme: value.scheme,
            host: value.host.trim(),
            port: Number(value.port),
            enabled: value.enabled,
          },
        })
        toast.success(t('portalEntry.createSuccess'))
        setCreating(false)
        invalidateRuleConflicts(queryClient)
        await loadEntries()
        selectEntry(created.name, true)
      } finally {
        setSaving(false)
      }
    },
    [loadEntries, selectEntry, t],
  )

  const handleUpdate = React.useCallback(
    async (value: PortalEntryFormValue) => {
      if (editingEntry == null) {
        return
      }
      setSaving(true)
      try {
        const updated = await portalEntryService.update({
          id: editingEntry.id,
          update: portalEntryFormValueToUpdate(value),
        })
        toast.success(t('portalEntry.updateSuccess'))
        setEditingEntry(null)
        invalidateRuleConflicts(queryClient)
        await loadEntries()
        selectEntry(updated.name, true)
      } finally {
        setSaving(false)
      }
    },
    [editingEntry, loadEntries, selectEntry, t],
  )

  const removeEntry = React.useCallback(async () => {
    if (deletingEntry == null) {
      return
    }
    setDeleting(true)
    try {
      await portalEntryService.remove({
        id: deletingEntry.id,
      })
      toast.success(t('portalEntry.deleteSuccess'))
      setDeletingEntry(null)
      invalidateRuleConflicts(queryClient)
      await loadEntries()
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setDeleting(false)
    }
  }, [deletingEntry, loadEntries, t])

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
    if (loading) {
      return
    }
    if (filteredEntries.length === 0) {
      setSelectedEntryName(null)
      return
    }

    if (!filteredEntries.some((entry) => entry.name === selectedEntryName)) {
      selectEntry(filteredEntries[0].name, true)
    }
  }, [filteredEntries, loading, pathname, selectEntry, selectedEntryName])
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
    <TooltipProvider>
      <section className="flex h-[calc(100dvh-3.5rem)] flex-col overflow-hidden bg-white">
      <div
        className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[var(--list-panel-width)_minmax(0,1fr)]"
        style={listPanel.gridStyle}
      >
        <aside className="relative flex min-h-0 flex-col border-b border-border/70 lg:border-r lg:border-b-0">
          <div className="grid gap-4 border-b border-border/70 p-4">
            <div className="relative w-full md:max-w-sm">
              <SearchInput
                value={query}
                placeholder={t('portalEntry.searchPlaceholder')}
                onValueChange={setQuery}
              />
            </div>
            <div className="flex items-center justify-end gap-2">
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
                  <Button
                    type="button"
                    size="sm"
                    disabled={readOnly}
                    onClick={() => setCreating(true)}
                  >
                    <Plus />
                    {t('action.create')}
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
                {entries.length === 0
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
                        {entry.enabled ? null : (
                          <Badge variant="secondary">{t('common.disabled')}</Badge>
                        )}
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
          <ListDetailFooter>
            {t('portalEntry.itemCount').replace(
              '{count}',
              String(entries.length),
            )}
          </ListDetailFooter>
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
          ) : creating ? (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b border-border/70 px-6 py-4">
                <div className="flex items-center gap-2">
                  <Compass className="size-4 shrink-0 text-primary" />
                  <h2 className="text-base font-semibold">
                    {t('portalEntry.createTitle')}
                  </h2>
                </div>
              </div>
              <div className="scrollbar-reserved min-h-0 flex-1 overflow-y-auto p-6 pr-4">
                <PortalEntryInlineEditor
                  entry={null}
                  saving={saving}
                  onCancel={() => setCreating(false)}
                  onSubmit={handleCreate}
                />
              </div>
            </div>
          ) : selectedEntry ? (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b border-border/70 px-6 py-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex min-w-0 items-center gap-2">
                      <Compass className="size-4 shrink-0 text-primary" />
                      <h2 className="truncate text-base font-semibold">
                        {selectedEntry.name}
                      </h2>
                      {selectedEntry.enabled ? null : (
                        <Badge variant="secondary">{t('common.disabled')}</Badge>
                      )}
                    </div>
                    <p className="mt-2 font-mono text-xs text-muted-foreground">
                      {portalEntryAddress(selectedEntry)}
                    </p>
                  </div>
                  {editingEntry?.name === selectedEntry.name ? null : (
                    <div className="flex items-center gap-2">
                      <Tooltip>
                        <TooltipTrigger render={<span className="inline-flex" />}>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={readOnly}
                            onClick={() => setEditingEntry(selectedEntry)}
                          >
                            <Edit3 />
                            {t('action.edit')}
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>{t('action.edit')}</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger render={<span className="inline-flex" />}>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon-sm"
                            disabled={
                              readOnly || selectedEntry.rules.length > 0
                            }
                            title={
                              selectedEntry.rules.length > 0
                                ? t('portalEntry.deleteWithRules')
                                : undefined
                            }
                            onClick={() => setDeletingEntry(selectedEntry)}
                          >
                            <Trash2 />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>
                          {selectedEntry.rules.length > 0
                            ? t('portalEntry.deleteWithRules')
                            : t('action.delete')}
                        </TooltipContent>
                      </Tooltip>
                    </div>
                  )}
                </div>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto">
                <div className="grid gap-5 px-6 pt-6 pb-6">
                  {editingEntry?.name === selectedEntry.name ? (
                    <PortalEntryInlineEditor
                      entry={selectedEntry}
                      saving={saving}
                      onCancel={() => setEditingEntry(null)}
                      onSubmit={handleUpdate}
                    />
                  ) : (
                    <>
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
                        {selectedEntry.rules.length === 0 ? (
                          <div className="rounded-lg border border-dashed px-4 py-6 text-center">
                            <div className="text-sm font-medium">
                              {t('portalEntry.noRules')}
                            </div>
                            <div className="mt-1 text-xs text-muted-foreground">
                              {t('portalEntry.noRulesDescription')}
                            </div>
                          </div>
                        ) : null}
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
                                  {entryRule.rule.enabled ? null : (
                                    <Badge variant="secondary">
                                      {t('common.disabled')}
                                    </Badge>
                                  )}
                                  {byRuleId.has(entryRule.rule.id) ? (
                                    <Badge variant="destructive">
                                      {t('portalRule.conflict')}
                                    </Badge>
                                  ) : null}
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
                    </>
                  )}
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
                  {entries.length === 0
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
      <PortalEntryDeleteDialog
        entry={deletingEntry}
        open={deletingEntry != null}
        deleting={deleting}
        onOpenChange={(open) => {
          if (!open) {
            setDeletingEntry(null)
          }
        }}
        onConfirm={() => void removeEntry()}
      />
      </section>
    </TooltipProvider>
  )
}
