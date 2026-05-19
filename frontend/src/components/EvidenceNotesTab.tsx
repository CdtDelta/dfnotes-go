import { useState, useEffect, useCallback } from 'react';
import { CommitNote, ListEvidenceNotes } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';
import NoteBlockCard from './NoteBlockCard';
import NoteEditor from './NoteEditor';
import TagFilter from './TagFilter';
import ErrorMessage from './ErrorMessage';

interface EvidenceNotesTabProps {
    caseId: string;
    evidenceItemId: string;
    evidenceItems?: services.EvidenceResponse[];
    onEvidenceClick?: (evidenceItemId: string) => void;
    onNavigateToTask?: (taskId: string) => void;
}

export default function EvidenceNotesTab({ caseId, evidenceItemId, evidenceItems, onEvidenceClick, onNavigateToTask }: EvidenceNotesTabProps) {
    const [content, setContent] = useState('');
    const [notes, setNotes] = useState<services.NoteBlockResponse[]>([]);
    const [loading, setLoading] = useState(true);
    const [committing, setCommitting] = useState(false);
    const [error, setError] = useState('');
    const [filterTagIds, setFilterTagIds] = useState<string[]>([]);

    const fetchNotes = useCallback(() => {
        setLoading(true);
        ListEvidenceNotes(caseId, evidenceItemId)
            .then((result) => setNotes(result || []))
            .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
            .finally(() => setLoading(false));
    }, [caseId, evidenceItemId]);

    useEffect(() => {
        fetchNotes();
    }, [fetchNotes]);

    const handleCommit = async () => {
        if (!content.trim()) return;
        setCommitting(true);
        setError('');
        try {
            await CommitNote({
                case_id: caseId,
                content: content.trim(),
                evidence_item_id: evidenceItemId,
            } as services.CommitNoteRequest);
            setContent('');
            fetchNotes();
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setCommitting(false);
        }
    };

    const reversedNotes = [...notes].reverse();
    const filteredNotes = filterTagIds.length > 0
        ? reversedNotes.filter((block) =>
            (block.tags || []).some((tag) => filterTagIds.includes(tag.tag_id))
        )
        : reversedNotes;

    return (
        <div className="space-y-6">
            {/* Editor Section */}
            <NoteEditor
                caseId={caseId}
                content={content}
                onChange={setContent}
                evidenceItems={evidenceItems}
                onEvidenceClick={onEvidenceClick}
            />

            <div className="flex justify-end">
                <button
                    onClick={handleCommit}
                    disabled={committing || !content.trim()}
                    className="bg-green-600 hover:bg-green-700 disabled:bg-gray-700 disabled:text-gray-500 text-white py-2 px-4 rounded text-sm transition-colors"
                >
                    {committing ? 'Committing...' : 'Commit Note'}
                </button>
            </div>

            <ErrorMessage message={error} onDismiss={() => setError('')} />

            {/* Committed Notes */}
            <div>
                <div className="flex items-center justify-between mb-3">
                    <h3 className="text-sm font-medium text-gray-400">
                        Committed Notes ({filteredNotes.length}{filterTagIds.length > 0 ? ` of ${notes.length}` : ''})
                    </h3>
                    <TagFilter selectedTagIds={filterTagIds} onChange={setFilterTagIds} />
                </div>
                {loading ? (
                    <div className="flex justify-center py-8">
                        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-400"></div>
                    </div>
                ) : filteredNotes.length === 0 ? (
                    <p className="text-gray-500 text-sm text-center py-8">
                        {filterTagIds.length > 0
                            ? 'No notes match the selected tags.'
                            : 'No notes committed yet. Write your first note above.'}
                    </p>
                ) : (
                    <div className="space-y-3">
                        {filteredNotes.map((block) => (
                            <NoteBlockCard
                                key={block.block_id}
                                block={block}
                                caseId={caseId}
                                evidenceItems={evidenceItems}
                                onEvidenceClick={onEvidenceClick}
                                onTagsChanged={fetchNotes}
                                onNavigateToTask={onNavigateToTask}
                            />
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
}
