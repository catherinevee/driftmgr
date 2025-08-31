// State Manager - Single Source of Truth for DriftMgr
class StateManager {
    constructor() {
        // Core state - this is the ONLY source of truth
        this.state = {
            resources: new Map(), // Map of resourceId -> resource
            drifts: new Map(),    // Map of driftId -> drift
            providers: new Map(), // Map of providerId -> provider info
            jobs: new Map(),      // Map of jobId -> job status
            lastUpdate: null,
            cacheVersion: 1,
            dataFreshness: {
                resources: null,
                drifts: null,
                stats: null
            }
        };
        
        // Computed stats - always derived from state
        this.computedStats = {};
        
        // WebSocket message deduplication
        this.messageHashes = new Set();
        this.messageWindow = 5000; // 5 second window for dedup
        
        // Subscribers for state changes
        this.subscribers = new Set();
        
        // Validation schemas
        this.schemas = {
            resource: this.getResourceSchema(),
            drift: this.getDriftSchema(),
            stats: this.getStatsSchema()
        };
        
        // Initialize from localStorage if available
        this.loadFromLocalStorage();
    }
    
    // Resource Schema for validation
    getResourceSchema() {
        return {
            id: { type: 'string', required: true },
            name: { type: 'string', required: true },
            type: { type: 'string', required: true },
            provider: { type: 'string', required: true },
            region: { type: 'string', required: false },
            status: { type: 'string', required: false },
            tags: { type: 'object', required: false },
            properties: { type: 'object', required: false },
            createdAt: { type: 'string', required: false },
            modifiedAt: { type: 'string', required: false },
            managed: { type: 'boolean', required: false, default: true },
            driftStatus: { type: 'string', required: false }
        };
    }
    
    getDriftSchema() {
        return {
            id: { type: 'string', required: true },
            resourceId: { type: 'string', required: true },
            resourceType: { type: 'string', required: true },
            provider: { type: 'string', required: true },
            severity: { type: 'string', required: false, default: 'medium' },
            changes: { type: 'object', required: false },
            detectedAt: { type: 'string', required: true },
            status: { type: 'string', required: false, default: 'active' }
        };
    }
    
    getStatsSchema() {
        return {
            totalResources: { type: 'number', min: 0 },
            driftedResources: { type: 'number', min: 0 },
            compliantResources: { type: 'number', min: 0 },
            activeProviders: { type: 'number', min: 0 },
            unmanagedResources: { type: 'number', min: 0 },
            missingResources: { type: 'number', min: 0 },
            costEstimate: { type: 'number', min: 0 },
            securityIssues: { type: 'number', min: 0 },
            criticalDrifts: { type: 'number', min: 0 },
            remediableCount: { type: 'number', min: 0 },
            complianceScore: { type: 'number', min: 0, max: 100 }
        };
    }
    
    // Validate data against schema
    validate(data, schema) {
        const errors = [];
        const validated = {};
        
        for (const [key, rules] of Object.entries(schema)) {
            const value = data[key];
            
            // Check required fields
            if (rules.required && (value === undefined || value === null)) {
                errors.push(`Missing required field: ${key}`);
                continue;
            }
            
            // Skip optional fields that aren't present
            if (!rules.required && (value === undefined || value === null)) {
                if (rules.default !== undefined) {
                    validated[key] = rules.default;
                }
                continue;
            }
            
            // Type validation
            if (rules.type) {
                const actualType = Array.isArray(value) ? 'array' : typeof value;
                if (actualType !== rules.type) {
                    errors.push(`Invalid type for ${key}: expected ${rules.type}, got ${actualType}`);
                    continue;
                }
            }
            
            // Range validation for numbers
            if (rules.type === 'number') {
                if (rules.min !== undefined && value < rules.min) {
                    errors.push(`${key} value ${value} is below minimum ${rules.min}`);
                }
                if (rules.max !== undefined && value > rules.max) {
                    errors.push(`${key} value ${value} is above maximum ${rules.max}`);
                }
            }
            
            validated[key] = value;
        }
        
        // Copy over any additional fields not in schema
        for (const [key, value] of Object.entries(data)) {
            if (!schema[key]) {
                validated[key] = value;
            }
        }
        
        return { valid: errors.length === 0, errors, data: validated };
    }
    
    // WebSocket message deduplication
    deduplicateMessage(message) {
        const messageStr = JSON.stringify(message);
        const hash = this.hashString(messageStr);
        
        if (this.messageHashes.has(hash)) {
            return true; // Duplicate message
        }
        
        this.messageHashes.add(hash);
        
        // Clean old hashes after window expires
        setTimeout(() => {
            this.messageHashes.delete(hash);
        }, this.messageWindow);
        
        return false;
    }
    
    hashString(str) {
        let hash = 0;
        for (let i = 0; i < str.length; i++) {
            const char = str.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash; // Convert to 32-bit integer
        }
        return hash.toString();
    }
    
    // Update resources with validation and deduplication
    updateResources(resources, source = 'unknown') {
        if (!Array.isArray(resources)) {
            console.error('Resources must be an array');
            return false;
        }
        
        let updated = false;
        const validResources = [];
        
        for (const resource of resources) {
            const validation = this.validate(resource, this.schemas.resource);
            if (!validation.valid) {
                console.warn('Invalid resource:', validation.errors, resource);
                continue;
            }
            
            const validatedResource = validation.data;
            const existingResource = this.state.resources.get(validatedResource.id);
            
            // Check if resource has actually changed
            if (!existingResource || JSON.stringify(existingResource) !== JSON.stringify(validatedResource)) {
                this.state.resources.set(validatedResource.id, validatedResource);
                updated = true;
                validResources.push(validatedResource);
            }
        }
        
        if (updated) {
            this.state.lastUpdate = new Date().toISOString();
            this.state.dataFreshness.resources = new Date().toISOString();
            this.computeStats();
            this.notifySubscribers('resources', { source, count: validResources.length });
            this.saveToLocalStorage();
        }
        
        return updated;
    }
    
    // Update drifts with validation
    updateDrifts(drifts, source = 'unknown') {
        if (!Array.isArray(drifts)) {
            console.error('Drifts must be an array');
            return false;
        }
        
        let updated = false;
        
        for (const drift of drifts) {
            const validation = this.validate(drift, this.schemas.drift);
            if (!validation.valid) {
                console.warn('Invalid drift:', validation.errors, drift);
                continue;
            }
            
            const validatedDrift = validation.data;
            const existingDrift = this.state.drifts.get(validatedDrift.id);
            
            if (!existingDrift || JSON.stringify(existingDrift) !== JSON.stringify(validatedDrift)) {
                this.state.drifts.set(validatedDrift.id, validatedDrift);
                updated = true;
                
                // Update associated resource drift status
                const resource = this.state.resources.get(validatedDrift.resourceId);
                if (resource) {
                    resource.driftStatus = validatedDrift.status;
                }
            }
        }
        
        if (updated) {
            this.state.lastUpdate = new Date().toISOString();
            this.state.dataFreshness.drifts = new Date().toISOString();
            this.computeStats();
            this.notifySubscribers('drifts', { source, count: drifts.length });
            this.saveToLocalStorage();
        }
        
        return updated;
    }
    
    // Compute statistics from state (single source of truth)
    computeStats() {
        const resources = Array.from(this.state.resources.values());
        const drifts = Array.from(this.state.drifts.values());
        const providers = new Set();
        const resourceTypes = new Set();
        const regions = new Set();
        
        let unmanagedCount = 0;
        let missingCount = 0;
        let compliantCount = 0;
        let securityIssues = 0;
        let criticalDrifts = 0;
        let remediableCount = 0;
        let costEstimate = 0;
        
        // Process resources
        for (const resource of resources) {
            if (resource.provider) providers.add(resource.provider);
            if (resource.type) resourceTypes.add(resource.type);
            if (resource.region) regions.add(resource.region);
            
            if (resource.managed === false) unmanagedCount++;
            if (resource.status === 'missing' || resource.status === 'deleted') missingCount++;
            if (!resource.driftStatus || resource.driftStatus === 'compliant') compliantCount++;
            
            // Calculate cost based on resource type and provider
            const baseCost = this.getResourceCost(resource.type, resource.provider);
            costEstimate += baseCost;
        }
        
        // Process drifts
        const activeDrifts = drifts.filter(d => d.status === 'active');
        for (const drift of activeDrifts) {
            if (drift.severity === 'critical') criticalDrifts++;
            if (drift.severity === 'high' && this.isSecurityRelated(drift)) securityIssues++;
            if (drift.remediable) remediableCount++;
        }
        
        // Calculate compliance score
        const totalResources = resources.length;
        const driftedResources = activeDrifts.length;
        let complianceScore = 100;
        if (totalResources > 0) {
            complianceScore = Math.round(((totalResources - driftedResources) / totalResources) * 100);
        }
        
        this.computedStats = {
            totalResources,
            driftedResources,
            compliantResources: compliantCount,
            activeProviders: providers.size,
            configuredProviders: Array.from(providers),
            unmanagedResources: unmanagedCount,
            missingResources: missingCount,
            costEstimate: Math.round(costEstimate),
            securityIssues,
            criticalDrifts,
            remediableCount,
            complianceScore,
            lastScanTime: this.state.lastUpdate,
            resourceTypes: Array.from(resourceTypes),
            regions: Array.from(regions)
        };
        
        this.state.dataFreshness.stats = new Date().toISOString();
        return this.computedStats;
    }
    
    // Get resource cost (replace with actual pricing data)
    getResourceCost(resourceType, provider) {
        const costMap = {
            'aws': {
                'ec2_instance': 50,
                'rds_instance': 100,
                's3_bucket': 5,
                'lambda_function': 2,
                'default': 10
            },
            'azure': {
                'virtual_machine': 55,
                'sql_database': 95,
                'storage_account': 8,
                'function_app': 3,
                'default': 12
            },
            'gcp': {
                'compute_instance': 48,
                'cloud_sql': 90,
                'storage_bucket': 6,
                'cloud_function': 2,
                'default': 11
            },
            'default': {
                'default': 15
            }
        };
        
        const providerCosts = costMap[provider] || costMap['default'];
        return providerCosts[resourceType] || providerCosts['default'];
    }
    
    // Check if drift is security-related
    isSecurityRelated(drift) {
        const securityKeywords = ['security', 'encryption', 'public', 'firewall', 'access', 'permission', 'role', 'policy'];
        const driftStr = JSON.stringify(drift).toLowerCase();
        return securityKeywords.some(keyword => driftStr.includes(keyword));
    }
    
    // Get current stats (always computed from state)
    getStats() {
        return this.computedStats;
    }
    
    // Get resources with optional filtering
    getResources(filter = {}) {
        let resources = Array.from(this.state.resources.values());
        
        if (filter.provider) {
            resources = resources.filter(r => r.provider === filter.provider);
        }
        if (filter.region) {
            resources = resources.filter(r => r.region === filter.region);
        }
        if (filter.type) {
            resources = resources.filter(r => r.type === filter.type);
        }
        if (filter.search) {
            const searchLower = filter.search.toLowerCase();
            resources = resources.filter(r => 
                r.name?.toLowerCase().includes(searchLower) ||
                r.id?.toLowerCase().includes(searchLower) ||
                r.type?.toLowerCase().includes(searchLower)
            );
        }
        
        return resources;
    }
    
    // Get drifts with optional filtering
    getDrifts(filter = {}) {
        let drifts = Array.from(this.state.drifts.values());
        
        if (filter.status) {
            drifts = drifts.filter(d => d.status === filter.status);
        }
        if (filter.severity) {
            drifts = drifts.filter(d => d.severity === filter.severity);
        }
        if (filter.provider) {
            drifts = drifts.filter(d => d.provider === filter.provider);
        }
        
        return drifts;
    }
    
    // Clear all data
    clearAll() {
        this.state.resources.clear();
        this.state.drifts.clear();
        this.state.providers.clear();
        this.state.jobs.clear();
        this.state.lastUpdate = null;
        this.state.cacheVersion++;
        this.computedStats = {};
        this.notifySubscribers('clear', {});
        this.saveToLocalStorage();
    }
    
    // Subscribe to state changes
    subscribe(callback) {
        this.subscribers.add(callback);
        return () => this.subscribers.delete(callback);
    }
    
    // Notify subscribers of state changes
    notifySubscribers(type, data) {
        for (const subscriber of this.subscribers) {
            try {
                subscriber(type, data, this);
            } catch (error) {
                console.error('Error notifying subscriber:', error);
            }
        }
    }
    
    // Save state to localStorage
    saveToLocalStorage() {
        try {
            const stateToSave = {
                resources: Array.from(this.state.resources.entries()),
                drifts: Array.from(this.state.drifts.entries()),
                providers: Array.from(this.state.providers.entries()),
                jobs: Array.from(this.state.jobs.entries()),
                lastUpdate: this.state.lastUpdate,
                cacheVersion: this.state.cacheVersion,
                dataFreshness: this.state.dataFreshness
            };
            localStorage.setItem('driftmgr_state', JSON.stringify(stateToSave));
            localStorage.setItem('driftmgr_state_version', this.state.cacheVersion.toString());
        } catch (error) {
            console.error('Failed to save state to localStorage:', error);
        }
    }
    
    // Load state from localStorage
    loadFromLocalStorage() {
        try {
            const savedState = localStorage.getItem('driftmgr_state');
            if (!savedState) return;
            
            const parsed = JSON.parse(savedState);
            
            // Restore maps from arrays
            this.state.resources = new Map(parsed.resources || []);
            this.state.drifts = new Map(parsed.drifts || []);
            this.state.providers = new Map(parsed.providers || []);
            this.state.jobs = new Map(parsed.jobs || []);
            this.state.lastUpdate = parsed.lastUpdate;
            this.state.cacheVersion = parsed.cacheVersion || 1;
            this.state.dataFreshness = parsed.dataFreshness || {};
            
            // Compute stats from loaded state
            this.computeStats();
            
            console.log('Loaded state from localStorage:', {
                resources: this.state.resources.size,
                drifts: this.state.drifts.size,
                version: this.state.cacheVersion
            });
        } catch (error) {
            console.error('Failed to load state from localStorage:', error);
        }
    }
    
    // Get data freshness information
    getDataFreshness() {
        const now = new Date();
        const freshness = {};
        
        for (const [key, timestamp] of Object.entries(this.state.dataFreshness)) {
            if (timestamp) {
                const age = now - new Date(timestamp);
                const minutes = Math.floor(age / 60000);
                freshness[key] = {
                    timestamp,
                    age: minutes,
                    status: minutes < 5 ? 'fresh' : minutes < 30 ? 'recent' : 'stale'
                };
            } else {
                freshness[key] = { timestamp: null, age: null, status: 'unknown' };
            }
        }
        
        return freshness;
    }
    
    // Merge state from WebSocket (with deduplication and validation)
    mergeWebSocketUpdate(type, data) {
        // Check for duplicate message
        if (this.deduplicateMessage({ type, data })) {
            console.log('Duplicate WebSocket message ignored:', type);
            return false;
        }
        
        switch (type) {
            case 'resources_updated':
                return this.updateResources(data.resources || [], 'websocket');
                
            case 'drift_detected':
                const drift = data.drift || data;
                return this.updateDrifts([drift], 'websocket');
                
            case 'stats_updated':
                // Stats should be computed, not directly set
                console.warn('Direct stats update ignored - stats are computed from resources/drifts');
                return false;
                
            case 'discovery_complete':
                if (data.resources) {
                    return this.updateResources(data.resources, 'discovery');
                }
                return false;
                
            default:
                console.log('Unhandled WebSocket update type:', type);
                return false;
        }
    }
}

// Export for use in main app
window.StateManager = StateManager;