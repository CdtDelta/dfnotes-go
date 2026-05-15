import type { IOCType } from './iocTypes';

export function defang(value: string, type: IOCType): string {
    switch (type) {
        case 'ipv4':
            return value.replace(/\./g, '[.]');
        case 'ipv6':
            return value.replace(/:/g, '[:]');
        case 'domain':
            return value.replace(/\.([^.]+)$/, '[.]$1');
        case 'url':
            return value
                .replace(/^https:\/\//, 'hxxps://')
                .replace(/^http:\/\//, 'hxxp://')
                .replace(/^ftp:\/\//, 'fxp://')
                .replace(/\./g, '[.]');
        case 'email': {
            const at = value.indexOf('@');
            if (at === -1) return value;
            const domain = value.slice(at + 1).replace(/\./g, '[.]');
            return `${value.slice(0, at)}[@]${domain}`;
        }
        default:
            return value;
    }
}
