import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import {
  ArrowRight,
  Edit3,
  Loader2,
  Plus,
  RefreshCw,
  Search,
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
  createPortalRuleService,
  createPortalSiteService,
} from '@/skeled'
import type {
  PortalRule,
  PortalRuleCreation,
  PortalRuleUpdate,
  PortalSite,
} from '@/skeled'

const portalRuleService = createPortalRuleService(vrpcClient)
const portalSiteService = createPortalSiteService(vrpcClient)
const PORTAL_RULE_LIST_DEFAULT_WIDTH = 352
const PORTAL_RULE_LIST_WIDTH_STORAGE_KEY = 'vinehub_portal_rule_list_width'
const targetTypes = [
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

type PortalRuleTargetType = (typeof targetTypes)[number]['value']
type PortalRuleFormErrors = Partial<Record<keyof PortalRuleFormValue, string>>

interface PortalRuleFormValue {
  name: string
  scheme: string
  host: string
  port: string
  pathPrefix: string
  targetType: PortalRuleTargetType
  siteName: string
  redirectionPattern: string
}

const emptyFormValue: PortalRuleFormValue = {
  name: '',
  scheme: 'http',
  host: '',
  port: '',
  pathPrefix: '',
  targetType: 'SITE',
  siteName: '',
  redirectionPattern: '',
}

const defaultPortsByScheme: Record<string, string> = {
  http: '80',
  https: '443',
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function ruleToFormValue(rule: PortalRule): PortalRuleFormValue {
  return {
    name: rule.name,
    scheme: rule.scheme,
    host: rule.host,
    port: rule.port === 0 ? '' : String(rule.port),
    pathPrefix: rule.pathPrefix,
    targetType: normalizeTargetType(rule.targetType),
    siteName: rule.siteName,
    redirectionPattern: rule.redirectionPattern,
  }
}

function normalizeTargetType(value: string): PortalRuleTargetType {
  return targetTypes.some((item) => item.value === value)
    ? (value as PortalRuleTargetType)
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
    value.targetType === 'SITE' ? value.siteName : value.redirectionPattern

  return [
    normalizeRuleNamePart(value.scheme, 'http'),
    normalizeRuleNamePart(value.host, 'all'),
    normalizeRuleNamePart(value.port, 'auto'),
    normalizeRuleNamePart(value.pathPrefix, 'root'),
    normalizeRuleNamePart(target, value.targetType.toLowerCase()),
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
  value: string,
) {
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

function formValueToCreation(value: PortalRuleFormValue): PortalRuleCreation {
  return {
    name: value.name.trim() || derivePortalRuleName(value),
    scheme: value.scheme,
    host: value.host.trim(),
    port: Number(value.port || 0),
    pathPrefix: value.pathPrefix.trim(),
    targetType: value.targetType,
    siteName: value.targetType === 'SITE' ? value.siteName.trim() : '',
    redirectionPattern:
      value.targetType === 'SITE' ? '' : value.redirectionPattern.trim(),
  }
}

function formValueToUpdate(value: PortalRuleFormValue): PortalRuleUpdate {
  const creation = formValueToCreation(value)

  return {
    name: creation.name,
    scheme: creation.scheme,
    host: creation.host,
    port: creation.port,
    pathPrefix: creation.pathPrefix,
    targetType: creation.targetType,
    siteName: creation.siteName,
    redirectionPattern: creation.redirectionPattern,
  }
}

function formatMatch(rule: PortalRule) {
  const port = rule.port === 0 ? '' : `:${rule.port}`
  const host = rule.host || '*'
  const pathPrefix = rule.pathPrefix || '/'

  return `${rule.scheme}://${host}${port}${pathPrefix}`
}

function targetTypeLabel(targetType: string) {
  return (
    targetTypes.find((item) => item.value === targetType)?.label ?? targetType
  )
}

function isRedirectTarget(targetType: string) {
  return (
    targetType === 'PERMANENT_REDIRECT' || targetType === 'TEMPORARY_REDIRECT'
  )
}

function selectablePortalSites(entries: Array<PortalSite>) {
  return entries
}

function validateFormValue(
  value: PortalRuleFormValue,
  t: ReturnType<typeof useLocale>['t'],
) {
  const port = Number(value.port)
  const errors: PortalRuleFormErrors = {}

  if (value.name.trim() === '') {
    errors.name = t('portalRule.nameRequired')
  }

  if (value.scheme !== 'http' && value.scheme !== 'https') {
    errors.scheme = t('portalRule.schemeInvalid')
  }

  if (
    value.port.trim() !== '' &&
    (!Number.isInteger(port) || port < 1 || port > 65535)
  ) {
    errors.port = t('portalRule.portInvalid')
  }

  if (value.targetType === 'SITE' && value.siteName.trim() === '') {
    errors.siteName = t('portalRule.siteRequired')
  }

  if (
    isRedirectTarget(value.targetType) &&
    value.redirectionPattern.trim() === ''
  ) {
    errors.redirectionPattern = t('portalRule.redirectRequired')
  }

  return errors
}

function hasFormErrors(errors: PortalRuleFormErrors) {
  return Object.keys(errors).length > 0
}

function portalRulePath(id: number) {
  return `/portal/rule/${id}`
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
  targetType,
  className,
}: {
  targetType: string
  className?: string
}) {
  const { tText } = useLocale()
  return (
    <Badge variant="outline" className={className}>
      {tText(targetTypeLabel(targetType))}
    </Badge>
  )
}

function PortalRuleDialog({
  mode,
  open,
  rule,
  saving,
  entries,
  onOpenChange,
  onSubmit,
}: {
  mode: 'create' | 'edit'
  open: boolean
  rule: PortalRule | null
  saving: boolean
  entries: Array<PortalSite>
  onOpenChange: (open: boolean) => void
  onSubmit: (value: PortalRuleFormValue) => Promise<void>
}) {
  const { t, tText } = useLocale()
  const [formValue, setFormValue] =
    React.useState<PortalRuleFormValue>(emptyFormValue)
  const [fieldErrors, setFieldErrors] = React.useState<PortalRuleFormErrors>({})
  const [formError, setFormError] = React.useState<string | null>(null)
  const isCreate = mode === 'create'
  const selectableEntries = React.useMemo(
    () => selectablePortalSites(entries),
    [entries],
  )
  const entryOptions = React.useMemo(() => {
    if (
      formValue.siteName === '' ||
      selectableEntries.some((entry) => entry.name === formValue.siteName)
    ) {
      return selectableEntries
    }

    return [
      ...selectableEntries,
      {
        id: 0,
        name: formValue.siteName,
        type: '',
        actorSkelName: '',
        actorVia: '',
        rpcgwServices: [],
        webName: '',
      },
    ]
  }, [formValue.siteName, selectableEntries])

  React.useEffect(() => {
    if (!open) {
      return
    }

    const nextFormValue = rule ? ruleToFormValue(rule) : emptyFormValue
    setFormValue({
      ...nextFormValue,
      name: nextFormValue.name || derivePortalRuleName(nextFormValue),
    })
    setFieldErrors({})
    setFormError(null)
  }, [open, rule])

  const setField = React.useCallback(
    (field: keyof PortalRuleFormValue, value: string) => {
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

  const handleSchemeChange = React.useCallback((scheme: string) => {
    setFormError(null)
    setFieldErrors((current) => {
      if (!current.scheme && !current.port) {
        return current
      }

      const { scheme: _scheme, port: _port, ...next } = current
      return next
    })
    setFormValue((current) => {
      const currentDefaultPort = defaultPortsByScheme[current.scheme]
      const nextDefaultPort = defaultPortsByScheme[scheme]
      const nextPort =
        currentDefaultPort && current.port === currentDefaultPort
          ? (nextDefaultPort ?? current.port)
          : current.port

      return syncDerivedName(current, {
        ...current,
        scheme,
        port: nextPort,
      })
    })
  }, [])

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const nextValue = {
        ...formValue,
        name: formValue.name.trim() || derivePortalRuleName(formValue),
      }
      const errors = validateFormValue(nextValue, t)
      if (hasFormErrors(errors)) {
        setFieldErrors(errors)
        return
      }

      setFieldErrors({})
      setFormError(null)

      try {
        await onSubmit(nextValue)
      } catch (error) {
        setFormError(getErrorMessage(error))
      }
    },
    [formValue, onSubmit],
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-4xl">
        <form className="grid gap-5" onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>
              {isCreate
                ? t('portalRule.createTitle')
                : t('portalRule.editTitle')}
            </DialogTitle>
            <DialogDescription>
              {t('portalRule.dialogDescription')}
            </DialogDescription>
          </DialogHeader>

          {formError ? (
            <Alert variant="destructive">
              <AlertDescription>{formError}</AlertDescription>
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
              title={t('portalRule.matchCondition')}
              description={t('portalRule.matchDescription')}
            >
              <Field label={t('portalRule.scheme')} error={fieldErrors.scheme}>
                <Select
                  value={formValue.scheme}
                  onValueChange={(value) => {
                    if (value) {
                      handleSchemeChange(value)
                    }
                  }}
                >
                  <SelectTrigger
                    aria-invalid={Boolean(fieldErrors.scheme)}
                    className="w-full"
                  >
                    <SelectValue placeholder={t('portalRule.selectScheme')} />
                  </SelectTrigger>
                  <SelectContent align="start">
                    <SelectItem value="http">http</SelectItem>
                    <SelectItem value="https">https</SelectItem>
                  </SelectContent>
                </Select>
              </Field>

              <Field label={t('portalRule.port')} error={fieldErrors.port}>
                <Input
                  aria-invalid={Boolean(fieldErrors.port)}
                  value={formValue.port}
                  inputMode="numeric"
                  placeholder={t('portalRule.portPlaceholder')}
                  onChange={(event) => setField('port', event.target.value)}
                />
              </Field>

              <Field label={t('portalRule.host')}>
                <Input
                  value={formValue.host}
                  placeholder={t('portalRule.hostPlaceholder')}
                  onChange={(event) => setField('host', event.target.value)}
                />
              </Field>

              <Field label={t('portalRule.pathPrefix')}>
                <Input
                  value={formValue.pathPrefix}
                  placeholder={t('portalRule.pathPrefixPlaceholder')}
                  onChange={(event) =>
                    setField('pathPrefix', event.target.value)
                  }
                />
              </Field>
            </RuleFlowSection>

            <div className="flex justify-center pt-0 md:pt-20">
              <div className="grid size-10 place-items-center rounded-full border bg-background text-muted-foreground shadow-xs">
                <ArrowRight className="size-4 rotate-90 md:rotate-0" />
              </div>
            </div>

            <RuleFlowSection
              title={t('portalRule.target')}
              description={t('portalRule.targetDescription')}
            >
              <Field
                label={t('portalRule.targetType')}
                error={fieldErrors.targetType}
              >
                <Select
                  value={formValue.targetType}
                  onValueChange={(value) => {
                    if (value) {
                      setField('targetType', normalizeTargetType(value))
                    }
                  }}
                >
                  <SelectTrigger
                    aria-invalid={Boolean(fieldErrors.targetType)}
                    className="w-full"
                  >
                    <SelectValue
                      placeholder={t('portalRule.selectTargetType')}
                    />
                  </SelectTrigger>
                  <SelectContent align="start">
                    {targetTypes.map((item) => (
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

              {formValue.targetType === 'SITE' ? (
                <Field
                  label={t('portalRule.site')}
                  error={fieldErrors.siteName}
                >
                  <Select
                    value={formValue.siteName}
                    onValueChange={(value) => {
                      if (value) {
                        setField('siteName', value)
                      }
                    }}
                  >
                    <SelectTrigger
                      aria-invalid={Boolean(fieldErrors.siteName)}
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
                  error={fieldErrors.redirectionPattern}
                >
                  <Input
                    aria-invalid={Boolean(fieldErrors.redirectionPattern)}
                    value={formValue.redirectionPattern}
                    placeholder="https://example.com{uri}"
                    onChange={(event) =>
                      setField('redirectionPattern', event.target.value)
                    }
                  />
                </Field>
              )}
            </RuleFlowSection>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={saving}
              onClick={() => onOpenChange(false)}
            >
              {t('action.cancel')}
            </Button>
            <Button type="submit" disabled={saving}>
              {saving ? <Loader2 className="animate-spin" /> : null}
              {t('action.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
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
  rule: PortalRule | null
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
  children,
  className,
}: {
  label: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      <div
        className={cn(
          'min-h-9 rounded-md border border-input bg-muted/20 px-3 py-2 text-sm',
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
  onCancel,
  onSubmit,
}: {
  rule: PortalRule | null
  saving: boolean
  entries: Array<PortalSite>
  onCancel: () => void
  onSubmit: (value: PortalRuleFormValue) => Promise<void>
}) {
  const { t, tText } = useLocale()
  const [formValue, setFormValue] = React.useState<PortalRuleFormValue>(() => {
    const nextFormValue = rule ? ruleToFormValue(rule) : emptyFormValue
    return {
      ...nextFormValue,
      name: nextFormValue.name || derivePortalRuleName(nextFormValue),
    }
  })
  const [fieldErrors, setFieldErrors] = React.useState<PortalRuleFormErrors>({})
  const [formError, setFormError] = React.useState<string | null>(null)
  const selectableEntries = React.useMemo(
    () => selectablePortalSites(entries),
    [entries],
  )
  const entryOptions = React.useMemo(() => {
    if (
      formValue.siteName === '' ||
      selectableEntries.some((entry) => entry.name === formValue.siteName)
    ) {
      return selectableEntries
    }

    return [
      ...selectableEntries,
      {
        id: 0,
        name: formValue.siteName,
        type: '',
        actorSkelName: '',
        actorVia: '',
        rpcgwServices: [],
        webName: '',
      },
    ]
  }, [formValue.siteName, selectableEntries])

  React.useEffect(() => {
    const nextFormValue = rule ? ruleToFormValue(rule) : emptyFormValue
    setFormValue({
      ...nextFormValue,
      name: nextFormValue.name || derivePortalRuleName(nextFormValue),
    })
    setFieldErrors({})
    setFormError(null)
  }, [rule])

  const setField = React.useCallback(
    (field: keyof PortalRuleFormValue, value: string) => {
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

  const handleSchemeChange = React.useCallback((scheme: string) => {
    setFormError(null)
    setFieldErrors((current) => {
      if (!current.scheme && !current.port) {
        return current
      }

      const { scheme: _scheme, port: _port, ...next } = current
      return next
    })
    setFormValue((current) => {
      const currentDefaultPort = defaultPortsByScheme[current.scheme]
      const nextDefaultPort = defaultPortsByScheme[scheme]
      const nextPort =
        currentDefaultPort && current.port === currentDefaultPort
          ? (nextDefaultPort ?? current.port)
          : current.port

      return syncDerivedName(current, {
        ...current,
        scheme,
        port: nextPort,
      })
    })
  }, [])

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const nextValue = {
        ...formValue,
        name: formValue.name.trim() || derivePortalRuleName(formValue),
      }
      const errors = validateFormValue(nextValue, t)
      if (hasFormErrors(errors)) {
        setFieldErrors(errors)
        return
      }

      setFieldErrors({})
      setFormError(null)

      try {
        await onSubmit(nextValue)
      } catch (error) {
        setFormError(getErrorMessage(error))
      }
    },
    [formValue, onSubmit],
  )

  return (
    <form className="grid gap-5" onSubmit={handleSubmit}>
      {formError ? (
        <Alert variant="destructive">
          <AlertDescription>{formError}</AlertDescription>
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
          title={t('portalRule.matchCondition')}
          description={t('portalRule.matchDescription')}
        >
          <Field label={t('portalRule.scheme')} error={fieldErrors.scheme}>
            <Select
              value={formValue.scheme}
              onValueChange={(value) => {
                if (value) {
                  handleSchemeChange(value)
                }
              }}
            >
              <SelectTrigger
                aria-invalid={Boolean(fieldErrors.scheme)}
                className="w-full"
              >
                <SelectValue placeholder={t('portalRule.selectScheme')} />
              </SelectTrigger>
              <SelectContent align="start">
                <SelectItem value="http">http</SelectItem>
                <SelectItem value="https">https</SelectItem>
              </SelectContent>
            </Select>
          </Field>

          <Field label={t('portalRule.port')} error={fieldErrors.port}>
            <Input
              aria-invalid={Boolean(fieldErrors.port)}
              value={formValue.port}
              inputMode="numeric"
              placeholder={t('portalRule.portPlaceholder')}
              onChange={(event) => setField('port', event.target.value)}
            />
          </Field>

          <Field label={t('portalRule.host')}>
            <Input
              value={formValue.host}
              placeholder={t('portalRule.hostPlaceholder')}
              onChange={(event) => setField('host', event.target.value)}
            />
          </Field>

          <Field label={t('portalRule.pathPrefix')}>
            <Input
              value={formValue.pathPrefix}
              placeholder={t('portalRule.pathPrefixPlaceholder')}
              onChange={(event) => setField('pathPrefix', event.target.value)}
            />
          </Field>
        </RuleFlowSection>

        <div className="flex justify-center pt-0 md:pt-20">
          <div className="grid size-10 place-items-center rounded-full border bg-background text-muted-foreground shadow-xs">
            <ArrowRight className="size-4 rotate-90 md:rotate-0" />
          </div>
        </div>

        <RuleFlowSection
          title={t('portalRule.target')}
          description={t('portalRule.targetDescription')}
        >
          <Field
            label={t('portalRule.targetType')}
            error={fieldErrors.targetType}
          >
            <Select
              value={formValue.targetType}
              onValueChange={(value) => {
                if (value) {
                  setField('targetType', normalizeTargetType(value))
                }
              }}
            >
              <SelectTrigger
                aria-invalid={Boolean(fieldErrors.targetType)}
                className="w-full"
              >
                <SelectValue placeholder={t('portalRule.selectTargetType')} />
              </SelectTrigger>
              <SelectContent align="start">
                {targetTypes.map((item) => (
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

          {formValue.targetType === 'SITE' ? (
            <Field label={t('portalRule.site')} error={fieldErrors.siteName}>
              <Select
                value={formValue.siteName}
                onValueChange={(value) => {
                  if (value) {
                    setField('siteName', value)
                  }
                }}
              >
                <SelectTrigger
                  aria-invalid={Boolean(fieldErrors.siteName)}
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
              error={fieldErrors.redirectionPattern}
            >
              <Input
                aria-invalid={Boolean(fieldErrors.redirectionPattern)}
                value={formValue.redirectionPattern}
                placeholder="https://example.com{uri}"
                onChange={(event) =>
                  setField('redirectionPattern', event.target.value)
                }
              />
            </Field>
          )}
        </RuleFlowSection>
      </div>

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
  const { t } = useLocale()
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const [rules, setRules] = React.useState<Array<PortalRule>>([])
  const [entries, setEntries] = React.useState<Array<PortalSite>>([])
  const [query, setQuery] = React.useState('')
  const listPanel = useResizableListPanel({
    defaultWidth: PORTAL_RULE_LIST_DEFAULT_WIDTH,
    storageKey: PORTAL_RULE_LIST_WIDTH_STORAGE_KEY,
  })
  const handleListScroll = useReservedScrollbar()
  const [loading, setLoading] = React.useState(true)
  const [saving, setSaving] = React.useState(false)
  const [deleting, setDeleting] = React.useState(false)
  const [editingRule, setEditingRule] = React.useState<PortalRule | null>(null)
  const [deleteRule, setDeleteRule] = React.useState<PortalRule | null>(null)
  const [isCreating, setIsCreating] = React.useState(false)
  const [selectedRuleId, setSelectedRuleId] = React.useState<number | null>(
    () =>
      selectedRuleIdFromPath(window.location.pathname) ??
      selectedRuleIdFromSearch(),
  )

  const loadRules = React.useCallback(async () => {
    setLoading(true)

    try {
      const [nextRules, nextEntries] = await Promise.all([
        portalRuleService.list(null),
        portalSiteService.list(null),
      ])
      setRules(nextRules)
      setEntries(nextEntries)
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
    return rules
  }, [rules])

  const filteredRules = React.useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (keyword === '') {
      return visibleRules
    }

    return visibleRules.filter((rule) => {
      const values = [
        rule.name,
        rule.scheme,
        rule.host,
        String(rule.port),
        rule.pathPrefix,
        rule.targetType,
        rule.siteName,
        rule.redirectionPattern,
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
  const selectedTargetSite = React.useMemo(() => {
    if (!selectedRule || selectedRule.targetType !== 'SITE') {
      return null
    }
    return entries.find((entry) => entry.name === selectedRule.siteName) ?? null
  }, [entries, selectedRule])

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
    if (filteredRules.length === 0) {
      setSelectedRuleId(null)
      return
    }

    if (!filteredRules.some((rule) => rule.id === selectedRuleId)) {
      selectRule(filteredRules[0].id, true)
    }
  }, [filteredRules, pathname, selectRule, selectedRuleId])
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
          creation: formValueToCreation(value),
        })
        toast.success(t('portalRule.created'))
        setIsCreating(false)
        setRules((current) => [...current, created])
        selectRule(created.id)
      } catch (error) {
        throw error
      } finally {
        setSaving(false)
      }
    },
    [selectRule],
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
                <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={query}
                  className="pl-8"
                  placeholder={t('portalRule.searchPlaceholder')}
                  onChange={(event) => setQuery(event.target.value)}
                />
              </div>
              <div className="flex items-center justify-between gap-2">
                <div className="text-xs text-muted-foreground">
                  {t('portalRule.itemCount').replace(
                    '{count}',
                    String(visibleRules.length),
                  )}
                </div>
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
                        targetType={rule.targetType}
                        className="absolute top-2.5 right-3"
                      />
                      <div className="flex min-w-0 items-center gap-2">
                        <span className="truncate text-sm font-medium">
                          {rule.name}
                        </span>
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
                        <TargetTypeBadge targetType={selectedRule.targetType} />
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
                              onClick={() => setEditingRule(selectedRule)}
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
                      rule={selectedRule}
                      saving={saving}
                      entries={entries}
                      onCancel={() => setEditingRule(null)}
                      onSubmit={handleUpdate}
                    />
                  ) : (
                    <div className="grid gap-5">
                      <div className="max-w-xl">
                        <ReadonlyField label={t('portalRule.name')}>
                          {selectedRule.name}
                        </ReadonlyField>
                      </div>
                      <div className="grid gap-5 md:grid-cols-[minmax(0,1fr)_2.5rem_minmax(0,1fr)] md:items-start">
                        <RuleFlowSection
                          title={t('portalRule.matchCondition')}
                          description={t('portalRule.matchDescription')}
                        >
                          <ReadonlyField label={t('portalRule.scheme')}>
                            {selectedRule.scheme}
                          </ReadonlyField>
                          <ReadonlyField label={t('portalRule.port')}>
                            {selectedRule.port === 0
                              ? t('portalRule.followScheme')
                              : selectedRule.port}
                          </ReadonlyField>
                          <ReadonlyField label={t('portalRule.host')}>
                            {selectedRule.host || t('portalRule.anyHost')}
                          </ReadonlyField>
                          <ReadonlyField label={t('portalRule.pathPrefix')}>
                            {selectedRule.pathPrefix || '/'}
                          </ReadonlyField>
                        </RuleFlowSection>

                        <div className="hidden h-10 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm md:mt-[5.25rem] md:flex">
                          <ArrowRight className="size-4" />
                        </div>

                        <RuleFlowSection
                          title={t('portalRule.target')}
                          description={t('portalRule.targetDescription')}
                        >
                          <ReadonlyField label={t('portalRule.targetType')}>
                            <TargetTypeBadge
                              targetType={selectedRule.targetType}
                            />
                          </ReadonlyField>
                          <ReadonlyField
                            label={
                              selectedRule.targetType === 'SITE'
                                ? t('portalRule.site')
                                : t('portalRule.redirectPattern')
                            }
                          >
                            {selectedRule.targetType === 'SITE' &&
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
                                {selectedRule.siteName}
                              </a>
                            ) : selectedRule.targetType === 'SITE' ? (
                              selectedRule.siteName
                            ) : (
                              selectedRule.redirectionPattern
                            )}
                          </ReadonlyField>
                        </RuleFlowSection>
                      </div>
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
                    <Button type="button" onClick={() => setIsCreating(true)}>
                      <Plus />
                      {t('portalRule.createTitle')}
                    </Button>
                  </EmptyContent>
                ) : null}
              </Empty>
            )}
          </main>
        </div>
        <PortalRuleDialog
          mode="edit"
          open={false}
          rule={editingRule}
          saving={saving}
          entries={entries}
          onOpenChange={(open) => {
            if (!open) {
              setEditingRule(null)
            }
          }}
          onSubmit={handleUpdate}
        />
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
