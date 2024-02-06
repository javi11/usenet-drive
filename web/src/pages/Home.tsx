import {
    createStyles,
    Badge,
    Group,
    Title,
    SimpleGrid,
    Container,
    rem,
    LoadingOverlay,
} from '@mantine/core';
import DiskUsageCard, { DiskUsage } from '../components/DiskUsageCard';
import UsenetConnectionsCard, { ProviderInfo } from '../components/UsenetConnectionsCard';
import { useEffect, useState } from 'react';
import { notifications } from '@mantine/notifications';
import { NetworkUsageCard } from '../components/NetworkUsage';

const useStyles = createStyles((theme) => ({
    title: {
        fontSize: rem(34),
        fontWeight: 900,

        [theme.fn.smallerThan('sm')]: {
            fontSize: rem(24),
        },
    },

    description: {
        maxWidth: 600,
        margin: 'auto',

        '&::after': {
            content: '""',
            display: 'block',
            backgroundColor: theme.fn.primaryColor(),
            width: rem(45),
            height: rem(2),
            marginTop: theme.spacing.sm,
            marginLeft: 'auto',
            marginRight: 'auto',
        },
    },
}));

interface ServerInfo {
    root_folder_disk_usage: DiskUsage;
    providers_info: ProviderInfo[];
    global_activity: {
        download_speed: number;
        upload_speed: number;
    };
}

export default function Home() {
    const { classes } = useStyles();
    const [loading, setIsLoading] = useState(true);
    const [serverInfo, setServerInfo] = useState<ServerInfo>({
        root_folder_disk_usage: {
            total: 0,
            used: 0,
            free: 0,
            folder: '',
        },
        providers_info: [],
        global_activity: {
            download_speed: 0,
            upload_speed: 0,
        }
    });
    useEffect(() => {

        const fetchServerInfo = async () => {
            try {
                const fetchURL = new URL(
                    '/api/v1/server-info',
                    process.env.NODE_ENV === 'production'
                        ? window.location.origin
                        : 'http://localhost:8081',
                );
                const res = await fetch(fetchURL.href);
                if (!res.ok) {
                    const err: Error = await res.json();
                    throw new Error(err.message);
                }
                const data: ServerInfo = await res.json();
                setServerInfo(data);
            } catch (error) {
                const err = error as Error
                notifications.show({
                    title: 'An error occurred.',
                    message: `Unable to get server info. ${err.message}`,
                    color: 'red',
                })
            } finally {
                setIsLoading(false);
            }
        };

        fetchServerInfo();

        const intervalId = setInterval(() => fetchServerInfo(), 20000);

        return () => clearInterval(intervalId);
    }, []);

    return (
        <Container size="lg" py="xl">
            <LoadingOverlay visible={loading} overlayBlur={2} />
            <Group position="center">
                <Badge variant="filled" size="lg">
                    Server Info
                </Badge>
            </Group>

            <Title order={2} className={classes.title} ta="center" mt="sm">
                Welcome to usenet drive
            </Title>

            <SimpleGrid cols={2} spacing="xl" mt={50} breakpoints={[{ maxWidth: 'md', cols: 1 }]}>
                <NetworkUsageCard data={serverInfo.global_activity} />
                <DiskUsageCard data={serverInfo.root_folder_disk_usage} />
                {serverInfo.providers_info.map((provider, index) => (
                    <UsenetConnectionsCard key={index} data={provider} />
                ))}
            </SimpleGrid>
        </Container>
    );
}