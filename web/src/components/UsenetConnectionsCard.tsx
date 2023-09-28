import { createStyles, ThemeIcon, Progress, Text, Group, Paper, rem } from '@mantine/core';
import { IconNetwork } from '@tabler/icons-react';

const ICON_SIZE = rem(60);

const useStyles = createStyles((theme) => ({
    card: {
        position: 'relative',
        overflow: 'visible',
        padding: theme.spacing.xl,
        paddingTop: `calc(${theme.spacing.xl} * 1.5 + ${ICON_SIZE} / 3)`,
    },

    icon: {
        position: 'absolute',
        top: `calc(-${ICON_SIZE} / 3)`,
        left: `calc(50% - ${ICON_SIZE} / 2)`,
    },

    title: {
        fontFamily: `Greycliff CF, ${theme.fontFamily}`,
        lineHeight: 1,
    },
}));

export interface UsenetConnections {
    total: number;
    active: number;
    free: number;
}

interface UsenetConnectionsCardProps {
    data: UsenetConnections
    title: string
}

export default function UsenetConnectionsCard({ data, title }: UsenetConnectionsCardProps) {
    const { classes } = useStyles();
    const percentage = data.active / data.total * 100

    return (
        <Paper radius="md" withBorder className={classes.card} mt={`calc(${ICON_SIZE} / 3)`}>
            <ThemeIcon className={classes.icon} size={ICON_SIZE} radius={ICON_SIZE}>
                <IconNetwork size="2rem" stroke={1.5} />
            </ThemeIcon>

            <Text ta="center" fw={700} className={classes.title}>
                {title} usenet connections
            </Text>
            <Text c="dimmed" ta="center" fz="sm">
                {data.active} of {data.total} available connections
            </Text>

            <Group position="apart" mt="xs">
                <Text fz="sm" color="dimmed">
                    Usage
                </Text>
                <Text fz="sm" color="dimmed">
                    {percentage.toFixed(2)}%
                </Text>
            </Group>

            <Progress value={percentage} mt={5} />
        </Paper>
    );
}