import assert from 'node:assert/strict'
import test from 'node:test'

import { startDefaultRequest } from './use-debug-default-request.ts'

function defaults(name: string) {
  return { json: JSON.stringify(name), traceId: name + '-trace', spanId: name + '-span' }
}

function pendingRequest() {
  let resolve!: (request: ReturnType<typeof defaults>) => void
  let reject!: (error: Error) => void
  const promise = new Promise<ReturnType<typeof defaults>>((yes, no) => {
    resolve = yes
    reject = no
  })
  return { promise, resolve, reject }
}

test('a late default cannot replace the current JSON, trace or span', async () => {
  const a = pendingRequest()
  const b = pendingRequest()
  const applied: Array<ReturnType<typeof defaults>> = []
  const apply = (value: ReturnType<typeof defaults>) => applied.push(value)
  const stopA = startDefaultRequest(() => a.promise, apply, () => assert.fail('unexpected error'))
  stopA()
  startDefaultRequest(() => b.promise, apply, () => assert.fail('unexpected error'))

  b.resolve(defaults('B'))
  await b.promise
  a.resolve(defaults('A'))
  await a.promise
  assert.deepEqual(applied, [defaults('B')])
})

test('only the active request can report a failure or finish loading', async () => {
  const first = pendingRequest()
  const reset = pendingRequest()
  const errors: unknown[] = []
  const fail = (error: unknown) => errors.push(error)
  let signal: AbortSignal | undefined
  const stop = startDefaultRequest((s) => {
    signal = s
    return first.promise
  }, () => assert.fail('unexpected defaults'), fail)
  stop()
  assert.equal(signal?.aborted, true)
  startDefaultRequest(() => reset.promise, () => assert.fail('unexpected defaults'), fail)
  first.reject(new Error('superseded'))
  await Promise.resolve()
  assert.deepEqual(errors, [])

  const currentError = new Error('reset failed')
  reset.reject(currentError)
  await Promise.resolve()
  assert.deepEqual(errors, [currentError])
})
