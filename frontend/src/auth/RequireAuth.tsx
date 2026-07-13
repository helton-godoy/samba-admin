import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuth } from './AuthProvider';
import { ErrorState, Loading } from '../components/Loading';

export function RequireAuth() {
  const { status, error, refresh } = useAuth();
  const location = useLocation();

  if (status === 'loading') return <Loading label="Verificando sessao administrativa..." />;
  if (status === 'unavailable') {
    return <ErrorState
      message="Nao foi possivel validar a sessao com a API."
      error={error}
      action={<button className="button" onClick={() => void refresh()}>Tentar novamente</button>}
    />;
  }
  if (status !== 'authenticated') {
    return <Navigate to="/login" replace state={{ from: `${location.pathname}${location.search}` }} />;
  }
  return <Outlet />;
}
