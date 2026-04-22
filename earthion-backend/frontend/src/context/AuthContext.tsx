import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import { authAPI, saveToken, getToken, removeToken } from '../services/api';

interface User {
  id: string;
  email: string;
  username: string;
}

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => void;
  error: string | null;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Check for existing session on load
  useEffect(() => {
    const checkAuth = async () => {
      const token = getToken();
      if (token) {
        try {
          const res = await authAPI.getMe(token);
          if (res.data.user) {
            setUser(res.data.user);
          } else {
            removeToken();
          }
        } catch {
          removeToken();
        }
      }
      setIsLoading(false);
    };
    checkAuth();
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    setError(null);
    setIsLoading(true);
    
    try {
      const res = await authAPI.signIn(email, password);
      
      if (res.data.access_token) {
        saveToken(res.data.access_token);
        setUser(res.data.user);
      } else {
        throw new Error(res.data.message || 'Login failed');
      }
    } catch (e: unknown) {
      const err = e as { response?: { data?: { detail?: string } } };
      throw new Error(err.response?.data?.detail || 'Login failed');
    } finally {
      setIsLoading(false);
    }
  }, []);

  const register = useCallback(async (email: string, password: string, name: string) => {
    setError(null);
    setIsLoading(true);
    
    try {
      const res = await authAPI.signUp(email, password, name);
      
      if (!res.data.success) {
        throw new Error(res.data.message || 'Registration failed');
      }
    } catch (e: unknown) {
      const err = e as { response?: { data?: { detail?: string } } };
      throw new Error(err.response?.data?.detail || 'Registration failed');
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(() => {
    const token = getToken();
    if (token) {
      authAPI.signOut(token).catch(() => {});
    }
    removeToken();
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ 
      user, 
      isAuthenticated: !!user, 
      isLoading, 
      login, 
      register, 
      logout,
      error 
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}