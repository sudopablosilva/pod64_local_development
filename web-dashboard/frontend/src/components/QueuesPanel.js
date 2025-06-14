import React from 'react';
import { MessageSquare, Activity, Clock, ExternalLink } from 'lucide-react';
import './QueuesPanel.css';

const QueuesPanel = ({ data }) => {
  const queues = data.queues || [];
  const count = data.count || 0;

  const handleQueueClick = (queueName) => {
    // Open LocalStack SQS admin interface
    const url = `http://localhost:4566/_aws/sqs/queues/${queueName}`;
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  const getTotalMessages = (queue) => {
    const visible = parseInt(queue.visibleMessages || 0);
    const notVisible = parseInt(queue.notVisibleMessages || 0);
    return visible + notVisible;
  };

  const getQueueStatus = (queue) => {
    const total = getTotalMessages(queue);
    if (total === 0) return 'idle';
    if (total > 10) return 'busy';
    return 'active';
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'busy':
        return <Activity className="status-icon status-icon--busy" />;
      case 'active':
        return <MessageSquare className="status-icon status-icon--active" />;
      default:
        return <Clock className="status-icon status-icon--idle" />;
    }
  };

  const getStatusClass = (status) => {
    switch (status) {
      case 'busy':
        return 'queue-card--busy';
      case 'active':
        return 'queue-card--active';
      default:
        return 'queue-card--idle';
    }
  };

  if (count === 0) {
    return (
      <div className="queues-panel">
        <div className="panel-header">
          <h3>SQS Queues</h3>
          <p>No queues found</p>
        </div>
        <div className="queues-empty">
          <MessageSquare className="empty-icon" />
          <h4>No Queues</h4>
          <p>No SQS queues are currently available.</p>
        </div>
      </div>
    );
  }

  const totalMessages = queues.reduce((sum, queue) => sum + getTotalMessages(queue), 0);
  const activeQueues = queues.filter(queue => getTotalMessages(queue) > 0).length;

  return (
    <div className="queues-panel">
      <div className="panel-header">
        <div className="header-content">
          <h3>SQS Queues</h3>
          <p>{count} queues managing message flow</p>
        </div>
        <div className="header-stats">
          <div className="stat-badge">
            <span className="stat-value">{totalMessages}</span>
            <span className="stat-label">Messages</span>
          </div>
          <div className="stat-badge">
            <span className="stat-value">{activeQueues}</span>
            <span className="stat-label">Active</span>
          </div>
        </div>
      </div>

      <div className="queues-grid">
        {queues.map((queue, index) => {
          const status = getQueueStatus(queue);
          const totalMsgs = getTotalMessages(queue);
          
          return (
            <div 
              key={queue.name || index} 
              className={`queue-card ${getStatusClass(status)}`}
              onClick={() => handleQueueClick(queue.name)}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  handleQueueClick(queue.name);
                }
              }}
            >
              <div className="queue-card__header">
                <div className="queue-info">
                  <h4 className="queue-name">{queue.name}</h4>
                  <span className="queue-type">SQS Queue</span>
                </div>
                <div className="queue-actions">
                  <ExternalLink className="external-link-icon" />
                </div>
              </div>
              
              <div className="queue-card__content">
                <div className="queue-status">
                  {getStatusIcon(status)}
                  <span className="status-text">{status.charAt(0).toUpperCase() + status.slice(1)}</span>
                </div>
                
                <div className="queue-metrics">
                  <div className="metric">
                    <span className="metric-value">{queue.visibleMessages || 0}</span>
                    <span className="metric-label">Visible</span>
                  </div>
                  <div className="metric">
                    <span className="metric-value">{queue.notVisibleMessages || 0}</span>
                    <span className="metric-label">Processing</span>
                  </div>
                  <div className="metric metric--total">
                    <span className="metric-value">{totalMsgs}</span>
                    <span className="metric-label">Total</span>
                  </div>
                </div>
              </div>
              
              <div className="queue-card__footer">
                <div className="queue-description">
                  {getQueueDescription(queue.name)}
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <div className="queues-info">
        <div className="info-card">
          <h4>Queue Information</h4>
          <div className="info-grid">
            <div className="info-section">
              <h5>Message Flow</h5>
              <div className="info-list">
                <div className="info-item">
                  <strong>job-requests:</strong> Initial job submission requests
                </div>
                <div className="info-item">
                  <strong>jmw-queue:</strong> Jobs processed by JMW service
                </div>
                <div className="info-item">
                  <strong>jmr-queue:</strong> Jobs handled by JMR service
                </div>
              </div>
            </div>
            
            <div className="info-section">
              <h5>Scheduling & Adaptation</h5>
              <div className="info-list">
                <div className="info-item">
                  <strong>sp-queue:</strong> Scheduler plugin processing
                </div>
                <div className="info-item">
                  <strong>spa-queue:</strong> Scheduler plugin adapter
                </div>
                <div className="info-item">
                  <strong>spaq-queue:</strong> Final queue processing
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Helper function to get queue descriptions
const getQueueDescription = (queueName) => {
  const descriptions = {
    'job-requests': 'Initial job submission and processing requests',
    'jmw-queue': 'Job Manager Worker processing queue',
    'jmr-queue': 'Job Manager Runner execution queue',
    'sp-queue': 'Scheduler Plugin processing queue',
    'spa-queue': 'Scheduler Plugin Adapter queue',
    'spaq-queue': 'Scheduler Plugin Adapter Queue final processing'
  };
  
  return descriptions[queueName] || 'Message queue for system communication';
};

export default QueuesPanel;
