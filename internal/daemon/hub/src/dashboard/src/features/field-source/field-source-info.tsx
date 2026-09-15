import * as React from 'react'
import { Info } from 'lucide-react'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { useLocale } from '@/i18n'
import type { FieldSource } from '@/skeled/admin'

export function FieldSourceInfo({ fields, path }: { fields: Array<FieldSource>; path: string }) {
  const { t } = useLocale()
  const [open, setOpen] = React.useState(false)
  const matching = fields.filter((field) =>
    (field.define || field.override || field.variables.length > 0) &&
    (field.path === path || field.path.startsWith(path + '/') || path.startsWith(field.path + '/')),
  )
  const origins = [...new Map(matching.map((field) => [JSON.stringify([field.source, field.define, field.override, field.variables, field.template, field.bindings]), field])).values()]
  if (origins.length === 0) return null
  return <Tooltip open={open} onOpenChange={setOpen}>
    <TooltipTrigger render={<button type="button" aria-label={`${t('fieldSource.title')} · ${path.slice(1)}`} className="inline-flex size-4 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />}>
      <Info className="size-3.5" />
    </TooltipTrigger>
    <TooltipContent side="top" align="start" className="block max-w-sm whitespace-normal">
      <div className="grid gap-2">{origins.map((field) => <div key={field.path} className="grid gap-1">
          {origins.length > 1 && <span className="font-mono break-all">{field.path}</span>}
          <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
            <dt className="font-mono">source</dt><dd className="break-all">{field.source || '—'}</dd>
            <dt className="font-mono">define</dt><dd className="break-all">{field.define || '—'}</dd>
            <dt className="font-mono">override</dt><dd className="break-all">{field.override || '—'}</dd>
            {field.template != null && <><dt className="font-mono">template</dt><dd className="break-all whitespace-pre-wrap font-mono">{field.template}</dd></>}
            {field.bindings.map((binding, index) => <React.Fragment key={index}><dt className="font-mono">{binding.variable}</dt><dd className="break-all whitespace-pre-wrap font-mono">{binding.path && `${binding.path}: `}{binding.reference} = {binding.value}{binding.defaultUsed && ' (default)'}</dd></React.Fragment>)}
            {field.bindings.length === 0 && field.variables.length > 0 && <><dt className="font-mono">variables</dt><dd className="break-all">{field.variables.join(', ')}</dd></>}
          </dl>
        </div>)}</div>
    </TooltipContent>
  </Tooltip>
}
