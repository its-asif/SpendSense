import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import toast from 'react-hot-toast';
import { Header } from '../components/layout/Header';
import { Layout } from '../components/layout/Layout';
import { Card } from '../components/common/Card';
import { Button } from '../components/common/Button';
import { Input } from '../components/common/Input';
import Modal from '../components/common/Modal';
import { SettingsTabNav } from '../components/settings/SettingsTabNav';
import { getDashboardSummary, getDashboardWidgets } from '../api/dashboard';
import { listWallets } from '../api/expenses';
import { listCurrencies } from '../api/currencies';
import { logoutAll, logoutOtherSessions, updateUserPreferences, updateUserProfile, changePassword, listSessions, revokeSession, me, twoFASetup, twoFAConfirm, twoFADisable } from '../api/auth';
import { readSession, saveSession } from '../lib/storage';
import { readUserSettings, saveUserSettings, type UserSettings } from '../lib/userSettings';
import { formatCurrency } from '../lib/userSettings';
import type { ApiSessionRow, AuthUser, CurrencyOption, DashboardSummary, DashboardWidgetsResponse, Wallet } from '../types';

type SettingsTabId = 'account' | 'general' | 'profile' | 'security' | 'session';

const tabs: Array<{ id: SettingsTabId; label: string; path: string }> = [
  { id: 'account', label: 'Account', path: 'account' },
  { id: 'general', label: 'General', path: 'general' },
  { id: 'profile', label: 'Profile', path: 'profile' },
  { id: 'security', label: 'Security', path: 'security' },
  { id: 'session', label: 'Session', path: 'sessions' },
];

type SettingsPageProps = {
  user: AuthUser;
  onLogout: () => void;
};

function getInitials(name: string) {
  return name
    .split(' ')
    .map((part) => part[0])
    .join('')
    .slice(0, 2)
    .toUpperCase();
}

function tabIdFromSlug(slug?: string): SettingsTabId {
  switch (slug) {
    case 'general':
    case 'profile':
    case 'security':
    case 'session':
      return slug;
    case 'sessions':
      return 'session';
    default:
      return 'account';
  }
}

function slugFromTabId(tabId: SettingsTabId): string {
  return tabId === 'session' ? 'sessions' : tabId;
}

export function SettingsPage({ user, onLogout }: SettingsPageProps) {
  const navigate = useNavigate();
  const { tab: tabSlug } = useParams<{ tab?: string }>();
  const [activeTab, setActiveTab] = useState<SettingsTabId>(() => tabIdFromSlug(tabSlug));
  const [currencies, setCurrencies] = useState<CurrencyOption[]>([]);
  const [isLoadingCurrencies, setIsLoadingCurrencies] = useState(false);
  const [settings, setSettings] = useState<UserSettings>(() => readUserSettings());
  const [profile, setProfile] = useState(() => ({ displayName: user.name, avatarUrl: '' }));
  const [dashboardSummary, setDashboardSummary] = useState<DashboardSummary | null>(null);
  const [dashboardWidgets, setDashboardWidgets] = useState<DashboardWidgetsResponse | null>(null);
  const [wallets, setWallets] = useState<Wallet[]>([]);
  const [isLoadingData, setIsLoadingData] = useState(true);
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isChangingPassword, setIsChangingPassword] = useState(false);
  const [isPasswordModalOpen, setIsPasswordModalOpen] = useState(false);
  const [sessions, setSessions] = useState<ApiSessionRow[]>([]);
  const [isLoadingSessions, setIsLoadingSessions] = useState(false);
  const [is2FAModalOpen, setIs2FAModalOpen] = useState(false);
  const [is2FAEnabled, setIs2FAEnabled] = useState(false);
  const [twoFASetupData, setTwoFASetup] = useState<{ secret: string; otp_auth_url: string; qr_data_url: string } | null>(null);
  const [twoFACode, setTwoFACode] = useState('');

  useEffect(() => {
    const nextTab = tabIdFromSlug(tabSlug);
    setActiveTab(nextTab);

    const expectedSlug = slugFromTabId(nextTab);
    if ((tabSlug ?? 'account') !== expectedSlug) {
      navigate(`/settings/${expectedSlug}`, { replace: true });
    }
  }, [navigate, tabSlug]);

  const handleChangeTab = (tabId: string) => {
    const nextTab = tabId as SettingsTabId;
    setActiveTab(nextTab);
    navigate(`/settings/${slugFromTabId(nextTab)}`);
  };

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setIsLoadingData(true);
      setIsLoadingCurrencies(true);
      try {
        const [summary, widgets, walletList, currencyList] = await Promise.all([
          getDashboardSummary(settings.defaultCurrency),
          getDashboardWidgets(settings.defaultCurrency),
          listWallets(),
          listCurrencies(settings.defaultCurrency),
        ]);

        if (!cancelled) {
          setDashboardSummary(summary);
          setDashboardWidgets(widgets);
          setWallets(walletList);
          setCurrencies(currencyList);
        }
      } catch {
        if (!cancelled) {
          toast.error('Failed to load settings data');
        }
      } finally {
        if (!cancelled) {
          setIsLoadingCurrencies(false);
          setIsLoadingData(false);
          // load sessions as well
          setIsLoadingSessions(true);
          void listSessions()
            .then((res) => {
              if (!cancelled) setSessions(res.sessions ?? []);
            })
            .catch(() => {
              if (!cancelled) {
                // ignore session load errors
              }
            })
            .finally(() => {
              if (!cancelled) setIsLoadingSessions(false);
            });
        // fetch current user to get 2FA status
        void me().then((m) => {
          if (!cancelled) setIs2FAEnabled(!!m.totp_enabled);
        }).catch(() => {});
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [settings.defaultCurrency]);

  const currencyCode = dashboardSummary?.base_currency ?? settings.defaultCurrency;
  const monthlyIncome = dashboardSummary?.monthly_income ?? 0;
  const monthlyExpenses = dashboardSummary?.monthly_expenses ?? 0;
  const monthlySavings = dashboardSummary?.monthly_savings ?? 0;
  const totalBalance = dashboardSummary?.total_balance ?? 0;
  const safeToSpend = dashboardSummary?.safe_to_spend ?? 0;

  const categoryTotals = new Map(dashboardSummary?.category_breakdown.map((entry) => [entry.category_id, entry.total]) ?? []);
  const totalCategorySpend = dashboardSummary?.category_breakdown.reduce((sum, entry) => sum + entry.total, 0) ?? 0;

  const handleSaveSettings = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    try {
      const updatedUser = await updateUserPreferences({
        baseCurrency: settings.defaultCurrency,
        timezone: settings.timezone,
        locale: settings.locale,
      });

      saveUserSettings(settings);

      const session = readSession();
      if (session) {
        saveSession({
          ...session,
          user: {
            ...session.user,
            name: updatedUser.display_name?.trim() || session.user.name,
            email: updatedUser.email,
            baseCurrency: updatedUser.base_currency,
          },
        });
      }

      toast.success('Settings saved');
    } catch {
      toast.error('Failed to save settings');
    }
  };

  const handleSaveProfile = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    try {
      const updatedUser = await updateUserProfile(profile);

      const nextDisplayName = updatedUser.display_name?.trim() || updatedUser.email.split('@')[0] || updatedUser.email;
      setProfile((current) => ({
        ...current,
        displayName: nextDisplayName,
        avatarUrl: updatedUser.avatar_url ?? '',
      }));

      const session = readSession();
      if (session) {
        saveSession({
          ...session,
          user: {
            ...session.user,
            name: nextDisplayName,
            email: updatedUser.email,
            baseCurrency: updatedUser.base_currency,
          },
        });
      }

      toast.success('Profile updated');
    } catch {
      toast.error('Failed to update profile');
    }
  };

  const handleLogoutAll = async () => {
    try {
      await logoutAll();
      await onLogout();
      toast.success('Signed out from all sessions');
    } catch {
      toast.error('Failed to sign out all sessions');
    }
  };

  const handleLogoutOtherSessions = async () => {
    try {
      await logoutOtherSessions();
      toast.success('Signed out other devices');
      const res = await listSessions();
      setSessions(res.sessions ?? []);
    } catch {
      toast.error('Failed to sign out other devices');
    }
  };

  const handleChangePassword = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (newPassword !== confirmPassword) {
      toast.error('New password and confirmation do not match');
      return;
    }

    setIsChangingPassword(true);
    try {
      await changePassword({ oldPassword: currentPassword, newPassword });
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
      toast.success('Password changed');
    } catch (err) {
      toast.error('Failed to change password');
    } finally {
      setIsChangingPassword(false);
    }
  };

  // 2FA modal handlers
  async function handleOpen2FA() {
    try {
      setIs2FAModalOpen(true);
      const setup = await twoFASetup();
      setTwoFASetup(setup);
    } catch {
      toast.error('Failed to start 2FA setup');
    }
  }

  async function handleConfirm2FA() {
    if (!twoFACode) return toast.error('Enter the verification code');
    try {
      await twoFAConfirm(twoFACode, twoFASetupData?.secret);
      setIs2FAEnabled(true);
      setIs2FAModalOpen(false);
      setTwoFASetup(null);
      setTwoFACode('');
      toast.success('2FA enabled');
    } catch {
      toast.error('Invalid verification code');
    }
  }

  async function handleDisable2FA() {
    try {
      await twoFADisable();
      setIs2FAEnabled(false);
      toast.success('2FA disabled');
    } catch {
      toast.error('Failed to disable 2FA');
    }
  }

  return (
    <Layout>
      <Header user={user} onLogout={onLogout} />

      <main className="min-h-[calc(100vh-88px-64px)] container mx-auto">
        <div className="py-4">
          <div className="flex flex-col gap-1 md:flex-row md:items-center md:justify-between">
            <div className="flex flex-col gap-1">
              <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
              <p className="text-sm text-text-muted">Manage your account, preferences, and connected resources.</p>
            </div>

            <nav className="flex items-center space-x-1 text-sm text-text-muted">
              <div className="flex items-center">
                <Link className="hover:text-foreground hover:underline" to="/">
                  Home
                </Link>
              </div>
              <div className="flex items-center gap-1">
                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-4 w-4">
                  <path d="m9 18 6-6-6-6" />
                </svg>
                <span className="pointer-events-none text-foreground">Settings</span>
              </div>
            </nav>
          </div>
        </div>

        <SettingsTabNav tabs={tabs} activeTab={activeTab} onChange={handleChangeTab} />

        <div className="py-6">
          {activeTab === 'account' && (
            <div className="grid gap-4 sm:gap-6 lg:grid-cols-3">
              <div className="space-y-4 sm:space-y-6 lg:col-span-2">
                <Card>
                  <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
                    <div className="mx-auto flex h-[80px] w-[80px] shrink-0 items-center justify-center rounded-full bg-dark-elevated text-2xl font-bold text-text-primary sm:mx-0">
                      {getInitials(user.name)}
                    </div>
                    <div className="flex-1 space-y-3 text-center sm:space-y-4 sm:text-left">
                      <div>
                        <h2 className="text-lg sm:text-xl font-bold text-foreground">Welcome, {user.name}!</h2>
                        <p className="text-xs sm:text-sm text-text-muted">
                          Your account is connected to {wallets.length} wallet{wallets.length === 1 ? '' : 's'} and the live dashboard summary.
                        </p>
                      </div>
                      <div className="flex flex-col gap-2 sm:flex-row sm:gap-3">
                        <Button type="button" onClick={() => handleChangeTab('general')}>
                          Verify account
                        </Button>
                        <Button type="button" variant="secondary" onClick={() => handleChangeTab('security')}>
                          Two-factor authentication (2FA)
                        </Button>
                      </div>
                    </div>
                  </div>
                </Card>

                <Card title="Information" action={<Button type="button" variant="secondary" onClick={() => handleChangeTab('general')}>Edit</Button>}>
                  <div className="grid gap-4 sm:gap-6 grid-cols-1 sm:grid-cols-2">
                    <div className="space-y-1">
                      <p className="text-xs sm:text-sm text-text-muted">EMAIL ADDRESS</p>
                      <p className="text-sm sm:text-base font-medium">{user.email}</p>
                    </div>
                    <div className="space-y-1">
                      <p className="text-xs sm:text-sm text-text-muted">BASE CURRENCY</p>
                      <p className="text-sm sm:text-base font-medium">{currencyCode}</p>
                    </div>
                    <div className="space-y-1">
                      <p className="text-xs sm:text-sm text-text-muted">LOCALE</p>
                      <p className="text-sm sm:text-base font-medium">{settings.locale}</p>
                    </div>
                    <div className="space-y-1">
                      <p className="text-xs sm:text-sm text-text-muted">TIMEZONE</p>
                      <p className="text-sm sm:text-base font-medium">{settings.timezone}</p>
                    </div>
                    <div className="space-y-1 sm:col-span-2">
                      <p className="text-xs sm:text-sm text-text-muted">FINANCIAL SNAPSHOT</p>
                      <p className="text-sm sm:text-base font-medium">
                        {isLoadingData ? 'Loading summary...' : `${formatCurrency(totalBalance, currencyCode, settings.locale)} total balance, ${formatCurrency(monthlySavings, currencyCode, settings.locale)} saved this month`}
                      </p>
                    </div>
                  </div>
                </Card>
              </div>

              <div className="space-y-4 sm:space-y-6">
                <Card title="Verify & Upgrade">
                  <div className="space-y-3 sm:space-y-4">
                    <div className="flex items-center gap-2 text-sm">
                      <span className="font-medium">Account Status :</span>
                      <span className={wallets.length ? 'text-accent-green' : 'text-accent-amber'}>
                        {wallets.length ? 'Active' : 'Pending'}
                      </span>
                    </div>
                    <p className="text-xs sm:text-sm text-text-muted">
                      {wallets.length
                        ? 'You already have connected wallets and can review balances, transfers, and preferences.'
                        : 'Add at least one wallet to unlock the full account overview and transaction tracking.'}
                    </p>
                    <Button type="button" className="w-full" onClick={() => handleChangeTab('security')}>
                      Open security
                    </Button>
                  </div>
                </Card>

              </div>
            </div>
          )}

          {activeTab === 'general' && (
            <div className="grid gap-4 lg:grid-cols-3">
              <Card title="General preferences" subtitle="Manage default currency and localization preferences" className="lg:col-span-2">
                <form className="space-y-4" onSubmit={handleSaveSettings}>
                  <label className="block">
                    <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Default currency</span>
                    <select
                      className="input"
                      value={settings.defaultCurrency}
                      onChange={(event) => setSettings((current) => ({ ...current, defaultCurrency: event.target.value }))}
                      disabled={isLoadingCurrencies}
                    >
                      {currencies.map((currency) => (
                        <option key={currency.code} value={currency.code}>
                          {currency.code} - {currency.name}
                        </option>
                      ))}
                    </select>
                  </label>

                  <Input
                    label="Locale"
                    value={settings.locale}
                    onChange={(event) => setSettings((current) => ({ ...current, locale: event.target.value }))}
                    placeholder="en-US"
                  />

                  <Input
                    label="Timezone"
                    value={settings.timezone}
                    onChange={(event) => setSettings((current) => ({ ...current, timezone: event.target.value }))}
                    placeholder="Asia/Dhaka"
                  />

                  <div className="flex items-center gap-3">
                    <Button type="submit">Save settings</Button>
                    <Button type="button" variant="secondary" onClick={() => handleChangeTab('account')}>
                      Preview account
                    </Button>
                  </div>
                </form>
              </Card>

              <Card title="Preview">
                <div className="space-y-3 text-sm">
                  <div className="flex items-center justify-between rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                    <span className="text-text-secondary">Selected currency</span>
                    <span className="font-semibold text-text-primary">{settings.defaultCurrency}</span>
                  </div>
                  <div className="flex items-center justify-between rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                    <span className="text-text-secondary">Locale</span>
                    <span className="font-semibold text-text-primary">{settings.locale}</span>
                  </div>
                  <div className="flex items-center justify-between rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                    <span className="text-text-secondary">Timezone</span>
                    <span className="font-semibold text-text-primary">{settings.timezone}</span>
                  </div>
                  <div className="rounded-2xl border border-dark-elevated bg-dark-bg p-4 text-text-muted">
                    Your dashboard and wallet summaries will use this currency for totals and conversions.
                  </div>
                </div>
              </Card>
            </div>
          )}

          {activeTab === 'profile' && (
            <div className="grid gap-4 lg:grid-cols-3">
              <Card title="Profile" subtitle="View the current account profile" className="lg:col-span-2">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
                  <div className="mx-auto flex h-[96px] w-[96px] shrink-0 items-center justify-center rounded-full bg-dark-elevated text-3xl font-bold text-text-primary sm:mx-0">
                    {getInitials(user.name)}
                  </div>
                  <div className="space-y-3 text-center sm:text-left">
                    <div>
                      <h2 className="text-xl font-bold tracking-tight text-foreground">{user.name}</h2>
                      <p className="text-sm text-text-muted">{user.email}</p>
                    </div>
                    <div className="flex flex-wrap justify-center gap-2 sm:justify-start">
                      <span className="badge badge-info">{currencyCode}</span>
                      <span className="badge badge-income">{wallets.length} wallets</span>
                      <span className="badge badge-expense">{dashboardWidgets?.budgets.length ?? 0} budgets</span>
                    </div>
                    <p className="max-w-2xl text-sm leading-6 text-text-secondary">
                      This profile is powered by the authenticated session and your saved preferences, so the summary stays in sync with the rest of the app.
                    </p>
                  </div>
                </div>
              </Card>

              <Card title="Personal info" subtitle="Update the name and avatar used across the app" className="lg:col-span-2">
                <form className="space-y-4" onSubmit={handleSaveProfile}>
                  <Input
                    label="Display name"
                    value={profile.displayName}
                    onChange={(event) => setProfile((current) => ({ ...current, displayName: event.target.value }))}
                    placeholder="Your name"
                  />

                  <Input
                    label="Avatar URL"
                    value={profile.avatarUrl}
                    onChange={(event) => setProfile((current) => ({ ...current, avatarUrl: event.target.value }))}
                    placeholder="https://example.com/avatar.png"
                  />

                  <div className="rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3 text-sm text-text-secondary">
                    Email address is managed by your login account and cannot be changed here.
                  </div>

                  <div className="flex items-center gap-3">
                    <Button type="submit">Save personal info</Button>
                    <Button type="button" variant="secondary" onClick={() => setProfile({ displayName: user.name, avatarUrl: '' })}>
                      Reset
                    </Button>
                  </div>
                </form>
              </Card>

              <Card title="Session snapshot">
                <div className="space-y-3 text-sm">
                  <div className="flex items-center justify-between rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                    <span className="text-text-secondary">Total balance</span>
                    <span className="font-semibold text-text-primary">{formatCurrency(totalBalance, currencyCode, settings.locale)}</span>
                  </div>
                  <div className="flex items-center justify-between rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                    <span className="text-text-secondary">Safe to spend</span>
                    <span className="font-semibold text-text-primary">{formatCurrency(safeToSpend, currencyCode, settings.locale)}</span>
                  </div>
                  <div className="flex items-center justify-between rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                    <span className="text-text-secondary">Monthly savings</span>
                    <span className="font-semibold text-accent-blue">{formatCurrency(monthlySavings, currencyCode, settings.locale)}</span>
                  </div>
                </div>
              </Card>
            </div>
          )}

          {activeTab === 'security' && (
            <div className="grid gap-4 lg:grid-cols-3">
              <Card title="Security" subtitle="Manage sign-in and session controls" className="lg:col-span-2">
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="rounded-2xl border border-dark-elevated bg-dark-bg p-4">
                    <p className="text-sm font-medium text-text-primary">Two-factor authentication</p>
                    <p className="mt-2 text-sm text-text-muted">{is2FAEnabled ? 'Enabled' : 'Not enabled'}</p>
                    {!is2FAEnabled ? (
                      <Button type="button" variant="secondary" className="mt-4 w-full" onClick={async () => {
                        try {
                          await handleOpen2FA();
                        } catch {
                          toast.error('Failed to start 2FA setup');
                        }
                      }}>
                        Enable 2FA
                      </Button>
                    ) : (
                      <Button type="button" variant="secondary" className="mt-4 w-full" onClick={async () => {
                        try {
                          await handleDisable2FA();
                        } catch {
                          toast.error('Failed to disable 2FA');
                        }
                      }}>
                        Disable 2FA
                      </Button>
                    )}
                  </div>
                  <div className="rounded-2xl border border-dark-elevated bg-dark-bg p-4">
                    <p className="text-sm font-medium text-text-primary">Password</p>
                    <p className="mt-2 text-sm text-text-muted">Change your account password</p>
                    <div className="mt-4">
                      <Button type="button" className="w-full" onClick={() => setIsPasswordModalOpen(true)}>
                        Change password
                      </Button>
                    </div>
                  </div>
                </div>
              </Card>

              <Card title="Sign out">
                <div className="space-y-3 text-sm">
                  <p className="text-text-secondary">Use this to revoke every other device while keeping the current one signed in.</p>
                  <Button type="button" className="w-full" onClick={handleLogoutOtherSessions}>
                    Sign out other devices
                  </Button>
                </div>
              </Card>
            </div>
          )}

          {activeTab === 'session' && (
            <div className="grid gap-4 lg:grid-cols-3">
              <Card title="Sessions" subtitle="Active sessions for this account" className="lg:col-span-2">
                <div className="space-y-3">
                  {isLoadingSessions ? (
                    <p className="text-sm text-text-muted">Loading sessions...</p>
                  ) : sessions.length === 0 ? (
                    <p className="text-sm text-text-muted">No active sessions found.</p>
                  ) : (
                    <div className="space-y-2">
                      {sessions.map((s) => (
                        <div key={s.session_id} className="flex items-center justify-between rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                          <div>
                            <div className="text-sm font-medium">{s.device || 'Unknown device'}</div>
                            <div className="text-xs text-text-muted">IP: {s.ip || 'Unknown'}</div>
                            <div className="text-xs text-text-muted">Last seen: {new Date(s.last_seen).toLocaleString()}</div>
                            <div className="text-xs text-text-muted">
                              Status:{' '}
                              <span className={s.revoked ? 'text-red-400 font-medium' : 'text-emerald-400 font-medium'}>
                                {s.revoked ? 'Revoked' : 'Active'}
                              </span>
                            </div>
                            <div className="text-xs text-text-muted">Created: {new Date(s.created_at).toLocaleString()}</div>
                            <div className="text-xs text-text-muted">Expires: {new Date(s.expires_at).toLocaleString()}</div>
                          </div>
                          <div className="flex items-center gap-2">
                            <Button type="button" variant="secondary" onClick={async () => {
                              try {
                                await revokeSession(s.session_id);
                                setSessions((cur) => cur.map((x) => x.session_id === s.session_id ? { ...x, revoked: true } : x));
                                toast.success('Session revoked');
                              } catch {
                                toast.error('Failed to revoke session');
                              }
                            }}>
                              Revoke
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </Card>

              <Card title="Access">
                <div className="space-y-3 text-sm">
                  <div className="rounded-2xl border border-dark-elevated bg-dark-bg p-4 text-text-secondary">
                    Use this to revoke all refresh tokens for the current user.
                  </div>
                  <Button type="button" className="w-full" onClick={handleLogoutAll}>
                    Sign out all devices
                  </Button>
                </div>
              </Card>
            </div>
          )}
        </div>
      </main>
        {isPasswordModalOpen && (
          <Modal title="Change password" onClose={() => setIsPasswordModalOpen(false)}>
            <form className="space-y-3" onSubmit={(e) => { void handleChangePassword(e); setIsPasswordModalOpen(false); }}>
              <Input
                label="Current password"
                type="password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                placeholder="Current password"
              />
              <Input
                label="New password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="New password"
              />
              <Input
                label="Confirm new password"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Confirm new password"
              />
              <div className="flex items-center gap-3">
                <Button type="submit" className="w-full" disabled={isChangingPassword}>
                  Change password
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => {
                    setCurrentPassword('');
                    setNewPassword('');
                    setConfirmPassword('');
                  }}
                >
                  Reset
                </Button>
              </div>
            </form>
          </Modal>
        )}
        {is2FAModalOpen && (
          <Modal onClose={() => { setIs2FAModalOpen(false); setTwoFASetup(null); setTwoFACode(''); }} title="Two-Factor Authentication (2FA)">
            <div className="flex flex-col items-center space-y-3">
              {twoFASetupData ? (
                <>
                  <div className="bg-muted p-3 sm:p-4 rounded-lg">
                    <img src={twoFASetupData.qr_data_url} alt="QR code" className="h-48 w-48" />
                  </div>
                  <div className="w-full rounded-2xl border border-dark-elevated bg-dark-bg p-3 text-xs sm:text-sm text-text-secondary break-all">
                    Secret: {twoFASetupData.secret}
                  </div>
                  <p className="text-xs sm:text-sm text-center text-muted-foreground">Scan this QR code with your authenticator app</p>
                  <div className="w-full space-y-1 sm:space-y-2">
                    <label className="font-medium text-xs sm:text-sm">Enter verification code</label>
                    <Input value={twoFACode} onChange={(e) => setTwoFACode(e.target.value)} placeholder="Enter 6-digit code" />
                  </div>
                </>
              ) : (
                <p>Preparing 2FA setup...</p>
              )}
            </div>
            <div className="mt-4 flex gap-2">
              <Button type="button" variant="secondary" onClick={() => { setIs2FAModalOpen(false); setTwoFASetup(null); setTwoFACode(''); }}>Cancel</Button>
              <Button type="button" onClick={async () => {
                await handleConfirm2FA();
              }}>Enable 2FA</Button>
            </div>
          </Modal>
        )}
    </Layout>
  );
}

export default SettingsPage;
