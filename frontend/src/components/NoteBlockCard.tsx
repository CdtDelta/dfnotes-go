import { useEffect, useRef, useState, useCallback, useMemo } from 'react';
import { TagBlock, UntagBlock, GetBlockIOCs, GetAttachment, GetLinkedTasks, LinkNoteToTask, ListTasks } from '../../wailsjs/go/main/App';
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
}

interface ContextMenuState {
    x: number;
    y: number;
    iocId: string;
    iocType: IOCType;
    iocValue: string;
    iocStatus: IOCStatus;
}

export default function NoteBlockCard({ block, caseId, evidenceItems, onEvidenceClick, onTagsChanged, onNavigateToTask }: NoteBlockCardProps) {
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
    }, [fetchIOCs]);

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
        setContextMenu({
            x: e.clientX,
            y: e.clientY,
            iocId,
            iocType: iocType as IOCType,
            iocValue,
            iocStatus: iocStatus as IOCStatus,
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

    const handleStatusChanged = (iocId: string, newStatus: IOCStatus) => {
        // Optimistic update: setIocs triggers highlightedHtml recomputation which
        // re-renders the spans with the new status class immediately.
        setIocs((prev) => prev.map((ioc) => ioc.ioc_id === iocId ? { ...ioc, status: newStatus } : ioc));
        fetchIOCs();
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
        </div>
    );
}
