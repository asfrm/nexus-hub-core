import { useState, useEffect } from 'react';
import Dashboard from './components/Dashboard';
import Jobs from './components/Jobs';
import Clusters from './components/Clusters';
import Analytics from './components/Analytics';
import Settings from './components/Settings';

function App() {
  const [currentPage, setCurrentPage] = useState('dashboard');
  const [logs, setLogs] = useState([]);

  useEffect(() => {
    // Initialize WebSocket connection
    const cleanup = window.wsCleanup || (() => { });
    cleanup();

    const unsubscribe = window.wsService.subscribe((packet) => {
      if (packet.event_type === 'SYSTEM_LOG' && packet.log) {
        setLogs(prev => [packet.log, ...prev].slice(0, 50));
      }
    });

    window.wsCleanup = unsubscribe;

    return () => unsubscribe();
  }, []);

  const renderPage = () => {
    switch (currentPage) {
      case 'dashboard':
        return <Dashboard logs={logs} />;
      case 'jobs':
        return <Jobs />;
      case 'clusters':
        return <Clusters />;
      case 'analytics':
        return <Analytics />;
      case 'settings':
        return <Settings />;
      default:
        return <Dashboard logs={logs} />;
    }
  };

  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'jobs', label: 'Job Queue', icon: '📋' },
    { id: 'clusters', label: 'Printers', icon: '🖨️' },
    { id: 'analytics', label: 'Analytics', icon: '📈' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      {/* Sidebar */}
      <aside className="fixed left-0 top-0 h-full w-64 bg-gray-800 border-r border-gray-700">
        <div className="p-6">
          <h1 className="text-2xl font-bold text-blue-400">Nexus Hub</h1>
          <p className="text-xs text-gray-400 mt-1">3D Farm Orchestration</p>
        </div>

        <nav className="mt-6">
          {navItems.map(item => (
            <button
              key={item.id}
              onClick={() => setCurrentPage(item.id)}
              className={`w-full px-6 py-3 text-left flex items-center gap-3 transition-colors ${currentPage === item.id
                  ? 'bg-blue-600 text-white'
                  : 'text-gray-300 hover:bg-gray-700'
                }`}
            >
              <span className="text-xl">{item.icon}</span>
              <span>{item.label}</span>
            </button>
          ))}
        </nav>

        {/* Connection Status */}
        <div className="absolute bottom-6 left-6 right-6">
          <div className="flex items-center gap-2 text-sm">
            <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div>
            <span className="text-gray-400">Connected</span>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="ml-64 p-8">
        {renderPage()}
      </main>
    </div>
  );
}

export default App;
