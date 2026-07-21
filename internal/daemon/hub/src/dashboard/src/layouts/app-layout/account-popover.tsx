import { UserRound } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { useLocale } from '@/i18n'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

import { MOCK_USER } from './nav-config'

export function AccountPopover() {
  const { t } = useLocale()

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            aria-label={t('account.menu')}
            className="h-8 gap-2 px-2 active:translate-y-0"
          />
        }
      >
        <span className="flex size-6 shrink-0 items-center justify-center rounded-full border border-border bg-primary/10 text-primary">
          <UserRound className="size-3.5" />
        </span>
        <span className="max-w-28 truncate text-sm font-medium text-foreground">
          {MOCK_USER.name}
        </span>
      </PopoverTrigger>

      <PopoverContent
        side="bottom"
        align="end"
        sideOffset={8}
        className="w-[280px] rounded-[12px] border border-border/70 p-0 shadow-[0_16px_40px_rgba(15,23,42,0.12)]"
      >
        <div className="p-2.5">
          <div className="rounded-[10px] bg-[#f7fbfc] px-3 py-2.5">
            <div className="flex items-start gap-3">
              <span className="flex size-11 shrink-0 items-center justify-center rounded-full border border-[#dce9ef] bg-white text-primary">
                <UserRound className="size-5" />
              </span>
              <div className="min-w-0 flex-1">
                <div className="min-w-0">
                  <div className="truncate text-[15px] leading-5 font-semibold text-foreground">
                    {MOCK_USER.name}
                  </div>
                </div>
                <div className="mt-0.5 truncate text-xs leading-4 text-muted-foreground">
                  {MOCK_USER.email}
                </div>
              </div>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
