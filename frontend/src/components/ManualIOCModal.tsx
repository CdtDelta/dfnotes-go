import { useState } from 'react';
import { CreateManualIOC } from '../../wailsjs/go/main/App';
import { IOC_PATTERNS, detectIOCType } from '../utils/iocPatterns';

interface ManualIOCModalProps {
    value: string;
    blockId: string;
    caseId: string;
    evidenceItemId: string; // empty string treated as null by the backend
    onClose: () => void;
    /** Called after CreateManualIOC succeeds; triggers iocVersion increment in the parent. */
    onSave: () => void;
}

const INPUT = 'w-full px-3 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-100 focus:outline-none focus:border-blue-500 placeholder-gray-600';
const LABEL = 'block text-xs text-gray-400 mb-1';

export default function ManualIOCModal({ value, blockId, caseId, evidenceItemId, onClose, onSave }: ManualIOCModalProps) {
    const [iocType, setIocType] = useState<string>(() => detectIOCType(value) ?? '');
    const [editValue, setEditValue] = useState(value);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');

    const handleSubmit = async () => {
        setError('');
        if (!iocType) { setError('Type is required.'); return; }
        if (!editValue.trim()) { setError('Value is required.'); return; }
        setSaving(true);
        try {
            await CreateManualIOC(caseId, blockId, evidenceItemId, iocType, editValue.trim());
            onSave();
            onClose();
        } catch (err: unknown) {
            setError(String(err));
        } finally {
            setSaving(false);
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) handleSubmit();
        if (e.key === 'Escape') onClose();
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
            <div className="bg-gray-800 border border-gray-700 rounded-lg shadow-xl w-full max-w-md p-4 space-y-3">
                <div className="flex items-center justify-between">
                    <h3 className="text-sm font-semibold text-gray-200">Mark as IOC</h3>
                    <button onClick={onClose} className="text-gray-500 hover:text-gray-300">
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                <div>
                    <label className={LABEL}>Type *</label>
                    <select
                        value={iocType}
                        onChange={(e) => setIocType(e.target.value)}
                        className={INPUT}
                        autoFocus={!iocType}
                    >
                        {!iocType && <option value="">-- select type --</option>}
                        {IOC_PATTERNS.map((p) => (
                            <option key={p.type} value={p.type}>{p.label}</option>
                        ))}
                    </select>
                </div>
                <div>
                    <label className={LABEL}>Value (stored raw, defanged on display) *</label>
                    <input
                        type="text"
                        value={editValue}
                        onChange={(e) => setEditValue(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className={INPUT}
                        autoFocus={!!iocType}
                    />
                </div>
                {error && <p className="text-xs text-red-400">{error}</p>}
                <div className="flex gap-2">
                    <button
                        onClick={handleSubmit}
                        disabled={saving}
                        className="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded transition-colors"
                    >
                        {saving ? 'Saving...' : 'Save IOC'}
                    </button>
                    <button
                        onClick={onClose}
                        className="px-4 py-1.5 text-sm border border-gray-600 hover:border-gray-400 text-gray-400 hover:text-gray-200 rounded transition-colors"
                    >
                        Cancel
                    </button>
                </div>
            </div>
        </div>
    );
}
