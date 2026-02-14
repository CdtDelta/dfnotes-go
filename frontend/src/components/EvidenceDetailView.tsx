import { useState } from 'react';
import { UpdateEvidenceStatus, AddCustodyEntry, TagEvidence, UntagEvidence } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';
import TagBadge from './TagBadge';
import TagSelector from './TagSelector';
import ErrorMessage from './ErrorMessage';

interface EvidenceDetailViewProps {
    item: services.EvidenceResponse;
    onBack: () => void;
    onUpdated: (item: services.EvidenceResponse) => void;
}

const STATUSES = ['COLLECTED', 'ANALYZING', 'PROCESSED', 'ARCHIVED', 'WITHDRAWN'];

const TYPE_COLORS: Record<string, string> = {
    DISK: 'bg-blue-900/50 text-blue-300 border-blue-700',
    MEMORY: 'bg-purple-900/50 text-purple-300 border-purple-700',
    NETWORK: 'bg-cyan-900/50 text-cyan-300 border-cyan-700',
    LOGS: 'bg-yellow-900/50 text-yellow-300 border-yellow-700',
    MALWARE: 'bg-red-900/50 text-red-300 border-red-700',
    OTHER: 'bg-gray-800/50 text-gray-300 border-gray-600',
};

const STATUS_COLORS: Record<string, string> = {
    COLLECTED: 'bg-green-900/50 text-green-300 border-green-700',
    ANALYZING: 'bg-yellow-900/50 text-yellow-300 border-yellow-700',
    PROCESSED: 'bg-blue-900/50 text-blue-300 border-blue-700',
    ARCHIVED: 'bg-gray-800/50 text-gray-400 border-gray-600',
    WITHDRAWN: 'bg-red-900/50 text-red-400 border-red-700',
};

export default function EvidenceDetailView({ item, onBack, onUpdated }: EvidenceDetailViewProps) {
    const [newStatus, setNewStatus] = useState(item.status);
    const [updatingStatus, setUpdatingStatus] = useState(false);
    const [custodyAction, setCustodyAction] = useState('');
    const [custodyDescription, setCustodyDescription] = useState('');
    const [addingEntry, setAddingEntry] = useState(false);
    const [error, setError] = useState('');

    const isWithdrawn = item.status === 'WITHDRAWN';
    const typeColor = TYPE_COLORS[item.evidence_type] || TYPE_COLORS.OTHER;
    const statusColor = STATUS_COLORS[item.status] || STATUS_COLORS.COLLECTED;
    const tags = item.tags || [];

    const handleStatusUpdate = async () => {
        if (newStatus === item.status) return;
        setUpdatingStatus(true);
        setError('');
        try {
            const updated = await UpdateEvidenceStatus({
                evidence_item_id: item.evidence_item_id,
                status: newStatus,
            } as services.UpdateEvidenceStatusRequest);
            onUpdated(updated);
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setUpdatingStatus(false);
        }
    };

    const handleAddCustodyEntry = async () => {
        if (!custodyAction.trim()) return;
        setAddingEntry(true);
        setError('');
        try {
            const updated = await AddCustodyEntry({
                evidence_item_id: item.evidence_item_id,
                action: custodyAction.trim(),
                description: custodyDescription.trim(),
            } as services.AddCustodyEntryRequest);
            setCustodyAction('');
            setCustodyDescription('');
            onUpdated(updated);
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setAddingEntry(false);
        }
    };

    const handleTag = async (tagId: string) => {
        try {
            await TagEvidence({ evidence_item_id: item.evidence_item_id, tag_id: tagId } as services.TagEvidenceRequest);
            // Update tags in place
            const newTag = { tag_id: tagId, name: '', color: '' };
            onUpdated({ ...item, tags: [...tags, newTag] });
        } catch { /* ignore */ }
    };

    const handleUntag = async (tagId: string) => {
        try {
            await UntagEvidence({ evidence_item_id: item.evidence_item_id, tag_id: tagId } as services.TagEvidenceRequest);
            onUpdated({ ...item, tags: tags.filter((t) => t.tag_id !== tagId) });
        } catch { /* ignore */ }
    };

    return (
        <div className="space-y-6">
            {/* Back button */}
            <button
                onClick={onBack}
                className="text-sm text-gray-400 hover:text-gray-200 inline-flex items-center gap-1 transition-colors"
            >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
                </svg>
                Back to Evidence List
            </button>

            <ErrorMessage message={error} onDismiss={() => setError('')} />

            {/* Metadata */}
            <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                <h3 className="text-lg font-semibold text-gray-100 mb-4">{item.name}</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                    <div>
                        <span className="text-gray-400">Type</span>
                        <div className="mt-1">
                            <span className={`text-xs px-2 py-0.5 rounded border ${typeColor}`}>
                                {item.evidence_type}
                            </span>
                        </div>
                    </div>
                    <div>
                        <span className="text-gray-400">Status</span>
                        <div className="mt-1">
                            <span className={`text-xs px-2 py-0.5 rounded border ${statusColor}`}>
                                {item.status}
                            </span>
                        </div>
                    </div>
                    <div>
                        <span className="text-gray-400">Content Hash</span>
                        <p className="text-gray-200 font-mono text-xs mt-1 break-all">
                            {item.content_hash || 'N/A'}
                        </p>
                    </div>
                    <div>
                        <span className="text-gray-400">Collected By</span>
                        <p className="text-gray-200 mt-1">{item.collected_by}</p>
                    </div>
                    <div>
                        <span className="text-gray-400">Collected At</span>
                        <p className="text-gray-200 mt-1">
                            {new Date(item.collected_at).toLocaleString()}
                        </p>
                    </div>
                    <div>
                        <span className="text-gray-400">Created At</span>
                        <p className="text-gray-200 mt-1">
                            {new Date(item.created_at).toLocaleString()}
                        </p>
                    </div>
                    {item.description && (
                        <div className="md:col-span-2">
                            <span className="text-gray-400">Description</span>
                            <p className="text-gray-200 mt-1">{item.description}</p>
                        </div>
                    )}
                    {/* Tags */}
                    <div className="md:col-span-2">
                        <span className="text-gray-400">Tags</span>
                        <div className="flex items-center gap-1.5 flex-wrap mt-1">
                            {tags.map((tag) => (
                                <TagBadge key={tag.tag_id} tag={tag} onRemove={() => handleUntag(tag.tag_id)} />
                            ))}
                            <TagSelector currentTags={tags} onTag={handleTag} onUntag={handleUntag} />
                        </div>
                    </div>
                </div>
            </div>

            {/* Status Update */}
            <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                <h4 className="text-sm font-medium text-gray-300 mb-3">Update Status</h4>
                {isWithdrawn ? (
                    <p className="text-sm text-red-400">
                        This evidence has been withdrawn. No further status changes are allowed.
                    </p>
                ) : (
                    <div className="flex items-center gap-3">
                        <select
                            value={newStatus}
                            onChange={(e) => setNewStatus(e.target.value)}
                            className="bg-gray-900 border border-gray-700 rounded px-3 py-1.5 text-sm text-gray-100 focus:outline-none focus:border-blue-500"
                        >
                            {STATUSES.map((s) => (
                                <option key={s} value={s}>{s}</option>
                            ))}
                        </select>
                        <button
                            onClick={handleStatusUpdate}
                            disabled={updatingStatus || newStatus === item.status}
                            className="bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white py-1.5 px-4 rounded text-sm transition-colors"
                        >
                            {updatingStatus ? 'Updating...' : 'Update'}
                        </button>
                        {newStatus === 'WITHDRAWN' && (
                            <span className="text-xs text-red-400">Warning: withdrawal is permanent</span>
                        )}
                    </div>
                )}
            </div>

            {/* Custody Chain */}
            <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                <h4 className="text-sm font-medium text-gray-300 mb-3">
                    Chain of Custody ({item.custody_log?.length || 0})
                </h4>
                {item.custody_log && item.custody_log.length > 0 ? (
                    <div className="space-y-3">
                        {item.custody_log.map((entry, idx) => (
                            <div key={idx} className="flex gap-3 text-sm">
                                <div className="flex flex-col items-center">
                                    <div className="w-2 h-2 rounded-full bg-blue-400 mt-1.5"></div>
                                    {idx < item.custody_log.length - 1 && (
                                        <div className="w-px flex-1 bg-gray-700 mt-1"></div>
                                    )}
                                </div>
                                <div className="pb-3 flex-1 min-w-0">
                                    <div className="flex items-center gap-2 flex-wrap">
                                        <span className="text-gray-200 font-medium">{entry.action}</span>
                                        <span className="text-gray-500">by {entry.handler}</span>
                                    </div>
                                    {entry.description && (
                                        <p className="text-gray-400 text-xs mt-0.5">{entry.description}</p>
                                    )}
                                    <p className="text-gray-500 text-xs mt-0.5">
                                        {new Date(entry.timestamp).toLocaleString()}
                                    </p>
                                </div>
                            </div>
                        ))}
                    </div>
                ) : (
                    <p className="text-gray-500 text-sm">No custody entries yet.</p>
                )}
            </div>

            {/* Add Custody Entry */}
            <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                <h4 className="text-sm font-medium text-gray-300 mb-3">Add Custody Entry</h4>
                <div className="space-y-3">
                    <input
                        type="text"
                        value={custodyAction}
                        onChange={(e) => setCustodyAction(e.target.value)}
                        placeholder="Action (e.g., TRANSFERRED, EXAMINED, COPIED)"
                        className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded text-gray-100 text-sm focus:outline-none focus:border-blue-500 placeholder-gray-600"
                    />
                    <textarea
                        value={custodyDescription}
                        onChange={(e) => setCustodyDescription(e.target.value)}
                        placeholder="Description (optional)"
                        rows={2}
                        className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded text-gray-100 text-sm resize-y focus:outline-none focus:border-blue-500 placeholder-gray-600"
                    />
                    <div className="flex justify-end">
                        <button
                            onClick={handleAddCustodyEntry}
                            disabled={addingEntry || !custodyAction.trim()}
                            className="bg-green-600 hover:bg-green-700 disabled:bg-gray-700 disabled:text-gray-500 text-white py-1.5 px-4 rounded text-sm transition-colors"
                        >
                            {addingEntry ? 'Adding...' : 'Add Entry'}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}
