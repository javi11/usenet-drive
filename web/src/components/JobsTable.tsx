import {
    Table,
    Group,
    Text,
    ActionIcon,
    ScrollArea,
    Pagination,
    Title,
} from '@mantine/core';
import { modals } from '@mantine/modals';
import { notifications } from '@mantine/notifications';
import { useCallback } from 'react';
import { IconRotateDot, IconTrash } from '@tabler/icons-react';
import { JobResponse, JobStatus } from '../data/job';
import Status from './Status';
import LargeText from './LargeText';

interface JobsTableProps {
    data: JobResponse;
    hasActions?: boolean;
    onPageChange: (page: number) => void;
}

export default function JobsTable({ data, hasActions, onPageChange }: JobsTableProps) {
    const totalPages = Math.ceil(data.totalCount / data.limit);

    const deleteModal = useCallback((id: number, status: JobStatus) => modals.openConfirmModal({
        title: <Title order={4}>Delete job</Title>,
        centered: true,
        children: (
            <Text size="sm">
                Are you sure you want to delete job {id}.
            </Text>
        ),
        labels: { confirm: 'Delete job', cancel: 'Cancel' },
        confirmProps: { color: 'red' },
        onCancel: () => { },
        onConfirm: async () => {
            try {
                const res = await fetch(`/api/v1/${status}jobs/${id}`, { method: "DELETE" });
                if (!res.ok) {
                    throw new Error(`Error deleting job ${id}.`);
                }
            } catch (error) {
                notifications.show({
                    title: 'An error occurred.',
                    message: `Unable to delete job ${id}.`,
                    color: 'red',
                })
            }
        },
    }), []);
    const retryModal = useCallback((id: number) => modals.openConfirmModal({
        title: <Title order={4}>Retry job</Title>,
        children: (
            <Text size="sm">
                Are you sure you want to retry the upload of job {id}.
            </Text>
        ),
        labels: { confirm: 'Retry', cancel: 'Cancel' },
        onCancel: () => { },
        onConfirm: async () => {
            try {
                const res = await fetch(`/api/v1/jobs/${id}/retry`, { method: "PUT" });
                if (!res.ok) {
                    throw new Error(`Error retrying job ${id}.`);
                }
            } catch (error) {
                notifications.show({
                    title: 'An error occurred.',
                    message: `Unable to retry job ${id}.`,
                    color: 'red',
                })
            }
        },
    }), []);

    const rows = data.entries.map((item) => (
        <tr key={item.id}>
            <td>
                <Text c="dimmed">
                    {item.id}
                </Text>
            </td>

            <td>
                <LargeText text={item.data} />
            </td>
            <td>
                <Text c="dimmed">
                    {item.createdAt}
                </Text>
            </td>
            <td>
                <Status status={item.status} error={item.error} />
            </td>
            {hasActions && <td>
                <Group spacing={0} position="right">
                    {item.error && <ActionIcon aria-label='Retry upload job' onClick={() => retryModal(item.id)}>
                        <IconRotateDot size="1rem" stroke={1.5} />
                    </ActionIcon>}
                    <ActionIcon aria-label='Delete upload job' color="red" onClick={() => deleteModal(item.id, item.status)}>
                        <IconTrash size="1rem" stroke={1.5} />
                    </ActionIcon>
                </Group>
            </td>}
        </tr>
    ));

    return (
        <ScrollArea>
            <Table sx={{ minWidth: 800 }} verticalSpacing="sm">
                <thead>
                    <tr>
                        <th>id</th>
                        <th>path</th>
                        <th>createdAt</th>
                        <th>status</th>
                        {hasActions && <th>actions</th>}
                        <th />
                    </tr>
                </thead>
                <tbody>{rows}</tbody>
                <tfoot><tr><td><Pagination total={totalPages} onChange={onPageChange} /></td></tr></tfoot>
            </Table>
        </ScrollArea>
    );
}