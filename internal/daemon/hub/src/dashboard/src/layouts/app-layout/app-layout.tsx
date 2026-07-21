import * as React from 'react'
import { useLocation } from '@tanstack/react-router'

import { AppSidebar } from './app-sidebar'
import { AppTopbar } from './app-topbar'
import { useLocale } from '@/i18n'
import {
  getActiveNavItem,
  getActiveScene,
  getBreadcrumbItems,
} from './nav-matching'

const SIDEBAR_COOKIE_NAME = 'sidebar_state'
const SIDEBAR_COOKIE_MAX_AGE = 60 * 60 * 24 * 7
const MOBILE_MEDIA_QUERY = '(max-width: 1023px)'

function getCookieSidebarState() {
  if (typeof document === 'undefined') {
    return true
  }

  const cookieValue = document.cookie
    .split('; ')
    .find((row) => row.startsWith(`${SIDEBAR_COOKIE_NAME}=`))
    ?.split('=')[1]

  return cookieValue !== 'false'
}

function getInitialIsMobile() {
  if (typeof window === 'undefined') {
    return false
  }

  return window.matchMedia(MOBILE_MEDIA_QUERY).matches
}

export function AppLayout({ children }: { children: React.ReactNode }) {
  const pathname = useLocation({
    select: (location) => location.pathname,
  })
  const [isMobile, setIsMobile] = React.useState(getInitialIsMobile)
  const [isSidebarOpen, setIsSidebarOpen] = React.useState(
    getCookieSidebarState,
  )
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = React.useState(false)
  const [isScenePanelOpen, setIsScenePanelOpen] = React.useState(false)
  const { locale } = useLocale()

  React.useEffect(() => {
    const mediaQuery = window.matchMedia(MOBILE_MEDIA_QUERY)

    function handleMediaQueryChange(event: MediaQueryListEvent) {
      setIsMobile(event.matches)
    }

    setIsMobile(mediaQuery.matches)
    mediaQuery.addEventListener('change', handleMediaQueryChange)

    return () => {
      mediaQuery.removeEventListener('change', handleMediaQueryChange)
    }
  }, [])

  React.useEffect(() => {
    setIsMobileSidebarOpen(false)
    setIsScenePanelOpen(false)
  }, [pathname])

  React.useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'b' && (event.metaKey || event.ctrlKey)) {
        event.preventDefault()
        if (window.matchMedia(MOBILE_MEDIA_QUERY).matches) {
          setIsMobileSidebarOpen((current) => !current)
          return
        }

        setIsSidebarOpen((current) => {
          const next = !current
          document.cookie = `${SIDEBAR_COOKIE_NAME}=${next}; path=/; max-age=${SIDEBAR_COOKIE_MAX_AGE}`
          return next
        })
      }
    }

    window.addEventListener('keydown', handleKeyDown)

    return () => {
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [])

  const toggleSidebar = React.useCallback(() => {
    if (isMobile) {
      setIsMobileSidebarOpen((current) => !current)
      return
    }

    setIsSidebarOpen((current) => {
      const next = !current
      document.cookie = `${SIDEBAR_COOKIE_NAME}=${next}; path=/; max-age=${SIDEBAR_COOKIE_MAX_AGE}`
      return next
    })
  }, [isMobile])

  const sidebarState = isSidebarOpen ? 'expanded' : 'collapsed'
  const activeScene = React.useMemo(
    () => getActiveScene(pathname, locale),
    [locale, pathname],
  )
  const activeItem = React.useMemo(
    () => getActiveNavItem(pathname, locale),
    [locale, pathname],
  )
  const breadcrumbItems = React.useMemo(
    () => getBreadcrumbItems(pathname, sidebarState, locale),
    [locale, pathname, sidebarState],
  )

  return (
    <div className="h-dvh overflow-hidden bg-white text-[#0f172a]">
      <div className="flex h-full min-h-0 w-full">
        <AppSidebar
          activeItem={activeItem}
          activeScene={activeScene}
          isMobile={isMobile}
          isMobileOpen={isMobileSidebarOpen}
          isScenePanelOpen={isScenePanelOpen}
          sidebarState={sidebarState}
          onCloseMobile={() => setIsMobileSidebarOpen(false)}
          onScenePanelOpenChange={setIsScenePanelOpen}
        />

        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
          <AppTopbar
            activeItem={activeItem}
            activeScene={activeScene}
            breadcrumbItems={breadcrumbItems}
            isMobile={isMobile}
            sidebarState={sidebarState}
            onToggleSidebar={toggleSidebar}
          />

          <main className="flex min-h-0 flex-1 flex-col overflow-auto bg-white">
            {children}
          </main>
        </div>
      </div>
    </div>
  )
}
