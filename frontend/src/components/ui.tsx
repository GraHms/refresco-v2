import type { CSSProperties } from 'react';
import { createContext, ReactNode, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { CircleCheck, CircleAlert, X } from 'lucide-react';
import { siFacebook, siInstagram, siMastercard, siSpotify, siTiktok, siVisa, siWhatsapp, siX, siYoutube } from 'simple-icons';

const BRANDS = { youtube: siYoutube, spotify: siSpotify, instagram: siInstagram, tiktok: siTiktok, x: siX, facebook: siFacebook, whatsapp: siWhatsapp, visa: siVisa, mastercard: siMastercard };
export type BrandName = keyof typeof BRANDS;

export function Brand({ name, size = 16, color, style }: { name: BrandName; size?: number; color?: string; style?: React.CSSProperties }) {
  const icon = BRANDS[name];
  return (
    <svg role="img" aria-label={icon.title} width={size} height={size} viewBox="0 0 24 24" fill={color || 'currentColor'} style={style}>
      <path d={icon.path} />
    </svg>
  );
}

export const SOCIALS: { key: BrandName; label: string; prefix: string; ph: string; url: (v: string) => string }[] = [
  { key: 'youtube', label: 'YouTube', prefix: 'youtube.com/', ph: '@canal', url: v => `https://youtube.com/${v.startsWith('@') ? v : '@' + v}` },
  { key: 'spotify', label: 'Spotify', prefix: 'spotify/', ph: 'o-teu-show', url: v => (v.startsWith('http') ? v : `https://open.spotify.com/search/${encodeURIComponent(v)}`) },
  { key: 'instagram', label: 'Instagram', prefix: '@', ph: 'utilizador', url: v => `https://instagram.com/${v.replace(/^@/, '')}` },
  { key: 'tiktok', label: 'TikTok', prefix: '@', ph: 'utilizador', url: v => `https://tiktok.com/@${v.replace(/^@/, '')}` },
  { key: 'facebook', label: 'Facebook', prefix: 'fb.com/', ph: 'pagina', url: v => `https://facebook.com/${v}` },
  { key: 'x', label: 'X', prefix: '@', ph: 'utilizador', url: v => `https://x.com/${v.replace(/^@/, '')}` },
];

export function Logo({ h = 32, white }: { h?: number; white?: boolean }) {
  return <img src={white ? '/assets/refres-logo-orange-white.png' : '/assets/refres-logo-orange.png'} alt="Refresco" style={{ height: h, width: 'auto', display: 'block' }} />;
}

export function Toggle({ on, onClick, label, disabled }: { on: boolean; onClick: () => void; label: string; disabled?: boolean }) {
  return <button type="button" className={'toggle' + (on ? ' on' : '')} onClick={onClick} aria-label={label} aria-pressed={on} disabled={disabled}><i /></button>;
}

export function Seg<T extends string>({ value, options, onChange, small, soon = [] }: { value: T; options: [T, string][]; onChange: (v: T) => void; small?: boolean; soon?: T[] }) {
  return (
    <div className="seg" style={small ? { background: '#F0EBE3', borderRadius: 13 } : undefined}>
      {options.map(([k, l]) => (
        <button key={k} type="button" disabled={soon.includes(k)} className={value === k ? 'on' : ''} onClick={() => onChange(k)} style={{ ...(small ? { height: 32, padding: '0 11px', fontSize: 12.5, borderRadius: 9 } : {}), ...(soon.includes(k) ? { opacity: 0.6, cursor: 'not-allowed', lineHeight: 1.1 } : {}) }}>
          {l}{soon.includes(k) && <span style={{ display: 'block', fontSize: 10.5, fontWeight: 600, color: '#A83700' }}>Em breve</span>}
        </button>
      ))}
    </div>
  );
}

export function Badge({ st, dot = true, style }: { st: readonly [string, string, string]; dot?: boolean; style?: React.CSSProperties }) {
  return <span className="badge" style={{ background: st[1], color: st[2], ...style }}>{dot && <i />}{st[0]}</span>;
}

export function Avatar({ text, bg, fg, size = 40, radius = 13, fontSize = 13 }: { text: string; bg: string; fg: string; size?: number; radius?: number; fontSize?: number }) {
  return <span className="avatar" style={{ width: size, height: size, borderRadius: radius, background: bg, color: fg, fontSize }}>{text}</span>;
}

export function ErrorBox({ children }: { children: ReactNode }) {
  return <div className="err-box" role="alert"><CircleAlert size={15} />{children}</div>;
}

export function Spinner() { return <span className="spinner" aria-hidden />; }

export function Modal({ onClose, children, label }: { onClose: () => void; children: ReactNode; label: string }) {
  useEscape(onClose);
  return (
    <>
      <div className="overlay" style={{ zIndex: 80 }} onClick={onClose} />
      <div className="modal" style={{ zIndex: 81 }} role="dialog" aria-modal="true" aria-label={label}>{children}</div>
    </>
  );
}

export function ModalHead({ title, sub, onClose }: { title: string; sub?: ReactNode; onClose: () => void }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 12 }}>
      <div>
        <div className="display" style={{ fontWeight: 800, fontSize: 22, lineHeight: 1.15, letterSpacing: '-0.03em' }}>{title}</div>
        {sub && <div style={{ marginTop: 4, fontSize: 13.5, color: '#57524B' }}>{sub}</div>}
      </div>
      <button className="icon-btn" style={{ width: 36, height: 36, borderRadius: 11 }} onClick={onClose} aria-label="Fechar"><X size={16} /></button>
    </div>
  );
}

export function Drawer({ onClose, children, label }: { onClose: () => void; children: ReactNode; label: string }) {
  useEscape(onClose);
  return (
    <>
      <div className="overlay" style={{ zIndex: 70, background: 'rgba(20,18,16,.28)' }} onClick={onClose} />
      <aside className="drawer" style={{ zIndex: 71 }} role="dialog" aria-modal="true" aria-label={label}>{children}</aside>
    </>
  );
}

export function useEscape(fn: () => void) {
  const ref = useRef(fn);
  ref.current = fn;
  useEffect(() => {
    const h = (e: KeyboardEvent) => { if (e.key === 'Escape') ref.current(); };
    window.addEventListener('keydown', h);
    return () => window.removeEventListener('keydown', h);
  }, []);
}

const ToastCtx = createContext<(msg: string, error?: boolean) => void>(() => {});
export const useToast = () => useContext(ToastCtx);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [t, setT] = useState<{ msg: string; err: boolean; k: number } | null>(null);
  const timer = useRef<number>();
  const show = useCallback((msg: string, err = false) => {
    window.clearTimeout(timer.current);
    setT({ msg, err, k: Date.now() });
    timer.current = window.setTimeout(() => setT(null), 2800);
  }, []);
  return (
    <ToastCtx.Provider value={show}>
      {children}
      {t && (
        <div key={t.k} className="toast" role="status">
          {t.err ? <CircleAlert size={17} color="#FF9A8A" /> : <CircleCheck size={17} color="#5BD08A" />}{t.msg}
        </div>
      )}
    </ToastCtx.Provider>
  );
}

export function useMedia(q: string) {
  const [m, setM] = useState(() => window.matchMedia(q).matches);
  useEffect(() => {
    const mq = window.matchMedia(q);
    const h = () => setM(mq.matches);
    mq.addEventListener('change', h);
    return () => mq.removeEventListener('change', h);
  }, [q]);
  return m;
}

export async function copy(text: string) {
  try { await navigator.clipboard.writeText(text); } catch { /* clipboard unavailable */ }
}

// The wallets' real logos, as they appear on the customer's phone.
export function WalletLogo({ wallet, size = 30, radius = 9, style }: { wallet: 'mpesa' | 'emola'; size?: number; radius?: number; style?: CSSProperties }) {
  return <img src={`/assets/methods/${wallet}.png`} alt={wallet === 'mpesa' ? 'M-Pesa' : 'e-Mola'} width={size} height={size} style={{ display: 'block', flex: 'none', width: size, height: size, borderRadius: radius, objectFit: 'cover', ...style }} />;
}

export function PoweredByRiha({ style }: { style?: CSSProperties }) {
  return (
    <a href="https://rihagateway.com" target="_blank" rel="noopener noreferrer" style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 12.5, color: '#6E6861', textDecoration: 'none', ...style }}>
      Powered by<img src="/assets/riha-icon-dark.png" alt="" width={15} height={15} style={{ display: 'block' }} /><b style={{ color: '#1A1410', letterSpacing: '.14em', fontSize: 11.5 }}>RIHA</b>
    </a>
  );
}
