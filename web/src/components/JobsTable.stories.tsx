import { JobStatus } from '../data/job';
import JobsTable from './JobsTable';

export default {
    component: JobsTable,
    title: 'JobsTable',
    tags: ['autodocs'],
};

export const Default = {
    args: {
        data: {
            entries: [
                {
                    id: 1,
                    data: "some data",
                    created_at: "2022-01-01T00:00:00.000Z",
                    error: "some error",
                    status: JobStatus.Pending,
                },
                {
                    id: 2,
                    data: "some other data",
                    created_at: "2022-01-02T00:00:00.000Z",
                    status: JobStatus.Pending,
                },
            ],
            total_count: 2,
            limit: 10,
            offset: 0,
        },
        onPageChange: () => null,
        onDelete: undefined,
        onRetry: undefined,
    },
};

export const LargePath = {
    args: {
        data: {
            entries: [
                {
                    id: 1,
                    data: "/some/very/long/path/that/doesnt/fit/on/one/line/and/should/be/truncated",
                    created_at: "2022-01-01T00:00:00.000Z",
                    status: JobStatus.Pending,
                },
                {
                    id: 2,
                    data: "/some/very/long/path/that/doesnt/fit/on/one/line/and/should/be/truncated/some/very/long/path/that/doesnt/fit/on/one/line/and/should/be/truncated",
                    created_at: "2022-01-02T00:00:00.000Z",
                    status: JobStatus.Pending,
                },
            ],
            total_count: 2,
            limit: 10,
            offset: 0,
        },
        onPageChange: () => null,
        onDelete: undefined,
        onRetry: undefined,
    },
};

export const FailingJobs = {
    args: {
        data: {
            entries: [
                {
                    id: 1,
                    data: "/some/very/long/path/that/doesnt/fit/on/one/line",
                    created_at: "2022-01-01T00:00:00.000Z",
                    error: "some error",
                    status: JobStatus.Failed,
                },
                {
                    id: 2,
                    data: "/some/very/long/path/that/doesnt/fit/on/one/line",
                    created_at: "2022-01-02T00:00:00.000Z",
                    error: "very long error message that doesnt fit on one line",
                    status: JobStatus.Failed,
                },
            ],
            total_count: 2,
            limit: 10,
            offset: 0,
        },
        onPageChange: () => null,
        onDelete: () => null,
        onRetry: () => null,
    },
};

export const InProgressJobs = {
    args: {
        data: {
            entries: [
                {
                    id: 1,
                    data: "/some/very/long/path/that/doesnt/fit/on/one/line",
                    created_at: "2022-01-01T00:00:00.000Z",
                    status: JobStatus.InProgress,
                },
                {
                    id: 2,
                    data: "/some/very/long/path/that/doesnt/fit/on/one/line",
                    created_at: "2022-01-02T00:00:00.000Z",
                    status: JobStatus.InProgress,
                },
            ],
            total_count: 2,
            limit: 10,
            offset: 0,
        },
        onPageChange: () => null,
        onDelete: () => null,
        onRetry: () => null,
    },
};