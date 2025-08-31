// DriftMgr Web Application
function driftmgrApp() {
    return {
        // Application state
        currentView: 'state',  // Default to state files view
        loading: false,
        wsConnected: false,
        ws: null,
        
        // Command Palette
        showCommandPalette: false,
        commandSearch: '',
        filteredCommands: [],
        availableCommands: [
            { id: 'state', label: 'Go to State Files', description: 'Terraform state management', icon: 'fas fa-file-code', shortcut: 'Alt+1', action: () => this.currentView = 'state' },
            { id: 'analysis', label: 'Go to Analysis', description: 'Analyze state files', icon: 'fas fa-chart-line', shortcut: 'Alt+2', action: () => this.currentView = 'analysis' },
            { id: 'drift', label: 'Go to Drift Detection', description: 'Check for configuration drift', icon: 'fas fa-exchange-alt', shortcut: 'Alt+3', action: () => this.currentView = 'drift' },
            { id: 'resources', label: 'Go to Resources', description: 'View cloud resources', icon: 'fas fa-cube', shortcut: 'Alt+4', action: () => this.currentView = 'resources' },
            { id: 'comparison', label: 'Go to Comparison', description: 'Compare state files', icon: 'fas fa-code-compare', shortcut: 'Alt+5', action: () => this.currentView = 'comparison' },
            { id: 'reports', label: 'Go to Reports', description: 'View analysis reports', icon: 'fas fa-file-alt', shortcut: 'Alt+6', action: () => this.currentView = 'reports' },
            { id: 'advanced', label: 'Go to Advanced', description: 'Advanced operations', icon: 'fas fa-cogs', shortcut: 'Alt+7', action: () => this.currentView = 'advanced' },
            { id: 'refresh', label: 'Refresh Data', description: 'Reload current view', icon: 'fas fa-sync-alt', shortcut: 'R', action: () => this.refreshData() },
            { id: 'discover', label: 'Run Discovery', description: 'Start resource discovery', icon: 'fas fa-radar', shortcut: 'D', action: () => this.startDiscovery() },
            { id: 'detect', label: 'Detect Drift', description: 'Run drift detection', icon: 'fas fa-crosshairs', shortcut: null, action: () => this.startDriftDetection() },
            { id: 'stats', label: 'Customize Stats', description: 'Configure dashboard statistics', icon: 'fas fa-chart-bar', shortcut: null, action: () => this.showStatsConfig = true },
            { id: 'export', label: 'Export Resources', description: 'Export resource data', icon: 'fas fa-download', shortcut: null, action: () => this.showExportModal = true },
            { id: 'help', label: 'Show Keyboard Shortcuts', description: 'View all keyboard shortcuts', icon: 'fas fa-keyboard', shortcut: '?', action: () => this.showKeyboardHelp() }
        ],
        quickFilter: 'all',
        
        // Loading bar state
        isLoading: false,
        loadingProgress: 0,
        loadingMessage: '',
        loadingTimeout: null,
        
        // Dashboard data
        stats: {
            totalResources: 0,
            driftedResources: 0,
            compliantResources: 0,
            activeProviders: 0,
            configuredProviders: [],
            unmanagedResources: 0,
            missingResources: 0,
            costEstimate: 0,
            securityIssues: 0,
            criticalDrifts: 0,
            remediableCount: 0,
            lastScanTime: null,
            complianceScore: 0
        },
        
        // Stats configuration
        showStatsConfig: false,
        availableStats: {
            totalResources: {
                title: 'Total Resources',
                description: 'Across all providers',
                icon: 'fa-server',
                colorClass: 'text-primary',
                category: 'Resources'
            },
            driftedResources: {
                title: 'Drifted Resources',
                description: 'Requires attention',
                icon: 'fa-exclamation-triangle',
                colorClass: 'text-error',
                category: 'Drift'
            },
            compliantResources: {
                title: 'Compliant',
                description: 'No issues detected',
                icon: 'fa-check-circle',
                colorClass: 'text-success',
                category: 'Compliance'
            },
            providers: {
                title: 'Providers',
                description: 'Configured/Active',
                icon: 'fa-cloud',
                colorClass: 'text-info',
                category: 'Infrastructure'
            },
            unmanagedResources: {
                title: 'Unmanaged',
                description: 'Out-of-band resources',
                icon: 'fa-question-circle',
                colorClass: 'text-warning',
                category: 'Resources'
            },
            missingResources: {
                title: 'Missing',
                description: 'In state, not in cloud',
                icon: 'fa-times-circle',
                colorClass: 'text-error',
                category: 'Resources'
            },
            costEstimate: {
                title: 'Est. Monthly Cost',
                description: 'Infrastructure spend',
                icon: 'fa-dollar-sign',
                colorClass: 'text-accent',
                category: 'Cost'
            },
            securityIssues: {
                title: 'Security Issues',
                description: 'Security-related drifts',
                icon: 'fa-shield-alt',
                colorClass: 'text-error',
                category: 'Security'
            },
            criticalDrifts: {
                title: 'Critical Drifts',
                description: 'High severity issues',
                icon: 'fa-fire',
                colorClass: 'text-error',
                category: 'Drift'
            },
            remediableCount: {
                title: 'Auto-Remediable',
                description: 'Can be auto-fixed',
                icon: 'fa-tools',
                colorClass: 'text-warning',
                category: 'Remediation'
            },
            complianceScore: {
                title: 'Compliance Score',
                description: 'Overall compliance',
                icon: 'fa-chart-line',
                colorClass: 'text-success',
                category: 'Compliance'
            },
            lastScanTime: {
                title: 'Last Scan',
                description: 'Time since last scan',
                icon: 'fa-clock',
                colorClass: 'text-base-content',
                category: 'System'
            }
        },
        
        // User's stats configuration
        statsConfig: {},
        
        recentDrifts: [],
        
        // Discovery data
        discovering: false,
        discoveryForm: {
            provider: '',
            regions: '',
            autoRemediate: false
        },
        discoveryJob: null,
        discoveryResults: [],
        
        // Resources data
        resources: [],
        filteredResources: [],
        resourcesLoading: false,
        resourceFilters: {
            provider: '',
            type: '',
            region: '',
            search: ''
        },
        availableProviders: [],
        availableResourceTypes: [],
        availableRegions: [],
        selectedResources: [],
        
        // Drift Detection data
        driftDetecting: false,
        driftReport: {
            summary: {
                total: 0,
                remediable: 0,
                security: 0,
                compliant: 0
            },
            drifts: []
        },
        driftOptions: {
            provider: '',
            resourceType: ''
        },
        filteredDrifts: [],
        selectedDrift: null,
        selectedDrifts: [],
        
        // Terminal
        showTerminal: false,
        terminalCommand: '',
        terminalStatus: 'idle',
        terminalOutput: [],
        terminalJobId: null,
        
        // Export/Import
        showExportModal: false,
        showImportModal: false,
        exportFormat: 'json',
        exportFilteredOnly: false,
        selectedImportFile: null,
        
        // Audit
        auditLogs: [],
        auditFilters: {
            severity: '',
            service: '',
            user: '',
            startDate: '',
            endDate: ''
        },
        
        // State Management
        stateFiles: [],
        stateDiscovering: false,
        stateImporting: false,
        selectedState: null,
        selectedStates: [],
        stateResources: [],
        totalStateResources: 0,
        localStatesCount: 0,
        remoteStatesCount: 0,
        cloudBackendsCount: 0,
        s3BackendsCount: 0,
        azureBackendsCount: 0,
        gcsBackendsCount: 0,
        importedStatesCount: 0,
        importedStates: new Set(),
        stateFilePath: '',
        remoteBackendType: '',
        
        // State Analysis
        stateAnalysis: {
            totalFiles: 0,
            totalResources: 0,
            uniqueProviders: 0,
            terraformVersions: [],
            modules: [],
            resourceTypes: {},
            providerDistribution: {}
        },
        
        // State Comparison
        comparison: {
            file1: '',
            file2: '',
            results: null
        },
        comparisonTab: 'added',
        
        // Reports
        generatedReports: [],
        reportGenerating: false,
        
        // State Content Viewer
        stateViewMode: 'json',
        stateContent: null,
        stateContentSearch: '',
        highlightedStateContent: '',
        stateTreeView: '',
        stateDiffView: '',
        stateFileSize: 0,
        filteredStateResources: [],
        
        // Credentials and Accounts
        credentialsStatus: [],
        multiAccountProfiles: {},
        accounts: [],
        currentAccount: null,
        environment: 'production',
        activeCredentials: [],
        
        // Advanced Operations
        advancedTab: 'batch',
        batchOperation: {
            type: '',
            filter: '',
            dryRun: true,
            force: false,
            includeDeps: false,
            results: null,
            success: false,
            message: ''
        },
        verification: {
            enhanced: false,
            validateOnly: false,
            compliance: true,
            costAnalysis: false,
            provider: '',
            results: null
        },
        currentConfig: {
            providers: {},
            discovery: {},
            remediation: {},
            export: {}
        },
        terminalInput: '',
        commandHistory: [],
        workspaces: ['default', 'production', 'staging', 'development'],
        selectedWorkspace: '',
        selectedRegions: [],
        availableRegions: {
            aws: ['us-east-1', 'us-west-2', 'eu-west-1', 'ap-southeast-1'],
            azure: ['eastus', 'westus', 'northeurope', 'westeurope'],
            gcp: ['us-central1', 'us-east1', 'europe-west1', 'asia-northeast1'],
            digitalocean: ['nyc1', 'sfo2', 'ams3', 'sgp1']
        },
        
        // Perspective data
        perspectiveAnalyzing: false,
        perspectiveReport: {
            summary: {},
            categories: {},
            unmanaged_resources: [],
            recommendations: []
        },
        perspectiveOptions: {
            provider: '',
            stateFile: '',
            regions: []
        },
        selectedUnmanagedResources: [],
        
        // Charts
        charts: {},
        
        // Initialize application
        async init() {
            console.log('Initializing DriftMgr Web App');
            this.connectWebSocket();
            
            // Load stats configuration from localStorage
            this.loadStatsConfig();
            
            // Initialize keyboard shortcuts
            this.initKeyboardShortcuts();
            
            // Initialize command palette
            this.initCommandPalette();
            
            // Load cached data immediately for instant UI population
            await this.loadCachedData();
            
            // Load data with proper error handling
            try {
                await Promise.all([
                    this.loadDashboardData(),
                    this.loadCredentialsStatus(),
                    this.loadAccounts(),
                    this.loadDriftReport()
                ]);
                console.log('All data loaded successfully');
            } catch (error) {
                console.error('Error loading initial data:', error);
            }
            
            this.initCharts();
            
            // Watch for view changes
            this.$watch('currentView', (newView) => {
                this.onViewChange(newView);
                if (newView === 'perspective') {
                    this.loadPerspectiveReport();
                }
            });
            
            // Set up periodic refresh
            setInterval(() => {
                if (this.currentView === 'dashboard') {
                    this.refreshData();
                }
            }, 30000); // Refresh every 30 seconds
            
            // Force a refresh after 1 second to ensure data is displayed
            setTimeout(() => {
                this.refreshData();
            }, 1000);
        },
        
        // Load cached data immediately on startup
        async loadCachedData() {
            console.log('Loading cached data for immediate UI population...');
            try {
                // Fetch cached discovery results
                const cachedResponse = await fetch('/api/v1/discovery/cached');
                if (cachedResponse.ok) {
                    const cachedData = await cachedResponse.json();
                    if (cachedData.resources && cachedData.resources.length > 0) {
                        console.log(`Loaded ${cachedData.count} cached resources`);
                        
                        // Populate resources immediately
                        this.resources = cachedData.resources;
                        this.filteredResources = cachedData.resources;
                        this.discoveryResults = cachedData.resources;
                        
                        // Extract filter options from cached data
                        this.extractFilterOptions();
                        
                        // Update stats based on cached data
                        this.updateStatsFromResources(cachedData.resources);
                        
                        // Show notification that cached data is loaded
                        this.showNotification(`Loaded ${cachedData.count} resources from cache`, 'success');
                    } else {
                        console.log('No cached resources available');
                    }
                }
            } catch (error) {
                console.error('Error loading cached data:', error);
                // Don't show error notification as this is non-critical
            }
        },
        
        // Update statistics from resources
        updateStatsFromResources(resources) {
            if (!resources || !Array.isArray(resources)) return;
            
            // Count resources by provider
            const byProvider = {};
            const providers = new Set();
            
            resources.forEach(resource => {
                const provider = resource.provider || 'unknown';
                byProvider[provider] = (byProvider[provider] || 0) + 1;
                providers.add(provider);
            });
            
            // Update stats
            this.stats.totalResources = resources.length;
            this.stats.activeProviders = providers.size;
            
            // Update provider counts for charts
            if (this.charts.driftByProvider && Object.keys(byProvider).length > 0) {
                this.charts.driftByProvider.data.labels = Object.keys(byProvider).map(p => p.toUpperCase());
                this.charts.driftByProvider.data.datasets[0].data = Object.values(byProvider);
                this.charts.driftByProvider.update();
            }
        },
        
        // Loading bar methods
        startLoading(message = 'Loading...', estimatedTime = 3000) {
            this.isLoading = true;
            this.loadingProgress = 0;
            this.loadingMessage = message;
            
            // Clear any existing timeout
            if (this.loadingTimeout) {
                clearInterval(this.loadingTimeout);
            }
            
            // Animate progress bar
            const increment = 100 / (estimatedTime / 100);
            this.loadingTimeout = setInterval(() => {
                if (this.loadingProgress < 90) {
                    this.loadingProgress = Math.min(90, this.loadingProgress + increment);
                }
            }, 100);
        },
        
        updateLoadingProgress(progress, message = null) {
            this.loadingProgress = Math.min(100, progress);
            if (message) {
                this.loadingMessage = message;
            }
        },
        
        stopLoading() {
            // Complete the progress bar
            this.loadingProgress = 100;
            
            // Clear the interval
            if (this.loadingTimeout) {
                clearInterval(this.loadingTimeout);
                this.loadingTimeout = null;
            }
            
            // Hide after a short delay
            setTimeout(() => {
                this.isLoading = false;
                this.loadingProgress = 0;
                this.loadingMessage = '';
            }, 300);
        },
        
        // Handle view changes
        onViewChange(view) {
            console.log('View changed to:', view);
            
            switch(view) {
                case 'resources':
                    this.loadResources();
                    break;
                case 'audit':
                    this.loadAuditLogs();
                    break;
                case 'state':
                    this.autoDiscoverStateFiles();
                    break;
                case 'discovery':
                    // Discovery view handles its own loading
                    break;
                case 'drift':
                    this.loadDriftReport();
                    break;
                case 'dashboard':
                    this.refreshData();
                    break;
                case 'advanced':
                    // Advanced view handles its own loading
                    break;
            }
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
                    
                case 'terminal_output':
                    this.appendTerminalOutput(data.text, data.output_type || 'info');
                    break;
                    
                case 'terminal_status':
                    this.terminalStatus = data.status;
                    if (data.status === 'completed' || data.status === 'failed') {
                        this.appendTerminalOutput(`\n[Process ${data.status}]`, data.status === 'completed' ? 'success' : 'error');
                    }
                    break;
            }
        },
        
        // Load dashboard data
        async loadDashboardData() {
            this.loading = true;
            this.startLoading('Loading dashboard data...', 2000);
            
            try {
                // Try enriched stats first for comprehensive data
                this.updateLoadingProgress(15, 'Fetching enriched statistics...');
                let statsData;
                try {
                    const enrichedStatsResponse = await fetch('/api/enriched/stats');
                    if (enrichedStatsResponse.ok) {
                        statsData = await enrichedStatsResponse.json();
                        console.log('Using enriched stats');
                    }
                } catch (e) {
                    console.log('Enriched stats not available');
                }
                
                // Fallback to regular stats
                if (!statsData) {
                    this.updateLoadingProgress(30, 'Fetching resource statistics...');
                    const statsResponse = await fetch('/api/v1/resources/stats');
                    statsData = await statsResponse.json();
                }
                
                // Load drift report
                this.updateLoadingProgress(60, 'Loading drift report...');
                const driftResponse = await fetch('/api/v1/drift/report');
                const driftData = await driftResponse.json();
                
                // Use real statistics from API
                this.updateLoadingProgress(80, 'Processing data...');
                const totalResources = statsData.total || 0;
                const driftedResources = driftData.summary?.drifted || 0;
                const compliantResources = driftData.summary?.compliant || Math.max(0, totalResources - driftedResources);
                const unmanagedResources = driftData.summary?.unmanaged || 0;
                const missingResources = driftData.summary?.missing || 0;
                
                // Calculate compliance score
                let complianceScore = 0;
                if (totalResources > 0) {
                    complianceScore = Math.round((compliantResources / totalResources) * 100);
                }
                
                // Calculate estimated costs based on actual cloud provider pricing
                const costEstimate = await this.calculateRealCosts(statsData);
                
                this.stats = {
                    totalResources: totalResources,
                    driftedResources: driftedResources,
                    compliantResources: compliantResources,
                    activeProviders: 4, // Total number of provider types
                    configuredProviders: statsData.configured_providers || [],
                    unmanagedResources: unmanagedResources,
                    missingResources: missingResources,
                    costEstimate: costEstimate,
                    securityIssues: driftData.summary?.security || 0,
                    criticalDrifts: driftData.summary?.critical || 0,
                    remediableCount: driftData.summary?.remediable || 0,
                    lastScanTime: new Date().toISOString(),
                    complianceScore: complianceScore
                };
                
                // Store drift data for charts and recent drifts
                this.driftReport = driftData;
                
                // Update recent drifts if available
                this.recentDrifts = [];
                if (driftData.drifts && Array.isArray(driftData.drifts)) {
                    this.recentDrifts = driftData.drifts.slice(0, 10);
                }
                
                this.updateLoadingProgress(90, 'Updating charts...');
                this.updateCharts();
                
            } catch (error) {
                console.error('Error loading dashboard data:', error);
                this.showNotification('Failed to load dashboard data', 'error');
            } finally {
                this.loading = false;
                this.stopLoading();
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
            try {
                // Get stats for resources by provider
                const statsResponse = await fetch('/api/v1/resources/stats');
                const statsData = await statsResponse.json();
                
                // Update drift by provider chart (showing resource distribution)
                if (this.charts.driftByProvider && statsData.by_provider) {
                    const providers = Object.keys(statsData.by_provider).filter(p => statsData.by_provider[p] > 0);
                    const counts = providers.map(p => statsData.by_provider[p]);
                    
                    // If no resources found, show configured providers with 0 counts
                    if (providers.length === 0 && statsData.configured_providers) {
                        this.charts.driftByProvider.data.labels = statsData.configured_providers.map(p => p.toUpperCase());
                        this.charts.driftByProvider.data.datasets[0].data = statsData.configured_providers.map(() => 0);
                    } else {
                        this.charts.driftByProvider.data.labels = providers.map(p => p.toUpperCase());
                        this.charts.driftByProvider.data.datasets[0].data = counts;
                    }
                    this.charts.driftByProvider.update();
                }
                
                // Use stored drift report data for severity chart
                if (this.charts.driftSeverity && this.driftReport?.by_severity) {
                    const severityData = [
                        this.driftReport.by_severity.critical || 0,
                        this.driftReport.by_severity.high || 0,
                        this.driftReport.by_severity.medium || 0,
                        this.driftReport.by_severity.low || 0
                    ];
                    
                    this.charts.driftSeverity.data.datasets[0].data = severityData;
                    this.charts.driftSeverity.update();
                }
            } catch (error) {
                console.error('Error updating charts:', error);
            }
        },
        
        // [Removed duplicate startDiscovery - using the more complete API-based implementation]
        
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
        
        // Get provider icon class
        getProviderIcon(provider) {
            const icons = {
                'aws': 'fab fa-aws text-orange-500',
                'azure': 'fab fa-microsoft text-blue-500',
                'gcp': 'fab fa-google text-blue-600',
                'digitalocean': 'fab fa-digital-ocean text-blue-400'
            };
            return icons[provider.toLowerCase()] || 'fas fa-cloud text-gray-500';
        },
        
        // Terminal functions
        openTerminal(command, jobId = null) {
            this.showTerminal = true;
            this.terminalCommand = command;
            this.terminalStatus = 'running';
            this.terminalOutput = [];
            this.terminalJobId = jobId;
            
            // Add initial output
            this.appendTerminalOutput(`$ ${command}`, 'info');
            this.appendTerminalOutput('Starting command execution...', 'debug');
        },
        
        appendTerminalOutput(text, type = 'info') {
            const timestamp = new Date().toLocaleTimeString();
            this.terminalOutput.push({
                id: Date.now() + Math.random(),
                timestamp: timestamp,
                text: this.escapeHtml(text).replace(/\n/g, '<br>'),
                type: type
            });
            
            // Auto-scroll to bottom
            this.$nextTick(() => {
                const terminal = document.getElementById('terminal-output');
                if (terminal) {
                    terminal.scrollTop = terminal.scrollHeight;
                }
            });
        },
        
        clearTerminal() {
            this.terminalOutput = [];
        },
        
        copyTerminalOutput() {
            const text = this.terminalOutput
                .map(line => `[${line.timestamp}] ${line.text.replace(/<br>/g, '\n')}`)
                .join('\n');
            
            navigator.clipboard.writeText(text).then(() => {
                this.showNotification('Output copied to clipboard', 'success');
            });
        },
        
        downloadTerminalLog() {
            const text = this.terminalOutput
                .map(line => `[${line.timestamp}] [${line.type}] ${line.text.replace(/<br>/g, '\n')}`)
                .join('\n');
            
            const blob = new Blob([text], { type: 'text/plain' });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `terminal-${this.terminalCommand.replace(/[^a-z0-9]/gi, '_')}-${Date.now()}.log`;
            a.click();
            window.URL.revokeObjectURL(url);
        },
        
        escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        },
        
        // Run command with terminal output
        async runCommandWithTerminal(command, endpoint, method = 'POST', body = null) {
            this.openTerminal(command);
            
            try {
                const options = {
                    method: method,
                    headers: {
                        'Content-Type': 'application/json',
                    }
                };
                
                if (body) {
                    options.body = JSON.stringify(body);
                }
                
                // Send command request
                const response = await fetch(endpoint, options);
                const result = await response.json();
                
                if (result.job_id) {
                    this.terminalJobId = result.job_id;
                    // WebSocket will handle output updates
                }
                
                return result;
            } catch (error) {
                this.terminalStatus = 'failed';
                this.appendTerminalOutput(`Error: ${error.message}`, 'error');
                throw error;
            }
        },
        
        // State Management Functions
        // Auto-discover state files when state view is opened
        async autoDiscoverStateFiles() {
            // Only auto-discover if no state files are loaded
            if (this.stateFiles.length === 0 && !this.stateDiscovering) {
                console.log('Auto-discovering state files from connected accounts...');
                await this.discoverStateFiles();
            } else if (this.stateFiles.length > 0) {
                console.log('State files already loaded:', this.stateFiles.length);
            }
        },
        
        async discoverStateFiles() {
            this.stateDiscovering = true;
            this.showNotification('Discovering state files from local and cloud storage...', 'info');
            
            try {
                const response = await fetch('/api/v1/state/discover');
                const data = await response.json();
                
                if (data.states) {
                    this.stateFiles = data.states;
                    this.calculateStateStats();
                    
                    // Group by location type
                    const localCount = data.states.filter(s => !s.is_remote).length;
                    const remoteCount = data.states.filter(s => s.is_remote).length;
                    
                    let message = `Discovered ${data.count} state files`;
                    if (localCount > 0 && remoteCount > 0) {
                        message += ` (${localCount} local, ${remoteCount} remote)`;
                    }
                    
                    this.showNotification(message, 'success');
                }
            } catch (error) {
                console.error('Error discovering state files:', error);
                this.showNotification('Failed to discover state files', 'error');
            } finally {
                this.stateDiscovering = false;
            }
        },
        
        async loadStateFiles() {
            try {
                const response = await fetch('/api/v1/state/list');
                const data = await response.json();
                
                if (data.states) {
                    this.stateFiles = data.states;
                    this.calculateStateStats();
                }
            } catch (error) {
                console.error('Error loading state files:', error);
                this.showNotification('Failed to load state files', 'error');
            }
        },
        
        calculateStateStats() {
            // Basic counts
            this.totalStateResources = this.stateFiles.reduce((total, state) => 
                total + (state.resource_count || 0), 0);
            this.localStatesCount = this.stateFiles.filter(state => !state.is_remote).length;
            this.remoteStatesCount = this.stateFiles.filter(state => state.is_remote).length;
            
            // Cloud backend counts
            this.s3BackendsCount = this.stateFiles.filter(state => state.backend === 's3').length;
            this.azureBackendsCount = this.stateFiles.filter(state => state.backend === 'azurerm').length;
            this.gcsBackendsCount = this.stateFiles.filter(state => state.backend === 'gcs').length;
            this.cloudBackendsCount = this.s3BackendsCount + this.azureBackendsCount + this.gcsBackendsCount;
            
            // Imported states count
            this.importedStatesCount = this.importedStates.size;
        },
        
        // Import state files into cache for analysis
        async importStateFiles() {
            if (this.stateFiles.length === 0) {
                this.showNotification('No state files to import', 'warning');
                return;
            }
            
            this.stateImporting = true;
            const selectedStates = this.stateFiles.filter(state => !this.importedStates.has(state.path));
            
            if (selectedStates.length === 0) {
                this.showNotification('All state files are already imported', 'info');
                this.stateImporting = false;
                return;
            }
            
            try {
                // Simulate importing state files
                // In a real implementation, this would send the state files to the backend
                const response = await fetch('/api/v1/state/import', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        state_files: selectedStates.map(s => ({
                            path: s.path,
                            backend: s.backend,
                            provider: s.provider
                        }))
                    })
                });
                
                if (response.ok) {
                    const data = await response.json();
                    
                    // Mark states as imported
                    selectedStates.forEach(state => {
                        this.importedStates.add(state.path);
                    });
                    
                    this.calculateStateStats();
                    this.showNotification(`Successfully imported ${selectedStates.length} state files`, 'success');
                    
                    // Trigger resource discovery from imported state files
                    if (data.discovered_resources) {
                        this.resources = [...this.resources, ...data.discovered_resources];
                        this.extractFilterOptions();
                        this.filterResources();
                    }
                } else {
                    // Fallback: mark as imported even if API fails (for demo)
                    selectedStates.forEach(state => {
                        this.importedStates.add(state.path);
                    });
                    this.calculateStateStats();
                    this.showNotification(`Imported ${selectedStates.length} state files (offline mode)`, 'success');
                }
            } catch (error) {
                console.error('Error importing state files:', error);
                // Still mark as imported for demo purposes
                selectedStates.forEach(state => {
                    this.importedStates.add(state.path);
                });
                this.calculateStateStats();
                this.showNotification(`Imported ${selectedStates.length} state files locally`, 'info');
            } finally {
                this.stateImporting = false;
            }
        },
        
        async viewStateDetails(state) {
            this.selectedState = state;
            this.stateResources = [];
            
            try {
                const response = await fetch(`/api/v1/state/details?path=${encodeURIComponent(state.path)}`);
                const data = await response.json();
                
                if (data.state && data.state.Resources) {
                    this.stateResources = data.state.Resources;
                }
                
                // Show modal
                document.getElementById('stateDetailsModal').showModal();
            } catch (error) {
                console.error('Error loading state details:', error);
                this.showNotification('Failed to load state details', 'error');
            }
        },
        
        async analyzeState(state) {
            try {
                const response = await fetch('/api/v1/state/analyze', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ path: state.path })
                });
                
                const data = await response.json();
                this.showNotification('State analysis complete', 'success');
                // Could show analysis results in a modal
            } catch (error) {
                console.error('Error analyzing state:', error);
                this.showNotification('Failed to analyze state', 'error');
            }
        },
        
        async detectDrift(state) {
            try {
                const response = await fetch('/api/v1/drift/detect', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ 
                        state_file: state.path,
                        provider: state.provider?.toLowerCase() 
                    })
                });
                
                const data = await response.json();
                this.showNotification('Drift detection started', 'info');
                // Navigate to drift view
                this.currentView = 'drift';
            } catch (error) {
                console.error('Error detecting drift:', error);
                this.showNotification('Failed to start drift detection', 'error');
            }
        },
        
        truncatePath(path) {
            if (!path) return '';
            const maxLength = 50;
            if (path.length <= maxLength) return path;
            
            const parts = path.split(/[\\\/]/);
            if (parts.length <= 3) return path;
            
            return '...' + path.slice(-maxLength);
        },
        
        formatFileSize(bytes) {
            if (!bytes) return '0 B';
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(1024));
            return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
        },
        
        formatDate(dateString) {
            if (!dateString) return 'Unknown';
            const date = new Date(dateString);
            return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
        },
        
        // Stats Configuration Functions
        loadStatsConfig() {
            const saved = localStorage.getItem('driftmgr_stats_config');
            if (saved) {
                try {
                    this.statsConfig = JSON.parse(saved);
                } catch (e) {
                    console.error('Failed to load stats config:', e);
                    this.selectDefaultStats();
                }
            } else {
                // First time - use default stats
                this.selectDefaultStats();
            }
        },
        
        saveStatsConfig() {
            localStorage.setItem('driftmgr_stats_config', JSON.stringify(this.statsConfig));
            this.showNotification('Dashboard statistics configuration saved', 'success');
            this.showStatsConfig = false;
        },
        
        selectDefaultStats() {
            this.statsConfig = {
                totalResources: true,
                driftedResources: true,
                compliantResources: true,
                providers: true,
                unmanagedResources: false,
                missingResources: false,
                costEstimate: false,
                securityIssues: false,
                criticalDrifts: false,
                remediableCount: false,
                complianceScore: true,
                lastScanTime: false
            };
        },
        
        selectAllStats() {
            Object.keys(this.availableStats).forEach(key => {
                this.statsConfig[key] = true;
            });
        },
        
        deselectAllStats() {
            Object.keys(this.availableStats).forEach(key => {
                this.statsConfig[key] = false;
            });
        },
        
        getEnabledStats() {
            return Object.keys(this.statsConfig).filter(key => this.statsConfig[key]);
        },
        
        // Keyboard Shortcuts
        initKeyboardShortcuts() {
            document.addEventListener('keydown', (e) => {
                // Command Palette (Ctrl+K or Cmd+K)
                if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                    e.preventDefault();
                    this.showCommandPalette = !this.showCommandPalette;
                    if (this.showCommandPalette) {
                        setTimeout(() => {
                            const input = document.querySelector('[x-ref="commandInput"]');
                            if (input) input.focus();
                        }, 100);
                    }
                }
                
                // Navigation shortcuts (Alt+1 through Alt+7)
                if (e.altKey && e.key >= '1' && e.key <= '7') {
                    e.preventDefault();
                    const views = ['dashboard', 'discovery', 'drift', 'perspective', 'resources', 'state', 'audit'];
                    const index = parseInt(e.key) - 1;
                    if (views[index]) {
                        this.currentView = views[index];
                    }
                }
                
                // Refresh (R key when not in input)
                if (e.key === 'r' && !this.isInputFocused()) {
                    e.preventDefault();
                    this.refreshData();
                }
                
                // Discovery (D key when not in input)
                if (e.key === 'd' && !this.isInputFocused()) {
                    e.preventDefault();
                    this.currentView = 'discovery';
                    setTimeout(() => this.startDiscovery(), 100);
                }
                
                // Help (? key)
                if (e.key === '?' && !this.isInputFocused()) {
                    e.preventDefault();
                    this.showKeyboardHelp();
                }
                
                // Escape to close modals/palettes
                if (e.key === 'Escape') {
                    this.showCommandPalette = false;
                    this.showExportModal = false;
                    this.showImportModal = false;
                    this.showStatsConfig = false;
                    this.showTerminal = false;
                }
            });
        },
        
        isInputFocused() {
            const activeElement = document.activeElement;
            return activeElement && (
                activeElement.tagName === 'INPUT' ||
                activeElement.tagName === 'TEXTAREA' ||
                activeElement.tagName === 'SELECT' ||
                activeElement.contentEditable === 'true'
            );
        },
        
        // Command Palette
        initCommandPalette() {
            this.filteredCommands = [...this.availableCommands];
        },
        
        filterCommands() {
            const search = this.commandSearch.toLowerCase();
            if (!search) {
                this.filteredCommands = [...this.availableCommands];
            } else {
                this.filteredCommands = this.availableCommands.filter(cmd =>
                    cmd.label.toLowerCase().includes(search) ||
                    cmd.description.toLowerCase().includes(search)
                );
            }
        },
        
        executeCommand(cmd) {
            this.showCommandPalette = false;
            this.commandSearch = '';
            if (cmd.action) {
                cmd.action();
            }
        },
        
        showKeyboardHelp() {
            const shortcuts = [
                { keys: 'Ctrl+K', description: 'Open command palette' },
                { keys: 'Alt+1-7', description: 'Navigate between views' },
                { keys: 'R', description: 'Refresh current data' },
                { keys: 'D', description: 'Start discovery' },
                { keys: '?', description: 'Show this help' },
                { keys: 'Escape', description: 'Close modals/dialogs' }
            ];
            
            let helpText = 'Keyboard Shortcuts:\n\n';
            shortcuts.forEach(s => {
                helpText += `${s.keys.padEnd(12)} - ${s.description}\n`;
            });
            
            alert(helpText);
        },
        
        // Quick Filters
        applyQuickFilter(filter) {
            this.quickFilter = filter;
            
            switch(filter) {
                case 'all':
                    // Show everything
                    this.resourceFilters = { provider: '', type: '', region: '', search: '' };
                    break;
                case 'issues':
                    // Show only resources with issues
                    this.currentView = 'drift';
                    break;
                case 'compliant':
                    // Show only compliant resources
                    this.currentView = 'resources';
                    // Filter for compliant resources would be applied here
                    break;
                case 'recent':
                    // Show recent changes
                    this.currentView = 'audit';
                    break;
            }
            
            // Refresh the current view
            this.onViewChange(this.currentView);
        },
        
        getStatValue(key) {
            switch(key) {
                case 'totalResources':
                    return this.stats.totalResources || 0;
                case 'driftedResources':
                    return this.stats.driftedResources || 0;
                case 'compliantResources':
                    return this.stats.compliantResources || 0;
                case 'providers':
                    return `${this.stats.configuredProviders.length || 0}/${this.stats.activeProviders || 0}`;
                case 'unmanagedResources':
                    return this.stats.unmanagedResources || 0;
                case 'missingResources':
                    return this.stats.missingResources || 0;
                case 'costEstimate':
                    const cost = this.stats.costEstimate || 0;
                    return cost > 0 ? `$${cost.toFixed(2)}` : '$0';
                case 'securityIssues':
                    return this.stats.securityIssues || 0;
                case 'criticalDrifts':
                    return this.stats.criticalDrifts || 0;
                case 'remediableCount':
                    return this.stats.remediableCount || 0;
                case 'complianceScore':
                    const score = this.stats.complianceScore || 0;
                    return `${score}%`;
                case 'lastScanTime':
                    if (this.stats.lastScanTime) {
                        const diff = Date.now() - new Date(this.stats.lastScanTime).getTime();
                        const mins = Math.floor(diff / 60000);
                        if (mins < 60) return `${mins}m ago`;
                        const hours = Math.floor(mins / 60);
                        if (hours < 24) return `${hours}h ago`;
                        return `${Math.floor(hours / 24)}d ago`;
                    }
                    return 'Never';
                default:
                    return 0;
            }
        },
        
        getProviderBadgeClass(provider) {
            const providerClasses = {
                'AWS': 'badge-warning',
                'aws': 'badge-warning',
                'Azure': 'badge-info',
                'azure': 'badge-info',
                'GCP': 'badge-success',
                'gcp': 'badge-success',
                'DigitalOcean': 'badge-primary',
                'digitalocean': 'badge-primary',
                'Terraform Cloud': 'badge-secondary',
            };
            return providerClasses[provider] || 'badge-ghost';
        },
        
        getProviderIconClass(provider) {
            const iconClasses = {
                'AWS': 'text-warning',
                'Azure': 'text-info',
                'GCP': 'text-success',
                'DigitalOcean': 'text-primary',
            };
            return iconClasses[provider] || 'text-base-content';
        },
        
        // Load credentials status
        async loadCredentialsStatus() {
            try {
                // Provider icon mapping
                const providerIcons = {
                    'aws': 'fab fa-aws',
                    'azure': 'fab fa-microsoft',
                    'gcp': 'fab fa-google',
                    'digitalocean': 'fas fa-water'
                };
                
                // Get credentials from the new status endpoint
                const response = await fetch('/api/v1/credentials/status');
                
                if (response.ok) {
                    const data = await response.json();
                    this.credentialsStatus = data.credentials || [];
                    
                    // Convert to activeCredentials format with icons
                    this.activeCredentials = this.credentialsStatus
                        .filter(c => c.status === 'configured')
                        .map(c => ({
                            provider: c.provider.toLowerCase(),
                            icon: providerIcons[c.provider.toLowerCase()] || 'fas fa-cloud',
                            active: c.valid === true,
                            details: c.details || 'Configured',
                            valid: c.valid,
                            status: c.status
                        }));
                } else {
                    // Fallback to detect endpoint if status endpoint doesn't exist
                    const detectResponse = await fetch('/api/v1/credentials/detect');
                    if (detectResponse.ok) {
                        const detectData = await detectResponse.json();
                        this.credentialsStatus = detectData.credentials || [];
                        // Convert to activeCredentials format
                        this.activeCredentials = this.credentialsStatus
                            .filter(c => c.status === 'configured')
                            .map(c => ({
                                provider: c.provider.toLowerCase(),
                                icon: providerIcons[c.provider.toLowerCase()] || 'fas fa-cloud',
                                active: true,
                                details: c.details?.method || 'Configured',
                                valid: true,
                                status: c.status
                            }));
                    }
                }
                
                // Extract configured providers for stats
                this.stats.configuredProviders = this.credentialsStatus
                    .filter(c => c.status === 'configured')
                    .map(c => c.provider.toLowerCase());
                    
            } catch (error) {
                console.error('Error loading credentials:', error);
                // Fallback to checking providers individually
                this.credentialsStatus = await this.checkAllProviders();
            }
        },
        
        // Check all providers individually
        async checkAllProviders() {
            const providers = ['aws', 'azure', 'gcp', 'digitalocean'];
            const credStatus = [];
            
            for (const provider of providers) {
                try {
                    const response = await fetch(`/api/v1/providers/${provider}/credentials`);
                    const data = await response.json();
                    
                    credStatus.push({
                        provider: provider.toUpperCase(),
                        status: data.configured ? 'configured' : 'not configured',
                        valid: data.valid || false,
                        details: {
                            method: data.configured ? 'Environment/Config' : null
                        }
                    });
                } catch (error) {
                    credStatus.push({
                        provider: provider.toUpperCase(),
                        status: 'error',
                        valid: false,
                        details: { error: error.message }
                    });
                }
            }
            
            return credStatus;
        },
        
        // Refresh credentials
        async refreshCredentials() {
            await this.loadCredentialsStatus();
            this.showNotification('Credentials status refreshed', 'success');
        },
        
        // Load accounts
        async loadAccounts() {
            try {
                const response = await fetch('/api/v1/accounts');
                const data = await response.json();
                
                if (data.accounts) {
                    this.accounts = data.accounts;
                    // Set first account as current if none selected
                    if (!this.currentAccount && this.accounts.length > 0) {
                        this.currentAccount = this.accounts[0];
                    }
                }
                
                // Also try to get multi-account profiles
                await this.loadMultiAccountProfiles();
                
            } catch (error) {
                console.error('Error loading accounts:', error);
            }
        },
        
        // Load multi-account profiles
        async loadMultiAccountProfiles() {
            try {
                const response = await fetch('/api/v1/accounts/profiles');
                if (response.ok) {
                    const data = await response.json();
                    this.multiAccountProfiles = data.profiles || {};
                }
            } catch (error) {
                // Endpoint might not exist, use empty object
                this.multiAccountProfiles = {};
            }
        },
        
        // Switch account
        async switchAccount(account) {
            this.currentAccount = account;
            
            try {
                const response = await fetch('/api/v1/accounts/use', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        provider: account.provider,
                        account_id: account.id
                    })
                });
                
                if (response.ok) {
                    this.showNotification(`Switched to ${account.name}`, 'success');
                    // Refresh dashboard data for new account
                    await this.loadDashboardData();
                }
            } catch (error) {
                console.error('Error switching account:', error);
                this.showNotification('Failed to switch account', 'error');
            }
        },
        
        // Set environment
        setEnvironment(env) {
            this.environment = env;
            localStorage.setItem('driftmgr_environment', env);
            this.showNotification(`Environment changed to ${env}`, 'info');
            
            // Reload data for the new environment
            this.loadDashboardData();
            this.loadCredentialsStatus();
            
            // Send environment change to backend
            fetch('/api/environment', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ environment: env })
            });
        },
        
        // Verify discovery configuration
        async verifyDiscovery() {
            this.showLoading('Verifying discovery configuration...');
            
            try {
                const config = {
                    provider: this.discoveryForm.provider,
                    regions: this.discoveryForm.regions.split(',').map(r => r.trim()).filter(r => r),
                    auto_remediate: this.discoveryForm.autoRemediate
                };
                
                const response = await fetch('/api/discovery/verify', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(config)
                });
                
                const result = await response.json();
                
                if (response.ok && result.valid) {
                    this.showNotification('Configuration verified successfully!', 'success');
                    
                    // Show detailed verification results
                    let message = `✓ Provider: ${result.provider}\n`;
                    message += `✓ Credentials: ${result.credentials_valid ? 'Valid' : 'Invalid'}\n`;
                    message += `✓ Regions: ${result.valid_regions?.join(', ') || 'All'}\n`;
                    message += `✓ Permissions: ${result.permissions_valid ? 'Sufficient' : 'Insufficient'}`;
                    
                    alert('Verification Results:\n\n' + message);
                } else {
                    this.showNotification(`Verification failed: ${result.message || 'Invalid configuration'}`, 'error');
                    
                    if (result.errors && result.errors.length > 0) {
                        alert('Verification Errors:\n\n' + result.errors.join('\n'));
                    }
                }
            } catch (error) {
                console.error('Error verifying discovery:', error);
                this.showNotification('Failed to verify configuration', 'error');
            } finally {
                this.hideLoading();
            }
        },
        
        // Delete selected resources
        async deleteSelectedResources() {
            if (this.selectedResources.length === 0) {
                this.showNotification('No resources selected', 'warning');
                return;
            }
            
            if (!confirm(`Are you sure you want to delete ${this.selectedResources.length} selected resources? This action cannot be undone.`)) {
                return;
            }
            
            this.showLoading(`Deleting ${this.selectedResources.length} resources...`);
            let successCount = 0;
            let failureCount = 0;
            const errors = [];
            
            try {
                // Process deletions in batches
                const batchSize = 5;
                for (let i = 0; i < this.selectedResources.length; i += batchSize) {
                    const batch = this.selectedResources.slice(i, i + batchSize);
                    
                    const promises = batch.map(async (resourceId) => {
                        try {
                            const response = await fetch(`/api/resources/${encodeURIComponent(resourceId)}`, {
                                method: 'DELETE'
                            });
                            
                            if (response.ok) {
                                successCount++;
                                // Remove from local resources array
                                const index = this.resources.findIndex(r => r.id === resourceId);
                                if (index > -1) {
                                    this.resources.splice(index, 1);
                                }
                            } else {
                                failureCount++;
                                const error = await response.text();
                                errors.push(`${resourceId}: ${error}`);
                            }
                        } catch (error) {
                            failureCount++;
                            errors.push(`${resourceId}: ${error.message}`);
                        }
                    });
                    
                    await Promise.all(promises);
                }
                
                // Clear selection
                this.selectedResources = [];
                
                // Show results
                let message = `Deleted ${successCount} resources`;
                if (failureCount > 0) {
                    message += `, ${failureCount} failed`;
                }
                
                this.showNotification(message, successCount > 0 ? 'success' : 'error');
                
                if (errors.length > 0 && errors.length <= 5) {
                    setTimeout(() => {
                        alert('Deletion Errors:\n\n' + errors.join('\n'));
                    }, 500);
                }
                
                // Refresh the resources list
                await this.loadResources();
                
            } catch (error) {
                console.error('Error deleting resources:', error);
                this.showNotification('Failed to delete resources', 'error');
            } finally {
                this.hideLoading();
            }
        },
        
        
        // [Removed duplicate showNotification - using the first implementation]
        
        // Refresh data
        async refreshData() {
            await this.loadDashboardData();
            await this.loadCredentialsStatus();
        },
        
        // Clear cache and trigger re-discovery
        async clearCache() {
            if (!confirm('This will clear all cached resources and trigger a full re-discovery of all configured cloud accounts. This may take several minutes. Continue?')) {
                return;
            }
            
            this.loading = true;
            try {
                const response = await fetch('/api/v1/resources/cache/clear', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                });
                
                const data = await response.json();
                if (data.success) {
                    // Show success message
                    this.showNotification('Cache cleared successfully. Discovery started...', 'success');
                    
                    // Clear current resources
                    this.resources = [];
                    this.stats.totalResources = 0;
                    
                    // Wait a bit then start polling for new resources
                    setTimeout(() => {
                        this.pollForResources();
                    }, 2000);
                } else {
                    this.showNotification('Failed to clear cache', 'error');
                }
            } catch (error) {
                console.error('Error clearing cache:', error);
                this.showNotification('Error clearing cache: ' + error.message, 'error');
            } finally {
                this.loading = false;
            }
        },
        
        // Poll for resources after cache clear
        async pollForResources() {
            let attempts = 0;
            const maxAttempts = 60; // Poll for up to 5 minutes
            
            const poll = setInterval(async () => {
                attempts++;
                
                try {
                    await this.loadResources();
                    await this.loadDashboardData();
                    
                    // If we have resources or max attempts reached, stop polling
                    if (this.resources.length > 0 || attempts >= maxAttempts) {
                        clearInterval(poll);
                        if (this.resources.length > 0) {
                            this.showNotification(`Discovery complete! Found ${this.resources.length} resources.`, 'success');
                        }
                    }
                } catch (error) {
                    console.error('Polling error:', error);
                }
            }, 5000); // Poll every 5 seconds
        },
        
        // Show resource details modal
        showResourceDetails(resource) {
            // Create modal content
            const modalContent = `
                <div class="modal modal-open">
                    <div class="modal-box max-w-4xl">
                        <h3 class="font-bold text-lg mb-4">Resource Details</h3>
                        
                        <div class="grid grid-cols-2 gap-4">
                            <div>
                                <label class="label"><span class="label-text font-bold">ID:</span></label>
                                <p class="text-sm">${resource.id}</p>
                            </div>
                            
                            <div>
                                <label class="label"><span class="label-text font-bold">Name:</span></label>
                                <p class="text-sm">${resource.name || '-'}</p>
                            </div>
                            
                            <div>
                                <label class="label"><span class="label-text font-bold">Type:</span></label>
                                <p class="text-sm">${resource.type}</p>
                            </div>
                            
                            <div>
                                <label class="label"><span class="label-text font-bold">Provider:</span></label>
                                <p class="text-sm">${resource.provider}</p>
                            </div>
                            
                            <div>
                                <label class="label"><span class="label-text font-bold">Region:</span></label>
                                <p class="text-sm">${resource.region || 'global'}</p>
                            </div>
                            
                            <div>
                                <label class="label"><span class="label-text font-bold">Status:</span></label>
                                <p class="text-sm">${resource.status || 'unknown'}</p>
                            </div>
                            
                            ${resource.arn ? `
                            <div class="col-span-2">
                                <label class="label"><span class="label-text font-bold">ARN:</span></label>
                                <p class="text-sm break-all">${resource.arn}</p>
                            </div>
                            ` : ''}
                            
                            ${resource.account ? `
                            <div>
                                <label class="label"><span class="label-text font-bold">Account:</span></label>
                                <p class="text-sm">${resource.account}</p>
                            </div>
                            ` : ''}
                            
                            ${resource.cost ? `
                            <div>
                                <label class="label"><span class="label-text font-bold">Estimated Cost:</span></label>
                                <p class="text-sm">$${resource.cost.toFixed(2)}</p>
                            </div>
                            ` : ''}
                            
                            <div class="col-span-2">
                                <label class="label"><span class="label-text font-bold">Tags:</span></label>
                                <div class="flex flex-wrap gap-1">
                                    ${resource.tags ? Object.entries(resource.tags).map(([key, value]) => 
                                        `<span class="badge badge-sm">${key}: ${value}</span>`
                                    ).join('') : '<span class="text-sm opacity-50">No tags</span>'}
                                </div>
                            </div>
                            
                            ${resource.properties ? `
                            <div class="col-span-2">
                                <label class="label"><span class="label-text font-bold">Properties:</span></label>
                                <pre class="bg-base-200 p-2 rounded text-xs overflow-auto max-h-64">${JSON.stringify(resource.properties, null, 2)}</pre>
                            </div>
                            ` : ''}
                        </div>
                        
                        <div class="modal-action">
                            <button class="btn" onclick="this.closest('.modal').remove()">Close</button>
                        </div>
                    </div>
                    <div class="modal-backdrop bg-black/50" onclick="this.closest('.modal').remove()"></div>
                </div>
            `;
            
            // Add modal to body
            const modalDiv = document.createElement('div');
            modalDiv.innerHTML = modalContent;
            document.body.appendChild(modalDiv.firstElementChild);
        },
        
        // [Removed another duplicate showNotification - using the first implementation]
        
        // Auto-remove notification helper
            setTimeout(() => {
                notification.remove();
            }, 5000);
        },
        
        // Load audit logs
        async loadAuditLogs() {
            try {
                const response = await fetch('/api/v1/audit/logs');
                const data = await response.json();
                
                if (data.logs) {
                    this.auditLogs = data.logs;
                } else {
                    this.auditLogs = [];
                }
            } catch (error) {
                console.error('Error loading audit logs:', error);
                this.auditLogs = [];
            }
        },
        
        // Export audit logs
        async exportAuditLogs(format) {
            try {
                const response = await fetch(`/api/v1/audit/export?format=${format}`);
                
                if (response.ok) {
                    const blob = await response.blob();
                    const url = window.URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = `audit-logs-${Date.now()}.${format}`;
                    a.click();
                    window.URL.revokeObjectURL(url);
                    
                    this.showNotification(`Audit logs exported as ${format}`, 'success');
                }
            } catch (error) {
                console.error('Error exporting audit logs:', error);
                this.showNotification('Failed to export audit logs', 'error');
            }
        },
        
        // Start discovery
        async startDiscovery() {
            this.discovering = true;
            this.startLoading('Initializing resource discovery...', 10000);
            
            // Load cached results immediately if available
            if (this.discoveryResults.length === 0) {
                try {
                    this.updateLoadingProgress(10, 'Checking cached results...');
                    const cachedResponse = await fetch('/api/v1/discovery/cached');
                    if (cachedResponse.ok) {
                        const cachedData = await cachedResponse.json();
                        if (cachedData.resources && cachedData.resources.length > 0) {
                            this.discoveryResults = cachedData.resources;
                            console.log('Loaded cached discovery results while starting new discovery');
                        }
                    }
                } catch (error) {
                    console.error('Error loading cached discovery results:', error);
                }
            }
            
            try {
                this.updateLoadingProgress(20, 'Starting discovery job...');
                const response = await fetch('/api/v1/discover', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        provider: this.discoveryForm.provider || '',
                        regions: this.discoveryForm.regions ? this.discoveryForm.regions.split(',') : [],
                        auto_remediate: this.discoveryForm.autoRemediate || false
                    })
                });
                
                const data = await response.json();
                
                if (data.job_id) {
                    this.discoveryJob = {
                        id: data.job_id,
                        status: 'running',
                        progress: 0,
                        message: 'Starting discovery...'
                    };
                    
                    // Poll for status
                    this.pollDiscoveryStatus(data.job_id);
                }
                
                this.showNotification('Discovery started', 'info');
                
            } catch (error) {
                console.error('Error starting discovery:', error);
                this.showNotification('Failed to start discovery', 'error');
            } finally {
                this.discovering = false;
            }
        },
        
        // Poll discovery status
        async pollDiscoveryStatus(jobId) {
            const checkStatus = async () => {
                try {
                    const response = await fetch(`/api/v1/discover/status?job_id=${jobId}`);
                    const data = await response.json();
                    
                    if (this.discoveryJob && this.discoveryJob.id === jobId) {
                        this.discoveryJob.status = data.status;
                        this.discoveryJob.progress = data.progress || 0;
                        this.discoveryJob.message = data.message || '';
                        
                        // Update loading bar with discovery progress
                        const progress = Math.min(90, 20 + (data.progress || 0) * 0.7);
                        this.updateLoadingProgress(progress, data.message || 'Discovering resources...');
                        
                        if (data.status === 'completed') {
                            this.updateLoadingProgress(95, 'Discovery completed, loading results...');
                            this.loadDiscoveryResults(jobId);
                            this.showNotification('Discovery completed', 'success');
                            this.stopLoading();
                        } else if (data.status === 'failed') {
                            this.showNotification('Discovery failed', 'error');
                            this.stopLoading();
                        } else {
                            // Continue polling
                            setTimeout(checkStatus, 2000);
                        }
                    }
                } catch (error) {
                    console.error('Error checking discovery status:', error);
                }
            };
            
            checkStatus();
        },
        
        // Load discovery results
        async loadDiscoveryResults(jobId) {
            try {
                const response = await fetch(`/api/v1/discover/results?job_id=${jobId}`);
                const data = await response.json();
                
                if (data.resources) {
                    this.discoveryResults = data.resources;
                    // Update stats
                    this.loadDashboardData();
                }
            } catch (error) {
                console.error('Error loading discovery results:', error);
            }
        },
        
        // Drift Detection Functions
        async startDriftDetection() {
            this.driftDetecting = true;
            
            try {
                const response = await fetch('/api/v1/drift/detect', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        provider: this.driftOptions.provider || '',
                        resource_type: this.driftOptions.resourceType || ''
                    })
                });
                
                const data = await response.json();
                
                if (data.job_id) {
                    this.showNotification('Drift detection started', 'info');
                    // Poll for results
                    setTimeout(() => this.loadDriftReport(), 3000);
                }
            } catch (error) {
                console.error('Error starting drift detection:', error);
                this.showNotification('Failed to start drift detection', 'error');
            } finally {
                this.driftDetecting = false;
            }
        },
        
        async loadDriftReport() {
            try {
                const response = await fetch('/api/v1/drift/report');
                const data = await response.json();
                
                // Properly map the summary data
                if (data.summary) {
                    this.driftReport.summary = {
                        total: data.summary.total_resources || data.summary.total || 0,
                        drifted: data.summary.drifted || 0,
                        compliant: data.summary.compliant || 0,
                        remediable: data.summary.remediable || 0,
                        security: data.summary.security || 0
                    };
                }
                
                // Store by_severity and by_provider for charts
                if (data.by_severity) {
                    this.driftReport.by_severity = data.by_severity;
                }
                if (data.by_provider) {
                    this.driftReport.by_provider = data.by_provider;
                }
                
                if (data.drifts) {
                    this.driftReport.drifts = data.drifts;
                    this.applyDriftFilters();
                }
            } catch (error) {
                console.error('Error loading drift report:', error);
            }
        },
        
        applyDriftFilters() {
            let filtered = [...this.driftReport.drifts];
            
            if (this.driftOptions.provider) {
                filtered = filtered.filter(d => d.provider === this.driftOptions.provider);
            }
            
            if (this.driftOptions.resourceType) {
                filtered = filtered.filter(d => d.resource_type === this.driftOptions.resourceType);
            }
            
            this.filteredDrifts = filtered;
        },
        
        viewDriftDetails(drift) {
            this.selectedDrift = drift;
            document.getElementById('driftDetailsModal').showModal();
        },
        
        async remediateDrift(drift) {
            if (!drift.remediable) {
                this.showNotification('This drift cannot be automatically remediated', 'warning');
                return;
            }
            
            try {
                const response = await fetch('/api/v1/drift/remediate', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        drift_id: drift.id,
                        resource_id: drift.resource_id,
                        provider: drift.provider
                    })
                });
                
                const data = await response.json();
                
                if (data.success) {
                    this.showNotification('Remediation started', 'success');
                    // Reload drift report after remediation
                    setTimeout(() => this.loadDriftReport(), 3000);
                } else {
                    this.showNotification(data.message || 'Remediation failed', 'error');
                }
            } catch (error) {
                console.error('Error remediating drift:', error);
                this.showNotification('Failed to remediate drift', 'error');
            }
        },
        
        async ignoreDrift(drift) {
            // Mark drift as ignored
            drift.ignored = true;
            this.showNotification('Drift marked as ignored', 'info');
            this.applyDriftFilters();
        },
        
        async remediateSelected() {
            if (this.selectedDrifts.length === 0) {
                this.showNotification('No drifts selected', 'warning');
                return;
            }
            
            const remediable = this.selectedDrifts.filter(d => d.remediable);
            if (remediable.length === 0) {
                this.showNotification('No selected drifts can be remediated', 'warning');
                return;
            }
            
            for (const drift of remediable) {
                await this.remediateDrift(drift);
            }
            
            this.selectedDrifts = [];
        },
        
        async exportDriftReport() {
            try {
                const response = await fetch('/api/v1/drift/report/export?format=json');
                
                if (response.ok) {
                    const blob = await response.blob();
                    const url = window.URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = `drift-report-${Date.now()}.json`;
                    a.click();
                    window.URL.revokeObjectURL(url);
                    
                    this.showNotification('Drift report exported', 'success');
                }
            } catch (error) {
                console.error('Error exporting drift report:', error);
                this.showNotification('Failed to export drift report', 'error');
            }
        },
        
        // Perspective Analysis Methods
        async runPerspectiveAnalysis() {
            this.perspectiveAnalyzing = true;
            this.showLoading('Running perspective analysis...');
            
            try {
                const response = await fetch('/api/perspective/analyze', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        provider: this.perspectiveOptions.provider || null,
                        regions: this.perspectiveOptions.regions || [],
                        state_file: this.perspectiveOptions.stateFile || null,
                        include_tags: true,
                        group_by_type: true
                    })
                });
                
                if (!response.ok) {
                    throw new Error(`Analysis failed: ${response.statusText}`);
                }
                
                this.perspectiveReport = await response.json();
                this.showNotification('Perspective analysis completed', 'success');
            } catch (error) {
                console.error('Error running perspective analysis:', error);
                this.showNotification('Failed to run perspective analysis', 'error');
            } finally {
                this.perspectiveAnalyzing = false;
                this.hideLoading();
            }
        },
        
        async loadPerspectiveReport() {
            try {
                const response = await fetch('/api/perspective/report');
                if (response.ok) {
                    const report = await response.json();
                    if (report.summary) {
                        this.perspectiveReport = report;
                    }
                }
            } catch (error) {
                console.error('Error loading perspective report:', error);
            }
        },
        
        toggleAllUnmanaged(checked) {
            if (checked) {
                this.selectedUnmanagedResources = this.perspectiveReport.unmanaged_resources.map(r => r.id);
            } else {
                this.selectedUnmanagedResources = [];
            }
        },
        
        async importResource(resource) {
            if (!confirm(`Import ${resource.id} into Terraform state?`)) return;
            
            this.showLoading('Importing resource...');
            try {
                const response = await fetch('/api/resources/import', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        resource_id: resource.id,
                        resource_type: resource.type,
                        provider: resource.provider
                    })
                });
                
                if (!response.ok) {
                    throw new Error(`Import failed: ${response.statusText}`);
                }
                
                this.showNotification(`Resource ${resource.id} imported successfully`, 'success');
                await this.runPerspectiveAnalysis();
            } catch (error) {
                console.error('Error importing resource:', error);
                this.showNotification(`Failed to import resource: ${error.message}`, 'error');
            } finally {
                this.hideLoading();
            }
        },
        
        async deleteResource(resource) {
            if (!confirm(`Delete ${resource.id}? This action cannot be undone.`)) return;
            
            this.showLoading('Deleting resource...');
            try {
                const response = await fetch(`/api/resources/${encodeURIComponent(resource.id)}`, {
                    method: 'DELETE'
                });
                
                if (!response.ok) {
                    throw new Error(`Delete failed: ${response.statusText}`);
                }
                
                this.showNotification(`Resource ${resource.id} deleted successfully`, 'success');
                await this.runPerspectiveAnalysis();
            } catch (error) {
                console.error('Error deleting resource:', error);
                this.showNotification(`Failed to delete resource: ${error.message}`, 'error');
            } finally {
                this.hideLoading();
            }
        },
        
        async bulkImportResources() {
            if (!confirm(`Import ${this.selectedUnmanagedResources.length} resources into Terraform state?`)) return;
            
            this.showLoading('Importing resources...');
            let successCount = 0;
            let failureCount = 0;
            
            for (const resourceId of this.selectedUnmanagedResources) {
                const resource = this.perspectiveReport.unmanaged_resources.find(r => r.id === resourceId);
                if (!resource) continue;
                
                try {
                    const response = await fetch('/api/resources/import', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            resource_id: resource.id,
                            resource_type: resource.type,
                            provider: resource.provider
                        })
                    });
                    
                    if (response.ok) {
                        successCount++;
                    } else {
                        failureCount++;
                    }
                } catch (error) {
                    failureCount++;
                }
            }
            
            this.selectedUnmanagedResources = [];
            this.showNotification(`Imported ${successCount} resources, ${failureCount} failed`, successCount > 0 ? 'success' : 'error');
            await this.runPerspectiveAnalysis();
            this.hideLoading();
        },
        
        async bulkDeleteResources() {
            if (!confirm(`Delete ${this.selectedUnmanagedResources.length} resources? This action cannot be undone.`)) return;
            
            this.showLoading('Deleting resources...');
            let successCount = 0;
            let failureCount = 0;
            
            for (const resourceId of this.selectedUnmanagedResources) {
                try {
                    const response = await fetch(`/api/resources/${encodeURIComponent(resourceId)}`, {
                        method: 'DELETE'
                    });
                    
                    if (response.ok) {
                        successCount++;
                    } else {
                        failureCount++;
                    }
                } catch (error) {
                    failureCount++;
                }
            }
            
            this.selectedUnmanagedResources = [];
            this.showNotification(`Deleted ${successCount} resources, ${failureCount} failed`, successCount > 0 ? 'success' : 'error');
            await this.runPerspectiveAnalysis();
            this.hideLoading();
        },
        
        // Resources Management Functions
        async loadResources() {
            // If resources are already loaded from cache, don't show loading state
            if (this.resources.length > 0) {
                console.log('Using cached resources:', this.resources.length);
                this.extractFilterOptions();
                this.filterResources();
                return;
            }
            
            this.resourcesLoading = true;
            this.startLoading('Loading cloud resources...', 3000);
            
            try {
                // Try enriched search endpoint for enhanced data
                this.updateLoadingProgress(10, 'Checking enriched cache...');
                try {
                    const enrichedResponse = await fetch('/api/enriched/search?limit=1000', {
                        method: 'GET'
                    });
                    if (enrichedResponse.ok) {
                        const enrichedData = await enrichedResponse.json();
                        if (enrichedData.results && enrichedData.results.length > 0) {
                            this.updateLoadingProgress(80, 'Processing enriched resources...');
                            // Convert enriched cache entries to resources
                            this.resources = enrichedData.results.map(entry => ({
                                id: entry.metadata?.custom?.id || entry.key,
                                name: entry.metadata?.custom?.name || entry.value?.name || 'Unknown',
                                type: entry.metadata?.resource_type || entry.value?.type || 'Unknown',
                                provider: entry.metadata?.provider || 'unknown',
                                region: entry.metadata?.region || 'global',
                                status: entry.metadata?.custom?.status || 'active',
                                cost: entry.metrics?.monthly_cost || 0,
                                compliance: entry.compliance || {},
                                tags: entry.tags || {},
                                enriched: true
                            }));
                            this.extractFilterOptions();
                            this.filterResources();
                            this.resourcesLoading = false;
                            this.stopLoading();
                            console.log('Loaded enriched resources:', this.resources.length);
                            return;
                        }
                    }
                } catch (enrichedError) {
                    console.log('Enriched endpoint not available, falling back to cached endpoint');
                }
                
                // Try cached endpoint for immediate response
                this.updateLoadingProgress(20, 'Checking cache...');
                const cachedResponse = await fetch('/api/v1/discovery/cached');
                if (cachedResponse.ok) {
                    const cachedData = await cachedResponse.json();
                    if (cachedData.resources && cachedData.resources.length > 0) {
                        this.updateLoadingProgress(80, 'Processing cached resources...');
                        this.resources = cachedData.resources;
                        this.extractFilterOptions();
                        this.filterResources();
                        this.resourcesLoading = false;
                        this.stopLoading();
                        return;
                    }
                }
                
                // Fallback to regular resources endpoint
                this.updateLoadingProgress(50, 'Fetching resources from API...');
                const response = await fetch('/api/v1/resources');
                const data = await response.json();
                
                this.updateLoadingProgress(80, 'Processing resources...');
                if (data.resources) {
                    this.resources = data.resources;
                } else {
                    this.resources = [];
                }
                
                this.extractFilterOptions();
                this.filterResources();
            } catch (error) {
                console.error('Error loading resources:', error);
                this.resources = [];
                this.filteredResources = [];
            } finally {
                this.resourcesLoading = false;
                this.stopLoading();
            }
        },
        
        // Extract available filter options from resources
        extractFilterOptions() {
            const providers = new Set();
            const resourceTypes = new Set();
            const regions = new Set();
            
            this.resources.forEach(resource => {
                if (resource.provider) providers.add(resource.provider);
                if (resource.type) resourceTypes.add(resource.type);
                if (resource.region) regions.add(resource.region);
            });
            
            this.availableProviders = Array.from(providers).sort();
            this.availableResourceTypes = Array.from(resourceTypes).sort();
            this.availableRegions = Array.from(regions).sort();
        },
        
        // Filter resources based on current filters
        filterResources() {
            let filtered = [...this.resources];
            
            // Apply provider filter
            if (this.resourceFilters.provider) {
                filtered = filtered.filter(r => 
                    r.provider === this.resourceFilters.provider
                );
            }
            
            // Apply type filter
            if (this.resourceFilters.type) {
                filtered = filtered.filter(r => 
                    r.type === this.resourceFilters.type
                );
            }
            
            // Apply region filter
            if (this.resourceFilters.region) {
                filtered = filtered.filter(r => 
                    r.region === this.resourceFilters.region
                );
            }
            
            // Apply search filter
            if (this.resourceFilters.search) {
                const searchLower = this.resourceFilters.search.toLowerCase();
                filtered = filtered.filter(r => {
                    return (r.name && r.name.toLowerCase().includes(searchLower)) ||
                           (r.id && r.id.toLowerCase().includes(searchLower)) ||
                           (r.type && r.type.toLowerCase().includes(searchLower));
                });
            }
            
            this.filteredResources = filtered;
        },
        
        // Clear all filters
        clearFilters() {
            this.resourceFilters = {
                provider: '',
                type: '',
                region: '',
                search: ''
            };
            this.filterResources();
        },
        
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
                    this.loadResources();
                } else {
                    this.showNotification('Failed to delete resource', 'error');
                }
            } catch (error) {
                console.error('Error deleting resource:', error);
                this.showNotification('Failed to delete resource', 'error');
            }
        },
        
        async exportResources() {
            if (!this.exportFormat) {
                this.showNotification('Please select an export format', 'warning');
                return;
            }
            
            this.showLoading(`Exporting resources as ${this.exportFormat.toUpperCase()}...`);
            
            try {
                // Build query params based on current filters
                const params = new URLSearchParams();
                params.append('format', this.exportFormat);
                
                if (this.resourceFilters.provider) {
                    params.append('provider', this.resourceFilters.provider);
                }
                if (this.resourceFilters.type) {
                    params.append('type', this.resourceFilters.type);
                }
                if (this.resourceFilters.region) {
                    params.append('region', this.resourceFilters.region);
                }
                if (this.selectedResources.length > 0) {
                    params.append('ids', this.selectedResources.join(','));
                }
                
                const response = await fetch(`/api/resources/export?${params}`);
                
                if (response.ok) {
                    const blob = await response.blob();
                    const url = window.URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = `resources-${Date.now()}.${this.exportFormat}`;
                    a.click();
                    window.URL.revokeObjectURL(url);
                    
                    this.showNotification('Resources exported successfully', 'success');
                    this.showExportModal = false;
                }
            } catch (error) {
                console.error('Error exporting resources:', error);
                this.showNotification('Failed to export resources', 'error');
            }
        },
        
        handleImportFileSelect(event) {
            const file = event.target.files[0];
            if (file) {
                this.selectedImportFile = file;
                this.showNotification(`Selected: ${file.name}`, 'info');
            }
        },
        
        async importSelectedFile() {
            if (!this.selectedImportFile) {
                this.showNotification('Please select a file first', 'warning');
                return;
            }
            
            await this.importResources(this.selectedImportFile);
            this.selectedImportFile = null;
        },
        
        async importResources(file) {
            if (!file) {
                this.showNotification('Please select a file to import', 'warning');
                return;
            }
            
            // Validate file type
            const validTypes = ['application/json', 'text/csv', 'application/x-yaml', 'text/yaml'];
            const fileExt = file.name.split('.').pop().toLowerCase();
            
            if (!['json', 'csv', 'yaml', 'yml'].includes(fileExt)) {
                this.showNotification('Invalid file type. Please select JSON, CSV, or YAML file', 'error');
                return;
            }
            
            this.showLoading(`Importing resources from ${file.name}...`);
            
            const formData = new FormData();
            formData.append('file', file);
            formData.append('format', fileExt);
            
            try {
                const response = await fetch('/api/resources/import', {
                    method: 'POST',
                    body: formData
                });
                
                const data = await response.json();
                
                if (response.ok && data.success) {
                    this.showNotification(`Successfully imported ${data.count || 0} resources`, 'success');
                    this.showImportModal = false;
                    
                    // Show import summary if available
                    if (data.summary) {
                        let summary = `Import Summary:\n`;
                        summary += `✓ Total: ${data.summary.total || 0}\n`;
                        summary += `✓ Success: ${data.summary.success || 0}\n`;
                        summary += `✗ Failed: ${data.summary.failed || 0}\n`;
                        summary += `⚠ Skipped: ${data.summary.skipped || 0}`;
                        
                        setTimeout(() => alert(summary), 500);
                    }
                    
                    // Reload resources
                    await this.loadResources();
                    await this.loadDashboardData();
                } else {
                    this.showNotification(data.message || 'Import failed', 'error');
                    
                    if (data.errors && data.errors.length > 0) {
                        console.error('Import errors:', data.errors);
                        alert('Import Errors:\n\n' + data.errors.slice(0, 5).join('\n'));
                    }
                }
            } catch (error) {
                console.error('Error importing resources:', error);
                this.showNotification('Failed to import resources', 'error');
            } finally {
                this.hideLoading();
                // Reset file input
                const fileInput = document.querySelector('input[type="file"]');
                if (fileInput) fileInput.value = '';
            }
        },
        
        // Advanced Operations: Batch Operations
        async executeBatchOperation() {
            if (!this.batchOperation.operation) {
                this.showNotification('Please select an operation', 'warning');
                return;
            }
            
            const params = new URLSearchParams({
                operation: this.batchOperation.operation,
                provider: this.batchOperation.filters.provider,
                region: this.batchOperation.filters.region,
                resourceType: this.batchOperation.filters.resourceType,
                tags: this.batchOperation.filters.tags,
                dryRun: this.batchOperation.options.dryRun,
                force: this.batchOperation.options.force,
                includeDeps: this.batchOperation.options.includeDeps
            });
            
            try {
                this.startLoading(`Executing batch ${this.batchOperation.operation}...`, 10000);
                const response = await fetch(`/api/v1/batch/execute?${params}`, {
                    method: 'POST'
                });
                
                const data = await response.json();
                
                if (data.success) {
                    this.showNotification(`Batch operation completed: ${data.affected} resources affected`, 'success');
                    
                    // Add to results
                    this.batchOperation.results.push({
                        operation: this.batchOperation.operation,
                        timestamp: new Date().toISOString(),
                        affected: data.affected,
                        details: data.details
                    });
                    
                    // Reload resources if not dry run
                    if (!this.batchOperation.options.dryRun) {
                        await this.loadResources();
                    }
                } else {
                    this.showNotification(data.message || 'Batch operation failed', 'error');
                }
            } catch (error) {
                console.error('Error executing batch operation:', error);
                this.showNotification('Failed to execute batch operation', 'error');
            } finally {
                this.stopLoading();
            }
        },
        
        // Advanced Operations: Verification
        async runVerification() {
            const params = new URLSearchParams({
                enhanced: this.verification.options.enhanced,
                validateOnly: this.verification.options.validateOnly,
                compliance: this.verification.compliance.enabled,
                complianceFramework: this.verification.compliance.framework,
                costAnalysis: this.verification.costAnalysis
            });
            
            try {
                this.startLoading('Running verification checks...', 15000);
                const response = await fetch(`/api/v1/verify?${params}`, {
                    method: 'POST'
                });
                
                const data = await response.json();
                
                if (data.results) {
                    this.verification.results = data.results;
                    
                    // Count issues by severity
                    const severityCounts = { critical: 0, warning: 0, info: 0 };
                    data.results.forEach(result => {
                        if (result.severity) {
                            severityCounts[result.severity] = (severityCounts[result.severity] || 0) + 1;
                        }
                    });
                    
                    this.showNotification(
                        `Verification complete: ${severityCounts.critical} critical, ${severityCounts.warning} warnings`,
                        severityCounts.critical > 0 ? 'error' : 'success'
                    );
                } else {
                    this.showNotification('No verification issues found', 'success');
                }
            } catch (error) {
                console.error('Error running verification:', error);
                this.showNotification('Failed to run verification', 'error');
            } finally {
                this.stopLoading();
            }
        },
        
        // Advanced Operations: Configuration Management
        async uploadConfig(event) {
            const file = event.target.files[0];
            if (!file) return;
            
            const formData = new FormData();
            formData.append('config', file);
            
            try {
                this.startLoading('Uploading configuration...', 3000);
                const response = await fetch('/api/v1/config/upload', {
                    method: 'POST',
                    body: formData
                });
                
                const data = await response.json();
                
                if (data.success) {
                    this.configManagement.currentConfig = data.config;
                    this.configManagement.lastModified = new Date().toISOString();
                    this.showNotification('Configuration uploaded successfully', 'success');
                } else {
                    this.showNotification(data.message || 'Upload failed', 'error');
                }
            } catch (error) {
                console.error('Error uploading config:', error);
                this.showNotification('Failed to upload configuration', 'error');
            } finally {
                this.stopLoading();
            }
        },
        
        async saveConfig() {
            if (!this.configManagement.currentConfig) {
                this.showNotification('No configuration to save', 'warning');
                return;
            }
            
            try {
                const response = await fetch('/api/v1/config/save', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(this.configManagement.currentConfig)
                });
                
                if (response.ok) {
                    this.configManagement.lastModified = new Date().toISOString();
                    this.showNotification('Configuration saved successfully', 'success');
                }
            } catch (error) {
                console.error('Error saving config:', error);
                this.showNotification('Failed to save configuration', 'error');
            }
        },
        
        async validateConfig() {
            if (!this.configManagement.currentConfig) {
                this.showNotification('No configuration to validate', 'warning');
                return;
            }
            
            try {
                const response = await fetch('/api/v1/config/validate', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(this.configManagement.currentConfig)
                });
                
                const data = await response.json();
                
                if (data.valid) {
                    this.configManagement.validationResult = { valid: true, message: 'Configuration is valid' };
                    this.showNotification('Configuration is valid', 'success');
                } else {
                    this.configManagement.validationResult = { valid: false, errors: data.errors };
                    this.showNotification('Configuration has errors', 'error');
                }
            } catch (error) {
                console.error('Error validating config:', error);
                this.showNotification('Failed to validate configuration', 'error');
            }
        },
        
        async exportConfig() {
            try {
                const response = await fetch('/api/v1/config/export');
                
                if (response.ok) {
                    const blob = await response.blob();
                    const url = window.URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = `driftmgr-config-${Date.now()}.yaml`;
                    a.click();
                    window.URL.revokeObjectURL(url);
                    
                    this.showNotification('Configuration exported successfully', 'success');
                }
            } catch (error) {
                console.error('Error exporting config:', error);
                this.showNotification('Failed to export configuration', 'error');
            }
        },
        
        // Advanced Operations: Terminal
        async executeCommand() {
            const command = this.terminal.currentCommand.trim();
            if (!command) return;
            
            // Add to history
            this.terminal.history.push({
                command: command,
                timestamp: new Date().toISOString(),
                output: null
            });
            
            try {
                this.terminal.isExecuting = true;
                
                const response = await fetch('/api/v1/terminal/execute', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ command: command })
                });
                
                const data = await response.json();
                
                // Update history with output
                const lastIndex = this.terminal.history.length - 1;
                this.terminal.history[lastIndex].output = data.output || data.error;
                
                if (!data.success && data.error) {
                    this.showNotification(`Command failed: ${data.error}`, 'error');
                }
                
                // Clear current command
                this.terminal.currentCommand = '';
                
                // Scroll terminal to bottom
                this.$nextTick(() => {
                    const terminal = document.querySelector('.terminal-output');
                    if (terminal) terminal.scrollTop = terminal.scrollHeight;
                });
            } catch (error) {
                console.error('Error executing command:', error);
                const lastIndex = this.terminal.history.length - 1;
                this.terminal.history[lastIndex].output = `Error: ${error.message}`;
            } finally {
                this.terminal.isExecuting = false;
            }
        },
        
        clearTerminal() {
            this.terminal.history = [];
            this.terminal.currentCommand = '';
        },
        
        // Handle terminal keyboard shortcuts
        handleTerminalKeydown(event) {
            if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault();
                this.executeCommand();
            } else if (event.key === 'l' && event.ctrlKey) {
                event.preventDefault();
                this.clearTerminal();
            }
        },
        
        // State File Operations
        async handleStateFileUpload(event) {
            const files = event.target.files;
            if (!files || files.length === 0) return;
            
            this.startLoading('Uploading state files...', 3000);
            
            for (const file of files) {
                const formData = new FormData();
                formData.append('file', file);
                
                try {
                    const response = await fetch('/api/v1/state/upload', {
                        method: 'POST',
                        body: formData
                    });
                    
                    if (response.ok) {
                        const result = await response.json();
                        this.stateFiles.push(result);
                        this.showNotification(`Uploaded ${file.name}`, 'success');
                    }
                } catch (error) {
                    console.error('Upload error:', error);
                    this.showNotification(`Failed to upload ${file.name}`, 'error');
                }
            }
            
            this.stopLoading();
            await this.analyzeAllStateFiles();
        },
        
        
        
        async connectRemoteBackend() {
            if (!this.remoteBackendType) return;
            
            this.startLoading(`Connecting to ${this.remoteBackendType} backend...`, 5000);
            
            try {
                const response = await fetch('/api/v1/state/remote', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ 
                        backend: this.remoteBackendType,
                        config: {} // Add backend-specific config here
                    })
                });
                
                if (response.ok) {
                    const files = await response.json();
                    this.stateFiles.push(...files);
                    this.showNotification(`Connected to ${this.remoteBackendType}`, 'success');
                    await this.analyzeAllStateFiles();
                } else {
                    this.showNotification('Failed to connect to remote backend', 'error');
                }
            } catch (error) {
                console.error('Remote backend error:', error);
                this.showNotification('Failed to connect to remote backend', 'error');
            } finally {
                this.stopLoading();
            }
        },
        
        selectAllStates(checked) {
            if (checked) {
                this.selectedStates = this.stateFiles.map(f => f.path);
            } else {
                this.selectedStates = [];
            }
        },
        
        async analyzeStateFile(file) {
            this.selectedState = file;
            this.startLoading(`Analyzing ${file.name}...`, 2000);
            
            try {
                const response = await fetch(`/api/v1/state/analyze/${encodeURIComponent(file.path)}`);
                
                if (response.ok) {
                    const analysis = await response.json();
                    this.stateResources = analysis.resources || [];
                    document.getElementById('stateDetailsModal').showModal();
                }
            } catch (error) {
                console.error('Analysis error:', error);
                this.showNotification('Failed to analyze state file', 'error');
            } finally {
                this.stopLoading();
            }
        },
        
        async analyzeAllStateFiles() {
            if (this.stateFiles.length === 0) return;
            
            this.startLoading('Analyzing all state files...', 5000);
            
            try {
                const response = await fetch('/api/v1/state/analyze-all', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ 
                        files: this.stateFiles.map(f => f.path)
                    })
                });
                
                if (response.ok) {
                    const analysis = await response.json();
                    this.stateAnalysis = analysis;
                    
                    // Update charts if on analysis view
                    if (this.currentView === 'analysis') {
                        this.updateAnalysisCharts();
                    }
                }
            } catch (error) {
                console.error('Analysis error:', error);
            } finally {
                this.stopLoading();
            }
        },
        
        async compareStateFiles() {
            if (!this.comparison.file1 || !this.comparison.file2) return;
            
            this.startLoading('Comparing state files...', 3000);
            
            try {
                const response = await fetch('/api/v1/state/compare', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        file1: this.comparison.file1,
                        file2: this.comparison.file2
                    })
                });
                
                if (response.ok) {
                    this.comparison.results = await response.json();
                    this.comparisonTab = 'added'; // Reset to first tab
                } else {
                    this.showNotification('Failed to compare state files', 'error');
                }
            } catch (error) {
                console.error('Comparison error:', error);
                this.showNotification('Failed to compare state files', 'error');
            } finally {
                this.stopLoading();
            }
        },
        
        viewDiff(resource) {
            // Create and show diff viewer modal
            const modal = document.createElement('div');
            modal.className = 'modal modal-open';
            modal.id = 'diff-viewer-modal';
            
            // Generate diff content
            const diffContent = this.generateDiffContent(resource);
            
            modal.innerHTML = `
                <div class="modal-box max-w-4xl">
                    <h3 class="font-bold text-lg mb-4">
                        <i class="fas fa-code-compare mr-2"></i>
                        Resource Drift Details: ${resource.name || resource.id}
                    </h3>
                    
                    <div class="mb-4">
                        <div class="stats shadow w-full">
                            <div class="stat">
                                <div class="stat-title">Resource Type</div>
                                <div class="stat-value text-sm">${resource.type}</div>
                            </div>
                            <div class="stat">
                                <div class="stat-title">Provider</div>
                                <div class="stat-value text-sm">${resource.provider}</div>
                            </div>
                            <div class="stat">
                                <div class="stat-title">Drift Type</div>
                                <div class="stat-value text-sm">${resource.drift_type || 'Modified'}</div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="diff-container" style="max-height: 60vh; overflow-y: auto;">
                        ${diffContent}
                    </div>
                    
                    <div class="modal-action">
                        <button class="btn btn-sm" onclick="this.closest('.modal').remove()">
                            <i class="fas fa-times mr-1"></i> Close
                        </button>
                        <button class="btn btn-sm btn-primary" onclick="window.app.copyDiffToClipboard('${resource.id}')">
                            <i class="fas fa-copy mr-1"></i> Copy Diff
                        </button>
                        <button class="btn btn-sm btn-warning" onclick="window.app.remediateDrift('${resource.id}')">
                            <i class="fas fa-wrench mr-1"></i> Remediate
                        </button>
                    </div>
                </div>
                <div class="modal-backdrop" onclick="this.parentElement.remove()"></div>
            `;
            
            document.body.appendChild(modal);
        },
        
        generateDiffContent(resource) {
            let html = '<div class="space-y-4">';
            
            if (resource.differences && resource.differences.length > 0) {
                html += '<div class="overflow-x-auto">';
                html += '<table class="table table-compact w-full">';
                html += '<thead><tr>';
                html += '<th>Property</th>';
                html += '<th>State Value</th>';
                html += '<th>Actual Value</th>';
                html += '<th>Action</th>';
                html += '</tr></thead>';
                html += '<tbody>';
                
                resource.differences.forEach(diff => {
                    const stateVal = this.formatDiffValue(diff.state_value);
                    const actualVal = this.formatDiffValue(diff.actual_value);
                    const isDeletion = !diff.actual_value && diff.state_value;
                    const isAddition = diff.actual_value && !diff.state_value;
                    
                    html += '<tr>';
                    html += `<td class="font-mono text-sm">${diff.path || diff.property}</td>`;
                    html += `<td class="${isDeletion ? 'text-error' : ''}">`;
                    html += `<pre class="bg-base-200 p-1 rounded text-xs max-w-xs overflow-x-auto">${stateVal}</pre>`;
                    html += '</td>';
                    html += `<td class="${isAddition ? 'text-success' : ''}">`;
                    html += `<pre class="bg-base-200 p-1 rounded text-xs max-w-xs overflow-x-auto">${actualVal}</pre>`;
                    html += '</td>';
                    html += '<td>';
                    if (isDeletion) {
                        html += '<span class="badge badge-error badge-sm">Removed</span>';
                    } else if (isAddition) {
                        html += '<span class="badge badge-success badge-sm">Added</span>';
                    } else {
                        html += '<span class="badge badge-warning badge-sm">Modified</span>';
                    }
                    html += '</td>';
                    html += '</tr>';
                });
                
                html += '</tbody></table></div>';
            } else if (resource.state_value && resource.actual_value) {
                // Show full JSON diff for complex objects
                html += '<div class="grid grid-cols-2 gap-4">';
                html += '<div>';
                html += '<h4 class="font-semibold mb-2">Terraform State</h4>';
                html += `<pre class="bg-base-200 p-3 rounded overflow-x-auto text-xs">${this.formatJSON(resource.state_value)}</pre>`;
                html += '</div>';
                html += '<div>';
                html += '<h4 class="font-semibold mb-2">Actual State</h4>';
                html += `<pre class="bg-base-200 p-3 rounded overflow-x-auto text-xs">${this.formatJSON(resource.actual_value)}</pre>`;
                html += '</div>';
                html += '</div>';
            } else {
                html += '<div class="alert alert-info">';
                html += '<i class="fas fa-info-circle"></i>';
                html += '<span>No detailed differences available for this resource.</span>';
                html += '</div>';
            }
            
            html += '</div>';
            return html;
        },
        
        formatDiffValue(value) {
            if (value === null || value === undefined) {
                return '<em class="text-base-content/50">null</em>';
            }
            if (typeof value === 'object') {
                return JSON.stringify(value, null, 2);
            }
            return String(value);
        },
        
        formatJSON(obj) {
            try {
                if (typeof obj === 'string') {
                    obj = JSON.parse(obj);
                }
                return JSON.stringify(obj, null, 2);
            } catch {
                return String(obj);
            }
        },
        
        copyDiffToClipboard(resourceId) {
            const resource = this.driftData.find(r => r.id === resourceId);
            if (!resource) return;
            
            let diffText = `Resource Drift: ${resource.name || resource.id}\n`;
            diffText += `Type: ${resource.type}\n`;
            diffText += `Provider: ${resource.provider}\n\n`;
            
            if (resource.differences) {
                diffText += 'Differences:\n';
                resource.differences.forEach(diff => {
                    diffText += `  ${diff.path}: ${diff.state_value} → ${diff.actual_value}\n`;
                });
            }
            
            navigator.clipboard.writeText(diffText).then(() => {
                this.showNotification('Diff copied to clipboard', 'success');
            });
        },
        
        async remediateDrift(resourceId) {
            const resource = this.driftData.find(r => r.id === resourceId);
            if (!resource) return;
            
            // Close the diff modal
            const modal = document.getElementById('diff-viewer-modal');
            if (modal) modal.remove();
            
            // Start remediation
            this.startLoading('Generating remediation plan...', 15000);
            
            try {
                const response = await fetch('/api/v1/drift/remediate', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ resource_id: resourceId })
                });
                
                if (response.ok) {
                    const result = await response.json();
                    this.showNotification('Remediation plan generated', 'success');
                    // Show remediation plan in a new modal or navigate to remediation view
                    this.showRemediationPlan(result);
                } else {
                    this.showNotification('Failed to generate remediation plan', 'error');
                }
            } catch (error) {
                console.error('Remediation error:', error);
                this.showNotification('Failed to generate remediation plan', 'error');
            } finally {
                this.stopLoading();
            }
        },
        
        showRemediationPlan(plan) {
            // Implementation for showing remediation plan
            console.log('Remediation plan:', plan);
            // This would show a modal with the remediation plan details
        },
        
        async generateReport(type) {
            if (this.selectedStates.length === 0) {
                this.showNotification('Please select state files first', 'warning');
                return;
            }
            
            this.reportGenerating = true;
            this.startLoading(`Generating ${type} report...`, 10000);
            
            try {
                const response = await fetch('/api/v1/state/report', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        type,
                        files: this.selectedStates
                    })
                });
                
                if (response.ok) {
                    const report = await response.json();
                    this.generatedReports.unshift({
                        id: Date.now(),
                        type,
                        timestamp: new Date().toISOString(),
                        fileCount: this.selectedStates.length,
                        status: 'completed',
                        data: report
                    });
                    this.showNotification(`${type} report generated`, 'success');
                } else {
                    this.showNotification('Failed to generate report', 'error');
                }
            } catch (error) {
                console.error('Report generation error:', error);
                this.showNotification('Failed to generate report', 'error');
            } finally {
                this.reportGenerating = false;
                this.stopLoading();
            }
        },
        
        viewReport(report) {
            // Create and show report viewer modal
            const modal = document.createElement('div');
            modal.className = 'modal modal-open';
            modal.id = 'report-viewer-modal';
            
            const reportContent = this.generateReportContent(report);
            
            modal.innerHTML = `
                <div class="modal-box max-w-5xl">
                    <h3 class="font-bold text-lg mb-4">
                        <i class="fas fa-file-alt mr-2"></i>
                        ${report.type.charAt(0).toUpperCase() + report.type.slice(1)} Report
                    </h3>
                    
                    <div class="mb-4">
                        <div class="flex justify-between items-center">
                            <div class="text-sm text-base-content/70">
                                Generated: ${new Date(report.timestamp).toLocaleString()}
                            </div>
                            <div class="flex gap-2">
                                <button class="btn btn-xs btn-outline" onclick="window.app.exportReport('${report.id}', 'json')">
                                    <i class="fas fa-download"></i> JSON
                                </button>
                                <button class="btn btn-xs btn-outline" onclick="window.app.exportReport('${report.id}', 'csv')">
                                    <i class="fas fa-file-csv"></i> CSV
                                </button>
                                <button class="btn btn-xs btn-outline" onclick="window.app.exportReport('${report.id}', 'pdf')">
                                    <i class="fas fa-file-pdf"></i> PDF
                                </button>
                            </div>
                        </div>
                    </div>
                    
                    <div class="report-container" style="max-height: 65vh; overflow-y: auto;">
                        ${reportContent}
                    </div>
                    
                    <div class="modal-action">
                        <button class="btn btn-sm" onclick="this.closest('.modal').remove()">
                            <i class="fas fa-times mr-1"></i> Close
                        </button>
                        <button class="btn btn-sm btn-primary" onclick="window.app.shareReport('${report.id}')">
                            <i class="fas fa-share-alt mr-1"></i> Share
                        </button>
                    </div>
                </div>
                <div class="modal-backdrop" onclick="this.parentElement.remove()"></div>
            `;
            
            document.body.appendChild(modal);
        },
        
        generateReportContent(report) {
            let html = '<div class="space-y-6">';
            
            // Report summary section
            if (report.data.summary) {
                html += this.generateReportSummary(report.data.summary);
            }
            
            // Report type-specific content
            switch (report.type) {
                case 'coverage':
                    html += this.generateCoverageReport(report.data);
                    break;
                case 'compliance':
                    html += this.generateComplianceReport(report.data);
                    break;
                case 'cost':
                    html += this.generateCostReport(report.data);
                    break;
                case 'drift':
                    html += this.generateDriftReport(report.data);
                    break;
                case 'security':
                    html += this.generateSecurityReport(report.data);
                    break;
                default:
                    html += this.generateGenericReport(report.data);
            }
            
            html += '</div>';
            return html;
        },
        
        generateReportSummary(summary) {
            let html = '<div class="card bg-base-200">';
            html += '<div class="card-body">';
            html += '<h4 class="card-title text-base">Summary</h4>';
            html += '<div class="stats stats-horizontal shadow">';
            
            Object.entries(summary).forEach(([key, value]) => {
                html += '<div class="stat">';
                html += `<div class="stat-title">${this.formatLabel(key)}</div>`;
                html += `<div class="stat-value text-2xl">${this.formatStatValue(value)}</div>`;
                html += '</div>';
            });
            
            html += '</div></div></div>';
            return html;
        },
        
        generateCoverageReport(data) {
            let html = '<div class="card bg-base-100">';
            html += '<div class="card-body">';
            html += '<h4 class="card-title text-base">Resource Coverage Analysis</h4>';
            
            if (data.coverage_by_type) {
                html += '<div class="overflow-x-auto">';
                html += '<table class="table table-compact w-full">';
                html += '<thead><tr>';
                html += '<th>Resource Type</th>';
                html += '<th>Managed</th>';
                html += '<th>Unmanaged</th>';
                html += '<th>Coverage</th>';
                html += '</tr></thead><tbody>';
                
                Object.entries(data.coverage_by_type).forEach(([type, coverage]) => {
                    const percentage = (coverage.managed / (coverage.managed + coverage.unmanaged) * 100).toFixed(1);
                    html += '<tr>';
                    html += `<td>${type}</td>`;
                    html += `<td>${coverage.managed}</td>`;
                    html += `<td>${coverage.unmanaged}</td>`;
                    html += `<td>`;
                    html += `<div class="flex items-center gap-2">`;
                    html += `<progress class="progress progress-primary w-20" value="${percentage}" max="100"></progress>`;
                    html += `<span class="text-xs">${percentage}%</span>`;
                    html += `</div></td>`;
                    html += '</tr>';
                });
                
                html += '</tbody></table></div>';
            }
            
            html += '</div></div>';
            return html;
        },
        
        generateComplianceReport(data) {
            let html = '<div class="card bg-base-100">';
            html += '<div class="card-body">';
            html += '<h4 class="card-title text-base">Compliance Status</h4>';
            
            if (data.violations) {
                html += '<div class="space-y-2">';
                data.violations.forEach(violation => {
                    const severity = violation.severity || 'medium';
                    const badgeClass = severity === 'critical' ? 'badge-error' : 
                                       severity === 'high' ? 'badge-warning' : 'badge-info';
                    
                    html += `<div class="alert alert-warning">`;
                    html += `<div class="flex-1">`;
                    html += `<h5 class="font-semibold">${violation.rule}</h5>`;
                    html += `<p class="text-sm">${violation.description}</p>`;
                    html += `<div class="mt-2">`;
                    html += `<span class="badge ${badgeClass} badge-sm">${severity}</span>`;
                    html += `<span class="ml-2 text-xs">Affects ${violation.resource_count} resources</span>`;
                    html += `</div></div></div>`;
                });
                html += '</div>';
            }
            
            html += '</div></div>';
            return html;
        },
        
        generateCostReport(data) {
            let html = '<div class="card bg-base-100">';
            html += '<div class="card-body">';
            html += '<h4 class="card-title text-base">Cost Analysis</h4>';
            
            if (data.total_cost !== undefined) {
                html += `<div class="stat">`;
                html += `<div class="stat-title">Total Monthly Cost</div>`;
                html += `<div class="stat-value text-3xl">$${data.total_cost.toFixed(2)}</div>`;
                html += `</div>`;
            }
            
            if (data.cost_by_service) {
                html += '<div class="overflow-x-auto mt-4">';
                html += '<table class="table table-compact w-full">';
                html += '<thead><tr><th>Service</th><th>Monthly Cost</th><th>% of Total</th></tr></thead>';
                html += '<tbody>';
                
                Object.entries(data.cost_by_service).forEach(([service, cost]) => {
                    const percentage = (cost / data.total_cost * 100).toFixed(1);
                    html += '<tr>';
                    html += `<td>${service}</td>`;
                    html += `<td>$${cost.toFixed(2)}</td>`;
                    html += `<td>${percentage}%</td>`;
                    html += '</tr>';
                });
                
                html += '</tbody></table></div>';
            }
            
            html += '</div></div>';
            return html;
        },
        
        generateDriftReport(data) {
            let html = '<div class="card bg-base-100">';
            html += '<div class="card-body">';
            html += '<h4 class="card-title text-base">Drift Analysis</h4>';
            
            if (data.drift_summary) {
                html += '<div class="grid grid-cols-2 gap-4 mb-4">';
                html += `<div class="stat bg-base-200 rounded">`;
                html += `<div class="stat-title">Total Resources</div>`;
                html += `<div class="stat-value text-2xl">${data.drift_summary.total_resources}</div>`;
                html += `</div>`;
                html += `<div class="stat bg-base-200 rounded">`;
                html += `<div class="stat-title">Drifted Resources</div>`;
                html += `<div class="stat-value text-2xl text-warning">${data.drift_summary.drifted_resources}</div>`;
                html += `</div>`;
                html += '</div>';
            }
            
            if (data.drift_items) {
                html += '<div class="space-y-2">';
                data.drift_items.forEach(item => {
                    html += `<div class="collapse collapse-arrow bg-base-200">`;
                    html += `<input type="checkbox" />`;
                    html += `<div class="collapse-title font-medium">`;
                    html += `${item.resource_name} (${item.resource_type})`;
                    html += `</div>`;
                    html += `<div class="collapse-content">`;
                    html += `<p class="text-sm">${item.differences.length} differences detected</p>`;
                    html += `</div></div>`;
                });
                html += '</div>';
            }
            
            html += '</div></div>';
            return html;
        },
        
        generateSecurityReport(data) {
            let html = '<div class="card bg-base-100">';
            html += '<div class="card-body">';
            html += '<h4 class="card-title text-base">Security Findings</h4>';
            
            if (data.findings) {
                const criticalFindings = data.findings.filter(f => f.severity === 'critical');
                const highFindings = data.findings.filter(f => f.severity === 'high');
                
                if (criticalFindings.length > 0) {
                    html += '<div class="alert alert-error mb-4">';
                    html += `<i class="fas fa-exclamation-triangle"></i>`;
                    html += `<span>${criticalFindings.length} critical security issues found</span>`;
                    html += '</div>';
                }
                
                html += '<div class="space-y-2">';
                data.findings.forEach(finding => {
                    const alertClass = finding.severity === 'critical' ? 'alert-error' :
                                      finding.severity === 'high' ? 'alert-warning' : 'alert-info';
                    
                    html += `<div class="alert ${alertClass}">`;
                    html += `<div class="flex-1">`;
                    html += `<h5 class="font-semibold">${finding.title}</h5>`;
                    html += `<p class="text-sm">${finding.description}</p>`;
                    html += `<p class="text-xs mt-1">Resource: ${finding.resource}</p>`;
                    html += `</div></div>`;
                });
                html += '</div>';
            }
            
            html += '</div></div>';
            return html;
        },
        
        generateGenericReport(data) {
            let html = '<div class="card bg-base-100">';
            html += '<div class="card-body">';
            html += '<pre class="bg-base-200 p-4 rounded overflow-x-auto">';
            html += JSON.stringify(data, null, 2);
            html += '</pre>';
            html += '</div></div>';
            return html;
        },
        
        formatLabel(key) {
            return key.split('_').map(word => 
                word.charAt(0).toUpperCase() + word.slice(1)
            ).join(' ');
        },
        
        formatStatValue(value) {
            if (typeof value === 'number') {
                if (value > 1000000) {
                    return (value / 1000000).toFixed(1) + 'M';
                } else if (value > 1000) {
                    return (value / 1000).toFixed(1) + 'K';
                }
                return value.toLocaleString();
            }
            return value;
        },
        
        async exportReport(reportId, format) {
            const report = this.generatedReports.find(r => r.id == reportId);
            if (!report) return;
            
            try {
                const response = await fetch('/api/v1/state/report/export', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        report: report.data,
                        format: format
                    })
                });
                
                if (response.ok) {
                    const blob = await response.blob();
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = `report-${report.type}-${Date.now()}.${format}`;
                    a.click();
                    URL.revokeObjectURL(url);
                } else {
                    this.showNotification(`Failed to export ${format} report`, 'error');
                }
            } catch (error) {
                console.error('Export error:', error);
                this.showNotification(`Failed to export ${format} report`, 'error');
            }
        },
        
        async shareReport(reportId) {
            const report = this.generatedReports.find(r => r.id == reportId);
            if (!report) return;
            
            try {
                const response = await fetch('/api/v1/state/report/share', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ report: report.data })
                });
                
                if (response.ok) {
                    const result = await response.json();
                    if (result.share_url) {
                        navigator.clipboard.writeText(result.share_url);
                        this.showNotification('Share link copied to clipboard', 'success');
                    }
                } else {
                    this.showNotification('Failed to generate share link', 'error');
                }
            } catch (error) {
                console.error('Share error:', error);
                this.showNotification('Failed to generate share link', 'error');
            }
        },
        
        async downloadReport(report) {
            const blob = new Blob([JSON.stringify(report.data, null, 2)], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `${report.type}-report-${Date.now()}.json`;
            a.click();
            URL.revokeObjectURL(url);
        },
        
        deleteReport(report) {
            const index = this.generatedReports.indexOf(report);
            if (index > -1) {
                this.generatedReports.splice(index, 1);
                this.showNotification('Report deleted', 'info');
            }
        },
        
        updateAnalysisCharts() {
            // Update Chart.js charts for resource type and provider distribution
            // This would integrate with Chart.js library
            console.log('Updating analysis charts with:', this.stateAnalysis);
        },
        
        formatDate(dateString) {
            const date = new Date(dateString);
            return date.toLocaleString();
        },
        
        // State File Content Viewer Functions
        async viewStateContent(state) {
            this.selectedState = state;
            this.stateViewMode = 'json';
            this.startLoading(`Loading ${state.name}...`, 2000);
            
            try {
                // Encode the path as base64 for safe URL transmission
                const encodedPath = btoa(state.path);
                const response = await fetch(`/api/v1/state/content/${encodedPath}`);
                
                if (response.ok) {
                    const content = await response.json();
                    this.stateContent = content.data;
                    this.stateFileSize = content.size;
                    
                    // Load the initial view
                    await this.loadStateContent();
                    
                    // Open the modal
                    document.getElementById('stateContentModal').showModal();
                } else {
                    this.showNotification('Failed to load state file', 'error');
                }
            } catch (error) {
                console.error('Error loading state file:', error);
                this.showNotification('Failed to load state file', 'error');
            } finally {
                this.stopLoading();
            }
        },
        
        async loadStateContent() {
            switch (this.stateViewMode) {
                case 'json':
                    this.loadJSONView();
                    break;
                case 'tree':
                    this.loadTreeView();
                    break;
                case 'readable':
                    this.loadReadableView();
                    break;
                case 'resources':
                    this.loadResourcesView();
                    break;
                case 'diff':
                    this.loadDiffView();
                    break;
            }
        },
        
        loadJSONView() {
            if (!this.stateContent) return;
            
            const jsonString = JSON.stringify(this.stateContent, null, 2);
            
            // Apply search highlighting if there's a search term
            if (this.stateContentSearch) {
                const searchRegex = new RegExp(this.stateContentSearch, 'gi');
                this.highlightedStateContent = jsonString.replace(
                    searchRegex, 
                    match => `<mark class="bg-yellow-300 text-black">${match}</mark>`
                );
            } else {
                // Apply syntax highlighting
                this.highlightedStateContent = this.syntaxHighlightJSON(jsonString);
            }
        },
        
        syntaxHighlightJSON(json) {
            // Simple JSON syntax highlighting
            return json
                .replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, function (match) {
                    let cls = 'text-gray-600'; // number
                    if (/^"/.test(match)) {
                        if (/:$/.test(match)) {
                            cls = 'text-blue-600 font-semibold'; // key
                        } else {
                            cls = 'text-green-600'; // string
                        }
                    } else if (/true|false/.test(match)) {
                        cls = 'text-purple-600'; // boolean
                    } else if (/null/.test(match)) {
                        cls = 'text-red-600'; // null
                    }
                    return `<span class="${cls}">${match}</span>`;
                });
        },
        
        loadTreeView() {
            if (!this.stateContent) return;
            
            this.stateTreeView = this.generateTreeView(this.stateContent, 'state');
        },
        
        generateTreeView(obj, name, level = 0) {
            let html = '';
            const indent = '  '.repeat(level);
            
            if (typeof obj === 'object' && obj !== null) {
                const isArray = Array.isArray(obj);
                const icon = isArray ? 'fa-list' : 'fa-folder';
                const entries = isArray ? obj.map((v, i) => [i, v]) : Object.entries(obj);
                
                html += `<div class="tree-node" style="margin-left: ${level * 20}px;">`;
                html += `<span class="tree-label cursor-pointer hover:bg-base-200 px-2 py-1 rounded" onclick="this.nextElementSibling.classList.toggle('hidden')">`;
                html += `<i class="fas ${icon} mr-2 text-primary"></i>`;
                html += `<span class="font-semibold">${name}</span>`;
                html += ` <span class="text-xs opacity-60">(${entries.length} items)</span>`;
                html += `</span>`;
                html += `<div class="tree-children">`;
                
                for (const [key, value] of entries) {
                    html += this.generateTreeView(value, key, level + 1);
                }
                
                html += `</div></div>`;
            } else {
                const icon = typeof obj === 'string' ? 'fa-text' : 
                           typeof obj === 'number' ? 'fa-hashtag' : 
                           typeof obj === 'boolean' ? 'fa-toggle-on' : 'fa-question';
                
                html += `<div class="tree-leaf" style="margin-left: ${level * 20}px;">`;
                html += `<i class="fas ${icon} mr-2 text-gray-400"></i>`;
                html += `<span class="font-medium">${name}:</span> `;
                html += `<span class="text-sm font-mono">${JSON.stringify(obj)}</span>`;
                html += `</div>`;
            }
            
            return html;
        },
        
        loadReadableView() {
            // The readable view is handled by Alpine.js templates in the HTML
            // Just ensure the data is available
            if (this.stateContent) {
                this.filteredStateResources = this.stateContent.resources || [];
            }
        },
        
        loadResourcesView() {
            if (!this.stateContent || !this.stateContent.resources) {
                this.filteredStateResources = [];
                return;
            }
            
            // Filter resources based on search
            if (this.stateContentSearch) {
                const search = this.stateContentSearch.toLowerCase();
                this.filteredStateResources = this.stateContent.resources.filter(r => {
                    return (r.type && r.type.toLowerCase().includes(search)) ||
                           (r.name && r.name.toLowerCase().includes(search)) ||
                           (r.provider && r.provider.toLowerCase().includes(search));
                });
            } else {
                this.filteredStateResources = this.stateContent.resources;
            }
            
            // Add expanded property for UI
            this.filteredStateResources = this.filteredStateResources.map(r => ({
                ...r,
                expanded: false,
                id: `${r.type}.${r.name}`
            }));
        },
        
        loadDiffView() {
            // This would compare with previous version if available
            // For now, show a placeholder
            this.stateDiffView = `
                <div class="text-center py-8 opacity-60">
                    <i class="fas fa-code-branch text-4xl mb-4"></i>
                    <p>Diff view requires version history</p>
                    <p class="text-sm mt-2">This feature will show changes between state file versions</p>
                </div>
            `;
        },
        
        expandResource(resource) {
            resource.expanded = !resource.expanded;
        },
        
        filterStateContent() {
            // Reload the current view with filtering
            this.loadStateContent();
        },
        
        clearSearch() {
            this.stateContentSearch = '';
            this.filterStateContent();
        },
        
        async copyStateContent() {
            const content = JSON.stringify(this.stateContent, null, 2);
            
            try {
                await navigator.clipboard.writeText(content);
                this.showNotification('State file copied to clipboard', 'success');
            } catch (error) {
                console.error('Failed to copy:', error);
                this.showNotification('Failed to copy to clipboard', 'error');
            }
        },
        
        async downloadStateFile() {
            if (!this.stateContent || !this.selectedState) return;
            
            const content = JSON.stringify(this.stateContent, null, 2);
            const blob = new Blob([content], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = this.selectedState.name || 'terraform.tfstate';
            a.click();
            URL.revokeObjectURL(url);
        },
        
        formatStateContent() {
            if (this.stateViewMode === 'json') {
                // Trigger re-formatting
                this.loadJSONView();
                this.showNotification('JSON formatted', 'success');
            }
        },
        
        countManagedResources() {
            if (!this.stateContent || !this.stateContent.resources) return 0;
            return this.stateContent.resources.filter(r => r.mode === 'managed').length;
        },
        
        countDataResources() {
            if (!this.stateContent || !this.stateContent.resources) return 0;
            return this.stateContent.resources.filter(r => r.mode === 'data').length;
        },
        
        countModules() {
            if (!this.stateContent || !this.stateContent.resources) return 0;
            const modules = new Set(this.stateContent.resources.map(r => r.module || 'root'));
            return modules.size;
        },
        
        countProviders() {
            if (!this.stateContent || !this.stateContent.resources) return 0;
            const providers = new Set(this.stateContent.resources.map(r => r.provider));
            return providers.size;
        },
        
        formatFileSize(bytes) {
            if (!bytes) return '0 B';
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(1024));
            return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
        },
        
        // Calculate real costs based on cloud provider pricing
        async calculateRealCosts(statsData) {
            let totalCost = 0;
            
            try {
                // Fetch pricing data from backend
                const response = await fetch('/api/v1/pricing/calculate', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        resources: statsData.by_type || {},
                        providers: statsData.by_provider || {}
                    })
                });
                
                if (response.ok) {
                    const pricingData = await response.json();
                    totalCost = pricingData.total_monthly_cost || 0;
                    
                    // Store detailed pricing for later use
                    this.pricingDetails = pricingData;
                } else {
                    // Fallback to estimation based on resource types
                    totalCost = this.estimateCostsByResourceType(statsData);
                }
            } catch (error) {
                console.error('Error fetching real pricing:', error);
                // Fallback to estimation
                totalCost = this.estimateCostsByResourceType(statsData);
            }
            
            return totalCost;
        },
        
        // Estimate costs based on resource types and typical pricing
        estimateCostsByResourceType(statsData) {
            const resourcePricing = {
                // AWS pricing estimates (monthly)
                'aws_instance': 50,              // t3.medium average
                'aws_db_instance': 100,           // db.t3.medium average
                'aws_s3_bucket': 5,               // Storage cost estimate
                'aws_lambda_function': 10,        // Based on typical usage
                'aws_vpc': 0,                     // VPCs are free
                'aws_security_group': 0,          // Security groups are free
                'aws_subnet': 0,                  // Subnets are free
                'aws_route_table': 0,             // Route tables are free
                'aws_internet_gateway': 20,       // Data transfer costs
                'aws_nat_gateway': 45,            // NAT gateway hourly + data
                'aws_eip': 3.6,                   // Elastic IP when not attached
                'aws_alb': 25,                    // Application Load Balancer
                'aws_nlb': 22.5,                  // Network Load Balancer
                'aws_autoscaling_group': 0,       // ASG itself is free
                'aws_ecs_cluster': 0,             // Cluster management free
                'aws_ecs_service': 30,            // Based on task resources
                'aws_eks_cluster': 72,            // EKS cluster management
                'aws_cloudfront_distribution': 20, // CDN costs
                'aws_route53_zone': 0.5,          // Hosted zone cost
                'aws_dynamodb_table': 25,         // On-demand pricing estimate
                'aws_elasticache_cluster': 35,    // cache.t3.micro
                'aws_sqs_queue': 2,               // Based on message volume
                'aws_sns_topic': 1,               // Based on notifications
                'aws_kms_key': 1,                 // Key management
                
                // Azure pricing estimates (monthly)
                'azurerm_virtual_machine': 55,
                'azurerm_sql_database': 120,
                'azurerm_storage_account': 8,
                'azurerm_function_app': 12,
                'azurerm_virtual_network': 0,
                'azurerm_network_security_group': 0,
                'azurerm_subnet': 0,
                'azurerm_public_ip': 4,
                'azurerm_application_gateway': 28,
                'azurerm_kubernetes_cluster': 75,
                'azurerm_container_registry': 5,
                'azurerm_key_vault': 3,
                'azurerm_cosmosdb_account': 40,
                
                // GCP pricing estimates (monthly)
                'google_compute_instance': 48,
                'google_sql_database_instance': 95,
                'google_storage_bucket': 6,
                'google_cloudfunctions_function': 11,
                'google_compute_network': 0,
                'google_compute_firewall': 0,
                'google_compute_subnetwork': 0,
                'google_compute_address': 4,
                'google_compute_forwarding_rule': 20,
                'google_container_cluster': 70,
                'google_kms_crypto_key': 1.5,
                'google_bigquery_dataset': 5,
                
                // DigitalOcean pricing estimates (monthly)
                'digitalocean_droplet': 40,
                'digitalocean_database_cluster': 75,
                'digitalocean_spaces_bucket': 5,
                'digitalocean_kubernetes_cluster': 60,
                'digitalocean_loadbalancer': 12,
                'digitalocean_floating_ip': 4,
                'digitalocean_volume': 10,
                'digitalocean_cdn': 10,
                'digitalocean_domain': 0,
                'digitalocean_firewall': 0
            };
            
            let totalCost = 0;
            
            // Calculate based on resource types
            if (statsData.by_type) {
                Object.entries(statsData.by_type).forEach(([type, count]) => {
                    const unitCost = resourcePricing[type] || 10; // Default cost if type not found
                    totalCost += unitCost * count;
                });
            }
            
            // Add overhead for unaccounted resources
            const totalResources = statsData.total || 0;
            const accountedResources = Object.values(statsData.by_type || {}).reduce((a, b) => a + b, 0);
            const unaccountedResources = totalResources - accountedResources;
            
            if (unaccountedResources > 0) {
                totalCost += unaccountedResources * 15; // Average cost for unknown resources
            }
            
            return totalCost;
        },
        
        // Switch workspace (State Management)
        async switchWorkspace() {
            if (!this.selectedWorkspace) return;
            
            try {
                this.startLoading(`Switching to workspace ${this.selectedWorkspace}...`, 2000);
                
                const response = await fetch(`/api/v1/state/workspace`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ workspace: this.selectedWorkspace })
                });
                
                if (response.ok) {
                    this.showNotification(`Switched to workspace: ${this.selectedWorkspace}`, 'success');
                    await this.loadStateFiles();
                } else {
                    this.showNotification('Failed to switch workspace', 'error');
                }
            } catch (error) {
                console.error('Error switching workspace:', error);
                this.showNotification('Failed to switch workspace', 'error');
            } finally {
                this.stopLoading();
            }
        }
    };
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    console.log('DriftMgr Web App loaded');
});