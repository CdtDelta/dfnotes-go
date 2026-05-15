import { useEffect, useState, useMemo } from 'react';
import { GetCaseIOCs, UpdateIOCStatus } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';
import type { IOCEntry, IOCStatus, IOCType } from '../utils/iocTypes';
import { IOC_PATTERNS } from '../utils/iocPatterns';
import { defang } from '../utils/defang';

interface IOCSummaryTabProps {
    caseId: string;
    evidenceItems: services.EvidenceResponse[];
    onNavigate: (tab: string, blockId?: string) => void;
}

type SortKey = 'type' | 'value' | 'status' | 'created_at';
type SortDir = 'asc' | 'desc';

const STATUS_ORDER: Record<IOCStatus, number> = { confirmed: 0, detected: 1, false_positive: 2 };

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
    registry_key: 'bg-gray-700 text-gray-300',
    cve: 'bg-red-900 text-red-300',
};

const STATUS_BADGE: Record<IOCStatus, string> = {
    detected: 'bg-yellow-900 text-yellow-300',
    confirmed: 'bg-red-900 text-red-300',
    false_positive: 'bg-gray-700 text-gray-400 line-through',
};

const STATUS_LABELS: Record<IOCStatus, string> = {
    detected: 'Detected',
    confirmed: 'Confirmed',
    false_positive: 'False Positive',
};

export default function IOCSummaryTab({ caseId, evidenceItems, onNavigate }: IOCSummaryTabProps) {
    const [iocs, setIocs] = useState<IOCEntry[]>([]);
    const [showFPs, setShowFPs] = useState(false);
    const [typeFilter, setTypeFilter] = useState<IOCType[]>([]);
    const [evidenceFilter, setEvidenceFilter] = useState<string>('');
    const [search, setSearch] = useState('');
    const [sortKey, setSortKey] = useState<SortKey>('status');
    const [sortDir, setSortDir] = useState<SortDir>('asc');

    useEffect(() => {
        GetCaseIOCs(caseId, showFPs)
            .then((result) => setIocs((result as IOCEntry[]) || []))
            .catch(() => {});
    }, [caseId, showFPs]);

    const evidenceMap = useMemo(() => {
        const m = new Map<string, string>();
        const sorted = [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at));
        sorted.forEach((item, idx) => {
            m.set(item.evidence_item_id, `E${String(idx + 1).padStart(3, '0')}`);
        });
        return m;
    }, [evidenceItems]);

    const allTypes = useMemo(() => Array.from(new Set(iocs.map((i) => i.type))).sort(), [iocs]);

    const filtered = useMemo(() => {
        let result = iocs;
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

    const handleSort = (key: SortKey) => {
        if (sortKey === key) {
            setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
        } else {
            setSortKey(key);
            setSortDir('asc');
        }
    };

    const handleStatusChange = async (iocId: string, newStatus: IOCStatus) => {
        try {
            await UpdateIOCStatus(iocId, newStatus);
            setIocs((prev) =>
                prev.map((i) => (i.ioc_id === iocId ? { ...i, status: newStatus } : i))
            );
        } catch { /* ignore */ }
    };

    const toggleTypeFilter = (type: IOCType) => {
        setTypeFilter((prev) =>
            prev.includes(type) ? prev.filter((t) => t !== type) : [...prev, type]
        );
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
                    {[...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at)).map((item, idx) => (
                        <option key={item.evidence_item_id} value={item.evidence_item_id}>
                            E{String(idx + 1).padStart(3, '0')} - {item.name}
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

            {/* Table */}
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
                            const typeLabel = IOC_PATTERNS.find((p) => p.type === ioc.type)?.label ?? ioc.type;
                            const source = ioc.evidence_item_id
                                ? (evidenceMap.get(ioc.evidence_item_id) ?? ioc.evidence_item_id)
                                : 'Master Notes';
                            return (
                                <tr key={ioc.ioc_id} className="hover:bg-gray-800/50">
                                    <td className="py-2 pr-4">
                                        <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${TYPE_COLORS[ioc.type]}`}>
                                            {typeLabel}
                                        </span>
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
                                            </>
                                        )}
                                        {ioc.status === 'confirmed' && (
                                            <>
                                                <button onClick={() => handleStatusChange(ioc.ioc_id, 'detected')} className="text-xs text-gray-500 hover:text-gray-200 transition-colors">Unconfirm</button>
                                                <span className="text-gray-700 mx-1">/</span>
                                                <button onClick={() => handleStatusChange(ioc.ioc_id, 'false_positive')} className="text-xs text-gray-500 hover:text-gray-200 transition-colors">FP</button>
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
        </div>
    );
}
