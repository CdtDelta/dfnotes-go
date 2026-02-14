import { useState, useEffect, useCallback } from 'react';
import { AddEvidence, ListEvidence, GetEvidence } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';
import EvidenceCard from './EvidenceCard';
import EvidenceDetailView from './EvidenceDetailView';
import ErrorMessage from './ErrorMessage';

interface EvidenceTabProps {
    caseId: string;
    onEvidenceChanged?: () => void;
}

const EVIDENCE_TYPES = ['DISK', 'MEMORY', 'NETWORK', 'LOGS', 'MALWARE', 'OTHER'];

export default function EvidenceTab({ caseId, onEvidenceChanged }: EvidenceTabProps) {
    const [items, setItems] = useState<services.EvidenceResponse[]>([]);
    const [selectedItem, setSelectedItem] = useState<services.EvidenceResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    // Add form state
    const [showForm, setShowForm] = useState(false);
    const [formName, setFormName] = useState('');
    const [formDescription, setFormDescription] = useState('');
    const [formType, setFormType] = useState('DISK');
    const [formHash, setFormHash] = useState('');
    const [adding, setAdding] = useState(false);

    const fetchItems = useCallback(() => {
        setLoading(true);
        ListEvidence(caseId)
            .then((result) => setItems(result || []))
            .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
            .finally(() => setLoading(false));
    }, [caseId]);

    useEffect(() => {
        fetchItems();
    }, [fetchItems]);

    const handleAdd = async () => {
        if (!formName.trim()) return;
        setAdding(true);
        setError('');
        try {
            await AddEvidence({
                case_id: caseId,
                name: formName.trim(),
                description: formDescription.trim(),
                evidence_type: formType,
                content_hash: formHash.trim(),
            } as services.AddEvidenceRequest);
            setFormName('');
            setFormDescription('');
            setFormType('DISK');
            setFormHash('');
            setShowForm(false);
            fetchItems();
            onEvidenceChanged?.();
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setAdding(false);
        }
    };

    const handleSelect = async (evidenceItemID: string) => {
        try {
            const item = await GetEvidence(evidenceItemID);
            setSelectedItem(item);
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleBack = () => {
        setSelectedItem(null);
        fetchItems();
    };

    const handleUpdated = (updated: services.EvidenceResponse) => {
        setSelectedItem(updated);
    };

    // Detail view
    if (selectedItem) {
        return (
            <EvidenceDetailView
                item={selectedItem}
                onBack={handleBack}
                onUpdated={handleUpdated}
            />
        );
    }

    // List view
    return (
        <div className="space-y-6">
            {/* Add Evidence Toggle */}
            <div>
                <button
                    onClick={() => setShowForm(!showForm)}
                    className="text-sm text-blue-400 hover:text-blue-300 transition-colors inline-flex items-center gap-1"
                >
                    <svg
                        className={`w-4 h-4 transition-transform ${showForm ? 'rotate-45' : ''}`}
                        fill="none"
                        viewBox="0 0 24 24"
                        strokeWidth={2}
                        stroke="currentColor"
                    >
                        <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                    </svg>
                    {showForm ? 'Cancel' : 'Add Evidence'}
                </button>
            </div>

            {/* Add Form */}
            {showForm && (
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <h3 className="text-sm font-medium text-gray-300 mb-3">New Evidence Item</h3>
                    <div className="space-y-3">
                        <input
                            type="text"
                            value={formName}
                            onChange={(e) => setFormName(e.target.value)}
                            placeholder="Name (required)"
                            className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded text-gray-100 text-sm focus:outline-none focus:border-blue-500 placeholder-gray-600"
                        />
                        <textarea
                            value={formDescription}
                            onChange={(e) => setFormDescription(e.target.value)}
                            placeholder="Description"
                            rows={2}
                            className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded text-gray-100 text-sm resize-y focus:outline-none focus:border-blue-500 placeholder-gray-600"
                        />
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                            <div>
                                <label className="text-xs text-gray-400 mb-1 block">Evidence Type</label>
                                <select
                                    value={formType}
                                    onChange={(e) => setFormType(e.target.value)}
                                    className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-blue-500"
                                >
                                    {EVIDENCE_TYPES.map((t) => (
                                        <option key={t} value={t}>{t}</option>
                                    ))}
                                </select>
                            </div>
                            <div>
                                <label className="text-xs text-gray-400 mb-1 block">Content Hash</label>
                                <input
                                    type="text"
                                    value={formHash}
                                    onChange={(e) => setFormHash(e.target.value)}
                                    placeholder="SHA-256, MD5, etc."
                                    className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded text-gray-100 text-sm font-mono focus:outline-none focus:border-blue-500 placeholder-gray-600"
                                />
                            </div>
                        </div>
                        <div className="flex justify-end">
                            <button
                                onClick={handleAdd}
                                disabled={adding || !formName.trim()}
                                className="bg-green-600 hover:bg-green-700 disabled:bg-gray-700 disabled:text-gray-500 text-white py-2 px-4 rounded text-sm transition-colors"
                            >
                                {adding ? 'Adding...' : 'Add Evidence'}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            <ErrorMessage message={error} onDismiss={() => setError('')} />

            {/* Evidence List */}
            <div>
                <h3 className="text-sm font-medium text-gray-400 mb-3">
                    Evidence Items ({items.length})
                </h3>
                {loading ? (
                    <div className="flex justify-center py-8">
                        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-400"></div>
                    </div>
                ) : items.length === 0 ? (
                    <p className="text-gray-500 text-sm text-center py-8">
                        No evidence items yet. Add your first evidence above.
                    </p>
                ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                        {items.map((item) => (
                            <EvidenceCard
                                key={item.evidence_item_id}
                                item={item}
                                onSelect={handleSelect}
                            />
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
}
