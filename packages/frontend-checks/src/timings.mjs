/** Native charged duration excludes all-hooks; phase observations use step wall clocks. */
export function testPhases(durationMs, steps) {
  const before = steps.find(step => step.category === 'hook' && step.title === 'Before Hooks')
  const after = steps.find(step => step.category === 'hook' && step.title === 'After Hooks')
  const setupMs = observedDuration(before?.duration)
  const teardownMs = observedDuration(after?.duration)
  const started = stepStart(before)
  const teardownStarted = stepStart(after)
  const bodyMs = interval(started === null || setupMs === null ? null : started + setupMs, teardownStarted)
  const ended = teardownStarted === null || teardownMs === null ? null : teardownStarted + teardownMs
  const wallDurationMs = interval(started, ended)
  const observations = flatten(steps).filter((step) => {
    return step.category !== 'hook' && observedDuration(step.duration) !== null
  })
  observations.sort((left, right) => right.duration - left.duration)
  return {
    durationMs: observedDuration(durationMs),
    wallDurationMs,
    setupMs,
    bodyMs,
    teardownMs,
    stepsAvailable: steps.length > 0,
    phaseTimingAvailable: bodyMs !== null && wallDurationMs !== null,
    slowestSteps: observations.slice(0, 5).map(step => ({
      title: step.title,
      category: step.category,
      durationMs: step.duration,
    })),
  }
}

function stepStart(step) {
  const value = step?.startTime?.getTime()
  return Number.isFinite(value) ? value : null
}

function observedDuration(value) {
  return Number.isFinite(value) && value >= 0 ? value : null
}

function interval(from, until) {
  if (from === null || until === null || until < from) {
    return null
  }
  return until - from
}

function flatten(steps) {
  return steps.flatMap(step => [
    step,
    ...flatten(step.steps),
  ])
}

export function timingSummary(cases) {
  const executed = cases.filter(item => item.status !== 'skipped')
  const measured = executed.map(item => observedDuration(item.durationMs)).filter(Number.isFinite)
  const durations = measured.toSorted((left, right) => left - right)
  const sum = key => executed.reduce((total, item) => total + (observedDuration(item[key]) ?? 0), 0)
  return {
    executed: executed.length,
    skipped: cases.length - executed.length,
    durationSamples: durations.length,
    phaseTimingCases: executed.filter(item => item.phaseTimingAvailable).length,
    summedCaseMs: sum('durationMs'),
    summedWallMs: sum('wallDurationMs'),
    setupMs: sum('setupMs'),
    bodyMs: sum('bodyMs'),
    teardownMs: sum('teardownMs'),
    p50Ms: durations[Math.max(0, Math.ceil(durations.length * 0.5) - 1)] ?? null,
    p95Ms: durations[Math.max(0, Math.ceil(durations.length * 0.95) - 1)] ?? null,
  }
}
