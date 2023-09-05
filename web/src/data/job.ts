export enum JobStatus {
    Pending = "pending",
    InProgress = "in-progress",
    Failed = "failed",
}

export interface JobData {
    id: number;
    data: string;
    createdAt: string;
    error?: string;
    status: JobStatus;
}


export interface JobResponse {
   entries: JobData[];
   totalCount: number;
   limit: number;
   offset: number;
}
