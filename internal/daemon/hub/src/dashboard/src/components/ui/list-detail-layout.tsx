import type { ReactNode } from 'react'

import {
  ResizableListHandle,
  useReservedScrollbar,
  useResizableListPanel,
} from './resizable-list-panel'

interface ListDetailLayoutProps {
  defaultWidth: number
  resizeLabel: string
  listHeader: ReactNode
  listFooter: ReactNode
  list: ReactNode
  children: ReactNode
}

export function ListDetailLayout({
  defaultWidth,
  resizeLabel,
  listHeader,
  listFooter,
  list,
  children,
}: ListDetailLayoutProps) {
  const panel = useResizableListPanel({ defaultWidth })
  const handleListScroll = useReservedScrollbar()

  return (
    <section className="flex h-[calc(100dvh-3.5rem)] flex-col overflow-hidden bg-white">
      <div
        className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[var(--list-panel-width)_minmax(0,1fr)]"
        style={panel.gridStyle}
      >
        <aside className="relative flex min-h-0 flex-col border-b border-border/70 lg:border-r lg:border-b-0">
          <div className="border-b border-border/70 p-4">{listHeader}</div>
          <div
            className="scrollbar-reserved min-h-0 flex-1 overflow-auto py-2 pr-1 pl-2"
            onScroll={handleListScroll}
          >
            {list}
          </div>
          <ListDetailFooter>{listFooter}</ListDetailFooter>
          <ResizableListHandle
            defaultWidth={defaultWidth}
            label={resizeLabel}
            panel={panel}
          />
        </aside>
        <main className="min-h-0 overflow-hidden">{children}</main>
      </div>
    </section>
  )
}

export function ListDetailFooter({ children }: { children: ReactNode }) {
  return (
    <div className="shrink-0 border-t border-border/70 bg-background px-4 py-3 text-xs text-muted-foreground">
      {children}
    </div>
  )
}
