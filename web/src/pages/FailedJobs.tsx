import { useCallback, useEffect, useState } from 'react';
import { JobResponse, JobStatus } from '../data/job';
import JobsTable from '../components/JobsTable';
import { Container, Group, Loader, Select, Title, createStyles, rem } from '@mantine/core';
import { notifications } from '@mantine/notifications';

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

export default function FailedJobs() {
    const [refreshInterval, setRefreshInterval] = useState(5000);
    const { classes } = useStyles();
    const [jobs, setJobs] = useState<JobResponse>({
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
                const res = await fetch(`/api/v1/jobs/failed?limit=${PAGE_SIZE}&offset=${offset}`);
                if (!res.ok) {
                    const err: Error = await res.json();
                    throw new Error(err.message);
                }
                const data: JobResponse = await res.json();
                setJobs({
                    ...data,
                    entries: data.entries.map((item) => ({ ...item, status: JobStatus.Failed }))
                });
                setIsLoading(false);
            } catch (error) {
                const err = error as Error
                notifications.show({
                    title: 'An error occurred.',
                    message: `Unable to get failed job list. ${err.message}`,
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


    return (
        <Container size="lg" className={classes.wrapper}>
            <Group>
                <Title align="center" className={classes.title}>
                    Failed jobs
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
                <JobsTable hasActions={true} data={jobs} onPageChange={handlePageChange} />
            )}
        </Container>
    );
}