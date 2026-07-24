export const DAY_SHORT = ['Pon', 'Uto', 'Sri', 'Čet', 'Pet', 'Sub', 'Ned']
export const DAY_FULL = ['Ponedjeljak', 'Utorak', 'Srijeda', 'Četvrtak', 'Petak', 'Subota', 'Nedjelja']
export const MONTH_GEN = [
  'siječnja','veljače','ožujka','travnja','svibnja','lipnja',
  'srpnja','kolovoza','rujna','listopada','studenoga','prosinca',
]

export function parseDateLocal(s: string): Date {
  const [y, m, d] = s.split('-').map(Number)
  return new Date(y, m - 1, d)
}

export const PALETTE = ['#3b82f6','#10b981','#f59e0b','#8b5cf6','#ef4444','#06b6d4','#f97316','#ec4899']

export function projectColor(id: string): string {
  let h = 0
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) >>> 0
  return PALETTE[h % PALETTE.length]
}

export function hexToRgb(hex: string) {
  return { r: parseInt(hex.slice(1,3),16), g: parseInt(hex.slice(3,5),16), b: parseInt(hex.slice(5,7),16) }
}

export interface EmpRow { id: string; name: string; role: string; rawRole: string }
