import { useState } from 'react';
import { useBlockchain } from '../hooks/useBlockchain';
import { formatHash, formatTime } from '../services/api';

interface Props { limit?: number; }
export default function BlockList({ limit = 20 }: Props) {
  const { blocks, loading } = useBlockchain();
  const [showAll, setShowAll] = useState(false);

  // Sort: newest (highest index) first
  const sortedBlocks = [...blocks].sort((a, b) => b.index - a.index);
  const displayBlocks = showAll ? sortedBlocks : sortedBlocks.slice(0, limit);

  if (loading) {
    return (
      <div className="card-hover animate-pulse">
        <div className="space-y-3">
          {[1,2,3,4,5].map(i => (
            <div key={i} className="h-14 bg-bone-200 rounded-lg"></div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="card-hover">
      {/* Header */}
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold text-gray-900 flex items-center space-x-2">
          <svg className="w-5 h-5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2" />
          </svg>
          <span>Recent Blocks</span>
        </h2>
        <div className="flex items-center space-x-2">
          <span className="text-xs text-gray-500">{blocks.length} blocks</span>
          {blocks.length > limit && (
            <button 
              onClick={() => setShowAll(!showAll)}
              className="text-xs px-2 py-1 rounded bg-bone-100 text-gray-600 hover:bg-bone-200 transition-colors"
            >
              {showAll ? 'Show Less' : `View All (${blocks.length})`}
            </button>
          )}
        </div>
      </div>
      
      {/* Empty State */}
      {blocks.length === 0 ? (
        <div className="text-center py-8">
          <svg className="w-12 h-12 mx-auto text-gray-300 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2" />
          </svg>
          <p className="text-gray-500">No blocks yet. Start mining!</p>
        </div>
      ) : (
        /* Blocks List */
        <div className="space-y-2">
          {displayBlocks.map((b, i) => (
            <div 
              key={b.hash} 
              className={`flex justify-between items-center p-3 rounded-lg transition-colors ${
                i === 0 ? 'bg-earthion/10 border border-earthion/20' : 'bg-bone-100 hover:bg-bone-200'
              }`}
            >
              {/* Left: Index + Hash + Time */}
              <div className="flex items-center space-x-3">
                <div 
                  className={`flex items-center justify-center w-10 h-10 rounded-lg text-sm font-mono ${
                    i === 0 ? 'bg-earthion text-white' : 'bg-bone-200 text-gray-600'
                  }`}
                >
                  #{b.index}
                </div>
                <div>
                  <p className="font-mono text-sm text-gray-900">{formatHash(b.hash, 8, 6)}</p>
                  <p className="text-xs text-gray-500 flex items-center space-x-1">
                    <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span>{formatTime(b.timestamp)}</span>
                  </p>
                </div>
              </div>
              
              {/* Right: Stats */}
              <div className="flex items-center space-x-4 text-sm text-gray-600">
                <div className="flex items-center space-x-1">
                  <svg className="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m4 0h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span>{b.txCount} txns</span>
                </div>
                <div className="flex items-center space-x-1">
                  <svg className="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                  <span>{b.difficulty}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}