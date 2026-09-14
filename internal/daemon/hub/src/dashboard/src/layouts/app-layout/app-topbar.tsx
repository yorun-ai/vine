import * as React from 'react'
import { ChevronRight, LockKeyhole, Menu } from 'lucide-react'

import { cn } from '@/lib/utils'
import { useConfigAccess } from '@/lib/config-access'
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
  const configAccess = useConfigAccess()
  const isConfigurationPage = activeItem !== null && [
    'app-config',
    'portal-entry',
    'portal-rule',
    'portal-site',
    'portal-cert',
    'settings-dashboard-port',
    'maintenance',
  ].includes(activeItem.id)
  const SidebarIcon =
    isSidebarIconHovered || isMobile
      ? Menu
      : (activeItem?.icon ?? activeScene.icon)

  return (
    <header className="flex min-h-14 shrink-0 flex-wrap items-center justify-between gap-x-4 gap-y-2 border-b border-border bg-background px-4 py-2">
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

      {isConfigurationPage && !configAccess.loading && !configAccess.error && configAccess.readOnly ? (
        <div className="order-last flex w-full min-w-0 justify-center lg:order-none lg:w-auto lg:flex-1">
          <div
            role="status"
            className="flex items-center gap-2 rounded-md border border-amber-300 bg-amber-100 px-3 py-1.5 text-xs text-amber-950 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-100"
          >
            <LockKeyhole className="size-4 shrink-0" aria-hidden="true" />
            <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
              <span className="font-semibold">
                {t('configAccess.readOnlyTitle')}
              </span>
              <span>
                {t('configAccess.readOnly')}
              </span>
            </div>
          </div>
        </div>
      ) : null}

      <div className="flex shrink-0 items-center gap-2">
        <LanguageToggle />
        <div className="pl-1">
          <AccountPopover />
        </div>
      </div>
    </header>
  )
}
