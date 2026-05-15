import { useEffect, useRef } from 'react';
import { UpdateIOCStatus } from '../../wailsjs/go/main/App';
import type { IOCStatus, IOCType } from '../utils/iocTypes';
import { IOC_PATTERNS } from '../utils/iocPatterns';

interface IOCContextMenuProps {
    x: number;
    y: number;
    iocId: string;
    iocType: IOCType;
    iocValue: string;
    iocStatus: IOCStatus;
    onClose: () => void;
    onStatusChanged: (iocId: string, newStatus: IOCStatus) => void;
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
};

const STATUS_LABELS: Record<IOCStatus, string> = {
    detected: 'Detected',
    confirmed: 'Confirmed',
    false_positive: 'False Positive',
};

export default function IOCContextMenu({
    x, y, iocId, iocType, iocValue, iocStatus, onClose, onStatusChanged,
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

    const handleAction = async (next: IOCStatus) => {
        try {
            await UpdateIOCStatus(iocId, next);
            onStatusChanged(iocId, next);
        } catch { /* ignore */ }
        onClose();
    };

    const MENU_W = 200;
    const MENU_H = 120;
    const left = Math.max(0, x + MENU_W > window.innerWidth  ? x - MENU_W : x);
    const top  = Math.max(0, y + MENU_H > window.innerHeight ? y - MENU_H : y);

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
                        className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-gray-700 transition-colors"
                    >
                        {action.label}
                    </button>
                ))}
            </div>
        </div>
    );
}
