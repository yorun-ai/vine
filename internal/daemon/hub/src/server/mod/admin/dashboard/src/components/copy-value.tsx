import { Copy } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { copyTextToClipboard } from '@/lib/clipboard'
import { useLocale } from '@/i18n'

export function CopyValue({ value }: { value: string }) {
  const { t } = useLocale()
  return (
    <span className="inline-flex max-w-full items-start gap-2">
      <span className="min-w-0 break-all font-mono" title={value}>{value}</span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-6 shrink-0 text-muted-foreground"
        aria-label={t('common.copyValue')}
        title={t('common.copyValue')}
        onClick={async () => {
          try {
            await copyTextToClipboard(value)
            toast.success(t('common.copied'))
          } catch {
            toast.error(t('common.copyFailed'))
          }
        }}
      >
        <Copy className="size-3.5" />
      </Button>
    </span>
  )
}
