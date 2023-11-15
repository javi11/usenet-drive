export enum Kind {
    download = 'download',
    upload = 'upload',
}

export interface Activity {
    session_id: string;
    kind: Kind;
    path: string;
    speed: number;
    total_bytes: number;
}