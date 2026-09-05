import * as React from 'react'
import { ArrowRight } from 'lucide-react'

import { useLocale } from '@/i18n'
import { previewRulePath } from './path-preview'

export function RulePathPreview({ matchPathPrefix, routePathPrefix }: {
  matchPathPrefix: string
  routePathPrefix: string
}) {
  const { t } = useLocale()
  const titleId = React.useId()
  const request = `${matchPathPrefix === '/' ? '' : matchPathPrefix}/example`
  const result = previewRulePath(request, matchPathPrefix, routePathPrefix)
  const error = result.kind === 'invalidRequest' || result.kind === 'invalidRoute'
  return (
    <section aria-labelledby={titleId} className="grid min-w-0 gap-3 border-t pt-5">
      <h3 id={titleId} className="text-sm font-medium">{t('portalRule.pathPreview')}</h3>
      <div className="grid min-w-0 gap-3 rounded-lg bg-muted/40 p-4 sm:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] sm:gap-5">
        <div className="grid min-w-0 content-start gap-2">
          <span className="text-xs text-muted-foreground">{t('portalRule.pathPreviewRequest')}</span>
          <code className="min-w-0 break-all text-sm">{request}</code>
        </div>
        <ArrowRight aria-hidden="true" className="size-4 rotate-90 self-center text-muted-foreground sm:rotate-0" />
        <div className="grid min-w-0 content-start gap-2">
          <span className="text-xs text-muted-foreground">{t('portalRule.pathPreviewResult')}</span>
          <div aria-live="polite" role="status" className={`min-w-0 break-all text-sm ${error ? 'text-destructive' : 'text-muted-foreground'}`}>
            {result.kind === 'matched' ? (
              <code className="text-primary">{result.path}</code>
            ) : result.kind === 'empty' ? t('portalRule.pathPreviewEmpty')
              : result.kind === 'noMatch' ? t('portalRule.pathPreviewNoMatch')
              : result.kind === 'invalidRoute' ? t('portalRule.routePathPrefixInvalid')
              : t('portalRule.pathPreviewInvalid')}
          </div>
        </div>
      </div>
      <p className="text-xs text-muted-foreground">{t('portalRule.pathPreviewHelp')}</p>
    </section>
  )
}
