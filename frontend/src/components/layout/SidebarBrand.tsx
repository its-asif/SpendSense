type SidebarBrandProps = {
  collapsed: boolean;
};

export function SidebarBrand({ collapsed }: SidebarBrandProps) {
  return (
    <div className={collapsed ? 'flex h-[88px] items-center justify-center' : 'rounded-[22px] border border-white/15 bg-white/10 px-4 py-4 backdrop-blur-sm'}>
      <a className="flex items-center" href="/" aria-label="SpendSense home">
        <div className="relative flex h-12 w-12 items-center justify-center overflow-hidden rounded-full bg-white text-[#1D4ED8] shadow-lg shadow-blue-950/20">
          <svg xmlns="http://www.w3.org/2000/svg" width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="relative z-10 text-[#1D4ED8]">
            <polyline points="22 7 13.5 15.5 8.5 10.5 2 17" />
            <polyline points="16 7 22 7 22 13" />
          </svg>
        </div>
        {!collapsed && (
          <div className="ml-3">
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-blue-100">SpendSense</p>
            <p className="text-sm font-semibold text-white">Finances, simplified</p>
            <p className="text-xs text-blue-100/90">Dashboard overview</p>
          </div>
        )}
      </a>
    </div>
  );
}
