'use client'
import { useCallback, useEffect, useState } from 'react';
import {
    Box,
    Spinner,
} from '@chakra-ui/react';
import { JobResponse, JobStatus } from '../data/job';
import JobList from '../components/JobList';

const PAGE_SIZE = 10; // number of items per page

export default function InProgressJobs() {
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
                    totalCount: 1,
                    limit: 10,
                    offset: 0,
                    entries: [
                        {
                            id: 2,
                            data: '/nzbs/path/to/nzb2/4',
                            createdAt: '2021-10-10',
                            status: JobStatus.InProgress,
                        }
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
                <JobList title="In progress jobs" captions={["id", "path", "createdAt", "status"]} data={jobs} onPageChange={handlePageChange} />
            )}
        </Box>
    );
}