/** "45 seconds", "1 minute", "15 minutes": how long to wait before retrying. */
export function formatWait(seconds: number): string {
  if (seconds < 60) {
    const s = Math.max(1, Math.ceil(seconds))
    return s === 1 ? '1 second' : `${s} seconds`
  }
  const minutes = Math.ceil(seconds / 60)
  return minutes === 1 ? '1 minute' : `${minutes} minutes`
}
