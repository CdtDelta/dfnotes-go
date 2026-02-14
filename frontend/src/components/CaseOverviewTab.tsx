import { services } from '../../wailsjs/go/models';
import ClassificationBadge from './ClassificationBadge';

interface CaseOverviewTabProps {
    caseData: services.CaseResponse;
}

export default function CaseOverviewTab({ caseData }: CaseOverviewTabProps) {
    const createdDate = new Date(caseData.created_at).toLocaleString();

    return (
        <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Case Number</label>
                    <p className="text-gray-100 font-mono mt-1">{caseData.case_number}</p>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Ticket Number</label>
                    <p className="text-gray-100 mt-1">{caseData.ticket_number || '—'}</p>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Examiner</label>
                    <p className="text-gray-100 mt-1">{caseData.examiner_name}</p>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Organization</label>
                    <p className="text-gray-100 mt-1">{caseData.organization}</p>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Classification</label>
                    <div className="mt-1">
                        <ClassificationBadge level={caseData.classification} />
                    </div>
                </div>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Created</label>
                    <p className="text-gray-100 mt-1">{createdDate}</p>
                </div>
            </div>
            {caseData.description && (
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <label className="text-xs text-gray-500 uppercase tracking-wide">Description</label>
                    <p className="text-gray-300 mt-2 whitespace-pre-wrap">{caseData.description}</p>
                </div>
            )}
        </div>
    );
}
