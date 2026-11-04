import { useEffect, useMemo, useState } from 'react';
import { Check, CircleAlert, CircleCheck, Hourglass, Info, PauseCircle, ShieldCheck, UserMinus, UserPlus } from 'lucide-react';
import { useSession } from '../../lib/session';
import { api, ApiError, MemberView } from '../../lib/api';
import { CH, digits, fmt, fmtPhone, ini, longDate, pct as fmtPct, SEG, walletOf } from '../../lib/format';
import { Modal, ModalHead, Seg, useToast } from '../../components/ui';
import { PageHead, useBo } from './Layout';

interface Group {
  auto_split: boolean; fee_bps: number; members: MemberView[]; can_edit: boolean;
  proposal: { id: string; splits: { member_id: string; pct: number }[]; waiting: string[]; created_at: string } | null;
  history: { text: string; when: string }[];
}

export default function Group() {
  const { mobile, full, setGroupSize } = useBo();
  const toast = useToast();
  const [g, setG] = useState<Group | null>(null);
  const [pcts, setPcts] = useState<Record<string, number>>({});
  const [sim, setSim] = useState(300);
  const [inviting, setInviting] = useState(false);
  const [busy, setBusy] = useState(false);

  const load = (x?: Group) => {
    const apply = (v: Group) => { setG(v); setGroupSize(v.members.length); setPcts(Object.fromEntries(v.members.map(m => [m.id, m.pct]))); };
    if (x) apply(x); else api.get<Group>('/backoffice/group').then(apply).catch(() => {});
  };
  useEffect(() => load(), []);

  const members = g?.members || [];
  const total = members.reduce((a, m) => a + (pcts[m.id] ?? 0), 0);
  const dirty = useMemo(() => members.some(m => pcts[m.id] !== m.pct), [members, pcts]);
  if (!g) return <div className="skeleton" style={{ height: 400 }} />;
  const ed = g.can_edit;
  const fee = g.fee_bps / 10000;

  const balance = () => {
    const t = total || 0;
    const next = members.map(m => ({ id: m.id, p: t ? Math.round((pcts[m.id] * 100) / t) : Math.floor(100 / members.length) }));
    next[0].p += 100 - next.reduce((a, x) => a + x.p, 0);
    setPcts(Object.fromEntries(next.map(x => [x.id, x.p])));
  };
  const err = (e: unknown) => toast(e instanceof ApiError ? e.message : 'Algo correu mal.', true);

  async function propose() {
    if (total !== 100) { toast('A divisão tem de somar 100%', true); return; }
    setBusy(true);
    try {
      const r = await api.post<Group>('/backoffice/group/proposals', { splits: members.map(m => ({ member_id: m.id, pct: pcts[m.id] })) });
      load(r); toast(r.proposal ? 'Pedido de aprovação enviado por SMS' : 'Divisão atualizada');
    } catch (e) { err(e); } finally { setBusy(false); }
  }
  async function toggleAuto() {
    try { const r = await api.put<Group>('/backoffice/group/auto-split', { on: !g!.auto_split }); load(r); toast(r.auto_split ? 'Divisão automática ativa' : 'Divisão automática pausada'); } catch (e) { err(e); }
  }
  async function remove(m: MemberView) {
    try { await api.del('/backoffice/group/members/' + m.id); toast(`${m.name} removido`); load(); } catch (e) { err(e); }
  }
  async function cancelProposal() {
    try { await api.del('/backoffice/group/proposals/current'); toast('Proposta cancelada'); load(); } catch (e) { err(e); }
  }

  const segs = members.map((m, i) => ({ m, i, p: pcts[m.id] ?? 0 })).filter(x => x.p > 0);
  const simNet = sim * (1 - fee);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <PageHead kicker={`Grupo · ${members.length} ${members.length === 1 ? 'membro' : 'membros'}`} title="Grupo e divisão" lead="Cada apoio é dividido no momento do pagamento e entra no saldo de cada membro."
        right={<button className="btn" style={{ height: 44, padding: '0 8px 0 16px', borderRadius: 14, gap: 12 }} onClick={toggleAuto} disabled={!ed}>Divisão automática<span className={'toggle' + (g.auto_split ? ' on' : '')} aria-hidden><i /></span></button>} />
      {g.proposal && <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 12, padding: '14px 16px', borderRadius: 16, background: '#FBEFD9', color: '#6B3F00' }}>
        <Hourglass size={18} /><span style={{ flex: 1 }}><b>A aguardar aprovação de {g.proposal.waiting.join(', ') || 'todos'}.</b> A divisão atual mantém-se até todos aprovarem ({g.proposal.splits.map(s => s.pct).join(' / ')}).</span>
        {ed && <button className="link-btn" style={{ color: '#6B3F00', textDecoration: 'underline' }} onClick={cancelProposal}>Cancelar proposta</button>}
      </div>}
      {!g.auto_split && <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '14px 16px', borderRadius: 16, background: '#FDE9DC', color: '#7A2E0A' }}><PauseCircle size={18} /><span style={{ flex: 1 }}><b>Divisão pausada.</b> Os novos apoios ficam todos no saldo do admin até voltares a ativar.</span></div>}

      <section className="card" style={{ padding: 'clamp(16px,2vw,22px)', display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12 }}>
          <div style={{ fontWeight: 600, fontSize: 15 }}>Como cada apoio é dividido</div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <span style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, fontWeight: 600, color: total === 100 ? '#17803D' : '#A83700' }}>{total === 100 ? <CircleCheck size={15} /> : <CircleAlert size={15} />}{total === 100 ? '100% distribuído' : total < 100 ? `${total}% · faltam ${100 - total}%` : `${total}% · excede em ${total - 100}%`}</span>
            {total !== 100 && ed && <button className="btn" style={{ height: 34, padding: '0 12px', borderRadius: 10, fontSize: 13 }} onClick={balance}>Equilibrar</button>}
          </div>
        </div>
        <div style={{ display: 'flex', gap: 5, height: 84 }}>
          {segs.map(({ m, i, p }) => (
            <div key={m.id} style={{ flex: `${p} 1 0`, minWidth: 0, background: SEG[i % SEG.length][0], color: SEG[i % SEG.length][1], borderRadius: 16, padding: '12px 14px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between', overflow: 'hidden', transition: 'flex-grow .35s cubic-bezier(.2,.8,.2,1)' }}>
              <span className="display" style={{ fontWeight: 800, fontSize: 22, letterSpacing: '-0.03em', lineHeight: 1 }}>{p}%</span>
              <span style={{ fontSize: 13, fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{m.name.split(' ')[0]}</span>
            </div>
          ))}
          {total < 100 && <div style={{ flex: `${100 - total} 1 0`, minWidth: 0, border: '2px dashed #D14400', borderRadius: 16, display: 'grid', placeItems: 'center', color: '#A83700', fontSize: 13, fontWeight: 600, overflow: 'hidden', whiteSpace: 'nowrap' }}>Faltam {100 - total}%</div>}
        </div>
      </section>

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, alignItems: 'flex-start' }}>
        <section className="card" style={{ flex: '2 1 520px', minWidth: 0, padding: 'clamp(16px,2vw,22px)', display: 'flex', flexDirection: 'column' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, marginBottom: 8 }}>
            <div style={{ fontWeight: 600, fontSize: 15 }}>Membros</div>
            {ed && <button className="btn btn-dark" style={{ height: 38, padding: '0 14px', borderRadius: 11, fontSize: 13 }} onClick={() => setInviting(true)}><UserPlus size={15} />Convidar</button>}
          </div>
          {members.map((m, i) => (
            <div key={m.id} style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: '12px 18px', padding: '16px 0', borderTop: '1px solid #F0EBE3' }}>
              <div style={{ flex: '1 1 200px', minWidth: 0, display: 'flex', alignItems: 'center', gap: 12 }}>
                <span className="avatar" style={{ width: 42, height: 42, borderRadius: 13, background: SEG[i % SEG.length][0], color: SEG[i % SEG.length][1], fontSize: 13 }}>{ini(m.name)}</span>
                <span style={{ minWidth: 0, lineHeight: 1.35 }}>
                  <span style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}><span style={{ fontWeight: 600 }}>{m.name}</span>{m.invited && <span style={{ height: 22, padding: '0 7px', borderRadius: 7, background: '#FBEFD9', color: '#9A5B00', fontSize: 11.5, fontWeight: 600, display: 'inline-flex', alignItems: 'center' }}>Convite enviado</span>}</span>
                  <span style={{ display: 'block', fontSize: 12.5, color: '#6E6861' }}>{m.role}</span>
                </span>
              </div>
              <div style={{ flex: '1 1 160px', minWidth: 0, lineHeight: 1.35 }}>
                <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}><span className="badge" style={{ height: 22, padding: '0 7px', borderRadius: 7, fontSize: 11.5, background: CH[m.wallet].bg, color: CH[m.wallet].fg }}>{CH[m.wallet].name}</span><span className="mono" style={{ fontSize: 12.5 }}>{m.number.replace('+258 ', '')}</span></span>
                <span style={{ display: 'block', marginTop: 3, fontSize: 12.5, color: '#6E6861' }}>Conta de levantamento</span>
              </div>
              <div style={{ flex: '1 1 200px', display: 'flex', alignItems: 'center', gap: 12 }}>
                <input type="range" min={0} max={100} step={1} value={pcts[m.id] ?? 0} disabled={!ed || !!g.proposal} onChange={e => setPcts(p => ({ ...p, [m.id]: +e.target.value }))} aria-label={'Percentagem de ' + m.name} style={{ flex: 1, minWidth: 0 }} />
                <span className="display" style={{ flex: 'none', width: 64, height: 38, borderRadius: 11, border: '1px solid #E4DDD1', display: 'grid', placeItems: 'center', fontWeight: 800, fontSize: 15 }}>{pcts[m.id] ?? 0}%</span>
                {m.is_admin ? <span title="Admin" style={{ flex: 'none', width: 34, height: 34, display: 'grid', placeItems: 'center', color: '#6E6861' }}><ShieldCheck size={16} /></span>
                  : ed && <button onClick={() => remove(m)} aria-label={'Remover ' + m.name} title={m.pct ? 'Põe a 0% e aprova antes de remover' : 'Remover membro'} style={{ flex: 'none', width: 34, height: 34, border: 0, borderRadius: 10, background: 'transparent', display: 'grid', placeItems: 'center', color: '#6E6861' }}><UserMinus size={16} /></button>}
              </div>
            </div>
          ))}
        </section>
        <section className="card-dark" style={{ flex: '1 1 300px', minWidth: 0, padding: 'clamp(16px,2vw,22px)', display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div><div className="mono" style={{ fontSize: 11, letterSpacing: '.08em', textTransform: 'uppercase', color: '#A39C92' }}>Simulador</div><div className="display" style={{ marginTop: 6, fontWeight: 700, fontSize: 20, letterSpacing: '-0.02em', lineHeight: 1.15 }}>Se um fã pagar…</div></div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4,1fr)', gap: 6 }}>
            {[100, 300, 500, 1000].map(v => <button key={v} onClick={() => setSim(v)} className="display" style={{ height: 40, borderRadius: 11, border: `1px solid ${sim === v ? '#F6EFE6' : '#3A3531'}`, background: sim === v ? '#F6EFE6' : 'transparent', color: sim === v ? '#1A1410' : '#F6EFE6', fontWeight: 700, fontSize: 13.5 }}>{fmt(v)}</button>)}
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}><span style={{ color: '#A39C92' }}>Valor pago</span><span className="display" style={{ fontWeight: 800, fontSize: 26, letterSpacing: '-0.03em' }}>{fmt(sim)} MT</span></div>
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, color: '#A39C92' }}><span>Taxa refres ({fmtPct(g.fee_bps)})</span><span className="mono">−{fmt(sim * fee, 2)} MT</span></div>
          <div>
            {members.map((m, i) => (
              <div key={m.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '11px 0', borderTop: '1px solid #2E2A26' }}>
                <span style={{ flex: 'none', width: 12, height: 12, borderRadius: 4, background: SEG[i % SEG.length][0], boxShadow: '0 0 0 1.5px #57524B' }} />
                <span style={{ flex: 1, minWidth: 0, lineHeight: 1.3 }}><span style={{ display: 'block', fontWeight: 600 }}>{m.name.split(' ')[0]} · {pcts[m.id] ?? 0}%</span><span style={{ display: 'block', fontSize: 12, color: '#A39C92', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{CH[m.wallet].name} · {m.number.replace('+258 ', '')}</span></span>
                <span style={{ textAlign: 'right', lineHeight: 1.3 }}><span className="mono" style={{ display: 'block', fontSize: 13.5 }}>{fmt(simNet * (pcts[m.id] ?? 0) / 100, 2)} MT</span><span style={{ display: 'block', fontSize: 12, color: '#5BD08A' }}>no saldo</span></span>
              </div>
            ))}
          </div>
        </section>
      </div>
      <section className="card" style={{ padding: 'clamp(16px,2vw,22px)', display: 'flex', flexWrap: 'wrap', gap: '20px 32px' }}>
        <div style={{ flex: '1 1 260px', display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div style={{ fontWeight: 600, fontSize: 15 }}>Regras do grupo</div>
          {['Qualquer alteração às percentagens precisa da aprovação de todos os membros.', 'Todos os membros veem os mesmos apoios, valores e envios.', 'Cada membro pede os seus levantamentos para M-Pesa ou conta bancária.'].map(t => (
            <div key={t} style={{ display: 'flex', gap: 10, fontSize: 13.5, color: '#2B2723' }}><Check size={16} color="#17803D" style={{ marginTop: 2 }} /><span>{t}</span></div>
          ))}
        </div>
        <div style={{ flex: '1 1 260px', display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div style={{ fontWeight: 600, fontSize: 15 }}>Histórico</div>
          {g.history.length === 0 && <div style={{ fontSize: 13.5, color: '#6E6861' }}>Ainda sem alterações.</div>}
          {g.history.map((h, i) => (
            <div key={i} style={{ display: 'flex', gap: 12, fontSize: 13.5 }}><span style={{ flex: 'none', width: 8, height: 8, marginTop: 7, borderRadius: '50%', background: /Aprovado/.test(h.text) ? '#17803D' : i === 0 ? '#1A1410' : '#D9D1C4' }} /><span style={{ flex: 1, lineHeight: 1.45 }}><span style={{ display: 'block' }}>{h.text}</span><span style={{ display: 'block', fontSize: 12.5, color: '#6E6861' }}>{longDate(h.when)}</span></span></div>
          ))}
        </div>
      </section>
      {dirty && ed && (
        <div className="savebar" style={{ left: mobile ? '50%' : full ? 'calc(50% + 126px)' : 'calc(50% + 38px)' }}>
          <span style={{ width: 8, height: 8, borderRadius: '50%', background: '#FF6A1A' }} />{!mobile && <span style={{ fontWeight: 500 }}>{total === 100 ? 'Nova divisão por aprovar' : `A divisão tem de somar 100% (agora ${total}%)`}</span>}
          <button className="btn-outline-dark" onClick={() => load(g)}>Descartar</button>
          <button className="btn btn-primary" style={{ height: 38, borderRadius: 11, fontSize: 13.5, background: total === 100 ? '#D14400' : '#57524B' }} disabled={total !== 100 || busy} onClick={propose}>Pedir aprovação</button>
        </div>
      )}
      {inviting && <InviteModal onClose={() => setInviting(false)} onDone={m => { setInviting(false); toast(`Convite enviado a ${m.name.split(' ')[0]} por SMS`); load(); }} />}
    </div>
  );
}

function InviteModal({ onClose, onDone }: { onClose: () => void; onDone: (m: MemberView) => void }) {
  const { me } = useSession();
  const emolaSoon = !!me?.coming_soon?.includes('emola');
  const [name, setName] = useState(''); const [role, setRole] = useState(''); const [ch, setCh] = useState<'mpesa' | 'emola'>('mpesa'); const [phone, setPhone] = useState('');
  const [err, setErr] = useState(''); const [busy, setBusy] = useState(false);
  const op = walletOf(phone);
  const valid = name.trim().length > 1 && phone.length === 9 && op === ch;
  const hint = phone.length < 2 ? [`Os levantamentos deste membro vão para este número ${CH[ch].name}.`, '#6E6861', 'info']
    : !op || op === 'tmcel' ? ['Número inválido. Deve começar por 84, 85, 86 ou 87.', '#A83700', 'alert']
      : op !== ch ? [op === 'mpesa' ? 'Este número é Vodacom: escolhe M-Pesa.' : emolaSoon ? 'O e-Mola chega em breve. Por agora usa um número M-Pesa.' : 'Este número é Movitel: escolhe e-Mola.', '#A83700', 'alert']
        : [op === 'mpesa' ? 'Número Vodacom · recebe por M-Pesa' : 'Número Movitel · recebe por e-Mola', '#17803D', 'ok'];
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!valid) return;
    setBusy(true); setErr('');
    try { onDone(await api.post<MemberView>('/backoffice/group/members', { name, role, wallet: ch, phone })); }
    catch (x) { setErr(x instanceof ApiError ? x.message : 'Algo correu mal.'); } finally { setBusy(false); }
  }
  return (
    <Modal onClose={onClose} label="Convidar membro">
      <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <ModalHead title="Convidar membro" sub="Enviamos um SMS com o convite. A pessoa recebe a sua parte no saldo e pede os levantamentos." onClose={onClose} />
        <label className="field"><span>Nome</span><input className="input" value={name} onChange={e => setName(e.target.value.slice(0, 40))} placeholder="Ex.: Tânia Bila" /></label>
        <label className="field"><span>Função no grupo</span><input className="input" value={role} onChange={e => setRole(e.target.value.slice(0, 40))} placeholder="Ex.: Redes sociais" /></label>
        <div className="field"><span>Carteira</span><Seg value={ch} onChange={setCh} options={[['mpesa', 'M-Pesa'], ['emola', 'e-Mola']]} soon={emolaSoon ? ['emola'] : []} /></div>
        <label className="field"><span>Número {CH[ch].name}</span>
          <span className="affix" style={{ height: 48 }}><span className="pre">+258</span><input type="tel" className="mono" style={{ fontSize: 15 }} value={fmtPhone(phone)} onChange={e => setPhone(digits(e.target.value, 9))} placeholder={ch === 'mpesa' ? '84 123 4567' : '86 123 4567'} /></span>
          <span style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12.5, color: hint[1] }}>{hint[2] === 'ok' ? <CircleCheck size={14} /> : hint[2] === 'alert' ? <CircleAlert size={14} /> : <Info size={14} />}{hint[0]}</span>
        </label>
        <div style={{ display: 'flex', gap: 10, padding: '12px 14px', borderRadius: 14, background: '#F8F5EF', fontSize: 12.5, lineHeight: 1.45, color: '#57524B' }}><Info size={15} style={{ marginTop: 1 }} /><span>O novo membro entra com 0%. Ajusta as percentagens e pede a aprovação de todos.</span></div>
        {err && <div className="err-box"><CircleAlert size={15} />{err}</div>}
        <button type="submit" className="btn btn-primary" style={{ height: 50, borderRadius: 14, fontSize: 15, background: valid ? '#D14400' : '#CFC6B8' }} disabled={!valid || busy}>Enviar convite</button>
      </form>
    </Modal>
  );
}

