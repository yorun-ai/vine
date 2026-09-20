import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { useLocale } from '@/i18n'

// EnabledField is the switch every Portal entity form shares. Hub keeps a
// disabled entity in its database and stops publishing it to Portal, so Portal
// never sees it.
export function EnabledField({
  enabled,
  id,
  onChange,
}: {
  enabled: boolean
  id: string
  onChange: (enabled: boolean) => void
}) {
  const { t } = useLocale()
  return (
    <div className="grid gap-2">
      <div className="flex items-center gap-2">
        <Switch
          id={id}
          checked={enabled}
          onCheckedChange={(checked) => onChange(checked === true)}
        />
        <Label htmlFor={id}>{t('common.enabled')}</Label>
      </div>
      <p className="text-xs leading-5 text-muted-foreground">
        {t('common.enabledHelp')}
      </p>
    </div>
  )
}
