import { useEffect, useRef, useState } from 'react';
import { GetFactTypes, PromoteIndicatorToFact } from '../../wailsjs/go/main/App';
import { models, services } from '../../wailsjs/go/models';
import { FACT_TYPES, formatFactType, iocTypeToFactType } from '../utils/factTypes';
import type { IOCType } from '../utils/iocTypes';
import Modal, { MODAL_BUTTON_PRIMARY, MODAL_BUTTON_SECONDARY, MODAL_ERROR, MODAL_INPUT, MODAL_LABEL, MODAL_SELECT } from './Modal';

interface PromoteFactModalProps {
    caseId: string;
    indicatorId: string;
    iocType: IOCType;
    iocValue: string;
    evidenceItems: services.EvidenceResponse[];
    /** '' means Case Level. */
    defaultEvidenceItemId: string;
    onClose: () => void;
    onPromoted: (fact: models.CaseFact) => void;
}

export default function PromoteFactModal({
    caseId, indicatorId, iocType, iocValue, evidenceItems, defaultEvidenceItemId, onClose, onPromoted,
}: PromoteFactModalProps) {
    const [factTypes, setFactTypes] = useState<string[]>(FACT_TYPES);
    const [factType, setFactType] = useState(() => iocTypeToFactType(iocType));
    const [evidenceId, setEvidenceId] = useState(defaultEvidenceItemId);
    const [label, setLabel] = useState('');
    const [value, setValue] = useState(iocValue);
    const [notes, setNotes] = useState('');
    const [error, setError] = useState('');
    const [promoting, setPromoting] = useState(false);
    const labelRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        GetFactTypes()
            .then((types) => { if (types?.length) setFactTypes(types); })
            .catch((err) => console.error('GetFactTypes failed; using built-in fact types:', err));
    }, []);

    const sortedEvidence = [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at));
    const hasInput = label.trim() !== '' || notes.trim() !== '' || value !== iocValue;

    // Closing while the request is in flight would hide its result.
    const handleClose = () => {
        if (!promoting) onClose();
    };

    const handlePromote = async () => {
        if (promoting) return;
        setError('');
        if (!label.trim()) { setError('Description is required.'); return; }
        if (!value.trim()) { setError('Value is required.'); return; }
        setPromoting(true);
        try {
            const fact = await PromoteIndicatorToFact(indicatorId, {
                caseId,
                type: factType,
                label: label.trim(),
                value: value.trim(),
                evidenceItemId: evidenceId || undefined,
                notes: notes.trim(),
            } as models.CreateCaseFactRequest);
            onPromoted(fact);
        } catch (err: unknown) {
            setError(String(err));
        } finally {
            setPromoting(false);
        }
    };

    return (
        <Modal
            title="Promote to Case Facts"
            onClose={handleClose}
            closeOnBackdrop={!promoting && !hasInput}
            initialFocusRef={labelRef}
            footer={
                <>
                    <button type="button" onClick={handlePromote} disabled={promoting} className={MODAL_BUTTON_PRIMARY}>
                        {promoting ? 'Promoting...' : 'Promote to Case Facts'}
                    </button>
                    <button type="button" onClick={handleClose} disabled={promoting} className={MODAL_BUTTON_SECONDARY}>
                        Cancel
                    </button>
                </>
            }
        >
            <div className="grid grid-cols-2 gap-3">
                <div>
                    <label htmlFor="promote-fact-type" className={MODAL_LABEL}>Type *</label>
                    <select id="promote-fact-type" value={factType} onChange={(e) => setFactType(e.target.value)} className={MODAL_SELECT}>
                        {factTypes.map((t) => <option key={t} value={t}>{formatFactType(t)}</option>)}
                    </select>
                </div>
                <div>
                    <label htmlFor="promote-fact-evidence" className={MODAL_LABEL}>Evidence Item</label>
                    <select id="promote-fact-evidence" value={evidenceId} onChange={(e) => setEvidenceId(e.target.value)} className={MODAL_SELECT}>
                        <option value="">Case Level</option>
                        {sortedEvidence.map((item) => (
                            <option key={item.evidence_item_id} value={item.evidence_item_id}>
                                {item.item_number} - {item.name}
                            </option>
                        ))}
                    </select>
                </div>
            </div>
            <div>
                <label htmlFor="promote-fact-label" className={MODAL_LABEL}>Description *</label>
                <input
                    id="promote-fact-label"
                    ref={labelRef}
                    type="text"
                    value={label}
                    onChange={(e) => setLabel(e.target.value)}
                    placeholder="e.g. Suspect workstation IP"
                    className={MODAL_INPUT}
                />
            </div>
            <div>
                <label htmlFor="promote-fact-value" className={MODAL_LABEL}>Value *</label>
                <input
                    id="promote-fact-value"
                    type="text"
                    value={value}
                    onChange={(e) => setValue(e.target.value)}
                    className={MODAL_INPUT}
                />
            </div>
            <div>
                <label htmlFor="promote-fact-notes" className={MODAL_LABEL}>Notes</label>
                <textarea
                    id="promote-fact-notes"
                    rows={2}
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    placeholder="Optional notes"
                    className={MODAL_INPUT + ' resize-none'}
                />
            </div>
            {error && <p role="alert" className={MODAL_ERROR}>{error}</p>}
        </Modal>
    );
}
