import { cn } from '@/lib/utils'
import { useLocale, type Locale } from '@/i18n'

const localeOptions: Array<{ label: string; value: Locale }> = [
  { label: 'EN', value: 'en' },
  { label: 'CN', value: 'cn' },
]

export function LanguageToggle() {
  const { locale, setLocale } = useLocale()
  const nextLocale = locale === 'en' ? 'cn' : 'en'
  const nextLocaleLabel = nextLocale === 'en' ? 'English' : 'Chinese'

  return (
    <button
      type="button"
      className="flex h-8 items-center rounded-md border border-border bg-background p-0.5 transition-colors hover:bg-primary/[0.04]"
      aria-label={`Switch language to ${nextLocaleLabel}`}
      onClick={() => setLocale(nextLocale)}
    >
      {localeOptions.map((option) => (
        <span
          key={option.value}
          className={cn(
            'flex h-6 items-center rounded px-2 text-xs font-semibold transition-colors',
            locale === option.value
              ? 'bg-primary text-primary-foreground'
              : 'text-muted-foreground',
          )}
        >
          {option.label}
        </span>
      ))}
    </button>
  )
}
