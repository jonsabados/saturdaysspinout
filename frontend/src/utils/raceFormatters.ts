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