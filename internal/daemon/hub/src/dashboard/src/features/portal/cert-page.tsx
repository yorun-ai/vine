import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import {
  BadgeCheck,
  Edit3,
  KeyRound,
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
import { createPortalCertService } from '@/skeled'
import type {
  PortalCert,
  PortalCertCreation,
  PortalCertUpdate,
} from '@/skeled'

const portalCertService = createPortalCertService(vrpcClient)
const PORTAL_CERT_LIST_DEFAULT_WIDTH = 352
const PORTAL_CERT_LIST_WIDTH_STORAGE_KEY = 'vinehub_portal_cert_list_width'

interface PortalCertFormValue {
  name: string
  publicKeyBase64: string
  privateKeyBase64: string
}

type PortalCertFormErrors = Partial<Record<keyof PortalCertFormValue, string>>

const emptyFormValue: PortalCertFormValue = {
  name: '',
  publicKeyBase64: '',
  privateKeyBase64: '',
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function portalCertPath(id: number) {
  return `/portal/cert/${id}`
}

function portalCertListItemDomId(id: number) {
  return `portal-cert-list-item:${id}`
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

function selectedCertIdFromPath(pathname: string) {
  const match = pathname.match(/^\/portal\/cert\/(\d+)$/)
  if (!match) {
    return null
  }
  const id = Number(match[1])
  return Number.isInteger(id) && id > 0 ? id : null
}

function isPortalCertPath(pathname: string) {
  return pathname === '/portal/cert' || pathname.startsWith('/portal/cert/')
}

function formatDateTime(value: string) {
  if (!value) {
    return '-'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return date.toLocaleString()
}

function certToFormValue(cert: PortalCert): PortalCertFormValue {
  return {
    name: cert.name,
    publicKeyBase64: cert.publicKeyBase64,
    privateKeyBase64: '',
  }
}

function formValueToCreation(value: PortalCertFormValue): PortalCertCreation {
  return {
    name: value.name.trim(),
    publicKeyBase64: value.publicKeyBase64.trim(),
    privateKeyBase64: value.privateKeyBase64.trim(),
  }
}

function formValueToUpdate(value: PortalCertFormValue): PortalCertUpdate {
  const creation = formValueToCreation(value)
  const privateKeyBase64 = value.privateKeyBase64.trim()

  return {
    name: creation.name,
    publicKeyBase64: creation.publicKeyBase64,
    privateKeyBase64: privateKeyBase64 === '' ? null : privateKeyBase64,
  }
}

function validateFormValue(
  value: PortalCertFormValue,
  isCreate: boolean,
  t: ReturnType<typeof useLocale>['t'],
) {
  const errors: PortalCertFormErrors = {}

  if (value.name.trim() === '') {
    errors.name = t('portalCert.nameRequired')
  }

  if (value.publicKeyBase64.trim() === '') {
    errors.publicKeyBase64 = t('portalCert.publicKeyRequired')
  }

  if (isCreate && value.privateKeyBase64.trim() === '') {
    errors.privateKeyBase64 = t('portalCert.privateKeyRequired')
  }

  return errors
}

function hasFormErrors(errors: PortalCertFormErrors) {
  return Object.keys(errors).length > 0
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

function PortalCertDialog({
  mode,
  open,
  cert,
  saving,
  onOpenChange,
  onSubmit,
}: {
  mode: 'create' | 'edit'
  open: boolean
  cert: PortalCert | null
  saving: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (value: PortalCertFormValue) => Promise<void>
}) {
  const { t } = useLocale()
  const [formValue, setFormValue] =
    React.useState<PortalCertFormValue>(emptyFormValue)
  const [fieldErrors, setFieldErrors] = React.useState<PortalCertFormErrors>({})
  const [formError, setFormError] = React.useState<string | null>(null)
  const isCreate = mode === 'create'

  React.useEffect(() => {
    if (!open) {
      return
    }

    setFormValue(cert ? certToFormValue(cert) : emptyFormValue)
    setFieldErrors({})
    setFormError(null)
  }, [cert, open])

  const setField = React.useCallback(
    (field: keyof PortalCertFormValue, value: string) => {
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

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const errors = validateFormValue(formValue, isCreate, t)
      if (hasFormErrors(errors)) {
        setFieldErrors(errors)
        return
      }

      setFieldErrors({})
      setFormError(null)

      try {
        await onSubmit(formValue)
      } catch (error) {
        setFormError(getErrorMessage(error))
      }
    },
    [formValue, isCreate, onSubmit],
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-3xl">
        <form className="grid gap-5" onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>
              {isCreate
                ? t('portalCert.createTitle')
                : t('portalCert.editTitle')}
            </DialogTitle>
            <DialogDescription>
              {t('portalCert.dialogDescription')}
            </DialogDescription>
          </DialogHeader>

          {formError ? (
            <Alert variant="destructive">
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          ) : null}

          <Field label={t('portalCert.name')} error={fieldErrors.name}>
            <Input
              aria-invalid={Boolean(fieldErrors.name)}
              value={formValue.name}
              placeholder="demo-cert"
              onChange={(event) => setField('name', event.target.value)}
            />
          </Field>

          <Field
            label={t('portalCert.publicKeyBase64')}
            error={fieldErrors.publicKeyBase64}
          >
            <Textarea
              aria-invalid={Boolean(fieldErrors.publicKeyBase64)}
              value={formValue.publicKeyBase64}
              className="h-36 resize-none overflow-y-auto font-mono text-xs"
              onChange={(event) =>
                setField('publicKeyBase64', event.target.value)
              }
            />
          </Field>

          <Field
            label={t('portalCert.privateKeyBase64')}
            error={fieldErrors.privateKeyBase64}
          >
            <Textarea
              aria-invalid={Boolean(fieldErrors.privateKeyBase64)}
              value={formValue.privateKeyBase64}
              className="h-36 resize-none overflow-y-auto font-mono text-xs"
              placeholder={
                isCreate
                  ? t('portalCert.privateKeyCreatePlaceholder')
                  : t('portalCert.privateKeyEditPlaceholder')
              }
              onChange={(event) =>
                setField('privateKeyBase64', event.target.value)
              }
            />
            <p className="text-xs leading-5 text-muted-foreground">
              {isCreate
                ? t('portalCert.privateKeyCreateHelp')
                : t('portalCert.privateKeyEditHelp')}
            </p>
          </Field>

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

function DeleteCertDialog({
  open,
  cert,
  deleting,
  onOpenChange,
  onConfirm,
}: {
  open: boolean
  cert: PortalCert | null
  deleting: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}) {
  const { t } = useLocale()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('portalCert.deleteTitle')}</DialogTitle>
          <DialogDescription>
            {t('portalCert.deleteDescription')}
          </DialogDescription>
        </DialogHeader>
        <div className="rounded-lg border bg-muted/30 px-3 py-2 text-sm">
          {cert?.name}
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

function PortalCertListSkeleton() {
  return (
    <div className="grid gap-2">
      {Array.from({ length: 5 }).map((_, index) => (
        <Skeleton key={index} className="h-16 w-full" />
      ))}
    </div>
  )
}

function certDomains(cert: PortalCert) {
  return Array.isArray(cert.domains) ? cert.domains : []
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

function PortalCertInlineEditor({
  cert,
  saving,
  onCancel,
  onSubmit,
}: {
  cert: PortalCert | null
  saving: boolean
  onCancel: () => void
  onSubmit: (value: PortalCertFormValue) => Promise<void>
}) {
  const { t } = useLocale()
  const isCreate = cert === null
  const [formValue, setFormValue] = React.useState<PortalCertFormValue>(() =>
    cert ? certToFormValue(cert) : emptyFormValue,
  )
  const [fieldErrors, setFieldErrors] = React.useState<PortalCertFormErrors>({})
  const [formError, setFormError] = React.useState<string | null>(null)

  React.useEffect(() => {
    setFormValue(cert ? certToFormValue(cert) : emptyFormValue)
    setFieldErrors({})
    setFormError(null)
  }, [cert])

  const setField = React.useCallback(
    (field: keyof PortalCertFormValue, value: string) => {
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

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const errors = validateFormValue(formValue, isCreate, t)
      if (hasFormErrors(errors)) {
        setFieldErrors(errors)
        return
      }

      setFieldErrors({})
      setFormError(null)

      try {
        await onSubmit(formValue)
      } catch (error) {
        setFormError(getErrorMessage(error))
      }
    },
    [formValue, isCreate, onSubmit],
  )

  return (
    <form className="grid gap-5" onSubmit={handleSubmit}>
      {formError ? (
        <Alert variant="destructive">
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      <Field label={t('portalCert.name')} error={fieldErrors.name}>
        <Input
          aria-invalid={Boolean(fieldErrors.name)}
          value={formValue.name}
          placeholder="demo-cert"
          onChange={(event) => setField('name', event.target.value)}
        />
      </Field>

      <Field
        label={t('portalCert.publicKeyBase64')}
        error={fieldErrors.publicKeyBase64}
      >
        <Textarea
          aria-invalid={Boolean(fieldErrors.publicKeyBase64)}
          value={formValue.publicKeyBase64}
          className="h-36 resize-none overflow-y-auto font-mono text-xs"
          onChange={(event) => setField('publicKeyBase64', event.target.value)}
        />
      </Field>

      <Field
        label={t('portalCert.privateKeyBase64')}
        error={fieldErrors.privateKeyBase64}
      >
        <Textarea
          aria-invalid={Boolean(fieldErrors.privateKeyBase64)}
          value={formValue.privateKeyBase64}
          className="h-36 resize-none overflow-y-auto font-mono text-xs"
          placeholder={
            isCreate
              ? t('portalCert.privateKeyCreatePlaceholder')
              : t('portalCert.privateKeyEditPlaceholder')
          }
          onChange={(event) => setField('privateKeyBase64', event.target.value)}
        />
        <p className="text-xs leading-5 text-muted-foreground">
          {isCreate
            ? t('portalCert.privateKeyCreateHelp')
            : t('portalCert.privateKeyEditHelp')}
        </p>
      </Field>

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

export function PortalCertPage() {
  const { t } = useLocale()
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const [certs, setCerts] = React.useState<Array<PortalCert>>([])
  const [query, setQuery] = React.useState('')
  const [loading, setLoading] = React.useState(true)
  const listPanel = useResizableListPanel({
    defaultWidth: PORTAL_CERT_LIST_DEFAULT_WIDTH,
    storageKey: PORTAL_CERT_LIST_WIDTH_STORAGE_KEY,
  })
  const handleListScroll = useReservedScrollbar()
  const [saving, setSaving] = React.useState(false)
  const [deleting, setDeleting] = React.useState(false)
  const [editingCert, setEditingCert] = React.useState<PortalCert | null>(null)
  const [deleteCert, setDeleteCert] = React.useState<PortalCert | null>(null)
  const [isCreating, setIsCreating] = React.useState(false)
  const [selectedCertId, setSelectedCertId] = React.useState<number | null>(
    () => selectedCertIdFromPath(window.location.pathname),
  )

  const loadCerts = React.useCallback(async () => {
    setLoading(true)

    try {
      const nextCerts = await portalCertService.list(null)
      setCerts(nextCerts)
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void loadCerts()
  }, [loadCerts])

  const filteredCerts = React.useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (keyword === '') {
      return certs
    }

    return certs.filter((cert) => {
      const values = [
        cert.name,
        cert.issuer,
        certDomains(cert).join(','),
        cert.validFrom,
        cert.validTo,
      ]

      return values.some((value) => value.toLowerCase().includes(keyword))
    })
  }, [certs, query])

  const selectedCert = React.useMemo(
    () =>
      filteredCerts.find((cert) => cert.id === selectedCertId) ??
      filteredCerts[0] ??
      null,
    [filteredCerts, selectedCertId],
  )

  const selectCert = React.useCallback(
    (id: number, replace = false) => {
      setSelectedCertId(id)
      void navigate({ replace, to: portalCertPath(id) })
    },
    [navigate],
  )

  React.useEffect(() => {
    if (!isPortalCertPath(pathname)) {
      return
    }
    setSelectedCertId(selectedCertIdFromPath(pathname))
  }, [pathname])

  React.useEffect(() => {
    if (!isPortalCertPath(pathname)) {
      return
    }
    if (filteredCerts.length === 0) {
      setSelectedCertId(null)
      return
    }

    if (!filteredCerts.some((cert) => cert.id === selectedCertId)) {
      selectCert(filteredCerts[0].id, true)
    }
  }, [filteredCerts, pathname, selectCert, selectedCertId])

  React.useEffect(() => {
    if (selectedCertId == null) {
      return
    }
    window.requestAnimationFrame(() => {
      document
        .getElementById(portalCertListItemDomId(selectedCertId))
        ?.scrollIntoView({
          block: 'nearest',
          inline: 'nearest',
        })
    })
  }, [filteredCerts, selectedCertId])

  const handleCreate = React.useCallback(
    async (value: PortalCertFormValue) => {
      setSaving(true)

      try {
        const created = await portalCertService.create({
          creation: formValueToCreation(value),
        })
        toast.success(t('portalCert.created'))
        setIsCreating(false)
        setCerts((current) => [...current, created])
        selectCert(created.id)
      } catch (error) {
        throw error
      } finally {
        setSaving(false)
      }
    },
    [selectCert],
  )

  const handleUpdate = React.useCallback(
    async (value: PortalCertFormValue) => {
      if (!editingCert) {
        return
      }

      setSaving(true)

      try {
        const updated = await portalCertService.update({
          id: editingCert.id,
          update: formValueToUpdate(value),
        })
        toast.success(t('portalCert.saved'))
        setEditingCert(null)
        setCerts((current) =>
          current.map((cert) => (cert.id === updated.id ? updated : cert)),
        )
      } catch (error) {
        throw error
      } finally {
        setSaving(false)
      }
    },
    [editingCert],
  )

  const handleDelete = React.useCallback(async () => {
    if (!deleteCert) {
      return
    }

    setDeleting(true)

    try {
      await portalCertService.remove({ id: deleteCert.id })
      toast.success(t('portalCert.deleted'))
      setDeleteCert(null)
      setCerts((current) => current.filter((cert) => cert.id !== deleteCert.id))
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setDeleting(false)
    }
  }, [deleteCert])

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
                  placeholder={t('portalCert.searchPlaceholder')}
                  onChange={(event) => setQuery(event.target.value)}
                />
              </div>
              <div className="flex items-center justify-between gap-2">
                <div className="text-xs text-muted-foreground">
                  {t('portalCert.itemCount').replace(
                    '{count}',
                    String(certs.length),
                  )}
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-7"
                    title={t('action.refreshList')}
                    onClick={() => void loadCerts()}
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
                      setEditingCert(null)
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
                <PortalCertListSkeleton />
              ) : filteredCerts.length === 0 ? (
                <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                  {certs.length === 0
                    ? t('portalCert.empty')
                    : t('portalCert.noMatch')}
                </div>
              ) : (
                <div className="space-y-1">
                  {filteredCerts.map((cert) => (
                    <a
                      key={cert.id}
                      id={portalCertListItemDomId(cert.id)}
                      href={portalCertPath(cert.id)}
                      onClick={(event) => {
                        if (shouldUseBrowserNavigation(event)) {
                          return
                        }
                        event.preventDefault()
                        selectCert(cert.id)
                      }}
                      className={cn(
                        'flex w-full flex-col gap-1 rounded-lg border px-3 py-2.5 text-left transition-colors',
                        selectedCert?.id === cert.id
                          ? 'border-primary/30 bg-primary/[0.06]'
                          : 'border-transparent hover:bg-primary/[0.05]',
                      )}
                    >
                      <span className="truncate text-sm font-medium">
                        {cert.name}
                      </span>
                      <span className="truncate text-xs text-muted-foreground">
                        {certDomains(cert).join(', ') || cert.issuer}
                      </span>
                    </a>
                  ))}
                </div>
              )}
            </div>
            <ResizableListHandle
              defaultWidth={PORTAL_CERT_LIST_DEFAULT_WIDTH}
              label={t('portalCert.resizeList')}
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
                    <KeyRound className="size-4 shrink-0 text-primary" />
                    <h2 className="text-base font-semibold">
                      {t('portalCert.createTitle')}
                    </h2>
                  </div>
                </div>
                <div className="scrollbar-reserved min-h-0 flex-1 overflow-y-auto p-6 pr-4">
                  <PortalCertInlineEditor
                    cert={null}
                    saving={saving}
                    onCancel={() => setIsCreating(false)}
                    onSubmit={handleCreate}
                  />
                </div>
              </div>
            ) : selectedCert ? (
              <div className="flex h-full min-h-0 flex-col">
                <div className="border-b border-border/70 px-6 py-4">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <KeyRound className="size-4 shrink-0 text-primary" />
                        <h2 className="min-w-0 truncate text-base font-semibold">
                          {selectedCert.name}
                        </h2>
                        {selectedCert.privateKeyConfigured ? (
                          <Badge variant="secondary">
                            <BadgeCheck />
                            {t('portalCert.privateKeyConfigured')}
                          </Badge>
                        ) : (
                          <Badge variant="destructive">
                            {t('portalCert.privateKeyMissing')}
                          </Badge>
                        )}
                      </div>
                      <p className="mt-2 font-mono text-xs text-muted-foreground">
                        #{selectedCert.id}
                      </p>
                    </div>
                    {editingCert?.id === selectedCert.id ? null : (
                      <div className="flex items-center gap-1">
                        <Tooltip>
                          <TooltipTrigger
                            render={
                              <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => setEditingCert(selectedCert)}
                              />
                            }
                          >
                            <Edit3 />
                            {t('action.edit')}
                          </TooltipTrigger>
                          <TooltipContent>{t('action.edit')}</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger
                            render={
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon-sm"
                                onClick={() => setDeleteCert(selectedCert)}
                              />
                            }
                          >
                            <Trash2 />
                          </TooltipTrigger>
                          <TooltipContent>{t('action.delete')}</TooltipContent>
                        </Tooltip>
                      </div>
                    )}
                  </div>
                </div>

                <div className="scrollbar-reserved min-h-0 flex-1 overflow-y-auto p-6 pr-4">
                  {editingCert?.id === selectedCert.id ? (
                    <PortalCertInlineEditor
                      cert={selectedCert}
                      saving={saving}
                      onCancel={() => setEditingCert(null)}
                      onSubmit={handleUpdate}
                    />
                  ) : (
                    <div className="grid gap-5">
                      <ReadonlyField label={t('portalCert.name')}>
                        {selectedCert.name}
                      </ReadonlyField>
                      <div className="grid gap-4 sm:grid-cols-2">
                        <ReadonlyField label={t('portalCert.issuer')}>
                          {selectedCert.issuer || t('portalCert.unparsed')}
                        </ReadonlyField>
                        <ReadonlyField label={t('portalCert.validity')}>
                          <span className="text-muted-foreground">
                            {formatDateTime(selectedCert.validFrom)}{' '}
                            {t('common.to')}{' '}
                            {formatDateTime(selectedCert.validTo)}
                          </span>
                        </ReadonlyField>
                      </div>
                      <ReadonlyField
                        label={t('portalCert.domains')}
                        className="min-h-20"
                      >
                        <div className="flex flex-wrap gap-1">
                          {certDomains(selectedCert).length > 0 ? (
                            certDomains(selectedCert).map((domain) => (
                              <Badge key={domain} variant="outline">
                                {domain}
                              </Badge>
                            ))
                          ) : (
                            <span className="text-muted-foreground">
                              {t('portalCert.unparsed')}
                            </span>
                          )}
                        </div>
                      </ReadonlyField>
                      <ReadonlyField
                        label={t('portalCert.publicKeyBase64')}
                        className="h-36 overflow-y-auto whitespace-pre-wrap break-all font-mono text-xs"
                      >
                        {selectedCert.publicKeyBase64 || (
                          <span className="font-sans text-sm text-muted-foreground">
                            {t('status.unconfigured')}
                          </span>
                        )}
                      </ReadonlyField>
                      <ReadonlyField
                        label={t('portalCert.privateKeyBase64')}
                        className="min-h-20"
                      >
                        <div className="grid gap-2">
                          <div>
                            {selectedCert.privateKeyConfigured ? (
                              <Badge variant="secondary">
                                <BadgeCheck />
                                {t('portalCert.privateKeyConfigured')}
                              </Badge>
                            ) : (
                              <Badge variant="destructive">
                                {t('portalCert.privateKeyMissing')}
                              </Badge>
                            )}
                          </div>
                          <p className="text-xs leading-5 text-muted-foreground">
                            {t('portalCert.privateKeyHidden')}
                          </p>
                        </div>
                      </ReadonlyField>
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <Empty className="h-full min-h-[24rem] rounded-none border-0">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <KeyRound />
                  </EmptyMedia>
                  <EmptyTitle>
                    {certs.length === 0
                      ? t('portalCert.empty')
                      : t('portalCert.noMatch')}
                  </EmptyTitle>
                  <EmptyDescription>
                    {certs.length === 0
                      ? t('portalCert.emptyDescription')
                      : t('common.adjustSearch')}
                  </EmptyDescription>
                </EmptyHeader>
                {certs.length === 0 ? (
                  <EmptyContent>
                    <Button type="button" onClick={() => setIsCreating(true)}>
                      <Plus />
                      {t('portalCert.createTitle')}
                    </Button>
                  </EmptyContent>
                ) : null}
              </Empty>
            )}
          </main>
        </div>

        <PortalCertDialog
          mode="edit"
          open={false}
          cert={editingCert}
          saving={saving}
          onOpenChange={(open) => {
            if (!open) {
              setEditingCert(null)
            }
          }}
          onSubmit={handleUpdate}
        />
        <DeleteCertDialog
          open={deleteCert !== null}
          cert={deleteCert}
          deleting={deleting}
          onOpenChange={(open) => {
            if (!open) {
              setDeleteCert(null)
            }
          }}
          onConfirm={handleDelete}
        />
      </section>
    </TooltipProvider>
  )
}
