import { useState, useEffect, useMemo } from 'react';
import ReactMarkdown from 'react-markdown';
import {
    ATTACHMENT_PREFIX,
    EVIDENCE_PREFIX,
    previewImageUri,
    previewLinkUri,
    remarkGfmNoAutolink,
} from '../utils/markdownPreview';
import { GetAttachment } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';

interface MarkdownRendererProps {
    content: string;
    caseId: string;
    evidenceItems?: services.EvidenceResponse[];
    onEvidenceClick?: (evidenceItemId: string) => void;
}

// Replace [[E001]] or [[name]] with markdown links before rendering. Lookup
// and label match the committed card (markdownToHtml.ts): item_number first,
// then name, labelled "item_number: name".
function preprocessEvidenceLinks(
    content: string,
    evidenceItems: services.EvidenceResponse[]
): string {
    const sorted = [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at));
    return content.replace(/\[\[([^\]]+)\]\]/g, (match, ref: string) => {
        const trimmed = ref.trim().toLowerCase();
        const item =
            sorted.find((e) => e.item_number.toLowerCase() === trimmed) ??
            sorted.find((e) => e.name.toLowerCase() === trimmed);
        if (!item) return match;
        // Escape Markdown punctuation so a name cannot break out of the link text.
        const label = `${item.item_number}: ${item.name}`.replace(/[\\`*_{}[\]()#+\-.!|~<>&]/g, '\\$&');
        return `[${label}](${EVIDENCE_PREFIX}${item.evidence_item_id})`;
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
            remarkPlugins={[remarkGfmNoAutolink]}
            transformLinkUri={previewLinkUri}
            transformImageUri={previewImageUri}
            components={{
                // node is react-markdown's AST node; keep it out of the DOM.
                a: ({ node: _node, href, children, ...props }) => {
                    if (href?.startsWith(EVIDENCE_PREFIX)) {
                        const evidenceItemId = href.replace(EVIDENCE_PREFIX, '');
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
                    // previewLinkUri returns '' for refused URLs: render an inert anchor.
                    return href ? <a href={href} {...props}>{children}</a> : <a {...props}>{children}</a>;
                },
                img: ({ node: _node, src, alt, ...props }) => {
                    if (src?.startsWith(ATTACHMENT_PREFIX)) {
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
