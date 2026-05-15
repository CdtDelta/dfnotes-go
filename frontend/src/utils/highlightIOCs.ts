import type { IOCEntry } from './iocTypes';

// Applies IOC highlights to an HTML string by transforming only text segments
// (never attribute values). Splits on tags so the replacements can never
// corrupt an href, src, or class attribute even if the IOC value appears there.
export function applyIOCHighlightsToHtml(html: string, iocs: IOCEntry[]): string {
    if (iocs.length === 0) return html;

    const byValue = new Map<string, IOCEntry>();
    for (const ioc of iocs) {
        byValue.set(ioc.value, ioc);
    }
    // Sort longest first so a longer overlapping value wins over a shorter one.
    const values = Array.from(byValue.keys()).sort((a, b) => b.length - a.length);

    // Splitting with a capturing group puts the delimiters (tags) at odd indices;
    // text content lands at even indices.
    const parts = html.split(/(<[^>]*>)/);
    return parts
        .map((part, i) => (i % 2 === 1 ? part : highlightTextSegment(part, values, byValue)))
        .join('');
}

function highlightTextSegment(
    text: string,
    values: string[],
    byValue: Map<string, IOCEntry>,
): string {
    if (!text) return text;
    let result = '';
    let pos = 0;

    while (pos < text.length) {
        let bestIdx = -1;
        let bestLen = 0;
        let bestIoc: IOCEntry | null = null;

        for (const value of values) {
            const idx = text.indexOf(value, pos);
            if (idx === -1) continue;
            if (
                bestIdx === -1 ||
                idx < bestIdx ||
                (idx === bestIdx && value.length > bestLen)
            ) {
                bestIdx = idx;
                bestLen = value.length;
                bestIoc = byValue.get(value)!;
            }
        }

        if (bestIdx === -1) {
            result += text.slice(pos);
            break;
        }

        result += text.slice(pos, bestIdx);

        const ioc = bestIoc!;
        const escaped = ioc.value
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;');
        result +=
            `<span class="ioc-highlight ioc-highlight--${ioc.status}"` +
            ` data-ioc-id="${ioc.ioc_id}"` +
            ` data-ioc-type="${ioc.type}"` +
            ` data-ioc-value="${escaped}"` +
            ` data-ioc-status="${ioc.status}">${escaped}</span>`;

        pos = bestIdx + bestLen;
    }

    return result;
}
