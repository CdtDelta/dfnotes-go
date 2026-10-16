import { LogPrint } from '../../wailsjs/runtime/runtime';

// Writes a security event to the Wails terminal log so it is visible without
// devtools. LogPrint is used because Wails prints it regardless of log level;
// production builds default to ERROR and would drop LogWarning.
export function securityLog(msg: string): void {
    try {
        LogPrint(msg);
    } catch {
        // Outside the Wails WebView there is no runtime to log through.
        console.warn(msg);
    }
}

export function installCSPReporter(): void {
    document.addEventListener('securitypolicyviolation', (e: SecurityPolicyViolationEvent) => {
        securityLog(`[CSP] blocked directive=${e.violatedDirective} uri=${e.blockedURI} source=${e.sourceFile}:${e.lineNumber}`);
    });
}
