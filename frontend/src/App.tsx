import { useEffect, useState } from 'react';
import { HashRouter, Routes, Route, useNavigate } from 'react-router-dom';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { AuthProvider, useAuth } from './context/AuthContext';
import { ThemeProvider } from './context/ThemeContext';
import LoadingSpinner from './components/LoadingSpinner';
import ThemePicker from './components/ThemePicker';
import BackupNotification from './components/BackupNotification';
import SetupWizard from './pages/SetupWizard';
import DBMissingPage from './pages/DBMissingPage';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import CaseCreatePage from './pages/CaseCreatePage';
import CaseDetailPage from './pages/CaseDetailPage';
import SettingsPage from './pages/SettingsPage';

function MenuHandler() {
    const navigate = useNavigate();

    useEffect(() => {
        const cleanupSettings = EventsOn('menu:settings', () => {
            navigate('/settings');
        });
        return cleanupSettings;
    }, [navigate]);

    return null;
}

function AppRouter() {
    const { appState } = useAuth();
    const [showThemePicker, setShowThemePicker] = useState(false);

    useEffect(() => {
        const cleanup = EventsOn('menu:theme', () => setShowThemePicker(true));
        return cleanup;
    }, []);

    if (appState === 'loading') {
        return <LoadingSpinner />;
    }

    if (appState === 'setup') {
        return <SetupWizard />;
    }

    if (appState === 'db-missing') {
        return <DBMissingPage />;
    }

    if (appState === 'login') {
        return <LoginPage />;
    }

    return (
        <HashRouter>
            <MenuHandler />
            <BackupNotification />
            {showThemePicker && <ThemePicker onClose={() => setShowThemePicker(false)} />}
            <Routes>
                <Route path="/" element={<DashboardPage />} />
                <Route path="/cases/new" element={<CaseCreatePage />} />
                <Route path="/cases/:caseId" element={<CaseDetailPage />} />
                <Route path="/settings" element={<SettingsPage />} />
            </Routes>
        </HashRouter>
    );
}

export default function App() {
    return (
        <ThemeProvider>
            <AuthProvider>
                <AppRouter />
            </AuthProvider>
        </ThemeProvider>
    );
}
