import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ListCases } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';
import CaseCard from '../components/CaseCard';
import ErrorMessage from '../components/ErrorMessage';

export default function DashboardPage() {
    const navigate = useNavigate();
    const [cases, setCases] = useState<services.CaseResponse[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    useEffect(() => {
        ListCases()
            .then((result) => {
                setCases(result || []);
            })
            .catch((err: unknown) => {
                setError(err instanceof Error ? err.message : String(err));
            })
            .finally(() => setLoading(false));
    }, []);

    return (
        <div className="min-h-screen p-6">
            <div>
                <div className="flex items-center justify-between mb-8">
                    <div>
                        <h1 className="text-2xl font-bold">Cases</h1>
                        <p className="text-sm text-gray-400">Digital Forensic Notebook</p>
                    </div>
                    <button
                        onClick={() => navigate('/cases/new')}
                        className="bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded transition-colors"
                    >
                        New Case
                    </button>
                </div>

                <ErrorMessage message={error} onDismiss={() => setError('')} />

                {loading ? (
                    <div className="flex justify-center py-12">
                        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-400"></div>
                    </div>
                ) : cases.length === 0 ? (
                    <div className="text-center py-16">
                        <p className="text-gray-400 text-lg mb-2">No cases yet</p>
                        <p className="text-gray-500 text-sm mb-6">
                            Create your first forensic case to get started.
                        </p>
                        <button
                            onClick={() => navigate('/cases/new')}
                            className="bg-blue-600 hover:bg-blue-700 text-white py-2 px-6 rounded transition-colors"
                        >
                            Create First Case
                        </button>
                    </div>
                ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        {cases.map((c) => (
                            <CaseCard key={c.case_id} caseData={c} />
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
}
