import { useState, useEffect, useMemo } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { GetAttachment } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';

interface MarkdownRendererProps {
    content: string;
    caseId: string;
    evidenceItems?: services.EvidenceResponse[];
    onEvidenceClick?: (evidenceItemId: string) => void;
}

// Replace [[E001]] or [[name]] with markdown links before rendering
function preprocessEvidenceLinks(
    content: string,
    evidenceItems: services.EvidenceResponse[]
): string {
    return content.replace(/\[\[([^\]]+)\]\]/g, (_match, ref: string) => {
        const trimmed = ref.trim();
        // Match by E### label
        const labelMatch = trimmed.match(/^E(\d+)$/i);
        if (labelMatch) {
            const idx = parseInt(labelMatch[1], 10) - 1;
            // Sort by created_at ASC to match tab ordering
            const sorted = [...evidenceItems].sort((a, b) =>
                a.created_at.localeCompare(b.created_at)
            );
            if (idx >= 0 && idx < sorted.length) {
                return `[${trimmed}](evidence://${sorted[idx].evidence_item_id})`;
            }
        }
        // Match by name
        const byName = evidenceItems.find(
            (e) => e.name.toLowerCase() === trimmed.toLowerCase()
        );
        if (byName) {
            const sorted = [...evidenceItems].sort((a, b) =>
                a.created_at.localeCompare(b.created_at)
            );
            const idx = sorted.findIndex((e) => e.evidence_item_id === byName.evidence_item_id);
            const label = `E${String(idx + 1).padStart(3, '0')}`;
            return `[${label}: ${byName.name}](evidence://${byName.evidence_item_id})`;
        }
        return `[[${ref}]]`;
    });
}

function AttachmentImage({ src, caseId, alt }: { src: string; caseId: string; alt?: string }) {
    const [dataUrl, setDataUrl] = useState<string | null>(null);
    const attachmentId = src.replace('attachment:', '');

    useEffect(() => {
        GetAttachment(caseId, attachmentId)
            .then((resp) => {
                setDataUrl(`data:${resp.content_type};base64,${resp.data}`);
            })
            .catch(() => {});
    }, [caseId, attachmentId]);

    if (!dataUrl) {
        return <span className="text-xs text-gray-500 italic">Loading image...</span>;
    }
    return <img src={dataUrl} alt={alt || 'attachment'} className="max-w-full rounded" />;
}

export default function MarkdownRenderer({ content, caseId, evidenceItems, onEvidenceClick }: MarkdownRendererProps) {
    const processed = useMemo(
        () => preprocessEvidenceLinks(content, evidenceItems || []),
        [content, evidenceItems]
    );

    return (
        <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            transformLinkUri={(uri) => uri}
            transformImageUri={(uri) => uri}
            components={{
                a: ({ href, children, ...props }) => {
                    if (href?.startsWith('evidence://')) {
                        const evidenceItemId = href.replace('evidence://', '');
                        return (
                            <button
                                onClick={() => onEvidenceClick?.(evidenceItemId)}
                                className="text-blue-400 hover:text-blue-300 underline cursor-pointer"
                                {...(props as any)}
                            >
                                {children}
                            </button>
                        );
                    }
                    return <a href={href} {...props}>{children}</a>;
                },
                img: ({ src, alt, ...props }) => {
                    if (src?.startsWith('attachment:')) {
                        return <AttachmentImage src={src} caseId={caseId} alt={alt} />;
                    }
                    return <img src={src} alt={alt} {...props} />;
                },
            }}
        >
            {processed}
        </ReactMarkdown>
    );
}
