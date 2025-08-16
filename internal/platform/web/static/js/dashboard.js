// DriftMgr Web Dashboard JavaScript
// This file contains the main JavaScript functionality for the web dashboard

class DriftMgrDashboard {
    constructor() {
        this.websocket = null;
        this.charts = {};
        this.init();
    }

    init() {
        this.setupWebSocket();
        this.setupEventListeners();
        this.loadInitialData();
    }

    setupWebSocket() {
        // Connect to WebSocket for real-time updates
        this.websocket = new WebSocket(`ws://${window.location.host}/ws`);
        
        this.websocket.onopen = () => {
            console.log('WebSocket connected');
            this.updateConnectionStatus(true);
        };

        this.websocket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleWebSocketMessage(data);
        };

        this.websocket.onclose = () => {
            console.log('WebSocket disconnected');
            this.updateConnectionStatus(false);
            // Attempt to reconnect after 5 seconds
            setTimeout(() => this.setupWebSocket(), 5000);
        };
    }

    setupEventListeners() {
        // File upload handling
        const fileUpload = document.getElementById('file-upload');
        if (fileUpload) {
            fileUpload.addEventListener('change', (e) => this.handleFileUpload(e));
        }

        // Discovery form handling
        const discoveryForm = document.getElementById('discovery-form');
        if (discoveryForm) {
            discoveryForm.addEventListener('submit', (e) => this.handleDiscovery(e));
        }

        // Analysis form handling
        const analysisForm = document.getElementById('analysis-form');
        if (analysisForm) {
            analysisForm.addEventListener('submit', (e) => this.handleAnalysis(e));
        }
    }

    async loadInitialData() {
        try {
            // Load resources
            const resourcesResponse = await fetch('/api/v1/discover');
            const resources = await resourcesResponse.json();
            this.updateResourcesTable(resources);

            // Load state files
            const stateFilesResponse = await fetch('/api/v1/statefiles');
            const stateFiles = await stateFilesResponse.json();
            this.updateStateFilesTable(stateFiles);

            // Load drift analysis
            const driftResponse = await fetch('/api/v1/drift');
            const driftData = await driftResponse.json();
            this.updateDriftCharts(driftData);

        } catch (error) {
            console.error('Error loading initial data:', error);
            this.showError('Failed to load initial data');
        }
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'discovery_progress':
                this.updateDiscoveryProgress(data);
                break;
            case 'analysis_progress':
                this.updateAnalysisProgress(data);
                break;
            case 'drift_update':
                this.updateDriftData(data);
                break;
            case 'resource_update':
                this.updateResourceData(data);
                break;
            default:
                console.log('Unknown message type:', data.type);
        }
    }

    async handleFileUpload(event) {
        const file = event.target.files[0];
        if (!file) return;

        const formData = new FormData();
        formData.append('file', file);

        try {
            const response = await fetch('/api/v1/upload', {
                method: 'POST',
                body: formData
            });

            if (response.ok) {
                const result = await response.json();
                this.showSuccess('File uploaded successfully');
                this.loadInitialData(); // Refresh data
            } else {
                this.showError('File upload failed');
            }
        } catch (error) {
            console.error('Upload error:', error);
            this.showError('File upload failed');
        }
    }

    async handleDiscovery(event) {
        event.preventDefault();
        const formData = new FormData(event.target);
        const provider = formData.get('provider');
        const regions = formData.get('regions').split(',').map(r => r.trim());

        try {
            const response = await fetch('/api/v1/discover', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    provider: provider,
                    regions: regions
                })
            });

            if (response.ok) {
                this.showSuccess('Discovery started');
                this.updateDiscoveryStatus('Running...');
            } else {
                this.showError('Discovery failed to start');
            }
        } catch (error) {
            console.error('Discovery error:', error);
            this.showError('Discovery failed');
        }
    }

    async handleAnalysis(event) {
        event.preventDefault();
        const formData = new FormData(event.target);
        const stateFileId = formData.get('statefile_id');

        try {
            const response = await fetch('/api/v1/analyze', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    statefile_id: stateFileId
                })
            });

            if (response.ok) {
                this.showSuccess('Analysis started');
                this.updateAnalysisStatus('Running...');
            } else {
                this.showError('Analysis failed to start');
            }
        } catch (error) {
            console.error('Analysis error:', error);
            this.showError('Analysis failed');
        }
    }

    updateConnectionStatus(connected) {
        const statusElement = document.getElementById('connection-status');
        if (statusElement) {
            statusElement.textContent = connected ? 'Connected' : 'Disconnected';
            statusElement.className = connected ? 'badge badge-success' : 'badge badge-error';
        }
    }

    updateDiscoveryProgress(data) {
        const progressElement = document.getElementById('discovery-progress');
        if (progressElement) {
            progressElement.style.width = `${data.progress}%`;
            progressElement.textContent = `${data.progress}%`;
        }

        if (data.complete) {
            this.showSuccess('Discovery completed');
            this.loadInitialData(); // Refresh data
        }
    }

    updateAnalysisProgress(data) {
        const progressElement = document.getElementById('analysis-progress');
        if (progressElement) {
            progressElement.style.width = `${data.progress}%`;
            progressElement.textContent = `${data.progress}%`;
        }

        if (data.complete) {
            this.showSuccess('Analysis completed');
            this.loadInitialData(); // Refresh data
        }
    }

    updateResourcesTable(resources) {
        const tableBody = document.getElementById('resources-table-body');
        if (!tableBody) return;

        tableBody.innerHTML = '';
        resources.forEach(resource => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${resource.name}</td>
                <td>${resource.type}</td>
                <td>${resource.provider}</td>
                <td>${resource.region}</td>
                <td>${resource.status}</td>
            `;
            tableBody.appendChild(row);
        });
    }

    updateStateFilesTable(stateFiles) {
        const tableBody = document.getElementById('statefiles-table-body');
        if (!tableBody) return;

        tableBody.innerHTML = '';
        stateFiles.forEach(stateFile => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${stateFile.id}</td>
                <td>${stateFile.path}</td>
                <td>${stateFile.size}</td>
                <td>${stateFile.modified}</td>
            `;
            tableBody.appendChild(row);
        });
    }

    updateDriftCharts(driftData) {
        // Update drift statistics
        const statsElement = document.getElementById('drift-stats');
        if (statsElement) {
            statsElement.innerHTML = `
                <div class="stat">
                    <div class="stat-title">Total Drifts</div>
                    <div class="stat-value">${driftData.total}</div>
                </div>
                <div class="stat">
                    <div class="stat-title">Critical</div>
                    <div class="stat-value text-error">${driftData.critical}</div>
                </div>
                <div class="stat">
                    <div class="stat-title">High</div>
                    <div class="stat-value text-warning">${driftData.high}</div>
                </div>
                <div class="stat">
                    <div class="stat-title">Medium</div>
                    <div class="stat-value text-info">${driftData.medium}</div>
                </div>
                <div class="stat">
                    <div class="stat-title">Low</div>
                    <div class="stat-value text-success">${driftData.low}</div>
                </div>
            `;
        }
    }

    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showError(message) {
        this.showNotification(message, 'error');
    }

    showNotification(message, type) {
        const notification = document.createElement('div');
        notification.className = `alert alert-${type === 'success' ? 'success' : 'error'} fixed top-4 right-4 z-50`;
        notification.innerHTML = `
            <span>${message}</span>
            <button class="btn btn-sm btn-circle" onclick="this.parentElement.remove()">✕</button>
        `;
        document.body.appendChild(notification);

        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (notification.parentElement) {
                notification.remove();
            }
        }, 5000);
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.driftMgrDashboard = new DriftMgrDashboard();
});
