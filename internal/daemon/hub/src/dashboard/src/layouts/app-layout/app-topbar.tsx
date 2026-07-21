import * as React from 'react'
import { ChevronRight, Menu } from 'lucide-react'

import { cn } from '@/lib/utils'
import { useLocale } from '@/i18n'

import { AccountPopover } from './account-popover'
import { LanguageToggle } from './language-toggle'
import type { BreadcrumbItem } from './nav-matching'
import type { AppNavItem, AppScene } from './nav-config'

interface AppTopbarProps {
  activeItem: AppNavItem | null
  activeScene: AppScene
  breadcrumbItems: Array<BreadcrumbItem>
  isMobile: boolean
  sidebarState: 'expanded' | 'collapsed'
  onToggleSidebar: () => void
}

export function AppTopbar({
  activeItem,
  activeScene,
  breadcrumbItems,
  isMobile,
  sidebarState,
  onToggleSidebar,
}: AppTopbarProps) {
  const [isSidebarIconHovered, setIsSidebarIconHovered] = React.useState(false)
  const { t } = useLocale()
  const SidebarIcon =
    isSidebarIconHovered || isMobile
      ? Menu
      : (activeItem?.icon ?? activeScene.icon)

  return (
    <header className="flex h-14 shrink-0 items-center justify-between border-b border-border bg-background px-4">
      <div className="flex min-w-0 items-center gap-3">
        <button
          type="button"
          aria-label={
            isMobile
              ? t('topbar.openSidebar')
              : sidebarState === 'collapsed'
                ? t('topbar.expandSidebar')
                : t('topbar.collapseSidebar')
          }
          className="flex h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-[10px] bg-primary/10 text-primary transition hover:bg-primary/[0.14] hover:text-primary active:translate-y-0"
          onClick={onToggleSidebar}
          onMouseEnter={() => setIsSidebarIconHovered(true)}
          onMouseLeave={() => setIsSidebarIconHovered(false)}
        >
          <SidebarIcon className="size-5" />
        </button>

        <div className="flex min-w-0 items-center gap-2 overflow-hidden text-base font-medium">
          {breadcrumbItems.map((item, index) => {
            const isLast = index === breadcrumbItems.length - 1

            return (
              <React.Fragment key={`${item.label}-${index}`}>
                {index > 0 ? (
                  <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
                ) : null}
                <span
                  className={cn(
                    'truncate',
                    isLast
                      ? 'font-semibold text-foreground'
                      : 'text-muted-foreground',
                  )}
                >
                  {item.label}
                </span>
              </React.Fragment>
            )
          })}
        </div>
      </div>

      <div className="flex items-center gap-2">
        <LanguageToggle />
        <div className="pl-1">
          <AccountPopover />
        </div>
      </div>
    </header>
  )
}
