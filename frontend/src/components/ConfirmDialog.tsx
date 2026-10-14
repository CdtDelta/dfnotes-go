import { useRef, useState, type ReactNode } from 'react';
import Modal, { MODAL_BUTTON_PRIMARY, MODAL_BUTTON_SECONDARY, MODAL_ERROR } from './Modal';

interface ConfirmDialogProps {
    title: string;
    message: ReactNode;
    confirmLabel: string;
    cancelLabel?: string;
    /** May return a Promise; the dialog stays locked until it settles. */
    onConfirm: () => void | Promise<void>;
    onCancel: () => void;
    destructive?: boolean;
    error?: string;
}

const BUTTON_DESTRUCTIVE = 'flex-1 py-2 px-4 rounded border border-[var(--text-error)] bg-[var(--bg-tertiary)] text-[var(--text-error)] text-sm transition-colors hover:bg-[var(--color-hover-bg)] disabled:opacity-50 disabled:cursor-not-allowed';

export default function ConfirmDialog({
    title, message, confirmLabel, cancelLabel = 'Cancel', onConfirm, onCancel, destructive = false, error,
}: ConfirmDialogProps) {
    const [busy, setBusy] = useState(false);
    const cancelRef = useRef<HTMLButtonElement>(null);

    // Escape, backdrop and the close button all route here; cancelling while
    // the action runs would hide its outcome.
    const handleCancel = () => {
        if (!busy) onCancel();
    };

    const handleConfirm = async () => {
        if (busy) return;
        setBusy(true);
        try {
            await onConfirm();
        } finally {
            setBusy(false);
        }
    };

    return (
        <Modal
            title={title}
            onClose={handleCancel}
            closeOnBackdrop={!busy}
            initialFocusRef={cancelRef}
            footer={
                <>
                    <button
                        type="button"
                        onClick={handleConfirm}
                        disabled={busy}
                        className={destructive ? BUTTON_DESTRUCTIVE : MODAL_BUTTON_PRIMARY}
                    >
                        {busy ? 'Working...' : confirmLabel}
                    </button>
                    <button ref={cancelRef} type="button" onClick={handleCancel} disabled={busy} className={MODAL_BUTTON_SECONDARY}>
                        {cancelLabel}
                    </button>
                </>
            }
        >
            <div className="text-sm text-[var(--text-secondary)] space-y-2">{message}</div>
            {error && <p role="alert" className={MODAL_ERROR}>{error}</p>}
        </Modal>
    );
}
