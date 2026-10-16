import { LogPrint } from '../../wailsjs/runtime/runtime';

// Routes CSP violations to the Wails terminal log so they are visible without
// devtools. LogPrint is used because Wails prints it regardless of log level;
// production builds default to ERROR and would drop LogWarning.
export function installCSPReporter(): void {
    document.addEventListener('securitypolicyviolation', (e: SecurityPolicyViolationEvent) => {
        const msg = `[CSP] blocked directive=${e.violatedDirective} uri=${e.blockedURI} source=${e.sourceFile}:${e.lineNumber}`;
        try {
            LogPrint(msg);
        } catch {
            // Outside the Wails WebView there is no runtime to log through.
            console.warn(msg);
        }
    });
}
