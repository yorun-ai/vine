import type { ComponentType, ReactNode } from 'react'

import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { cn } from '@/lib/utils'

interface ListEmptyStateProps {
  icon: ComponentType<{ className?: string }>
  title: string
  description?: string
  action?: ReactNode
  className?: string
}

/**
 * Shared empty state for the list and detail panels of a list/detail page.
 *
 * An empty panel should say what is missing and offer the single action that
 * fills it, instead of leaving a bare line of muted text in the middle of the
 * pane. Callers pass the same title/description they use for the matching
 * "no results" case, so searching never hides the way back.
 */
export function ListEmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
}: ListEmptyStateProps) {
  return (
    <Empty className={cn('min-h-40 px-4 py-8', className)}>
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Icon />
        </EmptyMedia>
        <EmptyTitle>{title}</EmptyTitle>
        {description ? (
          <EmptyDescription>{description}</EmptyDescription>
        ) : null}
      </EmptyHeader>
      {action ? <EmptyContent>{action}</EmptyContent> : null}
    </Empty>
  )
}
