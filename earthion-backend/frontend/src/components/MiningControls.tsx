import { useBlockchain, useMining } from '../hooks/useBlockchain';

export default function MiningControls() {
  const { stats, loading } = useBlockchain();
  const { mining, error, mine } = useMining();

  if (loading) {
    return (
      <div className="card-hover animate-pulse">
        <div className="h-28 bg-bone-200 rounded-lg"></div>
      </div>
    );
  }

  return (
    <div className="card-hover">
      {/* Header */}
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold text-gray-900 flex items-center space-x-2">
          <svg className="w-5 h-5 text-yellow-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
          <span>Mining</span>
        </h2>
        <span 
          className={`text-xs px-2 py-1 rounded-full flex items-center space-x-1 ${
            mining 
              ? 'bg-yellow-100 text-yellow-700 border border-yellow-200' 
              : 'bg-bone-100 text-gray-500 border border-bone-200'
          }`}
        >
          <span 
            className={`w-1.5 h-1.5 rounded-full ${mining ? 'bg-yellow-500 animate-pulse' : 'bg-gray-400'}`}
          ></span>
          <span>{mining ? 'Mining...' : 'Idle'}</span>
        </span>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-2 gap-3 mb-4">
        {/* Height */}
        <div 
          className="rounded-xl p-3 bg-bone-100"
        >
          <p className="text-xs text-gray-500 flex items-center space-x-1">
            <svg className="w-3 h-3 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2" />
            </svg>
            <span>Height</span>
          </p>
          <p className="text-xl font-bold text-gray-900">#{stats?.height || 0}</p>
        </div>

        {/* Difficulty */}
        <div 
          className="rounded-xl p-3 bg-bone-100"
        >
          <p className="text-xs text-gray-500 flex items-center space-x-1">
            <svg className="w-3 h-3 text-yellow-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
            <span>Difficulty</span>
          </p>
          <p className="text-xl font-bold text-gray-900">{stats?.difficulty || 0}</p>
        </div>

        {/* Reward */}
        <div 
          className="rounded-xl p-3 bg-bone-100"
        >
          <p className="text-xs text-gray-500 flex items-center space-x-1">
            <svg className="w-3 h-3 text-earthion" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v1m0 11a3 3 0 100-6 3 3 0 000 6z" />
            </svg>
            <span>Reward</span>
          </p>
          <p className="text-xl font-bold text-earthion">{stats?.currentReward || 0}</p>
        </div>

        {/* Supply */}
        <div 
          className="rounded-xl p-3 bg-bone-100"
        >
          <p className="text-xs text-gray-500 flex items-center space-x-1">
            <svg className="w-3 h-3 text-purple-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
            <span>Supply</span>
          </p>
          <p className="text-xl font-bold text-gray-900">{(stats?.totalMined || 0).toLocaleString()}</p>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="bg-red-50 text-red-600 px-4 py-2 rounded-lg text-sm mb-4 flex items-center space-x-2">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <span>{error}</span>
        </div>
      )}

      {/* Mine Button */}
      <button
        onClick={mine}
        disabled={mining}
        className="w-full py-3.5 rounded-xl font-semibold transition-all flex items-center justify-center space-x-2"
        style={{ 
          background: mining 
            ? 'linear-gradient(135deg, #a16207, #ca8a04)'
            : 'linear-gradient(135deg, #0cc930, #00a825)',
          boxShadow: mining ? 'none' : '0 4px 15px rgba(12, 201, 48, 0.3)',
          opacity: mining ? 0.8 : 1
        }}
      >
        {mining ? (
          <>
            <svg className="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span className="text-white">Mining...</span>
          </>
        ) : (
          <>
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
            <span>Mine Block</span>
          </>
        )}
      </button>

      {/* Progress */}
      {mining && (
        <div className="mt-3 rounded-full h-1.5 overflow-hidden bg-bone-200">
          <div 
            className="h-full rounded-full bg-gradient-to-r from-earthion to-yellow-500"
            style={{ animation: 'loading 2s ease-in-out infinite' }}
          />
        </div>
      )}
    </div>
  );
}