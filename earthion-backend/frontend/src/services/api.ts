import axios from 'axios';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8000';

// Types
export interface Block {
  index: number;
  timestamp: number;
  prevHash: string;
  merkleRoot: string;
  hash: string;
  nonce: number;
  difficulty: number;
  txCount: number;
  transactions?: unknown[];
}

export interface ChainStats {
  height: number;
  difficulty: number;
  totalWork: number;
  currentReward: number;
  totalMined: number;
  maxSupply: number;
}

export interface WalletInfo {
  address: string;
  raw: string;
}

// Simple API client
const api = axios.create({
  baseURL: API_URL,
  timeout: 30000,
});

// Global error interceptor
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      const message = error.response.data?.detail || error.response.data?.message || 'Request failed';
      console.error(`API Error: ${error.config?.url} - ${message}`);
      return Promise.reject(new Error(message));
    }
    if (error.request) {
      console.error('Network error: No response received');
      return Promise.reject(new Error('Network error. Please check your connection.'));
    }
    return Promise.reject(error);
  }
);

// API Functions
export const blockchainAPI = {
  getStats: () => api.get<{data: ChainStats}>('/api/stats'),
  getBlocks: (limit = 10) => api.get<{blocks: Block[]}>('/api/blocks', { params: { limit } }),
  getAddress: () => api.get<{address: string; raw: string}>('/api/wallet/address'),
  getBalance: () => api.get<{balance: number}>('/api/wallet/balance'),
  mine: () => api.post('/api/mining/mine'),
  send: (to: string, amount: number) => {
    if (!/^[0-9a-fA-F]{40}$/.test(to)) throw new Error('Invalid address');
    if (amount <= 0 || amount > 1_000_000_000) throw new Error('Invalid amount');
    return api.post('/api/wallet/send', { to: to.toLowerCase(), amount });
  },
};

// Utility functions
export const formatHash = (hash: string, start = 6, end = 6): string => {
  if (!hash || hash.length < start + end) return 'N/A';
  return `${hash.slice(0, start)}...${hash.slice(-end)}`;
};

export const formatTime = (ts: number): string => {
  if (!ts) return 'N/A';
  try { return new Date(ts * 1000).toLocaleString(); } catch { return 'Invalid'; }
};

// Auth API
export const authAPI = {
  signUp: (email: string, password: string, name: string) => 
    api.post('/api/auth/signup', { email, password, name }),
  
  signIn: (email: string, password: string) => 
    api.post('/api/auth/signin', { email, password }),
  
  signOut: (token: string) => 
    api.post('/api/auth/signout', {}, { 
      headers: { Authorization: `Bearer ${token}` } 
    }),
  
  getMe: (token: string) => 
    api.get('/api/auth/me', { 
      headers: { Authorization: `Bearer ${token}` } 
    }),
  
  resetPassword: (email: string) => 
    api.post('/api/auth/reset-password', { email }),
};

// Save/load token
export const saveToken = (token: string) => localStorage.setItem('token', token);
export const getToken = () => localStorage.getItem('token');
export const removeToken = () => localStorage.removeItem('token');

export default blockchainAPI;