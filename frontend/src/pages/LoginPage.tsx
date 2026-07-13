import { zodResolver } from '@hookform/resolvers/zod';
import { LockKeyhole, LogIn, ShieldCheck } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useLocation, useNavigate } from 'react-router-dom';
import { z } from 'zod';
import { useAuth } from '../auth/AuthProvider';
import { ErrorState } from '../components/Loading';

const loginSchema = z.object({
  username: z.string().trim().min(1, 'Informe o usuario.'),
  password: z.string().min(1, 'Informe a senha.')
});

type LoginValues = z.infer<typeof loginSchema>;
const totpSchema = z.object({ code: z.string().trim().regex(/^\d{6}$/, 'Informe o codigo TOTP de seis digitos.') });
type TotpValues = z.infer<typeof totpSchema>;

export function LoginPage() {
  const { login, verifyMfa } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [error, setError] = useState<unknown>();
  const [submitting, setSubmitting] = useState(false);
  const [mfaChallengeId, setMfaChallengeId] = useState<string>();
  const { register, handleSubmit, formState: { errors } } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { username: '', password: '' }
  });
  const { register: registerTotp, handleSubmit: handleTotpSubmit, formState: { errors: totpErrors } } = useForm<TotpValues>({
    resolver: zodResolver(totpSchema),
    defaultValues: { code: '' }
  });
  const from = (location.state as { from?: string } | null)?.from || '/';

  const submit = async (values: LoginValues) => {
    setSubmitting(true);
    setError(undefined);
    try {
      const result = await login(values);
      if (result.status === 'mfa_required') {
        setMfaChallengeId(result.challengeId);
        return;
      }
      navigate(from, { replace: true });
    } catch (nextError) {
      setError(nextError);
    } finally {
      setSubmitting(false);
    }
  };

  const submitTotp = async (values: TotpValues) => {
    if (!mfaChallengeId) return;
    setSubmitting(true);
    setError(undefined);
    try {
      const result = await verifyMfa(mfaChallengeId, values.code);
      if (result.status !== 'authenticated') throw new Error('A verificacao MFA nao concluiu a autenticacao.');
      navigate(from, { replace: true });
    } catch (nextError) {
      setError(nextError);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className="login-shell">
      <section className="login-card" aria-labelledby="login-title">
        <div className="login-mark"><ShieldCheck size={28} aria-hidden="true" /></div>
        <p className="eyebrow">HU-UFCAT / HUBRASIL</p>
        <h1 id="login-title">Console Samba</h1>
        <p className="muted">Acesso administrativo auditado para servidores FreeBSD.</p>
        {!mfaChallengeId ? <form onSubmit={handleSubmit(submit)} noValidate>
          <label>
            Usuario
            <input autoComplete="username" {...register('username')} />
            {errors.username && <span className="field-error">{errors.username.message}</span>}
          </label>
          <label>
            Senha
            <input type="password" autoComplete="current-password" {...register('password')} />
            {errors.password && <span className="field-error">{errors.password.message}</span>}
          </label>
          <button className="button login-submit" type="submit" disabled={submitting}>
            <LogIn size={17} /> {submitting ? 'Autenticando...' : 'Entrar'}
          </button>
        </form> : <form onSubmit={handleTotpSubmit(submitTotp)} noValidate>
          <div className="alert alert-info">A politica da sua conta exige confirmacao TOTP.</div>
          <label>
            Codigo de autenticacao
            <input autoComplete="one-time-code" inputMode="numeric" maxLength={6} {...registerTotp('code')} />
            {totpErrors.code && <span className="field-error">{totpErrors.code.message}</span>}
          </label>
          <button className="button login-submit" type="submit" disabled={submitting}>
            <ShieldCheck size={17} /> {submitting ? 'Verificando...' : 'Confirmar MFA'}
          </button>
          <button className="button button-secondary" type="button" onClick={() => { setMfaChallengeId(undefined); setError(undefined); }} disabled={submitting}>Voltar</button>
        </form>}
        {error ? <ErrorState error={error} /> : null}
        <div className="login-security-note"><LockKeyhole size={16} /> Sessao protegida por cookie HttpOnly e token CSRF.</div>
      </section>
    </main>
  );
}
