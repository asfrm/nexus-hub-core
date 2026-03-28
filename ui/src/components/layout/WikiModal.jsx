import { Modal, Button } from '../ui';

export default function WikiModal({ isOpen, onClose }) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title="📖 Help & Documentation" size="lg">
      <div className="space-y-6 text-gray-300">
        {/* Slicer Setup */}
        <section>
          <h3 className="text-lg font-semibold text-white mb-3 flex items-center gap-2">
            <span>🔌</span> Connecting Slicers
          </h3>
          <div className="glass rounded-lg p-4 space-y-3">
            <p className="text-sm">To send prints directly from OrcaSlicer or Bambu Studio:</p>
            <ol className="list-decimal list-inside space-y-2 text-sm ml-2">
              <li>Open your slicer settings</li>
              <li>Go to <strong className="text-white">Printer Settings → General</strong></li>
              <li>Set <strong className="text-white">Host Type</strong> to: <code className="bg-gray-800 px-2 py-0.5 rounded">OctoPrint</code></li>
              <li>Set <strong className="text-white">Hostname / IP</strong> to: <code className="bg-gray-800 px-2 py-0.5 rounded">http://YOUR-IP:8080</code></li>
              <li>Leave API key blank (not required)</li>
              <li>Click <strong className="text-white">Test Connection</strong> - should show "Success"</li>
            </ol>
            <p className="text-xs text-gray-400 mt-3">
              Files uploaded from the slicer will appear in the Job Queue with thumbnails and metadata automatically extracted.
            </p>
          </div>
        </section>

        {/* Adding Printers */}
        <section>
          <h3 className="text-lg font-semibold text-white mb-3 flex items-center gap-2">
            <span>🖨️</span> Adding Printers
          </h3>
          <div className="glass rounded-lg p-4 space-y-4">
            <div>
              <h4 className="font-medium text-white mb-2">Bambu Lab (LAN Mode)</h4>
              <ul className="list-disc list-inside space-y-1 text-sm text-gray-400 ml-2">
                <li>Enter printer IP address</li>
                <li>Click "Scan Network" to auto-discover</li>
                <li>Enter 8-character Access Code from printer screen</li>
                <li>Enter Serial Number (found in Bambu Handy app)</li>
              </ul>
            </div>
            <div>
              <h4 className="font-medium text-white mb-2">Moonraker / Klipper</h4>
              <ul className="list-disc list-inside space-y-1 text-sm text-gray-400 ml-2">
                <li>Enter printer name and IP address</li>
                <li>Ensure Moonraker is running and accessible</li>
                <li>Optional: Click "Scan Network" to discover</li>
              </ul>
            </div>
          </div>
        </section>

        {/* Filament Tracking */}
        <section>
          <h3 className="text-lg font-semibold text-white mb-3 flex items-center gap-2">
            <span>🧵</span> Filament Tracking
          </h3>
          <div className="glass rounded-lg p-4">
            <ul className="list-disc list-inside space-y-2 text-sm text-gray-400 ml-2">
              <li>Add spools in the <strong className="text-white">Filament</strong> tab</li>
              <li>Assign each spool to a printer</li>
              <li>Filament weight is automatically deducted when prints complete</li>
              <li>Progress bars show remaining filament at a glance</li>
              <li>Red warning when spool is below 20%</li>
            </ul>
          </div>
        </section>

        {/* Job Queue */}
        <section>
          <h3 className="text-lg font-semibold text-white mb-3 flex items-center gap-2">
            <span>📋</span> Job Queue Features
          </h3>
          <div className="glass rounded-lg p-4">
            <ul className="list-disc list-inside space-y-2 text-sm text-gray-400 ml-2">
              <li><strong className="text-white">Thumbnail preview</strong> - Extracted from 3MF files</li>
              <li><strong className="text-white">Estimated print time</strong> - From slicer metadata</li>
              <li><strong className="text-white">Filament weight</strong> - In grams for inventory tracking</li>
              <li><strong className="text-white">Filament type</strong> - PLA, PETG, ABS, etc.</li>
              <li><strong className="text-white">Printer compatibility</strong> - Green if cluster exists, red if not</li>
            </ul>
          </div>
        </section>

        {/* Keyboard Shortcuts */}
        <section>
          <h3 className="text-lg font-semibold text-white mb-3 flex items-center gap-2">
            <span>⌨️</span> Keyboard Shortcuts
          </h3>
          <div className="grid grid-cols-2 gap-3">
            <div className="glass rounded-lg p-3 flex justify-between items-center">
              <span className="text-sm text-gray-400">Dashboard</span>
              <kbd className="bg-gray-700 px-2 py-1 rounded text-xs">Ctrl+1</kbd>
            </div>
            <div className="glass rounded-lg p-3 flex justify-between items-center">
              <span className="text-sm text-gray-400">Job Queue</span>
              <kbd className="bg-gray-700 px-2 py-1 rounded text-xs">Ctrl+2</kbd>
            </div>
            <div className="glass rounded-lg p-3 flex justify-between items-center">
              <span className="text-sm text-gray-400">Printers</span>
              <kbd className="bg-gray-700 px-2 py-1 rounded text-xs">Ctrl+3</kbd>
            </div>
            <div className="glass rounded-lg p-3 flex justify-between items-center">
              <span className="text-sm text-gray-400">Filament</span>
              <kbd className="bg-gray-700 px-2 py-1 rounded text-xs">Ctrl+4</kbd>
            </div>
          </div>
        </section>

        {/* Footer */}
        <div className="pt-4 border-t border-gray-700">
          <p className="text-sm text-gray-500">
            <strong className="text-gray-400">Nexus Hub Core V2.0.3</strong> - Open Source 3D Printing Farm Orchestration
          </p>
          <p className="text-xs text-gray-500 mt-1">
            Built with Go + React | Supports Moonraker/Klipper & Bambu Lab
          </p>
        </div>

        {/* Close Button */}
        <div className="flex justify-end pt-4">
          <Button onClick={onClose}>Got it!</Button>
        </div>
      </div>
    </Modal>
  );
}
