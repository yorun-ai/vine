import type {
  MessageQueueConsumer,
  MessageQueueSubject,
} from '../../skeled/admin/data.ts'

export type QueueSort<Key extends string> = {
  key: Key
  direction: 'asc' | 'desc'
}

export type SubjectSortKey = 'subject' | 'messages'
export type ConsumerSortKey =
  'subject' | 'name' | 'pending' | 'ackPending' | 'redelivered' | 'waiting'

export function nextQueueSort<Key extends string>(
  current: QueueSort<Key>,
  key: Key,
): QueueSort<Key> {
  return {
    key,
    direction:
      current.key === key
        ? current.direction === 'asc'
          ? 'desc'
          : 'asc'
        : key === 'subject' || key === 'name'
          ? 'asc'
          : 'desc',
  }
}

function compareCount(a: string | number, b: string | number) {
  const left = BigInt(a)
  const right = BigInt(b)
  return left < right ? -1 : left > right ? 1 : 0
}

export function sortQueueSubjects(
  rows: MessageQueueSubject[],
  sort: QueueSort<SubjectSortKey>,
) {
  return [...rows].sort((a, b) => {
    const order =
      sort.key === 'subject'
        ? a.subject.localeCompare(b.subject)
        : compareCount(a.messages, b.messages)
    return (
      (sort.direction === 'asc' ? order : -order) ||
      a.subject.localeCompare(b.subject)
    )
  })
}

export function sortQueueConsumers(
  rows: MessageQueueConsumer[],
  sort: QueueSort<ConsumerSortKey>,
) {
  return [...rows].sort((a, b) => {
    const order =
      sort.key === 'subject'
        ? a.filterSubjects.join(', ').localeCompare(b.filterSubjects.join(', '))
        : sort.key === 'name'
          ? a.name.localeCompare(b.name)
          : compareCount(a[sort.key], b[sort.key])
    return (
      (sort.direction === 'asc' ? order : -order) ||
      a.name.localeCompare(b.name)
    )
  })
}
