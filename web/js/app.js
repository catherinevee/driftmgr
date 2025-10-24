// DriftMgr v3.0 Web Application - Aligned with CLAUDE.md Architecture
function driftMgrV3App() {
    return {
        // Core State
        currentView: 'backends',
        selectedEnvironment: 'development',
        
        // View States
        stateTab: 'browser',
        
        // Backend Discovery (v3.0 feature)
        backends: {
            s3: { count: 0, states: 0 },
            azure: { count: 0, states: 0 },
            gcs: { count: 0, states: 0 },
            local: { count: 0, states: 0 }
        },
        backendsList: [],
        
        // State Management (v3.0 enhanced)
        stateFiles: [],
        selectedState: null,
        
        
        // Resources (v3.0 feature)
        resources: [],
        selectedProvider: 'aws',
        
        // Drift Detection (v3.0 feature)
        driftResults: [],
        
        
        
        
        // Notifications
        notifications: [],
        
        // Progress tracking
        showProgress: false,
        progressTitle: '',
        progressMessage: '',
        progressValue: 0,
        
        // Initialize the application
        async init() {
            console.log('Initializing DriftMgr v3.0...');
            
            
            // Load initial data based on architecture
            await this.loadInitialData();
            
            // Start continuous monitoring
            this.startMonitoring();
            
            // Register keyboard shortcuts
            this.registerKeyboardShortcuts();
        },
        
        // WebSocket connection removed
        connectWebSocket() {
            // WebSocket functionality removed
        },
        
        // Event bus removed
        initEventBus() {
            // Event bus functionality removed
        },
        
        // WebSocket message handling removed
        handleWebSocketMessage(data) {
            // WebSocket message handling removed
        },
        
        // Load initial data based on v3.0 architecture
        async loadInitialData() {
            try {
                // Load backend discovery data
                await this.loadBackends();
                
                // Load state files
                await this.loadStateFiles();
                
                
                
                
                // Load resources
                await this.loadResources();
                
                // Load drift results
                await this.loadDriftResults();
                
                
            } catch (error) {
                console.error('Error loading initial data:', error);
                this.addNotification('Failed to load initial data', 'error');
            }
        },
        
        // Backend Discovery Functions (v3.0)
        async loadBackends() {
            try {
                const response = await fetch('/api/v1/backends/list');
                if (!response.ok) throw new Error('Failed to load backends');
                
                const apiResponse = await response.json();
                
                // Handle standardized API response format
                if (!apiResponse.success) {
                    throw new Error(apiResponse.error?.message || 'API request failed');
                }
                
                const backends = apiResponse.data || [];
                this.backendsList = backends;
                
                // Update counts
                this.backends.s3.count = backends.filter(b => b.type === 's3').length;
                this.backends.azure.count = backends.filter(b => b.type === 'azurerm').length;
                this.backends.gcs.count = backends.filter(b => b.type === 'gcs').length;
                this.backends.local.count = backends.filter(b => b.type === 'local').length;
                
                // Update state counts
                for (const backend of this.backendsList) {
                    if (backend.type === 's3') this.backends.s3.states += backend.stateCount || 0;
                    if (backend.type === 'azurerm') this.backends.azure.states += backend.stateCount || 0;
                    if (backend.type === 'gcs') this.backends.gcs.states += backend.stateCount || 0;
                    if (backend.type === 'local') this.backends.local.states += backend.stateCount || 0;
                }
            } catch (error) {
                console.error('Error loading backends:', error);
                this.addNotification('Failed to load backends: ' + error.message, 'error');
            }
        },
        
        async scanForBackends() {
            this.showProgress = true;
            this.progressTitle = 'Scanning for Backends';
            this.progressMessage = 'Discovering Terraform backend configurations...';
            
            try {
                const response = await fetch('/api/v1/backends/discover', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        paths: ['/terraform', '~/infrastructure'],
                        recursive: true
                    })
                });
                
                if (!response.ok) throw new Error('Backend discovery failed');
                
                const apiResponse = await response.json();
                
                // Handle standardized API response format
                if (!apiResponse.success) {
                    throw new Error(apiResponse.error?.message || 'Backend discovery failed');
                }
                
                const result = apiResponse.data;
                this.addNotification(`Discovered ${result.count} backends`, 'success');
                await this.loadBackends();
                
            } catch (error) {
                console.error('Backend discovery error:', error);
                this.addNotification('Backend discovery failed: ' + error.message, 'error');
            } finally {
                this.showProgress = false;
            }
        },
        
        async discoverBackends() {
            await this.scanForBackends();
        },
        
        configureBackend() {
            // Open backend configuration modal
            if (!this.backendConfigUI) {
                this.backendConfigUI = new BackendConfigurationUI(this);
                this.backendConfigUI.init();
            }
            this.backendConfigUI.open();
        },
        
        
        exploreBackend(backend) {
            console.log('Exploring backend:', backend);
            // Navigate to backend explorer view
            this.currentView = 'states';
            this.loadStateFilesFromBackend(backend);
        },
        
        // State Management Functions (v3.0 enhanced)
        async loadStateFiles() {
            try {
                const response = await fetch('/api/v1/state/list');
                if (!response.ok) throw new Error('Failed to load state files');
                
                const apiResponse = await response.json();
                
                // Handle standardized API response format
                if (!apiResponse.success) {
                    throw new Error(apiResponse.error?.message || 'Failed to load state files');
                }
                
                this.stateFiles = apiResponse.data || [];
                
            } catch (error) {
                console.error('Error loading state files:', error);
                this.addNotification('Failed to load state files: ' + error.message, 'error');
            }
        },
        
        async loadStateFilesFromBackend(backend) {
            try {
                const response = await fetch(`/api/v1/state/list?backend=${backend.id}`);
                if (!response.ok) throw new Error('Failed to load state files');
                
                const apiResponse = await response.json();
                
                // Handle standardized API response format
                if (!apiResponse.success) {
                    throw new Error(apiResponse.error?.message || 'Failed to load state files');
                }
                
                this.stateFiles = apiResponse.data || [];
                
            } catch (error) {
                console.error('Error loading state files:', error);
                this.addNotification('Failed to load state files: ' + error.message, 'error');
            }
        },
        
        async selectState(state) {
            this.selectedState = state;
            
            // Load detailed state information
            try {
                const response = await fetch(`/api/v1/state/details?path=${encodeURIComponent(state.path)}`);
                if (!response.ok) throw new Error('Failed to load state details');
                
                const apiResponse = await response.json();
                
                // Handle standardized API response format
                if (!apiResponse.success) {
                    throw new Error(apiResponse.error?.message || 'Failed to load state details');
                }
                
                const details = apiResponse.data;
                this.selectedState = { ...state, ...details };
                
            } catch (error) {
                console.error('Error loading state details:', error);
                this.addNotification('Failed to load state details: ' + error.message, 'error');
            }
        },
        
        
        
        openImportWizard() {
            if (!this.importWizardUI) {
                this.importWizardUI = new ImportWizardUI(this);
                this.importWizardUI.init();
            }
            this.importWizardUI.open();
        },
        
        
        
        
        // Drift Detection Functions
        async runDriftDetection() {
            this.showProgress = true;
            this.progressTitle = 'Drift Detection';
            this.progressMessage = 'Analyzing infrastructure drift...';
            
            try {
                const response = await fetch('/api/v1/drift/detect', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        incremental: true, // Use v3.0 incremental discovery
                        useCache: true
                    })
                });
                
                if (!response.ok) throw new Error('Drift detection failed');
                
                const apiResponse = await response.json();
                
                // Handle standardized API response format
                if (!apiResponse.success) {
                    throw new Error(apiResponse.error?.message || 'Drift detection failed');
                }
                
                const result = apiResponse.data;
                this.addNotification(
                    `Detected ${result.driftCount} drifted resources`,
                    result.driftCount > 0 ? 'warning' : 'success'
                );
                
                // Navigate to drift view
                this.currentView = 'drift';
                
            } catch (error) {
                console.error('Drift detection error:', error);
                this.addNotification('Drift detection failed: ' + error.message, 'error');
            } finally {
                this.showProgress = false;
            }
        },
        
        
        cancelOperation() {
            // WebSocket cancellation removed
            this.showProgress = false;
        },
        
        // Notifications
        addNotification(message, type = 'info') {
            const notification = {
                id: Date.now(),
                message,
                type,
                timestamp: new Date().toLocaleTimeString()
            };
            
            this.notifications.unshift(notification);
            
            // Keep only last 10 notifications
            if (this.notifications.length > 10) {
                this.notifications = this.notifications.slice(0, 10);
            }
            
            // Auto-remove after 5 seconds
            setTimeout(() => {
                const index = this.notifications.findIndex(n => n.id === notification.id);
                if (index > -1) {
                    this.notifications.splice(index, 1);
                }
            }, 5000);
        },
        
        // Keyboard Shortcuts
        registerKeyboardShortcuts() {
            document.addEventListener('keydown', (e) => {
                // Alt+1 through Alt+8 for navigation
                if (e.altKey && e.key >= '1' && e.key <= '8') {
                    const views = ['backends', 'states', 'drift', 'resources', 'terragrunt', 'remediation', 'compliance', 'monitoring'];
                    const index = parseInt(e.key) - 1;
                    if (views[index]) {
                        this.currentView = views[index];
                    }
                }
                
                // Ctrl+D for discovery
                if (e.ctrlKey && e.key === 'd') {
                    e.preventDefault();
                    this.discoverBackends();
                }
                
                // Ctrl+R for drift detection
                if (e.ctrlKey && e.key === 'r') {
                    e.preventDefault();
                    this.runDriftDetection();
                }
            });
        },
        
        // Resources Functions
        async loadResources() {
            try {
                const response = await fetch(`/api/v1/resources?provider=${this.selectedProvider || 'aws'}`);
                if (!response.ok) throw new Error('Failed to load resources');
                
                const apiResponse = await response.json();
                
                // Handle standardized API response format
                if (!apiResponse.success) {
                    throw new Error(apiResponse.error?.message || 'Failed to load resources');
                }
                
                this.resources = apiResponse.data || [];
                
            } catch (error) {
                console.error('Resources loading error:', error);
                this.addNotification('Failed to load resources: ' + error.message, 'error');
            }
        },
        
        
        // Drift Detection Functions
        async loadDriftResults() {
            try {
                const response = await fetch('/api/v1/drift/results');
                if (!response.ok) throw new Error('Failed to load drift results');
                
                const apiResponse = await response.json();
                
                // Handle standardized API response format
                if (!apiResponse.success) {
                    throw new Error(apiResponse.error?.message || 'Failed to load drift results');
                }
                
                this.driftResults = apiResponse.data || [];
                
            } catch (error) {
                console.error('Drift results loading error:', error);
                this.addNotification('Failed to load drift results: ' + error.message, 'error');
            }
        },
        
        
        scheduleDriftDetection() {
            this.addNotification('Drift detection scheduled', 'info');
        },
        
        

        // Helper Functions
    };
}