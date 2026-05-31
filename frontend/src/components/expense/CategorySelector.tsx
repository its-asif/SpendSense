import type { ExpenseCategory } from '../../types';

type CategorySelectorProps = {
  categories: ExpenseCategory[];
  value: string;
  onChange: (categoryId: string) => void;
  disabled?: boolean;
};

export function CategorySelector({ categories, value, onChange, disabled }: CategorySelectorProps) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Category</span>
      <select
        className="input"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
      >
        <option value="">Select category</option>
        {categories.map((category) => (
          <option key={category.id} value={category.id}>
            {category.name}
          </option>
        ))}
      </select>
    </label>
  );
}