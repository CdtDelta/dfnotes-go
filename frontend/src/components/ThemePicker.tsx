import { useEffect } from 'react';
import { THEMES } from '../themes';
import { useTheme } from '../context/ThemeContext';

interface ThemePickerProps {
    onClose: () => void;
}

export default function ThemePicker({ onClose }: ThemePickerProps) {
    const { activeTheme, setTheme } = useTheme();

    useEffect(() => {
        const handler = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose();
        };
        window.addEventListener('keydown', handler);
        return () => window.removeEventListener('keydown', handler);
    }, [onClose]);

    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center"
            style={{ backgroundColor: 'rgba(0,0,0,0.6)' }}
            onClick={onClose}
        >
            <div
                className="bg-gray-900 border border-gray-700 rounded-lg shadow-xl w-full max-w-md p-6"
                onClick={(e) => e.stopPropagation()}
            >
                <div className="flex items-center justify-between mb-4">
                    <h2 className="text-lg font-semibold text-gray-100">Select Theme</h2>
                    <button
                        onClick={onClose}
                        className="text-gray-400 hover:text-gray-200 transition-colors"
                        aria-label="Close"
                    >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                <div className="space-y-1 max-h-96 overflow-y-auto">
                    {THEMES.map((theme) => (
                        <button
                            key={theme.key}
                            onClick={() => {
                                setTheme(theme.key);
                                onClose();
                            }}
                            className="w-full flex items-center gap-3 px-3 py-2 rounded hover:bg-gray-800 transition-colors text-left"
                        >
                            <span
                                className="w-6 h-6 rounded border border-gray-600 flex-shrink-0"
                                style={{ backgroundColor: theme.swatch }}
                            />
                            <span className="text-sm text-gray-200 flex-1">{theme.name}</span>
                            {activeTheme === theme.key && (
                                <svg className="w-4 h-4 text-blue-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" strokeWidth={2.5} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                                </svg>
                            )}
                        </button>
                    ))}
                </div>
            </div>
        </div>
    );
}
