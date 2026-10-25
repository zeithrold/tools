import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { testPhases, timingSummary } from './timings.mjs'

/** Observation only: never changes test status, retries or timeout budgets. */
export default class TimingReporter {
  cases = []
  root = process.cwd()
  configuration = {}

  constructor(options) {
    if (typeof options?.outputFile !== 'string' || options.outputFile.length === 0) {
      throw new TypeError('Timing reporter requires an outputFile')
    }
    this.outputFile = path.resolve(options.outputFile)
  }

  onBegin(config) {
    this.configuration = {
      workers: config.workers,
      fullyParallel: config.fullyParallel,
      shard: config.shard,
      projects: config.projects.map(project => project.name),
      playwrightVersion: config.version,
      nodeVersion: process.version,
      platform: process.platform,
      architecture: process.arch,
    }
  }

  onTestEnd(test, result) {
    this.cases.push({
      file: path.relative(this.root, test.location.file),
      project: test.parent.project()?.name ?? '',
      title: test.titlePath().filter(Boolean).join(' › '),
      status: result.status,
      retry: result.retry,
      repeatEachIndex: test.repeatEachIndex,
      ...testPhases(result.duration, result.steps),
    })
  }

  async onEnd(result) {
    const projects = Array.from(new Set(this.cases.map(item => item.project)))
    await mkdir(path.dirname(this.outputFile), { recursive: true })
    await writeFile(this.outputFile, JSON.stringify({
      schemaVersion: 1,
      status: result.status,
      elapsedMs: result.duration,
      configuration: this.configuration,
      summary: timingSummary(this.cases),
      projects: Object.fromEntries(projects.map(project => [
        project,
        timingSummary(this.cases.filter(item => item.project === project)),
      ])),
      cases: this.cases,
    }, null, 2))
  }
}
