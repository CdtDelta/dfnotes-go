import type { BlockIOC } from './iocTypes';

// Matches micromark's text encoding (micromark-util-encode), which escapes
// exactly these four characters. A raw value has to be searched for in this
// form, or a URL containing "&" would never match the rendered "&amp;".
function escapeAsRenderedText(s: string): string {
    return s
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

function escapeAttr(s: string): string {
    return escapeAsRenderedText(s).replace(/'/g, '&#39;');
}

function isActive(ioc: BlockIOC): boolean {
    return ioc.status === 'detected' || ioc.status === 'confirmed';
}

// Returns true when candidate should replace current for the same raw text.
function outranks(candidate: BlockIOC, current: BlockIOC): boolean {
    if (isActive(candidate) !== isActive(current)) return isActive(candidate);
    if (candidate.status !== current.status) {
        if (candidate.status === 'confirmed' && current.status === 'detected') return true;
        if (candidate.status === 'detected' && current.status === 'confirmed') return false;
    }
    const candidateExplicit = candidate.detection_method !== 'auto';
    const currentExplicit = current.detection_method !== 'auto';
    if (candidateExplicit !== currentExplicit) return candidateExplicit;
    return false;
}

/*
 * Applies IOC highlights to rendered HTML by transforming only text segments
 * (never tags or attribute values), so a replacement can never corrupt an href,
 * src, or class even if the value appears there. Each occurrence is matched on
 * its own raw_value, exactly and case-sensitively, in the escaped form the
 * renderer emits. Scanning runs left to right; the earliest match wins, a longer
 * match wins at the same index, and highlights never nest.
 *
 * When more than one indicator has the identical raw_value in this block, one
 * span is emitted for the indicator chosen by this priority:
 *   1. status detected or confirmed beats false_positive or promoted;
 *   2. confirmed beats detected;
 *   3. detection_method manual or propagated beats auto;
 *   4. otherwise the first in input order.
 * If the chosen indicator is false_positive or promoted, the text is left plain.
 */
export function applyIOCHighlightsToHtml(html: string, iocs: BlockIOC[]): string {
    if (iocs.length === 0) return html;

    const byText = new Map<string, BlockIOC>();
    for (const ioc of iocs) {
        if (!ioc.raw_value) continue;
        const key = escapeAsRenderedText(ioc.raw_value);
        const current = byText.get(key);
        if (!current || outranks(ioc, current)) byText.set(key, ioc);
    }
    // Longest first so a longer value wins over a shorter one at the same index.
    const texts = Array.from(byText.keys()).sort((a, b) => b.length - a.length);

    // Splitting with a capturing group puts the delimiters (tags) at odd indices;
    // text content lands at even indices.
    const parts = html.split(/(<[^>]*>)/);
    return parts
        .map((part, i) => (i % 2 === 1 ? part : highlightTextSegment(part, texts, byText)))
        .join('');
}

function highlightTextSegment(
    text: string,
    texts: string[],
    byText: Map<string, BlockIOC>,
): string {
    if (!text) return text;
    let result = '';
    let pos = 0;

    while (pos < text.length) {
        let bestIdx = -1;
        let bestText = '';

        for (const candidate of texts) {
            const idx = text.indexOf(candidate, pos);
            if (idx === -1) continue;
            if (
                bestIdx === -1 ||
                idx < bestIdx ||
                (idx === bestIdx && candidate.length > bestText.length)
            ) {
                bestIdx = idx;
                bestText = candidate;
            }
        }

        if (bestIdx === -1) {
            result += text.slice(pos);
            break;
        }

        result += text.slice(pos, bestIdx);

        const ioc = byText.get(bestText)!;
        if (!isActive(ioc)) {
            result += bestText;
        } else {
            result +=
                `<span class="ioc-highlight ioc-highlight--${escapeAttr(ioc.status)}"` +
                ` data-indicator-id="${escapeAttr(ioc.indicator_id)}"` +
                ` data-occurrence-id="${escapeAttr(ioc.occurrence_id)}"` +
                ` data-ioc-type="${escapeAttr(ioc.type)}"` +
                ` data-ioc-value="${escapeAttr(ioc.raw_value)}"` +
                ` data-ioc-status="${escapeAttr(ioc.status)}">${bestText}</span>`;
        }

        pos = bestIdx + bestText.length;
    }

    return result;
}
