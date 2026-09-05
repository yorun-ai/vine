import { useLocale } from '@/i18n'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export function DomainFilter({
  domains,
  query,
  onQueryChange,
  loading,
}: {
  domains: string[]
  query: string
  onQueryChange: (query: string) => void
  loading: boolean
}) {
  const { t } = useLocale()

  return (
    <Select
      value={domains.includes(query) ? query : null}
      onValueChange={(domain) => onQueryChange(domain ?? '')}
      disabled={loading || domains.length === 0}
    >
      <SelectTrigger
        size="sm"
        className="w-32 min-w-0 shrink-0 text-left text-xs"
        aria-label={t('common.selectDomain')}
      >
        <SelectValue
          className="min-w-0 truncate"
          placeholder={t('common.selectDomain')}
        />
      </SelectTrigger>
      <SelectContent align="start" alignItemWithTrigger={false}>
        <SelectItem value="">{t('common.allDomains')}</SelectItem>
        {domains.map((domain) => (
          <SelectItem key={domain} value={domain}>
            {domain}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
