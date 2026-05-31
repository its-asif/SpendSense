type HeaderSearchBarProps = {
  placeholder?: string;
};

export function HeaderSearchBar({ placeholder = 'Search here...' }: HeaderSearchBarProps) {
  return (
    <div className="relative w-full max-w-md">
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted">
        <circle cx="11" cy="11" r="8" />
        <path d="m21 21-4.3-4.3" />
      </svg>
      <input
        type="search"
        placeholder={placeholder}
        aria-label={placeholder}
        className="flex h-10 w-full rounded-md border border-dark-elevated bg-dark-bg px-3 py-2 pl-9 pr-4 text-base ring-offset-background placeholder:text-text-muted focus-visible:outline-none focus-visible:ring-0 disabled:cursor-not-allowed disabled:opacity-50 md:text-sm"
      />
    </div>
  );
}