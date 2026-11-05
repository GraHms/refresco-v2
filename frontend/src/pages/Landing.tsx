import { FormEvent, ReactNode, useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ArrowRight, ChartColumn, ChevronDown, Check, CircleAlert, CircleCheck, CreditCard, CupSoda, History, Inbox, Link as LinkIcon, Mic, Minus, PanelsTopLeft, Plus, ShieldCheck, Zap } from 'lucide-react';
import { api } from '../lib/api';
import { digits, fmt, fmtPhone, walletOf } from '../lib/format';
import { useSession } from '../lib/session';
import { SUPPORT_EMAIL, SUPPORT_MAILTO } from '../lib/support';
import { Brand, Logo, PoweredByRiha } from '../components/ui';
import { homeFor } from './Auth';
import './landing.css';

const DOMAIN = 'pagarefresco.com';
const FEE = 0.15;

const FACTS = ['O fã paga com M-Pesa', 'Sem app e sem registo', 'Sem mensalidade', 'Valores em meticais', 'Levantas em até 72 horas', 'Cada membro do grupo recebe a sua parte', 'Feito em Maputo'];

const SPLIT_EXAMPLES: [string, number][] = [['Hélio Tembe', 300], ['Ivânia Cuamba', 500], ['@arsenio.mz', 100], ['Shelsia Manhiça', 500], ['Lurdes Novela', 1000]];
const SPLIT_MEMBERS: [string, number][] = [['Cardo', 70], ['Produção', 20], ['Edição', 10]];

const FAQ: [string, string][] = [
  ['Os fãs precisam de criar conta?', 'Não. Escolhem um valor, assinam com o nome ou com o @ e confirmam com o PIN do M-Pesa no telemóvel.'],
  ['Como recebo o dinheiro?', 'Os apoios acumulam no teu saldo. Pedes o levantamento no painel, para o M-Pesa, e é pago em até 72 horas. O levantamento para o banco chega em breve.'],
  ['Como crio uma conta?', `Por agora o Refresco funciona por convite. Se recebeste um código, cria a conta em ${DOMAIN}/registar. Se não, entra na lista de espera nessa mesma página.`],
  ['Quanto custa?', 'O Refresco fica com 15% de cada apoio recebido. Não há mensalidade, custo de adesão nem taxa de levantamento.'],
  ['Como funciona a divisão para grupos?', 'O grupo define as percentagens uma vez. Cada apoio é dividido no momento do pagamento e cada parte entra no saldo de cada membro. Mudar a divisão exige a aprovação de todos.'],
  ['Posso usar com e-Mola, Tmcel ou cartão?', 'Ainda não. Por agora os fãs pagam com M-Pesa, em números começados por 84 ou 85. O e-Mola e o cartão chegam em breve.'],
];

// Keeps names with a hyphen on one line.
function nb(text: string): ReactNode {
  return text.split(/(e-Mola|M-Pesa|Dá-lhe)/).map((t, i) => (i % 2 ? <span key={i} className="lp-nb">{t}</span> : t));
}

function Eyebrow({ children, light }: { children: ReactNode; light?: boolean }) {
  return <div className={'lp-eyebrow' + (light ? ' lp-eyebrow-light' : '')}><i />{children}</div>;
}

function Tick({ children, icon }: { children: ReactNode; icon?: ReactNode }) {
  return <span className="lp-tick">{icon || <Check size={18} color="#D14400" />}<span>{children}</span></span>;
}

type HandleState = { text: string; tone: 'quiet' | 'bad' | 'ok'; blocked: boolean };

function useHandle() {
  const [handle, setHandle] = useState('');
  const [avail, setAvail] = useState<{ h: string; ok: boolean | null } | null>(null);
  const wellFormed = /^[a-z0-9.]{3,24}$/.test(handle);
  useEffect(() => {
    if (!wellFormed) return;
    let live = true;
    const t = setTimeout(() => {
      api.get<{ available: boolean }>('/handles/' + handle)
        .then(r => { if (live) setAvail({ h: handle, ok: r.available }); })
        // The check is only a convenience: the registration form checks again.
        .catch(() => { if (live) setAvail({ h: handle, ok: null }); });
    }, 350);
    return () => { live = false; clearTimeout(t); };
  }, [handle, wellFormed]);
  const known = avail && avail.h === handle ? avail.ok : undefined;
  const state: HandleState = !handle ? { text: 'Escolhe o teu link. Para o criar, precisas de um convite.', tone: 'quiet', blocked: false }
    : !wellFormed ? { text: 'Usa 3 a 24 letras minúsculas, números ou pontos.', tone: 'bad', blocked: true }
      : known === undefined ? { text: 'A verificar…', tone: 'quiet', blocked: false }
        : known === null ? { text: 'Confirmamos se está livre quando criares a conta.', tone: 'quiet', blocked: false }
          : known ? { text: `${DOMAIN}/${handle} está disponível!`, tone: 'ok', blocked: false }
            : { text: `${DOMAIN}/${handle} não está disponível.`, tone: 'bad', blocked: true };
  return { handle, state, onChange: (v: string) => setHandle(v.toLowerCase().replace(/[^a-z0-9.]/g, '').slice(0, 24)) };
}

function ClaimForm({ h, dark, id }: { h: ReturnType<typeof useHandle>; dark?: boolean; id: string }) {
  const nav = useNavigate();
  function submit(e: FormEvent) {
    e.preventDefault();
    if (h.state.blocked) return;
    nav(h.handle ? '/registar?handle=' + encodeURIComponent(h.handle) : '/registar');
  }
  return (
    <form className={'lp-claim' + (dark ? ' lp-claim-on-brand' : '')} onSubmit={submit}>
      <label htmlFor={id} className="lp-claim-field">
        <span className="mono lp-claim-prefix">{DOMAIN}/</span>
        <input id={id} className="mono" value={h.handle} onChange={e => h.onChange(e.target.value)} placeholder="o.teu.nome" autoCapitalize="none" autoCorrect="off" spellCheck={false} aria-label="O teu link" aria-describedby={id + '-hint'} />
      </label>
      <button type="submit" className={'lp-btn lp-claim-go ' + (dark ? 'lp-btn-dark' : 'lp-btn-primary')}>Criar a minha página<ArrowRight size={18} /></button>
    </form>
  );
}

function HandleHint({ state, id }: { state: HandleState; id: string }) {
  const Icon = state.tone === 'ok' ? CircleCheck : state.tone === 'bad' ? CircleAlert : LinkIcon;
  return <span id={id} role="status" className={'lp-hint lp-hint-' + state.tone}><Icon size={15} />{state.text}</span>;
}

function HeroScene() {
  return (
    <div className="lp-scene" aria-hidden="true">
      <div className="lp-scene-mark"><span /><span /></div>
      <div className="lp-mock-page">
        <div className="lp-row">
          <span className="lp-avatar display">TPT</span>
          <span className="lp-mock-who"><b>Tu Pra Tu Podcast</b><span className="mono">{DOMAIN}/tupratu</span></span>
        </div>
        <div className="display lp-mock-title">Apoia o Tu Pra Tu</div>
        <div className="lp-mock-tiers">
          <span className="lp-mock-tier"><b className="display">100</b><span className="mono">MT</span></span>
          <span className="lp-mock-tier on"><span><b className="display">300</b><span className="mono">MT</span></span><em>Sugerido</em></span>
          <span className="lp-mock-tier"><b className="display">500</b><span className="mono">MT</span></span>
          <span className="lp-mock-tier"><b className="display">1.000</b><span className="mono">MT</span></span>
        </div>
        <span className="lp-mock-social"><Brand name="x" size={13} /><ChevronDown size={12} color="#6E6861" />@helio.tembe</span>
        <span className="lp-mock-pay">Apoiar com 300 MT<ArrowRight size={15} /></span>
      </div>
      <div className="lp-mock-notice">
        <div className="lp-row lp-mock-notice-head"><span className="lp-mock-r display">r</span><b>Refresco</b><span>agora</span></div>
        <div>Novo apoio de <b>300 MT</b> de @helio.tembe. Já está no saldo do grupo.</div>
      </div>
      <div className="lp-mock-split">
        <div className="lp-row lp-between"><b>Dividido na hora</b><span className="lp-mock-instant"><Zap size={11} />No saldo</span></div>
        <div className="lp-mock-bar"><span style={{ flex: 70, background: '#D14400' }} /><span style={{ flex: 20, background: '#1A1410' }} /><span style={{ flex: 10, background: '#D9D1C4' }} /></div>
        <div className="lp-row lp-between"><span>Cardo · 70%</span><span className="mono">178,50 MT</span></div>
        <div className="lp-row lp-between"><span>Produção · 20%</span><span className="mono">51,00 MT</span></div>
        <div className="lp-row lp-between"><span>Edição · 10%</span><span className="mono">25,50 MT</span></div>
      </div>
      <div className="lp-mock-message">
        <div>“Episódio brutal. Continuem com as conversas de Tu Pra Tu!”</div>
        <div className="lp-mock-message-by">@helio.tembe · 300 MT</div>
      </div>
    </div>
  );
}

function FactsBand() {
  const items = (hidden: boolean) => FACTS.map(t => <span key={t + hidden} aria-hidden={hidden || undefined}><i />{t}</span>);
  return <div className="lp-band"><div className="lp-band-track">{items(false)}{items(true)}</div></div>;
}

function Steps() {
  return (
    <section id="como-funciona" className="lp-section">
      <Eyebrow>Como funciona</Eyebrow>
      <h2 className="lp-h2" style={{ maxWidth: 820 }}>Três passos. Nenhum é complicado.</h2>
      <div className="lp-grid3">
        <div className="lp-card">
          <span className="display lp-num">01</span>
          <div><div className="display lp-card-title">Cria a tua página</div><p>Escolhe o teu link, os valores sugeridos e as tuas redes. Leva dois minutos.</p></div>
          <div className="lp-card-foot lp-card-foot-col">
            <span className="mono lp-mock-input"><span>{DOMAIN}/</span>meliacanta</span>
            <span className="lp-hint lp-hint-ok lp-hint-sm"><CircleCheck size={14} />Disponível</span>
          </div>
        </div>
        <div className="lp-card">
          <span className="display lp-num">02</span>
          <div><div className="display lp-card-title">Partilha o link</div><p>Na bio do Instagram, no estado do WhatsApp, no fim de cada episódio ou concerto.</p></div>
          <div className="lp-card-foot lp-chips">
            <span><Brand name="instagram" size={14} />Bio</span>
            <span><Brand name="whatsapp" size={14} />Estado</span>
            <span><Brand name="youtube" size={14} />Descrição</span>
            <span className="on"><Mic size={14} />No palco</span>
          </div>
        </div>
        <div className="lp-card">
          <span className="display lp-num lp-num-brand">03</span>
          <div><div className="display lp-card-title">Levanta quando quiseres</div><p>{nb('Os apoios acumulam no teu saldo, com a mensagem de quem os enviou. Levantas para o M-Pesa. O banco chega em breve.')}</p></div>
          <div className="lp-card-foot lp-withdraw">
            <span className="lp-withdraw-m display">M</span>
            <span className="lp-withdraw-text"><b>Levantar 5.000 MT</b><span>Para M-Pesa · 84 ··· 0457</span></span>
            <span className="mono lp-withdraw-when">Até 72 h</span>
          </div>
        </div>
      </div>
    </section>
  );
}

function Payments() {
  const [phone, setPhone] = useState('');
  const op = walletOf(phone);
  const demo = op === 'mpesa' ? { label: 'M-Pesa · Vodacom', bg: '#D14400', fg: '#FFFFFF' }
    : op === 'emola' ? { label: 'e-Mola · Movitel, em breve', bg: '#1A1410', fg: '#FFFFFF' }
      : op === 'tmcel' ? { label: 'Tmcel: ainda não suportada', bg: '#FBEFD9', fg: '#9A5B00' }
        : phone.length >= 2 ? { label: 'Número inválido', bg: '#FDE9DC', fg: '#A83700' }
          : { label: 'Escreve um número', bg: '#ECE6DB', fg: '#57524B' };
  return (
    <section id="pagamentos" className="lp-section">
      <div className="lp-section-head">
        <div>
          <Eyebrow>Pagamentos</Eyebrow>
          <h2 className="lp-h2" style={{ maxWidth: 760 }}>Os fãs pagam como já pagam tudo.</h2>
        </div>
        <p className="lp-muted" style={{ maxWidth: 380, margin: 0 }}>{nb('Sem conta e sem app nova. Basta o número de telemóvel e o PIN do M-Pesa.')}</p>
      </div>
      <div className="lp-grid3">
        <div className="lp-pay lp-pay-mpesa">
          <div className="lp-row lp-between lp-top"><span className="lp-pay-mark display">M</span><span className="mono lp-pay-tag">84 · 85</span></div>
          <div><div className="display lp-pay-name">M-Pesa</div><div className="lp-pay-sub">Vodacom</div></div>
          <div className="lp-pay-steps">
            <span><i className="mono">1</i>Recebe o pedido no telemóvel</span>
            <span><i className="mono">2</i>Confirma com o PIN M-Pesa</span>
            <span><i className="mono">3</i>Pronto, em segundos</span>
          </div>
        </div>
        <div className="lp-pay lp-pay-emola">
          <div className="lp-row lp-between lp-top"><span className="lp-pay-mark display">e</span><span className="mono lp-pay-tag">86 · 87</span></div>
          <div><div className="display lp-pay-name">e-Mola</div><div className="lp-pay-sub">Movitel</div></div>
          <div className="lp-pay-soon"><span className="lp-soon">Em breve</span>Até lá, os pagamentos são por M-Pesa.</div>
        </div>
        <div className="lp-pay lp-pay-card">
          <div className="lp-row lp-between lp-top"><span className="lp-pay-mark"><CreditCard size={24} /></span><span className="lp-pay-brands"><Brand name="visa" size={18} color="#1A1F71" /><Brand name="mastercard" size={16} color="#EB001B" /></span></div>
          <div><div className="display lp-pay-name">Cartão</div><div className="lp-pay-sub">Visa e Mastercard</div></div>
          <div className="lp-pay-soon"><span className="lp-soon lp-soon-light">Em breve</span>Até lá, os pagamentos são por M-Pesa.</div>
        </div>
      </div>
      <div className="lp-detect">
        <label htmlFor="lp-phone" className="lp-detect-label">Detetamos o operador pelo número. Experimenta:</label>
        <span className="lp-phone"><span className="mono">+258</span><input id="lp-phone" className="mono" type="tel" inputMode="numeric" value={fmtPhone(phone)} onChange={e => setPhone(digits(e.target.value, 9))} placeholder="84 123 4567" /></span>
        <span className="lp-detect-out"><ArrowRight size={16} color="#6E6861" /><span role="status" style={{ background: demo.bg, color: demo.fg }}>{demo.label}</span></span>
      </div>
    </section>
  );
}

function Amounts() {
  const [sel, setSel] = useState(300);
  return (
    <section className="lp-section">
      <div className="lp-split">
        <div className="lp-split-text">
          <Eyebrow>Valores</Eyebrow>
          <h2 className="lp-h2">Tu escolhes os valores sugeridos.</h2>
          <p className="lp-lead">Define quatro valores em meticais e marca um como sugerido. Quem quiser dar outro valor escreve-o, a partir do mínimo que definires.</p>
          <div className="lp-ticks">
            <Tick>Valores em meticais, sem conversões</Tick>
            <Tick>O fã pode assinar com o nome ou com o @ das redes</Tick>
            <Tick>Mudas os valores quando quiseres no painel</Tick>
          </div>
        </div>
        <div className="lp-split-side lp-tiers">
          <div className="lp-tiers-grid">
            {[100, 300, 500, 1000].map(v => (
              <button key={v} type="button" className={'lp-tier' + (sel === v ? ' on' : '')} aria-pressed={sel === v} onClick={() => setSel(v)}>
                <span className="lp-tier-v"><b className="display">{fmt(v)}</b><span className="mono">MT</span></span>
                {v === 300 && <em>Sugerido</em>}
              </button>
            ))}
          </div>
          <div className="lp-tier-other"><span>Outro valor</span><span className="mono">mín. 5 MT</span></div>
        </div>
      </div>
    </section>
  );
}

function Groups() {
  const [i, setI] = useState(0);
  const [flash, setFlash] = useState(false);
  useEffect(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    let off: ReturnType<typeof setTimeout>;
    const t = setInterval(() => {
      setI(n => (n + 1) % SPLIT_EXAMPLES.length);
      setFlash(true);
      clearTimeout(off);
      off = setTimeout(() => setFlash(false), 700);
    }, 3400);
    return () => { clearInterval(t); clearTimeout(off); };
  }, []);
  const [fan, gross] = SPLIT_EXAMPLES[i];
  const net = gross * (1 - FEE);
  return (
    <section id="grupos" className="lp-section">
      <div className="lp-groups">
        <div className="lp-split-text">
          <Eyebrow light>Grupos</Eyebrow>
          <h2 className="lp-h2">Em grupo, cada um recebe a sua parte.</h2>
          <p className="lp-lead lp-lead-dark">Bandas, podcasts, coletivos. Definem as percentagens uma vez e cada apoio divide-se sozinho, no segundo em que é pago.</p>
          <div className="lp-ticks">
            <Tick icon={<Zap size={18} color="#FF6A1A" />}>Cada parte entra no saldo de cada membro</Tick>
            <Tick icon={<ShieldCheck size={18} color="#FF6A1A" />}>Mudar a divisão exige a aprovação de todos</Tick>
            <Tick icon={<History size={18} color="#FF6A1A" />}>O histórico de cada divisão fica guardado</Tick>
          </div>
          <Link to="/registar" className="lp-btn lp-btn-primary lp-btn-lg" style={{ marginTop: 30 }}>Criar a página do grupo<ArrowRight size={18} /></Link>
        </div>
        <div className="lp-split-side lp-groups-demo">
          <div className="lp-groups-in">
            <span className="lp-groups-cup"><CupSoda size={20} /></span>
            <span className="lp-groups-who"><span>Exemplo de apoio</span><b>{fan}</b></span>
            <span className="display lp-groups-gross">{fmt(gross)} MT</span>
          </div>
          <div className="lp-groups-line"><span /></div>
          <div className="lp-groups-bars">
            {SPLIT_MEMBERS.map(([name, p], n) => <div key={name} className={'lp-groups-bar lp-groups-bar-' + n} style={{ flex: p }}><b className="display">{p}%</b><span>{name}</span></div>)}
          </div>
          <div className="lp-groups-rows">
            {SPLIT_MEMBERS.map(([name, p]) => (
              <div key={name} className={flash ? 'flash' : ''}>
                <span>{name}</span>
                <span className="mono">{fmt(net * p / 100, 2)} MT</span>
                <span className="lp-groups-when">No saldo</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

function Dashboard({ panelTo, panelLabel }: { panelTo: string; panelLabel: string }) {
  const rows = [
    ['HT', '#D14400', '#FFFFFF', 'Hélio Tembe', 'há 4 min', 'Episódio brutal. Continuem!', '300 MT', true],
    ['IC', '#D9D1C4', '#1A1410', 'Ivânia Cuamba', 'há 22 min', 'Para o microfone novo do estúdio.', '500 MT', true],
    ['TB', '#FBD5BF', '#7A2E0A', 'Tânia Bila', 'há 2 h', 'Um pequeno patrocínio da nossa loja.', '1.000 MT', false],
  ] as const;
  const bars = [40, 55, 100, 50, 45, 62, 95, 48];
  return (
    <section className="lp-section">
      <div className="lp-split lp-split-reverse">
        <div className="lp-split-side lp-panel" aria-hidden="true">
          <div className="lp-panel-list">
            <div className="lp-panel-head"><b>Apoios</b><span className="lp-panel-filters"><span className="on">Todos</span><span>M-Pesa</span></span></div>
            {rows.map(([ini, bg, fg, name, when, msg, amt, unread]) => (
              <div key={name} className="lp-panel-row">
                <span className="lp-panel-dot" style={{ background: unread ? '#D14400' : 'transparent' }} />
                <span className="lp-panel-ini" style={{ background: bg, color: fg }}>{ini}</span>
                <span className="lp-panel-msg"><span><b style={{ fontWeight: unread ? 700 : 600 }}>{name}</b> <small>{when}</small></span><span className="lp-panel-text">{msg}</span></span>
                <span className="display lp-panel-amt">{amt}</span>
              </div>
            ))}
          </div>
          <div className="lp-panel-total">
            <span>Exemplo · 30 dias</span>
            <b className="display">96.500 MT</b>
            <span className="lp-panel-bars">{bars.map((h, n) => <span key={n} style={{ height: h + '%', background: h > 90 ? '#D14400' : '#2E2A26' }} />)}</span>
          </div>
        </div>
        <div className="lp-split-text">
          <Eyebrow>Painel</Eyebrow>
          <h2 className="lp-h2">Sabes quem te apoiou, com quanto e porquê.</h2>
          <div className="lp-features">
            <div><Inbox size={20} /><span><b>Mensagens com valor, canal e contacto</b><span>Agradece por WhatsApp com um toque.</span></span></div>
            <div><PanelsTopLeft size={20} /><span><b>A tua página, editada ao vivo</b><span>Perfil, valores e redes com pré-visualização.</span></span></div>
            <div><ChartColumn size={20} /><span><b>Números por canal</b><span>Exporta tudo para CSV quando precisares.</span></span></div>
          </div>
          <Link to={panelTo} className="lp-btn lp-btn-line lp-btn-lg" style={{ marginTop: 22 }}>{panelLabel}<ArrowRight size={18} /></Link>
        </div>
      </div>
    </section>
  );
}

function Pricing() {
  const [amount, setAmount] = useState(10000);
  return (
    <section id="precos" className="lp-section">
      <div className="lp-center">
        <Eyebrow>Preços</Eyebrow>
        <h2 className="lp-h2">Grátis para começar.</h2>
        <p className="lp-lead" style={{ marginTop: 16, maxWidth: 460 }}>Sem mensalidade nem fidelização. Só pagas quando recebes.</p>
      </div>
      <div className="lp-price">
        <div className="lp-price-fee">
          <span className="lp-price-note">Por cada apoio recebido</span>
          <span className="display lp-price-pct">15%</span>
          <span className="lp-price-note">A taxa do M-Pesa está incluída. Sem taxa de levantamento.</span>
          <div className="lp-calc">
            <label htmlFor="lp-calc"><span>Se receberes</span><span className="lp-calc-input"><input id="lp-calc" className="mono" inputMode="numeric" value={fmt(amount)} onChange={e => setAmount(Math.min(99999999, +e.target.value.replace(/\D/g, '') || 0))} /><span className="mono">MT</span></span></label>
            <div><span>ficas com</span><b className="display">{fmt(amount * (1 - FEE), 2)} MT</b></div>
          </div>
        </div>
        <div className="lp-price-list">
          <b className="lp-price-head">Tudo incluído</b>
          <Tick>Página com o teu link {DOMAIN}</Tick>
          <Tick>{nb('Pagamentos por M-Pesa e e-Mola, com cartão em breve')}</Tick>
          <Tick>Divisão automática para grupos</Tick>
          <Tick>Painel com mensagens, contactos e relatórios</Tick>
          <Tick>Ajuda em português: <a href={SUPPORT_MAILTO} className="lp-tel">{SUPPORT_EMAIL}</a></Tick>
          <Link to="/registar" className="lp-btn lp-btn-primary lp-price-go">Criar a minha página<ArrowRight size={18} /></Link>
        </div>
      </div>
    </section>
  );
}

function Questions() {
  const [open, setOpen] = useState(0);
  return (
    <section className="lp-section lp-faq">
      <h2 className="display lp-faq-title">Perguntas frequentes</h2>
      {FAQ.map(([q, a], i) => (
        <div key={q} className="lp-faq-item">
          <button type="button" aria-expanded={open === i} onClick={() => setOpen(open === i ? -1 : i)}>
            <span>{nb(q)}</span><span className={'lp-faq-icon' + (open === i ? ' on' : '')}>{open === i ? <Minus size={16} /> : <Plus size={16} />}</span>
          </button>
          {open === i && <p>{nb(a)}</p>}
        </div>
      ))}
    </section>
  );
}

export default function Landing() {
  const { me } = useSession();
  const h = useHandle();
  useEffect(() => { document.title = 'Recebe apoios dos teus fãs por M-Pesa | Refresco'; }, []);
  const home = me ? homeFor(me) : '';
  return (
    <div className="lp">
      <header className="lp-header">
        <div className="lp-header-in">
          <Link to="/" aria-label="Refresco"><Logo h={36} /></Link>
          <nav className="lp-nav" aria-label="Secções">
            <a href="#como-funciona">Como funciona</a>
            <a href="#pagamentos">Pagamentos</a>
            <a href="#grupos">Grupos</a>
            <a href="#precos">Preços</a>
          </nav>
          <span style={{ flex: 1 }} />
          {me ? <Link className="lp-btn lp-btn-primary" to={home}>Abrir painel</Link> : <>
            <Link className="lp-signin" to="/entrar">Entrar</Link>
            <Link className="lp-btn lp-btn-primary" to="/registar">Criar página</Link>
          </>}
        </div>
      </header>

      <main>
        <section className="lp-hero">
          <div className="lp-hero-text">
            <div className="lp-pill"><i />Para criadores em Moçambique</div>
            <h1 className="display lp-h1">
              Os teus fãs<br />pagam-te um<br />
              <span className="lp-h1-mark"><span className="lp-h1-dot" /><span className="lp-h1-bar" /><span className="lp-h1-word"><span /><span /><span>refresco</span></span><span>.</span></span>
            </h1>
            <p className="lp-hero-lead">{nb('Cria a tua página de apoio em dois minutos. Os fãs pagam com M-Pesa e o dinheiro acumula no teu saldo até pedires o levantamento. O e-Mola e o cartão chegam em breve.')}</p>
            {me
              ? <Link className="lp-btn lp-btn-primary lp-btn-lg lp-self-start" to={home}>Abrir painel<ArrowRight size={18} /></Link>
              : <div className="lp-hero-claim"><ClaimForm h={h} id="lp-handle" /><HandleHint state={h.state} id="lp-handle-hint" /></div>}
            <div className="lp-hero-ticks">
              <span><Check size={16} color="#D14400" />Sem mensalidade</span>
              <span><Check size={16} color="#D14400" />15% por apoio</span>
              <span><Check size={16} color="#D14400" />Divisão automática para grupos</span>
            </div>
          </div>
          <HeroScene />
        </section>

        <FactsBand />
        <Steps />
        <Payments />
        <Amounts />
        <Groups />
        <Dashboard panelTo={me ? home : '/entrar'} panelLabel={me ? 'Abrir painel' : 'Entrar no painel'} />
        <Pricing />
        <Questions />

        <section className="lp-section">
          <div className="lp-final">
            <div className="lp-final-mark" aria-hidden="true"><span /><span /><span><i /><i /></span></div>
            <h2 className="display lp-final-title">{nb('O teu público já quer ajudar. Dá-lhe um link.')}</h2>
            {me
              ? <Link className="lp-btn lp-btn-dark lp-btn-lg lp-self-start" to={home}>Abrir painel<ArrowRight size={18} /></Link>
              : <div className="lp-final-claim"><ClaimForm h={h} dark id="lp-handle-2" /><HandleHint state={h.state} id="lp-handle-2-hint" /></div>}
          </div>
        </section>
      </main>

      <footer className="lp-footer">
        <div className="lp-footer-brand"><Logo h={40} /><span>Feito em Maputo para quem cria em Moçambique.</span></div>
        <div className="lp-footer-cols">
          <div><span className="mono">Produto</span><a href="#como-funciona">Como funciona</a><a href="#grupos">Grupos</a><a href="#precos">Preços</a></div>
          <div><span className="mono">Conta</span><Link to="/entrar">Entrar</Link><Link to="/registar">Criar página</Link><a href={SUPPORT_MAILTO}>Ajuda: {SUPPORT_EMAIL}</a></div>
        </div>
        <div className="lp-footer-end"><span>© 2026 Refresco, um produto RIHA · Maputo, Moçambique</span><PoweredByRiha /><span className="mono">{DOMAIN}</span></div>
      </footer>
    </div>
  );
}
