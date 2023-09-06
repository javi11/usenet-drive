import { useCallback, useEffect, useState } from 'react';
import { JobResponse, JobStatus } from '../data/job';
import JobsTable from '../components/JobsTable';
import { Container, Loader, Title, createStyles, rem } from '@mantine/core';
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

export default function PendingJobs() {
    const { classes } = useStyles();
    const [jobs, setJobs] = useState<JobResponse>({
        totalCount: 0,
        limit: PAGE_SIZE,
        offset: 0,
        entries: []
    });
    const [isLoading, setIsLoading] = useState(true);
    const [offset, setOffset] = useState(0);

    useEffect(() => {
        const fetchJobs = async (offset: number) => {
            try {
                const res = await fetch(`/api/v1/jobs/pending?limit=${PAGE_SIZE}&offset=${offset}`);
                if (!res.ok) {
                    const err: Error = await res.json();
                    throw new Error(err.message);
                }
                const data: JobResponse = await res.json();
                setJobs({
                    ...data,
                    entries: data.entries.map((item) => ({ ...item, status: JobStatus.Pending }))
                });
                setIsLoading(false);
            } catch (error) {
                const err = error as Error
                notifications.show({
                    title: 'An error occurred.',
                    message: `Unable to get pending job list. ${err.message}`,
                    color: 'red',
                })
            } finally {
                setIsLoading(false);
            }
        };

        fetchJobs(offset);

        const intervalId = setInterval(() => fetchJobs(offset), 5000);

        return () => clearInterval(intervalId);
    }, [offset]);
    const handlePageChange = useCallback((page: number) => {
        setOffset((page - 1) * PAGE_SIZE);
    }, []);

    return (
        <Container size="lg" className={classes.wrapper}>
            <Title align="center" className={classes.title}>
                Pending jobs
            </Title>
            {isLoading ? (
                <Loader />
            ) : (
                <JobsTable hasActions={true} data={jobs} onPageChange={handlePageChange} />
            )}
        </Container>
    );
}