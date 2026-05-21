import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { GetConfig, SaveConfig, GetDBPath, ChooseDirectory, GetBackupStatus, TriggerBackupNow, LoadTemplates, SaveTemplates, GetVersion } from '../../wailsjs/go/main/App';
import { config, backup } from '../../wailsjs/go/models';
import ErrorMessage from '../components/ErrorMessage';
import DBLocationDialog from '../components/DBLocationDialog';
import { useAuth } from '../context/AuthContext';

const TPL_INPUT = 'w-full px-3 py-1.5 bg-gray-700 border border-gray-600 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500 placeholder-gray-500';

function TemplateEditor({
    template,
    onChange,
    onSave,
    onCancel,
}: {
    template: TaskTemplate;
    onChange: (t: TaskTemplate) => void;
    onSave: () => void;
    onCancel: () => void;
}) {
    const updateTask = (idx: number, field: keyof TemplateTask, value: string) => {
        const tasks = template.tasks.map((t, i) => i === idx ? { ...t, [field]: value } : t);
        onChange({ ...template, tasks });
    };
    const addTask = () => onChange({ ...template, tasks: [...template.tasks, { title: '', description: '' }] });
    const removeTask = (idx: number) => onChange({ ...template, tasks: template.tasks.filter((_, i) => i !== idx) });

    return (
        <div className="border border-gray-600 rounded-lg p-3 space-y-3 bg-gray-750">
            <div>
                <label className="block text-xs text-gray-400 mb-1">Template Name</label>
                <input
                    type="text"
                    value={template.name}
                    onChange={(e) => onChange({ ...template, name: e.target.value })}
                    placeholder="e.g. Hard Drive Imaging"
                    className={TPL_INPUT}
                    autoFocus
                />
            </div>
            <div className="space-y-2">
                <label className="block text-xs text-gray-400">Tasks</label>
                {template.tasks.map((task, idx) => (
                    <div key={idx} className="flex gap-2 items-start">
                        <div className="flex-1 space-y-1">
                            <input
                                type="text"
                                value={task.title}
                                onChange={(e) => updateTask(idx, 'title', e.target.value)}
                                placeholder="Task title"
                                className={TPL_INPUT}
                            />
                            <input
                                type="text"
                                value={task.description}
                                onChange={(e) => updateTask(idx, 'description', e.target.value)}
                                placeholder="Description (optional)"
                                className={TPL_INPUT}
                            />
                        </div>
                        <button
                            onClick={() => removeTask(idx)}
                            className="mt-1 text-gray-500 hover:text-red-400 transition-colors text-xs"
                            title="Remove task"
                        >
                            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                            </svg>
                        </button>
                    </div>
                ))}
                <button
                    onClick={addTask}
                    className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                >
                    + Add Task
                </button>
            </div>
            <div className="flex gap-2">
                <button
                    onClick={onSave}
                    className="px-3 py-1.5 text-xs bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors"
                >
                    Save Template
                </button>
                <button
                    onClick={onCancel}
                    className="px-3 py-1.5 text-xs border border-gray-600 hover:border-gray-400 text-gray-400 hover:text-gray-200 rounded transition-colors"
                >
                    Cancel
                </button>
            </div>
        </div>
    );
}

interface TemplateTask {
    title: string;
    description: string;
}

interface TaskTemplate {
    name: string;
    tasks: TemplateTask[];
}

function emptyTemplate(): TaskTemplate {
    return { name: '', tasks: [{ title: '', description: '' }] };
}

export default function SettingsPage() {
    const navigate = useNavigate();
    const { resetToLogin } = useAuth();
    const [cfg, setCfg] = useState<config.Config | null>(null);
    const [initialCfg, setInitialCfg] = useState<config.Config | null>(null);
    const [dbPath, setDbPath] = useState('');
    const [savedMsg, setSavedMsg] = useState('');
    const [error, setError] = useState('');
    const [showDBDialog, setShowDBDialog] = useState(false);
    const [backupStatus, setBackupStatus] = useState<backup.Status | null>(null);
    const [backupMsg, setBackupMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
    const [appVersion, setAppVersion] = useState('');

    // Templates state
    const [templates, setTemplates] = useState<TaskTemplate[]>([]);
    const [editingTemplate, setEditingTemplate] = useState<TaskTemplate | null>(null);
    const [editingIndex, setEditingIndex] = useState<number | null>(null); // null = new
    const [confirmDeleteTpl, setConfirmDeleteTpl] = useState<number | null>(null);
    const [tplError, setTplError] = useState('');

    const isDirty = cfg !== null && initialCfg !== null && (
        cfg.backup_enabled !== initialCfg.backup_enabled ||
        cfg.backup_dest_path !== initialCfg.backup_dest_path ||
        cfg.backup_interval_hours !== initialCfg.backup_interval_hours ||
        cfg.backup_keep_count !== initialCfg.backup_keep_count
    );

    useEffect(() => {
        Promise.all([GetConfig(), GetDBPath(), GetBackupStatus(), LoadTemplates(), GetVersion()])
            .then(([c, p, bs, tpls, ver]) => {
                setCfg(c);
                setInitialCfg(c);
                setDbPath(p);
                setBackupStatus(bs);
                setAppVersion(ver);
                setTemplates((tpls || []).map((t: config.TaskTemplate) => ({
                    name: t.name,
                    tasks: (t.tasks || []).map((tt: config.TemplateTask) => ({ title: tt.title, description: tt.description })),
                })));
            })
            .catch((err: unknown) => setError(String(err)));
    }, []);

    const handleSave = async () => {
        if (!cfg) return;
        try {
            await SaveConfig(cfg);
            setInitialCfg(cfg);
            setSavedMsg('Saved');
            setTimeout(() => setSavedMsg(''), 2000);
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    const handleBrowseBackupDest = async () => {
        try {
            const dir = await ChooseDirectory();
            if (dir && cfg) {
                setCfg({ ...cfg, backup_dest_path: dir });
            }
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    const updateCfg = (patch: Partial<config.Config>) => {
        if (cfg) setCfg({ ...cfg, ...patch });
    };

    const saveTemplates = async (updated: TaskTemplate[]) => {
        try {
            await SaveTemplates(updated as config.TaskTemplate[]);
            setTemplates(updated);
            setTplError('');
        } catch (err: unknown) {
            setTplError(String(err));
        }
    };

    const handleSaveTemplate = async () => {
        if (!editingTemplate) return;
        if (!editingTemplate.name.trim()) { setTplError('Template name is required.'); return; }
        const updated = [...templates];
        if (editingIndex === null) {
            updated.push({ ...editingTemplate, name: editingTemplate.name.trim() });
        } else {
            updated[editingIndex] = { ...editingTemplate, name: editingTemplate.name.trim() };
        }
        await saveTemplates(updated);
        setEditingTemplate(null);
        setEditingIndex(null);
    };

    const handleDeleteTemplate = async (idx: number) => {
        const updated = templates.filter((_, i) => i !== idx);
        await saveTemplates(updated);
        setConfirmDeleteTpl(null);
    };

    if (!cfg) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-400" />
            </div>
        );
    }

    return (
        <div className="min-h-screen p-6 max-w-2xl mx-auto">
            <button
                onClick={() => navigate(-1)}
                className="text-sm text-gray-400 hover:text-gray-200 mb-6 inline-flex items-center gap-1 transition-colors"
            >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
                </svg>
                Back
            </button>

            <h1 className="text-2xl font-bold text-gray-100 mb-8">Settings</h1>

            <ErrorMessage message={error} onDismiss={() => setError('')} />

            {/* Database */}
            <section className="mb-8">
                <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-4">Database</h2>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4 space-y-3">
                    <div>
                        <p className="text-xs text-gray-400 mb-1">Current database path</p>
                        <p className="text-sm text-gray-300 font-mono break-all">{dbPath}</p>
                    </div>
                    <button
                        onClick={() => setShowDBDialog(true)}
                        className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
                    >
                        Change Location...
                    </button>
                </div>
            </section>

            {/* Backup */}
            <section className="mb-8">
                <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-4">Backup</h2>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4 space-y-4">
                    <label className="flex items-center gap-3 cursor-pointer">
                        <input
                            type="checkbox"
                            checked={cfg.backup_enabled}
                            onChange={(e) => updateCfg({ backup_enabled: e.target.checked })}
                            className="rounded"
                        />
                        <span className="text-sm text-gray-200">Enable automated backups</span>
                    </label>

                    <div>
                        <label className="block text-xs text-gray-400 mb-1">Backup destination</label>
                        <div className="flex gap-2">
                            <input
                                type="text"
                                value={cfg.backup_dest_path}
                                onChange={(e) => updateCfg({ backup_dest_path: e.target.value })}
                                placeholder="Select a directory..."
                                className="flex-1 px-3 py-1.5 bg-gray-700 border border-gray-600 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
                            />
                            <button
                                onClick={handleBrowseBackupDest}
                                className="px-3 py-1.5 bg-gray-700 hover:bg-gray-600 border border-gray-600 rounded text-sm text-gray-200 transition-colors"
                            >
                                Browse
                            </button>
                        </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="block text-xs text-gray-400 mb-1">Interval (hours)</label>
                            <input
                                type="number"
                                min={1}
                                max={168}
                                value={cfg.backup_interval_hours}
                                onChange={(e) => updateCfg({ backup_interval_hours: Math.min(168, Math.max(1, Number(e.target.value))) })}
                                className="w-full px-3 py-1.5 bg-gray-700 border border-gray-600 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
                            />
                        </div>
                        <div>
                            <label className="block text-xs text-gray-400 mb-1">Backups to keep</label>
                            <input
                                type="number"
                                min={1}
                                max={100}
                                value={cfg.backup_keep_count}
                                onChange={(e) => updateCfg({ backup_keep_count: Math.min(100, Math.max(1, Number(e.target.value))) })}
                                className="w-full px-3 py-1.5 bg-gray-700 border border-gray-600 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
                            />
                        </div>
                    </div>

                    <div className="pt-1 border-t border-gray-700 space-y-2">
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-xs text-gray-400">
                                    Last backup:
                                    <span className={`ml-1 ${backupStatus?.last_backup_status === 'success' ? 'text-green-400' : backupStatus?.last_backup_status === 'failed' ? 'text-red-400' : 'text-gray-500'}`}>
                                        {backupStatus ? (backupStatus.last_backup_status === 'never' ? 'Never' : backupStatus.last_backup_status) : '...'}
                                    </span>
                                </p>
                                {backupStatus?.last_backup_time && (
                                    <p className="text-xs text-gray-500">{new Date(backupStatus.last_backup_time).toLocaleString()}</p>
                                )}
                            </div>
                            <button
                                onClick={async () => {
                                    setBackupMsg(null);
                                    if (isDirty) {
                                        setBackupMsg({ type: 'error', text: 'Save changes first' });
                                        return;
                                    }
                                    try {
                                        await TriggerBackupNow();
                                        const bs = await GetBackupStatus();
                                        setBackupStatus(bs);
                                        setBackupMsg({ type: 'success', text: 'Backup complete' });
                                        setTimeout(() => setBackupMsg(null), 3000);
                                    } catch (err: unknown) {
                                        setBackupMsg({ type: 'error', text: String(err) });
                                    }
                                }}
                                className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                            >
                                Back up now
                            </button>
                        </div>
                        {backupMsg && (
                            <p className={`text-xs ${backupMsg.type === 'success' ? 'text-green-400' : 'text-red-400'}`}>
                                {backupMsg.text}
                            </p>
                        )}
                    </div>
                </div>
            </section>

            {/* Templates */}
            <section className="mb-8">
                <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-4">Templates</h2>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4 space-y-3">
                    {tplError && (
                        <p className="text-xs text-red-400">{tplError}</p>
                    )}
                    {templates.length === 0 && !editingTemplate && (
                        <p className="text-sm text-gray-500">No templates yet.</p>
                    )}
                    {templates.map((tpl, idx) => (
                        <div key={idx}>
                            {confirmDeleteTpl === idx ? (
                                <div className="flex items-center gap-3 py-1">
                                    <span className="text-sm text-red-300">Delete "{tpl.name}"? Tasks already added to cases will not be affected.</span>
                                    <button
                                        onClick={() => handleDeleteTemplate(idx)}
                                        className="text-xs bg-red-700 hover:bg-red-600 text-white px-3 py-1 rounded transition-colors"
                                    >
                                        Delete
                                    </button>
                                    <button
                                        onClick={() => setConfirmDeleteTpl(null)}
                                        className="text-xs text-gray-400 hover:text-gray-200 px-3 py-1 rounded border border-gray-600 transition-colors"
                                    >
                                        Cancel
                                    </button>
                                </div>
                            ) : editingIndex === idx && editingTemplate ? (
                                <TemplateEditor
                                    template={editingTemplate}
                                    onChange={setEditingTemplate}
                                    onSave={handleSaveTemplate}
                                    onCancel={() => { setEditingTemplate(null); setEditingIndex(null); setTplError(''); }}
                                />
                            ) : (
                                <div className="flex items-center justify-between py-1">
                                    <span className="text-sm text-gray-200">{tpl.name} <span className="text-xs text-gray-500">({tpl.tasks.length} task{tpl.tasks.length !== 1 ? 's' : ''})</span></span>
                                    <div className="flex gap-3">
                                        <button
                                            onClick={() => { setEditingTemplate({ ...tpl, tasks: tpl.tasks.map((t) => ({ ...t })) }); setEditingIndex(idx); setTplError(''); }}
                                            className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                                        >
                                            Edit
                                        </button>
                                        <button
                                            onClick={() => setConfirmDeleteTpl(idx)}
                                            className="text-xs text-gray-500 hover:text-red-400 transition-colors"
                                        >
                                            Delete
                                        </button>
                                    </div>
                                </div>
                            )}
                        </div>
                    ))}

                    {editingIndex === null && editingTemplate ? (
                        <TemplateEditor
                            template={editingTemplate}
                            onChange={setEditingTemplate}
                            onSave={handleSaveTemplate}
                            onCancel={() => { setEditingTemplate(null); setTplError(''); }}
                        />
                    ) : (
                        !editingTemplate && (
                            <button
                                onClick={() => { setEditingTemplate(emptyTemplate()); setEditingIndex(null); setTplError(''); }}
                                className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
                            >
                                + New Template
                            </button>
                        )
                    )}
                </div>
            </section>

            {/* About */}
            <section className="mb-8">
                <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-4">About</h2>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <p className="text-sm text-gray-300">dfnotes-go <span className="text-gray-500">v{appVersion}</span></p>
                    <p className="text-sm text-gray-400 mt-1">Digital Forensic Notebook</p>
                </div>
            </section>

            {/* Save */}
            <div className="flex items-center gap-4">
                <button
                    onClick={handleSave}
                    className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors text-sm font-medium"
                >
                    Save
                </button>
                {isDirty && !savedMsg && (
                    <span className="text-sm text-amber-400 flex items-center gap-1.5">
                        <span className="w-2 h-2 rounded-full bg-amber-400 inline-block" />
                        Unsaved changes
                    </span>
                )}
                {savedMsg && (
                    <span className="text-sm text-green-400 flex items-center gap-1">
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2.5} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                        </svg>
                        {savedMsg}
                    </span>
                )}
            </div>

            {showDBDialog && (
                <DBLocationDialog
                    onClose={() => setShowDBDialog(false)}
                    onSuccess={async () => {
                        setShowDBDialog(false);
                        await resetToLogin();
                    }}
                />
            )}
        </div>
    );
}
