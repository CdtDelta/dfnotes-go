import { useEffect } from 'react';
import { WindowShow, WindowSetAlwaysOnTop } from '../../wailsjs/runtime/runtime';
import { SnoozeDockReminder, PauseDocReminder } from '../../wailsjs/go/main/App';

interface Props {
    minutesElapsed: number;
    onClose: () => void;
    onPause: () => void;
}

export default function DocReminderModal({ minutesElapsed, onClose, onPause }: Props) {
    useEffect(() => {
        WindowShow();
        WindowSetAlwaysOnTop(true);
    }, []);

    const dismiss = async (action: () => Promise<void> | void) => {
        await action();
        await WindowSetAlwaysOnTop(false);
        onClose();
    };

    const handleDocumentNow = async () => {
        await dismiss(() => {});
        setTimeout(() => {
            document.querySelector<HTMLTextAreaElement>('textarea')?.focus();
        }, 80);
    };

    const handleSnooze15 = () => dismiss(() => SnoozeDockReminder(15));

    const handleSnooze30 = () => dismiss(() => SnoozeDockReminder(30));

    const handlePause = () =>
        dismiss(async () => {
            await PauseDocReminder();
            onPause();
        });

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70">
            <div className="w-full max-w-md mx-4 bg-[var(--bg-secondary)] border border-[var(--border-color)] rounded-xl shadow-2xl p-8 space-y-6">
                <div className="space-y-2">
                    <h2 className="text-xl font-bold text-[var(--text-primary)]">
                        Documentation Reminder
                    </h2>
                    <p className="text-[var(--text-secondary)]">
                        It has been{' '}
                        <span className="font-semibold text-[var(--text-primary)]">
                            {minutesElapsed} minute{minutesElapsed !== 1 ? 's' : ''}
                        </span>{' '}
                        since your last documentation entry.
                    </p>
                    <p className="text-sm text-[var(--text-muted)]">
                        Take a moment to commit your notes before continuing.
                    </p>
                </div>

                <div className="space-y-3">
                    <button
                        onClick={handleDocumentNow}
                        className="w-full py-2.5 px-4 rounded-lg font-medium text-sm transition-colors bg-[var(--color-accent)] hover:opacity-90 text-white"
                    >
                        Document Now
                    </button>

                    <div className="grid grid-cols-2 gap-3">
                        <button
                            onClick={handleSnooze15}
                            className="py-2 px-4 rounded-lg text-sm border border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--border-accent)] transition-colors"
                        >
                            Snooze 15 min
                        </button>
                        <button
                            onClick={handleSnooze30}
                            className="py-2 px-4 rounded-lg text-sm border border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--border-accent)] transition-colors"
                        >
                            Snooze 30 min
                        </button>
                    </div>

                    <button
                        onClick={handlePause}
                        className="w-full py-2 px-4 rounded-lg text-sm text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
                    >
                        Pause reminders
                    </button>
                </div>
            </div>
        </div>
    );
}
