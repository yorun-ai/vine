import { TriangleAlert } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'

interface DeprecatedProps {
  deprecated: boolean
  deprecatedReason?: string | null
}

export function DeprecatedBadge({
  deprecated,
}: Pick<DeprecatedProps, 'deprecated'>) {
  const { t } = useLocale()

  if (!deprecated) {
    return null
  }

  return (
    <Badge
      variant="outline"
      className="border-destructive/40 bg-destructive/5 text-destructive"
    >
      <TriangleAlert data-icon="inline-start" />
      {t('common.deprecated')}
    </Badge>
  )
}

export function DeprecatedReason({
  deprecated,
  deprecatedReason,
  className,
}: DeprecatedProps & { className?: string }) {
  if (!deprecated || !deprecatedReason) {
    return null
  }

  return (
    <div className={cn('text-xs text-destructive', className)}>
      {deprecatedReason}
    </div>
  )
}

export function DeprecatedNotice({
  deprecated,
  deprecatedReason,
  className,
}: DeprecatedProps & { className?: string }) {
  const { t } = useLocale()

  if (!deprecated) {
    return null
  }

  return (
    <div
      role="note"
      className={cn(
        'flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2.5 text-destructive',
        className,
      )}
    >
      <TriangleAlert className="mt-0.5 size-4 shrink-0" />
      <div className="grid min-w-0 gap-0.5">
        <div className="text-sm font-medium">{t('common.deprecated')}</div>
        {deprecatedReason ? (
          <div className="text-sm text-destructive/90">
            {deprecatedReason}
          </div>
        ) : null}
      </div>
    </div>
  )
}
