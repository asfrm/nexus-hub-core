import { useState, useEffect } from 'react';
import { api } from '../services/api';

function Jobs() {
  const [pendingJobs, setPendingJobs] = useState([]);
  const [activeJobs, setActiveJobs] = useState([]);
  const [clusters, setClusters] = useState([]);
  const [uploading, setUploading] = useState(false);
  const [showAssignModal, setShowAssignModal] = useState(false);
  const [selectedJob, setSelectedJob] = useState(null);
  const [selectedCluster, setSelectedCluster] = useState('');
  const [selectedPrinter, setSelectedPrinter] = useState('');

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
  }, []);

  async function loadData() {
    try {
      const [pending, active, clusterData] = await Promise.all([
        api.getPendingJobs(),
        api.getActiveJobs(),
        api.getClusters(),
      ]);
      setPendingJobs(pending);
      setActiveJobs(active);
      setClusters(clusterData);
    } catch (err) {
      console.error('Failed to load jobs:', err);
    }
  }

  async function handleFileUpload(e) {
    const file = e.target.files[0];
    if (!file) return;

    setUploading(true);
    try {
      await api.uploadGCode(file);
      loadData();
    } catch (err) {
      alert('Upload failed: ' + err.message);
    } finally {
      setUploading(false);
      e.target.value = '';
    }
  }

  function openAssignModal(job) {
    setSelectedJob(job);
    setShowAssignModal(true);
  }

  async function handleAssign() {
    if (!selectedJob || !selectedCluster || !selectedPrinter) return;

    try {
      await api.assignJob(selectedJob.id, parseInt(selectedCluster), parseInt(selectedPrinter));
      setShowAssignModal(false);
      setSelectedJob(null);
      setSelectedCluster('');
      setSelectedPrinter('');
      loadData();
    } catch (err) {
      alert('Assignment failed: ' + err.message);
    }
  }

  function getClusterForModel(model) {
    return clusters.find(c => c.target_model === model);
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-3xl font-bold">Job Queue</h2>
        <label className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg cursor-pointer transition-colors">
          <input type="file" accept=".gcode,.g" onChange={handleFileUpload} className="hidden" />
          {uploading ? 'Uploading...' : '📁 Upload G-Code'}
        </label>
      </div>

      {/* Pending Jobs */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 overflow-hidden">
        <div className="p-4 border-b border-gray-700 bg-yellow-500/10">
          <h3 className="text-lg font-semibold text-yellow-400">⏳ Pending Jobs ({pendingJobs.length})</h3>
        </div>
        {pendingJobs.length === 0 ? (
          <p className="p-4 text-gray-400">No pending jobs</p>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 p-4">
            {pendingJobs.map(job => (
              <div key={job.id} className="bg-gray-900 rounded-lg border border-gray-700 overflow-hidden">
                {job.thumbnail_url ? (
                  <div className="h-32 bg-gray-800 overflow-hidden">
                    <img
                      src={job.thumbnail_url}
                      alt={job.filename}
                      className="w-full h-full object-cover"
                      onError={(e) => { e.target.style.display = 'none'; }}
                    />
                  </div>
                ) : (
                  <div className="h-32 bg-gray-800 flex items-center justify-center text-gray-600">
                    📄 No Preview
                  </div>
                )}
                <div className="p-3">
                  <h4 className="font-medium text-sm truncate" title={job.filename}>{job.filename}</h4>
                  <div className="mt-2 flex flex-wrap gap-2 text-xs">
                    <span className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded">
                      ⏱️ {formatTime(job.estimated_time_s)}
                    </span>
                    {job.filament_weight_g > 0 && (
                      <span className="px-2 py-1 bg-green-500/20 text-green-400 rounded">
                        🧵 {job.filament_weight_g.toFixed(1)}g
                      </span>
                    )}
                    <span className="px-2 py-1 bg-purple-500/20 text-purple-400 rounded">
                      {job.filament_type || 'Unknown'}
                    </span>
                  </div>
                  <div className="mt-3 flex justify-between items-center">
                    <span className={`px-2 py-1 rounded text-xs ${getClusterForModel(job.printer_model)
                        ? 'bg-green-500/20 text-green-400'
                        : 'bg-red-500/20 text-red-400'
                      }`}>
                      {job.printer_model}
                    </span>
                    <button
                      onClick={() => openAssignModal(job)}
                      className="px-3 py-1 bg-blue-600 hover:bg-blue-700 rounded text-xs transition-colors"
                    >
                      Assign
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Active Jobs */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 overflow-hidden">
        <div className="p-4 border-b border-gray-700 bg-green-500/10">
          <h3 className="text-lg font-semibold text-green-400">▶️ Active Jobs ({activeJobs.length})</h3>
        </div>
        {activeJobs.length === 0 ? (
          <p className="p-4 text-gray-400">No active jobs</p>
        ) : (
          <table className="w-full">
            <thead className="bg-gray-900 text-xs uppercase text-gray-400">
              <tr>
                <th className="px-4 py-2 text-left">Filename</th>
                <th className="px-4 py-2 text-left">Status</th>
                <th className="px-4 py-2 text-left">Printer</th>
                <th className="px-4 py-2 text-left">Created</th>
              </tr>
            </thead>
            <tbody>
              {activeJobs.map(job => (
                <tr key={job.id} className="border-t border-gray-700 hover:bg-gray-700/50">
                  <td className="px-4 py-3 text-sm font-medium">{job.filename}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${job.status === 'PRINTING' ? 'bg-green-500/20 text-green-400' : 'bg-yellow-500/20 text-yellow-400'
                      }`}>
                      {job.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-300">
                    Printer #{job.target_printer_id}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-400">
                    {new Date(job.created_at).toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Assignment Modal */}
      {showAssignModal && selectedJob && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-gray-800 rounded-xl p-6 w-full max-w-md border border-gray-700">
            <h3 className="text-xl font-bold mb-4">Assign Job</h3>
            <p className="text-gray-400 mb-4">{selectedJob.filename}</p>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-gray-400 mb-1">Cluster</label>
                <select
                  value={selectedCluster}
                  onChange={(e) => {
                    setSelectedCluster(e.target.value);
                    setSelectedPrinter('');
                  }}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                >
                  <option value="">Select cluster...</option>
                  {clusters.filter(c => c.target_model === selectedJob.printer_model).map(c => (
                    <option key={c.id} value={c.id}>{c.name} ({c.printers.length} printers)</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Printer</label>
                <select
                  value={selectedPrinter}
                  onChange={(e) => setSelectedPrinter(e.target.value)}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                  disabled={!selectedCluster}
                >
                  <option value="">Select printer...</option>
                  {clusters
                    .find(c => c.id === parseInt(selectedCluster))?.printers
                    .filter(p => p.status === 'IDLE')
                    .map(p => (
                      <option key={p.id} value={p.id}>{p.name} ({p.ip_address})</option>
                    ))}
                </select>
              </div>
            </div>

            <div className="flex gap-3 mt-6">
              <button
                onClick={handleAssign}
                disabled={!selectedCluster || !selectedPrinter}
                className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed rounded transition-colors"
              >
                Assign & Start
              </button>
              <button
                onClick={() => {
                  setShowAssignModal(false);
                  setSelectedJob(null);
                  setSelectedCluster('');
                  setSelectedPrinter('');
                }}
                className="px-4 py-2 bg-gray-600 hover:bg-gray-700 rounded transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function formatTime(seconds) {
  if (!seconds) return 'Unknown';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}

export default Jobs;
