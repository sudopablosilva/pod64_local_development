# POC BDD Web Dashboard

A modern, responsive web dashboard for monitoring the POC BDD microservices architecture with real-time updates and UX best practices.

## 🌟 Features

### Real-time Monitoring
- **Live Updates**: WebSocket-based real-time data streaming
- **Service Health**: Continuous monitoring of all microservices
- **Auto-refresh**: Fallback polling when WebSocket connection fails
- **Connection Status**: Visual indicators for connection health

### Modern UX/UI Design
- **Responsive Design**: Works seamlessly on desktop, tablet, and mobile
- **Accessibility**: WCAG 2.1 compliant with keyboard navigation and screen reader support
- **Dark Mode**: Automatic dark mode support based on system preferences
- **High Contrast**: Support for high contrast accessibility mode
- **Reduced Motion**: Respects user's motion preferences

### Interactive Dashboard
- **Tabbed Interface**: Organized views for different data types
- **Search & Filter**: Advanced filtering for executions and data
- **Click-to-Navigate**: Direct links to LocalStack admin interfaces
- **Loading States**: Smooth loading indicators and skeleton screens
- **Error Handling**: Graceful error recovery with retry mechanisms

### Performance Optimized
- **Lazy Loading**: Components load on demand
- **Caching**: Smart data caching to reduce API calls
- **Compression**: Gzip compression for faster loading
- **Security**: Helmet.js security headers and CORS protection

## 🏗️ Architecture

### Frontend (React)
- **React 18**: Latest React with concurrent features
- **Socket.IO Client**: Real-time communication
- **Lucide React**: Modern icon library
- **Date-fns**: Lightweight date formatting
- **CSS Modules**: Scoped styling with CSS custom properties

### Backend (Node.js)
- **Express.js**: Fast, minimalist web framework
- **Socket.IO**: Real-time bidirectional communication
- **Axios**: HTTP client for API calls
- **Helmet**: Security middleware
- **Compression**: Response compression

## 📊 Dashboard Sections

### 1. Overview Tab
- **System Health Summary**: Overall system status
- **Service Grid**: Visual status of all microservices
- **Key Metrics**: Important statistics at a glance
- **Health Indicators**: Color-coded status indicators

### 2. Executions Tab
- **Execution List**: All tracked executions with versioning
- **Search & Filter**: Find executions by name, status, or stage
- **Detailed View**: Complete execution metadata
- **Status Tracking**: Real-time execution status updates

### 3. Tables Tab
- **DynamoDB Tables**: All available database tables
- **Table Information**: Descriptions and purposes
- **Direct Access**: Click to open LocalStack admin interface
- **Usage Statistics**: Table-specific metrics

### 4. Queues Tab
- **SQS Queues**: All message queues in the system
- **Message Counts**: Visible and processing message counts
- **Queue Status**: Active, busy, or idle indicators
- **Flow Visualization**: Understanding message flow

## 🚀 Getting Started

### Prerequisites
- Node.js 18+ 
- npm or yarn
- Docker/Finch for containerized deployment

### Development Setup

1. **Install Dependencies**
   ```bash
   cd web-dashboard
   npm install
   npm run install:frontend
   ```

2. **Start Development Server**
   ```bash
   # Backend (port 3000)
   npm run dev
   
   # Frontend (port 3001) - in another terminal
   cd frontend
   npm start
   ```

3. **Access Dashboard**
   - Frontend: http://localhost:3001
   - Backend API: http://localhost:3000
   - Health Check: http://localhost:3000/health

### Production Deployment

1. **Build Application**
   ```bash
   npm run build
   ```

2. **Start Production Server**
   ```bash
   npm start
   ```

3. **Docker Deployment**
   ```bash
   # Build image
   docker build -t poc-bdd-dashboard .
   
   # Run container
   docker run -p 3000:3000 poc-bdd-dashboard
   ```

## 🔌 API Endpoints

### Dashboard Data
- `GET /api/dashboard` - Complete dashboard data
- `GET /api/services` - Service health status
- `GET /api/executions` - Execution data
- `GET /api/tables` - DynamoDB tables
- `GET /api/queues` - SQS queues

### WebSocket Events
- `dashboard-data` - Real-time data updates
- `connect` - Connection established
- `disconnect` - Connection lost

### Health & Status
- `GET /health` - Service health check
- WebSocket connection status in header

## 🎨 Customization

### Theming
The dashboard supports extensive theming through CSS custom properties:

```css
:root {
  --primary-color: #3b82f6;
  --success-color: #10b981;
  --warning-color: #f59e0b;
  --error-color: #ef4444;
  --background-color: #f8fafc;
}
```

### Component Styling
Each component has its own CSS file with:
- Responsive breakpoints
- Dark mode variants
- High contrast support
- Reduced motion alternatives

## 🔧 Configuration

### Environment Variables
```bash
# Server Configuration
PORT=3000
NODE_ENV=production

# API Endpoints (automatically configured)
JMI_ENDPOINT=http://localhost:4333
LOCALSTACK_ENDPOINT=http://localhost:4566
```

### WebSocket Configuration
```javascript
// Automatic fallback configuration
const socket = io(window.location.origin, {
  transports: ['websocket', 'polling']
});
```

## 📱 Mobile Support

The dashboard is fully responsive with:
- **Touch-friendly**: Large touch targets and gestures
- **Mobile Navigation**: Collapsible menus and tabs
- **Optimized Performance**: Reduced data usage on mobile
- **Offline Indicators**: Clear connection status

## ♿ Accessibility Features

- **Keyboard Navigation**: Full keyboard support
- **Screen Reader**: ARIA labels and semantic HTML
- **Focus Management**: Visible focus indicators
- **Color Contrast**: WCAG AA compliant colors
- **Motion Preferences**: Respects reduced motion settings

## 🔒 Security

- **CORS Protection**: Configured for specific origins
- **Security Headers**: Helmet.js security middleware
- **Input Validation**: Sanitized user inputs
- **Error Handling**: No sensitive data in error messages

## 🐛 Troubleshooting

### Common Issues

1. **WebSocket Connection Failed**
   - Check if backend is running on port 3000
   - Verify firewall settings
   - Dashboard will fallback to HTTP polling

2. **Data Not Loading**
   - Ensure JMI service is running on port 4333
   - Check LocalStack is accessible on port 4566
   - Verify network connectivity

3. **Build Errors**
   - Clear node_modules and reinstall
   - Check Node.js version (18+ required)
   - Verify all dependencies are installed

### Debug Mode
Enable debug logging:
```bash
DEBUG=dashboard:* npm start
```

## 🤝 Contributing

1. Follow the existing code style
2. Add tests for new features
3. Update documentation
4. Ensure accessibility compliance
5. Test on multiple devices and browsers

## 📄 License

This project is part of the POC BDD microservices architecture and follows the same licensing terms.
