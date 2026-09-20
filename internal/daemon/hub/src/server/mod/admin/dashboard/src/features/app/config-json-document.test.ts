import JSON5 from 'json5'
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createConfigJsonDocument, extractConfigJson, getConfigJsonDirtyLines, getConfigJsonTypeLinks, normalizeConfigJson, getFreeConfigJsonErrors, isConfigValueChange } from './config-json-document.ts'

test('descriptions become comments and never enter the saved configuration', () => {
  const value = { scheme: 'http', port: 7099, nested: { hosts: ['a', 'b'] } }
  const document = createConfigJsonDocument(JSON.stringify(value), [
    { name: 'scheme', type: 'string', description: '访问协议\n"http" 或 "https"' },
    { name: 'port', type: 'int', description: '访问端口' },
  ])
  assert.equal(document.doc.includes('// @type int\n  // @desc 访问端口'), true)
  assert.deepEqual(JSON5.parse(document.doc), value)
  assert.deepEqual(JSON.parse(extractConfigJson(document.doc, document.ranges)), value)
})

test('configuration keys resembling comments remain real keys', () => {
  const value = { name: 'a', '// name': 'b', '/// name': 'c', ['__proto__']: null }
  const document = createConfigJsonDocument(JSON.stringify(value), [])
  assert.equal(Object.keys(JSON5.parse(document.doc)).length, Object.keys(value).length)
  assert.deepEqual(JSON.parse(extractConfigJson(document.doc, document.ranges)), value)
})

test('only changes entirely inside one configuration value are allowed', () => {
  const { doc, ranges } = createConfigJsonDocument('{"port":7099,"host":"localhost"}', [])
  const range = ranges[0]
  assert.equal(isConfigValueChange(ranges, range.from, range.to), true)
  assert.equal(isConfigValueChange(ranges, range.from, range.from), true)
  assert.equal(isConfigValueChange(ranges, range.to, range.to), true)
  assert.equal(isConfigValueChange(ranges, 0, doc.length), false)
  assert.equal(isConfigValueChange(ranges, range.from - 1, range.to), false)
  assert.equal(isConfigValueChange(ranges, range.to, ranges[1].from), false)
})

test('invalid or injected value syntax cannot be extracted for saving', () => {
  for (const value of ['', 'tru', '1, "injected": true', '"unterminated']) {
    assert.throws(() => extractConfigJson(value, [{ name: 'port', from: 0, to: value.length }]))
  }
})

test('JSON5 values serialize to standard JSON', () => {
  const value = "{host: 'localhost', ports: [0x50, +443,], /* note */ ratio: .5,}"
  const doc = `{\n  // Configuration\n  "config": ${value}\n}`
  const ranges = [{ name: 'config', from: doc.indexOf(value), to: doc.indexOf(value) + value.length }]
  assert.deepEqual(JSON.parse(extractConfigJson(doc, ranges)), {
    config: { host: 'localhost', ports: [80, 443], ratio: 0.5 },
  })
})

test('non-finite numbers cannot silently become null in saved JSON', () => {
  for (const value of ['NaN', 'Infinity', '-Infinity', '1e999', '{nested: [Infinity]}']) {
    const doc = `{"value": ${value}}`
    assert.throws(() => extractConfigJson(doc, [{ name: 'value', from: 10, to: 10 + value.length }]))
  }
})

test('all description line separators are safely prefixed as comments', () => {
  const document = createConfigJsonDocument('{"port":7099}', [
    { name: 'port', type: 'int', description: 'first\r\nsecond\rthird\nfourth\u2028fifth\u2029last */ "bad": true' },
  ])
  assert.deepEqual(JSON5.parse(document.doc), { port: 7099 })
})

test('a value comment cannot swallow the protected field separator', () => {
  const doc = '{"port": 80 // comment,\n"host": "localhost"}'
  assert.throws(() => extractConfigJson(doc, [{ name: 'port', from: 9, to: 22 }]))
})

test('type links target known definitions only, including nested collection types', () => {
  const document = createConfigJsonDocument('{"categories":[],"enabled":true,"other":null}', [
    { name: 'categories', type: 'map<string, list<booker.BookCategory>>', description: 'booker.BookCategory description stays plain' },
    { name: 'enabled', type: 'bool', description: '' },
    { name: 'other', type: 'booker.Unknown?', description: '' },
  ])
  const definitions = new Map([['booker.BookCategory', { skelName: 'booker.BookCategory' }]])
  const links = getConfigJsonTypeLinks(document, document.ranges, definitions)
  assert.equal(links.length, 1)
  assert.equal(document.doc.slice(links[0].from, links[0].to), 'booker.BookCategory')
  assert.equal(links[0].href, '/skeleton/data/booker.BookCategory')
  assert.equal(links[0].from < document.doc.indexOf('description stays plain'), true)
})

test('schema-free documents preserve raw text without synthetic keys or comments', () => {
  for (const value of ['{"name":"demo"}', '[1,2]', 'null', '"string"', '{unfinished']) {
    const document = createConfigJsonDocument(value, [], false)
    assert.equal(document.doc, value)
    assert.equal(document.lockKeys, false)
    assert.deepEqual(document.typeRanges, [])
  }
})

test('schema-free JSON5 accepts key edits and converts to standard JSON', () => {
  assert.deepEqual(JSON.parse(normalizeConfigJson("{ // config\n renamed: 'value', added: [1,2,], }")), {
    renamed: 'value', added: [1, 2],
  })
  assert.throws(() => normalizeConfigJson('{number: Infinity}'))
})

test('schema-free syntax errors have bounded positions and clear after correction', () => {
  for (const source of ['', '{', '{\n value: "unterminated\n}']) {
    const errors = getFreeConfigJsonErrors(source)
    assert.equal(errors.length, 1)
    assert.equal(errors[0].from >= 0 && errors[0].to <= source.length, true)
  }
  assert.deepEqual(getFreeConfigJsonErrors("{value: 'valid'}"), [])
})

test('dirty highlighting covers type, description and every value line only', () => {
  const document = createConfigJsonDocument('{"items":["a","b"],"other":true}', [
    { name: 'items', type: 'list<string>', description: 'Allowed items' },
    { name: 'other', type: 'bool', description: 'Other' },
  ])
  const lines = getConfigJsonDirtyLines(document.doc, document.ranges, new Set(['items']), new Map())
  const highlighted = lines.map((from) => document.doc.slice(from).split('\n')[0])
  assert.deepEqual(highlighted, [
    '  // @type list<string>', '  // @desc Allowed items', '  "items": [', '    "a",', '    "b"', '  ],',
  ])
  assert.deepEqual(getConfigJsonDirtyLines(document.doc, document.ranges, new Set(), new Map()), [])
  assert.deepEqual(getConfigJsonDirtyLines(document.doc, document.ranges, new Set(['items']), new Map([['items', 'Invalid']])), [])
})

test('unsafe integers block saving including nested and rounded backend values', () => {
  for (const number of ['9007199254740993', '-9007199254740993', '9007199254740992', '1e20']) {
    const raw = `{"nested":{"items":[${number}]}}`
    assert.throws(() => normalizeConfigJson(raw), /safe range/)
    assert.match(getFreeConfigJsonErrors(raw)[0].message, /safe range/)
    assert.equal(createConfigJsonDocument(raw, [], false).doc, raw)
  }
  assert.doesNotThrow(() => normalizeConfigJson('{"max":9007199254740991,"min":-9007199254740991,"fraction":1.5,"text":"9007199254740993"}'))
})

test('source metadata is protected commentary and never part of saved JSON', () => {
  const value = { enabled: true }
  const document = createConfigJsonDocument(JSON.stringify(value), [{
    name: 'enabled', type: 'bool', description: 'Enabled',
    commentTags: ['type', 'desc', 'define', 'override', 'source'],
    sourceComment: '@define domain/user\n@override profile/dev',
  }])
  assert.ok(document.doc.includes('// @define   domain/user'))
  assert.ok(document.doc.includes('// @override profile/dev'))
  const offset = document.doc.indexOf('domain/user')
  assert.equal(isConfigValueChange(document.ranges, offset, offset + 1), false)
  assert.deepEqual(JSON.parse(extractConfigJson(document.doc, document.ranges)), value)
})

test('field documentation renders line comment tags with optional origins', () => {
  const document = createConfigJsonDocument('{"enabled":true}', [{
    name: 'enabled', type: 'bool', description: 'Enable feature',
    commentTags: ['type', 'desc', 'define', 'override', 'source'],
    sourceComment: '@define domain/demo\n@override app/default',
  }])
  assert.equal(document.doc, `{
  // @type     bool
  // @desc     Enable feature
  // @define   domain/demo
  // @override app/default
  "enabled": true
}`)
  assert.deepEqual(JSON.parse(extractConfigJson(document.doc, document.ranges)), { enabled: true })
})

test('comment filters preserve config values and type links follow the visible type tag', () => {
  for (const commentTags of [[], ['desc'], ['type'], ['source'], ['define', 'override']]) {
    const document = createConfigJsonDocument('{"category":"A"}', [{
      name: 'category', type: 'booker.Category', description: 'Category', commentTags,
      sourceComment: '@source app/default\n@define domain/booker\n@override app/default',
    }])
    for (const tag of ['type', 'desc', 'define', 'override', 'source']) {
      assert.equal(document.doc.includes('@' + tag), commentTags.includes(tag))
    }
    const links = getConfigJsonTypeLinks(document, document.ranges, new Map([['booker.Category', {skelName: 'booker.Category'}]]))
    assert.equal(links.length, commentTags.includes('type') ? 1 : 0)
    assert.deepEqual(JSON.parse(extractConfigJson(document.doc, document.ranges)), {category: 'A'})
  }
})

test('source annotations are hidden by default and enabled empty overrides retain a line', () => {
  const field = {
    name: 'enabled', type: 'bool', description: 'Enabled',
    sourceComment: '@source domain/demo\n@define domain/demo\n@override ',
  }
  const defaults = createConfigJsonDocument('{"enabled":true}', [field]).doc
  assert.ok(defaults.includes('@type'))
  assert.ok(defaults.includes('@desc'))
  assert.equal(defaults.includes('@source'), false)
  assert.equal(defaults.includes('@define'), false)
  assert.equal(defaults.includes('@override'), false)
  const explicit = createConfigJsonDocument('{"enabled":true}', [{...field, commentTags: ['override']}])
  assert.match(explicit.doc, /\/\/ @override +\n/)
  assert.deepEqual(JSON.parse(extractConfigJson(explicit.doc, explicit.ranges)), {enabled: true})
})
