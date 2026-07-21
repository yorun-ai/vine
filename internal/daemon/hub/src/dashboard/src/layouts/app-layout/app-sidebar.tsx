import * as React from 'react'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { useNavigate } from '@tanstack/react-router'

import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { useReservedScrollbar } from '@/components/ui/resizable-list-panel'
import { cn } from '@/lib/utils'
import { useLocale } from '@/i18n'

import { getCategorizedScenes } from './nav-matching'
import type { AppNavItem, AppScene } from './nav-config'

const activeStyle =
  'border border-transparent bg-primary/[0.08] text-primary hover:bg-primary/[0.08] hover:text-primary active:bg-primary/[0.1] active:text-primary'
const defaultStyle =
  'border border-transparent text-sidebar-foreground/80 hover:bg-primary/[0.06] hover:text-sidebar-foreground active:bg-primary/[0.08] active:text-sidebar-foreground'
const SIDEBAR_LOGO_SWAP_DELAY_MS = 200
const SIDEBAR_TEXT_COLLAPSE_DELAY_MS = 200
const SIDEBAR_WIDTH_STORAGE_KEY = 'vinehub_sidebar_width'
const SIDEBAR_COLLAPSED_GROUPS_STORAGE_KEY = 'vinehub_sidebar_collapsed_groups'
const SIDEBAR_DEFAULT_WIDTH = 208
const SIDEBAR_MIN_WIDTH = 176
const SIDEBAR_MAX_WIDTH = 360

function clampSidebarWidth(width: number) {
  return Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, width))
}

function getInitialSidebarWidth() {
  if (typeof window === 'undefined') {
    return SIDEBAR_DEFAULT_WIDTH
  }

  const storedWidth = Number(
    window.localStorage.getItem(SIDEBAR_WIDTH_STORAGE_KEY),
  )
  if (!Number.isFinite(storedWidth) || storedWidth <= 0) {
    return SIDEBAR_DEFAULT_WIDTH
  }

  return clampSidebarWidth(storedWidth)
}

function getInitialCollapsedGroups() {
  if (typeof window === 'undefined') {
    return {}
  }

  const storedGroups = window.localStorage.getItem(
    SIDEBAR_COLLAPSED_GROUPS_STORAGE_KEY,
  )
  if (storedGroups === null) {
    return {}
  }

  try {
    const parsedGroups = JSON.parse(storedGroups) as Record<string, unknown>
    return Object.fromEntries(
      Object.entries(parsedGroups).filter(([, collapsed]) => collapsed === true),
    ) as Record<string, true>
  } catch {
    return {}
  }
}

interface AppSidebarProps {
  activeItem: AppNavItem | null
  activeScene: AppScene
  isMobile: boolean
  isMobileOpen: boolean
  isScenePanelOpen: boolean
  sidebarState: 'expanded' | 'collapsed'
  onCloseMobile: () => void
  onScenePanelOpenChange: (open: boolean) => void
}

function SceneToggleIcon({
  className,
  open,
}: {
  className?: string
  open: boolean
}) {
  return open ? (
    <ChevronUp aria-hidden="true" className={className} />
  ) : (
    <ChevronDown aria-hidden="true" className={className} />
  )
}

function shouldUseBrowserNavigation(
  event: React.MouseEvent<HTMLAnchorElement>,
) {
  return (
    event.defaultPrevented ||
    event.button !== 0 ||
    event.metaKey ||
    event.altKey ||
    event.ctrlKey ||
    event.shiftKey
  )
}

export function AppSidebar({
  activeItem,
  activeScene,
  isMobile,
  isMobileOpen,
  isScenePanelOpen,
  sidebarState,
  onCloseMobile,
  onScenePanelOpenChange,
}: AppSidebarProps) {
  const navigate = useNavigate()
  const { locale, t } = useLocale()
  const categorizedScenes = React.useMemo(
    () => getCategorizedScenes(locale),
    [locale],
  )
  const isCollapsed = !isMobile && sidebarState === 'collapsed'
  const [isLogoExpanded, setIsLogoExpanded] = React.useState(!isCollapsed)
  const [isNavCompact, setIsNavCompact] = React.useState(isCollapsed)
  const [sidebarWidth, setSidebarWidth] = React.useState(getInitialSidebarWidth)
  const [isResizing, setIsResizing] = React.useState(false)
  const [collapsedGroups, setCollapsedGroups] = React.useState(
    getInitialCollapsedGroups,
  )
  const handleScrollAreaScroll = useReservedScrollbar()
  const resizeStartRef = React.useRef({
    pointerX: 0,
    width: SIDEBAR_DEFAULT_WIDTH,
  })
  const isCollapsedCompact = isCollapsed && isNavCompact

  const updateSidebarWidth = React.useCallback((nextWidth: number) => {
    const clampedWidth = clampSidebarWidth(nextWidth)
    setSidebarWidth(clampedWidth)
    window.localStorage.setItem(SIDEBAR_WIDTH_STORAGE_KEY, String(clampedWidth))
  }, [])

  const toggleGroupCollapsed = React.useCallback((groupId: string) => {
    setCollapsedGroups((currentGroups) => {
      const nextGroups = { ...currentGroups }
      if (nextGroups[groupId]) {
        delete nextGroups[groupId]
      } else {
        nextGroups[groupId] = true
      }
      window.localStorage.setItem(
        SIDEBAR_COLLAPSED_GROUPS_STORAGE_KEY,
        JSON.stringify(nextGroups),
      )
      return nextGroups
    })
  }, [])

  const handleResizePointerDown = React.useCallback(
    (event: React.PointerEvent<HTMLButtonElement>) => {
      if (isMobile || isCollapsed) {
        return
      }

      event.preventDefault()
      resizeStartRef.current = {
        pointerX: event.clientX,
        width: sidebarWidth,
      }
      setIsResizing(true)
    },
    [isCollapsed, isMobile, sidebarWidth],
  )

  React.useEffect(() => {
    if (!isResizing) {
      return
    }

    const previousCursor = document.body.style.cursor
    const previousUserSelect = document.body.style.userSelect
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'

    function handlePointerMove(event: PointerEvent) {
      const deltaX = event.clientX - resizeStartRef.current.pointerX
      updateSidebarWidth(resizeStartRef.current.width + deltaX)
    }

    function handlePointerUp() {
      setIsResizing(false)
    }

    window.addEventListener('pointermove', handlePointerMove)
    window.addEventListener('pointerup', handlePointerUp)
    window.addEventListener('pointercancel', handlePointerUp)

    return () => {
      document.body.style.cursor = previousCursor
      document.body.style.userSelect = previousUserSelect
      window.removeEventListener('pointermove', handlePointerMove)
      window.removeEventListener('pointerup', handlePointerUp)
      window.removeEventListener('pointercancel', handlePointerUp)
    }
  }, [isResizing, updateSidebarWidth])

  React.useEffect(() => {
    if (!isCollapsed) {
      setIsNavCompact(false)
      return
    }

    const timer = window.setTimeout(() => {
      setIsNavCompact(true)
    }, SIDEBAR_TEXT_COLLAPSE_DELAY_MS)

    return () => {
      window.clearTimeout(timer)
    }
  }, [isCollapsed])

  React.useEffect(() => {
    if (isCollapsed) {
      setIsLogoExpanded(false)
      return
    }

    const timer = window.setTimeout(() => {
      setIsLogoExpanded(true)
    }, SIDEBAR_LOGO_SWAP_DELAY_MS)

    return () => {
      window.clearTimeout(timer)
    }
  }, [isCollapsed])

  const handleNavigate = React.useCallback(
    (to: string) => {
      onScenePanelOpenChange(false)
      onCloseMobile()
      void navigate({ to })
    },
    [navigate, onCloseMobile, onScenePanelOpenChange],
  )

  return (
    <>
      <div
        className={cn(
          'fixed inset-0 z-30 bg-slate-950/28 transition-opacity lg:hidden',
          isMobileOpen ? 'opacity-100' : 'pointer-events-none opacity-0',
        )}
        onClick={onCloseMobile}
      />

      <aside
        className={cn(
          'relative z-40 flex h-dvh shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground after:pointer-events-none after:absolute after:inset-y-0 after:right-0 after:z-10 after:w-3 after:bg-[linear-gradient(to_left,rgba(15,23,42,0.05),rgba(15,23,42,0.02),transparent)] transition-[width,transform] duration-200 ease-linear',
          isResizing && 'transition-none',
          isMobile
            ? cn(
                'fixed inset-y-0 left-0 w-[18rem] shadow-[0_18px_48px_rgba(15,23,42,0.14)] lg:hidden',
                isMobileOpen ? 'translate-x-0' : '-translate-x-full',
              )
            : isCollapsed
              ? 'sticky top-0 w-[3.4rem]'
              : 'sticky top-0',
        )}
        style={
          !isMobile && !isCollapsed ? { width: `${sidebarWidth}px` } : undefined
        }
      >
        <div className="h-14 overflow-hidden bg-sidebar p-0">
          <div
            className={cn(
              'flex h-full items-center',
              isLogoExpanded ? 'w-full px-3' : 'w-[3.4rem] justify-center',
            )}
          >
            {isLogoExpanded ? (
              <img
                src="/brand/vinehub-full.png"
                alt="Vine Hub"
                className="h-8 w-auto shrink-0 object-contain"
              />
            ) : (
              <img
                src="/brand/vinehub.png"
                alt="Vine Hub"
                className="h-8 w-auto shrink-0 object-contain"
              />
            )}
          </div>
        </div>

        <div
          className={cn(
            'relative shrink-0 bg-sidebar px-3 pt-1 pb-1 transition-[padding] duration-200 ease-out',
            isNavCompact && 'px-0',
          )}
        >
          <Popover
            open={isScenePanelOpen}
            onOpenChange={onScenePanelOpenChange}
          >
            <PopoverTrigger
              aria-label={`${t('sidebar.sceneSwitch.current')}${activeScene.label}`}
              className={cn(
                'flex h-14 w-full cursor-pointer items-center justify-between overflow-hidden rounded-[8px] border border-sidebar-border bg-background px-2 text-sidebar-foreground shadow-[0_2px_12px_rgba(15,23,42,0.04)] transition-[width,height,padding,border-radius,background-color,color,border-color] duration-200 ease-out hover:bg-primary/[0.06] hover:text-primary',
                isNavCompact &&
                  'mx-auto flex size-10 items-stretch justify-center p-0',
                isCollapsedCompact &&
                  (isScenePanelOpen
                    ? '!border-[#8fd9d2] !bg-[#eefbf9] !text-[#109c95] hover:!bg-[#e7f8f6]'
                    : '!border-primary/60 !bg-primary/80 !text-primary-foreground hover:!bg-primary/70'),
                isCollapsed &&
                  !isNavCompact &&
                  !isScenePanelOpen &&
                  '!border-primary/20 !bg-primary/[0.08] !text-primary hover:!bg-primary/[0.1]',
              )}
            >
              <span
                className={cn(
                  'flex min-w-0 flex-1 items-center gap-3 overflow-hidden transition-[gap,padding] duration-200 ease-out',
                  isNavCompact &&
                    'size-full min-w-0 flex-none items-stretch justify-center gap-0 pr-0',
                )}
              >
                <span
                  className={cn(
                    'flex size-9 shrink-0 items-center justify-center rounded-[8px] bg-primary/10 text-primary transition-[width,height,border-radius,background-color,color] duration-200 ease-out',
                    isNavCompact &&
                      'size-full rounded-[inherit] bg-transparent',
                    isCollapsedCompact
                      ? isScenePanelOpen
                        ? 'text-[#109c95]'
                        : 'text-primary-foreground'
                      : 'text-primary',
                  )}
                >
                  <activeScene.icon className="size-5 shrink-0" />
                </span>
                <span
                  className={cn(
                    'min-w-0 flex-1 overflow-hidden text-left',
                    isNavCompact && 'hidden',
                  )}
                >
                  <span
                    className={cn(
                      'block truncate whitespace-nowrap text-sm font-semibold leading-5',
                      isCollapsedCompact
                        ? 'text-primary-foreground'
                        : 'text-primary',
                    )}
                  >
                    {activeScene.label}
                  </span>
                </span>
              </span>
              <SceneToggleIcon
                open={isScenePanelOpen}
                className={cn(
                  'ml-1 size-4 shrink-0',
                  isCollapsedCompact
                    ? 'text-primary-foreground'
                    : 'text-primary',
                  isNavCompact && 'hidden',
                )}
              />
            </PopoverTrigger>

            <PopoverContent
              side={isCollapsed ? 'right' : 'bottom'}
              align="start"
              sideOffset={isCollapsed ? 10 : 8}
              className={cn(
                'border border-border/50 p-6 shadow-xl',
                isCollapsed
                  ? 'w-fit max-w-[calc(100vw-6rem)]'
                  : isMobile
                    ? 'w-[calc(100vw-2.75rem)] max-w-[20rem]'
                    : 'w-fit max-w-[calc(100vw-2.75rem)]',
              )}
            >
              <div className="grid grid-cols-1 gap-x-8 gap-y-6 md:grid-flow-col md:auto-cols-[16rem] md:grid-cols-none">
                {categorizedScenes.map((category) => (
                  <section
                    key={category.title}
                    className="flex min-w-0 flex-col gap-4"
                  >
                    <div className="px-1 text-xs font-semibold tracking-wide text-muted-foreground">
                      {category.title}
                    </div>
                    <div className="flex flex-col gap-3">
                      {category.scenes.map((scene) => {
                        const isCurrent = scene.id === activeScene.id

                        return (
                          <a
                            key={scene.id}
                            href={scene.defaultTo}
                            className={cn(
                              'group relative flex w-full items-center gap-3.5 rounded-lg border border-border bg-card px-4 py-3.5 text-left transition-colors duration-200',
                              'cursor-pointer',
                              isCurrent
                                ? 'border-primary/30 bg-primary/[0.06]'
                                : 'hover:bg-primary/[0.06] active:bg-primary/[0.08]',
                            )}
                            onClick={(event) => {
                              if (shouldUseBrowserNavigation(event)) {
                                return
                              }
                              event.preventDefault()
                              handleNavigate(scene.defaultTo)
                            }}
                          >
                            <span
                              className={cn(
                                'flex size-10 shrink-0 items-center justify-center rounded bg-transparent transition-colors',
                                isCurrent
                                  ? 'text-primary'
                                  : 'text-muted-foreground/70',
                              )}
                            >
                              <scene.icon className="size-5 shrink-0" />
                            </span>
                            <span className="min-w-0 flex-1">
                              <span
                                className={cn(
                                  'truncate text-sm font-semibold',
                                  isCurrent
                                    ? 'text-primary'
                                    : 'text-foreground',
                                )}
                              >
                                {scene.label}
                              </span>
                              <span
                                className={cn(
                                  'mt-0.5 block line-clamp-2 text-xs leading-relaxed',
                                  isCurrent
                                    ? 'text-primary/80'
                                    : 'text-muted-foreground',
                                )}
                              >
                                {scene.description}
                              </span>
                            </span>
                          </a>
                        )
                      })}
                    </div>
                  </section>
                ))}
              </div>
            </PopoverContent>
          </Popover>
        </div>

        <div
          className={cn(
            'scrollbar-reserved flex-1 overflow-y-auto bg-sidebar pt-1.5 pb-2',
            isCollapsed && 'pt-1',
          )}
          onScroll={handleScrollAreaScroll}
        >
          <div className="flex flex-col">
            {activeScene.groups.map((group) => {
              const isGroupCollapsed =
                !isCollapsed && collapsedGroups[group.id] === true
              const activeGroupItem = group.items.find(
                (item) => item.id === activeItem?.id,
              )
              const hasActiveGroupItem = activeGroupItem !== undefined
              const shouldHighlightGroupTitle =
                isGroupCollapsed && hasActiveGroupItem

              return (
                <section
                  key={group.id}
                  className={cn(
                    'px-3',
                    group.title ? 'py-0' : 'py-1.5',
                    isCollapsed && 'py-1.5',
                  )}
                >
                  {group.title ? (
                    <button
                      type="button"
                      aria-expanded={!isGroupCollapsed}
                      title={activeGroupItem?.label}
                      className={cn(
                        'my-1 flex h-8 w-full cursor-pointer items-center justify-between gap-2 rounded-md px-2 text-left text-xs font-semibold tracking-[0.08em] text-muted-foreground/70 transition-colors hover:bg-primary/[0.06] hover:text-sidebar-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none',
                        shouldHighlightGroupTitle &&
                          'bg-primary/[0.08] text-primary hover:bg-primary/[0.08] hover:text-primary',
                        isCollapsed &&
                          'my-0 h-0 min-h-0 overflow-hidden px-0 opacity-0',
                      )}
                      onClick={() => toggleGroupCollapsed(group.id)}
                    >
                      <span
                        className={cn('truncate', isCollapsed && 'hidden')}
                      >
                        {group.title}
                      </span>
                      <ChevronDown
                        aria-hidden="true"
                        className={cn(
                          'size-3 shrink-0 transition-transform',
                          isGroupCollapsed && '-rotate-90',
                          isCollapsed && 'hidden',
                        )}
                      />
                    </button>
                  ) : null}

                  <div
                    className={cn(
                      'flex flex-col gap-1',
                      isCollapsed && 'items-center',
                      isGroupCollapsed && 'hidden',
                    )}
                  >
                    {group.items.map((item) => {
                      const isActive = activeItem?.id === item.id

                      return (
                        <a
                          key={item.id}
                          href={item.to}
                          title={item.label}
                          className={cn(
                            isActive ? activeStyle : defaultStyle,
                            'flex h-8 w-full items-center gap-2 rounded-md border px-2 text-left text-sm transition-colors',
                            isCollapsed && 'size-8 justify-center gap-0 p-2',
                          )}
                          onClick={(event) => {
                            if (shouldUseBrowserNavigation(event)) {
                              return
                            }
                            event.preventDefault()
                            handleNavigate(item.to)
                          }}
                        >
                          <item.icon className="size-4 shrink-0" />
                          <span
                            className={cn(
                              'truncate whitespace-nowrap',
                              isCollapsed && 'hidden',
                            )}
                          >
                            {item.label}
                          </span>
                        </a>
                      )
                    })}
                  </div>
                </section>
              )
            })}
          </div>
        </div>

        {!isMobile && !isCollapsed ? (
          <button
            type="button"
            role="separator"
            aria-orientation="vertical"
            aria-label={t('sidebar.resize.label')}
            aria-valuemin={SIDEBAR_MIN_WIDTH}
            aria-valuemax={SIDEBAR_MAX_WIDTH}
            aria-valuenow={sidebarWidth}
            title={t('sidebar.resize.title')}
            className={cn(
              'absolute top-0 right-0 z-20 h-full w-2 cursor-col-resize touch-none outline-none',
              'before:absolute before:top-0 before:right-0 before:h-full before:w-px before:bg-transparent before:transition-colors',
              'hover:before:bg-primary/45 focus-visible:before:bg-primary/70',
              isResizing && 'before:bg-primary/70',
            )}
            onDoubleClick={() => updateSidebarWidth(SIDEBAR_DEFAULT_WIDTH)}
            onKeyDown={(event) => {
              if (event.key === 'ArrowLeft') {
                event.preventDefault()
                updateSidebarWidth(sidebarWidth - 12)
              }
              if (event.key === 'ArrowRight') {
                event.preventDefault()
                updateSidebarWidth(sidebarWidth + 12)
              }
              if (event.key === 'Home') {
                event.preventDefault()
                updateSidebarWidth(SIDEBAR_MIN_WIDTH)
              }
              if (event.key === 'End') {
                event.preventDefault()
                updateSidebarWidth(SIDEBAR_MAX_WIDTH)
              }
              if (event.key === 'Enter') {
                event.preventDefault()
                updateSidebarWidth(SIDEBAR_DEFAULT_WIDTH)
              }
            }}
            onPointerDown={handleResizePointerDown}
          />
        ) : null}
      </aside>
    </>
  )
}
