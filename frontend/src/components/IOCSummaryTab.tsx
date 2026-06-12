import { useCallback, useEffect, useState, useMemo } from 'react';
import {
    GetCaseIOCs, UpdateIOCStatus, UpdateIOCType,
    PromoteIOCToFact, RestorePromotedIOC, GetCaseFacts, GetFactTypes,
} from '../../wailsjs/go/main/App';
import { services, models } from '../../wailsjs/go/models';
import type { IOCEntry, IOCStatus, IOCType } from '../utils/iocTypes';
import { IOC_PATTERNS } from '../utils/iocPatterns';
import { defang } from '../utils/defang';

interface IOCSummaryTabProps {
    caseId: string;
    evidenceItems: services.EvidenceResponse[];
    onNavigate: (tab: string, blockId?: string) => void;
    onIocStatusChange: () => void;
    evidencePrefix: string;
    evidenceSeqDigits: number;
}

type SortKey = 'type' | 'value' | 'status' | 'created_at';
type SortDir = 'asc' | 'desc';

const STATUS_ORDER: Record<IOCStatus, number> = {
    confirmed: 0, detected: 1, false_positive: 2, promoted: 3,
};

const TYPE_COLORS: Record<IOCType, string> = {
    ipv4: 'bg-blue-900 text-blue-300',
    ipv6: 'bg-blue-900 text-blue-300',
    domain: 'bg-purple-900 text-purple-300',
    url: 'bg-violet-900 text-violet-300',
    email: 'bg-cyan-900 text-cyan-300',
    md5: 'bg-amber-900 text-amber-300',
    sha1: 'bg-amber-900 text-amber-300',
    sha256: 'bg-amber-900 text-amber-300',
    file_path: 'bg-gray-700 text-gray-300',
    file: 'bg-gray-700 text-gray-300',
    registry_key: 'bg-gray-700 text-gray-300',
    cve: 'bg-red-900 text-red-300',
};

const STATUS_BADGE: Record<IOCStatus, string> = {
    detected: 'bg-yellow-900 text-yellow-300',
    confirmed: 'bg-red-900 text-red-300',
    false_positive: 'bg-gray-700 text-gray-400 line-through',
    promoted: 'bg-green-900/40 text-green-300',
};

const STATUS_LABELS: Record<IOCStatus, string> = {
    detected: 'Detected',
    confirmed: 'Confirmed',
    false_positive: 'False Positive',
    promoted: 'Promoted',
};

const FACT_TYPES = [
    'username', 'hostname', 'ip_address', 'mac_address', 'os_version', 'timezone',
    'email_address', 'account_sid', 'full_name', 'phone_number', 'device_serial',
    'url', 'file_path', 'domain', 'registry_key', 'custom',
];

function formatFactType(t: string): string {
    const special: Record<string, string> = {
        ip_address: 'IP Address', mac_address: 'MAC Address',
        os_version: 'OS Version', account_sid: 'Account SID', url: 'URL',
    };
    return special[t] ?? t.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
}

function iocTypeToFactType(iocType: IOCType): string {
    const map: Partial<Record<IOCType, string>> = {
        ipv4: 'ip_address', ipv6: 'ip_address',
        domain: 'domain', url: 'url',
        email: 'email_address',
        file_path: 'file_path', file: 'file_path',
        registry_key: 'registry_key',
    };
    return map[iocType] ?? 'custom';
}

const INPUT_CLASS = 'w-full px-3 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-100 focus:outline-none focus:border-blue-500 placeholder-gray-600';
const LABEL_CLASS = 'block text-xs text-gray-400 mb-1';

interface PromotedFactEntry { factId: string; promotedAt: string; }

export default function IOCSummaryTab({ caseId, evidenceItems, onNavigate, onIocStatusChange, evidencePrefix, evidenceSeqDigits }: IOCSummaryTabProps) {
    const [iocs, setIocs] = useState<IOCEntry[]>([]);
    const [showFPs, setShowFPs] = useState(false);
    const [typeFilter, setTypeFilter] = useState<IOCType[]>([]);
    const [evidenceFilter, setEvidenceFilter] = useState<string>('');
    const [search, setSearch] = useState('');
    const [sortKey, setSortKey] = useState<SortKey>('status');
    const [sortDir, setSortDir] = useState<SortDir>('asc');

    // Promote state
    const [factTypes, setFactTypes] = useState<string[]>(FACT_TYPES);
    const [promotedFactsMap, setPromotedFactsMap] = useState<Map<string, PromotedFactEntry>>(new Map());
    const [showPromoteModal, setShowPromoteModal] = useState(false);
    const [promoteIoc, setPromoteIoc] = useState<IOCEntry | null>(null);
    const [promoteType, setPromoteType] = useState('');
    const [promoteLabel, setPromoteLabel] = useState('');
    const [promoteValue, setPromoteValue] = useState('');
    const [promoteEvidenceId, setPromoteEvidenceId] = useState('');
    const [promoteNotes, setPromoteNotes] = useState('');
    const [promoteError, setPromoteError] = useState('');
    const [promoting, setPromoting] = useState(false);

    // Restore state
    const [restoreConfirmId, setRestoreConfirmId] = useState<string | null>(null);
    const [restoring, setRestoring] = useState(false);

    // Fetch fact types once
    useEffect(() => {
        GetFactTypes()
            .then(types => { if (types?.length) setFactTypes(types); })
            .catch(() => {});
    }, []);

    const fetchAll = useCallback(() => {
        Promise.all([
            GetCaseIOCs(caseId, showFPs),
            GetCaseFacts(caseId),
        ]).then(([iocResult, factsResult]) => {
            setIocs((iocResult as IOCEntry[]) || []);
            const m = new Map<string, PromotedFactEntry>();
            (factsResult || []).forEach(f => {
                if (f.sourceIocId) m.set(f.sourceIocId, { factId: f.factId, promotedAt: f.createdAt });
            });
            setPromotedFactsMap(m);
        }).catch(() => {});
    }, [caseId, showFPs]);

    useEffect(() => {
        fetchAll();
    }, [fetchAll]);

    const evidenceMap = useMemo(() => {
        const m = new Map<string, string>();
        const sorted = [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at));
        sorted.forEach((item, idx) => {
            m.set(item.evidence_item_id, `${evidencePrefix}${String(idx + 1).padStart(evidenceSeqDigits, '0')}`);
        });
        return m;
    }, [evidenceItems, evidencePrefix, evidenceSeqDigits]);

    const sortedEvidenceItems = useMemo(
        () => [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at)),
        [evidenceItems],
    );

    const allTypes = useMemo(() => Array.from(new Set(iocs.map((i) => i.type))).sort(), [iocs]);

    const filtered = useMemo(() => {
        let result = iocs.filter(i => i.status !== 'promoted');
        if (typeFilter.length > 0) {
            result = result.filter((i) => typeFilter.includes(i.type));
        }
        if (evidenceFilter) {
            result = result.filter((i) => i.evidence_item_id === evidenceFilter);
        }
        if (search) {
            const q = search.toLowerCase();
            result = result.filter((i) => i.value.toLowerCase().includes(q));
        }
        return result;
    }, [iocs, typeFilter, evidenceFilter, search]);

    const sorted = useMemo(() => {
        return [...filtered].sort((a, b) => {
            let cmp = 0;
            if (sortKey === 'status') {
                cmp = STATUS_ORDER[a.status] - STATUS_ORDER[b.status];
                if (cmp === 0) cmp = b.created_at.localeCompare(a.created_at);
            } else if (sortKey === 'type') {
                cmp = a.type.localeCompare(b.type);
            } else if (sortKey === 'value') {
                cmp = a.value.localeCompare(b.value);
            } else if (sortKey === 'created_at') {
                cmp = a.created_at.localeCompare(b.created_at);
            }
            return sortDir === 'asc' ? cmp : -cmp;
        });
    }, [filtered, sortKey, sortDir]);

    const promotedIOCs = useMemo(() => iocs.filter(i => i.status === 'promoted'), [iocs]);

    const handleSort = (key: SortKey) => {
        if (sortKey === key) {
            setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
        } else {
            setSortKey(key);
            setSortDir('asc');
        }
    };

    const handleTypeChange = async (iocId: string, newType: IOCType) => {
        const prev_type = iocs.find((i) => i.ioc_id === iocId)?.type;
        setIocs((prev) =>
            prev.map((i) => (i.ioc_id === iocId ? { ...i, type: newType, status: 'detected' } : i))
        );
        try {
            await UpdateIOCType(iocId, newType);
            onIocStatusChange();
        } catch {
            setIocs((prev) =>
                prev.map((i) => (i.ioc_id === iocId ? { ...i, type: prev_type ?? i.type } : i))
            );
        }
    };

    const handleStatusChange = async (iocId: string, newStatus: IOCStatus) => {
        try {
            await UpdateIOCStatus(iocId, newStatus);
            setIocs((prev) =>
                prev.map((i) => (i.ioc_id === iocId ? { ...i, status: newStatus } : i))
            );
            onIocStatusChange();
        } catch { /* ignore */ }
    };

    const toggleTypeFilter = (type: IOCType) => {
        setTypeFilter((prev) =>
            prev.includes(type) ? prev.filter((t) => t !== type) : [...prev, type]
        );
    };

    const openPromoteModal = (ioc: IOCEntry) => {
        setPromoteIoc(ioc);
        setPromoteType(iocTypeToFactType(ioc.type));
        setPromoteLabel('');
        setPromoteValue(ioc.value);
        setPromoteEvidenceId(ioc.evidence_item_id ?? '');
        setPromoteNotes('');
        setPromoteError('');
        setShowPromoteModal(true);
    };

    const handlePromote = async () => {
        if (!promoteIoc) return;
        setPromoteError('');
        if (!promoteLabel.trim()) { setPromoteError('Description is required.'); return; }
        if (!promoteValue.trim()) { setPromoteError('Value is required.'); return; }
        setPromoting(true);
        try {
            await PromoteIOCToFact(promoteIoc.ioc_id, {
                caseId,
                type: promoteType,
                label: promoteLabel.trim(),
                value: promoteValue.trim(),
                evidenceItemId: promoteEvidenceId || undefined,
                notes: promoteNotes.trim(),
            } as models.CreateCaseFactRequest);
            setShowPromoteModal(false);
            onIocStatusChange();
            fetchAll();
        } catch (err: unknown) {
            setPromoteError(String(err));
        } finally {
            setPromoting(false);
        }
    };

    const handleRestore = async (iocId: string) => {
        const entry = promotedFactsMap.get(iocId);
        if (!entry) return;
        setRestoring(true);
        try {
            await RestorePromotedIOC(iocId, entry.factId);
            setRestoreConfirmId(null);
            onIocStatusChange();
            fetchAll();
        } catch { /* ignore */ } finally {
            setRestoring(false);
        }
    };

    const SortHeader = ({ label, k }: { label: string; k: SortKey }) => (
        <button
            onClick={() => handleSort(k)}
            className="flex items-center gap-1 text-left font-medium text-gray-400 hover:text-gray-200 transition-colors"
        >
            {label}
            {sortKey === k && (
                <span className="text-xs">{sortDir === 'asc' ? '↑' : '↓'}</span>
            )}
        </button>
    );

    return (
        <div className="space-y-4">
            {/* Controls */}
            <div className="flex flex-wrap items-center gap-3">
                <input
                    type="text"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    placeholder="Search IOC value..."
                    className="px-3 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-100 focus:outline-none focus:border-blue-500 placeholder-gray-600 w-52"
                />
                <select
                    value={evidenceFilter}
                    onChange={(e) => setEvidenceFilter(e.target.value)}
                    className="px-3 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-100 focus:outline-none focus:border-blue-500"
                >
                    <option value="">All Sources</option>
                    {sortedEvidenceItems.map((item, idx) => (
                        <option key={item.evidence_item_id} value={item.evidence_item_id}>
                            {evidencePrefix}{String(idx + 1).padStart(evidenceSeqDigits, '0')} - {item.name}
                        </option>
                    ))}
                </select>
                <label className="flex items-center gap-2 text-sm text-gray-400 cursor-pointer select-none">
                    <input
                        type="checkbox"
                        checked={showFPs}
                        onChange={(e) => setShowFPs(e.target.checked)}
                        className="rounded border-gray-600"
                    />
                    Show False Positives
                </label>
            </div>

            {/* Type filter pills */}
            {allTypes.length > 0 && (
                <div className="flex flex-wrap gap-2">
                    {allTypes.map((type) => {
                        const label = IOC_PATTERNS.find((p) => p.type === type)?.label ?? type;
                        const active = typeFilter.includes(type);
                        return (
                            <button
                                key={type}
                                onClick={() => toggleTypeFilter(type)}
                                className={`px-2 py-0.5 rounded text-xs font-medium transition-colors ${
                                    active ? TYPE_COLORS[type] + ' ring-1 ring-white/20' : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
                                }`}
                            >
                                {label}
                            </button>
                        );
                    })}
                </div>
            )}

            {/* Main IOC Table */}
            <div className="overflow-x-auto">
                <table className="w-full text-sm">
                    <thead>
                        <tr className="text-left border-b border-gray-700">
                            <th className="pb-2 pr-4"><SortHeader label="Type" k="type" /></th>
                            <th className="pb-2 pr-4"><SortHeader label="Value (defanged)" k="value" /></th>
                            <th className="pb-2 pr-4"><SortHeader label="Status" k="status" /></th>
                            <th className="pb-2 pr-4 text-gray-400 font-medium">Source</th>
                            <th className="pb-2 pr-4"><SortHeader label="Detected At" k="created_at" /></th>
                            <th className="pb-2 pr-4 font-medium text-gray-400">Actions</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-800">
                        {sorted.length === 0 && (
                            <tr>
                                <td colSpan={6} className="py-8 text-center text-gray-500">
                                    No IOCs found.
                                </td>
                            </tr>
                        )}
                        {sorted.map((ioc) => {
                            const source = ioc.evidence_item_id
                                ? (evidenceMap.get(ioc.evidence_item_id) ?? ioc.evidence_item_id)
                                : 'Master Notes';
                            return (
                                <tr key={ioc.ioc_id} className="hover:bg-gray-800">
                                    <td className="py-2 pr-4">
                                        <select
                                            value={ioc.type}
                                            onChange={(e) => handleTypeChange(ioc.ioc_id, e.target.value as IOCType)}
                                            className={`rounded text-xs font-medium px-1.5 py-0.5 border-0 focus:outline-none focus:ring-1 focus:ring-blue-500 cursor-pointer ${TYPE_COLORS[ioc.type]}`}
                                        >
                                            {IOC_PATTERNS.map((p) => (
                                                <option key={p.type} value={p.type}>{p.label}</option>
                                            ))}
                                        </select>
                                    </td>
                                    <td className="py-2 pr-4 font-mono text-xs text-gray-200 max-w-xs truncate" title={defang(ioc.value, ioc.type)}>
                                        {defang(ioc.value, ioc.type)}
                                    </td>
                                    <td className="py-2 pr-4">
                                        <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${STATUS_BADGE[ioc.status]}`}>
                                            {STATUS_LABELS[ioc.status]}
                                        </span>
                                    </td>
                                    <td className="py-2 pr-4">
                                        <button
                                            onClick={() => onNavigate(
                                                ioc.evidence_item_id
                                                    ? `evidence-notes-${ioc.evidence_item_id}`
                                                    : 'notes',
                                                ioc.block_id,
                                            )}
                                            className="text-xs text-blue-400 hover:text-blue-300 underline transition-colors"
                                        >
                                            {source}
                                        </button>
                                    </td>
                                    <td className="py-2 pr-4 text-xs text-gray-500 font-mono whitespace-nowrap">
                                        {new Date(ioc.created_at).toLocaleString()}
                                    </td>
                                    <td className="py-2 pr-4 text-xs whitespace-nowrap">
                                        {ioc.status === 'detected' && (
                                            <>
                                                <button onClick={() => handleStatusChange(ioc.ioc_id, 'confirmed')} className="text-xs text-gray-500 hover:text-gray-200 transition-colors">Confirm</button>
                                                <span className="text-gray-700 mx-1">/</span>
                                                <button onClick={() => handleStatusChange(ioc.ioc_id, 'false_positive')} className="text-xs text-gray-500 hover:text-gray-200 transition-colors">FP</button>
                                                <span className="text-gray-700 mx-1">/</span>
                                                <button onClick={() => openPromoteModal(ioc)} className="text-xs text-gray-500 hover:text-green-400 transition-colors">Promote</button>
                                            </>
                                        )}
                                        {ioc.status === 'confirmed' && (
                                            <>
                                                <button onClick={() => handleStatusChange(ioc.ioc_id, 'detected')} className="text-xs text-gray-500 hover:text-gray-200 transition-colors">Unconfirm</button>
                                                <span className="text-gray-700 mx-1">/</span>
                                                <button onClick={() => handleStatusChange(ioc.ioc_id, 'false_positive')} className="text-xs text-gray-500 hover:text-gray-200 transition-colors">FP</button>
                                                <span className="text-gray-700 mx-1">/</span>
                                                <button onClick={() => openPromoteModal(ioc)} className="text-xs text-gray-500 hover:text-green-400 transition-colors">Promote</button>
                                            </>
                                        )}
                                        {ioc.status === 'false_positive' && (
                                            <>
                                                <button onClick={() => handleStatusChange(ioc.ioc_id, 'confirmed')} className="text-xs text-gray-500 hover:text-gray-200 transition-colors">Confirm</button>
                                                <span className="text-gray-700 mx-1">/</span>
                                                <button onClick={() => handleStatusChange(ioc.ioc_id, 'detected')} className="text-xs text-gray-500 hover:text-gray-200 transition-colors">Restore</button>
                                            </>
                                        )}
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>

            {/* Promoted to Case Facts section */}
            {promotedIOCs.length > 0 && (
                <div className="mt-2">
                    <div className="flex items-center gap-2 mb-3">
                        <span className="text-sm font-medium text-gray-300">Promoted to Case Facts</span>
                        <span className="px-1.5 py-0.5 rounded text-xs font-medium bg-green-900/40 text-green-300 border border-green-700/50">
                            {promotedIOCs.length}
                        </span>
                        <div className="flex-1 border-t border-gray-700" />
                    </div>
                    <div className="overflow-x-auto">
                        <table className="w-full text-sm">
                            <thead>
                                <tr className="text-left border-b border-gray-700">
                                    <th className="pb-2 pr-4 font-medium text-gray-400">IOC Type</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Value (defanged)</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Evidence Item</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Promoted At</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-800">
                                {promotedIOCs.map((ioc) => {
                                    const factEntry = promotedFactsMap.get(ioc.ioc_id);
                                    const sourceLabel = ioc.evidence_item_id
                                        ? (evidenceMap.get(ioc.evidence_item_id) ?? ioc.evidence_item_id)
                                        : 'Case Level';

                                    if (restoreConfirmId === ioc.ioc_id) {
                                        return (
                                            <tr key={ioc.ioc_id} className="bg-amber-900/20">
                                                <td colSpan={5} className="py-3 px-2">
                                                    <div className="flex items-center gap-3">
                                                        <span className="text-sm text-amber-300">
                                                            Restore this IOC to detected status? The associated case fact will be deleted.
                                                        </span>
                                                        <button
                                                            onClick={() => handleRestore(ioc.ioc_id)}
                                                            disabled={restoring}
                                                            className="text-xs bg-amber-700 hover:bg-amber-600 disabled:bg-gray-700 text-white px-3 py-1 rounded transition-colors"
                                                        >
                                                            {restoring ? 'Restoring...' : 'Restore'}
                                                        </button>
                                                        <button
                                                            onClick={() => setRestoreConfirmId(null)}
                                                            className="text-xs text-gray-400 hover:text-gray-200 px-3 py-1 rounded border border-gray-600 transition-colors"
                                                        >
                                                            Cancel
                                                        </button>
                                                    </div>
                                                </td>
                                            </tr>
                                        );
                                    }

                                    return (
                                        <tr key={ioc.ioc_id} className="hover:bg-gray-800">
                                            <td className="py-2 pr-4">
                                                <span className={`rounded text-xs font-medium px-1.5 py-0.5 ${TYPE_COLORS[ioc.type]}`}>
                                                    {IOC_PATTERNS.find(p => p.type === ioc.type)?.label ?? ioc.type}
                                                </span>
                                            </td>
                                            <td className="py-2 pr-4 font-mono text-xs text-gray-200 max-w-xs truncate" title={defang(ioc.value, ioc.type)}>
                                                {defang(ioc.value, ioc.type)}
                                            </td>
                                            <td className="py-2 pr-4 text-xs text-gray-400">{sourceLabel}</td>
                                            <td className="py-2 pr-4 text-xs text-gray-500 font-mono whitespace-nowrap">
                                                {factEntry ? new Date(factEntry.promotedAt).toLocaleString() : ''}
                                            </td>
                                            <td className="py-2 pr-4 text-xs">
                                                <button
                                                    onClick={() => setRestoreConfirmId(ioc.ioc_id)}
                                                    className="text-xs text-gray-500 hover:text-amber-400 transition-colors"
                                                >
                                                    Restore
                                                </button>
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}

            {/* Promote Modal */}
            {showPromoteModal && promoteIoc && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
                    <div className="bg-gray-800 border border-gray-700 rounded-lg shadow-xl w-full max-w-md p-4 space-y-3">
                        <div className="flex items-center justify-between">
                            <h3 className="text-sm font-semibold text-gray-200">Promote to Case Facts</h3>
                            <button onClick={() => setShowPromoteModal(false)} className="text-gray-500 hover:text-gray-300">
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                                </svg>
                            </button>
                        </div>
                        <div className="grid grid-cols-2 gap-3">
                            <div>
                                <label className={LABEL_CLASS}>Type *</label>
                                <select value={promoteType} onChange={(e) => setPromoteType(e.target.value)} className={INPUT_CLASS}>
                                    {factTypes.map(t => <option key={t} value={t}>{formatFactType(t)}</option>)}
                                </select>
                            </div>
                            <div>
                                <label className={LABEL_CLASS}>Evidence Item</label>
                                <select value={promoteEvidenceId} onChange={(e) => setPromoteEvidenceId(e.target.value)} className={INPUT_CLASS}>
                                    <option value="">Case Level</option>
                                    {sortedEvidenceItems.map((item, idx) => (
                                        <option key={item.evidence_item_id} value={item.evidence_item_id}>
                                            {evidencePrefix}{String(idx + 1).padStart(evidenceSeqDigits, '0')} - {item.name}
                                        </option>
                                    ))}
                                </select>
                            </div>
                        </div>
                        <div>
                            <label className={LABEL_CLASS}>Description *</label>
                            <input
                                type="text"
                                value={promoteLabel}
                                onChange={(e) => setPromoteLabel(e.target.value)}
                                placeholder="e.g. Suspect workstation IP"
                                className={INPUT_CLASS}
                                autoFocus
                            />
                        </div>
                        <div>
                            <label className={LABEL_CLASS}>Value *</label>
                            <input
                                type="text"
                                value={promoteValue}
                                onChange={(e) => setPromoteValue(e.target.value)}
                                className={INPUT_CLASS}
                            />
                        </div>
                        <div>
                            <label className={LABEL_CLASS}>Notes</label>
                            <textarea
                                rows={2}
                                value={promoteNotes}
                                onChange={(e) => setPromoteNotes(e.target.value)}
                                placeholder="Optional notes"
                                className={INPUT_CLASS + ' resize-none'}
                            />
                        </div>
                        {promoteError && <p className="text-xs text-red-400">{promoteError}</p>}
                        <div className="flex gap-2">
                            <button
                                onClick={handlePromote}
                                disabled={promoting}
                                className="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded transition-colors"
                            >
                                {promoting ? 'Promoting...' : 'Promote to Case Facts'}
                            </button>
                            <button
                                onClick={() => setShowPromoteModal(false)}
                                className="px-4 py-1.5 text-sm border border-gray-600 hover:border-gray-400 text-gray-400 hover:text-gray-200 rounded transition-colors"
                            >
                                Cancel
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
