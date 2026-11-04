import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { CircleCheck } from 'lucide-react';
import { api, ApiError } from '../lib/api';
import { SEG } from '../lib/format';
import { Logo } from '../components/ui';

interface Approval { account: string; member: string; splits: { id: string; name: string; pct: number }[]; approved: boolean; applied?: boolean }

export function SplitApproval() {
  const [token] = useState(() => decodeURIComponent(location.hash.slice(1)));
  const [a, setA] = useState<Approval | null>(null);
  const [err, setErr] = useState(''); const [done, setDone] = useState<Approval | null>(null); const [busy, setBusy] = useState(false);
  useEffect(() => {
    try { history.replaceState(null, '', '/aprovar'); } catch { /* ignore */ }
    if (!token) { setErr('Link inválido.'); return; }
    api.post<Approval>('/split-approvals/lookup', { token }).then(setA).catch(e => setErr(e instanceof ApiError ? e.message : 'Link inválido.'));
  }, [token]);
  async function approve() {
    setBusy(true);
    try { setDone(await api.post<Approval>('/split-approvals/approve', { token })); } catch (e) { setErr(e instanceof ApiError ? e.message : 'Algo correu mal.'); } finally { setBusy(false); }
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
