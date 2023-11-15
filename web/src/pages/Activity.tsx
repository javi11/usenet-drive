import { useMemo } from 'react';
import {
    MantineReactTable,
    useMantineReactTable,
    type MRT_ColumnDef,
    MRT_ToggleFiltersButton,
} from 'mantine-react-table';
import {
    useQuery
} from '@tanstack/react-query';
import { ActionIcon, Badge, Flex, Tooltip, rem } from '@mantine/core';
import { IconDownload, IconRefresh, IconUpload } from '@tabler/icons-react';
import prettyBytes from 'pretty-bytes';
import { Activity, Kind } from '../data/activity';

function buildURL() {
    const fetchURL = new URL(
        `/api/v1/activity`,
        process.env.NODE_ENV === 'production'
            ? window.location.origin
            : 'http://localhost:8081',
    );
    return fetchURL;
}

function useActivity() {
    const fetchURL = buildURL()

    return useQuery<Activity[]>({
        queryKey: ['activity', fetchURL.href], //refetch whenever the URL changes (columnFilters, sorting, pagination)
        queryFn: () => fetch(fetchURL.href).then((res) => res.json()),
        refetchInterval: 3000,
    });
}

export default function Activity() {
    const columns = useMemo<MRT_ColumnDef<Activity>[]>(
        () => [
            {
                accessorKey: 'session_id',
                header: 'Session ID',
            },
            {
                accessorKey: 'path',
                header: 'File Name',
                Cell: ({ row }) => (
                    <Tooltip
                        label={row.original.path}
                        position="top"
                        withArrow
                    >
                        <div>{row.original.path.split(/[\\/]/).pop()}</div>
                    </Tooltip>
                ),
            },
            {
                accessorKey: 'speed',
                header: 'Speed',
                Cell: ({ row }) => {
                    const speed = (row.original.speed ? prettyBytes(row.original.speed) : '0') + '/s'
                    if (row.original.kind === Kind.download) {
                        return (
                            <Badge color="green" leftSection={<IconDownload style={{ width: rem(12), height: rem(12) }} />}>
                                {speed}
                            </Badge>
                        )
                    }

                    return (
                        <Badge color="blue" leftSection={<IconUpload style={{ width: rem(12), height: rem(12) }} />}>
                            {speed}
                        </Badge>
                    )
                }
            },
            {
                accessorKey: 'total_bytes',
                header: 'Total Bytes',
                Cell: ({ row }) => {
                    const totalBytes = (row.original.total_bytes ? prettyBytes(row.original.total_bytes) : '0')
                    if (row.original.kind === Kind.download) {
                        return (
                            <Badge color="orange" leftSection={<IconDownload style={{ width: rem(12), height: rem(12) }} />}>
                                {totalBytes}
                            </Badge>
                        )
                    }

                    return (
                        <Badge color="orange" leftSection={<IconUpload style={{ width: rem(12), height: rem(12) }} />}>
                            {totalBytes}
                        </Badge>
                    )
                }
            },
        ],
        [],
    );

    //call our custom react-query hook
    const { data, isError, isFetching, isLoading, refetch } = useActivity();
    const fetchedItems = data ?? [];
    const totalRowCount = 1;

    const table = useMantineReactTable({
        columns,
        enablePagination: false,
        data: fetchedItems,
        enableRowSelection: false,
        enableSelectAll: false,
        enableColumnFilterModes: false,
        enableRowActions: false,
        enablePinning: true,
        initialState: { showColumnFilters: false },
        manualFiltering: false,
        manualPagination: false,
        manualSorting: false,
        mantineToolbarAlertBannerProps: isError
            ? {
                color: 'red',
                children: 'Error loading data',
            }
            : undefined,
        rowCount: totalRowCount,
        state: {
            isLoading,
            showAlertBanner: isError,
            showProgressBars: isFetching,
        },
        renderTopToolbar: ({ table }) => {
            return (
                <Flex p="md" justify="space-between">
                    <Flex gap="xs">
                        <Tooltip label="Refresh Data">
                            <ActionIcon size="lg" onClick={() => refetch()}>
                                <IconRefresh />
                            </ActionIcon>
                        </Tooltip>
                        <MRT_ToggleFiltersButton table={table} />
                    </Flex>
                </Flex>
            );
        },
    });

    return <MantineReactTable table={table} />;
}