import React from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';
import './ErrorBoundary.css';

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true };
  }

  componentDidCatch(error, errorInfo) {
    this.setState({
      error: error,
      errorInfo: errorInfo
    });
    
    // Log error to console for debugging
    console.error('ErrorBoundary caught an error:', error, errorInfo);
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
  };

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="error-boundary">
          <div className="error-boundary__container">
            <div className="error-boundary__icon">
              <AlertTriangle />
            </div>
            
            <div className="error-boundary__content">
              <h2>Something went wrong</h2>
              <p>
                The dashboard encountered an unexpected error. This might be a temporary issue.
              </p>
              
              {process.env.NODE_ENV === 'development' && this.state.error && (
                <details className="error-boundary__details">
                  <summary>Error Details (Development Mode)</summary>
                  <div className="error-boundary__stack">
                    <h4>Error:</h4>
                    <pre>{this.state.error.toString()}</pre>
                    
                    {this.state.errorInfo && (
                      <>
                        <h4>Component Stack:</h4>
                        <pre>{this.state.errorInfo.componentStack}</pre>
                      </>
                    )}
                  </div>
                </details>
              )}
              
              <div className="error-boundary__actions">
                <button 
                  onClick={this.handleRetry}
                  className="error-boundary__button error-boundary__button--primary"
                >
                  <RefreshCw className="button-icon" />
                  Try Again
                </button>
                
                <button 
                  onClick={this.handleReload}
                  className="error-boundary__button error-boundary__button--secondary"
                >
                  Reload Page
                </button>
              </div>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
