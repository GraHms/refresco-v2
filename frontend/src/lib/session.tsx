import { createContext, ReactNode, useCallback, useContext, useEffect, useState } from 'react';
import { api, Me } from './api';

interface SessionCtx { me: Me | null | undefined; refresh: () => Promise<Me | null>; logout: () => Promise<void>; set: (m: Me | null) => void }

const Ctx = createContext<SessionCtx>({ me: undefined, refresh: async () => null, logout: async () => {}, set: () => {} });
export const useSession = () => useContext(Ctx);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [me, setMe] = useState<Me | null | undefined>(undefined);
  const refresh = useCallback(async () => {
    try { const m = await api.get<Me>('/me'); setMe(m); return m; } catch { setMe(null); return null; }
  }, []);
  const logout = useCallback(async () => { try { await api.post('/auth/logout'); } finally { setMe(null); } }, []);
  useEffect(() => {
    const h = () => setMe(null);
    window.addEventListener('refres:unauthorized', h);
    return () => window.removeEventListener('refres:unauthorized', h);
  }, []);
  return <Ctx.Provider value={{ me, refresh, logout, set: setMe }}>{children}</Ctx.Provider>;
}
