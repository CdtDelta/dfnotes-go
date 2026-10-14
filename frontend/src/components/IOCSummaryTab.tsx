import { Fragment, useCallback, useEffect, useMemo, useState } from 'react';
import {
    GetCaseFacts, GetCaseIndicators, GetIndicatorLinkedFacts,
    RestorePromotedIndicator, UpdateIndicatorStatus, UpdateIndicatorType,
} from '../../wailsjs/go/main/App';
import { services, models } from '../../wailsjs/go/models';
import type { Indicator, IOCStatus, IOCType, Occurrence, TypeChangeResult, TypeCollision } from '../utils/iocTypes';
import { IOC_PATTERNS } from '../utils/iocPatterns';
import { defang } from '../utils/defang';
import ConfirmDialog from './ConfirmDialog';
import PromoteFactModal from './PromoteFactModal';

interface IOCSummaryTabProps {
    caseId: string;
    evidenceItems: services.EvidenceResponse[];
    onNavigate: (tab: string, blockId?: string) => void;
    onIocStatusChange: () => void;
    iocVersion: number;
}

type SortKey = 'type' | 'value' | 'status' | 'occurrences' | 'created_at';
type SortDir = 'asc' | 'desc';

// Evidence filter value for blocks with no evidence item. Evidence item IDs
// are UUIDs, so this cannot collide with one.
const MASTER_FILTER = '__master__';

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

const ACTION_BUTTON = 'text-xs text-gray-500 hover:text-gray-200 transition-colors';

/** '' means Master Notes (no evidence item). */
function placeKey(occ: Occurrence): string {
    return occ.evidence_item_id ?? '';
}

/** Distinct places in order of first appearance; occurrences arrive in chain order. */
function distinctPlaces(ind: Indicator): string[] {
    return Array.from(new Set(ind.occurrences.map(placeKey)));
}

function distinctBlockCount(ind: Indicator): number {
    return new Set(ind.occurrences.map((o) => o.block_id)).size;
}

function blockTab(occ: Occurrence): string {
    return occ.evidence_item_id ? `evidence-notes-${occ.evidence_item_id}` : 'notes';
}

function typeLabel(t: string): string {
    return IOC_PATTERNS.find((p) => p.type === t)?.label ?? t;
}

interface TypeChangeState {
    indicator: Indicator;
    newType: IOCType;
    collision: TypeCollision;
    error: string;
}

interface RestoreState {
    indicator: Indicator;
    facts: models.CaseFact[];
    error: string;
}

export default function IOCSummaryTab({ caseId, evidenceItems, onNavigate, onIocStatusChange, iocVersion }: IOCSummaryTabProps) {
    const [indicators, setIndicators] = useState<Indicator[]>([]);
    const [facts, setFacts] = useState<models.CaseFact[]>([]);
    const [error, setError] = useState('');
    const [showFPs, setShowFPs] = useState(false);
    const [typeFilter, setTypeFilter] = useState<IOCType[]>([]);
    const [evidenceFilter, setEvidenceFilter] = useState<string>('');
    const [search, setSearch] = useState('');
    const [sortKey, setSortKey] = useState<SortKey>('status');
    const [sortDir, setSortDir] = useState<SortDir>('asc');
    const [expanded, setExpanded] = useState<Set<string>>(new Set());

    const [promoteTarget, setPromoteTarget] = useState<Indicator | null>(null);
    const [typeChange, setTypeChange] = useState<TypeChangeState | null>(null);
    const [restore, setRestore] = useState<RestoreState | null>(null);

    useEffect(() => {
        let cancelled = false;
        Promise.all([GetCaseIndicators(caseId, showFPs), GetCaseFacts(caseId)])
            .then(([indResult, factResult]) => {
                if (cancelled) return;
                setIndicators((indResult as Indicator[]) || []);
                setFacts(factResult || []);
            })
            .catch((err: unknown) => {
                if (!cancelled) setError(`Could not load IOCs: ${String(err)}`);
            });
        // A slower earlier load must not overwrite a newer one.
        return () => { cancelled = true; };
    }, [caseId, showFPs, iocVersion]);

    const itemNumbers = useMemo(() => {
        const m = new Map<string, string>();
        evidenceItems.forEach((item) => m.set(item.evidence_item_id, item.item_number));
        return m;
    }, [evidenceItems]);

    const sortedEvidenceItems = useMemo(
        () => [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at)),
        [evidenceItems],
    );

    const evidenceLabel = useCallback(
        (key: string) => (key ? (itemNumbers.get(key) ?? key) : 'Master'),
        [itemNumbers],
    );

    const promotedAt = useMemo(() => {
        const m = new Map<string, string>();
        facts.forEach((f) => {
            if (!f.sourceIocId) return;
            const prev = m.get(f.sourceIocId);
            if (!prev || f.createdAt < prev) m.set(f.sourceIocId, f.createdAt);
        });
        return m;
    }, [facts]);

    const active = useMemo(() => indicators.filter((i) => i.status !== 'promoted'), [indicators]);
    const promoted = useMemo(() => indicators.filter((i) => i.status === 'promoted'), [indicators]);

    const allTypes = useMemo(() => Array.from(new Set(active.map((i) => i.type))).sort(), [active]);

    const filtered = useMemo(() => {
        let result = active;
        if (typeFilter.length > 0) {
            result = result.filter((i) => typeFilter.includes(i.type));
        }
        if (evidenceFilter) {
            const want = evidenceFilter === MASTER_FILTER ? '' : evidenceFilter;
            result = result.filter((i) => i.occurrences.some((o) => placeKey(o) === want));
        }
        if (search) {
            const q = search.toLowerCase();
            result = result.filter((i) =>
                i.value.toLowerCase().includes(q) ||
                i.occurrences.some((o) => o.raw_value.toLowerCase().includes(q)));
        }
        return result;
    }, [active, typeFilter, evidenceFilter, search]);

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
            } else if (sortKey === 'occurrences') {
                cmp = distinctBlockCount(a) - distinctBlockCount(b);
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

    const toggleTypeFilter = (type: IOCType) => {
        setTypeFilter((prev) =>
            prev.includes(type) ? prev.filter((t) => t !== type) : [...prev, type]
        );
    };

    const toggleExpanded = (id: string) => {
        setExpanded((prev) => {
            const next = new Set(prev);
            if (next.has(id)) next.delete(id); else next.add(id);
            return next;
        });
    };

    const goToOccurrence = (occ: Occurrence) => onNavigate(blockTab(occ), occ.block_id);

    const handleStatusChange = async (ind: Indicator, newStatus: IOCStatus) => {
        try {
            await UpdateIndicatorStatus(ind.indicator_id, newStatus);
            onIocStatusChange();
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    // The dropdown is not updated optimistically; it shows server state once
    // the iocVersion refetch lands.
    const handleTypeChange = async (ind: Indicator, newType: IOCType) => {
        try {
            const res = await UpdateIndicatorType(ind.indicator_id, newType, false) as TypeChangeResult;
            if (res.collision) {
                setTypeChange({ indicator: ind, newType, collision: res.collision, error: '' });
                return;
            }
            if (res.changed) onIocStatusChange();
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    const confirmMerge = async () => {
        if (!typeChange) return;
        try {
            const res = await UpdateIndicatorType(typeChange.indicator.indicator_id, typeChange.newType, true) as TypeChangeResult;
            if (!res.changed) {
                setTypeChange((s) => s && { ...s, error: 'The type was not changed.' });
                return;
            }
            setTypeChange(null);
            onIocStatusChange();
        } catch (err: unknown) {
            setTypeChange((s) => s && { ...s, error: String(err) });
        }
    };

    const openRestore = async (ind: Indicator) => {
        try {
            const linked = await GetIndicatorLinkedFacts(ind.indicator_id);
            setRestore({ indicator: ind, facts: linked || [], error: '' });
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    const confirmRestore = async () => {
        if (!restore) return;
        try {
            await RestorePromotedIndicator(restore.indicator.indicator_id);
            setRestore(null);
            onIocStatusChange();
        } catch (err: unknown) {
            setRestore((s) => s && { ...s, error: String(err) });
        }
    };

    /** The shared place when every occurrence is in one place, else Case Level. */
    const defaultPromoteEvidence = (ind: Indicator): string => {
        const places = distinctPlaces(ind);
        return places.length === 1 ? places[0] : '';
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

    const Sep = () => <span className="text-gray-700 mx-1">/</span>;

    return (
        <div className="space-y-4">
            {error && (
                <div role="alert" className="flex items-start justify-between gap-3 text-sm text-[var(--text-error)]">
                    <span>{error}</span>
                    <button
                        type="button"
                        onClick={() => setError('')}
                        aria-label="Dismiss error"
                        className="text-[var(--text-error)] hover:text-[var(--text-primary)] transition-colors"
                    >
                        &times;
                    </button>
                </div>
            )}

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
                    <option value={MASTER_FILTER}>Master Notes</option>
                    {sortedEvidenceItems.map((item) => (
                        <option key={item.evidence_item_id} value={item.evidence_item_id}>
                            {item.item_number} - {item.name}
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
                        const isActive = typeFilter.includes(type);
                        return (
                            <button
                                key={type}
                                onClick={() => toggleTypeFilter(type)}
                                className={`px-2 py-0.5 rounded text-xs font-medium transition-colors ${
                                    isActive ? TYPE_COLORS[type] + ' ring-1 ring-white/20' : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
                                }`}
                            >
                                {typeLabel(type)}
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
                            <th className="pb-2 pr-4"><SortHeader label="Occurrences" k="occurrences" /></th>
                            <th className="pb-2 pr-4 font-medium text-gray-400">Evidence</th>
                            <th className="pb-2 pr-4 font-medium text-gray-400">Detection</th>
                            <th className="pb-2 pr-4"><SortHeader label="First Seen" k="created_at" /></th>
                            <th className="pb-2 pr-4 font-medium text-gray-400">Actions</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-800">
                        {sorted.length === 0 && (
                            <tr>
                                <td colSpan={8} className="py-8 text-center text-gray-500">
                                    No IOCs found.
                                </td>
                            </tr>
                        )}
                        {sorted.map((ind) => {
                            const display = defang(ind.value, ind.type);
                            const first = ind.occurrences[0];
                            const noteCount = distinctBlockCount(ind);
                            const isExpanded = expanded.has(ind.indicator_id);
                            const methods = Array.from(new Set(ind.occurrences.map((o) => o.detection_method))).join(' + ');
                            return (
                                <Fragment key={ind.indicator_id}>
                                    <tr className="hover:bg-gray-800">
                                        <td className="py-2 pr-4">
                                            <select
                                                value={ind.type}
                                                onChange={(e) => handleTypeChange(ind, e.target.value as IOCType)}
                                                disabled={ind.status === 'promoted'}
                                                className={`rounded text-xs font-medium px-1.5 py-0.5 border-0 focus:outline-none focus:ring-1 focus:ring-blue-500 cursor-pointer disabled:cursor-not-allowed ${TYPE_COLORS[ind.type]}`}
                                            >
                                                {IOC_PATTERNS.map((p) => (
                                                    <option key={p.type} value={p.type}>{p.label}</option>
                                                ))}
                                            </select>
                                        </td>
                                        <td className="py-2 pr-4 font-mono text-xs text-gray-200 max-w-xs truncate" title={display}>
                                            {first ? (
                                                <button
                                                    type="button"
                                                    onClick={() => goToOccurrence(first)}
                                                    className="hover:underline text-left"
                                                >
                                                    {display}
                                                </button>
                                            ) : display}
                                        </td>
                                        <td className="py-2 pr-4">
                                            <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${STATUS_BADGE[ind.status]}`}>
                                                {STATUS_LABELS[ind.status]}
                                            </span>
                                        </td>
                                        <td className="py-2 pr-4 whitespace-nowrap">
                                            <button
                                                type="button"
                                                onClick={() => toggleExpanded(ind.indicator_id)}
                                                aria-expanded={isExpanded}
                                                className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                                            >
                                                {isExpanded ? '▾' : '▸'} {noteCount} {noteCount === 1 ? 'note' : 'notes'}
                                            </button>
                                        </td>
                                        <td className="py-2 pr-4 text-xs text-gray-400">
                                            {distinctPlaces(ind).map(evidenceLabel).join(', ')}
                                        </td>
                                        <td className="py-2 pr-4 text-xs text-gray-400 whitespace-nowrap">{methods}</td>
                                        <td className="py-2 pr-4 text-xs text-gray-500 font-mono whitespace-nowrap">
                                            {new Date(ind.created_at).toLocaleString()}
                                        </td>
                                        <td className="py-2 pr-4 text-xs whitespace-nowrap">
                                            {ind.status === 'detected' && (
                                                <>
                                                    <button onClick={() => handleStatusChange(ind, 'confirmed')} className={ACTION_BUTTON}>Confirm</button>
                                                    <Sep />
                                                    <button onClick={() => handleStatusChange(ind, 'false_positive')} className={ACTION_BUTTON}>FP</button>
                                                    <Sep />
                                                    <button onClick={() => setPromoteTarget(ind)} className="text-xs text-gray-500 hover:text-green-400 transition-colors">Promote</button>
                                                </>
                                            )}
                                            {ind.status === 'confirmed' && (
                                                <>
                                                    <button onClick={() => handleStatusChange(ind, 'detected')} className={ACTION_BUTTON}>Unconfirm</button>
                                                    <Sep />
                                                    <button onClick={() => handleStatusChange(ind, 'false_positive')} className={ACTION_BUTTON}>FP</button>
                                                    <Sep />
                                                    <button onClick={() => setPromoteTarget(ind)} className="text-xs text-gray-500 hover:text-green-400 transition-colors">Promote</button>
                                                </>
                                            )}
                                            {ind.status === 'false_positive' && (
                                                <>
                                                    <button onClick={() => handleStatusChange(ind, 'confirmed')} className={ACTION_BUTTON}>Confirm</button>
                                                    <Sep />
                                                    <button onClick={() => handleStatusChange(ind, 'detected')} className={ACTION_BUTTON}>Restore to Detected</button>
                                                </>
                                            )}
                                        </td>
                                    </tr>
                                    {isExpanded && (
                                        <tr>
                                            <td colSpan={8} className="pb-2 pl-8 pr-4">
                                                <ul className="space-y-1">
                                                    {ind.occurrences.map((occ) => (
                                                        <li key={occ.occurrence_id}>
                                                            <button
                                                                type="button"
                                                                onClick={() => goToOccurrence(occ)}
                                                                className="flex flex-wrap items-center gap-3 text-xs text-left text-gray-400 hover:text-gray-200 transition-colors"
                                                            >
                                                                <span className="text-blue-400 underline">
                                                                    {occ.evidence_item_id ? evidenceLabel(occ.evidence_item_id) : 'Master Notes'}
                                                                </span>
                                                                <span className="font-mono">{new Date(occ.committed_at).toLocaleString()}</span>
                                                                <span>{occ.detection_method}</span>
                                                                {occ.raw_value !== ind.value && (
                                                                    <span className="font-mono text-gray-300">{defang(occ.raw_value, ind.type)}</span>
                                                                )}
                                                            </button>
                                                        </li>
                                                    ))}
                                                </ul>
                                            </td>
                                        </tr>
                                    )}
                                </Fragment>
                            );
                        })}
                    </tbody>
                </table>
            </div>

            {/* Promoted to Case Facts section */}
            {promoted.length > 0 && (
                <div className="mt-2">
                    <div className="flex items-center gap-2 mb-3">
                        <span className="text-sm font-medium text-gray-300">Promoted to Case Facts</span>
                        <span className="px-1.5 py-0.5 rounded text-xs font-medium bg-green-900/40 text-green-300 border border-green-700/50">
                            {promoted.length}
                        </span>
                        <div className="flex-1 border-t border-gray-700" />
                    </div>
                    <div className="overflow-x-auto">
                        <table className="w-full text-sm">
                            <thead>
                                <tr className="text-left border-b border-gray-700">
                                    <th className="pb-2 pr-4 font-medium text-gray-400">IOC Type</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Value (defanged)</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Evidence</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Promoted At</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-800">
                                {promoted.map((ind) => {
                                    const display = defang(ind.value, ind.type);
                                    const at = promotedAt.get(ind.indicator_id);
                                    return (
                                        <tr key={ind.indicator_id} className="hover:bg-gray-800">
                                            <td className="py-2 pr-4">
                                                <span className={`rounded text-xs font-medium px-1.5 py-0.5 ${TYPE_COLORS[ind.type]}`}>
                                                    {typeLabel(ind.type)}
                                                </span>
                                            </td>
                                            <td className="py-2 pr-4 font-mono text-xs text-gray-200 max-w-xs truncate" title={display}>
                                                {display}
                                            </td>
                                            <td className="py-2 pr-4 text-xs text-gray-400">
                                                {distinctPlaces(ind).map(evidenceLabel).join(', ')}
                                            </td>
                                            <td className="py-2 pr-4 text-xs text-gray-500 font-mono whitespace-nowrap">
                                                {at ? new Date(at).toLocaleString() : ''}
                                            </td>
                                            <td className="py-2 pr-4 text-xs">
                                                <button
                                                    onClick={() => openRestore(ind)}
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

            {promoteTarget && (
                <PromoteFactModal
                    caseId={caseId}
                    indicatorId={promoteTarget.indicator_id}
                    iocType={promoteTarget.type}
                    iocValue={promoteTarget.value}
                    evidenceItems={evidenceItems}
                    defaultEvidenceItemId={defaultPromoteEvidence(promoteTarget)}
                    onClose={() => setPromoteTarget(null)}
                    onPromoted={() => {
                        setPromoteTarget(null);
                        onIocStatusChange();
                    }}
                />
            )}

            {typeChange && (
                <ConfirmDialog
                    title="Merge indicators?"
                    message={
                        <p>
                            {typeChange.newType}:{typeChange.indicator.value} already exists with
                            status {typeChange.collision.target_status}. Merge {typeChange.collision.occurrences_to_move} occurrence(s) into it?
                        </p>
                    }
                    confirmLabel="Merge"
                    onConfirm={confirmMerge}
                    onCancel={() => setTypeChange(null)}
                    error={typeChange.error}
                />
            )}

            {restore && (
                <ConfirmDialog
                    title="Restore indicator?"
                    destructive
                    message={
                        restore.facts.length === 0 ? (
                            <p>No linked case facts remain. The indicator returns to detected.</p>
                        ) : (
                            <>
                                <p>These linked case facts will be deleted:</p>
                                <ul className="list-disc pl-5 space-y-1">
                                    {restore.facts.map((f) => (
                                        <li key={f.factId}>
                                            {f.label} ({f.evidenceItemId ? (itemNumbers.get(f.evidenceItemId) ?? f.evidenceItemId) : 'Case Level'})
                                        </li>
                                    ))}
                                </ul>
                                <p>The indicator returns to detected.</p>
                            </>
                        )
                    }
                    confirmLabel="Restore"
                    onConfirm={confirmRestore}
                    onCancel={() => setRestore(null)}
                    error={restore.error}
                />
            )}
        </div>
    );
}
