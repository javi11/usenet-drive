import {
    Table,
    Group,
    Text,
    ActionIcon,
    ScrollArea,
    Pagination,
    Tooltip,
    Badge,
    rem,
} from '@mantine/core';
import { IconTrash, IconX } from '@tabler/icons-react';
import { CorruptedNzbResponse } from '../data/corrupted-nzb';

interface CorruptedNzbTableProps {
    data: CorruptedNzbResponse;
    onPageChange: (page: number) => void;
    onDelete: (id: number, path: string) => void;
}

export default function CorruptedNzbTable({ data, onPageChange, onDelete }: CorruptedNzbTableProps) {
    const totalPages = Math.ceil(data.total_count / data.limit);

    const rows = data.entries.map((item) => (
        <tr key={item.id}>
            <td>
                <Text c="dimmed">
                    {item.path}
                </Text>
            </td>
            <td>
                <Text c="dimmed">
                    {item.created_at}
                </Text>
            </td>
            <td>
                <Tooltip label={item.error}>
                    <Badge pr={20} color="red" leftSection={<IconX size={rem(10)} />}>
                        Error
                    </Badge>
                </Tooltip>
            </td>
            <td>
                <Group spacing={0} position="right">
                    <ActionIcon aria-label='Delete corrupted nzb' color="red" onClick={() => onDelete(item.id, item.path)}>
                        <IconTrash size="1rem" stroke={1.5} />
                    </ActionIcon>
                </Group>
            </td>
        </tr>
    ));

    return (
        <ScrollArea>
            <Table sx={{ minWidth: 200 }} verticalSpacing="sm">
                <thead>
                    <tr>
                        <th>path</th>
                        <th>createdAt</th>
                        <th>error</th>
                        <th>actions</th>
                    </tr>
                </thead>
                <tbody>{rows}</tbody>
            </Table>
            <Pagination total={totalPages} onChange={onPageChange} />
            <Text mt="md" size="sm" color="dimmed">
                Showing {data.entries.length} of {data.total_count} corrupted nzbs
            </Text>
        </ScrollArea>
    );
}