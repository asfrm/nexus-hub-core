import { useState, useEffect } from 'react';
import { api } from '../services/api';

function Spools() {
  const [spools, setSpools] = useState([]);
  const [showAddModal, setShowAddModal] = useState(false);
  const [clusters, setClusters] = useState([]);
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
      const [spoolsData, clustersData] = await Promise.all([
        api.getSpools(),
        api.getClusters(),
      ]);
      setSpools(spoolsData);
      setClusters(clustersData);
    } catch (err) {
      console.error('Failed to load spools:', err);
    }
  }

  async function handleAddSpool() {
    if (!newSpool.color || !newSpool.assignedTo) {
      alert('Please fill in all required fields');
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
      setNewSpool({
        material: 'PLA',
        color: '',
        colorHex: '#FF0000',
        totalWeight: 1000,
        assignedTo: '',
      });
      setShowAddModal(false);
      loadData();
    } catch (err) {
      alert('Failed to add spool: ' + err.message);
    }
  }

  function getRemainingPercent(spool) {
    if (spool.total_weight_g <= 0) return 0;
    return (spool.remaining_weight_g / spool.total_weight_g) * 100;
  }

  function getPrinterName(printerId) {
    for (const cluster of clusters) {
      const printer = cluster.printers.find(p => p.id === printerId);
      if (printer) return printer.name;
    }
    return 'Unassigned';
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-3xl font-bold">🧵 Filament Inventory</h2>
        <button
          onClick={() => setShowAddModal(true)}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
        >
          + Add Spool
        </button>
      </div>

      {spools.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          <p className="text-xl mb-2">No spools registered</p>
          <p>Add your first filament spool to track usage</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {spools.map(spool => (
            <div
              key={spool.id}
              className="bg-gray-800 rounded-xl border border-gray-700 overflow-hidden"
            >
              {/* Color Header */}
              <div
                className="h-24 flex items-center justify-center"
                style={{ backgroundColor: spool.color_hex + '40' }}
              >
                <div
                  className="w-16 h-16 rounded-full border-4 border-white shadow-lg"
                  style={{ backgroundColor: spool.color_hex }}
                />
              </div>

              {/* Spool Info */}
              <div className="p-4 space-y-3">
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="font-bold">{spool.color}</h3>
                    <p className="text-sm text-gray-400">{spool.material}</p>
                  </div>
                  <span className="text-xs text-gray-500">
                    #{spool.id}
                  </span>
                </div>

                {/* Progress Bar */}
                <div>
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-gray-400">Remaining</span>
                    <span>
                      {spool.remaining_weight_g.toFixed(0)}g / {spool.total_weight_g.toFixed(0)}g
                    </span>
                  </div>
                  <div className="h-2 bg-gray-700 rounded-full overflow-hidden">
                    <div
                      className="h-full transition-all duration-300"
                      style={{
                        width: `${getRemainingPercent(spool)}%`,
                        backgroundColor: getRemainingPercent(spool) > 20 ? '#22c55e' : '#ef4444',
                      }}
                    />
                  </div>
                </div>

                {/* Assigned Printer */}
                <div className="pt-3 border-t border-gray-700">
                  <p className="text-xs text-gray-400 mb-1">Assigned to:</p>
                  <p className="text-sm font-medium">
                    {spool.assigned_printer_id ? getPrinterName(spool.assigned_printer_id) : 'None'}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Add Spool Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-gray-800 rounded-xl p-6 w-full max-w-md border border-gray-700">
            <h3 className="text-xl font-bold mb-4">Add Filament Spool</h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-gray-400 mb-1">Material</label>
                <select
                  value={newSpool.material}
                  onChange={(e) => setNewSpool({ ...newSpool, material: e.target.value })}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                >
                  <option value="PLA">PLA</option>
                  <option value="PETG">PETG</option>
                  <option value="ABS">ABS</option>
                  <option value="ASA">ASA</option>
                  <option value="TPU">TPU</option>
                  <option value="PC">Polycarbonate</option>
                  <option value="Nylon">Nylon</option>
                  <option value="PVA">PVA (Support)</option>
                </select>
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Color Name</label>
                <input
                  type="text"
                  value={newSpool.color}
                  onChange={(e) => setNewSpool({ ...newSpool, color: e.target.value })}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                  placeholder="e.g., Silk Red"
                />
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Color Hex</label>
                <div className="flex gap-2">
                  <input
                    type="color"
                    value={newSpool.colorHex}
                    onChange={(e) => setNewSpool({ ...newSpool, colorHex: e.target.value })}
                    className="w-12 h-10 rounded cursor-pointer"
                  />
                  <input
                    type="text"
                    value={newSpool.colorHex}
                    onChange={(e) => setNewSpool({ ...newSpool, colorHex: e.target.value })}
                    className="flex-1 bg-gray-900 border border-gray-600 rounded px-3 py-2"
                    placeholder="#RRGGBB"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Total Weight (grams)</label>
                <input
                  type="number"
                  value={newSpool.totalWeight}
                  onChange={(e) => setNewSpool({ ...newSpool, totalWeight: e.target.value })}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                  placeholder="1000"
                />
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Assign to Printer</label>
                <select
                  value={newSpool.assignedTo}
                  onChange={(e) => setNewSpool({ ...newSpool, assignedTo: e.target.value })}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                >
                  <option value="">Select printer...</option>
                  {clusters.flatMap(c => c.printers.map(p => (
                    <option key={p.id} value={p.id}>
                      {c.name} - {p.name}
                    </option>
                  )))}
                </select>
              </div>
            </div>

            <div className="flex gap-3 mt-6">
              <button
                onClick={handleAddSpool}
                className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded transition-colors"
              >
                Add Spool
              </button>
              <button
                onClick={() => setShowAddModal(false)}
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

export default Spools;
