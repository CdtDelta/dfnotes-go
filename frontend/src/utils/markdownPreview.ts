import { uriTransformer } from 'react-markdown';
import { gfmStrikethrough } from 'micromark-extension-gfm-strikethrough';
import { gfmTable } from 'micromark-extension-gfm-table';
import { gfmTaskListItem } from 'micromark-extension-gfm-task-list-item';
import { gfmStrikethroughFromMarkdown } from 'mdast-util-gfm-strikethrough';
import { gfmTableFromMarkdown } from 'mdast-util-gfm-table';
import { gfmTaskListItemFromMarkdown } from 'mdast-util-gfm-task-list-item';

// URI prefixes the Editor Preview resolves itself. Evidence links are written
// by preprocessEvidenceLinks in MarkdownRenderer.tsx; attachment references
// are inserted by NoteEditor.tsx.
export const EVIDENCE_PREFIX = 'evidence://';
export const ATTACHMENT_PREFIX = 'attachment:';

// Evidence links pass through; everything else goes through react-markdown's
// default sanitizer, which keeps http, https, mailto, tel and relative URLs
// and turns any other scheme (javascript:, data:, file:, ...) into
// javascript:void(0). That marker becomes '' so the link renders without an
// href, matching the committed card.
const REFUSED = 'javascript:void(0)';

export function previewLinkUri(href: string): string {
    if (href.startsWith(EVIDENCE_PREFIX)) return href;
    const safe = uriTransformer(href);
    return safe === REFUSED ? '' : safe;
}

// Only attachments render as images. Anything else, including remote http(s)
// images, gets an empty src so no request is attempted.
export function previewImageUri(src: string): string {
    return src.startsWith(ATTACHMENT_PREFIX) ? src : '';
}

// GFM without autolink literals or footnotes, matching the committed card in
// markdownToHtml.ts. remark-gfm 3.0.1 has no option to drop autolink
// literals, so this registers the same micromark syntax extensions the card
// uses plus their mdast counterparts. Autolink literals must stay off: they
// turn bare domains (IOC values) into links.
export function remarkGfmNoAutolink(this: { data(): Record<string, unknown> }): void {
    const data = this.data();
    // Push each extension separately: micromark does not flatten nested arrays.
    const add = (field: string, values: unknown[]) => {
        const list = (data[field] ??= []) as unknown[];
        list.push(...values);
    };
    add('micromarkExtensions', [gfmStrikethrough(), gfmTable, gfmTaskListItem]);
    add('fromMarkdownExtensions', [gfmStrikethroughFromMarkdown, gfmTableFromMarkdown, gfmTaskListItemFromMarkdown]);
}
