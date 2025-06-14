const express = require('express');
const http = require('http');
const socketIo = require('socket.io');
const cors = require('cors');
const axios = require('axios');
const compression = require('compression');
const helmet = require('helmet');
const path = require('path');

const app = express();
const server = http.createServer(app);
const io = socketIo(server, {
  cors: {
    origin: "*",
    methods: ["GET", "POST"]
  }
});

const PORT = process.env.PORT || 3000;

// Determine if we're running in Docker or locally
const isDocker = process.env.NODE_ENV === 'production';
const baseUrl = isDocker ? 'http://jmi:8080' : 'http://localhost:4333';

// Security and performance middleware
app.use(helmet({
  contentSecurityPolicy: false // Disable for development
}));
app.use(compression());
app.use(cors());
app.use(express.json());

// Serve static files from frontend build
app.use(express.static(path.join(__dirname, 'frontend/build')));

// Service configuration
const SERVICES = [
  { name: 'Control-M', port: isDocker ? 8080 : 4333, host: isDocker ? 'control-m' : 'localhost', endpoint: '/health' },
  { name: 'JMI', port: isDocker ? 8080 : 4333, host: isDocker ? 'jmi' : 'localhost', endpoint: '/health' },
  { name: 'JMW', port: isDocker ? 8080 : 8080, host: isDocker ? 'jmw' : 'localhost', endpoint: '/health' },
  { name: 'JMR', port: isDocker ? 8080 : 8084, host: isDocker ? 'jmr' : 'localhost', endpoint: '/health' },
  { name: 'Scheduler', port: isDocker ? 8080 : 8085, host: isDocker ? 'scheduler-plugin' : 'localhost', endpoint: '/health' },
  { name: 'SPA', port: isDocker ? 8080 : 4444, host: isDocker ? 'spa' : 'localhost', endpoint: '/health' },
  { name: 'SPAQ', port: isDocker ? 8080 : 8087, host: isDocker ? 'spaq' : 'localhost', endpoint: '/health' }
];

// Data endpoints configuration
const DATA_ENDPOINTS = {
  executions: `${baseUrl}/executions`,
  tables: `${baseUrl}/tables`,
  queues: `${baseUrl}/queues`
};

// Cache for reducing API calls
let dataCache = {
  services: [],
  executions: { count: 0, executions: [] },
  tables: { count: 0, tables: [] },
  queues: { count: 0, queues: [] },
  lastUpdate: null
};

// Utility function to check service health
async function checkServiceHealth(service) {
  try {
    const url = `http://${service.host}:${service.port}${service.endpoint}`;
    const response = await axios.get(url, {
      timeout: 2000
    });
    return {
      ...service,
      status: 'online',
      responseTime: response.headers['x-response-time'] || 'N/A',
      lastCheck: new Date().toISOString()
    };
  } catch (error) {
    return {
      ...service,
      status: 'offline',
      error: error.message,
      lastCheck: new Date().toISOString()
    };
  }
}

// Utility function to normalize field names from API responses
function normalizeExecutionData(executions) {
  if (!Array.isArray(executions)) return [];
  
  return executions.map(execution => ({
    executionName: execution.ExecutionName || execution.executionName,
    originalName: execution.OriginalName || execution.originalName,
    executionUuid: execution.ExecutionUuid || execution.executionUuid,
    status: execution.Status || execution.status,
    stage: execution.Stage || execution.stage,
    processedBy: execution.ProcessedBy || execution.processedBy,
    version: execution.Version || execution.version,
    timestamp: execution.Timestamp || execution.timestamp,
    createdAt: execution.CreatedAt || execution.createdAt,
    updatedAt: execution.UpdatedAt || execution.updatedAt
  }));
}

// Utility function to normalize queue data
function normalizeQueueData(queues) {
  if (!Array.isArray(queues)) return [];
  
  return queues.map(queue => ({
    name: queue.name || queue.Name,
    visibleMessages: queue.visibleMessages || queue.visible || queue.VisibleMessages || '0',
    notVisibleMessages: queue.notVisibleMessages || queue.processing || queue.NotVisibleMessages || '0',
    url: queue.url || queue.Url
  }));
}

// Utility function to fetch data from endpoints
async function fetchEndpointData(url) {
  try {
    const response = await axios.get(url, { timeout: 3000 });
    const data = response.data;
    
    // Normalize data based on endpoint
    if (url.includes('/executions') && data.executions) {
      return {
        ...data,
        executions: normalizeExecutionData(data.executions)
      };
    } else if (url.includes('/queues') && data.queues) {
      return {
        ...data,
        queues: normalizeQueueData(data.queues)
      };
    }
    
    return data;
  } catch (error) {
    console.error(`Error fetching data from ${url}:`, error.message);
    return { error: error.message, count: 0 };
  }
}

// Function to collect all system data
async function collectSystemData() {
  try {
    // Check all services health
    const servicePromises = SERVICES.map(service => checkServiceHealth(service));
    const services = await Promise.all(servicePromises);

    // Fetch data from endpoints
    const [executions, tables, queues] = await Promise.all([
      fetchEndpointData(DATA_ENDPOINTS.executions),
      fetchEndpointData(DATA_ENDPOINTS.tables),
      fetchEndpointData(DATA_ENDPOINTS.queues)
    ]);

    // Update cache
    dataCache = {
      services,
      executions: executions || { count: 0, executions: [] },
      tables: tables || { count: 0, tables: [] },
      queues: queues || { count: 0, queues: [] },
      lastUpdate: new Date().toISOString()
    };

    return dataCache;
  } catch (error) {
    console.error('Error collecting system data:', error);
    return dataCache;
  }
}

// API Routes
app.get('/api/dashboard', async (req, res) => {
  try {
    const data = await collectSystemData();
    res.json(data);
  } catch (error) {
    res.status(500).json({ error: 'Failed to fetch dashboard data' });
  }
});

app.get('/api/services', async (req, res) => {
  try {
    const servicePromises = SERVICES.map(service => checkServiceHealth(service));
    const services = await Promise.all(servicePromises);
    res.json({ services, lastUpdate: new Date().toISOString() });
  } catch (error) {
    res.status(500).json({ error: 'Failed to fetch services data' });
  }
});

app.get('/api/executions', async (req, res) => {
  try {
    const data = await fetchEndpointData(DATA_ENDPOINTS.executions);
    res.json(data);
  } catch (error) {
    res.status(500).json({ error: 'Failed to fetch executions data' });
  }
});

app.get('/api/tables', async (req, res) => {
  try {
    const data = await fetchEndpointData(DATA_ENDPOINTS.tables);
    res.json(data);
  } catch (error) {
    res.status(500).json({ error: 'Failed to fetch tables data' });
  }
});

app.get('/api/queues', async (req, res) => {
  try {
    const data = await fetchEndpointData(DATA_ENDPOINTS.queues);
    res.json(data);
  } catch (error) {
    res.status(500).json({ error: 'Failed to fetch queues data' });
  }
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ 
    status: 'healthy', 
    service: 'web-dashboard',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    environment: isDocker ? 'docker' : 'local',
    baseUrl: baseUrl
  });
});

// Serve React app for all other routes
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'frontend/build/index.html'));
});

// Socket.IO for real-time updates
io.on('connection', (socket) => {
  console.log('Client connected:', socket.id);
  
  // Send initial data
  socket.emit('dashboard-data', dataCache);
  
  socket.on('disconnect', () => {
    console.log('Client disconnected:', socket.id);
  });
});

// Real-time data broadcasting
setInterval(async () => {
  const data = await collectSystemData();
  io.emit('dashboard-data', data);
}, 5000); // Update every 5 seconds

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully');
  server.close(() => {
    console.log('Process terminated');
  });
});

server.listen(PORT, () => {
  console.log(`🚀 POC BDD Web Dashboard running on port ${PORT}`);
  console.log(`📊 Dashboard available at: http://localhost:${PORT}`);
  console.log(`🔌 WebSocket server ready for real-time updates`);
  console.log(`🌐 Environment: ${isDocker ? 'Docker' : 'Local'}`);
  console.log(`🔗 Base URL: ${baseUrl}`);
});
