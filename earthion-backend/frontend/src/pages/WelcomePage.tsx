import { useBlockchain } from '../hooks/useBlockchain';
import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function WelcomePage() {
  const { stats, connection } = useBlockchain();
  const { user, logout, isAuthenticated } = useAuth();

  return (
    <div className="min-h-screen hero-gradient">
      {/* Hero Section */}
      <div className="relative overflow-hidden">
        {/* Background Pattern */}
        <div className="absolute inset-0 pattern-dots"></div>

        {/* Navbar - Cream background */}
        <header className="header sticky top-0 z-50">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between items-center h-16">
              {/* Logo */}
              <div className="flex items-center">
                <img 
                  src="/earth-logo-BCK.png" 
                  alt="Earthion" 
                  className="w-20 h-20 rounded-lg object-contain"
                />
                <span className="text-4xl tracking-wider" style={{ fontFamily: 'Anton, sans-serif', letterSpacing: '0.1em', textShadow: '0 2px 10px rgba(12, 201, 48, 0.3)' }}>
                  <span className="text-earthion">EARTH</span><span className="text-gray-900">ion</span>
                </span>
              </div>

              {/* Auth Buttons */}
              <div className="flex items-center space-x-4">
                {isAuthenticated ? (
                  <div className="flex items-center space-x-4">
                    <span className="text-sm text-gray-600">
                      Welcome, <span className="font-medium text-gray-900">{user?.username}</span>
                    </span>
                    <Link
                      to="/dashboard"
                      className="btn btn-primary"
                    >
                      Go to Dashboard
                    </Link>
                    <button
                      onClick={logout}
                      className="text-sm text-gray-500 hover:text-gray-700"
                    >
                      Logout
                    </button>
                  </div>
                ) : (
                  <Link
                    to="/auth"
                    className="btn btn-primary"
                  >
                    Login / Register
                  </Link>
                )}
              </div>
            </div>
          </div>
        </header>

        {/* Hero Content */}
        <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20 text-center">
          <h1 className="text-5xl md:text-6xl font-bold text-gray-900 mb-6">
            The Future of{' '}
            <span className="text-earthion">Decentralized</span> Finance
          </h1>
          <p className="text-xl text-gray-600 max-w-2xl mx-auto mb-10">
            Earthion is a next-generation blockchain platform built for scalability, 
            security, and sustainable crypto mining.
          </p>
          
          {/* CTA Buttons */}
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            {!isAuthenticated && (
              <Link to="/auth" className="btn btn-primary px-8 py-3 text-lg shadow-md">
                Get Started
              </Link>
            )}
            {isAuthenticated && (
              <Link to="/dashboard" className="btn btn-primary px-8 py-3 text-lg shadow-md">
                Open Dashboard
              </Link>
            )}
            <a href="#features" className="btn btn-secondary px-8 py-3 text-lg">
              Learn More
            </a>
          </div>
        </div>
      </div>

      {/* Stats Section - Cream background */}
      <div className="bg-cream-100 border-y border-bone-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
            <div className="text-center">
              <p className="text-3xl font-bold text-earthion">{stats?.height || 0}</p>
              <p className="text-sm text-gray-500 mt-1">Block Height</p>
            </div>
            <div className="text-center">
              <p className="text-3xl font-bold text-earthion">{stats?.difficulty || 0}</p>
              <p className="text-sm text-gray-500 mt-1">Difficulty</p>
            </div>
            <div className="text-center">
              <p className="text-3xl font-bold text-earthion">{(stats?.totalMined || 0).toLocaleString()}</p>
              <p className="text-sm text-gray-500 mt-1">Total ion Mined</p>
            </div>
            <div className="text-center">
              <p className="text-3xl font-bold text-earthion">
                {connection.isOnline ? 'Online' : 'Offline'}
              </p>
              <p className="text-sm text-gray-500 mt-1">Network Status</p>
            </div>
          </div>
        </div>
      </div>

      {/* Features Section */}
      <div className="py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl font-bold text-gray-900 mb-4">Why Choose Earthion?</h2>
            <p className="text-lg text-gray-600 max-w-2xl mx-auto">
              Built with cutting-edge technology to deliver the best blockchain experience
            </p>
          </div>

          <div className="grid md:grid-cols-3 gap-8">
            {/* Feature 1 */}
            <div className="card-hover">
              <div className="w-12 h-12 bg-[#0cc930] rounded-lg flex items-center justify-center mb-4">
                <span className="text-2xl">⛏️</span>
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Proof of Work Mining</h3>
              <p className="text-gray-600">
                Traditional proof-of-work consensus with sustainable mining rewards. 
                Earn ion tokens by contributing to network security.
              </p>
            </div>

            {/* Feature 2 */}
            <div className="card-hover">
              <div className="w-12 h-12 bg-[#0cc930] rounded-lg flex items-center justify-center mb-4">
                <span className="text-2xl">🔒</span>
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Secure Transactions</h3>
              <p className="text-gray-600">
                Cryptographically secured transactions with ECDSA signatures. 
                Your assets are protected by military-grade encryption.
              </p>
            </div>

            {/* Feature 3 */}
            <div className="card-hover">
              <div className="w-12 h-12 bg-[#0cc930] rounded-lg flex items-center justify-center mb-4">
                <span className="text-2xl">⚡</span>
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Fast & Scalable</h3>
              <p className="text-gray-600">
                Optimized for speed with efficient block propagation. 
                Handle thousands of transactions with minimal fees.
              </p>
            </div>

            {/* Feature 4 */}
            <div className="card-hover">
              <div className="w-12 h-12 bg-[#0cc930] rounded-lg flex items-center justify-center mb-4">
                <span className="text-2xl">💳</span>
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Wallet Management</h3>
              <p className="text-gray-600">
                Full wallet functionality including address generation, 
                balance tracking, and transaction sending.
              </p>
            </div>

            {/* Feature 5 */}
            <div className="card-hover">
              <div className="w-12 h-12 bg-[#0cc930] rounded-lg flex items-center justify-center mb-4">
                <span className="text-2xl">🌐</span>
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Decentralized</h3>
              <p className="text-gray-600">
                No central authority. The network is maintained by nodes worldwide, 
                ensuring true decentralization and censorship resistance.
              </p>
            </div>

            {/* Feature 6 */}
            <div className="card-hover">
              <div className="w-12 h-12 bg-[#0cc930] rounded-lg flex items-center justify-center mb-4">
                <span className="text-2xl">📊</span>
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Real-time Stats</h3>
              <p className="text-gray-600">
                Monitor blockchain health with live statistics including 
                block height, difficulty, and network hashrate.
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* How It Works - Cream background */}
      <div className="bg-cream-50 py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl font-bold text-gray-900 mb-4">How It Works</h2>
            <p className="text-lg text-gray-600 max-w-2xl mx-auto">
              Get started with Earthion in three simple steps
            </p>
          </div>

          <div className="grid md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="w-16 h-16 bg-earthion text-white rounded-full flex items-center justify-center text-2xl font-bold mx-auto mb-4 shadow-md">
                1
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Create Account</h3>
              <p className="text-gray-600">
                Register with your username and email to get started. 
                It's free and takes less than a minute.
              </p>
            </div>

            <div className="text-center">
              <div className="w-16 h-16 bg-earthion text-white rounded-full flex items-center justify-center text-2xl font-bold mx-auto mb-4 shadow-md">
                2
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Access Dashboard</h3>
              <p className="text-gray-600">
                Once logged in, access your personal dashboard with 
                wallet, mining controls, and blockchain explorer.
              </p>
            </div>

            <div className="text-center">
              <div className="w-16 h-16 bg-earthion text-white rounded-full flex items-center justify-center text-2xl font-bold mx-auto mb-4 shadow-md">
                3
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Start Mining</h3>
              <p className="text-gray-600">
                Click the mine button to start earning ion tokens. 
                Rewards are added directly to your wallet.
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Footer - Bone background */}
      <footer className="footer py-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row justify-between items-center">
            <div className="flex items-center space-x-2 mb-4 md:mb-0">
              <div className="w-8 h-8 bg-earthion rounded-full flex items-center justify-center">
                <span className="text-white font-bold">E</span>
              </div>
              <span className="text-lg font-bold text-white">Earthion</span>
            </div>
            <div className="text-sm">
              © 2026 Earthion Blockchain. All rights reserved.
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}