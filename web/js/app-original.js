// DriftMgr Web Application
function driftmgrApp() {
    return {
        // Application state
        currentView: 'dashboard',
        loading: false,
        wsConnected: false,
        ws: null,
        
        // Dashboard data
        stats: {
            totalResources: 0,
            driftedResources: 0,
            compliantResources: 0,
            activeProviders: 0
        },
        
        recentDrifts: [],
        
        // Discovery data
        discovering: false,
        discoveryForm: {
            provider: '',
            regions: ''
        },
        discoveryJob: null,
        discoveryResults: [],
        
        // Charts
        charts: {},
        
        // Initialize application
        init() {
            console.log('Initializing DriftMgr Web App');
            this.initTheme();
            this.connectWebSocket();
            this.loadDashboardData();
            this.initCharts();
            
            // Set up periodic refresh
            setInterval(() => {
                if (this.currentView === 'dashboard') {
                    this.refreshData();
                }
            }, 30000); // Refresh every 30 seconds
        },
        
        // WebSocket connection
        connectWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;
            
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.wsConnected = true;
            };
            
            this.ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            };
            
            this.ws.onclose = () => {
                console.log('WebSocket disconnected');
                this.wsConnected = false;
                // Attempt to reconnect after 5 seconds
                setTimeout(() => this.connectWebSocket(), 5000);
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
            };
        },
        
        // Handle WebSocket messages
        handleWebSocketMessage(data) {
            console.log('WebSocket message:', data);
            
            switch (data.type) {
                case 'discovery_started':
                    this.showNotification('Discovery started', 'info');
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
                    this.recentDrifts.unshift(data.drift);
                    if (this.recentDrifts.length > 10) {
                        this.recentDrifts.pop();
                    }
                    this.updateDriftCharts();
                    this.showNotification(`New drift detected: ${data.drift.resource}`, 'warning');
                    break;
                    
                case 'stats_updated':
                    this.stats = data.stats;
                    this.updateCharts();
                    break;
            }
        },
        
        // Load dashboard data
        async loadDashboardData() {
            this.loading = true;
            
            try {
                // Load stats
                const statsResponse = await fetch('/api/v1/resources/stats');
                const statsData = await statsResponse.json();
                
                // Load recent drifts
                const driftResponse = await fetch('/api/v1/drift/report');
                const driftData = await driftResponse.json();
                
                // Use real statistics from API
                this.stats = {
                    totalResources: statsData.total || 0,
                    driftedResources: driftData.summary?.drifted || 0,
                    compliantResources: driftData.summary?.compliant || (statsData.total || 0),
                    activeProviders: Object.keys(statsData.by_provider || {}).length
                };
                
                // Use real drift data from API
                this.recentDrifts = [];
                if (driftData.drifts && Array.isArray(driftData.drifts)) {
                    this.recentDrifts = driftData.drifts.slice(0, 10); // Show only recent 10
                }
                
                this.updateCharts();
                
            } catch (error) {
                console.error('Error loading dashboard data:', error);
                this.showNotification('Failed to load dashboard data', 'error');
            } finally {
                this.loading = false;
            }
        },
        
        // Initialize charts
        initCharts() {
            // Get current theme colors from DaisyUI
            const computedStyle = getComputedStyle(document.documentElement);
            const primaryColor = computedStyle.getPropertyValue('--p');
            const secondaryColor = computedStyle.getPropertyValue('--s');
            const accentColor = computedStyle.getPropertyValue('--a');
            const neutralColor = computedStyle.getPropertyValue('--n');
            
            // Drift by Provider Chart
            const driftByProviderCtx = document.getElementById('driftByProviderChart');
            if (driftByProviderCtx) {
                this.charts.driftByProvider = new Chart(driftByProviderCtx, {
                    type: 'doughnut',
                    data: {
                        labels: ['AWS', 'Azure', 'GCP', 'DigitalOcean'],
                        datasets: [{
                            data: [0, 0, 0, 0], // Will be updated with real data
                            backgroundColor: [
                                '#ff9900',
                                '#0078d4',
                                '#4285f4',
                                '#0080ff'
                            ],
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
                                    color: computedStyle.getPropertyValue('--bc'),
                                    padding: 15,
                                    font: {
                                        size: 12
                                    }
                                }
                            }
                        }
                    }
                });
            }
            
            // Drift Severity Chart
            const driftSeverityCtx = document.getElementById('driftSeverityChart');
            if (driftSeverityCtx) {
                this.charts.driftSeverity = new Chart(driftSeverityCtx, {
                    type: 'bar',
                    data: {
                        labels: ['Critical', 'High', 'Medium', 'Low'],
                        datasets: [{
                            label: 'Drift Count',
                            data: [0, 0, 0, 0], // Will be updated with real data
                            backgroundColor: [
                                computedStyle.getPropertyValue('--er'),
                                computedStyle.getPropertyValue('--wa'),
                                computedStyle.getPropertyValue('--in'),
                                computedStyle.getPropertyValue('--su')
                            ],
                            borderWidth: 0
                        }]
                    },
                    options: {
                        responsive: true,
                        plugins: {
                            legend: {
                                display: false
                            }
                        },
                        scales: {
                            y: {
                                beginAtZero: true,
                                ticks: {
                                    color: computedStyle.getPropertyValue('--bc')
                                },
                                grid: {
                                    color: computedStyle.getPropertyValue('--b2'),
                                    borderColor: computedStyle.getPropertyValue('--b3')
                                }
                            },
                            x: {
                                ticks: {
                                    color: computedStyle.getPropertyValue('--bc')
                                },
                                grid: {
                                    display: false
                                }
                            }
                        }
                    }
                });
            }
        },
        
        // Update charts with real data
        async updateCharts() {
            // Fetch real drift report data
            try {
                const driftResponse = await fetch('/api/v1/drift/report');
                const driftData = await driftResponse.json();
                
                // Update drift by provider chart
                if (this.charts.driftByProvider && driftData.by_provider) {
                    const providers = Object.keys(driftData.by_provider);
                    const counts = Object.values(driftData.by_provider);
                    
                    this.charts.driftByProvider.data.labels = providers.map(p => p.toUpperCase());
                    this.charts.driftByProvider.data.datasets[0].data = counts;
                    this.charts.driftByProvider.update();
                }
                
                // Update drift severity chart
                if (this.charts.driftSeverity && driftData.by_severity) {
                    const severityData = [
                        driftData.by_severity.critical || 0,
                        driftData.by_severity.high || 0,
                        driftData.by_severity.medium || 0,
                        driftData.by_severity.low || 0
                    ];
                    
                    this.charts.driftSeverity.data.datasets[0].data = severityData;
                    this.charts.driftSeverity.update();
                }
            } catch (error) {
                console.error('Error updating charts:', error);
            }
        },
        
        // Start discovery
        async startDiscovery() {
            if (!this.discoveryForm.provider) {
                this.showNotification('Please select a provider', 'error');
                return;
            }
            
            this.discovering = true;
            
            const regions = this.discoveryForm.regions
                .split(',')
                .map(r => r.trim())
                .filter(r => r);
            
            try {
                const response = await fetch('/api/v1/discover', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        provider: this.discoveryForm.provider,
                        regions: regions.length > 0 ? regions : ['us-east-1']
                    })
                });
                
                const data = await response.json();
                
                this.discoveryJob = {
                    id: data.job_id,
                    status: 'running',
                    progress: 0,
                    message: 'Discovery started...'
                };
                
                // Poll for status
                this.pollDiscoveryStatus(data.job_id);
                
            } catch (error) {
                console.error('Error starting discovery:', error);
                this.showNotification('Failed to start discovery', 'error');
            } finally {
                this.discovering = false;
            }
        },
        
        // Poll discovery status
        async pollDiscoveryStatus(jobId) {
            const pollInterval = setInterval(async () => {
                try {
                    const response = await fetch(`/api/v1/discover/status?job_id=${jobId}`);
                    const status = await response.json();
                    
                    this.discoveryJob = status;
                    
                    if (status.status === 'completed' || status.status === 'failed') {
                        clearInterval(pollInterval);
                        
                        if (status.status === 'completed') {
                            this.loadDiscoveryResults(jobId);
                        } else {
                            this.showNotification('Discovery failed: ' + status.error, 'error');
                        }
                    }
                } catch (error) {
                    console.error('Error polling discovery status:', error);
                    clearInterval(pollInterval);
                }
            }, 2000);
        },
        
        // Load discovery results
        async loadDiscoveryResults(jobId) {
            try {
                const response = await fetch(`/api/v1/discover/results?job_id=${jobId}`);
                const data = await response.json();
                
                this.discoveryResults = data.resources || [];
                this.showNotification(`Discovered ${this.discoveryResults.length} resources`, 'success');
                
            } catch (error) {
                console.error('Error loading discovery results:', error);
                this.showNotification('Failed to load discovery results', 'error');
            }
        },
        
        // Refresh data
        refreshData() {
            if (this.currentView === 'dashboard') {
                this.loadDashboardData();
            }
        },
        
        // Show notification
        showNotification(message, type = 'info') {
            // Create DaisyUI toast notification
            const toast = document.createElement('div');
            toast.className = 'toast toast-top toast-end z-50';
            
            const alertTypes = {
                'info': 'alert-info',
                'success': 'alert-success',
                'warning': 'alert-warning',
                'error': 'alert-error'
            };
            
            const alertIcons = {
                'info': `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="stroke-current shrink-0 w-6 h-6"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>`,
                'success': `<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>`,
                'warning': `<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>`,
                'error': `<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>`
            };
            
            toast.innerHTML = `
                <div class="alert ${alertTypes[type] || 'alert-info'}">
                    ${alertIcons[type] || alertIcons['info']}
                    <span>${message}</span>
                </div>
            `;
            
            document.body.appendChild(toast);
            
            // Auto-dismiss after 5 seconds
            setTimeout(() => {
                toast.remove();
            }, 5000);
        },
        
        // Theme management
        setTheme(theme) {
            document.documentElement.setAttribute('data-theme', theme);
            localStorage.setItem('driftmgr-theme', theme);
            this.showNotification(`Theme changed to ${theme}`, 'info');
        },
        
        // Initialize theme from localStorage
        initTheme() {
            const savedTheme = localStorage.getItem('driftmgr-theme');
            if (savedTheme) {
                document.documentElement.setAttribute('data-theme', savedTheme);
            }
        }
    };
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    console.log('DriftMgr Web App loaded');
});