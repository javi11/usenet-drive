import { createStyles, Text, Group, rem, Card, RingProgress, useMantineTheme, Mark, Popover, ActionIcon, Flex } from '@mantine/core';
import { Kind } from '../data/activity';
import { IconInfoCircle } from '@tabler/icons-react';

const useStyles = createStyles((theme) => ({
    card: {
        backgroundColor: theme.colorScheme === 'dark' ? theme.colors.dark[7] : theme.white,
    },

    label: {
        fontWeight: 700,
        lineHeight: 1
    },

    lead: {
        fontWeight: 700,
        fontSize: rem(22),
        lineHeight: 1,
    },

    inner: {
        display: 'flex',
    },

    ring: {
        flex: 1,
        display: "flex",
        justifyContent: "flex-end"
    },
}));

export interface ProviderInfo {
    host: string
    username: string
    usedConnections: number
    maxConnections: number
    type: Kind
}

interface UsenetConnectionsCardProps {
    data: ProviderInfo
}

export default function UsenetConnectionsCard({ data }: UsenetConnectionsCardProps) {
    const { classes } = useStyles();
    const theme = useMantineTheme();
    const percentage = data.usedConnections / data.maxConnections * 100

    return (
        <Card withBorder p="xl" radius="md" className={classes.card}>
            <div className={classes.inner}>
                <div>
                    <Text fz="xl" className={classes.label}>
                        <Mark color="rgba(255, 177, 33, 1)" style={{ textTransform: 'capitalize' }}>{data.type}</Mark> usenet connections
                    </Text>
                    <div>
                        <Text className={classes.lead} mt={30}>
                            Host
                        </Text>
                        <Text fz="xs" c="dimmed">
                            {data.host}
                        </Text>
                    </div>
                    <Group mt="lg">
                        <div key="username">
                            <Text className={classes.label}>Username</Text>
                            <Text size="xs" c="dimmed">
                                {data.username}
                            </Text>
                        </div>
                        <div key="username">
                            <Flex align="center">
                                <Text className={classes.label}>Usage</Text>
                                <Popover width={200} position="bottom" withArrow shadow="md">
                                    <Popover.Target>
                                        <ActionIcon variant="subtle"> <IconInfoCircle size="1.8rem" stroke={1.5} /></ActionIcon>
                                    </Popover.Target>
                                    <Popover.Dropdown>
                                        <Text size="xs">Connections can be still in use even if there is no active download, they will be auto closed when max ttl is reached</Text>
                                    </Popover.Dropdown>
                                </Popover>
                            </Flex>
                            <Text size="xs" c="dimmed">
                                {data.usedConnections} of {data.maxConnections} available connections
                            </Text>
                        </div>
                    </Group>
                </div>

                <div className={classes.ring}>
                    <RingProgress
                        roundCaps
                        thickness={6}
                        size={150}
                        sections={[{ value: percentage, color: theme.primaryColor }]}
                        label={
                            <div>
                                <Text ta="center" fz="lg" className={classes.label}>
                                    {percentage.toFixed(0)}%
                                </Text>
                                <Text ta="center" fz="xs" c="dimmed">
                                    Usage
                                </Text>
                            </div>
                        }
                    />
                </div>
            </div>
        </Card>
    );
}