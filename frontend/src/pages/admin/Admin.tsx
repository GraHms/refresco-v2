import { useEffect, useState } from 'react';
import { NavLink, Outlet, useNavigate, useOutletContext, useParams } from 'react-router-dom';
import { ArrowUpRight, Ban, Banknote, Check, Copy, CupSoda, Eye, FileDown, Handshake, Landmark, Smartphone, LayoutDashboard, LogOut, PiggyBank, Search, Send, TicketPlus, Ticket, TriangleAlert, UsersRound, Wallet, X } from 'lucide-react';
import { api, ApiError, SeriesPoint, WithdrawalView } from '../../lib/api';
import { avOf, bucketLabel, DEST, ini, longDate, mt, mtN, pct, relative, shortDateTime, WD_ST } from '../../lib/format';
import { useSession } from '../../lib/session';
import { Badge, copy, Drawer, Logo, Modal, Toggle, useMedia, useToast } from '../../components/ui';

const NAV = [['painel', 'Painel', LayoutDashboard], ['contas', 'Contas', UsersRound], ['levantamentos', 'Levantamentos', Banknote], ['convites', 'Convites', Ticket]] as const;
const AST = { active: ['Ativa', '#E3F3E8', '#17803D'], pending: ['Por verificar', '#FBEFD9', '#9A5B00'], suspended: ['Suspensa', '#FDE8E4', '#B42318'] } as const;
const STD = 1500;

interface Row { id: string; name: string; handle: string; category: string; owner: string; count: number; volume: number; fee_bps: number; negotiated: boolean; revenue: number; balance: number; status: keyof typeof AST }
interface Dash { revenue: number; volume: number; count: number; avg_fee_bps: number; balances: number; active_count: number; pending_amount: number; pending_count: number; overdue_count: number; negotiated_count: number; series: SeriesPoint[]; top: Row[] }
interface AdminW extends WithdrawalView { account_id: string; account_name: string; account_status: keyof typeof AST; balance: number }
interface Ctx { reloadBadge: () => void }

export function AdminLayout() {
  const { me, logout } = useSession();
  const nav = useNavigate();
  const wide = useMedia('(min-width: 980px)');
  const [badge, setBadge] = useState(0);
  const [neg, setNeg] = useState(0);
  const reloadBadge = () => api.get<Dash>('/admin/dashboard').then(d => { setBadge(d.pending_count); setNeg(d.negotiated_count); }).catch(() => {});
  useEffect(() => { reloadBadge(); document.title = 'Admin · refres.co'; }, []);
  if (!me) return null;
  const out = async () => { await logout(); nav('/entrar', { replace: true }); };
  const item = (id: string, label: string, Icon: typeof Ticket, compact: boolean) => (
    <NavLink key={id} to={'/admin/' + id} style={({ isActive }) => ({ flex: compact ? 'none' : undefined, display: 'flex', alignItems: 'center', gap: compact ? 7 : 12, height: compact ? 36 : 44, padding: '0 12px', borderRadius: compact ? 11 : 13, background: isActive ? (compact ? '#F6EFE6' : '#2E251E') : 'transparent', color: isActive ? (compact ? '#1A1410' : '#fff') : '#CFC6B8', fontWeight: isActive || compact ? 600 : 500, fontSize: compact ? 13 : 14 })}>
      <Icon size={compact ? 15 : 19} /><span style={{ flex: compact ? undefined : 1 }}>{label}</span>
      {id === 'levantamentos' && badge > 0 && <span style={{ minWidth: 20, height: 20, padding: '0 6px', borderRadius: 7, background: '#FF6A1A', color: '#1A1410', fontSize: 11.5, fontWeight: 700, display: 'grid', placeItems: 'center' }}>{badge}</span>}
    </NavLink>
  );
  return (
    <div style={{ minHeight: '100vh', display: 'flex', background: '#F6EFE6' }}>
      {wide && (
        <aside style={{ position: 'sticky', top: 0, height: '100vh', width: 240, flex: 'none', display: 'flex', flexDirection: 'column', gap: 22, padding: '22px 14px', background: '#1A1410', color: '#F6EFE6' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '0 6px' }}><Logo h={30} white /><span className="mono" style={{ height: 22, padding: '0 7px', borderRadius: 7, background: '#FF6A1A', color: '#1A1410', fontSize: 10.5, fontWeight: 600, letterSpacing: '.06em', display: 'flex', alignItems: 'center' }}>ADMIN</span></div>
          <nav style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>{NAV.map(([id, l, I]) => item(id, l, I, false))}</nav>
          <div style={{ flex: 1 }} />
          <div style={{ padding: 14, borderRadius: 16, background: '#251E18', display: 'flex', flexDirection: 'column', gap: 6 }}>
            <span className="mono" style={{ fontSize: 10.5, letterSpacing: '.08em', textTransform: 'uppercase', color: '#A89C8F' }}>Taxa padrão</span>
            <span className="display" style={{ fontWeight: 800, fontSize: 26, letterSpacing: '-0.03em' }}>15%</span>
            <span style={{ fontSize: 12.5, color: '#A89C8F' }}>{neg} {neg === 1 ? 'conta' : 'contas'} com taxa negociada</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: 6 }}>
            <span className="avatar" style={{ width: 34, height: 34, borderRadius: 11, background: '#FF6A1A', color: '#1A1410', fontSize: 12 }}>SA</span>
            <span style={{ flex: 1, minWidth: 0, lineHeight: 1.25 }}><span style={{ display: 'block', fontWeight: 600, fontSize: 13.5 }}>Super admin</span><span style={{ display: 'block', fontSize: 12, color: '#A89C8F', overflow: 'hidden', textOverflow: 'ellipsis' }}>{me.email}</span></span>
            <button onClick={out} title="Sair" aria-label="Sair" style={{ border: 0, background: 'transparent', color: '#A89C8F', display: 'grid' }}><LogOut size={16} /></button>
          </div>
        </aside>
      )}
      <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        {!wide && (
          <div style={{ position: 'sticky', top: 0, zIndex: 40, background: '#1A1410', color: '#F6EFE6', padding: '12px 14px 0' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, paddingBottom: 10 }}><Logo h={24} white /><span className="mono" style={{ height: 20, padding: '0 6px', borderRadius: 6, background: '#FF6A1A', color: '#1A1410', fontSize: 10, fontWeight: 600, display: 'flex', alignItems: 'center' }}>ADMIN</span><span style={{ flex: 1 }} /><button onClick={out} aria-label="Sair" style={{ border: 0, background: 'transparent', color: '#A89C8F' }}><LogOut size={16} /></button></div>
            <div style={{ display: 'flex', gap: 4, overflowX: 'auto', paddingBottom: 10 }}>{NAV.map(([id, l, I]) => item(id, l, I, true))}</div>
          </div>
        )}
        <main style={{ flex: 1, minWidth: 0, padding: '28px clamp(14px,3vw,40px) 60px' }}>
          <div style={{ maxWidth: 1200, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 16 }}>{me.sandbox && <div role="status" style={{ padding: '12px 16px', borderRadius: 16, background: '#FDE8E4', color: '#7A1F14', fontSize: 13.5 }}><b>Modo de teste.</b> Os pagamentos são simulados. Não transfiras dinheiro real para estes levantamentos.</div>}<Outlet context={{ reloadBadge } satisfies Ctx} /></div>
        </main>
      </div>
    </div>
  );
}

function Head({ kicker, title, lead }: { kicker: string; title: string; lead?: string }) {
  return <div style={{ marginBottom: 6 }}><div className="kicker"><i />{kicker}</div><h1 className="h1">{title}</h1>{lead && <p className="lead">{lead}</p>}</div>;
}

export function AdminHome() {
  const nav = useNavigate();
  const [d, setD] = useState<Dash | null>(null);
  const [hot, setHot] = useState<number | null>(null);
  useEffect(() => { api.get<Dash>('/admin/dashboard').then(setD).catch(() => {}); }, []);
  const pts = d?.series || [];
  const max = Math.max(1, ...pts.map(p => p.total));
  const hp = hot != null ? pts[hot] : null;
  const card = (dark: boolean): React.CSSProperties => ({ background: dark ? '#1A1410' : '#fff', color: dark ? '#F6EFE6' : undefined, border: dark ? 0 : '1px solid #E4DDD1', borderRadius: 22, padding: 20, display: 'flex', flexDirection: 'column', gap: 14 });
  const big: React.CSSProperties = { fontFamily: 'var(--display)', fontStretch: '125%', fontWeight: 800, fontSize: 'clamp(28px,2.8vw,36px)', letterSpacing: '-0.035em', lineHeight: 1 };
  return <>
    <Head kicker="Últimos 30 dias" title="Painel da plataforma" />
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(min(100%,220px),1fr))', gap: 14 }}>
      <div style={card(true)}><span style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13.5, color: '#A89C8F' }}>A tua receita<PiggyBank size={17} /></span><span style={big}>{d ? mtN(d.revenue, 2) : '—'} <span style={{ fontSize: 15, color: '#A89C8F' }}>MT</span></span><span style={{ fontSize: 12.5, color: '#FF9A5C' }}>Taxa média {d ? pct(d.avg_fee_bps) : '—'}</span></div>
      <div style={card(false)}><span style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13.5, color: '#57524B' }}>Volume de refrescos<CupSoda size={17} /></span><span style={big}>{d ? mtN(d.volume) : '—'} <span style={{ fontSize: 15, color: '#6E6861' }}>MT</span></span><span style={{ fontSize: 12.5, color: '#6E6861' }}>{d ? `${d.count} refrescos` : ''}</span></div>
      <div style={card(false)}><span style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13.5, color: '#57524B' }}>Saldo dos criadores<Wallet size={17} /></span><span style={big}>{d ? mtN(d.balances, 2) : '—'} <span style={{ fontSize: 15, color: '#6E6861' }}>MT</span></span><span style={{ fontSize: 12.5, color: '#6E6861' }}>Por levantar, em {d?.active_count ?? '—'} contas ativas</span></div>
      <button onClick={() => nav('/admin/levantamentos')} style={{ ...card(false), background: '#FFF4EC', border: '1.5px solid #FBD5BF', textAlign: 'left' }}><span style={{ display: 'flex', justifyContent: 'space-between', width: '100%', fontSize: 13.5, color: '#7A2E0A' }}>Levantamentos por pagar<ArrowUpRight size={17} /></span><span style={big}>{d ? mtN(d.pending_amount) : '—'} <span style={{ fontSize: 15, color: '#A83700' }}>MT</span></span><span style={{ fontSize: 12.5, color: '#7A2E0A' }}>{d ? `${d.pending_count} pedidos · ${d.overdue_count} fora do prazo de 72 h` : ''}</span></button>
    </div>
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, alignItems: 'stretch' }}>
      <div style={{ ...card(false), flex: '2 1 520px', minWidth: 0 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 12, flexWrap: 'wrap' }}>
          <div><div style={{ fontSize: 13.5, color: '#57524B' }}>{hp ? bucketLabel(hp.start, false, false) : 'Receita refres por dia'}</div><div className="display" style={{ fontWeight: 800, fontSize: 24, letterSpacing: '-0.03em' }}>{hp ? `${mt(hp.fee)} de ${mt(hp.total)}` : d ? `${mt(d.revenue)} em 30 dias` : '—'}</div></div>
          <div style={{ display: 'flex', gap: 14, fontSize: 12.5, color: '#57524B' }}><span style={{ display: 'flex', alignItems: 'center', gap: 6 }}><span style={{ width: 10, height: 10, borderRadius: 3, background: '#D14400' }} />Receita refres</span><span style={{ display: 'flex', alignItems: 'center', gap: 6 }}><span style={{ width: 10, height: 10, borderRadius: 3, background: '#ECE6DB' }} />Volume</span></div>
        </div>
        <div onMouseLeave={() => setHot(null)} style={{ display: 'flex', alignItems: 'flex-end', gap: 4, height: 200, borderBottom: '1px solid #E4DDD1' }} role="img" aria-label="Receita e volume por dia">
          {pts.map((p, i) => (
            <div key={i} onMouseEnter={() => setHot(i)} style={{ flex: 1, minWidth: 0, height: (p.total / max * 100).toFixed(1) + '%', display: 'flex', flexDirection: 'column', justifyContent: 'flex-end', borderRadius: '5px 5px 0 0', background: '#ECE6DB', overflow: 'hidden', opacity: hot == null || hot === i ? 1 : 0.35, transition: 'opacity .15s', cursor: 'pointer' }}>
              <div style={{ height: p.total ? (p.fee / p.total * 100).toFixed(1) + '%' : 0, background: '#D14400' }} />
            </div>
          ))}
        </div>
      </div>
      <div style={{ ...card(false), flex: '1 1 300px', minWidth: 0, padding: '20px 14px 12px', gap: 4 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0 6px 8px' }}><span style={{ fontWeight: 600, fontSize: 15 }}>Top contas</span><button className="link-btn" style={{ fontSize: 13 }} onClick={() => nav('/admin/contas')}>Ver todas</button></div>
        {d?.top.map(t => { const [bg, fg] = avOf(t.id); return (
          <button key={t.id} className="row-btn hover-paper" style={{ gap: 10, padding: '9px 6px', borderRadius: 12 }} onClick={() => nav('/admin/contas/' + t.id)}>
            <span className="avatar" style={{ width: 34, height: 34, borderRadius: 11, background: bg, color: fg, fontSize: 12 }}>{ini(t.name)}</span>
            <span style={{ flex: 1, minWidth: 0, lineHeight: 1.3 }}><span style={{ display: 'block', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{t.name}</span><span style={{ display: 'block', fontSize: 12, color: '#6E6861' }}>{t.negotiated ? `Taxa negociada ${pct(t.fee_bps)}` : 'Taxa padrão 15%'}</span></span>
            <span style={{ textAlign: 'right', lineHeight: 1.3 }}><span className="mono" style={{ display: 'block', fontSize: 12.5 }}>{mt(t.volume)}</span><span style={{ display: 'block', fontSize: 12, color: '#A83700' }}>+{mt(t.revenue)}</span></span>
          </button>); })}
        {d && d.top.length === 0 && <div style={{ padding: 20, color: '#6E6861', textAlign: 'center' }}>Ainda sem contas.</div>}
      </div>
    </div>
  </>;
}

export function AdminAccounts() {
  const nav = useNavigate();
  const { id } = useParams();
  const [q, setQ] = useState(''); const [st, setSt] = useState('all');
  const [rows, setRows] = useState<Row[] | null>(null);
  const [tick, setTick] = useState(0);
  useEffect(() => {
    const t = setTimeout(() => {
      const p = new URLSearchParams(); if (q.trim()) p.set('q', q.trim()); if (st !== 'all') p.set('status', st);
      api.get<{ items: Row[] }>('/admin/accounts?' + p).then(r => setRows(r.items)).catch(() => {});
    }, 200);
    return () => clearTimeout(t);
  }, [q, st, tick]);
  const cols = 'minmax(240px,1.6fr) 96px 120px 96px 120px 112px 110px';
  const active = rows?.filter(r => r.status === 'active').length ?? 0;
  return <>
    <Head kicker={rows ? `${rows.length} contas · ${active} ativas` : 'Contas'} title="Contas" />
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, alignItems: 'center' }}>
      <label style={{ flex: '1 1 240px', display: 'flex', alignItems: 'center', gap: 10, height: 44, padding: '0 14px', borderRadius: 13, border: '1px solid #E4DDD1', background: '#fff' }}><Search size={17} color="#6E6861" /><input value={q} onChange={e => setQ(e.target.value)} placeholder="Procurar conta, link ou responsável" aria-label="Procurar" style={{ flex: 1, minWidth: 0, border: 0, outline: 'none', background: 'transparent' }} /></label>
      <div className="seg" style={{ background: '#E9E1D5' }}>{[['all', 'Todas'], ['active', 'Ativas'], ['pending', 'Por verificar'], ['suspended', 'Suspensas'], ['negotiated', 'Taxa negociada']].map(([k, l]) => <button key={k} className={st === k ? 'on' : ''} onClick={() => setSt(k)}>{l}</button>)}</div>
    </div>
    <div className="card" style={{ borderRadius: 22, overflow: 'hidden' }}><div style={{ overflowX: 'auto' }}><div style={{ minWidth: 1040 }}>
      <div className="mono" style={{ display: 'grid', gridTemplateColumns: cols, gap: 14, padding: '12px 20px', background: '#FBF7F1', fontSize: 11, letterSpacing: '.08em', textTransform: 'uppercase', color: '#6E6861' }}><span>Conta</span><span>Refrescos</span><span>Volume 30d</span><span>Taxa</span><span>A tua receita</span><span>Saldo</span><span>Estado</span></div>
      {rows?.map(a => { const [bg, fg] = avOf(a.id); return (
        <button key={a.id} className="hover-paper" onClick={() => nav('/admin/contas/' + a.id)} style={{ display: 'grid', gridTemplateColumns: cols, gap: 14, alignItems: 'center', width: '100%', padding: '13px 20px', border: 0, borderTop: '1px solid #F2EAE0', background: '#fff', textAlign: 'left' }}>
          <span style={{ display: 'flex', alignItems: 'center', gap: 12, minWidth: 0 }}><span className="avatar" style={{ width: 38, height: 38, borderRadius: 12, background: bg, color: fg, fontSize: 12.5 }}>{ini(a.name)}</span><span style={{ minWidth: 0, lineHeight: 1.3 }}><span style={{ display: 'block', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{a.name}</span><span className="mono" style={{ display: 'block', fontSize: 12, color: '#6E6861', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>/{a.handle} · {a.category}</span></span></span>
          <span className="mono" style={{ fontSize: 13 }}>{a.count}</span>
          <span className="mono" style={{ fontSize: 13 }}>{mt(a.volume)}</span>
          <span><span style={{ display: 'inline-flex', alignItems: 'center', gap: 5, height: 26, padding: '0 9px', borderRadius: 8, background: a.negotiated ? '#FDE9DC' : '#F6EFE6', color: a.negotiated ? '#A83700' : '#57524B', fontWeight: 700, fontSize: 12.5 }}>{pct(a.fee_bps)}{a.negotiated && <Handshake size={13} />}</span></span>
          <span className="mono" style={{ fontSize: 13, color: '#A83700' }}>{mt(a.revenue)}</span>
          <span className="mono" style={{ fontSize: 13 }}>{mt(a.balance)}</span>
          <span><Badge st={AST[a.status]} /></span>
        </button>); })}
      {rows && rows.length === 0 && <div style={{ padding: 48, textAlign: 'center', color: '#6E6861', borderTop: '1px solid #F2EAE0' }}>Nenhuma conta encontrada.</div>}
    </div></div></div>
    {id && <AccountDrawer id={id} onClose={() => nav('/admin/contas')} onChanged={() => setTick(t => t + 1)} />}
  </>;
}

interface Detail extends Row { total_volume: number; total_revenue: number; fee_note: string; fee_until: string | null; status_note: string; owner_phone: string; owner_email: string; members: { name: string; pct: number }[]; created_at: string; invite_code: string; log: { text: string; when: string }[] }

function AccountDrawer({ id, onClose, onChanged }: { id: string; onClose: () => void; onChanged: () => void }) {
  const toast = useToast();
  const { reloadBadge } = useOutletContext<Ctx>();
  const [a, setA] = useState<Detail | null>(null);
  const [ng, setNg] = useState({ on: false, fee: 1200, months: 0, note: '' });
  const [busy, setBusy] = useState(false);
  const init = (x: Detail) => { setA(x); setNg({ on: x.negotiated, fee: x.negotiated ? x.fee_bps : 1200, months: 0, note: x.fee_note || '' }); };
  useEffect(() => { api.get<Detail>('/admin/accounts/' + id).then(init).catch(() => onClose()); }, [id]);
  if (!a) return <Drawer onClose={onClose} label="Conta"><div style={{ padding: 24 }}><div className="skeleton" style={{ height: 240 }} /></div></Drawer>;
  const [bg, fg] = avOf(a.id);
  const dirty = ng.on !== a.negotiated || (ng.on && (ng.fee !== a.fee_bps || ng.note !== (a.fee_note || '') || ng.months !== 0));
  const err = (e: unknown) => toast(e instanceof ApiError ? e.message : 'Algo correu mal.', true);
  async function saveFee() {
    setBusy(true);
    try { const r = await api.put<Detail>(`/admin/accounts/${a!.id}/fee`, { enabled: ng.on, fee_bps: ng.fee, months: ng.months, note: ng.note }); init(r); onChanged(); reloadBadge(); toast(`Taxa de ${r.name}: ${pct(r.fee_bps)}`); }
    catch (e) { err(e); } finally { setBusy(false); }
  }
  async function toggleStatus() {
    const next = a!.status === 'active' ? 'suspended' : 'active';
    if (next === 'suspended' && !confirm(`Suspender ${a!.name}? A página deixa de receber apoios.`)) return;
    setBusy(true);
    try { const r = await api.put<Detail>(`/admin/accounts/${a!.id}/status`, { status: next }); init(r); onChanged(); reloadBadge(); toast(next === 'suspended' ? `${r.name} suspensa` : `${r.name} ativa`); }
    catch (e) { err(e); } finally { setBusy(false); }
  }
  const members = a.members.length > 1 ? `${a.members.length} · divisão ${a.members.map(m => m.pct).join('/')}` : 'Individual';
  const stats: [string, string][] = [['Refrescos · 30d', String(a.count)], ['Volume · 30d', mt(a.volume)], ['Volume total', mt(a.total_volume)], ['Saldo por levantar', mt(a.balance)], ['A tua receita · 30d', mt(a.revenue)], ['A tua receita · total', mt(a.total_revenue)]];
  return (
    <Drawer onClose={onClose} label={a.name}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '18px 18px 16px', borderBottom: '1px solid #F2EAE0' }}>
        <span className="avatar" style={{ width: 46, height: 46, borderRadius: 14, background: bg, color: fg, fontSize: 14 }}>{ini(a.name)}</span>
        <span style={{ flex: 1, minWidth: 0, lineHeight: 1.3 }}><span style={{ display: 'block', fontWeight: 700, fontSize: 16 }}>{a.name}</span><span className="mono" style={{ display: 'block', fontSize: 12, color: '#6E6861' }}>pagarefresco.com/{a.handle}</span></span>
        <Badge st={AST[a.status]} dot={false} />
        <button className="icon-btn" onClick={onClose} aria-label="Fechar"><X size={17} /></button>
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: 18, display: 'flex', flexDirection: 'column', gap: 18 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
          {stats.map(([l, v]) => <div key={l} style={{ padding: '12px 14px', borderRadius: 14, background: '#FBF7F1' }}><span style={{ display: 'block', fontSize: 12, color: '#6E6861' }}>{l}</span><span className="display" style={{ display: 'block', fontWeight: 800, fontSize: 18, letterSpacing: '-0.02em' }}>{v}</span></div>)}
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, padding: 16, borderRadius: 18, border: `1.5px solid ${ng.on ? '#FBD5BF' : '#E4DDD1'}`, background: ng.on ? '#FFF4EC' : '#FBF7F1' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 10 }}><span style={{ display: 'flex', alignItems: 'center', gap: 8, fontWeight: 700 }}><Handshake size={17} />Taxa negociada</span><Toggle on={ng.on} label="Taxa negociada" onClick={() => setNg(n => ({ ...n, on: !n.on }))} /></div>
          {!ng.on ? <div style={{ fontSize: 13.5, color: '#57524B' }}>Esta conta paga a taxa padrão de 15%.</div> : <>
            <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}><input type="range" min={500} max={2000} step={50} value={ng.fee} onChange={e => setNg(n => ({ ...n, fee: +e.target.value }))} aria-label="Taxa" style={{ flex: 1, minWidth: 0 }} /><span className="display" style={{ flex: 'none', width: 72, height: 42, borderRadius: 12, background: '#fff', border: '1px solid #E4DDD1', display: 'grid', placeItems: 'center', fontWeight: 800, fontSize: 17 }}>{pct(ng.fee)}</span></div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
              <label style={{ display: 'flex', flexDirection: 'column', gap: 5 }}><span style={{ fontSize: 12, color: '#57524B' }}>Válida por</span><select value={ng.months} onChange={e => setNg(n => ({ ...n, months: +e.target.value }))} style={{ height: 40, padding: '0 10px', borderRadius: 10, border: '1px solid #E4DDD1', background: '#fff', outline: 'none', fontSize: 13.5 }}><option value={0}>{a.fee_until ? 'Manter / sem prazo' : 'Sem prazo'}</option><option value={3}>3 meses</option><option value={6}>6 meses</option><option value={12}>12 meses</option></select></label>
              <label style={{ display: 'flex', flexDirection: 'column', gap: 5 }}><span style={{ fontSize: 12, color: '#57524B' }}>Motivo</span><input value={ng.note} onChange={e => setNg(n => ({ ...n, note: e.target.value.slice(0, 60) }))} placeholder="Ex.: parceiro de lançamento" style={{ height: 40, padding: '0 10px', borderRadius: 10, border: '1px solid #E4DDD1', outline: 'none', fontSize: 13.5, minWidth: 0 }} /></label>
            </div>
            {a.fee_until && <div style={{ fontSize: 12.5, color: '#6E6861' }}>Válida até {longDate(a.fee_until)}.</div>}
            <div style={{ padding: 12, borderRadius: 12, background: '#fff', fontSize: 13, lineHeight: 1.5, color: '#2B2723' }}>Com o volume dos últimos 30 dias ({mt(a.volume)}), a tua receita passa de <b>{mt(a.volume * STD / 10000)}</b> para <b style={{ color: '#A83700' }}>{mt(a.volume * ng.fee / 10000)}</b> por mês.</div>
          </>}
          {dirty && <button className="btn btn-primary" style={{ height: 44 }} onClick={saveFee} disabled={busy}>Guardar taxa</button>}
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div className="label-sm">Responsável</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6, padding: '12px 14px', borderRadius: 14, border: '1px solid #EFE7DC', fontSize: 13.5 }}>
            {([['Nome', a.owner], ['Contacto', a.owner_phone], ['Email', a.owner_email], ['Membros', members], ['Criada', `${longDate(a.created_at)} · convite ${a.invite_code}`]] as const).map(([k, v]) =>
              <div key={k} style={{ display: 'flex', justifyContent: 'space-between', gap: 10 }}><span style={{ color: '#6E6861' }}>{k}</span><span className={k === 'Contacto' ? 'mono' : ''} style={{ fontWeight: k === 'Nome' ? 600 : 400, textAlign: 'right', overflowWrap: 'anywhere' }}>{v}</span></div>)}
          </div>
        </div>
        {a.status_note && a.status !== 'active' && <div style={{ display: 'flex', gap: 10, padding: '12px 14px', borderRadius: 14, background: '#FDE8E4', color: '#7A1F14', fontSize: 13 }}><TriangleAlert size={16} style={{ marginTop: 1 }} /><span>{a.status_note}</span></div>}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div className="label-sm">Histórico</div>
          {a.log.map((l, i) => <div key={i} style={{ display: 'flex', gap: 10, fontSize: 13 }}><span style={{ flex: 'none', width: 7, height: 7, marginTop: 7, borderRadius: '50%', background: '#D14400' }} /><span style={{ flex: 1, lineHeight: 1.45 }}><span style={{ display: 'block' }}>{l.text}</span><span style={{ display: 'block', fontSize: 12, color: '#6E6861' }}>{longDate(l.when)}</span></span></div>)}
        </div>
      </div>
      <div style={{ display: 'flex', gap: 8, padding: '14px 18px 18px', borderTop: '1px solid #F2EAE0' }}>
        <a className="btn" style={{ flex: 1, height: 44 }} href={'/' + a.handle} target="_blank" rel="noopener">Ver página<ArrowUpRight size={14} /></a>
        <button className="btn" style={{ flex: 1, height: 44, border: 0, background: a.status === 'active' ? '#B42318' : '#17803D', color: '#fff' }} onClick={toggleStatus} disabled={busy}>{a.status === 'active' ? 'Suspender conta' : a.status === 'pending' ? 'Verificar conta' : 'Reativar conta'}</button>
      </div>
    </Drawer>
  );
}

const REASONS = ['NIB inválido', 'Conta por verificar', 'Conta suspensa', 'Suspeita de fraude'];

export function AdminWithdrawals() {
  const toast = useToast();
  const { reloadBadge } = useOutletContext<Ctx>();
  const [data, setData] = useState<{ items: AdminW[]; summary: Record<string, { count: number; amount: number }> } | null>(null);
  const [f, setF] = useState('open');
  const [modal, setModal] = useState<{ type: 'reject' | 'complete'; w: AdminW; reason: string; note: string; ref: string } | null>(null);
  const [detail, setDetail] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const load = () => api.get<typeof data>('/admin/withdrawals').then(setData).catch(() => {});
  useEffect(() => { load(); }, []);
  const now = Date.now();
  const closed = (w: AdminW) => w.status === 'paid' || w.status === 'rejected';
  const left = (w: AdminW) => Math.round((new Date(w.due_at).getTime() - now) / 3600000);
  const items = (data?.items || []).filter(w => f === 'all' || (f === 'open' ? !closed(w) : w.status === f))
    .sort((x, y) => +closed(x) - +closed(y) || (closed(x) ? 0 : left(x) - left(y)));
  async function act(w: AdminW, body: Record<string, string>, msg: string) {
    setBusy(true);
    try { const r = await api.post<typeof data>(`/admin/withdrawals/${w.id}/actions`, body); setData(r); setModal(null); setDetail(d => (d ? d + '#' : d)); toast(msg); reloadBadge(); }
    catch (e) { toast(e instanceof ApiError ? e.message : 'Algo correu mal.', true); load(); } finally { setBusy(false); }
  }
  const sum = (k: string) => data?.summary[k] || { count: 0, amount: 0 };
  return <>
    <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
      <Head kicker="Prazo: 72 horas por pedido" title="Levantamentos" />
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 6 }}>
        {sum('approved').count > 0 && <a className="btn" href="/api/v1/admin/withdrawals.pdf?status=approved" download><FileDown size={16} />PDF dos aprovados ({sum('approved').count})</a>}
        {sum('requested').count > 0 && <a className="btn" href="/api/v1/admin/withdrawals.pdf?status=requested" download><FileDown size={16} />PDF dos pedidos ({sum('requested').count})</a>}
      </div>
    </div>
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(min(100%,180px),1fr))', gap: 12 }}>
      {([['open', 'Por tratar', '#D14400'], ['approved', 'Aprovados', '#9A5B00'], ['paid', 'Pagos', '#17803D'], ['rejected', 'Rejeitados', '#B42318']] as const).map(([k, l, dot]) => (
        <button key={k} onClick={() => setF(f === k ? 'all' : k)} style={{ padding: 16, borderRadius: 18, border: `1.5px solid ${f === k ? '#1A1410' : '#E4DDD1'}`, background: f === k ? '#fff' : '#FBF7F1', textAlign: 'left', display: 'flex', flexDirection: 'column', gap: 6 }}>
          <span style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: '#57524B' }}><span style={{ width: 8, height: 8, borderRadius: '50%', background: dot }} />{l}</span>
          <span className="display" style={{ fontWeight: 800, fontSize: 24, letterSpacing: '-0.03em', lineHeight: 1.1 }}>{sum(k).count}</span>
          <span className="mono" style={{ fontSize: 12, color: '#6E6861' }}>{mt(sum(k).amount)}</span>
        </button>
      ))}
    </div>
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {items.map(w => {
        const [bg, fg] = avOf(w.account_id), h = left(w), flag = w.account_status !== 'active' && !closed(w) ? (w.account_status === 'suspended' ? 'Conta suspensa' : 'Conta por verificar') : '';
        const sla = closed(w) ? ['', ''] : h < 0 ? [`Atrasado ${-h} h`, '#B42318'] : [`Faltam ${h} h`, h <= 24 ? '#B42318' : '#9A5B00'];
        return (
          <div key={w.id} style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: '12px 18px', padding: '16px 18px', borderRadius: 20, background: '#fff', border: `1px solid ${!closed(w) && (h < 0 || flag) ? '#F2C1BA' : '#E4DDD1'}` }}>
            <div style={{ flex: '1 1 240px', minWidth: 0, display: 'flex', alignItems: 'center', gap: 12 }}><span className="avatar" style={{ width: 40, height: 40, borderRadius: 13, background: bg, color: fg, fontSize: 12.5 }}>{ini(w.account_name)}</span><span style={{ minWidth: 0, lineHeight: 1.35 }}><span style={{ display: 'block', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{w.member_name} · {w.account_name}</span><span style={{ display: 'block', fontSize: 12.5, color: '#6E6861' }}>{w.code} · pedido {relative(w.created_at)}</span></span></div>
            <div style={{ flex: '1 1 180px', minWidth: 0, lineHeight: 1.35 }}><span style={{ display: 'block', fontWeight: 600 }}>{w.dest_kind === 'nib' ? `${DEST.nib} · ${w.dest_bank}` : DEST[w.dest_kind]}</span><span className="mono" style={{ display: 'block', fontSize: 12.5, color: '#57524B', overflowWrap: 'anywhere' }}>{w.dest_number}</span></div>
            <div style={{ flex: '0 0 130px', lineHeight: 1.3 }}><span className="display" style={{ display: 'block', fontWeight: 800, fontSize: 19, letterSpacing: '-0.02em' }}>{mt(w.amount)}</span><span style={{ display: 'block', fontSize: 12, color: '#6E6861' }}>Saldo: {mt(w.balance)}</span></div>
            <div style={{ flex: '0 0 150px', display: 'flex', flexDirection: 'column', gap: 4 }}><Badge st={WD_ST[w.status]} style={{ alignSelf: 'flex-start' }} /><span style={{ fontSize: 12, fontWeight: 600, color: flag ? '#B42318' : sla[1] }}>{flag ? '⚠ ' + flag : sla[0]}</span></div>
            <div style={{ flex: '1 1 220px', display: 'flex', justifyContent: 'flex-end', gap: 8, flexWrap: 'wrap' }}>
              <button className="btn" style={{ height: 40, padding: '0 12px', fontSize: 13.5, borderColor: '#E4DDD1' }} onClick={() => setDetail(w.id)} aria-label={'Ver detalhes de ' + w.code}><Eye size={15} />Detalhes</button>
              {w.status === 'requested' && !flag && <button className="btn btn-dark" style={{ height: 40, padding: '0 14px', fontSize: 13.5 }} disabled={busy} onClick={() => act(w, { action: 'approve' }, `${w.code} aprovado · pronto a pagar`)}><Check size={15} />Aprovar</button>}
              {w.status === 'approved' && <button className="btn btn-primary" style={{ height: 40, padding: '0 14px', fontSize: 13.5 }} onClick={() => setModal({ type: 'complete', w, reason: '', note: '', ref: '' })}><Send size={15} />Marcar como pago</button>}
              {!closed(w) && <button className="btn" style={{ height: 40, padding: '0 14px', fontSize: 13.5, color: '#B42318', borderColor: '#E4DDD1' }} onClick={() => setModal({ type: 'reject', w, reason: flag === 'Conta suspensa' ? flag : flag ? 'Conta por verificar' : '', note: '', ref: '' })}>Rejeitar</button>}
              {closed(w) && <span style={{ fontSize: 12.5, color: '#6E6861', textAlign: 'right', maxWidth: 240 }}>{w.status === 'paid' ? `Pago ${w.paid_at ? shortDateTime(w.paid_at) : ''} · ${w.transfer_ref}` : `Rejeitado ${w.decided_at ? shortDateTime(w.decided_at) : ''} · ${w.reason}`}</span>}
            </div>
          </div>
        );
      })}
      {data && items.length === 0 && <div className="card" style={{ padding: 48, textAlign: 'center', color: '#6E6861', borderRadius: 20 }}>Nada para mostrar aqui.</div>}
    </div>
    {detail && <WithdrawalDrawer id={detail.replace(/#+$/, '')} version={detail} onClose={() => setDetail(null)} busy={busy}
      onApprove={w => act(w, { action: 'approve' }, `${w.code} aprovado · pronto a pagar`)}
      onComplete={w => setModal({ type: 'complete', w, reason: '', note: '', ref: '' })}
      onReject={w => setModal({ type: 'reject', w, reason: w.account_status === 'suspended' ? 'Conta suspensa' : w.account_status === 'pending' ? 'Conta por verificar' : '', note: '', ref: '' })} />}
    {modal && (
      <Modal onClose={() => setModal(null)} label="Levantamento">
        <div className="display" style={{ fontWeight: 800, fontSize: 21, letterSpacing: '-0.03em', lineHeight: 1.15 }}>{modal.type === 'reject' ? `Rejeitar ${modal.w.code}` : `Marcar ${modal.w.code} como pago`}</div>
        <div style={{ fontSize: 13.5, color: '#57524B', marginTop: -6 }}>{mt(modal.w.amount, 2)} para {modal.w.member_name} · {modal.w.account_name} · {DEST[modal.w.dest_kind]}</div>
        {modal.type === 'reject' ? <>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>{REASONS.map(r => <button key={r} onClick={() => setModal(m => m && { ...m, reason: r })} style={{ height: 36, padding: '0 12px', borderRadius: 10, border: `1.5px solid ${modal.reason === r ? '#B42318' : '#E4DDD1'}`, background: modal.reason === r ? '#FDE8E4' : '#fff', fontWeight: 600, fontSize: 13 }}>{r}</button>)}</div>
          <textarea className="input" rows={2} value={modal.note} onChange={e => setModal(m => m && { ...m, note: e.target.value.slice(0, 280) })} placeholder="Mensagem para o criador (opcional)" style={{ resize: 'none' }} />
          <div style={{ fontSize: 12.5, color: '#6E6861' }}>O valor volta ao saldo do criador.</div>
        </> : <>
          <label className="field"><span style={{ fontSize: 12.5 }}>Referência da transferência</span><input className="input mono" autoFocus value={modal.ref} onChange={e => setModal(m => m && { ...m, ref: e.target.value.toUpperCase().replace(/[^A-Z0-9-]/g, '').slice(0, 40) })} placeholder={modal.w.dest_kind === 'nib' ? 'BCI-TRF-12345' : modal.w.dest_kind === 'emola' ? 'EM-PAY-12345' : 'MP-PAY-12345'} /></label>
          <div style={{ fontSize: 12.5, color: '#6E6861' }}>O criador recebe uma notificação com esta referência.</div>
        </>}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.3fr', gap: 8, marginTop: 4 }}>
          <button className="btn" style={{ height: 46 }} onClick={() => setModal(null)}>Cancelar</button>
          <button className="btn" style={{ height: 46, border: 0, color: '#fff', background: modal.type === 'reject' ? '#B42318' : '#D14400' }} disabled={busy || (modal.type === 'reject' ? !modal.reason : modal.ref.length < 4)}
            onClick={() => modal.type === 'reject' ? act(modal.w, { action: 'reject', reason: modal.reason, note: modal.note }, `${modal.w.code} rejeitado · valor devolvido ao saldo`) : act(modal.w, { action: 'complete', ref: modal.ref }, `${modal.w.code} pago · criador notificado`)}>
            {modal.type === 'reject' ? 'Rejeitar pedido' : 'Confirmar pagamento'}
          </button>
        </div>
      </Modal>
    )}
  </>;
}

interface WDetail extends AdminW { member_role: string; member_phone: string; member_pct: number; account_handle: string; owner_name: string; owner_email: string; owner_phone: string; recent: WithdrawalView[]; paid_total: number; dest_added_at?: string; dest_first_use: boolean }

function CopyRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  const toast = useToast();
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px', borderTop: '1px solid #F0EBE3' }}>
      <span style={{ width: 120, flex: 'none', color: '#6E6861' }}>{label}</span>
      <span className={mono ? 'mono' : ''} style={{ flex: 1, minWidth: 0, fontWeight: mono ? 600 : 500, overflowWrap: 'anywhere' }}>{value || '—'}</span>
      {value && <button className="link-btn" style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12.5 }} onClick={() => { copy(value.replace(/^\+258 /, '').replace(/\s/g, mono ? '' : ' ')); toast(label + ' copiado'); }} aria-label={'Copiar ' + label}><Copy size={13} />Copiar</button>}
    </div>
  );
}

function WithdrawalDrawer({ id, version, onClose, busy, onApprove, onComplete, onReject }: { id: string; version: string; onClose: () => void; busy: boolean; onApprove: (w: AdminW) => void; onComplete: (w: AdminW) => void; onReject: (w: AdminW) => void }) {
  const [w, setW] = useState<WDetail | null>(null);
  useEffect(() => { api.get<WDetail>('/admin/withdrawals/' + id).then(setW).catch(() => onClose()); }, [id, version]);
  if (!w) return <Drawer onClose={onClose} label="Levantamento"><div style={{ padding: 24 }}><div className="skeleton" style={{ height: 260 }} /></div></Drawer>;
  const closed = w.status === 'paid' || w.status === 'rejected';
  const h = Math.round((new Date(w.due_at).getTime() - Date.now()) / 3600000);
  const flag = !closed && w.account_status !== 'active';
  const nib = w.dest_kind === 'nib';
  const section = (t: string) => <div className="label-sm" style={{ marginTop: 4 }}>{t}</div>;
  const box: React.CSSProperties = { flex: 'none', borderRadius: 14, border: '1px solid #ECE7DF', overflow: 'hidden', fontSize: 13.5 };
  const timeline: [string, string | undefined][] = [['Pedido', w.created_at], [w.status === 'rejected' ? 'Rejeitado' : 'Aprovado', w.decided_at], ['Pago', w.paid_at]];
  return (
    <Drawer onClose={onClose} label={'Levantamento ' + w.code}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '18px 18px 16px', borderBottom: '1px solid #F2EAE0' }}>
        <span style={{ flex: 'none', width: 44, height: 44, borderRadius: 14, background: nib ? '#ECE6DB' : '#D14400', color: nib ? '#1A1410' : '#fff', display: 'grid', placeItems: 'center' }}>{nib ? <Landmark size={19} /> : <Smartphone size={19} />}</span>
        <span style={{ flex: 1, minWidth: 0, lineHeight: 1.3 }}><span className="mono" style={{ display: 'block', fontWeight: 700, fontSize: 16 }}>{w.code}</span><span style={{ display: 'block', fontSize: 12.5, color: '#6E6861' }}>{w.member_name} · {w.account_name}</span></span>
        <Badge st={WD_ST[w.status]} />
        <button className="icon-btn" onClick={onClose} aria-label="Fechar"><X size={17} /></button>
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: 18, display: 'flex', flexDirection: 'column', gap: 14 }}>
        <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', gap: 12 }}>
          <div><div className="display" style={{ fontWeight: 800, fontSize: 38, letterSpacing: '-0.04em', lineHeight: 1 }}>{mt(w.amount, 2)}</div>
            <div style={{ marginTop: 6, fontSize: 13, color: closed ? '#6E6861' : h < 0 || h <= 24 ? '#B42318' : '#9A5B00', fontWeight: closed ? 400 : 600 }}>{closed ? `Pedido a ${shortDateTime(w.created_at)}` : h < 0 ? `Atrasado ${-h} h · prazo ${shortDateTime(w.due_at)}` : `Faltam ${h} h · pagar até ${shortDateTime(w.due_at)}`}</div></div>
        </div>
        {flag && <div style={{ display: 'flex', gap: 10, padding: '12px 14px', borderRadius: 14, background: '#FDE8E4', color: '#7A1F14', fontSize: 13 }}><TriangleAlert size={16} style={{ marginTop: 1 }} /><span>{w.account_status === 'suspended' ? 'A conta está suspensa. Não pagues este pedido.' : 'A conta ainda não foi verificada. Verifica-a em Contas antes de aprovar.'}</span></div>}
        {!closed && w.dest_first_use && (() => {
          const hours = w.dest_added_at ? (new Date(w.created_at).getTime() - new Date(w.dest_added_at).getTime()) / 3600000 : null;
          const recent = hours != null && hours < 72;
          return <div style={{ display: 'flex', gap: 10, padding: '12px 14px', borderRadius: 14, background: recent ? '#FDE8E4' : '#FBEFD9', color: recent ? '#7A1F14' : '#6B3F00', fontSize: 13 }}><TriangleAlert size={16} style={{ marginTop: 1 }} /><span><b>Primeiro levantamento para este destino.</b> {recent ? `A conta foi adicionada ${hours! < 1 ? 'menos de 1 hora' : Math.round(hours!) + ' h'} antes do pedido. Confirma com o criador por telefone antes de pagar.` : 'Confirma os dados antes de pagar.'}</span></div>;
        })()}
        {nib && !w.dest_holder && !closed && <div style={{ display: 'flex', gap: 10, padding: '12px 14px', borderRadius: 14, background: '#FBEFD9', color: '#6B3F00', fontSize: 13 }}><TriangleAlert size={16} style={{ marginTop: 1 }} /><span>O titular da conta bancária não foi indicado. Confirma com o criador antes de transferir.</span></div>}
        {section('Dados para pagamento')}
        <div style={box}>
          <div style={{ display: 'flex', gap: 10, padding: '10px 14px' }}><span style={{ width: 120, flex: 'none', color: '#6E6861' }}>Meio</span><span style={{ fontWeight: 600 }}>{nib ? `Transferência bancária · ${w.dest_bank}` : DEST[w.dest_kind]}</span></div>
          <CopyRow label={nib ? 'NIB' : 'Número'} value={w.dest_number} mono />
          <CopyRow label={nib ? 'Titular' : 'Nome'} value={nib ? w.dest_holder || '' : w.member_name} />
          <CopyRow label="Valor" value={mtN(w.amount, 2)} mono />
          <CopyRow label="Descritivo" value={'REFRES ' + w.code} mono />
        </div>
        {section('Beneficiário')}
        <div style={box}>
          {([['Membro', `${w.member_name}${w.member_role ? ' · ' + w.member_role : ''} · ${w.member_pct}%`], ['Contacto', w.member_phone], ['Conta', `${w.account_name} · pagarefresco.com/${w.account_handle}`], ['Estado da conta', AST[w.account_status][0]], ['Responsável', `${w.owner_name} · ${w.owner_email}`], ['Saldo atual', mt(w.balance, 2)], ['Já levantado', mt(w.paid_total, 2)]] as const).map(([k, v], i) =>
            <div key={k} style={{ display: 'flex', gap: 10, padding: '10px 14px', borderTop: i ? '1px solid #F0EBE3' : 0 }}><span style={{ width: 120, flex: 'none', color: '#6E6861' }}>{k}</span><span style={{ flex: 1, minWidth: 0, overflowWrap: 'anywhere' }}>{v}</span></div>)}
        </div>
        {section('Histórico')}
        {timeline.filter(([, t]) => t).map(([k, t]) => <div key={k} style={{ display: 'flex', gap: 10, fontSize: 13 }}><span style={{ flex: 'none', width: 7, height: 7, marginTop: 7, borderRadius: '50%', background: '#D14400' }} /><span style={{ flex: 1 }}><b style={{ fontWeight: 600 }}>{k}</b> · {shortDateTime(t!)}{k === 'Pago' && w.transfer_ref ? ` · ref. ${w.transfer_ref}` : ''}{k === 'Rejeitado' && w.reason ? ` · ${w.reason}` : ''}</span></div>)}
        {w.note && <div style={{ padding: '10px 12px', borderRadius: 12, background: '#F8F5EF', fontSize: 13 }}>{w.note}</div>}
        {w.recent.length > 0 && <>{section('Levantamentos anteriores deste membro')}
          <div style={box}>{w.recent.map((r, i) => <div key={r.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '9px 14px', borderTop: i ? '1px solid #F0EBE3' : 0 }}><span className="mono" style={{ fontSize: 12.5 }}>{r.code}</span><span style={{ flex: 1, fontSize: 12.5, color: '#6E6861' }}>{shortDateTime(r.created_at)}</span><span className="mono" style={{ fontSize: 12.5 }}>{mt(r.amount)}</span><Badge st={WD_ST[r.status]} style={{ height: 22, fontSize: 11.5 }} /></div>)}</div></>}
      </div>
      <div style={{ display: 'flex', gap: 8, padding: '14px 18px 18px', borderTop: '1px solid #F2EAE0', flexWrap: 'wrap' }}>
        <a className="btn" style={{ flex: '1 1 120px', height: 44 }} href={`/api/v1/admin/withdrawals/${w.id}/pdf`} download><FileDown size={15} />Descarregar PDF</a>
        {w.status === 'requested' && !flag && <button className="btn btn-dark" style={{ flex: '1 1 120px', height: 44 }} disabled={busy} onClick={() => onApprove(w)}><Check size={15} />Aprovar</button>}
        {w.status === 'approved' && <button className="btn btn-primary" style={{ flex: '1 1 120px', height: 44 }} onClick={() => onComplete(w)}><Send size={15} />Marcar como pago</button>}
        {!closed && <button className="btn" style={{ flex: '1 1 100px', height: 44, color: '#B42318' }} onClick={() => onReject(w)}>Rejeitar</button>}
      </div>
    </Drawer>
  );
}

interface Inv { code: string; note: string; max_uses: number; uses: number; used_by: string[]; fee_bps: number; status: 'active' | 'revoked'; created_at: string }

export function AdminInvites() {
  const toast = useToast();
  const [rows, setRows] = useState<Inv[] | null>(null);
  const [note, setNote] = useState(''); const [uses, setUses] = useState(1); const [fee, setFee] = useState(1500);
  const [fresh, setFresh] = useState(''); const [busy, setBusy] = useState(false);
  const load = () => api.get<{ items: Inv[] }>('/admin/invites').then(r => setRows(r.items)).catch(() => {});
  useEffect(() => { load(); }, []);
  const link = (c: string) => `${location.origin}/registar?convite=${c}`;
  async function create() {
    setBusy(true);
    try { const r = await api.post<Inv>('/admin/invites', { note, max_uses: uses, fee_bps: fee }); setFresh(r.code); setNote(''); copy(link(r.code)); toast(`Código ${r.code} criado e copiado`); load(); }
    catch (e) { toast(e instanceof ApiError ? e.message : 'Algo correu mal.', true); } finally { setBusy(false); }
  }
  async function revoke(c: string) {
    if (!confirm(`Revogar ${c}?`)) return;
    try { await api.post(`/admin/invites/${c}/revoke`); toast(`${c} revogado`); load(); } catch (e) { toast(e instanceof ApiError ? e.message : 'Algo correu mal.', true); }
  }
  const active = rows?.filter(i => i.status === 'active' && i.uses < i.max_uses).length ?? 0;
  const created = rows?.reduce((a, i) => a + i.used_by.length, 0) ?? 0;
  const darkInput: React.CSSProperties = { height: 44, padding: '0 12px', borderRadius: 12, border: '1px solid #3A3029', background: '#251E18', outline: 'none', fontSize: 14, color: '#F6EFE6' };
  return <>
    <Head kicker={`${active} códigos ativos · ${created} contas criadas`} title="Convites" lead="Só quem tem um código pode criar conta." />
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, alignItems: 'flex-start' }}>
      <section className="card-dark" style={{ flex: '1 1 300px', minWidth: 0, borderRadius: 22, padding: 20, display: 'flex', flexDirection: 'column', gap: 14 }}>
        <div className="display" style={{ fontWeight: 700, fontSize: 18, letterSpacing: '-0.02em' }}>Novo convite</div>
        <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}><span style={{ fontSize: 12.5, color: '#A89C8F' }}>Para quem (nota interna)</span><input value={note} onChange={e => setNote(e.target.value.slice(0, 40))} placeholder="Ex.: Banda Xitique" style={darkInput} /></label>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}><span style={{ fontSize: 12.5, color: '#A89C8F' }}>Utilizações</span><div style={{ display: 'grid', gridTemplateColumns: 'repeat(4,1fr)', gap: 6 }}>
          {[1, 5, 10, 25].map(u => <button key={u} onClick={() => setUses(u)} style={{ height: 38, borderRadius: 10, border: `1px solid ${uses === u ? '#FF6A1A' : '#3A3029'}`, background: uses === u ? '#FF6A1A' : 'transparent', color: uses === u ? '#1A1410' : '#F6EFE6', fontWeight: 700, fontSize: 13 }}>{u}</button>)}
        </div></div>
        <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}><span style={{ fontSize: 12.5, color: '#A89C8F' }}>Taxa ao registar</span><select value={fee} onChange={e => setFee(+e.target.value)} style={darkInput}><option value={1500}>15% · padrão</option><option value={1200}>12% · parceiro</option><option value={1000}>10% · lançamento</option></select></label>
        <button onClick={create} disabled={busy} style={{ height: 46, borderRadius: 13, border: 0, background: '#FF6A1A', color: '#1A1410', fontWeight: 700, fontSize: 14.5, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}><TicketPlus size={17} />Gerar código</button>
      </section>
      <section className="card" style={{ flex: '2.2 1 520px', minWidth: 0, borderRadius: 22, overflow: 'hidden' }}>
        {rows?.map((i, n) => {
          const full = i.uses >= i.max_uses;
          const st = i.status === 'revoked' ? ['Revogado', '#FDE8E4', '#B42318'] as const : full ? ['Esgotado', '#ECE6DB', '#57524B'] as const : ['Ativo', '#E3F3E8', '#17803D'] as const;
          return (
            <div key={i.code} style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: '10px 16px', padding: '14px 18px', borderTop: n ? '1px solid #F2EAE0' : 0 }}>
              <span style={{ flex: '1 1 160px', minWidth: 0, lineHeight: 1.35 }}><span style={{ display: 'flex', alignItems: 'center', gap: 8 }}><span className="mono" style={{ fontWeight: 600, fontSize: 14, letterSpacing: '.04em' }}>{i.code}</span>{i.code === fresh && <span style={{ height: 20, padding: '0 6px', borderRadius: 6, background: '#FF6A1A', color: '#1A1410', fontSize: 10.5, fontWeight: 700, display: 'flex', alignItems: 'center' }}>NOVO</span>}</span><span style={{ display: 'block', fontSize: 12.5, color: '#6E6861' }}>{i.note} · {longDate(i.created_at)}</span></span>
              <span style={{ flex: '1 1 140px', minWidth: 0, fontSize: 13, lineHeight: 1.35 }}><span style={{ display: 'block', fontWeight: 600 }}>{i.uses} de {i.max_uses} {i.max_uses === 1 ? 'utilização' : 'utilizações'}</span><span style={{ display: 'block', color: '#6E6861', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{i.used_by.length ? i.used_by.join(', ') : 'Ainda não usado'}</span></span>
              <span style={{ flex: 'none', height: 24, padding: '0 8px', borderRadius: 7, background: '#F6EFE6', fontSize: 12, fontWeight: 600, display: 'flex', alignItems: 'center' }}>{pct(i.fee_bps)}</span>
              <Badge st={st} dot={false} />
              <span style={{ flex: 'none', display: 'flex', gap: 6 }}>
                <button className="icon-btn" style={{ width: 34, height: 34, borderRadius: 10 }} title="Copiar link" aria-label="Copiar link" onClick={() => { copy(link(i.code)); toast('Link de convite copiado'); }}><Copy size={14} /></button>
                {i.status === 'active' && !full && <button className="icon-btn" style={{ width: 34, height: 34, borderRadius: 10, color: '#B42318' }} title="Revogar" aria-label="Revogar" onClick={() => revoke(i.code)}><Ban size={14} /></button>}
              </span>
            </div>
          );
        })}
        {rows && rows.length === 0 && <div style={{ padding: 40, textAlign: 'center', color: '#6E6861' }}>Ainda não há convites.</div>}
      </section>
    </div>
  </>;
}
