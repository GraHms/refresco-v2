import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ArrowRight, ArrowUpRight, HeartHandshake, Split, TrendingUp, UsersRound, Zap } from 'lucide-react';
import { api, SeriesPoint, SupportView } from '../../lib/api';
import { bucketLabel, CH, greeting, mt, mtN, pct, SEG, todayLabel, weekdayOf } from '../../lib/format';
import { Seg } from '../../components/ui';
import { PageHead, useBo } from './Layout';
import { SupportRow } from './Inbox';

interface Overview {
  range: string; gross: number; net: number; fee: number; gross_prev: number; count: number; count_prev: number; supporters: number;
  average: number; top_amount: number; by_channel: { mpesa: number; emola: number; bim: number; card: number }; series: SeriesPoint[]; bucket: 'day' | 'month';
  fee_bps: number; split: { member_id: string; name: string; pct: number; wallet: 'mpesa' | 'emola'; amount: number }[]; recent: SupportView[]; unread: number;
}

const RANGES: [string, string, string][] = [['7d', '7 dias', 'últimos 7 dias'], ['30d', '30 dias', 'últimos 30 dias'], ['12m', '12 meses', 'últimos 12 meses']];

function delta(cur: number, prev: number) {
  if (!prev) return cur ? 'novo' : '0%';
  const v = ((cur - prev) / prev) * 100;
  return (v >= 0 ? '+' : '') + v.toFixed(1).replace('.', ',') + '%';
}

export default function Overview() {
  const { me, mobile, setUnread } = useBo();
  const nav = useNavigate();
  const [range, setRange] = useState('30d');
  const [o, setO] = useState<Overview | null>(null);
  const [hot, setHot] = useState<number | null>(null);

  useEffect(() => { api.get<Overview>('/backoffice/overview?range=' + range).then(x => { setO(x); setUnread(x.unread); }).catch(() => {}); }, [range]);

  const R = RANGES.find(r => r[0] === range)!;
  const pts = o?.series || [];
  const max = Math.max(1, ...pts.map(p => p.total));
  const hp = hot != null ? pts[hot] : null;
  const month = o?.bucket === 'month';
  const n = pts.length;
  const barGap = n > 20 ? (mobile ? '2px' : '5px') : n > 10 ? '8px' : '14px';
  const tot = o?.by_channel || { mpesa: 0, emola: 0, bim: 0, card: 0 };
  // Cards only show if there ever were card payments.
  const shown = (['mpesa', 'emola', 'bim', 'card'] as const).filter(k => k !== 'card' || tot.card > 0);
  const sum = tot.mpesa + tot.emola + tot.bim + tot.card;
  const lbl = (p: SeriesPoint, i: number) => month ? (mobile ? (i % 3 === 0 ? bucketLabel(p.start, true, true) : '') : bucketLabel(p.start, true, true))
    : n <= 7 ? ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'][weekdayOf(p.start)] : weekdayOf(p.start) === 5 ? bucketLabel(p.start, false, true) : '';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <PageHead kicker={todayLabel()} title={`${greeting()}, ${me.member?.name || me.name}`} right={<>
        <Seg value={range} onChange={v => { setRange(v); setHot(null); }} options={RANGES.map(r => [r[0], r[1]] as [string, string])} />
        {!mobile && <a href={'/' + me.account!.handle} target="_blank" rel="noopener" className="btn">Ver página<ArrowUpRight size={16} /></a>}
      </>} />
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
        <div className="card-dark" style={{ flex: '2 1 400px', minWidth: 0, position: 'relative', overflow: 'hidden', padding: 24, display: 'flex', flexDirection: 'column', justifyContent: 'space-between', gap: 28, minHeight: 220 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 16 }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              <span style={{ fontSize: 14, color: '#A39C92' }}>Recebido · {R[2]}</span>
              <span className="mono" style={{ alignSelf: 'flex-start', padding: '4px 9px', borderRadius: 8, background: 'rgba(91,208,138,.14)', color: '#5BD08A', fontSize: 12 }}>{o ? delta(o.gross, o.gross_prev) : '…'} vs. período anterior</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'flex-end', gap: 7, flex: 'none' }} aria-hidden>
              <span style={{ width: 12, height: 12, borderRadius: '50%', background: '#FF6A1A' }} />
              <span style={{ width: 16, height: 34, borderRadius: 5, background: '#FF6A1A', marginBottom: 18 }} />
              <span style={{ display: 'flex', gap: 7 }}><span style={{ width: 34, height: 76, border: '7px solid #FF6A1A', borderRight: 0, borderRadius: '18px 0 0 18px' }} /><span style={{ width: 34, height: 76, border: '7px solid #FF6A1A', borderLeft: 0, borderRadius: '0 18px 18px 0' }} /></span>
            </div>
          </div>
          <div>
            <div className="display" style={{ display: 'flex', alignItems: 'baseline', flexWrap: 'wrap', gap: '4px 12px', fontWeight: 800, letterSpacing: '-0.04em' }}>
              <span style={{ fontSize: 'clamp(40px,4.6vw,62px)', lineHeight: 0.95 }}>{o ? mtN(o.gross) : '—'}</span><span style={{ fontSize: 22, color: '#A39C92' }}>MT</span>
            </div>
            <div style={{ marginTop: 10, fontSize: 14, color: '#A39C92' }}>{o ? `${mtN(o.net)} MT líquidos depois da taxa de ${pct(o.fee_bps)} · ${o.count} apoios` : ' '}</div>
          </div>
        </div>
        <Kpi label="Apoios" Icon={HeartHandshake} value={o ? String(o.count) : '—'} note={o ? delta(o.count, o.count_prev) + ' vs. período anterior' : ''} />
        <Kpi label="Apoiantes" Icon={UsersRound} value={o ? String(o.supporters) : '—'} note={o ? `${Math.max(0, o.count - o.supporters)} apoios de fãs que voltaram` : ''} />
        <Kpi label="Média por apoio" Icon={TrendingUp} value={o ? mt(o.average) : '—'} note={o && o.top_amount ? `Valor mais escolhido: ${mt(o.top_amount)}` : 'Ainda sem apoios'} />
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, alignItems: 'stretch' }}>
        <div className="card" style={{ flex: '2 1 520px', minWidth: 0, padding: 22, display: 'flex', flexDirection: 'column', gap: 18 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: 14 }}>
            <div><div style={{ fontSize: 14, color: '#57524B' }}>{hp ? bucketLabel(hp.start, month, false) : 'Total · ' + R[2]}</div>
              <div className="display" style={{ marginTop: 2, fontWeight: 800, fontSize: 26, letterSpacing: '-0.03em', lineHeight: 1.1 }}>{mt(hp ? hp.total : sum)}</div></div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px 16px', fontSize: 13 }}>
              {shown.map(k => (
                <span key={k} style={{ display: 'flex', alignItems: 'center', gap: 7 }}><span style={{ width: 10, height: 10, borderRadius: 3, background: CH[k].bar }} /><span style={{ color: '#57524B' }}>{CH[k].name}</span><span className="mono" style={{ fontSize: 12.5 }}>{mt(hp ? hp.by_channel[k] : tot[k])}</span></span>
              ))}
            </div>
          </div>
          <div style={{ position: 'relative' }}>
            <div style={{ position: 'absolute', inset: 0, display: 'flex', flexDirection: 'column', justifyContent: 'space-between', pointerEvents: 'none' }}>
              <span style={{ borderTop: '1px dashed #ECE7DF' }} /><span style={{ borderTop: '1px dashed #ECE7DF' }} /><span style={{ borderTop: '1px dashed #ECE7DF' }} /><span style={{ borderTop: '1px solid #E4DDD1' }} />
            </div>
            <div onMouseLeave={() => setHot(null)} style={{ position: 'relative', display: 'flex', alignItems: 'flex-end', gap: barGap, height: mobile ? 170 : 220 }} role="img" aria-label="Apoios por período e canal">
              {pts.map((p, i) => (
                <div key={i} onMouseEnter={() => setHot(i)} onClick={() => setHot(i)} style={{ flex: 1, minWidth: 0, height: '100%', display: 'flex', flexDirection: 'column', justifyContent: 'flex-end', cursor: 'pointer' }}>
                  <div style={{ height: (p.total / max * 100).toFixed(1) + '%', display: 'flex', flexDirection: 'column', gap: p.total ? 2 : 0, opacity: hot == null || hot === i ? 1 : 0.3, transition: 'height .5s cubic-bezier(.2,.8,.2,1),opacity .15s' }}>
                    <div style={{ flex: p.by_channel.card, background: '#CFC6B8', borderRadius: '5px 5px 2px 2px' }} />
                    <div style={{ flex: p.by_channel.bim, background: '#D0005C', borderRadius: 2 }} />
                    <div style={{ flex: p.by_channel.emola, background: '#F7871F', borderRadius: 2 }} />
                    <div style={{ flex: p.by_channel.mpesa, background: '#E60000', borderRadius: '2px 2px 4px 4px' }} />
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div style={{ display: 'flex', gap: barGap, marginTop: -8 }}>
            {pts.map((p, i) => <span key={i} className="mono" style={{ flex: 1, minWidth: 0, display: 'flex', justifyContent: 'center', whiteSpace: 'nowrap', fontSize: 11, color: '#6E6861' }}>{lbl(p, i)}</span>)}
          </div>
        </div>
        <div className="card" style={{ flex: '1 1 280px', minWidth: 0, padding: 22, display: 'flex', flexDirection: 'column', gap: 18 }}>
          <div style={{ fontWeight: 600, fontSize: 15 }}>Por canal</div>
          <div style={{ display: 'flex', gap: 4, height: 14 }}>
            {shown.map(k => <span key={k} style={{ flex: tot[k] || (sum ? 0 : 1), background: CH[k].bar, borderRadius: 4, transition: 'flex-grow .5s' }} />)}
          </div>
          <div>
            {shown.map(k => (
              <div key={k} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '12px 0', borderTop: '1px solid #F0EBE3' }}>
                <span style={{ width: 10, height: 10, borderRadius: 3, background: CH[k].bar }} />
                <span style={{ flex: 1, lineHeight: 1.3 }}><span style={{ display: 'block', fontWeight: 600 }}>{CH[k].name}</span><span style={{ display: 'block', fontSize: 12.5, color: '#6E6861' }}>{CH[k].op}</span></span>
                <span style={{ textAlign: 'right', lineHeight: 1.3 }}><span className="mono" style={{ display: 'block', fontSize: 13 }}>{mt(tot[k])}</span><span style={{ display: 'block', fontSize: 12.5, color: '#6E6861' }}>{sum ? Math.round(tot[k] / sum * 100) : 0}%</span></span>
              </div>
            ))}
          </div>
          <div style={{ marginTop: 'auto', display: 'flex', gap: 10, padding: '12px 14px', borderRadius: 14, background: '#F8F5EF', fontSize: 12.5, lineHeight: 1.45, color: '#57524B' }}><Zap size={15} style={{ marginTop: 1 }} /><span>Cada apoio é dividido na hora e fica no saldo de cada membro até pedir o levantamento.</span></div>
        </div>
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, alignItems: 'stretch' }}>
        <div className="card" style={{ flex: '2 1 520px', minWidth: 0, padding: '22px 14px 14px', display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0 8px 6px' }}>
            <span style={{ fontWeight: 600, fontSize: 15 }}>Últimos apoios</span>
            <Link to="/painel/apoios" style={{ display: 'flex', alignItems: 'center', gap: 6, fontWeight: 600, fontSize: 13, color: '#A83700' }}>Ver todos<ArrowRight size={15} /></Link>
          </div>
          {o && o.recent.length === 0 && <div style={{ padding: '30px 8px', textAlign: 'center', color: '#6E6861' }}>Ainda não há apoios. Partilha o teu link!</div>}
          {o?.recent.map(s => <SupportRow key={s.id} s={s} compact={mobile} onOpen={() => nav('/painel/apoios/' + s.id)} />)}
        </div>
        <div className="card" style={{ flex: '1 1 280px', minWidth: 0, padding: 22, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ fontWeight: 600, fontSize: 15 }}>Divisão do grupo</span>
            <span className="badge" style={{ background: '#E3F3E8', color: '#17803D' }}><Split size={13} />Automática</span>
          </div>
          <div style={{ display: 'flex', gap: 4, height: 40 }}>
            {o?.split.map((m, i) => <span key={m.member_id} className="mono" style={{ flex: m.pct, background: SEG[i % SEG.length][0], color: SEG[i % SEG.length][1], borderRadius: 10, display: 'flex', alignItems: 'center', padding: '0 10px', fontSize: 12, overflow: 'hidden' }}>{m.pct}%</span>)}
          </div>
          <div>
            {o?.split.map((m, i) => (
              <div key={m.member_id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '11px 0', borderTop: '1px solid #F0EBE3' }}>
                <span className="avatar" style={{ width: 34, height: 34, borderRadius: 11, background: SEG[i % SEG.length][0], color: SEG[i % SEG.length][1], fontSize: 12 }}>{m.name.slice(0, 2).toUpperCase()}</span>
                <span style={{ flex: 1, minWidth: 0, lineHeight: 1.3 }}><span style={{ display: 'block', fontWeight: 600 }}>{m.name}</span><span style={{ display: 'block', fontSize: 12.5, color: '#6E6861' }}>{m.pct}% · {CH[m.wallet].name}</span></span>
                <span className="mono" style={{ fontSize: 13 }}>{mt(m.amount, 2)}</span>
              </div>
            ))}
          </div>
          <div style={{ fontSize: 12.5, color: '#6E6861' }}>Valores líquidos, depois da taxa de {o ? pct(o.fee_bps) : '15%'}.</div>
          <Link to="/painel/grupo" className="btn" style={{ marginTop: 'auto' }}>Gerir divisão<ArrowRight size={15} /></Link>
        </div>
      </div>
    </div>
  );
}

function Kpi({ label, Icon, value, note }: { label: string; Icon: typeof Zap; value: string; note: string }) {
  return (
    <div className="card" style={{ flex: '1 1 170px', padding: 22, display: 'flex', flexDirection: 'column', justifyContent: 'space-between', gap: 18 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', color: '#57524B', fontSize: 14 }}><span>{label}</span><Icon size={18} /></div>
      <div><div className="display" style={{ fontWeight: 800, fontSize: 34, letterSpacing: '-0.035em', lineHeight: 1 }}>{value}</div><div style={{ marginTop: 8, fontSize: 13, color: '#6E6861' }}>{note}</div></div>
    </div>
  );
}
