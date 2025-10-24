// Interactive Visualizations for DriftMgr Dashboard

class DriftMgrVisualizations {
    constructor() {
        this.charts = {};
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.initializeCharts();
    }

    setupEventListeners() {
        // Chart refresh buttons
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('chart-refresh')) {
                const chartId = e.target.dataset.chart;
                this.refreshChart(chartId);
            }
        });

        // Chart export buttons
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('chart-export')) {
                const chartId = e.target.dataset.chart;
                this.exportChart(chartId);
            }
        });
    }

    initializeCharts() {
        this.createDriftTrendChart();
        this.createResourceDistributionChart();
        this.createCostAnalysisChart();
        this.createHealthScoreChart();
        this.createRemediationSuccessChart();
        this.createProviderComparisonChart();
    }

    // Drift Trend Chart
    createDriftTrendChart() {
        const ctx = document.getElementById('drift-trend-chart');
        if (!ctx) return;

        this.charts.driftTrend = new Chart(ctx, {
            type: 'line',
            data: {
                labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep'],
                datasets: [{
                    label: 'Configuration Drift',
                    data: [12, 19, 15, 25, 22, 18, 30, 28, 23],
                    borderColor: '#ef4444',
                    backgroundColor: 'rgba(239, 68, 68, 0.1)',
                    tension: 0.4,
                    fill: true
                }, {
                    label: 'Resource Drift',
                    data: [8, 12, 10, 18, 15, 12, 20, 17, 14],
                    borderColor: '#f59e0b',
                    backgroundColor: 'rgba(245, 158, 11, 0.1)',
                    tension: 0.4,
                    fill: true
                }, {
                    label: 'Tag Drift',
                    data: [25, 30, 28, 35, 32, 28, 40, 38, 33],
                    borderColor: '#10b981',
                    backgroundColor: 'rgba(16, 185, 129, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    title: {
                        display: true,
                        text: 'Drift Detection Trends Over Time'
                    },
                    legend: {
                        position: 'top',
                    },
                    tooltip: {
                        mode: 'index',
                        intersect: false,
                    }
                },
                scales: {
                    x: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Month'
                        }
                    },
                    y: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Number of Drifts'
                        },
                        beginAtZero: true
                    }
                },
                interaction: {
                    mode: 'nearest',
                    axis: 'x',
                    intersect: false
                }
            }
        });
    }

    // Resource Distribution Chart
    createResourceDistributionChart() {
        const ctx = document.getElementById('resource-distribution-chart');
        if (!ctx) return;

        this.charts.resourceDistribution = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['AWS EC2', 'AWS S3', 'AWS RDS', 'Azure VMs', 'Azure Storage', 'GCP Compute', 'Others'],
                datasets: [{
                    data: [35, 25, 15, 12, 8, 3, 2],
                    backgroundColor: [
                        '#3b82f6',
                        '#10b981',
                        '#f59e0b',
                        '#ef4444',
                        '#8b5cf6',
                        '#06b6d4',
                        '#6b7280'
                    ],
                    borderWidth: 2,
                    borderColor: '#ffffff'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    title: {
                        display: true,
                        text: 'Resource Distribution by Type'
                    },
                    legend: {
                        position: 'bottom',
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                const percentage = ((context.parsed / total) * 100).toFixed(1);
                                return `${context.label}: ${context.parsed} (${percentage}%)`;
                            }
                        }
                    }
                }
            }
        });
    }

    // Cost Analysis Chart
    createCostAnalysisChart() {
        const ctx = document.getElementById('cost-analysis-chart');
        if (!ctx) return;

        this.charts.costAnalysis = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep'],
                datasets: [{
                    label: 'Actual Cost',
                    data: [8500, 9200, 8800, 10500, 11200, 9800, 12500, 11800, 10200],
                    backgroundColor: '#3b82f6',
                    borderColor: '#2563eb',
                    borderWidth: 1
                }, {
                    label: 'Optimized Cost',
                    data: [7800, 8400, 8000, 9500, 10200, 8900, 11200, 10600, 9200],
                    backgroundColor: '#10b981',
                    borderColor: '#059669',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    title: {
                        display: true,
                        text: 'Monthly Cost Analysis ($)'
                    },
                    legend: {
                        position: 'top',
                    }
                },
                scales: {
                    x: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Month'
                        }
                    },
                    y: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Cost ($)'
                        },
                        beginAtZero: true,
                        ticks: {
                            callback: function(value) {
                                return '$' + value.toLocaleString();
                            }
                        }
                    }
                }
            }
        });
    }

    // Health Score Chart
    createHealthScoreChart() {
        const ctx = document.getElementById('health-score-chart');
        if (!ctx) return;

        this.charts.healthScore = new Chart(ctx, {
            type: 'radar',
            data: {
                labels: ['Security', 'Performance', 'Availability', 'Cost Optimization', 'Compliance', 'Automation'],
                datasets: [{
                    label: 'Current Score',
                    data: [85, 92, 88, 75, 90, 82],
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59, 130, 246, 0.2)',
                    pointBackgroundColor: '#3b82f6',
                    pointBorderColor: '#ffffff',
                    pointHoverBackgroundColor: '#ffffff',
                    pointHoverBorderColor: '#3b82f6'
                }, {
                    label: 'Target Score',
                    data: [95, 95, 95, 90, 95, 90],
                    borderColor: '#10b981',
                    backgroundColor: 'rgba(16, 185, 129, 0.2)',
                    pointBackgroundColor: '#10b981',
                    pointBorderColor: '#ffffff',
                    pointHoverBackgroundColor: '#ffffff',
                    pointHoverBorderColor: '#10b981'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    title: {
                        display: true,
                        text: 'Health Score Analysis'
                    },
                    legend: {
                        position: 'top',
                    }
                },
                scales: {
                    r: {
                        beginAtZero: true,
                        max: 100,
                        ticks: {
                            stepSize: 20
                        }
                    }
                }
            }
        });
    }

    // Remediation Success Chart
    createRemediationSuccessChart() {
        const ctx = document.getElementById('remediation-success-chart');
        if (!ctx) return;

        this.charts.remediationSuccess = new Chart(ctx, {
            type: 'pie',
            data: {
                labels: ['Successful', 'Failed', 'Pending', 'Cancelled'],
                datasets: [{
                    data: [75, 15, 8, 2],
                    backgroundColor: [
                        '#10b981',
                        '#ef4444',
                        '#f59e0b',
                        '#6b7280'
                    ],
                    borderWidth: 2,
                    borderColor: '#ffffff'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    title: {
                        display: true,
                        text: 'Remediation Job Success Rate'
                    },
                    legend: {
                        position: 'bottom',
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                const percentage = ((context.parsed / total) * 100).toFixed(1);
                                return `${context.label}: ${context.parsed} (${percentage}%)`;
                            }
                        }
                    }
                }
            }
        });
    }

    // Provider Comparison Chart
    createProviderComparisonChart() {
        const ctx = document.getElementById('provider-comparison-chart');
        if (!ctx) return;

        this.charts.providerComparison = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: ['AWS', 'Azure', 'GCP', 'DigitalOcean'],
                datasets: [{
                    label: 'Resources',
                    data: [450, 280, 150, 80],
                    backgroundColor: '#3b82f6',
                    borderColor: '#2563eb',
                    borderWidth: 1
                }, {
                    label: 'Drift Issues',
                    data: [25, 18, 12, 8],
                    backgroundColor: '#ef4444',
                    borderColor: '#dc2626',
                    borderWidth: 1
                }, {
                    label: 'Monthly Cost ($)',
                    data: [8500, 5200, 3200, 1800],
                    backgroundColor: '#10b981',
                    borderColor: '#059669',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    title: {
                        display: true,
                        text: 'Cloud Provider Comparison'
                    },
                    legend: {
                        position: 'top',
                    }
                },
                scales: {
                    x: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Cloud Provider'
                        }
                    },
                    y: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Count / Cost'
                        },
                        beginAtZero: true
                    }
                }
            }
        });
    }

    // Interactive Features
    refreshChart(chartId) {
        if (this.charts[chartId]) {
            // Simulate data refresh
            this.showLoadingState(chartId);
            
            setTimeout(() => {
                this.updateChartData(chartId);
                this.hideLoadingState(chartId);
            }, 1000);
        }
    }

    updateChartData(chartId) {
        const chart = this.charts[chartId];
        if (!chart) return;

        // Generate new random data based on chart type
        switch (chartId) {
            case 'driftTrend':
                chart.data.datasets.forEach(dataset => {
                    dataset.data = dataset.data.map(() => Math.floor(Math.random() * 30) + 5);
                });
                break;
            case 'resourceDistribution':
                chart.data.datasets[0].data = chart.data.datasets[0].data.map(() => Math.floor(Math.random() * 40) + 10);
                break;
            case 'costAnalysis':
                chart.data.datasets.forEach(dataset => {
                    dataset.data = dataset.data.map(() => Math.floor(Math.random() * 5000) + 5000);
                });
                break;
            case 'healthScore':
                chart.data.datasets[0].data = chart.data.datasets[0].data.map(() => Math.floor(Math.random() * 20) + 80);
                break;
            case 'remediationSuccess':
                const total = 100;
                const successful = Math.floor(Math.random() * 20) + 70;
                const failed = Math.floor(Math.random() * 10) + 10;
                const pending = Math.floor(Math.random() * 10) + 5;
                const cancelled = total - successful - failed - pending;
                chart.data.datasets[0].data = [successful, failed, pending, cancelled];
                break;
            case 'providerComparison':
                chart.data.datasets.forEach(dataset => {
                    dataset.data = dataset.data.map(() => Math.floor(Math.random() * 200) + 50);
                });
                break;
        }

        chart.update();
    }

    exportChart(chartId) {
        if (this.charts[chartId]) {
            const chart = this.charts[chartId];
            const url = chart.toBase64Image();
            
            // Create download link
            const link = document.createElement('a');
            link.download = `${chartId}-chart.png`;
            link.href = url;
            link.click();
        }
    }

    showLoadingState(chartId) {
        const canvas = document.getElementById(chartId);
        if (canvas) {
            canvas.style.opacity = '0.5';
            canvas.style.pointerEvents = 'none';
        }
    }

    hideLoadingState(chartId) {
        const canvas = document.getElementById(chartId);
        if (canvas) {
            canvas.style.opacity = '1';
            canvas.style.pointerEvents = 'auto';
        }
    }

    // Real-time Updates
    updateDriftTrend(newData) {
        if (this.charts.driftTrend) {
            const chart = this.charts.driftTrend;
            chart.data.labels.push(newData.timestamp);
            chart.data.datasets.forEach((dataset, index) => {
                dataset.data.push(newData.values[index]);
            });
            
            // Keep only last 12 data points
            if (chart.data.labels.length > 12) {
                chart.data.labels.shift();
                chart.data.datasets.forEach(dataset => {
                    dataset.data.shift();
                });
            }
            
            chart.update('none');
        }
    }

    updateHealthScore(newScore) {
        if (this.charts.healthScore) {
            const chart = this.charts.healthScore;
            chart.data.datasets[0].data = newScore;
            chart.update();
        }
    }

    // Chart Interaction Handlers
    setupChartInteractions() {
        // Add click handlers for chart elements
        Object.keys(this.charts).forEach(chartId => {
            const chart = this.charts[chartId];
            if (chart) {
                chart.canvas.addEventListener('click', (event) => {
                    const points = chart.getElementsAtEventForMode(event, 'nearest', { intersect: true }, true);
                    if (points.length > 0) {
                        this.handleChartClick(chartId, points[0]);
                    }
                });
            }
        });
    }

    handleChartClick(chartId, point) {
        console.log(`Chart ${chartId} clicked:`, point);
        
        // Show detailed information based on chart type
        switch (chartId) {
            case 'driftTrend':
                this.showDriftDetails(point.index);
                break;
            case 'resourceDistribution':
                this.showResourceDetails(point.index);
                break;
            case 'costAnalysis':
                this.showCostDetails(point.index);
                break;
            case 'providerComparison':
                this.showProviderDetails(point.index);
                break;
        }
    }

    showDriftDetails(monthIndex) {
        const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep'];
        const month = months[monthIndex];
        
        // Create modal with detailed drift information
        this.showModal(`Drift Details for ${month}`, `
            <div class="drift-details">
                <h3>Configuration Drift</h3>
                <ul>
                    <li>Instance type changes: 5</li>
                    <li>Security group modifications: 3</li>
                    <li>Tag updates: 8</li>
                </ul>
                <h3>Resource Drift</h3>
                <ul>
                    <li>Deleted resources: 2</li>
                    <li>New resources: 4</li>
                    <li>Modified resources: 6</li>
                </ul>
            </div>
        `);
    }

    showResourceDetails(resourceIndex) {
        const resources = ['AWS EC2', 'AWS S3', 'AWS RDS', 'Azure VMs', 'Azure Storage', 'GCP Compute', 'Others'];
        const resource = resources[resourceIndex];
        
        this.showModal(`Resource Details: ${resource}`, `
            <div class="resource-details">
                <h3>Resource Information</h3>
                <p><strong>Type:</strong> ${resource}</p>
                <p><strong>Count:</strong> ${Math.floor(Math.random() * 100) + 50}</p>
                <p><strong>Status:</strong> Active</p>
                <p><strong>Last Updated:</strong> ${new Date().toLocaleString()}</p>
            </div>
        `);
    }

    showCostDetails(monthIndex) {
        const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep'];
        const month = months[monthIndex];
        
        this.showModal(`Cost Details for ${month}`, `
            <div class="cost-details">
                <h3>Cost Breakdown</h3>
                <ul>
                    <li>Compute: $4,200</li>
                    <li>Storage: $1,800</li>
                    <li>Network: $900</li>
                    <li>Database: $1,200</li>
                    <li>Other: $800</li>
                </ul>
                <h3>Optimization Opportunities</h3>
                <ul>
                    <li>Reserved instances: Save $500/month</li>
                    <li>Unused storage: Save $200/month</li>
                    <li>Right-sizing: Save $300/month</li>
                </ul>
            </div>
        `);
    }

    showProviderDetails(providerIndex) {
        const providers = ['AWS', 'Azure', 'GCP', 'DigitalOcean'];
        const provider = providers[providerIndex];
        
        this.showModal(`Provider Details: ${provider}`, `
            <div class="provider-details">
                <h3>${provider} Statistics</h3>
                <p><strong>Total Resources:</strong> ${Math.floor(Math.random() * 500) + 100}</p>
                <p><strong>Active Resources:</strong> ${Math.floor(Math.random() * 400) + 80}</p>
                <p><strong>Monthly Cost:</strong> $${Math.floor(Math.random() * 10000) + 2000}</p>
                <p><strong>Drift Issues:</strong> ${Math.floor(Math.random() * 30) + 5}</p>
                <p><strong>Health Score:</strong> ${Math.floor(Math.random() * 20) + 80}%</p>
            </div>
        `);
    }

    showModal(title, content) {
        // Create modal element
        const modal = document.createElement('div');
        modal.className = 'modal show';
        modal.innerHTML = `
            <div class="modal-content">
                <div class="modal-header">
                    <h2>${title}</h2>
                    <button class="modal-close" onclick="this.closest('.modal').remove()">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <div class="modal-body">
                    ${content}
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        // Remove modal when clicking outside
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.remove();
            }
        });
    }
}

// Initialize visualizations when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    window.driftMgrVisualizations = new DriftMgrVisualizations();
});
