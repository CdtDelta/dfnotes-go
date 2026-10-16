import React from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import './index.css';
import { installCSPReporter } from './utils/cspReporter';
import { installNavigationGuard } from './utils/navigationGuard';

installCSPReporter();
installNavigationGuard();

const container = document.getElementById('root');
const root = createRoot(container!);
root.render(
    <React.StrictMode>
        <App />
    </React.StrictMode>
);
