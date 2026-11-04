import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowUpRight, Check, ChevronDown, CircleAlert, CircleCheck, CreditCard, Info, Landmark, Lock, RotateCw } from 'lucide-react';
import { api, ApiError, Channel, Page, SupportStatus } from '../lib/api';
import { dateTime, digits, fmt, fmtPhone, luhn, mtN, uid, walletOf } from '../lib/format';
import { Brand, BrandName, Logo, SOCIALS } from '../components/ui';

const NETS: [string, string, BrandName][] = [['x', 'X', 'x'], ['instagram', 'Instagram', 'instagram'], ['tiktok', 'TikTok', 'tiktok'], ['facebook', 'Facebook', 'facebook'], ['youtube', 'YouTube', 'youtube']];
const CHN = { mpesa: 'M-Pesa', emola: 'e-Mola' } as const;
const RED = '#B42318';

type Phase = 'form' | 'wait' | 'otp' | 'done';

export default function SupportPage() {
  const { handle = '' } = useParams();
  const [page, setPage] = useState<Page | null>(null);
  const [missing, setMissing] = useState(false);

  const [offline, setOffline] = useState(false);
  const [attempt, setAttempt] = useState(0);

  // Only the server can say a page doesn't exist: a dropped request says nothing.
  useEffect(() => {
    api.get<Page>('/pages/' + encodeURIComponent(handle.toLowerCase()))
      .then(p => { setOffline(false); setPage(p); document.title = `Apoia ${p.name} com um refresco | refres.co`; })
      .catch(e => { if (e instanceof ApiError && e.status === 404) setMissing(true); else setOffline(true); });
  }, [handle, attempt]);

  return (
    <div style={{ minHeight: '100vh', background: '#F6EFE6', fontSize: 15 }}>
      <header style={{ maxWidth: 600, margin: '0 auto', padding: '16px 20px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16 }}>
        <Link to="/" style={{ display: 'flex' }}><Logo h={32} /></Link>
        <Link to="/registar" className="btn" style={{ height: 38, padding: '0 13px', fontSize: 13.5 }}>Cria a tua página<ArrowUpRight size={15} /></Link>
      </header>
      <main style={{ maxWidth: 600, margin: '0 auto', padding: 'clamp(12px,3vw,32px) 20px 40px', display: 'flex', flexDirection: 'column', gap: 24 }}>
        {page?.sandbox && <div role="status" style={{ padding: '10px 14px', borderRadius: 14, background: '#FBEFD9', color: '#6B3F00', fontSize: 13.5, textAlign: 'center' }}><b>Modo de teste.</b> Nenhum pagamento é real e nada é cobrado.</div>}
        {missing ? <NotFound /> : offline && !page ? <Offline retry={() => { setOffline(false); setAttempt(a => a + 1); }} /> : !page ? <Loading /> : <>
          <Profile page={page} />
          <SupportCard page={page} />
        </>}
        <footer style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, fontSize: 12.5, color: '#6E6861' }}>Pagamento seguro com<Logo h={18} /></footer>
      </main>
    </div>
  );
}

function Loading() {
  return <div style={{ display: 'flex', flexDirection: 'column', gap: 16, alignItems: 'center' }}>
    <div className="skeleton" style={{ width: 260, height: 42 }} /><div className="skeleton" style={{ width: 340, height: 48 }} /><div className="skeleton" style={{ width: '100%', height: 520, borderRadius: 28 }} />
  </div>;
}

function Offline({ retry }: { retry: () => void }) {
  return <section style={{ textAlign: 'center', padding: '60px 0', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
    <h1 className="display" style={{ margin: 0, fontWeight: 800, fontSize: 32, letterSpacing: '-0.04em' }}>Sem ligação</h1>
    <p style={{ margin: 0, color: '#57524B' }}>Não foi possível abrir esta página. Verifica a internet e tenta de novo.</p>
    <button type="button" className="btn btn-primary" onClick={retry}>Tentar de novo</button>
  </section>;
}

function NotFound() {
  return <section style={{ textAlign: 'center', padding: '60px 0', display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
    <h1 className="display" style={{ margin: 0, fontWeight: 800, fontSize: 32, letterSpacing: '-0.04em' }}>Página não encontrada</h1>
    <p style={{ margin: 0, color: '#57524B' }}>Confirma o link com o criador.</p>
    <Link to="/" className="btn">Ir para refres.co</Link>
  </section>;
}

function Profile({ page }: { page: Page }) {
  const socials = SOCIALS.filter(s => page.socials[s.key]);
  return (
    <section style={{ display: 'flex', flexDirection: 'column', gap: 12, textAlign: 'center', alignItems: 'center' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontFamily: 'var(--mono)', fontSize: 11.5, letterSpacing: '.1em', textTransform: 'uppercase', color: '#6E6861' }}>
        <span style={{ width: 7, height: 7, borderRadius: '50%', background: '#FF6A1A' }} />{page.category} · Moçambique
      </div>
      <h1 className="display" style={{ margin: 0, fontWeight: 800, fontSize: 'clamp(30px,6vw,42px)', lineHeight: 1.02, letterSpacing: '-0.04em', textWrap: 'balance' } as React.CSSProperties}>{page.name}</h1>
      {page.bio && <p style={{ margin: 0, maxWidth: 440, fontSize: 16, lineHeight: 1.55, color: '#57524B' }}>{page.bio}</p>}
      {socials.length > 0 && <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
        {socials.map(s => (
          <a key={s.key} href={s.url(page.socials[s.key])} target="_blank" rel="noopener noreferrer nofollow" title={s.label} className="icon-btn" style={{ width: 42, height: 42, borderRadius: 13, borderColor: '#E4DDD1' }}>
            <Brand name={s.key} size={s.key === 'x' ? 15 : 17} />
          </a>
        ))}
      </div>}
    </section>
  );
}

function SupportCard({ page }: { page: Page }) {
  const presets = page.amounts.map(a => a / 100);
  const def = presets[page.default_idx] ?? presets[0];
  const min = page.min_amount / 100;
  const methods = (['mpesa', 'emola', 'card'] as Channel[]).filter(c => page.channels[c]);
  const soon = (page.coming_soon || []) as Channel[];
  const cardRef = useRef<HTMLElement>(null);

  const [phase, setPhase] = useState<Phase>('form');
  const [amt, setAmt] = useState(def);
  const [custom, setCustom] = useState('');
  const [method, setMethod] = useState<Channel>(methods[0] || 'mpesa');
  const [phone, setPhone] = useState('');
  const [card, setCard] = useState(''); const [exp, setExp] = useState(''); const [cvv, setCvv] = useState('');
  const [name, setName] = useState(''); const [net, setNet] = useState('x'); const [netOpen, setNetOpen] = useState(false); const [nameFocus, setNameFocus] = useState(false);
  const [msg, setMsg] = useState('');
  const [tried, setTried] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [sup, setSup] = useState<SupportStatus | null>(null);
  // The USSD prompt cuts the phone's data, so losing the connection here is normal.
  const [reconnecting, setReconnecting] = useState(false);
  const resumeKey = 'refres_support_' + page.handle;
  const [otp, setOtp] = useState('');
  const [now, setNow] = useState(Date.now());
  const idem = useRef(uid());
  const nameRef = useRef<HTMLInputElement>(null);

  const total = custom ? parseInt(custom, 10) || 0 : amt;
  const isWallet = method !== 'card';
  const ch = isWallet ? method as 'mpesa' | 'emola' : 'mpesa';
  const op = walletOf(phone);
  const isHandle = name.startsWith('@');

  const ph = useMemo(() => {
    let hint = `Vais receber um pedido ${CHN[ch]} para confirmar com o PIN.`, color = '#6E6861', icon: 'info' | 'alert' | 'ok' = 'info', valid = false, canSwitch = false;
    if (phone.length >= 2) {
      const opName = op === 'mpesa' ? 'Número Vodacom' : 'Número Movitel';
      if (op === 'tmcel') { hint = page.channels.card ? 'A Tmcel ainda não é suportada. Paga com cartão.' : 'A Tmcel ainda não é suportada. Usa um número M-Pesa (84 ou 85).'; color = RED; icon = 'alert'; }
      else if (!op) { hint = 'Número inválido. Deve começar por 84, 85, 86 ou 87.'; color = RED; icon = 'alert'; }
      else if (op !== ch) { hint = op === 'mpesa' ? 'Este número é Vodacom.' : page.channels.emola ? 'Este número é Movitel.' : 'O e-Mola chega em breve. Por agora usa um número M-Pesa (84 ou 85).'; color = RED; icon = 'alert'; canSwitch = page.channels[op]; }
      else if (phone.length < 9) { const left = 9 - phone.length; hint = `${opName} · ${left === 1 ? 'falta 1 dígito' : `faltam ${left} dígitos`}`; if (tried) { color = RED; icon = 'alert'; } }
      else { valid = true; hint = `${opName} · vais receber o pedido ${CHN[op]}`; color = '#17803D'; icon = 'ok'; }
    } else if (tried) { hint = `Escreve o teu número ${CHN[ch]}.`; color = RED; icon = 'alert'; }
    return { hint, color, icon, valid, canSwitch };
  }, [phone, op, ch, tried, page.channels]);

  const brand = card[0] === '4' ? 'visa' : /^(5[1-5]|2[2-7])/.test(card) ? 'mastercard' : '';
  const mm = parseInt(exp.slice(0, 2), 10), yy = parseInt(exp.slice(2), 10);
  const nowD = new Date(), expOk = exp.length === 4 && mm >= 1 && mm <= 12 && (2000 + yy > nowD.getFullYear() || (2000 + yy === nowD.getFullYear() && mm >= nowD.getMonth() + 1));
  const cardValid = !!brand && card.length >= 15 && luhn(card) && expOk && cvv.length === 3;
  const brandName = brand === 'mastercard' ? 'Mastercard' : 'Visa';
  const amountOk = presets.includes(total) || (page.allow_custom && total >= min);

  useEffect(() => {
    if (phase !== 'wait' && phase !== 'otp') return;
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [phase]);

  // iOS can reload this tab while the customer is on the USSD screen.
  useEffect(() => {
    let id: string | null = null;
    try { id = sessionStorage.getItem(resumeKey); } catch { /* private mode */ }
    if (!id) return;
    api.get<SupportStatus>('/supports/' + id).then(s => {
      if (s.status === 'confirmed') { setSup(s); remember(null); go('done'); }
      else if (s.status === 'pending') { setSup(s); go('wait'); }
      else remember(null);
    }).catch(() => {});
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!sup || phase !== 'wait') return;
    let alive = true, checking = false;
    const check = async () => {
      if (checking || document.hidden) return;
      checking = true;
      try {
        const s = await api.get<SupportStatus>('/supports/' + sup.id);
        if (!alive) return;
        setReconnecting(false);
        setSup(s);
        if (s.status === 'confirmed') { remember(null); go('done'); }
      } catch { if (alive) setReconnecting(true); }
      finally { checking = false; }
    };
    const t = setInterval(check, 2000);
    // Back from the PIN, or the connection returned: answer at once.
    document.addEventListener('visibilitychange', check);
    window.addEventListener('online', check);
    return () => { alive = false; clearInterval(t); document.removeEventListener('visibilitychange', check); window.removeEventListener('online', check); };
  }, [sup?.id, phase]); // eslint-disable-line react-hooks/exhaustive-deps

  function remember(id: string | null) {
    try { if (id) sessionStorage.setItem(resumeKey, id); else sessionStorage.removeItem(resumeKey); } catch { /* private mode */ }
  }

  function go(p: Phase) {
    setPhase(p); setNetOpen(false); setTried(false); setErr('');
    const el = cardRef.current;
    if (el && el.getBoundingClientRect().top < 0) window.scrollTo({ top: el.getBoundingClientRect().top + window.scrollY - 16, behavior: 'smooth' });
  }

  async function pay() {
    if (busy) return;
    if (!amountOk || (isWallet && !ph.valid) || (!isWallet && !cardValid)) { setTried(true); return; }
    setBusy(true); setErr('');
    try {
      const body: Record<string, unknown> = { amount: total * 100, channel: method, name: name.trim(), net: isHandle ? net : '', message: msg.trim() };
      if (isWallet) body.phone = phone;
      else Object.assign(body, { card_brand: brand, card_last4: card.slice(-4), card_token: 'mock_tok_' + uid().replace(/-/g, '') });
      // The prompt can cut the data before the answer arrives. Asking again with the
      // same key returns the same payment, and never sends a second prompt.
      let s: SupportStatus | null = null;
      for (let tries = 0; !s; tries++) {
        try {
          s = await api.post<SupportStatus>(`/pages/${page.handle}/supports`, body, { 'Idempotency-Key': idem.current });
        } catch (e) {
          if (e instanceof ApiError || tries >= 45) throw e;
          setReconnecting(true);
          await new Promise(r => setTimeout(r, 2000));
        }
      }
      setReconnecting(false);
      idem.current = uid();
      if (isWallet) remember(s.id);
      setSup(s); setOtp('');
      go(isWallet ? 'wait' : 'otp');
    } catch (e) { setReconnecting(false); setErr(e instanceof ApiError ? e.message : 'Sem ligação. Tenta de novo.'); }
    finally { setBusy(false); }
  }

  async function confirmOtp() {
    if (!sup || otp.length !== 6 || busy) return;
    setBusy(true); setErr('');
    try {
      const s = await api.post<SupportStatus>(`/supports/${sup.id}/otp`, { code: otp });
      setSup(s);
      if (s.status === 'confirmed') go('done'); else setErr(s.fail_reason || 'O banco recusou o pagamento.');
    } catch (e) { setErr(e instanceof ApiError ? e.message : 'Sem ligação. Tenta de novo.'); }
    finally { setBusy(false); }
  }

  async function cancel() {
    if (sup && sup.status === 'pending') api.post(`/supports/${sup.id}/cancel`).catch(() => {});
    remember(null); setReconnecting(false); setSup(null); go('form');
  }

  async function resend() {
    if (!sup) return;
    try { const s = await api.post<SupportStatus>(`/supports/${sup.id}/resend`); setSup(s); setNow(Date.now()); }
    catch (e) { setErr(e instanceof ApiError ? e.message : 'Não foi possível reenviar.'); }
  }

  async function simulate() {
    if (!sup?.simulate) return;
    try { const s = await api.post<SupportStatus>(`/supports/${sup.id}/simulate`, { success: true }); setSup(s); if (s.status === 'confirmed') go('done'); } catch { /* ignore */ }
  }

  const secs = sup ? Math.max(0, Math.round((new Date(sup.expires_at).getTime() - now) / 1000)) : 60;
  const failed = sup && (sup.status === 'failed' || sup.status === 'expired');
  const first = isHandle ? name : name.trim().split(/\s+/)[0];

  return (
    <section ref={cardRef} style={{ background: '#fff', border: '1px solid #E4DDD1', borderRadius: 28, padding: 'clamp(18px,4vw,28px)', boxShadow: '0 30px 60px -36px rgba(26,20,16,.3)', scrollMarginTop: 16 }}>
      {phase === 'form' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 18 }}>
          <div className="display" style={{ fontWeight: 700, fontSize: 20, lineHeight: 1.15, letterSpacing: '-0.025em' }}>Apoia o {page.name}</div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2,minmax(0,1fr))', gap: 8 }}>
            {presets.map((v, i) => {
              const on = !custom && amt === v;
              return (
                <button key={i} type="button" onClick={() => { setAmt(v); setCustom(''); setTried(false); }}
                  style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8, height: 62, padding: '0 14px', borderRadius: 16, border: `1.5px solid ${on ? '#D14400' : '#EFE7DC'}`, background: on ? '#D14400' : '#FBF7F1', color: on ? '#fff' : '#1A1410', transition: 'background-color .2s,border-color .2s,color .2s' }}>
                  <span style={{ display: 'flex', alignItems: 'baseline', gap: 5 }}>
                    <span className="display" style={{ fontWeight: 800, fontSize: 22, lineHeight: 1, letterSpacing: '-0.03em' }}>{fmt(v)}</span>
                    <span className="mono" style={{ fontSize: 12, color: on ? 'rgba(255,255,255,.8)' : '#6E6861' }}>MT</span>
                  </span>
                  {i === page.default_idx && <span style={{ fontSize: 10.5, fontWeight: 600, padding: '3px 6px', borderRadius: 6, background: on ? 'rgba(255,255,255,.2)' : '#fff', color: on ? '#fff' : '#57524B' }}>Sugerido</span>}
                </button>
              );
            })}
          </div>
          {page.allow_custom && (
            <label style={{ display: 'flex', alignItems: 'center', height: 50, marginTop: -8, borderRadius: 14, border: `1.5px solid ${custom ? '#D14400' : '#EFE7DC'}`, background: '#FBF7F1', overflow: 'hidden' }}>
              <span style={{ paddingLeft: 14, fontSize: 14, color: '#57524B', whiteSpace: 'nowrap' }}>Outro valor</span>
              <input type="tel" inputMode="numeric" value={custom ? fmt(parseInt(custom, 10) || 0) : ''} onChange={e => { setCustom(e.target.value.replace(/\D/g, '').slice(0, 6)); setTried(false); }} placeholder="0" aria-label="Outro valor"
                className="display" style={{ flex: 1, minWidth: 0, height: '100%', border: 0, outline: 'none', background: 'transparent', textAlign: 'right', padding: '0 8px', fontWeight: 700, fontSize: 17 }} />
              <span className="mono" style={{ paddingRight: 14, fontSize: 12.5, color: '#6E6861' }}>MT</span>
            </label>
          )}
          {tried && !amountOk && <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: -10, fontSize: 13, color: RED }}><CircleAlert size={14} />O valor mínimo é {fmt(min)} MT.</div>}

          <div style={{ display: 'flex', flexDirection: 'column', gap: 10, paddingTop: 4, borderTop: '1px dashed #E4DDD1' }}>
            <span style={{ fontSize: 13, fontWeight: 600, paddingTop: 12 }}>Pagar com</span>
            <div style={{ display: 'grid', gridTemplateColumns: `repeat(${methods.length + soon.length},minmax(0,1fr))`, gap: 8 }}>
              {methods.map(k => {
                const on = method === k;
                const [label, letter, bb, bf] = ({ mpesa: ['M-Pesa', 'M', '#D14400', '#fff'], emola: ['e-Mola', 'e', '#1A1410', '#fff'], card: ['Cartão', '', '#ECE6DB', '#1A1410'] } as const)[k];
                return (
                  <button key={k} type="button" onClick={() => { setMethod(k); setTried(false); }} aria-pressed={on}
                    style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 6, height: 72, borderRadius: 16, border: `1.5px solid ${on ? '#D14400' : '#E4DDD1'}`, background: on ? '#FFF4EC' : '#fff', transition: 'border-color .2s,background-color .2s' }}>
                    <span className="display" style={{ width: 30, height: 30, borderRadius: 9, background: bb, color: bf, display: 'grid', placeItems: 'center', fontWeight: 900, fontSize: 13 }}>{letter}{k === 'card' && <CreditCard size={15} />}</span>
                    <span style={{ fontSize: 13, fontWeight: 600 }}>{label}</span>
                  </button>
                );
              })}
              {soon.map(k => {
                const [label, letter, bb, bf] = ({ mpesa: ['M-Pesa', 'M', '#D14400', '#fff'], emola: ['e-Mola', 'e', '#1A1410', '#fff'], card: ['Cartão', '', '#ECE6DB', '#1A1410'] } as const)[k];
                return (
                  <div key={k} aria-disabled="true" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 4, height: 72, borderRadius: 16, border: '1.5px dashed #E4DDD1', background: '#FBF7F1', color: '#6E6861' }}>
                    <span className="display" style={{ width: 24, height: 24, borderRadius: 7, background: bb, color: bf, opacity: 0.45, display: 'grid', placeItems: 'center', fontWeight: 900, fontSize: 11 }}>{letter}{k === 'card' && <CreditCard size={13} />}</span>
                    <span style={{ fontSize: 12, fontWeight: 600 }}>{label}</span>
                    <span style={{ fontSize: 10.5, fontWeight: 600, color: '#A83700' }}>Em breve</span>
                  </div>
                );
              })}
            </div>
            {isWallet ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <div style={{ display: 'flex', alignItems: 'center', height: 52, borderRadius: 14, border: `1.5px solid ${ph.color === RED ? RED : ph.valid ? '#1A1410' : '#E4DDD1'}`, background: '#fff', overflow: 'hidden', transition: 'border-color .2s' }}>
                  <span className="mono" style={{ padding: '0 12px', height: '100%', display: 'flex', alignItems: 'center', borderRight: '1px solid #EFE7DC', fontSize: 14, color: '#57524B', background: '#FBF7F1' }}>+258</span>
                  <input type="tel" inputMode="numeric" autoComplete="tel-national" value={fmtPhone(phone)} onChange={e => setPhone(digits(e.target.value, 9))} placeholder={ch === 'mpesa' ? '84 123 4567' : '86 123 4567'} aria-label={'Número ' + CHN[ch]}
                    className="mono" style={{ flex: 1, minWidth: 0, height: '100%', border: 0, outline: 'none', padding: '0 12px', fontSize: 16, background: 'transparent' }} />
                </div>
                <span style={{ display: 'flex', alignItems: 'flex-start', flexWrap: 'wrap', gap: 6, fontSize: 12.5, lineHeight: 1.4, color: ph.color }}>
                  {ph.icon === 'ok' ? <CircleCheck size={14} style={{ marginTop: 1 }} /> : ph.icon === 'alert' ? <CircleAlert size={14} style={{ marginTop: 1 }} /> : <Info size={14} style={{ marginTop: 1 }} />}
                  <span>{ph.hint}</span>
                  {ph.canSwitch && op && op !== 'tmcel' && <button type="button" onClick={() => setMethod(op)} style={{ border: 0, background: 'transparent', padding: 0, fontWeight: 600, textDecoration: 'underline', color: 'inherit' }}>{op === 'mpesa' ? 'Usar M-Pesa' : 'Usar e-Mola'}</button>}
                </span>
              </div>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                <div style={{ display: 'flex', alignItems: 'center', height: 52, borderRadius: 14, border: `1.5px solid ${tried && !cardValid ? RED : cardValid ? '#1A1410' : '#E4DDD1'}`, background: '#fff', overflow: 'hidden' }}>
                  <input type="tel" inputMode="numeric" autoComplete="cc-number" value={card.replace(/(\d{4})(?=\d)/g, '$1 ')} onChange={e => setCard(e.target.value.replace(/\D/g, '').slice(0, 16))} placeholder="Número do cartão" aria-label="Número do cartão"
                    className="mono" style={{ flex: 1, minWidth: 0, height: '100%', border: 0, outline: 'none', padding: '0 14px', fontSize: 15, letterSpacing: '.03em', background: 'transparent' }} />
                  <span style={{ marginRight: 12, display: 'flex', gap: 6 }}>
                    <Brand name="visa" size={22} color="#1A1F71" style={{ opacity: brand === 'mastercard' ? 0.3 : 1 }} />
                    <Brand name="mastercard" size={18} color="#EB001B" style={{ opacity: brand === 'visa' ? 0.3 : 1 }} />
                  </span>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                  <input type="tel" inputMode="numeric" autoComplete="cc-exp" value={exp.length > 2 ? exp.slice(0, 2) + '/' + exp.slice(2) : exp} onChange={e => setExp(e.target.value.replace(/\D/g, '').slice(0, 4))} placeholder="MM/AA" aria-label="Validade"
                    className="mono input" style={{ height: 50, borderRadius: 14, border: '1.5px solid #E4DDD1', fontSize: 15 }} />
                  <input type="password" inputMode="numeric" autoComplete="cc-csc" value={cvv} onChange={e => setCvv(e.target.value.replace(/\D/g, '').slice(0, 3))} placeholder="CVV" aria-label="CVV"
                    className="mono input" style={{ height: 50, borderRadius: 14, border: '1.5px solid #E4DDD1', fontSize: 15 }} />
                </div>
                {tried && !cardValid && <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12.5, color: RED }}><CircleAlert size={14} />Confirma o número, a validade e o CVV.</div>}
              </div>
            )}
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 10, borderTop: '1px dashed #E4DDD1' }}>
            <span style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', paddingTop: 14, fontSize: 13, fontWeight: 600 }}>Deixa o teu nome <span style={{ fontWeight: 400, color: '#6E6861' }}>Opcional</span></span>
            <div style={{ position: 'relative' }}>
              <div style={{ display: 'flex', alignItems: 'center', height: 50, borderRadius: 14, border: `1.5px solid ${nameFocus || netOpen ? '#1A1410' : '#E4DDD1'}`, background: '#fff', transition: 'border-color .2s' }}>
                {isHandle && <button type="button" onClick={() => setNetOpen(o => !o)} aria-label="Rede social" style={{ flex: 'none', display: 'flex', alignItems: 'center', gap: 4, height: '100%', padding: '0 4px 0 14px', border: 0, background: 'transparent' }}>
                  <Brand name={NETS.find(n => n[0] === net)![2]} size={15} /><ChevronDown size={13} color="#6E6861" />
                </button>}
                <input ref={nameRef} value={name} onChange={e => { const v = e.target.value.slice(0, 40); setName(v); setNetOpen(v === '@' ? true : v.startsWith('@') ? netOpen : false); }}
                  onFocus={() => setNameFocus(true)} onBlur={() => setNameFocus(false)} placeholder="Nome ou @rede_social" aria-label="Nome ou rede social"
                  style={{ flex: 1, minWidth: 0, height: '100%', border: 0, outline: 'none', background: 'transparent', padding: `0 14px 0 ${isHandle ? '4px' : '14px'}`, fontSize: 15 }} />
              </div>
              {netOpen && isHandle && (
                <div style={{ position: 'absolute', left: 0, top: 'calc(100% + 6px)', zIndex: 20, width: 210, padding: 6, borderRadius: 16, background: '#fff', border: '1px solid #E4DDD1', boxShadow: '0 20px 40px -16px rgba(26,20,16,.3)' }}>
                  {NETS.map(([k, label, icon]) => (
                    <button key={k} type="button" className="hover-paper" onClick={() => { setNet(k); setNetOpen(false); requestAnimationFrame(() => { const el = nameRef.current; if (el) { el.focus(); el.setSelectionRange(el.value.length, el.value.length); } }); }}
                      style={{ width: '100%', display: 'flex', alignItems: 'center', gap: 10, height: 40, padding: '0 10px', border: 0, borderRadius: 11, background: net === k ? '#FBF7F1' : 'transparent', textAlign: 'left', fontSize: 14, fontWeight: 500 }}>
                      <Brand name={icon} size={15} /><span style={{ flex: 1 }}>{label}</span>{net === k && <Check size={15} color="#D14400" />}
                    </button>
                  ))}
                </div>
              )}
            </div>
            <div style={{ position: 'relative' }}>
              <textarea value={msg} onChange={e => setMsg(e.target.value.slice(0, 280))} rows={2} placeholder="Mensagem (opcional)" aria-label="Mensagem"
                className="input" style={{ display: 'block', padding: '13px 14px 22px', borderRadius: 14, border: '1.5px solid #E4DDD1', fontSize: 15, resize: 'none' }} />
              <span className="mono" style={{ position: 'absolute', right: 12, bottom: 6, fontSize: 11, color: '#9A938A' }}>{msg.length}/280</span>
            </div>
          </div>
          {err && <div className="err-box"><CircleAlert size={15} />{err}</div>}
          <button type="button" onClick={pay} disabled={busy} className="btn btn-primary" style={{ height: 58, borderRadius: 16, fontSize: 16.5, gap: 10 }}>
            {busy ? <span className="spinner" /> : <Lock size={16} />}{reconnecting ? 'A aguardar ligação…' : amountOk && total > 0 ? `Pagar ${fmt(total)} MT` : 'Pagar'}
          </button>
          <div style={{ marginTop: -8, textAlign: 'center', fontSize: 12.5, lineHeight: 1.45, color: '#6E6861' }}>
            {isWallet ? 'Confirmas com o PIN no teu telemóvel. Nunca te pedimos o PIN aqui.' : 'Visa e Mastercard. O teu banco pode pedir um código de confirmação.'}
          </div>
          <div style={{ marginTop: -10, textAlign: 'center', fontSize: 12, lineHeight: 1.45, color: '#8A837A' }}>{isWallet ? `${page.name} vê o valor, o teu nome, a mensagem e o teu número, para te poder agradecer.` : `${page.name} vê o valor, o teu nome e a mensagem. Não partilhamos os dados do cartão.`}</div>
        </div>
      )}

      {phase === 'wait' && sup && (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', gap: 18 }}>
          <button type="button" onClick={simulate} title={sup.simulate ? 'Simular PIN confirmado (sandbox)' : undefined} aria-label="Telemóvel" style={{ position: 'relative', width: 150, height: 196, border: 0, padding: 0, margin: '14px 0 4px', background: 'transparent', cursor: sup.simulate ? 'pointer' : 'default' }}>
            {!failed && <span style={{ position: 'absolute', inset: 0, pointerEvents: 'none' }}>
              <span style={{ position: 'absolute', inset: -12, borderRadius: 38, border: '2px solid #D14400', animation: 'rfPulse 2s ease-out infinite' }} />
              <span style={{ position: 'absolute', inset: -12, borderRadius: 38, border: '2px solid #D14400', animation: 'rfPulse 2s 1s ease-out infinite' }} />
            </span>}
            <span style={{ position: 'absolute', inset: 0, borderRadius: 28, background: '#1A1410', padding: 8, display: 'flex' }}>
              <span style={{ flex: 1, borderRadius: 21, background: '#2E2721', display: 'flex', flexDirection: 'column', justifyContent: 'center', padding: 9 }}>
                <span style={{ background: '#fff', borderRadius: 12, padding: 10, textAlign: 'left', display: 'flex', flexDirection: 'column', gap: 7 }}>
                  <span className="mono" style={{ fontSize: 9.5, letterSpacing: '.06em', color: '#6E6861', textTransform: 'uppercase' }}>{CHN[sup.channel as 'mpesa']}</span>
                  <span style={{ fontSize: 10.5, lineHeight: 1.35 }}>Pagar <b>{mtN(sup.amount, 2)} MT</b> a REFRES*{page.name.toUpperCase()}. Insere o PIN:</span>
                  <span className="mono" style={{ height: 22, borderRadius: 6, border: '1px solid #D9D1C4', display: 'flex', alignItems: 'center', padding: '0 7px', letterSpacing: '.3em', fontSize: 11 }}>••••</span>
                </span>
              </span>
            </span>
          </button>
          <div>
            <div className="display" style={{ fontWeight: 700, fontSize: 22, lineHeight: 1.12, letterSpacing: '-0.025em' }}>{failed ? 'O pagamento não foi concluído' : 'Confirma no teu telemóvel'}</div>
            <p style={{ margin: '8px auto 0', maxWidth: 320, fontSize: 14.5, color: '#57524B' }}>
              {failed ? <>{sup.fail_reason || 'O pedido expirou'}. Podes reenviar o pedido ou voltar.</> :
                <>Enviámos um pedido de <b style={{ color: '#1A1410' }}>{mtN(sup.amount)} MT</b> para <span className="mono" style={{ color: '#1A1410', whiteSpace: 'nowrap' }}>+258 {fmtPhone(phone)}</span>. Insere o teu PIN {CHN[sup.channel as 'mpesa']}.</>}
            </p>
          </div>
          {reconnecting && <div role="status" style={{ width: '100%', padding: '11px 14px', borderRadius: 12, background: '#FBEFD9', color: '#5B4600', fontSize: 13.5, lineHeight: 1.45 }}>O M-Pesa pode cortar a internet enquanto o pedido está aberto. Não precisas de fazer nada: o pagamento continua e atualizamos assim que a ligação voltar.</div>}
          {!failed && <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 8 }}>
            <div style={{ height: 4, borderRadius: 2, background: '#F2EAE0', overflow: 'hidden' }}><div style={{ height: '100%', width: `${Math.min(100, (secs / 120) * 100)}%`, background: '#D14400', transition: 'width 1s linear' }} /></div>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, color: '#6E6861' }}>
              <span>{secs > 0 ? 'O pedido expira em' : 'O pedido expirou. Toca em Reenviar.'}</span>
              <span className="mono" style={{ color: '#1A1410' }}>{Math.floor(secs / 60)}:{String(secs % 60).padStart(2, '0')}</span>
            </div>
          </div>}
          {err && <div className="err-box" style={{ width: '100%' }}><CircleAlert size={15} />{err}</div>}
          <div style={{ width: '100%', display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
            <button type="button" className="btn" style={{ height: 46, borderRadius: 14, borderColor: '#E4DDD1' }} onClick={cancel}>Cancelar</button>
            <button type="button" className="btn" style={{ height: 46, borderRadius: 14, borderColor: '#E4DDD1' }} onClick={resend}><RotateCw size={14} />Reenviar</button>
          </div>
        </div>
      )}

      {phase === 'otp' && sup && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ display: 'flex', alignItems: 'center', gap: 8, fontWeight: 600 }}><Landmark size={17} />O teu banco</span>
            <span className="mono" style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11, letterSpacing: '.06em', color: '#57524B' }}><Lock size={12} />3-D SECURE</span>
          </div>
          <div style={{ borderRadius: 16, background: '#FBF7F1', padding: 14, display: 'flex', flexDirection: 'column', gap: 8, fontSize: 13.5 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}><span style={{ color: '#6E6861' }}>Comerciante</span><span style={{ fontWeight: 600 }}>REFRES*{page.name.toUpperCase()}</span></div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}><span style={{ color: '#6E6861' }}>Valor</span><span style={{ fontWeight: 600 }}>{mtN(sup.amount, 2)} MT</span></div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}><span style={{ color: '#6E6861' }}>Cartão</span><span className="mono">{brandName} ···· {card.slice(-4)}</span></div>
          </div>
          <label htmlFor="otp" style={{ fontSize: 14, color: '#2B2723' }}>Introduz o código que o teu banco enviou por SMS.</label>
          <div style={{ position: 'relative' }}>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6,1fr)', gap: 6 }}>
              {Array.from({ length: 6 }, (_, i) => (
                <span key={i} className="display" style={{ height: 48, borderRadius: 12, border: `1.5px solid ${i < otp.length ? '#1A1410' : i === otp.length ? '#D14400' : '#E4DDD1'}`, display: 'grid', placeItems: 'center', fontWeight: 800, fontSize: 19, transition: 'border-color .2s' }}>{otp[i] || ''}</span>
              ))}
            </div>
            <input id="otp" type="tel" inputMode="numeric" autoComplete="one-time-code" autoFocus value={otp} onChange={e => setOtp(e.target.value.replace(/\D/g, '').slice(0, 6))}
              style={{ position: 'absolute', inset: 0, opacity: 0, width: '100%', height: '100%', border: 0 }} />
          </div>
          {err && <div className="err-box"><CircleAlert size={15} />{err}</div>}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.4fr', gap: 8 }}>
            <button type="button" className="btn" style={{ height: 50, borderRadius: 14, borderColor: '#E4DDD1' }} onClick={cancel}>Cancelar</button>
            <button type="button" className="btn btn-dark" style={{ height: 50, borderRadius: 14, fontSize: 14.5 }} onClick={confirmOtp} disabled={otp.length !== 6 || busy}>{busy ? 'A confirmar…' : 'Confirmar'}</button>
          </div>
        </div>
      )}

      {phase === 'done' && sup && (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', gap: 18, paddingTop: 4 }}>
          <div style={{ width: 68, height: 68, borderRadius: 22, background: '#D14400', display: 'grid', placeItems: 'center' }}>
            <svg width={34} height={34} viewBox="0 0 24 24" fill="none"><path d="M5 12.5l4.5 4.5L19 7.5" stroke="#fff" strokeWidth={2.6} strokeLinecap="round" strokeLinejoin="round" style={{ strokeDasharray: 24, strokeDashoffset: 24, animation: 'rfDraw .5s .2s ease-out forwards' }} /></svg>
          </div>
          <div>
            <div className="mono" style={{ fontSize: 11.5, letterSpacing: '.08em', textTransform: 'uppercase', color: '#17803D' }}>Pagamento confirmado</div>
            <div className="display" style={{ marginTop: 8, fontWeight: 800, fontSize: 28, lineHeight: 1.05, letterSpacing: '-0.035em', overflowWrap: 'anywhere' }}>Obrigado{first ? ', ' + first : ''}!</div>
            <div style={{ marginTop: 6, fontSize: 14.5, color: '#57524B' }}>Pagaste <b style={{ color: '#1A1410' }}>{mtN(sup.amount)} MT</b> ao {page.name}.</div>
          </div>
          {page.thanks && <div style={{ width: '100%', textAlign: 'left', padding: '14px 16px', borderRadius: 16, background: '#FBF7F1', fontSize: 14.5, lineHeight: 1.5, color: '#2B2723' }}>
            <span style={{ display: 'block', fontSize: 12.5, fontWeight: 600, color: '#1A1410', marginBottom: 2 }}>{page.name}</span>{page.thanks}
          </div>}
          <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 7, fontSize: 13 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12 }}><span style={{ color: '#6E6861' }}>Referência</span><span className="mono">{sup.ref}</span></div>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12 }}><span style={{ color: '#6E6861' }}>Pagamento</span><span className="mono">{sup.channel === 'card' ? `${brandName} ···· ${card.slice(-4)}` : `${CHN[sup.channel as 'mpesa']} · ${fmtPhone(phone)}`}</span></div>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12 }}><span style={{ color: '#6E6861' }}>Data</span><span className="mono">{dateTime(sup.confirmed_at || sup.created_at)}</span></div>
          </div>
          <button type="button" className="btn" style={{ width: '100%', height: 50, borderRadius: 16, fontSize: 15 }}
            onClick={() => { setAmt(def); setCustom(''); setMsg(''); setCard(''); setExp(''); setCvv(''); setSup(null); go('form'); }}>Fazer outro apoio</button>
        </div>
      )}
    </section>
  );
}
