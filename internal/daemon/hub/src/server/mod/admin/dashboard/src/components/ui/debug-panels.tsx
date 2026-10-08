import * as React from 'react'
import { useLocale } from '@/i18n'

/** Adjustable request/result panels with independently scrollable content. */
export function DebugPanels({ children }: { children: React.ReactNode }) {
  const { t } = useLocale()
  const container = React.useRef<HTMLDivElement>(null)
  const [percent, setPercent] = React.useState(62)
  const [dragging, setDragging] = React.useState(false)
  const panels = React.Children.toArray(children)
  const clamp = (value: number) => Math.min(80, Math.max(30, value))
  return (
    <main ref={container} className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="flex min-h-0 flex-col overflow-hidden [&>section]:flex-1" style={{ height: `${percent}%` }}>
        {panels[0]}
      </div>
      <div
        role="separator"
        tabIndex={0}
        aria-label={t('debug.resizePanels')}
        aria-orientation="horizontal"
        aria-valuenow={Math.round(percent)}
        aria-valuemin={30}
        aria-valuemax={80}
        className={`group flex h-2 shrink-0 cursor-row-resize touch-none items-center justify-center bg-muted/40 outline-none hover:bg-accent focus-visible:bg-accent ${dragging ? 'bg-accent' : ''}`}
        onDoubleClick={() => setPercent(62)}
        onKeyDown={(event) => {
          if (event.key === 'ArrowUp' || event.key === 'ArrowDown') {
            event.preventDefault()
            setPercent((current) => clamp(current + (event.key === 'ArrowUp' ? -5 : 5)))
          } else if (event.key === 'Home' || event.key === 'End') {
            event.preventDefault()
            setPercent(event.key === 'Home' ? 30 : 80)
          }
        }}
        onPointerDown={(event) => {
          event.currentTarget.setPointerCapture(event.pointerId)
          setDragging(true)
        }}
        onPointerMove={(event) => {
          if (!dragging || !container.current) return
          const bounds = container.current.getBoundingClientRect()
          setPercent(clamp((event.clientY - bounds.top) / bounds.height * 100))
        }}
        onPointerUp={() => setDragging(false)}
        onLostPointerCapture={() => setDragging(false)}
      >
        <span className="h-0.5 w-8 rounded-full bg-border group-hover:bg-primary/40 group-focus-visible:bg-primary" />
      </div>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden [&>section]:flex-1">
        {panels[1]}
      </div>
    </main>
  )
}
