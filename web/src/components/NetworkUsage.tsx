import { Text, Paper, Group, rem, createStyles, ThemeIcon } from '@mantine/core';
import {
    IconDownload, IconUpload,
} from '@tabler/icons-react';
import prettyBytes from 'pretty-bytes';

const ICON_SIZE = rem(60);
const useStyles = createStyles((theme) => ({
    card: {
        position: 'relative',
        overflow: 'visible',
        padding: theme.spacing.xl,
    },
    title: {
        fontFamily: `Greycliff CF, ${theme.fontFamily}`,
        lineHeight: 1,
    },
}));


export interface NetworkUsage {
    upload_speed: number;
    download_speed: number
}

export interface NetworkUsageProps {
    data: NetworkUsage
}

export function NetworkUsageCard({ data }: NetworkUsageProps) {
    const { classes } = useStyles();

    return (
        <Group style={{ flex: 1 }} position="center" grow>
            <Paper radius="md" withBorder className={classes.card} >
                <Group position="center">
                    <ThemeIcon color='green' size={ICON_SIZE} radius={ICON_SIZE}>
                        <IconDownload size="2rem" stroke={1.5} />
                    </ThemeIcon>

                    <div>
                        <Text ta="center" fw={700} className={classes.title}>
                            Downloading
                        </Text>
                        <Text c="dimmed" ta="center" fz="sm">
                            <span>{prettyBytes(data.download_speed)}</span> / s
                        </Text>
                    </div>
                </Group>
            </Paper>
            <Paper radius="md" withBorder className={classes.card} >
                <Group position="center">
                    <ThemeIcon color='blue' size={ICON_SIZE} radius={ICON_SIZE}>
                        <IconUpload size="2rem" stroke={1.5} />
                    </ThemeIcon>

                    <div>
                        <Text ta="center" fw={700} className={classes.title}>
                            Uploading
                        </Text>
                        <Text c="dimmed" ta="center" fz="sm">
                            <span>{prettyBytes(data.upload_speed)}</span> / s
                        </Text>
                    </div>
                </Group>
            </Paper>
        </Group>
    );
}