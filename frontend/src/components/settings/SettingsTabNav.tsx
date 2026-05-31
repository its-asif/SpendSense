import { Link } from 'react-router-dom';

type SettingsTab = {
  id: string;
  label: string;
  path: string;
};

type SettingsTabNavProps = {
  tabs: SettingsTab[];
  activeTab: string;
  onChange: (tabId: string) => void;
};

export function SettingsTabNav({ tabs, activeTab, onChange }: SettingsTabNavProps) {
  return (
    <div className="w-full">
      <div className="md:hidden w-full mb-6">
        <select
          className="h-10 w-full rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground"
          value={activeTab}
          onChange={(event) => onChange(event.target.value)}
          aria-label="Settings section"
        >
          {tabs.map((tab) => (
            <option key={tab.id} value={tab.id}>
              {tab.label}
            </option>
          ))}
        </select>
      </div>

      <div className="hidden md:block overflow-x-auto">
        <div role="tablist" aria-orientation="horizontal" className="inline-flex h-16 w-full items-center justify-start rounded-none bg-transparent p-0 text-muted-foreground" tabIndex={0}>
          {tabs.map((tab) => {
            const selected = tab.id === activeTab;

            return (
              <Link
                key={tab.id}
                role="tab"
                aria-selected={selected}
                data-state={selected ? 'active' : 'inactive'}
                to={`/settings/${tab.path}`}
                className={[
                  'inline-flex h-full items-center justify-center whitespace-nowrap rounded-none border-b-2 border-transparent px-4 py-1.5 text-sm font-medium transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
                  selected ? 'border-primary text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground',
                ].join(' ')}
              >
                {tab.label}
              </Link>
            );
          })}
        </div>
      </div>
    </div>
  );
}