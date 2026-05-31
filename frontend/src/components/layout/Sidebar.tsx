import { useSidebarCollapsed } from './useSidebarCollapsed';
import { SidebarBrand } from './SidebarBrand';
import { SidebarNavItem, type SidebarItem } from './SidebarNavItem';

const items: SidebarItem[] = [
  {
    label: 'Dashboard',
    to: '/',
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-5 w-5">
        <path d="M15 21v-8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v8" />
        <path d="M3 10a2 2 0 0 1 .709-1.528l7-5.999a2 2 0 0 1 2.582 0l7 5.999A2 2 0 0 1 21 10v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
      </svg>
    ),
  },
  {
    label: 'Expenses',
    to: '/expenses',
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-5 w-5">
        <path d="M4 19V5" />
        <path d="M8 19V11" />
        <path d="M12 19V8" />
        <path d="M16 19V14" />
        <path d="M20 19V3" />
      </svg>
    ),
  },
  {
    label: 'Incomes',
    to: '/incomes',
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-5 w-5">
        <path d="M4 17l6-6 4 4 6-6" />
        <path d="M14 5h6v6" />
      </svg>
    ),
  },
  {
    label: 'Wallets',
    to: '/wallets',
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-5 w-5">
        <rect width="20" height="14" x="2" y="5" rx="2" />
        <line x1="2" x2="22" y1="10" y2="10" />
      </svg>
    ),
  },
  {
    label: 'Categories',
    to: '/categories',
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-5 w-5">
        <path d="M7 3h10a2 2 0 0 1 2 2v2H5V5a2 2 0 0 1 2-2Z" />
        <path d="M5 7h14v12H5z" />
        <path d="M9 11h6" />
        <path d="M9 15h6" />
      </svg>
    ),
  },
  {
    label: 'Reports',
    to: '/reports',
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-5 w-5">
        <path d="M12 20v-6" />
        <path d="M18 20V10" />
        <path d="M6 20V4" />
      </svg>
    ),
  },
  {
    label: 'Settings',
    to: '/settings/account',
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-5 w-5">
        <path d="M12 20a8 8 0 1 0 0-16 8 8 0 0 0 0 16Z" />
        <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z" />
        <path d="M12 2v2" />
        <path d="M12 22v-2" />
        <path d="m17 20.66-1-1.73" />
        <path d="M11 10.27 7 3.34" />
        <path d="m20.66 17-1.73-1" />
        <path d="m3.34 7 1.73 1" />
        <path d="M14 12h8" />
        <path d="M2 12h2" />
        <path d="m20.66 7-1.73 1" />
        <path d="m3.34 17 1.73-1" />
        <path d="m17 3.34-1 1.73" />
        <path d="m11 13.73-4 6.93" />
      </svg>
    ),
  },
];

export function Sidebar() {
  const { collapsed, toggleCollapsed } = useSidebarCollapsed();

  return (
    <aside className={[
      'rounded-2xl  hidden lg:fixed lg:inset-y-4 lg:left-4 lg:z-50 lg:flex lg:flex-col bg-[#1D4ED8] text-white shadow-[0_24px_60px_-24px_rgba(29,78,216,0.65)] transition-all duration-200',
      collapsed ? 'w-[80px]' : 'w-72',
    ].join(' ')}>
      <div className={collapsed ? 'flex h-[88px] items-center justify-center' : 'p-5'}>
        <SidebarBrand collapsed={collapsed} />
      </div>

      <nav className={[
        'flex-1 overflow-auto py-4',
        collapsed ? 'mx-4 rounded-full bg-[#1D4ED8] px-0' : '',
      ].join(' ')}>
        <div className={collapsed ? 'flex flex-col gap-2' : 'space-y-2'}>
        {items.map((item) => (
          <SidebarNavItem key={item.label} item={item} collapsed={collapsed} />
        ))}
        </div>
      </nav>

      <div className="p-4">
        <button
          type="button"
          onClick={toggleCollapsed}
          className="flex w-full items-center justify-center rounded-2xl border border-white/15 bg-white/10 py-3 text-sm font-semibold text-white/90 transition-colors hover:bg-white/15"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="mr-2 h-4 w-4">
            {collapsed ? <path d="m9 18 6-6-6-6" /> : <path d="m15 18-6-6 6-6" />}
          </svg>
          {!collapsed && <span>Minimize sidebar</span>}
        </button>
      </div>

    </aside>
  );
}