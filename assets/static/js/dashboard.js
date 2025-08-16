// Dashboard Enhancement JavaScript

class DashboardManager {
    constructor() {
        this.ws = null;
        this.charts = {};
        this.notifications = [];
        this.autoRefreshInterval = null;
        this.currentUser = null;
        this.authToken = null;
        this.init();
    }

    init() {
        this.checkAuthentication();
        this.setupWebSocket();
        this.setupEventListeners();
        this.setupAutoRefresh();
        this.setupThemeToggle();
        this.setupCharts();
        this.setupNotifications();
        this.setupUserInterface();
    }

    checkAuthentication() {
        this.authToken = localStorage.getItem('auth_token');
        const userStr = localStorage.getItem('user');
        
        if (!this.authToken || !userStr) {
            window.location.href = '/login';
            return;
        }
        
        try {
            this.currentUser = JSON.parse(userStr);
            console.log('Authenticated as:', this.currentUser.username);
        } catch (error) {
            console.error('Error parsing user data:', error);
            this.logout();
        }
    }

    setupUserInterface() {
        // Update UI based on user role
        if (this.currentUser.role === 'readonly') {
            this.hideAdminElements();
        }
        
        // Update user info in navbar
        const userInfo = document.getElementById('user-info');
        if (userInfo) {
            userInfo.textContent = `${this.currentUser.username} (${this.currentUser.role})`;
        }
    }

    hideAdminElements() {
        // Hide admin-only elements for readonly users
        const adminElements = document.querySelectorAll('[data-admin-only]');
        adminElements.forEach(el => el.style.display = 'none');
        
        // Disable admin actions
        const adminButtons = document.querySelectorAll('[data-admin-action]');
        adminButtons.forEach(btn => {
            btn.disabled = true;
            btn.title = 'Admin access required';
        });
    }

    logout() {
        localStorage.removeItem('auth_token');
        localStorage.removeItem('user');
        window.location.href = '/login';
    }

    // Add authentication headers to all API requests
    async apiRequest(url, options = {}) {
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.authToken}`
            }
        };
        
        const finalOptions = { ...defaultOptions, ...options };
        finalOptions.headers = { ...defaultOptions.headers, ...options.headers };
        
        const response = await fetch(url, finalOptions);
        
        if (response.status === 401) {
            this.logout();
            return null;
        }
        
        return response;
    }

    setupWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.showNotification('Connected to real-time updates', 'success');
        };
        
        this.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.showNotification('Connection error', 'error');
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.showNotification('Connection lost', 'warning');
            // Attempt to reconnect after 5 seconds
            setTimeout(() => this.setupWebSocket(), 5000);
        };
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'drift_detected':
                this.handleDriftDetected(data.data);
                break;
            case 'drift_remediated':
                this.handleDriftRemediated(data.data);
                break;
            case 'cost_update':
                this.handleCostUpdate(data.data);
                break;
            case 'security_alert':
                this.handleSecurityAlert(data.data);
                break;
            case 'compliance_update':
                this.handleComplianceUpdate(data.data);
                break;
            default:
                console.log('Unknown message type:', data.type);
        }
    }

    handleDriftDetected(drift) {
        this.showNotification(`New drift detected: ${drift.ResourceID}`, 'warning');
        this.updateDriftTable(drift);
        this.updateMetrics();
    }

    handleDriftRemediated(drift) {
        this.showNotification(`Drift remediated: ${drift.ResourceID}`, 'success');
        this.updateDriftTable(drift);
        this.updateMetrics();
    }

    handleCostUpdate(costData) {
        this.updateCostCharts(costData);
        this.updateMetrics();
    }

    handleSecurityAlert(alert) {
        this.showNotification(`Security alert: ${alert.description}`, 'error');
        this.updateSecurityMetrics();
    }

    handleComplianceUpdate(compliance) {
        this.updateComplianceStatus(compliance);
        this.updateMetrics();
    }

    setupEventListeners() {
        // Smooth scrolling for navigation
        document.querySelectorAll('a[href^="#"]').forEach(anchor => {
            anchor.addEventListener('click', (e) => {
                e.preventDefault();
                const target = document.querySelector(anchor.getAttribute('href'));
                if (target) {
                    target.scrollIntoView({
                        behavior: 'smooth',
                        block: 'start'
                    });
                }
            });
        });

        // Table row click handlers
        document.querySelectorAll('.table tbody tr').forEach(row => {
            row.addEventListener('click', () => {
                this.showResourceDetails(row);
            });
        });

        // Action button handlers
        document.querySelectorAll('.btn-remediate').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.remediateDrift(btn.dataset.driftId);
            });
        });

        // Search functionality
        const searchInput = document.getElementById('search-input');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.filterTable(e.target.value);
            });
        }

        // Filter dropdowns
        document.querySelectorAll('.filter-dropdown').forEach(dropdown => {
            dropdown.addEventListener('change', (e) => {
                this.applyFilters();
            });
        });
    }

    setupAutoRefresh() {
        // Refresh data every 30 seconds
        this.autoRefreshInterval = setInterval(() => {
            this.refreshData();
        }, 30000);
    }

    setupThemeToggle() {
        const themeToggle = document.getElementById('theme-toggle');
        if (themeToggle) {
            themeToggle.addEventListener('click', () => {
                this.toggleTheme();
            });
        }
    }

    setupCharts() {
        this.initializeDriftChart();
        this.initializeCostChart();
        this.initializeSecurityChart();
        this.initializeComplianceChart();
    }

    initializeDriftChart() {
        const ctx = document.getElementById('driftChart');
        if (!ctx) return;

        this.charts.drift = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Drifts Detected',
                    data: [],
                    borderColor: 'rgb(75, 192, 192)',
                    backgroundColor: 'rgba(75, 192, 192, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'top',
                    },
                    title: {
                        display: true,
                        text: 'Drift Timeline'
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true
                    }
                }
            }
        });
    }

    initializeCostChart() {
        const ctx = document.getElementById('costChart');
        if (!ctx) return;

        this.charts.cost = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'Monthly Cost ($)',
                    data: [],
                    backgroundColor: 'rgba(54, 162, 235, 0.2)',
                    borderColor: 'rgb(54, 162, 235)',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'top',
                    },
                    title: {
                        display: true,
                        text: 'Cost Trends'
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true
                    }
                }
            }
        });
    }

    initializeSecurityChart() {
        const ctx = document.getElementById('securityChart');
        if (!ctx) return;

        this.charts.security = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['Secure', 'Vulnerable', 'Critical'],
                datasets: [{
                    data: [85, 10, 5],
                    backgroundColor: [
                        'rgba(75, 192, 192, 0.8)',
                        'rgba(255, 205, 86, 0.8)',
                        'rgba(255, 99, 132, 0.8)'
                    ],
                    borderWidth: 2
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'bottom',
                    }
                }
            }
        });
    }

    initializeComplianceChart() {
        const ctx = document.getElementById('complianceChart');
        if (!ctx) return;

        this.charts.compliance = new Chart(ctx, {
            type: 'radar',
            data: {
                labels: ['SOC2', 'ISO27001', 'PCI-DSS', 'HIPAA', 'GDPR'],
                datasets: [{
                    label: 'Compliance Score',
                    data: [95, 88, 92, 85, 90],
                    backgroundColor: 'rgba(75, 192, 192, 0.2)',
                    borderColor: 'rgb(75, 192, 192)',
                    borderWidth: 2
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    r: {
                        beginAtZero: true,
                        max: 100
                    }
                }
            }
        });
    }

    setupNotifications() {
        // Create notification container if it doesn't exist
        if (!document.getElementById('notification-container')) {
            const container = document.createElement('div');
            container.id = 'notification-container';
            container.className = 'toast-container';
            document.body.appendChild(container);
        }
    }

    showNotification(message, type = 'info', duration = 5000) {
        const container = document.getElementById('notification-container');
        if (!container) return;

        const toast = document.createElement('div');
        toast.className = `alert alert-${type} toast`;
        
        const icon = this.getNotificationIcon(type);
        toast.innerHTML = `
            ${icon}
            <span>${message}</span>
            <button class="btn btn-sm btn-ghost" onclick="this.parentElement.remove()">×</button>
        `;
        
        container.appendChild(toast);
        
        // Auto-remove after duration
        setTimeout(() => {
            if (toast.parentElement) {
                toast.remove();
            }
        }, duration);
    }

    getNotificationIcon(type) {
        const icons = {
            success: '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>',
            error: '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>',
            warning: '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"></path></svg>',
            info: '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>'
        };
        return icons[type] || icons.info;
    }

    updateDriftTable(drift) {
        const table = document.querySelector('#drifts table tbody');
        if (!table) return;

        // Add new row or update existing
        const existingRow = table.querySelector(`[data-drift-id="${drift.ID}"]`);
        if (existingRow) {
            this.updateTableRow(existingRow, drift);
        } else {
            this.addTableRow(table, drift);
        }
    }

    addTableRow(table, drift) {
        const row = document.createElement('tr');
        row.setAttribute('data-drift-id', drift.ID);
        row.className = 'fade-in';
        
        row.innerHTML = `
            <td>
                <div class="flex items-center space-x-3">
                    <div class="avatar">
                        <div class="mask mask-squircle w-12 h-12">
                            <div class="bg-${this.getSeverityColor(drift.Severity)} text-white flex items-center justify-center">
                                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
                                </svg>
                            </div>
                        </div>
                    </div>
                    <div>
                        <div class="font-bold">${drift.ResourceID}</div>
                        <div class="text-sm opacity-50">${drift.ResourceType}</div>
                    </div>
                </div>
            </td>
            <td>${drift.DriftType}</td>
            <td><div class="badge badge-outline">${drift.Provider}</div></td>
            <td>
                <div class="badge badge-${this.getSeverityColor(drift.Severity)} gap-1">
                    <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"></path>
                    </svg>
                    ${drift.Severity}
                </div>
            </td>
            <td>
                <div class="badge badge-${this.getStatusColor(drift.Status)} gap-1">
                    ${drift.Status}
                </div>
            </td>
            <td>
                <div class="dropdown dropdown-left">
                    <div tabindex="0" role="button" class="btn btn-ghost btn-xs">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"></path>
                        </svg>
                    </div>
                    <ul tabindex="0" class="dropdown-content menu p-2 shadow bg-base-100 rounded-box w-52">
                        <li><a onclick="dashboard.showResourceDetails('${drift.ID}')">View Details</a></li>
                        <li><a onclick="dashboard.remediateDrift('${drift.ID}')">Remediate</a></li>
                        <li><a onclick="dashboard.ignoreDrift('${drift.ID}')">Ignore</a></li>
                    </ul>
                </div>
            </td>
        `;
        
        table.insertBefore(row, table.firstChild);
    }

    updateTableRow(row, drift) {
        // Update status badge
        const statusCell = row.querySelector('td:nth-child(5) .badge');
        if (statusCell) {
            statusCell.className = `badge badge-${this.getStatusColor(drift.Status)} gap-1`;
            statusCell.textContent = drift.Status;
        }
        
        // Add update animation
        row.classList.add('slide-up');
        setTimeout(() => row.classList.remove('slide-up'), 300);
    }

    getSeverityColor(severity) {
        const colors = {
            'high': 'error',
            'medium': 'warning',
            'low': 'info'
        };
        return colors[severity] || 'info';
    }

    getStatusColor(status) {
        const colors = {
            'new': 'error',
            'remediated': 'success',
            'ignored': 'warning'
        };
        return colors[status] || 'info';
    }

    updateMetrics() {
        // Update summary statistics
        this.fetchAndUpdateSummary();
    }

    async fetchAndUpdateSummary() {
        try {
            const response = await fetch('/api/summary');
            const data = await response.json();
            this.updateSummaryCards(data);
        } catch (error) {
            console.error('Error fetching summary:', error);
        }
    }

    updateSummaryCards(data) {
        // Update total resources
        const totalResources = document.querySelector('[data-metric="total-resources"]');
        if (totalResources) {
            totalResources.textContent = data.TotalResources;
        }

        // Update drifts found
        const driftsFound = document.querySelector('[data-metric="drifts-found"]');
        if (driftsFound) {
            driftsFound.textContent = data.DriftsFound;
        }

        // Update security score
        const securityScore = document.querySelector('[data-metric="security-score"]');
        if (securityScore) {
            securityScore.textContent = data.SecurityScore + '%';
        }

        // Update cost savings
        const costSavings = document.querySelector('[data-metric="cost-savings"]');
        if (costSavings) {
            costSavings.textContent = '$' + data.CostSavings.toFixed(2);
        }
    }

    updateCostCharts(costData) {
        if (this.charts.cost) {
            this.charts.cost.data.labels = costData.labels;
            this.charts.cost.data.datasets[0].data = costData.values;
            this.charts.cost.update();
        }
    }

    updateSecurityMetrics() {
        // Update security chart and metrics
        this.fetchAndUpdateSecurityData();
    }

    async fetchAndUpdateSecurityData() {
        try {
            const response = await fetch('/api/security');
            const data = await response.json();
            
            if (this.charts.security) {
                this.charts.security.data.datasets[0].data = [
                    data.secure,
                    data.vulnerable,
                    data.critical
                ];
                this.charts.security.update();
            }
        } catch (error) {
            console.error('Error fetching security data:', error);
        }
    }

    updateComplianceStatus(compliance) {
        // Update compliance chart and status
        if (this.charts.compliance) {
            this.charts.compliance.data.datasets[0].data = compliance.scores;
            this.charts.compliance.update();
        }
    }

    showResourceDetails(resourceId) {
        // Show modal with resource details
        const modal = document.getElementById('resource-modal');
        if (modal) {
            modal.classList.add('modal-open');
            this.loadResourceDetails(resourceId);
        }
    }

    async loadResourceDetails(resourceId) {
        try {
            const response = await fetch(`/api/resources/${resourceId}`);
            const data = await response.json();
            this.populateResourceModal(data);
        } catch (error) {
            console.error('Error loading resource details:', error);
        }
    }

    populateResourceModal(data) {
        const modalContent = document.getElementById('resource-modal-content');
        if (modalContent) {
            modalContent.innerHTML = `
                <div class="card-body">
                    <h3 class="card-title">${data.Name}</h3>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <p><strong>Type:</strong> ${data.Type}</p>
                            <p><strong>Provider:</strong> ${data.Provider}</p>
                            <p><strong>Region:</strong> ${data.Region}</p>
                        </div>
                        <div>
                            <p><strong>Status:</strong> ${data.State}</p>
                            <p><strong>Created:</strong> ${new Date(data.CreatedAt).toLocaleDateString()}</p>
                            <p><strong>Updated:</strong> ${new Date(data.UpdatedAt).toLocaleDateString()}</p>
                        </div>
                    </div>
                </div>
            `;
        }
    }

    async remediateDrift(driftId) {
        try {
            const response = await fetch(`/api/drifts/${driftId}/remediate`, {
                method: 'POST'
            });
            
            if (response.ok) {
                this.showNotification('Drift remediation initiated', 'success');
            } else {
                this.showNotification('Failed to remediate drift', 'error');
            }
        } catch (error) {
            console.error('Error remediating drift:', error);
            this.showNotification('Error remediating drift', 'error');
        }
    }

    async ignoreDrift(driftId) {
        try {
            const response = await fetch(`/api/drifts/${driftId}/ignore`, {
                method: 'POST'
            });
            
            if (response.ok) {
                this.showNotification('Drift ignored', 'info');
            } else {
                this.showNotification('Failed to ignore drift', 'error');
            }
        } catch (error) {
            console.error('Error ignoring drift:', error);
            this.showNotification('Error ignoring drift', 'error');
        }
    }

    filterTable(searchTerm) {
        const rows = document.querySelectorAll('#drifts table tbody tr');
        rows.forEach(row => {
            const text = row.textContent.toLowerCase();
            const matches = text.includes(searchTerm.toLowerCase());
            row.style.display = matches ? '' : 'none';
        });
    }

    applyFilters() {
        // Apply selected filters to the table
        const filters = this.getActiveFilters();
        this.filterTableByFilters(filters);
    }

    getActiveFilters() {
        const filters = {};
        document.querySelectorAll('.filter-dropdown').forEach(dropdown => {
            if (dropdown.value) {
                filters[dropdown.name] = dropdown.value;
            }
        });
        return filters;
    }

    filterTableByFilters(filters) {
        const rows = document.querySelectorAll('#drifts table tbody tr');
        rows.forEach(row => {
            let show = true;
            
            Object.entries(filters).forEach(([key, value]) => {
                const cell = row.querySelector(`[data-${key}]`);
                if (cell && cell.getAttribute(`data-${key}`) !== value) {
                    show = false;
                }
            });
            
            row.style.display = show ? '' : 'none';
        });
    }

    toggleTheme() {
        const html = document.documentElement;
        const currentTheme = html.getAttribute('data-theme');
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        
        html.setAttribute('data-theme', newTheme);
        localStorage.setItem('theme', newTheme);
        
        this.showNotification(`Switched to ${newTheme} theme`, 'info');
    }

    async refreshData() {
        // Refresh all dashboard data
        await Promise.all([
            this.fetchAndUpdateSummary(),
            this.fetchAndUpdateSecurityData(),
            this.refreshCharts()
        ]);
    }

    async refreshCharts() {
        // Refresh chart data
        try {
            const [driftData, costData] = await Promise.all([
                fetch('/api/drift').then(r => r.json()),
                fetch('/api/costs').then(r => r.json())
            ]);
            
            this.updateDriftChart(driftData);
            this.updateCostCharts(costData);
        } catch (error) {
            console.error('Error refreshing charts:', error);
        }
    }

    updateDriftChart(data) {
        if (this.charts.drift) {
            this.charts.drift.data.labels = data.labels;
            this.charts.drift.data.datasets[0].data = data.values;
            this.charts.drift.update();
        }
    }

    destroy() {
        if (this.ws) {
            this.ws.close();
        }
        if (this.autoRefreshInterval) {
            clearInterval(this.autoRefreshInterval);
        }
        Object.values(this.charts).forEach(chart => {
            if (chart) {
                chart.destroy();
            }
        });
    }
}

// Initialize dashboard when DOM is loaded
let dashboard;
document.addEventListener('DOMContentLoaded', () => {
    dashboard = new DashboardManager();
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (dashboard) {
        dashboard.destroy();
    }
});
