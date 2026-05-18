import { useState } from 'react';
import { ChooseDBOpenPath, PointDatabase, GetConfig, SaveConfig } from '../../wailsjs/go/main/App';
import { useAuth } from '../context/AuthContext';

export default function DBMissingPage() {
    const { missingPath, resetToLogin } = useAuth();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const handleLocateFile = async () => {
        setError('');
        setLoading(true);
        try {
            const selected = await ChooseDBOpenPath();
            if (!selected) return;
            await PointDatabase(selected);
            await resetToLogin();
        } catch (err: unknown) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    const handleCreateNew = async () => {
        setError('');
        setLoading(true);
        try {
            const cfg = await GetConfig();
            await SaveConfig({ ...cfg, database_path: '' });
            window.location.reload();
        } catch (err: unknown) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center p-4">
            <div className="w-full max-w-md">
                <div className="flex items-center gap-3 mb-6">
                    <div className="w-10 h-10 rounded-full bg-red-900 flex items-center justify-center flex-shrink-0">
                        <svg className="w-5 h-5 text-red-400" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                        </svg>
                    </div>
                    <div>
                        <h1 className="text-lg font-semibold text-gray-100">Database Not Found</h1>
                        <p className="text-sm text-gray-400">The configured database file is missing</p>
                    </div>
                </div>

                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4 mb-6">
                    <p className="text-xs text-gray-400 mb-1">Expected location</p>
                    <p className="text-sm text-gray-300 font-mono break-all">{missingPath || 'Unknown path'}</p>
                    <p className="text-xs text-gray-500 mt-2">
                        The file may have been moved, deleted, or is on a disconnected drive.
                    </p>
                </div>

                {error && (
                    <div className="bg-red-900 border border-red-700 rounded-lg p-3 mb-4">
                        <p className="text-sm text-red-300">{error}</p>
                    </div>
                )}

                <div className="space-y-3">
                    <button
                        onClick={handleLocateFile}
                        disabled={loading}
                        className="w-full px-4 py-3 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg text-sm font-medium transition-colors"
                    >
                        {loading ? 'Working...' : 'Locate File...'}
                    </button>
                    <button
                        onClick={handleCreateNew}
                        disabled={loading}
                        className="w-full px-4 py-3 bg-gray-700 hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed text-gray-200 rounded-lg text-sm font-medium transition-colors"
                    >
                        Create New Database
                    </button>
                </div>
            </div>
        </div>
    );
}
