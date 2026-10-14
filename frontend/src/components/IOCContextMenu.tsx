import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { UpdateIndicatorStatus } from '../../wailsjs/go/main/App';
import type { IOCStatus, IOCType } from '../utils/iocTypes';
import { IOC_PATTERNS } from '../utils/iocPatterns';

interface IOCContextMenuProps {
    x: number;
    y: number;
    indicatorId: string;
    iocType: IOCType;
    iocValue: string;
    iocStatus: IOCStatus;
    onClose: () => void;
    onStatusChanged: (indicatorId: string, newStatus: IOCStatus) => void;
    onPromoteRequested?: () => void;
    // Selection-based options appended below IOC options when a text selection
    // was active at right-click time.
    selectionText?: string;
    onMarkAsIOC?: () => void;
    onAddAsCaseFact?: () => void;
}

const STATUS_ACTIONS: Record<IOCStatus, { label: string; next: IOCStatus }[]> = {
    detected: [
        { label: 'Confirm as IOC', next: 'confirmed' },
        { label: 'Mark as False Positive', next: 'false_positive' },
    ],
    confirmed: [
        { label: 'Unconfirm (back to detected)', next: 'detected' },
        { label: 'Mark as False Positive', next: 'false_positive' },
    ],
    false_positive: [
        { label: 'Confirm as IOC', next: 'confirmed' },
        { label: 'Restore to Detected', next: 'detected' },
    ],
    promoted: [],
};

const STATUS_LABELS: Record<IOCStatus, string> = {
    detected: 'Detected',
    confirmed: 'Confirmed',
    false_positive: 'False Positive',
    promoted: 'Promoted',
};

export default function IOCContextMenu({
    x, y, indicatorId, iocType, iocValue, iocStatus, onClose, onStatusChanged, onPromoteRequested,
    selectionText, onMarkAsIOC, onAddAsCaseFact,
}: IOCContextMenuProps) {
    const menuRef = useRef<HTMLDivElement>(null);
    const typeLabel = IOC_PATTERNS.find((p) => p.type === iocType)?.label ?? iocType;

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose();
        };
        const handleClick = (e: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
                onClose();
            }
        };
        document.addEventListener('keydown', handleKeyDown);
        document.addEventListener('mousedown', handleClick);
        return () => {
            document.removeEventListener('keydown', handleKeyDown);
            document.removeEventListener('mousedown', handleClick);
        };
    }, [onClose]);

    const [busy, setBusy] = useState(false);
    const [error, setError] = useState('');

    // On failure the menu stays open with the backend's message so the
    // examiner sees that the status did not change.
    const handleAction = async (next: IOCStatus) => {
        if (busy) return;
        setBusy(true);
        setError('');
        try {
            await UpdateIndicatorStatus(indicatorId, next);
        } catch (err: unknown) {
            setError(String(err));
            setBusy(false);
            return;
        }
        onStatusChanged(indicatorId, next);
        onClose();
    };

    const handlePromote = () => {
        onClose();
        onPromoteRequested?.();
    };

    const hasSelectionOptions = !!selectionText && (!!onMarkAsIOC || !!onAddAsCaseFact);
    // Estimates for the first paint; the layout effect below replaces them with
    // the measured size, which includes an error message when one is shown.
    const MENU_W = 220;
    const MENU_H = hasSelectionOptions ? 210 : 140;
    const [size, setSize] = useState({ w: MENU_W, h: MENU_H });

    useLayoutEffect(() => {
        const el = menuRef.current;
        if (!el) return;
        const w = el.offsetWidth;
        const h = el.offsetHeight;
        setSize((prev) => (prev.w === w && prev.h === h ? prev : { w, h }));
    }, [error, hasSelectionOptions, iocStatus]);

    const left = Math.max(0, x + size.w > window.innerWidth  ? x - size.w : x);
    const top  = Math.max(0, y + size.h > window.innerHeight ? y - size.h : y);

    const showPromote = onPromoteRequested && (iocStatus === 'detected' || iocStatus === 'confirmed');

    return (
        <div
            ref={menuRef}
            className="fixed z-50 bg-gray-800 border border-gray-600 rounded-lg shadow-xl py-1 min-w-48"
            style={{ left, top }}
        >
            <div className="px-3 py-2 border-b border-gray-700">
                <div className="flex items-center gap-2 mb-0.5">
                    <span className="text-xs font-semibold text-blue-400 uppercase tracking-wide">{typeLabel}</span>
                    <span className="text-xs text-gray-500">{STATUS_LABELS[iocStatus]}</span>
                </div>
                <div className="text-xs text-gray-300 font-mono truncate max-w-64" title={iocValue}>
                    {iocValue}
                </div>
            </div>
            <div className="py-1">
                {STATUS_ACTIONS[iocStatus].map((action) => (
                    <button
                        key={action.next}
                        onClick={() => handleAction(action.next)}
                        disabled={busy}
                        className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-gray-700 transition-colors disabled:opacity-50"
                    >
                        {action.label}
                    </button>
                ))}
                {error && (
                    <p role="alert" className="px-3 py-1.5 text-xs text-[var(--text-error)] max-w-64 break-words">
                        {error}
                    </p>
                )}
                {showPromote && STATUS_ACTIONS[iocStatus].length > 0 && (
                    <div className="border-t border-gray-700 my-1" />
                )}
                {showPromote && (
                    <button
                        onClick={handlePromote}
                        className="w-full text-left px-3 py-1.5 text-sm text-green-400 hover:bg-gray-700 transition-colors"
                    >
                        Promote to Case Facts
                    </button>
                )}
                {iocStatus === 'promoted' && (
                    <p className="px-3 py-1.5 text-xs text-gray-500">Promoted to Case Facts</p>
                )}
                {hasSelectionOptions && (
                    <>
                        <div className="border-t border-gray-700 my-1" />
                        <div className="px-3 py-1 text-xs text-gray-500 truncate" title={selectionText}>
                            Selection: &quot;{selectionText!.length > 28 ? selectionText!.slice(0, 28) + '…' : selectionText}&quot;
                        </div>
                        {onMarkAsIOC && (
                            <button
                                onClick={() => { onClose(); onMarkAsIOC!(); }}
                                className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-gray-700 transition-colors"
                            >
                                Mark as IOC
                            </button>
                        )}
                        {onAddAsCaseFact && (
                            <button
                                onClick={() => { onClose(); onAddAsCaseFact!(); }}
                                className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-gray-700 transition-colors"
                            >
                                Add as Case Fact
                            </button>
                        )}
                    </>
                )}
            </div>
        </div>
    );
}
