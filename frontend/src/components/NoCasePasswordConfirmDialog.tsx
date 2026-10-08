import { useEffect } from 'react';

interface Props {
    onGoBack: () => void;
    onConfirm: () => void;
}

// Shown both when the "Require a case password" box is unchecked and again on
// Create, so the examiner confirms the irreversible choice twice.
export default function NoCasePasswordConfirmDialog({ onGoBack, onConfirm }: Props) {
    // Escape must only ever cancel; there is no keyboard path to confirm
    // other than tabbing to the confirm button deliberately.
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                e.preventDefault();
                onGoBack();
            }
        };
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [onGoBack]);

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div
                className="absolute inset-0 bg-[var(--bg-primary)] opacity-80"
                onClick={onGoBack}
                aria-hidden="true"
            />
            <div
                role="dialog"
                aria-modal="true"
                aria-labelledby="no-case-password-title"
                className="relative w-full max-w-md mx-4 bg-[var(--bg-secondary)] border border-[var(--border-color)] rounded-xl shadow-2xl p-6 space-y-4"
            >
                <h2 id="no-case-password-title" className="text-lg font-bold text-[var(--text-primary)]">
                    Create this case without a case password?
                </h2>
                <div className="space-y-3 text-sm text-[var(--text-secondary)]">
                    <p>
                        Without a case password, anyone who can log in to dfnotes-go on this computer
                        can open this case without being asked for anything else. If you walk away
                        while you're logged in, this case won't be locked behind its own password.
                    </p>
                    <p>
                        Your notes are still encrypted, signed, and checked for tampering exactly the
                        same way.
                    </p>
                    <p>
                        <strong className="text-[var(--text-primary)]">This can't be changed after the case is created.</strong>{' '}
                        If you need a password on it later, you'll have to create a new case.
                    </p>
                </div>
                <div className="flex gap-2 pt-2">
                    <button
                        type="button"
                        onClick={onGoBack}
                        autoFocus
                        className="flex-1 py-2 px-4 rounded border border-[var(--border-accent-bright)] bg-[var(--bg-accent)] text-[var(--text-primary)] transition-colors"
                    >
                        Go back
                    </button>
                    <button
                        type="button"
                        onClick={onConfirm}
                        className="flex-1 py-2 px-4 rounded border border-[var(--border-color)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
                    >
                        Create without password
                    </button>
                </div>
            </div>
        </div>
    );
}
