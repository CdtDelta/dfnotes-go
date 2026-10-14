import type { IOCType } from './iocTypes';

// Fallback used until GetFactTypes() returns the backend's list.
export const FACT_TYPES = [
    'username', 'hostname', 'ip_address', 'mac_address', 'os_version', 'timezone',
    'email_address', 'account_sid', 'full_name', 'phone_number', 'device_serial',
    'url', 'file_path', 'domain', 'registry_key', 'custom',
];

export function formatFactType(t: string): string {
    const special: Record<string, string> = {
        ip_address: 'IP Address', mac_address: 'MAC Address',
        os_version: 'OS Version', account_sid: 'Account SID', url: 'URL',
    };
    return special[t] ?? t.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
}

export function iocTypeToFactType(iocType: IOCType): string {
    const map: Partial<Record<IOCType, string>> = {
        ipv4: 'ip_address', ipv6: 'ip_address',
        domain: 'domain', url: 'url',
        email: 'email_address',
        file_path: 'file_path', file: 'file_path',
        registry_key: 'registry_key',
    };
    return map[iocType] ?? 'custom';
}
