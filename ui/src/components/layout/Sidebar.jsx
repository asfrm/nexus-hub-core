import { useState } from 'react';
import WikiModal from './WikiModal';

export default function Sidebar({ currentPage, onPageChange }) {
  const [showWiki, setShowWiki] = useState(false);

  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'jobs', label: 'Job Queue', icon: '📋' },
    { id: 'clusters', label: 'Printers', icon: '🖨️' },
    { id: 'spools', label: 'Filament', icon: '🧵' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];

  return (
    <>
      <aside className="fixed left-0 top-0 h-full w-64 glass border-r border-gray-700/50">
        {/* Logo */}
        <div className="p-6 border-b border-gray-700/50">
          <h1 className="text-2xl font-bold gradient-text">Nexus Hub</h1>
          <p className="text-xs text-gray-400 mt-1">Farm Orchestration V2.0.3</p>
        </div>

        {/* Navigation */}
        <nav className="p-4 space-y-1">
          {navItems.map(item => (
            <button
              key={item.id}
              onClick={() => onPageChange(item.id)}
              className={`w-full px-4 py-3 rounded-lg flex items-center gap-3 transition-all ${
                currentPage === item.id
                  ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/30'
                  : 'text-gray-400 hover:text-white hover:bg-gray-700/50'
              }`}
            >
              <span className="text-xl">{item.icon}</span>
              <span className="font-medium">{item.label}</span>
            </button>
          ))}
        </nav>

        {/* Wiki Button */}
        <div className="absolute bottom-20 left-0 right-0 px-4">
          <button
            onClick={() => setShowWiki(true)}
            className="w-full px-4 py-3 bg-gray-700/50 hover:bg-gray-600/50 rounded-lg flex items-center gap-3 transition-colors text-gray-300 hover:text-white"
          >
            <span className="text-xl">📖</span>
            <span className="font-medium">Help & Documentation</span>
          </button>
        </div>

        {/* Connection Status */}
        <div className="absolute bottom-6 left-0 right-0 px-4">
          <div className="flex items-center gap-2 text-sm">
            <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
            <span className="text-gray-400">Connected</span>
          </div>
        </div>
      </aside>

      {/* Wiki Modal */}
      <WikiModal isOpen={showWiki} onClose={() => setShowWiki(false)} />
    </>
  );
}
