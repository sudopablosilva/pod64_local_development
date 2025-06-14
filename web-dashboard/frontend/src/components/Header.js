import React from 'react';
import { Activity, Wifi, WifiOff, AlertCircle } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import './Header.css';

const Header = ({ connectionStatus, lastUpdate }) => {
  const getConnectionIcon = () => {
    switch (connectionStatus) {
      case 'connected':
        return <Wifi className="connection-icon connected" />;
      case 'disconnected':
        return <WifiOff className="connection-icon disconnected" />;
      case 'error':
        return <AlertCircle className="connection-icon error" />;
      default:
        return <div className="connection-spinner" />;
    }
  };

  const getConnectionText = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'Connected';
      case 'disconnected':
        return 'Disconnected';
      case 'error':
        return 'Connection Error';
      default:
        return 'Connecting...';
    }
  };

  const getLastUpdateText = () => {
    if (!lastUpdate) return '';
    try {
      return `Updated ${formatDistanceToNow(new Date(lastUpdate), { addSuffix: true })}`;
    } catch {
      return 'Recently updated';
    }
  };

  return (
    <header className="header">
      <div className="container">
        <div className="header-content">
          <div className="header-left">
            <div className="logo">
              <Activity className="logo-icon" />
              <div className="logo-text">
                <h1>POC BDD Dashboard</h1>
                <span className="subtitle">Microservices Monitoring</span>
              </div>
            </div>
          </div>
          
          <div className="header-right">
            <div className="status-info">
              {lastUpdate && (
                <span className="last-update">{getLastUpdateText()}</span>
              )}
              <div className={`connection-status ${connectionStatus}`}>
                {getConnectionIcon()}
                <span className="connection-text">{getConnectionText()}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
};

export default Header;
