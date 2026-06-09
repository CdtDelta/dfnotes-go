import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { CreateCase, GetUserInfo } from '../../wailsjs/go/main/App';
import ErrorMessage from '../components/ErrorMessage';
import PasswordInput from '../components/PasswordInput';

const CLASSIFICATIONS = [
    'UNCLASSIFIED',
    'CONFIDENTIAL',
    'SECRET',
    'TOP SECRET',
];

export default function CaseCreatePage() {
    const navigate = useNavigate();
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const [caseNumber, setCaseNumber] = useState('');
    const [title, setTitle] = useState('');
    const [examinerName, setExaminerName] = useState('');
    const [organization, setOrganization] = useState('');
    const [classification, setClassification] = useState('UNCLASSIFIED');
    const [ticketNumber, setTicketNumber] = useState('');
    const [description, setDescription] = useState('');
    const [evidencePrefix, setEvidencePrefix] = useState('E');
    const [evidenceSeqDigits, setEvidenceSeqDigits] = useState(3);
    const [prefixError, setPrefixError] = useState('');
    const [casePassword, setCasePassword] = useState('');
    const [confirmCasePassword, setConfirmCasePassword] = useState('');
    const [attorneyClientPrivilege, setAttorneyClientPrivilege] = useState(false);

    const PREFIX_REGEX = /^[A-Za-z0-9_-]+$/;

    const handlePrefixChange = (value: string) => {
        setEvidencePrefix(value);
        if (value && !PREFIX_REGEX.test(value)) {
            setPrefixError('Alphanumeric, hyphens, and underscores only.');
        } else {
            setPrefixError('');
        }
    };

    useEffect(() => {
        GetUserInfo()
            .then((info) => {
                setExaminerName(info.name);
                setOrganization(info.organization);
            })
            .catch(() => {
                // Non-critical: fields remain empty for manual entry
            });
    }, []);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError('');

        if (!caseNumber.trim()) {
            setError('Case number is required');
            return;
        }
        if (!title.trim()) {
            setError('Case title is required');
            return;
        }
        if (!casePassword) {
            setError('Case password is required');
            return;
        }
        if (casePassword.length < 8) {
            setError('Case password must be at least 8 characters');
            return;
        }
        if (casePassword !== confirmCasePassword) {
            setError('Passwords do not match');
            return;
        }
        if (!evidencePrefix || !PREFIX_REGEX.test(evidencePrefix)) {
            setError('Evidence prefix must contain only letters, digits, hyphens, and underscores');
            return;
        }

        setLoading(true);
        try {
            await CreateCase({
                case_number: caseNumber.trim(),
                title: title.trim(),
                classification,
                ticket_number: ticketNumber.trim(),
                description: description.trim(),
                case_password: casePassword,
                evidence_prefix: evidencePrefix,
                evidence_seq_digits: evidenceSeqDigits,
                attorney_client_privilege: attorneyClientPrivilege,
            });
            navigate('/');
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen p-6">
            <div className="max-w-2xl mx-auto">
                <div className="flex items-center justify-between mb-8">
                    <h1 className="text-2xl font-bold">New Case</h1>
                    <button
                        onClick={() => navigate('/')}
                        className="text-gray-400 hover:text-gray-200 transition-colors"
                    >
                        Cancel
                    </button>
                </div>

                <ErrorMessage message={error} onDismiss={() => setError('')} />

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Case Number *</label>
                            <input
                                type="text"
                                value={caseNumber}
                                onChange={(e) => setCaseNumber(e.target.value)}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                                placeholder="e.g. CASE-2024-001"
                                autoFocus
                            />
                        </div>
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Ticket Number</label>
                            <input
                                type="text"
                                value={ticketNumber}
                                onChange={(e) => setTicketNumber(e.target.value)}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                                placeholder="Optional"
                            />
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm text-gray-400 mb-1">Case Title *</label>
                        <input
                            type="text"
                            value={title}
                            onChange={(e) => setTitle(e.target.value)}
                            className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                            placeholder="Brief case title"
                        />
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Examiner Name</label>
                            <input
                                type="text"
                                value={examinerName}
                                onChange={(e) => setExaminerName(e.target.value)}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                            />
                        </div>
                        <div>
                            <label className="block text-sm text-gray-400 mb-1">Organization</label>
                            <input
                                type="text"
                                value={organization}
                                onChange={(e) => setOrganization(e.target.value)}
                                className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                            />
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm text-gray-400 mb-1">Classification Level</label>
                        <select
                            value={classification}
                            onChange={(e) => setClassification(e.target.value)}
                            className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                        >
                            {CLASSIFICATIONS.map((c) => (
                                <option key={c} value={c}>{c}</option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm text-gray-400 mb-1">Description</label>
                        <textarea
                            value={description}
                            onChange={(e) => setDescription(e.target.value)}
                            rows={3}
                            className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none resize-y"
                            placeholder="Optional case summary"
                        />
                    </div>

                    <div className="border-t border-gray-700 pt-4">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                            <div>
                                <label className="block text-sm text-gray-400 mb-1">Evidence prefix</label>
                                <input
                                    type="text"
                                    value={evidencePrefix}
                                    onChange={(e) => handlePrefixChange(e.target.value)}
                                    className={`w-full bg-gray-800 border rounded px-3 py-2 text-gray-100 focus:outline-none font-mono ${prefixError ? 'border-red-500 focus:border-red-500' : 'border-gray-600 focus:border-blue-500'}`}
                                    placeholder="E"
                                />
                                {prefixError
                                    ? <p className="text-xs text-red-400 mt-1">{prefixError}</p>
                                    : <p className="text-xs text-gray-500 mt-1">Added before each evidence number. Alphanumeric, hyphens, and underscores only.</p>
                                }
                            </div>
                            <div>
                                <label className="block text-sm text-gray-400 mb-1">Sequence digits</label>
                                <input
                                    type="number"
                                    value={evidenceSeqDigits}
                                    onChange={(e) => setEvidenceSeqDigits(Math.min(6, Math.max(1, parseInt(e.target.value) || 1)))}
                                    min={1}
                                    max={6}
                                    className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                                />
                                <p className="text-xs text-gray-500 mt-1">Zero-padded length of the sequence number (e.g. 3 produces 001, 002...).</p>
                            </div>
                        </div>
                        {evidencePrefix && !prefixError && (
                            <p className="text-xs text-gray-400 mb-4 font-mono">
                                Evidence items will be numbered:{' '}
                                {evidencePrefix}{String(1).padStart(evidenceSeqDigits, '0')},{' '}
                                {evidencePrefix}{String(2).padStart(evidenceSeqDigits, '0')}...
                            </p>
                        )}
                    </div>

                    <div className="border-t border-gray-700 pt-4">
                        <p className="text-sm text-gray-400 mb-3">
                            The case password is used for per-case encryption key derivation.
                        </p>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div>
                                <label className="block text-sm text-gray-400 mb-1">Case Password *</label>
                                <PasswordInput
                                    value={casePassword}
                                    onChange={(e) => setCasePassword(e.target.value)}
                                    className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                                    placeholder="Minimum 8 characters"
                                    showPaste
                                />
                            </div>
                            <div>
                                <label className="block text-sm text-gray-400 mb-1">Confirm Password *</label>
                                <PasswordInput
                                    value={confirmCasePassword}
                                    onChange={(e) => setConfirmCasePassword(e.target.value)}
                                    className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
                                    placeholder="Re-enter password"
                                />
                            </div>
                        </div>
                    </div>

                    <div className="border-t border-gray-700 pt-4">
                        <label className="flex items-start gap-3 cursor-pointer select-none">
                            <input
                                type="checkbox"
                                checked={attorneyClientPrivilege}
                                onChange={(e) => setAttorneyClientPrivilege(e.target.checked)}
                                className="mt-0.5 w-4 h-4 rounded border-gray-600 bg-gray-800 text-amber-500 focus:ring-amber-500 focus:ring-offset-gray-900 cursor-pointer"
                            />
                            <div>
                                <span className="text-sm text-gray-200">Attorney-Client Privilege</span>
                                <p className="text-xs text-gray-500 mt-0.5">Mark this case as subject to attorney-client privilege.</p>
                            </div>
                        </label>
                    </div>

                    <div className="flex gap-2 pt-2">
                        <button
                            type="button"
                            onClick={() => navigate('/')}
                            className="flex-1 bg-gray-700 hover:bg-gray-600 text-white py-2 px-4 rounded transition-colors"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={loading}
                            className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 disabled:cursor-not-allowed text-white py-2 px-4 rounded transition-colors"
                        >
                            {loading ? 'Creating...' : 'Create Case'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}
