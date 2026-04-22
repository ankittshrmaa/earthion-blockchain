import { useState, useCallback } from 'react';
import { useSendTransaction } from '../hooks/useBlockchain';

const isValidAddress = (addr: string) => /^[0-9a-fA-F]{40}$/.test(addr);
const isValidAmount = (amt: number) => amt > 0 && amt <= 1_000_000_000 && Number.isInteger(amt);

const getAddressError = (addr: string): string | null => {
  if (!addr) return null;
  if (!/^[0-9a-fA-F]+$/.test(addr)) return 'Invalid characters';
  if (addr.length !== 40) return `Need ${40 - addr.length} more chars`;
  return null;
};

const getAmountError = (amt: string): string | null => {
  if (!amt) return null;
  if (isNaN(Number(amt))) return 'Invalid number';
  const num = Number(amt);
  if (num <= 0) return 'Amount must be positive';
  if (num > 1_000_000_000) return 'Max 1B ion';
  if (!Number.isInteger(num)) return 'Must be whole number';
  return null;
};

export default function SendTransaction() {
  const [to, setTo] = useState('');
  const [amount, setAmount] = useState('');
  const [touched, setTouched] = useState({ to: false, amount: false });
  const { sending, txId, error, send, reset } = useSendTransaction();

  const handleSend = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    const addr = to.trim().toLowerCase();
    const amt = parseInt(amount, 10);
    if (isValidAddress(addr) && isValidAmount(amt)) await send(addr, amt);
  }, [to, amount, send]);

  const handleBlur = (field: 'to' | 'amount') => {
    setTouched(prev => ({ ...prev, [field]: true }));
  };

  const validated = isValidAddress(to) && isValidAmount(parseInt(amount) || 0);
  const toError = touched.to ? getAddressError(to) : null;
  const amountError = touched.amount ? getAmountError(amount) : null;

  return (
    <div className="card-hover">
      <h2 className="text-lg font-semibold text-gray-900 flex items-center space-x-2 mb-4">
        <svg className="w-5 h-5 text-earthion" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
        </svg>
        <span>Send ion</span>
      </h2>
      
      {/* Success State */}
      {txId ? (
        <div className="space-y-4">
          <div 
            className="rounded-lg px-4 py-3 flex items-center justify-center space-x-2"
            style={{ 
              background: 'rgba(12, 201, 48, 0.1)',
              border: '1px solid rgba(12, 201, 48, 0.2)'
            }}
          >
            <svg className="w-5 h-5 text-earthion" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            <span className="text-earthion font-semibold">Transaction Sent!</span>
          </div>
          
          <div className="rounded-lg p-3 bg-bone-100">
            <p className="text-xs text-gray-500 mb-1">Transaction ID</p>
            <code className="text-xs font-mono text-gray-700 break-all">{txId}</code>
          </div>
          
          <button 
            onClick={reset} 
            className="w-full py-2.5 rounded-lg font-medium bg-bone-100 text-gray-700 hover:bg-bone-200 transition-colors"
          >
            Send Another
          </button>
        </div>
      ) : (
        /* Form */
        <form onSubmit={handleSend} className="space-y-4" noValidate>
          {/* Recipient */}
          <div>
            <label htmlFor="recipient" className="block text-sm font-medium text-gray-700 mb-1.5">
              Recipient Address
            </label>
            <input
              id="recipient"
              type="text"
              value={to}
              onChange={e => setTo(e.target.value)}
              onBlur={() => handleBlur('to')}
              placeholder="0x..."
              className="input font-mono text-sm"
              aria-describedby={toError ? 'to-error' : undefined}
              autoComplete="off"
            />
            {toError && (
              <p id="to-error" className="text-xs text-red-500 mt-1" role="alert">
                {toError}
              </p>
            )}
            {to && !toError && isValidAddress(to) && (
              <p className="text-xs text-earthion mt-1 flex items-center space-x-1 font-medium" role="alert">
              <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              <span>Valid address</span>
            </p>
            )}
          </div>

          {/* Amount */}
          <div>
            <label htmlFor="amount" className="block text-sm font-medium text-gray-700 mb-1.5">
              Amount (ion)
            </label>
            <div className="relative">
              <input
                id="amount"
                type="text"
                value={amount}
                onChange={e => setAmount(e.target.value.replace(/[^0-9]/g, ''))}
                onBlur={() => handleBlur('amount')}
                placeholder="0"
                className="input pr-12"
                aria-describedby={amountError ? 'amount-error' : undefined}
              />
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 text-sm">ion</span>
            </div>
            {amountError && (
              <p id="amount-error" className="text-xs text-red-500 mt-1" role="alert">
                {amountError}
              </p>
            )}
          </div>

          {/* Error */}
          {error && (
            <div className="bg-red-50 text-red-600 px-4 py-2 rounded-lg text-sm flex items-center space-x-2">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <span>{error}</span>
            </div>
          )}

          {/* Send Button */}
          <button
            type="submit"
            disabled={sending || !validated}
            className="btn btn-primary w-full py-3 flex items-center justify-center space-x-2"
          >
            {sending ? (
              <>
                <svg className="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                <span>Sending...</span>
              </>
            ) : (
              <>
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                </svg>
                <span>Send ion</span>
              </>
            )}
          </button>
        </form>
      )}
    </div>
  );
}