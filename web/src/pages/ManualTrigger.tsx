import { createStyles, Text, Title, TextInput, Button, Image, rem, LoadingOverlay, Box } from '@mantine/core';
import pick from '../assets/scan.svg'
import { useCallback, useState } from 'react';
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

        [theme.fn.smallerThan('sm')]: {
            flexDirection: 'column-reverse',
            padding: theme.spacing.xl,
        },
    },

    image: {
        maxWidth: '40%',

        [theme.fn.smallerThan('sm')]: {
            maxWidth: '100%',
        },
    },

    body: {
        paddingRight: `calc(${theme.spacing.xl} * 4)`,

        [theme.fn.smallerThan('sm')]: {
            paddingRight: 0,
            marginTop: theme.spacing.xl,
        },
    },

    title: {
        color: theme.colorScheme === 'dark' ? theme.white : theme.black,
        fontFamily: `Greycliff CF, ${theme.fontFamily}`,
        lineHeight: 1,
        marginBottom: theme.spacing.md,
    },

    controls: {
        display: 'flex',
        marginTop: theme.spacing.xl,
    },

    inputWrapper: {
        width: '100%',
        flex: '1',
    },

    input: {
        borderTopRightRadius: 0,
        borderBottomRightRadius: 0,
        borderRight: 0,
    },

    control: {
        borderTopLeftRadius: 0,
        borderBottomLeftRadius: 0,
    },
}));

export default function ManualTrigger() {
    const { classes } = useStyles();
    const [loading, setLoading] = useState(false);
    const [value, setValue] = useState('');
    const [error, setInputError] = useState('');
    const triggerScan = useCallback(async (path: string) => {
        setLoading(true);
        if (!path.startsWith('/') || path.includes('..')) {
            setInputError('Invalid path');
            setLoading(false);
            return;
        }

        try {
            const res = await fetch(`/api/v1/manual-scan`, {
                headers: { "content-type": "application/json" },
                method: "POST",
                body: JSON.stringify({ file_path: path })
            });
            if (!res.ok) {
                const err: Error = await res.json();
                throw new Error(err.message);
            }
            notifications.show({
                title: 'Success',
                message: `Path ${path} added to the upload queue.`,
                color: 'green',
            })
            setLoading(false);
        } catch (error) {
            setLoading(false);
            const err = error as Error
            notifications.show({
                title: 'An error occurred.',
                message: `Unable to get scanning given path. ${err.message}`,
                color: 'red',
            })
        }
    }, []);

    return (
        <div className={classes.wrapper}>
            <div className={classes.body}>
                <Title className={classes.title}>Trigger a manual upload</Title>
                <Text fw={500} fz="lg" mb={5}>
                    Add a file or directory under the specified nzbs path on the config to the upload queue
                </Text>
                <Text fz="sm" c="dimmed">
                    For example: /nzbs/Media/TV/my-video.mkv
                </Text>
                <Box pos="relative" className={classes.controls}>
                    <LoadingOverlay visible={loading} overlayBlur={2} />
                    <TextInput
                        error={error}
                        value={value}
                        placeholder="File path or directory"
                        classNames={{ input: classes.input, root: classes.inputWrapper }}
                        onChange={(event) => {
                            setValue(event.currentTarget.value)
                            if (error) {
                                setInputError('')
                            }
                        }}
                    />
                    <Button className={classes.control} onClick={() => triggerScan(value)}>Scan</Button>
                </Box>
            </div>
            <Image src={pick} className={classes.image} />
        </div>
    );
}