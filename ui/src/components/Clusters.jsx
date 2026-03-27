import { useState, useEffect } from 'react';
import { api } from '../services/api';
import BambuManager from './BambuManager';

function Clusters() {
  const [clusters, setClusters] = useState([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showAddPrinterModal, setShowAddPrinterModal] = useState(false);
  const [selectedCluster, setSelectedCluster] = useState(null);
  const [newClusterName, setNewClusterName] = useState('');
  const [newClusterModel, setNewClusterModel] = useState('');
  const [newPrinterName, setNewPrinterName] = useState('');
  const [newPrinterIP, setNewPrinterIP] = useState('');
  const [printerSerial, setPrinterSerial] = useState('');
  const [accessCode, setAccessCode] = useState('');
  const [printerType, setPrinterType] = useState('MOONRAKER');
  const [unassignedModels, setUnassignedModels] = useState([]);
  const [scanning, setScanning] = useState(false);
  const [discoveredPrinters, setDiscoveredPrinters] = useState([]);

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 10000);
    return () => clearInterval(interval);
  }, []);

  async function loadData() {
    try {
      const [clusterData, unassigned] = await Promise.all([
        api.getClusters(),
        api.getUnassignedModels(),
      ]);
      setClusters(clusterData);
      setUnassignedModels(unassigned);
    } catch (err) {
      console.error('Failed to load clusters:', err);
    }
  }

  async function handleScanNetwork() {
    setScanning(true);
    try {
      const result = await api.scanNetwork();
      if (result.printers && result.printers.length > 0) {
        setDiscoveredPrinters(result.printers);
        // Auto-fill first discovered printer
        const first = result.printers[0];
        setNewPrinterIP(first.ip_address || '');
        setPrinterSerial(first.serial || '');
      }
    } catch (err) {
      alert('Network scan failed: ' + err.message);
    } finally {
      setScanning(false);
    }
  }

  async function handleCreateCluster() {
    if (!newClusterName || !newClusterModel) return;
    try {
      await api.createCluster(newClusterName, newClusterModel);
      setNewClusterName('');
      setNewClusterModel('');
      setShowCreateModal(false);
      loadData();
    } catch (err) {
      alert('Failed to create cluster: ' + err.message);
    }
  }

  async function handleDeleteCluster(id) {
    if (!confirm('Delete this cluster and all its printers?')) return;
    try {
      await api.deleteCluster(id);
      loadData();
    } catch (err) {
      alert('Failed to delete cluster: ' + err.message);
    }
  }

  async function handleToggleCluster(id, enabled) {
    try {
      await api.toggleCluster(id, !enabled);
      loadData();
    } catch (err) {
      alert('Failed to toggle cluster: ' + err.message);
    }
  }

  async function handlePrinterControl(id, action) {
    try {
      await api.controlPrinter(id, action);
    } catch (err) {
      alert('Failed to control printer: ' + err.message);
    }
  }

  const statusColors = {
    IDLE: 'bg-gray-500/20 text-gray-400',
    PRINTING: 'bg-green-500/20 text-green-400',
    ERROR: 'bg-red-500/20 text-red-400',
    PAUSED: 'bg-yellow-500/20 text-yellow-400',
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-3xl font-bold">Clusters & Printers</h2>
        <button
          onClick={() => setShowCreateModal(true)}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
        >
          + Create Cluster
        </button>
      </div>

      {/* Unassigned Models Alert */}
      {unassignedModels.length > 0 && (
        <div className="bg-yellow-500/10 border border-yellow-500/50 rounded-xl p-4">
          <h3 className="text-yellow-400 font-semibold mb-2">⚠️ Unassigned Printer Models</h3>
          <p className="text-gray-400 text-sm mb-3">
            The following printer models have pending jobs but no clusters assigned:
          </p>
          <div className="flex flex-wrap gap-2">
            {unassignedModels.map(model => (
              <span key={model} className="px-3 py-1 bg-yellow-500/20 text-yellow-400 rounded-full text-sm">
                {model}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Clusters Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {clusters.map(cluster => (
          <div key={cluster.id} className="bg-gray-800 rounded-xl border border-gray-700 overflow-hidden">
            <div className="p-4 border-b border-gray-700 flex justify-between items-center">
              <div>
                <h3 className="text-lg font-semibold">{cluster.name}</h3>
                <p className="text-sm text-gray-400">Target: {cluster.target_model}</p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleToggleCluster(cluster.id, cluster.isEnabled)}
                  className={`px-3 py-1 rounded text-sm transition-colors ${cluster.isEnabled
                    ? 'bg-green-500/20 text-green-400 hover:bg-green-500/30'
                    : 'bg-gray-500/20 text-gray-400 hover:bg-gray-500/30'
                    }`}
                >
                  {cluster.isEnabled ? '● Enabled' : '○ Disabled'}
                </button>
                <button
                  onClick={() => handleDeleteCluster(cluster.id)}
                  className="p-2 text-red-400 hover:bg-red-500/20 rounded transition-colors"
                  title="Delete cluster"
                >
                  🗑️
                </button>
              </div>
            </div>

            {/* Printers List */}
            <div className="p-4">
              <div className="flex justify-between items-center mb-3">
                <h4 className="text-sm font-medium text-gray-400">
                  Printers ({cluster.printers.length})
                </h4>
                <button
                  onClick={() => {
                    setSelectedCluster(cluster);
                    setShowAddPrinterModal(true);
                  }}
                  className="text-sm text-blue-400 hover:text-blue-300"
                >
                  + Add Printer
                </button>
              </div>

              {cluster.printers.length === 0 ? (
                <p className="text-gray-500 text-sm">No printers in this cluster</p>
              ) : (
                <div className="space-y-2">
                  {cluster.printers.map(printer => (
                    <div
                      key={printer.id}
                      className="flex items-center justify-between p-3 bg-gray-900 rounded-lg"
                    >
                      <div>
                        <p className="font-medium">{printer.name}</p>
                        <p className="text-sm text-gray-400">{printer.ip_address}</p>
                        {printer.printer_type && (
                          <span className="text-xs text-gray-500">{printer.printer_type}</span>
                        )}
                      </div>
                      <div className="flex items-center gap-3">
                        <span className={`px-2 py-1 rounded text-xs font-medium ${statusColors[printer.status] || statusColors.IDLE}`}>
                          {printer.status}
                        </span>
                        <div className="flex gap-1">
                          <button
                            onClick={() => handlePrinterControl(printer.id, 'pause')}
                            className="p-1.5 text-yellow-400 hover:bg-yellow-500/20 rounded transition-colors"
                            title="Pause"
                          >
                            ⏸️
                          </button>
                          <button
                            onClick={() => handlePrinterControl(printer.id, 'resume')}
                            className="p-1.5 text-green-400 hover:bg-green-500/20 rounded transition-colors"
                            title="Resume"
                          >
                            ▶️
                          </button>
                          <button
                            onClick={() => handlePrinterControl(printer.id, 'stop')}
                            className="p-1.5 text-red-400 hover:bg-red-500/20 rounded transition-colors"
                            title="Stop"
                          >
                            ⏹️
                          </button>
                          <button
                            onClick={() => handlePrinterControl(printer.id, 'estop')}
                            className="p-1.5 text-orange-400 hover:bg-orange-500/20 rounded transition-colors"
                            title="Emergency Stop"
                          >
                            🚨
                          </button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ))}
      </div>

      {clusters.length === 0 && (
        <div className="text-center py-12 text-gray-400">
          <p className="text-xl mb-2">No clusters yet</p>
          <p>Create a cluster to start managing your printers</p>
        </div>
      )}

      {/* Create Cluster Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-gray-800 rounded-xl p-6 w-full max-w-md border border-gray-700">
            <h3 className="text-xl font-bold mb-4">Create Cluster</h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-gray-400 mb-1">Cluster Name</label>
                <input
                  type="text"
                  value={newClusterName}
                  onChange={(e) => setNewClusterName(e.target.value)}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                  placeholder="e.g., Production Line A"
                />
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Target Printer Model</label>
                <input
                  type="text"
                  value={newClusterModel}
                  onChange={(e) => setNewClusterModel(e.target.value)}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                  placeholder="e.g., Voron 2.4"
                />
                {unassignedModels.length > 0 && (
                  <div className="mt-2">
                    <p className="text-xs text-gray-500 mb-1">Available from pending jobs:</p>
                    <div className="flex flex-wrap gap-1">
                      {unassignedModels.map(model => (
                        <button
                          key={model}
                          onClick={() => setNewClusterModel(model)}
                          className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs hover:bg-blue-500/30"
                        >
                          {model}
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>

            <div className="flex gap-3 mt-6">
              <button
                onClick={handleCreateCluster}
                className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded transition-colors"
              >
                Create
              </button>
              <button
                onClick={() => setShowCreateModal(false)}
                className="px-4 py-2 bg-gray-600 hover:bg-gray-700 rounded transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Printer Modal */}
      {showAddPrinterModal && selectedCluster && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-gray-800 rounded-xl p-6 w-full max-w-lg border border-gray-700">
            <h3 className="text-xl font-bold mb-4">Add Printer</h3>
            <p className="text-gray-400 mb-4">to {selectedCluster.name}</p>

            <div className="mb-4">
              <label className="block text-sm text-gray-400 mb-1">Printer Type</label>
              <select
                value={printerType}
                onChange={(e) => setPrinterType(e.target.value)}
                className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
              >
                <option value="MOONRAKER">Moonraker / Klipper</option>
                <option value="BAMBULAB">Bambu Lab (Cloud/Local)</option>
              </select>
            </div>

            {printerType === 'BAMBULAB' ? (
              <BambuManager
                clusters={clusters}
                onPrinterAdded={() => {
                  setShowAddPrinterModal(false);
                  setSelectedCluster(null);
                  loadData();
                }}
              />
            ) : (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm text-gray-400 mb-1">Printer Name</label>
                  <input
                    type="text"
                    value={newPrinterName}
                    onChange={(e) => setNewPrinterName(e.target.value)}
                    className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                    placeholder="e.g., Printer 1"
                  />
                </div>

                <div>
                  <label className="block text-sm text-gray-400 mb-1">IP Address / Hostname</label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={newPrinterIP}
                      onChange={(e) => setNewPrinterIP(e.target.value)}
                      className="flex-1 bg-gray-900 border border-gray-600 rounded px-3 py-2"
                      placeholder="e.g., 192.168.1.100"
                    />
                    <button
                      onClick={handleScanNetwork}
                      disabled={scanning}
                      className="px-3 py-2 bg-purple-600 hover:bg-purple-700 disabled:bg-gray-600 rounded transition-colors text-sm"
                      title="Scan network for printers"
                    >
                      {scanning ? '🔍' : '🔍 Scan'}
                    </button>
                  </div>
                  {discoveredPrinters.length > 0 && (
                    <div className="mt-2 text-xs text-green-400">
                      Found: {discoveredPrinters.length} printer(s) - Auto-filled
                    </div>
                  )}
                </div>

                <div>
                  <label className="block text-sm text-gray-400 mb-1">Serial Number</label>
                  <input
                    type="text"
                    value={printerSerial}
                    onChange={(e) => setPrinterSerial(e.target.value.toUpperCase())}
                    className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                    placeholder="e.g., ABC123456"
                  />
                </div>

                <div>
                  <label className="block text-sm text-gray-400 mb-1">Access Code (8 chars)</label>
                  <input
                    type="text"
                    value={accessCode}
                    onChange={(e) => setAccessCode(e.target.value.toUpperCase())}
                    className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2 tracking-widest"
                    placeholder="ABCDEFGH"
                    maxLength="8"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    For Bambu Lab: Settings → About → Access Code
                  </p>
                </div>

                <div className="flex gap-3">
                  <button
                    onClick={async () => {
                      if (!newPrinterName || !newPrinterIP) {
                        alert('Please fill in all fields');
                        return;
                      }
                      try {
                        await api.addPrinter(
                          selectedCluster.id,
                          newPrinterName,
                          newPrinterIP,
                          printerType,
                          printerSerial,
                          accessCode
                        );
                        setNewPrinterName('');
                        setNewPrinterIP('');
                        setPrinterSerial('');
                        setAccessCode('');
                        setShowAddPrinterModal(false);
                        setSelectedCluster(null);
                        setDiscoveredPrinters([]);
                        loadData();
                      } catch (err) {
                        alert('Failed to add printer: ' + err.message);
                      }
                    }}
                    className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded transition-colors"
                  >
                    Add Printer
                  </button>
                  <button
                    onClick={() => {
                      setShowAddPrinterModal(false);
                      setSelectedCluster(null);
                      setNewPrinterName('');
                      setNewPrinterIP('');
                      setPrinterSerial('');
                      setAccessCode('');
                      setDiscoveredPrinters([]);
                    }}
                    className="px-4 py-2 bg-gray-600 hover:bg-gray-700 rounded transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}

            {printerType !== 'BAMBULAB' && (
              <div className="flex gap-3 mt-6">
                <button
                  onClick={() => {
                    setShowAddPrinterModal(false);
                    setSelectedCluster(null);
                  }}
                  className="flex-1 px-4 py-2 bg-gray-600 hover:bg-gray-700 rounded transition-colors"
                >
                  Close
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export default Clusters;
