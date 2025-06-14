import React from 'react';
import { Database, Table, ExternalLink } from 'lucide-react';
import './TablesPanel.css';

const TablesPanel = ({ data }) => {
  const tables = data.tables || [];
  const count = data.count || 0;

  const handleTableClick = (tableName) => {
    // Open LocalStack DynamoDB admin interface
    const url = `http://localhost:4566/_aws/dynamodb/tables/${tableName}`;
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  if (count === 0) {
    return (
      <div className="tables-panel">
        <div className="panel-header">
          <h3>DynamoDB Tables</h3>
          <p>No tables found</p>
        </div>
        <div className="tables-empty">
          <Database className="empty-icon" />
          <h4>No Tables</h4>
          <p>No DynamoDB tables are currently available.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="tables-panel">
      <div className="panel-header">
        <div className="header-content">
          <h3>DynamoDB Tables</h3>
          <p>{count} tables available in the system</p>
        </div>
        <div className="header-stats">
          <div className="stat-badge">
            <span className="stat-value">{count}</span>
            <span className="stat-label">Tables</span>
          </div>
        </div>
      </div>

      <div className="tables-grid">
        {tables.map((tableName, index) => (
          <div 
            key={tableName || index} 
            className="table-card"
            onClick={() => handleTableClick(tableName)}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                handleTableClick(tableName);
              }
            }}
          >
            <div className="table-card__header">
              <div className="table-icon">
                <Table />
              </div>
              <div className="table-actions">
                <ExternalLink className="external-link-icon" />
              </div>
            </div>
            
            <div className="table-card__content">
              <h4 className="table-name">{tableName}</h4>
              <p className="table-description">
                {getTableDescription(tableName)}
              </p>
            </div>
            
            <div className="table-card__footer">
              <span className="table-type">DynamoDB</span>
              <span className="table-status">Active</span>
            </div>
          </div>
        ))}
      </div>

      <div className="tables-info">
        <div className="info-card">
          <h4>Table Information</h4>
          <div className="info-list">
            <div className="info-item">
              <strong>executions:</strong> Stores execution records with versioning
            </div>
            <div className="info-item">
              <strong>jobs:</strong> Job definitions and configurations
            </div>
            <div className="info-item">
              <strong>schedules:</strong> Scheduling information and rules
            </div>
            <div className="info-item">
              <strong>adapters:</strong> Adapter configurations and settings
            </div>
            <div className="info-item">
              <strong>queue_messages:</strong> Message logs and statistics
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Helper function to get table descriptions
const getTableDescription = (tableName) => {
  const descriptions = {
    'executions': 'Stores execution records with versioning and metadata',
    'jobs': 'Job definitions, configurations, and status information',
    'schedules': 'Scheduling rules, configurations, and timing data',
    'adapters': 'Adapter configurations and connection settings',
    'queue_messages': 'Message logs, statistics, and processing history'
  };
  
  return descriptions[tableName] || 'DynamoDB table for system data storage';
};

export default TablesPanel;
