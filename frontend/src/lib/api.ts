export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) { super(message); }
}

async function request<T>(method: string, path: string, body?: unknown, headers?: Record<string, string>): Promise<T> {
  const res = await fetch('/api/v1' + path, {
    method,
    credentials: 'same-origin',
    headers: { ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}), ...headers },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (res.status === 204) return undefined as T;
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    const e = data?.error;
    if (res.status === 401 && !path.startsWith('/auth') && !location.pathname.startsWith('/entrar')) {
      window.dispatchEvent(new Event('refres:unauthorized'));
    }
    throw new ApiError(res.status, e?.code || 'error', e?.message || 'Algo correu mal. Tenta de novo.');
  }
  return data as T;
}

export const api = {
  get: <T>(p: string) => request<T>('GET', p),
  post: <T>(p: string, b?: unknown, h?: Record<string, string>) => request<T>('POST', p, b ?? {}, h),
  put: <T>(p: string, b: unknown) => request<T>('PUT', p, b),
  del: <T>(p: string) => request<T>('DELETE', p),
};

export type Channel = 'mpesa' | 'emola' | 'card';
export type Money = number;

export interface Page {
  id: string; name: string; handle: string; category: string; bio: string; thanks: string;
  amounts: Money[]; default_idx: number; allow_custom: boolean; min_amount: Money;
  socials: Record<string, string>; channels: { mpesa: boolean; emola: boolean; card: boolean }; sandbox?: boolean; coming_soon?: Channel[];
}

export interface Me {
  id: string; name: string; email: string; role: 'creator' | 'superadmin';
  account?: Page; member?: { id: string; name: string; role: string; is_admin: boolean; pct: number };
  account_status?: 'active' | 'pending' | 'suspended'; sandbox?: boolean; coming_soon?: Channel[];
}

export interface SupportStatus {
  id: string; ref: string; status: 'pending' | 'confirmed' | 'failed' | 'expired'; next_action: 'wallet_pin' | 'otp';
  expires_at: string; amount: Money; channel: Channel; phone?: string; card?: string; fail_reason?: string;
  name: string; confirmed_at?: string; created_at: string; page: string; thanks: string; simulate: boolean;
}

export interface SupportView {
  id: string; ref: string; name: string; net?: string; message: string; amount: Money; preset: boolean;
  fee: Money; fee_bps: number; net_amount: Money; channel: Channel; contact: string; card?: string;
  status: 'pending' | 'confirmed' | 'failed' | 'expired'; fail_reason?: string; read: boolean; created_at: string;
  splits?: { member_id: string; name: string; pct: number; amount: Money }[]; hidden?: boolean;
}

export interface SeriesPoint { start: string; total: Money; fee: Money; by_channel: { mpesa: Money; emola: Money; card: Money }; count: number }

export interface MemberView { id: string; name: string; role: string; pct: number; wallet: 'mpesa' | 'emola'; number: string; is_admin: boolean; invited: boolean; balance: Money }

export interface WithdrawalView {
  id: string; code: string; member_id: string; member_name: string; amount: Money; dest_kind: 'mpesa' | 'emola' | 'nib';
  dest_number: string; dest_bank?: string; dest_holder?: string; status: 'requested' | 'approved' | 'paid' | 'rejected'; reason?: string; note?: string;
  transfer_ref?: string; due_at: string; decided_at?: string; paid_at?: string; created_at: string;
}
