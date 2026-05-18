import { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { CheckFirstLaunch, GetUserInfo } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';

type AppState = 'loading' | 'setup' | 'login' | 'authenticated' | 'db-missing';

interface AuthContextType {
    appState: AppState;
    user: services.LoginResponse | null;
    loginScreenInfo: services.LoginScreenInfo | null;
    missingPath: string;
    setAuthenticated: (user: services.LoginResponse) => void;
    setSetupComplete: (user: services.LoginResponse) => void;
    resetToLogin: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
    const [appState, setAppState] = useState<AppState>('loading');
    const [user, setUser] = useState<services.LoginResponse | null>(null);
    const [loginScreenInfo, setLoginScreenInfo] = useState<services.LoginScreenInfo | null>(null);
    const [missingPath, setMissingPath] = useState('');

    const fetchLoginInfo = async () => {
        try {
            setLoginScreenInfo(await GetUserInfo());
        } catch {
            // Login screen renders without user info if the DB is unavailable
        }
    };

    useEffect(() => {
        CheckFirstLaunch()
            .then(async (isFirstLaunch) => {
                if (isFirstLaunch) {
                    setAppState('setup');
                } else {
                    await fetchLoginInfo();
                    setAppState('login');
                }
            })
            .catch((err: unknown) => {
                const msg = String(err);
                if (msg.includes('not found')) {
                    const afterAt = msg.split(' at ')[1] ?? '';
                    setMissingPath(afterAt.split(' --')[0].trim());
                    setAppState('db-missing');
                } else {
                    setAppState('setup');
                }
            });
    }, []);

    const setAuthenticated = (loginUser: services.LoginResponse) => {
        setUser(loginUser);
        setAppState('authenticated');
    };

    const setSetupComplete = (loginUser: services.LoginResponse) => {
        setUser(loginUser);
        setAppState('authenticated');
    };

    const resetToLogin = async () => {
        setUser(null);
        setAppState('loading');
        await fetchLoginInfo();
        setAppState('login');
    };

    return (
        <AuthContext.Provider value={{ appState, user, loginScreenInfo, missingPath, setAuthenticated, setSetupComplete, resetToLogin }}>
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth() {
    const ctx = useContext(AuthContext);
    if (!ctx) throw new Error('useAuth must be inside AuthProvider');
    return ctx;
}
