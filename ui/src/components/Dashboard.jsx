import { useState, useEffect } from 'react';
import { api } from '../services/api';

function Dashboard({ logs }) {
  const [stats, setStats] = useState({
    pendingJobs: 0,
    activeJobs: 0,
    totalClusters: 0,
    totalPrinters: 0,
    activePrinters: 0,
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStats();
    const interval = setInterval(loadStats, 10000);
    return () => clearInterval(interval);
  }, []);

  async function loadStats() {
    try {
      const [pendingJobs, activeJobs, clusters] = await Promise.all([
        api.getPendingJobs(),
        api.getActiveJobs(),
        api.getClusters(),
      ]);

      let totalPrinters = 0;
      let activePrinters = 0;
      clusters.forEach(c => {
        c.printers.forEach(p => {
          totalPrinters++;
          if (p.status === 'PRINTING') activePrinters++;
        });
      });

      setStats({
        pendingJobs: pendingJobs.length,
        activeJobs: activeJobs.length,
        totalClusters: clusters.length,
        totalPrinters,
        activePrinters,
      });
      setLoading(false);
    } catch (err) {
      console.error('Failed to load stats:', err);
    }
  }

  const statCards = [
    { label: 'Pending Jobs', value: stats.pendingJobs, color: 'yellow', icon: '📋' },
    { label: 'Active Jobs', value: stats.activeJobs, color: 'green', icon: '▶️' },
    { label: 'Clusters', value: stats.totalClusters, color: 'blue', icon: '🖨️' },
    { label: 'Active Printers', value: `${stats.activePrinters}/${stats.totalPrinters}`, color: 'purple', icon: '⚡' },
  ];

  const colorClasses = {
    yellow: 'bg-yellow-500/20 border-yellow-500/50 text-yellow-400',
    green: 'bg-green-500/20 border-green-500/50 text-green-400',
    blue: 'bg-blue-500/20 border-blue-500/50 text-blue-400',
    purple: 'bg-purple-500/20 border-purple-500/50 text-purple-400',
  };

  return (
    <div className="space-y-6">
      <h2 className="text-3xl font-bold">Dashboard</h2>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {statCards.map(card => (
          <div
            key={card.label}
            className={`p-6 rounded-xl border ${colorClasses[card.color]} transition-transform hover:scale-105`}
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm opacity-70">{card.label}</p>
                <p className="text-3xl font-bold mt-1">{card.value}</p>
              </div>
              <span className="text-4xl">{card.icon}</span>
            </div>
          </div>
        ))}
      </div>

      {/* System Logs */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 overflow-hidden">
        <div className="p-4 border-b border-gray-700">
          <h3 className="text-lg font-semibold">System Logs</h3>
        </div>
        <div className="max-h-96 overflow-y-auto">
          {logs.length === 0 ? (
            <p className="p-4 text-gray-400">No logs yet</p>
          ) : (
            <table className="w-full">
              <thead className="bg-gray-900 text-xs uppercase text-gray-400">
                <tr>
                  <th className="px-4 py-2 text-left">Time</th>
                  <th className="px-4 py-2 text-left">Level</th>
                  <th className="px-4 py-2 text-left">Module</th>
                  <th className="px-4 py-2 text-left">Message</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log, idx) => (
                  <tr key={idx} className="border-t border-gray-700 hover:bg-gray-700/50">
                    <td className="px-4 py-2 text-sm text-gray-400">
                      {new Date(log.timestamp).toLocaleTimeString()}
                    </td>
                    <td className="px-4 py-2">
                      <span className={`px-2 py-1 rounded text-xs font-medium ${
                        log.level === 'ERROR' ? 'bg-red-500/20 text-red-400' :
                        log.level === 'INFO' ? 'bg-blue-500/20 text-blue-400' :
                        'bg-gray-500/20 text-gray-400'
                      }`}>
                        {log.level}
                      </span>
                    </td>
                    <td className="px-4 py-2 text-sm text-gray-300">{log.module}</td>
                    <td className="px-4 py-2 text-sm text-gray-300">{log.message}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}

export default Dashboard;
