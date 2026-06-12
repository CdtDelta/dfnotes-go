import { useEffect, useRef, useState, useCallback, useMemo } from 'react';
import { TagBlock, UntagBlock, GetBlockIOCs, GetAttachment, GetLinkedTasks, LinkNoteToTask, ListTasks, PromoteIOCToFact, GetFactTypes } from '../../wailsjs/go/main/App';
import { services, models } from '../../wailsjs/go/models';
import TagBadge from './TagBadge';
import TagSelector from './TagSelector';
import IOCContextMenu from './IOCContextMenu';
import { renderMarkdown } from '../utils/markdownToHtml';
import { applyIOCHighlightsToHtml } from '../utils/highlightIOCs';
import type { IOCEntry, IOCStatus, IOCType } from '../utils/iocTypes';

interface NoteBlockCardProps {
    block: services.NoteBlockResponse;
    caseId: string;
    evidenceItems?: services.EvidenceResponse[];
    onEvidenceClick?: (evidenceItemId: string) => void;
    onTagsChanged?: () => void;
    onNavigateToTask?: (taskId: string) => void;
    iocVersion: number;
    onIocStatusChange: () => void;
}

interface ContextMenuState {
    x: number;
    y: number;
    iocId: string;
    iocType: IOCType;
    iocValue: string;
    iocStatus: IOCStatus;
    evidenceItemId?: string;
}

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

const MODAL_INPUT = 'w-full px-3 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-100 focus:outline-none focus:border-blue-500 placeholder-gray-600';
const MODAL_LABEL = 'block text-xs text-gray-400 mb-1';

export default function NoteBlockCard({ block, caseId, evidenceItems, onEvidenceClick, onTagsChanged, onNavigateToTask, iocVersion, onIocStatusChange }: NoteBlockCardProps) {
    const createdDate = new Date(block.created_at).toLocaleString();
    const shortHash = block.content_hash.substring(0, 12);
    const shortPrev = block.prev_hash === 'genesis'
        ? 'genesis'
        : block.prev_hash.substring(0, 12);
    const tags = block.tags || [];

    const containerRef = useRef<HTMLDivElement>(null);
    const [iocs, setIocs] = useState<IOCEntry[]>([]);
    const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
    const [linkedTasks, setLinkedTasks] = useState<models.Task[]>([]);
    const [showTaskPicker, setShowTaskPicker] = useState(false);
    const [caseTasks, setCaseTasks] = useState<models.Task[]>([]);

    // Promote modal
    const [showPromoteModal, setShowPromoteModal] = useState(false);
    const [promoteIocId, setPromoteIocId] = useState('');
    const [promoteEvidenceId, setPromoteEvidenceId] = useState('');
    const [promoteType, setPromoteType] = useState('');
    const [promoteLabel, setPromoteLabel] = useState('');
    const [promoteValue, setPromoteValue] = useState('');
    const [promoteNotes, setPromoteNotes] = useState('');
    const [promoteError, setPromoteError] = useState('');
    const [promoting, setPromoting] = useState(false);
    const [factTypes, setFactTypes] = useState<string[]>(FACT_TYPES);
    const factTypesLoadedRef = useRef(false);

    const fetchIOCs = useCallback(() => {
        GetBlockIOCs(block.block_id)
            .then((result) => {
                const all = (result as IOCEntry[]) || [];
                // Unix file paths are detected by the backend but excluded from
                // client-side highlighting because they produce too many false
                // positives in rendered markdown. Only highlight Windows paths.
                setIocs(all.filter(
                    (ioc) => ioc.type !== 'file_path' || /^[A-Za-z]:\\/.test(ioc.value)
                ));
            })
            .catch(() => {});
    }, [block.block_id]);

    useEffect(() => {
        fetchIOCs();
    }, [iocVersion, fetchIOCs]);

    const fetchLinkedTasks = useCallback(() => {
        GetLinkedTasks(block.block_id)
            .then((result) => setLinkedTasks(result || []))
            .catch(() => {});
    }, [block.block_id]);

    useEffect(() => {
        fetchLinkedTasks();
    }, [fetchLinkedTasks]);

    const openTaskPicker = async () => {
        try {
            const tasks = await ListTasks(caseId);
            setCaseTasks(tasks || []);
            setShowTaskPicker(true);
        } catch { /* ignore */ }
    };

    const handleLinkToTask = async (taskId: string) => {
        try {
            await LinkNoteToTask(taskId, block.block_id);
            setShowTaskPicker(false);
            fetchLinkedTasks();
        } catch { /* ignore */ }
    };

    // Render markdown to HTML and apply IOC highlights in one string pass.
    // dangerouslySetInnerHTML takes React's reconciler out of the subtree entirely,
    // preventing the insertBefore crashes caused by the previous TreeWalker approach.
    const highlightedHtml = useMemo(
        () => applyIOCHighlightsToHtml(renderMarkdown(block.content, evidenceItems || []), iocs),
        [block.content, evidenceItems, iocs],
    );

    // Load attachment images that were pre-converted to <img data-attachment-id>.
    useEffect(() => {
        const container = containerRef.current;
        if (!container) return;
        const imgs = container.querySelectorAll<HTMLImageElement>('img[data-attachment-id]');
        for (const img of imgs) {
            const id = img.dataset.attachmentId!;
            GetAttachment(caseId, id)
                .then((resp) => { img.src = `data:${resp.content_type};base64,${resp.data}`; })
                .catch(() => {});
        }
    }, [highlightedHtml, caseId]);

    const handleContextMenu = (e: React.MouseEvent) => {
        const target = (e.target as HTMLElement).closest<HTMLElement>('.ioc-highlight');
        if (!target) return;
        e.preventDefault();
        const { iocId, iocType, iocValue, iocStatus } = target.dataset;
        if (!iocId || !iocType || !iocValue || !iocStatus) return;
        const fullIoc = iocs.find(i => i.ioc_id === iocId);
        setContextMenu({
            x: e.clientX,
            y: e.clientY,
            iocId,
            iocType: iocType as IOCType,
            iocValue,
            iocStatus: iocStatus as IOCStatus,
            evidenceItemId: fullIoc?.evidence_item_id,
        });
    };

    const openPromoteModal = () => {
        if (!contextMenu) return;
        if (!factTypesLoadedRef.current) {
            factTypesLoadedRef.current = true;
            GetFactTypes()
                .then(types => { if (types?.length) setFactTypes(types); })
                .catch(() => {});
        }
        setPromoteIocId(contextMenu.iocId);
        setPromoteType(iocTypeToFactType(contextMenu.iocType));
        setPromoteLabel('');
        setPromoteValue(contextMenu.iocValue);
        setPromoteEvidenceId(contextMenu.evidenceItemId ?? '');
        setPromoteNotes('');
        setPromoteError('');
        setShowPromoteModal(true);
    };

    const handlePromote = async () => {
        setPromoteError('');
        if (!promoteLabel.trim()) { setPromoteError('Description is required.'); return; }
        if (!promoteValue.trim()) { setPromoteError('Value is required.'); return; }
        setPromoting(true);
        try {
            await PromoteIOCToFact(promoteIocId, {
                caseId,
                type: promoteType,
                label: promoteLabel.trim(),
                value: promoteValue.trim(),
                evidenceItemId: promoteEvidenceId || undefined,
                notes: promoteNotes.trim(),
            } as models.CreateCaseFactRequest);
            handleStatusChanged(promoteIocId, 'promoted');
            setShowPromoteModal(false);
        } catch (err: unknown) {
            setPromoteError(String(err));
        } finally {
            setPromoting(false);
        }
    };

    // Evidence link clicks delegated here because the spans inside
    // dangerouslySetInnerHTML have no React event handlers.
    const handleClick = (e: React.MouseEvent) => {
        const span = (e.target as HTMLElement).closest<HTMLElement>('[data-evidence-id]');
        if (span?.dataset.evidenceId) {
            onEvidenceClick?.(span.dataset.evidenceId);
        }
    };

    const handleStatusChanged = (iocId: string, newStatus: IOCStatus) => {
        // Optimistic update: setIocs triggers highlightedHtml recomputation which
        // re-renders the spans with the new status class immediately.
        setIocs((prev) => prev.map((ioc) => ioc.ioc_id === iocId ? { ...ioc, status: newStatus } : ioc));
        fetchIOCs();
        onIocStatusChange();
    };

    const handleTag = async (tagId: string) => {
        try {
            await TagBlock({ block_id: block.block_id, tag_id: tagId } as services.TagBlockRequest);
            onTagsChanged?.();
        } catch { /* ignore */ }
    };

    const handleUntag = async (tagId: string) => {
        try {
            await UntagBlock({ block_id: block.block_id, tag_id: tagId } as services.TagBlockRequest);
            onTagsChanged?.();
        } catch { /* ignore */ }
    };

    return (
        <div id={block.block_id} className="bg-gray-800 border border-gray-700 rounded-lg p-4">
            <div className="flex items-center justify-between mb-3">
                <span className="text-xs text-gray-500">{createdDate}</span>
                {block.verified ? (
                    <span className="inline-flex items-center gap-1 text-xs font-medium text-green-400">
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        Verified
                    </span>
                ) : (
                    <span className="inline-flex items-center gap-1 text-xs font-medium text-red-400">
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
                        </svg>
                        Unverified
                    </span>
                )}
            </div>
            <div
                ref={containerRef}
                className="prose prose-sm max-w-none mb-3 break-words"
                onContextMenu={handleContextMenu}
                onClick={handleClick}
                dangerouslySetInnerHTML={{ __html: highlightedHtml }}
            />
            {/* Tags */}
            <div className="flex items-center gap-1.5 flex-wrap mb-2">
                {tags.map((tag) => (
                    <TagBadge key={tag.tag_id} tag={tag} onRemove={() => handleUntag(tag.tag_id)} />
                ))}
                <TagSelector currentTags={tags} onTag={handleTag} onUntag={handleUntag} />
            </div>
            {/* Linked Tasks */}
            <div className="flex items-center gap-1.5 flex-wrap mb-2">
                {linkedTasks.length > 0 && (
                    <span className="text-xs text-gray-500">
                        Linked tasks ({linkedTasks.length}):{' '}
                        {linkedTasks.map((t, i) => (
                            <button
                                key={t.task_id}
                                onClick={() => onNavigateToTask?.(t.task_id)}
                                className="text-blue-400 hover:text-blue-300 transition-colors"
                            >
                                {t.title}{i < linkedTasks.length - 1 ? ', ' : ''}
                            </button>
                        ))}
                    </span>
                )}
                <button
                    onClick={openTaskPicker}
                    className="text-xs text-gray-500 hover:text-blue-400 transition-colors border border-gray-700 hover:border-blue-700 rounded px-1.5 py-0.5"
                >
                    + Link to Task
                </button>
            </div>
            <div className="flex items-center gap-4 text-xs text-gray-600 font-mono border-t border-gray-700 pt-2">
                <span title={block.content_hash}>hash: {shortHash}...</span>
                <span title={block.prev_hash}>prev: {shortPrev}{block.prev_hash !== 'genesis' && '...'}</span>
            </div>
            {contextMenu && (
                <IOCContextMenu
                    x={contextMenu.x}
                    y={contextMenu.y}
                    iocId={contextMenu.iocId}
                    iocType={contextMenu.iocType}
                    iocValue={contextMenu.iocValue}
                    iocStatus={contextMenu.iocStatus}
                    onClose={() => setContextMenu(null)}
                    onStatusChanged={handleStatusChanged}
                    onPromoteRequested={openPromoteModal}
                />
            )}

            {showPromoteModal && (
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
                                <label className={MODAL_LABEL}>Type *</label>
                                <select value={promoteType} onChange={(e) => setPromoteType(e.target.value)} className={MODAL_INPUT}>
                                    {factTypes.map(t => <option key={t} value={t}>{formatFactType(t)}</option>)}
                                </select>
                            </div>
                            <div>
                                <label className={MODAL_LABEL}>Evidence Item</label>
                                <select value={promoteEvidenceId} onChange={(e) => setPromoteEvidenceId(e.target.value)} className={MODAL_INPUT}>
                                    <option value="">Case Level</option>
                                    {[...(evidenceItems || [])].sort((a, b) => a.created_at.localeCompare(b.created_at)).map((item) => (
                                        <option key={item.evidence_item_id} value={item.evidence_item_id}>
                                            {item.item_number} - {item.name}
                                        </option>
                                    ))}
                                </select>
                            </div>
                        </div>
                        <div>
                            <label className={MODAL_LABEL}>Description *</label>
                            <input
                                type="text"
                                value={promoteLabel}
                                onChange={(e) => setPromoteLabel(e.target.value)}
                                placeholder="e.g. Suspect workstation IP"
                                className={MODAL_INPUT}
                                autoFocus
                            />
                        </div>
                        <div>
                            <label className={MODAL_LABEL}>Value *</label>
                            <input
                                type="text"
                                value={promoteValue}
                                onChange={(e) => setPromoteValue(e.target.value)}
                                className={MODAL_INPUT}
                            />
                        </div>
                        <div>
                            <label className={MODAL_LABEL}>Notes</label>
                            <textarea
                                rows={2}
                                value={promoteNotes}
                                onChange={(e) => setPromoteNotes(e.target.value)}
                                placeholder="Optional notes"
                                className={MODAL_INPUT + ' resize-none'}
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

            {showTaskPicker && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
                    <div className="bg-gray-800 border border-gray-700 rounded-lg shadow-xl w-full max-w-md p-4 space-y-3">
                        <div className="flex items-center justify-between">
                            <h3 className="text-sm font-semibold text-gray-200">Link to Task</h3>
                            <button onClick={() => setShowTaskPicker(false)} className="text-gray-500 hover:text-gray-300">
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                                </svg>
                            </button>
                        </div>
                        {caseTasks.length === 0 ? (
                            <p className="text-sm text-gray-400 py-2">No tasks found. Add tasks in the Tasks tab first.</p>
                        ) : (
                            <div className="max-h-64 overflow-y-auto space-y-1">
                                {caseTasks.map((t) => (
                                    <button
                                        key={t.task_id}
                                        onClick={() => handleLinkToTask(t.task_id)}
                                        className="w-full text-left px-3 py-2 border border-gray-700 rounded hover:border-blue-600 hover:bg-gray-700 transition-colors"
                                    >
                                        <span className="text-sm text-gray-200">{t.title}</span>
                                        <span className="ml-2 text-xs text-gray-500">{t.status}</span>
                                    </button>
                                ))}
                            </div>
                        )}
                        <button
                            onClick={() => setShowTaskPicker(false)}
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
