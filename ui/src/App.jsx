import { useState, useEffect } from 'react';
import Dashboard from './components/Dashboard';
import Jobs from './components/Jobs';
import Clusters from './components/Clusters';
import Spools from './components/Spools';
import Settings from './components/Settings';

function App() {
  const [currentPage, setCurrentPage] = useState('dashboard');
  const [logs, setLogs] = useState([]);
  const [showWiki, setShowWiki] = useState(false);

  useEffect(() => {
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
      case 'dashboard': return <Dashboard logs={logs} />;
      case 'jobs': return <Jobs />;
      case 'clusters': return <Clusters />;
      case 'spools': return <Spools />;
      case 'settings': return <Settings />;
      default: return <Dashboard logs={logs} />;
    }
  };

  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'jobs', label: 'Job Queue', icon: '📋' },
    { id: 'clusters', label: 'Printers', icon: '🖨️' },
    { id: 'spools', label: 'Filament', icon: '🧵' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];

  return (
    <div className="min-h-screen bg-gray-900 text-white">
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

        <div className="absolute bottom-20 left-6 right-6">
          <button
            onClick={() => setShowWiki(true)}
            className="w-full px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg flex items-center gap-2 transition-colors"
          >
            <span className="text-xl">📖</span>
            <span>Help & Wiki</span>
          </button>
        </div>

        <div className="absolute bottom-6 left-6 right-6">
          <div className="flex items-center gap-2 text-sm">
            <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div>
            <span className="text-gray-400">Connected</span>
          </div>
        </div>
      </aside>

      <main className="ml-64 p-8">
        {renderPage()}
      </main>

      {showWiki && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4">
          <div className="bg-gray-800 rounded-xl max-w-4xl w-full max-h-[80vh] overflow-hidden border border-gray-700 flex flex-col">
            <div className="p-6 border-b border-gray-700 flex justify-between items-center">
              <h2 className="text-2xl font-bold">📖 Nexus Hub V2 - Help</h2>
              <button onClick={() => setShowWiki(false)} className="text-gray-400 hover:text-white text-2xl">✕</button>
            </div>
            <div className="p-6 overflow-y-auto flex-1">
              <WikiContent />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function WikiContent() {
  return (
    <div className="space-y-6 text-gray-300">
      <section>
        <h3 className="text-xl font-bold text-white mb-3">🔌 Connecting Slicers</h3>
        <div className="bg-gray-900 rounded-lg p-4">
          <p className="mb-3">To send prints directly from OrcaSlicer/Bambu Studio:</p>
          <ol className="list-decimal list-inside space-y-2 text-sm">
            <li>Open slicer settings → Printer Settings → General</li>
            <li>Set <strong>Host Type</strong> to: <code className="bg-gray-800 px-2 py-1 rounded">OctoPrint</code></li>
            <li>Set <strong>Hostname</strong> to: <code className="bg-gray-800 px-2 py-1 rounded">http://YOUR-IP:8080</code></li>
            <li>Leave API key blank</li>
            <li>Click <strong>Test Connection</strong></li>
          </ol>
        </div>
      </section>

      <section>
        <h3 className="text-xl font-bold text-white mb-3">🖨️ Adding Printers</h3>
        <div className="bg-gray-900 rounded-lg p-4 space-y-3 text-sm">
          <div>
            <h4 className="font-semibold text-white">Bambu Lab (LAN):</h4>
            <ul className="list-disc list-inside mt-1">
              <li>Enter IP and click "Scan" to auto-discover</li>
              <li>Enter 8-char Access Code from printer screen</li>
              <li>Enter Serial Number (from Bambu Handy app)</li>
            </ul>
          </div>
          <div>
            <h4 className="font-semibold text-white">Bambu Lab (Cloud):</h4>
            <ul className="list-disc list-inside mt-1">
              <li>Use BambuManager in Add Printer modal</li>
              <li>Login via popup and copy Access Token</li>
              <li>Discover and bind printers automatically</li>
            </ul>
          </div>
        </div>
      </section>

      <section>
        <h3 className="text-xl font-bold text-white mb-3">🧵 Filament Tracking</h3>
        <div className="bg-gray-900 rounded-lg p-4 text-sm">
          <ul className="list-disc list-inside space-y-1">
            <li>Add spools in the Filament tab</li>
            <li>Assign each spool to a printer</li>
            <li>Weight auto-deducted on print completion</li>
            <li>Red warning when below 20%</li>
          </ul>
        </div>
      </section>

      <section className="pt-4 border-t border-gray-700 text-sm text-gray-500">
        <p><strong>Nexus Hub Core V2.0</strong> - Open Source Farm Orchestration</p>
        <p className="mt-1">Built with Go + React | Moonraker & Bambu Lab Support</p>
      </section>
    </div>
  );
}

export default App;
