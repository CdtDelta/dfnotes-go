import { useEffect, useRef, useState, useCallback, useMemo } from 'react';
import { TagBlock, UntagBlock, GetBlockIndicators, GetAttachment, GetLinkedTasks, LinkNoteToTask, ListTasks } from '../../wailsjs/go/main/App';
import ManualIOCModal from './ManualIOCModal';
import PromoteFactModal from './PromoteFactModal';
import { services, models } from '../../wailsjs/go/models';
import TagBadge from './TagBadge';
import TagSelector from './TagSelector';
import IOCContextMenu from './IOCContextMenu';
import { renderMarkdown } from '../utils/markdownToHtml';
import { applyIOCHighlightsToHtml } from '../utils/highlightIOCs';
import type { BlockIOC, IOCStatus, IOCType } from '../utils/iocTypes';

interface NoteBlockCardProps {
    block: services.NoteBlockResponse;
    caseId: string;
    evidenceItems?: services.EvidenceResponse[];
    onEvidenceClick?: (evidenceItemId: string) => void;
    onTagsChanged?: () => void;
    onNavigateToTask?: (taskId: string) => void;
    iocVersion: number;
    onIocStatusChange: () => void;
    /** Evidence item this block belongs to; the default evidence item when promoting an indicator. Absent for Master Notes. */
    blockEvidenceItemId?: string;
    /** Called when user selects "Add as Case Fact" from the context menu. */
    onAddAsCaseFact?: (value: string) => void;
}

interface ContextMenuState {
    x: number;
    y: number;
    indicatorId: string;
    iocType: IOCType;
    iocValue: string;
    iocStatus: IOCStatus;
    selectionText?: string;
}

interface PromoteTarget {
    indicatorId: string;
    iocType: IOCType;
    iocValue: string;
}

interface SelectionMenuState {
    x: number;
    y: number;
    text: string;
}

export default function NoteBlockCard({ block, caseId, evidenceItems, onEvidenceClick, onTagsChanged, onNavigateToTask, iocVersion, onIocStatusChange, blockEvidenceItemId, onAddAsCaseFact }: NoteBlockCardProps) {
    const createdDate = new Date(block.created_at).toLocaleString();
    const shortHash = block.content_hash.substring(0, 12);
    const shortPrev = block.prev_hash === 'genesis'
        ? 'genesis'
        : block.prev_hash.substring(0, 12);
    const tags = block.tags || [];

    const containerRef = useRef<HTMLDivElement>(null);
    const [iocs, setIocs] = useState<BlockIOC[]>([]);
    const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
    const [linkedTasks, setLinkedTasks] = useState<models.Task[]>([]);
    const [showTaskPicker, setShowTaskPicker] = useState(false);
    const [caseTasks, setCaseTasks] = useState<models.Task[]>([]);

    const [promoteTarget, setPromoteTarget] = useState<PromoteTarget | null>(null);

    // Selection-based context menu (no IOC highlight at cursor)
    const [selectionMenu, setSelectionMenu] = useState<SelectionMenuState | null>(null);
    const selectionMenuRef = useRef<HTMLDivElement>(null);

    // Manual IOC modal
    const [showManualIOCModal, setShowManualIOCModal] = useState(false);
    const [manualIOCValue, setManualIOCValue] = useState('');

    const fetchIOCs = useCallback(() => {
        GetBlockIndicators(block.block_id)
            .then((result) => {
                const all = (result as BlockIOC[]) || [];
                // Unix file paths are detected by the backend but excluded from
                // client-side highlighting because they produce too many false
                // positives in rendered markdown. Only highlight Windows paths.
                setIocs(all.filter(
                    (ioc) => ioc.type !== 'file_path' || /^[A-Za-z]:\\/.test(ioc.raw_value)
                ));
            })
            // Keep the current highlights; a failed refresh should not blank them.
            .catch((err) => console.error(`GetBlockIndicators failed for block ${block.block_id}:`, err));
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

    useEffect(() => {
        if (!selectionMenu) return;
        const onClick = (e: MouseEvent) => {
            if (selectionMenuRef.current && !selectionMenuRef.current.contains(e.target as Node)) {
                setSelectionMenu(null);
            }
        };
        const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setSelectionMenu(null); };
        document.addEventListener('mousedown', onClick);
        document.addEventListener('keydown', onKey);
        return () => {
            document.removeEventListener('mousedown', onClick);
            document.removeEventListener('keydown', onKey);
        };
    }, [selectionMenu]);

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
        // Capture any active text selection before doing anything else.
        const selection = window.getSelection()?.toString().trim() ?? '';
        const target = (e.target as HTMLElement).closest<HTMLElement>('.ioc-highlight');

        if (!target && !selection) return;

        e.preventDefault();
        setSelectionMenu(null);

        if (target) {
            const { indicatorId, occurrenceId, iocType, iocValue, iocStatus } = target.dataset;
            if (!indicatorId || !occurrenceId || !iocType || !iocValue || !iocStatus) {
                if (selection) setSelectionMenu({ x: e.clientX, y: e.clientY, text: selection });
                return;
            }
            setContextMenu({
                x: e.clientX,
                y: e.clientY,
                indicatorId,
                iocType: iocType as IOCType,
                iocValue,
                iocStatus: iocStatus as IOCStatus,
                // If a selection spans a markdown formatting boundary, getSelection()
                // returns correct plain text so the stored value is fine, but the
                // highlight pipeline may not match it in the rendered HTML after
                // iocVersion increments. The IOC still appears in IOC Summary. Do not
                // engineer around this.
                selectionText: selection || undefined,
            });
        } else {
            setSelectionMenu({ x: e.clientX, y: e.clientY, text: selection });
        }
    };

    // The backend derives the evidence item from the block, so none is passed.
    const openManualIOCModal = (value: string) => {
        setContextMenu(null);
        setSelectionMenu(null);
        setManualIOCValue(value);
        setShowManualIOCModal(true);
    };

    const handleAddAsCaseFact = (value: string) => {
        setContextMenu(null);
        setSelectionMenu(null);
        onAddAsCaseFact?.(value);
    };

    const openPromoteModal = () => {
        if (!contextMenu) return;
        setPromoteTarget({
            indicatorId: contextMenu.indicatorId,
            iocType: contextMenu.iocType,
            iocValue: contextMenu.iocValue,
        });
    };

    // Evidence link clicks delegated here because the spans inside
    // dangerouslySetInnerHTML have no React event handlers.
    const handleClick = (e: React.MouseEvent) => {
        const span = (e.target as HTMLElement).closest<HTMLElement>('[data-evidence-id]');
        if (span?.dataset.evidenceId) {
            onEvidenceClick?.(span.dataset.evidenceId);
        }
    };

    // Called only after the backend accepted the change; IOCContextMenu keeps
    // itself open with the error otherwise, so there is nothing to revert here.
    const handleStatusChanged = (indicatorId: string, newStatus: IOCStatus) => {
        // Optimistic update ahead of the refetch: status is per indicator, so
        // every occurrence of it in this block changes class immediately.
        setIocs((prev) => prev.map((ioc) => ioc.indicator_id === indicatorId ? { ...ioc, status: newStatus } : ioc));
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
                    indicatorId={contextMenu.indicatorId}
                    iocType={contextMenu.iocType}
                    iocValue={contextMenu.iocValue}
                    iocStatus={contextMenu.iocStatus}
                    onClose={() => setContextMenu(null)}
                    onStatusChanged={handleStatusChanged}
                    onPromoteRequested={openPromoteModal}
                    selectionText={contextMenu.selectionText}
                    onMarkAsIOC={contextMenu.selectionText
                        ? () => openManualIOCModal(contextMenu.selectionText ?? '')
                        : undefined}
                    onAddAsCaseFact={contextMenu.selectionText
                        ? () => handleAddAsCaseFact(contextMenu.selectionText ?? '')
                        : undefined}
                />
            )}

            {selectionMenu && (() => {
                const MENU_W = 220;
                const MENU_H = 110;
                const left = Math.max(0, selectionMenu.x + MENU_W > window.innerWidth ? selectionMenu.x - MENU_W : selectionMenu.x);
                const top  = Math.max(0, selectionMenu.y + MENU_H > window.innerHeight ? selectionMenu.y - MENU_H : selectionMenu.y);
                return (
                    <div
                        ref={selectionMenuRef}
                        className="fixed z-50 bg-gray-800 border border-gray-600 rounded-lg shadow-xl py-1 min-w-48"
                        style={{ left, top }}
                    >
                        <div className="px-3 py-2 border-b border-gray-700">
                            <div className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-0.5">Selection</div>
                            <div className="text-xs text-gray-300 font-mono truncate max-w-64" title={selectionMenu.text}>
                                {selectionMenu.text.length > 32 ? selectionMenu.text.slice(0, 32) + '…' : selectionMenu.text}
                            </div>
                        </div>
                        <div className="py-1">
                            <button
                                onClick={() => openManualIOCModal(selectionMenu.text)}
                                className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-gray-700 transition-colors"
                            >
                                Mark as IOC
                            </button>
                            <button
                                onClick={() => handleAddAsCaseFact(selectionMenu.text)}
                                className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-gray-700 transition-colors"
                            >
                                Add as Case Fact
                            </button>
                        </div>
                    </div>
                );
            })()}

            {promoteTarget && (
                <PromoteFactModal
                    caseId={caseId}
                    indicatorId={promoteTarget.indicatorId}
                    iocType={promoteTarget.iocType}
                    iocValue={promoteTarget.iocValue}
                    evidenceItems={evidenceItems || []}
                    defaultEvidenceItemId={blockEvidenceItemId ?? ''}
                    onClose={() => setPromoteTarget(null)}
                    onPromoted={() => {
                        const { indicatorId } = promoteTarget;
                        setPromoteTarget(null);
                        handleStatusChanged(indicatorId, 'promoted');
                    }}
                />
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

            {showManualIOCModal && (
                <ManualIOCModal
                    caseId={caseId}
                    blockId={block.block_id}
                    value={manualIOCValue}
                    onClose={() => setShowManualIOCModal(false)}
                    onSave={() => {
                        fetchIOCs();
                        onIocStatusChange();
                    }}
                />
            )}
        </div>
    );
}
