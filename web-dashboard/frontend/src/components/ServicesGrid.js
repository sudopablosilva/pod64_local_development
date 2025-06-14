import React from 'react';
import { CheckCircle, XCircle, Clock, ExternalLink } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import './ServicesGrid.css';

const ServicesGrid = ({ services }) => {
  const getStatusIcon = (status) => {
    switch (status) {
      case 'online':
        return <CheckCircle className="status-icon status-icon--online" />;
      case 'offline':
        return <XCircle className="status-icon status-icon--offline" />;
      default:
        return <Clock className="status-icon status-icon--pending" />;
    }
  };

  const getStatusText = (status) => {
    switch (status) {
      case 'online':
        return 'Online';
      case 'offline':
        return 'Offline';
      default:
        return 'Checking...';
    }
  };

  const getLastCheckText = (lastCheck) => {
    if (!lastCheck) return 'Never checked';
    try {
      return formatDistanceToNow(new Date(lastCheck), { addSuffix: true });
    } catch {
      return 'Recently checked';
    }
  };

  const handleServiceClick = (service) => {
    const url = `http://localhost:${service.port}/health`;
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  if (!services || services.length === 0) {
    return (
      <div className="services-grid">
        <div className="services-empty">
          <Clock className="empty-icon" />
          <h3>No Services Data</h3>
          <p>Waiting for service information...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="services-grid">
      <div className="services-header">
        <h3>Microservices Status</h3>
        <p>Real-time health monitoring of all system components</p>
      </div>
      
      <div className="services-list">
        {services.map((service) => (
          <div 
            key={`${service.name}-${service.port}`} 
            className={`service-card ${service.status}`}
            onClick={() => handleServiceClick(service)}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                handleServiceClick(service);
              }
            }}
          >
            <div className="service-card__header">
              <div className="service-info">
                <h4 className="service-name">{service.name}</h4>
                <span className="service-port">Port {service.port}</span>
              </div>
              <div className="service-actions">
                <ExternalLink className="external-link-icon" />
              </div>
            </div>
            
            <div className="service-card__content">
              <div className="service-status">
                {getStatusIcon(service.status)}
                <span className="status-text">{getStatusText(service.status)}</span>
              </div>
              
              <div className="service-details">
                <div className="service-detail">
                  <span className="detail-label">Last Check:</span>
                  <span className="detail-value">
                    {getLastCheckText(service.lastCheck)}
                  </span>
                </div>
                
                {service.responseTime && service.responseTime !== 'N/A' && (
                  <div className="service-detail">
                    <span className="detail-label">Response Time:</span>
                    <span className="detail-value">{service.responseTime}</span>
                  </div>
                )}
                
                {service.error && (
                  <div className="service-error">
                    <span className="error-label">Error:</span>
                    <span className="error-message" title={service.error}>
                      {service.error.length > 50 
                        ? `${service.error.substring(0, 50)}...` 
                        : service.error
                      }
                    </span>
                  </div>
                )}
              </div>
            </div>
            
            <div className={`service-card__indicator ${service.status}`} />
          </div>
        ))}
      </div>
      
      <div className="services-summary">
        <div className="summary-stat">
          <span className="summary-value">
            {services.filter(s => s.status === 'online').length}
          </span>
          <span className="summary-label">Online</span>
        </div>
        <div className="summary-stat">
          <span className="summary-value">
            {services.filter(s => s.status === 'offline').length}
          </span>
          <span className="summary-label">Offline</span>
        </div>
        <div className="summary-stat">
          <span className="summary-value">{services.length}</span>
          <span className="summary-label">Total</span>
        </div>
      </div>
    </div>
  );
};

export default ServicesGrid;
