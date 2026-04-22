import { useBlockchain, useOffline } from '../hooks/useBlockchain';
import BlockList from '../components/BlockList';
import WalletCard from '../components/WalletCard';
import MiningControls from '../components/MiningControls';
import SendTransaction from '../components/SendTransaction';
import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function DashboardPage() {
  const { stats, loading, connection, refresh } = useBlockchain();
  const isOffline = useOffline();
  const { user, logout } = useAuth();

  if (loading && !connection.isOnline) {
    return (
      <div className="min-h-screen hero-gradient flex items-center justify-center">
        <div className="text-center">
          <svg className="animate-spin w-12 h-12 mx-auto mb-4 text-earthion" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <p className="text-gray-600">Connecting to blockchain...</p>
        </div>
      </div>
    );
  }

  if (!connection.isOnline) {
    return (
      <div className="min-h-screen hero-gradient flex items-center justify-center">
        <div className="card-hover max-w-md text-center">
          <svg className="w-12 h-12 mx-auto mb-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 5.636a9 9 0 010 12.728m0 0l-2.829-2.829m2.829 2.829L21 21M15.536 8.464a5 5 0 010 7.072m0 0l-2.829-2.829m-4.243 2.829a4 4 0 01-5.523-5.523m7.072 7.072L10.343 21" />
          </svg>
          <h1 className="text-xl font-bold text-gray-900 mb-2">Connection Lost</h1>
          <p className="text-gray-600 mb-4">{connection.lastError}</p>
          <button onClick={refresh} className="btn btn-primary">
            Try Again
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen hero-gradient">
      {/* Header */}
      <header className="header sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            {/* Logo */}
            <div className="flex items-center">
              <Link to="/" className="flex items-center hover:opacity-80 transition-opacity">
                <img 
                  src="/earth-logo-BCK.png" 
                  alt="Earthion" 
                  className="w-20 h-20 rounded-lg object-contain"
                />
                <span className="text-2xl tracking-wider" style={{ fontFamily: 'Anton, sans-serif', letterSpacing: '0.1em', textShadow: '0 2px 10px rgba(12, 201, 48, 0.3)' }}>
                  <span className="text-earthion">EARTH</span><span className="text-gray-900">ion</span>
                </span>
              </Link>
            </div>

            {/* Stats & Status */}
            <div className="flex items-center space-x-4">
              {/* Height */}
              <div className="hidden sm:flex items-center space-x-2 px-3 py-1.5 bg-bone-100 rounded-lg">
                <svg className="w-4 h-4 text-earthion" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v4M5 11h14" />
                </svg>
                <span className="text-sm text-gray-600">
                  <span className="font-mono font-medium text-gray-900">{stats?.height || 0}</span>
                </span>
              </div>
              
              {/* Status */}
              <div 
                className="flex items-center space-x-2 px-3 py-1.5 rounded-lg"
                style={{ 
                  background: connection.isReconnecting ? 'rgba(234, 179, 8, 0.1)' : 'rgba(12, 201, 48, 0.1)',
                  border: `1px solid ${connection.isReconnecting ? 'rgba(234, 179, 8, 0.3)' : 'rgba(12, 201, 48, 0.2)'}`
                }}
              >
                <span 
                  className={`w-2 h-2 rounded-full ${connection.isReconnecting ? 'bg-yellow-500' : 'bg-earthion'}`}
                  style={!connection.isReconnecting ? { animation: 'blink 1s infinite' } : undefined}
                />
                <span className={`text-sm font-medium ${connection.isReconnecting ? 'text-yellow-600' : 'text-earthion'}`}>
                  {connection.isReconnecting ? 'Syncing' : 'Live'}
                </span>
              </div>

              <div className="h-6 w-px bg-bone-300"></div>
              
              {/* User Name - Visible */}
              <div className="flex items-center space-x-2">
                <div className="w-8 h-8 bg-earthion rounded-full flex items-center justify-center">
                  <span className="text-white text-xs font-bold">{user?.username?.charAt(0).toUpperCase()}</span>
                </div>
                <span className="text-sm font-medium text-gray-700">{user?.username}</span>
              </div>
              
              <Link
                to="/"
                onClick={logout}
                className="p-2 text-gray-500 hover:text-earthion transition-colors"
                title="Logout"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                </svg>
              </Link>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* Left Sidebar */}
          <div className="lg:col-span-4 space-y-6">
            <WalletCard />
            <MiningControls />
          </div>

          {/* Right Content */}
          <div className="lg:col-span-8 space-y-6">
            <BlockList limit={10} />
            <SendTransaction />
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="footer mt-auto">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex justify-between items-center text-sm text-bone-500">
            <div className="flex items-center space-x-2">
              <svg className="w-4 h-4 text-earthion" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2" />
              </svg>
              <span>Earthion Blockchain</span>
            </div>
            <div className="flex items-center space-x-4">
              <span>v1.0.0</span>
              <span>•</span>
              <span>UTXO Model</span>
              <span>•</span>
              <span>PoW Mining</span>
            </div>
          </div>
        </div>
      </footer>

      {/* Offline Banner */}
      {isOffline && (
        <div className="fixed bottom-4 right-4 bg-yellow-50 border border-yellow-300 text-yellow-800 px-4 py-2 rounded-lg shadow-lg flex items-center space-x-2">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 5.636a9 9 0 010 12.728m0 0l-2.829-2.829m2.829 2.829L21 21M15.536 8.464a5 5 0 010 7.072m0 0l-2.829-2.829m-4.243 2.829a4 4 0 01-5.523-5.523m7.072 7.072L10.343 21" />
          </svg>
          <span className="text-sm">Reconnecting...</span>
        </div>
      )}
    </div>
  );
}