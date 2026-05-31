import type { AuthUser } from '../../types';
import { useSidebarCollapsed } from './useSidebarCollapsed';
import { HeaderNotificationsButton } from './HeaderNotificationsButton';
import { HeaderSearchBar } from './HeaderSearchBar';
import { HeaderThemeToggle } from './HeaderThemeToggle';
import { HeaderProfileMenu } from './HeaderProfileMenu';

type HeaderProps = {
  user: AuthUser;
  onLogout: () => void;
};

export function Header({ user, onLogout }: HeaderProps) {
  const { collapsed } = useSidebarCollapsed();

  return (
    <header
      className="fixed right-6 top-4 z-40 flex h-16 items-center rounded-3xl border border-dark-elevated bg-dark-bg/95 px-4 shadow-[0_20px_45px_rgba(15,23,42,0.24)] backdrop-blur-xl"
      style={{ left: collapsed ? '108px' : '296px' }}
    >
      <div className="flex w-full items-center gap-4">
        <HeaderSearchBar />

        <div className="ml-auto flex items-center gap-2 text-text-primary">
          <HeaderThemeToggle />
          <HeaderNotificationsButton />
          <HeaderProfileMenu user={user} onLogout={onLogout} />
        </div>
      </div>
    </header>
  );
}