import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowDown, ArrowUp, ArrowUpDown, RefreshCw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { SearchInput } from '@/components/ui/search-input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { vrpcClient } from '@/config/vrpc-client'
import { useLocale } from '@/i18n'
import { createMessageQueueStatusApiService } from '@/skeled/admin'
import type { MessageQueueStatusView } from '@/skeled/admin'
import {
  nextQueueSort,
  sortQueueConsumers,
  sortQueueSubjects,
} from './message-queue-sort'
import type {
  ConsumerSortKey,
  QueueSort,
  SubjectSortKey,
} from './message-queue-sort'

const service = createMessageQueueStatusApiService(vrpcClient)
const number = (value: string | number) => BigInt(value).toLocaleString()
const cell = 'px-4 py-3 text-left align-top'
const metrics = ['pending', 'ackPending', 'redelivered', 'waiting'] as const

function SortHeader<Key extends string>({
  label,
  column,
  sort,
  onSort,
}: {
  label: string
  column: Key
  sort: QueueSort<Key>
  onSort: (key: Key) => void
}) {
  const active = sort.key === column
  const Icon = active
    ? sort.direction === 'asc'
      ? ArrowUp
      : ArrowDown
    : ArrowUpDown
  return (
    <th
      scope="col"
      className="sticky top-0 z-10 bg-background text-left"
      aria-sort={
        active
          ? sort.direction === 'asc'
            ? 'ascending'
            : 'descending'
          : 'none'
      }
    >
      <button
        type="button"
        onClick={() => onSort(column)}
        className="flex w-full cursor-pointer items-center gap-2 whitespace-nowrap px-4 py-3 text-left hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
      >
        {label}
        <Icon aria-hidden="true" className="size-3.5 shrink-0" />
      </button>
    </th>
  )
}

function QueueStream({
  item,
  query,
}: {
  item: MessageQueueStatusView
  query: string
}) {
  const { t } = useLocale()
  const [subjectSort, setSubjectSort] = useState<QueueSort<SubjectSortKey>>({
    key: 'subject',
    direction: 'asc',
  })
  const [consumerSort, setConsumerSort] = useState<QueueSort<ConsumerSortKey>>({
    key: 'subject',
    direction: 'asc',
  })
  const keyword = query.trim().toLowerCase()
  const { subjects, consumers } = useMemo(() => {
    const matches = (value: string) => value.toLowerCase().includes(keyword)
    return {
      subjects: sortQueueSubjects(
        item.subjects.filter((row) => matches(row.subject)),
        subjectSort,
      ),
      consumers: sortQueueConsumers(
        item.consumers.filter(
          (row) => matches(row.name) || row.filterSubjects.some(matches),
        ),
        consumerSort,
      ),
    }
  }, [item, keyword, subjectSort, consumerSort])
  const sortSubjects = (key: SubjectSortKey) =>
    setSubjectSort((current) => nextQueueSort(current, key))
  const sortConsumers = (key: ConsumerSortKey) =>
    setConsumerSort((current) => nextQueueSort(current, key))

  return (
    <section className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border">
      <div className="flex shrink-0 flex-wrap items-center gap-3 border-b bg-muted/30 p-3">
        <span className="font-mono text-xs text-muted-foreground">
          {item.stream}
        </span>
        {item.exists ? (
          <div className="ml-auto flex flex-wrap gap-2">
            <Badge variant="secondary">
              {t('queue.stored')}: {number(item.messages)}
            </Badge>
            <Badge variant="outline">{number(item.bytes)} B</Badge>
            <Badge variant="outline">
              {t('queue.consumers')}: {number(item.consumers.length)}
            </Badge>
          </div>
        ) : (
          <Badge variant="outline">{t('queue.missing')}</Badge>
        )}
      </div>
      {!item.exists ? (
        <p className="overflow-auto p-4 text-sm text-muted-foreground">
          {t('queue.missingHelp')}
        </p>
      ) : (
        <Tabs
          defaultValue="subjects"
          className="min-h-0 flex-1 gap-0 overflow-hidden"
        >
          <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b px-3 py-2">
            <TabsList aria-label={t('queue.views')}>
              <TabsTrigger value="subjects">{t('queue.subjects')}</TabsTrigger>
              <TabsTrigger value="consumers">
                {t('queue.consumers')}
              </TabsTrigger>
            </TabsList>
            <span className="text-xs text-muted-foreground">
              {t('queue.sortHint')}
            </span>
          </div>
          <TabsContent
            value="subjects"
            className="min-h-0 overflow-auto"
            tabIndex={0}
          >
            <table className="w-full text-sm">
              <thead className="text-xs text-muted-foreground">
                <tr>
                  <SortHeader
                    label="Subject"
                    column="subject"
                    sort={subjectSort}
                    onSort={sortSubjects}
                  />
                  <SortHeader
                    label={t('queue.stored')}
                    column="messages"
                    sort={subjectSort}
                    onSort={sortSubjects}
                  />
                </tr>
              </thead>
              <tbody>
                {subjects.map((row) => (
                  <tr key={row.subject} className="border-t">
                    <td
                      className={`${cell} min-w-64 break-all font-mono text-xs`}
                    >
                      {row.subject}
                    </td>
                    <td className={`${cell} tabular-nums`}>
                      {number(row.messages)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!subjects.length && (
              <p className="p-4 text-sm text-muted-foreground">
                {keyword ? t('queue.noMatch') : t('queue.noSubjects')}
              </p>
            )}
          </TabsContent>
          <TabsContent
            value="consumers"
            className="flex min-h-0 flex-col overflow-hidden"
          >
            {item.kind === 'event' && (
              <p className="max-h-20 shrink-0 overflow-auto border-b bg-muted/20 px-4 py-2 text-xs text-muted-foreground">
                {t('queue.eventHelp')}
              </p>
            )}
            <div
              className="min-h-0 flex-1 overflow-auto"
              tabIndex={0}
              role="region"
              aria-label={t('queue.consumers')}
            >
              <table className="w-full text-sm">
                <thead className="text-xs text-muted-foreground">
                  <tr>
                    <SortHeader
                      label="Subject"
                      column="subject"
                      sort={consumerSort}
                      onSort={sortConsumers}
                    />
                    <SortHeader
                      label="Consumer"
                      column="name"
                      sort={consumerSort}
                      onSort={sortConsumers}
                    />
                    {metrics.map((metric) => (
                      <SortHeader
                        key={metric}
                        label={t(`queue.${metric}`)}
                        column={metric}
                        sort={consumerSort}
                        onSort={sortConsumers}
                      />
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {consumers.map((row) => (
                    <tr key={row.name} className="border-t">
                      <td
                        className={`${cell} min-w-64 break-all font-mono text-xs`}
                      >
                        {row.filterSubjects.map((subject) => (
                          <div key={subject}>{subject}</div>
                        ))}
                      </td>
                      <td
                        className={`${cell} min-w-52 break-all font-mono text-xs text-muted-foreground`}
                      >
                        {row.name}
                      </td>
                      {metrics.map((metric) => (
                        <td key={metric} className={`${cell} tabular-nums`}>
                          {number(row[metric])}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
              {!consumers.length && (
                <p className="p-4 text-sm text-muted-foreground">
                  {keyword ? t('queue.noMatch') : t('queue.noConsumers')}
                </p>
              )}
            </div>
          </TabsContent>
        </Tabs>
      )}
    </section>
  )
}

export function MessageQueuePage({ kind }: { kind: 'task' | 'event' }) {
  const { t } = useLocale()
  const [query, setQuery] = useState('')
  const snapshot = useQuery({
    queryKey: ['message-queues'],
    queryFn: ({ signal }) =>
      service.list(null, { requestInit: { signal }, timeoutMs: 8000 }),
    refetchInterval: 5000,
  })
  const item = snapshot.data?.find((stream) => stream.kind === kind)
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden p-4 md:p-6">
      <div className="flex shrink-0 flex-wrap items-center gap-3">
        <div className="min-w-48 flex-1">
          <SearchInput
            value={query}
            onValueChange={setQuery}
            placeholder={t('queue.search')}
          />
        </div>
        <span className="text-xs text-muted-foreground">
          {t('queue.refreshHint')}
        </span>
        <Button
          variant="outline"
          disabled={snapshot.isFetching}
          onClick={() => void snapshot.refetch()}
        >
          <RefreshCw className={snapshot.isFetching ? 'animate-spin' : ''} />
          {t('action.refresh')}
        </Button>
      </div>
      {snapshot.isError && (
        <div
          role="alert"
          className="max-h-24 shrink-0 overflow-auto rounded-lg border border-destructive/40 p-3 text-sm text-destructive"
        >
          {t('queue.error')}
        </div>
      )}
      {snapshot.isPending && (
        <p role="status" className="text-sm text-muted-foreground">
          {t('queue.loading')}
        </p>
      )}
      <div className="flex shrink-0 items-start justify-between gap-3 text-xs text-muted-foreground">
        <details className="min-w-0 flex-1">
          <summary className="w-fit cursor-pointer hover:text-foreground">
            {t('queue.metricHelp')}
          </summary>
          <ul className="mt-2 max-h-28 list-disc space-y-1 overflow-auto pl-5 text-sm">
            {metrics.map((metric) => (
              <li key={metric}>
                <span className="font-medium text-foreground">
                  {t(`queue.${metric}`)}
                </span>
                {': '}
                {t(`queue.${metric}Help`)}
              </li>
            ))}
          </ul>
        </details>
        {snapshot.dataUpdatedAt > 0 && (
          <p className="shrink-0">
            {t('queue.updated')}:{' '}
            {new Date(snapshot.dataUpdatedAt).toLocaleTimeString()}
          </p>
        )}
      </div>
      {item && <QueueStream key={kind} item={item} query={query} />}
    </div>
  )
}
