import { useState } from 'react';
import { services } from '../../wailsjs/go/models';
import { UpdateCaseClassification, ToggleAttorneyClientPrivilege } from '../../wailsjs/go/main/App';
import ClassificationBadge from './ClassificationBadge';

const CLASSIFICATION_LEVELS = [
    { value: 'UNCLASSIFIED', label: 'Unclassified' },
    { value: 'CONFIDENTIAL', label: 'Confidential' },
    { value: 'RESTRICTED', label: 'Restricted' },
    { value: 'SECRET', label: 'Secret' },
    { value: 'TOP SECRET', label: 'Top Secret' },
];

interface CaseOverviewTabProps {
    caseData: services.CaseResponse;
    onClassificationChanged?: (level: string) => void;
    onPrivilegeChanged?: (value: boolean) => void;
}

export default function CaseOverviewTab({ caseData, onClassificationChanged, onPrivilegeChanged }: CaseOverviewTabProps) {
    const createdDate = new Date(caseData.created_at).toLocaleString();
    const [classification, setClassification] = useState(caseData.classification);
    const [editing, setEditing] = useState(false);
    const [draft, setDraft] = useState(caseData.classification);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    const [privilege, setPrivilege] = useState(caseData.attorney_client_privilege);
    const [privilegeSaving, setPrivilegeSaving] = useState(false);

    const handleEdit = () => {
        setDraft(classification);
        setError('');
        setEditing(true);
    };

    const handleCancel = () => {
        setEditing(false);
        setError('');
    };

    const handleTogglePrivilege = async () => {
        const prev = privilege;
        const next = !prev;
        setPrivilege(next);
        setPrivilegeSaving(true);
        try {
            await ToggleAttorneyClientPrivilege(caseData.case_id);
            onPrivilegeChanged?.(next);
        } catch {
            setPrivilege(prev);
        } finally {
            setPrivilegeSaving(false);
        }
    };

    const handleSave = async () => {
        if (draft === classification) {
            setEditing(false);
            return;
        }
        const prev = classification;
        setClassification(draft);
        setEditing(false);
        setSaving(true);
        setError('');
        try {
            await UpdateCaseClassification(caseData.case_id, draft);
            onClassificationChanged?.(draft);
        } catch (err: unknown) {
            setClassification(prev);
            setDraft(prev);
            setEditing(true);
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setSaving(false);
        }
    };

    return (
        <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Case Number</label>
                    <p className="text-gray-100 font-mono mt-1">{caseData.case_number}</p>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Ticket Number</label>
                    <p className="text-gray-100 mt-1">{caseData.ticket_number || '—'}</p>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Examiner</label>
                    <p className="text-gray-100 mt-1">{caseData.examiner_name}</p>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Organization</label>
                    <p className="text-gray-100 mt-1">{caseData.organization}</p>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Classification</label>
                    <div className="mt-1">
                        {editing ? (
                            <div className="space-y-2">
                                <select
                                    value={draft}
                                    onChange={(e) => setDraft(e.target.value)}
                                    disabled={saving}
                                    autoFocus
                                    className="w-full px-2 py-1.5 bg-gray-700 border border-gray-600 rounded text-sm text-gray-100 focus:outline-none focus:border-blue-500"
                                >
                                    {CLASSIFICATION_LEVELS.map((lvl) => (
                                        <option key={lvl.value} value={lvl.value}>{lvl.label}</option>
                                    ))}
                                </select>
                                {error && <p className="text-xs text-red-400">{error}</p>}
                                <div className="flex gap-2">
                                    <button
                                        onClick={handleSave}
                                        disabled={saving}
                                        className="px-3 py-1 text-xs bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded transition-colors"
                                    >
                                        {saving ? 'Saving...' : 'Save'}
                                    </button>
                                    <button
                                        onClick={handleCancel}
                                        disabled={saving}
                                        className="px-3 py-1 text-xs text-gray-400 hover:text-gray-200 border border-gray-600 rounded transition-colors"
                                    >
                                        Cancel
                                    </button>
                                </div>
                            </div>
                        ) : (
                            <div className="flex items-center gap-2">
                                <ClassificationBadge level={classification} />
                                <button
                                    onClick={handleEdit}
                                    className="text-xs text-gray-500 hover:text-gray-300 transition-colors"
                                >
                                    Edit
                                </button>
                            </div>
                        )}
                    </div>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Created</label>
                    <p className="text-gray-100 mt-1">{createdDate}</p>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Evidence numbering</label>
                    <p className="text-gray-100 font-mono mt-1">
                        {caseData.evidence_prefix}{String(1).padStart(caseData.evidence_seq_digits, '0')},{' '}
                        {caseData.evidence_prefix}{String(2).padStart(caseData.evidence_seq_digits, '0')}...{' '}
                        <span className="text-gray-500">({caseData.evidence_seq_digits} digits)</span>
                    </p>
                </div>
            </div>
            <div className="flex items-center gap-3 px-1">
                {privilege ? (
                    <>
                        <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold bg-amber-700 text-amber-100">
                            Attorney-Client Privilege
                        </span>
                        <button
                            onClick={handleTogglePrivilege}
                            disabled={privilegeSaving}
                            className="text-xs text-gray-500 hover:text-gray-300 disabled:opacity-50 transition-colors"
                        >
                            Remove
                        </button>
                    </>
                ) : (
                    <button
                        onClick={handleTogglePrivilege}
                        disabled={privilegeSaving}
                        className="text-xs text-gray-500 hover:text-gray-300 disabled:opacity-50 border border-gray-700 hover:border-gray-500 rounded px-2 py-1 transition-colors"
                    >
                        + Add privilege marker
                    </button>
                )}
            </div>
            {caseData.description && (
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Description</label>
                    <p className="text-gray-300 mt-2 whitespace-pre-wrap">{caseData.description}</p>
                </div>
            )}
        </div>
    );
}
