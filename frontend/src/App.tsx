import { HashRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import LoadingSpinner from './components/LoadingSpinner';
import SetupWizard from './pages/SetupWizard';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import CaseCreatePage from './pages/CaseCreatePage';
import CaseDetailPage from './pages/CaseDetailPage';

function AppRouter() {
    const { appState } = useAuth();

    if (appState === 'loading') {
        return <LoadingSpinner />;
    }

    if (appState === 'setup') {
        return <SetupWizard />;
    }

    if (appState === 'login') {
        return <LoginPage />;
    }

    return (
        <HashRouter>
            <Routes>
                <Route path="/" element={<DashboardPage />} />
                <Route path="/cases/new" element={<CaseCreatePage />} />
                <Route path="/cases/:caseId" element={<CaseDetailPage />} />
            </Routes>
        </HashRouter>
    );
}

export default function App() {
    return (
        <AuthProvider>
            <AppRouter />
        </AuthProvider>
    );
}
