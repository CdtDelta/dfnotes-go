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
    | 'file'
    | 'registry_key'
    | 'cve';

export type IOCStatus = 'detected' | 'confirmed' | 'false_positive' | 'promoted';

export type DetectionMethod = 'auto' | 'manual' | 'propagated';

// The interfaces below mirror the Go structs in internal/ioc/indicator_types.go
// field for field. Fields tagged omitempty in Go are optional here.

export interface Indicator {
    indicator_id: string;
    case_id: string;
    type: IOCType;
    value: string;
    match_key: string;
    match_key_version: number;
    status: IOCStatus;
    created_at: string;
    updated_at: string;
    confirmed_at?: string;
    user_id: string;
    occurrences: Occurrence[];
}

export interface Occurrence {
    occurrence_id: string;
    indicator_id: string;
    block_id: string;
    evidence_item_id?: string;
    committed_at: string;
    raw_value: string;
    detection_method: DetectionMethod;
    created_at: string;
    user_id: string;
}

export interface BlockIOC {
    occurrence_id: string;
    indicator_id: string;
    block_id: string;
    type: IOCType;
    raw_value: string;
    status: IOCStatus;
    detection_method: DetectionMethod;
}

export interface UpsertResult {
    indicator_id: string;
    status: IOCStatus;
    created: boolean;
    already_tracked: boolean;
    status_changed: boolean;
}

export interface TypeCollision {
    target_indicator_id: string;
    target_status: IOCStatus;
    occurrences_to_move: number;
}

export interface TypeChangeResult {
    changed: boolean;
    merged: boolean;
    collision?: TypeCollision;
}
