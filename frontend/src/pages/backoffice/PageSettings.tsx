import { useEffect, useMemo, useState } from 'react';
import { ArrowUpRight, ChevronDown, CircleAlert, CircleCheck, CreditCard, Eye, Link as LinkIcon, Smartphone, X } from 'lucide-react';
import { api, ApiError, Page } from '../../lib/api';
import { CH, fmt, mtN } from '../../lib/format';
import { useSession } from '../../lib/session';
import { Brand, Logo, SOCIALS, Toggle, useToast } from '../../components/ui';
import { PageHead, useBo } from './Layout';

interface Resp { page: Page; categories: string[]; can_edit: boolean }

export default function PageSettings() {
  const { mobile, full } = useBo();
  const { refresh } = useSession();
  const toast = useToast();
  const [data, setData] = useState<Resp | null>(null);
  const [pg, setPg] = useState<Page | null>(null);
  const [saving, setSaving] = useState(false);
  const [pvOpen, setPvOpen] = useState(false);
  const [avail, setAvail] = useState<{ h: string; ok: boolean } | null>(null);

  useEffect(() => { api.get<Resp>('/backoffice/page').then(r => { setData(r); setPg(r.page); }).catch(() => {}); }, []);
  const dirty = useMemo(() => !!pg && !!data && JSON.stringify(pg) !== JSON.stringify(data.page), [pg, data]);

  useEffect(() => {
    if (!pg || !data || pg.handle === data.page.handle || !/^[a-z0-9.]{3,24}$/.test(pg.handle)) return;
    const h = pg.handle;
    const t = setTimeout(() => api.get<{ available: boolean }>('/handles/' + h).then(r => setAvail({ h, ok: r.available })).catch(() => {}), 300);
    return () => clearTimeout(t);
  }, [pg?.handle]);

  if (!pg || !data) return <div className="skeleton" style={{ height: 400 }} />;
  const set = (patch: Partial<Page>) => setPg(p => ({ ...p!, ...patch }));
  const saved = data.page;
  const h = pg.handle;
  const hs = !/^[a-z0-9.]{3,24}$/.test(h) ? { text: 'Usa 3 a 24 letras minúsculas, números ou pontos.', color: '#A83700', bad: true }
    : h === saved.handle ? { text: 'É o teu link atual.', color: '#6E6861', bad: false }
      : !avail || avail.h !== h ? { text: 'A verificar…', color: '#6E6861', bad: true }
        : !avail.ok ? { text: 'Este link já está ocupado.', color: '#A83700', bad: true } : { text: 'Disponível!', color: '#17803D', bad: false };
  const noChannel = !pg.channels.mpesa && !pg.channels.emola && !pg.channels.card;

  async function save() {
    if (hs.bad) { toast('Corrige o link antes de guardar', true); return; }
    setSaving(true);
    try {
      const r = await api.put<Resp>('/backoffice/page', pg);
      setData(r); setPg(r.page); toast('Página atualizada'); refresh();
    } catch (e) { toast(e instanceof ApiError ? e.message : 'Não foi possível guardar.', true); }
    finally { setSaving(false); }
  }

  const section: React.CSSProperties = { padding: 'clamp(16px,2vw,22px)', display: 'flex', flexDirection: 'column', gap: 16 };
  const head = (t: string, s: string) => <div><div style={{ fontWeight: 600, fontSize: 15 }}>{t}</div><div style={{ fontSize: 13, color: '#6E6861' }}>{s}</div></div>;
  const ro = !data.can_edit;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <PageHead kicker={'refres.co/' + saved.handle} title="A tua página" lead="Perfil, valores e redes, com pré-visualização ao vivo." right={<>
        {mobile && <button className="btn btn-dark" onClick={() => setPvOpen(true)}><Eye size={16} />Pré-visualizar</button>}
        <a className="btn" href={'/' + saved.handle} target="_blank" rel="noopener">Ver página<ArrowUpRight size={16} /></a>
      </>} />
      {ro && <div style={{ padding: '12px 16px', borderRadius: 16, background: '#FBEFD9', color: '#6B3F00', fontSize: 13.5 }}>Só o admin do grupo pode alterar a página.</div>}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 20, alignItems: 'flex-start' }}>
        <fieldset disabled={ro} style={{ flex: '1 1 480px', minWidth: 0, display: 'flex', flexDirection: 'column', gap: 16, border: 0, padding: 0, margin: 0 }}>
          <section className="card" style={{ ...section, gap: 18 }}>
            {head('Perfil', 'Como os fãs te veem.')}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(min(100%,220px),1fr))', gap: 14 }}>
              <label className="field"><span>Nome da página</span><input className="input" value={pg.name} onChange={e => set({ name: e.target.value.slice(0, 40) })} /></label>
              <label className="field"><span>Categoria</span>
                <span style={{ position: 'relative', display: 'block' }}>
                  <select className="input" value={pg.category} onChange={e => set({ category: e.target.value })} style={{ paddingRight: 40, appearance: 'none', WebkitAppearance: 'none' }}>
                    {data.categories.map(c => <option key={c}>{c}</option>)}
                  </select>
                  <ChevronDown size={16} color="#6E6861" style={{ position: 'absolute', right: 14, top: 15, pointerEvents: 'none' }} />
                </span>
              </label>
            </div>
            <label className="field"><span>Link da página</span>
              <span className="affix"><span className="pre">refres.co/</span><input className="mono" style={{ fontSize: 13.5 }} value={pg.handle} onChange={e => set({ handle: e.target.value.toLowerCase().replace(/[^a-z0-9.]/g, '').slice(0, 24) })} /></span>
              <span style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12.5, color: hs.color }}>{hs.color === '#17803D' ? <CircleCheck size={14} /> : hs.color === '#A83700' ? <CircleAlert size={14} /> : <LinkIcon size={14} />}{hs.text}</span>
            </label>
            <label className="field"><span style={{ display: 'flex', justifyContent: 'space-between' }}><span>Bio</span><span className="mono" style={{ fontSize: 12, fontWeight: 400, color: '#6E6861' }}>{pg.bio.length}/200</span></span>
              <textarea className="input" rows={3} value={pg.bio} onChange={e => set({ bio: e.target.value.slice(0, 200) })} /></label>
          </section>

          <section className="card" style={section}>
            {head('Valores sugeridos', 'Os quatro valores que os fãs veem primeiro. Marca um como sugerido.')}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(min(100%,130px),1fr))', gap: 10 }}>
              {pg.amounts.map((v, i) => {
                const on = pg.default_idx === i;
                return (
                  <div key={i} style={{ display: 'flex', flexDirection: 'column', gap: 10, padding: 12, borderRadius: 16, background: on ? '#FFF4EC' : '#F8F5EF', border: `1.5px solid ${on ? '#D14400' : '#ECE7DF'}`, transition: 'border-color .2s,background-color .2s' }}>
                    <span style={{ display: 'flex', alignItems: 'center', height: 42, borderRadius: 11, border: '1px solid #E4DDD1', background: '#fff', overflow: 'hidden' }}>
                      <input type="tel" inputMode="numeric" aria-label={`Valor ${i + 1}`} value={v ? fmt(v / 100) : ''} onChange={e => { const a = [...pg.amounts]; a[i] = (parseInt(e.target.value.replace(/\D/g, '').slice(0, 7), 10) || 0) * 100; set({ amounts: a }); }}
                        className="display" style={{ flex: 1, minWidth: 0, height: '100%', border: 0, outline: 'none', padding: '0 10px', fontWeight: 800, fontSize: 17, background: 'transparent' }} />
                      <span className="mono" style={{ paddingRight: 10, fontSize: 12, color: '#6E6861' }}>MT</span>
                    </span>
                    <button type="button" onClick={() => set({ default_idx: i })} style={{ display: 'flex', alignItems: 'center', gap: 7, border: 0, background: 'transparent', padding: 0, fontSize: 12.5, fontWeight: 600, color: on ? '#A83700' : '#57524B' }}>
                      <span style={{ width: 16, height: 16, borderRadius: '50%', border: `2px solid ${on ? '#D14400' : '#CFC6B8'}`, display: 'grid', placeItems: 'center' }}><span style={{ width: 7, height: 7, borderRadius: '50%', background: on ? '#D14400' : 'transparent' }} /></span>
                      {on ? 'Sugerido' : 'Marcar sugerido'}
                    </button>
                  </div>
                );
              })}
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 14, paddingTop: 4 }}>
              <span style={{ flex: '1 1 220px', lineHeight: 1.35 }}><span style={{ display: 'block', fontWeight: 600 }}>Permitir outro valor</span><span style={{ display: 'block', fontSize: 13, color: '#6E6861' }}>O fã pode escrever o valor que quiser.</span></span>
              {pg.allow_custom && <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: '#57524B' }}>Mínimo
                <span style={{ display: 'flex', alignItems: 'center', height: 38, width: 110, borderRadius: 10, border: '1px solid #E4DDD1', overflow: 'hidden', background: '#fff' }}>
                  <input type="tel" inputMode="numeric" value={pg.min_amount / 100 || ''} onChange={e => set({ min_amount: (parseInt(e.target.value.replace(/\D/g, '').slice(0, 6), 10) || 0) * 100 })} className="mono" style={{ flex: 1, minWidth: 0, height: '100%', border: 0, outline: 'none', padding: '0 10px', fontSize: 13.5 }} />
                  <span className="mono" style={{ paddingRight: 10, fontSize: 12, color: '#6E6861' }}>MT</span>
                </span></label>}
              <Toggle on={pg.allow_custom} onClick={() => set({ allow_custom: !pg.allow_custom })} label="Permitir outro valor" />
            </div>
          </section>

          <section className="card" style={{ ...section, gap: 14 }}>
            {head('Redes sociais', 'Só aparecem na página as que preencheres.')}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(min(100%,240px),1fr))', gap: 10 }}>
              {SOCIALS.map(so => (
                <label key={so.key} style={{ display: 'flex', alignItems: 'center', height: 46, border: '1px solid #E4DDD1', borderRadius: 12, overflow: 'hidden', background: '#fff' }}>
                  <span title={so.label} style={{ flex: 'none', width: 46, height: '100%', display: 'grid', placeItems: 'center', borderRight: '1px solid #ECE7DF', background: '#F8F5EF' }}><Brand name={so.key} size={16} /></span>
                  <span className="mono" style={{ paddingLeft: 12, fontSize: 12.5, color: '#6E6861', whiteSpace: 'nowrap' }}>{so.prefix}</span>
                  <input aria-label={so.label} value={pg.socials[so.key] || ''} onChange={e => set({ socials: { ...pg.socials, [so.key]: e.target.value.replace(/\s/g, '').slice(0, 80) } })} placeholder={so.ph}
                    className="mono" style={{ flex: 1, minWidth: 0, border: 0, outline: 'none', height: '100%', padding: '0 12px 0 1px', fontSize: 12.5, background: 'transparent' }} />
                </label>
              ))}
            </div>
          </section>

          <section className="card" style={{ ...section, gap: 6 }}>
            <div style={{ marginBottom: 6 }}>{head('Formas de pagamento', 'Como os fãs podem pagar.')}</div>
            {(['mpesa', 'emola', 'card'] as const).map(k => (
              <div key={k} style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '12px 0', borderTop: '1px solid #F0EBE3' }}>
                <span style={{ flex: 'none', width: 40, height: 40, borderRadius: 12, background: k === 'mpesa' ? '#D14400' : k === 'emola' ? '#1A1410' : '#ECE6DB', color: k === 'card' ? '#1A1410' : '#fff', display: 'grid', placeItems: 'center' }}>{k === 'card' ? <CreditCard size={18} /> : <Smartphone size={18} />}</span>
                <span style={{ flex: 1, minWidth: 0, lineHeight: 1.35 }}>
                  <span style={{ display: 'block', fontWeight: 600 }}>{k === 'card' ? 'Cartão bancário' : CH[k].name}</span>
                  <span style={{ display: 'block', fontSize: 13, color: '#6E6861' }}>{k === 'mpesa' ? 'Fãs Vodacom confirmam com o PIN M-Pesa' : k === 'emola' ? 'Fãs Movitel confirmam com o PIN e-Mola' : 'Visa e Mastercard, com confirmação do banco'}</span>
                </span>
                <Toggle on={pg.channels[k]} onClick={() => set({ channels: { ...pg.channels, [k]: !pg.channels[k] } })} label={CH[k].name} />
              </div>
            ))}
            {noChannel && <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '10px 12px', borderRadius: 12, background: '#FDE9DC', color: '#A83700', fontSize: 13, fontWeight: 600 }}><CircleAlert size={15} />Ativa pelo menos uma forma de pagamento.</div>}
          </section>

          <section className="card" style={{ ...section, gap: 12 }}>
            {head('Mensagem de agradecimento', 'Aparece ao fã logo depois do pagamento.')}
            <textarea className="input" rows={3} value={pg.thanks} onChange={e => set({ thanks: e.target.value.slice(0, 200) })} />
          </section>
        </fieldset>

        {(!mobile || pvOpen) && (
          <div style={mobile ? { position: 'fixed', inset: 0, zIndex: 85, background: 'rgba(20,18,16,.72)', padding: 16, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12, overflowY: 'auto' }
            : { position: 'sticky', top: 24, flex: '0 1 340px', minWidth: 300, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12 }}>
            <div style={{ width: '100%', maxWidth: 340, display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 4px' }}>
              <span className="mono" style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 11.5, letterSpacing: '.08em', textTransform: 'uppercase', color: mobile ? '#F6EFE6' : '#6E6861' }}><span style={{ width: 7, height: 7, borderRadius: '50%', background: '#17803D' }} />Pré-visualização ao vivo</span>
              {mobile && <button className="icon-btn" style={{ width: 36, height: 36, borderRadius: 11, border: 0 }} onClick={() => setPvOpen(false)} aria-label="Fechar"><X size={16} /></button>}
            </div>
            <Preview pg={pg} />
          </div>
        )}
      </div>
      {dirty && !ro && (
        <div className="savebar" style={{ left: mobile ? '50%' : full ? 'calc(50% + 126px)' : 'calc(50% + 38px)' }}>
          <span style={{ width: 8, height: 8, borderRadius: '50%', background: '#FF6A1A' }} />{!mobile && <span style={{ fontWeight: 500 }}>Tens alterações por guardar</span>}
          <button className="btn-outline-dark" onClick={() => setPg(data.page)}>Descartar</button>
          <button className="btn btn-primary" style={{ height: 38, borderRadius: 11, fontSize: 13.5 }} onClick={save} disabled={saving || noChannel}>{saving ? 'A guardar…' : 'Guardar'}</button>
        </div>
      )}
    </div>
  );
}

function Preview({ pg }: { pg: Page }) {
  const socials = SOCIALS.filter(s => (pg.socials[s.key] || '').trim());
  return (
    <div style={{ width: '100%', maxWidth: 340, height: 640, flex: 'none', borderRadius: 44, background: '#1A1410', padding: 10, boxShadow: '0 30px 60px -30px rgba(20,18,16,.45)' }}>
      <div style={{ height: '100%', borderRadius: 35, background: '#F6EFE6', overflowY: 'auto', overflowX: 'hidden' }}>
        <div style={{ padding: '16px 14px 24px', display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}><Logo h={20} /><span style={{ height: 24, padding: '0 9px', borderRadius: 8, border: '1px solid #D9D1C4', background: '#fff', fontSize: 10.5, fontWeight: 600, display: 'flex', alignItems: 'center' }}>Cria a tua página</span></div>
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', gap: 8, padding: '6px 4px 0' }}>
            <div className="mono" style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 9.5, letterSpacing: '.1em', textTransform: 'uppercase', color: '#6E6861' }}><span style={{ width: 6, height: 6, borderRadius: '50%', background: '#FF6A1A' }} />{pg.category} · Moçambique</div>
            <div className="display" style={{ fontWeight: 800, fontSize: 24, lineHeight: 1.02, letterSpacing: '-0.04em', overflowWrap: 'anywhere' }}>{pg.name || 'O teu nome'}</div>
            {pg.bio && <p style={{ margin: 0, fontSize: 12, lineHeight: 1.5, color: '#57524B' }}>{pg.bio}</p>}
            {socials.length > 0 && <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', justifyContent: 'center' }}>{socials.map(s => <span key={s.key} style={{ width: 30, height: 30, borderRadius: 9, background: '#fff', border: '1px solid #E4DDD1', display: 'grid', placeItems: 'center' }}><Brand name={s.key} size={13} /></span>)}</div>}
          </div>
          <div style={{ background: '#fff', border: '1px solid #E4DDD1', borderRadius: 18, padding: 14, display: 'flex', flexDirection: 'column', gap: 10 }}>
            <div className="display" style={{ fontWeight: 700, fontSize: 14, lineHeight: 1.15, letterSpacing: '-0.02em' }}>Apoia o {pg.name || 'O teu nome'}</div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2,minmax(0,1fr))', gap: 6 }}>
              {pg.amounts.map((v, i) => { const on = i === pg.default_idx; return (
                <span key={i} style={{ display: 'flex', alignItems: 'baseline', gap: 4, padding: 10, borderRadius: 11, background: on ? '#D14400' : '#F8F5EF', color: on ? '#fff' : '#1A1410', border: `1px solid ${on ? '#D14400' : '#ECE7DF'}` }}>
                  <span className="display" style={{ fontWeight: 800, fontSize: 15, lineHeight: 1 }}>{mtN(v)}</span><span className="mono" style={{ fontSize: 9.5, color: on ? 'rgba(255,255,255,.85)' : '#6E6861' }}>MT</span>
                </span>); })}
            </div>
            {pg.allow_custom && <span style={{ height: 32, borderRadius: 10, background: '#F8F5EF', border: '1px solid #ECE7DF', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 10px', fontSize: 10.5, color: '#57524B' }}><span>Outro valor</span><span className="mono">mín. {mtN(pg.min_amount)} MT</span></span>}
            <div style={{ display: 'flex', gap: 4, justifyContent: 'center', flexWrap: 'wrap' }}>
              {(['mpesa', 'emola', 'card'] as const).filter(k => pg.channels[k]).map(k => <span key={k} style={{ padding: '2px 6px', borderRadius: 5, background: CH[k].bg, color: CH[k].fg, fontSize: 9.5, fontWeight: 600 }}>{CH[k].name}</span>)}
            </div>
            <span style={{ height: 36, borderRadius: 11, background: '#D14400', color: '#fff', fontWeight: 600, fontSize: 12, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>Pagar {mtN(pg.amounts[pg.default_idx] || 0)} MT</span>
          </div>
          <div style={{ padding: 12, borderRadius: 14, background: '#1A1410', color: '#F6EFE6', fontSize: 11.5, lineHeight: 1.45 }}>
            <span className="mono" style={{ display: 'block', fontSize: 9, letterSpacing: '.08em', textTransform: 'uppercase', color: '#A39C92', marginBottom: 4 }}>Depois do pagamento</span>{pg.thanks}
          </div>
        </div>
      </div>
    </div>
  );
}
