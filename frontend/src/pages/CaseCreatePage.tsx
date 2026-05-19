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
    const [casePassword, setCasePassword] = useState('');
    const [confirmCasePassword, setConfirmCasePassword] = useState('');

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

        setLoading(true);
        try {
            await CreateCase({
                case_number: caseNumber.trim(),
                title: title.trim(),
                classification,
                ticket_number: ticketNumber.trim(),
                description: description.trim(),
                case_password: casePassword,
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
