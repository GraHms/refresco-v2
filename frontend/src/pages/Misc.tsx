import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowRight, CircleCheck, Split, Smartphone, Wallet } from 'lucide-react';
import { api, ApiError } from '../lib/api';
import { SEG } from '../lib/format';
import { useSession } from '../lib/session';
import { Logo } from '../components/ui';
import { homeFor } from './Auth';

export function Landing() {
  const { me } = useSession();
  useEffect(() => { document.title = 'refres.co · Os teus fãs pagam-te um refresco'; }, []);
  const feats = [
    [Smartphone, 'M-Pesa, e-Mola e cartão', 'O fã escolhe o valor e confirma com o PIN no telemóvel. Sem apps, sem registo.'],
    [Split, 'Divisão automática', 'Grupos e bandas dividem cada apoio na hora, pela percentagem que todos aprovaram.'],
    [Wallet, 'Levanta quando quiseres', 'O saldo acumula. Pedes o levantamento para o M-Pesa ou para o banco, pago em até 72 h.'],
  ] as const;
  return (
    <div style={{ minHeight: '100vh', background: '#F6EFE6' }}>
      <header style={{ maxWidth: 1180, margin: '0 auto', padding: '20px clamp(16px,3vw,32px)', display: 'flex', alignItems: 'center', gap: 12 }}>
        <Logo h={36} /><span style={{ flex: 1 }} />
        {me ? <Link className="btn btn-dark" to={homeFor(me)}>Abrir painel</Link> : <><Link className="btn btn-ghost" to="/entrar">Entrar</Link><Link className="btn btn-primary" to="/registar">Criar página</Link></>}
      </header>
      <main style={{ maxWidth: 1180, margin: '0 auto', padding: 'clamp(24px,6vw,80px) clamp(16px,3vw,32px) 60px', display: 'flex', flexDirection: 'column', gap: 48 }}>
        <section style={{ display: 'flex', flexDirection: 'column', gap: 22, maxWidth: 820 }}>
          <div className="kicker"><i />Acesso por convite · Moçambique</div>
          <h1 className="display" style={{ margin: 0, fontWeight: 800, fontSize: 'clamp(44px,7vw,92px)', lineHeight: 0.95, letterSpacing: '-0.05em' }}>Os teus fãs pagam-te um refresco.</h1>
          <p style={{ margin: 0, fontSize: 19, lineHeight: 1.55, color: '#57524B', maxWidth: 620 }}>Um link para receberes apoios dos teus fãs em M-Pesa, e-Mola ou cartão. Para criadores, podcasts, bandas e coletivos.</p>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            <Link className="btn btn-primary" style={{ height: 54, padding: '0 22px', borderRadius: 15, fontSize: 16 }} to="/registar">Tenho um convite<ArrowRight size={17} /></Link>
            <Link className="btn" style={{ height: 54, padding: '0 22px', borderRadius: 15, fontSize: 16 }} to="/tupratu">Ver um exemplo</Link>
          </div>
        </section>
        <section style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(min(100%,280px),1fr))', gap: 16 }}>
          {feats.map(([I, t, d], i) => (
            <div key={t} className={i === 1 ? 'card-dark' : 'card'} style={{ padding: 24, display: 'flex', flexDirection: 'column', gap: 14 }}>
              <span style={{ width: 44, height: 44, borderRadius: 14, background: i === 1 ? '#FF6A1A' : '#FDE9DC', color: i === 1 ? '#1A1410' : '#A83700', display: 'grid', placeItems: 'center' }}><I size={20} /></span>
              <div className="display" style={{ fontWeight: 700, fontSize: 21, letterSpacing: '-0.02em' }}>{t}</div>
              <div style={{ fontSize: 15, lineHeight: 1.55, color: i === 1 ? '#CFC6B8' : '#57524B' }}>{d}</div>
            </div>
          ))}
        </section>
        <section className="card" style={{ padding: 'clamp(20px,3vw,32px)', display: 'flex', flexWrap: 'wrap', gap: 20, alignItems: 'center', justifyContent: 'space-between' }}>
          <div><div className="display" style={{ fontWeight: 800, fontSize: 28, letterSpacing: '-0.03em' }}>15% por apoio. Mais nada.</div><div style={{ color: '#57524B', marginTop: 6 }}>Sem mensalidade, sem taxa de levantamento.</div></div>
          <Link className="btn btn-dark" style={{ height: 50, padding: '0 20px', borderRadius: 14 }} to="/registar">Criar a minha página</Link>
        </section>
      </main>
      <footer style={{ textAlign: 'center', padding: '24px 16px 40px', fontSize: 12.5, color: '#6E6861' }}>© 2026 Refres, Lda · Maputo</footer>
    </div>
  );
}

interface Approval { account: string; member: string; splits: { id: string; name: string; pct: number }[]; approved: boolean; applied?: boolean }

export function SplitApproval() {
  const { token = '' } = useParams();
  const [a, setA] = useState<Approval | null>(null);
  const [err, setErr] = useState(''); const [done, setDone] = useState<Approval | null>(null); const [busy, setBusy] = useState(false);
  useEffect(() => { api.get<Approval>('/split-approvals/' + encodeURIComponent(token)).then(setA).catch(e => setErr(e instanceof ApiError ? e.message : 'Link inválido.')); }, [token]);
  async function approve() {
    setBusy(true);
    try { setDone(await api.post<Approval>('/split-approvals/' + encodeURIComponent(token))); } catch (e) { setErr(e instanceof ApiError ? e.message : 'Algo correu mal.'); } finally { setBusy(false); }
  }
  return (
    <div style={{ minHeight: '100vh', background: '#F6EFE6', display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '24px 20px' }}>
      <Link to="/"><Logo h={32} /></Link>
      <div className="card" style={{ marginTop: 32, width: '100%', maxWidth: 460, padding: 24, borderRadius: 28, display: 'flex', flexDirection: 'column', gap: 16 }}>
        {err ? <><div className="display" style={{ fontWeight: 800, fontSize: 24 }}>Link indisponível</div><div style={{ color: '#57524B' }}>{err}</div></> :
          done || a?.approved ? <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', gap: 12 }}>
            <CircleCheck size={40} color="#17803D" /><div className="display" style={{ fontWeight: 800, fontSize: 24 }}>Aprovado</div>
            <div style={{ color: '#57524B' }}>{done?.applied ? 'Todos aprovaram. A nova divisão já está ativa.' : 'Obrigado. A divisão muda quando todos aprovarem.'}</div>
          </div> : !a ? <div className="skeleton" style={{ height: 200 }} /> : <>
            <div className="kicker"><i />{a.account}</div>
            <div className="display" style={{ fontWeight: 800, fontSize: 26, letterSpacing: '-0.03em', lineHeight: 1.1 }}>Olá {a.member}, aprovas a nova divisão?</div>
            <div style={{ display: 'flex', gap: 5, height: 64 }}>
              {a.splits.filter(s => s.pct > 0).map((s, i) => <div key={s.id} style={{ flex: s.pct, background: SEG[i % SEG.length][0], color: SEG[i % SEG.length][1], borderRadius: 14, padding: '10px 12px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between', overflow: 'hidden' }}><b className="display">{s.pct}%</b><span style={{ fontSize: 12.5, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{s.name}</span></div>)}
            </div>
            {a.splits.map(s => <div key={s.id} style={{ display: 'flex', justifyContent: 'space-between', borderTop: '1px solid #F0EBE3', paddingTop: 10 }}><span>{s.name}</span><b>{s.pct}%</b></div>)}
            <button className="btn btn-primary" style={{ height: 52, borderRadius: 14, fontSize: 15 }} onClick={approve} disabled={busy}>Aprovar divisão</button>
          </>}
      </div>
    </div>
  );
}
