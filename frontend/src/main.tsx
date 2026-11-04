import { lazy, StrictMode, Suspense, useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter, Navigate, Route, Routes, useLocation } from 'react-router-dom';
import '@fontsource-variable/archivo/wdth.css';
import '@fontsource-variable/instrument-sans/wdth.css';
import '@fontsource-variable/geist-mono';
import './styles.css';
import { SessionProvider, useSession } from './lib/session';
import { ToastProvider } from './components/ui';
import ErrorBoundary from './components/ErrorBoundary';
import SupportPage from './pages/SupportPage';
import Auth, { homeFor } from './pages/Auth';
import { Landing, SplitApproval } from './pages/Misc';

const BackofficeLayout = lazy(() => import('./pages/backoffice/Layout'));
const Overview = lazy(() => import('./pages/backoffice/Overview'));
const Inbox = lazy(() => import('./pages/backoffice/Inbox'));
const PageSettings = lazy(() => import('./pages/backoffice/PageSettings'));
const Group = lazy(() => import('./pages/backoffice/Group'));
const Payouts = lazy(() => import('./pages/backoffice/Payouts'));
const AdminAccounts = lazy(() => import('./pages/admin/Admin').then(m => ({ default: m.AdminAccounts })));
const AdminHome = lazy(() => import('./pages/admin/Admin').then(m => ({ default: m.AdminHome })));
const AdminInvites = lazy(() => import('./pages/admin/Admin').then(m => ({ default: m.AdminInvites })));
const AdminLayout = lazy(() => import('./pages/admin/Admin').then(m => ({ default: m.AdminLayout })));
const AdminWithdrawals = lazy(() => import('./pages/admin/Admin').then(m => ({ default: m.AdminWithdrawals })));

function Guard({ role, children }: { role: 'creator' | 'superadmin'; children: JSX.Element }) {
  const { me, refresh } = useSession();
  const loc = useLocation();
  useEffect(() => { if (me === undefined) refresh(); }, [me]);
  if (me === undefined) return <div style={{ padding: 40 }}><div className="skeleton" style={{ height: 300, maxWidth: 1180, margin: '0 auto' }} /></div>;
  if (!me) return <Navigate to={'/entrar?next=' + encodeURIComponent(loc.pathname)} replace />;
  if (me.role !== role) return <Navigate to={homeFor(me)} replace />;
  return children;
}

function Boot({ children }: { children: JSX.Element }) {
  const { me, refresh } = useSession();
  const loc = useLocation();
  useEffect(() => { if (me === undefined && loc.pathname === '/') refresh(); }, [loc.pathname]);
  return children;
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
    <BrowserRouter>
      <SessionProvider>
        <ToastProvider>
          <Boot>
            <Suspense fallback={<div style={{ padding: 40 }}><div className="skeleton" style={{ height: 300, maxWidth: 1100, margin: '0 auto' }} /></div>}>
            <Routes>
              <Route path="/" element={<Landing />} />
              <Route path="/entrar" element={<Auth />} />
              <Route path="/registar" element={<Auth />} />
              <Route path="/recuperar" element={<Auth />} />
              <Route path="/aprovar" element={<SplitApproval />} />
              <Route path="/painel" element={<Guard role="creator"><BackofficeLayout /></Guard>}>
                <Route index element={<Navigate to="visao-geral" replace />} />
                <Route path="visao-geral" element={<Overview />} />
                <Route path="apoios" element={<Inbox />} />
                <Route path="apoios/:id" element={<Inbox />} />
                <Route path="pagina" element={<PageSettings />} />
                <Route path="grupo" element={<Group />} />
                <Route path="levantamentos" element={<Payouts />} />
                <Route path="*" element={<Navigate to="visao-geral" replace />} />
              </Route>
              <Route path="/admin" element={<Guard role="superadmin"><AdminLayout /></Guard>}>
                <Route index element={<Navigate to="painel" replace />} />
                <Route path="painel" element={<AdminHome />} />
                <Route path="contas" element={<AdminAccounts />} />
                <Route path="contas/:id" element={<AdminAccounts />} />
                <Route path="levantamentos" element={<AdminWithdrawals />} />
                <Route path="convites" element={<AdminInvites />} />
                <Route path="*" element={<Navigate to="painel" replace />} />
              </Route>
              <Route path="/:handle" element={<SupportPage />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
            </Suspense>
          </Boot>
        </ToastProvider>
      </SessionProvider>
    </BrowserRouter>
    </ErrorBoundary>
  </StrictMode>,
);
