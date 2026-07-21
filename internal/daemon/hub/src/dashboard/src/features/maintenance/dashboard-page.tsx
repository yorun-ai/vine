import * as React from 'react'
import { ExternalLink, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { vrpcClient } from '@/config/vrpc-client'
import { useLocale } from '@/i18n'
import {
  createPortalCertService,
  createPortalRuleService,
} from '@/skeled'
import type { PortalCert } from '@/skeled'

const portalRuleService = createPortalRuleService(vrpcClient)
const portalCertService = createPortalCertService(vrpcClient)
const dashboardRedirectSeconds = 3
const dashboardReadyProbeMaxAttempts = 10
const dashboardReadyProbeAsset = '/brand/vinehub.png'

type DashboardScheme = 'http' | 'https'

interface DashboardRedirectState {
  attempts: number
  countdown: number
  failed: boolean
  ready: boolean
  targetUrl: string
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function normalizedPathPrefix(pathPrefix: string) {
  const trimmed = pathPrefix.trim()
  if (trimmed === '') {
    return '/'
  }
  return trimmed.startsWith('/') ? trimmed : `/${trimmed}`
}

function nextDashboardUrl({
  host,
  pathPrefix,
  port,
  scheme,
}: {
  host: string
  pathPrefix: string
  port: number
  scheme: DashboardScheme
}) {
  const url = new URL(window.location.href)
  url.protocol = `${scheme}:`
  if (host.trim() !== '') {
    url.hostname = host.trim()
  }
  url.port = port > 0 ? String(port) : ''
  url.pathname = normalizedPathPrefix(pathPrefix)
  url.search = ''
  url.hash = ''
  return url.toString()
}

function dashboardReadyProbeUrl(targetUrl: string) {
  const url = new URL(targetUrl)
  const basePath = normalizedPathPrefix(url.pathname)
  url.pathname =
    basePath === '/'
      ? dashboardReadyProbeAsset
      : `${basePath.replace(/\/$/, '')}${dashboardReadyProbeAsset}`
  url.search = `?t=${Date.now()}`
  url.hash = ''
  return url.toString()
}

function probeDashboardReady(targetUrl: string) {
  return new Promise<boolean>((resolve) => {
    const image = new Image()
    image.onload = () => resolve(true)
    image.onerror = () => resolve(false)
    image.src = dashboardReadyProbeUrl(targetUrl)
  })
}

function validPortValue(value: string) {
  if (value.trim() === '') {
    return true
  }
  const port = Number(value)
  return Number.isInteger(port) && port >= 1 && port <= 65535
}

function parsedPortValue(value: string) {
  return value.trim() === '' ? 0 : Number(value)
}

function certDomainMatchesHost(domain: string, host: string) {
  const normalizedDomain = domain.trim().toLowerCase()
  const normalizedHost = host.trim().toLowerCase()
  if (normalizedDomain === '' || normalizedHost === '') {
    return false
  }
  if (normalizedDomain === normalizedHost) {
    return true
  }
  if (normalizedDomain.startsWith('*.')) {
    const suffix = normalizedDomain.slice(1)
    return (
      normalizedHost.endsWith(suffix) &&
      normalizedHost.split('.').length === normalizedDomain.split('.').length
    )
  }
  return false
}

function hasConfiguredCertForHost(certs: Array<PortalCert>, host: string) {
  return certs.some(
    (cert) =>
      cert.privateKeyConfigured &&
      cert.domains.some((domain) => certDomainMatchesHost(domain, host)),
  )
}

export function DashboardSettingsPage() {
  const { t } = useLocale()
  const [schemeValue, setSchemeValue] = React.useState<DashboardScheme>(() =>
    window.location.protocol === 'https:' ? 'https' : 'http',
  )
  const [portValue, setPortValue] = React.useState(() => window.location.port)
  const [hostValue, setHostValue] = React.useState(
    () => window.location.hostname,
  )
  const [pathPrefixValue, setPathPrefixValue] = React.useState('/')
  const [saving, setSaving] = React.useState(false)
  const [loadingAccess, setLoadingAccess] = React.useState(true)
  const [canUpdateAccess, setCanUpdateAccess] = React.useState(true)
  const [error, setError] = React.useState<string | null>(null)
  const [portError, setPortError] = React.useState<string | null>(null)
  const [hostError, setHostError] = React.useState<string | null>(null)
  const [redirect, setRedirect] = React.useState<DashboardRedirectState | null>(
    null,
  )

  const parsedPort = parsedPortValue(portValue)
  const accessDisabled = loadingAccess || saving || !canUpdateAccess

  const jumpToDashboardPort = React.useCallback(() => {
    if (!redirect) {
      return
    }
    window.location.assign(redirect.targetUrl)
  }, [redirect])

  React.useEffect(() => {
    if (!redirect) {
      return
    }

    if (redirect.countdown <= 0 && redirect.ready) {
      window.location.assign(redirect.targetUrl)
      return
    }

    if (redirect.countdown <= 0 || redirect.ready || redirect.failed) {
      return
    }

    const timer = window.setTimeout(() => {
      setRedirect((current) =>
        current
          ? { ...current, countdown: Math.max(0, current.countdown - 1) }
          : current,
      )
    }, 1000)

    return () => window.clearTimeout(timer)
  }, [redirect])

  React.useEffect(() => {
    if (
      !redirect ||
      redirect.countdown > 0 ||
      redirect.ready ||
      redirect.failed
    ) {
      return
    }

    let cancelled = false
    void probeDashboardReady(redirect.targetUrl).then((ready) => {
      if (cancelled) {
        return
      }

      setRedirect((current) => {
        if (!current) {
          return current
        }
        if (ready) {
          return { ...current, ready: true }
        }
        const attempts = current.attempts + 1
        return {
          ...current,
          attempts,
          countdown:
            attempts >= dashboardReadyProbeMaxAttempts
              ? current.countdown
              : dashboardRedirectSeconds,
          failed: attempts >= dashboardReadyProbeMaxAttempts,
        }
      })
    })

    return () => {
      cancelled = true
    }
  }, [
    redirect?.countdown,
    redirect?.failed,
    redirect?.ready,
    redirect?.targetUrl,
  ])

  React.useEffect(() => {
    let cancelled = false
    setLoadingAccess(true)

    void portalRuleService
      .getDashboardAccess(null)
      .then((access) => {
        if (cancelled) {
          return
        }
        setSchemeValue(access.scheme === 'https' ? 'https' : 'http')
        setHostValue(access.host)
        setPortValue(access.port > 0 ? String(access.port) : '')
        setPathPrefixValue(normalizedPathPrefix(access.pathPrefix))
        setCanUpdateAccess(access.canUpdate)
      })
      .catch((nextError) => {
        if (cancelled) {
          return
        }
        setError(getErrorMessage(nextError))
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingAccess(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [])

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      if (!canUpdateAccess) {
        return
      }

      if (!validPortValue(portValue)) {
        setPortError(t('dashboardPort.invalid'))
        return
      }

      setSaving(true)
      setError(null)
      setPortError(null)
      setHostError(null)

      if (schemeValue === 'https') {
        const trimmedHost = hostValue.trim()
        if (trimmedHost === '') {
          setHostError(t('dashboardAccess.hostRequired'))
          setSaving(false)
          return
        }
        try {
          const certs = await portalCertService.list(null)
          if (!hasConfiguredCertForHost(certs, trimmedHost)) {
            setHostError(t('dashboardAccess.missingCert'))
            setSaving(false)
            return
          }
        } catch (nextError) {
          setError(getErrorMessage(nextError))
          setSaving(false)
          return
        }
      }

      try {
        await portalRuleService.updateDashboardAccess({
          scheme: schemeValue,
          host: hostValue.trim(),
          port: parsedPort,
          pathPrefix: normalizedPathPrefix(pathPrefixValue),
        })
        toast.success(t('dashboardPort.updatedTitle'))
        setRedirect({
          attempts: 0,
          countdown: dashboardRedirectSeconds,
          failed: false,
          ready: false,
          targetUrl: nextDashboardUrl({
            scheme: schemeValue,
            host: hostValue,
            port: parsedPort,
            pathPrefix: pathPrefixValue,
          }),
        })
      } catch (nextError) {
        setError(getErrorMessage(nextError))
      } finally {
        setSaving(false)
      }
    },
    [
      canUpdateAccess,
      hostValue,
      parsedPort,
      pathPrefixValue,
      portValue,
      schemeValue,
      t,
    ],
  )

  return (
    <section className="flex h-[calc(100dvh-3.5rem)] flex-col overflow-hidden bg-white">
      <div className="min-h-0 flex-1 overflow-y-auto p-6">
        <div className="grid max-w-3xl gap-6">
          {error ? (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          <form className="grid gap-5 pb-6" onSubmit={handleSubmit}>
            <p className="text-sm text-muted-foreground">
              {t('dashboardAccess.description')}
            </p>

            {!loadingAccess && !canUpdateAccess ? (
              <Alert variant="destructive">
                <AlertDescription>{t('dashboardAccess.locked')}</AlertDescription>
              </Alert>
            ) : null}

            <div className="grid max-w-xl gap-5">
              <div className="grid gap-2">
                <Label htmlFor="dashboard-scheme">
                  {t('portalRule.scheme')}
                </Label>
                <Select
                  value={schemeValue}
                  disabled={accessDisabled}
                  onValueChange={(value) => {
                    setSchemeValue(value === 'https' ? 'https' : 'http')
                    setHostError(null)
                    setError(null)
                  }}
                >
                  <SelectTrigger id="dashboard-scheme" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="http">http</SelectItem>
                    <SelectItem value="https">https</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="dashboard-port">
                  {t('dashboardPort.portLabel')}
                </Label>
                <Input
                  id="dashboard-port"
                  value={portValue}
                  inputMode="numeric"
                  disabled={accessDisabled}
                  aria-invalid={Boolean(portError)}
                  onChange={(event) => {
                    setPortValue(event.target.value)
                    setPortError(null)
                    setError(null)
                  }}
                  placeholder={t('portalRule.portPlaceholder')}
                />
                {portError ? (
                  <p className="text-xs text-destructive">{portError}</p>
                ) : (
                  <p className="text-xs text-muted-foreground">
                    {t('dashboardPort.currentPort').replace(
                      '{port}',
                      window.location.port || t('dashboardPort.defaultPort'),
                    )}
                  </p>
                )}
              </div>

              <div className="grid gap-2">
                <Label htmlFor="dashboard-host">{t('portalRule.host')}</Label>
                <Input
                  id="dashboard-host"
                  value={hostValue}
                  disabled={accessDisabled}
                  aria-invalid={Boolean(hostError)}
                  placeholder={t('portalRule.hostPlaceholder')}
                  onChange={(event) => {
                    setHostValue(event.target.value)
                    setHostError(null)
                    setError(null)
                  }}
                />
                {hostError ? (
                  <p className="text-xs text-destructive">{hostError}</p>
                ) : null}
              </div>

              <div className="grid gap-2">
                <Label htmlFor="dashboard-path-prefix">
                  {t('portalRule.pathPrefix')}
                </Label>
                <Input
                  id="dashboard-path-prefix"
                  value={pathPrefixValue}
                  disabled={accessDisabled}
                  placeholder={t('portalRule.pathPrefixPlaceholder')}
                  onChange={(event) => {
                    setPathPrefixValue(event.target.value)
                    setError(null)
                  }}
                />
              </div>
            </div>

            <div className="flex items-center gap-2">
              <Button
                type="submit"
                disabled={accessDisabled || !validPortValue(portValue)}
              >
                {saving || loadingAccess ? (
                  <Loader2 className="animate-spin" />
                ) : null}
                {t('action.save')}
              </Button>
            </div>
          </form>
        </div>
      </div>

      {redirect ? (
        <div className="fixed inset-0 z-50 grid place-items-center bg-black/10 p-6 backdrop-blur-xs">
          <div className="grid w-full max-w-md gap-4 rounded-md border bg-background p-5 shadow-lg">
            <div className="grid gap-1">
              <h2 className="text-base font-semibold">
                {t('dashboardPort.updatedTitle')}
              </h2>
              <p className="text-sm text-muted-foreground">
                {t('dashboardPort.updatedDescription')}
              </p>
            </div>
            <div className="grid gap-2 rounded-md border bg-muted/30 p-3">
              <div className="text-2xl font-semibold tabular-nums">
                {redirect.ready
                  ? 'Ready'
                  : redirect.failed
                    ? t('dashboardPort.notReady')
                    : `${redirect.countdown}s`}
              </div>
              <a
                href={redirect.targetUrl}
                className="inline-flex min-w-0 items-center gap-1 font-mono text-xs text-primary underline-offset-4 hover:underline"
              >
                <span className="truncate">{redirect.targetUrl}</span>
                <ExternalLink className="size-3 shrink-0" />
              </a>
            </div>
            <div className="flex justify-end gap-2">
              {redirect.failed ? (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setRedirect(null)}
                >
                  {t('dashboardPort.stay')}
                </Button>
              ) : null}
              <Button type="button" onClick={jumpToDashboardPort}>
                {t('dashboardPort.jumpNow')}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  )
}
