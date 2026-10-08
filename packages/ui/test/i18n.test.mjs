import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createI18nAdapter, validateI18nResources } from '@ztd-me/ui'

const resources = {
  'en': { common: { retry: 'Retry {{count}}' }, editor: { save: 'Save' } },
  'zh-CN': { common: { retry: '重试 {{count}}' }, editor: { save: '保存' } },
}
test('i18n uses the explicitly injected instance with per-call locale and namespace', () => {
  const calls = []
  const instance = {
    translate: (request) => {
      calls.push(request)
      return request.locale
    },
  }
  const adapter = createI18nAdapter({ instance, resources })
  assert.equal(adapter.instance, instance)
  assert.equal(adapter.t('en', 'common', 'retry', { count: 2 }), 'en')
  assert.equal(adapter.t('zh-CN', 'editor', 'save'), 'zh-CN')
  assert.deepEqual(calls, [
    { locale: 'en', namespace: 'common', key: 'retry', values: { count: 2 } },
    { locale: 'zh-CN', namespace: 'editor', key: 'save', values: {} },
  ])
})
test('coverage rejects missing, extra, blank and mismatched interpolation keys', () => {
  const incomplete = { 'en': resources.en, 'zh-CN': { common: { retry: '重试 {{other}}', extra: '多余' } } }
  assert.deepEqual(validateI18nResources(incomplete).map(issue => issue.reason).sort(), [
    'extra',
    'interpolation',
    'missing',
  ])
  assert.equal(validateI18nResources(null)[0].reason, 'invalid')
  assert.equal(validateI18nResources({ en: resources.en })[0].reason, 'missing')
  assert.ok(validateI18nResources({ ...resources, 'zh-CN': { ...resources['zh-CN'], editor: { save: '' } } })
    .some(issue => issue.reason === 'invalid'))
  assert.throws(() => createI18nAdapter({ resources: incomplete, instance: { translate: () => '' } }), /Incomplete/u)
  assert.deepEqual(validateI18nResources(resources), [])
})
test('separate request adapters do not share an engine or mutate the resources', () => {
  const before = structuredClone(resources)
  const first = createI18nAdapter({ resources, instance: { translate: () => 'first' } })
  const second = createI18nAdapter({ resources, instance: { translate: () => 'second' } })
  assert.equal(first.t('en', 'editor', 'save'), 'first')
  assert.equal(second.t('zh-CN', 'editor', 'save'), 'second')
  assert.deepEqual(resources, before)
})
