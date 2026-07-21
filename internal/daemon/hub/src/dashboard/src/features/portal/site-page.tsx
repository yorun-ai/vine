import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import {
  Edit3,
  Globe2,
  Loader2,
  Plus,
  RefreshCw,
  Search,
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
import { Textarea } from '@/components/ui/textarea'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { vrpcClient } from '@/config/vrpc-client'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import { createPortalSiteService } from '@/skeled'
import type {
  PortalSite,
  PortalSiteCreation,
  PortalCorsMode,
  PortalSiteOptions,
  PortalSiteUpdate,
} from '@/skeled'

import {
  skeletonActorHref,
  skeletonServiceHref,
  skeletonWebHref,
} from '../skeleton/model'

const portalSiteService = createPortalSiteService(vrpcClient)
const PORTAL_SITE_LIST_DEFAULT_WIDTH = 352
const PORTAL_SITE_LIST_WIDTH_STORAGE_KEY = 'vinehub_portal_site_list_width'

const portalSiteTypes = [
  { value: 'RPCGW', label: 'RPC Gateway', description: 'Forward to RPC services' },
  { value: 'WEBGW', label: 'Web Gateway', description: 'Forward to Web apps' },
] as const

type PortalSiteType = (typeof portalSiteTypes)[number]['value']

const portalCorsModes = [
  {
    value: 'DISABLED',
    label: 'portalSite.corsDisabled',
    description: 'portalSite.corsDisabledDescription',
  },
  {
    value: 'SAME_DOMAIN',
    label: 'portalSite.corsSameDomain',
    description: 'portalSite.corsSameDomainDescription',
  },
  {
    value: 'STRICT',
    label: 'portalSite.corsStrict',
    description: 'portalSite.corsStrictDescription',
  },
] as const

type PortalCorsModeValue = Extract<
  PortalCorsMode,
  'DISABLED' | 'SAME_DOMAIN' | 'STRICT'
>

interface PortalSiteFormValue {
  name: string
  type: PortalSiteType
  actorSkelName: string
  actorVia: string
  corsMode: PortalCorsModeValue
  corsAllowedOrigins: string
  rpcgwServices: string
  webName: string
}

type PortalSiteFormErrors = Partial<Record<keyof PortalSiteFormValue, string>>

const emptyFormValue: PortalSiteFormValue = {
  name: '',
  type: 'WEBGW',
  actorSkelName: '',
  actorVia: 'client',
  corsMode: 'SAME_DOMAIN',
  corsAllowedOrigins: '',
  rpcgwServices: '',
  webName: '',
}

const emptyPortalSiteOptions: PortalSiteOptions = {
  actors: [],
  services: [],
  webs: [],
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function portalSitePath(id: number) {
  return `/portal/site/${id}`
}

function portalSiteListItemDomId(id: number) {
  return `portal-site-list-item:${id}`
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

function selectedSiteIdFromPath(pathname: string) {
  const match = pathname.match(/^\/portal\/site\/(\d+)$/)
  if (!match) {
    return null
  }
  const id = Number(match[1])
  return Number.isInteger(id) && id > 0 ? id : null
}

function isPortalSitePath(pathname: string) {
  return pathname === '/portal/site' || pathname.startsWith('/portal/site/')
}

function splitLines(value: string) {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function normalizePortalCorsMode(value: string | null | undefined) {
  return portalCorsModes.some((item) => item.value === value)
    ? (value as PortalCorsModeValue)
    : 'SAME_DOMAIN'
}

function portalSiteToFormValue(entry: PortalSite): PortalSiteFormValue {
  const rpcgwServices = portalSiteRpcgwServices(entry)
  const cors = entry.cors

  return {
    name: entry.name,
    type: normalizePortalSiteType(entry.type),
    actorSkelName: entry.actorSkelName,
    actorVia: entry.actorVia,
    corsMode: normalizePortalCorsMode(cors?.mode),
    corsAllowedOrigins: (cors?.allowedOrigins ?? []).join('\n'),
    rpcgwServices: rpcgwServices.join('\n'),
    webName: entry.webName,
  }
}

function portalSiteRpcgwServices(entry: PortalSite) {
  return Array.isArray(entry.rpcgwServices) ? entry.rpcgwServices : []
}

function normalizePortalSiteType(value: string): PortalSiteType {
  return portalSiteTypes.some((item) => item.value === value)
    ? (value as PortalSiteType)
    : 'WEBGW'
}

function formValueToCreation(value: PortalSiteFormValue): PortalSiteCreation {
  return {
    name: value.name.trim(),
    type: value.type,
    actorSkelName: value.actorSkelName.trim(),
    actorVia: value.actorVia.trim(),
    cors: {
      mode: value.corsMode,
      allowedOrigins:
        value.corsMode === 'STRICT'
          ? splitLines(value.corsAllowedOrigins)
          : [],
    },
    webName: value.type === 'WEBGW' ? value.webName.trim() : '',
  }
}

function formValueToUpdate(value: PortalSiteFormValue): PortalSiteUpdate {
  const creation = formValueToCreation(value)

  return {
    name: creation.name,
    type: creation.type,
    actorSkelName: creation.actorSkelName,
    actorVia: creation.actorVia,
    cors: creation.cors,
    webName: creation.webName,
  }
}

function validateFormValue(
  value: PortalSiteFormValue,
  t: ReturnType<typeof useLocale>['t'],
) {
  const errors: PortalSiteFormErrors = {}

  if (value.name.trim() === '') {
    errors.name = t('portalSite.nameRequired')
  }
  if (value.actorSkelName.trim() === '') {
    errors.actorSkelName = t('portalSite.actorRequired')
  }
  if (value.actorVia.trim() === '') {
    errors.actorVia = t('portalSite.actorViaRequired')
  }
  if (
    value.corsMode === 'STRICT' &&
    splitLines(value.corsAllowedOrigins).length === 0
  ) {
    errors.corsAllowedOrigins = t('portalSite.originRequired')
  }
  if (value.type === 'RPCGW' && splitLines(value.rpcgwServices).length === 0) {
    errors.rpcgwServices = t('portalSite.rpcServiceRequired')
  }
  if (value.type === 'WEBGW' && value.webName.trim() === '') {
    errors.webName = t('portalSite.webRequired')
  }

  return errors
}

function hasFormErrors(errors: PortalSiteFormErrors) {
  return Object.keys(errors).length > 0
}

function portalSiteTypeLabel(value: string) {
  return portalSiteTypes.find((item) => item.value === value)?.label ?? value
}

function portalCorsModeLabel(value: string | null | undefined) {
  const mode = normalizePortalCorsMode(value)
  return (
    portalCorsModes.find((item) => item.value === mode)?.label ??
    'portalSite.corsSameDomain'
  )
}

function includesValue(values: Array<string>, value: string) {
  return value !== '' && values.includes(value)
}

function FieldRow({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <div className={cn('grid gap-4 sm:grid-cols-2', className)}>{children}</div>
  )
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

function PortalCorsFields({
  formValue,
  fieldErrors,
  setField,
}: {
  formValue: PortalSiteFormValue
  fieldErrors: PortalSiteFormErrors
  setField: (field: keyof PortalSiteFormValue, value: string) => void
}) {
  const { t } = useLocale()

  return (
    <div className="grid gap-4 border-t pt-4">
      <Field label={t('portalSite.corsMode')} error={fieldErrors.corsMode}>
        <Select
          value={formValue.corsMode}
          onValueChange={(value) => {
            if (value) {
              setField('corsMode', normalizePortalCorsMode(value))
            }
          }}
        >
          <SelectTrigger
            aria-invalid={Boolean(fieldErrors.corsMode)}
            className="w-full"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent align="start">
            {portalCorsModes.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                <span className="flex flex-col">
                  <span>{t(item.label)}</span>
                  <span className="text-xs text-muted-foreground">
                    {t(item.description)}
                  </span>
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>
      {formValue.corsMode === 'STRICT' && (
        <Field
          label={t('portalSite.allowedOrigins')}
          error={fieldErrors.corsAllowedOrigins}
        >
          <Textarea
            aria-invalid={Boolean(fieldErrors.corsAllowedOrigins)}
            className="min-h-24 font-mono text-xs"
            value={formValue.corsAllowedOrigins}
            placeholder="https://console.example.com"
            onChange={(event) =>
              setField('corsAllowedOrigins', event.target.value)
            }
          />
        </Field>
      )}
    </div>
  )
}

function PortalSiteDialog({
  mode,
  open,
  saving,
  entry,
  options,
  onOpenChange,
  onSubmit,
}: {
  mode: 'create' | 'edit'
  open: boolean
  saving: boolean
  entry: PortalSite | null
  options: PortalSiteOptions
  onOpenChange: (open: boolean) => void
  onSubmit: (value: PortalSiteFormValue) => Promise<void>
}) {
  const { t, tText } = useLocale()
  const [formValue, setFormValue] =
    React.useState<PortalSiteFormValue>(emptyFormValue)
  const [fieldErrors, setFieldErrors] = React.useState<PortalSiteFormErrors>({})
  const [formError, setFormError] = React.useState<string | null>(null)
  const isCreate = mode === 'create'
  const currentActor = React.useMemo(
    () =>
      options.actors.find(
        (actor) => actor.skelName === formValue.actorSkelName,
      ),
    [formValue.actorSkelName, options.actors],
  )
  const actorOptions = React.useMemo(() => {
    if (
      formValue.actorSkelName === '' ||
      options.actors.some((actor) => actor.skelName === formValue.actorSkelName)
    ) {
      return options.actors
    }

    return [
      ...options.actors,
      {
        name: formValue.actorSkelName,
        skelName: formValue.actorSkelName,
        actorVias: includesValue([], formValue.actorVia)
          ? []
          : [formValue.actorVia].filter(Boolean),
      },
    ]
  }, [formValue.actorVia, formValue.actorSkelName, options.actors])
  const actorViaOptions = React.useMemo(() => {
    const actorVias = currentActor?.actorVias ?? []
    if (includesValue(actorVias, formValue.actorVia)) {
      return actorVias
    }

    return [formValue.actorVia, ...actorVias].filter(Boolean)
  }, [currentActor, formValue.actorVia])
  const availableServices = React.useMemo(
    () =>
      options.services.filter((service) =>
        service.actorSkelNames.includes(formValue.actorSkelName),
      ),
    [formValue.actorSkelName, options.services],
  )
  const availableWebs = React.useMemo(
    () =>
      options.webs.filter((web) =>
        web.actorSkelNames.includes(formValue.actorSkelName),
      ),
    [formValue.actorSkelName, options.webs],
  )
  const webOptions = React.useMemo(() => {
    if (
      formValue.webName === '' ||
      availableWebs.some((web) => web.skelName === formValue.webName)
    ) {
      return availableWebs
    }

    return [
      ...availableWebs,
      {
        name: formValue.webName,
        skelName: formValue.webName,
        actorSkelNames: [],
      },
    ]
  }, [availableWebs, formValue.webName])

  const normalizeDerivedFields = React.useCallback(
    (value: PortalSiteFormValue): PortalSiteFormValue => ({
      ...value,
      rpcgwServices:
        value.type === 'RPCGW'
          ? availableServices.map((service) => service.skelName).join('\n')
          : '',
    }),
    [availableServices],
  )

  React.useEffect(() => {
    if (!open) {
      return
    }

    setFormValue(entry ? portalSiteToFormValue(entry) : emptyFormValue)
    setFieldErrors({})
    setFormError(null)
  }, [open, entry])

  const setField = React.useCallback(
    (field: keyof PortalSiteFormValue, value: string) => {
      setFormError(null)
      setFieldErrors((current) => {
        if (!current[field]) {
          return current
        }

        const { [field]: _removed, ...next } = current
        return next
      })
      setFormValue((current) => ({
        ...current,
        [field]: value,
      }))
    },
    [],
  )

  const handleTypeChange = React.useCallback(
    (type: PortalSiteType) => {
      setFormError(null)
      setFieldErrors((current) => {
        const {
          type: _type,
          rpcgwServices: _rpcgwServices,
          webName: _webName,
          ...next
        } = current
        return next
      })
      setFormValue((current) => ({
        ...current,
        type,
        webName:
          type === 'WEBGW' && current.webName === '' && availableWebs[0]
            ? availableWebs[0].skelName
            : current.webName,
      }))
    },
    [availableWebs],
  )

  const handleActorChange = React.useCallback(
    (actorSkelName: string) => {
      const actor = options.actors.find(
        (item) => item.skelName === actorSkelName,
      )
      const nextActorVia =
        actor && !actor.actorVias.includes(formValue.actorVia)
          ? (actor.actorVias[0] ?? '')
          : formValue.actorVia
      const nextWeb = options.webs.find((web) =>
        web.actorSkelNames.includes(actorSkelName),
      )

      setFormError(null)
      setFieldErrors((current) => {
        const {
          actorSkelName: _actorSkelName,
          actorVia: _actorVia,
          rpcgwServices: _rpcgwServices,
          webName: _webName,
          ...next
        } = current
        return next
      })
      setFormValue((current) => ({
        ...current,
        actorSkelName,
        actorVia: nextActorVia,
        webName:
          current.type === 'WEBGW' &&
          current.webName !== '' &&
          !options.webs.some(
            (web) =>
              web.skelName === current.webName &&
              web.actorSkelNames.includes(actorSkelName),
          )
            ? (nextWeb?.skelName ?? '')
            : current.webName,
      }))
    },
    [formValue.actorVia, options.actors, options.webs],
  )

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const nextValue = normalizeDerivedFields(formValue)
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
    [formValue, normalizeDerivedFields, onSubmit],
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-2xl">
        <form className="grid gap-5" onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>
              {isCreate
                ? t('portalSite.createTitle')
                : t('portalSite.editTitle')}
            </DialogTitle>
            <DialogDescription>
              {t('portalSite.dialogDescription')}
            </DialogDescription>
          </DialogHeader>

          {formError ? (
            <Alert variant="destructive">
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          ) : null}

          <FieldRow>
            <Field label={t('portalSite.name')} error={fieldErrors.name}>
              <Input
                aria-invalid={Boolean(fieldErrors.name)}
                value={formValue.name}
                placeholder="vine.hub.DashboardWeb-web"
                onChange={(event) => setField('name', event.target.value)}
              />
            </Field>
            <Field label={t('portalSite.type')} error={fieldErrors.type}>
              <Select
                value={formValue.type}
                onValueChange={(value) => {
                  if (value) {
                    handleTypeChange(normalizePortalSiteType(value))
                  }
                }}
              >
                <SelectTrigger
                  aria-invalid={Boolean(fieldErrors.type)}
                  className="w-full"
                >
                  <SelectValue placeholder={t('portalSite.selectType')} />
                </SelectTrigger>
                <SelectContent align="start">
                  {portalSiteTypes.map((item) => (
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
          </FieldRow>

          <FieldRow>
            <Field label="Actor Skel" error={fieldErrors.actorSkelName}>
              <Select
                value={formValue.actorSkelName}
                onValueChange={(value) => {
                  if (value) {
                    handleActorChange(value)
                  }
                }}
              >
                <SelectTrigger
                  aria-invalid={Boolean(fieldErrors.actorSkelName)}
                  className="w-full"
                >
                  <SelectValue placeholder={t('portalSite.selectActor')} />
                </SelectTrigger>
                <SelectContent align="start">
                  {actorOptions.map((actor) => (
                    <SelectItem key={actor.skelName} value={actor.skelName}>
                      <span className="flex flex-col">
                        <span>{actor.skelName}</span>
                        <span className="text-xs text-muted-foreground">
                          {actor.name}
                        </span>
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field
              label={t('portalSite.actorVia')}
              error={fieldErrors.actorVia}
            >
              <Select
                value={formValue.actorVia}
                onValueChange={(value) => {
                  if (value) {
                    setField('actorVia', value)
                  }
                }}
              >
                <SelectTrigger
                  aria-invalid={Boolean(fieldErrors.actorVia)}
                  className="w-full"
                >
                  <SelectValue
                    placeholder={t('portalSite.selectActorVia')}
                  />
                </SelectTrigger>
                <SelectContent align="start">
                  {actorViaOptions.map((actorVia) => (
                    <SelectItem key={actorVia} value={actorVia}>
                      {actorVia}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
          </FieldRow>

          {formValue.type === 'RPCGW' ? (
            <Field
              label={t('portalSite.rpcService')}
              error={fieldErrors.rpcgwServices}
            >
              <div
                aria-invalid={Boolean(fieldErrors.rpcgwServices)}
                className={cn(
                  'min-h-20 rounded-lg border border-input bg-muted/20 px-2.5 py-2',
                  fieldErrors.rpcgwServices &&
                    'border-destructive ring-3 ring-destructive/20',
                )}
              >
                {availableServices.length === 0 ? (
                  <div className="text-sm text-muted-foreground">
                    {t('portalSite.noActorRpcServices')}
                  </div>
                ) : (
                  <div className="flex flex-wrap gap-1">
                    {availableServices.map((service) => (
                      <Badge key={service.skelName} variant="outline">
                        {service.skelName}
                      </Badge>
                    ))}
                  </div>
                )}
              </div>
            </Field>
          ) : (
            <Field label={t('portalSite.webName')} error={fieldErrors.webName}>
              <Select
                value={formValue.webName}
                onValueChange={(value) => {
                  if (value) {
                    setField('webName', value)
                  }
                }}
              >
                <SelectTrigger
                  aria-invalid={Boolean(fieldErrors.webName)}
                  className="w-full"
                >
                  <SelectValue placeholder={t('portalSite.selectWeb')} />
                </SelectTrigger>
                <SelectContent align="start">
                  {webOptions.map((web) => (
                    <SelectItem key={web.skelName} value={web.skelName}>
                      <span className="flex flex-col">
                        <span>{web.skelName}</span>
                        <span className="text-xs text-muted-foreground">
                          {web.actorSkelNames.length === 0
                            ? t('portalSite.webNotFound')
                            : web.name}
                        </span>
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
          )}

          <PortalCorsFields
            formValue={formValue}
            fieldErrors={fieldErrors}
            setField={setField}
          />

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

function DeletePortalSiteDialog({
  deleting,
  open,
  entry,
  onConfirm,
  onOpenChange,
}: {
  deleting: boolean
  open: boolean
  entry: PortalSite | null
  onConfirm: () => void
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useLocale()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('portalSite.deleteTitle')}</DialogTitle>
          <DialogDescription>
            {t('portalSite.deleteDescription')}
          </DialogDescription>
        </DialogHeader>
        <div className="rounded-lg border bg-muted/30 px-3 py-2 text-sm">
          {entry?.name}
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

function PortalSiteListSkeleton() {
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

function PortalSiteInlineEditor({
  saving,
  entry,
  options,
  onCancel,
  onSubmit,
}: {
  saving: boolean
  entry: PortalSite | null
  options: PortalSiteOptions
  onCancel: () => void
  onSubmit: (value: PortalSiteFormValue) => Promise<void>
}) {
  const { t, tText } = useLocale()
  const [formValue, setFormValue] = React.useState<PortalSiteFormValue>(() =>
    entry ? portalSiteToFormValue(entry) : emptyFormValue,
  )
  const [fieldErrors, setFieldErrors] = React.useState<PortalSiteFormErrors>({})
  const [formError, setFormError] = React.useState<string | null>(null)
  const currentActor = React.useMemo(
    () =>
      options.actors.find(
        (actor) => actor.skelName === formValue.actorSkelName,
      ),
    [formValue.actorSkelName, options.actors],
  )
  const actorOptions = React.useMemo(() => {
    if (
      formValue.actorSkelName === '' ||
      options.actors.some((actor) => actor.skelName === formValue.actorSkelName)
    ) {
      return options.actors
    }

    return [
      ...options.actors,
      {
        name: formValue.actorSkelName,
        skelName: formValue.actorSkelName,
        actorVias: includesValue([], formValue.actorVia)
          ? []
          : [formValue.actorVia].filter(Boolean),
      },
    ]
  }, [formValue.actorVia, formValue.actorSkelName, options.actors])
  const actorViaOptions = React.useMemo(() => {
    const actorVias = currentActor?.actorVias ?? []
    if (includesValue(actorVias, formValue.actorVia)) {
      return actorVias
    }

    return [formValue.actorVia, ...actorVias].filter(Boolean)
  }, [currentActor, formValue.actorVia])
  const availableServices = React.useMemo(
    () =>
      options.services.filter((service) =>
        service.actorSkelNames.includes(formValue.actorSkelName),
      ),
    [formValue.actorSkelName, options.services],
  )
  const availableWebs = React.useMemo(
    () =>
      options.webs.filter((web) =>
        web.actorSkelNames.includes(formValue.actorSkelName),
      ),
    [formValue.actorSkelName, options.webs],
  )
  const webOptions = React.useMemo(() => {
    if (
      formValue.webName === '' ||
      availableWebs.some((web) => web.skelName === formValue.webName)
    ) {
      return availableWebs
    }

    return [
      ...availableWebs,
      {
        name: formValue.webName,
        skelName: formValue.webName,
        actorSkelNames: [],
      },
    ]
  }, [availableWebs, formValue.webName])
  const normalizeDerivedFields = React.useCallback(
    (value: PortalSiteFormValue): PortalSiteFormValue => ({
      ...value,
      rpcgwServices:
        value.type === 'RPCGW'
          ? availableServices.map((service) => service.skelName).join('\n')
          : '',
    }),
    [availableServices],
  )

  React.useEffect(() => {
    setFormValue(entry ? portalSiteToFormValue(entry) : emptyFormValue)
    setFieldErrors({})
    setFormError(null)
  }, [entry])

  const setField = React.useCallback(
    (field: keyof PortalSiteFormValue, value: string) => {
      setFormError(null)
      setFieldErrors((current) => {
        if (!current[field]) {
          return current
        }

        const { [field]: _removed, ...next } = current
        return next
      })
      setFormValue((current) => ({
        ...current,
        [field]: value,
      }))
    },
    [],
  )

  const handleTypeChange = React.useCallback(
    (type: PortalSiteType) => {
      setFormError(null)
      setFieldErrors((current) => {
        const {
          type: _type,
          rpcgwServices: _rpcgwServices,
          webName: _webName,
          ...next
        } = current
        return next
      })
      setFormValue((current) => ({
        ...current,
        type,
        webName:
          type === 'WEBGW' && current.webName === '' && availableWebs[0]
            ? availableWebs[0].skelName
            : current.webName,
      }))
    },
    [availableWebs],
  )

  const handleActorChange = React.useCallback(
    (actorSkelName: string) => {
      const actor = options.actors.find(
        (item) => item.skelName === actorSkelName,
      )
      const nextActorVia =
        actor && !actor.actorVias.includes(formValue.actorVia)
          ? (actor.actorVias[0] ?? '')
          : formValue.actorVia
      const nextWeb = options.webs.find((web) =>
        web.actorSkelNames.includes(actorSkelName),
      )

      setFormError(null)
      setFieldErrors((current) => {
        const {
          actorSkelName: _actorSkelName,
          actorVia: _actorVia,
          rpcgwServices: _rpcgwServices,
          webName: _webName,
          ...next
        } = current
        return next
      })
      setFormValue((current) => ({
        ...current,
        actorSkelName,
        actorVia: nextActorVia,
        webName:
          current.type === 'WEBGW' &&
          current.webName !== '' &&
          !options.webs.some(
            (web) =>
              web.skelName === current.webName &&
              web.actorSkelNames.includes(actorSkelName),
          )
            ? (nextWeb?.skelName ?? '')
            : current.webName,
      }))
    },
    [formValue.actorVia, options.actors, options.webs],
  )

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const nextValue = normalizeDerivedFields(formValue)
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
    [formValue, normalizeDerivedFields, onSubmit],
  )

  return (
    <form className="grid gap-5" onSubmit={handleSubmit}>
      {formError ? (
        <Alert variant="destructive">
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      <FieldRow>
        <Field label={t('portalSite.name')} error={fieldErrors.name}>
          <Input
            aria-invalid={Boolean(fieldErrors.name)}
            value={formValue.name}
            placeholder="vine.hub.DashboardWeb-web"
            onChange={(event) => setField('name', event.target.value)}
          />
        </Field>
        <Field label={t('portalSite.type')} error={fieldErrors.type}>
          <Select
            value={formValue.type}
            onValueChange={(value) => {
              if (value) {
                handleTypeChange(normalizePortalSiteType(value))
              }
            }}
          >
            <SelectTrigger
              aria-invalid={Boolean(fieldErrors.type)}
              className="w-full"
            >
              <SelectValue placeholder={t('portalSite.selectType')} />
            </SelectTrigger>
            <SelectContent align="start">
              {portalSiteTypes.map((item) => (
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
      </FieldRow>

      <FieldRow>
        <Field label="Actor Skel" error={fieldErrors.actorSkelName}>
          <Select
            value={formValue.actorSkelName}
            onValueChange={(value) => {
              if (value) {
                handleActorChange(value)
              }
            }}
          >
            <SelectTrigger
              aria-invalid={Boolean(fieldErrors.actorSkelName)}
              className="w-full"
            >
              <SelectValue placeholder={t('portalSite.selectActor')} />
            </SelectTrigger>
            <SelectContent align="start">
              {actorOptions.map((actor) => (
                <SelectItem key={actor.skelName} value={actor.skelName}>
                  <span className="flex flex-col">
                    <span>{actor.skelName}</span>
                    <span className="text-xs text-muted-foreground">
                      {actor.name}
                    </span>
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field
          label={t('portalSite.actorVia')}
          error={fieldErrors.actorVia}
        >
          <Select
            value={formValue.actorVia}
            onValueChange={(value) => {
              if (value) {
                setField('actorVia', value)
              }
            }}
          >
            <SelectTrigger
              aria-invalid={Boolean(fieldErrors.actorVia)}
              className="w-full"
            >
              <SelectValue placeholder={t('portalSite.selectActorVia')} />
            </SelectTrigger>
            <SelectContent align="start">
              {actorViaOptions.map((actorVia) => (
                <SelectItem key={actorVia} value={actorVia}>
                  {actorVia}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
      </FieldRow>

      {formValue.type === 'RPCGW' ? (
        <Field
          label={t('portalSite.rpcService')}
          error={fieldErrors.rpcgwServices}
        >
          <div
            aria-invalid={Boolean(fieldErrors.rpcgwServices)}
            className={cn(
              'min-h-20 rounded-lg border border-input bg-muted/20 px-2.5 py-2',
              fieldErrors.rpcgwServices &&
                'border-destructive ring-3 ring-destructive/20',
            )}
          >
            {availableServices.length === 0 ? (
              <div className="text-sm text-muted-foreground">
                {t('portalSite.noActorRpcServices')}
              </div>
            ) : (
              <div className="flex flex-wrap gap-1">
                {availableServices.map((service) => (
                  <Badge key={service.skelName} variant="outline">
                    {service.skelName}
                  </Badge>
                ))}
              </div>
            )}
          </div>
        </Field>
      ) : (
        <Field label={t('portalSite.webName')} error={fieldErrors.webName}>
          <Select
            value={formValue.webName}
            onValueChange={(value) => {
              if (value) {
                setField('webName', value)
              }
            }}
          >
            <SelectTrigger
              aria-invalid={Boolean(fieldErrors.webName)}
              className="w-full"
            >
              <SelectValue placeholder={t('portalSite.selectWeb')} />
            </SelectTrigger>
            <SelectContent align="start">
              {webOptions.map((web) => (
                <SelectItem key={web.skelName} value={web.skelName}>
                  <span className="flex flex-col">
                    <span>{web.skelName}</span>
                    <span className="text-xs text-muted-foreground">
                      {web.actorSkelNames.length === 0
                        ? t('portalSite.webNotFound')
                        : web.name}
                    </span>
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
      )}

      <PortalCorsFields
        formValue={formValue}
        fieldErrors={fieldErrors}
        setField={setField}
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

export function PortalSitePage() {
  const { t, tText } = useLocale()
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const [entries, setEntries] = React.useState<Array<PortalSite>>([])
  const [portalSiteOptions, setPortalSiteOptions] =
    React.useState<PortalSiteOptions>(emptyPortalSiteOptions)
  const [query, setQuery] = React.useState('')
  const listPanel = useResizableListPanel({
    defaultWidth: PORTAL_SITE_LIST_DEFAULT_WIDTH,
    storageKey: PORTAL_SITE_LIST_WIDTH_STORAGE_KEY,
  })
  const handleListScroll = useReservedScrollbar()
  const [loading, setLoading] = React.useState(true)
  const [saving, setSaving] = React.useState(false)
  const [deleting, setDeleting] = React.useState(false)
  const [editingEntry, setEditingEntry] = React.useState<PortalSite | null>(
    null,
  )
  const [deleteEntry, setDeleteEntry] = React.useState<PortalSite | null>(null)
  const [isCreating, setIsCreating] = React.useState(false)
  const [selectedEntryId, setSelectedEntryId] = React.useState<number | null>(
    () => selectedSiteIdFromPath(window.location.pathname),
  )

  const loadEntries = React.useCallback(async () => {
    setLoading(true)

    try {
      const [nextEntries, nextOptions] = await Promise.all([
        portalSiteService.list(null),
        portalSiteService.listOptions(null),
      ])
      setEntries(nextEntries)
      setPortalSiteOptions(nextOptions)
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void loadEntries()
  }, [loadEntries])

  const visibleEntries = React.useMemo(() => {
    return entries
  }, [entries])

  const filteredEntries = React.useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (keyword === '') {
      return visibleEntries
    }

    return visibleEntries.filter((entry) => {
      const rpcgwServices = portalSiteRpcgwServices(entry)
      const cors = entry.cors
      const values = [
        entry.name,
        entry.type,
        entry.actorSkelName,
        entry.actorVia,
        cors?.mode ?? '',
        ...(cors?.allowedOrigins ?? []),
        rpcgwServices.join(','),
        entry.webName,
      ]

      return values.some((value) => value.toLowerCase().includes(keyword))
    })
  }, [query, visibleEntries])

  const selectedEntry = React.useMemo(
    () =>
      filteredEntries.find((entry) => entry.id === selectedEntryId) ??
      filteredEntries[0] ??
      null,
    [filteredEntries, selectedEntryId],
  )

  const selectEntry = React.useCallback(
    (id: number, replace = false) => {
      setSelectedEntryId(id)
      void navigate({ replace, to: portalSitePath(id) })
    },
    [navigate],
  )

  React.useEffect(() => {
    if (!isPortalSitePath(pathname)) {
      return
    }
    setSelectedEntryId(selectedSiteIdFromPath(pathname))
  }, [pathname])

  React.useEffect(() => {
    if (!isPortalSitePath(pathname)) {
      return
    }
    if (filteredEntries.length === 0) {
      setSelectedEntryId(null)
      return
    }

    if (!filteredEntries.some((entry) => entry.id === selectedEntryId)) {
      selectEntry(filteredEntries[0].id, true)
    }
  }, [filteredEntries, pathname, selectEntry, selectedEntryId])

  React.useEffect(() => {
    if (selectedEntryId == null) {
      return
    }
    window.requestAnimationFrame(() => {
      document
        .getElementById(portalSiteListItemDomId(selectedEntryId))
        ?.scrollIntoView({
          block: 'nearest',
          inline: 'nearest',
        })
    })
  }, [filteredEntries, selectedEntryId])

  const handleCreate = React.useCallback(
    async (value: PortalSiteFormValue) => {
      setSaving(true)

      try {
        const created = await portalSiteService.create({
          creation: formValueToCreation(value),
        })
        toast.success(t('portalSite.created'))
        setIsCreating(false)
        setEntries((current) => [...current, created])
        selectEntry(created.id)
      } catch (error) {
        throw error
      } finally {
        setSaving(false)
      }
    },
    [selectEntry],
  )

  const handleUpdate = React.useCallback(
    async (value: PortalSiteFormValue) => {
      if (!editingEntry) {
        return
      }

      setSaving(true)

      try {
        const updated = await portalSiteService.update({
          id: editingEntry.id,
          update: formValueToUpdate(value),
        })
        toast.success(t('portalSite.saved'))
        setEditingEntry(null)
        setEntries((current) =>
          current.map((entry) => (entry.id === updated.id ? updated : entry)),
        )
      } catch (error) {
        throw error
      } finally {
        setSaving(false)
      }
    },
    [editingEntry],
  )

  const handleDelete = React.useCallback(async () => {
    if (!deleteEntry) {
      return
    }

    setDeleting(true)

    try {
      await portalSiteService.remove({ id: deleteEntry.id })
      toast.success(t('portalSite.deleted'))
      setDeleteEntry(null)
      setEntries((current) =>
        current.filter((entry) => entry.id !== deleteEntry.id),
      )
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setDeleting(false)
    }
  }, [deleteEntry])

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
                  placeholder={t('portalSite.searchPlaceholder')}
                  onChange={(event) => setQuery(event.target.value)}
                />
              </div>
              <div className="flex items-center justify-between gap-2">
                <div className="text-xs text-muted-foreground">
                  {t('portalSite.itemCount').replace(
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
                  <Button
                    type="button"
                    size="sm"
                    onClick={() => {
                      setEditingEntry(null)
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
                <PortalSiteListSkeleton />
              ) : filteredEntries.length === 0 ? (
                <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                  {visibleEntries.length === 0
                    ? t('portalSite.empty')
                    : t('portalSite.noMatch')}
                </div>
              ) : (
                <div className="space-y-1">
                  {filteredEntries.map((entry) => (
                    <a
                      key={entry.id}
                      id={portalSiteListItemDomId(entry.id)}
                      href={portalSitePath(entry.id)}
                      onClick={(event) => {
                        if (shouldUseBrowserNavigation(event)) {
                          return
                        }
                        event.preventDefault()
                        selectEntry(entry.id)
                      }}
                      className={cn(
                        'relative flex w-full flex-col gap-1 rounded-lg border px-3 py-2.5 pr-20 text-left transition-colors',
                        selectedEntry?.id === entry.id
                          ? 'border-primary/30 bg-primary/[0.06]'
                          : 'border-transparent hover:bg-primary/[0.05]',
                      )}
                    >
                      <Badge
                        variant="outline"
                        className="absolute top-2.5 right-3"
                      >
                        {tText(portalSiteTypeLabel(entry.type))}
                      </Badge>
                      <div className="flex min-w-0 items-center gap-2">
                        <span className="truncate text-sm font-medium">
                          <span>{entry.name}</span>
                        </span>
                      </div>
                      <div className="flex min-w-0 items-center gap-2">
                        <span className="truncate font-mono text-xs text-muted-foreground">
                          {entry.type === 'RPCGW'
                            ? portalSiteRpcgwServices(entry).join(', ')
                            : entry.webName}
                        </span>
                      </div>
                    </a>
                  ))}
                </div>
              )}
            </div>
            <ResizableListHandle
              defaultWidth={PORTAL_SITE_LIST_DEFAULT_WIDTH}
              label={t('portalSite.resizeList')}
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
                    <Globe2 className="size-4 shrink-0 text-primary" />
                    <h2 className="text-base font-semibold">
                      {t('portalSite.createTitle')}
                    </h2>
                  </div>
                </div>
                <div className="scrollbar-reserved min-h-0 flex-1 overflow-y-auto p-6 pr-4">
                  <PortalSiteInlineEditor
                    saving={saving}
                    entry={null}
                    options={portalSiteOptions}
                    onCancel={() => setIsCreating(false)}
                    onSubmit={handleCreate}
                  />
                </div>
              </div>
            ) : selectedEntry ? (
              <div className="flex h-full min-h-0 flex-col">
                <div className="border-b border-border/70 px-6 py-4">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <Globe2 className="size-4 shrink-0 text-primary" />
                        <h2 className="min-w-0 truncate text-base font-semibold">
                          {selectedEntry.name}
                        </h2>
                        <Badge variant="secondary">
                          {tText(portalSiteTypeLabel(selectedEntry.type))}
                        </Badge>
                      </div>
                      <p className="mt-2 font-mono text-xs text-muted-foreground">
                        #{selectedEntry.id}
                      </p>
                    </div>
                    {editingEntry?.id === selectedEntry.id ? null : (
                      <div className="flex items-center gap-1">
                        <Tooltip>
                          <TooltipTrigger
                            render={<span className="inline-flex" />}
                          >
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              onClick={() => setEditingEntry(selectedEntry)}
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
                              onClick={() => setDeleteEntry(selectedEntry)}
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
                  {editingEntry?.id === selectedEntry.id ? (
                    <PortalSiteInlineEditor
                      saving={saving}
                      entry={selectedEntry}
                      options={portalSiteOptions}
                      onCancel={() => setEditingEntry(null)}
                      onSubmit={handleUpdate}
                    />
                  ) : (
                    <div className="grid gap-5">
                      <FieldRow>
                        <ReadonlyField label={t('portalSite.name')}>
                          {selectedEntry.name}
                        </ReadonlyField>
                        <ReadonlyField label={t('portalSite.type')}>
                          <Badge variant="secondary">
                            {tText(portalSiteTypeLabel(selectedEntry.type))}
                          </Badge>
                        </ReadonlyField>
                      </FieldRow>
                      <FieldRow>
                        <ReadonlyField label="Actor Skel">
                          <a
                            href={skeletonActorHref(
                              selectedEntry.actorSkelName,
                            )}
                            className="inline-flex rounded bg-muted px-1.5 py-0.5 text-xs font-mono text-foreground underline-offset-4 hover:text-primary hover:underline"
                        >
                          {selectedEntry.actorSkelName}
                        </a>
                      </ReadonlyField>
                        <ReadonlyField label={t('portalSite.actorVia')}>
                          <Badge variant="outline">
                            {selectedEntry.actorVia}
                          </Badge>
                        </ReadonlyField>
                      </FieldRow>
                      <FieldRow>
                        <ReadonlyField label={t('portalSite.corsMode')}>
                          <Badge variant="outline">
                            {t(portalCorsModeLabel(selectedEntry.cors?.mode))}
                          </Badge>
                        </ReadonlyField>
                        <ReadonlyField label={t('portalSite.allowedOrigins')}>
                          {selectedEntry.cors?.mode === 'STRICT' &&
                          (selectedEntry.cors.allowedOrigins ?? []).length >
                            0 ? (
                            <div className="flex flex-wrap gap-1">
                              {selectedEntry.cors.allowedOrigins.map(
                                (origin) => (
                                  <Badge key={origin} variant="outline">
                                    {origin}
                                  </Badge>
                                ),
                              )}
                            </div>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </ReadonlyField>
                      </FieldRow>
                      <ReadonlyField
                        label={
                          selectedEntry.type === 'RPCGW'
                            ? t('portalSite.rpcService')
                            : t('portalSite.webName')
                        }
                        className={
                          selectedEntry.type === 'RPCGW'
                            ? 'min-h-24'
                            : undefined
                        }
                      >
                        {selectedEntry.type === 'RPCGW' ? (
                          <div className="flex flex-wrap gap-1">
                            {portalSiteRpcgwServices(selectedEntry).map(
                              (serviceName) => (
                                <a
                                  key={serviceName}
                                  href={skeletonServiceHref(serviceName)}
                                  className="inline-flex rounded bg-muted px-1.5 py-0.5 text-xs font-mono text-foreground underline-offset-4 hover:text-primary hover:underline"
                                >
                                  {serviceName}
                                </a>
                              ),
                            )}
                          </div>
                        ) : (
                          <a
                            href={skeletonWebHref(selectedEntry.webName)}
                            className="inline-flex rounded bg-muted px-1.5 py-0.5 text-xs font-mono text-foreground underline-offset-4 hover:text-primary hover:underline"
                          >
                            {selectedEntry.webName}
                          </a>
                        )}
                      </ReadonlyField>
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <Empty className="h-full min-h-[24rem] rounded-none border-0">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <Globe2 />
                  </EmptyMedia>
                  <EmptyTitle>
                    {visibleEntries.length === 0
                      ? t('portalSite.empty')
                      : t('portalSite.noMatch')}
                  </EmptyTitle>
                  <EmptyDescription>
                    {visibleEntries.length === 0
                      ? t('portalSite.emptyDescription')
                      : t('common.adjustSearch')}
                  </EmptyDescription>
                </EmptyHeader>
                {visibleEntries.length === 0 ? (
                  <EmptyContent>
                    <Button type="button" onClick={() => setIsCreating(true)}>
                      <Plus />
                      {t('portalSite.createTitle')}
                    </Button>
                  </EmptyContent>
                ) : null}
              </Empty>
            )}
          </main>
        </div>

        <PortalSiteDialog
          mode="edit"
          open={false}
          saving={saving}
          entry={editingEntry}
          options={portalSiteOptions}
          onOpenChange={(open) => {
            if (!open) {
              setEditingEntry(null)
            }
          }}
          onSubmit={handleUpdate}
        />
        <DeletePortalSiteDialog
          deleting={deleting}
          open={deleteEntry !== null}
          entry={deleteEntry}
          onOpenChange={(open) => {
            if (!open) {
              setDeleteEntry(null)
            }
          }}
          onConfirm={handleDelete}
        />
      </section>
    </TooltipProvider>
  )
}
