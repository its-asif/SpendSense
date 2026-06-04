import React, { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Layout } from '../components/layout/Layout';
import { Header } from '../components/layout/Header';
import { Card } from '../components/common/Card';
import { Button } from '../components/common/Button';
import { Input } from '../components/common/Input';
import Modal from '../components/common/Modal';
import { RecurringPaymentForm } from '../components/recurring/RecurringPaymentForm';
import { RecurringPaymentList } from '../components/recurring/RecurringPaymentList';
import { useDashboardMeta } from '../hooks/useDashboardMeta';
import {
  listRecurringPayments,
  createRecurringPayment,
  updateRecurringPayment,
  deleteRecurringPayment,
  payRecurringPayment,
  type RecurringPayment,
  type CreateRecurringRequest,
} from '../api/recurring';
import type { AuthUser } from '../types';

type RecurringPageProps = {
  user: AuthUser;
  onLogout: () => void;
};

export function RecurringPage({ user, onLogout }: RecurringPageProps) {
  const [payments, setPayments] = useState<RecurringPayment[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [editingPayment, setEditingPayment] = useState<RecurringPayment | null>(null);
  const [pendingDelete, setPendingDelete] = useState<RecurringPayment | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  // Pay modal state
  const [payingPayment, setPayingPayment] = useState<RecurringPayment | null>(null);
  const [paymentDate, setPaymentDate] = useState(new Date().toISOString().slice(0, 10));
  const [fineAmount, setFineAmount] = useState('0');
  const [isPaying, setIsPaying] = useState(false);

  const { categories, wallets, syncWallets } = useDashboardMeta();

  const refreshPayments = async () => {
    setIsLoading(true);
    try {
      const response = await listRecurringPayments();
      setPayments(response.recurring_payments);
    } catch {
      toast.error('Failed to load recurring payments.');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    void refreshPayments();
  }, []);

  const handleCreate = async (data: CreateRecurringRequest) => {
    const created = await createRecurringPayment(data);
    setPayments((prev) => [created, ...prev]);
    toast.success('Recurring payment configured successfully!');
  };

  const handleUpdate = async (data: CreateRecurringRequest) => {
    if (!editingPayment) return;
    const updated = await updateRecurringPayment(editingPayment.id, data);
    setPayments((prev) => prev.map((p) => (p.id === updated.id ? updated : p)));
    setEditingPayment(null);
    toast.success('Recurring payment updated successfully!');
  };

  const handleDelete = async () => {
    if (!pendingDelete) return;
    setIsDeleting(true);
    try {
      await deleteRecurringPayment(pendingDelete.id);
      setPayments((prev) => prev.filter((p) => p.id !== pendingDelete.id));
      setPendingDelete(null);
      toast.success('Recurring payment template deleted.');
    } catch {
      toast.error('Failed to delete recurring payment.');
    } finally {
      setIsDeleting(false);
    }
  };

  const handleOpenPay = (payment: RecurringPayment) => {
    setPayingPayment(payment);
    setPaymentDate(new Date().toISOString().slice(0, 10));
    setFineAmount('0');
  };

  const handleConfirmPay = async () => {
    if (!payingPayment) return;
    setIsPaying(true);
    try {
      const updated = await payRecurringPayment(payingPayment.id, {
        payment_date: paymentDate,
        fine: Number(fineAmount) || 0,
      });

      // Update state and refresh wallet balances
      setPayments((prev) => prev.map((p) => (p.id === updated.id ? updated : p)));
      setPayingPayment(null);
      void syncWallets();
      
      // Dispatch refresh events to update the dashboard and notifications immediately!
      window.dispatchEvent(new CustomEvent('spendsense-refresh-notifications'));
      window.dispatchEvent(new CustomEvent('spendsense-refresh-dashboard'));

      toast.success('Payment recorded and posted to expenses successfully!');
    } catch (err) {
      toast.error('Failed to record recurring payment.');
    } finally {
      setIsPaying(false);
    }
  };

  const isLate = payingPayment && paymentDate > payingPayment.deadline;

  return (
    <Layout>
      <Header user={user} onLogout={onLogout} />

      <section className="grid gap-4 xl:grid-cols-[0.8fr_1.2fr]">
        <Card title="Configure Recurring Payment" subtitle="Define automated recurring templates">
          <RecurringPaymentForm
            categories={categories}
            wallets={wallets}
            onSubmit={handleCreate}
          />
        </Card>

        <Card
          title="Recurring Obligations"
          subtitle={`${payments.length} templates configured`}
          action={<Button variant="secondary" onClick={() => void refreshPayments()}>Refresh</Button>}
        >
          <RecurringPaymentList
            payments={payments}
            isLoading={isLoading}
            onEdit={setEditingPayment}
            onDelete={setPendingDelete}
            onPay={handleOpenPay}
          />
        </Card>
      </section>

      {/* Edit Modal */}
      {editingPayment && (
        <Modal title="Edit Recurring Payment" onClose={() => setEditingPayment(null)}>
          <RecurringPaymentForm
            categories={categories}
            wallets={wallets}
            onSubmit={handleUpdate}
            initialPayment={editingPayment}
            onCancel={() => setEditingPayment(null)}
          />
        </Modal>
      )}

      {/* Confirm Delete Modal */}
      {pendingDelete && (
        <Modal title="Confirm Delete" onClose={() => setPendingDelete(null)}>
          <p className="text-sm text-text-secondary">
            Are you sure you want to delete the recurring payment template for{' '}
            <strong className="text-text-primary">{pendingDelete.title}</strong>?
            This will not affect previously posted expenses.
          </p>
          <div className="mt-4 flex items-center justify-end gap-3">
            <Button variant="secondary" onClick={() => setPendingDelete(null)} disabled={isDeleting}>
              Cancel
            </Button>
            <Button onClick={handleDelete} disabled={isDeleting}>
              {isDeleting ? 'Deleting...' : 'Delete Template'}
            </Button>
          </div>
        </Modal>
      )}

      {/* Pay / Record Payment Modal */}
      {payingPayment && (
        <Modal title="Record Payment" onClose={() => setPayingPayment(null)}>
          <div className="space-y-4">
            <div>
              <h4 className="text-sm font-semibold text-text-primary">{payingPayment.title}</h4>
              <p className="text-xs text-text-secondary">
                Amount: {new Intl.NumberFormat('en-US', { style: 'currency', currency: payingPayment.currency }).format(payingPayment.amount)}
              </p>
              <p className="text-xs text-text-muted">
                Deadline: {payingPayment.deadline}
              </p>
            </div>

            <Input
              label="Payment Date"
              type="date"
              value={paymentDate}
              onChange={(e) => setPaymentDate(e.target.value)}
              required
            />

            {isLate && (
              <div className="rounded-xl border border-accent-red/20 bg-accent-red/10 p-3 text-xs text-accent-red">
                ⚠️ <strong>Overdue Notice:</strong> The selected payment date ({paymentDate}) is after the payment deadline ({payingPayment.deadline}). A penalty/fine may apply.
              </div>
            )}

            <Input
              label="Fine / Penalty Amount (Optional)"
              type="number"
              min="0"
              step="0.01"
              value={fineAmount}
              onChange={(e) => setFineAmount(e.target.value)}
              placeholder="0.00"
            />

            <div className="mt-4 flex items-center justify-end gap-3 pt-4 border-t border-dark-elevated">
              <Button variant="secondary" onClick={() => setPayingPayment(null)} disabled={isPaying}>
                Cancel
              </Button>
              <Button onClick={handleConfirmPay} disabled={isPaying}>
                {isPaying ? 'Confirming...' : 'Confirm Payment'}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </Layout>
  );
}
