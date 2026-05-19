import { useState, useEffect, useCallback, useRef } from 'react';
import {
    ListTasks, CreateTask, UpdateTaskStatus, UpdateTask, DeleteTask,
    LinkNoteToTask, UnlinkNoteFromTask, GetLinkedTasks,
    LoadTemplates, ApplyTemplate, ListNotes, ListEvidenceNotes,
} from '../../wailsjs/go/main/App';
import { models, config, services } from '../../wailsjs/go/models';
import ErrorMessage from './ErrorMessage';

interface TaskListTabProps {
    caseId: string;
    evidenceItems: services.EvidenceResponse[];
    onNavigate?: (tab: string, blockId?: string) => void;
}

type StatusFilter = 'all' | 'open' | 'in_progress' | 'blocked' | 'complete' | 'not_applicable';

const STATUS_OPTIONS: { value: string; label: string }[] = [
    { value: 'open', label: 'Open' },
    { value: 'in_progress', label: 'In Progress' },
    { value: 'blocked', label: 'Blocked' },
    { value: 'complete', label: 'Complete' },
    { value: 'not_applicable', label: 'N/A' },
];

const STATUS_FILTER_LABELS: { value: StatusFilter; label: string }[] = [
    { value: 'all', label: 'All' },
    { value: 'open', label: 'Open' },
    { value: 'in_progress', label: 'In Progress' },
    { value: 'blocked', label: 'Blocked' },
    { value: 'complete', label: 'Complete' },
    { value: 'not_applicable', label: 'N/A' },
];

function statusBadgeClass(status: string): string {
    switch (status) {
        case 'open': return 'bg-gray-700 text-gray-300';
        case 'in_progress': return 'bg-[var(--bg-accent)] text-[var(--border-accent-bright)] border border-[var(--border-accent-bright)]';
        case 'blocked': return 'bg-amber-900/40 text-amber-400 border border-amber-700';
        case 'complete': return 'bg-green-900/40 text-[var(--color-success,#4ade80)] border border-green-700';
        case 'not_applicable': return 'bg-gray-800 text-gray-500 border border-gray-700';
        default: return 'bg-gray-700 text-gray-400';
    }
}

function statusLabel(status: string): string {
    return STATUS_OPTIONS.find((o) => o.value === status)?.label ?? status;
}

function evidenceLabel(evidenceItemId: string, evidenceItems: services.EvidenceResponse[]): string {
    const sorted = [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at));
    const idx = sorted.findIndex((e) => e.evidence_item_id === evidenceItemId);
    if (idx === -1) return evidenceItemId;
    return `E${String(idx + 1).padStart(3, '0')}`;
}

function evidenceTabId(evidenceItemId: string): string {
    return `evidence-notes-${evidenceItemId}`;
}

const INPUT_CLASS = 'w-full px-3 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-100 focus:outline-none focus:border-blue-500 placeholder-gray-600';
const LABEL_CLASS = 'block text-xs text-gray-400 mb-1';

export default function TaskListTab({ caseId, evidenceItems, onNavigate }: TaskListTabProps) {
    const [tasks, setTasks] = useState<models.Task[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    // Filters
    const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
    const [evidenceFilter, setEvidenceFilter] = useState<string>('all');

    // Add task form
    const [showAddForm, setShowAddForm] = useState(false);
    const [addTitle, setAddTitle] = useState('');
    const [addDescription, setAddDescription] = useState('');
    const [addEvidenceId, setAddEvidenceId] = useState('');
    const [addError, setAddError] = useState('');
    const [adding, setAdding] = useState(false);

    // Detail panel
    const [detailTask, setDetailTask] = useState<models.Task | null>(null);
    const [editTitle, setEditTitle] = useState('');
    const [editDescription, setEditDescription] = useState('');
    const [editEvidenceId, setEditEvidenceId] = useState('');
    const [saving, setSaving] = useState(false);
    const [detailError, setDetailError] = useState('');

    // Note picker for linking
    const [showNotePicker, setShowNotePicker] = useState(false);
    const [allBlocks, setAllBlocks] = useState<services.NoteBlockResponse[]>([]);
    const [notePickerSearch, setNotePickerSearch] = useState('');
    const [linkingTaskId, setLinkingTaskId] = useState('');

    // Apply template
    const [showTemplateModal, setShowTemplateModal] = useState(false);
    const [templates, setTemplates] = useState<config.TaskTemplate[]>([]);
    const [selectedTemplate, setSelectedTemplate] = useState('');
    const [templateEvidenceId, setTemplateEvidenceId] = useState('');
    const [applying, setApplying] = useState(false);

    // Highlight
    const [highlightedTaskId, setHighlightedTaskId] = useState('');
    const highlightRef = useRef<NodeJS.Timeout | null>(null);

    const sortedEvidenceItems = [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at));

    const fetchTasks = useCallback(() => {
        setLoading(true);
        ListTasks(caseId)
            .then((result) => setTasks(result || []))
            .catch((err: unknown) => setError(String(err)))
            .finally(() => setLoading(false));
    }, [caseId]);

    useEffect(() => {
        fetchTasks();
    }, [fetchTasks]);

    const filteredTasks = tasks.filter((t) => {
        if (statusFilter !== 'all' && t.status !== statusFilter) return false;
        if (evidenceFilter === 'all') return true;
        if (evidenceFilter === 'case_level') return !t.evidence_item_id;
        return t.evidence_item_id === evidenceFilter;
    });

    const handleAddTask = async () => {
        setAddError('');
        if (!addTitle.trim()) { setAddError('Title is required.'); return; }
        setAdding(true);
        try {
            await CreateTask(caseId, addTitle.trim(), addDescription.trim(), addEvidenceId || null);
            setAddTitle('');
            setAddDescription('');
            setAddEvidenceId('');
            setShowAddForm(false);
            fetchTasks();
        } catch (err: unknown) {
            setAddError(String(err));
        } finally {
            setAdding(false);
        }
    };

    const handleStatusChange = async (taskId: string, status: string) => {
        try {
            await UpdateTaskStatus(taskId, status);
            fetchTasks();
            if (detailTask?.task_id === taskId) {
                setDetailTask((prev) => prev ? { ...prev, status: status as models.Task['status'] } : null);
            }
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    const handleSaveDetail = async () => {
        if (!detailTask) return;
        setDetailError('');
        if (!editTitle.trim()) { setDetailError('Title is required.'); return; }
        setSaving(true);
        try {
            await UpdateTask(detailTask.task_id, editTitle.trim(), editDescription.trim(), editEvidenceId || null);
            fetchTasks();
            setDetailTask((prev) => prev ? {
                ...prev,
                title: editTitle.trim(),
                description: editDescription.trim(),
                evidence_item_id: editEvidenceId || null,
            } : null);
        } catch (err: unknown) {
            setDetailError(String(err));
        } finally {
            setSaving(false);
        }
    };

    const handleDeleteTask = async (taskId: string) => {
        try {
            await DeleteTask(taskId);
            if (detailTask?.task_id === taskId) setDetailTask(null);
            fetchTasks();
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    const openDetail = (task: models.Task) => {
        setDetailTask(task);
        setEditTitle(task.title);
        setEditDescription(task.description);
        setEditEvidenceId(task.evidence_item_id ?? '');
        setDetailError('');
    };

    const handleUnlink = async (taskId: string, blockId: string) => {
        try {
            await UnlinkNoteFromTask(taskId, blockId);
            fetchTasks();
            if (detailTask?.task_id === taskId) {
                const updated = tasks.find((t) => t.task_id === taskId);
                if (updated) {
                    setDetailTask({ ...updated, linked_blocks: updated.linked_blocks.filter((b) => b.block_id !== blockId) });
                }
            }
        } catch (err: unknown) {
            setDetailError(String(err));
        }
    };

    const openNotePicker = async (taskId: string) => {
        setLinkingTaskId(taskId);
        setNotePickerSearch('');
        setShowNotePicker(true);
        try {
            const [masterBlocks, ...evidenceBlockArrays] = await Promise.all([
                ListNotes(caseId),
                ...evidenceItems.map((item) => ListEvidenceNotes(caseId, item.evidence_item_id)),
            ]);
            setAllBlocks([
                ...(masterBlocks || []),
                ...evidenceBlockArrays.flatMap((blocks) => blocks || []),
            ]);
        } catch { /* ignore */ }
    };

    const handleLinkNote = async (blockId: string) => {
        try {
            await LinkNoteToTask(linkingTaskId, blockId);
            setShowNotePicker(false);
            fetchTasks();
        } catch (err: unknown) {
            setDetailError(String(err));
        }
    };

    const handleNavigateToBlock = (block: models.LinkedBlock) => {
        const tab = block.source ? evidenceTabId(block.source) : 'notes';
        onNavigate?.(tab, block.block_id);
    };

    const openTemplateModal = async () => {
        try {
            const tpls = await LoadTemplates();
            setTemplates(tpls || []);
            setSelectedTemplate(tpls?.[0]?.name ?? '');
            setTemplateEvidenceId('');
            setShowTemplateModal(true);
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    const handleApplyTemplate = async () => {
        if (!selectedTemplate) return;
        setApplying(true);
        try {
            await ApplyTemplate(caseId, selectedTemplate, templateEvidenceId || null);
            setShowTemplateModal(false);
            fetchTasks();
        } catch (err: unknown) {
            setError(String(err));
        } finally {
            setApplying(false);
        }
    };

    const highlightRow = (taskId: string) => {
        if (highlightRef.current) clearTimeout(highlightRef.current);
        setHighlightedTaskId(taskId);
        highlightRef.current = setTimeout(() => setHighlightedTaskId(''), 1500);
    };

    const selectedTemplateObj = templates.find((t) => t.name === selectedTemplate);

    const filteredPickerBlocks = allBlocks.filter((b) => {
        if (!notePickerSearch.trim()) return true;
        return b.content.toLowerCase().includes(notePickerSearch.toLowerCase());
    });

    const [confirmDeleteId, setConfirmDeleteId] = useState('');

    // When tasks list refreshes, sync the open detail panel to the latest data.
    useEffect(() => {
        if (!detailTask) return;
        const fresh = tasks.find((t) => t.task_id === detailTask.task_id);
        if (fresh) setDetailTask(fresh);
    }, [tasks, detailTask?.task_id]); // eslint-disable-line react-hooks/exhaustive-deps

    return (
        <div className="space-y-4">
            {/* Toolbar */}
            <div className="flex flex-wrap items-center gap-2">
                <button
                    onClick={() => setShowAddForm((v) => !v)}
                    className="px-3 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors"
                >
                    {showAddForm ? 'Cancel' : '+ Add Task'}
                </button>
                <button
                    onClick={openTemplateModal}
                    className="px-3 py-1.5 text-sm border border-gray-600 hover:border-gray-400 text-gray-300 hover:text-gray-100 rounded transition-colors"
                >
                    Apply Template
                </button>
                <div className="flex-1" />
                {/* Status filter */}
                <div className="flex gap-1 flex-wrap">
                    {STATUS_FILTER_LABELS.map((f) => (
                        <button
                            key={f.value}
                            onClick={() => setStatusFilter(f.value)}
                            className={`px-2 py-1 text-xs rounded border transition-colors ${
                                statusFilter === f.value
                                    ? 'border-[var(--border-accent-bright)] bg-[var(--bg-accent)] text-gray-100'
                                    : 'border-gray-700 text-gray-400 hover:text-gray-200'
                            }`}
                        >
                            {f.label}
                        </button>
                    ))}
                </div>
                {/* Evidence filter */}
                <select
                    value={evidenceFilter}
                    onChange={(e) => setEvidenceFilter(e.target.value)}
                    className="px-2 py-1 text-xs bg-gray-800 border border-gray-700 rounded text-gray-300 focus:outline-none"
                >
                    <option value="all">All Evidence</option>
                    <option value="case_level">Case Level</option>
                    {sortedEvidenceItems.map((item, idx) => (
                        <option key={item.evidence_item_id} value={item.evidence_item_id}>
                            E{String(idx + 1).padStart(3, '0')} - {item.name}
                        </option>
                    ))}
                </select>
            </div>

            {/* Add Task Form */}
            {showAddForm && (
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4 space-y-3">
                    <h3 className="text-sm font-medium text-gray-300">New Task</h3>
                    <div>
                        <label className={LABEL_CLASS}>Title *</label>
                        <input
                            type="text"
                            value={addTitle}
                            onChange={(e) => setAddTitle(e.target.value)}
                            onKeyDown={(e) => { if (e.key === 'Enter') handleAddTask(); }}
                            placeholder="Task title"
                            className={INPUT_CLASS}
                            autoFocus
                        />
                    </div>
                    <div>
                        <label className={LABEL_CLASS}>Description</label>
                        <textarea
                            rows={2}
                            value={addDescription}
                            onChange={(e) => setAddDescription(e.target.value)}
                            placeholder="Optional description"
                            className={INPUT_CLASS + ' resize-none'}
                        />
                    </div>
                    <div>
                        <label className={LABEL_CLASS}>Evidence Item</label>
                        <select
                            value={addEvidenceId}
                            onChange={(e) => setAddEvidenceId(e.target.value)}
                            className={INPUT_CLASS}
                        >
                            <option value="">Case Level</option>
                            {sortedEvidenceItems.map((item, idx) => (
                                <option key={item.evidence_item_id} value={item.evidence_item_id}>
                                    E{String(idx + 1).padStart(3, '0')} - {item.name}
                                </option>
                            ))}
                        </select>
                    </div>
                    {addError && <p className="text-xs text-red-400">{addError}</p>}
                    <div className="flex gap-2">
                        <button
                            onClick={handleAddTask}
                            disabled={adding}
                            className="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded transition-colors"
                        >
                            {adding ? 'Adding...' : 'Add Task'}
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

            {/* Main layout: task list + detail panel */}
            <div className={`flex gap-4 ${detailTask ? 'items-start' : ''}`}>
                {/* Task Table */}
                <div className={detailTask ? 'flex-1 min-w-0' : 'w-full'}>
                    {loading ? (
                        <div className="flex justify-center py-8">
                            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-400" />
                        </div>
                    ) : filteredTasks.length === 0 ? (
                        <p className="text-gray-500 text-sm text-center py-8">
                            {tasks.length === 0 ? 'No tasks yet. Add a task or apply a template.' : 'No tasks match the current filters.'}
                        </p>
                    ) : (
                        <div className="overflow-x-auto">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="text-left border-b border-gray-700">
                                        <th className="pb-2 pr-3 font-medium text-gray-400 w-28">Status</th>
                                        <th className="pb-2 pr-3 font-medium text-gray-400">Title</th>
                                        <th className="pb-2 pr-3 font-medium text-gray-400 w-24">Evidence</th>
                                        <th className="pb-2 pr-3 font-medium text-gray-400 w-20">Notes</th>
                                        <th className="pb-2 pr-3 font-medium text-gray-400 w-36">Completed</th>
                                        <th className="pb-2 font-medium text-gray-400 w-24"></th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-gray-800">
                                    {filteredTasks.map((task) => {
                                        if (confirmDeleteId === task.task_id) {
                                            return (
                                                <tr key={task.task_id} className="bg-red-900/20">
                                                    <td colSpan={6} className="py-3 px-2">
                                                        <div className="flex items-center gap-3">
                                                            <span className="text-sm text-red-300">Delete this task? This cannot be undone.</span>
                                                            <button
                                                                onClick={() => { handleDeleteTask(task.task_id); setConfirmDeleteId(''); }}
                                                                className="text-xs bg-red-700 hover:bg-red-600 text-white px-3 py-1 rounded transition-colors"
                                                            >
                                                                Delete
                                                            </button>
                                                            <button
                                                                onClick={() => setConfirmDeleteId('')}
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
                                            <tr
                                                key={task.task_id}
                                                id={task.task_id}
                                                className={`hover:bg-gray-800 ${highlightedTaskId === task.task_id ? 'block-target' : ''}`}
                                            >
                                                <td className="py-2 pr-3 align-middle">
                                                    <select
                                                        value={task.status}
                                                        onChange={(e) => handleStatusChange(task.task_id, e.target.value)}
                                                        className={`text-xs px-2 py-0.5 rounded font-medium border-0 focus:outline-none cursor-pointer ${statusBadgeClass(task.status)}`}
                                                    >
                                                        {STATUS_OPTIONS.map((o) => (
                                                            <option key={o.value} value={o.value}>{o.label}</option>
                                                        ))}
                                                    </select>
                                                </td>
                                                <td className="py-2 pr-3 align-middle">
                                                    <button
                                                        onClick={() => openDetail(task)}
                                                        className="text-left text-gray-200 hover:text-blue-400 transition-colors"
                                                    >
                                                        {task.title}
                                                    </button>
                                                </td>
                                                <td className="py-2 pr-3 align-middle text-xs">
                                                    {task.evidence_item_id ? (
                                                        <button
                                                            onClick={() => onNavigate?.(evidenceTabId(task.evidence_item_id!))}
                                                            className="text-blue-400 hover:text-blue-300 underline transition-colors"
                                                        >
                                                            {evidenceLabel(task.evidence_item_id, evidenceItems)}
                                                        </button>
                                                    ) : (
                                                        <span className="text-gray-600">Case Level</span>
                                                    )}
                                                </td>
                                                <td className="py-2 pr-3 align-middle text-xs text-gray-400">
                                                    {task.linked_blocks?.length ?? 0}
                                                </td>
                                                <td className="py-2 pr-3 align-middle text-xs font-mono text-gray-500">
                                                    {task.completed_at ?? ''}
                                                </td>
                                                <td className="py-2 align-middle">
                                                    <button
                                                        onClick={() => setConfirmDeleteId(task.task_id)}
                                                        className="text-xs text-gray-500 hover:text-red-400 transition-colors"
                                                    >
                                                        Delete
                                                    </button>
                                                </td>
                                            </tr>
                                        );
                                    })}
                                </tbody>
                            </table>
                        </div>
                    )}
                </div>

                {/* Detail Panel */}
                {detailTask && (
                    <div className="w-80 flex-shrink-0 bg-gray-800 border border-gray-700 rounded-lg p-4 space-y-4">
                        <div className="flex items-center justify-between">
                            <h3 className="text-sm font-semibold text-gray-200">Task Detail</h3>
                            <button
                                onClick={() => setDetailTask(null)}
                                className="text-gray-500 hover:text-gray-300 transition-colors"
                                aria-label="Close panel"
                            >
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                                </svg>
                            </button>
                        </div>

                        <div>
                            <label className={LABEL_CLASS}>Title</label>
                            <input
                                type="text"
                                value={editTitle}
                                onChange={(e) => setEditTitle(e.target.value)}
                                className={INPUT_CLASS}
                            />
                        </div>
                        <div>
                            <label className={LABEL_CLASS}>Description</label>
                            <textarea
                                rows={3}
                                value={editDescription}
                                onChange={(e) => setEditDescription(e.target.value)}
                                className={INPUT_CLASS + ' resize-none'}
                            />
                        </div>
                        <div>
                            <label className={LABEL_CLASS}>Status</label>
                            <select
                                value={detailTask.status}
                                onChange={(e) => handleStatusChange(detailTask.task_id, e.target.value)}
                                className={INPUT_CLASS}
                            >
                                {STATUS_OPTIONS.map((o) => (
                                    <option key={o.value} value={o.value}>{o.label}</option>
                                ))}
                            </select>
                        </div>
                        <div>
                            <label className={LABEL_CLASS}>Evidence Item</label>
                            <select
                                value={editEvidenceId}
                                onChange={(e) => setEditEvidenceId(e.target.value)}
                                className={INPUT_CLASS}
                            >
                                <option value="">Case Level</option>
                                {sortedEvidenceItems.map((item, idx) => (
                                    <option key={item.evidence_item_id} value={item.evidence_item_id}>
                                        E{String(idx + 1).padStart(3, '0')} - {item.name}
                                    </option>
                                ))}
                            </select>
                        </div>

                        {detailTask.status === 'complete' && detailTask.completed_at && (
                            <div>
                                <p className="text-xs text-gray-400">Completed</p>
                                <p className="text-xs font-mono text-gray-300">{detailTask.completed_at}</p>
                            </div>
                        )}

                        {detailTask.template_name && (
                            <div>
                                <p className="text-xs text-gray-500">From template: <span className="text-gray-400">{detailTask.template_name}</span></p>
                            </div>
                        )}

                        {detailError && <p className="text-xs text-red-400">{detailError}</p>}

                        <div className="flex gap-2">
                            <button
                                onClick={handleSaveDetail}
                                disabled={saving}
                                className="flex-1 px-3 py-1.5 text-xs bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded transition-colors"
                            >
                                {saving ? 'Saving...' : 'Save Changes'}
                            </button>
                            <button
                                onClick={() => setDetailTask(null)}
                                className="px-3 py-1.5 text-xs border border-gray-600 hover:border-gray-400 text-gray-400 hover:text-gray-200 rounded transition-colors"
                            >
                                Close
                            </button>
                        </div>

                        {/* Linked Notes */}
                        <div className="border-t border-gray-700 pt-3">
                            <div className="flex items-center justify-between mb-2">
                                <h4 className="text-xs font-medium text-gray-400 uppercase tracking-wider">Linked Notes</h4>
                                <button
                                    onClick={() => openNotePicker(detailTask.task_id)}
                                    className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                                >
                                    + Link a Note
                                </button>
                            </div>
                            {(!detailTask.linked_blocks || detailTask.linked_blocks.length === 0) ? (
                                <p className="text-xs text-gray-600">No notes linked.</p>
                            ) : (
                                <div className="space-y-2">
                                    {detailTask.linked_blocks.map((block) => (
                                        <div key={block.block_id} className="bg-gray-750 border border-gray-700 rounded p-2 space-y-1">
                                            <div className="flex items-center justify-between">
                                                <span className="text-xs text-gray-500 font-mono">
                                                    {block.source ? evidenceLabel(block.source, evidenceItems) : 'Master Notes'}
                                                </span>
                                                <div className="flex gap-2">
                                                    <button
                                                        onClick={() => handleNavigateToBlock(block)}
                                                        className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                                                    >
                                                        Navigate
                                                    </button>
                                                    <button
                                                        onClick={() => handleUnlink(detailTask.task_id, block.block_id)}
                                                        className="text-xs text-gray-500 hover:text-red-400 transition-colors"
                                                    >
                                                        Unlink
                                                    </button>
                                                </div>
                                            </div>
                                            <p className="text-xs text-gray-500 font-mono">{block.committed_at}</p>
                                            {block.preview && (
                                                <p className="text-xs text-gray-400 truncate">{block.preview}</p>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>
                )}
            </div>

            {/* Note Picker Modal */}
            {showNotePicker && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
                    <div className="bg-gray-800 border border-gray-700 rounded-lg shadow-xl w-full max-w-lg p-4 space-y-3">
                        <div className="flex items-center justify-between">
                            <h3 className="text-sm font-semibold text-gray-200">Link a Note</h3>
                            <button onClick={() => setShowNotePicker(false)} className="text-gray-500 hover:text-gray-300">
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                                </svg>
                            </button>
                        </div>
                        <input
                            type="text"
                            placeholder="Search notes..."
                            value={notePickerSearch}
                            onChange={(e) => setNotePickerSearch(e.target.value)}
                            className={INPUT_CLASS}
                            autoFocus
                        />
                        <div className="max-h-72 overflow-y-auto space-y-1">
                            {filteredPickerBlocks.length === 0 ? (
                                <p className="text-xs text-gray-500 py-2 text-center">No committed notes found.</p>
                            ) : (
                                filteredPickerBlocks.map((block) => (
                                    <button
                                        key={block.block_id}
                                        onClick={() => handleLinkNote(block.block_id)}
                                        className="w-full text-left px-3 py-2 bg-gray-750 border border-gray-700 rounded hover:border-blue-600 hover:bg-gray-700 transition-colors space-y-0.5"
                                    >
                                        <div className="flex gap-2 text-xs text-gray-500 font-mono">
                                            <span>{block.created_at}</span>
                                        </div>
                                        <p className="text-xs text-gray-300 truncate">{block.content.slice(0, 120)}</p>
                                    </button>
                                ))
                            )}
                        </div>
                        <button
                            onClick={() => setShowNotePicker(false)}
                            className="w-full py-1.5 text-sm text-gray-400 hover:text-gray-200 border border-gray-700 rounded transition-colors"
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            )}

            {/* Apply Template Modal */}
            {showTemplateModal && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
                    <div className="bg-gray-800 border border-gray-700 rounded-lg shadow-xl w-full max-w-md p-4 space-y-4">
                        <div className="flex items-center justify-between">
                            <h3 className="text-sm font-semibold text-gray-200">Apply Template</h3>
                            <button onClick={() => setShowTemplateModal(false)} className="text-gray-500 hover:text-gray-300">
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                                </svg>
                            </button>
                        </div>

                        {templates.length === 0 ? (
                            <p className="text-sm text-gray-400 py-2">No templates configured. Add templates in Settings.</p>
                        ) : (
                            <>
                                <div>
                                    <label className={LABEL_CLASS}>Template</label>
                                    <select
                                        value={selectedTemplate}
                                        onChange={(e) => setSelectedTemplate(e.target.value)}
                                        className={INPUT_CLASS}
                                    >
                                        {templates.map((t) => (
                                            <option key={t.name} value={t.name}>{t.name}</option>
                                        ))}
                                    </select>
                                </div>

                                {selectedTemplateObj && (
                                    <div>
                                        <label className={LABEL_CLASS}>Tasks to be added</label>
                                        <ul className="space-y-1 max-h-40 overflow-y-auto">
                                            {selectedTemplateObj.tasks.map((tt, i) => (
                                                <li key={i} className="text-xs text-gray-300 flex items-start gap-1.5">
                                                    <span className="text-gray-600 mt-0.5">-</span>
                                                    <span>{tt.title}</span>
                                                </li>
                                            ))}
                                        </ul>
                                    </div>
                                )}

                                <div>
                                    <label className={LABEL_CLASS}>Assign to</label>
                                    <select
                                        value={templateEvidenceId}
                                        onChange={(e) => setTemplateEvidenceId(e.target.value)}
                                        className={INPUT_CLASS}
                                    >
                                        <option value="">Case Level</option>
                                        {sortedEvidenceItems.map((item, idx) => (
                                            <option key={item.evidence_item_id} value={item.evidence_item_id}>
                                                E{String(idx + 1).padStart(3, '0')} - {item.name}
                                            </option>
                                        ))}
                                    </select>
                                </div>

                                <div className="flex gap-2">
                                    <button
                                        onClick={handleApplyTemplate}
                                        disabled={applying}
                                        className="flex-1 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded transition-colors"
                                    >
                                        {applying ? 'Applying...' : 'Apply Template'}
                                    </button>
                                    <button
                                        onClick={() => setShowTemplateModal(false)}
                                        className="px-4 py-1.5 text-sm border border-gray-600 hover:border-gray-400 text-gray-400 hover:text-gray-200 rounded transition-colors"
                                    >
                                        Cancel
                                    </button>
                                </div>
                            </>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}
