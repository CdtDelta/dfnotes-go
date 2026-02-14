import { services } from '../../wailsjs/go/models';
import TagBadge from './TagBadge';

interface EvidenceCardProps {
    item: services.EvidenceResponse;
    onSelect: (evidenceItemID: string) => void;
}

const TYPE_COLORS: Record<string, string> = {
    DISK: 'bg-blue-900/50 text-blue-300 border-blue-700',
    MEMORY: 'bg-purple-900/50 text-purple-300 border-purple-700',
    NETWORK: 'bg-cyan-900/50 text-cyan-300 border-cyan-700',
    LOGS: 'bg-yellow-900/50 text-yellow-300 border-yellow-700',
    MALWARE: 'bg-red-900/50 text-red-300 border-red-700',
    OTHER: 'bg-gray-800/50 text-gray-300 border-gray-600',
};

const STATUS_COLORS: Record<string, string> = {
    COLLECTED: 'bg-green-900/50 text-green-300 border-green-700',
    ANALYZING: 'bg-yellow-900/50 text-yellow-300 border-yellow-700',
    PROCESSED: 'bg-blue-900/50 text-blue-300 border-blue-700',
    ARCHIVED: 'bg-gray-800/50 text-gray-400 border-gray-600',
    WITHDRAWN: 'bg-red-900/50 text-red-400 border-red-700',
};

export default function EvidenceCard({ item, onSelect }: EvidenceCardProps) {
    const isWithdrawn = item.status === 'WITHDRAWN';
    const typeColor = TYPE_COLORS[item.evidence_type] || TYPE_COLORS.OTHER;
    const statusColor = STATUS_COLORS[item.status] || STATUS_COLORS.COLLECTED;
    const tags = item.tags || [];

    return (
        <button
            onClick={() => onSelect(item.evidence_item_id)}
            className={`w-full text-left bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-gray-500 transition-colors ${
                isWithdrawn ? 'opacity-50' : ''
            }`}
        >
            <div className="flex items-start justify-between mb-2">
                <h4 className="text-sm font-semibold text-gray-100 truncate mr-2">{item.name}</h4>
                <div className="flex items-center gap-1.5 shrink-0">
                    <span className={`text-xs px-1.5 py-0.5 rounded border ${typeColor}`}>
                        {item.evidence_type}
                    </span>
                    <span className={`text-xs px-1.5 py-0.5 rounded border ${statusColor}`}>
                        {item.status}
                    </span>
                </div>
            </div>
            {tags.length > 0 && (
                <div className="flex items-center gap-1 flex-wrap mb-2">
                    {tags.map((tag) => (
                        <TagBadge key={tag.tag_id} tag={tag} />
                    ))}
                </div>
            )}
            {item.content_hash && (
                <p className="text-xs text-gray-500 font-mono truncate mb-1">
                    {item.content_hash.substring(0, 16)}...
                </p>
            )}
            <p className="text-xs text-gray-400">
                Collected {new Date(item.collected_at).toLocaleDateString()}
            </p>
        </button>
    );
}
