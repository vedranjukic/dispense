import React from 'react';
import { useDashboard } from '../../contexts/DashboardContext';
import Sidebar from './Sidebar';
import MainContent from './MainContent';
import RightPanel from './RightPanel';

export default function DashboardLayout() {
  const { state } = useDashboard();

  return (
    <div className="flex h-full bg-gray-50">
      {/* Left Sidebar */}
      <div
        className={`${
          state.sidebarCollapsed ? 'w-16' : 'w-80'
        } transition-all duration-200 bg-white border-r border-gray-200 flex-shrink-0`}
      >
        <Sidebar />
      </div>

      {/* Main Content Area */}
      <div className="flex-1 flex min-w-0 w-0">
        <MainContent />
      </div>

      {/* Right Panel */}
      <div className="w-64 bg-white border-l border-gray-200 flex-shrink-0">
        <RightPanel />
      </div>
    </div>
  );
}