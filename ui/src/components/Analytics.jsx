import { useState, useEffect } from 'react';
import { api } from '../services/api';

function Analytics() {
  const [analytics, setAnalytics] = useState(null);
  const [wearStats, setWearStats] = useState([]);
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 30000); // Refresh every 30s
    return () => clearInterval(interval);
  }, []);

  async function loadData() {
    try {
      const [analyticsData, wearData, historyData] = await Promise.all([
        api.getAnalytics(),
        api.getWearStats(),
        api.getHistory(20),
      ]);
      setAnalytics(analyticsData);
      setWearStats(wearData);
      setHistory(historyData);
      setLoading(false);
    } catch (err) {
      console.error('Failed to load analytics:', err);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-400">Loading analytics...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h2 className="text-3xl font-bold">Analytics & History</h2>

      {/* System Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-400">Total Prints</p>
              <p className="text-3xl font-bold mt-1">{analytics?.total_prints || 0}</p>
            </div>
            <span className="text-4xl">📊</span>
          </div>
        </div>

        <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-400">Total Print Time</p>
              <p className="text-3xl font-bold mt-1">
                {analytics?.total_print_time_hours?.toFixed(1) || 0}h
              </p>
            </div>
            <span className="text-4xl">⏱️</span>
          </div>
        </div>

        <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-400">Success Rate</p>
              <p className="text-3xl font-bold mt-1">
                {analytics?.success_rate?.toFixed(1) || 0}%
              </p>
            </div>
            <span className="text-4xl">✅</span>
          </div>
        </div>

        <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-400">Avg Print Time</p>
              <p className="text-3xl font-bold mt-1">
                {analytics?.avg_print_time_minutes?.toFixed(0) || 0}m
              </p>
            </div>
            <span className="text-4xl">⏰</span>
          </div>
        </div>
      </div>

      {/* Wear Level Stats */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 overflow-hidden">
        <div className="p-4 border-b border-gray-700">
          <h3 className="text-lg font-semibold">🔧 Printer Wear Levels</h3>
          <p className="text-sm text-gray-400 mt-1">
            Lower wear score = less used = preferred for new jobs
          </p>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-900 text-xs uppercase text-gray-400">
              <tr>
                <th className="px-4 py-3 text-left">Printer</th>
                <th className="px-4 py-3 text-right">Total Time</th>
                <th className="px-4 py-3 text-right">Total Prints</th>
                <th className="px-4 py-3 text-right">Success Rate</th>
                <th className="px-4 py-3 text-right">Avg Print</th>
                <th className="px-4 py-3 text-right">Wear Score</th>
                <th className="px-4 py-3 text-left">Status</th>
              </tr>
            </thead>
            <tbody>
              {wearStats.map(stat => (
                <tr key={stat.printer_id} className="border-t border-gray-700 hover:bg-gray-700/50">
                  <td className="px-4 py-3">
                    <div>
                      <p className="font-medium">{stat.printer_name}</p>
                      <p className="text-xs text-gray-400">ID: {stat.printer_id}</p>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right text-gray-300">
                    {(stat.total_print_time_s / 3600).toFixed(1)}h
                  </td>
                  <td className="px-4 py-3 text-right text-gray-300">
                    {stat.total_prints}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${
                      stat.success_rate >= 90 ? 'bg-green-500/20 text-green-400' :
                      stat.success_rate >= 70 ? 'bg-yellow-500/20 text-yellow-400' :
                      'bg-red-500/20 text-red-400'
                    }`}>
                      {stat.success_rate.toFixed(1)}%
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right text-gray-300">
                    {(stat.avg_print_time_s / 60).toFixed(0)}m
                  </td>
                  <td className="px-4 py-3 text-right">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${
                      stat.wear_score < 10 ? 'bg-green-500/20 text-green-400' :
                      stat.wear_score < 50 ? 'bg-yellow-500/20 text-yellow-400' :
                      'bg-red-500/20 text-red-400'
                    }`}>
                      {stat.wear_score.toFixed(1)}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <div className={`w-2 h-2 rounded-full ${
                        stat.wear_score < 10 ? 'bg-green-500' :
                        stat.wear_score < 50 ? 'bg-yellow-500' :
                        'bg-red-500'
                      }`} />
                      <span className="text-sm text-gray-300">
                        {stat.wear_score < 10 ? 'Low' : stat.wear_score < 50 ? 'Medium' : 'High'}
                      </span>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Print History */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 overflow-hidden">
        <div className="p-4 border-b border-gray-700">
          <h3 className="text-lg font-semibold">📜 Recent Print History</h3>
        </div>
        <div className="max-h-96 overflow-y-auto">
          <table className="w-full">
            <thead className="bg-gray-900 text-xs uppercase text-gray-400 sticky top-0">
              <tr>
                <th className="px-4 py-3 text-left">File</th>
                <th className="px-4 py-3 text-left">Printer</th>
                <th className="px-4 py-3 text-right">Duration</th>
                <th className="px-4 py-3 text-left">Status</th>
                <th className="px-4 py-3 text-left">Completed</th>
              </tr>
            </thead>
            <tbody>
              {history.map(job => (
                <tr key={job.id} className="border-t border-gray-700 hover:bg-gray-700/50">
                  <td className="px-4 py-3">
                    <div>
                      <p className="font-medium text-sm">{job.filename}</p>
                      <p className="text-xs text-gray-400">{job.filament_type}</p>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-300">
                    {job.printer_name}
                  </td>
                  <td className="px-4 py-3 text-right text-sm text-gray-300">
                    {(job.actual_time_s / 60).toFixed(0)}m
                  </td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${
                      job.status === 'COMPLETED' ? 'bg-green-500/20 text-green-400' :
                      job.status === 'FAILED' ? 'bg-red-500/20 text-red-400' :
                      'bg-gray-500/20 text-gray-400'
                    }`}>
                      {job.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-400">
                    {new Date(job.completed_at).toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

export default Analytics;
