import { Card, CardHeader, CardContent, Button, Badge, Modal, useToast } from '../ui';
import { useState, useEffect } from 'react';

export default function Dashboard() {
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [showHealthModal, setShowHealthModal] = useState(false);
  const toast = useToast();

  useEffect(() => {
    loadStats();
    const interval = setInterval(loadStats, 30000);
    return () => clearInterval(interval);
  }, []);

  async function loadStats() {
    try {
      const response = await fetch('/api/health');
      const data = await response.json();
      setStats(data);
      setLoading(false);
    } catch (err) {
      toast.error('Failed to load system stats: ' + err.message);
      setLoading(false);
    }
  }

  function navigateTo(page) {
    // Dispatch custom event for navigation
    window.dispatchEvent(new CustomEvent('navigate', { detail: { page } }));
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <h2 className="text-3xl font-bold text-white">Dashboard</h2>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          {[...Array(4)].map((_, i) => (
            <Card key={i}><div className="h-24 animate-pulse bg-gray-700 rounded-xl" /></Card>
          ))}
        </div>
      </div>
    );
  }

  const statCards = [
    { label: 'System Status', value: stats?.status === 'healthy' ? 'Online' : 'Issues', color: stats?.status === 'healthy' ? 'green' : 'red', icon: stats?.status === 'healthy' ? '✅' : '⚠️', subtext: `Uptime: ${stats?.uptime || 'N/A'}` },
    { label: 'Connected Printers', value: stats?.printers || 0, color: 'blue', icon: '🖨️', subtext: 'Total devices' },
    { label: 'Database', value: stats?.database === 'connected' ? 'Connected' : 'Disconnected', color: stats?.database === 'connected' ? 'green' : 'red', icon: stats?.database === 'connected' ? '💾' : '❌', subtext: 'SQLite' },
    { label: 'Version', value: stats?.version || 'Unknown', color: 'purple', icon: '📦', subtext: 'Latest' },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold text-white">Dashboard</h2>
        <div className="flex gap-2">
          <Badge variant="blue">v{stats?.version}</Badge>
          <Button variant="secondary" size="sm" onClick={() => setShowHealthModal(true)}>📊 View Health</Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {statCards.map(card => (
          <Card key={card.label} className="card-hover">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-400">{card.label}</p>
                <p className="text-2xl font-bold text-white mt-1">{card.value}</p>
                <p className="text-xs text-gray-500 mt-1">{card.subtext}</p>
              </div>
              <span className="text-4xl">{card.icon}</span>
            </div>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader title="⚡ Quick Actions" subtitle="Common tasks at your fingertips" />
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <QuickAction icon="📁" title="Job Queue" description="Manage print jobs" onClick={() => navigateTo('jobs')} />
            <QuickAction icon="🖨️" title="Printers" description="Manage clusters" onClick={() => navigateTo('clusters')} />
            <QuickAction icon="🧵" title="Filament" description="Track spools" onClick={() => navigateTo('spools')} />
            <QuickAction icon="📊" title="System Health" description="View diagnostics" onClick={() => setShowHealthModal(true)} />
          </div>
        </CardContent>
      </Card>

      {/* Health Modal */}
      <Modal isOpen={showHealthModal} onClose={() => setShowHealthModal(false)} title="📊 System Health" size="md">
        {stats && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <HealthItem label="Status" value={stats.status} icon={stats.status === 'healthy' ? '✅' : '⚠️'} />
              <HealthItem label="Version" value={stats.version} icon="📦" />
              <HealthItem label="Uptime" value={stats.uptime} icon="⏱️" />
              <HealthItem label="Printers" value={stats.printers} icon="🖨️" />
              <HealthItem label="Database" value={stats.database} icon={stats.database === 'connected' ? '✅' : '❌'} />
              <HealthItem label="Timestamp" value={new Date(stats.timestamp).toLocaleString()} icon="🕐" />
            </div>
            <div className="pt-4 border-t border-gray-700">
              <Button onClick={() => { loadStats(); toast.success('Health check refreshed'); }} className="w-full">🔄 Refresh</Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}

function QuickAction({ icon, title, description, onClick }) {
  return (
    <div onClick={onClick} className="glass rounded-xl p-4 card-hover cursor-pointer group">
      <div className="text-3xl mb-3 group-hover:scale-110 transition-transform">{icon}</div>
      <h4 className="font-medium text-white group-hover:text-blue-400 transition-colors">{title}</h4>
      <p className="text-sm text-gray-400 mt-1">{description}</p>
    </div>
  );
}

function HealthItem({ label, value, icon }) {
  return (
    <div className="glass rounded-lg p-3 flex items-center gap-3">
      <span className="text-2xl">{icon}</span>
      <div>
        <p className="text-xs text-gray-400">{label}</p>
        <p className="text-sm font-medium text-white">{value}</p>
      </div>
    </div>
  );
}
