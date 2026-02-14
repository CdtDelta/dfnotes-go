const classificationColors: Record<string, string> = {
    'UNCLASSIFIED': 'bg-green-700 text-green-100',
    'CONFIDENTIAL': 'bg-blue-700 text-blue-100',
    'SECRET': 'bg-red-700 text-red-100',
    'TOP SECRET': 'bg-orange-700 text-orange-100',
};

interface ClassificationBadgeProps {
    level: string;
}

export default function ClassificationBadge({ level }: ClassificationBadgeProps) {
    const colorClass = classificationColors[level] || 'bg-gray-700 text-gray-100';

    return (
        <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-semibold uppercase ${colorClass}`}>
            {level}
        </span>
    );
}
