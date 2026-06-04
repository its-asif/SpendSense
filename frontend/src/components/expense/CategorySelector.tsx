import type { CategoryKind, ExpenseCategory } from '../../types';

type CategorySelectorProps = {
  categories: ExpenseCategory[];
  value: string;
  onChange: (categoryId: string) => void;
  disabled?: boolean;
  kind?: CategoryKind;
  label?: string;
};

export function CategorySelector({ categories, value, onChange, disabled, kind, label = 'Category' }: CategorySelectorProps) {
  const filtered = kind ? categories.filter((category) => category.kind === kind) : categories;

  return (
    <label className="block">
      <span className="mb-1.5 block text-xs font-semibold text-text-secondary">{label}</span>
      <select
        className="input"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
      >
        <option value="">Select category</option>
        {filtered.map((category) => (
          <option key={category.id} value={category.id}>
            {category.is_default ? category.name : `${category.name} (yours)`}
          </option>
        ))}
      </select>
    </label>
  );
}
