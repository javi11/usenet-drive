import { useCallback, useEffect, useState } from 'react';
import { JobData, JobResponse, JobStatus } from '../data/job';
import JobsTable from '../components/JobsTable';
import { Button, Container, Group, Loader, Select, Title, createStyles, rem } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { modals } from '@mantine/modals';
import JobInfo from '../components/JobInfo';


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

export default function InProgressJobs() {
    const { classes } = useStyles();
    const [refreshInterval, setRefreshInterval] = useState(5000);
    const [jobs, setJobs] = useState<JobResponse>({
        total_count: 0,
        limit: PAGE_SIZE,
        offset: 0,
        entries: []
    });
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        const fetchJobs = async () => {
            try {
                const res = await fetch('/api/v1/jobs/in-progres');
                if (!res.ok) {
                    const err: Error = await res.json();
                    throw new Error(err.message);
                }
                const data: JobData[] = await res.json();
                setJobs({
                    total_count: data.length,
                    limit: PAGE_SIZE,
                    offset: 10,
                    entries: data.map((item) => ({
                        ...item,
                        status: JobStatus.InProgress
                    }))
                });
            } catch (error) {
                const err = error as Error
                notifications.show({
                    title: 'An error occurred.',
                    message: `Unable to get in progress job list. ${err.message}`,
                    color: 'red',
                })
            } finally {
                setIsLoading(false);
            }
        };

        fetchJobs();

        const intervalId = setInterval(() => fetchJobs(), refreshInterval);

        return () => clearInterval(intervalId);
    }, [refreshInterval]);

    const onOpenInfo = useCallback((id: number) => modals.open({
        title: `${id} info`,
        children: (
            <>
                <JobInfo id={id} />
                <Button fullWidth onClick={() => modals.closeAll()} mt="md">
                    Close
                </Button>
            </>
        ),
    }), []);

    return (
        <Container size="lg" className={classes.wrapper}>
            <Group>
                <Title align="center" className={classes.title}>
                    Jobs in progress
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
                <JobsTable data={jobs} onOpenInfo={onOpenInfo} />
            )}
        </Container>
    );
}