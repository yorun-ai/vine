import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { SearchInput } from '@/components/ui/search-input'
import { vrpcClient } from '@/config/vrpc-client'
import { useLocale } from '@/i18n'
import { createMessageQueueStatusApiService } from '@/skeled/admin'
import type { MessageQueueStatusView } from '@/skeled/admin'

const service = createMessageQueueStatusApiService(vrpcClient)
const number = (value: string | number) => BigInt(value).toLocaleString()
const cell = 'px-4 py-3 text-left align-top'

function QueueStream({
  item,
  query,
}: {
  item: MessageQueueStatusView
  query: string
}) {
  const { t } = useLocale()
  const keyword = query.trim().toLowerCase()
  const matches = (value: string) => value.toLowerCase().includes(keyword)
  const all = matches(item.kind) || matches(item.stream)
  const subjects = item.subjects.filter((row) => all || matches(row.subject))
  const consumers = item.consumers.filter(
    (row) => all || matches(row.name) || row.filterSubjects.some(matches),
  )

  return (
    <section className="overflow-hidden rounded-xl border">
      <div className="flex flex-wrap items-center gap-3 border-b bg-muted/30 p-4">
        <h2 className="font-semibold">
          {item.kind === 'task' ? 'Task' : 'Event'}
        </h2>
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
        <p className="p-4 text-sm text-muted-foreground">
          {t('queue.missingHelp')}
        </p>
      ) : (
        <>
          <h3 className="px-4 pt-4 text-sm font-semibold">
            {t('queue.subjects')}
          </h3>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className={cell}>
                    Subject
                  </th>
                  <th scope="col" className={cell}>
                    {t('queue.stored')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {subjects.map((row) => (
                  <tr key={row.subject} className="border-t">
                    <td className={`${cell} break-all font-mono text-xs`}>
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
              <p className="px-4 pb-4 text-sm text-muted-foreground">
                {keyword ? t('queue.noMatch') : t('queue.noSubjects')}
              </p>
            )}
          </div>
          <h3 className="border-t px-4 pt-4 text-sm font-semibold">
            {t('queue.consumers')}
          </h3>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className={cell}>
                    Consumer / Subject
                  </th>
                  <th scope="col" className={cell}>
                    {t('queue.pending')}
                  </th>
                  <th scope="col" className={cell}>
                    {t('queue.ackPending')}
                  </th>
                  <th scope="col" className={cell}>
                    {t('queue.redelivered')}
                  </th>
                  <th scope="col" className={cell}>
                    {t('queue.waiting')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {consumers.map((row) => (
                  <tr key={row.name} className="border-t">
                    <td
                      className={`${cell} min-w-64 break-all font-mono text-xs`}
                    >
                      <div>{row.name}</div>
                      <div className="mt-1 text-muted-foreground">
                        {row.filterSubjects.join(', ')}
                      </div>
                    </td>
                    <td className={`${cell} tabular-nums`}>
                      {number(row.pending)}
                    </td>
                    <td className={`${cell} tabular-nums`}>
                      {number(row.ackPending)}
                    </td>
                    <td className={`${cell} tabular-nums`}>
                      {number(row.redelivered)}
                    </td>
                    <td className={`${cell} tabular-nums`}>
                      {number(row.waiting)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!consumers.length && (
              <p className="px-4 pb-4 text-sm text-muted-foreground">
                {keyword ? t('queue.noMatch') : t('queue.noConsumers')}
              </p>
            )}
          </div>
        </>
      )}
    </section>
  )
}

export function MessageQueuePage() {
  const { t } = useLocale()
  const [query, setQuery] = useState('')
  const snapshot = useQuery({
    queryKey: ['message-queues'],
    queryFn: ({ signal }) => service.list(null, { requestInit: { signal }, timeoutMs: 8000 }),
    refetchInterval: 5000,
  })
  return (
    <div className="space-y-5 p-4 md:p-6">
      <div className="flex flex-wrap items-center gap-3">
        <div className="min-w-64 flex-1">
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
          className="rounded-lg border border-destructive/40 p-4 text-sm text-destructive"
        >
          {t('queue.error')}
        </div>
      )}
      {snapshot.isPending && (
        <p role="status" className="text-sm text-muted-foreground">
          {t('queue.loading')}
        </p>
      )}
      {snapshot.dataUpdatedAt > 0 && (
        <p className="text-xs text-muted-foreground">
          {t('queue.updated')}:{' '}
          {new Date(snapshot.dataUpdatedAt).toLocaleTimeString()}
        </p>
      )}
      <div className="space-y-2 text-sm text-muted-foreground">
        <ul className="list-disc space-y-1 pl-5">
          {(['pending', 'ackPending', 'redelivered', 'waiting'] as const).map(
            (metric) => (
              <li key={metric}>
                <span className="font-medium text-foreground">
                  {t(`queue.${metric}`)}
                </span>
                {': '}
                {t(`queue.${metric}Help`)}
              </li>
            ),
          )}
        </ul>
        <p>{t('queue.eventHelp')}</p>
      </div>
      {snapshot.data?.map((item) => (
        <QueueStream key={item.stream} item={item} query={query} />
      ))}
    </div>
  )
}
