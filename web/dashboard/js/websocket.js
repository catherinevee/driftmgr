/**
 * WebSocket client for real-time updates
 */
class DriftMgrWebSocket {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectInterval = 1000; // Start with 1 second
        this.maxReconnectInterval = 30000; // Max 30 seconds
        this.isConnected = false;
        this.messageHandlers = new Map();
        this.connectionCallbacks = [];
        this.disconnectionCallbacks = [];
        
        // Bind methods
        this.connect = this.connect.bind(this);
        this.disconnect = this.disconnect.bind(this);
        this.send = this.send.bind(this);
        this.onMessage = this.onMessage.bind(this);
        this.onConnection = this.onConnection.bind(this);
        this.onDisconnection = this.onDisconnection.bind(this);
    }

    /**
     * Connect to the WebSocket server
     */
    connect() {
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;
            
            console.log('Connecting to WebSocket:', wsUrl);
            
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = (event) => {
                console.log('WebSocket connected');
                this.isConnected = true;
                this.reconnectAttempts = 0;
                this.reconnectInterval = 1000;
                
                // Notify connection callbacks
                this.connectionCallbacks.forEach(callback => {
                    try {
                        callback(event);
                    } catch (error) {
                        console.error('Error in connection callback:', error);
                    }
                });
                
                // Show connection status
                this.updateConnectionStatus(true);
            };
            
            this.ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this.handleMessage(message);
                } catch (error) {
                    console.error('Error parsing WebSocket message:', error);
                }
            };
            
            this.ws.onclose = (event) => {
                console.log('WebSocket disconnected:', event.code, event.reason);
                this.isConnected = false;
                
                // Notify disconnection callbacks
                this.disconnectionCallbacks.forEach(callback => {
                    try {
                        callback(event);
                    } catch (error) {
                        console.error('Error in disconnection callback:', error);
                    }
                });
                
                // Show connection status
                this.updateConnectionStatus(false);
                
                // Attempt to reconnect if not a clean close
                if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
                    this.scheduleReconnect();
                }
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.updateConnectionStatus(false);
            };
            
        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
            this.scheduleReconnect();
        }
    }

    /**
     * Disconnect from the WebSocket server
     */
    disconnect() {
        if (this.ws) {
            this.ws.close(1000, 'Client disconnect');
            this.ws = null;
        }
        this.isConnected = false;
        this.updateConnectionStatus(false);
    }

    /**
     * Send a message to the server
     */
    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            try {
                this.ws.send(JSON.stringify(message));
                return true;
            } catch (error) {
                console.error('Error sending WebSocket message:', error);
                return false;
            }
        } else {
            console.warn('WebSocket not connected, cannot send message');
            return false;
        }
    }

    /**
     * Handle incoming messages
     */
    handleMessage(message) {
        console.log('Received WebSocket message:', message);
        
        // Call specific message handlers
        if (this.messageHandlers.has(message.type)) {
            const handlers = this.messageHandlers.get(message.type);
            handlers.forEach(handler => {
                try {
                    handler(message);
                } catch (error) {
                    console.error(`Error in message handler for ${message.type}:`, error);
                }
            });
        }
        
        // Handle specific message types
        switch (message.type) {
            case 'connection_established':
                this.handleConnectionEstablished(message);
                break;
            case 'drift_detection':
                this.handleDriftDetection(message);
                break;
            case 'remediation_update':
                this.handleRemediationUpdate(message);
                break;
            case 'resource_update':
                this.handleResourceUpdate(message);
                break;
            case 'state_update':
                this.handleStateUpdate(message);
                break;
            case 'backend_update':
                this.handleBackendUpdate(message);
                break;
            case 'system_alert':
                this.handleSystemAlert(message);
                break;
            case 'heartbeat':
                this.handleHeartbeat(message);
                break;
            case 'connection_stats':
                this.handleConnectionStats(message);
                break;
            default:
                console.log('Unknown message type:', message.type);
        }
    }

    /**
     * Register a message handler
     */
    onMessage(messageType, handler) {
        if (!this.messageHandlers.has(messageType)) {
            this.messageHandlers.set(messageType, []);
        }
        this.messageHandlers.get(messageType).push(handler);
    }

    /**
     * Register a connection callback
     */
    onConnection(callback) {
        this.connectionCallbacks.push(callback);
    }

    /**
     * Register a disconnection callback
     */
    onDisconnection(callback) {
        this.disconnectionCallbacks.push(callback);
    }

    /**
     * Schedule a reconnection attempt
     */
    scheduleReconnect() {
        this.reconnectAttempts++;
        const delay = Math.min(this.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1), this.maxReconnectInterval);
        
        console.log(`Scheduling reconnect attempt ${this.reconnectAttempts} in ${delay}ms`);
        
        setTimeout(() => {
            if (this.reconnectAttempts <= this.maxReconnectAttempts) {
                this.connect();
            } else {
                console.error('Max reconnection attempts reached');
                this.updateConnectionStatus(false, 'Connection failed');
            }
        }, delay);
    }

    /**
     * Update the connection status indicator
     */
    updateConnectionStatus(connected, message = null) {
        const statusElement = document.getElementById('ws-status');
        if (statusElement) {
            if (connected) {
                statusElement.className = 'ws-status connected';
                statusElement.textContent = 'Connected';
                statusElement.title = 'WebSocket connected';
            } else {
                statusElement.className = 'ws-status disconnected';
                statusElement.textContent = message || 'Disconnected';
                statusElement.title = 'WebSocket disconnected';
            }
        }
    }

    /**
     * Handle connection established message
     */
    handleConnectionEstablished(message) {
        console.log('Connection established:', message.data);
        this.showNotification('WebSocket Connected', 'Real-time updates are now active', 'success');
    }

    /**
     * Handle drift detection updates
     */
    handleDriftDetection(message) {
        console.log('Drift detection update:', message.data);
        this.showNotification('Drift Detection', 'New drift detected', 'info');
        
        // Refresh drift results if on the drift page
        if (window.currentPage === 'drift') {
            this.refreshDriftResults();
        }
    }

    /**
     * Handle remediation updates
     */
    handleRemediationUpdate(message) {
        console.log('Remediation update:', message.data);
        this.showNotification('Remediation Update', 'Remediation job updated', 'info');
        
        // Refresh remediation jobs if on the remediation page
        if (window.currentPage === 'remediation') {
            this.refreshRemediationJobs();
        }
    }

    /**
     * Handle resource updates
     */
    handleResourceUpdate(message) {
        console.log('Resource update:', message.data);
        this.showNotification('Resource Update', 'Resource information updated', 'info');
        
        // Refresh resources if on the resources page
        if (window.currentPage === 'resources') {
            this.refreshResources();
        }
    }

    /**
     * Handle state updates
     */
    handleStateUpdate(message) {
        console.log('State update:', message.data);
        this.showNotification('State Update', 'Terraform state updated', 'info');
        
        // Refresh state if on the state page
        if (window.currentPage === 'state') {
            this.refreshStateFiles();
        }
    }

    /**
     * Handle backend updates
     */
    handleBackendUpdate(message) {
        console.log('Backend update:', message.data);
        this.showNotification('Backend Update', 'Backend discovery updated', 'info');
        
        // Refresh backends if on the backend page
        if (window.currentPage === 'backend') {
            this.refreshBackends();
        }
    }

    /**
     * Handle system alerts
     */
    handleSystemAlert(message) {
        console.log('System alert:', message.data);
        this.showNotification('System Alert', message.data.message || 'System alert received', 'warning');
    }

    /**
     * Handle heartbeat messages
     */
    handleHeartbeat(message) {
        // Update last heartbeat time
        const heartbeatElement = document.getElementById('ws-heartbeat');
        if (heartbeatElement) {
            heartbeatElement.textContent = new Date(message.data.timestamp * 1000).toLocaleTimeString();
        }
    }

    /**
     * Handle connection statistics
     */
    handleConnectionStats(message) {
        console.log('Connection stats:', message.data);
        
        // Update stats display if available
        const statsElement = document.getElementById('ws-stats');
        if (statsElement) {
            statsElement.textContent = `${message.data.total_connections} connections`;
        }
    }

    /**
     * Show a notification
     */
    showNotification(title, message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <div class="notification-content">
                <h4>${title}</h4>
                <p>${message}</p>
            </div>
            <button class="notification-close" onclick="this.parentElement.remove()">×</button>
        `;
        
        // Add to notification container
        let container = document.getElementById('notification-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'notification-container';
            container.className = 'notification-container';
            document.body.appendChild(container);
        }
        
        container.appendChild(notification);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (notification.parentElement) {
                notification.remove();
            }
        }, 5000);
    }

    /**
     * Refresh methods for different pages
     */
    refreshDriftResults() {
        if (typeof loadDriftResults === 'function') {
            loadDriftResults();
        }
    }

    refreshRemediationJobs() {
        if (typeof loadRemediationJobs === 'function') {
            loadRemediationJobs();
        }
    }

    refreshResources() {
        if (typeof loadResources === 'function') {
            loadResources();
        }
    }

    refreshStateFiles() {
        if (typeof loadStateFiles === 'function') {
            loadStateFiles();
        }
    }

    refreshBackends() {
        if (typeof loadBackends === 'function') {
            loadBackends();
        }
    }
}

// Global WebSocket instance
window.driftMgrWS = new DriftMgrWebSocket();

// Auto-connect when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.driftMgrWS.connect();
});

// Clean up on page unload
window.addEventListener('beforeunload', () => {
    window.driftMgrWS.disconnect();
});
