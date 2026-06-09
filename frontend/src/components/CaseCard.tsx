import { useNavigate } from 'react-router-dom';
import { services } from '../../wailsjs/go/models';
import ClassificationBadge from './ClassificationBadge';

interface CaseCardProps {
    caseData: services.CaseResponse;
}

export default function CaseCard({ caseData }: CaseCardProps) {
    const navigate = useNavigate();
    const createdDate = new Date(caseData.created_at).toLocaleDateString();

    return (
        <div
            className="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-blue-500 transition-colors cursor-pointer"
            onClick={() => navigate(`/cases/${caseData.case_id}`)}
        >
            <div className="flex items-start justify-between mb-2">
                <span className="text-sm text-gray-400 font-mono">{caseData.case_number}</span>
                <div className="flex items-center gap-1.5">
                    {caseData.attorney_client_privilege && (
                        <span className="inline-block px-1.5 py-0.5 rounded text-xs font-medium bg-amber-800 text-amber-200">
                            Privileged
                        </span>
                    )}
                    <ClassificationBadge level={caseData.classification} />
                </div>
            </div>
            <h3 className="text-lg font-semibold text-gray-100 mb-1">{caseData.title}</h3>
            {caseData.description && (
                <p className="text-sm text-gray-400 mb-3 line-clamp-2">{caseData.description}</p>
            )}
            <div className="flex items-center justify-between text-xs text-gray-500">
                <span>{caseData.examiner_name}</span>
                <span>{createdDate}</span>
            </div>
            {caseData.ticket_number && (
                <div className="mt-2 text-xs text-gray-500">
                    Ticket: {caseData.ticket_number}
                </div>
            )}
        </div>
    );
}
