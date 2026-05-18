import { createContext, useContext, useEffect, useState, useCallback, ReactNode } from 'react';
import { GetConfig, SaveConfig } from '../../wailsjs/go/main/App';
import { THEMES, lightThemes } from '../themes';

interface ThemeContextValue {
    activeTheme: string;
    setTheme: (key: string) => void;
}

const ThemeContext = createContext<ThemeContextValue>({
    activeTheme: 'forensic-dark',
    setTheme: () => {},
});

export function useTheme() {
    return useContext(ThemeContext);
}

function applyTheme(key: string) {
    const theme = THEMES.find((t) => t.key === key) ?? THEMES[0];
    const root = document.documentElement;
    Object.entries(theme.vars).forEach(([prop, value]) => {
        root.style.setProperty(prop, value as string);
    });
    root.style.setProperty('--color-scheme', lightThemes.has(key) ? 'light' : 'dark');
}

export function ThemeProvider({ children }: { children: ReactNode }) {
    const [activeTheme, setActiveTheme] = useState('forensic-dark');

    useEffect(() => {
        GetConfig()
            .then((cfg) => {
                const key = cfg.theme || 'forensic-dark';
                setActiveTheme(key);
                applyTheme(key);
            })
            .catch(() => {
                applyTheme('forensic-dark');
            });
    }, []);

    const setTheme = useCallback((key: string) => {
        setActiveTheme(key);
        applyTheme(key);
        GetConfig()
            .then((cfg) => SaveConfig({ ...cfg, theme: key }))
            .catch(() => {});
    }, []);

    return (
        <ThemeContext.Provider value={{ activeTheme, setTheme }}>
            {children}
        </ThemeContext.Provider>
    );
}
