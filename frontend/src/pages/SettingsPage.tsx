import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { GetConfig, SaveConfig, GetDBPath, ChooseDirectory, GetBackupStatus, TriggerBackupNow } from '../../wailsjs/go/main/App';
import { config, backup } from '../../wailsjs/go/models';
import ErrorMessage from '../components/ErrorMessage';
import DBLocationDialog from '../components/DBLocationDialog';
import { useAuth } from '../context/AuthContext';

const APP_VERSION = '0.4.0';

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

    const isDirty = cfg !== null && initialCfg !== null && (
        cfg.backup_enabled !== initialCfg.backup_enabled ||
        cfg.backup_dest_path !== initialCfg.backup_dest_path ||
        cfg.backup_interval_hours !== initialCfg.backup_interval_hours ||
        cfg.backup_keep_count !== initialCfg.backup_keep_count
    );

    useEffect(() => {
        Promise.all([GetConfig(), GetDBPath(), GetBackupStatus()])
            .then(([c, p, bs]) => {
                setCfg(c);
                setInitialCfg(c);
                setDbPath(p);
                setBackupStatus(bs);
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

            {/* About */}
            <section className="mb-8">
                <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-4">About</h2>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <p className="text-sm text-gray-300">dfnotes-go <span className="text-gray-500">v{APP_VERSION}</span></p>
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
