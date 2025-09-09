# DriftMgr Web Dashboard

A modern, responsive web dashboard for DriftMgr Cloud Health Management platform.

## Features

### 🎨 Modern UI/UX
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Dark/Light Theme**: Automatic theme switching based on system preferences
- **Interactive Charts**: Real-time data visualization with Chart.js
- **Smooth Animations**: CSS transitions and JavaScript animations
- **Accessibility**: WCAG 2.1 compliant design

### 📊 Dashboard Pages
- **Overview**: High-level metrics and recent activity
- **Resources**: Infrastructure resource management
- **Drift Detection**: Configuration drift monitoring and alerts
- **Health Monitoring**: System health status and metrics
- **Cost Analysis**: Cost tracking and optimization insights
- **Security**: Security compliance and policy management
- **Automation**: Workflow and rule management
- **Analytics**: Predictive analytics and insights
- **Business Intelligence**: Dashboards, reports, and data export
- **Multi-Tenant**: Tenant and account management
- **Integrations**: External service integrations
- **Settings**: System configuration and preferences

### 🔄 Real-time Updates
- **WebSocket Integration**: Real-time data updates
- **Live Notifications**: Toast notifications for important events
- **Auto-refresh**: Automatic data refresh with configurable intervals
- **Event Streaming**: Real-time event processing

### 🎯 Interactive Features
- **Search**: Global search across resources and alerts
- **Filtering**: Advanced filtering and sorting options
- **Pagination**: Efficient data pagination for large datasets
- **Modal Dialogs**: Interactive forms and detailed views
- **Context Menus**: Right-click actions and shortcuts

## Getting Started

### Prerequisites
- DriftMgr API server running on port 8080
- Modern web browser with JavaScript enabled
- Node.js (optional, for development)

### Quick Start

1. **Start the Web Dashboard**:
   ```bash
   driftmgr web start
   ```

2. **Access the Dashboard**:
   - Open your browser and navigate to `http://localhost:3000`
   - The dashboard will automatically connect to the API server

3. **Check Status**:
   ```bash
   driftmgr web status
   ```

### Manual Setup

1. **Build the Dashboard**:
   ```bash
   driftmgr web build
   ```

2. **Start API Server**:
   ```bash
   driftmgr api server start
   ```

3. **Serve Static Files**:
   ```bash
   # Using Python
   python -m http.server 3000
   
   # Using Node.js
   npx serve -s . -l 3000
   
   # Using Go
   go run cmd/web/main.go
   ```

## Architecture

### Frontend Structure
```
web/
├── dashboard/
│   ├── index.html          # Main dashboard page
│   ├── styles.css          # CSS styles and themes
│   └── script.js           # JavaScript functionality
├── assets/                 # Static assets (images, icons)
├── components/             # Reusable UI components
└── README.md              # This file
```

### Key Components

#### HTML Structure
- **Semantic HTML5**: Proper semantic markup for accessibility
- **Responsive Grid**: CSS Grid and Flexbox for layout
- **Component-based**: Modular HTML structure

#### CSS Architecture
- **CSS Custom Properties**: CSS variables for theming
- **Mobile-first**: Responsive design approach
- **Component Styles**: Scoped CSS for components
- **Animation Library**: Smooth transitions and animations

#### JavaScript Features
- **ES6+ Syntax**: Modern JavaScript features
- **Class-based Architecture**: Object-oriented design
- **Event-driven**: Event handling and delegation
- **Chart.js Integration**: Interactive data visualization
- **WebSocket Client**: Real-time communication

## Configuration

### Environment Variables
```bash
# API Server Configuration
DRIFTMGR_API_HOST=localhost
DRIFTMGR_API_PORT=8080
DRIFTMGR_API_SSL=false

# Web Server Configuration
DRIFTMGR_WEB_HOST=localhost
DRIFTMGR_WEB_PORT=3000
DRIFTMGR_WEB_SSL=false

# Feature Flags
DRIFTMGR_ENABLE_WEBSOCKET=true
DRIFTMGR_ENABLE_NOTIFICATIONS=true
DRIFTMGR_ENABLE_ANALYTICS=true
```

### Customization

#### Themes
The dashboard supports custom themes through CSS variables:

```css
:root {
  --primary-color: #667eea;
  --secondary-color: #764ba2;
  --success-color: #28a745;
  --warning-color: #ffc107;
  --danger-color: #dc3545;
  --info-color: #17a2b8;
  --light-color: #f8f9fa;
  --dark-color: #343a40;
}
```

#### Branding
Update the sidebar header in `index.html`:
```html
<div class="sidebar-header">
    <h2><i class="fas fa-cloud"></i> Your Brand</h2>
    <p>Your Tagline</p>
</div>
```

## API Integration

### REST API Endpoints
The dashboard integrates with the DriftMgr API:

- `GET /api/v1/resources` - List resources
- `GET /api/v1/drift/reports` - Get drift reports
- `GET /api/v1/health` - Health status
- `GET /api/v1/cost/analysis` - Cost analysis
- `GET /api/v1/security/scan` - Security scan
- `GET /api/v1/automation/workflows` - List workflows
- `GET /api/v1/analytics/models` - Analytics models
- `GET /api/v1/bi/dashboards` - BI dashboards
- `GET /api/v1/tenants` - List tenants
- `GET /api/v1/integrations` - List integrations

### WebSocket Events
Real-time updates via WebSocket:

- `drift_detected` - New drift detection
- `health_update` - Health status change
- `cost_update` - Cost data update
- `security_alert` - Security alert
- `automation_event` - Automation workflow event
- `analytics_update` - Analytics data update

## Development

### Local Development

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/catherinevee/driftmgr.git
   cd driftmgr
   ```

2. **Start Development Server**:
   ```bash
   # Start API server
   driftmgr api server start
   
   # Start web dashboard
   driftmgr web start
   ```

3. **Make Changes**:
   - Edit HTML, CSS, or JavaScript files
   - Changes are reflected immediately
   - Use browser developer tools for debugging

### Building for Production

1. **Build the Dashboard**:
   ```bash
   driftmgr web build
   ```

2. **Deploy**:
   - Copy the `web/` directory to your web server
   - Configure your web server to serve static files
   - Ensure API server is accessible

### Testing

1. **Manual Testing**:
   - Test all dashboard pages
   - Verify responsive design
   - Check browser compatibility
   - Test WebSocket functionality

2. **Automated Testing**:
   ```bash
   # Run API tests
   go test ./internal/api/...
   
   # Run integration tests
   go test ./cmd/web/...
   ```

## Browser Support

### Supported Browsers
- **Chrome**: 80+
- **Firefox**: 75+
- **Safari**: 13+
- **Edge**: 80+

### Required Features
- **ES6 Support**: Arrow functions, classes, modules
- **CSS Grid**: Modern layout support
- **WebSocket**: Real-time communication
- **Fetch API**: HTTP requests
- **Local Storage**: Data persistence

## Performance

### Optimization Features
- **Lazy Loading**: Load components on demand
- **Image Optimization**: Compressed and optimized images
- **CSS Minification**: Minified CSS for production
- **JavaScript Bundling**: Optimized JavaScript bundles
- **Caching**: Browser caching for static assets

### Performance Metrics
- **First Contentful Paint**: < 1.5s
- **Largest Contentful Paint**: < 2.5s
- **Cumulative Layout Shift**: < 0.1
- **First Input Delay**: < 100ms

## Security

### Security Features
- **HTTPS Support**: SSL/TLS encryption
- **CORS Configuration**: Cross-origin resource sharing
- **Content Security Policy**: XSS protection
- **Input Validation**: Client-side validation
- **Authentication**: JWT token support

### Best Practices
- **Secure Headers**: Security headers configuration
- **Input Sanitization**: Prevent XSS attacks
- **API Authentication**: Secure API communication
- **Session Management**: Secure session handling

## Troubleshooting

### Common Issues

1. **Dashboard Not Loading**:
   - Check if API server is running
   - Verify port 3000 is available
   - Check browser console for errors

2. **WebSocket Connection Failed**:
   - Ensure API server supports WebSocket
   - Check firewall settings
   - Verify WebSocket URL configuration

3. **Charts Not Displaying**:
   - Check Chart.js library loading
   - Verify data format
   - Check browser console for errors

4. **Responsive Design Issues**:
   - Test on different screen sizes
   - Check CSS media queries
   - Verify viewport meta tag

### Debug Mode

Enable debug mode by adding `?debug=true` to the URL:
```
http://localhost:3000?debug=true
```

This will:
- Show additional console logs
- Display performance metrics
- Enable development tools
- Show API request/response details

## Contributing

### Development Guidelines
1. **Code Style**: Follow existing code patterns
2. **Testing**: Add tests for new features
3. **Documentation**: Update documentation
4. **Performance**: Optimize for performance
5. **Accessibility**: Ensure accessibility compliance

### Pull Request Process
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests and documentation
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- **Documentation**: [DriftMgr Docs](https://docs.driftmgr.com)
- **Issues**: [GitHub Issues](https://github.com/catherinevee/driftmgr/issues)
- **Discussions**: [GitHub Discussions](https://github.com/catherinevee/driftmgr/discussions)
- **Email**: support@driftmgr.com
