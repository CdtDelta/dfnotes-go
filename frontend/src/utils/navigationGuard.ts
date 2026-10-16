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

// The app has no legitimate anchor navigation: routing is HashRouter driven
// by navigate() calls, and the only anchors come from note content. So even
// same-origin and fragment links are blocked; an empty href would reload the
// app (dropping to the login screen) and a fragment would change the
// HashRouter route. Only the app's internal schemes pass.
export function isAllowedNavigation(href: string, base: URL): boolean {
    try {
        return APP_SCHEMES.has(new URL(href, base).protocol);
    } catch {
        return false;
    }
}

// Note content can carry attacker-controlled links. Following one replaces
// the app with a remote page in the same window, so block every link except
// the internal schemes. Blocked links are logged, never opened externally.
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
