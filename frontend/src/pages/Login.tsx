import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '../components/common/Button';
import { Card } from '../components/common/Card';
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
    <div className="mx-auto flex min-h-screen w-full max-w-6xl items-center justify-center p-4">
      <div className="grid w-full gap-4 lg:grid-cols-[1.1fr_0.9fr]">
        <Card className="p-6 md:p-8" title="SpendSense" subtitle="A quiet dashboard for personal finance">
          <div className="space-y-4">
            <p className="text-sm text-text-secondary">
              Sign in or create an account to connect to the backend API and view the dashboard.
            </p>
            <ul className="space-y-2 text-sm text-text-secondary">
              <li>• Real API-backed auth</li>
              <li>• Protected dashboard route</li>
              <li>• Token refresh support</li>
            </ul>
          </div>
        </Card>

        <Card className="p-6 md:p-8">
          <div className="mb-5 flex gap-2">
            <button className={`btn ${mode === 'sign-in' ? 'btn-primary' : 'btn-secondary'}`} onClick={() => setMode('sign-in')} type="button">
              Sign in
            </button>
            <button className={`btn ${mode === 'sign-up' ? 'btn-primary' : 'btn-secondary'}`} onClick={() => setMode('sign-up')} type="button">
              Sign up
            </button>
          </div>

          <form
            className="space-y-4"
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
                } else {
                  // sign-in flow: attempt login; if server requests TOTP, prompt and retry
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
                }
              } finally {
                setIsSubmitting(false);
              }
            }}
          >
            <Input label="Email" type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="you@example.com" required />
            <Input label="Password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="••••••••" required />
            {totpRequired && (
              <Input label="Authenticator code" type="text" value={totpCode} onChange={(event) => setTotpCode(event.target.value)} placeholder="123456" required />
            )}
            {error && <p className="text-sm text-accent-red">{error}</p>}
            <Button type="submit" className="w-full">
              {isSubmitting ? 'Working...' : mode === 'sign-in' ? 'Sign in' : 'Create account'}
            </Button>
          </form>
        </Card>
      </div>
    </div>
  );
}