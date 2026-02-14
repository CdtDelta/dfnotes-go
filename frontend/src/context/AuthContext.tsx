import { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { CheckFirstLaunch, GetUserInfo } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';

type AppState = 'loading' | 'setup' | 'login' | 'authenticated';

interface AuthContextType {
    appState: AppState;
    user: services.LoginResponse | null;
    loginScreenInfo: services.LoginScreenInfo | null;
    setAuthenticated: (user: services.LoginResponse) => void;
    setSetupComplete: (user: services.LoginResponse) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
    const [appState, setAppState] = useState<AppState>('loading');
    const [user, setUser] = useState<services.LoginResponse | null>(null);
    const [loginScreenInfo, setLoginScreenInfo] = useState<services.LoginScreenInfo | null>(null);

    useEffect(() => {
        CheckFirstLaunch()
            .then(async (isFirstLaunch) => {
                if (isFirstLaunch) {
                    setAppState('setup');
                } else {
                    try {
                        const info = await GetUserInfo();
                        setLoginScreenInfo(info);
                    } catch {
                        // If we can't get user info, still show login
                    }
                    setAppState('login');
                }
            })
            .catch(() => {
                setAppState('setup');
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

    return (
        <AuthContext.Provider value={{ appState, user, loginScreenInfo, setAuthenticated, setSetupComplete }}>
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth() {
    const ctx = useContext(AuthContext);
    if (!ctx) throw new Error('useAuth must be inside AuthProvider');
    return ctx;
}
