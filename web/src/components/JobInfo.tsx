import { Container, LoadingOverlay } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useEffect, useState } from 'react';
import { Prism } from '@mantine/prism';

interface JobInfoProps {
    id: number;
}

export default function JobInfo({ id }: JobInfoProps) {
    const refreshInterval = 5000;
    const [text, setText] = useState('');
    const [loading, setIsLoading] = useState(true);
    useEffect(() => {
        setIsLoading(true);
        const fetchLogs = async () => {
            try {
                const res = await fetch(`/api/v1/jobs/in-progres/${id}/logs`);
                if (!res.ok) {
                    const err: Error = await res.json();
                    throw new Error(err.message);
                }
                const data = await res.text();
                setText(`${text}\n${data}`);
            } catch (error) {
                const err = error as Error
                notifications.show({
                    title: 'An error occurred.',
                    message: `Unable to get in progress job list.${err.message}`,
                    color: 'red',
                })
            } finally {
                setIsLoading(false);
            }
        };

        fetchLogs();

        const intervalId = setInterval(() => fetchLogs(), refreshInterval);

        return () => clearInterval(intervalId);
    }, [refreshInterval]);

    return (
        <Container size="lg" py="xl">
            <LoadingOverlay visible={loading} overlayBlur={2} />
            <Prism scrollAreaComponent="div" colorScheme="dark" language="bash">{text}</Prism>
        </Container>
    )
}