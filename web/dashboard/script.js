// Dashboard JavaScript functionality

class DriftMgrDashboard {
    constructor() {
        this.currentPage = 'overview';
        this.charts = {};
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.initializeCharts();
        this.loadDashboardData();
        this.setupWebSocket();
    }

    setupEventListeners() {
        // Sidebar menu navigation
        document.querySelectorAll('.menu-item').forEach(item => {
            item.addEventListener('click', (e) => {
                const page = e.currentTarget.dataset.page;
                this.navigateToPage(page);
            });
        });

        // Sidebar toggle for mobile
        const sidebarToggle = document.querySelector('.sidebar-toggle');
        if (sidebarToggle) {
            sidebarToggle.addEventListener('click', () => {
                document.querySelector('.sidebar').classList.toggle('open');
            });
        }

        // Search functionality
        const searchInput = document.querySelector('.search-box input');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.handleSearch(e.target.value);
            });
        }

        // Notification click
        const notifications = document.querySelector('.notifications');
        if (notifications) {
            notifications.addEventListener('click', () => {
                this.showNotifications();
            });
        }

        // User menu
        const userMenu = document.querySelector('.user-menu');
        if (userMenu) {
            userMenu.addEventListener('click', () => {
                this.showUserMenu();
            });
        }

        // Button actions
        this.setupButtonActions();
    }

    setupButtonActions() {
        // Add Resource button
        const addResourceBtn = document.querySelector('[data-action="add-resource"]');
        if (addResourceBtn) {
            addResourceBtn.addEventListener('click', () => {
                this.showAddResourceModal();
            });
        }

        // Refresh button
        const refreshBtn = document.querySelector('[data-action="refresh"]');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.refreshCurrentPage();
            });
        }

        // Run Detection button
        const runDetectionBtn = document.querySelector('[data-action="run-detection"]');
        if (runDetectionBtn) {
            runDetectionBtn.addEventListener('click', () => {
                this.runDriftDetection();
            });
        }

        // Export Report button
        const exportReportBtn = document.querySelector('[data-action="export-report"]');
        if (exportReportBtn) {
            exportReportBtn.addEventListener('click', () => {
                this.exportReport();
            });
        }
    }

    navigateToPage(page) {
        // Update active menu item
        document.querySelectorAll('.menu-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`[data-page="${page}"]`).classList.add('active');

        // Update page content
        document.querySelectorAll('.page').forEach(pageEl => {
            pageEl.classList.remove('active');
        });
        document.getElementById(`${page}-page`).classList.add('active');

        // Update page title
        const pageTitle = document.getElementById('page-title');
        const titles = {
            'overview': 'Overview',
            'resources': 'Resources',
            'drift': 'Drift Detection',
            'health': 'Health Monitoring',
            'cost': 'Cost Analysis',
            'security': 'Security',
            'automation': 'Automation',
            'analytics': 'Analytics',
            'bi': 'Business Intelligence',
            'tenants': 'Multi-Tenant',
            'integrations': 'Integrations',
            'settings': 'Settings'
        };
        pageTitle.textContent = titles[page] || 'Dashboard';

        this.currentPage = page;
        this.loadPageData(page);
    }

    initializeCharts() {
        // Resource Distribution Chart
        const resourceCtx = document.getElementById('resourceChart');
        if (resourceCtx) {
            this.charts.resource = new Chart(resourceCtx, {
                type: 'doughnut',
                data: {
                    labels: ['AWS', 'Azure', 'GCP', 'DigitalOcean'],
                    datasets: [{
                        data: [45, 25, 20, 10],
                        backgroundColor: [
                            '#FF6384',
                            '#36A2EB',
                            '#FFCE56',
                            '#4BC0C0'
                        ],
                        borderWidth: 2,
                        borderColor: '#fff'
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: 'bottom'
                        }
                    }
                }
            });
        }

        // Cost Trend Chart
        const costCtx = document.getElementById('costChart');
        if (costCtx) {
            this.charts.cost = new Chart(costCtx, {
                type: 'line',
                data: {
                    labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'],
                    datasets: [{
                        label: 'Monthly Cost',
                        data: [8500, 9200, 8800, 10200, 11500, 12450],
                        borderColor: '#667eea',
                        backgroundColor: 'rgba(102, 126, 234, 0.1)',
                        borderWidth: 3,
                        fill: true,
                        tension: 0.4
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        y: {
                            beginAtZero: true,
                            ticks: {
                                callback: function(value) {
                                    return '$' + value.toLocaleString();
                                }
                            }
                        }
                    },
                    plugins: {
                        legend: {
                            display: false
                        }
                    }
                }
            });
        }

        // Health Status Chart
        const healthCtx = document.getElementById('healthChart');
        if (healthCtx) {
            this.charts.health = new Chart(healthCtx, {
                type: 'bar',
                data: {
                    labels: ['Database', 'Storage', 'Network', 'Compute', 'Security'],
                    datasets: [{
                        label: 'Health Score',
                        data: [95, 88, 92, 96, 89],
                        backgroundColor: [
                            '#28a745',
                            '#ffc107',
                            '#28a745',
                            '#28a745',
                            '#ffc107'
                        ],
                        borderWidth: 0
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        y: {
                            beginAtZero: true,
                            max: 100,
                            ticks: {
                                callback: function(value) {
                                    return value + '%';
                                }
                            }
                        }
                    },
                    plugins: {
                        legend: {
                            display: false
                        }
                    }
                }
            });
        }

        // Drift Detection Chart
        const driftCtx = document.getElementById('driftChart');
        if (driftCtx) {
            this.charts.drift = new Chart(driftCtx, {
                type: 'line',
                data: {
                    labels: ['Week 1', 'Week 2', 'Week 3', 'Week 4'],
                    datasets: [{
                        label: 'Drift Detections',
                        data: [12, 19, 15, 23],
                        borderColor: '#dc3545',
                        backgroundColor: 'rgba(220, 53, 69, 0.1)',
                        borderWidth: 3,
                        fill: true,
                        tension: 0.4
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    },
                    plugins: {
                        legend: {
                            display: false
                        }
                    }
                }
            });
        }
    }

    loadDashboardData() {
        // Simulate loading dashboard data
        this.showLoadingState();
        
        setTimeout(() => {
            this.updateStats();
            this.updateCharts();
            this.updateRecentActivity();
            this.hideLoadingState();
        }, 1000);
    }

    loadPageData(page) {
        // Load specific page data
        switch (page) {
            case 'resources':
                this.loadResourcesData();
                break;
            case 'drift':
                this.loadDriftData();
                break;
            case 'health':
                this.loadHealthData();
                break;
            case 'cost':
                this.loadCostData();
                break;
            default:
                break;
        }
    }

    updateStats() {
        // Update stat cards with real-time data
        const stats = {
            resources: Math.floor(Math.random() * 100) + 1200,
            drift: Math.floor(Math.random() * 10) + 20,
            health: Math.floor(Math.random() * 5) + 90,
            cost: Math.floor(Math.random() * 1000) + 12000
        };

        // Update stat numbers
        document.querySelectorAll('.stat-number').forEach((stat, index) => {
            const values = Object.values(stats);
            if (values[index]) {
                stat.textContent = values[index].toLocaleString();
            }
        });
    }

    updateCharts() {
        // Update charts with new data
        if (this.charts.resource) {
            this.charts.resource.data.datasets[0].data = [
                Math.floor(Math.random() * 20) + 40,
                Math.floor(Math.random() * 15) + 20,
                Math.floor(Math.random() * 10) + 15,
                Math.floor(Math.random() * 10) + 5
            ];
            this.charts.resource.update();
        }

        if (this.charts.cost) {
            const newData = Array.from({ length: 6 }, () => 
                Math.floor(Math.random() * 2000) + 8000
            );
            this.charts.cost.data.datasets[0].data = newData;
            this.charts.cost.update();
        }
    }

    updateRecentActivity() {
        // Update recent activity with new events
        const activities = [
            {
                icon: 'fas fa-exclamation-triangle',
                class: 'text-warning',
                title: 'Drift detected',
                description: 'in production-web-server-02',
                time: '1 minute ago'
            },
            {
                icon: 'fas fa-check-circle',
                class: 'text-success',
                title: 'Health check passed',
                description: 'for load-balancer-01',
                time: '3 minutes ago'
            },
            {
                icon: 'fas fa-dollar-sign',
                class: 'text-info',
                title: 'Cost optimization',
                description: 'saved $200 this week',
                time: '30 minutes ago'
            }
        ];

        const activityList = document.querySelector('.activity-list');
        if (activityList) {
            activityList.innerHTML = activities.map(activity => `
                <div class="activity-item">
                    <div class="activity-icon">
                        <i class="${activity.icon} ${activity.class}"></i>
                    </div>
                    <div class="activity-content">
                        <p><strong>${activity.title}</strong> ${activity.description}</p>
                        <span class="activity-time">${activity.time}</span>
                    </div>
                </div>
            `).join('');
        }
    }

    loadResourcesData() {
        // Load resources data
        console.log('Loading resources data...');
    }

    loadDriftData() {
        // Load drift detection data
        console.log('Loading drift data...');
    }

    loadHealthData() {
        // Load health monitoring data
        console.log('Loading health data...');
    }

    loadCostData() {
        // Load cost analysis data
        console.log('Loading cost data...');
    }

    handleSearch(query) {
        // Handle search functionality
        if (query.length > 2) {
            console.log('Searching for:', query);
            // Implement search logic
        }
    }

    showNotifications() {
        // Show notifications panel
        console.log('Showing notifications...');
    }

    showUserMenu() {
        // Show user menu
        console.log('Showing user menu...');
    }

    showAddResourceModal() {
        // Show add resource modal
        console.log('Showing add resource modal...');
    }

    refreshCurrentPage() {
        // Refresh current page data
        console.log('Refreshing page:', this.currentPage);
        this.loadPageData(this.currentPage);
    }

    runDriftDetection() {
        // Run drift detection
        console.log('Running drift detection...');
        this.showNotification('Drift detection started', 'info');
    }

    exportReport() {
        // Export report
        console.log('Exporting report...');
        this.showNotification('Report exported successfully', 'success');
    }

    setupWebSocket() {
        // Setup WebSocket connection for real-time updates
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/drift`;
        
        try {
            const ws = new WebSocket(wsUrl);
            
            ws.onopen = () => {
                console.log('WebSocket connected');
            };
            
            ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            };
            
            ws.onclose = () => {
                console.log('WebSocket disconnected');
                // Reconnect after 5 seconds
                setTimeout(() => this.setupWebSocket(), 5000);
            };
            
            ws.onerror = (error) => {
                console.error('WebSocket error:', error);
            };
        } catch (error) {
            console.error('Failed to setup WebSocket:', error);
        }
    }

    handleWebSocketMessage(data) {
        // Handle incoming WebSocket messages
        switch (data.type) {
            case 'drift_detected':
                this.handleDriftDetected(data);
                break;
            case 'health_update':
                this.handleHealthUpdate(data);
                break;
            case 'cost_update':
                this.handleCostUpdate(data);
                break;
            default:
                console.log('Unknown message type:', data.type);
        }
    }

    handleDriftDetected(data) {
        // Handle drift detection notification
        this.showNotification(`Drift detected in ${data.resource}`, 'warning');
        this.updateRecentActivity();
    }

    handleHealthUpdate(data) {
        // Handle health update
        this.updateStats();
        if (this.charts.health) {
            this.charts.health.update();
        }
    }

    handleCostUpdate(data) {
        // Handle cost update
        this.updateStats();
        if (this.charts.cost) {
            this.charts.cost.update();
        }
    }

    showNotification(message, type = 'info') {
        // Show notification toast
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <i class="fas fa-${this.getNotificationIcon(type)}"></i>
            <span>${message}</span>
            <button class="notification-close">&times;</button>
        `;
        
        document.body.appendChild(notification);
        
        // Auto remove after 5 seconds
        setTimeout(() => {
            notification.remove();
        }, 5000);
        
        // Close button
        notification.querySelector('.notification-close').addEventListener('click', () => {
            notification.remove();
        });
    }

    getNotificationIcon(type) {
        const icons = {
            'success': 'check-circle',
            'warning': 'exclamation-triangle',
            'error': 'times-circle',
            'info': 'info-circle'
        };
        return icons[type] || 'info-circle';
    }

    showLoadingState() {
        // Show loading state
        document.body.classList.add('loading');
    }

    hideLoadingState() {
        // Hide loading state
        document.body.classList.remove('loading');
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new DriftMgrDashboard();
});

// Add notification styles
const notificationStyles = `
    .notification {
        position: fixed;
        top: 20px;
        right: 20px;
        background: white;
        padding: 1rem 1.5rem;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        display: flex;
        align-items: center;
        gap: 0.5rem;
        z-index: 10000;
        animation: slideIn 0.3s ease;
    }
    
    .notification-success {
        border-left: 4px solid #28a745;
    }
    
    .notification-warning {
        border-left: 4px solid #ffc107;
    }
    
    .notification-error {
        border-left: 4px solid #dc3545;
    }
    
    .notification-info {
        border-left: 4px solid #17a2b8;
    }
    
    .notification-close {
        background: none;
        border: none;
        font-size: 1.2rem;
        cursor: pointer;
        color: #666;
        margin-left: 0.5rem;
    }
    
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    .loading {
        cursor: wait;
    }
    
    .loading * {
        pointer-events: none;
    }
`;

// Add notification styles to head
const styleSheet = document.createElement('style');
styleSheet.textContent = notificationStyles;
document.head.appendChild(styleSheet);
