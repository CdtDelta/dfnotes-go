export type IOCType =
    | 'ipv4'
    | 'ipv6'
    | 'domain'
    | 'url'
    | 'email'
    | 'md5'
    | 'sha1'
    | 'sha256'
    | 'file_path'
    | 'registry_key'
    | 'cve';

export type IOCStatus = 'detected' | 'confirmed' | 'false_positive';

export interface IOCEntry {
    ioc_id: string;
    case_id: string;
    block_id: string;
    evidence_item_id?: string;
    type: IOCType;
    value: string;
    status: IOCStatus;
    detection_method: string;
    notes?: string;
    created_at: string;
    confirmed_at?: string;
    user_id: string;
}
