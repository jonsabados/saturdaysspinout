/**
 * Convert 0-indexed position from iRacing API to 1-indexed for display
 */
export function toDisplayPosition(position: number): number {
  return position + 1
}

/**
 * Format a lap time from iRacing's 1/10000ths of a second format
 */
export function formatLapTime(timeInCentiseconds: number): string {
  if (timeInCentiseconds <= 0) return '-'
  const totalSeconds = timeInCentiseconds / 10000
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes > 0) {
    return `${minutes}:${seconds.toFixed(3).padStart(6, '0')}`
  }
  return seconds.toFixed(3)
}

/**
 * Format a lap-time delta (in iRacing's 1/10000ths of a second) as a signed
 * seconds value, e.g. -0.234 (faster) or +0.150 (slower).
 */
export function formatLapDelta(delta: number): string {
  // Round to display precision first so tiny sub-millisecond deltas don't
  // produce a signed zero like "-0.000".
  const seconds = Number((delta / 10000).toFixed(3))
  const sign = seconds > 0 ? '+' : seconds < 0 ? '-' : ''
  return `${sign}${Math.abs(seconds).toFixed(3)}`
}

/**
 * Format an interval to the leader
 */
export function formatInterval(
  interval: number,
  isLeader: boolean,
  driverLaps: number,
  leaderLaps: number
): string {
  if (isLeader) return '-'
  if (driverLaps < leaderLaps) return '-' // Laps down
  if (interval < 1) return '-' // < 0.0001s is effectively 0
  return `+${formatLapTime(interval)}`
}