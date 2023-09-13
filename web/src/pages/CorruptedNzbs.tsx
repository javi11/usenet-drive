import { useCallback, useEffect, useState } from 'react';
import { Container, Group, Loader, Select, Title, createStyles, rem, Text, Blockquote } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { modals } from '@mantine/modals';
import { CorruptedNzbResponse } from '../data/corrupted-nzb';
import CorruptedNzbTable from '../components/CorruptedNzbTable';

const useStyles = createStyles((theme) => ({
    wrapper: {
        display: 'flex',
        alignItems: 'center',
        padding: `calc(${theme.spacing.xl} * 2)`,
        borderRadius: theme.radius.md,
        backgroundColor: theme.colorScheme === 'dark' ? theme.colors.dark[8] : theme.white,
        border: `${rem(1)} solid ${theme.colorScheme === 'dark' ? theme.colors.dark[8] : theme.colors.gray[3]
            }`,
        flexDirection: 'column',
    },
    title: {
        color: theme.colorScheme === 'dark' ? theme.white : theme.black,
        fontFamily: `Greycliff CF, ${theme.fontFamily}`,
        lineHeight: 1,
        marginBottom: theme.spacing.md,
    },
}));

const PAGE_SIZE = 10; // number of items per page

export default function CorruptedNzbs() {
    const [refreshInterval, setRefreshInterval] = useState(5000);
    const { classes } = useStyles();
    const [cNzbs, setCNzbs] = useState<CorruptedNzbResponse>({
        total_count: 0,
        limit: PAGE_SIZE,
        offset: 0,
        entries: []
    });
    const [isLoading, setIsLoading] = useState(true);
    const [offset, setOffset] = useState(0);

    useEffect(() => {
        const fetchJobs = async (offset: number) => {
            try {
                const res = await fetch(`/api/v1/nzbs/corrupted?limit=${PAGE_SIZE}&offset=${offset}`);
                if (!res.ok) {
                    const err: Error = await res.json();
                    throw new Error(err.message);
                }
                const data: CorruptedNzbResponse = await res.json();
                setCNzbs(data);
                setIsLoading(false);
            } catch (error) {
                const err = error as Error
                notifications.show({
                    title: 'An error occurred.',
                    message: `Unable to get corrupted nzbs list. ${err.message}`,
                    color: 'red',
                })
            } finally {
                setIsLoading(false);
            }
        };

        fetchJobs(offset);

        const intervalId = setInterval(() => fetchJobs(offset), refreshInterval);

        return () => clearInterval(intervalId);
    }, [offset, refreshInterval]);
    const handlePageChange = useCallback((page: number) => {
        setOffset((page - 1) * PAGE_SIZE);
    }, []);
    const onDelete = useCallback((id: number, path: string) => modals.openConfirmModal({
        title: <Title order={4}>Delete corrupted nzb</Title>,
        centered: true,
        children: (
            <Text size="sm">
                Are you sure you want to delete the file "{path}".
            </Text>
        ),
        labels: { confirm: 'Delete file', cancel: 'Cancel' },
        confirmProps: { color: 'red' },
        onCancel: () => { },
        onConfirm: async () => {
            try {
                const res = await fetch(`/api/v1/nzbs/corrupted/${id}`, { method: "DELETE" });
                if (!res.ok) {
                    throw new Error(`Error deleting corrupted nzb ${id}.`);
                }
                const entries = cNzbs.entries = cNzbs.entries.filter((item) => item.id !== id);
                setCNzbs({
                    ...cNzbs,
                    total_count: cNzbs.total_count - 1,
                    entries
                });

            } catch (error) {
                notifications.show({
                    title: 'An error occurred.',
                    message: `Unable to delete corrupted nzb ${id}.`,
                    color: 'red',
                })
            }
        },
    }), []);

    return (
        <Container size="lg" className={classes.wrapper}>
            <Group>
                <Title align="center" className={classes.title}>
                    Corrupted Nzbs
                </Title>
                <Container size="xs" px="xs">
                    <Select
                        label="Refresh interval"
                        placeholder="Pick one"
                        defaultValue={refreshInterval.toString()}
                        value={refreshInterval.toString()}
                        onChange={(event) => setRefreshInterval(parseInt(event!))}
                        data={[
                            { value: '5000', label: '5s' },
                            { value: '10000', label: '10s' },
                            { value: '20000', label: '20s' },
                            { value: '30000', label: '30s' },
                        ]}

                        size='xs'
                    />
                </Container>
            </Group>
            {isLoading ? (
                <Loader />
            ) : (
                <CorruptedNzbTable data={cNzbs} onPageChange={handlePageChange} onDelete={onDelete} />
            )}
            <Blockquote  >
                These are nzbs that for some reason has an inparseable format, therefore, they won't appears on the file system.
            </Blockquote>
        </Container>
    );
}