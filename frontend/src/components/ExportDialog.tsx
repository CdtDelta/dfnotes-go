import { useEffect, useState } from 'react';
import { ExportCase, ChooseExportSavePath } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

type Stage = 'password' | 'progress' | 'complete' | 'error';

interface ExportDialogProps {
    caseID: string;
    caseNumber: string;
    onClose: () => void;
}

function passwordStrength(pw: string): { label: string; color: string } {
    if (pw.length === 0) return { label: '', color: '' };
    if (pw.length < 8) return { label: 'Weak', color: 'text-red-400' };
    if (pw.length < 16) return { label: 'Fair', color: 'text-yellow-400' };
    return { label: 'Strong', color: 'text-green-400' };
}

export default function ExportDialog({ caseID, caseNumber, onClose }: ExportDialogProps) {
    const [stage, setStage] = useState<Stage>('password');
    const [password, setPassword] = useState('');
    const [confirm, setConfirm] = useState('');
    const [validationError, setValidationError] = useState('');
    const [progressStage, setProgressStage] = useState('');
    const [progressPercent, setProgressPercent] = useState(0);
    const [archivePath, setArchivePath] = useState('');
    const [exportError, setExportError] = useState('');

    useEffect(() => {
        const offProgress = EventsOn('export:progress', (data: { stage: string; percent: number }) => {
            setProgressStage(data.stage);
            setProgressPercent(data.percent);
        });
        const offComplete = EventsOn('export:complete', (data: { path: string }) => {
            setArchivePath(data.path);
            setStage('complete');
        });
        const offError = EventsOn('export:error', (data: { message: string }) => {
            setExportError(data.message);
            setStage('error');
        });
        return () => { offProgress(); offComplete(); offError(); };
    }, []);

    const handleExport = async () => {
        setValidationError('');
        if (password.length < 8) {
            setValidationError('Password must be at least 8 characters.');
            return;
        }
        if (password !== confirm) {
            setValidationError('Passwords do not match.');
            return;
        }

        // Prompt for save location before starting the background export.
        const pad = (n: number) => String(n).padStart(2, '0');
        const now = new Date();
        const ts = `${now.getUTCFullYear()}${pad(now.getUTCMonth() + 1)}${pad(now.getUTCDate())}T${pad(now.getUTCHours())}${pad(now.getUTCMinutes())}${pad(now.getUTCSeconds())}Z`;
        const defaultName = `dfnotes-go_export_${caseNumber}_${ts}.7z`;

        let archivePath: string;
        try {
            archivePath = await ChooseExportSavePath(defaultName);
        } catch (err: unknown) {
            setValidationError(String(err));
            return;
        }
        if (!archivePath) return; // user cancelled the dialog

        setStage('progress');
        setProgressStage('starting');
        setProgressPercent(0);
        try {
            await ExportCase(caseID, password, archivePath);
        } catch (err: unknown) {
            setExportError(String(err));
            setStage('error');
        }
    };

    const strength = passwordStrength(password);

    return (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
            <div className="bg-gray-800 border border-gray-700 rounded-lg shadow-xl w-full max-w-md p-6">
                <h2 className="text-lg font-semibold text-gray-100 mb-4">Export Case</h2>

                {stage === 'password' && (
                    <div className="space-y-4">
                        <p className="text-sm text-gray-400">
                            The archive will be encrypted with AES-256. Choose a strong password.
                        </p>

                        <div>
                            <label className="block text-xs text-gray-400 mb-1">Archive password</label>
                            <input
                                type="password"
                                value={password}
                                onChange={e => setPassword(e.target.value)}
                                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
                                autoFocus
                                onKeyDown={e => e.key === 'Enter' && handleExport()}
                            />
                            {strength.label && (
                                <p className={`text-xs mt-1 ${strength.color}`}>Strength: {strength.label}</p>
                            )}
                        </div>

                        <div>
                            <label className="block text-xs text-gray-400 mb-1">Confirm password</label>
                            <input
                                type="password"
                                value={confirm}
                                onChange={e => setConfirm(e.target.value)}
                                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
                                onKeyDown={e => e.key === 'Enter' && handleExport()}
                            />
                        </div>

                        {validationError && (
                            <p className="text-sm text-red-400">{validationError}</p>
                        )}

                        <div className="flex justify-end gap-3 pt-2">
                            <button
                                onClick={onClose}
                                className="px-4 py-2 text-sm text-gray-400 hover:text-gray-200 transition-colors"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleExport}
                                disabled={!password || !confirm}
                                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded text-sm font-medium transition-colors"
                            >
                                Export
                            </button>
                        </div>
                    </div>
                )}

                {stage === 'progress' && (
                    <div className="space-y-4">
                        <div className="flex items-center gap-3">
                            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-400 flex-shrink-0" />
                            <p className="text-sm text-gray-300 capitalize">{progressStage}...</p>
                        </div>
                        <div className="w-full bg-gray-700 rounded-full h-2">
                            <div
                                className="bg-blue-500 h-2 rounded-full transition-all duration-300"
                                style={{ width: `${progressPercent}%` }}
                            />
                        </div>
                        <p className="text-xs text-gray-500 text-right">{progressPercent}%</p>
                    </div>
                )}

                {stage === 'complete' && (
                    <div className="space-y-4">
                        <div className="flex items-center gap-2 text-green-400">
                            <svg className="w-5 h-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            <span className="text-sm font-medium">Export complete</span>
                        </div>
                        <div>
                            <p className="text-xs text-gray-400 mb-1">Archive saved to:</p>
                            <p className="text-xs text-gray-300 font-mono break-all bg-gray-900 rounded p-2">{archivePath}</p>
                        </div>
                        <div className="flex justify-end">
                            <button
                                onClick={onClose}
                                className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-200 rounded text-sm transition-colors"
                            >
                                Close
                            </button>
                        </div>
                    </div>
                )}

                {stage === 'error' && (
                    <div className="space-y-4">
                        <div className="flex items-start gap-2 text-red-400">
                            <svg className="w-5 h-5 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                            </svg>
                            <div>
                                <p className="text-sm font-medium">Export failed</p>
                                <p className="text-xs text-red-300 mt-1">{exportError}</p>
                            </div>
                        </div>
                        <div className="flex justify-end">
                            <button
                                onClick={onClose}
                                className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-200 rounded text-sm transition-colors"
                            >
                                Close
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
