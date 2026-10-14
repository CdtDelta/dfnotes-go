import { useRef, useState } from 'react';
import { MarkManualIOC } from '../../wailsjs/go/main/App';
import { IOC_PATTERNS, detectIOCType } from '../utils/iocPatterns';
import type { UpsertResult } from '../utils/iocTypes';
import Modal, { MODAL_BUTTON_PRIMARY, MODAL_BUTTON_SECONDARY, MODAL_ERROR, MODAL_INPUT, MODAL_LABEL, MODAL_SELECT } from './Modal';

interface ManualIOCModalProps {
    value: string;
    blockId: string;
    caseId: string;
    onClose: () => void;
    /** Called after MarkManualIOC records a change; triggers iocVersion increment in the parent. */
    onSave: () => void;
}

export default function ManualIOCModal({ value, blockId, caseId, onClose, onSave }: ManualIOCModalProps) {
    const [iocType, setIocType] = useState<string>(() => detectIOCType(value) ?? '');
    const [editValue, setEditValue] = useState(value);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    // Set when the backend reports the value is already tracked on this block.
    const [alreadyTracked, setAlreadyTracked] = useState('');

    const selectRef = useRef<HTMLSelectElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);
    // Chosen once at mount, matching the old autoFocus behavior: the type
    // select when no type was detected, otherwise the value field.
    const [initialFocusRef] = useState(() => (detectIOCType(value) ? inputRef : selectRef));

    const handleClose = () => {
        if (!saving) onClose();
    };

    const handleSubmit = async () => {
        if (saving) return;
        if (alreadyTracked) { onClose(); return; }
        setError('');
        if (!iocType) { setError('Type is required.'); return; }
        const trimmed = editValue.trim();
        if (!trimmed) { setError('Value is required.'); return; }
        setSaving(true);
        try {
            const result = (await MarkManualIOC(caseId, blockId, iocType, trimmed)) as UpsertResult;
            if (result.already_tracked) {
                setAlreadyTracked(`Already tracked as ${iocType}:${trimmed} (${result.status.replace('_', ' ')}).`);
                return;
            }
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
    };

    // Editing after an "already tracked" answer means the user wants to try a
    // different type or value, so the form goes back to saving.
    const clearAlreadyTracked = () => {
        if (alreadyTracked) setAlreadyTracked('');
    };

    const primaryLabel = alreadyTracked ? 'Close' : saving ? 'Saving...' : 'Save IOC';

    return (
        <Modal
            title="Mark as IOC"
            onClose={handleClose}
            closeOnBackdrop={!!alreadyTracked}
            initialFocusRef={initialFocusRef}
            footer={
                <>
                    <button type="button" onClick={handleSubmit} disabled={saving} className={MODAL_BUTTON_PRIMARY}>
                        {primaryLabel}
                    </button>
                    {!alreadyTracked && (
                        <button type="button" onClick={handleClose} disabled={saving} className={MODAL_BUTTON_SECONDARY}>
                            Cancel
                        </button>
                    )}
                </>
            }
        >
            <div>
                <label htmlFor="manual-ioc-type" className={MODAL_LABEL}>Type *</label>
                <select
                    id="manual-ioc-type"
                    ref={selectRef}
                    value={iocType}
                    onChange={(e) => { setIocType(e.target.value); clearAlreadyTracked(); }}
                    className={MODAL_SELECT}
                >
                    {!iocType && <option value="">-- select type --</option>}
                    {IOC_PATTERNS.map((p) => (
                        <option key={p.type} value={p.type}>{p.label}</option>
                    ))}
                </select>
            </div>
            <div>
                <label htmlFor="manual-ioc-value" className={MODAL_LABEL}>Value (stored raw, defanged on display) *</label>
                <input
                    id="manual-ioc-value"
                    ref={inputRef}
                    type="text"
                    value={editValue}
                    onChange={(e) => { setEditValue(e.target.value); clearAlreadyTracked(); }}
                    onKeyDown={handleKeyDown}
                    className={MODAL_INPUT}
                />
            </div>
            {alreadyTracked && <p className="text-xs text-[var(--text-secondary)]">{alreadyTracked}</p>}
            {error && <p role="alert" className={MODAL_ERROR}>{error}</p>}
        </Modal>
    );
}
