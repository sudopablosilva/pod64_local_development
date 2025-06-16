import React, { useState } from 'react';
import { Play, Clock, CheckCircle, AlertCircle, Search, Filter } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import './ExecutionsPanel.css';

const ExecutionsPanel = ({ data }) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [stageFilter, setStageFilter] = useState('all');

  const executions = data.executions || [];
  const count = data.count || 0;

  // Get unique stages and statuses for filters
  const uniqueStages = [...new Set(executions.map(exec => exec.stage).filter(Boolean))];
  const uniqueStatuses = [...new Set(executions.map(exec => exec.status).filter(Boolean))];

  // Filter executions
  const filteredExecutions = executions.filter(execution => {
    const matchesSearch = !searchTerm || 
      execution.executionName?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      execution.originalName?.toLowerCase().includes(searchTerm.toLowerCase());
    
    const matchesStatus = statusFilter === 'all' || execution.status === statusFilter;
    const matchesStage = stageFilter === 'all' || execution.stage === stageFilter;
    
    return matchesSearch && matchesStatus && matchesStage;
  });

  const getStatusIcon = (status) => {
    switch (status?.toLowerCase()) {
      case 'completed':
      case 'success':
        return <CheckCircle className="status-icon status-icon--success" />;
      case 'failed':
      case 'error':
        return <AlertCircle className="status-icon status-icon--error" />;
      case 'running':
      case 'processing':
        return <Play className="status-icon status-icon--running" />;
      default:
        return <Clock className="status-icon status-icon--pending" />;
    }
  };

  const getStatusClass = (status) => {
    switch (status?.toLowerCase()) {
      case 'completed':
      case 'success':
        return 'execution-card--success';
      case 'failed':
      case 'error':
        return 'execution-card--error';
      case 'running':
      case 'processing':
        return 'execution-card--running';
      default:
        return 'execution-card--pending';
    }
  };

  const formatTimestamp = (timestamp) => {
    if (!timestamp) return 'Unknown';
    try {
      const date = new Date(timestamp * 1000); // Convert from Unix timestamp
      return formatDistanceToNow(date, { addSuffix: true });
    } catch {
      return 'Invalid date';
    }
  };

  if (count === 0) {
    return (
      <div className="executions-panel">
        <div className="panel-header">
          <h3>Executions</h3>
          <p>No executions found</p>
        </div>
        <div className="executions-empty">
          <Play className="empty-icon" />
          <h4>No Executions</h4>
          <p>No execution data available at the moment.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="executions-panel">
      <div className="panel-header">
        <div className="header-content">
          <h3>Executions</h3>
          <p>{count} total executions tracked</p>
        </div>
        <div className="header-stats">
          <div className="stat-badge">
            <span className="stat-value">{filteredExecutions.length}</span>
            <span className="stat-label">Filtered</span>
          </div>
        </div>
      </div>

      <div className="panel-controls">
        <div className="search-box">
          <Search className="search-icon" />
          <input
            type="text"
            placeholder="Search executions..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="search-input"
          />
        </div>
        
        <div className="filters">
          <div className="filter-group">
            <Filter className="filter-icon" />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Statuses</option>
              {uniqueStatuses.map(status => (
                <option key={status} value={status}>{status}</option>
              ))}
            </select>
          </div>
          
          <div className="filter-group">
            <select
              value={stageFilter}
              onChange={(e) => setStageFilter(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Stages</option>
              {uniqueStages.map(stage => (
                <option key={stage} value={stage}>{stage}</option>
              ))}
            </select>
          </div>
        </div>
      </div>

      <div className="executions-list">
        {filteredExecutions.length === 0 ? (
          <div className="no-results">
            <Search className="no-results-icon" />
            <h4>No Results Found</h4>
            <p>Try adjusting your search or filter criteria.</p>
          </div>
        ) : (
          filteredExecutions.map((execution, index) => (
            <div 
              key={execution.executionName || index} 
              className={`execution-card ${getStatusClass(execution.status)}`}
            >
              <div className="execution-card__header">
                <div className="execution-info">
                  <h4 className="execution-name">
                    {execution.originalName || execution.executionName || 'Unknown'}
                  </h4>
                  <span className="execution-id">
                    {execution.executionName || 'No ID'}
                  </span>
                </div>
                <div className="execution-status">
                  {getStatusIcon(execution.status)}
                  <span className="status-text">{execution.status || 'Unknown'}</span>
                </div>
              </div>
              
              <div className="execution-card__content">
                <div className="execution-details">
                  <div className="detail-row">
                    <span className="detail-label">Stage:</span>
                    <span className="detail-value">{execution.stage || 'N/A'}</span>
                  </div>
                  
                  <div className="detail-row">
                    <span className="detail-label">Processed By:</span>
                    <span className="detail-value">{execution.processedBy || 'N/A'}</span>
                  </div>
                  
                  <div className="detail-row">
                    <span className="detail-label">Version:</span>
                    <span className="detail-value">v{execution.version || 1}</span>
                  </div>
                  
                  <div className="detail-row">
                    <span className="detail-label">Last Updated:</span>
                    <span className="detail-value">
                      {formatTimestamp(execution.timestamp)}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default ExecutionsPanel;
