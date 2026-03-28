import { Card, CardHeader, CardContent, Button, Badge, Input, Select, Modal, useToast, TemperatureGauge, ProgressBar } from '../ui';
import { api } from '../../services/api';
import { useState, useEffect } from 'react';

export default function Clusters({ livePrinters }) {
  const [clusters, setClusters] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showAddPrinterModal, setShowAddPrinterModal] = useState(false);
  const [selectedCluster, setSelectedCluster] = useState(null);
  const toast = useToast();

  const [newCluster, setNewCluster] = useState({ name: '', targetModel: '' });
  const [newPrinter, setNewPrinter] = useState({
    name: '',
    ip: '',
    serial: '',
    accessCode: '',
    type: 'BAMBULAB',
  });
  const [scanning, setScanning] = useState(false);

  useEffect(() => {
    loadClusters();
    const interval = setInterval(loadClusters, 10000);
    return () => clearInterval(interval);
  }, [livePrinters]);

  async function loadClusters() {
    try {
      const data = await api.getClusters();
      const clustersWithLive = data.map(cluster => ({
        ...cluster,
        printers: cluster.printers.map(printer => ({
          ...printer,
          ...(livePrinters[printer.id] || {}),
        })),
      }));
      setClusters(clustersWithLive);
      setLoading(false);
    } catch (err) {
      toast.error('Failed to load clusters: ' + err.message);
      setLoading(false);
    }
  }

  async function handleCreateCluster() {
    if (!newCluster.name || !newCluster.targetModel) {
      toast.warning('Please fill in all fields');
      return;
    }
    try {
      await api.createCluster(newCluster.name, newCluster.targetModel);
      toast.success('Cluster created successfully');
      setNewCluster({ name: '', targetModel: '' });
      setShowCreateModal(false);
      loadClusters();
    } catch (err) {
      toast.error('Failed to create cluster: ' + err.message);
    }
  }

  async function handleDeleteCluster(id) {
    if (!confirm('Delete this cluster and all its printers?')) return;
    try {
      await api.deleteCluster(id);
      toast.success('Cluster deleted');
      loadClusters();
    } catch (err) {
      toast.error('Failed to delete cluster: ' + err.message);
    }
  }

  async function handleToggleCluster(id, enabled) {
    try {
      await api.toggleCluster(id, !enabled);
      toast.success(`Cluster ${!enabled ? 'enabled' : 'disabled'}`);
      loadClusters();
    } catch (err) {
      toast.error('Failed to toggle cluster: ' + err.message);
    }
  }

  async function handlePrinterControl(id, action) {
    try {
      await api.controlPrinter(id, action);
      toast.success(`Command "${action}" sent`);
    } catch (err) {
      toast.error('Failed to control printer: ' + err.message);
    }
  }

  async function handleScanNetwork() {
    setScanning(true);
    try {
      const result = await api.scanNetwork();
      if (result.printers && result.printers.length > 0) {
        const first = result.printers[0];
        setNewPrinter(prev => ({
          ...prev,
          ip: first.ip_address || '',
          serial: first.serial || '',
        }));
        toast.success(`Found ${result.printers.length} printer(s)`);
      } else {
        toast.info('No printers found on network');
      }
    } catch (err) {
      toast.error('Network scan failed: ' + err.message);
    } finally {
      setScanning(false);
    }
  }

  async function handleAddPrinter() {
    if (!newPrinter.name || !newPrinter.ip) {
      toast.warning('Please enter printer name and IP');
      return;
    }
    if (newPrinter.type === 'BAMBULAB') {
      if (!newPrinter.serial) {
        toast.warning('Serial number required for Bambu Lab');
        return;
      }
      if (!newPrinter.accessCode || newPrinter.accessCode.length !== 8) {
        toast.warning('Access code must be 8 characters');
        return;
      }
    }
    try {
      await api.addPrinter(
        selectedCluster.id,
        newPrinter.name,
        newPrinter.ip,
        newPrinter.type,
        newPrinter.type === 'BAMBULAB' ? newPrinter.serial : '',
        newPrinter.type === 'BAMBULAB' ? newPrinter.accessCode : ''
      );
      toast.success('Printer added successfully');
      setNewPrinter({ name: '', ip: '', serial: '', accessCode: '', type: 'BAMBULAB' });
      setShowAddPrinterModal(false);
      setSelectedCluster(null);
      loadClusters();
    } catch (err) {
      toast.error('Failed to add printer: ' + err.message);
    }
  }

  const statusColors = {
    IDLE: 'gray',
    PRINTING: 'green',
    PAUSED: 'yellow',
    ERROR: 'red',
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-3xl font-bold text-white">Printers & Clusters</h2>
          <Button onClick={() => setShowCreateModal(true)}>+ Create Cluster</Button>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {[1, 2].map(i => (
            <Card key={i}><div className="h-48 animate-pulse bg-gray-700 rounded-xl" /></Card>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold text-white">Printers & Clusters</h2>
        <Button onClick={() => setShowCreateModal(true)}>+ Create Cluster</Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {clusters.map(cluster => (
          <Card key={cluster.id} className="card-hover">
            <CardHeader
              title={cluster.name}
              subtitle={`Target: ${cluster.target_model}`}
              action={
                <div className="flex items-center gap-2">
                  <Badge variant={cluster.is_enabled ? 'green' : 'gray'}>
                    <span className={`w-1.5 h-1.5 rounded-full ${cluster.is_enabled ? 'bg-green-500' : 'bg-gray-500'}`} />
                    {cluster.is_enabled ? 'Enabled' : 'Disabled'}
                  </Badge>
                  <button onClick={() => handleToggleCluster(cluster.id, cluster.is_enabled)} className="p-1.5 text-gray-400 hover:text-white transition-colors" title={cluster.is_enabled ? 'Disable' : 'Enable'}>⚡</button>
                  <button onClick={() => handleDeleteCluster(cluster.id)} className="p-1.5 text-red-400 hover:text-red-300 transition-colors" title="Delete cluster">🗑️</button>
                </div>
              }
            />
            <CardContent>
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-sm font-medium text-gray-400">Printers ({cluster.printers.length})</h4>
                <button onClick={() => { setSelectedCluster(cluster); setShowAddPrinterModal(true); }} className="text-sm text-blue-400 hover:text-blue-300 transition-colors font-medium">+ Add Printer</button>
              </div>

              {cluster.printers.length === 0 ? (
                <p className="text-gray-500 text-sm py-4 text-center">No printers in this cluster</p>
              ) : (
                <div className="space-y-2">
                  {cluster.printers.map(printer => (
                    <div key={printer.id} className="glass rounded-lg p-3">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="font-medium text-white">{printer.name}</span>
                            <Badge variant={statusColors[printer.status] || 'gray'}>
                              <span className={`w-1.5 h-1.5 rounded-full ${printer.status === 'PRINTING' ? 'bg-green-500 animate-pulse' :
                                printer.status === 'PAUSED' ? 'bg-yellow-500' :
                                  printer.status === 'ERROR' ? 'bg-red-500 animate-pulse' :
                                    'bg-gray-500'
                                }`} />
                              {printer.status}
                            </Badge>
                          </div>
                          <p className="text-sm text-gray-400">{printer.ip_address}</p>
                          {printer.printer_type === 'BAMBULAB' && printer.serial && (
                            <p className="text-xs text-gray-500 mt-0.5">S/N: {printer.serial}</p>
                          )}

                          {printer.printer_type === 'BAMBULAB' && printer.status === 'PRINTING' && (
                            <div className="mt-2 space-y-1">
                              <TemperatureGauge
                                current={printer.nozzle_temper || 0}
                                target={printer.nozzle_target_temper || 0}
                                label="Nozzle"
                                icon="🌡️"
                              />
                              <TemperatureGauge
                                current={printer.bed_temper || 0}
                                target={printer.bed_target_temper || 0}
                                label="Bed"
                                icon="♨️"
                              />
                              {printer.progress > 0 && <ProgressBar value={printer.progress} className="mt-2" />}
                              {printer.progress > 0 && <p className="text-xs text-gray-400 text-center">{printer.progress.toFixed(1)}% complete</p>}
                            </div>
                          )}
                        </div>

                        <div className="flex gap-1 ml-2">
                          {printer.printer_type === 'BAMBULAB' ? (
                            <>
                              <button onClick={() => handlePrinterControl(printer.id, 'pause')} className="p-2 text-yellow-400 hover:bg-yellow-500/20 rounded transition-colors" title="Pause" disabled={printer.status !== 'PRINTING'}>⏸️</button>
                              <button onClick={() => handlePrinterControl(printer.id, 'resume')} className="p-2 text-green-400 hover:bg-green-500/20 rounded transition-colors" title="Resume" disabled={printer.status !== 'PAUSED'}>▶️</button>
                              <button onClick={() => handlePrinterControl(printer.id, 'stop')} className="p-2 text-red-400 hover:bg-red-500/20 rounded transition-colors" title="Stop" disabled={printer.status !== 'PRINTING' && printer.status !== 'PAUSED'}>⏹️</button>
                              <button onClick={() => handlePrinterControl(printer.id, 'g28')} className="p-2 text-blue-400 hover:bg-blue-500/20 rounded transition-colors disabled:opacity-30 disabled:cursor-not-allowed" title="Home All (G28)" disabled={printer.status === 'PRINTING' || printer.status === 'PAUSED'}>🏠</button>
                            </>
                          ) : (
                            <>
                              <button onClick={() => handlePrinterControl(printer.id, 'pause')} className="p-2 text-yellow-400 hover:bg-yellow-500/20 rounded transition-colors" title="Pause">⏸️</button>
                              <button onClick={() => handlePrinterControl(printer.id, 'resume')} className="p-2 text-green-400 hover:bg-green-500/20 rounded transition-colors" title="Resume">▶️</button>
                              <button onClick={() => handlePrinterControl(printer.id, 'stop')} className="p-2 text-red-400 hover:bg-red-500/20 rounded transition-colors" title="Stop">⏹️</button>
                            </>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {clusters.length === 0 && (
        <Card>
          <div className="text-center py-12">
            <div className="text-6xl mb-4">🖨️</div>
            <h3 className="text-lg font-medium text-white mb-2">No clusters yet</h3>
            <p className="text-gray-400 mb-6">Create a cluster to start managing your printers</p>
            <Button onClick={() => setShowCreateModal(true)}>+ Create Cluster</Button>
          </div>
        </Card>
      )}

      <Modal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} title="Create Cluster">
        <div className="space-y-4">
          <Input label="Cluster Name" value={newCluster.name} onChange={(e) => setNewCluster(prev => ({ ...prev, name: e.target.value }))} placeholder="e.g., Production Line A" />
          <Input label="Target Printer Model" value={newCluster.targetModel} onChange={(e) => setNewCluster(prev => ({ ...prev, targetModel: e.target.value }))} placeholder="e.g., BambuLab X1C" />
          <div className="flex gap-3 pt-4">
            <Button onClick={handleCreateCluster} className="flex-1">Create</Button>
            <Button variant="secondary" onClick={() => setShowCreateModal(false)}>Cancel</Button>
          </div>
        </div>
      </Modal>

      <Modal isOpen={showAddPrinterModal} onClose={() => { setShowAddPrinterModal(false); setSelectedCluster(null); setNewPrinter({ name: '', ip: '', serial: '', accessCode: '', type: 'BAMBULAB' }); }} title={selectedCluster ? `Add Printer to ${selectedCluster.name}` : 'Add Printer'}>
        <div className="space-y-4">
          <Select label="Printer Type" value={newPrinter.type} onChange={(e) => setNewPrinter(prev => ({ ...prev, type: e.target.value }))}>
            <option value="BAMBULAB">Bambu Lab (LAN)</option>
            <option value="MOONRAKER">Moonraker / Klipper</option>
          </Select>
          <Input label="Printer Name" value={newPrinter.name} onChange={(e) => setNewPrinter(prev => ({ ...prev, name: e.target.value }))} placeholder="e.g., Printer 1" />
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">IP Address</label>
            <div className="flex gap-2">
              <input type="text" className="w-full bg-gray-800 border border-gray-600 rounded-lg px-4 py-2.5 text-white focus:outline-none focus:ring-2 focus:ring-blue-500" value={newPrinter.ip} onChange={(e) => setNewPrinter(prev => ({ ...prev, ip: e.target.value }))} placeholder="e.g., 192.168.1.100" />
              <Button variant="secondary" size="sm" onClick={handleScanNetwork} disabled={scanning}>{scanning ? '⏳' : '🔍'}</Button>
            </div>
          </div>
          {newPrinter.type === 'BAMBULAB' && (
            <>
              <Input label="Serial Number *" value={newPrinter.serial} onChange={(e) => setNewPrinter(prev => ({ ...prev, serial: e.target.value.toUpperCase() }))} placeholder="e.g., 0300DA5C0900350" helperText="Found in Bambu Handy app" />
              <Input label="Access Code (8 chars) *" value={newPrinter.accessCode} onChange={(e) => setNewPrinter(prev => ({ ...prev, accessCode: e.target.value.toUpperCase() }))} placeholder="ABCDEFGH" maxLength={8} helperText="On printer: Settings → About → Access Code" />
            </>
          )}
          <div className="flex gap-3 pt-4">
            <Button onClick={handleAddPrinter} className="flex-1">Add Printer</Button>
            <Button variant="secondary" onClick={() => { setShowAddPrinterModal(false); setSelectedCluster(null); setNewPrinter({ name: '', ip: '', serial: '', accessCode: '', type: 'BAMBULAB' }); }}>Cancel</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
