import React, { useState } from 'react';
import ServicesGrid from './ServicesGrid';
import ExecutionsPanel from './ExecutionsPanel';
import TablesPanel from './TablesPanel';
import QueuesPanel from './QueuesPanel';
import StatsOverview from './StatsOverview';
import { Activity, Database, MessageSquare, Play } from 'lucide-react';
import './Dashboard.css';

const Dashboard = ({ data }) => {
  const [activeTab, setActiveTab] = useState('overview');

  const tabs = [
    { id: 'overview', label: 'Overview', icon: Activity },
    { id: 'executions', label: 'Executions', icon: Play, count: data.executions?.count || 0 },
    { id: 'tables', label: 'Tables', icon: Database, count: data.tables?.count || 0 },
    { id: 'queues', label: 'Queues', icon: MessageSquare, count: data.queues?.count || 0 }
  ];

  const renderTabContent = () => {
    switch (activeTab) {
      case 'overview':
        return (
          <div className="overview-content">
            <StatsOverview data={data} />
            <ServicesGrid services={data.services || []} />
          </div>
        );
      case 'executions':
        return <ExecutionsPanel data={data.executions || { count: 0, executions: [] }} />;
      case 'tables':
        return <TablesPanel data={data.tables || { count: 0, tables: [] }} />;
      case 'queues':
        return <QueuesPanel data={data.queues || { count: 0, queues: [] }} />;
      default:
        return null;
    }
  };

  return (
    <div className="dashboard">
      <div className="container">
        <div className="dashboard-header">
          <h2>System Monitoring Dashboard</h2>
          <p className="dashboard-subtitle">
            Real-time monitoring of POC BDD microservices architecture
          </p>
        </div>

        <div className="dashboard-tabs">
          {tabs.map((tab) => {
            const IconComponent = tab.icon;
            return (
              <button
                key={tab.id}
                className={`tab-button ${activeTab === tab.id ? 'active' : ''}`}
                onClick={() => setActiveTab(tab.id)}
                aria-selected={activeTab === tab.id}
                role="tab"
              >
                <IconComponent className="tab-icon" />
                <span className="tab-label">{tab.label}</span>
                {tab.count !== undefined && (
                  <span className="tab-count">{tab.count}</span>
                )}
              </button>
            );
          })}
        </div>

        <div className="dashboard-content" role="tabpanel">
          {renderTabContent()}
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
