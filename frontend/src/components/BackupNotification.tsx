import { useEffect, useState } from 'react';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export default function BackupNotification() {
    const [message, setMessage] = useState<string | null>(null);
    const [snoozedUntil, setSnoozedUntil] = useState<number>(0);

    useEffect(() => {
        const cleanup = EventsOn('backup:failed', (errMsg: string) => {
            if (Date.now() < snoozedUntil) return;
            setMessage(errMsg);
        });
        return cleanup;
    }, [snoozedUntil]);

    if (!message) return null;

    const handleSnooze = () => {
        setSnoozedUntil(Date.now() + 30 * 60 * 1000);
        setMessage(null);
    };

    const handleDismiss = () => {
        setMessage(null);
    };

    return (
        <div className="fixed top-0 left-0 right-0 z-50 bg-red-900/90 border-b border-red-700 px-4 py-3 flex items-center gap-3">
            <svg className="w-5 h-5 text-red-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
            <span className="text-sm text-red-200 flex-1 min-w-0 truncate">
                Backup failed: {message}
            </span>
            <div className="flex gap-2 flex-shrink-0">
                <button
                    onClick={handleSnooze}
                    className="text-xs text-red-300 hover:text-white transition-colors px-2 py-1 rounded border border-red-700 hover:border-red-500"
                >
                    Snooze 30m
                </button>
                <button
                    onClick={handleDismiss}
                    className="text-xs text-red-300 hover:text-white transition-colors px-2 py-1 rounded border border-red-700 hover:border-red-500"
                >
                    Dismiss
                </button>
            </div>
        </div>
    );
}
