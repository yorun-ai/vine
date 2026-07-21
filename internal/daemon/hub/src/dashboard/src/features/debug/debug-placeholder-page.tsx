import type { LucideIcon } from 'lucide-react'

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import type { TranslationKey } from '@/i18n'
import { useLocale } from '@/i18n'

interface DebugPlaceholderPageProps {
  icon: LucideIcon
  titleKey: TranslationKey
  descriptionKey: TranslationKey
}

export function DebugPlaceholderPage({
  icon: Icon,
  titleKey,
  descriptionKey,
}: DebugPlaceholderPageProps) {
  const { t } = useLocale()

  return (
    <div className="flex min-h-full flex-col p-6">
      <Empty className="min-h-[360px] rounded-none border-0">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Icon />
          </EmptyMedia>
          <EmptyTitle>{t(titleKey)}</EmptyTitle>
          <EmptyDescription>{t(descriptionKey)}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    </div>
  )
}
