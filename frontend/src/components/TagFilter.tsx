import { useState, useEffect } from 'react';
import { ListTags } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';

interface TagFilterProps {
    selectedTagIds: string[];
    onChange: (tagIds: string[]) => void;
}

export default function TagFilter({ selectedTagIds, onChange }: TagFilterProps) {
    const [allTags, setAllTags] = useState<services.TagResponse[]>([]);
    const [expanded, setExpanded] = useState(false);

    useEffect(() => {
        ListTags().then((t) => setAllTags(t || [])).catch(() => {});
    }, [expanded]);

    const toggle = (tagId: string) => {
        if (selectedTagIds.includes(tagId)) {
            onChange(selectedTagIds.filter((id) => id !== tagId));
        } else {
            onChange([...selectedTagIds, tagId]);
        }
    };

    if (allTags.length === 0) return null;

    return (
        <div className="flex items-center gap-2 flex-wrap">
            <button
                onClick={() => setExpanded(!expanded)}
                className="text-xs text-gray-400 hover:text-gray-200 transition-colors inline-flex items-center gap-1"
            >
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 3c2.755 0 5.455.232 8.083.678.533.09.917.556.917 1.096v1.044a2.25 2.25 0 01-.659 1.591l-5.432 5.432a2.25 2.25 0 00-.659 1.591v2.927a2.25 2.25 0 01-1.244 2.013L9.75 21v-6.568a2.25 2.25 0 00-.659-1.591L3.659 7.409A2.25 2.25 0 013 5.818V4.774c0-.54.384-1.006.917-1.096A48.32 48.32 0 0112 3z" />
                </svg>
                Filter{selectedTagIds.length > 0 && ` (${selectedTagIds.length})`}
            </button>
            {expanded && (
                <>
                    {allTags.map((tag) => {
                        const active = selectedTagIds.includes(tag.tag_id);
                        return (
                            <button
                                key={tag.tag_id}
                                onClick={() => toggle(tag.tag_id)}
                                className={`text-xs px-1.5 py-0.5 rounded border transition-colors ${
                                    active ? 'opacity-100' : 'opacity-40 hover:opacity-70'
                                }`}
                                style={{
                                    backgroundColor: tag.color + '22',
                                    color: tag.color,
                                    borderColor: tag.color + (active ? '88' : '44'),
                                }}
                            >
                                {tag.name}
                            </button>
                        );
                    })}
                    {selectedTagIds.length > 0 && (
                        <button
                            onClick={() => onChange([])}
                            className="text-xs text-gray-500 hover:text-gray-300 transition-colors"
                        >
                            Clear
                        </button>
                    )}
                </>
            )}
        </div>
    );
}
