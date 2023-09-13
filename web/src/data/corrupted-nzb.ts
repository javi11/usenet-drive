export interface CorruptedNzb {
    id: number;
    path: string;
    created_at: string;
    error: string;
}


export interface CorruptedNzbResponse {
   entries: CorruptedNzb[];
   total_count: number;
   limit: number;
   offset: number;
}
