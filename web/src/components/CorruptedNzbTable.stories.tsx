import CorruptedNzbTable from './CorruptedNzbTable';

export default {
    component: CorruptedNzbTable,
    title: 'CorruptedNzbTable',
    tags: ['autodocs'],
};

export const Default = {
    args: {
        data: {
            entries: [
                {
                    id: 1,
                    path: "some/path",
                    created_at: "2022-01-01T00:00:00.000Z",
                    error: "some error"
                },
                {
                    id: 2,
                    path: "some/path",
                    created_at: "2022-01-02T00:00:00.000Z",
                    error: "some other error"
                },
            ],
            total_count: 2,
            limit: 10,
            offset: 0,
        },
        onPageChange: () => null,
        onDelete: () => null,
    },
};

export const LargePath = {
    args: {
        data: {
            entries: [
                {
                    id: 1,
                    path: "/some/very/long/path/that/doesnt/fit/on/one/line/and/should/be/truncated",
                    created_at: "2022-01-01T00:00:00.000Z",
                    error: "some error"
                },
                {
                    id: 2,
                    path: "/some/very/long/path/that/doesnt/fit/on/one/line/and/should/be/truncated/some/very/long/path/that/doesnt/fit/on/one/line/and/should/be/truncated",
                    created_at: "2022-01-02T00:00:00.000Z",
                    error: "some other error"
                },
            ],
            total_count: 2,
            limit: 10,
            offset: 0,
        },
        onPageChange: () => null,
        onDelete: () => null,
    },
};