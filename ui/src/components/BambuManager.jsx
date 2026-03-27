import { useState, useEffect } from 'react';
import { api } from '../services/api';

function BambuManager({ onPrinterAdded, clusters }) {
  const [connectionMode, setConnectionMode] = useState('LAN'); // LAN or CLOUD
  const [authMethod, setAuthMethod] = useState('TOKEN'); // TOKEN or PASSWORD

  // LAN mode fields
  const [lanIP, setLanIP] = useState('');
  const [lanAccessCode, setLanAccessCode] = useState(''); // Now supports 8 characters

  // Cloud mode - Token auth (OAuth-style)
  const [cloudEmail, setCloudEmail] = useState('');
  const [cloudUsername, setCloudUsername] = useState('');
  const [cloudUID, setCloudUID] = useState('');
  const [cloudToken, setCloudToken] = useState('');
  const [cloudRegion, setCloudRegion] = useState('US');

  // Cloud mode - Password auth (legacy)
  const [cloudPassword, setCloudPassword] = useState('');

  // 2FA
  const [show2FA, setShow2FA] = useState(false);
  const [tfaCode, setTfaCode] = useState('');

  // Discovery
  const [showDiscover, setShowDiscover] = useState(false);
  const [discoveredPrinters, setDiscoveredPrinters] = useState([]);
  const [selectedPrinter, setSelectedPrinter] = useState(null);
  const [targetCluster, setTargetCluster] = useState('');

  // Network scanner
  const [scanning, setScanning] = useState(false);
  const [scannedPrinters, setScannedPrinters] = useState([]);

  // Get login URL on mount
  useEffect(() => {
    loadLoginURL();
  }, []);

  async function loadLoginURL() {
    try {
      const result = await api.bambuGetLoginURL();
      // Store for use when user clicks login button
      window.bambuLoginURL = result.login_url;
    } catch (err) {
      console.error('Failed to load login URL:', err);
    }
  }

  function handleOpenBambuLogin() {
    // Open Bambu login in popup
    const width = 600;
    const height = 700;
    const left = window.screenX + (window.outerWidth - width) / 2;
    const top = window.screenY + (window.outerHeight - height) / 2;

    const popup = window.open(
      window.bambuLoginURL || 'https://bambulab.com/en/login',
      'BambuLogin',
      `width=${width},height=${height},left=${left},top=${top}`
    );

    // Listen for callback from popup
    window.addEventListener('message', handleOAuthCallback);

    // Check if popup was closed
    const checkClosed = setInterval(() => {
      if (popup && popup.closed) {
        clearInterval(checkClosed);
        window.removeEventListener('message', handleOAuthCallback);
      }
    }, 500);
  }

  function handleOAuthCallback(event) {
    if (event.data && event.data.success) {
      setCloudEmail(event.data.email);
      // Auto-discover printers after successful auth
      handleDiscover();
    }
  }

  async function handleTokenLogin() {
    if (!cloudEmail || !cloudToken) {
      alert('Please enter email and access token');
      return;
    }

    try {
      await api.bambuTokenLogin({
        email: cloudEmail,
        username: cloudUsername || cloudEmail.split('@')[0],
        uid: cloudUID || '',
        access_token: cloudToken,
        region: cloudRegion,
      });

      alert('Token saved successfully!');
      handleDiscover();
    } catch (err) {
      alert('Token login failed: ' + err.message);
    }
  }

  async function handlePasswordLogin() {
    if (!cloudEmail || !cloudPassword) {
      alert('Please enter email and password');
      return;
    }

    try {
      await api.bambuLogin(cloudEmail, cloudPassword, cloudRegion);
      handleDiscover();
    } catch (err) {
      const result = await api.bambuDiscover(cloudEmail);
      if (result.tfa_required) {
        setShow2FA(true);
        return;
      }
      alert('Login failed: ' + err.message);
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

  async function handleScanNetwork() {
    setScanning(true);
    try {
      const result = await api.scannerDiscover();
      setScannedPrinters(result.printers || []);
    } catch (err) {
      alert('Network scan failed: ' + err.message);
    } finally {
      setScanning(false);
    }
  }

  async function handleAddLANPrinter() {
    if (!lanIP || !lanAccessCode) {
      alert('Please enter IP address and access code');
      return;
    }

    // Validate 8-character access code
    if (lanAccessCode.length !== 8) {
      alert('Access code must be exactly 8 characters');
      return;
    }

    try {
      // Add printer with LAN credentials
      const extra = JSON.stringify({
        mode: 'LAN',
        local_ip: lanIP,
        access_code: lanAccessCode,
      });

      // Find or create cluster for Bambu Lab
      let clusterId = targetCluster;
      if (!clusterId) {
        // User needs to select a cluster first
        alert('Please select a target cluster first');
        return;
      }

      await api.addPrinter(
        parseInt(clusterId),
        `Bambu ${lanIP}`,
        lanIP,
        'BAMBULAB',
        '', '', '', '', extra
      );

      alert(`LAN printer added successfully at ${lanIP}`);
      onPrinterAdded();
    } catch (err) {
      alert('Failed to add LAN printer: ' + err.message);
    }
  }

  return (
    <div className="space-y-4">
      {/* Connection Mode Selection */}
      <div>
        <label className="block text-sm text-gray-400 mb-2">Connection Mode</label>
        <div className="flex gap-2">
          <button
            onClick={() => setConnectionMode('LAN')}
            className={`flex-1 px-4 py-2 rounded transition-colors ${connectionMode === 'LAN'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
              }`}
          >
            🏠 LAN (Local)
          </button>
          <button
            onClick={() => setConnectionMode('CLOUD')}
            className={`flex-1 px-4 py-2 rounded transition-colors ${connectionMode === 'CLOUD'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
              }`}
          >
            ☁️ Cloud (Bambu)
          </button>
        </div>
      </div>

      {/* Network Scanner (LAN mode) */}
      {connectionMode === 'LAN' && (
        <div className="p-4 bg-gray-900 rounded-lg">
          <div className="flex justify-between items-center mb-3">
            <h4 className="font-medium">Network Scanner</h4>
            <button
              onClick={handleScanNetwork}
              disabled={scanning}
              className="px-3 py-1 bg-purple-600 hover:bg-purple-700 disabled:bg-gray-600 rounded text-sm transition-colors"
            >
              {scanning ? '🔍 Scanning...' : '🔍 Scan Network'}
            </button>
          </div>

          {scannedPrinters.length > 0 && (
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {scannedPrinters.filter(p => p.printer_type === 'BAMBULAB').map((p, idx) => (
                <div
                  key={idx}
                  onClick={() => {
                    setLanIP(p.ip_address);
                  }}
                  className="p-2 bg-gray-800 rounded cursor-pointer hover:bg-gray-700"
                >
                  <div className="flex justify-between">
                    <span>{p.ip_address}</span>
                    <span className="text-xs text-gray-400">{p.printer_type}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

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
            <label className="block text-sm text-gray-400 mb-1">Access Code (8 characters)</label>
            <input
              type="text"
              value={lanAccessCode}
              onChange={(e) => setLanAccessCode(e.target.value.toUpperCase())}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 tracking-widest"
              placeholder="ABCDEFGH"
              maxLength="8"
            />
            <p className="text-xs text-gray-500 mt-1">
              Found on printer screen: Settings → About → Access Code (8 characters)
            </p>
            <p className={`text-xs mt-1 ${lanAccessCode.length === 8 ? 'text-green-400' : 'text-yellow-400'}`}>
              {lanAccessCode.length}/8 characters entered
            </p>
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">Target Cluster</label>
            <select
              value={targetCluster}
              onChange={(e) => setTargetCluster(e.target.value)}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2"
            >
              <option value="">Select a cluster...</option>
              {clusters.map((c) => (
                <option key={c.id} value={c.id}>{c.name} ({c.target_model})</option>
              ))}
            </select>
          </div>

          <button
            onClick={handleAddLANPrinter}
            disabled={!targetCluster || lanAccessCode.length !== 8}
            className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed rounded transition-colors"
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

          {/* Auth Method Selection */}
          <div className="flex gap-2 mb-3">
            <button
              onClick={() => setAuthMethod('TOKEN')}
              className={`flex-1 px-3 py-2 rounded text-sm transition-colors ${authMethod === 'TOKEN'
                  ? 'bg-green-600 text-white'
                  : 'bg-gray-700 text-gray-300'
                }`}
            >
              🔑 Access Token (Recommended)
            </button>
            <button
              onClick={() => setAuthMethod('PASSWORD')}
              className={`flex-1 px-3 py-2 rounded text-sm transition-colors ${authMethod === 'PASSWORD'
                  ? 'bg-green-600 text-white'
                  : 'bg-gray-700 text-gray-300'
                }`}
            >
              🔒 Password (Legacy)
            </button>
          </div>

          {/* Token Auth (OAuth-style) */}
          {authMethod === 'TOKEN' && (
            <>
              <button
                onClick={handleOpenBambuLogin}
                className="w-full px-4 py-3 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 rounded-lg transition-all font-medium"
              >
                🌐 Sign in via Bambu Lab
              </button>

              <div className="relative my-4">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-gray-700"></div>
                </div>
                <div className="relative flex justify-center text-sm">
                  <span className="px-2 bg-gray-900 text-gray-400">Or enter token manually</span>
                </div>
              </div>

              <div className="space-y-3">
                <div>
                  <label className="block text-sm text-gray-400 mb-1">Email</label>
                  <input
                    type="email"
                    value={cloudEmail}
                    onChange={(e) => setCloudEmail(e.target.value)}
                    className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2"
                    placeholder="your@email.com"
                  />
                </div>

                <div>
                  <label className="block text-sm text-gray-400 mb-1">
                    Bambu Cloud Token (User Access Token)
                  </label>
                  <input
                    type="text"
                    value={cloudToken}
                    onChange={(e) => setCloudToken(e.target.value)}
                    className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 font-mono"
                    placeholder="Paste your token here"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    📍 Find it at: Bambu Lab Profile → Access Token
                  </p>
                </div>

                <div>
                  <label className="block text-sm text-gray-400 mb-1">Region</label>
                  <select
                    value={cloudRegion}
                    onChange={(e) => setCloudRegion(e.target.value)}
                    className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2"
                  >
                    <option value="US">North America</option>
                    <option value="EU">Europe</option>
                    <option value="CN">China</option>
                  </select>
                </div>

                <button
                  onClick={handleTokenLogin}
                  className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded transition-colors"
                >
                  Save Token & Connect
                </button>
              </div>
            </>
          )}

          {/* Password Auth (Legacy) */}
          {authMethod === 'PASSWORD' && (
            <div className="space-y-3">
              <div>
                <label className="block text-sm text-gray-400 mb-1">Email</label>
                <input
                  type="email"
                  value={cloudEmail}
                  onChange={(e) => setCloudEmail(e.target.value)}
                  className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2"
                  placeholder="your@email.com"
                />
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Password</label>
                <input
                  type="password"
                  value={cloudPassword}
                  onChange={(e) => setCloudPassword(e.target.value)}
                  className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2"
                  placeholder="••••••••"
                />
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Region</label>
                <select
                  value={cloudRegion}
                  onChange={(e) => setCloudRegion(e.target.value)}
                  className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2"
                >
                  <option value="US">North America</option>
                  <option value="EU">Europe</option>
                  <option value="CN">China</option>
                </select>
              </div>

              <button
                onClick={handlePasswordLogin}
                className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded transition-colors"
              >
                Sign In
              </button>
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

          {/* Printer Discovery Modal */}
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
                        className={`p-3 rounded-lg border cursor-pointer transition-colors ${selectedPrinter?.serial === p.serial
                            ? 'bg-blue-600/20 border-blue-500'
                            : 'bg-gray-900 border-gray-700 hover:border-gray-600'
                          }`}
                      >
                        <div className="flex justify-between items-center">
                          <div>
                            <p className="font-medium">{p.name}</p>
                            <p className="text-sm text-gray-400">{p.model} ({p.serial})</p>
                          </div>
                          <span className={`px-2 py-1 rounded text-xs ${p.online ? 'bg-green-500/20 text-green-400' : 'bg-gray-500/20 text-gray-400'
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
        </div>
      )}
    </div>
  );
}

export default BambuManager;
