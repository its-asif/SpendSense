import { Navigate, Route, Routes } from 'react-router-dom';
import { useState } from 'react';
import { Dashboard } from './pages/Dashboard';
import { ExpensesPage } from './pages/ExpensesPage';
import { IncomesPage } from './pages/IncomesPage';
import { WalletsPage } from './pages/WalletsPage';
import { CategoriesPage } from './pages/CategoriesPage';
import { ReportsPage } from './pages/ReportsPage';
import { SettingsPage } from './pages/SettingsPage';
import { Login } from './pages/Login';
import { clearSession, readSession } from './lib/storage';
import { logout } from './api/auth';
import type { AuthSession } from './types';

export default function App() {
  const [session, setSession] = useState<AuthSession | null>(() => readSession());

  const handleLogout = async () => {
    if (session?.refreshToken) {
      try {
        await logout(session.refreshToken);
      } catch {
        // Ignore logout failures and clear local state anyway.
      }
    }

    clearSession();
    setSession(null);
  };

  return (
    <Routes>
      <Route path="/auth" element={session ? <Navigate to="/" replace /> : <Login onAuthenticated={setSession} />} />
      <Route path="/" element={session ? <Dashboard user={session.user} onLogout={handleLogout} /> : <Navigate to="/auth" replace />} />
      <Route path="/expenses" element={session ? <ExpensesPage user={session.user} onLogout={handleLogout} /> : <Navigate to="/auth" replace />} />
      <Route path="/incomes" element={session ? <IncomesPage user={session.user} onLogout={handleLogout} /> : <Navigate to="/auth" replace />} />
      <Route path="/wallets" element={session ? <WalletsPage user={session.user} onLogout={handleLogout} /> : <Navigate to="/auth" replace />} />
      <Route path="/categories" element={session ? <CategoriesPage user={session.user} onLogout={handleLogout} /> : <Navigate to="/auth" replace />} />
      <Route path="/reports" element={session ? <ReportsPage user={session.user} onLogout={handleLogout} /> : <Navigate to="/auth" replace />} />
      <Route path="/settings" element={<Navigate to={session ? '/settings/account' : '/auth'} replace />} />
      <Route path="/settings/:tab" element={session ? <SettingsPage user={session.user} onLogout={handleLogout} /> : <Navigate to="/auth" replace />} />
      <Route path="*" element={<Navigate to={session ? '/' : '/auth'} replace />} />
    </Routes>
  );
}