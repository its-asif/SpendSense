import type { AuthUser } from '../../types';

type HeaderUserButtonProps = {
  user: AuthUser;
};

function getInitials(name: string) {
  return name
    .split(' ')
    .map((part) => part[0])
    .join('')
    .slice(0, 2)
    .toUpperCase();
}

export function HeaderUserButton({ user }: HeaderUserButtonProps) {
  return (
    <button
      type="button"
      className="inline-flex h-9 w-9 items-center justify-center rounded-full bg-transparent p-0 transition-colors hover:bg-transparent"
      aria-label={user.name}
    >
      <span className="relative flex h-8 w-8 shrink-0 overflow-hidden rounded-full border border-dark-elevated bg-dark-bg">
        <span className="flex h-full w-full items-center justify-center text-xs font-semibold text-text-primary">
          {getInitials(user.name)}
        </span>
      </span>
    </button>
  );
}