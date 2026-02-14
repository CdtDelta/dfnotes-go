import { useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { SetupIdentity, ConfirmTOTPSetup } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';
import { useAuth } from '../context/AuthContext';
import ErrorMessage from '../components/ErrorMessage';

type Step = 'identity' | 'password' | 'totp' | 'recovery';

export default function SetupWizard() {
    const { setSetupComplete } = useAuth();
    const [step, setStep] = useState<Step>('identity');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

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

    // Manual setup toggle
    const [showManual, setShowManual] = useState(false);

    // Recovery state
    const [savedCodes, setSavedCodes] = useState(false);

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
            // Call setup and show TOTP step
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
            // No TOTP: setup and authenticate
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

    return (
        <div className="min-h-screen flex items-center justify-center p-4">
            <div className="w-full max-w-md">
                <h1 className="text-2xl font-bold text-center mb-2">dfnotes-go</h1>
                <p className="text-gray-400 text-center mb-8">Digital Forensic Notebook</p>

                {/* Step indicators */}
                <div className="flex justify-center mb-8 gap-2">
                    {(['identity', 'password', ...(enableTOTP ? ['totp', 'recovery'] : [])] as Step[]).map((s) => (
                        <div
                            key={s}
                            className={`h-2 w-8 rounded-full ${
                                s === step ? 'bg-blue-500' :
                                (['identity', 'password', 'totp', 'recovery'].indexOf(s) < ['identity', 'password', 'totp', 'recovery'].indexOf(step)) ? 'bg-blue-700' : 'bg-gray-700'
                            }`}
                        />
                    ))}
                </div>

                <ErrorMessage message={error} onDismiss={() => setError('')} />

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
                            <input
                                type="password"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                                placeholder="Minimum 8 characters"
                                autoFocus
                            />
                        </div>
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Confirm Password *</label>
                            <input
                                type="password"
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
                                        {(() => {
                                            try { return new URL(totpURL).pathname.split(':').pop() || name; }
                                            catch { return name; }
                                        })()}
                                    </p>
                                    <p className="text-xs text-gray-400 mb-1">Secret Key</p>
                                    <p className="font-mono text-sm text-gray-100 break-all select-all">
                                        {(() => {
                                            try { return new URL(totpURL).searchParams.get('secret') || ''; }
                                            catch { return ''; }
                                        })()}
                                    </p>
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
