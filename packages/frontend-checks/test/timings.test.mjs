import assert from 'node:assert/strict'
import { mkdtemp, readFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import process from 'node:process'
import { test } from 'node:test'
import TimingReporter from '../src/timing-reporter.mjs'
import { testPhases, timingSummary } from '../src/timings.mjs'

test('timing phases do not double-count nested hooks or assertions', () => {
  const phases = testPhases(1000, [
    { title: 'Before Hooks', category: 'hook', duration: 200, startTime: new Date(0), steps: [
      { title: 'fixture reset', category: 'pw:api', duration: 150, steps: [] },
    ] },
    { title: 'Expect visible', category: 'expect', duration: 100, steps: [
      { title: 'Locator wait', category: 'pw:api', duration: 95, steps: [] },
    ] },
    { title: 'After Hooks', category: 'hook', duration: 50, startTime: new Date(950), steps: [] },
  ])
  assert.equal(phases.setupMs, 200)
  assert.equal(phases.bodyMs, 750)
  assert.equal(phases.teardownMs, 50)
  assert.equal(phases.wallDurationMs, 1000)
  assert.equal(phases.phaseTimingAvailable, true)
  assert.equal(phases.stepsAvailable, true)
  assert.deepEqual(phases.slowestSteps.map(step => step.durationMs), [
    150,
    100,
    95,
  ])
})

test('empty and skipped cases do not fabricate timings', () => {
  const phases = testPhases(0, [])
  assert.equal(phases.stepsAvailable, false)
  assert.equal(phases.bodyMs, null)
  assert.equal(phases.phaseTimingAvailable, false)
  assert.deepEqual(timingSummary([]), {
    executed: 0,
    skipped: 0,
    durationSamples: 0,
    phaseTimingCases: 0,
    summedCaseMs: 0,
    summedWallMs: 0,
    setupMs: 0,
    bodyMs: 0,
    teardownMs: 0,
    p50Ms: null,
    p95Ms: null,
  })
})

test('beforeAll and afterAll wall time never gets subtracted from the charged test duration', () => {
  const phases = testPhases(55, [
    { title: 'Before Hooks', category: 'hook', duration: 205, startTime: new Date(0), steps: [] },
    { title: 'After Hooks', category: 'hook', duration: 300, startTime: new Date(255), steps: [] },
  ])
  assert.equal(phases.durationMs, 55)
  assert.equal(phases.bodyMs, 50)
  assert.equal(phases.setupMs, 205)
  assert.equal(phases.teardownMs, 300)
  assert.equal(phases.wallDurationMs, 555)
})

test('missing or backwards wall timestamps remain unknown rather than hiding them as zero', () => {
  const phases = testPhases(55, [
    { title: 'Before Hooks', category: 'hook', duration: 205, startTime: new Date(300), steps: [] },
    { title: 'After Hooks', category: 'hook', duration: 0, startTime: new Date(255), steps: [] },
  ])
  assert.equal(phases.bodyMs, null)
  assert.equal(phases.wallDurationMs, null)
  assert.equal(phases.phaseTimingAvailable, false)
})

test('unfinished native steps with duration -1 are unknown, not negative completed phases', () => {
  const phases = testPhases(55, [
    { title: 'Before Hooks', category: 'hook', duration: 205, startTime: new Date(0), steps: [] },
    { title: 'After Hooks', category: 'hook', duration: -1, startTime: new Date(255), steps: [
      { title: 'Unfinished action', category: 'pw:api', duration: -1, steps: [] },
    ] },
  ])
  assert.equal(phases.teardownMs, null)
  assert.equal(phases.wallDurationMs, null)
  assert.equal(phases.phaseTimingAvailable, false)
  assert.deepEqual(phases.slowestSteps, [])
  assert.equal(testPhases(-1, []).durationMs, null)
  const summary = timingSummary([
    { status: 'failed', ...testPhases(-1, []) },
    { status: 'passed', ...testPhases(50, []) },
  ])
  assert.equal(summary.executed, 2)
  assert.equal(summary.durationSamples, 1)
  assert.equal(summary.summedCaseMs, 50)
  assert.equal(summary.p50Ms, 50)
  assert.equal(summary.p95Ms, 50)
})

test('summaries retain failed attempts and distinguish total work from elapsed time', () => {
  const sample = (duration, status) => ({ status, ...testPhases(duration, []) })
  const summary = timingSummary([
    sample(100, 'passed'),
    sample(300, 'failed'),
    sample(999, 'skipped'),
  ])
  assert.equal(summary.executed, 2)
  assert.equal(summary.skipped, 1)
  assert.equal(summary.summedCaseMs, 400)
  assert.equal(summary.p50Ms, 100)
  assert.equal(summary.p95Ms, 300)
})

test('reporter retains failure, project and retry metadata without changing the result', async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), 'test-timings-'))
  const outputFile = path.join(directory, 'nested', 'test-timings.json')
  const reporter = new TimingReporter({ outputFile })
  const project = 'synthetic-browser'
  reporter.onBegin({
    workers: 2,
    fullyParallel: false,
    shard: { current: 1, total: 2 },
    projects: [
      { name: project },
    ],
    version: '1.62.0',
  })
  reporter.onTestEnd({
    location: { file: path.join(process.cwd(), 'fixture.spec.mjs') },
    parent: { project: () => ({ name: project }) },
    titlePath: () => ['fixture', 'example'],
  }, { status: 'failed', retry: 0, duration: 300, steps: [] })
  const result = { status: 'failed', duration: 200 }
  assert.equal(await reporter.onEnd(result), undefined)
  assert.deepEqual(result, { status: 'failed', duration: 200 })
  const report = JSON.parse(await readFile(outputFile, 'utf8'))
  assert.equal(report.status, 'failed')
  assert.equal(report.elapsedMs, 200)
  assert.equal(report.summary.summedCaseMs, 300)
  assert.equal(report.projects[project].executed, 1)
  assert.equal(report.cases[0].retry, 0)
  assert.equal(report.cases[0].file, 'fixture.spec.mjs')
})
