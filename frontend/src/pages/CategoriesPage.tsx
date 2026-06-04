import { useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { Header } from '../components/layout/Header';
import { Layout } from '../components/layout/Layout';
import { Card } from '../components/common/Card';
import { Button } from '../components/common/Button';
import { Input } from '../components/common/Input';
import Modal from '../components/common/Modal';
import { createCategory, deleteCategory, listCategories, updateCategory } from '../api/categories';
import type { AuthUser, CategoryKind, ExpenseCategory } from '../types';

const ITEMS_PER_PAGE = 8;

const EMOJI_CATEGORIES = {
  Food: ['🍔', '🍕', '🥗', '☕', '🍺', '🍇', '🍎', '🥦', '🍦', '🍪', '🍽️', '🍰', '🍜', '🍣', '🍫'],
  Travel: ['🚗', '✈️', '🏠', '🏨', '🏖️', '🏔️', '🚲', '🚇', '⛽', '🗺️', '🚕', '🚂', '🛥️', '⛺', '🎡'],
  Activity: ['🎬', '🎮', '🏋️', '🎨', '🎵', '⚽', '🎭', '🎪', '🎟️', '🎳', '🎧', '🎸', '🏊', '🚴', '🏆'],
  Shopping: ['🛍️', '💰', '💵', '💳', '🛒', '🏷️', '🎁', '💎', '💼', '📈', '💸', '🧾', '🪙', '👠', '🔑'],
  Health: ['💊', '🏥', '🦷', '🩺', '💆', '🧘', '🚿', '🧼', '💈', '🩹', '🌡️', '🧬', '🔬', '😷', '🩹'],
  Utilities: ['⚡', '💧', '🔥', '⚙️', '🔧', '🔒', '📦', '📧', '📌', '💡', '🔔', '🔌', '📡', '🧹', '🧺'],
  Education: ['📚', '🎓', '📝', '📎', '✏️', '📓', '📐', '📁', '🎒', '🧩', '🔬', '🗺️', '💭', '🗣️', '📖']
};

type EmojiItem = { char: string; name: string; category: string };

const EMOJI_LIST: EmojiItem[] = [
  // Food & Drink
  { char: '🍔', name: 'burger hamburger food fastfood eat lunch dinner', category: 'Food' },
  { char: '🍕', name: 'pizza food fastfood cheese eat italian', category: 'Food' },
  { char: '🥗', name: 'salad food healthy vegetable green eat diet', category: 'Food' },
  { char: '☕', name: 'coffee tea cafe drink hot morning cup mug beverage', category: 'Food' },
  { char: '🍺', name: 'beer drink alcohol bar party pub beverage', category: 'Food' },
  { char: '🍇', name: 'grapes fruit food healthy purple sweet', category: 'Food' },
  { char: '🍎', name: 'apple fruit food healthy red eat apple', category: 'Food' },
  { char: '🥦', name: 'broccoli vegetable food healthy green tree', category: 'Food' },
  { char: '🍦', name: 'icecream sweet dessert food cold cream vanilla', category: 'Food' },
  { char: '🍪', name: 'cookie sweet dessert food biscuit chocolate chip', category: 'Food' },
  { char: '🍽️', name: 'plate fork knife dinner restaurant food kitchen table', category: 'Food' },
  { char: '🍰', name: 'cake sweet dessert food birthday celebration', category: 'Food' },
  { char: '🍜', name: 'ramen noodles soup food asian hot bowl', category: 'Food' },
  { char: '🍣', name: 'sushi fish food japanese asian seafood', category: 'Food' },
  { char: '🍫', name: 'chocolate sweet dessert food sugar cocoa bar', category: 'Food' },
  { char: '🍷', name: 'wine alcohol drink glass restaurant red wine', category: 'Food' },
  { char: '🍩', name: 'donut doughnut sweet dessert food pink', category: 'Food' },
  { char: ' popcorn', name: 'popcorn movie snack food theater butter', category: 'Food' },
  { char: '🍳', name: 'egg cooking pan breakfast food fry yolk', category: 'Food' },

  // Travel & Places
  { char: '🚗', name: 'car auto vehicle transport travel drive red car', category: 'Travel' },
  { char: '✈️', name: 'airplane plane flight travel transport airport jet sky', category: 'Travel' },
  { char: '🏠', name: 'house home building live residence structure', category: 'Travel' },
  { char: '🏨', name: 'hotel building stay vacation travel accommodation', category: 'Travel' },
  { char: '🏖️', name: 'beach vacation summer travel sea sand umbrella sun', category: 'Travel' },
  { char: '🏔️', name: 'mountain snow cold travel nature peak', category: 'Travel' },
  { char: '🚲', name: 'bicycle bike transport ride exercise cycle wheels', category: 'Travel' },
  { char: '🚇', name: 'subway metro train transport travel rail underground', category: 'Travel' },
  { char: '⛽', name: 'gas fuel petrol station car auto oil pump', category: 'Travel' },
  { char: '🗺️', name: 'map travel navigation guide location paper geography', category: 'Travel' },
  { char: '🚕', name: 'taxi cab car transport travel driver yellow cab', category: 'Travel' },
  { char: '🏕️', name: 'camp camping tent nature outdoor travel forest', category: 'Travel' },
  { char: '🎡', name: 'ferris wheel amusement park fair fun travel wheel park', category: 'Travel' },
  { char: '🚀', name: 'rocket space transport launch speed shuttle', category: 'Travel' },
  { char: '🛳️', name: 'cruise ship boat travel sea vacation harbor passenger', category: 'Travel' },
  { char: '🏟️', name: 'stadium sports game arena building matching field', category: 'Travel' },

  // Activity & Entertainment
  { char: '🎬', name: 'movie cinema film director board entertainment clapboard', category: 'Activity' },
  { char: '🎮', name: 'game controller console playstation xbox nintendo fun play', category: 'Activity' },
  { char: '🏋️', name: 'weight gym workout fitness exercise lift weights barbell', category: 'Activity' },
  { char: '🎨', name: 'art paint brush palette design creative artistic paint', category: 'Activity' },
  { char: '🎵', name: 'music note song sound audio melody tune notes', category: 'Activity' },
  { char: '⚽', name: 'soccer football ball game sport play football match', category: 'Activity' },
  { char: '🎭', name: 'theater drama mask actor performance art masks comedy tragedy', category: 'Activity' },
  { char: '🎟️', name: 'ticket movie concert show entry event pass coupon', category: 'Activity' },
  { char: '🎧', name: 'headphones music sound audio listen ears', category: 'Activity' },
  { char: '🎸', name: 'guitar instrument music rock sound electric acoustic string', category: 'Activity' },
  { char: '🏆', name: 'trophy award win prize first champion cup gold medal', category: 'Activity' },
  { char: '🎯', name: 'dart target goal hit accuracy game center bullseye', category: 'Activity' },
  { char: '🎲', name: 'dice game play boardgame luck cube game rolling', category: 'Activity' },
  { char: '🧘', name: 'yoga meditate peace mental health mindfulness exercise posture stretch', category: 'Activity' },

  // Shopping & Finance
  { char: '🛍️', name: 'bag shopping bag mall clothes store buy sale purchase', category: 'Shopping' },
  { char: '💰', name: 'money bag cash wealth rich gold finance savings loan dollar', category: 'Shopping' },
  { char: '💵', name: 'dollar cash money green currency bank note bill paper money', category: 'Shopping' },
  { char: '💳', name: 'creditcard card bank payment visa pay mastercard swipe plastic', category: 'Shopping' },
  { char: '🛒', name: 'cart shopping cart supermarket grocery buy store trolley wheels', category: 'Shopping' },
  { char: '🏷️', name: 'tag price label discount sale shopping price tag ticket hanging', category: 'Shopping' },
  { char: '🎁', name: 'gift present birthday christmas box surprise package ribbon bow', category: 'Shopping' },
  { char: '💎', name: 'diamond gem jewel expensive luxury wealth ring crystal precious', category: 'Shopping' },
  { char: '💼', name: 'briefcase work job business portfolio bag office leather suit', category: 'Shopping' },
  { char: '📈', name: 'chart up graph growth profit business stock marketing stats success', category: 'Shopping' },
  { char: '💸', name: 'money wings fly lose spend bill transaction cash losing cash dollar', category: 'Shopping' },
  { char: '🧾', name: 'receipt bill invoice tax paper transaction payment printed invoice', category: 'Shopping' },
  { char: '🪙', name: 'coin gold silver money currency cash metallic shiny cents round', category: 'Shopping' },
  { char: '👠', name: 'heels shoe fashion clothing ladies shopping footwear red heels', category: 'Shopping' },
  { char: '👗', name: 'dress fashion clothing ladies wear shopping style wardrobe', category: 'Shopping' },
  { char: '👔', name: 'tie shirt fashion clothing mens wear business office professional', category: 'Shopping' },

  // Health & Self-Care
  { char: '💊', name: 'pill capsule medicine drug pharmacy health sick doctor prescription tablet', category: 'Health' },
  { char: '🏥', name: 'hospital building clinic medical health care emergency red cross doctor', category: 'Health' },
  { char: '🦷', name: 'tooth dentist dental hygiene oral clean dentist visit toothache', category: 'Health' },
  { char: '🩺', name: 'stethoscope doctor medical health tools checkup heart beat', category: 'Health' },
  { char: '💆', name: 'massage spa relaxation health care salon beauty facial wellness', category: 'Health' },
  { char: '💈', name: 'barber haircut hair salon style beauty trim pole shaving', category: 'Health' },
  { char: '🩹', name: 'bandage bandaid cut injury heal first aid care protection', category: 'Health' },
  { char: '💉', name: 'syringe injection vaccine blood test medical doctor needle shot', category: 'Health' },
  { char: '🚿', name: 'shower bathroom water clean wash hygiene head spraying water', category: 'Health' },
  { char: '🧼', name: 'soap bubbles clean wash hygiene bathroom sink bar soap', category: 'Health' },

  // Utilities & Bills
  { char: '⚡', name: 'electricity lightning power energy voltage flash charge thunderbolt', category: 'Utilities' },
  { char: '💧', name: 'water drop rain utility hydration clean liquid droplet water bill', category: 'Utilities' },
  { char: '🔥', name: 'fire flame gas heat hot utility burn heating gas bill stove', category: 'Utilities' },
  { char: '⚙️', name: 'gear cog setting system configure tool utility options maintenance', category: 'Utilities' },
  { char: '🔧', name: 'wrench tool repair fix adjust hardware mechanic spanner tool', category: 'Utilities' },
  { char: '🔒', name: 'lock secure privacy safe key block close padlock shut', category: 'Utilities' },
  { char: '📦', name: 'box package delivery shipping post cardboard container parcel package', category: 'Utilities' },
  { char: '📧', name: 'email mail envelope letter message communication contact inbox message', category: 'Utilities' },
  { char: '🔑', name: 'key lock door unlock access open password secret keychain', category: 'Utilities' },
  { char: '💡', name: 'lightbulb idea bright intelligence power utility electricity lamp light', category: 'Utilities' },
  { char: '🔔', name: 'bell notification alert sound ring reminder bells notifications sound', category: 'Utilities' },
  { char: '🔌', name: 'plug electric wire power adapter cable utility connection wall socket', category: 'Utilities' },
  { char: '🧹', name: 'broom clean dust sweep housework utility chores sweep floor clean', category: 'Utilities' },
  { char: '🗑️', name: 'trashcan bin garbage waste disposal recycle utility dustbin trash bin', category: 'Utilities' },

  // Education & Work
  { char: '📚', name: 'books read library study school university book pile textbook read', category: 'Education' },
  { char: '🎓', name: 'graduation cap university college school degree student learn mortarboard hat', category: 'Education' },
  { char: '📝', name: 'memo pencil paper write notebook memo document letter list', category: 'Education' },
  { char: '📎', name: 'paperclip office attach document paper supply school clamp clip', category: 'Education' },
  { char: '✏️', name: 'pencil write draw office supply school tool yellow pencil graphite', category: 'Education' },
  { char: '📓', name: 'notebook book write journal diary office supply school notebook', category: 'Education' },
  { char: '📐', name: 'ruler triangle measure math school geometry tool drawing drafting', category: 'Education' },
  { char: '🎒', name: 'backpack bag school student travel carry schoolbag rucksack', category: 'Education' },
  { char: '🧩', name: 'puzzle piece game problem solve logic brain pieces connecting', category: 'Education' },
  { char: '🔬', name: 'microscope science laboratory research biology test study instrument lens', category: 'Education' },
  { char: '💭', name: 'thought bubble think idea dream reflect imagination cloud', category: 'Education' },
  { char: '🗣️', name: 'speech speak voice talk discuss lecture teacher shadow talking speaking', category: 'Education' }
];

const COLOR_OPTIONS = [
  '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4', '#FFEAA7', 
  '#DDA15E', '#BC6C25', '#6A4C93', '#FFB703', '#10B981', 
  '#3B82F6', '#8B5CF6', '#F59E0B', '#06B6D4', '#64748B', 
  '#EC4899', '#14B8A6', '#EF4444', '#84CC16', '#6366F1'
];

type IconSelectorProps = {
  value: string;
  onChange: (val: string) => void;
  label?: string;
};

function IconSelector({ value, onChange, label = "Icon" }: IconSelectorProps) {
  const [selectedCategory, setSelectedCategory] = useState<keyof typeof EMOJI_CATEGORIES>('Food');
  const [searchQuery, setSearchQuery] = useState('');

  const displayedEmojis = useMemo(() => {
    if (!searchQuery) {
      return EMOJI_CATEGORIES[selectedCategory];
    }
    const lowerQuery = searchQuery.toLowerCase().trim();
    return EMOJI_LIST.filter(item => item.name.includes(lowerQuery)).map(item => item.char);
  }, [selectedCategory, searchQuery]);

  return (
    <div className="space-y-2">
      <Input 
        label={label} 
        value={value} 
        onChange={(event) => onChange(event.target.value)} 
        placeholder="Enter or choose an emoji..."
      />
      <div className="space-y-2">
        <div className="flex gap-2 items-center justify-between">
          <span className="text-xs text-text-muted">Or choose a preset:</span>
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search emojis (e.g. food, bill)..."
            className="text-xs bg-dark-elevated text-text-primary px-2.5 py-1 rounded-lg border border-dark-elevated focus:outline-none focus:border-primary w-48"
          />
        </div>
        {!searchQuery && (
          <div className="flex gap-1 overflow-x-auto pb-1.5 border-b border-dark-elevated">
            {Object.keys(EMOJI_CATEGORIES).map((cat) => (
              <button
                key={cat}
                type="button"
                onClick={() => setSelectedCategory(cat as keyof typeof EMOJI_CATEGORIES)}
                className={`text-xs px-2.5 py-1 rounded-lg border transition-all cursor-pointer whitespace-nowrap ${
                  selectedCategory === cat 
                    ? 'bg-primary border-primary text-text-primary font-semibold' 
                    : 'bg-dark-elevated border-dark-elevated text-text-secondary hover:text-text-primary'
                }`}
              >
                {cat}
              </button>
            ))}
          </div>
        )}
        <div className="flex flex-wrap gap-2 max-h-28 overflow-y-auto p-1.5 border border-dark-elevated rounded-xl bg-dark-bg">
          {displayedEmojis.length > 0 ? (
            displayedEmojis.map((ico) => (
              <button
                key={ico}
                type="button"
                onClick={() => onChange(ico)}
                className={`text-lg p-1 rounded-lg transition-transform hover:scale-125 focus:outline-none cursor-pointer ${
                  value === ico ? 'bg-dark-elevated scale-110 border border-primary' : ''
                }`}
              >
                {ico}
              </button>
            ))
          ) : (
            <span className="text-xs text-text-muted p-1">No matching emojis found.</span>
          )}
        </div>
      </div>
    </div>
  );
}

type ColorSelectorProps = {
  value: string;
  onChange: (val: string) => void;
  label?: string;
};

function ColorSelector({ value, onChange, label = "Color" }: ColorSelectorProps) {
  return (
    <div className="space-y-2">
      <div className="flex gap-3 items-end">
        <div className="flex-1">
          <Input 
            label={label} 
            value={value} 
            onChange={(event) => onChange(event.target.value)} 
            placeholder="#22c55e"
          />
        </div>
        <div 
          className="w-10 h-10 rounded-xl border border-dark-elevated shadow-sm transition-colors"
          style={{ backgroundColor: value || 'transparent' }}
        />
      </div>
      <div className="space-y-1">
        <span className="text-xs text-text-muted">Or choose a preset (or custom):</span>
        <div className="flex flex-wrap items-center gap-2 p-1.5 border border-dark-elevated rounded-xl bg-dark-bg">
          {COLOR_OPTIONS.map((col) => (
            <button
              key={col}
              type="button"
              onClick={() => onChange(col)}
              className={`w-6 h-6 rounded-full transition-all hover:scale-125 focus:outline-none border-2 cursor-pointer ${
                value.toUpperCase() === col.toUpperCase() ? 'scale-110 border-text-primary shadow-md ring-2 ring-primary ring-offset-2 ring-offset-dark-bg' : 'border-transparent'
              }`}
              style={{ backgroundColor: col }}
              title={col}
            />
          ))}
          {/* Native Color Picker for full spectrum */}
          <div 
            className="relative w-6 h-6 rounded-full overflow-hidden border border-dark-elevated hover:scale-125 transition-transform cursor-pointer"
            title="Custom color picker"
          >
            <input
              type="color"
              value={value.startsWith('#') && value.length === 7 ? value : '#10B981'}
              onChange={(event) => onChange(event.target.value)}
              className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
            />
            <div 
              className="w-full h-full"
              style={{
                background: 'conic-gradient(from 0deg, red, yellow, green, cyan, blue, magenta, red)',
              }}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

type CategoriesPageProps = {
  user: AuthUser;
  onLogout: () => void;
};

export function CategoriesPage({ user, onLogout }: CategoriesPageProps) {
  const [activeKind, setActiveKind] = useState<CategoryKind>('EXPENSE');
  const [categories, setCategories] = useState<ExpenseCategory[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [name, setName] = useState('');
  const [icon, setIcon] = useState('');
  const [color, setColor] = useState('');
  const [editingCategory, setEditingCategory] = useState<ExpenseCategory | null>(null);
  const [pendingDelete, setPendingDelete] = useState<ExpenseCategory | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [currentPage, setCurrentPage] = useState(1);

  const refresh = async (kind: CategoryKind) => {
    setIsLoading(true);
    try {
      const items = await listCategories(kind);
      setCategories(items);
    } catch {
      toast.error('Failed to load categories');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    void refresh(activeKind);
    setCurrentPage(1);
    setName('');
    setIcon('');
    setColor('');
  }, [activeKind]);

  const filteredCategories = useMemo(() => {
    const query = searchTerm.trim().toLowerCase();
    if (!query) {
      return categories;
    }
    return categories.filter((category) => [category.name, category.icon, category.color].filter(Boolean).some((value) => String(value).toLowerCase().includes(query)));
  }, [categories, searchTerm]);

  const totalPages = Math.max(Math.ceil(filteredCategories.length / ITEMS_PER_PAGE), 1);
  const pageCategories = filteredCategories.slice((currentPage - 1) * ITEMS_PER_PAGE, currentPage * ITEMS_PER_PAGE);

  const kindLabel = activeKind === 'EXPENSE' ? 'expense' : 'income';

  return (
    <Layout>
      <Header user={user} onLogout={onLogout} />

      <div className="mb-4 flex flex-wrap gap-2">
        <Button variant={activeKind === 'EXPENSE' ? 'primary' : 'secondary'} onClick={() => setActiveKind('EXPENSE')}>
          Expense categories
        </Button>
        <Button variant={activeKind === 'INCOME' ? 'primary' : 'secondary'} onClick={() => setActiveKind('INCOME')}>
          Income categories
        </Button>
      </div>

      <section className="grid gap-4 xl:grid-cols-[1fr_1fr]">
        <Card title={`Add ${kindLabel} category`} subtitle="Custom categories are private to your account">
          <form
            className="space-y-4"
            onSubmit={async (event) => {
              event.preventDefault();
              try {
                const created = await createCategory({
                  name,
                  kind: activeKind,
                  icon: icon || undefined,
                  color: color || undefined,
                });
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
            <IconSelector label="Icon (optional)" value={icon} onChange={setIcon} />
            <ColorSelector label="Color (optional)" value={color} onChange={setColor} />
            <Button type="submit">Add category</Button>
          </form>
        </Card>

        <Card title={`${activeKind === 'EXPENSE' ? 'Expense' : 'Income'} categories`} subtitle="System defaults plus your own">
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
                      <p className="mt-1 text-xs text-text-muted">
                        {category.is_default ? 'System default' : 'Your category'} · {category.icon ?? 'No icon'} · {category.color ?? 'No color'}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      {category.is_owned && (
                        <>
                          <Button variant="secondary" onClick={() => setEditingCategory(category)}>Edit</Button>
                          <Button variant="secondary" onClick={() => setPendingDelete(category)}>Delete</Button>
                        </>
                      )}
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
            <IconSelector label="Icon" value={editingCategory.icon ?? ''} onChange={(val) => setEditingCategory({ ...editingCategory, icon: val || null })} />
            <ColorSelector label="Color" value={editingCategory.color ?? ''} onChange={(val) => setEditingCategory({ ...editingCategory, color: val || null })} />
            <div className="flex items-center gap-3">
              <Button type="submit">Save</Button>
              <Button type="button" variant="secondary" onClick={() => setEditingCategory(null)}>Cancel</Button>
            </div>
          </form>
        </Modal>
      )}

      {pendingDelete && (
        <Modal title="Confirm delete" onClose={() => setPendingDelete(null)}>
          <p className="text-sm text-text-secondary">Delete category &quot;{pendingDelete.name}&quot;?</p>
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
