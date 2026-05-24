import { ResumeDocReminder } from '../../wailsjs/go/main/App';

interface Props {
    onResume: () => void;
}

export default function DocReminderPausedBanner({ onResume }: Props) {
    const handleResume = async () => {
        await ResumeDocReminder();
        onResume();
    };

    return (
        <div className="flex items-center justify-between px-4 py-1.5 bg-[var(--bg-secondary)] border-b border-[var(--border-color)] text-sm text-[var(--text-muted)]">
            <span>Documentation reminders paused</span>
            <button
                onClick={handleResume}
                className="text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)] underline transition-colors"
            >
                Resume
            </button>
        </div>
    );
}
