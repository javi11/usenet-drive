'use client'
import { useCallback, useEffect, useState } from 'react';
import {
    Box,
    Spinner,
} from '@chakra-ui/react';
import { JobResponse, JobStatus } from '../data/job';
import JobList from '../components/JobList';

const PAGE_SIZE = 10; // number of items per page

export default function FailedJobs() {
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
                const data: JobResponse = {
                    totalCount: 25,
                    limit: 10,
                    offset: 0,
                    entries: [
                        {
                            id: 1,
                            data: '/nzbs/path/to/nzb1/4',
                            createdAt: '2021-10-09',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 2,
                            data: '/nzbs/path/to/nzb2/4',
                            createdAt: '2021-10-10',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 3,
                            data: '/nzbs/path/to/nzb3/4',
                            createdAt: '2021-10-11',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 4,
                            data: '/nzbs/path/to/nzb4/4',
                            createdAt: '2021-10-12',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 5,
                            data: '/nzbs/path/to/nzb5/4',
                            createdAt: '2021-10-13',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 6,
                            data: '/nzbs/path/to/nzb6/4',
                            createdAt: '2021-10-14',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 7,
                            data: '/nzbs/path/to/nzb7/4',
                            createdAt: '2021-10-15',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 8,
                            data: '/nzbs/path/to/nzb8/4',
                            createdAt: '2021-10-16',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 9,
                            data: '/nzbs/path/to/nzb9/4',
                            createdAt: '2021-10-17',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 10,
                            data: '/nzbs/path/to/nzb10/4',
                            createdAt: '2021-10-18',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 11,
                            data: '/nzbs/path/to/nzb11/4',
                            createdAt: '2021-10-19',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 12,
                            data: '/nzbs/path/to/nzb12/4',
                            createdAt: '2021-10-20',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 13,
                            data: '/nzbs/path/to/nzb13/4',
                            createdAt: '2021-10-21',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 14,
                            data: '/nzbs/path/to/nzb14/4',
                            createdAt: '2021-10-22',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 15,
                            data: '/nzbs/path/to/nzb15/4',
                            createdAt: '2021-10-23',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 16,
                            data: '/nzbs/path/to/nzb16/4',
                            createdAt: '2021-10-24',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 17,
                            data: '/nzbs/path/to/nzb17/4',
                            createdAt: '2021-10-25',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 18,
                            data: '/nzbs/path/to/nzb18/4',
                            createdAt: '2021-10-26',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 19,
                            data: '/nzbs/path/to/nzb19/4',
                            createdAt: '2021-10-27',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 20,
                            data: '/nzbs/path/to/nzb20/4',
                            createdAt: '2021-10-28',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 21,
                            data: '/nzbs/path/to/nzb21/4',
                            createdAt: '2021-10-29',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 22,
                            data: '/nzbs/path/to/nzb22/4',
                            createdAt: '2021-10-30',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 23,
                            data: '/nzbs/path/to/nzb23/4',
                            createdAt: '2021-10-31',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 24,
                            data: '/nzbs/path/to/nzb24/4',
                            createdAt: '2021-11-01',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                        {
                            id: 25,
                            data: '/nzbs/path/to/nzb25/4',
                            createdAt: '2021-11-02',
                            status: JobStatus.Failed,
                            error: 'Failed to download NZB',
                        },
                    ]
                }
                const currentJobs = data.entries.slice(offset, offset + PAGE_SIZE);
                setJobs({
                    ...data,
                    entries: currentJobs,
                    offset: offset,
                });
                setIsLoading(false);
            } catch (error) {
                console.error(error);
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
        <Box maxW="7xl" mx={'auto'} pt={5} px={{ base: 2, sm: 12, md: 17 }}>
            {isLoading ? (
                <Spinner />
            ) : (
                <JobList title="Failed jobs" captions={["id", "path", "createdAt", "status", "actions"]} data={jobs} onPageChange={handlePageChange} />
            )}
        </Box>
    );
}