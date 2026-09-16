import { FieldSourceInfo } from '@/features/field-source/field-source-info'
import { EnabledField } from './enabled-field'
import { useConfigAccess } from '@/lib/config-access'
import { RulePathPreview } from './rule-path-preview'
import {
  effectiveWebMountPrefixes,
  lockedWebMountPath,
  rulePathsForSave,
} from './web-mount-path'
import { ListDetailFooter } from '@/components/ui/list-detail-layout'
import { SearchInput } from '@/components/ui/search-input'
import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import {
  ArrowRight,
  Boxes,
  Edit3,
  Loader2,
  Plus,
  RefreshCw,
  ShieldCheck,
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
  EmptyContent,
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
import {
  createPortalEntryApiService,
  createPortalRuleApiService,
  createPortalSiteApiService,
} from '@/skeled/admin'
import type {
  PortalEntry,
  PortalRule,
  PortalRuleCreation,
  PortalRuleListItem,
  PortalRuleUpdate,
  PortalSiteListItem,
} from '@/skeled/admin'

const portalEntryService = createPortalEntryApiService(vrpcClient)
const portalRuleService = createPortalRuleApiService(vrpcClient)
const portalSiteService = createPortalSiteApiService(vrpcClient)
const PORTAL_RULE_LIST_DEFAULT_WIDTH = 352
const routeTypes = [
  {
    value: 'SITE',
    label: 'Site',
    description: 'Forward to Portal Site',
  },
  {
    value: 'PERMANENT_REDIRECT',
    label: 'Permanent Redirect',
    description: 'Return 308 redirect',
  },
  {
    value: 'TEMPORARY_REDIRECT',
    label: 'Temporary Redirect',
    description: 'Return 307 redirect',
  },
] as const

type PortalRuleRouteType = (typeof routeTypes)[number]['value']
type PortalRuleFormErrors = Partial<Record<keyof PortalRuleFormValue, string>>

interface PortalRuleFormValue {
  name: string
  enabled: boolean
  matchScheme: string
  matchHost: string
  matchPort: string
  matchPathPrefix: string
  routeType: PortalRuleRouteType
  routeSiteName: string
  routeRedirectionPattern: string
  routePathPrefix: string
}

const emptyFormValue: PortalRuleFormValue = {
  name: '',
  enabled: true,
  matchScheme: 'http',
  matchHost: '',
  matchPort: '',
  matchPathPrefix: '',
  routeType: 'SITE',
  routeSiteName: '',
  routeRedirectionPattern: '',
  routePathPrefix: '',
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function ruleToFormValue(rule: PortalRuleListItem): PortalRuleFormValue {
  return {
    name: rule.name,
    enabled: rule.enabled,
    matchScheme: rule.matchScheme,
    matchHost: rule.matchHost,
    matchPort: rule.matchPort === 0 ? '' : String(rule.matchPort),
    matchPathPrefix: rule.matchPathPrefix,
    routeType: normalizeTargetType(rule.routeType),
    routeSiteName: rule.routeSiteName,
    routeRedirectionPattern: rule.routeRedirectionPattern,
    routePathPrefix: rule.routePathPrefix ?? '',
  }
}

function normalizeTargetType(value: string): PortalRuleRouteType {
  return routeTypes.some((item) => item.value === value)
    ? (value as PortalRuleRouteType)
    : 'SITE'
}

function normalizeRuleNamePart(value: string, fallback: string) {
  const normalized = value
    .trim()
    .replace(/^\/+|\/+$/g, '')
    .replace(/[/:]+/g, '.')
  return normalized === '' ? fallback : normalized
}

function derivePortalRuleName(value: PortalRuleFormValue) {
  const target =
    value.routeType === 'SITE' ? value.routeSiteName : value.routeRedirectionPattern

  return [
    normalizeRuleNamePart(value.matchScheme, 'http'),
    normalizeRuleNamePart(value.matchHost, 'all'),
    normalizeRuleNamePart(value.matchPort, 'auto'),
    normalizeRuleNamePart(value.matchPathPrefix, 'root'),
    normalizeRuleNamePart(target, value.routeType.toLowerCase()),
  ].join('.')
}

function syncDerivedName(
  current: PortalRuleFormValue,
  next: PortalRuleFormValue,
) {
  const currentDerivedName = derivePortalRuleName(current)
  if (current.name.trim() === '' || current.name === currentDerivedName) {
    return {
      ...next,
      name: derivePortalRuleName(next),
    }
  }

  return next
}

function updatePortalRuleField(
  current: PortalRuleFormValue,
  field: keyof PortalRuleFormValue,
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
      return {
        ...current,
        name: derivePortalRuleName(current),
      }
    }

    return {
      ...current,
      name: value,
    }
  }

  return syncDerivedName(current, {
    ...current,
    [field]: value,
  })
}

// ruleFormTargetValue returns the fields a rule owns: the Portal entry owns the
// access, so the target and the path are everything a rule declares.
function ruleFormTargetValue(value: PortalRuleFormValue) {
  return {
    name: value.name.trim() || derivePortalRuleName(value),
    enabled: value.enabled,
    matchPathPrefix: value.matchPathPrefix.trim(),
    routeType: value.routeType,
    routeSiteName: value.routeType === 'SITE' ? value.routeSiteName.trim() : '',
    routeRedirectionPattern:
      value.routeType === 'SITE' ? '' : value.routeRedirectionPattern.trim(),
    routePathPrefix: value.routeType === 'SITE' ? value.routePathPrefix.trim() : '',
  }
}

function formValueToCreation(
  value: PortalRuleFormValue,
  accessEntries: Array<PortalEntry>,
): PortalRuleCreation {
  const entry = portalRuleFormAccessEntry(value, accessEntries)

  return {
    ...ruleFormTargetValue(value),
    entryName: entry?.name ?? '',
  }
}

function formValueToUpdate(value: PortalRuleFormValue): PortalRuleUpdate {
  return {
    ...ruleFormTargetValue(value),
  }
}

function formatMatch(rule: PortalRuleListItem) {
  const matchPort = rule.matchPort === 0 ? '' : `:${rule.matchPort}`
  const matchHost = rule.matchHost || '*'
  const matchPathPrefix = rule.matchPathPrefix || '/'

  return `${rule.matchScheme}://${matchHost}${matchPort}${matchPathPrefix}`
}

function routeTypeLabel(routeType: string) {
  return (
    routeTypes.find((item) => item.value === routeType)?.label ?? routeType
  )
}

function isRedirectTarget(routeType: string) {
  return (
    routeType === 'PERMANENT_REDIRECT' || routeType === 'TEMPORARY_REDIRECT'
  )
}

function validateFormValue(
  value: PortalRuleFormValue,
  accessEntries: Array<PortalEntry>,
  t: ReturnType<typeof useLocale>['t'],
) {
  const errors: PortalRuleFormErrors = {}

  if (value.name.trim() === '') {
    errors.name = t('portalRule.nameRequired')
  }

  // The rule matches the access of its Portal entry, so a rule without an entry
  // has no request to match.
  if (portalRuleFormAccessEntry(value, accessEntries) === null) {
    errors.matchScheme = t('portalRule.entryRequired')
  }

  if (value.routeType === 'SITE' && value.routePathPrefix.trim() !== '') {
    const path = value.routePathPrefix.trim()
    try {
      const decoded = decodeURIComponent(path)
      if (
        !path.startsWith('/') ||
        path.startsWith('//') ||
        /[?#\s]/.test(path) ||
        /[\\\x00-\x1f\x7f]/.test(decoded) ||
        decoded.split('/').some((part) => part === '.' || part === '..')
      ) {
        errors.routePathPrefix = t('portalRule.routePathPrefixInvalid')
      }
    } catch {
      errors.routePathPrefix = t('portalRule.routePathPrefixInvalid')
    }
  }

  if (value.routeType === 'SITE' && value.routeSiteName.trim() === '') {
    errors.routeSiteName = t('portalRule.siteRequired')
  }

  if (
    isRedirectTarget(value.routeType) &&
    value.routeRedirectionPattern.trim() === ''
  ) {
    errors.routeRedirectionPattern = t('portalRule.redirectRequired')
  }

  return errors
}

function hasFormErrors(errors: PortalRuleFormErrors) {
  return Object.keys(errors).length > 0
}

function portalRulePath(id: number) {
  return `/portal/rule/${id}`
}

function portalEntryPath(name: string) {
  return `/portal/entry/${encodeURIComponent(name)}`
}

const portalEntryListPath = '/portal/entry'

// portalRuleAccessEntry returns the entry that owns the access a rule matches.
// Protocol, host, and port belong to the entry, so the rule page displays them
// and links to the entry instead of editing them here.
function portalRuleAccessEntry(
  rule: PortalRuleListItem,
  accessEntries: Array<PortalEntry>,
) {
  return (
    accessEntries.find(
      (entry) =>
        entry.scheme === rule.matchScheme &&
        entry.host === rule.matchHost &&
        entry.port === rule.matchPort,
    ) ?? null
  )
}

function portalRuleFormAccessEntry(
  value: PortalRuleFormValue,
  accessEntries: Array<PortalEntry>,
) {
  const port = Number(value.matchPort)
  return (
    accessEntries.find(
      (entry) =>
        entry.scheme === value.matchScheme &&
        entry.host === value.matchHost &&
        entry.port === port,
    ) ?? null
  )
}

function portalRuleListItemDomId(id: number) {
  return `portal-rule-list-item:${id}`
}

function portalSitePath(id: number) {
  return `/portal/site/${id}`
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

function selectedRuleIdFromPath(pathname: string) {
  const match = pathname.match(/^\/portal\/rule\/(\d+)$/)
  if (!match) {
    return null
  }
  const id = Number(match[1])
  return Number.isInteger(id) && id > 0 ? id : null
}

function isPortalRulePath(pathname: string) {
  return pathname === '/portal/rule' || pathname.startsWith('/portal/rule/')
}

function selectedRuleIdFromSearch() {
  const value = new URLSearchParams(window.location.search).get('ruleId')
  const id = Number(value)
  return Number.isInteger(id) && id > 0 ? id : null
}

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

function RuleFlowSection({
  children,
  description,
  title,
}: {
  children: React.ReactNode
  description: string
  title: string
}) {
  return (
    <section className="grid gap-4">
      <div className="grid gap-1 border-b pb-3">
        <h3 className="text-sm font-semibold tracking-normal">{title}</h3>
        <p className="text-xs leading-5 text-muted-foreground">{description}</p>
      </div>
      <div className="grid gap-4">{children}</div>
    </section>
  )
}

function TargetTypeBadge({
  routeType,
  className,
}: {
  routeType: string
  className?: string
}) {
  const { tText } = useLocale()
  return (
    <Badge variant="outline" className={className}>
      {tText(routeTypeLabel(routeType))}
    </Badge>
  )
}

function DeleteRuleDialog({
  open,
  rule,
  deleting,
  onOpenChange,
  onConfirm,
}: {
  open: boolean
  rule: PortalRuleListItem | null
  deleting: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}) {
  const { t } = useLocale()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('portalRule.deleteTitle')}</DialogTitle>
          <DialogDescription>
            {t('portalRule.deleteDescription')}
          </DialogDescription>
        </DialogHeader>
        <div className="rounded-lg border bg-muted/30 px-3 py-2 text-sm">
          {rule?.name}
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

function PortalRuleListSkeleton() {
  return (
    <div className="grid gap-2">
      {Array.from({ length: 5 }).map((_, index) => (
        <Skeleton key={index} className="h-16 w-full" />
      ))}
    </div>
  )
}

function ReadonlyField({
  label,
  source,
  children,
  className,
}: {
  source: React.ReactNode
  label: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <div className="grid gap-2">
      <Label className="flex items-center gap-1.5">{label}{source}</Label>
      <div
        className={cn(
          'min-h-9 py-2 text-sm',
          className,
        )}
      >
        {children}
      </div>
    </div>
  )
}

function PortalRuleInlineEditor({
  rule,
  saving,
  entries,
  accessEntries,
  onCancel,
  onSubmit,
}: {
  rule: PortalRuleListItem | null
  saving: boolean
  entries: Array<PortalSiteListItem>
  accessEntries: Array<PortalEntry>
  onCancel: () => void
  onSubmit: (value: PortalRuleFormValue) => Promise<void>
}) {
  const { t, tText } = useLocale()
  const navigate = useNavigate()
  const [formValue, setFormValue] = React.useState<PortalRuleFormValue>(() => {
    const nextFormValue = rule ? ruleToFormValue(rule) : emptyFormValue
    return {
      ...nextFormValue,
      name: nextFormValue.name || derivePortalRuleName(nextFormValue),
    }
  })
  const [fieldErrors, setFieldErrors] = React.useState<PortalRuleFormErrors>({})
  const [formError, setFormError] = React.useState<string | null>(null)
  // Protocol, host, and port belong to the entry, so the editor only chooses an
  // entry: a new rule picks one, and an existing rule shows the entry it has.
  const accessEntry = React.useMemo(
    () => portalRuleFormAccessEntry(formValue, accessEntries),
    [accessEntries, formValue],
  )
  const selectAccessEntry = React.useCallback(
    (name: string) => {
      const entry = accessEntries.find((item) => item.name === name)
      if (entry === undefined) {
        return
      }
      setFormError(null)
      setFieldErrors((current) => {
        const { matchScheme: _matchScheme, ...next } = current
        return next
      })
      // Route the access through the shared field update so the derived rule
      // name keeps following the entry.
      setFormValue((current) => {
        const withScheme = updatePortalRuleField(
          current,
          'matchScheme',
          entry.scheme,
        )
        const withHost = updatePortalRuleField(
          withScheme,
          'matchHost',
          entry.host,
        )
        return updatePortalRuleField(withHost, 'matchPort', String(entry.port))
      })
    },
    [accessEntries],
  )
  const entryOptions = React.useMemo(() => {
    if (
      formValue.routeSiteName === '' ||
      entries.some((entry) => entry.name === formValue.routeSiteName)
    ) {
      return entries
    }

    return [
      ...entries,
      {
        id: 0,
        name: formValue.routeSiteName,
        type: '',
        actorSkelName: '',
        actorVia: '',
        rpcgwServices: [],
        webName: '',
      },
    ]
  }, [formValue.routeSiteName, entries])

  React.useEffect(() => {
    const nextFormValue = rule ? ruleToFormValue(rule) : emptyFormValue
    setFormValue({
      ...nextFormValue,
      name: nextFormValue.name || derivePortalRuleName(nextFormValue),
    })
    setFieldErrors({})
    setFormError(null)
  }, [rule])

  const lockedMountPath = React.useMemo(
    () =>
      lockedWebMountPath(
        formValue.routeType,
        formValue.routeSiteName,
        entries,
      ),
    [formValue.routeSiteName, formValue.routeType, entries],
  )

  const setField = React.useCallback(
    (field: keyof PortalRuleFormValue, value: string | boolean) => {
      setFormError(null)
      setFieldErrors((current) => {
        if (!current[field]) {
          return current
        }

        const { [field]: _removed, ...next } = current
        return next
      })
      setFormValue((current) => updatePortalRuleField(current, field, value))
    },
    [],
  )

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const nextValue = {
        ...formValue,
        name: formValue.name.trim() || derivePortalRuleName(formValue),
      }
      const errors = validateFormValue(nextValue, accessEntries, t)
      if (hasFormErrors(errors)) {
        setFieldErrors(errors)
        return
      }

      setFieldErrors({})
      setFormError(null)

      try {
        await onSubmit(rulePathsForSave(nextValue, lockedMountPath))
      } catch (error) {
        setFormError(getErrorMessage(error))
      }
    },
    [formValue, lockedMountPath, onSubmit],
  )

  return (
    <form className="grid gap-5" onSubmit={handleSubmit}>
      {formError ? (
        <Alert variant="destructive">
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      {rule === null && accessEntries.length === 0 ? (
        <Alert>
          <AlertDescription>
            {t('portalRule.noEntry')}{' '}
            <a
              href={portalEntryListPath}
              className="text-primary hover:underline"
              onClick={(event) => {
                if (shouldUseBrowserNavigation(event)) {
                  return
                }
                event.preventDefault()
                void navigate({ to: portalEntryListPath })
              }}
            >
              {t('portalRule.openEntry')}
            </a>
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="max-w-xl">
        <Field label={t('portalRule.name')} error={fieldErrors.name}>
          <Input
            aria-invalid={Boolean(fieldErrors.name)}
            value={formValue.name}
            placeholder={derivePortalRuleName(formValue)}
            onChange={(event) => setField('name', event.target.value)}
          />
        </Field>
      </div>

      <div className="grid gap-5 md:grid-cols-[minmax(0,1fr)_2.5rem_minmax(0,1fr)] md:items-start">
        <RuleFlowSection
          title={t('portalRule.match')}
          description={t('portalRule.matchDescription')}
        >
          <Field label={t('portalRule.entry')} error={fieldErrors.matchScheme}>
            {rule === null ? (
              <Select
                value={accessEntry?.name ?? ''}
                onValueChange={(value) => {
                  if (value) {
                    selectAccessEntry(value)
                  }
                }}
              >
                <SelectTrigger
                  aria-invalid={Boolean(fieldErrors.matchScheme)}
                  className="w-full"
                >
                  <SelectValue placeholder={t('portalRule.selectEntry')} />
                </SelectTrigger>
                <SelectContent align="start">
                  {accessEntries.map((entry) => (
                    <SelectItem key={entry.name} value={entry.name}>
                      <span className="flex flex-col">
                        <span>{entry.name}</span>
                        <span className="text-xs text-muted-foreground">
                          {t('portalEntry.ruleCount').replace(
                            '{count}',
                            String(entry.rules.length),
                          )}
                        </span>
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <div className="font-mono text-sm">
                {accessEntry?.name ?? t('portalRule.entryMissing')}
              </div>
            )}
            <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <span>{t('portalRule.entryHelp')}</span>
              {accessEntry === null ? null : (
                <a
                  href={portalEntryPath(accessEntry.name)}
                  className="inline-flex items-center gap-1 text-primary hover:underline"
                  onClick={(event) => {
                    if (shouldUseBrowserNavigation(event)) {
                      return
                    }
                    event.preventDefault()
                    void navigate({ to: portalEntryPath(accessEntry.name) })
                  }}
                >
                  <Boxes className="size-3.5" />
                  {t('portalRule.openEntry')}
                </a>
              )}
            </div>
          </Field>

          <Field label={t('portalRule.matchPathPrefix')}>
            <Input
              className={lockedMountPath === null ? undefined : 'text-destructive'}
              value={formValue.matchPathPrefix}
              placeholder={t('portalRule.pathPrefixPlaceholder')}
              onChange={(event) => setField('matchPathPrefix', event.target.value)}
            />
          </Field>
        </RuleFlowSection>

        <div className="flex justify-center pt-0 md:pt-20">
          <div className="grid size-10 place-items-center rounded-full border bg-background text-muted-foreground shadow-xs">
            <ArrowRight className="size-4 rotate-90 md:rotate-0" />
          </div>
        </div>

        <RuleFlowSection
          title={t('portalRule.route')}
          description={t('portalRule.routeDescription')}
        >
          <Field
            label={t('portalRule.routeType')}
            error={fieldErrors.routeType}
          >
            <Select
              value={formValue.routeType}
              onValueChange={(value) => {
                if (value) {
                  setField('routeType', normalizeTargetType(value))
                }
              }}
            >
              <SelectTrigger
                aria-invalid={Boolean(fieldErrors.routeType)}
                className="w-full"
              >
                <SelectValue placeholder={t('portalRule.selectRouteType')} />
              </SelectTrigger>
              <SelectContent align="start">
                {routeTypes.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    <span className="flex flex-col">
                      <span>{tText(item.label)}</span>
                      <span className="text-xs text-muted-foreground">
                        {tText(item.description)}
                      </span>
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>

          {formValue.routeType === 'SITE' ? (
            <Field label={t('portalRule.routeSiteName')} error={fieldErrors.routeSiteName}>
              <Select
                value={formValue.routeSiteName}
                onValueChange={(value) => {
                  if (value) {
                    setField('routeSiteName', value)
                  }
                }}
              >
                <SelectTrigger
                  aria-invalid={Boolean(fieldErrors.routeSiteName)}
                  className="w-full"
                >
                  <SelectValue placeholder={t('portalRule.selectSite')} />
                </SelectTrigger>
                <SelectContent align="start">
                  {entryOptions.map((entry) => (
                    <SelectItem
                      key={`${entry.id}:${entry.name}`}
                      value={entry.name}
                    >
                      <span className="flex flex-col">
                        <span>{entry.name}</span>
                        <span className="text-xs text-muted-foreground">
                          {entry.id === 0
                            ? t('portalRule.siteNotFound')
                            : entry.type === 'RPCGW'
                              ? t('portalSite.rpcGateway')
                              : entry.type === 'WEBGW'
                                ? t('portalSite.webGateway')
                                : entry.type}
                        </span>
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
          ) : (
            <Field
              label={t('portalRule.redirectPattern')}
              error={fieldErrors.routeRedirectionPattern}
            >
              <Input
                aria-invalid={Boolean(fieldErrors.routeRedirectionPattern)}
                value={formValue.routeRedirectionPattern}
                placeholder="https://example.com{uri}"
                onChange={(event) =>
                  setField('routeRedirectionPattern', event.target.value)
                }
              />
            </Field>
          )}
          {formValue.routeType === 'SITE' ? (
            <Field label={t('portalRule.routePathPrefix')} error={fieldErrors.routePathPrefix}>
              <Input
                className={lockedMountPath === null ? undefined : 'text-destructive'}
                value={formValue.routePathPrefix}
                placeholder="/internal"
                aria-invalid={Boolean(fieldErrors.routePathPrefix)}
                onChange={(event) => setField('routePathPrefix', event.target.value)}
              />
              <p className="text-xs text-muted-foreground">{t('portalRule.routePathPrefixHelp')}</p>
            </Field>
          ) : null}
        </RuleFlowSection>
      </div>
      {formValue.routeType === 'SITE' ? (
        <RulePathPreview
          mountPath={lockedMountPath}
          matchPathPrefix={(lockedMountPath === null ? formValue.matchPathPrefix : effectiveWebMountPrefixes(lockedMountPath).matchPathPrefix).trim()}
          routePathPrefix={(lockedMountPath === null ? formValue.routePathPrefix : effectiveWebMountPrefixes(lockedMountPath).routePathPrefix).trim()}
        />
      ) : null}

      <EnabledField
        id="portal-rule-enabled"
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

export function PortalRulePage() {
  const { readOnly } = useConfigAccess()
  const { t } = useLocale()
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const [rules, setRules] = React.useState<Array<PortalRuleListItem>>([])
  const [entries, setEntries] = React.useState<Array<PortalSiteListItem>>([])
  const [accessEntries, setAccessEntries] = React.useState<Array<PortalEntry>>(
    [],
  )
  const [query, setQuery] = React.useState('')
  const listPanel = useResizableListPanel({
    defaultWidth: PORTAL_RULE_LIST_DEFAULT_WIDTH,
  })
  const handleListScroll = useReservedScrollbar()
  const [loading, setLoading] = React.useState(true)
  const [saving, setSaving] = React.useState(false)
  const [deleting, setDeleting] = React.useState(false)
  const [editingRule, setEditingRule] = React.useState<PortalRuleListItem | null>(null)
  const [deleteRule, setDeleteRule] = React.useState<PortalRuleListItem | null>(null)
  const [isCreating, setIsCreating] = React.useState(false)
  const [selectedRuleId, setSelectedRuleId] = React.useState<number | null>(
    () =>
      selectedRuleIdFromPath(window.location.pathname) ??
      selectedRuleIdFromSearch(),
  )

  const loadRules = React.useCallback(async () => {
    setLoading(true)

    try {
      const [nextRules, nextEntries, nextAccessEntries] = await Promise.all([
        portalRuleService.list(null),
        portalSiteService.list(null),
        portalEntryService.list(null),
      ])
      setRules(nextRules)
      setEntries(nextEntries)
      setAccessEntries(nextAccessEntries)
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void loadRules()
  }, [loadRules])

  const visibleRules = React.useMemo(() => {
    return rules.map((rule) => ({
      ...rule,
      matchPathPrefix: rule.resolvedMatchPathPrefix,
      routePathPrefix: rule.resolvedRoutePathPrefix,
    }))
  }, [rules])

  const filteredRules = React.useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (keyword === '') {
      return visibleRules
    }

    return visibleRules.filter((rule) => {
      const values = [
        rule.name,
        rule.matchScheme,
        rule.matchHost,
        String(rule.matchPort),
        rule.matchPathPrefix,
        rule.routeType,
        rule.routeSiteName,
        rule.routeRedirectionPattern,
        rule.routePathPrefix,
      ]

      return values.some((value) => value.toLowerCase().includes(keyword))
    })
  }, [query, visibleRules])

  const selectedRule = React.useMemo(
    () =>
      filteredRules.find((rule) => rule.id === selectedRuleId) ??
      filteredRules[0] ??
      null,
    [filteredRules, selectedRuleId],
  )
  const rawSelectedRule = React.useMemo(
    () =>
      (selectedRule && rules.find((rule) => rule.id === selectedRule.id)) ??
      selectedRule,
    [rules, selectedRule],
  )
  const selectedTargetSite = React.useMemo(() => {
    if (!selectedRule || selectedRule.routeType !== 'SITE') {
      return null
    }
    return entries.find((entry) => entry.name === selectedRule.routeSiteName) ?? null
  }, [entries, selectedRule])

  // Protocol, host, and port belong to the Portal entry that owns the access, so
  // the detail view links to it instead of offering an edit here.
  const selectedAccessEntry = React.useMemo(
    () =>
      selectedRule === null
        ? null
        : portalRuleAccessEntry(selectedRule, accessEntries),
    [accessEntries, selectedRule],
  )

  const selectedMountPath = selectedRule
    ? lockedWebMountPath(selectedRule.routeType, selectedRule.routeSiteName, entries)
    : null

  // The list returns rules without their seed provenance, so the detail view
  // reads the selected rule once more.
  const [ruleDetail, setRuleDetail] = React.useState<PortalRule | null>(null)

  React.useEffect(() => {
    const id = selectedRule?.id
    if (id == null) {
      setRuleDetail(null)
      return
    }

    let active = true
    portalRuleService.get({ id }).then(
      (rule) => {
        if (active) {
          setRuleDetail(rule)
        }
      },
      () => {
        if (active) {
          setRuleDetail(null)
        }
      },
    )
    return () => {
      active = false
    }
  }, [selectedRule?.id])

  const selectedRuleFields =
    ruleDetail !== null && ruleDetail.id === selectedRule?.id
      ? ruleDetail.fieldSources
      : []

  const selectRule = React.useCallback(
    (id: number, replace = false) => {
      setIsCreating(false)
      setEditingRule(null)
      setSelectedRuleId(id)
      void navigate({ replace, to: portalRulePath(id) })
    },
    [navigate],
  )

  React.useEffect(() => {
    if (!isPortalRulePath(pathname)) {
      return
    }
    setSelectedRuleId(selectedRuleIdFromPath(pathname))
  }, [pathname])

  React.useEffect(() => {
    if (!isPortalRulePath(pathname)) {
      return
    }
    if (loading) {
      return
    }
    if (filteredRules.length === 0) {
      setSelectedRuleId(null)
      return
    }

    if (!filteredRules.some((rule) => rule.id === selectedRuleId)) {
      selectRule(filteredRules[0].id, true)
    }
  }, [filteredRules, loading, pathname, selectRule, selectedRuleId])
  React.useEffect(() => {
    if (selectedRuleId == null) {
      return
    }
    window.requestAnimationFrame(() => {
      document
        .getElementById(portalRuleListItemDomId(selectedRuleId))
        ?.scrollIntoView({
          block: 'nearest',
          inline: 'nearest',
        })
    })
  }, [filteredRules, selectedRuleId])

  const handleCreate = React.useCallback(
    async (value: PortalRuleFormValue) => {
      setSaving(true)

      try {
        const created = await portalRuleService.create({
          creation: formValueToCreation(value, accessEntries),
        })
        toast.success(t('portalRule.created'))
        setIsCreating(false)
        setRules((current) => [...current, created])
        setRuleDetail(created)
        selectRule(created.id)
      } catch (error) {
        throw error
      } finally {
        setSaving(false)
      }
    },
    [accessEntries, selectRule],
  )

  const handleUpdate = React.useCallback(
    async (value: PortalRuleFormValue) => {
      if (!editingRule) {
        return
      }

      setSaving(true)

      try {
        const updated = await portalRuleService.update({
          id: editingRule.id,
          update: formValueToUpdate(value),
        })
        toast.success(t('portalRule.saved'))
        setEditingRule(null)
        setRuleDetail(updated)
        setRules((current) =>
          current.map((rule) => (rule.id === updated.id ? updated : rule)),
        )
      } catch (error) {
        throw error
      } finally {
        setSaving(false)
      }
    },
    [editingRule],
  )

  const handleDelete = React.useCallback(async () => {
    if (!deleteRule) {
      return
    }

    setDeleting(true)

    try {
      await portalRuleService.remove({ id: deleteRule.id })
      toast.success(t('portalRule.deleted'))
      setDeleteRule(null)
      setRules((current) => current.filter((rule) => rule.id !== deleteRule.id))
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setDeleting(false)
    }
  }, [deleteRule])

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
                  placeholder={t('portalRule.searchPlaceholder')}
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
                    onClick={() => void loadRules()}
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
                    onClick={() => {
                      setEditingRule(null)
                      setIsCreating(true)
                    }}
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
                <PortalRuleListSkeleton />
              ) : filteredRules.length === 0 ? (
                <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                  {visibleRules.length === 0
                    ? t('portalRule.empty')
                    : t('portalRule.noMatch')}
                </div>
              ) : (
                <div className="space-y-1">
                  {filteredRules.map((rule) => (
                    <a
                      key={rule.id}
                      id={portalRuleListItemDomId(rule.id)}
                      href={portalRulePath(rule.id)}
                      onClick={(event) => {
                        if (shouldUseBrowserNavigation(event)) {
                          return
                        }
                        event.preventDefault()
                        selectRule(rule.id)
                      }}
                      className={cn(
                        'relative flex w-full flex-col gap-1 rounded-lg border px-3 py-2.5 pr-16 text-left transition-colors',
                        selectedRule?.id === rule.id
                          ? 'border-primary/30 bg-primary/[0.06]'
                          : 'border-transparent hover:bg-primary/[0.05]',
                      )}
                    >
                      <TargetTypeBadge
                        routeType={rule.routeType}
                        className="absolute top-2.5 right-3"
                      />
                      <div className="flex min-w-0 items-center gap-2">
                        <span className="truncate text-sm font-medium">
                          {rule.name}
                        </span>
                        {rule.enabled ? null : (
                          <Badge variant="secondary">{t('common.disabled')}</Badge>
                        )}
                      </div>
                      <div className="flex min-w-0 items-center gap-2">
                        <span className="truncate font-mono text-xs text-muted-foreground">
                          {formatMatch(rule)}
                        </span>
                      </div>
                    </a>
                  ))}
                </div>
              )}
            </div>
            <ListDetailFooter>
              {t('portalRule.itemCount').replace(
                '{count}',
                String(visibleRules.length),
              )}
            </ListDetailFooter>
            <ResizableListHandle
              defaultWidth={PORTAL_RULE_LIST_DEFAULT_WIDTH}
              label={t('portalRule.resizeList')}
              panel={listPanel}
            />
          </aside>

          <main className="min-h-0 overflow-hidden">
            {loading ? (
              <div className="space-y-4 p-6">
                <Skeleton className="h-8 w-56" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-[18rem] w-full" />
              </div>
            ) : isCreating ? (
              <div className="flex h-full min-h-0 flex-col">
                <div className="border-b border-border/70 px-6 py-4">
                  <div className="flex items-center gap-2">
                    <ShieldCheck className="size-4 shrink-0 text-primary" />
                    <h2 className="text-base font-semibold">
                      {t('portalRule.createTitle')}
                    </h2>
                  </div>
                </div>
                <div className="scrollbar-reserved min-h-0 flex-1 overflow-y-auto p-6 pr-4">
                  <PortalRuleInlineEditor
                    rule={null}
                    saving={saving}
                    entries={entries}
                    accessEntries={accessEntries}
                    onCancel={() => setIsCreating(false)}
                    onSubmit={handleCreate}
                  />
                </div>
              </div>
            ) : selectedRule ? (
              <div className="flex h-full min-h-0 flex-col">
                <div className="border-b border-border/70 px-6 py-4">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <ShieldCheck className="size-4 shrink-0 text-primary" />
                        <h2 className="min-w-0 truncate text-base font-semibold">
                          {selectedRule.name}
                        </h2>
                        {selectedRule.enabled ? null : (
                          <Badge variant="secondary">{t('common.disabled')}</Badge>
                        )}
                        <TargetTypeBadge routeType={selectedRule.routeType} />
                      </div>
                      <p className="mt-2 font-mono text-xs text-muted-foreground">
                        #{selectedRule.id}
                      </p>
                    </div>
                    {editingRule?.id === selectedRule.id ? null : (
                      <div className="flex items-center gap-2">
                        <Tooltip>
                          <TooltipTrigger
                            render={<span className="inline-flex" />}
                          >
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              disabled={readOnly}
                              onClick={() => {
                                const rawRule = rules.find((rule) => rule.id === selectedRule.id)
                                setEditingRule(rawRule ?? selectedRule)
                              }}
                            >
                              <Edit3 />
                              {t('action.edit')}
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>{t('action.edit')}</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger
                            render={<span className="inline-flex" />}
                          >
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon-sm"
                              disabled={readOnly}
                              onClick={() => setDeleteRule(selectedRule)}
                            >
                              <Trash2 />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>{t('action.delete')}</TooltipContent>
                        </Tooltip>
                      </div>
                    )}
                  </div>
                </div>

                <div className="scrollbar-reserved min-h-0 flex-1 overflow-y-auto p-6 pr-4">
                  {editingRule?.id === selectedRule.id ? (
                    <PortalRuleInlineEditor
                      rule={rawSelectedRule ?? selectedRule}
                      saving={saving}
                      entries={entries}
                      accessEntries={accessEntries}
                      onCancel={() => setEditingRule(null)}
                      onSubmit={handleUpdate}
                    />
                  ) : (
                    <div className="grid gap-5">
                      <div className="max-w-xl">
                        <ReadonlyField source={<FieldSourceInfo fields={selectedRuleFields} path="/name" />} label={t('portalRule.name')}>
                          {selectedRule.name}
                        </ReadonlyField>
                      </div>
                      <div className="grid gap-5 md:grid-cols-[minmax(0,1fr)_2.5rem_minmax(0,1fr)] md:items-start">
                        <RuleFlowSection
                          title={t('portalRule.match')}
                          description={t('portalRule.matchDescription')}
                        >
                          <div className="flex flex-wrap items-center gap-2">
                            {selectedAccessEntry === null ? (
                              <span className="font-mono text-xs">
                                {t('portalRule.entryMissing')}
                              </span>
                            ) : (
                              <a
                                href={portalEntryPath(selectedAccessEntry.name)}
                                className="inline-flex items-center gap-1.5 rounded-md border bg-muted/30 px-2.5 py-1.5 font-mono text-xs transition-colors hover:text-primary hover:underline"
                                onClick={(event) => {
                                  if (shouldUseBrowserNavigation(event)) {
                                    return
                                  }
                                  event.preventDefault()
                                  void navigate({
                                    to: portalEntryPath(selectedAccessEntry.name),
                                  })
                                }}
                              >
                                <Boxes className="size-3.5" />
                                {selectedAccessEntry.name}
                              </a>
                            )}
                            <span className="text-xs text-muted-foreground">
                              {t('portalRule.entryHelp')}
                            </span>
                          </div>
                          {/* The entry chip above already names the protocol,
                              host, and port the rule matches. */}
                          <ReadonlyField source={selectedMountPath === null ? <FieldSourceInfo fields={selectedRuleFields} path="/matchPathPrefix" /> : undefined} label={t('portalRule.matchPathPrefix')}>
                            {selectedMountPath === null ? (
                              selectedRule.matchPathPrefix || '/'
                            ) : (
                              <span className="text-destructive line-through">
                                {rawSelectedRule?.matchPathPrefix || '/'}
                              </span>
                            )}
                          </ReadonlyField>
                        </RuleFlowSection>

                        <div className="hidden h-10 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm md:mt-[5.25rem] md:flex">
                          <ArrowRight className="size-4" />
                        </div>

                        <RuleFlowSection
                          title={t('portalRule.route')}
                          description={t('portalRule.routeDescription')}
                        >
                          <ReadonlyField source={<FieldSourceInfo fields={selectedRuleFields} path="/routeType" />} label={t('portalRule.routeType')}>
                            <TargetTypeBadge
                              routeType={selectedRule.routeType}
                            />
                          </ReadonlyField>
                          <ReadonlyField source={<FieldSourceInfo fields={selectedRuleFields} path={selectedRule.routeType === 'SITE' ? '/routeSiteName' : '/routeRedirectionPattern'} />}
                            label={
                              selectedRule.routeType === 'SITE'
                                ? t('portalRule.routeSiteName')
                                : t('portalRule.redirectPattern')
                            }
                          >
                            {selectedRule.routeType === 'SITE' &&
                            selectedTargetSite ? (
                              <a
                                href={portalSitePath(selectedTargetSite.id)}
                                className="font-medium text-primary underline-offset-2 hover:underline"
                                onClick={(event) => {
                                  if (shouldUseBrowserNavigation(event)) {
                                    return
                                  }
                                  event.preventDefault()
                                  void navigate({
                                    to: portalSitePath(selectedTargetSite.id),
                                  })
                                }}
                              >
                                {selectedRule.routeSiteName}
                              </a>
                            ) : selectedRule.routeType === 'SITE' ? (
                              selectedRule.routeSiteName
                            ) : (
                              selectedRule.routeRedirectionPattern
                            )}
                          </ReadonlyField>
                          {selectedRule.routeType === 'SITE' ? (
                            <ReadonlyField source={selectedMountPath === null ? <FieldSourceInfo fields={selectedRuleFields} path="/routePathPrefix" /> : undefined} label={t('portalRule.routePathPrefix')}>
                              {selectedMountPath === null ? (
                                selectedRule.routePathPrefix || '/'
                              ) : (
                                <span className="text-destructive line-through">
                                  {rawSelectedRule?.routePathPrefix || '/'}
                                </span>
                              )}
                            </ReadonlyField>
                          ) : null}
                        </RuleFlowSection>
                      </div>
                      {selectedRule.routeType === 'SITE' ? (
                        <RulePathPreview
                          key={selectedRule.id}
                          mountPath={selectedMountPath}
                          matchPathPrefix={selectedRule.matchPathPrefix}
                          routePathPrefix={selectedRule.routePathPrefix}
                        />
                      ) : null}
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <Empty className="h-full min-h-[24rem] rounded-none border-0">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <ShieldCheck />
                  </EmptyMedia>
                  <EmptyTitle>
                    {visibleRules.length === 0
                      ? t('portalRule.empty')
                      : t('portalRule.noMatch')}
                  </EmptyTitle>
                  <EmptyDescription>
                    {visibleRules.length === 0
                      ? t('portalRule.emptyDescription')
                      : t('common.adjustSearch')}
                  </EmptyDescription>
                </EmptyHeader>
                {visibleRules.length === 0 ? (
                  <EmptyContent>
                    <Button type="button" disabled={readOnly} onClick={() => setIsCreating(true)}>
                      <Plus />
                      {t('portalRule.createTitle')}
                    </Button>
                  </EmptyContent>
                ) : null}
              </Empty>
            )}
          </main>
        </div>
        <DeleteRuleDialog
          open={deleteRule !== null}
          rule={deleteRule}
          deleting={deleting}
          onOpenChange={(open) => {
            if (!open) {
              setDeleteRule(null)
            }
          }}
          onConfirm={handleDelete}
        />{' '}
      </section>
    </TooltipProvider>
  )
}
