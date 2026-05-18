import { useState } from 'react';
import { MoveDatabase, PointDatabase, ChooseDBSavePath, ChooseDBOpenPath } from '../../wailsjs/go/main/App';
import { useAuth } from '../context/AuthContext';
import ErrorMessage from './ErrorMessage';

interface DBLocationDialogProps {
    onClose: () => void;
    onSuccess: () => void;
}

type Option = 'move' | 'point' | null;

export default function DBLocationDialog({ onClose, onSuccess }: DBLocationDialogProps) {
    const { resetToLogin } = useAuth();
    const [selected, setSelected] = useState<Option>(null);
    const [chosenPath, setChosenPath] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [pendingReset, setPendingReset] = useState(false);

    const handleBrowseMove = async () => {
        try {
            const path = await ChooseDBSavePath();
            if (path) setChosenPath(path);
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    const handleBrowsePoint = async () => {
        try {
            const path = await ChooseDBOpenPath();
            if (path) setChosenPath(path);
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    const handleConfirm = async () => {
        if (!chosenPath || !selected) return;
        setLoading(true);
        setError('');
        try {
            if (selected === 'move') {
                await MoveDatabase(chosenPath);
            } else {
                await PointDatabase(chosenPath);
            }
            onSuccess();
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : String(err);
            if (selected === 'move') {
                setError(`Move failed: ${msg}. Your original database is unchanged.`);
                setPendingReset(true);
            } else {
                setError(msg);
            }
        } finally {
            setLoading(false);
        }
    };

    const handleSelectOption = (opt: Option) => {
        setSelected(opt);
        setChosenPath('');
        setError('');
    };

    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center"
            style={{ backgroundColor: 'rgba(0,0,0,0.6)' }}
        >
            <div className="bg-gray-900 border border-gray-700 rounded-lg shadow-xl w-full max-w-lg p-6">
                <div className="flex items-center justify-between mb-4">
                    <h2 className="text-lg font-semibold text-gray-100">Change Database Location</h2>
                    <button
                        onClick={pendingReset ? resetToLogin : onClose}
                        className="text-gray-400 hover:text-gray-200 transition-colors"
                        aria-label="Close"
                    >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>

                <p className="text-sm text-gray-400 mb-4">
                    All cases must be locked before changing the database location.
                </p>

                <ErrorMessage message={error} onDismiss={() => setError('')} />

                <div className="space-y-3 mb-6">
                    {/* Option A: Move */}
                    <button
                        onClick={() => handleSelectOption('move')}
                        className={`w-full text-left p-4 rounded-lg border transition-colors ${
                            selected === 'move'
                                ? 'border-blue-500 bg-blue-900/20'
                                : 'border-gray-700 hover:border-gray-500'
                        }`}
                    >
                        <div className="flex items-start gap-3">
                            <div className={`w-4 h-4 rounded-full border-2 mt-0.5 flex-shrink-0 ${
                                selected === 'move' ? 'border-blue-500 bg-blue-500' : 'border-gray-500'
                            }`} />
                            <div>
                                <p className="text-sm font-medium text-gray-100">Move existing database</p>
                                <p className="text-xs text-gray-400 mt-1">
                                    The current database file will be moved to a new location you choose. Your active database moves with it.
                                </p>
                            </div>
                        </div>
                    </button>

                    {/* Option B: Point */}
                    <button
                        onClick={() => handleSelectOption('point')}
                        className={`w-full text-left p-4 rounded-lg border transition-colors ${
                            selected === 'point'
                                ? 'border-blue-500 bg-blue-900/20'
                                : 'border-gray-700 hover:border-gray-500'
                        }`}
                    >
                        <div className="flex items-start gap-3">
                            <div className={`w-4 h-4 rounded-full border-2 mt-0.5 flex-shrink-0 ${
                                selected === 'point' ? 'border-blue-500 bg-blue-500' : 'border-gray-500'
                            }`} />
                            <div>
                                <p className="text-sm font-medium text-gray-100">Use a different database</p>
                                <p className="text-xs text-gray-400 mt-1">
                                    The app will open a different database file at a path you choose. The current database file is not moved or deleted.
                                </p>
                            </div>
                        </div>
                    </button>
                </div>

                {selected && (
                    <div className="mb-6">
                        <label className="block text-xs text-gray-400 mb-1">
                            {selected === 'move' ? 'Move to location' : 'Select database file'}
                        </label>
                        <div className="flex gap-2">
                            <input
                                type="text"
                                value={chosenPath}
                                onChange={(e) => setChosenPath(e.target.value)}
                                placeholder={selected === 'move' ? 'Choose destination path...' : 'Choose .db file...'}
                                className="flex-1 px-3 py-2 bg-gray-800 border border-gray-600 rounded text-sm text-gray-200 font-mono focus:outline-none focus:border-blue-500"
                                readOnly
                            />
                            <button
                                onClick={selected === 'move' ? handleBrowseMove : handleBrowsePoint}
                                className="px-3 py-2 bg-gray-700 hover:bg-gray-600 border border-gray-600 rounded text-sm text-gray-200 transition-colors"
                            >
                                Browse
                            </button>
                        </div>
                    </div>
                )}

                <div className="flex gap-3 justify-end">
                    {pendingReset ? (
                        <button
                            onClick={resetToLogin}
                            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded transition-colors"
                        >
                            Return to Login
                        </button>
                    ) : (
                        <>
                            <button
                                onClick={onClose}
                                className="px-4 py-2 text-sm text-gray-400 hover:text-gray-200 transition-colors"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleConfirm}
                                disabled={!selected || !chosenPath || loading}
                                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white text-sm rounded transition-colors"
                            >
                                {loading ? 'Working...' : 'Confirm'}
                            </button>
                        </>
                    )}
                </div>
            </div>
        </div>
    );
}
