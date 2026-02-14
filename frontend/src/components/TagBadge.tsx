import { services } from '../../wailsjs/go/models';

interface TagBadgeProps {
    tag: services.TagResponse;
    onRemove?: () => void;
}

export default function TagBadge({ tag, onRemove }: TagBadgeProps) {
    return (
        <span
            className="inline-flex items-center gap-1 text-xs px-1.5 py-0.5 rounded"
            style={{ backgroundColor: tag.color + '22', color: tag.color, border: `1px solid ${tag.color}44` }}
        >
            {tag.name}
            {onRemove && (
                <button
                    onClick={(e) => { e.stopPropagation(); onRemove(); }}
                    className="hover:opacity-70 ml-0.5"
                >
                    &times;
                </button>
            )}
        </span>
    );
}
