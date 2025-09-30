import React from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';

// Enable React strict mode in development
const StrictMode = process.env.NODE_ENV === 'development' ? React.StrictMode : React.Fragment;

const container = document.getElementById('root');
if (!container) {
  throw new Error('Root container element not found');
}

const root = createRoot(container);

root.render(
  <StrictMode>
    <App />
  </StrictMode>
);