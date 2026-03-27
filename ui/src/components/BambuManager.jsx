import { useState } from 'react';
import { api } from '../services/api';

function BambuManager({ onPrinterAdded, clusters }) {
  const [connectionMode, setConnectionMode] = useState('LAN'); // LAN or CLOUD
  const [showCloudLogin, setShowCloudLogin] = useState(false);
  const [showDiscover, setShowDiscover] = useState(false);
  const [show2FA, setShow2FA] = useState(false);
  const [discoveredPrinters, setDiscoveredPrinters] = useState([]);
  
  // LAN mode fields
  const [lanIP, setLanIP] = useState('');
  const [lanAccessCode, setLanAccessCode] = useState('');
  
  // Cloud mode fields
  const [cloudEmail, setCloudEmail] = useState('');
  const [cloudPassword, setCloudPassword] = useState('');
  const [cloudRegion, setCloudRegion] = useState('US');
  const [tfaCode, setTfaCode] = useState('');
  
  // Binding
  const [selectedPrinter, setSelectedPrinter] = useState(null);
  const [targetCluster, setTargetCluster] = useState('');

  async function handleCloudLogin() {
    try {
      await api.bambuLogin(cloudEmail, cloudPassword, cloudRegion);
      setShowCloudLogin(false);
      handleDiscover();
    } catch (err) {
      alert('Login failed: ' + err.message);
    }
  }

  async function handleDiscover() {
    try {
      const result = await api.bambuDiscover(cloudEmail);
      if (result.tfa_required) {
        setShow2FA(true);
        return;
      }
      setDiscoveredPrinters(result.printers || []);
      setShowDiscover(true);
    } catch (err) {
      alert('Discovery failed: ' + err.message);
    }
  }

  async function handleVerify2FA() {
    try {
      await api.bambuVerify2FA(cloudEmail, tfaCode);
      setShow2FA(false);
      handleDiscover();
    } catch (err) {
      alert('2FA verification failed: ' + err.message);
    }
  }

  async function handleBindPrinter() {
    if (!selectedPrinter || !targetCluster) {
      alert('Please select a printer and cluster');
      return;
    }

    try {
      await api.bambuBind(
        selectedPrinter.serial,
        selectedPrinter.name,
        selectedPrinter.model,
        parseInt(targetCluster),
        selectedPrinter.ip || ''
      );
      
      setShowDiscover(false);
      setDiscoveredPrinters([]);
      setSelectedPrinter(null);
      onPrinterAdded();
      alert(`Printer '${selectedPrinter.name}' bound successfully!`);
    } catch (err) {
      alert('Binding failed: ' + err.message);
    }
  }

  async function handleAddLANPrinter() {
    if (!lanIP || !lanAccessCode) {
      alert('Please enter IP address and access code');
      return;
    }

    // For LAN mode, user needs to manually create cluster first
    // This is a simplified flow - in production you'd validate the connection
    alert(`LAN mode configured for ${lanIP}\nAccess Code: ${'*'.repeat(lanAccessCode.length)}\n\nPlease ensure you have created a cluster for this printer model first.`);
    
    // In production, this would add the printer directly
    onPrinterAdded();
  }

  return (
    <div className="space-y-4">
      {/* Connection Mode Selection */}
      <div>
        <label className="block text-sm text-gray-400 mb-2">Connection Mode</label>
        <div className="flex gap-2">
          <button
            onClick={() => setConnectionMode('LAN')}
            className={`flex-1 px-4 py-2 rounded transition-colors ${
              connectionMode === 'LAN'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
            }`}
          >
            🏠 LAN (Local)
          </button>
          <button
            onClick={() => setConnectionMode('CLOUD')}
            className={`flex-1 px-4 py-2 rounded transition-colors ${
              connectionMode === 'CLOUD'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
            }`}
          >
            ☁️ Cloud (Bambu)
          </button>
        </div>
      </div>

      {/* LAN Mode Fields */}
      {connectionMode === 'LAN' && (
        <div className="space-y-3 p-4 bg-gray-900 rounded-lg">
          <p className="text-sm text-gray-400">
            Connect directly to your Bambu printer on the local network
          </p>
          
          <div>
            <label className="block text-sm text-gray-400 mb-1">Printer IP Address</label>
            <input
              type="text"
              value={lanIP}
              onChange={(e) => setLanIP(e.target.value)}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2"
              placeholder="e.g., 192.168.1.100"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">Access Code</label>
            <input
              type="text"
              value={lanAccessCode}
              onChange={(e) => setLanAccessCode(e.target.value)}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2"
              placeholder="6-character code from printer screen"
              maxLength="6"
            />
            <p className="text-xs text-gray-500 mt-1">
              Found on printer screen: Settings → About → Access Code
            </p>
          </div>

          <button
            onClick={handleAddLANPrinter}
            className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded transition-colors"
          >
            Add LAN Printer
          </button>
        </div>
      )}

      {/* Cloud Mode Fields */}
      {connectionMode === 'CLOUD' && (
        <div className="space-y-3 p-4 bg-gray-900 rounded-lg">
          <p className="text-sm text-gray-400">
            Connect via Bambu Lab cloud to access all your printers
          </p>

          {!showCloudLogin && !showDiscover ? (
            <button
              onClick={() => setShowCloudLogin(true)}
              className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded transition-colors"
            >
              🔐 Sign in to Bambu Lab
            </button>
          ) : null}

          {/* Cloud Login Modal */}
          {showCloudLogin && (
            <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
              <div className="bg-gray-800 rounded-xl p-6 w-full max-w-md border border-gray-700">
                <h3 className="text-xl font-bold mb-4">Bambu Lab Login</h3>
                
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm text-gray-400 mb-1">Email</label>
                    <input
                      type="email"
                      value={cloudEmail}
                      onChange={(e) => setCloudEmail(e.target.value)}
                      className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                      placeholder="your@email.com"
                    />
                  </div>

                  <div>
                    <label className="block text-sm text-gray-400 mb-1">Password</label>
                    <input
                      type="password"
                      value={cloudPassword}
                      onChange={(e) => setCloudPassword(e.target.value)}
                      className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                      placeholder="••••••••"
                    />
                  </div>

                  <div>
                    <label className="block text-sm text-gray-400 mb-1">Region</label>
                    <select
                      value={cloudRegion}
                      onChange={(e) => setCloudRegion(e.target.value)}
                      className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                    >
                      <option value="US">North America</option>
                      <option value="EU">Europe</option>
                      <option value="CN">China</option>
                    </select>
                  </div>
                </div>

                <div className="flex gap-3 mt-6">
                  <button
                    onClick={handleCloudLogin}
                    className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded transition-colors"
                  >
                    Sign In
                  </button>
                  <button
                    onClick={() => setShowCloudLogin(false)}
                    className="px-4 py-2 bg-gray-600 hover:bg-gray-700 rounded transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* 2FA Modal */}
          {show2FA && (
            <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
              <div className="bg-gray-800 rounded-xl p-6 w-full max-w-md border border-gray-700">
                <h3 className="text-xl font-bold mb-4">2FA Verification</h3>
                <p className="text-gray-400 mb-4">
                  Enter the 6-digit code from your authenticator app
                </p>
                
                <input
                  type="text"
                  value={tfaCode}
                  onChange={(e) => setTfaCode(e.target.value)}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2 text-center text-2xl tracking-widest"
                  placeholder="000000"
                  maxLength="6"
                />

                <div className="flex gap-3 mt-6">
                  <button
                    onClick={handleVerify2FA}
                    className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded transition-colors"
                  >
                    Verify
                  </button>
                  <button
                    onClick={() => setShow2FA(false)}
                    className="px-4 py-2 bg-gray-600 hover:bg-gray-700 rounded transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Printer Discovery */}
          {showDiscover && (
            <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 overflow-y-auto p-4">
              <div className="bg-gray-800 rounded-xl p-6 w-full max-w-2xl border border-gray-700 my-8">
                <h3 className="text-xl font-bold mb-4">Select Printer to Bind</h3>
                
                {discoveredPrinters.length === 0 ? (
                  <p className="text-gray-400">No printers found in your account</p>
                ) : (
                  <div className="space-y-2 max-h-64 overflow-y-auto mb-4">
                    {discoveredPrinters.map((p) => (
                      <div
                        key={p.serial}
                        onClick={() => setSelectedPrinter(p)}
                        className={`p-3 rounded-lg border cursor-pointer transition-colors ${
                          selectedPrinter?.serial === p.serial
                            ? 'bg-blue-600/20 border-blue-500'
                            : 'bg-gray-900 border-gray-700 hover:border-gray-600'
                        }`}
                      >
                        <div className="flex justify-between items-center">
                          <div>
                            <p className="font-medium">{p.name}</p>
                            <p className="text-sm text-gray-400">{p.model} ({p.serial})</p>
                          </div>
                          <span className={`px-2 py-1 rounded text-xs ${
                            p.online ? 'bg-green-500/20 text-green-400' : 'bg-gray-500/20 text-gray-400'
                          }`}>
                            {p.online ? '🟢 Online' : '⚪ Offline'}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {selectedPrinter && (
                  <div className="mb-4">
                    <label className="block text-sm text-gray-400 mb-1">Select Cluster</label>
                    <select
                      value={targetCluster}
                      onChange={(e) => setTargetCluster(e.target.value)}
                      className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2"
                    >
                      <option value="">Choose a cluster...</option>
                      {clusters.map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name} ({c.target_model})
                        </option>
                      ))}
                    </select>
                  </div>
                )}

                <div className="flex gap-3">
                  <button
                    onClick={handleBindPrinter}
                    disabled={!selectedPrinter || !targetCluster}
                    className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed rounded transition-colors"
                  >
                    Bind Printer
                  </button>
                  <button
                    onClick={() => {
                      setShowDiscover(false);
                      setDiscoveredPrinters([]);
                      setSelectedPrinter(null);
                    }}
                    className="px-4 py-2 bg-gray-600 hover:bg-gray-700 rounded transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}

          {showDiscover && (
            <button
              onClick={() => setShowDiscover(true)}
              className="w-full px-4 py-2 bg-green-600 hover:bg-green-700 rounded transition-colors"
            >
              🖨️ Discover Printers
            </button>
          )}
        </div>
      )}
    </div>
  );
}

export default BambuManager;
