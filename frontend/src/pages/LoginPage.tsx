import { useState } from 'react';
import { Login } from '../../wailsjs/go/main/App';
import { useAuth } from '../context/AuthContext';
import ErrorMessage from '../components/ErrorMessage';

export default function LoginPage() {
    const { loginScreenInfo, setAuthenticated } = useAuth();
    const [password, setPassword] = useState('');
    const [totpCode, setTotpCode] = useState('');
    const [showTOTP, setShowTOTP] = useState(loginScreenInfo?.totp_enabled ?? false);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const handleLogin = async () => {
        setError('');
        if (!password) {
            setError('Password is required');
            return;
        }

        setLoading(true);
        try {
            const resp = await Login({
                password,
                totp_code: totpCode || undefined,
            });
            setAuthenticated(resp);
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : String(err);
            if (msg.includes('invalid password')) {
                setError('Invalid password');
            } else if (msg.includes('TOTP required')) {
                setShowTOTP(true);
                setError('Please enter your two-factor authentication code');
            } else if (msg.includes('TOTP code invalid')) {
                setError('Invalid authentication code');
            } else {
                setError(msg);
            }
        } finally {
            setLoading(false);
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') handleLogin();
    };

    return (
        <div className="min-h-screen flex items-center justify-center p-4">
            <div className="w-full max-w-sm">
                <h1 className="text-2xl font-bold text-center mb-2">dfnotes-go</h1>
                <p className="text-gray-400 text-center mb-8">Digital Forensic Notebook</p>

                {loginScreenInfo && (
                    <div className="text-center mb-6">
                        <p className="text-lg text-gray-200">{loginScreenInfo.name}</p>
                        {loginScreenInfo.organization && (
                            <p className="text-sm text-gray-400">{loginScreenInfo.organization}</p>
                        )}
                    </div>
                )}

                <ErrorMessage message={error} onDismiss={() => setError('')} />

                <div className="space-y-4">
                    <div>
                        <label className="block text-sm text-gray-400 mb-1">Password</label>
                        <input
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            onKeyDown={handleKeyDown}
                            className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                            placeholder="Enter your password"
                            autoFocus
                        />
                    </div>

                    {showTOTP && (
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">
                                Authentication Code
                            </label>
                            <input
                                type="text"
                                value={totpCode}
                                onChange={(e) => setTotpCode(e.target.value)}
                                onKeyDown={handleKeyDown}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none text-center text-lg tracking-widest"
                                placeholder="000000 or recovery code"
                                maxLength={8}
                            />
                        </div>
                    )}

                    <button
                        onClick={handleLogin}
                        disabled={loading}
                        className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 disabled:cursor-not-allowed text-white py-2 px-4 rounded transition-colors"
                    >
                        {loading ? 'Authenticating...' : 'Unlock'}
                    </button>
                </div>
            </div>
        </div>
    );
}
