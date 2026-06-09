import { micromark } from 'micromark';
import { gfmStrikethrough, gfmStrikethroughHtml } from 'micromark-extension-gfm-strikethrough';
import { gfmTable, gfmTableHtml } from 'micromark-extension-gfm-table';
import { gfmTaskListItem, gfmTaskListItemHtml } from 'micromark-extension-gfm-task-list-item';
import { gfmTagfilterHtml } from 'micromark-extension-gfm-tagfilter';
import { services } from '../../wailsjs/go/models';

// GFM autolink-literal is intentionally excluded. It converts bare URLs and
// www. domains into <a href> tags, which causes the Wails WebView to attempt
// DNS resolution for any domain value that appears in a note. All IOC values
// must remain as plain text -- no network activity from rendering.
const GFM_EXTENSIONS = [gfmStrikethrough(), gfmTable, gfmTaskListItem];
const GFM_HTML_EXTENSIONS = [gfmStrikethroughHtml, gfmTableHtml, gfmTaskListItemHtml, gfmTagfilterHtml];

const escHtml = (s: string): string =>
    s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
     .replace(/"/g, '&quot;').replace(/'/g, '&#39;');

function preprocessMarkdown(
    content: string,
    evidenceItems: services.EvidenceResponse[],
): string {
    const sorted = [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at));

    // [[item_number]] / [[name]] -- convert to raw HTML span BEFORE micromark so the
    // evidence:// URI is not stripped by micromark's protocol sanitizer.
    let result = content.replace(/\[\[([^\]]+)\]\]/g, (_match, ref: string) => {
        const trimmed = ref.trim();
        const byNumber = sorted.find((e) => e.item_number.toLowerCase() === trimmed.toLowerCase());
        if (byNumber) {
            return `<span class="text-blue-400 hover:text-blue-300 underline cursor-pointer" data-evidence-id="${escHtml(byNumber.evidence_item_id)}" role="button">${escHtml(byNumber.item_number)}: ${escHtml(byNumber.name)}</span>`;
        }
        const byName = sorted.find((e) => e.name.toLowerCase() === trimmed.toLowerCase());
        if (byName) {
            return `<span class="text-blue-400 hover:text-blue-300 underline cursor-pointer" data-evidence-id="${escHtml(byName.evidence_item_id)}" role="button">${escHtml(byName.item_number)}: ${escHtml(byName.name)}</span>`;
        }
        return `[[${ref}]]`;
    });

    // ![alt](attachment:ID) -- convert to img with data attribute; NoteBlockCard
    // loads the data URL via Wails in a useEffect after innerHTML is set.
    result = result.replace(
        /!\[([^\]]*)\]\(attachment:([^)]+)\)/g,
        (_match, alt, id) =>
            `<img data-attachment-id="${id}" alt="${alt.replace(/"/g, '&quot;')}" class="max-w-full rounded" />`,
    );

    return result;
}

export function renderMarkdown(
    content: string,
    evidenceItems: services.EvidenceResponse[],
): string {
    const preprocessed = preprocessMarkdown(content, evidenceItems);
    return micromark(preprocessed, {
        allowDangerousHtml: true,
        extensions: GFM_EXTENSIONS,
        htmlExtensions: GFM_HTML_EXTENSIONS,
    });
}
