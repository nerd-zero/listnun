// The design's own line icons, copied from the canvas files
// (~/listnun-redesign/canvas/p0/*.dc.html, the `I` map and nav items).
// Each is a single path on a 16×16 grid, drawn with a 1.4px round stroke.
// PrimeIcons stand-ins drifted from the drawings, so the shell uses these.
export const PATHS = {
  dash: 'M2 2h5v5H2zM9 2h5v3H9zM9 7h5v7H9zM2 9h5v5H2z',
  lists: 'M5.5 4h8.5M5.5 8h8.5M5.5 12h8.5M2.2 4h.1M2.2 8h.1M2.2 12h.1',
  subs: 'M6 7a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5zM1.5 14c0-2.5 2-4 4.5-4s4.5 1.5 4.5 4M11 2.5a2.5 2.5 0 0 1 0 4.5M12.5 10c1.3.5 2 1.8 2 4',
  camp: 'M2 6v4h2l5 3V3L4 6zM12 5.5a3 3 0 0 1 0 5',
  code: 'm5 4-4 4 4 4M11 4l4 4-4 4',
  key: 'M10.5 2a3.5 3.5 0 1 1-2.6 5.8L2 13.5V14h2.5v-1.5H6V11h1.5l1-1M11 5h.1',
  file: 'M3.5 2h6.5l2.5 2.5V14h-9zM10 2v2.5h2.5M6 8h4M6 10.5h4',
  send: 'M14 2 2 7l5 2 2 5zM7 9l3-3',
  log: 'M3 2.5h10v11H3zM5.5 5.5h5M5.5 8h5M5.5 10.5h3',
  image: 'M2 3h12v10H2zM2 11l3.5-3.5 3 3 2-2L14 12M10.5 6h.1',
  webhook: 'M6.5 5.5a2.5 2.5 0 1 1 3.5 2.2L11.5 10M4 9a2.5 2.5 0 1 0 3 3.5h4.5M11.5 8.5a2.5 2.5 0 1 1-.5 5',
  users: 'M8 7a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5zM3 14c0-2.8 2.2-4.5 5-4.5s5 1.7 5 4.5',
  gear: 'M8 10a2 2 0 1 0 0-4 2 2 0 0 0 0 4zM8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.4 3.4l1.4 1.4M11.2 11.2l1.4 1.4M3.4 12.6l1.4-1.4M11.2 4.8l1.4-1.4',
  external: 'M9 2h5v5M14 2 7 9M12 9.5V14H2V4h4.5',
  collapse: 'M2 2.5h12v11H2zM6 2.5v11M10.5 6 8.5 8l2 2',
  expand: 'M2 2.5h12v11H2zM6 2.5v11M9 6l2 2-2 2',
  menu: 'M2.5 4h11M2.5 8h11M2.5 12h11',
  refresh: 'M13.5 8A5.5 5.5 0 1 1 11.8 4M13.5 2v3h-3',
  moon: 'M13.5 9.5A5.5 5.5 0 0 1 6.5 2.5a5.5 5.5 0 1 0 7 7z',
  sun: 'M8 10.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5zM8 1.5V3M8 13v1.5M1.5 8H3M13 8h1.5M3.4 3.4l1 1M11.6 11.6l1 1M3.4 12.6l1-1M11.6 4.4l1-1',
  close: 'm4 4 8 8M12 4l-8 8',
  chevronRight: 'm6 4 4 4-4 4',
  chevronDown: 'm4 6 4 4 4-4',
} as const;

export type LnIconName = keyof typeof PATHS;

export function isLnIcon(name: string): name is LnIconName {
  return Object.prototype.hasOwnProperty.call(PATHS, name);
}
