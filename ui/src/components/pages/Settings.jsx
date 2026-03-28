import { Card, CardHeader, CardContent, Button, Badge, Modal, useToast } from '../ui';
import { useState } from 'react';

export default function Settings() {
  const [showHealthModal, setShowHealthModal] = useState(false);
  const [health, setHealth] = useState(null);
  const toast = useToast();

  async function loadHealth() {
    try {
      const response = await fetch('/api/health');
      const data = await response.json();
      setHealth(data);
      setShowHealthModal(true);
    } catch (err) {
      toast.error('Failed to load health data: ' + err.message);
    }
  }

  return (
    <div className="space-y-6">
      <h2 className="text-3xl font-bold text-white">Settings</h2>

      {/* System Information */}
      <Card>
        <CardHeader title="ℹ️ System Information" subtitle="About your Nexus Hub instance" />
        <CardContent>
          <div className="grid grid-cols-2 gap-4">
            <InfoItem label="Version" value="2.0.3" />
            <InfoItem label="Platform" value="Cross-Platform" />
            <InfoItem label="Database" value="SQLite" />
            <InfoItem label="Protocol" value="MQTT + HTTP" />
          </div>
        </CardContent>
      </Card>

      {/* API Endpoints Reference */}
      <Card>
        <CardHeader title="🔌 API Endpoints" subtitle="For slicer integration" />
        <CardContent>
          <div className="space-y-2 text-sm">
            <ApiEndpoint method="GET" path="/api/jobs" description="Get pending jobs" />
            <ApiEndpoint method="GET" path="/api/jobs/list" description="Get active jobs" />
            <ApiEndpoint method="GET" path="/api/clusters" description="Get clusters" />
            <ApiEndpoint method="GET" path="/api/spools" description="Get filament spools" />
            <ApiEndpoint method="GET" path="/api/health" description="System health" />
            <ApiEndpoint method="POST" path="/api/files/local" description="Upload G-Code" />
            <ApiEndpoint method="WS" path="/ws" description="WebSocket updates" />
          </div>
        </CardContent>
      </Card>

      {/* Slicer Setup Guide */}
      <Card>
        <CardHeader title="🔧 Slicer Setup" subtitle="Connect OrcaSlicer or Bambu Studio" />
        <CardContent>
          <div className="glass rounded-lg p-4 text-sm">
            <h4 className="font-medium text-white mb-2">For OrcaSlicer / Bambu Studio:</h4>
            <ol className="list-decimal list-inside space-y-1 ml-4 text-gray-300">
              <li>Open <strong>Printer Settings → General</strong></li>
              <li>Set <strong>Host Type</strong> to: <code className="bg-gray-800 px-2 py-0.5 rounded text-xs">OctoPrint</code></li>
              <li>Set <strong>Hostname</strong> to: <code className="bg-gray-800 px-2 py-0.5 rounded text-xs">http://YOUR-IP:8080</code></li>
              <li>Leave API key blank</li>
              <li>Click <strong>Test Connection</strong></li>
            </ol>
          </div>
        </CardContent>
      </Card>

      {/* Health Modal */}
      <Modal isOpen={showHealthModal} onClose={() => setShowHealthModal(false)} title="📊 System Health" size="md">
        {health && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <HealthItem label="Status" value={health.status} icon={health.status === 'healthy' ? '✅' : '⚠️'} />
              <HealthItem label="Version" value={health.version} icon="📦" />
              <HealthItem label="Uptime" value={health.uptime} icon="⏱️" />
              <HealthItem label="Printers" value={health.printers} icon="🖨️" />
              <HealthItem label="Database" value={health.database} icon={health.database === 'connected' ? '✅' : '❌'} />
              <HealthItem label="Time" value={new Date(health.timestamp).toLocaleString()} icon="🕐" />
            </div>
            <div className="pt-4 border-t border-gray-700">
              <Button onClick={() => { loadHealth(); toast.success('Refreshed'); }} className="w-full">🔄 Refresh</Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}

function InfoItem({ label, value }) {
  return (
    <div className="glass rounded-lg p-3">
      <p className="text-xs text-gray-400">{label}</p>
      <p className="text-sm font-medium text-white mt-1">{value}</p>
    </div>
  );
}

function ApiEndpoint({ method, path, description }) {
  const colors = { GET: 'blue', POST: 'green', WS: 'purple' };
  return (
    <div className="flex items-center gap-3 glass rounded-lg p-2">
      <Badge variant={colors[method] || 'gray'}>{method}</Badge>
      <code className="text-sm text-gray-300 flex-1">{path}</code>
      <span className="text-xs text-gray-400">{description}</span>
    </div>
  );
}

function HealthItem({ label, value, icon }) {
  return (
    <div className="glass rounded-lg p-3 flex items-center gap-3">
      <span className="text-2xl">{icon}</span>
      <div>
        <p className="text-xs text-gray-400">{label}</p>
        <p className="text-sm font-medium text-white">{value}</p>
      </div>
    </div>
  );
}
