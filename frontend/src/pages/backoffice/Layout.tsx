import { createContext, useContext, useEffect, useState } from 'react';
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { ArrowUpRight, ChevronsUpDown, Copy, Inbox, LayoutDashboard, Link as LinkIcon, LogOut, PanelsTopLeft, UsersRound, Wallet } from 'lucide-react';
import { api, Me } from '../../lib/api';
import { ini } from '../../lib/format';
import { SUPPORT_PHONE } from '../../lib/support';
import { useSession } from '../../lib/session';
import { copy, Logo, useMedia, useToast } from '../../components/ui';

const NAV = [
  ['visao-geral', 'Visão geral', 'Início', LayoutDashboard], ['apoios', 'Apoios', 'Apoios', Inbox], ['pagina', 'A tua página', 'Página', PanelsTopLeft],
  ['grupo', 'Grupo e divisão', 'Grupo', UsersRound], ['levantamentos', 'Levantamentos', 'Saldo', Wallet],
] as const;

interface BoCtx { me: Me; unread: number; setUnread: (n: number) => void; mobile: boolean; full: boolean; groupSize: number; setGroupSize: (n: number) => void }
const Ctx = createContext<BoCtx>(null!);
export const useBo = () => useContext(Ctx);

export default function BackofficeLayout() {
  const { me, logout } = useSession();
  const nav = useNavigate();
  const loc = useLocation();
  const toast = useToast();
  const mobile = useMedia('(max-width: 759px)');
  const full = useMedia('(min-width: 1080px)');
  const [unread, setUnread] = useState(0);
  const [groupSize, setGroupSize] = useState(1);

  useEffect(() => {
    if (!me) return;
    api.get<{ unread: number }>('/backoffice/supports?limit=1').then(r => setUnread(r.unread)).catch(() => {});
    api.get<{ members: unknown[] }>('/backoffice/group').then(g => setGroupSize(g.members.length)).catch(() => {});
  }, [me?.id]);
  useEffect(() => { document.title = 'Painel · Refresco'; }, []);

  if (!me || !me.account) return null;
  const handle = me.account.handle;
  const link = `${location.origin}/${handle}`;
  const copyLink = () => { copy(link); toast('Link copiado: pagarefresco.com/' + handle); };
  const out = async () => { await logout(); nav('/entrar', { replace: true }); };

  return (
    <Ctx.Provider value={{ me, unread, setUnread, mobile, full, groupSize, setGroupSize }}>
      <div style={{ minHeight: '100vh', display: 'flex', background: '#F6EFE6' }}>
        {!mobile && (
          <aside style={{ position: 'sticky', top: 0, height: '100vh', width: full ? 252 : 76, flex: 'none', display: 'flex', flexDirection: 'column', gap: 20, padding: '20px 14px', borderRight: '1px solid #E4DDD1', background: '#EEE9E0', overflowY: 'auto' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: full ? 'flex-start' : 'center', padding: '0 6px', height: 40 }}><Link to="/"><Logo h={full ? 32 : 20} /></Link></div>
            {full && (
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: 10, borderRadius: 16, background: '#fff', border: '1px solid #E4DDD1' }}>
                <span className="display" style={{ flex: 'none', width: 36, height: 36, borderRadius: 11, background: '#D14400', color: '#fff', display: 'grid', placeItems: 'center', fontWeight: 900, fontSize: 11 }}>{ini(me.account.name).slice(0, 3)}</span>
                <span style={{ flex: 1, minWidth: 0, lineHeight: 1.25 }}>
                  <span style={{ display: 'block', fontWeight: 600, fontSize: 14, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{me.account.name}</span>
                  <span style={{ display: 'block', fontSize: 12, color: '#6E6861' }}>{groupSize > 1 ? `Grupo · ${groupSize} membros` : 'Individual'}</span>
                </span>
                <ChevronsUpDown size={16} color="#6E6861" />
              </div>
            )}
            <nav style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              {NAV.map(([id, label, , Icon]) => (
                <NavLink key={id} to={'/painel/' + id} title={label} style={({ isActive }) => ({ display: 'flex', alignItems: 'center', justifyContent: full ? 'flex-start' : 'center', gap: 12, height: 44, padding: '0 12px', borderRadius: 13, background: isActive ? '#fff' : 'transparent', boxShadow: isActive ? '0 1px 2px rgba(20,18,16,.08), 0 0 0 1px #E4DDD1' : 'none', color: isActive ? '#1A1410' : '#57524B', fontWeight: isActive ? 600 : 500, fontSize: 14 })}>
                  <Icon size={19} />
                  {full && <span style={{ flex: 1 }}>{label}</span>}
                  {full && id === 'apoios' && unread > 0 && <span style={{ minWidth: 20, height: 20, padding: '0 6px', borderRadius: 7, background: '#D14400', color: '#fff', fontSize: 11.5, fontWeight: 700, display: 'grid', placeItems: 'center' }}>{unread}</span>}
                </NavLink>
              ))}
            </nav>
            <div style={{ flex: 1 }} />
            {full && (
              <div className="card-dark" style={{ padding: 16, borderRadius: 18, display: 'flex', flexDirection: 'column', gap: 12 }}>
                <div className="mono" style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 11, letterSpacing: '.08em', textTransform: 'uppercase', color: '#A39C92' }}>
                  <span style={{ width: 7, height: 7, borderRadius: '50%', background: me.account_status === 'suspended' ? '#FF6A1A' : '#5BD08A' }} />{me.account_status === 'suspended' ? 'Página suspensa' : 'Página ativa'}
                </div>
                <div className="mono" style={{ fontSize: 13, wordBreak: 'break-all' }}>pagarefresco.com/{handle}</div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                  <button onClick={copyLink} style={{ height: 36, borderRadius: 10, border: '1px solid #3A3531', background: 'transparent', color: '#F6EFE6', fontSize: 13, fontWeight: 600, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}><Copy size={14} />Copiar</button>
                  <a href={'/' + handle} target="_blank" rel="noopener" style={{ height: 36, borderRadius: 10, background: '#D14400', color: '#fff', fontSize: 13, fontWeight: 600, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}>Abrir<ArrowUpRight size={14} /></a>
                </div>
              </div>
            )}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: full ? 'flex-start' : 'center', gap: 10, padding: 6 }}>
              <span className="avatar" style={{ width: 36, height: 36, borderRadius: 11, background: '#1A1410', color: '#fff', fontSize: 12 }}>{ini(me.member?.name || me.name)}</span>
              {full && <><span style={{ flex: 1, minWidth: 0, lineHeight: 1.25 }}><span style={{ display: 'block', fontWeight: 600, fontSize: 13.5 }}>{me.member?.name}</span><span style={{ display: 'block', fontSize: 12, color: '#6E6861' }}>{me.member?.is_admin ? 'Admin' : 'Membro'}</span></span>
                <button onClick={out} title="Sair" aria-label="Sair" style={{ border: 0, background: 'transparent', display: 'grid', color: '#6E6861' }}><LogOut size={16} /></button></>}
            </div>
          </aside>
        )}
        <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          {mobile && (
            <div style={{ position: 'sticky', top: 0, zIndex: 40, display: 'flex', alignItems: 'center', gap: 12, padding: '12px 16px', background: 'rgba(243,239,231,.92)', backdropFilter: 'blur(12px)', WebkitBackdropFilter: 'blur(12px)', borderBottom: '1px solid #E4DDD1' }}>
              <Logo h={26} /><span style={{ flex: 1 }} />
              <button onClick={copyLink} className="mono" style={{ display: 'flex', alignItems: 'center', gap: 6, height: 34, padding: '0 10px', borderRadius: 10, border: '1px solid #D9D1C4', background: '#fff', fontSize: 11.5 }}><LinkIcon size={13} />/{handle}</button>
              <button onClick={out} aria-label="Sair" className="avatar" style={{ width: 34, height: 34, borderRadius: 11, background: '#1A1410', color: '#fff', fontSize: 12, border: 0 }}>{ini(me.member?.name || me.name)}</button>
            </div>
          )}
          <main style={{ flex: 1, minWidth: 0, padding: mobile ? '18px 14px 110px' : '28px clamp(16px,3vw,40px) 110px' }}>
            <div style={{ maxWidth: 1180, margin: '0 auto' }}>
              {me.sandbox && <Notice>Modo de teste: os pagamentos são simulados e o saldo não é dinheiro real.</Notice>}
              {me.account_status === 'pending' && !loc.pathname.endsWith('/pagina') && <Notice>A tua conta está por verificar. Já podes receber apoios; os levantamentos são aprovados depois da verificação.</Notice>}
              {me.account_status === 'suspended' && <Notice red>A tua conta está suspensa. A página não recebe apoios. Fala com a equipa Refresco: {SUPPORT_PHONE}.</Notice>}
              <Outlet />
            </div>
          </main>
        </div>
        {mobile && (
          <nav style={{ position: 'fixed', left: 0, right: 0, bottom: 0, zIndex: 45, display: 'grid', gridTemplateColumns: 'repeat(5,1fr)', padding: '8px 8px 10px', background: 'rgba(255,255,255,.94)', backdropFilter: 'blur(14px)', WebkitBackdropFilter: 'blur(14px)', borderTop: '1px solid #E4DDD1' }}>
            {NAV.map(([id, , short, Icon]) => {
              const on = loc.pathname.startsWith('/painel/' + id);
              return (
                <NavLink key={id} to={'/painel/' + id} style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3, height: 52, justifyContent: 'center', color: on ? '#1A1410' : '#6E6861', fontSize: 11, fontWeight: on ? 700 : 500 }}>
                  <span style={{ position: 'relative', width: 44, height: 28, borderRadius: 10, background: on ? '#FDE9DC' : 'transparent', display: 'grid', placeItems: 'center' }}>
                    <Icon size={19} />
                    {id === 'apoios' && unread > 0 && <span style={{ position: 'absolute', top: -4, right: 2, minWidth: 17, height: 17, padding: '0 4px', borderRadius: 6, background: '#D14400', color: '#fff', fontSize: 10, fontWeight: 700, display: 'grid', placeItems: 'center' }}>{unread}</span>}
                  </span>{short}
                </NavLink>
              );
            })}
          </nav>
        )}
      </div>
    </Ctx.Provider>
  );
}

function Notice({ children, red }: { children: React.ReactNode; red?: boolean }) {
  return <div style={{ marginBottom: 16, padding: '12px 16px', borderRadius: 16, background: red ? '#FDE8E4' : '#FBEFD9', color: red ? '#7A1F14' : '#6B3F00', fontSize: 13.5 }}>{children}</div>;
}

export function PageHead({ kicker, title, lead, right }: { kicker: string; title: string; lead?: string; right?: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', flexWrap: 'wrap', gap: 16, marginBottom: 8 }}>
      <div>
        <div className="kicker"><i />{kicker}</div>
        <h1 className="h1">{title}</h1>
        {lead && <p className="lead" style={{ maxWidth: 620 }}>{lead}</p>}
      </div>
      {right && <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>{right}</div>}
    </div>
  );
}
