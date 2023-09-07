export enum JobStatus {
    Pending = "pending",
    InProgress = "in-progress",
    Failed = "failed",
}

export interface JobData {
    id: number;
    data: string;
    created_at: string;
    error?: string;
    status: JobStatus;
}


export interface JobResponse {
   entries: JobData[];
   total_count: number;
   limit: number;
   offset: number;
}
