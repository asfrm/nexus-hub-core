import { Card, CardHeader, CardContent, Button, Input, Select, Badge, Modal, ProgressBar, useToast } from '../ui';
import { api } from '../../services/api';
import { useState, useEffect } from 'react';

export default function Spools() {
  const [spools, setSpools] = useState([]);
  const [clusters, setClusters] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const toast = useToast();

  const [newSpool, setNewSpool] = useState({
    material: 'PLA',
    color: '',
    colorHex: '#FF0000',
    totalWeight: 1000,
    assignedTo: '',
  });

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 30000);
    return () => clearInterval(interval);
  }, []);

  async function loadData() {
    try {
      const [spoolsData, clustersData] = await Promise.all([api.getSpools(), api.getClusters()]);
      setSpools(spoolsData);
      setClusters(clustersData);
      setLoading(false);
    } catch (err) {
      toast.error('Failed to load data: ' + err.message);
      setLoading(false);
    }
  }

  async function handleAddSpool() {
    if (!newSpool.color || !newSpool.assignedTo) {
      toast.warning('Please fill in all required fields');
      return;
    }
    try {
      await api.addSpool(
        newSpool.material,
        newSpool.color,
        newSpool.colorHex,
        parseFloat(newSpool.totalWeight),
        parseInt(newSpool.assignedTo)
      );
      toast.success('Spool added successfully');
      setNewSpool({ material: 'PLA', color: '', colorHex: '#FF0000', totalWeight: 1000, assignedTo: '' });
      setShowAddModal(false);
      loadData();
    } catch (err) {
      toast.error('Failed to add spool: ' + err.message);
    }
  }

  function getPrinterName(printerId) {
    for (const cluster of clusters) {
      const printer = cluster.printers.find(p => p.id === printerId);
      if (printer) return `${cluster.name} - ${printer.name}`;
    }
    return 'Unassigned';
  }

  function getRemainingPercent(spool) {
    if (spool.total_weight_g <= 0) return 0;
    return ((spool.remaining_weight_g / spool.total_weight_g) * 100).toFixed(1);
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-3xl font-bold text-white">Filament Inventory</h2>
          <Button onClick={() => setShowAddModal(true)}>+ Add Spool</Button>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map(i => <Card key={i}><div className="h-48 animate-pulse bg-gray-700 rounded-xl" /></Card>)}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold text-white">Filament Inventory</h2>
        <Button onClick={() => setShowAddModal(true)}>+ Add Spool</Button>
      </div>

      {spools.length === 0 ? (
        <Card>
          <div className="text-center py-12">
            <div className="text-6xl mb-4">🧵</div>
            <h3 className="text-lg font-medium text-white mb-2">No spools registered</h3>
            <p className="text-gray-400 mb-6">Add your first filament spool to track usage</p>
            <Button onClick={() => setShowAddModal(true)}>+ Add Spool</Button>
          </div>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {spools.map(spool => {
            const remainingPercent = getRemainingPercent(spool);
            const isLow = remainingPercent < 20;
            return (
              <Card key={spool.id} className="card-hover">
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-full border-4 border-white shadow-lg" style={{ backgroundColor: spool.color_hex }} />
                    <div>
                      <h3 className="font-bold text-white">{spool.color}</h3>
                      <Badge variant="blue">{spool.material}</Badge>
                    </div>
                  </div>
                  <Badge variant={isLow ? 'red' : 'green'}>{isLow ? '⚠️ Low' : '✓ OK'}</Badge>
                </div>

                <div className="space-y-3">
                  <div>
                    <div className="flex justify-between text-xs mb-1">
                      <span className="text-gray-400">Remaining</span>
                      <span className="text-white">{spool.remaining_weight_g.toFixed(0)}g / {spool.total_weight_g.toFixed(0)}g</span>
                    </div>
                    <ProgressBar value={remainingPercent} max={100} />
                    <p className="text-xs text-gray-400 text-center mt-1">{remainingPercent}% remaining</p>
                  </div>

                  <div className="pt-3 border-t border-gray-700">
                    <p className="text-xs text-gray-400 mb-1">Assigned to:</p>
                    <p className="text-sm font-medium text-white">{spool.assigned_printer_id ? getPrinterName(spool.assigned_printer_id) : 'None'}</p>
                  </div>
                </div>
              </Card>
            );
          })}
        </div>
      )}

      <Modal isOpen={showAddModal} onClose={() => setShowAddModal(false)} title="🧵 Add Filament Spool" size="md">
        <div className="space-y-4">
          <Select label="Material" value={newSpool.material} onChange={(e) => setNewSpool(prev => ({ ...prev, material: e.target.value }))}>
            <option value="PLA">PLA</option>
            <option value="PETG">PETG</option>
            <option value="ABS">ABS</option>
            <option value="ASA">ASA</option>
            <option value="TPU">TPU</option>
            <option value="PC">Polycarbonate</option>
            <option value="Nylon">Nylon</option>
            <option value="PVA">PVA (Support)</option>
          </Select>

          <Input label="Color Name" value={newSpool.color} onChange={(e) => setNewSpool(prev => ({ ...prev, color: e.target.value }))} placeholder="e.g., Silk Red" />

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">Color</label>
            <div className="flex gap-2">
              <input type="color" className="w-12 h-10 rounded cursor-pointer bg-transparent" value={newSpool.colorHex} onChange={(e) => setNewSpool(prev => ({ ...prev, colorHex: e.target.value }))} />
              <input type="text" className="flex-1 bg-gray-800 border border-gray-600 rounded-lg px-4 py-2.5 text-white" value={newSpool.colorHex} onChange={(e) => setNewSpool(prev => ({ ...prev, colorHex: e.target.value }))} placeholder="#RRGGBB" />
            </div>
          </div>

          <Input label="Total Weight (grams)" type="number" value={newSpool.totalWeight} onChange={(e) => setNewSpool(prev => ({ ...prev, totalWeight: e.target.value }))} placeholder="1000" />

          <Select label="Assign to Printer" value={newSpool.assignedTo} onChange={(e) => setNewSpool(prev => ({ ...prev, assignedTo: e.target.value }))}>
            <option value="">Select printer...</option>
            {clusters.flatMap(c => c.printers.map(p => (
              <option key={p.id} value={p.id}>{c.name} - {p.name}</option>
            )))}
          </Select>

          <div className="flex gap-3 pt-4">
            <Button onClick={handleAddSpool} className="flex-1">Add Spool</Button>
            <Button variant="secondary" onClick={() => setShowAddModal(false)}>Cancel</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
