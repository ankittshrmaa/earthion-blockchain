import { useState, useEffect, useCallback, useRef } from 'react';
import { blockchainAPI, type ChainStats, type Block, type WalletInfo } from '../services/api';

export interface ConnectionState {
  isOnline: boolean;
  isReconnecting: boolean;
  lastError: string | null;
}

export function useBlockchain() {
  const [stats, setStats] = useState<ChainStats | null>(null);
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [wallet, setWallet] = useState<WalletInfo | null>(null);
  const [balance, setBalance] = useState(0);
  const [loading, setLoading] = useState(true);
  const [connection, setConnection] = useState<ConnectionState>({
    isOnline: true, isReconnecting: false, lastError: null
  });
  const retryRef = useRef(0);
  const intervalRef = useRef<number | null>(null);

  // Exponential backoff delay calculation
  const getBackoffDelay = (attempt: number): number => {
    const baseDelay = 1000;
    const maxDelay = 30000;
    const delay = Math.min(baseDelay * Math.pow(2, attempt), maxDelay);
    return delay;
  };

  const loadAll = useCallback(async () => {
    try {
      const [s, b, w, bal] = await Promise.all([
        blockchainAPI.getStats(),
        blockchainAPI.getBlocks(10),
        blockchainAPI.getAddress(),
        blockchainAPI.getBalance(),
      ]);
      setStats(s.data.data || s.data);
      setBlocks(b.data.blocks || []);
      setWallet(w.data);
      setBalance(bal.data.balance);
      setConnection({ isOnline: true, isReconnecting: false, lastError: null });
      setLoading(false);
      retryRef.current = 0;
    } catch (e) {
      const err = e as Error;
      retryRef.current++;
      setConnection(prev => ({
        ...prev,
        isOnline: retryRef.current >= 5 ? false : true,
        isReconnecting: retryRef.current < 5,
        lastError: err.message,
      }));
    }
  }, []);

  // Smart polling with exponential backoff
  const startPolling = useCallback(() => {
    const scheduleNext = () => {
      const delay = getBackoffDelay(retryRef.current);
      intervalRef.current = window.setTimeout(() => {
        loadAll();
        scheduleNext();
      }, Math.min(delay, 30000));
    };
    scheduleNext();
  }, [loadAll]);

  useEffect(() => {
    loadAll();
    startPolling();
    return () => {
      if (intervalRef.current) clearTimeout(intervalRef.current);
    };
  }, [loadAll, startPolling]);

  const refresh = useCallback(() => {
    retryRef.current = 0;
    setLoading(true);
    loadAll();
    // Restart polling with fresh delay
    if (intervalRef.current) clearTimeout(intervalRef.current);
    startPolling();
  }, [loadAll, startPolling]);

  return { stats, blocks, wallet, balance, loading, connection, refresh };
}

export function useWallet() {
  const [wallet, setWallet] = useState<WalletInfo | null>(null);
  const [balance, setBalance] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      try {
        const [w, b] = await Promise.all([blockchainAPI.getAddress(), blockchainAPI.getBalance()]);
        setWallet(w.data);
        setBalance(b.data.balance);
      } catch (e) {
        console.error('Wallet load error:', e);
      } finally {
        setLoading(false);
      }
    };
    load();
    const id = setInterval(load, 10000);
    return () => clearInterval(id);
  }, []);

  const copyAddress = useCallback(() => {
    if (wallet?.address) {
      navigator.clipboard.writeText(wallet.address);
      return true;
    }
    return false;
  }, [wallet]);

  return { wallet, balance, loading, copyAddress };
}

export function useMining() {
  const [mining, setMining] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const mine = useCallback(async () => {
    if (mining) return;
    setMining(true);
    setError(null);
    try {
      await blockchainAPI.mine();
    } catch (e) {
      const err = e as Error;
      // Map common errors to user-friendly messages
      let message = err.message || 'Mining failed';
      if (message.includes('rate limit') || message.includes('Rate limit')) {
        message = 'Rate limit exceeded. Please wait and try again.';
      } else if (message.includes('Connection')) {
        message = 'Cannot connect to server. Please check if the backend is running.';
      } else if (message.includes('insufficient')) {
        message = 'Insufficient balance for transaction.';
      }
      setError(message);
    } finally {
      setMining(false);
    }
  }, [mining]);

  return { mining, error, mine };
}

export function useSendTransaction() {
  const [sending, setSending] = useState(false);
  const [txId, setTxId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const send = useCallback(async (to: string, amount: number) => {
    setSending(true);
    setError(null);
    setTxId(null);
    try {
      const r = await blockchainAPI.send(to, amount);
      setTxId(r.data.txid);
    } catch (e) {
      const err = e as Error;
      let message = err.message || 'Transaction failed';
      if (message.includes('rate limit') || message.includes('Rate limit')) {
        message = 'Rate limit exceeded. Please wait and try again.';
      } else if (message.includes('Connection')) {
        message = 'Cannot connect to server.';
      } else if (message.includes('insufficient')) {
        message = 'Insufficient balance.';
      } else if (message.includes('Invalid')) {
        message = 'Invalid address or amount.';
      }
      setError(message);
    } finally {
      setSending(false);
    }
  }, []);

  const reset = useCallback(() => { setTxId(null); setError(null); }, []);

  return { sending, txId, error, send, reset };
}

export function useOffline() {
  const [offline, setOffline] = useState(false);

  useEffect(() => {
    const handleOffline = () => setOffline(true);
    const handleOnline = () => setOffline(false);

    window.addEventListener('offline', handleOffline);
    window.addEventListener('online', handleOnline);

    return () => {
      window.removeEventListener('offline', handleOffline);
      window.removeEventListener('online', handleOnline);
    };
  }, []);

  return offline;
}