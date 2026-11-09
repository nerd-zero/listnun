// Progress bars that start slower and speed up towards the end are judged
// faster than linear ones, and much faster than ones that slow down
// (Harrison et al., "Rethinking the Progress Bar", UIST 2007). This maps a
// real percentage onto that kind of curve for drawing a bar. Show the real
// numbers in text; this is only for the bar.

// The curve's power. Gentle on purpose: halfway draws at 38%, 90% at 86%,
// so the bar never looks stuck or far from the counts next to it.
const CURVE = 1.4;

// perceivedProgress maps a real 0-100 percentage to the one to draw. 0 and
// 100 are exact, it never runs ahead of the real value, and it always moves
// forward when the real value does.
export function perceivedProgress(pct: number): number {
  const p = Math.min(100, Math.max(0, pct)) / 100;
  return p ** CURVE * 100;
}

export default perceivedProgress;
