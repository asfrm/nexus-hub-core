import { useState } from 'react';
import { api } from '../services/api';

function Settings() {
  const [analyzingFile, setAnalyzingFile] = useState(false);
  const [analysisResult, setAnalysisResult] = useState(null);
  const [autoMode, setAutoMode] = useState(false);

  async function handleAnalyzeFile(e) {
    const file = e.target.files[0];
    if (!file) return;

    setAnalyzingFile(true);
    setAnalysisResult(null);
    try {
      const result = await api.analyzeGCode(file);
      setAnalysisResult(result);
    } catch (err) {
      alert('Analysis failed: ' + err.message);
    } finally {
      setAnalyzingFile(false);
      e.target.value = '';
    }
  }

  return (
    <div className="space-y-6">
      <h2 className="text-3xl font-bold">Settings</h2>

      {/* Dispatch Mode */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
        <h3 className="text-xl font-semibold mb-4">Dispatch Mode</h3>
        <div className="flex items-center gap-4">
          <button
            onClick={() => setAutoMode(!autoMode)}
            className={`relative w-16 h-8 rounded-full transition-colors ${
              autoMode ? 'bg-blue-600' : 'bg-gray-600'
            }`}
          >
            <div
              className={`absolute top-1 w-6 h-6 bg-white rounded-full transition-transform ${
                autoMode ? 'left-9' : 'left-1'
              }`}
            />
          </button>
          <span className="text-gray-300">
            {autoMode ? 'Automatic Job Assignment' : 'Manual Job Assignment'}
          </span>
        </div>
        <p className="text-gray-400 text-sm mt-2">
          When enabled, jobs will be automatically assigned to available printers in matching clusters
        </p>
      </div>

      {/* G-Code Analysis Tool */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
        <h3 className="text-xl font-semibold mb-4">G-Code Analysis Tool</h3>
        <p className="text-gray-400 mb-4">
          Analyze a G-Code file to extract metadata before uploading
        </p>

        <label className="block w-full px-4 py-3 bg-blue-600 hover:bg-blue-700 rounded-lg cursor-pointer transition-colors text-center">
          <input type="file" accept=".gcode,.g" onChange={handleAnalyzeFile} className="hidden" />
          {analyzingFile ? 'Analyzing...' : '📁 Select G-Code File'}
        </label>

        {analysisResult && (
          <div className="mt-4 p-4 bg-gray-900 rounded-lg border border-gray-600">
            <h4 className="font-medium text-green-400 mb-3">✓ Analysis Complete</h4>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-gray-400">Printer Model</p>
                <p className="text-lg">{analysisResult.detected_model || 'Unknown'}</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">Filament Type</p>
                <p className="text-lg">{analysisResult.filament_type || 'Unknown'}</p>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* API Information */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
        <h3 className="text-xl font-semibold mb-4">API Endpoints</h3>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between py-2 border-b border-gray-700">
            <span className="text-gray-400">WebSocket</span>
            <code className="text-blue-400">/ws</code>
          </div>
          <div className="flex justify-between py-2 border-b border-gray-700">
            <span className="text-gray-400">Jobs API</span>
            <code className="text-blue-400">/api/jobs</code>
          </div>
          <div className="flex justify-between py-2 border-b border-gray-700">
            <span className="text-gray-400">Clusters API</span>
            <code className="text-blue-400">/api/clusters</code>
          </div>
          <div className="flex justify-between py-2 border-b border-gray-700">
            <span className="text-gray-400">File Upload</span>
            <code className="text-blue-400">/server/files/upload</code>
          </div>
        </div>
      </div>

      {/* System Information */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
        <h3 className="text-xl font-semibold mb-4">System Information</h3>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between py-2 border-b border-gray-700">
            <span className="text-gray-400">Nexus Hub Core</span>
            <code className="text-green-400">v1.0.0</code>
          </div>
          <div className="flex justify-between py-2 border-b border-gray-700">
            <span className="text-gray-400">Platform</span>
            <code className="text-gray-300">Cross-Platform (Linux/Windows)</code>
          </div>
          <div className="flex justify-between py-2 border-b border-gray-700">
            <span className="text-gray-400">Database</span>
            <code className="text-gray-300">SQLite</code>
          </div>
          <div className="flex justify-between py-2">
            <span className="text-gray-400">Printer Protocol</span>
            <code className="text-gray-300">Moonraker/Klipper</code>
          </div>
        </div>
      </div>
    </div>
  );
}

export default Settings;
