import { useEffect, useState } from 'react';
import { ArrowDownToLine, CircleAlert, CircleCheck, Clock3, Landmark, Plus, Smartphone, Trash2, Wallet } from 'lucide-react';
import { api, ApiError, WithdrawalView } from '../../lib/api';
import { digits, fmt, fmtPhone, ini, MOL, monthOf, mt, mtN, SEG, shortDateTime, walletOf, WD_ST } from '../../lib/format';
import { Badge, Modal, ModalHead, Seg, useToast } from '../../components/ui';
import { PageHead } from './Layout';

interface Acct { id: string; kind: 'mpesa' | 'emola' | 'nib'; number: string; bank?: string; holder?: string; is_main: boolean }
interface WalletV {
  member_id: string; balance: number; processing: number; processing_count: number; paid_month: number; paid_month_count: number; min_withdrawal: number;
  accounts: Acct[]; members: { id: string; name: string; pct: number; balance: number; last_paid?: number; last_when?: string }[];
  withdrawals: WithdrawalView[]; account_status: string;
}

const KIND = { mpesa: ['M-Pesa', '#D14400', '#FFFFFF'], emola: ['e-Mola', '#1A1410', '#FFFFFF'], nib: ['Conta bancária', '#ECE6DB', '#1A1410'] } as const;
const BANKS = ['BCI', 'Millennium bim', 'Standard Bank', 'Absa', 'Moza Banco', 'FNB', 'Nedbank', 'Access Bank', 'Ecobank', 'Letshego'];

export default function Payouts() {
  const toast = useToast();
  const [w, setW] = useState<WalletV | null>(null);
  const [open, setOpen] = useState(false);
  const [adding, setAdding] = useState(false);
  const [mF, setMF] = useState('all');
  const [sF, setSF] = useState('all');
  const load = () => api.get<WalletV>('/backoffice/wallet').then(setW).catch(() => {});
  useEffect(() => { load(); }, []);
  if (!w) return <div className="skeleton" style={{ height: 400 }} />;

  const idx = (id: string) => Math.max(0, w.members.findIndex(m => m.id === id));
  const rows = w.withdrawals.filter(x => (mF === 'all' || x.member_id === mF) && (sF === 'all' || x.status === sF || (sF === 'processing' && (x.status === 'requested' || x.status === 'approved'))));
  const monthName = MOL[monthOf(new Date().toISOString())];
  const destLabel = (x: WithdrawalView) => x.dest_kind === 'nib' ? `NIB · ${x.dest_bank}` : KIND[x.dest_kind][0];

  async function del(a: Acct) {
    try { await api.del('/backoffice/wallet/accounts/' + a.id); toast('Conta removida'); load(); }
    catch (e) { toast(e instanceof ApiError ? e.message : 'Algo correu mal.', true); }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <PageHead kicker="Saldo e levantamentos" title="Levantamentos" lead="Cada apoio entra no saldo de cada membro, já com a divisão feita. Pede o levantamento para o M-Pesa ou para a tua conta bancária. Cada pedido é verificado e pago em até 72 horas." />
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
        <div className="card-dark" style={{ flex: '1.5 1 300px', minWidth: 0, padding: 22, display: 'flex', flexDirection: 'column', gap: 20 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 14, color: '#A39C92' }}><span>O teu saldo disponível</span><Wallet size={17} /></div>
          <div className="display" style={{ display: 'flex', alignItems: 'baseline', flexWrap: 'wrap', gap: '4px 10px', fontWeight: 800, letterSpacing: '-0.04em' }}><span style={{ fontSize: 'clamp(34px,3.6vw,48px)', lineHeight: 1 }}>{mtN(w.balance, 2)}</span><span style={{ fontSize: 18, color: '#A39C92' }}>MT</span></div>
          <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 12 }}>
            <button className="btn btn-primary" style={{ height: 46, padding: '0 18px', borderRadius: 13, fontSize: 14.5 }} onClick={() => setOpen(true)} disabled={!w.accounts.length || w.account_status === 'suspended'}><ArrowDownToLine size={17} />Pedir levantamento</button>
            <span style={{ fontSize: 13, color: '#A39C92' }}>Mínimo {mt(w.min_withdrawal)} · pago em até 72 h</span>
          </div>
        </div>
        <Stat label="Em processamento" Icon={Clock3} value={mtN(w.processing, 2)} note={w.processing_count ? `${w.processing_count} ${w.processing_count === 1 ? 'pedido' : 'pedidos'}` : 'Nenhum pedido pendente'} />
        <Stat label={`Levantado em ${monthName}`} Icon={CircleCheck} value={mtN(w.paid_month, 2)} note={`${w.paid_month_count} ${w.paid_month_count === 1 ? 'levantamento' : 'levantamentos'}`} />
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, alignItems: 'flex-start' }}>
        <section className="card" style={{ flex: '1 1 320px', minWidth: 0, padding: 'clamp(16px,2vw,22px)', display: 'flex', flexDirection: 'column', gap: 6 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, marginBottom: 6 }}><span style={{ fontWeight: 600, fontSize: 15 }}>As tuas contas</span><button className="btn" style={{ height: 36, padding: '0 12px', borderRadius: 11, fontSize: 13 }} onClick={() => setAdding(true)}><Plus size={14} />Adicionar</button></div>
          {w.accounts.map(a => (
            <div key={a.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '12px 0', borderTop: '1px solid #F0EBE3' }}>
              <span style={{ flex: 'none', width: 40, height: 40, borderRadius: 12, background: KIND[a.kind][1], color: KIND[a.kind][2], display: 'grid', placeItems: 'center' }}>{a.kind === 'nib' ? <Landmark size={18} /> : <Smartphone size={18} />}</span>
              <span style={{ flex: 1, minWidth: 0, lineHeight: 1.35 }}>
                <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}><span style={{ fontWeight: 600, whiteSpace: 'nowrap' }}>{KIND[a.kind][0]}{a.bank ? ' · ' + a.bank : ''}</span>{a.is_main && <span style={{ height: 20, padding: '0 6px', borderRadius: 6, background: '#FDE9DC', color: '#A83700', fontSize: 11, fontWeight: 600, display: 'inline-flex', alignItems: 'center' }}>Principal</span>}</span>
                <span className="mono" style={{ display: 'block', fontSize: 12.5, color: '#2B2723' }}>{a.number}</span>
                {a.holder && <span style={{ display: 'block', fontSize: 12, color: '#6E6861' }}>Titular: {a.holder}</span>}
              </span>
              <span style={{ flex: 'none', fontSize: 12.5, color: '#6E6861' }}>Até 72 h</span>
              {!a.is_main && <button onClick={() => del(a)} aria-label="Remover conta" style={{ flex: 'none', width: 32, height: 32, border: 0, borderRadius: 9, background: 'transparent', color: '#6E6861', display: 'grid', placeItems: 'center' }}><Trash2 size={15} /></button>}
            </div>
          ))}
        </section>
        <section className="card" style={{ flex: '1.4 1 420px', minWidth: 0, padding: 'clamp(16px,2vw,22px)', display: 'flex', flexDirection: 'column', gap: 6 }}>
          <div style={{ fontWeight: 600, fontSize: 15, marginBottom: 6 }}>Saldos do grupo</div>
          {w.members.map((m, i) => (
            <div key={m.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '12px 0', borderTop: '1px solid #F0EBE3' }}>
              <span className="avatar" style={{ width: 38, height: 38, borderRadius: 12, background: SEG[i % SEG.length][0], color: SEG[i % SEG.length][1], fontSize: 12.5 }}>{ini(m.name)}</span>
              <span style={{ flex: 1, minWidth: 0, lineHeight: 1.35 }}>
                <span style={{ display: 'block', fontWeight: 600 }}>{m.name} <span style={{ fontWeight: 400, color: '#6E6861' }}>· {m.pct}%</span></span>
                <span style={{ display: 'block', fontSize: 12.5, color: '#6E6861', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{m.last_paid && m.last_when ? `Último levantamento: ${mt(m.last_paid)} · ${shortDateTime(m.last_when)}` : 'Ainda sem levantamentos'}</span>
              </span>
              <span className="display" style={{ fontWeight: 700, fontSize: 15, whiteSpace: 'nowrap' }}>{mt(m.balance, 2)}</span>
            </div>
          ))}
        </section>
      </div>
      <div className="card" style={{ overflow: 'hidden' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12, padding: '16px clamp(14px,2vw,20px)' }}>
          <span style={{ fontWeight: 600, fontSize: 15 }}>Pedidos de levantamento</span>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', maxWidth: '100%' }}>
            {w.members.length > 1 && <Seg small value={mF} onChange={setMF} options={[['all', 'Todos'], ...w.members.map(m => [m.id, m.name] as [string, string])]} />}
            <Seg small value={sF} onChange={setSF} options={[['all', 'Todos'], ['processing', 'Em processamento'], ['paid', 'Pagos'], ['rejected', 'Rejeitados']]} />
          </div>
        </div>
        {rows.map(x => {
          const i = idx(x.member_id), open = x.status === 'requested' || x.status === 'approved';
          return (
            <div key={x.id} style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: '8px 14px', padding: '12px clamp(14px,2vw,20px)', borderTop: '1px solid #F0EBE3' }}>
              <span className="avatar" style={{ width: 34, height: 34, borderRadius: 11, background: SEG[i % SEG.length][0], color: SEG[i % SEG.length][1], fontSize: 11.5 }}>{ini(x.member_name)}</span>
              <span style={{ flex: '1 1 180px', minWidth: 0, lineHeight: 1.35 }}>
                <span style={{ display: 'block', fontWeight: 600 }}>{x.member_name} <span style={{ fontWeight: 400, color: '#6E6861' }}>→ {destLabel(x)}</span></span>
                <span style={{ display: 'block', fontSize: 12.5, color: '#6E6861' }}>{x.code} · {shortDateTime(x.created_at)}{x.reason ? ' · ' + x.reason : open ? ' · pago até ' + shortDateTime(x.due_at) : x.transfer_ref ? ' · ' + x.transfer_ref : ''}</span>
              </span>
              <span className="mono" style={{ fontSize: 13.5, whiteSpace: 'nowrap' }}>{mt(x.amount, 2)}</span>
              <span style={{ flex: 'none', width: 150, display: 'flex', justifyContent: 'flex-end' }}><Badge st={x.status === 'requested' ? ['Em processamento', '#FBEFD9', '#9A5B00'] : WD_ST[x.status]} /></span>
            </div>
          );
        })}
        {rows.length === 0 && <div style={{ padding: '48px 20px', textAlign: 'center', color: '#6E6861', borderTop: '1px solid #F0EBE3' }}>Nenhum pedido com estes filtros.</div>}
      </div>
      {open && <WithdrawModal w={w} onClose={() => setOpen(false)} onDone={(n) => { setOpen(false); toast('Pedido de ' + mt(n) + ' enviado · processado em até 72 h'); load(); }} />}
      {adding && <AddAccount onClose={() => setAdding(false)} onDone={() => { setAdding(false); toast('Conta adicionada'); load(); }} />}
    </div>
  );
}

function Stat({ label, Icon, value, note }: { label: string; Icon: typeof Wallet; value: string; note: string }) {
  return (
    <div className="card" style={{ flex: '1 1 220px', minWidth: 0, padding: 22, display: 'flex', flexDirection: 'column', justifyContent: 'space-between', gap: 20 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 14, color: '#57524B' }}><span>{label}</span><Icon size={17} /></div>
      <div><div className="display" style={{ fontWeight: 800, fontSize: 30, letterSpacing: '-0.035em', lineHeight: 1 }}>{value} <span style={{ fontSize: 16, color: '#6E6861' }}>MT</span></div><div style={{ marginTop: 8, fontSize: 13, color: '#6E6861' }}>{note}</div></div>
    </div>
  );
}

function WithdrawModal({ w, onClose, onDone }: { w: WalletV; onClose: () => void; onDone: (n: number) => void }) {
  const [amt, setAmt] = useState(''); const [dest, setDest] = useState(w.accounts.find(a => a.is_main)?.id || w.accounts[0]?.id);
  const [tried, setTried] = useState(false); const [err, setErr] = useState(''); const [busy, setBusy] = useState(false);
  const n = (parseInt(amt, 10) || 0) * 100;
  const ok = n >= w.min_withdrawal && n <= w.balance;
  const vErr = !tried ? '' : n < w.min_withdrawal ? `O mínimo é ${mt(w.min_withdrawal)}.` : n > w.balance ? 'O valor é maior do que o teu saldo.' : '';
  async function submit() {
    if (!ok) { setTried(true); return; }
    setBusy(true); setErr('');
    try { await api.post('/backoffice/wallet/withdrawals', { amount: n, payout_account_id: dest }); onDone(n); }
    catch (e) { setErr(e instanceof ApiError ? e.message : 'Algo correu mal.'); } finally { setBusy(false); }
  }
  return (
    <Modal onClose={onClose} label="Pedir levantamento">
      <ModalHead title="Pedir levantamento" sub={<>Saldo disponível: <b style={{ color: '#1A1410' }}>{mtN(w.balance, 2)} MT</b></>} onClose={onClose} />
      <label style={{ display: 'flex', alignItems: 'center', height: 64, borderRadius: 16, border: `1.5px solid ${vErr ? '#B42318' : '#E4DDD1'}`, padding: '0 16px', gap: 8 }}>
        <input type="tel" inputMode="numeric" autoFocus value={amt ? fmt(parseInt(amt, 10)) : ''} onChange={e => { setAmt(e.target.value.replace(/\D/g, '').slice(0, 7)); setTried(false); }} placeholder="0" aria-label="Valor"
          className="display" style={{ flex: 1, minWidth: 0, border: 0, outline: 'none', background: 'transparent', fontWeight: 800, fontSize: 28, letterSpacing: '-0.03em' }} />
        <span className="mono" style={{ fontSize: 14, color: '#6E6861' }}>MT</span>
      </label>
      <div style={{ display: 'flex', gap: 6, marginTop: -6 }}>
        {([['25%', 0.25], ['50%', 0.5], ['Tudo', 1]] as const).map(([l, f]) => <button key={l} className="btn" style={{ height: 34, padding: '0 12px', borderRadius: 10, fontSize: 13, borderColor: '#E4DDD1' }} onClick={() => { setAmt(String(Math.floor((w.balance * f) / 100))); setTried(false); }}>{l}</button>)}
      </div>
      {vErr && <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: -6, fontSize: 13, color: '#B42318' }}><CircleAlert size={14} />{vErr}</div>}
      <span style={{ fontSize: 13, fontWeight: 600, marginBottom: -6 }}>Para onde?</span>
      {w.accounts.map(a => { const on = dest === a.id; return (
        <button key={a.id} onClick={() => setDest(a.id)} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '12px 14px', borderRadius: 16, border: `1.5px solid ${on ? '#D14400' : '#E4DDD1'}`, background: on ? '#FFF4EC' : '#fff', textAlign: 'left' }}>
          <span style={{ flex: 'none', width: 38, height: 38, borderRadius: 12, background: KIND[a.kind][1], color: KIND[a.kind][2], display: 'grid', placeItems: 'center' }}>{a.kind === 'nib' ? <Landmark size={17} /> : <Smartphone size={17} />}</span>
          <span style={{ flex: 1, minWidth: 0, lineHeight: 1.3 }}><span style={{ display: 'block', fontWeight: 600 }}>{KIND[a.kind][0]}{a.bank ? ' · ' + a.bank : ''}</span><span className="mono" style={{ display: 'block', fontSize: 12, color: '#57524B' }}>{a.number}</span></span>
          <span style={{ flex: 'none', fontSize: 12, color: '#6E6861' }}>Até 72 h</span>
          <span style={{ flex: 'none', width: 20, height: 20, borderRadius: '50%', border: `2px solid ${on ? '#D14400' : '#CFC6B8'}`, display: 'grid', placeItems: 'center' }}><span style={{ width: 9, height: 9, borderRadius: '50%', background: on ? '#D14400' : 'transparent' }} /></span>
        </button>); })}
      <div style={{ borderRadius: 14, background: '#F8F5EF', padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: 6, fontSize: 13.5 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between' }}><span style={{ color: '#6E6861' }}>Recebes</span><span className="mono">{mt(n, 2)}</span></div>
        <div style={{ display: 'flex', justifyContent: 'space-between' }}><span style={{ color: '#6E6861' }}>Chega</span><span>Até 72 h</span></div>
      </div>
      {err && <div className="err-box"><CircleAlert size={15} />{err}</div>}
      <button className="btn btn-primary" style={{ height: 52, borderRadius: 14, fontSize: 15, background: ok ? '#D14400' : '#CFC6B8' }} onClick={submit} disabled={busy}>Confirmar levantamento</button>
    </Modal>
  );
}

function AddAccount({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [kind, setKind] = useState<'mpesa' | 'emola' | 'nib'>('mpesa');
  const [num, setNum] = useState(''); const [bank, setBank] = useState(BANKS[0]); const [holder, setHolder] = useState(''); const [pw, setPw] = useState(''); const [main, setMain] = useState(false);
  const [err, setErr] = useState(''); const [busy, setBusy] = useState(false);
  const valid = kind === 'nib' ? num.length === 21 && holder.trim().length >= 3 : num.length === 9 && walletOf(num) === kind;
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!pw) { setErr('Escreve a tua palavra-passe.'); return; }
    if (!valid) { setErr(kind === 'nib' ? (num.length !== 21 ? 'O NIB tem 21 dígitos.' : 'Escreve o nome do titular da conta.') : 'O número não corresponde à carteira escolhida.'); return; }
    setBusy(true); setErr('');
    try { await api.post('/backoffice/wallet/accounts', { kind, number: num, bank: kind === 'nib' ? bank : '', holder: kind === 'nib' ? holder : '', is_main: main, password: pw }); onDone(); }
    catch (x) { setErr(x instanceof ApiError ? x.message : 'Algo correu mal.'); } finally { setBusy(false); }
  }
  return (
    <Modal onClose={onClose} label="Adicionar conta">
      <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <ModalHead title="Adicionar conta" sub="Para onde podes pedir levantamentos." onClose={onClose} />
        <Seg value={kind} onChange={k => { setKind(k); setNum(''); setErr(''); }} options={[['mpesa', 'M-Pesa'], ['emola', 'e-Mola'], ['nib', 'Conta bancária']]} />
        {kind === 'nib' ? <>
          <label className="field"><span>Banco</span><select className="input" value={bank} onChange={e => setBank(e.target.value)}>{BANKS.map(b => <option key={b}>{b}</option>)}</select></label>
          <label className="field"><span>NIB</span><input className="input mono" inputMode="numeric" value={num.replace(/(\d{4})(?=\d)/g, '$1 ')} onChange={e => setNum(e.target.value.replace(/\D/g, '').slice(0, 21))} placeholder="0008 0000 0412 3456 7810 1" /></label>
          <label className="field"><span>Titular da conta</span><input className="input" value={holder} onChange={e => setHolder(e.target.value.slice(0, 80))} placeholder="Nome como aparece no banco" autoComplete="name" /></label>
        </> : <label className="field"><span>Número {KIND[kind][0]}</span>
          <span className="affix"><span className="pre">+258</span><input type="tel" className="mono" value={fmtPhone(num)} onChange={e => setNum(digits(e.target.value, 9))} placeholder={kind === 'mpesa' ? '84 123 4567' : '86 123 4567'} /></span></label>}
        <label className="field"><span>Confirma com a tua palavra-passe</span><input className="input" type="password" autoComplete="current-password" value={pw} onChange={e => setPw(e.target.value)} /></label>
        <div style={{ fontSize: 12.5, color: '#6E6861', marginTop: -8 }}>Por segurança, avisamos-te por SMS e email sempre que uma conta nova é adicionada.</div>
        <label style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 13.5 }}><input type="checkbox" checked={main} onChange={e => setMain(e.target.checked)} />Usar como conta principal</label>
        {err && <div className="err-box"><CircleAlert size={15} />{err}</div>}
        <button type="submit" className="btn btn-primary" style={{ height: 50, borderRadius: 14 }} disabled={busy}>Adicionar conta</button>
      </form>
    </Modal>
  );
}
