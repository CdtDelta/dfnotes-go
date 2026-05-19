import { useState, useEffect } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { SetupIdentity, ConfirmTOTPSetup, InitializeDatabase, GetDefaultDBPath, ChooseDBSavePath, PointDatabase } from '../../wailsjs/go/main/App';
import { ClipboardSetText } from '../../wailsjs/runtime/runtime';
import { services } from '../../wailsjs/go/models';
import { useAuth } from '../context/AuthContext';
import ErrorMessage from '../components/ErrorMessage';
import PasswordInput from '../components/PasswordInput';

type Step = 'dbpath' | 'identity' | 'password' | 'totp' | 'recovery';

export default function SetupWizard() {
    const { setSetupComplete } = useAuth();
    const [step, setStep] = useState<Step>('dbpath');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [showFileExistsPrompt, setShowFileExistsPrompt] = useState(false);

    // DB path state
    const [dbPath, setDbPath] = useState('');

    // Form state
    const [name, setName] = useState('');
    const [organization, setOrganization] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [enableTOTP, setEnableTOTP] = useState(false);

    // TOTP state
    const [totpURL, setTotpURL] = useState('');
    const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
    const [totpCode, setTotpCode] = useState('');
    const [setupResponse, setSetupResponse] = useState<services.SetupResponse | null>(null);

    const [showManual, setShowManual] = useState(false);
    const [savedCodes, setSavedCodes] = useState(false);

    // Derived from totpURL -- extracted here so both the display and Copy button can reference it
    const totpSecret = (() => {
        try { return new URL(totpURL).searchParams.get('secret') || ''; }
        catch { return ''; }
    })();
    const totpAccount = (() => {
        try { return new URL(totpURL).pathname.split(':').pop() || name; }
        catch { return name; }
    })();

    useEffect(() => {
        GetDefaultDBPath()
            .then(setDbPath)
            .catch(() => {});
    }, []);

    const handleBrowseDBPath = async () => {
        try {
            const chosen = await ChooseDBSavePath();
            if (chosen) setDbPath(chosen);
        } catch (err: unknown) {
            setError(String(err));
        }
    };

    const handleDBPathNext = async () => {
        setError('');
        setShowFileExistsPrompt(false);
        if (!dbPath.trim()) {
            setError('Database path is required');
            return;
        }
        setLoading(true);
        try {
            await InitializeDatabase(dbPath.trim());
            setStep('identity');
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : String(err);
            if (msg.includes('FILE_EXISTS')) {
                setShowFileExistsPrompt(true);
            } else {
                setError(msg);
            }
        } finally {
            setLoading(false);
        }
    };

    const handleOpenExisting = async () => {
        setError('');
        setLoading(true);
        try {
            await PointDatabase(dbPath.trim());
            setStep('identity');
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
            setShowFileExistsPrompt(false);
        } finally {
            setLoading(false);
        }
    };

    const handleIdentityNext = () => {
        setError('');
        if (!name.trim()) {
            setError('Name is required');
            return;
        }
        setStep('password');
    };

    const handlePasswordNext = async () => {
        setError('');
        if (!password) {
            setError('Password is required');
            return;
        }
        if (password.length < 8) {
            setError('Password must be at least 8 characters');
            return;
        }
        if (password !== confirmPassword) {
            setError('Passwords do not match');
            return;
        }

        if (enableTOTP) {
            setLoading(true);
            try {
                const resp = await SetupIdentity({
                    name: name.trim(),
                    organization: organization.trim(),
                    password,
                    enable_totp: true,
                });
                setSetupResponse(resp);
                setTotpURL(resp.totp_url || '');
                setRecoveryCodes(resp.recovery_codes || []);
                setStep('totp');
            } catch (err: unknown) {
                setError(err instanceof Error ? err.message : String(err));
            } finally {
                setLoading(false);
            }
        } else {
            setLoading(true);
            try {
                const resp = await SetupIdentity({
                    name: name.trim(),
                    organization: organization.trim(),
                    password,
                    enable_totp: false,
                });
                setSetupComplete({
                    user_id: resp.user_id,
                    name: name.trim(),
                    organization: organization.trim(),
                    totp_enabled: false,
                });
            } catch (err: unknown) {
                setError(err instanceof Error ? err.message : String(err));
            } finally {
                setLoading(false);
            }
        }
    };

    const handleTOTPVerify = async () => {
        setError('');
        if (!totpCode) {
            setError('Enter the code from your authenticator app');
            return;
        }
        setLoading(true);
        try {
            const valid = await ConfirmTOTPSetup(totpCode);
            if (valid) {
                setStep('recovery');
            } else {
                setError('Invalid code. Please try again.');
            }
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    };

    const handleFinish = () => {
        if (!savedCodes) {
            setError('Please confirm you have saved the recovery codes');
            return;
        }
        if (setupResponse) {
            setSetupComplete({
                user_id: setupResponse.user_id,
                name: name.trim(),
                organization: organization.trim(),
                totp_enabled: true,
            });
        }
    };

    const allSteps: Step[] = ['dbpath', 'identity', 'password', ...(enableTOTP ? (['totp', 'recovery'] as Step[]) : [])];

    return (
        <div className="min-h-screen flex items-center justify-center p-4">
            <div className="w-full max-w-md">
                <h1 className="text-2xl font-bold text-center mb-2">dfnotes-go</h1>
                <p className="text-gray-400 text-center mb-8">Digital Forensic Notebook</p>

                {/* Step indicators */}
                <div className="flex justify-center mb-8 gap-2">
                    {allSteps.map((s) => (
                        <div
                            key={s}
                            className={`h-2 w-8 rounded-full ${
                                s === step ? 'bg-blue-500' :
                                allSteps.indexOf(s) < allSteps.indexOf(step) ? 'bg-blue-700' : 'bg-gray-700'
                            }`}
                        />
                    ))}
                </div>

                <ErrorMessage message={error} onDismiss={() => setError('')} />

                {step === 'dbpath' && (
                    <div className="space-y-4">
                        <h2 className="text-lg font-semibold">Database Location</h2>
                        <p className="text-sm text-gray-400">
                            Choose where to store your case database. The default location is recommended.
                        </p>
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Database path</label>
                            <div className="flex gap-2">
                                <input
                                    type="text"
                                    value={dbPath}
                                    onChange={(e) => setDbPath(e.target.value)}
                                    className="flex-1 bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none font-mono text-sm"
                                    autoFocus
                                />
                                <button
                                    onClick={handleBrowseDBPath}
                                    className="px-3 py-2 bg-gray-700 hover:bg-gray-600 border border-gray-600 rounded text-sm text-gray-200 transition-colors"
                                >
                                    Browse
                                </button>
                            </div>
                        </div>
                        {showFileExistsPrompt ? (
                            <div className="border border-yellow-600 bg-yellow-900 rounded-lg p-4 space-y-3">
                                <p className="text-sm text-yellow-200 font-medium">A database file already exists at this path.</p>
                                <p className="text-xs text-yellow-300">Do you want to open the existing database, or choose a different path?</p>
                                <div className="flex gap-2">
                                    <button
                                        onClick={handleOpenExisting}
                                        disabled={loading}
                                        className="flex-1 px-3 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded text-sm font-medium transition-colors"
                                    >
                                        {loading ? 'Opening...' : 'Open Existing Database'}
                                    </button>
                                    <button
                                        onClick={() => setShowFileExistsPrompt(false)}
                                        disabled={loading}
                                        className="flex-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 disabled:opacity-50 text-gray-200 rounded text-sm transition-colors"
                                    >
                                        Choose Different Path
                                    </button>
                                </div>
                            </div>
                        ) : (
                            <button
                                onClick={handleDBPathNext}
                                disabled={loading || !dbPath.trim()}
                                className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white py-2 px-4 rounded transition-colors"
                            >
                                {loading ? 'Creating...' : 'Next'}
                            </button>
                        )}
                    </div>
                )}

                {step === 'identity' && (
                    <div className="space-y-4">
                        <h2 className="text-lg font-semibold">Create Your Identity</h2>
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Name *</label>
                            <input
                                type="text"
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                                placeholder="Your name"
                                autoFocus
                            />
                        </div>
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Organization</label>
                            <input
                                type="text"
                                value={organization}
                                onChange={(e) => setOrganization(e.target.value)}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                                placeholder="Your organization"
                            />
                        </div>
                        <button
                            onClick={handleIdentityNext}
                            className="w-full bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded transition-colors"
                        >
                            Next
                        </button>
                    </div>
                )}

                {step === 'password' && (
                    <div className="space-y-4">
                        <h2 className="text-lg font-semibold">Set Password</h2>
                        <p className="text-sm text-gray-400">
                            This password protects your signing key and encrypts sensitive data.
                        </p>
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Password *</label>
                            <PasswordInput
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                                placeholder="Minimum 8 characters"
                                autoFocus
                                showPaste
                            />
                        </div>
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Confirm Password *</label>
                            <PasswordInput
                                value={confirmPassword}
                                onChange={(e) => setConfirmPassword(e.target.value)}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                                placeholder="Re-enter password"
                            />
                        </div>
                        <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
                            <input
                                type="checkbox"
                                checked={enableTOTP}
                                onChange={(e) => setEnableTOTP(e.target.checked)}
                                className="rounded bg-gray-800 border-gray-600"
                            />
                            Enable two-factor authentication (TOTP)
                        </label>
                        <div className="flex gap-2">
                            <button
                                onClick={() => setStep('identity')}
                                className="flex-1 bg-gray-700 hover:bg-gray-600 text-white py-2 px-4 rounded transition-colors"
                            >
                                Back
                            </button>
                            <button
                                onClick={handlePasswordNext}
                                disabled={loading}
                                className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 disabled:cursor-not-allowed text-white py-2 px-4 rounded transition-colors"
                            >
                                {loading ? 'Setting up...' : enableTOTP ? 'Next' : 'Create Identity'}
                            </button>
                        </div>
                    </div>
                )}

                {step === 'totp' && (
                    <div className="space-y-4">
                        <h2 className="text-lg font-semibold">Set Up Two-Factor Auth</h2>

                        {!showManual ? (
                            <>
                                <p className="text-sm text-gray-400">
                                    Scan this QR code with your authenticator app, then enter the code to verify.
                                </p>
                                {totpURL && (
                                    <div className="flex justify-center bg-white p-4 rounded">
                                        <QRCodeSVG value={totpURL} size={200} />
                                    </div>
                                )}
                                <button
                                    type="button"
                                    onClick={() => setShowManual(true)}
                                    className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
                                >
                                    Can't scan? Enter key manually
                                </button>
                            </>
                        ) : (
                            <>
                                <p className="text-sm text-gray-400">
                                    Enter this key manually in your authenticator app.
                                </p>
                                <div className="bg-gray-800 border border-gray-600 rounded p-3">
                                    <p className="text-xs text-gray-400 mb-1">Account</p>
                                    <p className="text-sm text-gray-200 mb-3 break-all">
                                        {totpAccount}
                                    </p>
                                    <p className="text-xs text-gray-400 mb-1">Secret Key</p>
                                    <div className="flex items-start gap-2">
                                        <p className="font-mono text-sm text-gray-100 break-all select-all flex-1">
                                            {totpSecret}
                                        </p>
                                        <button
                                            type="button"
                                            onClick={() => ClipboardSetText(totpSecret)}
                                            className="text-xs px-2 py-0.5 rounded shrink-0 hover:opacity-75 transition-opacity"
                                            style={{ color: 'var(--text-secondary)', border: '1px solid var(--border-primary)' }}
                                        >
                                            Copy
                                        </button>
                                    </div>
                                    <p className="text-xs text-gray-400 mt-3 mb-1">Type</p>
                                    <p className="text-sm text-gray-200">Time-based (TOTP)</p>
                                </div>
                                <button
                                    type="button"
                                    onClick={() => setShowManual(false)}
                                    className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
                                >
                                    Show QR code instead
                                </button>
                            </>
                        )}

                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Verification Code</label>
                            <input
                                type="text"
                                value={totpCode}
                                onChange={(e) => setTotpCode(e.target.value)}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none text-center text-lg tracking-widest"
                                placeholder="000000"
                                maxLength={6}
                                autoFocus
                            />
                        </div>
                        <button
                            onClick={handleTOTPVerify}
                            disabled={loading}
                            className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 disabled:cursor-not-allowed text-white py-2 px-4 rounded transition-colors"
                        >
                            {loading ? 'Verifying...' : 'Verify'}
                        </button>
                    </div>
                )}

                {step === 'recovery' && (
                    <div className="space-y-4">
                        <h2 className="text-lg font-semibold">Recovery Codes</h2>
                        <p className="text-sm text-gray-400">
                            Save these codes in a safe place. Each code can be used once if you lose access to your authenticator.
                        </p>
                        <div className="bg-gray-800 border border-gray-600 rounded p-4 grid grid-cols-2 gap-2 font-mono text-sm">
                            {recoveryCodes.map((code, i) => (
                                <div key={i} className="text-gray-200">{code}</div>
                            ))}
                        </div>
                        <div className="flex justify-end">
                            <button
                                type="button"
                                onClick={() => ClipboardSetText(recoveryCodes.join('\n'))}
                                className="text-xs px-2 py-0.5 rounded hover:opacity-75 transition-opacity"
                                style={{ color: 'var(--text-secondary)', border: '1px solid var(--border-primary)' }}
                            >
                                Copy All
                            </button>
                        </div>
                        <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
                            <input
                                type="checkbox"
                                checked={savedCodes}
                                onChange={(e) => setSavedCodes(e.target.checked)}
                                className="rounded bg-gray-800 border-gray-600"
                            />
                            I have saved these recovery codes
                        </label>
                        <button
                            onClick={handleFinish}
                            className="w-full bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded transition-colors"
                        >
                            Finish Setup
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
}
