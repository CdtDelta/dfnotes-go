import { useState, useEffect, useRef } from 'react';
import { ListTags, CreateTag } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';

interface TagSelectorProps {
    currentTags: services.TagResponse[];
    onTag: (tagId: string) => void;
    onUntag: (tagId: string) => void;
}

export default function TagSelector({ currentTags, onTag, onUntag }: TagSelectorProps) {
    const [open, setOpen] = useState(false);
    const [allTags, setAllTags] = useState<services.TagResponse[]>([]);
    const [search, setSearch] = useState('');
    const [creating, setCreating] = useState(false);
    const ref = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (open) {
            ListTags().then((t) => setAllTags(t || [])).catch(() => {});
        }
    }, [open]);

    useEffect(() => {
        const handleClick = (e: MouseEvent) => {
            if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
        };
        document.addEventListener('mousedown', handleClick);
        return () => document.removeEventListener('mousedown', handleClick);
    }, []);

    const currentIds = new Set(currentTags.map((t) => t.tag_id));
    const filtered = allTags.filter((t) =>
        t.name.toLowerCase().includes(search.toLowerCase())
    );

    const handleCreate = async () => {
        if (!search.trim()) return;
        setCreating(true);
        try {
            const tag = await CreateTag({ name: search.trim(), color: '#6B7280' } as services.CreateTagRequest);
            onTag(tag.tag_id);
            setSearch('');
            setAllTags((prev) => [...prev, tag]);
        } catch { /* ignore */ }
        setCreating(false);
    };

    const exactMatch = allTags.some((t) => t.name.toLowerCase() === search.trim().toLowerCase());

    return (
        <div className="relative inline-block" ref={ref}>
            <button
                onClick={() => setOpen(!open)}
                className="text-xs text-gray-500 hover:text-gray-300 transition-colors"
                title="Manage tags"
            >
                + tag
            </button>
            {open && (
                <div className="absolute z-20 top-full left-0 mt-1 w-56 bg-gray-800 border border-gray-600 rounded-lg shadow-xl">
                    <div className="p-2">
                        <input
                            type="text"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            placeholder="Search or create tag..."
                            className="w-full px-2 py-1 bg-gray-900 border border-gray-700 rounded text-xs text-gray-100 focus:outline-none focus:border-blue-500 placeholder-gray-600"
                            autoFocus
                        />
                    </div>
                    <div className="max-h-48 overflow-y-auto px-1 pb-1">
                        {filtered.map((tag) => {
                            const attached = currentIds.has(tag.tag_id);
                            return (
                                <button
                                    key={tag.tag_id}
                                    onClick={() => { attached ? onUntag(tag.tag_id) : onTag(tag.tag_id); }}
                                    className="w-full text-left px-2 py-1.5 text-xs rounded hover:bg-gray-700 flex items-center gap-2 transition-colors"
                                >
                                    <span
                                        className="w-2.5 h-2.5 rounded-full shrink-0"
                                        style={{ backgroundColor: tag.color }}
                                    />
                                    <span className="text-gray-200 flex-1 truncate">{tag.name}</span>
                                    {attached && <span className="text-green-400 text-[10px]">&#10003;</span>}
                                </button>
                            );
                        })}
                        {search.trim() && !exactMatch && (
                            <button
                                onClick={handleCreate}
                                disabled={creating}
                                className="w-full text-left px-2 py-1.5 text-xs rounded hover:bg-gray-700 text-blue-400 transition-colors"
                            >
                                {creating ? 'Creating...' : `Create "${search.trim()}"`}
                            </button>
                        )}
                        {filtered.length === 0 && !search.trim() && (
                            <p className="px-2 py-1.5 text-xs text-gray-500">No tags available</p>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}
