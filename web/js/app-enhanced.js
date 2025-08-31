// DriftMgr Enhanced Web Application
function driftmgrApp() {
    return {
        // Application state
        currentView: 'dashboard',
        loading: false,
        wsConnected: false,
        ws: null,
        environment: localStorage.getItem('driftmgr-environment') || 'production',
        
        // Dashboard data
        stats: {
            totalResources: 0,
            driftedResources: 0,
            compliantResources: 0,
            activeProviders: 0,
            configuredProviders: []
        },
        
        recentDrifts: [],
        
        // Discovery data
        discovering: false,
        discoveryForm: {
            provider: '',
            regions: '',
            environment: 'production'
        },
        discoveryJob: null,
        discoveryResults: [],
        
        // Resources data
        resources: [],
        resourceFilters: {
            search: '',
            provider: '',
            type: '',
            state: ''
        },
        selectedResources: [],
        showImportModal: false,
        showExportModal: false,
        exportFormat: 'json',
        
        // Accounts data
        accounts: [],
        currentAccount: null,
        showAccountSwitcher: false,
        
        // Audit logs
        auditLogs: [],
        auditFilters: {
            severity: '',
            event: '',
            user: '',
            dateFrom: '',
            dateTo: ''
        },
        
        // State management
        stateFiles: [],
        selectedStateFile: null,
        
        // Charts
        charts: {},
        
        // Initialize application
        init() {
            console.log('Initializing Enhanced DriftMgr Web App');
            this.initTheme();
            this.connectWebSocket();
            this.loadDashboardData();
            this.loadAccounts();
            this.initCharts();
            
            // Set up periodic refresh
            setInterval(() => {
                if (this.currentView === 'dashboard') {
                    this.refreshData();
                }
            }, 30000);
        },
        
        // Environment management
        setEnvironment(env) {
            this.environment = env;
            localStorage.setItem('driftmgr-environment', env);
            this.showNotification(`Environment switched to ${env}`, 'info');
            this.refreshData();
        },
        
        // WebSocket connection with enhanced updates
        connectWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;
            
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.wsConnected = true;
                this.ws.send(JSON.stringify({
                    type: 'subscribe',
                    environment: this.environment
                }));
            };
            
            this.ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            };
            
            this.ws.onclose = () => {
                console.log('WebSocket disconnected');
                this.wsConnected = false;
                setTimeout(() => this.connectWebSocket(), 5000);
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
            };
        },
        
        // Enhanced WebSocket message handling
        handleWebSocketMessage(data) {
            console.log('WebSocket message:', data);
            
            switch (data.type) {
                case 'discovery_started':
                    this.showNotification('Discovery started', 'info');
                    this.discoveryJob = {
                        id: data.job_id,
                        status: 'running',
                        progress: 0
                    };
                    break;
                    
                case 'discovery_progress':
                    if (this.discoveryJob && this.discoveryJob.id === data.job_id) {
                        this.discoveryJob.progress = data.progress;
                        this.discoveryJob.message = data.message;
                    }
                    break;
                    
                case 'discovery_completed':
                    if (this.discoveryJob && this.discoveryJob.id === data.job_id) {
                        this.discoveryJob.status = 'completed';
                        this.loadDiscoveryResults(data.job_id);
                        this.showNotification('Discovery completed', 'success');
                    }
                    break;
                    
                case 'drift_detected':
                    this.showNotification(`Drift detected: ${data.resource}`, 'warning');
                    this.refreshDriftData();
                    break;
                    
                case 'resource_deleted':
                    this.showNotification(`Resource deleted: ${data.resource}`, 'success');
                    this.refreshResources();
                    break;
                    
                case 'remediation_completed':
                    this.showNotification(`Remediation completed: ${data.resource}`, 'success');
                    this.refreshData();
                    break;
            }
        },
        
        // Enhanced dashboard data loading
        async loadDashboardData() {
            this.loading = true;
            
            try {
                // Load stats with environment context
                const statsResponse = await fetch(`/api/v1/resources/stats?environment=${this.environment}`);
                const statsData = await statsResponse.json();
                
                // Load recent drifts
                const driftResponse = await fetch(`/api/v1/drift/report?environment=${this.environment}`);
                const driftData = await driftResponse.json();
                
                // Enhanced statistics including configured providers
                this.stats = {
                    totalResources: statsData.total || 0,
                    driftedResources: driftData.summary?.drifted || 0,
                    compliantResources: driftData.summary?.compliant || (statsData.total || 0),
                    activeProviders: Object.keys(statsData.by_provider || {}).length,
                    configuredProviders: statsData.configured_providers || []
                };
                
                this.recentDrifts = driftData.drifts?.slice(0, 10) || [];
                this.updateCharts();
                
            } catch (error) {
                console.error('Error loading dashboard data:', error);
                this.showNotification('Failed to load dashboard data', 'error');
            } finally {
                this.loading = false;
            }
        },
        
        // Load cloud accounts
        async loadAccounts() {
            try {
                const response = await fetch('/api/v1/accounts');
                const data = await response.json();
                this.accounts = data.accounts || [];
                
                // Set current account if available
                if (this.accounts.length > 0 && !this.currentAccount) {
                    this.currentAccount = this.accounts[0];
                }
            } catch (error) {
                console.error('Error loading accounts:', error);
            }
        },
        
        // Switch account
        async switchAccount(account) {
            try {
                const response = await fetch('/api/v1/accounts/use', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({
                        provider: account.provider,
                        account_id: account.id
                    })
                });
                
                if (response.ok) {
                    this.currentAccount = account;
                    this.showAccountSwitcher = false;
                    this.showNotification(`Switched to ${account.name}`, 'success');
                    this.refreshData();
                }
            } catch (error) {
                console.error('Error switching account:', error);
                this.showNotification('Failed to switch account', 'error');
            }
        },
        
        // Resources management
        async loadResources() {
            this.loading = true;
            try {
                const params = new URLSearchParams({
                    environment: this.environment,
                    ...this.resourceFilters
                });
                
                const response = await fetch(`/api/v1/resources?${params}`);
                const data = await response.json();
                this.resources = data.resources || [];
            } catch (error) {
                console.error('Error loading resources:', error);
                this.showNotification('Failed to load resources', 'error');
            } finally {
                this.loading = false;
            }
        },
        
        // Delete resource
        async deleteResource(resourceId) {
            if (!confirm('Are you sure you want to delete this resource?')) {
                return;
            }
            
            try {
                const response = await fetch(`/api/v1/resources/${resourceId}`, {
                    method: 'DELETE'
                });
                
                if (response.ok) {
                    this.showNotification('Resource deleted successfully', 'success');
                    this.resources = this.resources.filter(r => r.id !== resourceId);
                } else {
                    throw new Error('Failed to delete resource');
                }
            } catch (error) {
                console.error('Error deleting resource:', error);
                this.showNotification('Failed to delete resource', 'error');
            }
        },
        
        // Bulk delete resources
        async deleteSelectedResources() {
            if (this.selectedResources.length === 0) {
                this.showNotification('No resources selected', 'warning');
                return;
            }
            
            if (!confirm(`Delete ${this.selectedResources.length} resources?`)) {
                return;
            }
            
            const deletePromises = this.selectedResources.map(id => 
                fetch(`/api/v1/resources/${id}`, { method: 'DELETE' })
            );
            
            try {
                await Promise.all(deletePromises);
                this.showNotification(`Deleted ${this.selectedResources.length} resources`, 'success');
                this.selectedResources = [];
                this.loadResources();
            } catch (error) {
                console.error('Error deleting resources:', error);
                this.showNotification('Failed to delete some resources', 'error');
            }
        },
        
        // Export resources
        async exportResources() {
            try {
                const response = await fetch(`/api/v1/resources/export?format=${this.exportFormat}`);
                const blob = await response.blob();
                
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `resources-${Date.now()}.${this.exportFormat}`;
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                a.remove();
                
                this.showNotification('Resources exported successfully', 'success');
                this.showExportModal = false;
            } catch (error) {
                console.error('Error exporting resources:', error);
                this.showNotification('Failed to export resources', 'error');
            }
        },
        
        // Import resources
        async importResources(file) {
            const formData = new FormData();
            formData.append('file', file);
            
            try {
                const response = await fetch('/api/v1/resources/import', {
                    method: 'POST',
                    body: formData
                });
                
                if (response.ok) {
                    const result = await response.json();
                    this.showNotification(`Imported ${result.count} resources`, 'success');
                    this.showImportModal = false;
                    this.loadResources();
                }
            } catch (error) {
                console.error('Error importing resources:', error);
                this.showNotification('Failed to import resources', 'error');
            }
        },
        
        // Enhanced discovery with auto-remediation
        async startDiscovery() {
            this.discovering = true;
            
            const payload = {
                provider: this.discoveryForm.provider || 'auto',
                regions: this.discoveryForm.regions.split(',').map(r => r.trim()),
                environment: this.environment,
                auto_remediate: this.discoveryForm.autoRemediate || false
            };
            
            try {
                const response = await fetch('/api/v1/discover', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                
                const result = await response.json();
                this.discoveryJob = {
                    id: result.job_id,
                    status: 'running',
                    progress: 0
                };
                
                // Poll for status
                this.pollDiscoveryStatus(result.job_id);
                
            } catch (error) {
                console.error('Error starting discovery:', error);
                this.showNotification('Failed to start discovery', 'error');
            } finally {
                this.discovering = false;
            }
        },
        
        // Verify discovery
        async verifyDiscovery() {
            try {
                const response = await fetch('/api/v1/verify', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({
                        provider: this.discoveryForm.provider,
                        environment: this.environment
                    })
                });
                
                const result = await response.json();
                this.showNotification(`Verification complete: ${result.accuracy}% accurate`, 'info');
            } catch (error) {
                console.error('Error verifying discovery:', error);
                this.showNotification('Failed to verify discovery', 'error');
            }
        },
        
        // Load audit logs
        async loadAuditLogs() {
            this.loading = true;
            try {
                const params = new URLSearchParams(this.auditFilters);
                const response = await fetch(`/api/v1/audit/logs?${params}`);
                const data = await response.json();
                this.auditLogs = data.events || [];
            } catch (error) {
                console.error('Error loading audit logs:', error);
                this.showNotification('Failed to load audit logs', 'error');
            } finally {
                this.loading = false;
            }
        },
        
        // Export audit logs
        async exportAuditLogs(format) {
            try {
                const response = await fetch(`/api/v1/audit/export?format=${format}`);
                const blob = await response.blob();
                
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `audit-logs-${Date.now()}.${format}`;
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                a.remove();
                
                this.showNotification('Audit logs exported', 'success');
            } catch (error) {
                console.error('Error exporting audit logs:', error);
                this.showNotification('Failed to export audit logs', 'error');
            }
        },
        
        // Auto-remediation management
        async enableAutoRemediation(dryRun = true) {
            try {
                const response = await fetch('/api/v1/drift/auto-remediate', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({
                        enabled: true,
                        dry_run: dryRun,
                        environment: this.environment
                    })
                });
                
                if (response.ok) {
                    this.showNotification(
                        dryRun ? 'Auto-remediation enabled (dry-run)' : 'Auto-remediation enabled',
                        'success'
                    );
                }
            } catch (error) {
                console.error('Error enabling auto-remediation:', error);
                this.showNotification('Failed to enable auto-remediation', 'error');
            }
        },
        
        // Enhanced chart updates with provider indicators
        updateCharts() {
            // Update provider chart to show configured vs active
            if (this.charts.driftByProvider) {
                const providers = ['aws', 'azure', 'gcp', 'digitalocean'];
                const data = providers.map(p => {
                    const isConfigured = this.stats.configuredProviders.includes(p);
                    const count = this.stats.by_provider?.[p] || 0;
                    return isConfigured ? count : 0;
                });
                
                this.charts.driftByProvider.data.datasets[0].data = data;
                this.charts.driftByProvider.update();
            }
        },
        
        // Utility functions
        refreshData() {
            this.loadDashboardData();
            if (this.currentView === 'resources') {
                this.loadResources();
            } else if (this.currentView === 'audit') {
                this.loadAuditLogs();
            }
        },
        
        refreshResources() {
            this.loadResources();
        },
        
        refreshDriftData() {
            this.loadDashboardData();
        },
        
        pollDiscoveryStatus(jobId) {
            const checkStatus = async () => {
                try {
                    const response = await fetch(`/api/v1/discover/status?job_id=${jobId}`);
                    const status = await response.json();
                    
                    if (this.discoveryJob && this.discoveryJob.id === jobId) {
                        this.discoveryJob.progress = status.progress;
                        this.discoveryJob.status = status.status;
                        
                        if (status.status === 'completed') {
                            this.loadDiscoveryResults(jobId);
                        } else if (status.status === 'running') {
                            setTimeout(checkStatus, 2000);
                        }
                    }
                } catch (error) {
                    console.error('Error checking discovery status:', error);
                }
            };
            
            setTimeout(checkStatus, 2000);
        },
        
        async loadDiscoveryResults(jobId) {
            try {
                const response = await fetch(`/api/v1/discover/results?job_id=${jobId}`);
                const data = await response.json();
                this.discoveryResults = data.resources || [];
                this.refreshData();
            } catch (error) {
                console.error('Error loading discovery results:', error);
            }
        },
        
        showNotification(message, type = 'info') {
            // Same as before
            const toast = document.createElement('div');
            toast.className = 'toast toast-top toast-end z-50';
            
            const alertTypes = {
                'info': 'alert-info',
                'success': 'alert-success',
                'warning': 'alert-warning',
                'error': 'alert-error'
            };
            
            toast.innerHTML = `
                <div class="alert ${alertTypes[type] || 'alert-info'}">
                    <span>${message}</span>
                </div>
            `;
            
            document.body.appendChild(toast);
            setTimeout(() => toast.remove(), 5000);
        },
        
        setTheme(theme) {
            document.documentElement.setAttribute('data-theme', theme);
            localStorage.setItem('driftmgr-theme', theme);
            this.showNotification(`Theme changed to ${theme}`, 'info');
        },
        
        initTheme() {
            const savedTheme = localStorage.getItem('driftmgr-theme');
            if (savedTheme) {
                document.documentElement.setAttribute('data-theme', savedTheme);
            }
        },
        
        // Initialize charts
        initCharts() {
            // Same as before but with enhancements
            const computedStyle = getComputedStyle(document.documentElement);
            
            // Provider chart with configured indicator
            const driftByProviderCtx = document.getElementById('driftByProviderChart');
            if (driftByProviderCtx) {
                this.charts.driftByProvider = new Chart(driftByProviderCtx, {
                    type: 'doughnut',
                    data: {
                        labels: ['AWS', 'Azure', 'GCP', 'DigitalOcean'],
                        datasets: [{
                            data: [0, 0, 0, 0],
                            backgroundColor: ['#ff9900', '#0078d4', '#4285f4', '#0080ff'],
                            borderWidth: 2,
                            borderColor: computedStyle.getPropertyValue('--b1')
                        }]
                    },
                    options: {
                        responsive: true,
                        plugins: {
                            legend: {
                                position: 'bottom',
                                labels: {
                                    generateLabels: (chart) => {
                                        const original = Chart.defaults.plugins.legend.labels.generateLabels(chart);
                                        return original.map((label, i) => {
                                            const provider = ['aws', 'azure', 'gcp', 'digitalocean'][i];
                                            const isConfigured = this.stats.configuredProviders.includes(provider);
                                            label.text = isConfigured ? `${label.text} ✓` : `${label.text} ✗`;
                                            return label;
                                        });
                                    }
                                }
                            }
                        }
                    }
                });
            }
        }
    };
}