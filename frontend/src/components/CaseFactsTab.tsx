import { useState, useEffect, useCallback, useRef } from 'react';
import {
    GetCaseFacts, CreateCaseFact, UpdateCaseFact, DeleteCaseFact, GetFactTypes,
    ListNotes, ListEvidenceNotes,
} from '../../wailsjs/go/main/App';
import { models, services } from '../../wailsjs/go/models';
import ErrorMessage from './ErrorMessage';

interface CaseFactsTabProps {
    caseId: string;
    evidenceItems: services.EvidenceResponse[];
    onNavigateToBlock: (blockId: string, evidenceItemId: string | null) => void;
    iocVersion?: number;
    /** Pre-populate the add form value and open it; cleared after first use. */
    preAddValue?: string;
    onPreAddConsumed?: () => void;
}

interface BlockEntry {
    block: services.NoteBlockResponse;
    evidenceItemId: string | null;
}

const INPUT_CLASS = 'w-full px-3 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-100 focus:outline-none focus:border-blue-500 placeholder-gray-600';
const LABEL_CLASS = 'block text-xs text-gray-400 mb-1';

function formatFactType(t: string): string {
    const special: Record<string, string> = {
        ip_address: 'IP Address',
        mac_address: 'MAC Address',
        os_version: 'OS Version',
        account_sid: 'Account SID',
        url: 'URL',
    };
    return special[t] ?? t.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
}

export default function CaseFactsTab({ caseId, evidenceItems, onNavigateToBlock, iocVersion, preAddValue, onPreAddConsumed }: CaseFactsTabProps) {
    const [facts, setFacts] = useState<models.CaseFact[]>([]);
    const [factTypes, setFactTypes] = useState<string[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    const [filterType, setFilterType] = useState('');
    const [filterEvidenceItem, setFilterEvidenceItem] = useState('');

    const [showAddForm, setShowAddForm] = useState(false);
    const [addType, setAddType] = useState('');
    const [addLabel, setAddLabel] = useState('');
    const [addValue, setAddValue] = useState('');
    const [addEvidenceId, setAddEvidenceId] = useState('');
    const [addSourceBlockId, setAddSourceBlockId] = useState('');
    const [addNotes, setAddNotes] = useState('');
    const [addError, setAddError] = useState('');
    const [adding, setAdding] = useState(false);

    const [editingFactId, setEditingFactId] = useState<string | null>(null);
    const [editType, setEditType] = useState('');
    const [editLabel, setEditLabel] = useState('');
    const [editValue, setEditValue] = useState('');
    const [editEvidenceId, setEditEvidenceId] = useState('');
    const [editSourceBlockId, setEditSourceBlockId] = useState('');
    const [editNotes, setEditNotes] = useState('');
    const [editError, setEditError] = useState('');
    const [saving, setSaving] = useState(false);

    const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
    const [deleting, setDeleting] = useState(false);

    const [showBlockPicker, setShowBlockPicker] = useState(false);
    const [blockPickerMode, setBlockPickerMode] = useState<'add' | 'edit'>('add');
    const [allBlocks, setAllBlocks] = useState<BlockEntry[]>([]);
    const [blockPickerSearch, setBlockPickerSearch] = useState('');
    const [blockEvidenceMap, setBlockEvidenceMap] = useState<Record<string, string | null>>({});
    const blocksLoadedRef = useRef(false);

    const sortedEvidenceItems = [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at));

    useEffect(() => {
        GetFactTypes()
            .then((types) => {
                const t = types || [];
                setFactTypes(t);
                if (t.length > 0) setAddType(t[0]);
            })
            .catch(() => {});
    }, []);

    const fetchFacts = useCallback(() => {
        setLoading(true);
        GetCaseFacts(caseId)
            .then((result) => setFacts(result || []))
            .catch((err: unknown) => setError(String(err)))
            .finally(() => setLoading(false));
    }, [caseId]);

    useEffect(() => {
        fetchFacts();
    }, [fetchFacts, iocVersion]);

    useEffect(() => {
        if (!preAddValue) return;
        setAddValue(preAddValue);
        setShowAddForm(true);
        onPreAddConsumed?.();
    }, [preAddValue, onPreAddConsumed]);

    const loadBlocksOnce = useCallback(async () => {
        if (blocksLoadedRef.current) return;
        blocksLoadedRef.current = true;
        try {
            const [masterBlocks, ...evidenceBlockArrays] = await Promise.all([
                ListNotes(caseId),
                ...evidenceItems.map(item => ListEvidenceNotes(caseId, item.evidence_item_id)),
            ]);
            const entries: BlockEntry[] = [
                ...(masterBlocks || []).map(b => ({ block: b, evidenceItemId: null as string | null })),
                ...evidenceBlockArrays.flatMap((blocks, i) =>
                    (blocks || []).map(b => ({ block: b, evidenceItemId: evidenceItems[i].evidence_item_id }))
                ),
            ];
            const bMap: Record<string, string | null> = {};
            entries.forEach(({ block, evidenceItemId }) => { bMap[block.block_id] = evidenceItemId; });
            setAllBlocks(entries);
            setBlockEvidenceMap(bMap);
        } catch {
            blocksLoadedRef.current = false;
        }
    }, [caseId, evidenceItems]);

    useEffect(() => {
        loadBlocksOnce();
    }, [loadBlocksOnce]);

    const openBlockPicker = (mode: 'add' | 'edit') => {
        setBlockPickerMode(mode);
        setBlockPickerSearch('');
        setShowBlockPicker(true);
        loadBlocksOnce();
    };

    const handleBlockPicked = (entry: BlockEntry) => {
        if (blockPickerMode === 'add') {
            setAddSourceBlockId(entry.block.block_id);
        } else {
            setEditSourceBlockId(entry.block.block_id);
        }
        setShowBlockPicker(false);
    };

    const handleAddFact = async () => {
        setAddError('');
        if (!addType) { setAddError('Type is required.'); return; }
        if (!addLabel.trim()) { setAddError('Description is required.'); return; }
        if (!addValue.trim()) { setAddError('Value is required.'); return; }
        setAdding(true);
        try {
            await CreateCaseFact({
                caseId,
                type: addType,
                label: addLabel.trim(),
                value: addValue.trim(),
                evidenceItemId: addEvidenceId || undefined,
                sourceBlockId: addSourceBlockId || undefined,
                notes: addNotes.trim(),
            } as models.CreateCaseFactRequest);
            setAddLabel('');
            setAddValue('');
            setAddEvidenceId('');
            setAddSourceBlockId('');
            setAddNotes('');
            setShowAddForm(false);
            fetchFacts();
        } catch (err: unknown) {
            setAddError(String(err));
        } finally {
            setAdding(false);
        }
    };

    const startEdit = (fact: models.CaseFact) => {
        setEditingFactId(fact.factId);
        setEditType(fact.type);
        setEditLabel(fact.label);
        setEditValue(fact.value);
        setEditEvidenceId(fact.evidenceItemId ?? '');
        setEditSourceBlockId(fact.sourceBlockId ?? '');
        setEditNotes(fact.notes);
        setEditError('');
    };

    const handleSaveEdit = async () => {
        if (!editingFactId) return;
        setEditError('');
        if (!editType) { setEditError('Type is required.'); return; }
        if (!editLabel.trim()) { setEditError('Description is required.'); return; }
        if (!editValue.trim()) { setEditError('Value is required.'); return; }
        setSaving(true);
        try {
            await UpdateCaseFact(editingFactId, {
                type: editType,
                label: editLabel.trim(),
                value: editValue.trim(),
                evidenceItemId: editEvidenceId || undefined,
                sourceBlockId: editSourceBlockId || undefined,
                notes: editNotes.trim(),
            } as models.UpdateCaseFactRequest);
            setEditingFactId(null);
            fetchFacts();
        } catch (err: unknown) {
            setEditError(String(err));
        } finally {
            setSaving(false);
        }
    };

    const handleDeleteFact = async (factId: string) => {
        setDeleting(true);
        try {
            await DeleteCaseFact(factId);
            setDeleteConfirmId(null);
            fetchFacts();
        } catch (err: unknown) {
            setError(String(err));
        } finally {
            setDeleting(false);
        }
    };

    const filteredFacts = facts.filter((f) => {
        if (filterType && f.type !== filterType) return false;
        if (filterEvidenceItem === 'case_level') return !f.evidenceItemId;
        if (filterEvidenceItem) return f.evidenceItemId === filterEvidenceItem;
        return true;
    });

    const filteredPickerBlocks = allBlocks.filter(({ block }) => {
        if (!blockPickerSearch.trim()) return true;
        return (block.content || '').toLowerCase().includes(blockPickerSearch.toLowerCase());
    });

    const blockPreview = (blockId: string): string => {
        const entry = allBlocks.find(({ block }) => block.block_id === blockId);
        if (!entry) return blockId.slice(0, 8) + '...';
        const content = (entry.block.content || '').trim();
        return content.slice(0, 40) || blockId.slice(0, 8) + '...';
    };

    const evidenceItemLabel = (id: string): string => {
        const item = sortedEvidenceItems.find(e => e.evidence_item_id === id);
        return item?.item_number ?? id;
    };

    return (
        <div className="space-y-4">
            {/* Toolbar */}
            <div className="flex flex-wrap items-center gap-2">
                <button
                    onClick={() => setShowAddForm(v => !v)}
                    className="px-3 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors"
                >
                    {showAddForm ? 'Cancel' : '+ Add Fact'}
                </button>
                <div className="flex-1" />
                <select
                    value={filterType}
                    onChange={(e) => setFilterType(e.target.value)}
                    className="px-2 py-1 text-xs bg-gray-800 border border-gray-700 rounded text-gray-300 focus:outline-none"
                >
                    <option value="">All Types</option>
                    {factTypes.map(t => (
                        <option key={t} value={t}>{formatFactType(t)}</option>
                    ))}
                </select>
                <select
                    value={filterEvidenceItem}
                    onChange={(e) => setFilterEvidenceItem(e.target.value)}
                    className="px-2 py-1 text-xs bg-gray-800 border border-gray-700 rounded text-gray-300 focus:outline-none"
                >
                    <option value="">All Items</option>
                    <option value="case_level">Case Level</option>
                    {sortedEvidenceItems.map(item => (
                        <option key={item.evidence_item_id} value={item.evidence_item_id}>
                            {item.item_number} - {item.name}
                        </option>
                    ))}
                </select>
            </div>

            {/* Add Fact Form */}
            {showAddForm && (
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4 space-y-3">
                    <h3 className="text-sm font-medium text-gray-300">New Fact</h3>
                    <div className="grid grid-cols-2 gap-3">
                        <div>
                            <label className={LABEL_CLASS}>Type *</label>
                            <select value={addType} onChange={(e) => setAddType(e.target.value)} className={INPUT_CLASS}>
                                {factTypes.map(t => <option key={t} value={t}>{formatFactType(t)}</option>)}
                            </select>
                        </div>
                        <div>
                            <label className={LABEL_CLASS}>Evidence Item</label>
                            <select value={addEvidenceId} onChange={(e) => setAddEvidenceId(e.target.value)} className={INPUT_CLASS}>
                                <option value="">Case Level</option>
                                {sortedEvidenceItems.map(item => (
                                    <option key={item.evidence_item_id} value={item.evidence_item_id}>
                                        {item.item_number} - {item.name}
                                    </option>
                                ))}
                            </select>
                        </div>
                    </div>
                    <div>
                        <label className={LABEL_CLASS}>Description *</label>
                        <input
                            type="text"
                            value={addLabel}
                            onChange={(e) => setAddLabel(e.target.value)}
                            placeholder="e.g. Suspect workstation IP"
                            className={INPUT_CLASS}
                            autoFocus
                        />
                    </div>
                    <div>
                        <label className={LABEL_CLASS}>Value *</label>
                        <input
                            type="text"
                            value={addValue}
                            onChange={(e) => setAddValue(e.target.value)}
                            placeholder="Fact value"
                            className={INPUT_CLASS}
                        />
                    </div>
                    <div>
                        <label className={LABEL_CLASS}>Source Block</label>
                        <div className="flex gap-2">
                            <input
                                type="text"
                                value={addSourceBlockId ? blockPreview(addSourceBlockId) : ''}
                                readOnly
                                placeholder="None"
                                className={INPUT_CLASS + ' flex-1 cursor-default'}
                            />
                            <button
                                type="button"
                                onClick={() => openBlockPicker('add')}
                                className="px-3 py-1.5 text-xs border border-gray-600 hover:border-gray-400 text-gray-300 hover:text-gray-100 rounded transition-colors whitespace-nowrap"
                            >
                                {addSourceBlockId ? 'Change' : 'Pick Block'}
                            </button>
                            {addSourceBlockId && (
                                <button
                                    type="button"
                                    onClick={() => setAddSourceBlockId('')}
                                    className="text-xs text-gray-500 hover:text-red-400 px-2 transition-colors"
                                >
                                    Clear
                                </button>
                            )}
                        </div>
                    </div>
                    <div>
                        <label className={LABEL_CLASS}>Notes</label>
                        <textarea
                            rows={2}
                            value={addNotes}
                            onChange={(e) => setAddNotes(e.target.value)}
                            placeholder="Optional notes"
                            className={INPUT_CLASS + ' resize-none'}
                        />
                    </div>
                    {addError && <p className="text-xs text-red-400">{addError}</p>}
                    <div className="flex gap-2">
                        <button
                            onClick={handleAddFact}
                            disabled={adding}
                            className="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded transition-colors"
                        >
                            {adding ? 'Adding...' : 'Add Fact'}
                        </button>
                        <button
                            onClick={() => setShowAddForm(false)}
                            className="px-4 py-1.5 text-sm border border-gray-600 hover:border-gray-400 text-gray-400 hover:text-gray-200 rounded transition-colors"
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            )}

            <ErrorMessage message={error} onDismiss={() => setError('')} />

            {/* Table */}
            {loading ? (
                <div className="flex justify-center py-8">
                    <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-400" />
                </div>
            ) : facts.length === 0 && !filterType && !filterEvidenceItem ? (
                <p className="text-gray-500 text-sm text-center py-8">
                    No case facts recorded. Add facts to track informational details about the investigation.
                </p>
            ) : filteredFacts.length === 0 ? (
                <p className="text-gray-500 text-sm text-center py-8">No facts match the current filters.</p>
            ) : (
                <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                        <thead>
                            <tr className="text-left border-b border-gray-700">
                                <th className="pb-2 pr-3 font-medium text-gray-400 w-28">Type</th>
                                <th className="pb-2 pr-3 font-medium text-gray-400 w-36">Description</th>
                                <th className="pb-2 pr-3 font-medium text-gray-400">Value</th>
                                <th className="pb-2 pr-3 font-medium text-gray-400 w-24">Evidence</th>
                                <th className="pb-2 pr-3 font-medium text-gray-400 w-20">Source</th>
                                <th className="pb-2 pr-3 font-medium text-gray-400">Notes</th>
                                <th className="pb-2 font-medium text-gray-400 w-20"></th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-800">
                            {filteredFacts.map((fact) => {
                                if (deleteConfirmId === fact.factId) {
                                    return (
                                        <tr key={fact.factId} className="bg-red-900/20">
                                            <td colSpan={7} className="py-3 px-2">
                                                <div className="flex items-center gap-3">
                                                    <span className="text-sm text-red-300">Delete this case fact? This cannot be undone.</span>
                                                    <button
                                                        onClick={() => handleDeleteFact(fact.factId)}
                                                        disabled={deleting}
                                                        className="text-xs bg-red-700 hover:bg-red-600 disabled:bg-gray-700 text-white px-3 py-1 rounded transition-colors"
                                                    >
                                                        {deleting ? 'Deleting...' : 'Delete'}
                                                    </button>
                                                    <button
                                                        onClick={() => setDeleteConfirmId(null)}
                                                        className="text-xs text-gray-400 hover:text-gray-200 px-3 py-1 rounded border border-gray-600 transition-colors"
                                                    >
                                                        Cancel
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    );
                                }
                                if (editingFactId === fact.factId) {
                                    return (
                                        <tr key={fact.factId} className="bg-gray-800/60">
                                            <td colSpan={7} className="py-3 px-2">
                                                <div className="space-y-3">
                                                    <div className="grid grid-cols-3 gap-3">
                                                        <div>
                                                            <label className={LABEL_CLASS}>Type *</label>
                                                            <select value={editType} onChange={(e) => setEditType(e.target.value)} className={INPUT_CLASS}>
                                                                {factTypes.map(t => <option key={t} value={t}>{formatFactType(t)}</option>)}
                                                            </select>
                                                        </div>
                                                        <div>
                                                            <label className={LABEL_CLASS}>Description *</label>
                                                            <input type="text" value={editLabel} onChange={(e) => setEditLabel(e.target.value)} className={INPUT_CLASS} />
                                                        </div>
                                                        <div>
                                                            <label className={LABEL_CLASS}>Value *</label>
                                                            <input type="text" value={editValue} onChange={(e) => setEditValue(e.target.value)} className={INPUT_CLASS} />
                                                        </div>
                                                    </div>
                                                    <div className="grid grid-cols-3 gap-3">
                                                        <div>
                                                            <label className={LABEL_CLASS}>Evidence Item</label>
                                                            <select value={editEvidenceId} onChange={(e) => setEditEvidenceId(e.target.value)} className={INPUT_CLASS}>
                                                                <option value="">Case Level</option>
                                                                {sortedEvidenceItems.map(item => (
                                                                    <option key={item.evidence_item_id} value={item.evidence_item_id}>
                                                                        {item.item_number} - {item.name}
                                                                    </option>
                                                                ))}
                                                            </select>
                                                        </div>
                                                        <div>
                                                            <label className={LABEL_CLASS}>Source Block</label>
                                                            <div className="flex gap-1.5">
                                                                <input
                                                                    type="text"
                                                                    value={editSourceBlockId ? blockPreview(editSourceBlockId) : ''}
                                                                    readOnly
                                                                    placeholder="None"
                                                                    className="flex-1 px-2 py-1 bg-gray-800 border border-gray-700 rounded text-xs text-gray-400 cursor-default focus:outline-none"
                                                                />
                                                                <button
                                                                    type="button"
                                                                    onClick={() => openBlockPicker('edit')}
                                                                    className="px-2 py-1 text-xs border border-gray-600 hover:border-gray-400 text-gray-300 rounded transition-colors"
                                                                >
                                                                    {editSourceBlockId ? 'Change' : 'Pick'}
                                                                </button>
                                                                {editSourceBlockId && (
                                                                    <button
                                                                        type="button"
                                                                        onClick={() => setEditSourceBlockId('')}
                                                                        className="text-xs text-gray-500 hover:text-red-400 px-1 transition-colors"
                                                                    >
                                                                        X
                                                                    </button>
                                                                )}
                                                            </div>
                                                        </div>
                                                        <div>
                                                            <label className={LABEL_CLASS}>Notes</label>
                                                            <textarea rows={1} value={editNotes} onChange={(e) => setEditNotes(e.target.value)} className={INPUT_CLASS + ' resize-none'} />
                                                        </div>
                                                    </div>
                                                    {fact.sourceIocId && (
                                                        <p className="text-xs text-gray-500">From IOC promotion -- source IOC cannot be changed.</p>
                                                    )}
                                                    {editError && <p className="text-xs text-red-400">{editError}</p>}
                                                    <div className="flex gap-2">
                                                        <button
                                                            onClick={handleSaveEdit}
                                                            disabled={saving}
                                                            className="px-4 py-1.5 text-xs bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded transition-colors"
                                                        >
                                                            {saving ? 'Saving...' : 'Save'}
                                                        </button>
                                                        <button
                                                            onClick={() => setEditingFactId(null)}
                                                            className="px-4 py-1.5 text-xs border border-gray-600 hover:border-gray-400 text-gray-400 hover:text-gray-200 rounded transition-colors"
                                                        >
                                                            Cancel
                                                        </button>
                                                    </div>
                                                </div>
                                            </td>
                                        </tr>
                                    );
                                }
                                return (
                                    <tr key={fact.factId} className="hover:bg-gray-800">
                                        <td className="py-2 pr-3 align-middle text-xs text-gray-300">{formatFactType(fact.type)}</td>
                                        <td className="py-2 pr-3 align-middle text-gray-200">{fact.label}</td>
                                        <td className="py-2 pr-3 align-middle font-mono text-xs text-gray-200">{fact.value}</td>
                                        <td className="py-2 pr-3 align-middle text-xs">
                                            {fact.evidenceItemId
                                                ? <span className="text-gray-300">{evidenceItemLabel(fact.evidenceItemId)}</span>
                                                : <span className="text-gray-600">Case Level</span>
                                            }
                                        </td>
                                        <td className="py-2 pr-3 align-middle">
                                            <div className="flex flex-col gap-0.5">
                                                {fact.sourceIocId && (
                                                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-[var(--bg-accent)] text-[var(--border-accent-bright)] border border-[var(--border-accent-bright)]">
                                                        IOC
                                                    </span>
                                                )}
                                                {fact.sourceBlockId && (
                                                    <button
                                                        onClick={() => {
                                                            const evId = blockEvidenceMap[fact.sourceBlockId!] ?? null;
                                                            onNavigateToBlock(fact.sourceBlockId!, evId);
                                                        }}
                                                        className="text-blue-400 hover:text-blue-300 underline text-xs transition-colors text-left"
                                                    >
                                                        Block
                                                    </button>
                                                )}
                                            </div>
                                        </td>
                                        <td className="py-2 pr-3 align-middle text-xs text-gray-500">{fact.notes}</td>
                                        <td className="py-2 align-middle">
                                            <div className="flex gap-2">
                                                <button
                                                    onClick={() => startEdit(fact)}
                                                    className="text-xs text-gray-400 hover:text-blue-400 transition-colors"
                                                >
                                                    Edit
                                                </button>
                                                <button
                                                    onClick={() => setDeleteConfirmId(fact.factId)}
                                                    className="text-xs text-gray-500 hover:text-red-400 transition-colors"
                                                >
                                                    Delete
                                                </button>
                                            </div>
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                </div>
            )}

            {/* Block Picker Modal */}
            {showBlockPicker && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
                    <div className="bg-gray-800 border border-gray-700 rounded-lg shadow-xl w-full max-w-lg p-4 space-y-3">
                        <div className="flex items-center justify-between">
                            <h3 className="text-sm font-semibold text-gray-200">Pick Source Block</h3>
                            <button onClick={() => setShowBlockPicker(false)} className="text-gray-500 hover:text-gray-300">
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                                </svg>
                            </button>
                        </div>
                        <input
                            type="text"
                            placeholder="Search blocks..."
                            value={blockPickerSearch}
                            onChange={(e) => setBlockPickerSearch(e.target.value)}
                            className={INPUT_CLASS}
                            autoFocus
                        />
                        <div className="max-h-72 overflow-y-auto space-y-1">
                            {filteredPickerBlocks.length === 0 ? (
                                <p className="text-xs text-gray-500 py-2 text-center">No committed blocks found.</p>
                            ) : (
                                filteredPickerBlocks.map(({ block, evidenceItemId }) => (
                                    <button
                                        key={block.block_id}
                                        onClick={() => handleBlockPicked({ block, evidenceItemId })}
                                        className="w-full text-left px-3 py-2 bg-gray-750 border border-gray-700 rounded hover:border-blue-600 hover:bg-gray-700 transition-colors space-y-0.5"
                                    >
                                        <div className="flex gap-2 text-xs text-gray-500 font-mono">
                                            <span>{block.created_at}</span>
                                            {evidenceItemId && (
                                                <span className="text-gray-600">{evidenceItemLabel(evidenceItemId)}</span>
                                            )}
                                        </div>
                                        <p className="text-xs text-gray-300 truncate">{(block.content || '').slice(0, 120)}</p>
                                    </button>
                                ))
                            )}
                        </div>
                        <button
                            onClick={() => setShowBlockPicker(false)}
                            className="w-full py-1.5 text-sm text-gray-400 hover:text-gray-200 border border-gray-700 rounded transition-colors"
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}
