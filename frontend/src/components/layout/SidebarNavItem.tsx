import { NavLink } from 'react-router-dom';

export type SidebarItem = {
  label: string;
  to: string;
  icon: JSX.Element;
};

type SidebarNavItemProps = {
  item: SidebarItem;
  collapsed: boolean;
};

export function SidebarNavItem({ item, collapsed }: SidebarNavItemProps) {
  return (
    <NavLink
      to={item.to}
      className={({ isActive }) => [
        'group flex items-center rounded-full p-3 w-12 transition-colors',
        collapsed ? 'mx-auto justify-center' : 'w-full gap-3 rounded-2xl px-4 py-3 font-semibold',
        isActive
          ? 'bg-white/10 text-white'
          : 'text-white/70 hover:bg-white/10 hover:text-white',
      ].join(' ')}
      end={item.to === '/'}
    >
      {item.icon}
      <span className="sr-only">{item.label}</span>
      {!collapsed && <span>{item.label}</span>}
    </NavLink>
  );
}
