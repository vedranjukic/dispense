import React, { useEffect } from 'react';
import { DashboardProvider } from './contexts/DashboardContext';
import DashboardLayout from './components/layout/DashboardLayout';
import { apiService } from './services/api';
import './styles/globals.css';

// Error Boundary Component
class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { hasError: boolean; error?: Error }
> {
  constructor(props: { children: React.ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Dashboard Error:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="h-screen flex items-center justify-center bg-gray-50">
          <div className="text-center p-8">
            <div className="mb-4">
              <svg
                className="mx-auto h-12 w-12 text-red-500"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"
                />
              </svg>
            </div>
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Something went wrong</h1>
            <p className="text-gray-600 mb-4">
              {this.state.error?.message || 'An unexpected error occurred'}
            </p>
            <button
              onClick={() => window.location.reload()}
              className="btn btn-primary"
            >
              Reload Page
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

// Health Check Component
function HealthCheck() {
  useEffect(() => {
    const checkHealth = async () => {
      try {
        await apiService.healthCheck();
        console.log('API health check passed');
      } catch (error) {
        console.warn('API health check failed:', error);
      }
    };

    checkHealth();

    // Set up periodic health checks
    const interval = setInterval(checkHealth, 30000); // Every 30 seconds

    return () => clearInterval(interval);
  }, []);

  return null;
}

// Main App Component
function App() {
  useEffect(() => {
    // Set page title
    document.title = 'Dispense Dashboard';

    // Configure API base URL if needed
    const baseUrl = window.location.origin;
    console.log('Dashboard initialized with API base URL:', baseUrl);

    // Handle keyboard shortcuts
    const handleKeydown = (e: KeyboardEvent) => {
      // Global keyboard shortcuts can be added here
      if (e.metaKey || e.ctrlKey) {
        switch (e.key) {
          case 'k':
            // Future: Open command palette
            e.preventDefault();
            break;
          case '/':
            // Future: Focus search
            e.preventDefault();
            break;
        }
      }
    };

    document.addEventListener('keydown', handleKeydown);

    return () => {
      document.removeEventListener('keydown', handleKeydown);
    };
  }, []);

  return (
    <ErrorBoundary>
      <DashboardProvider>
        <div className="h-screen overflow-hidden bg-gray-50">
          {/* Main Content */}
          <main className="flex-1 overflow-hidden h-full min-h-0">
            <div className="h-full w-full flex flex-col min-h-0">
              <DashboardLayout />
            </div>
          </main>

          {/* Health Check Component */}
          <HealthCheck />
        </div>
      </DashboardProvider>
    </ErrorBoundary>
  );
}

export default App;