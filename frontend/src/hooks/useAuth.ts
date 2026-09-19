
import { useCallback, useEffect, useState } from 'react';
import { clearSession, getToken, saveSession } from '../api/client';
import { login } from '../api/auth';
import type { UserSession } from '../types/domain';

export function useAuth() {
  const [session, setSession] = useState<UserSession | null>(null);
  const [loading, setLoading] = useState(true);
  const authenticate = useCallback(async () => {
    setLoading(true);
    try { const next = await login(); saveSession(next); setSession(next); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { void authenticate(); }, [authenticate]);
  const logout = () => { clearSession(); setSession(null); void authenticate(); };
  return { session, loading, authenticated: Boolean(getToken()), logout };
}
