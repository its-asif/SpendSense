import { useEffect, useMemo, useState } from 'react';
import { Header } from '../components/layout/Header';
import { Layout } from '../components/layout/Layout';
import { Card } from '../components/common/Card';
import { Button } from '../components/common/Button';
import { listExpenses } from '../api/expenses';
import { listIncomes } from '../api/incomes';
import { useUserSettings } from '../hooks/useUserSettings';
import { formatCurrency } from '../lib/userSettings';
import { useDashboardMeta } from '../hooks/useDashboardMeta';
import type { AuthUser, Expense, Income } from '../types';

import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  Legend,
  PieChart,
  Pie,
  Cell
} from 'recharts';

function parseLocalDate(dateStr: string): Date {
  const parts = dateStr.split('T')[0].split('-');
  if (parts.length === 3) {
    const year = parseInt(parts[0], 10);
    const month = parseInt(parts[1], 10) - 1;
    const day = parseInt(parts[2], 10);
    return new Date(year, month, day);
  }
  return new Date(dateStr);
}

type ReportsPageProps = {
  user: AuthUser;
  onLogout: () => void;
};

export function ReportsPage({ user, onLogout }: ReportsPageProps) {
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [incomes, setIncomes] = useState<Income[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  
  const [selectedFilter, setSelectedFilter] = useState('last-30-days');
  const [selectedWallet, setSelectedWallet] = useState('all');

  const settings = useUserSettings();
  const { categories, incomeCategories, wallets } = useDashboardMeta();

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setIsLoading(true);
      try {
        // Fetch up to 1000 items to cover historical trends locally
        const [expenseResponse, incomeResponse] = await Promise.all([
          listExpenses(1000),
          listIncomes(1000)
        ]);
        if (!cancelled) {
          setExpenses(expenseResponse.expenses);
          setIncomes(incomeResponse.incomes);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, []);

  // Filtered Lists
  const filteredExpenses = useMemo(() => {
    return expenses.filter(exp => {
      if (selectedWallet !== 'all' && exp.wallet_id !== selectedWallet) {
        return false;
      }
      
      const date = parseLocalDate(exp.date);
      const now = new Date();
      
      switch (selectedFilter) {
        case 'this-month':
          return date.getMonth() === now.getMonth() && date.getFullYear() === now.getFullYear();
        case 'last-30-days': {
          const limit = new Date();
          limit.setDate(now.getDate() - 30);
          return date >= limit;
        }
        case 'last-3-months': {
          const limit = new Date();
          limit.setMonth(now.getMonth() - 3);
          return date >= limit;
        }
        case 'last-6-months': {
          const limit = new Date();
          limit.setMonth(now.getMonth() - 6);
          return date >= limit;
        }
        case 'this-year':
          return date.getFullYear() === now.getFullYear();
        case 'all-time':
        default:
          return true;
      }
    });
  }, [expenses, selectedFilter, selectedWallet]);

  const filteredIncomes = useMemo(() => {
    return incomes.filter(inc => {
      if (selectedWallet !== 'all' && inc.wallet_id !== selectedWallet) {
        return false;
      }

      const date = parseLocalDate(inc.income_date);
      const now = new Date();

      switch (selectedFilter) {
        case 'this-month':
          return date.getMonth() === now.getMonth() && date.getFullYear() === now.getFullYear();
        case 'last-30-days': {
          const limit = new Date();
          limit.setDate(now.getDate() - 30);
          return date >= limit;
        }
        case 'last-3-months': {
          const limit = new Date();
          limit.setMonth(now.getMonth() - 3);
          return date >= limit;
        }
        case 'last-6-months': {
          const limit = new Date();
          limit.setMonth(now.getMonth() - 6);
          return date >= limit;
        }
        case 'this-year':
          return date.getFullYear() === now.getFullYear();
        case 'all-time':
        default:
          return true;
      }
    });
  }, [incomes, selectedFilter, selectedWallet]);

  // Totals calculations
  const totals = useMemo(() => {
    const totalIncome = filteredIncomes.reduce((sum, inc) => sum + Number(inc.amount || 0), 0);
    const totalExpense = filteredExpenses.reduce((sum, exp) => sum + Number(exp.amount || 0), 0);
    const net = totalIncome - totalExpense;
    const savingsRate = totalIncome > 0 ? Math.max(0, Math.min(100, (net / totalIncome) * 100)) : 0;

    return {
      income: totalIncome,
      expense: totalExpense,
      net,
      savingsRate
    };
  }, [filteredExpenses, filteredIncomes]);

  // Category mapping
  const categoryBreakdown = useMemo(() => {
    const grouped: Record<string, { name: string; amount: number; color: string; icon?: string }> = {};
    const categoryMap = new Map(categories.map(c => [c.id, c]));

    filteredExpenses.forEach(exp => {
      const catId = exp.category_id || 'uncategorized';
      const catObj = categoryMap.get(catId);
      
      const catName = catObj?.name || 'Uncategorized';
      const catColor = catObj?.color || '#64748B';
      const catIcon = catObj?.icon || '🏷️';

      if (!grouped[catId]) {
        grouped[catId] = {
          name: catName,
          amount: 0,
          color: catColor,
          icon: catIcon
        };
      }
      grouped[catId].amount += Number(exp.amount || 0);
    });

    const totalExpense = Object.values(grouped).reduce((sum, item) => sum + item.amount, 0);

    return Object.values(grouped)
      .map(item => ({
        ...item,
        percent: totalExpense > 0 ? Math.round((item.amount / totalExpense) * 100) : 0
      }))
      .sort((a, b) => b.amount - a.amount);
  }, [filteredExpenses, categories]);

  // Monthly trends grouping
  const monthlyTrendData = useMemo(() => {
    const monthlyMap: Record<string, { label: string; income: number; expense: number; sortKey: string }> = {};

    const getYearMonthKey = (d: Date) => {
      const year = d.getFullYear();
      const month = String(d.getMonth() + 1).padStart(2, '0');
      return `${year}-${month}`;
    };

    const getMonthLabel = (d: Date) => {
      return d.toLocaleDateString(settings.locale || 'en-US', { month: 'short', year: 'numeric' });
    };

    filteredIncomes.forEach(inc => {
      const d = parseLocalDate(inc.income_date);
      const key = getYearMonthKey(d);
      if (!monthlyMap[key]) {
        monthlyMap[key] = { label: getMonthLabel(d), income: 0, expense: 0, sortKey: key };
      }
      monthlyMap[key].income += Number(inc.amount || 0);
    });

    filteredExpenses.forEach(exp => {
      const d = parseLocalDate(exp.date);
      const key = getYearMonthKey(d);
      if (!monthlyMap[key]) {
        monthlyMap[key] = { label: getMonthLabel(d), income: 0, expense: 0, sortKey: key };
      }
      monthlyMap[key].expense += Number(exp.amount || 0);
    });

    return Object.values(monthlyMap).sort((a, b) => a.sortKey.localeCompare(b.sortKey));
  }, [filteredIncomes, filteredExpenses, settings.locale]);

  // CSV Export utility
  const handleExportCSV = () => {
    const headers = ['Date', 'Type', 'Category', 'Wallet', 'Amount', 'Currency', 'Notes/Merchant'];
    
    const walletMap = new Map(wallets.map(w => [w.id, w]));
    const categoryMap = new Map(categories.map(c => [c.id, c]));
    const incomeCategoryMap = new Map(incomeCategories.map(c => [c.id, c]));

    const rows = [
      ...filteredIncomes.map(inc => {
        const walletName = walletMap.get(inc.wallet_id)?.name || 'N/A';
        const catName = inc.category_id ? (incomeCategoryMap.get(inc.category_id)?.name || 'Uncategorized') : 'Uncategorized';
        return [
          inc.income_date,
          'Income',
          catName,
          walletName,
          inc.amount,
          inc.currency,
          inc.source_name || inc.notes || ''
        ];
      }),
      ...filteredExpenses.map(exp => {
        const walletName = walletMap.get(exp.wallet_id)?.name || 'N/A';
        const catName = categoryMap.get(exp.category_id)?.name || 'Uncategorized';
        return [
          exp.date,
          'Expense',
          catName,
          walletName,
          exp.amount,
          exp.currency,
          exp.merchant || exp.notes || ''
        ];
      })
    ];

    rows.sort((a, b) => String(b[0]).localeCompare(String(a[0])));

    const csvContent = [
      headers.join(','),
      ...rows.map(row => row.map(val => `"${String(val).replace(/"/g, '""')}"`).join(','))
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.setAttribute('href', url);
    link.setAttribute('download', `spendsense_report_${selectedFilter}_${new Date().toISOString().split('T')[0]}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <Layout>
      <Header user={user} onLogout={onLogout} />

      {/* Filter and Utility Bar */}
      <div className="mb-6 flex flex-wrap items-center justify-between gap-4 bg-dark-elevated border border-dark-elevated p-4 rounded-2xl shadow-sm">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex flex-col">
            <span className="text-xs text-text-muted mb-1 font-semibold">Time Period</span>
            <select
              value={selectedFilter}
              onChange={(e) => setSelectedFilter(e.target.value)}
              className="bg-dark-bg text-text-primary px-3 py-2 rounded-xl border border-dark-elevated focus:outline-none focus:border-primary text-sm min-w-[150px] cursor-pointer"
            >
              <option value="this-month">This Month</option>
              <option value="last-30-days">Last 30 Days</option>
              <option value="last-3-months">Last 3 Months</option>
              <option value="last-6-months">Last 6 Months</option>
              <option value="this-year">This Year (YTD)</option>
              <option value="all-time">All Time</option>
            </select>
          </div>

          <div className="flex flex-col">
            <span className="text-xs text-text-muted mb-1 font-semibold">Wallet</span>
            <select
              value={selectedWallet}
              onChange={(e) => setSelectedWallet(e.target.value)}
              className="bg-dark-bg text-text-primary px-3 py-2 rounded-xl border border-dark-elevated focus:outline-none focus:border-primary text-sm min-w-[150px] cursor-pointer"
            >
              <option value="all">All Wallets</option>
              {wallets.map((w) => (
                <option key={w.id} value={w.id}>{w.name}</option>
              ))}
            </select>
          </div>
        </div>

        <Button onClick={handleExportCSV} variant="secondary" className="flex items-center gap-2">
          <span>📥</span> Export CSV
        </Button>
      </div>

      {/* KPI Stats Cards */}
      <section className="grid gap-4 md:grid-cols-4 mb-6">
        <Card title="Income" subtitle="Total income received">
          <p className="font-mono text-2xl font-bold text-accent-green">
            {isLoading ? '...' : formatCurrency(totals.income, settings.defaultCurrency, settings.locale)}
          </p>
        </Card>
        <Card title="Expenses" subtitle="Total amount spent">
          <p className="font-mono text-2xl font-bold text-accent-amber">
            {isLoading ? '...' : formatCurrency(totals.expense, settings.defaultCurrency, settings.locale)}
          </p>
        </Card>
        <Card title="Net Cash Flow" subtitle="Surplus / deficit">
          <p className={`font-mono text-2xl font-bold ${totals.net >= 0 ? 'text-accent-blue' : 'text-accent-red'}`}>
            {isLoading ? '...' : formatCurrency(totals.net, settings.defaultCurrency, settings.locale)}
          </p>
        </Card>
        <Card title="Savings Rate" subtitle="Percentage of income saved">
          <p className="font-mono text-2xl font-bold text-indigo-400">
            {isLoading ? '...' : `${totals.savingsRate.toFixed(1)}%`}
          </p>
        </Card>
      </section>

      {/* Main Charts Area */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Monthly Trend Area Chart (2 cols width) */}
        <div className="lg:col-span-2">
          <Card title="Income vs Expense Trend" subtitle="Monthly financial flow timeline">
            {isLoading ? (
              <p className="text-sm text-text-secondary h-[350px] flex items-center justify-center">Loading trend data...</p>
            ) : monthlyTrendData.length > 0 ? (
              <div className="h-[350px]">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={monthlyTrendData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                    <defs>
                      <linearGradient id="colorIncome" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#10B981" stopOpacity={0.2}/>
                        <stop offset="95%" stopColor="#10B981" stopOpacity={0}/>
                      </linearGradient>
                      <linearGradient id="colorExpense" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#EF4444" stopOpacity={0.2}/>
                        <stop offset="95%" stopColor="#EF4444" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#1E293B" vertical={false} />
                    <XAxis 
                      dataKey="label" 
                      stroke="#64748B" 
                      tick={{ fill: '#64748B', fontSize: 11 }} 
                      axisLine={false}
                      tickLine={false}
                    />
                    <YAxis 
                      stroke="#64748B" 
                      tick={{ fill: '#64748B', fontSize: 11 }}
                      axisLine={false}
                      tickLine={false}
                      tickFormatter={(value: number) => {
                        if (value >= 1000) return `${(value / 1000).toFixed(0)}k`;
                        return String(value);
                      }}
                    />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: '#1E293B',
                        borderRadius: '12px',
                        border: '1px solid #334155',
                        color: '#F8FAFC'
                      }}
                      formatter={(value: any) => [
                        formatCurrency(Number(value || 0), settings.defaultCurrency, settings.locale),
                      ]}
                    />
                    <Legend 
                      verticalAlign="top" 
                      height={36}
                      iconType="circle"
                      wrapperStyle={{ color: '#F8FAFC', fontSize: '13px' }}
                    />
                    <Area 
                      type="monotone" 
                      dataKey="income" 
                      name="Income" 
                      stroke="#10B981" 
                      strokeWidth={2}
                      fillOpacity={1} 
                      fill="url(#colorIncome)" 
                    />
                    <Area 
                      type="monotone" 
                      dataKey="expense" 
                      name="Expenses" 
                      stroke="#EF4444" 
                      strokeWidth={2}
                      fillOpacity={1} 
                      fill="url(#colorExpense)" 
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            ) : (
              <p className="text-sm text-text-muted h-[350px] flex items-center justify-center">
                No monthly data found for the selected filters.
              </p>
            )}
          </Card>
        </div>

        {/* Category Breakdown list & Donut Chart (1 col width) */}
        <div className="lg:col-span-1">
          <Card title="Category Spending" subtitle="Breakdown of expense categories">
            {isLoading ? (
              <p className="text-sm text-text-secondary h-[350px] flex items-center justify-center">Loading breakdown...</p>
            ) : categoryBreakdown.length > 0 ? (
              <div className="space-y-6">
                <div className="h-44">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={categoryBreakdown}
                        dataKey="amount"
                        nameKey="name"
                        cx="50%"
                        cy="50%"
                        innerRadius={50}
                        outerRadius={70}
                        paddingAngle={3}
                      >
                        {categoryBreakdown.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.color} />
                        ))}
                      </Pie>
                      <Tooltip
                        contentStyle={{
                          backgroundColor: '#1E293B',
                          borderRadius: '12px',
                          border: '1px solid #334155',
                          color: '#F8FAFC'
                        }}
                        formatter={(value: any) => [
                          formatCurrency(Number(value || 0), settings.defaultCurrency, settings.locale),
                          'Spent'
                        ]}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </div>

                <div className="space-y-3 max-h-[190px] overflow-y-auto pr-1">
                  {categoryBreakdown.map((item, idx) => (
                    <div key={idx} className="flex items-center justify-between gap-3 text-sm border-b border-dark-elevated pb-2 last:border-0 last:pb-0">
                      <div className="flex items-center gap-2">
                        <span className="text-base">{item.icon || '🏷️'}</span>
                        <span className="font-medium text-text-primary">{item.name}</span>
                      </div>
                      <div className="text-right">
                        <span className="font-semibold text-text-primary block">
                          {formatCurrency(item.amount, settings.defaultCurrency, settings.locale)}
                        </span>
                        <span className="text-xs text-text-muted block">
                          {item.percent}% of total
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <p className="text-sm text-text-muted h-[350px] flex items-center justify-center text-center">
                No expense data found for the selected filters.
              </p>
            )}
          </Card>
        </div>
      </div>
    </Layout>
  );
}
