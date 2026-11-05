export const MO = ['jan', 'fev', 'mar', 'abr', 'mai', 'jun', 'jul', 'ago', 'set', 'out', 'nov', 'dez'];
export const MOL = ['janeiro', 'fevereiro', 'março', 'abril', 'maio', 'junho', 'julho', 'agosto', 'setembro', 'outubro', 'novembro', 'dezembro'];
export const WD = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'];
export const WDL = ['Domingo', 'Segunda', 'Terça', 'Quarta', 'Quinta', 'Sexta', 'Sábado'];
export const TZ = 'Africa/Maputo';

export function fmt(n: number, d = 0): string {
  const [i, f] = (Math.round(n * 100) / 100).toFixed(d).split('.');
  return i.replace(/\B(?=(\d{3})+(?!\d))/g, '.') + (f ? ',' + f : '');
}

export const mt = (cents: number, d = 0) => fmt(cents / 100, d) + ' MT';
export const mtN = (cents: number, d = 0) => fmt(cents / 100, d);
export const pct = (bps: number) => String(bps / 100).replace('.', ',') + '%';

export const fmtPhone = (d: string) => (d.length <= 2 ? d : [d.slice(0, 2), d.slice(2, 5), d.slice(5, 9)].filter(Boolean).join(' '));
export const digits = (v: string, max: number) => v.replace(/\D/g, '').replace(/^258(?=\d{9})/, '').slice(0, max);

export function walletOf(d: string): 'mpesa' | 'emola' | 'tmcel' | null {
  const p = d.slice(0, 2);
  return ['84', '85'].includes(p) ? 'mpesa' : ['86', '87'].includes(p) ? 'emola' : ['82', '83'].includes(p) ? 'tmcel' : null;
}

function parts(iso: string | Date) {
  const dt = typeof iso === 'string' ? new Date(iso) : iso;
  const p = new Intl.DateTimeFormat('en-GB', { timeZone: TZ, year: 'numeric', month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit', weekday: 'short', hour12: false }).formatToParts(dt);
  const g = (t: string) => p.find(x => x.type === t)?.value || '';
  const wd = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].indexOf(g('weekday'));
  return { y: +g('year'), m: +g('month') - 1, d: +g('day'), hh: g('hour'), mm: g('minute'), wd };
}

export const dayMonth = (iso: string) => { const p = parts(iso); return `${String(p.d).padStart(2, '0')} ${MO[p.m]}`; };
export const dateTime = (iso: string) => { const p = parts(iso); return `${String(p.d).padStart(2, '0')} ${MO[p.m]} ${p.y}, ${p.hh}:${p.mm}`; };
export const shortDateTime = (iso: string) => { const p = parts(iso); return `${String(p.d).padStart(2, '0')} ${MO[p.m]}, ${p.hh}:${p.mm}`; };
export const longDate = (iso: string) => { const p = parts(iso); return `${String(p.d).padStart(2, '0')} ${MO[p.m]} ${p.y}`; };
export const todayLabel = () => { const p = parts(new Date()); return `${WDL[p.wd]}, ${p.d} de ${MOL[p.m]}`; };
export const greeting = () => { const h = +parts(new Date()).hh; return h < 12 ? 'Bom dia' : h < 19 ? 'Boa tarde' : 'Boa noite'; };
export const bucketLabel = (iso: string, month: boolean, short: boolean) => {
  const p = parts(iso);
  if (month) return short ? MO[p.m] : `${MOL[p.m]} ${p.y}`;
  return short ? `${p.d} ${MO[p.m]}` : `${WD[p.wd]}, ${p.d} ${MO[p.m]}`;
};
export const weekdayOf = (iso: string) => parts(iso).wd;
export const monthOf = (iso: string) => parts(iso).m;

export function relative(iso: string): string {
  const s = (Date.now() - new Date(iso).getTime()) / 1000;
  if (s < 60) return 'agora';
  if (s < 3600) return `há ${Math.floor(s / 60)} min`;
  if (s < 86400) return `há ${Math.floor(s / 3600)} h`;
  if (s < 172800) return 'ontem';
  return dayMonth(iso);
}

export function ini(n: string): string {
  n = (n || '').trim();
  if (!n) return '?';
  if (n.startsWith('@')) return n.slice(1, 3).toUpperCase();
  const w = n.split(/\s+/).filter(x => x.length > 2 || /^[A-ZÀ-Ý]/.test(x));
  return (w.length > 1 ? w.map(x => x[0]).slice(0, 2).join('') : (w[0] || n).slice(0, 2)).toUpperCase();
}

export const AV: [string, string][] = [['#1A1410', '#FFFFFF'], ['#D14400', '#FFFFFF'], ['#D9D1C4', '#1A1410'], ['#FBD5BF', '#7A2E0A'], ['#57524B', '#FFFFFF']];
export const SEG: [string, string][] = [['#D14400', '#FFFFFF'], ['#1A1410', '#FFFFFF'], ['#D9D1C4', '#1A1410'], ['#8C857B', '#FFFFFF'], ['#FBD5BF', '#7A2E0A'], ['#57524B', '#FFFFFF']];
export const avOf = (key: string) => { let h = 0; for (const c of key) h = (h * 31 + c.charCodeAt(0)) >>> 0; return AV[h % AV.length]; };

export const CH = {
  mpesa: { name: 'M-Pesa', op: 'Vodacom', bg: '#FDE7E7', fg: '#B80000', bar: '#E60000' },
  emola: { name: 'e-Mola', op: 'Movitel', bg: '#FFF0E0', fg: '#B45F00', bar: '#F7871F' },
  card: { name: 'Cartão', op: 'Visa · Mastercard', bg: '#ECE6DB', fg: '#1A1410', bar: '#CFC6B8' },
} as const;

export const SUP_ST = {
  confirmed: ['Confirmado', '#E3F3E8', '#17803D'], pending: ['Pendente', '#FBEFD9', '#9A5B00'],
  failed: ['Falhou', '#FDE8E4', '#B42318'], expired: ['Expirou', '#FDE8E4', '#B42318'],
} as const;

export const WD_ST = {
  requested: ['Pedido', '#ECE6DB', '#57524B'], approved: ['Aprovado', '#FBEFD9', '#9A5B00'],
  paid: ['Pago', '#E3F3E8', '#17803D'], rejected: ['Rejeitado', '#FDE8E4', '#B42318'],
} as const;

export const DEST = { mpesa: 'M-Pesa', emola: 'e-Mola', nib: 'Conta bancária' } as const;

export function luhn(num: string): boolean {
  let sum = 0, alt = false;
  for (let i = num.length - 1; i >= 0; i--) {
    let n = +num[i];
    if (alt) { n *= 2; if (n > 9) n -= 9; }
    sum += n; alt = !alt;
  }
  return num.length >= 13 && sum % 10 === 0;
}

export const uid = () => (crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + Math.random().toString(16).slice(2));
