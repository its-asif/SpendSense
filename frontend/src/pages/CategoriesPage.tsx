import { useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { Header } from '../components/layout/Header';
import { Layout } from '../components/layout/Layout';
import { Card } from '../components/common/Card';
import { Button } from '../components/common/Button';
import { Input } from '../components/common/Input';
import Modal from '../components/common/Modal';
import { createCategory, deleteCategory, listCategories, updateCategory } from '../api/categories';
import type { AuthUser, ExpenseCategory } from '../types';

const ITEMS_PER_PAGE = 8;

type CategoriesPageProps = {
  user: AuthUser;
  onLogout: () => void;
};

export function CategoriesPage({ user, onLogout }: CategoriesPageProps) {
  const [categories, setCategories] = useState<ExpenseCategory[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [name, setName] = useState('');
  const [icon, setIcon] = useState('');
  const [color, setColor] = useState('');
  const [editingCategory, setEditingCategory] = useState<ExpenseCategory | null>(null);
  const [pendingDelete, setPendingDelete] = useState<ExpenseCategory | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [currentPage, setCurrentPage] = useState(1);

  const refresh = async () => {
    setIsLoading(true);
    try {
      const items = await listCategories();
      setCategories(items);
    } catch {
      toast.error('Failed to load categories');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    void refresh();
  }, []);

  const filteredCategories = useMemo(() => {
    const query = searchTerm.trim().toLowerCase();
    if (!query) {
      return categories;
    }

    return categories.filter((category) => [category.name, category.icon, category.color].filter(Boolean).some((value) => String(value).toLowerCase().includes(query)));
  }, [categories, searchTerm]);

  const totalPages = Math.max(Math.ceil(filteredCategories.length / ITEMS_PER_PAGE), 1);
  const pageCategories = filteredCategories.slice((currentPage - 1) * ITEMS_PER_PAGE, currentPage * ITEMS_PER_PAGE);

  return (
    <Layout>
      <Header user={user} onLogout={onLogout} />

      <section className="grid gap-4 xl:grid-cols-[1fr_1fr]">
        <Card title="Add category" subtitle="Create custom categories for expenses and incomes">
          <form
            className="space-y-4"
            onSubmit={async (event) => {
              event.preventDefault();
              try {
                const created = await createCategory({ name, icon: icon || undefined, color: color || undefined });
                setCategories((current) => [created, ...current]);
                setCurrentPage(1);
                setName('');
                setIcon('');
                setColor('');
                toast.success('Category created');
              } catch {
                toast.error('Failed to create category');
              }
            }}
          >
            <Input label="Name" value={name} onChange={(event) => setName(event.target.value)} required />
            <Input label="Icon (optional)" value={icon} onChange={(event) => setIcon(event.target.value)} />
            <Input label="Color (optional)" value={color} onChange={(event) => setColor(event.target.value)} placeholder="#22c55e" />
            <Button type="submit">Add category</Button>
          </form>
        </Card>

        <Card title="Manage categories" subtitle="Edit or remove categories">
          {isLoading ? (
            <p className="text-sm text-text-secondary">Loading categories...</p>
          ) : categories.length === 0 ? (
            <p className="text-sm text-text-secondary">No categories yet.</p>
          ) : filteredCategories.length === 0 ? (
            <p className="text-sm text-text-secondary">No matching categories found.</p>
          ) : (
            <>
              <div className="mb-4 grid gap-3 md:grid-cols-[1fr_auto] md:items-end">
                <Input
                  label="Search"
                  value={searchTerm}
                  onChange={(event) => {
                    setSearchTerm(event.target.value);
                    setCurrentPage(1);
                  }}
                  placeholder="Search name, icon, or color..."
                />
                <p className="text-xs text-text-muted">Page {currentPage} of {totalPages}</p>
              </div>

              <div className="space-y-3">
                {pageCategories.map((category) => (
                  <div key={category.id} className="flex items-center justify-between gap-3 rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                    <div>
                      <p className="font-semibold text-text-primary">{category.name}</p>
                      <p className="mt-1 text-xs text-text-muted">{category.icon ?? 'No icon'} · {category.color ?? 'No color'}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button variant="secondary" onClick={() => setEditingCategory(category)}>Edit</Button>
                      <Button variant="secondary" onClick={() => setPendingDelete(category)}>Delete</Button>
                    </div>
                  </div>
                ))}
              </div>

              <div className="mt-4 flex items-center justify-between gap-3">
                <Button variant="secondary" onClick={() => setCurrentPage((current) => Math.max(current - 1, 1))} disabled={currentPage === 1}>
                  Previous
                </Button>
                <Button variant="secondary" onClick={() => setCurrentPage((current) => Math.min(current + 1, totalPages))} disabled={currentPage >= totalPages}>
                  Next
                </Button>
              </div>
            </>
          )}
        </Card>
      </section>

      {editingCategory && (
        <Modal title="Edit category" onClose={() => setEditingCategory(null)}>
          <form
            className="space-y-4"
            onSubmit={async (event) => {
              event.preventDefault();
              try {
                const updated = await updateCategory(editingCategory.id, {
                  name: editingCategory.name,
                  icon: editingCategory.icon ?? undefined,
                  color: editingCategory.color ?? undefined,
                });
                setCategories((current) => current.map((item) => (item.id === updated.id ? updated : item)));
                setEditingCategory(null);
                toast.success('Category updated');
              } catch {
                toast.error('Failed to update category');
              }
            }}
          >
            <Input label="Name" value={editingCategory.name} onChange={(event) => setEditingCategory({ ...editingCategory, name: event.target.value })} required />
            <Input label="Icon" value={editingCategory.icon ?? ''} onChange={(event) => setEditingCategory({ ...editingCategory, icon: event.target.value || null })} />
            <Input label="Color" value={editingCategory.color ?? ''} onChange={(event) => setEditingCategory({ ...editingCategory, color: event.target.value || null })} />
            <div className="flex items-center gap-3">
              <Button type="submit">Save</Button>
              <Button type="button" variant="secondary" onClick={() => setEditingCategory(null)}>Cancel</Button>
            </div>
          </form>
        </Modal>
      )}

      {pendingDelete && (
        <Modal title="Confirm delete" onClose={() => setPendingDelete(null)}>
          <p className="text-sm text-text-secondary">Delete category "{pendingDelete.name}"?</p>
          <div className="mt-4 flex items-center gap-3">
            <Button variant="secondary" onClick={() => setPendingDelete(null)}>Cancel</Button>
            <Button
              onClick={async () => {
                try {
                  await deleteCategory(pendingDelete.id);
                  setCategories((current) => current.filter((item) => item.id !== pendingDelete.id));
                  setPendingDelete(null);
                  toast.success('Category deleted');
                } catch {
                  toast.error('Failed to delete category');
                }
              }}
            >
              Delete
            </Button>
          </div>
        </Modal>
      )}
    </Layout>
  );
}
