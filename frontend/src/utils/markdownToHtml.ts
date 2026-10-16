import { micromark } from 'micromark';
import { gfmStrikethrough, gfmStrikethroughHtml } from 'micromark-extension-gfm-strikethrough';
import { gfmTable, gfmTableHtml } from 'micromark-extension-gfm-table';
import { gfmTaskListItem, gfmTaskListItemHtml } from 'micromark-extension-gfm-task-list-item';
import { services } from '../../wailsjs/go/models';
import { applyIOCHighlights } from './highlightIOCs';
import type { BlockIOC } from './iocTypes';

// GFM autolink-literal is intentionally excluded. It converts bare URLs and
// www. domains into <a href> tags, which causes the Wails WebView to attempt
// DNS resolution for any domain value that appears in a note. All IOC values
// must remain as plain text -- no network activity from rendering.
const GFM_EXTENSIONS = [gfmStrikethrough(), gfmTable, gfmTaskListItem];
const GFM_HTML_EXTENSIONS = [gfmStrikethroughHtml, gfmTableHtml, gfmTaskListItemHtml];

/*
 * Note content is untrusted, so micromark runs with raw HTML off and its
 * default URL sanitizer on. Evidence links and attachment images are not
 * Markdown, so before parsing each one is swapped for a placeholder made of
 * private-use characters (U+E000 kind index U+E001), which micromark passes
 * through as ordinary text. After parsing, a DOM pass in an inert document
 * replaces the placeholders with elements built by DOM APIs. Any U+E000 or
 * U+E001 already in the note is itself encoded as a placeholder first, so it
 * cannot forge one and is restored verbatim.
 */
const OPEN = '';
const CLOSE = '';
const TOKEN = /([EAL])(\d+)/g;
// The same token after micromark percent-encodes it inside a URL.
const ENCODED_TOKEN = /%EE%80%80([EAL])(\d+)%EE%80%81/g;

// Attachment IDs are the lowercase UUID from uuid.New().String() in
// note_service.go; anything else stays an ordinary Markdown image.
const ATTACHMENT = /!\[([^\]]*)\]\(attachment:([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\)/g;

const EVIDENCE_CLASS = 'text-blue-400 hover:text-blue-300 underline cursor-pointer';
const ATTACHMENT_CLASS = 'max-w-full rounded';

type Placeholder =
    | { kind: 'E'; source: string; evidenceId: string; label: string }
    | { kind: 'A'; source: string; attachmentId: string; alt: string }
    | { kind: 'L'; source: string };

function substitutePlaceholders(
    content: string,
    evidenceItems: services.EvidenceResponse[],
): { text: string; table: Placeholder[] } {
    const table: Placeholder[] = [];
    const token = (p: Placeholder) => {
        table.push(p);
        return `${OPEN}${p.kind}${table.length - 1}${CLOSE}`;
    };

    let text = content.replace(/[]/g, (ch) => token({ kind: 'L', source: ch }));

    const sorted = [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at));
    text = text.replace(/\[\[([^\]]+)\]\]/g, (match, ref: string) => {
        const trimmed = ref.trim().toLowerCase();
        const item =
            sorted.find((e) => e.item_number.toLowerCase() === trimmed) ??
            sorted.find((e) => e.name.toLowerCase() === trimmed);
        if (!item) return match;
        return token({
            kind: 'E',
            source: match,
            evidenceId: item.evidence_item_id,
            label: `${item.item_number}: ${item.name}`,
        });
    });

    text = text.replace(ATTACHMENT, (match, alt: string, id: string) =>
        token({ kind: 'A', source: match, attachmentId: id, alt }),
    );

    return { text, table };
}

// Restores the original source text for every placeholder in s.
function restoreSource(s: string, table: Placeholder[]): string {
    return s.replace(TOKEN, (m, _k, i) => table[Number(i)]?.source ?? m);
}

function buildPlaceholder(doc: Document, p: Placeholder): Node {
    if (p.kind === 'E') {
        const span = doc.createElement('span');
        span.className = EVIDENCE_CLASS;
        span.setAttribute('data-evidence-id', p.evidenceId);
        span.setAttribute('role', 'button');
        span.textContent = p.label;
        return span;
    }
    if (p.kind === 'A') {
        const img = doc.createElement('img');
        img.setAttribute('data-attachment-id', p.attachmentId);
        img.setAttribute('alt', p.alt);
        img.className = ATTACHMENT_CLASS;
        return img;
    }
    return doc.createTextNode(p.source);
}

function expandTextNode(node: Text, table: Placeholder[]): void {
    const data = node.data;
    if (!data.includes(OPEN)) return;

    // Inside code the reader expects the literal Markdown back.
    if (node.parentElement?.closest('code, pre')) {
        node.data = restoreSource(data, table);
        return;
    }

    const doc = node.ownerDocument;
    const out: Node[] = [];
    let last = 0;
    for (const m of data.matchAll(TOKEN)) {
        const p = table[Number(m[2])];
        if (!p) continue;
        if (m.index! > last) out.push(doc.createTextNode(data.slice(last, m.index)));
        out.push(buildPlaceholder(doc, p));
        last = m.index! + m[0].length;
    }
    if (last < data.length) out.push(doc.createTextNode(data.slice(last)));
    node.replaceWith(...out);
}

function restoreAttributes(root: Element, table: Placeholder[]): void {
    for (const el of root.querySelectorAll('*')) {
        for (const attr of Array.from(el.attributes)) {
            let v = attr.value;
            if (v.includes(OPEN)) v = restoreSource(v, table);
            if (/%EE%80%80/i.test(v)) {
                v = v.replace(ENCODED_TOKEN, (m, _k, i) => {
                    const p = table[Number(i)];
                    return p ? encodeURI(p.source) : m;
                });
            }
            if (v !== attr.value) el.setAttribute(attr.name, v);
        }
    }
}

const isAttachmentImg = (n: Node): boolean =>
    n instanceof Element && n.localName === 'img' && n.hasAttribute('data-attachment-id');

// The pre-pass-3 renderer emitted attachments as raw HTML, and an <img> alone
// on its lines was a CommonMark HTML block with no <p> around it. Unwrap
// paragraphs that hold only attachment images to keep that layout.
function unwrapAttachmentParagraphs(root: Element): void {
    for (const p of Array.from(root.querySelectorAll('p'))) {
        const kids = Array.from(p.childNodes);
        const onlyAttachments =
            kids.some(isAttachmentImg) &&
            kids.every((n) => isAttachmentImg(n) || (n.nodeType === Node.TEXT_NODE && !n.textContent!.trim()));
        if (onlyAttachments) p.replaceWith(...kids);
    }
}

// Remote images never load (CSP would block them anyway) and links whose URL
// micromark refused (javascript:, data:, file:, ...) lose their href, so a
// click can neither navigate nor reload the app.
function neutralizeUrls(root: Element): void {
    for (const img of root.querySelectorAll('img:not([data-attachment-id])')) {
        img.setAttribute('src', '');
    }
    for (const a of root.querySelectorAll('a[href=""]')) {
        a.removeAttribute('href');
    }
}

/*
 * Renders a note block to HTML for NoteBlockCard. The result is produced
 * from micromark's safe output plus elements built with DOM APIs in an inert
 * document (nothing loads or runs there), so no note text is ever parsed as
 * markup. IOC highlights are applied last, to text nodes only.
 */
export function renderNoteHtml(
    content: string,
    evidenceItems: services.EvidenceResponse[],
    iocs: BlockIOC[],
): string {
    const { text, table } = substitutePlaceholders(content, evidenceItems);
    const html = micromark(text, {
        extensions: GFM_EXTENSIONS,
        htmlExtensions: GFM_HTML_EXTENSIONS,
    });

    const doc = document.implementation.createHTMLDocument('');
    const root = doc.body;
    root.innerHTML = html;

    const walker = doc.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    const textNodes: Text[] = [];
    for (let n = walker.nextNode(); n; n = walker.nextNode()) textNodes.push(n as Text);
    for (const node of textNodes) expandTextNode(node, table);

    restoreAttributes(root, table);
    unwrapAttachmentParagraphs(root);
    neutralizeUrls(root);
    applyIOCHighlights(root, iocs);

    return root.innerHTML;
}
