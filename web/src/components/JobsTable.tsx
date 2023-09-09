import {
    Table,
    Group,
    Text,
    ActionIcon,
    ScrollArea,
    Pagination,
} from '@mantine/core';
import { IconInfoCircle, IconRotateDot, IconTrash } from '@tabler/icons-react';
import { JobResponse, JobStatus } from '../data/job';
import Status from './Status';

interface JobsTableProps {
    data: JobResponse;
    onPageChange?: (page: number) => void;
    onDelete?: (id: number, status: JobStatus) => void;
    onRetry?: (id: number) => void;
    onOpenInfo?: (id: number) => void;
}

export default function JobsTable({ data, onPageChange, onDelete, onRetry, onOpenInfo }: JobsTableProps) {
    const totalPages = Math.ceil(data.total_count / data.limit);
    const hasActions = !!onDelete || !!onRetry || !!onOpenInfo;

    const rows = data.entries.map((item) => (
        <tr key={item.id}>
            <td>
                <Text c="dimmed">
                    {item.data}
                </Text>
            </td>
            <td>
                <Text c="dimmed">
                    {item.created_at}
                </Text>
            </td>
            <td>
                <Status status={item.status} error={item.error} />
            </td>
            {hasActions && <td>
                <Group spacing={0} position="right">
                    {onRetry && <ActionIcon aria-label='Retry upload job' onClick={() => onRetry(item.id)}>
                        <IconRotateDot size="1rem" stroke={1.5} />
                    </ActionIcon>}
                    {onDelete && <ActionIcon aria-label='Delete upload job' color="red" onClick={() => onDelete(item.id, item.status)}>
                        <IconTrash size="1rem" stroke={1.5} />
                    </ActionIcon>}
                    {onOpenInfo && <ActionIcon aria-label='See job progress' onClick={() => onOpenInfo(item.id)}>
                        <IconInfoCircle size="1rem" stroke={1.5} />
                    </ActionIcon>}
                </Group>
            </td>}
        </tr>
    ));

    return (
        <ScrollArea>
            <Table sx={{ minWidth: 200 }} verticalSpacing="sm">
                <thead>
                    <tr>
                        <th>path</th>
                        <th>createdAt</th>
                        <th>status</th>
                        {hasActions && <th>actions</th>}
                        <th />
                    </tr>
                </thead>
                <tbody>{rows}</tbody>
            </Table>
            {!!onPageChange ?? <Pagination total={totalPages} onChange={onPageChange} />}
        </ScrollArea>
    );
}