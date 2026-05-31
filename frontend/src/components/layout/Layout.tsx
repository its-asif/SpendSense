import type { ReactNode } from 'react';
import { Sidebar } from './Sidebar';
import { useSidebarCollapsed } from './useSidebarCollapsed';

type LayoutProps = {
  children: ReactNode;
};

export function Layout({ children }: LayoutProps) {
  const { collapsed } = useSidebarCollapsed();

  return (
    <div className="min-h-screen bg-dark-bg">
      <div className="mx-auto flex min-h-screen w-full max-w-[1600px] gap-4 p-4 lg:p-6">
        <Sidebar />
        <div className={collapsed ? 'hidden lg:block lg:w-[100px]' : 'hidden lg:block lg:w-72'} aria-hidden="true" />
        <main className="flex min-w-0 flex-1 flex-col gap-4 pt-[104px]">{children}</main>
      </div>
    </div>
  );
}