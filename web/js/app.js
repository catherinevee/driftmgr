// DriftMgr v3.0 Web Application - Aligned with CLAUDE.md Architecture
function driftMgrV3App() {
    return {
        // Core State
        currentView: 'backends',
        selectedEnvironment: 'development',
        wsConnected: false,
        ws: null,
        eventBus: null,
        
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
        pushStateFile: null,
        pushTarget: 's3',
        pullSource: 's3',
        pullStateKey: '',
        
        // Terragrunt Support (v3.0 feature)
        terragruntModules: [],
        selectedTerragruntModule: null,
        
        // Compliance (v3.0 feature)
        policyViolations: [],
        complianceReports: {},
        
        // Monitoring (v3.0 enhanced)
        metrics: {
            discoveryRate: 0,
            driftRate: 0,
            remediationSuccess: 0,
            costImpact: 0
        },
        eventStream: [],
        webhooks: {
            eventbridge: false,
            eventgrid: false,
            pubsub: false
        },
        
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
            
            // Connect to enhanced WebSocket with event bridge
            this.connectWebSocket();
            
            // Initialize event bus for real-time updates
            this.initEventBus();
            
            // Load initial data based on architecture
            await this.loadInitialData();
            
            // Start continuous monitoring
            this.startMonitoring();
            
            // Register keyboard shortcuts
            this.registerKeyboardShortcuts();
        },
        
        // WebSocket connection with event bridge (v3.0)
        connectWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws/enhanced`; // Use enhanced endpoint
            
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                this.wsConnected = true;
                console.log('WebSocket connected to event bridge');
                this.addNotification('Connected to DriftMgr server', 'success');
                
                // Subscribe to event types
                this.ws.send(JSON.stringify({
                    type: 'subscribe',
                    events: ['discovery', 'drift', 'remediation', 'state', 'terragrunt', 'compliance']
                }));
            };
            
            this.ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            };
            
            this.ws.onclose = () => {
                this.wsConnected = false;
                console.log('WebSocket disconnected, reconnecting...');
                setTimeout(() => this.connectWebSocket(), 5000);
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.addNotification('Connection error', 'error');
            };
        },
        
        // Initialize event bus for internal communication
        initEventBus() {
            this.eventBus = {
                listeners: {},
                emit(event, data) {
                    if (this.listeners[event]) {
                        this.listeners[event].forEach(callback => callback(data));
                    }
                },
                on(event, callback) {
                    if (!this.listeners[event]) {
                        this.listeners[event] = [];
                    }
                    this.listeners[event].push(callback);
                }
            };
            
            // Register event handlers
            this.eventBus.on('discovery.started', (data) => {
                this.showProgress = true;
                this.progressTitle = 'Discovery in Progress';
                this.progressMessage = `Discovering resources in ${data.provider}...`;
            });
            
            this.eventBus.on('drift.detected', (data) => {
                this.addNotification(`Drift detected: ${data.resourceCount} resources`, 'warning');
                this.metrics.driftRate = data.driftPercentage;
            });
            
            this.eventBus.on('state.updated', (data) => {
                this.addNotification(`State updated: ${data.stateFile}`, 'info');
                this.loadStateFiles();
            });
            
            this.eventBus.on('terragrunt.dependency', (data) => {
                console.log('Terragrunt dependency resolved:', data);
                this.updateTerragruntModules();
            });
        },
        
        // Handle WebSocket messages from server
        handleWebSocketMessage(data) {
            console.log('WebSocket message:', data);
            
            // Add to event stream for monitoring
            this.eventStream.unshift({
                id: Date.now(),
                timestamp: new Date().toLocaleTimeString(),
                type: data.type,
                message: data.message || data.description
            });
            
            // Keep only last 100 events
            if (this.eventStream.length > 100) {
                this.eventStream = this.eventStream.slice(0, 100);
            }
            
            // Route to appropriate handler based on event type
            switch(data.type) {
                case 'backend.discovered':
                    this.handleBackendDiscovered(data);
                    break;
                case 'state.changed':
                    this.handleStateChanged(data);
                    break;
                case 'drift.detected':
                    this.handleDriftDetected(data);
                    break;
                case 'remediation.completed':
                    this.handleRemediationCompleted(data);
                    break;
                case 'terragrunt.update':
                    this.handleTerragruntUpdate(data);
                    break;
                case 'compliance.violation':
                    this.handleComplianceViolation(data);
                    break;
                case 'progress.update':
                    this.updateProgress(data.progress);
                    break;
                default:
                    console.log('Unhandled event type:', data.type);
            }
        },
        
        // Load initial data based on v3.0 architecture
        async loadInitialData() {
            try {
                // Load backend discovery data
                await this.loadBackends();
                
                // Load state files
                await this.loadStateFiles();
                
                // Load Terragrunt modules
                await this.loadTerragruntModules();
                
                // Load compliance status
                await this.loadComplianceStatus();
                
                // Load monitoring metrics
                await this.loadMetrics();
                
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
                
                const data = await response.json();
                this.backendsList = data.backends || [];
                
                // Update counts
                this.backends.s3.count = data.backends.filter(b => b.type === 's3').length;
                this.backends.azure.count = data.backends.filter(b => b.type === 'azurerm').length;
                this.backends.gcs.count = data.backends.filter(b => b.type === 'gcs').length;
                this.backends.local.count = data.backends.filter(b => b.type === 'local').length;
                
                // Update state counts
                for (const backend of this.backendsList) {
                    if (backend.type === 's3') this.backends.s3.states += backend.stateCount;
                    if (backend.type === 'azurerm') this.backends.azure.states += backend.stateCount;
                    if (backend.type === 'gcs') this.backends.gcs.states += backend.stateCount;
                    if (backend.type === 'local') this.backends.local.states += backend.stateCount;
                }
            } catch (error) {
                console.error('Error loading backends:', error);
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
                
                const result = await response.json();
                this.addNotification(`Discovered ${result.count} backends`, 'success');
                await this.loadBackends();
                
            } catch (error) {
                console.error('Backend discovery error:', error);
                this.addNotification('Backend discovery failed', 'error');
            } finally {
                this.showProgress = false;
            }
        },
        
        async discoverBackends() {
            await this.scanForBackends();
        },
        
        configureBackend() {
            // Open backend configuration modal
            console.log('Configure backend');
            // TODO: Implement backend configuration UI
        },
        
        async testConnections() {
            this.showProgress = true;
            this.progressTitle = 'Testing Connections';
            this.progressMessage = 'Verifying backend connectivity...';
            
            try {
                const response = await fetch('/api/v1/backends/test', {
                    method: 'POST'
                });
                
                if (!response.ok) throw new Error('Connection test failed');
                
                const results = await response.json();
                const successful = results.tests.filter(t => t.success).length;
                const total = results.tests.length;
                
                this.addNotification(`${successful}/${total} backends connected`, 
                    successful === total ? 'success' : 'warning');
                    
            } catch (error) {
                console.error('Connection test error:', error);
                this.addNotification('Connection test failed', 'error');
            } finally {
                this.showProgress = false;
            }
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
                
                const data = await response.json();
                this.stateFiles = data.states || [];
                
            } catch (error) {
                console.error('Error loading state files:', error);
            }
        },
        
        async loadStateFilesFromBackend(backend) {
            try {
                const response = await fetch(`/api/v1/state/list?backend=${backend.id}`);
                if (!response.ok) throw new Error('Failed to load state files');
                
                const data = await response.json();
                this.stateFiles = data.states || [];
                
            } catch (error) {
                console.error('Error loading state files:', error);
            }
        },
        
        async selectState(state) {
            this.selectedState = state;
            
            // Load detailed state information
            try {
                const response = await fetch(`/api/v1/state/details?path=${encodeURIComponent(state.path)}`);
                if (!response.ok) throw new Error('Failed to load state details');
                
                const details = await response.json();
                this.selectedState = { ...state, ...details };
                
            } catch (error) {
                console.error('Error loading state details:', error);
            }
        },
        
        async pushStateToBackend() {
            if (!this.pushStateFile) {
                this.addNotification('Please select a state file', 'warning');
                return;
            }
            
            this.showProgress = true;
            this.progressTitle = 'Pushing State';
            this.progressMessage = `Uploading state to ${this.pushTarget}...`;
            
            try {
                const formData = new FormData();
                formData.append('file', this.pushStateFile);
                formData.append('backend', this.pushTarget);
                
                const response = await fetch('/api/v1/state/push', {
                    method: 'POST',
                    body: formData
                });
                
                if (!response.ok) throw new Error('State push failed');
                
                this.addNotification('State pushed successfully', 'success');
                await this.loadStateFiles();
                
            } catch (error) {
                console.error('State push error:', error);
                this.addNotification('State push failed', 'error');
            } finally {
                this.showProgress = false;
            }
        },
        
        async pullStateFromBackend() {
            if (!this.pullStateKey) {
                this.addNotification('Please enter state key', 'warning');
                return;
            }
            
            this.showProgress = true;
            this.progressTitle = 'Pulling State';
            this.progressMessage = `Downloading state from ${this.pullSource}...`;
            
            try {
                const response = await fetch('/api/v1/state/pull', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        backend: this.pullSource,
                        key: this.pullStateKey
                    })
                });
                
                if (!response.ok) throw new Error('State pull failed');
                
                // Download the state file
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = this.pullStateKey.split('/').pop();
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                document.body.removeChild(a);
                
                this.addNotification('State pulled successfully', 'success');
                
            } catch (error) {
                console.error('State pull error:', error);
                this.addNotification('State pull failed', 'error');
            } finally {
                this.showProgress = false;
            }
        },
        
        async pullState(backend) {
            this.pullSource = backend.type;
            this.pullStateKey = backend.defaultKey || '';
            await this.pullStateFromBackend();
        },
        
        async pushState(backend) {
            this.pushTarget = backend.type;
            // Open file selector
            const input = document.createElement('input');
            input.type = 'file';
            input.accept = '.tfstate';
            input.onchange = (e) => {
                this.pushStateFile = e.target.files[0];
                this.pushStateToBackend();
            };
            input.click();
        },
        
        moveResource() {
            console.log('Move resource operation');
            // TODO: Implement resource move UI
        },
        
        removeResource() {
            console.log('Remove resource operation');
            // TODO: Implement resource removal UI
        },
        
        importResource() {
            console.log('Import resource operation');
            this.openImportWizard();
        },
        
        openImportWizard() {
            // TODO: Implement import wizard UI
            console.log('Opening import wizard');
        },
        
        // Terragrunt Functions (v3.0 feature)
        async loadTerragruntModules() {
            try {
                const response = await fetch('/api/v1/terragrunt/modules');
                if (!response.ok) throw new Error('Failed to load Terragrunt modules');
                
                const data = await response.json();
                this.terragruntModules = data.modules || [];
                
                // Initialize dependency graph if modules exist
                if (this.terragruntModules.length > 0) {
                    this.initializeDependencyGraph();
                }
                
            } catch (error) {
                console.error('Error loading Terragrunt modules:', error);
            }
        },
        
        selectTerragruntModule(module) {
            this.selectedTerragruntModule = module;
        },
        
        async runTerragruntPlan() {
            if (!this.selectedTerragruntModule) {
                this.addNotification('Please select a module', 'warning');
                return;
            }
            
            try {
                const response = await fetch('/api/v1/terragrunt/plan', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        module: this.selectedTerragruntModule.path
                    })
                });
                
                if (!response.ok) throw new Error('Terragrunt plan failed');
                
                const result = await response.json();
                this.addNotification('Plan completed', 'success');
                console.log('Plan result:', result);
                
            } catch (error) {
                console.error('Terragrunt plan error:', error);
                this.addNotification('Plan failed', 'error');
            }
        },
        
        async runTerragruntApply() {
            if (!this.selectedTerragruntModule) {
                this.addNotification('Please select a module', 'warning');
                return;
            }
            
            if (!confirm('Are you sure you want to apply changes?')) {
                return;
            }
            
            try {
                const response = await fetch('/api/v1/terragrunt/apply', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        module: this.selectedTerragruntModule.path
                    })
                });
                
                if (!response.ok) throw new Error('Terragrunt apply failed');
                
                const result = await response.json();
                this.addNotification('Apply completed', 'success');
                console.log('Apply result:', result);
                
            } catch (error) {
                console.error('Terragrunt apply error:', error);
                this.addNotification('Apply failed', 'error');
            }
        },
        
        async runTerragruntRunAll() {
            if (!confirm('Are you sure you want to run all modules?')) {
                return;
            }
            
            this.showProgress = true;
            this.progressTitle = 'Running All Modules';
            this.progressMessage = 'Executing Terragrunt run-all...';
            
            try {
                const response = await fetch('/api/v1/terragrunt/run-all', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        command: 'plan' // or 'apply'
                    })
                });
                
                if (!response.ok) throw new Error('Terragrunt run-all failed');
                
                const result = await response.json();
                this.addNotification('Run-all completed', 'success');
                console.log('Run-all result:', result);
                
            } catch (error) {
                console.error('Terragrunt run-all error:', error);
                this.addNotification('Run-all failed', 'error');
            } finally {
                this.showProgress = false;
            }
        },
        
        updateTerragruntModules() {
            this.loadTerragruntModules();
        },
        
        initializeDependencyGraph() {
            // Use vis.js to create dependency graph
            const container = document.getElementById('terragrunt-dependency-graph');
            if (!container) return;
            
            const nodes = this.terragruntModules.map(m => ({
                id: m.path,
                label: m.name,
                color: '#7B42BC'
            }));
            
            const edges = [];
            this.terragruntModules.forEach(m => {
                m.dependencies.forEach(dep => {
                    edges.push({
                        from: m.path,
                        to: dep,
                        arrows: 'to'
                    });
                });
            });
            
            const data = { nodes, edges };
            const options = {
                layout: {
                    hierarchical: {
                        direction: 'UD',
                        sortMethod: 'directed'
                    }
                },
                physics: false
            };
            
            new vis.Network(container, data, options);
        },
        
        // Compliance Functions (v3.0 feature)
        async loadComplianceStatus() {
            try {
                const response = await fetch('/api/v1/compliance/status');
                if (!response.ok) throw new Error('Failed to load compliance status');
                
                const data = await response.json();
                this.policyViolations = data.violations || [];
                this.complianceReports = data.reports || {};
                
            } catch (error) {
                console.error('Error loading compliance status:', error);
            }
        },
        
        async generateReport(type) {
            this.showProgress = true;
            this.progressTitle = 'Generating Report';
            this.progressMessage = `Creating ${type.toUpperCase()} compliance report...`;
            
            try {
                const response = await fetch(`/api/v1/compliance/report/${type}`, {
                    method: 'POST'
                });
                
                if (!response.ok) throw new Error('Report generation failed');
                
                // Download the report
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `${type}-compliance-report.pdf`;
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                document.body.removeChild(a);
                
                this.addNotification('Report generated successfully', 'success');
                
            } catch (error) {
                console.error('Report generation error:', error);
                this.addNotification('Report generation failed', 'error');
            } finally {
                this.showProgress = false;
            }
        },
        
        async remediateViolation(violation) {
            if (!confirm(`Remediate violation: ${violation.description}?`)) {
                return;
            }
            
            try {
                const response = await fetch('/api/v1/compliance/remediate', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        violationId: violation.id
                    })
                });
                
                if (!response.ok) throw new Error('Remediation failed');
                
                this.addNotification('Violation remediated', 'success');
                await this.loadComplianceStatus();
                
            } catch (error) {
                console.error('Remediation error:', error);
                this.addNotification('Remediation failed', 'error');
            }
        },
        
        handleComplianceViolation(data) {
            this.policyViolations.unshift(data.violation);
            this.addNotification(`Policy violation: ${data.violation.policy}`, 'warning');
        },
        
        getSeverityClass(severity) {
            const classes = {
                'critical': 'badge-error',
                'high': 'badge-warning',
                'medium': 'badge-info',
                'low': 'badge-success'
            };
            return classes[severity] || 'badge-ghost';
        },
        
        // Monitoring Functions (v3.0 enhanced)
        async loadMetrics() {
            try {
                const response = await fetch('/api/v1/monitoring/metrics');
                if (!response.ok) throw new Error('Failed to load metrics');
                
                const data = await response.json();
                this.metrics = data.metrics || this.metrics;
                
            } catch (error) {
                console.error('Error loading metrics:', error);
            }
        },
        
        startMonitoring() {
            // Update metrics every 30 seconds
            setInterval(() => {
                this.loadMetrics();
            }, 30000);
        },
        
        async configureWebhook(type) {
            try {
                const response = await fetch(`/api/v1/monitoring/webhook/${type}`, {
                    method: this.webhooks[type] ? 'DELETE' : 'POST',
                    headers: { 'Content-Type': 'application/json' }
                });
                
                if (!response.ok) throw new Error('Webhook configuration failed');
                
                this.webhooks[type] = !this.webhooks[type];
                this.addNotification(
                    `${type} webhook ${this.webhooks[type] ? 'enabled' : 'disabled'}`,
                    'success'
                );
                
            } catch (error) {
                console.error('Webhook configuration error:', error);
                this.addNotification('Webhook configuration failed', 'error');
            }
        },
        
        getEventTypeClass(type) {
            const classes = {
                'discovery': 'badge-primary',
                'drift': 'badge-warning',
                'remediation': 'badge-success',
                'error': 'badge-error',
                'info': 'badge-info'
            };
            return classes[type] || 'badge-ghost';
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
                
                const result = await response.json();
                this.addNotification(
                    `Detected ${result.driftCount} drifted resources`,
                    result.driftCount > 0 ? 'warning' : 'success'
                );
                
                // Navigate to drift view
                this.currentView = 'drift';
                
            } catch (error) {
                console.error('Drift detection error:', error);
                this.addNotification('Drift detection failed', 'error');
            } finally {
                this.showProgress = false;
            }
        },
        
        handleDriftDetected(data) {
            this.metrics.driftRate = data.driftPercentage;
            this.addNotification(`Drift detected: ${data.resourceCount} resources`, 'warning');
        },
        
        // Event Handlers
        handleBackendDiscovered(data) {
            this.loadBackends();
            this.addNotification(`Discovered ${data.backend} backend`, 'info');
        },
        
        handleStateChanged(data) {
            this.loadStateFiles();
            this.addNotification(`State changed: ${data.stateFile}`, 'info');
        },
        
        handleRemediationCompleted(data) {
            this.metrics.remediationSuccess = data.successRate;
            this.addNotification('Remediation completed', 'success');
        },
        
        handleTerragruntUpdate(data) {
            this.updateTerragruntModules();
        },
        
        // Progress Management
        updateProgress(value) {
            this.progressValue = value;
            if (value >= 100) {
                setTimeout(() => {
                    this.showProgress = false;
                }, 1000);
            }
        },
        
        cancelOperation() {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.ws.send(JSON.stringify({
                    type: 'cancel',
                    operation: 'current'
                }));
            }
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
        
        // Helper Functions
        getBackendBadgeClass(type) {
            const classes = {
                's3': 'badge-primary',
                'azurerm': 'badge-info',
                'gcs': 'badge-success',
                'local': 'badge-warning'
            };
            return classes[type] || 'badge-ghost';
        }
    };
}