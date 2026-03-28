import { useState, useEffect } from 'react';
import { ToastProvider, useToast } from './components/ui/Toast';
import { api } from './services/api';
import Sidebar from './components/layout/Sidebar';
import Dashboard from './components/pages/Dashboard';
import Jobs from './components/pages/Jobs';
import Clusters from './components/pages/Clusters';
import Spools from './components/pages/Spools';
import Settings from './components/pages/Settings';

function AppContent() {
  const [currentPage, setCurrentPage] = useState('dashboard');
  const [printers, setPrinters] = useState({});
  const toast = useToast();

  // WebSocket connection for real-time updates
  useEffect(() => {
    const cleanup = window.wsCleanup || (() => { });
    cleanup();

    const unsubscribe = window.wsService.subscribe((packet) => {
      if (packet.event_type === 'SYSTEM_LOG' && packet.log) {
        if (packet.log.level === 'ERROR') {
          toast.error(packet.log.message);
        } else if (packet.log.level === 'INFO') {
          toast.info(packet.log.message, 3000);
        }
      } else if (packet.event_type === 'PRINTER_STATUS' && packet.payload) {
        const p = packet.payload;
        setPrinters(prev => ({
          ...prev,
          [p.printer_id]: {
            ...prev[p.printer_id],
            status: p.status,
            progress: p.progress,
            nozzle_temper: p.nozzle_temper,
            bed_temper: p.bed_temper,
            nozzle_target_temper: p.nozzle_target_temper,
            bed_target_temper: p.bed_target_temper,
            fan_speed: p.fan_speed,
            layer: p.layer,
            total_layers: p.total_layers,
            time_left: p.time_left,
            filename: p.filename,
          }
        }));
      }
    });

    window.wsCleanup = unsubscribe;

    // Listen for navigation events from child components
    const handleNavigate = (event) => {
      if (event.detail && event.detail.page) {
        setCurrentPage(event.detail.page);
      }
    };

    window.addEventListener('navigate', handleNavigate);

    return () => {
      unsubscribe();
      window.removeEventListener('navigate', handleNavigate);
    };
  }, [toast]);

  const renderPage = () => {
    switch (currentPage) {
      case 'dashboard':
        return <Dashboard />;
      case 'jobs':
        return <Jobs />;
      case 'clusters':
        return <Clusters livePrinters={printers} />;
      case 'spools':
        return <Spools />;
      case 'settings':
        return <Settings />;
      default:
        return <Dashboard />;
    }
  };

  return (
    <div className="min-h-screen">
      <Sidebar currentPage={currentPage} onPageChange={setCurrentPage} />
      <main className="ml-64 p-8">
        {renderPage()}
      </main>
    </div>
  );
}

function App() {
  return (
    <ToastProvider>
      <AppContent />
    </ToastProvider>
  );
}

export default App;
