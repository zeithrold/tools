import assert from 'node:assert/strict'
import { test } from 'node:test'
import { SiteFooter } from '@ztd-me/ui/client'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

test('footer renders only consumer-supplied identity and arbitrary safe links', () => {
  const html = renderToStaticMarkup(createElement(SiteFooter, {
    copyright: '© Harbor Studio',
    links: [
      { label: 'Source', href: 'https://codeberg.org/example/studio', ariaLabel: 'Studio source' },
      { label: 'Contact', href: 'mailto:support@harbor.example' },
      { label: 'Privacy', href: '/privacy' },
    ],
  }))
  assert.ok(html.includes('© Harbor Studio'))
  assert.ok(html.includes('https://codeberg.org/example/studio'))
  assert.ok(html.includes('aria-label="Studio source"'))
  assert.ok(html.includes('mailto:support@harbor.example'))
  assert.ok(html.includes('href="/privacy"'))
  const empty = renderToStaticMarkup(createElement(SiteFooter))
  assert.ok(!empty.includes('<a'))
  assert.ok(!empty.includes('©'))
})

test('footer rejects executable, malformed and disguised unsafe URLs', () => {
  for (const href of [
    'javascript:alert(1)',
    'data:text/html,unsafe',
    'file:///etc/passwd',
    '//unselected.example',
    '/\\unselected.example',
    'java\nscript:alert(1)',
    '',
  ]) {
    assert.throws(() => renderToStaticMarkup(createElement(SiteFooter, { links: [
      { label: 'Unsafe', href },
    ] })))
  }
})
