import React, { useState, useEffect } from 'react';
import io from 'socket.io-client';
import Dashboard from './components/Dashboard';
import Header from './components/Header';
import ErrorBoundary from './components/ErrorBoundary';
import LoadingSpinner from './components/LoadingSpinner';
import './App.css';

function App() {
  const [dashboardData, setDashboardData] = useState(null);
  const [connectionStatus, setConnectionStatus] = useState('connecting');
  const [socket, setSocket] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    // Initialize socket connection
    const newSocket = io(window.location.origin, {
      transports: ['websocket', 'polling']
    });

    newSocket.on('connect', () => {
      console.log('Connected to dashboard server');
      setConnectionStatus('connected');
      setError(null);
    });

    newSocket.on('disconnect', () => {
      console.log('Disconnected from dashboard server');
      setConnectionStatus('disconnected');
    });

    newSocket.on('connect_error', (err) => {
      console.error('Connection error:', err);
      setConnectionStatus('error');
      setError('Failed to connect to dashboard server');
    });

    newSocket.on('dashboard-data', (data) => {
      setDashboardData(data);
      setError(null);
    });

    setSocket(newSocket);

    // Cleanup on unmount
    return () => {
      newSocket.close();
    };
  }, []);

  // Fallback data fetching if WebSocket fails
  useEffect(() => {
    if (connectionStatus === 'error' || connectionStatus === 'disconnected') {
      const fetchData = async () => {
        try {
          const response = await fetch('/api/dashboard');
          if (response.ok) {
            const data = await response.json();
            setDashboardData(data);
            setError(null);
          } else {
            throw new Error('Failed to fetch dashboard data');
          }
        } catch (err) {
          setError(err.message);
        }
      };

      fetchData();
      const interval = setInterval(fetchData, 10000); // Fallback polling every 10 seconds

      return () => clearInterval(interval);
    }
  }, [connectionStatus]);

  if (error && !dashboardData) {
    return (
      <div className="app">
        <Header connectionStatus="error" />
        <div className="error-container">
          <div className="error-message">
            <h2>Connection Error</h2>
            <p>{error}</p>
            <button 
              onClick={() => window.location.reload()} 
              className="retry-button"
            >
              Retry Connection
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (!dashboardData) {
    return (
      <div className="app">
        <Header connectionStatus={connectionStatus} />
        <div className="loading-container">
          <LoadingSpinner />
          <p>Loading dashboard data...</p>
        </div>
      </div>
    );
  }

  return (
    <ErrorBoundary>
      <div className="app">
        <Header 
          connectionStatus={connectionStatus} 
          lastUpdate={dashboardData.lastUpdate}
        />
        <main className="main-content">
          <Dashboard data={dashboardData} />
        </main>
        {error && (
          <div className="error-toast">
            <span>{error}</span>
            <button onClick={() => setError(null)}>×</button>
          </div>
        )}
      </div>
    </ErrorBoundary>
  );
}

export default App;
