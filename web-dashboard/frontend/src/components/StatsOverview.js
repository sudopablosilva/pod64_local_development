import React from 'react';
import { Activity, Database, MessageSquare, Play, CheckCircle, XCircle } from 'lucide-react';
import './StatsOverview.css';

const StatsOverview = ({ data }) => {
  const onlineServices = data.services?.filter(service => service.status === 'online').length || 0;
  const totalServices = data.services?.length || 0;
  const offlineServices = totalServices - onlineServices;

  const stats = [
    {
      id: 'services',
      title: 'Services Online',
      value: `${onlineServices}/${totalServices}`,
      icon: Activity,
      color: onlineServices === totalServices ? 'green' : offlineServices > 0 ? 'red' : 'yellow',
      subtitle: offlineServices > 0 ? `${offlineServices} offline` : 'All systems operational'
    },
    {
      id: 'executions',
      title: 'Active Executions',
      value: data.executions?.count || 0,
      icon: Play,
      color: 'blue',
      subtitle: 'Total executions tracked'
    },
    {
      id: 'tables',
      title: 'DynamoDB Tables',
      value: data.tables?.count || 0,
      icon: Database,
      color: 'purple',
      subtitle: 'Tables available'
    },
    {
      id: 'queues',
      title: 'SQS Queues',
      value: data.queues?.count || 0,
      icon: MessageSquare,
      color: 'orange',
      subtitle: 'Message queues active'
    }
  ];

  const getColorClasses = (color) => {
    const colors = {
      green: 'stat-card--green',
      red: 'stat-card--red',
      yellow: 'stat-card--yellow',
      blue: 'stat-card--blue',
      purple: 'stat-card--purple',
      orange: 'stat-card--orange'
    };
    return colors[color] || 'stat-card--blue';
  };

  return (
    <div className="stats-overview">
      <div className="stats-grid">
        {stats.map((stat) => {
          const IconComponent = stat.icon;
          return (
            <div key={stat.id} className={`stat-card ${getColorClasses(stat.color)}`}>
              <div className="stat-card__header">
                <div className="stat-card__icon">
                  <IconComponent />
                </div>
                <div className="stat-card__title">{stat.title}</div>
              </div>
              <div className="stat-card__content">
                <div className="stat-card__value">{stat.value}</div>
                <div className="stat-card__subtitle">{stat.subtitle}</div>
              </div>
            </div>
          );
        })}
      </div>

      {/* System Health Summary */}
      <div className="health-summary">
        <div className="health-summary__header">
          <h3>System Health</h3>
          <div className={`health-indicator ${onlineServices === totalServices ? 'healthy' : 'warning'}`}>
            {onlineServices === totalServices ? (
              <>
                <CheckCircle className="health-icon" />
                <span>All Systems Operational</span>
              </>
            ) : (
              <>
                <XCircle className="health-icon" />
                <span>Issues Detected</span>
              </>
            )}
          </div>
        </div>
        
        <div className="health-details">
          <div className="health-metric">
            <span className="health-metric__label">Service Availability</span>
            <div className="health-metric__bar">
              <div 
                className="health-metric__fill"
                style={{ width: `${(onlineServices / totalServices) * 100}%` }}
              />
            </div>
            <span className="health-metric__value">
              {Math.round((onlineServices / totalServices) * 100)}%
            </span>
          </div>
          
          <div className="health-stats">
            <div className="health-stat">
              <span className="health-stat__value">{data.executions?.count || 0}</span>
              <span className="health-stat__label">Executions</span>
            </div>
            <div className="health-stat">
              <span className="health-stat__value">{data.tables?.count || 0}</span>
              <span className="health-stat__label">Tables</span>
            </div>
            <div className="health-stat">
              <span className="health-stat__value">{data.queues?.count || 0}</span>
              <span className="health-stat__label">Queues</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default StatsOverview;
