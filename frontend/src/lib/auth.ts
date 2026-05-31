export const AUTH_STORAGE_KEY = 'spendsense-auth-user';

export type DemoUser = {
  name: string;
  email: string;
};

export function getStoredUser(): DemoUser | null {
  if (typeof window === 'undefined') {
    return null;
  }

  const raw = window.localStorage.getItem(AUTH_STORAGE_KEY);
  if (!raw) {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as Partial<DemoUser>;
    if (typeof parsed.name === 'string' && typeof parsed.email === 'string') {
      return { name: parsed.name, email: parsed.email };
    }
  } catch {
    return null;
  }

  return null;
}

export function saveStoredUser(user: DemoUser) {
  window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(user));
}

export function clearStoredUser() {
  window.localStorage.removeItem(AUTH_STORAGE_KEY);
}