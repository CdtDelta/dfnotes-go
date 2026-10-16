import type { BlockIOC } from './iocTypes';

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
 * Applies IOC highlights to the text nodes under root, in place. Only Text
 * node data is searched and highlights are built with DOM APIs, so markup is
 * never parsed and attribute values are never touched. Each text node is
 * scanned on its own: a value split across elements (for example by
 * emphasis) is not matched. Each occurrence is matched on its own raw_value,
 * exactly and case-sensitively. Scanning runs left to right; the earliest
 * match wins, a longer match wins at the same index, and highlights never
 * nest.
 *
 * When more than one indicator has the identical raw_value in this block, one
 * span is emitted for the indicator chosen by this priority:
 *   1. status detected or confirmed beats false_positive or promoted;
 *   2. confirmed beats detected;
 *   3. detection_method manual or propagated beats auto;
 *   4. otherwise the first in input order.
 * If the chosen indicator is false_positive or promoted, the text is left plain.
 */
export function applyIOCHighlights(root: Node, iocs: BlockIOC[]): void {
    if (iocs.length === 0) return;

    const byText = new Map<string, BlockIOC>();
    for (const ioc of iocs) {
        if (!ioc.raw_value) continue;
        const current = byText.get(ioc.raw_value);
        if (!current || outranks(ioc, current)) byText.set(ioc.raw_value, ioc);
    }
    // Longest first so a longer value wins over a shorter one at the same index.
    const texts = Array.from(byText.keys()).sort((a, b) => b.length - a.length);

    // Collect first: highlighting replaces nodes, which would upset a live walk.
    const doc = root.ownerDocument ?? (root as Document);
    const walker = doc.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    const nodes: Text[] = [];
    for (let n = walker.nextNode(); n; n = walker.nextNode()) nodes.push(n as Text);

    for (const node of nodes) highlightTextNode(node, texts, byText);
}

function highlightTextNode(node: Text, texts: string[], byText: Map<string, BlockIOC>): void {
    const text = node.data;
    if (!text) return;
    const doc = node.ownerDocument;
    const out: Node[] = [];
    let plain = '';
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
            plain += text.slice(pos);
            break;
        }

        plain += text.slice(pos, bestIdx);
        const ioc = byText.get(bestText)!;
        if (!isActive(ioc)) {
            plain += bestText;
        } else {
            if (plain) out.push(doc.createTextNode(plain));
            plain = '';
            const span = doc.createElement('span');
            span.className = `ioc-highlight ioc-highlight--${ioc.status}`;
            span.setAttribute('data-indicator-id', ioc.indicator_id);
            span.setAttribute('data-occurrence-id', ioc.occurrence_id);
            span.setAttribute('data-ioc-type', ioc.type);
            span.setAttribute('data-ioc-value', ioc.raw_value);
            span.setAttribute('data-ioc-status', ioc.status);
            span.textContent = bestText;
            out.push(span);
        }
        pos = bestIdx + bestText.length;
    }

    if (out.length === 0) return; // nothing highlighted; leave the node as is
    if (plain) out.push(doc.createTextNode(plain));
    node.replaceWith(...out);
}
