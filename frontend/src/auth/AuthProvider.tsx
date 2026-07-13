import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { api, isApiError, SESSION_EXPIRED_EVENT, type AuthenticationResult } from '../api/client';
import type { LoginRequest, User } from '../api/generated';

export type AuthenticationStatus = 'loading' | 'authenticated' | 'anonymous' | 'unavailable';

interface AuthContextValue {
  status: AuthenticationStatus;
  user?: User;
  error?: Error;
  login(credentials: LoginRequest): Promise<AuthenticationResult>;
  verifyMfa(challengeId: string, code: string): Promise<AuthenticationResult>;
  logout(): Promise<void>;
  refresh(): Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthenticationStatus>('loading');
  const [user, setUser] = useState<User>();
  const [error, setError] = useState<Error>();

  const setAnonymous = () => {
    setUser(undefined);
    setError(undefined);
    setStatus('anonymous');
  };

  const refresh = async () => {
    try {
      const nextUser = await api.auth.refresh();
      setUser(nextUser);
      setError(undefined);
      setStatus('authenticated');
    } catch (nextError) {
      if (isApiError(nextError) && nextError.status === 401) {
        setAnonymous();
        return;
      }
      setUser(undefined);
      setError(nextError instanceof Error ? nextError : new Error('Falha ao verificar a sessao.'));
      setStatus('unavailable');
    }
  };

  const acceptAuthentication = (result: AuthenticationResult): AuthenticationResult => {
    if (result.status === 'authenticated') {
      setUser(result.user);
      setError(undefined);
      setStatus('authenticated');
    } else {
      setUser(undefined);
      setError(undefined);
      setStatus('anonymous');
    }
    return result;
  };

  const login = async (credentials: LoginRequest) => {
    return acceptAuthentication(await api.auth.login(credentials));
  };

  const verifyMfa = async (challengeId: string, code: string) => {
    return acceptAuthentication(await api.auth.verifyMfa({ challengeId, code }));
  };

  const logout = async () => {
    try {
      await api.auth.logout();
    } finally {
      setAnonymous();
    }
  };

  useEffect(() => {
    const handleSessionExpired = () => setAnonymous();
    window.addEventListener(SESSION_EXPIRED_EVENT, handleSessionExpired);
    void refresh();
    return () => window.removeEventListener(SESSION_EXPIRED_EVENT, handleSessionExpired);
  }, []);

  return (
    <AuthContext.Provider value={{ status, user, error, login, verifyMfa, logout, refresh }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth deve ser usado dentro de AuthProvider.');
  return context;
}
