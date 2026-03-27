// API Service Layer for backend communication
const API_BASE = '';

export const api = {
  // Jobs
  async getPendingJobs() {
    const res = await fetch(`${API_BASE}/api/jobs`);
    if (!res.ok) throw new Error('Failed to fetch pending jobs');
    return res.json();
  },

  async getActiveJobs() {
    const res = await fetch(`${API_BASE}/api/jobs/list`);
    if (!res.ok) throw new Error('Failed to fetch active jobs');
    return res.json();
  },

  async assignJob(jobId, clusterId, printerId) {
    const res = await fetch(`${API_BASE}/api/jobs/assign`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ job_id: jobId, cluster_id: clusterId, printer_id: printerId }),
    });
    if (!res.ok) throw new Error('Failed to assign job');
    return res.json();
  },

  // Clusters
  async getClusters() {
    const res = await fetch(`${API_BASE}/api/clusters`);
    if (!res.ok) throw new Error('Failed to fetch clusters');
    return res.json();
  },

  async createCluster(name, targetModel) {
    const res = await fetch(`${API_BASE}/api/clusters/create`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, target_model: targetModel }),
    });
    if (!res.ok) throw new Error('Failed to create cluster');
    return res.json();
  },

  async deleteCluster(id) {
    const res = await fetch(`${API_BASE}/api/clusters/delete?id=${id}`, {
      method: 'DELETE',
    });
    if (!res.ok) throw new Error('Failed to delete cluster');
    return res.json();
  },

  async toggleCluster(id, enabled) {
    const res = await fetch(`${API_BASE}/api/clusters/toggle?id=${id}&enabled=${enabled}`);
    if (!res.ok) throw new Error('Failed to toggle cluster');
    return res.json();
  },

  // Printers
  async addPrinter(clusterId, name, ipAddress) {
    const res = await fetch(`${API_BASE}/api/printers/add`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cluster_id: clusterId, name, ip_address: ipAddress }),
    });
    if (!res.ok) throw new Error('Failed to add printer');
    return res.json();
  },

  async controlPrinter(id, action) {
    const res = await fetch(`${API_BASE}/api/printers/control?id=${id}&action=${action}`);
    if (!res.ok) throw new Error('Failed to control printer');
    return res.json();
  },

  // Setup
  async getUnassignedModels() {
    const res = await fetch(`${API_BASE}/api/setup/unassigned`);
    if (!res.ok) throw new Error('Failed to fetch unassigned models');
    return res.json();
  },

  async analyzeGCode(file) {
    const formData = new FormData();
    formData.append('file', file);
    const res = await fetch(`${API_BASE}/api/setup/analyze-gcode`, {
      method: 'POST',
      body: formData,
    });
    if (!res.ok) throw new Error('Failed to analyze G-Code');
    return res.json();
  },

  // File Upload
  async uploadGCode(file) {
    const formData = new FormData();
    formData.append('file', file);
    const res = await fetch(`${API_BASE}/server/files/upload`, {
      method: 'POST',
      body: formData,
    });
    if (!res.ok) throw new Error('Failed to upload G-Code');
    return res.json();
  },
};
