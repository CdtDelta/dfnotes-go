import { TagBlock, UntagBlock } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';
import MarkdownRenderer from './MarkdownRenderer';
import TagBadge from './TagBadge';
import TagSelector from './TagSelector';

interface NoteBlockCardProps {
    block: services.NoteBlockResponse;
    caseId: string;
    evidenceItems?: services.EvidenceResponse[];
    onEvidenceClick?: (evidenceItemId: string) => void;
    onTagsChanged?: () => void;
}

export default function NoteBlockCard({ block, caseId, evidenceItems, onEvidenceClick, onTagsChanged }: NoteBlockCardProps) {
    const createdDate = new Date(block.created_at).toLocaleString();
    const shortHash = block.content_hash.substring(0, 12);
    const shortPrev = block.prev_hash === 'genesis'
        ? 'genesis'
        : block.prev_hash.substring(0, 12);
    const tags = block.tags || [];

    const handleTag = async (tagId: string) => {
        try {
            await TagBlock({ block_id: block.block_id, tag_id: tagId } as services.TagBlockRequest);
            onTagsChanged?.();
        } catch { /* ignore */ }
    };

    const handleUntag = async (tagId: string) => {
        try {
            await UntagBlock({ block_id: block.block_id, tag_id: tagId } as services.TagBlockRequest);
            onTagsChanged?.();
        } catch { /* ignore */ }
    };

    return (
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
            <div className="flex items-center justify-between mb-3">
                <span className="text-xs text-gray-500">{createdDate}</span>
                {block.verified ? (
                    <span className="inline-flex items-center gap-1 text-xs font-medium text-green-400">
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        Verified
                    </span>
                ) : (
                    <span className="inline-flex items-center gap-1 text-xs font-medium text-red-400">
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
                        </svg>
                        Unverified
                    </span>
                )}
            </div>
            <div className="prose prose-invert prose-sm max-w-none mb-3 break-words">
                <MarkdownRenderer
                    content={block.content}
                    caseId={caseId}
                    evidenceItems={evidenceItems}
                    onEvidenceClick={onEvidenceClick}
                />
            </div>
            {/* Tags */}
            <div className="flex items-center gap-1.5 flex-wrap mb-2">
                {tags.map((tag) => (
                    <TagBadge key={tag.tag_id} tag={tag} onRemove={() => handleUntag(tag.tag_id)} />
                ))}
                <TagSelector currentTags={tags} onTag={handleTag} onUntag={handleUntag} />
            </div>
            <div className="flex items-center gap-4 text-xs text-gray-600 font-mono border-t border-gray-700 pt-2">
                <span title={block.content_hash}>hash: {shortHash}...</span>
                <span title={block.prev_hash}>prev: {shortPrev}{block.prev_hash !== 'genesis' && '...'}</span>
            </div>
        </div>
    );
}
