export interface CorruptedNzbSegment {
    id: string;
    number: number;
    uuid: string;
    created_at: string;
}

export interface CorruptedNzb {
    id: number;
    path: string;
    created_at: string;
    error: string;
    segments: CorruptedNzbSegment[];
}


export interface CorruptedNzbResponse {
   entries: CorruptedNzb[];
   total_count: number;
   limit: number;
   offset: number;
}
