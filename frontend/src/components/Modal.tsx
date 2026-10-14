import { useEffect, useId, useRef, type ReactNode, type RefObject } from 'react';

interface ModalProps {
    title: string;
    onClose: () => void;
    children: ReactNode;
    footer?: ReactNode;
    /** Callers pass false while a form has unsaved input or a request is in flight. */
    closeOnBackdrop?: boolean;
    initialFocusRef?: RefObject<HTMLElement | null>;
}

// Form and button classes for dialogs built on Modal, kept here so they stay
// consistent and theme-driven.
export const MODAL_LABEL = 'block text-xs text-[var(--text-secondary)] mb-1';
export const MODAL_INPUT = 'w-full px-3 py-1.5 bg-[var(--bg-primary)] border border-[var(--border-color)] rounded text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:outline-none focus:border-[var(--border-accent-bright)]';
export const MODAL_SELECT = 'w-full px-3 py-1.5 bg-[var(--select-bg)] border border-[var(--border-color)] rounded text-sm text-[var(--select-text)] [color-scheme:var(--color-scheme)] focus:outline-none focus:border-[var(--border-accent-bright)]';
export const MODAL_BUTTON_PRIMARY = 'flex-1 py-2 px-4 rounded border border-[var(--border-accent-bright)] bg-[var(--bg-accent)] text-[var(--text-primary)] text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed';
export const MODAL_BUTTON_SECONDARY = 'flex-1 py-2 px-4 rounded border border-[var(--border-color)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)] text-sm hover:text-[var(--text-primary)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed';
export const MODAL_ERROR = 'text-xs text-[var(--text-error)]';

// Shared dialog shell. Colors come only from theme CSS variables so every
// theme, including Light, renders it correctly.
export default function Modal({ title, onClose, children, footer, closeOnBackdrop = true, initialFocusRef }: ModalProps) {
    const titleId = useId();
    const panelRef = useRef<HTMLDivElement>(null);

    // Read through a ref so a parent re-render with a new onClose does not
    // re-register the listener.
    const onCloseRef = useRef(onClose);
    onCloseRef.current = onClose;

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                e.preventDefault();
                onCloseRef.current();
            }
        };
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, []);

    // Focus once on open only; refocusing on every render would steal focus
    // from whatever field the user is typing in.
    useEffect(() => {
        (initialFocusRef?.current ?? panelRef.current)?.focus();
    }, []);

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div
                className="absolute inset-0 bg-[var(--bg-primary)] opacity-80"
                onClick={closeOnBackdrop ? onClose : undefined}
                aria-hidden="true"
            />
            <div
                ref={panelRef}
                role="dialog"
                aria-modal="true"
                aria-labelledby={titleId}
                tabIndex={-1}
                className="relative w-full max-w-md mx-4 bg-[var(--bg-secondary)] border border-[var(--border-color)] rounded-xl shadow-2xl p-6 space-y-4 focus:outline-none"
            >
                <div className="flex items-center justify-between gap-4">
                    <h2 id={titleId} className="text-lg font-bold text-[var(--text-primary)]">
                        {title}
                    </h2>
                    <button
                        type="button"
                        onClick={onClose}
                        aria-label="Close dialog"
                        className="text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                    >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor" aria-hidden="true">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                {children}
                {footer && <div className="flex gap-2 pt-2">{footer}</div>}
            </div>
        </div>
    );
}
