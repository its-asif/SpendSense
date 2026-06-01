import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '../components/common/Button';
import { Input } from '../components/common/Input';
import { clearSession, saveSession } from '../lib/storage';
import { login, register } from '../api/auth';
import type { AuthSession } from '../types';

type LoginProps = {
  onAuthenticated: (session: AuthSession) => void;
};

export function Login({ onAuthenticated }: LoginProps) {
  const navigate = useNavigate();
  const [mode, setMode] = useState<'sign-in' | 'sign-up'>('sign-in');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [totpRequired, setTotpRequired] = useState(false);
  const [totpCode, setTotpCode] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  return (
    <div className="auth-shell">
      <div className="auth-orb auth-orb-left" aria-hidden="true" />
      <div className="auth-orb auth-orb-right" aria-hidden="true" />

      <main className="auth-stage">
        <section className="auth-card">
          <div className="auth-card__header">
            <div className="auth-brand-row">
              <div>
                <p className="auth-kicker">SpendSense</p>
                <h1 className="auth-title">Your money, all in one place.</h1>
              </div>
            </div>
            <p className="auth-copy">
              Sign in or create an account to unlock your personalized dashboard, sync your wallets, and get insights into your spending habits.
            </p>
            
          </div>

          <div className="auth-toggle" role="tablist" aria-label="Authentication mode">
            <button
              className={mode === 'sign-in' ? 'auth-toggle__button is-active' : 'auth-toggle__button'}
              onClick={() => setMode('sign-in')}
              type="button"
              role="tab"
              aria-selected={mode === 'sign-in'}
            >
              Sign in
            </button>
            <button
              className={mode === 'sign-up' ? 'auth-toggle__button is-active' : 'auth-toggle__button'}
              onClick={() => setMode('sign-up')}
              type="button"
              role="tab"
              aria-selected={mode === 'sign-up'}
            >
              Sign up
            </button>
          </div>

          <form
            className="auth-form"
            onSubmit={async (event) => {
              event.preventDefault();
              setError(null);
              setIsSubmitting(true);

              try {
                clearSession();

                if (mode === 'sign-up') {
                  const session = await register({ email, password });
                  saveSession(session);
                  onAuthenticated(session);
                  navigate('/');
                  return;
                }

                try {
                  const session = await login({ email, password, totp_code: totpRequired ? totpCode : undefined });
                  saveSession(session);
                  onAuthenticated(session);
                  navigate('/');
                } catch (err: any) {
                  const code = err?.response?.data?.error?.code;
                  if (code === 'TOTP_REQUIRED') {
                    setTotpRequired(true);
                    setError('Two-factor authentication code required. Enter your code below and sign in again.');
                  } else if (code === 'INVALID_CODE') {
                    setError('Invalid two-factor authentication code.');
                  } else {
                    setError('Unable to sign in. Check your credentials and backend server.');
                    throw err;
                  }
                }
              } finally {
                setIsSubmitting(false);
              }
            }}
          >
            <Input
              label="Email"
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="you@example.com"
              autoComplete="email"
              required
            />
            <Input
              label="Password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="••••••••"
              autoComplete={mode === 'sign-up' ? 'new-password' : 'current-password'}
              required
            />
            {totpRequired && (
              <Input
                label="Authenticator code"
                type="text"
                value={totpCode}
                onChange={(event) => setTotpCode(event.target.value)}
                placeholder="123456"
                inputMode="numeric"
                autoComplete="one-time-code"
                required
              />
            )}
            {error && <p className="auth-error">{error}</p>}
            <Button type="submit" className="w-full justify-center">
              {isSubmitting ? 'Working...' : mode === 'sign-in' ? 'Sign in' : 'Create account'}
            </Button>
          </form>

          <p className="auth-footer">
            {mode === 'sign-in' ? 'New here?' : 'Already have an account?'}{' '}
            <button
              className="auth-inline-link"
              type="button"
              onClick={() => {
                setMode(mode === 'sign-in' ? 'sign-up' : 'sign-in');
                setTotpRequired(false);
                setTotpCode('');
                setError(null);
              }}
            >
              {mode === 'sign-in' ? 'Create your account' : 'Go back to sign in'}
            </button>
          </p>
        </section>
      </main>
    </div>
  );
}