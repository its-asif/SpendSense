import type { AuthSession, AuthUser } from '../types';

const AUTH_KEY = 'spendsense-auth-session';

type StoredSession = AuthSession & {
  rawUser?: unknown;
};

function toDisplayName(email: string, displayName?: string | null) {
  if (displayName && displayName.trim()) {
    return displayName.trim();
  }

  const localPart = email.split('@')[0]?.trim();
  return localPart ? localPart.replace(/[._-]+/g, ' ') : email;
}

export function toAuthUser(email: string, displayName?: string | null, baseCurrency = 'USD'): AuthUser {
  return {
    name: toDisplayName(email, displayName),
    email,
    baseCurrency: baseCurrency.trim().toUpperCase() || 'USD',
  };
}

export function readSession(): AuthSession | null {
  if (typeof window === 'undefined') {
    return null;
  }

  const raw = window.localStorage.getItem(AUTH_KEY);
  if (!raw) {
    return null;
  }

  try {
    const session = JSON.parse(raw) as Partial<StoredSession>;
    if (
      typeof session.accessToken === 'string' &&
      typeof session.refreshToken === 'string' &&
      session.user &&
      typeof session.user.name === 'string' &&
      typeof session.user.email === 'string'
    ) {
      return {
        accessToken: session.accessToken,
        refreshToken: session.refreshToken,
        user: {
          name: session.user.name,
          email: session.user.email,
          baseCurrency: typeof session.user.baseCurrency === 'string' && session.user.baseCurrency.trim() ? session.user.baseCurrency.trim().toUpperCase() : 'USD',
        },
      };
    }
  } catch {
    return null;
  }

  return null;
}

export function saveSession(session: AuthSession) {
  window.localStorage.setItem(AUTH_KEY, JSON.stringify(session));
}

export function clearSession() {
  window.localStorage.removeItem(AUTH_KEY);
}

export function readUser(): AuthUser | null {
  return readSession()?.user ?? null;
}

export function saveUser(user: AuthUser) {
  saveSession({ accessToken: '', refreshToken: '', user });
}

export function clearUser() {
  clearSession();
}