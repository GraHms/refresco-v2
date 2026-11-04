import { useEffect, useState } from 'react';
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { ArrowLeft, ArrowRight, Check, CircleAlert, CircleCheck, Eye, EyeOff, Info, Link as LinkIcon, PartyPopper, ShieldCheck, Ticket } from 'lucide-react';
import { api, ApiError, Me } from '../lib/api';
import { digits, fmtPhone } from '../lib/format';
import { useSession } from '../lib/session';
import { Brand, ErrorBox, Logo, Spinner, useMedia } from '../components/ui';

const RED = '#B42318';
const bd = (bad?: boolean, good?: boolean) => (bad ? RED : good ? '#1A1410' : '#E4DDD1');
const okId = (v: string) => /^\S+@\S+\.\S+$/.test(v.trim()) || /^(8[2-7])\d{7}$/.test(v.replace(/\D/g, '').replace(/^258/, ''));
const errMsg = (e: unknown) => (e instanceof ApiError ? e.message : 'Sem ligação. Tenta de novo.');
const h2: React.CSSProperties = { margin: 0, fontFamily: 'var(--display)', fontStretch: '125%', fontWeight: 800, fontSize: 32, lineHeight: 1.05, letterSpacing: '-0.04em' };
const inp = (border: string, extra?: React.CSSProperties): React.CSSProperties => ({ height: 52, padding: '0 16px', borderRadius: 14, border: `1.5px solid ${border}`, background: '#fff', outline: 'none', fontSize: 15, minWidth: 0, ...extra });
const primary: React.CSSProperties = { height: 54, border: 0, borderRadius: 15, fontSize: 16, gap: 10 };

export function homeFor(m: Me | null | undefined) { return m?.role === 'superadmin' ? '/admin' : '/painel'; }

type Screen = 'login' | 'register' | 'created' | 'forgot';

export default function Auth() {
  const loc = useLocation();
  const wide = useMedia('(min-width: 900px)');
  const screen: Screen = loc.pathname.startsWith('/registar') ? 'register' : loc.pathname.startsWith('/recuperar') ? 'forgot' : 'login';
  const [created, setCreated] = useState<string>('');
  const [showPw, setShowPw] = useState(false);
  const pw = { type: showPw ? 'text' : 'password', toggle: () => setShowPw(s => !s), Icon: showPw ? EyeOff : Eye };

  useEffect(() => { document.title = (screen === 'register' ? 'Criar conta' : screen === 'forgot' ? 'Recuperar palavra-passe' : 'Entrar') + ' · refres.co'; }, [screen]);

  return (
    <div style={{ minHeight: '100vh', display: 'flex', background: '#F6EFE6', fontSize: 15 }}>
      {wide && <Aside />}
      <main style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', padding: wide ? '28px clamp(24px,4vw,56px)' : '18px 20px 24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          {!wide && <Link to="/"><Logo h={32} /></Link>}
          <span style={{ flex: 1 }} />
          <Link to="/" style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 14, fontWeight: 600, color: '#57524B' }}><ArrowLeft size={15} />Voltar ao site</Link>
        </div>
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '32px 0' }}>
          <div style={{ width: '100%', maxWidth: 420, display: 'flex', flexDirection: 'column', gap: 22 }}>
            {created ? <Created handle={created} /> :
              screen === 'login' ? <Login pw={pw} /> :
                screen === 'register' ? <Register pw={pw} onCreated={setCreated} /> : <Forgot pw={pw} />}
          </div>
        </div>
        <div style={{ textAlign: 'center', fontSize: 12.5, color: '#6E6861' }}>© 2026 Refres, Lda · Maputo</div>
      </main>
    </div>
  );
}

type Pw = { type: string; toggle: () => void; Icon: typeof Eye };

function Aside() {
  return (
    <aside style={{ flex: '0 0 44%', position: 'sticky', top: 0, height: '100vh', overflow: 'hidden', background: '#1A1410', color: '#F6EFE6', padding: '40px 48px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
      <Link to="/" style={{ position: 'relative', zIndex: 1, alignSelf: 'flex-start' }}><Logo h={40} white /></Link>
      <div style={{ position: 'absolute', right: -90, bottom: -110, display: 'flex', gap: 40, opacity: 0.9, pointerEvents: 'none' }}>
        <span style={{ width: 220, height: 460, border: '38px solid #FF6A1A', borderRight: 0, borderRadius: '100px 0 0 100px' }} />
        <span style={{ width: 220, height: 460, border: '38px solid #FF6A1A', borderLeft: 0, borderRadius: '0 100px 100px 0' }} />
      </div>
      <div style={{ position: 'relative', zIndex: 1, display: 'flex', flexDirection: 'column', gap: 22, maxWidth: 440 }}>
        <div className="mono" style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12, letterSpacing: '.1em', textTransform: 'uppercase', color: '#A89C8F' }}><span style={{ width: 8, height: 8, borderRadius: '50%', background: '#FF6A1A' }} />Acesso por convite</div>
        <h1 className="display" style={{ margin: 0, fontWeight: 800, fontSize: 'clamp(36px,3.6vw,52px)', lineHeight: 0.98, letterSpacing: '-0.045em' }}>Os teus fãs pagam-te um refresco.</h1>
        <p style={{ margin: 0, fontSize: 17, lineHeight: 1.55, color: '#CFC6B8' }}>Apoios por M-Pesa num só link, com e-Mola e cartão em breve. Por agora a refres.co está aberta apenas a criadores convidados.</p>
      </div>
      <div style={{ position: 'relative', zIndex: 1, maxWidth: 360, padding: 18, borderRadius: 20, background: '#F6EFE6', color: '#1A1410', display: 'flex', flexDirection: 'column', gap: 10, boxShadow: '0 30px 60px -20px rgba(0,0,0,.5)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}><span style={{ fontWeight: 600, fontSize: 14 }}>Tu Pra Tu Podcast</span><span className="mono" style={{ fontSize: 11.5, color: '#6E6861' }}>há 4 min</span></div>
        <div style={{ fontSize: 14, lineHeight: 1.45, color: '#2B2723' }}>“Episódio brutal. Continuem com as conversas de Tu Pra Tu!”</div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12.5, color: '#57524B' }}><Brand name="x" size={12} />@helio.tembe</span>
          <span className="display" style={{ fontWeight: 800, fontSize: 18, color: '#D14400' }}>+300 MT</span>
        </div>
      </div>
    </aside>
  );
}

function PwInput({ value, onChange, pw, border, placeholder, autoComplete, h = 52 }: { value: string; onChange: (v: string) => void; pw: Pw; border: string; placeholder: string; autoComplete: string; h?: number }) {
  return (
    <span style={{ display: 'flex', alignItems: 'center', height: h, borderRadius: 14, border: `1.5px solid ${border}`, background: '#fff', overflow: 'hidden' }}>
      <input type={pw.type} value={value} onChange={e => onChange(e.target.value)} placeholder={placeholder} autoComplete={autoComplete} aria-label="Palavra-passe"
        style={{ flex: 1, minWidth: 0, height: '100%', border: 0, outline: 'none', padding: '0 16px', fontSize: 15, background: 'transparent' }} />
      <button type="button" onClick={pw.toggle} aria-label="Mostrar palavra-passe" style={{ width: 48, height: '100%', border: 0, background: 'transparent', display: 'grid', placeItems: 'center', color: '#6E6861' }}><pw.Icon size={18} /></button>
    </span>
  );
}

function Login({ pw }: { pw: Pw }) {
  const nav = useNavigate();
  const { set } = useSession();
  const [params] = useSearchParams();
  const [id, setId] = useState(''); const [p, setP] = useState('');
  const [tried, setTried] = useState(false); const [busy, setBusy] = useState(false); const [err, setErr] = useState('');
  const idBad = tried && !okId(id), pwBad = tried && p.length < 6;
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!okId(id) || p.length < 6) { setTried(true); return; }
    setBusy(true); setErr('');
    try {
      const m = await api.post<Me>('/auth/login', { identifier: id, password: p });
      set(m);
      const next = params.get('next');
      nav(next && next.startsWith('/') && !next.startsWith('//') ? next : homeFor(m), { replace: true });
    } catch (e) { setErr(errMsg(e)); } finally { setBusy(false); }
  }
  return (
    <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 22 }}>
      <div><h2 style={h2}>Entrar</h2><p style={{ margin: '8px 0 0', color: '#57524B' }}>Bem-vindo de volta. Entra para veres os teus apoios.</p></div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <label style={{ display: 'flex', flexDirection: 'column', gap: 7 }}><span style={{ fontSize: 13, fontWeight: 600 }}>Email ou telemóvel</span>
          <input value={id} onChange={e => { setId(e.target.value); setErr(''); }} autoComplete="username" placeholder="nome@email.com ou 84 123 4567" style={inp(bd(idBad))} /></label>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 7 }}>
          <span style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, fontWeight: 600 }}><span>Palavra-passe</span><Link to="/recuperar" state={{ id }} style={{ fontWeight: 600, color: '#A83700' }}>Esqueceste-te?</Link></span>
          <PwInput value={p} onChange={v => { setP(v); setErr(''); }} pw={pw} border={bd(pwBad)} placeholder="••••••••" autoComplete="current-password" />
        </div>
        {(err || idBad || pwBad) && <ErrorBox>{err || (idBad ? 'Escreve um email ou número válido.' : 'A palavra-passe tem pelo menos 6 caracteres.')}</ErrorBox>}
        <button type="submit" className="btn btn-primary" style={primary} disabled={busy}>{busy && <Spinner />}{busy ? 'A entrar…' : 'Entrar'}</button>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, color: '#A0988D', fontSize: 13 }}><span style={{ flex: 1, height: 1, background: '#E4DDD1' }} />ainda não tens conta?<span style={{ flex: 1, height: 1, background: '#E4DDD1' }} /></div>
      <Link to="/registar" className="btn" style={{ height: 52, borderRadius: 15, border: '1.5px solid #1A1410', background: 'transparent', fontSize: 15 }}><Ticket size={17} />Tenho um convite</Link>
    </form>
  );
}

function Register({ pw, onCreated }: { pw: Pw; onCreated: (h: string) => void }) {
  const { set } = useSession();
  const [params] = useSearchParams();
  const [code, setCode] = useState((params.get('convite') || '').toUpperCase().replace(/[^A-Z0-9-]/g, '').slice(0, 24));
  const [cs, setCs] = useState<'idle' | 'ok' | 'bad' | 'used' | 'empty'>('idle');
  const [checking, setChecking] = useState(false);
  const [name, setName] = useState(''); const [handle, setHandle] = useState(''); const [touched, setTouched] = useState(false);
  const [email, setEmail] = useState(''); const [phone, setPhone] = useState(''); const [p, setP] = useState(''); const [terms, setTerms] = useState(false);
  const [tried, setTried] = useState(false); const [busy, setBusy] = useState(false); const [err, setErr] = useState('');
  const [avail, setAvail] = useState<{ h: string; ok: boolean; why: string } | null>(null);
  const [wl, setWl] = useState(''); const [wlPos, setWlPos] = useState(0);

  async function check(c = code) {
    if (!c.trim()) { setCs('empty'); return; }
    setChecking(true);
    try { const r = await api.post<{ valid: boolean; reason?: string }>('/auth/invites/check', { code: c }); setCs(r.valid ? 'ok' : r.reason === 'used' ? 'used' : 'bad'); }
    catch { setCs('bad'); } finally { setChecking(false); }
  }
  useEffect(() => { if (code) check(code); }, []);

  useEffect(() => {
    if (!/^[a-z0-9.]{3,24}$/.test(handle)) { setAvail(null); return; }
    const t = setTimeout(() => {
      api.get<{ available: boolean; reason: string }>('/handles/' + handle).then(r => setAvail({ h: handle, ok: r.available, why: r.reason })).catch(() => {});
    }, 300);
    return () => clearTimeout(t);
  }, [handle]);

  const invOk = cs === 'ok';
  const hs = !handle ? { text: 'Podes mudar depois.', color: '#6E6861', icon: 'info' }
    : !/^[a-z0-9.]{3,24}$/.test(handle) ? { text: 'Usa 3 a 24 letras minúsculas, números ou pontos.', color: RED, icon: 'alert' }
      : !avail || avail.h !== handle ? { text: 'A verificar…', color: '#6E6861', icon: 'info' }
        : !avail.ok ? { text: 'Este link já está ocupado.', color: RED, icon: 'alert' } : { text: 'Disponível!', color: '#17803D', icon: 'ok' };
  const score = p.length === 0 ? 0 : Math.min(4, +(p.length >= 8) + +/[A-Z]/.test(p) + +/\d/.test(p) + +/[^A-Za-z0-9]/.test(p));
  const emailOk = /^\S+@\S+\.\S+$/.test(email), phoneOk = /^8[4-7]\d{7}$/.test(phone);
  const valid = name.trim().length > 1 && hs.color === '#17803D' && emailOk && phoneOk && p.length >= 8 && terms;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!valid) { setTried(true); return; }
    setBusy(true); setErr('');
    try {
      const m = await api.post<Me>('/auth/register', { code, name, handle, email, phone, password: p, terms });
      set(m); onCreated(handle);
    } catch (e) { setErr(errMsg(e)); } finally { setBusy(false); }
  }

  const hint = invOk ? 'Convite válido · enviado por Equipa refres.co' : cs === 'bad' ? 'Este código não existe. Confirma com quem te convidou.' : cs === 'used' ? 'Este código já foi usado.' : cs === 'empty' ? 'Escreve o código que recebeste.' : 'Escreve o código de convite que recebeste.';
  const hintColor = invOk ? '#17803D' : cs === 'idle' ? '#6E6861' : RED;
  const field: React.CSSProperties = { height: 50, padding: '0 14px', borderRadius: 13, background: '#fff', outline: 'none', fontSize: 15, minWidth: 0 };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 22 }}>
      <div><h2 style={h2}>Criar conta</h2><p style={{ margin: '8px 0 0', color: '#57524B' }}>A refres.co está aberta só a criadores convidados. Começa pelo teu código.</p></div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8, padding: 16, borderRadius: 20, background: invOk ? '#F1F8F3' : '#fff', border: `1.5px solid ${invOk ? '#BFE3CB' : cs === 'bad' || cs === 'used' ? '#F2C1BA' : '#E4DDD1'}`, transition: 'background-color .2s,border-color .2s' }}>
        <span style={{ fontSize: 13, fontWeight: 600 }}>Código de convite</span>
        <form onSubmit={e => { e.preventDefault(); check(); }} style={{ display: 'flex', gap: 8 }}>
          <input value={code} onChange={e => { setCode(e.target.value.toUpperCase().replace(/[^A-Z0-9-]/g, '').slice(0, 24)); setCs('idle'); }} placeholder="EX.: REFRES-BETA" aria-label="Código de convite"
            className="mono" style={{ ...field, flex: 1, border: '1.5px solid #E4DDD1', letterSpacing: '.06em', textTransform: 'uppercase' }} />
          <button type="submit" className="btn" style={{ flex: 'none', height: 50, padding: '0 18px', borderRadius: 13, border: 0, background: invOk ? '#17803D' : '#1A1410', color: '#fff', gap: 6 }}>
            {invOk ? <Check size={16} /> : <Ticket size={16} />}{invOk ? 'Válido' : checking ? 'A validar' : 'Validar'}
          </button>
        </form>
        <span style={{ display: 'flex', alignItems: 'flex-start', gap: 6, fontSize: 13, lineHeight: 1.4, color: hintColor }}>{invOk ? <CircleCheck size={14} style={{ marginTop: 2 }} /> : cs === 'idle' ? <Info size={14} style={{ marginTop: 2 }} /> : <CircleAlert size={14} style={{ marginTop: 2 }} />}{hint}</span>
      </div>

      {invOk && (
        <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 7 }}><span style={{ fontSize: 13, fontWeight: 600 }}>Nome da página</span>
            <input value={name} onChange={e => { const v = e.target.value.slice(0, 40); setName(v); if (!touched) setHandle(v.toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '').replace(/[^a-z0-9]/g, '').slice(0, 24)); }}
              placeholder="Ex.: Tu Pra Tu Podcast" style={{ ...field, border: `1.5px solid ${bd(tried && name.trim().length < 2)}` }} /></label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 7 }}><span style={{ fontSize: 13, fontWeight: 600 }}>O teu link</span>
            <span style={{ display: 'flex', alignItems: 'center', height: 50, borderRadius: 13, border: `1.5px solid ${bd(hs.color === RED || (tried && !handle), hs.color === '#17803D')}`, background: '#fff', overflow: 'hidden' }}>
              <span className="mono" style={{ height: '100%', display: 'flex', alignItems: 'center', padding: '0 12px', background: '#FBF7F1', borderRight: '1px solid #EFE7DC', fontSize: 14, color: '#57524B' }}>pagarefresco.com/</span>
              <input value={handle} onChange={e => { setHandle(e.target.value.toLowerCase().replace(/[^a-z0-9.]/g, '').slice(0, 24)); setTouched(true); }} placeholder="o-teu-nome" className="mono" style={{ flex: 1, minWidth: 0, height: '100%', border: 0, outline: 'none', padding: '0 12px', fontSize: 14, background: 'transparent' }} />
            </span>
            <span style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12.5, color: hs.color }}>{hs.icon === 'ok' ? <CircleCheck size={13} /> : hs.icon === 'alert' ? <CircleAlert size={13} /> : <Info size={13} />}{hs.text}</span>
          </label>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(min(100%,180px),1fr))', gap: 12 }}>
            <label style={{ display: 'flex', flexDirection: 'column', gap: 7 }}><span style={{ fontSize: 13, fontWeight: 600 }}>Email</span>
              <input type="email" autoComplete="email" value={email} onChange={e => setEmail(e.target.value.trim())} placeholder="nome@email.com" style={{ ...field, border: `1.5px solid ${bd(tried && !emailOk)}` }} /></label>
            <label style={{ display: 'flex', flexDirection: 'column', gap: 7 }}><span style={{ fontSize: 13, fontWeight: 600 }}>Telemóvel</span>
              <span style={{ display: 'flex', alignItems: 'center', height: 50, borderRadius: 13, border: `1.5px solid ${bd(tried && !phoneOk)}`, background: '#fff', overflow: 'hidden' }}>
                <span className="mono" style={{ padding: '0 10px', fontSize: 13.5, color: '#57524B' }}>+258</span>
                <input type="tel" autoComplete="tel-national" value={fmtPhone(phone)} onChange={e => setPhone(digits(e.target.value, 9))} placeholder="84 123 4567" className="mono" style={{ flex: 1, minWidth: 0, height: '100%', border: 0, outline: 'none', padding: '0 10px 0 0', fontSize: 14.5, background: 'transparent' }} />
              </span></label>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 7 }}><span style={{ fontSize: 13, fontWeight: 600 }}>Palavra-passe</span>
            <PwInput value={p} onChange={setP} pw={pw} border={bd(tried && p.length < 8)} placeholder="Pelo menos 8 caracteres" autoComplete="new-password" h={50} />
            <span style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <span style={{ flex: 1, display: 'flex', gap: 4 }}>{[1, 2, 3, 4].map(i => <span key={i} style={{ flex: 1, height: 4, borderRadius: 2, background: i <= score ? ['#B42318', '#D98B00', '#17803D', '#17803D'][score - 1] : '#E4DDD1', transition: 'background-color .2s' }} />)}</span>
              <span style={{ fontSize: 12, color: '#6E6861', minWidth: 64, textAlign: 'right' }}>{['', 'Fraca', 'Razoável', 'Boa', 'Forte'][score]}</span>
            </span>
          </div>
          <button type="button" onClick={() => setTerms(t => !t)} aria-pressed={terms} style={{ display: 'flex', alignItems: 'flex-start', gap: 10, border: 0, background: 'transparent', padding: 0, textAlign: 'left', fontSize: 13.5, color: '#57524B' }}>
            <span style={{ flex: 'none', width: 20, height: 20, marginTop: 1, borderRadius: 6, border: `1.5px solid ${terms ? '#D14400' : tried && !terms ? RED : '#CFC6B8'}`, background: terms ? '#D14400' : '#fff', display: 'grid', placeItems: 'center', color: '#fff' }}>{terms && <Check size={13} />}</span>
            <span>Aceito os <b style={{ textDecoration: 'underline' }}>Termos</b> e a <b style={{ textDecoration: 'underline' }}>Política de privacidade</b>, incluindo a taxa de 15% por apoio.</span>
          </button>
          {(err || (tried && !valid)) && <ErrorBox>{err || (!terms && name && emailOk && phoneOk && p.length >= 8 ? 'Aceita os termos para continuar.' : 'Confirma os campos assinalados.')}</ErrorBox>}
          <button type="submit" className="btn btn-primary" style={primary} disabled={busy}>{busy && <Spinner />}{busy ? 'A criar conta…' : 'Criar conta'}</button>
        </form>
      )}
      {!invOk && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10, padding: 16, borderRadius: 20, background: '#fff', border: '1px solid #E4DDD1' }}>
          <div style={{ fontWeight: 600 }}>Ainda não tens convite?</div>
          <div style={{ fontSize: 13.5, color: '#57524B', marginTop: -6 }}>Deixa o teu contacto e avisamos quando abrirmos mais vagas.</div>
          {!wlPos ? <form onSubmit={async e => { e.preventDefault(); if (!/^\S+@\S+\.\S+$/.test(wl)) return; try { const r = await api.post<{ position: number }>('/auth/waitlist', { email: wl }); setWlPos(r.position); } catch { /* ignore */ } }} style={{ display: 'flex', gap: 8 }}>
            <input type="email" value={wl} onChange={e => setWl(e.target.value)} placeholder="nome@email.com" aria-label="Email" style={{ flex: 1, minWidth: 0, height: 46, padding: '0 14px', borderRadius: 12, border: '1.5px solid #E4DDD1', outline: 'none', fontSize: 14.5 }} />
            <button type="submit" className="btn btn-dark" style={{ height: 46, padding: '0 16px', borderRadius: 12 }}>Entrar na lista</button>
          </form> : <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 14, color: '#17803D', fontWeight: 600 }}><CircleCheck size={16} />Estás na lista. Posição {wlPos}.</div>}
        </div>
      )}
      <div style={{ textAlign: 'center', fontSize: 14, color: '#57524B' }}>Já tens conta? <Link to="/entrar" style={{ fontWeight: 600, color: '#A83700' }}>Entrar</Link></div>
    </div>
  );
}

function Created({ handle }: { handle: string }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', gap: 18 }}>
      <span style={{ width: 72, height: 72, borderRadius: 24, background: '#D14400', color: '#fff', display: 'grid', placeItems: 'center' }}><PartyPopper size={32} /></span>
      <div><h2 style={{ ...h2, fontSize: 30 }}>Conta criada!</h2><p style={{ margin: '10px 0 0', color: '#57524B' }}>A tua página já existe. Antes do primeiro levantamento, a equipa refres.co vai verificar a tua conta.</p></div>
      <div style={{ width: '100%', display: 'flex', alignItems: 'center', gap: 10, padding: '14px 16px', borderRadius: 16, background: '#fff', border: '1px solid #E4DDD1' }}>
        <LinkIcon size={16} color="#6E6861" /><span className="mono" style={{ flex: 1, textAlign: 'left', fontSize: 14 }}>pagarefresco.com/{handle}</span>
        <span style={{ height: 24, padding: '0 8px', borderRadius: 7, background: '#E3F3E8', color: '#17803D', fontSize: 12, fontWeight: 600, display: 'flex', alignItems: 'center' }}>Ativa</span>
      </div>
      <Link to="/painel/pagina" className="btn btn-primary" style={{ ...primary, width: '100%' }}>Configurar a minha página<ArrowRight size={17} /></Link>
    </div>
  );
}

function Forgot({ pw }: { pw: Pw }) {
  const loc = useLocation();
  const [step, setStep] = useState(1);
  const [id, setId] = useState<string>((loc.state as { id?: string } | null)?.id || '');
  const [code, setCode] = useState(''); const [token, setToken] = useState('');
  const [p1, setP1] = useState(''); const [p2, setP2] = useState('');
  const [tried, setTried] = useState(false); const [busy, setBusy] = useState(false); const [err, setErr] = useState(''); const [secs, setSecs] = useState(0);

  useEffect(() => { if (secs <= 0) return; const t = setTimeout(() => setSecs(s => s - 1), 1000); return () => clearTimeout(t); }, [secs]);

  const masked = id.includes('@') ? id.replace(/^(.)(.*)(@.*)$/, (_m, a, b, c) => a + '•'.repeat(Math.max(2, b.length)) + c)
    : '+258 ' + fmtPhone(id.replace(/\D/g, '').replace(/^258/, '')).replace(/\d(?=.{4})/g, '•');

  async function send() {
    if (!okId(id)) { setTried(true); return; }
    setBusy(true); setErr('');
    try { await api.post('/auth/password/forgot', { identifier: id }); setStep(2); setTried(false); setSecs(45); } catch (e) { setErr(errMsg(e)); } finally { setBusy(false); }
  }
  async function verify() {
    if (code.length !== 6) { setTried(true); return; }
    setBusy(true); setErr('');
    try { const r = await api.post<{ reset_token: string }>('/auth/password/verify', { identifier: id, code }); setToken(r.reset_token); setStep(3); setTried(false); } catch (e) { setErr(errMsg(e)); } finally { setBusy(false); }
  }
  async function save() {
    if (p1.length < 8 || p1 !== p2) { setTried(true); return; }
    setBusy(true); setErr('');
    try { await api.post('/auth/password/reset', { reset_token: token, password: p1 }); setStep(4); } catch (e) { setErr(errMsg(e)); } finally { setBusy(false); }
  }
  const fErr = tried && step === 3 ? (p1.length < 8 ? 'A palavra-passe tem de ter pelo menos 8 caracteres.' : p1 !== p2 ? 'As palavras-passe não são iguais.' : '') : '';
  const h: React.CSSProperties = { ...h2, fontSize: 30 };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 22 }}>
      <Link to="/entrar" style={{ alignSelf: 'flex-start', display: 'flex', alignItems: 'center', gap: 6, fontWeight: 600, fontSize: 14 }}><ArrowLeft size={16} />Voltar a entrar</Link>
      <div style={{ display: 'flex', gap: 6 }}>{[1, 2, 3].map(i => <span key={i} style={{ flex: 1, height: 4, borderRadius: 2, background: i < step || step === 4 ? '#1A1410' : i === step ? '#D14400' : '#E4DDD1', transition: 'background-color .2s' }} />)}</div>
      {step === 1 && <form onSubmit={e => { e.preventDefault(); send(); }} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div><h2 style={h}>Recuperar palavra-passe</h2><p style={{ margin: '8px 0 0', color: '#57524B' }}>Enviamos um código de 6 dígitos para o email da tua conta.</p></div>
        <label style={{ display: 'flex', flexDirection: 'column', gap: 7 }}><span style={{ fontSize: 13, fontWeight: 600 }}>Email ou telemóvel</span>
          <input value={id} onChange={e => setId(e.target.value)} autoComplete="username" placeholder="nome@email.com ou 84 123 4567" style={inp(bd(tried && !okId(id)))} /></label>
        {err && <ErrorBox>{err}</ErrorBox>}
        <button type="submit" className="btn btn-primary" style={primary} disabled={busy}>{busy && <Spinner />}Enviar código</button>
      </form>}
      {step === 2 && <form onSubmit={e => { e.preventDefault(); verify(); }} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div><h2 style={h}>Escreve o código</h2><p style={{ margin: '8px 0 0', color: '#57524B' }}>Se a conta existir, enviámos {id.includes('@') ? <>para <b style={{ color: '#1A1410' }}>{masked}</b></> : 'para o email da conta'}.</p></div>
        <input type="tel" inputMode="numeric" autoComplete="one-time-code" value={code} onChange={e => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))} placeholder="000000" aria-label="Código"
          className="mono" style={inp(bd(tried && code.length !== 6), { height: 64, borderRadius: 16, fontSize: 28, letterSpacing: '.5em', textAlign: 'center' })} />
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13.5, color: '#6E6861' }}>
          <span>{secs > 0 ? `Podes pedir outro código em 0:${String(secs).padStart(2, '0')}` : 'Não recebeste o código?'}</span>
          <button type="button" onClick={() => { if (secs === 0) send(); }} style={{ border: 0, background: 'transparent', padding: 0, fontWeight: 600, color: secs > 0 ? '#A0988D' : '#A83700' }} disabled={secs > 0}>Reenviar código</button>
        </div>
        {err && <ErrorBox>{err}</ErrorBox>}
        <button type="submit" className="btn btn-primary" style={primary} disabled={busy}>Confirmar código</button>
      </form>}
      {step === 3 && <form onSubmit={e => { e.preventDefault(); save(); }} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div><h2 style={h}>Nova palavra-passe</h2><p style={{ margin: '8px 0 0', color: '#57524B' }}>Escolhe uma palavra-passe com pelo menos 8 caracteres.</p></div>
        <input type={pw.type} value={p1} onChange={e => setP1(e.target.value)} autoComplete="new-password" placeholder="Nova palavra-passe" style={inp('#E4DDD1')} />
        <input type={pw.type} value={p2} onChange={e => setP2(e.target.value)} autoComplete="new-password" placeholder="Repete a palavra-passe" style={inp(bd(tried && p1 !== p2))} />
        {(fErr || err) && <ErrorBox>{fErr || err}</ErrorBox>}
        <button type="submit" className="btn btn-primary" style={primary} disabled={busy}>Guardar palavra-passe</button>
      </form>}
      {step === 4 && <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', gap: 16 }}>
        <span style={{ width: 68, height: 68, borderRadius: 22, background: '#E3F3E8', color: '#17803D', display: 'grid', placeItems: 'center' }}><ShieldCheck size={30} /></span>
        <div><h2 style={{ ...h2, fontSize: 28 }}>Palavra-passe alterada</h2><p style={{ margin: '8px 0 0', color: '#57524B' }}>Já podes entrar com a nova palavra-passe.</p></div>
        <Link to="/entrar" className="btn btn-primary" style={{ ...primary, width: '100%' }}>Entrar</Link>
      </div>}
    </div>
  );
}
