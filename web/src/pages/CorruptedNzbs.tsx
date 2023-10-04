import { useMemo, useState } from 'react';
import { Tooltip, ActionIcon, Flex, Button } from '@mantine/core';
import {
    MantineReactTable,
    useMantineReactTable,
    type MRT_ColumnDef,
    type MRT_ColumnFiltersState,
    type MRT_PaginationState,
    type MRT_SortingState,
    MRT_ColumnFilterFnsState,
    MRT_ToggleFiltersButton,
} from 'mantine-react-table';
import {
    useMutation,
    useQuery, useQueryClient,
} from '@tanstack/react-query';
import { IconFileCheck, IconRefresh } from '@tabler/icons-react';
import { CorruptedNzb, CorruptedNzbResponse } from '../data/corrupted-nzb';
import { notifications } from '@mantine/notifications';
import { modals } from '@mantine/modals';
import FileContent from '../components/FileContent';

interface Params {
    columnFilterFns: MRT_ColumnFilterFnsState;
    columnFilters: MRT_ColumnFiltersState;
    sorting: MRT_SortingState;
    pagination: MRT_PaginationState;
}

function buildURL({
    columnFilterFns,
    columnFilters,
    sorting,
    pagination,
}: Params) {
    //build the URL (https://www.mantine-react-table.com/api/data?start=0&size=10&filters=[]&sorting=[])
    const fetchURL = new URL(
        `/api/v1/nzbs/corrupted`,
        process.env.NODE_ENV === 'production'
            ? window.location.origin
            : 'http://localhost:8081',
    );
    fetchURL.searchParams.set(
        'offset',
        `${pagination.pageIndex * pagination.pageSize}`,
    );
    fetchURL.searchParams.set('limit', `${pagination.pageSize}`);
    fetchURL.searchParams.set('filters', JSON.stringify(columnFilters ?? []));
    fetchURL.searchParams.set(
        'filterModes',
        JSON.stringify(columnFilterFns ?? {}),
    );
    fetchURL.searchParams.set('sorting', JSON.stringify(sorting ?? []));

    return fetchURL;
}

function useCorruptedNzbs(params: Params) {
    const fetchURL = buildURL(params)

    return useQuery<CorruptedNzbResponse>({
        queryKey: ['corruptednzb', fetchURL.href], //refetch whenever the URL changes (columnFilters, sorting, pagination)
        queryFn: () => fetch(fetchURL.href).then((res) => res.json()),
        keepPreviousData: true, //useful for paginated queries by keeping data from previous pages on screen while fetching the next page
        staleTime: 30_000, //don't refetch previously viewed pages until cache is more than 30 seconds old
    });
};

function useDiscardNzb(params: Params) {
    const fetchURL = buildURL(params)

    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (nzbPath: string) => {
            const fetchURL = new URL(
                '/api/v1/nzbs/corrupted/discard',
                process.env.NODE_ENV === 'production'
                    ? window.location.origin
                    : 'http://localhost:8081',
            );
            const res = await fetch(fetchURL.href, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ path: nzbPath }),
            });

            if (res.ok) {
                notifications.show({ title: 'NZB discarded', color: 'green', message: nzbPath })

                return
            } else {
                const { error } = await res.json();
                notifications.show({ title: 'Error discarding NZB', color: 'red', message: error })

                throw new Error(error)
            }
        },
        //client side optimistic update
        onMutate: (path: string) => {
            queryClient.setQueryData(
                ['corruptednzb', fetchURL],
                (prevNzbs: any) => {
                    const res: CorruptedNzbResponse = prevNzbs
                    return {
                        ...res,
                        entries: res.entries.filter((prevNzb: CorruptedNzb) => prevNzb.path !== path),
                        total_count: res.total_count - 1
                    }
                },
            );
        },
        onSettled: () => queryClient.invalidateQueries({ queryKey: ['corruptednzb', fetchURL] }), //refetch nzbs after mutation, disabled for demo
    });
}

function useDeleteNzb(params: Params) {
    const fetchURL = buildURL(params)

    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (nzbPath: string) => {
            const fetchURL = new URL(
                '/api/v1/nzbs/corrupted',
                process.env.NODE_ENV === 'production'
                    ? window.location.origin
                    : 'http://localhost:8081',
            );
            const res = await fetch(fetchURL.href, {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ path: nzbPath }),
            });

            if (res.ok) {
                notifications.show({ title: 'NZB deleted', color: 'green', message: nzbPath })

                return
            } else {
                const { error } = await res.json();
                notifications.show({ title: 'Error deleting NZB', color: 'red', message: error })

                throw new Error(error)
            }
        },
        //client side optimistic update
        onMutate: (path: string) => {
            queryClient.setQueryData(
                ['corruptednzb', fetchURL],
                (prevNzb: any) =>
                    prevNzb?.filter((prevNzb: CorruptedNzb) => prevNzb.path !== path),
            );
        },
        onSettled: () => queryClient.invalidateQueries({ queryKey: ['corruptednzb', fetchURL] }), //refetch nzbs after mutation, disabled for demo
    });
}

async function getNzbContent(id: string): Promise<ReadableStream<Uint8Array> | null> {
    const res = await fetch(`/api/v1/nzbs/corrupted/${id}`, {
        method: 'GET'
    });

    if (res.ok) {
        return res.body;
    } else {
        const { error } = await res.json();
        notifications.show({ title: 'Error getting NZB content', color: 'red', message: error })

        return null;
    }
}

export default function CorruptedNzbs() {
    const openFileContentModal = (contentStream: ReadableStream<Uint8Array>) =>
        modals.openConfirmModal({
            title: 'File Content',
            children: (
                <FileContent contentStream={contentStream} />
            ),
        });

    const columns = useMemo<MRT_ColumnDef<CorruptedNzb>[]>(
        () => [
            {
                accessorKey: 'id',
                header: 'ID',
                enableHiding: true,
            },
            {
                accessorKey: 'path',
                header: 'Path',
            },
            {
                accessorKey: 'created_at',
                header: 'Created At',
            },
            {
                accessorKey: 'error',
                header: 'Error',
            },
        ],
        [],
    );

    //Manage MRT state that we want to pass to our API
    const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>(
        [],
    );
    const [columnFilterFns, setColumnFilterFns] = //filter modes
        useState<MRT_ColumnFilterFnsState>(
            Object.fromEntries(
                columns.map(({ accessorKey }) => [accessorKey, 'contains']),
            ),
        ); //default to "contains" for all columns
    const [sorting, setSorting] = useState<MRT_SortingState>([]);
    const [pagination, setPagination] = useState<MRT_PaginationState>({
        pageIndex: 0,
        pageSize: 10,
    });

    //call our custom react-query hook
    const { data, isError, isFetching, isLoading, refetch } = useCorruptedNzbs({
        columnFilterFns,
        columnFilters,
        pagination,
        sorting,
    });
    const { mutateAsync: deleteNzb, isLoading: isDeleting } = useDeleteNzb({
        columnFilterFns,
        columnFilters,
        pagination,
        sorting,
    });
    const { mutateAsync: discardNzb, isLoading: isDiscarding } = useDiscardNzb({
        columnFilterFns,
        columnFilters,
        pagination,
        sorting,
    });
    const fetchedItems = data?.entries ?? [];
    const totalRowCount = data?.total_count ?? 0;

    const table = useMantineReactTable({
        columns,
        data: fetchedItems,
        enableRowSelection: true,
        enableSelectAll: true,
        enableColumnFilterModes: true,
        enableRowActions: true,
        enablePinning: true,
        columnFilterModeOptions: ['contains', 'startsWith', 'endsWith'],
        initialState: { showColumnFilters: true },
        manualFiltering: true,
        manualPagination: true,
        manualSorting: true,
        mantineToolbarAlertBannerProps: isError
            ? {
                color: 'red',
                children: 'Error loading data',
            }
            : undefined,
        onColumnFilterFnsChange: setColumnFilterFns,
        onColumnFiltersChange: setColumnFilters,
        onPaginationChange: setPagination,
        onSortingChange: setSorting,
        rowCount: totalRowCount,
        state: {
            columnFilterFns,
            columnFilters,
            isLoading,
            pagination,
            showAlertBanner: isError,
            showProgressBars: isFetching,
            sorting,
            isSaving: isDiscarding || isDeleting,
        },
        renderRowActions: ({ row }) => (
            <Flex gap="md">
                <Tooltip label="View file content">
                    <ActionIcon onClick={async () => {
                        const contentStream = await getNzbContent(row.getValue('id'));
                        if (contentStream) {
                            openFileContentModal(contentStream);
                        }
                    }} >
                        <IconFileCheck />
                    </ActionIcon>
                </Tooltip>
            </Flex>
        ),
        renderTopToolbar: ({ table }) => {
            const handleDelete = async () => {
                await Promise.allSettled(table.getSelectedRowModel().flatRows.map((row) => deleteNzb(row.getValue('path'))));
            };
            const handleDiscard = async () => {
                await Promise.allSettled(table.getSelectedRowModel().flatRows.map((row) => discardNzb(row.getValue('path'))));
            };

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
                    <Flex sx={{ gap: '8px' }}>
                        <Button
                            color="red"
                            disabled={!table.getIsSomeRowsSelected() && !table.getIsAllRowsSelected()}
                            onClick={handleDelete}
                            variant="filled"
                        >
                            Delete
                        </Button>
                        <Button
                            color="red"
                            disabled={!table.getIsSomeRowsSelected() && !table.getIsAllRowsSelected()}
                            onClick={handleDiscard}
                            variant="filled"
                        >
                            Discard
                        </Button>
                    </Flex>
                </Flex>
            );
        },
    });

    return <MantineReactTable table={table} />;
}