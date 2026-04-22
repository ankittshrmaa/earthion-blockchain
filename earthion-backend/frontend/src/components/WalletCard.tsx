import { useState } from 'react';
import { useWallet } from '../hooks/useBlockchain';

export default function WalletCard() {
  const { wallet, balance, loading, copyAddress } = useWallet();
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    const success = copyAddress();
    if (success) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  if (loading) {
    return (
      <div className="card-hover animate-pulse" role="status" aria-label="Loading wallet">
        <div className="h-28 bg-bone-200 rounded-lg"></div>
      </div>
    );
  }

  return (
    <div className="card-hover">
      {/* Header */}
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold text-gray-900 flex items-center space-x-2">
          <svg className="w-5 h-5 text-earthion" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
          </svg>
          <span>My Wallet</span>
        </h2>
        <span 
          className="text-xs px-2 py-1 rounded-full flex items-center space-x-1 bg-earthion/10 text-earthion border border-earthion/20"
        >
          <span className="w-1.5 h-1.5 rounded-full bg-earthion"></span>
          <span>Active</span>
        </span>
      </div>

      {/* Balance - Prominent */}
      <div 
        className="rounded-xl p-5 mb-4"
        style={{ 
          background: 'linear-gradient(135deg, rgba(12, 201, 48, 0.1) 0%, rgba(12, 201, 48, 0.05) 100%)',
          border: '1px solid rgba(12, 201, 48, 0.2)'
        }}
      >
        <p className="text-sm text-gray-500 mb-1">Total Balance</p>
        <p className="text-4xl font-bold text-gray-900 flex items-baseline">
          {balance.toLocaleString()}
          <span className="text-lg text-earthion ml-2 font-medium">ion</span>
        </p>
      </div>

      {/* Address */}
      <div className="mb-4">
        <p className="text-sm text-gray-500 mb-2" id="address-label">Wallet Address</p>
        <div className="flex items-center space-x-2">
          <button
            onClick={handleCopy}
            className="flex-1 px-3 py-2.5 rounded-lg text-left font-mono text-xs break-all transition-all hover:bg-bone-200 border bg-bone-100 border-bone-200"
            style={{ borderColor: 'rgba(12, 201, 48, 0.2)' }}
            aria-labelledby="address-label"
            aria-label={copied ? 'Address copied' : 'Click to copy address'}
          >
            <span className="text-gray-700">
              {wallet?.address || 'N/A'}
            </span>
          </button>
          <button
            onClick={handleCopy}
            className="p-2.5 rounded-lg border bg-bone-100 border-bone-200 hover:bg-bone-200 transition-all"
            style={{ borderColor: 'rgba(12, 201, 48, 0.2)' }}
            aria-label="Copy address"
          >
            {copied ? (
              <svg className="w-5 h-5 text-earthion" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            ) : (
              <svg className="w-5 h-5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
              </svg>
            )}
          </button>
        </div>
        {copied && (
          <p className="text-xs text-earthion mt-1.5 flex items-center space-x-1 font-medium" role="status" aria-live="polite">
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            <span>Copied to clipboard</span>
          </p>
        )}
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-2 gap-3">
        <div 
          className="rounded-lg p-3 bg-bone-100"
        >
          <p className="text-xs text-gray-500 flex items-center space-x-1">
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m4 0h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>Transactions</span>
          </p>
          <p className="text-lg font-semibold text-gray-900">0</p>
        </div>
        <div 
          className="rounded-lg p-3 bg-bone-100"
        >
          <p className="text-xs text-gray-500 flex items-center space-x-1">
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
            <span>UTXOs</span>
          </p>
          <p className="text-lg font-semibold text-gray-900">{balance > 0 ? 1 : 0}</p>
        </div>
      </div>
    </div>
  );
}