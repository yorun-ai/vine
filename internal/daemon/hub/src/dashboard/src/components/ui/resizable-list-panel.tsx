import * as React from 'react'

import { cn } from '@/lib/utils'

const DEFAULT_MIN_WIDTH = 260
const DEFAULT_MAX_WIDTH = 520
const DEFAULT_STEP = 16
const SCROLLBAR_HIDE_DELAY_MS = 900

interface UseResizableListPanelOptions {
  storageKey: string
  defaultWidth: number
  minWidth?: number
  maxWidth?: number
}

export interface ResizableListPanelState {
  gridStyle: React.CSSProperties
  isResizing: boolean
  maxWidth: number
  minWidth: number
  resizeTo: (width: number) => void
  startResize: (event: React.PointerEvent<HTMLButtonElement>) => void
  width: number
}

export function useReservedScrollbar() {
  const scrollHideTimers = React.useRef(new WeakMap<Element, number>())

  return React.useCallback((event: React.UIEvent<HTMLElement>) => {
    const target = event.currentTarget
    target.dataset.scrolling = 'true'

    const currentTimer = scrollHideTimers.current.get(target)
    if (currentTimer !== undefined) {
      window.clearTimeout(currentTimer)
    }

    const nextTimer = window.setTimeout(() => {
      delete target.dataset.scrolling
      scrollHideTimers.current.delete(target)
    }, SCROLLBAR_HIDE_DELAY_MS)

    scrollHideTimers.current.set(target, nextTimer)
  }, [])
}

function clampWidth(width: number, minWidth: number, maxWidth: number) {
  return Math.min(maxWidth, Math.max(minWidth, width))
}

function initialWidth({
  defaultWidth,
  maxWidth,
  minWidth,
  storageKey,
}: Required<UseResizableListPanelOptions>) {
  if (typeof window === 'undefined') {
    return defaultWidth
  }

  const storedWidth = Number(window.localStorage.getItem(storageKey))
  if (!Number.isFinite(storedWidth) || storedWidth <= 0) {
    return defaultWidth
  }

  return clampWidth(storedWidth, minWidth, maxWidth)
}

export function useResizableListPanel({
  defaultWidth,
  maxWidth = DEFAULT_MAX_WIDTH,
  minWidth = DEFAULT_MIN_WIDTH,
  storageKey,
}: UseResizableListPanelOptions): ResizableListPanelState {
  const options = React.useMemo(
    () => ({ defaultWidth, maxWidth, minWidth, storageKey }),
    [defaultWidth, maxWidth, minWidth, storageKey],
  )
  const [width, setWidth] = React.useState(() => initialWidth(options))
  const [isResizing, setIsResizing] = React.useState(false)
  const resizeStartRef = React.useRef({ pointerX: 0, width: defaultWidth })

  const resizeTo = React.useCallback(
    (nextWidth: number) => {
      const clampedWidth = clampWidth(nextWidth, minWidth, maxWidth)
      setWidth(clampedWidth)
      window.localStorage.setItem(storageKey, String(clampedWidth))
    },
    [maxWidth, minWidth, storageKey],
  )

  const startResize = React.useCallback(
    (event: React.PointerEvent<HTMLButtonElement>) => {
      event.preventDefault()
      resizeStartRef.current = {
        pointerX: event.clientX,
        width,
      }
      setIsResizing(true)
    },
    [width],
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
      resizeTo(resizeStartRef.current.width + deltaX)
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
  }, [isResizing, resizeTo])

  return {
    gridStyle: {
      '--list-panel-width': `${width}px`,
    } as React.CSSProperties,
    isResizing,
    maxWidth,
    minWidth,
    resizeTo,
    startResize,
    width,
  }
}

export function ResizableListHandle({
  className,
  defaultWidth,
  label = 'Resize list',
  panel,
  step = DEFAULT_STEP,
  title = 'Drag to resize the list. Double-click to reset.',
}: {
  className?: string
  defaultWidth: number
  label?: string
  panel: ResizableListPanelState
  step?: number
  title?: string
}) {
  return (
    <button
      type="button"
      role="separator"
      aria-orientation="vertical"
      aria-label={label}
      aria-valuemin={panel.minWidth}
      aria-valuemax={panel.maxWidth}
      aria-valuenow={panel.width}
      title={title}
      className={cn(
        'absolute top-0 right-0 z-10 hidden h-full w-2 cursor-col-resize touch-none outline-none lg:block',
        'before:absolute before:top-0 before:right-0 before:h-full before:w-px before:bg-transparent before:transition-colors',
        'hover:before:bg-primary/45 focus-visible:before:bg-primary/70',
        panel.isResizing && 'before:bg-primary/70',
        className,
      )}
      onDoubleClick={() => panel.resizeTo(defaultWidth)}
      onKeyDown={(event) => {
        if (event.key === 'ArrowLeft') {
          event.preventDefault()
          panel.resizeTo(panel.width - step)
        }
        if (event.key === 'ArrowRight') {
          event.preventDefault()
          panel.resizeTo(panel.width + step)
        }
        if (event.key === 'Home') {
          event.preventDefault()
          panel.resizeTo(panel.minWidth)
        }
        if (event.key === 'End') {
          event.preventDefault()
          panel.resizeTo(panel.maxWidth)
        }
        if (event.key === 'Enter') {
          event.preventDefault()
          panel.resizeTo(defaultWidth)
        }
      }}
      onPointerDown={panel.startResize}
    />
  )
}
