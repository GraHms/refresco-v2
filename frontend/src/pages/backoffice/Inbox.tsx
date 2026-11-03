import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { CircleAlert, Copy, CreditCard, Download, Hourglass, Mail, Phone, Search, Smartphone, X } from 'lucide-react';
import { api, Channel, SupportView } from '../../lib/api';
import { avOf, CH, dateTime, ini, mt, pct, relative, SEG, SUP_ST } from '../../lib/format';
import { Badge, Brand, BrandName, copy, Drawer, Seg, useToast } from '../../components/ui';
import { PageHead, useBo } from './Layout';

interface List { items: SupportView[]; total: number; all: number; unread: number; channels: Record<Channel, number> }

const NETS: Record<string, BrandName> = { x: 'x', instagram: 'instagram', tiktok: 'tiktok', facebook: 'facebook', youtube: 'youtube' };

export function SupportRow({ s, compact, onOpen }: { s: SupportView; compact?: boolean; onOpen: () => void }) {
  const [bg, fg] = avOf(s.id);
  const name = s.name || 'Anónimo';
  return (
    <button onClick={onOpen} className="row-btn hover-paper" style={{ padding: '11px 8px', borderRadius: 14 }}>
      <span className="avatar" style={{ width: 40, height: 40, borderRadius: 13, background: bg, color: fg, fontSize: 13 }}>{ini(name)}</span>
      <span style={{ flex: 1, minWidth: 0, lineHeight: 1.35 }}>
        <span style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 0 }}>
          {s.net && NETS[s.net] && <Brand name={NETS[s.net]} size={13} />}
          <span style={{ fontWeight: s.read || s.status !== 'confirmed' ? 600 : 700, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{name}</span>
          <span style={{ flex: 'none', width: 7, height: 7, borderRadius: '50%', background: !s.read && s.status === 'confirmed' ? '#D14400' : 'transparent' }} />
          <span style={{ flex: 'none', fontSize: 12.5, color: '#6E6861' }}>{relative(s.created_at)}</span>
        </span>
        <span style={{ display: 'block', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', color: s.message ? '#57524B' : '#9A938A', fontStyle: s.message ? 'normal' : 'italic' }}>{s.message || 'Sem mensagem'}</span>
      </span>
      {!compact && <span className="badge" style={{ background: CH[s.channel].bg, color: CH[s.channel].fg }}>{CH[s.channel].name}</span>}
      <span className="display" style={{ flex: 'none', minWidth: 76, textAlign: 'right', fontWeight: 700, fontSize: 15, whiteSpace: 'nowrap' }}>{mt(s.amount)}</span>
    </button>
  );
}

export default function Inbox() {
  const { id } = useParams();
  const nav = useNavigate();
  const { mobile, setUnread } = useBo();
  const [q, setQ] = useState(''); const [dq, setDq] = useState('');
  const [chF, setChF] = useState<'all' | Channel>('all');
  const [stF, setStF] = useState<'all' | 'confirmed' | 'pending' | 'failed'>('all');
  const [list, setList] = useState<List | null>(null);
  const [limit, setLimit] = useState(50);
  const [tick, setTick] = useState(0);

  useEffect(() => { const t = setTimeout(() => setDq(q), 250); return () => clearTimeout(t); }, [q]);
  const qs = () => {
    const p = new URLSearchParams();
    if (dq.trim()) p.set('q', dq.trim());
    if (chF !== 'all') p.set('channel', chF);
    if (stF !== 'all') p.set('status', stF);
    return p;
  };
  useEffect(() => {
    const p = qs(); p.set('limit', String(limit));
    api.get<List>('/backoffice/supports?' + p).then(l => { setList(l); setUnread(l.unread); }).catch(() => {});
  }, [dq, chF, stF, limit, tick]);

  const rows = list?.items || [];
  const cols = 'minmax(0,1fr) 104px 100px 210px 124px';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <PageHead kicker={list ? `${list.total} de ${list.all} apoios · ${list.unread} por ler` : 'Apoios'} title="Apoios" lead="Mensagens, valores, canais e contactos de quem te apoia."
        right={<a className="btn" href={'/api/v1/backoffice/supports.csv?' + qs()} download><Download size={16} />Exportar CSV</a>} />
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, alignItems: 'center' }}>
        <label style={{ flex: '1 1 260px', display: 'flex', alignItems: 'center', gap: 10, height: 44, padding: '0 14px', borderRadius: 13, border: '1px solid #E4DDD1', background: '#fff' }}>
          <Search size={17} color="#6E6861" />
          <input value={q} onChange={e => setQ(e.target.value)} placeholder="Procurar por nome, mensagem ou contacto" aria-label="Procurar" style={{ flex: 1, minWidth: 0, border: 0, outline: 'none', background: 'transparent', fontSize: 14 }} />
        </label>
        <div style={{ display: 'flex', gap: 6, overflowX: 'auto', maxWidth: '100%' }}>
          {([['all', 'Todos'], ['mpesa', 'M-Pesa'], ['emola', 'e-Mola'], ['card', 'Cartão']] as const).map(([k, l]) => (
            <button key={k} className={'chip' + (chF === k ? ' on' : '')} onClick={() => setChF(k)}>{l}<span className="mono" style={{ fontSize: 11.5, opacity: 0.65 }}>{list ? (k === 'all' ? list.all : list.channels[k] || 0) : ''}</span></button>
          ))}
        </div>
        <Seg value={stF} onChange={setStF} options={[['all', 'Todos'], ['confirmed', 'Confirmados'], ['pending', 'Pendentes'], ['failed', 'Falhados']]} />
      </div>
      {!mobile ? (
        <div className="card" style={{ borderRadius: 22, overflow: 'hidden' }}>
          <div style={{ overflowX: 'auto' }}><div style={{ minWidth: 880 }}>
            <div className="mono" style={{ display: 'grid', gridTemplateColumns: cols, gap: 16, padding: '12px 20px', background: '#FBF9F5', fontSize: 11, letterSpacing: '.08em', textTransform: 'uppercase', color: '#6E6861' }}>
              <span style={{ paddingLeft: 66 }}>Apoiante e mensagem</span><span>Valor</span><span>Canal</span><span>Contacto</span><span>Estado</span>
            </div>
            {rows.map(s => {
              const [bg, fg] = avOf(s.id), unread = !s.read && s.status === 'confirmed';
              return (
                <button key={s.id} onClick={() => nav('/painel/apoios/' + s.id)} className="hover-paper" style={{ display: 'grid', gridTemplateColumns: cols, gap: 16, alignItems: 'center', width: '100%', padding: '14px 20px', border: 0, borderTop: '1px solid #F0EBE3', background: id === s.id ? '#FCF5EF' : '#fff', textAlign: 'left' }}>
                  <span style={{ display: 'flex', gap: 12, alignItems: 'center', minWidth: 0 }}>
                    <span style={{ flex: 'none', width: 8, height: 8, borderRadius: '50%', background: unread ? '#D14400' : 'transparent' }} />
                    <span className="avatar" style={{ width: 40, height: 40, borderRadius: 13, background: bg, color: fg, fontSize: 13 }}>{ini(s.name || 'Anónimo')}</span>
                    <span style={{ minWidth: 0, flex: 1, lineHeight: 1.35 }}>
                      <span style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 0 }}>{s.net && NETS[s.net] && <Brand name={NETS[s.net]} size={13} />}<span style={{ fontWeight: unread ? 700 : 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.name || 'Anónimo'}</span><span style={{ flex: 'none', fontSize: 12.5, color: '#6E6861', whiteSpace: 'nowrap' }}>{relative(s.created_at)}</span></span>
                      <span style={{ display: 'block', color: s.message ? '#57524B' : '#9A938A', fontStyle: s.message ? 'normal' : 'italic', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.message || 'Sem mensagem'}</span>
                    </span>
                  </span>
                  <span style={{ lineHeight: 1.3 }}><span className="display" style={{ display: 'block', fontWeight: 700, fontSize: 15, whiteSpace: 'nowrap' }}>{mt(s.amount)}</span><span style={{ display: 'block', fontSize: 12, color: '#6E6861' }}>{s.preset ? 'Valor sugerido' : 'Valor livre'}</span></span>
                  <span><span className="badge" style={{ background: CH[s.channel].bg, color: CH[s.channel].fg }}>{CH[s.channel].name}</span></span>
                  <span className="mono" style={{ fontSize: 12.5, color: '#2B2723', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.contact}</span>
                  <span><Badge st={SUP_ST[s.status]} /></span>
                </button>
              );
            })}
          </div></div>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {rows.map(s => {
            const [bg, fg] = avOf(s.id), unread = !s.read && s.status === 'confirmed';
            return (
              <button key={s.id} onClick={() => nav('/painel/apoios/' + s.id)} style={{ display: 'flex', flexDirection: 'column', gap: 10, width: '100%', padding: 14, borderRadius: 18, border: '1px solid #E4DDD1', background: '#fff', textAlign: 'left' }}>
                <span style={{ display: 'flex', alignItems: 'center', gap: 10, width: '100%' }}>
                  <span className="avatar" style={{ width: 38, height: 38, borderRadius: 12, background: bg, color: fg, fontSize: 12.5 }}>{ini(s.name || 'Anónimo')}</span>
                  <span style={{ flex: 1, minWidth: 0, lineHeight: 1.3 }}>
                    <span style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 0 }}>{s.net && NETS[s.net] && <Brand name={NETS[s.net]} size={12} />}<span style={{ fontWeight: unread ? 700 : 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.name || 'Anónimo'}</span><span style={{ flex: 'none', width: 7, height: 7, borderRadius: '50%', background: unread ? '#D14400' : 'transparent' }} /></span>
                    <span style={{ display: 'block', fontSize: 12, color: '#6E6861' }}>{relative(s.created_at)}</span>
                  </span>
                  <span className="display" style={{ fontWeight: 800, fontSize: 17, whiteSpace: 'nowrap' }}>{mt(s.amount)}</span>
                </span>
                <span style={{ display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden', fontSize: 14, lineHeight: 1.45, color: s.message ? '#57524B' : '#9A938A', fontStyle: s.message ? 'normal' : 'italic' }}>{s.message || 'Sem mensagem'}</span>
                <span style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap', width: '100%' }}>
                  <span className="badge" style={{ height: 24, fontSize: 12, background: CH[s.channel].bg, color: CH[s.channel].fg }}>{CH[s.channel].name}</span>
                  <Badge st={SUP_ST[s.status]} style={{ height: 24, fontSize: 12 }} />
                  <span className="mono" style={{ flex: 1, minWidth: 0, textAlign: 'right', fontSize: 11.5, color: '#57524B', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.contact}</span>
                </span>
              </button>
            );
          })}
        </div>
      )}
      {list && rows.length === 0 && <div className="card" style={{ padding: '56px 20px', textAlign: 'center', color: '#6E6861', borderRadius: 22 }}><div style={{ fontWeight: 600, color: '#1A1410' }}>Nenhum apoio encontrado</div><div style={{ marginTop: 4 }}>Experimenta outro filtro ou limpa a pesquisa.</div></div>}
      {list && rows.length < list.total && <button className="btn" style={{ alignSelf: 'center' }} onClick={() => setLimit(l => Math.min(200, l + 50))}>Mostrar mais</button>}
      {id && <Detail id={id} onClose={() => nav('/painel/apoios')} onChanged={() => setTick(t => t + 1)} />}
    </div>
  );
}

function Detail({ id, onClose, onChanged }: { id: string; onClose: () => void; onChanged: () => void }) {
  const { me } = useBo();
  const toast = useToast();
  const [s, setS] = useState<SupportView | null>(null);
  const [missing, setMissing] = useState(false);
  useEffect(() => { setS(null); api.get<SupportView>('/backoffice/supports/' + id).then(x => { setS(x); onChanged(); }).catch(() => setMissing(true)); }, [id]);

  if (missing) return <Drawer onClose={onClose} label="Apoio"><div style={{ padding: 24 }}>Apoio não encontrado. <button className="link-btn" onClick={onClose}>Fechar</button></div></Drawer>;
  if (!s) return <Drawer onClose={onClose} label="Apoio"><div style={{ padding: 24 }}><div className="skeleton" style={{ height: 200 }} /></div></Drawer>;

  const [bg, fg] = avOf(s.id);
  const name = s.name || 'Anónimo';
  const isPhone = s.channel !== 'card';
  const first = name.startsWith('@') ? name : name.split(' ')[0];
  const page = me.account!.name;
  const wa = `https://wa.me/${s.contact.replace(/\D/g, '')}?text=${encodeURIComponent(`Olá ${first}! Obrigado pelo apoio ao ${page}. Kanimambo!`)}`;
  const unread = async () => { await api.post(`/backoffice/supports/${s.id}/unread`).catch(() => {}); onChanged(); onClose(); toast('Marcado como não lido'); };
  const row = (k: string, v: string, strong?: boolean) => (
    <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, padding: '10px 14px', borderTop: '1px solid #F0EBE3', background: strong ? '#FBF9F5' : undefined, fontWeight: strong ? 600 : undefined }}>
      <span style={{ color: strong ? undefined : '#6E6861' }}>{k}</span><span className="mono">{v}</span>
    </div>
  );

  return (
    <Drawer onClose={onClose} label={'Apoio de ' + name}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '18px 18px 16px', borderBottom: '1px solid #F0EBE3' }}>
        <span className="avatar" style={{ width: 44, height: 44, borderRadius: 14, background: bg, color: fg, fontSize: 14 }}>{ini(name)}</span>
        <span style={{ flex: 1, minWidth: 0, lineHeight: 1.3 }}>
          <span style={{ display: 'flex', alignItems: 'center', gap: 6, fontWeight: 600, fontSize: 16 }}>{s.net && NETS[s.net] && <Brand name={NETS[s.net]} size={14} />}{name}</span>
          <span style={{ display: 'block', fontSize: 12.5, color: '#6E6861' }}>{dateTime(s.created_at)} · {relative(s.created_at)}</span>
        </span>
        <button className="icon-btn" onClick={onClose} aria-label="Fechar"><X size={17} /></button>
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 18px', display: 'flex', flexDirection: 'column', gap: 18 }}>
        <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', gap: 12 }}>
          <div><div className="display" style={{ fontWeight: 800, fontSize: 40, letterSpacing: '-0.04em', lineHeight: 1 }}>{mt(s.amount)}</div><div style={{ marginTop: 6, fontSize: 13, color: '#6E6861' }}>{s.preset ? 'Valor sugerido' : 'Valor livre'} · {CH[s.channel].name}</div></div>
          <Badge st={SUP_ST[s.status]} style={{ height: 28, padding: '0 10px', borderRadius: 9 }} />
        </div>
        {s.message ? (
          <div style={{ position: 'relative', padding: '18px 18px 18px 20px', borderRadius: 18, background: '#F8F5EF' }}>
            <span style={{ position: 'absolute', left: 0, top: 18, bottom: 18, width: 3, borderRadius: '0 3px 3px 0', background: '#FF6A1A' }} />
            <div style={{ fontSize: 17, lineHeight: 1.5, whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{s.message}</div>
          </div>
        ) : <div style={{ padding: '14px 16px', borderRadius: 16, background: '#F8F5EF', color: '#6E6861', fontStyle: 'italic' }}>Este apoio veio sem mensagem.</div>}
        {(s.status === 'failed' || s.status === 'expired') && <div style={{ display: 'flex', gap: 10, padding: '12px 14px', borderRadius: 14, background: '#FDE9DC', color: '#7A2E0A', fontSize: 13 }}><CircleAlert size={16} style={{ marginTop: 1 }} /><span><b>{s.fail_reason || 'Falhou'}.</b> O pagamento não foi concluído e nada foi dividido.</span></div>}
        {s.status === 'pending' && <div style={{ display: 'flex', gap: 10, padding: '12px 14px', borderRadius: 14, background: '#FBEFD9', color: '#6B3F00', fontSize: 13 }}><Hourglass size={16} style={{ marginTop: 1 }} /><span>{s.channel === 'card' ? 'O banco do fã ainda não confirmou o pagamento (3-D Secure).' : 'O fã ainda não confirmou com o PIN.'}</span></div>}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div className="label-sm">Contacto</div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '12px 14px', borderRadius: 14, border: '1px solid #ECE7DF' }}>
            {isPhone ? <Smartphone size={17} color="#6E6861" /> : <CreditCard size={17} color="#6E6861" />}
            <span className="mono" style={{ flex: 1, minWidth: 0, fontSize: 13.5, overflowWrap: 'anywhere' }}>{s.contact}</span>
            {isPhone && <button className="link-btn" style={{ display: 'flex', alignItems: 'center', gap: 5, fontSize: 12.5 }} onClick={() => { copy(s.contact); toast('Contacto copiado'); }}><Copy size={14} />Copiar</button>}
          </div>
          {isPhone && <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
            <a className="btn" href={'tel:' + s.contact.replace(/\s/g, '')}><Phone size={15} />Ligar</a>
            <a className="btn" href={wa} target="_blank" rel="noopener noreferrer"><Brand name="whatsapp" size={15} />WhatsApp</a>
          </div>}
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div className="label-sm">Pagamento</div>
          <div style={{ borderRadius: 14, border: '1px solid #ECE7DF', overflow: 'hidden', fontSize: 13.5 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, padding: '10px 14px' }}><span style={{ color: '#6E6861' }}>Referência</span><span className="mono">{s.ref}</span></div>
            {row('Meio', s.card || `${CH[s.channel].name} · ${s.contact.replace('+258 ', '')}`)}
            {row('Valor pago', mt(s.amount, 2))}
            {s.status === 'confirmed' && <>{row(`Taxa refres (${pct(s.fee_bps)})`, '−' + mt(s.fee, 2))}{row('Líquido para o grupo', mt(s.net_amount, 2), true)}</>}
          </div>
        </div>
        {s.splits && s.splits.length > 0 && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <div className="label-sm">Divisão deste apoio</div>
            {s.splits.map((m, i) => (
              <div key={m.member_id} style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <span className="avatar" style={{ width: 32, height: 32, borderRadius: 10, background: SEG[i % SEG.length][0], color: SEG[i % SEG.length][1], fontSize: 11 }}>{ini(m.name)}</span>
                <span style={{ flex: 1, minWidth: 0, fontWeight: 600 }}>{m.name} <span style={{ fontWeight: 400, color: '#6E6861' }}>· {m.pct}%</span></span>
                <span style={{ textAlign: 'right', lineHeight: 1.3 }}><span className="mono" style={{ display: 'block', fontSize: 13 }}>{mt(m.amount, 2)}</span><span style={{ display: 'block', fontSize: 12, fontWeight: 600, color: '#17803D' }}>No saldo</span></span>
              </div>
            ))}
          </div>
        )}
      </div>
      <div style={{ display: 'flex', gap: 8, padding: '14px 18px 18px', borderTop: '1px solid #F0EBE3' }}>
        <button className="btn" style={{ flex: 1, height: 46, borderRadius: 13 }} onClick={unread}>Marcar não lido</button>
        {isPhone ? <a className="btn btn-primary" style={{ flex: 1.3, height: 46, borderRadius: 13 }} href={wa} target="_blank" rel="noopener noreferrer"><Brand name="whatsapp" size={15} />Agradecer no WhatsApp</a>
          : <a className="btn btn-primary" style={{ flex: 1.3, height: 46, borderRadius: 13 }} href={'/' + me.account!.handle} target="_blank" rel="noopener"><Mail size={15} />Ver a página</a>}
      </div>
    </Drawer>
  );
}
