import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App.jsx';
import { wsService } from './services/websocket.js';

// Initialize WebSocket service globally
window.wsService = wsService;
wsService.connect();

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
