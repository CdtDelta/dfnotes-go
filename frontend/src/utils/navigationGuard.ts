import { securityLog } from './cspReporter';

// Schemes the app resolves itself; following them never leaves the app.
const APP_SCHEMES = new Set(['evidence:', 'attachment:']);

const XLINK_NS = 'http://www.w3.org/1999/xlink';

// Returns the raw href of the nearest link (HTML <a>, <area>, or SVG <a>) in
// the event path, or null when the click did not hit a link.
function linkHref(e: Event): string | null {
    for (const node of e.composedPath()) {
        if (!(node instanceof Element)) continue;
        if (node.localName !== 'a' && node.localName !== 'area') continue;
        const href = node.getAttribute('href') ?? node.getAttributeNS(XLINK_NS, 'href');
        if (href !== null) return href;
    }
    return null;
}

// Compares protocol and host rather than URL.origin: the Linux and macOS app
// origin is wails://wails, and URL.origin is "null" for non-special schemes.
export function isAllowedNavigation(href: string, current: URL): boolean {
    let target: URL;
    try {
        target = new URL(href, current);
    } catch {
        return false;
    }
    if (APP_SCHEMES.has(target.protocol)) return true;
    return target.protocol === current.protocol && target.host === current.host;
}

// Note content can carry attacker-controlled links. Following one replaces
// the app with a remote page in the same window, so block anything that is
// not the app itself. Blocked links are logged, never opened externally.
export function installNavigationGuard(): void {
    const guard = (e: MouseEvent) => {
        if (e.defaultPrevented) return;
        const href = linkHref(e);
        if (href === null || isAllowedNavigation(href, new URL(document.baseURI))) return;
        e.preventDefault();
        securityLog(`[NAV] blocked link href=${href}`);
    };
    // Capture on window runs before any element handler. auxclick covers
    // middle-click, which would otherwise try to open a new window.
    window.addEventListener('click', guard, true);
    window.addEventListener('auxclick', guard, true);
}
